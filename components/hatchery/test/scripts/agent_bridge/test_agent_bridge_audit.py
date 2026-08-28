#!/usr/bin/env python3
"""
集成测试：Agent-Bridge TAT 审计回调接口

覆盖接口：
    POST /agent-bridge/audit    TAT 审计回调写入
    GET  /admin/audit           审计日志查询（action/resource_id/user_id/username 筛选）

测试场景：
    1. 认证三件套（无认证 / 错误 token / 非管理员）
    2. 正常写入 → 通过 /admin/audit 验证落库
    3. action 前缀校验（必须以 agent_bridge_ 开头）
    4. status 白名单校验
    5. 必填字段校验（action / status 为空）
    6. 方法校验（GET → 405）
    7. action 筛选查询验证
    8. resource_id 筛选查询验证
    9. user_id、username 默认精确与显式 fuzzy=1 验证
    10. 多种合法 status 值写入验证
"""
import os
import sys
import time
from datetime import datetime, timezone

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers.api import (  # noqa: E402
    seed, anon, bad_token, ApiClient,
    health_check, run_tests, auth_test_suite,
    ADMIN_TOKEN,
)

# 用时间戳生成唯一前缀，避免与其他测试冲突
TS = int(time.time())
ACTION_PREFIX = f"agent_bridge_inttest_{TS}"


# ─── 工具函数 ───

def do_audit(body: dict, headers=None):
    """调用 POST /agent-bridge/audit，返回原始 Response。"""
    if headers:
        tmp = ApiClient("", timeout=30)
        return tmp.post("/agent-bridge/audit", json=body,
                        expect=None, raw=True, extra_headers=headers)
    return seed.post("/agent-bridge/audit", json=body, expect=None, raw=True)


def audit_query_params(action=None, resource_id=None, username=None, user_id=None,
                       fuzzy=False):
    """构造审计日志筛选参数。"""
    params = {}
    if action:
        params["action"] = action
    if resource_id:
        params["resource_id"] = resource_id
    if username:
        params["username"] = username
    if user_id is not None:
        params["user_id"] = user_id
    if fuzzy:
        params["fuzzy"] = "1"
    return params


def query_audit(action=None, resource_id=None, username=None, user_id=None,
                fuzzy=False, page_size=100):
    """调用 GET /admin/audit 查询审计日志。"""
    params = audit_query_params(action, resource_id, username, user_id, fuzzy)
    params["page_size"] = page_size
    return seed.get("/admin/audit", params=params)


# ─── 测试用例 ───

def test_01_auth():
    """认证三件套：无认证 / 错误 token → 401/403"""
    auth_test_suite(
        lambda headers: do_audit({
            "action": "agent_bridge_auth_test",
            "status": "success",
        }, headers=headers),
        label="agent_bridge_audit",
        check_admin=False,  # 非管理员专属接口，普通用户 hk- token 也可调用
    )


def test_02_normal_write_and_query():
    """正常写入审计记录 → 通过 /admin/audit 验证落库"""
    action = f"{ACTION_PREFIX}_normal"
    resource_id = f"ins-inttest-{TS}"

    resp = do_audit({
        "platform_id": "hatchery",
        "action": action,
        "resource": "instance",
        "resource_id": resource_id,
        "invocation_id": f"inv-{TS}",
        "script_name": "install_test.sh",
        "status": "success",
        "trace_id": f"trace-{TS}",
        "started_at": int(time.time()) - 10,
    })
    assert resp.status_code == 200, f"期望 200，实际 {resp.status_code}: {resp.text[:300]}"
    data = resp.json()
    assert data.get("ok") is True, f"期望 ok=true: {data}"

    # 等待异步写入完成
    time.sleep(1)

    # 通过 /admin/audit 验证落库
    audit_data = query_audit(action=action)
    logs = audit_data.get("logs", [])
    assert len(logs) >= 1, f"期望至少 1 条审计记录，实际 {len(logs)} 条"

    log = logs[0]
    assert log.get("action") == action, f"action 不匹配: {log.get('action')}"
    assert log.get("resource") == "instance", f"resource 不匹配: {log.get('resource')}"
    assert log.get("resource_id") == resource_id, f"resource_id 不匹配: {log.get('resource_id')}"
    assert log.get("status") == "success", f"status 不匹配: {log.get('status')}"
    print(f"    写入成功 ✓  action={action}, resource_id={resource_id}")


def test_03_action_prefix_required():
    """action 必须以 agent_bridge_ 开头 → 400"""
    resp = do_audit({
        "action": "desktop_install",  # 缺少 agent_bridge_ 前缀
        "status": "success",
    })
    assert resp.status_code == 400, f"期望 400，实际 {resp.status_code}"
    err = resp.json().get("error", "")
    assert "agent_bridge_" in err, f"错误信息应提及前缀要求: {err}"
    print(f"    前缀校验 ✓  error={err}")


def test_04_status_whitelist():
    """status 必须为合法值 → 400"""
    resp = do_audit({
        "action": "agent_bridge_test_status",
        "status": "unknown_status",
    })
    assert resp.status_code == 400, f"期望 400，实际 {resp.status_code}"
    err = resp.json().get("error", "")
    assert "success" in err and "failed" in err, f"错误信息应列出合法值: {err}"
    print(f"    status 白名单校验 ✓  error={err}")


def test_05_action_required():
    """action 为空 → 400"""
    resp = do_audit({
        "action": "",
        "status": "success",
    })
    assert resp.status_code == 400, f"期望 400，实际 {resp.status_code}"
    err = resp.json().get("error", "")
    assert "action" in err.lower(), f"错误信息应提及 action: {err}"
    print(f"    action 必填校验 ✓")


def test_06_status_required():
    """status 为空 → 400"""
    resp = do_audit({
        "action": "agent_bridge_test_empty_status",
        "status": "",
    })
    assert resp.status_code == 400, f"期望 400，实际 {resp.status_code}"
    err = resp.json().get("error", "")
    assert "status" in err.lower(), f"错误信息应提及 status: {err}"
    print(f"    status 必填校验 ✓")


def test_07_method_not_allowed():
    """GET /agent-bridge/audit → 405"""
    resp = seed.get("/agent-bridge/audit", expect=None, raw=True)
    assert resp.status_code == 405, f"期望 405，实际 {resp.status_code}"
    print(f"    方法校验 ✓  GET → 405")


def test_08_query_by_action_filter():
    """通过 action 前缀筛选审计记录"""
    # 写入一条带唯一 action 的记录
    unique_action = f"{ACTION_PREFIX}_filter_action"
    resp = do_audit({
        "action": unique_action,
        "resource": "instance",
        "resource_id": f"ins-filter-{TS}",
        "status": "failed",
    })
    assert resp.status_code == 200

    time.sleep(1)

    # 用 action 前缀筛选
    audit_data = query_audit(action=unique_action)
    logs = audit_data.get("logs", [])
    assert len(logs) >= 1, f"action 筛选应返回至少 1 条记录，实际 {len(logs)}"
    for log in logs:
        assert log.get("action").startswith(unique_action), \
            f"筛选结果 action 不匹配: {log.get('action')}"
    print(f"    action 筛选 ✓  返回 {len(logs)} 条")


def test_09_query_by_resource_id_filter():
    """通过 resource_id 精确筛选审计记录"""
    unique_resource = f"ins-resfilter-{TS}"
    resp = do_audit({
        "action": f"{ACTION_PREFIX}_filter_res",
        "resource": "instance",
        "resource_id": unique_resource,
        "status": "success",
    })
    assert resp.status_code == 200

    time.sleep(1)

    # 用 resource_id 精确筛选
    audit_data = query_audit(resource_id=unique_resource)
    logs = audit_data.get("logs", [])
    assert len(logs) >= 1, f"resource_id 筛选应返回至少 1 条记录，实际 {len(logs)}"
    for log in logs:
        assert log.get("resource_id") == unique_resource, \
            f"筛选结果 resource_id 不匹配: {log.get('resource_id')}"
    print(f"    resource_id 筛选 ✓  返回 {len(logs)} 条")


def test_10_user_filter_and_username_modes():
    """验证 user_id、username 默认精确和显式 fuzzy=1"""
    unique_action = f"{ACTION_PREFIX}_filter_username_{time.time_ns()}"
    resp = do_audit({
        "action": unique_action,
        "resource": "instance",
        "resource_id": f"ins-userfilter-{TS}",
        "status": "success",
    })
    assert resp.status_code == 200

    time.sleep(1)

    audit_data = query_audit(action=unique_action)
    logs = audit_data.get("logs", [])
    assert len(logs) >= 1, "应先通过 action 查询到审计记录"
    user_id = logs[0].get("user_id")
    username = logs[0].get("username")
    assert user_id is not None, f"审计记录缺少 user_id: {logs[0]}"
    assert username, f"审计记录缺少 username: {logs[0]}"

    selected_data = query_audit(action=unique_action, user_id=user_id)
    selected_logs = selected_data.get("logs", [])
    assert len(selected_logs) >= 1, "user_id 应命中审计记录"
    for log in selected_logs:
        assert log.get("user_id") == user_id, \
            f"user_id 返回了其他用户记录: {log.get('user_id')}"

    exact_data = query_audit(action=unique_action, username=username)
    assert len(exact_data.get("logs", [])) >= 1, \
        f"username 默认精确查询未命中: username={username}"

    username_fragment = username[:max(1, len(username) // 2)]
    if username_fragment != username:
        partial_exact_data = query_audit(
            action=unique_action, username=username_fragment)
        assert len(partial_exact_data.get("logs", [])) == 0, \
            "未传 fuzzy 时部分 username 不应命中"

    fuzzy_data = query_audit(
        action=unique_action, username=username_fragment, fuzzy=True)
    assert len(fuzzy_data.get("logs", [])) >= 1, \
        f"username fuzzy=1 查询未命中: fragment={username_fragment}"

    assert selected_data.get("total", 0) >= 1, \
        f"user_id 精确查询应返回同步总数: {selected_data}"
    print(f"    user_id/username 精确/fuzzy=1 ✓  user_id={user_id}")


def test_11_all_valid_statuses():
    """验证所有合法 status 值均可写入"""
    valid_statuses = ["success", "failed", "timeout", "dispatched"]
    for status in valid_statuses:
        resp = do_audit({
            "action": f"{ACTION_PREFIX}_status_{status}",
            "resource": "instance",
            "resource_id": f"ins-status-{TS}",
            "status": status,
        })
        assert resp.status_code == 200, \
            f"status={status} 应返回 200，实际 {resp.status_code}: {resp.text[:200]}"
    print(f"    所有合法 status 值写入 ✓  ({', '.join(valid_statuses)})")


def test_12_started_at_zero_uses_current_time():
    """started_at 为 0 时使用当前时间"""
    action = f"{ACTION_PREFIX}_started_zero"
    before = int(time.time())
    resp = do_audit({
        "action": action,
        "status": "success",
        "started_at": 0,
    })
    assert resp.status_code == 200

    time.sleep(1)

    audit_data = query_audit(action=action)
    logs = audit_data.get("logs", [])
    assert len(logs) >= 1, "应有审计记录"
    # started_at 应在 before 附近（±5s 容差）
    log = logs[0]
    started = log.get("started_at", "")
    if isinstance(started, (int, float)):
        started_ts = started
    else:
        # ISO 格式解析，如 "2026-06-18T17:13:37+08:00" 或 "2026-06-18T09:13:37Z"
        parsed = datetime.fromisoformat(started.replace("Z", "+00:00"))
        started_ts = parsed.timestamp()
    assert abs(started_ts - before) < 5, (
        f"started_at 与请求时间偏差过大: started_at={started}, before={before}"
    )
    print(f"    started_at=0 写入 ✓  (记录已落库, started_at 接近当前时间)")


def test_13_empty_body_returns_400():
    """空请求体 → 400"""
    resp = seed.post("/agent-bridge/audit", data="", expect=None, raw=True,
                     extra_headers={"Content-Type": "application/json"})
    # 空 body 解析失败或缺少必填字段
    assert 400 <= resp.status_code < 500, \
        f"期望 4xx，实际 {resp.status_code}"
    print(f"    空请求体 ✓  status={resp.status_code}")


# ─── 入口 ───

def main():
    health_check()
    run_tests(globals(), title="Agent-Bridge TAT 审计回调",
              ordered=True, abort_on_fail=True)


if __name__ == "__main__":
    main()
