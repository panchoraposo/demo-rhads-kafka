# ${{values.component_id}}

Same CloudEvents HITL and plan JSON shape as the Quarkus LangChain4j trip planner. Multi-step MaaS planning with structured logs (`[planner]`, `[kafka]`, `[api]`).

## Demo path

1. Open Dev Spaces (or run locally against Compose Kafka).
2. Set `MAAS_API_KEY`.
3. Plan a trip from the UI.
4. Approve / Reject via Kafka `flow-in` / `flow-out`.
