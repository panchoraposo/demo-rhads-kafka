# RHADS + Kafka — Agentic trip planner demo

**Ansible + OpenShift GitOps** installer based on [demo-rhads](https://github.com/panchoraposo/demo-rhads), extended with **Streams for Apache Kafka**, the **Kafka Console**, **Cluster Observability Operator / Perses**, and **two** Developer Hub templates:

| Template | Default name | Stack |
| --- | --- | --- |
| **Agentic Trip Planner — Quarkus Flow/Kafka** | `trip-quarkus` | RHBOQ 3.33 + community [Quarkus Flow](https://docs.quarkiverse.io/quarkus-flow/dev/) + Kafka CloudEvents HITL |
| **Agentic Trip Planner — Go/Kafka** | `trip-go` | Go + same CloudEvents contract + community modules/images (ACS CVEs); switch to `Dockerfile.ubi` for Red Hat UBI |

This GitHub repository is the **source of truth for a repeatable install**. Clone it, log in to a cluster, run `./install.sh`. Ansible publishes the working tree to in-cluster GitLab and points Argo CD at that copy.

Stack (RHADS base + Kafka/observability):

| Product | Channel / notes |
| --- | --- |
| OpenShift GitOps / Pipelines / RHDH / Dev Spaces | same as demo-rhads |
| RHTAS / TPA / ACS / Quay / RHBK / GitLab / ODF MCG / Vault / ESO / Nexus | same as demo-rhads |
| Streams for Apache Kafka | `stable` (AMQ Streams) — cluster `rhads-kafka`, topics `flow-in` / `flow-out` |
| Streams for Apache Kafka Console | Console CR → `https://kafka-console.apps.<cluster>/` |
| Cluster Observability Operator | Monitoring UIPlugin with **Perses** dashboards |

Quarkus Flow is a **Quarkiverse community** extension (not in the Red Hat build of Quarkus supported BOM). The demo runs it on RHBOQ 3.33 for the workshop-style pause/resume HITL over Kafka.

## Architecture

```
GitHub (this repo)  ──install.sh──► cluster (operators + Kafka + Perses)
                         │
                         └── GitLab platform-engineers/rhads-platform  ◄── Argo CD

Developer Hub  ──template──► GitLab (source + *-gitops)
      │                         │
      └── Dev Spaces            ├── push → Nexus + Builds + SBOM + cosign + ACS + TPA → dev
         RHDA / gitsign         ├── Trip UI → Kafka flow-in/out → Approve/Reject
                                └── Kafka Console + Observe → Dashboards (Perses)
```

Details: [docs/architecture.md](docs/architecture.md). Live script: [docs/demo-script.md](docs/demo-script.md). Credentials: [docs/credentials.md](docs/credentials.md).

## Prerequisites

- OpenShift **4.20+** with `cluster-admin` (typical sandbox: **16 CPU / 64 Gi**; Kafka + COO need extra headroom).
- Storage class **`gp3-csi`**. ODF Multicloud Object Gateway for ObjectBucketClaims.
- `oc` logged in. `python3`. Helm 3 is installed by `install.sh` if missing.

## Installation

```bash
git clone https://github.com/panchoraposo/demo-rhads-kafka.git
cd demo-rhads-kafka
oc login --server=https://api.<cluster> --token=...
./install.sh
```

The playbook installs the RHADS stack **plus** AMQ Streams, Kafka Console, and Cluster Observability Operator (Perses). Typical time: **50–80 minutes** on a single-node sandbox.

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
2. Create → **Agentic Trip Planner — Quarkus Flow/Kafka** (paste MaaS API key). Optional second beat: **Go/Kafka**.
3. Dev Spaces → RHDA on `pom.xml` / `go.mod` → TPA SBOM → ACS.
4. Open Trip UI → plan a trip → Kafka Console (`flow-in` / `flow-out`) → **Approve**.
5. OpenShift console → **Observe → Dashboards (Perses)** (Kafka + trip apps).
6. Continue the RHADS promotion story (unsigned tag deny → signed commit → staging/prod) as in [docs/demo-script.md](docs/demo-script.md).

## Uninstall

```bash
./venv/bin/ansible-playbook -i ansible/inventory ansible/playbooks/uninstall.yaml
```
