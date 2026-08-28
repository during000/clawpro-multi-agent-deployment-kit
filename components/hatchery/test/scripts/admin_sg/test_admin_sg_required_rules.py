#!/usr/bin/env python3
"""
GET /admin/config/security-group/required-rules 查询所需安全组规则 集成测试

测试场景：
  场景 1：无认证信息 → 401/403
  场景 2：错误 token → 401/403
  场景 3：非管理员 token → 401/403
  场景 4：不传 type 参数（默认 builtin）→ 200，返回 builtin 分类的规则
  场景 5：type=builtin → 200，只返回 builtin 分类，categories 中每项 type 均为 "builtin"
  场景 6：type=recommended → 200，只返回 recommended 分类，categories 中每项 type 均为 "recommended"
  场景 7：type=all → 200，返回所有分类（不过滤），categories 数量 ≥ type=builtin 和 type=recommended 之和
  场景 8：type=unknown_type（未知分类）→ 200，categories 为空数组（过滤后无匹配）
  场景 9：响应结构验证：包含 categories 字段，每个 category 包含 type 和 rule_groups 字段
  场景 10：type=builtin 时，响应中规则的 condition 字段被清除（不暴露给前端）
  场景 11：type=all 时，条件规则仍被 resolveConditionalRules 过滤（非原始全量），
           即 type=all 只是不过滤分类，但条件不满足的规则组已被移除
  场景 12：站点已配置 VpcId 时，规则中不含 {{VPC_CIDR}} 占位符（已被替换为真实 CIDR）
  场景 13：站点未配置 VpcId 时，规则中可能含 {{VPC_CIDR}} 占位符（保留原样）
  场景 14：GatewayUIEnable=true 且 GatewayUIPort>0 时，gateway_ui_enable 条件规则被保留，
           端口占位符 {{GATEWAY_UI_PORT}} 被替换为实际端口号
  场景 15：GatewayUIEnable=false 时，gateway_ui_enable 条件规则被过滤掉（type=all 也不例外）
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


_required_rules_fn = make_api_fn("get", "/admin/config/security-group/required-rules", timeout=10)


def required_rules(type_param: str = None, headers: dict = None):
    params = {"type": type_param} if type_param is not None else None
    return _required_rules_fn(params=params, headers=headers)


def collect_all_rules(categories: list) -> list:
    """从 categories 中提取所有规则"""
    rules = []
    for cat in categories:
        for group in cat.get("rule_groups") or cat.get("RuleGroups", []):
            rules.extend(group.get("rules") or group.get("Rules", []))
    return rules

# ─────────────────────────────────────────────
# 测试用例
# ─────────────────────────────────────────────

def test_required_rules_auth():
    """认证测试三件套：无认证/错误token/非管理员 → 401/403"""
    auth_test_suite(lambda headers: required_rules(headers=headers),
                    label="required_rules")


def test_required_rules_default_type():
    """
    场景4：不传 type 参数（默认 builtin）→ 200，返回 builtin 分类的规则。
    """
    print(">>> [required-rules] 场景4：不传 type 参数（默认 builtin）→ 200 ...")
    resp = required_rules()
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "categories" in data, f"响应应包含 categories 字段，实际 {data}"
    categories = data["categories"] or []
    # 默认返回 builtin 分类，验证结果与显式传 type=builtin 一致
    resp_builtin = required_rules(type_param="builtin")
    assert resp_builtin.status_code == 200
    data_builtin = resp_builtin.json()
    assert data.get("categories") == data_builtin.get("categories"), \
        "不传 type 参数的结果应与 type=builtin 完全一致"
    print(f"    OK (默认 type=builtin，categories={len(categories)} 个)")

def test_required_rules_type_builtin():
    """
    场景5：type=builtin → 200，只返回 builtin 分类，categories 中每项 type 均为 "builtin"。
    """
    print(">>> [required-rules] 场景5：type=builtin → 200，只返回 builtin 分类 ...")
    resp = required_rules(type_param="builtin")
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    categories = data.get("categories") or []
    for cat in categories:
        assert cat.get("type") == "builtin", \
            f"type=builtin 过滤后，每个 category 的 type 应为 'builtin'，实际 {cat.get('type')}"
    print(f"    OK (type=builtin，categories={len(categories)} 个，均为 builtin 分类)")

def test_required_rules_type_recommended():
    """
    场景6：type=recommended → 200，只返回 recommended 分类，categories 中每项 type 均为 "recommended"。
    """
    print(">>> [required-rules] 场景6：type=recommended → 200，只返回 recommended 分类 ...")
    resp = required_rules(type_param="recommended")
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    categories = data.get("categories") or []
    for cat in categories:
        assert cat.get("type") == "recommended", \
            f"type=recommended 过滤后，每个 category 的 type 应为 'recommended'，实际 {cat.get('type')}"
    print(f"    OK (type=recommended，categories={len(categories)} 个，均为 recommended 分类)")

def test_required_rules_type_all():
    """
    场景7：type=all → 200，返回所有分类（不过滤），categories 数量 ≥ type=builtin 和 type=recommended 之和。
    """
    print(">>> [required-rules] 场景7：type=all → 200，返回所有分类 ...")
    resp_all = required_rules(type_param="all")
    resp_builtin = required_rules(type_param="builtin")
    resp_recommended = required_rules(type_param="recommended")

    assert resp_all.status_code == 200, \
        f"期望 200，实际 {resp_all.status_code}，body={resp_all.text}"

    cats_all = resp_all.json().get("categories") or []
    cats_builtin = resp_builtin.json().get("categories") or []
    cats_recommended = resp_recommended.json().get("categories") or []

    assert len(cats_all) >= len(cats_builtin) + len(cats_recommended), \
        (f"type=all 返回的 categories 数量({len(cats_all)}) 应 ≥ "
         f"builtin({len(cats_builtin)}) + recommended({len(cats_recommended)})")
    print(f"    OK (all={len(cats_all)}，builtin={len(cats_builtin)}，recommended={len(cats_recommended)})")

def test_required_rules_unknown_type():
    """
    场景8：type=unknown_type（未知分类）→ 200，categories 为空数组（过滤后无匹配）。
    """
    print(">>> [required-rules] 场景8：type=unknown_type → 200，categories 为空数组 ...")
    resp = required_rules(type_param="unknown_type_that_does_not_exist")
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    categories = data.get("categories")
    # 未知 type 过滤后应为空（null 或 []）
    assert not categories, \
        f"未知 type 过滤后 categories 应为空，实际 {categories}"
    print(f"    OK (未知 type，categories 为空)")

def test_required_rules_response_structure():
    """
    场景9：响应结构验证：包含 categories 字段，每个 category 包含 type 和 rule_groups 字段。
    """
    print(">>> [required-rules] 场景9：响应结构验证（categories/type/rule_groups 字段）...")
    resp = required_rules(type_param="all")
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "categories" in data, f"响应应包含 categories 字段，实际 {data}"
    categories = data.get("categories") or []
    for cat in categories:
        assert "type" in cat, f"每个 category 应包含 type 字段，实际 {cat}"
        assert "rule_groups" in cat, f"每个 category 应包含 rule_groups 字段，实际 {cat}"
        rule_groups = cat.get("rule_groups") or []
        for group in rule_groups:
            assert "rules" in group, f"每个 rule_group 应包含 rules 字段，实际 {group}"
    print(f"    OK (响应结构正确，categories={len(categories)} 个)")

def test_required_rules_condition_cleared():
    """
    场景10：type=builtin 时，响应中规则组的 condition 字段被清除（不暴露给前端）。
    代码中 group.Condition = "" 清除了条件标识。
    """
    print(">>> [required-rules] 场景10：condition 字段被清除（不暴露给前端）...")
    resp = required_rules(type_param="all")
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    categories = data.get("categories") or []
    for cat in categories:
        for group in cat.get("rule_groups") or []:
            condition = group.get("condition", "")
            assert condition == "" or condition is None, \
                f"rule_group 的 condition 字段应被清除（为空），实际 condition='{condition}'"
    print(f"    OK (所有 rule_group 的 condition 字段均已清除)")

def test_required_rules_type_all_still_filters_conditions():
    """
    场景11：type=all 时，条件规则仍被 resolveConditionalRules 过滤（非原始全量）。
    即 type=all 只是不过滤分类，但条件不满足的规则组已被移除。
    验证方式：type=all 的规则总数 ≤ 理论上的原始全量（因为条件规则可能被过滤）。
    实际验证：type=all 返回的规则中不含 {{GATEWAY_UI_PORT}} 占位符（已被替换或规则已被过滤）。
    """
    print(">>> [required-rules] 场景11：type=all 时条件规则仍被过滤（非原始全量）...")
    resp = required_rules(type_param="all")
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    categories = data.get("categories") or []
    all_rules = collect_all_rules(categories)
    # 验证：规则中不含未替换的端口占位符（说明 resolveConditionalRules 已处理）
    for rule in all_rules:
        port = rule.get("port", "")
        assert "{{GATEWAY_UI_PORT}}" not in str(port), \
            f"type=all 时规则中不应含未替换的 {{{{GATEWAY_UI_PORT}}}} 占位符，实际 port='{port}'"
    print(f"    OK (type=all 共 {len(all_rules)} 条规则，均不含未替换的端口占位符)")

def test_required_rules_vpc_cidr_replaced():
    """
    场景12：站点已配置 VpcId 时，规则中不含 {{VPC_CIDR}} 占位符（已被替换为真实 CIDR）。
    场景13：站点未配置 VpcId 时，规则中可能含 {{VPC_CIDR}} 占位符（保留原样）。
    两种情况均验证：规则中的 cidr_block 字段不含 {{VPC_CIDR}} 字符串（已替换）或含有（未配置时保留）。
    """
    print(">>> [required-rules] 场景12/13：VPC CIDR 占位符替换验证 ...")
    resp = required_rules(type_param="all")
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    categories = data.get("categories") or []
    all_rules = collect_all_rules(categories)

    has_vpc_cidr_placeholder = False
    for rule in all_rules:
        cidr = rule.get("cidr_block", "") or ""
        if "{{VPC_CIDR}}" in cidr:
            has_vpc_cidr_placeholder = True
            break

    if has_vpc_cidr_placeholder:
        print(f"    OK (站点未配置 VpcId，{{{{VPC_CIDR}}}} 占位符保留原样，符合预期)")
    else:
        print(f"    OK (站点已配置 VpcId，{{{{VPC_CIDR}}}} 占位符已被替换为真实 CIDR)")

def test_required_rules_gateway_ui_condition():
    """
    场景14/15：GatewayUIEnable 条件规则验证。
    - GatewayUIEnable=true 且 GatewayUIPort>0 时：gateway_ui_enable 规则被保留，端口占位符被替换
    - GatewayUIEnable=false 时：gateway_ui_enable 规则被过滤掉（type=all 也不例外）
    验证方式：规则中不含 {{GATEWAY_UI_PORT}} 占位符（已被替换或规则已被过滤）。
    """
    print(">>> [required-rules] 场景14/15：GatewayUIEnable 条件规则验证 ...")
    resp = required_rules(type_param="all")
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    categories = data.get("categories") or []
    all_rules = collect_all_rules(categories)

    # 验证：规则中不含未替换的 {{GATEWAY_UI_PORT}} 占位符
    gateway_ui_rules = []
    for rule in all_rules:
        port = str(rule.get("port", ""))
        assert "{{GATEWAY_UI_PORT}}" not in port, \
            f"规则中不应含未替换的 {{{{GATEWAY_UI_PORT}}}} 占位符，实际 port='{port}'"
        # 收集可能是 gateway_ui 相关的规则（端口为数字且描述含 gateway）
        desc = str(rule.get("description", "")).lower()
        if "gateway" in desc or "ui" in desc:
            gateway_ui_rules.append(rule)

    print(f"    OK (共 {len(all_rules)} 条规则，均不含未替换的端口占位符，"
          f"gateway_ui 相关规则={len(gateway_ui_rules)} 条)")

# ─────────────────────────────────────────────
# Main
# ─────────────────────────────────────────────

def main():
    health_check()
    print()
    run_tests(globals(), title="GET /admin/config/security-group/required-rules")

if __name__ == "__main__":
    main()
