# ${{values.component_id}}

${{values.description}}

Agentic **Trip Planner** inspired by [quarkus-workshop-langchain4j](https://quarkus.io/quarkus-workshop-langchain4j/) section-3/step-04: **Red Hat langchain4j** agents + Kafka CloudEvents for human approval after planning. Runtime is **Red Hat build of Quarkus 3.33** and **JDK 21**. Application CI/CD is **OpenShift Pipelines**; GitLab CI only builds TechDocs.

## Inner loop (Dev Spaces + MaaS)

1. Catalog → **OpenShift Dev Spaces (VS Code)**.
2. Command palette → **Write .env for Red Hat MaaS** if `MAAS_API_KEY` is empty (OpenShift AI → Gen AI studio → API keys).
3. Command palette → **Quarkus dev (trip UI + Dev UI on 8080)**. That step selects **JDK 21** (the UDI default is 17; compiling `release` 21 with 17 fails). For local Kafka Dev Services leave `KAFKA_BOOTSTRAP_SERVERS` unset; against the cluster set it to `rhads-kafka-kafka-bootstrap.kafka.svc:9092`.
4. Open **PORTS** (or **Endpoints**):
   - **trip-ui** → planner UI (`/`)
   - **quarkus-dev-ui** → `/q/dev-ui` (needs the Dev Spaces task so `QUARKUS_DEV_UI_CONTEXT_ROOT` is set)
   - **swagger-ui** → `/q/swagger-ui`

Plan a trip from the UI (`POST /trip/plan`). The Kafka engine consumes `flow-in`, runs LangChain4j agents, and publishes to `flow-out` (pause for approval). Approve or reject with `PUT /trip/approve` (resumes via `flow-in`).

The cluster **Trip Planner UI (dev)** is `https://trip-quarkus-trip-quarkus-dev.<apps-domain>/`. Staging/prod stay at 0 replicas until a GitLab tag/release promote.

## Supply chain

Signed `git commit` / `git push` → Nexus, OpenShift Builds, SBOM, cosign, TPA, Conforma. GitLab **tag** promotes to staging; **release** promotes to production.
