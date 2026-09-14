# Demo Device Publisher

A virtual MQTT device is available for testing the full pipeline without physical ESP32 hardware.

Published telemetry is per-sensor (one message per sensor):

- `dht11-temp`
- `dht11-humidity`

Published status metadata goes to dedicated topic:

- Topic: `agronode/{deviceId}/status`
- Fields: `online`, `signal_strength`

## Start demo device

```bash
docker compose --profile demo up -d demo-device
```

## Stop demo device

```bash
docker compose stop demo-device
```

## Optional configuration

Set values in your shell before starting:

- `DEMO_DEVICE_ID` (default: `demo-device-1`)
- `DEMO_PUBLISH_INTERVAL_SECONDS` (default: `15`)
- `DEMO_MQTT_HOST` (default: `mosquitto`)
- `DEMO_MQTT_PORT` (default: `1883`)

## Verify data flow

```bash
wget -qO- http://localhost:8080/api/latest/demo-device-1
```
