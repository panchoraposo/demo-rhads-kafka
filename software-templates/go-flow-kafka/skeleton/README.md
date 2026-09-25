# ${{values.component_id}} — Agentic Trip Planner (Go + Kafka)

Same demo narrative as **Quarkus Flow/Kafka**: plan a trip, pause for human approval over Kafka CloudEvents (`flow-in` / `flow-out`), approve or reject without holding the planning HTTP request.

## Demo beats

1. RHDA on `go.mod` (intentional community module CVEs: jwt-go, gorilla/websocket, yaml.v2, x/net).
2. First pipeline build uses **community** images (`golang:1.21.0-bookworm` + `debian:12.0-slim`) → ACS / Syft show base-OS CVEs.
3. Switch runtime to Red Hat UBI (`Dockerfile.ubi`) → rebuild → compare ACS findings.
4. Open Trip UI → Plan → Kafka Console → Approve.

### Community → Red Hat base image

| File | Builder | Runtime |
| --- | --- | --- |
| `Dockerfile` (default) | `docker.io/library/golang:1.21.0-bookworm` | `docker.io/library/debian:12.0-slim` |
| `Dockerfile.ubi` | `registry.access.redhat.com/ubi9/go-toolset:1.21` | `registry.access.redhat.com/ubi9/ubi-minimal:9.4` |

Both images follow OpenShift **restricted** SCC (arbitrary non-root UID, files group-owned by `0` with `g=u`, `HOME=/tmp`). No `anyuid` SCC required.

In Dev Spaces (or a local edit), point the build Dockerfile at UBI and push:

```bash
# Option A — replace default Dockerfile
cp Dockerfile.ubi Dockerfile && git add Dockerfile && git commit -m "Switch runtime to Red Hat UBI" && git push

# Option B — keep both; set dockerfile in the gitops build values / Argo helm values
#   image.dockerfile: ./Dockerfile.ubi
```

## Configuration

| Env | Default |
| --- | --- |
| `KAFKA_BOOTSTRAP_SERVERS` | `rhads-kafka-kafka-bootstrap.kafka.svc:9092` |
| `MAAS_BASE_URL` / `MAAS_API_KEY` / `MAAS_MODEL` | From Vault/ESO |
| `GOPROXY` (Dev Spaces / Tekton) | Nexus `go-group` |

Shared password for platform users: `backstage`.
