#include <WiFi.h>
#include <WiFiClientSecure.h>
#include <HTTPClient.h>
#include <Preferences.h>
#include <PubSubClient.h>
#include <DHT.h>
#include <time.h>

// Per-device secrets (WiFi, hosts, CA certificate, one-time provisioning token).
// Copy secrets.example.h to secrets.h; secrets.h is git-ignored and must never be committed.
#include "secrets.h"

#define DHT_PIN 4
#define DHT_TYPE DHT11
#define ACTIVATION_PIN 2

const char* FIRMWARE_VERSION = "1.1.0";
const char* SENSOR_MODULE_ID = "dht11";
const char* TEMPERATURE_SENSOR_ID = "dht11-temp";
const char* HUMIDITY_SENSOR_ID = "dht11-humidity";
const unsigned long PUBLISH_INTERVAL_MS = 5000;
const unsigned long ACTIVATION_SIGNAL_DURATION_MS = 5000;

const long GMT_OFFSET_SEC = 0;
const int DAYLIGHT_OFFSET_SEC = 0;
const char* NTP_SERVER = "pool.ntp.org";

// Reconnect backoff: 1s doubling up to 60s (5 min once the broker refuses our credentials).
const unsigned long MQTT_BACKOFF_MIN_MS = 1000;
const unsigned long MQTT_BACKOFF_MAX_MS = 60000;
const unsigned long MQTT_BACKOFF_REJECTED_MS = 300000;
const int REGISTER_MAX_ATTEMPTS = 3;

// Persistent credential state stored in NVS (namespace "agronode").
//   PROV_NONE        no credentials and no registration in flight: registration may run
//   PROV_REGISTERING a registration request was started; the one-time token may already be consumed
//   PROV_DONE        device id and MQTT password are stored
enum ProvisionState : uint8_t { PROV_NONE = 0, PROV_REGISTERING = 1, PROV_DONE = 2 };

// RECOVERY means: do not register again, wait for an operator (see README / docs).
enum DeviceMode { MODE_RUN, MODE_RECOVERY };

const size_t DEVICE_ID_LEN = 36;
const size_t PASSWORD_MIN_LEN = 20;
const size_t PASSWORD_MAX_LEN = 63;

DHT dht(DHT_PIN, DHT_TYPE);
WiFiClientSecure secureClient;
PubSubClient mqttClient(secureClient);
Preferences prefs;

DeviceMode deviceMode = MODE_RUN;
const char* recoveryReason = "";
unsigned long lastRecoveryNoticeMs = 0;
unsigned long mqttNextAttemptMs = 0;
unsigned long mqttBackoffMs = MQTT_BACKOFF_MIN_MS;
unsigned long lastPublishMs = 0;
unsigned long activationSignalUntilMs = 0;
bool telemetryPublishingEnabled = true;
bool temperaturePublishingEnabled = true;
bool humidityPublishingEnabled = true;
char topicBuffer[128];
char statusTopicBuffer[128];
char activationTopicBuffer[128];
char deviceIdBuffer[64];
char mqttPasswordBuffer[PASSWORD_MAX_LEN + 1];
char serialLine[160];
size_t serialLineLen = 0;

void handleActivationPayload(const String& payload);
void onMqttMessage(char* topic, byte* payload, unsigned int length);
bool payloadHasDeviceId(const String& payload, const char* expectedDeviceId);
bool payloadHasBooleanField(const String& payload, const char* fieldName, bool expectedValue);
bool payloadHasStringField(const String& payload, const char* fieldName, const char* expectedValue);
String payloadStringFieldValue(const String& payload, const char* fieldName);
bool publishDeviceStatus(bool online, unsigned long epochSeconds);

void enterRecovery(const char* reason) {
  deviceMode = MODE_RECOVERY;
  recoveryReason = reason;
  Serial.print("RECOVERY: ");
  Serial.println(reason);
}

void formatHardwareId(char* out, size_t outSize) {
  uint64_t mac = ESP.getEfuseMac();
  snprintf(out, outSize, "%02X:%02X:%02X:%02X:%02X:%02X",
    (unsigned)(mac & 0xFF), (unsigned)((mac >> 8) & 0xFF), (unsigned)((mac >> 16) & 0xFF),
    (unsigned)((mac >> 24) & 0xFF), (unsigned)((mac >> 32) & 0xFF), (unsigned)((mac >> 40) & 0xFF));
}

bool credentialsLookValid(const String& deviceId, const String& password) {
  if (deviceId.length() != DEVICE_ID_LEN) {
    return false;
  }
  for (size_t i = 0; i < deviceId.length(); i++) {
    char c = deviceId[i];
    bool hex = (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F');
    bool dash = (i == 8 || i == 13 || i == 18 || i == 23);
    if (dash ? c != '-' : !hex) {
      return false;
    }
  }
  return password.length() >= PASSWORD_MIN_LEN && password.length() <= PASSWORD_MAX_LEN;
}

bool storeCredentials(const String& deviceId, const String& password) {
  if (!credentialsLookValid(deviceId, password)) {
    return false;
  }
  bool ok = prefs.putString("devid", deviceId) == deviceId.length() &&
            prefs.putString("mqttpw", password) == password.length();
  if (ok) {
    prefs.putUChar("state", PROV_DONE);
  }
  return ok;
}

// Returns true when valid credentials were loaded into the runtime buffers.
bool loadCredentials() {
  String deviceId = prefs.getString("devid", "");
  String password = prefs.getString("mqttpw", "");
  if (!credentialsLookValid(deviceId, password)) {
    return false;
  }
  strlcpy(deviceIdBuffer, deviceId.c_str(), sizeof(deviceIdBuffer));
  strlcpy(mqttPasswordBuffer, password.c_str(), sizeof(mqttPasswordBuffer));
  return true;
}

void ensureWiFiConnected() {
  if (WiFi.status() == WL_CONNECTED) {
    return;
  }

  WiFi.mode(WIFI_STA);
  WiFi.begin(WIFI_SSID, WIFI_PASSWORD);

  Serial.print("Connecting WiFi");
  while (WiFi.status() != WL_CONNECTED) {
    delay(500);
    Serial.print(".");
  }

  Serial.println();
  Serial.print("WiFi connected, IP: ");
  Serial.println(WiFi.localIP());
}

void syncClock() {
  Serial.println("Syncing clock...");
  configTime(GMT_OFFSET_SEC, DAYLIGHT_OFFSET_SEC, NTP_SERVER);

  time_t now = time(nullptr);
  int retries = 0;

  while (now < 100000 && retries < 20) {
    delay(500);
    Serial.print(".");
    now = time(nullptr);
    retries++;
  }

  Serial.println();
  Serial.print("Epoch now: ");
  Serial.println((unsigned long)now);
}

bool clockIsValid() {
  return time(nullptr) > 1000000000;
}

// Extracts the value of a JSON string field; the registration response is a small, fixed-shape object.
String jsonStringField(const String& json, const char* field) {
  String needle = String("\"") + field + "\":\"";
  int start = json.indexOf(needle);
  if (start < 0) {
    return "";
  }
  start += needle.length();
  int end = json.indexOf('"', start);
  return end < 0 ? String("") : json.substring(start, end);
}

// One registration pass. The state marker is written BEFORE the request: if power or the response is lost
// afterwards, the next boot sees PROV_REGISTERING and enters recovery instead of burning another token.
void registerWithBackend() {
  if (strlen(PROVISIONING_TOKEN) == 0) {
    enterRecovery("no provisioning token configured (set PROVISIONING_TOKEN in secrets.h and reflash)");
    return;
  }
  if (!clockIsValid()) {
    // Without a valid clock TLS validation would fail; nothing was sent, so stay in PROV_NONE.
    enterRecovery("clock not synchronized; cannot validate TLS certificates");
    return;
  }

  char hardwareId[20];
  formatHardwareId(hardwareId, sizeof(hardwareId));

  unsigned long delayMs = 2000;
  for (int attempt = 1; attempt <= REGISTER_MAX_ATTEMPTS; attempt++) {
    prefs.putUChar("state", PROV_REGISTERING);

    WiFiClientSecure httpsClient;
    httpsClient.setCACert(ROOT_CA_PEM);
    HTTPClient http;
    http.setTimeout(15000);
    if (!http.begin(httpsClient, REGISTER_URL)) {
      prefs.putUChar("state", PROV_NONE);
      enterRecovery("registration URL rejected");
      return;
    }
    http.addHeader("Content-Type", "application/json");

    String body = String("{\"hardwareId\":\"") + hardwareId + "\",\"provisioningToken\":\"" + PROVISIONING_TOKEN + "\"}";
    int code = http.POST(body);
    String response = code > 0 ? http.getString() : String("");
    http.end();
    body = "";

    Serial.print("Registration HTTP status: ");
    Serial.println(code);

    if (code == 201) {
      String deviceId = jsonStringField(response, "username");
      String password = jsonStringField(response, "password");
      response = "";
      if (!storeCredentials(deviceId, password)) {
        // The server created the device but we could not keep its credentials: recovery, never re-register.
        enterRecovery("registration succeeded but credentials could not be stored; request credential rotation");
        return;
      }
      loadCredentials();
      Serial.println("Registration OK, credentials stored");
      return;
    }

    // -1 = connection/TLS failure before any request reached the server; 429 = rejected before the token was touched.
    if ((code == HTTPC_ERROR_CONNECTION_REFUSED || code == 429) && attempt < REGISTER_MAX_ATTEMPTS) {
      prefs.putUChar("state", PROV_NONE);
      Serial.println("Registration not delivered, retrying with backoff");
      delay(delayMs);
      delayMs *= 2;
      continue;
    }

    if (code == 400 || code == 401 || code == 403 || code == 409 || code == 429 || code == HTTPC_ERROR_CONNECTION_REFUSED) {
      // Definitive refusal without creating a device: the token was not consumed (or was already unusable).
      prefs.putUChar("state", PROV_NONE);
      enterRecovery(code == 409 ? "hardware already registered; ask an operator to rotate credentials"
                                : "registration refused; check the provisioning token and server URL");
      return;
    }

    // Anything else (5xx, timeout mid-response, ...) is ambiguous: the token may be consumed. Keep the marker.
    enterRecovery("registration outcome unknown; ask an operator to rotate credentials");
    return;
  }
}

// Serial recovery channel. Accepts "PROVISION <deviceId> <mqttPassword>" (from the credential rotation API)
// and "ERASE" (factory reset of the stored credentials). Secrets are never echoed.
void handleSerialCommand(char* line) {
  if (strcmp(line, "ERASE") == 0) {
    prefs.clear();
    Serial.println("Credentials erased. Reboot to register with a new provisioning token.");
    return;
  }
  if (strncmp(line, "PROVISION ", 10) == 0) {
    char* deviceId = line + 10;
    char* space = strchr(deviceId, ' ');
    if (space == nullptr) {
      Serial.println("Usage: PROVISION <deviceId> <mqttPassword>");
      return;
    }
    *space = '\0';
    if (storeCredentials(String(deviceId), String(space + 1))) {
      Serial.println("Credentials stored. Rebooting...");
      delay(500);
      ESP.restart();
    }
    Serial.println("Rejected: credentials malformed");
    return;
  }
  Serial.println("Unknown command");
}

void pollSerialCommands() {
  while (Serial.available() > 0) {
    char c = (char)Serial.read();
    if (c == '\r' || c == '\n') {
      if (serialLineLen > 0) {
        serialLine[serialLineLen] = '\0';
        handleSerialCommand(serialLine);
      }
      serialLineLen = 0;
      memset(serialLine, 0, sizeof(serialLine));
    } else if (serialLineLen < sizeof(serialLine) - 1) {
      serialLine[serialLineLen++] = c;
    }
  }
}

void setup() {
  Serial.begin(115200);
  delay(1000);

  Serial.print("Firmware version: ");
  Serial.println(FIRMWARE_VERSION);
  Serial.print("Sensor module: ");
  Serial.println(SENSOR_MODULE_ID);

  dht.begin();
  pinMode(ACTIVATION_PIN, OUTPUT);
  digitalWrite(ACTIVATION_PIN, LOW);

  prefs.begin("agronode", false);

  ensureWiFiConnected();
  syncClock();

  // Certificate verification stays on: the broker certificate must chain to ROOT_CA_PEM.
  secureClient.setCACert(ROOT_CA_PEM);
  mqttClient.setServer(MQTT_HOST, MQTT_PORT);
  mqttClient.setCallback(onMqttMessage);
  mqttClient.setBufferSize(512);

  uint8_t state = prefs.getUChar("state", PROV_NONE);
  if (state == PROV_DONE) {
    if (!loadCredentials()) {
      enterRecovery("stored credentials are missing or corrupted; not registering again");
    }
  } else if (state == PROV_REGISTERING) {
    enterRecovery("a previous registration did not complete; not registering again");
  } else {
    registerWithBackend();
  }

  if (deviceMode == MODE_RUN) {
    Serial.print("DEVICE_ID: ");
    Serial.println(deviceIdBuffer);
  }
}

// Non-blocking: returns true when connected. Failed attempts are spaced by bounded exponential backoff.
bool ensureMqttConnected() {
  if (mqttClient.connected()) {
    mqttBackoffMs = MQTT_BACKOFF_MIN_MS;
    return true;
  }

  unsigned long nowMs = millis();
  if ((long)(nowMs - mqttNextAttemptMs) < 0) {
    return false;
  }

  Serial.println("Connecting MQTT");
  // Username and client id are both the device id; the broker binds them together.
  if (mqttClient.connect(deviceIdBuffer, deviceIdBuffer, mqttPasswordBuffer)) {
    Serial.println("MQTT connected");
    mqttBackoffMs = MQTT_BACKOFF_MIN_MS;

    snprintf(activationTopicBuffer, sizeof(activationTopicBuffer), "agronode/%s/activation", deviceIdBuffer);
    bool subscribed = mqttClient.subscribe(activationTopicBuffer);
    Serial.print("Subscribe activation topic: ");
    Serial.println(subscribed ? "OK" : "FAILED");
    return true;
  }

  int state = mqttClient.state();
  Serial.print("MQTT connect failed, state=");
  Serial.println(state);

  // Credentials are never erased on failure: a refusal can mean revoked, disabled, rotated or a broker outage.
  bool refused = (state == MQTT_CONNECT_BAD_CREDENTIALS || state == MQTT_CONNECT_UNAUTHORIZED);
  if (refused) {
    Serial.println("Broker refused credentials (revoked, disabled or rotated?). Backing off; use the recovery procedure if it persists.");
    mqttBackoffMs = MQTT_BACKOFF_REJECTED_MS;
  } else {
    mqttBackoffMs = min(mqttBackoffMs * 2, MQTT_BACKOFF_MAX_MS);
  }
  mqttNextAttemptMs = nowMs + mqttBackoffMs + (unsigned long)(esp_random() % 1000);
  return false;
}

void onMqttMessage(char* topic, byte* payload, unsigned int length) {
  String payloadText;
  payloadText.reserve(length + 1);
  for (unsigned int index = 0; index < length; index++) {
    payloadText += (char)payload[index];
  }

  Serial.print("MQTT message topic: ");
  Serial.println(topic);
  Serial.print("MQTT message payload: ");
  Serial.println(payloadText);

  if (String(topic) != String(activationTopicBuffer)) {
    return;
  }

  handleActivationPayload(payloadText);
}

void handleActivationPayload(const String& payload) {
  if (!payloadHasDeviceId(payload, deviceIdBuffer)) {
    Serial.println("Activation payload ignored: deviceId mismatch");
    return;
  }

  String sensorTarget = payloadStringFieldValue(payload, "sensor");
  sensorTarget.trim();
  if (sensorTarget.length() == 0) {
    sensorTarget = "all";
  }

  bool isGlobalTarget =
    sensorTarget.equalsIgnoreCase("all") ||
    sensorTarget.equalsIgnoreCase("telemetry");
  bool isTemperatureTarget =
    sensorTarget.equalsIgnoreCase(TEMPERATURE_SENSOR_ID) ||
    sensorTarget.equalsIgnoreCase("temperature");
  bool isHumidityTarget =
    sensorTarget.equalsIgnoreCase(HUMIDITY_SENSOR_ID) ||
    sensorTarget.equalsIgnoreCase("humidity") ||
    sensorTarget.equalsIgnoreCase("humidity_dht11");

  if (payloadHasStringField(payload, "trigger", "stream_pause")) {
    if (isGlobalTarget) {
      telemetryPublishingEnabled = false;
      temperaturePublishingEnabled = false;
      humidityPublishingEnabled = false;
      Serial.println("Telemetry publishing paused (all sensors)");
      return;
    }

    if (isTemperatureTarget) {
      temperaturePublishingEnabled = false;
      Serial.println("Telemetry publishing paused (temperature)");
      return;
    }

    if (isHumidityTarget) {
      humidityPublishingEnabled = false;
      Serial.println("Telemetry publishing paused (humidity)");
      return;
    }

    Serial.print("Telemetry pause ignored for unknown sensor target: ");
    Serial.println(sensorTarget);
    return;
  }

  if (payloadHasStringField(payload, "trigger", "stream_resume")) {
    if (isGlobalTarget) {
      telemetryPublishingEnabled = true;
      temperaturePublishingEnabled = true;
      humidityPublishingEnabled = true;
      Serial.println("Telemetry publishing resumed (all sensors)");
      return;
    }

    if (isTemperatureTarget) {
      temperaturePublishingEnabled = true;
      Serial.println("Telemetry publishing resumed (temperature)");
      return;
    }

    if (isHumidityTarget) {
      humidityPublishingEnabled = true;
      Serial.println("Telemetry publishing resumed (humidity)");
      return;
    }

    Serial.print("Telemetry resume ignored for unknown sensor target: ");
    Serial.println(sensorTarget);
    return;
  }

  bool activated = payloadHasBooleanField(payload, "activated", true);
  if (activated) {
    activationSignalUntilMs = millis() + ACTIVATION_SIGNAL_DURATION_MS;
    digitalWrite(ACTIVATION_PIN, HIGH);

    Serial.print("ACTIVATION ON, pin=");
    Serial.print(ACTIVATION_PIN);
    Serial.print(", durationMs=");
    Serial.println(ACTIVATION_SIGNAL_DURATION_MS);
    return;
  }

  if (payloadHasBooleanField(payload, "activated", false)) {
    activationSignalUntilMs = 0;
    digitalWrite(ACTIVATION_PIN, LOW);
    Serial.println("ACTIVATION OFF");
    return;
  }

  Serial.println("Activation payload ignored: missing activated flag");
}

bool payloadHasDeviceId(const String& payload, const char* expectedDeviceId) {
  String compactPattern = String("\"deviceId\":\"") + expectedDeviceId + "\"";
  String spacedPattern = String("\"deviceId\": \"") + expectedDeviceId + "\"";

  return payload.indexOf(compactPattern) >= 0 || payload.indexOf(spacedPattern) >= 0;
}

bool payloadHasBooleanField(const String& payload, const char* fieldName, bool expectedValue) {
  String keyPattern = String("\"") + fieldName + "\"";
  int keyIndex = payload.indexOf(keyPattern);
  if (keyIndex < 0) {
    return false;
  }

  int valueIndex = keyIndex + keyPattern.length();
  while (valueIndex < payload.length() && (payload[valueIndex] == ' ' || payload[valueIndex] == ':')) {
    valueIndex++;
  }

  if (expectedValue) {
    return payload.startsWith("true", valueIndex);
  }

  return payload.startsWith("false", valueIndex);
}

bool payloadHasStringField(const String& payload, const char* fieldName, const char* expectedValue) {
  String compactPattern = String("\"") + fieldName + "\":\"" + expectedValue + "\"";
  String spacedPattern = String("\"") + fieldName + "\": \"" + expectedValue + "\"";

  return payload.indexOf(compactPattern) >= 0 || payload.indexOf(spacedPattern) >= 0;
}

String payloadStringFieldValue(const String& payload, const char* fieldName) {
  String keyPattern = String("\"") + fieldName + "\"";
  int keyIndex = payload.indexOf(keyPattern);
  if (keyIndex < 0) {
    return "";
  }

  int colonIndex = payload.indexOf(':', keyIndex + keyPattern.length());
  if (colonIndex < 0) {
    return "";
  }

  int quoteStart = payload.indexOf('"', colonIndex + 1);
  if (quoteStart < 0) {
    return "";
  }

  int quoteEnd = payload.indexOf('"', quoteStart + 1);
  if (quoteEnd < 0) {
    return "";
  }

  return payload.substring(quoteStart + 1, quoteEnd);
}

bool publishSensorTelemetry(const char* sensorID, float sensorValue, unsigned long epochSeconds) {
  char payload[448];
  int rssi = (int)WiFi.RSSI();
  unsigned long uptimeSeconds = millis() / 1000UL;
  int length = snprintf(
    payload,
    sizeof(payload),
    "{\"deviceId\":\"%s\",\"sensorId\":\"%s\",\"timestamp\":%lu,\"version\":1,"
    "\"meta\":{\"fw\":\"%s\",\"ip\":\"%s\",\"rssi\":%d,\"uptime\":%lu},"
    "\"sensors\":{\"%s\":%.2f}}",
    deviceIdBuffer,
    sensorID,
    epochSeconds,
    FIRMWARE_VERSION,
    WiFi.localIP().toString().c_str(),
    rssi,
    uptimeSeconds,
    sensorID,
    sensorValue
  );

  if (length <= 0 || length >= (int)sizeof(payload)) {
    Serial.println("Payload build failed");
    return false;
  }

  Serial.print("Telemetry payload length: ");
  Serial.println(length);

  snprintf(topicBuffer, sizeof(topicBuffer), "agronode/%s/telemetry", deviceIdBuffer);

  Serial.print("Telemetry SENSOR_ID: ");
  Serial.println(sensorID);
  Serial.print("Telemetry firmware: ");
  Serial.println(FIRMWARE_VERSION);
  Serial.print("Telemetry value: ");
  Serial.println(sensorValue, 2);
  Serial.print("Topic: ");
  Serial.println(topicBuffer);
  Serial.print("Payload: ");
  Serial.println(payload);

  bool ok = mqttClient.publish(topicBuffer, payload);

  Serial.print("Publish: ");
  Serial.println(ok ? "OK" : "FAILED");
  if (!ok) {
    Serial.print("MQTT state on publish fail: ");
    Serial.println(mqttClient.state());
  }

  return ok;
}

bool publishDeviceStatus(bool online, unsigned long epochSeconds) {
  char payload[384];
  int rssi = (int)WiFi.RSSI();
  unsigned long uptimeSeconds = millis() / 1000UL;
  int length = snprintf(
    payload,
    sizeof(payload),
    "{\"deviceId\":\"%s\",\"timestamp\":%lu,\"online\":%s,\"signal_strength\":%d,"
    "\"meta\":{\"fw\":\"%s\",\"ip\":\"%s\",\"uptime\":%lu}}",
    deviceIdBuffer,
    epochSeconds,
    online ? "true" : "false",
    rssi,
    FIRMWARE_VERSION,
    WiFi.localIP().toString().c_str(),
    uptimeSeconds
  );

  if (length <= 0 || length >= (int)sizeof(payload)) {
    Serial.println("Status payload build failed");
    return false;
  }

  snprintf(statusTopicBuffer, sizeof(statusTopicBuffer), "agronode/%s/status", deviceIdBuffer);

  Serial.print("Status topic: ");
  Serial.println(statusTopicBuffer);
  Serial.print("Status payload: ");
  Serial.println(payload);

  bool ok = mqttClient.publish(statusTopicBuffer, payload);
  Serial.print("Status publish: ");
  Serial.println(ok ? "OK" : "FAILED");
  if (!ok) {
    Serial.print("MQTT state on status publish fail: ");
    Serial.println(mqttClient.state());
  }

  return ok;
}

unsigned long currentEpochSeconds() {
  time_t now = time(nullptr);
  if (now < 1000000000) {
    return 0;
  }

  return (unsigned long)now;
}

void loop() {
  pollSerialCommands();

  if (deviceMode == MODE_RECOVERY) {
    if (lastRecoveryNoticeMs == 0 || millis() - lastRecoveryNoticeMs > 30000) {
      lastRecoveryNoticeMs = millis();
      Serial.print("RECOVERY MODE: ");
      Serial.println(recoveryReason);
      Serial.println("Commands: PROVISION <deviceId> <mqttPassword> | ERASE");
    }
    delay(200);
    return;
  }

  ensureWiFiConnected();
  if (!ensureMqttConnected()) {
    delay(100);
    return;
  }
  mqttClient.loop();

  if (activationSignalUntilMs > 0 && millis() >= activationSignalUntilMs) {
    activationSignalUntilMs = 0;
    digitalWrite(ACTIVATION_PIN, LOW);
    Serial.println("ACTIVATION AUTO-OFF");
  }

  unsigned long nowMs = millis();
  if (nowMs - lastPublishMs < PUBLISH_INTERVAL_MS) {
    delay(100);
    return;
  }

  lastPublishMs = nowMs;

  unsigned long epochSeconds = currentEpochSeconds();
  if (!telemetryPublishingEnabled || (!temperaturePublishingEnabled && !humidityPublishingEnabled)) {
    bool statusPublished = publishDeviceStatus(true, epochSeconds);
    if (!statusPublished) {
      Serial.println("Status publish failed while telemetry paused");
    }

    delay(100);
    return;
  }

  float humidity = dht.readHumidity();
  float temperature = dht.readTemperature();

  if (isnan(temperature) || isnan(humidity)) {
    Serial.println("DHT read failed");
    Serial.println("Check wiring, GPIO pin, power, and 10k pull-up");
    delay(2000);
    return;
  }

  Serial.print("Temperature: ");
  Serial.println(temperature);
  Serial.print("Humidity: ");
  Serial.println(humidity);

  bool temperaturePublished = true;
  if (temperaturePublishingEnabled) {
    temperaturePublished = publishSensorTelemetry(TEMPERATURE_SENSOR_ID, temperature, epochSeconds);
    delay(40);
  }

  bool humidityPublished = true;
  if (humidityPublishingEnabled) {
    humidityPublished = publishSensorTelemetry(HUMIDITY_SENSOR_ID, humidity, epochSeconds);
    delay(40);
  }

  bool statusPublished = publishDeviceStatus(true, epochSeconds);

  if (!temperaturePublished || !humidityPublished || !statusPublished) {
    Serial.println("One or more publishes failed");
  }
}

void testDHT() {
  Serial.println("DHT test start");
  dht.begin();
  delay(2000);

  float h = dht.readHumidity();
  float t = dht.readTemperature();

  if (isnan(t) || isnan(h)) {
    Serial.println("DHT read failed");
  } else {
    Serial.print("Temp: ");
    Serial.println(t);
    Serial.print("Hum: ");
    Serial.println(h);
  }

  delay(500);
}
