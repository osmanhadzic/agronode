# AgroNode

AgroNode is a production-style IoT platform for greenhouse and agriculture monitoring.

The platform collects real-time sensor data from ESP32 devices using MQTT and displays telemetry through a modern dashboard.

Architecture:

ESP32 -> MQTT Broker -> Go Backend -> PostgreSQL -> React Frontend

---

# Features

- Real-time sensor monitoring
- MQTT communication
- Temperature and humidity tracking
- Multi-device support
- REST API
- PostgreSQL storage
- Dockerized infrastructure
- Responsive dashboard
- Scalable architecture

---

# Tech Stack

## Firmware
- ESP32
- Arduino Framework
- PubSubClient
- DHT22

## Backend
- Go
- Gin
- GORM
- PostgreSQL
- Eclipse Paho MQTT

## Frontend
- React
- TypeScript
- Vite
- Recharts

## Infrastructure
- Docker
- Docker Compose
- Eclipse Mosquitto

---

# Project Structure

```txt
/agronode
  /firmware
  /backend
  /frontend
  /.github
  docker-compose.yml
```

---

# Architecture

```txt
ESP32 Devices
      ↓
MQTT Broker (Mosquitto)
      ↓
Go Backend Subscriber
      ↓
PostgreSQL Database
      ↓
React Dashboard
```

---

# MQTT Topics

Topic format:

```txt
weather/{deviceId}
```

Example:

```txt
weather/device-1
```

Payload example:

```json
{
  "temperature": 24.5,
  "humidity": 60
}
```

---

# Backend API

## Get all weather data

```http
GET /api/weather
```

## Get weather data by device

```http
GET /api/weather/:deviceId
```

---

# Getting Started

## Clone repository

```bash
git clone https://github.com/YOUR_USERNAME/agronode.git
cd agronode
```

---

# Run with Docker

```bash
docker compose up --build
```

---

# Services

| Service | Port |
|---|---|
| Frontend | 5173 |
| Backend API | 8080 |
| MQTT Broker | 1883 |
| PostgreSQL | 5432 |

---

# Environment Variables

Example backend `.env`:

```env
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=agronode

MQTT_BROKER=tcp://mosquitto:1883
MQTT_TOPIC=weather/#
```

---

# Future Improvements

- Soil moisture monitoring
- CO2 sensors
- Alert system
- Mobile application
- AI analytics
- Irrigation automation
- Cloud deployment
- Device authentication

---

# Goals

AgroNode aims to become a scalable smart agriculture platform suitable for:
- greenhouses
- farms
- indoor growing systems
- industrial agriculture monitoring

---

# License

MIT
