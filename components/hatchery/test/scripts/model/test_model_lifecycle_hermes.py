#!/usr/bin/env python3
"""
TC-H1.1 Hermes 模型全生命周期：CRUD + 启用/禁用可见性联动

验证管理员创建自定义模型、启用、用户可见、更新属性、禁用→不可见、重新启用→可见、删除的完整流程。
与 openclaw 版本逻辑一致，但使用 hermes 实例验证用户侧可见性。

使用方式：
  export API=http://134.175.254.166
  export ADMIN_TOKEN=xxx
  export MODEL_ID=xxx  MODEL_API_KEY=xxx  MODEL_URL=xxx  MODEL_TYPE=xxx
  python3 test_hermes_model_lifecycle.py
"""

import os
import sys
import traceback

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers import config
import helpers
from helpers import (
    check_env, require_model_config, setup_admin,
    setup_user, get_field,
)


def main():
    check_env()
    require_model_config()
    print()

    # ── Setup ──
    admin = setup_admin("hermes-model-life")
    user = setup_user(admin.token, "hermes-model-life", instance_quota=0)
    model_db_id = None

    try:
        model_name = f"IntTest Hermes Lifecycle ({config.MODEL_ID})"

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

        # 步骤 2：查询模型列表，记录 model_db_id
        print(">>> 步骤 2：查询模型列表 ...")
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

        # 步骤 4：用户侧查询 → 模型可见
        print(">>> 步骤 4：验证用户可见 ...")
        user_models = helpers.user_get_models(user.token)
        user_model_names = [get_field(m, "ModelName", "model_name") for m in user_models]
        assert model_name in user_model_names, f"用户侧未看到模型，列表: {user_model_names}"
        print("    用户可见 ✓")

        # 步骤 5：修改每日 Token 上限
        print(">>> 步骤 5：更新 quota_day=10000 ...")
        data = helpers.admin_update_model(
            admin.token, model_db_id,
            model_id=config.MODEL_ID,
            model_name=model_name,
            url=config.MODEL_URL,
            model_type=config.MODEL_TYPE,
            quota_day="10000",
        )
        assert data.get("ok"), f"更新模型失败: {data}"
        print("    更新成功 ✓")

        # 步骤 6：确认 quota_day 已更新
        print(">>> 步骤 6：确认 quota_day ...")
        models = helpers.admin_get_models(admin.token)
        model = next(
            (m for m in models if get_field(m, "ID", "id") == model_db_id),
            None,
        )
        assert model, "更新后未找到模型"
        assert get_field(model, "QuotaDay", "quota_day") == 10000, (
            f"quota_day 期望 10000，实际 {model}"
        )
        print("    quota_day=10000 ✓")

        # 步骤 7：禁用模型 → 用户不可见
        print(">>> 步骤 7：禁用模型 ...")
        helpers.admin_toggle_model(admin.token, model_db_id)
        print("    禁用成功 ✓")

        print(">>> 步骤 8：验证用户不可见 ...")
        user_models = helpers.user_get_models(user.token)
        user_model_names = [get_field(m, "ModelName", "model_name") for m in user_models]
        assert model_name not in user_model_names, f"用户不应可见: {user_model_names}"
        print("    用户不可见 ✓")

        # 步骤 9：重新启用 → 用户可见
        print(">>> 步骤 9：重新启用模型 ...")
        helpers.admin_toggle_model(admin.token, model_db_id)
        print("    重新启用 ✓")

        print(">>> 步骤 10：验证用户重新可见 ...")
        user_models = helpers.user_get_models(user.token)
        user_model_names = [get_field(m, "ModelName", "model_name") for m in user_models]
        assert model_name in user_model_names, f"用户应重新可见: {user_model_names}"
        print("    用户重新可见 ✓")

        # 步骤 11：删除模型
        print(">>> 步骤 11：删除模型 ...")
        model_db_id_deleted = model_db_id
        data = helpers.admin_delete_model(admin.token, model_db_id)
        assert data.get("ok"), f"删除模型失败: {data}"
        model_db_id = None
        print("    删除成功 ✓")

        # 步骤 12：验证模型已删除
        print(">>> 步骤 12：验证模型已删除 ...")
        models = helpers.admin_get_models(admin.token)
        found = any(
            get_field(m, "ID", "id") == model_db_id_deleted
            for m in models
        )
        assert not found, f"模型 db_id={model_db_id_deleted} 仍然存在"
        print("    验证通过 ✓")

        print()
        print("TC-H1.1 测试通过 ✅")

    except Exception as e:
        print(f"\nTC-H1.1 测试失败 ❌: {e}")
        traceback.print_exc()
        sys.exit(1)

    finally:
        if model_db_id:
            helpers.admin_delete_model(admin.token, model_db_id)


if __name__ == "__main__":
    main()
