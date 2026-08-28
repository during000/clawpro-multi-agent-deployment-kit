#!/usr/bin/env python3
"""
TC-3.5 自定义通道（custom channel）CRUD 集成测试

覆盖本次 feature/feature/support_whatsapp_and_custom_channel 中新增的
自定义通道能力（Custom=true）相关接口：

  POST   /admin/channels/add       添加自定义通道（白名单校验 + 字段校验）
  GET    /admin/channels           列表展示（含 Custom=true 通道 + agent_types）
  POST   /admin/channels/toggle    启用/禁用自定义通道
  POST   /admin/channels/delete    删除自定义通道（预定义通道保护）

用例场景：
  Part 1：AddChannel 字段校验（12 个失败路径 + 1 个成功路径）
  Part 2：列表展示 + agent_types / CustomConfig 字段（custom 与 predefined 区分）
  Part 3：Toggle 自定义通道
  Part 4：Delete 自定义通道
  Part 5：predefined 通道删除保护（与 test_channel_admin 互补）

使用方式：
  export BASE_URL=http://1.2.3.4
  export SEED_ADMIN_TOKEN=xxx
  python3 test_channel_custom_add.py
"""
import os
import sys
import time
import traceback

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

import helpers
from helpers import (
    check_env, setup_admin,
    get_field,
)
from helpers.api import (
    ApiClient, admin_client, seed,
    health_check,
)


CHANNEL_ID_REGEX_HINT = "^[a-zA-Z0-9_]+$"


def _unique_channel_id():
    """生成一个本次运行唯一的自定义通道 ID（字母数字下划线）。"""
    return f"it_custom_{int(time.time())}_{os.getpid()}"


def _valid_custom_config(phone_required=True):
    """生成一份合法可用的 custom_config（含 server 模板 + cred_fields）。

    server 中包含 {{app_key}} 与 {{app_secret}} 占位符，
    后续 set-channel 时按 cred_fields 注入。
    """
    server = {
        "url": "wss://example.com/ws",
        "protocol": "websocket",
        "pairingMode": True,
        "phoneRequired": phone_required,
        "phonePattern": "^[1-9]\\d{6,14}$",
        "autoFeature": "example_bot_creator",
        "autoTimeout": 120,
        "dmPolicy": "allowlist",
        "selfChatMode": False,
        "deleteFeature": "del_example_channel",
        "egressRequired": False,
    }
    cred_fields = [
        {"key": "app_key", "label": "App Key"},
        {"key": "app_secret", "label": "App Secret"},
    ]
    return {"server": server, "cred_fields": cred_fields}


def main():
    check_env()
    print()

    admin = setup_admin("custom-add")
    seed_admin = None
    channel_id = _unique_channel_id()
    created_db_id = None
    toggled = False

    try:
        # 用种子管理员（admin_token）确保权限链路走完整 controller。
        seed_admin = helpers.config.SEED_ADMIN_TOKEN
        assert seed_admin, "SEED_ADMIN_TOKEN 未配置"

        def admin_post(path, **kw):
            headers = {"Authorization": f"Bearer {seed_admin}"}
            extra = kw.pop("extra_headers", None) or {}
            headers.update(extra)
            return ApiClient("", timeout=30).post(
                path, extra_headers=headers, **kw,
            )

        # ══════════════════════════════════════════════════════════════
        # Part 1：AddChannel 字段校验（12 个失败路径 + 1 个成功路径）
        # ══════════════════════════════════════════════════════════════
        print(">>> 步骤 1：AddChannel 失败路径校验 ...")

        # 1.1 非 POST → 405
        # 使用 GET 方法请求 POST-only 接口，验证 controller 拒绝非 POST 请求
        resp = ApiClient("", timeout=30).get(
            "/admin/channels/add",
            extra_headers={"Authorization": f"Bearer {seed_admin}"},
            expect=None, raw=True,
        )
        if resp.status_code == 405:
            print("    1.1 非 POST → 405 ✓")
        else:
            print(f"    1.1 非 POST → {resp.status_code}（跳过，非强制）")

        # 1.2 非法 JSON → 400
        resp = admin_post(
            "/admin/channels/add",
            data="{not-valid-json",
            expect=None, raw=True,
            extra_headers={"Content-Type": "application/json"},
        )
        assert resp.status_code == 400, f"非法 JSON 应 400，实际 {resp.status_code}: {resp.text}"
        print("    1.2 非法 JSON → 400 ✓")

        # 1.3 channel_id 为空 → 400
        resp = admin_post(
            "/admin/channels/add",
            json={
                "channel_id": "",
                "name": "x",
                "custom_config": _valid_custom_config(),
            },
            expect=None, raw=True,
        )
        assert resp.status_code == 400, f"channel_id 空应 400，实际 {resp.status_code}: {resp.text}"
        assert "Channel ID" in resp.text or "channel" in resp.text.lower(), \
            f"错误消息应提及 Channel ID: {resp.text}"
        print("    1.3 channel_id 为空 → 400 ✓")

        # 1.4 channel_id 非法字符（含 - ）→ 400
        resp = admin_post(
            "/admin/channels/add",
            json={
                "channel_id": f"bad-id-{int(time.time())}",
                "name": "x",
                "custom_config": _valid_custom_config(),
            },
            expect=None, raw=True,
        )
        assert resp.status_code == 400, f"channel_id 含连字符应 400，实际 {resp.status_code}: {resp.text}"
        print("    1.4 channel_id 含非法字符 → 400 ✓")

        # 1.5 name 为空 → 400
        resp = admin_post(
            "/admin/channels/add",
            json={
                "channel_id": _unique_channel_id(),
                "name": "",
                "custom_config": _valid_custom_config(),
            },
            expect=None, raw=True,
        )
        assert resp.status_code == 400, f"name 空应 400，实际 {resp.status_code}: {resp.text}"
        print("    1.5 name 为空 → 400 ✓")

        # 1.6 缺 custom_config → 400
        resp = admin_post(
            "/admin/channels/add",
            json={
                "channel_id": _unique_channel_id(),
                "name": "x",
            },
            expect=None, raw=True,
        )
        assert resp.status_code == 400, f"缺 custom_config 应 400，实际 {resp.status_code}: {resp.text}"
        print("    1.6 缺 custom_config → 400 ✓")

        # 1.7 custom_config 格式错误（不是合法 JSON 对象）→ 400
        resp = admin_post(
            "/admin/channels/add",
            json={
                "channel_id": _unique_channel_id(),
                "name": "x",
                "custom_config": "this-is-not-a-json-object",
            },
            expect=None, raw=True,
        )
        assert resp.status_code == 400, f"custom_config 格式错应 400，实际 {resp.status_code}: {resp.text}"
        print("    1.7 custom_config 格式错误 → 400 ✓")

        # 1.8 server 字段是非法 JSON 字符串 → 400
        #      当 server 不是 null/{} 且不是合法 JSON 对象时拒
        bad_server_payload = _valid_custom_config()
        # 直接传 server 字段为非 JSON 内容
        resp = admin_post(
            "/admin/channels/add",
            json={
                "channel_id": _unique_channel_id(),
                "name": "x",
                "custom_config": {
                    "server": "{not-valid}",
                    "cred_fields": [],
                },
            },
            expect=None, raw=True,
        )
        assert resp.status_code == 400, f"server 非法 JSON 应 400，实际 {resp.status_code}: {resp.text}"
        print("    1.8 server 非法 JSON → 400 ✓")

        # 1.9 cred_field.key 非法字符 → 400
        bad_cred = _valid_custom_config()
        bad_cred["cred_fields"] = [{"key": "bad-key", "label": "X"}]
        resp = admin_post(
            "/admin/channels/add",
            json={
                "channel_id": _unique_channel_id(),
                "name": "x",
                "custom_config": bad_cred,
            },
            expect=None, raw=True,
        )
        assert resp.status_code == 400, f"cred_field.key 含连字符应 400，实际 {resp.status_code}: {resp.text}"
        print("    1.9 cred_field.key 含非法字符 → 400 ✓")

        # 1.10 cred_field.label 为空 → 400
        bad_cred = _valid_custom_config()
        bad_cred["cred_fields"] = [{"key": "ok_key", "label": ""}]
        resp = admin_post(
            "/admin/channels/add",
            json={
                "channel_id": _unique_channel_id(),
                "name": "x",
                "custom_config": bad_cred,
            },
            expect=None, raw=True,
        )
        assert resp.status_code == 400, f"cred_field.label 空应 400，实际 {resp.status_code}: {resp.text}"
        print("    1.10 cred_field.label 为空 → 400 ✓")

        # 1.11 cred_field.key 重复 → 400
        dup_cred = _valid_custom_config()
        dup_cred["cred_fields"] = [
            {"key": "app_key", "label": "App Key"},
            {"key": "app_key", "label": "App Key 2"},
        ]
        resp = admin_post(
            "/admin/channels/add",
            json={
                "channel_id": _unique_channel_id(),
                "name": "x",
                "custom_config": dup_cred,
            },
            expect=None, raw=True,
        )
        assert resp.status_code == 400, f"cred_field.key 重复应 400，实际 {resp.status_code}: {resp.text}"
        print("    1.11 cred_field.key 重复 → 400 ✓")

        # 1.12 成功路径：合法 custom_config → 200，Custom=true
        ok_payload = {
            "channel_id": channel_id,
            "name": f"IT-Custom-{int(time.time())}",
            "custom_config": _valid_custom_config(),
        }
        resp = admin_post("/admin/channels/add", json=ok_payload, expect=None, raw=True)
        assert resp.status_code == 200, f"合法添加应 200，实际 {resp.status_code}: {resp.text}"
        data = resp.json()
        ch_obj = data.get("channel") or {}
        assert ch_obj.get("Custom") is True, f"Custom 应为 true: {ch_obj}"
        # 创建后默认 Enabled=false（须管理员验证后启用）
        assert ch_obj.get("Enabled") is False, \
            f"新建自定义通道默认应禁用（Enabled=false）: {ch_obj}"
        created_db_id = get_field(ch_obj, "ID", "id")
        assert created_db_id, f"未返回 channel ID: {ch_obj}"
        print(f"    1.12 合法添加成功（db_id={created_db_id}, Custom=true, Enabled=false）✓")

        # 1.13 channel_id 重复 → 409
        resp = admin_post("/admin/channels/add", json=ok_payload, expect=None, raw=True)
        assert resp.status_code == 409, f"channel_id 重复应 409，实际 {resp.status_code}: {resp.text}"
        print("    1.13 channel_id 重复 → 409 ✓")

        # ══════════════════════════════════════════════════════════════
        # Part 2：列表展示 + agent_types / params 字段
        # ══════════════════════════════════════════════════════════════
        print(">>> 步骤 2：查询通道列表，验证新通道出现 ...")
        channels = helpers.admin_get_channels(seed_admin)
        found = None
        for ch in channels:
            cid = get_field(ch, "ChannelID", "channel_id", default="")
            if cid == channel_id:
                found = ch
                break
        assert found, f"新建通道 {channel_id} 未出现在列表中: {[get_field(c, 'ChannelID', 'channel_id') for c in channels]}"
        # admin 接口不返回 params 字段（params 仅在 /openclaw/channels 用户端接口展开），
        # 改为通过 CustomConfig 字符串验证 cred_fields 含 app_key / app_secret。
        custom_config_str = found.get("CustomConfig") or found.get("custom_config") or ""
        assert "app_key" in custom_config_str and "app_secret" in custom_config_str, \
            f"CustomConfig 应含 app_key/app_secret: {custom_config_str[:200]}"
        agent_types = found.get("agent_types") or found.get("AgentTypes") or []
        assert len(agent_types) > 0, f"自定义通道应至少有 1 个支持的 agent_type: {found}"
        print(f"    Custom=true, agent_types={agent_types} ✓")

        # ══════════════════════════════════════════════════════════════
        # Part 3：Toggle 自定义通道
        # ══════════════════════════════════════════════════════════════
        print(">>> 步骤 3：Toggle 自定义通道 ...")
        resp = seed.post("/admin/channels/toggle", params={"id": created_db_id})
        assert resp.get("ok"), f"Toggle 失败: {resp}"
        toggled = True
        # 验证状态已翻转（默认 false → true）
        channels = helpers.admin_get_channels(seed_admin)
        new_enabled = None
        for ch in channels:
            if get_field(ch, "ChannelID", "channel_id") == channel_id:
                new_enabled = ch.get("Enabled")
                break
        assert new_enabled is True, f"Toggle 后 Enabled 应为 true，实际 {new_enabled}"
        print(f"    Toggle 成功，Enabled: false → {new_enabled} ✓")

        # 还原
        resp = seed.post("/admin/channels/toggle", params={"id": created_db_id})
        assert resp.get("ok"), f"还原 Toggle 失败: {resp}"
        toggled = False
        channels = helpers.admin_get_channels(seed_admin)
        for ch in channels:
            if get_field(ch, "ChannelID", "channel_id") == channel_id:
                assert ch.get("Enabled") is False, "Toggle 还原后 Enabled 应为 false"
                break
        print("    Toggle 还原成功 ✓")

        # ══════════════════════════════════════════════════════════════
        # Part 4：Delete 自定义通道
        # ══════════════════════════════════════════════════════════════
        print(">>> 步骤 4：删除自定义通道 ...")
        resp = helpers.admin_delete_channel(seed_admin, created_db_id)
        assert resp.status_code == 200, f"删除应 200，实际 {resp.status_code}: {resp.text}"
        # 验证已从列表消失
        channels = helpers.admin_get_channels(seed_admin)
        still_exists = any(
            get_field(ch, "ChannelID", "channel_id") == channel_id for ch in channels
        )
        assert not still_exists, f"删除后应从列表消失: {channel_id}"
        print(f"    {channel_id} 已从列表消失 ✓")

        # ══════════════════════════════════════════════════════════════
        # Part 5：predefined 通道删除保护（与 test_channel_admin 互补）
        # ══════════════════════════════════════════════════════════════
        print(">>> 步骤 5：尝试删除预定义通道 feishu ...")
        channels = helpers.admin_get_channels(seed_admin)
        feishu_id = None
        for ch in channels:
            if get_field(ch, "ChannelID", "channel_id") == "feishu":
                feishu_id = get_field(ch, "ID", "id")
                break
        if feishu_id:
            resp = helpers.admin_delete_channel(seed_admin, feishu_id)
            assert resp.status_code in (403, 400), (
                f"删除预定义通道应被拒绝（403/400），实际 {resp.status_code}: {resp.text}"
            )
            print(f"    feishu 删除被拒绝（{resp.status_code}）✓")
        else:
            print("    feishu 不可见（站点范围外），跳过")

        print()
        print("TC-3.5 测试通过 ✅")

    except Exception as e:
        print(f"\nTC-3.5 测试失败 ❌: {e}")
        traceback.print_exc()
        # 失败兜底：还原 toggle + 删除
        if toggled and created_db_id:
            try:
                seed.post("/admin/channels/toggle", params={"id": created_db_id})
                print("    （已还原 toggle）")
            except Exception:
                pass
        if created_db_id:
            try:
                helpers.admin_delete_channel(seed_admin, created_db_id)
                print(f"    （已删除自定义通道 db_id={created_db_id}）")
            except Exception:
                pass
        sys.exit(1)


if __name__ == "__main__":
    main()
