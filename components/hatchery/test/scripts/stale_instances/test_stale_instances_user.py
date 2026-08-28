#!/usr/bin/env python3
"""
存量实例分组归属处理 — 用户端 API 集成测试

覆盖接口：
    POST /openclaw/stale-instances/rebind       认证 / 参数校验 / 权限 / 正常流程
    POST /openclaw/stale-instances/initiate     认证 / 参数校验 / 权限 / 正常流程
    POST /openclaw/stale-instances/cancel       认证 / 权限 / 正常流程
    POST /openclaw/stale-instances/accept       认证 / 权限 / 正常流程
    POST /openclaw/stale-instances/reject       认证 / 权限 / 正常流程

前置依赖：创建分组 → 创建用户 → 创建实例 → admin apply pending_user → 用户端操作
"""
import os
import sys
import time
import traceback

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers.api import (  # noqa: E402
    seed, anon, bad_token, ApiClient,
    health_check, IDENTIFIER,
)
from helpers.user_groups import (  # noqa: E402
    extract_group_id, cleanup_by_prefix,
)
from helpers.instance import (  # noqa: E402
    create_instance, get_instance_db_id, wait_instance_ready,
)
from helpers.user_mgmt import (  # noqa: E402
    admin_create_user, admin_get_user_token, admin_enable_token,
)

PREFIX = f"it-stale-user-{int(time.time())}"

state = {
    "group_a": None,
    "group_b": None,
    "user_a_id": None,
    "user_a_token": None,
    "user_b_id": None,
    "user_b_token": None,
    "instance_db_id": None,
}


def _create_group(name):
    resp = seed.post("/admin/user-groups/create",
                     json={"name": name, "description": "stale-user测试"},
                     expect=None, raw=True)
    if resp.status_code == 200:
        return extract_group_id(resp.json())
    return None


def _create_user(username, group_ids=None):
    payload = {
        "username": username,
        "password": "Aa12345!",
        "role": "user",
        "instance_quota": "3",
        "token_quota_day": "500000",
    }
    # /admin/create 只支持 form data，group_ids 通过 members/add 单独加入
    resp = seed.post("/admin/create", data=payload, expect=None, raw=True)
    if resp.status_code != 200:
        return None, None
    uid = resp.json().get("id")
    if group_ids:
        for gid in group_ids:
            _add_member(gid, uid)
    admin_enable_token(seed.token, uid)
    token_data = seed.get("/admin/user-token", params={"id": uid}, expect=None, raw=True)
    token = None
    if token_data.status_code == 200:
        token = token_data.json().get("token")
    return uid, token


def _user_client(token):
    """创建带 X-OpenAPI:1 的用户客户端"""
    return ApiClient(token, openapi=True)


def _add_member(group_id, user_id):
    seed.post("/admin/user-groups/members/add",
              json={"id": group_id, "user_ids": [user_id]},
              expect=None, raw=True)


def _remove_member(group_id, user_id):
    seed.post("/admin/user-groups/members/remove",
              json={"id": group_id, "user_ids": [user_id]},
              expect=None, raw=True)


def _set_pending_user(inst_id, allow_migrate=False, allow_handover=False):
    """通过 admin apply 给实例打 pending_user 标记"""
    actions = [{"action": "pending_user", "instance_ids": [inst_id]}]
    if allow_migrate:
        actions[0]["allow_migrate"] = True
    if allow_handover:
        actions[0]["allow_same_group_handover"] = True
    if not allow_migrate and not allow_handover:
        actions[0]["allow_migrate"] = True  # 至少一个子选项
    resp = seed.post("/admin/stale-instances/apply", json={
        "trigger_source": "user_edit",
        "actions": actions,
    }, expect=None, raw=True)
    assert resp.status_code == 200, \
        f"admin apply pending_user 失败: {resp.status_code} {resp.text[:300]}"
    # 检查 result status，noop 说明场景未对齐
    results = resp.json().get("results", [])
    for r in results:
        if r.get("status") != "success":
            print(f"    [WARN] apply result: {r}")
        assert r.get("status") == "success", \
            f"pending_user 未生效 (status={r.get('status')}): {r}"


def main():
    health_check()
    print()

    # ─── Step 0: 创建分组 + 用户 ───
    print(">>> Step 0: 创建分组 + 用户 ...")
    state["group_a"] = _create_group(f"{PREFIX}-GA")
    state["group_b"] = _create_group(f"{PREFIX}-GB")
    assert state["group_a"] and state["group_b"], "分组创建失败"

    state["user_a_id"], state["user_a_token"] = _create_user(
        f"{PREFIX}-uA", [state["group_a"]])
    state["user_b_id"], state["user_b_token"] = _create_user(
        f"{PREFIX}-uB", [state["group_a"]])
    assert state["user_a_id"] and state["user_b_id"], "用户创建失败"
    assert state["user_a_token"] and state["user_b_token"], "Token 获取失败"
    print(f"    GA={state['group_a']} GB={state['group_b']} "
          f"uA={state['user_a_id']} uB={state['user_b_id']} ✓")

    try:
        # ─── Step 1: 创建实例 ───
        print(">>> Step 1: 用户 A 创建实例 ...")
        user_a = _user_client(state["user_a_token"])
        create_data = create_instance(state["user_a_token"], f"{PREFIX}-inst",
                                      group_id=state["group_a"])
        assert create_data.get("ok"), f"创建实例失败: {create_data}"
        cvm_id = create_data.get("instance_id", "")
        state["instance_db_id"] = get_instance_db_id(state["user_a_token"], cvm_id)
        assert state["instance_db_id"], "获取实例 DB ID 失败"
        print(f"    实例创建成功 ✓  db_id={state['instance_db_id']} cvm={cvm_id}")

        # 等待实例就绪
        print(">>> 等待实例就绪 ...")
        wait_instance_ready(state["user_a_token"], state["instance_db_id"])
        print("    实例就绪 ✓")

        inst_id = state["instance_db_id"]

        # ─── Step 2: 用户端 API 认证测试 ───
        print(">>> Step 2: 用户端 API 认证测试 ...")

        # rebind: 无认证 → 401
        resp = anon.post("/openclaw/stale-instances/rebind",
                         json={"id": inst_id, "target_group_id": state["group_a"]},
                         expect=None, raw=True)
        assert resp.status_code in (401, 403), \
            f"rebind 无认证应 401/403，实际 {resp.status_code}"

        # rebind: GET → 405
        resp = user_a.get("/openclaw/stale-instances/rebind", expect=None, raw=True)
        assert resp.status_code == 405, f"rebind GET 应 405，实际 {resp.status_code}"

        # rebind: 缺参数 → 400
        resp = user_a.post("/openclaw/stale-instances/rebind",
                           json={}, expect=None, raw=True)
        assert resp.status_code == 400, f"rebind 缺参数应 400，实际 {resp.status_code}"

        # initiate: 无认证 → 401
        resp = anon.post("/openclaw/stale-instances/initiate",
                         json={"id": inst_id, "target_username": f"{PREFIX}-uB"},
                         expect=None, raw=True)
        assert resp.status_code in (401, 403), \
            f"initiate 无认证应 401/403，实际 {resp.status_code}"

        # initiate: GET → 405
        resp = user_a.get("/openclaw/stale-instances/initiate", expect=None, raw=True)
        assert resp.status_code == 405

        print("    认证测试通过 ✓")

        # ─── Step 3: rebind 流程 ───
        print(">>> Step 3: rebind 流程（pending_user + allow_migrate）...")

        # 3a: 先把用户 A 移出分组 A（创建 stale 场景）
        _remove_member(state["group_a"], state["user_a_id"])
        time.sleep(1)

        # 3b: admin apply pending_user + allow_migrate
        _set_pending_user(inst_id, allow_migrate=True)
        print("    已打 pending_user + allow_migrate 标记 ✓")

        # 3c: rebind 没有标记的实例 → 400（先测试另一个实例不存在的情况）
        resp = user_a.post("/openclaw/stale-instances/rebind",
                           json={"id": 999999, "target_group_id": state["group_a"]},
                           expect=None, raw=True)
        assert resp.status_code == 404, f"不存在实例应 404，实际 {resp.status_code}"

        # 3d: rebind 成功（用户 A 重新加入分组 A）
        _add_member(state["group_a"], state["user_a_id"])
        time.sleep(1)
        resp = user_a.post("/openclaw/stale-instances/rebind",
                           json={"id": inst_id, "target_group_id": state["group_a"]},
                           expect=None, raw=True)
        assert resp.status_code == 200, \
            f"rebind 应 200，实际 {resp.status_code} body={resp.text[:300]}"
        data = resp.json()
        assert data.get("ok") is True
        print("    rebind 成功 ✓")

        # ─── Step 4: handover 流程（initiate → accept） ───
        print(">>> Step 4: handover 流程（initiate → accept）...")

        # 4a: 将用户 A 移到分组 B（scenario A，保留 allowHandover），再打 pending_user + allow_same_group_handover
        _remove_member(state["group_a"], state["user_a_id"])
        _add_member(state["group_b"], state["user_a_id"])
        time.sleep(1)
        _set_pending_user(inst_id, allow_migrate=True, allow_handover=True)
        print("    已打 pending_user + allow_same_group_handover 标记 ✓")

        # 4b: initiate — 目标用户不存在
        resp = user_a.post("/openclaw/stale-instances/initiate",
                           json={"id": inst_id, "target_username": "nobody"},
                           expect=None, raw=True)
        assert resp.status_code == 400, \
            f"initiate 目标不存在应 400，实际 {resp.status_code}"

        # 4c: initiate — 目标是自己
        resp = user_a.post("/openclaw/stale-instances/initiate",
                           json={"id": inst_id, "target_username": f"{PREFIX}-uA"},
                           expect=None, raw=True)
        assert resp.status_code == 400, \
            f"initiate 自己应 400，实际 {resp.status_code}"

        # 4d: initiate 成功
        resp = user_a.post("/openclaw/stale-instances/initiate",
                           json={"id": inst_id, "target_username": f"{PREFIX}-uB"},
                           expect=None, raw=True)
        assert resp.status_code == 200, \
            f"initiate 应 200，实际 {resp.status_code} body={resp.text[:300]}"
        print("    initiate 成功 ✓")

        # 4e: cancel — 非 owner 操作
        user_b = _user_client(state["user_b_token"])
        resp = user_b.post("/openclaw/stale-instances/cancel",
                           json={"id": inst_id}, expect=None, raw=True)
        assert resp.status_code == 403, \
            f"cancel 非 owner 应 403，实际 {resp.status_code}"

        # 4f: cancel 成功（owner 取消）
        resp = user_a.post("/openclaw/stale-instances/cancel",
                           json={"id": inst_id}, expect=None, raw=True)
        assert resp.status_code == 200, \
            f"cancel 应 200，实际 {resp.status_code} body={resp.text[:300]}"
        print("    cancel 成功 ✓")

        # 4g: 再次 initiate
        resp = user_a.post("/openclaw/stale-instances/initiate",
                           json={"id": inst_id, "target_username": f"{PREFIX}-uB"},
                           expect=None, raw=True)
        assert resp.status_code == 200

        # 4h: accept — 非目标用户操作
        resp = user_a.post("/openclaw/stale-instances/accept",
                           json={"id": inst_id}, expect=None, raw=True)
        assert resp.status_code == 403, \
            f"accept 非目标应 403，实际 {resp.status_code}"

        # 4i: accept 成功（目标用户 B 接收）
        resp = user_b.post("/openclaw/stale-instances/accept",
                           json={"id": inst_id}, expect=None, raw=True)
        assert resp.status_code == 200, \
            f"accept 应 200，实际 {resp.status_code} body={resp.text[:300]}"
        print("    accept 成功 ✓")

        # ─── Step 5: handover reject 流程 ───
        print(">>> Step 5: handover reject 流程 ...")

        # 把用户 A 加回分组 A（作为 reject 流程的目标用户）
        _add_member(state["group_a"], state["user_a_id"])
        # 将用户 B 移到分组 B（scenario A，保留 allowHandover），再重新打标记
        _remove_member(state["group_a"], state["user_b_id"])
        _add_member(state["group_b"], state["user_b_id"])
        time.sleep(1)
        _set_pending_user(inst_id, allow_migrate=True, allow_handover=True)

        # 5a: 用户 B 发起移交给用户 A
        resp = user_b.post("/openclaw/stale-instances/initiate",
                           json={"id": inst_id, "target_username": f"{PREFIX}-uA"},
                           expect=None, raw=True)
        assert resp.status_code == 200, \
            f"initiate(B→A) 应 200，实际 {resp.status_code}"

        # 5b: reject — 非目标用户操作
        resp = user_b.post("/openclaw/stale-instances/reject",
                           json={"id": inst_id}, expect=None, raw=True)
        assert resp.status_code == 403, \
            f"reject 非目标应 403，实际 {resp.status_code}"

        # 5c: reject 成功（目标用户 A 拒绝）
        resp = user_a.post("/openclaw/stale-instances/reject",
                           json={"id": inst_id}, expect=None, raw=True)
        assert resp.status_code == 200, \
            f"reject 应 200，实际 {resp.status_code} body={resp.text[:300]}"
        print("    reject 成功 ✓")

        # ─── Step 6: cancel 无活跃移交 ───
        print(">>> Step 6: 边界场景 ...")

        # cancel 无活跃移交 → 400
        resp = user_b.post("/openclaw/stale-instances/cancel",
                           json={"id": inst_id}, expect=None, raw=True)
        assert resp.status_code == 400, \
            f"cancel 无活跃应 400，实际 {resp.status_code}"

        # accept 无活跃移交 → 400
        resp = user_a.post("/openclaw/stale-instances/accept",
                           json={"id": inst_id}, expect=None, raw=True)
        assert resp.status_code == 400, \
            f"accept 无活跃应 400，实际 {resp.status_code}"

        # reject 无活跃移交 → 400
        resp = user_b.post("/openclaw/stale-instances/reject",
                           json={"id": inst_id}, expect=None, raw=True)
        assert resp.status_code == 400, \
            f"reject 无活跃应 400，实际 {resp.status_code}"

        print("    边界场景测试通过 ✓")

        # ─── Step 7: 不存在的实例 ───
        print(">>> Step 7: 不存在的实例 ...")

        for endpoint in ("cancel", "accept", "reject"):
            resp = user_b.post(f"/openclaw/stale-instances/{endpoint}",
                               json={"id": 999999}, expect=None, raw=True)
            assert resp.status_code == 404, \
                f"{endpoint} 不存在实例应 404，实际 {resp.status_code}"
        print("    不存在实例测试通过 ✓")

        print()
        print("存量实例分组归属处理 — 用户端 API 测试通过 ✅")

    except Exception as e:
        print(f"\n测试失败 ❌: {e}")
        traceback.print_exc()
        sys.exit(1)

    finally:
        # 清理
        try:
            # 删除实例
            if state.get("instance_db_id"):
                try:
                    seed.post("/admin/instances/delete",
                              json={"ids": [state["instance_db_id"]]},
                              expect=None, raw=True, timeout=60)
                except Exception:
                    pass
            cleanup_by_prefix(group_prefix=PREFIX, user_prefix=PREFIX)
        except Exception as e:
            print(f"[cleanup] {e}")


if __name__ == "__main__":
    main()
