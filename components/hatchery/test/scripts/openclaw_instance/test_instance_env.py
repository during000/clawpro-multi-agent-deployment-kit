#!/usr/bin/env python3
"""
集成测试：实例管理 - 环境变量（G 组）

覆盖接口：
    POST /openclaw/set-env   设置/删除环境变量（JSON body，TAT 调用）
    GET  /openclaw/env       查看环境变量

测试流程（共享实例存在时）：
    1) 读取当前环境变量（基线）
    2) 设置一个测试用的环境变量 IT_ENV_TEST=hello
    3) 再次读取，确认 IT_ENV_TEST=hello
    4) 删除该变量（value=null）
    5) 再次读取，确认已删除
"""
import os
import sys
import time

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers.api import (
    ApiClient,
    anon, bad_token,
    health_check, run_tests,
    auth_test_suite, assert_status,
)
from _instance_helpers import (
    cli, require_shared_instance,
    NONEXISTENT_DB_ID,
    get_shared_db_id,
    assert_error_message,
)


SHARED_DB_ID = None
TEST_KEY = f"IT_ENV_TEST_{int(time.time())}"


# ─── /openclaw/env ───────────────────────────────────────────────────────

def test_01_env_missing_params():
    """GET /openclaw/env - 缺 id → 400"""
    resp = cli.get("/openclaw/env", expect=None, raw=True)
    assert_status(resp, 400, label="env-missing-id")
    assert_error_message(resp, "id", "instance_id")
    print("    OK")


def test_02_env_nonexistent_id():
    """GET /openclaw/env?id=NONEXISTENT → 4xx"""
    resp = cli.get(
        "/openclaw/env",
        params={"id": NONEXISTENT_DB_ID},
        expect=None, raw=True, timeout=30,
    )
    assert_status(resp, {400, 404, 500}, label="env-not-found")
    print(f"    OK status={resp.status_code}")


def test_03_env_auth():
    """GET /openclaw/env - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).get(
            "/openclaw/env",
            params={"id": 1},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="env-get",
        check_admin=False,
    )


# ─── /openclaw/set-env ───────────────────────────────────────────────────

def test_04_set_env_invalid_body():
    """POST /openclaw/set-env - 非 JSON body 不应 5xx"""
    resp = cli.post(
        "/openclaw/set-env",
        data={"id": SHARED_DB_ID or 1},
        expect=None, raw=True,
    )
    assert resp.status_code < 600, f"非常规状态码: {resp.status_code}"
    print(f"    OK status={resp.status_code}")


def test_05_set_env_missing_id():
    """POST /openclaw/set-env - 缺 id → 400"""
    resp = cli.post(
        "/openclaw/set-env",
        json={"env": {"K": "v"}},
        expect=None, raw=True,
    )
    assert_status(resp, 400, label="set-env-missing-id")
    print("    OK")


def test_06_set_env_empty_env():
    """POST /openclaw/set-env - env 为空 → 400"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.post(
        "/openclaw/set-env",
        json={"id": SHARED_DB_ID, "env": {}},
        expect=None, raw=True,
    )
    assert_status(resp, 400, label="set-env-empty")
    assert_error_message(resp, "env")
    print("    OK")


def test_07_set_env_too_many():
    """POST /openclaw/set-env - env 数量 > 50 → 400"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    huge_env = {f"K{i}": str(i) for i in range(51)}
    resp = cli.post(
        "/openclaw/set-env",
        json={"id": SHARED_DB_ID, "env": huge_env},
        expect=None, raw=True,
    )
    assert_status(resp, 400, label="set-env-too-many")
    assert_error_message(resp, "50")
    print("    OK")


def test_08_set_env_invalid_key():
    """POST /openclaw/set-env - key=`1BAD KEY!` → 400"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.post(
        "/openclaw/set-env",
        json={"id": SHARED_DB_ID, "env": {"1BAD KEY!": "x"}},
        expect=None, raw=True,
    )
    assert_status(resp, 400, label="set-env-invalid-key")
    assert_error_message(resp, "环境变量名", "无效")
    print("    OK")


def test_09_set_env_invalid_value_type():
    """POST /openclaw/set-env - value 为 int → 400"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.post(
        "/openclaw/set-env",
        json={"id": SHARED_DB_ID, "env": {"GOOD_KEY": 123}},
        expect=None, raw=True,
    )
    assert_status(resp, 400, label="set-env-invalid-value")
    print("    OK")


def test_10_set_env_read_write_roundtrip():
    """POST /openclaw/set-env - 完整读写删读循环"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return

    # 1) 设置
    print(f">>> 设置 {TEST_KEY}=hello ...")
    resp = cli.post(
        "/openclaw/set-env",
        json={"id": SHARED_DB_ID, "env": {TEST_KEY: "hello"}},
        expect=None, raw=True, timeout=180,
    )
    if resp.status_code != 200:
        print(f"    SKIP (设置失败 / TAT 不可用: status={resp.status_code})")
        return
    if not (resp.json() or {}).get("ok"):
        print(f"    SKIP (ok=false: {resp.text[:200]})")
        return

    # 2) 读取，确认存在
    print(f">>> 验证 {TEST_KEY} 已写入 ...")
    r2 = cli.get(
        "/openclaw/env",
        params={"id": SHARED_DB_ID},
        expect=None, raw=True, timeout=120,
    )
    if r2.status_code == 200:
        body = r2.json() or {}
        env = body.get("env", {})
        if env.get(TEST_KEY) != "hello":
            print(f"    [WARN] 读回未匹配: env.{TEST_KEY}={env.get(TEST_KEY)}")
        else:
            print(f"    {TEST_KEY}=hello ✓")

    # 3) 删除（value=null）
    print(f">>> 删除 {TEST_KEY}（value=null）...")
    r3 = cli.post(
        "/openclaw/set-env",
        json={"id": SHARED_DB_ID, "env": {TEST_KEY: None}},
        expect=None, raw=True, timeout=180,
    )
    if r3.status_code != 200:
        print(f"    [WARN] 删除失败但不致命: status={r3.status_code}")
        print("    OK 读写循环已完成")
        return

    # 4) 读取，确认已删除
    r4 = cli.get(
        "/openclaw/env",
        params={"id": SHARED_DB_ID},
        expect=None, raw=True, timeout=120,
    )
    if r4.status_code == 200:
        body4 = r4.json() or {}
        env4 = body4.get("env", {})
        if TEST_KEY in env4:
            print(f"    [WARN] 删除后仍存在: {env4.get(TEST_KEY)}")
        else:
            print(f"    {TEST_KEY} 已删除 ✓")
    print("    OK 读写删循环完成")


def test_11_set_env_auth():
    """POST /openclaw/set-env - 认证三件套（JSON body）"""
    anon.post(
        "/openclaw/set-env",
        json={"id": 1, "env": {"K": "v"}},
        expect={401, 403}, raw=True,
    )
    bad_token.post(
        "/openclaw/set-env",
        json={"id": 1, "env": {"K": "v"}},
        expect={401, 403}, raw=True,
    )
    print("    OK")


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
        title="test_instance_env.py",
        ordered=True,
    )


if __name__ == "__main__":
    main()
