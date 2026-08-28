#!/usr/bin/env python3
"""Independent ResourcePolicy group inheritance integration test (I07)."""

import os
import sys
import uuid

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers.api import health_check, run_tests, seed  # noqa: E402
from helpers.user_groups import (  # noqa: E402
    extract_group_id,
    find_groups_by_prefix,
    list_all_groups,
    pick_group,
)

PREFIX = f"it-resource-policy-{os.environ.get('IDENTIFIER') or 'local'}-{uuid.uuid4().hex[:8]}"
PARENT_NAME = f"{PREFIX}-parent"
CHILD_NAME = f"{PREFIX}-child"
POLICY_NAME = f"{PREFIX}-policy"
POLICY_VALUE = {
    "instance_type": "Ai2.MEDIUM2",
    "system_disk": {"disk_type": "cloud_ssd", "disk_size": 60},
    "future_group_field": "ignored",
}
EXPECTED_VALUE = {
    "instance_type": "Ai2.MEDIUM2",
    "system_disk": {"disk_type": "CLOUD_SSD", "disk_size": 60},
}

state = {"parent_id": None, "child_id": None, "policy_id": None}


def _create_group(name, parent_id=0):
    body = {"name": name, "description": "resource policy integration test"}
    if parent_id:
        body["parent_id"] = parent_id
    resp = seed.post("/admin/user-groups/create", json=body, expect=None, raw=True)
    assert resp.status_code == 200, f"create group failed: {resp.status_code} {resp.text[:300]}"
    group_id = extract_group_id(resp.json())
    if not group_id:
        group = pick_group(list_all_groups(), name=name)
        group_id = group and (group.get("id") or group.get("ID"))
    assert group_id, f"created group not found: {resp.text[:300]}"
    return group_id


def _find_tree_node(nodes, group_id):
    for node in nodes or []:
        if node.get("id") == group_id:
            return node
        found = _find_tree_node(node.get("children"), group_id)
        if found:
            return found
    return None


def _overview_entry(group_id):
    body = seed.get(
        "/admin/user-groups/config-overview",
        params={"group_ids": str(group_id), "keys": "resourcePolicy"},
    )
    results = body.get("results") or []
    assert len(results) == 1, body
    categories = results[0].get("categories") or []
    category = next((item for item in categories if item.get("key") == "resourcePolicy"), None)
    assert category and len(category.get("entries") or []) == 1, body
    return category["entries"][0]


def test_01_create_parent_and_child_groups():
    state["parent_id"] = _create_group(PARENT_NAME)
    state["child_id"] = _create_group(CHILD_NAME, state["parent_id"])


def test_02_create_policy_and_query_inverse_scope():
    resp = seed.post(
        "/admin/resource-policies/create",
        json={
            "name": POLICY_NAME,
            "resource_config": POLICY_VALUE,
            "group_ids": [state["parent_id"]],
        },
        expect=None,
        raw=True,
    )
    assert resp.status_code == 200, resp.text[:300]
    state["policy_id"] = resp.json().get("id")
    assert state["policy_id"], resp.text[:300]

    body = seed.get("/admin/resource-policies", params={"page": 1, "page_size": 100})
    item = next((row for row in body.get("items") or [] if row.get("id") == state["policy_id"]), None)
    assert item, body
    assert item.get("resource_config") == EXPECTED_VALUE, item
    assert [group.get("id") for group in item.get("groups") or []] == [state["parent_id"]], item


def test_03_tree_marks_only_direct_group():
    tree = seed.get(
        "/admin/user-groups/tree",
        params={"with_user_counts": "false", "with_resource_policy": "true"},
    )
    roots = (tree.get("org_tree") or []) + (tree.get("user_groups") or [])
    parent = _find_tree_node(roots, state["parent_id"])
    child = _find_tree_node(roots, state["child_id"])
    assert (parent.get("direct_resource_policy") or {}).get("id") == state["policy_id"], parent
    assert not child.get("direct_resource_policy"), child


def test_04_child_overview_inherits_then_falls_back_default():
    inherited = _overview_entry(state["child_id"])
    assert inherited.get("id") == str(state["policy_id"]), inherited
    assert (inherited.get("source") or {}).get("type") == "inherited", inherited
    assert (inherited.get("meta") or {}).get("resource_config") == EXPECTED_VALUE, inherited
    assert (inherited.get("meta") or {}).get("value") == EXPECTED_VALUE, inherited

    resp = seed.post(
        "/admin/resource-policies/delete",
        json={"id": state["policy_id"]},
        expect=None,
        raw=True,
    )
    assert resp.status_code == 200, resp.text[:300]
    state["policy_id"] = None
    fallback = _overview_entry(state["child_id"])
    assert (fallback.get("source") or {}).get("type") == "site_default", fallback
    assert (fallback.get("meta") or {}).get("is_default") is True, fallback


def cleanup():
    errors = []
    if state.get("policy_id"):
        try:
            resp = seed.post(
                "/admin/resource-policies/delete",
                json={"id": state["policy_id"]},
                expect=None,
                raw=True,
            )
            if resp.status_code not in (200, 404):
                errors.append(f"policy delete: {resp.status_code} {resp.text[:200]}")
        except Exception as exc:
            errors.append(f"policy cleanup: {exc}")

    for key in ("child_id", "parent_id"):
        group_id = state.get(key)
        if not group_id:
            continue
        try:
            resp = seed.post(
                "/admin/user-groups/delete",
                json={"id": group_id},
                expect=None,
                raw=True,
            )
            if resp.status_code not in (200, 400, 404):
                errors.append(f"group delete {group_id}: {resp.status_code} {resp.text[:200]}")
        except Exception as exc:
            errors.append(f"group cleanup {group_id}: {exc}")

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
            title="Independent ResourcePolicy - group lifecycle (I07)",
            ordered=True,
            abort_on_fail=True,
        )
    finally:
        cleanup()


if __name__ == "__main__":
    main()
