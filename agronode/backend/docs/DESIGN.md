# Backend Design - AgroNode

## Overview

The backend ingests MQTT telemetry, validates and normalizes readings, stores them in PostgreSQL, and serves both REST and WebSocket clients.

## Modules

- `handlers`: HTTP/WebSocket transport layer
- `services`: business logic and validation
- `repositories`: persistence abstractions
- `mqtt`: MQTT subscriber client and message parsing
- `models`: data contracts between layers
- `database`: DB connection, store, and migrations
- `config`: environment and logger setup
- `realtime`: pub/sub hub for WebSocket broadcast

## Data Flow

1. Device publishes to `agronode/{deviceId}/telemetry`.
2. MQTT module subscribes to `agronode/#`.
3. Service validates telemetry and prepares canonical reading.
4. Repository stores record in PostgreSQL.
5. Service emits reading to realtime hub.
6. Handlers expose data through:
   - REST: `/api/data`, `/api/data/:deviceId`, `/api/latest/:deviceId`
   - WebSocket: `/ws/telemetry`

## Layer Rules

- Handlers stay thin and do transport concerns only.
- Services own business rules and validation.
- Repositories only handle database access.
- MQTT module does topic/payload ingestion and forwarding.
- Cross-layer communication uses explicit interfaces where useful.

## Runtime Characteristics

- Graceful shutdown via context cancellation and HTTP server shutdown.
- Structured logs through `slog` JSON handler.
- Environment-driven configuration (`APP_PORT`, DB/MQTT settings).
- Migrations run on startup.

## Scalability Direction

- Horizontal API scaling behind a load balancer.
- Multiple device fleets by topic segmentation.
- Authentication/authorization layer on API and MQTT edges.
- Enhanced analytics pipelines from stored telemetry.
