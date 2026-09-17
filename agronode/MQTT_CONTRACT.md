# MQTT Contract - AgroNode

## Topic Structure

Devices publish telemetry and status to two topics:

### Telemetry Topic (Device → Backend)
```txt
agronode/{deviceId}/telemetry
```

Example:
```txt
agronode/esp32-lab/telemetry
```

### Status Topic (Device → Backend)
```txt
agronode/{deviceId}/status
```

Example:
```txt
agronode/esp32-lab/status
```

### Registration Topic (Device → Backend)
```txt
agronode/{deviceId}/register
```

Example:
```txt
agronode/esp32-lab/register
```

---

## Telemetry Payload Format

Devices MUST send per-sensor telemetry in this format:

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

### Key Fields:
- `deviceId` (required): unique device identifier
- `sensorId` (optional): which sensor sent this reading (e.g., `dht11-temp`, `dht11-humidity`, `co2`, `signal_strength`)
- `timestamp` (optional): Unix timestamp in seconds
- `sensors` (required): map of sensor names to numeric values

### Custom Sensor IDs

Backends must NOT assume fixed sensor types. Examples:
```json
{
  "deviceId": "esp32-lab",
  "sensorId": "dht11-temp",
  "sensors": {
    "dht11-temp": 25.0,
    "dht11-humidity": 55.0,
    "signal_strength": -65,
    "battery": 87.5
  }
}
```

---

## Status Payload Format

Devices MAY publish device status separately:

```json
{
  "deviceId": "esp32-lab",
  "online": true,
  "signal_strength": -65,
  "uptime": 3600,
  "firmware": "v1.2.3"
}
```

### Key Fields:
- `signal_strength` (optional int): RSSI in dBm
- `uptime` (optional int): seconds since boot

---

## Registration Payload Format

New devices can self-register via MQTT:

```json
{
  "deviceId": "esp32-lab",
  "firmware": "v1.2.3",
  "metadata": {
    "signalStrength": -61,
    "hardware": {
      "model": "ESP32",
      "board": "devkit"
    }
  },
  "tags": ["greenhouse", "live"]
}
```

Backend behavior:
- Creates device if not exists
- Updates device if exists (idempotent)
- Defaults new MQTT-registered devices to type `publisher`
- Stores metadata and tags

---

## Activation Topic (Backend → Device)

When a trigger is activated, backend publishes activation commands:

```txt
agronode/{targetDeviceId}/activation
```

`targetDeviceId` defaults to source telemetry device if not explicitly configured on trigger.

Payload format:

```json
{
  "deviceId": "esp32-lab",
  "trigger": "above_max",
  "sensor": "dht11-temp",
  "limitType": "max",
  "value": 28.5,
  "threshold": 30.0,
  "activated": true,
  "timestamp": 1715539200
}
```

ESP32 behavior:

- Device subscribes to `agronode/{deviceId}/activation`
- When payload contains `"activated": true`, device sets activation output pin HIGH
- Activation pin is auto-reset to LOW after 5 seconds (firmware default)
- Device logs activation event with sensor name and threshold

---

## Payload Format (Legacy)

Older devices MAY still use legacy format with root-level numeric fields:

```json
{
  "deviceId": "device-1",
  "temperature": 24.5,
  "humidity": 60
}
```

Backend behavior:
- Accepts both new and legacy formats
- Extracts sensor values from `sensors` map if present
- Falls back to legacy fields (`temperature`, `humidity`) if `sensors` is empty
- Infers single-field sensor ID when map has one entry

---

## Versioning

Future updates must include:

```json
{
  "version": 1
}
```

So backend can support multiple message formats simultaneously.

---

## Backward Compatibility Rules

- Backend must ignore unknown fields
- Backend must not break if new sensor types appear
- Devices MAY omit optional fields
- Devices SHOULD include `sensorId` for clarity, but backend infers from data if missing
