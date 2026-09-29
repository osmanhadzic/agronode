# ESP8266 DHT22 Publisher

ESP8266 firmware for AgroNode that:

- registers device over MQTT (`agronode/{deviceId}/register`)
- publishes DHT22 telemetry (`agronode/{deviceId}/telemetry`)
- publishes status (`agronode/{deviceId}/status`)
- listens for activation/stream control (`agronode/{deviceId}/activation`)

## Supported payload contract

This firmware follows `MQTT_CONTRACT.md` and sends per-sensor telemetry messages:

```json
{
  "deviceId": "esp8266-greenhouse-ABC123",
  "sensorId": "dht22-temp",
  "timestamp": 1715539200,
  "sensors": {
    "dht22-temp": 24.31
  }
}
```

## Configure before upload

Open `esp8266_dht22_publisher.ino` and set:

- `WIFI_SSID`
- `WIFI_PASSWORD`
- `MQTT_HOST`
- optional pin mapping (`DHT_PIN`, `ACTIVATION_PIN`)

Default pin mapping (NodeMCU):

- `DHT_PIN = 4` (D2)
- `ACTIVATION_PIN = 5` (D1)

## Dependencies

Install Arduino libraries:

- `PubSubClient`
- `DHT sensor library`
- `Adafruit Unified Sensor`

## Upload with arduino-cli

```bash
cd agronode/firmware/esp8266_dht22_publisher
arduino-cli board list
arduino-cli compile --fqbn esp8266:esp8266:nodemcuv2 esp8266_dht22_publisher.ino
arduino-cli upload -p /dev/ttyUSB0 --fqbn esp8266:esp8266:nodemcuv2 esp8266_dht22_publisher.ino
```

## Behavior

- `stream_pause` / `stream_resume` trigger payloads toggle telemetry globally or per sensor.
- If payload includes `"activated": true`, `ACTIVATION_PIN` goes HIGH for `ACTIVATION_SIGNAL_DURATION_MS`.
- If payload includes `"activated": false`, `ACTIVATION_PIN` is set LOW immediately.
