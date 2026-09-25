#!/usr/bin/env bash
# Bootstrap a Python venv, install Ansible collections, then run the RHADS demo installer.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${ROOT}"

if ! command -v oc >/dev/null 2>&1; then
  echo "ERROR: oc CLI is required." >&2
  exit 1
fi

if ! oc whoami >/dev/null 2>&1; then
  echo "ERROR: oc must be logged in to the target cluster. Select your kube context, then rerun." >&2
  exit 1
fi

echo "Using oc context: $(oc config current-context)"

if ! command -v helm >/dev/null 2>&1; then
  echo "Installing Helm 3..."
  curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
fi

python3 -m venv "${ROOT}/venv"
# shellcheck disable=SC1091
source "${ROOT}/venv/bin/activate"

pip install --upgrade pip
pip install -r ansible/requirements.txt
ansible-galaxy collection install -r ansible/collections/requirements.yml

exec ansible-playbook -i ansible/inventory ansible/playbooks/install.yaml "$@"
