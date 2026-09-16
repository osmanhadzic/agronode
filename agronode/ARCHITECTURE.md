# AgroNode Architecture

## Overview

AgroNode is a scalable IoT platform designed for greenhouse and agriculture monitoring.

It collects real-time per-sensor telemetry from ESP32 devices, transports it via MQTT, processes it in a backend service with device type classification, stores it in a database, evaluates sensor triggers, and exposes live/historical data to a frontend dashboard.

---

## System Architecture

### High-Level Flow

```
ESP32 Devices (Publisher / Receiver)
    │
    │  (MQTT Publish: telemetry, status, registration)
    ▼
MQTT Broker (Mosquitto)
    │
    │  (Subscribe & Forward)
    ▼
Go Backend (MQTT Consumer + API)
    │
    ├── Device Registry (type: publisher|receiver|unknown)
    ├── Per-Sensor Telemetry Storage
    ├── Trigger Evaluation Engine
    ├── Realtime Hub (WebSocket broadcast)
    │
    ├── PostgreSQL (Persistent Storage)
    │
    └── REST + WebSocket APIs
          │
          ▼
   React Frontend Dashboard (Tabbed UI)
   - Overview (device status & type)
   - Telemetry (historical charts per sensor)
   - Sensors (sensor discovery & metadata)
   - Triggers (min/max thresholds with target device)
```

---

## Data Flow Explanation

### 1. ESP32 Devices (Edge Layer)

Each ESP32 device is responsible for:

- Reading sensor data (temperature, humidity, etc.)
- Connecting to WiFi
- Publishing data to MQTT broker on **three channels**:
  - `/telemetry` - per-sensor readings
  - `/status` - device metadata (signal strength, uptime, firmware)
  - `/register` - self-registration (optional, on boot)

Example telemetry topic:
```txt
agronode/esp32-lab/telemetry
```

Example payload (per-sensor format):
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

Example status topic:
```txt
agronode/esp32-lab/status
```

Status payload:
```json
{
  "deviceId": "esp32-lab",
  "online": true,
  "signal_strength": -65,
  "firmware": "v1.2.3"
}
```

---

### 2. MQTT Broker (Message Layer)

We use:

- **Eclipse Mosquitto** (production-ready)

Role:

- Receives messages from ESP32 devices
- Routes messages to backend subscribers
- Decouples devices from backend
- Enables device-to-device commands (activation)

Why MQTT:

- Lightweight protocol (ideal for rural WiFi)
- Reliable delivery (QoS levels)
- Real-time communication
- Scalable for many simultaneous devices

---

### 3. Backend (Processing Layer)

The backend is written in **Go** with structured logging and modular design.

#### A) MQTT Consumer

Subscribes to:
- `agronode/+/telemetry` - device sensor readings
- `agronode/+/status` - device metadata updates
- `agronode/+/register` - self-registration requests

For each telemetry message:
1. Parse JSON and extract `deviceId`, `sensorId`, `sensors` map
2. Validate telemetry (required fields, data types)
3. Store reading in PostgreSQL with per-sensor tracking
4. Upsert sensor metadata in `sensors` table
5. Broadcast to realtime hub (WebSocket clients)
6. Evaluate sensor thresholds against stored triggers
7. If trigger fires, publish activation command to device

#### B) REST API

Exposes:
- `GET /api/devices` - paginated device list with type
- `POST /api/devices/register` - manual device registration
- `GET /api/data/:deviceId` - telemetry history for device
- `GET /api/data/:deviceId/:sensorId` - history per sensor
- `GET /api/latest/:deviceId` - latest readings (all sensors)
- `GET /api/triggers/:deviceId` - list triggers for device
- `POST /api/triggers/:deviceId/:sensorId` - create/update trigger
- `DELETE /api/triggers/:deviceId/:sensorId` - remove trigger

#### C) WebSocket (Realtime)

Endpoint: `GET /ws/telemetry`

Sends:
- Live telemetry readings as they arrive
- Device status events (online/offline transitions)
- Trigger activation events

---

### 4. Database (Persistence Layer)

We use:

- **PostgreSQL** (relational + JSON support)

Core tables:

- `devices` - device registry with `device_type` column
- `sensors` - auto-discovered sensor metadata per device
- `sensor_data` - time-series telemetry readings
- `sensor_triggers` - trigger configuration (min/max thresholds)
- `users` - user accounts (future multi-tenant)
- `organizations` - tenant/farm grouping

Key schema features:
- `sensor_data.sensor_id` tracks which sensor sent each reading
- `sensor_data.sensors` is JSONB storing full sensor map
- `devices.device_type` enables filtering by role (`publisher` vs `receiver`)
- Foreign keys maintain referential integrity
- Indexes on device_id, created_at for fast queries

---

### 5. Frontend (Presentation Layer)

React + TypeScript + Vite

#### Tabbed Dashboard:

1. **Overview Tab**
   - Device selector dropdown (shows device type)
   - Current status (online/offline)
   - Device type badge
   - Latest sensor values (cached from live)

2. **Telemetry Tab**
   - Historical time-series chart
   - Sensor filtering (checkbox list)
   - Date range picker
   - Downsampling for large datasets

3. **Sensors Tab**
   - Auto-discovered sensor cards
   - Per-sensor metadata
   - Trigger visibility link
   - Edit trigger button

4. **Triggers Tab**
   - Sensor selector
   - Min/max threshold inputs
   - Target device selector
   - Trigger history/events log

#### Live Data Behavior:

- WebSocket connects to `/ws/telemetry`
- Frontend batches incoming messages (500ms)
- Maintains cached snapshot of last value per sensor
- Cards display last known value even when no new data
- Charts show live updates in real-time

#### Device Type Display:

- Device selector shows `deviceId (deviceType)` format
- Overview tab displays device type as badge
- API response includes `deviceType` for filtering/sorting

---

## Device Classification

### DeviceType Enum

```go
const (
  DeviceTypePublisher   = "publisher"   // Sends telemetry
  DeviceTypeReceiver    = "receiver"    // Receives activation commands
  DeviceTypeUnknown     = "unknown"     // Unclassified
)
```

### Registration Path→Type Mapping

- **MQTT `/register`** → defaults to `publisher`
- **REST `/devices/register`** → can specify type (optional)
- **Manual override** → update via API with explicit type

---

## Trigger Evaluation

Triggers are per-device and per-sensor.

Configuration:
```json
{
  "deviceId": "esp32-lab",
  "sensorId": "dht11-temp",
  "min": 15.0,
  "max": 30.0,
  "targetDeviceId": "pump-node-1"
}
```

Behavior:
- When telemetry exceeds threshold, activation command fires once
- Auto-recovery when value returns to safe range
- Target device can differ from source device (cross-device automation)
- Defaults to same device if target not specified

Activation payload:
```json
{
  "deviceId": "esp32-lab",
  "trigger": "above_max",
  "sensor": "dht11-temp",
  "limitType": "max",
  "value": 35.5,
  "threshold": 30.0,
  "activated": true,
  "timestamp": 1715539200
}
```

---

## Scaling Strategy

### Phase 1 (MVP) ✅
- Single MQTT broker
- Single backend instance
- PostgreSQL single node
- Basic device registry

### Phase 2 (Growth)
- Horizontal backend scaling (load balancer)
- Device authentication (API keys, certificates)
- Multi-organizaion/farm support
- Advanced analytics

### Phase 3 (Production SaaS)
- Cloud deployment (AWS/GCP/Azure)
- Managed PostgreSQL
- Kafka for event streaming
- Real-time dashboards per user
- Mobile app support

---

## Key Design Decisions

### Why Per-Sensor Telemetry?

- Devices may have multiple sensors
- Not all sensors report at same frequency
- Flexible sensor discovery without schema changes
- Enables sensor-specific trigger logic

### Why Live Value Caching in Frontend?

- Network latency doesn't blank UI
- Better UX for slow devices
- Previous reading remains visible until new data arrives
- Especially useful for battery-powered ESP32s

### Why Device Type Classification?

- Distinguishes publishers (send data) from receivers (receive commands)
- Enables device-specific UI/filtering
- Supports mixed deployments (publishers + actuators)

### Why WebSocket + REST?

- WebSocket: real-time push for live dashboard
- REST: historical queries, filtering, triggering

### Why MQTT-based Device Autodiscovery?

- Devices self-register without manual admin
- Reduces deployment friction
- Easier to scale to many devices
- Firmware can embed device ID

---

## Summary

AgroNode is designed as a **modular IoT system** where:

- Devices are independent and self-describing
- Communication is decoupled via MQTT
- Backend handles ingestion, validation, persistence, and trigger evaluation
- Frontend provides real-time + historical visualization
- Device types enable mixed deployment scenarios
- Architecture scales from small greenhouse to enterprise SaaS
```
agronode/#
```

- Parses incoming messages
- Extracts `deviceId` from topic
- Validates payload
- Sends data to service layer
- Evaluates configured sensor triggers for source device
- Publishes activation command to `agronode/{targetDeviceId}/activation`

Trigger routing rule:

- If trigger contains `targetDeviceId`, activation is sent to that device
- If `targetDeviceId` is missing, activation is sent back to source `deviceId`

---

#### B) API + Realtime

Exposes data to frontend:

```txt
GET /api/health
GET /api/data
GET /api/data/:deviceId
GET /api/latest/:deviceId
```

Realtime stream:

```txt
GET /ws/telemetry (WebSocket)
```

---

#### Backend Internal Architecture

```
/internal
  /handlers      -> HTTP layer
  /services      -> business logic
  /repositories  -> database access
  /mqtt          -> MQTT client
  /models        -> data structures
  /database      -> DB connection
  /config        -> environment config
```

---

### 4. Database Layer (PostgreSQL)

Stores all sensor readings.

### Table: sensor_data

```sql
CREATE TABLE sensor_data (
    id SERIAL PRIMARY KEY,
    device_id TEXT NOT NULL,
    temperature FLOAT NOT NULL,
    humidity FLOAT NOT NULL,
  sensors JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW()
);
```

### Responsibilities:

- Persist all incoming telemetry
- Enable historical queries
- Support analytics in future versions

---

### 5. Frontend (Visualization Layer)

Built in React + TypeScript.

Responsibilities:

- Display real-time sensor data
- Show historical charts
- Allow device selection
- Auto-refresh every few seconds

### UI Flow:

```
API → React Service Layer → Components → Dashboard UI
```

---

## Docker Architecture

All services run via Docker Compose:

```
+-------------------+
| Frontend (React)  |
+-------------------+
          ▲
          |
+-------------------+
| Backend (Go API)  |
+-------------------+
          ▲
          |
+-------------------+
| MQTT Broker       |
| (Mosquitto)       |
+-------------------+
          ▲
          |
+-------------------+
| PostgreSQL        |
+-------------------+
```

---

## Scaling Strategy

## Phase 1 (MVP)
- Single broker
- Single backend instance
- Single database

## Phase 2 (Growth)
- Multiple MQTT topics per farm
- Device authentication
- Horizontal backend scaling

## Phase 3 (Production SaaS)
- Multi-tenant system (farms/users)
- Cloud deployment
- Load balancer
- Metrics + monitoring

---

## Key Design Decisions

## Why MQTT?
- Low bandwidth usage
- Real-time communication
- Ideal for unstable rural networks

## Why Go backend?
- High performance
- Concurrency support
- Ideal for MQTT consumers

## Why PostgreSQL?
- Reliable relational storage
- Good for time-series extensions
- Easy analytics integration later

---

## Future Enhancements

- Device authentication layer
- Alerting system (email/SMS)
- Soil moisture sensors
- AI prediction of irrigation needs
- Grafana integration for analytics
- Mobile app (Flutter / React Native)

---

## Summary

AgroNode is designed as a modular IoT system where:

- Devices are independent
- Communication is decoupled via MQTT
- Backend acts as a processing + API layer
- Frontend is purely visualization

This architecture is scalable from a small greenhouse setup to a full agriculture IoT SaaS platform.
