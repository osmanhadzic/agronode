# AgroNode API Documentation

Base URL: `http://localhost:8080`

Organization scoping: All endpoints require `X-Organization-ID` header for multi-tenant support.

---

## Health

### GET /api/health

Response:

```json
{
  "status": "ok"
}
```

---

## Devices

### GET /api/devices

Returns a paginated list of devices.

Query params:

- `page` (optional, default `1`)
- `limit` (optional, default `20`, max `100`)
- `status` (optional: `unknown`, `online`, `offline`)
- `search` (optional, searches `deviceId` and `firmwareVersion`)
- `tags` (optional, comma-separated tag filter)

Example:

```
GET /api/devices?page=1&limit=10&status=online&search=lab&tags=greenhouse,live
```

Response: `200 OK`

```json
[
  {
    "id": 1,
    "deviceId": "esp32-lab",
    "deviceType": "publisher",
    "status": "online",
    "firmwareVersion": "v1.2.3",
    "metadata": {
      "battery": 87.5,
      "signalStrength": -61,
      "hardware": {
        "model": "ESP32",
        "rev": "1"
      }
    },
    "tags": ["greenhouse", "live"],
    "lastSeen": "2026-06-01T12:00:00Z",
    "createdAt": "2026-06-01T11:55:00Z",
    "updatedAt": "2026-06-01T12:00:00Z"
  }
]
```

Possible errors:

- `400` for invalid `page`, `limit`, `status`, or `search`
- `500` for server errors

---

### GET /api/devices/:deviceId

Returns device details.

Example:

```
GET /api/devices/esp32-lab
```

Response: `200 OK`

```json
{
  "id": 1,
  "deviceId": "esp32-lab",
  "deviceType": "publisher",
  "status": "online",
  "firmwareVersion": "v1.2.3",
  "metadata": {
    "battery": 87.5,
    "signalStrength": -61,
    "hardware": {
      "model": "ESP32",
      "rev": "1"
    }
  },
  "tags": ["greenhouse"],
  "lastSeen": "2026-06-01T12:00:00Z",
  "createdAt": "2026-06-01T11:55:00Z",
  "updatedAt": "2026-06-01T12:00:00Z"
}
```

Possible errors:

- `404` if device not found
- `500` for server errors

---

### POST /api/devices/register

Registers a new device or updates an existing one (idempotent).

Request body:

```json
{
  "deviceId": "esp32-lab",
  "deviceType": "publisher",
  "firmwareVersion": "v1.2.3",
  "metadata": {
    "battery": 87.5,
    "signalStrength": -61,
    "hardware": {
      "model": "ESP32",
      "board": "devkit"
    }
  },
  "apiKey": "optional-api-key",
  "provisioningToken": "optional-provisioning-token",
  "tags": ["greenhouse", "live"]
}
```

Response: `200 OK`

```json
{
  "id": 1,
  "deviceId": "esp32-lab",
  "deviceType": "publisher",
  "status": "unknown",
  "firmwareVersion": "v1.2.3",
  "metadata": {
    "battery": 87.5,
    "signalStrength": -61,
    "hardware": {
      "model": "ESP32",
      "board": "devkit"
    }
  },
  "tags": ["greenhouse", "live"],
  "createdAt": "2026-06-01T11:55:00Z",
  "updatedAt": "2026-06-01T11:55:00Z"
}
```

Notes:

- `deviceType` is optional (defaults to `publisher`)
- Only non-empty fields update existing device
- Device status starts as `unknown` until first telemetry

Possible errors:

- `400` for invalid device ID or metadata
- `500` for server errors

---

### GET /api/devices/:deviceId/sensors

Lists all sensors for a device.

Example:

```
GET /api/devices/esp32-lab/sensors
```

Response: `200 OK`

```json
[
  {
    "id": 1,
    "deviceId": "esp32-lab",
    "sensorId": "dht11-temp",
    "createdAt": "2026-06-01T11:55:00Z",
    "updatedAt": "2026-06-01T12:00:00Z"
  },
  {
    "id": 2,
    "deviceId": "esp32-lab",
    "sensorId": "dht11-humidity",
    "createdAt": "2026-06-01T11:55:00Z",
    "updatedAt": "2026-06-01T12:00:00Z"
  }
]
```

---

## Telemetry

### GET /api/data

Returns all telemetry readings (paginated).

Query params:

- `page` (optional)
- `limit` (optional)
- `deviceId` (optional, filter by device)
- `sensorId` (optional, filter by sensor)

Example:

```
GET /api/data?deviceId=esp32-lab&sensorId=dht11-temp
```

Response: `200 OK`

```json
[
  {
    "deviceId": "esp32-lab",
    "sensorId": "dht11-temp",
    "temperature": 24.5,
    "humidity": 60,
    "sensors": {
      "dht11-temp": 24.5,
      "dht11-humidity": 60
    },
    "createdAt": "2026-05-14T12:00:00Z"
  }
]
```

---

### GET /api/data/:deviceId

Returns telemetry history for one device.

Examples:

```
GET /api/data/esp32-lab
GET /api/data/Plastenik-1
GET /api/data/device-1?limit=100
```

Query params:

- `limit` (optional, default 100)
- `startDate` (optional, ISO 8601)
- `endDate` (optional, ISO 8601)

Response: `200 OK`

```json
[
  {
    "deviceId": "esp32-lab",
    "sensorId": "dht11-temp",
    "sensors": {
      "dht11-temp": 24.5,
      "dht11-humidity": 60
    },
    "createdAt": "2026-05-14T12:00:00Z"
  }
]
```

Sorted by descending `createdAt`.

Possible errors:

- `400` for invalid `deviceId`
- `500` for server errors

---

### GET /api/data/:deviceId/:sensorId

Returns telemetry history for a specific sensor on a device.

Example:

```
GET /api/data/esp32-lab/dht11-temp
GET /api/data/esp32-lab/dht11-temp?limit=500&startDate=2026-06-01T00:00:00Z
```

Query params:

- `limit` (optional)
- `startDate` (optional, ISO 8601)
- `endDate` (optional, ISO 8601)

Response: `200 OK`

```json
[
  {
    "deviceId": "esp32-lab",
    "sensorId": "dht11-temp",
    "sensors": {
      "dht11-temp": 24.5
    },
    "createdAt": "2026-05-14T12:00:00Z"
  }
]
```

---

### GET /api/latest/:deviceId

Returns latest telemetry record for one device (all sensors).

Example:

```
GET /api/latest/esp32-lab
```

Response: `200 OK`

```json
{
  "deviceId": "esp32-lab",
  "sensorId": "dht11-temp",
  "sensors": {
    "dht11-temp": 24.5,
    "dht11-humidity": 60
  },
  "createdAt": "2026-06-01T12:00:00Z"
}
```

Possible errors:

- `404` if device has no readings yet
- `500` for server errors

---

## Triggers

### GET /api/triggers/:deviceId

Lists all triggers for a device.

Example:

```
GET /api/triggers/esp32-lab
```

Response: `200 OK`

```json
{
  "deviceId": "esp32-lab",
  "triggers": [
    {
      "sensorId": "dht11-temp",
      "min": 15.0,
      "max": 30.0,
      "targetDeviceId": "pump-node-1"
    },
    {
      "sensorId": "dht11-humidity",
      "min": 30.0,
      "targetDeviceId": "esp32-lab"
    }
  ]
}
```

---

### GET /api/triggers/:deviceId/:sensorId

Returns specific trigger for a sensor.

Example:

```
GET /api/triggers/esp32-lab/dht11-temp
```

Response: `200 OK`

```json
{
  "sensorId": "dht11-temp",
  "min": 15.0,
  "max": 30.0,
  "targetDeviceId": "pump-node-1"
}
```

Possible errors:

- `404` if trigger not found
- `500` for server errors

---

### POST /api/triggers/:deviceId/:sensorId

Creates or updates a trigger.

Request body:

```json
{
  "min": 15.0,
  "max": 30.0,
  "targetDeviceId": "pump-node-1"
}
```

Response: `200 OK`

```json
{
  "sensorId": "dht11-temp",
  "min": 15.0,
  "max": 30.0,
  "targetDeviceId": "pump-node-1"
}
```

Notes:

- At least one of `min` or `max` is required
- `targetDeviceId` is optional (defaults to source device)
- Activation command is published to target device when threshold crossed

Possible errors:

- `400` if validation fails (invalid thresholds, etc.)
- `500` for server errors

---

### DELETE /api/triggers/:deviceId/:sensorId

Removes a trigger.

Example:

```
DELETE /api/triggers/esp32-lab/dht11-temp
```

Response: `204 No Content`

Possible errors:

- `404` if trigger not found
- `500` for server errors

---

## WebSocket

### GET /ws/telemetry

Establishes WebSocket connection for real-time telemetry and status events.

Example:

```
ws://localhost:8080/ws/telemetry
```

Messages sent from server:

**Telemetry update:**

```json
{
  "type": "telemetry",
  "data": {
    "deviceId": "esp32-lab",
    "sensorId": "dht11-temp",
    "sensors": {
      "dht11-temp": 24.5,
      "dht11-humidity": 60
    },
    "createdAt": "2026-06-01T12:00:00Z"
  }
}
```

**Device status event:**

```json
{
  "type": "status",
  "data": {
    "deviceId": "esp32-lab",
    "oldStatus": "unknown",
    "newStatus": "online",
    "eventType": "device.online",
    "timestamp": "2026-06-01T12:00:00Z"
  }
}
```

**Trigger activation event:**

```json
{
  "type": "trigger",
  "data": {
    "deviceId": "esp32-lab",
    "trigger": "above_max",
    "sensor": "dht11-temp",
    "limitType": "max",
    "value": 35.5,
    "threshold": 30.0,
    "activated": true,
    "timestamp": "2026-06-01T12:00:00Z"
  }
}
```

---

## Headers

All requests should include:

```
X-Organization-ID: <organization_id>
Content-Type: application/json
```

---

## Error Responses

Standard error response format:

```json
{
  "error": "error message"
}
```

Status codes:

- `200` OK
- `204` No Content
- `400` Bad Request (validation error)
- `401` Unauthorized (missing auth)
- `404` Not Found
- `500` Internal Server Error
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

## Triggers

### GET /api/triggers/:deviceId

Returns all configured triggers for a device.

Response:

```json
{
  "deviceId": "esp32-lab",
  "triggers": [
    {
      "sensor": "temperature",
      "min": 18,
      "max": 30,
      "targetDeviceId": "pump-node-1"
    },
    {
      "sensor": "humidity",
      "min": 40,
      "max": 80
    }
  ]
}
```

If no triggers are configured, `triggers` is an empty array.

Possible errors:

- `400` for invalid `deviceId`
- `500` for server errors

### PUT /api/triggers/:deviceId/:sensor

Sets min and/or max trigger for one sensor on one device.

Request body:

```json
{
  "min": 18,
  "max": 30,
  "targetDeviceId": "pump-node-1"
}
```

At least one of `min` or `max` is required.

`targetDeviceId` is optional. If omitted, activation is sent to the same `deviceId` from the route.

Response:

```json
{
  "deviceId": "esp32-lab",
  "sensor": "humidity",
  "min": 40,
  "max": 80,
  "targetDeviceId": "pump-node-1"
}
```

Possible errors:

- `400` for invalid thresholds or `deviceId`
- `500` for server errors

### GET /api/triggers/:deviceId/:sensor

Returns configured min/max trigger for one device sensor.

Response:

```json
{
  "deviceId": "esp32-lab",
  "sensor": "humidity",
  "min": 40,
  "max": 80,
  "targetDeviceId": "pump-node-1"
}
```

Possible errors:

- `400` for invalid `deviceId`
- `500` for server errors

If no trigger is configured yet, response is still `200` with only `deviceId` and `sensor`.

### DELETE /api/triggers/:deviceId/:sensor

Deletes a configured trigger for one device sensor.

Response:

- `204 No Content` when trigger is deleted

Possible errors:

- `400` for invalid `deviceId` or `sensor`
- `404` when trigger does not exist
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
