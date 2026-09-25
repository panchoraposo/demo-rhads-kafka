# ${{values.component_id}}

Postgres **CDC** with the shared Red Hat build of Debezium (`rhads-debezium` KafkaConnect), schema in **Apicurio Registry**, secrets in **Vault**.

## Paths

| Resource | Value |
|----------|-------|
| Kafka topic | `${{values.kafka_topic}}` |
| Apicurio group / artifact | `${{values.component_id}}` / `order-change` |
| Vault path | `secret/apps/${{values.component_id}}` |

## Trigger events

```sql
INSERT INTO orders (customer, amount, status) VALUES ('demo', 10.00, 'new');
UPDATE orders SET status = 'paid' WHERE customer = 'demo';
DELETE FROM orders WHERE customer = 'demo';
```

Watch the consumer UI and Kafka Console (`${{values.kafka_topic}}`).
