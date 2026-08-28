"""
本地 agent（source=local）集成测试辅助模块

本地 agent 一期的集成测试全部走这里的 helper：
  - reporter 三接口的 Python 冒充实现（report / sync / commands/ack）
  - 本地实例创建脚手架（setup_local_instance）
  - 门户开关（enable_local_agent_feature）
  - 便利常量与工具（16-hex agent_id 生成、CID 派生规则等）

设计原则
--------
1. **无 CVM / 无 Gateway 依赖**：本地实例走的是纯 hatchery 数据面，我们从 Python
   直接 HTTP 冒充 reporter，走完 report → sync → ack 的完整闭环。
2. **随 slug 造 skill**：本地 add-skill 路径不查 skills 表（handleAddSkillLocal
   走 ClawHub 兜底），因此测试可以随意造 slug（如 `test-fake-<rand>`）不需要
   预置 skills 表行。
3. **安全 setup**：`enable_local_agent_feature` 幂等；重复调用无副作用。测试
   完成后不主动关闭（其它测试与本地 agent 无关，不受影响）。

关键字段派生规则
----------------
派生 CID（`instance_id` 字符串）：`local-<agent_type>-<local_agent_id[:5]>`
（详见 controller/local_agent.go 的 validateLocalAgentInputs）。

`local_agent_id` 必须是 **16 位 hex**（0-9a-f），否则 report 接口 400。

用法
----
::

    from helpers import (
        LocalAgent,
        enable_local_agent_feature,
        setup_local_instance,
        reporter_report,
        reporter_sync,
        reporter_ack,
    )

    enable_local_agent_feature(admin.token)
    la = setup_local_instance(user.token, "lifecycle")
    # la.db_id / la.instance_id / la.agent_id / la.agent_type ...
    commands = reporter_sync(user.token, la)
"""

from __future__ import annotations

import secrets
import time
from dataclasses import dataclass, field
from typing import Any

from helpers.api import admin_client, user_client, admin_get_config, admin_update_config


# ═══════════════════════════════════════════════════════════════════════════
# 常量
# ═══════════════════════════════════════════════════════════════════════════

DEFAULT_AGENT_TYPE = "workbuddy"
DEFAULT_AGENT_VERSION = "0.0.1-integration-test"
DEFAULT_OS = "linux/amd64"
LOCAL_AGENT_ID_HEX_LEN = 16


# ═══════════════════════════════════════════════════════════════════════════
# 数据类
# ═══════════════════════════════════════════════════════════════════════════


@dataclass
class LocalAgent:
    """冒充 reporter 需要维护的最小状态。

    ``db_id`` 与 ``instance_id`` 在首次 ``reporter_report`` 后回填。
    """

    agent_id: str  # 16-hex，reporter 视角的 local_agent_id
    agent_type: str = DEFAULT_AGENT_TYPE
    agent_version: str = DEFAULT_AGENT_VERSION
    host_name: str = "integration-host"
    os: str = DEFAULT_OS
    started_at: str = ""
    skills: list[dict[str, Any]] = field(default_factory=list)
    # 首次 report 后由服务端回填
    instance_id: str = ""
    db_id: int = 0

    def add_installed_skill(
        self, slug: str, version: str = "1.0.0",
        display_name: str | None = None, source: str = "clawpro",
    ) -> None:
        """把一条已装 skill 加入下次 report 会带的 skills 数组。"""
        entry = {"slug": slug, "version": version, "source": source}
        if display_name is not None:
            entry["display_name"] = display_name
        self.skills.append(entry)

    def remove_installed_skill(self, slug: str) -> None:
        """把一条已装 skill 从下次 report 会带的 skills 数组里移除。"""
        self.skills = [s for s in self.skills if s.get("slug") != slug]


# ═══════════════════════════════════════════════════════════════════════════
# 门户开关
# ═══════════════════════════════════════════════════════════════════════════


def enable_local_agent_feature(admin_token: str) -> None:
    """确保 SiteConfig.local_agent_enabled=true（幂等）。

    reporter 接口双层守卫之一。测试环境的 feature_allowlist 表默认为空
    → 「空表全开」自动放行第 ① 层，本函数只需保证第 ② 层放行即可。
    """
    print(">>> Setup：确保 local_agent_enabled=true ...")
    site_cfg = admin_get_config(admin_token)
    if site_cfg.get("local_agent_enabled") in (True, "true", 1):
        print("    local_agent_enabled 已开启 ✓")
        return
    admin_update_config(admin_token, local_agent_enabled="true")
    print("    local_agent_enabled=true ✓")


# ═══════════════════════════════════════════════════════════════════════════
# 工具
# ═══════════════════════════════════════════════════════════════════════════


def random_local_agent_id() -> str:
    """生成 16-hex 的伪 local_agent_id（reporter 里由机器信息 hash 派生）。"""
    return secrets.token_hex(LOCAL_AGENT_ID_HEX_LEN // 2)


def now_rfc3339() -> str:
    """当前 UTC 时间的 RFC3339 字符串（含 Z 后缀）。"""
    return time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())


def _report_body(la: LocalAgent) -> dict[str, Any]:
    return {
        "local_agent_id": la.agent_id,
        "agent_type": la.agent_type,
        "agent_version": la.agent_version,
        "host_name": la.host_name,
        "os": la.os,
        "started_at": la.started_at or now_rfc3339(),
        "skills": la.skills,
    }


# ═══════════════════════════════════════════════════════════════════════════
# reporter 三接口（Python 冒充）
# ═══════════════════════════════════════════════════════════════════════════


def reporter_report(user_token: str, la: LocalAgent, *, expect: int | None = 200):
    """POST /local-agent/report

    首次调用会创建 hatchery 侧的本地实例；后续调用会 upsert 已装 skill 列表并
    刷新 last_report_at。

    ``expect`` 为 None 时返回 raw Response，方便测负向；为 200 时返回 JSON dict
    并把 ``instance_id`` 回填到 ``la``。
    """
    body = _report_body(la)
    if expect is None:
        return user_client(user_token).post(
            "/local-agent/report", json=body, expect=None, raw=True, timeout=30,
        )
    data = user_client(user_token).post(
        "/local-agent/report", json=body, expect=expect, timeout=30,
    )
    if isinstance(data, dict):
        iid = data.get("instance_id")
        if iid:
            la.instance_id = iid
    return data


def reporter_sync(user_token: str, la: LocalAgent, *, status: str = "running",
                  expect: int | None = 200):
    """POST /local-agent/sync

    拉取待执行 commands 并刷新 last_report_at。返回 JSON dict（含
    ``commands`` 数组），或 ``expect=None`` 时返回 raw Response。
    """
    body = {
        "local_agent_id": la.agent_id,
        "agent_type": la.agent_type,
        "status": status,
    }
    if expect is None:
        return user_client(user_token).post(
            "/local-agent/sync", json=body, expect=None, raw=True, timeout=30,
        )
    return user_client(user_token).post(
        "/local-agent/sync", json=body, expect=expect, timeout=30,
    )


def reporter_ack(user_token: str, record_id: int, status: str,
                 *, error: str = "", version: str = "", ack_type: str = "",
                 result: str = "", session_id: str = "",
                 expect: int | None = 200):
    """POST /local-agent/commands/ack

    ``status`` 取值 ``success`` / ``failed``。``error`` 在 failed 时应非空；
    ``version`` 可选（success 时上报实际安装版本）。
    ``ack_type`` 与 sync 返回的 command.type 对齐，用于后端区分 skill/rule/
    uninstall_teamai 记录；默认为空串（skill 路径向后兼容）。
    """
    body: dict[str, Any] = {"id": record_id, "status": status}
    if ack_type:
        body["type"] = ack_type
    if error:
        body["error"] = error
    if version:
        body["version"] = version
    if result:
        body["result"] = result
    if session_id:
        body["session_id"] = session_id
    if expect is None:
        return user_client(user_token).post(
            "/local-agent/commands/ack", json=body, expect=None, raw=True, timeout=30,
        )
    return user_client(user_token).post(
        "/local-agent/commands/ack", json=body, expect=expect, timeout=30,
    )


# ═══════════════════════════════════════════════════════════════════════════
# 用户/管理员端本地实例查询
# ═══════════════════════════════════════════════════════════════════════════


def user_remove_local_agent(user_token: str, instance_db_id: int, *, expect: int = 200):
    """POST /local-agent/remove — 用户端移除自己的本地 agent（走卸载链路）。

    返回 JSON dict（含 ``task_id``），卸载本身由 reporter 异步执行。
    """
    return user_client(user_token).post(
        "/local-agent/remove",
        json={"instance_id": instance_db_id},
        expect=expect, timeout=30,
    )


def user_get_local_agent_availability(user_token: str, *, expect: int = 200):
    """GET /local-agent/availability → {"enabled": bool}"""
    return user_client(user_token).get("/local-agent/availability", expect=expect)


def admin_get_local_agent_types(admin_token: str, *, expect: int = 200):
    """GET /admin/local-agent-types"""
    return admin_client(admin_token).get("/admin/local-agent-types", expect=expect)


def admin_check_feature_allowlist(admin_token: str, feature_type: str,
                                  identifier: str, *, expect: int | None = 200):
    """GET /admin/feature-allowlist/check"""
    params = {"type": feature_type, "identifier": identifier}
    if expect is None:
        return admin_client(admin_token).get(
            "/admin/feature-allowlist/check",
            params=params, expect=None, raw=True,
        )
    return admin_client(admin_token).get(
        "/admin/feature-allowlist/check",
        params=params, expect=expect,
    )


# ═══════════════════════════════════════════════════════════════════════════
# 本地实例脚手架
# ═══════════════════════════════════════════════════════════════════════════


def _find_local_instance_db_id(user_token: str, instance_id: str) -> int:
    """从 /openclaw/list 里按 instance_id 找到 db_id。"""
    data = user_client(user_token).get(
        "/openclaw/list", params={"page": 1, "page_size": 100},
    )
    instances = data.get("instances", [])
    for inst in instances:
        if inst.get("instance_id") == instance_id or inst.get("InstanceId") == instance_id:
            return inst.get("id") or inst.get("ID")
    raise RuntimeError(
        f"本地实例 instance_id={instance_id} 未在 /openclaw/list 中找到"
    )


def setup_local_instance(user_token: str, scenario: str, *,
                         agent_type: str = DEFAULT_AGENT_TYPE,
                         agent_version: str = DEFAULT_AGENT_VERSION,
                         host_name: str | None = None,
                         installed_skills: list[dict[str, Any]] | None = None,
                         ) -> LocalAgent:
    """创建本地 agent 实例（冒充 reporter 首次 report）。

    等价于 ``setup_instance`` 的本地版：调用 report → 回填 instance_id / db_id →
    返回 ``LocalAgent`` 供后续 sync/ack 使用。**不涉及 CVM，也不等待 Gateway。**

    ``installed_skills`` 允许在首次 report 就带上「已装 skill」列表（模拟
    reporter 端已经预装了一些 skill 的场景）。
    """
    la = LocalAgent(
        agent_id=random_local_agent_id(),
        agent_type=agent_type,
        agent_version=agent_version,
        host_name=host_name or f"integ-{scenario}-{int(time.time())}",
        skills=list(installed_skills or []),
    )
    print(f">>> 冒充 reporter report 创建本地实例 (agent_id={la.agent_id[:8]}..., "
          f"agent_type={la.agent_type}) ...")
    data = reporter_report(user_token, la)
    assert isinstance(data, dict), f"report 响应非 dict: {data}"
    assert la.instance_id, "report 响应未返回 instance_id"

    la.db_id = _find_local_instance_db_id(user_token, la.instance_id)
    print(f"    本地实例就绪 ✓  db_id={la.db_id}, instance_id={la.instance_id}")
    return la
