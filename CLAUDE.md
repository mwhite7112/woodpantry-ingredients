# woodpantry-ingredients — Ingredient Dictionary Service

## Role in Architecture

The canonical ingredient de-duplication and normalization registry. This is the shared vocabulary layer that all other services reference. Its primary job is ensuring "garlic", "garlic clove", and "minced garlic" all resolve to the same ID.

The Dictionary is **not pre-seeded**. It starts empty and grows organically through write-through from the Recipe and Pantry ingest flows. Only ingredients that actually get cooked or bought end up here — no noise.

This service is called **synchronously** by Recipe Service, Pantry Service, and Ingestion Pipeline before any ingredient is created or linked. It has no RabbitMQ dependency.

## Technology

- Language: Go
- HTTP: chi
- Database: PostgreSQL (`dictionary_db`) via sqlc
- No RabbitMQ (called synchronously by other services)

## Service Dependencies

- **Called by**: Recipe Service, Pantry Service, Ingestion Pipeline
- **Calls**: none
- **Publishes**: nothing
- **Subscribes to**: nothing

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/ingredients` | List all ingredients |
| POST | `/ingredients` | Manually create a canonical ingredient |
| GET | `/ingredients/:id` | Fetch by ID |
| PUT | `/ingredients/:id` | Update — primary use is adding aliases after a merge |
| POST | `/ingredients/resolve` | **Critical path.** Takes raw text string, returns best match with confidence score. If below threshold, auto-creates and returns new entry. |
| POST | `/ingredients/merge` | Merge two entries — moves references, adds losing name as alias on winner. |

## Key Patterns

### Write-Through Resolve
`POST /ingredients/resolve` is the core of this service. Every ingest flow calls it before creating or linking an ingredient. The caller always receives a canonical `ingredient_id` — either an existing one or a freshly auto-created one. The caller never has to worry about whether the ingredient already existed.

### Fuzzy Matching
Before auto-creating a new ingredient entry, the resolve endpoint runs fuzzy matching (e.g. trigram similarity or Levenshtein distance) against existing names and aliases. If similarity is above `RESOLVE_THRESHOLD`, the existing entry is returned. If below threshold, a new entry is auto-created.

### Concurrent Write Safety
Two simultaneous ingest jobs may both try to create the same new ingredient. Handle this with a DB unique constraint on normalized ingredient name and upsert semantics (`ON CONFLICT DO NOTHING` or `ON CONFLICT DO UPDATE`). The last caller to resolve gets the same entry as the first — no duplicates.

### Alias Accumulation
When a user confirms a near-match merge ("garlic clove" → "garlic"), the losing name is added to the winner's `aliases` array. This merge never needs to happen again for that pair. Over time the Dictionary converges to reflect exactly what you cook with.

## Data Models

```
ingredients
  id              UUID  PK
  name            TEXT  UNIQUE (normalized, lowercase)
  aliases         TEXT[]
  category        TEXT  -- produce|dairy|protein|pantry|spice|liquid|other
  default_unit    TEXT
  created_at      TIMESTAMPTZ

unit_conversions
  ingredient_id   UUID  FK
  from_unit       TEXT
  to_unit         TEXT
  factor          FLOAT8

ingredient_substitutes
  ingredient_id   UUID  FK  -- the original
  substitute_id   UUID  FK  -- the acceptable substitute
  ratio           FLOAT8
  notes           TEXT
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP listen port |
| `DB_URL` | required | PostgreSQL connection string |
| `RESOLVE_THRESHOLD` | `0.8` | Fuzzy match confidence threshold (0.0–1.0). Below this, auto-create. |
| `LOG_LEVEL` | `info` | Logging level |

## Directory Layout

```
woodpantry-ingredients/
├── cmd/ingredients/main.go
├── internal/
│   ├── api/
│   │   └── handlers.go        ← chi route handlers
│   ├── db/
│   │   ├── migrations/
│   │   ├── queries/           ← .sql files for sqlc
│   │   └── sqlc.yaml
│   └── service/
│       ├── resolve.go         ← fuzzy matching + write-through logic
│       └── merge.go           ← merge flow
├── kubernetes/
├── Dockerfile
├── go.mod
└── go.sum
```

## What to Avoid

- Do not pre-seed this DB with any external ingredient dataset.
- Do not add RabbitMQ — this service is synchronous only.
- Do not allow callers to create ingredients by bypassing `/ingredients/resolve` — all creation should go through the resolve endpoint to enforce dedup.
- Do not make the fuzzy threshold so high that every ingredient gets flagged for merge review — aim for a threshold that catches obvious near-duplicates only.
