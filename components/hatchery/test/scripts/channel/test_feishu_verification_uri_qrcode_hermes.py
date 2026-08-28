#!/usr/bin/env python3
"""
TC-H4.8 Hermes 飞书 auto-channel 二维码返回格式验证

验证 hermes 实例的 Feishu Device Code Flow 返回 {"verification_uri": "..."} 时，
auto-channel SSE 对前端输出的 qrcode payload 已归一化为：
  - action == "show_qrcode"
  - mode == "url"
  - content 是裸 http(s) URL

使用方式：
  export API=http://134.175.254.166
  export ADMIN_TOKEN=xxx
  export MODEL_ID=xxx  MODEL_API_KEY=xxx  MODEL_URL=xxx  MODEL_TYPE=xxx
  python3 test_feishu_verification_uri_qrcode_hermes.py
"""

import os
import sys
import traceback

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

import helpers
from helpers import (
    check_env, require_model_config, setup_admin,
    setup_user,
    setup_hermes_instance,
    setup_model, teardown_model,
    retry_on_gateway_restart,
)
from helpers.sse import verify_feishu_qrcode_payload
from helpers import config
import time


def main():
    check_env()
    require_model_config()
    print()

    admin = setup_admin("hermes-ch-feishu-qr")
    user = None
    inst = None
    model_ctx = None

    try:
        helpers.ensure_gateway_ui_enabled(admin.token)

        # Setup：创建模型 + 用户 + hermes 实例 + 绑定模型
        model_ctx = setup_model(
            admin.token, model_id=config.MODEL_ID,
            model_name=f"IntTest Hermes CH Feishu QR ({config.MODEL_ID})",
            api_key=config.MODEL_API_KEY, url=config.MODEL_URL,
        )
        user = setup_user(admin.token, "hermes-ch-feishu-qr")
        inst = setup_hermes_instance(user.token, "ch-feishu-qr")

        # 用 /openclaw/set-model（维护 primary 单模型）而非 /openclaw/add-model：
        # channel 测试 setup 只需要"实例上有一个可用模型"，无需多模型 fallback 语义。
        print(">>> Setup：为实例绑定有效模型 ...")
        resp = retry_on_gateway_restart(
            lambda: helpers.user_set_model(user.token, inst.db_id, model_ctx.db_id)
        )
        assert resp.status_code == 200, f"绑定模型失败: {resp.status_code} {resp.text}"
        print("    模型绑定成功 ✓")
        time.sleep(5)

        # 验证飞书二维码 payload
        verify_feishu_qrcode_payload(user.token, inst.db_id)

        print()
        print("TC-H4.8 测试通过 ✅")

    except Exception as e:
        print(f"\nTC-H4.8 测试失败 ❌: {e}")
        traceback.print_exc()
        sys.exit(1)

    finally:
        if model_ctx:
            teardown_model(admin.token, model_ctx)


if __name__ == "__main__":
    main()
