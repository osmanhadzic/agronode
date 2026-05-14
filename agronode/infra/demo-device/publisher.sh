#!/bin/sh

set -eu

DEVICE_ID="${DEVICE_ID:-demo-device-1}"
INTERVAL_SECONDS="${PUBLISH_INTERVAL_SECONDS:-5}"
MQTT_HOST="${MQTT_HOST:-mosquitto}"
MQTT_PORT="${MQTT_PORT:-1883}"

echo "[demo-device] starting publisher for ${DEVICE_ID} -> ${MQTT_HOST}:${MQTT_PORT}"

while true
do
  timestamp="$(date +%s)"

  temperature_raw=$((200 + RANDOM % 120))
  humidity_raw=$((450 + RANDOM % 350))

  temperature="$(printf '%d.%d' $((temperature_raw / 10)) $((temperature_raw % 10)))"
  humidity="$(printf '%d.%d' $((humidity_raw / 10)) $((humidity_raw % 10)))"

  topic="agronode/${DEVICE_ID}/telemetry"
  payload="{\"deviceId\":\"${DEVICE_ID}\",\"timestamp\":${timestamp},\"version\":1,\"sensors\":{\"temperature\":${temperature},\"humidity\":${humidity}}}"

  mosquitto_pub -h "${MQTT_HOST}" -p "${MQTT_PORT}" -t "${topic}" -m "${payload}"
  echo "[demo-device] published ${topic} ${payload}"

  sleep "${INTERVAL_SECONDS}"
done
