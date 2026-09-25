#!/usr/bin/env python3
"""Register OpenShift's Camel runtime icon in the RHDH Topology plugin.

The community topology plugin (2.12.x) maps app.openshift.io/runtime to a
fixed set of webpack SVG assets. Camel is in the OpenShift console set but
not in that plugin, so Hub falls back to icon-default (generic graph).
"""
from __future__ import annotations

import glob
import os
import shutil
import sys

SVG_CANDIDATES = (
    "/opt/app-root/src/camel.svg",
    os.path.join(os.path.dirname(__file__), "camel.svg"),
)
PLUGIN_STATIC = (
    "/opt/app-root/src/dynamic-plugins-root/"
    "backstage-community-plugin-topology-*/dist-scalprum/static"
)
MARKER = '.set("icon-camel"'
NEEDLE = '.set("icon-quarkus",'
REPLACEMENT = '.set("icon-camel",n.p+"static/camel.svg").set("icon-quarkus",'


def main() -> int:
    svg = next((p for p in SVG_CANDIDATES if os.path.isfile(p)), None)
    if not svg:
        print("camel.svg not found; skip")
        return 0
    patched = 0
    for static_dir in glob.glob(PLUGIN_STATIC):
        shutil.copy(svg, os.path.join(static_dir, "camel.svg"))
        for js in glob.glob(os.path.join(static_dir, "*.js")):
            if "syntax" in js or "highlight" in js:
                continue
            with open(js, "r", encoding="utf-8", errors="ignore") as fh:
                text = fh.read()
            if NEEDLE not in text:
                continue
            if MARKER in text:
                print(f"already patched {js}")
                patched += 1
                continue
            updated = text.replace(NEEDLE, REPLACEMENT, 1)
            if updated == text:
                continue
            with open(js, "w", encoding="utf-8") as fh:
                fh.write(updated)
            print(f"patched {js}")
            patched += 1
    print(f"done patched={patched}")
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except Exception as exc:  # never fail the Hub container
        print(f"inject-camel-topology-icon skipped: {exc}", file=sys.stderr)
        sys.exit(0)
