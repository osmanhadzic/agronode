#include <WiFi.h>
#include <PubSubClient.h>
#include <DHT.h>
#include <time.h>

#define DHT_PIN 4
#define DHT_TYPE DHT22

const char* WIFI_SSID = "YOUR_WIFI_SSID";
const char* WIFI_PASSWORD = "YOUR_WIFI_PASSWORD";

const char* MQTT_HOST = "192.168.1.100";
const uint16_t MQTT_PORT = 1883;

const char* DEVICE_ID = "device-1";
const unsigned long PUBLISH_INTERVAL_MS = 5000;

const long GMT_OFFSET_SEC = 0;
const int DAYLIGHT_OFFSET_SEC = 0;
const char* NTP_SERVER = "pool.ntp.org";

DHT dht(DHT_PIN, DHT_TYPE);
WiFiClient wifiClient;
PubSubClient mqttClient(wifiClient);

unsigned long lastPublishMs = 0;
char topicBuffer[128];

void ensureWiFiConnected() {
  if (WiFi.status() == WL_CONNECTED) {
    return;
  }

  WiFi.mode(WIFI_STA);
  WiFi.begin(WIFI_SSID, WIFI_PASSWORD);

  while (WiFi.status() != WL_CONNECTED) {
    delay(500);
  }
}

void ensureMqttConnected() {
  if (mqttClient.connected()) {
    return;
  }

  while (!mqttClient.connected()) {
    if (mqttClient.connect(DEVICE_ID)) {
      return;
    }

    delay(1000);
  }
}

void syncClock() {
  configTime(GMT_OFFSET_SEC, DAYLIGHT_OFFSET_SEC, NTP_SERVER);

  time_t now = time(nullptr);
  while (now < 1000000000) {
    delay(300);
    now = time(nullptr);
  }
}

unsigned long currentEpochSeconds() {
  time_t now = time(nullptr);
  if (now < 1000000000) {
    return 0;
  }

  return (unsigned long)now;
}

bool publishTelemetry(float temperature, float humidity, unsigned long epochSeconds) {

  char payload[256];
  int length = snprintf(
    payload,
    sizeof(payload),
    "{\"deviceId\":\"%s\",\"timestamp\":%lu,\"version\":1,\"sensors\":{\"temperature\":%.2f,\"humidity\":%.2f}}",
    DEVICE_ID,
    epochSeconds,
    temperature,
    humidity
  );

  if (length <= 0 || length >= (int)sizeof(payload)) {
    return false;
  }

  snprintf(topicBuffer, sizeof(topicBuffer), "agronode/%s/telemetry", DEVICE_ID);
  return mqttClient.publish(topicBuffer, payload);
}

void setup() {
  Serial.begin(115200);
  dht.begin();

  ensureWiFiConnected();
  syncClock();
  mqttClient.setServer(MQTT_HOST, MQTT_PORT);
}

void loop() {
  ensureWiFiConnected();
  ensureMqttConnected();
  mqttClient.loop();

  unsigned long now = millis();
  if (now - lastPublishMs < PUBLISH_INTERVAL_MS) {
    delay(100);
    return;
  }

  float humidity = dht.readHumidity();
  float temperature = dht.readTemperature();
  unsigned long epochSeconds = currentEpochSeconds();

  if (!isnan(humidity) && !isnan(temperature) && epochSeconds > 0) {
    publishTelemetry(temperature, humidity, epochSeconds);
  }

  lastPublishMs = now;
}
