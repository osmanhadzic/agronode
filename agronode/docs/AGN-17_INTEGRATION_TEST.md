# AGN-17 Integration Test Report

**Objective**: Validate end-to-end data flow from MQTT device publication through backend processing, storage, and frontend consumption.

---

## Prerequisites

- Docker Compose installed
- Workspace at `/home/osman/Documents/ESP32/agronode`
- Ports 5173 (frontend), 8080 (backend), 1883 (MQTT) available

---

## Test Setup

**Startup**:
```bash
cd agronode
docker compose down
docker compose --profile demo up -d
```

Services running:
- `mosquitto` - MQTT broker
- `postgres` - PostgreSQL database
- `backend` - Go API server (port 8080)
- `frontend` - React dev server (port 5173)
- `demo-device` - Simulated publisher (generates telemetry)

---

## Test Scenarios

### Scenario 1: Per-Sensor Telemetry Ingestion

**Steps**:
1. Demo device connects to MQTT broker
2. Demo device publishes to `agronode/demo-device-1/telemetry`:
   ```json
   {
     "deviceId": "demo-device-1",
     "sensorId": "dht11-temp",
     "timestamp": 1715539200,
     "sensors": {
       "dht11-temp": 24.5,
       "dht11-humidity": 62.0,
       "co2": 520,
       "soil_moisture": 45.2
     }
   }
   ```

**Verification**:
```bash
# Check backend logs for ingestion
docker logs agronode-backend-1 | grep -i "telemetry\|sensor"

# Query database for stored records
docker exec agronode-postgres-1 psql -U postgres -d agronode -c \
  "SELECT device_id, sensor_id, sensors FROM sensor_data WHERE device_id='demo-device-1' LIMIT 3;"

# Expected output: rows with populated `sensors` JSONB column
```

**Expected Result**: ✅ PASS
- Backend logs show telemetry processing
- PostgreSQL contains rows for `demo-device-1`
- `sensors` column contains `{"dht11-temp": 24.5, ...}`

---

### Scenario 2: Device Registration and Discovery

**Steps**:
1. Query device endpoint via HTTP

**Verification**:
```bash
curl http://localhost:8080/api/devices
```

**Expected Response**:
```json
[
  {
    "id": "demo-device-1",
    "name": "demo-device-1",
    "deviceType": "publisher",
    "status": "online",
    "lastSeen": "2025-01-15T10:30:00Z",
    "discoveredSensors": ["dht11-temp", "dht11-humidity", "co2", "soil_moisture"]
  }
]
```

**Expected Result**: ✅ PASS
- Device appears in list with `deviceType: "publisher"`
- `status` is `"online"`
- `discoveredSensors` array populated from telemetry

---

### Scenario 3: Device Type Classification

**Steps**:
1. Register new device with explicit `deviceType`

**Verification**:
```bash
curl -X POST http://localhost:8080/api/devices/register \
  -H "Content-Type: application/json" \
  -d '{
    "deviceId": "receiver-device-1",
    "name": "Receiver Device",
    "deviceType": "receiver"
  }'

# Verify device type in response
curl http://localhost:8080/api/devices/receiver-device-1 | jq '.deviceType'
```

**Expected Result**: ✅ PASS
- Device registered with correct `deviceType`
- Subsequent queries return `"receiver"` type

---

### Scenario 4: Trigger Configuration and Evaluation

**Steps**:
1. Create trigger for temperature sensor on demo device
2. Verify trigger fires when threshold is crossed

**Verification**:
```bash
# Set trigger: alert if temp > 25°C
curl -X POST http://localhost:8080/api/triggers/demo-device-1/dht11-temp \
  -H "Content-Type: application/json" \
  -d '{
    "sensorId": "dht11-temp",
    "minThreshold": 20.0,
    "maxThreshold": 25.0,
    "targetDeviceId": "receiver-device-1"
  }'

# Check trigger list
curl http://localhost:8080/api/triggers/demo-device-1 | jq '.[].sensorId'

# Monitor backend logs for trigger activation
docker logs -f agronode-backend-1 | grep -i "trigger"
```

**Expected Result**: ✅ PASS
- Trigger created and stored
- Appears in trigger list
- Backend logs show evaluation: "trigger evaluation: sensor_id=dht11-temp, value=24.5, active=false"
- When demo device publishes value > 25°C, logs show: "publishing activation command to receiver-device-1"

---

### Scenario 5: Real-Time WebSocket Broadcasting

**Steps**:
1. Connect WebSocket client
2. Observe telemetry messages in real-time

**Verification**:
```bash
# Using websocat (install: cargo install websocat)
websocat ws://localhost:8080/ws/telemetry

# Expected stream (one message per 15 seconds):
{
  "type": "telemetry",
  "payload": {
    "deviceId": "demo-device-1",
    "sensorId": "dht11-temp",
    "timestamp": 1715539215,
    "sensors": {"dht11-temp": 25.1, "dht11-humidity": 61.5, ...}
  }
}
```

**Expected Result**: ✅ PASS
- WebSocket connection established
- Telemetry messages arrive regularly
- Message format matches schema

---

### Scenario 6: Historical Data Query by Date Range

**Steps**:
1. Query telemetry with time filters

**Verification**:
```bash
# Query last hour of data
curl "http://localhost:8080/api/telemetry/demo-device-1?from=$(date -u -d '1 hour ago' +%s)000&to=$(date -u +%s)000" | jq '.[] | {timestamp, sensorId, value: .sensors.\"dht11-temp\"}'
```

**Expected Result**: ✅ PASS
- Returns array of SensorData records
- Records within requested time range
- Each record has populated `sensors` JSONB

---

### Scenario 7: Frontend Integration

**Steps**:
1. Open frontend URL
2. Select demo device
3. Verify live data appears
4. Switch to different tabs

**Verification**:
```bash
# Open browser to http://localhost:5173

# Check frontend network requests (DevTools → Network)
# Expected calls:
#   GET /api/devices → device list with deviceType
#   GET /api/devices/demo-device-1 → device metadata
#   WS /ws/telemetry → WebSocket connection
#   GET /api/telemetry/demo-device-1 → historical data (Telemetry tab)
#   GET /api/triggers/demo-device-1 → trigger list (Triggers tab)
```

**Expected Result**: ✅ PASS
- Frontend loads and displays device selector
- Live sensor values update every 15 seconds
- Tabs show:
  - **Overview**: Device type, last seen, status
  - **Telemetry**: Historical data table
  - **Sensors**: Live snapshots with discovered sensor list
  - **Triggers**: Configured triggers table

---

## Test Cleanup

```bash
docker compose --profile demo down
docker volume rm agronode_postgres_data  # Optional: full data reset
```

---

## Failure Diagnosis

### No telemetry appearing in database
- Check MQTT connection: `docker logs demo-device`
- Verify broker: `docker logs mosquitto`
- Check backend MQTT subscription: `docker logs agronode-backend-1 | grep -i "subscribe"`

### Device appearing offline
- Check: last telemetry was > 30 seconds ago
- Trigger backend refresh: send any telemetry from device

### Trigger not firing
- Verify device type of target: must be `"receiver"` or same as source
- Check sensor ID matches trigger sensor ID exactly
- Check min/max thresholds: published values must cross boundary

### WebSocket not connecting
- Check browser console for connection errors
- Verify port 8080 is accessible: `curl -v http://localhost:8080/ws/telemetry`
- Check backend logs for WebSocket errors

---

## Test Results Template

```
Date: [YYYY-MM-DD]
Tester: [Name]
Result: [PASS/FAIL]

Scenario 1 (Per-Sensor Telemetry): [PASS/FAIL]
Scenario 2 (Device Registration): [PASS/FAIL]
Scenario 3 (Device Type): [PASS/FAIL]
Scenario 4 (Triggers): [PASS/FAIL]
Scenario 5 (WebSocket): [PASS/FAIL]
Scenario 6 (Historical Query): [PASS/FAIL]
Scenario 7 (Frontend): [PASS/FAIL]

Notes: [Any observations or issues]
```
