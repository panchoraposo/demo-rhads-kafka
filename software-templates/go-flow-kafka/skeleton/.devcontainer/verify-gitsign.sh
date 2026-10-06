#!/usr/bin/env bash
set -euo pipefail
export PATH="${HOME}/.local/bin:${PATH}"

if ! command -v gitsign >/dev/null 2>&1; then
  echo "FAIL: gitsign is not installed. Run: Configure Sigstore git commit signing (gitsign + RHTAS TUF)"
  exit 1
fi

: "${GITSIGN_OIDC_ISSUER:?GITSIGN_OIDC_ISSUER is not set (devfile env)}"
GITSIGN_TUF_URL="${GITSIGN_TUF_URL:-}"
if [ -z "${GITSIGN_TUF_URL}" ] && [ -n "${CLUSTER_SUBDOMAIN:-}" ]; then
  GITSIGN_TUF_URL="https://tuf-trusted-artifact-signer.${CLUSTER_SUBDOMAIN}"
fi
IDENTITY="${GITSIGN_CERT_IDENTITY:-dev1@rhads.demo}"

if [ -n "${GITSIGN_TUF_URL}" ]; then
  ROOT_FILE="${HOME}/.sigstore-rhtas-root.json"
  if [ ! -f "${ROOT_FILE}" ]; then
    echo "Initializing RHTAS TUF root ${GITSIGN_TUF_URL}"
    curl -fsSL -k -o "${ROOT_FILE}" "${GITSIGN_TUF_URL}/root.json"
    gitsign initialize --mirror "${GITSIGN_TUF_URL}" --root "${ROOT_FILE}"
  fi
fi

echo "============================================================"
echo "[SSSC] Verify Sigstore commit signature (RHTAS TUF)"
echo "HEAD identity must be ${IDENTITY}"
echo "Issuer: ${GITSIGN_OIDC_ISSUER}"
echo "============================================================"
echo
git log -1 --pretty=full
echo
gitsign verify HEAD \
  --certificate-identity "${IDENTITY}" \
  --certificate-oidc-issuer "${GITSIGN_OIDC_ISSUER}"
echo
echo "PASS: HEAD is Sigstore-signed with Fulcio identity ${IDENTITY}."
echo "Next: git push origin main"
