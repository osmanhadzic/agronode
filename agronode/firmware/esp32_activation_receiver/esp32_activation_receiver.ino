#include <WiFi.h>
#include <PubSubClient.h>
#include <time.h>
#include <stdarg.h>

#define ACTIVATION_PIN 2

const char* WIFI_SSID = "hadzic";
const char* WIFI_PASSWORD = "techno123";

const char* MQTT_HOST = "192.168.193.106";
const uint16_t MQTT_PORT = 1883;

const char* DEVICE_ID_BASE = "pump-node";
const char* FIRMWARE_VERSION = "1.0.0";
const unsigned long ACTIVATION_SIGNAL_DURATION_MS = 5000;
const unsigned long ACTIVATION_BLINK_INTERVAL_MS = 250;
const unsigned long HEARTBEAT_INTERVAL_MS = 10000;

const long GMT_OFFSET_SEC = 0;
const int DAYLIGHT_OFFSET_SEC = 0;
const char* NTP_SERVER = "pool.ntp.org";

WiFiClient wifiClient;
PubSubClient mqttClient(wifiClient);

unsigned long activationSignalUntilMs = 0;
unsigned long activationNextBlinkMs = 0;
bool activationBlinking = false;
bool activationOutputState = false;
unsigned long nextHeartbeatMs = 0;
char activationTopicBuffer[128];
char registerTopicBuffer[128];
char deviceIdBuffer[64];

void buildRuntimeDeviceId() {
  uint64_t chipId = ESP.getEfuseMac();
  uint32_t suffix = (uint32_t)(chipId & 0xFFFFFF);
  snprintf(deviceIdBuffer, sizeof(deviceIdBuffer), "%s-%06lX", DEVICE_ID_BASE, (unsigned long)suffix);
}

void ensureWiFiConnected();
void ensureMqttConnected();
void syncClock();
void registerDevice();
void onMqttMessage(char* topic, byte* payload, unsigned int length);
void handleActivationPayload(const String& payload);
bool payloadHasDeviceId(const String& payload, const char* expectedDeviceId);
bool payloadHasBooleanField(const String& payload, const char* fieldName, bool expectedValue);
void logInfo(const char* message);
void logf(const char* level, const char* format, ...);
void printHeartbeat();

void logInfo(const char* message) {
  logf("INFO", "%s", message);
}

void logf(const char* level, const char* format, ...) {
  char message[256];
  va_list args;
  va_start(args, format);
  vsnprintf(message, sizeof(message), format, args);
  va_end(args);

  Serial.print("[");
  Serial.print(millis());
  Serial.print("ms] ");
  Serial.print(level);
  Serial.print(" ");
  Serial.println(message);
}

void printHeartbeat() {
  logf(
    "HEARTBEAT",
    "wifi=%s mqtt=%s device=%s topic=%s led=%s",
    WiFi.status() == WL_CONNECTED ? "OK" : "DOWN",
    mqttClient.connected() ? "OK" : "DOWN",
    deviceIdBuffer,
    activationTopicBuffer,
    activationOutputState ? "ON" : "OFF"
  );
}

void setup() {
  Serial.begin(115200);
  delay(1000);

  buildRuntimeDeviceId();
  logf("BOOT", "Runtime DEVICE_ID=%s", deviceIdBuffer);

  pinMode(ACTIVATION_PIN, OUTPUT);
  digitalWrite(ACTIVATION_PIN, LOW);

  mqttClient.setServer(MQTT_HOST, MQTT_PORT);
  mqttClient.setCallback(onMqttMessage);

  snprintf(activationTopicBuffer, sizeof(activationTopicBuffer), "agronode/%s/activation", deviceIdBuffer);
  snprintf(registerTopicBuffer, sizeof(registerTopicBuffer), "agronode/%s/register", deviceIdBuffer);

  ensureWiFiConnected();
  ensureMqttConnected();
  syncClock();
  registerDevice();
  nextHeartbeatMs = millis() + HEARTBEAT_INTERVAL_MS;
  printHeartbeat();
}

void ensureWiFiConnected() {
  if (WiFi.status() == WL_CONNECTED) {
    return;
  }

  WiFi.mode(WIFI_STA);
  WiFi.begin(WIFI_SSID, WIFI_PASSWORD);

  logf("WIFI", "Connecting to SSID=%s", WIFI_SSID);
  while (WiFi.status() != WL_CONNECTED) {
    delay(500);
    Serial.print('.');
  }

  Serial.println();
  logf("WIFI", "Connected IP=%s RSSI=%d", WiFi.localIP().toString().c_str(), (int)WiFi.RSSI());
}

void ensureMqttConnected() {
  if (mqttClient.connected()) {
    return;
  }

  logf("MQTT", "Connecting host=%s:%u client=%s", MQTT_HOST, MQTT_PORT, deviceIdBuffer);
  while (!mqttClient.connected()) {
    if (mqttClient.connect(deviceIdBuffer)) {
      logInfo("MQTT connected");
      bool subscribed = mqttClient.subscribe(activationTopicBuffer);
      logf("MQTT", "Subscribe topic=%s status=%s", activationTopicBuffer, subscribed ? "OK" : "FAILED");
      return;
    }

    logf("MQTT", "connect retry state=%d", mqttClient.state());
    delay(1000);
  }
}

void syncClock() {
  configTime(GMT_OFFSET_SEC, DAYLIGHT_OFFSET_SEC, NTP_SERVER);
}

void registerDevice() {
  if (!mqttClient.connected()) {
    ensureMqttConnected();
  }

  char payload[512];
  int rssi = (int)WiFi.RSSI();
  snprintf(
    payload,
    sizeof(payload),
    "{\"deviceId\":\"%s\",\"firmware\":\"%s\","
    "\"metadata\":{\"signalStrength\":%d,\"hardware\":{\"model\":\"ESP32\",\"source\":\"firmware-register\",\"ip\":\"%s\"}},"
    "\"tags\":[\"live\",\"esp32\",\"actuator\"]}",
    deviceIdBuffer,
    FIRMWARE_VERSION,
    rssi,
    WiFi.localIP().toString().c_str()
  );

  mqttClient.loop();
  bool ok = mqttClient.publish(registerTopicBuffer, payload);
  logf("REGISTER", "topic=%s status=%s", registerTopicBuffer, ok ? "SENT" : "FAILED");
}

void onMqttMessage(char* topic, byte* payload, unsigned int length) {
  String payloadText;
  payloadText.reserve(length + 1);
  for (unsigned int index = 0; index < length; index++) {
    payloadText += (char)payload[index];
  }

  if (String(topic) != String(activationTopicBuffer)) {
    return;
  }

  logf("MQTT", "RX topic=%s payload=%s", topic, payloadText.c_str());
  handleActivationPayload(payloadText);
}

void handleActivationPayload(const String& payload) {
  if (!payloadHasDeviceId(payload, deviceIdBuffer)) {
    logInfo("Activation ignored: deviceId mismatch");
    return;
  }

  if (payloadHasBooleanField(payload, "activated", true)) {
    activationSignalUntilMs = millis() + ACTIVATION_SIGNAL_DURATION_MS;
    activationNextBlinkMs = millis() + ACTIVATION_BLINK_INTERVAL_MS;
    activationBlinking = true;
    activationOutputState = true;
    digitalWrite(ACTIVATION_PIN, HIGH);
    logf("TRIGGER", "ACTIVATION ON (BLINKING) durationMs=%lu", ACTIVATION_SIGNAL_DURATION_MS);
    return;
  }

  if (payloadHasBooleanField(payload, "activated", false)) {
    activationSignalUntilMs = 0;
    activationNextBlinkMs = 0;
    activationBlinking = false;
    activationOutputState = false;
    digitalWrite(ACTIVATION_PIN, LOW);
    logInfo("TRIGGER ACTIVATION OFF");
    return;
  }

  logInfo("Activation ignored: missing activated flag");
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

void loop() {
  ensureWiFiConnected();
  ensureMqttConnected();
  mqttClient.loop();

  if (activationBlinking && activationSignalUntilMs > 0) {
    unsigned long nowMs = millis();

    if (nowMs >= activationSignalUntilMs) {
      activationSignalUntilMs = 0;
      activationNextBlinkMs = 0;
      activationBlinking = false;
      activationOutputState = false;
      digitalWrite(ACTIVATION_PIN, LOW);
      logInfo("TRIGGER ACTIVATION AUTO-OFF");
    } else if (nowMs >= activationNextBlinkMs) {
      activationOutputState = !activationOutputState;
      digitalWrite(ACTIVATION_PIN, activationOutputState ? HIGH : LOW);
      activationNextBlinkMs = nowMs + ACTIVATION_BLINK_INTERVAL_MS;
    }
  }

  if (millis() >= nextHeartbeatMs) {
    printHeartbeat();
    nextHeartbeatMs = millis() + HEARTBEAT_INTERVAL_MS;
  }

  delay(20);
}
