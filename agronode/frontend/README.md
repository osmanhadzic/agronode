# AgroNode Frontend

React + TypeScript dashboard for AgroNode telemetry visualization.

## Features

- Device selection
- Sensor cards (temperature, humidity, dynamic sensors)
- Sensor visibility toggles
- Realtime chart updates via WebSocket
- API fallback/history loading via REST

## Run locally

```bash
npm install
npm run dev
```

Default URL: `http://localhost:5173`

## Environment

Optional:

- `VITE_API_BASE_URL` (example: `http://localhost:8080`)

If not set, frontend defaults to backend at `http://<host>:8080` and WebSocket at `ws://<host>:8080/ws/telemetry`.

## API dependencies

- `GET /api/data`
- `GET /api/latest/:deviceId`
- `GET /ws/telemetry` (WebSocket)

See UI design notes in `docs/UI.md`.
