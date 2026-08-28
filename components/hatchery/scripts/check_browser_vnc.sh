#!/bin/bash

# 确保使用 bash 执行（TAT Agent 可能使用 dash）
if [ -z "${BASH_VERSION:-}" ]; then
    exec /bin/bash "$0" "$@"
fi

set -uo pipefail

# ========== 云端浏览器（Browser VNC）环境检查脚本（优化版）==========
# 输出：最后一行 JSON: {"ready": true/false, "checks": {...}, "elapsed_ms": N}

CHECK_START_TS=$(date +%s%N 2>/dev/null || echo "$(date +%s)000000000")

ALL_OK=true
declare -A CHECKS
MISSING=()

# ========== 并行采集慢数据 ==========
# 以下命令各自较慢（dpkg-query ~200ms, fc-list ~1-3s, ss ~50ms, systemctl ~300ms）
# 并行执行后只取最慢一个的耗时

# 临时文件（用 $$ 确保唯一）
T_PKG="/tmp/vnc-chk-pkg.$$"
T_FC="/tmp/vnc-chk-fc.$$"
T_SS="/tmp/vnc-chk-ss.$$"
T_SVC="/tmp/vnc-chk-svc.$$"

# 采集用包列表（用于 dpkg-query 并行采集，包含核心包 + 增强包）
QUERY_PACKAGES=(
    "tigervnc-standalone-server"
    "openbox"
    "dbus"
    "dbus-x11"
    "python3-numpy"
)

# 桌面扩展包（云桌面模式需要，仅浏览器模式不需要，不影响 ready 判定）
DESKTOP_PACKAGES=(
    "xfce4-panel"
    "xfce4-terminal"
    "xfce4-settings"
    "xfdesktop4"
    "fcitx5"
    "fcitx5-chinese-addons"
    "fonts-wqy-zenhei"
    "xclip"
    "autocutsel"
)

REQUIRED_SERVICES=(
    "browser-vnc-xvnc"
    "browser-vnc-websockify"
    "browser-vnc-openbox"
    "browser-vnc-session"
    "browser-vnc-chromium"
)

# 并行启动采集
dpkg-query -W -f='${Package}\t${Status}\t${Version}\n' "${QUERY_PACKAGES[@]}" "${DESKTOP_PACKAGES[@]}" > "$T_PKG" 2>/dev/null &
PID_PKG=$!

fc-list 2>/dev/null > "$T_FC" &
PID_FC=$!

ss -tlnp > "$T_SS" 2>/dev/null &
PID_SS=$!

# 批量查询服务状态（一次 systemctl 调用代替 5 次）
{
    for svc in "${REQUIRED_SERVICES[@]}"; do
        status=$(systemctl is-active "$svc" 2>/dev/null) || status="inactive"
        ts=""
        if [ "$status" = "active" ]; then
            ts=$(systemctl show "$svc" --property=ActiveEnterTimestamp --value 2>/dev/null || true)
        fi
        echo "${svc}|${status}|${ts}"
    done
} > "$T_SVC" &
PID_SVC=$!

# 等待所有并行采集完成
wait $PID_PKG $PID_FC $PID_SS $PID_SVC 2>/dev/null

# ========== 检查 1：系统依赖包 ==========
PACKAGES_OK=true
PKG_INFO=$(cat "$T_PKG" 2>/dev/null || true)

# 核心包：缺少任何一个都会导致 ready=false
CORE_PACKAGES=(
    "tigervnc-standalone-server"
    "openbox"
    "dbus"
    "dbus-x11"
)

for pkg in "${CORE_PACKAGES[@]}"; do
    PKG_LINE=$(echo "$PKG_INFO" | grep -F "${pkg}	" || echo "")
    if ! echo "$PKG_LINE" | grep -Fq "install ok installed"; then
        PACKAGES_OK=false
        MISSING+=("package:$pkg")
    fi
done

# 增强包：缺少时记录但不影响 ready 判定（旧版机器可能没装）
ENHANCED_PACKAGES=(
    "python3-numpy"
)
ENHANCED_PKGS_OK=true
for pkg in "${ENHANCED_PACKAGES[@]}"; do
    PKG_LINE=$(echo "$PKG_INFO" | grep -F "${pkg}	" || echo "")
    if ! echo "$PKG_LINE" | grep -Fq "install ok installed"; then
        ENHANCED_PKGS_OK=false
    fi
done

if [ "$PACKAGES_OK" = true ]; then
    if [ "$ENHANCED_PKGS_OK" = true ]; then
        CHECKS["packages"]="ok"
    else
        CHECKS["packages"]="ok_partial"
    fi
else
    CHECKS["packages"]="missing"
    ALL_OK=false
fi

# ========== 检查 2：noVNC + websockify ==========
NOVNC_OK=true
[ -d "/opt/browser-vnc/noVNC" ] && [ -f "/opt/browser-vnc/noVNC/vnc.html" ] || { NOVNC_OK=false; MISSING+=("noVNC"); }
[ -d "/opt/browser-vnc/websockify" ] && [ -f "/opt/browser-vnc/websockify/run" ] || { NOVNC_OK=false; MISSING+=("websockify"); }

if [ "$NOVNC_OK" = true ]; then
    CHECKS["novnc"]="ok"
else
    CHECKS["novnc"]="missing"
    ALL_OK=false
fi

# ========== 检查 3：Google Chrome（严格验证，必须是真正的 Chrome 而非 Chromium 符号链接）==========
CHROME_CHECK="missing"
if command -v google-chrome-stable >/dev/null 2>&1; then
    CHROME_REAL_PATH=$(readlink -f "$(which google-chrome-stable)" 2>/dev/null || echo "")
    CHROME_VER_FULL=$(google-chrome-stable --version 2>/dev/null || echo "")
    if echo "$CHROME_REAL_PATH" | grep -qiE 'chromium'; then
        # google-chrome-stable 是 Chromium 的符号链接，不算真正的 Chrome
        CHROME_CHECK="chromium_symlink"
    elif ! echo "$CHROME_VER_FULL" | grep -qi "Google Chrome"; then
        # 版本信息不包含 Google Chrome 标识
        CHROME_CHECK="not_genuine"
    else
        CHROME_CHECK="ok"
    fi
fi

if [ "$CHROME_CHECK" = "ok" ]; then
    CHECKS["chrome"]="ok"
else
    CHECKS["chrome"]="$CHROME_CHECK"
    ALL_OK=false
    MISSING+=("chrome:$CHROME_CHECK")
fi

# ========== 检测服务管理器类型 ==========
# 旧版使用 supervisor，新版使用 systemd
SERVICE_MANAGER="unknown"
if [ -f "/etc/systemd/system/browser-vnc-xvnc.service" ]; then
    SERVICE_MANAGER="systemd"
elif [ -f "/etc/supervisor/conf.d/browser-vnc.conf" ]; then
    SERVICE_MANAGER="supervisor"
elif command -v supervisorctl >/dev/null 2>&1; then
    if supervisorctl status browser-vnc-xvnc >/dev/null 2>&1; then
        SERVICE_MANAGER="supervisor"
    fi
fi

# ========== 检查 4：服务配置 ==========
if [ "$SERVICE_MANAGER" = "systemd" ]; then
    SYSTEMD_UNITS_OK=true
    for unit in "${REQUIRED_SERVICES[@]}"; do
        [ -f "/etc/systemd/system/${unit}.service" ] || { SYSTEMD_UNITS_OK=false; break; }
    done
    if [ "$SYSTEMD_UNITS_OK" = true ]; then
        CHECKS["systemd_config"]="ok"
    else
        CHECKS["systemd_config"]="missing"
        ALL_OK=false
        MISSING+=("systemd-units")
    fi
elif [ "$SERVICE_MANAGER" = "supervisor" ]; then
    # 旧版 supervisor 用户：检查 supervisor 配置文件存在
    if [ -f "/etc/supervisor/conf.d/browser-vnc.conf" ]; then
        CHECKS["systemd_config"]="ok_supervisor"
    else
        CHECKS["systemd_config"]="missing"
        ALL_OK=false
        MISSING+=("supervisor-config")
    fi
else
    CHECKS["systemd_config"]="missing"
    ALL_OK=false
    MISSING+=("no-service-manager")
fi

# ========== 检查 5：服务进程状态 ==========
SERVICES_OK=true

if [ "$SERVICE_MANAGER" = "systemd" ]; then
    SVC_INFO=$(cat "$T_SVC" 2>/dev/null || true)
    while IFS='|' read -r svc status ts; do
        [ -z "$svc" ] && continue
        if [ "$status" != "active" ]; then
            SERVICES_OK=false
            MISSING+=("service:$svc")
        fi
    done <<< "$SVC_INFO"
elif [ "$SERVICE_MANAGER" = "supervisor" ]; then
    # 旧版：通过 supervisorctl 检查服务状态
    # 旧版可能只有 3-4 个服务（不一定有 browser-vnc-session），只检查核心 3 个
    for svc in browser-vnc-xvnc browser-vnc-websockify browser-vnc-chromium; do
        svc_status=$(supervisorctl status "$svc" 2>/dev/null | awk '{print $2}')
        if [ "$svc_status" != "RUNNING" ]; then
            SERVICES_OK=false
            MISSING+=("service:$svc")
        fi
    done
else
    SERVICES_OK=false
    MISSING+=("service:no-manager")
fi

if [ "$SERVICES_OK" = true ]; then
    CHECKS["services"]="ok"
else
    CHECKS["services"]="not_running"
    ALL_OK=false
fi

# ========== 检查 6：端口监听状态（使用并行采集结果）==========
PORTS_OK=true
SS_OUTPUT=$(cat "$T_SS" 2>/dev/null || true)

for port in 5900 6080 9222; do
    if ! echo "$SS_OUTPUT" | grep -q ":${port} "; then
        PORTS_OK=false
        MISSING+=("port:$port")
    fi
done

if [ "$PORTS_OK" = true ]; then
    CHECKS["ports"]="ok"
else
    CHECKS["ports"]="not_listening"
    ALL_OK=false
fi

# ========== 关键检查：CDP 端口归属（必须是 Browser VNC 的 Chrome，非 OpenClaw chromium）==========
CDP_PID=$(echo "$SS_OUTPUT" | grep ':9222 ' | grep -oP 'pid=\K[0-9]+' | head -1)
if [ -n "$CDP_PID" ]; then
    CDP_CMDLINE=$(cat /proc/$CDP_PID/cmdline 2>/dev/null | tr '\0' ' ')
    CDP_EXE=$(readlink -f /proc/$CDP_PID/exe 2>/dev/null || echo "")

    # 正向验证：必须使用 /opt/browser-vnc/chrome-data 数据目录（Browser VNC 的 Chrome 特征）
    if echo "$CDP_CMDLINE" | grep -q '/opt/browser-vnc/chrome-data'; then
        # 进一步排除 headless 模式（Browser VNC 的 Chrome 是可视化的，不应该是 headless）
        if echo "$CDP_CMDLINE" | grep -q '\-\-headless'; then
            CHECKS["cdp_owner"]="conflict"
            ALL_OK=false
            MISSING+=("cdp:vnc-chrome-headless")
        else
            # 验证可执行文件路径不是 Chromium
            if echo "$CDP_EXE" | grep -qiE 'chromium' && ! echo "$CDP_EXE" | grep -qi 'google'; then
                CHECKS["cdp_owner"]="conflict"
                ALL_OK=false
                MISSING+=("cdp:chromium-not-chrome")
            else
                CHECKS["cdp_owner"]="ok"
            fi
        fi
    else
        # 不包含 /opt/browser-vnc/chrome-data：可能是旧版安装或其他进程
        # 先排除明确的冲突进程（OpenClaw headless chromium、Playwright 等）
        if echo "$CDP_CMDLINE" | grep -q '\-\-headless'; then
            # headless 模式 = OpenClaw 的 chromium，绝对冲突
            CHECKS["cdp_owner"]="conflict"
            ALL_OK=false
            MISSING+=("cdp:headless-conflict")
        elif echo "$CDP_CMDLINE" | grep -q 'ms-playwright'; then
            CHECKS["cdp_owner"]="conflict"
            ALL_OK=false
            MISSING+=("cdp:playwright-conflict")
        elif echo "$CDP_CMDLINE" | grep -q 'lighthouse'; then
            CHECKS["cdp_owner"]="conflict"
            ALL_OK=false
            MISSING+=("cdp:lighthouse-conflict")
        elif [ "$SERVICE_MANAGER" = "supervisor" ] && echo "$CDP_EXE" | grep -qi 'chrome'; then
            # 旧版 supervisor 模式：Chrome 进程（非 headless、非 playwright、非 lighthouse）
            # 可能使用不同的 user-data-dir，但只要是真正的 Chrome 就算 ok
            CHECKS["cdp_owner"]="ok_legacy"
        else
            # 未知进程占用 9222 端口
            CHECKS["cdp_owner"]="conflict"
            ALL_OK=false
            MISSING+=("cdp:unknown-process")
        fi
    fi
else
    # CDP_PID 为空：可能是 9222 未监听，也可能是端口在监听但 ss 无法获取 PID（权限不足）
    if echo "$SS_OUTPUT" | grep -q ':9222 '; then
        # 端口在监听但无法获取 PID（非 root 运行或内核限制）
        CHECKS["cdp_owner"]="unknown_pid"
        ALL_OK=false
        MISSING+=("cdp:cannot-identify-process")
    else
        # 9222 确实未监听
        CHECKS["cdp_owner"]="no_listener"
        ALL_OK=false
        MISSING+=("cdp:not-listening")
    fi
fi

# ========== 附加检查：SSL 证书 ==========
if [ -f "/opt/browser-vnc/cert.pem" ] && [ -f "/opt/browser-vnc/key.pem" ]; then
    if openssl x509 -checkend 0 -noout -in "/opt/browser-vnc/cert.pem" 2>/dev/null; then
        CHECKS["ssl_cert"]="ok"
    else
        CHECKS["ssl_cert"]="expired"
        ALL_OK=false
        MISSING+=("ssl:cert-expired")
    fi
elif [ "$SERVICE_MANAGER" = "supervisor" ]; then
    # 旧版 supervisor 用户可能没有 SSL 证书（websockify 使用 ws://），不影响 ready
    CHECKS["ssl_cert"]="not_configured"
else
    CHECKS["ssl_cert"]="missing"
    ALL_OK=false
    MISSING+=("ssl:cert-missing")
fi

# ========== 附加检查：fcitx5（信息性，不影响 ready）==========
if command -v fcitx5 >/dev/null 2>&1; then
    if pgrep -x fcitx5 >/dev/null 2>&1; then
        CHECKS["fcitx5"]="ok"
    else
        CHECKS["fcitx5"]="installed_not_running"
    fi
else
    CHECKS["fcitx5"]="missing"
fi

# ========== 附加检查：CJK 字体（信息性，不影响 ready）==========
FC_OUTPUT=$(cat "$T_FC" 2>/dev/null || true)
if echo "$FC_OUTPUT" | grep -qi "CJK\|WenQuanYi\|wqy\|Noto Sans.*SC\|Zen Hei"; then
    CHECKS["cjk_fonts"]="ok"
else
    CHECKS["cjk_fonts"]="missing"
fi

# ========== 附加检查：noVNC 补丁 ==========
if [ -f "/opt/browser-vnc/noVNC/vnc.html" ] && grep -q "VNC_GATEWAY_PATCH" /opt/browser-vnc/noVNC/vnc.html 2>/dev/null; then
    CHECKS["novnc_patch"]="ok"
else
    CHECKS["novnc_patch"]="missing"
fi

# ========== 版本检测：desktop_mode（不影响 ready 判定）==========
# 判断当前安装是"仅浏览器"（旧版）还是"完整云桌面"（新版）
# 旧版特征：有 Chrome + noVNC 但缺少 xfce4 桌面组件或仍使用 supervisor
# 新版特征：有 xfce4-panel + xfce4-terminal + xfce4-settings + autocutsel + systemd 管理
DESKTOP_MODE="none"
UPGRADE_AVAILABLE=true

# 基础浏览器功能是否存在（真正的 Chrome + noVNC + websockify）
# 复用检查 3 的结果，确保与 Chrome 真伪验证一致
BROWSER_BASE=false
if [ "${CHECKS[chrome]:-}" = "ok" ] && \
   [ -d "/opt/browser-vnc/noVNC" ] && \
   [ -d "/opt/browser-vnc/websockify" ]; then
    BROWSER_BASE=true
fi

if [ "$BROWSER_BASE" = true ]; then
    # 检测云桌面组件（xfce4 全家桶 + autocutsel）
    DESKTOP_PKGS_OK=true
    for pkg_name in xfce4-panel xfce4-terminal xfce4-settings autocutsel; do
        if ! echo "$PKG_INFO" | grep -F "${pkg_name}	" | grep -Fq "install ok installed"; then
            DESKTOP_PKGS_OK=false
            break
        fi
    done

    # 云桌面还需要 systemd 管理（非 supervisor）
    if [ "$DESKTOP_PKGS_OK" = true ] && [ "$SERVICE_MANAGER" = "systemd" ]; then
        DESKTOP_MODE="full"
        UPGRADE_AVAILABLE=false
    else
        DESKTOP_MODE="browser_only"
        UPGRADE_AVAILABLE=true
    fi
fi

# ========== 清理临时文件 ==========
rm -f "$T_PKG" "$T_FC" "$T_SS" "$T_SVC" 2>/dev/null

# ========== 总耗时 ==========
CHECK_END_TS=$(date +%s%N 2>/dev/null || echo "$(date +%s)000000000")
TOTAL_ELAPSED_MS=$(( (CHECK_END_TS - CHECK_START_TS) / 1000000 ))

# ========== 输出 JSON 结果 ==========
MISSING_JSON="[]"
if [ ${#MISSING[@]} -gt 0 ]; then
    MISSING_JSON="["
    first=true
    for item in "${MISSING[@]}"; do
        if [ "$first" = true ]; then first=false; else MISSING_JSON="${MISSING_JSON},"; fi
        MISSING_JSON="${MISSING_JSON}\"${item}\""
    done
    MISSING_JSON="${MISSING_JSON}]"
fi

if [ "$ALL_OK" = true ]; then
    echo "{\"ready\":true,\"elapsed_ms\":${TOTAL_ELAPSED_MS},\"desktop_mode\":\"${DESKTOP_MODE}\",\"upgrade_available\":${UPGRADE_AVAILABLE},\"service_manager\":\"${SERVICE_MANAGER}\",\"checks\":{\"packages\":\"${CHECKS[packages]:-unknown}\",\"novnc\":\"${CHECKS[novnc]:-unknown}\",\"chrome\":\"${CHECKS[chrome]:-unknown}\",\"systemd_config\":\"${CHECKS[systemd_config]:-unknown}\",\"services\":\"${CHECKS[services]:-unknown}\",\"ports\":\"${CHECKS[ports]:-unknown}\",\"cdp_owner\":\"${CHECKS[cdp_owner]:-unknown}\",\"ssl_cert\":\"${CHECKS[ssl_cert]:-unknown}\",\"fcitx5\":\"${CHECKS[fcitx5]:-unknown}\",\"cjk_fonts\":\"${CHECKS[cjk_fonts]:-unknown}\",\"novnc_patch\":\"${CHECKS[novnc_patch]:-unknown}\"}}"
else
    echo "{\"ready\":false,\"elapsed_ms\":${TOTAL_ELAPSED_MS},\"desktop_mode\":\"${DESKTOP_MODE}\",\"upgrade_available\":${UPGRADE_AVAILABLE},\"service_manager\":\"${SERVICE_MANAGER}\",\"checks\":{\"packages\":\"${CHECKS[packages]:-unknown}\",\"novnc\":\"${CHECKS[novnc]:-unknown}\",\"chrome\":\"${CHECKS[chrome]:-unknown}\",\"systemd_config\":\"${CHECKS[systemd_config]:-unknown}\",\"services\":\"${CHECKS[services]:-unknown}\",\"ports\":\"${CHECKS[ports]:-unknown}\",\"cdp_owner\":\"${CHECKS[cdp_owner]:-unknown}\",\"ssl_cert\":\"${CHECKS[ssl_cert]:-unknown}\",\"fcitx5\":\"${CHECKS[fcitx5]:-unknown}\",\"cjk_fonts\":\"${CHECKS[cjk_fonts]:-unknown}\",\"novnc_patch\":\"${CHECKS[novnc_patch]:-unknown}\"},\"missing\":${MISSING_JSON}}"
fi
