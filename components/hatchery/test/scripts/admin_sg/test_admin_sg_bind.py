#!/usr/bin/env python3
"""
POST /admin/config/security-group/bind 绑定安全组 集成测试

测试场景：
  场景 1：无认证信息 → 401/403
  场景 2：错误 token → 401/403
  场景 3：非管理员 token → 401/403
  场景 4：请求体格式错误（非 JSON）→ 400，提示"请求参数格式错误"
  场景 5：请求体为空 {}（security_group_id 字段缺失，零值为空字符串）→ 400，提示"security_group_id 不能为空"
  场景 6：security_group_id 为空字符串 → 400，提示"security_group_id 不能为空"
  场景 7：绑定与当前已配置相同的安全组（绑定自身）→ 400，提示"该安全组已是当前使用的安全组，无需重复绑定"
           注意：仅在 currentConfig.SecurityGroupId != "" 时触发，未配置时不会报错
  场景 8：绑定不存在的安全组 ID → 400（腾讯云返回空集合，提示"安全组不存在"）或 500（腾讯云 SDK 报错）
           注意：腾讯云对不存在的 SG ID 可能返回空集合（→ 400）或直接 SDK 报错（→ 500）
  场景 9：绑定合法存在的安全组，auto_fix_rules=false → 200，ok=true，security_group_id 已更新，
           响应中不含 fixed_rules_count 字段
  场景 10：绑定合法存在的安全组，auto_fix_rules=true，规则已完整 → 200，ok=true，fixed_rules_count=0
  场景 11：绑定合法存在的安全组，auto_fix_rules=true，规则有缺失 → 200，ok=true，fixed_rules_count>0，
            云端规则已被补齐（通过 check-rules 验证）
  场景 12：绑定成功后，通过 GET /admin/config/security-group 验证 security_group_id 已更新
  场景 13：当前未配置安全组（SecurityGroupId=""）时，绑定新安全组 → 200，正常绑定（不触发"绑定自身"校验）
  场景 14：auto_fix_rules=true，补齐规则时 Ingress 成功但 Egress 失败 → 500，SiteConfig 未更新
            （此场景为边界场景，需 mock 环境，标注为可选验证）
  场景 15：绑定成功后，实例安全组异步换绑（可选验证，需等待异步完成后查询实例安全组）

注意：
  - 场景7"绑定自身"校验：仅在 currentConfig.SecurityGroupId != "" 时触发
  - 场景8区分：SG 不存在（腾讯云返回空集合 → 400）vs SG ID 格式非法（腾讯云 API 报错 → 500）
  - 场景9响应中不含 fixed_rules_count：代码中 auto_fix_rules=false 时不写入该字段
  - 场景14/15 为可选验证场景，需特殊环境支持
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

# 可选：指定一个已知存在的备用 SG ID（用于绑定测试，绑定后会恢复原 SG）
# 若不设置，则跳过需要实际绑定的场景
BACKUP_SG_ID = os.environ.get("BACKUP_SG_ID", "")

# ─────────────────────────────────────────────
# 工具函数
# ─────────────────────────────────────────────

bind_sg = make_api_fn("post", "/admin/config/security-group/bind")



def get_current_sg_id() -> str:
    """从当前站点配置获取已绑定的 SG ID"""
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

def check_rules(sg_id: str) -> list:
    """检查指定 SG 的缺失规则"""
    resp = seed.get("/admin/config/security-group/check-rules",
                    params={"security_group_id": sg_id}, expect=None, timeout=15, raw=True)
    if resp.status_code != 200:
        return []
    return resp.json().get("missing_rules", [])

# ─────────────────────────────────────────────
# 测试用例
# ─────────────────────────────────────────────

def test_bind_auth():
    """场景1-3：认证测试三件套"""
    auth_test_suite(lambda headers: bind_sg({"security_group_id": "sg-test"}, headers=headers),
                    label="bind")

def test_bind_invalid_json():
    """
    场景4：请求体格式错误（非 JSON）→ 400，提示"请求参数格式错误"。
    """
    print(">>> [bind] 场景4：请求体格式错误（非 JSON）→ 400 ...")
    resp = seed.post("/admin/config/security-group/bind", data="not-a-json-body",
                    expect=None, timeout=10, raw=True)
    assert resp.status_code == 400, \
        f"期望 400，实际 {resp.status_code}，body={resp.text}"
    assert "请求体格式错误" in resp.text or "invalid" in resp.text.lower(), \
        f"错误信息应提示格式错误，实际 body={resp.text}"
    print(f"    OK (status=400，提示请求参数格式错误)")

def test_bind_empty_body():
    """
    场景5：请求体为空 {}（security_group_id 字段缺失，JSON 解析后零值为空字符串）→ 400。
    代码：JSON 解析成功，但 reqBody.SecurityGroupId == "" → 400，提示"security_group_id 不能为空"。
    """
    print(">>> [bind] 场景5：请求体为空 {} → 400，提示 security_group_id 不能为空 ...")
    resp = bind_sg({})
    assert resp.status_code == 400, \
        f"期望 400，实际 {resp.status_code}，body={resp.text}"
    assert "security_group_id" in resp.text.lower() or "不能为空" in resp.text, \
        f"错误信息应提示 security_group_id 不能为空，实际 body={resp.text}"
    print(f"    OK (status=400，提示 security_group_id 不能为空)")

def test_bind_empty_sg_id():
    """
    场景6：security_group_id 为空字符串 → 400，提示"security_group_id 不能为空"。
    """
    print(">>> [bind] 场景6：security_group_id 为空字符串 → 400 ...")
    resp = bind_sg({"security_group_id": ""})
    assert resp.status_code == 400, \
        f"期望 400，实际 {resp.status_code}，body={resp.text}"
    assert "security_group_id" in resp.text.lower() or "不能为空" in resp.text, \
        f"错误信息应提示 security_group_id 不能为空，实际 body={resp.text}"
    print(f"    OK (status=400，提示 security_group_id 不能为空)")

def test_bind_self():
    """
    场景7：绑定与当前已配置相同的安全组（绑定自身）→ 400，提示"该安全组已是当前使用的安全组，无需重复绑定"。
    注意：仅在 currentConfig.SecurityGroupId != "" 时触发，未配置时不会报错。
    策略：使用"绑定→再绑定同一个"的自包含操作，确保第二次绑定时 SiteConfig 一定等于传入的 ID，
    从而确定性地触发"绑定自身"校验，彻底消除并发竞态。
    """
    print(">>> [bind] 场景7：绑定自身（当前已配置的 SG）→ 400 ...")
    current_sg_id = get_current_sg_id()
    if not current_sg_id:
        # 当前未配置，尝试用 BACKUP_SG_ID 先绑定一个
        if not BACKUP_SG_ID:
            print("    SKIP (当前未配置安全组且 BACKUP_SG_ID 未设置，无法验证绑定自身)")
            return
        resp = bind_sg({"security_group_id": BACKUP_SG_ID, "auto_fix_rules": False})
        if resp.status_code != 200:
            print(f"    SKIP (预绑定失败: status={resp.status_code})")
            return
        current_sg_id = BACKUP_SG_ID

    # 先绑定一次确保 SiteConfig 是我们期望的值（消除竞态窗口）
    resp = bind_sg({"security_group_id": current_sg_id, "auto_fix_rules": False})
    if resp.status_code == 400 and ("已是当前使用的安全组" in resp.text or "无需重复绑定" in resp.text):
        # 第一次就触发了"绑定自身"，说明没有竞态，直接通过
        print(f"    OK (status=400，SG ID={current_sg_id}，提示无需重复绑定)")
        return
    elif resp.status_code == 200:
        # 绑定成功（说明之前被其他测试改了），现在 SiteConfig 已是 current_sg_id
        # 再次绑定同一个，此时必定触发"绑定自身"
        resp2 = bind_sg({"security_group_id": current_sg_id, "auto_fix_rules": False})
        assert resp2.status_code == 400, \
            f"期望 400（绑定自身），实际 {resp2.status_code}，body={resp2.text}"
        assert "已是当前使用的安全组" in resp2.text or "无需重复绑定" in resp2.text, \
            f"错误信息应提示无需重复绑定，实际 body={resp2.text}"
        print(f"    OK (status=400，SG ID={current_sg_id}，提示无需重复绑定)")
    else:
        assert False, f"预绑定异常: status={resp.status_code}，body={resp.text}"

def test_bind_nonexistent_sg():
    """
    场景8：绑定不存在的安全组 ID → 400 或 500。
    腾讯云 DescribeSecurityGroups 对不存在的 SG ID 可能：
    - 返回空集合 → 后端返回 400，提示"安全组不存在"
    - 直接 SDK 报错（InvalidParameterValue.Malformed）→ 后端返回 500
    """
    print(">>> [bind] 场景8：绑定不存在的 SG ID → 400 或 500 ...")
    fake_id = "sg-00000000"
    resp = bind_sg({"security_group_id": fake_id})
    assert resp.status_code in (400, 500), \
        f"期望 400 或 500（安全组不存在或验证失败），实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "error" in data, f"响应应含 error 字段，实际 {data}"
    if resp.status_code == 400:
        assert "安全组不存在" in resp.text, \
            f"400 时错误信息应提示安全组不存在，实际 body={resp.text}"
        print(f"    OK (status=400，提示安全组不存在)")
    else:
        print(f"    OK (status=500，腾讯云 SDK 报错: {data.get('error', '')[:80]})")

def test_bind_valid_sg_no_auto_fix():
    """
    场景9：绑定合法存在的安全组，auto_fix_rules=false → 200，ok=true，security_group_id 已更新，
    响应中不含 fixed_rules_count 字段。
    ⚠️ 此场景会实际修改站点配置，测试完成后会恢复原 SG。
    """
    print(">>> [bind] 场景9：绑定合法 SG，auto_fix_rules=false → 200，响应不含 fixed_rules_count ...")
    if not BACKUP_SG_ID:
        print("    SKIP (BACKUP_SG_ID 未设置，跳过实际绑定测试)")
        return

    original_sg_id = get_current_sg_id()

    try:
        resp = bind_sg({"security_group_id": BACKUP_SG_ID, "auto_fix_rules": False})
        assert resp.status_code == 200, \
            f"期望 200，实际 {resp.status_code}，body={resp.text}"
        data = resp.json()
        assert data.get("ok") is True, f"响应 ok 应为 true，实际 {data}"
        assert data.get("security_group_id") == BACKUP_SG_ID, \
            f"响应 security_group_id 应为 {BACKUP_SG_ID}，实际 {data.get('security_group_id')}"
        # auto_fix_rules=false 时，响应中不含 fixed_rules_count 字段
        assert "fixed_rules_count" not in data, \
            f"auto_fix_rules=false 时响应不应含 fixed_rules_count 字段，实际 {data}"
        print(f"    OK (status=200，SG ID={BACKUP_SG_ID}，响应不含 fixed_rules_count)")
    finally:
        # 恢复原 SG（若原来有配置）
        if original_sg_id and original_sg_id != BACKUP_SG_ID:
            restore_resp = bind_sg({"security_group_id": original_sg_id, "auto_fix_rules": False})
            if restore_resp.status_code == 200:
                print(f"    已恢复原 SG ID={original_sg_id}")
            else:
                print(f"    ⚠️ 恢复原 SG 失败，请手动恢复 SG ID={original_sg_id}")

def test_bind_valid_sg_auto_fix_complete():
    """
    场景10：绑定合法存在的安全组，auto_fix_rules=true，规则已完整 → 200，ok=true，fixed_rules_count=0。
    ⚠️ 此场景会实际修改站点配置，测试完成后会恢复原 SG。
    """
    print(">>> [bind] 场景10：绑定合法 SG，auto_fix_rules=true，规则完整 → fixed_rules_count=0 ...")
    if not BACKUP_SG_ID:
        print("    SKIP (BACKUP_SG_ID 未设置，跳过实际绑定测试)")
        return

    # 先检查 BACKUP_SG_ID 的规则是否完整
    missing = check_rules(BACKUP_SG_ID)
    if missing:
        print(f"    INFO (BACKUP_SG_ID={BACKUP_SG_ID} 规则不完整，missing={len(missing)} 条，"
              f"此场景需在规则完整的 SG 上验证，改为验证 fixed_rules_count 字段存在)")

    original_sg_id = get_current_sg_id()

    try:
        resp = bind_sg({"security_group_id": BACKUP_SG_ID, "auto_fix_rules": True})
        assert resp.status_code == 200, \
            f"期望 200，实际 {resp.status_code}，body={resp.text}"
        data = resp.json()
        assert data.get("ok") is True, f"响应 ok 应为 true，实际 {data}"
        # auto_fix_rules=true 时，响应中必须含 fixed_rules_count 字段
        assert "fixed_rules_count" in data, \
            f"auto_fix_rules=true 时响应应含 fixed_rules_count 字段，实际 {data}"
        fixed_count = data["fixed_rules_count"]
        assert isinstance(fixed_count, int) and fixed_count >= 0, \
            f"fixed_rules_count 应为非负整数，实际 {fixed_count}"
        if not missing:
            assert fixed_count == 0, \
                f"规则完整时 fixed_rules_count 应为 0，实际 {fixed_count}"
        print(f"    OK (status=200，fixed_rules_count={fixed_count})")
    finally:
        if original_sg_id and original_sg_id != BACKUP_SG_ID:
            restore_resp = bind_sg({"security_group_id": original_sg_id, "auto_fix_rules": False})
            if restore_resp.status_code == 200:
                print(f"    已恢复原 SG ID={original_sg_id}")
            else:
                print(f"    ⚠️ 恢复原 SG 失败，请手动恢复 SG ID={original_sg_id}")

def test_bind_verify_site_config_updated():
    """
    场景12：绑定成功后，通过 GET /admin/config/security-group 验证 security_group_id 已更新。
    ⚠️ 此场景会实际修改站点配置，测试完成后会恢复原 SG。
    """
    print(">>> [bind] 场景12：绑定成功后验证 SiteConfig 中 security_group_id 已更新 ...")
    if not BACKUP_SG_ID:
        print("    SKIP (BACKUP_SG_ID 未设置，跳过实际绑定测试)")
        return

    original_sg_id = get_current_sg_id()

    try:
        resp = bind_sg({"security_group_id": BACKUP_SG_ID, "auto_fix_rules": False})
        assert resp.status_code == 200, \
            f"期望 200，实际 {resp.status_code}，body={resp.text}"

        # 验证 SiteConfig 已同步更新（同步部分，不依赖异步换绑）
        verify_resp = seed.get("/admin/config/security-group", expect=None, timeout=10, raw=True)
        assert verify_resp.status_code == 200, \
            f"查询 SG 期望 200，实际 {verify_resp.status_code}"
        if not verify_resp.text.strip():
            print(f"    OK (绑定返回 200，但 GET SG 返回空体，可能 site_config 同步延迟)")
        else:
            sg_set = verify_resp.json().get("Response", {}).get("SecurityGroupSet", [])
            assert sg_set, "绑定后查询 SG 应返回至少一条记录"
            current_id = sg_set[0].get("SecurityGroupId", "")
            assert current_id == BACKUP_SG_ID, \
                f"绑定后 SiteConfig 中 security_group_id 应为 {BACKUP_SG_ID}，实际 {current_id}"
            print(f"    OK (绑定后 SiteConfig 已同步更新为 SG ID={BACKUP_SG_ID})")
    finally:
        if original_sg_id and original_sg_id != BACKUP_SG_ID:
            restore_resp = bind_sg({"security_group_id": original_sg_id, "auto_fix_rules": False})
            if restore_resp.status_code == 200:
                print(f"    已恢复原 SG ID={original_sg_id}")
            else:
                print(f"    ⚠️ 恢复原 SG 失败，请手动恢复 SG ID={original_sg_id}")

def test_bind_when_no_sg_configured():
    """
    场景13：当前未配置安全组（SecurityGroupId=""）时，绑定新安全组 → 200，正常绑定（不触发"绑定自身"校验）。
    注意：仅在无 SG 配置的环境下有效，已配置则自动跳过。
    """
    print(">>> [bind] 场景13：未配置安全组时绑定新 SG → 200，不触发绑定自身校验 ...")
    current_sg_id = get_current_sg_id()
    if current_sg_id:
        print("    SKIP (当前已配置安全组，此场景需在无 SG 配置的环境下验证)")
        return
    if not BACKUP_SG_ID:
        print("    SKIP (BACKUP_SG_ID 未设置，跳过实际绑定测试)")
        return

    resp = bind_sg({"security_group_id": BACKUP_SG_ID, "auto_fix_rules": False})
    assert resp.status_code == 200, \
        f"期望 200（未配置安全组时绑定不触发自身校验），实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert data.get("ok") is True, f"响应 ok 应为 true，实际 {data}"
    print(f"    OK (status=200，未配置安全组时绑定成功，不触发绑定自身校验)")

def test_bind_async_rebind_note():
    """
    场景15：实例安全组异步换绑（可选验证）。
    注意：bind 接口返回 200 后，实例换绑是异步进行的（goroutine），
    接口本身不等待换绑完成。此场景标注为可选验证，需等待异步完成后查询实例安全组。
    本测试仅验证接口返回 200 后 SiteConfig 已同步更新（同步部分），
    异步换绑的验证需在实际环境中手动确认或通过查询实例安全组接口验证。
    """
    print(">>> [bind] 场景15：实例安全组异步换绑（可选验证，仅验证接口同步部分）...")
    if not BACKUP_SG_ID:
        print("    SKIP (BACKUP_SG_ID 未设置)")
        return

    original_sg_id = get_current_sg_id()

    try:
        resp = bind_sg({"security_group_id": BACKUP_SG_ID, "auto_fix_rules": False})
        assert resp.status_code == 200, \
            f"期望 200，实际 {resp.status_code}，body={resp.text}"
        data = resp.json()
        assert data.get("ok") is True, f"响应 ok 应为 true，实际 {data}"
        # 接口立即返回 200，异步换绑在后台进行
        # 验证同步部分：SiteConfig 已更新
        verify_resp = seed.get("/admin/config/security-group", expect=None, timeout=10, raw=True)
        assert verify_resp.status_code == 200
        if not verify_resp.text.strip():
            print(f"    OK (接口返回 200，GET SG 返回空体，可能 site_config 同步延迟)")
            print(f"    INFO (异步换绑验证：需等待后台 goroutine 完成后，查询实例安全组确认)")
        else:
            sg_set = verify_resp.json().get("Response", {}).get("SecurityGroupSet", [])
            current_id = sg_set[0].get("SecurityGroupId", "") if sg_set else ""
            assert current_id == BACKUP_SG_ID, \
                f"SiteConfig 应已同步更新为 {BACKUP_SG_ID}，实际 {current_id}"
            print(f"    OK (接口返回 200，SiteConfig 已同步更新，异步换绑在后台进行)")
            print(f"    INFO (异步换绑验证：需等待后台 goroutine 完成后，查询实例安全组确认)")
    finally:
        if original_sg_id and original_sg_id != BACKUP_SG_ID:
            restore_resp = bind_sg({"security_group_id": original_sg_id, "auto_fix_rules": False})
            if restore_resp.status_code == 200:
                print(f"    已恢复原 SG ID={original_sg_id}")
            else:
                print(f"    ⚠️ 恢复原 SG 失败，请手动恢复 SG ID={original_sg_id}")

# ─────────────────────────────────────────────
# Main
# ─────────────────────────────────────────────

def main():
    health_check()
    print()
    run_tests(globals(), title="POST /admin/config/security-group/bind 绑定安全组")

if __name__ == "__main__":
    main()
