#!/usr/bin/env python3
"""
Test script: verify /admin/usage/data API.

Covered scenarios:
  - Health check
  - Response structure: required top-level fields (start_date, end_date, group_by, rows,
                        global_token_quota_day) and per-row token fields
  - Date filter: default date range (today), explicit range, future date returns empty rows
  - group_by parameter:
      - default (no group_by) → group_by contains "date" and "model"
      - group_by=date         → rows contain "date" field
      - group_by=user         → rows contain user fields (user_id, user_name; user_email omitempty)
      - group_by=model        → rows contain model fields (ai_model_id; model_name omitempty)
      - group_by=instance     → rows contain instance fields (instance_id, instance_name)
      - group_by=user,model   → combined dimensions
      - invalid value ignored → falls back to default
  - order_by parameter:
      - default (total_tokens DESC)
      - order_by=request_count with order=desc
      - order_by=invalid_field → 400 Bad Request
  - Optional filters: user_id / ai_model_id / instance_id / group_id (non-existent → empty rows)
  - Auth: no token → 401/403, wrong token → 401/403, non-admin token → 401/403
"""

import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))
from helpers.api import (
    seed, ApiClient,
    health_check, auth_test_suite, assert_fields, assert_sorted,
    run_tests,
    TODAY, YESTERDAY, TOMORROW,
)


def get_data(params=None, headers=None):
    if headers:
        tmp = ApiClient("", timeout=30)
        return tmp.get("/admin/usage/data", params=params, expect=None, raw=True, extra_headers=headers)
    return seed.get("/admin/usage/data", params=params, expect=None, raw=True)


# ─────────────────────────────────────────────
# Response structure
# ─────────────────────────────────────────────

def test_response_structure():
    print(">>> Check response structure ...")
    resp = get_data({"start_date": TODAY, "end_date": TODAY})
    resp.raise_for_status()
    data = resp.json()

    assert_fields(data, ("start_date", "end_date", "group_by", "rows", "global_token_quota_day"),
                  context="top-level")

    assert isinstance(data["group_by"], list), \
        f"group_by should be a list, got {type(data['group_by'])}"
    assert isinstance(data["rows"], list), \
        f"rows should be a list, got {type(data['rows'])}"

    rows = data["rows"]
    if len(rows) > 0:
        assert_fields(rows[0], ("total_tokens", "prompt_tokens", "completion_tokens", "request_count"),
                      context="rows[0]")

    print(f"    OK (group_by={data['group_by']}, rows_count={len(rows)})")


# ─────────────────────────────────────────────
# Date filter
# ─────────────────────────────────────────────

def test_default_date_range():
    print(">>> Check default date range (today) ...")
    resp = get_data()
    resp.raise_for_status()
    data = resp.json()
    assert data.get("start_date") == TODAY, \
        f"Expected start_date={TODAY}, got {data.get('start_date')}"
    assert data.get("end_date") == TODAY, \
        f"Expected end_date={TODAY}, got {data.get('end_date')}"
    print(f"    OK (start_date={data['start_date']}, end_date={data['end_date']})")


def test_explicit_date_range():
    print(f">>> Check explicit date range ({YESTERDAY} ~ {TODAY}) ...")
    resp = get_data({"start_date": YESTERDAY, "end_date": TODAY})
    resp.raise_for_status()
    data = resp.json()
    assert data.get("start_date") == YESTERDAY, \
        f"Expected start_date={YESTERDAY}, got {data.get('start_date')}"
    assert data.get("end_date") == TODAY, \
        f"Expected end_date={TODAY}, got {data.get('end_date')}"
    print(f"    OK (start_date={data['start_date']}, end_date={data['end_date']})")


def test_future_date_no_data():
    print(f">>> Check future date returns empty rows ({TOMORROW}) ...")
    resp = get_data({"start_date": TOMORROW, "end_date": TOMORROW})
    resp.raise_for_status()
    data = resp.json()
    rows = data.get("rows") or []
    assert len(rows) == 0, \
        f"Expected empty rows for future date, got {len(rows)} rows"
    print(f"    OK (rows_count={len(rows)})")


# ─────────────────────────────────────────────
# group_by parameter
# ─────────────────────────────────────────────

def test_group_by_default():
    print(">>> Check default group_by (date,model) ...")
    resp = get_data()
    resp.raise_for_status()
    data = resp.json()
    group_by = data.get("group_by") or []
    assert "date" in group_by, \
        f"Default group_by should contain 'date', got {group_by}"
    assert "model" in group_by, \
        f"Default group_by should contain 'model', got {group_by}"
    print(f"    OK (group_by={group_by})")


def test_group_by_date():
    print(">>> Check group_by=date ...")
    resp = get_data({"group_by": "date", "start_date": TODAY, "end_date": TODAY})
    resp.raise_for_status()
    data = resp.json()
    assert "date" in (data.get("group_by") or []), \
        f"group_by should contain 'date', got {data.get('group_by')}"
    rows = data.get("rows") or []
    if len(rows) > 0:
        assert "date" in rows[0], f"rows[0] should have 'date' field when group_by=date"
    print(f"    OK (rows_count={len(rows)})")


def test_group_by_user():
    print(">>> Check group_by=user ...")
    resp = get_data({"group_by": "user", "start_date": TODAY, "end_date": TODAY})
    resp.raise_for_status()
    data = resp.json()
    assert "user" in (data.get("group_by") or []), \
        f"group_by should contain 'user', got {data.get('group_by')}"
    rows = data.get("rows") or []
    if len(rows) > 0:
        row = rows[0]
        # user_id, user_name 必须存在；user_email 使用 omitempty，用户未设置时不返回
        for field in ("user_id", "user_name"):
            assert field in row, f"rows[0] should have '{field}' field when group_by=user"
    print(f"    OK (rows_count={len(rows)})")


def test_group_by_model():
    print(">>> Check group_by=model ...")
    resp = get_data({"group_by": "model", "start_date": TODAY, "end_date": TODAY})
    resp.raise_for_status()
    data = resp.json()
    assert "model" in (data.get("group_by") or []), \
        f"group_by should contain 'model', got {data.get('group_by')}"
    rows = data.get("rows") or []
    if len(rows) > 0:
        row = rows[0]
        # ai_model_id 必须存在；model_name 使用 omitempty，DisplayName 为空时不返回
        assert "ai_model_id" in row, \
            f"rows[0] should have 'ai_model_id' field when group_by=model"
    print(f"    OK (rows_count={len(rows)})")


def test_group_by_instance():
    print(">>> Check group_by=instance ...")
    resp = get_data({"group_by": "instance", "start_date": TODAY, "end_date": TODAY})
    resp.raise_for_status()
    data = resp.json()
    assert "instance" in (data.get("group_by") or []), \
        f"group_by should contain 'instance', got {data.get('group_by')}"
    rows = data.get("rows") or []
    if len(rows) > 0:
        row = rows[0]
        for field in ("instance_id", "instance_name"):
            assert field in row, f"rows[0] should have '{field}' field when group_by=instance"
    print(f"    OK (rows_count={len(rows)})")


def test_group_by_combined():
    print(">>> Check group_by=user,model (combined) ...")
    resp = get_data({"group_by": "user,model", "start_date": TODAY, "end_date": TODAY})
    resp.raise_for_status()
    data = resp.json()
    group_by = data.get("group_by") or []
    assert "user" in group_by, f"group_by should contain 'user', got {group_by}"
    assert "model" in group_by, f"group_by should contain 'model', got {group_by}"
    print(f"    OK (group_by={group_by})")


def test_group_by_invalid_falls_back_to_default():
    print(">>> Check invalid group_by value falls back to default ...")
    resp = get_data({"group_by": "invalid_dimension"})
    resp.raise_for_status()
    data = resp.json()
    group_by = data.get("group_by") or []
    # invalid value is ignored, falls back to default: date + model
    assert "date" in group_by, \
        f"Invalid group_by should fall back to default containing 'date', got {group_by}"
    assert "model" in group_by, \
        f"Invalid group_by should fall back to default containing 'model', got {group_by}"
    print(f"    OK (group_by={group_by})")


# ─────────────────────────────────────────────
# order_by parameter
# ─────────────────────────────────────────────

def test_order_by_invalid():
    print(">>> Check order_by=invalid_field returns 400 ...")
    resp = get_data({"order_by": "invalid_field"})
    assert resp.status_code == 400, \
        f"Expected 400 for invalid order_by, got {resp.status_code}, body={resp.text}"
    data = resp.json()
    assert "error" in data, f"400 response should contain 'error' field, got {data}"
    print(f"    OK (status=400, error={data.get('error')})")


def test_order_by_total_tokens():
    print(">>> Check order_by=total_tokens (default) with order=desc ...")
    resp = get_data({"group_by": "user", "order_by": "total_tokens", "order": "desc"})
    resp.raise_for_status()
    data = resp.json()
    rows = data.get("rows") or []
    assert_sorted(rows, "total_tokens", reverse=True)
    print(f"    OK (rows_count={len(rows)})")


def test_order_by_request_count():
    print(">>> Check order_by=request_count with order=desc ...")
    resp = get_data({"group_by": "user", "order_by": "request_count", "order": "desc"})
    resp.raise_for_status()
    data = resp.json()
    rows = data.get("rows") or []
    assert_sorted(rows, "request_count", reverse=True)
    print(f"    OK (rows_count={len(rows)})")


# ─────────────────────────────────────────────
# Optional filters
# ─────────────────────────────────────────────

def test_filter_user_id():
    print(">>> Check filter by non-existent user_id ...")
    resp = get_data({"user_id": "999999999"})
    resp.raise_for_status()
    rows = resp.json().get("rows") or []
    assert len(rows) == 0, \
        f"Expected empty rows for non-existent user_id, got {len(rows)}"
    print(f"    OK (rows_count={len(rows)})")


def test_filter_ai_model_id():
    print(">>> Check filter by non-existent ai_model_id ...")
    resp = get_data({"ai_model_id": "999999999"})
    resp.raise_for_status()
    rows = resp.json().get("rows") or []
    assert len(rows) == 0, \
        f"Expected empty rows for non-existent ai_model_id, got {len(rows)}"
    print(f"    OK (rows_count={len(rows)})")


def test_filter_instance_id():
    print(">>> Check filter by non-existent instance_id ...")
    resp = get_data({"instance_id": "999999999"})
    resp.raise_for_status()
    rows = resp.json().get("rows") or []
    assert len(rows) == 0, \
        f"Expected empty rows for non-existent instance_id, got {len(rows)}"
    print(f"    OK (rows_count={len(rows)})")


def test_filter_group_id():
    print(">>> Check filter by non-existent group_id ...")
    resp = get_data({"group_id": "999999999"})
    resp.raise_for_status()
    rows = resp.json().get("rows") or []
    assert len(rows) == 0, \
        f"Expected empty rows for non-existent group_id, got {len(rows)}"
    print(f"    OK (rows_count={len(rows)})")


def test_filter_combined():
    print(">>> Check combined filters (all non-existent) ...")
    resp = get_data({
        "user_id": "999999999",
        "ai_model_id": "999999999",
        "instance_id": "999999999",
    })
    resp.raise_for_status()
    rows = resp.json().get("rows") or []
    assert len(rows) == 0, \
        f"Expected empty rows for combined non-existent filters, got {len(rows)}"
    print(f"    OK (rows_count={len(rows)})")


# ─────────────────────────────────────────────
# Auth
# ─────────────────────────────────────────────

def test_auth():
    """认证测试三件套"""
    auth_test_suite(lambda headers: get_data(headers=headers), label="usage/data")


# ─────────────────────────────────────────────
# Main
# ─────────────────────────────────────────────

def main():
    health_check()
    print()
    run_tests(globals(), title="GET /admin/usage/data")


if __name__ == "__main__":
    main()
