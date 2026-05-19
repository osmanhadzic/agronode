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
