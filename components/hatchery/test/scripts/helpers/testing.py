#!/usr/bin/env python3
"""
测试辅助工具

包含:
- 日期常量（TODAY / YESTERDAY / TOMORROW / FUTURE）
- 通用断言工具（assert_fields / assert_status / assert_sorted）
- 认证测试三件套（auth_test_suite）
- 健康检查 / 环境检查
- 站点配置工具
- Gateway 重启重试
- 通用辅助函数（get_field / ensure_gateway_ui_enabled / clear_instance_models）
- 用户管理小工具（pick_user / extract_uid / cleanup_users_by_prefix）
- 测试运行器（run_tests）
"""

import datetime as _dt
import inspect as _inspect
import os
import sys
import time
import traceback
from typing import Callable, TypeVar

import requests

from helpers import config
from helpers.client import (
    # 核心变量（需要在本模块中引用）
    API, ADMIN_TOKEN, HEADERS, NON_ADMIN_TOKEN,
    # 客户端工厂
    ApiClient, admin_client,
    # 预置全局客户端
    seed,
    # Header 工具
    no_auth_headers, wrong_token_headers, non_admin_headers,
    # 颜色工具
    GREEN, RED, YELLOW, CYAN, GRAY, BOLD,
    # 截断工具
    truncate,
    # 帧记录收集器
    _start_collecting, _stop_collecting,
)

T = TypeVar("T")


# ═══════════════════════════════════════════════════════════════════════════════
# 日期常量（UTC+8）
# ═══════════════════════════════════════════════════════════════════════════════

try:
    TZ_CN = _dt.timezone(_dt.timedelta(hours=8))
except Exception:
    TZ_CN = None

_now_cn = _dt.datetime.now(TZ_CN) if TZ_CN else _dt.datetime.now()
TODAY: str = _now_cn.strftime("%Y-%m-%d")
YESTERDAY: str = (_now_cn - _dt.timedelta(days=1)).strftime("%Y-%m-%d")
TOMORROW: str = (_now_cn + _dt.timedelta(days=1)).strftime("%Y-%m-%d")
FUTURE: str = "2099-12-31"


# ═══════════════════════════════════════════════════════════════════════════════
# 通用断言工具
# ═══════════════════════════════════════════════════════════════════════════════

def _last_frame_hint() -> str:
    """从帧收集器获取最近一帧的简要上下文（用于错误提示）"""
    from helpers.client import _frame_collector
    if not _frame_collector:
        return ""
    f = _frame_collector[-1]
    return (f"\n    ↳ 最近请求: [{f['method']} {f['path']}] "
            f"status={f['status_code']} ({f['duration_ms']}ms)")


def assert_fields(data: dict, fields, context: str = ""):
    """批量断言 dict 包含指定字段，不存在时给出清晰报错"""
    missing = [f for f in fields if f not in data]
    if missing:
        ctx = f" ({context})" if context else ""
        hint = _last_frame_hint()
        raise AssertionError(
            f"响应缺少字段{ctx}: {missing}，实际 keys={list(data.keys())}{hint}"
        )


def assert_status(resp, expected, label: str = ""):
    """断言响应状态码，支持单个 int 或集合"""
    if isinstance(expected, int):
        expected = {expected}
    if resp.status_code not in expected:
        ctx = f" [{label}]" if label else ""
        body_snippet = truncate(resp.text, 300) if hasattr(resp, 'text') else ""
        hint = _last_frame_hint()
        raise AssertionError(
            f"状态码不符{ctx}: 期望 {sorted(expected)}，"
            f"实际 {resp.status_code}，body={body_snippet}{hint}"
        )


def assert_sorted(rows: list, key: str, *, reverse: bool = True):
    """断言列表按指定字段有序"""
    if len(rows) < 2:
        return
    values = [r.get(key, 0) for r in rows]
    expected = sorted(values, reverse=reverse)
    if values != expected:
        order = "DESC" if reverse else "ASC"
        raise AssertionError(f"rows 未按 {key} {order} 排序: {values[:10]}")


# ═══════════════════════════════════════════════════════════════════════════════
# API 调用工厂（减少 admin_sg 等模块的 boilerplate）
# ═══════════════════════════════════════════════════════════════════════════════

def make_api_fn(method: str, path: str, *, default_client=None, timeout: int = 30):
    """
    生成标准 API 调用函数，自动支持 auth_test_suite 的 headers 参数。

    返回的函数签名: fn(body=None, params=None, headers=None) → requests.Response

    当传入 headers 时，使用一个空 token 的临时 ApiClient 并附加 extra_headers（
    用于认证测试），否则使用 default_client（默认 seed）。

    示例:
        bind_sg = make_api_fn("post", "/admin/config/security-group/bind")
        list_sg = make_api_fn("get", "/admin/config/security-group/list", timeout=15)
    """
    client = default_client or seed

    def fn(body=None, params=None, headers=None, *, raw_data=None):
        if raw_data is not None:
            # 用于发送非 JSON 的原始数据（测试格式错误场景）
            if headers:
                # 认证测试：使用空 token + 自定义 headers
                tmp = ApiClient("", timeout=timeout)
                return tmp.post(path, data=raw_data,
                                expect=None, raw=True, extra_headers=headers)
            # 正常场景：使用默认客户端（带管理员 token）发送非 JSON 数据
            return client.post(path, data=raw_data,
                               expect=None, timeout=timeout, raw=True)
        if headers:
            tmp = ApiClient("", timeout=timeout)
            call = getattr(tmp, method)
            return call(path, json=body, params=params,
                        expect=None, raw=True, extra_headers=headers)
        call = getattr(client, method)
        return call(path, json=body, params=params,
                    expect=None, timeout=timeout, raw=True)

    return fn


# ═══════════════════════════════════════════════════════════════════════════════
# 认证测试三件套
# ═══════════════════════════════════════════════════════════════════════════════

def auth_test_suite(api_call_fn, *, label: str = "", check_admin: bool = True):
    """
    生成标准认证测试三件套并立即执行：
      1. 无认证 → 401/403
      2. 错误 token → 401/403
      3. 非管理员 token → 401/403（若 NON_ADMIN_TOKEN 未设置则 SKIP）

    api_call_fn: callable(headers=dict) → requests.Response

    参数：
      check_admin:
        True（默认）  - 接口要求管理员权限；非管理员 token 应返回 401/403。
        False        - 接口为用户侧接口，不做管理员校验；跳过第 3 步。
                       例如 /openclaw/* 下的实例管理接口，对普通用户 token
                       会返回 200/4xx 业务错误而非 401/403。
    """
    prefix = f"[{label}] " if label else ""

    print(f">>> {prefix}认证测试 - 无认证信息 → 401/403 ...")
    resp = api_call_fn(headers=no_auth_headers())
    assert_status(resp, {401, 403}, label="无认证")
    print(f"    OK (status={resp.status_code})")

    print(f">>> {prefix}认证测试 - 错误 token → 401/403 ...")
    resp = api_call_fn(headers=wrong_token_headers())
    assert_status(resp, {401, 403}, label="错误token")
    print(f"    OK (status={resp.status_code})")

    if not check_admin:
        print(f">>> {prefix}认证测试 - 非管理员 token → SKIP (用户侧接口)")
        return

    print(f">>> {prefix}认证测试 - 非管理员 token → 401/403 ...")
    if not NON_ADMIN_TOKEN:
        print("    SKIP (NON_ADMIN_TOKEN 未设置)")
    else:
        resp = api_call_fn(headers=non_admin_headers())
        assert_status(resp, {401, 403}, label="非管理员")
        print(f"    OK (status={resp.status_code})")


# ═══════════════════════════════════════════════════════════════════════════════
# 健康检查
# ═══════════════════════════════════════════════════════════════════════════════

def health_check():
    """检查服务是否可用"""
    resp = requests.get(f"{config.BASE_URL}/health", timeout=10)
    resp.raise_for_status()
    data = resp.json()
    assert data.get("status") == "ok", f"健康检查失败: {data}"
    return data


# ═══════════════════════════════════════════════════════════════════════════════
# 站点配置
# ═══════════════════════════════════════════════════════════════════════════════

def admin_get_config(admin_token):
    """获取站点配置"""
    data = admin_client(admin_token).get("/admin/config")
    return data.get("config", data)


def admin_update_config(admin_token, **fields):
    """更新站点配置"""
    return admin_client(admin_token).post("/admin/config", data=fields)


# ═══════════════════════════════════════════════════════════════════════════════
# 环境检查
# ═══════════════════════════════════════════════════════════════════════════════

def check_env():
    """执行健康检查（环境变量已由 config.py 在导入时保证存在）"""
    print(">>> 健康检查 ...")
    health_check()
    print("    服务正常 ✓")


def require_model_config():
    """
    确认模型配置已就绪（MODEL_ID / MODEL_API_KEY / MODEL_URL）。
    环境变量存在性由 config.py 惰性加载保证；此处做运行时非空检查并打印确认信息。
    """
    if not config.MODEL_ID or not config.MODEL_API_KEY or not config.MODEL_URL:
        print("错误: MODEL_ID / MODEL_API_KEY / MODEL_URL 必须全部设置")
        sys.exit(1)
    print(f">>> 模型配置: MODEL_ID={config.MODEL_ID}, MODEL_URL={config.MODEL_URL}")
    print("    模型配置已就绪 ✓")


def require_model_config_multi(count=2):
    """
    检查多模型配置是否完整。
    count=2 需要 MODEL_ID + MODEL_ID_2；count=3 额外需要 MODEL_ID_3。
    """
    require_model_config()

    extra_models = {
        2: ("MODEL_ID_2", config.MODEL_ID_2, "export MODEL_ID_2=gpt-4o-mini"),
        3: ("MODEL_ID_3", config.MODEL_ID_3, "export MODEL_ID_3=gpt-3.5-turbo"),
    }
    for level, (name, value, hint) in extra_models.items():
        if count >= level and not value:
            print(f"错误: 多模型 Fallback 测试需要 {name} 环境变量")
            print(f"  {hint}")
            sys.exit(1)

    if count >= 2:
        print(f"    多模型配置已就绪 ✓  (共 {count} 个模型 ID)")


# ═══════════════════════════════════════════════════════════════════════════════
# 通用辅助函数
# ═══════════════════════════════════════════════════════════════════════════════

def get_field(record: dict, *keys, default=None):
    """
    按优先级从 dict 中取值，兼容 CamelCase 和 snake_case 响应。

    示例:
        get_field(m, "ai_model_id", "AiModelID", "AIModelID")
        get_field(m, "ModelName", "model_name")
    """
    for k in keys:
        v = record.get(k)
        if v is not None:
            return v
    return default


def ensure_gateway_ui_enabled(admin_token):
    """确保 gateway_ui_enable 已开启（多测试脚本通用 setup 步骤）"""
    print(">>> Setup：确保 gateway_ui_enable 已开启 ...")
    site_cfg = admin_get_config(admin_token)
    if not site_cfg.get("gateway_ui_enable"):
        admin_update_config(admin_token, gateway_ui_enable="true")
    print("    gateway_ui_enable=true ✓")


def clear_instance_models(user_token, instance_db_id):
    """
    清理实例默认绑定的模型，排除默认模型配置干扰。

    健壮性策略：
    - del-model 可能 DB 删除成功但 gateway 重启失败（返回 500）
    - 重试时记录已不存在会返回 400
    - 因此在调用处捕获 "记录不存在" 的 400 错误视为已删除
    - 每轮删除后重新查询列表确认实际状态（最多 3 轮）

    返回被清理的模型数量。
    """
    from helpers.model import user_get_instance_models, user_del_model

    def _del_model_idempotent(im_id):
        """幂等删除：400 "记录不存在" 视为已删除成功。"""
        try:
            return user_del_model(user_token, instance_db_id, im_id)
        except AssertionError as e:
            if "400" in str(e) and "不存在" in str(e):
                print(f"    [idempotent] 模型 {im_id} 已不存在，视为删除成功")
                return {"ok": True, "_idempotent": True}
            raise

    def _delete_models(models):
        """批量删除一组模型，返回实际尝试删除的数量。"""
        deleted = 0
        for m in models:
            im_id = get_field(m, "instance_model_id", "InstanceModelID", "ID", "id")
            if not im_id:
                continue
            retry_on_gateway_restart(lambda _id=im_id: _del_model_idempotent(_id))
            deleted += 1
        return deleted

    print(">>> Setup：清理实例默认绑定的模型 ...")
    im_data = user_get_instance_models(user_token, instance_db_id)
    default_models = im_data.get("models", im_data.get("data", []))
    if not default_models:
        print("    无默认模型，跳过 ✓")
        return 0

    total_deleted = _delete_models(default_models)

    # 验证式循环：确认模型列表已清空，最多重试 2 轮
    max_verify_rounds = 2
    for round_idx in range(max_verify_rounds):
        time.sleep(2)
        im_data = user_get_instance_models(user_token, instance_db_id)
        remaining = im_data.get("models", im_data.get("data", []))
        if not remaining:
            break
        print(f"    ⚠ 仍有 {len(remaining)} 个模型未清理，第 {round_idx + 1} 次补偿 ...")
        _delete_models(remaining)
    else:
        # 循环正常结束（未 break），做最后一次检查
        time.sleep(2)
        im_data = user_get_instance_models(user_token, instance_db_id)
        remaining = im_data.get("models", im_data.get("data", []))
        if remaining:
            raise AssertionError(
                f"clear_instance_models 失败：经过 {max_verify_rounds} 轮补偿后"
                f"仍有 {len(remaining)} 个模型未清理: {remaining}"
            )

    print(f"    已清理 {total_deleted} 个默认模型 ✓")
    time.sleep(3)
    return total_deleted


# ═══════════════════════════════════════════════════════════════════════════════
# Gateway 重启重试工具
# ═══════════════════════════════════════════════════════════════════════════════

def _is_gateway_restart_error(text: str) -> bool:
    """判断错误文本是否包含 gateway 重启相关关键字。

    匹配规则：
    1. 包含 "gateway" 关键字（直接匹配 gateway 重启错误）
    2. 包含 "命令执行失败"（TAT 脚本执行失败的通用错误，常见于 gateway
       重启失败时脚本因 set -e 退出。此时脚本输出可能被 Go 层
       scriptFailureDetail 过滤，错误信息中不含 "gateway" 关键字，
       需兜底匹配以保证 retry_on_gateway_restart 正常触发）
    """
    text_lower = text.lower()
    if "gateway" in text_lower:
        return True
    if "命令执行失败" in text:
        return True
    return False


def retry_on_gateway_restart(
    fn: Callable[[], T],
    *,
    max_retries: int = 3,
    retry_interval: int = 15,
    verbose: bool = True,
    is_success: "Callable[[T], bool] | None" = None,
    is_already_done: "Callable[[], bool] | None" = None,
) -> T:
    """
    执行一个函数，若遇到 Gateway 重启失败则自动重试。

    支持两种返回类型:
    - requests.Response 对象（通过 status_code 判断）
    - dict（JSON 响应，通过 ok 字段判断）

    参数:
        fn:             无参数的可调用对象
        max_retries:    最大重试次数（不含首次调用）
        retry_interval: 重试间隔（秒）
        verbose:        是否打印重试日志
        is_success:     自定义成功判断函数。默认:
                        - Response 对象: status_code == 200
                        - dict 对象: data.get("ok") == True
                        - 其他: 视为成功
        is_already_done:
                        可选回调，在 gateway 重启失败时调用。
                        如果返回 True，说明操作的核心效果已经生效
                        （如 DB 已删除），无需重试，直接返回成功。
                        典型场景：del-model DB 删除成功但 gateway 重启失败。

    返回:
        fn() 的返回值

    示例:
        # Response 模式
        resp = retry_on_gateway_restart(
            lambda: helpers.user_add_model(token, db_id, model_id)
        )
        assert resp.status_code == 200

        # JSON dict 模式（带 is_already_done 验证）
        data = retry_on_gateway_restart(
            lambda: helpers.user_del_model(token, db_id, im_id),
            is_already_done=lambda: im_id not in get_current_model_ids(),
        )
        assert data.get("ok")
    """
    result = None
    for attempt in range(max_retries + 1):
        try:
            result = fn()
        except Exception as e:
            err_str = str(e)
            if _is_gateway_restart_error(err_str) and attempt < max_retries:
                # 先检查操作是否已生效
                if is_already_done is not None:
                    try:
                        if is_already_done():
                            if verbose:
                                print(f"    Gateway 重启失败，但操作已生效，跳过重试 ✓")
                            return {"ok": True, "_already_done": True}
                    except Exception:
                        pass  # is_already_done 本身出错不影响重试逻辑
                if verbose:
                    print(f"    Gateway 重启失败（异常），等待 {retry_interval}s 后重试 "
                          f"({attempt+1}/{max_retries}) ...")
                time.sleep(retry_interval)
                continue
            raise

        # 判断是否成功
        if is_success is not None:
            if is_success(result):
                return result
        elif hasattr(result, "status_code"):
            # requests.Response 模式
            if result.status_code == 200:
                return result
        elif isinstance(result, dict):
            # JSON dict 模式
            if result.get("ok"):
                return result
        else:
            # 其他类型视为成功
            return result

        # 判断是否为 gateway 重启错误、是否应该重试
        should_retry = False
        if attempt < max_retries:
            if hasattr(result, "status_code"):
                if result.status_code == 500 and _is_gateway_restart_error(result.text):
                    should_retry = True
            elif isinstance(result, dict):
                error_str = str(result.get("error", "")) + str(result.get("detail", ""))
                if _is_gateway_restart_error(error_str):
                    should_retry = True

        if should_retry:
            # 先检查操作是否已生效
            if is_already_done is not None:
                try:
                    if is_already_done():
                        if verbose:
                            print(f"    Gateway 重启失败，但操作已生效，跳过重试 ✓")
                        return {"ok": True, "_already_done": True}
                except Exception:
                    pass
            if verbose:
                print(f"    Gateway 重启失败，等待 {retry_interval}s 后重试 "
                      f"({attempt+1}/{max_retries}) ...")
            time.sleep(retry_interval)
        else:
            return result

    return result


# ═══════════════════════════════════════════════════════════════════════════════
# 兼容旧脚本的小工具
# ═══════════════════════════════════════════════════════════════════════════════

def pick_user(users, *, username=None, uid=None):
    """从用户列表中按 username 或 id 找记录"""
    for u in users or []:
        if username is not None:
            name = u.get("username") or u.get("Username")
            if name == username:
                return u
        if uid is not None:
            i = u.get("id") or u.get("ID")
            if i == uid:
                return u
    return None


def extract_uid(resp_json):
    """从创建/更新响应中提取 uid"""
    if not isinstance(resp_json, dict):
        return None
    for key in ("id", "ID", "uid", "user_id"):
        if key in resp_json and resp_json[key]:
            return resp_json[key]
    user = resp_json.get("user") or resp_json.get("User") or {}
    if isinstance(user, dict):
        return user.get("id") or user.get("ID")
    return None


def list_users_by_prefix(prefix, page_size=200):
    """按 username 前缀扫描所有用户"""
    if not prefix:
        return []
    out = []
    resp = seed.get(
        "/admin/users",
        params={
            "username": prefix,
            "include_deleted": "true",
            "page_size": page_size,
        },
        expect=None, raw=True,
    )
    if resp.status_code == 200:
        users = (resp.json() or {}).get("users") or []
        for u in users:
            name = u.get("username") or u.get("Username") or ""
            if name.startswith(prefix):
                out.append(u)
        if out:
            return out

    page = 1
    while True:
        resp = seed.get(
            "/admin/users",
            params={
                "page": page,
                "page_size": page_size,
                "include_deleted": "true",
            },
            expect=None, raw=True,
        )
        if resp.status_code != 200:
            break
        users = (resp.json() or {}).get("users") or []
        if not users:
            break
        hit = [
            u
            for u in users
            if (u.get("username") or u.get("Username") or "").startswith(prefix)
        ]
        out.extend(hit)
        if len(users) < page_size:
            break
        page += 1
    return out


def cleanup_users_by_prefix(prefix, *, verbose=True):
    """按 username 前缀彻底硬删用户"""
    tried_total, succ_total = 0, 0
    for _ in range(2):
        users = list_users_by_prefix(prefix)
        if not users:
            break
        for u in users:
            uid = u.get("id") or u.get("ID")
            if not uid:
                continue
            tried_total += 1
            r = seed.post(
                "/admin/hard-delete",
                params={"id": uid},
                data={},
                expect=None, raw=True,
            )
            if 200 <= r.status_code < 300:
                succ_total += 1

    leftover = list_users_by_prefix(prefix)
    if verbose:
        if tried_total or leftover:
            tag = "✓" if not leftover else "⚠"
            print(
                f"[cleanup] {tag} 按前缀 '{prefix}' 清理用户 "
                f"成功={succ_total}/{tried_total} 残留={len(leftover)}"
            )
            for u in leftover[:5]:
                print(
                    "          - 残留 id={} username={}".format(
                        u.get("id") or u.get("ID"),
                        u.get("username") or u.get("Username"),
                    )
                )
    return tried_total, succ_total


# ═══════════════════════════════════════════════════════════════════════════════
# 测试运行器
# ═══════════════════════════════════════════════════════════════════════════════

def _print_frame_summary(frames: list, label: str):
    """打印单个测试用例的帧摘要表"""
    if not frames:
        return
    # 计算统计
    total_ms = sum(f["duration_ms"] for f in frames)
    ok_count = sum(1 for f in frames if f["ok"])
    fail_count = len(frames) - ok_count

    status_str = GREEN(f"{ok_count}✓") if fail_count == 0 else f"{GREEN(f'{ok_count}✓')} {RED(f'{fail_count}✗')}"
    print(GRAY(f"    ┄┄┄ 帧摘要: {len(frames)} 请求, {status_str}, 总耗时 {total_ms}ms ┄┄┄"))

    # 紧凑表格（仅失败用例或 TRACE=1 时展示每行）
    show_detail = os.environ.get("TRACE") == "1" or fail_count > 0
    if show_detail:
        for f in frames:
            tag = GREEN("OK") if f["ok"] else RED("FAIL")
            exp = f.get("expect")
            exp_s = "any" if exp is None else str(exp)
            print(
                GRAY(f"    │ #{f['seq']:03d} {f['method']:6s} {f['path']:<40s} "
                     f"→ {f['status_code']} (expect={exp_s}) {f['duration_ms']}ms [{tag}]")
            )


def run_tests(scope: dict, *, prefix: str = "test_", title: str = "",
              ordered: bool = False, abort_on_fail: bool = False):
    """
    自动发现并执行 scope 中以 prefix 开头的测试函数。

    参数：
      scope: globals() — 从中发现以 prefix 开头的可调用函数
      prefix: 函数前缀，默认 "test_"
      title: 可选标题
      ordered: 是否保持定义顺序执行（True 时按文件中出现的行号排序）
      abort_on_fail: 首个测试失败后是否中止后续（需 ordered=True）

    返回 (total, passed, failed) 元组。

    增强特性:
    - 每个测试用例执行前后自动收集帧记录
    - 测试结束后输出帧摘要（请求数/成功失败/总耗时）
    - 失败用例或 TRACE=1 时展示逐帧明细
    """
    funcs = [
        (name, fn) for name, fn in scope.items()
        if name.startswith(prefix) and callable(fn)
    ]
    if not funcs:
        print("⚠️  未发现测试函数")
        return 0, 0, 0

    if ordered:
        funcs.sort(key=lambda x: _inspect.getsourcelines(x[1])[1])
    else:
        funcs.sort(key=lambda x: x[0])

    if title:
        print("=" * 60)
        print(title)
        print("=" * 60)

    passed, failed = 0, 0
    failures = []
    all_frames = []  # 全局帧记录（所有用例）
    for name, fn in funcs:
        doc = fn.__doc__
        label = doc.strip().split('\n')[0] if doc else name
        print(f"\n>>> {BOLD(label)}")
        _start_collecting()
        try:
            fn()
            frames = _stop_collecting()
            all_frames.extend(frames)
            _print_frame_summary(frames, label)
            passed += 1
            print(f"    {GREEN('✓ PASS')}")
        except AssertionError as e:
            frames = _stop_collecting()
            all_frames.extend(frames)
            _print_frame_summary(frames, label)
            failed += 1
            failures.append((name, str(e)))
            print(f"    {RED('✗ FAIL')}: {e}")
            if abort_on_fail:
                print(RED("    [ABORT] 关键步骤失败，中止后续测试"))
                break
        except Exception as e:
            frames = _stop_collecting()
            all_frames.extend(frames)
            _print_frame_summary(frames, label)
            failed += 1
            failures.append((name, f"{type(e).__name__}: {e}"))
            print(f"    {RED('✗ ERROR')}: {type(e).__name__}: {e}")
            if os.environ.get("TRACE") == "1":
                traceback.print_exc()
            if abort_on_fail:
                print(RED("    [ABORT] 关键步骤异常，中止后续测试"))
                break

    total = passed + failed
    print()
    print(BOLD("=" * 60))
    if title:
        print(BOLD(f"结果 - {title}"))
    else:
        print(BOLD("结果"))
    print(BOLD("-" * 60))
    # 全局统计
    if all_frames:
        total_reqs = len(all_frames)
        total_ms = sum(f["duration_ms"] for f in all_frames)
        ok_reqs = sum(1 for f in all_frames if f["ok"])
        fail_reqs = total_reqs - ok_reqs
        print(f"  HTTP 请求: {total_reqs} 总计, {ok_reqs} 成功, "
              f"{fail_reqs} 失败, 总耗时 {total_ms}ms")
    print(BOLD("-" * 60))
    if failed == 0:
        print(f"  {GREEN(f'全部通过 ✓ ({total} tests)')}")
    else:
        print(f"  {passed} passed / {RED(f'{failed} failed')} / {total} total")
        for name, err in failures:
            print(f"  {RED('✗')} {name}: {err[:120]}")
    print(BOLD("=" * 60))

    if failed > 0:
        sys.exit(1)
    return total, passed, failed
