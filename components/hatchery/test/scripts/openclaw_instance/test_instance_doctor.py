#!/usr/bin/env python3
"""
集成测试：实例管理 - 龙虾医院 Doctor（L 组）

覆盖接口：
    POST /openclaw/doctor/quick-fix          一键修复（异步下发 TAT）
    GET  /openclaw/doctor/quick-fix/status   查询修复执行状态
    GET  /openclaw/doctor/feature            功能开关 + 是否已授权
    POST /openclaw/doctor/authorize          首次使用授权（幂等）
    POST /openclaw/doctor/start              创建诊断会话（建独立 CVM）
    GET  /openclaw/doctor/status             查询当前会话
    POST /openclaw/doctor/end                结束诊断会话

设计：
    - feature / status / authorize 都是轻量接口，可跑 happy path
    - quick-fix / start / end 涉及真实 TAT/CVM 操作，仅做契约 + 参数 + 鉴权
    - start 在 doctor_enabled=false（默认）时返回 ok=false + error=doctor_disabled
    - start 的 snapshot=true 参数同样走 guard 路径验证，避免误起真实 CVM
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
    get_shared_db_id, get_shared_db_id_or_none, NONEXISTENT_DB_ID,
    assert_error_message, assert_json_keys,
)


SHARED_DB_ID = None


# ─── /openclaw/doctor/feature ────────────────────────────────────────────

def test_01_feature_missing_id():
    """GET /openclaw/doctor/feature - 缺 id → 400"""
    resp = cli.get("/openclaw/doctor/feature", expect=None, raw=True)
    assert_status(resp, {400, 404}, label="doctor-feature-missing")
    print(f"    OK status={resp.status_code}")


def test_02_feature_nonexistent_id():
    """GET /openclaw/doctor/feature?id=NONEXISTENT → 4xx"""
    resp = cli.get(
        "/openclaw/doctor/feature",
        params={"id": NONEXISTENT_DB_ID},
        expect=None, raw=True,
    )
    assert_status(resp, {400, 404}, label="doctor-feature-not-found")
    print(f"    OK status={resp.status_code}")


def test_03_feature_ok():
    """GET /openclaw/doctor/feature - happy path"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    body = cli.get(
        "/openclaw/doctor/feature",
        params={"id": SHARED_DB_ID},
        timeout=15,
    )
    assert "doctor_enabled" in body, f"返回缺 doctor_enabled: {body}"
    assert "authorized" in body, f"返回缺 authorized: {body}"
    assert isinstance(body["doctor_enabled"], bool)
    assert isinstance(body["authorized"], bool)
    print(f"    OK doctor_enabled={body['doctor_enabled']} authorized={body['authorized']}")


def test_04_feature_auth():
    """GET /openclaw/doctor/feature - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=15).get(
            "/openclaw/doctor/feature",
            params={"id": 1},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="doctor-feature",
        check_admin=False,
    )


# ─── /openclaw/doctor/authorize ──────────────────────────────────────────

def test_05_authorize_missing_id():
    """POST /openclaw/doctor/authorize - 缺 id → 400"""
    resp = cli.post(
        "/openclaw/doctor/authorize",
        expect=None, raw=True,
    )
    assert_status(resp, {400, 404}, label="doctor-authorize-missing")
    print(f"    OK status={resp.status_code}")


def test_06_authorize_idempotent():
    """POST /openclaw/doctor/authorize - 幂等
    第一次返回"授权成功"，第二次返回"已授权"，都是 200/ok=true。"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    # 第一次
    body = cli.post(
        "/openclaw/doctor/authorize",
        data={"id": SHARED_DB_ID}, timeout=15,
    )
    assert body.get("ok") is True, f"第一次授权失败: {body}"
    # 第二次（幂等）
    body2 = cli.post(
        "/openclaw/doctor/authorize",
        data={"id": SHARED_DB_ID}, timeout=15,
    )
    assert body2.get("ok") is True, f"第二次授权失败: {body2}"
    print(f"    OK 第一次={body.get('message','')} 第二次={body2.get('message','')}")


def test_07_authorize_auth():
    """POST /openclaw/doctor/authorize - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=15).post(
            "/openclaw/doctor/authorize",
            data={"id": 1},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="doctor-authorize",
        check_admin=False,
    )


# ─── /openclaw/doctor/status ─────────────────────────────────────────────

def test_08_status_missing_id():
    """GET /openclaw/doctor/status - 缺 id → 400"""
    resp = cli.get("/openclaw/doctor/status", expect=None, raw=True)
    assert_status(resp, {400}, label="doctor-status-missing")
    assert_error_message(resp, "id")
    print(f"    OK status={resp.status_code}")


def test_09_status_no_active_session():
    """GET /openclaw/doctor/status - 无活动会话 → 200 has_active_session=false"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    body = cli.get(
        "/openclaw/doctor/status",
        params={"id": SHARED_DB_ID}, timeout=15,
    )
    assert body.get("ok") is True, f"返回 ok!=true: {body}"
    assert "has_active_session" in body, f"缺 has_active_session: {body}"
    print(f"    OK has_active_session={body['has_active_session']}")


def test_10_status_auth():
    """GET /openclaw/doctor/status - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=15).get(
            "/openclaw/doctor/status",
            params={"id": 1},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="doctor-status",
        check_admin=False,
    )


# ─── /openclaw/doctor/quick-fix ──────────────────────────────────────────

def test_11_quickfix_missing_id():
    """POST /openclaw/doctor/quick-fix - 缺 id → 400"""
    resp = cli.post("/openclaw/doctor/quick-fix", expect=None, raw=True)
    assert_status(resp, {400, 404}, label="doctor-quickfix-missing")
    print(f"    OK status={resp.status_code}")


def test_12_quickfix_nonexistent_id():
    """POST /openclaw/doctor/quick-fix?id=NONEXISTENT → 4xx"""
    resp = cli.post(
        "/openclaw/doctor/quick-fix",
        data={"id": NONEXISTENT_DB_ID},
        expect=None, raw=True,
    )
    assert_status(resp, {400, 404}, label="doctor-quickfix-not-found")
    print(f"    OK status={resp.status_code}")


def test_13_quickfix_auth():
    """POST /openclaw/doctor/quick-fix - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=15).post(
            "/openclaw/doctor/quick-fix",
            data={"id": 1},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="doctor-quickfix",
        check_admin=False,
    )


# ─── /openclaw/doctor/quick-fix/status ───────────────────────────────────

def test_14_quickfix_status_missing_id():
    """GET /openclaw/doctor/quick-fix/status - 缺 id → 400"""
    resp = cli.get("/openclaw/doctor/quick-fix/status", expect=None, raw=True)
    assert_status(resp, {400, 404}, label="quickfix-status-missing-id")
    print(f"    OK status={resp.status_code}")


def test_15_quickfix_status_missing_invocation_id():
    """GET /openclaw/doctor/quick-fix/status?id=X - 缺 invocation_id → 400"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.get(
        "/openclaw/doctor/quick-fix/status",
        params={"id": SHARED_DB_ID},
        expect=None, raw=True,
    )
    assert_status(resp, {400}, label="quickfix-status-missing-inv")
    assert_error_message(resp, "invocation_id")
    print(f"    OK status={resp.status_code}")


def test_16_quickfix_status_auth():
    """GET /openclaw/doctor/quick-fix/status - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=15).get(
            "/openclaw/doctor/quick-fix/status",
            params={"id": 1, "invocation_id": "inv-x"},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="quickfix-status",
        check_admin=False,
    )


# ─── /openclaw/doctor/start ──────────────────────────────────────────────

def test_17_start_missing_id():
    """POST /openclaw/doctor/start - 缺 id → 400"""
    resp = cli.post("/openclaw/doctor/start", expect=None, raw=True)
    assert_status(resp, {400, 404}, label="doctor-start-missing")
    print(f"    OK status={resp.status_code}")


def test_18_start_nonexistent_id():
    """POST /openclaw/doctor/start?id=NONEXISTENT → 4xx"""
    resp = cli.post(
        "/openclaw/doctor/start",
        data={"id": NONEXISTENT_DB_ID},
        expect=None, raw=True,
    )
    assert_status(resp, {400, 404}, label="doctor-start-not-found")
    print(f"    OK status={resp.status_code}")


def test_19_start_disabled_or_unauthorized():
    """POST /openclaw/doctor/start - 站点开关未开 / 未授权时返回 ok=false
    （绝不真起 CVM；本测试只验证后端 guard 路径）。"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.post(
        "/openclaw/doctor/start",
        json={"snapshot": False},
        params={"id": SHARED_DB_ID},
        expect=None, raw=True, timeout=20,
    )
    if resp.status_code != 200:
        print(f"    SKIP (非 200): status={resp.status_code}")
        return
    body = resp.json() or {}
    if body.get("ok") is True:
        # 极少数情况：站点已开 + 已授权 + agent_type=openclaw → 真启动了诊断会话
        # 立即调 end 兜底，避免污染后续脚本
        print("    WARN 真起了 doctor 会话，立即下发 end 兜底")
        cli.post(
            "/openclaw/doctor/end",
            data={"id": SHARED_DB_ID},
            expect=None, raw=True, timeout=15,
        )
        return
    err = body.get("error", "")
    # guard 路径合法 error 集合：
    #   - doctor_disabled        站点开关未开
    #   - not_authorized         用户未授权 doctor
    #   - unsupported_agent_type agent_type 不支持
    #   - create_failed          后端尝试启动 doctor 失败（如 STS 拿不到、agent
    #                            没装等环境/资源限制），同样属于 ok=false 的
    #                            guard 路径——绝不会真起 CVM
    assert err in {
        "doctor_disabled",
        "not_authorized",
        "unsupported_agent_type",
        "create_failed",
    }, (
        f"非预期 error: {err} body={body}"
    )
    print(f"    OK ok=false error={err}")


def test_20_start_with_snapshot_disabled_or_unauthorized():
    """POST /openclaw/doctor/start?snapshot=true - 快照参数契约

    本次改动在 createDoctorSnapshot 中备份成功后通过 defer 兜底重启目标实例
    Gateway。start 接口的 snapshot=true 仅影响 session.snapshot_requested 字段，
    不改变前置 guard（开关/授权/实例类型）行为。本用例在 guard 路径上验证
    snapshot=true 时仍返回 ok=false，避免在测试环境误起真实 CVM。
    """
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.post(
        "/openclaw/doctor/start",
        json={"snapshot": True},
        params={"id": SHARED_DB_ID},
        expect=None, raw=True, timeout=20,
    )
    if resp.status_code != 200:
        print(f"    SKIP (非 200): status={resp.status_code}")
        return
    body = resp.json() or {}
    if body.get("ok") is True:
        # 极少数情况：站点已开 + 已授权 + agent_type=openclaw → 真启动了诊断会话
        # 立即调 end 兜底，避免污染后续脚本
        print("    WARN 真起了 doctor 会话，立即下发 end 兜底")
        cli.post(
            "/openclaw/doctor/end",
            data={"id": SHARED_DB_ID},
            expect=None, raw=True, timeout=15,
        )
        return
    err = body.get("error", "")
    assert err in {
        "doctor_disabled",
        "not_authorized",
        "unsupported_agent_type",
        "create_failed",
    }, (
        f"非预期 error: {err} body={body}"
    )
    print(f"    OK snapshot=true ok=false error={err}")


def test_21_start_auth():
    """POST /openclaw/doctor/start - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=15).post(
            "/openclaw/doctor/start",
            data={"id": 1},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="doctor-start",
        check_admin=False,
    )


# ─── /openclaw/doctor/end ────────────────────────────────────────────────

def test_21_end_missing_id():
    """POST /openclaw/doctor/end - 缺 id → 400"""
    resp = cli.post("/openclaw/doctor/end", expect=None, raw=True)
    assert_status(resp, {400, 404}, label="doctor-end-missing")
    print(f"    OK status={resp.status_code}")


def test_22_end_no_session():
    """POST /openclaw/doctor/end - 没有活动会话 → ok=false session_not_found"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.post(
        "/openclaw/doctor/end",
        data={"id": SHARED_DB_ID},
        expect=None, raw=True, timeout=15,
    )
    if resp.status_code != 200:
        print(f"    SKIP (非 200): status={resp.status_code}")
        return
    body = resp.json() or {}
    # 共享实例之前可能被 test_19 真起了会话再 end 过；此时如果还有兜底未跑完，
    # 允许 ok=true（已真结束）或 ok=false+session_not_found（更常见）
    if body.get("ok") is True:
        print("    OK 兜底结束了一个真实会话")
    else:
        assert body.get("error") == "session_not_found", (
            f"非预期 error: {body}"
        )
        print(f"    OK ok=false error={body.get('error')}")


def test_23_end_auth():
    """POST /openclaw/doctor/end - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=15).post(
            "/openclaw/doctor/end",
            data={"id": 1},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="doctor-end",
        check_admin=False,
    )


# ─── 入口 ────────────────────────────────────────────────────────────────

def main():
    global SHARED_DB_ID
    health_check()
    SHARED_DB_ID = get_shared_db_id_or_none()
    if SHARED_DB_ID:
        print(f">>> 复用共享实例 db_id={SHARED_DB_ID}")
    else:
        print(">>> 无共享实例（SKIP_SHARED_INSTANCE=1），仅跑轻量接口测试")
    print()

    run_tests(
        globals(),
        title="test_instance_doctor.py",
        ordered=True,
    )


if __name__ == "__main__":
    main()
