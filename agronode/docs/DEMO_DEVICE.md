# Demo Device Publisher

A virtual MQTT device is available for testing the full pipeline without physical ESP32 hardware. It simulates a multi-sensor IoT device with realistic telemetry patterns.

Published telemetry is per-sensor (one message per sensor):

- `dht11-temp`
- `dht11-humidity`

Published status metadata goes to dedicated topic:

- Topic: `agronode/{deviceId}/status`
- Fields: `online`, `signal_strength`

The demo device publishes readings for these sensors:

| Sensor ID | Description | Type | Range | Unit |
|-----------|-------------|------|-------|------|
| `dht11-temp` | Temperature | float | 15–35 | °C |
| `dht11-humidity` | Humidity | float | 20–95 | % |
| `co2` | CO₂ level | float | 300–1000 | ppm |
| `soil_moisture` | Soil moisture | float | 0–100 | % |
| `battery` | Battery level | float | 0–100 | % |
| `signal_strength` | WiFi RSSI | float | -120–-30 | dBm |

---

## Device Type and Registration

The demo device registers as a **publisher** device type (default via MQTT `/register` topic).

- **Device ID**: `demo-device-1` (configurable)
- **Device Type**: `publisher` (sends telemetry)
- **Status Topic**: `agronode/demo-device-1/status` (signal strength, uptime)
- **Telemetry Topic**: `agronode/demo-device-1/telemetry` (sensor readings)

---

## Start Demo Device

```bash
docker compose --profile demo up -d demo-device
```

This spawns a container running `infra/demo-device/publisher.sh` that:
1. Registers device via MQTT `/register` topic
2. Publishes sensor readings every `DEMO_PUBLISH_INTERVAL_SECONDS`
3. Sends status updates (signal strength, uptime metadata)

---

## Stop Demo Device

```bash
docker compose --profile demo stop demo-device
```

Device status changes to **offline** after no readings for 30 seconds. Triggers configured on the device will stop firing.

---

## Configuration

Set environment variables before starting to customize behavior:

```bash
# Device identity
export DEMO_DEVICE_ID=demo-device-1

# Publishing interval (seconds)
export DEMO_PUBLISH_INTERVAL_SECONDS=15

# MQTT broker location
export DEMO_MQTT_HOST=mosquitto
export DEMO_MQTT_PORT=1883

# Docker Compose profile
docker compose --profile demo up -d demo-device
```

---

## Telemetry Format

Published to `agronode/{DEMO_DEVICE_ID}/telemetry`:

```json
{
  "deviceId": "demo-device-1",
  "sensorId": "dht11-temp",
  "timestamp": 1715539200,
  "sensors": {
    "dht11-temp": 24.5,
    "dht11-humidity": 62.0,
    "co2": 520,
    "soil_moisture": 45.2,
    "battery": 87.5,
    "signal_strength": -65.0
  }
}
```

---

## Verify Data Flow

### Check Device Registration

```bash
curl http://localhost:8080/api/devices
```

Response includes demo-device with `deviceType: "publisher"`.

### Get Latest Readings

```bash
curl http://localhost:8080/api/latest/demo-device-1
```

Returns recent SensorData records with all sensor values.

### Stream Live Data

Open WebSocket connection:
```bash
websocat ws://localhost:8080/ws/telemetry
```

Will display telemetry messages from all devices including the demo device.

### Check MQTT Directly (Optional)

```bash
docker exec -it mosquitto mosquitto_sub -h localhost -t 'agronode/demo-device-1/#'
```

Should show telemetry and status messages every 15 seconds.

---

## Testing Triggers

1. **Create a min/max trigger** in the UI:
   - Device: `demo-device-1`
   - Sensor: `dht11-temp`
   - Min: 20°C, Max: 30°C
   - Target: any receiver device (or self)

2. **Watch activation** as temperature fluctuates in published sensor data.

3. **Monitor backend logs** for trigger evaluation:
   ```bash
   docker logs agronode-backend-1 | grep -i trigger
   ```

---

## Simulated Data Patterns

The demo device uses synthetic data with realistic variance:

- **Temperature**: sine wave 20–30°C with ±1°C random noise
- **Humidity**: inverse correlation with temperature, ±2% noise
- **CO₂**: baseline ~500 ppm + ±50 random drift
- **Soil Moisture**: random walk 30–70%
- **Battery**: slow decline (drain simulation)
- **Signal Strength**: random -50 to -75 dBm

This allows testing of:
- Trigger activation/recovery cycles
- Data smoothing and filtering
- Real-time charting
- Historical data retention

---

## Troubleshooting

**Demo device not appearing in UI:**
- Check backend logs: `docker logs agronode-backend-1 | grep register`
- Verify MQTT connectivity: `docker logs demo-device` for connection errors
- Ensure PostgreSQL is running: `docker logs agronode-postgres-1`

**No telemetry data:**
- Confirm publish interval: check `docker logs demo-device`
- Check WebSocket connection: browser DevTools → Network tab → WS
- Verify database: `docker exec agronode-postgres-1 psql -U postgres -d agronode -c "SELECT COUNT(*) FROM sensor_data WHERE device_id='demo-device-1'"`

**Triggers not firing:**
- Ensure trigger target device exists and is type `receiver`
- Check backend logs for evaluation errors
- Verify device is still online (refresh dashboard)

---

## Integration Test Scenario

Demo device is used in **AGN-17 Integration Test** to validate:
1. Device registration and discovery
2. Per-sensor telemetry ingestion
3. Trigger evaluation and activation
4. WebSocket real-time broadcast
5. Historical query and charting

See [AGN-17_INTEGRATION_TEST.md](AGN-17_INTEGRATION_TEST.md) for full test suite.
