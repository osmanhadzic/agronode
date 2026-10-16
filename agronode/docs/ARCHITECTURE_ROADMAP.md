# AgroNode 2.0 – architecture roadmap

Target: the "AgroNode 2.0 Technical Architecture" diagram. This document compares it with the current state and splits the work into phases.

## Current vs. target

| Area | Current | Target (2.0) | Status |
|------|---------|--------------|--------|
| Devices | ESP32/ESP8266 firmware (sensor publisher, activation receiver) | Same, with X.509 identity | Partial |
| MQTT broker | VerneMQ 1.12.3, anonymous, no TLS, ACL `topic #` | mTLS, per-device ACL | Partial |
| Ingestion | Inside the Go backend (monolith) | Separate MQTT Ingestion service, gRPC to core | Missing |
| Backend / rules | Go backend, fuzzy trigger engine | Same + Automation engine | Present |
| Storage | PostgreSQL 16 (relational) + TimescaleDB (one `ts_<sensor>` hypertable per sensor, created on first reading) | Same | Present |
| AI | .NET MCP server + Ollama | Same | Present |
| Frontend | React/TS dashboard, triggers, fuzzy editor | + Automation, Alerts, Integrations, Pairing pages | Partial |
| Observability | Prometheus, Grafana, Loki, Promtail, Alertmanager | + OpenTelemetry, Tempo traces | Partial |
| Integrations | HTTP webhook fuzzy action | Weather (Open-Meteo/OpenWeather), Email/Telegram, Home Assistant | Missing |
| Deployment | docker-compose | Kubernetes (future) | Missing |

## Phases

### Phase 1 – TimescaleDB
- **Goal:** efficient time-series storage.
- **Status:** done. Two databases: `postgres` (devices, users, triggers, ...) and `timescaledb` (`ts_*` per-sensor hypertables plus the `sensor_series_tables` registry, port 5433, `TSDB_*` env vars). The backend uses two connections; org scoping is resolved against Postgres, readings are queried in TimescaleDB (no cross-DB FKs/joins). Migrations: `internal/database/migrations` (Postgres) and `internal/database/tsmigrations` (Timescale). An old non-empty Postgres `sensor_data` is renamed to `sensor_data_legacy`.
- **Remaining:** retention policy, continuous aggregates per sensor table. The old `sensor_data` table and `sensor_data_hourly` were dropped (migration `tsmigrations/000004`); device meta is no longer stored with readings.
- **Dependencies:** none. **Risk:** data migration of existing rows; keep a backup. **Effort:** ~2–3 days.

### Phase 2 – Device identity and broker security
- **Goal:** only paired devices can publish/subscribe.
- **Deliverables:** internal CA, X.509 per device, pairing flow (provisioning token → certificate), VerneMQ TLS 8883 with `require_certificate`, per-device ACL (`agronode/%c/#`), firmware TLS support, disable anonymous.
- **Dependencies:** none (Phase 5 Pairing page depends on it). **Risks:** ESP8266 memory with TLS, certificate rotation/expiry, time sync on devices (NTP). **Effort:** ~1–2 weeks.

### Phase 3 – Ingestion split and gRPC
- **Goal:** scale ingestion independently of the API.
- **Deliverables:** `ingestion` Go service (MQTT subscribe, validation, batching), protobuf contract, gRPC to core, backpressure and retry, metrics.
- **Dependencies:** Phase 1 recommended. **Risk:** more moving parts; duplicate/ordering handling. **Effort:** ~1–2 weeks.

### Phase 4 – OpenTelemetry and Tempo
- **Goal:** end-to-end trace MQTT message → rule evaluation → activation.
- **Deliverables:** OTel SDK in Go and .NET, Tempo + collector in compose, Grafana datasource, trace IDs in logs (Loki correlation).
- **Dependencies:** Phase 3 gives the most value. **Effort:** ~3–4 days.

### Phase 5 – Integrations and UI
- **Goal:** automation beyond the dashboard.
- **Deliverables:**
  - Weather provider (Open-Meteo first, no API key) as a virtual sensor usable in fuzzy rules.
  - Notification channels: Telegram, Email; alert rules → Alertmanager/backend.
  - Home Assistant (MQTT discovery or webhook).
  - Frontend pages: Automation, Alerts, Integrations, Pairing.
  - Fuzzy action target device field in the Go `Action` struct.
- **Dependencies:** Phase 2 for Pairing. **Risks:** external API limits, secret storage. **Effort:** ~2–3 weeks.

### Phase 6 – Kubernetes (future)
- **Goal:** production deployment.
- **Deliverables:** Helm chart/manifests, secrets, ingress with TLS, VerneMQ cluster, Timescale operator or managed DB, CI/CD.
- **Dependencies:** Phases 1–4. **Effort:** ~2+ weeks.

## Suggested order

1 → 2 → 3 → 4 → 5 → 6. Phase 5 weather and notification work can start in parallel after Phase 1.

## Already done (supporting)

- Fuzzy engine with 7 membership types, shared validation in the frontend.
- ESP32 activation receiver parses `fuzzy:` triggers (see `firmware/esp32_activation_receiver`).
