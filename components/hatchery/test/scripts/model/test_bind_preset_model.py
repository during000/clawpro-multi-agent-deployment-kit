#!/usr/bin/env python3
"""
TC-2.1 绑定预配置模型（add-model）

验证用户为实例绑定系统预配置模型，及重复绑定的冲突处理。

使用方式：
  export BASE_URL=http://134.175.254.166
  export SEED_ADMIN_TOKEN=xxx
  python3 test_bind_preset_model.py
"""

import os
import sys
import time
import traceback

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers import config
import helpers
from helpers import verify_model_available, verify_model_config_via_inst
from helpers import retry_on_gateway_restart, get_field
from helpers import (
    check_env, require_model_config, setup_admin,
    setup_user,
    setup_instance,
    setup_model, teardown_model,
    wait_gateway_ready,
)


def main():
    check_env()
    require_model_config()
    print()

    admin = setup_admin("bind-model")
    user = None
    inst = None
    model_alpha = None
    model_beta = None

    try:
        helpers.ensure_gateway_ui_enabled(admin.token)

        # Setup 模型
        model_alpha = setup_model(
            admin.token, model_id=config.MODEL_ID, model_name=f"IntTest Bind Alpha ({config.MODEL_ID})",
            api_key=config.MODEL_API_KEY, url=config.MODEL_URL,
        )
        model_beta = setup_model(
            admin.token, model_id=config.MODEL_ID_2, model_name=f"IntTest Bind Beta ({config.MODEL_ID_2})",
            api_key=config.MODEL_API_KEY, url=config.MODEL_URL,
        )

        # Setup 用户 + 实例
        user = setup_user(admin.token, "bind-model")
        inst = setup_instance(user.token, "bind-model")

        # 清理默认绑定的模型，排除默认模型配置的干扰
        helpers.clear_instance_models(user.token, inst.db_id)

        # 步骤 1：查询可用模型（按 model_name 断言，避免并发时同 MODEL_ID 干扰）
        print(">>> 步骤 1：查询可用模型列表 ...")
        user_models = helpers.user_get_models(user.token)
        user_model_names = [get_field(m, "ModelName", "model_name") for m in user_models]
        alpha_name = f"IntTest Bind Alpha ({config.MODEL_ID})"
        beta_name = f"IntTest Bind Beta ({config.MODEL_ID_2})"
        assert alpha_name in user_model_names, f"模型 alpha 不可见: {user_model_names}"
        assert beta_name in user_model_names, f"模型 beta 不可见: {user_model_names}"
        print("    两个模型均可见 ✓")

        # 步骤 2：添加模型 α → primary
        print(">>> 步骤 2：添加模型 α ...")
        resp = retry_on_gateway_restart(
            lambda: helpers.user_add_model(user.token, inst.db_id, model_alpha.db_id)
        )
        assert resp.status_code == 200, f"添加模型 α 失败: {resp.status_code} {resp.text}"
        data = resp.json()
        assert data.get("ok"), f"添加模型 α 失败: {data}"
        print("    添加成功 ✓")
        time.sleep(5)

        # 构建 model_ref → model_name 映射，用于日志可读性
        model_name_map = {
            f"hatchery-{config.MODEL_ID}/{config.MODEL_ID}": f"IntTest Bind Alpha ({config.MODEL_ID})",
            f"hatchery-{config.MODEL_ID_2}/{config.MODEL_ID_2}": f"IntTest Bind Beta ({config.MODEL_ID_2})",
        }

        # 步骤 3：查询实例模型列表 → α 已绑定且为 primary（默认模型已被清理）
        print(">>> 步骤 3：查询实例模型列表 ...")
        im_data = helpers.user_get_instance_models(user.token, inst.db_id)
        models_list = im_data.get("models", im_data.get("data", []))
        assert len(models_list) >= 1, f"模型列表应非空: {models_list}"

        alpha_record = next(
            (m for m in models_list
             if get_field(m, "ai_model_id", "AiModelID", "AIModelID") == model_alpha.db_id),
            None,
        )
        assert alpha_record, f"未找到模型 α 绑定记录: {models_list}"
        alpha_role = get_field(alpha_record, "role", "Role")
        assert alpha_role == "primary", (
            f"清理默认模型后，α 应为 primary，实际: {alpha_role}, record={alpha_record}"
        )
        print("    α = primary ✓")

        # 步骤 4：验证模型 α 可用性
        print(">>> 步骤 4：验证模型 α 可用性（检查 agent 服务就绪）...")
        wait_gateway_ready(user.token, inst.db_id, timeout=60, poll_interval=5)
        verify_model_available(inst, timeout=180)
        print("    模型 α 可用性验证通过 ✓")

        # 步骤 4b：验证 Gateway 模型配置（α=primary, 无 fallback）
        print(">>> 步骤 4b：验证 Gateway 模型配置 ...")
        verify_model_config_via_inst(inst, expected_fallback_count=0, model_name_map=model_name_map, timeout=180)
        print("    模型配置验证通过 ✓")

        # 步骤 5：重复添加 → 409
        print(">>> 步骤 5：重复添加同一模型 → 409 ...")
        resp = helpers.user_add_model(user.token, inst.db_id, model_alpha.db_id)
        assert resp.status_code == 409, (
            f"重复添加应返回 409，实际: {resp.status_code} {resp.text}"
        )
        print("    重复添加返回 409 ✓")

        # 步骤 6：添加模型 β → fallback
        print(">>> 步骤 6：添加模型 β ...")
        resp = retry_on_gateway_restart(
            lambda: helpers.user_add_model(user.token, inst.db_id, model_beta.db_id)
        )
        assert resp.status_code == 200, f"添加模型 β 失败: {resp.status_code} {resp.text}"
        print("    添加成功 ✓")
        time.sleep(5)

        # 步骤 7：查询确认 α 和 β 都已绑定，β=fallback
        # 并发环境下可能有默认模型，所以不硬性要求刚好 2 条
        print(">>> 步骤 7：查询确认角色分配 ...")
        im_data = helpers.user_get_instance_models(user.token, inst.db_id)
        models_list = im_data.get("models", im_data.get("data", []))
        # 至少包含 α 和 β
        our_models = [
            m for m in models_list
            if get_field(m, "ai_model_id", "AiModelID", "AIModelID")
            in (model_alpha.db_id, model_beta.db_id)
        ]
        assert len(our_models) == 2, (
            f"应找到 α 和 β 共 2 条绑定记录，实际找到 {len(our_models)}: {models_list}"
        )

        beta_record = next(
            (m for m in models_list
             if get_field(m, "ai_model_id", "AiModelID", "AIModelID") == model_beta.db_id),
            None,
        )
        assert beta_record, f"未找到模型 β 绑定记录: {models_list}"
        assert get_field(beta_record, "role", "Role") == "fallback", (
            f"模型 β 应为 fallback: {beta_record}"
        )
        print("    α 和 β 均已绑定，β=fallback ✓")

        # 步骤 7b：验证 Gateway 模型配置 — α=primary, β=fallback
        print(">>> 步骤 7b：验证 Gateway 模型配置 ...")
        verify_model_config_via_inst(inst, expected_fallback_count=1, model_name_map=model_name_map, timeout=180)
        print("    模型配置验证通过 ✓")

        print()
        print("TC-2.1 测试通过 ✅")

    except Exception as e:
        print(f"\nTC-2.1 测试失败 ❌: {e}")
        traceback.print_exc()
        sys.exit(1)

    finally:
        if model_alpha:
            teardown_model(admin.token, model_alpha)
        if model_beta:
            teardown_model(admin.token, model_beta)


if __name__ == "__main__":
    main()
