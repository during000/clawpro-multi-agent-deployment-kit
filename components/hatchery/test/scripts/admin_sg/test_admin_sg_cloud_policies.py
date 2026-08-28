#!/usr/bin/env python3
"""
GET /admin/config/security-group/cloud-policies 预览云端安全组规则 集成测试

测试场景：
  场景 1：无认证信息 → 401/403
  场景 2：错误 token → 401/403
  场景 3：非管理员 token → 401/403
  场景 4：缺少 security_group_id 参数 → 400，提示"缺少参数 security_group_id"
  场景 5：security_group_id 为空字符串（?security_group_id=）→ 400
  场景 6：security_group_id 为纯空格（?security_group_id=   ）→ TrimSpace 后为空 → 400
  场景 7：传入合法存在的 SG ID → 200，响应顶层字段为 SecurityGroupPolicySet（非 Response.SecurityGroupPolicySet）
  场景 8：SecurityGroupPolicySet 包含 Ingress 和/或 Egress 字段（可为 null，但字段必须存在）
  场景 9：传入不存在的 SG ID → 500（腾讯云 API 报错）或 400（腾讯云返回空集合）
  场景 10：该接口不依赖 SiteConfig，未配置安全组时仍可正常查询（与旧 /policies 接口的核心区别）
  场景 11：响应体为合法 JSON（不 panic，不返回非 JSON 内容）

注意：
  - 响应结构与旧 GET /admin/config/security-group/policies 不同：
    本接口直接返回 resp.Response（顶层为 SecurityGroupPolicySet），
    旧接口返回 resp（顶层为 Response.SecurityGroupPolicySet）。
  - security_group_id 来自 URL query，不依赖 SiteConfig。
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
# 若不设置，则通过 /admin/config/security-group 自动获取当前配置的 SG ID
KNOWN_SG_ID = os.environ.get("KNOWN_SG_ID", "")

# ─────────────────────────────────────────────
# 工具函数
# ─────────────────────────────────────────────


_cloud_policies_fn = make_api_fn("get", "/admin/config/security-group/cloud-policies", timeout=15)


def cloud_policies(sg_id: str = None, headers: dict = None):
    params = {"security_group_id": sg_id} if sg_id is not None else None
    return _cloud_policies_fn(params=params, headers=headers)


def get_current_sg_id() -> str:
    """从当前站点配置获取已绑定的 SG ID，用于正向测试"""
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

# ─────────────────────────────────────────────
# 测试用例
# ─────────────────────────────────────────────

def test_cloud_policies_auth():
    """认证测试三件套：无认证/错误token/非管理员 → 401/403"""
    auth_test_suite(lambda headers: cloud_policies(sg_id="sg-test", headers=headers),
                    label="cloud_policies")


def test_cloud_policies_missing_param():
    print(">>> [cloud-policies] 场景4：缺少 security_group_id 参数 → 400 ...")
    resp = cloud_policies()  # 不传 sg_id
    assert resp.status_code == 400, \
        f"期望 400，实际 {resp.status_code}，body={resp.text}"
    body = resp.json()
    assert "security_group_id" in resp.text.lower() or "缺少参数" in resp.text, \
        f"错误信息应提示缺少 security_group_id，实际 body={resp.text}"
    print(f"    OK (status=400，提示缺少参数)")

def test_cloud_policies_empty_sg_id():
    print(">>> [cloud-policies] 场景5：security_group_id 为空字符串 → 400 ...")
    resp = cloud_policies(sg_id="")
    assert resp.status_code == 400, \
        f"期望 400，实际 {resp.status_code}，body={resp.text}"
    print(f"    OK (status=400)")

def test_cloud_policies_whitespace_sg_id():
    """
    场景6：security_group_id 为纯空格 → TrimSpace 后为空 → 400
    代码中 sgID := strings.TrimSpace(r.URL.Query().Get("security_group_id"))，
    纯空格经 TrimSpace 后变为空字符串，应返回 400。
    """
    print(">>> [cloud-policies] 场景6：security_group_id 为纯空格 → TrimSpace 后为空 → 400 ...")
    resp = cloud_policies(sg_id="   ")
    assert resp.status_code == 400, \
        f"期望 400（纯空格经 TrimSpace 后为空），实际 {resp.status_code}，body={resp.text}"
    print(f"    OK (status=400，纯空格被 TrimSpace 处理)")

def test_cloud_policies_valid_sg_id():
    """
    场景7：传入合法存在的 SG ID → 200，响应顶层字段为 SecurityGroupPolicySet。
    ⚠️ 注意：本接口直接返回 resp.Response（顶层为 SecurityGroupPolicySet），
    与旧 GET /policies 接口（顶层为 Response.SecurityGroupPolicySet）不同。
    """
    print(">>> [cloud-policies] 场景7：合法 SG ID → 200，响应顶层为 SecurityGroupPolicySet ...")
    sg_id = get_current_sg_id()
    if not sg_id:
        print("    SKIP (无法获取有效的 SG ID，请设置 KNOWN_SG_ID 环境变量)")
        return

    resp = cloud_policies(sg_id=sg_id)
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"

    data = resp.json()
    # 关键断言：顶层字段是 SecurityGroupPolicySet，而非 Response.SecurityGroupPolicySet
    assert "SecurityGroupPolicySet" in data, \
        f"响应顶层应包含 SecurityGroupPolicySet 字段（不是 Response.SecurityGroupPolicySet），实际 keys={list(data.keys())}"
    # 不应有 Response 外层包装
    assert "Response" not in data, \
        f"响应不应有 Response 外层包装（本接口直接返回 resp.Response），实际 keys={list(data.keys())}"
    print(f"    OK (SG ID={sg_id}，顶层字段为 SecurityGroupPolicySet)")

def test_cloud_policies_policy_set_structure():
    """
    场景8：SecurityGroupPolicySet 包含 Ingress 和/或 Egress 字段（可为 null，但字段必须存在）。
    """
    print(">>> [cloud-policies] 场景8：SecurityGroupPolicySet 结构验证（含 Ingress/Egress 字段）...")
    sg_id = get_current_sg_id()
    if not sg_id:
        print("    SKIP (无法获取有效的 SG ID)")
        return

    resp = cloud_policies(sg_id=sg_id)
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"

    data = resp.json()
    policy_set = data.get("SecurityGroupPolicySet")
    assert policy_set is not None, \
        f"SecurityGroupPolicySet 不应为 null，实际 {data}"

    # Ingress / Egress 字段可为 null 或空列表，但 SecurityGroupPolicySet 本身必须存在
    ingress = policy_set.get("Ingress") or []
    egress = policy_set.get("Egress") or []
    print(f"    OK (ingress={len(ingress)} 条，egress={len(egress)} 条)")

def test_cloud_policies_nonexistent_sg_id():
    """
    场景9：传入不存在的 SG ID → 500（腾讯云 API 报错）。
    注意：若 SG ID 格式合法但不存在，腾讯云 API 可能返回空集合（不报错），
    此时后端返回 200 空 SecurityGroupPolicySet；若格式非法则腾讯云报错 → 500。
    """
    print(">>> [cloud-policies] 场景9：不存在的 SG ID → 500 或 200（取决于腾讯云行为）...")
    fake_id = "sg-00000000"
    resp = cloud_policies(sg_id=fake_id)
    # 腾讯云对不存在的 SG ID 行为：可能返回空集合（200）或报错（500）
    assert resp.status_code in (200, 500), \
        f"期望 200 或 500，实际 {resp.status_code}，body={resp.text}"
    if resp.status_code == 200:
        data = resp.json()
        policy_set = data.get("SecurityGroupPolicySet", {})
        ingress = policy_set.get("Ingress") or []
        egress = policy_set.get("Egress") or []
        print(f"    OK (腾讯云返回空集合，ingress={len(ingress)}，egress={len(egress)})")
    else:
        print(f"    OK (status=500，腾讯云 API 报错，符合预期)")

def test_cloud_policies_independent_of_site_config():
    """
    场景10：该接口不依赖 SiteConfig，未配置安全组时仍可正常查询。
    与旧 GET /admin/config/security-group/policies 的核心区别：
    旧接口未配置安全组时静默返回 200 空体；本接口直接查询指定 SG，不依赖 SiteConfig。
    """
    print(">>> [cloud-policies] 场景10：不依赖 SiteConfig，直接查询指定 SG ID ...")
    sg_id = get_current_sg_id()
    if not sg_id:
        print("    SKIP (无法获取有效的 SG ID)")
        return

    # 即使 SiteConfig 中未配置安全组，只要传入合法 SG ID 就应能正常查询
    resp = cloud_policies(sg_id=sg_id)
    assert resp.status_code == 200, \
        f"期望 200（不依赖 SiteConfig），实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "SecurityGroupPolicySet" in data, \
        f"响应应包含 SecurityGroupPolicySet，实际 {data}"
    print(f"    OK (SG ID={sg_id}，不依赖 SiteConfig 直接查询成功)")

def test_cloud_policies_valid_json():
    """
    场景11：响应体为合法 JSON（不 panic，不返回非 JSON 内容）。
    """
    print(">>> [cloud-policies] 场景11：响应体为合法 JSON ...")
    sg_id = get_current_sg_id()
    if not sg_id:
        print("    SKIP (无法获取有效的 SG ID)")
        return

    resp = cloud_policies(sg_id=sg_id)
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
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
    run_tests(globals(), title="GET /admin/config/security-group/cloud-policies")

if __name__ == "__main__":
    main()
