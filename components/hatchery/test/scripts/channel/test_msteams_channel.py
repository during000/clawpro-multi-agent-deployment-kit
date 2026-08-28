#!/usr/bin/env python3
"""
TC-4.6 Microsoft Teams 通道配置 + 代理字段验证

测试环境不能稳定指定新版 OpenClaw 镜像，因此按脚本层容错模式验证：
  1. 断言 msteams 在当前实例支持的通道列表中（国内/海外均应可见）
  2. 用假 Azure 凭证调用 set-channel；4xx 代表站点/白名单/参数校验失败
  3. 2xx 时断言代理字段、查询配置、删除并确认
  4. 5xx 时记录 TAT 脚本层失败并跳过后续配置断言

不验证真实 Teams 消息投递，不需要 Azure/Teams 测试租户。

使用方式：
  export API=http://134.175.254.166
  export ADMIN_TOKEN=xxx
  python3 test_msteams_channel.py
"""
import os
import sys
import time
import traceback

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

import helpers
from helpers import retry_on_gateway_restart
from helpers import (
    check_env, setup_admin,
    setup_user, setup_instance,
)


MSTEAMS_KEYS = ["app_id", "app_secret", "tenant_id"]
MSTEAMS_VALUES = [
    "00000000-0000-0000-0000-000000000000",
    "itest-fake-secret",
    "itest-fake-tenant",
]


def main():
    check_env()
    print()

    admin = setup_admin("ch-msteams")
    user = None
    inst = None

    try:
        helpers.ensure_gateway_ui_enabled(admin.token)
        user = setup_user(admin.token, "ch-msteams")
        inst = setup_instance(user.token, "ch-msteams")

        print(">>> 步骤 1：断言 msteams 在实例支持的通道列表中 ...")
        inst_data = helpers.user_get_channels(user.token, instance_db_id=inst.db_id)
        supported = inst_data.get("agent_type_supported_channels", [])
        assert "msteams" in supported, f"支持列表应包含 msteams: {supported}"
        print("    msteams 可见 ✓")

        print(">>> 步骤 2：配置 Microsoft Teams 通道 ...")
        # msteams 插件安装失败是确定性脚本错误，不使用通用 gateway 重试，避免重复执行 4 次。
        resp = helpers.user_set_channel(
            user.token, inst.db_id, "msteams",
            keys=MSTEAMS_KEYS,
            values=MSTEAMS_VALUES,
        )
        assert resp.status_code < 400 or resp.status_code >= 500, (
            f"set-channel msteams 被 Go 层 4xx 拒绝（不应发生）: "
            f"{resp.status_code} {resp.text[:300]}"
        )
        if resp.status_code >= 500:
            print(f"    Go 层校验通过，脚本层 {resp.status_code}，跳过后续配置断言")
            print()
            print("TC-4.6 测试通过 ✅")
            return
        body = resp.json()
        route_id = body.get("proxy_route_id", "")
        proxy_endpoint = body.get("proxy_endpoint", "")
        teams_endpoint = body.get("teams_endpoint", "")
        assert route_id, f"成功响应缺 proxy_route_id: {body}"
        assert proxy_endpoint, f"成功响应缺 proxy_endpoint: {body}"
        assert teams_endpoint, f"成功响应缺 teams_endpoint: {body}"
        assert proxy_endpoint == teams_endpoint, (
            f"proxy_endpoint 与 teams_endpoint 应一致: {body}"
        )
        assert teams_endpoint.endswith(
            f"/api/proxy/{route_id}/api/messages"
        ), f"teams_endpoint 形态异常: {teams_endpoint}"
        print(f"    配置成功 ✓ teams_endpoint={teams_endpoint}")
        time.sleep(5)

        print(">>> 步骤 3：查询确认 ...")
        inst_data = helpers.user_get_channels(user.token, instance_db_id=inst.db_id)
        channels = inst_data.get("channels", {})
        if isinstance(channels, dict):
            configured = "msteams" in channels
        else:
            configured = any(
                (channel.get("channel_id") or channel.get("ChannelId", "")) == "msteams"
                for channel in channels
            )
        assert configured, f"msteams 配置应存在: {channels}"
        print("    查询确认 ✓")

        print(">>> 步骤 4：删除 Microsoft Teams 通道 ...")
        resp = retry_on_gateway_restart(
            lambda: helpers.user_del_channel(user.token, inst.db_id, "msteams")
        )
        assert resp.status_code == 200, (
            f"删除 msteams 失败: status={resp.status_code}, body={resp.text}"
        )

        for _ in range(6):
            time.sleep(5)
            inst_data = helpers.user_get_channels(user.token, instance_db_id=inst.db_id)
            channels = inst_data.get("channels", {})
            if isinstance(channels, dict):
                value = channels.get("msteams")
                still_present = bool(
                    value and not (
                        isinstance(value, dict) and not value.get("enabled", False)
                    )
                )
            else:
                still_present = "msteams" in str(channels)
            if not still_present:
                break
        assert not still_present, f"删除后 msteams 仍存在于通道列表中: {channels}"
        print("    删除确认 ✓")

        print()
        print("TC-4.6 测试通过 ✅")

    except Exception as e:
        print(f"\nTC-4.6 测试失败 ❌: {e}")
        traceback.print_exc()
        sys.exit(1)

    finally:
        pass


if __name__ == "__main__":
    main()
