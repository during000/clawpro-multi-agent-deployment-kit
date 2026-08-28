#!/usr/bin/env python3
"""
集成测试：实例管理 - POST /openclaw/create 参数校验（B 组）

只测异常路径（正常创建链路在 test_instance_lifecycle.py 已覆盖）：
    - name 为空 → 400
    - name 超 128 字符 → 400
    - role_id 不存在 → 400
    - agent_type 不存在/无对应启用镜像 → 400/500
    - user_data 非合法 base64（开关开启时）→ 400 / 未开启时 → 403
    - tags 非 JSON 数组 → 400
    - 认证三件套
"""
import os
import sys
import time

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers.api import (
    ApiClient,
    health_check, run_tests,
    auth_test_suite, assert_status,
)
from helpers import config
from _instance_helpers import (
    cli,
    assert_error_message,
)


# ─── 工具：生成不会真创建实例的入参（保证失败前置发生） ─────────────

def _safe_name():
    # 即便意外创建成功，也带前缀以便清理
    return f"{config.INSTANCE_NAME_PREFIX}validation-{int(time.time())}-{os.getpid()}"


# ─── 用例 ─────────────────────────────────────────────────────────────────

def test_01_create_empty_name():
    """POST /openclaw/create - name 为空 → 400"""
    resp = cli.post(
        "/openclaw/create",
        data={"name": ""},
        expect=None, raw=True,
    )
    assert_status(resp, 400, label="create-empty-name")
    assert_error_message(resp, "实例名称")
    print("    OK")


def test_02_create_name_too_long():
    """POST /openclaw/create - name 超 128 字符 → 400"""
    long_name = "x" * 129
    resp = cli.post(
        "/openclaw/create",
        data={"name": long_name},
        expect=None, raw=True,
    )
    assert_status(resp, 400, label="create-name-too-long")
    assert_error_message(resp, "128", "实例名称")
    print("    OK")


def test_03_create_invalid_role_id():
    """POST /openclaw/create - role_id 不存在 → 400"""
    resp = cli.post(
        "/openclaw/create",
        data={
            "name": _safe_name(),
            "role_id": "999999999",
        },
        expect=None, raw=True,
    )
    assert_status(resp, {400, 403, 404}, label="create-invalid-role")
    print(f"    OK status={resp.status_code}")


def test_04_create_invalid_agent_type():
    """POST /openclaw/create - agent_type 无对应启用镜像 → 400/500"""
    resp = cli.post(
        "/openclaw/create",
        data={
            "name": _safe_name(),
            "agent_type": "__not_exist_agent_type_xyz__",
        },
        expect=None, raw=True,
    )
    assert resp.status_code != 200, (
        f"未知 agent_type 应失败, 实际 200: {resp.text[:200]}"
    )
    assert resp.status_code < 600, f"非常规状态码: {resp.status_code}"
    print(f"    OK status={resp.status_code}")


def test_05_create_invalid_user_data():
    """POST /openclaw/create - user_data 非合法 base64 → 400/403"""
    resp = cli.post(
        "/openclaw/create",
        data={
            "name": _safe_name(),
            "user_data": "!!!not-valid-base64@@@",
        },
        expect=None, raw=True,
    )
    # 开关未开启 → 403；开启后非法 base64 → 400
    assert_status(resp, {400, 403}, label="create-invalid-user-data")
    if resp.status_code == 400:
        assert_error_message(resp, "user_data", "base64")
    print(f"    OK status={resp.status_code}")


def test_06_create_user_data_too_large():
    """POST /openclaw/create - user_data 超 12KB → 400/403"""
    huge = "A" * (13 * 1024)
    resp = cli.post(
        "/openclaw/create",
        data={
            "name": _safe_name(),
            "user_data": huge,
        },
        expect=None, raw=True,
    )
    assert_status(resp, {400, 403}, label="create-user-data-too-large")
    print(f"    OK status={resp.status_code}")


def test_07_create_invalid_group_id():
    """POST /openclaw/create - group_id 不存在 → 400"""
    resp = cli.post(
        "/openclaw/create",
        data={
            "name": _safe_name(),
            "group_id": "999999999",
        },
        expect=None, raw=True,
    )
    assert_status(resp, {400, 403, 404}, label="create-invalid-group")
    print(f"    OK status={resp.status_code}")


def test_08_create_invalid_tags():
    """POST /openclaw/create - tags 必须为 JSON 数组字符串 → 400"""
    resp = cli.post(
        "/openclaw/create",
        data={
            "name": _safe_name(),
            "tags": '{"key":"env","value":"prod"}',
        },
        expect=None, raw=True,
    )
    assert_status(resp, 400, label="create-invalid-tags")
    print("    OK")


def test_09_create_auth():
    """POST /openclaw/create - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).post(
            "/openclaw/create",
            data={"name": "auth-test"},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="create",
        check_admin=False,
    )


# ─── 入口 ────────────────────────────────────────────────────────────────

def main():
    health_check()
    print()

    run_tests(
        globals(),
        title="test_instance_create_validation.py",
        ordered=True,
    )


if __name__ == "__main__":
    main()
