#!/usr/bin/env python3
"""
测试脚本：验证 GET /quota/data 接口

覆盖场景：

一、认证鉴权
  - 无 Cookie/Token 访问 → 401
  - 错误 Cookie 访问 → 401
  - 有效 Session Cookie 正常访问 → 200

二、响应结构验证
  - 顶层字段完整性：start_date、end_date、group_by、rows、quota_day、quota_period
  - rows 中 token 字段完整性：total_tokens、prompt_tokens、completion_tokens、request_count

三、数据隔离（核心）
  - 用户 A 只能看到自己的数据，看不到用户 B 的数据
  - 接口强制以当前登录用户过滤，不接受外部 user_id 参数

四、日期过滤
  - 默认日期范围（今天）
  - 明确指定日期范围
  - start_date > end_date 时自动交换（返回 200，start_date ≤ end_date）
  - 非法日期格式容错（静默回退为今天，返回 200）
  - 未来日期返回空 rows

五、group_by 参数
  - 默认值（不传）→ group_by 含 date 和 model
  - group_by=date → rows 含 date 字段
  - group_by=model → rows 含 ai_model_id、model_name 字段
  - group_by=instance → rows 含 instance_id、instance_name 字段
  - group_by=department → 200，group_by 含 department，rows 含 department 字段
  - group_by=group → 200，group_by 含 group，rows 含 group_id、group_name 字段
  - group_by=user（唯一维度，被删除后为空 map）→ 200，group_by 为空数组，rows 只有 token 汇总
  - group_by=user,model（user 被删，model 保留）→ 200，group_by 只含 model
  - group_by=date,model（组合）→ 200，group_by 含 date 和 model
  - 无效 group_by 值 → 回退到默认 date,model

六、order_by 参数
  - 默认（不传）→ 200 正常返回
  - order_by=total_tokens&order=desc → rows 按 total_tokens 降序
  - order_by=request_count&order=desc → rows 按 request_count 降序
  - order=asc（不传 order 或传 asc）→ 200 正常返回
  - order_by=invalid_field → 400 Bad Request

七、可选过滤参数
  - ai_model_id 过滤（不存在的 ID → 空 rows）
  - instance_id 过滤（不存在的 ID → 空 rows）
  - group_id 过滤（不存在的 ID → 空 rows）

八、quota_day 和 quota_period 字段
  - 不传 group_id：quota_day 为用户自身配额（整数 ≥ 0），quota_period 为 "day" 或 "month"
  - 传有效 group_id：quota_period 值为 "day" 或 "month" 之一
  - 传不存在的 group_id：quota_day 回退到站点默认值，quota_period 为 "day" 或 "month"
"""

import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))
from helpers.api import (
    SESSION_COOKIE, SESSION_COOKIE_B,
    health_check, cookie_header, bearer_header, user_headers,
    assert_fields, assert_sorted, assert_status, run_tests,
    ApiClient,
    TODAY, YESTERDAY, FUTURE,
)

# 优先使用 Bearer Token，其次 Session Cookie
HEADERS = user_headers()


def get_data(params: dict = None, headers: dict = None):
    h = headers if headers is not None else HEADERS
    tmp = ApiClient("", timeout=10)
    return tmp.get("/quota/data", params=params, expect=None, raw=True, extra_headers=h)


# ─────────────────────────────────────────────
# 一、认证鉴权
# ─────────────────────────────────────────────

def test_no_cookie_rejected():
    print(">>> [认证] 无认证信息访问 → 401 ...")
    resp = get_data(headers={"Accept": "application/json"})
    assert resp.status_code in (401, 403), \
        f"Expected 401/403 without auth, got {resp.status_code}, body={resp.text}"
    print(f"    OK (status={resp.status_code})")


def test_wrong_cookie_rejected():
    print(">>> [认证] 错误认证信息访问 → 401 ...")
    resp = get_data(headers={"Authorization": "Bearer wrong-token-that-does-not-exist", "Accept": "application/json"})
    assert resp.status_code in (401, 403), \
        f"Expected 401/403 with wrong token, got {resp.status_code}, body={resp.text}"
    print(f"    OK (status={resp.status_code})")


def test_valid_cookie_ok():
    print(">>> [认证] 有效认证信息正常访问 → 200 ...")
    resp = get_data()
    assert resp.status_code == 200, \
        f"Expected 200 with valid auth, got {resp.status_code}, body={resp.text}"
    print(f"    OK (status=200)")


# ─────────────────────────────────────────────
# 二、响应结构验证
# ─────────────────────────────────────────────

def test_response_structure():
    print(">>> [响应结构] 顶层字段完整性 ...")
    resp = get_data({"start_date": TODAY, "end_date": TODAY})
    resp.raise_for_status()
    data = resp.json()

    top_fields = ("start_date", "end_date", "group_by", "rows", "quota_day", "quota_period")
    for field in top_fields:
        assert field in data, f"响应缺少顶层字段: {field}，实际响应: {data}"

    assert isinstance(data["group_by"], list), \
        f"group_by 应为 list，实际类型: {type(data['group_by'])}"
    assert isinstance(data["rows"], list), \
        f"rows 应为 list，实际类型: {type(data['rows'])}"
    assert isinstance(data["quota_day"], int), \
        f"quota_day 应为 int，实际类型: {type(data['quota_day'])}"
    assert data["quota_period"] in ("day", "month"), \
        f"quota_period 应为 'day' 或 'month'，实际值: {data['quota_period']}"

    rows = data["rows"]
    if len(rows) > 0:
        row = rows[0]
        for field in ("total_tokens", "prompt_tokens", "completion_tokens", "request_count"):
            assert field in row, f"rows[0] 缺少字段: {field}"

    print(f"    OK (group_by={data['group_by']}, rows_count={len(rows)}, "
          f"quota_day={data['quota_day']}, quota_period={data['quota_period']})")


# ─────────────────────────────────────────────
# 三、数据隔离
# ─────────────────────────────────────────────

def test_data_isolation_user_id_param_ignored():
    """接口强制以当前登录用户过滤，传入 user_id 参数应被忽略（不报错，但数据仍为当前用户）"""
    print(">>> [数据隔离] 传入 user_id 参数应被忽略，数据仍为当前用户 ...")
    # 传一个不存在的 user_id，如果接口真的用了这个参数，rows 应为空
    # 但接口强制用当前用户，所以结果应与不传 user_id 一致
    params_base = {"start_date": TODAY, "end_date": TODAY, "group_by": "model"}
    resp_with = get_data({**params_base, "user_id": "999999999"})
    resp_without = get_data(params_base)
    resp_with.raise_for_status()
    resp_without.raise_for_status()

    data_with = resp_with.json()
    data_without = resp_without.json()
    rows_with = data_with.get("rows") or []
    rows_without = data_without.get("rows") or []

    # 数量必须相同
    assert len(rows_with) == len(rows_without), \
        f"user_id 参数应被忽略，但 rows 数量不同: with={len(rows_with)}, without={len(rows_without)}"

    # total_tokens 汇总必须相同（内容一致，不仅仅是数量）
    total_with = sum(r.get("total_tokens", 0) for r in rows_with)
    total_without = sum(r.get("total_tokens", 0) for r in rows_without)
    assert total_with == total_without, \
        f"user_id 参数应被忽略，但 total_tokens 汇总不同: with={total_with}, without={total_without}"

    print(f"    OK (rows_count={len(rows_without)}, total_tokens={total_without}, user_id 参数已被忽略)")


def test_data_isolation_between_users():
    """用户 A 和用户 B 的数据互相隔离"""
    print(">>> [数据隔离] 用户 A 和用户 B 数据互相隔离 ...")
    if not SESSION_COOKIE_B:
        print("    SKIP (SESSION_COOKIE_B 未设置)")
        return

    # 使用 group_by=model 方便比较各自的模型用量
    params = {"start_date": TODAY, "end_date": TODAY, "group_by": "model"}
    resp_a = get_data(params=params, headers=cookie_header(SESSION_COOKIE))
    resp_b = get_data(params=params, headers=cookie_header(SESSION_COOKIE_B))
    resp_a.raise_for_status()
    resp_b.raise_for_status()

    data_a = resp_a.json()
    data_b = resp_b.json()
    rows_a = data_a.get("rows") or []
    rows_b = data_b.get("rows") or []

    print(f"    用户A rows_count={len(rows_a)}, quota_day={data_a.get('quota_day')}")
    print(f"    用户B rows_count={len(rows_b)}, quota_day={data_b.get('quota_day')}")

    # 核心断言：用户 A 的 ai_model_id 集合与用户 B 的不应完全重叠（如果双方都有数据）
    # 至少验证：两个用户看到的 quota_day 是各自独立的（响应结构正确）
    assert "quota_day" in data_a, "用户A 响应缺少 quota_day 字段"
    assert "quota_day" in data_b, "用户B 响应缺少 quota_day 字段"

    # 如果两个用户都有数据，验证 ai_model_id 集合不完全相同（各自只看到自己的数据）
    if len(rows_a) > 0 and len(rows_b) > 0:
        ids_a = {r.get("ai_model_id") for r in rows_a}
        ids_b = {r.get("ai_model_id") for r in rows_b}
        # 两个用户的 total_tokens 汇总不应完全相同（除非恰好相等，概率极低）
        total_a = sum(r.get("total_tokens", 0) for r in rows_a)
        total_b = sum(r.get("total_tokens", 0) for r in rows_b)
        print(f"    用户A model_ids={ids_a}, total_tokens={total_a}")
        print(f"    用户B model_ids={ids_b}, total_tokens={total_b}")
        # 验证用户 B 的数据不包含用户 A 的 total_tokens（数据隔离）
        # 注意：如果两用户恰好使用了相同模型且 token 数相同，此断言可能误判，但这是合理的边界
        assert total_a != total_b or ids_a != ids_b, \
            "用户A 和用户B 的数据完全相同，可能存在数据隔离问题（或两用户数据恰好相同）"

    print("    OK (数据隔离验证通过)")


# ─────────────────────────────────────────────
# 四、日期过滤
# ─────────────────────────────────────────────

def test_default_date_range():
    print(f">>> [日期过滤] 默认日期范围（今天={TODAY}）...")
    resp = get_data()
    resp.raise_for_status()
    data = resp.json()
    assert data.get("start_date") == TODAY, \
        f"Expected start_date={TODAY}, got {data.get('start_date')}"
    assert data.get("end_date") == TODAY, \
        f"Expected end_date={TODAY}, got {data.get('end_date')}"
    print(f"    OK (start_date={data['start_date']}, end_date={data['end_date']})")


def test_explicit_date_range():
    print(f">>> [日期过滤] 明确指定日期范围（{YESTERDAY} ~ {TODAY}）...")
    resp = get_data({"start_date": YESTERDAY, "end_date": TODAY})
    resp.raise_for_status()
    data = resp.json()
    assert data.get("start_date") == YESTERDAY, \
        f"Expected start_date={YESTERDAY}, got {data.get('start_date')}"
    assert data.get("end_date") == TODAY, \
        f"Expected end_date={TODAY}, got {data.get('end_date')}"
    print(f"    OK (start_date={data['start_date']}, end_date={data['end_date']})")


def test_start_date_after_end_date_auto_swap():
    print(f">>> [日期过滤] start_date > end_date 时自动交换 ...")
    # 传入 start_date=未来, end_date=昨天，服务端应自动交换
    resp = get_data({"start_date": FUTURE, "end_date": YESTERDAY})
    assert resp.status_code == 200, \
        f"Expected 200 when start_date > end_date, got {resp.status_code}"
    data = resp.json()
    start = data.get("start_date")
    end = data.get("end_date")
    assert start <= end, \
        f"服务端应自动交换日期，但 start_date={start} > end_date={end}"
    print(f"    OK (自动交换后: start_date={start}, end_date={end})")


def test_invalid_date_format_fallback():
    print(">>> [日期过滤] 非法日期格式容错（静默回退为今天）...")
    resp = get_data({"start_date": "not-a-date", "end_date": "also-invalid"})
    assert resp.status_code == 200, \
        f"Expected 200 for invalid date format, got {resp.status_code}"
    data = resp.json()
    # 非法日期回退为今天
    assert data.get("start_date") == TODAY, \
        f"非法 start_date 应回退为今天 {TODAY}，实际: {data.get('start_date')}"
    assert data.get("end_date") == TODAY, \
        f"非法 end_date 应回退为今天 {TODAY}，实际: {data.get('end_date')}"
    print(f"    OK (start_date={data['start_date']}, end_date={data['end_date']})")


def test_future_date_no_data():
    print(f">>> [日期过滤] 未来日期返回空 rows（{FUTURE}）...")
    resp = get_data({"start_date": FUTURE, "end_date": FUTURE})
    resp.raise_for_status()
    rows = resp.json().get("rows") or []
    assert len(rows) == 0, \
        f"Expected empty rows for future date, got {len(rows)} rows"
    print(f"    OK (rows_count={len(rows)})")


# ─────────────────────────────────────────────
# 五、group_by 参数
# ─────────────────────────────────────────────

def test_group_by_default():
    print(">>> [group_by] 默认值（不传）→ group_by 含 date 和 model ...")
    resp = get_data()
    resp.raise_for_status()
    data = resp.json()
    group_by = data.get("group_by") or []
    assert "date" in group_by, \
        f"默认 group_by 应含 'date'，实际: {group_by}"
    assert "model" in group_by, \
        f"默认 group_by 应含 'model'，实际: {group_by}"
    print(f"    OK (group_by={group_by})")


def test_group_by_date():
    print(">>> [group_by] group_by=date → rows 含 date 字段 ...")
    resp = get_data({"group_by": "date"})
    resp.raise_for_status()
    data = resp.json()
    assert "date" in (data.get("group_by") or []), \
        f"group_by 应含 'date'，实际: {data.get('group_by')}"
    rows = data.get("rows") or []
    if len(rows) > 0:
        assert "date" in rows[0], \
            f"group_by=date 时 rows[0] 应含 'date' 字段，实际: {rows[0]}"
    print(f"    OK (rows_count={len(rows)})")


def test_group_by_model():
    print(">>> [group_by] group_by=model → rows 含 ai_model_id、model_name 字段 ...")
    resp = get_data({"group_by": "model"})
    resp.raise_for_status()
    data = resp.json()
    assert "model" in (data.get("group_by") or []), \
        f"group_by 应含 'model'，实际: {data.get('group_by')}"
    rows = data.get("rows") or []
    if len(rows) > 0:
        for field in ("ai_model_id", "model_name"):
            assert field in rows[0], \
                f"group_by=model 时 rows[0] 应含 '{field}' 字段，实际: {rows[0]}"
    print(f"    OK (rows_count={len(rows)})")


def test_group_by_instance():
    print(">>> [group_by] group_by=instance → rows 含 instance_id、instance_name 字段 ...")
    resp = get_data({"group_by": "instance"})
    resp.raise_for_status()
    data = resp.json()
    assert "instance" in (data.get("group_by") or []), \
        f"group_by 应含 'instance'，实际: {data.get('group_by')}"
    rows = data.get("rows") or []
    if len(rows) > 0:
        for field in ("instance_id", "instance_name"):
            assert field in rows[0], \
                f"group_by=instance 时 rows[0] 应含 '{field}' 字段，实际: {rows[0]}"
    print(f"    OK (rows_count={len(rows)})")


def test_group_by_department():
    print(">>> [group_by] group_by=department → 200，group_by 含 department ...")
    resp = get_data({"group_by": "department"})
    assert resp.status_code == 200, \
        f"Expected 200 for group_by=department, got {resp.status_code}, body={resp.text}"
    data = resp.json()
    group_by = data.get("group_by") or []
    assert "department" in group_by, \
        f"group_by 应含 'department'，实际: {group_by}"
    rows = data.get("rows") or []
    if len(rows) > 0:
        row = rows[0]
        for field in ("department_id", "department_name", "department_path"):
            assert field in row, \
                f"group_by=department 时 rows[0] 应含 '{field}' 字段，实际: {row}"
    print(f"    OK (group_by={group_by}, rows_count={len(rows)})")


def test_group_by_group():
    print(">>> [group_by] group_by=group → 200，group_by 含 group ...")
    resp = get_data({"group_by": "group"})
    assert resp.status_code == 200, \
        f"Expected 200 for group_by=group, got {resp.status_code}, body={resp.text}"
    data = resp.json()
    group_by = data.get("group_by") or []
    assert "group" in group_by, \
        f"group_by 应含 'group'，实际: {group_by}"
    rows = data.get("rows") or []
    if len(rows) > 0:
        row = rows[0]
        for field in ("group_id", "group_name", "group_full_path"):
            assert field in row, \
                f"group_by=group 时 rows[0] 应含 '{field}' 字段，实际: {row}"
    print(f"    OK (group_by={group_by}, rows_count={len(rows)})")


def test_group_by_user_only_deleted():
    """group_by=user 是唯一维度，被服务端删除后 groupBy 为空 map，
    不触发默认值回退，最终 group_by 为空数组，rows 只有 token 汇总"""
    print(">>> [group_by] group_by=user（唯一维度，被删除后为空 map）→ group_by 为空数组 ...")
    resp = get_data({"group_by": "user"})
    assert resp.status_code == 200, \
        f"Expected 200 for group_by=user, got {resp.status_code}, body={resp.text}"
    data = resp.json()
    group_by = data.get("group_by") or []
    assert "user" not in group_by, \
        f"group_by 不应含 'user'（已被服务端删除），实际: {group_by}"
    # group_by 应为空数组（不回退到默认 date,model）
    assert len(group_by) == 0, \
        f"group_by=user 被删除后应为空数组，实际: {group_by}"
    rows = data.get("rows") or []
    # rows 中不应有维度字段，只有 token 汇总
    if len(rows) > 0:
        row = rows[0]
        assert "user_id" not in row, \
            f"group_by=user 被删除后 rows[0] 不应含 'user_id'，实际: {row}"
        assert "date" not in row, \
            f"group_by=user 被删除后 rows[0] 不应含 'date'，实际: {row}"
    print(f"    OK (group_by={group_by}, rows_count={len(rows)})")


def test_group_by_user_model_user_deleted():
    """group_by=user,model 时，user 被删除，model 保留"""
    print(">>> [group_by] group_by=user,model → user 被删，group_by 只含 model ...")
    resp = get_data({"group_by": "user,model"})
    assert resp.status_code == 200, \
        f"Expected 200 for group_by=user,model, got {resp.status_code}, body={resp.text}"
    data = resp.json()
    group_by = data.get("group_by") or []
    assert "user" not in group_by, \
        f"group_by 不应含 'user'（已被服务端删除），实际: {group_by}"
    assert "model" in group_by, \
        f"group_by 应含 'model'，实际: {group_by}"
    print(f"    OK (group_by={group_by})")


def test_group_by_combined_date_model():
    print(">>> [group_by] group_by=date,model（组合）→ group_by 含 date 和 model ...")
    resp = get_data({"group_by": "date,model"})
    resp.raise_for_status()
    data = resp.json()
    group_by = data.get("group_by") or []
    assert "date" in group_by, \
        f"group_by 应含 'date'，实际: {group_by}"
    assert "model" in group_by, \
        f"group_by 应含 'model'，实际: {group_by}"
    print(f"    OK (group_by={group_by})")


def test_group_by_invalid_falls_back_to_default():
    print(">>> [group_by] 无效 group_by 值 → 回退到默认 date,model ...")
    resp = get_data({"group_by": "invalid_dimension"})
    resp.raise_for_status()
    data = resp.json()
    group_by = data.get("group_by") or []
    assert "date" in group_by, \
        f"无效 group_by 应回退到默认含 'date'，实际: {group_by}"
    assert "model" in group_by, \
        f"无效 group_by 应回退到默认含 'model'，实际: {group_by}"
    print(f"    OK (group_by={group_by})")


# ─────────────────────────────────────────────
# 六、order_by 参数
# ─────────────────────────────────────────────

def test_order_by_default():
    print(">>> [order_by] 默认（不传）→ 200 正常返回 ...")
    resp = get_data()
    assert resp.status_code == 200, \
        f"Expected 200 for default order_by, got {resp.status_code}"
    print(f"    OK (status=200)")


def test_order_by_total_tokens_desc():
    print(">>> [order_by] order_by=total_tokens&order=desc → rows 按 total_tokens 降序 ...")
    resp = get_data({"group_by": "model", "order_by": "total_tokens", "order": "desc"})
    resp.raise_for_status()
    rows = resp.json().get("rows") or []
    if len(rows) >= 2:
        assert_sorted(rows, "total_tokens", reverse=True)
    print(f"    OK (rows_count={len(rows)})")


def test_order_by_request_count_desc():
    print(">>> [order_by] order_by=request_count&order=desc → rows 按 request_count 降序 ...")
    resp = get_data({"group_by": "model", "order_by": "request_count", "order": "desc"})
    resp.raise_for_status()
    rows = resp.json().get("rows") or []
    if len(rows) >= 2:
        assert_sorted(rows, "request_count", reverse=True)
    print(f"    OK (rows_count={len(rows)})")


def test_order_asc():
    print(">>> [order_by] order=asc（升序/无序）→ 200 正常返回 ...")
    resp = get_data({"order_by": "total_tokens", "order": "asc"})
    assert resp.status_code == 200, \
        f"Expected 200 for order=asc, got {resp.status_code}"
    print(f"    OK (status=200)")


def test_order_not_specified():
    print(">>> [order_by] 不传 order 参数 → 200 正常返回 ...")
    resp = get_data({"order_by": "total_tokens"})
    assert resp.status_code == 200, \
        f"Expected 200 when order not specified, got {resp.status_code}"
    print(f"    OK (status=200)")


def test_order_by_invalid():
    print(">>> [order_by] order_by=invalid_field → 400 Bad Request ...")
    resp = get_data({"order_by": "invalid_field"})
    assert resp.status_code == 400, \
        f"Expected 400 for invalid order_by, got {resp.status_code}, body={resp.text}"
    data = resp.json()
    assert "error" in data, \
        f"400 响应应含 'error' 字段，实际: {data}"
    print(f"    OK (status=400, error={data.get('error')})")


# ─────────────────────────────────────────────
# 七、可选过滤参数
# ─────────────────────────────────────────────

def test_filter_ai_model_id():
    print(">>> [过滤] ai_model_id 过滤（不存在的 ID → 空 rows）...")
    resp = get_data({"ai_model_id": "999999999"})
    resp.raise_for_status()
    rows = resp.json().get("rows") or []
    assert len(rows) == 0, \
        f"Expected empty rows for non-existent ai_model_id, got {len(rows)}"
    print(f"    OK (rows_count={len(rows)})")


def test_filter_instance_id():
    print(">>> [过滤] instance_id 过滤（不存在的 ID → 空 rows）...")
    resp = get_data({"instance_id": "999999999"})
    resp.raise_for_status()
    rows = resp.json().get("rows") or []
    assert len(rows) == 0, \
        f"Expected empty rows for non-existent instance_id, got {len(rows)}"
    print(f"    OK (rows_count={len(rows)})")


def test_filter_group_id():
    print(">>> [过滤] group_id 过滤（不存在的 ID → 空 rows）...")
    resp = get_data({"group_id": "999999999"})
    resp.raise_for_status()
    rows = resp.json().get("rows") or []
    assert len(rows) == 0, \
        f"Expected empty rows for non-existent group_id, got {len(rows)}"
    print(f"    OK (rows_count={len(rows)})")


# ─────────────────────────────────────────────
# 八、quota_day 和 quota_period 字段
# ─────────────────────────────────────────────

def test_quota_fields_without_group_id():
    print(">>> [配额字段] 不传 group_id → quota_day 为用户自身配额，quota_period 合法 ...")
    resp = get_data()
    resp.raise_for_status()
    data = resp.json()
    quota_day = data.get("quota_day")
    quota_period = data.get("quota_period")
    assert isinstance(quota_day, int) and quota_day >= 0, \
        f"quota_day 应为非负整数，实际: {quota_day}"
    assert quota_period in ("day", "month"), \
        f"quota_period 应为 'day' 或 'month'，实际: {quota_period}"
    print(f"    OK (quota_day={quota_day}, quota_period={quota_period})")


def test_quota_fields_with_valid_group_id():
    print(">>> [配额字段] 传有效 group_id → quota_period 为 'day' 或 'month' ...")
    # 先查询一次获取数据，如果有 group 数据则取第一个 group_id
    resp_group = get_data({"group_by": "group"})
    resp_group.raise_for_status()
    rows = resp_group.json().get("rows") or []
    if not rows:
        print("    SKIP (当前用户无分组数据)")
        return

    group_id = rows[0].get("group_id")
    if not group_id:
        print("    SKIP (无法获取有效 group_id)")
        return

    resp = get_data({"group_id": str(group_id)})
    resp.raise_for_status()
    data = resp.json()
    quota_period = data.get("quota_period")
    assert quota_period in ("day", "month"), \
        f"quota_period 应为 'day' 或 'month'，实际: {quota_period}"
    print(f"    OK (group_id={group_id}, quota_day={data.get('quota_day')}, quota_period={quota_period})")


def test_quota_fields_with_nonexistent_group_id():
    print(">>> [配额字段] 传不存在的 group_id → quota_day 回退到站点默认值，quota_period 合法 ...")
    # group_id=999999999 不存在，ResolvePolicyIntForGroup 会回退到站点默认值
    resp = get_data({"group_id": "999999999"})
    resp.raise_for_status()
    data = resp.json()
    quota_day = data.get("quota_day")
    quota_period = data.get("quota_period")
    assert isinstance(quota_day, int) and quota_day >= -1, \
        f"quota_day 应为整数（-1 表示无限制），实际: {quota_day}"
    assert quota_period in ("day", "month"), \
        f"quota_period 应为 'day' 或 'month'，实际: {quota_period}"
    print(f"    OK (quota_day={quota_day}, quota_period={quota_period})")


# ─────────────────────────────────────────────
# Main
# ─────────────────────────────────────────────

def main():
    health_check()
    print()
    run_tests(globals(), title="GET /quota/data")


if __name__ == "__main__":
    main()
