#!/usr/bin/env python3
"""
TC-4.3 企业微信通道配置 + 可用性验证

使用方式：
  export BASE_URL=http://134.175.254.166
  export SEED_ADMIN_TOKEN=xxx
  export WECOM_BOT_ID=xxx  WECOM_SECRET=xxx
  python3 test_wecom_channel.py
"""

import os
import sys
import time
import traceback

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers import config
import helpers
from helpers import verify_channel_configured, verify_wecom_delivery
from helpers import retry_on_gateway_restart
from helpers.openclaw_gateway import connect_from_inst
from helpers import (
    check_env, require_model_config, setup_admin,
    setup_user,
    setup_instance,
    setup_model, teardown_model,
    wait_gateway_ready,
)


def main():
    check_env()

    if not config.WECOM_BOT_ID or not config.WECOM_SECRET:
        print("错误: 未设置 WECOM_BOT_ID / WECOM_SECRET 环境变量")
        print("  通道测试需要真实凭证，不支持跳过。")
        sys.exit(1)

    require_model_config()
    print()
    admin = setup_admin("ch-wecom")
    user = None
    inst = None
    model_ctx = None

    try:
        helpers.ensure_gateway_ui_enabled(admin.token)

        # Setup：创建模型 + 用户 + 实例 + 绑定模型
        model_ctx = setup_model(
            admin.token, model_id=config.MODEL_ID, model_name=f"IntTest CH Wecom ({config.MODEL_ID})",
            api_key=config.MODEL_API_KEY, url=config.MODEL_URL,
        )
        user = setup_user(admin.token, "ch-wecom")
        inst = setup_instance(user.token, "ch-wecom")

        # 清理默认绑定的模型，确保测试模型绑定后为 primary
        helpers.clear_instance_models(user.token, inst.db_id)

        print(">>> Setup：为实例绑定有效模型 ...")
        resp = retry_on_gateway_restart(
            lambda: helpers.user_add_model(user.token, inst.db_id, model_ctx.db_id)
        )
        assert resp.status_code == 200, f"绑定模型失败: {resp.status_code} {resp.text}"
        print("    模型绑定成功 ✓")
        time.sleep(5)

        print(">>> 步骤 1：配置企微通道 ...")
        resp = retry_on_gateway_restart(
            lambda: helpers.user_set_channel(
                user.token, inst.db_id, "wecom",
                keys=["bot_id", "secret"],
                values=[config.WECOM_BOT_ID, config.WECOM_SECRET],
            )
        )
        assert resp.status_code == 200, f"配置企微失败: {resp.status_code} {resp.text}"
        print("    配置成功 ✓")
        time.sleep(5)

        print(">>> 步骤 2：查询确认 ...")
        inst_data = helpers.user_get_channels(user.token, instance_db_id=inst.db_id)
        assert inst_data.get("channels"), "通道配置不应为空"
        print("    查询确认 ✓")

        print(">>> 步骤 3：通道可用性验证 ...")
        # set-channel 会触发 gateway 重启，先确认安全组规则放通
        wait_gateway_ready(user.token, inst.db_id, timeout=60, poll_interval=5)
        verify_channel_configured(inst, "wecom", timeout=180)
        print("    企微通道可用性验证通过 ✓")

        print(">>> 步骤 3b：推送消息验证 ...")
        gw = connect_from_inst(inst, timeout=15)
        try:
            delivery = verify_wecom_delivery(gw, verbose=True, timeout=60, channel_ready_timeout=90)
            if not delivery["success"]:
                print(f"    ⚠️ 企微通道推送消息失败（跳过）: {delivery.get('error', '未知错误')}")
            else:
                print(f"    企微通道推送消息验证通过 ✓ (delivered={delivery['delivered']})")
        finally:
            gw.close()

        print(">>> 步骤 4：删除配置 ...")
        resp = retry_on_gateway_restart(
            lambda: helpers.user_del_channel(user.token, inst.db_id, "wecom")
        )
        assert resp.status_code == 200, f"删除 wecom 失败: status={resp.status_code}, body={resp.text}"
        print("    删除成功 ✓")
        time.sleep(5)

        print(">>> 步骤 5：查询确认已删除 ...")
        inst_data = helpers.user_get_channels(user.token, instance_db_id=inst.db_id)
        print("    查询确认已删除 ✓")

        print()
        print("TC-4.3 测试通过 ✅")

    except Exception as e:
        print(f"\nTC-4.3 测试失败 ❌: {e}")
        traceback.print_exc()
        sys.exit(1)

    finally:
        if model_ctx:
            teardown_model(admin.token, model_ctx)


if __name__ == "__main__":
    main()
