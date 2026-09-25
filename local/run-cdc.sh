#!/usr/bin/env bash
# Run CDC Go consumer against local Kafka topic orders-cdc.public.orders (port 8083).
set -euo pipefail
DIR="$(cd "$(dirname "$0")" && pwd)"
APP="${DIR}/apps/orders-cdc"

if [[ ! -d "$APP" ]]; then
  echo "App missing — run ./materialize.sh first" >&2
  exit 1
fi

if ! command -v go >/dev/null 2>&1; then
  echo "Go not found. Install with: brew install go" >&2
  exit 1
fi

export KAFKA_BOOTSTRAP_SERVERS="${KAFKA_BOOTSTRAP_SERVERS:-localhost:9092}"
export KAFKA_TOPIC="${KAFKA_TOPIC:-orders-cdc.public.orders}"
export PORT="${PORT:-8083}"
export WEB_ROOT="${WEB_ROOT:-web}"
export APICURIO_REGISTRY_URL="${APICURIO_REGISTRY_URL:-http://localhost:8080}"

cd "$APP"
echo "CDC consumer → http://localhost:${PORT}/  topic=${KAFKA_TOPIC}"
echo "Publish a sample event:  ${DIR}/publish-cdc-sample.sh"
# Same local sarama module-path fix as run-go.sh
# IBM/sarama v1.38.1 declares Shopify path — rewrite require for local tidy.
sed -i.bak 's|github.com/IBM/sarama v1.38.1|github.com/IBM/sarama v1.43.3|' go.mod
rm -f go.mod.bak
go mod tidy
exec go run ./cmd/server
