#!/usr/bin/env bash
set -euo pipefail

ctx="${OC_CONTEXT:?OC_CONTEXT required}"
ns="${VAULT_NAMESPACE:?VAULT_NAMESPACE required}"

exec_vault() {
  oc --context "$ctx" -n "$ns" exec deploy/vault -c vault -- env VAULT_ADDR=http://127.0.0.1:8200 "$@"
}

for _ in $(seq 1 36); do
  exec_vault vault status -format=json > /tmp/vault-status.json || true
  if grep -q '"initialized"' /tmp/vault-status.json 2>/dev/null; then
    break
  fi
  sleep 5
done
test -s /tmp/vault-status.json

initialized=$(python3 -c "import json; print(json.load(open('/tmp/vault-status.json')).get('initialized', False))")
ROOT=""
UNSEAL=""
if oc --context "$ctx" -n "$ns" get secret vault-init >/dev/null 2>&1; then
  ROOT=$(oc --context "$ctx" -n "$ns" get secret vault-init -o jsonpath='{.data.root_token}' | base64 -d)
  UNSEAL=$(oc --context "$ctx" -n "$ns" get secret vault-init -o jsonpath='{.data.unseal_key}' | base64 -d)
fi

if [ "$initialized" != "True" ]; then
  exec_vault vault operator init -key-shares=1 -key-threshold=1 -format=json > /tmp/vault-init.json
  ROOT=$(python3 -c "import json; print(json.load(open('/tmp/vault-init.json'))['root_token'])")
  UNSEAL=$(python3 -c "import json; print(json.load(open('/tmp/vault-init.json'))['unseal_keys_b64'][0])")
  oc --context "$ctx" -n "$ns" create secret generic vault-init \
    --from-literal=root_token="$ROOT" \
    --from-literal=unseal_key="$UNSEAL" \
    --dry-run=client -o yaml | oc --context "$ctx" apply -f -
fi

if [ -z "$ROOT" ] || [ -z "$UNSEAL" ]; then
  echo "FAIL: Vault init material is missing (secret vault/vault-init)" >&2
  exit 1
fi

is_unsealed() {
  python3 -c "import json,sys; d=json.load(open('/tmp/vault-status.json')); sys.exit(0 if d.get('sealed') is False else 1)"
}

for _ in $(seq 1 36); do
  exec_vault env VAULT_TOKEN="$ROOT" vault operator unseal "$UNSEAL" >/dev/null || true
  exec_vault vault status -format=json > /tmp/vault-status.json || true
  if is_unsealed; then
    echo "Vault is unsealed"
    exit 0
  fi
  sleep 5
done

echo "FAIL: Vault stayed sealed" >&2
exit 1
