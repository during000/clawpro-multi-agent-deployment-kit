"""
企业 MCP（管控端）管理 API 辅助函数

对应 controller/admin_mcp.go / admin_mcp_distribute.go / admin_mcp_instances.go，
覆盖企业 MCP "新增 → 下发 → 查询实例安装情况" 三个核心接口，用于验证企业 MCP
在实例重装后可被再次下发：
  - POST /admin/mcp/create     新增 MCP（JSON，含 service_id / transport_type / config_json）
  - POST /admin/mcp/distribute 批量下发 MCP 到实例（JSON）
  - GET  /admin/mcp/instances  查询某 MCP 的实例安装情况（重装后仍应能查到实例）

典型用法（验证重装后再次下发）：
    service_id = f"e2e-mcp-{int(time.time())}"
    admin_create_mcp(admin.token, service_id)              # 默认创建 1.0.0 版本
    admin_distribute_mcp(admin.token, service_id, "1.0.0", [inst.db_id])
    assert admin_mcp_find_instance(admin.token, service_id, inst.db_id)  # 首次下发可查到
    # ... 重装实例 ...
    admin_distribute_mcp(admin.token, service_id, "1.0.0", [inst.db_id])  # 再次下发
    assert admin_mcp_find_instance(admin.token, service_id, inst.db_id)   # 重装后仍可查到
"""

import json
import time

from helpers import config
from helpers.api import admin_client


def build_mcp_config_json(url="http://localhost:8000/sse", transport="sse",
                          timeout=60):
    """
    构造一个最小的 MCP config_json 字符串（扁平结构）。
    """
    return json.dumps({
        "transportType": transport,
        "url": url,
        "timeout": timeout,
    })


def admin_create_mcp(admin_token, service_id, name="", *, transport_type="sse",
                     config_json=None, description="E2E 集成测试 MCP",
                     usage_doc_md="", tool_doc_md="", **kwargs):
    """新增企业 MCP（JSON）。创建后自动生成 1.0.0 版本。

    返回原始 Response 以便检查状态码；成功时（201）响应体形如：
        {"id": 1, "service_id": "...", "latest_version": "1.0.0", "version_id": 1}
    """
    if config_json is None:
        config_json = build_mcp_config_json()
    body = {
        "service_id": service_id,
        "name": name or service_id,
        "description": description,
        "transport_type": transport_type,
        "config_json": config_json,
        "usage_doc_md": usage_doc_md,
        "tool_doc_md": tool_doc_md,
    }
    for k, v in kwargs.items():
        body[k] = v
    return admin_client(admin_token).post(
        "/admin/mcp/create",
        json=body,
        timeout=60,
        expect=None, raw=True,
    )


def admin_distribute_mcp(
        admin_token, service_id, version, instance_ids=None, *, select_all=False,
        statuses=None, group_ids=None, search=None):
    """批量下发企业 MCP 到实例（JSON）。

    instance_ids 与 select_all=True 二选一；全选模式可按状态、用户组和 search 筛选。
    返回原始 Response；成功时（202）响应体形如：
        {"task_id": 1, "total": 1, "per_instance": [...], "warnings": [...]}
    """
    body = {"service_id": service_id, "version": version}
    if instance_ids is not None:
        body["instance_ids"] = instance_ids
    if select_all:
        body["select_all"] = True
    if statuses is not None:
        body["statuses"] = statuses
    if group_ids is not None:
        body["group_ids"] = group_ids
    if search is not None:
        body["search"] = search
    return admin_client(admin_token).post(
        "/admin/mcp/distribute",
        json=body,
        timeout=60,
        expect=None, raw=True,
    )


def admin_mcp_instances(admin_token, service_id, **params):
    """查询某 MCP 的实例安装情况（GET /admin/mcp/instances）。

    返回解析后的 dict，形如：
        {"instances": [...], "page": 1, "page_size": 20, "total": N}
    每个实例项含 instance_id（实例 DB 主键）、cvm_instance_id、status（安装状态）等。
    """
    params["service_id"] = service_id
    return admin_client(admin_token).get("/admin/mcp/instances", params=params)


def admin_mcp_find_instance(admin_token, service_id, instance_db_id, **params):
    """在某 MCP 的实例安装情况列表中查找指定实例。

    用于验证"重装后调用 /admin/mcp/instances 仍能查到这台实例"。
    命中返回该实例项 dict，未命中返回 None。
    """
    data = admin_mcp_instances(admin_token, service_id, **params)
    for inst in data.get("instances", []):
        if inst.get("instance_id") == instance_db_id:
            return inst
    return None


# 下发已收敛（不再处于过渡态）的状态集合：
#   installed/outdated/failed 均为终态；installing 表示仍在下发中。
MCP_SETTLED_STATUSES = ("installed", "outdated", "failed")


def wait_mcp_instance_status(admin_token, service_id, instance_db_id, target,
                             timeout=None, **params):
    """轮询等待某实例在 MCP 实例列表中达到期望安装状态集合。

    target 为可接受状态的集合（如 ("installing", "installed")）。
    额外 **params 透传给 /admin/mcp/instances（如 search=ins-xxx 缩小范围）。
    命中返回该实例项 dict；超时未命中则抛 TimeoutError。
    """
    timeout = timeout or config.SKILL_POLL_TIMEOUT
    if isinstance(target, str):
        target = (target,)
    start = time.time()
    last = None
    while True:
        inst = admin_mcp_find_instance(admin_token, service_id, instance_db_id, **params)
        if inst is not None:
            last = inst.get("status")
            if last in target:
                return inst
        if time.time() - start > timeout:
            raise TimeoutError(
                f"MCP {service_id} 在 {timeout}s 内未在实例 {instance_db_id} 上达到 "
                f"{target}（当前 status={last}）"
            )
        time.sleep(config.POLL_INTERVAL)


def wait_mcp_distributed(admin_token, service_id, instance_db_id, timeout=None, **params):
    """轮询等待某实例进入"已下发/下发中"状态（installing 或已收敛终态）。

    用于下发后确认实例已被纳入下发范围（status 离开 uninstalled）。
    返回命中的实例项 dict；超时未命中则抛 TimeoutError。
    """
    return wait_mcp_instance_status(
        admin_token, service_id, instance_db_id,
        ("installing",) + MCP_SETTLED_STATUSES,
        timeout=timeout, **params,
    )


def wait_mcp_settled(admin_token, service_id, instance_db_id, timeout=None, **params):
    """轮询等待某实例的 MCP 下发收敛到终态（installed/outdated/failed，不再 installing）。

    用于下发后等待下发任务跑完、释放 mcp_distribute 分布式锁，避免后续再次下发 409。
    """
    return wait_mcp_instance_status(
        admin_token, service_id, instance_db_id, MCP_SETTLED_STATUSES,
        timeout=timeout, **params,
    )
