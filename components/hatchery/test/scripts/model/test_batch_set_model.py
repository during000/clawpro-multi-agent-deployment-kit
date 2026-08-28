#!/usr/bin/env python3
"""
POST /admin/instances/batch-set-model HandleAdminBatchSetModel

接口契约（来自 controller/admin_instances.go HandleAdminBatchSetModel）：
  - 请求：POST /admin/instances/batch-set-model
  - 鉴权：必须管理员（requireAdmin）
  - 限制：单次最多 20 个目标实例
  - 参数：ids (uint[]) 或 instance_ids (string[]) 至少提供一个，双方都提供时 ids 优先
  - 必填：ai_model_id (uint)
  - 可选：fallbacks (array)，每个元素与 primary 同一模型字段结构
  - 请求级校验失败 → 400
  - 全部目标不存在 → 200，results 数组返回 failed

测试场景（全部避免创建实实例/CVM，不触发 TAT）：
  S1   认证三件套（无认证 / 错误 token / 非管理员） → 401 / 403
  S2   非 POST 方法 → 405
  S3   请求体 JSON 格式错误 → 400
  S4   缺少 ids 与 instance_ids → 400
  S5   缺少 ai_model_id → 400
  S6   ids 超过 20 个 → 400
  S7   instance_ids 超过 20 个 → 400
  S8   fallback 与 primary 的 ai_model_id 重复 → 400
  S9   全部 ids 不存在 → 200，results 含 failed
  S10  全部 instance_ids 不存在 → 200，results 含 failed
"""

import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))
from helpers.api import (
    health_check, make_api_fn,
    auth_test_suite, run_tests,
    assert_status,
    seed,
)


# ─────────────────────────────────────────────
# API 调用器
# ─────────────────────────────────────────────

post_batch_set_model = make_api_fn("post", "/admin/instances/batch-set-model")


# ─────────────────────────────────────────────
# 测试用例
# ─────────────────────────────────────────────

def test_batch_set_model_auth():
    """S1：认证三件套（无认证 / 错误 token / 非管理员） → 401 / 403"""
    auth_test_suite(
        lambda headers: post_batch_set_model(
            {"ids": [1], "ai_model_id": 1}, headers=headers,
        ),
        label="batch_set_model",
    )


def test_batch_set_model_method_not_allowed():
    """S2：非 POST 方法 → 405"""
    print(">>> [批量设置模型] 场景2：GET 方法 → 405 ...")
    resp = seed.get("/admin/instances/batch-set-model", expect=None, raw=True)
    assert_status(resp, 405, label="GET batch-set-model")
    print(f"    OK (status=405)")


def test_batch_set_model_invalid_json():
    """S3：请求体 JSON 格式错误 → 400"""
    print(">>> [批量设置模型] 场景3：无效 JSON 请求体 → 400 ...")
    resp = post_batch_set_model(raw_data="not json")
    assert_status(resp, 400, label="无效JSON")
    print(f"    OK (status=400)")


def test_batch_set_model_missing_selectors():
    """S4：既没有 ids 也没有 instance_ids → 400"""
    print(">>> [批量设置模型] 场景4：缺少 ids 与 instance_ids → 400 ...")
    resp = post_batch_set_model({"ai_model_id": 1})
    assert_status(resp, 400, label="缺少选择器")
    data = resp.json()
    print(f"    OK (status=400, error={data.get('error', '')[:60]})")


def test_batch_set_model_missing_ai_model_id():
    """S5：有 ids 但缺少 ai_model_id → 400"""
    print(">>> [批量设置模型] 场景5：缺少 ai_model_id → 400 ...")
    resp = post_batch_set_model({"ids": [1, 2]})
    assert_status(resp, 400, label="缺少ai_model_id")
    data = resp.json()
    print(f"    OK (status=400, error={data.get('error', '')[:60]})")


def test_batch_set_model_oversize_ids():
    """S6：ids 超过 20 个 → 400（覆盖 ids 参数）"""
    print(">>> [批量设置模型] 场景6：ids 超过 20 → 400 ...")
    resp = post_batch_set_model({
        "ids": list(range(1, 22)),
        "ai_model_id": 1,
    })
    assert_status(resp, 400, label="ids超上限")
    data = resp.json()
    print(f"    OK (status=400, error={data.get('error', '')[:80]})")


def test_batch_set_model_oversize_instance_ids():
    """S7：instance_ids 超过 20 个 → 400（覆盖 instance_ids 参数）"""
    print(">>> [批量设置模型] 场景7：instance_ids 超过 20 → 400 ...")
    resp = post_batch_set_model({
        "instance_ids": [f"ins-{i:03d}" for i in range(1, 22)],
        "ai_model_id": 1,
    })
    assert_status(resp, 400, label="instance_ids超上限")
    data = resp.json()
    print(f"    OK (status=400, error={data.get('error', '')[:80]})")


def test_batch_set_model_duplicate_fallback():
    """S8：fallback 与 primary 的 ai_model_id 重复 → 400（覆盖 fallbacks 参数）"""
    print(">>> [批量设置模型] 场景8：fallback 与 primary 重复 → 400 ...")
    resp = post_batch_set_model({
        "ids": [1],
        "ai_model_id": 42,
        "fallbacks": [
            {"ai_model_id": 42},
        ],
    })
    assert_status(resp, 400, label="重复fallback")
    data = resp.json()
    print(f"    OK (status=400, error={data.get('error', '')[:80]})")


def test_batch_set_model_all_nonexistent_ids():
    """S9：全部 ids 不存在 → 200，results 数组含 failed（覆盖 ids 参数 + 200 响应形态）"""
    print(">>> [批量设置模型] 场景9：全部 ids 不存在 → 200 ...")
    resp = post_batch_set_model({
        "ids": [9999991, 9999992],
        "ai_model_id": 1,
    })
    assert_status(resp, 200, label="全部不存在ids")
    data = resp.json()
    assert data.get("ok") is True, f"ok 应为 true: {data}"
    results = data.get("results", [])
    assert isinstance(results, list), f"results 应为 list: {data}"
    assert len(results) == 2, f"results 长度=2: {len(results)}"
    for r in results:
        assert r.get("status") == "failed", f"不存在的目标应 failed: {r}"
        assert "message" in r, f"failed 结果应有 message: {r}"
    print(f"    OK (status=200, results={len(results)} failed)")


def test_batch_set_model_all_nonexistent_instance_ids():
    """S10：全部 instance_ids 不存在 → 200，results 数组含 failed（覆盖 instance_ids 参数 + 200 响应形态）"""
    print(">>> [批量设置模型] 场景10：全部 instance_ids 不存在 → 200 ...")
    resp = post_batch_set_model({
        "instance_ids": ["ins-nonexistent-1", "ins-nonexistent-2"],
        "ai_model_id": 1,
    })
    assert_status(resp, 200, label="全部不存在instance_ids")
    data = resp.json()
    assert data.get("ok") is True, f"ok 应为 true: {data}"
    results = data.get("results", [])
    assert isinstance(results, list), f"results 应为 list: {data}"
    assert len(results) == 2, f"results 长度=2: {len(results)}"
    for r in results:
        assert r.get("status") == "failed", f"不存在的目标应 failed: {r}"
        assert "message" in r, f"failed 结果应有 message: {r}"
    print(f"    OK (status=200, results={len(results)} failed)")


# ─────────────────────────────────────────────
# Main
# ─────────────────────────────────────────────

def main():
    health_check()
    print()
    run_tests(globals(), title="POST /admin/instances/batch-set-model")


if __name__ == "__main__":
    main()
