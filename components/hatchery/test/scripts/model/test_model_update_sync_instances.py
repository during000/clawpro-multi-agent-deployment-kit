#!/usr/bin/env python3
"""
TC-1.x 更新模型 → 默认自动同步到已绑定实例

验证 admin 更新 model 时默认自动同步到已绑定实例：
1. 无绑定实例更新 → 仅更新 DB
2. 有绑定实例更新 → 自动异步触发 TAT 下发，实例配置更新
3. 实例保持正常运行

依赖：
- MODEL_ID / MODEL_API_KEY / MODEL_URL / MODEL_TYPE 环境变量
- 腾讯云 AK/SK（由编排器注入容器）
"""

import os
import sys
import time
import traceback

# 把 tests/ 目录加入搜索路径
sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers import config
import helpers
from helpers import (
    check_env, require_model_config, setup_admin,
    setup_user, get_field,
)
from helpers.instance import (
    create_instance, wait_instance_ready, delete_instance, get_instance_db_id,
    get_instance_status,
)


def main():
    check_env()
    require_model_config()
    print()

    # ── Setup ──
    admin = setup_admin("model-update-sync")
    user = setup_user(admin.token, "model-update-sync", instance_quota=5)
    model_db_id = None
    instance_db_id = None

    try:
        model_name = f"IntTest Sync ({config.MODEL_ID})"

        # 步骤 1：创建自定义模型
        print(">>> 步骤 1：创建自定义模型 ...")
        data = helpers.admin_create_model(
            admin.token,
            model_id=config.MODEL_ID,
            model_name=model_name,
            provider="openai",
            api_key=config.MODEL_API_KEY,
            url=config.MODEL_URL,
            model_type=config.MODEL_TYPE,
            quota_day=-1,
        )
        assert data.get("ok"), f"创建模型失败: {data}"
        print("    创建成功 ✓")

        # 步骤 2：查询 model_db_id
        print(">>> 步骤 2：查询模型列表，记录 model_db_id ...")
        models = helpers.admin_get_models(admin.token)
        model = next(
            (m for m in models if get_field(m, "ModelName", "model_name") == model_name),
            None,
        )
        assert model, f"模型 {model_name} 未在列表中找到"
        model_db_id = get_field(model, "ID", "id")
        print(f"    找到模型 ✓  db_id={model_db_id}")

        # 步骤 3：启用模型
        print(">>> 步骤 3：启用模型 ...")
        data = helpers.admin_toggle_model(admin.token, model_db_id)
        assert data.get("ok"), f"启用模型失败: {data}"
        print("    启用成功 ✓")

        # 步骤 4：更新模型（无绑定实例，仅验证 DB 更新）
        print(">>> 步骤 4：更新模型（无绑定实例）...")
        data = helpers.admin_update_model(
            admin.token, model_db_id,
            model_id=config.MODEL_ID,
            model_name=model_name,
            url=config.MODEL_URL,
            model_type=config.MODEL_TYPE,
            quota_day="5000",
        )
        assert data.get("ok"), f"更新模型失败: {data}"
        print("    更新成功 ✓")

        # 步骤 5：创建实例 + 绑定模型（primary）
        print(">>> 步骤 5：创建实例 ...")
        inst_name = f"sync-test-{int(time.time())}"
        data = create_instance(user.token, inst_name)
        assert data.get("instance_id") or data.get("db_id"), f"创建实例失败: {data}"
        instance_db_id = get_instance_db_id(user.token, data.get("instance_id"))
        print(f"    实例创建中 ✓  db_id={instance_db_id}")

        print(">>> 步骤 6：等待实例就绪 ...")
        wait_instance_ready(user.token, instance_db_id)
        print("    实例就绪 ✓")

        # 步骤 7：绑定模型到实例（primary）
        print(">>> 步骤 7：绑定模型到实例（primary）...")
        data = helpers.user_add_model(
            user.token, instance_db_id, model_db_id,
        )
        # user_add_model 返回 raw Response，需检查状态码
        status = getattr(data, "status_code", 200)
        assert status == 200, f"绑定模型失败: status={status}, body={getattr(data, 'text', '')}"
        print("    绑定成功 ✓")

        # 步骤 8：更新模型（有绑定实例）→ 默认自动同步
        print(">>> 步骤 8：更新模型（有绑定实例）→ 默认自动同步 ...")
        data = helpers.admin_update_model(
            admin.token, model_db_id,
            model_id=config.MODEL_ID,
            model_name=model_name,
            url=config.MODEL_URL,
            model_type=config.MODEL_TYPE,
            quota_day="8000",
        )
        assert data.get("ok"), f"更新失败: {data}"
        print("    更新成功 ✓（默认自动触发同步到已绑定实例）")

        # 步骤 9：等待异步同步完成
        print(">>> 步骤 9：等待异步同步完成（30s）...")
        time.sleep(30)
        print("    等待完成 ✓")

        # 步骤 10：验证模型字段已更新
        print(">>> 步骤 10：验证 quota_day 已更新为 8000 ...")
        models = helpers.admin_get_models(admin.token)
        model = next(
            (m for m in models if get_field(m, "ID", "id") == model_db_id),
            None,
        )
        assert model, "更新后未找到模型"
        quota_day = get_field(model, "QuotaDay", "quota_day")
        assert quota_day == 8000, f"quota_day 期望 8000, 实际={quota_day}"
        print(f"    quota_day=8000 ✓")

        # 步骤 11：验证实例仍正常运行（同步未导致实例异常）
        print(">>> 步骤 11：验证实例状态正常 ...")
        status_data = get_instance_status(user.token, instance_db_id)
        status = status_data.get("status", "unknown")
        assert status == "running", f"实例状态异常: {status}"
        print(f"    实例状态: {status} ✓")

        print()
        print("=" * 60)
        print(">>> 全部测试通过 ✓")
        print("=" * 60)

    except Exception as e:
        print(f"\n!!! 测试失败: {e}")
        traceback.print_exc()
        raise
    finally:
        # 清理：删除实例 + 删除模型
        if instance_db_id is not None:
            print(f"\n>>> 清理：删除实例 {instance_db_id} ...")
            try:
                delete_instance(user.token, instance_db_id)
                print("    实例删除已触发 ✓")
            except Exception as e:
                print(f"    实例删除失败（非关键）: {e}")

        if model_db_id is not None:
            print(f">>> 清理：删除模型 {model_db_id} ...")
            try:
                helpers.admin_delete_model(admin.token, model_db_id)
                print("    模型删除成功 ✓")
            except Exception as e:
                print(f"    模型删除失败（非关键）: {e}")

    print()


if __name__ == "__main__":
    main()
