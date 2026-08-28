#!/usr/bin/env python3
"""Resource policy precedence through the real instance creation path."""

import copy
import json
import os
import sys
import time
import uuid

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers.api import (  # noqa: E402
    ADMIN_TOKEN,
    IDENTIFIER,
    cleanup_users_by_prefix,
    health_check,
    run_tests,
    seed,
    user_client,
)
from helpers.user_groups import (  # noqa: E402
    extract_group_id,
    find_groups_by_prefix,
    list_all_groups,
    pick_group,
)
from helpers.user_mgmt import (  # noqa: E402
    admin_enable_token,
    admin_get_user_token,
)
from _instance_helpers import wait_for_destroyed, wait_for_running  # noqa: E402

RESOURCE_PATH = "/admin/resource-policies"
RESOURCE_CREATE_PATH = "/admin/resource-policies/create"
RESOURCE_UPDATE_PATH = "/admin/resource-policies/update"
RESOURCE_DELETE_PATH = "/admin/resource-policies/delete"
INSTANCE_TYPES_PATH = "/admin/resource-policies/options/instance-types"
SYSTEM_DISKS_PATH = "/admin/resource-policies/options/system-disks"
CHARGE_TYPE = "POSTPAID_BY_HOUR"
ALLOWED_INSTANCE_TYPES = {"Ai2.MEDIUM2", "Ai2.MEDIUM4", "Ai2.LARGE8", "Ai2.2XLARGE16"}
ALLOWED_DISK_TYPES = {"CLOUD_SSD", "CLOUD_PREMIUM", "CLOUD_BSSD", "CLOUD_HSSD"}
CANDIDATE_ZONES = ("ap-guangzhou-6", "ap-guangzhou-7")

RUN_ID = f"{IDENTIFIER or 'local'}-{uuid.uuid4().hex[:8]}"
PREFIX = f"it-rc-create-{RUN_ID}"
GROUP_NAME = f"{PREFIX}-group"
PASSWORD = "RcPolicy-It-12345!"

state = {
    "original_resource_config": None,
    "default_policy_id": None,
    "default_mutated": False,
    "group_id": None,
    "group_policy_id": None,
    "users": {},
    "active_instances": [],
    "zone": None,
    "instance_type": None,
    "disk_type": None,
    "default_disk_size": None,
    "group_disk_size": None,
    "user_disk_size": None,
}


def _capture_default_policy():
    body = seed.get(RESOURCE_PATH, params={"page": 1, "page_size": 100})
    policy = next((item for item in body.get("items") or [] if item.get("is_default")), None)
    assert policy, f"default resource policy missing: {body}"
    state["default_policy_id"] = policy["id"]
    state["original_resource_config"] = copy.deepcopy(policy.get("resource_config") or {})


def _update_default_policy(resource_config):
    resp = seed.post(
        RESOURCE_UPDATE_PATH,
        json={"id": state["default_policy_id"], "resource_config": resource_config},
        expect=None,
        raw=True,
    )
    assert resp.status_code == 200, (
        f"default resource policy update failed: {resp.status_code} {resp.text[:300]}"
    )
    state["default_mutated"] = True


def _create_group():
    resp = seed.post(
        "/admin/user-groups/create",
        json={"name": GROUP_NAME, "description": "instance resource policy integration test"},
        expect=None,
        raw=True,
    )
    assert resp.status_code == 200, f"create group failed: {resp.status_code} {resp.text[:300]}"
    group_id = extract_group_id(resp.json())
    if not group_id:
        group = pick_group(list_all_groups(), name=GROUP_NAME)
        group_id = group and (group.get("id") or group.get("ID"))
    assert group_id, f"created group not found: {resp.text[:300]}"
    state["group_id"] = group_id


def _create_group_policy(resource_config):
    resp = seed.post(
        RESOURCE_CREATE_PATH,
        json={
            "name": f"{PREFIX}-group-policy",
            "resource_config": resource_config,
            "group_ids": [state["group_id"]],
        },
        expect=None,
        raw=True,
    )
    assert resp.status_code == 200, (
        f"create group resource policy failed: {resp.status_code} {resp.text[:300]}"
    )
    state["group_policy_id"] = resp.json().get("id")
    assert state["group_policy_id"], resp.text[:300]


def _create_user(label, group_ids=None):
    username = f"{PREFIX}-{label}"
    payload = {
        "username": username,
        "password": PASSWORD,
        "role": "user",
        "instance_quota": 5,
    }
    if group_ids:
        payload["group_ids"] = group_ids
    data = seed.post("/admin/create", json=payload)
    user_id = data.get("id")
    assert data.get("ok") and user_id, f"create user failed: {data}"
    admin_enable_token(ADMIN_TOKEN, user_id)
    token = admin_get_user_token(ADMIN_TOKEN, user_id)
    assert token, f"ordinary user token missing: {username}"
    state["users"][label] = {"id": user_id, "username": username, "token": token}


def _available_resource_tuple():
    errors = []
    for zone in CANDIDATE_ZONES:
        types_resp = seed.get(
            INSTANCE_TYPES_PATH,
            params={
                "zone": zone,
                "instance_charge_type": CHARGE_TYPE,
                "refresh": "1",
            },
            expect=None,
            raw=True,
            timeout=60,
        )
        if types_resp.status_code != 200:
            errors.append(f"{zone}: instance-types {types_resp.status_code}")
            continue
        for type_item in types_resp.json().get("instance_types") or []:
            instance_type = type_item.get("instance_type")
            if instance_type not in ALLOWED_INSTANCE_TYPES:
                continue
            disks_resp = seed.get(
                SYSTEM_DISKS_PATH,
                params={
                    "zone": zone,
                    "instance_charge_type": CHARGE_TYPE,
                    "instance_type": instance_type,
                    "refresh": "1",
                },
                expect=None,
                raw=True,
                timeout=60,
            )
            if disks_resp.status_code != 200:
                errors.append(f"{zone}/{instance_type}: system-disks {disks_resp.status_code}")
                continue
            for disk in disks_resp.json().get("system_disk_options") or []:
                disk_type = disk.get("disk_type")
                if disk_type not in ALLOWED_DISK_TYPES:
                    continue
                minimum = max(50, int(disk.get("min_disk_size") or 50))
                maximum = int(disk.get("max_disk_size") or 0)
                step = max(1, int(disk.get("step_size") or 1))
                sizes = (minimum, minimum + step, minimum + 2 * step)
                if maximum == 0 or sizes[-1] <= maximum:
                    return zone, instance_type, disk_type, sizes
        errors.append(f"{zone}: no compatible instance/disk tuple")
    raise AssertionError("no live resource tuple available: " + "; ".join(errors))


def _resource_config(disk_size):
    return {
        "instance_charge_type": CHARGE_TYPE,
        "instance_type": state["instance_type"],
        "system_disk": {
            "disk_type": state["disk_type"],
            "disk_size": disk_size,
        },
    }


def _configure_policies():
    zone, instance_type, disk_type, sizes = _available_resource_tuple()
    state["zone"] = zone
    state["instance_type"] = instance_type
    state["disk_type"] = disk_type
    state["default_disk_size"], state["group_disk_size"], state["user_disk_size"] = sizes
    _update_default_policy(_resource_config(state["default_disk_size"]))
    _create_group_policy(_resource_config(state["group_disk_size"]))


def _create_request(token, name, group_id=None, resource_config=None):
    data = {"name": name, "agent_type": "openclaw"}
    if group_id:
        data["group_id"] = str(group_id)
    if resource_config is not None:
        data["resource_config"] = json.dumps(resource_config, separators=(",", ":"))
    return user_client(token).post(
        "/openclaw/create",
        data=data,
        extra_headers={"X-Request-ID": f"{PREFIX}-{uuid.uuid4().hex[:8]}"},
        expect=None,
        raw=True,
        timeout=120,
    )


def _resolve_db_id(token, cvm_id, name, timeout=60):
    deadline = time.time() + timeout
    while time.time() < deadline:
        body = user_client(token).get(
            "/openclaw/list",
            params={"instance_id": cvm_id, "page": 1, "page_size": 100},
        )
        for item in body.get("instances") or []:
            item_cvm_id = item.get("instance_id") or item.get("InstanceId")
            item_name = item.get("name") or item.get("Name")
            if item_cvm_id == cvm_id or item_name == name:
                db_id = item.get("id") or item.get("ID")
                if db_id:
                    return db_id
        time.sleep(2)
    raise AssertionError(f"created instance not found in Hatchery list: {cvm_id} {name}")


def _wait_admin_resources(db_id, expected_disk_size, timeout=300):
    deadline = time.time() + timeout
    last = None
    while time.time() < deadline:
        body = seed.get(
            "/admin/instances",
            params={"ids": str(db_id), "page": 1, "page_size": 100},
            timeout=120,
        )
        items = body.get("instances") or []
        if len(items) == 1:
            last = items[0]
            if (
                last.get("cvm_instance_type") == state["instance_type"]
                and last.get("system_disk_type") == state["disk_type"]
                and int(last.get("system_disk_size") or 0) == expected_disk_size
            ):
                return last
        time.sleep(3)
    raise AssertionError(f"admin resource cache did not reach expected values: {last}")


def _delete_instance(record):
    token = record["token"]
    db_id = record["db_id"]
    client = user_client(token)
    try:
        wait_for_running(db_id, timeout=420, client=client)
    except Exception as exc:
        print(f"    instance did not become user-deletable; using admin cleanup: {exc}")
    resp = client.post(
        "/openclaw/delete",
        data={"id": str(db_id)},
        expect=None,
        raw=True,
        timeout=60,
    )
    if resp.status_code == 409:
        resp = seed.post(
            "/admin/instances/delete",
            json={"ids": [db_id]},
            expect=None,
            raw=True,
            timeout=60,
        )
    assert resp.status_code == 200, (
        f"instance delete failed: status={resp.status_code} body={resp.text[:500]}"
    )
    wait_for_destroyed(db_id, timeout=600, client=client)
    if record in state["active_instances"]:
        state["active_instances"].remove(record)


def _run_create_case(label, user_label, expected_disk_size, group_id=None, resource_config=None):
    user = state["users"][user_label]
    name = f"{PREFIX}-{label}-{int(time.time())}"
    resp = _create_request(user["token"], name, group_id, resource_config)
    assert resp.status_code == 200, (
        f"{label} create failed: status={resp.status_code} body={resp.text[:500]}"
    )
    body = resp.json()
    assert body.get("ok") is True and body.get("instance_id"), body
    db_id = _resolve_db_id(user["token"], body["instance_id"], name)
    record = {
        "db_id": db_id,
        "cvm_id": body["instance_id"],
        "name": name,
        "token": user["token"],
    }
    state["active_instances"].append(record)
    wait_for_running(db_id, timeout=600, client=user_client(user["token"]))
    _wait_admin_resources(db_id, expected_disk_size)
    _delete_instance(record)


def test_01_prepare_resource_policies_and_users():
    """Prepare default and group policies plus grouped and ungrouped users."""
    assert not find_groups_by_prefix(PREFIX), f"stale groups exist for {PREFIX}"
    cleanup_users_by_prefix(PREFIX, verbose=False)
    _capture_default_policy()
    _create_group()
    _create_user("default-user")
    _create_user("group-user", [state["group_id"]])
    _configure_policies()
    assert len({
        state["default_disk_size"],
        state["group_disk_size"],
        state["user_disk_size"],
    }) == 3, state


def test_02_default_policy_applies_to_ungrouped_instance():
    _run_create_case("default", "default-user", state["default_disk_size"])


def test_03_group_policy_overrides_default_policy():
    _run_create_case(
        "group",
        "group-user",
        state["group_disk_size"],
        group_id=state["group_id"],
    )


def test_04_user_resource_config_overrides_group_policy():
    _run_create_case(
        "user",
        "group-user",
        state["user_disk_size"],
        group_id=state["group_id"],
        resource_config=_resource_config(state["user_disk_size"]),
    )


def cleanup():
    errors = []
    for record in list(reversed(state["active_instances"])):
        try:
            _delete_instance(record)
        except Exception as exc:
            errors.append(f"instance cleanup {record}: {exc}")

    try:
        cleanup_users_by_prefix(PREFIX, verbose=False)
    except Exception as exc:
        errors.append(f"user cleanup: {exc}")

    if state.get("group_policy_id"):
        try:
            resp = seed.post(
                RESOURCE_DELETE_PATH,
                json={"id": state["group_policy_id"]},
                expect=None,
                raw=True,
            )
            if resp.status_code not in (200, 404):
                raise AssertionError(f"status={resp.status_code} body={resp.text[:300]}")
            state["group_policy_id"] = None
        except Exception as exc:
            errors.append(f"group resource policy cleanup: {exc}")

    if state.get("group_id"):
        try:
            resp = seed.post(
                "/admin/user-groups/delete",
                json={"id": state["group_id"]},
                expect=None,
                raw=True,
            )
            if resp.status_code not in (200, 404):
                raise AssertionError(f"status={resp.status_code} body={resp.text[:300]}")
        except Exception as exc:
            errors.append(f"group cleanup: {exc}")

    if state.get("default_mutated"):
        try:
            resp = seed.post(
                RESOURCE_UPDATE_PATH,
                json={
                    "id": state["default_policy_id"],
                    "resource_config": state["original_resource_config"],
                },
                expect=None,
                raw=True,
            )
            assert resp.status_code == 200, resp.text[:300]
            state["default_mutated"] = False
        except Exception as exc:
            errors.append(f"default resource policy restore: {exc}")

    leftovers = find_groups_by_prefix(PREFIX)
    if leftovers:
        errors.append(f"group leftovers: {leftovers[:3]}")
    if errors:
        raise AssertionError("; ".join(errors))


def main():
    health_check()
    try:
        run_tests(
            globals(),
            title="Resource Config Policy - instance creation precedence",
            ordered=True,
            abort_on_fail=True,
        )
    finally:
        cleanup()


if __name__ == "__main__":
    main()
