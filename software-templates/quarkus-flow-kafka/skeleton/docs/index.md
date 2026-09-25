# ${{values.component_id}}

${{values.description}}

Agentic Trip Planner ([quarkus-workshop-langchain4j](https://quarkus.io/quarkus-workshop-langchain4j/) section-3/step-04): **Quarkus Flow** (community) + Kafka for human-in-the-loop approval after planning. Application CI/CD runs on **OpenShift Pipelines**. GitLab CI only builds TechDocs.

## Inner loop (Dev Spaces + MaaS)

1. Catalog → **OpenShift Dev Spaces (VS Code)**.
2. Command palette → **Write .env for Red Hat MaaS** (or edit `.env` with `MAAS_API_KEY` from OpenShift AI → Gen AI studio → API keys).
3. Command palette → **Quarkus dev (trip UI + Dev UI on 8080)**. The command installs **JDK 21** if the UDI still defaults to 17 (`release version 21 not supported`).
4. Open the workspace endpoints:
   - **trip-ui** → planner UI (`/`)
   - **quarkus-dev-ui** → `/q/dev-ui`
   - **swagger-ui** → `/q/swagger-ui`
5. Submit a trip plan. When the Flow pauses, use **Approve** / **Reject** (Kafka `flow-in` / `flow-out`).

## Supply chain

Signed `git commit` / `git push` → Nexus, OpenShift Builds, SBOM, cosign, TPA, Conforma. GitLab **tag** promotes to staging; **release** promotes to production.
