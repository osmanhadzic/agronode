# Backend Design - AgroNode

## Overview

The backend ingests MQTT telemetry on a per-sensor basis, validates and normalizes readings, stores them in PostgreSQL, evaluates sensor triggers, and serves both REST and WebSocket clients with real-time and historical data.

---

## Architecture Layers

### 1. Transport Layer (handlers)

- HTTP handlers for REST endpoints
- WebSocket handler for real-time connections
- Request validation and error responses
- Organization scoping via `X-Organization-ID` header

Key components:

- `device_handler.go` - device registration and listing
- `telemetry_handler.go` - historical data queries
- `trigger_handler.go` - trigger CRUD operations
- `realtime_handler.go` - WebSocket upgrade and messaging

---

### 2. Service Layer (services)

Business logic and workflows:

- `device_service.go` - device registration, presence updates, sensor discovery
- `telemetry_service.go` - telemetry processing, trigger evaluation, broadcast
- `trigger_command_publishing` - MQTT activation commands

Key behaviors:

- **Device Registration**: Idempotent, defaults MQTT devices to type `publisher`
- **Presence Tracking**: Updates `device.last_seen` and status (online/offline)
- **Sensor Discovery**: Auto-discovers new sensor IDs from telemetry
- **Trigger Evaluation**: Compares sensor values against min/max thresholds
- **Activation**: Publishes commands to target device when threshold crossed
- **Live Broadcast**: Emits readings to WebSocket clients in real-time

---

### 3. Repository Layer (repositories)

Database abstractions:

- `device_repository.go` - device CRUD and queries
- `telemetry_repository.go` - telemetry reads and writes
- `trigger_repository.go` - trigger CRUD
- `sensor_repository.go` - sensor metadata management

All repositories honor organization scoping.

---

### 4. MQTT Layer (mqtt)

Message ingestion and device registration:

- `client.go` - MQTT subscriber and publisher
- Topic subscriptions: `agronode/+/telemetry`, `agronode/+/status`, `agronode/+/register`
- Payload parsing for per-sensor format
- Device auto-registration from `/register` topic
- Activation command publishing to `/activation` topics

---

### 5. Realtime Layer (realtime)

WebSocket hub for live broadcasting:

- `hub.go` - connection management and message fan-out
- Subscribers receive: telemetry, status events, trigger activations
- Connection pooling and graceful disconnects

---

### 6. Data Layer (models, database)

- `models/device.go` - device with `DeviceType` field
- `models/sensor_data.go` - telemetry record with `sensor_id` tracking
- `models/sensor_trigger.go` - trigger configuration
- `database/postgres.go` - connection pool
- `database/migrations/` - schema versioning

---

## Data Flow

### Telemetry Ingestion

```
MQTT Broker
  ↓ (agronode/{deviceId}/telemetry)
MQTT Client
  ↓ deserialize JSON
TelemetryService.HandleTelemetry()
  ├─ Validate (deviceId, sensors map)
  ├─ Infer sensorId if not explicit
  ├─ Save to TelemetryRepository
  ├─ Upsert Sensor entry
  ├─ UpdatePresence (device.status = online)
  ├─ Broadcast to realtime hub
  ├─ UpdateDiscoveredSensors
  ├─ UpdateMetadata (RSSI, firmware, etc.)
  └─ EvaluateSensorTriggers()
       ├─ For each trigger for this device
       ├─ Compare sensor value vs min/max
       ├─ If threshold crossed and not recently fired
       └─ PublishActivationCommand to MQTT
```

### Trigger Setup

```
REST POST /api/triggers/:deviceId/:sensorId
  ↓
TriggerHandler.SetSensorTrigger()
  ↓
TriggerService.SetSensorTrigger()
  ├─ Validate min/max thresholds
  ├─ Store in TriggerRepository
  └─ Cache in memory for fast checks
```

### Trigger Activation

```
Sensor value exceeds threshold
  ↓
TriggerService.EvaluateSensorTriggers()
  ├─ Load trigger from cache
  ├─ Check state (first time crossing? or already active?)
  ├─ If state changes (activation or recovery)
  └─ PublishActivationCommand()
       ↓
MQTT Client.PublishActivationCommand()
  ├─ Build JSON command
  ├─ Publish to agronode/{targetDeviceId}/activation
  └─ Emit event to WebSocket
```

### Live Data Broadcasting

```
TelemetryService.Broadcast(reading)
  ↓
RealtimeHub.Publish(TelemetryMessage)
  ├─ Iterate all WebSocket connections
  └─ Send message to each subscriber
```

---

## Per-Sensor Telemetry Model

### Data Structure

```json
{
  "deviceId": "esp32-lab",
  "sensorId": "dht11-temp",
  "timestamp": 1715539200,
  "sensors": {
    "dht11-temp": 24.5,
    "dht11-humidity": 60.5
  }
}
```

### Storage in PostgreSQL

```sql
-- Table: sensor_data
CREATE TABLE sensor_data (
  id INTEGER PRIMARY KEY,
  device_id TEXT NOT NULL,
  sensor_id TEXT,
  temperature REAL,            -- legacy fallback
  humidity REAL,               -- legacy fallback
  sensors JSONB NOT NULL,      -- canonical {"dht11-temp": 24.5, ...}
  meta JSONB,
  created_at DATETIME NOT NULL
);
```

### Retrieval Rules

1. If `sensors` JSONB has data → use that as canonical
2. If `sensors` is empty but `sensor_id` is set:
   - Try to fetch from legacy `temperature` / `humidity` columns
   - Fallback to empty map
3. Sensor value queries use `sensors` map primarily, legacy columns secondarily

---

## Device Type Classification

### Enum

```go
const (
  DeviceTypePublisher = "publisher"    // Sends telemetry
  DeviceTypeReceiver  = "receiver"     // Receives activation commands
  DeviceTypeUnknown   = "unknown"      // Unclassified
)
```

### Assignment Rules

- **MQTT `/register`** → defaults to `DeviceTypePublisher`
- **REST `/devices/register`** → respects `deviceType` field if provided
- **Manual update** → admin can override via API

### Usage

- Frontend device selector shows type as label
- Trigger target device selector can filter by type
- Future: separate activation subscriptions by device type

---

## Trigger State Machine

### Per-Sensor State

```go
type SensorTriggerState struct {
  MinActive  bool   // true if value < min threshold
  MaxActive  bool   // true if value > max threshold
}
```

### Transitions

```
Initial: MinActive=false, MaxActive=false

Value goes BELOW min:
  MinActive: false → true (activate if min configured)
  Emit: activation command

Value returns ABOVE min:
  MinActive: true → false
  Emit: recovery command (activated=false)

Value goes ABOVE max:
  MaxActive: false → true (activate if max configured)
  Emit: activation command

Value returns BELOW max:
  MaxActive: true → false
  Emit: recovery command (activated=false)
```

### Idempotency

- Activation command only published on **state change**
- Multiple readings at same value do **not** re-publish
- Ensures device doesn't reset repeatedly

---

## Live Value Caching (Frontend)

Backend broadcasts all readings to WebSocket. Frontend manages caching:

```typescript
const [liveSensorSnapshots, setLiveSensorSnapshots] = 
  useState<Record<string, Record<string, number>>>({})

// On telemetry message:
setLiveSensorSnapshots(previous => {
  const next = {...previous}
  const deviceSnapshot = next[deviceId] || {}
  
  for (const [sensorKey, value] of Object.entries(payload.sensors)) {
    deviceSnapshot[sensorKey] = value
  }
  
  next[deviceId] = deviceSnapshot
  return next
})
```

**Benefit**: Sensor cards keep displaying last known value even during network gaps.

---

## Error Handling

- **Validation errors** → 400 Bad Request with clear message
- **Not found** → 404 Not Found
- **Server errors** → 500 Internal Server Error with logs
- **MQTT connection loss** → auto-reconnect with exponential backoff
- **Database errors** → logged and surfaced to client

---

## Performance Considerations

### Indexing

- `devices.device_id` (unique)
- `sensor_data.device_id` (query filter)
- `sensor_data.created_at` (time-range queries)
- `sensor_triggers.device_id, sensor_id` (trigger lookups)

### Caching

- In-memory trigger map (fast threshold checks)
- In-memory device status map
- WebSocket connection pooling

### Downsampling

- Frontend downsamples large datasets for charts (100-point target)
- Backend returns raw data; frontend handles aggregation

---

## Configuration

Environment variables:

```bash
APP_PORT=8080                    # Server port
APP_LOG_LEVEL=info               # Structured log level
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=...
DB_NAME=agronode
MQTT_BROKER=mosquitto:1883
MQTT_TOPIC=agronode/#
MQTT_ACTIVATION_TOPIC=agronode/{deviceId}/activation
DEFAULT_ORG_ID=1                 # For MQTT-registered devices
```

---

## Graceful Shutdown

1. Stop accepting new HTTP requests
2. Wait for in-flight requests to complete
3. Disconnect MQTT subscriber (stop ingesting)
4. Close WebSocket connections
5. Flush remaining events to database
6. Close database connection

---

## Testing Strategy

- Unit tests for service layer (logic)
- Mocked repositories for isolation
- Integration tests with SQLite in-memory DB
- Handler tests with test HTTP server
- No external MQTT deps for tests (mocked)

---

## Future Enhancements

- Batch inserts for high-volume telemetry
- ReadModel pattern for fast queries
- Event sourcing for trigger audit trail
- Device certificate-based authentication
- Multi-device activation workflows
- Time-series specific database (InfluxDB, TimescaleDB)
