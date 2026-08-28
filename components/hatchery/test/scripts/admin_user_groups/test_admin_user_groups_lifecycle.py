#!/usr/bin/env python3
"""
集成测试：用户组管理全生命周期 (E2E)

覆盖图片所列 9 个接口（管理接口 - 用户组管理）：
    GET  /admin/user-groups                    查询用户组列表
    POST /admin/user-groups/create             创建用户组
    POST /admin/user-groups/update             更新用户组
    POST /admin/user-groups/delete             删除用户组
    GET  /admin/user-groups/members            查询用户组成员
    POST /admin/user-groups/members/set        设置用户组成员（全量替换）
    POST /admin/user-groups/members/add        添加用户组成员
    POST /admin/user-groups/members/remove     移除用户组成员
    GET  /admin/user-groups/associated-models  查询用户组关联模型
"""
import os
import sys
import time

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers.api import (  # noqa: E402
    seed,
    health_check, run_tests,
    extract_uid, pick_user,
)
from helpers.user_groups import (  # noqa: E402
    extract_group_id, pick_group, list_all_groups, find_groups_by_prefix,
    cleanup_by_prefix,
)

PREFIX = f"it-grp-{int(time.time())}"
GROUP_NAME = f"{PREFIX}-A"
GROUP_NAME_RENAMED = f"{PREFIX}-A-renamed"

# 跨用例共享状态
state = {
    "group_id": None,
    "user_ids": [],   # 测试期间创建的若干普通用户，用作组成员
}


# ─── 前置：批量创建若干测试用户作为组成员 ─────────────────────────────────
def _ensure_test_users(n=3):
    if state["user_ids"]:
        return state["user_ids"]
    items = [
        {"username": f"{PREFIX}-u{i}", "password": "Aa12345!", "role": "user"}
        for i in range(n)
    ]
    resp = seed.post("/admin/batch-create", json=items, expect=None, raw=True)
    if resp.status_code != 200:
        for it in items:
            seed.post("/admin/create", data=it, expect=None, raw=True)
    uids = []
    for it in items:
        users = seed.get("/admin/users",
                         params={"username": it["username"]},
                         expect=None, raw=True).json().get("users") or []
        target = pick_user(users, username=it["username"])
        if target:
            uid = target.get("id") or target.get("ID")
            if uid:
                uids.append(uid)
    state["user_ids"] = uids
    return uids


# ─── 用例 ──────────────────────────────────────────────────────────────────
def test_01_create_group():
    """创建用户组"""
    resp = seed.post("/admin/user-groups/create", json={
        "name": GROUP_NAME,
        "description": "集成测试自动创建",
    }, raw=True)
    data = resp.json()
    assert not (isinstance(data, dict) and data.get("error")), \
        f"创建失败: {data.get('error')}"
    gid = extract_group_id(data)
    if not gid:
        groups = list_all_groups()
        target = pick_group(groups, name=GROUP_NAME)
        gid = target and (target.get("id") or target.get("ID"))
    assert gid, f"创建响应未包含 id 且列表查不到: {data}"
    state["group_id"] = gid
    print(f"    group_id={gid}")


def test_02_list_groups():
    """分页查询用户组列表，确认新建用户组可见"""
    gid = state["group_id"]
    data = seed.get("/admin/user-groups",
                    params={"page": 1, "page_size": 100})
    groups = data.get("groups") or data.get("Groups") or []
    if not pick_group(groups, gid=gid):
        all_groups = list_all_groups()
        assert pick_group(all_groups, gid=gid), \
            f"列表中未找到 group_id={gid}"
    total = data.get("total") or data.get("Total")
    print(f"    total={total}")


def test_03_update_group():
    """更新用户组的名称与描述"""
    gid = state["group_id"]
    seed.post("/admin/user-groups/update", json={
        "id": gid,
        "name": GROUP_NAME_RENAMED,
        "description": "已重命名描述",
    })
    groups = list_all_groups()
    target = pick_group(groups, gid=gid)
    assert target, "更新后查不到该用户组"
    new_name = target.get("name") or target.get("Name")
    assert new_name == GROUP_NAME_RENAMED, \
        f"name 未生效: 实际={new_name} 期望={GROUP_NAME_RENAMED}"


def test_04_set_members_full():
    """全量设置用户组成员 (members/set)"""
    gid = state["group_id"]
    uids = _ensure_test_users(3)
    assert len(uids) >= 3, f"前置失败：仅准备到 {len(uids)} 个测试用户"
    seed.post("/admin/user-groups/members/set", json={
        "id": gid,
        "user_ids": uids,
    })
    data = seed.get("/admin/user-groups/members",
                    params={"id": gid, "page": 1, "page_size": 100})
    total = data.get("total") or data.get("Total") or 0
    members = data.get("members") or data.get("Members") or []
    member_ids = {m.get("user_id") or m.get("UserID") for m in members}
    assert total == len(uids), f"成员总数 {total} != 期望 {len(uids)}"
    assert set(uids).issubset(member_ids), \
        f"成员 ID 不一致: got={member_ids} expect={uids}"


def test_05_add_members_idempotent():
    """批量添加成员 (members/add) - 含已存在成员，应幂等成功"""
    gid = state["group_id"]
    uids = state["user_ids"]
    seed.post("/admin/user-groups/members/add", json={
        "id": gid,
        "user_ids": list(uids),
    })
    data = seed.get("/admin/user-groups/members",
                    params={"id": gid, "page": 1, "page_size": 100})
    total = (data.get("total") or data.get("Total") or 0)
    assert total == len(uids), \
        f"幂等添加后成员总数 {total} != {len(uids)}"


def test_06_remove_members():
    """批量移除成员 (members/remove) - 移除 1 个，剩余应为 N-1"""
    gid = state["group_id"]
    uids = state["user_ids"]
    to_remove = uids[:1]
    seed.post("/admin/user-groups/members/remove", json={
        "id": gid,
        "user_ids": to_remove,
    })
    data = seed.get("/admin/user-groups/members",
                    params={"id": gid, "page": 1, "page_size": 100})
    total = data.get("total") or data.get("Total") or 0
    members = data.get("members") or data.get("Members") or []
    member_ids = {m.get("user_id") or m.get("UserID") for m in members}
    assert total == len(uids) - 1, \
        f"移除后总数 {total} != {len(uids) - 1}"
    assert to_remove[0] not in member_ids, \
        f"被移除成员仍在列表中: {to_remove[0]}"


def test_07_query_members_paging():
    """查询用户组成员 - 分页参数"""
    gid = state["group_id"]
    uids = state["user_ids"]
    # 先把成员设回完整 N
    seed.post("/admin/user-groups/members/set",
              json={"id": gid, "user_ids": uids}, expect=None)
    # 第一页 page_size=2
    data = seed.get("/admin/user-groups/members",
                    params={"id": gid, "page": 1, "page_size": 2})
    page_members = data.get("members") or data.get("Members") or []
    total = data.get("total") or data.get("Total") or 0
    assert total == len(uids), f"分页查询 total {total} != {len(uids)}"
    assert len(page_members) <= 2, \
        f"page_size=2 但实际返回 {len(page_members)}"


def test_08_associated_models():
    """查询用户组关联模型 (associated-models)"""
    gid = state["group_id"]
    data = seed.get("/admin/user-groups/associated-models",
                    params={"group_id": gid})
    assert isinstance(data, dict), f"返回非对象: {data}"
    assert "count" in data or "Count" in data, \
        f"响应缺少 count 字段: keys={list(data.keys())}"
    assert "models" in data or "Models" in data, \
        f"响应缺少 models 字段: keys={list(data.keys())}"
    models = data.get("models") or data.get("Models") or []
    assert isinstance(models, list), \
        f"models 应为数组，实际 {type(models).__name__}"


def test_09_delete_group():
    """删除用户组，应级联清理成员"""
    gid = state["group_id"]
    seed.post("/admin/user-groups/delete", json={"id": gid})
    groups = list_all_groups()
    assert not pick_group(groups, gid=gid), \
        f"删除后用户组仍在列表中 group_id={gid}"
    # 成员查询应失败或返回空
    resp = seed.get("/admin/user-groups/members",
                    params={"id": gid, "page": 1, "page_size": 10},
                    expect=None, raw=True)
    if 200 <= resp.status_code < 300:
        members = (resp.json().get("members") or resp.json().get("Members") or [])
        assert not members, f"删除后成员仍存在: {len(members)} 条"


def test_10_delete_again_should_4xx():
    """再次删除已不存在的用户组，应 4xx"""
    gid = state["group_id"]
    resp = seed.post("/admin/user-groups/delete", json={"id": gid},
                     expect=None, raw=True)
    assert 400 <= resp.status_code < 500, \
        f"重复删除未被拒绝，状态码 {resp.status_code}"


def cleanup():
    """兜底清理"""
    try:
        cleanup_by_prefix(group_prefix=PREFIX, user_prefix=PREFIX)
    except Exception as e:
        print(f"[cleanup] 异常: {e}")


def main():
    health_check()
    try:
        run_tests(globals(), title="用户组管理全生命周期",
                  ordered=True, abort_on_fail=True)
    finally:
        cleanup()


if __name__ == "__main__":
    main()
