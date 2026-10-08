# AgroNode AI Coding Instructions

AgroNode is a production-style, open-source IoT platform for intelligent greenhouse automation.

The system is designed to support:

- ESP32 IoT devices
- secure MQTT communication
- scalable Go backend services
- PostgreSQL application data
- TimescaleDB time-series telemetry
- fuzzy-logic automation
- weather and external integrations
- React frontend
- MCP-based AI tools
- observability
- Docker-based deployment with future Kubernetes support

The codebase must remain:

- modular
- scalable
- secure
- readable
- testable
- production-ready

---

# 1. Architecture

High-level architecture:

```text
ESP32
  │
  │ MQTT / TLS / mTLS
  ▼
VerneMQ
  │
  │ MQTT
  ▼
MQTT Ingestion
  │
  ▼
Telemetry Service
  │
  ├── TimescaleDB
  │
  └── Automation Service
          │
          ├── PostgreSQL
          └── TimescaleDB

React
  │
  │ REST / WebSocket
  ▼
API Gateway
  │
  ├── Device Service
  ├── Telemetry Service
  ├── Automation Service
  └── Integration Service

External APIs
  │
  │ HTTPS
  ▼
Integration Service

LLM
  │
  │ MCP
  ▼
.NET MCP Server
  │
  │ gRPC
  ▼
Go Backend Services

Observability:

Go Services
  │
  ▼
OpenTelemetry
  │
  ├── Prometheus
  ├── Loki
  └── Tempo
        │
        ▼
      Grafana
```

---

# 2. Architecture Principles

Prefer:

- Clean Architecture
- separation of concerns
- dependency injection
- composition over inheritance
- small focused modules
- explicit dependencies
- interfaces at boundaries
- context.Context propagation
- structured logging
- testable business logic
- meaningful names
- simple solutions

Avoid:

- giant files
- giant functions
- global mutable state
- unnecessary abstractions
- premature optimization
- duplicated business logic
- tight coupling between services
- direct database access from handlers
- business logic inside HTTP handlers
- business logic inside MQTT callbacks

Do not overengineer.

---

# 3. Backend

Backend language:

Go

Backend communication:

- REST for public HTTP APIs
- WebSocket for frontend realtime updates
- gRPC + Protobuf for internal service communication
- MQTT for IoT device communication

Backend services:

```text
backend/
  api-gateway/
  device-service/
  telemetry-service/
  automation-service/
  integration-service/
  mqtt-ingestion/
```

Each service must have a clear responsibility.

---

# 4. Go Project Structure

Prefer this structure:

```text
service/
  cmd/
    main.go

  internal/
    handlers/
    services/
    repositories/
    models/
    config/
    transport/
    middleware/

  proto/

  tests/
```

Do not place business logic in `main.go`.

`main.go` should primarily:

- load configuration
- initialize dependencies
- initialize database
- initialize clients
- construct services
- configure routes
- start the server
- handle graceful shutdown

---

# 5. Dependency Injection

Use dependency injection.

Prefer:

```go
type TelemetryService struct {
    repository TelemetryRepository
}

func NewTelemetryService(
    repository TelemetryRepository,
) *TelemetryService {
    return &TelemetryService{
        repository: repository,
    }
}
```

Avoid:

```go
var db *gorm.DB
```

Do not use global database connections, MQTT clients, configuration, or services.

---

# 6. HTTP Handlers

Handlers must remain thin.

Handlers should:

1. validate request
2. extract parameters
3. call service
4. map result to HTTP response

Business logic belongs in services.

Example:

```go
func (h *TelemetryHandler) GetLatest(c *gin.Context) {
    deviceID := c.Param("deviceId")

    telemetry, err := h.service.GetLatest(
        c.Request.Context(),
        deviceID,
    )

    if err != nil {
        handleError(c, err)
        return
    }

    c.JSON(http.StatusOK, telemetry)
}
```

Do not put database queries or complex business logic inside handlers.

---

# 7. Services

Services contain business logic.

Examples:

```text
DeviceService
TelemetryService
AutomationService
IntegrationService
PairingService
WeatherService
```

Services should depend on interfaces rather than concrete database implementations where useful.

Example:

```go
type DeviceRepository interface {
    GetByID(ctx context.Context, id string) (*Device, error)
    Create(ctx context.Context, device *Device) error
}
```

---

# 8. Repositories

Repositories are responsible only for persistence.

Repositories should:

- access PostgreSQL/TimescaleDB
- execute queries
- map database models
- return domain/application data

Repositories must not:

- perform HTTP requests
- evaluate fuzzy rules
- publish MQTT commands
- contain application workflows

Use GORM where practical.

Raw SQL is allowed when necessary for:

- TimescaleDB-specific queries
- complex aggregations
- performance-critical queries
- database-specific features

Document why raw SQL is required.

---

# 9. Database

Primary application database:

PostgreSQL

Time-series database:

TimescaleDB

Use PostgreSQL for:

- users
- devices
- greenhouses
- device certificates
- pairing sessions
- automation rules
- fuzzy configuration
- integrations
- notifications

Use TimescaleDB for:

- sensor telemetry
- weather measurements
- device events
- automation execution history
- other high-volume time-series data

Use migrations.

Never modify production schema manually when the change can be represented as a migration.

---

# 10. Database Models

Keep database models simple.

Example:

```go
type Device struct {
    ID        string    `gorm:"primaryKey"`
    Name      string
    Status    string
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

Use appropriate indexes.

Common indexed fields include:

- device_id
- timestamp
- greenhouse_id
- status
- certificate_serial

Avoid adding indexes without a reason.

---

# 11. Time-Series Data

Telemetry should be stored in TimescaleDB.

Example:

```text
telemetry

time
device_id
temperature
humidity
soil_moisture
co2
light
pressure
```

Queries should consider time-series characteristics.

Prefer:

```sql
time_bucket(...)
```

for historical aggregation where appropriate.

Use continuous aggregates and retention policies when they provide a clear operational benefit.

---

# 12. MQTT

MQTT broker:

VerneMQ

ESP32 communicates with VerneMQ using:

MQTT + TLS/mTLS

Backend services communicate with the broker using MQTT.

MQTT is the device communication protocol.

Do not use gRPC directly from ESP32.

---

# 13. MQTT Topics

Use versioned topics:

```text
agronode/v1/devices/{deviceId}/telemetry
agronode/v1/devices/{deviceId}/status
agronode/v1/devices/{deviceId}/events
agronode/v1/devices/{deviceId}/commands
agronode/v1/devices/{deviceId}/config
```

Never hardcode device IDs.

Build topics using a dedicated topic builder/helper.

Example:

```go
func TelemetryTopic(deviceID string) string {
    return fmt.Sprintf(
        "agronode/v1/devices/%s/telemetry",
        deviceID,
    )
}
```

---

# 14. MQTT Payload Validation

Never trust MQTT payloads.

Every incoming message must be:

1. decoded
2. validated
3. normalized
4. processed

Malformed messages must not crash the MQTT consumer.

Example:

```json
{
  "device_id": "esp32-lab",
  "temperature": 24.5,
  "humidity": 60,
  "timestamp": "2026-05-14T12:00:00Z"
}
```

Validate:

- device ID
- required fields
- data types
- value ranges
- timestamp
- payload size

Reject invalid messages safely.

---

# 15. MQTT Connection Management

MQTT clients must:

- reconnect automatically
- log connection events
- log disconnects
- handle broker failures
- use connection timeouts
- support graceful shutdown

Do not create an MQTT connection per message.

Use a long-lived managed client.

---

# 16. MQTT ACL

Each device must only access its own topics.

Example:

Device:

```text
device-001
```

Allowed:

```text
PUBLISH:
agronode/v1/devices/device-001/telemetry
agronode/v1/devices/device-001/status
agronode/v1/devices/device-001/events

SUBSCRIBE:
agronode/v1/devices/device-001/commands
agronode/v1/devices/device-001/config
```

A device must never access:

```text
agronode/v1/devices/device-002/...
```

Authorization must be enforced at the broker level.

---

# 17. Device Provisioning

Devices must have unique identities.

Each ESP32 should generate its own key pair.

The private key must never leave the device.

Provisioning flow:

```text
ESP32
  │
  │ generate key pair
  ▼
Pairing Service
  │
  │ challenge / nonce
  ▼
ESP32
  │
  │ signed challenge
  ▼
Pairing Service
  │
  │ verify signature
  ▼
Device CA
  │
  │ issue X.509 certificate
  ▼
ESP32
  │
  │ MQTT TLS/mTLS
  ▼
VerneMQ
```

Device lifecycle:

```text
PENDING
   ↓
PROVISIONED
   ↓
ACTIVE
   ↓
REVOKED
```

Never store device private keys on the backend.

---

# 18. Certificates

Use X.509 certificates for device authentication.

Architecture:

```text
AgroNode Root CA
      │
      ▼
AgroNode Device CA
      │
      ├── device-001.crt
      ├── device-002.crt
      └── device-003.crt
```

Root CA should be strongly protected.

Prefer short-lived device certificates with automatic renewal.

Certificate revocation must also invalidate the device's MQTT access.

---

# 19. Fuzzy Logic Automation

Automation must support fuzzy logic rather than only simple min/max thresholds.

Example:

```text
Temperature:

very_cold
cold
comfortable
warm
hot
```

Humidity:

```text
low
normal
high
```

Soil moisture:

```text
very_dry
dry
normal
wet
very_wet
```

Example rule:

```text
IF temperature IS hot
AND humidity IS high
THEN ventilation IS very_high
```

Another example:

```text
IF temperature IS warm
AND soil_moisture IS dry
AND rain_probability IS low
THEN irrigation IS medium
```

The fuzzy engine must be deterministic and testable.

Do not place fuzzy evaluation inside HTTP handlers.

---

# 20. Automation Safety

AI must not directly control safety-critical realtime automation.

The deterministic automation engine is responsible for:

- fuzzy evaluation
- thresholds
- actuator decisions
- safety limits
- execution

AI may:

- analyze telemetry
- explain decisions
- detect anomalies
- recommend configuration
- generate insights

AI recommendations must pass through normal authorization and automation validation before affecting devices.

---

# 21. Weather Integration

Weather must be implemented through the Integration Service.

Do not couple Automation Service directly to a specific weather provider.

Example:

```text
Integration Service
        │
        ├── Open-Meteo
        ├── OpenWeather
        ├── WeatherAPI
        └── Future providers
```

Normalize external APIs into an internal model.

Example:

```go
type WeatherData struct {
    Temperature     float64
    Humidity        float64
    RainProbability float64
    Precipitation   float64
    WindSpeed       float64
    UVIndex         float64
}
```

Automation should depend on normalized weather data rather than provider-specific APIs.

---

# 22. External Integrations

The Integration Service is responsible for external systems.

Possible integrations:

- weather
- email
- Telegram
- webhooks
- Home Assistant
- external REST APIs
- agricultural services

External provider-specific logic must stay inside the Integration Service.

---

# 23. gRPC

Use gRPC + Protobuf for internal service-to-service communication.

Example:

```text
API Gateway
    │
    ├── gRPC → Device Service
    ├── gRPC → Telemetry Service
    ├── gRPC → Automation Service
    └── gRPC → Integration Service
```

Use:

- deadlines
- context propagation
- typed protobuf contracts
- structured errors
- health checks

Do not use gRPC for browser APIs.

Do not use gRPC directly from ESP32.

---

# 24. REST API

REST is the public application API.

Use:

```text
GET
POST
PUT
PATCH
DELETE
```

Use consistent resource naming.

Example:

```text
GET    /api/devices
GET    /api/devices/{id}
POST   /api/devices
PATCH  /api/devices/{id}
DELETE /api/devices/{id}

GET    /api/devices/{id}/telemetry
GET    /api/devices/{id}/status
```

Return appropriate HTTP status codes.

Never expose internal stack traces to clients.

---

# 25. WebSockets

WebSockets are used for realtime frontend updates.

Flow:

```text
ESP32
  ↓
VerneMQ
  ↓
MQTT Ingestion
  ↓
Telemetry Service
  ↓
API Gateway
  ↓
WebSocket
  ↓
React
```

Do not make the frontend connect directly to VerneMQ.

---

# 26. MCP / AI

AgroNode supports an MCP server implemented in .NET.

Architecture:

```text
LLM
 │
 │ MCP
 ▼
.NET MCP Server
 │
 │ gRPC
 ▼
Go Services
```

The MCP server exposes controlled tools.

Examples:

```text
get_devices()
get_device_status(device_id)
get_latest_telemetry(device_id)
get_telemetry_history(device_id, from, to)
get_weather(location)
get_weather_forecast(location)
get_automation_rules()
get_rule_executions(device_id)
analyze_greenhouse(device_id)
detect_anomaly(device_id)
explain_automation_decision(execution_id)
```

MCP tools must enforce authorization.

Never expose:

- database credentials
- MQTT credentials
- device private keys
- certificate private keys

to the LLM.

---

# 27. AI Actions

Read-only AI tools should be preferred.

Actions such as:

```text
send_device_command()
enable_rule()
disable_rule()
```

must:

- verify authorization
- validate input
- validate device state
- enforce safety rules
- produce an audit log

Never allow arbitrary AI-generated commands to reach MQTT directly.

---

# 28. Frontend

Frontend stack:

- React
- TypeScript
- Vite
- Recharts

Structure:

```text
frontend/
  src/
    components/
    pages/
    hooks/
    api/
    types/
    charts/
    layouts/
    utils/
```

Use functional components.

Prefer hooks.

Keep components small.

Avoid prop drilling.

Use typed API clients.

Keep API communication outside presentation components.

---

# 29. Frontend Pages

The application should eventually support:

```text
Dashboard
Devices
Telemetry
Automation
Fuzzy Rules
AI Assistant
Alerts
Integrations
Pairing
Settings
```

Dashboard should support:

- greenhouse/device selection
- live telemetry
- historical charts
- weather
- automation state
- recent automation executions
- alerts
- device health
- system health

---

# 30. Charts

Use reusable chart components.

Charts should support:

- temperature
- humidity
- soil moisture
- CO2
- light
- pressure
- weather
- automation history

Do not duplicate chart configuration across pages.

---

# 31. Authentication

The architecture must be prepared for authentication.

Do not assume every API endpoint is public.

Authentication and authorization should be implemented at the API boundary.

Do not trust:

- tenant IDs
- user IDs
- device IDs
- permissions

provided by the frontend without server-side validation.

---

# 32. Security

Never commit:

- passwords
- API keys
- tokens
- private keys
- certificates containing private material

Use environment variables or secure secret management.

Validate all API input.

Validate all MQTT payloads.

Use TLS for external communication.

Use mTLS for device authentication where supported.

Do not expose internal errors.

Log security-relevant events without logging secrets.

---

# 33. Configuration

Use environment variables.

Configuration should be centralized.

Example:

```text
DATABASE_URL
MQTT_BROKER_URL
MQTT_USERNAME
MQTT_PASSWORD
MQTT_TLS_ENABLED
MQTT_CA_CERT
GRPC_DEVICE_SERVICE_URL
GRPC_TELEMETRY_SERVICE_URL
GRPC_AUTOMATION_SERVICE_URL
WEATHER_API_URL
WEATHER_API_KEY
```

Do not hardcode environment-specific values.

---

# 34. Logging

Use structured logging.

Prefer fields such as:

```text
service
device_id
request_id
trace_id
operation
duration
error
```

Never log:

- passwords
- API tokens
- private keys
- MQTT credentials
- certificate private material

Logs should be useful for debugging production failures.

---

# 35. Observability

Use OpenTelemetry.

Collect:

- metrics
- logs
- traces

Metrics should include examples such as:

```text
mqtt_messages_total
mqtt_messages_failed
device_online_total
device_offline_total
telemetry_ingestion_rate
telemetry_processing_latency
automation_executions_total
automation_failures_total
fuzzy_evaluations_total
weather_api_requests_total
weather_api_failures_total
grpc_requests_total
grpc_request_latency
mcp_requests_total
mcp_tool_execution_latency
```

Use:

```text
OpenTelemetry
    ↓
Prometheus / Loki / Tempo
    ↓
Grafana
```

---

# 36. Error Handling

Use typed/structured application errors.

Distinguish between:

- validation errors
- not found
- unauthorized
- forbidden
- conflict
- dependency failure
- internal error

Do not expose internal implementation details.

Example response:

```json
{
  "error": {
    "code": "DEVICE_NOT_FOUND",
    "message": "Device was not found"
  }
}
```

---

# 37. Testing

Business logic must be testable without infrastructure where possible.

Prioritize tests for:

- fuzzy logic
- automation rules
- MQTT payload validation
- device provisioning
- certificate validation
- repositories
- API handlers
- gRPC services
- weather normalization

Use mocks/fakes at service boundaries when useful.

Do not mock everything unnecessarily.

Prefer integration tests for:

- PostgreSQL
- TimescaleDB
- MQTT
- gRPC

when testing actual infrastructure behavior.

---

# 38. Docker

All services must be containerizable.

Use:

```text
Docker
Docker Compose
```

Docker Compose should remain readable.

Do not place application source code logic inside Docker configuration.

Use multi-stage builds for production images where appropriate.

Containers should:

- run as non-root where practical
- expose only required ports
- receive configuration through environment variables
- have health checks where appropriate

---

# 39. Local Development

Local development should be possible using Docker Compose.

Expected infrastructure:

```text
VerneMQ
PostgreSQL
TimescaleDB
Prometheus
Grafana
```

Application services should be independently startable.

---

# 40. Kubernetes

Kubernetes is a future deployment target.

Do not introduce Kubernetes-specific complexity into the application layer.

Application code should remain platform-independent.

Kubernetes configuration belongs under:

```text
infrastructure/kubernetes/
```

---

# 41. Repository Structure

Preferred repository structure:

```text
agronode/
├── backend/
│   ├── api-gateway/
│   ├── device-service/
│   ├── telemetry-service/
│   ├── automation-service/
│   ├── integration-service/
│   └── mqtt-ingestion/
│
├── mcp/
│   └── agronode-mcp/
│
├── frontend/
│   └── web/
│
├── firmware/
│   └── esp32/
│
├── proto/
│   ├── device.proto
│   ├── telemetry.proto
│   ├── automation.proto
│   └── integration.proto
│
├── database/
│   ├── postgres/
│   └── timescaledb/
│
├── infrastructure/
│   ├── vernemq/
│   ├── docker/
│   ├── kubernetes/
│   └── certificates/
│
├── observability/
│   ├── prometheus/
│   ├── grafana/
│   └── otel/
│
├── docs/
│   ├── architecture/
│   ├── mqtt/
│   ├── provisioning/
│   └── api/
│
└── README.md
```

---

# 42. API and Protocol Contracts

Keep contracts explicit.

REST:

```text
OpenAPI
```

Internal communication:

```text
Protocol Buffers
```

MQTT:

```text
Versioned topic structure
Versioned payload schemas
```

Breaking protocol changes must introduce a new version.

Avoid silently changing existing contracts.

---

# 43. Graceful Shutdown

Every backend service must support graceful shutdown.

On shutdown:

1. stop accepting new requests
2. stop background workers
3. disconnect MQTT safely
4. close database connections
5. stop gRPC servers
6. flush telemetry/logging where appropriate
7. exit cleanly

---

# 44. Background Workers

Background workers must:

- support cancellation through context.Context
- have bounded retries
- use exponential backoff where appropriate
- avoid infinite retry loops
- expose metrics
- log failures

Do not create uncontrolled goroutines.

Every goroutine must have a clear lifecycle.

---

# 45. Performance

Optimize only when necessary.

Prefer:

- database indexes
- pagination
- batching
- connection pooling
- time-series aggregation
- bounded concurrency

Avoid:

- premature caching
- unnecessary distributed systems
- unnecessary goroutines
- excessive abstraction

Measure before optimizing.

---

# 46. Scalability

Design for multiple greenhouses and devices.

Never assume:

```text
one device
one greenhouse
one user
```

Use identifiers such as:

```text
user_id
greenhouse_id
device_id
```

All device-related operations must be scoped correctly.

---

# 47. Future Features

The architecture should support future:

- multiple greenhouses
- multiple devices
- authentication
- authorization
- WebSockets
- alerting
- analytics
- OTA firmware updates
- cloud deployment
- Kubernetes
- AI assistants
- anomaly detection
- additional weather providers
- additional external integrations

Future support must not justify unnecessary abstraction today.

---

# 48. Coding Decision Rule

When implementing a feature, prefer this order:

1. simplest correct solution
2. clean separation of responsibility
3. testability
4. observability
5. security
6. scalability

Do not implement speculative infrastructure without a concrete requirement.

---

# 49. Before Writing Code

Before making a significant change:

1. inspect the existing architecture
2. reuse existing patterns
3. identify the correct service/module
4. avoid duplicating existing functionality
5. consider API and database compatibility
6. consider observability and error handling
7. add or update tests

Do not introduce a new abstraction if an existing abstraction already solves the problem.

---

# 50. Definition of Done

A feature is not considered complete until, where applicable:

- code follows the existing architecture
- business logic is in the correct service
- inputs are validated
- errors are handled
- logs are structured
- metrics/tracing are considered
- database changes have migrations
- API contracts are updated
- tests are added
- Docker/local development still works
- secrets are not introduced
- documentation is updated when architecture changes

---

# Final Principle

AgroNode should remain a practical production-style IoT platform.

Prefer:

```text
Simple
Explicit
Secure
Observable
Testable
Maintainable
```

over:

```text
Complex
Implicit
Tightly coupled
Over-engineered
```

Every architectural decision should make AgroNode easier to operate, understand, test, and extend.
