# AgroNode API Documentation

Base URL: `http://localhost:8080`

## Health

### GET /api/health

Response:

```json
{
  "status": "ok"
}
```

## Devices

### GET /api/devices

Returns a paginated list of devices.

Query params:

- `page` (optional, default `1`)
- `limit` (optional, default `20`, max `100`)
- `status` (optional: `unknown`, `online`, `offline`)
- `search` (optional, searches `deviceId` and `firmwareVersion`)

Example:

- `GET /api/devices?page=1&limit=10&status=online&search=lab`

Response:

```json
[
  {
    "id": 1,
    "deviceId": "esp32-lab",
    "status": "online",
    "firmwareVersion": "v1.2.3",
    "lastSeen": "2026-06-01T12:00:00Z",
    "createdAt": "2026-06-01T11:55:00Z",
    "updatedAt": "2026-06-01T12:00:00Z"
  }
]
```

Possible errors:

- `400` for invalid `page`, `limit`, `status`, or `search`
- `500` for server errors

### POST /api/devices/register

Registers or updates a device record.

Request body:

```json
{
  "deviceId": "esp32-lab",
  "firmwareVersion": "v1.2.3",
  "metadata": {
    "battery": 87.5,
    "signalStrength": -61,
    "hardware": {
      "model": "ESP32",
      "board": "devkit"
    }
  }
}
```

Possible errors:

- `400` for invalid device ID or metadata
- `500` for server errors

## Telemetry

### GET /api/data

Returns all telemetry readings.

Response:

```json
[
  {
    "deviceId": "esp32-lab",
    "temperature": 24.5,
    "humidity": 60,
    "sensors": {
      "temperature": 24.5,
      "humidity": 60,
      "co2": 450,
      "soil_moisture": 33
    },
    "createdAt": "2026-05-14T12:00:00Z"
  }
]
```

### GET /api/data/:deviceId

Returns telemetry history for one device.

Examples:

- `GET /api/data/esp32-lab`
- `GET /api/data/Plastenik-1`

Response:

```json
[
  {
    "deviceId": "esp32-lab",
    "temperature": 24.5,
    "humidity": 60,
    "sensors": {
      "temperature": 24.5,
      "humidity": 60
    },
    "createdAt": "2026-05-14T12:00:00Z"
  }
]
```

Possible errors:

- `400` for invalid `deviceId`
- `500` for server errors

### GET /api/latest/:deviceId

Returns latest telemetry record for one device.

Example:

- `GET /api/latest/esp32-lab`

Response:

```json
{
  "deviceId": "esp32-lab",
  "temperature": 24.5,
  "humidity": 60,
  "sensors": {
    "temperature": 24.5,
    "humidity": 60,
    "battery": 87
  },
  "createdAt": "2026-05-14T12:00:00Z"
}
```

Possible errors:

- `400` for invalid `deviceId`
- `404` when no telemetry exists for `deviceId`
- `500` for server errors

## Realtime Stream

### GET /ws/telemetry (WebSocket)

Streams telemetry readings as JSON messages.

Message shape:

```json
{
  "deviceId": "esp32-lab",
  "temperature": 24.5,
  "humidity": 60,
  "sensors": {
    "temperature": 24.5,
    "humidity": 60,
    "co2": 450
  },
  "createdAt": "2026-05-14T12:00:00Z"
}
```
