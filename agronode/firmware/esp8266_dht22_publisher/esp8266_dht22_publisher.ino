#include <ESP8266WiFi.h>
#include <PubSubClient.h>
#include <DHT.h>
#include <time.h>

#define DHT_PIN 4               // D2 on NodeMCU
#define DHT_TYPE DHT22
#define ACTIVATION_PIN 5        // D1 on NodeMCU

const char* WIFI_SSID = "CHANGE_ME";
const char* WIFI_PASSWORD = "CHANGE_ME";

const char* MQTT_HOST = "192.168.1.100";
const uint16_t MQTT_PORT = 1883;

const char* DEVICE_ID_BASE = "esp8266-greenhouse";
const char* FIRMWARE_VERSION = "1.0.0";
const char* SENSOR_MODULE_ID = "dht22";
const char* TEMPERATURE_SENSOR_ID = "dht22-temp";
const char* HUMIDITY_SENSOR_ID = "dht22-humidity";

const unsigned long PUBLISH_INTERVAL_MS = 10000;
const unsigned long STATUS_INTERVAL_MS = 60000;
const unsigned long ACTIVATION_SIGNAL_DURATION_MS = 5000;

const long GMT_OFFSET_SEC = 0;
const int DAYLIGHT_OFFSET_SEC = 0;
const char* NTP_SERVER = "pool.ntp.org";

DHT dht(DHT_PIN, DHT_TYPE);
WiFiClient wifiClient;
PubSubClient mqttClient(wifiClient);

char deviceIdBuffer[64];
char telemetryTopicBuffer[128];
char statusTopicBuffer[128];
char registerTopicBuffer[128];
char activationTopicBuffer[128];

bool telemetryPublishingEnabled = true;
bool temperaturePublishingEnabled = true;
bool humidityPublishingEnabled = true;

unsigned long lastPublishMs = 0;
unsigned long lastStatusMs = 0;
unsigned long activationSignalUntilMs = 0;

void buildRuntimeDeviceId() {
  uint32_t chipId = ESP.getChipId();
  snprintf(deviceIdBuffer, sizeof(deviceIdBuffer), "%s-%06X", DEVICE_ID_BASE, chipId);
}

void buildTopics() {
  snprintf(telemetryTopicBuffer, sizeof(telemetryTopicBuffer), "agronode/%s/telemetry", deviceIdBuffer);
  snprintf(statusTopicBuffer, sizeof(statusTopicBuffer), "agronode/%s/status", deviceIdBuffer);
  snprintf(registerTopicBuffer, sizeof(registerTopicBuffer), "agronode/%s/register", deviceIdBuffer);
  snprintf(activationTopicBuffer, sizeof(activationTopicBuffer), "agronode/%s/activation", deviceIdBuffer);
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

  return expectedValue ? payload.startsWith("true", valueIndex) : payload.startsWith("false", valueIndex);
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

  int firstQuoteIndex = payload.indexOf('"', colonIndex + 1);
  if (firstQuoteIndex < 0) {
    return "";
  }

  int secondQuoteIndex = payload.indexOf('"', firstQuoteIndex + 1);
  if (secondQuoteIndex < 0) {
    return "";
  }

  return payload.substring(firstQuoteIndex + 1, secondQuoteIndex);
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

bool publishDeviceStatus(bool online) {
  if (!mqttClient.connected()) {
    return false;
  }

  char payload[320];
  unsigned long uptimeSeconds = millis() / 1000UL;
  int rssi = (int)WiFi.RSSI();

  snprintf(
    payload,
    sizeof(payload),
    "{\"deviceId\":\"%s\",\"online\":%s,\"signal_strength\":%d,\"uptime\":%lu,\"firmware\":\"%s\"}",
    deviceIdBuffer,
    online ? "true" : "false",
    rssi,
    uptimeSeconds,
    FIRMWARE_VERSION
  );

  return mqttClient.publish(statusTopicBuffer, payload, true);
}

void ensureMqttConnected() {
  if (mqttClient.connected()) {
    return;
  }

  Serial.print("Connecting MQTT");
  while (!mqttClient.connected()) {
    bool connected = mqttClient.connect(deviceIdBuffer);

    if (connected) {
      Serial.println(" connected");

      bool subscribed = mqttClient.subscribe(activationTopicBuffer);
      Serial.print("Subscribe activation topic: ");
      Serial.print(activationTopicBuffer);
      Serial.print(" -> ");
      Serial.println(subscribed ? "OK" : "FAILED");

      publishDeviceStatus(true);
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

unsigned long currentEpochSeconds() {
  time_t now = time(nullptr);
  if (now <= 0) {
    return millis() / 1000UL;
  }

  return (unsigned long)now;
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
    "\"metadata\":{\"signalStrength\":%d,\"hardware\":{\"model\":\"ESP8266\",\"module\":\"%s\",\"ip\":\"%s\"}},"
    "\"tags\":[\"live\",\"esp8266\",\"publisher\"]}",
    deviceIdBuffer,
    FIRMWARE_VERSION,
    rssi,
    SENSOR_MODULE_ID,
    WiFi.localIP().toString().c_str()
  );

  bool ok = mqttClient.publish(registerTopicBuffer, payload);
  Serial.print("Device registration: ");
  Serial.println(ok ? "SENT" : "FAILED");
}

void publishTelemetry() {
  if (!telemetryPublishingEnabled) {
    return;
  }

  float temperature = dht.readTemperature();
  float humidity = dht.readHumidity();

  if (isnan(temperature) || isnan(humidity)) {
    Serial.println("DHT22 read failed");
    return;
  }

  unsigned long epochSeconds = currentEpochSeconds();
  char payload[320];

  if (temperaturePublishingEnabled) {
    snprintf(
      payload,
      sizeof(payload),
      "{\"deviceId\":\"%s\",\"sensorId\":\"%s\",\"timestamp\":%lu,\"sensors\":{\"%s\":%.2f}}",
      deviceIdBuffer,
      TEMPERATURE_SENSOR_ID,
      epochSeconds,
      TEMPERATURE_SENSOR_ID,
      temperature
    );

    mqttClient.publish(telemetryTopicBuffer, payload);
  }

  if (humidityPublishingEnabled) {
    snprintf(
      payload,
      sizeof(payload),
      "{\"deviceId\":\"%s\",\"sensorId\":\"%s\",\"timestamp\":%lu,\"sensors\":{\"%s\":%.2f}}",
      deviceIdBuffer,
      HUMIDITY_SENSOR_ID,
      epochSeconds,
      HUMIDITY_SENSOR_ID,
      humidity
    );

    mqttClient.publish(telemetryTopicBuffer, payload);
  }

  Serial.print("Telemetry sent, temp=");
  Serial.print(temperature);
  Serial.print("C humidity=");
  Serial.print(humidity);
  Serial.println("%");
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

  bool isGlobalTarget = sensorTarget.equalsIgnoreCase("all") || sensorTarget.equalsIgnoreCase("telemetry");
  bool isTemperatureTarget = sensorTarget.equalsIgnoreCase(TEMPERATURE_SENSOR_ID) || sensorTarget.equalsIgnoreCase("temperature");
  bool isHumidityTarget = sensorTarget.equalsIgnoreCase(HUMIDITY_SENSOR_ID) || sensorTarget.equalsIgnoreCase("humidity");

  if (payloadHasStringField(payload, "trigger", "stream_pause")) {
    if (isGlobalTarget) {
      telemetryPublishingEnabled = false;
      temperaturePublishingEnabled = false;
      humidityPublishingEnabled = false;
      Serial.println("Telemetry paused (all sensors)");
      return;
    }

    if (isTemperatureTarget) {
      temperaturePublishingEnabled = false;
      Serial.println("Telemetry paused (temperature)");
      return;
    }

    if (isHumidityTarget) {
      humidityPublishingEnabled = false;
      Serial.println("Telemetry paused (humidity)");
      return;
    }
  }

  if (payloadHasStringField(payload, "trigger", "stream_resume")) {
    if (isGlobalTarget) {
      telemetryPublishingEnabled = true;
      temperaturePublishingEnabled = true;
      humidityPublishingEnabled = true;
      Serial.println("Telemetry resumed (all sensors)");
      return;
    }

    if (isTemperatureTarget) {
      temperaturePublishingEnabled = true;
      Serial.println("Telemetry resumed (temperature)");
      return;
    }

    if (isHumidityTarget) {
      humidityPublishingEnabled = true;
      Serial.println("Telemetry resumed (humidity)");
      return;
    }
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

  Serial.println("Activation payload ignored: unsupported payload");
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

void setup() {
  Serial.begin(115200);
  delay(1000);

  buildRuntimeDeviceId();
  buildTopics();

  Serial.print("Runtime DEVICE_ID: ");
  Serial.println(deviceIdBuffer);

  pinMode(ACTIVATION_PIN, OUTPUT);
  digitalWrite(ACTIVATION_PIN, LOW);

  dht.begin();

  mqttClient.setServer(MQTT_HOST, MQTT_PORT);
  mqttClient.setCallback(onMqttMessage);
  mqttClient.setBufferSize(512);

  ensureWiFiConnected();
  ensureMqttConnected();
  syncClock();
  registerDevice();
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

  unsigned long nowMs = millis();

  if (nowMs - lastPublishMs >= PUBLISH_INTERVAL_MS) {
    lastPublishMs = nowMs;
    publishTelemetry();
  }

  if (nowMs - lastStatusMs >= STATUS_INTERVAL_MS) {
    lastStatusMs = nowMs;
    publishDeviceStatus(true);
  }

  delay(20);
}
