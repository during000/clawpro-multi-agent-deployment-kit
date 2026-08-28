#!/usr/bin/env python3
"""
集成测试：实例管理 - POST /openclaw/reset & /openclaw/retry（B 组）

注意：reset 是破坏性操作（重装系统会丢失数据），本测试**不在共享实例上执行
真实重装**，仅覆盖：
    - reset 缺参 → 400
    - reset 不存在 id → 4xx
    - retry 缺参 → 400
    - retry 不存在 id → 4xx
    - retry 当前状态非 load_failed → 400「当前状态为 xxx，只有 load_failed
      状态才能重试」
    - 认证三件套
"""
import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers.api import (
    ApiClient,
    health_check, run_tests,
    auth_test_suite, assert_status,
)
from _instance_helpers import (
    cli, require_shared_instance,
    get_shared_db_id, NONEXISTENT_DB_ID,
    assert_error_message,
)


SHARED_DB_ID = None


# ─── reset 用例 ───────────────────────────────────────────────────────────

def test_01_reset_missing_params():
    """POST /openclaw/reset - 缺 id/instance_id → 400"""
    resp = cli.post("/openclaw/reset", data={}, expect=None, raw=True)
    assert_status(resp, 400, label="reset-missing-id")
    print(f"    OK status={resp.status_code}")


def test_02_reset_nonexistent_id():
    """POST /openclaw/reset?id=NONEXISTENT → 4xx"""
    resp = cli.post(
        "/openclaw/reset",
        data={"id": NONEXISTENT_DB_ID},
        expect=None, raw=True, timeout=60,
    )
    assert_status(resp, {400, 404, 500}, label="reset-not-found")
    print(f"    OK status={resp.status_code}")


def test_03_reset_auth():
    """POST /openclaw/reset - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).post(
            "/openclaw/reset",
            data={"id": 1},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="reset",
        check_admin=False,
    )


# ─── retry 用例 ───────────────────────────────────────────────────────────

def test_04_retry_missing_params():
    """POST /openclaw/retry - 缺 id/instance_id → 400"""
    resp = cli.post("/openclaw/retry", data={}, expect=None, raw=True)
    assert_status(resp, 400, label="retry-missing-id")
    print(f"    OK status={resp.status_code}")


def test_05_retry_nonexistent_id():
    """POST /openclaw/retry?id=NONEXISTENT → 4xx"""
    resp = cli.post(
        "/openclaw/retry",
        data={"id": NONEXISTENT_DB_ID},
        expect=None, raw=True, timeout=60,
    )
    assert_status(resp, {400, 404, 500}, label="retry-not-found")
    print(f"    OK status={resp.status_code}")


def test_06_retry_state_reject():
    """POST /openclaw/retry - 共享实例当前应为 running，retry 应返回 400"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.post(
        "/openclaw/retry",
        data={"id": SHARED_DB_ID},
        expect=None, raw=True, timeout=60,
    )
    # API.md 规定 400「当前状态为 xxx，只有 load_failed 状态才能重试」
    assert_status(resp, {400, 409}, label="retry-state-reject")
    if resp.status_code == 400:
        assert_error_message(resp, "load_failed", "重试")
    print(f"    OK status={resp.status_code}")


def test_07_retry_auth():
    """POST /openclaw/retry - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).post(
            "/openclaw/retry",
            data={"id": 1},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="retry",
        check_admin=False,
    )


# ─── 入口 ────────────────────────────────────────────────────────────────

def main():
    global SHARED_DB_ID
    health_check()
    SHARED_DB_ID = require_shared_instance().db_id
    if SHARED_DB_ID:
        print(f">>> 复用共享实例 db_id={SHARED_DB_ID}")
    print()

    run_tests(
        globals(),
        title="test_instance_reset_retry.py",
        ordered=True,
    )


if __name__ == "__main__":
    main()
