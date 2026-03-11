#!/usr/bin/env bash
set -euo pipefail

APP_ID=${APP_ID:-test-app}
JOB_ID=${JOB_ID:-job_test_7}
MQTT_HOST=${MQTT_HOST:-localhost}
MQTT_PORT=${MQTT_PORT:-1883}
MODEL=${MODEL:-"gpt-5.1-codex-mini"}

PAYLOAD=$(cat <<EOF
{
  "job_id": "${JOB_ID}",
  "app_id": "${APP_ID}",
  "prompt": "What is the meaning to all life and everything in the universe?",
  "provider": "openai",
  "model": "${MODEL}",
  "params": { "temperature": 0.2 }
}
EOF
)

docker run --rm --network host eclipse-mosquitto:2.0 \
  mosquitto_sub -h "${MQTT_HOST}" -p "${MQTT_PORT}" -t "jobs/complete/${APP_ID}" -v &
SUB_PID=$!

sleep 0.5

docker run --rm --network host eclipse-mosquitto:2.0 \
  mosquitto_pub -h "${MQTT_HOST}" -p "${MQTT_PORT}" -t "jobs/enqueue/${APP_ID}" -m "${PAYLOAD}"

wait ${SUB_PID}
