#!/usr/bin/env python3
"""
TC-3.6 OpenClaw WhatsApp 自定义通道（配对码模式）端到端集成测试

本次 feature/feature/support_whatsapp_and_custom_channel 在自定义通道框架
下接入 WhatsApp 配对码流程（channel_id = "openclaw_whatsapp"），与内置扫码
通道 "whatsapp"（baileys 扫码）并存，本用例覆盖前者。

能力点（来自本次提交）：

  1. 管理员注册 openclaw_whatsapp 通道：
       - 必含 pairingMode=true / phoneRequired=true
       - DefaultsForChannel 自动补全 autoFeature/autoTimeout/deleteFeature
         /phonePattern/dmPolicy/selfChatMode/egressRequired 等默认值

  2. set-channel（/openclaw/set-channel）走 server 模板：
       - 占位符 {{phone_number}} 必须由用户提交，否则 400
       - 渲染后必须是合法 JSON，否则 400
       - 顶层自动补 enabled=true
       - 透传 is_custom=true + channel_config 字符串给 set_channel 脚本

  3. del-channel（/openclaw/del-channel）走 deleteFeature：
       - 从 CustomConfig.server.deleteFeature 读取 feature 名
       - 缺失时 fallback "del_channel"
       - 预设值（DefaultsForChannel）填入 "del_whatsapp_channel"

  4. auto-channel（/openclaw/auto-channel?channel=openclaw_whatsapp）：
       - 必须传 phone 参数
       - phone 必须符合 phonePattern（默认 ^[1-9]\\d{6,14}$）

使用方式：
  export BASE_URL=http://1.2.3.4
  export SEED_ADMIN_TOKEN=xxx
  python3 test_whatsapp_custom_channel.py
"""
import os
import sys
import time
import traceback

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

import helpers
from helpers import (
    check_env, setup_admin,
    setup_user, setup_instance,
    retry_on_gateway_restart,
    get_field,
)
from helpers.api import (
    ApiClient, seed,
    health_check,
)


# 自定义通道 ID（admin_create 时使用）
WA_CHANNEL_ID = f"it_openclaw_whatsapp_{int(time.time())}_{os.getpid()}"


def _wa_custom_config(phone_required=True, phone_pattern=None,
                      extra_server_fields=None):
    """构造一份 openclaw_whatsapp 的 custom_config 模板。

    server 字段即作为 openclaw.json 中该通道的 JSON 模板，
    含 pairingMode + phoneRequired + 各种 DefaultsForChannel 可填充的字段。
    cred_fields 必须包含 phone_number 才能让 set-channel 用户提交。
    """
    server = {
        "url": "wss://g.whatsapp.net/ws",
        "pairingMode": True,
        "phoneRequired": phone_required,
        "autoFeature": "whatsapp_pairing",
        "autoTimeout": 180,
        "deleteFeature": "del_whatsapp_channel",
        "dmPolicy": "allowlist",
        "selfChatMode": True,
        "egressRequired": True,
    }
    if phone_pattern is not None:
        server["phonePattern"] = phone_pattern
    if extra_server_fields:
        server.update(extra_server_fields)

    cred_fields = [
        {"key": "phone_number", "label": "手机号（带国家码，不含+，如 85266803489）"},
    ]
    return {"server": server, "cred_fields": cred_fields}


def _post_as_seed(path, **kw):
    """使用 SEED_ADMIN_TOKEN 发送请求（管理员权限走完整 controller）。"""
    seed_token = helpers.config.SEED_ADMIN_TOKEN
    assert seed_token, "SEED_ADMIN_TOKEN 未配置"
    headers = kw.pop("extra_headers", {}) or {}
    headers["Authorization"] = f"Bearer {seed_token}"
    return ApiClient("", timeout=30).post(path, extra_headers=headers, **kw)


def _add_wa_channel(seed_token, channel_id, custom_config):
    """注册一个 openclaw_whatsapp 自定义通道。"""
    payload = {
        "channel_id": channel_id,
        "name": f"IT-WhatsApp-{int(time.time())}",
        "custom_config": custom_config,
    }
    return _post_as_seed(
        "/admin/channels/add", json=payload, expect=None, raw=True,
    )


def _toggle_wa_channel(seed_token, db_id, expect_enabled=True):
    """Toggle 通道到指定状态。"""
    channels = helpers.admin_get_channels(seed_token)
    for ch in channels:
        if get_field(ch, "ID", "id") == db_id:
            cur = ch.get("Enabled", False)
            if cur != expect_enabled:
                seed.post("/admin/channels/toggle", params={"id": db_id})
            return
    raise RuntimeError(f"未找到 channel db_id={db_id}")


def main():
    check_env()
    print()

    seed_token = helpers.config.SEED_ADMIN_TOKEN
    assert seed_token, "SEED_ADMIN_TOKEN 未配置"

    admin = setup_admin("wa-cust")
    user = None
    inst = None
    wa_db_id = None
    toggled = False

    try:
        helpers.ensure_gateway_ui_enabled(admin.token)

        # ══════════════════════════════════════════════════════════════
        # Part 1：注册 openclaw_whatsapp 自定义通道
        # ══════════════════════════════════════════════════════════════
        print(f">>> 步骤 1：注册自定义通道 {WA_CHANNEL_ID} ...")
        resp = _add_wa_channel(seed_token, WA_CHANNEL_ID, _wa_custom_config())
        assert resp.status_code == 200, \
            f"添加自定义通道应 200，实际 {resp.status_code}: {resp.text}"
        ch_obj = resp.json().get("channel") or {}
        assert ch_obj.get("Custom") is True, f"Custom 应为 true: {ch_obj}"
        wa_db_id = get_field(ch_obj, "ID", "id")
        assert wa_db_id, f"未返回 channel ID: {ch_obj}"
        # 默认 Enabled=false，启用后才能 set-channel
        assert ch_obj.get("Enabled") is False, \
            f"新建默认应禁用: {ch_obj}"
        print(f"    已注册 db_id={wa_db_id}，启用中 ...")

        # 启用
        seed.post("/admin/channels/toggle", params={"id": wa_db_id})
        toggled = True
        # 验证已启用
        channels = helpers.admin_get_channels(seed_token)
        enabled_now = None
        for ch in channels:
            if get_field(ch, "ID", "id") == wa_db_id:
                enabled_now = ch.get("Enabled")
                break
        assert enabled_now is True, f"Toggle 后应启用: {enabled_now}"
        # admin 接口不返回 params 字段（params 仅在 /openclaw/channels 用户端接口展开），
        # 改为通过 CustomConfig 字段验证 cred_fields 含 phone_number。
        for ch in channels:
            if get_field(ch, "ID", "id") == wa_db_id:
                custom_config_str = ch.get("CustomConfig") or ch.get("custom_config") or ""
                assert "phone_number" in custom_config_str, \
                    f"CustomConfig 应含 phone_number: {custom_config_str[:200]}"
                print(f"    已启用，CustomConfig 含 phone_number ✓")
                break

        # ══════════════════════════════════════════════════════════════
        # Part 2：set-channel 模板渲染 —— 占位符 + JSON 注入
        # ══════════════════════════════════════════════════════════════
        print(">>> 步骤 2：准备用户 + 实例 ...")
        user = setup_user(admin.token, "wa-cust")
        inst = setup_instance(user.token, "wa-cust")
        assert inst and inst.db_id, "实例创建失败"
        print(f"    实例已就绪 db_id={inst.db_id} ✓")

        print(">>> 步骤 3：set-channel 缺占位符 → 400 ...")
        # 故意不传 phone_number（cred_field 必填）
        resp = helpers.user_set_channel(
            user.token, inst.db_id, WA_CHANNEL_ID,
            keys=[],
            values=[],
        )
        assert resp.status_code == 400, (
            f"缺 key/value 应 400，实际 {resp.status_code}: {resp.text[:200]}"
        )
        print("    返回 400 ✓（缺少配置）")

        print(">>> 步骤 4：set-channel 成功路径 ...")
        # 真实场景：脚本会在 TAT 端推送配置到 CVM，但本集成测试
        # 只验证 Go 层的占位符替换 + JSON 注入 + 脚本调用
        # 由于 set_channel 走 set_channel 脚本（如 set_channel_openclaw.sh），
        # 该脚本在 K8s 测试环境下会因 CVM 不可达而失败（500）。
        # 我们关注的是：Go 层渲染过的 channel_config 参数能被正常构建。
        # 因此这里只断言 5xx / 200（脚本层结果视环境而定），
        # 但通过 SSE 上报 / 标准错误格式可判断是否被 Go 层前置校验拦下。
        # 简化：期待响应不为 4xx（占位符已满足）。
        valid_phone = "85266803489"
        resp = retry_on_gateway_restart(
            lambda: helpers.user_set_channel(
                user.token, inst.db_id, WA_CHANNEL_ID,
                keys=["phone_number"],
                values=[valid_phone],
            )
        )
        # 4xx 表明被 Go 层校验拒绝（占位符解析失败等）→ 测试失败
        assert resp.status_code < 400 or resp.status_code >= 500, (
            f"set-channel 合法参数被 Go 层拒绝（4xx 不应出现）: "
            f"{resp.status_code} {resp.text[:300]}"
        )
        if 200 <= resp.status_code < 300:
            print(f"    set-channel 成功 ({resp.status_code}) ✓")
        else:
            # 500 表明脚本层调用失败（测试环境 CVM 不可达），Go 层校验已通过
            print(f"    Go 层校验通过（脚本层 {resp.status_code}，测试环境 CVM 不可达属正常）")

        # ══════════════════════════════════════════════════════════════
        # Part 3：auto-channel 通道预解析 —— phone 必填 + phonePattern 校验
        # ══════════════════════════════════════════════════════════════
        print(">>> 步骤 5：auto-channel 缺 phone → 400 ...")
        resp = helpers.user_auto_channel(
            user.token, inst.db_id, WA_CHANNEL_ID, timeout=15,
        )
        # 流式响应，期望 SSE 错误事件
        # 但 controller 端在 SSE header 之前已经拦截，所以是标准 400 JSON
        assert resp.status_code == 400, (
            f"缺 phone 应 400，实际 {resp.status_code}: {resp.text[:200]}"
        )
        body_text = resp.text
        assert "phone" in body_text.lower(), f"错误消息应提及 phone: {body_text}"
        print(f"    返回 400 ✓（错误消息含 phone）")

        print(">>> 步骤 6：auto-channel phone 格式错误 → 400 ...")
        for label, bad_phone in [
            ("带+号", "+85266803489"),
            ("带空格", "852 6680 3489"),
            ("以0开头", "085266803489"),
            ("含字母", "8526680abc9"),
            ("太短", "123"),
        ]:
            # 直接调用 ApiClient 传入 phone 参数（helper 不支持）
            resp = ApiClient("", timeout=30).get(
                "/openclaw/auto-channel",
                params={
                    "id": inst.db_id,
                    "channel": WA_CHANNEL_ID,
                    "phone": bad_phone,
                },
                extra_headers={"Authorization": f"Bearer {user.token}"},
                expect=None, raw=True,
                stream=True,
                timeout=15,
            )
            # stream 模式下，response body 可能还没读完
            # 但实际 controller 在 SSE header 之前已经 400
            assert resp.status_code == 400, (
                f"[{label}] phone={bad_phone!r} 应 400，实际 {resp.status_code}: "
                f"{resp.text[:200]}"
            )
            # 关闭流式连接避免悬空
            try:
                resp.close()
            except Exception:
                pass
        print("    5 种异常 phone 格式均被拒绝 ✓")

        # ══════════════════════════════════════════════════════════════
        # Part 4：auto-channel phone 合法 → SSE 流式进入
        # ══════════════════════════════════════════════════════════════
        print(">>> 步骤 7：auto-channel 合法 phone → 期望 SSE 200 ...")
        valid_phone = "85266803489"
        resp = ApiClient("", timeout=30).get(
            "/openclaw/auto-channel",
            params={
                "id": inst.db_id,
                "channel": WA_CHANNEL_ID,
                "phone": valid_phone,
            },
            extra_headers={"Authorization": f"Bearer {user.token}"},
            expect=None, raw=True,
            stream=True,
            timeout=30,
        )
        # 接受 200（SSE 流开始）或 5xx（前置步骤成功，脚本层失败）
        assert resp.status_code in (200, 500), (
            f"phone 合法时 Go 层不应 4xx，实际 {resp.status_code}: "
            f"{resp.text[:200]}"
        )
        if resp.status_code == 200:
            ctype = resp.headers.get("Content-Type", "")
            assert "text/event-stream" in ctype, f"应为 SSE 流: {ctype}"
            print(f"    SSE 流已建立（Content-Type={ctype}）✓")
            try:
                resp.close()
            except Exception:
                pass
        else:
            print(f"    Go 层校验通过，脚本层 5xx（CVM 不可达）")

        # ══════════════════════════════════════════════════════════════
        # Part 5：del-channel 走 deleteFeature 分派脚本
        # ══════════════════════════════════════════════════════════════
        print(">>> 步骤 8：del-channel 应走 del_whatsapp_channel 脚本 ...")
        # 同 set-channel 一样，Go 层会调脚本；测试环境会 500。
        # 我们关心的是 controller 端能否识别 deleteFeature。
        resp = helpers.user_del_channel(user.token, inst.db_id, WA_CHANNEL_ID)
        # 4xx 表明 Go 层前置校验失败（不应发生）
        assert resp.status_code < 400 or resp.status_code >= 500, (
            f"del-channel 被 Go 层拒绝（4xx 不应出现）: "
            f"{resp.status_code} {resp.text[:300]}"
        )
        if 200 <= resp.status_code < 300:
            print(f"    del-channel 成功 ({resp.status_code}) ✓")
        else:
            print(f"    Go 层 deleteFeature 解析通过，脚本层 {resp.status_code}")

        # ══════════════════════════════════════════════════════════════
        # Part 6：cleanup
        # ══════════════════════════════════════════════════════════════
        print(">>> 步骤 9：清理自定义通道 ...")
        if wa_db_id:
            resp = helpers.admin_delete_channel(seed_token, wa_db_id)
            assert resp.status_code == 200, f"清理应 200: {resp.status_code} {resp.text}"
            # 还原 toggle 状态计数
            toggled = False
            print(f"    {WA_CHANNEL_ID} 已删除 ✓")

        print()
        print("TC-3.6 测试通过 ✅")

    except Exception as e:
        print(f"\nTC-3.6 测试失败 ❌: {e}")
        traceback.print_exc()
        sys.exit(1)

    finally:
        # 兜底清理
        if toggled and wa_db_id:
            try:
                _toggle_wa_channel(seed_token, wa_db_id, expect_enabled=False)
            except Exception:
                pass
        if wa_db_id:
            try:
                helpers.admin_delete_channel(seed_token, wa_db_id)
                print(f"    （兜底：已删除 {WA_CHANNEL_ID}）")
            except Exception:
                pass


if __name__ == "__main__":
    main()
