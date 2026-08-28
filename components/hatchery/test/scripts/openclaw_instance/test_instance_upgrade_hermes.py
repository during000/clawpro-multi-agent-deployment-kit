#!/usr/bin/env python3
"""
集成测试：实例管理 - Hermes 一键升级（C 组，仅契约级）

背景：
    一键升级链路（备份→SMH上传→重装→恢复）最初仅支持 OpenClaw，后扩展到 Hermes
    （见 model.AgentType.SupportsUpgrade / NeedsRuntimeUserCorrection、
    controller.ResolveScript 对 backup_pre_reinstall / restore_post_reinstall 的
    hermes 分支）。此前该能力只有 test_instance_upgrade.py 一份通用契约测试，
    未区分 agent_type，无法验证：
        1) /admin/agent-types 返回的 hermes 条目 supports_upgrade 确实为 true；
        2) Hermes 类型实例调用升级接口时，能正确进入"类型校验通过"分支
           （而不是被 checkInstanceSupportsUpgrade 误拒绝）；
        3) 契约层（参数校验/鉴权/状态拒绝）对 Hermes 实例同样成立。

覆盖接口：
    GET  /admin/agent-types       校验 hermes.supports_upgrade == true
    POST /openclaw/upgrade        一键升级（用户侧接口，Hermes 实例，仅契约级）
    POST /openclaw/upgrade/retry  升级失败重试（用户侧接口，Hermes 实例，仅契约级）

升级真实流程会重装系统、耗时 15+ 分钟，本测试**绝不在共享实例上触发真实升级**：
    - 只测参数校验、状态拒绝、404、鉴权、agent_type 能力位契约
    - happy path 路径：仅在确认实例处于 upgrade_failed 状态时才尝试 retry
"""
import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers import setup_admin, setup_user
from helpers.api import (
    ApiClient, seed, user_client,
    health_check, run_tests,
    auth_test_suite, assert_status,
)
from helpers.hermes import HERMES_AGENT_TYPE, setup_hermes_instance
from _instance_helpers import NONEXISTENT_DB_ID, get_status


HERMES_DB_ID = None
HERMES_USER_CLIENT = None


def _client():
    """Hermes 实例所属用户的 ApiClient；未成功创建实例时退化为 seed（admin token）
    以便纯契约用例（缺 id/404/鉴权等，与具体实例无关）仍可运行。
    """
    return HERMES_USER_CLIENT if HERMES_USER_CLIENT is not None else seed


# ─── /admin/agent-types 能力位契约 ───────────────────────────────────────

def test_01_hermes_supports_upgrade_flag():
    """GET /admin/agent-types - hermes 条目 supports_upgrade 必须为 true

    回归保护：checkInstanceSupportsUpgrade 依赖 model.AgentTypeSupportsUpgrade
    读取该能力位，若被误改为 false，Hermes 升级入口会被 400 拒绝，
    但通用契约测试（test_instance_upgrade.py）不感知具体 agent_type，
    无法捕获此类回归，因此在此专门断言。
    """
    resp = seed.get("/admin/agent-types", expect=None, raw=True)
    assert_status(resp, {200}, label="admin-agent-types")
    body = resp.json() or {}
    types = body.get("agent_types") or body.get("types") or (
        body if isinstance(body, list) else []
    )
    hermes_entry = None
    for t in types:
        code = t.get("code") or t.get("Code")
        if code == HERMES_AGENT_TYPE:
            hermes_entry = t
            break
    assert hermes_entry is not None, f"未找到 hermes 类型条目: {types}"
    supports_upgrade = hermes_entry.get("supports_upgrade")
    assert supports_upgrade is True, (
        f"hermes.supports_upgrade 应为 true，实际: {supports_upgrade}，"
        f"完整条目: {hermes_entry}"
    )
    print("    hermes.supports_upgrade == true ✓")


# ─── /openclaw/upgrade（Hermes 实例）──────────────────────────────────────

def test_02_hermes_upgrade_missing_id():
    """POST /openclaw/upgrade - 缺 id → 400（与 agent_type 无关的契约前置校验）"""
    resp = _client().post("/openclaw/upgrade", data={}, expect=None, raw=True)
    assert_status(resp, {400, 404}, label="hermes-upgrade-missing-id")
    print(f"    OK status={resp.status_code}")


def test_03_hermes_upgrade_nonexistent_id():
    """POST /openclaw/upgrade?id=NONEXISTENT → 4xx"""
    resp = _client().post(
        "/openclaw/upgrade",
        data={"id": NONEXISTENT_DB_ID},
        expect=None, raw=True, timeout=30,
    )
    assert_status(resp, {400, 404, 500}, label="hermes-upgrade-not-found")
    print(f"    OK status={resp.status_code}")


def test_04_hermes_upgrade_no_real_trigger():
    """POST /openclaw/upgrade - Hermes 实例仅契约级（默认不真实触发升级）

    关键回归点：请求不应因为 agent_type=hermes 被 checkInstanceSupportsUpgrade
    拒绝（那样会返回业务错误码 400 且 error 含"不支持一键升级"字样）。
    默认不触发真实升级，仅验证请求没有被类型校验挡在门外。
    """
    if not HERMES_DB_ID:
        print("    SKIP (无 hermes 共享实例)")
        return
    if os.environ.get("ALLOW_REAL_UPGRADE") == "1":
        resp = _client().post(
            "/openclaw/upgrade",
            data={"id": HERMES_DB_ID},
            expect=None, raw=True, timeout=30,
        )
        assert resp.status_code in (200, 409), (
            f"期望 200（已开始/已最新）或 409（状态拒绝）, 实际 {resp.status_code}"
        )
        print(f"    OK status={resp.status_code}")
        return

    # 不真实触发：仅当实例处于 running 时才做该断言，避免其它状态下
    # 400（状态拒绝）与"类型不支持"混淆。若返回码是 400 且 error 命中
    # "不支持一键升级"关键字，说明 SupportsUpgrade 能力位配置发生回归。
    status_data = get_status(HERMES_DB_ID, client=_client())
    current_status = status_data.get("status", "")
    if current_status != "running":
        print(f"    SKIP (hermes 实例当前状态={current_status}，非 running 不做该断言)")
        return

    resp = _client().post(
        "/openclaw/upgrade",
        data={"id": HERMES_DB_ID},
        expect=None, raw=True, timeout=30,
    )
    if resp.status_code == 400:
        body = resp.json() if resp.content else {}
        err = body.get("error", "")
        assert "不支持一键升级" not in err and "supports_upgrade" not in err.lower(), (
            f"Hermes 实例被 checkInstanceSupportsUpgrade 误拒绝，回归了能力位配置: {err}"
        )
        print(f"    OK status=400 但非类型不支持错误 error={err!r}")
    else:
        print(f"    OK status={resp.status_code}（未被类型校验拒绝）")


def test_05_hermes_upgrade_auth():
    """POST /openclaw/upgrade - 认证三件套（用户侧接口，跳过管理员检查）"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).post(
            "/openclaw/upgrade",
            data={"id": 1},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="hermes-upgrade",
        check_admin=False,
    )


# ─── /openclaw/upgrade/retry（Hermes 实例）───────────────────────────────

def test_06_hermes_upgrade_retry_missing_id():
    """POST /openclaw/upgrade/retry - 缺 id → 400"""
    resp = _client().post(
        "/openclaw/upgrade/retry", data={}, expect=None, raw=True,
    )
    assert_status(resp, {400, 404}, label="hermes-upgrade-retry-missing-id")
    print(f"    OK status={resp.status_code}")


def test_07_hermes_upgrade_retry_nonexistent_id():
    """POST /openclaw/upgrade/retry?id=NONEXISTENT → 4xx"""
    resp = _client().post(
        "/openclaw/upgrade/retry",
        data={"id": NONEXISTENT_DB_ID},
        expect=None, raw=True, timeout=30,
    )
    assert_status(resp, {400, 404, 500}, label="hermes-upgrade-retry-not-found")
    print(f"    OK status={resp.status_code}")


def test_08_hermes_upgrade_retry_state_reject():
    """POST /openclaw/upgrade/retry - 非 upgrade_failed 状态 → 400"""
    if not HERMES_DB_ID:
        print("    SKIP (无 hermes 共享实例)")
        return
    status_data = get_status(HERMES_DB_ID, client=_client())
    if status_data.get("status") == "upgrade_failed":
        print("    SKIP (实例处于 upgrade_failed，跳过避免触发真实重试)")
        return
    resp = _client().post(
        "/openclaw/upgrade/retry",
        data={"id": HERMES_DB_ID},
        expect=None, raw=True, timeout=30,
    )
    assert_status(resp, {400, 409}, label="hermes-upgrade-retry-state-reject")
    print(f"    OK status={resp.status_code}")


def test_09_hermes_upgrade_retry_auth():
    """POST /openclaw/upgrade/retry - 认证三件套（用户侧接口，跳过管理员检查）"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).post(
            "/openclaw/upgrade/retry",
            data={"id": 1},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="hermes-upgrade-retry",
        check_admin=False,
    )


# ─── 入口 ────────────────────────────────────────────────────────────────

def main():
    global HERMES_DB_ID, HERMES_USER_CLIENT
    health_check()

    # 创建一个独立的 hermes 实例用于本文件的升级契约测试（不复用跨文件共享实例，
    # 因为共享实例默认 agent_type=openclaw，无法验证 hermes 专属能力位契约）。
    # 若创建失败（如镜像未导入），依赖实例的用例会自动 SKIP，不影响纯契约用例。
    try:
        admin_ctx = setup_admin("hermes-upgrade")
        user_ctx = setup_user(admin_ctx.token, "hermes-upgrade")
        inst = setup_hermes_instance(user_ctx.token, "upgrade")
        HERMES_DB_ID = inst.db_id
        HERMES_USER_CLIENT = user_client(user_ctx.token)
        print(f">>> 创建 hermes 实例用于升级契约测试 db_id={HERMES_DB_ID}")
    except Exception as e:
        print(f">>> [WARN] 创建 hermes 实例失败，依赖实例的用例将 SKIP: {e}")
        HERMES_DB_ID = None
        HERMES_USER_CLIENT = None
    print()

    run_tests(
        globals(),
        title="test_instance_upgrade_hermes.py",
        ordered=True,
    )


if __name__ == "__main__":
    main()
