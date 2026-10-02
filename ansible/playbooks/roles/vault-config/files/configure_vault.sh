#!/usr/bin/env bash
set -euo pipefail

ctx="${OC_CONTEXT:?OC_CONTEXT required}"
ns="${VAULT_NAMESPACE:?VAULT_NAMESPACE required}"
admin_pass="${VAULT_ADMIN_PASSWORD:?VAULT_ADMIN_PASSWORD required}"

ROOT=$(oc --context "$ctx" -n "$ns" get secret vault-init -o jsonpath='{.data.root_token}' | base64 -d)

exec_vault() {
  oc --context "$ctx" -n "$ns" exec deploy/vault -c vault -- env VAULT_ADDR=http://127.0.0.1:8200 VAULT_TOKEN="$ROOT" "$@"
}

exec_vault vault secrets enable -path=secret kv-v2 || true
exec_vault vault auth enable kubernetes || true
exec_vault vault write auth/kubernetes/config \
  kubernetes_host=https://kubernetes.default.svc \
  kubernetes_ca_cert=@/var/run/secrets/kubernetes.io/serviceaccount/ca.crt \
  disable_iss_validation=true

cat <<'EOF' | oc --context "$ctx" -n "$ns" exec -i deploy/vault -c vault -- env VAULT_ADDR=http://127.0.0.1:8200 VAULT_TOKEN="$ROOT" vault policy write eso -
path "secret/data/rhads/*" {
  capabilities = ["read"]
}
path "secret/metadata/rhads/*" {
  capabilities = ["read", "list"]
}
path "secret/data/apps/*" {
  capabilities = ["read"]
}
path "secret/metadata/apps/*" {
  capabilities = ["read", "list"]
}
EOF

cat <<'EOF' | oc --context "$ctx" -n "$ns" exec -i deploy/vault -c vault -- env VAULT_ADDR=http://127.0.0.1:8200 VAULT_TOKEN="$ROOT" vault policy write app-seed -
path "secret/data/apps/*" {
  capabilities = ["create", "update", "read"]
}
path "secret/metadata/apps/*" {
  capabilities = ["create", "update", "read", "list"]
}
EOF

cat <<'EOF' | oc --context "$ctx" -n "$ns" exec -i deploy/vault -c vault -- env VAULT_ADDR=http://127.0.0.1:8200 VAULT_TOKEN="$ROOT" vault policy write demo-admin -
path "secret/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}
path "sys/mounts" {
  capabilities = ["read"]
}
path "auth/token/lookup-self" {
  capabilities = ["read"]
}
EOF

exec_vault vault write auth/kubernetes/role/eso \
  bound_service_account_names=external-secrets \
  bound_service_account_namespaces=external-secrets \
  policies=eso \
  ttl=1h

exec_vault vault write auth/kubernetes/role/app-seed \
  bound_service_account_names=pipeline \
  bound_service_account_namespaces='*' \
  policies=app-seed \
  ttl=15m

cat <<'EOF' | oc --context "$ctx" -n "$ns" exec -i deploy/vault -c vault -- env VAULT_ADDR=http://127.0.0.1:8200 VAULT_TOKEN="$ROOT" vault policy write rhdh-vault-list -
path "secret/metadata/apps/*" {
  capabilities = ["list", "read"]
}
path "secret/metadata/*" {
  capabilities = ["list"]
}
path "secret/data/apps/*" {
  capabilities = ["read"]
}
path "auth/token/renew-self" {
  capabilities = ["update"]
}
path "auth/token/lookup-self" {
  capabilities = ["read"]
}
EOF

# Static token for Developer Hub Vault plugin (LIST + read metadata only).
# Parse with python: vault JSON nests client_token under auth, and sed is brittle.
HUB_TOKEN=$(oc --context "$ctx" -n "$ns" exec deploy/vault -c vault -- \
  env VAULT_ADDR=http://127.0.0.1:8200 VAULT_TOKEN="$ROOT" \
  vault token create -policy=rhdh-vault-list -ttl=768h -format=json \
  | python3 -c 'import json,sys; print(json.load(sys.stdin)["auth"]["client_token"])')
if [ -n "${HUB_TOKEN}" ]; then
  oc --context "$ctx" -n "$ns" create secret generic vault-rhdh-token \
    --from-literal=token="${HUB_TOKEN}" \
    --dry-run=client -o yaml | oc --context "$ctx" -n "$ns" apply -f -
  echo "PASS: vault-rhdh-token secret created"
else
  echo "ERROR: could not create Hub Vault token" >&2
  exit 1
fi

exec_vault vault auth enable userpass || true
exec_vault vault write auth/userpass/users/vaultadmin \
  password="$admin_pass" \
  policies=demo-admin
