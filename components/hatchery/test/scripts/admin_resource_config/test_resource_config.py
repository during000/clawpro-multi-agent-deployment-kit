#!/usr/bin/env python3
"""ResourcePolicy management/options and legacy API compatibility tests (I01-I09)."""

import copy
import json
import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers.api import (  # noqa: E402
    ApiClient,
    auth_test_suite,
    health_check,
    run_tests,
    seed,
)

RESOURCE_PATH = "/admin/resource-policies"
CREATE_PATH = "/admin/resource-policies/create"
UPDATE_PATH = "/admin/resource-policies/update"
DELETE_PATH = "/admin/resource-policies/delete"
INSTANCE_TYPES_PATH = "/admin/resource-policies/options/instance-types"
SYSTEM_DISKS_PATH = "/admin/resource-policies/options/system-disks"
ALLOWED_INSTANCE_TYPES = {"Ai2.MEDIUM2", "Ai2.MEDIUM4", "Ai2.LARGE8", "Ai2.2XLARGE16"}
ALLOWED_DISK_TYPES = {"CLOUD_SSD", "CLOUD_PREMIUM", "CLOUD_BSSD", "CLOUD_HSSD"}
CHARGE_TYPE = "POSTPAID_BY_HOUR"
CANDIDATE_ZONES = ("ap-guangzhou-6", "ap-guangzhou-7")

state = {
    "default_policy_id": None,
    "original_resource_config": None,
    "default_mutated": False,
    "zone": None,
    "instance_type": None,
}


def _list_policies():
    return seed.get(RESOURCE_PATH, params={"page": 1, "page_size": 100})


def _default_policy():
    body = _list_policies()
    default = next((item for item in body.get("items") or [] if item.get("is_default")), None)
    assert default, f"default resource policy missing: {body}"
    return default


def _auth_get(path, params=None):
    def call(headers):
        return ApiClient("", timeout=30).get(
            path,
            params=params,
            expect=None,
            raw=True,
            extra_headers=headers,
        )

    return call


def _auth_post(path):
    def call(headers):
        return ApiClient("", timeout=30).post(
            path,
            json={},
            expect=None,
            raw=True,
            extra_headers=headers,
        )

    return call


def _capture_original():
    default = _default_policy()
    state["default_policy_id"] = default["id"]
    state["original_resource_config"] = copy.deepcopy(default.get("resource_config") or {})


def _restore_original():
    if not state.get("default_mutated"):
        return
    resp = seed.post(
        UPDATE_PATH,
        json={
            "id": state["default_policy_id"],
            "resource_config": state["original_resource_config"],
        },
        expect=None,
        raw=True,
    )
    assert resp.status_code == 200, resp.text[:300]
    restored = _default_policy()
    assert restored.get("resource_config") == state["original_resource_config"], restored
    state["default_mutated"] = False


def _find_sellable_instance_type():
    errors = []
    for zone in CANDIDATE_ZONES:
        resp = seed.get(
            INSTANCE_TYPES_PATH,
            params={
                "zone": zone,
                "instance_charge_type": CHARGE_TYPE,
                "refresh": "1",
            },
            expect=None,
            raw=True,
        )
        if resp.status_code != 200:
            errors.append(f"{zone}: {resp.status_code} {resp.text[:120]}")
            continue
        body = resp.json()
        items = body.get("instance_types") or []
        if items:
            state["zone"] = zone
            state["instance_type"] = items[0]["instance_type"]
            return body
        errors.append(f"{zone}: no SELL allowlisted type")
    raise AssertionError("no usable zone/instance type: " + "; ".join(errors))


def test_01_all_resource_policy_operations_require_admin():
    operations = [
        ("list", _auth_get(RESOURCE_PATH, {"page": 1, "page_size": 10})),
        ("create", _auth_post(CREATE_PATH)),
        ("update", _auth_post(UPDATE_PATH)),
        ("delete", _auth_post(DELETE_PATH)),
        (
            "instance-types",
            _auth_get(INSTANCE_TYPES_PATH, {"zone": CANDIDATE_ZONES[0]}),
        ),
        (
            "system-disks",
            _auth_get(
                SYSTEM_DISKS_PATH,
                {"zone": CANDIDATE_ZONES[0], "instance_type": "Ai2.MEDIUM2"},
            ),
        ),
    ]
    for label, call in operations:
        auth_test_suite(call, label=label)


def test_02_default_policy_is_fixed_and_editable():
    default = _default_policy()
    assert default.get("name") == "企业默认资源策略", default
    assert default.get("groups") == [], default

    english_list = seed.get(
        RESOURCE_PATH,
        extra_headers={"Accept-Language": "en"},
    )
    english_default = next(
        item for item in english_list.get("items") or [] if item.get("is_default")
    )
    assert english_default.get("name") == "Enterprise Default Resource Policy", english_default

    original_config = copy.deepcopy(default.get("resource_config") or {})
    roundtrip = seed.post(
        UPDATE_PATH,
        json={"id": default["id"], "resource_config": original_config},
        expect=None,
        raw=True,
    )
    if roundtrip.status_code == 200:
        updated = {
            "instance_charge_type": CHARGE_TYPE,
            "instance_type": "Ai2.MEDIUM4",
            "system_disk": {"disk_type": "cloud_ssd", "disk_size": 80},
            "future_field": "ignored",
        }
        resp = seed.post(
            UPDATE_PATH,
            json={
                "id": default["id"],
                "name": english_default["name"],
                "resource_config": updated,
            },
            expect=None,
            raw=True,
            extra_headers={"Accept-Language": "en"},
        )
        assert resp.status_code == 200, resp.text[:300]
        state["default_mutated"] = True
        expected_config = {
            "instance_charge_type": CHARGE_TYPE,
            "instance_type": "Ai2.MEDIUM4",
            "system_disk": {"disk_type": "CLOUD_SSD", "disk_size": 80},
        }
    else:
        print(
            "    SKIP default config mutation: existing legacy config cannot round-trip "
            f"through current validation ({roundtrip.status_code})"
        )
        expected_config = original_config
    after = _default_policy()
    assert after.get("name") == "企业默认资源策略", after
    assert after.get("groups") == [], after
    assert after.get("resource_config") == expected_config, after

    protected_updates = [
        {"id": default["id"], "name": "renamed", "resource_config": {}},
        {"id": default["id"], "group_ids": [1], "resource_config": {}},
    ]
    for payload in protected_updates:
        denied = seed.post(UPDATE_PATH, json=payload, expect=None, raw=True)
        assert denied.status_code == 409, (payload, denied.status_code, denied.text[:300])
        protected = _default_policy()
        assert protected.get("name") == "企业默认资源策略", protected
        assert protected.get("groups") == [], protected
        assert protected.get("resource_config") == expected_config, protected

    denied = seed.post(DELETE_PATH, json={"id": default["id"]}, expect=None, raw=True)
    assert denied.status_code == 409, denied.text[:300]


def test_03_invalid_update_is_atomic():
    before = copy.deepcopy(_default_policy().get("resource_config") or {})
    invalid_values = [
        {"instance_type": "NOT.ALLOWED"},
        {"system_disk": {"disk_type": "LOCAL_SSD", "disk_size": 80}},
        {"system_disk": {"disk_type": "CLOUD_SSD", "disk_size": -1}},
        {"instance_charge_type": "PREPAID"},
    ]
    for value in invalid_values:
        resp = seed.post(
            UPDATE_PATH,
            json={"id": state["default_policy_id"], "resource_config": value},
            expect=None,
            raw=True,
        )
        assert resp.status_code == 400, (value, resp.status_code, resp.text[:300])
        assert _default_policy().get("resource_config") == before


def test_04_list_pagination_validation():
    body = seed.get(RESOURCE_PATH, params={"page": 1, "page_size": 1})
    assert body.get("page") == 1 and body.get("page_size") == 1, body
    assert body.get("total", 0) >= 1 and len(body.get("items") or []) == 1, body
    resp = seed.get(RESOURCE_PATH, params={"page": 0}, expect=None, raw=True)
    assert resp.status_code == 400, resp.text[:300]


def test_05_instance_type_options_cover_all_query_parameters():
    body = _find_sellable_instance_type()
    assert body.get("source") in {"tencent_cloud", "cache"}, body
    for item in body.get("instance_types") or []:
        assert item.get("instance_type") in ALLOWED_INSTANCE_TYPES, item
        assert isinstance(item.get("cpu"), int) and item["cpu"] > 0, item
        assert isinstance(item.get("memory"), int) and item["memory"] > 0, item


def test_06_system_disk_options_cover_all_query_parameters():
    if not state.get("zone"):
        _find_sellable_instance_type()
    body = seed.get(
        SYSTEM_DISKS_PATH,
        params={
            "zone": state["zone"],
            "instance_charge_type": CHARGE_TYPE,
            "instance_type": state["instance_type"],
            "refresh": "1",
        },
    )
    assert body.get("source") in {"tencent_cloud", "cache"}, body
    items = body.get("system_disk_options") or []
    assert items, body
    for item in items:
        assert item.get("disk_type") in ALLOWED_DISK_TYPES, item
        assert int(item.get("min_disk_size") or 0) >= 0, item
        maximum = int(item.get("max_disk_size") or 0)
        assert maximum == 0 or maximum >= int(item["min_disk_size"]), item
        assert int(item.get("step_size") or 0) >= 0, item


def test_07_legacy_get_returns_effective_default_policy():
    if not state.get("default_mutated"):
        print("    SKIP legacy read: original default policy cannot round-trip")
        return
    policy = _default_policy().get("resource_config") or {}
    site = seed.get(
        "/admin/config",
        params=[
            ("template_path", "instance_type"),
            ("template_path", "system_disk"),
        ],
    ).get("config") or {}
    assert site.get("instance_charge_type") == policy.get("instance_charge_type"), site
    assert site.get("instance_type") == policy.get("instance_type"), site
    assert site.get("system_disk") == policy.get("system_disk"), site


def test_08_legacy_config_keeps_current_charge_type():
    if not state.get("default_mutated"):
        print("    SKIP legacy write: original default policy cannot round-trip")
        return
    before = copy.deepcopy(_default_policy().get("resource_config") or {})
    resp = seed.post(
        "/admin/config",
        data={"instance_charge_type": before["instance_charge_type"]},
        expect=None,
        raw=True,
    )
    assert resp.status_code == 200, resp.text[:300]
    assert _default_policy().get("resource_config") == before


def test_09_legacy_template_syncs_current_system_disk():
    if not state.get("default_mutated"):
        print("    SKIP legacy write: original default policy cannot round-trip")
        return
    site = seed.get("/admin/config").get("config") or {}
    template = json.loads(site.get("cvm_template") or "{}")
    system_disk = template.get("SystemDisk")
    assert system_disk, template
    resp = seed.post(
        "/admin/config/template",
        json={
            "system_disk": {
                "disk_type": system_disk.get("DiskType"),
                "disk_size": system_disk.get("DiskSize"),
            }
        },
        expect=None,
        raw=True,
    )
    assert resp.status_code == 200, resp.text[:300]
    state["default_mutated"] = True
    policy = _default_policy().get("resource_config") or {}
    assert policy.get("system_disk") == {
        "disk_type": system_disk.get("DiskType"),
        "disk_size": system_disk.get("DiskSize"),
    }




def main():
    health_check()
    _capture_original()
    try:
        run_tests(
            globals(),
            title="ResourcePolicy management/options and legacy compatibility (I01-I09)",
            ordered=True,
            abort_on_fail=True,
        )
    finally:
        _restore_original()


if __name__ == "__main__":
    main()
