#!/usr/bin/env python3
"""
本地 agent — 全生命周期黄金路径

一次跑完最重要的正向流程：
  1. SiteConfig.local_agent_enabled 恒为 true（测试环境保证）
  2. reporter 首次 report → 创建本地实例
  3. 用户端 /openclaw/list 与 admin /admin/instances?source=local 都能看到
  4. reporter 幂等再 report（模拟 agent 重启）→ 同一 instance_id
  5. sync 无待办命令 → commands=[]，同时刷新 last_report_at
  6. 用户端查 status=running（心跳新鲜）
  7. 用户 /openclaw/delete 删除本地实例应被拒绝（400，本地实例不支持远程删除），再删仍 4xx

不覆盖 skill 分发链路（那是 test_local_agent_skill_flow.py 的职责）。
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
    admin_client,
    user_client,
    enable_local_agent_feature,
    setup_local_instance,
    reporter_report,
    reporter_sync,
)


def main():
    check_env()
    print()

    admin = setup_admin("local-life")
    user = None
    la = None

    try:
        enable_local_agent_feature(admin.token)
        user = setup_user(admin.token, "local-life")

        # ══════════════════════════════════════════════════════════════
        # 步骤 1：冒充 reporter 首次 report → 创建本地实例
        # ══════════════════════════════════════════════════════════════
        la = setup_local_instance(user.token, "life")
        assert la.db_id > 0, f"db_id 应回填: {la}"
        assert la.instance_id.startswith("local-"), (
            f"instance_id 应以 local- 开头: {la.instance_id}"
        )
        assert la.agent_type in la.instance_id, (
            f"instance_id 应包含 agent_type: {la.instance_id}"
        )
        print()

        # ══════════════════════════════════════════════════════════════
        # 步骤 2：用户端列表能看到 source=local 的实例
        # ══════════════════════════════════════════════════════════════
        print(">>> 步骤 2：用户 /openclaw/list 检查 source=local ...")
        data = user_client(user.token).get("/openclaw/list", params={"page_size": 100})
        instances = data.get("instances", [])
        found = next(
            (x for x in instances if x.get("instance_id") == la.instance_id
             or x.get("InstanceId") == la.instance_id),
            None,
        )
        assert found is not None, (
            f"用户列表应含新建的本地实例 {la.instance_id}，"
            f"实际返回 {len(instances)} 条"
        )
        source = found.get("source") or found.get("Source")
        assert source == "local", f"实例 source 应为 local，实际={source}"
        print(f"    找到本地实例 ✓  source={source}")
        print()

        # ══════════════════════════════════════════════════════════════
        # 步骤 3：admin /admin/instances?source=local 能看到
        # ══════════════════════════════════════════════════════════════
        print(">>> 步骤 3：admin /admin/instances?source=local 检查 ...")
        admin_data = admin_client(admin.token).get(
            "/admin/instances", params={"source": "local", "page_size": 100},
        )
        admin_instances = admin_data.get("instances", [])
        admin_found = next(
            (x for x in admin_instances if x.get("instance_id") == la.instance_id
             or x.get("InstanceId") == la.instance_id),
            None,
        )
        assert admin_found is not None, (
            f"admin 视角应能过滤到本地实例 {la.instance_id}"
        )
        print(f"    admin 视角找到 ✓")
        print()

        # ══════════════════════════════════════════════════════════════
        # 步骤 4：幂等 report（模拟 agent 重启）→ 同一 instance_id
        # ══════════════════════════════════════════════════════════════
        print(">>> 步骤 4：二次 report（幂等）...")
        first_iid = la.instance_id
        first_db_id = la.db_id
        data = reporter_report(user.token, la)
        assert data.get("instance_id") or data.get("skills_synced") is not None, (
            f"二次 report 失败: {data}"
        )
        assert la.instance_id == first_iid, (
            f"二次 report 应返回同一 instance_id: {first_iid} vs {la.instance_id}"
        )
        # 二次 report 后 db_id 也不该变（同一行 upsert）
        recheck_data = user_client(user.token).get(
            "/openclaw/list", params={"page_size": 100},
        )
        for inst in recheck_data.get("instances", []):
            if (inst.get("instance_id") == la.instance_id
                    or inst.get("InstanceId") == la.instance_id):
                new_id = inst.get("id") or inst.get("ID")
                assert new_id == first_db_id, (
                    f"二次 report 不应产生新 db 行: {first_db_id} vs {new_id}"
                )
                break
        print(f"    幂等 ✓  instance_id={first_iid}, db_id={first_db_id}")
        print()

        # ══════════════════════════════════════════════════════════════
        # 步骤 5：sync 无待办命令
        # ══════════════════════════════════════════════════════════════
        print(">>> 步骤 5：reporter sync（无待办命令）...")
        sync_data = reporter_sync(user.token, la)
        assert sync_data.get("ok"), f"sync 失败: {sync_data}"
        commands = sync_data.get("commands", [])
        assert commands == [] or commands == None, (
            f"新建实例首个 sync 应无 commands，实际={commands}"
        )
        print(f"    commands=[] ✓")
        print()

        # ══════════════════════════════════════════════════════════════
        # 步骤 6：用户查 status=running（心跳刚上报，肯定活）
        # ══════════════════════════════════════════════════════════════
        print(">>> 步骤 6：用户 /openclaw/status 查本地实例状态 ...")
        status_data = user_client(user.token).get(
            "/openclaw/status", params={"id": la.db_id},
        )
        status = status_data.get("status", "")
        actions = status_data.get("actions", None)
        assert status == "running", (
            f"心跳新鲜的本地实例应返回 running，实际={status}"
        )
        # actions 应裁剪为空数组（本地实例 hatchery 无法远程操作）
        assert actions == [], (
            f"本地实例 actions 应为 []，实际={actions}"
        )
        print(f"    status=running, actions=[] ✓")
        print()

        # ══════════════════════════════════════════════════════════════
        # 步骤 7：用户 /openclaw/delete 删除本地实例
        # ══════════════════════════════════════════════════════════════
        print(">>> 步骤 7：用户 /openclaw/delete 删除本地实例 ...")
        del_resp = user_client(user.token).post(
            "/openclaw/delete", data={"id": str(la.db_id)},
            expect=None, raw=True, timeout=30,
        )
        assert del_resp.status_code == 400, (
            f"本地实例删除应被拒绝（400，不支持此操作），实际={del_resp.status_code} body={del_resp.text}"
        )
        print("    本地实例拒绝远程删除 ✓")

        # 再次删除仍应 400（本地实例始终不支持远程删除）
        print(">>> 步骤 7b：再次删除（仍应 4xx）...")
        del2_resp = user_client(user.token).post(
            "/openclaw/delete", data={"id": str(la.db_id)},
            expect=None, raw=True, timeout=30,
        )
        assert del2_resp.status_code in (400, 404), (
            f"再次删除应 4xx，实际={del2_resp.status_code} body={del2_resp.text}"
        )
        print(f"    二次删除 {del2_resp.status_code} ✓")
        # 标记已清理，跳过 finally 清理
        la = None

        print()
        print("本地 agent 生命周期测试通过 ✅")

    except Exception as e:
        print(f"\n本地 agent 生命周期测试失败 ❌: {e}")
        traceback.print_exc()
        sys.exit(1)

    finally:
        # 兜底清理（若中间断开）
        if la is not None and la.db_id:
            try:
                user_client(user.token).post(
                    "/openclaw/delete", data={"id": str(la.db_id)},
                    expect=None, raw=True, timeout=30,
                )
            except Exception:
                pass


if __name__ == "__main__":
    main()
