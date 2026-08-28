#!/usr/bin/env python3
"""
集成测试：批量创建 /admin/batch-create

覆盖：
    POST /admin/batch-create   正常 / 部分失败 / 空列表 / 越界
    GET  /admin/departments    批量后接口可访问（联动验证）
"""
import os
import sys
import time

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

import helpers.api as _http  # noqa: E402
import helpers.client as _client  # noqa: E402
from helpers.api import (  # noqa: E402
    seed,
    health_check, run_tests,
    cleanup_users_by_prefix,
)

PREFIX = f"it-batch-{int(time.time())}"


def test_01_batch_normal():
    """批量创建 5 个用户"""
    items = [{"username": f"{PREFIX}-{i}", "password": "Aa12345!",
              "role": "user"} for i in range(5)]
    resp = seed.post("/admin/batch-create", json=items, expect=None, raw=True)
    data = resp.json()
    results = data.get("results") or data.get("Results") or []
    assert len(results) == 5, f"返回 results 数量 {len(results)} 期望 5"
    fails = [r for r in results if r.get("error")]
    assert not fails, f"部分失败: {fails}"
    print("    全部成功 (5/5)")


def test_02_batch_partial_fail():
    """批量创建：含 1 个重名应部分失败，其余成功"""
    # 先独立创建一个"重名锚点"用户
    anchor = f"{PREFIX}-dup-anchor"
    seed_resp = seed.post(
        "/admin/batch-create",
        json=[{"username": anchor, "password": "Aa12345!", "role": "user"}],
        expect=None, raw=True,
    )
    assert 200 <= seed_resp.status_code < 300, \
        f"准备重名锚点失败: status={seed_resp.status_code} resp={seed_resp.text[:200]}"
    seed_results = (seed_resp.json() or {}).get("results") \
        or (seed_resp.json() or {}).get("Results") or []
    assert seed_results and not seed_results[0].get("error"), \
        f"重名锚点未成功创建: {seed_results}"

    items = [
        {"username": anchor, "password": "Aa12345!", "role": "user"},      # 已存在
        {"username": f"{PREFIX}-new1", "password": "Aa12345!", "role": "user"},
        {"username": f"{PREFIX}-new2", "password": "Aa12345!", "role": "user"},
    ]
    resp = seed.post("/admin/batch-create", json=items, expect=None, raw=True)
    assert 200 <= resp.status_code < 500, \
        f"期望 2xx/4xx，实际 {resp.status_code}; resp={resp.text[:200]}"
    results = (resp.json() or {}).get("results") \
        or (resp.json() or {}).get("Results") or []
    assert len(results) == 3, f"results 数量应为 3，实际 {len(results)}"
    assert results[0].get("error"), f"首项重名应报错，实际 {results[0]}"
    assert not results[1].get("error") and not results[2].get("error"), \
        f"后续两项不应失败: {results}"


def test_03_batch_empty():
    """批量创建：空列表应 4xx"""
    resp = seed.post("/admin/batch-create", json=[], expect=None, raw=True)
    assert 400 <= resp.status_code < 500, \
        f"期望 4xx，实际 {resp.status_code}"


def test_04_batch_oversize():
    """批量创建：5001 条应 4xx（上限保护）"""
    items = [{"username": f"oversize-{i}", "password": "Aa12345!"}
             for i in range(5001)]
    # 临时静默帧打印避免 stdout 溢出
    print("     (oversize 用例：临时静默单次帧打印，body=5001 条)")
    _prev_quiet = _client.QUIET
    _client.QUIET = True
    try:
        resp = seed.post("/admin/batch-create", json=items, expect=None, raw=True)
    finally:
        _client.QUIET = _prev_quiet
    assert 400 <= resp.status_code < 500, \
        f"期望 4xx，实际 {resp.status_code}"


def test_05_departments_after_batch():
    """批量后查询 /admin/departments，接口应可用"""
    data = seed.get("/admin/departments")
    assert isinstance(data, dict), f"返回非对象: {data}"
    assert "departments" in data and "department_tree" in data, \
        f"departments/department_tree 字段缺失: keys={list(data.keys())}"
    print(f"    departments={len(data['departments'])}, tree={len(data['department_tree'])}")


def cleanup():
    """按 PREFIX 前缀双轮硬删"""
    try:
        cleanup_users_by_prefix(PREFIX)
        cleanup_users_by_prefix("oversize-")
    except Exception as e:
        print(f"[cleanup] 异常: {e}")


def main():
    health_check()
    try:
        run_tests(globals(), title="批量创建 /admin/batch-create", ordered=True)
    finally:
        cleanup()


if __name__ == "__main__":
    main()
