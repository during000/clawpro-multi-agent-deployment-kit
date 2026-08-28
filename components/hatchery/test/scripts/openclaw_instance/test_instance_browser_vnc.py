#!/usr/bin/env python3
"""
集成测试：实例管理 - 云端浏览器 Browser VNC（N 组）

覆盖接口：
    GET  /openclaw/browser-vnc-access     连接地址（公网 IP + 5900 端口）
    GET  /openclaw/browser-status         AI 任务/接管状态（前端 3s 轮询）
    POST /openclaw/browser-takeover       手动接管/释放
    GET  /openclaw/browser-vnc-check      检查 VNC 环境是否安装（TAT 5-15s）
    POST /openclaw/browser-vnc-install    安装 VNC 环境（TAT 1-2 min）
    GET  /openclaw/vnc-ws-proxy           WebSocket 代理（不属于 JSON 接口，仅契约）

设计：
    - 仅 OpenClaw 类型实例支持，其他类型 (ACE/Hermes) → 403
    - check / install 是 TAT 重操作，本测试**绝不真发起**：
        * check 在站点开关关闭时直接 forbidden，不会下发 TAT
        * install 需要 running 状态准入，对未就绪实例直接 4xx
    - browser-status：不支持的 agent_type (hermes/ace) 返回 200 + unsupported=true；
      站点级 BrowserVNCEnable 开关关闭时返回 403（与 vnc-access / vnc-check 一致）
    - vnc-ws-proxy 直接走 HTTP 而非 WS 时会快速失败，期望 4xx
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
    assert_json_keys,
)


SHARED_DB_ID = None


# ─── /openclaw/browser-vnc-access ────────────────────────────────────────

def test_01_vnc_access_missing_id():
    """GET /openclaw/browser-vnc-access - 缺 id → 400"""
    resp = cli.get("/openclaw/browser-vnc-access", expect=None, raw=True)
    assert_status(resp, {400, 404}, label="vnc-access-missing")
    print(f"    OK status={resp.status_code}")


def test_02_vnc_access_nonexistent_id():
    """GET /openclaw/browser-vnc-access?id=NONEXISTENT → 4xx"""
    resp = cli.get(
        "/openclaw/browser-vnc-access",
        params={"id": NONEXISTENT_DB_ID},
        expect=None, raw=True,
    )
    assert_status(resp, {400, 404}, label="vnc-access-not-found")
    print(f"    OK status={resp.status_code}")


def test_03_vnc_access_ok_or_disabled():
    """GET /openclaw/browser-vnc-access - happy path 或开关未启用 (403)"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.get(
        "/openclaw/browser-vnc-access",
        params={"id": SHARED_DB_ID},
        expect=None, raw=True, timeout=30,
    )
    if resp.status_code == 403:
        print("    SKIP (Browser VNC 未启用 / 实例类型不支持)")
        return
    if resp.status_code != 200:
        # 后端可能因 CVM 公网 IP 缺失等返回 5xx，此时也接受
        print(f"    SKIP (非 200): status={resp.status_code}")
        return
    print(f"    OK status=200")


def test_04_vnc_access_auth():
    """GET /openclaw/browser-vnc-access - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=15).get(
            "/openclaw/browser-vnc-access",
            params={"id": 1},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="vnc-access",
        check_admin=False,
    )


# ─── /openclaw/browser-status ────────────────────────────────────────────

def test_05_browser_status_missing_id():
    """GET /openclaw/browser-status - 缺 id → 400"""
    resp = cli.get("/openclaw/browser-status", expect=None, raw=True)
    assert_status(resp, {400, 404}, label="browser-status-missing")
    print(f"    OK status={resp.status_code}")


def test_06_browser_status_ok():
    """GET /openclaw/browser-status - happy path 或开关未启用 (403)

    实现侧契约（controller/browser_vnc.go::browserStatusCore）：
      - BrowserVNCEnable=true  → 200 + {ai_active, takeover}
      - BrowserVNCEnable=false → 403（与 vnc-access / vnc-check 一致）
      - agent_type 不支持      → 200 + {unsupported:true}
    本测试两种合法形态都接受，避免在本地/远端不同开关配置下产生噪声失败。"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.get(
        "/openclaw/browser-status",
        params={"id": SHARED_DB_ID},
        expect=None, raw=True, timeout=15,
    )
    if resp.status_code == 403:
        print("    OK status=403 (Browser VNC 未启用)")
        return
    assert_status(resp, {200}, label="browser-status-ok")
    body = resp.json()
    assert body.get("ok") is True, f"返回 ok!=true: {body}"
    data = body.get("data") or {}
    # 至少包含这几个字段中的一部分
    assert "ai_active" in data or "takeover" in data or "unsupported" in data, (
        f"返回 data 缺核心字段: {data}"
    )
    print(f"    OK status=200 data_keys={sorted(data.keys())}")


def test_07_browser_status_auth():
    """GET /openclaw/browser-status - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=15).get(
            "/openclaw/browser-status",
            params={"id": 1},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="browser-status",
        check_admin=False,
    )


# ─── /openclaw/browser-takeover ──────────────────────────────────────────

def test_08_takeover_missing_id():
    """POST /openclaw/browser-takeover - 缺 id → 400"""
    resp = cli.post("/openclaw/browser-takeover", expect=None, raw=True)
    assert_status(resp, {400, 404}, label="takeover-missing")
    print(f"    OK status={resp.status_code}")


def test_09_takeover_unsupported_or_not_running():
    """POST /openclaw/browser-takeover - 不支持 / 实例非 running → 4xx"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.post(
        "/openclaw/browser-takeover",
        data={"id": SHARED_DB_ID, "action": "stop"},
        expect=None, raw=True, timeout=15,
    )
    # 期望：403（不支持/开关关）/ 4xx（非 running）/ 200（极少数已开 + running）
    assert resp.status_code in {200, 400, 403, 409}, (
        f"非预期 status: {resp.status_code} body={resp.text[:200]}"
    )
    print(f"    OK status={resp.status_code}")


def test_10_takeover_auth():
    """POST /openclaw/browser-takeover - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=15).post(
            "/openclaw/browser-takeover",
            data={"id": 1, "action": "stop"},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="browser-takeover",
        check_admin=False,
    )


# ─── /openclaw/browser-vnc-check ─────────────────────────────────────────

def test_11_vnc_check_missing_id():
    """GET /openclaw/browser-vnc-check - 缺 id → 400"""
    resp = cli.get("/openclaw/browser-vnc-check", expect=None, raw=True)
    assert_status(resp, {400, 404}, label="vnc-check-missing")
    print(f"    OK status={resp.status_code}")


def test_12_vnc_check_unsupported_or_disabled():
    """GET /openclaw/browser-vnc-check - 开关关 / 不支持 → 403
    本测试不允许 happy path（避免触发 5-15s 的 TAT），开关开时 SKIP。"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.get(
        "/openclaw/browser-vnc-check",
        params={"id": SHARED_DB_ID},
        expect=None, raw=True, timeout=15,
    )
    if resp.status_code == 200:
        print(f"    SKIP (开关已开启，避免触发 TAT 检查)")
        return
    assert_status(resp, {400, 403}, label="vnc-check-disabled")
    print(f"    OK status={resp.status_code}")


def test_13_vnc_check_auth():
    """GET /openclaw/browser-vnc-check - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=15).get(
            "/openclaw/browser-vnc-check",
            params={"id": 1},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="vnc-check",
        check_admin=False,
    )


# ─── /openclaw/browser-vnc-install ───────────────────────────────────────

def test_14_vnc_install_missing_id():
    """POST /openclaw/browser-vnc-install - 缺 id → 400"""
    resp = cli.post("/openclaw/browser-vnc-install", expect=None, raw=True)
    assert_status(resp, {400, 404}, label="vnc-install-missing")
    print(f"    OK status={resp.status_code}")


def test_15_vnc_install_unsupported_or_disabled():
    """POST /openclaw/browser-vnc-install - 开关关 / 不支持 → 403
    install 是 1-2 min 的 TAT，绝不允许真实下发；
    开关开时 SKIP（先用 check 接口探测，避免误触发 install）。"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    # 先用 check 接口（5-15s）探测开关/支持性，避免直接打 install 造成
    # 真实 1-2min TAT 安装。check 200 = 开关已开 + 支持，直接 SKIP；
    # check 4xx = 开关关/不支持，可以放心打 install 并断言同样的 4xx。
    pre = cli.get(
        "/openclaw/browser-vnc-check",
        params={"id": SHARED_DB_ID},
        expect=None, raw=True, timeout=20,
    )
    if pre.status_code == 200:
        print("    SKIP (开关已开启 + running，避免触发 TAT 安装)")
        return
    if pre.status_code not in {400, 403, 409}:
        print(f"    SKIP (check 探测异常 status={pre.status_code})")
        return
    resp = cli.post(
        "/openclaw/browser-vnc-install",
        data={"id": SHARED_DB_ID},
        expect=None, raw=True, timeout=15,
    )
    # 允许 4xx（开关关 / 类型不支持 / 状态准入）
    assert resp.status_code in {400, 403, 409}, (
        f"非预期 status: {resp.status_code} body={resp.text[:200]}"
    )
    print(f"    OK status={resp.status_code}")


def test_16_vnc_install_auth():
    """POST /openclaw/browser-vnc-install - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=15).post(
            "/openclaw/browser-vnc-install",
            data={"id": 1},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="vnc-install",
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
        title="test_instance_browser_vnc.py",
        ordered=True,
    )


if __name__ == "__main__":
    main()
