#!/usr/bin/env python3
"""
GET /admin/config/security-group/list 查询云端安全组列表 集成测试

测试场景：
  场景 1：无认证信息 → 401/403
  场景 2a：错误 token → 401/403
  场景 2b：非管理员 token → 401/403
  场景 3：默认分页（不传任何参数）→ 200，返回 security_groups 数组和 total_count，
           security_groups 不为 null（空时为 []），total_count 为非负整数
  场景 4：传 offset + limit 分页参数 → 200，返回结果数量 ≤ limit
  场景 5：传 keyword 关键字过滤 → 200，返回结果中安全组名称均包含 keyword（或结果为空）
  场景 6：传单个 security_group_id 精确查询 → 200，返回结果中 security_group_id 与传入一致
  场景 7：传逗号分隔多个 security_group_id → 200，返回结果中每条 security_group_id 均在传入列表中
  场景 8：结果中不含 clawpro-sg- 前缀的安全组（托管 SG 被过滤）→ 验证过滤逻辑生效
  场景 9：total_count 为非负整数，且 ≥ len(security_groups)（过滤后 total_count 已调整）
  场景 10：传不存在的 security_group_id（sg-前缀标准格式）→ 500（腾讯云 SDK 报错）或 200（空列表）
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


list_sg = make_api_fn("get", "/admin/config/security-group/list", timeout=15)


# ─────────────────────────────────────────────
# 测试用例
# ─────────────────────────────────────────────

def test_list_sg_auth():
    """认证测试三件套：无认证/错误token/非管理员 → 401/403"""
    auth_test_suite(lambda headers: list_sg(headers=headers),
                    label="list_sg")


def test_list_sg_default_pagination():
    """
    场景3：默认分页（不传任何参数）→ 200，
    返回 security_groups 数组（不为 null，空时为 []）和 total_count（非负整数）。
    """
    print(">>> [查询云端SG列表] 场景3：默认分页（不传参数）→ 200，结构正确 ...")
    resp = list_sg()
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    assert "security_groups" in data, \
        f"响应应包含 security_groups 字段，实际 {data}"
    assert "total_count" in data, \
        f"响应应包含 total_count 字段，实际 {data}"
    sg_list = data["security_groups"]
    assert isinstance(sg_list, list), \
        f"security_groups 应为数组（空时为 []，不为 null），实际 {type(sg_list)}"
    total = data["total_count"]
    assert isinstance(total, int) and total >= 0, \
        f"total_count 应为非负整数，实际 {total}"
    print(f"    OK (total_count={total}, returned={len(sg_list)})")

def test_list_sg_pagination_params():
    """
    场景4：传 offset + limit 分页参数 → 200，返回结果数量 ≤ limit。
    """
    print(">>> [查询云端SG列表] 场景4：传 offset + limit 分页参数 → 200，结果数量 ≤ limit ...")
    limit = 5
    resp = list_sg(params={"offset": 0, "limit": limit})
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    sg_list = data.get("security_groups", [])
    assert isinstance(sg_list, list), f"security_groups 应为数组，实际 {type(sg_list)}"
    assert len(sg_list) <= limit, \
        f"返回结果数量应 ≤ limit={limit}，实际 {len(sg_list)}"
    print(f"    OK (limit={limit}, returned={len(sg_list)})")

def test_list_sg_keyword_filter():
    """
    场景5：传 keyword 关键字过滤 → 200，返回结果中安全组名称均包含 keyword（或结果为空）。
    """
    print(">>> [查询云端SG列表] 场景5：传 keyword 关键字过滤 → 200，结果名称均含 keyword ...")
    # 先获取一个真实存在的 SG 名称前缀作为 keyword
    base_resp = list_sg(params={"limit": 1})
    if base_resp.status_code != 200:
        print("    SKIP (基础查询失败)")
        return
    base_list = base_resp.json().get("security_groups", [])
    if not base_list:
        print("    SKIP (当前账号下无安全组，跳过 keyword 过滤测试)")
        return

    # 取第一个 SG 名称的前 3 个字符作为 keyword
    first_name = base_list[0].get("security_group_name", "")
    if not first_name or len(first_name) < 2:
        print("    SKIP (SG 名称过短，跳过 keyword 过滤测试)")
        return
    keyword = first_name[:3]

    resp = list_sg(params={"keyword": keyword})
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    sg_list = data.get("security_groups", [])
    # 腾讯云按名称过滤，结果中每条 SG 名称应包含 keyword（或结果为空）
    for sg in sg_list:
        name = sg.get("security_group_name", "")
        assert keyword.lower() in name.lower(), \
            f"keyword='{keyword}' 过滤后，SG 名称应包含 keyword，实际 name='{name}'"
    print(f"    OK (keyword='{keyword}', returned={len(sg_list)})")

def test_list_sg_exact_id_query():
    """
    场景6：传单个 security_group_id 精确查询 → 200，返回结果中 security_group_id 与传入一致。
    """
    print(">>> [查询云端SG列表] 场景6：传单个 security_group_id 精确查询 → 200，ID 匹配 ...")
    # 先获取一个真实存在的 SG ID
    base_resp = list_sg(params={"limit": 5})
    if base_resp.status_code != 200:
        print("    SKIP (基础查询失败)")
        return
    base_list = base_resp.json().get("security_groups", [])
    if not base_list:
        print("    SKIP (当前账号下无安全组，跳过精确查询测试)")
        return
    # 防御性过滤：只使用 sg- 前缀的标准格式 ID，避免旧格式 ID 导致 API 报错
    valid_ids = [sg.get("security_group_id", "") for sg in base_list
                 if sg.get("security_group_id", "").startswith("sg-")]
    if not valid_ids:
        print("    SKIP (无标准格式 sg- 前缀的 SG ID，跳过)")
        return
    target_id = valid_ids[0]

    resp = list_sg(params={"security_group_id": target_id})
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    sg_list = data.get("security_groups", [])
    assert len(sg_list) >= 1, \
        f"精确查询 SG ID={target_id} 应至少返回 1 条，实际 {len(sg_list)}"
    returned_id = sg_list[0].get("security_group_id", "")
    assert returned_id == target_id, \
        f"返回的 security_group_id 应为 {target_id}，实际 {returned_id}"
    print(f"    OK (security_group_id={target_id}，精确查询结果匹配)")

def test_list_sg_multi_id_query():
    """
    场景7：传逗号分隔多个 security_group_id → 200，返回结果中每条 security_group_id 均在传入列表中。
    """
    print(">>> [查询云端SG列表] 场景7：传逗号分隔多个 security_group_id → 200，结果 ID 均在传入列表中 ...")
    # 先获取至少 2 个真实存在的 SG ID
    base_resp = list_sg(params={"limit": 10})
    if base_resp.status_code != 200:
        print("    SKIP (基础查询失败)")
        return
    base_list = base_resp.json().get("security_groups", [])
    if len(base_list) < 2:
        print("    SKIP (当前账号下安全组数量不足 2 个，跳过多 ID 查询测试)")
        return

    # 防御性过滤：只使用 sg- 前缀的标准格式 ID，避免旧格式 ID 导致 API 报错
    id_list = [sg["security_group_id"] for sg in base_list
               if sg.get("security_group_id", "").startswith("sg-")][:2]
    if len(id_list) < 2:
        print("    SKIP (标准格式 sg- 前缀的 SG ID 不足 2 个，跳过)")
        return
    id_param = ",".join(id_list)

    resp = list_sg(params={"security_group_id": id_param})
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    sg_list = data.get("security_groups", [])
    id_set = set(id_list)
    for sg in sg_list:
        returned_id = sg.get("security_group_id", "")
        assert returned_id in id_set, \
            f"多 ID 查询结果中出现了未传入的 SG ID={returned_id}，传入列表={id_list}"
    print(f"    OK (传入 {len(id_list)} 个 ID，返回 {len(sg_list)} 条，均在传入列表中)")

def test_list_sg_no_clawpro_prefix():
    """
    场景8：结果中不含 clawpro-sg- 前缀的安全组（托管 SG 被过滤）。
    验证后端过滤逻辑：名称以 clawpro-sg- 开头的 SG 不应出现在列表中。
    """
    print(">>> [查询云端SG列表] 场景8：结果中不含 clawpro-sg- 前缀的安全组（托管 SG 被过滤）...")
    resp = list_sg(params={"limit": 100})
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    sg_list = data.get("security_groups", [])
    for sg in sg_list:
        name = sg.get("security_group_name", "")
        assert not name.startswith("clawpro-sg-"), \
            f"结果中不应包含 clawpro-sg- 前缀的安全组，实际出现 name='{name}'"
    print(f"    OK (共 {len(sg_list)} 条结果，均不含 clawpro-sg- 前缀)")

def test_list_sg_total_count_consistency():
    """
    场景9：total_count 为非负整数，且 ≥ len(security_groups)（过滤后 total_count 已调整）。
    """
    print(">>> [查询云端SG列表] 场景9：total_count ≥ len(security_groups)，数值一致性验证 ...")
    resp = list_sg(params={"limit": 20})
    assert resp.status_code == 200, \
        f"期望 200，实际 {resp.status_code}，body={resp.text}"
    data = resp.json()
    sg_list = data.get("security_groups", [])
    total = data.get("total_count", -1)
    assert isinstance(total, int) and total >= 0, \
        f"total_count 应为非负整数，实际 {total}"
    assert total >= len(sg_list), \
        f"total_count={total} 应 ≥ 当前页返回数量 {len(sg_list)}"
    print(f"    OK (total_count={total}, returned={len(sg_list)})")

def test_list_sg_nonexistent_id():
    """
    场景10：传不存在的 security_group_id → 500（腾讯云 SDK 报错）或 200（空列表）。
    腾讯云 DescribeSecurityGroups 对不存在的 SG ID 可能直接返回 SDK 错误，
    后端透传为 500；也可能返回空集合，后端返回 200 + 空列表。
    """
    print(">>> [查询云端SG列表] 场景10：传不存在的 security_group_id → 500 或 200（空列表）...")
    fake_id = "sg-00000000"
    resp = list_sg(params={"security_group_id": fake_id})
    assert resp.status_code in (200, 500), \
        f"期望 200 或 500，实际 {resp.status_code}，body={resp.text}"
    if resp.status_code == 200:
        data = resp.json()
        sg_list = data.get("security_groups", None)
        assert isinstance(sg_list, list), \
            f"security_groups 应为数组，实际 {type(sg_list)}"
        assert len(sg_list) == 0, \
            f"不存在的 SG ID 查询结果应为空列表，实际 {sg_list}"
        total = data.get("total_count", -1)
        assert total == 0, \
            f"不存在的 SG ID 查询 total_count 应为 0，实际 {total}"
        print(f"    OK (status=200, security_groups=[], total_count=0)")
    else:
        data = resp.json()
        assert "error" in data, f"500 响应应含 error 字段，实际 {data}"
        print(f"    OK (status=500, 腾讯云 SDK 报错: {data.get('error', '')[:80]})")

# ─────────────────────────────────────────────
# Main
# ─────────────────────────────────────────────

def main():
    health_check()
    print()
    run_tests(globals(), title="GET /admin/config/security-group/list")

if __name__ == "__main__":
    main()
