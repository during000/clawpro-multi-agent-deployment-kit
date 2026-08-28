#!/usr/bin/env python3
"""
HTTP 客户端基础设施

包含:
- 核心变量（API / ADMIN_TOKEN / HEADERS 等）
- 显示控制 / ANSI 颜色
- OOP 风格 ApiClient 类（内置帧记录引擎）
- 预置全局客户端（seed / anon / bad_token）
- cURL 生成 / 帧输出
- Header 构建工具
"""

import json as _json
import os
import shlex
import sys
import time
import atexit
from urllib.parse import urlencode as _urlencode, urlparse as _urlparse, parse_qs as _parse_qs

import requests

from helpers import config


# ═══════════════════════════════════════════════════════════════════════════════
# 核心变量
# ═══════════════════════════════════════════════════════════════════════════════

API: str = config.BASE_URL
ADMIN_TOKEN: str = config.SEED_ADMIN_TOKEN

DEFAULT_TIMEOUT = 30

# 可选：普通用户 token（用于权限测试）
NON_ADMIN_TOKEN: str = os.environ.get("TOKEN", "")

# 测试标识符（用于资源命名，便于 cleanup.py 按 IDENTIFIER 匹配清理）
IDENTIFIER: str = os.environ.get("IDENTIFIER", "")

# 可选：Session Cookie 认证方式
SESSION_COOKIE: str = os.environ.get("SESSION_COOKIE", "")
SESSION_COOKIE_B: str = os.environ.get("SESSION_COOKIE_B", "")

# 普通用户 token（OpenAPI 风格）
TOKEN: str = os.environ.get("TOKEN", "")


# ═══════════════════════════════════════════════════════════════════════════════
# 显示控制
# ═══════════════════════════════════════════════════════════════════════════════

SHOW_TOKEN = os.environ.get("SHOW_TOKEN") == "1"
QUIET = os.environ.get("QUIET") == "1"
NO_COLOR = os.environ.get("NO_COLOR") == "1" or not sys.stdout.isatty()
try:
    RESP_MAX = int(os.environ.get("RESP_MAX", "0"))
except ValueError:
    RESP_MAX = 0


# ═══════════════════════════════════════════════════════════════════════════════
# ANSI 颜色
# ═══════════════════════════════════════════════════════════════════════════════

def _c(code, s):
    if NO_COLOR:
        return s
    return f"\033[{code}m{s}\033[0m"


GREEN = lambda s: _c("32", s)
RED = lambda s: _c("31", s)
YELLOW = lambda s: _c("33", s)
CYAN = lambda s: _c("36", s)
GRAY = lambda s: _c("90", s)
BOLD = lambda s: _c("1", s)


# ═══════════════════════════════════════════════════════════════════════════════
# cURL 与帧输出工具
# ═══════════════════════════════════════════════════════════════════════════════

def _safe_token(token):
    """脱敏 token 显示"""
    if SHOW_TOKEN:
        return f"Bearer {token}"
    return "Bearer ***"


def _build_curl(method, url, headers, data=None, json_body=None):
    """组装一个可直接复制到 shell 执行的 cURL 命令"""
    parts = ["curl", "-X", method, shlex.quote(url)]
    for k, v in headers.items():
        if k.lower() == "authorization":
            # 脱敏
            if not SHOW_TOKEN:
                v = "Bearer ***"
        parts += ["-H", shlex.quote(f"{k}: {v}")]
    if json_body is not None:
        parts += ["-H", shlex.quote("Content-Type: application/json")]
        parts += ["--data", shlex.quote(_json.dumps(json_body, ensure_ascii=False))]
    elif data:
        if isinstance(data, dict):
            parts += ["--data", shlex.quote(_urlencode(data))]
        elif isinstance(data, list):
            # list of tuples
            parts += ["--data", shlex.quote(_urlencode(data))]
        else:
            parts += ["--data", shlex.quote(str(data))]
    return " ".join(parts)


def _full_url(base_url, path, params):
    if not params:
        return f"{base_url}{path}"
    if isinstance(params, dict):
        qs = _urlencode([(k, v) for k, v in params.items() if v is not None])
    else:
        qs = _urlencode(params)
    return f"{base_url}{path}?{qs}" if qs else f"{base_url}{path}"


def truncate(text, limit):
    if limit <= 0 or text is None:
        return text or ""
    if len(text) <= limit:
        return text
    return text[:limit] + f"... <truncated, total={len(text)}>"


# 向后兼容别名
_truncate = truncate


def _format_resp(resp):
    body = resp.text or ""
    ctype = resp.headers.get("Content-Type", "")
    if "json" in ctype.lower():
        try:
            obj = resp.json()
            return _json.dumps(obj, ensure_ascii=False, indent=2)
        except Exception:
            pass
    return body


def _format_request_body(data=None, json_body=None):
    if json_body is not None:
        try:
            return ("json", _json.dumps(json_body, ensure_ascii=False, indent=2))
        except Exception:
            return ("json", str(json_body))
    if data:
        if isinstance(data, dict):
            if not data:
                return None
            lines = [f"{k}={v}" for k, v in data.items()]
            return ("form", "\n".join(lines))
        if isinstance(data, list):
            # list of tuples
            lines = [f"{k}={v}" for k, v in data]
            return ("form", "\n".join(lines))
        return ("raw", str(data))
    return None


def _print_frame(frame):
    # 收集帧记录（如果收集器已激活）
    if _frame_collector is not None:
        _frame_collector.append(frame)
    if QUIET:
        return
    seq = frame.get("seq", 0)
    method = frame["method"]
    path = frame["path"]
    status = frame["status_code"]
    dur = frame["duration_ms"]
    ok_flag = frame["ok"]
    expect = frame.get("expect")
    tag = GREEN("OK ") if ok_flag else RED("FAIL")
    if expect is None:
        expect_hint = GRAY(" [expect=any]")
    elif isinstance(expect, (list, tuple, set)):
        expect_hint = GRAY(f" [expect∈{sorted(list(expect))}]")
    else:
        expect_hint = GRAY(f" [expect={expect}]")
    head = (
        f"  ── [#{seq:03d}] {BOLD(method)} {CYAN(path)}  "
        f"→ {tag} {status}{expect_hint}  {GRAY(f'({dur}ms)')}"
    )
    print(head)
    print(GRAY("     cURL: ") + frame["curl"])
    req_body = frame.get("req_body")
    if req_body:
        kind, text = req_body
        label = {"json": "Body (JSON)", "form": "Body (form)", "raw": "Body"}.get(kind, "Body")
        print(GRAY(f"     {label}:"))
        for ln in text.splitlines():
            print("           " + ln)
    body_pretty = _truncate(frame["resp_text"], RESP_MAX)
    body_indented = "\n".join("           " + ln for ln in body_pretty.splitlines()) \
        if body_pretty else "           <empty>"
    print(GRAY("     Resp:"))
    print(body_indented)


# 全局调用计数
_call_seq = 0

# ── 帧记录收集器（供 testing.py run_tests 使用）──
_frame_collector: "list | None" = None


def _start_collecting():
    """开始收集帧记录（在 run_tests 中每个 test case 前调用）"""
    global _frame_collector
    _frame_collector = []


def _stop_collecting() -> list:
    """停止收集并返回帧记录列表"""
    global _frame_collector
    frames = _frame_collector or []
    _frame_collector = None
    return frames


# ── API Coverage 数据收集（当 COVERAGE_DIR 环境变量存在时激活）──
_coverage_frames: list = []
_COVERAGE_DIR: str = os.environ.get("COVERAGE_DIR", "")


def _json_body_keys(value, prefix=""):
    """Return dotted coverage keys for nested JSON objects and arrays."""
    keys = set()
    if isinstance(value, dict):
        for key, nested in value.items():
            path = f"{prefix}.{key}" if prefix else key
            keys.add(path)
            keys.update(_json_body_keys(nested, path))
    elif isinstance(value, list):
        item_prefix = f"{prefix}[]" if prefix else "[]"
        for item in value:
            keys.update(_json_body_keys(item, item_prefix))
    return keys


def _record_coverage_frame(method: str, path: str, url: str, status_code: int,
                           data=None, json_body=None):
    """记录一条精简的覆盖率帧数据"""
    if not _COVERAGE_DIR:
        return
    # 提取 query 参数 key
    parsed = _urlparse(url)
    query_keys = sorted(set(_parse_qs(parsed.query).keys())) if parsed.query else []
    # 提取 body 参数 key
    body_keys = []
    if isinstance(json_body, dict):
        body_keys = sorted(_json_body_keys(json_body))
    elif data:
        if isinstance(data, dict):
            body_keys = sorted(data.keys())
        elif isinstance(data, list):
            # list of tuples
            body_keys = sorted(set(k for k, _ in data))
    _coverage_frames.append({
        "method": method,
        "path": path,
        "status_code": status_code,
        "query_keys": query_keys,
        "body_keys": body_keys,
    })


def _flush_coverage():
    """atexit hook：将覆盖率数据写入 COVERAGE_DIR/<script_name>.json"""
    if not _COVERAGE_DIR or not _coverage_frames:
        return
    try:
        os.makedirs(_COVERAGE_DIR, exist_ok=True)
        # 使用主脚本文件名（去掉 .py 后缀）作为 JSON 文件名
        script_name = os.path.basename(sys.argv[0]).removesuffix(".py") if sys.argv else "unknown"
        out_path = os.path.join(_COVERAGE_DIR, f"{script_name}.json")
        with open(out_path, "w", encoding="utf-8") as f:
            _json.dump(_coverage_frames, f, ensure_ascii=False)
    except Exception:
        pass  # 覆盖率收集失败不影响测试


if _COVERAGE_DIR:
    atexit.register(_flush_coverage)


# ═══════════════════════════════════════════════════════════════════════════════
# HTTP 请求客户端（OOP 风格，内置帧记录引擎）
# ═══════════════════════════════════════════════════════════════════════════════

class ApiClient:
    """
    统一 HTTP API 客户端 —— 所有请求经帧记录引擎。

    特性:
    - 自动构建 Authorization header
    - 每个请求自动打印 cURL + 帧输出（可通过 QUIET=1 关闭）
    - 支持 expect 参数（None=不断言 / int / tuple/list/set=多选）
    - 支持 raw=True 直接返回 Response 对象（不解 JSON）
    - GET/POST/PUT/DELETE/PATCH 快捷方法

    用法:
        client = ApiClient(token, openapi=True)
        data = client.get("/openclaw/models")
        resp = client.post("/openclaw/add-model", json={...}, raw=True)
        client.delete("/admin/models/delete", params={"id": 1}, expect=200)
    """

    def __init__(self, token: str, *, openapi: bool = False,
                 base_url: str | None = None, timeout: int = 60,
                 trace: bool = True):
        self.token = token
        self.openapi = openapi
        self.base_url = (base_url or config.BASE_URL).rstrip("/")
        self.timeout = timeout
        self.trace = trace  # 是否走帧记录

    @property
    def headers(self) -> dict[str, str]:
        h: dict[str, str] = {"Accept": "application/json"}
        if self.token:
            h["Authorization"] = f"Bearer {self.token}"
        if self.openapi:
            h["X-OpenAPI"] = "1"
        return h

    def url(self, path: str) -> str:
        """拼接完整 URL"""
        return f"{self.base_url}{path}"

    # ─── 快捷方法 ───

    def get(self, path: str, *, params=None, json=None, expect=200,
            timeout: int | None = None, raw: bool = False,
            extra_headers: dict | None = None, **kwargs) -> "dict | requests.Response":
        """GET 请求"""
        return self._execute("GET", path, params=params, json_body=json,
                             expect=expect, timeout=timeout, raw=raw,
                             extra_headers=extra_headers, **kwargs)

    def post(self, path: str, *, params=None, data=None, json=None,
             expect=200, timeout: int | None = None, raw: bool = False,
             extra_headers: dict | None = None, **kwargs) -> "dict | requests.Response":
        """POST 请求"""
        return self._execute("POST", path, params=params, data=data,
                             json_body=json, expect=expect, timeout=timeout,
                             raw=raw, extra_headers=extra_headers, **kwargs)

    def put(self, path: str, *, params=None, data=None, json=None,
            expect=200, timeout: int | None = None, raw: bool = False,
            extra_headers: dict | None = None, **kwargs) -> "dict | requests.Response":
        """PUT 请求"""
        return self._execute("PUT", path, params=params, data=data,
                             json_body=json, expect=expect, timeout=timeout,
                             raw=raw, extra_headers=extra_headers, **kwargs)

    def delete(self, path: str, *, params=None, data=None, json=None,
               expect=200, timeout: int | None = None, raw: bool = False,
               extra_headers: dict | None = None, **kwargs) -> "dict | requests.Response":
        """DELETE 请求"""
        return self._execute("DELETE", path, params=params, data=data,
                             json_body=json, expect=expect, timeout=timeout,
                             raw=raw, extra_headers=extra_headers, **kwargs)

    def patch(self, path: str, *, params=None, data=None, json=None,
              expect=200, timeout: int | None = None, raw: bool = False,
              extra_headers: dict | None = None, **kwargs) -> "dict | requests.Response":
        """PATCH 请求"""
        return self._execute("PATCH", path, params=params, data=data,
                             json_body=json, expect=expect, timeout=timeout,
                             raw=raw, extra_headers=extra_headers, **kwargs)

    def request(self, method: str, path: str, *, params=None, data=None,
                json=None, expect=200, timeout: int | None = None,
                raw: bool = False, extra_headers: dict | None = None,
                **kwargs) -> "dict | requests.Response":
        """任意 HTTP 方法请求"""
        return self._execute(method.upper(), path, params=params, data=data,
                             json_body=json, expect=expect, timeout=timeout,
                             raw=raw, extra_headers=extra_headers, **kwargs)

    # ─── 核心执行引擎 ───

    def _execute(self, method: str, path: str, *, params=None, data=None,
                 json_body=None, expect=200, timeout: int | None = None,
                 raw: bool = False, extra_headers: dict | None = None,
                 **kwargs) -> "dict | requests.Response":
        """
        核心执行引擎：帧记录 + expect 断言 + 返回值处理。

        expect:
          - None: 不做断言，无论状态码都返回
          - int: 断言 status_code == expect
          - tuple/list/set: 断言 status_code in expect
        raw:
          - False: 成功时返回 resp.json()
          - True: 返回原始 Response 对象
        """
        global _call_seq
        timeout = timeout or self.timeout
        h = self.headers
        if extra_headers:
            h = {**h, **extra_headers}

        url = _full_url(self.base_url, path, params)
        curl = _build_curl(method, url, h, data=data, json_body=json_body)
        body_pretty_req = _format_request_body(data=data, json_body=json_body)

        started = time.time()
        try:
            resp = requests.request(
                method, url,
                headers=h,
                json=json_body,
                data=data if json_body is None else None,
                timeout=timeout,
                **kwargs,
            )
        except Exception as e:
            dur = int((time.time() - started) * 1000)
            _call_seq += 1
            if self.trace:
                frame = {
                    "method": method, "path": path, "url": url, "curl": curl,
                    "status_code": -1, "duration_ms": dur, "ok": False,
                    "resp_text": f"<request error> {e}", "expect": expect,
                    "req_body": body_pretty_req, "seq": _call_seq,
                }
                _print_frame(frame)
            raise

        dur = int((time.time() - started) * 1000)
        is_stream = kwargs.get("stream", False)
        body_pretty = "<streaming response>" if is_stream else _format_resp(resp)

        if expect is None:
            ok_flag = True
        elif isinstance(expect, (list, tuple, set)):
            ok_flag = resp.status_code in expect
        else:
            ok_flag = resp.status_code == expect

        _call_seq += 1
        if self.trace:
            frame = {
                "method": method, "path": path, "url": url, "curl": curl,
                "status_code": resp.status_code, "duration_ms": dur, "ok": ok_flag,
                "resp_text": body_pretty, "expect": expect,
                "req_body": body_pretty_req, "seq": _call_seq,
            }
            _print_frame(frame)

        # 覆盖率数据收集（独立于 trace 和 frame_collector）
        _record_coverage_frame(method, path, url, resp.status_code,
                               data=data, json_body=json_body)

        if expect is not None and not ok_flag:
            err = (f"{method} {path} 期望 {expect}, 实际 {resp.status_code}; "
                   f"resp={_truncate(body_pretty, 300)}")
            raise AssertionError(err)

        if raw:
            return resp
        # 非 raw 模式：尝试解析 JSON
        try:
            return resp.json()
        except Exception:
            return resp


# ═══════════════════════════════════════════════════════════════════════════════
# 预置全局客户端
# ═══════════════════════════════════════════════════════════════════════════════

# 种子管理员客户端（全帧记录）
seed = ApiClient(ADMIN_TOKEN, base_url=API, timeout=DEFAULT_TIMEOUT)

# 无认证客户端（用于测试 401/403）
anon = ApiClient("", base_url=API, timeout=DEFAULT_TIMEOUT)

# 错误 token 客户端（用于测试 401/403）
bad_token = ApiClient("wrong-token-that-does-not-exist", base_url=API, timeout=DEFAULT_TIMEOUT)


# ═══════════════════════════════════════════════════════════════════════════════
# 预置客户端工厂
# ═══════════════════════════════════════════════════════════════════════════════

def admin_client(admin_token):
    """管理员测试用户客户端"""
    return ApiClient(admin_token)


def user_client(user_token):
    """普通测试用户客户端（带 X-OpenAPI: 1）"""
    return ApiClient(user_token, openapi=True)


# 全局管理员 HEADERS（向后兼容旧脚本直接引用 HEADERS）
HEADERS: dict = seed.headers


# ═══════════════════════════════════════════════════════════════════════════════
# 常用 Header 构建工具
# ═══════════════════════════════════════════════════════════════════════════════

def no_auth_headers() -> dict:
    """无认证信息的 header"""
    return {"Accept": "application/json", "Content-Type": "application/json"}


def wrong_token_headers() -> dict:
    """错误 token 的 header"""
    return {
        "Authorization": "Bearer wrong-token-that-does-not-exist",
        "Accept": "application/json",
        "Content-Type": "application/json",
    }


def non_admin_headers() -> dict:
    """非管理员（普通用户）token 的 header"""
    return {
        "Authorization": f"Bearer {NON_ADMIN_TOKEN}",
        "Accept": "application/json",
        "Content-Type": "application/json",
    }


def bearer_header(token: str) -> dict:
    """指定 token 的 Bearer header"""
    return {
        "Authorization": f"Bearer {token}",
        "Accept": "application/json",
    }


def cookie_header(cookie_str: str) -> dict:
    """指定 Cookie 的 header"""
    return {
        "Cookie": cookie_str,
        "Accept": "application/json",
    }


def user_headers() -> dict:
    """普通用户请求 header（优先 TOKEN，否则 SESSION_COOKIE）"""
    if TOKEN:
        return bearer_header(TOKEN)
    if SESSION_COOKIE:
        return cookie_header(SESSION_COOKIE)
    return {"Accept": "application/json"}

