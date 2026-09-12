# AgroNode Frontend

React + TypeScript dashboard for AgroNode telemetry visualization.

## Features

- Device selection
- Sensor cards (temperature, humidity, dynamic sensors)
- Sensor visibility toggles
- Realtime chart updates via WebSocket
- API fallback/history loading via REST
- Trigger configuration per sensor (min/max)
- Target device selection for trigger activation

## Run locally

```bash
npm install
npm run dev
```

Default URL: `http://localhost:5173`

## Environment

Optional:

- `VITE_API_BASE_URL` (example: `http://localhost:8080`)

Backend runtime variables:

- `SEED_DEMO_DATA` (`true` by default; set to `false` to skip demo data seeding)

Login is handled by the backend with these default development credentials unless overridden by environment variables:

- email: `admin@agronode.local`
- password: `admin123`

On first startup, the backend seeds these into the `users` table as a hashed bootstrap account.

It also seeds demo data for `organizations`, `farms`, `fields`, `zones`, and a couple of `devices`.
Additional seeded user:

- email: `operator@agronode.local`
- password: `operator123`

If not set, frontend defaults to backend at `http://<host>:8080` and WebSocket at `ws://<host>:8080/ws/telemetry`.

## API dependencies

- `GET /api/data`
- `GET /api/latest/:deviceId`
- `GET /api/triggers/:deviceId`
- `GET /api/triggers/:deviceId/:sensor`
- `PUT /api/triggers/:deviceId/:sensor`
- `DELETE /api/triggers/:deviceId/:sensor`
- `GET /ws/telemetry` (WebSocket)

See UI design notes in `docs/UI.md`.
