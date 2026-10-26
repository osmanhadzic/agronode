# Secure Device Provisioning (Phase 2)

## Lifecycle
1. Admin issues a one-time token: `POST /api/devices/provisioning-tokens` (session-authenticated; org users are scoped to their org).
   Body: `{"organizationId": "...", "assetUnitId": "...optional", "ttlMinutes": 60}`. The plaintext token is shown once; only its SHA-256 hash is stored.
2. ESP32 calls `POST /api/v1/devices/register` over HTTPS with `{"hardwareId","provisioningToken"}`.
   The token is consumed atomically and the device is created (`lifecycle_status=active`) with a server UUID `deviceId`, MQTT username/clientId (= deviceId) and a random password (bcrypt-hashed server-side).
   The 201 response contains the password once. Codes: 400 validation, 401 invalid/expired/used token, 403 non-HTTPS, 409 hardware already registered, 429 rate limit.
3. Firmware stores credentials in NVS and connects to VerneMQ over TLS (8883). Later boots/reconnects never call register.

## Lost response / lost credentials
Registration is never repeated automatically. The firmware marks NVS `REGISTERING` before the call; an ambiguous outcome puts it in RECOVERY mode.
An admin recovers it with `POST /api/devices/:deviceId/credentials/rotate` (returns a new password once), then on the device serial console: `PROVISION <deviceId> <password>`; `ERASE` clears NVS.
Legacy devices (no credentials) use the same rotate workflow.

## Disable / revoke
`PATCH /api/devices/:deviceId/lifecycle` with `{"status":"disabled|revoked|active"}` (revoked is terminal).
New connects/publishes are denied immediately (no auth cache). Existing sessions persist until disconnect: `vmq-admin session disconnect client-id=<deviceId>`.

## Broker auth and ACL
VerneMQ 1.12.3 `vmq_webhooks` call the backend internal listener (`VERNEMQ_WEBHOOK_ADDR`): `/vernemq/auth_on_register`, `auth_on_publish`, `auth_on_subscribe`. `vmq_acl` and anonymous access are off in `docker-compose.secure.yml`. Failures deny (fail closed). MQTT 5 is denied (no `_m5` hooks).

| Principal | Publish | Subscribe |
|---|---|---|
| device | `agronode/{ownDeviceId}/telemetry`, `/status` | `agronode/{ownDeviceId}/activation` (no wildcards) |
| backend account | `agronode/...` | `agronode/...` |

Ownership/organization are derived from the DB row, never from topic or payload.

## Configuration
See `.env.example`: `MQTT_USERNAME`, `MQTT_PASSWORD`, `MQTT_CA_FILE`, `VERNEMQ_WEBHOOK_ADDR`, `DEVICE_REGISTRATION_REQUIRE_HTTPS`, `DEVICE_REGISTRATION_RATE_LIMIT_PER_MIN`.
Put broker `ca.crt`, `server.crt`, `server.key` in `infra/vernemq/certs/` (git-ignored).
Run: `docker compose -f docker-compose.yml -f docker-compose.secure.yml up -d`.

## Migration
Migration 000025 is applied automatically at backend start (embedded list in `internal/database/migrate.go`). Manual rollback SQL: `000025_secure_device_provisioning.down.sql`. Legacy devices become `pending` with no credentials.

## ESP32
Copy `firmware/esp32_dht22_publisher/secrets.example.h` to `secrets.h` (git-ignored), fill in WiFi, host, `REGISTER_URL`, CA PEM and `PROVISIONING_TOKEN`; flash. Reconnects use bounded backoff (1s to 60s, 5 min if credentials are refused) and never erase credentials. After registration, remove the token from `secrets.h` and reflash.
Compile check: `arduino-cli compile --fqbn esp32:esp32:esp32 <sketch dir>` (needs a `secrets.h`).

## Tests
`cd backend && go test ./...`
