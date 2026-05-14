# AgroNode UI Design

---

# Layout

+---------------------------+
| Device Selector (Navbar) |
+---------------------------+
| Sensor Cards             |
| Temperature | Humidity   |
+---------------------------+
| Live Chart               |
+---------------------------+
| History Table            |
+---------------------------+

---

# Pages

## Dashboard
- Live sensor data
- Charts
- Device switching

---

# Components

## SensorCard
- label
- value
- unit

## Chart
- realtime line chart
- auto refresh

## DeviceSelector
- dropdown list of devices

---

# Data Flow

Frontend → REST API → Backend → Database

---

# Rules
- No MQTT in frontend
- API only communication
- Responsive UI
- Reusable components
