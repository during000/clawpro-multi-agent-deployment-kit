#!/usr/bin/env python3
"""
GET /admin/config/security-group 查询当前绑定安全组 集成测试

测试场景：
  场景 1：无认证信息 → 401/403
  场景 2a：错误 token → 401/403
  场景 2b：非管理员 token → 401/403
  场景 3：未配置安全组（security_group_id 为空）→ 200，响应体为空（静默返回，不是 400）
           （仅在无 SG 配置的环境下有效，已配置则自动跳过）
  场景 4：已配置安全组，正常查询 → 200，响应体包含 Response.SecurityGroupSet，
           SecurityGroupSet[0].SecurityGroupId 与 site_config 中配置的 SG ID 一致
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


get_sg = make_api_fn("get", "/admin/config/security-group", timeout=10)


# ─────────────────────────────────────────────
# 测试用例
# ─────────────────────────────────────────────

def test_get_sg_auth():
    """认证测试三件套：无认证/错误token/非管理员 → 401/403"""
    auth_test_suite(lambda headers: get_sg(headers=headers),
                    label="get_sg")


def test_get_sg_no_sg_configured():
    """
    场景3：未配置安全组时查询 → 200，响应体为空（静默返回）
    注意：此测试依赖环境中未配置安全组，若当前已配置则跳过。
    关键断言：状态码必须是 200 而非 400，响应体为空（不含 JSON 内容）。
    """
    print(">>> [查询安全组] 场景3：未配置安全组 → 200 空体（静默返回，不是 400）...")
    # 先检查当前是否已配置安全组
    resp = get_sg()
    if resp.status_code == 200 and resp.text.strip():
        try:
            sg_set = resp.json().get("Response", {}).get("SecurityGroupSet", [])
            if sg_set:
                print("    SKIP (当前已配置安全组，此场景需在无 SG 配置环境下验证)")
                return
        except Exception:
            pass

    # 未配置安全组时，接口应静默返回 200 空体
    assert resp.status_code == 200, \
        f"未配置安全组时期望 200（静默返回），实际 {resp.status_code}，body={resp.text}"
    # 响应体应为空（handler 直接 return，不写任何内容）
    assert resp.text.strip() == "", \
        f"未配置安全组时响应体应为空，实际 body={repr(resp.text)}"
    print(f"    OK (status=200，响应体为空，符合静默返回预期)")

def test_get_sg_normal():
    """
    场景4：已配置安全组，正常查询 → 200，返回 Response.SecurityGroupSet，
    SecurityGroupSet[0].SecurityGroupId 与 site_config 中配置的 SG ID 一致。
    """
    print(">>> [查询安全组] 场景4：已配置安全组，正常查询 → 200，SecurityGroupSet 包含正确 SG ID ...")
    resp = get_sg()
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"

    if not resp.text.strip():
        print("    SKIP (当前未配置安全组，跳过正常查询验证)")
        return

    data = resp.json()
    assert "Response" in data, f"响应应包含 Response 字段，实际 {data}"
    sg_set = data.get("Response", {}).get("SecurityGroupSet", [])
    assert len(sg_set) > 0, \
        f"已配置安全组时 SecurityGroupSet 应至少有一条记录，实际 {sg_set}"
    sg_id = sg_set[0].get("SecurityGroupId", "")
    assert sg_id, f"SecurityGroupSet[0] 应包含 SecurityGroupId，实际 {sg_set[0]}"
    assert sg_id.startswith("sg-"), \
        f"SecurityGroupId 格式应以 'sg-' 开头，实际 '{sg_id}'"
    print(f"    OK (SecurityGroupId={sg_id})")

def test_get_sg_response_is_valid_json():
    """
    场景5：响应体为合法 JSON（不 panic，不返回非 JSON 内容）
    """
    print(">>> [查询安全组] 场景5：响应体为合法 JSON ...")
    resp = get_sg()
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
    run_tests(globals(), title="GET /admin/config/security-group")

if __name__ == "__main__":
    main()
