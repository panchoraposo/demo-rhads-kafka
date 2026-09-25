#!/usr/bin/env python3
"""Idempotent Quay org + robot bootstrap using CSRF cookie login."""
from __future__ import annotations

import json
import os
import ssl
import sys
import urllib.error
import urllib.request
from http.cookiejar import CookieJar

QUAY = os.environ["QUAY_URL"].rstrip("/")
USER = os.environ["QUAY_USER"]
PASS = os.environ["QUAY_PASS"]
ORG = os.environ["QUAY_ORG"]
ROBOT = os.environ["QUAY_ROBOT"]
CTX = ssl._create_unverified_context()
JAR = CookieJar()
OPENER = urllib.request.build_opener(
    urllib.request.HTTPSHandler(context=CTX),
    urllib.request.HTTPCookieProcessor(JAR),
)


def request(method: str, path: str, body: dict | None = None) -> tuple[int, dict | str]:
    csrf_raw = OPENER.open(urllib.request.Request(f"{QUAY}/csrf_token"), timeout=30).read()
    csrf = json.loads(csrf_raw)["csrf_token"]
    data = None if body is None else json.dumps(body).encode()
    req = urllib.request.Request(
        f"{QUAY}{path}",
        data=data,
        method=method,
        headers={
            "X-CSRF-Token": csrf,
            "X-Requested-With": "XMLHttpRequest",
            "Accept": "application/json",
            "Origin": QUAY,
            "Referer": f"{QUAY}/",
            **({"Content-Type": "application/json"} if body is not None else {}),
        },
    )
    try:
        with OPENER.open(req, timeout=30) as resp:
            raw = resp.read()
            code = resp.status
    except urllib.error.HTTPError as exc:
        raw = exc.read()
        code = exc.code
    text = raw.decode("utf-8", errors="replace")
    try:
        parsed: dict | str = json.loads(text) if text else {}
    except json.JSONDecodeError:
        parsed = text
    return code, parsed


def must_ok(code: int, allowed: set[int], action: str, payload: dict | str) -> None:
    if code not in allowed:
        raise SystemExit(f"{action} failed http={code} body={payload!r:.400}")


def main() -> None:
    code, payload = request("POST", "/api/v1/signin", {"username": USER, "password": PASS})
    must_ok(code, {200}, "signin", payload)

    code, payload = request(
        "POST",
        "/api/v1/organization/",
        {"name": ORG, "email": f"{ORG}@rhads.demo"},
    )
    must_ok(code, {200, 201, 400, 409}, "create org", payload)

    code, payload = request(
        "PUT",
        f"/api/v1/organization/{ORG}/robots/{ROBOT}",
        {"description": "RHADS pipeline robot"},
    )
    must_ok(code, {200, 201, 400}, "create robot", payload)
    robot = payload if isinstance(payload, dict) else {}
    if not robot.get("token"):
        code, payload = request("GET", f"/api/v1/organization/{ORG}/robots/{ROBOT}")
        must_ok(code, {200}, "get robot", payload)
        robot = payload if isinstance(payload, dict) else {}

    code, payload = request(
        "PUT",
        f"/api/v1/organization/{ORG}/team/pipeline-creators?includeMembers=false",
        {"name": "pipeline-creators", "role": "creator"},
    )
    must_ok(code, {200, 201, 400, 409}, "create team", payload)

    code, payload = request(
        "PUT",
        f"/api/v1/organization/{ORG}/team/pipeline-creators/members/{ORG}+{ROBOT}",
        {},
    )
    must_ok(code, {200, 201, 204, 400, 409}, "add robot to team", payload)

    name = robot.get("name") or f"{ORG}+{ROBOT}"
    token = robot.get("token") or ""
    if not token:
        raise SystemExit(f"robot token missing: {robot!r:.400}")
    json.dump({"username": name, "token": token}, sys.stdout)
    sys.stdout.write("\n")


if __name__ == "__main__":
    main()
