#!/usr/bin/env python3
"""
集成测试：实例管理 - POST /openclaw/rename（B 组）

覆盖：
    - happy path：改名 → 反查 list 确认
    - 缺参（id/instance_id 都不传）→ 400
    - name 为空 → 400
    - name 超 128 字符 → 400
    - 不存在 id → 404/400
    - 改名后回滚（保持原名，不污染共享实例）
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
from _instance_helpers import (
    cli, require_shared_instance,
    NONEXISTENT_DB_ID,
    get_shared_db_id,
    find_instance_by_db_id,
    assert_error_message,
)


SHARED_DB_ID = None
ORIGINAL_NAME = None


def test_01_rename_missing_params():
    """POST /openclaw/rename - 缺 id/instance_id → 400"""
    resp = cli.post(
        "/openclaw/rename",
        data={"name": "x"},
        expect=None, raw=True,
    )
    assert_status(resp, 400, label="rename-missing-id")
    assert_error_message(resp, "id", "instance_id")
    print("    OK")


def test_02_rename_empty_name():
    """POST /openclaw/rename - name 为空 → 400"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.post(
        "/openclaw/rename",
        data={"id": SHARED_DB_ID, "name": ""},
        expect=None, raw=True,
    )
    assert_status(resp, 400, label="rename-empty-name")
    assert_error_message(resp, "实例名称")
    print("    OK")


def test_03_rename_name_too_long():
    """POST /openclaw/rename - name 超 128 字符 → 400"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.post(
        "/openclaw/rename",
        data={"id": SHARED_DB_ID, "name": "x" * 129},
        expect=None, raw=True,
    )
    assert_status(resp, 400, label="rename-name-too-long")
    print("    OK")


def test_04_rename_nonexistent_id():
    """POST /openclaw/rename?id=NONEXISTENT → 4xx"""
    resp = cli.post(
        "/openclaw/rename",
        data={"id": NONEXISTENT_DB_ID, "name": "any-name"},
        expect=None, raw=True,
    )
    assert_status(resp, {400, 404}, label="rename-not-found")
    print(f"    OK status={resp.status_code}")


def test_05_rename_ok_and_rollback():
    """POST /openclaw/rename - happy path + 回滚（避免污染共享实例）"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    new_name = f"{ORIGINAL_NAME}-rn{int(time.time()) % 100000}"
    resp = cli.post(
        "/openclaw/rename",
        data={"id": SHARED_DB_ID, "name": new_name},
        expect=None, raw=True, timeout=60,
    )
    if resp.status_code == 409:
        print(f"    SKIP (409 状态冲突: {resp.text[:120]})")
        return
    assert_status(resp, 200, label="rename-ok")

    # 反查确认改名生效
    inst = find_instance_by_db_id(SHARED_DB_ID)
    assert inst, f"改名后未在 list 中找到 db_id={SHARED_DB_ID}"
    cur = inst.get("name") or inst.get("Name")
    assert cur == new_name, f"改名未生效: list 返回 {cur}"

    # 回滚（避免影响后续用例）
    rb = cli.post(
        "/openclaw/rename",
        data={"id": SHARED_DB_ID, "name": ORIGINAL_NAME},
        expect=None, raw=True, timeout=60,
    )
    if rb.status_code != 200:
        print(f"    [WARN] 回滚改名失败 status={rb.status_code} resp={rb.text[:120]}")
    print(f"    OK 改名生效并已回滚 → {ORIGINAL_NAME}")


def test_06_rename_auth():
    """POST /openclaw/rename - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).post(
            "/openclaw/rename",
            data={"id": 1, "name": "x"},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="rename",
        check_admin=False,
    )


# ─── 入口 ────────────────────────────────────────────────────────────────

def main():
    global SHARED_DB_ID, ORIGINAL_NAME
    health_check()
    SHARED_DB_ID = require_shared_instance().db_id
    if SHARED_DB_ID:
        inst = find_instance_by_db_id(SHARED_DB_ID)
        ORIGINAL_NAME = (
            inst.get("name") or inst.get("Name") if inst else None
        )
        print(f">>> 复用共享实例 db_id={SHARED_DB_ID} name={ORIGINAL_NAME}")
    else:
        print(">>> 未找到共享实例，happy path 将跳过；仅跑参数校验/鉴权")
    print()

    run_tests(
        globals(),
        title="test_instance_rename.py",
        ordered=True,
    )


if __name__ == "__main__":
    main()
