# AgroNode UI Design

---

# Layout

+---------------------------+
| Device Selector (Navbar) |
+---------------------------+
| Sensor Visibility        |
+---------------------------+
| Sensor Cards             |
+--------------------------+
| Live Chart               |
+---------------------------+

---

# Pages

## Dashboard
- Live sensor data
- Charts
- Device switching
- Sensor visibility filtering

---

# Components

## SensorCard
- label
- value
- unit

## TelemetryLineChart
- realtime line chart
- dynamic sensor series

## DeviceSelector
- dropdown list of devices

## SensorVisibilitySelector
- checkbox list for available sensors
- toggles visible cards and chart lines

---

# Data Flow

Frontend → REST API + WebSocket → Backend → Database

---

# Rules
- No MQTT in frontend
- API only communication
- Responsive UI
- Reusable components
