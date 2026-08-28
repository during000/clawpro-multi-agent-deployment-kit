#!/usr/bin/env python3
"""
TC-2.2 多模型 Fallback + 切换主模型 + 删除自动提升

验证多模型 Fallback 机制：添加多模型 → 切换主模型 → 删除 primary 自动提升。

使用方式：
  export BASE_URL=http://134.175.254.166
  export SEED_ADMIN_TOKEN=xxx
  python3 test_fallback_switch.py
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
    check_env, require_model_config_multi, setup_admin,
    setup_user,
    setup_instance,
    setup_model, teardown_model,
    wait_gateway_ready,
)


def main():
    check_env()
    require_model_config_multi(3)
    print()

    admin = setup_admin("fallback")
    user = None
    inst = None
    model_a = None
    model_b = None
    model_c = None

    try:
        helpers.ensure_gateway_ui_enabled(admin.token)

        # Setup 3 个模型（使用 config 中的模型配置）
        model_a = setup_model(admin.token, config.MODEL_ID, f"IntTest FB-A ({config.MODEL_ID})",
                              api_key=config.MODEL_API_KEY, url=config.MODEL_URL)
        model_b = setup_model(admin.token, config.MODEL_ID_2, f"IntTest FB-B ({config.MODEL_ID_2})",
                              api_key=config.MODEL_API_KEY, url=config.MODEL_URL)
        model_c = setup_model(admin.token, config.MODEL_ID_3, f"IntTest FB-C ({config.MODEL_ID_3})",
                              api_key=config.MODEL_API_KEY, url=config.MODEL_URL)

        user = setup_user(admin.token, "fallback")
        inst = setup_instance(user.token, "fallback")

        # 清理默认绑定的模型，排除默认模型配置的干扰
        helpers.clear_instance_models(user.token, inst.db_id)

        # 步骤 1：查询可用模型（按 model_name 断言，避免并发时同 MODEL_ID 干扰）
        print(">>> 步骤 1：查询可用模型列表 ...")
        user_models = helpers.user_get_models(user.token)
        user_model_names = [get_field(m, "ModelName", "model_name") for m in user_models]
        expected_names = {
            config.MODEL_ID: f"IntTest FB-A ({config.MODEL_ID})",
            config.MODEL_ID_2: f"IntTest FB-B ({config.MODEL_ID_2})",
            config.MODEL_ID_3: f"IntTest FB-C ({config.MODEL_ID_3})",
        }
        for mid, mname in expected_names.items():
            assert mname in user_model_names, f"模型 {mid} ({mname}) 不可见: {user_model_names}"
        print("    三个模型均可见 ✓")

        # 构建 model_ref → model_name 映射，用于日志可读性
        model_name_map = {
            f"hatchery-{config.MODEL_ID}/{config.MODEL_ID}": f"IntTest FB-A ({config.MODEL_ID})",
            f"hatchery-{config.MODEL_ID_2}/{config.MODEL_ID_2}": f"IntTest FB-B ({config.MODEL_ID_2})",
            f"hatchery-{config.MODEL_ID_3}/{config.MODEL_ID_3}": f"IntTest FB-C ({config.MODEL_ID_3})",
        }

        # 步骤 2：添加 A → primary
        print(">>> 步骤 2：添加模型 A → primary ...")
        resp = retry_on_gateway_restart(
            lambda: helpers.user_add_model(user.token, inst.db_id, model_a.db_id)
        )
        assert resp.status_code == 200, f"添加模型 A 失败: status={resp.status_code}, body={resp.text}"
        print("    A 添加成功 ✓")
        time.sleep(20)

        # 步骤 3：验证模型 A 可用性
        print(">>> 步骤 3：验证模型 A 可用性 ...")
        wait_gateway_ready(user.token, inst.db_id, timeout=60, poll_interval=5)
        verify_model_available(inst, timeout=180)
        print("    模型 A 可用性验证通过 ✓")

        # 步骤 3b：验证 Gateway 模型配置 — A=primary, fallback=0
        print(">>> 步骤 3b：验证 Gateway 模型配置（A 为 primary）...")
        verify_model_config_via_inst(inst, expected_fallback_count=0, model_name_map=model_name_map, timeout=180)
        print("    模型配置验证通过 ✓")
        time.sleep(20)

        # 步骤 4~5：添加 B, C → fallback
        print(">>> 步骤 4-5：添加模型 B、C → fallback ...")
        resp = retry_on_gateway_restart(
            lambda: helpers.user_add_model(user.token, inst.db_id, model_b.db_id)
        )
        assert resp.status_code == 200, f"添加模型 B 失败: {resp.status_code} {resp.text}"
        print("    B 添加成功 ✓")
        time.sleep(20)
        resp = retry_on_gateway_restart(
            lambda: helpers.user_add_model(user.token, inst.db_id, model_c.db_id)
        )
        assert resp.status_code == 200, f"添加模型 C 失败: {resp.status_code} {resp.text}"
        print("    C 添加成功 ✓")
        time.sleep(20)

        # 步骤 6：查询确认 3 条
        print(">>> 步骤 6：查询确认 3 条记录 ...")
        im_data = helpers.user_get_instance_models(user.token, inst.db_id)
        models_list = im_data.get("models", im_data.get("data", []))
        assert len(models_list) == 3, f"应有 3 条: {models_list}"
        print("    3 条记录 ✓")

        # 步骤 6b：验证 Gateway 模型配置 — A=primary, 2 个 fallback
        print(">>> 步骤 6b：验证 Gateway 模型配置（A=primary + 2 fallbacks）...")
        verify_model_config_via_inst(inst, expected_fallback_count=2, model_name_map=model_name_map, timeout=180)
        print("    模型配置验证通过 ✓")
        time.sleep(20)

        # 找到 B 的 instance_model_id
        b_record = next(
            (m for m in models_list
             if get_field(m, "ai_model_id", "AiModelID", "AIModelID") == model_b.db_id),
            None,
        )
        assert b_record, f"未找到模型 B 的绑定记录: {models_list}"
        b_im_id = get_field(b_record, "instance_model_id", "InstanceModelID", "id", "ID")

        # 步骤 7：将 B 提升为 primary
        print(">>> 步骤 7：将 B 提升为 primary ...")
        data = retry_on_gateway_restart(
            lambda: helpers.user_switch_primary_model(user.token, inst.db_id, b_im_id)
        )
        assert data.get("ok"), f"切换主模型失败: {data}"
        print("    切换成功 ✓")
        time.sleep(20)

        # 步骤 8：查询确认 B = primary
        print(">>> 步骤 8：查询确认 B = primary ...")
        im_data = helpers.user_get_instance_models(user.token, inst.db_id)
        models_list = im_data.get("models", im_data.get("data", []))
        b_record = next(
            (m for m in models_list
             if get_field(m, "ai_model_id", "AiModelID", "AIModelID") == model_b.db_id),
            None,
        )
        assert get_field(b_record, "role", "Role") == "primary", (
            f"B 应为 primary: {b_record}"
        )
        print("    B = primary ✓")

        # 步骤 9：验证模型 B 可用性（切换后）
        print(">>> 步骤 9：验证模型 B 可用性（切换后为主模型）...")
        verify_model_available(inst, timeout=180)
        print("    模型 B 可用性验证通过 ✓")

        # 步骤 9b：验证 Gateway 模型配置 — B=primary, 2 个 fallback (A + C)
        print(">>> 步骤 9b：验证 Gateway 模型配置（B=primary 切换后）...")
        verify_model_config_via_inst(inst, expected_fallback_count=2, model_name_map=model_name_map, timeout=180)
        print("    切换后模型配置验证通过 ✓")
        time.sleep(20)

        # 步骤 10：删除 primary B → 自动提升
        print(">>> 步骤 10：删除 primary B ...")
        data = retry_on_gateway_restart(
            lambda: helpers.user_del_model(user.token, inst.db_id, b_im_id)
        )
        assert data.get("ok"), f"删除模型 B 失败: {data}"
        print("    删除成功 ✓")
        time.sleep(20)

        # 步骤 11：查询确认自动提升
        print(">>> 步骤 11：查询确认自动提升 ...")
        im_data = helpers.user_get_instance_models(user.token, inst.db_id)
        models_list = im_data.get("models", im_data.get("data", []))
        assert len(models_list) == 2, f"应有 2 条: {models_list}"

        primaries = [
            m for m in models_list
            if get_field(m, "role", "Role") == "primary"
        ]
        assert len(primaries) == 1, f"应有恰好 1 个 primary: {models_list}"
        print("    自动提升 1 个 primary ✓")

        # 步骤 12：验证自动提升后的主模型可用性
        print(">>> 步骤 12：验证自动提升后的主模型可用性 ...")
        verify_model_available(inst, timeout=180)
        print("    自动提升后模型可用性验证通过 ✓")

        # 步骤 12b：验证 Gateway 模型配置 — 自动提升后 primary 存在, 1 个 fallback
        print(">>> 步骤 12b：验证 Gateway 模型配置（自动提升后）...")
        verify_model_config_via_inst(inst, expected_fallback_count=1, model_name_map=model_name_map, timeout=180)
        print("    自动提升后模型配置验证通过 ✓")

        print()
        print("TC-2.2 测试通过 ✅")

    except Exception as e:
        print(f"\nTC-2.2 测试失败 ❌: {e}")
        traceback.print_exc()
        sys.exit(1)

    finally:
        for model in [model_a, model_b, model_c]:
            if model:
                try:
                    teardown_model(admin.token, model)
                except Exception:
                    pass


if __name__ == "__main__":
    main()
