#!/usr/bin/env python3
"""
TC-H4.1 Hermes 通道实例配置（用户视角）：白名单校验 + 多通道 CRUD + 聚合查询

验证：
  1. Hermes 白名单机制（支持 openclaw-weixin, wecom, feishu, ddingtalk, qqbot；不支持 wecom_app）
  2. 非白名单通道被拒绝
  3. 多通道配置后聚合查询
  4. 通道删除与清空确认

使用方式：
  export API=http://134.175.254.166
  export ADMIN_TOKEN=xxx
  python3 test_hermes_channel_instance.py
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
    setup_user,
    setup_hermes_instance,
    filter_site_visible_channels,
    HERMES_WHITELIST_CHANNELS,
)

# Hermes 不支持的通道（用于负面测试）
HERMES_UNSUPPORTED_CHANNELS = {"wecom_app"}


def main():
    check_env()
    print()

    admin = setup_admin("hermes-ch-inst")
    user = None
    inst = None

    try:
        helpers.ensure_gateway_ui_enabled(admin.token)

        user = setup_user(admin.token, "hermes-ch-inst")
        inst = setup_hermes_instance(user.token, "ch-inst")

        # ══════════════════════════════════════════════════════════════
        # Part 1：全局 vs 实例通道列表
        # ══════════════════════════════════════════════════════════════
        print(">>> 步骤 1：查询全局通道列表（不传 id）...")
        global_data = helpers.user_get_channels(user.token)
        assert global_data, "全局通道列表不应为空"
        print("    全局列表非空 ✓")

        print(">>> 步骤 2：查询实例通道配置（传 id）...")
        inst_data = helpers.user_get_channels(user.token, instance_db_id=inst.db_id)
        assert inst_data, "实例通道数据不应为空"
        print("    实例通道数据非空 ✓")

        # ══════════════════════════════════════════════════════════════
        # Part 2：白名单校验
        # ══════════════════════════════════════════════════════════════
        print(">>> 步骤 3：断言 hermes 白名单 ...")
        supported = inst_data.get("agent_type_supported_channels", [])
        if isinstance(supported, list) and supported:
            expected_whitelist = filter_site_visible_channels(HERMES_WHITELIST_CHANNELS)
            hidden_whitelist = HERMES_WHITELIST_CHANNELS - expected_whitelist
            for ch_name in expected_whitelist:
                assert ch_name in supported, f"Hermes 白名单应包含 {ch_name}: {supported}"
            for ch_name in hidden_whitelist:
                assert ch_name not in supported, f"当前站点 Hermes 白名单不应包含 {ch_name}: {supported}"
            # 验证 wecom_app 不在白名单中
            for ch_name in HERMES_UNSUPPORTED_CHANNELS:
                assert ch_name not in supported, (
                    f"Hermes 白名单不应包含 {ch_name}: {supported}"
                )
            print(f"    白名单包含 {len(expected_whitelist)} 个当前站点可见通道，"
                  f"不含 {HERMES_UNSUPPORTED_CHANNELS} ✓")
        else:
            print(f"    ⚠ agent_type_supported_channels 为空或不存在，跳过白名单断言")

        print(">>> 步骤 4：配置非白名单通道 wecom_app → 应被拒绝 ...")
        resp = helpers.user_set_channel(
            user.token, inst.db_id, "wecom_app",
            keys=["corp_id", "corp_secret", "agent_id", "token", "encoding_aes_key"],
            values=["fake-corp", "fake-secret", "fake-agent", "fake-token", "fake-aes"],
        )
        assert resp.status_code == 400, (
            f"非白名单通道 wecom_app 应返回 400，实际: {resp.status_code} {resp.text}"
        )
        print("    返回 400 ✓")

        print(">>> 步骤 4b：配置不存在的通道 → 应被拒绝 ...")
        resp = helpers.user_set_channel(
            user.token, inst.db_id, "nonexistent_channel",
            keys=["token"],
            values=["fake"],
        )
        assert resp.status_code == 400, (
            f"不存在的通道应返回 400，实际: {resp.status_code} {resp.text}"
        )
        print("    返回 400 ✓")

        # ══════════════════════════════════════════════════════════════
        # Part 3：多通道配置 + 聚合查询
        # ══════════════════════════════════════════════════════════════
        print(">>> 步骤 5：配置飞书通道 ...")
        resp = retry_on_gateway_restart(
            lambda: helpers.user_set_channel(
                user.token, inst.db_id, "feishu",
                keys=["app_id", "app_secret"],
                values=["fake-app-id", "fake-secret"],
            )
        )
        assert resp.status_code == 200, f"配置飞书失败: {resp.status_code} {resp.text}"
        print("    飞书配置成功 ✓")
        time.sleep(30)

        print(">>> 步骤 6：配置企微通道 ...")
        resp = retry_on_gateway_restart(
            lambda: helpers.user_set_channel(
                user.token, inst.db_id, "wecom",
                keys=["bot_id", "secret"],
                values=["fake-bot-id", "fake-secret"],
            )
        )
        assert resp.status_code == 200, f"配置企微失败: {resp.status_code} {resp.text}"
        print("    企微配置成功 ✓")
        time.sleep(30)

        print(">>> 步骤 7：配置钉钉通道 ...")
        resp = retry_on_gateway_restart(
            lambda: helpers.user_set_channel(
                user.token, inst.db_id, "ddingtalk",
                keys=["client_id", "client_secret"],
                values=["fake-cid", "fake-csecret"],
            )
        )
        assert resp.status_code == 200, f"配置钉钉失败: {resp.status_code} {resp.text}"
        print("    钉钉配置成功 ✓")
        time.sleep(30)

        print(">>> 步骤 8：查询聚合列表 ...")
        inst_data = helpers.user_get_channels(user.token, instance_db_id=inst.db_id)
        channels = inst_data.get("channels", {})
        _gw_name_map = {"ddingtalk": "dingtalk-connector"}
        for ch_name in ["feishu", "wecom", "ddingtalk"]:
            gw_name = _gw_name_map.get(ch_name, ch_name)
            assert ch_name in channels or gw_name in channels or any(
                (c.get("channel_id") or c.get("ChannelId", "")) in (ch_name, gw_name)
                for c in (channels if isinstance(channels, list) else [])
            ), f"通道 {ch_name} (gateway={gw_name}) 配置应存在: {channels}"
        print("    三个通道聚合显示 ✓")

        # ══════════════════════════════════════════════════════════════
        # Part 4：通道删除 + 清空确认
        # ══════════════════════════════════════════════════════════════
        print(">>> 步骤 9：删除飞书通道 ...")
        resp = retry_on_gateway_restart(
            lambda: helpers.user_del_channel(user.token, inst.db_id, "feishu")
        )
        assert resp.status_code == 200, f"删除飞书失败: {resp.status_code}"
        print("    删除成功 ✓")
        time.sleep(5)

        print(">>> 步骤 10：删除剩余通道 ...")
        # Hermes 删除时使用前端契约名（与 set 时一致）：ddingtalk 而非 dingtalk-connector。
        # del_channel_hermes.sh 内部会把 ddingtalk 映射为 acli/harness 的 dingtalk platform，
        # 由 acli/harness 负责清理底层 dingtalk-connector 配置。
        # 别名映射：API 删除名 → 实际可能在 channels 列表里出现的 key（包括 gateway 别名）
        _alias_map = {"ddingtalk": ("ddingtalk", "dingtalk-connector")}
        for ch_name in ["wecom", "ddingtalk"]:
            resp = retry_on_gateway_restart(
                lambda name=ch_name: helpers.user_del_channel(user.token, inst.db_id, name)
            )
            assert resp.status_code == 200, (
                f"删除 {ch_name} 失败: status={resp.status_code}, body={resp.text}"
            )
            # 删除后轮询确认通道（含 gateway 别名）都已从列表移除
            aliases = _alias_map.get(ch_name, (ch_name,))
            for attempt in range(6):
                time.sleep(5)
                inst_data = helpers.user_get_channels(user.token, instance_db_id=inst.db_id)
                channels = inst_data.get("channels", {})
                if isinstance(channels, dict):
                    still_present = any(
                        k in channels and (channels[k].get("enabled", False)
                                           if isinstance(channels[k], dict) else True)
                        for k in aliases
                    )
                else:
                    still_present = any(k in str(channels) for k in aliases)
                if not still_present:
                    break
            assert not still_present, f"删除 {ch_name} 后仍存在于通道列表中（含别名 {aliases}）: {channels}"
            print(f"    {ch_name} 删除确认 ✓")
        print("    全部删除成功 ✓")

        print(">>> 步骤 11：确认通道已清空 ...")
        configured = None
        channels = None
        for attempt in range(4):
            if attempt > 0:
                print(f"    第 {attempt} 次重试，等待 15s ...")
                time.sleep(15)
            inst_data = helpers.user_get_channels(user.token, instance_db_id=inst.db_id)
            channels = inst_data.get("channels", {})
            if isinstance(channels, dict):
                configured = [
                    k for k, v in channels.items()
                    if v and not (isinstance(v, dict) and not v.get("enabled", False))
                ]
            elif isinstance(channels, list):
                configured = channels
            else:
                configured = []
            if len(configured) == 0:
                break
            print(f"    检查仍有残留: {configured}")
        assert len(configured) == 0, f"通道应已清空，实际: {channels}"
        print("    通道已清空 ✓")

        print()
        print("TC-H4.1 测试通过 ✅")

    except Exception as e:
        print(f"\nTC-H4.1 测试失败 ❌: {e}")
        traceback.print_exc()
        sys.exit(1)

    finally:
        pass


if __name__ == "__main__":
    main()
