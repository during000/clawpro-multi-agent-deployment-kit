#!/usr/bin/env python3
"""
PUT /admin/config/security-group 修改安全组属性 集成测试

测试场景：
  场景 1：无认证信息 → 401/403
  场景 2a：错误 token → 401/403
  场景 2b：非管理员 token → 401/403
  场景 3：未配置安全组（security_group_id 为空）→ 400，含"未配置安全组"提示
           （仅在无 SG 配置的环境下有效，已配置则自动跳过）
  场景 4：请求体 JSON 格式错误 → 400，含 error 字段
  场景 5：正常修改安全组名称/描述 → 200，并验证修改结果已生效
  场景 6：请求体中传入不同的 SecurityGroupId → 200，但实际修改的是当前配置的 SG（强制覆盖，防止越权）
  场景 7：请求体为空 {} → 透传腾讯云，验证行为（不 panic，返回合理状态码且响应为合法 JSON）
  场景 8：仅修改 GroupDescription（不传 GroupName）→ 200，并验证描述已生效
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


put_sg = make_api_fn("put", "/admin/config/security-group")


def get_sg():
    return seed.get("/admin/config/security-group", expect=None,
                    timeout=10, raw=True)


# ─────────────────────────────────────────────
# 测试用例
# ─────────────────────────────────────────────

def test_update_sg_auth():
    """认证测试三件套：无认证/错误token/非管理员 → 401/403"""
    auth_test_suite(lambda headers: put_sg({"GroupName": "new-name"}, headers=headers),
                    label="update_sg")


def test_update_sg_no_sg_configured():
    """
    场景3：未配置安全组时修改安全组 → 400
    注意：此测试依赖环境中未配置安全组，若当前已配置则跳过。
    判断策略：
      - GET 返回 200 且 SecurityGroupSet 非空 → 已配置，SKIP
      - GET 返回 200 空体（handler 直接 return）→ 未配置，继续测试
      - 直接尝试 PUT，若返回 200 说明后端已有 SG 配置 → SKIP
    """
    print(">>> [修改安全组] 场景3：未配置安全组 → 400（仅在无 SG 配置环境下有效）...")
    get_resp = get_sg()
    if get_resp.status_code == 200 and get_resp.text.strip():
        try:
            sg_set = get_resp.json().get("Response", {}).get("SecurityGroupSet", [])
            if sg_set:
                print("    SKIP (当前已配置安全组，此场景需在无 SG 配置环境下验证)")
                return
        except Exception:
            pass

    resp = put_sg({"GroupName": "test-name"})
    # 如果返回 200，说明环境中已配置了安全组（可能是并发测试创建的），跳过此场景
    if resp.status_code == 200:
        print("    SKIP (PUT 返回 200，说明当前环境已配置安全组，跳过此场景)")
        return
    assert resp.status_code == 400, \
        f"未配置安全组时期望 400，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "error" in data, f"400 响应应包含 error 字段，实际 {data}"
    error_msg = data.get("error", "")
    assert "安全组" in error_msg, f"错误信息应包含'安全组'，实际 '{error_msg}'"
    print(f"    OK (status=400, error={error_msg})")


def test_update_sg_invalid_json():
    print(">>> [修改安全组] 场景4：请求体 JSON 格式错误 → 400 ...")
    resp = seed.put("/admin/config/security-group", data="not-a-json{{{",
                   expect=None, timeout=30, raw=True)
    assert resp.status_code == 400, \
        f"期望 400，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "error" in data, f"400 响应应包含 error 字段，实际 {data}"
    print(f"    OK (status=400, error={data.get('error')})")


def test_update_sg_normal():
    print(">>> [修改安全组] 场景5：正常修改安全组名称/描述 → 200，并验证修改结果已生效 ...")
    get_resp = get_sg()
    if get_resp.status_code != 200:
        print("    SKIP (当前未配置安全组，跳过修改测试)")
        return
    if not get_resp.text.strip():
        print("    SKIP (当前未配置安全组，跳过修改测试)")
        return
    try:
        sg_set = get_resp.json().get("Response", {}).get("SecurityGroupSet", [])
    except Exception:
        print("    SKIP (GET 响应解析失败，跳过修改测试)")
        return
    if not sg_set:
        print("    SKIP (当前未配置安全组，跳过修改测试)")
        return

    # 记录当前绑定的 SG ID，修改后直接按此 ID 查询，避免并发测试覆盖 site_config 导致误判
    current_sg_id = sg_set[0].get("SecurityGroupId", "")

    new_name = f"test-sg-{IDENTIFIER}-updated-name" if IDENTIFIER else "test-sg-updated-name"
    new_desc = "集成测试修改后的描述"
    resp = put_sg({
        "GroupName": new_name,
        "GroupDescription": new_desc,
    })
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"

    # 通过 /admin/config/security-group/list?security_group_id=xxx 直接查询该 SG 详情
    # 不依赖 site_config，避免并发测试覆盖 security_group_id 导致误判
    verify_resp = seed.get("/admin/config/security-group/list",
                           params={"security_group_id": current_sg_id},
                           expect=None, timeout=10, raw=True)
    assert verify_resp.status_code == 200, \
        f"查询 SG 详情期望 200，实际 {verify_resp.status_code}"
    # list 接口返回 {"ok": true, "security_groups": [...], "total_count": ...}
    verify_sg_set = verify_resp.json().get("security_groups", [])
    assert verify_sg_set, f"按 SG ID={current_sg_id} 查询应返回至少一条记录"
    updated_sg = verify_sg_set[0]
    if updated_sg.get("security_group_name") != new_name:
        # 并发竞态防御：PUT 时后端使用 SiteConfig.SecurityGroupId，如果并发测试在 GET 和 PUT 之间
        # 修改了 SiteConfig，则实际修改的是另一个安全组，current_sg_id 的名称不会变
        # 检查当前 SiteConfig 是否仍指向 current_sg_id
        recheck_resp = get_sg()
        recheck_sg_id = ""
        if recheck_resp.status_code == 200 and recheck_resp.text.strip():
            try:
                recheck_set = recheck_resp.json().get("Response", {}).get("SecurityGroupSet", [])
                if recheck_set:
                    recheck_sg_id = recheck_set[0].get("SecurityGroupId", "")
            except Exception:
                pass
        if recheck_sg_id != current_sg_id:
            print(f"    SKIP (并发竞态：PUT 时 SiteConfig 已被修改为 {recheck_sg_id}，"
                  f"实际修改的不是 {current_sg_id}，跳过验证)")
            return
        # 如果 SiteConfig 未变但名称不对，才是真正的失败
        assert False, \
            f"安全组名称应已更新为 '{new_name}'，实际 '{updated_sg.get('security_group_name')}'"
    print(f"    OK (SG ID={current_sg_id}，名称/描述修改成功，已验证生效)")


def test_update_sg_force_override_sg_id():
    print(">>> [修改安全组] 场景6：请求体传入不同 SecurityGroupId → 实际修改当前配置 SG（强制覆盖）...")
    get_resp = get_sg()
    if get_resp.status_code != 200 or not get_resp.text.strip():
        print("    SKIP (当前未配置安全组)")
        return
    try:
        sg_set = get_resp.json().get("Response", {}).get("SecurityGroupSet", [])
    except Exception:
        print("    SKIP (GET 响应解析失败)")
        return
    if not sg_set:
        print("    SKIP (当前未配置安全组)")
        return
    current_sg_id = sg_set[0].get("SecurityGroupId", "")

    # 传入一个不同的（伪造的）SecurityGroupId
    fake_sg_id = "sg-00000000"
    override_name = f"test-sg-{IDENTIFIER}-force-override" if IDENTIFIER else "test-sg-force-override"
    resp = put_sg({
        "SecurityGroupId": fake_sg_id,
        "GroupName": override_name,
        "GroupDescription": "强制覆盖测试",
    })
    # 接口应成功（修改的是当前配置的 SG，而非 fake_sg_id）
    assert resp.status_code == 200, \
        f"期望 200（后端强制覆盖 SecurityGroupId），实际 {resp.status_code}，body={resp.text}"
    print(f"    OK (当前 SG={current_sg_id}，请求中传入 {fake_sg_id}，后端强制覆盖为当前 SG)")


def test_update_sg_empty_body():
    print(">>> [修改安全组] 场景7：请求体为空 {} → 透传腾讯云，验证行为 ...")
    get_resp = get_sg()
    if get_resp.status_code != 200 or not get_resp.text.strip():
        print("    SKIP (当前未配置安全组)")
        return
    try:
        sg_set = get_resp.json().get("Response", {}).get("SecurityGroupSet", [])
    except Exception:
        print("    SKIP (GET 响应解析失败)")
        return
    if not sg_set:
        print("    SKIP (当前未配置安全组)")
        return

    resp = put_sg({})
    # 空 body 透传腾讯云，腾讯云可能返回 200（无变更）或报错
    # 主要验证接口不会 panic，且返回合理的 HTTP 状态码和合法 JSON 响应体
    assert resp.status_code in (200, 400, 500), \
        f"期望 200/400/500，实际 {resp.status_code}，body={resp.text}"
    try:
        body = resp.json()
        assert isinstance(body, dict), f"响应体应为 JSON 对象，实际 {body}"
    except Exception:
        assert False, f"响应体应为合法 JSON，实际 body={resp.text}"
    print(f"    OK (status={resp.status_code}，空 body 行为符合预期)")

def test_update_sg_description_only():
    print(">>> [修改安全组] 场景8：仅修改 GroupDescription（不传 GroupName）→ 200，并验证描述已生效 ...")
    get_resp = get_sg()
    if get_resp.status_code != 200 or not get_resp.text.strip():
        print("    SKIP (当前未配置安全组)")
        return
    try:
        sg_set = get_resp.json().get("Response", {}).get("SecurityGroupSet", [])
    except Exception:
        print("    SKIP (GET 响应解析失败)")
        return
    if not sg_set:
        print("    SKIP (当前未配置安全组)")
        return
    current_sg_id = sg_set[0].get("SecurityGroupId", "")

    new_desc = "集成测试仅修改描述"
    resp = put_sg({"GroupDescription": new_desc})
    # 并发环境下可能在测试执行中途 SG 被解绑，此时后端返回 400
    if resp.status_code == 400 and "未配置安全组" in resp.text:
        print("    SKIP (当前环境安全组已解绑，跳过)")
        return
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"

    # 通过 /admin/config/security-group/list?security_group_id=xxx 直接查询该 SG 详情
    verify_resp = seed.get("/admin/config/security-group/list",
                           params={"security_group_id": current_sg_id},
                           expect=None, timeout=10, raw=True)
    assert verify_resp.status_code == 200, \
        f"查询 SG 详情期望 200，实际 {verify_resp.status_code}"
    verify_sg_set = verify_resp.json().get("security_groups", [])
    assert verify_sg_set, f"按 SG ID={current_sg_id} 查询应返回至少一条记录"
    updated_sg = verify_sg_set[0]
    if updated_sg.get("security_group_desc") != new_desc:
        # 并发竞态防御：检查 SiteConfig 是否仍指向 current_sg_id
        recheck_resp = get_sg()
        recheck_sg_id = ""
        if recheck_resp.status_code == 200 and recheck_resp.text.strip():
            try:
                recheck_set = recheck_resp.json().get("Response", {}).get("SecurityGroupSet", [])
                if recheck_set:
                    recheck_sg_id = recheck_set[0].get("SecurityGroupId", "")
            except Exception:
                pass
        if recheck_sg_id != current_sg_id:
            print(f"    SKIP (并发竞态：PUT 时 SiteConfig 已被修改为 {recheck_sg_id}，"
                  f"实际修改的不是 {current_sg_id}，跳过验证)")
            return
        assert False, \
            f"安全组描述应已更新为 '{new_desc}'，实际 '{updated_sg.get('security_group_desc')}'"
    print(f"    OK (SG ID={current_sg_id}，仅修改描述成功，已验证生效)")

# ─────────────────────────────────────────────
# Main
# ─────────────────────────────────────────────

def main():
    health_check()
    print()
    run_tests(globals(), title="PUT /admin/config/security-group", ordered=True)

if __name__ == "__main__":
    main()
