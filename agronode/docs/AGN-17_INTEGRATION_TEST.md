# AGN-17 Integration Test Report

Date: 2026-05-14

## Scope
Verify end-to-end flow:
ESP32 publisher (simulated) -> MQTT -> Backend -> PostgreSQL -> Frontend API consumption.

## Test Steps
1. Start stack services with Docker Compose.
2. Start backend app runtime inside backend container on port `19191`.
3. Publish telemetry messages to topic `agronode/esp32-lab/telemetry`.
4. Verify records in PostgreSQL table `sensor_data`.
5. Verify backend endpoints:
   - `GET /api/data/esp32-lab`
   - `GET /api/latest/esp32-lab`
6. Verify frontend container can consume backend API (`http://backend:19191/api/latest/esp32-lab`).

## Result
PASS - End-to-end telemetry data flow is working.

## Evidence Snapshot
- Backend logs include `telemetry processed` entries for `esp32-lab`.
- PostgreSQL query returns ingested rows for `esp32-lab`.
- Backend API returns expected JSON telemetry payloads.
- Frontend container network can fetch latest telemetry successfully.
