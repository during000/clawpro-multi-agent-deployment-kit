#!/usr/bin/env python3
"""
本地 agent — 卸载全链路端到端测试

覆盖 /local-agent/remove → sync → ack 的完整闭环（三期卸载链路）：

阶段 A：成功路径
  1. reporter report 建实例 + 带一条已装 skill
  2. 用户 /local-agent/remove → 返回 task_id，实例进入 destroying 中间态
     （current_operation=uninstall_local_agent）
  3. reporter sync → commands 含 {type: uninstall_teamai, id: <task_id>}
  4. reporter ack success → task=success
  5. 用户 /openclaw/list 查不到该实例（已软删），DB 侧实例 last_known_status=destroyed
     + current_operation 清空
  6. reporter 再次 report → 实例复活为 running（等同本地重新接入，符合预期语义）

阶段 B：失败路径
  1. 重新建实例 + remove 拿到 task
  2. reporter sync 拿 command
  3. reporter ack failed（带 error）→ task=failed
  4. 实例回到 running、current_operation 清空、未被软删（保留可重试）
"""

import os
import sys
import time
import traceback

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

import helpers  # noqa: E402
from helpers import (  # noqa: E402
    check_env,
    setup_admin,
    setup_user,
    enable_local_agent_feature,
    setup_local_instance,
    reporter_report,
    reporter_sync,
    reporter_ack,
    user_remove_local_agent,
    user_client,
)


def _find_uninstall_command(commands):
    """从 sync 返回的 commands 里找 uninstall_teamai 命令。"""
    for cmd in commands or []:
        if cmd.get("type") == "uninstall_teamai":
            return cmd
    return None


def _get_instance_status(user_token, instance_id):
    """从 /openclaw/list 找实例当前状态（找不到返回 None，表示已软删不可见）。"""
    data = user_client(user_token).get(
        "/openclaw/list", params={"page": 1, "page_size": 100},
    )
    instances = data.get("instances", [])
    for inst in instances:
        iid = inst.get("instance_id") or inst.get("InstanceId")
        if iid == instance_id:
            return inst
    return None


def main():
    check_env()
    print()

    admin = setup_admin("local-rm")
    user = None
    la = None

    try:
        enable_local_agent_feature(admin.token)
        user = setup_user(admin.token, "local-rm")

        # ════════════════════════════════════════════════════════════════
        # 阶段 A：成功路径
        # ════════════════════════════════════════════════════════════════
        print("═══ 阶段 A：卸载成功路径 ═══")
        la = setup_local_instance(
            user.token, "rm-a",
            installed_skills=[{"slug": "skill-rm-a", "version": "1.0.0"}],
        )
        db_id = la.db_id
        iid = la.instance_id
        print(f"    本地实例就绪 ✓  db_id={db_id}, instance_id={iid}")

        # 步骤 A1：用户端发起卸载
        print(">>> 步骤 A1：用户 /local-agent/remove 发起卸载 ...")
        rm = user_remove_local_agent(user.token, db_id)
        task_id = rm.get("task_id")
        assert task_id, f"remove 响应应含 task_id: {rm}"
        print(f"    task_id={task_id} ✓")

        # 步骤 A2：下发后实例进入 destroying 中间态
        print(">>> 步骤 A2：卸载下发后实例应进入 destroying 中间态 ...")
        inst_after_remove = _get_instance_status(user.token, iid)
        assert inst_after_remove is not None, "卸载下发后实例应仍在列表"
        status_after_remove = (
            inst_after_remove.get("LastKnownStatus")
            or inst_after_remove.get("last_known_status")
        )
        assert status_after_remove == "destroying", (
            f"下发后 LastKnownStatus 应为 destroying，实际={status_after_remove}"
        )
        print(f"    status={status_after_remove} ✓")

        # 步骤 A3：reporter sync 拉取 uninstall_teamai 命令
        print(">>> 步骤 A3：reporter sync 拉取 uninstall_teamai 命令 ...")
        sync_data = reporter_sync(user.token, la)
        cmd = _find_uninstall_command(sync_data.get("commands", []))
        assert cmd, f"sync commands 未含 uninstall_teamai: {sync_data}"
        assert cmd.get("id") == task_id, (
            f"command id 应等于 remove 返回的 task_id: {cmd.get('id')} vs {task_id}"
        )
        print(f"    id={cmd['id']}, type={cmd.get('type')} ✓")

        # 步骤 A4：reporter ack success
        print(">>> 步骤 A4：reporter ack success ...")
        ack_data = reporter_ack(user.token, task_id, "success", ack_type="uninstall_teamai")
        assert ack_data.get("status") == "success", (
            f"ack 应返回 status=success: {ack_data}"
        )
        print(f"    ack status={ack_data.get('status')} ✓")

        # 步骤 A5：ack 后实例从列表消失（已软删），DB 侧置 destroyed + 清 current_operation
        print(">>> 步骤 A5：ack 后实例应软删（列表不可见）+ 置 destroyed 终态 ...")
        inst_after_ack = _get_instance_status(user.token, iid)
        assert inst_after_ack is None, (
            f"ack 后实例应已软删（列表不可见），实际仍可见: {inst_after_ack}"
        )
        # 通过 DB 直查（这里用 list 找不到即视为软删；current_operation/destroyed 由
        # 单测 TestLocalAgentUninstall_Flow_E2E 覆盖，集成脚本只验证「列表消失」这一可见行为）
        print("    实例已从列表消失（已软删）✓")

        # 步骤 A6：reporter 再次 report → 实例复活为 running
        print(">>> 步骤 A6：reporter 再次 report 应复活实例为 running ...")
        reporter_report(user.token, la)
        inst_after_report = _get_instance_status(user.token, iid)
        assert inst_after_report is not None, "再次 report 后实例应复活（列表可见）"
        status_after_report = (
            inst_after_report.get("LastKnownStatus")
            or inst_after_report.get("last_known_status")
        )
        assert status_after_report == "running", (
            f"复活后 LastKnownStatus 应为 running，实际={status_after_report}"
        )
        print(f"    复活 status={status_after_report} ✓")
        print()

        # ════════════════════════════════════════════════════════════════
        # 阶段 B：失败路径
        # ════════════════════════════════════════════════════════════════
        print("═══ 阶段 B：卸载失败路径 ═══")
        la_b = setup_local_instance(user.token, "rm-b")
        db_id_b = la_b.db_id
        iid_b = la_b.instance_id
        print(f"    本地实例就绪 ✓  db_id={db_id_b}, instance_id={iid_b}")

        rm_b = user_remove_local_agent(user.token, db_id_b)
        task_id_b = rm_b.get("task_id")
        assert task_id_b, f"remove 响应应含 task_id: {rm_b}"

        sync_b = reporter_sync(user.token, la_b)
        cmd_b = _find_uninstall_command(sync_b.get("commands", []))
        assert cmd_b, f"sync commands 未含 uninstall_teamai: {sync_b}"

        print(">>> 步骤 B1：reporter ack failed（带 error）...")
        ack_b = reporter_ack(
            user.token, task_id_b, "failed", error="uninstall timeout",
            ack_type="uninstall_teamai",
        )
        assert ack_b.get("status") == "failed", (
            f"ack 应返回 status=failed: {ack_b}"
        )
        print(f"    ack status={ack_b.get('status')} ✓")

        print(">>> 步骤 B2：ack failed 后实例回到 running、未被软删 ...")
        inst_after_fail = _get_instance_status(user.token, iid_b)
        assert inst_after_fail is not None, "ack failed 后实例应仍在列表（未被软删）"
        status_after_fail = (
            inst_after_fail.get("LastKnownStatus")
            or inst_after_fail.get("last_known_status")
        )
        assert status_after_fail == "running", (
            f"ack failed 后 LastKnownStatus 应为 running，实际={status_after_fail}"
        )
        print(f"    复活 status={status_after_fail} ✓")

        print()
        print("✅ 本地 agent 卸载全链路端到端测试全部通过")
        return True

    except AssertionError as e:
        print(f"\n❌ 断言失败：{e}")
        traceback.print_exc()
        return False
    except Exception as e:  # noqa: BLE001
        print(f"\n❌ 异常：{e}")
        traceback.print_exc()
        return False
    finally:
        # 清理：reporter 再次 report 复活 Phase A 实例，避免残留
        if la is not None:
            try:
                reporter_report(user.token, la)
            except Exception:  # noqa: BLE001
                pass


if __name__ == "__main__":
    ok = main()
    sys.exit(0 if ok else 1)
