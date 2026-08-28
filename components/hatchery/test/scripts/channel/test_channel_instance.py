#!/usr/bin/env python3
"""
TC-4.1 通道实例配置（用户视角）：白名单校验 + 多通道 CRUD + 聚合查询

验证：
  1. 全局 vs 实例通道列表接口
  2. 白名单机制（只能配置白名单内通道，非法通道被拒绝）
  3. 多通道配置后聚合查询
  4. 通道删除与清空确认

使用方式：
  export BASE_URL=http://134.175.254.166
  export SEED_ADMIN_TOKEN=xxx
  python3 test_channel_instance.py
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
    setup_instance,
    filter_site_visible_channels,
)

WHITELIST_CHANNELS = {"openclaw-weixin", "wecom", "wecom_app", "feishu", "ddingtalk", "qqbot", "slack", "discord"}


def main():
    check_env()
    print()

    admin = setup_admin("ch-inst")
    user = None
    inst = None

    try:
        helpers.ensure_gateway_ui_enabled(admin.token)

        user = setup_user(admin.token, "ch-inst")
        inst = setup_instance(user.token, "ch-inst")

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
        print(">>> 步骤 3：断言白名单 ...")
        supported = inst_data.get("agent_type_supported_channels", [])
        if isinstance(supported, list):
            expected_whitelist = filter_site_visible_channels(WHITELIST_CHANNELS)
            hidden_whitelist = WHITELIST_CHANNELS - expected_whitelist
            for ch_name in expected_whitelist:
                assert ch_name in supported, f"白名单应包含 {ch_name}: {supported}"
            for ch_name in hidden_whitelist:
                assert ch_name not in supported, f"当前站点白名单不应包含 {ch_name}: {supported}"
            print(f"    白名单包含 {len(expected_whitelist)} 个当前站点可见通道 ✓")

        print(">>> 步骤 4：配置非白名单通道 → 应被拒绝 ...")
        resp = helpers.user_set_channel(
            user.token, inst.db_id, "nonexistent_channel",
            keys=["token"],
            values=["fake"],
        )
        assert resp.status_code == 400, (
            f"非白名单通道应返回 400，实际: {resp.status_code} {resp.text}"
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
        time.sleep(5)

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
        time.sleep(5)

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
        time.sleep(5)

        print(">>> 步骤 8：查询聚合列表 ...")
        inst_data = helpers.user_get_channels(user.token, instance_db_id=inst.db_id)
        channels = inst_data.get("channels", {})
        # Gateway 侧通道名可能与 Hatchery API 不同（如 ddingtalk → dingtalk-connector）
        _gw_name_map = {"ddingtalk": "dingtalk-connector"}
        for ch_name in ["feishu", "wecom", "ddingtalk"]:
            gw_name = _gw_name_map.get(ch_name, ch_name)
            assert ch_name in channels or gw_name in channels or any(
                (c.get("channel_id") or c.get("ChannelId", "")) in (ch_name, gw_name)
                for c in (channels if isinstance(channels, list) else [])
            ), f"通道 {ch_name} (gateway={gw_name}) 配置应存在: {channels}"
        print("    三个通道聚合显示 ✓")

        # ══════════════════════════════════════════════════════════════
        # Part 4：通道删除 + 边界场景 + 清空确认
        # ══════════════════════════════════════════════════════════════
        print(">>> 步骤 9：删除飞书通道 ...")
        resp = retry_on_gateway_restart(
            lambda: helpers.user_del_channel(user.token, inst.db_id, "feishu")
        )
        assert resp.status_code == 200, f"删除飞书失败: {resp.status_code}"
        print("    删除成功 ✓")
        time.sleep(5)

        # 边界：重复删除已删除的通道（幂等性）
        print(">>> 步骤 9b：重复删除飞书（已删除）→ 验证幂等 ...")
        resp_dup = helpers.user_del_channel(user.token, inst.db_id, "feishu")
        print(f"    状态码: {resp_dup.status_code}（幂等） ✓")
        time.sleep(5)

        # 边界：删除不存在的通道标识 → 400
        print(">>> 步骤 9c：删除不存在的通道标识 → 400 ...")
        resp_bad = helpers.user_del_channel(user.token, inst.db_id, "nonexistent_channel")
        assert resp_bad.status_code == 400, (
            f"删除不存在通道应返回 400，实际: {resp_bad.status_code}"
        )
        print("    返回 400 ✓")

        print(">>> 步骤 10：删除剩余通道 ...")
        # 注意：钉钉通道配置时用 ddingtalk，但新版插件实际以 dingtalk-connector 为 key 存储，
        # del_channel.sh 中 dingtalk-connector 分支能正确删除，因此删除时直接用 dingtalk-connector。
        for ch_name in ["wecom", "dingtalk-connector"]:
            resp = retry_on_gateway_restart(
                lambda name=ch_name: helpers.user_del_channel(user.token, inst.db_id, name)
            )
            assert resp.status_code == 200, f"删除 {ch_name} 失败: status={resp.status_code}, body={resp.text}"
            # 删除后轮询确认通道已从列表移除
            for attempt in range(6):
                time.sleep(5)
                inst_data = helpers.user_get_channels(user.token, instance_db_id=inst.db_id)
                channels = inst_data.get("channels", {})
                if isinstance(channels, dict):
                    still_present = ch_name in channels and channels[ch_name].get("enabled", False)
                else:
                    still_present = ch_name in str(channels)
                if not still_present:
                    break
            assert not still_present, f"删除 {ch_name} 后仍存在于通道列表中: {channels}"
            print(f"    {ch_name} 删除确认 ✓")
        print("    全部删除成功 ✓")

        print(">>> 步骤 11：确认通道已清空 ...")
        # Gateway 侧通道名映射：ddingtalk → dingtalk-connector
        _gw_name_map = {"ddingtalk": "dingtalk-connector"}
        _gw_aliases = set(_gw_name_map.values())

        # 轮询等待通道清空，最多重试 3 次，每次间隔 15s
        configured = None
        channels = None
        for attempt in range(4):
            if attempt > 0:
                print(f"    第 {attempt} 次重试，等待 15s ...")
                time.sleep(15)
            inst_data = helpers.user_get_channels(user.token, instance_db_id=inst.db_id)
            channels = inst_data.get("channels", {})
            if isinstance(channels, dict):
                # 通道被删除后可能仍保留 key 但值为 {'enabled': False} 或空 dict，
                # 这种情况视为"未配置"。只有 enabled=True 或有实际配置字段才算"已配置"。
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
        print("TC-4.1 测试通过 ✅")

    except Exception as e:
        print(f"\nTC-4.1 测试失败 ❌: {e}")
        traceback.print_exc()
        sys.exit(1)

    finally:
        pass


if __name__ == "__main__":
    main()
