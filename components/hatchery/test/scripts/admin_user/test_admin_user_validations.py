#!/usr/bin/env python3
"""
集成测试：用户管理参数校验与业务规则

覆盖接口（异常路径）：
    POST /admin/create        缺参/越界/重名
    POST /admin/update-user   不存在 id
    POST /admin/delete        初始管理员保护
    POST /admin/hard-delete   初始管理员保护
    POST /admin/reset-password 空密码
    GET  /admin/users         未鉴权
"""
import os
import sys
import time

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers.api import (  # noqa: E402
    seed, anon,
    health_check, run_tests,
    extract_uid,
    cleanup_users_by_prefix,
)

PREFIX = f"it-val-{int(time.time())}"


def test_01_create_missing_username():
    """创建：缺少 username 应 4xx"""
    resp = seed.post("/admin/create", data={"password": "Aa12345!"}, expect=None, raw=True)
    assert 400 <= resp.status_code < 500, \
        f"期望 4xx，实际 {resp.status_code}"


def test_02_create_quota_out_of_range():
    """创建：instance_quota 越界应 4xx"""
    resp = seed.post("/admin/create", data={
        "username": f"{PREFIX}-q",
        "password": "Aa12345!",
        "instance_quota": "1000000",
    }, expect=None, raw=True)
    assert 400 <= resp.status_code < 500, \
        f"期望 4xx，实际 {resp.status_code}"


def test_03_create_duplicate():
    """创建：用户名重复应 4xx"""
    name = f"{PREFIX}-dup"
    seed.post("/admin/create", data={"username": name, "password": "Aa12345!"}, raw=True)
    resp = seed.post("/admin/create",
                     data={"username": name, "password": "Aa12345!"},
                     expect=None, raw=True)
    assert 400 <= resp.status_code < 500, \
        f"重复创建应 4xx，实际 {resp.status_code}"
    # 清理
    users = seed.get("/admin/users",
                     params={"username": name}, expect=None, raw=True).json().get("users") or []
    for u in users:
        uid = u.get("id") or u.get("ID")
        if uid:
            seed.post("/admin/hard-delete", params={"id": uid}, data={}, expect=None, raw=True)


def test_04_update_unknown_user():
    """更新：不存在的 id 应 4xx"""
    resp = seed.post("/admin/update-user",
                     params={"id": 9999999},
                     data={"instance_quota": "1"},
                     expect=None, raw=True)
    assert 400 <= resp.status_code < 500, \
        f"期望 4xx，实际 {resp.status_code}"


def test_05_delete_initial_admin():
    """删除：初始管理员（id=1）应被保护，返回 4xx"""
    resp = seed.post("/admin/delete", params={"id": 1}, data={}, expect=None, raw=True)
    assert 400 <= resp.status_code < 500, \
        f"期望 4xx，实际 {resp.status_code}"


def test_06_hard_delete_initial_admin():
    """硬删除：初始管理员应被保护，返回 4xx"""
    resp = seed.post("/admin/hard-delete", params={"id": 1}, data={}, expect=None, raw=True)
    assert 400 <= resp.status_code < 500, \
        f"期望 4xx，实际 {resp.status_code}"


def test_07_unauthorized():
    """无 Authorization 头访问 /admin/users 应 401/403"""
    r = anon.get("/admin/users", expect=None, timeout=10, raw=True)
    assert r.status_code in (401, 403), \
        f"期望 401/403，实际 {r.status_code}"


def test_08_reset_password_empty():
    """重置密码：密码为空应 4xx"""
    name = f"{PREFIX}-rp"
    r = seed.post("/admin/create",
                  data={"username": name, "password": "Aa12345!"}, raw=True)
    uid = extract_uid(r.json())
    if not uid:
        users = seed.get("/admin/users",
                         params={"username": name}).get("users") or []
        uid = users and (users[0].get("id") or users[0].get("ID"))
    assert uid, "用例前置：未取到 uid"

    try:
        resp = seed.post("/admin/reset-password",
                         params={"id": uid},
                         data={"password": ""},
                         expect=None, raw=True)
        assert 400 <= resp.status_code < 500, \
            f"期望 4xx，实际 {resp.status_code}"
    finally:
        seed.post("/admin/hard-delete", params={"id": uid}, data={}, expect=None, raw=True)


def cleanup():
    """兜底清理"""
    try:
        cleanup_users_by_prefix(PREFIX)
    except Exception as e:
        print(f"[cleanup] 异常: {e}")


def main():
    health_check()
    try:
        run_tests(globals(), title="用户管理参数校验", ordered=True)
    finally:
        cleanup()


if __name__ == "__main__":
    main()
