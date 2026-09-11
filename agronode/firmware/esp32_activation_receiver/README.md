# ESP32 Activation Receiver

Firmware for a target device that only receives trigger activation commands from AgroNode backend.

## Topic

Subscribes to:

- `agronode/{DEVICE_ID}/activation`

## Expected payload

```json
{
  "deviceId": "pump-node-1",
  "trigger": "above_max",
  "sensor": "co2",
  "limitType": "max",
  "value": 800,
  "threshold": 700,
  "activated": true,
  "timestamp": 1715539200
}
```

## Behavior

- If `deviceId` matches `DEVICE_ID` and `activated` is `true`, sets `ACTIVATION_PIN` HIGH.
- Automatically sets `ACTIVATION_PIN` LOW after `ACTIVATION_SIGNAL_DURATION_MS`.
- If `activated` is `false`, immediately sets `ACTIVATION_PIN` LOW.

## Configure before upload

In `esp32_activation_receiver.ino` set:

- `WIFI_SSID`
- `WIFI_PASSWORD`
- `MQTT_HOST`
- `DEVICE_ID` (must match trigger `targetDeviceId`)
- `ACTIVATION_PIN`

## Upload

```bash
cd agronode/firmware/esp32_activation_receiver
arduino-cli upload -p /dev/ttyUSB0 --fqbn esp32:esp32:esp32doit-devkit-v1 esp32_activation_receiver.ino
```
