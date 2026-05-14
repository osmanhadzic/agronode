# AgroNode Architecture

## Overview

AgroNode is a scalable IoT platform designed for greenhouse and agriculture monitoring.

It collects real-time sensor data from ESP32 devices, transports it via MQTT, processes it in a backend service, stores it in a database, and exposes it to a frontend dashboard.

---

# System Architecture

## High-Level Flow

```
ESP32 Devices
    │
    │  (MQTT Publish)
    ▼
MQTT Broker (Mosquitto)
    │
    │  (Subscribe)
    ▼
Go Backend (MQTT Consumer + API)
    │
    ├── PostgreSQL (Persistent Storage)
    │
    └── REST API (Data Access Layer)
          │
          ▼
   React Frontend Dashboard
```

---

# Data Flow Explanation

## 1. ESP32 Devices (Edge Layer)

Each ESP32 device is responsible for:

- Reading sensor data (temperature, humidity, etc.)
- Connecting to WiFi
- Publishing data to MQTT broker
- Using topic format:

```
agronode/{deviceId}
```

Example:

```
agronode/device-1
```

Payload:

```json
{
  "temperature": 24.5,
  "humidity": 60
}
```

---

## 2. MQTT Broker (Message Layer)

We use:

- Eclipse Mosquitto

Role:

- Receives messages from ESP32 devices
- Routes messages to backend subscribers
- Decouples devices from backend

Why MQTT:

- Lightweight
- Reliable for unstable networks
- Ideal for IoT scale systems

---

## 3. Backend (Processing Layer)

The backend is written in Go.

It has two responsibilities:

### A) MQTT Consumer
- Subscribes to:
```
agronode/#
```

- Parses incoming messages
- Extracts `deviceId` from topic
- Validates payload
- Sends data to service layer

---

### B) REST API

Exposes data to frontend:

```
GET /api/data
GET /api/data/:deviceId
GET /api/latest/:deviceId
```

---

### Backend Internal Architecture

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

## 4. Database Layer (PostgreSQL)

Stores all sensor readings.

### Table: sensor_data

```sql
CREATE TABLE sensor_data (
    id SERIAL PRIMARY KEY,
    device_id TEXT NOT NULL,
    temperature FLOAT NOT NULL,
    humidity FLOAT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
```

### Responsibilities:

- Persist all incoming telemetry
- Enable historical queries
- Support analytics in future versions

---

## 5. Frontend (Visualization Layer)

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

# Docker Architecture

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

# Scaling Strategy

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

# Key Design Decisions

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

# Future Enhancements

- Device authentication layer
- Alerting system (email/SMS)
- Soil moisture sensors
- AI prediction of irrigation needs
- Grafana integration for analytics
- Mobile app (Flutter / React Native)

---

# Summary

AgroNode is designed as a modular IoT system where:

- Devices are independent
- Communication is decoupled via MQTT
- Backend acts as a processing + API layer
- Frontend is purely visualization

This architecture is scalable from a small greenhouse setup to a full agriculture IoT SaaS platform.
