"""
实例管理 + 实例 setup 脚手架
"""

import time
from dataclasses import dataclass

from helpers import config
from helpers.api import user_client, get_field


# ═══════════════════════════════════════════════════════════════════
# 上下文数据类
# ═══════════════════════════════════════════════════════════════════

@dataclass
class InstanceContext:
    """实例上下文"""
    db_id: int
    instance_id: str
    user_token: str
    gateway_url: str | None = None
    gateway_token: str | None = None


def create_instance(user_token, name, agent_type=None, role_id=None, group_id=None, image_id=None):
    """创建实例

    image_id：可选，**内部测试隐藏参数**，仅当部署所在腾讯云账号 UIN 命中后端白名单时生效；
    其他部署上传该参数会被静默忽略，行为与未传一致。用于测试 "先用低版本镜像创建实例、再触发升级" 等场景。

    ⚠️ 重要：使用 image_id 时，调用方必须显式传入与该镜像匹配的 agent_type。
    后端会强校验 image.agent_type == request.agent_type，不一致会直接返回 400。
    例如：image_id="img-b52f7vd0"（openclaw-4.23）必须搭配 agent_type="openclaw"。
    """
    data = {"name": name, "agent_type": agent_type or config.AGENT_TYPE}
    if role_id is not None:
        data["role_id"] = str(role_id)
    if group_id is not None:
        data["group_id"] = str(group_id)
    if image_id is not None:
        data["image_id"] = str(image_id)
    return user_client(user_token).post("/openclaw/create", data=data, timeout=60)


def delete_instance(user_token, instance_db_id):
    """删除实例"""
    return user_client(user_token).post(
        "/openclaw/delete", data={"id": str(instance_db_id)}, timeout=60,
    )


def get_instance_status(user_token, instance_db_id):
    """查询实例状态。

    后端契约：实例记录完全不存在（或不属于本人）时返回 404。为兼容调用方
    既有的「空状态 = 记录已消失」判定语义，此处把 404 归一化为空状态对象。
    软删实例仍返回 200 + status="destroyed"，不走这条兜底。
    """
    resp = user_client(user_token).get(
        "/openclaw/status", params={"id": instance_db_id},
        expect=None, raw=True,
    )
    if resp.status_code == 404:
        return {"status": "", "label": "", "tooltip": "", "actions": [],
                "transient": False}
    assert resp.status_code == 200, (
        f"/openclaw/status 期望 200/404，实际 {resp.status_code}: {resp.text}"
    )
    return resp.json()


def list_instances(user_token, page=1, page_size=30):
    """列出实例"""
    return user_client(user_token).get(
        "/openclaw/list", params={"page": page, "page_size": page_size},
    )


def wait_instance_ready(user_token, instance_db_id, timeout=None):
    """
    轮询等待实例就绪，返回 status_data。
    超时或异常终态抛出异常。
    """
    timeout = timeout or config.POLL_TIMEOUT
    start = time.time()
    last_status = None

    while True:
        elapsed = time.time() - start
        if elapsed > timeout:
            raise TimeoutError(
                f"实例 {instance_db_id} 在 {timeout}s 内未就绪，最后状态: {last_status}"
            )

        status_data = get_instance_status(user_token, instance_db_id)
        status = status_data.get("status", "unknown")
        label = status_data.get("label", "")

        if status != last_status:
            print(f"    [{int(elapsed)}s] 状态: {status} ({label})", flush=True)
        last_status = status

        if status == "running" and not status_data.get("transient", True):
            return status_data

        if status in ("create_failed", "stopped", "destroyed", "load_failed"):
            raise RuntimeError(
                f"实例 {instance_db_id} 异常终态: {status} - "
                f"{status_data.get('tooltip', '')}"
            )

        time.sleep(config.POLL_INTERVAL)



def get_instance_db_id(user_token, instance_id=None):
    """
    从实例列表中获取数据库自增 ID。
    如果传了 instance_id（形如 ins-xxx），按它匹配；否则取最新的。
    """
    data = list_instances(user_token)
    instances = data.get("instances", [])
    assert instances, "实例列表为空"

    if instance_id:
        for inst in instances:
            iid = get_field(inst, "instance_id", "InstanceId")
            if iid == instance_id:
                return get_field(inst, "id", "ID")
    return get_field(instances[0], "id", "ID")


def check_gateway_access(user_token, instance_db_id):
    """
    检查 Gateway 是否可访问。
    返回: dict (accessible, port, securityGroupIds, message)
    若请求失败返回 None。
    """
    resp = user_client(user_token).get(
        "/openclaw/check-gateway-access",
        params={"id": instance_db_id},
        timeout=60,
        expect=None,
        raw=True,
    )
    if resp.status_code != 200:
        return None
    return resp.json()


def get_gateway_connection(user_token, instance_db_id, network_type="public"):
    """
    获取 Gateway 连接信息（URL + Token）。
    返回: dict (gatewayUI, token)
    """
    return user_client(user_token).post(
        "/openclaw/set-gateway-ui",
        data={"id": str(instance_db_id), "network_type": network_type},
        timeout=60,
    )


def wait_gateway_ready(user_token, instance_db_id, timeout=None, poll_interval=None):
    """
    轮询等待 Gateway 可访问。
    在实例状态为 running 之后调用，确保 Gateway 端口可通 + 服务就绪。
    返回: gateway 连接信息 dict (gatewayUI, token)
    """
    timeout = timeout or config.POLL_TIMEOUT
    poll_interval = poll_interval or config.POLL_INTERVAL
    start = time.time()
    last_reason = ""

    while True:
        elapsed = time.time() - start
        if elapsed > timeout:
            raise TimeoutError(
                f"Gateway 就绪超时（{timeout}s），最后原因: {last_reason}"
            )

        try:
            data = check_gateway_access(user_token, instance_db_id)
            if data is None:
                last_reason = "check-gateway-access 返回非 200"
            elif data.get("accessible"):
                return get_gateway_connection(user_token, instance_db_id)
            else:
                last_reason = data.get("message", "accessible=false")
        except Exception as e:
            last_reason = str(e)

        print(f"    等待 Gateway 就绪... ({elapsed:.0f}s, {last_reason})", flush=True)
        time.sleep(poll_interval)


# ═══════════════════════════════════════════════════════════════════
# 脚手架：实例 setup
# ═══════════════════════════════════════════════════════════════════

def setup_instance(user_token, scenario, group_id=None):
    """创建实例 + 等待就绪 + 等待 Gateway 就绪"""
    name = f"{config.INSTANCE_NAME_PREFIX}{scenario}-{int(time.time())}"
    print(f">>> 创建实例: {name} ...")

    create_data = create_instance(user_token, name, group_id=group_id)
    assert create_data.get("ok"), f"创建实例失败: {create_data}"

    instance_id = create_data.get("instance_id", "")
    db_id = get_instance_db_id(user_token, instance_id)

    print(f">>> 等待实例就绪 (db_id={db_id}) ...")
    wait_instance_ready(user_token, db_id)
    print(f"    实例就绪 ✓  db_id={db_id}, instance_id={instance_id}")

    print(">>> 等待 Gateway 就绪 ...")
    gateway_conn = wait_gateway_ready(user_token, db_id)
    gateway_url = gateway_conn.get("gatewayUI", "")
    gateway_token = gateway_conn.get("token", "")
    print(f"    Gateway 就绪 ✓  url={gateway_url[:60]}...")

    # 等待 gateway 服务完全稳定，避免后续操作触发重启时 systemd rate limiting
    print("    等待 Gateway 稳定 (30s) ...")
    time.sleep(30)

    return InstanceContext(
        db_id=db_id,
        instance_id=instance_id,
        user_token=user_token,
        gateway_url=gateway_url,
        gateway_token=gateway_token,
    )
