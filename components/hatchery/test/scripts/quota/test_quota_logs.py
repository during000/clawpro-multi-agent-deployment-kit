#!/usr/bin/env python3
"""
测试脚本：验证 GET /quota/logs 接口

覆盖场景：

一、认证鉴权
  - 无 Cookie/Token 访问 → 401/403
  - 错误 Cookie 访问 → 401/403
  - 有效 Session Cookie 正常访问 → 200
  - 用户 Bearer Token 正常访问 → 200，且数据隔离生效

二、响应结构验证
  - 顶层字段完整性：start_date、end_date、page、page_size、total、logs
  - logs 行字段完整性：id、provider、model、prompt_tokens、completion_tokens、total_tokens、status_code、latency、created_at
  - 响应中不含 user_name 字段（与 /admin/usage/logs 的关键区别）

三、数据隔离（核心）
  - user_id 参数被忽略，数据仍为当前用户（强制 user_id = 当前用户）
  - 用户 A 和用户 B 数据互相隔离（需 SESSION_COOKIE_B）

四、日期过滤
  - 默认日期范围（今天）
  - 明确指定日期范围
  - start_date > end_date 时自动交换（返回 200，start_date ≤ end_date）
  - 非法日期格式容错（静默回退为今天，返回 200）
  - 未来日期返回 total=0

五、可选过滤参数
  - ai_model_id 过滤（不存在的 ID → total=0）
  - instance_id 过滤（不存在的 ID → total=0）
  - group_id 过滤（不存在的 ID → total=0，/admin/usage/logs 没有此参数）
  - 多条件组合过滤（均不存在 → total=0）

六、分页
  - page/page_size 参数反映在响应中
  - total 不受分页影响
  - 不传 page_size 返回全部数据（logs 数量等于 total）
  - page_size 限制返回数量（page_size=1 时 logs ≤ 1）
  - 传 page_size 不传 page → 响应 page=1（服务端修正）
  - 不传 page_size 时 → 响应 page=0（未触发修正）
  - page 超出范围 → logs 为空，total 不变
  - page_size=0 或负数 → 等同于不传，返回全部数据

七、排序
  - 固定按 created_at DESC（第一条 created_at >= 最后一条）

八、边界场景
  - 无数据时返回空 logs 列表，total=0

使用方式：
  export API=http://127.0.0.1:9000
  export SESSION_COOKIE='session=xxx'
  python3 test_quota_logs.py

可选：
  export SESSION_COOKIE_B='session=yyy'   # 用于数据隔离测试
  export USER_TOKEN=hk-xxx                # 用于 Bearer Token 认证测试
"""

import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))
from helpers.api import (
    TOKEN, SESSION_COOKIE, SESSION_COOKIE_B,
    health_check, cookie_header, bearer_header, user_headers,
    assert_fields, assert_status, run_tests,
    ApiClient,
    TODAY, YESTERDAY, FUTURE,
)

# USER_TOKEN 为额外的用户 token 认证测试
USER_TOKEN = os.environ.get("USER_TOKEN", "")

# 优先使用 Bearer Token，其次 Session Cookie
HEADERS = user_headers()


def get_logs(params: dict = None, headers: dict = None):
    h = headers if headers is not None else HEADERS
    tmp = ApiClient("", timeout=10)
    return tmp.get("/quota/logs", params=params, expect=None, raw=True, extra_headers=h)


# ─────────────────────────────────────────────
# 一、认证鉴权
# ─────────────────────────────────────────────

def test_no_cookie_rejected():
    print(">>> [认证] 无 Cookie/Token 访问 → 401/403 ...")
    resp = get_logs(headers={"Accept": "application/json"})
    assert resp.status_code in (401, 403), \
        f"Expected 401/403 without cookie, got {resp.status_code}, body={resp.text}"
    print(f"    OK (status={resp.status_code})")


def test_wrong_cookie_rejected():
    print(">>> [认证] 错误认证信息访问 → 401/403 ...")
    resp = get_logs(headers={"Authorization": "Bearer wrong-token-that-does-not-exist", "Accept": "application/json"})
    assert resp.status_code in (401, 403), \
        f"Expected 401/403 with wrong token, got {resp.status_code}, body={resp.text}"
    print(f"    OK (status={resp.status_code})")


def test_valid_cookie_ok():
    print(">>> [认证] 有效认证信息正常访问 → 200 ...")
    resp = get_logs()
    assert resp.status_code == 200, \
        f"Expected 200 with valid auth, got {resp.status_code}, body={resp.text}"
    print("    OK (status=200)")


def test_bearer_token_ok():
    print(">>> [认证] 用户 Bearer Token 正常访问 → 200，数据隔离生效 ...")
    if not USER_TOKEN and not TOKEN:
        print("    SKIP (USER_TOKEN/TOKEN 未设置)")
        return
    test_token = USER_TOKEN or TOKEN
    resp = get_logs(headers=bearer_header(test_token))
    assert resp.status_code == 200, \
        f"Expected 200 with user bearer token, got {resp.status_code}, body={resp.text}"
    data = resp.json()
    # 验证响应结构正确
    for field in ("start_date", "end_date", "page", "page_size", "total", "logs"):
        assert field in data, f"Bearer Token 响应缺少字段: {field}"
    print(f"    OK (status=200, total={data.get('total')})")


# ─────────────────────────────────────────────
# 二、响应结构验证
# ─────────────────────────────────────────────

def test_response_top_fields():
    print(">>> [响应结构] 顶层字段完整性 ...")
    resp = get_logs({"start_date": TODAY, "end_date": TODAY})
    resp.raise_for_status()
    data = resp.json()

    for field in ("start_date", "end_date", "page", "page_size", "total", "logs"):
        assert field in data, f"响应缺少顶层字段: {field}，实际响应: {data}"

    assert isinstance(data["total"], int) and data["total"] >= 0, \
        f"total 应为非负整数，实际: {data['total']}"
    assert isinstance(data["logs"], list), \
        f"logs 应为 list，实际类型: {type(data['logs'])}"

    print(f"    OK (total={data['total']}, page={data['page']}, page_size={data['page_size']})")


def test_response_log_row_fields():
    print(">>> [响应结构] logs 行字段完整性 ...")
    resp = get_logs({"page_size": "1"})
    resp.raise_for_status()
    data = resp.json()
    logs = data.get("logs") or []
    if len(logs) == 0:
        print("    SKIP (当前用户无日志数据)")
        return

    row = logs[0]
    for field in ("id", "provider", "model", "prompt_tokens", "completion_tokens",
                  "total_tokens", "status_code", "latency", "created_at"):
        assert field in row, f"logs[0] 缺少字段: {field}，实际: {row}"

    # 关键区别：响应中不含 user_name（/admin/usage/logs 才有）
    assert "user_name" not in row, \
        f"logs[0] 不应含 'user_name' 字段（普通用户接口不暴露），实际: {row}"

    print(f"    OK (row keys={list(row.keys())})")


def test_response_no_user_name_field():
    print(">>> [响应结构] 响应中不含 user_name 字段（与 /admin/usage/logs 的关键区别）...")
    resp = get_logs({"page_size": "5"})
    resp.raise_for_status()
    logs = resp.json().get("logs") or []
    for i, row in enumerate(logs):
        assert "user_name" not in row, \
            f"logs[{i}] 不应含 'user_name' 字段，实际: {row}"
    print(f"    OK (验证了 {len(logs)} 条记录均无 user_name 字段)")


# ─────────────────────────────────────────────
# 三、数据隔离
# ─────────────────────────────────────────────

def test_user_id_param_ignored():
    """接口强制以当前登录用户过滤，传入 user_id 参数应被忽略"""
    print(">>> [数据隔离] 传入 user_id 参数应被忽略，数据仍为当前用户 ...")
    resp_base = get_logs({"start_date": TODAY, "end_date": TODAY})
    resp_with_uid = get_logs({"start_date": TODAY, "end_date": TODAY, "user_id": "999999999"})
    resp_base.raise_for_status()
    resp_with_uid.raise_for_status()

    total_base = resp_base.json().get("total", -1)
    total_with_uid = resp_with_uid.json().get("total", -1)

    # 如果 user_id 被接受，不存在的 user_id 会导致 total=0；
    # 如果 user_id 被忽略，total 应与不传时一致
    assert total_base == total_with_uid, \
        f"user_id 参数应被忽略，但 total 不同: base={total_base}, with_user_id={total_with_uid}"
    print(f"    OK (total={total_base}，user_id 参数已被忽略)")


def test_data_isolation_between_users():
    """用户 A 和用户 B 的数据互相隔离"""
    print(">>> [数据隔离] 用户 A 和用户 B 数据互相隔离 ...")
    if not SESSION_COOKIE_B:
        print("    SKIP (SESSION_COOKIE_B 未设置)")
        return

    resp_a = get_logs({"start_date": TODAY, "end_date": TODAY},
                      headers=cookie_header(SESSION_COOKIE))
    resp_b = get_logs({"start_date": TODAY, "end_date": TODAY},
                      headers=cookie_header(SESSION_COOKIE_B))
    resp_a.raise_for_status()
    resp_b.raise_for_status()

    data_a = resp_a.json()
    data_b = resp_b.json()
    total_a = data_a.get("total", 0)
    total_b = data_b.get("total", 0)
    logs_a = data_a.get("logs") or []
    logs_b = data_b.get("logs") or []

    print(f"    用户A total={total_a}, 用户B total={total_b}")

    # 如果两个用户都有数据，验证 log id 集合不重叠（各自只看到自己的记录）
    if len(logs_a) > 0 and len(logs_b) > 0:
        ids_a = {r.get("id") for r in logs_a}
        ids_b = {r.get("id") for r in logs_b}
        overlap = ids_a & ids_b
        assert len(overlap) == 0, \
            f"用户A 和用户B 的日志 id 存在重叠，可能存在数据隔离问题，重叠 id: {overlap}"

    print("    OK (数据隔离验证通过)")


# ─────────────────────────────────────────────
# 四、日期过滤
# ─────────────────────────────────────────────

def test_default_date_range():
    print(f">>> [日期过滤] 默认日期范围（今天={TODAY}）...")
    resp = get_logs()
    resp.raise_for_status()
    data = resp.json()
    assert data.get("start_date") == TODAY, \
        f"Expected start_date={TODAY}, got {data.get('start_date')}"
    assert data.get("end_date") == TODAY, \
        f"Expected end_date={TODAY}, got {data.get('end_date')}"
    print(f"    OK (start_date={data['start_date']}, end_date={data['end_date']})")


def test_explicit_date_range():
    print(f">>> [日期过滤] 明确指定日期范围（{YESTERDAY} ~ {TODAY}）...")
    resp = get_logs({"start_date": YESTERDAY, "end_date": TODAY})
    resp.raise_for_status()
    data = resp.json()
    assert data.get("start_date") == YESTERDAY, \
        f"Expected start_date={YESTERDAY}, got {data.get('start_date')}"
    assert data.get("end_date") == TODAY, \
        f"Expected end_date={TODAY}, got {data.get('end_date')}"
    print(f"    OK (start_date={data['start_date']}, end_date={data['end_date']})")


def test_start_date_after_end_date_auto_swap():
    print(">>> [日期过滤] start_date > end_date 时自动交换 ...")
    resp = get_logs({"start_date": FUTURE, "end_date": YESTERDAY})
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
    resp = get_logs({"start_date": "not-a-date", "end_date": "also-invalid"})
    assert resp.status_code == 200, \
        f"Expected 200 for invalid date format, got {resp.status_code}"
    data = resp.json()
    assert data.get("start_date") == TODAY, \
        f"非法 start_date 应回退为今天 {TODAY}，实际: {data.get('start_date')}"
    assert data.get("end_date") == TODAY, \
        f"非法 end_date 应回退为今天 {TODAY}，实际: {data.get('end_date')}"
    print(f"    OK (start_date={data['start_date']}, end_date={data['end_date']})")


def test_future_date_no_data():
    print(f">>> [日期过滤] 未来日期返回 total=0（{FUTURE}）...")
    resp = get_logs({"start_date": FUTURE, "end_date": FUTURE})
    resp.raise_for_status()
    total = resp.json().get("total", -1)
    assert total == 0, f"Expected total=0 for future date, got total={total}"
    print(f"    OK (total={total})")


# ─────────────────────────────────────────────
# 五、可选过滤参数
# ─────────────────────────────────────────────

def test_filter_ai_model_id():
    print(">>> [过滤] ai_model_id 过滤（不存在的 ID → total=0）...")
    resp = get_logs({"ai_model_id": "999999999"})
    resp.raise_for_status()
    total = resp.json().get("total", -1)
    assert total == 0, \
        f"Expected total=0 for non-existent ai_model_id, got total={total}"
    print(f"    OK (total={total})")


def test_filter_instance_id():
    print(">>> [过滤] instance_id 过滤（不存在的 ID → total=0）...")
    resp = get_logs({"instance_id": "999999999"})
    resp.raise_for_status()
    total = resp.json().get("total", -1)
    assert total == 0, \
        f"Expected total=0 for non-existent instance_id, got total={total}"
    print(f"    OK (total={total})")


def test_filter_group_id():
    print(">>> [过滤] group_id 过滤（不存在的 ID → total=0，/admin/usage/logs 没有此参数）...")
    resp = get_logs({"group_id": "999999999"})
    resp.raise_for_status()
    total = resp.json().get("total", -1)
    assert total == 0, \
        f"Expected total=0 for non-existent group_id, got total={total}"
    print(f"    OK (total={total})")


def test_filter_multi_conditions():
    print(">>> [过滤] 多条件组合过滤（均不存在 → total=0）...")
    resp = get_logs({
        "ai_model_id": "999999999",
        "instance_id": "999999999",
        "group_id": "999999999",
    })
    resp.raise_for_status()
    total = resp.json().get("total", -1)
    assert total == 0, \
        f"Expected total=0 for all non-existent filter IDs, got total={total}"
    print(f"    OK (total={total})")


# ─────────────────────────────────────────────
# 六、分页
# ─────────────────────────────────────────────

def test_pagination_params_reflected():
    print(">>> [分页] page/page_size 参数反映在响应中 ...")
    resp = get_logs({"page": "2", "page_size": "10"})
    resp.raise_for_status()
    data = resp.json()
    assert data.get("page") == 2, \
        f"Expected page=2, got {data.get('page')}"
    assert data.get("page_size") == 10, \
        f"Expected page_size=10, got {data.get('page_size')}"
    print(f"    OK (page={data['page']}, page_size={data['page_size']})")


def test_total_not_affected_by_pagination():
    print(">>> [分页] total 不受分页影响 ...")
    resp_all = get_logs()
    resp_paged = get_logs({"page": "1", "page_size": "1"})
    resp_all.raise_for_status()
    resp_paged.raise_for_status()

    total_all = resp_all.json().get("total", 0)
    total_paged = resp_paged.json().get("total", -1)
    assert total_all == total_paged, \
        f"total 应不受分页影响: 全量 total={total_all}，分页 total={total_paged}"
    print(f"    OK (total={total_all}，分页前后一致)")


def test_no_page_size_returns_all():
    print(">>> [分页] 不传 page_size 返回全部数据（logs 数量等于 total）...")
    resp = get_logs()
    resp.raise_for_status()
    data = resp.json()
    total = data.get("total", 0)
    logs_count = len(data.get("logs") or [])
    if total > 0:
        assert logs_count == total, \
            f"不传 page_size 时 logs 数量({logs_count})应等于 total({total})"
    print(f"    OK (total={total}, logs_count={logs_count})")


def test_page_size_limits_logs():
    print(">>> [分页] page_size=1 时 logs 数量不超过 1 ...")
    resp = get_logs({"page": "1", "page_size": "1"})
    resp.raise_for_status()
    logs_count = len(resp.json().get("logs") or [])
    assert logs_count <= 1, \
        f"page_size=1 时 logs 数量应 ≤ 1，实际 {logs_count}"
    print(f"    OK (logs_count={logs_count})")


def test_page_default_when_page_size_set():
    """传 page_size 但不传 page → 服务端将 page 修正为 1，响应 page=1"""
    print(">>> [分页] 传 page_size 不传 page → 响应 page=1（服务端修正）...")
    resp = get_logs({"page_size": "10"})
    resp.raise_for_status()
    page = resp.json().get("page")
    assert page == 1, \
        f"传 page_size 不传 page 时，服务端应将 page 修正为 1，实际 page={page}"
    print(f"    OK (page={page})")


def test_page_zero_when_no_page_size():
    """不传 page_size 时，page 不会被修正，响应 page=0"""
    print(">>> [分页] 不传 page_size 时 → 响应 page=0（未触发修正）...")
    resp = get_logs()
    resp.raise_for_status()
    page = resp.json().get("page")
    assert page == 0, \
        f"不传 page_size 时 page 应为 0（未触发修正），实际 page={page}"
    print(f"    OK (page={page})")


def test_page_out_of_range():
    """page 超出范围 → logs 为空，total 不变"""
    print(">>> [分页] page 超出范围 → logs 为空，total 不变 ...")
    resp_all = get_logs()
    resp_all.raise_for_status()
    total = resp_all.json().get("total", 0)

    resp_oob = get_logs({"page": "9999", "page_size": "10"})
    resp_oob.raise_for_status()
    data_oob = resp_oob.json()
    logs_count = len(data_oob.get("logs") or [])
    total_oob = data_oob.get("total", -1)

    assert logs_count == 0, \
        f"page 超出范围时 logs 应为空，实际 logs_count={logs_count}"
    assert total_oob == total, \
        f"page 超出范围时 total 应不变: 全量 total={total}，超出范围 total={total_oob}"
    print(f"    OK (logs_count={logs_count}, total={total_oob})")


def test_page_size_zero_returns_all():
    """page_size=0 等同于不传，返回全部数据"""
    print(">>> [分页] page_size=0 等同于不传，返回全部数据 ...")
    resp_zero = get_logs({"page_size": "0"})
    resp_all = get_logs()
    resp_zero.raise_for_status()
    resp_all.raise_for_status()

    total_zero = resp_zero.json().get("total", -1)
    total_all = resp_all.json().get("total", -1)
    logs_zero = len(resp_zero.json().get("logs") or [])
    logs_all = len(resp_all.json().get("logs") or [])

    assert total_zero == total_all, \
        f"page_size=0 时 total 应与不传一致: zero={total_zero}, all={total_all}"
    assert logs_zero == logs_all, \
        f"page_size=0 时 logs 数量应与不传一致: zero={logs_zero}, all={logs_all}"
    print(f"    OK (total={total_all}, logs_count={logs_all})")


# ─────────────────────────────────────────────
# 七、排序
# ─────────────────────────────────────────────

def test_default_order_created_at_desc():
    print(">>> [排序] 固定按 created_at DESC（第一条 created_at >= 最后一条）...")
    resp = get_logs({"page_size": "50"})
    resp.raise_for_status()
    logs = resp.json().get("logs") or []
    if len(logs) < 2:
        print(f"    SKIP (数据不足 2 条，无法验证排序，当前 logs_count={len(logs)})")
        return
    first_ts = logs[0].get("created_at", "")
    last_ts = logs[-1].get("created_at", "")
    assert first_ts >= last_ts, \
        f"期望降序排列，但 first={first_ts} < last={last_ts}"
    print(f"    OK (first={first_ts}, last={last_ts})")


# ─────────────────────────────────────────────
# 八、边界场景
# ─────────────────────────────────────────────

def test_empty_logs_future_date():
    print(f">>> [边界] 无数据时返回空 logs 列表，total=0（查询未来日期）...")
    resp = get_logs({"start_date": FUTURE, "end_date": FUTURE})
    resp.raise_for_status()
    data = resp.json()
    total = data.get("total", -1)
    logs = data.get("logs") or []
    assert total == 0, f"Expected total=0, got total={total}"
    assert len(logs) == 0, f"Expected empty logs list, got {len(logs)} items"
    print(f"    OK (total={total}, logs=[])")


# ─────────────────────────────────────────────
# Main
# ─────────────────────────────────────────────

def main():
    health_check()
    print()
    run_tests(globals(), title="GET /quota/logs")


if __name__ == "__main__":
    main()
