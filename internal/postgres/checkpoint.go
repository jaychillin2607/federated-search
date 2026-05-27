package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Epoch is the default last_synced_at value.
var Epoch = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)

type SyncState struct {
	Name             string     `json:"name"`
	LastSyncedAt     time.Time  `json:"last_synced_at"`
	LastSuccessCount int        `json:"last_success_count"`
	LastError        *string    `json:"last_error"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type CheckpointStore struct {
	pool *pgxpool.Pool
}

func NewCheckpointStore(pool *pgxpool.Pool) *CheckpointStore {
	return &CheckpointStore{pool: pool}
}

// Get returns the checkpoint for a source, or (Epoch, nil) if there is none.
func (s *CheckpointStore) Get(ctx context.Context, source string) (time.Time, error) {
	var ts time.Time
	err := s.pool.QueryRow(ctx,
		`SELECT last_synced_at FROM search_sync_state WHERE source_name = $1`, source,
	).Scan(&ts)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Epoch, nil
		}
		return time.Time{}, err
	}
	return ts, nil
}

// Upsert writes a new checkpoint for the source. lastError == nil clears the error field.
func (s *CheckpointStore) Upsert(ctx context.Context, source string, lastSyncedAt time.Time, count int, lastError *string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO search_sync_state (source_name, last_synced_at, last_success_count, last_error, updated_at)
         VALUES ($1, $2, $3, $4, NOW())
         ON CONFLICT (source_name) DO UPDATE
           SET last_synced_at     = EXCLUDED.last_synced_at,
               last_success_count = EXCLUDED.last_success_count,
               last_error         = EXCLUDED.last_error,
               updated_at         = NOW()`,
		source, lastSyncedAt, count, lastError,
	)
	return err
}

// ResetToEpoch sets the checkpoint to epoch, creating the row if needed.
// Used by the reindex endpoint.
func (s *CheckpointStore) ResetToEpoch(ctx context.Context, source string) error {
	return s.Upsert(ctx, source, Epoch, 0, nil)
}

// ResetAllToEpoch resets every existing row to epoch.
func (s *CheckpointStore) ResetAllToEpoch(ctx context.Context) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE search_sync_state SET last_synced_at = $1, last_error = NULL, updated_at = NOW()`,
		Epoch,
	)
	return err
}

// List returns the status of every registered source. Sources that have never
// run will be missing — callers can fill in epoch defaults.
func (s *CheckpointStore) List(ctx context.Context) ([]SyncState, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT source_name, last_synced_at, last_success_count, last_error, updated_at
           FROM search_sync_state ORDER BY source_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SyncState
	for rows.Next() {
		var st SyncState
		if err := rows.Scan(&st.Name, &st.LastSyncedAt, &st.LastSuccessCount, &st.LastError, &st.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	return out, rows.Err()
}
