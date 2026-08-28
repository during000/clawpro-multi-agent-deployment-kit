#!/bin/bash
if [ -z "${BASH_VERSION:-}" ]; then exec /bin/bash "$0" "$@"; fi
set -uo pipefail

# 输出：最后一行 JSON: {"installed": true} 或 {"installed": false, "error": "..."}
# stdout 仅输出关键进度行 + 最终 JSON（避免 TAT stdout 截断）
# 完整日志写入 LOG_FILE
LOG_DIR="/var/log/clawpro"
mkdir -p "$LOG_DIR"
LOG_FILE="$LOG_DIR/install_browser_vnc.log"
# 将 stderr 和详细输出重定向到日志文件，stdout 保持干净
exec 3>&1                          # fd3 = 原始 stdout（给 TAT 读）
exec >> "$LOG_FILE" 2>&1           # fd1/fd2 全部写日志文件
# log() 同时写日志文件（当前 stdout）和原始 stdout（fd3）
log() { echo "$*"; echo "$*" >&3; }
log ""
log "=== 安装开始: $(date '+%Y-%m-%d %H:%M:%S') ==="

cleanup_tmp() {
    rm -f /tmp/browser-vnc-dl-novnc-status.$$ /tmp/browser-vnc-dl-ws-status.$$
    rm -f /tmp/browser-vnc-chrome-dl-status.$$ /tmp/browser-vnc-wallpaper-status.$$
}
trap cleanup_tmp EXIT

fail() {
    local msg="$1"
    msg=$(echo "$msg" | sed 's/\\/\\\\/g; s/"/\\"/g' | tr '\n' ' ')
    echo "错误: $1" >&2
    # 同时输出到日志和原始 stdout（fd3），确保 TAT 能读到 JSON
    local json="{\"installed\": false, \"error\": \"${msg}\"}"
    echo "$json"
    echo "$json" >&3
    exit 1
}

[ "$(id -u)" = "0" ] || fail "需要以 root 权限运行"

# ========== 幂等性检查：如果完全安装好了，直接返回成功 ==========
# 幂等性检查：所有条件都满足才跳过安装
idempotent_check() {
    local CORE_PKGS=(
        tigervnc-standalone-server openbox dbus dbus-x11
        xfce4-panel xfce4-terminal xfce4-settings xfdesktop4
        fcitx5 fcitx5-chinese-addons fonts-wqy-zenhei
        xclip autocutsel locales python3-numpy psmisc wget curl jq
        fonts-liberation libu2f-udev
    )
    local PKG_STATUS
    PKG_STATUS=$(dpkg-query -W -f='${Package}\t${Status}\n' "${CORE_PKGS[@]}" 2>/dev/null) || return 1
    for pkg in "${CORE_PKGS[@]}"; do
        echo "$PKG_STATUS" | grep -Fq "${pkg}	install ok installed" || return 1
    done

    command -v google-chrome-stable >/dev/null 2>&1 || return 1
    local CHROME_REAL
    CHROME_REAL=$(readlink -f "$(which google-chrome-stable)" 2>/dev/null || echo "")
    echo "$CHROME_REAL" | grep -qiE 'chromium' && return 1
    google-chrome-stable --version 2>/dev/null | grep -qi "Google Chrome" || return 1

    [ -d "/opt/browser-vnc/noVNC" ] && [ -f "/opt/browser-vnc/noVNC/vnc.html" ] || return 1
    [ -d "/opt/browser-vnc/websockify" ] && [ -f "/opt/browser-vnc/websockify/run" ] || return 1
    grep -q "VNC_GATEWAY_PATCH" /opt/browser-vnc/noVNC/vnc.html 2>/dev/null || return 1

    [ -f "/opt/browser-vnc/cert.pem" ] && [ -f "/opt/browser-vnc/key.pem" ] || return 1
    openssl x509 -checkend 2592000 -noout -in "/opt/browser-vnc/cert.pem" 2>/dev/null || return 1

    [ -x "/opt/browser-vnc/start-chromium.sh" ] || return 1
    [ -x "/opt/browser-vnc/start-openbox.sh" ] || return 1
    [ -x "/opt/browser-vnc/start-browser-session.sh" ] || return 1
    [ -f "/opt/browser-vnc/openbox.xml" ] || return 1

    local REQUIRED_UNITS=(browser-vnc-xvnc browser-vnc-websockify browser-vnc-openbox browser-vnc-session browser-vnc-chromium)
    for unit in "${REQUIRED_UNITS[@]}"; do [ -f "/etc/systemd/system/${unit}.service" ] || return 1; done

    local SVC_STATUS SVC_COUNT
    SVC_STATUS=$(systemctl is-active "${REQUIRED_UNITS[@]}" 2>/dev/null) || true
    SVC_COUNT=$(echo "$SVC_STATUS" | grep -c '^active$')
    [ "$SVC_COUNT" -eq ${#REQUIRED_UNITS[@]} ] || return 1

    local SS_OUT
    SS_OUT=$(ss -tlnp 2>/dev/null)
    echo "$SS_OUT" | grep -q ':5900 ' || return 1
    echo "$SS_OUT" | grep -q ':6080 ' || return 1
    echo "$SS_OUT" | grep -q ':9222 ' || return 1

    local CDP_PID
    CDP_PID=$(echo "$SS_OUT" | grep ':9222 ' | grep -oP 'pid=\K[0-9]+' | head -1)
    [ -n "$CDP_PID" ] || return 1
    local CDP_CMD
    CDP_CMD=$(cat /proc/$CDP_PID/cmdline 2>/dev/null | tr '\0' ' ')
    echo "$CDP_CMD" | grep -q '/opt/browser-vnc/chrome-data' || return 1
    echo "$CDP_CMD" | grep -q '\-\-headless' && return 1
    curl -s --max-time 2 http://localhost:9222/json/version >/dev/null 2>&1 || return 1

    local OPENCLAW_CFG="$HOME/.openclaw/openclaw.json"
    if [ -f "$OPENCLAW_CFG" ] && command -v jq >/dev/null 2>&1; then
        jq -e '.browser.enabled == true' "$OPENCLAW_CFG" >/dev/null 2>&1 || return 1
        jq -e '.browser.profiles.user.driver == "existing-session"' "$OPENCLAW_CFG" >/dev/null 2>&1 || return 1
        jq -e '.browser.profiles.user.cdpUrl == "http://localhost:9222"' "$OPENCLAW_CFG" >/dev/null 2>&1 || return 1
    fi
    XDG_RUNTIME_DIR=/run/user/0 systemctl --user is-active --quiet openclaw-gateway 2>/dev/null || return 1

    [ -d "/opt/browser-vnc/chrome-data" ] || return 1
    local CHROME_LINK
    CHROME_LINK=$(readlink -f /root/.config/google-chrome 2>/dev/null || echo "")
    [ "$CHROME_LINK" = "/opt/browser-vnc/chrome-data" ] || return 1
    locale -a 2>/dev/null | grep -qi 'zh_CN.utf' || return 1
    [ -f "/usr/share/applications/google-chrome-vnc.desktop" ] || return 1
    [ -e "/root/Desktop/google-chrome-vnc.desktop" ] || return 1
    return 0
}

if idempotent_check; then
    log "=== 幂等性检查通过：所有组件已安装且运行正常，跳过安装 ==="
    echo '{"installed": true}'
    echo '{"installed": true}' >&3
    exit 0
fi
log "=== 幂等性检查未通过，执行完整安装流程 ==="

INSTALL_START_TS=$(date +%s)

log ""
log "[0/7] 检查 APT 源..."
ensure_apt_source() {
    if apt-get update -qq 2>/dev/null; then APT_SOURCE_UPDATED=true; return 0; fi
    local codename
    codename=$(lsb_release -cs 2>/dev/null || grep VERSION_CODENAME /etc/os-release 2>/dev/null | cut -d= -f2)
    [ -z "$codename" ] && return 1
    cp /etc/apt/sources.list /etc/apt/sources.list.bak.$(date +%s) 2>/dev/null || true
    local mirror_host="mirrors.tencentyun.com"
    curl -s --max-time 3 "http://${mirror_host}/" >/dev/null 2>&1 || mirror_host="mirrors.tencent.com"
    if [ -f /etc/apt/sources.list.d/ubuntu.sources ]; then
        cat > /etc/apt/sources.list.d/ubuntu.sources << SRCEOF
Types: deb
URIs: http://${mirror_host}/ubuntu/
Suites: ${codename} ${codename}-updates ${codename}-security
Components: main restricted universe multiverse
Signed-By: /usr/share/keyrings/ubuntu-archive-keyring.gpg
SRCEOF
    else
        cat > /etc/apt/sources.list << SRCEOF
deb http://${mirror_host}/ubuntu/ ${codename} main restricted universe multiverse
deb http://${mirror_host}/ubuntu/ ${codename}-security main restricted universe multiverse
deb http://${mirror_host}/ubuntu/ ${codename}-updates main restricted universe multiverse
SRCEOF
    fi
    log "  ✓ 切换镜像源: ${mirror_host}"
}
APT_SOURCE_UPDATED=false
ensure_apt_source

log ""
log "[1/7] 更新包列表..."
APT_UPDATE_START=$(date +%s)
if [ "$APT_SOURCE_UPDATED" = true ]; then
    log "  ✓ 已完成，跳过"
else
    apt-get update -qq || fail "apt-get update 失败"
fi
APT_UPDATE_ELAPSED=$(( $(date +%s) - APT_UPDATE_START ))
log "  ✓ apt update (${APT_UPDATE_ELAPSED}s)"

log ""
log "[2/7] 安装系统依赖包..."
DEBIAN_FRONTEND=noninteractive apt-get --fix-broken install -y -qq 2>/dev/null || true
dpkg --configure -a 2>/dev/null || true

APT_PACKAGES=(
    tigervnc-standalone-server x11-xserver-utils openbox
    dbus dbus-x11
    xfce4-panel xfce4-terminal xfce4-settings xfdesktop4 thunar
    fcitx5 fcitx5-chinese-addons fcitx5-frontend-gtk3
    fonts-wqy-zenhei
    xclip xsel autocutsel
    locales python3-numpy psmisc wget curl jq xdg-utils
    fonts-liberation libu2f-udev
)

CHROME_DEP_BASES=(
    libatk-bridge2.0-0 libatk1.0-0 libatspi2.0-0 libcups2 libdrm2
    libgbm1 libgtk-3-0 libnspr4 libnss3 libxcomposite1
    libxdamage1 libxfixes3 libxkbcommon0 libxrandr2 libvulkan1
)
ALL_CANDIDATES=()
for base in "${CHROME_DEP_BASES[@]}"; do ALL_CANDIDATES+=("${base}t64" "$base"); done
AVAILABLE_PKGS=$(apt-cache show "${ALL_CANDIDATES[@]}" 2>/dev/null | grep '^Package:' | awk '{print $2}' | sort -u)
for base in "${CHROME_DEP_BASES[@]}"; do
    if echo "$AVAILABLE_PKGS" | grep -qx "${base}t64"; then
        APT_PACKAGES+=("${base}t64")
    else
        APT_PACKAGES+=("$base")
    fi
done

CHROME_DEB="/tmp/google-chrome-stable_current_amd64.deb"
CHROME_BG_PID=""
CHROME_BG_STATUS="/tmp/browser-vnc-chrome-dl-status.$$"
if ! command -v google-chrome-stable >/dev/null 2>&1; then
    log "  ⚡ 后台预下载 Chrome..."
    (
        CHROME_URLS=(
            "https://finnie-1258344699.cos.ap-guangzhou.myqcloud.com/chrome/google-chrome-stable_current_amd64.deb"
            "https://dl.google.com/linux/direct/google-chrome-stable_current_amd64.deb"
        )
        for url in "${CHROME_URLS[@]}"; do
            if wget --timeout=30 --tries=1 -qO "$CHROME_DEB" "$url" 2>/dev/null; then
                if [ -s "$CHROME_DEB" ] && file "$CHROME_DEB" 2>/dev/null | grep -q 'Debian'; then
                    echo "OK:$url" > "$CHROME_BG_STATUS"
                    exit 0
                fi
                rm -f "$CHROME_DEB"
            fi
        done
        rm -f "$CHROME_DEB"
        echo "FAIL" > "$CHROME_BG_STATUS"
    ) &
    CHROME_BG_PID=$!
fi

mkdir -p /opt/browser-vnc
NOVNC_DIR=/opt/browser-vnc/noVNC
WEBSOCKIFY_DIR=/opt/browser-vnc/websockify
NOVNC_BG_PID=""
WS_BG_PID=""

if [ ! -d "$NOVNC_DIR" ] || [ ! -f "$NOVNC_DIR/vnc.html" ]; then
    log "  ⚡ 后台预下载 noVNC..."
    (
        rm -rf "$NOVNC_DIR"
        for url in \
            "https://mirrors.tencent.com/github.com/novnc/noVNC/archive/refs/tags/v1.5.0.tar.gz" \
            "https://ghfast.top/https://github.com/novnc/noVNC/archive/refs/tags/v1.5.0.tar.gz" \
            "https://github.com/novnc/noVNC/archive/refs/tags/v1.5.0.tar.gz"; do
            if wget --timeout=30 --tries=1 -qO /tmp/noVNC.tar.gz "$url" 2>/dev/null; then
                if [ -s /tmp/noVNC.tar.gz ] && file /tmp/noVNC.tar.gz 2>/dev/null | grep -q 'gzip'; then
                    tar -xzf /tmp/noVNC.tar.gz -C /opt/browser-vnc/
                    mv /opt/browser-vnc/noVNC-1.5.0 "$NOVNC_DIR"
                    rm -f /tmp/noVNC.tar.gz
                    echo "OK:$url" > /tmp/browser-vnc-dl-novnc-status.$$
                    exit 0
                fi
                rm -f /tmp/noVNC.tar.gz
            fi
        done
        echo "FAIL" > /tmp/browser-vnc-dl-novnc-status.$$
        exit 1
    ) &
    NOVNC_BG_PID=$!
fi

if [ ! -d "$WEBSOCKIFY_DIR" ] || [ ! -f "$WEBSOCKIFY_DIR/run" ]; then
    log "  ⚡ 后台预下载 websockify..."
    (
        rm -rf "$WEBSOCKIFY_DIR"
        for url in \
            "https://mirrors.tencent.com/github.com/novnc/websockify/archive/refs/tags/v0.12.0.tar.gz" \
            "https://ghfast.top/https://github.com/novnc/websockify/archive/refs/tags/v0.12.0.tar.gz" \
            "https://github.com/novnc/websockify/archive/refs/tags/v0.12.0.tar.gz"; do
            if wget --timeout=30 --tries=1 -qO /tmp/websockify.tar.gz "$url" 2>/dev/null; then
                if [ -s /tmp/websockify.tar.gz ] && file /tmp/websockify.tar.gz 2>/dev/null | grep -q 'gzip'; then
                    tar -xzf /tmp/websockify.tar.gz -C /opt/browser-vnc/
                    mv /opt/browser-vnc/websockify-0.12.0 "$WEBSOCKIFY_DIR"
                    rm -f /tmp/websockify.tar.gz
                    echo "OK:$url" > /tmp/browser-vnc-dl-ws-status.$$
                    exit 0
                fi
                rm -f /tmp/websockify.tar.gz
            fi
        done
        echo "FAIL" > /tmp/browser-vnc-dl-ws-status.$$
        exit 1
    ) &
    WS_BG_PID=$!
fi

PKGS_TO_INSTALL=()
INSTALLED_STATUS=$(dpkg-query -W -f='${Package}\t${Status}\n' "${APT_PACKAGES[@]}" 2>/dev/null || true)
for pkg in "${APT_PACKAGES[@]}"; do
    echo "$INSTALLED_STATUS" | grep -Fq "${pkg}	install ok installed" || PKGS_TO_INSTALL+=("$pkg")
done

log "  已安装: $((${#APT_PACKAGES[@]} - ${#PKGS_TO_INSTALL[@]})), 需安装: ${#PKGS_TO_INSTALL[@]}"

APT_INSTALL_START=$(date +%s)
if [ ${#PKGS_TO_INSTALL[@]} -gt 0 ]; then
    DEBIAN_FRONTEND=noninteractive apt-get install -y -q \
        --no-install-recommends \
        -o Dpkg::Options::="--force-confdef" \
        -o Dpkg::Options::="--force-confold" \
        -o APT::Status-Fd=-1 \
        "${PKGS_TO_INSTALL[@]}" \
        || fail "安装系统依赖包失败"
else
    log "  ✓ 全部已安装"
fi
APT_INSTALL_ELAPSED=$(( $(date +%s) - APT_INSTALL_START ))
log "  ✓ APT 安装完成 (${APT_INSTALL_ELAPSED}s)"

localedef -i zh_CN -c -f UTF-8 -A /usr/share/locale/locale.alias zh_CN.UTF-8 2>/dev/null || true
[ -f /etc/default/locale ] && cp /etc/default/locale /etc/default/locale.bak.browser-vnc 2>/dev/null || true
cat > /etc/default/locale << 'LOCALE_EOF'
LANG=zh_CN.UTF-8
LC_ALL=zh_CN.UTF-8
LANGUAGE=zh_CN:zh
LOCALE_EOF
grep -q 'LANG=zh_CN.UTF-8' /etc/environment 2>/dev/null || echo 'LANG=zh_CN.UTF-8' >> /etc/environment
export LANG=zh_CN.UTF-8
export LC_ALL=zh_CN.UTF-8
fc-cache -f 2>/dev/null || true
mkdir -p /opt/browser-vnc/chrome-data/.config/fontconfig
cat > /opt/browser-vnc/chrome-data/.config/fontconfig/fonts.conf << 'FONTCONF_EOF'
<?xml version="1.0"?><!DOCTYPE fontconfig SYSTEM "fonts.dtd"><fontconfig><alias><family>sans-serif</family><prefer><family>WenQuanYi Zen Hei</family><family>DejaVu Sans</family></prefer></alias><alias><family>serif</family><prefer><family>WenQuanYi Zen Hei</family><family>DejaVu Serif</family></prefer></alias><alias><family>monospace</family><prefer><family>WenQuanYi Zen Hei Mono</family><family>DejaVu Sans Mono</family></prefer></alias></fontconfig>
FONTCONF_EOF
mkdir -p /root/.config/fontconfig
cp /opt/browser-vnc/chrome-data/.config/fontconfig/fonts.conf /root/.config/fontconfig/fonts.conf 2>/dev/null || true

if command -v fcitx5 >/dev/null 2>&1; then
    FCITX_CONF="/opt/browser-vnc/chrome-data/.config/fcitx5"
    mkdir -p "$FCITX_CONF/conf" "/opt/browser-vnc/chrome-data/.local/share/fcitx5"
    cat > "$FCITX_CONF/profile" << 'FCITX_PROFILE'
[Groups/0]
Name=Default
Default Layout=us
DefaultIM=pinyin

[Groups/0/Items/0]
Name=keyboard-us
Layout=

[Groups/0/Items/1]
Name=pinyin
Layout=

[GroupOrder]
0=Default
FCITX_PROFILE
    cat > "$FCITX_CONF/config" << 'FCITX_CONFIG'
[Hotkey]
EnumerateWithTriggerKeys=True
[Hotkey/TriggerKeys]
0=Control+space
1=Super+space
[Hotkey/EnumerateForwardKeys]
0=Control+space
[Hotkey/EnumerateBackwardKeys]
0=Control+Shift+space
[Behavior]
DefaultPageSize=5
ShareInputState=All
PreeditEnabledByDefault=True
FCITX_CONFIG
fi

GLOBAL_DESKTOP_DIR='/usr/share/applications'
mkdir -p "$GLOBAL_DESKTOP_DIR"
cat > "${GLOBAL_DESKTOP_DIR}/google-chrome-vnc.desktop" << 'DESKTOP_EOF'
[Desktop Entry]
Type=Application
Name=Google Chrome
GenericName=Web Browser
Exec=/usr/bin/google-chrome-stable --no-sandbox --disable-setuid-sandbox --no-first-run --no-default-browser-check --password-store=basic --disable-session-crashed-bubble --disable-infobars --disable-dev-shm-usage --disable-gpu %U
Icon=google-chrome
Terminal=false
Categories=Network;WebBrowser;
MimeType=text/html;text/xml;application/xhtml+xml;x-scheme-handler/http;x-scheme-handler/https;
StartupNotify=true
DESKTOP_EOF
chmod 0644 "${GLOBAL_DESKTOP_DIR}/google-chrome-vnc.desktop"
update-desktop-database "$GLOBAL_DESKTOP_DIR" 2>/dev/null || true

DESKTOP_DIR="/root/Desktop"
mkdir -p "$DESKTOP_DIR"
rm -f "${DESKTOP_DIR}/google-chrome-vnc.desktop"
ln -sf "${GLOBAL_DESKTOP_DIR}/google-chrome-vnc.desktop" "${DESKTOP_DIR}/google-chrome-vnc.desktop"
if command -v xfce4-terminal >/dev/null 2>&1; then
    if [ -f "${GLOBAL_DESKTOP_DIR}/xfce4-terminal.desktop" ]; then
        ln -sf "${GLOBAL_DESKTOP_DIR}/xfce4-terminal.desktop" "${DESKTOP_DIR}/xfce4-terminal.desktop"
    else
        printf '[Desktop Entry]\nType=Application\nName=Terminal\nExec=xfce4-terminal\nIcon=utilities-terminal\nTerminal=false\n' > "${DESKTOP_DIR}/xfce4-terminal.desktop"
        chmod 0755 "${DESKTOP_DIR}/xfce4-terminal.desktop"
    fi
fi
if command -v thunar >/dev/null 2>&1; then
    [ -f "${GLOBAL_DESKTOP_DIR}/thunar.desktop" ] && ln -sf "${GLOBAL_DESKTOP_DIR}/thunar.desktop" "${DESKTOP_DIR}/thunar.desktop"
fi

mkdir -p /root/.config
cat > /root/.config/mimeapps.list << 'MIME_EOF'
[Default Applications]
x-scheme-handler/http=google-chrome-vnc.desktop
x-scheme-handler/https=google-chrome-vnc.desktop
text/html=google-chrome-vnc.desktop
MIME_EOF

AUTOSTART_DIR="/root/.config/autostart"
mkdir -p "$AUTOSTART_DIR"
[ -f /etc/xdg/autostart/xfce-polkit.desktop ] && \
    printf '[Desktop Entry]\nType=Application\nName=XFCE PolicyKit Agent (disabled)\nHidden=true\n' > "${AUTOSTART_DIR}/xfce-polkit.desktop"

COLORD_POLICY='/usr/share/polkit-1/actions/org.freedesktop.color-manager.policy'
mkdir -p "$(dirname "$COLORD_POLICY")"
cat > "$COLORD_POLICY" << 'COLORD_EOF'
<?xml version="1.0" encoding="UTF-8"?><!DOCTYPE policyconfig PUBLIC "-//freedesktop//DTD PolicyKit Policy Configuration 1.0//EN" "http://www.freedesktop.org/standards/PolicyKit/1/policyconfig.dtd"><policyconfig><action id="org.freedesktop.color-manager.create-device"><defaults><allow_any>yes</allow_any><allow_inactive>yes</allow_inactive><allow_active>yes</allow_active></defaults></action><action id="org.freedesktop.color-manager.create-profile"><defaults><allow_any>yes</allow_any><allow_inactive>yes</allow_inactive><allow_active>yes</allow_active></defaults></action><action id="org.freedesktop.color-manager.modify-device"><defaults><allow_any>yes</allow_any><allow_inactive>yes</allow_inactive><allow_active>yes</allow_active></defaults></action><action id="org.freedesktop.color-manager.modify-profile"><defaults><allow_any>yes</allow_any><allow_inactive>yes</allow_inactive><allow_active>yes</allow_active></defaults></action></policyconfig>
COLORD_EOF
chmod 0644 "$COLORD_POLICY"
systemctl restart polkit 2>/dev/null || true

cat > /opt/browser-vnc/trust-desktop-icons.sh << 'TRUST_EOF'
#!/bin/sh
command -v gio >/dev/null 2>&1 || exit 0
for f in /root/Desktop/*.desktop; do
    [ -f "$f" ] || continue
    chmod +x "$f" 2>/dev/null || true
    gio set "$f" metadata::trusted true 2>/dev/null || true
done
command -v xfdesktop >/dev/null 2>&1 && xfdesktop --reload >/dev/null 2>&1 || true
TRUST_EOF
chmod +x /opt/browser-vnc/trust-desktop-icons.sh
cat > "${AUTOSTART_DIR}/vnc-trust-icons.desktop" << 'TRUST_AUTO_EOF'
[Desktop Entry]
Type=Application
Name=Trust Desktop Launchers
Exec=sh -c "sleep 3 && /opt/browser-vnc/trust-desktop-icons.sh"
TRUST_AUTO_EOF

WALLPAPER_FILE="/usr/share/backgrounds/vnc-gateway-wallpaper.png"
WALLPAPER_BG_PID=""
mkdir -p /usr/share/backgrounds
(
    wget --timeout=15 --tries=2 -qO "$WALLPAPER_FILE" "https://finnie-1258344699.cos.ap-guangzhou.myqcloud.com/wallpaper/4.png" 2>/dev/null && \
        [ -s "$WALLPAPER_FILE" ] && chmod 0644 "$WALLPAPER_FILE" && echo "OK" > /tmp/browser-vnc-wallpaper-status.$$ || \
        echo "FAIL" > /tmp/browser-vnc-wallpaper-status.$$
    if [ ! -s "$WALLPAPER_FILE" ]; then
        printf '<svg xmlns="http://www.w3.org/2000/svg" width="1920" height="1080"><rect width="100%%" height="100%%" fill="#1a1a2e"/></svg>' > "${WALLPAPER_FILE%.png}.svg"
        echo "FALLBACK" > /tmp/browser-vnc-wallpaper-status.$$
    fi
) &
WALLPAPER_BG_PID=$!

log ""
log "[3/7] 创建目录和配置..."
mkdir -p /opt/browser-vnc/chrome-data/.local/share/applications /opt/browser-vnc/chrome-data/Crashpad
mkdir -p /root/.config
[ -d /root/.config/google-chrome ] && [ ! -L /root/.config/google-chrome ] && \
    mv /root/.config/google-chrome /root/.config/google-chrome.bak.$(date +%s) 2>/dev/null || true
ln -sf /opt/browser-vnc/chrome-data /root/.config/google-chrome

SSL_CERT="/opt/browser-vnc/cert.pem"
SSL_KEY="/opt/browser-vnc/key.pem"
SSL_NEED_GEN=false
[ ! -f "$SSL_CERT" ] || [ ! -f "$SSL_KEY" ] && SSL_NEED_GEN=true
[ "$SSL_NEED_GEN" = false ] && ! openssl x509 -checkend 2592000 -noout -in "$SSL_CERT" 2>/dev/null && SSL_NEED_GEN=true
if [ "$SSL_NEED_GEN" = true ]; then
    openssl req -new -x509 -days 3650 -nodes \
        -out "$SSL_CERT" -keyout "$SSL_KEY" -subj "/CN=browser-vnc" \
        2>/dev/null || fail "生成 SSL 证书失败"
    chmod 644 "$SSL_CERT"; chmod 600 "$SSL_KEY"
fi

log ""
log "[4/7] 等待 noVNC + websockify..."
DL_START=$(date +%s)
if [ -n "${NOVNC_BG_PID:-}" ]; then
    wait "$NOVNC_BG_PID" || fail "下载 noVNC 失败"
    rm -f /tmp/browser-vnc-dl-novnc-status.$$
fi
if [ -n "${WS_BG_PID:-}" ]; then
    wait "$WS_BG_PID" || fail "下载 websockify 失败"
    rm -f /tmp/browser-vnc-dl-ws-status.$$
fi
DL_ELAPSED=$(( $(date +%s) - DL_START ))
log "  ✓ noVNC+websockify (${DL_ELAPSED}s)"

NOVNC_HTML="/opt/browser-vnc/noVNC/vnc.html"
NOVNC_PATCH_MARKER="<!-- VNC_GATEWAY_PATCH -->"
if [ -f "$NOVNC_HTML" ] && ! grep -q "$NOVNC_PATCH_MARKER" "$NOVNC_HTML"; then
    PATCH_CSS="$NOVNC_PATCH_MARKER<style>#noVNC_control_bar_anchor,#noVNC_control_bar_hint,#noVNC_transition,#noVNC_transition_text{display:none!important}html,body,#noVNC_container,#noVNC_screen{background:#000!important;width:100%!important;height:100%!important;margin:0!important;padding:0!important}</style>"
    sed -i "s|</head>|${PATCH_CSS}\n</head>|" "$NOVNC_HTML" 2>/dev/null || true
fi

log ""
log "[5/7] 安装 Google Chrome..."
if ! command -v google-chrome-stable >/dev/null 2>&1; then
    CHROME_DOWNLOADED=false
    CHROME_SOURCE="unknown"
    CHROME_INSTALL_START=$(date +%s)
    if [ -n "${CHROME_BG_PID:-}" ]; then
        wait "$CHROME_BG_PID" 2>/dev/null || true
        BG_RESULT=$(cat "$CHROME_BG_STATUS" 2>/dev/null || echo "FAIL")
        rm -f "$CHROME_BG_STATUS"
        if [[ "$BG_RESULT" == OK:* ]]; then
            CHROME_DOWNLOADED=true
            CHROME_SOURCE="${BG_RESULT#OK:}"
        fi
    fi
    if [ "$CHROME_DOWNLOADED" != true ]; then
        rm -f "$CHROME_DEB"
        fail "Google Chrome 下载失败，不支持 Chromium 替代"
    fi
    if [ -f "$CHROME_DEB" ]; then
        dpkg -i "$CHROME_DEB" 2>/dev/null || DEBIAN_FRONTEND=noninteractive apt-get install -f -y -q || fail "安装 Google Chrome 失败"
        rm -f "$CHROME_DEB"
    fi
    CHROME_INSTALL_ELAPSED=$(( $(date +%s) - CHROME_INSTALL_START ))
    CHROME_REAL_PATH=$(readlink -f /usr/bin/google-chrome-stable 2>/dev/null || echo "")
    [ -z "$CHROME_REAL_PATH" ] || [ ! -x "$CHROME_REAL_PATH" ] && fail "google-chrome-stable 不存在或不可执行"
    echo "$CHROME_REAL_PATH" | grep -qiE 'chromium' && fail "google-chrome-stable 指向 Chromium ($CHROME_REAL_PATH)"
    CHROME_VER_FULL=$(google-chrome-stable --version 2>/dev/null || echo "")
    echo "$CHROME_VER_FULL" | grep -qi "Google Chrome" || fail "不是 Google Chrome: ${CHROME_VER_FULL}"
    CHROME_VER=$(echo "$CHROME_VER_FULL" | awk '{print $NF}')
    ln -sf /usr/bin/google-chrome-stable /usr/local/bin/chromium-browser
    log "  ✓ Chrome ${CHROME_VER} (${CHROME_SOURCE}, ${CHROME_INSTALL_ELAPSED}s)"
else
    CHROME_REAL_PATH=$(readlink -f /usr/bin/google-chrome-stable 2>/dev/null || echo "")
    if echo "$CHROME_REAL_PATH" | grep -qiE 'chromium'; then
        rm -f /usr/bin/google-chrome-stable
        fail "google-chrome-stable 是 Chromium 符号链接"
    fi
    CHROME_VER_FULL=$(google-chrome-stable --version 2>/dev/null || echo "")
    echo "$CHROME_VER_FULL" | grep -qi "Google Chrome" || fail "不是 Google Chrome: ${CHROME_VER_FULL}"
    CHROME_VER=$(echo "$CHROME_VER_FULL" | awk '{print $NF}')
    log "  ✓ Chrome 已安装: ${CHROME_VER}"
fi

log ""
log "[6/7] 创建启动脚本..."

cat > /opt/browser-vnc/start-browser-session.sh << 'SCRIPT_EOF'
#!/bin/bash
export DISPLAY=:99.0 HOME=/opt/browser-vnc/chrome-data
export LANG=zh_CN.UTF-8 LC_ALL=zh_CN.UTF-8
export XDG_CONFIG_HOME=/opt/browser-vnc/chrome-data/.config
export XDG_DATA_HOME=/opt/browser-vnc/chrome-data/.local/share
export XDG_RUNTIME_DIR=/tmp/runtime-browser
if command -v fcitx5 >/dev/null 2>&1; then
    export GTK_IM_MODULE=fcitx QT_IM_MODULE=fcitx XMODIFIERS=@im=fcitx SDL_IM_MODULE=fcitx INPUT_METHOD=fcitx
fi
mkdir -p "$XDG_RUNTIME_DIR"; chmod 700 "$XDG_RUNTIME_DIR"
for i in $(seq 1 100); do xdpyinfo -display :99 >/dev/null 2>&1 && break; sleep 0.3; done
eval "$(dbus-launch --sh-syntax)"
cat > /tmp/browser-dbus-env << ENVEOF
export DBUS_SESSION_BUS_ADDRESS="$DBUS_SESSION_BUS_ADDRESS"
export DBUS_SESSION_BUS_PID="$DBUS_SESSION_BUS_PID"
ENVEOF
chmod 644 /tmp/browser-dbus-env
if command -v autocutsel >/dev/null 2>&1; then
    autocutsel -fork >/dev/null 2>&1 || true
    autocutsel -selection PRIMARY -fork >/dev/null 2>&1 || true
fi
command -v fcitx5 >/dev/null 2>&1 && fcitx5 -d --replace >/dev/null 2>&1 &
wait "$DBUS_SESSION_BUS_PID" 2>/dev/null || sleep infinity
SCRIPT_EOF
chmod +x /opt/browser-vnc/start-browser-session.sh

cat > /opt/browser-vnc/start-chromium.sh << 'SCRIPT_EOF'
#!/bin/bash
export DISPLAY=:99.0 HOME=/opt/browser-vnc/chrome-data GOOGLE_API_KEY=""
export LANG=zh_CN.UTF-8 LC_ALL=zh_CN.UTF-8
export XDG_CONFIG_HOME=/opt/browser-vnc/chrome-data/.config
export XDG_DATA_HOME=/opt/browser-vnc/chrome-data/.local/share
export XDG_RUNTIME_DIR=/tmp/runtime-browser
if command -v fcitx5 >/dev/null 2>&1; then
    export GTK_IM_MODULE=fcitx QT_IM_MODULE=fcitx XMODIFIERS=@im=fcitx SDL_IM_MODULE=fcitx INPUT_METHOD=fcitx
fi
mkdir -p "$XDG_RUNTIME_DIR" "$HOME/.local/share/applications" "$HOME/Crashpad"
chmod 700 "$XDG_RUNTIME_DIR"
for i in $(seq 1 100); do xdpyinfo -display :99 >/dev/null 2>&1 && break; sleep 0.3; done
for i in $(seq 1 50); do
    [ -f /tmp/browser-dbus-env ] && { source /tmp/browser-dbus-env; [ -n "$DBUS_SESSION_BUS_ADDRESS" ] && break; }
    sleep 0.3
done
for i in $(seq 1 50); do pgrep -x openbox >/dev/null 2>&1 && break; sleep 0.3; done
command -v fcitx5 >/dev/null 2>&1 && { for i in $(seq 1 30); do pgrep -x fcitx5 >/dev/null 2>&1 && break; sleep 0.3; done; }

pkill -9 -f 'chrome.*--headless' 2>/dev/null || true
pkill -9 -f 'ms-playwright.*chrome' 2>/dev/null || true
pkill -9 -f 'chrome.*--user-data-dir=/opt/browser-vnc/chrome-data' 2>/dev/null || true
ss -tlnp 2>/dev/null | grep -q ':9222 ' && { fuser -k -9 9222/tcp 2>/dev/null || true; sleep 0.5; }

/usr/bin/google-chrome-stable \
    --no-sandbox --disable-setuid-sandbox \
    --user-data-dir=/opt/browser-vnc/chrome-data \
    --hide-crash-restore-bubble --disable-session-crashed-bubble \
    --no-first-run --no-default-browser-check --disable-infobars \
    --disable-gpu --disable-dev-shm-usage \
    --disable-background-timer-throttling \
    --disable-backgrounding-occluded-windows \
    --disable-renderer-backgrounding \
    --disable-features=MediaSessionService \
    --start-maximized --password-store=basic \
    --remote-debugging-port=9222 --remote-debugging-address=127.0.0.1 \
    https://cloud.tencent.com &
CHROME_PID=$!

for i in $(seq 1 30); do
    if curl -s http://localhost:9222/json/version >/dev/null 2>&1; then
        WS_PATH=$(curl -s http://localhost:9222/json/version 2>/dev/null \
            | python3 -c "import sys,json; print(json.load(sys.stdin).get('webSocketDebuggerUrl','').split('localhost:9222',1)[-1])" 2>/dev/null)
        if [ -n "$WS_PATH" ]; then
            echo -e "9222\n$WS_PATH" > /opt/browser-vnc/chrome-data/DevToolsActivePort && chmod 644 /opt/browser-vnc/chrome-data/DevToolsActivePort
            GW_SESSION_DIR="/root/.openclaw/browser-existing-session"
            [ -d "$GW_SESSION_DIR" ] && echo -e "9222\n$WS_PATH" > "$GW_SESSION_DIR/DevToolsActivePort" && chmod 644 "$GW_SESSION_DIR/DevToolsActivePort"
        fi
        break
    fi
    sleep 1
done
wait $CHROME_PID
SCRIPT_EOF
chmod +x /opt/browser-vnc/start-chromium.sh

cat > /opt/browser-vnc/start-openbox.sh << 'SCRIPT_EOF'
#!/bin/bash
export DISPLAY=:99.0 HOME=/root XDG_RUNTIME_DIR=/tmp/runtime-browser
export XDG_CONFIG_HOME=/root/.config XDG_DATA_HOME=/root/.local/share XDG_CURRENT_DESKTOP="XFCE"
export LANG=zh_CN.UTF-8 LC_ALL=zh_CN.UTF-8 LANGUAGE=zh_CN:zh
mkdir -p "$XDG_RUNTIME_DIR"; chmod 700 "$XDG_RUNTIME_DIR"
for i in $(seq 1 100); do xdpyinfo -display :99 >/dev/null 2>&1 && break; sleep 0.3; done
pkill -f xfce4-screensaver 2>/dev/null || true
pkill -f xscreensaver 2>/dev/null || true
xset -display :99 s off s noblank s 0 0 2>/dev/null || true
xset -display :99 dpms 0 0 0 2>/dev/null || true
xset -display :99 -dpms 2>/dev/null || true
xfsettingsd --display=:99 --replace &>/dev/null &
command -v xfdesktop >/dev/null 2>&1 && xfdesktop --display=:99 &>/dev/null &
sleep 1
xfce4-panel --display=:99 &>/dev/null &
(
    sleep 3
    if command -v xfconf-query >/dev/null 2>&1; then
        W="/usr/share/backgrounds/vnc-gateway-wallpaper.png"
        [ -f "$W" ] || W="/usr/share/backgrounds/vnc-gateway-wallpaper.svg"
        if [ -f "$W" ]; then
            for ws in 0 1; do
                xfconf-query -c xfce4-desktop -p "/backdrop/screen0/monitorVNC-0/workspace${ws}/last-image" --create -t string -s "$W" 2>/dev/null || true
                xfconf-query -c xfce4-desktop -p "/backdrop/screen0/monitorVNC-0/workspace${ws}/image-style" --create -t int -s 5 2>/dev/null || true
            done
        fi
        xfconf-query -c xfce4-desktop -p /desktop-icons/file-icons/show-trash --create -t bool -s true 2>/dev/null || true
        xfconf-query -c xfce4-desktop -p /desktop-icons/file-icons/show-home --create -t bool -s true 2>/dev/null || true
        xfconf-query -c xfce4-desktop -p /desktop-icons/file-icons/show-filesystem --create -t bool -s true 2>/dev/null || true
        xfconf-query -c xfce4-screensaver -p /screensaver/enabled --create -t bool -s false 2>/dev/null || true
        xfconf-query -c xfce4-screensaver -p /lock/enabled --create -t bool -s false 2>/dev/null || true
        xfconf-query -c xfce4-power-manager -p /xfce4-power-manager/dpms-enabled --create -t bool -s false 2>/dev/null || true
        xfconf-query -c xfce4-power-manager -p /xfce4-power-manager/blank-on-ac --create -t int -s 0 2>/dev/null || true
        xfconf-query -c xfce4-power-manager -p /xfce4-power-manager/dpms-on-ac-sleep --create -t uint -s 0 2>/dev/null || true
        xfconf-query -c xfce4-power-manager -p /xfce4-power-manager/dpms-on-ac-off --create -t uint -s 0 2>/dev/null || true
        xset -display :99 s off s noblank s 0 0 2>/dev/null || true
        xset -display :99 dpms 0 0 0 2>/dev/null || true
        xset -display :99 -dpms 2>/dev/null || true
    fi
    [ -x /opt/browser-vnc/trust-desktop-icons.sh ] && /opt/browser-vnc/trust-desktop-icons.sh
) &>/dev/null &
exec openbox --config-file /opt/browser-vnc/openbox.xml
SCRIPT_EOF
chmod +x /opt/browser-vnc/start-openbox.sh

cat > /opt/browser-vnc/openbox.xml << 'SCRIPT_EOF'
<?xml version="1.0" encoding="UTF-8"?><openbox_config><desktops><number>1</number></desktops></openbox_config>
SCRIPT_EOF

log ""
log "[7/7] 配置 systemd 并启动..."
if [ -n "${WALLPAPER_BG_PID:-}" ]; then
    wait "$WALLPAPER_BG_PID" 2>/dev/null || true
    rm -f /tmp/browser-vnc-wallpaper-status.$$
fi

cat > /etc/systemd/system/browser-vnc-xvnc.service << 'UNIT_EOF'
[Unit]
Description=Browser VNC - Xvnc
After=network.target
[Service]
Type=simple
ExecStartPre=-/bin/rm -f /tmp/.X99-lock /tmp/.X11-unix/X99
ExecStart=/usr/bin/Xvnc :99 -depth 24 -geometry 1920x1080 -rfbport 5900 -localhost yes -AlwaysShared -SecurityTypes None -AcceptCutText -SendCutText
Restart=on-failure
RestartSec=2
[Install]
WantedBy=multi-user.target
UNIT_EOF

cat > /etc/systemd/system/browser-vnc-websockify.service << 'UNIT_EOF'
[Unit]
Description=Browser VNC - websockify
After=browser-vnc-xvnc.service
Requires=browser-vnc-xvnc.service
[Service]
Type=simple
ExecStart=/opt/browser-vnc/websockify/run --cert=/opt/browser-vnc/cert.pem --key=/opt/browser-vnc/key.pem 6080 localhost:5900
Restart=on-failure
RestartSec=2
[Install]
WantedBy=multi-user.target
UNIT_EOF

cat > /etc/systemd/system/browser-vnc-openbox.service << 'UNIT_EOF'
[Unit]
Description=Browser VNC - Openbox
After=browser-vnc-xvnc.service browser-vnc-session.service
Requires=browser-vnc-xvnc.service
[Service]
Type=simple
ExecStart=/opt/browser-vnc/start-openbox.sh
Restart=on-failure
RestartSec=2
[Install]
WantedBy=multi-user.target
UNIT_EOF

cat > /etc/systemd/system/browser-vnc-session.service << 'UNIT_EOF'
[Unit]
Description=Browser VNC - D-Bus session
After=browser-vnc-xvnc.service
Requires=browser-vnc-xvnc.service
[Service]
Type=simple
ExecStart=/opt/browser-vnc/start-browser-session.sh
Restart=on-failure
RestartSec=2
[Install]
WantedBy=multi-user.target
UNIT_EOF

cat > /etc/systemd/system/browser-vnc-chromium.service << 'UNIT_EOF'
[Unit]
Description=Browser VNC - Chrome CDP
After=browser-vnc-session.service browser-vnc-openbox.service
Requires=browser-vnc-xvnc.service
StartLimitIntervalSec=300
StartLimitBurst=10
[Service]
Type=simple
ExecStart=/opt/browser-vnc/start-chromium.sh
Restart=always
RestartSec=5
TimeoutStartSec=120
[Install]
WantedBy=multi-user.target
UNIT_EOF

# 停止旧服务并清理
command -v supervisorctl >/dev/null 2>&1 && {
    supervisorctl stop browser-vnc-chromium browser-vnc-session browser-vnc-openbox browser-vnc-websockify browser-vnc-xvnc 2>/dev/null || true
    rm -f /etc/supervisor/conf.d/browser-vnc.conf; supervisorctl reread 2>/dev/null; supervisorctl update 2>/dev/null
} || true
systemctl stop browser-vnc-chromium browser-vnc-session browser-vnc-openbox browser-vnc-websockify browser-vnc-xvnc 2>/dev/null || true

# 接管 CDP 端口
for cmd in 'stop openclaw-gateway' 'stop lighthouse-chromium' 'disable lighthouse-chromium'; do
    runuser -l root -c "XDG_RUNTIME_DIR=/run/user/0 systemctl --user $cmd" 2>/dev/null || \
        XDG_RUNTIME_DIR=/run/user/0 systemctl --user $cmd 2>/dev/null || true
done

pkill -9 -f 'chrome.*--headless' 2>/dev/null || true
pkill -9 -f 'ms-playwright.*chrome' 2>/dev/null || true
pkill -9 -f '/opt/google/chrome/chrome' 2>/dev/null || true
for p in 9222 5900 6080; do fuser -k $p/tcp 2>/dev/null || true; done
sleep 1
ss -tlnp 2>/dev/null | grep -q ':9222 ' && { fuser -k -9 9222/tcp 2>/dev/null || true; sleep 1; }
rm -f /tmp/browser-dbus-env /opt/browser-vnc/chrome-data/Singleton{Lock,Socket,Cookie}
rm -f /var/log/browser-vnc-{xvnc,websockify,openbox,session,chromium}.{log,err.log} 2>/dev/null

log "启动服务..."
SVC_START_TS=$(date +%s)
BVNC_SVCS="browser-vnc-xvnc browser-vnc-websockify browser-vnc-openbox browser-vnc-session browser-vnc-chromium"
systemctl daemon-reload
systemctl enable $BVNC_SVCS
systemctl start $BVNC_SVCS

log "等待 CDP 就绪..."
CDP_READY=false
for i in $(seq 1 90); do
    if curl -s --max-time 1 http://localhost:9222/json/version >/dev/null 2>&1; then
        CDP_READY=true
        break
    fi
    sleep 1
done
SVC_ELAPSED=$(( $(date +%s) - SVC_START_TS ))

if [ "$CDP_READY" != true ]; then
    journalctl -u browser-vnc-chromium --no-pager -n 20 2>/dev/null || true
    fail "Chrome CDP 9222 未在 90s 内就绪 (${SVC_ELAPSED}s)"
fi

CDP_9222_PID=$(ss -tlnp 2>/dev/null | grep ':9222 ' | grep -oP 'pid=\K[0-9]+' | head -1)
[ -z "$CDP_9222_PID" ] && fail "CDP 9222 无法获取 PID"
log "  ✓ CDP 就绪 PID=$CDP_9222_PID (${SVC_ELAPSED}s)"
CDP_9222_CMD=$(cat /proc/$CDP_9222_PID/cmdline 2>/dev/null | tr '\0' ' ')
echo "$CDP_9222_CMD" | grep -q '/opt/browser-vnc/chrome-data' || fail "9222 非 Browser VNC Chrome"
echo "$CDP_9222_CMD" | grep -q '\-\-headless' && fail "9222 是 headless 浏览器"

ALL_RUNNING=true
for svc in $BVNC_SVCS; do
    if systemctl is-active --quiet "$svc" 2>/dev/null; then
        log "  $svc: active"
    else
        log "  $svc: $(systemctl is-active "$svc" 2>/dev/null || echo unknown)"
        ALL_RUNNING=false
    fi
done

log ""
INSTALL_TOTAL_ELAPSED=$(( $(date +%s) - INSTALL_START_TS ))
log "=== 安装完成: $(date '+%Y-%m-%d %H:%M:%S') (总耗时: ${INSTALL_TOTAL_ELAPSED}s) ==="
log "--- 安装报告 ---"
log "  Chrome:     $(google-chrome-stable --version 2>/dev/null | awk '{print $NF}' || echo 'N/A')"
log "  Xvnc:       $(Xvnc -version 2>/tmp/xvnc_ver_$$.txt; head -1 /tmp/xvnc_ver_$$.txt 2>/dev/null || echo 'N/A'; rm -f /tmp/xvnc_ver_$$.txt)"
log "  noVNC:      $([ -d /opt/browser-vnc/noVNC ] && echo 'v1.5.0' || echo 'N/A')"
log "  websockify:  $([ -d /opt/browser-vnc/websockify ] && echo 'v0.12.0' || echo 'N/A')"
log "  Kernel:     $(uname -r)"
log "  总耗时:     ${INSTALL_TOTAL_ELAPSED}s"
log "----------------"

if [ "$ALL_RUNNING" = true ]; then
    OPENCLAW_CFG="$HOME/.openclaw/openclaw.json"
    if [ -f "$OPENCLAW_CFG" ] && command -v jq >/dev/null 2>&1; then
        cp "$OPENCLAW_CFG" "${OPENCLAW_CFG}.bak.$(date +%s)"
        jq '.browser={"enabled":true,"executablePath":"/usr/bin/google-chrome-stable","noSandbox":true,"defaultProfile":"user","ssrfPolicy":{"dangerouslyAllowPrivateNetwork":true},"profiles":{"user":{"cdpUrl":"http://localhost:9222","driver":"existing-session","attachOnly":true,"color":"#4285F4"}}}' "$OPENCLAW_CFG" > "${OPENCLAW_CFG}.tmp" && mv "${OPENCLAW_CFG}.tmp" "$OPENCLAW_CFG"

        BROWSER_SESSION_DIR="$HOME/.openclaw/browser-existing-session"
        mkdir -p "$BROWSER_SESSION_DIR"
        if curl -s --max-time 2 "http://localhost:9222/json/version" >/dev/null 2>&1; then
            BROWSER_ID=$(curl -s "http://localhost:9222/json/version" | jq -r '.webSocketDebuggerUrl' 2>/dev/null | sed 's|.*/browser/||')
            [ -n "$BROWSER_ID" ] && [ "$BROWSER_ID" != "null" ] && \
                echo -e "9222\n/devtools/browser/${BROWSER_ID}" > "${BROWSER_SESSION_DIR}/DevToolsActivePort"
        fi

        # 异步重启 Gateway，不阻塞安装完成（Gateway 初始化可能需要较长时间）
        (
            GATEWAY_RESTARTED=false
            runuser -l root -c 'XDG_RUNTIME_DIR=/run/user/0 systemctl --user restart openclaw-gateway' 2>/dev/null && GATEWAY_RESTARTED=true
            [ "$GATEWAY_RESTARTED" = false ] && XDG_RUNTIME_DIR=/run/user/0 systemctl --user restart openclaw-gateway 2>/dev/null && GATEWAY_RESTARTED=true

            if [ "$GATEWAY_RESTARTED" = true ]; then
                GW_BROWSER_READY=false
                for i in $(seq 1 90); do
                    journalctl --user -u openclaw-gateway --since "2 min ago" --no-pager 2>/dev/null | grep -q 'Browser control service ready' && { GW_BROWSER_READY=true; break; }
                    ss -tlnp 2>/dev/null | grep -q 'openclaw.*LISTEN' && { GW_BROWSER_READY=true; break; }
                    sleep 1
                done
                echo "  Gateway restart: $([ "$GW_BROWSER_READY" = true ] && echo 'ready' || echo 'pending')" >> "$LOG_FILE"
            fi
        ) &
        log "  ✓ Gateway: restarting (async)"
    fi

    WORKSPACE_DIR="$HOME/.openclaw/workspace"
    VNC_MARKER="<!-- VNC_CLOUD_BROWSER_INJECTED -->"
    if [ -d "$WORKSPACE_DIR" ]; then
        SOUL_FILE="$WORKSPACE_DIR/SOUL.md"
        [ -f "$SOUL_FILE" ] && ! grep -q "$VNC_MARKER" "$SOUL_FILE" && \
            printf '\n%s\n## Cloud Browser (VNC Mode)\n- Actions visible in real-time. Be deliberate.\n- Stop all browser ops when user takes control.\n' "$VNC_MARKER" >> "$SOUL_FILE"
        AGENTS_FILE="$WORKSPACE_DIR/AGENTS.md"
        [ -f "$AGENTS_FILE" ] && ! grep -q "$VNC_MARKER" "$AGENTS_FILE" && \
            printf '\n%s\n**Cloud Browser (VNC):** AI actions visible via noVNC. CDP 9222 controls visible Chrome.\n' "$VNC_MARKER" >> "$AGENTS_FILE"
        SKILL_FILE="$WORKSPACE_DIR/skills/browser-use/SKILL.md"
        [ -f "$SKILL_FILE" ] && ! grep -q "$VNC_MARKER" "$SKILL_FILE" && \
            printf '\n%s\n## VNC Mode\n- Detect: `systemctl is-active --quiet browser-vnc-chromium`\n- User is watching. No headless fallback.\n- Restart: `systemctl restart browser-vnc-chromium`\n- On timeout: RETRY 3x before giving up. Verify CDP: `curl -s http://localhost:9222/json/version`\n' "$VNC_MARKER" >> "$SKILL_FILE"
    fi

    echo '{"installed": true}'
    echo '{"installed": true}' >&3
else
    FAILED_SVCS=""
    for svc in $BVNC_SVCS; do
        systemctl is-active --quiet "$svc" 2>/dev/null || FAILED_SVCS="${FAILED_SVCS}${svc} "
    done
    fail "服务未启动: ${FAILED_SVCS}"
fi
