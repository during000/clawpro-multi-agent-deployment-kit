#!/usr/bin/env python3
"""
End-to-end integration coverage for admin CVM specification and system-disk adjustment.

The script owns one disposable CVM and one local-agent row. It validates the
public API contract, performs irreversible upgrades in ascending order, checks
operation locking/idempotency, observes Tencent Cloud state through the existing
admin cloud proxy, verifies audit records, and destroys the CVM in finally.
"""
from __future__ import annotations

import os
import sys
import time
import traceback
from typing import Any


sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers import config  # noqa: E402
from helpers.api import (  # noqa: E402
    ApiClient,
    assert_status,
    auth_test_suite,
    health_check,
    seed,
    user_client,
)
from helpers.instance import create_instance  # noqa: E402
from helpers.local_agent import (  # noqa: E402
    enable_local_agent_feature,
    setup_local_instance,
)
from _instance_helpers import (  # noqa: E402
    wait_for_destroyed,
    wait_for_running,
)

VALIDATE_PATH = "/admin/instances/adjust-config/validate"
SUBMIT_PATH = "/admin/instances/adjust-config"
MISSING_DB_ID = 999_999_991
MISSING_INSTANCE_ID = "ins-adjustment-integration-missing"
TIERS = ["Ai2.MEDIUM2", "Ai2.MEDIUM4", "Ai2.LARGE8", "Ai2.2XLARGE16"]
POLL_INTERVAL = 2
ADJUSTMENT_TIMEOUT = 1_100
SCRIPT_STARTED_AT = int(time.time())
SKIP_SPEC_STAGES = os.environ.get("ADJUSTMENT_IT_SKIP_SPEC", "") == "1"

BOOTSTRAP_ADMIN_TOKEN = os.environ.get("BOOTSTRAP_ADMIN_TOKEN", "").strip()
cloud_admin = ApiClient(BOOTSTRAP_ADMIN_TOKEN)

USER_TOKEN = os.environ.get("TOKEN", "").strip()
USER = user_client(USER_TOKEN) if USER_TOKEN else None
CTX: dict[str, Any] = {
    "db_id": None,
    "instance_id": None,
    "local_db_id": None,
    "spec_targets": [],
    "results": [],
}


def run_case(title: str, fn):
    started = time.time()
    print(f"\n>>> {title}", flush=True)
    fn()
    duration = time.time() - started
    CTX["results"].append({"title": title, "duration_seconds": round(duration, 3)})
    print(f"    PASS ({duration:.1f}s)", flush=True)


def response_json(resp) -> dict:
    try:
        data = resp.json()
    except Exception as exc:
        raise AssertionError(f"expected JSON, status={resp.status_code}, body={resp.text[:500]}") from exc
    assert isinstance(data, dict), f"expected JSON object, got {type(data).__name__}: {data}"
    return data


def post_adjust(path: str, payload: dict, *, expect=200) -> dict:
    resp = seed.post(path, json=payload, expect=None, raw=True, timeout=120)
    assert_status(resp, expect, label=path)
    return response_json(resp)


def one_result(data: dict) -> dict:
    results = data.get("results") or []
    assert len(results) == 1, f"expected one result: {data}"
    return results[0]


def admin_item() -> dict:
    db_id = CTX["db_id"]
    data = seed.get(
        "/admin/instances",
        params={"ids": str(db_id), "page": 1, "page_size": 100},
        timeout=120,
    )
    items = data.get("instances") or []
    assert len(items) == 1, f"admin list did not return db_id={db_id}: {data}"
    return items[0]


def wait_admin_main_status(target: str, *, timeout=420) -> dict:
    deadline = time.time() + timeout
    last = None
    while time.time() < deadline:
        last = admin_item()
        if last.get("status") == target and not last.get("transient", False):
            return last
        time.sleep(POLL_INTERVAL)
    raise TimeoutError(f"admin status cache did not reach {target}: {last}")


def admin_status() -> dict:
    return seed.get(
        "/admin/instances/status",
        params={"id": CTX["db_id"]},
        timeout=120,
    )


def cloud_describe_instance() -> dict:
    response = cloud_admin.post(
        "/admin/cloud/query/cvm",
        json={"InstanceIds": [CTX["instance_id"]]},
        extra_headers={"X-TC-Action": "DescribeInstances"},
        expect=None,
        raw=True,
        timeout=60,
    )
    data = response_json(response)
    error = (data.get("Response") or {}).get("Error")
    assert response.status_code == 200 and not error, (
        f"proxied CVM DescribeInstances failed: status={response.status_code}, error={error}"
    )
    instances = (data.get("Response") or {}).get("InstanceSet") or []
    assert len(instances) == 1, f"DescribeInstances did not return target: {data}"
    return instances[0]


def wait_cloud_stable(*, timeout=360) -> dict:
    deadline = time.time() + timeout
    last = None
    stable_observations = 0
    while time.time() < deadline:
        last = cloud_describe_instance()
        if (
            last.get("InstanceState") in {"RUNNING", "STOPPED"}
            and last.get("LatestOperationState") != "OPERATING"
        ):
            stable_observations += 1
            if stable_observations >= 5:
                return last
        else:
            stable_observations = 0
        time.sleep(POLL_INTERVAL)
    raise TimeoutError(f"CVM did not reach a stable state: {last}")


def resolve_created_instance(cvm_id: str, *, timeout=60) -> dict:
    deadline = time.time() + timeout
    while time.time() < deadline:
        data = USER.get(
            "/openclaw/list",
            params={"instance_id": cvm_id, "page": 1, "page_size": 100},
            timeout=60,
        )
        for item in data.get("instances") or []:
            if (item.get("instance_id") or item.get("InstanceId")) == cvm_id:
                return item
        time.sleep(2)
    raise TimeoutError(f"created instance {cvm_id} was not visible in /openclaw/list")


def wait_adjustment(
    label: str,
    *,
    expected_state: str,
    expected_instance_type: str | None = None,
    expected_disk_size: int | None = None,
) -> dict:
    deadline = time.time() + ADJUSTMENT_TIMEOUT
    trace: list[dict] = []
    last_signature = None

    while time.time() < deadline:
        item = admin_item()
        detail = admin_status()
        point = {
            "elapsed_seconds": round(ADJUSTMENT_TIMEOUT - (deadline - time.time()), 3),
            "adjustment_status": item.get("adjustment_status", ""),
            "adjustment_type": item.get("adjustment_type", ""),
            "adjustment_error_code": item.get("adjustment_error_code", ""),
            "current_operation": item.get("current_operation", ""),
            "main_status": item.get("status", ""),
            "instance_type": item.get("cvm_instance_type", ""),
            "disk_size": item.get("system_disk_size", 0),
            "detail_state": detail.get("state", ""),
        }
        trace.append(point)
        signature = tuple(point[key] for key in (
            "adjustment_status",
            "adjustment_error_code",
            "current_operation",
            "main_status",
            "instance_type",
            "disk_size",
        ))
        if signature != last_signature:
            print(
                "    poll "
                f"adjustment={point['adjustment_status'] or '<empty>'} "
                f"operation={point['current_operation'] or '<empty>'} "
                f"status={point['main_status']} type={point['instance_type']} "
                f"disk={point['disk_size']}",
                flush=True,
            )
            last_signature = signature

        if point["adjustment_status"] == "failed":
            raise AssertionError(f"{label} failed: {item}")

        target_reached = True
        if expected_instance_type is not None:
            target_reached = item.get("cvm_instance_type") == expected_instance_type
        if expected_disk_size is not None:
            target_reached = int(item.get("system_disk_size") or 0) >= expected_disk_size

        if (
            point["adjustment_status"] == ""
            and point["current_operation"] == ""
            and point["main_status"] == expected_state
            and target_reached
        ):
            return {"outcome": "success", "item": item, "detail": detail, "trace": trace}

        time.sleep(POLL_INTERVAL)

    raise TimeoutError(f"{label} did not finish within {ADJUSTMENT_TIMEOUT}s")


def validate_after_cloud_settles(payload: dict, *, timeout=600) -> dict:
    deadline = time.time() + timeout
    last = None
    while time.time() < deadline:
        last = one_result(post_adjust(VALIDATE_PATH, payload))
        if last.get("reason_code") != "cvm_operation_in_progress":
            return last
        time.sleep(5)
    raise TimeoutError(f"validation stayed blocked by a CVM operation: {last}")


def next_disk_target(*, resize_mode: str) -> tuple[int, dict]:
    item = admin_item()
    current = int(item.get("system_disk_size") or 0)
    assert current > 0, f"missing current system disk size: {item}"
    probe_payload = {
        "ids": [CTX["db_id"]],
        "adjustment_type": "system_disk",
        "target_system_disk_size": current + 1,
        "resize_mode": resize_mode,
    }
    probe = validate_after_cloud_settles(probe_payload)
    if probe.get("adjustable"):
        target = current + 1
    else:
        assert probe.get("reason_code") == "invalid_disk_size", (
            f"disk quota probe failed before size validation: {probe}"
        )
        target = int(probe.get("min_disk_size") or 0)
    assert target > current, f"no larger disk target available: current={current}, probe={probe}"
    assert target <= int(probe.get("max_disk_size") or target), f"target exceeds max: {probe}"

    payload = dict(probe_payload)
    payload["target_system_disk_size"] = target
    validated = validate_after_cloud_settles(payload)
    assert validated.get("adjustable"), f"normalized disk target rejected: {validated}"
    assert int(validated.get("step_size") or 0) > 0, f"missing disk step size: {validated}"
    return target, validated


def submit_payload(payload: dict) -> dict:
    body = post_adjust(SUBMIT_PATH, payload)
    result = one_result(body)
    return {"body": body, "result": result}


def cleanup_local_row():
    local_db_id = CTX.get("local_db_id")
    if not local_db_id or USER is None:
        return
    try:
        USER.post(
            "/openclaw/delete",
            data={"id": local_db_id},
            expect=None,
            raw=True,
            timeout=60,
        )
        print(f"[cleanup] local row delete requested: db_id={local_db_id}")
    except Exception as exc:
        print(f"[cleanup] local row delete failed: {exc}")
    CTX["local_db_id"] = None


def cleanup_cvm():
    db_id = CTX.get("db_id")
    if not db_id or USER is None:
        return
    deadline = time.time() + 900
    last_error = ""
    while time.time() < deadline:
        try:
            resp = USER.post(
                "/openclaw/delete",
                data={"id": db_id},
                expect=None,
                raw=True,
                timeout=90,
            )
            if resp.status_code in {200, 404}:
                if resp.status_code == 200:
                    try:
                        wait_for_destroyed(db_id, timeout=480, client=USER)
                    except Exception:
                        pass
                print(f"[cleanup] CVM delete completed/requested: db_id={db_id}")
                CTX["db_id"] = None
                return
            last_error = f"status={resp.status_code} body={resp.text[:300]}"
        except Exception as exc:
            last_error = str(exc)
        print(f"[cleanup] CVM delete retry: {last_error}", flush=True)
        time.sleep(10)
    raise RuntimeError(f"dedicated CVM cleanup timed out: {last_error}")


def check_request_validation():
    payload = {
        "ids": [MISSING_DB_ID],
        "adjustment_type": "instance_type",
        "target_instance_type": "Ai2.MEDIUM4",
        "target_system_disk_size": 100,
        "resize_mode": "ignored-for-instance-type",
    }
    for path in (VALIDATE_PATH, SUBMIT_PATH):
        auth_test_suite(
            lambda headers, target=path: ApiClient("", timeout=30).post(
                target,
                json=payload,
                expect=None,
                raw=True,
                extra_headers=headers,
            ),
            label=path,
        )
        method = seed.get(path, expect=None, raw=True)
        assert_status(method, 405, label=f"{path} method")

    unknown = seed.post(
        VALIDATE_PATH,
        json={**payload, "unknown": True},
        expect=None,
        raw=True,
    )
    assert_status(unknown, 400, label="unknown request field")

    full_calls = [
        (
            VALIDATE_PATH,
            payload,
        ),
        (
            VALIDATE_PATH,
            {
                "instance_ids": [MISSING_INSTANCE_ID],
                "adjustment_type": "system_disk",
                "target_instance_type": "ignored",
                "target_system_disk_size": 100,
                "resize_mode": "online",
            },
        ),
        (
            SUBMIT_PATH,
            payload,
        ),
        (
            SUBMIT_PATH,
            {
                "instance_ids": [MISSING_INSTANCE_ID],
                "adjustment_type": "system_disk",
                "target_instance_type": "ignored",
                "target_system_disk_size": 100,
                "resize_mode": "offline",
            },
        ),
    ]
    for path, body in full_calls:
        result = one_result(post_adjust(path, body))
        assert result.get("reason_code") == "instance_not_found", result


def prepare_and_validate_instances():
    assert USER is not None, "TOKEN is required for the disposable instance"
    name = f"{config.INSTANCE_NAME_PREFIX}adjust-config-{int(time.time())}"
    create_data = create_instance(USER_TOKEN, name, agent_type="openclaw")
    assert create_data.get("ok"), f"instance create failed: {create_data}"
    CTX["instance_id"] = create_data.get("instance_id")
    assert CTX["instance_id"], create_data
    created = resolve_created_instance(CTX["instance_id"])
    CTX["db_id"] = created.get("id") or created.get("ID")
    assert CTX["db_id"], created
    wait_for_running(CTX["db_id"], timeout=600, client=USER)
    wait_cloud_stable(timeout=600)
    wait_admin_main_status("running", timeout=600)

    enable_local_agent_feature(config.SEED_ADMIN_TOKEN)
    local = setup_local_instance(USER_TOKEN, "adjust-config")
    CTX["local_db_id"] = local.db_id

    item = admin_item()
    current_type = item.get("cvm_instance_type")
    assert current_type in TIERS, f"unsupported initial AI2 type: {item}"
    rank = TIERS.index(current_type)
    assert rank + 2 < len(TIERS), (
        f"integration environment needs two higher AI2 tiers, current={current_type}"
    )
    CTX["spec_targets"] = [TIERS[rank + 1], TIERS[rank + 2]]

    mixed = post_adjust(
        VALIDATE_PATH,
        {
            "ids": [CTX["local_db_id"], MISSING_DB_ID, CTX["db_id"]],
            "adjustment_type": "instance_type",
            "target_instance_type": CTX["spec_targets"][0],
        },
    )
    results = mixed.get("results") or []
    assert [result.get("id") for result in results] == [
        CTX["local_db_id"],
        MISSING_DB_ID,
        CTX["db_id"],
    ], mixed
    assert results[0].get("reason_code") == "cloud_instance_required", results[0]
    assert results[1].get("reason_code") == "instance_not_found", results[1]
    assert results[2].get("adjustable") is True, results[2]
    assert mixed.get("adjustable_count") == 1 and mixed.get("non_adjustable_count") == 2, mixed

    invalid = one_result(post_adjust(
        VALIDATE_PATH,
        {
            "instance_ids": [CTX["instance_id"]],
            "adjustment_type": "instance_type",
            "target_instance_type": current_type,
        },
    ))
    assert invalid.get("reason_code") == "instance_type_unchanged", invalid

    all_items = seed.get(
        "/admin/instances",
        params={"ids": f"{CTX['db_id']},{CTX['local_db_id']}", "page_size": 100},
    ).get("instances") or []
    local_item = next(item for item in all_items if (item.get("ID") or item.get("id")) == CTX["local_db_id"])
    for field in (
        "cvm_instance_type",
        "cpu",
        "memory_gb",
        "system_disk_type",
        "system_disk_size",
        "adjustment_status",
        "public_ip",
        "internet_charge_type",
        "internet_max_bandwidth_out",
    ):
        assert field not in local_item, f"local row leaked {field}: {local_item}"



def check_operation_locks():
    for path in (
        "/admin/instances/stop",
        "/admin/instances/delete",
        "/admin/instances/restart-gateway",
    ):
        resp = seed.post(
            path,
            data={"id": CTX["db_id"]},
            expect=None,
            raw=True,
            timeout=60,
        )
        assert_status(resp, 409, label=f"adjustment lock {path}")
    item = admin_item()
    assert item.get("adjustment_status") == "processing", item
    assert item.get("actions") == [], item
    detail = admin_status()
    assert detail.get("adjustment_status") == "processing", detail


def check_adjustment_idempotency(payload: dict, different_target: str):
    before = admin_item()
    duplicate = submit_payload(payload)
    assert duplicate["body"].get("already_processing_count") == 1, duplicate
    assert duplicate["result"].get("status") == "already_processing", duplicate
    after = admin_item()
    assert before.get("adjustment_updated_at") == after.get("adjustment_updated_at"), (
        "idempotent submit rewrote persisted adjustment state"
    )

    conflicting = dict(payload)
    conflicting["target_instance_type"] = different_target
    conflict = submit_payload(conflicting)
    assert conflict["body"].get("rejected_count") == 1, conflict
    assert conflict["result"].get("reason_code") == "operation_in_progress", conflict


def adjust_instance_type():
    target = CTX["spec_targets"][0]
    payload = {
        "ids": [CTX["db_id"]],
        "adjustment_type": "instance_type",
        "target_instance_type": target,
    }
    validated = one_result(post_adjust(VALIDATE_PATH, payload))
    assert validated.get("adjustable") is True, validated
    submitted = submit_payload(payload)
    assert submitted["body"].get("accepted_count") == 1, submitted
    assert submitted["result"].get("status") == "accepted", submitted

    run_case("operation locks and read visibility during processing", check_operation_locks)
    run_case(
        "same-target idempotency and different-target conflict",
        lambda: check_adjustment_idempotency(payload, CTX["spec_targets"][1]),
    )
    wait_adjustment(
        "running specification upgrade",
        expected_state="running",
        expected_instance_type=target,
    )



def ensure_running():
    item = admin_item()
    if item.get("status") != "running":
        USER.post("/openclaw/start", data={"id": CTX["db_id"]}, timeout=90)
        wait_for_running(CTX["db_id"], timeout=420, client=USER)
    wait_cloud_stable()
    wait_admin_main_status("running")


def expand_disk_online():
    ensure_running()
    target, validated = next_disk_target(resize_mode="online")
    assert validated.get("adjustable") is True, validated
    payload = {
        "ids": [CTX["db_id"]],
        "adjustment_type": "system_disk",
        "target_system_disk_size": target,
        "resize_mode": "online",
    }
    submitted = submit_payload(payload)
    assert submitted["body"].get("accepted_count") == 1, submitted
    wait_adjustment(
        "running online disk expansion",
        expected_state="running",
        expected_disk_size=target,
    )




def expand_disk_offline():
    ensure_running()
    target, validated = next_disk_target(resize_mode="offline")
    payload = {
        "ids": [CTX["db_id"]],
        "adjustment_type": "system_disk",
        "target_system_disk_size": target,
        "resize_mode": "offline",
    }
    accepted = submit_payload(payload)
    assert accepted["body"].get("accepted_count") == 1, accepted
    processing = admin_item()
    assert processing.get("adjustment_status") == "processing", processing
    assert not processing.get("adjustment_error_code"), processing
    completed = wait_adjustment(
        "running offline disk expansion",
        expected_state="running",
        expected_disk_size=target,
    )
    assert completed.get("outcome") == "success", completed
    final = admin_item()
    assert final.get("adjustment_status", "") == "", final
    assert not final.get("adjustment_error_code"), final




def check_list_resource_fields():
    item = admin_item()
    instance_type = item.get("cvm_instance_type")
    disk_size = int(item.get("system_disk_size") or 0)
    data = seed.get(
        "/admin/instances",
        params={
            "cvm_instance_type": f"Ai2.MEDIUM2,{instance_type}",
            "system_disk_size": f"{disk_size},999999",
            "ids": str(CTX["db_id"]),
            "page": 1,
            "page_size": 10,
        },
    )
    items = data.get("instances") or []
    assert len(items) == 1, data
    assert data.get("total") == 1, data
    assert (data.get("stats") or {}).get("total") == 1, data

    range_cases = (
        ("system_disk_size_lt", disk_size + 1, 1),
        ("system_disk_size_lt", disk_size, 0),
        ("system_disk_size_gt", disk_size - 1, 1),
        ("system_disk_size_gt", disk_size, 0),
    )
    for parameter, value, expected_total in range_cases:
        ranged = seed.get(
            "/admin/instances",
            params={
                parameter: value,
                "ids": str(CTX["db_id"]),
                "page": 1,
                "page_size": 10,
            },
        )
        assert ranged.get("total") == expected_total, (parameter, value, ranged)
        assert (ranged.get("stats") or {}).get("total") == expected_total, (
            parameter,
            value,
            ranged,
        )
        assert len(ranged.get("instances") or []) == expected_total, (
            parameter,
            value,
            ranged,
        )
    filtered = items[0]
    expected_fields = {
        "cvm_instance_type",
        "cpu",
        "memory_gb",
        "system_disk_type",
        "system_disk_size",
        "public_ip",
        "internet_charge_type",
        "internet_max_bandwidth_out",
    }
    assert expected_fields <= set(filtered), filtered
    detail = admin_status()
    assert expected_fields <= set(detail), detail
    assert detail.get("cvm_instance_type") == instance_type, detail
    assert int(detail.get("system_disk_size") or 0) == disk_size, detail
    assert detail.get("state") == "RUNNING", detail


def check_final_state_and_audit():
    final = admin_item()
    assert final.get("adjustment_status", "") == "", final
    assert not final.get("adjustment_error_code"), final
    audits = seed.get(
        "/admin/audit",
        params={
            "action": "instance_adjust_config",
            "start_time": SCRIPT_STARTED_AT - 5,
            "page": 1,
            "page_size": 1000,
        },
    )
    logs = audits.get("logs") or []
    assert int(audits.get("total") or 0) >= 1 and logs, audits
    assert all(log.get("action") == "instance_adjust_config" for log in logs), logs


def main():
    health_check()
    assert USER is not None, "TOKEN must be provided by the integration orchestrator"
    assert BOOTSTRAP_ADMIN_TOKEN, "integration runner did not provide bootstrap admin token"
    print()
    try:
        run_case("route authentication and complete request parameter coverage", check_request_validation)
        run_case("mixed local, missing, valid, and invalid-target validation", prepare_and_validate_instances)
        if not SKIP_SPEC_STAGES:
            run_case("running instance specification upgrade", adjust_instance_type)
        run_case("running online system-disk expansion", expand_disk_online)
        run_case("running offline system-disk expansion and terminal clearing", expand_disk_offline)
        run_case("list filters, pagination totals, and status resource fields", check_list_resource_fields)
        run_case("submit audit and successful terminal state", check_final_state_and_audit)
        print("\nSelected integration scenarios passed")
    except Exception as exc:
        print(f"\nIntegration failure: {exc}")
        traceback.print_exc()
        raise
    finally:
        cleanup_local_row()
        cleanup_cvm()
        print("\nCase timing summary:")
        for result in CTX["results"]:
            print(f"  PASS {result['duration_seconds']}s - {result['title']}")


if __name__ == "__main__":
    main()
