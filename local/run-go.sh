#!/usr/bin/env bash
# Run Go trip planner against local Kafka (port 8082).
set -euo pipefail
DIR="$(cd "$(dirname "$0")" && pwd)"
APP="${DIR}/apps/trip-go"

if [[ ! -d "$APP" ]]; then
  echo "App missing — run ./materialize.sh first" >&2
  exit 1
fi

if ! command -v go >/dev/null 2>&1; then
  echo "Go not found. Install with: brew install go" >&2
  exit 1
fi

if [[ ! -f "$APP/.env" ]]; then
  cat >"$APP/.env" <<EOF
MAAS_BASE_URL=https://maas-rhdp.apps.maas.redhatworkshops.io/v1/
MAAS_MODEL=qwen3-14b
MAAS_API_KEY=
KAFKA_BOOTSTRAP_SERVERS=localhost:9092
PORT=8082
WEB_ROOT=web
SKILLS_DIR=skills
EOF
  echo "Created $APP/.env — set MAAS_API_KEY for real LLM plans"
fi

set -a
# shellcheck disable=SC1091
source "$APP/.env"
set +a

export KAFKA_BOOTSTRAP_SERVERS="${KAFKA_BOOTSTRAP_SERVERS:-localhost:9092}"
export PORT="${PORT:-8082}"
export WEB_ROOT="${WEB_ROOT:-web}"
export SKILLS_DIR="${SKILLS_DIR:-skills}"
export MAAS_BASE_URL="${MAAS_BASE_URL:-https://maas-rhdp.apps.maas.redhatworkshops.io/v1/}"
export MAAS_MODEL="${MAAS_MODEL:-qwen3-14b}"

cd "$APP"
echo "Go trip → http://localhost:${PORT}/  (Kafka ${KAFKA_BOOTSTRAP_SERVERS})"
if [[ -z "${MAAS_API_KEY:-}" ]]; then
  echo "WARN: MAAS_API_KEY empty — planning will fail with maas_api_key_missing (same as Quarkus)."
fi
# IBM/sarama v1.38.1 declares Shopify path — bump for local tidy.
sed -i.bak 's|github.com/IBM/sarama v1.38.1|github.com/IBM/sarama v1.43.3|' go.mod
rm -f go.mod.bak
go mod tidy
exec go run ./cmd/server
