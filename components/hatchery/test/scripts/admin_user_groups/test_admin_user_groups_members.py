#!/usr/bin/env python3
"""
集成测试：用户组成员管理专项

聚焦三个成员变更接口的语义和边界：
    POST /admin/user-groups/members/set      全量替换 / 清空（[]）
    POST /admin/user-groups/members/add      幂等
    POST /admin/user-groups/members/remove   静默忽略不存在成员
    GET  /admin/user-groups/members          分页 / 总数
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

PREFIX = f"it-grpm-{int(time.time())}"

state = {
    "group_id": None,
    "user_ids": [],     # 准备好的若干普通用户
}


def _ensure_users(n=4):
    if state["user_ids"]:
        return state["user_ids"]
    items = [
        {"username": f"{PREFIX}-u{i}", "password": "Aa12345!", "role": "user"}
        for i in range(n)
    ]
    seed.post("/admin/batch-create", json=items, expect=None)
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


def _ensure_group():
    if state["group_id"]:
        return state["group_id"]
    name = f"{PREFIX}-G"
    resp = seed.post("/admin/user-groups/create",
                     json={"name": name, "description": "members专项"},
                     expect=None, raw=True)
    gid = None
    if resp.status_code == 200:
        gid = extract_group_id(resp.json())
    if not gid:
        target = pick_group(list_all_groups(), name=name)
        gid = target and (target.get("id") or target.get("ID"))
    state["group_id"] = gid
    return gid


def _members_count(gid):
    resp = seed.get("/admin/user-groups/members",
                    params={"id": gid, "page": 1, "page_size": 100},
                    expect=None, raw=True)
    if resp.status_code != 200:
        return -1, []
    data = resp.json() or {}
    members = data.get("members") or data.get("Members") or []
    total = data.get("total") or data.get("Total") or 0
    return total, members


def test_01_prepare():
    """前置：创建 4 个测试用户 + 1 个用户组"""
    uids = _ensure_users(4)
    assert len(uids) >= 4, f"期望 4 个测试用户，实际 {len(uids)}"
    gid = _ensure_group()
    assert gid, "用户组创建失败"
    print(f"    group_id={gid}, user_ids={uids}")


def test_02_set_full():
    """members/set: 全量设置为 [u0, u1, u2]"""
    gid = state["group_id"]
    uids = state["user_ids"][:3]
    seed.post("/admin/user-groups/members/set",
              json={"id": gid, "user_ids": uids})
    total, members = _members_count(gid)
    member_ids = {m.get("user_id") or m.get("UserID") for m in members}
    assert total == 3, f"total {total} != 3"
    assert set(uids) == member_ids, \
        f"成员不一致: got={member_ids} expect={uids}"


def test_03_set_replace():
    """members/set: 全量替换为 [u2, u3]，应仅留这两个"""
    gid = state["group_id"]
    uids = state["user_ids"][2:4]
    seed.post("/admin/user-groups/members/set",
              json={"id": gid, "user_ids": uids})
    total, members = _members_count(gid)
    member_ids = {m.get("user_id") or m.get("UserID") for m in members}
    assert total == 2 and set(uids) == member_ids, \
        f"全量替换语义未生效: total={total} ids={member_ids} expect={uids}"


def test_04_add_idempotent():
    """members/add: 添加 [u0, u2]（u2 已在组内）应幂等成功"""
    gid = state["group_id"]
    u_all = state["user_ids"]
    add_uids = [u_all[0], u_all[2]]
    seed.post("/admin/user-groups/members/add",
              json={"id": gid, "user_ids": add_uids})
    total, members = _members_count(gid)
    member_ids = {m.get("user_id") or m.get("UserID") for m in members}
    expect = {u_all[0], u_all[2], u_all[3]}
    assert total == 3 and expect == member_ids, \
        f"幂等添加失败: total={total} ids={member_ids} expect={expect}"


def test_05_remove_silent_ignore():
    """members/remove: 移除 [u9999, u0]，未在组内的成员应被静默忽略"""
    gid = state["group_id"]
    u_all = state["user_ids"]
    remove_uids = [9999999, u_all[0]]
    resp = seed.post("/admin/user-groups/members/remove",
                     json={"id": gid, "user_ids": remove_uids},
                     expect=None, raw=True)
    assert 200 <= resp.status_code < 300, \
        f"移除应 2xx，实际 {resp.status_code}"
    total, members = _members_count(gid)
    member_ids = {m.get("user_id") or m.get("UserID") for m in members}
    expect = {u_all[2], u_all[3]}
    assert expect == member_ids, \
        f"移除后成员错误: got={member_ids} expect={expect}"


def test_06_set_clear():
    """members/set: user_ids=[] 应清空所有成员"""
    gid = state["group_id"]
    seed.post("/admin/user-groups/members/set",
              json={"id": gid, "user_ids": []})
    total, members = _members_count(gid)
    assert total == 0 and not members, \
        f"清空失败 total={total} len={len(members)}"


def test_07_members_paging_after_clear():
    """查询成员: 清空后再分页查询应总数为 0"""
    gid = state["group_id"]
    data = seed.get("/admin/user-groups/members",
                    params={"id": gid, "page": 1, "page_size": 5})
    total = data.get("total") or data.get("Total") or 0
    members = data.get("members") or data.get("Members") or []
    assert total == 0 and not members, \
        f"期望空成员，实际 total={total} len={len(members)}"


def cleanup():
    """兜底清理"""
    try:
        cleanup_by_prefix(group_prefix=PREFIX, user_prefix=PREFIX)
    except Exception as e:
        print(f"[cleanup] 异常: {e}")


def main():
    health_check()
    try:
        run_tests(globals(), title="用户组成员管理专项",
                  ordered=True, abort_on_fail=True)
    finally:
        cleanup()


if __name__ == "__main__":
    main()
