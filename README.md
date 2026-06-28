# EstateLink Lead Engine

A production-style Go backend for ingesting, scoring, and ranking property investment leads at scale.

It ingests raw property listings (single or bulk JSON), normalises and deduplicates them, runs them through two independent scoring engines — a general **explainable lead score** and per-strategy **investment fit scores** across six strategies (buy-to-let, BRRRR, flip, buy-and-hold, HMO, development) — and exposes the results through a JWT-secured REST API with role-based access control and a full audit trail.

Built to demonstrate clean architecture, concurrent processing, and the kind of operational detail (idempotent imports, cancellable jobs, activity logging) that real production systems need but tutorials usually skip.

---

## Highlights

- **Two scoring engines, not one.** A general explainability-first lead score (0–100, grade A–D, with reasons) plus six strategy-specific scores (buy-to-let, BRRRR, flip, buy-and-hold, HMO, development) computed and stored independently for every listing.
- **Concurrent, cancellable bulk import.** Bulk JSON imports run as background jobs with a bounded worker pool (`IMPORT_WORKERS`, default 4 concurrent workers), live progress tracking, and mid-flight cancellation via context propagation.
- **Idempotent ingestion.** Re-importing the same source listing updates it in place instead of creating duplicates — enforced at the database layer with unique constraints and `ON CONFLICT` upserts, not just application logic.
- **Full audit trail.** Logins, listing ingestion, and every import lifecycle event are recorded to an admin-only activity log with actor, metadata, IP, and user agent.
- **Locked-down auth.** Public registration is hardcoded to the lowest-privilege role; role escalation is only possible through an admin-only endpoint, and there's a CLI for bootstrapping the first admin.

---

## Tech Stack

**Backend:** Go · Chi router · PostgreSQL · pgx / pgxpool · Goose migrations · JWT (`golang-jwt/jwt/v5`) · bcrypt

**Tooling:** Docker Compose · GitHub Actions (CI) · Postman

---

## Architecture

Clean/hexagonal-inspired layering, with domain logic kept independent of HTTP and database concerns:

```txt
cmd/
  api/            entrypoint, router, middleware wiring
  createadmin/    CLI to bootstrap/promote an admin user

internal/
  domain/         entities and pure business rules (no I/O)
    listing/      raw + normalised listing model
    lead/         explainable score, grade, read model, filters
    strategy/     6 investment-strategy score models
    importjob/    bulk import job state machine
    rawlisting/   raw payload tracking for imports
    activitylog/  audit log entries
    user/         account + role model

  application/    use cases that orchestrate domain + infrastructure
    auth/         register, login, refresh, role updates
    ingestlisting/    single-listing pipeline: normalise -> score -> persist
    importlistings/   bulk import orchestration: jobs, concurrency, cancellation
    scorestrategies/  the 6-strategy scoring engine
    readleads/        filtered, paginated lead queries
    logactivity/      audit log writes/reads

  infrastructure/  PostgreSQL repositories, DB connectivity
  transport/http/  Chi handlers, middleware, request/response shaping

migrations/        Goose SQL migrations
```

**Request flow for ingestion:** `POST /api/listings` → normalise listing → upsert listing → compute lead score → compute 6 strategy scores → persist all → return combined result. Bulk imports run the same pipeline per row, fanned out across a worker pool.

---

## API Endpoints

### Public

| Method | Path | Notes |
|---|---|---|
| `GET` | `/health` | Liveness check |
| `POST` | `/api/auth/register` | Always creates a `viewer` account, regardless of requested role |
| `POST` | `/api/auth/login` | Returns access + refresh JWT pair |
| `POST` | `/api/auth/refresh` | Exchanges a refresh token for a new pair |

### Authenticated (any role)

| Method | Path | Notes |
|---|---|---|
| `GET` | `/api/me` | Current user from JWT claims |
| `GET` | `/api/leads` | Paginated/filterable leads (city, postcode area, property type, source, min score) |
| `GET` | `/api/leads/{id}` | Single lead with lead score + all strategy scores |

### Admin or Analyst

| Method | Path | Notes |
|---|---|---|
| `POST` | `/api/listings` | Ingest a single listing synchronously |
| `POST` | `/api/imports/clean-listings` | Bulk-import pre-cleaned JSON listings; returns a job ID (202 Accepted) |
| `GET` | `/api/imports` | List import jobs |
| `GET` | `/api/imports/{jobId}` | Job status + progress counters |
| `POST` | `/api/imports/{jobId}/cancel` | Cancel an in-flight import job |

### Admin only

| Method | Path | Notes |
|---|---|---|
| `PATCH` | `/api/admin/users/{id}/role` | Promote/change another user's role |
| `GET` | `/api/activity-logs` | Paginated audit log |
| `GET` | `/api/activity-logs/{id}` | Single audit log entry |

Auth header for protected routes: `Authorization: Bearer <token>`.

---

## Scoring Engines

### Explainable lead score (0–100, grade A–D)

A single general-purpose score with every contributing factor returned as a `{code, message, points}` reason, so the API response always explains *why* a lead scored the way it did:

- Rental yield (up to 25 pts at 8%+)
- Below-market price (up to 25 pts at 15%+ discount)
- Seller-motivation keywords in title/description (20 pts — "motivated seller", "quick sale", "chain free", etc.)
- Days-on-market staleness (up to 15 pts)
- Data completeness (up to 15 pts)

### Strategy fit scores (6 independent scores per listing)

Each strategy starts from its own baseline and applies strategy-specific heuristics over yield, bedroom count, property type, and days on market, producing its own score, grade, and reasons:

| Strategy | Rewards |
|---|---|
| `buy_to_let` | Strong yield, 2–4 beds, flat/house |
| `brrrr` | Yield + long days-on-market (renegotiation potential) + 3+ beds |
| `flip` | Staleness + house type + 3+ beds |
| `buy_and_hold` | Usable price data, 2+ beds, known location, moderate yield |
| `hmo` | 4+ beds heavily, house type, high yield |
| `development` | Detached/semi-detached, 3+ beds, long market exposure |

Both engines run inline on every ingest and are stored independently, so strategy scores can be recomputed or extended without touching the general lead score.

---

## Bulk Import Pipeline

`POST /api/imports/clean-listings` accepts an array of pre-cleaned listing JSON and processes it as a background job:

- **Request safety limits** — the request body is capped at `MAX_REQUEST_BODY_BYTES` (413 if exceeded) and the row count is capped at `MAX_IMPORT_ROWS` (400 if exceeded). Both checks happen before an `import_jobs` row is created or any raw listing is written, so an oversized import leaves no trace in the database.
- **Concurrency** — a bounded worker pool (`IMPORT_WORKERS`, default 4) processes rows in parallel against a pool sized for it (`pgxpool` max 25 connections).
- **Cancellation** — each job owns a `context.CancelFunc`; `POST /api/imports/{jobId}/cancel` cancels it, in-flight work finishes naturally, and no new rows are dispatched.
- **Dedup / idempotency** — enforced in the database, not just application code: unique constraints on `(source, external_property_id)` for raw listings, `(source_platform, external_property_id)` for listings, and one row per `listing_id` for lead scores, with repositories using `ON CONFLICT` upserts. Re-importing the same listing updates it instead of duplicating it.
- **Partial failure tolerance** — a bad row increments the job's failed count without aborting the batch; the job only fails outright if every row fails.
- **Auditing** — `import.started` / `import.completed` / `import.failed` / `import.cancelled` events are written to the activity log with row counts and job metadata.

---

## Auth & Access Control

- JWT (HS256) access + refresh token pairs, configurable TTLs.
- Roles: `admin`, `analyst`, `viewer` — enforced both by a DB check constraint and by route middleware.
- Public registration always creates a `viewer` — there is no way to self-elevate through the API.
- Role changes only happen through the admin-only `PATCH /api/admin/users/{id}/role` endpoint, and take effect on the user's next token refresh.
- `go run ./cmd/createadmin` bootstraps the first admin (or promotes an existing user by email) from `ADMIN_EMAIL` / `ADMIN_PASSWORD` env vars — no manual SQL required to get started.

---

## Activity Logging

Every login, manual listing ingestion, and import lifecycle event is written to an append-only `activity_log` table (actor, action, entity type/id, JSONB metadata, IP, user agent). Logging is best-effort — a logging failure never blocks the action it's recording — and the log is only readable by admins via `/api/activity-logs`.

---

## Running Locally

```bash
# 1. Start PostgreSQL
docker compose up -d

# 2. Run migrations
goose -dir migrations postgres "$DATABASE_URL" up

# 3. Bootstrap an admin user
ADMIN_EMAIL=<your-email> ADMIN_PASSWORD=<your-password> go run ./cmd/createadmin

# 4. Start the API
go run ./cmd/api
```

### Environment Variables

Required (see `.env.example`):

```txt
DATABASE_URL=postgres://<user>:<password>@localhost:5432/<db>?sslmode=disable
JWT_SECRET=<long-random-secret>
```

Optional:

```txt
ACCESS_TOKEN_TTL=24h        # default 24h
REFRESH_TOKEN_TTL=168h      # default 168h (7 days)
PORT=8080                   # default 8080
ALLOWED_ORIGINS=http://localhost:5173   # CSV list for CORS
ENV_FILE=.env.staging       # load a specific env file instead of .env
MAX_IMPORT_ROWS=5000            # default 5000; rows over this are rejected with 400 before any DB write
MAX_REQUEST_BODY_BYTES=25000000 # default 25,000,000 bytes (~25MB); bodies over this are rejected with 413
IMPORT_WORKERS=4                 # default 4; bounded worker pool size for bulk import processing
```

`ENV_FILE` lets you point at any env file (`.env.staging`, `.env.production.local`, ...) without changing code — useful for running the same binary against multiple environments.

---

## Testing

```bash
go test ./...
```

Unit tests cover the auth service and token issuance, the listing ingestion pipeline, lead and strategy scoring, listing normalisation, lead read filtering, the auth middleware, and the bulk import safety limits (row count and request body size).

`scripts/smoke-staging.sh` runs a black-box smoke test against a deployed environment: health check, login, a small successful import, job listing/lookup, and an oversized import rejection. Requires `STAGING_URL`, `STAGING_EMAIL`, `STAGING_PASSWORD`.

---

## CI/CD

GitHub Actions (`.github/workflows/test.yml`) runs `go test ./...` on every push and pull request.

---

## Deployment

A multi-stage `Dockerfile` (Go builder → Alpine runtime) builds a minimal production image of the API. `docker-compose.yml` covers local development with hot reload; a production compose file is scaffolded but not yet filled in.

---

## Known Limitations / Next Up

- `docker-compose-prod.yml` is a stub — no production compose/orchestration yet.
- No rate limiting or structured metrics/tracing (logging is plain stdout, no Prometheus/OpenTelemetry).
- No OpenAPI/Swagger spec — API is documented here and via the Postman collection.
- `property_images` has a domain model and schema but no API surface yet.
- `development` strategy scoring is intentionally limited until plot size and planning data are available.
- Go version is pinned to 1.25 in the Dockerfile but 1.24 in CI — worth aligning.

---

## Why This Project

Built to practice and demonstrate the parts of backend engineering that go beyond CRUD: explainable scoring logic, concurrent job processing with cancellation, idempotent data ingestion, role-based access control done defensively (no client-side trust for privilege), and an audit trail that would actually hold up in a real production incident review.
