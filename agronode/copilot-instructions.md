# Agronode AI Coding Instructions

This repository is a production-style IoT platform.

Architecture:

ESP32 -> MQTT Broker -> Go Backend -> PostgreSQL -> React Frontend

The codebase must remain:
- modular
- scalable
- readable
- production-ready

---

# General Rules

- Prefer clean architecture
- Prefer composition over inheritance
- Avoid large files
- Keep functions focused
- Avoid duplicated logic
- Use meaningful names
- Write maintainable code
- Add comments only where useful
- Do not overengineer

---

# Backend Standards (Go)

Backend stack:
- Gin
- GORM
- PostgreSQL
- MQTT (Paho)

Structure:

backend/
  internal/
    handlers/
    services/
    repositories/
    models/
    mqtt/
    config/
    database/

Rules:
- Use dependency injection
- Business logic belongs in services
- Handlers should stay thin
- Repositories only access DB
- Use interfaces where appropriate
- Use context.Context
- Return structured errors
- Add structured logging
- Avoid global state
- Use environment variables

---

# MQTT Rules

Topic structure:

weather/{deviceId}/reading

Example:

weather/esp32-lab/reading

Payload example:

```json
{
  "device_id": "esp32-lab",
  "temperature": 24.5,
  "humidity": 60,
  "timestamp": "2026-05-14T12:00:00Z"
}
```

Rules:
- Never hardcode topics
- Validate payloads
- Handle malformed messages safely
- Log MQTT connection events
- Reconnect automatically

---

# Database Rules

PostgreSQL is the primary database.

Requirements:
- Use migrations
- Avoid raw SQL unless necessary
- Add indexes where useful
- Use timestamps consistently
- Keep models simple

---

# Frontend Standards

Frontend stack:
- React
- TypeScript
- Vite
- Recharts

Rules:
- Use functional components
- Prefer hooks
- Keep components small
- Separate API layer
- Use typed API responses
- Avoid prop drilling
- Keep charts reusable
- Use responsive layouts

Suggested structure:

src/
  components/
  pages/
  hooks/
  api/
  types/
  charts/

---

# Infrastructure Rules

Infrastructure uses:
- Docker
- Docker Compose
- Mosquitto
- PostgreSQL

Requirements:
- Services must be containerized
- Use environment variables
- Keep docker-compose readable
- Separate infrastructure configs

---

# Security Rules

- Never commit secrets
- Use .env files
- Validate API input
- Sanitize MQTT payloads
- Avoid exposing internal errors
- Prepare architecture for authentication

---

# Code Style

Prefer:
- readability over cleverness
- explicit code over magic
- maintainability over premature optimization

Avoid:
- giant functions
- tight coupling
- duplicated code
- unnecessary abstractions

---

# Future Expansion

The system should later support:
- multiple devices
- WebSockets
- authentication
- alerting
- analytics
- OTA updates
- cloud deployment
- device management

Design decisions should consider future scalability.
