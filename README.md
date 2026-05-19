# AgroNode

AgroNode is an IoT telemetry platform for greenhouse and agriculture monitoring.

It streams sensor data from ESP32 devices through MQTT, processes and stores telemetry in Go + PostgreSQL, and visualizes it in a React dashboard.

## Architecture

ESP32 -> Mosquitto MQTT -> Go Backend -> PostgreSQL -> React Frontend

## Repository Layout

This repository contains the project under `agronode/`.

```txt
agronode/
      firmware/
      backend/
      frontend/
      infra/
      docs/
      docker-compose.yml
```

## Core Features

- MQTT telemetry ingestion (`agronode/{deviceId}/telemetry`)
- REST API for historical and latest readings
- WebSocket stream for live dashboard updates (`/ws/telemetry`)
- PostgreSQL persistence with migrations
- Docker Compose full-stack development environment

## API Summary

Base URL: `http://localhost:8080`

- `GET /api/health`
- `GET /api/data`
- `GET /api/data/:deviceId`
- `GET /api/latest/:deviceId`
- `GET /ws/telemetry` (WebSocket)

Full API details: `agronode/backend/docs/API.md`

## MQTT Contract

Topic:

```txt
agronode/{deviceId}/telemetry
```

Payload shape:

```json
{
      "deviceId": "esp32-lab",
      "timestamp": 1715539200,
      "version": 1,
      "sensors": {
            "temperature": 24.5,
            "humidity": 60,
            "co2": 450
      }
}
```

See full contract in `agronode/MQTT_CONTRACT.md`.

## Quick Start

```bash
cd agronode
docker compose up --build
```

Services:

- Frontend: `http://localhost:5173`
- Backend API: `http://localhost:8080`
- MQTT: `localhost:1883`
- PostgreSQL: `localhost:5432`

## Environment (Backend)

```env
APP_PORT=8080
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=agronode
MQTT_BROKER=tcp://mosquitto:1883
MQTT_TOPIC=agronode/#
```
