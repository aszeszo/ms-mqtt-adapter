#!/usr/bin/env -S uv run --quiet --script
# /// script
# dependencies = ["websocket-client", "requests"]
# ///
"""Validate and upload ms-mqtt-adapter config.yaml using add-on ingress APIs."""

import json
import os
import sys
import time
from pathlib import Path
from typing import Any, Dict, List, Optional

import requests
from websocket import create_connection

HASS_SERVER = os.getenv("HASS_SERVER", "").rstrip("/")
HASS_TOKEN = os.getenv("HASS_TOKEN", "")
CONFIG_FILE = Path(os.getenv("MS_MQTT_ADAPTER_CONFIG", "config.yaml"))
ADDON_SLUG_OVERRIDE = os.getenv("MS_MQTT_ADAPTER_SLUG", "").strip()
POLL_ATTEMPTS = 10
POLL_INTERVAL_SECONDS = 0.5

if not HASS_SERVER or not HASS_TOKEN:
    sys.exit("Error: HASS_SERVER and HASS_TOKEN must be set")

if not CONFIG_FILE.is_file():
    sys.exit(f"Error: config file not found: {CONFIG_FILE}")


def normalize_text(content: str) -> str:
    """Normalize text for stable idempotency checks."""
    return content.replace("\r\n", "\n").replace("\r", "\n").strip() + "\n"


desired_config = CONFIG_FILE.read_text()
desired_config_normalized = normalize_text(desired_config)

ws_url = HASS_SERVER.replace("http://", "ws://").replace("https://", "wss://")
ws_url = f"{ws_url}/api/websocket"

print("Connecting to Home Assistant WebSocket...")
ws = create_connection(ws_url)
msg_id = 1


def call_supervisor(
    *, endpoint: str, method: str, data: Optional[Dict[str, Any]] = None, timeout=None
) -> Dict[str, Any]:
    """Send supervisor/api command and return result payload."""
    global msg_id
    payload: Dict[str, Any] = {
        "id": msg_id,
        "type": "supervisor/api",
        "endpoint": endpoint,
        "method": method,
        "timeout": timeout,
    }
    if data is not None:
        payload["data"] = data

    ws.send(json.dumps(payload))
    expected_id = msg_id
    msg_id += 1
    result = json.loads(ws.recv())
    if result.get("id") != expected_id or not result.get("success"):
        sys.exit(f"Supervisor call failed ({endpoint}): {result}")
    return result["result"]


def find_addon_slug(installed_addons: List[Dict[str, Any]]) -> str:
    if ADDON_SLUG_OVERRIDE:
        return ADDON_SLUG_OVERRIDE

    preferred = ["ms_mqtt_adapter"]
    slugs = [addon.get("slug", "") for addon in installed_addons]

    for slug in preferred:
        if slug in slugs:
            return slug

    matches = [slug for slug in slugs if slug.endswith("ms_mqtt_adapter")]
    if matches:
        return sorted(matches)[0]

    sys.exit(
        "Could not find installed ms-mqtt-adapter add-on. "
        "Set MS_MQTT_ADAPTER_SLUG if the slug is custom."
    )


try:
    auth_required = json.loads(ws.recv())
    if auth_required.get("type") != "auth_required":
        sys.exit(f"Unexpected message: {auth_required}")

    ws.send(json.dumps({"type": "auth", "access_token": HASS_TOKEN}))
    auth_result = json.loads(ws.recv())
    if auth_result.get("type") != "auth_ok":
        sys.exit(f"Authentication failed: {auth_result}")

    print("✓ Authenticated")

    installed_addons = call_supervisor(endpoint="/addons", method="get")["addons"]
    addon_slug = find_addon_slug(installed_addons)
    print(f"Using add-on slug: {addon_slug}")

    addon_info = call_supervisor(endpoint=f"/addons/{addon_slug}/info", method="get")

    if addon_info.get("state") != "started":
        print("Starting add-on to access ingress config API...")
        call_supervisor(
            endpoint=f"/addons/{addon_slug}/start", method="post", timeout=120
        )
        print("✓ Add-on started")
        addon_info = call_supervisor(endpoint=f"/addons/{addon_slug}/info", method="get")

    ingress_url = addon_info.get("ingress_url")
    if not ingress_url:
        sys.exit("Add-on does not expose ingress_url; cannot configure via add-on APIs.")

    ingress_session = call_supervisor(endpoint="/ingress/session", method="post").get(
        "session"
    )
    if not ingress_session:
        sys.exit("Failed to create ingress session.")

    cookies = {"ingress_session": ingress_session}
    ingress_base = f"{HASS_SERVER}{ingress_url.rstrip('/')}"

    def ingress_get_raw() -> str:
        resp = requests.get(f"{ingress_base}/api/config/raw", cookies=cookies, timeout=30)
        if resp.status_code != 200:
            sys.exit(f"Failed to fetch current add-on config: HTTP {resp.status_code} {resp.text}")
        try:
            return resp.content.decode("utf-8")
        except UnicodeDecodeError as exc:
            sys.exit(f"Failed to decode add-on config as UTF-8: {exc}")

    def ingress_validate(config_text: str) -> Dict[str, Any]:
        resp = requests.post(
            f"{ingress_base}/api/config/validate",
            cookies=cookies,
            headers={"Content-Type": "text/yaml"},
            data=config_text,
            timeout=30,
        )
        if resp.status_code != 200:
            sys.exit(f"Validation request failed: HTTP {resp.status_code} {resp.text}")
        try:
            return resp.json()
        except ValueError:
            sys.exit(f"Validation returned non-JSON response: {resp.text!r}")

    def ingress_put_raw(config_text: str) -> Dict[str, Any]:
        resp = requests.put(
            f"{ingress_base}/api/config/raw",
            cookies=cookies,
            headers={"Content-Type": "text/yaml"},
            data=config_text,
            timeout=30,
        )
        if resp.status_code != 200:
            sys.exit(f"Upload request failed: HTTP {resp.status_code} {resp.text}")
        try:
            return resp.json()
        except ValueError:
            sys.exit(f"Upload returned non-JSON response: {resp.text!r}")

    current_config = ingress_get_raw()
    current_config_normalized = normalize_text(current_config)

    if current_config_normalized == desired_config_normalized:
        print("✓ Add-on config already matches local config.yaml")
        print("\n✓ ms-mqtt-adapter configuration is already up to date")
        sys.exit(0)

    print("Validating config.yaml with add-on API...")
    validation = ingress_validate(desired_config)
    if not validation.get("valid", False):
        error = validation.get("error") or validation.get("message") or str(validation)
        sys.exit(f"Config validation failed: {error}")
    print("✓ Validation passed")

    print("Uploading config.yaml using add-on API...")
    upload = ingress_put_raw(desired_config)
    if upload.get("status") != "ok":
        sys.exit(f"Upload failed: {upload}")

    reflected = False
    for _ in range(POLL_ATTEMPTS):
        read_back = ingress_get_raw()
        if normalize_text(read_back) == desired_config_normalized:
            reflected = True
            break
        time.sleep(POLL_INTERVAL_SECONDS)

    if not reflected:
        sys.exit(
            "Upload request completed, but add-on config did not reflect local config.yaml."
        )

    print("✓ Uploaded config.yaml to add-on")
    print("\n✓ ms-mqtt-adapter configuration updated")
finally:
    ws.close()
