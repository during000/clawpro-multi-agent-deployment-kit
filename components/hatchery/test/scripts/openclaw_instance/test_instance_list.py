#!/usr/bin/env python3
"""
集成测试：实例管理 - GET /openclaw/list（B 组）

覆盖：
    - happy path（默认分页）+ 字段契约
    - page / page_size 边界（page=0、page_size=1/100/超大值/负数）
    - id 精确查询（共享实例 + 不存在 id）
    - instance_id 精确查询（共享实例对应的 cvm-id + 不存在 cvm-id）
    - 认证三件套
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
    NONEXISTENT_DB_ID,
    get_shared_db_id,
    find_instance_by_db_id,
    assert_json_keys,
)


SHARED_DB_ID = None
SHARED_INSTANCE_ID = None  # cvm 实例 ID（ins-xxx）


# ─── happy path ──────────────────────────────────────────────────────────

def test_01_list_default():
    """GET /openclaw/list - 默认分页"""
    resp = cli.get("/openclaw/list", raw=True)
    body = assert_json_keys(
        resp, "instances", "total", "page", "page_size", "total_pages",
    )
    assert isinstance(body["instances"], list), (
        f"instances 应为数组: {type(body['instances']).__name__}"
    )
    assert body["page"] == 1, f"默认 page 应为 1, 实际 {body['page']}"
    assert body["page_size"] == 30, (
        f"默认 page_size 应为 30, 实际 {body['page_size']}"
    )
    print(
        f"    OK total={body['total']} page_size={body['page_size']} "
        f"count={len(body['instances'])}"
    )


def test_02_list_field_contract():
    """GET /openclaw/list - 实例对象字段契约"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    inst = find_instance_by_db_id(SHARED_DB_ID)
    assert inst, f"未在 list 中找到共享实例 db_id={SHARED_DB_ID}"
    pk = inst.get("id") or inst.get("ID")
    iid = inst.get("instance_id") or inst.get("InstanceId")
    name = inst.get("name") or inst.get("Name")
    assert pk, f"实例缺 id/ID 字段: keys={list(inst.keys())}"
    assert name, f"实例缺 name/Name 字段: keys={list(inst.keys())}"
    print(f"    OK db_id={pk} instance_id={iid} name={name}")


# ─── 分页参数边界 ──────────────────────────────────────────────────────────

def test_03_list_page_size_1():
    """GET /openclaw/list - page_size=1"""
    resp = cli.get(
        "/openclaw/list",
        params={"page": 1, "page_size": 1},
        raw=True,
    )
    body = assert_json_keys(resp, "instances", "page_size")
    assert body["page_size"] == 1, f"page_size 应为 1, 实际 {body['page_size']}"
    assert len(body["instances"]) <= 1, (
        f"page_size=1 时 instances 应 ≤ 1, 实际 {len(body['instances'])}"
    )
    print(f"    OK count={len(body['instances'])} total={body.get('total')}")


def test_04_list_page_size_100_max():
    """GET /openclaw/list - page_size=100（最大值）"""
    resp = cli.get(
        "/openclaw/list",
        params={"page": 1, "page_size": 100},
        raw=True,
    )
    body = assert_json_keys(resp, "page_size")
    assert body["page_size"] in (100, 30), (
        f"page_size 应被接受为 100 或回退默认: 实际 {body['page_size']}"
    )
    print(f"    OK page_size={body['page_size']}")


def test_05_list_page_size_huge():
    """GET /openclaw/list - page_size=99999（应被裁剪到上限）"""
    resp = cli.get(
        "/openclaw/list",
        params={"page": 1, "page_size": 99999},
        expect=None, raw=True,
    )
    if resp.status_code == 200:
        body = resp.json() or {}
        assert body.get("page_size", 0) <= 100, (
            f"page_size 未被裁剪到 100: {body.get('page_size')}"
        )
        print(f"    OK 裁剪到 page_size={body.get('page_size')}")
    elif resp.status_code == 400:
        print("    OK 超大值返回 400")
    else:
        raise AssertionError(
            f"超大 page_size 期望 200/400, 实际 {resp.status_code}"
        )


def test_06_list_page_size_negative():
    """GET /openclaw/list - page_size=-1（异常值）"""
    resp = cli.get(
        "/openclaw/list",
        params={"page": 1, "page_size": -1},
        expect=None, raw=True,
    )
    assert resp.status_code < 500, (
        f"负数 page_size 不应触发 5xx: {resp.status_code}"
    )
    print(f"    OK status={resp.status_code}")


def test_07_list_page_zero():
    """GET /openclaw/list - page=0（异常值）"""
    resp = cli.get(
        "/openclaw/list",
        params={"page": 0, "page_size": 30},
        expect=None, raw=True,
    )
    assert resp.status_code < 500, (
        f"page=0 不应触发 5xx: {resp.status_code}"
    )
    print(f"    OK status={resp.status_code}")


def test_08_list_page_overflow():
    """GET /openclaw/list - page 远超 total_pages 应返回空数组"""
    resp = cli.get(
        "/openclaw/list",
        params={"page": 99999, "page_size": 30},
        raw=True,
    )
    body = assert_json_keys(resp, "instances")
    assert not body["instances"], (
        f"超大 page 应返回空数组: {len(body['instances'])}"
    )
    print(f"    OK empty page count={len(body['instances'])}")


# ─── 精确查询 ──────────────────────────────────────────────────────────────

def test_09_list_filter_by_id():
    """GET /openclaw/list?id=<shared>"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.get("/openclaw/list", params={"id": SHARED_DB_ID}, raw=True)
    body = assert_json_keys(resp, "instances")
    assert len(body["instances"]) == 1, (
        f"按 id 精确查询应返回 1 条, 实际 {len(body['instances'])}"
    )
    ret_id = body["instances"][0].get("id") or body["instances"][0].get("ID")
    assert ret_id == SHARED_DB_ID, (
        f"返回 id 不一致: 期望 {SHARED_DB_ID}, 实际 {ret_id}"
    )
    print(f"    OK 匹配到 db_id={ret_id}")


def test_10_list_filter_by_nonexistent_id():
    """GET /openclaw/list?id=NONEXISTENT 应返回空数组"""
    resp = cli.get(
        "/openclaw/list",
        params={"id": NONEXISTENT_DB_ID},
        raw=True,
    )
    body = assert_json_keys(resp, "instances")
    assert not body["instances"], (
        f"不存在的 id 应返回空数组: {len(body['instances'])}"
    )
    print("    OK empty")


def test_11_list_filter_by_instance_id():
    """GET /openclaw/list?instance_id=<shared>"""
    if not SHARED_INSTANCE_ID:
        print("    SKIP (无共享 cvm instance_id)")
        return
    resp = cli.get(
        "/openclaw/list",
        params={"instance_id": SHARED_INSTANCE_ID},
        raw=True,
    )
    body = assert_json_keys(resp, "instances")
    assert len(body["instances"]) == 1, (
        f"按 instance_id 精确查询应返回 1 条, 实际 {len(body['instances'])}"
    )
    ret_iid = (
        body["instances"][0].get("instance_id")
        or body["instances"][0].get("InstanceId")
    )
    assert ret_iid == SHARED_INSTANCE_ID, (
        f"返回 instance_id 不一致: 期望 {SHARED_INSTANCE_ID}, 实际 {ret_iid}"
    )
    print(f"    OK 匹配到 instance_id={ret_iid}")


def test_12_list_filter_by_nonexistent_instance_id():
    """GET /openclaw/list?instance_id=ins-xxx_not_exist 应返回空"""
    resp = cli.get(
        "/openclaw/list",
        params={"instance_id": "ins-xxxxxxxxxxxxx_not_exist"},
        raw=True,
    )
    body = assert_json_keys(resp, "instances")
    assert not body["instances"], (
        f"不存在的 instance_id 应返回空: {len(body['instances'])}"
    )
    print("    OK empty")


def test_13_list_id_priority_over_instance_id():
    """API.md：id 与 instance_id 同时传时以 id 为准"""
    if not SHARED_DB_ID:
        print("    SKIP (无共享实例)")
        return
    resp = cli.get(
        "/openclaw/list",
        params={"id": SHARED_DB_ID, "instance_id": "ins-fake-not-exist"},
        raw=True,
    )
    body = assert_json_keys(resp, "instances")
    assert len(body["instances"]) == 1, (
        f"id 优先时应返回 1 条, 实际 {len(body['instances'])}"
    )
    ret_id = body["instances"][0].get("id") or body["instances"][0].get("ID")
    assert ret_id == SHARED_DB_ID, f"id 优先未生效: 返回 {ret_id}"
    print(f"    OK id 优先生效 → db_id={ret_id}")


# ─── 认证 ─────────────────────────────────────────────────────────────────

def test_14_list_auth():
    """GET /openclaw/list - 认证三件套"""
    auth_test_suite(
        lambda headers: ApiClient("", timeout=30).get(
            "/openclaw/list",
            expect=None, raw=True, extra_headers=headers,
        ),
        label="list",
        check_admin=False,
    )


# ─── 入口 ────────────────────────────────────────────────────────────────

def main():
    global SHARED_DB_ID, SHARED_INSTANCE_ID
    health_check()
    SHARED_DB_ID = require_shared_instance().db_id
    if SHARED_DB_ID:
        inst = find_instance_by_db_id(SHARED_DB_ID)
        if inst:
            SHARED_INSTANCE_ID = (
                inst.get("instance_id") or inst.get("InstanceId")
            )
        print(
            f">>> 复用共享实例 db_id={SHARED_DB_ID} "
            f"instance_id={SHARED_INSTANCE_ID}"
        )
    else:
        print(">>> 未找到共享实例，按 id/instance_id 的 happy path 用例将跳过")
    print()

    run_tests(
        globals(),
        title="test_instance_list.py",
        ordered=True,
    )


if __name__ == "__main__":
    main()
