# ${{values.component_id}}

Change Data Capture (CDC) demo generated from the **debezium-cdc-postgres** software template.

## What you get

| Piece | Purpose |
|-------|---------|
| Postgres (`orders`) | Source DB with `orders` table + logical replication |
| Debezium connector | Shared Red Hat build of Debezium KafkaConnect publishes `${{values.kafka_topic}}` |
| Order Desk (Go) | Writes orders to Postgres and shows live CDC events (SSE) |
| Apicurio schema | JSON Schema `${{values.owner}}/${{values.component_id}}-order-cdc` |
| Vault secrets | Path `apps/${{values.component_id}}` (visible in Developer Hub) |

## Live demo (dev UI)

1. Open the **CDC Order Desk** Route.
2. Click **New order** — Postgres write → Debezium → Kafka → feed flashes (pipeline strip pulses).
3. Select the row → **Update amount** or **Mark paid** — UPDATE with before → after.
4. Optional: Kafka Console topic `${{values.kafka_topic}}`; Apicurio artifact link in the header.

No debug-pod SQL required on stage.

## Developer Hub

- **Catalog** → Component `${{values.component_id}}` → Vault card + Apicurio API entity
- **Apicurio Registry UI**: https://apicurio.${{values.cluster_subdomain}}
- **Vault UI**: https://vault.${{values.cluster_subdomain}}

## Local run (laptop, untracked `local/`)

From the demo repo:

```bash
cd local
podman-compose up -d          # Kafka :9092 + Postgres :5432 (required)
./materialize.sh              # copy this skeleton → apps/orders-cdc
./run-cdc.sh                  # http://127.0.0.1:8083  (LOCAL_CDC_FANOUT=true)
./demo-cdc-traffic.sh 6       # optional CREATE/UPDATE/DELETE load
```

Or run the binary directly (use `127.0.0.1` to avoid IPv6 `localhost` misses):

```bash
export KAFKA_BOOTSTRAP_SERVERS=127.0.0.1:9092
export KAFKA_TOPIC=${{values.kafka_topic}}
export DATABASE_HOST=127.0.0.1
export DATABASE_USER=orders
export DATABASE_PASSWORD=backstage
export DATABASE_NAME=orders
export LOCAL_CDC_FANOUT=true   # no Debezium Connect locally — app emits CDC-shaped events after writes
go run ./cmd/server
```

`LOCAL_CDC_FANOUT` must stay **off** on the cluster so Debezium is what captures changes.

## Manual SQL (optional)

```sql
INSERT INTO orders (customer, amount, status) VALUES ('alice', 42.50, 'NEW');
UPDATE orders SET status = 'paid' WHERE customer = 'alice';
```
