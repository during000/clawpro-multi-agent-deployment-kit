"""
技能共建审核 API 辅助函数

覆盖员工提交/下架/撤回与管理员审核相关接口：
  - POST /openclaw/skills/contribute
  - POST /openclaw/skills/takedown
  - POST /openclaw/skills/contributions/withdraw
  - GET  /openclaw/skills/contributions
  - GET  /openclaw/skills/contributions/detail
  - GET  /admin/contributions
  - GET  /admin/contributions/detail
  - POST /admin/contributions/approve
  - POST /admin/contributions/reject
"""

from helpers.admin_skill import build_skill_zip
from helpers.api import admin_client, user_client


def contribute_skill(user_token, slug, name, version, *, zip_data=None,
                     description="IT 共建技能", **kwargs):
    """员工提交技能（multipart）。返回原始 Response。"""
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
    return user_client(user_token).post(
        "/openclaw/skills/contribute",
        data=data,
        files=files,
        timeout=60,
        expect=None, raw=True,
    )


def takedown_skill(user_token, slug, reason="不再需要"):
    """员工申请下架。返回原始 Response。"""
    return user_client(user_token).post(
        "/openclaw/skills/takedown",
        json={"slug": slug, "reason": reason},
        expect=None, raw=True,
    )


def withdraw_contribution(user_token, request_id):
    """员工撤回自己的审核申请。返回原始 Response。"""
    return user_client(user_token).post(
        "/openclaw/skills/contributions/withdraw",
        json={"id": request_id},
        expect=None, raw=True,
    )


def my_contributions(user_token, **params):
    """员工查看自己的申请列表。返回解析后的 dict。"""
    return user_client(user_token).get(
        "/openclaw/skills/contributions", params=params,
    )


def my_contribution_detail(user_token, request_id):
    """员工查看申请详情。返回原始 Response。"""
    return user_client(user_token).get(
        "/openclaw/skills/contributions/detail",
        params={"id": request_id},
        expect=None, raw=True,
    )


def admin_list_contributions(admin_token, **params):
    """管理员审核列表。返回解析后的 dict。"""
    return admin_client(admin_token).get("/admin/contributions", params=params)


def admin_contribution_detail(admin_token, request_id):
    """管理员审核详情。返回原始 Response。"""
    return admin_client(admin_token).get(
        "/admin/contributions/detail",
        params={"id": request_id},
        expect=None, raw=True,
    )


def admin_approve_contribution(admin_token, request_id):
    """管理员通过审核。返回原始 Response。"""
    return admin_client(admin_token).post(
        "/admin/contributions/approve",
        json={"id": request_id},
        expect=None, raw=True,
    )


def admin_reject_contribution(admin_token, request_id, review_comment="不符合规范"):
    """管理员拒绝审核。返回原始 Response。"""
    return admin_client(admin_token).post(
        "/admin/contributions/reject",
        json={"id": request_id, "review_comment": review_comment},
        expect=None, raw=True,
    )


def skillstore_has_slug(user_token, slug, page_size=100):
    """技能广场是否包含指定 slug。

    优先走 detail（按 slug 精确查 published）；失败时再回退列表首页扫描。
    """
    detail = user_client(user_token).get(
        "/openclaw/skillstore/detail",
        params={"slug": slug},
        expect=None, raw=True,
    )
    if detail.status_code == 200:
        return True
    if detail.status_code == 404:
        return False
    # 兼容异常状态码：回退列表
    data = user_client(user_token).get(
        "/openclaw/skillstore",
        params={"page": 1, "page_size": page_size},
    )
    skills = data.get("skills") or []
    return any(s.get("slug") == slug for s in skills)
