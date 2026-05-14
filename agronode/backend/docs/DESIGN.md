
---

# 📁 `backend/docs/DESIGN.md`

```md
# Backend Design - AgroNode

---

# Architecture

- Handlers (HTTP layer)
- Services (business logic)
- Repositories (DB layer)
- MQTT (message ingestion)

---

# Data Flow

ESP32 → MQTT → Backend → Service → Repository → PostgreSQL

---

# MQTT Module
- Subscribe to agronode/#
- Parse JSON payload
- Extract deviceId from topic
- Send to service layer

---

# Service Layer
- Business logic only
- No DB or MQTT code

---

# Repository Layer
- Database queries only
- Use GORM

---

# Rules
- No business logic in handlers
- Use interfaces
- Keep modules independent

---

# Scalability
Phase 1: single device
Phase 2: multi-device
Phase 3: multi-tenant SaaS
