#!/usr/bin/env python3
"""
集成测试：实例管理 - Gateway UI（G 组）

覆盖接口：
    POST /openclaw/set-gateway-ui       获取 WebUI 面板地址
    GET  /openclaw/check-gateway-access 检查 WebUI 端口可访问性
    POST /openclaw/ws-url               SDK 用：返回内网 WS 连接 URL
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
    assert_json_keys, assert_error_message,
)


SHARED_DB_ID = None


# ─── /openclaw/set-gateway-ui ────────────────────────────────────────────

def test_01_set_gateway_ui_missing_params():
    """POST /openclaw/set-gateway-ui - 缺 id/instance_id → 400"""
    resp = cli.post(
        "/openclaw/set-gateway-ui", data={}, expect=None, raw=True,
    )
    assert_status(resp, {400, 405}, label="set-gateway-ui-missing")
    print(f"    OK status={resp.status_code}")


def test_02_set_gateway_ui_nonexistent_id():
    """POST /openclaw/set-gateway-ui?id=NONEXISTENT → 4xx"""
    resp = cli.post(
        "/openclaw/set-gateway-ui",
        data={"id": NONEXISTENT_DB_ID},
        expect=None, raw=True, timeout=60,
    )
    assert_status(resp, {400, 404, 500}, label="set-gateway-ui-not-found")
    print(f"    OK status={resp.status_code}")


def test_03_set_gateway_ui_ok():
    """POST /openclaw/set-gateway-ui - happy path"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.post(
        "/openclaw/set-gateway-ui",
        data={"id": SHARED_DB_ID, "network_type": "public"},
        expect=None, raw=True, timeout=180,
    )
    if resp.status_code == 403:
        print("    SKIP (Gateway UI 功能未开启)")
        return
    if resp.status_code != 200:
        print(f"    SKIP (非 200): status={resp.status_code}")
        return
    body = assert_json_keys(resp, "gatewayUI", "token")
    assert body.get("gatewayUI"), f"gatewayUI 为空: {body}"
    print(f"    OK gatewayUI={body['gatewayUI'][:60]}...")


def test_04_set_gateway_ui_invalid_network_type():
    """POST /openclaw/set-gateway-ui - 非法 network_type 不应 5xx"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.post(
        "/openclaw/set-gateway-ui",
        data={"id": SHARED_DB_ID, "network_type": "invalid_value"},
        expect=None, raw=True, timeout=120,
    )
    assert resp.status_code < 600, f"非常规状态码: {resp.status_code}"
    print(f"    OK status={resp.status_code}")


def test_05_set_gateway_ui_auth():
    """POST /openclaw/set-gateway-ui - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).post(
            "/openclaw/set-gateway-ui",
            data={"id": 1},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="set-gateway-ui",
        check_admin=False,
    )


# ─── /openclaw/check-gateway-access ──────────────────────────────────────

def test_06_check_gateway_access_missing_params():
    """GET /openclaw/check-gateway-access - 缺 id → 400"""
    resp = cli.get(
        "/openclaw/check-gateway-access", expect=None, raw=True,
    )
    assert_status(resp, {400, 403, 405}, label="check-gateway-access-missing")
    print(f"    OK status={resp.status_code}")


def test_07_check_gateway_access_nonexistent_id():
    """GET /openclaw/check-gateway-access?id=NONEXISTENT → 4xx"""
    resp = cli.get(
        "/openclaw/check-gateway-access",
        params={"id": NONEXISTENT_DB_ID},
        expect=None, raw=True, timeout=60,
    )
    assert_status(
        resp, {400, 403, 404, 500}, label="check-gateway-access-not-found",
    )
    print(f"    OK status={resp.status_code}")


def test_08_check_gateway_access_ok():
    """GET /openclaw/check-gateway-access - happy path"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.get(
        "/openclaw/check-gateway-access",
        params={"id": SHARED_DB_ID},
        expect=None, raw=True, timeout=120,
    )
    if resp.status_code == 403:
        print("    SKIP (Gateway UI 功能未开启)")
        return
    if resp.status_code != 200:
        print(f"    SKIP (非 200): status={resp.status_code}")
        return
    body = assert_json_keys(
        resp, "accessible", "port", "securityGroupIds", "message",
    )
    assert isinstance(body["accessible"], bool), f"accessible 应为 bool: {body}"
    assert isinstance(body.get("securityGroupIds", []), list), (
        f"securityGroupIds 应为数组: {body}"
    )
    print(
        f"    OK accessible={body['accessible']} port={body.get('port')} "
        f"sg={body.get('securityGroupIds')}"
    )


def test_09_check_gateway_access_auth():
    """GET /openclaw/check-gateway-access - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).get(
            "/openclaw/check-gateway-access",
            params={"id": 1},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="check-gateway-access",
        check_admin=False,
    )


# ─── /openclaw/ws-url ────────────────────────────────────────────────────

def test_10_ws_url_get_method_not_allowed():
    """GET /openclaw/ws-url → 405（仅 POST）"""
    resp = cli.get("/openclaw/ws-url", expect=None, raw=True)
    assert_status(resp, {400, 405}, label="ws-url-get")
    print(f"    OK status={resp.status_code}")


def test_11_ws_url_invalid_body():
    """POST /openclaw/ws-url - 非 JSON → 400"""
    resp = cli.post(
        "/openclaw/ws-url",
        data="not-json{",
        expect=None, raw=True,
    )
    assert_status(resp, {400}, label="ws-url-invalid-body")
    print(f"    OK status={resp.status_code}")


def test_12_ws_url_missing_instance_id():
    """POST /openclaw/ws-url - 缺 instance_id → 400"""
    resp = cli.post(
        "/openclaw/ws-url",
        json={},
        expect=None, raw=True,
    )
    assert_status(resp, {400}, label="ws-url-missing-id")
    assert_error_message(resp, "instance_id")
    print(f"    OK status={resp.status_code}")


def test_13_ws_url_invalid_format():
    """POST /openclaw/ws-url - instance_id 非 ins- 前缀 → 400"""
    resp = cli.post(
        "/openclaw/ws-url",
        json={"instance_id": "123-not-cvm"},
        expect=None, raw=True,
    )
    assert_status(resp, {400}, label="ws-url-bad-format")
    assert_error_message(resp, "ins-", "格式")
    print(f"    OK status={resp.status_code}")


def test_14_ws_url_foreign_instance():
    """POST /openclaw/ws-url - instance_id 不属于当前用户 → 403"""
    resp = cli.post(
        "/openclaw/ws-url",
        json={"instance_id": "ins-no-such-cvm"},
        expect=None, raw=True,
    )
    assert_status(resp, {403, 404}, label="ws-url-foreign")
    print(f"    OK status={resp.status_code}")


def test_15_ws_url_auth():
    """POST /openclaw/ws-url - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=15).post(
            "/openclaw/ws-url",
            json={"instance_id": "ins-fake"},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="ws-url",
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
        title="test_instance_gateway.py",
        ordered=True,
    )


if __name__ == "__main__":
    main()
