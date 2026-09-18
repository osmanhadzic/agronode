# AgroNode Backend

Go-based REST API and real-time WebSocket service for handling MQTT device telemetry, sensor data ingestion, trigger evaluation, and multi-tenant data management.

---

## Quick Start

### Prerequisites

- Go 1.20+
- PostgreSQL 14+
- MQTT Broker (Mosquitto or EMQX)
- Docker & Docker Compose (for containerized development)

### Local Development

```bash
# Clone and navigate
cd agronode/backend

# Install dependencies
go mod download

# Set up environment
cp .env.example .env
# Edit .env with your database and MQTT credentials

# Run migrations
go run ./cmd/api migrate

# Start server
go run ./cmd/api
# Server runs on http://localhost:8080
```

### Docker Compose (Recommended)

```bash
cd agronode
docker compose up -d
# Backend available at http://localhost:8080
```

---

## Architecture

### Directory Structure

```
backend/
├── cmd/api/                    # Entry point
│   └── main.go
├── internal/
│   ├── config/                 # Configuration loading
│   ├── database/               # Database setup, migrations
│   │   └── migrations/
│   ├── handlers/               # HTTP/WebSocket handlers
│   ├── models/                 # Data types
│   ├── mqtt/                   # MQTT client and parsing
│   ├── realtime/               # WebSocket hub
│   ├── repositories/           # Database access abstractions
│   ├── server/                 # Router setup
│   ├── services/               # Business logic
│   └── workers/                # Background workers
├── docs/
│   ├── API.md                  # Endpoint reference
│   ├── DESIGN.md               # Architecture details
├── go.mod
└── go.sum
```

### Core Components

**1. Transport Layer** (`handlers/`)
- HTTP REST endpoints for devices, telemetry, triggers
- WebSocket handler for real-time connections
- Request validation and error handling
- Organization scoping via `X-Organization-ID` header

**2. Service Layer** (`services/`)
- Device registration and presence management
- Telemetry validation and normalization
- Trigger evaluation logic
- Real-time broadcasting
- MQTT command publishing

**3. Data Layer** (`repositories/`)
- Device CRUD operations
- Telemetry reads/writes
- Trigger persistence
- Organization-scoped queries

**4. MQTT Integration** (`mqtt/`)
- Subscriber for `agronode/+/telemetry`, `agronode/+/status`, `agronode/+/register`
- Message parsing for per-sensor format
- Device auto-registration
- Activation command publishing

**5. Realtime Broadcasting** (`realtime/`)
- WebSocket connection pooling
- Message fan-out to subscribers
- Graceful connection lifecycle management

---

## API Overview

### Devices

- `GET /api/devices` - List all devices
- `GET /api/devices/:id` - Device details with metadata
- `POST /api/devices/register` - Register device via HTTP
- `PUT /api/devices/:id` - Update device properties

### Telemetry

- `GET /api/telemetry/:deviceId` - Historical data (query params: `from`, `to`, `sensorId`, `limit`)
- `GET /api/latest/:deviceId` - Latest readings (last 10 minutes)
- `POST /api/telemetry` - Publish telemetry (admin only)

### Triggers

- `GET /api/triggers/:deviceId` - List triggers for device
- `POST /api/triggers/:deviceId/:sensorId` - Create/update trigger
- `DELETE /api/triggers/:deviceId/:sensorId` - Delete trigger
- `GET /api/triggers/:deviceId/:sensorId` - Trigger details

### WebSocket

- `GET /ws/telemetry` - Real-time telemetry stream
  - Message format: `{ type: "telemetry", payload: {...} }`

### Health & Status

- `GET /api/health` - Server health check
- `GET /api/status` - System status (connections, uptime)

**Full API documentation**: See [docs/API.md](docs/API.md)

---

## Data Model

### Device

```go
type Device struct {
  ID              string    // Unique identifier
  Name            string    // Display name
  DeviceType      string    // "publisher" | "receiver" | "unknown"
  Status          string    // "online" | "offline"
  LastSeen        time.Time // Last telemetry timestamp
  DiscoveredSensors []string // Auto-discovered sensor IDs
  Metadata        JSONB     // Custom fields (RSSI, firmware, etc.)
  OrganizationID  string    // Tenant scoping
  CreatedAt       time.Time
  UpdatedAt       time.Time
}
```

### Sensor Data

```go
type SensorData struct {
  ID          int64             // Row ID
  DeviceID    string            // Device reference
  SensorID    string            // Primary sensor (nullable)
  Sensors     map[string]float64 // Canonical per-sensor map (JSONB)
  Timestamp   time.Time
  Metadata    JSONB             // Extra fields
  OrganizationID string
}
```

### Sensor Trigger

```go
type SensorTrigger struct {
  ID            string    // Unique ID
  DeviceID      string    // Source device
  SensorID      string    // Sensor to monitor
  MinThreshold  *float64  // Optional min boundary
  MaxThreshold  *float64  // Optional max boundary
  TargetDeviceID string   // Activation target (optional, defaults to source)
  Active        bool      // Is currently triggered
  OrganizationID string
  CreatedAt     time.Time
  UpdatedAt     time.Time
}
```

### Database naming (generic)

Asset hierarchy is now database-generic (not agriculture-specific) while API JSON fields remain backward compatible.

**Table mapping**

- `farms` -> `asset_groups`
- `fields` -> `asset_sections`
- `zones` -> `asset_units`

**Column mapping**

- `asset_sections.farm_id` -> `asset_sections.asset_group_id`
- `asset_units.field_id` -> `asset_units.asset_section_id`
- `devices.zone_id` -> `devices.asset_unit_id`

**Migration sequence**

- `000019`: consolidated rename migration (tables, columns, indexes, constraints, sequences)

This migration is idempotent and safe to run on mixed environments where old names may still exist.

---

## Configuration

Environment variables (see [`.env.example`](.env.example)):

```bash
# Server
APP_PORT=8080                    # HTTP server port
APP_LOG_LEVEL=info               # Log verbosity

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=<password>
DB_NAME=agronode
DB_SSL_MODE=disable              # Development; use "require" in production

# MQTT
MQTT_BROKER=mosquitto:1883       # Broker address
MQTT_USERNAME=                   # Optional auth
MQTT_PASSWORD=
MQTT_TOPIC=agronode/#            # Subscribe pattern
MQTT_ACTIVATION_TOPIC=agronode/{deviceId}/activation

# Organization
DEFAULT_ORG_ID=1                 # Used for MQTT-registered devices (no auth header)
```

### Logging

Structured JSON logs via stdlib `slog`:

```json
{
  "time": "2025-01-15T10:30:00Z",
  "level": "INFO",
  "logger": "backend",
  "msg": "telemetry received",
  "device_id": "esp32-lab",
  "sensor_count": 4,
  "duration_ms": 5
}
```

---

## Data Flow

### Telemetry Ingestion Pipeline

```
MQTT Publish
├─ Topic: agronode/{deviceId}/telemetry
├─ Payload: { deviceId, sensorId?, sensors: {...} }
├─ Broker receives message
└─ Backend subscriber processes

Backend Processing
├─ Validate: deviceId and sensors map not empty
├─ Infer: sensorId from first sensor if missing
├─ Upsert: Device (if new)
├─ Save: SensorData record to database
├─ Update: Device lastSeen, status, discoveredSensors
├─ Broadcast: Message to all WebSocket subscribers
└─ Evaluate: Sensor triggers
   ├─ For each trigger for this sensorId
   ├─ Compare: value vs minThreshold and maxThreshold
   ├─ Check: state change (not already active)
   └─ Publish: Activation command if threshold crossed
```

### Trigger Activation Flow

```
Backend evaluates trigger
├─ Load trigger from memory cache
├─ Current value crosses threshold (false → true)
├─ Check: not already in fire state
└─ Publish activation command

MQTT Publish
├─ Topic: agronode/{targetDeviceId}/activation
├─ Payload: { deviceId, sensorId, activated: true }
├─ Emit: Event to WebSocket clients
└─ Target device receives command (typically a receiver type)

Recovery
└─ When value returns below threshold
   ├─ Publish deactivation (activated: false)
   └─ Update trigger state in memory
```

### WebSocket Broadcasting

```
TelemetryService publishes
    ↓
RealtimeHub receives message
    ↓
Iterates all active subscriptions
    ↓
Sends JSON payload to each client connection
    ↓
Message Format:
    { type: "telemetry"|"status"|"activation", payload: {...} }
```

---

## Testing

### Run Tests

```bash
# All tests
go test ./...

# With coverage
go test -cover ./...

# Verbose output
go test -v ./...

# Specific package
go test ./internal/services

# Run benchmarks
go test -bench=. ./...
```

### Test Organization

- Unit tests alongside source files (`*_test.go`)
- Mocked repositories for service layer tests
- Integration tests use SQLite in-memory DB
- No external MQTT/Postgres dependencies in test suite

### Example: Device Service Test

```go
func TestRegisterDevice(t *testing.T) {
  repo := &MockDeviceRepository{}
  service := NewDeviceService(repo)
  
  device, err := service.RegisterDevice("esp32-lab", "publisher")
  
  assert.NoError(t, err)
  assert.Equal(t, "esp32-lab", device.ID)
  assert.Equal(t, "publisher", device.DeviceType)
}
```

---

## Deployment

### Docker Compose

```bash
# Development stack
docker compose up -d

# With demo device
docker compose --profile demo up -d

# Stop services
docker compose down

# View logs
docker compose logs -f backend
```

### Kubernetes (Future)

- Helm charts planned in `/infra/k8s`
- Stateless API design ready for horizontal scaling
- PostgreSQL as stateful dependency

### Production Checklist

- [ ] Use managed PostgreSQL (AWS RDS, Azure Database)
- [ ] Use managed MQTT broker (AWS IoT Core, Azure IoT Hub)
- [ ] Enable SSL/TLS: `DB_SSL_MODE=require`, MQTT TLS
- [ ] Set strong `DB_PASSWORD` and MQTT credentials
- [ ] Use environment variable secrets (not .env files)
- [ ] Enable API authentication (JWT, API keys)
- [ ] Configure CORS for frontend domain
- [ ] Set up monitoring and alerting
- [ ] Run database backups
- [ ] Review trigger evaluation performance under load

---

## Scaling

### Horizontal Scaling (Multiple Backend Instances)

- Stateless REST API: run multiple instances behind load balancer
- MQTT subscriber: one instance per topic partition (or use broker clustering)
- Realtime Hub: connect via Redis Pub/Sub bridge (future enhancement)

### Vertical Scaling

- Database indexing: ensure indexes on `device_id`, `created_at`
- Connection pooling: tune `DB_MAX_OPEN_CONNS`
- MQTT subscriptions: filter topics at broker if possible
- Caching: in-memory trigger cache reduces DB lookups

### Monitoring Metrics

- HTTP request latency (p50, p95, p99)
- MQTT message lag
- Database query time
- WebSocket active connections
- Trigger evaluation time per sensor

---

## Development Workflow

### Adding a New Endpoint

1. **Define model** in `internal/models/`
2. **Create repository** in `internal/repositories/`
3. **Add service logic** in `internal/services/`
4. **Write handler** in `internal/handlers/`
5. **Register route** in `internal/server/router.go`
6. **Write tests** alongside logic
7. **Update API docs** in `docs/API.md`

### Debugging

```bash
# Enable debug logging
APP_LOG_LEVEL=debug go run ./cmd/api

# Inspect database directly
docker exec agronode-postgres-1 psql -U postgres -d agronode
  \dt                    # List tables
  SELECT * FROM devices; # Query devices

# Monitor MQTT messages
docker exec mosquitto mosquitto_sub -t "agronode/#" -v

# Use curl for API testing
curl -H "X-Organization-ID: 1" http://localhost:8080/api/devices | jq
```

---

## Troubleshooting

### Database Connection Errors

```
error: "failed to connect to database"
```

- Check PostgreSQL is running: `docker ps | grep postgres`
- Verify credentials in `.env`
- Test connection: `psql -h localhost -U postgres -d agronode`

### MQTT Connection Errors

```
error: "failed to subscribe to MQTT"
```

- Check Mosquitto is running: `docker ps | grep mosquitto`
- Verify broker address in `MQTT_BROKER`
- Test connection: `mosquitto_sub -h mosquitto -t "test"`

### Telemetry Not Appearing

- Check backend logs: `docker logs agronode-backend-1 | grep -i telemetry`
- Verify MQTT message format: `sensorId` and `sensors` map must be present
- Check organization ID: MQTT devices use `DEFAULT_ORG_ID`

### Triggers Not Firing

- Verify trigger exists: `curl http://localhost:8080/api/triggers/{deviceId}`
- Check sensor ID matches exactly
- Ensure target device has type `"receiver"`
- Monitor logs: `docker logs agronode-backend-1 | grep -i trigger`

---

## Performance Tips

1. **Query Optimization**: Use `from` and `to` parameters to limit date range
2. **Sensor Filtering**: Query by `sensorId` to reduce result set
3. **Batching**: Publish multiple sensors in single MQTT message
4. **Downsampling**: Frontend aggregates large datasets; backend returns raw
5. **Connection Pooling**: Tune `DB_MAX_OPEN_CONNS` (default: 25)

---

## Contributing

- Follow Go conventions (gofmt, golint)
- Write tests for new features
- Update API docs
- Commit with clear messages

---

## License

See [LICENSE](../../LICENSE)

---

## References

- [API Documentation](docs/API.md)
- [Design Document](docs/DESIGN.md)
- [MQTT Contract](../MQTT_CONTRACT.md)
- [Architecture Overview](../ARCHITECTURE.md)
