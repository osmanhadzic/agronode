#pragma once
// Copy to secrets.h (git-ignored) and fill in. Never commit real values.

#define WIFI_SSID "your-wifi-ssid"
#define WIFI_PASSWORD "your-wifi-password"

// Broker TLS endpoint (VerneMQ listener.ssl, port 8883).
#define MQTT_HOST "mqtt.example.com"
#define MQTT_PORT 8883

// HTTPS registration endpoint (must be served over TLS).
#define REGISTER_URL "https://api.example.com/api/v1/devices/register"

// One-time token from POST /api/devices/provisioning-tokens. It is consumed on first successful
// registration and is useless afterwards. Leave empty to forbid registration on this build.
#define PROVISIONING_TOKEN ""

// PEM of the CA that signed BOTH the API and broker certificates. Certificate verification is always on.
static const char ROOT_CA_PEM[] PROGMEM = R"PEM(
-----BEGIN CERTIFICATE-----
REPLACE_WITH_YOUR_CA_CERTIFICATE
-----END CERTIFICATE-----
)PEM";
