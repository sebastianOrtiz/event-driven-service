# Event-Driven Onboarding Service

Event-driven onboarding service that demonstrates asynchronous event processing using Go, Gin, Redis Streams, and PostgreSQL. When a user registers in NexusCRM, this service processes a 4-step onboarding flow.

## Stack

| Layer | Technology |
|---|---|
| Language | Go 1.23 |
| HTTP | Gin |
| Messaging | Redis Streams (consumer groups) |
| Persistence | PostgreSQL 16 (schema: `events`) |
| Driver | pgx/v5 |
| Redis client | go-redis/v9 |
| Logging | log/slog (structured JSON) |

## Architecture

The service runs as two processes: an HTTP API and a worker. The worker spawns 4 goroutines, each consuming from a Redis Stream and producing the next event in the chain.

```
user.registered
    |
    v
[Worker 1: Verify Email] --> email.verified
    |
    v
[Worker 2: Create Organization] --> organization.created
    |
    v
[Worker 3: Provision Demo Data] --> demo_data.provisioned
    |
    v
[Worker 4: Send Welcome Email] --> onboarding.completed
```

Each worker is idempotent, uses a correlation ID for end-to-end tracing, and supports retries with exponential backoff. All events are persisted in PostgreSQL as an event store.

## API Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Health check (Redis + PostgreSQL) |
| `POST` | `/api/v1/onboarding/trigger` | Start an onboarding flow |
| `GET` | `/api/v1/onboarding/:id` | Get flow status |
| `GET` | `/api/v1/onboarding/:id/events` | Get all events for a flow |

All endpoints require an `X-API-Key` header for authentication.

## Running

```bash
# Terminal 1: API (port 8081)
go run ./cmd/api

# Terminal 2: Workers
go run ./cmd/worker
```

## Configuration

| Variable | Default | Description |
|---|---|---|
| `REDIS_URL` | `localhost:6379` | Redis server address |
| `DATABASE_URL` | `postgres://sebasing:devpassword@localhost:5432/sebasing_dev?sslmode=disable` | PostgreSQL connection |
| `HTTP_PORT` | `8081` | HTTP server port |
| `DB_SCHEMA` | `events` | PostgreSQL schema |
| `MAX_RETRIES` | `3` | Max retries per event |
| `API_KEY` | — | Required API key for authentication |

## Integration with NexusCRM

The CRM API calls `POST /api/v1/onboarding/trigger` with an `X-API-Key` header when a user registers. The CRM dashboard reads onboarding status via the CRM API's proxy endpoints (`/api/v1/events/*`).

## Tests

37 tests.

```bash
go test ./...
```

## Part of sebasing.dev

| Project | Stack |
|---|---|
| [portfolio-web](../portfolio-web) | Next.js, TypeScript, Tailwind |
| [nexus-crm-api](../nexus-crm-api) | FastAPI, SQLAlchemy, PostgreSQL |
| [nexus-crm-dashboard](../nexus-crm-dashboard) | Angular, TypeScript, Tailwind |
| **event-driven-service** (this) | Go, Gin, Redis Streams |
| [semantic-search-api](../semantic-search-api) | FastAPI, ChromaDB, sentence-transformers |

## License

MIT
