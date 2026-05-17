# EstateLink Lead Engine

A production-style Go backend portfolio project for ingesting, scoring, ranking, and managing property investment leads.

The system ingests property listings, normalises data, calculates explainable investment opportunity scores, stores results in PostgreSQL, and exposes a secure REST API with JWT authentication and role-based access control.

---

# Tech Stack

## Backend

- Go
- Chi Router
- PostgreSQL
- pgx / pgxpool
- Goose migrations
- JWT authentication
- bcrypt password hashing

## Tooling

- Docker Compose
- GitHub Actions
- Postman
- PowerShell

---

# Architecture

The project follows a clean, production-inspired architecture:

```txt
cmd/api
internal/application
internal/domain
internal/infrastructure
internal/transport
migrations
```

## Layers

### Domain

Contains core business entities and rules.

Examples:

```txt
listing
lead
user
```

### Application

Contains business use cases and orchestration logic.

Examples:

```txt
ingestlisting
auth
```

### Infrastructure

External implementations:

```txt
PostgreSQL repositories
database connectivity
```

### Transport

HTTP handlers, middleware, request/response handling.

---

# Features

# Sprint 0 — Foundation ✅

Implemented:

- Go module setup
- Clean project structure
- Listing domain model
- Listing normalisation
- Explainable lead scoring engine
- PostgreSQL integration
- Goose migrations
- Listing persistence
- Lead score persistence
- `POST /api/listings`
- `/health` endpoint
- Docker Compose setup
- Automated tests

---

# Sprint 1 — Auth + Users ✅

Implemented:

- Users table
- bcrypt password hashing
- JWT authentication
- Register endpoint
- Login endpoint
- Auth middleware
- Role middleware
- `GET /api/me`
- Role-based protected routes

Roles:

```txt
admin
analyst
viewer
```

Protected endpoints:

```txt
POST /api/listings
```

Allowed roles:

```txt
admin
analyst
```

---

# API Endpoints

## Health

```http
GET /health
```

---

## Register

```http
POST /api/auth/register
```

Example request:

```json
{
  "email": "admin@estatelink.dev",
  "password": "Password123!",
  "role": "admin"
}
```

---

## Login

```http
POST /api/auth/login
```

Example request:

```json
{
  "email": "admin@estatelink.dev",
  "password": "Password123!"
}
```

Example response:

```json
{
  "token": "jwt-token",
  "user": {
    "id": "uuid",
    "email": "admin@estatelink.dev",
    "role": "admin"
  }
}
```

---

## Current User

```http
GET /api/me
```

Requires:

```txt
Authorization: Bearer <token>
```

---

## Ingest Listing

```http
POST /api/listings
```

Requires:

```txt
Authorization: Bearer <token>
```

Allowed roles:

```txt
admin
analyst
```

---

# Running Locally

## Start PostgreSQL

```bash
docker compose up -d
```

---

## Run migrations

```bash
goose -dir migrations postgres "postgres://estatelink:estatelink_local_dev@localhost:5433/estatelink?sslmode=disable" up
```

---

## Start API

```bash
go run ./cmd/api
```

---

# Environment Variable

Required:

```txt
DATABASE_URL
```

Example:

```txt
postgres://estatelink:estatelink_local_dev@localhost:5433/estatelink?sslmode=disable
```

---

# Running Tests

```bash
go test ./...
```

---

# CI/CD

GitHub Actions automatically runs:

```bash
go test ./...
```

on:

- push
- pull request

---

# Postman

The API is tested through Postman collections covering:

- authentication
- role protection
- listing ingestion
- negative authorization scenarios

---

# Roadmap

## Sprint 2 — Lead Read API

Planned:

- `GET /api/leads`
- `GET /api/leads/{id}`
- pagination
- score ranking
- filters
- score reasons

## Sprint 3 — Elite Search

Planned filters:

- city
- postcode area
- property type
- price range
- bedrooms
- score
- yield
- days on market

## Sprint 4 — Score History

- rescoring
- score history tracking
- grade change detection

## Sprint 5 — Event Log System

- event logs
- lead lifecycle events
- event feed API

## Sprint 6 — CSV Import Pipeline

- CSV ingestion
- validation
- batch processing
- import reporting

## Sprint 7 — Observability

- structured logging
- metrics
- panic recovery
- request IDs
- DB health checks

## Sprint 8 — Frontend Dashboard

- authentication UI
- lead dashboard
- search UI
- event feed

## Sprint 9 — Production Add-ons

- RabbitMQ
- Redis
- Prometheus
- OpenTelemetry
- gRPC
- CI/CD pipeline improvements

---

# Known Future Improvements

- Public registration should default to `viewer`
- Admin-only role promotion endpoint
- Refresh tokens
- Rate limiting
- Structured audit logging
- OpenAPI / Swagger documentation
- Full integration testing
- Dockerised production deployment

---

# Goal of the Project

The purpose of this project is to demonstrate:

- backend engineering fundamentals
- clean architecture
- production-style API design
- authentication and authorization
- PostgreSQL integration
- testing practices
- scalable service structure
- DevOps awareness
- real-world backend patterns
