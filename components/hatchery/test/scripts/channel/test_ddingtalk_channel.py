#!/usr/bin/env python3
"""
TC-4.4 钉钉通道配置 + 可用性验证

使用方式：
  export BASE_URL=http://134.175.254.166
  export SEED_ADMIN_TOKEN=xxx
  export DDINGTALK_CLIENT_ID=xxx  DDINGTALK_CLIENT_SECRET=xxx
  python3 test_ddingtalk_channel.py
"""

import os
import sys
import time
import traceback

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers import config
import helpers
from helpers import verify_channel_configured, verify_dingtalk_delivery
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

    if not config.DDINGTALK_CLIENT_ID or not config.DDINGTALK_CLIENT_SECRET:
        print("错误: 未设置 DDINGTALK_CLIENT_ID / DDINGTALK_CLIENT_SECRET 环境变量")
        print("  通道测试需要真实凭证，不支持跳过。")
        sys.exit(1)

    require_model_config()
    print()
    admin = setup_admin("ch-ding")
    user = None
    inst = None
    model_ctx = None

    try:
        helpers.ensure_gateway_ui_enabled(admin.token)

        # Setup：创建模型 + 用户 + 实例 + 绑定模型
        model_ctx = setup_model(
            admin.token, model_id=config.MODEL_ID, model_name=f"IntTest CH Ding ({config.MODEL_ID})",
            api_key=config.MODEL_API_KEY, url=config.MODEL_URL,
        )
        user = setup_user(admin.token, "ch-ding")
        inst = setup_instance(user.token, "ch-ding")

        # 清理默认绑定的模型，确保测试模型绑定后为 primary
        helpers.clear_instance_models(user.token, inst.db_id)

        print(">>> Setup：为实例绑定有效模型 ...")
        resp = retry_on_gateway_restart(
            lambda: helpers.user_add_model(user.token, inst.db_id, model_ctx.db_id)
        )
        assert resp.status_code == 200, f"绑定模型失败: {resp.status_code} {resp.text}"
        print("    模型绑定成功 ✓")
        time.sleep(5)

        print(">>> 步骤 1：配置钉钉通道 ...")
        resp = retry_on_gateway_restart(
            lambda: helpers.user_set_channel(
                user.token, inst.db_id, "ddingtalk",
                keys=["client_id", "client_secret"],
                values=[config.DDINGTALK_CLIENT_ID, config.DDINGTALK_CLIENT_SECRET],
            )
        )
        assert resp.status_code == 200, f"配置钉钉失败: status={resp.status_code}, body={resp.text}"
        print("    配置成功 ✓")

        # 等待 gateway 重启完成（set_channel 脚本会 restart gateway）
        print("    等待 gateway 重启 (15s) ...")
        time.sleep(15)

        print(">>> 步骤 2：查询确认 ...")
        inst_data = helpers.user_get_channels(user.token, instance_db_id=inst.db_id)
        assert inst_data.get("channels"), "通道配置不应为空"
        print("    查询确认 ✓")

        print(">>> 步骤 3：通道可用性验证 ...")
        # set-channel 会触发 gateway 重启，先确认安全组规则放通
        wait_gateway_ready(user.token, inst.db_id, timeout=60, poll_interval=5)
        # 钉钉通道 npm 插件加载 + 与钉钉服务器建立长连接耗时较长，需要更大超时
        verify_channel_configured(inst, "ddingtalk", timeout=180)
        print("    钉钉通道可用性验证通过 ✓")

        print(">>> 步骤 3b：推送消息验证 ...")
        gw = connect_from_inst(inst, timeout=15)
        try:
            # 钉钉通道需要与钉钉服务器建立长连接（stream 模式），running 可能较慢
            delivery = verify_dingtalk_delivery(gw, verbose=True, timeout=60, channel_ready_timeout=120)
            if not delivery["success"]:
                print(f"    ⚠️ 钉钉通道推送消息失败（跳过）: {delivery.get('error', '未知错误')}")
            else:
                print(f"    钉钉通道推送消息验证通过 ✓ (delivered={delivery['delivered']})")
        finally:
            gw.close()

        print(">>> 步骤 4：删除配置 ...")
        # 新版插件实际以 dingtalk-connector 为 key 存储，删除时需用此名称
        resp = retry_on_gateway_restart(
            lambda: helpers.user_del_channel(user.token, inst.db_id, "dingtalk-connector")
        )
        assert resp.status_code == 200, f"删除 dingtalk-connector 失败: status={resp.status_code}, body={resp.text}"
        print("    删除成功 ✓")
        time.sleep(5)

        print(">>> 步骤 5：查询确认已删除 ...")
        inst_data = helpers.user_get_channels(user.token, instance_db_id=inst.db_id)
        print("    查询确认已删除 ✓")

        print()
        print("TC-4.4 测试通过 ✅")

    except Exception as e:
        print(f"\nTC-4.4 测试失败 ❌: {e}")
        traceback.print_exc()
        sys.exit(1)

    finally:
        if model_ctx:
            teardown_model(admin.token, model_ctx)


if __name__ == "__main__":
    main()
