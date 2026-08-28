#!/usr/bin/env python3
"""
存量实例分组归属处理 — 管理端 API 集成测试

覆盖接口：
    POST /admin/stale-instances/action-options   认证 / 参数校验 / 空场景 / 正常返回
    POST /admin/stale-instances/config-diff      认证 / 参数校验 / 正常返回
    POST /admin/stale-instances/apply            认证 / 参数校验 / 枚举校验 / 正常返回
    GET  /admin/stale-instances/records          认证 / 分页 / 过滤
    POST /admin/instances/group-check            认证 / 参数校验 / 正常返回
"""
import os
import sys
import time

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers.api import (  # noqa: E402
    seed, anon, bad_token, ApiClient,
    health_check, run_tests, auth_test_suite,
    IDENTIFIER,
)
from helpers.user_groups import (  # noqa: E402
    extract_group_id, cleanup_by_prefix,
)

PREFIX = f"it-stale-admin-{int(time.time())}"

state = {
    "group_a": None,      # 分组 A
    "group_b": None,      # 分组 B
    "user_a_id": None,    # 用户 A（在分组 A 中）
    "user_a_token": None,
    "user_b_id": None,    # 用户 B（在分组 A 中）
    "user_b_token": None,
    "instance_id": None,  # 用户 A 创建的实例 ID
}


# ─── 工具函数 ───

def _create_group(name):
    resp = seed.post("/admin/user-groups/create",
                     json={"name": name, "description": "stale-instances测试"},
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
    # 获取 token
    seed.post("/admin/token/enable", params={"id": uid}, expect=None, raw=True)
    token_data = seed.get("/admin/user-token", params={"id": uid}, expect=None, raw=True)
    token = None
    if token_data.status_code == 200:
        token = token_data.json().get("token")
    return uid, token


def _add_member(group_id, user_id):
    seed.post("/admin/user-groups/members/add",
              json={"id": group_id, "user_ids": [user_id]},
              expect=None, raw=True)


def _remove_member(group_id, user_id):
    seed.post("/admin/user-groups/members/remove",
              json={"id": group_id, "user_ids": [user_id]},
              expect=None, raw=True)


# ─── 前置准备 ───

def test_01_prepare_groups_users():
    """创建 2 个分组 + 2 个用户（都在分组 A 中）"""
    state["group_a"] = _create_group(f"{PREFIX}-GA")
    state["group_b"] = _create_group(f"{PREFIX}-GB")
    assert state["group_a"], "分组 A 创建失败"
    assert state["group_b"], "分组 B 创建失败"

    state["user_a_id"], state["user_a_token"] = _create_user(
        f"{PREFIX}-uA", [state["group_a"]])
    state["user_b_id"], state["user_b_token"] = _create_user(
        f"{PREFIX}-uB", [state["group_a"]])
    assert state["user_a_id"], "用户 A 创建失败"
    assert state["user_b_id"], "用户 B 创建失败"
    assert state["user_a_token"], "用户 A token 获取失败"
    assert state["user_b_token"], "用户 B token 获取失败"
    print(f"    GA={state['group_a']} GB={state['group_b']} "
          f"uA={state['user_a_id']} uB={state['user_b_id']}")


# ─── action-options: 认证 / 校验 / 空场景 ───

def test_02_action_options_auth():
    """action-options: 无认证 / 错误 token / 非 admin → 401/403"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).post(
            "/admin/stale-instances/action-options",
            json={}, expect=None, raw=True, extra_headers=headers),
        label="action-options",
    )


def test_03_action_options_method_not_allowed():
    """action-options: GET → 405"""
    resp = seed.get("/admin/stale-instances/action-options", expect=None, raw=True)
    assert resp.status_code == 405, f"GET 应返回 405，实际 {resp.status_code}"


def test_04_action_options_bad_json():
    """action-options: 非 JSON body → 400"""
    resp = seed.post("/admin/stale-instances/action-options",
                     data="not-json", expect=None, raw=True)
    assert resp.status_code == 400, f"非 JSON 应返回 400，实际 {resp.status_code}"


def test_05_action_options_empty():
    """action-options: 空请求 → 200，no_group/user_removed/subtree 三段"""
    data = seed.post("/admin/stale-instances/action-options", json={})
    assert data.get("ok") is True
    assert "no_group" in data
    assert "user_removed" in data
    assert "subtree" in data
    print(f"    no_group={len(data['no_group'].get('options', []))} "
          f"user_removed={len(data['user_removed'].get('options', []))} "
          f"subtree={len(data['subtree'].get('groups', []))}")


def test_06_action_options_with_user_group_pairs():
    """action-options: 传 user_group_ids → 200，user_removed 非空"""
    data = seed.post("/admin/stale-instances/action-options", json={
        "user_group_ids": [
            {"user_id": state["user_a_id"], "group_id": state["group_a"]},
        ],
    })
    assert data.get("ok") is True
    # user_removed 应包含用户 A 的实例（目前还没有实例，但结构应正确）
    opts = data.get("user_removed", {}).get("options", [])
    print(f"    user_removed options={len(opts)}")


def test_07_action_options_with_group_ids():
    """action-options: 传 group_ids → 200，subtree 非空"""
    data = seed.post("/admin/stale-instances/action-options", json={
        "group_ids": [state["group_a"]],
    })
    assert data.get("ok") is True
    groups = data.get("subtree", {}).get("groups", [])
    print(f"    subtree groups={len(groups)}")


# ─── config-diff: 认证 / 校验 ───

def test_08_config_diff_auth():
    """config-diff: 无认证 → 401"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).post(
            "/admin/stale-instances/config-diff",
            json={"instance_ids": [1], "target_group_id": 1},
            expect=None, raw=True, extra_headers=headers),
        label="config-diff",
    )


def test_09_config_diff_validation():
    """config-diff: 缺 instance_ids / 缺 target_group_id / 超量 → 400"""
    # 缺 instance_ids
    resp = seed.post("/admin/stale-instances/config-diff",
                     json={"target_group_id": 1}, expect=None, raw=True)
    assert resp.status_code == 400, f"缺 instance_ids 应 400，实际 {resp.status_code}"

    # 缺 target_group_id
    resp = seed.post("/admin/stale-instances/config-diff",
                     json={"instance_ids": [1]}, expect=None, raw=True)
    assert resp.status_code == 400, f"缺 target_group_id 应 400，实际 {resp.status_code}"

    # 超量 instance_ids
    ids = list(range(1, 102))
    resp = seed.post("/admin/stale-instances/config-diff",
                     json={"instance_ids": ids, "target_group_id": 1},
                     expect=None, raw=True)
    assert resp.status_code == 400, f"超量应 400，实际 {resp.status_code}"


# ─── apply: 认证 / 校验 / 枚举 ───

def test_10_apply_auth():
    """apply: 无认证 → 401"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).post(
            "/admin/stale-instances/apply",
            json={"trigger_source": "user_edit", "actions": []},
            expect=None, raw=True, extra_headers=headers),
        label="apply",
    )


def test_11_apply_validation():
    """apply: 参数校验各场景 → 400"""
    # 缺 trigger_source
    resp = seed.post("/admin/stale-instances/apply",
                     json={"actions": []}, expect=None, raw=True)
    assert resp.status_code == 400

    # 非法 trigger_source
    resp = seed.post("/admin/stale-instances/apply",
                     json={"trigger_source": "invalid", "actions": []},
                     expect=None, raw=True)
    assert resp.status_code == 400

    # 空 actions
    resp = seed.post("/admin/stale-instances/apply",
                     json={"trigger_source": "user_edit", "actions": []},
                     expect=None, raw=True)
    assert resp.status_code == 400

    # 非法 action 枚举
    resp = seed.post("/admin/stale-instances/apply",
                     json={"trigger_source": "user_edit",
                           "actions": [{"action": "delete", "instance_ids": [1]}]},
                     expect=None, raw=True)
    assert resp.status_code == 400

    # migrate 缺 target_group_id
    resp = seed.post("/admin/stale-instances/apply",
                     json={"trigger_source": "user_edit",
                           "actions": [{"action": "migrate", "instance_ids": [1]}]},
                     expect=None, raw=True)
    assert resp.status_code == 400

    # handover 缺 target_user_id
    resp = seed.post("/admin/stale-instances/apply",
                     json={"trigger_source": "user_edit",
                           "actions": [{"action": "handover", "instance_ids": [1]}]},
                     expect=None, raw=True)
    assert resp.status_code == 400

    # 空 instance_ids
    resp = seed.post("/admin/stale-instances/apply",
                     json={"trigger_source": "user_edit",
                           "actions": [{"action": "archive_stop", "instance_ids": []}]},
                     expect=None, raw=True)
    assert resp.status_code == 400


# ─── records: 认证 / 查询 ───

def test_12_records_auth():
    """records: 无认证 → 401"""
    resp = anon.get("/admin/stale-instances/records", expect=None, raw=True)
    assert resp.status_code in (401, 403), f"无认证应 401/403，实际 {resp.status_code}"


def test_13_records_method_not_allowed():
    """records: POST → 405"""
    resp = seed.post("/admin/stale-instances/records", expect=None, raw=True)
    assert resp.status_code == 405


def test_14_records_query():
    """records: GET 查询 → 200，含分页字段"""
    data = seed.get("/admin/stale-instances/records",
                    params={"page": 1, "page_size": 5})
    assert data.get("ok") is True
    assert "total" in data
    assert "records" in data
    assert "page" in data
    assert "page_size" in data
    print(f"    total={data['total']} page={data['page']} page_size={data['page_size']}")


def test_15_records_query_with_filters():
    """records: 带过滤条件查询 → 200"""
    data = seed.get("/admin/stale-instances/records", params={
        "action": "migrate",
        "trigger_source": "user_edit",
        "page": 1, "page_size": 10,
    })
    assert data.get("ok") is True
    print(f"    filtered total={data['total']}")


def test_16_records_query_with_instance_id():
    """records: 按字符串 instance_id 查询 → 200"""
    data = seed.get("/admin/stale-instances/records", params={
        "instance_id": "ins-nonexistent",
    })
    assert data.get("ok") is True
    assert data["total"] == 0


# ─── group-check: 认证 / 校验 / 正常返回 ───

def test_17_group_check_auth():
    """group-check: 无认证 → 401"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).post(
            "/admin/instances/group-check",
            json={"ids": [1]}, expect=None, raw=True, extra_headers=headers),
        label="group-check",
    )


def test_18_group_check_validation():
    """group-check: 非 JSON / 空 ids → 400/200"""
    # 非 JSON
    resp = seed.post("/admin/instances/group-check",
                     data="bad", expect=None, raw=True)
    assert resp.status_code == 400

    # 空 ids（去重去零后为空）→ 200 空结果
    data = seed.post("/admin/instances/group-check",
                     json={"ids": [0, 0], "check_user_group": True})
    assert data.get("ok") is not False
    results = data.get("results", [])
    assert len(results) == 0


def test_19_group_check_success():
    """group-check: 正常请求 → 200，返回 results 数组"""
    # 使用不存在的实例 ID，预期返回空 results（DB 查不到）
    data = seed.post("/admin/instances/group-check", json={
        "ids": [999999],
        "check_user_group": True,
        "check_config_drift": False,
    })
    assert "results" in data
    # DB 中不存在该实例，results 应为空
    print(f"    results count={len(data['results'])}")


# ─── apply: 正常流程（migrate 场景 B → target=0） ───

def test_20_apply_nonexistent_instance():
    """apply: 对不存在的实例执行 archive_stop → results 含 failed"""
    data = seed.post("/admin/stale-instances/apply", json={
        "trigger_source": "user_edit",
        "actions": [{"action": "archive_stop", "instance_ids": [999999]}],
    })
    assert data.get("ok") is True
    results = data.get("results", [])
    assert len(results) == 1
    assert results[0]["status"] == "failed", f"不存在的实例应 failed，实际 {results[0]['status']}"
    print(f"    result: {results[0]}")


def test_21_apply_migrate_nonexistent_target_group():
    """apply: migrate 目标分组不存在 → failed"""
    data = seed.post("/admin/stale-instances/apply", json={
        "trigger_source": "user_edit",
        "actions": [{"action": "migrate", "instance_ids": [999999],
                      "target_group_id": 999999}],
    })
    assert data.get("ok") is True
    results = data.get("results", [])
    assert len(results) == 1
    assert results[0]["status"] == "failed"


def test_22_apply_too_many_instances():
    """apply: instance_ids 总数超 500 → 400"""
    ids = list(range(1, 502))
    resp = seed.post("/admin/stale-instances/apply",
                     json={"trigger_source": "user_edit",
                           "actions": [{"action": "archive_stop", "instance_ids": ids}]},
                     expect=None, raw=True)
    assert resp.status_code == 400, f"超量应 400，实际 {resp.status_code}"


# ─── 清理 ───

def cleanup():
    try:
        cleanup_by_prefix(group_prefix=PREFIX, user_prefix=PREFIX)
    except Exception as e:
        print(f"[cleanup] {e}")


# ─── 入口 ───

def main():
    health_check()
    try:
        run_tests(globals(), title="存量实例分组归属处理 — 管理端 API",
                  ordered=True, abort_on_fail=True)
    finally:
        cleanup()


if __name__ == "__main__":
    main()
