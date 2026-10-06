#!/usr/bin/env bash
set -euo pipefail
echo "============================================================"
echo "[SSSC] Dev Spaces: Sigstore commit signing"
echo "gitsign uses the same Fulcio + Rekor + TUF trust root as cosign."
echo "============================================================"
mkdir -p "${HOME}/.local/bin"
export PATH="${HOME}/.local/bin:${PATH}"

: "${GITSIGN_FULCIO_URL:?GITSIGN_FULCIO_URL is not set (devfile env)}"
: "${GITSIGN_REKOR_URL:?GITSIGN_REKOR_URL is not set (devfile env)}"
: "${GITSIGN_OIDC_ISSUER:?GITSIGN_OIDC_ISSUER is not set (devfile env)}"
GITSIGN_TUF_URL="${GITSIGN_TUF_URL:-}"
if [ -z "${GITSIGN_TUF_URL}" ] && [ -n "${CLUSTER_SUBDOMAIN:-}" ]; then
  GITSIGN_TUF_URL="https://tuf-trusted-artifact-signer.${CLUSTER_SUBDOMAIN}"
fi
: "${GITSIGN_TUF_URL:?GITSIGN_TUF_URL is not set (devfile env)}"
IDENTITY="${GITSIGN_CERT_IDENTITY:-dev1@rhads.demo}"

if ! command -v gitsign >/dev/null 2>&1; then
  echo "Installing gitsign into ~/.local/bin ..."
  curl -fsSL -o "${HOME}/.local/bin/gitsign" \
    "https://github.com/sigstore/gitsign/releases/download/v0.13.0/gitsign_0.13.0_linux_amd64"
  chmod +x "${HOME}/.local/bin/gitsign"
fi

git config --global --add safe.directory "*" || true
git config --global user.name "${GITSIGN_GIT_NAME:-dev1}"
git config --global user.email "${IDENTITY}"
git config --global gpg.x509.program gitsign
git config --global gpg.format x509
git config --global commit.gpgsign true
git config --global tag.gpgsign true
git config --global gitsign.fulcio "${GITSIGN_FULCIO_URL}"
git config --global gitsign.rekor "${GITSIGN_REKOR_URL}"
git config --global gitsign.issuer "${GITSIGN_OIDC_ISSUER}"
git config --global gitsign.clientid "${GITSIGN_OIDC_CLIENT_ID:-trusted-artifact-signer}"
git config --global gitsign.tufmirror "${GITSIGN_TUF_URL}"

echo "Initializing RHTAS TUF root ${GITSIGN_TUF_URL}"
ROOT_FILE="${HOME}/.sigstore-rhtas-root.json"
curl -fsSL -k -o "${ROOT_FILE}" "${GITSIGN_TUF_URL}/root.json"
gitsign initialize --mirror "${GITSIGN_TUF_URL}" --root "${ROOT_FILE}"

if ! grep -q '.local/bin' "${HOME}/.bashrc" 2>/dev/null; then
  echo 'export PATH="${HOME}/.local/bin:${PATH}"' >> "${HOME}/.bashrc"
fi

echo
echo "gitsign=$(command -v gitsign)"
echo "Fulcio: ${GITSIGN_FULCIO_URL}"
echo "Rekor:  ${GITSIGN_REKOR_URL}"
echo "Issuer: ${GITSIGN_OIDC_ISSUER}"
echo "TUF:    ${GITSIGN_TUF_URL}"
echo "Identity: ${IDENTITY}"
echo
echo "PASS: workspace is ready to Sigstore-sign commits."
echo
echo "Demo next steps:"
echo "  1. Edit a file (tool description or index.html)."
echo "  2. Terminal:  git add -A && git commit -m \"demo: signed change from Dev Spaces\""
echo "  3. xdg-open is missing in Dev Spaces — copy the printed URL, log in as"
echo "     ${GITSIGN_GIT_NAME:-dev1} / backstage, paste the verification code."
echo "  4. Command palette → Verify Sigstore-signed HEAD (gitsign + RHTAS TUF)"
echo "  5. Terminal:  git push origin main"
echo "The pipeline task gitsign-verify checks this signature before OpenShift Builds."
