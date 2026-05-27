package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jaychillin2607/federated-search/internal/config"
	"github.com/jaychillin2607/federated-search/internal/indexer"
	osq "github.com/jaychillin2607/federated-search/internal/opensearch"
	"github.com/jaychillin2607/federated-search/internal/postgres"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	osClient, err := osq.New(osq.Options{
		URL:                cfg.OpenSearchURL,
		Username:           cfg.OpenSearchUsername,
		Password:           cfg.OpenSearchPassword,
		Index:              cfg.OpenSearchIndex,
		InsecureSkipVerify: cfg.OpenSearchInsecureSkipVerify,
	})
	if err != nil {
		slog.Error("opensearch client init failed", "err", err)
		os.Exit(1)
	}

	bootstrapCtx, cancelBootstrap := context.WithTimeout(ctx, 30*time.Second)
	if err := osClient.EnsureIndex(bootstrapCtx); err != nil {
		cancelBootstrap()
		slog.Error("ensure index failed", "err", err)
		os.Exit(1)
	}
	cancelBootstrap()

	pool, err := postgres.NewPool(ctx, cfg.PostgresDSN)
	if err != nil {
		slog.Error("postgres pool init failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	worker := indexer.NewWorker(indexer.Config{
		Pool:       pool,
		OpenSearch: osClient,
		Checkpoint: postgres.NewCheckpointStore(pool),
		Sources:    indexer.RegisteredSources(),
		Interval:   time.Duration(cfg.IndexerIntervalSeconds) * time.Second,
		BatchSize:  cfg.IndexerBatchSize,
	})

	slog.Info("indexer starting",
		"interval_seconds", cfg.IndexerIntervalSeconds,
		"batch_size", cfg.IndexerBatchSize,
		"sources", len(indexer.RegisteredSources()),
	)
	worker.Run(ctx)
	slog.Info("indexer stopped cleanly")
}
