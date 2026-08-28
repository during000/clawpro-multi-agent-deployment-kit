#!/usr/bin/env python3
"""Passwordless login integration tests against a deployed Hatchery instance.

Required environment:
    API                     Backend base URL
    ADMIN_TOKEN             Administrator user's API token
    TOKEN                   Regular user's API token for authorization checks

The target tenant must already have a feature_allowlists row with
(type='passwordless-login', identifier=IDENTIFIER), and its trusted domain must
be an absolute HTTPS URL.
"""

import os
import sys
import time
from urllib.parse import parse_qs, urlparse

# The one-use token is intentionally never emitted in test logs or generated cURL.
os.environ.setdefault("QUIET", "1")

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers.api import (  # noqa: E402
    ApiClient,
    IDENTIFIER,
    anon,
    cleanup_users_by_prefix,
    extract_uid,
    health_check,
    run_tests,
    seed,
)

PREFIX = f"it-passwordless-{IDENTIFIER or int(time.time())}"
TARGET_USERNAME = f"{PREFIX}-target"
TARGET_PASSWORD = "Passwordless-IT-123!"
STATE = {}




def test_01_issue_and_complete_login():
    """An administrator API token issues a link; token establishes target session."""
    created = seed.post(
        "/admin/create",
        json={
            "username": TARGET_USERNAME,
            "password": TARGET_PASSWORD,
            "role": "user",
            "instance_quota": 0,
        },
    )
    target_id = extract_uid(created)
    assert target_id, f"missing target user id: {created}"

    issued = seed.post(
        "/admin/passwordless-login-link",
        json={"user_id": target_id},
    )
    assert issued.get("expires_in") == 120, issued
    assert issued.get("expires_at"), issued

    parsed = urlparse(issued.get("link", ""))
    assert parsed.scheme == "https", issued
    assert parsed.path == "/passwordless-login", issued
    token = parse_qs(parsed.fragment).get("passwordless_login_token", [""])[0]
    assert len(token) == 43, "issued token must be 43 URL-safe characters"

    consumed = anon.post(
        "/auth/passwordless-login",
        json={"token": token},
        raw=True,
    )
    payload = consumed.json()
    assert payload.get("ok") is True, payload
    assert payload.get("redirect") == "/", payload
    assert payload.get("role") == "user", payload
    session_cookies = consumed.cookies.get_dict()
    assert "hatchery-session" in session_cookies, session_cookies

    checked = anon.get("/", raw=True, cookies=session_cookies)
    checked_payload = checked.json()
    assert checked_payload.get("username") == TARGET_USERNAME, checked_payload
    assert checked_payload.get("role") == "user", checked_payload

    STATE.update(
        target_id=target_id,
        token=token,
        session_cookies=session_cookies,
    )


def test_02_replay_is_rejected():
    """The same token cannot establish a second session."""
    replay = anon.post(
        "/auth/passwordless-login",
        json={"token": STATE["token"]},
        expect=401,
        raw=True,
    )
    assert replay.json().get("error"), replay.text

    checked = anon.get("/", raw=True, cookies=STATE["session_cookies"])
    assert checked.json().get("username") == TARGET_USERNAME, checked.text


def test_03_regular_user_api_token_cannot_issue():
    """A regular user's API token cannot invoke an administrator API."""
    regular_user_token = os.environ.get("TOKEN", "")
    assert regular_user_token, "TOKEN is required"
    denied = ApiClient(regular_user_token).post(
        "/admin/passwordless-login-link",
        json={"user_id": STATE["target_id"]},
        expect=403,
        raw=True,
    )
    assert denied.json().get("error"), denied.text


def test_04_required_parameters_are_validated():
    """Both new request bodies reject missing required fields."""
    missing_user = seed.post(
        "/admin/passwordless-login-link",
        json={},
        expect=400,
        raw=True,
    )
    assert missing_user.json().get("error"), missing_user.text

    missing_token = anon.post(
        "/auth/passwordless-login",
        json={},
        expect=400,
        raw=True,
    )
    assert missing_token.json().get("error"), missing_token.text


def test_05_forged_token_does_not_create_session():
    """A well-shaped but forged token returns 401 and no authenticated state."""
    forged = anon.post(
        "/auth/passwordless-login",
        json={"token": "A" * 43},
        expect=401,
        raw=True,
    )
    assert forged.json().get("error"), forged.text
    assert "hatchery-session" not in forged.cookies.get_dict(), forged.cookies

    checked = anon.get("/", raw=True)
    payload = checked.json()
    assert payload.get("authenticated") is False, payload


def cleanup():
    cleanup_users_by_prefix(PREFIX)


def main():
    health_check()
    try:
        run_tests(
            globals(),
            title="Passwordless login",
            ordered=True,
            abort_on_fail=True,
        )
    finally:
        cleanup()


if __name__ == "__main__":
    main()
