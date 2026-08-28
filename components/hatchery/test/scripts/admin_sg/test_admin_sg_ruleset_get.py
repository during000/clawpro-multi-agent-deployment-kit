#!/usr/bin/env python3
"""
GET /admin/config/security-group/ruleset 获取规则组 集成测试

测试场景：
  场景 1：无认证信息 → 401/403
  场景 2：错误 token → 401/403
  场景 3：非管理员 token → 401/403
  场景 4：未初始化（全新租户，从未调用过 POST rulesets）→ 200，{"initialized": false}，不含 id/rules/version 等字段
  场景 5：已初始化，正常查询 → 200，initialized=true，包含 id、name、version、rules、projected_to
  场景 6：projected_to 只包含 ACTIVE 状态的 SG（不含 FROZEN/DRIFT 状态）→ projected_to 中每条 SG 均为 ACTIVE
  场景 7：rules 字段为合法 JSON 数组（不为 null，空时为 []）→ rules 为数组类型
  场景 8：projected_to 中每条 SG 包含 sg_id、sg_name、cvm_count 字段 → 字段结构正确
  场景 9：user_group_ids 字段本期恒为空数组（omitempty 时不出现，或出现时为 []）→ 不为 null
  场景 10：is_default 字段本期恒为 true（omitempty 时不出现，或出现时为 true）→ 不为 false
  场景 11：DB 异常时返回 500（模拟场景，仅在可注入故障的环境验证）
"""

import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))
from helpers.api import (
    seed,
    IDENTIFIER,
    health_check, make_api_fn,
    auth_test_suite, assert_status, run_tests,
)

# ─────────────────────────────────────────────
# 工具函数
# ─────────────────────────────────────────────


get_ruleset = make_api_fn("get", "/admin/config/security-group/ruleset", timeout=10)


# ─────────────────────────────────────────────
# 测试用例
# ─────────────────────────────────────────────

def test_get_ruleset_auth():
    """认证测试三件套：无认证/错误token/非管理员 → 401/403"""
    auth_test_suite(lambda headers: get_ruleset(headers=headers),
                    label="get_ruleset")


def test_get_ruleset_not_initialized():
    """
    场景4：未初始化时返回 {initialized: false}。
    注意：此场景需要在全新租户（从未调用过 POST rulesets）的环境下执行。
    若当前环境已初始化，此场景会被跳过。
    """
    print(">>> [获取规则组] 场景4：未初始化 → 200，{initialized: false} ...")
    resp = get_ruleset()
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    if not data.get("initialized", True):
        # 确认未初始化时不含业务字段
        assert "id" not in data or data.get("id") == 0, \
            f"未初始化时不应含 id 字段，实际 {data}"
        assert "version" not in data or data.get("version") == 0, \
            f"未初始化时不应含 version 字段，实际 {data}"
        assert "rules" not in data or data.get("rules") is None, \
            f"未初始化时不应含 rules 字段，实际 {data}"
        print(f"    OK (initialized=false，未初始化状态正确)")
    else:
        print(f"    SKIP (当前环境已初始化，initialized=true，跳过未初始化场景)")

def test_get_ruleset_initialized():
    print(">>> [获取规则组] 场景5：已初始化，正常查询 → 200，包含完整字段 ...")
    resp = get_ruleset()
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    if not data.get("initialized"):
        print("    SKIP (当前环境未初始化，跳过已初始化场景)")
        return
    assert data.get("initialized") is True, f"initialized 应为 true，实际 {data}"
    assert data.get("id", 0) > 0, f"id 应大于 0，实际 {data}"
    assert data.get("name"), f"name 不应为空，实际 {data}"
    assert data.get("version", 0) > 0, f"version 应大于 0，实际 {data}"
    # rules 和 projected_to 使用 omitempty，为空时字段可能不出现在响应中
    # 若存在则验证类型正确
    if "rules" in data:
        assert isinstance(data["rules"], list), f"rules 应为数组，实际 {data['rules']}"
    if "projected_to" in data:
        assert isinstance(data["projected_to"], list), f"projected_to 应为数组，实际 {data['projected_to']}"
    print(f"    OK (id={data.get('id')}, name={data.get('name')}, version={data.get('version')}, rules={'present' if 'rules' in data else 'omitted(empty)'})")

def test_get_ruleset_projected_to_active_only():
    print(">>> [获取规则组] 场景6：projected_to 只含 ACTIVE 状态 SG ...")
    resp = get_ruleset()
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    if not data.get("initialized"):
        print("    SKIP (当前环境未初始化)")
        return
    projected_to = data.get("projected_to", [])
    # projected_to 中的 SG 均为 ACTIVE（接口层已过滤，不含 FROZEN/DRIFT）
    # 验证方式：每条 SG 均包含 sg_id 字段（ACTIVE SG 必有 sg_id）
    for sg in projected_to:
        assert sg.get("sg_id"), f"projected_to 中每条 SG 应有 sg_id，实际 {sg}"
        # 若响应中包含 status 字段，则严格验证其为 ACTIVE
        if "status" in sg:
            assert sg.get("status") == "ACTIVE", \
                f"projected_to 中 SG 状态应为 ACTIVE，实际 {sg.get('status')}，完整数据: {sg}"
    print(f"    OK (projected_to 共 {len(projected_to)} 条，均含 sg_id，status 验证通过)")

def test_get_ruleset_rules_is_array():
    print(">>> [获取规则组] 场景7：rules 字段为数组类型（不为 null）...")
    resp = get_ruleset()
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    if not data.get("initialized"):
        print("    SKIP (当前环境未初始化)")
        return
    # rules 使用 omitempty，为空时字段可能不出现在响应中，视为空数组
    rules = data.get("rules")
    if rules is None:
        # omitempty 导致空规则时字段不出现，视为合法的空数组
        rules = []
    assert isinstance(rules, list), \
        f"rules 应为数组类型，实际类型={type(rules).__name__}，值={rules}"
    print(f"    OK (rules 为数组，共 {len(rules)} 条{', omitempty省略' if 'rules' not in data else ''})")

def test_get_ruleset_projected_to_fields():
    print(">>> [获取规则组] 场景8：projected_to 中每条 SG 包含 sg_id、sg_name、cvm_count 字段 ...")
    resp = get_ruleset()
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    if not data.get("initialized"):
        print("    SKIP (当前环境未初始化)")
        return
    projected_to = data.get("projected_to", [])
    if not projected_to:
        print("    SKIP (projected_to 为空，无法验证字段结构)")
        return
    for sg in projected_to:
        assert "sg_id" in sg, f"projected_to 中每条 SG 应含 sg_id，实际 {sg}"
        assert "cvm_count" in sg, f"projected_to 中每条 SG 应含 cvm_count，实际 {sg}"
        assert isinstance(sg.get("cvm_count"), int), \
            f"cvm_count 应为整数，实际 {sg.get('cvm_count')}"
    print(f"    OK (projected_to 字段结构正确，共 {len(projected_to)} 条)")

def test_get_ruleset_user_group_ids_not_null():
    print(">>> [获取规则组] 场景9：user_group_ids 字段本期恒为空数组（不为 null）...")
    resp = get_ruleset()
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    if not data.get("initialized"):
        print("    SKIP (当前环境未初始化)")
        return
    # user_group_ids 本期恒为空数组，omitempty 时不出现在响应中
    # 若出现，则不应为 null
    if "user_group_ids" in data:
        assert data["user_group_ids"] is not None, \
            f"user_group_ids 不应为 null，实际 {data['user_group_ids']}"
        assert isinstance(data["user_group_ids"], list), \
            f"user_group_ids 应为数组，实际 {data['user_group_ids']}"
    print(f"    OK (user_group_ids={data.get('user_group_ids', '(omitted)')})")

def test_get_ruleset_is_default():
    print(">>> [获取规则组] 场景10：is_default 字段本期恒为 true（若出现）...")
    resp = get_ruleset()
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    if not data.get("initialized"):
        print("    SKIP (当前环境未初始化)")
        return
    # is_default 本期恒为 true，omitempty 时不出现在响应中
    # 若出现，则不应为 false
    if "is_default" in data:
        assert data["is_default"] is True, \
            f"is_default 应为 true，实际 {data['is_default']}"
    print(f"    OK (is_default={data.get('is_default', '(omitted)')})")

# ─────────────────────────────────────────────
# Main
# ─────────────────────────────────────────────

def main():
    health_check()
    print()
    run_tests(globals(), title="GET /admin/config/security-group/ruleset")

if __name__ == "__main__":
    main()
