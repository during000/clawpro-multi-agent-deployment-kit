"""
企业插件（管控端）管理 API 辅助函数

对应 controller/admin_plugins.go，覆盖企业插件"创建 → 下发 → 查询实例安装情况"
三个核心接口，用于验证企业插件在实例重装后可被再次下发：
  - POST /admin/plugins/create     创建企业插件（multipart/form-data，上传插件 zip）
  - POST /admin/plugins/distribute 批量下发企业插件到实例（JSON）
  - GET  /admin/plugins/instances  查询某插件的实例安装情况（重装后仍应能查到实例）

典型用法（验证重装后再次下发）：
    slug = f"e2e-plugin-{int(time.time())}"
    admin_create_plugin(admin.token, slug, "E2E 插件", "1.0.0")
    admin_distribute_plugin(admin.token, slug, "1.0.0", [inst.db_id])
    assert admin_plugin_find_instance(admin.token, slug, inst.db_id)  # 首次下发可查到
    # ... 重装实例 ...
    admin_distribute_plugin(admin.token, slug, "1.0.0", [inst.db_id])  # 再次下发
    assert admin_plugin_find_instance(admin.token, slug, inst.db_id)  # 重装后仍可查到
"""

import io
import json
import time
import zipfile

from helpers import config
from helpers.api import admin_client


def build_plugin_zip(slug, name, version, *, plugin_id=None, kind="",
                     description="E2E 集成测试插件", manifest_extra=None):
    """构造一个最小可用的插件 zip（内存字节流）。

    后端 validatePluginZip 以 openclaw.plugin.json 为锚点确定插件根目录，
    manifest 中 id 为必填、kind 仅允许 ""/"memory"/"context-engine"。这里生成
    {slug}/openclaw.plugin.json 结构，默认 plugin_id=slug、kind 为空。
    """
    if plugin_id is None:
        plugin_id = slug
    manifest = {
        "id": plugin_id,
        "name": name,
        "version": version,
        "description": description,
        "kind": kind,
        "format": "directory",
    }
    if manifest_extra:
        manifest.update(manifest_extra)
    buf = io.BytesIO()
    with zipfile.ZipFile(buf, "w", zipfile.ZIP_DEFLATED) as zf:
        zf.writestr(f"{slug}/openclaw.plugin.json", json.dumps(manifest))
    buf.seek(0)
    return buf.getvalue()


def admin_create_plugin(admin_token, slug, name, version, *, zip_data=None,
                        description="E2E 集成测试插件", **kwargs):
    """创建企业插件（multipart/form-data，含 zip 文件上传）。

    返回原始 Response 以便检查状态码；成功时响应体形如：
        {"ok": true, "id": 1, "slug": "...", "version": "1.0.0", "plugin_id": "..."}
    """
    if zip_data is None:
        zip_data = build_plugin_zip(slug, name, version, description=description)
    data = {
        "slug": slug,
        "name": name,
        "version": version,
        "description": description,
    }
    for k, v in kwargs.items():
        data[k] = v
    files = {"file": (f"{slug}-{version}.zip", zip_data, "application/zip")}
    return admin_client(admin_token).post(
        "/admin/plugins/create",
        data=data,
        files=files,
        timeout=60,
        expect=None, raw=True,
    )


def admin_distribute_plugin(
        admin_token, slug, version, instance_ids=None, *, select_all=False,
        statuses=None, group_ids=None, search=None):
    """批量下发企业插件到实例（JSON）。

    instance_ids 与 select_all=True 二选一；全选模式可按状态、用户组和 search 筛选。
    返回原始 Response；成功时响应体形如：
        {"ok": true, "task_id": 1, "version": "1.0.0"}
    """
    body = {"slug": slug, "version": version}
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
        "/admin/plugins/distribute",
        json=body,
        timeout=60,
        expect=None, raw=True,
    )


def admin_plugin_instances(admin_token, slug, **params):
    """查询某插件的实例安装情况（GET /admin/plugins/instances）。

    返回解析后的 dict，形如：
        {"instances": [...], "page": 1, "page_size": 500, "total": N}
    每个实例项含 instance_id（实例 DB 主键）、cvm_instance_id、status（安装状态）等。
    """
    params["slug"] = slug
    return admin_client(admin_token).get("/admin/plugins/instances", params=params)


def admin_plugin_find_instance(admin_token, slug, instance_db_id, **params):
    """在某插件的实例安装情况列表中查找指定实例。

    用于验证"重装后调用 /admin/plugins/instances 仍能查到这台实例"。
    命中返回该实例项 dict，未命中返回 None。
    """
    data = admin_plugin_instances(admin_token, slug, **params)
    for inst in data.get("instances", []):
        if inst.get("instance_id") == instance_db_id:
            return inst
    return None


# 下发已收敛（不再处于过渡态）的状态集合：
#   installed/failed/upgrade_failed/outdated 均为终态；installing 表示仍在下发中。
PLUGIN_SETTLED_STATUSES = ("installed", "failed", "upgrade_failed", "outdated")


def wait_plugin_instance_status(admin_token, slug, instance_db_id, target,
                                timeout=None, **params):
    """轮询等待某实例在插件实例列表中达到期望安装状态集合。

    target 为可接受状态的集合（如 ("installing", "installed")）。
    额外 **params 透传给 /admin/plugins/instances（如 search=ins-xxx 缩小范围）。
    命中返回该实例项 dict；超时未命中则抛 TimeoutError。
    """
    timeout = timeout or config.SKILL_POLL_TIMEOUT
    if isinstance(target, str):
        target = (target,)
    start = time.time()
    last = None
    while True:
        inst = admin_plugin_find_instance(admin_token, slug, instance_db_id, **params)
        if inst is not None:
            last = inst.get("status")
            if last in target:
                return inst
        if time.time() - start > timeout:
            raise TimeoutError(
                f"插件 {slug} 在 {timeout}s 内未在实例 {instance_db_id} 上达到 "
                f"{target}（当前 status={last}）"
            )
        time.sleep(config.POLL_INTERVAL)


def wait_plugin_distributed(admin_token, slug, instance_db_id, timeout=None, **params):
    """轮询等待某实例进入"已下发/下发中"状态（installing 或已收敛终态）。

    用于下发后确认实例已被纳入下发范围（status 离开 uninstalled）。
    返回命中的实例项 dict；超时未命中则抛 TimeoutError。
    """
    return wait_plugin_instance_status(
        admin_token, slug, instance_db_id,
        ("installing",) + PLUGIN_SETTLED_STATUSES,
        timeout=timeout, **params,
    )


def wait_plugin_settled(admin_token, slug, instance_db_id, timeout=None, **params):
    """轮询等待某实例的插件下发收敛到终态（installed/failed/...，不再 installing）。

    用于下发后等待下发任务跑完、释放 plugin_distribute 分布式锁，避免后续再次下发 409。
    """
    return wait_plugin_instance_status(
        admin_token, slug, instance_db_id, PLUGIN_SETTLED_STATUSES,
        timeout=timeout, **params,
    )
