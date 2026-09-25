# ${{values.component_id}}

Change Data Capture (CDC) demo generated from the **debezium-cdc-postgres** software template.

## What you get

| Piece | Purpose |
|-------|---------|
| Postgres (`orders`) | Source DB with `orders` table + logical replication |
| Debezium connector | Shared Red Hat build of Debezium KafkaConnect publishes `${{values.kafka_topic}}` |
| Go consumer | Reads CDC events and shows them in a small UI |
| Apicurio schema | JSON Schema `${{values.owner}}/${{values.component_id}}-order-cdc` |
| Vault secrets | Path `apps/${{values.component_id}}` (visible in Developer Hub) |

## Developer Hub

- **Catalog** → Component `${{values.component_id}}` → Vault card + Apicurio API entity
- **Apicurio Registry UI**: https://apicurio.${{values.cluster_subdomain}}
- **Vault UI**: https://vault.${{values.cluster_subdomain}}

## Local run

```bash
export KAFKA_BOOTSTRAP_SERVERS=localhost:9092
export KAFKA_TOPIC=${{values.kafka_topic}}
export APICURIO_REGISTRY_URL=http://localhost:8080
go run ./cmd/server
```

## Trigger CDC

```sql
INSERT INTO orders (customer, amount, status) VALUES ('alice', 42.50, 'NEW');
UPDATE orders SET status = 'PAID' WHERE customer = 'alice';
```
