# AgroNode Frontend

React + TypeScript application for real-time monitoring of IoT device telemetry, sensor trigger configuration, and historical data visualization.

---

## Quick Start

### Prerequisites

- Node.js 18+
- npm or yarn
- Running AgroNode backend (http://localhost:8080)

### Local Development

```bash
# Install dependencies
cd agronode/frontend
npm install

# Create environment file
cp .env.example .env
# Edit .env if needed:
# VITE_API_URL=http://localhost:8080

# Start dev server
npm run dev
# Open http://localhost:5173 in browser
```

### Docker Compose (Recommended)

```bash
cd agronode
docker compose up -d
# Frontend available at http://localhost:5173
```

---

## Architecture

### Directory Structure

```
frontend/
├── src/
│   ├── App.tsx                 # Main app component
│   ├── App.css
│   ├── index.css
│   ├── main.tsx
│   ├── api/
│   │   ├── httpClient.ts       # Axios instance with org header
│   │   ├── telemetryApi.ts     # REST endpoints
│   │   └── telemetrySocket.ts  # WebSocket client
│   ├── assets/
│   ├── charts/
│   │   └── TelemetryLineChart.tsx  # Recharts wrapper
│   ├── components/
│   │   ├── DataModeSelector.tsx    # Live vs Historical toggle
│   │   ├── DateFilter.tsx          # Date range picker
│   │   ├── DeviceMetaPanel.tsx     # Device info display
│   │   ├── DeviceSelector.tsx      # Device dropdown
│   │   ├── SensorCard.tsx          # Individual sensor display
│   │   └── SensorVisibilitySelector.tsx
│   ├── pages/
│   │   └── DashboardPage.tsx       # Tab-based main view
│   ├── types/
│   │   └── telemetry.ts            # TypeScript interfaces
│   └── index.css
├── public/                     # Static assets
├── vite.config.ts
├── tsconfig.json
├── eslint.config.js
├── package.json
└── README.md (this file)
```

### Component Hierarchy

```
App
└── DashboardPage
    ├── DeviceSelector
    ├── DataModeSelector
    ├── DateFilter
    ├── DeviceMetaPanel
    ├── Tabs
    │   ├── OverviewTab
    │   │   └── SensorCard[]
    │   ├── TelemetryTab
    │   │   └── DataTable
    │   ├── SensorsTab
    │   │   ├── SensorVisibilitySelector
    │   │   └── SensorCard[]
    │   └── TriggersTab
    │       └── TriggersTable
    └── TelemetryLineChart
```

---

## Key Features

### Real-Time Monitoring

- **Live Value Caching**: Maintains last known sensor values across network gaps
- **WebSocket Streaming**: Receives updates every 15 seconds (or as published)
- **Automatic Reconnection**: Exponential backoff on connection loss
- **Sensor Cards**: Color-coded status (green/yellow/red)

### Historical Data

- **Date Range Queries**: Filter telemetry by time window
- **Per-Sensor Views**: Compare individual sensors over time
- **Chart Visualization**: Recharts line chart with dynamic series
- **Auto-Downsampling**: Reduces 1000+ points to 100 for performance

### Trigger Management

- **Min/Max Configuration**: Set threshold boundaries per sensor
- **Target Device Selection**: Route activation commands to receiver devices
- **Visual Status**: See which triggers are currently active
- **Edit/Delete Actions**: Modify or remove triggers

### Multi-Device Support

- **Device Type Badges**: Visual distinction (publisher/receiver/unknown)
- **Quick Switch**: Dropdown selector for all registered devices
- **Metadata Display**: Signal strength, firmware, last seen timestamp
- **Sensor Discovery**: Auto-populated list from telemetry

---

## API Integration

### HTTP Endpoints

Defined in `src/api/telemetryApi.ts`:

```typescript
// Devices
getDevices()           // GET /api/devices
getDevice(id)          // GET /api/devices/:id
registerDevice(data)   // POST /api/devices/register

// Telemetry
getTelemetry(deviceId, query) // GET /api/telemetry/:deviceId?from=...&to=...
getLatest(deviceId)           // GET /api/latest/:deviceId

// Triggers
getTriggers(deviceId)                    // GET /api/triggers/:deviceId
setTrigger(deviceId, sensorId, data)     // POST /api/triggers/:deviceId/:sensorId
deleteTrigger(deviceId, sensorId)        // DELETE /api/triggers/:deviceId/:sensorId
```

### WebSocket Connection

Established in `src/api/telemetrySocket.ts`:

```typescript
// Auto-connects on mount
const socket = new TelemetrySocket('ws://localhost:8080/ws/telemetry')

// Listen for telemetry
socket.onTelemetry((message) => {
  console.log(message)
  // { deviceId, sensorId, sensors: {...}, timestamp }
})

// Auto-reconnects with exponential backoff on disconnect
```

### Context & Headers

All requests include `X-Organization-ID` header for multi-tenancy:

```typescript
// Set once on app load:
const orgId = localStorage.getItem('orgId') || '1'
httpClient.defaults.headers['X-Organization-ID'] = orgId
```

---

## Data Flow

### Live Mode (WebSocket)

```
User opens DashboardPage
    ↓
DeviceSelector mounts → fetch device list
    ↓
TelemetrySocket established → connection to backend
    ↓
WebSocket receives message every 15s:
    {
      type: "telemetry",
      payload: {
        deviceId: "esp32-lab",
        sensorId: "dht11-temp",
        sensors: { "dht11-temp": 24.5, ... },
        timestamp: 1715539200
      }
    }
    ↓
Update liveSensorSnapshots state:
    { "esp32-lab": { "dht11-temp": 24.5, ... } }
    ↓
Re-render SensorCard components
    ↓
Add point to TelemetryLineChart
```

### Historical Mode (REST)

```
User selects date range
    ↓
Click "Load Historical Data"
    ↓
Frontend calls:
    GET /api/telemetry/{deviceId}?from=...&to=...&sensorId=...
    ↓
Backend returns array of SensorData records
    ↓
Frontend renders in table + chart
    ↓
Chart aggregates points (downsample to 100)
```

### Trigger Configuration

```
User clicks "Set Trigger" on SensorCard
    ↓
Modal opens with:
    - Min threshold input
    - Max threshold input
    - Target device selector (filtered by receiver type)
    ↓
User enters: min=20, max=30, target=receiver-device-1
    ↓
Click "Save Trigger"
    ↓
Frontend calls:
    POST /api/triggers/{deviceId}/{sensorId}
    payload: { minThreshold: 20, maxThreshold: 30, targetDeviceId: "..." }
    ↓
Backend validates and stores
    ↓
Trigger table updates
    ↓
Backend begins evaluating sensor values
    ↓
When threshold crossed, backend publishes activation command
```

---

## State Management

### Component-Level State

```typescript
// DashboardPage.tsx
const [selectedDeviceId, setSelectedDeviceId] = useState<string>("")
const [liveSensorSnapshots, setLiveSensorSnapshots] = useState<Record<string, Record<string, number>>>({})
const [visibleSensors, setVisibleSensors] = useState<Set<string>>(new Set())
const [triggers, setTriggers] = useState<SensorTrigger[]>([])
const [telemetryData, setTelemetryData] = useState<SensorData[]>([])
const [dataMode, setDataMode] = useState<"live" | "historical">("live")
const [dateRange, setDateRange] = useState<[Date, Date]>([...])
const [deviceMetadata, setDeviceMetadata] = useState<DeviceSummary | null>(null)
```

### Persistence

- **localStorage**: Sensor visibility, selected organization, user preferences
- **URL Query Params**: Selected device (`?device=esp32-lab`) for bookmarking
- **Session State**: Data mode and date range (cleared on refresh)

---

## Type Definitions

See `src/types/telemetry.ts`:

```typescript
interface DeviceSummary {
  id: string
  name: string
  deviceType?: "publisher" | "receiver" | "unknown"
  status?: "online" | "offline"
  lastSeen?: string
  discoveredSensors?: string[]
}

interface SensorData {
  id: number
  deviceId: string
  sensorId: string
  sensors: Record<string, number>
  timestamp: number
  metadata?: Record<string, any>
}

interface SensorTrigger {
  id: string
  deviceId: string
  sensorId: string
  minThreshold?: number
  maxThreshold?: number
  targetDeviceId?: string
  activated: boolean
  createdAt: string
}

interface TelemetryMessage {
  type: "telemetry" | "status" | "activation"
  payload: {
    deviceId: string
    sensorId?: string
    sensors: Record<string, number>
    timestamp: number
    metadata?: Record<string, any>
  }
}
```

---

## Dashboard Tabs

### 1. Overview Tab

**Display**:
- Device ID and name
- Device type badge (publisher/receiver/unknown)
- Last seen timestamp
- Current status (online/offline)
- Signal strength (RSSI)
- Discovered sensors count

**Refresh**: On device change or manual refresh

### 2. Telemetry Tab

**Display**:
- Historical data table
- Date range filter (presets + custom)
- Columns: Timestamp, Sensor ID, Value, Unit

**Query**:
```
GET /api/telemetry/{deviceId}?from={ts}&to={ts}&limit=100
```

**Export**: CSV export (optional feature)

### 3. Sensors Tab

**Display**:
- Multi-column sensor grid
- Each sensor shows:
  - Sensor ID
  - Live value from snapshot
  - Last update time
  - Min/max trigger thresholds
  - Enable/disable toggle

**Actions**:
- Click card to view/edit triggers
- Visibility checkboxes (toggles chart display)

### 4. Triggers Tab

**Display**:
- Table with columns: Sensor ID, Min, Max, Target Device, Activated, Actions
- Color-coded status icons (green=inactive, red=active)

**Actions**:
- Edit trigger
- Delete trigger
- View activation history (future)

---

## Charts & Visualization

### TelemetryLineChart

Powered by Recharts:

```typescript
<TelemetryLineChart
  data={telemetryData}           // Array of SensorData
  visibleSensors={visibleSensors} // Set<sensorId>
  darkMode={false}
  height={400}
/>
```

**Features**:
- Symbol at each data point
- Legend with color indicators
- Tooltip on hover
- X-axis: timestamp
- Y-axis: sensor value
- Auto-scaling axes
- Responsive sizing

**Performance**:
- Downsamples datasets > 100 points to 100-point target
- Uses useMemo to prevent unnecessary re-renders
- Lazy renders for hidden series

---

## Responsive Design

### Breakpoints

- **Mobile** (< 640px): Single column, stacked cards
- **Tablet** (640px–1024px): Two-column grid, sidebar filters
- **Desktop** (> 1024px): Three-column layout, full-width chart

### CSS Strategy

- Tailwind CSS (or CSS Modules)
- Flexbox for layouts
- Media queries for responsive behavior

---

## Error Handling

### Connection Errors

```typescript
// WebSocket reconnect with exponential backoff
socket.onError(() => {
  showToast("Connection lost. Retrying...", "warning")
  // Auto-retry with 1s, 2s, 4s, 8s delays
})

// Fallback to REST polling
if (socket.disconnected && dataMode === "live") {
  startPollingTelemetry()
}
```

### API Errors

```typescript
httpClient.interceptors.response.use(
  response => response,
  error => {
    if (error.response?.status === 401) {
      showToast("Unauthorized. Please log in.", "error")
      // Redirect to login
    } else if (error.response?.status === 404) {
      showToast("Device not found.", "error")
    } else {
      showToast(`Error: ${error.message}`, "error")
    }
    return Promise.reject(error)
  }
)
```

### Missing Data

- Display `--` only if value has **never** been received
- Otherwise show last known value (from liveSensorSnapshots)
- Chart shows gap in line if no data point for time range

---

## Performance Tips

1. **Memoize Components**: Use `React.memo()` for SensorCard to prevent re-renders
2. **Lazy Load Tab Content**: Load historical data only when Telemetry tab selected
3. **Debounce Filters**: Delay chart updates while date range is being adjusted
4. **Virtual Lists**: For tables with 1000+ rows, use react-virtual
5. **Code Splitting**: Dynamic imports for large components

---

## Development

### Build for Production

```bash
npm run build
# Generates dist/ folder with optimized assets
```

### Run Production Build Locally

```bash
npm run preview
# Serves dist/ at http://localhost:4173
```

### Linting

```bash
npm run lint
# ESLint with configured rules
```

### TypeScript Check

```bash
npm run type-check
# Full type verification without building
```

---

## Environment Configuration

### Development (`.env.development`)

```
VITE_API_URL=http://localhost:8080
VITE_LOG_LEVEL=debug
```

### Production (`.env.production`)

```
VITE_API_URL=https://api.example.com
VITE_LOG_LEVEL=info
```

### Build-Time Variables

Accessed via `import.meta.env.VITE_*`:

```typescript
const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:8080'
```

---

## Deployment

### Docker Image

```dockerfile
FROM node:18-alpine as builder
WORKDIR /app
COPY . .
RUN npm install && npm run build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

### Docker Compose

```bash
docker compose up -d frontend
# Runs on port 5173 in dev mode
```

### Vercel / Netlify

```bash
# Vercel
vercel deploy

# Netlify
netlify deploy --prod --dir=dist
```

### Static Hosting (S3, CloudFront, etc.)

```bash
npm run build
# Upload dist/ to S3 bucket
# Configure CloudFront as CDN
```

---

## Accessibility

- Semantic HTML (`<main>`, `<nav>`, `<section>`)
- ARIA labels for interactive elements
- Keyboard navigation support
- Color contrast compliance (AA standard)
- Form labels associated with inputs

---

## Browser Support

- Chrome 90+
- Firefox 88+
- Safari 14+
- Edge 90+

(Tested via Vite's browserslist config)

---

## Troubleshooting

### WebSocket Connection Failed

```
error: Failed to establish WebSocket connection
```

- Verify backend is running: `curl http://localhost:8080/api/health`
- Check CORS headers: `Access-Control-Allow-Origin: *`
- Check firewall rules for port 8080

### API Errors (401, 403, 500)

- Verify `X-Organization-ID` header is set
- Check `VITE_API_URL` points to correct backend
- Review backend logs: `docker logs agronode-backend-1`

### Data Not Updating

- Check WebSocket VS REST mode in DataModeSelector
- Verify device is publishing telemetry (check backend logs)
- Try manual refresh or toggle device selector

### Chart Not Rendering

- Verify telemetryData array is populated
- Check visibleSensors includes sensor ID
- Check console for chart errors (DevTools)

---

## Contributing

- Follow TypeScript strict mode
- Write tests for new components
- Update type definitions in `types/telemetry.ts`
- Keep component props simple and documented
- Use Storybook for component documentation (future)

---

## Future Enhancements

- Dark mode toggle
- Custom data aggregation (avg, min, max per bucket)
- Alert history and notifications
- Batch device management
- Data export (CSV, JSON, PDF)
- Mobile-first responsive redesign
- React Native mobile app
- Real-time collaborative dashboards

---

## License

See [LICENSE](../../LICENSE)

---

## References

- [UI Design Document](docs/UI.md)
- [API Documentation](../backend/docs/API.md)
- [Architecture Overview](../ARCHITECTURE.md)
- [Vite Documentation](https://vitejs.dev)
- [React Documentation](https://react.dev)
- [TypeScript Handbook](https://www.typescriptlang.org/docs/)
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
