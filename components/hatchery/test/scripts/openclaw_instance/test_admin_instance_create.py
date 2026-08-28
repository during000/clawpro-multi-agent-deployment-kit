#!/usr/bin/env python3
"""
Integration tests for POST /admin/instances/create.

All cases stop before createInstance invokes CVM RunInstances. They exercise
the real deployed route, authorization, strict JSON decoding, complete nested
request coverage, and secret-safe error responses without allocating a
billable CVM.
"""
import os
import sys
import time

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers.api import (
    ApiClient,
    assert_status,
    auth_test_suite,
    health_check,
    run_tests,
    seed,
)
from helpers import config


PATH = "/admin/instances/create"
MISSING_USER_ID = 4_294_967_295
DUMMY_APP_SECRET = "integration-dummy-app-secret"
DUMMY_BOT_SECRET = "integration-dummy-bot-secret"


def _safe_name(suffix):
    return (
        f"{config.INSTANCE_NAME_PREFIX}admin-create-validation-"
        f"{suffix}-{int(time.time())}-{os.getpid()}"
    )


def _full_payload():
    """Return every documented request field while targeting a missing user."""
    return {
        "user_id": MISSING_USER_ID,
        "name": _safe_name("full"),
        "group_id": 123,
        "agent_type": "openclaw",
        "role_id": 456,
        "disk_type": "CLOUD_BSSD",
        "tags": [
            {"key": "env", "value": "integration"},
            {"key": "team", "value": "clawpro"},
        ],
        "models": {
            "primary": 10,
            "fallbacks": [11, 12],
        },
        "channels": [
            {
                "channel": "feishu",
                "config": {
                    "app_id": "integration-dummy-app-id",
                    "app_secret": DUMMY_APP_SECRET,
                },
            },
            {
                "channel": "wecom",
                "config": {
                    "bot_id": "integration-dummy-bot-id",
                    "secret": DUMMY_BOT_SECRET,
                },
            },
        ],
        "skills": [
            {
                "source": "enterprise",
                "slug": "integration-dummy-skill",
                "version": "1.2.0",
            },
        ],
    }


def test_01_authentication():
    """Admin create rejects missing, invalid, and non-admin credentials."""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).post(
            PATH,
            json={
                "user_id": MISSING_USER_ID,
                "name": _safe_name("auth"),
                "agent_type": "openclaw",
            },
            expect=None,
            raw=True,
            extra_headers=headers,
        ),
        label="admin-instance-create",
    )


def test_02_invalid_json_and_unknown_fields():
    """Admin create enforces valid JSON and rejects unknown fields."""
    malformed = seed.post(
        PATH,
        data="{not-valid-json",
        expect=None,
        raw=True,
        extra_headers={"Content-Type": "application/json"},
    )
    assert_status(malformed, 400, label="admin-create-invalid-json")

    payload = {
        "user_id": MISSING_USER_ID,
        "name": _safe_name("unknown-field"),
        "agent_type": "openclaw",
        "unexpected_field": True,
    }
    unknown = seed.post(PATH, json=payload, expect=None, raw=True)
    assert_status(unknown, 400, label="admin-create-unknown-field")

    payload = {
        "user_id": MISSING_USER_ID,
        "name": _safe_name("unknown-tag-field"),
        "agent_type": "openclaw",
        "tags": [{"key": "env", "value": "prod", "scope": "instance"}],
    }
    unknown_tag = seed.post(PATH, json=payload, expect=None, raw=True)
    assert_status(unknown_tag, 400, label="admin-create-unknown-tag-field")


def test_03_required_fields():
    """Admin create validates user_id, name, and agent_type before allocation."""
    cases = [
        ({"name": _safe_name("missing-user"), "agent_type": "openclaw"}, "user_id"),
        ({"user_id": MISSING_USER_ID, "name": "", "agent_type": "openclaw"}, "name"),
        ({"user_id": MISSING_USER_ID, "name": _safe_name("missing-type")}, "agent_type"),
    ]
    for payload, label in cases:
        resp = seed.post(PATH, json=payload, expect=None, raw=True)
        assert_status(resp, 400, label=f"admin-create-{label}")


def test_04_full_nested_payload_fails_before_cvm_and_redacts_secrets():
    """A full request records nested params but a missing owner prevents CVM use."""
    resp = seed.post(PATH, json=_full_payload(), expect=None, raw=True)
    assert_status(resp, 400, label="admin-create-missing-owner-full-payload")
    assert DUMMY_APP_SECRET not in resp.text, "channel app secret leaked in error response"
    assert DUMMY_BOT_SECRET not in resp.text, "channel bot secret leaked in error response"


def test_05_method_not_allowed():
    """Admin create accepts POST only."""
    resp = seed.get(PATH, expect=None, raw=True)
    assert_status(resp, 405, label="admin-create-method")




def main():
    health_check()
    print()
    run_tests(
        globals(),
        title="test_admin_instance_create.py",
        ordered=True,
    )


if __name__ == "__main__":
    main()
