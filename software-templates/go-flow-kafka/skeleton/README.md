# ${{values.component_id}}

${{values.description}}

Agentic **Trip Planner** with the same behaviour as the Quarkus LangChain4j template: MaaS multi-step planning (vehicle + itinerary → evaluators → reviser → costs), Kafka CloudEvents HITL on `flow-in` / `flow-out`, and the same UI/API contract. Runtime is **Go** with community modules/images for the RHDA/ACS CVE story (switch to `Dockerfile.ubi` for Red Hat UBI).

## Inner loop

1. Set `MAAS_API_KEY` (and optional `MAAS_BASE_URL` / `MAAS_MODEL`).
2. Point `KAFKA_BOOTSTRAP_SERVERS` at the cluster (or local Compose `localhost:9092`).
3. `go run ./cmd/server` — UI on the configured `PORT` (default 8080 in cluster, 8082 local).

Plan a trip from the UI (`POST /trip/plan`). The Kafka engine logs each agent step, publishes to `flow-out`, and pauses for approval. Approve or reject with `PUT /trip/approve`.

## Supply chain

Signed `git commit` / `git push` → OpenShift Builds, SBOM, cosign, TPA, Conforma. GitLab **tag** promotes to staging; **release** promotes to production.
