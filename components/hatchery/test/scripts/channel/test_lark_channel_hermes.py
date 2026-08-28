#!/usr/bin/env python3
"""
TC-H4.7 Hermes Lark 通道配置 + 查询确认

验证 hermes 实例的 Lark 通道凭证配置、查询确认、删除完整流程。
注：Hermes 不走 WebSocket，无法通过 OpenClawGateway 验证消息投递，
仅验证通道配置 API 的正确性。

Lark 为 overseas-only 通道：非海外站点会自动跳过。
Lark 复用 feishu 的协议（app_id + app_secret），脚本层将 FEISHU_DOMAIN 设为 lark。

使用方式：
  export API=http://134.175.254.166
  export ADMIN_TOKEN=xxx
  export MODEL_ID=xxx  MODEL_API_KEY=xxx  MODEL_URL=xxx  MODEL_TYPE=xxx
  export LARK_APP_ID=xxx  LARK_APP_SECRET=xxx
  python3 test_lark_channel_hermes.py
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
    is_overseas_site,
)


def main():
    check_env()

    # Lark 为 overseas-only 通道，非海外站点跳过
    if not is_overseas_site():
        print("⚠ 当前站点非海外站点，Lark 通道不可见，跳过测试。")
        return

    if not config.LARK_APP_ID or not config.LARK_APP_SECRET:
        print("错误: 未设置 LARK_APP_ID / LARK_APP_SECRET 环境变量")
        print("  通道测试需要真实凭证，不支持跳过。")
        sys.exit(1)

    require_model_config()
    print()
    admin = setup_admin("hermes-ch-lark")
    user = None
    inst = None
    model_ctx = None

    try:
        helpers.ensure_gateway_ui_enabled(admin.token)

        # Setup：创建模型 + 用户 + hermes 实例 + 绑定模型
        model_ctx = setup_model(
            admin.token, model_id=config.MODEL_ID,
            model_name=f"IntTest Hermes CH Lark ({config.MODEL_ID})",
            api_key=config.MODEL_API_KEY, url=config.MODEL_URL,
        )
        user = setup_user(admin.token, "hermes-ch-lark")
        inst = setup_hermes_instance(user.token, "ch-lark")

        # 用 /openclaw/set-model（维护 primary 单模型）而非 /openclaw/add-model：
        # channel 测试 setup 只需要"实例上有一个可用模型"，无需多模型 fallback 语义。
        print(">>> Setup：为实例绑定有效模型 ...")
        resp = retry_on_gateway_restart(
            lambda: helpers.user_set_model(user.token, inst.db_id, model_ctx.db_id)
        )
        assert resp.status_code == 200, f"绑定模型失败: {resp.status_code} {resp.text}"
        print("    模型绑定成功 ✓")
        time.sleep(5)

        # 步骤 1：配置 Lark 通道
        # ChannelParams: app_id + app_secret（与 feishu 同协议，脚本层设置 FEISHU_DOMAIN=lark）
        print(">>> 步骤 1：配置 Lark 通道 ...")
        resp = retry_on_gateway_restart(
            lambda: helpers.user_set_channel(
                user.token, inst.db_id, "lark",
                keys=["app_id", "app_secret"],
                values=[config.LARK_APP_ID, config.LARK_APP_SECRET],
            )
        )
        assert resp.status_code == 200, (
            f"配置 Lark 失败: status={resp.status_code}, body={resp.text}"
        )
        print("    配置成功 ✓")
        time.sleep(10)

        # 步骤 2：查询确认
        # Lark 复用 feishu 配置字段，gateway 侧可能以 lark 或 feishu 为 key 显示
        print(">>> 步骤 2：查询确认 ...")
        inst_data = helpers.user_get_channels(user.token, instance_db_id=inst.db_id)
        channels = inst_data.get("channels", {})
        assert channels, "通道配置不应为空"
        _lark_keys = ("lark", "feishu")
        assert any(
            k in channels or any(
                c.get("channel_id", "") == k
                for c in (channels if isinstance(channels, list) else [])
            )
            for k in _lark_keys
        ), f"Lark 通道应存在于配置中（key 可为 lark/feishu）: {channels}"
        print("    查询确认 ✓")

        # 步骤 3：验证 hermes 服务仍然可用（配置通道后服务不应崩溃）
        print(">>> 步骤 3：验证 hermes 服务可用性 ...")
        verify_hermes_service(user.token, inst.db_id)
        print("    Hermes 服务可用 ✓")

        # 步骤 3b：软检查 hermes-gateway 是否真的把 lark 通道拉起来了
        print(">>> 步骤 3b：检查 hermes channel 连接状态 ...")
        expect_hermes_channel_connected(user.token, inst.db_id, min_enabled=1, timeout=90)

        # 步骤 4：删除配置
        print(">>> 步骤 4：删除 Lark 通道配置 ...")
        resp = retry_on_gateway_restart(
            lambda: helpers.user_del_channel(user.token, inst.db_id, "lark")
        )
        assert resp.status_code == 200, (
            f"删除 lark 失败: status={resp.status_code}, body={resp.text}"
        )
        print("    删除成功 ✓")
        time.sleep(5)

        # 步骤 5：查询确认已删除（轮询确认通道从列表移除）
        # Lark 复用 feishu 字段，删除后 lark / feishu 两个 key 都应消失
        print(">>> 步骤 5：查询确认已删除 ...")
        still_present = True
        for attempt in range(6):
            inst_data = helpers.user_get_channels(user.token, instance_db_id=inst.db_id)
            channels = inst_data.get("channels", {})
            if isinstance(channels, dict):
                still_present = any(
                    k in channels and (
                        channels[k].get("enabled", False)
                        if isinstance(channels.get(k), dict)
                        else True
                    )
                    for k in _lark_keys
                )
            else:
                still_present = any(k in str(channels) for k in _lark_keys)
            if not still_present:
                break
            time.sleep(5)
        assert not still_present, (
            f"删除 lark 后仍存在于通道列表中（含 feishu 别名）: {channels}"
        )
        print("    lark 已从列表移除 ✓")

        print()
        print("TC-H4.7 测试通过 ✅")

    except Exception as e:
        print(f"\nTC-H4.7 测试失败 ❌: {e}")
        traceback.print_exc()
        sys.exit(1)

    finally:
        if model_ctx:
            teardown_model(admin.token, model_ctx)


if __name__ == "__main__":
    main()
