#!/usr/bin/env python3
"""
集成测试：用户 Token 管理

覆盖接口：
    GET  /admin/user-token
    POST /admin/token/disable
    POST /admin/token/enable
    POST /admin/export-tokens
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

PREFIX = f"it-tok-{int(time.time())}"
state = {"uid": None}


def test_01_setup_user():
    """前置：创建测试用户"""
    r = seed.post("/admin/create", data={
        "username": f"{PREFIX}-u",
        "password": "Aa12345!",
        "role": "user",
    }, raw=True)
    uid = extract_uid(r.json())
    if not uid:
        users = seed.get("/admin/users",
                         params={"username": f"{PREFIX}-u"}).get("users") or []
        uid = users and (users[0].get("id") or users[0].get("ID"))
    assert uid, "前置失败：未取到 uid"
    state["uid"] = uid
    print(f"    uid={uid}")


def test_02_query_token():
    """查询用户 Token 应 exists=true"""
    uid = state["uid"]
    data = seed.get("/admin/user-token", params={"id": uid})
    exists = data.get("exists")
    if exists is None:
        exists = data.get("Exists")
    assert exists, f"exists 应为 true: {data}"
    assert data.get("token") or data.get("Token"), \
        f"应返回 token 字段: {data}"


def test_03_query_token_unknown():
    """查询不存在用户 Token 应 4xx"""
    resp = seed.get("/admin/user-token", params={"id": 9999999}, expect=None, raw=True)
    assert 400 <= resp.status_code < 500, \
        f"期望 4xx，实际 {resp.status_code}"


def test_04_disable_enable_cycle():
    """禁用 → 重复禁用 4xx → 启用 → 重复启用 4xx"""
    uid = state["uid"]
    seed.post("/admin/token/disable", params={"id": uid}, data={}, raw=True)

    r = seed.post("/admin/token/disable", params={"id": uid}, data={}, expect=None, raw=True)
    assert not (200 <= r.status_code < 300), \
        f"重复禁用未被拒绝 status={r.status_code}"

    seed.post("/admin/token/enable", params={"id": uid}, data={}, raw=True)

    r = seed.post("/admin/token/enable", params={"id": uid}, data={}, expect=None, raw=True)
    assert not (200 <= r.status_code < 300), \
        f"重复启用未被拒绝 status={r.status_code}"


def test_05_export_tokens():
    """批量导出 Token /admin/export-tokens"""
    resp = seed.post("/admin/export-tokens", data={}, raw=True)
    ctype = resp.headers.get("Content-Type", "")
    if "json" in ctype:
        data = resp.json()
        if isinstance(data, dict):
            items = data.get("tokens") or data.get("results") or data.get("users") or []
        else:
            items = data
        assert isinstance(items, list), f"返回应为列表: {data}"
        print(f"    导出 {len(items)} 条")
    else:
        body = resp.text or ""
        assert body.strip(), "导出内容为空"
        print(f"    导出内容长度 {len(body)} (Content-Type={ctype})")


def test_06_export_tokens_method_not_allowed():
    """GET /admin/export-tokens 应 4xx/405（仅支持 POST）"""
    resp = seed.get("/admin/export-tokens", expect=None, raw=True)
    assert 400 <= resp.status_code < 500, \
        f"期望 4xx，实际 {resp.status_code}"


def test_07_export_tokens_after_hard_delete():
    """目标用户被硬删后再次导出 Token，接口应仍可用"""
    uid = state["uid"]
    # 先把当前测试用户硬删
    seed.post("/admin/hard-delete", params={"id": uid}, data={}, expect=None, raw=True)
    state["uid"] = None  # 防止 cleanup 重复硬删失败
    resp = seed.post("/admin/export-tokens", data={}, expect=None, raw=True)
    assert 200 <= resp.status_code < 300, \
        f"删除后导出应仍 2xx，实际 {resp.status_code}"
    ctype = resp.headers.get("Content-Type", "")
    if "json" in ctype:
        data = resp.json()
        items = (data.get("tokens") or data.get("results")
                 or data.get("users") or []) if isinstance(data, dict) else data
        assert isinstance(items, list), f"返回应为列表: {data}"
        print(f"    删除后仍可导出 {len(items)} 条")
    else:
        print(f"    删除后仍可导出 (Content-Type={ctype}, len={len(resp.text or '')})")


def cleanup():
    """兜底清理"""
    try:
        cleanup_users_by_prefix(PREFIX)
    except Exception as e:
        print(f"[cleanup] 异常: {e}")


def main():
    health_check()
    try:
        run_tests(globals(), title="用户 Token 管理",
                  ordered=True, abort_on_fail=True)
    finally:
        cleanup()


if __name__ == "__main__":
    main()
