"""
通道管理 —— 管理员接口 + 用户侧接口
"""

from helpers.api import admin_client, anon, user_client


SITE_SCOPED_CHANNELS = {
    "slack": {"overseas"},
    "discord": {"overseas"},
    "lark": {"overseas"},
    "line": {"overseas"},
}


def is_overseas_site():
    """Return whether the tested deployment is an overseas site."""
    try:
        data = anon.get("/site", expect=200)
    except Exception as exc:
        print(f"    ⚠ 获取 /site 失败，按国内站点处理: {exc}")
        return False
    return bool(data.get("is_overseas"))


def filter_site_visible_channels(channel_ids):
    """Filter expected selectable channels by the current site scope."""
    overseas = is_overseas_site()
    site = "overseas" if overseas else "domestic"
    return {
        channel_id
        for channel_id in channel_ids
        if site in SITE_SCOPED_CHANNELS.get(channel_id, {"domestic", "overseas"})
    }


# ═══════════════════════════════════════════════════════════════════
# 管理员接口
# ═══════════════════════════════════════════════════════════════════

def admin_get_channels(admin_token):
    """获取管理员通道列表"""
    data = admin_client(admin_token).get("/admin/channels")
    return data.get("channels", [])


def admin_toggle_channel(admin_token, channel_db_id):
    """启用/禁用通道"""
    return admin_client(admin_token).post(
        "/admin/channels/toggle", params={"id": channel_db_id},
    )


def admin_delete_channel(admin_token, channel_db_id):
    """删除通道（返回原始 Response 以便检查状态码）"""
    return admin_client(admin_token).post(
        "/admin/channels/delete", params={"id": channel_db_id},
        expect=None, raw=True,
    )


# ═══════════════════════════════════════════════════════════════════
# 用户侧接口
# ═══════════════════════════════════════════════════════════════════

def user_get_channels(user_token, instance_db_id=None, agent_id=None):
    """用户侧查询通道列表"""
    params = {}
    if instance_db_id is not None:
        params["id"] = instance_db_id
    if agent_id is not None:
        params["agent_id"] = agent_id
    return user_client(user_token).get("/openclaw/channels", params=params)


def user_set_channel(user_token, instance_db_id, channel, keys, values):
    """配置通道凭证（返回原始 Response 以便检查状态码）

    Go 后端使用 r.Form["key"] / r.Form["value"] 读取多值字段，按索引
    一一配对。正确的表单编码是 key/value 成对交替：
        key=app_id&value=v1&key=app_secret&value=v2
    requests 库的 data 参数传入列表-of-tuple 即可保持顺序。
    """
    fields = [
        ("id", str(instance_db_id)),
        ("channel", channel),
    ]
    for k, v in zip(keys, values):
        fields.append(("key", k))
        fields.append(("value", v))

    return user_client(user_token).post(
        "/openclaw/set-channel",
        data=fields,
        timeout=60,
        expect=None, raw=True,
    )


def user_del_channel(user_token, instance_db_id, channel):
    """删除通道配置（返回原始 Response 以便检查状态码）"""
    return user_client(user_token).post(
        "/openclaw/del-channel",
        data={"id": str(instance_db_id), "channel": channel},
        expect=None, raw=True,
    )


def user_auto_channel(user_token, instance_db_id, channel, timeout=180):
    """调用 auto-channel SSE 流，返回 Response（stream=True）"""
    return user_client(user_token).get(
        "/openclaw/auto-channel",
        params={"id": instance_db_id, "channel": channel},
        expect=None, raw=True,
        stream=True,
        timeout=timeout,
    )


# ═══════════════════════════════════════════════════════════════════
# 辅助函数
# ═══════════════════════════════════════════════════════════════════

def extract_user_channel_ids(user_channels):
    """从用户通道响应中提取通道 ID 列表"""
    # 用户侧接口直接返回数组，或包在 data/channels 中
    if isinstance(user_channels, list):
        ch_list = user_channels
    elif isinstance(user_channels, dict):
        ch_list = user_channels.get("data", user_channels.get("channels", []))
    else:
        return []
    return [c.get("ChannelID") or c.get("channel_id", "") for c in ch_list]
