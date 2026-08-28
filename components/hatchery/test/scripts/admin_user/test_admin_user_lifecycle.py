#!/usr/bin/env python3
"""
集成测试：用户管理全生命周期 (E2E)

覆盖接口（10/15）：
    GET  /admin/users
    POST /admin/create
    GET  /admin/user-token
    POST /admin/token/disable
    POST /admin/token/enable
    POST /admin/update-user
    POST /admin/reset-password
    POST /admin/delete
    POST /admin/restore
    POST /admin/hard-delete
"""
import os
import sys
import time

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers.api import (  # noqa: E402
    health_check, run_tests,
    seed,
    pick_user, extract_uid,
    cleanup_users_by_prefix,
)

USERNAME = f"it-user-{int(time.time())}"
PASSWORD_INIT = "Init@P@ss123"
PASSWORD_NEW = "Reset@P@ss456"

# 跨用例共享的状态（uid 在 create_user 后填入）
state = {"uid": None}


def test_01_create_user():
    """创建用户"""
    resp = seed.post("/admin/create", data={
        "username": USERNAME,
        "password": PASSWORD_INIT,
        "role": "user",
        "email": f"{USERNAME}@example.com",
        "instance_quota": "5",
        "token_quota_day": "100",
    }, raw=True)
    data = resp.json()
    assert not (isinstance(data, dict) and data.get("error")), \
        f"创建失败: {data.get('error')}"
    uid = extract_uid(data)
    if not uid:
        users = seed.get("/admin/users",
                         params={"username": USERNAME}).get("users") or []
        target = pick_user(users, username=USERNAME)
        uid = target and (target.get("id") or target.get("ID"))
    assert uid, f"创建响应未包含 id 且列表查不到: {data}"
    state["uid"] = uid
    print(f"    uid={uid}")


def test_02_list_and_find():
    """查询用户列表，确认新用户可见"""
    uid = state["uid"]
    data = seed.get("/admin/users", params={"username": USERNAME})
    users = data.get("users") or []
    assert pick_user(users, uid=uid), \
        f"用户列表未找到 uid={uid}, 共 {len(users)} 条"


def test_03_get_token():
    """查询 API Token"""
    uid = state["uid"]
    data = seed.get("/admin/user-token", params={"id": uid})
    exists = data.get("exists") if data.get("exists") is not None else data.get("Exists")
    assert exists, f"exists 期望 true, 实际 {data}"


def test_04_disable_token():
    """禁用 Token"""
    uid = state["uid"]
    seed.post("/admin/token/disable", data={}, params={"id": uid})


def test_05_disable_token_again():
    """重复禁用 Token，期望 4xx"""
    uid = state["uid"]
    resp = seed.post("/admin/token/disable", data={}, params={"id": uid},
                     expect=None, raw=True)
    assert not (200 <= resp.status_code < 300), \
        f"重复禁用未被拒绝，状态码 {resp.status_code}"


def test_06_enable_token():
    """启用 Token"""
    uid = state["uid"]
    seed.post("/admin/token/enable", data={}, params={"id": uid})


def test_07_update_user():
    """更新用户的配额与邮箱"""
    uid = state["uid"]
    # ⚠️ hatchery /admin/update-user ：id 走 query string；
    # 业务字段必须走 JSON body，因为客户端带了 Accept: application/json，
    # 服务端 wantsJSON()=true 会走 json.Decode 分支；form body 会被静默丢弃。
    seed.post("/admin/update-user",
              json={
                  "instance_quota": 10,
                  "token_quota_day": 200,
                  "email": f"{USERNAME}-new@example.com",
              },
              params={"id": uid})
    data = seed.get("/admin/users", params={"username": USERNAME})
    users = data.get("users") or []
    target = pick_user(users, uid=uid)
    assert target, "更新后查不到用户"
    quota = target.get("instance_quota")
    if quota is None:
        quota = target.get("InstanceQuota")
    assert str(quota) == "10", f"instance_quota 未生效: {quota}"


def test_08_reset_password():
    """重置密码"""
    uid = state["uid"]
    # ⚠️ /admin/reset-password 与 /admin/update-user 不同：
    #   它无 isJSON 分支，统一使用 r.FormValue("password") 读取，
    #   因此必须用 form-urlencoded body（或放 query string），不能用 JSON。
    seed.post("/admin/reset-password",
              params={"id": uid},
              data={"password": PASSWORD_NEW})


def test_09_soft_delete():
    """软删除"""
    uid = state["uid"]
    seed.post("/admin/delete", data={}, params={"id": uid})
    # ⚠️ hatchery 语义：/admin/users 默认就是 Unscoped 查询，软删用户仍会返回，
    # 仅通过 deleted_at 非空表示"已禁用"。
    data = seed.get("/admin/users", params={"username": USERNAME})
    users = data.get("users") or []
    target = pick_user(users, uid=uid)
    assert target, "软删后用户应仍在列表中（标记为禁用），实际查不到"
    deleted_at = target.get("deleted_at") or target.get("DeletedAt")
    assert deleted_at, f"软删后 deleted_at 应非空，实际 {target}"
    print(f"    deleted_at={deleted_at}")


def test_10_restore():
    """恢复用户"""
    uid = state["uid"]
    seed.post("/admin/restore", data={}, params={"id": uid})
    data = seed.get("/admin/users", params={"username": USERNAME})
    users = data.get("users") or []
    target = pick_user(users, uid=uid)
    assert target, "恢复后列表查不到用户"
    deleted_at = target.get("deleted_at") or target.get("DeletedAt")
    assert not deleted_at, f"恢复后 deleted_at 应为空，实际 {deleted_at}"


def test_11_restore_active_user():
    """对活跃用户重复调用 restore 应 2xx 且保持活跃（幂等）"""
    uid = state["uid"]
    # 前置确认当前用户处于活跃态
    data = seed.get("/admin/users", params={"username": USERNAME})
    users = data.get("users") or []
    target = pick_user(users, uid=uid)
    assert target, "前置失败：列表中查不到用户"
    assert not (target.get("deleted_at") or target.get("DeletedAt")), \
        f"前置失败：用户应为活跃态"

    # 对活跃用户再次调用 restore，期望 2xx（幂等接口）
    resp = seed.post("/admin/restore", data={}, params={"id": uid},
                     expect=None, raw=True)
    assert 200 <= resp.status_code < 300, \
        f"幂等 restore 应 2xx，实际 {resp.status_code}"

    # 调用后再核对：用户依旧活跃
    data = seed.get("/admin/users", params={"username": USERNAME})
    users = data.get("users") or []
    target = pick_user(users, uid=uid)
    assert target, "调用后查不到用户"
    deleted_at = target.get("deleted_at") or target.get("DeletedAt")
    assert not deleted_at, \
        f"幂等 restore 不应破坏活跃状态，实际 deleted_at={deleted_at}"


def test_12_hard_delete():
    """硬删除"""
    uid = state["uid"]
    seed.post("/admin/hard-delete", data={}, params={"id": uid})
    data = seed.get("/admin/users",
                    params={"username": USERNAME, "include_deleted": "true"})
    users = data.get("users") or []
    assert not pick_user(users, uid=uid), "硬删除后仍残留"


def test_13_restore_after_hard_delete():
    """对已硬删用户调用 restore 应 4xx"""
    uid = state["uid"]
    resp = seed.post("/admin/restore", data={}, params={"id": uid},
                     expect=None, raw=True)
    assert not (200 <= resp.status_code < 300), \
        f"对已硬删用户调用 restore 未被拒绝，状态码 {resp.status_code}"
    assert 400 <= resp.status_code < 500, \
        f"期望 4xx，实际 {resp.status_code}"


def cleanup():
    """兜底清理"""
    try:
        cleanup_users_by_prefix(USERNAME)
    except Exception as e:
        print(f"[cleanup] 异常: {e}")


def main():
    health_check()
    try:
        run_tests(globals(), title="用户管理全生命周期",
                  ordered=True, abort_on_fail=True)
    finally:
        cleanup()


if __name__ == "__main__":
    main()
