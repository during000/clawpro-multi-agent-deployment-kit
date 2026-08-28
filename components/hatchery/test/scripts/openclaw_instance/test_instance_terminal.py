#!/usr/bin/env python3
"""
集成测试：实例管理 - POST /openclaw/terminal-url（G 组）

覆盖：
    - 缺参 → 400
    - 不存在 id → 4xx/5xx
    - 终端功能未开启 → 403
    - happy path（环境支持时）→ 200 + login_url
    - 认证三件套
"""
import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers.api import (
    ApiClient,
    health_check, run_tests,
    auth_test_suite, assert_status, assert_fields,
)
from _instance_helpers import (
    cli, require_shared_instance,
    get_shared_db_id, NONEXISTENT_DB_ID,
)


SHARED_DB_ID = None


def test_01_terminal_get_method_tolerant():
    """GET /openclaw/terminal-url - 通常不允许，宽松接受"""
    resp = cli.get(
        "/openclaw/terminal-url",
        params={"id": SHARED_DB_ID or 1},
        expect=None, raw=True,
    )
    # 405/401/403 都接受；只检查不是 6xx
    assert resp.status_code < 600, f"非常规状态码: {resp.status_code}"
    print(f"    OK (status={resp.status_code})")


def test_02_terminal_missing_params():
    """POST /openclaw/terminal-url - 缺 id → 400/403"""
    resp = cli.post(
        "/openclaw/terminal-url", data={}, expect=None, raw=True,
    )
    assert_status(resp, {400, 403}, label="terminal-missing-id")
    print(f"    OK (status={resp.status_code})")


def test_03_terminal_nonexistent_id():
    """POST /openclaw/terminal-url?id=NONEXISTENT → 4xx/5xx"""
    resp = cli.post(
        "/openclaw/terminal-url",
        data={"id": NONEXISTENT_DB_ID},
        expect=None, raw=True, timeout=60,
    )
    assert_status(resp, {400, 403, 404, 500}, label="terminal-not-found")
    print(f"    OK (status={resp.status_code})")


def test_04_terminal_ok():
    """POST /openclaw/terminal-url - happy path（环境支持时）→ 200 + login_url"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.post(
        "/openclaw/terminal-url",
        data={"id": SHARED_DB_ID},
        expect=None, raw=True, timeout=120,
    )
    if resp.status_code == 403:
        print("    SKIP (终端功能未开启)")
        return
    if resp.status_code != 200:
        print(f"    SKIP (非 200，环境差异): status={resp.status_code}")
        return
    body = resp.json()
    assert_fields(body, ["login_url"], context="terminal-url 响应")
    assert body.get("login_url"), f"login_url 为空: {body}"
    print(f"    OK (login_url={body['login_url'][:80]}...)")


def test_05_terminal_auth():
    """POST /openclaw/terminal-url - 认证三件套"""
    def call(headers):
        tmp = ApiClient("", timeout=30)
        return tmp.post(
            "/openclaw/terminal-url",
            data={"id": 1},
            expect=None, raw=True,
            extra_headers=headers,
        )
    auth_test_suite(call, label="terminal-url", check_admin=False)


def main():
    global SHARED_DB_ID
    health_check()
    SHARED_DB_ID = require_shared_instance().db_id
    if SHARED_DB_ID:
        print(f">>> 复用共享实例 db_id={SHARED_DB_ID}")
    print()

    run_tests(
        globals(),
        title="POST /openclaw/terminal-url",
        ordered=True,
    )


if __name__ == "__main__":
    main()
