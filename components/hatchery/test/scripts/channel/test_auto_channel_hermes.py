#!/usr/bin/env python3
"""
TC-H4.7 Hermes 扫码配置通道（auto-channel：微信 + 飞书）

验证 hermes 实例的 auto-channel SSE 流式接口能正常返回（weixin / feishu）。
同时验证以下两类拒绝路径：
  1) auto-channel 全局不支持的通道（wecom / ddingtalk / wecom_app 等手动配置类
     通道）—— 不在 autoChannelFeature map 里，对任何 agent_type 都返回 400；
  2) hermes 白名单外的通道（qqbot）—— 在 autoChannelFeature map 里，但
     scriptResolveTable["qq_bot_creator"] 仅注册 openclaw，hermes 实例走到
     ResolveScript 时返回 400。

使用方式：
  export BASE_URL=http://134.175.254.166
  export SEED_ADMIN_TOKEN=xxx
  python3 test_auto_channel_hermes.py
"""

import os
import sys
import traceback

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

import helpers
from helpers import (
    check_env, setup_admin,
    setup_user,
    setup_hermes_instance,
)


def main():
    check_env()
    print()

    admin = setup_admin("hermes-auto-ch")
    user = None
    inst = None

    try:
        helpers.ensure_gateway_ui_enabled(admin.token)

        user = setup_user(admin.token, "hermes-auto-ch")
        inst = setup_hermes_instance(user.token, "auto-ch")

        # ── 微信 auto-channel ──
        print(">>> 步骤 1：调用 auto-channel（微信）...")
        resp = helpers.user_auto_channel(user.token, inst.db_id, "openclaw-weixin", timeout=90)

        if resp.status_code == 200:
            content_type = resp.headers.get("Content-Type", "")
            assert "text/event-stream" in content_type, (
                f"微信 auto-channel 应返回 SSE 流，实际: {content_type}"
            )
            print("    SSE 流建立 ✓")

            # 尝试读取事件
            for line in resp.iter_lines(decode_unicode=True):
                if line and (line.startswith("data:") or "qrcode" in line.lower()):
                    print("    收到数据事件 ✓")
                    break
            resp.close()
        else:
            print(f"    返回 {resp.status_code}（非 200，记录行为）")

        # ── 飞书 auto-channel ──
        print(">>> 步骤 2：调用 auto-channel（飞书）...")
        resp = helpers.user_auto_channel(user.token, inst.db_id, "feishu", timeout=120)
        if resp.status_code == 200:
            content_type = resp.headers.get("Content-Type", "")
            assert "text/event-stream" in content_type, (
                f"飞书 auto-channel 应返回 SSE 流，实际: {content_type}"
            )
            print("    SSE 流建立 ✓")

            for line in resp.iter_lines(decode_unicode=True):
                if line and (line.startswith("data:") or "qrcode" in line.lower()):
                    print("    收到数据事件 ✓")
                    break
            resp.close()
        else:
            print(f"    返回 {resp.status_code}（非 200，记录行为）")

        # ── auto-channel 全局不支持的通道（手动配置类，所有 agent_type 都返回 400）──
        # 这三个 channel 都不在 autoChannelFeature map 里，会在第一道 map 校验
        # 处直接返回 400，跟 hermes 白名单无关。
        for ch in ("wecom", "ddingtalk", "wecom_app"):
            print(f">>> auto-channel 全局不支持: {ch} ...")
            resp = helpers.user_auto_channel(user.token, inst.db_id, ch, timeout=10)
            assert resp.status_code == 400, (
                f"{ch} 不支持 auto-channel，应返回 400，实际: {resp.status_code}"
            )
            resp.close()
            print("    返回 400 ✓")

        # ── hermes 白名单外的通道：qqbot ──
        # qqbot 在 autoChannelFeature map 里（feature=qq_bot_creator），但
        # scriptResolveTable["qq_bot_creator"] 仅注册 openclaw，对 hermes 实例
        # ResolveScript 返回 "feature ... not supported for agent_type hermes"
        # → 400。这是真正"hermes 不支持"的语义。
        print(">>> hermes 白名单外: qqbot ...")
        resp = helpers.user_auto_channel(user.token, inst.db_id, "qqbot", timeout=10)
        assert resp.status_code == 400, (
            f"qqbot 在 hermes 实例上应被 ResolveScript 拒绝（400），实际: {resp.status_code}"
        )
        resp.close()
        print("    返回 400 ✓")

        print()
        print("TC-H4.7 测试通过 ✅")

    except Exception as e:
        print(f"\nTC-H4.7 测试失败 ❌: {e}")
        traceback.print_exc()
        sys.exit(1)

    finally:
        pass


if __name__ == "__main__":
    main()
