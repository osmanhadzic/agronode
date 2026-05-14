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

STATUS: TODO

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
