"""
插件库管理 API 辅助函数
"""

import time

from helpers.api import admin_client


def admin_create_plugin(admin_token, slug, name, version, **kwargs):
    """创建插件（multipart/form-data）"""
    client = admin_client(admin_token)
    data = {
        "slug": slug,
        "name": name,
        "version": version,
    }
    for k, v in kwargs.items():
        data[k] = v
    return client.post(
        "/admin/plugins/create",
        data=data,
        expect=None, raw=True,
    )


def admin_list_plugins(admin_token, **params):
    """查询插件列表"""
    return admin_client(admin_token).get("/admin/plugins", params=params)


def admin_plugin_detail(admin_token, slug, version=None):
    """查询插件详情"""
    params = {"slug": slug}
    if version:
        params["version"] = version
    return admin_client(admin_token).get("/admin/plugins/detail", params=params)


def admin_update_plugin(admin_token, slug, name, version, **kwargs):
    """版本更新（上传新版本）"""
    client = admin_client(admin_token)
    data = {
        "slug": slug,
        "name": name,
        "version": version,
    }
    for k, v in kwargs.items():
        data[k] = v
    return client.post(
        "/admin/plugins/update",
        data=data,
        expect=None, raw=True,
    )


def admin_distribute_plugin(admin_token, slug, instance_ids):
    """下发插件"""
    return admin_client(admin_token).post(
        "/admin/plugins/distribute",
        json={"slug": slug, "instance_ids": instance_ids},
        expect=None, raw=True,
    )


def admin_uninstall_plugin(admin_token, slug, instance_ids):
    """卸载插件"""
    return admin_client(admin_token).post(
        "/admin/plugins/uninstall",
        json={"slug": slug, "instance_ids": instance_ids},
        expect=None, raw=True,
    )


def admin_plugin_instances(admin_token, slug, **params):
    """查询插件实例安装状态"""
    params["slug"] = slug
    return admin_client(admin_token).get("/admin/plugins/instances", params=params)


def admin_plugin_tasks(admin_token, slug, **params):
    """查询插件下发/卸载任务列表"""
    params["slug"] = slug
    return admin_client(admin_token).get("/admin/plugins/tasks", params=params)


def admin_delete_plugin(admin_token, slug, version):
    """删除插件版本"""
    return admin_client(admin_token).post(
        "/admin/plugins/delete",
        json={"slug": slug, "version": version},
        expect=None, raw=True,
    )


def wait_plugin_task(admin_token, slug, task_id, timeout=30):
    """等待插件任务完成"""
    deadline = time.time() + timeout
    while time.time() < deadline:
        data = admin_plugin_tasks(admin_token, slug)
        tasks = data.get("tasks", [])
        for t in tasks:
            if t.get("id") == task_id and t.get("status") == "completed":
                return t
        time.sleep(0.5)
    raise TimeoutError(f"插件任务 {task_id} 超时未完成")
