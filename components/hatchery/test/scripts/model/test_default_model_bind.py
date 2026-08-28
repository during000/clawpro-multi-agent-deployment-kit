#!/usr/bin/env python3
"""
TC-1.2 默认模型开关 + 自动绑定验证

验证管理员设置/取消默认模型后，新创建实例的模型绑定行为：
1）设置默认模型开关 → 验证 default_model_id 生效
2）开启默认模型 → 创建实例 → 自动绑定 primary 模型
3）关闭默认模型 → 创建实例 → 模型列表为空

使用方式：
  export BASE_URL=http://134.175.254.166
  export SEED_ADMIN_TOKEN=xxx
  python3 test_default_model_bind.py
"""

import os
import sys
import time
import traceback

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

import helpers
from helpers import config
from helpers import verify_model_available, verify_model_config_via_inst
from helpers import get_field
from helpers import (
    check_env, require_model_config, setup_admin,
    setup_user,
    setup_instance,
    wait_gateway_ready,
)


def wait_default_model_stable(admin_token, expected_model_id, model_db_id, timeout=30, interval=3):
    """
    轮询验证 default_model_id 稳定为 expected_model_id。

    唯一需要预防的并发场景：其他用例在我们操作之后，用 GORM Save 把 site_config
    的老快照（含老的 default_model_id）写回去，导致我们的设置被覆盖。
    - 设置默认模型后：老快照中 default_model_id=0，被覆盖回 0
    - 取消默认模型后：老快照中 default_model_id=model_db_id，被覆盖回 model_db_id

    策略：检测到被覆盖就重新 toggle，直到连续两次查询都稳定。
    """
    is_zero = expected_model_id in (0, None)

    def matches(val):
        if is_zero:
            return val in (0, None, "", "0")
        return val == expected_model_id

    start = time.time()
    while True:
        actual = helpers.admin_get_default_model_id(admin_token)
        if matches(actual):
            # 确认后再等一轮复查，防止刚设置/清零就被并发覆盖
            time.sleep(interval)
            actual = helpers.admin_get_default_model_id(admin_token)
            if matches(actual):
                return actual
        elapsed = int(time.time() - start)
        if elapsed >= timeout:
            return actual
        # 被并发覆盖了，重新 toggle 修正
        # 设置场景：当前被覆盖为 0，toggle 会设回 model_db_id
        # 取消场景：当前被覆盖为 model_db_id，toggle 会清零
        print(f"    [{elapsed}s] default_model_id={actual!r}，"
              f"期望 {'0' if is_zero else expected_model_id}，被并发覆盖，重新 toggle ...")
        helpers.admin_toggle_default_model(admin_token, model_db_id)
        time.sleep(interval)


def main():
    check_env()
    require_model_config()
    print()

    admin = setup_admin("default-model")
    model_db_id = None
    user = None
    inst_with_default = None
    inst_without_default = None

    model_name = f"IntTest Default Bind ({config.MODEL_ID})"

    try:
        helpers.ensure_gateway_ui_enabled(admin.token)

        # ════════════════════════════════════════════
        # 步骤 1~4：创建模型 → 启用 → 设为默认 → 验证
        # ════════════════════════════════════════════
        print(">>> 步骤 1：创建模型 ...")
        data = helpers.admin_create_model(
            admin.token,
            model_id=config.MODEL_ID,
            model_name=model_name,
        )
        assert data.get("ok"), f"创建模型失败: {data}"

        models = helpers.admin_get_models(admin.token)
        model = next(
            (m for m in models if get_field(m, "ModelName", "model_name") == model_name),
            None,
        )
        assert model, f"模型 {model_name} 未找到"
        model_db_id = get_field(model, "ID", "id")
        print(f"    模型创建成功 ✓  db_id={model_db_id}")

        print(">>> 步骤 2：启用模型 ...")
        helpers.admin_toggle_model(admin.token, model_db_id)
        print("    启用成功 ✓")

        print(">>> 步骤 3：设为默认模型 ...")
        helpers.admin_toggle_default_model(admin.token, model_db_id)
        print("    设为默认 ✓")

        # 步骤 4：验证 default_model_id 稳定生效
        print(">>> 步骤 4：验证 default_model_id 已设置 ...")
        actual_default = wait_default_model_stable(admin.token, model_db_id, model_db_id)
        assert actual_default == model_db_id, (
            f"default_model_id 无法稳定为 {model_db_id}，"
            f"实际 {actual_default}（可能被并发用例覆盖）"
        )
        print(f"    default_model_id={actual_default} ✓")

        # ════════════════════════════════════════════
        # 步骤 5~8：开启默认模型 → 创建实例 → 验证自动绑定
        # ════════════════════════════════════════════
        user = setup_user(admin.token, "default-model")
        inst_with_default = setup_instance(user.token, "default-model")

        print(">>> 步骤 5：查询实例模型列表（有默认模型，等待异步注入完成）...")
        # 默认模型注入是异步 goroutine（每 10s 轮询 agent_ready → 执行 TAT 脚本），
        # 需要等待一段时间才能在 instance_models 中看到记录。
        poll_timeout = 120
        poll_interval = 5
        poll_start = time.time()
        models_list = []
        while time.time() - poll_start < poll_timeout:
            im_data = helpers.user_get_instance_models(user.token, inst_with_default.db_id)
            models_list = im_data.get("models", im_data.get("data", []))
            if models_list:
                break
            elapsed = int(time.time() - poll_start)
            print(f"    [{elapsed}s] 模型列表仍为空，等待异步注入 ...", flush=True)
            time.sleep(poll_interval)
        assert models_list, (
            f"实例模型列表在 {poll_timeout}s 内仍为空，"
            f"默认模型异步注入可能失败（default_model_id={model_db_id}）"
        )
        print(f"    模型列表非空 ✓  共 {len(models_list)} 条（等待 {int(time.time() - poll_start)}s）")

        print(">>> 步骤 6：断言自动绑定为 primary ...")
        primary = [
            m for m in models_list
            if get_field(m, "role", "Role") == "primary"
        ]
        assert primary, f"未找到 primary 模型: {models_list}"

        primary_model = primary[0]
        bound_model_id = get_field(primary_model, "ai_model_id", "AiModelID", "AIModelID")
        assert bound_model_id == model_db_id, (
            f"自动绑定的模型 ID 不匹配: 期望 {model_db_id}，实际 {bound_model_id}"
        )
        print("    自动绑定验证通过 ✓")

        print(">>> 步骤 7：验证模型可用性（检查 agent 服务就绪）...")
        wait_gateway_ready(user.token, inst_with_default.db_id, timeout=60, poll_interval=5)
        verify_model_available(inst_with_default, timeout=180)
        print("    模型可用性验证通过 ✓")

        print(">>> 步骤 8：验证 Gateway 模型配置（primary 已绑定）...")
        verify_model_config_via_inst(inst_with_default, timeout=180)
        print("    模型配置验证通过 ✓")

        # ════════════════════════════════════════════
        # 步骤 9~11：关闭默认模型 → 创建实例 → 验证模型列表为空
        # ════════════════════════════════════════════
        print(">>> 步骤 9：取消默认模型 ...")
        helpers.admin_toggle_default_model(admin.token, model_db_id)
        print("    取消默认 ✓")

        # 步骤 10：验证 default_model_id 已清零（复用公共函数）
        print(">>> 步骤 10：验证 default_model_id == 0 ...")
        actual_default = wait_default_model_stable(admin.token, 0, model_db_id)
        assert actual_default in (0, None, "", "0"), (
            f"default_model_id 无法稳定为 0，"
            f"实际 {actual_default!r}（可能被并发用例覆盖）"
        )
        print(f"    确认无默认模型 ✓  (default_model_id={actual_default!r})")

        inst_without_default = setup_instance(user.token, "no-default")

        print(">>> 步骤 11：查询实例模型列表（无默认模型）...")
        # 短暂等待确保没有异步注入触发
        time.sleep(5)
        im_data = helpers.user_get_instance_models(user.token, inst_without_default.db_id)
        models_list = im_data.get("models", im_data.get("data", []))
        assert len(models_list) == 0, (
            f"实例模型列表应为空，实际: {models_list}"
        )
        print("    模型列表为空 ✓")

        print()
        print("TC-1.2 测试通过 ✅")

    except Exception as e:
        print(f"\nTC-1.2 测试失败 ❌: {e}")
        traceback.print_exc()
        sys.exit(1)

    finally:
        # 清理：取消默认模型 + 删除模型
        if model_db_id:
            try:
                # 确保取消默认（如果当前仍是默认的话）
                actual = helpers.admin_get_default_model_id(admin.token)
                if actual == model_db_id:
                    helpers.admin_toggle_default_model(admin.token, model_db_id)
            except Exception:
                pass
            try:
                helpers.admin_delete_model(admin.token, model_db_id)
            except Exception:
                pass


if __name__ == "__main__":
    main()
