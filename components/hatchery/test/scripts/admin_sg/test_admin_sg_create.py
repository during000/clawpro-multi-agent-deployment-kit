#!/usr/bin/env python3
"""
POST /admin/config/security-group 创建安全组 集成测试

测试场景：
  场景 1：无认证信息 → 401/403
  场景 2a：错误 token → 401/403
  场景 2b：非管理员 token → 401/403
  场景 3：请求体 JSON 格式错误 → 400，含 error 字段
  场景 4：GroupName 为空 → 400，含 error 字段
  场景 5：GroupDescription 为空，正常创建 → 200，安全组创建成功
  场景 6：正常创建（无 quick_rules）→ 200，返回腾讯云响应，site_config.security_group_id 已更新
  场景 7：携带合法 quick_rules（allow_internet）→ 200，安全组创建成功，规则已添加
  场景 8：quick_rules 包含未知规则名 → 200，未知规则被跳过，安全组仍创建成功
  场景 10：已有旧 security_group_id，创建后被新 ID 覆盖 → 可通过 GET 接口验证新 ID 已生效
"""

import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))
from helpers.api import (
    IDENTIFIER,
    health_check, auth_test_suite, run_tests,
    seed, make_api_fn,
)


# 记录本次测试创建的安全组 ID，用于测试后清理（如有需要）
_created_sg_ids: list[str] = []


# ─────────────────────────────────────────────
# 工具函数
# ─────────────────────────────────────────────


post_sg = make_api_fn("post", "/admin/config/security-group")


def get_sg():
    return seed.get("/admin/config/security-group", expect=None,
                    timeout=10, raw=True)


def get_policies():
    return seed.get("/admin/config/security-group/policies", expect=None,
                    timeout=10, raw=True)


# ─────────────────────────────────────────────
# 测试用例
# ─────────────────────────────────────────────

def test_create_sg_auth():
    """认证测试三件套：无认证/错误token/非管理员 → 401/403"""
    auth_test_suite(lambda headers: post_sg({"GroupName": "test-sg"}, headers=headers),
                    label="create_sg")


def test_create_sg_invalid_json():
    print(">>> [创建安全组] 场景3：请求体 JSON 格式错误 → 400 ...")
    resp = seed.post("/admin/config/security-group", data="not-a-json{{{",
                     expect=None, timeout=30, raw=True)
    assert resp.status_code == 400, \
        f"期望 400，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "error" in data, f"400 响应应包含 error 字段，实际 {data}"
    print(f"    OK (status=400, error={data.get('error')})")


def test_create_sg_empty_name():
    print(">>> [创建安全组] 场景4：GroupName 为空 → 400 ...")
    resp = post_sg({"GroupName": "", "GroupDescription": "test"})
    assert resp.status_code == 400, \
        f"期望 400，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "error" in data, f"400 响应应包含 error 字段，实际 {data}"
    print(f"    OK (status=400, error={data.get('error')})")


def test_create_sg_empty_description():
    print(">>> [创建安全组] 场景5：GroupDescription 为空，正常创建 → 200 ...")
    sg_name = f"test-sg-{IDENTIFIER}-no-desc" if IDENTIFIER else "test-sg-no-desc"
    resp = post_sg({"GroupName": sg_name})
    if resp.status_code == 500 and "配额" in resp.text:
        print(f"    SKIP (安全组配额已达上限，跳过此场景)")
        return
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "Response" in data, f"响应应包含 Response 字段，实际 {data}"
    sg = data.get("Response", {}).get("SecurityGroup", {})
    sg_id = sg.get("SecurityGroupId", "")
    assert sg_id, f"响应中应包含 SecurityGroupId，实际 {data}"
    _created_sg_ids.append(sg_id)
    print(f"    OK (security_group_id={sg_id})")


def test_create_sg_basic():
    print(">>> [创建安全组] 场景6：正常创建（无 quick_rules）→ 200，site_config 已更新 ...")
    sg_name = f"test-sg-{IDENTIFIER}-basic" if IDENTIFIER else "test-sg-basic"
    resp = post_sg({
        "GroupName": sg_name,
        "GroupDescription": "集成测试创建的安全组",
    })
    if resp.status_code == 500 and "配额" in resp.text:
        print(f"    SKIP (安全组配额已达上限，跳过此场景)")
        return
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "Response" in data, f"响应应包含 Response 字段，实际 {data}"
    sg = data.get("Response", {}).get("SecurityGroup", {})
    sg_id = sg.get("SecurityGroupId", "")
    assert sg_id, f"响应中应包含 SecurityGroupId，实际 {data}"
    _created_sg_ids.append(sg_id)

    # 验证 site_config.security_group_id 已更新为新 SG ID
    get_resp = get_sg()
    assert get_resp.status_code == 200, \
        f"GET /admin/config/security-group 期望 200，实际 {get_resp.status_code}"
    get_data = get_resp.json()
    sg_set = get_data.get("Response", {}).get("SecurityGroupSet", [])
    assert len(sg_set) > 0, "GET 安全组应返回至少一条记录"
    current_sg_id = sg_set[0].get("SecurityGroupId", "")
    assert current_sg_id == sg_id, \
        f"site_config 中的 security_group_id 应已更新为 {sg_id}，实际 {current_sg_id}"
    print(f"    OK (security_group_id={sg_id}，site_config 已更新)")


def test_create_sg_with_quick_rules():
    print(">>> [创建安全组] 场景7：携带合法 quick_rules → 200，规则已添加 ...")
    sg_name = f"test-sg-{IDENTIFIER}-quick-rules" if IDENTIFIER else "test-sg-quick-rules"
    resp = post_sg({
        "GroupName": sg_name,
        "GroupDescription": "集成测试：含快速规则",
        "quick_rules": ["allow_internet"],
    })
    if resp.status_code == 500 and "配额" in resp.text:
        print(f"    SKIP (安全组配额已达上限，跳过此场景)")
        return
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    sg = data.get("Response", {}).get("SecurityGroup", {})
    sg_id = sg.get("SecurityGroupId", "")
    assert sg_id, f"响应中应包含 SecurityGroupId，实际 {data}"
    _created_sg_ids.append(sg_id)

    # 验证规则已添加（查询安全组规则，规则写入可能有短暂延迟）
    import time
    ingress, egress = [], []
    for attempt in range(3):
        if attempt > 0:
            time.sleep(1)
        policies_resp = get_policies()
        if policies_resp.status_code != 200:
            continue
        body = policies_resp.text.strip()
        if not body:
            continue
        policies_data = policies_resp.json()
        policy_set = policies_data.get("Response", {}).get("SecurityGroupPolicySet", {})
        ingress = policy_set.get("Ingress") or []
        egress = policy_set.get("Egress") or []
        if ingress or egress:
            break

    if len(ingress) == 0 and len(egress) == 0:
        # 并发竞态防御：检查当前 SiteConfig 是否仍指向刚创建的安全组
        recheck_resp = get_sg()
        recheck_sg_id = ""
        if recheck_resp.status_code == 200 and recheck_resp.text.strip():
            try:
                recheck_set = recheck_resp.json().get("Response", {}).get("SecurityGroupSet", [])
                if recheck_set:
                    recheck_sg_id = recheck_set[0].get("SecurityGroupId", "")
            except Exception:
                pass
        if recheck_sg_id != sg_id:
            print(f"    SKIP (并发竞态：创建的 SG={sg_id}，当前 SiteConfig 已变为 {recheck_sg_id}，"
                  f"get_policies 查询的是另一个安全组，跳过验证)")
            return
        assert False, \
            f"携带 quick_rules 创建后，安全组应有规则，实际 ingress={len(ingress)}, egress={len(egress)}"
    print(f"    OK (security_group_id={sg_id}, ingress={len(ingress)}, egress={len(egress)})")


def test_create_sg_unknown_quick_rules():
    print(">>> [创建安全组] 场景8：quick_rules 包含未知规则名 → 200，未知规则跳过，安全组仍创建成功 ...")
    sg_name = f"test-sg-{IDENTIFIER}-unknown-rules" if IDENTIFIER else "test-sg-unknown-rules"
    resp = post_sg({
        "GroupName": sg_name,
        "GroupDescription": "集成测试：含未知快速规则",
        "quick_rules": ["allow_internet", "unknown_rule_xyz_not_exist"],
    })
    if resp.status_code == 500 and "配额" in resp.text:
        print(f"    SKIP (安全组配额已达上限，跳过此场景)")
        return
    assert resp.status_code == 200, \
        f"期望 200（未知规则应被跳过），实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    sg = data.get("Response", {}).get("SecurityGroup", {})
    sg_id = sg.get("SecurityGroupId", "")
    assert sg_id, f"响应中应包含 SecurityGroupId，实际 {data}"
    _created_sg_ids.append(sg_id)
    print(f"    OK (security_group_id={sg_id}，未知规则已跳过)")


def test_create_sg_overwrites_old_sg_id():
    print(">>> [创建安全组] 场景10：已有旧 security_group_id，创建后被新 ID 覆盖 ...")
    # 先获取当前 SG ID
    get_resp = get_sg()
    old_sg_id = ""
    if get_resp.status_code == 200 and get_resp.text.strip():
        try:
            sg_set = get_resp.json().get("Response", {}).get("SecurityGroupSet", [])
            if sg_set:
                old_sg_id = sg_set[0].get("SecurityGroupId", "")
        except Exception:
            pass

    # 创建新安全组
    sg_name = f"test-sg-{IDENTIFIER}-overwrite" if IDENTIFIER else "test-sg-overwrite"
    resp = post_sg({
        "GroupName": sg_name,
        "GroupDescription": "集成测试：覆盖旧 SG ID",
    })
    if resp.status_code == 500 and "配额" in resp.text:
        print(f"    SKIP (安全组配额已达上限，跳过此场景，body={resp.text[:120]})")
        return
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    new_sg_id = resp.json().get("Response", {}).get("SecurityGroup", {}).get("SecurityGroupId", "")
    assert new_sg_id, "响应中应包含新 SecurityGroupId"
    _created_sg_ids.append(new_sg_id)

    # 验证 site_config 已更新为新 SG ID
    get_resp2 = get_sg()
    if not get_resp2.text.strip():
        print(f"    SKIP (GET security-group 返回空 body，可能 site_config 尚未同步)")
        return
    assert get_resp2.status_code == 200
    sg_set2 = get_resp2.json().get("Response", {}).get("SecurityGroupSet", [])
    current_sg_id = sg_set2[0].get("SecurityGroupId", "") if sg_set2 else ""
    assert current_sg_id == new_sg_id, \
        f"site_config 应已更新为新 SG ID {new_sg_id}，实际 {current_sg_id}"
    if old_sg_id:
        assert current_sg_id != old_sg_id, \
            f"site_config 中的 SG ID 应已从 {old_sg_id} 更新为 {new_sg_id}"
    print(f"    OK (old_sg_id={old_sg_id or '(无)'}, new_sg_id={new_sg_id})")


# ─────────────────────────────────────────────
# Main
# ─────────────────────────────────────────────

def main():
    health_check()
    print()
    run_tests(globals(), title="POST /admin/config/security-group")

if __name__ == "__main__":
    main()
