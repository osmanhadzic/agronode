# AgroNode - Task Board

This file is the single source of truth for implementation order.

RULE:
- Do NOT implement outside the current phase
- Complete tasks in order
- Each phase must be finished before moving to the next

---

# 🟢 EPIC 1: PROJECT BOOTSTRAP

## AGN-1: Initialize repository structure
- Create folders:
  /backend
  /frontend
  /firmware
  /infra
  /backend/docs
  /frontend/docs

STATUS: DONE

---

## AGN-2: Setup docker-compose
- Add services:
  - mosquitto
  - postgres
  - backend
  - frontend
- Ensure networking between services

STATUS: DONE

---

## AGN-3: Setup environment files
- Create .env.example
- Define DB and MQTT configs
- Ensure backend reads env variables

STATUS: DONE

---

# 🟡 EPIC 2: MQTT INFRASTRUCTURE

## AGN-4: Configure MQTT broker
- Setup Eclipse Mosquitto
- Enable port 1883
- Add config file

STATUS: DONE

---

## AGN-5: Define MQTT contract
- Implement topic structure:
  agronode/{deviceId}/telemetry
- Define JSON payload structure
- Create MQTT_CONTRACT.md

STATUS: DONE

---

## AGN-6: ESP32 firmware (basic publisher)
- Connect to WiFi
- Connect to MQTT broker
- Publish temperature + humidity every 5 seconds
- Use DHT22 sensor

STATUS: DONE

---

# 🔵 EPIC 3: BACKEND CORE

## AGN-7: Initialize Go backend
- Setup Gin framework
- Setup project structure (clean architecture)

STATUS: DONE

---

## AGN-8: Database setup
- Setup PostgreSQL connection
- Create sensor_data table
- Add migrations

STATUS: DONE

---

## AGN-9: MQTT subscriber service
- Subscribe to agronode/#
- Parse JSON payload
- Extract deviceId from topic
- Forward data to service layer

STATUS: DONE

---

## AGN-10: Service layer implementation
- Validate sensor data
- Process incoming telemetry
- Prepare data for storage

STATUS: DONE

---

## AGN-11: Repository layer
- Implement GORM repository
- Save sensor data into DB
- Query by deviceId

STATUS: DONE

---

## AGN-12: REST API implementation
- GET /api/data
- GET /api/data/:deviceId
- GET /api/latest/:deviceId

STATUS: DONE

---

# 🟣 EPIC 4: FRONTEND DASHBOARD

## AGN-13: Initialize React app
- Setup Vite + React + TypeScript
- Setup folder structure

STATUS: DONE

---

## AGN-14: API integration layer
- Create Axios client
- Connect to backend API
- Handle data fetching

STATUS: DONE

---

## AGN-15: Dashboard UI
- Create sensor cards
- Show temperature & humidity
- Device selector dropdown

STATUS: DONE

---

## AGN-16: Live chart
- Implement Recharts line chart
- Auto refresh every 5 seconds

STATUS: DONE

---

# ⚙️ EPIC 5: INTEGRATION

## AGN-17: Full system integration test
- ESP32 → MQTT → Backend → DB → Frontend
- Verify end-to-end data flow

STATUS: DONE

---

## AGN-18: Debug & fix issues
- Fix MQTT parsing issues
- Fix API inconsistencies
- Fix frontend data sync

STATUS: DONE

---

# 🚀 EPIC 6: PRODUCTION READY

## AGN-19: Docker full deployment test
- Run full stack via docker-compose
- Validate system stability

STATUS: DONE

---

## AGN-20: Documentation finalization
- Update README
- Finalize ARCHITECTURE.md
- Ensure all docs are consistent

STATUS: DONE

---

# ⚙️ EPIC 6: DEVICE REGISTRY

## AGN-21: Create devices database schema
- Create devices table
- Add unique device_id
- Add timestamps
- Add status field
- Add firmware_version field

STATUS: DONE

---

## AGN-22: Create device entity/model
- Create Device entity/model
- Add validation rules
- Add serialization support

STATUS: DONE

---

## AGN-23: Implement device registration API
- Create POST /api/devices/register
- Validate payload
- Implement idempotent registration

STATUS: DONE

---

## AGN-24: Implement get device API
- Create GET /api/devices/:deviceId
- Return device details
- Handle 404 errors

STATUS: DONE

---

## AGN-25: Implement list devices API
- Create GET /api/devices
- Add pagination
- Add filtering
- Add search support

STATUS: DONE

---

## AGN-26: MQTT telemetry presence integration
- Subscribe to telemetry topics
- Extract deviceId
- Update lastSeen
- Mark device online

STATUS: DONE

---

## AGN-27: Implement offline detection worker
- Create scheduled worker/cron
- Add inactivity threshold
- Mark inactive devices offline

STATUS: DONE

---

## AGN-28: Add device status events
- Emit device.online event
- Emit device.offline event
- Prevent duplicate events

STATUS: DONE

---

## AGN-29: Add device metadata support
- Support battery metadata
- Support signal_strength metadata
- Support hardware metadata

STATUS: DONE

---

## AGN-30: Implement dynamic sensor discovery
- Parse sensor keys dynamically
- Store discovered sensors
- Avoid hardcoded sensor types

STATUS: DONE

---

## AGN-31: Prepare device authentication support
- Add api_key field
- Add provisioning_token field
- Add hashing support

STATUS: DONE

---

## AGN-32: Implement real-time device presence updates
- Add WebSocket support
- Broadcast online/offline changes
- Sync frontend dashboard

STATUS: DONE

---

## AGN-33: Add device audit logging
- Log registrations
- Log status changes
- Log firmware updates

STATUS: DONE

---

## AGN-34: Add device tags support
- Add tags field
- Support filtering by tags
- Support multiple tags

STATUS: DONE

---

## AGN-35: Prepare device shadow support
- Add desired_state field
- Add reported_state field
- Prepare future sync architecture

STATUS: DONE

---

# 🤖 COPILOT EXECUTION MODE

Use this workflow:

## STEP 1
"Implement AGN-1 only"

## STEP 2
"Implement AGN-2 only and ensure docker-compose works"

## STEP 3
"Continue with next task ONLY after previous is verified"

RULES FOR COPILOT:
- Do not skip tasks
- Do not implement multiple tasks at once
- Always follow architecture docs
- Always respect MQTT contract
