#!/bin/bash
set -euo pipefail

export PATH="$HOME/.npm-global/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u)

python3 -u - <<'EOF'
# Copyright (C) 2025 Tencent. All rights reserved.
#
# 本软件由腾讯轻量云团队自主研发，受中华人民共和国著作权法及国际版权公约保护。
# 未经腾讯书面授权，任何单位或个人不得以任何形式复制、修改、传播或用于商业用途。违者将承担相应的法律责任。
#
# This software is independently developed by Tencent Lighthouse Team.
# Unauthorized copying, modification, distribution, or commercial use
# of this software, in whole or in part, is strictly prohibited.
# Violators will be held liable under applicable laws.
#
# Author: Author: Tencent Lighthouse Team
"""
QQ 开放平台 - 自动创建机器人 (一键式)

用法:
    python3 qq_bot_creator.py

流程: 获取 session_id → 展示二维码链接 → 轮询扫码 → 创建机器人
"""

import json
import os
import ssl
import sys
import time
import urllib.request
import urllib.error

# ============================================================
# 强制 stdout 行缓冲 / write-through（管道环境下 Python 默认全缓冲，
# 导致前端读不到 QR JSON）。等效于 python3 -u 但无需调用方配合。
# ============================================================
if hasattr(sys.stdout, "reconfigure"):
    sys.stdout.reconfigure(write_through=True)
elif hasattr(sys.stdout, "buffer"):
    import io as _io
    sys.stdout = _io.TextIOWrapper(
        sys.stdout.buffer, write_through=True,
        encoding=sys.stdout.encoding, errors=sys.stdout.errors,
    )

# ============================================================
# 常量
# ============================================================
BASE_URL = "https://q.qq.com"
BOT_API = "https://bot.q.qq.com"
STATE_FILE = "/tmp/qq-bot-creator-state.json"
OPENCLAW_CONFIG = os.path.expanduser("~/.openclaw/openclaw.json")

# 通用 SSL 上下文 (忽略证书校验)
_SSL_CTX = ssl.create_default_context()
_SSL_CTX.check_hostname = False
_SSL_CTX.verify_mode = ssl.CERT_NONE

_HEADERS = {
    "User-Agent": ("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) "
                   "AppleWebKit/537.36 (KHTML, like Gecko) "
                   "Chrome/145.0.0.0 Safari/537.36"),
    "Accept": "*/*",
}


# ============================================================
# 工具函数
# ============================================================
def _log(msg: str) -> None:
    print(msg, file=sys.stderr)


def _save_state(data: dict) -> None:
    with open(STATE_FILE, "w") as f:
        json.dump(data, f, ensure_ascii=False)


def _load_state() -> dict:
    if not os.path.isfile(STATE_FILE):
        return {}
    with open(STATE_FILE) as f:
        return json.load(f)


def _http_get(url: str, headers: dict = None, raw_response: bool = False):
    """发起 GET 请求，返回 (response_body, response_headers, cookies_header)。"""
    h = {**_HEADERS, **(headers or {})}
    req = urllib.request.Request(url, headers=h, method="GET")
    resp = urllib.request.urlopen(req, context=_SSL_CTX)
    body = resp.read()
    # 收集 Set-Cookie
    cookies = resp.headers.get_all("Set-Cookie") or []
    if raw_response:
        return body, resp.headers, cookies
    try:
        return json.loads(body), resp.headers, cookies
    except (json.JSONDecodeError, ValueError):
        return body.decode(errors="replace"), resp.headers, cookies


def _http_post(url: str, payload: dict, headers: dict = None):
    """发起 POST 请求，返回 JSON。"""
    h = {**_HEADERS, "Content-Type": "application/json", **(headers or {})}
    data = json.dumps(payload).encode()
    req = urllib.request.Request(url, data=data, headers=h, method="POST")
    resp = urllib.request.urlopen(req, context=_SSL_CTX)
    return json.loads(resp.read())


def _parse_cookies(set_cookie_headers: list) -> str:
    """从 Set-Cookie 头中提取 cookie 字符串。"""
    cookies = {}
    for sc in set_cookie_headers:
        part = sc.split(";")[0].strip()
        if "=" in part:
            k, v = part.split("=", 1)
            cookies[k.strip()] = v.strip()
    return "; ".join(f"{k}={v}" for k, v in cookies.items())


def _merge_cookies(existing_str: str, new_cookies: list) -> str:
    """合并已有 cookie 字符串和新的 Set-Cookie 头。"""
    merged = _parse_cookies(new_cookies)
    if not merged:
        return existing_str
    existing = {}
    if existing_str:
        for part in existing_str.split("; "):
            if "=" in part:
                k, v = part.split("=", 1)
                existing[k] = v
    for part in merged.split("; "):
        if "=" in part:
            k, v = part.split("=", 1)
            existing[k] = v
    return "; ".join(f"{k}={v}" for k, v in existing.items())


def _write_openclaw_config(app_id: str, client_secret: str) -> bool:
    """将 QQ 机器人 appId/clientSecret 写入 openclaw.json 配置文件。"""
    _log(f"[配置] 写入 openclaw 配置: {OPENCLAW_CONFIG}")

    config_dir = os.path.dirname(OPENCLAW_CONFIG)
    os.makedirs(config_dir, exist_ok=True)

    config = {}
    if os.path.isfile(OPENCLAW_CONFIG):
        try:
            with open(OPENCLAW_CONFIG) as f:
                config = json.load(f)
        except (json.JSONDecodeError, IOError) as e:
            _log(f"  [警告] 读取现有配置失败: {e}, 将创建新配置")

    if "channels" not in config:
        config["channels"] = {}
    config["channels"]["qqbot"] = {
        "enabled": True,
        "appId": app_id,
        "clientSecret": client_secret,
    }

    try:
        with open(OPENCLAW_CONFIG, "w") as f:
            json.dump(config, f, ensure_ascii=False, indent=2)
        _log(f"  [成功] 已写入 qqbot 配置 (appId={app_id})")
        return True
    except IOError as e:
        _log(f"  [失败] 写入配置文件: {e}")
        return False


# ============================================================
# 轮询扫码状态
# ============================================================
def _step_poll_login(session_id: str, cookie_str: str):
    """轮询 poll 接口等待扫码登录，成功返回 cookie_str，超时返回 None。"""
    _log("[create] 等待扫码登录 ...")
    poll_url = f"{BASE_URL}/lite/poll?session_id={session_id}&bkn=5381"
    req_headers = {}
    if cookie_str:
        req_headers["Cookie"] = cookie_str

    max_wait = 180
    poll_interval = 2
    deadline = time.time() + max_wait
    last_code = None

    while time.time() < deadline:
        try:
            body, headers, new_cookies = _http_get(poll_url, headers=req_headers)
        except Exception as e:
            _log(f"  [异常] 轮询: {e}, {poll_interval}s 后重试...")
            time.sleep(poll_interval)
            continue

        # 合并新 cookie
        cookie_str = _merge_cookies(cookie_str, new_cookies)
        if cookie_str:
            req_headers["Cookie"] = cookie_str

        # 解析 poll 响应
        code = None
        message = ""
        if isinstance(body, dict):
            data_obj = body.get("data", {})
            if isinstance(data_obj, dict):
                code = data_obj.get("code")
                message = data_obj.get("message", "")

        if code != last_code:
            if code == 3:
                _log("[create] 已扫码，请在手机上点击下一步确认")
            elif code == 0:
                _log("[create] 扫码登录成功!")
            elif code == 1:
                _log("[create] 等待扫码...")
            else:
                _log(f"[create] 轮询状态: code={code}, message={message}")
            last_code = code

        if code == 0:
            return cookie_str

        time.sleep(poll_interval)

    _log(f"[create] 轮询超时 ({max_wait}s)")
    return None


# ============================================================
# 步骤 3: 创建机器人
# ============================================================
def _step_create_bot(cookie_str: str) -> dict:
    """调用 lite_create 创建机器人，返回响应 data。"""
    _log("[create] 创建机器人 ...")

    timestamp = int(time.time() * 1000)
    payload = {
        "apply_source": 1,
        "idempotency_key": str(timestamp),
    }

    url = f"{BOT_API}/cgi-bin/lite_create?bkn=9999"

    _log(f"  POST {url}")
    _log(f"  payload: {json.dumps(payload)}")

    try:
        body = _http_post(url, payload, headers={
            "Cookie": cookie_str,
            "Origin": BASE_URL,
            "Referer": f"{BASE_URL}/",
        })
    except urllib.error.HTTPError as e:
        error_body = e.read().decode(errors="replace")
        _log(f"  [失败] HTTP {e.code}: {error_body[:500]}")
        print(json.dumps({"status": "error", "message": f"创建失败 HTTP {e.code}: {error_body[:300]}"}))
        sys.exit(1)
    except Exception as e:
        _log(f"  [失败] {e}")
        print(json.dumps({"status": "error", "message": f"创建失败: {e}"}))
        sys.exit(1)

    _log(f"  响应: {json.dumps(body, ensure_ascii=False)}")

    # retcode == 0 表示成功
    retcode = body.get("retcode")
    code = body.get("code")
    if retcode == 0 or code == 0:
        _log("[create] 机器人创建成功!")
        return body.get("data", body)

    # 失败：用 _log 输出错误详情
    err_msg = body.get("msg") or body.get("message", "")
    if retcode == 16003 or "已达上限" in err_msg:
        display_msg = "当前开发者账号下机器人数量已达上限5个，请删除历史机器人后重试"
    else:
        display_msg = f"retcode={retcode}, msg={err_msg}"
    _log(f"  [失败] {display_msg}")
    print(json.dumps({
        "status": "error",
        "message": display_msg,
        "retcode": retcode,
    }, ensure_ascii=False))
    sys.exit(1)


def _create_session():
    """获取新的 session_id 和初始 cookie，返回 (session_id, cookie_str, login_url)。"""
    try:
        body, headers, cookies = _http_get(f"{BASE_URL}/lite/create_session")
    except Exception as e:
        return None, "", ""

    session_id = None
    if isinstance(body, dict):
        session_id = body.get("session_id") or body.get("data", {}).get("session_id")
    if not session_id and isinstance(body, dict):
        for key in body:
            if "session" in key.lower():
                session_id = body[key]
                break

    cookie_str = _parse_cookies(cookies)
    login_url = f"https://q.qq.com/qqbot/openclaw/entity-picker.html?session_id={session_id}" if session_id else ""
    return session_id, cookie_str, login_url


# ============================================================
# 入口: 一键执行全流程
# ============================================================
def main():
    # 抑制 stderr，先输出 login_url 到 stdout
    _orig_stderr = sys.stderr
    _orig_fd2 = os.dup(2)
    sys.stderr = open(os.devnull, "w")
    os.dup2(os.open(os.devnull, os.O_WRONLY), 2)

    # 获取 session_id
    session_id, cookie_str, login_url = _create_session()
    if not session_id:
        sys.stderr = _orig_stderr
        os.dup2(_orig_fd2, 2)
        os.close(_orig_fd2)
        print(json.dumps({"status": "error", "message": "未获取到 session_id"}))
        sys.exit(1)

    # stdout 输出二维码链接
    print(login_url)

    # 恢复 stderr
    sys.stderr = _orig_stderr
    os.dup2(_orig_fd2, 2)
    os.close(_orig_fd2)

    _log(f"[create] 二维码链接: {login_url}")

    # 轮询扫码
    result_cookie = _step_poll_login(session_id, cookie_str)
    if result_cookie is None:
        print(json.dumps({"status": "expired", "message": "扫码超时，退出"}))
        sys.exit(1)
    cookie_str = result_cookie

    # 创建机器人
    bot_data = _step_create_bot(cookie_str)

    # 写入 openclaw 配置
    app_id = bot_data.get("appid") or bot_data.get("appId") or bot_data.get("app_id", "")
    client_secret = bot_data.get("client_secret") or bot_data.get("clientSecret", "")
    if app_id and client_secret:
        _write_openclaw_config(app_id, client_secret)
    else:
        _log(f"[create] [警告] 未从 bot_data 中获取到 appId/clientSecret, keys={list(bot_data.keys())}")

    # 清理
    if os.path.isfile(STATE_FILE):
        os.remove(STATE_FILE)

    # 输出最终结果
    result = {"status": "ok", "data": bot_data}
    print(json.dumps(result, ensure_ascii=False))


if __name__ == "__main__":
    main()
EOF

# 配置完成后，先将插件 ID 加入 plugins.allow，再 enable qqbot 插件（内置插件）
# 将插件 ID 加入 plugins.allow（幂等：已存在则不重复追加）
_cfg="$HOME/.openclaw/openclaw.json"
if [ -f "$_cfg" ]; then
    jq --arg id "qqbot" \
        '.plugins.allow = ((.plugins.allow // []) | if index($id) then . else . + [$id] end)' \
        "$_cfg" > /tmp/openclaw_allow.json && mv /tmp/openclaw_allow.json "$_cfg"
    echo "✓ qqbot 已加入 plugins.allow"
fi
echo "启用 qqbot 插件..."
openclaw plugins enable qqbot || true
echo "✓ qqbot 插件已启用"
echo "重启 gateway 使插件生效..."
systemctl --user restart openclaw-gateway || true
echo "✓ gateway 已重启"
