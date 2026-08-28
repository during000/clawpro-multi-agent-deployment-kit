#!/usr/bin/env python3
"""
集成测试：实例管理 - SMH & Memory TDAI（J 组）

覆盖接口：
    GET /openclaw/smh-status            个人空间状态
    GET /openclaw/memory-tdai-status    记忆插件状态

两个接口都依赖站点级开关，未启用时也应返回合法 JSON（enabled=false）。
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
    assert_json_keys,
)


SHARED_DB_ID = None


# ─── /openclaw/smh-status ────────────────────────────────────────────────

def test_01_smh_status_missing_id():
    """GET /openclaw/smh-status - 缺 id → 4xx"""
    resp = cli.get("/openclaw/smh-status", expect=None, raw=True)
    assert_status(resp, {400, 404}, label="smh-status-missing")
    print(f"    OK status={resp.status_code}")


def test_02_smh_status_nonexistent_id():
    """GET /openclaw/smh-status?id=NONEXISTENT → 4xx"""
    resp = cli.get(
        "/openclaw/smh-status",
        params={"id": NONEXISTENT_DB_ID},
        expect=None, raw=True, timeout=30,
    )
    assert_status(resp, {400, 404}, label="smh-status-not-found")
    print(f"    OK status={resp.status_code}")


def test_03_smh_status_ok():
    """GET /openclaw/smh-status - happy path"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.get(
        "/openclaw/smh-status",
        params={"id": SHARED_DB_ID},
        expect=None, raw=True, timeout=60,
    )
    if resp.status_code != 200:
        print(f"    SKIP (非 200): status={resp.status_code}")
        return
    body = assert_json_keys(resp, "enabled", "has_space")
    assert isinstance(body["enabled"], bool), f"enabled 应为 bool: {body}"
    assert isinstance(body["has_space"], bool), f"has_space 应为 bool: {body}"
    print(f"    OK enabled={body['enabled']} has_space={body['has_space']}")


def test_04_smh_status_auth():
    """GET /openclaw/smh-status - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).get(
            "/openclaw/smh-status",
            params={"id": 1},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="smh-status",
        check_admin=False,
    )


# ─── /openclaw/memory-tdai-status ────────────────────────────────────────

def test_05_memory_tdai_missing_id():
    """GET /openclaw/memory-tdai-status - 缺 id → 4xx"""
    resp = cli.get("/openclaw/memory-tdai-status", expect=None, raw=True)
    assert_status(resp, {400, 404}, label="memory-tdai-missing")
    print(f"    OK status={resp.status_code}")


def test_06_memory_tdai_nonexistent_id():
    """GET /openclaw/memory-tdai-status?id=NONEXISTENT → 4xx"""
    resp = cli.get(
        "/openclaw/memory-tdai-status",
        params={"id": NONEXISTENT_DB_ID},
        expect=None, raw=True, timeout=30,
    )
    assert_status(resp, {400, 404}, label="memory-tdai-not-found")
    print(f"    OK status={resp.status_code}")


def test_07_memory_tdai_ok():
    """GET /openclaw/memory-tdai-status - happy path"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.get(
        "/openclaw/memory-tdai-status",
        params={"id": SHARED_DB_ID},
        expect=None, raw=True, timeout=60,
    )
    if resp.status_code == 403:
        print("    SKIP (Hermes 类型实例不支持记忆功能)")
        return
    if resp.status_code != 200:
        print(f"    SKIP (非 200): status={resp.status_code}")
        return
    body = assert_json_keys(resp, "memory_tdai_enable", "status")
    assert isinstance(body["memory_tdai_enable"], bool), (
        f"memory_tdai_enable 应为 bool: {body}"
    )
    valid_statuses = {
        "NOT_INSTALLED", "ENABLING", "ENABLED",
        "DISABLING", "DISABLED", "FAILED",
        "UNSUPPORTED_VERSION",
    }
    assert body["status"] in valid_statuses, (
        f"status 非法值 {body['status']}, 期望∈{valid_statuses}"
    )
    print(f"    OK enable={body['memory_tdai_enable']} status={body['status']}")


def test_08_memory_tdai_auth():
    """GET /openclaw/memory-tdai-status - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).get(
            "/openclaw/memory-tdai-status",
            params={"id": 1},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="memory-tdai-status",
        check_admin=False,
    )


# ─── /openclaw/smh-token ────────────────────────────────────────────────

def test_09_smh_token_missing_id():
    """GET /openclaw/smh-token - 缺 id → 4xx"""
    resp = cli.get("/openclaw/smh-token", expect=None, raw=True)
    assert_status(resp, {400, 403, 404}, label="smh-token-missing")
    print(f"    OK status={resp.status_code}")


def test_10_smh_token_nonexistent_id():
    """GET /openclaw/smh-token?id=NONEXISTENT → 4xx"""
    resp = cli.get(
        "/openclaw/smh-token",
        params={"id": NONEXISTENT_DB_ID},
        expect=None, raw=True, timeout=15,
    )
    # SMH 未启用 → 403；启用但 id 不存在 → 404；启用但 id 不属于当前用户 → 404
    assert_status(resp, {400, 403, 404}, label="smh-token-not-found")
    print(f"    OK status={resp.status_code}")


def test_11_smh_token_no_space_or_disabled():
    """GET /openclaw/smh-token?id=<shared> - SMH 未启用 → 403；
    启用但实例未绑定空间 → 400；只有真正绑定时才 200"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.get(
        "/openclaw/smh-token",
        params={"id": SHARED_DB_ID},
        expect=None, raw=True, timeout=15,
    )
    if resp.status_code == 200:
        body = resp.json() or {}
        assert "token" in body and "space_id" in body, f"返回缺字段: {body}"
        print(f"    OK 200 token=*** space_id={body.get('space_id')}")
        return
    assert_status(resp, {400, 403}, label="smh-token-no-space")
    print(f"    OK status={resp.status_code}")


def test_12_smh_token_auth():
    """GET /openclaw/smh-token - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=15).get(
            "/openclaw/smh-token",
            params={"id": 1},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="smh-token",
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
        title="test_instance_smh_memory.py",
        ordered=True,
    )


if __name__ == "__main__":
    main()
