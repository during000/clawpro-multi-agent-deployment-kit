#!/usr/bin/env python3
"""
GET /admin/config/security-group/policies 查询安全组规则 集成测试

测试场景：
  场景 1：无认证信息 → 401/403
  场景 2a：错误 token → 401/403
  场景 2b：非管理员 token → 401/403
  场景 3：未配置安全组（security_group_id 为空）→ 200，响应体为空（静默返回，不是 400）
           （仅在无 SG 配置的环境下有效，已配置则自动跳过）
           ⚠️ 注意：与 POST /policies 未配置时返回 400 不同，GET 是静默返回 200 空体
  场景 4：已配置安全组，正常查询 → 200，响应体包含 Response.SecurityGroupPolicySet，
           SecurityGroupPolicySet 包含 Ingress 和/或 Egress 字段
  场景 5：响应体为合法 JSON（不 panic，不返回非 JSON 内容）
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


def get_sg():
    return seed.get("/admin/config/security-group", expect=None,
                    timeout=10, raw=True)

get_policies = make_api_fn("get", "/admin/config/security-group/policies", timeout=10)


# ─────────────────────────────────────────────
# 测试用例
# ─────────────────────────────────────────────

def test_get_policies_auth():
    """认证测试三件套：无认证/错误token/非管理员 → 401/403"""
    auth_test_suite(lambda headers: get_policies(headers=headers),
                    label="get_policies")


def test_get_policies_no_sg_configured():
    """
    场景3：未配置安全组时查询规则 → 200，响应体为空（静默返回）
    注意：此测试依赖环境中未配置安全组，若当前已配置则跳过。
    ⚠️ 关键断言：状态码必须是 200 而非 400（与 POST /policies 未配置时返回 400 行为不同）。
    """
    print(">>> [查询安全组规则] 场景3：未配置安全组 → 200 空体（静默返回，不是 400）...")
    # 先检查当前是否已配置安全组
    sg_resp = get_sg()
    if sg_resp.status_code == 200 and sg_resp.text.strip():
        try:
            sg_set = sg_resp.json().get("Response", {}).get("SecurityGroupSet", [])
            if sg_set:
                print("    SKIP (当前已配置安全组，此场景需在无 SG 配置环境下验证)")
                return
        except Exception:
            pass

    resp = get_policies()
    # 未配置安全组时，接口应静默返回 200 空体（不是 400）
    assert resp.status_code == 200, \
        f"未配置安全组时期望 200（静默返回），实际 {resp.status_code}，body={resp.text}"
    assert resp.text.strip() == "", \
        f"未配置安全组时响应体应为空，实际 body={repr(resp.text)}"
    print(f"    OK (status=200，响应体为空，符合静默返回预期)")

def test_get_policies_normal():
    """
    场景4：已配置安全组，正常查询 → 200，返回 Response.SecurityGroupPolicySet，
    SecurityGroupPolicySet 包含 Ingress 和/或 Egress 字段（可为空列表，但字段必须存在）。
    """
    print(">>> [查询安全组规则] 场景4：已配置安全组，正常查询 → 200，SecurityGroupPolicySet 结构正确 ...")
    sg_resp = get_sg()
    if sg_resp.status_code != 200 or not sg_resp.text.strip():
        print("    SKIP (当前未配置安全组，跳过正常查询验证)")
        return
    try:
        sg_set = sg_resp.json().get("Response", {}).get("SecurityGroupSet", [])
        if not sg_set:
            print("    SKIP (当前未配置安全组，跳过正常查询验证)")
            return
    except Exception:
        print("    SKIP (GET SG 响应解析失败，跳过)")
        return

    resp = get_policies()
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    assert resp.text.strip(), \
        "已配置安全组时响应体不应为空"

    data = resp.json()
    assert "Response" in data, f"响应应包含 Response 字段，实际 {data}"
    policy_set = data.get("Response", {}).get("SecurityGroupPolicySet")
    assert policy_set is not None, \
        f"Response 应包含 SecurityGroupPolicySet 字段，实际 {data.get('Response')}"
    # Ingress / Egress 字段可为 null 或空列表，但 SecurityGroupPolicySet 本身必须存在
    ingress = policy_set.get("Ingress") or []
    egress = policy_set.get("Egress") or []
    print(f"    OK (ingress={len(ingress)} 条，egress={len(egress)} 条)")

def test_get_policies_response_is_valid_json():
    """
    场景5：响应体为合法 JSON（不 panic，不返回非 JSON 内容）
    """
    print(">>> [查询安全组规则] 场景5：响应体为合法 JSON ...")
    resp = get_policies()
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"

    if not resp.text.strip():
        # 未配置安全组时响应体为空，属于正常情况
        print("    OK (响应体为空，符合未配置安全组时的静默返回预期)")
        return

    try:
        data = resp.json()
        assert isinstance(data, dict), f"响应体应为 JSON 对象，实际 {type(data)}"
    except Exception as e:
        assert False, f"响应体应为合法 JSON，实际 body={resp.text}，error={e}"
    print(f"    OK (响应体为合法 JSON)")

# ─────────────────────────────────────────────
# Main
# ─────────────────────────────────────────────

def main():
    health_check()
    print()
    run_tests(globals(), title="GET /admin/config/security-group/policies")

if __name__ == "__main__":
    main()
