CREATE TABLE IF NOT EXISTS search_sync_state (
  source_name        TEXT PRIMARY KEY,
  last_synced_at     TIMESTAMPTZ NOT NULL DEFAULT '1970-01-01T00:00:00Z',
  last_success_count INT NOT NULL DEFAULT 0,
  last_error         TEXT,
  updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
