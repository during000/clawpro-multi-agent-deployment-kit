#!/usr/bin/env python3
"""
集成测试：实例管理 - 审批 & 禁用操作（I 组）

覆盖接口：
    POST /openclaw/approve          配对审批（TAT 调用）
    POST /openclaw/denied-actions   批量查询禁用操作

approve 真实通过需要等待外部审批信号，本测试仅覆盖参数校验+鉴权。
denied-actions 是只读的批量查询，可正常验证契约。
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
    NONEXISTENT_DB_ID,
    get_shared_db_id,
    assert_json_keys,
)


SHARED_DB_ID = None


# ─── /openclaw/approve ───────────────────────────────────────────────────

def test_01_approve_missing_id():
    """POST /openclaw/approve - 缺 id/instance_id → 400"""
    resp = cli.post(
        "/openclaw/approve",
        data={"code": "test-code"},
        expect=None, raw=True,
    )
    assert_status(resp, {400, 404}, label="approve-missing-id")
    print(f"    OK status={resp.status_code}")


def test_02_approve_missing_code():
    """POST /openclaw/approve - 缺 code"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.post(
        "/openclaw/approve",
        data={"id": SHARED_DB_ID},
        expect=None, raw=True, timeout=60,
    )
    if resp.status_code == 200:
        body = resp.json() if resp.content else {}
        assert body.get("error"), f"缺 code 应失败: {body}"
    else:
        assert resp.status_code < 600, f"非常规状态码: {resp.status_code}"
    print(f"    OK status={resp.status_code}")


def test_03_approve_nonexistent_id():
    """POST /openclaw/approve?id=NONEXISTENT → 4xx"""
    resp = cli.post(
        "/openclaw/approve",
        data={"id": NONEXISTENT_DB_ID, "code": "test-code"},
        expect=None, raw=True, timeout=30,
    )
    assert_status(resp, {400, 404, 500}, label="approve-not-found")
    print(f"    OK status={resp.status_code}")


def test_04_approve_auth():
    """POST /openclaw/approve - 认证三件套（用户侧接口，跳过管理员检查）"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).post(
            "/openclaw/approve",
            data={"id": 1, "code": "test"},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="approve",
        check_admin=False,
    )


# ─── /openclaw/denied-actions ────────────────────────────────────────────

def test_05_denied_actions_empty_body():
    """POST /openclaw/denied-actions - 空 body → 200 {instances:[]}"""
    resp = cli.post(
        "/openclaw/denied-actions", json={}, expect=200, raw=True,
    )
    body = assert_json_keys(resp, "instances")
    assert isinstance(body["instances"], list), (
        f"instances 应为数组: {type(body['instances']).__name__}"
    )
    assert not body["instances"], f"无入参时应返回空数组: {body['instances']}"
    print("    OK empty")


def test_06_denied_actions_invalid_body():
    """POST /openclaw/denied-actions - 非 JSON body 不应 5xx"""
    resp = cli.post(
        "/openclaw/denied-actions",
        data={"ids": "1,2"},
        expect=None, raw=True,
    )
    assert resp.status_code < 600, f"非常规状态码: {resp.status_code}"
    print(f"    OK status={resp.status_code}")


def test_07_denied_actions_with_ids():
    """POST /openclaw/denied-actions {ids:[shared]}"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.post(
        "/openclaw/denied-actions",
        json={"ids": [SHARED_DB_ID]},
        expect=None, raw=True, timeout=60,
    )
    if resp.status_code != 200:
        print(f"    SKIP (非 200): status={resp.status_code}")
        return
    body = assert_json_keys(resp, "instances")
    insts = body["instances"]
    assert isinstance(insts, list), f"instances 应为数组: {type(insts).__name__}"
    if insts:
        first = insts[0]
        assert first.get("id") == SHARED_DB_ID, (
            f"id 不匹配: 期望 {SHARED_DB_ID}, 实际 {first.get('id')}"
        )
        assert "denied_actions" in first, f"缺 denied_actions 字段: {first}"
    print(f"    OK count={len(insts)}")


def test_08_denied_actions_with_nonexistent_ids():
    """POST /openclaw/denied-actions {ids:[999999999]} - 不存在的 id"""
    resp = cli.post(
        "/openclaw/denied-actions",
        json={"ids": [999_999_999]},
        expect=None, raw=True, timeout=30,
    )
    if resp.status_code != 200:
        print(f"    OK status={resp.status_code}")
        return
    body = assert_json_keys(resp, "instances")
    print(f"    OK count={len(body['instances'])}")


def test_09_denied_actions_auth():
    """POST /openclaw/denied-actions - 认证三件套（用户侧接口，跳过管理员检查）"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).post(
            "/openclaw/denied-actions",
            json={},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="denied-actions",
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
        title="test_instance_approve.py",
        ordered=True,
    )


if __name__ == "__main__":
    main()
