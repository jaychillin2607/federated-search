# Federated Search (POC)

Go microservice that exposes a single text-search API returning results
grouped by category, spanning both Postgres-backed resources (turfs, gyms,
coaches, classes) and static app features. OpenSearch is the search index;
Postgres is the source of truth. A background indexer keeps OpenSearch in
sync via timestamp-based polling with soft-delete handling.

## Architecture

![AWS Deployment](./aws-deployment.png)

## Local setup

```bash
docker compose up --build
```

Once Postgres + OpenSearch + server + indexer are healthy:

```bash
# Optional: seed the static features and sample resources directly into OpenSearch.
INTERNAL_API_KEY=devkey make seed
```

The indexer runs every `INDEXER_INTERVAL_SECONDS` (default 60) and ingests
from Postgres into OpenSearch.

## Configuration

| Var | Default |
|---|---|
| `PORT` | `8080` |
| `LOG_LEVEL` | `info` |
| `OPENSEARCH_URL` | `http://localhost:9200` |
| `OPENSEARCH_USERNAME` | _empty_ |
| `OPENSEARCH_PASSWORD` | _empty_ |
| `OPENSEARCH_INDEX` | `federated_search` |
| `OPENSEARCH_INSECURE_SKIP_VERIFY` | `false` |
| `POSTGRES_DSN` | `postgres://postgres:postgres@localhost:5432/gym?sslmode=disable` |
| `INTERNAL_API_KEY` | _required, no default_ |
| `INDEXER_INTERVAL_SECONDS` | `60` |
| `INDEXER_BATCH_SIZE` | `500` |

## API

### Public

```bash
curl 'http://localhost:8080/healthz'
curl 'http://localhost:8080/readyz'
curl 'http://localhost:8080/api/v1/search/categories' | jq
curl 'http://localhost:8080/api/v1/search?q=crossfit' | jq
curl 'http://localhost:8080/api/v1/search?q=cart' | jq
```

### Internal (require `X-Internal-Api-Key`)

```bash
# Upsert documents
curl -s -X POST http://localhost:8080/internal/v1/documents \
  -H 'X-Internal-Api-Key: devkey' -H 'Content-Type: application/json' \
  -d '{"documents":[{"id":"test_1","category":"gym","title":"Test Gym","deeplink":"/g/1"}]}'

# Delete one
curl -s -X DELETE -H 'X-Internal-Api-Key: devkey' \
  http://localhost:8080/internal/v1/documents/test_1

# Bulk delete
curl -s -X POST http://localhost:8080/internal/v1/documents/bulk-delete \
  -H 'X-Internal-Api-Key: devkey' -H 'Content-Type: application/json' \
  -d '{"ids":["test_1","test_2"]}'

# Sync status
curl -s -H 'X-Internal-Api-Key: devkey' \
  http://localhost:8080/internal/v1/sync/status | jq

# Reindex (resets last_synced_at to epoch so the next tick picks up everything)
curl -s -X POST -H 'X-Internal-Api-Key: devkey' -H 'Content-Type: application/json' \
  http://localhost:8080/internal/v1/reindex -d '{"source":"turfs"}'

# Reindex all sources
curl -s -X POST -H 'X-Internal-Api-Key: devkey' -H 'Content-Type: application/json' \
  http://localhost:8080/internal/v1/reindex -d '{"source":"all"}'
```

## Adding a new resource source

1. Add a migration for the new table under `migrations/`. Include `updated_at`
   and `deleted_at` columns plus a `BEFORE UPDATE` trigger calling
   `set_updated_at()`.
2. Add a new constant in `internal/model/category.go` (and an entry in
   `DisplayNames`).
3. Append a `Source` entry to `internal/indexer/sources.go` with the SQL,
   category and a `Transform` mapping each row into a `model.Document`.
4. Restart the indexer (`docker compose restart indexer`).

The next tick will index every row in the new table (last checkpoint defaults
to epoch).
