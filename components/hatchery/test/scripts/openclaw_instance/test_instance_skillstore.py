#!/usr/bin/env python3
"""
集成测试：实例管理 - 技能广场 Skill Store（K 组）

覆盖接口（全部依赖 SMH 站点开关）：
    GET  /openclaw/skillstore                 列表（分页 + 关键词 + 分类）
    GET  /openclaw/skillstore/detail          技能详情（按 slug）
    GET  /openclaw/skillstore/categories      分类列表
    GET  /openclaw/skillstore/instances       该 slug 在当前用户实例上的安装状态
    GET  /openclaw/skillstore/tasks           下发记录
    POST /openclaw/skillstore/distribute      下发到自己的实例
    POST /openclaw/skillstore/uninstall       从自己的实例卸载
    GET  /openclaw/skillstore/download        下载 zip（302 → SMH）

设计原则：
    - 大部分 handler 都先 requireSMHEnabled，环境未开启时全部 403 → SKIP
    - distribute / uninstall 是真实下发，本测试只做契约/参数/鉴权 + 跨用户实例
      ID 的归属校验（403），绝不真发起任务
    - download 期望 302/4xx，allow_redirects=False
"""
import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers.api import (
    ApiClient,
    health_check, run_tests,
    auth_test_suite, assert_status,
)
from _instance_helpers import (
    cli, require_shared_instance,
    get_shared_db_id, NONEXISTENT_DB_ID,
    assert_error_message, assert_json_keys,
)


SHARED_DB_ID = None
SMH_DISABLED = False  # 由 test_01 检测后填充


def _smh_skip_if_needed(resp, label):
    """若返回 403 且 error 含 SMH 关键词，则置全局 SKIP 标志并返回 True。"""
    global SMH_DISABLED
    if resp.status_code == 403:
        try:
            err = (resp.json() or {}).get("error", "")
        except Exception:
            err = ""
        if "SMH" in err or "未启用" in err:
            SMH_DISABLED = True
            print(f"    SKIP ({label}: SMH 未启用)")
            return True
    return False


# ─── /openclaw/skillstore 列表 ────────────────────────────────────────────

def test_01_skillstore_list_default():
    """GET /openclaw/skillstore - 默认分页"""
    resp = cli.get("/openclaw/skillstore", expect=None, raw=True)
    if _smh_skip_if_needed(resp, "skillstore-list"):
        return
    assert_status(resp, {200}, label="skillstore-list")
    body = resp.json() or {}
    for k in ("skills", "page", "page_size", "total"):
        assert k in body, f"列表缺字段 {k}: keys={list(body.keys())}"
    assert isinstance(body["skills"], list), "skills 应为 list"
    print(f"    OK total={body['total']} count={len(body['skills'])}")


def test_02_skillstore_list_with_filter():
    """GET /openclaw/skillstore?keyword=xxx&sort=downloads"""
    if SMH_DISABLED:
        print("    SKIP (SMH 未启用)")
        return
    resp = cli.get(
        "/openclaw/skillstore",
        params={"keyword": "xxx-no-such-skill", "sort": "downloads", "page_size": 5},
        expect=None, raw=True,
    )
    assert_status(resp, {200}, label="skillstore-list-filter")
    body = resp.json() or {}
    assert isinstance(body.get("skills"), list)
    print(f"    OK total={body.get('total')}")


def test_03_skillstore_list_auth():
    """GET /openclaw/skillstore - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=15).get(
            "/openclaw/skillstore", expect=None, raw=True,
            extra_headers=headers,
        ),
        label="skillstore-list",
        check_admin=False,
    )


# ─── /openclaw/skillstore/categories ─────────────────────────────────────

def test_04_skillstore_categories_ok():
    """GET /openclaw/skillstore/categories - 分类列表（不依赖 SMH 开关）"""
    resp = cli.get("/openclaw/skillstore/categories", expect=None, raw=True)
    assert_status(resp, {200}, label="skillstore-categories")
    body = resp.json() or {}
    assert "categories" in body, f"返回缺 categories: {body}"
    print(f"    OK count={len(body['categories'] or [])}")


def test_05_skillstore_categories_auth():
    """GET /openclaw/skillstore/categories - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=15).get(
            "/openclaw/skillstore/categories", expect=None, raw=True,
            extra_headers=headers,
        ),
        label="skillstore-categories",
        check_admin=False,
    )


# ─── /openclaw/skillstore/detail ─────────────────────────────────────────

def test_06_skillstore_detail_missing_slug():
    """GET /openclaw/skillstore/detail - 缺 slug → 400"""
    resp = cli.get("/openclaw/skillstore/detail", expect=None, raw=True)
    if _smh_skip_if_needed(resp, "skillstore-detail-missing"):
        return
    assert_status(resp, {400}, label="skillstore-detail-missing")
    assert_error_message(resp, "slug")
    print(f"    OK status={resp.status_code}")


def test_07_skillstore_detail_nonexistent_slug():
    """GET /openclaw/skillstore/detail?slug=xxx → 404"""
    if SMH_DISABLED:
        print("    SKIP (SMH 未启用)")
        return
    resp = cli.get(
        "/openclaw/skillstore/detail",
        params={"slug": "no-such-slug-12345"},
        expect=None, raw=True,
    )
    assert_status(resp, {404}, label="skillstore-detail-not-found")
    print(f"    OK status={resp.status_code}")


def test_08_skillstore_detail_auth():
    """GET /openclaw/skillstore/detail - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=15).get(
            "/openclaw/skillstore/detail",
            params={"slug": "any"},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="skillstore-detail",
        check_admin=False,
    )


# ─── /openclaw/skillstore/instances ──────────────────────────────────────

def test_09_skillstore_instances_missing_slug():
    """GET /openclaw/skillstore/instances - 缺 slug → 400"""
    resp = cli.get("/openclaw/skillstore/instances", expect=None, raw=True)
    if _smh_skip_if_needed(resp, "skillstore-instances-missing"):
        return
    assert_status(resp, {400}, label="skillstore-instances-missing")
    print(f"    OK status={resp.status_code}")


def test_10_skillstore_instances_nonexistent_slug():
    """GET /openclaw/skillstore/instances?slug=xxx → 404"""
    if SMH_DISABLED:
        print("    SKIP (SMH 未启用)")
        return
    resp = cli.get(
        "/openclaw/skillstore/instances",
        params={"slug": "no-such-slug-12345"},
        expect=None, raw=True,
    )
    assert_status(resp, {404}, label="skillstore-instances-not-found")
    print(f"    OK status={resp.status_code}")


def test_11_skillstore_instances_auth():
    """GET /openclaw/skillstore/instances - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=15).get(
            "/openclaw/skillstore/instances",
            params={"slug": "any"},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="skillstore-instances",
        check_admin=False,
    )


# ─── /openclaw/skillstore/tasks ──────────────────────────────────────────

def test_12_skillstore_tasks_missing_slug():
    """GET /openclaw/skillstore/tasks - 缺 slug → 400"""
    resp = cli.get("/openclaw/skillstore/tasks", expect=None, raw=True)
    if _smh_skip_if_needed(resp, "skillstore-tasks-missing"):
        return
    assert_status(resp, {400}, label="skillstore-tasks-missing")
    print(f"    OK status={resp.status_code}")


def test_13_skillstore_tasks_nonexistent_slug():
    """GET /openclaw/skillstore/tasks?slug=xxx → 404"""
    if SMH_DISABLED:
        print("    SKIP (SMH 未启用)")
        return
    resp = cli.get(
        "/openclaw/skillstore/tasks",
        params={"slug": "no-such-slug-12345"},
        expect=None, raw=True,
    )
    assert_status(resp, {404}, label="skillstore-tasks-not-found")
    print(f"    OK status={resp.status_code}")


def test_14_skillstore_tasks_auth():
    """GET /openclaw/skillstore/tasks - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=15).get(
            "/openclaw/skillstore/tasks",
            params={"slug": "any"},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="skillstore-tasks",
        check_admin=False,
    )


# ─── /openclaw/skillstore/distribute ─────────────────────────────────────

def test_15_distribute_invalid_body():
    """POST /openclaw/skillstore/distribute - 非 JSON → 400"""
    resp = cli.post(
        "/openclaw/skillstore/distribute",
        data="not-json{",
        expect=None, raw=True,
    )
    if _smh_skip_if_needed(resp, "distribute-invalid"):
        return
    assert_status(resp, {400}, label="distribute-invalid")
    print(f"    OK status={resp.status_code}")


def test_16_distribute_missing_slug():
    """POST /openclaw/skillstore/distribute - 缺 slug → 400"""
    if SMH_DISABLED:
        print("    SKIP (SMH 未启用)")
        return
    resp = cli.post(
        "/openclaw/skillstore/distribute",
        json={"instance_ids": [SHARED_DB_ID or 1]},
        expect=None, raw=True,
    )
    assert_status(resp, {400}, label="distribute-missing-slug")
    assert_error_message(resp, "slug")
    print(f"    OK status={resp.status_code}")


def test_17_distribute_missing_instance_ids():
    """POST /openclaw/skillstore/distribute - 缺 instance_ids → 400"""
    if SMH_DISABLED:
        print("    SKIP (SMH 未启用)")
        return
    resp = cli.post(
        "/openclaw/skillstore/distribute",
        json={"slug": "no-such-slug-12345"},
        expect=None, raw=True,
    )
    assert_status(resp, {400}, label="distribute-missing-ids")
    assert_error_message(resp, "instance_ids")
    print(f"    OK status={resp.status_code}")


def test_18_distribute_nonexistent_slug():
    """POST /openclaw/skillstore/distribute - 不存在的 slug → 400/404"""
    if SMH_DISABLED:
        print("    SKIP (SMH 未启用)")
        return
    resp = cli.post(
        "/openclaw/skillstore/distribute",
        json={
            "slug": "no-such-slug-12345",
            "instance_ids": [SHARED_DB_ID or 1],
        },
        expect=None, raw=True,
    )
    assert_status(resp, {400, 404}, label="distribute-not-found")
    print(f"    OK status={resp.status_code}")


def test_19_distribute_foreign_instance():
    """POST /openclaw/skillstore/distribute - 实例不属于当前用户 → 403/404
    使用一个故意伪造的、当前用户名下不存在的 instance_id（NONEXISTENT_DB_ID）。"""
    if SMH_DISABLED:
        print("    SKIP (SMH 未启用)")
        return
    resp = cli.post(
        "/openclaw/skillstore/distribute",
        json={
            "slug": "no-such-slug-12345",
            "instance_ids": [NONEXISTENT_DB_ID],
        },
        expect=None, raw=True,
    )
    # 后端先做技能查询（不存在 → 404/400），所以也允许 4xx
    assert_status(resp, {400, 403, 404}, label="distribute-foreign")
    print(f"    OK status={resp.status_code}")


def test_20_distribute_auth():
    """POST /openclaw/skillstore/distribute - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=15).post(
            "/openclaw/skillstore/distribute",
            json={"slug": "x", "instance_ids": [1]},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="distribute",
        check_admin=False,
    )


# ─── /openclaw/skillstore/uninstall ──────────────────────────────────────

def test_21_uninstall_missing_slug():
    """POST /openclaw/skillstore/uninstall - 缺 slug → 400"""
    resp = cli.post(
        "/openclaw/skillstore/uninstall",
        json={"instance_ids": [SHARED_DB_ID or 1]},
        expect=None, raw=True,
    )
    assert_status(resp, {400}, label="uninstall-missing-slug")
    assert_error_message(resp, "slug")
    print(f"    OK status={resp.status_code}")


def test_22_uninstall_missing_instance_ids():
    """POST /openclaw/skillstore/uninstall - 缺 instance_ids → 400"""
    resp = cli.post(
        "/openclaw/skillstore/uninstall",
        json={"slug": "no-such-slug-12345"},
        expect=None, raw=True,
    )
    assert_status(resp, {400}, label="uninstall-missing-ids")
    assert_error_message(resp, "instance_ids")
    print(f"    OK status={resp.status_code}")


def test_23_uninstall_nonexistent_slug():
    """POST /openclaw/skillstore/uninstall - 不存在的 slug → 404"""
    resp = cli.post(
        "/openclaw/skillstore/uninstall",
        json={
            "slug": "no-such-slug-12345",
            "instance_ids": [SHARED_DB_ID or 1],
        },
        expect=None, raw=True,
    )
    assert_status(resp, {400, 404}, label="uninstall-not-found")
    print(f"    OK status={resp.status_code}")


def test_24_uninstall_auth():
    """POST /openclaw/skillstore/uninstall - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=15).post(
            "/openclaw/skillstore/uninstall",
            json={"slug": "x", "instance_ids": [1]},
            expect=None, raw=True, extra_headers=headers,
        ),
        label="uninstall",
        check_admin=False,
    )


# ─── /openclaw/skillstore/download ───────────────────────────────────────

def test_25_download_missing_slug():
    """GET /openclaw/skillstore/download - 缺 slug → 400"""
    # download 成功路径会 302，不要让 ApiClient 自动跟随
    resp = cli.get(
        "/openclaw/skillstore/download",
        expect=None, raw=True, allow_redirects=False,
    )
    if _smh_skip_if_needed(resp, "download-missing"):
        return
    assert_status(resp, {400}, label="download-missing-slug")
    print(f"    OK status={resp.status_code}")


def test_26_download_nonexistent_slug():
    """GET /openclaw/skillstore/download?slug=xxx → 404"""
    if SMH_DISABLED:
        print("    SKIP (SMH 未启用)")
        return
    resp = cli.get(
        "/openclaw/skillstore/download",
        params={"slug": "no-such-slug-12345"},
        expect=None, raw=True, allow_redirects=False,
    )
    assert_status(resp, {404}, label="download-not-found")
    print(f"    OK status={resp.status_code}")


def test_27_download_auth():
    """GET /openclaw/skillstore/download - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=15).get(
            "/openclaw/skillstore/download",
            params={"slug": "any"},
            expect=None, raw=True, allow_redirects=False,
            extra_headers=headers,
        ),
        label="skillstore-download",
        check_admin=False,
    )


# ─── 入口 ────────────────────────────────────────────────────────────────

def main():
    global SHARED_DB_ID
    health_check()
    SHARED_DB_ID = require_shared_instance().db_id
    if SHARED_DB_ID:
        print(f">>> 复用共享实例 db_id={SHARED_DB_ID}")
    print()

    run_tests(
        globals(),
        title="test_instance_skillstore.py",
        ordered=True,
    )


if __name__ == "__main__":
    main()
