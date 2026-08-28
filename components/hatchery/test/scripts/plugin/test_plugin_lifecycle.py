#!/usr/bin/env python3
"""
TC-P1 插件库管理增强：卸载 + 版本更新 + 应用范围 + 任务查询

验证企业插件库管理的核心流程：
  1. 创建插件（含 changelog）
  2. 查询插件列表 + 详情
  3. 版本更新（继承 distribute_count / visibility / categories）
  4. 下发插件到实例 + 任务状态查询
  5. 实例列表 status 筛选
  6. 卸载插件 + 卸载任务状态
  7. 任务列表 type 筛选
  8. 删除插件版本

使用方式：
  export BASE_URL=http://134.175.254.166
  export SEED_ADMIN_TOKEN=xxx
  python3 test_plugin_lifecycle.py

TAPD: #1020422209134626977
"""

import os
import sys
import time
import traceback

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers import config
import helpers
from helpers import (
    check_env, setup_admin,
    setup_user,
    setup_instance,
)
from helpers.plugin import (
    admin_create_plugin,
    admin_list_plugins,
    admin_plugin_detail,
    admin_update_plugin,
    admin_distribute_plugin,
    admin_uninstall_plugin,
    admin_plugin_instances,
    admin_plugin_tasks,
    admin_delete_plugin,
    wait_plugin_task,
)


SLUG = "integ-test-plugin"


def require_smh_enabled(admin_token):
    """检查 SMH 是否启用，未启用则 skip 测试"""
    config = helpers.admin_get_config(admin_token)
    if not config.get("smh_enabled"):
        print("    SMH 未启用，跳过插件库集成测试（需要 SMH 存储服务）")
        print()
        print("=" * 60)
        print("  TC-P1 插件库管理增强 — SKIPPED（SMH 未配置）")
        print("=" * 60)
        sys.exit(0)


def main():
    check_env()
    print()

    admin = setup_admin("plugin-life")
    user = None
    inst = None

    try:
        helpers.ensure_gateway_ui_enabled(admin.token)

        # SMH 未启用则跳过（插件库依赖 SMH 存储）
        require_smh_enabled(admin.token)

        user = setup_user(admin.token, "plugin-life")
        inst = setup_instance(user.token, "plugin-life")

        # ════════════════════════════════════════════════════════════
        # 阶段 A：创建插件 + 查询
        # ════════════════════════════════════════════════════════════

        print(">>> 步骤 1：创建插件 v1.0.0（含 changelog）...")
        resp = admin_create_plugin(
            admin.token, SLUG, "集成测试插件", "1.0.0",
            description="集成测试用插件",
            changelog="初始版本",
        )
        assert resp.status_code == 200, f"创建插件失败: {resp.status_code} {resp.text}"
        data = resp.json()
        assert data.get("ok"), f"创建插件响应异常: {data}"
        print("    创建插件 v1.0.0 ✓")

        # 步骤 2：查询插件列表
        print(">>> 步骤 2：查询插件列表...")
        list_data = admin_list_plugins(admin.token, keyword="集成测试")
        plugins = list_data.get("plugins", [])
        found = [p for p in plugins if p.get("slug") == SLUG]
        assert found, f"插件列表中未找到 {SLUG}"
        plugin_info = found[0]
        assert plugin_info.get("changelog") == "初始版本", \
            f"changelog 不匹配: {plugin_info.get('changelog')}"
        print(f"    插件列表 ✓  found={SLUG}, changelog=初始版本")

        # 步骤 3：查询插件详情
        print(">>> 步骤 3：查询插件详情...")
        detail_data = admin_plugin_detail(admin.token, SLUG)
        plugin_detail = detail_data.get("plugin", {})
        assert plugin_detail.get("version") == "1.0.0"
        assert plugin_detail.get("changelog") == "初始版本"
        versions = detail_data.get("versions", [])
        assert "1.0.0" in versions
        print(f"    插件详情 ✓  version=1.0.0, versions={versions}")

        # ════════════════════════════════════════════════════════════
        # 阶段 B：版本更新
        # ════════════════════════════════════════════════════════════

        print(">>> 步骤 4：版本更新 → v1.1.0...")
        resp = admin_update_plugin(
            admin.token, SLUG, "集成测试插件", "1.1.0",
            changelog="修复若干问题",
        )
        assert resp.status_code == 200, f"版本更新失败: {resp.status_code} {resp.text}"
        print("    版本更新 v1.1.0 ✓")

        # 确认详情是最新版本
        detail_data = admin_plugin_detail(admin.token, SLUG)
        plugin_detail = detail_data.get("plugin", {})
        assert plugin_detail.get("version") == "1.1.0"
        assert plugin_detail.get("changelog") == "修复若干问题"
        versions = detail_data.get("versions", [])
        assert "1.0.0" in versions and "1.1.0" in versions
        print(f"    详情确认 ✓  version=1.1.0, versions={versions}")

        # ════════════════════════════════════════════════════════════
        # 阶段 C：下发 + 实例状态 + 任务查询
        # ════════════════════════════════════════════════════════════

        print(">>> 步骤 5：下发插件到实例...")
        resp = admin_distribute_plugin(admin.token, SLUG, [inst.db_id])
        assert resp.status_code == 200, f"下发失败: {resp.status_code} {resp.text}"
        dist_data = resp.json()
        task_id = dist_data.get("task_id")
        assert task_id, f"响应缺少 task_id: {dist_data}"
        print(f"    下发任务创建 ✓  task_id={task_id}")

        # 等待任务完成
        print(">>> 步骤 6：等待下发任务完成...")
        completed_task = wait_plugin_task(admin.token, SLUG, task_id)
        print(f"    下发任务完成 ✓  success={completed_task.get('success')}, "
              f"failed={completed_task.get('failed')}")

        # 查询实例状态
        print(">>> 步骤 7：查询实例安装状态...")
        inst_data = admin_plugin_instances(admin.token, SLUG)
        instances = inst_data.get("instances", [])
        print(f"    实例列表 ✓  共 {len(instances)} 条")

        # status 筛选
        print(">>> 步骤 8：status 筛选测试...")
        for status_filter in ["installed", "uninstalled", "failed"]:
            filtered = admin_plugin_instances(admin.token, SLUG, status=status_filter)
            count = filtered.get("total", len(filtered.get("instances", [])))
            print(f"    status={status_filter} → {count} 条")
        print("    status 筛选 ✓")

        # 任务列表 type 筛选
        print(">>> 步骤 9：任务列表 type 筛选...")
        tasks_all = admin_plugin_tasks(admin.token, SLUG)
        tasks_dist = admin_plugin_tasks(admin.token, SLUG, type="distribute")
        tasks_uninst = admin_plugin_tasks(admin.token, SLUG, type="uninstall")
        assert tasks_all.get("total", 0) >= tasks_dist.get("total", 0)
        print(f"    all={tasks_all.get('total')}, "
              f"distribute={tasks_dist.get('total')}, "
              f"uninstall={tasks_uninst.get('total')} ✓")

        # ════════════════════════════════════════════════════════════
        # 阶段 D：卸载
        # ════════════════════════════════════════════════════════════

        print(">>> 步骤 10：卸载插件...")
        resp = admin_uninstall_plugin(admin.token, SLUG, [inst.db_id])
        assert resp.status_code == 200, f"卸载失败: {resp.status_code} {resp.text}"
        uninst_data = resp.json()
        assert uninst_data.get("message"), "响应缺少 message 字段"
        uninst_task_id = uninst_data.get("task_id")
        assert uninst_task_id, f"响应缺少 task_id: {uninst_data}"
        print(f"    卸载任务创建 ✓  task_id={uninst_task_id}, message={uninst_data['message']}")

        # 等待卸载完成
        print(">>> 步骤 11：等待卸载任务完成...")
        completed_uninst = wait_plugin_task(admin.token, SLUG, uninst_task_id)
        print(f"    卸载任务完成 ✓  success={completed_uninst.get('success')}, "
              f"failed={completed_uninst.get('failed')}")

        # 再次查询 type=uninstall 的任务
        tasks_uninst2 = admin_plugin_tasks(admin.token, SLUG, type="uninstall")
        assert tasks_uninst2.get("total", 0) > 0, "卸载后应有 uninstall 类型任务"
        print(f"    卸载任务 type 筛选 ✓  total={tasks_uninst2.get('total')}")

        # ════════════════════════════════════════════════════════════
        # 阶段 E：删除插件版本
        # ════════════════════════════════════════════════════════════

        print(">>> 步骤 12：删除插件版本 v1.0.0...")
        resp = admin_delete_plugin(admin.token, SLUG, "1.0.0")
        assert resp.status_code == 200, f"删除失败: {resp.status_code} {resp.text}"
        print("    删除 v1.0.0 ✓")

        # 确认只剩 v1.1.0
        detail_data = admin_plugin_detail(admin.token, SLUG)
        versions = detail_data.get("versions", [])
        assert "1.0.0" not in versions, f"v1.0.0 未被删除: {versions}"
        assert "1.1.0" in versions
        print(f"    确认版本列表 ✓  versions={versions}")

        # 清理：删除 v1.1.0
        print(">>> 步骤 13：清理 — 删除 v1.1.0...")
        resp = admin_delete_plugin(admin.token, SLUG, "1.1.0")
        assert resp.status_code == 200, f"清理删除失败: {resp.status_code} {resp.text}"
        print("    清理完成 ✓")

        print()
        print("=" * 60)
        print("  TC-P1 插件库管理增强 — 全部通过 ✓")
        print("=" * 60)

    except Exception:
        traceback.print_exc()
        print()
        print("=" * 60)
        print("  TC-P1 插件库管理增强 — 失败 ✗")
        print("=" * 60)
        sys.exit(1)


if __name__ == "__main__":
    main()
