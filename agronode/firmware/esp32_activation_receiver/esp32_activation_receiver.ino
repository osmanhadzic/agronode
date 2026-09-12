#include <WiFi.h>
#include <PubSubClient.h>
#include <time.h>

#define ACTIVATION_PIN 2

const char* WIFI_SSID = "hadzic";
const char* WIFI_PASSWORD = "techno123";

const char* MQTT_HOST = "192.168.193.106";
const uint16_t MQTT_PORT = 1883;

const char* DEVICE_ID = "pump-node-1";
const char* FIRMWARE_VERSION = "1.0.0";
const unsigned long ACTIVATION_SIGNAL_DURATION_MS = 5000;

const long GMT_OFFSET_SEC = 0;
const int DAYLIGHT_OFFSET_SEC = 0;
const char* NTP_SERVER = "pool.ntp.org";

WiFiClient wifiClient;
PubSubClient mqttClient(wifiClient);

unsigned long activationSignalUntilMs = 0;
char activationTopicBuffer[128];
char registerTopicBuffer[128];

void ensureWiFiConnected();
void ensureMqttConnected();
void syncClock();
void registerDevice();
void onMqttMessage(char* topic, byte* payload, unsigned int length);
void handleActivationPayload(const String& payload);
bool payloadHasDeviceId(const String& payload, const char* expectedDeviceId);
bool payloadHasBooleanField(const String& payload, const char* fieldName, bool expectedValue);

void setup() {
  Serial.begin(115200);
  delay(1000);

  pinMode(ACTIVATION_PIN, OUTPUT);
  digitalWrite(ACTIVATION_PIN, LOW);

  mqttClient.setServer(MQTT_HOST, MQTT_PORT);
  mqttClient.setCallback(onMqttMessage);

  snprintf(activationTopicBuffer, sizeof(activationTopicBuffer), "agronode/%s/activation", DEVICE_ID);
  snprintf(registerTopicBuffer, sizeof(registerTopicBuffer), "agronode/%s/register", DEVICE_ID);

  ensureWiFiConnected();
  ensureMqttConnected();
  syncClock();
  registerDevice();
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

void ensureMqttConnected() {
  if (mqttClient.connected()) {
    return;
  }

  Serial.print("Connecting MQTT");
  while (!mqttClient.connected()) {
    if (mqttClient.connect(DEVICE_ID)) {
      Serial.println(" connected");
      bool subscribed = mqttClient.subscribe(activationTopicBuffer);
      Serial.print("Subscribe activation topic: ");
      Serial.print(activationTopicBuffer);
      Serial.print(" -> ");
      Serial.println(subscribed ? "OK" : "FAILED");
      return;
    }

    Serial.print(".");
    Serial.print(" state=");
    Serial.println(mqttClient.state());
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
    DEVICE_ID,
    FIRMWARE_VERSION,
    rssi,
    WiFi.localIP().toString().c_str()
  );

  mqttClient.loop();
  bool ok = mqttClient.publish(registerTopicBuffer, payload);
  Serial.print("Device registration: ");
  Serial.println(ok ? "SENT" : "FAILED");
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

  Serial.print("MQTT activation payload: ");
  Serial.println(payloadText);
  handleActivationPayload(payloadText);
}

void handleActivationPayload(const String& payload) {
  if (!payloadHasDeviceId(payload, DEVICE_ID)) {
    Serial.println("Activation payload ignored: deviceId mismatch");
    return;
  }

  if (payloadHasBooleanField(payload, "activated", true)) {
    activationSignalUntilMs = millis() + ACTIVATION_SIGNAL_DURATION_MS;
    digitalWrite(ACTIVATION_PIN, HIGH);
    Serial.println("ACTIVATION ON");
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

void loop() {
  ensureWiFiConnected();
  ensureMqttConnected();
  mqttClient.loop();

  if (activationSignalUntilMs > 0 && millis() >= activationSignalUntilMs) {
    activationSignalUntilMs = 0;
    digitalWrite(ACTIVATION_PIN, LOW);
    Serial.println("ACTIVATION AUTO-OFF");
  }

  delay(20);
}
