# Weeto Backend

Go API and background worker for [Weeto](docs/PRD.md) — interview scheduling for hiring teams in Iran.

## Stack

- **HTTP:** Go standard library (`net/http`)
- **Database:** PostgreSQL
- **Migrations:** `db/migrations/` (goose or golang-migrate)
- **Queries:** `db/queries/` → `sqlc generate` → `internal/db/`
- **Driver:** pgx

## Layout

```
cmd/
  api/          HTTP server entrypoint
  worker/       Notification outbox processor

db/
  migrations/   SQL schema migrations (source of truth)
  queries/    Hand-written SQL for sqlc

internal/
  config/       Environment and app configuration
  db/           sqlc-generated data access (do not edit by hand)
  handler/      HTTP handlers by domain
  middleware/   Auth, logging, request context
  service/      Business logic
  platform/     External integrations (Google, SMS, email)

test/
  integration/  Handler and DB integration tests
  fixtures/     Test data helpers
```

## Docs

Product requirements: [docs/PRD.md](docs/PRD.md)
