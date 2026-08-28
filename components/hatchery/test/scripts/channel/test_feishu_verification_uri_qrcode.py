#!/usr/bin/env python3
"""
TC-4.8 飞书 auto-channel 二维码返回格式验证

验证 Feishu Device Code Flow 返回 {"verification_uri": "..."} 时，
auto-channel SSE 对前端输出的 qrcode payload 已归一化为：
  - action == "show_qrcode"
  - mode == "url"
  - content 是裸 http(s) URL

使用方式：
  export BASE_URL=http://134.175.254.166
  export SEED_ADMIN_TOKEN=xxx
  python3 test_feishu_verification_uri_qrcode.py
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
from helpers.sse import verify_feishu_qrcode_payload


def main():
    check_env()
    print()

    admin = setup_admin("ch-feishu-qr")
    user = None
    inst = None

    try:
        helpers.ensure_gateway_ui_enabled(admin.token)

        user = setup_user(admin.token, "ch-feishu-qr")
        inst = setup_instance(user.token, "ch-feishu-qr")

        verify_feishu_qrcode_payload(user.token, inst.db_id)

        print()
        print("TC-4.8 测试通过 ✅")

    except Exception as e:
        print(f"\nTC-4.8 测试失败 ❌: {e}")
        traceback.print_exc()
        sys.exit(1)


if __name__ == "__main__":
    main()
