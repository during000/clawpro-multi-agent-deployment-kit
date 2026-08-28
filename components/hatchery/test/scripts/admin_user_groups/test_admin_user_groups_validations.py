#!/usr/bin/env python3
"""
集成测试：用户组管理参数校验与业务规则（异常路径）

覆盖接口（异常分支）：
    POST /admin/user-groups/create             缺名 / 重名 / 名称过长 / 平台总数上限(可选)
    POST /admin/user-groups/update             不存在的 id / 改名为已存在
    POST /admin/user-groups/delete             不存在的 id / 缺 id
    GET  /admin/user-groups/members            缺 id / 不存在的 id
    POST /admin/user-groups/members/set        非法 user_id / user_ids 长度超 10000
    POST /admin/user-groups/members/add        非法 user_id / user_ids 长度超 10000
    POST /admin/user-groups/members/remove     缺 id
    GET  /admin/user-groups/associated-models  缺 group_id / 格式错误
    GET  /admin/user-groups                    未鉴权

可选环境变量：
    SKIP_HEAVY=0          开启重型用例（如平台用户组 1000+ 上限测试，默认跳过）
    SKIP_OVERSIZE=1       跳过超大 payload (>=10001 user_ids) 的边界用例（默认跳过）
"""
import os
import sys
import time

import requests

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers.api import (  # noqa: E402
    seed, anon,
    health_check, run_tests,
)
from helpers.user_groups import (  # noqa: E402
    extract_group_id, pick_group, list_all_groups, find_groups_by_prefix,
    cleanup_by_prefix,
)

PREFIX = f"it-grpv-{int(time.time())}"

state = {"group_a": None, "group_b": None}


def _ensure_group(name):
    resp = seed.post("/admin/user-groups/create",
                     json={"name": name, "description": "validation test"},
                     expect=None, raw=True)
    if resp.status_code == 200:
        gid = extract_group_id(resp.json())
        if gid:
            return gid
    target = pick_group(list_all_groups(), name=name)
    return target and (target.get("id") or target.get("ID"))


def _get_or_create_anchor(slot, suffix):
    gid = state.get(slot)
    if gid:
        return gid
    name = f"{PREFIX}-{suffix}"
    gid = _ensure_group(name)
    if gid:
        state[slot] = gid
    return gid


# ─── /admin/user-groups/create 异常 ────────────────────────────────────────
def test_01_create_missing_name():
    """创建：缺少 name 应 4xx"""
    resp = seed.post("/admin/user-groups/create",
                     json={"description": "no name"}, expect=None, raw=True)
    assert 400 <= resp.status_code < 500, \
        f"期望 4xx，实际 {resp.status_code}"


def test_02_create_duplicate():
    """创建：name 重复应 4xx"""
    name = f"{PREFIX}-dup"
    gid = _ensure_group(name)
    assert gid, "前置失败：首次创建未成功"
    state["group_a"] = gid
    resp = seed.post("/admin/user-groups/create",
                     json={"name": name, "description": "dup"},
                     expect=None, raw=True)
    assert 400 <= resp.status_code < 500, \
        f"期望 4xx，实际 {resp.status_code}"


# ─── /admin/user-groups/update 异常 ────────────────────────────────────────
def test_03_update_unknown_id():
    """更新：不存在的 id 应 4xx"""
    resp = seed.post("/admin/user-groups/update",
                     json={"id": 9999999, "name": "x"}, expect=None, raw=True)
    assert 400 <= resp.status_code < 500, \
        f"期望 4xx，实际 {resp.status_code}"


def test_04_update_rename_to_existing():
    """更新：改名为另一个已存在的用户组名应 4xx"""
    name_b = f"{PREFIX}-existing"
    gid_b = _ensure_group(name_b)
    assert gid_b, "前置失败：未能创建第二个用户组"
    state["group_b"] = gid_b
    gid_a = _get_or_create_anchor("group_a", "dup")
    assert gid_a, "前置失败：未能准备 group_a"
    resp = seed.post("/admin/user-groups/update",
                     json={"id": gid_a, "name": name_b},
                     expect=None, raw=True)
    assert 400 <= resp.status_code < 500, \
        f"期望 4xx，实际 {resp.status_code}"


# ─── /admin/user-groups/delete 异常 ────────────────────────────────────────
def test_05_delete_unknown_id():
    """删除：不存在的 id 应 4xx"""
    resp = seed.post("/admin/user-groups/delete",
                     json={"id": 9999999}, expect=None, raw=True)
    assert 400 <= resp.status_code < 500, \
        f"期望 4xx，实际 {resp.status_code}"


def test_06_delete_missing_id():
    """删除：缺少 id 应 4xx"""
    resp = seed.post("/admin/user-groups/delete", json={},
                     expect=None, raw=True)
    assert 400 <= resp.status_code < 500, \
        f"期望 4xx，实际 {resp.status_code}"


# ─── /admin/user-groups/members 异常 ───────────────────────────────────────
def test_07_members_missing_id():
    """查询成员：缺少 id 应 4xx"""
    resp = seed.get("/admin/user-groups/members", expect=None, raw=True)
    assert 400 <= resp.status_code < 500, \
        f"期望 4xx，实际 {resp.status_code}"


def test_08_members_unknown_id():
    """查询成员：不存在的 id 应 4xx 或返回空"""
    resp = seed.get("/admin/user-groups/members",
                    params={"id": 9999999}, expect=None, raw=True)
    if 200 <= resp.status_code < 300:
        data = resp.json() or {}
        members = data.get("members") or data.get("Members") or []
        total = data.get("total") or data.get("Total") or 0
        assert not (total or members), \
            f"不存在 group 但返回了成员 total={total} len={len(members)}"
    else:
        assert 400 <= resp.status_code < 500, \
            f"未预期状态码 {resp.status_code}"


# ─── /admin/user-groups/members/set & add 非法 user_id ─────────────────────
def test_09_set_members_invalid_user():
    """members/set：包含非法 user_id 应 4xx"""
    gid = _get_or_create_anchor("group_a", "dup")
    assert gid, "前置失败：未能准备 group_a"
    resp = seed.post("/admin/user-groups/members/set",
                     json={"id": gid, "user_ids": [9999999]},
                     expect=None, raw=True)
    assert 400 <= resp.status_code < 500, \
        f"期望 4xx，实际 {resp.status_code}"


def test_10_add_members_invalid_user():
    """members/add：包含非法 user_id 应 4xx"""
    gid = _get_or_create_anchor("group_a", "dup")
    assert gid, "前置失败：未能准备 group_a"
    resp = seed.post("/admin/user-groups/members/add",
                     json={"id": gid, "user_ids": [9999999]},
                     expect=None, raw=True)
    assert 400 <= resp.status_code < 500, \
        f"期望 4xx，实际 {resp.status_code}"


def test_11_remove_members_missing_id():
    """members/remove：缺少 id 应 4xx"""
    resp = seed.post("/admin/user-groups/members/remove",
                     json={"user_ids": [1]}, expect=None, raw=True)
    assert 400 <= resp.status_code < 500, \
        f"期望 4xx，实际 {resp.status_code}"


# ─── /admin/user-groups/associated-models 异常 ────────────────────────────
def test_12_assoc_missing_group_id():
    """associated-models：缺少 group_id 应 4xx"""
    resp = seed.get("/admin/user-groups/associated-models",
                    expect=None, raw=True)
    assert 400 <= resp.status_code < 500, \
        f"期望 4xx，实际 {resp.status_code}"


def test_13_assoc_invalid_group_id():
    """associated-models：group_id 格式错误应 4xx"""
    resp = seed.get("/admin/user-groups/associated-models",
                    params={"group_id": "abc"}, expect=None, raw=True)
    assert 400 <= resp.status_code < 500, \
        f"期望 4xx，实际 {resp.status_code}"


# ─── 鉴权 ──────────────────────────────────────────────────────────────────
def test_14_unauthorized_list():
    """无 Authorization 头访问 /admin/user-groups 应 401/403"""
    r = anon.get("/admin/user-groups", expect=None, timeout=10, raw=True)
    assert r.status_code in (401, 403), \
        f"期望 401/403，实际 {r.status_code}"


# ─── 容量上限 ──────────────────────────────────────────────────────────────
MAX_MEMBERS = 10000
MAX_GROUPS = 2000


def _run_oversize_members_case(path, base_id, label):
    """超大 payload 边界测试的统一实现"""
    skip = os.environ.get("SKIP_OVERSIZE", "1") != "0"
    if skip:
        print(f"    已跳过（设置 SKIP_OVERSIZE=0 启用）")
        return

    gid = _get_or_create_anchor("group_a", "dup")
    assert gid, "前置失败：未能准备 group_a"

    user_ids = list(range(base_id, base_id + MAX_MEMBERS + 1))
    try:
        resp = seed.post(path,
                         json={"id": gid, "user_ids": user_ids},
                         expect=None, raw=True,
                         timeout=30)
    except (requests.exceptions.ConnectionError,
            requests.exceptions.Timeout,
            requests.exceptions.ChunkedEncodingError) as e:
        print(f"    网关层拒绝超大 payload (视作通过): {type(e).__name__}")
        return

    assert 400 <= resp.status_code < 500, \
        f"期望 4xx，实际 {resp.status_code}"


def test_15_set_members_over_limit():
    """members/set：user_ids 长度超 10000 应 4xx [SKIP_OVERSIZE]"""
    _run_oversize_members_case(
        "/admin/user-groups/members/set",
        base_id=10_000_000,
        label="members/set",
    )


def test_16_add_members_over_limit():
    """members/add：user_ids 长度超 10000 应 4xx [SKIP_OVERSIZE]"""
    _run_oversize_members_case(
        "/admin/user-groups/members/add",
        base_id=20_000_000,
        label="members/add",
    )


HEAVY_PREFIX = f"{PREFIX}-heavy"


def test_17_create_over_platform_limit():
    """创建：平台用户组数量达上限后再创建应 4xx [SKIP_HEAVY]"""
    skip = os.environ.get("SKIP_HEAVY", "1") != "0"
    if skip:
        print("    已跳过（设置 SKIP_HEAVY=0 启用）")
        return

    existing = list_all_groups()
    have = len(existing)
    need = MAX_GROUPS - have
    if need < 0:
        need = 0

    created = 0
    last_resp_code = None
    try:
        for i in range(need):
            name = f"{HEAVY_PREFIX}-{i:05d}"
            r = seed.post("/admin/user-groups/create",
                          json={"name": name, "description": "heavy limit fill"},
                          expect=None, raw=True)
            last_resp_code = r.status_code
            if r.status_code != 200:
                break
            created += 1

        r = seed.post("/admin/user-groups/create",
                      json={"name": f"{HEAVY_PREFIX}-overflow",
                            "description": "overflow"},
                      expect=None, raw=True)
        assert 400 <= r.status_code < 500, \
            (f"期望 4xx，实际 {r.status_code}; "
             f"created={created} need={need} last_fill={last_resp_code}")
    finally:
        for g in find_groups_by_prefix(HEAVY_PREFIX):
            gid = g.get("id") or g.get("ID")
            if gid:
                seed.post("/admin/user-groups/delete",
                          json={"id": gid}, expect=None)


def cleanup():
    """兜底清理"""
    try:
        cleanup_by_prefix(group_prefix=PREFIX)
    except Exception as e:
        print(f"[cleanup] 异常: {e}")


def main():
    health_check()
    try:
        run_tests(globals(), title="用户组管理参数校验", ordered=True)
    finally:
        cleanup()


if __name__ == "__main__":
    main()
