"""
企业技能（管控端）管理 API 辅助函数

对应 controller/admin_skills.go，覆盖企业技能"创建 → 下发 → 查询实例安装情况"
三个核心接口，用于验证企业技能在实例重装后可被再次下发：
  - POST /admin/skills/create     创建企业技能（multipart/form-data，上传技能 zip）
  - POST /admin/skills/distribute 批量下发企业技能到实例（JSON）
  - GET  /admin/skills/instances  查询某技能的实例安装情况（重装后仍应能查到实例）

典型用法（验证重装后再次下发）：
    slug = f"e2e-skill-{int(time.time())}"
    admin_create_skill(admin.token, slug, "E2E 技能", "1.0.0")
    admin_distribute_skill(admin.token, slug, "1.0.0", [inst.db_id])
    assert admin_skill_find_instance(admin.token, slug, inst.db_id)  # 首次下发可查到
    # ... 重装实例 ...
    admin_distribute_skill(admin.token, slug, "1.0.0", [inst.db_id])  # 再次下发
    assert admin_skill_find_instance(admin.token, slug, inst.db_id)  # 重装后仍可查到
"""

import io
import time
import zipfile

from helpers import config
from helpers.api import admin_client


def build_skill_zip(slug, name, description="E2E 集成测试技能", body=None):
    """构造一个最小可用的技能 zip（内存字节流）。

    后端 validateSkillZip 以 SKILL.md 为锚点提取技能文件，因此只需保证 zip 中
    存在唯一一个 SKILL.md 即可通过校验。这里生成 {slug}/SKILL.md 结构。
    """
    if body is None:
        body = (
            f"---\n"
            f"name: {name}\n"
            f"description: {description}\n"
            f"---\n\n"
            f"# {name}\n\n"
            f"由集成测试自动生成的企业技能，用于验证重装后再次下发。\n"
        )
    buf = io.BytesIO()
    with zipfile.ZipFile(buf, "w", zipfile.ZIP_DEFLATED) as zf:
        zf.writestr(f"{slug}/SKILL.md", body)
    buf.seek(0)
    return buf.getvalue()


def admin_create_skill(admin_token, slug, name, version, *, zip_data=None,
                       description="E2E 集成测试技能", **kwargs):
    """创建企业技能（multipart/form-data，含 zip 文件上传）。

    返回原始 Response 以便检查状态码；成功时响应体形如：
        {"ok": true, "id": 1, "slug": "...", "version": "1.0.0", ...}
    """
    if zip_data is None:
        zip_data = build_skill_zip(slug, name, description)
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
        "/admin/skills/create",
        data=data,
        files=files,
        timeout=60,
        expect=None, raw=True,
    )


def admin_distribute_skill(
        admin_token, slug, version, instance_ids=None, *, source="enterprise",
        select_all=False, statuses=None, group_ids=None, search=None):
    """批量下发企业或公共技能到实例（JSON）。

    instance_ids 与 select_all=True 二选一；全选模式可按状态、用户组和 search 筛选。
    返回原始 Response；成功时响应体形如：
        {"ok": true, "task_id": 1, "version": "1.0.0"}
    """
    body = {"source": source, "slug": slug, "version": version}
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
        "/admin/skills/distribute",
        json=body,
        timeout=60,
        expect=None, raw=True,
    )


def admin_skill_instances(admin_token, slug, **params):
    """查询某技能的实例安装情况（GET /admin/skills/instances）。

    返回解析后的 dict，形如：
        {"instances": [...], "page": 1, "page_size": 500, "total": N}
    每个实例项含 instance_id（实例 DB 主键）、cvm_instance_id、status（安装状态）等。
    """
    params["slug"] = slug
    return admin_client(admin_token).get("/admin/skills/instances", params=params)


def admin_skill_find_instance(admin_token, slug, instance_db_id, **params):
    """在某技能的实例安装情况列表中查找指定实例。

    用于验证"重装后调用 /admin/skills/instances 仍能查到这台实例"。
    命中返回该实例项 dict，未命中返回 None。
    """
    data = admin_skill_instances(admin_token, slug, **params)
    for inst in data.get("instances", []):
        if inst.get("instance_id") == instance_db_id:
            return inst
    return None


# 下发已收敛（不再处于过渡态）的状态集合：
#   installed/failed/upgrade_failed/outdated 均为终态；installing 表示仍在下发中。
SKILL_SETTLED_STATUSES = ("installed", "failed", "upgrade_failed", "outdated")


def wait_skill_instance_status(admin_token, slug, instance_db_id, target,
                               timeout=None, **params):
    """轮询等待某实例在技能实例列表中达到期望安装状态集合。

    target 为可接受状态的集合（如 ("installing", "installed")）。
    额外 **params 透传给 /admin/skills/instances（如 search=ins-xxx 缩小范围）。
    命中返回该实例项 dict；超时未命中则抛 TimeoutError。
    """
    timeout = timeout or config.SKILL_POLL_TIMEOUT
    if isinstance(target, str):
        target = (target,)
    start = time.time()
    last = None
    while True:
        inst = admin_skill_find_instance(admin_token, slug, instance_db_id, **params)
        if inst is not None:
            last = inst.get("status")
            if last in target:
                return inst
        if time.time() - start > timeout:
            raise TimeoutError(
                f"技能 {slug} 在 {timeout}s 内未在实例 {instance_db_id} 上达到 "
                f"{target}（当前 status={last}）"
            )
        time.sleep(config.POLL_INTERVAL)


def wait_skill_distributed(admin_token, slug, instance_db_id, timeout=None, **params):
    """轮询等待某实例进入"已下发/下发中"状态（installing 或已收敛终态）。

    用于下发后确认实例已被纳入下发范围（status 离开 uninstalled）。
    返回命中的实例项 dict；超时未命中则抛 TimeoutError。
    """
    return wait_skill_instance_status(
        admin_token, slug, instance_db_id,
        ("installing",) + SKILL_SETTLED_STATUSES,
        timeout=timeout, **params,
    )


def wait_skill_settled(admin_token, slug, instance_db_id, timeout=None, **params):
    """轮询等待某实例的技能下发收敛到终态（installed/failed/...，不再 installing）。

    用于下发后等待下发任务跑完、释放 skill_dist 分布式锁，避免后续再次下发 409。
    """
    return wait_skill_instance_status(
        admin_token, slug, instance_db_id, SKILL_SETTLED_STATUSES,
        timeout=timeout, **params,
    )


def admin_list_skills(admin_token, **params):
    """查询企业技能列表（GET /admin/skills）。返回解析后的 dict。"""
    return admin_client(admin_token).get("/admin/skills", params=params)


def admin_find_skill(admin_token, slug, **params):
    """按 slug 查找技能列表中的第一条记录；未命中返回 None。"""
    params = dict(params)
    params.setdefault("page", 1)
    params.setdefault("page_size", 10)
    params["slug"] = slug
    data = admin_list_skills(admin_token, **params)
    skills = data.get("skills") or []
    return skills[0] if skills else None


def admin_skill_status(admin_token, slug):
    """返回技能 status 字符串；不存在返回 'NOT_FOUND'。"""
    skill = admin_find_skill(admin_token, slug)
    if not skill:
        return "NOT_FOUND"
    return skill.get("status") or ""


def admin_delete_skill(admin_token, slug, version=None, *, cascade=True):
    """删除技能（POST /admin/skills/delete）。返回原始 Response。"""
    data = {"slug": slug}
    if version is not None:
        data["version"] = version
    if cascade:
        data["cascade"] = "true"
    return admin_client(admin_token).post(
        "/admin/skills/delete",
        data=data,
        expect=None, raw=True,
    )


def admin_offline_skill(admin_token, slug):
    """管理员下架技能（POST /admin/skills/offline）。返回原始 Response。"""
    return admin_client(admin_token).post(
        "/admin/skills/offline",
        data={"slug": slug},
        expect=None, raw=True,
    )


def admin_online_skill(admin_token, slug):
    """管理员上架技能（POST /admin/skills/online）。返回原始 Response。"""
    return admin_client(admin_token).post(
        "/admin/skills/online",
        data={"slug": slug},
        expect=None, raw=True,
    )
