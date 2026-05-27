package indexer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jaychillin2607/federated-search/internal/model"
	osq "github.com/jaychillin2607/federated-search/internal/opensearch"
	"github.com/jaychillin2607/federated-search/internal/postgres"
)

type Worker struct {
	pool       *pgxpool.Pool
	osClient   *osq.Client
	checkpoint *postgres.CheckpointStore
	sources    []Source
	interval   time.Duration
	batchSize  int
}

type Config struct {
	Pool       *pgxpool.Pool
	OpenSearch *osq.Client
	Checkpoint *postgres.CheckpointStore
	Sources    []Source
	Interval   time.Duration
	BatchSize  int
}

func NewWorker(cfg Config) *Worker {
	return &Worker{
		pool:       cfg.Pool,
		osClient:   cfg.OpenSearch,
		checkpoint: cfg.Checkpoint,
		sources:    cfg.Sources,
		interval:   cfg.Interval,
		batchSize:  cfg.BatchSize,
	}
}

// Run executes the tick loop until ctx is canceled. The current tick gets up to
// 10 seconds to finish (checkpoint persisted) after cancellation.
func (w *Worker) Run(ctx context.Context) {
	// Run an immediate tick on startup so docker-compose tests don't have to wait
	// the full interval.
	w.tickOnce(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("indexer worker stopping")
			return
		case <-ticker.C:
			tickCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			// If the parent has already been canceled, propagate it so the
			// in-progress tick aborts.
			if ctx.Err() != nil {
				cancel()
				return
			}
			w.tickOnce(tickCtx)
			cancel()
		}
	}
}

func (w *Worker) tickOnce(ctx context.Context) {
	for _, src := range w.sources {
		w.syncSource(ctx, src)
	}
}

func (w *Worker) syncSource(ctx context.Context, src Source) {
	start := time.Now()
	checkpoint, err := w.checkpoint.Get(ctx, src.Name)
	if err != nil {
		slog.Error("checkpoint get failed", "source", src.Name, "err", err)
		return
	}

	rows, err := w.pool.Query(ctx, src.SQL, checkpoint, w.batchSize)
	if err != nil {
		slog.Error("source query failed", "source", src.Name, "err", err)
		errStr := err.Error()
		_ = w.checkpoint.Upsert(ctx, src.Name, checkpoint, 0, &errStr)
		return
	}
	defer rows.Close()

	fields := rows.FieldDescriptions()
	toIndex := make([]model.Document, 0, w.batchSize)
	toDelete := make([]string, 0)
	maxUpdated := checkpoint
	rowCount := 0

	for rows.Next() {
		values, scanErr := rows.Values()
		if scanErr != nil {
			slog.Error("scan failed", "source", src.Name, "err", scanErr)
			continue
		}
		row := rowToMap(fields, values)
		rowCount++

		if ts, ok := row["updated_at"].(time.Time); ok && ts.After(maxUpdated) {
			maxUpdated = ts
		}

		if del, ok := row["deleted_at"].(time.Time); ok && !del.IsZero() {
			id := documentID(src, row)
			toDelete = append(toDelete, id)
			continue
		}

		doc, tErr := src.Transform(row)
		if tErr != nil {
			slog.Error("transform failed", "source", src.Name, "err", tErr)
			continue
		}
		toIndex = append(toIndex, doc)
	}
	if err := rows.Err(); err != nil {
		slog.Error("rows iteration failed", "source", src.Name, "err", err)
		errStr := err.Error()
		_ = w.checkpoint.Upsert(ctx, src.Name, checkpoint, 0, &errStr)
		return
	}
	rows.Close()

	indexed := 0
	deleted := 0
	var firstErr error

	if len(toIndex) > 0 {
		res, bErr := w.osClient.BulkUpsert(ctx, toIndex)
		if bErr != nil {
			firstErr = fmt.Errorf("bulk upsert: %w", bErr)
		} else {
			indexed = res.Indexed
			if res.Failed > 0 && firstErr == nil {
				firstErr = fmt.Errorf("%d docs failed to index", res.Failed)
			}
		}
	}
	if len(toDelete) > 0 {
		d, _, dErr := w.osClient.BulkDelete(ctx, toDelete)
		if dErr != nil && firstErr == nil {
			firstErr = fmt.Errorf("bulk delete: %w", dErr)
		}
		deleted = d
	}

	var lastErr *string
	if firstErr != nil {
		s := firstErr.Error()
		lastErr = &s
	}
	if err := w.checkpoint.Upsert(ctx, src.Name, maxUpdated, indexed+deleted, lastErr); err != nil {
		slog.Error("checkpoint upsert failed", "source", src.Name, "err", err)
	}

	slog.Info("sync complete",
		"source", src.Name,
		"rows_seen", rowCount,
		"indexed", indexed,
		"deleted", deleted,
		"checkpoint_before", checkpoint,
		"checkpoint_after", maxUpdated,
		"duration_ms", time.Since(start).Milliseconds(),
		"err", firstErr,
	)
}

func documentID(src Source, row map[string]any) string {
	id := toInt64(row["id"])
	return fmt.Sprintf("%s_%d", src.Category, id)
}

func rowToMap(fields []pgconn.FieldDescription, values []any) map[string]any {
	out := make(map[string]any, len(fields))
	for i, fd := range fields {
		out[string(fd.Name)] = values[i]
	}
	return out
}

// silence unused-package linter when pgx error sentinel isn't needed.
var _ = pgx.ErrNoRows

// ErrNoSources is returned when worker is started without any sources.
var ErrNoSources = errors.New("no sources registered")
