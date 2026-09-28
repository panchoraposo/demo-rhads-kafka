# RHADS + Kafka — Agentic trip planner + CDC demo

**Ansible + OpenShift GitOps** installer based on [demo-rhads](https://github.com/panchoraposo/demo-rhads), extended with **Streams for Apache Kafka**, the **Kafka Console**, **Cluster Observability Operator / Perses**, **Red Hat build of Apicurio Registry**, **shared Red Hat build of Debezium**, and **three** Developer Hub templates:

| Template | Default name | Stack |
| --- | --- | --- |
| **Agentic Trip Planner — LangChain4j/Kafka** | `trip-quarkus` | RHBOQ 3.33 + Red Hat [langchain4j](https://docs.quarkiverse.io/quarkus-langchain4j/dev/) agents + Kafka CloudEvents HITL |
| **Agentic Trip Planner — Go/Kafka** | `trip-go` | Go + same UI/plan JSON + multi-step MaaS + Kafka HITL; community modules/images (ACS CVEs); switch to `Dockerfile.ubi` for Red Hat UBI |
| **CDC — Debezium PostgreSQL** | `orders-cdc` | Postgres + shared Debezium + Order Desk UI (write DB → live CDC) + Apicurio + Vault |

This GitHub repository is the **source of truth for a repeatable install**. Clone it, log in to a cluster, run `./install.sh`. Ansible publishes the working tree to in-cluster GitLab and points Argo CD at that copy.

Stack (RHADS base + Kafka/observability/CDC):

| Product | Channel / notes |
| --- | --- |
| OpenShift GitOps / Pipelines / RHDH / Dev Spaces | same as demo-rhads |
| RHTAS / TPA / ACS / Quay / RHBK / GitLab / ODF MCG / Vault / ESO / Nexus | same as demo-rhads |
| Streams for Apache Kafka | `stable` (AMQ Streams) — cluster `rhads-kafka`, topics `flow-in` / `flow-out` |
| Streams for Apache Kafka Console | Console CR → `https://kafka-console.apps.<cluster>/` |
| Red Hat build of Apicurio Registry | `apicurio` NS → `https://apicurio.apps.<cluster>/` (KafkaSQL) |
| Red Hat build of Debezium | Shared `KafkaConnect` `rhads-debezium` in `debezium` NS — templates only add connectors |
| Cluster Observability Operator | Monitoring UIPlugin with **Perses** dashboards |
| HashiCorp Vault plugin (RHDH) | Overview card via `vault.io/secrets-path: apps/{app}` |

Trip planning uses the **Red Hat langchain4j** BOM on RHBOQ 3.33; Kafka CloudEvents on `flow-in` / `flow-out` carry the workshop-style pause/resume HITL.

## Architecture

```
GitHub (this repo)  ──install.sh──► cluster (operators + Kafka + Apicurio + Debezium + Perses)
                         │
                         └── GitLab platform-engineers/rhads-platform  ◄── Argo CD

Developer Hub  ──template──► GitLab (source + *-gitops)
      │                         │
      └── Dev Spaces            ├── push → Nexus + Builds + SBOM + cosign + ACS + TPA → dev
         RHDA / gitsign         ├── Trip UI / CDC Order Desk → Kafka → Approve or live CDC
                                ├── Apicurio schemas (catalog API + UI links)
                                ├── Vault secrets card on the component
                                └── Kafka Console + Observe → Dashboards (Perses)
```

Details: [docs/architecture.md](docs/architecture.md). Live script: [docs/demo-script.md](docs/demo-script.md). Credentials: [docs/credentials.md](docs/credentials.md).

**Local (laptop):** optional untracked `local/` workspace (Podman Compose + materialized template apps) for smoke tests before the cluster. It is gitignored and not part of this repository.

## Prerequisites

- OpenShift **4.20+** with `cluster-admin` (typical sandbox: **16 CPU / 64 Gi**; Kafka + COO + Debezium need extra headroom).
- A **default** StorageClass (whatever the cluster provides). PVCs omit `storageClassName`. ODF Multicloud Object Gateway for ObjectBucketClaims.
- `oc` logged in. `python3`. Helm 3 is installed by `install.sh` if missing.

## Installation

```bash
git clone https://github.com/panchoraposo/demo-rhads-kafka.git
cd demo-rhads-kafka
oc login --server=https://api.<cluster> --token=...
./install.sh
```

The playbook installs the RHADS stack **plus** AMQ Streams, Kafka Console, Apicurio, Debezium Connect, and Cluster Observability Operator (Perses). Typical time: **50–80 minutes** on a single-node sandbox.

After install, GitOps self-heals from **GitLab**. Commit platform changes to GitHub `main` before the next sandbox recreate.

## Users

| Who | Username | Password | Where |
| --- | --- | --- | --- |
| Developer | `dev1` / `dev2` / `dev3` | `backstage` | Keycloak → RHDH, GitLab, gitsign |
| Platform engineer | `pe1` / `pe2` / `pe3` | `backstage` | Keycloak → RHDH (admin RBAC) |
| GitLab root | `root` | `backstage` | GitLab |
| Quay | `quayadmin` | `backstage` | Quay |
| Nexus | `admin` | `admin123` | Nexus |
| Vault | `vaultadmin` | `backstage` | Vault UI |
| TPA | `tpa-admin` | `backstage` | realm `trustify` |

## Demo flow (short)

1. Sign in to Developer Hub as `dev1`.
2. Create → **Agentic Trip Planner — LangChain4j/Kafka** (paste MaaS API key). Optional: **Go/Kafka**.
3. Create → **CDC — Debezium PostgreSQL** — no Connect build; shared platform Connect + Postgres + schema registration.
4. On the new component Overview: **Vault** card (`apps/{name}`) and **Apicurio** API / links.
5. Dev Spaces → RHDA on `pom.xml` / `go.mod` → TPA SBOM → ACS.
6. Trip UI → plan → Kafka Console → **Approve**. CDC Order Desk → **New order** / **Mark paid** → events on `{app}.public.orders`.
7. OpenShift console → **Observe → Dashboards (Perses)**.
8. Continue the RHADS promotion story as in [docs/demo-script.md](docs/demo-script.md).

## Uninstall

```bash
./venv/bin/ansible-playbook -i ansible/inventory ansible/playbooks/uninstall.yaml
```
