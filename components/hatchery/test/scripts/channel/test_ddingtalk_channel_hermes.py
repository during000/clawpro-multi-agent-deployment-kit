#!/usr/bin/env python3
"""
TC-H4.4 Hermes 钉钉通道配置 + 查询确认

验证 hermes 实例的钉钉通道凭证配置、查询确认、删除完整流程。

使用方式：
  export API=http://134.175.254.166
  export ADMIN_TOKEN=xxx
  export MODEL_ID=xxx  MODEL_API_KEY=xxx  MODEL_URL=xxx  MODEL_TYPE=xxx
  export DDINGTALK_CLIENT_ID=xxx  DDINGTALK_CLIENT_SECRET=xxx
  python3 test_hermes_ddingtalk_channel.py
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

    if not config.DDINGTALK_CLIENT_ID or not config.DDINGTALK_CLIENT_SECRET:
        print("错误: 未设置 DDINGTALK_CLIENT_ID / DDINGTALK_CLIENT_SECRET 环境变量")
        print("  通道测试需要真实凭证，不支持跳过。")
        sys.exit(1)

    require_model_config()
    print()
    admin = setup_admin("hermes-ch-ding")
    user = None
    inst = None
    model_ctx = None

    try:
        helpers.ensure_gateway_ui_enabled(admin.token)

        # Setup：创建模型 + 用户 + hermes 实例 + 绑定模型
        model_ctx = setup_model(
            admin.token, model_id=config.MODEL_ID,
            model_name=f"IntTest Hermes CH Ding ({config.MODEL_ID})",
            api_key=config.MODEL_API_KEY, url=config.MODEL_URL,
        )
        user = setup_user(admin.token, "hermes-ch-ding")
        inst = setup_hermes_instance(user.token, "ch-ding")

        # 用 /openclaw/set-model（维护 primary 单模型）而非 /openclaw/add-model：
        # channel 测试 setup 只需要"实例上有一个可用模型"，无需多模型 fallback 语义。
        print(">>> Setup：为实例绑定有效模型 ...")
        resp = retry_on_gateway_restart(
            lambda: helpers.user_set_model(user.token, inst.db_id, model_ctx.db_id)
        )
        assert resp.status_code == 200, f"绑定模型失败: {resp.status_code} {resp.text}"
        print("    模型绑定成功 ✓")
        time.sleep(5)

        # 步骤 1：配置钉钉通道
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

        # 等待 gateway 重启完成
        print("    等待 gateway 重启 (15s) ...")
        time.sleep(15)

        # 步骤 2：查询确认
        print(">>> 步骤 2：查询确认 ...")
        inst_data = helpers.user_get_channels(user.token, instance_db_id=inst.db_id)
        channels = inst_data.get("channels", {})
        assert channels, "通道配置不应为空"
        # 钉钉通道可能以 ddingtalk 或 dingtalk-connector 为 key
        _gw_name_map = {"ddingtalk": "dingtalk-connector"}
        gw_name = _gw_name_map.get("ddingtalk", "ddingtalk")
        assert "ddingtalk" in channels or gw_name in channels or any(
            c.get("channel_id", "") in ("ddingtalk", gw_name)
            for c in (channels if isinstance(channels, list) else [])
        ), f"钉钉通道应存在于配置中: {channels}"
        print("    查询确认 ✓")

        # 步骤 3：验证 hermes 服务仍然可用
        print(">>> 步骤 3：验证 hermes 服务可用性 ...")
        verify_hermes_service(user.token, inst.db_id)
        print("    Hermes 服务可用 ✓")

        # 步骤 3b：软检查 hermes-gateway 是否真的把 dingtalk 通道拉起来了
        # ddingtalk 走 stream 长连接，连接耗时较长，timeout 放大到 120s
        print(">>> 步骤 3b：检查 hermes channel 连接状态 ...")
        expect_hermes_channel_connected(user.token, inst.db_id, min_enabled=1, timeout=120)

        # 步骤 4：删除配置
        print(">>> 步骤 4：删除钉钉通道配置 ...")
        # Hermes 白名单只允许前端契约名 ddingtalk（dingtalk-connector 是 gateway 内部
        # 真实 key，不在 hermes 白名单中）。前端 UI 删除时同样传 ddingtalk，
        # del_channel_hermes.sh 内部会将其映射为 acli/harness 的 dingtalk platform，
        # 由 acli/harness 负责真正清理底层 dingtalk-connector 通道配置。
        resp = retry_on_gateway_restart(
            lambda: helpers.user_del_channel(user.token, inst.db_id, "ddingtalk")
        )
        assert resp.status_code == 200, (
            f"删除 ddingtalk 失败: status={resp.status_code}, body={resp.text}"
        )
        print("    删除成功 ✓")
        time.sleep(5)

        # 步骤 5：查询确认已删除（轮询确认 ddingtalk / dingtalk-connector 都从列表移除）
        print(">>> 步骤 5：查询确认已删除 ...")
        still_present = True
        for attempt in range(6):
            inst_data = helpers.user_get_channels(user.token, instance_db_id=inst.db_id)
            channels = inst_data.get("channels", {})
            if isinstance(channels, dict):
                # gateway 实际以 dingtalk-connector 为 key 存储；删除链路应同时清理它
                ddt = channels.get("dingtalk-connector") or channels.get("ddingtalk")
                still_present = bool(ddt) and (
                    ddt.get("enabled", False) if isinstance(ddt, dict) else True
                )
            else:
                still_present = ("dingtalk-connector" in str(channels)
                                 or "ddingtalk" in str(channels))
            if not still_present:
                break
            time.sleep(5)
        assert not still_present, (
            f"删除 ddingtalk 后仍存在于通道列表中: {channels}"
        )
        print("    ddingtalk 已从列表移除 ✓")

        print()
        print("TC-H4.4 测试通过 ✅")

    except Exception as e:
        print(f"\nTC-H4.4 测试失败 ❌: {e}")
        traceback.print_exc()
        sys.exit(1)

    finally:
        if model_ctx:
            teardown_model(admin.token, model_ctx)


if __name__ == "__main__":
    main()
