#!/usr/bin/env bash
# Publish a sample Debezium-style JSON event to orders-cdc.public.orders (no Connect needed).
set -euo pipefail
DIR="$(cd "$(dirname "$0")" && pwd)"
TOPIC="${KAFKA_TOPIC:-orders-cdc.public.orders}"

id=$((RANDOM % 9000 + 1000))
amount=$(awk -v r="$RANDOM" 'BEGIN { printf "%.2f", 5 + (r % 9950) / 100 }')
customers=(alice bob carol dave eve frank grace heidi ivan judy)
customer="${customers[$((RANDOM % ${#customers[@]}))]}"
statuses=(new pending paid shipped cancelled)
status="${statuses[$((RANDOM % ${#statuses[@]}))]}"
ts_ms=$(($(date +%s) * 1000))
ops=(c u)
op="${ops[$((RANDOM % ${#ops[@]}))]}"

if [[ "$op" == "u" ]]; then
  payload=$(printf '{"payload":{"before":{"id":%d,"customer":"%s","amount":1.00,"status":"new"},"after":{"id":%d,"customer":"%s","amount":%s,"status":"%s"},"op":"u","ts_ms":%d}}' \
    "$id" "$customer" "$id" "$customer" "$amount" "$status" "$ts_ms")
else
  payload=$(printf '{"payload":{"before":null,"after":{"id":%d,"customer":"%s","amount":%s,"status":"%s"},"op":"c","ts_ms":%d}}' \
    "$id" "$customer" "$amount" "$status" "$ts_ms")
fi

if command -v podman >/dev/null 2>&1; then
  echo "$payload" | podman exec -i rhads-local-kafka \
    /opt/kafka/bin/kafka-console-producer.sh --bootstrap-server localhost:9092 --topic "$TOPIC"
elif command -v docker >/dev/null 2>&1; then
  echo "$payload" | docker exec -i rhads-local-kafka \
    /opt/kafka/bin/kafka-console-producer.sh --bootstrap-server localhost:9092 --topic "$TOPIC"
else
  echo "Need podman or docker to publish via the kafka container" >&2
  exit 1
fi

echo "Published ${op} id=${id} customer=${customer} amount=${amount} status=${status} → ${TOPIC}"
