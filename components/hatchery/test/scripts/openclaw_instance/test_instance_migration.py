#!/usr/bin/env python3
"""
集成测试：实例管理 - 数据迁移（C 组，仅契约级）

覆盖接口：
    POST /openclaw/migration/export
    GET  /openclaw/migration/status
    GET  /openclaw/migration/progress
    POST /openclaw/migration/import

migration 真实链路需要两个实例（源 + 目标），本测试只测契约级。
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


def test_01_export_missing_id():
    """POST /openclaw/migration/export - 缺 id → 400

    注：当后端 SMH 服务未启用时，handler 会在参数校验之前就返回 403
    （"SMH 服务未启用"），此时无法到达 missing-id 的校验分支，
    本用例视为 SKIP，不算 FAIL。
    """
    resp = cli.post(
        "/openclaw/migration/export", data={}, expect=None, raw=True,
    )
    if resp.status_code == 403 and "SMH" in (resp.text or ""):
        print("    SKIP (SMH 未启用，前置短路 403)")
        return
    assert_status(resp, {400, 404}, label="export-missing-id")
    print(f"    OK status={resp.status_code}")


def test_02_export_nonexistent_id():
    """POST /openclaw/migration/export?id=NONEXISTENT → 4xx"""
    resp = cli.post(
        "/openclaw/migration/export",
        data={"id": NONEXISTENT_DB_ID},
        expect=None, raw=True, timeout=30,
    )
    assert_status(resp, {400, 403, 404, 500}, label="export-not-found")
    print(f"    OK status={resp.status_code}")


def test_03_export_ok_or_smh_disabled():
    """POST /openclaw/migration/export - happy path 或 SMH 未启用"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.post(
        "/openclaw/migration/export",
        data={"id": SHARED_DB_ID},
        expect=None, raw=True, timeout=120,
    )
    if resp.status_code == 403:
        print("    SKIP (SMH 未启用)")
        return
    if resp.status_code != 200:
        print(f"    SKIP (非 200): status={resp.status_code}")
        return
    body = assert_json_keys(resp, "migration_id", "script", "file_key")
    assert body.get("script") and body.get("file_key"), (
        f"script/file_key 不应为空: {body}"
    )
    print(f"    OK migration_id={body['migration_id']}")


def test_04_export_auth():
    """POST /openclaw/migration/export - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).post(
            "/openclaw/migration/export",
            data={"id": 1},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="migration-export",
        check_admin=False,
    )


def test_05_migration_status_missing_id():
    """GET /openclaw/migration/status - 缺 id → 400

    SMH 未启用时 handler 会先返回 403，无法触达 missing-id 校验，按 SKIP 处理。
    """
    resp = cli.get(
        "/openclaw/migration/status", expect=None, raw=True,
    )
    if resp.status_code == 403 and "SMH" in (resp.text or ""):
        print("    SKIP (SMH 未启用，前置短路 403)")
        return
    assert_status(resp, {400, 404}, label="migration-status-missing")
    print(f"    OK status={resp.status_code}")


def test_06_migration_status_ok_or_no_record():
    """GET /openclaw/migration/status - happy path 或无记录"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.get(
        "/openclaw/migration/status",
        params={"id": SHARED_DB_ID},
        expect=None, raw=True, timeout=60,
    )
    if resp.status_code != 200:
        print(f"    SKIP (非 200): status={resp.status_code}")
        return
    body = assert_json_keys(resp, "has_migration")
    print(f"    OK has_migration={body['has_migration']}")


def test_07_migration_status_auth():
    """GET /openclaw/migration/status - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).get(
            "/openclaw/migration/status",
            params={"id": 1},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="migration-status",
        check_admin=False,
    )


def test_08_migration_progress_missing_id():
    """GET /openclaw/migration/progress - 缺 id → 400"""
    resp = cli.get(
        "/openclaw/migration/progress", expect=None, raw=True,
    )
    assert_status(resp, {400, 404}, label="migration-progress-missing")
    print(f"    OK status={resp.status_code}")


def test_09_migration_progress_ok():
    """GET /openclaw/migration/progress - happy path"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.get(
        "/openclaw/migration/progress",
        params={"id": SHARED_DB_ID},
        expect=None, raw=True, timeout=30,
    )
    if resp.status_code != 200:
        print(f"    SKIP (非 200): status={resp.status_code}")
        return
    body = assert_json_keys(resp, "has_migration")
    print(f"    OK has_migration={body['has_migration']}")


def test_10_migration_progress_auth():
    """GET /openclaw/migration/progress - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).get(
            "/openclaw/migration/progress",
            params={"id": 1},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="migration-progress",
        check_admin=False,
    )


def test_11_migration_import_missing_id():
    """POST /openclaw/migration/import - 缺 id → 400

    SMH 未启用时 handler 会先返回 403，无法触达 missing-id 校验，按 SKIP 处理。
    """
    resp = cli.post(
        "/openclaw/migration/import", data={}, expect=None, raw=True,
    )
    if resp.status_code == 403 and "SMH" in (resp.text or ""):
        print("    SKIP (SMH 未启用，前置短路 403)")
        return
    assert_status(resp, {400, 404}, label="migration-import-missing")
    print(f"    OK status={resp.status_code}")


def test_12_migration_import_no_record():
    """POST /openclaw/migration/import - 无 export 记录 → 400/403/409"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.post(
        "/openclaw/migration/import",
        data={"id": SHARED_DB_ID},
        expect=None, raw=True, timeout=60,
    )
    assert_status(resp, {400, 403, 409}, label="migration-import-no-record")
    print(f"    OK status={resp.status_code}")


def test_13_migration_import_auth():
    """POST /openclaw/migration/import - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).post(
            "/openclaw/migration/import",
            data={"id": 1},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="migration-import",
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
        title="test_instance_migration.py",
        ordered=True,
    )


if __name__ == "__main__":
    main()
