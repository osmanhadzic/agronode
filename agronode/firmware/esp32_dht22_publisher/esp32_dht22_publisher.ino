#include <WiFi.h>
#include <PubSubClient.h>
#include <DHT.h>
#include <time.h>

#define DHT_PIN 4
#define DHT_TYPE DHT11
#define ACTIVATION_PIN 2

const char* WIFI_SSID = "hadzic";
const char* WIFI_PASSWORD = "techno123";

const char* MQTT_HOST = "192.168.193.106";
const uint16_t MQTT_PORT = 1883;

const char* DEVICE_ID = "Plastenik-1";
const char* FIRMWARE_VERSION = "1.0.1";
const unsigned long PUBLISH_INTERVAL_MS = 5000;
const unsigned long ACTIVATION_SIGNAL_DURATION_MS = 5000;

const long GMT_OFFSET_SEC = 0;
const int DAYLIGHT_OFFSET_SEC = 0;
const char* NTP_SERVER = "pool.ntp.org";

DHT dht(DHT_PIN, DHT_TYPE);
WiFiClient wifiClient;
PubSubClient mqttClient(wifiClient);

unsigned long lastPublishMs = 0;
unsigned long activationSignalUntilMs = 0;
char topicBuffer[128];
char activationTopicBuffer[128];

void handleActivationPayload(const String& payload);
void onMqttMessage(char* topic, byte* payload, unsigned int length);
bool payloadHasDeviceId(const String& payload, const char* expectedDeviceId);
bool payloadHasBooleanField(const String& payload, const char* fieldName, bool expectedValue);

void setup() {
  Serial.begin(115200);
  delay(1000);
  Serial.println();
  Serial.println("BOOT OK");

  dht.begin();
  delay(2000);

  pinMode(ACTIVATION_PIN, OUTPUT);
  digitalWrite(ACTIVATION_PIN, LOW);

  Serial.print("DHT pin: ");
  Serial.println(DHT_PIN);
  Serial.println("DHT init done");

  mqttClient.setServer(MQTT_HOST, MQTT_PORT);
  mqttClient.setCallback(onMqttMessage);

  ensureWiFiConnected();
  syncClock();

  Serial.println("SETUP DONE");
}

void ensureWiFiConnected() {
  if (WiFi.status() == WL_CONNECTED) {
    Serial.println("WiFi already connected");
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

      snprintf(activationTopicBuffer, sizeof(activationTopicBuffer), "agronode/%s/activation", DEVICE_ID);
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
  if (!payloadHasDeviceId(payload, DEVICE_ID)) {
    Serial.println("Activation payload ignored: deviceId mismatch");
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

bool publishTelemetry(float temperature, float humidity, unsigned long epochSeconds) {
  char payload[384];
  int length = snprintf(
    payload,
    sizeof(payload),
    "{\"deviceId\":\"%s\",\"timestamp\":%lu,\"version\":1,"
    "\"meta\":{\"fw\":\"%s\",\"ip\":\"%s\",\"rssi\":%d,\"uptime\":%lu},"
    "\"sensors\":{\"temperature\":%.2f,\"humidity\":%.2f}}",
    DEVICE_ID,
    epochSeconds,
    FIRMWARE_VERSION,
    WiFi.localIP().toString().c_str(),
    (int)WiFi.RSSI(),
    millis() / 1000UL,
    temperature,
    humidity
  );

  if (length <= 0 || length >= (int)sizeof(payload)) {
    Serial.println("Payload build failed");
    return false;
  }

  snprintf(topicBuffer, sizeof(topicBuffer), "agronode/%s/telemetry", DEVICE_ID);

  Serial.print("Topic: ");
  Serial.println(topicBuffer);
  Serial.print("Payload: ");
  Serial.println(payload);

  bool ok = mqttClient.publish(topicBuffer, payload);

  Serial.print("Publish: ");
  Serial.println(ok ? "OK" : "FAILED");

  return ok;
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

unsigned long currentEpochSeconds() {
  time_t now = time(nullptr);
  if (now < 1000000000) {
    return 0;
  }

  return (unsigned long)now;
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
  if (nowMs - lastPublishMs < PUBLISH_INTERVAL_MS) {
    delay(100);
    return;
  }

  lastPublishMs = nowMs;

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

  unsigned long epochSeconds = currentEpochSeconds();
  publishTelemetry(temperature, humidity, epochSeconds);
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

  delay(3000);
}
