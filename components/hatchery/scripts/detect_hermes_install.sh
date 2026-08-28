#!/bin/bash
# detect_hermes_install.sh — 检测 hermes 装在哪个用户下
#
# 上层 agent_checker.go 只读 runtime_user / runtime_home 两个字段，
# 故脚本只做这一件事：
#   1) 当前用户下有 hermes → 当前用户
#   2) 否则扫 /home/* 和 /root，找到第一个装了 hermes 的用户
#   3) 都没找到 → 返回空值（探测失败，由上层 agent_checker 决策）
#
# 为避免机器刚启动时外部命令 hang：不调用 hermes/harness/systemctl，只做文件系统检查。
# 脚本始终 exit 0，失败语义体现在 runtime_user 的取值上。
set -u

# ========== 日志系统初始化 ==========
# 重要：调用方（Go 后端 detectAndSaveRuntimeUser / admin_instances 探测接口）
# 会对 stdout 做 json.Unmarshal，因此 stdout 必须保持为纯 JSON，不能混入任何日志文本。
# 做法：用 fd 3 保存原始 stdout 供最后输出 JSON 使用；
#       同时把脚本 stdout/stderr 全部重定向到日志文件，所有 echo 日志都只落盘，不进 stdout。
LOG_DIR="/var/log/clawpro"
if [ ! -d "$LOG_DIR" ]; then
    mkdir -p "$LOG_DIR" 2>/dev/null || true
fi
SCRIPT_NAME="detect_hermes_install"
LOG_FILE="$LOG_DIR/${SCRIPT_NAME}.log"

# 用 fd 3 复制一份原始 stdout（供最后输出 JSON）
exec 3>&1
# 将脚本的 stdout/stderr 全部重定向到日志文件（追加）
# 若日志目录/文件不可写，则退回到 /dev/null，避免 exec 失败导致脚本整体崩溃
if { : >> "$LOG_FILE"; } 2>/dev/null; then
    exec >> "$LOG_FILE" 2>&1
else
    exec >/dev/null 2>&1
fi

# 兜底：若脚本在未输出 JSON 前异常退出，仍向 fd 3 输出一个最小可解析的 JSON，
# 避免 Go 侧 json.Unmarshal 收到空 stdout 报 "unexpected end of JSON input"。
_JSON_EMITTED=0
_emit_fallback_json() {
  [ "$_JSON_EMITTED" = "1" ] && return 0
  local u="${RUNTIME_USER:-}"
  local h="${RUNTIME_HOME:-}"
  printf '{"runtime_user":"%s","runtime_home":"%s"}\n' "$u" "$h" >&3
  _JSON_EMITTED=1
}
trap '_emit_fallback_json' EXIT

echo ""
echo "========== 日志开始: $(date '+%Y-%m-%d %H:%M:%S') =========="
echo "=== detect_hermes_install 开始 ==="

# 判断给定 HOME 下是否已安装 hermes
# 注意：不能仅凭 .hermes 目录存在就判定，因为其他 hermes 脚本（set_model、get_version_info 等）
# 在日志初始化时会无条件 mkdir -p ~/.hermes/logs，即使 hermes 未实际安装。
# 需要检查真正的安装产物。
has_hermes() {
    local h="$1"
    [ -x "$h/.local/bin/hermes" ] && [ -f "$h/.hermes/config.yaml" ] && return 0
    return 1
}

# 判断给定用户是否真实存在于 /etc/passwd。
# 背景：爱马仕镜像升级过程中，老版本可能残留 /home/agentuser 目录（内含 .local/bin/hermes），
# 但新镜像（v0.0.12+）已经把 agentuser 用户从 /etc/passwd 中移除了。
# 若 has_hermes() 只按目录判定，就会把不存在的 agentuser 当作运行用户回给上层，
# 导致后续 restore 脚本的 chown/runuser 全部挂掉（"runtime user 'agentuser' 不存在于 /etc/passwd"）。
# 这个函数用于给 has_hermes 的命中结果加一道用户存在性校验。
user_exists_in_passwd() {
    local u="$1"
    [ -n "$u" ] || return 1
    awk -F: -v u="$u" '$1==u{found=1; exit} END{exit !found}' /etc/passwd 2>/dev/null
}

CURRENT_USER="$(id -un 2>/dev/null || echo 'unknown')"
CURRENT_HOME="${HOME:-/root}"

RUNTIME_USER=""
RUNTIME_HOME=""
DETECT_SOURCE=""   # current_user / other_user / root / fallback_none

echo "CURRENT_USER: $CURRENT_USER"
echo "CURRENT_HOME: $CURRENT_HOME"

# 1) 优先：当前用户自己装了
echo ""
echo ">>> [步骤 1/4] 检查当前用户 HOME 是否已安装 hermes: $CURRENT_HOME"
if has_hermes "$CURRENT_HOME"; then
    RUNTIME_USER="$CURRENT_USER"
    RUNTIME_HOME="$CURRENT_HOME"
    DETECT_SOURCE="current_user"
    echo "✓ 命中：当前用户 $CURRENT_USER 下已安装 hermes"
else
    echo "× 未命中：当前用户 $CURRENT_USER 下未发现 hermes"
fi

# 2) 否则：扫其他用户
echo ""
echo ">>> [步骤 2/4] 扫描 /home/* 其他用户"
if [ -z "$RUNTIME_USER" ]; then
    shopt -s nullglob 2>/dev/null || true
    SCANNED_COUNT=0
    for user_home in /home/*; do
        [ -d "$user_home" ] || continue
        user_name="$(basename "$user_home")"
        [ "$user_name" = "$CURRENT_USER" ] && continue
        SCANNED_COUNT=$((SCANNED_COUNT + 1))
        echo "  - 探测 $user_home ..."
        if has_hermes "$user_home"; then
            # 关键校验：目录里有 hermes 产物 ≠ 该用户在系统中真实存在。
            # 爱马仕镜像升级路径上，老版本可能留下 /home/agentuser 目录（含 .local/bin/hermes），
            # 但新镜像（v0.0.12+）已从 /etc/passwd 移除该用户。若不校验就把 agentuser 回给上层，
            # 后续 restore 会在 chown/runuser 上挂掉。
            if ! user_exists_in_passwd "$user_name"; then
                echo "  ⚠ $user_home 有 hermes 产物但用户 '$user_name' 不在 /etc/passwd 中（老镜像残留目录），跳过"
                continue
            fi
            RUNTIME_USER="$user_name"
            RUNTIME_HOME="$user_home"
            DETECT_SOURCE="other_user"
            echo "✓ 命中：用户 $user_name ($user_home) 下已安装 hermes"
            break
        fi
    done
    shopt -u nullglob 2>/dev/null || true
    if [ -z "$RUNTIME_USER" ]; then
        echo "× 未命中：/home 下共探测 $SCANNED_COUNT 个用户，均未发现 hermes"
    fi
else
    echo "（已在上一步命中，跳过）"
fi

# 3) 再检查 root
echo ""
echo ">>> [步骤 3/4] 检查 /root"
if [ -z "$RUNTIME_USER" ] && [ "$CURRENT_USER" != "root" ]; then
    if has_hermes "/root"; then
        RUNTIME_USER="root"
        RUNTIME_HOME="/root"
        DETECT_SOURCE="root"
        echo "✓ 命中：/root 下已安装 hermes"
    else
        echo "× 未命中：/root 下未发现 hermes"
    fi
else
    if [ -n "$RUNTIME_USER" ]; then
        echo "（已在上一步命中，跳过）"
    else
        echo "（当前用户即 root，跳过）"
    fi
fi

# 4) 所有路径均未找到：不兜底，返回空值表示探测失败，由上层决策
echo ""
echo ">>> [步骤 4/4] 最终判定"
if [ -z "$RUNTIME_USER" ]; then
    DETECT_SOURCE="not_found"
    echo "✗ 所有路径均未发现 hermes，返回空值（探测失败，由上层处理）"
else
    echo "✓ 已命中，来源: $DETECT_SOURCE"
fi

echo ""
echo "=== 最终结果 ==="
echo "RUNTIME_USER:   $RUNTIME_USER"
echo "RUNTIME_HOME:   $RUNTIME_HOME"
echo "DETECT_SOURCE:  $DETECT_SOURCE"

# 输出纯 JSON 到 fd 3（原始 stdout），Go 侧将读取这段输出做 json.Unmarshal。
RESULT_JSON=$(cat <<EOJSON
{
  "runtime_user": "$RUNTIME_USER",
  "runtime_home": "$RUNTIME_HOME"
}
EOJSON
)
echo ""
echo "=== 输出到 stdout 的 JSON ==="
echo "$RESULT_JSON"
printf '%s\n' "$RESULT_JSON" >&3
_JSON_EMITTED=1

echo ""
echo "========== 日志结束: $(date '+%Y-%m-%d %H:%M:%S') =========="

# 正常结束：撤销 EXIT trap，避免 trap 兜底再次输出 JSON
trap - EXIT
exit 0
