#!/usr/bin/env python3
"""
TC-2.3 添加自定义模型（ai_model_id=0）

验证用户自行输入模型配置，ai_model_id=0 表示自定义。

使用方式：
  export BASE_URL=http://134.175.254.166
  export SEED_ADMIN_TOKEN=xxx
  python3 test_custom_model.py
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
    wait_gateway_ready,
)


def main():
    check_env()
    require_model_config()
    print()

    admin = setup_admin("custom-model")
    user = None
    inst = None

    try:
        helpers.ensure_gateway_ui_enabled(admin.token)

        user = setup_user(admin.token, "custom-model")
        inst = setup_instance(user.token, "custom-model")

        # 清理默认绑定的模型，确保自定义模型添加后为 primary
        helpers.clear_instance_models(user.token, inst.db_id)

        # 步骤 0：确认自定义模型开关已开启
        # 该用例需要两个层级的开关都打开：
        #   (a) site_config.user_config_model_enabled —— "允许用户查看与配置模型"
        #       这是更上层的总开关，默认 true，但若被运维关闭则需先翻开。
        #   (b) ai_models.hatchery/custom 占位记录 Enabled+Visible —— "自定义模型功能开启"
        #       这才是 /openclaw/add-model (ai_model_id=0) 实际依赖的开关，
        #       占位记录默认 false/false，必须主动翻开。
        # 早期实现只翻 (a)，导致此用例在干净环境下被 customModel() 用 (b) 拦在 403。
        print(">>> 步骤 0：确认自定义模型开关已开启 ...")
        site_config = helpers.admin_get_config(admin.token)
        if not site_config.get("user_config_model_enabled"):
            helpers.admin_update_config(admin.token, user_config_model_enabled="true")
        helpers.ensure_custom_model_flag(admin.token)
        print("    开关已开启 ✓")

        # 步骤 1：添加自定义模型
        print(">>> 步骤 1：添加自定义模型（ai_model_id=0）...")
        resp = retry_on_gateway_restart(
            lambda: helpers.user_add_model(
                user.token, inst.db_id, 0,
                model_id="my-custom-gpt",
                model_name="My Custom GPT",
                api_key=config.MODEL_API_KEY,
                url=config.MODEL_URL,
                model_type=config.MODEL_TYPE,
            )
        )
        assert resp.status_code == 200, f"添加自定义模型失败: {resp.status_code} {resp.text}"
        data = resp.json()
        assert data.get("ok"), f"添加自定义模型失败: {data}"
        print("    添加成功 ✓")
        time.sleep(5)

        # 步骤 2：查询实例模型列表
        print(">>> 步骤 2：查询实例模型列表 ...")
        im_data = helpers.user_get_instance_models(user.token, inst.db_id)
        models_list = im_data.get("models", im_data.get("data", []))
        assert len(models_list) >= 1, f"模型列表应非空: {models_list}"
        print(f"    模型列表共 {len(models_list)} 条")

        custom = next(
            (m for m in models_list
             if get_field(m, "model_id", "ModelID") == "my-custom-gpt"),
            None,
        )
        assert custom, f"未找到自定义模型 my-custom-gpt: {[m.get('model_id') for m in models_list]}"
        custom_role = get_field(custom, "role", "Role")
        assert custom_role == "primary", (
            f"清理默认模型后，自定义模型应为 primary，实际: {custom_role}"
        )
        print(f"    自定义模型已绑定 ✓  role={custom_role}")

        # 步骤 3：验证自定义模型可用性（自定义模型绑定后 agent 重启较慢，加大超时）
        print(">>> 步骤 3：验证自定义模型可用性 ...")
        wait_gateway_ready(user.token, inst.db_id, timeout=60, poll_interval=5)
        verify_model_available(inst, timeout=180)
        print("    自定义模型可用性验证通过 ✓")

        # 步骤 3b：验证 Gateway 模型配置
        print(">>> 步骤 3b：验证 Gateway 模型配置 ...")
        # 清理默认模型后只有自定义模型（primary），fallback 应为 0
        verify_model_config_via_inst(inst, expected_fallback_count=0, timeout=180)
        print("    模型配置验证通过 ✓")

        # 步骤 4：删除自定义模型
        print(">>> 步骤 4：删除自定义模型 ...")
        im_id = get_field(custom, "instance_model_id", "InstanceModelID", "id", "ID")
        data = retry_on_gateway_restart(
            lambda: helpers.user_del_model(user.token, inst.db_id, im_id)
        )
        assert data.get("ok"), f"删除自定义模型失败: {data}"
        print("    删除成功 ✓")
        time.sleep(5)

        print()
        print("TC-2.3 测试通过 ✅")

    except Exception as e:
        print(f"\nTC-2.3 测试失败 ❌: {e}")
        traceback.print_exc()
        sys.exit(1)

    finally:
        pass


if __name__ == "__main__":
    main()
