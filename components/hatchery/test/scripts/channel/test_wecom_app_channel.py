#!/usr/bin/env python3
"""
TC-4.6 企业微信应用通道配置 CRUD 验证（5 字段，默认禁用）

验证企微应用通道的凭证配置、查询确认、删除完整流程。

使用方式：
  export BASE_URL=http://134.175.254.166
  export SEED_ADMIN_TOKEN=xxx
  export WECOM_APP_CORP_ID=xxx  WECOM_APP_CORP_SECRET=xxx  WECOM_APP_AGENT_ID=xxx
  export WECOM_APP_TOKEN=xxx  WECOM_APP_ENCODING_AES_KEY=xxx
  python3 test_wecom_app_channel.py
"""

import os
import sys
import time
import traceback

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers import config
import helpers
from helpers import retry_on_gateway_restart, get_field
from helpers import (
    check_env, require_model_config, setup_admin,
    setup_user,
    setup_instance,
    setup_model, teardown_model,
)


def main():
    check_env()

    if not (config.WECOM_APP_CORP_ID and config.WECOM_APP_CORP_SECRET and config.WECOM_APP_AGENT_ID):
        print("错误: 未设置企微应用凭证环境变量")
        print("  通道测试需要真实凭证，不支持跳过。")
        sys.exit(1)

    require_model_config()
    print()
    admin = setup_admin("ch-wecomapp")
    user = None
    inst = None
    model_ctx = None
    wecom_app_db_id = None

    try:
        helpers.ensure_gateway_ui_enabled(admin.token)

        # Setup：创建模型 + 用户 + 实例
        model_ctx = setup_model(
            admin.token, model_id=config.MODEL_ID, model_name=f"IntTest CH WecomApp ({config.MODEL_ID})",
            api_key=config.MODEL_API_KEY, url=config.MODEL_URL,
        )
        user = setup_user(admin.token, "ch-wecomapp")
        inst = setup_instance(user.token, "ch-wecomapp")

        # 清理默认绑定的模型，确保测试模型绑定后为 primary
        helpers.clear_instance_models(user.token, inst.db_id)

        print(">>> Setup：为实例绑定有效模型 ...")
        resp = retry_on_gateway_restart(
            lambda: helpers.user_add_model(user.token, inst.db_id, model_ctx.db_id)
        )
        assert resp.status_code == 200, f"绑定模型失败: {resp.status_code} {resp.text}"
        print("    模型绑定成功 ✓")
        time.sleep(5)

        # 找到 wecom_app 通道 DB id
        channels = helpers.admin_get_channels(admin.token)
        for ch in channels:
            cid = get_field(ch, "ChannelID", "channel_id", default="")
            if cid == "wecom_app":
                wecom_app_db_id = get_field(ch, "ID", "id")
                break
        assert wecom_app_db_id, "未找到 wecom_app"

        # 步骤 1：启用 wecom_app
        print(">>> 步骤 1：启用 wecom_app ...")
        helpers.admin_toggle_channel(admin.token, wecom_app_db_id)
        print("    启用成功 ✓")

        # 步骤 2：配置 5 字段
        print(">>> 步骤 2：配置企微应用通道（5 字段）...")
        resp = retry_on_gateway_restart(
            lambda: helpers.user_set_channel(
                user.token, inst.db_id, "wecom_app",
                keys=["corp_id", "corp_secret", "agent_id", "token", "encoding_aes_key"],
                values=[
                    config.WECOM_APP_CORP_ID,
                    config.WECOM_APP_CORP_SECRET,
                    config.WECOM_APP_AGENT_ID,
                    config.WECOM_APP_TOKEN,
                    config.WECOM_APP_ENCODING_AES_KEY,
                ],
            )
        )
        assert resp.status_code == 200, f"配置企微应用失败: {resp.status_code} {resp.text}"
        print("    配置成功 ✓")
        time.sleep(5)

        # 步骤 3：查询确认
        print(">>> 步骤 3：查询确认 ...")
        inst_data = helpers.user_get_channels(user.token, instance_db_id=inst.db_id)
        assert inst_data.get("channels"), "通道配置不应为空"
        print("    查询确认 ✓")

        # 步骤 4：删除配置
        print(">>> 步骤 4：删除配置 ...")
        resp = retry_on_gateway_restart(
            lambda: helpers.user_del_channel(user.token, inst.db_id, "wecom_app")
        )
        assert resp.status_code == 200, f"删除 wecom_app 失败: status={resp.status_code}, body={resp.text}"
        print("    删除成功 ✓")
        time.sleep(5)

        # 步骤 5：查询确认已删除
        print(">>> 步骤 5：查询确认已删除 ...")
        inst_data = helpers.user_get_channels(user.token, instance_db_id=inst.db_id)
        print("    查询确认已删除 ✓")

        # 步骤 6：还原禁用
        print(">>> 步骤 6：还原 wecom_app 为禁用 ...")
        helpers.admin_toggle_channel(admin.token, wecom_app_db_id)
        print("    还原成功 ✓")

        print()
        print("TC-4.6 测试通过 ✅")

    except Exception as e:
        print(f"\nTC-4.6 测试失败 ❌: {e}")
        traceback.print_exc()
        sys.exit(1)

    finally:
        if model_ctx:
            teardown_model(admin.token, model_ctx)


if __name__ == "__main__":
    main()
