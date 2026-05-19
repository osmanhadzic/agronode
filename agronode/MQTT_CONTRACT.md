# MQTT Contract - AgroNode

## Topic Structure

```txt
agronode/{deviceId}/telemetry
```

Example:

```txt
agronode/device-1/telemetry
```

---

## Payload Format

All devices MUST send data in this format:

```json
{
  "deviceId": "string",
  "timestamp": 1715539200,
  "version": 1,
  "sensors": {
    "sensor_name": 0
  }
}
```

---

## Sensor Rules

- Sensors are dynamic key-value pairs
- Backend must NOT assume fixed sensor types
- New sensors can be added without backend changes

Example:

```json
{
  "temperature": 24.5,
  "humidity": 60,
  "co2": 450,
  "soil_moisture": 33
}
```

---

## Optional Metadata

Devices MAY include:

```json
{
  "battery": 87,
  "signal_strength": -70
}
```

---

## Versioning (IMPORTANT)

Future updates must include:

```json
{
  "version": 1
}
```

So backend can support multiple formats.

---

## Backward Compatibility Rule

- Backend must ignore unknown fields
- Backend must not break if new sensors appear
