# AGN-19 Docker Stability Test Report

Date: 2026-05-14

## Goal
Run full stack via Docker Compose and validate system stability.

## Deployment Mode
- `backend`: runs `go run ./cmd/api` in container
- `frontend`: runs `npm run dev -- --host 0.0.0.0 --port 5173` in container
- `mosquitto`: Eclipse Mosquitto
- `postgres`: PostgreSQL 16

## Steps Executed
1. Clean restart with `docker compose down` then `docker compose up -d`.
2. Verified all containers remained `Up` after warm-up period.
3. Inspected logs for startup/runtime errors.
4. Published telemetry event to MQTT topic `agronode/agn19/telemetry`.
5. Verified backend API response from host:
   - `GET http://localhost:8080/api/latest/agn19`
6. Verified frontend availability:
   - `GET http://localhost:5173`
7. Verified DB persistence in `sensor_data` for device `agn19`.

## Result
PASS - Full stack runs via Docker Compose and remains stable through end-to-end telemetry flow.

## Notes
- Frontend currently runs in dev mode (`vite`) for this compose profile.
- Backend uses `go run` for runtime; suitable for development stability checks.
