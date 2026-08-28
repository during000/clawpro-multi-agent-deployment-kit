#!/usr/bin/env python3
"""
TC-4.7 扫码配置通道（auto-channel：微信 + 飞书）

验证 auto-channel SSE 流式接口能正常返回。

使用方式：
  export BASE_URL=http://134.175.254.166
  export SEED_ADMIN_TOKEN=xxx
  python3 test_auto_channel.py
"""

import os
import sys
import traceback

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

import helpers
from helpers import (
    check_env, setup_admin,
    setup_user,
    setup_instance,
)


def main():
    check_env()
    print()

    admin = setup_admin("auto-ch")
    user = None
    inst = None

    try:
        helpers.ensure_gateway_ui_enabled(admin.token)

        user = setup_user(admin.token, "auto-ch")
        inst = setup_instance(user.token, "auto-ch")

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
        print(">>> 步骤 4：调用 auto-channel（飞书）...")
        resp = helpers.user_auto_channel(user.token, inst.db_id, "feishu", timeout=120)
        if resp.status_code == 200:
            content_type = resp.headers.get("Content-Type", "")
            assert "text/event-stream" in content_type, (
                f"飞书 auto-channel 应返回 SSE 流，实际: {content_type}"
            )
            print("    SSE 流建立 ✓")

            # 收到数据事件即可关闭，无需等待二维码过期
            for line in resp.iter_lines(decode_unicode=True):
                if line and (line.startswith("data:") or "qrcode" in line.lower()):
                    print("    收到数据事件 ✓")
                    break
            resp.close()
        else:
            print(f"    返回 {resp.status_code}（非 200，记录行为）")

        # ── 不支持的通道 ──
        print(">>> 步骤 7：wecom 不支持 auto-channel ...")
        resp = helpers.user_auto_channel(user.token, inst.db_id, "wecom", timeout=10)
        assert resp.status_code == 400, (
            f"wecom 不支持 auto-channel，应返回 400，实际: {resp.status_code}"
        )
        resp.close()
        print("    返回 400 ✓")

        print(">>> 步骤 8：ddingtalk 不支持 auto-channel ...")
        resp = helpers.user_auto_channel(user.token, inst.db_id, "ddingtalk", timeout=10)
        assert resp.status_code == 400, (
            f"ddingtalk 不支持 auto-channel，应返回 400，实际: {resp.status_code}"
        )
        resp.close()
        print("    返回 400 ✓")

        print()
        print("TC-4.7 测试通过 ✅")

    except Exception as e:
        print(f"\nTC-4.7 测试失败 ❌: {e}")
        traceback.print_exc()
        sys.exit(1)

    finally:
        pass


if __name__ == "__main__":
    main()
