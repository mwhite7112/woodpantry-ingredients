# woodpantry-ingredients

Ingredient Dictionary Service for WoodPantry. The canonical ingredient de-duplication and normalization registry — ensures "garlic", "garlic clove", and "minced garlic" all resolve to the same ID.

The Dictionary starts empty and grows organically through write-through from the Recipe and Pantry ingest flows.

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/healthz` | Health check |
| GET | `/ingredients` | List all ingredients |
| POST | `/ingredients` | Manually create a canonical ingredient |
| GET | `/ingredients/:id` | Fetch ingredient by ID |
| PUT | `/ingredients/:id` | Update ingredient (e.g. add aliases) |
| POST | `/ingredients/resolve` | Resolve raw text to canonical ID (write-through) |
| POST | `/ingredients/merge` | Merge two near-duplicate entries |

### POST /ingredients/resolve

The critical path endpoint. Accepts a raw ingredient name string, runs fuzzy matching against existing entries and aliases. Returns the best match above the confidence threshold, or auto-creates a new entry if no match is found.

```json
// Request
{ "name": "garlic clove" }

// Response — existing match
{ "id": "uuid", "name": "garlic", "confidence": 0.94, "created": false }

// Response — new entry auto-created
{ "id": "uuid", "name": "garlic clove", "confidence": 0.0, "created": true }
```

### POST /ingredients/merge

Merges two entries. The losing entry's name is added as an alias on the winner. All foreign key references in Recipe and Pantry services must be updated by the caller.

```json
{ "winner_id": "uuid-a", "loser_id": "uuid-b" }
```

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `PORT` | `8080` | HTTP listen port |
| `DB_URL` | required | PostgreSQL connection string for `dictionary_db` |
| `RESOLVE_THRESHOLD` | `0.8` | Fuzzy match threshold — below this, auto-create |
| `LOG_LEVEL` | `info` | Log level |

## Development

```bash
# Run migrations
go run ./cmd/ingredients migrate

# Start the service
go run ./cmd/ingredients/main.go

# Generate sqlc
sqlc generate -f internal/db/sqlc.yaml
```

## Role in Architecture

Called synchronously by Recipe Service, Pantry Service, and Ingestion Pipeline before any ingredient is created or linked. No RabbitMQ dependency. All other services treat the returned `ingredient_id` as the canonical reference.
