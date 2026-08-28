#!/usr/bin/env python3
"""
集成测试：/admin/usage/logs 接口

覆盖场景：
  - 基础场景：无数据返回空列表、有数据正常返回
  - 日期过滤：默认今天、指定范围、范围外数据不返回
  - 可选过滤参数：user_id / ai_model_id / instance_id / 多条件组合
  - 分页：page + page_size 正常分页、total 不受分页影响、不传 page_size 返回全部
  - 排序：默认按 created_at DESC
  - 权限：无 token 返回 401/403、非 admin token 返回 401/403

使用方式：
  export API=http://127.0.0.1:8080
  export ADMIN_TOKEN=your-admin-token
  python3 test_admin_usage_logs.py

可选：
  export NON_ADMIN_TOKEN=your-non-admin-token   # 用于权限测试，不设置则跳过
"""

import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))
from helpers.api import (
    API, seed, ApiClient,
    health_check, auth_test_suite, run_tests,
    TODAY, YESTERDAY, TOMORROW,
)

# 兼容原变量名
ADMIN_HEADERS = None  # unused, kept for reference

def get_logs(params=None, headers=None):
    if headers:
        tmp = ApiClient("", timeout=30)
        return tmp.get("/admin/usage/logs", params=params, expect=None, raw=True, extra_headers=headers)
    return seed.get("/admin/usage/logs", params=params, expect=None, raw=True)

# 轻量 ok/fail —— 兼容本文件中已有的测试写法
PASS = 0
FAIL = 0


def ok(name: str):
    global PASS
    PASS += 1
    print(f"  ✓  {name}")


def fail(name: str, reason: str):
    global FAIL
    FAIL += 1
    print(f"  ✗  {name}")
    print(f"       原因: {reason}")


# ─────────────────────────────────────────────
# 基础场景
# ─────────────────────────────────────────────

def test_empty_logs():
    """无数据时返回空 logs 列表，total=0（使用远古日期确保无数据）"""
    name = "基础场景 - 无数据返回空列表"
    # 使用一个极早的日期范围，确保该区间不会有任何日志
    resp = get_logs({"start_date": "2000-01-01", "end_date": "2000-01-01"})
    if resp.status_code != 200:
        fail(name, f"期望 200，实际 {resp.status_code}，body={resp.text}")
        return
    data = resp.json()
    if data.get("total", -1) != 0:
        fail(name, f"期望 total=0，实际 total={data.get('total')}")
        return
    if data.get("logs") != [] and data.get("logs") is not None and len(data.get("logs", [])) != 0:
        fail(name, f"期望 logs 为空列表，实际 logs={data.get('logs')}")
        return
    ok(name)


def test_has_logs():
    """有数据时正常返回，字段结构正确"""
    name = "基础场景 - 有数据正常返回字段结构"
    resp = get_logs({"start_date": TODAY, "end_date": TODAY})
    if resp.status_code != 200:
        fail(name, f"期望 200，实际 {resp.status_code}，body={resp.text}")
        return
    data = resp.json()
    # 验证顶层字段存在
    for field in ("start_date", "end_date", "page", "page_size", "total", "logs"):
        if field not in data:
            fail(name, f"响应缺少字段: {field}")
            return
    logs = data.get("logs", [])
    if len(logs) > 0:
        row = logs[0]
        for field in ("id", "user_name", "provider", "model", "prompt_tokens",
                      "completion_tokens", "total_tokens", "status_code", "latency", "created_at"):
            if field not in row:
                fail(name, f"logs[0] 缺少字段: {field}")
                return
    ok(name)


# ─────────────────────────────────────────────
# 日期过滤
# ─────────────────────────────────────────────

def test_default_date_range():
    """不传日期参数，默认返回今天的数据（start_date/end_date 均为今天）"""
    name = "日期过滤 - 默认日期范围为今天"
    resp = get_logs()
    if resp.status_code != 200:
        fail(name, f"期望 200，实际 {resp.status_code}，body={resp.text}")
        return
    data = resp.json()
    if data.get("start_date") != TODAY:
        fail(name, f"期望 start_date={TODAY}，实际 {data.get('start_date')}")
        return
    if data.get("end_date") != TODAY:
        fail(name, f"期望 end_date={TODAY}，实际 {data.get('end_date')}")
        return
    ok(name)


def test_date_range_filter():
    """指定 start_date/end_date，响应中 start_date/end_date 与参数一致"""
    name = "日期过滤 - 指定日期范围响应字段正确"
    resp = get_logs({"start_date": YESTERDAY, "end_date": TODAY})
    if resp.status_code != 200:
        fail(name, f"期望 200，实际 {resp.status_code}，body={resp.text}")
        return
    data = resp.json()
    if data.get("start_date") != YESTERDAY:
        fail(name, f"期望 start_date={YESTERDAY}，实际 {data.get('start_date')}")
        return
    if data.get("end_date") != TODAY:
        fail(name, f"期望 end_date={TODAY}，实际 {data.get('end_date')}")
        return
    ok(name)


def test_future_date_no_data():
    """查询未来日期，应返回 total=0"""
    name = "日期过滤 - 未来日期无数据"
    resp = get_logs({"start_date": TOMORROW, "end_date": TOMORROW})
    if resp.status_code != 200:
        fail(name, f"期望 200，实际 {resp.status_code}，body={resp.text}")
        return
    data = resp.json()
    if data.get("total", -1) != 0:
        fail(name, f"未来日期应无数据，实际 total={data.get('total')}")
        return
    ok(name)


# ─────────────────────────────────────────────
# 可选过滤参数
# ─────────────────────────────────────────────

def test_filter_user_id():
    """user_id 过滤：传入不存在的 user_id，应返回 total=0"""
    name = "可选过滤 - user_id 过滤（不存在的用户）"
    resp = get_logs({"user_id": "999999999"})
    if resp.status_code != 200:
        fail(name, f"期望 200，实际 {resp.status_code}，body={resp.text}")
        return
    data = resp.json()
    if data.get("total", -1) != 0:
        fail(name, f"不存在的 user_id 应返回 total=0，实际 total={data.get('total')}")
        return
    ok(name)


def test_filter_ai_model_id():
    """ai_model_id 过滤：传入不存在的 ai_model_id，应返回 total=0"""
    name = "可选过滤 - ai_model_id 过滤（不存在的模型）"
    resp = get_logs({"ai_model_id": "999999999"})
    if resp.status_code != 200:
        fail(name, f"期望 200，实际 {resp.status_code}，body={resp.text}")
        return
    data = resp.json()
    if data.get("total", -1) != 0:
        fail(name, f"不存在的 ai_model_id 应返回 total=0，实际 total={data.get('total')}")
        return
    ok(name)


def test_filter_instance_id():
    """instance_id 过滤：传入不存在的 instance_id，应返回 total=0"""
    name = "可选过滤 - instance_id 过滤（不存在的实例）"
    resp = get_logs({"instance_id": "999999999"})
    if resp.status_code != 200:
        fail(name, f"期望 200，实际 {resp.status_code}，body={resp.text}")
        return
    data = resp.json()
    if data.get("total", -1) != 0:
        fail(name, f"不存在的 instance_id 应返回 total=0，实际 total={data.get('total')}")
        return
    ok(name)


def test_filter_multi_conditions():
    """多个过滤条件组合：同时传 user_id + ai_model_id，均不存在时返回 total=0"""
    name = "可选过滤 - 多条件组合（均不存在）"
    resp = get_logs({"user_id": "999999999", "ai_model_id": "999999999", "instance_id": "999999999"})
    if resp.status_code != 200:
        fail(name, f"期望 200，实际 {resp.status_code}，body={resp.text}")
        return
    data = resp.json()
    if data.get("total", -1) != 0:
        fail(name, f"多条件均不存在应返回 total=0，实际 total={data.get('total')}")
        return
    ok(name)


# ─────────────────────────────────────────────
# 分页
# ─────────────────────────────────────────────

def test_pagination_params_reflected():
    """page + page_size 参数被正确反映在响应中"""
    name = "分页 - page/page_size 参数反映在响应中"
    resp = get_logs({"page": "2", "page_size": "10"})
    if resp.status_code != 200:
        fail(name, f"期望 200，实际 {resp.status_code}，body={resp.text}")
        return
    data = resp.json()
    if data.get("page") != 2:
        fail(name, f"期望 page=2，实际 {data.get('page')}")
        return
    if data.get("page_size") != 10:
        fail(name, f"期望 page_size=10，实际 {data.get('page_size')}")
        return
    ok(name)


def test_total_not_affected_by_pagination():
    """total 字段反映过滤后总数，不受分页影响"""
    name = "分页 - total 不受分页影响"
    # 获取全量 total
    resp_all = get_logs({"page_size": "200"})
    if resp_all.status_code != 200:
        fail(name, f"期望 200，实际 {resp_all.status_code}，body={resp_all.text}")
        return
    total_all = resp_all.json().get("total", 0)

    # 分页查询 page=1, page_size=1
    resp_paged = get_logs({"page": "1", "page_size": "1"})
    if resp_paged.status_code != 200:
        fail(name, f"期望 200，实际 {resp_paged.status_code}，body={resp_paged.text}")
        return
    total_paged = resp_paged.json().get("total", -1)

    if total_all != total_paged:
        fail(name, f"total 应不受分页影响：全量 total={total_all}，分页 total={total_paged}")
        return
    ok(name)


def test_no_page_size_returns_all():
    """不传 page_size 时，logs 数量等于 total（返回全部数据）"""
    name = "分页 - 不传 page_size 返回全部数据"
    resp = get_logs()
    if resp.status_code != 200:
        fail(name, f"期望 200，实际 {resp.status_code}，body={resp.text}")
        return
    data = resp.json()
    total = data.get("total", 0)
    logs_count = len(data.get("logs", []))
    if total > 0 and logs_count != total:
        fail(name, f"不传 page_size 时 logs 数量({logs_count})应等于 total({total})")
        return
    ok(name)


def test_page_size_limits_logs():
    """传入 page_size=1 时，logs 数量不超过 1"""
    name = "分页 - page_size=1 时 logs 数量不超过 1"
    resp = get_logs({"page": "1", "page_size": "1"})
    if resp.status_code != 200:
        fail(name, f"期望 200，实际 {resp.status_code}，body={resp.text}")
        return
    data = resp.json()
    logs_count = len(data.get("logs", []))
    if logs_count > 1:
        fail(name, f"page_size=1 时 logs 数量应 ≤ 1，实际 {logs_count}")
        return
    ok(name)


# ─────────────────────────────────────────────
# 排序
# ─────────────────────────────────────────────

def test_default_order_desc():
    """默认按 created_at DESC 排序：第一条记录的 created_at >= 最后一条"""
    name = "排序 - 默认按 created_at DESC"
    resp = get_logs({"page_size": "50"})
    if resp.status_code != 200:
        fail(name, f"期望 200，实际 {resp.status_code}，body={resp.text}")
        return
    data = resp.json()
    logs = data.get("logs", [])
    if len(logs) < 2:
        ok(name)  # 数据不足 2 条，无法验证排序，视为通过
        return
    first_ts = logs[0].get("created_at", "")
    last_ts = logs[-1].get("created_at", "")
    if first_ts < last_ts:
        fail(name, f"期望降序排列，但 first={first_ts} < last={last_ts}")
        return
    ok(name)


# ─────────────────────────────────────────────
# 权限
# ─────────────────────────────────────────────

def test_auth():
    """认证测试三件套"""
    auth_test_suite(lambda headers: get_logs(headers=headers), label="usage/logs")


# ─────────────────────────────────────────────
# 主入口
# ─────────────────────────────────────────────

def main():
    health_check()
    print(f"目标服务: {API}")
    print(f"测试日期: {TODAY}")
    print()

    # 使用自定义 ok/fail 计数，手动汇总
    test_empty_logs()
    test_has_logs()
    test_default_date_range()
    test_date_range_filter()
    test_future_date_no_data()
    test_filter_user_id()
    test_filter_ai_model_id()
    test_filter_instance_id()
    test_filter_multi_conditions()
    test_pagination_params_reflected()
    test_total_not_affected_by_pagination()
    test_no_page_size_returns_all()
    test_page_size_limits_logs()
    test_default_order_desc()
    test_auth()

    print()
    print(f"{'='*40}")
    print(f"结果: {PASS} 通过 / {FAIL} 失败")
    if FAIL > 0:
        sys.exit(1)
    print("所有测试通过 ✓")


if __name__ == "__main__":
    main()
