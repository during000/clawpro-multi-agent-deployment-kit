#!/bin/bash
#
# lib_qr_render.sh
#
# 作用：把 URL 渲染成 UTF8 字符画二维码（与 OpenClaw weixin_bot_creator.sh
# 输出契约一致），供各 *_bot_creator_*.sh wrapper source 使用。
#
# 对外导出：
#   render_qr_utf8 "https://example.com/qr"   → stdout 打印 UTF8 字符画
#                                                失败时退出码非 0
#
# 依赖：qrencode (Linux: qrencode; macOS: brew install qrencode)
#   - 优先使用已安装的 qrencode
#   - 未安装时尝试用 yum/apt/dnf 无交互安装（需 root 或 sudo 免密）
#   - 仍不可用时返回非 0，调用方应降级为输出 URL
#
# 输出字符集：U+2580~U+259F + 空格（与 openclaw 原字符画字符集
# "█▄▀▐▌▓░■□ " 兼容，hatchery Go 侧 / 前端零改动）

# shellcheck shell=bash

_qr_ensure_qrencode() {
    command -v qrencode >/dev/null 2>&1 && return 0

    if command -v yum >/dev/null 2>&1; then
        yum install -y -q qrencode >/dev/null 2>&1 || true
    elif command -v dnf >/dev/null 2>&1; then
        dnf install -y -q qrencode >/dev/null 2>&1 || true
    elif command -v apt-get >/dev/null 2>&1; then
        DEBIAN_FRONTEND=noninteractive apt-get update -qq >/dev/null 2>&1 || true
        DEBIAN_FRONTEND=noninteractive apt-get install -y -qq qrencode >/dev/null 2>&1 || true
    fi

    command -v qrencode >/dev/null 2>&1
}

# render_qr_utf8 <url>
#   stdout → UTF8 字符画二维码
#   exit 0 成功；非 0 失败（qrencode 不可用或渲染错误）
render_qr_utf8() {
    local url="$1"
    if [ -z "$url" ]; then
        echo "render_qr_utf8: url is empty" >&2
        return 2
    fi
    if ! _qr_ensure_qrencode; then
        echo "render_qr_utf8: qrencode not installed and auto-install failed" >&2
        return 3
    fi
    # -t UTF8        半角字符（Unicode block elements），终端友好
    # -m 2           外边距 2 模块，便于摄像头识别
    # --level=L      低纠错等级（URL 短，扫描速度更快）
    qrencode -t UTF8 -m 2 --level=L "$url" 2>/dev/null
}

# qr_to_json_content <url>
#   把 URL 渲染成字符画并做一次 JSON 字符串转义（\n → \\n, " → \"），
#   供 emit show_qrcode 事件的 content 字段原样拼接。
#   渲染失败时：fallback 为 "请在浏览器中打开以下链接完成扫码：\n<url>"
qr_to_json_content() {
    local url="$1"
    local qr
    qr="$(render_qr_utf8 "$url" 2>/dev/null)" || {
        qr="请在浏览器中打开以下链接完成扫码："$'\n'"$url"
    }
    # JSON 字符串转义：\ → \\，" → \"，换行 → \n
    qr="${qr//\\/\\\\}"
    qr="${qr//\"/\\\"}"
    qr="${qr//$'\n'/\\n}"
    qr="${qr//$'\r'/}"
    printf '%s' "$qr"
}
