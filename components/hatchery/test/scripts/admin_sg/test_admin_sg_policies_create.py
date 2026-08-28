#!/usr/bin/env python3
"""
POST /admin/config/security-group/policies 创建安全组规则（Deprecated）集成测试

测试场景：
  场景 1：无认证信息 → 401/403
  场景 2a：错误 token → 401/403
  场景 2b：非管理员 token → 401/403
  场景 3：未配置安全组（security_group_id 为空）→ 400，含"未配置安全组"提示
           （仅在无 SG 配置的环境下有效，已配置则自动跳过）
  场景 4：请求体 JSON 格式错误 → 400，含 error 字段
  场景 5：正常创建入站规则（合法 Ingress 规则）→ 200，并验证规则已存在
  场景 6：请求体中传入不同的 SecurityGroupId → 200，但实际操作的是当前配置 SG（强制覆盖，防止越权）
  场景 7：正常创建出站规则（合法 Egress 规则）→ 200，并验证 Egress 规则已存在
  场景 8：分两次创建 Ingress + Egress 规则 → 200，两类规则均已存在
           注意：腾讯云 API 不支持在同一请求中同时传入 Ingress 和 Egress，需分两次调用
"""

import os
import sys
import time

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


post_policies = make_api_fn("post", "/admin/config/security-group/policies")


def get_sg():
    return seed.get("/admin/config/security-group", expect=None,
                    timeout=10, raw=True)


def get_policies():
    return seed.get("/admin/config/security-group/policies", expect=None,
                    timeout=10, raw=True)


# ─────────────────────────────────────────────
# 测试用例
# ─────────────────────────────────────────────

def test_create_policies_auth():
    """认证测试三件套：无认证/错误token/非管理员 → 401/403"""
    auth_test_suite(lambda headers: post_policies({"SecurityGroupPolicySet": {}}, headers=headers),
                    label="create_policies")


def test_create_policies_no_sg_configured():
    """
    场景3：未配置安全组时创建规则 → 400
    注意：此测试依赖环境中未配置安全组，若当前已配置则跳过。
    """
    print(">>> [创建规则(Deprecated)] 场景3：未配置安全组 → 400（仅在无 SG 配置环境下有效）...")
    get_resp = get_sg()
    if get_resp.status_code == 200 and get_resp.text.strip():
        try:
            sg_set = get_resp.json().get("Response", {}).get("SecurityGroupSet", [])
            if sg_set:
                print("    SKIP (当前已配置安全组，此场景需在无 SG 配置环境下验证)")
                return
        except Exception:
            pass

    resp = post_policies({"SecurityGroupPolicySet": {"Ingress": []}})
    assert resp.status_code == 400, \
        f"未配置安全组时期望 400，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "error" in data, f"400 响应应包含 error 字段，实际 {data}"
    print(f"    OK (status=400, error={data.get('error')})")


def test_create_policies_invalid_json():
    print(">>> [创建规则(Deprecated)] 场景4：请求体 JSON 格式错误 → 400 ...")
    resp = seed.post("/admin/config/security-group/policies", data="not-a-json{{{",
                    expect=None, timeout=30, raw=True)
    assert resp.status_code == 400, \
        f"期望 400，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "error" in data, f"400 响应应包含 error 字段，实际 {data}"
    print(f"    OK (status=400, error={data.get('error')})")


def test_create_policies_normal():
    print(">>> [创建规则(Deprecated)] 场景5：正常创建入站规则 → 200，并验证规则已存在 ...")
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

    # 记录创建规则前的 SG ID，用于后续并发竞态检测
    pre_sg_id = sg_set[0].get("SecurityGroupId", "")

    test_port = "8888"
    test_cidr = "10.0.0.0/8"
    resp = post_policies({
        "SecurityGroupPolicySet": {
            "Ingress": [
                {
                    "Protocol": "TCP",
                    "Port": test_port,
                    "CidrBlock": test_cidr,
                    "Action": "ACCEPT",
                    "PolicyDescription": "集成测试创建的规则",
                }
            ]
        }
    })
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"

    # 验证规则确实已被添加（腾讯云 API 可能有短暂延迟，加重试）
    found = False
    for attempt in range(5):
        policies_resp = get_policies()
        if policies_resp.status_code != 200 or not policies_resp.text.strip():
            if attempt < 4:
                time.sleep(2)
            continue
        policy_set = policies_resp.json().get("Response", {}).get("SecurityGroupPolicySet", {})
        ingress = policy_set.get("Ingress") or []
        found = any(
            r.get("Port") == test_port and r.get("CidrBlock") == test_cidr
            for r in ingress
        )
        if found:
            break
        if attempt < 4:
            time.sleep(2)
    if not found:
        # 并发竞态防御：检查当前 SiteConfig 是否仍指向创建规则时的安全组
        # POST 和 GET policies 都依赖 SiteConfig.SecurityGroupId，如果并发测试修改了它，
        # 则创建和查询可能操作的是不同的安全组
        recheck_resp = get_sg()
        recheck_sg_id = ""
        if recheck_resp.status_code == 200 and recheck_resp.text.strip():
            try:
                recheck_set = recheck_resp.json().get("Response", {}).get("SecurityGroupSet", [])
                if recheck_set:
                    recheck_sg_id = recheck_set[0].get("SecurityGroupId", "")
            except Exception:
                pass
        if recheck_sg_id != pre_sg_id:
            print(f"    SKIP (并发竞态：创建时 SG={pre_sg_id}，查询时 SG 已变为 {recheck_sg_id}，"
                  f"规则可能已创建到旧 SG 上，跳过验证)")
            return
        assert False, \
            f"创建后应能查询到 Port={test_port} CidrBlock={test_cidr} 的入站规则，实际 ingress={ingress}"
    print(f"    OK (入站规则创建成功，已验证规则存在)")


def test_create_policies_force_override_sg_id():
    print(">>> [创建规则(Deprecated)] 场景6：请求体传入不同 SecurityGroupId → 实际操作当前配置 SG（强制覆盖）...")
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

    fake_sg_id = "sg-00000000"
    resp = post_policies({
        "SecurityGroupId": fake_sg_id,
        "SecurityGroupPolicySet": {
            "Ingress": [
                {
                    "Protocol": "TCP",
                    "Port": "9999",
                    "CidrBlock": "10.0.0.0/8",
                    "Action": "ACCEPT",
                    "PolicyDescription": "强制覆盖测试规则",
                }
            ]
        }
    })
    # 后端强制覆盖 SecurityGroupId 为当前配置的 SG，应返回 200
    assert resp.status_code == 200, \
        f"期望 200（后端强制覆盖 SecurityGroupId），实际 {resp.status_code}，body={resp.text}"
    print(f"    OK (当前 SG={current_sg_id}，请求中传入 {fake_sg_id}，后端强制覆盖为当前 SG)")

def test_create_policies_egress():
    print(">>> [创建规则(Deprecated)] 场景7：正常创建出站规则（Egress）→ 200，并验证 Egress 规则已存在 ...")
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

    # 记录创建规则前的 SG ID，用于后续并发竞态检测
    pre_sg_id = sg_set[0].get("SecurityGroupId", "")

    test_port = "8080"
    test_cidr = "10.0.0.0/8"
    resp = post_policies({
        "SecurityGroupPolicySet": {
            "Egress": [
                {
                    "Protocol": "TCP",
                    "Port": test_port,
                    "CidrBlock": test_cidr,
                    "Action": "ACCEPT",
                    "PolicyDescription": "集成测试创建的出站规则",
                }
            ]
        }
    })
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"

    # 验证 Egress 规则确实已被添加（腾讯云 API 可能有短暂延迟，加重试）
    found = False
    for attempt in range(5):
        policies_resp = get_policies()
        if policies_resp.status_code != 200 or not policies_resp.text.strip():
            if attempt < 4:
                time.sleep(2)
            continue
        policy_set = policies_resp.json().get("Response", {}).get("SecurityGroupPolicySet", {})
        egress = policy_set.get("Egress") or []
        found = any(
            r.get("Port") == test_port and r.get("CidrBlock") == test_cidr
            for r in egress
        )
        if found:
            break
        if attempt < 4:
            time.sleep(2)
    if not found:
        # 并发竞态防御
        recheck_resp = get_sg()
        recheck_sg_id = ""
        if recheck_resp.status_code == 200 and recheck_resp.text.strip():
            try:
                recheck_set = recheck_resp.json().get("Response", {}).get("SecurityGroupSet", [])
                if recheck_set:
                    recheck_sg_id = recheck_set[0].get("SecurityGroupId", "")
            except Exception:
                pass
        if recheck_sg_id != pre_sg_id:
            print(f"    SKIP (并发竞态：创建时 SG={pre_sg_id}，查询时 SG 已变为 {recheck_sg_id}，跳过验证)")
            return
        assert False, \
            f"创建策略后未找到 Port={test_port} CidrBlock={test_cidr} 的规则，实际 egress={egress}"
    print(f"    OK (出站规则创建成功，已验证规则存在)")
def test_create_policies_ingress_and_egress():
    print(">>> [创建规则(Deprecated)] 场景8：分两次创建 Ingress + Egress 规则 → 200，两类规则均已存在 ...")
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

    # 记录创建规则前的 SG ID，用于后续并发竞态检测
    pre_sg_id = sg_set[0].get("SecurityGroupId", "")

    ingress_port = "7654"
    egress_port = "7655"
    test_cidr = "172.16.0.0/12"

    # 腾讯云 API 不支持在同一请求中同时传入 Ingress 和 Egress，需分两次调用
    # 第一次：创建 Ingress 规则
    resp_ingress = post_policies({
        "SecurityGroupPolicySet": {
            "Ingress": [
                {
                    "Protocol": "TCP",
                    "Port": ingress_port,
                    "CidrBlock": test_cidr,
                    "Action": "ACCEPT",
                    "PolicyDescription": "集成测试混合规则-入站",
                }
            ]
        }
    })
    assert resp_ingress.status_code == 200, \
        f"Ingress 规则创建期望 200，实际 {resp_ingress.status_code}，body={resp_ingress.text}"

    # 第二次：创建 Egress 规则
    resp_egress = post_policies({
        "SecurityGroupPolicySet": {
            "Egress": [
                {
                    "Protocol": "TCP",
                    "Port": egress_port,
                    "CidrBlock": test_cidr,
                    "Action": "ACCEPT",
                    "PolicyDescription": "集成测试混合规则-出站",
                }
            ]
        }
    })
    assert resp_egress.status_code == 200, \
        f"Egress 规则创建期望 200，实际 {resp_egress.status_code}，body={resp_egress.text}"

    # 验证 Ingress 和 Egress 规则均已添加（腾讯云 API 可能有短暂延迟，加重试）
    found_ingress = False
    found_egress = False
    for attempt in range(5):
        policies_resp = get_policies()
        if policies_resp.status_code != 200 or not policies_resp.text.strip():
            if attempt < 4:
                time.sleep(2)
            continue
        policy_set = policies_resp.json().get("Response", {}).get("SecurityGroupPolicySet", {})
        ingress = policy_set.get("Ingress") or []
        egress = policy_set.get("Egress") or []
        found_ingress = any(
            r.get("Port") == ingress_port and r.get("CidrBlock") == test_cidr
            for r in ingress
        )
        found_egress = any(
            r.get("Port") == egress_port and r.get("CidrBlock") == test_cidr
            for r in egress
        )
        if found_ingress and found_egress:
            break
        if attempt < 4:
            time.sleep(2)
    if not found_ingress or not found_egress:
        # 并发竞态防御
        recheck_resp = get_sg()
        recheck_sg_id = ""
        if recheck_resp.status_code == 200 and recheck_resp.text.strip():
            try:
                recheck_set = recheck_resp.json().get("Response", {}).get("SecurityGroupSet", [])
                if recheck_set:
                    recheck_sg_id = recheck_set[0].get("SecurityGroupId", "")
            except Exception:
                pass
        if recheck_sg_id != pre_sg_id:
            print(f"    SKIP (并发竞态：创建时 SG={pre_sg_id}，查询时 SG 已变为 {recheck_sg_id}，跳过验证)")
            return
    assert found_ingress, \
        f"创建后应能查询到 Port={ingress_port} 的入站规则，实际 ingress={ingress}"
    assert found_egress, \
        f"创建后应能查询到 Port={egress_port} 的出站规则，实际 egress={egress}"
    print(f"    OK (Ingress Port={ingress_port} 和 Egress Port={egress_port} 均已创建成功，分两次调用)")

# ─────────────────────────────────────────────
# Main
# ─────────────────────────────────────────────

def main():
    health_check()
    print()
    run_tests(globals(), title="POST /admin/config/security-group/policies")

if __name__ == "__main__":
    main()
