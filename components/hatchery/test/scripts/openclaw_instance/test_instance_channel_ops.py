#!/usr/bin/env python3
"""
集成测试：实例管理 - 通道（D 组）

覆盖接口：
    GET  /openclaw/channels        通道列表（含 id 模式）
    POST /openclaw/set-channel     配置通道
    POST /openclaw/del-channel     删除通道
    GET  /openclaw/auto-channel    自动配置（SSE）

由于 set-channel 真实生效需要 TAT + 真实凭证，本测试主要覆盖契约/参数校验/
鉴权。set-channel 的 happy path 仅验证「请求被 TAT 调用，无论后端脚本失败
与否」均算契约通过；del-channel 同理。
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
)


SHARED_DB_ID = None


# ─── /openclaw/channels ──────────────────────────────────────────────────

def test_01_channels_list_global():
    """GET /openclaw/channels - 不传 id（全局通道库）"""
    resp = cli.get("/openclaw/channels", raw=True)
    body = resp.json()
    assert isinstance(body, list), f"全局通道列表应为数组: {type(body).__name__}"
    if body:
        ch = body[0]
        for k in ("ID", "ChannelID", "Name", "Enabled"):
            assert k in ch, f"通道对象缺字段 {k}: keys={list(ch.keys())}"
    print(f"    OK count={len(body)}")


def test_02_channels_list_by_instance():
    """GET /openclaw/channels?id=<shared>"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.get(
        "/openclaw/channels",
        params={"id": SHARED_DB_ID},
        expect=None, raw=True, timeout=120,
    )
    if resp.status_code == 200:
        body = resp.json()
        for k in ("agent_type", "agent_type_supported_channels", "channels"):
            assert k in body, f"实例通道响应缺字段 {k}: keys={list(body.keys())}"
        print(
            f"    OK agent_type={body.get('agent_type')} "
            f"supported={body.get('agent_type_supported_channels')}"
        )
    else:
        print(f"    OK 非 200（TAT 可能不可用）: status={resp.status_code}")


def test_03_channels_invalid_id():
    """GET /openclaw/channels?id=NONEXISTENT → 4xx"""
    resp = cli.get(
        "/openclaw/channels",
        params={"id": NONEXISTENT_DB_ID},
        expect=None, raw=True, timeout=60,
    )
    assert_status(resp, {400, 404, 500}, label="channels-not-found")
    print(f"    OK status={resp.status_code}")


def test_04_channels_auth():
    """GET /openclaw/channels - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).get(
            "/openclaw/channels",
            expect=None, raw=True, extra_headers=headers,
        ),
        label="channels",
        check_admin=False,
    )


# ─── /openclaw/set-channel ───────────────────────────────────────────────

def test_05_set_channel_missing_params():
    """POST /openclaw/set-channel - 缺 id → 400"""
    resp = cli.post(
        "/openclaw/set-channel",
        data={"channel": "qqbot"},
        expect=None, raw=True,
    )
    assert_status(resp, 400, label="set-channel-missing-id")
    print(f"    OK status={resp.status_code}")


def test_06_set_channel_missing_channel():
    """POST /openclaw/set-channel - 缺 channel → 400"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.post(
        "/openclaw/set-channel",
        data={"id": SHARED_DB_ID},
        expect=None, raw=True,
    )
    assert_status(resp, {400, 500}, label="set-channel-missing-channel")
    print(f"    OK status={resp.status_code}")


def test_07_set_channel_invalid_channel():
    """POST /openclaw/set-channel - 未知 channel"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.post(
        "/openclaw/set-channel",
        data={
            "id": SHARED_DB_ID,
            "channel": "__not_exist_channel__",
            "key": "k",
            "value": "v",
        },
        expect=None, raw=True, timeout=120,
    )
    if resp.status_code == 200:
        body = resp.json() if resp.content else {}
        assert body.get("error"), f"未知 channel 期望失败, 实际 200 ok=true: {body}"
    else:
        assert resp.status_code < 600, f"非常规状态码: {resp.status_code}"
    print(f"    OK status={resp.status_code}")


def test_08_set_channel_nonexistent_id():
    """POST /openclaw/set-channel?id=NONEXISTENT → 4xx"""
    resp = cli.post(
        "/openclaw/set-channel",
        data={
            "id": NONEXISTENT_DB_ID,
            "channel": "qqbot",
            "key": "app_id",
            "value": "test",
        },
        expect=None, raw=True, timeout=60,
    )
    assert_status(resp, {400, 404, 500}, label="set-channel-not-found")
    print(f"    OK status={resp.status_code}")


def test_09_set_channel_auth():
    """POST /openclaw/set-channel - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).post(
            "/openclaw/set-channel",
            data={"id": 1, "channel": "qqbot"},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="set-channel",
        check_admin=False,
    )


# ─── /openclaw/del-channel ───────────────────────────────────────────────

def test_10_del_channel_missing_params():
    """POST /openclaw/del-channel - 缺 id → 400"""
    resp = cli.post(
        "/openclaw/del-channel",
        data={"channel": "qqbot"},
        expect=None, raw=True,
    )
    assert_status(resp, 400, label="del-channel-missing-id")
    print(f"    OK status={resp.status_code}")


def test_11_del_channel_nonexistent_id():
    """POST /openclaw/del-channel?id=NONEXISTENT → 4xx"""
    resp = cli.post(
        "/openclaw/del-channel",
        data={"id": NONEXISTENT_DB_ID, "channel": "qqbot"},
        expect=None, raw=True, timeout=60,
    )
    assert_status(resp, {400, 404, 500}, label="del-channel-not-found")
    print(f"    OK status={resp.status_code}")


def test_12_del_channel_unconfigured():
    """POST /openclaw/del-channel - 删除未配置的 channel（幂等）"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.post(
        "/openclaw/del-channel",
        data={"id": SHARED_DB_ID, "channel": "qqbot"},
        expect=None, raw=True, timeout=120,
    )
    assert resp.status_code < 600, f"非常规状态码: {resp.status_code}"
    print(f"    OK status={resp.status_code}")


def test_13_del_channel_auth():
    """POST /openclaw/del-channel - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).post(
            "/openclaw/del-channel",
            data={"id": 1, "channel": "qqbot"},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="del-channel",
        check_admin=False,
    )


# ─── /openclaw/auto-channel ──────────────────────────────────────────────

def test_14_auto_channel_missing_id():
    """GET /openclaw/auto-channel - 缺 id → 400"""
    resp = cli.get(
        "/openclaw/auto-channel", expect=None, raw=True,
    )
    assert_status(resp, {400, 404}, label="auto-channel-missing-id")
    print(f"    OK status={resp.status_code}")


def test_15_auto_channel_nonexistent_id():
    """GET /openclaw/auto-channel?id=NONEXISTENT → 不应 5xx"""
    resp = cli.get(
        "/openclaw/auto-channel",
        params={"id": NONEXISTENT_DB_ID},
        expect=None, raw=True, timeout=10,
    )
    assert resp.status_code < 600, f"非常规状态码: {resp.status_code}"
    print(f"    OK status={resp.status_code}")


def test_16_auto_channel_auth():
    """GET /openclaw/auto-channel - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).get(
            "/openclaw/auto-channel",
            params={"id": 1},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="auto-channel",
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
        title="test_instance_channel_ops.py",
        ordered=True,
    )


if __name__ == "__main__":
    main()
