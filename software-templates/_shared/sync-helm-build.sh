#!/usr/bin/env bash
# Copy shared build pipeline templates into every software template.
# Run after editing software-templates/_shared/helm-build/templates/.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)/software-templates"
SHARED="$ROOT/_shared/helm-build/templates"
FILES=(pipeline-build.yaml triggertemplates.yaml)
TEMPLATES=(quarkus-flow-kafka go-flow-kafka debezium-cdc-postgres)
for t in "${TEMPLATES[@]}"; do
  dest="$ROOT/$t/manifests/helm/build/templates"
  for f in "${FILES[@]}"; do
    cp "$SHARED/$f" "$dest/$f"
    echo "synced $t/$f"
  done
done
