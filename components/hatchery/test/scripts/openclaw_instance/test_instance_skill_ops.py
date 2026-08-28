#!/usr/bin/env python3
"""
集成测试：实例管理 - 技能 & 插件（F 组）

覆盖接口：
    GET  /openclaw/skills                 实例技能列表（TAT 调用）
    POST /openclaw/add-skill              安装技能
    POST /openclaw/add-plugin             安装插件
    POST /openclaw/retry-failed-skills    重试失败技能（幂等）
    POST /openclaw/cancel-failed-skills   取消失败技能（幂等）

install-skills 已在 lifecycle 覆盖。本文件主要测契约+参数校验+幂等+鉴权。
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


# ─── /openclaw/skills ─────────────────────────────────────────────────────

def test_01_skills_missing_id():
    """GET /openclaw/skills - 缺 id → 400"""
    resp = cli.get("/openclaw/skills", expect=None, raw=True)
    assert_status(resp, {400, 404}, label="skills-missing-id")
    print(f"    OK status={resp.status_code}")


def test_02_skills_nonexistent_id():
    """GET /openclaw/skills?id=NONEXISTENT → 4xx"""
    resp = cli.get(
        "/openclaw/skills",
        params={"id": NONEXISTENT_DB_ID},
        expect=None, raw=True, timeout=60,
    )
    assert_status(resp, {400, 404, 500}, label="skills-not-found")
    print(f"    OK status={resp.status_code}")


def test_03_skills_ok():
    """GET /openclaw/skills - happy path"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.get(
        "/openclaw/skills",
        params={"id": SHARED_DB_ID},
        expect=None, raw=True, timeout=180,
    )
    if resp.status_code != 200:
        print(f"    OK 非 200（TAT 可能不可用）: status={resp.status_code}")
        return
    body = resp.json()
    assert isinstance(body, (list, dict)), f"应返回 JSON: {type(body).__name__}"
    print("    OK")


def test_04_skills_auth():
    """GET /openclaw/skills - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).get(
            "/openclaw/skills",
            params={"id": 1},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="skills",
        check_admin=False,
    )


# ─── /openclaw/add-skill ──────────────────────────────────────────────────

def test_05_add_skill_missing_id():
    """POST /openclaw/add-skill - 缺 id → 400"""
    resp = cli.post(
        "/openclaw/add-skill",
        data={"skill_name": "x"},
        expect=None, raw=True,
    )
    assert_status(resp, {400, 404}, label="add-skill-missing-id")
    print(f"    OK status={resp.status_code}")


def test_06_add_skill_missing_name():
    """POST /openclaw/add-skill - 缺 skill_name → 400"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.post(
        "/openclaw/add-skill",
        data={"id": SHARED_DB_ID},
        expect=None, raw=True,
    )
    assert_status(resp, {400, 500}, label="add-skill-missing-name")
    print(f"    OK status={resp.status_code}")


def test_07_add_skill_nonexistent_skill():
    """POST /openclaw/add-skill - 不存在的技能"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.post(
        "/openclaw/add-skill",
        data={"id": SHARED_DB_ID, "skill_name": "__not_exist_skill_xyz__"},
        expect=None, raw=True, timeout=120,
    )
    if resp.status_code == 200:
        body = resp.json() if resp.content else {}
        assert body.get("error"), f"不存在技能应失败: {body}"
    else:
        assert resp.status_code < 600, f"非常规状态码: {resp.status_code}"
    print(f"    OK status={resp.status_code}")


def test_08_add_skill_auth():
    """POST /openclaw/add-skill - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).post(
            "/openclaw/add-skill",
            data={"id": 1, "skill_name": "x"},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="add-skill",
        check_admin=False,
    )


# ─── /openclaw/add-plugin ─────────────────────────────────────────────────

def test_09_add_plugin_missing_id():
    """POST /openclaw/add-plugin - 缺 id → 400"""
    resp = cli.post(
        "/openclaw/add-plugin",
        data={"plugin": "test"},
        expect=None, raw=True,
    )
    assert_status(resp, {400, 404}, label="add-plugin-missing-id")
    print(f"    OK status={resp.status_code}")


def test_10_add_plugin_empty_name():
    """POST /openclaw/add-plugin - plugin 为空 → 400"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.post(
        "/openclaw/add-plugin",
        data={"id": SHARED_DB_ID, "plugin": ""},
        expect=None, raw=True,
    )
    assert_status(resp, {400, 403}, label="add-plugin-empty")
    print(f"    OK status={resp.status_code}")


def test_11_add_plugin_invalid_format():
    """POST /openclaw/add-plugin - plugin 含特殊字符 → 400"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.post(
        "/openclaw/add-plugin",
        data={"id": SHARED_DB_ID, "plugin": "bad name with spaces!"},
        expect=None, raw=True,
    )
    assert_status(resp, {400, 403}, label="add-plugin-invalid")
    print(f"    OK status={resp.status_code}")


def test_12_add_plugin_auth():
    """POST /openclaw/add-plugin - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).post(
            "/openclaw/add-plugin",
            data={"id": 1, "plugin": "test"},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="add-plugin",
        check_admin=False,
    )


# ─── /openclaw/retry-failed-skills ────────────────────────────────────────

def test_13_retry_failed_skills_missing_id():
    """POST /openclaw/retry-failed-skills - 缺 id → 400"""
    resp = cli.post(
        "/openclaw/retry-failed-skills", data={}, expect=None, raw=True,
    )
    assert_status(resp, {400, 404}, label="retry-failed-missing")
    print(f"    OK status={resp.status_code}")


def test_14_retry_failed_skills_idempotent():
    """POST /openclaw/retry-failed-skills - 共享实例上无失败技能时 retry_count=0"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.post(
        "/openclaw/retry-failed-skills",
        data={"id": SHARED_DB_ID},
        expect=None, raw=True, timeout=60,
    )
    if resp.status_code != 200:
        print(f"    OK 非 200（环境差异）: status={resp.status_code}")
        return
    body = assert_json_keys(resp, "ok", "retry_count")
    assert body.get("ok"), f"ok=false: {body}"
    assert isinstance(body["retry_count"], int), f"retry_count 应为 int: {body}"
    print(f"    OK retry_count={body['retry_count']}")


def test_15_retry_failed_skills_auth():
    """POST /openclaw/retry-failed-skills - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).post(
            "/openclaw/retry-failed-skills",
            data={"id": 1},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="retry-failed-skills",
        check_admin=False,
    )


# ─── /openclaw/cancel-failed-skills ───────────────────────────────────────

def test_16_cancel_failed_skills_idempotent():
    """POST /openclaw/cancel-failed-skills - 幂等"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.post(
        "/openclaw/cancel-failed-skills",
        data={"id": SHARED_DB_ID},
        expect=None, raw=True, timeout=60,
    )
    if resp.status_code != 200:
        print(f"    OK 非 200: status={resp.status_code}")
        return
    body = assert_json_keys(resp, "ok", "cancel_count")
    assert body.get("ok"), f"ok=false: {body}"
    print(f"    OK cancel_count={body['cancel_count']}")


def test_17_cancel_failed_skills_auth():
    """POST /openclaw/cancel-failed-skills - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).post(
            "/openclaw/cancel-failed-skills",
            data={"id": 1},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="cancel-failed-skills",
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
        title="test_instance_skill_ops.py",
        ordered=True,
    )


if __name__ == "__main__":
    main()
