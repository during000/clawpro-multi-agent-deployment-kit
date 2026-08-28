#!/usr/bin/env python3
"""
集成测试：用户管理统计/只读类接口

覆盖接口：
    GET  /admin/user-limit
    GET  /admin/user-vpc
    GET  /admin/departments
"""
import os
import sys
import time

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers.api import (  # noqa: E402
    seed,
    health_check, run_tests,
    extract_uid,
    cleanup_users_by_prefix,
)

PREFIX = f"it-stat-{int(time.time())}"
state = {"uid": None}


def test_01_setup_user():
    """前置：创建临时用户用于 VPC 查询"""
    r = seed.post("/admin/create", data={
        "username": f"{PREFIX}-u",
        "password": "Aa12345!",
    }, raw=True)
    uid = extract_uid(r.json())
    if not uid:
        users = seed.get("/admin/users",
                         params={"username": f"{PREFIX}-u"}).get("users") or []
        uid = users and (users[0].get("id") or users[0].get("ID"))
    assert uid, "前置失败：未取到 uid"
    state["uid"] = uid
    print(f"    uid={uid}")


def test_02_user_limit():
    """查询 /admin/user-limit"""
    data = seed.get("/admin/user-limit")
    assert isinstance(data, dict), f"返回应为对象: {data}"
    assert "limit" in data or "Limit" in data, \
        f"返回应包含 limit 字段: {data}"
    print(f"    {data}")


def test_03_user_limit_method_not_allowed():
    """POST /admin/user-limit 应 4xx/405（仅支持 GET）"""
    resp = seed.post("/admin/user-limit", data={}, expect=None, raw=True)
    assert 400 <= resp.status_code < 500, \
        f"期望 4xx，实际 {resp.status_code}"
    try:
        data = resp.json()
    except Exception:
        data = {}
    err_msg = (data or {}).get("error", "") if isinstance(data, dict) else ""
    print(f"    status={resp.status_code} error={err_msg!r}")


def test_04_user_vpc_known():
    """查询 /admin/user-vpc?id=<uid>"""
    uid = state["uid"]
    data = seed.get("/admin/user-vpc", params={"id": uid})
    assert isinstance(data, dict), f"返回应为对象: {data}"
    print(f"    keys={list(data.keys())[:5]}")


def test_05_user_vpc_unknown():
    """查询不存在用户的 VPC 应 4xx"""
    resp = seed.get("/admin/user-vpc", params={"id": 9999999}, expect=None, raw=True)
    assert 400 <= resp.status_code < 500, \
        f"期望 4xx，实际 {resp.status_code}"


def test_06_departments():
    """查询 /admin/departments，应同时返回 departments 与 department_tree"""
    data = seed.get("/admin/departments")
    assert isinstance(data, dict), f"返回应为对象: {data}"
    assert "departments" in data, \
        f"返回应包含 departments 字段: keys={list(data.keys())}"
    assert "department_tree" in data, \
        f"返回应包含 department_tree 字段: keys={list(data.keys())}"
    assert isinstance(data["departments"], list), \
        f"departments 应为列表: {type(data['departments'])}"
    assert isinstance(data["department_tree"], list), \
        f"department_tree 应为列表: {type(data['department_tree'])}"
    if data["department_tree"]:
        node = data["department_tree"][0]
        for k in ("id", "name", "path", "parent_id", "has_child"):
            assert k in node, f"department_tree 节点缺少字段 {k}: {node}"
    print(f"    departments={len(data['departments'])}, tree={len(data['department_tree'])}")


def test_07_departments_method_not_allowed():
    """POST /admin/departments 应 4xx/405（仅支持 GET）"""
    resp = seed.post("/admin/departments", data={}, expect=None, raw=True)
    assert 400 <= resp.status_code < 500, \
        f"期望 4xx，实际 {resp.status_code}"


def cleanup():
    """兜底清理"""
    try:
        cleanup_users_by_prefix(PREFIX)
    except Exception as e:
        print(f"[cleanup] 异常: {e}")


def main():
    health_check()
    try:
        run_tests(globals(), title="用户管理统计/只读接口",
                  ordered=True, abort_on_fail=True)
    finally:
        cleanup()


if __name__ == "__main__":
    main()
