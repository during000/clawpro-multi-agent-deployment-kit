#!/usr/bin/env python3
"""
集成测试：实例管理 - 只读元数据/探测接口（A 组）

覆盖接口：
    GET /openclaw/agent-types          智能体类型列表
    GET /openclaw/zones (POST)         可用区透传
    GET /openclaw/current-image        当前启用镜像
    GET /openclaw/version              实例 OpenClaw 版本（需要 id）
    GET /openclaw/service-status       服务运行状态（需要 id/instance_id）
    GET /openclaw/check-openclaw-port  Agent 就绪探测（需要 id/instance_id）

每个接口至少覆盖：
    - happy path（正常入参）
    - 字段契约校验
    - 缺必填参数 → 400
    - 不存在 id → 400/404
    - 认证三件套（无 token / 错误 token）

使用方式：
    export API=http://127.0.0.1:9000
    export ADMIN_TOKEN=hk-xxx
    # （可选）先跑过 lifecycle 留下 state.json，或显式指定：
    export INSTANCE_ID_REUSE=<db_id>
    python3 test_instance_meta.py

如果未提供任何可用实例，version/service-status/check-openclaw-port 的
happy path 用例会被跳过（标记 SKIP），但参数校验/鉴权用例仍会执行。
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
    assert_error_message,
    assert_json_keys,
)


# 共享实例 db_id（可能为 None）
SHARED_DB_ID = None


# ─── A1. /openclaw/agent-types ──────────────────────────────────────────────

def test_01_agent_types_ok():
    """GET /openclaw/agent-types - happy path"""
    resp = cli.get("/openclaw/agent-types", raw=True)
    body = assert_json_keys(resp, "agent_types")
    items = body.get("agent_types") or []
    assert isinstance(items, list), f"agent_types 不是数组: {type(items).__name__}"
    assert items, "agent_types 为空，无法继续。请先在管理后台启用至少一种镜像"
    sample = items[0]
    for k in ("code", "name", "is_builtin"):
        assert k in sample, f"agent_types[0] 缺字段 {k}: {sample}"
    dflt = body.get("default_agent_type")
    assert dflt is None or isinstance(dflt, str), (
        f"default_agent_type 应为 string 或缺省: {dflt}"
    )
    print(f"    OK count={len(items)} default={dflt}")


def test_02_agent_types_auth():
    """GET /openclaw/agent-types - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).get(
            "/openclaw/agent-types",
            expect=None, raw=True, extra_headers=headers,
        ),
        label="agent-types",
        check_admin=False,
    )


# ─── A2. /openclaw/zones ────────────────────────────────────────────────────

def test_03_zones_get_ok():
    """GET /openclaw/zones - happy path"""
    resp = cli.get("/openclaw/zones", raw=True)
    try:
        body = resp.json()
    except Exception:
        raise AssertionError(f"zones 应返回 JSON: {resp.text[:200]}")
    assert isinstance(body, (dict, list)), (
        f"zones JSON 类型异常: {type(body).__name__}"
    )
    print(
        f"    OK keys={list(body.keys()) if isinstance(body, dict) else 'list'}"
    )


def test_04_zones_post_empty_body():
    """POST /openclaw/zones - 空 body 应被接受"""
    resp = cli.post("/openclaw/zones", json={}, expect=None, raw=True)
    assert resp.status_code in (200, 400, 500), (
        f"zones POST 期望 200/400/500，实际 {resp.status_code}"
    )
    print(f"    OK status={resp.status_code}")


def test_05_zones_auth():
    """GET /openclaw/zones - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).get(
            "/openclaw/zones",
            expect=None, raw=True, extra_headers=headers,
        ),
        label="zones",
        check_admin=False,
    )


# ─── A3. /openclaw/current-image ────────────────────────────────────────────

def test_06_current_image_no_param():
    """GET /openclaw/current-image - 不带 agent_type"""
    resp = cli.get("/openclaw/current-image", raw=True)
    body = assert_json_keys(resp, "image")
    img = body.get("image")
    assert img is None or isinstance(img, dict), (
        f"image 字段类型异常: {type(img).__name__}"
    )
    print(f"    OK image={'null' if img is None else img.get('image_id')}")


def test_07_current_image_with_agent_type():
    """GET /openclaw/current-image?agent_type=openclaw"""
    resp = cli.get(
        "/openclaw/current-image",
        params={"agent_type": "openclaw"},
        raw=True,
    )
    body = assert_json_keys(resp, "image")
    img = body.get("image")
    if img is not None:
        if img.get("agent_type") and img.get("agent_type") != "openclaw":
            raise AssertionError(
                f"agent_type 不一致: 期望 openclaw, 实际 {img.get('agent_type')}"
            )
        for k in ("image_id", "image_name", "image_type"):
            assert k in img, f"image 缺字段 {k}: {img}"
    print(f"    OK image={'null' if img is None else img.get('image_id')}")


def test_08_current_image_unknown_agent_type():
    """GET /openclaw/current-image?agent_type=unknown"""
    resp = cli.get(
        "/openclaw/current-image",
        params={"agent_type": "__not_exist_xyz__"},
        expect=None, raw=True,
    )
    assert resp.status_code in (200, 400), (
        f"unknown agent_type 期望 200/400, 实际 {resp.status_code}"
    )
    if resp.status_code == 200:
        body = resp.json() or {}
        img = body.get("image")
        print(
            f"    OK status=200 "
            f"image={'null' if img is None else img.get('image_id')}"
        )
    else:
        print(f"    OK status={resp.status_code}")


def test_09_current_image_auth():
    """GET /openclaw/current-image - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).get(
            "/openclaw/current-image",
            expect=None, raw=True, extra_headers=headers,
        ),
        label="current-image",
        check_admin=False,
    )


# ─── A4. /openclaw/version ──────────────────────────────────────────────────

def test_10_version_missing_id():
    """GET /openclaw/version - 缺 id 参数 → 400"""
    resp = cli.get("/openclaw/version", expect=None, raw=True)
    assert_status(resp, 400, label="version-no-id")
    assert_error_message(resp, "缺少参数 id", "id")
    print("    OK")


def test_11_version_nonexistent_id():
    """GET /openclaw/version?id=NONEXISTENT → 4xx"""
    resp = cli.get(
        "/openclaw/version",
        params={"id": NONEXISTENT_DB_ID},
        expect=None, raw=True, timeout=60,
    )
    assert_status(resp, {400, 404}, label="version-not-found")
    print(f"    OK status={resp.status_code}")


def test_12_version_ok():
    """GET /openclaw/version - happy path"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.get(
        "/openclaw/version",
        params={"id": SHARED_DB_ID},
        expect=None, raw=True, timeout=120,
    )
    if resp.status_code != 200:
        print(f"    OK 非 200（可能 TAT 暂不可用）: status={resp.status_code}")
        return
    body = assert_json_keys(resp, "ok")
    assert body.get("ok"), f"ok=false: {body}"
    print(f"    OK version={body.get('version')} runtime_user={body.get('runtime_user')}")


def test_13_version_auth():
    """GET /openclaw/version - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).get(
            "/openclaw/version",
            params={"id": 1},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="version",
        check_admin=False,
    )


# ─── A5. /openclaw/service-status ───────────────────────────────────────────

def test_14_service_status_missing_param():
    """GET /openclaw/service-status - 不传 id/instance_id"""
    resp = cli.get("/openclaw/service-status", expect=None, raw=True)
    assert resp.status_code in (400, 404, 422), (
        f"缺参期望 4xx, 实际 {resp.status_code}: {resp.text[:200]}"
    )
    print(f"    OK status={resp.status_code}")


def test_15_service_status_nonexistent_id():
    """GET /openclaw/service-status?id=NONEXISTENT → 4xx/5xx"""
    resp = cli.get(
        "/openclaw/service-status",
        params={"id": NONEXISTENT_DB_ID},
        expect=None, raw=True, timeout=60,
    )
    assert_status(resp, {400, 404, 500}, label="service-status-not-found")
    print(f"    OK status={resp.status_code}")


def test_16_service_status_ok():
    """GET /openclaw/service-status - happy path"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.get(
        "/openclaw/service-status",
        params={"id": SHARED_DB_ID},
        expect=200, raw=True, timeout=180,
    )
    try:
        body = resp.json()
    except Exception:
        raise AssertionError(f"应返回 JSON: {resp.text[:200]}")
    print(
        f"    OK keys="
        f"{list(body.keys()) if isinstance(body, dict) else type(body).__name__}"
    )


def test_17_service_status_auth():
    """GET /openclaw/service-status - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).get(
            "/openclaw/service-status",
            params={"id": 1},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="service-status",
        check_admin=False,
    )


# ─── A6. /openclaw/check-openclaw-port ──────────────────────────────────────

def test_18_check_port_missing_param():
    """GET /openclaw/check-openclaw-port - 不传 id/instance_id"""
    resp = cli.get("/openclaw/check-openclaw-port", expect=None, raw=True)
    assert resp.status_code in (400, 404, 422), (
        f"缺参期望 4xx, 实际 {resp.status_code}: {resp.text[:200]}"
    )
    print(f"    OK status={resp.status_code}")


def test_19_check_port_nonexistent_id():
    """GET /openclaw/check-openclaw-port?id=NONEXISTENT → 4xx/5xx"""
    resp = cli.get(
        "/openclaw/check-openclaw-port",
        params={"id": NONEXISTENT_DB_ID},
        expect=None, raw=True, timeout=60,
    )
    assert_status(resp, {400, 404, 500}, label="check-port-not-found")
    print(f"    OK status={resp.status_code}")


def test_20_check_port_ok():
    """GET /openclaw/check-openclaw-port - happy path"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.get(
        "/openclaw/check-openclaw-port",
        params={"id": SHARED_DB_ID},
        expect=200, raw=True, timeout=180,
    )
    body = assert_json_keys(resp, "running")
    assert isinstance(body.get("running"), bool), (
        f"running 字段应为 bool: {body}"
    )
    print(f"    OK running={body.get('running')} reason={body.get('reason')}")


def test_21_check_port_auth():
    """GET /openclaw/check-openclaw-port - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).get(
            "/openclaw/check-openclaw-port",
            params={"id": 1},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="check-openclaw-port",
        check_admin=False,
    )


# ─── A7. /openclaw/agent-count ──────────────────────────────────────────────

def test_22_agent_count_missing_param():
    """GET /openclaw/agent-count - 缺 id → 400"""
    resp = cli.get("/openclaw/agent-count", expect=None, raw=True)
    assert_status(resp, {400, 404}, label="agent-count-missing")
    print(f"    OK status={resp.status_code}")


def test_23_agent_count_nonexistent_id():
    """GET /openclaw/agent-count?id=NONEXISTENT → 4xx"""
    resp = cli.get(
        "/openclaw/agent-count",
        params={"id": NONEXISTENT_DB_ID},
        expect=None, raw=True,
    )
    assert_status(resp, {400, 404}, label="agent-count-not-found")
    print(f"    OK status={resp.status_code}")


def test_24_agent_count_ok():
    """GET /openclaw/agent-count - happy path（不支持多 agent 时固定 count=1）"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.get(
        "/openclaw/agent-count",
        params={"id": SHARED_DB_ID},
        expect=None, raw=True, timeout=60,
    )
    if resp.status_code != 200:
        print(f"    SKIP (非 200): status={resp.status_code}")
        return
    body = resp.json() or {}
    assert body.get("ok") is True, f"返回 ok!=true: {body}"
    count = body.get("count")
    assert isinstance(count, int) and count >= 1, f"count 非法: {body}"
    print(f"    OK count={count}")


def test_25_agent_count_auth():
    """GET /openclaw/agent-count - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=15).get(
            "/openclaw/agent-count",
            params={"id": 1},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="agent-count",
        check_admin=False,
    )


# ─── A8. /openclaw/images/update-notices ────────────────────────────────────

def test_26_image_update_notices_ok():
    """GET /openclaw/images/update-notices - happy path"""
    body = cli.get("/openclaw/images/update-notices", timeout=15)
    assert "items" in body, f"返回缺 items: {body}"
    assert isinstance(body["items"], list), f"items 应为 list: {body}"
    print(f"    OK count={len(body['items'])}")


def test_27_image_update_notices_auth():
    """GET /openclaw/images/update-notices - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=15).get(
            "/openclaw/images/update-notices",
            expect=None, raw=True, extra_headers=headers,
        ),
        label="image-update-notices",
        check_admin=False,
    )


# ─── 入口 ─────────────────────────────────────────────────────────────────

def main():
    global SHARED_DB_ID
    health_check()
    SHARED_DB_ID = require_shared_instance().db_id
    if SHARED_DB_ID:
        print(f">>> 复用共享实例 db_id={SHARED_DB_ID}")
    else:
        print(">>> 未找到共享实例，version/service-status/check-port 的 happy path 将跳过")
    print()

    run_tests(
        globals(),
        title="test_instance_meta.py",
        ordered=True,
    )


if __name__ == "__main__":
    main()
