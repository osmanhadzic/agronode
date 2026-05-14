# MQTT Contract - AgroNode

## Topic Structure

agronode/{deviceId}/telemetry

Example:
agronode/device-1/telemetry

---

## Payload Format

All devices MUST send data in this format:

{
  "deviceId": "string",
  "timestamp": "unix_epoch",
  "sensors": {
    "sensor_name": "value"
  }
}

---

## Sensor Rules

- Sensors are dynamic key-value pairs
- Backend must NOT assume fixed sensor types
- New sensors can be added without backend changes

Example:

{
  "temperature": 24.5,
  "humidity": 60,
  "co2": 450,
  "soil_moisture": 33
}

---

## Optional Metadata

Devices MAY include:

{
  "battery": 87,
  "signal_strength": -70
}

---

## Versioning (IMPORTANT)

Future updates must include:

{
  "version": 1
}

So backend can support multiple formats.

---

## Backward Compatibility Rule

- Backend must ignore unknown fields
- Backend must not break if new sensors appear
