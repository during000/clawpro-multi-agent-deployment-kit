#!/usr/bin/env python3
"""
集成测试：实例管理 - 消息通知（H 组）

覆盖接口：
    GET  /openclaw/notifications              通知列表（分页 + 过滤）
    POST /openclaw/notifications/read         标记已读（单条/全部/按类别）
    GET  /openclaw/notifications/count        未读计数（按类别）
    POST /openclaw/notifications/delete       删除（单条/批量/全部/按类别）

通知是用户级资源（与实例无关），不依赖共享实例。
"""
import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers.api import (
    ApiClient,
    anon, bad_token,
    health_check, run_tests,
    auth_test_suite, assert_status,
)
from _instance_helpers import (
    cli,
    assert_error_message,
    assert_json_keys,
)


# ─── /notifications ──────────────────────────────────────────────────────

def test_01_notif_list_default():
    """GET /openclaw/notifications - 默认分页"""
    resp = cli.get("/openclaw/notifications", raw=True)
    body = assert_json_keys(resp, "notifications", "page", "page_size", "total")
    assert isinstance(body["notifications"], list), (
        f"notifications 应为数组: {type(body['notifications']).__name__}"
    )
    assert body["page"] == 1, f"默认 page 应为 1, 实际 {body['page']}"
    assert body["page_size"] == 20, (
        f"默认 page_size 应为 20, 实际 {body['page_size']}"
    )
    print(f"    OK total={body['total']} count={len(body['notifications'])}")


def test_02_notif_list_pagination():
    """GET /openclaw/notifications?page=1&page_size=5"""
    resp = cli.get(
        "/openclaw/notifications",
        params={"page": 1, "page_size": 5},
        raw=True,
    )
    body = assert_json_keys(resp, "notifications", "page_size")
    assert body["page_size"] == 5, f"page_size 应为 5: {body['page_size']}"
    assert len(body["notifications"]) <= 5, (
        f"分页限制未生效: {len(body['notifications'])}"
    )
    print(f"    OK page_size={body['page_size']}")


def test_03_notif_filter_by_is_read():
    """GET /openclaw/notifications?is_read=true（仅已读）"""
    resp = cli.get(
        "/openclaw/notifications",
        params={"is_read": "true"},
        raw=True,
    )
    body = assert_json_keys(resp, "notifications")
    for n in body["notifications"]:
        assert n.get("IsRead"), (
            f"is_read=true 时返回的通知 IsRead 应为 true: {n}"
        )
    print(f"    OK all_read={len(body['notifications'])}")


def test_04_notif_filter_by_category_valid():
    """GET /openclaw/notifications?category=success"""
    resp = cli.get(
        "/openclaw/notifications",
        params={"category": "success"},
        raw=True,
    )
    body = assert_json_keys(resp, "notifications")
    for n in body["notifications"]:
        assert n.get("Category") == "success", (
            f"category=success 但返回 {n.get('Category')}: {n}"
        )
    print(f"    OK count={len(body['notifications'])}")


def test_05_notif_filter_by_category_invalid():
    """GET /openclaw/notifications?category=__bad__ → 400"""
    resp = cli.get(
        "/openclaw/notifications",
        params={"category": "__bad__"},
        expect=None, raw=True,
    )
    assert_status(resp, 400, label="notif-bad-category")
    assert_error_message(resp, "category")
    print("    OK")


def test_06_notif_list_auth():
    """GET /openclaw/notifications - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).get(
            "/openclaw/notifications",
            expect=None, raw=True, extra_headers=headers,
        ),
        label="notifications",
        check_admin=False,
    )


# ─── /notifications/count ────────────────────────────────────────────────

def test_07_notif_count_ok():
    """GET /openclaw/notifications/count"""
    resp = cli.get("/openclaw/notifications/count", raw=True)
    body = assert_json_keys(resp, "unread_count", "unread_by_category")
    assert isinstance(body["unread_count"], int), (
        f"unread_count 应为 int: {body}"
    )
    assert isinstance(body["unread_by_category"], dict), (
        f"unread_by_category 应为 dict: {body}"
    )
    for k in body["unread_by_category"].keys():
        assert k in ("success", "error", "notice"), (
            f"unread_by_category 含未知 key {k}"
        )
    print(
        f"    OK unread={body['unread_count']} "
        f"by_cat={body['unread_by_category']}"
    )


def test_08_notif_count_auth():
    """GET /openclaw/notifications/count - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).get(
            "/openclaw/notifications/count",
            expect=None, raw=True, extra_headers=headers,
        ),
        label="notifications-count",
        check_admin=False,
    )


# ─── /notifications/read ─────────────────────────────────────────────────

def test_09_notif_read_all():
    """POST /openclaw/notifications/read - 全部已读 {id:0}"""
    body = cli.post("/openclaw/notifications/read", json={"id": 0}, expect=200)
    assert body.get("ok"), f"ok 应为 true: {body}"
    print("    OK")


def test_10_notif_read_by_category():
    """POST /openclaw/notifications/read - {id:0, category:notice}"""
    body = cli.post(
        "/openclaw/notifications/read",
        json={"id": 0, "category": "notice"},
        expect=200,
    )
    assert body.get("ok"), "ok 应为 true"
    print("    OK")


def test_11_notif_read_invalid_category():
    """POST /openclaw/notifications/read - {category: bad} → 400"""
    resp = cli.post(
        "/openclaw/notifications/read",
        json={"id": 0, "category": "__bad__"},
        expect=None, raw=True,
    )
    assert_status(resp, 400, label="notif-read-bad-category")
    print("    OK")


def test_12_notif_read_after_count_zero():
    """全部已读后 unread_count 应为 0"""
    cli.post("/openclaw/notifications/read", json={"id": 0}, expect=200)
    body = cli.get("/openclaw/notifications/count")
    assert body.get("unread_count", 0) == 0, (
        f"全部已读后 unread_count 应为 0, 实际 {body.get('unread_count')}"
    )
    print("    OK unread_count=0")


def test_13_notif_read_auth():
    """POST /openclaw/notifications/read - 认证三件套"""
    anon.post(
        "/openclaw/notifications/read", json={"id": 0},
        expect={401, 403}, raw=True,
    )
    bad_token.post(
        "/openclaw/notifications/read", json={"id": 0},
        expect={401, 403}, raw=True,
    )
    print("    OK")


# ─── /notifications/delete ───────────────────────────────────────────────

def test_14_notif_delete_invalid_category():
    """POST /openclaw/notifications/delete - 非法 category → 400"""
    resp = cli.post(
        "/openclaw/notifications/delete",
        json={"category": "__bad__"},
        expect=None, raw=True,
    )
    assert_status(resp, 400, label="notif-delete-bad-category")
    print("    OK")


def test_15_notif_delete_too_many_ids():
    """POST /openclaw/notifications/delete - ids 超 100 → 400"""
    huge_ids = list(range(1, 102))
    resp = cli.post(
        "/openclaw/notifications/delete",
        json={"ids": huge_ids},
        expect=None, raw=True,
    )
    assert_status(resp, 400, label="notif-delete-too-many")
    assert_error_message(resp, "100")
    print("    OK")


def test_16_notif_delete_nonexistent_id():
    """POST /openclaw/notifications/delete - 单条不存在 id（幂等）"""
    resp = cli.post(
        "/openclaw/notifications/delete",
        json={"id": 999_999_999},
        expect=None, raw=True,
    )
    if resp.status_code == 200:
        body = assert_json_keys(resp, "ok", "deleted")
        if body.get("deleted", 1) != 0:
            print(f"    [INFO] deleted={body.get('deleted')}")
        print(f"    OK deleted={body.get('deleted')}")
    elif resp.status_code in (400, 404):
        print(f"    OK status={resp.status_code}")
    else:
        raise AssertionError(f"非常规状态码: {resp.status_code}")


def test_17_notif_delete_auth():
    """POST /openclaw/notifications/delete - 认证三件套"""
    anon.post(
        "/openclaw/notifications/delete", json={"id": 1},
        expect={401, 403}, raw=True,
    )
    bad_token.post(
        "/openclaw/notifications/delete", json={"id": 1},
        expect={401, 403}, raw=True,
    )
    print("    OK")


# ─── 入口 ────────────────────────────────────────────────────────────────

def main():
    health_check()
    print()

    run_tests(
        globals(),
        title="test_instance_notifications.py",
        ordered=True,
    )


if __name__ == "__main__":
    main()
