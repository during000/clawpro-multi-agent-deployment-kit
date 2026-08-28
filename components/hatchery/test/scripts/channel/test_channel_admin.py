#!/usr/bin/env python3
"""
TC-3.1 通道管理（管理员视角）：预定义通道验证 + 启用/禁用切换

验证：
  1. 预定义通道存在性与属性
  2. 预定义通道删除保护
  3. 通道 Enabled 状态 toggle 翻转与还原

使用方式：
  export BASE_URL=http://134.175.254.166
  export SEED_ADMIN_TOKEN=xxx
  python3 test_channel_admin.py
"""

import os
import sys
import traceback

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

import helpers
from helpers import config
from helpers import (
    check_env, setup_admin,
    setup_user,
    extract_user_channel_ids,
    filter_site_visible_channels,
    get_field,
)
from helpers.api import seed

PREDEFINED_CHANNELS = {"openclaw-weixin", "wecom", "wecom_app", "feishu", "ddingtalk", "qqbot", "slack", "discord"}


def main():
    check_env()
    print()

    admin = setup_admin("ch-admin")
    user = None
    target_channel = "wecom_app"
    target_db_id = None
    initial_enabled = None
    toggled = False

    try:
        user = setup_user(admin.token, "ch-admin", instance_quota=0)

        # ══════════════════════════════════════════════════════════════
        # Part 1：预定义通道存在性 + 属性验证
        # ══════════════════════════════════════════════════════════════
        print(">>> 步骤 1：查询管理员通道列表 ...")
        channels = helpers.admin_get_channels(admin.token)
        channel_ids = set()
        for ch in channels:
            cid = get_field(ch, "ChannelID", "channel_id", default="")
            channel_ids.add(cid)

        expected_predefined = filter_site_visible_channels(PREDEFINED_CHANNELS)
        hidden_predefined = PREDEFINED_CHANNELS - expected_predefined
        for pid in expected_predefined:
            assert pid in channel_ids, f"缺少预定义通道 {pid}: {channel_ids}"
        for pid in hidden_predefined:
            assert pid not in channel_ids, f"当前站点不应显示预定义通道 {pid}: {channel_ids}"
        print(f"    包含 {len(expected_predefined)} 个当前站点可见预定义通道 ✓")

        print(">>> 步骤 2：断言预定义通道属性 ...")
        feishu_db_id = None
        for ch in channels:
            cid = get_field(ch, "ChannelID", "channel_id", default="")
            if cid in expected_predefined:
                custom = ch.get("Custom") or ch.get("custom", False)
                assert not custom, f"预定义通道 {cid} 的 Custom 应为 false"
                if cid == "feishu":
                    feishu_db_id = get_field(ch, "ID", "id")

        assert feishu_db_id, "未找到 feishu"
        print("    属性验证通过 ✓")

        # ══════════════════════════════════════════════════════════════
        # Part 2：用户可见性
        # ══════════════════════════════════════════════════════════════
        print(">>> 步骤 3：用户查询通道列表 ...")
        user_channels = helpers.user_get_channels(user.token)
        user_ch_ids = extract_user_channel_ids(user_channels)
        assert len(user_ch_ids) > 0, f"用户通道列表不应为空: {user_ch_ids}"
        print(f"    用户可见 {len(user_ch_ids)} 个通道 ✓")

        # ══════════════════════════════════════════════════════════════
        # Part 3：预定义通道删除保护
        # ══════════════════════════════════════════════════════════════
        print(">>> 步骤 4：尝试删除预定义通道 feishu ...")
        resp = helpers.admin_delete_channel(admin.token, feishu_db_id)
        assert resp.status_code in (403, 400), (
            f"删除预定义通道应被拒绝（403/400），实际: {resp.status_code} {resp.text}"
        )
        print(f"    返回 {resp.status_code} ✓（预定义通道受保护）")

        # ══════════════════════════════════════════════════════════════
        # Part 4：通道启用/禁用 Toggle
        # ══════════════════════════════════════════════════════════════
        print(f">>> 步骤 5：查询 {target_channel} 当前状态 ...")
        channels = helpers.admin_get_channels(config.SEED_ADMIN_TOKEN)
        for ch in channels:
            cid = get_field(ch, "ChannelID", "channel_id", default="")
            if cid == target_channel:
                target_db_id = get_field(ch, "ID", "id")
                initial_enabled = ch.get("Enabled", False)
                break
        assert target_db_id, f"未找到 {target_channel}"
        print(f"    {target_channel} 当前状态: Enabled={initial_enabled}  db_id={target_db_id}")

        print(f">>> 步骤 6：Toggle {target_channel} ...")
        resp = seed.post("/admin/channels/toggle", params={"id": target_db_id})
        assert resp.get("ok"), f"Toggle 失败: {resp}"
        toggled = True
        print("    Toggle 请求成功 ✓")

        print(">>> 步骤 7：验证状态已翻转 ...")
        channels = helpers.admin_get_channels(config.SEED_ADMIN_TOKEN)
        for ch in channels:
            cid = get_field(ch, "ChannelID", "channel_id", default="")
            if cid == target_channel:
                new_enabled = ch.get("Enabled", False)
                assert new_enabled != initial_enabled, (
                    f"Toggle 后状态应翻转: 期望 {not initial_enabled}，实际 {new_enabled}"
                )
                print(f"    Enabled: {initial_enabled} → {new_enabled} ✓")
                break

        print(f">>> 步骤 8：再次 Toggle（还原）...")
        resp = seed.post("/admin/channels/toggle", params={"id": target_db_id})
        assert resp.get("ok"), f"还原 Toggle 失败: {resp}"
        toggled = False
        print("    还原 Toggle 请求成功 ✓")

        print(">>> 步骤 9：验证状态已还原 ...")
        channels = helpers.admin_get_channels(config.SEED_ADMIN_TOKEN)
        for ch in channels:
            cid = get_field(ch, "ChannelID", "channel_id", default="")
            if cid == target_channel:
                restored = ch.get("Enabled", False)
                assert restored == initial_enabled, (
                    f"还原后状态应恢复: 期望 {initial_enabled}，实际 {restored}"
                )
                print(f"    Enabled 已还原为 {restored} ✓")
                break

        print()
        print("TC-3.1 测试通过 ✅")

    except Exception as e:
        print(f"\nTC-3.1 测试失败 ❌: {e}")
        traceback.print_exc()
        # 如果测试失败且通道已 toggle，尝试还原
        if toggled and target_db_id:
            try:
                seed.post("/admin/channels/toggle", params={"id": target_db_id})
                print("    （已还原通道状态）")
            except Exception:
                pass
        sys.exit(1)

    finally:
        pass


if __name__ == "__main__":
    main()
