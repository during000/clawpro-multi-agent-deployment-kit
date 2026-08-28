#!/usr/bin/env python3
"""
集成测试：实例管理 - 用户端 MCP（M 组）

覆盖接口：
    GET  /openclaw/mcp/available         企业可选 MCP 列表（按实例视角）
    POST /openclaw/mcp/add               添加（批量、JSON body）
    GET  /openclaw/mcp/list              当前实例已添加的 MCP
    POST /openclaw/mcp/refresh-status    刷新连接状态
    POST /openclaw/mcp/update-config     编辑配置
    POST /openclaw/mcp/delete            删除
    POST /openclaw/mcp/toggle            启用/禁用

设计：
    - GET /available, /list 走 happy path（依赖共享实例）
    - 其余写操作（add/update/delete/toggle/refresh）只测契约 + 参数校验 +
      跨用户实例归属 + 鉴权三件套，**不真发起 TAT**
    - mcp/add 检测「instance_ids=[NONEXISTENT_DB_ID]」时后端会返回 200 +
      results[*].status=skipped，不会污染共享实例
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
    assert_error_message, assert_json_keys,
)


SHARED_DB_ID = None


# ─── /openclaw/mcp/available ─────────────────────────────────────────────

def test_01_available_missing_id():
    """GET /openclaw/mcp/available - 缺 id → 400"""
    resp = cli.get("/openclaw/mcp/available", expect=None, raw=True)
    assert_status(resp, {400, 404}, label="mcp-available-missing")
    print(f"    OK status={resp.status_code}")


def test_02_available_nonexistent_id():
    """GET /openclaw/mcp/available?id=NONEXISTENT → 4xx"""
    resp = cli.get(
        "/openclaw/mcp/available",
        params={"id": NONEXISTENT_DB_ID},
        expect=None, raw=True,
    )
    assert_status(resp, {400, 404}, label="mcp-available-not-found")
    print(f"    OK status={resp.status_code}")


def test_03_available_ok():
    """GET /openclaw/mcp/available - happy path"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.get(
        "/openclaw/mcp/available",
        params={"id": SHARED_DB_ID},
        expect=None, raw=True, timeout=15,
    )
    if resp.status_code == 403:
        # checkInstanceSupportsMcp 版本门控不通过
        print(f"    SKIP (实例版本不支持 MCP, 403)")
        return
    assert_status(resp, {200}, label="mcp-available-ok")
    body = resp.json() or {}
    assert "items" in body, f"返回缺 items: {body}"
    assert "total" in body, f"返回缺 total: {body}"
    print(f"    OK total={body['total']}")


def test_04_available_auth():
    """GET /openclaw/mcp/available - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=15).get(
            "/openclaw/mcp/available",
            params={"id": 1},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="mcp-available",
        check_admin=False,
    )


# ─── /openclaw/mcp/list ──────────────────────────────────────────────────

def test_05_list_missing_id():
    """GET /openclaw/mcp/list - 缺 id → 400"""
    resp = cli.get("/openclaw/mcp/list", expect=None, raw=True)
    assert_status(resp, {400, 404}, label="mcp-list-missing")
    print(f"    OK status={resp.status_code}")


def test_06_list_ok():
    """GET /openclaw/mcp/list - happy path"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    body = cli.get(
        "/openclaw/mcp/list",
        params={"id": SHARED_DB_ID}, timeout=15,
    )
    assert "items" in body, f"返回缺 items: {body}"
    assert isinstance(body["items"], list)
    print(f"    OK count={len(body['items'])}")


def test_07_list_auth():
    """GET /openclaw/mcp/list - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=15).get(
            "/openclaw/mcp/list",
            params={"id": 1},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="mcp-list",
        check_admin=False,
    )


# ─── /openclaw/mcp/add ───────────────────────────────────────────────────

def test_08_add_invalid_body():
    """POST /openclaw/mcp/add - 非 JSON → 400"""
    resp = cli.post(
        "/openclaw/mcp/add",
        data="not-json{",
        expect=None, raw=True,
    )
    assert_status(resp, {400}, label="mcp-add-invalid-body")
    print(f"    OK status={resp.status_code}")


def test_09_add_missing_instance_ids():
    """POST /openclaw/mcp/add - 缺 instance_ids → 400"""
    resp = cli.post(
        "/openclaw/mcp/add",
        json={"service_id": "x", "config_json": "{}"},
        expect=None, raw=True,
    )
    assert_status(resp, {400}, label="mcp-add-missing-instance-ids")
    assert_error_message(resp, "instance_ids")
    print(f"    OK status={resp.status_code}")


def test_10_add_too_many_instance_ids():
    """POST /openclaw/mcp/add - instance_ids > 50 → 400"""
    resp = cli.post(
        "/openclaw/mcp/add",
        json={
            "instance_ids": list(range(1, 60)),
            "service_id": "x",
            "config_json": '{"url": "http://x"}',
        },
        expect=None, raw=True,
    )
    assert_status(resp, {400}, label="mcp-add-too-many")
    assert_error_message(resp, "50")
    print(f"    OK status={resp.status_code}")


def test_11_add_missing_service_id():
    """POST /openclaw/mcp/add - 缺 service_id → 400"""
    resp = cli.post(
        "/openclaw/mcp/add",
        json={
            "instance_ids": [SHARED_DB_ID or 1],
            "config_json": '{"url": "http://x"}',
        },
        expect=None, raw=True,
    )
    assert_status(resp, {400}, label="mcp-add-missing-service")
    assert_error_message(resp, "service_id")
    print(f"    OK status={resp.status_code}")


def test_12_add_invalid_config_json():
    """POST /openclaw/mcp/add - config_json 非合法 JSON → 400"""
    resp = cli.post(
        "/openclaw/mcp/add",
        json={
            "instance_ids": [SHARED_DB_ID or 1],
            "service_id": "x",
            "config_json": "not-json{",
        },
        expect=None, raw=True,
    )
    assert_status(resp, {400}, label="mcp-add-invalid-config")
    assert_error_message(resp, "config_json")
    print(f"    OK status={resp.status_code}")


def test_13_add_config_missing_url_or_command():
    """POST /openclaw/mcp/add - config_json 缺 url/command → 400"""
    resp = cli.post(
        "/openclaw/mcp/add",
        json={
            "instance_ids": [SHARED_DB_ID or 1],
            "service_id": "x",
            "config_json": '{"foo": "bar"}',
        },
        expect=None, raw=True,
    )
    assert_status(resp, {400}, label="mcp-add-config-no-url")
    assert_error_message(resp, "url", "command")
    print(f"    OK status={resp.status_code}")


def test_14_add_nonexistent_service():
    """POST /openclaw/mcp/add - service_id 不存在 → 404"""
    resp = cli.post(
        "/openclaw/mcp/add",
        json={
            "instance_ids": [SHARED_DB_ID or 1],
            "service_id": "no-such-mcp-service-12345",
            "config_json": '{"url": "http://x"}',
        },
        expect=None, raw=True,
    )
    assert_status(resp, {404}, label="mcp-add-not-found")
    print(f"    OK status={resp.status_code}")


def test_15_add_auth():
    """POST /openclaw/mcp/add - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=15).post(
            "/openclaw/mcp/add",
            json={
                "instance_ids": [1],
                "service_id": "x",
                "config_json": '{"url": "http://x"}',
            },
            expect=None, raw=True, extra_headers=headers,
        ),
        label="mcp-add",
        check_admin=False,
    )


# ─── /openclaw/mcp/refresh-status ────────────────────────────────────────

def test_16_refresh_invalid_body():
    """POST /openclaw/mcp/refresh-status - 非 JSON → 400"""
    resp = cli.post(
        "/openclaw/mcp/refresh-status",
        data="not-json{",
        expect=None, raw=True,
    )
    assert_status(resp, {400}, label="mcp-refresh-invalid-body")
    print(f"    OK status={resp.status_code}")


def test_17_refresh_missing_id():
    """POST /openclaw/mcp/refresh-status - 缺 id → 400"""
    resp = cli.post(
        "/openclaw/mcp/refresh-status",
        json={},
        expect=None, raw=True,
    )
    assert_status(resp, {400}, label="mcp-refresh-missing-id")
    print(f"    OK status={resp.status_code}")


def test_18_refresh_nonexistent_id():
    """POST /openclaw/mcp/refresh-status - id 不存在 → 4xx"""
    resp = cli.post(
        "/openclaw/mcp/refresh-status",
        json={"id": NONEXISTENT_DB_ID},
        expect=None, raw=True,
    )
    assert_status(resp, {400, 404}, label="mcp-refresh-not-found")
    print(f"    OK status={resp.status_code}")


def test_19_refresh_ok_empty():
    """POST /openclaw/mcp/refresh-status - happy path（实例无 MCP 时返回空 items）"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.post(
        "/openclaw/mcp/refresh-status",
        json={"id": SHARED_DB_ID, "service_ids": ["no-such-mcp"]},
        expect=None, raw=True, timeout=30,
    )
    if resp.status_code == 409:
        print("    SKIP (其他探测在进行中)")
        return
    assert_status(resp, {200}, label="mcp-refresh-ok")
    body = resp.json() or {}
    assert "items" in body
    print(f"    OK count={len(body['items'])}")


def test_20_refresh_auth():
    """POST /openclaw/mcp/refresh-status - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=15).post(
            "/openclaw/mcp/refresh-status",
            json={"id": 1},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="mcp-refresh",
        check_admin=False,
    )


# ─── /openclaw/mcp/update-config ─────────────────────────────────────────

def test_21_update_missing_service_id():
    """POST /openclaw/mcp/update-config - 缺 service_id → 400"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.post(
        "/openclaw/mcp/update-config",
        json={"id": SHARED_DB_ID, "config_json": '{"url":"http://x"}'},
        expect=None, raw=True,
    )
    # 实例如果未 RUNNING / agent 未就绪 → 409；否则 → 400 缺 service_id
    assert_status(resp, {400, 409}, label="mcp-update-missing-service")
    print(f"    OK status={resp.status_code}")


def test_22_update_nonexistent_installation():
    """POST /openclaw/mcp/update-config - service_id 未安装 → 404 / 409 / 400"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.post(
        "/openclaw/mcp/update-config",
        json={
            "id": SHARED_DB_ID,
            "service_id": "no-such-mcp-service",
            "config_json": '{"url":"http://x"}',
        },
        expect=None, raw=True, timeout=15,
    )
    # 实例若未 RUNNING/agent 未就绪先 409；否则未安装 → 404
    assert_status(resp, {404, 409, 400}, label="mcp-update-not-found")
    print(f"    OK status={resp.status_code}")


def test_23_update_auth():
    """POST /openclaw/mcp/update-config - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=15).post(
            "/openclaw/mcp/update-config",
            json={"id": 1, "service_id": "x", "config_json": "{}"},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="mcp-update-config",
        check_admin=False,
    )


# ─── /openclaw/mcp/delete ────────────────────────────────────────────────

def test_24_delete_missing_service_id():
    """POST /openclaw/mcp/delete - 缺 service_id → 400/409"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.post(
        "/openclaw/mcp/delete",
        json={"id": SHARED_DB_ID},
        expect=None, raw=True,
    )
    assert_status(resp, {400, 409}, label="mcp-delete-missing-service")
    print(f"    OK status={resp.status_code}")


def test_25_delete_nonexistent_installation():
    """POST /openclaw/mcp/delete - service_id 未安装 → 404/409/400"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.post(
        "/openclaw/mcp/delete",
        json={"id": SHARED_DB_ID, "service_id": "no-such-mcp-service"},
        expect=None, raw=True,
    )
    assert_status(resp, {404, 409, 400}, label="mcp-delete-not-found")
    print(f"    OK status={resp.status_code}")


def test_26_delete_auth():
    """POST /openclaw/mcp/delete - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=15).post(
            "/openclaw/mcp/delete",
            json={"id": 1, "service_id": "x"},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="mcp-delete",
        check_admin=False,
    )


# ─── /openclaw/mcp/toggle ────────────────────────────────────────────────

def test_27_toggle_missing_service_id():
    """POST /openclaw/mcp/toggle - 缺 service_id → 400/409"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.post(
        "/openclaw/mcp/toggle",
        json={"id": SHARED_DB_ID, "disabled": True},
        expect=None, raw=True,
    )
    assert_status(resp, {400, 409}, label="mcp-toggle-missing-service")
    print(f"    OK status={resp.status_code}")


def test_28_toggle_nonexistent_installation():
    """POST /openclaw/mcp/toggle - service_id 未安装 → 404/409/400"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.post(
        "/openclaw/mcp/toggle",
        json={
            "id": SHARED_DB_ID,
            "service_id": "no-such-mcp-service",
            "disabled": True,
        },
        expect=None, raw=True,
    )
    assert_status(resp, {404, 409, 400}, label="mcp-toggle-not-found")
    print(f"    OK status={resp.status_code}")


def test_29_toggle_auth():
    """POST /openclaw/mcp/toggle - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=15).post(
            "/openclaw/mcp/toggle",
            json={"id": 1, "service_id": "x", "disabled": True},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="mcp-toggle",
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
        title="test_instance_mcp.py",
        ordered=True,
    )


if __name__ == "__main__":
    main()
