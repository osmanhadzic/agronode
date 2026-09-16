# AgroNode UI Design

---

## Overview

The frontend is a React-based dashboard for monitoring device telemetry in real-time, configuring sensor triggers, and managing multi-device deployments. It communicates exclusively via REST API and WebSocket, with no direct MQTT access.

---

## Application Structure

```
src/
  App.tsx (main router)
  pages/
    DashboardPage.tsx (route: /)
  components/
    DeviceSelector.tsx
    DataModeSelector.tsx
    DateFilter.tsx
    DeviceMetaPanel.tsx
    SensorVisibilitySelector.tsx
    SensorCard.tsx
  charts/
    TelemetryLineChart.tsx
  api/
    httpClient.ts
    telemetryApi.ts
    telemetrySocket.ts
  types/
    telemetry.ts
```

---

## Dashboard Page (`pages/DashboardPage.tsx`)

Tabbed interface with four main sections:

### Tab 1: Overview
- Device name, ID, and **device type** badge (publisher/receiver/unknown)
- Last seen timestamp
- Connection status (online/offline based on last_seen)
- Discovered sensors count
- Quick stats

### Tab 2: Telemetry
- Historical data queries by date range
- Table view with timestamp, sensor, value, and unit
- Export to CSV (optional)

### Tab 3: Sensors
- Full sensor catalog for selected device
- Per-sensor configuration:
  - Sensor ID
  - Last value (from live cache)
  - Last update time
  - Min/max trigger thresholds
  - Target device for activation
  - Enable/disable toggle

### Tab 4: Triggers
- Table of all triggers for selected device
- Columns: sensor ID, min threshold, max threshold, target device, activated status
- Actions: edit, delete, test activation (optional)

---

## Real-Time Data Caching

### Live Value Snapshots

Frontend maintains `liveSensorSnapshots` state:

```typescript
interface LiveSensorSnapshots {
  [deviceId: string]: {
    [sensorId: string]: number
  }
}
```

When WebSocket telemetry arrives:
1. Update latest value for `{deviceId, sensorId}`
2. Re-render sensor cards and charts
3. Persist last known value across network gaps
4. Display `--` only if **never** received a value

### Benefits
- Smooth UX during intermittent MQTT delays
- Accurate last-known-good values
- Fast sensor card rendering

---

## Key Components

### DeviceSelector
```typescript
interface DeviceSummary {
  id: string
  name: string
  deviceType?: "publisher" | "receiver" | "unknown"
}
```

- Dropdown list of all registered devices
- Displays `{id} ({deviceType})`
- Updates on selection
- Filters trigger targets by `deviceType` when appropriate

### SensorCard
- Sensor ID (sensorId)
- Current live value from snapshot
- Unit (derived from sensors map metadata)
- Color-coded status
  - Green: value within safe range
  - Yellow: approaching boundary
  - Red: exceeding threshold
- Click to view/edit triggers

### TelemetryLineChart (Recharts)
- X-axis: timestamp
- Y-axis: sensor value
- Dynamic series based on sensor visibility
- Auto-downsampling for datasets > 100 points
- Responsive sizing
- Legend with sensor color coding

### SensorVisibilitySelector
- Checkbox list of all available sensors
- Toggles visibility in cards and chart
- Persistent toggle state (localStorage)

### DateFilter
- Date range picker for historical queries
- Presets: last 24h, last 7d, last 30d, custom
- Sends range to backend for filtered telemetry

### DataModeSelector
- Toggle between "Live" (WebSocket) and "Historical" (REST query)
- Live mode: real-time updates
- Historical mode: date-filtered dataset

### DeviceMetaPanel
- Displays `device_type`, `last_seen`, `signal_strength`, `firmware_version`
- Metadata from backend `/devices/:id` endpoint
- Updates on device selection or manual refresh

---

## Data Flow

### Telemetry Ingestion (Real-Time)

```
WebSocket Connection (ws://host/ws/telemetry)
  ↓
Receive TelemetryMessage
  {
    deviceId: string
    sensorId: string
    sensors: Record<string, number>
    timestamp: number
  }
  ↓
Update liveSensorSnapshots state
  ↓
Re-render SensorCard(s) with live value
  ↓
Add data point to TelemetryLineChart
```

### Trigger Configuration

```
User clicks "Set Trigger" in SensorCard
  ↓
Modal opens with:
  - Min threshold input
  - Max threshold input
  - Target device selector (filtered by receiver type)
  ↓
Save button → POST /api/triggers/{deviceId}/{sensorId}
  ↓
Backend validates and stores
  ↓
Trigger table updates
  ↓
Backend starts evaluating sensor values
```

### Historical Query

```
User selects date range + clicks "Load"
  ↓
Frontend calls GET /api/telemetry/{deviceId}?from=...&to=...&sensorId=...
  ↓
Backend returns SensorData array
  ↓
Frontend populates table and chart (non-live mode)
  ↓
Chart shows historical trend
```

---

## API Integration

### httpClient Setup
- Base URL from environment (`import.meta.env.VITE_API_URL`)
- Headers: `X-Organization-ID` (from localStorage or session)
- Error interceptor for 401/403/500 responses

### telemetryApi
- GET `/devices` - list all devices
- GET `/devices/:id` - device details with metadata
- GET `/telemetry/:deviceId` - historical data (query params: from, to, sensorId, limit)
- POST `/triggers/:deviceId/:sensorId` - create/update trigger
- GET `/triggers/:deviceId` - list triggers for device
- DELETE `/triggers/:deviceId/:sensorId` - delete trigger

### telemetrySocket (WebSocket)
- URL: `ws://host/ws/telemetry`
- Message format: `{ type: "telemetry", payload: {...} }`
- Auto-reconnect with exponential backoff
- Graceful disconnect on unmount

---

## State Management

### Local Component State
```typescript
const [selectedDeviceId, setSelectedDeviceId] = useState<string>("")
const [liveSensorSnapshots, setLiveSensorSnapshots] = useState<Record<string, Record<string, number>>>({})
const [visibleSensors, setVisibleSensors] = useState<Set<string>>(new Set())
const [triggers, setTriggers] = useState<SensorTrigger[]>([])
const [dataMode, setDataMode] = useState<"live" | "historical">("live")
const [dateRange, setDateRange] = useState<[Date, Date]>([...])
```

### Persistence
- Device ID: URL query param (`?device=esp32-lab`)
- Visible sensors: localStorage (`agronode_visible_sensors_{deviceId}`)
- Data mode: session state
- Date range: component state

---

## Responsive Design

- Mobile: single column, stacked cards
- Tablet: two-column sensor grid, sidebar for filters
- Desktop: three-column layout with full chart width

---

## Error Handling

- **Connection lost**: WebSocket reconnect toast + fallback to REST polling
- **API error**: Toast with error message (status + detail)
- **Missing data**: Placeholder text or chart skeleton
- **Timeout**: Retry button, exponential backoff

---

## Performance Optimizations

- Memoized sensor cards to prevent unnecessary re-renders
- Chart data downsampling (100-point target for visuals)
- Lazy load historical data (pagination)
- WebSocket batching (multiple readings per frame)
- Index by `deviceId` and `sensorId` for O(1) lookups

---

## Future Enhancements

- Drag-and-drop card reordering
- Custom chart aggregations (avg, min, max per time bucket)
- Alert history panel
- Batch device management (multi-select, bulk actions)
- Dark mode toggle
- Data export (CSV, JSON)
- Mobile app (React Native)
