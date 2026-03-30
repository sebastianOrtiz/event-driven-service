# Event-Driven Onboarding Service

Servicio de onboarding basado en eventos que demuestra arquitectura event-driven usando **Go**, **Redis Streams** y **PostgreSQL**.

## Arquitectura

Cuando un usuario se registra en NexusCRM, este servicio procesa un flujo de onboarding asincrono con 4 pasos secuenciales:

```
user.registered
    |
    v
[Worker 1: Verify Email]
    |
    v
email.verified
    |
    v
[Worker 2: Create Organization]
    |
    v
organization.created
    |
    v
[Worker 3: Provision Demo Data]
    |
    v
demo_data.provisioned
    |
    v
[Worker 4: Send Welcome Email]
    |
    v
onboarding.completed
```

Cada paso es un **worker** que consume de un Redis Stream, procesa el evento, lo registra en PostgreSQL (event store) y publica el siguiente evento en la cadena.

## Stack

| Componente | Tecnologia |
|---|---|
| Lenguaje | Go 1.23 |
| Mensajeria | Redis Streams |
| Persistencia | PostgreSQL 16 (schema `events`) |
| Driver DB | pgx/v5 |
| Cliente Redis | go-redis/v9 |
| Logging | log/slog (JSON estructurado) |
| HTTP | net/http (sin frameworks) |

## Estructura del proyecto

```
event-driven-service/
├── cmd/
│   ├── api/main.go           # HTTP API
│   └── worker/main.go        # Worker (4 goroutines)
├── internal/
│   ├── config/               # Configuracion desde env vars
│   ├── events/               # Tipos y constantes de eventos
│   ├── models/               # Modelos de dominio
│   ├── store/                # PostgreSQL event store
│   ├── publisher/            # Publicacion a Redis Streams
│   ├── consumer/             # Consumer groups con reintentos
│   ├── handlers/             # Logica de cada paso del onboarding
│   └── api/                  # Rutas y handlers HTTP
├── Dockerfile                # Multi-stage build
└── go.mod
```

## Configuracion

Variables de entorno con valores por defecto:

| Variable | Default | Descripcion |
|---|---|---|
| `REDIS_URL` | `localhost:6379` | Direccion del servidor Redis |
| `DATABASE_URL` | `postgres://sebasing:devpassword@localhost:5432/sebasing_dev?sslmode=disable` | Connection string PostgreSQL |
| `HTTP_PORT` | `8081` | Puerto del servidor HTTP |
| `DB_SCHEMA` | `events` | Schema de PostgreSQL |
| `MAX_RETRIES` | `3` | Reintentos maximos por evento |
| `RETRY_BACKOFF_MS` | `1000` | Backoff base en milisegundos |
| `CONSUMER_GROUP` | `onboarding-workers` | Nombre del consumer group |

## Como ejecutar

### Requisitos

- Go 1.23+
- PostgreSQL 16
- Redis 7

### Build

```bash
go build -o bin/api ./cmd/api
go build -o bin/worker ./cmd/worker
```

### Ejecutar

```bash
# Terminal 1: API
./bin/api

# Terminal 2: Workers
./bin/worker
```

### Docker

```bash
# Build
docker build -t event-driven-service .

# API
docker run -p 8081:8081 event-driven-service api

# Workers
docker run event-driven-service worker
```

## API Endpoints

### Health Check

```
GET /health
```

Respuesta:
```json
{
  "status": "ok",
  "redis": "connected",
  "postgres": "connected"
}
```

### Trigger Onboarding

```
POST /api/v1/onboarding/trigger
Content-Type: application/json

{
  "email": "user@example.com",
  "name": "John Doe",
  "orgName": "Acme Inc"
}
```

Respuesta (202 Accepted):
```json
{
  "correlationId": "550e8400-e29b-41d4-a716-446655440000",
  "status": "pending",
  "message": "onboarding flow started"
}
```

### Get Flow Status

```
GET /api/v1/onboarding/{correlation_id}
```

Respuesta:
```json
{
  "id": "...",
  "correlationId": "...",
  "userEmail": "user@example.com",
  "status": "completed",
  "startedAt": "2024-01-01T00:00:00Z",
  "completedAt": "2024-01-01T00:00:05Z",
  "createdAt": "2024-01-01T00:00:00Z"
}
```

### Get Flow Events

```
GET /api/v1/onboarding/{correlation_id}/events
```

Respuesta:
```json
{
  "correlationId": "...",
  "events": [
    {
      "id": "...",
      "flowId": "...",
      "eventType": "user.registered",
      "payload": {...},
      "status": "completed",
      "retryCount": 0,
      "createdAt": "...",
      "processedAt": "..."
    }
  ]
}
```

## Decisiones tecnicas

- **Sin frameworks HTTP**: Solo `net/http` estandar para mantener dependencias minimas
- **Idempotencia**: Cada handler verifica si el evento ya fue procesado antes de actuar
- **Correlation ID**: UUID unico por flujo para trazabilidad end-to-end
- **Reintentos con backoff exponencial**: Hasta `MAX_RETRIES` intentos con backoff `base * 2^attempt`
- **Graceful shutdown**: Captura SIGINT/SIGTERM y espera que los workers terminen
- **Event store**: Todos los eventos se persisten en PostgreSQL para auditoria
- **Consumer groups**: Cada tipo de worker usa Redis consumer groups para procesamiento distribuido
