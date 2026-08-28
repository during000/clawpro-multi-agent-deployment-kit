"""
Hermes 专用辅助函数

包含:
- ensure_hermes_image: 确保 hermes 镜像已启用
- setup_hermes_instance: 创建 hermes 实例 + 等待就绪
- verify_hermes_service: 验证 hermes 服务可用性
"""

import sys
import time

from helpers import config
from helpers.api import seed, user_client
from helpers.instance import (
    InstanceContext,
    create_instance,
    get_instance_db_id,
    wait_instance_ready,
    wait_gateway_ready,
)

HERMES_AGENT_TYPE = "hermes"

# Hermes 支持的通道白名单（不含 wecom_app）
# discord / lark 为 overseas-only，实际可见性由 filter_site_visible_channels 按站点过滤
HERMES_WHITELIST_CHANNELS = {"openclaw-weixin", "wecom", "feishu", "ddingtalk", "qqbot", "slack", "discord", "lark", "line"}


def ensure_hermes_agent_type_enabled():
    """通过 admin 接口确保 hermes agent type 在站点配置中已启用（未被禁用）"""
    print(">>> 确保 hermes agent type 已启用 ...")
    resp = seed.post(
        "/admin/agent-types/enabled",
        data={"agent_type": HERMES_AGENT_TYPE, "enabled": "true"},
        raw=True,
    )
    if resp.status_code == 200:
        data = resp.json()
        if data.get("enabled"):
            print("    Hermes agent type 已启用 ✓")
        else:
            raise RuntimeError(f"启用 hermes agent type 失败: {data}")
    else:
        raise RuntimeError(
            f"POST /admin/agent-types/enabled 失败 ({resp.status_code}): {resp.text}"
        )


def ensure_hermes_image():
    """通过 admin 接口确保 hermes 镜像已启用"""
    print(">>> 确保 hermes 镜像已启用 ...")
    resp = seed.get("/admin/images", raw=True)
    if resp.status_code != 200:
        raise RuntimeError(f"GET /admin/images 失败 ({resp.status_code})")

    data = resp.json()
    images = data.get("images", [])

    # 找到 hermes 类型的镜像
    hermes_images = [img for img in images if img.get("agent_type") == HERMES_AGENT_TYPE]
    if not hermes_images:
        raise RuntimeError(
            "系统中未找到 hermes 镜像，请先导入: POST /admin/images/import with agent_type=hermes"
        )

    # 检查是否有已启用的
    enabled = [img for img in hermes_images if img.get("enabled")]
    if enabled:
        img = enabled[0]
        print(f"    已启用: id={img.get('id')}, image_id={img.get('image_id')}, "
              f"version={img.get('agent_version')}")
        return

    # 启用第一个 hermes 镜像
    target = hermes_images[0]
    db_id = target.get("id") or target.get("ID")
    print(f"    启用 hermes 镜像: id={db_id}, image_id={target.get('image_id')} ...")

    resp = seed.post("/admin/images/enable", params={"id": db_id}, raw=True)
    if resp.status_code != 200:
        raise RuntimeError(f"启用 hermes 镜像失败 ({resp.status_code}): {resp.text}")

    print("    Hermes 镜像启用成功 ✓")


def setup_hermes_instance(user_token, scenario, group_id=None):
    """创建 hermes 实例 + 等待就绪 + 等待 Gateway 就绪

    与 setup_instance 类似，但强制 agent_type="hermes" 并在创建前确保镜像已启用
    且 agent type 在站点配置中未被禁用。

    注意：ensure_hermes_agent_type_enabled 必须紧贴 create_instance 调用，
    避免被并发的 POST /admin/config（全量 Save）覆盖 disabled_agent_types。
    """
    ensure_hermes_image()

    # 紧贴 create_instance 前调用，缩小被并发 Save(&config) 覆盖的窗口
    ensure_hermes_agent_type_enabled()

    name = f"{config.INSTANCE_NAME_PREFIX}hermes-{scenario}-{int(time.time())}"
    print(f">>> 创建 hermes 实例: {name} ...")

    # 若仍因并发覆盖导致 403，最多重试一次
    for attempt in range(2):
        create_data = create_instance(user_token, name, agent_type=HERMES_AGENT_TYPE, group_id=group_id)
        if create_data.get("ok"):
            break
        err_msg = str(create_data)
        if "暂不可创建" in err_msg and attempt == 0:
            print("    ⚠ 疑似并发竞态导致 agent type 被覆盖为禁用，重新启用后重试 ...")
            ensure_hermes_agent_type_enabled()
            continue
        assert False, f"创建 hermes 实例失败: {create_data}"

    instance_id = create_data.get("instance_id", "")
    db_id = get_instance_db_id(user_token, instance_id)

    print(f">>> 等待 hermes 实例就绪 (db_id={db_id}) ...")
    wait_instance_ready(user_token, db_id)
    print(f"    实例就绪 ✓  db_id={db_id}, instance_id={instance_id}")

    # 已知 race：实例 status=running 后，hatchery 仍需异步执行
    # detect_hermes_install.sh 把真实的 RuntimeUser（agentuser）写回 DB；
    # 这一步在 agent_checker 周期任务里跑（一次 TAT 调用，正常 5~15s 完成）。
    # 此前若立即触发依赖 instance.RuntimeUser 的接口（典型如 set-gateway-ui），
    # hatchery 会拿到 RuntimeUser="" → fallback root，脚本里 $HOME=/root
    # 找不到 ~agentuser/.local/bin/hermes，报 "hermes not found"。
    #
    # 这里多等一会儿，给 detectAndSaveRuntimeUser 留足时间窗。
    _RUNTIME_USER_SETTLE_SECS = 10
    print(f">>> 等待 runtime user 切换稳定 ({_RUNTIME_USER_SETTLE_SECS}s) ...")
    time.sleep(_RUNTIME_USER_SETTLE_SECS)

    print(">>> 等待 Gateway 就绪 ...")
    gateway_conn = wait_gateway_ready(user_token, db_id)
    gateway_url = gateway_conn.get("gatewayUI", "")
    gateway_token = gateway_conn.get("token", "")
    print(f"    Gateway 就绪 ✓  url={gateway_url[:60]}...")

    return InstanceContext(
        db_id=db_id,
        instance_id=instance_id,
        user_token=user_token,
        gateway_url=gateway_url,
        gateway_token=gateway_token,
    )


def verify_hermes_service(user_token, instance_db_id, timeout=180):
    """验证 hermes 服务可用性（通过 check-openclaw-port 接口）

    Hermes 实例也通过 check-openclaw-port 接口检查服务状态。
    返回的 data 中通常包含 gateway / update / channelSummary 三段（与
    scripts/check_service_hermes.sh 输出契约一致）。
    """
    print(f">>> 验证 hermes 服务可用性 (db_id={instance_db_id}) ...")
    start = time.time()

    while True:
        elapsed = time.time() - start
        if elapsed > timeout:
            raise TimeoutError(f"Hermes 服务在 {timeout}s 内未就绪")

        resp = user_client(user_token).get(
            "/openclaw/check-openclaw-port",
            params={"id": instance_db_id},
            timeout=120,
            expect=None,
            raw=True,
        )

        if resp.status_code == 200:
            data = resp.json()
            if data.get("running"):
                print(f"    Hermes 服务可用 ✓ (耗时 {int(elapsed)}s)")
                return data

        print(f"    [{int(elapsed)}s] 服务未就绪，重试...", flush=True)
        time.sleep(config.POLL_INTERVAL)


def expect_hermes_channel_connected(user_token, instance_db_id, min_enabled=1, timeout=60):
    """轮询 channelSummary.enabled，软检查 hermes-gateway 是否真的把通道拉起来了。

    与 verify_hermes_service 不同：
      - verify_hermes_service 只检查服务进程在跑（不能证明通道连接成功）
      - 本函数额外要求 channelSummary.enabled >= min_enabled

    若 channelSummary 字段不存在（旧版 check_service_hermes.sh），不强制断言，
    仅打印警告，避免给老环境引入 flakiness。
    """
    print(f">>> 验证 hermes channel 已连接 (db_id={instance_db_id}, min_enabled={min_enabled}) ...")
    start = time.time()
    last_summary = None
    has_summary_field = False

    while time.time() - start < timeout:
        elapsed = time.time() - start
        resp = user_client(user_token).get(
            "/openclaw/check-openclaw-port",
            params={"id": instance_db_id},
            timeout=60,
            expect=None,
            raw=True,
        )
        if resp.status_code == 200:
            data = resp.json()
            summary = data.get("channelSummary")
            if summary is not None:
                has_summary_field = True
                last_summary = summary
                enabled = int(summary.get("enabled", 0) or 0)
                if enabled >= min_enabled:
                    print(f"    channel enabled={enabled}/{summary.get('total', '?')} ✓ "
                          f"(耗时 {int(elapsed)}s)")
                    return summary
        time.sleep(config.POLL_INTERVAL)

    if not has_summary_field:
        # 老版本 check_service_hermes.sh 不输出 channelSummary，软跳过
        print("    ⚠ channelSummary 字段缺失，无法校验 channel 连接状态（已软跳过）")
        return None

    # 有 channelSummary 字段但始终未达标 → 抛错（明显的 hermes 通道连接失败）
    raise AssertionError(
        f"Hermes 通道在 {timeout}s 内未达到 enabled >= {min_enabled}，"
        f"最后一次 summary={last_summary}"
    )
