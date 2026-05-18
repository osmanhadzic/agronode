# Demo Device Publisher

A virtual MQTT device is available for testing the full pipeline without physical ESP32 hardware.

Published sensors include:

- `temperature`
- `humidity`
- `co2`
- `soil_moisture`
- `battery`
- `signal_strength`

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
