#!/bin/sh

set -eu

DEVICE_ID="${DEVICE_ID:-demo-device-1}"
INTERVAL_SECONDS="${PUBLISH_INTERVAL_SECONDS:-15}"
MQTT_HOST="${MQTT_HOST:-mosquitto}"
MQTT_PORT="${MQTT_PORT:-1883}"

echo "[demo-device] starting publisher for ${DEVICE_ID} -> ${MQTT_HOST}:${MQTT_PORT}"

rand_between() {
  min="$1"
  max="$2"
  random_value="$(od -An -N2 -tu2 /dev/urandom | tr -d ' ')"
  range=$((max - min + 1))
  echo $((min + (random_value % range)))
}

publish_sensor() {
  sensor_id="$1"
  sensor_value="$2"
  timestamp="$3"
  topic="agronode/${DEVICE_ID}/telemetry"
  payload="{\"deviceId\":\"${DEVICE_ID}\",\"sensorId\":\"${sensor_id}\",\"timestamp\":${timestamp},\"version\":1,\"sensors\":{\"${sensor_id}\":${sensor_value}}}"

  mosquitto_pub -h "${MQTT_HOST}" -p "${MQTT_PORT}" -t "${topic}" -m "${payload}"
  echo "[demo-device] published ${topic} ${payload}"
}

publish_status() {
  signal_strength="$1"
  timestamp="$2"
  topic="agronode/${DEVICE_ID}/status"
  payload="{\"deviceId\":\"${DEVICE_ID}\",\"timestamp\":${timestamp},\"online\":true,\"signal_strength\":${signal_strength},\"meta\":{\"source\":\"demo-device\"}}"

  mosquitto_pub -h "${MQTT_HOST}" -p "${MQTT_PORT}" -t "${topic}" -m "${payload}"
  echo "[demo-device] published ${topic} ${payload}"
}

while true
do
  timestamp="$(date +%s)"

  temperature_raw="$(rand_between 200 319)"
  humidity_raw="$(rand_between 450 799)"
  signal_strength="$(rand_between -90 -61)"

  temperature="$(printf '%d.%d' $((temperature_raw / 10)) $((temperature_raw % 10)))"
  humidity="$(printf '%d.%d' $((humidity_raw / 10)) $((humidity_raw % 10)))"

  publish_sensor "dht11-temp" "${temperature}" "${timestamp}"
  sleep 1
  publish_sensor "dht11-humidity" "${humidity}" "${timestamp}"
  sleep 1
  publish_status "${signal_strength}" "${timestamp}"

  sleep "${INTERVAL_SECONDS}"
done
