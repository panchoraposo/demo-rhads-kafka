# ${{values.component_id}}

Postgres **CDC** with the shared Red Hat build of Debezium (`rhads-debezium` KafkaConnect), schema in **Apicurio Registry**, secrets in **Vault**.

## Paths

| Resource | Value |
|----------|-------|
| Kafka topic | `${{values.kafka_topic}}` |
| Apicurio group / artifact | `${{values.component_id}}` / `order-change` |
| Vault path | `secret/apps/${{values.component_id}}` |

## Live demo

Open the Order Desk UI:

1. **New order** — write Postgres; Debezium captures; feed flashes.
2. **Mark paid** / **Cancel** — UPDATE with before → after.
3. Optional Kafka Console on `${{values.kafka_topic}}`.
