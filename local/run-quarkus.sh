#!/usr/bin/env bash
# Run Quarkus trip planner against local Kafka (port 8081).
set -euo pipefail
DIR="$(cd "$(dirname "$0")" && pwd)"
APP="${DIR}/apps/trip-quarkus"

if [[ ! -d "$APP" ]]; then
  echo "App missing — run ./materialize.sh first" >&2
  exit 1
fi

if [[ ! -f "$APP/.env" ]]; then
  cp "${DIR}/env.quarkus.example" "$APP/.env"
  echo "Created $APP/.env — set MAAS_API_KEY for real LLM plans"
fi

set -a
# shellcheck disable=SC1091
source "$APP/.env"
set +a

export KAFKA_BOOTSTRAP_SERVERS="${KAFKA_BOOTSTRAP_SERVERS:-localhost:9092}"
export KAFKA_DEVSERVICES="${KAFKA_DEVSERVICES:-false}"
export QUARKUS_HTTP_PORT="${QUARKUS_HTTP_PORT:-8081}"
# LangChain4j rejects empty api-key; placeholder lets the app start (plans fail until a real key is set).
if [[ -z "${MAAS_API_KEY:-}" ]]; then
  export MAAS_API_KEY="local-dev-placeholder"
fi

cd "$APP"
echo "Quarkus → http://localhost:${QUARKUS_HTTP_PORT}/  (Kafka ${KAFKA_BOOTSTRAP_SERVERS})"
echo "First start downloads Maven deps (can take several minutes) — progress is shown below."
if [[ "${MAAS_API_KEY}" == "local-dev-placeholder" ]]; then
  echo "WARN: using placeholder MAAS_API_KEY — set a real key in apps/trip-quarkus/.env for LLM plans."
fi
exec mvn quarkus:dev
