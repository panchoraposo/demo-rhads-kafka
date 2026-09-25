#!/usr/bin/env bash
# Copy MaaS settings into .env for quarkus:dev (gitignored).
set -euo pipefail
if [ -f .env ]; then
  echo ".env already exists — not overwriting"
  exit 0
fi
if [ -z "${MAAS_API_KEY:-}" ]; then
  echo "WARN: MAAS_API_KEY is empty. Set it in .env (OpenShift AI → Gen AI studio → API keys)."
fi
cat > .env <<ENVEOF
MAAS_BASE_URL=${MAAS_BASE_URL:-https://maas-rhdp.apps.maas.redhatworkshops.io/v1/}
MAAS_MODEL=${MAAS_MODEL:-qwen3-14b}
MAAS_API_KEY=${MAAS_API_KEY:-}
KAFKA_BOOTSTRAP_SERVERS=${KAFKA_BOOTSTRAP_SERVERS:-rhads-kafka-kafka-bootstrap.kafka.svc:9092}
KAFKA_DEVSERVICES=${KAFKA_DEVSERVICES:-false}
ENVEOF
echo "Wrote .env (MAAS_MODEL=${MAAS_MODEL:-qwen3-14b})"
