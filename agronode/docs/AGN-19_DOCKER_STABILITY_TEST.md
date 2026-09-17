# AGN-19 Docker Stability Test Report

**Objective**: Validate full AgroNode stack stability when running via Docker Compose with all services orchestrated.

---

## Test Environment

**Configuration**:
```yaml
Services:
  backend: Go API on port 8080
  frontend: React dev server on port 5173
  mosquitto: MQTT broker on port 1883
  postgres: PostgreSQL 16 on port 5432
  demo-device: (optional) simulated MQTT publisher

Volumes:
  postgres_data: persistent storage

Network: docker compose default bridge
```

---

## Startup Procedure

```bash
cd /home/osman/Documents/ESP32/agronode

# Full clean start
docker compose down -v
docker compose --profile demo up -d

# Wait 10 seconds for services to initialize
sleep 10

# Verify all containers are running
docker compose ps
```

**Health Check**:
```bash
# All containers should show "Up" status
# Expected output:
# NAME                 STATUS
# agronode-postgres    Up X seconds
# agronode-mosquitto   Up X seconds
# agronode-backend     Up X seconds
# agronode-frontend    Up X seconds
# agronode-demo-device Up X seconds
```

---

## Test Suite

### T1: Container Stability

**Objective**: Verify containers start and remain running without crashes.

**Steps**:
```bash
# Get container uplimes after 5 minutes
sleep 300
docker compose ps

# Check for restarts (should be 0)
docker compose ps | grep -E "Up|Restarting"

# Check logs for critical errors
docker logs agronode-backend-1 | grep -i "error\|panic\|fatal" | head -5
docker logs agronode-frontend-1 | grep -i "error\|fatal" | head -5
docker logs agronode-postgres-1 | grep -i "error" | head -5
```

**Pass Criteria**:
- ✅ All containers show "Up" after 5 minutes
- ✅ No containers have restarted (restart count = 0)
- ✅ No fatal/panic errors in logs

---

### T2: Database Availability

**Objective**: Verify PostgreSQL initializes and accepts connections.

**Steps**:
```bash
# Test connection to PostgreSQL
docker exec agronode-postgres-1 psql -U postgres -d agronode -c "SELECT 1"

# Verify schema exists
docker exec agronode-postgres-1 psql -U postgres -d agronode -c \
  "SELECT table_name FROM information_schema.tables WHERE table_schema='public' LIMIT 5"

# Check migrations applied
docker exec agronode-postgres-1 psql -U postgres -d agronode -c \
  "SELECT version, dirty FROM schema_migrations ORDER BY version DESC LIMIT 3"
```

**Pass Criteria**:
- ✅ Connection successful ("1" returned)
- ✅ Tables visible: `devices`, `sensor_data`, `sensor_triggers`, etc.
- ✅ Latest migration applied and not dirty

---

### T3: MQTT Broker Connectivity

**Objective**: Verify MQTT broker accepts connections and forwards messages.

**Steps**:
```bash
# Test MQTT connection from backend
docker logs agronode-backend-1 | grep -i "mqtt\|connected" | head -10

# Verify broker is listening
docker exec agronode-mosquitto netstat -ln | grep 1883

# Publish test message
docker exec agronode-mosquitto mosquitto_pub -t test/topic -m "hello"

# Verify message was received (via backend logs or subscriber)
docker exec agronode-mosquitto mosquitto_sub -t "agronode/+/telemetry" -C 1 --wait-for-msg 10 &
# This should receive a message within 10 seconds if demo-device is running
```

**Pass Criteria**:
- ✅ Backend logs show successful MQTT connection
- ✅ Mosquitto listening on port 1883
- ✅ Messages flow through broker

---

### T4: Backend API Health

**Objective**: Verify backend accepts HTTP requests and serves responses.

**Steps**:
```bash
# Health endpoint
curl -s http://localhost:8080/api/health | jq '.'

# Device list endpoint
curl -s http://localhost:8080/api/devices | jq 'length'

# Specific device endpoint (if demo-device running)
curl -s http://localhost:8080/api/devices/demo-device-1 | jq '.deviceType'
```

**Pass Criteria**:
- ✅ Health endpoint returns 200 OK
- ✅ Devices endpoint returns array (even if empty)
- ✅ Device queries return correct `deviceType` field

---

### T5: End-to-End Telemetry Flow

**Objective**: Verify complete data path: MQTT publish → backend process → database store → API retrieve.

**Steps**:
```bash
# Baseline telemetry count
BEFORE=$(docker exec agronode-postgres-1 psql -U postgres -d agronode -t -c \
  "SELECT COUNT(*) FROM sensor_data WHERE device_id='demo-device-1'")

# Wait for demo device to publish (default interval 15 seconds)
sleep 20

# Count after interval
AFTER=$(docker exec agronode-postgres-1 psql -U postgres -d agronode -t -c \
  "SELECT COUNT(*) FROM sensor_data WHERE device_id='demo-device-1'")

# Should have at least 1 new record
echo "Before: $BEFORE, After: $AFTER"
[ "$AFTER" -gt "$BEFORE" ] && echo "✅ New records detected" || echo "❌ No new records"

# Verify latest data via API
curl -s http://localhost:8080/api/latest/demo-device-1 | jq '.[] | {timestamp, sensors}'
```

**Pass Criteria**:
- ✅ Database row count increases over time
- ✅ API returns sensor data with `sensors` JSONB populated
- ✅ Data freshness < 30 seconds old

---

### T6: WebSocket Real-Time Connection

**Objective**: Verify WebSocket endpoint streams live telemetry.

**Steps**:
```bash
# Connect WebSocket and capture one message (30 second timeout)
timeout 30s bash -c 'exec 3<>/dev/tcp/localhost/8080; echo -e "GET /ws/telemetry HTTP/1.1\r\nHost: localhost\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: AQ==\r\nSec-WebSocket-Version: 13\r\n\r\n" >&3; cat <&3' | grep -i "upgrade\|connection"

# Alternative using websocat (if installed):
timeout 5s websocat ws://localhost:8080/ws/telemetry | head -1 | jq '.type'
```

**Pass Criteria**:
- ✅ WebSocket upgrade successful
- ✅ Message received within timeout
- ✅ Message has `type: "telemetry"`

---

### T7: Frontend Availability

**Objective**: Verify frontend dev server serves HTML and assets.

**Steps**:
```bash
# Fetch HTML
curl -s http://localhost:5173 | head -20

# Check for Vite script
curl -s http://localhost:5173 | grep -i "vite\|import"

# Frontend logs for startup issues
docker logs agronode-frontend-1 | tail -20
```

**Pass Criteria**:
- ✅ HTTP 200 response
- ✅ HTML contains Vite/React bootstrap
- ✅ No red errors in frontend logs

---

### T8: Persistence and Recovery

**Objective**: Verify data persists and system recovers after container restart.

**Steps**:
```bash
# Record current device count
COUNT_BEFORE=$(curl -s http://localhost:8080/api/devices | jq 'length')

# Stop and restart backend only
docker compose restart agronode-backend-1

# Wait for readiness
sleep 3

# Verify device count unchanged
COUNT_AFTER=$(curl -s http://localhost:8080/api/devices | jq 'length')

echo "Devices before restart: $COUNT_BEFORE"
echo "Devices after restart: $COUNT_AFTER"
[ "$COUNT_BEFORE" = "$COUNT_AFTER" ] && echo "✅ Data persisted" || echo "❌ Data mismatch"
```

**Pass Criteria**:
- ✅ Device count unchanged after backend restart
- ✅ Database records still accessible
- ✅ No data loss

---

### T9: Trigger State Persistence

**Objective**: Verify trigger configurations survive service restarts.

**Steps**:
```bash
# Create trigger
curl -X POST http://localhost:8080/api/triggers/demo-device-1/dht11-temp \
  -H "Content-Type: application/json" \
  -d '{"sensorId": "dht11-temp", "minThreshold": 20, "maxThreshold": 30}'

# Verify trigger exists
TRIGGER_COUNT=$(curl -s http://localhost:8080/api/triggers/demo-device-1 | jq 'length')

# Restart backend
docker compose restart agronode-backend-1
sleep 3

# Re-check trigger count
TRIGGER_COUNT_AFTER=$(curl -s http://localhost:8080/api/triggers/demo-device-1 | jq 'length')

echo "Triggers before: $TRIGGER_COUNT, After: $TRIGGER_COUNT_AFTER"
[ "$TRIGGER_COUNT" = "$TRIGGER_COUNT_AFTER" ] && echo "✅ Triggers persisted" || echo "❌ Triggers lost"
```

**Pass Criteria**:
- ✅ Trigger count unchanged after restart
- ✅ Trigger configuration intact

---

## Stress Test (Optional)

### T10: High Volume Telemetry

**Objective**: Verify system handles burst of messages.

**Steps**:
```bash
# Publish 100 messages rapidly
for i in {1..100}; do
  docker exec agronode-mosquitto mosquitto_pub -h localhost -t "agronode/stress-test/telemetry" \
    -m "{\"deviceId\": \"stress-test\", \"sensors\": {\"value\": $i}}"
done

# Check processing
sleep 2
docker exec agronode-postgres-1 psql -U postgres -d agronode -c \
  "SELECT COUNT(*) FROM sensor_data WHERE device_id='stress-test'"

# Monitor backend memory (should remain stable)
docker stats agronode-backend-1 --no-stream
```

**Pass Criteria**:
- ✅ All 100 messages processed and stored
- ✅ Backend memory stable (< 500MB)
- ✅ No dropped messages in logs

---

## Cleanup

```bash
# Stop and remove containers
docker compose --profile demo down

# Full reset (delete volumes)
docker compose --profile demo down -v

# Optional: check disk usage
du -sh ~/.docker/volumes/
```

---

## Test Report Template

```
Date: [YYYY-MM-DD HH:MM]
Tester: [Name]
Duration: [Minutes]

T1 (Stability):        [PASS/FAIL]
T2 (Database):         [PASS/FAIL]
T3 (MQTT):             [PASS/FAIL]
T4 (API):              [PASS/FAIL]
T5 (E2E Flow):         [PASS/FAIL]
T6 (WebSocket):        [PASS/FAIL]
T7 (Frontend):         [PASS/FAIL]
T8 (Persistence):      [PASS/FAIL]
T9 (Trigger State):    [PASS/FAIL]
T10 (Stress):          [PASS/FAIL - Optional]

Overall Result: [PASS/FAIL]

Issues Found:
- [List any failures]

Notes:
- [Performance observations, improvements]
```
