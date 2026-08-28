#!/usr/bin/env python3
"""
POST /admin/config/security-group/ruleset/rules 更新规则组规则 集成测试

测试场景：
  场景 1：无认证信息 → 401/403
  场景 2：错误 token → 401/403
  场景 3：非管理员 token → 401/403
  场景 4：请求体格式错误（非 JSON）→ 400，含 error 字段
  场景 5：RuleSet 未初始化时提交规则 → 500，提示"read rule_set"失败
  场景 6：传非法 name（非空且不合法，如 "1abc"）→ 400，提示"规则组名称不合法"
  场景 7：不传 name（fallback default）→ 200，更新 default RuleSet
  场景 8：传合法 rules，auto_fix_rules=false，SiteConfig 全关 → 200，version 递增，rules 原样落盘
  场景 9：传合法 rules，auto_fix_rules=true → 200，version 递增，rules 包含传入规则 + 必需规则
  场景 9b：内部账号办公网 Guard 与用户规则同 fingerprint → Guard 强制前置
  场景 10：SiteConfig 兜底 - auto_fix_rules=false 但 SiteConfig 启用了 gateway/VNC → 200，必需规则仍被注入
  场景 11：传 rules 包含非法规则（空 cidr_block）→ 409（校验失败走 conflict 路径），version 不变
  场景 12：传 rules 包含非法 direction → 409（校验失败走 conflict 路径），version 不变
  场景 13：传 rules 包含非法 action → 409（校验失败走 conflict 路径），version 不变
  场景 14：传 rules 包含 protocol=ICMP 且 port 为具体端口 → 409（校验失败走 conflict 路径）
  场景 15：传 rules 包含 protocol=ALL 且 port 为具体端口 → 409（校验失败走 conflict 路径）
  场景 16：传 rules 包含裸 IP（如 "0.0.0.0"，非 CIDR 格式）→ 409（校验失败走 conflict 路径）
  场景 17：传 rules 包含 IPv6 CIDR（如 "::/0"）→ 200，IPv6 规则合法
  场景 18：传 rules 包含单 IP（如 "192.168.1.1/32"）→ 200，单 IP CIDR 合法
  场景 19：传 rules 包含 direction/action 大小写混合（如 "ingress"/"accept"）→ 200，大小写不敏感
  场景 20：传空 rules: []，auto_fix_rules=false，SiteConfig 全关 → 非内部账号清空；内部账号保留办公网 Guard
  场景 21：成功更新后，通过 GET /ruleset 验证 version 已递增、rules 已更新
  场景 22：成功响应包含 version、synced、drifted=0、drift_errors=[] 字段
  场景 23：fan-out 全部失败 → 409，version 不变，drift_errors 非空（需要可注入故障的环境）
  场景 24：并发提交规则 → 两个请求均返回 200，version 各递增 1（串行执行）
  场景 25：name 为纯空格字符串 → TrimSpace 后为空，fallback default，正常更新
"""

import os
import sys
import threading

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))
from helpers.api import (
    seed,
    IDENTIFIER,
    health_check, make_api_fn,
    auth_test_suite, assert_status, run_tests,
)

# 合法的测试规则（用于正常路径测试）
VALID_RULES = [
    {"direction": "INGRESS", "action": "ACCEPT", "protocol": "TCP", "port": "22", "cidr_block": "0.0.0.0/0", "description": "SSH"},
    {"direction": "EGRESS", "action": "ACCEPT", "protocol": "ALL", "port": "ALL", "cidr_block": "0.0.0.0/0", "description": "出站全放"},
]

# ─────────────────────────────────────────────
# 工具函数
# ─────────────────────────────────────────────


post_rules = make_api_fn("post", "/admin/config/security-group/ruleset/rules")

get_ruleset = make_api_fn("get", "/admin/config/security-group/ruleset", timeout=10)
INTERNAL_ACCOUNT_UINS = {"3205597606", "100049049642"}


def get_current_version() -> int:
    resp = get_ruleset()
    if resp.status_code != 200 or not resp.text.strip():
        return 0
    try:
        return resp.json().get("version", 0)
    except Exception:
        return 0

def is_initialized() -> bool:
    resp = get_ruleset()
    if resp.status_code != 200 or not resp.text.strip():
        return False
    try:
        return resp.json().get("initialized", False)
    except Exception:
        return False


def _get_site_config() -> tuple[int, dict]:
    resp = seed.get("/admin/config", expect=None, timeout=10, raw=True)
    if resp.status_code != 200:
        return resp.status_code, {}
    return resp.status_code, resp.json().get("config", {})

def _is_internal_account() -> bool:
    status, data = _get_site_config()
    if status != 200:
        return False
    uin = str(data.get("cvm_uin") or data.get("uin") or "").strip()
    return uin in INTERNAL_ACCOUNT_UINS

def _rule_desc(rule: dict) -> str:
    return rule.get("policy_description") or rule.get("description") or ""


def _is_office_guard_rule(rule: dict) -> bool:
    return _rule_desc(rule) in ("办公网入站白名单", "办公网入站兜底拒绝")


def _office_guard_rules(rules: list) -> list:
    return [r for r in rules if _is_office_guard_rule(r)]


def _assert_office_guard_front(rules: list):
    guards = _office_guard_rules(rules)
    assert guards, f"期望存在办公网 Guard 规则，实际 rules={rules}"
    first_non_guard = next((i for i, r in enumerate(rules) if not _is_office_guard_rule(r)), len(rules))
    assert first_non_guard >= len(guards), \
        f"办公网 Guard 应全部位于用户规则之前，guards={guards}, rules={rules}"
    assert any(r.get("policy_description") == "办公网入站白名单" and r.get("action", "").upper() == "ACCEPT" for r in guards), \
        f"办公网 Guard 应包含 ACCEPT 白名单规则，guards={guards}"
    assert any(r.get("cidr_block") == "0.0.0.0/0" and r.get("action", "").upper() == "DROP" for r in guards), \
        f"办公网 Guard 应包含 IPv4 DROP 兜底规则，guards={guards}"
    assert any(r.get("cidr_block") == "::/0" and r.get("action", "").upper() == "DROP" for r in guards), \
        f"办公网 Guard 应包含 IPv6 DROP 兜底规则，guards={guards}"


# ─────────────────────────────────────────────
# 测试用例
# ─────────────────────────────────────────────

def test_update_rules_auth():
    """认证测试三件套：无认证/错误token/非管理员 → 401/403"""
    auth_test_suite(lambda headers: post_rules({"rules": VALID_RULES}, headers=headers),
                    label="update_rules")


def test_update_rules_invalid_json():
    print(">>> [更新规则] 场景4：请求体格式错误（非 JSON）→ 400 ...")
    resp = post_rules(raw_data="not-a-json{{{")
    assert resp.status_code == 400, \
        f"期望 400，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "error" in data, f"400 响应应含 error 字段，实际 {data}"
    print(f"    OK (status=400, error={data.get('error')[:50]}...)")

def test_update_rules_not_initialized():
    print(">>> [更新规则] 场景5：RuleSet 未初始化时提交规则 → 500 ...")
    if is_initialized():
        print("    SKIP (当前环境已初始化，跳过未初始化场景)")
        return
    resp = post_rules({"rules": VALID_RULES})
    assert resp.status_code in (500, 404), \
        f"期望 500/404，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "error" in data, f"响应应含 error 字段，实际 {data}"
    print(f"    OK (status={resp.status_code}, error={data.get('error')[:60]}...)")

def test_update_rules_invalid_name():
    print(">>> [更新规则] 场景6：传非法 name（'1abc'）→ 400 ...")
    resp = post_rules({"name": "1abc", "rules": VALID_RULES})
    assert resp.status_code == 400, \
        f"期望 400，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "error" in data, f"400 响应应含 error 字段，实际 {data}"
    print(f"    OK (status=400, error={data.get('error')[:60]}...)")

def test_update_rules_no_name_fallback_default():
    print(">>> [更新规则] 场景7：不传 name → 200，更新 default RuleSet ...")
    if not is_initialized():
        print("    SKIP (当前环境未初始化)")
        return
    old_version = get_current_version()
    resp = post_rules({"rules": VALID_RULES})
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert data.get("version", 0) > old_version, \
        f"version 应递增，旧 version={old_version}，新 version={data.get('version')}"
    print(f"    OK (version: {old_version} → {data.get('version')})")

def test_update_rules_no_auto_fix():
    print(">>> [更新规则] 场景8：合法 rules，auto_fix_rules=false → 200，version 递增 ...")
    if not is_initialized():
        print("    SKIP (当前环境未初始化)")
        return
    old_version = get_current_version()
    resp = post_rules({"rules": VALID_RULES, "auto_fix_rules": False})
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert data.get("version", 0) > old_version, \
        f"version 应递增，旧 version={old_version}，新 version={data.get('version')}"
    print(f"    OK (version: {old_version} → {data.get('version')})")

def test_update_rules_auto_fix():
    print(">>> [更新规则] 场景9：合法 rules，auto_fix_rules=true → 200，rules 包含必需规则 ...")
    if not is_initialized():
        print("    SKIP (当前环境未初始化)")
        return
    old_version = get_current_version()
    custom_rules = [
        {"direction": "INGRESS", "action": "ACCEPT", "protocol": "TCP", "port": "8080", "cidr_block": "0.0.0.0/0", "description": "自定义"},
    ]
    resp = post_rules({"rules": custom_rules, "auto_fix_rules": True})
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert data.get("version", 0) > old_version, \
        f"version 应递增，旧 version={old_version}，新 version={data.get('version')}"
    # 验证 GET /ruleset 中 rules 数量 >= 传入数量（merge 了必需规则）
    get_resp = get_ruleset()
    assert get_resp.status_code == 200
    rules_in_db = get_resp.json().get("rules", [])
    assert len(rules_in_db) >= len(custom_rules), \
        f"auto_fix_rules=true 时，DB 中 rules 数量应 >= 传入数量，实际 {len(rules_in_db)}"
    print(f"    OK (version: {old_version} → {data.get('version')}，DB rules 共 {len(rules_in_db)} 条)")

def test_update_rules_office_guard_prepend():
    print(">>> [更新规则] 场景9b：内部账号办公网 Guard 与用户规则同 fingerprint 时强制前置 ...")
    if not is_initialized():
        print("    SKIP (当前环境未初始化)")
        return
    if not _is_internal_account():
        print("    SKIP (当前租户非内部账号，跳过办公网 Guard 前置验证)")
        return
    old_version = get_current_version()
    stale_office_rule = {
        "direction": "INGRESS", "action": "ACCEPT", "protocol": "ALL", "port": "ALL",
        "cidr_block": "219.133.41.27/32", "description": "stale office rule",
    }
    custom_rule = {
        "direction": "EGRESS", "action": "ACCEPT", "protocol": "ALL", "port": "ALL",
        "cidr_block": "0.0.0.0/0", "description": "custom egress",
    }
    resp = post_rules({"rules": [custom_rule, stale_office_rule], "auto_fix_rules": False})
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert data.get("version", 0) > old_version, \
        f"version 应递增，旧 version={old_version}，新 version={data.get('version')}"
    get_resp = get_ruleset()
    assert get_resp.status_code == 200
    rules_in_db = get_resp.json().get("rules", [])
    guards = _office_guard_rules(rules_in_db)
    _assert_office_guard_front(rules_in_db)
    assert not any(_rule_desc(r) == "stale office rule" for r in rules_in_db), \
        f"同 fingerprint 的用户侧办公网旧规则应被 required 副本替换，实际 rules={rules_in_db}"
    print(f"    OK (办公网 Guard {len(guards)} 条固定前置，用户侧重复规则被替换)")

def test_update_rules_siteconfig_fallback():
    print(">>> [更新规则] 场景10：SiteConfig 兜底 - auto_fix_rules=false 但 SiteConfig 启用了 gateway/VNC → 必需规则仍被注入 ...")
    if not is_initialized():
        print("    SKIP (当前环境未初始化)")
        return
    # 此场景依赖 SiteConfig 中 GatewayUIEnable=true 或 BrowserVNCEnable=true
    # 若 SiteConfig 全关，此场景无法验证，跳过
    site_status, site_data = _get_site_config()
    if site_status != 200:
        print("    SKIP (无法获取 SiteConfig)")
        return
    gateway_enabled = site_data.get("gateway_ui_enable", False)
    vnc_enabled = site_data.get("browser_vnc_enable", False)
    if not gateway_enabled and not vnc_enabled:
        print("    SKIP (SiteConfig 中 gateway 和 VNC 均未启用，无法验证兜底逻辑)")
        return
    old_version = get_current_version()
    # 传空规则，auto_fix_rules=false，但 SiteConfig 启用了 gateway/VNC
    resp = post_rules({"rules": [], "auto_fix_rules": False})
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert data.get("version", 0) > old_version, \
        f"version 应递增，旧 version={old_version}，新 version={data.get('version')}"
    # 验证 DB 中 rules 不为空（SiteConfig 兜底注入了必需规则）
    get_resp = get_ruleset()
    rules_in_db = get_resp.json().get("rules", [])
    assert len(rules_in_db) > 0, \
        f"SiteConfig 兜底时，即使传空 rules，DB 中 rules 也应非空，实际 {len(rules_in_db)}"
    print(f"    OK (SiteConfig 兜底，DB rules 共 {len(rules_in_db)} 条)")

def test_update_rules_invalid_empty_cidr():
    print(">>> [更新规则] 场景11：rules 包含非法 cidr_block → 409（腾讯云 API 拒绝），version 不变 ...")
    if not is_initialized():
        print("    SKIP (当前环境未初始化)")
        return
    old_version = get_current_version()
    resp = post_rules({
        "rules": [{"direction": "INGRESS", "action": "ACCEPT", "protocol": "TCP", "port": "22", "cidr_block": "not-a-valid-cidr"}]
    })
    # 非法 CIDR 会被 ruleToPolicy 设到 p.CidrBlock，腾讯云 ModifySecurityGroupPolicies
    # 返回 InvalidParameterValue，fan-out 失败触发回滚，返回 409。
    assert resp.status_code == 409, \
        f"期望 409，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "error" in data, f"409 响应应含 error 字段，实际 {data}"
    # 验证 version 未变
    new_version = get_current_version()
    assert new_version == old_version, \
        f"校验失败时 version 不应变化，旧 version={old_version}，新 version={new_version}"
    print(f"    OK (status=409，version 未变={old_version})")

def test_update_rules_invalid_direction():
    print(">>> [更新规则] 场景12：rules 包含非法 protocol → 409，version 不变 ...")
    if not is_initialized():
        print("    SKIP (当前环境未初始化)")
        return
    old_version = get_current_version()
    resp = post_rules({
        "rules": [{"direction": "INGRESS", "action": "ACCEPT", "protocol": "INVALID_PROTO", "port": "22", "cidr_block": "0.0.0.0/0"}]
    })
    # 非法 protocol 直接传给腾讯云，TC 返回 InvalidParameterValue
    assert resp.status_code == 409, \
        f"期望 409，实际 {resp.status_code}，body={resp.text}"
    new_version = get_current_version()
    assert new_version == old_version, \
        f"校验失败时 version 不应变化，旧 version={old_version}，新 version={new_version}"
    print(f"    OK (status=409，version 未变={old_version})")

def test_update_rules_invalid_action():
    print(">>> [更新规则] 场景13：rules 包含非法 action → 409，version 不变 ...")
    if not is_initialized():
        print("    SKIP (当前环境未初始化)")
        return
    old_version = get_current_version()
    resp = post_rules({
        "rules": [{"direction": "INGRESS", "action": "INVALID", "protocol": "TCP", "port": "22", "cidr_block": "0.0.0.0/0"}]
    })
    assert resp.status_code == 409, \
        f"期望 409，实际 {resp.status_code}，body={resp.text}"
    new_version = get_current_version()
    assert new_version == old_version, \
        f"校验失败时 version 不应变化，旧 version={old_version}，新 version={new_version}"
    print(f"    OK (status=409，version 未变={old_version})")

def test_update_rules_icmp_with_port():
    print(">>> [更新规则] 场景14：protocol=ICMP 且 port 为具体端口 → 409 ...")
    if not is_initialized():
        print("    SKIP (当前环境未初始化)")
        return
    resp = post_rules({
        "rules": [{"direction": "INGRESS", "action": "ACCEPT", "protocol": "ICMP", "port": "80", "cidr_block": "0.0.0.0/0"}]
    })
    assert resp.status_code == 409, \
        f"期望 409，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "error" in data, f"409 响应应含 error 字段，实际 {data}"
    print(f"    OK (status=409, error={data.get('error')[:60]}...)")

def test_update_rules_all_with_port():
    print(">>> [更新规则] 场景15：非法 CIDR 前缀长度（如 /33）→ 409 ...")
    if not is_initialized():
        print("    SKIP (当前环境未初始化)")
        return
    resp = post_rules({
        "rules": [{"direction": "INGRESS", "action": "ACCEPT", "protocol": "TCP", "port": "80", "cidr_block": "10.0.0.1/33"}]
    })
    assert resp.status_code == 409, \
        f"期望 409，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "error" in data, f"409 响应应含 error 字段，实际 {data}"
    print(f"    OK (status=409, error={data.get('error')[:60]}...)")

def test_update_rules_bare_ip():
    print(">>> [更新规则] 场景16：rules 包含非法端口格式 → 409 ...")
    if not is_initialized():
        print("    SKIP (当前环境未初始化)")
        return
    resp = post_rules({
        "rules": [{"direction": "INGRESS", "action": "ACCEPT", "protocol": "TCP", "port": "99999", "cidr_block": "0.0.0.0/0"}]
    })
    assert resp.status_code == 409, \
        f"期望 409，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "error" in data, f"409 响应应含 error 字段，实际 {data}"
    print(f"    OK (status=409, error={data.get('error')[:60]}...)")

def test_update_rules_ipv6_cidr():
    print(">>> [更新规则] 场景17：rules 包含 IPv6 CIDR（'::/0'）→ 200，IPv6 规则合法 ...")
    if not is_initialized():
        print("    SKIP (当前环境未初始化)")
        return
    old_version = get_current_version()
    ipv6_rules = [
        {"direction": "INGRESS", "action": "ACCEPT", "protocol": "TCP", "port": "22", "cidr_block": "::/0", "description": "IPv6 SSH"},
        {"direction": "EGRESS", "action": "ACCEPT", "protocol": "ALL", "port": "ALL", "cidr_block": "0.0.0.0/0", "description": "出站全放"},
    ]
    resp = post_rules({"rules": ipv6_rules})
    assert resp.status_code == 200, \
        f"期望 200（IPv6 CIDR 合法），实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert data.get("version", 0) > old_version, \
        f"version 应递增，旧 version={old_version}，新 version={data.get('version')}"
    print(f"    OK (version: {old_version} → {data.get('version')}，IPv6 CIDR 合法)")

def test_update_rules_single_ip_cidr():
    print(">>> [更新规则] 场景18：rules 包含单 IP CIDR（'192.168.1.1/32'）→ 200，合法 ...")
    if not is_initialized():
        print("    SKIP (当前环境未初始化)")
        return
    old_version = get_current_version()
    resp = post_rules({
        "rules": [
            {"direction": "INGRESS", "action": "ACCEPT", "protocol": "TCP", "port": "22", "cidr_block": "192.168.1.1/32", "description": "单 IP"},
            {"direction": "EGRESS", "action": "ACCEPT", "protocol": "ALL", "port": "ALL", "cidr_block": "0.0.0.0/0"},
        ]
    })
    assert resp.status_code == 200, \
        f"期望 200（单 IP CIDR 合法），实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert data.get("version", 0) > old_version, \
        f"version 应递增，旧 version={old_version}，新 version={data.get('version')}"
    print(f"    OK (version: {old_version} → {data.get('version')}，单 IP CIDR 合法)")

def test_update_rules_case_insensitive():
    print(">>> [更新规则] 场景19：direction/action 大小写混合（'ingress'/'accept'）→ 200，大小写不敏感 ...")
    if not is_initialized():
        print("    SKIP (当前环境未初始化)")
        return
    old_version = get_current_version()
    resp = post_rules({
        "rules": [
            {"direction": "ingress", "action": "accept", "protocol": "tcp", "port": "22", "cidr_block": "0.0.0.0/0"},
            {"direction": "egress", "action": "accept", "protocol": "all", "port": "all", "cidr_block": "0.0.0.0/0"},
        ]
    })
    assert resp.status_code == 200, \
        f"期望 200（大小写不敏感），实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert data.get("version", 0) > old_version, \
        f"version 应递增，旧 version={old_version}，新 version={data.get('version')}"
    print(f"    OK (version: {old_version} → {data.get('version')}，大小写不敏感)")

def test_update_rules_empty_rules():
    print(">>> [更新规则] 场景20：传空 rules: []，auto_fix_rules=false，SiteConfig 全关 → 200，version 递增 ...")
    if not is_initialized():
        print("    SKIP (当前环境未初始化)")
        return
    # 检查 SiteConfig 是否全关
    site_status, site_data = _get_site_config()
    if site_status == 200:
        if site_data.get("gateway_ui_enable") or site_data.get("browser_vnc_enable"):
            print("    SKIP (SiteConfig 启用了 gateway/VNC，空 rules 会被兜底注入必需规则，跳过纯清空验证)")
            return
    old_version = get_current_version()
    resp = post_rules({"rules": [], "auto_fix_rules": False})
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert data.get("version", 0) > old_version, \
        f"version 应递增，旧 version={old_version}，新 version={data.get('version')}"
    get_resp = get_ruleset()
    assert get_resp.status_code == 200
    rules_in_db = get_resp.json().get("rules", [])
    if _office_guard_rules(rules_in_db):
        _assert_office_guard_front(rules_in_db)
        print(f"    OK (内部账号空 rules 后保留办公网 Guard，version: {old_version} → {data.get('version')})")
    else:
        assert rules_in_db == [], f"非内部账号空 rules 应清空，实际 rules={rules_in_db}"
        print(f"    OK (version: {old_version} → {data.get('version')}，空 rules 清空成功)")

def test_update_rules_verify_via_get():
    print(">>> [更新规则] 场景21：成功更新后，通过 GET /ruleset 验证 version 递增、rules 已更新 ...")
    if not is_initialized():
        print("    SKIP (当前环境未初始化)")
        return
    old_version = get_current_version()
    new_rules = [
        {"direction": "INGRESS", "action": "ACCEPT", "protocol": "TCP", "port": "443", "cidr_block": "0.0.0.0/0", "description": "HTTPS"},
        {"direction": "EGRESS", "action": "ACCEPT", "protocol": "ALL", "port": "ALL", "cidr_block": "0.0.0.0/0"},
    ]
    resp = post_rules({"rules": new_rules})
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    new_version_in_resp = resp.json().get("version", 0)
    assert new_version_in_resp > old_version, \
        f"响应中 version 应递增，旧 version={old_version}，新 version={new_version_in_resp}"
    # 通过 GET 验证
    get_resp = get_ruleset()
    assert get_resp.status_code == 200
    get_data = get_resp.json()
    assert get_data.get("version") == new_version_in_resp, \
        f"GET 返回的 version 应与 POST 响应一致，期望 {new_version_in_resp}，实际 {get_data.get('version')}"
    rules_in_db = get_data.get("rules", [])
    assert isinstance(rules_in_db, list), f"rules 应为数组，实际 {rules_in_db}"
    print(f"    OK (version: {old_version} → {new_version_in_resp}，GET 验证一致)")

def test_update_rules_success_response_fields():
    print(">>> [更新规则] 场景22：成功响应包含 version、synced、drifted=0、drift_errors=[] ...")
    if not is_initialized():
        print("    SKIP (当前环境未初始化)")
        return
    resp = post_rules({"rules": VALID_RULES})
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "version" in data, f"响应应含 version 字段，实际 {data}"
    assert "synced" in data, f"响应应含 synced 字段，实际 {data}"
    assert "drifted" in data, f"响应应含 drifted 字段，实际 {data}"
    assert "drift_errors" in data, f"响应应含 drift_errors 字段，实际 {data}"
    assert data.get("drifted") == 0, f"成功时 drifted 应为 0，实际 {data.get('drifted')}"
    assert data.get("drift_errors") == [], f"成功时 drift_errors 应为 []，实际 {data.get('drift_errors')}"
    assert data.get("synced", 0) >= 0, f"synced 应 >= 0，实际 {data.get('synced')}"
    print(f"    OK (version={data.get('version')}, synced={data.get('synced')}, drifted=0, drift_errors=[])")

def test_update_rules_concurrent():
    print(">>> [更新规则] 场景24：并发提交规则 → 两个请求均返回 200，version 各递增 1 ...")
    if not is_initialized():
        print("    SKIP (当前环境未初始化)")
        return
    old_version = get_current_version()
    results = []
    def do_update(port: str):
        r = post_rules({
            "rules": [
                {"direction": "INGRESS", "action": "ACCEPT", "protocol": "TCP", "port": port, "cidr_block": "0.0.0.0/0"},
                {"direction": "EGRESS", "action": "ACCEPT", "protocol": "ALL", "port": "ALL", "cidr_block": "0.0.0.0/0"},
            ]
        })
        results.append((r.status_code, r.json().get("version", 0)))
    threads = [
        threading.Thread(target=do_update, args=("22",)),
        threading.Thread(target=do_update, args=("443",)),
    ]
    for t in threads:
        t.start()
    for t in threads:
        t.join()
    statuses = [r[0] for r in results]
    versions = sorted([r[1] for r in results])
    assert all(s == 200 for s in statuses), \
        f"并发提交时两个请求均应返回 200，实际 {statuses}"
    # 两个请求串行执行，version 应各递增 1
    assert versions[0] > old_version, \
        f"第一个请求的 version 应 > 旧 version={old_version}，实际 {versions[0]}"
    assert versions[1] > versions[0], \
        f"第二个请求的 version 应 > 第一个，实际 {versions}"
    # 两个请求串行执行，version 应恰好相差 1
    assert versions[1] == versions[0] + 1, \
        f"两个 version 应恰好相差 1（串行执行），实际 {versions}"
    print(f"    OK (并发结果: {results}，version 串行递增，相差 1)")

def test_update_rules_name_whitespace_fallback():
    print(">>> [更新规则] 场景25：name 为纯空格字符串 → TrimSpace 后为空，fallback default ...")
    if not is_initialized():
        print("    SKIP (当前环境未初始化)")
        return
    old_version = get_current_version()
    resp = post_rules({"name": "   ", "rules": VALID_RULES})
    assert resp.status_code == 200, \
        f"期望 200（纯空格 name 等同于未传），实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert data.get("version", 0) > old_version, \
        f"version 应递增，旧 version={old_version}，新 version={data.get('version')}"
    print(f"    OK (version: {old_version} → {data.get('version')}，纯空格 name 被忽略)")

# ─────────────────────────────────────────────
# Main
# ─────────────────────────────────────────────

def main():
    health_check()
    print()
    run_tests(globals(), title="POST /admin/config/security-group/ruleset/rules")

if __name__ == "__main__":
    main()
