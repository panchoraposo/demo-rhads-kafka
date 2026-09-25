#!/usr/bin/env bash
# Materialize software-template skeletons into local/apps/ with placeholders filled.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
exec python3 - "$ROOT" <<'PY'
import os, re, shutil, sys
from pathlib import Path

ROOT = Path(sys.argv[1])
OUT = ROOT / "local" / "apps"
TEMPLATES = ROOT / "software-templates"

MAAS_BASE_URL = os.environ.get("MAAS_BASE_URL", "https://maas-rhdp.apps.maas.redhatworkshops.io/v1/")
MAAS_MODEL = os.environ.get("MAAS_MODEL", "qwen3-14b")
CLUSTER_SUBDOMAIN = os.environ.get("CLUSTER_SUBDOMAIN", "apps.local.rhads.demo")
HOST = os.environ.get("HOST", "gitlab.local.rhads.demo")

TEXT_SUFFIXES = {
    ".go", ".java", ".xml", ".properties", ".yaml", ".yml", ".md", ".html",
    ".js", ".json", ".avsc", ".env", ".mod", ".sum", ".example",
}
TEXT_NAMES = {
    "Dockerfile", "Dockerfile.ubi", ".gitignore", ".devfile.yaml",
    "go.mod", "go.sum", "mkdocs.yaml", "openapi.yaml", ".gitlab-ci.yml",
}

def is_text(p: Path) -> bool:
    if p.name in TEXT_NAMES or p.name.startswith(".env") or p.name.startswith("Dockerfile"):
        return True
    return p.suffix in TEXT_SUFFIXES

def normalize(text: str) -> str:
    text = re.sub(r"\$\{\{\s*values\.([a-zA-Z0-9_]+)\s*\|\s*dump\s*\}\}", r"${{values.\1}}", text)
    text = re.sub(r"\$\{\{\s*values\.([a-zA-Z0-9_]+)\s*\}\}", r"${{values.\1}}", text)
    # Drop simple jinja if/endif around description
    text = re.sub(r"\{%-?\s*if\s+values\.[^\n%]*%\}.*?\{%-?\s*endif\s*-?%\}", "", text, flags=re.S)
    return text

def apply(text: str, values: dict) -> str:
    text = normalize(text)
    for k, v in values.items():
        # Literal scaffolder token: ${{values.KEY}}  (not an f-string brace escape)
        text = text.replace("${{values." + k + "}}", str(v))
    return text

def materialize(name: str, skeleton: Path, values: dict) -> Path:
    dest = OUT / name
    print(f"==> Materializing {name}")
    preserved_env = None
    env_file = dest / ".env"
    if env_file.is_file():
        preserved_env = env_file.read_text(encoding="utf-8")
        print(f"    preserving existing {env_file.relative_to(OUT)}")
    if dest.exists():
        shutil.rmtree(dest)
    shutil.copytree(skeleton, dest)
    for path in dest.rglob("*"):
        if not path.is_file() or not is_text(path):
            continue
        try:
            raw = path.read_text(encoding="utf-8")
        except UnicodeDecodeError:
            continue
        path.write_text(apply(raw, values), encoding="utf-8")
    if preserved_env is not None:
        env_file.write_text(preserved_env, encoding="utf-8")
    return dest

OUT.mkdir(parents=True, exist_ok=True)

common = {
    "owner": "developers",
    "gitlab_group": "developers",
    "cluster_subdomain": CLUSTER_SUBDOMAIN,
    "host": HOST,
    "image_organization": "rhads",
    "gitops_namespace": "openshift-gitops",
    "maas_base_url": MAAS_BASE_URL,
    "maas_model": MAAS_MODEL,
}

materialize(
    "trip-quarkus",
    TEMPLATES / "quarkus-flow-kafka" / "skeleton",
    {
        **common,
        "component_id": "trip-quarkus",
        "description": "Agentic Trip Planner — LangChain4j/Kafka (local)",
        "port": "8081",
        "module_path": "com.tripplanner",
        "group_id": "com.tripplanner",
        "artifact_id": "trip-quarkus",
        "maas_api_key": "",
        "destination": "developers/trip-quarkus",
        "namespace": "trip-quarkus-dev",
        "kafka_topic": "",
    },
)

materialize(
    "trip-go",
    TEMPLATES / "go-flow-kafka" / "skeleton",
    {
        **common,
        "component_id": "trip-go",
        "description": "Agentic Trip Planner — Go/Kafka (local)",
        "port": "8082",
        "module_path": "github.com/rhads/trip-go",
        "destination": "developers/trip-go",
        "namespace": "trip-go-dev",
        "kafka_topic": "",
    },
)

materialize(
    "orders-cdc",
    TEMPLATES / "debezium-cdc-postgres" / "skeleton",
    {
        **common,
        "component_id": "orders-cdc",
        "description": "CDC — Debezium PostgreSQL (local)",
        "port": "8083",
        "module_path": "github.com/rhads/orders-cdc",
        "destination": "developers/orders-cdc",
        "namespace": "orders-cdc-dev",
        "kafka_topic": "orders-cdc.public.orders",
    },
)

# Quarkus local port
props = OUT / "trip-quarkus" / "src" / "main" / "resources" / "application.properties"
if props.exists():
    t = props.read_text(encoding="utf-8")
    if "quarkus.http.port=" not in t:
        props.write_text(t + "\nquarkus.http.port=${QUARKUS_HTTP_PORT:8081}\n", encoding="utf-8")

env_path = OUT / "trip-quarkus" / ".env"
if not env_path.exists():
    env_path.write_text(
        f"MAAS_BASE_URL={MAAS_BASE_URL}\n"
        f"MAAS_MODEL={MAAS_MODEL}\n"
        "MAAS_API_KEY=\n"
        "KAFKA_BOOTSTRAP_SERVERS=localhost:9092\n"
        "KAFKA_DEVSERVICES=false\n"
        "QUARKUS_HTTP_PORT=8081\n",
        encoding="utf-8",
    )

go_env = OUT / "trip-go" / ".env"
if not go_env.exists():
    go_env.write_text(
        f"MAAS_BASE_URL={MAAS_BASE_URL}\n"
        f"MAAS_MODEL={MAAS_MODEL}\n"
        "MAAS_API_KEY=\n"
        "KAFKA_BOOTSTRAP_SERVERS=localhost:9092\n"
        "PORT=8082\n"
        "WEB_ROOT=web\n"
        "SKILLS_DIR=skills\n",
        encoding="utf-8",
    )

print(f"\nMaterialized apps under {OUT}")
print("  trip-quarkus  → ./run-quarkus.sh")
print("  trip-go       → ./run-go.sh")
print("  orders-cdc    → ./run-cdc.sh")
PY
