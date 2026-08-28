#!/usr/bin/env python3
"""
TC-H4.2 Hermes 飞书通道配置 + 查询确认

验证 hermes 实例的飞书通道凭证配置、查询确认、删除完整流程。
注：Hermes 不走 WebSocket，无法通过 OpenClawGateway 验证消息投递，
仅验证通道配置 API 的正确性。

使用方式：
  export API=http://134.175.254.166
  export ADMIN_TOKEN=xxx
  export MODEL_ID=xxx  MODEL_API_KEY=xxx  MODEL_URL=xxx  MODEL_TYPE=xxx
  export FEISHU_APP_ID=xxx  FEISHU_APP_SECRET=xxx
  python3 test_hermes_feishu_channel.py
"""

import os
import sys
import time
import traceback

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers import config
import helpers
from helpers import retry_on_gateway_restart
from helpers import (
    check_env, require_model_config, setup_admin,
    setup_user,
    setup_hermes_instance, verify_hermes_service,
    expect_hermes_channel_connected,
    setup_model, teardown_model,
)


def main():
    check_env()

    if not config.FEISHU_APP_ID or not config.FEISHU_APP_SECRET:
        print("错误: 未设置 FEISHU_APP_ID / FEISHU_APP_SECRET 环境变量")
        print("  通道测试需要真实凭证，不支持跳过。")
        sys.exit(1)

    require_model_config()
    print()
    admin = setup_admin("hermes-ch-feishu")
    user = None
    inst = None
    model_ctx = None

    try:
        helpers.ensure_gateway_ui_enabled(admin.token)

        # Setup：创建模型 + 用户 + hermes 实例 + 绑定模型
        model_ctx = setup_model(
            admin.token, model_id=config.MODEL_ID,
            model_name=f"IntTest Hermes CH Feishu ({config.MODEL_ID})",
            api_key=config.MODEL_API_KEY, url=config.MODEL_URL,
        )
        user = setup_user(admin.token, "hermes-ch-feishu")
        inst = setup_hermes_instance(user.token, "ch-feishu")

        # 用 /openclaw/set-model（维护 primary 单模型）而非 /openclaw/add-model：
        # channel 测试 setup 只需要"实例上有一个可用模型"，无需多模型 fallback 语义。
        print(">>> Setup：为实例绑定有效模型 ...")
        resp = retry_on_gateway_restart(
            lambda: helpers.user_set_model(user.token, inst.db_id, model_ctx.db_id)
        )
        assert resp.status_code == 200, f"绑定模型失败: {resp.status_code} {resp.text}"
        print("    模型绑定成功 ✓")
        time.sleep(5)

        # 步骤 1：配置飞书通道
        print(">>> 步骤 1：配置飞书通道 ...")
        resp = retry_on_gateway_restart(
            lambda: helpers.user_set_channel(
                user.token, inst.db_id, "feishu",
                keys=["app_id", "app_secret"],
                values=[config.FEISHU_APP_ID, config.FEISHU_APP_SECRET],
            )
        )
        assert resp.status_code == 200, f"配置飞书失败: {resp.status_code} {resp.text}"
        print("    配置成功 ✓")
        time.sleep(10)

        # 步骤 2：查询确认
        print(">>> 步骤 2：查询确认 ...")
        inst_data = helpers.user_get_channels(user.token, instance_db_id=inst.db_id)
        channels = inst_data.get("channels", {})
        assert channels, "通道配置不应为空"
        # 确认飞书通道存在
        assert "feishu" in channels or any(
            c.get("channel_id", "") == "feishu"
            for c in (channels if isinstance(channels, list) else [])
        ), f"飞书通道应存在于配置中: {channels}"
        print("    查询确认 ✓")

        # 步骤 3：验证 hermes 服务仍然可用（配置通道后服务不应崩溃）
        print(">>> 步骤 3：验证 hermes 服务可用性 ...")
        verify_hermes_service(user.token, inst.db_id)
        print("    Hermes 服务可用 ✓")

        # 步骤 3b：软检查 hermes-gateway 是否真的把 feishu 通道拉起来了
        # 需依赖 check_service_hermes.sh 的 channelSummary 字段；老版本会软跳过
        print(">>> 步骤 3b：检查 hermes channel 连接状态 ...")
        expect_hermes_channel_connected(user.token, inst.db_id, min_enabled=1, timeout=90)

        # 步骤 4：删除配置
        print(">>> 步骤 4：删除飞书通道配置 ...")
        resp = retry_on_gateway_restart(
            lambda: helpers.user_del_channel(user.token, inst.db_id, "feishu")
        )
        assert resp.status_code == 200, f"删除 feishu 失败: status={resp.status_code}, body={resp.text}"
        print("    删除成功 ✓")
        time.sleep(5)

        # 步骤 5：查询确认已删除（轮询确认通道从列表移除）
        print(">>> 步骤 5：查询确认已删除 ...")
        still_present = True
        for attempt in range(6):
            inst_data = helpers.user_get_channels(user.token, instance_db_id=inst.db_id)
            channels = inst_data.get("channels", {})
            if isinstance(channels, dict):
                still_present = "feishu" in channels and (
                    channels["feishu"].get("enabled", False)
                    if isinstance(channels.get("feishu"), dict)
                    else True
                )
            else:
                still_present = "feishu" in str(channels)
            if not still_present:
                break
            time.sleep(5)
        assert not still_present, f"删除 feishu 后仍存在于通道列表中: {channels}"
        print("    feishu 已从列表移除 ✓")

        print()
        print("TC-H4.2 测试通过 ✅")

    except Exception as e:
        print(f"\nTC-H4.2 测试失败 ❌: {e}")
        traceback.print_exc()
        sys.exit(1)

    finally:
        if model_ctx:
            teardown_model(admin.token, model_ctx)


if __name__ == "__main__":
    main()
