# AgroNode API Documentation

Base URL:
http://localhost:8080

---

# Endpoints

## Get all sensor data
GET /api/data

### Response
```json
[
  {
    "deviceId": "device-1",
    "temperature": 24.5,
    "humidity": 60,
    "sensors": {
      "temperature": 24.5,
      "humidity": 60,
      "co2": 450,
      "soil_moisture": 33
    },
    "createdAt": "2026-01-01T12:00:00Z"
  }
]
