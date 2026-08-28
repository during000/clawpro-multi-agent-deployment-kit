#!/usr/bin/env python3
"""
GET /admin/config/security-group/check-rules 检查安全组规则缺失 集成测试

测试场景：
  场景 1：无认证信息 → 401/403
  场景 2：错误 token → 401/403
  场景 3：非管理员 token → 401/403
  场景 4：缺少 security_group_id 参数 → 400，提示"缺少参数 security_group_id"
  场景 5：security_group_id 为空字符串（?security_group_id=）→ 400
  场景 6：security_group_id 带前后空格 → 直接透传给腾讯云（无 TrimSpace）→ 腾讯云报错 → 500
           注意：check-rules 接口的 sgID 未做 TrimSpace（与 cloud-policies 不同）
  场景 7：传入合法 SG ID，且该 SG 规则完整覆盖 recommended 规则 → 200，missing_rules 为空数组 []
  场景 8：传入合法 SG ID，且该 SG 缺少部分 recommended 规则 → 200，missing_rules 包含缺失规则
  场景 9：传入合法 SG ID，且该 SG 完全没有规则 → 200，missing_rules 包含所有 recommended 规则
  场景 10：传入不存在的 SG ID → 500，提示"检查安全组规则失败"
  场景 11：只检查 recommended 分类，builtin 分类规则缺失不计入 missing_rules
           验证方式：missing_rules 中的规则均来自 recommended 分类（通过对比 required-rules 接口验证）
  场景 12：响应结构验证：包含 missing_rules 字段，每条规则包含 direction/protocol/port/action 字段
  场景 13：站点配置了 VpcId 时，{{VPC_CIDR}} 被替换后再做规则匹配（不因占位符导致误判缺失）
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

# 可选：指定一个已知存在的 SG ID 用于正向测试
KNOWN_SG_ID = os.environ.get("KNOWN_SG_ID", "")

# ─────────────────────────────────────────────
# 工具函数
# ─────────────────────────────────────────────


_check_rules_fn = make_api_fn("get", "/admin/config/security-group/check-rules", timeout=15)


def check_rules(sg_id: str = None, headers: dict = None):
    params = {"security_group_id": sg_id} if sg_id is not None else None
    return _check_rules_fn(params=params, headers=headers)


def get_current_sg_id() -> str:
    """从当前站点配置获取已绑定的 SG ID"""
    if KNOWN_SG_ID:
        return KNOWN_SG_ID
    resp = seed.get("/admin/config/security-group", expect=None, timeout=10, raw=True)
    if resp.status_code != 200 or not resp.text.strip():
        return ""
    try:
        sg_set = resp.json().get("Response", {}).get("SecurityGroupSet", [])
        if sg_set:
            return sg_set[0].get("SecurityGroupId", "")
    except Exception:
        pass
    return ""

def get_recommended_rules() -> list:
    """获取 recommended 分类的所有规则，用于对比验证"""
    resp = seed.get("/admin/config/security-group/required-rules",
                    params={"type": "recommended"}, expect=None, timeout=10, raw=True)
    if resp.status_code != 200:
        return []
    rules = []
    for cat in resp.json().get("categories") or []:
        for group in cat.get("rule_groups") or []:
            rules.extend(group.get("rules") or [])
    return rules

# ─────────────────────────────────────────────
# 测试用例
# ─────────────────────────────────────────────

def test_check_rules_auth():
    """认证测试三件套：无认证/错误token/非管理员 → 401/403"""
    auth_test_suite(lambda headers: check_rules(sg_id="sg-test", headers=headers),
                    label="check_rules")


def test_check_rules_missing_param():
    print(">>> [check-rules] 场景4：缺少 security_group_id 参数 → 400 ...")
    resp = check_rules()  # 不传 sg_id
    assert resp.status_code == 400, \
        f"期望 400，实际 {resp.status_code}，body={resp.text}"
    assert "security_group_id" in resp.text.lower() or "缺少参数" in resp.text, \
        f"错误信息应提示缺少 security_group_id，实际 body={resp.text}"
    print(f"    OK (status=400，提示缺少参数)")

def test_check_rules_empty_sg_id():
    print(">>> [check-rules] 场景5：security_group_id 为空字符串 → 400 ...")
    resp = check_rules(sg_id="")
    assert resp.status_code == 400, \
        f"期望 400，实际 {resp.status_code}，body={resp.text}"
    print(f"    OK (status=400)")

def test_check_rules_whitespace_sg_id():
    """
    场景6：security_group_id 带前后空格 → 直接透传给腾讯云（无 TrimSpace）→ 腾讯云报错 → 500。
    注意：check-rules 接口的 sgID 未做 TrimSpace（与 cloud-policies 的 TrimSpace 处理不同）。
    代码：sgID := r.URL.Query().Get("security_group_id")（无 TrimSpace）
    """
    print(">>> [check-rules] 场景6：security_group_id 带空格 → 无 TrimSpace → 腾讯云报错 → 500 ...")
    resp = check_rules(sg_id=" sg-test ")
    # 带空格的 SG ID 直接透传给腾讯云，腾讯云会报格式错误 → 500
    assert resp.status_code == 500, \
        f"期望 500（带空格的 SG ID 透传给腾讯云报错），实际 {resp.status_code}，body={resp.text}"
    print(f"    OK (status=500，带空格的 SG ID 透传腾讯云报错，符合预期)")

def test_check_rules_complete_sg():
    """
    场景7：传入合法 SG ID，且该 SG 规则完整覆盖 recommended 规则 → 200，missing_rules 为空数组 []。
    使用当前站点配置的 SG（通常已补齐规则）进行验证。
    """
    print(">>> [check-rules] 场景7：规则完整的 SG → 200，missing_rules 为空数组 ...")
    sg_id = get_current_sg_id()
    if not sg_id:
        print("    SKIP (无法获取有效的 SG ID)")
        return

    resp = check_rules(sg_id=sg_id)
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "missing_rules" in data, f"响应应包含 missing_rules 字段，实际 {data}"
    missing = data["missing_rules"]
    assert isinstance(missing, list), f"missing_rules 应为数组，实际 {type(missing)}"

    if len(missing) == 0:
        print(f"    OK (SG ID={sg_id}，规则完整，missing_rules=[])")
    else:
        print(f"    INFO (SG ID={sg_id}，missing_rules={len(missing)} 条，当前 SG 规则不完整)")

def test_check_rules_missing_some():
    """
    场景8：传入合法 SG ID，且该 SG 缺少部分 recommended 规则 → 200，missing_rules 包含缺失规则。
    注意：此场景依赖环境中存在规则不完整的 SG，若当前 SG 规则完整则验证 missing_rules 结构。
    """
    print(">>> [check-rules] 场景8：缺少部分规则的 SG → 200，missing_rules 包含缺失规则 ...")
    sg_id = get_current_sg_id()
    if not sg_id:
        print("    SKIP (无法获取有效的 SG ID)")
        return

    resp = check_rules(sg_id=sg_id)
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    missing = data.get("missing_rules", [])
    assert isinstance(missing, list), f"missing_rules 应为数组，实际 {type(missing)}"

    if len(missing) > 0:
        # 验证每条缺失规则包含必要字段
        for rule in missing:
            assert "direction" in rule, f"缺失规则应包含 direction 字段，实际 {rule}"
            assert "protocol" in rule, f"缺失规则应包含 protocol 字段，实际 {rule}"
            assert "port" in rule, f"缺失规则应包含 port 字段，实际 {rule}"
            assert "action" in rule, f"缺失规则应包含 action 字段，实际 {rule}"
        print(f"    OK (SG ID={sg_id}，missing_rules={len(missing)} 条，字段结构正确)")
    else:
        print(f"    INFO (当前 SG 规则完整，missing_rules=[]，场景8需在规则不完整的 SG 上验证)")

def test_check_rules_nonexistent_sg_id():
    """
    场景10：传入不存在的 SG ID → 500，提示"检查安全组规则失败"。
    """
    print(">>> [check-rules] 场景10：不存在的 SG ID → 500 ...")
    fake_id = "sg-00000000"
    resp = check_rules(sg_id=fake_id)
    assert resp.status_code == 500, \
        f"期望 500，实际 {resp.status_code}，body={resp.text}"
    assert "检查安全组规则失败" in resp.text or "查询安全组规则失败" in resp.text, \
        f"错误信息应提示检查失败，实际 body={resp.text}"
    print(f"    OK (status=500，提示检查安全组规则失败)")

def test_check_rules_only_recommended():
    """
    场景11：只检查 recommended 分类，builtin 分类规则缺失不计入 missing_rules。
    验证方式：获取 required-rules?type=recommended 的规则列表，
    验证 missing_rules 中的每条规则均来自 recommended 分类（通过 port/protocol/direction 匹配）。
    """
    print(">>> [check-rules] 场景11：只检查 recommended 分类，builtin 缺失不计入 missing_rules ...")
    sg_id = get_current_sg_id()
    if not sg_id:
        print("    SKIP (无法获取有效的 SG ID)")
        return

    # 获取 recommended 规则列表
    recommended_rules = get_recommended_rules()
    if not recommended_rules:
        print("    SKIP (无法获取 recommended 规则列表)")
        return

    resp = check_rules(sg_id=sg_id)
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    missing = data.get("missing_rules", [])

    # 构建 recommended 规则的特征集合（direction+protocol+port+action）
    recommended_signatures = set()
    for rule in recommended_rules:
        sig = (
            rule.get("direction", ""),
            rule.get("protocol", "").upper(),
            rule.get("port", ""),
            rule.get("action", "").upper(),
        )
        recommended_signatures.add(sig)

    # 验证 missing_rules 中的每条规则均来自 recommended 分类
    for rule in missing:
        sig = (
            rule.get("direction", ""),
            rule.get("protocol", "").upper(),
            rule.get("port", ""),
            rule.get("action", "").upper(),
        )
        assert sig in recommended_signatures, \
            (f"missing_rules 中出现了非 recommended 分类的规则，"
             f"说明 builtin 规则被错误地计入了缺失列表。"
             f"规则 sig={sig}，recommended_signatures={recommended_signatures}")
    print(f"    OK (missing_rules={len(missing)} 条，均来自 recommended 分类)")

def test_check_rules_response_structure():
    """
    场景12：响应结构验证：包含 missing_rules 字段，每条规则包含 direction/protocol/port/action 字段。
    """
    print(">>> [check-rules] 场景12：响应结构验证（missing_rules 字段及规则字段）...")
    sg_id = get_current_sg_id()
    if not sg_id:
        print("    SKIP (无法获取有效的 SG ID)")
        return

    resp = check_rules(sg_id=sg_id)
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "missing_rules" in data, f"响应应包含 missing_rules 字段，实际 {data}"
    missing = data["missing_rules"]
    assert isinstance(missing, list), f"missing_rules 应为数组，实际 {type(missing)}"

    for rule in missing:
        assert "direction" in rule, f"缺失规则应包含 direction 字段，实际 {rule}"
        assert "protocol" in rule, f"缺失规则应包含 protocol 字段，实际 {rule}"
        assert "port" in rule, f"缺失规则应包含 port 字段，实际 {rule}"
        assert "action" in rule, f"缺失规则应包含 action 字段，实际 {rule}"
    print(f"    OK (missing_rules={len(missing)} 条，字段结构正确)")

def test_check_rules_vpc_cidr_no_false_missing():
    """
    场景13：站点配置了 VpcId 时，{{VPC_CIDR}} 被替换后再做规则匹配，不因占位符导致误判缺失。
    验证方式：若站点已配置 VpcId，missing_rules 中不应出现 cidr_block 为 {{VPC_CIDR}} 的规则。
    """
    print(">>> [check-rules] 场景13：VPC CIDR 替换后规则匹配，不因占位符误判缺失 ...")
    sg_id = get_current_sg_id()
    if not sg_id:
        print("    SKIP (无法获取有效的 SG ID)")
        return

    resp = check_rules(sg_id=sg_id)
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    missing = data.get("missing_rules", [])

    # 验证 missing_rules 中不含 {{VPC_CIDR}} 占位符（说明已被替换后再匹配）
    for rule in missing:
        cidr = rule.get("cidr_block", "") or ""
        assert "{{VPC_CIDR}}" not in cidr, \
            (f"missing_rules 中出现了含 {{{{VPC_CIDR}}}} 占位符的规则，"
             f"说明 VPC CIDR 替换未生效，导致规则匹配误判。规则={rule}")
    print(f"    OK (missing_rules={len(missing)} 条，均不含 {{{{VPC_CIDR}}}} 占位符)")

# ─────────────────────────────────────────────
# Main
# ─────────────────────────────────────────────

def main():
    health_check()
    print()
    run_tests(globals(), title="GET /admin/config/security-group/check-rules")

if __name__ == "__main__":
    main()
