#!/usr/bin/env python3
"""Copy rhads-ci Kubernetes Secrets into Vault KV v2 (secret/rhads/ci/*)."""
from __future__ import annotations

import base64
import json
import os
import subprocess
import sys

CTX = os.environ["OC_CONTEXT"]
VAULT_NS = os.environ["VAULT_NAMESPACE"]
CI_NS = os.environ["CI_NAMESPACE"]
ROOT = os.environ["VAULT_ROOT_TOKEN"]

MAPPING = {
    "gitlab-token": "rhads/ci/gitlab-token",
    "quay-dockerconfig": "rhads/ci/quay-dockerconfig",
    "cosign-signing-key": "rhads/ci/cosign-signing-key",
    "acs-ci": "rhads/ci/acs-ci",
    "tpa-oidc": "rhads/ci/tpa-oidc",
    "rhads-pipeline-env": "rhads/ci/rhads-pipeline-env",
}


def oc_json(namespace: str, args: list[str]) -> dict:
    out = subprocess.check_output(
        ["oc", "--context", CTX, "-n", namespace, *args],
        text=True,
    )
    return json.loads(out)


def vault_put(path: str, data: dict) -> None:
    # HashiCorp Vault image has the vault CLI (no curl/wget).
    # `vault write ... @-` treats "-" as a filename. KV v2 wants stdin via `-`.
    payload = json.dumps(data)
    proc = subprocess.run(
        [
            "oc",
            "--context",
            CTX,
            "-n",
            VAULT_NS,
            "exec",
            "-i",
            "deploy/vault",
            "-c",
            "vault",
            "--",
            "env",
            "VAULT_ADDR=http://127.0.0.1:8200",
            f"VAULT_TOKEN={ROOT}",
            "vault",
            "kv",
            "put",
            "-format=json",
            "-mount=secret",
            path,
            "-",
        ],
        input=payload,
        text=True,
        capture_output=True,
    )
    if proc.returncode != 0:
        sys.stderr.write(proc.stderr or proc.stdout or f"vault put {path} failed\n")
        raise SystemExit(proc.returncode)
    print(f"seeded secret/{path}", flush=True)


def decode_secret(secret: dict) -> dict:
    out: dict[str, str] = {}
    name = secret["metadata"]["name"]
    for key, raw in (secret.get("data") or {}).items():
        decoded = base64.b64decode(raw).decode("utf-8", errors="replace")
        if name == "quay-dockerconfig" and key == ".dockerconfigjson":
            out["dockerconfigjson"] = decoded
        else:
            out[key] = decoded
    return out


def main() -> None:
    for secret_name, vault_path in MAPPING.items():
        try:
            secret = oc_json(CI_NS, ["get", "secret", secret_name, "-o", "json"])
        except subprocess.CalledProcessError:
            print(f"skip missing {secret_name}", flush=True)
            continue
        data = decode_secret(secret)
        if not data:
            print(f"skip empty {secret_name}", flush=True)
            continue
        vault_put(vault_path, data)


if __name__ == "__main__":
    main()
