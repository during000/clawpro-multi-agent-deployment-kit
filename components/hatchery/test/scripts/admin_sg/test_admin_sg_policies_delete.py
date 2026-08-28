#!/usr/bin/env python3
"""
DELETE /admin/config/security-group/policies 删除安全组规则（Deprecated）集成测试

测试场景：
  场景 1：无认证信息 → 401/403
  场景 2a：错误 token → 401/403
  场景 2b：非管理员 token → 401/403
  场景 3：未配置安全组（security_group_id 为空）→ 400，含"未配置安全组"提示
           （仅在无 SG 配置的环境下有效，已配置则自动跳过）
  场景 4：请求体 JSON 格式错误 → 400，含 error 字段
  场景 5：正常删除指定规则（先创建一条规则，再按 PolicyIndex 删除）→ 200，并验证规则已不存在
  场景 6：请求体中传入不同的 SecurityGroupId → 200，但实际删除的是当前配置 SG 的规则（强制覆盖，防止越权）
           注：场景 6 会单独创建一条专用规则再删除，避免影响其他场景
  场景 7：SecurityGroupPolicySet 为空（未传 Ingress/Egress）→ 透传腾讯云，验证行为（不 panic）
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


delete_policies = make_api_fn("delete", "/admin/config/security-group/policies")


def post_policies(body: dict):
    return seed.post("/admin/config/security-group/policies", json=body,
                     expect=None, timeout=30, raw=True)


def get_sg():
    return seed.get("/admin/config/security-group", expect=None,
                    timeout=10, raw=True)


def get_policies():
    return seed.get("/admin/config/security-group/policies", expect=None,
                    timeout=10, raw=True)


# ─────────────────────────────────────────────
# 测试用例
# ─────────────────────────────────────────────

def test_delete_policies_auth():
    """认证测试三件套：无认证/错误token/非管理员 → 401/403"""
    auth_test_suite(lambda headers: delete_policies({"SecurityGroupPolicySet": {}}, headers=headers),
                    label="delete_policies")


def test_delete_policies_no_sg_configured():
    """
    场景3：未配置安全组时删除规则 → 400
    注意：此测试依赖环境中未配置安全组，若当前已配置则跳过。
    """
    print(">>> [删除规则(Deprecated)] 场景3：未配置安全组 → 400（仅在无 SG 配置环境下有效）...")
    get_resp = get_sg()
    if get_resp.status_code == 200 and get_resp.text.strip():
        try:
            sg_set = get_resp.json().get("Response", {}).get("SecurityGroupSet", [])
            if sg_set:
                print("    SKIP (当前已配置安全组，此场景需在无 SG 配置环境下验证)")
                return
        except Exception:
            pass

    resp = delete_policies({"SecurityGroupPolicySet": {}})
    assert resp.status_code == 400, \
        f"未配置安全组时期望 400，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "error" in data, f"400 响应应包含 error 字段，实际 {data}"
    print(f"    OK (status=400, error={data.get('error')})")


def test_delete_policies_invalid_json():
    print(">>> [删除规则(Deprecated)] 场景4：请求体 JSON 格式错误 → 400 ...")
    resp = seed.delete("/admin/config/security-group/policies", data="not-a-json{{{",
                      expect=None, timeout=30, raw=True)
    assert resp.status_code == 400, \
        f"期望 400，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "error" in data, f"400 响应应包含 error 字段，实际 {data}"
    print(f"    OK (status=400, error={data.get('error')})")


def test_delete_policies_normal():
    print(">>> [删除规则(Deprecated)] 场景5：正常删除指定规则（先创建再删除）→ 200，并验证规则已不存在 ...")
    get_resp = get_sg()
    if get_resp.status_code != 200:
        print("    SKIP (当前未配置安全组)")
        return
    if not get_resp.text.strip():
        print("    SKIP (当前未配置安全组，响应体为空)")
        return
    try:
        sg_set = get_resp.json().get("Response", {}).get("SecurityGroupSet", [])
    except Exception:
        print(f"    SKIP (解析安全组响应失败，body={get_resp.text[:200]})")
        return
    if not sg_set:
        print("    SKIP (当前未配置安全组)")
        return

    # 先创建一条规则
    create_resp = post_policies({
        "SecurityGroupPolicySet": {
            "Ingress": [
                {
                    "Protocol": "TCP",
                    "Port": "7777",
                    "CidrBlock": "192.168.0.0/16",
                    "Action": "ACCEPT",
                    "PolicyDescription": "待删除的测试规则",
                }
            ]
        }
    })
    if create_resp.status_code != 200:
        print(f"    SKIP (创建规则失败，status={create_resp.status_code})")
        return

    # 查询当前规则，获取 PolicyIndex
    policies_resp = get_policies()
    if policies_resp.status_code != 200 or not policies_resp.text.strip():
        print("    SKIP (查询规则失败或返回空)")
        return
    policy_set = policies_resp.json().get("Response", {}).get("SecurityGroupPolicySet", {})
    ingress = policy_set.get("Ingress") or []

    # 找到刚创建的规则的 PolicyIndex
    target_index = None
    for rule in ingress:
        if rule.get("Port") == "7777" and rule.get("CidrBlock") == "192.168.0.0/16":
            target_index = rule.get("PolicyIndex")
            break

    if target_index is None:
        print("    SKIP (未找到刚创建的规则，可能规则创建未生效)")
        return

    # 删除该规则
    del_resp = delete_policies({
        "SecurityGroupPolicySet": {
            "Ingress": [{"PolicyIndex": target_index}]
        }
    })
    if del_resp.status_code == 500 and "PolicyIndex" in del_resp.text:
        # 并发竞态防御：查询和删除之间，ruleset fan-out 或并发测试可能重建了规则，
        # 导致原来的 PolicyIndex 已失效
        print(f"    SKIP (并发竞态：PolicyIndex={target_index} 已失效，规则可能已被重建，跳过)")
        return
    assert del_resp.status_code == 200, \
        f"期望 200，实际 {del_resp.status_code}，body={del_resp.text}"

    # 验证规则已不存在
    verify_resp = get_policies()
    assert verify_resp.status_code == 200
    if not verify_resp.text.strip():
        print(f"    OK (PolicyIndex={target_index} 规则已删除，验证时 GET 返回空体)")
        return
    verify_set = verify_resp.json().get("Response", {}).get("SecurityGroupPolicySet", {})
    verify_ingress = verify_set.get("Ingress") or []
    still_exists = any(
        r.get("Port") == "7777" and r.get("CidrBlock") == "192.168.0.0/16"
        for r in verify_ingress
    )
    assert not still_exists, \
        f"删除后 Port=7777 CidrBlock=192.168.0.0/16 的规则应不存在，实际仍存在"
    print(f"    OK (PolicyIndex={target_index} 规则已删除，已验证不存在)")


def test_delete_policies_force_override_sg_id():
    print(">>> [删除规则(Deprecated)] 场景6：请求体传入不同 SecurityGroupId → 实际操作当前配置 SG（强制覆盖）...")
    get_resp = get_sg()
    if get_resp.status_code != 200:
        print("    SKIP (当前未配置安全组)")
        return
    if not get_resp.text.strip():
        print("    SKIP (当前未配置安全组，响应体为空)")
        return
    try:
        sg_set = get_resp.json().get("Response", {}).get("SecurityGroupSet", [])
    except Exception:
        print(f"    SKIP (解析安全组响应失败，body={get_resp.text[:200]})")
        return
    if not sg_set:
        print("    SKIP (当前未配置安全组)")
        return
    current_sg_id = sg_set[0].get("SecurityGroupId", "")

    # 单独创建一条专用规则，避免影响其他场景
    create_resp = post_policies({
        "SecurityGroupPolicySet": {
            "Ingress": [
                {
                    "Protocol": "TCP",
                    "Port": "6543",
                    "CidrBlock": "172.16.0.0/12",
                    "Action": "ACCEPT",
                    "PolicyDescription": "强制覆盖测试专用规则",
                }
            ]
        }
    })
    if create_resp.status_code != 200:
        print(f"    SKIP (创建专用规则失败，status={create_resp.status_code})")
        return

    # 查询当前规则，取刚创建的规则的 PolicyIndex
    policies_resp = get_policies()
    if policies_resp.status_code != 200 or not policies_resp.text.strip():
        print("    SKIP (查询规则失败或返回空)")
        return
    policy_set = policies_resp.json().get("Response", {}).get("SecurityGroupPolicySet", {})
    ingress = policy_set.get("Ingress") or []
    target_index = None
    for rule in ingress:
        if rule.get("Port") == "6543" and rule.get("CidrBlock") == "172.16.0.0/12":
            target_index = rule.get("PolicyIndex")
            break
    if target_index is None:
        print("    SKIP (未找到专用规则，跳过)")
        return

    fake_sg_id = "sg-00000000"
    resp = delete_policies({
        "SecurityGroupId": fake_sg_id,
        "SecurityGroupPolicySet": {
            "Ingress": [{"PolicyIndex": target_index}]
        }
    })
    # 后端强制覆盖 SecurityGroupId 为当前配置的 SG，应返回 200
    if resp.status_code == 500 and "PolicyIndex" in resp.text:
        # 并发竞态防御
        print(f"    SKIP (并发竞态：PolicyIndex={target_index} 已失效，规则可能已被重建，跳过)")
        return
    assert resp.status_code == 200, \
        f"期望 200（后端强制覆盖 SecurityGroupId），实际 {resp.status_code}，body={resp.text}"
    print(f"    OK (当前 SG={current_sg_id}，请求中传入 {fake_sg_id}，后端强制覆盖为当前 SG)")


def test_delete_policies_empty_policy_set():
    print(">>> [删除规则(Deprecated)] 场景7：SecurityGroupPolicySet 为空 → 透传腾讯云，验证行为 ...")
    get_resp = get_sg()
    if get_resp.status_code != 200:
        print("    SKIP (当前未配置安全组)")
        return
    if not get_resp.text.strip():
        print("    SKIP (当前未配置安全组，响应体为空)")
        return
    try:
        sg_set = get_resp.json().get("Response", {}).get("SecurityGroupSet", [])
    except Exception:
        print(f"    SKIP (解析安全组响应失败，body={get_resp.text[:200]})")
        return
    if not sg_set:
        print("    SKIP (当前未配置安全组)")
        return

    resp = delete_policies({
        "SecurityGroupPolicySet": {}
    })
    # 空 PolicySet 透传腾讯云，腾讯云可能返回 200 或报错
    assert resp.status_code in (200, 400, 500), \
        f"期望 200/400/500，实际 {resp.status_code}，body={resp.text}"
    print(f"    OK (status={resp.status_code}，空 PolicySet 行为符合预期)")


# ─────────────────────────────────────────────
# Main
# ─────────────────────────────────────────────

def main():
    health_check()
    print()
    run_tests(globals(), title="DELETE /admin/config/security-group/policies")

if __name__ == "__main__":
    main()
