# Live demo script (~40–45 min) — RHADS + Kafka + CDC

Shared password: `backstage`. Keep `https://dashboard.apps.<cluster>/` open.

## 0. Context (2 min)

**Software supply chain** (who built what, with which dependencies, signed by whom, promoted under which policy) **plus event-driven agentic HITL** over Kafka **plus Change Data Capture** with Debezium.

- Developer Hub catalogs **three** templates: LangChain4j/Kafka, Go/Kafka, and Debezium CDC/Postgres.
- Quarkus trip planner uses the **Red Hat langchain4j** BOM on Red Hat build of Quarkus 3.33; HITL is Kafka CloudEvents.
- Streams for Apache Kafka carries CloudEvents (`flow-in` / `flow-out`); the Kafka Console shows them live.
- **Apicurio Registry** holds trip and CDC schemas; **Vault** secrets appear on each app Overview card.
- Shared **Red Hat build of Debezium** (`rhads-debezium`) — templates only create connectors + DB.
- Perses (Cluster Observability Operator) shows Kafka/app dashboards in the OpenShift console.

## 1. Portal — create the Quarkus app (4 min)

1. Developer Hub → login as **`dev1`**.
2. Create → **Agentic Trip Planner — LangChain4j/Kafka** (`trip-quarkus`, ≤18 chars). Paste the MaaS API key.
3. Wait for source + gitops repos and Argo apps (`trip-quarkus-build|dev|staging|prod`).
4. Bootstrap Jobs seed Vault/ESO (MaaS), Quay repo, and the first PipelineRun (first SBOM in TPA). Do **not** tag the unsigned scaffold yet.
5. Catalog → component Overview: **Vault** card (`apps/trip-quarkus`) and Apicurio schema link.

## 2. Inner loop — RHDA / TPA / ACS (6 min)

1. Catalog → **OpenShift Dev Spaces**. Prefer a workspace already Running.
2. Command palette → **Red Hat Dependency Analytics** on `pom.xml` (commons-text 1.9 / snakeyaml 1.33).
3. **Trusted Profile Analyzer** — SBOM from the first build.
4. Optionally bump CVE deps (do not delete them) after the unsigned-tag demo beat later.
5. **Configure Sigstore git commit signing** when ready for the signed promotion path.

## 3. Kafka HITL — plan and approve (8 min)

1. Open **Trip Planner UI (dev)** from the catalog (or the Route).
2. Submit a family trip (future start date). `POST /trip/plan` returns when status is `awaiting_approval` — the HTTP call finishes; the workflow waits on Kafka.
3. **Kafka Console** (`https://kafka-console.apps.<cluster>/`):
   - Topic `flow-in`: `com.tripplanner.trip.requested`
   - Topic `flow-out`: `com.tripplanner.trip.approval.requested` with `flowinstanceid`
4. Refresh the browser — same plan and instance id (in-memory while the pod lives).
5. Click **Approve Trip** → decision on `flow-in` (`approval.done`) → `booking.finalized` on `flow-out` with a simulated `MOS-…` reference.
6. Optional: plan another trip and **Reject** — no booking event.

## 4. CDC — Debezium PostgreSQL (6 min)

1. Create → **CDC — Debezium PostgreSQL** (`orders-cdc`).
2. Argo apps: `orders-cdc-build`, `orders-cdc-cdc` (Postgres + connector Job), `orders-cdc-dev|staging|prod`.
3. Catalog → Overview: Vault card + API entity **order-change** → Apicurio UI.
4. Open **CDC Order Desk** (dev Route).
5. Click **New order** — pipeline strip pulses; event feed shows `CREATE` with capture latency.
6. Select the row → **Mark paid** — `UPDATE` with expandable before → after.
7. Optional: Kafka Console topic `orders-cdc.public.orders`; Apicurio artifact from the header link.

No debug-pod SQL on stage — the UI writes Postgres and Debezium captures it.

## 5. Perses observability (3 min)

OpenShift console → **Observe → Dashboards (Perses)** → project `rhads-observability`:

- **RHADS Kafka — Trip Planner events**
- **RHADS Trip Planner — app health**

Correlate message rates while approving a trip.

## 6. Supply-chain promotion (10 min)

Same RHADS beats as the base demo:

1. Pipeline table: clone → gitsign → Maven/Go → OpenShift Build → SBOM → cosign → ACS → TPA → Conforma (dev) → GitOps → Chains.
2. GitLab **tag** `v1.0.0` on the **unsigned** scaffold → staging Conforma STRICT **denies** `rhads_source.git_commit_signed`.
3. Dev Spaces: signed commit (optionally with CVE bumps) → push → new build.
4. Tag `v1.0.1` on signed commit → staging; **release** → production.

## 7. Optional — Go template (5–8 min)

Create → **Agentic Trip Planner — Go/Kafka** (`trip-go`).

- RHDA on `go.mod` (jwt-go, gorilla/websocket, yaml.v2, x/net — community module CVEs).
- First image uses **community** bases (`golang:1.21.0-bookworm` + `debian:12.0-slim`) → ACS shows base-OS CVEs.
- Switch to Red Hat: `cp Dockerfile.ubi Dockerfile`, commit, push → rebuild → ACS/TPA show the UBI difference.
- Same Trip UI / Kafka topics / Approve path (hand-rolled workflow engine, same CloudEvents types).

## Cheat sheet

| URL | Purpose |
| --- | --- |
| Developer Hub | Scaffold trip + CDC templates; Vault + Apicurio on Overview |
| Trip UI / CDC Order Desk | Plan / approve; create / pay / cancel orders live |
| Apicurio | `https://apicurio.apps.<cluster>/` schemas |
| Vault | Secrets under `secret/apps/{app}` |
| Kafka Console | `flow-in` / `flow-out` / `{app}.public.orders` |
| Observe → Perses | Kafka + app dashboards |
| TPA / ACS / Quay / Rekor | Supply chain evidence |
