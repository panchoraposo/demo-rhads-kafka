# Default = community images so ACS / Syft show base-OS CVEs in the demo.
# Switch the pipeline to Dockerfile.ubi (Red Hat UBI) and rebuild to compare.
#
# OpenShift rootless / restricted SCC:
#   - process runs as an arbitrary UID from the namespace range (not necessarily 1001)
#   - supplementary group 0 (root) — files must be group-owned by 0 with g=u
#   - no root, no CAP_*, no writable paths required beyond /tmp
#
ARG GO_BUILDER=docker.io/library/golang:1.26.8-bookworm
ARG GO_RUNTIME=docker.io/library/debian:12.0-slim

FROM ${GO_BUILDER} AS builder
WORKDIR /src
COPY . .
# CI: Tekton rhads-build-source vendors modules via Nexus (go-group).
# Local: fall back to the default Go module proxy when vendor/ is absent.
RUN if [ -d vendor ]; then \
      CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o /tmp/trip-planner ./cmd/server; \
    else \
      CGO_ENABLED=0 GOOS=linux go build -o /tmp/trip-planner ./cmd/server; \
    fi

FROM ${GO_RUNTIME}
# Minimal OpenShift-friendly layout on Debian slim (no passwd/useradd needed).
COPY --from=builder /tmp/trip-planner /usr/local/bin/trip-planner
COPY --from=builder /src/web /web
COPY --from=builder /src/go.mod /go.mod
COPY --from=builder /src/go.sum /go.sum
RUN chgrp -R 0 /usr/local/bin/trip-planner /web /go.mod /go.sum \
    && chmod -R g=u /web \
    && chmod g=u /usr/local/bin/trip-planner /go.mod /go.sum \
    && chmod 775 /web
ENV WEB_ROOT=/web \
    HOME=/tmp \
    PORT=8080
EXPOSE 8080
# Numeric USER for local runs; OpenShift overrides with an arbitrary non-root UID.
USER 1001
ENTRYPOINT ["/usr/local/bin/trip-planner"]
