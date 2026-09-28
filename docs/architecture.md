# Architecture — RHADS + Kafka agentic demo

## Goal

A repeatable platform (Ansible + OpenShift GitOps) that demonstrates **software supply chain security** (RHADS) **and** **event-driven agentic HITL** over **Streams for Apache Kafka**, with the **Kafka Console** and **Perses** dashboards.

## Source of truth

| Copy | Role |
| --- | --- |
| **GitHub** `panchoraposo/demo-rhads-kafka` (`main`) | Installer and templates for the **next** cluster. |
| Local working tree | What `./install.sh` applies, then publishes to GitLab. |
| GitLab `platform-engineers/rhads-platform` | What Argo CD reconciles **after** install (placeholders substituted). |

```mermaid
flowchart LR
  GH[GitHub demo-rhads-kafka]
  AN[Ansible install.sh]
  GL[GitLab rhads-platform]
  ACD[OpenShift GitOps]
  GH --> AN
  AN --> GL
  AN --> ACD
  GL --> ACD
```

## Components

RHADS base (Keycloak, RHDH, Dev Spaces, GitLab, Pipelines, Nexus, Quay, RHTAS, TPA, ACS, Vault, ESO, ODF MCG) plus:

| Component | Namespace / notes |
| --- | --- |
| Streams for Apache Kafka | `kafka` — `Kafka`/`KafkaNodePool` `rhads-kafka`, topics `flow-in`/`flow-out` |
| Kafka Console | `kafka-console` — Console CR, host `kafka-console.apps.<cluster>` |
| Red Hat build of Apicurio Registry | `apicurio` — `ApicurioRegistry` `rhads-registry` (KafkaSQL), host `apicurio.apps.<cluster>` |
| Red Hat build of Debezium | `debezium` — shared `KafkaConnect` `rhads-debezium` (Postgres plugin); apps add `KafkaConnector` only |
| Cluster Observability Operator | UIPlugin Monitoring + Perses; dashboards in `rhads-observability` |

Platform routes use **short hostnames** (`<component>.apps.<cluster>`), for example `quay.apps…`, `gitlab.apps…`, `rhdh.apps…`, `argocd.apps…`, `acs.apps…`, `nexus.apps…`, `vault.apps…`, `apicurio.apps…` (not the default OpenShift `<route>-<namespace>.apps…` form).

## Software templates (catalog)

Three scaffolder templates are registered in `catalog/templates.yaml`:

1. **quarkus-flow-kafka** — Trip planner from the [LangChain4j workshop step-04](https://quarkus.io/quarkus-workshop-langchain4j/section-3/step-04/) pattern: Red Hat langchain4j agents + Kafka CloudEvents HITL + MaaS.
2. **go-flow-kafka** — Same UI, plan JSON, multi-step MaaS agents, and CloudEvents HITL as Quarkus; hand-rolled Kafka engine with structured logs; community Go modules for RHDA/TPA/ACS CVEs.
3. **debezium-cdc-postgres** — Postgres + Debezium CDC connector (shared Connect) + Go consumer UI + Apicurio schema registration. Developers never build KafkaConnect themselves.

```mermaid
flowchart TD
  Browser -->|POST plan| App
  App -->|trip.requested| FlowIn[flow-in]
  FlowIn --> Engine[Flow or Go engine]
  Engine -->|approval.requested| FlowOut[flow-out]
  FlowOut --> App
  Browser -->|Approve Reject| App
  App -->|approval.done| FlowIn
  Engine -->|finalized or rejected| FlowOut
  FlowIn --> Console[KafkaConsole]
  FlowOut --> Console
```

```mermaid
flowchart LR
  Desk[Order Desk UI] -->|SQL write| PG[(Postgres orders)]
  PG -->|WAL pgoutput| DBZ[rhads-debezium]
  DBZ -->|topic app.public.orders| K[Kafka]
  K --> Desk
  Schema[Apicurio order-change] -.-> Desk
  Vault[Vault apps/app] -.-> Hub[Developer Hub]
  Schema -.-> Hub
```

## CI / supply chain

Unchanged from demo-rhads: Tekton build/promote, Syft SBOM to TPA, cosign + Rekor, ACS, Conforma STRICT on tag/release, Tekton Chains. Dependencies resolve through **Nexus** (`maven-public`, `npm-group`, `go-group`). Go apps use `build.language: go`: Tekton `rhads-build-source` runs `go mod tidy` + `go mod vendor` via Nexus, then the image build compiles with `-mod=vendor` (offline). The default Go image is **community** (`golang` + `debian`); `Dockerfile.ubi` swaps to Red Hat UBI for an ACS CVE contrast.

## Secrets and schemas in Developer Hub

- **Vault plugin**: every new app sets `vault.io/secrets-path: apps/{component_id}`. Overview shows the HashiCorp Vault card (DB / MaaS keys seeded by the build chart).
- **Apicurio**: trip templates link to shared CloudEvents artifacts; CDC registers `{component_id}/order-change` and exposes an API entity + Apicurio UI link in the catalog.

Kafka bootstrap defaults to `rhads-kafka-kafka-bootstrap.kafka.svc:9092` (internal plain listener for the demo).
