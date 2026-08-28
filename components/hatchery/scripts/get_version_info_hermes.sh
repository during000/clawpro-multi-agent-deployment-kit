#!/bin/bash
# get_version_info_hermes.sh
# 获取 Hermes Agent 主程序版本 + 已安装 skills 列表，输出 JSON。
# 输出契约（与 openclaw 版一致）：
#   {"agent_version":"0.9.0","agent_type":"hermes","plugins":{"<skill>":"<version>",...}}
#
# Hermes 特性：
#   - 主程序命令：`hermes`（Python 启动），版本从 `hermes --version` 获取
#   - 管理工具：`harness` CLI，skills 管理与 gateway 交互都走 harness
#   - 可选路径：通过 `harness skills list --output json` 获取 skill 清单（参考
#     scripts/check_hermes_ready.sh / scripts/add_skill_hermes.sh 的使用模式）
#
# 所有诊断信息写日志文件，stdout 保持干净（最后一行是 JSON）。

set -uo pipefail
export PATH="$HOME/.local/bin:$HOME/.cargo/bin:$HOME/.npm-global/bin:/usr/local/bin:/usr/bin:/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u 2>/dev/null || echo 0)

# %INCLUDE% lib_acli_compat.sh

# ========== 日志系统初始化 ==========
LOG_DIR="${HOME}/.hermes/logs"
mkdir -p "$LOG_DIR" 2>/dev/null || true
SCRIPT_NAME="get_version_info_hermes"
LOG_FILE="$LOG_DIR/${SCRIPT_NAME}.log"
exec 2>>"$LOG_FILE"
echo "" >>"$LOG_FILE"
echo "========== 日志开始: $(date '+%Y-%m-%d %H:%M:%S') ==========" >>"$LOG_FILE"

# ===== 脚本级探测一次，后续复用 =====
_acli_mode="$(ensure_acli_light 2>>"$LOG_FILE")"

HERMES_VERSION=""

# 优先 acli backend version
# 输出：{"success":true,"data":{"version":"1.2.3","backend":"hermes"}}
if [ "$_acli_mode" = "acli" ] && command -v jq >/dev/null 2>&1; then
    _acli_ver_raw="$(acli backend version 2>>"$LOG_FILE" || true)"
    if [ -n "$_acli_ver_raw" ]; then
        HERMES_VERSION="$(printf '%s' "$_acli_ver_raw" | jq -r '.data.version // empty' 2>>"$LOG_FILE" || true)"
        if [ -n "$HERMES_VERSION" ]; then
            echo "✓ 从 acli backend version 读取: $HERMES_VERSION" >>"$LOG_FILE"
        fi
    fi
fi

# ========== 1. 查找 hermes 命令（多路径 fallback，风格与 set_hermes_ui.sh 一致） ==========
locate_hermes_bin() {
    if command -v hermes >/dev/null 2>&1; then
        command -v hermes
        return 0
    fi
    for p in "$HOME/.local/bin/hermes" "/usr/local/bin/hermes" "/root/.local/bin/hermes"; do
        [ -x "$p" ] && echo "$p" && return 0
    done
    for d in /home/*; do
        [ -x "$d/.local/bin/hermes" ] && echo "$d/.local/bin/hermes" && return 0
    done
    return 1
}

if [ -z "$HERMES_VERSION" ]; then
    HERMES_BIN="$(locate_hermes_bin || true)"
    echo ">>> [步骤 1/2] 获取 Hermes 版本 (HERMES_BIN=${HERMES_BIN:-<missing>})" >>"$LOG_FILE"

    if [ -n "$HERMES_BIN" ]; then
        raw=$("$HERMES_BIN" --version 2>>"$LOG_FILE" || true)
        [ -z "$raw" ] && raw=$("$HERMES_BIN" version 2>>"$LOG_FILE" || true)
        HERMES_VERSION=$(echo "$raw" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1 || true)
        if [ -n "$HERMES_VERSION" ]; then
            echo "✓ 从 hermes --version 读取: $HERMES_VERSION (raw=$raw)" >>"$LOG_FILE"
        fi
    fi
fi

# fallback：有些打包方式下 hermes 版本记录在 $HOME/.hermes/VERSION 或类似文件
if [ -z "$HERMES_VERSION" ]; then
    for vfile in "$HOME/.hermes/VERSION" "$HOME/.hermes/version" "$HOME/.hermes/meta/version"; do
        if [ -f "$vfile" ]; then
            v=$(cat "$vfile" 2>>"$LOG_FILE" | tr -d '[:space:]')
            v=$(echo "$v" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1 || true)
            if [ -n "$v" ]; then
                HERMES_VERSION="$v"
                echo "✓ 从 $vfile 读取: $HERMES_VERSION" >>"$LOG_FILE"
                break
            fi
        fi
    done
fi

if [ -z "$HERMES_VERSION" ]; then
    echo "✗ 无法获取 Hermes 版本" >>"$LOG_FILE"
fi

# ========== 2. 已安装 skills ==========
echo "" >>"$LOG_FILE"
echo ">>> [步骤 2/2] 读取已安装 skills" >>"$LOG_FILE"

PLUGINS_JSON="{}"

# 注意：acli skills list 依赖 skillhub 注册表，hatchery 安装的技能不走 skillhub 注册，
# 所以这里不用 acli skills list，直接走 harness 或目录扫描。
if command -v harness >/dev/null 2>&1 && command -v jq >/dev/null 2>&1; then
    raw_json=""
    for args in "--output json" "--format json" "-o json" ""; do
        # shellcheck disable=SC2086
        raw_json=$(harness skills list $args 2>>"$LOG_FILE" || true)
        if [ -n "$raw_json" ] && echo "$raw_json" | jq empty >/dev/null 2>&1; then
            break
        fi
        raw_json=""
    done

    if [ -n "$raw_json" ]; then
        normalized=$(echo "$raw_json" | jq -c '
          (.skills // .) as $list
          | ($list // [])
          | map(select((.enabled // true) == true))
          | map({key: (.name // .slug // ""), value: (.version // "installed")})
          | from_entries
        ' 2>>"$LOG_FILE" || true)
        if [ -n "$normalized" ] && [ "$normalized" != "null" ]; then
            PLUGINS_JSON="$normalized"
            echo "✓ 从 harness skills list (JSON) 解析成功" >>"$LOG_FILE"
        else
            echo "  ⚠ harness skills list JSON 解析失败，回退空 map" >>"$LOG_FILE"
        fi
    else
        echo "  ⚠ harness skills list 无 JSON 输出模式，跳过插件解析" >>"$LOG_FILE"
    fi
fi

# 如果 harness 也拿不到，用目录扫描兜底
if [ "$PLUGINS_JSON" = "{}" ] && command -v jq >/dev/null 2>&1; then
    _scan_dir="$HOME/.hermes/skills"
    if [ -d "$_scan_dir" ]; then
        _scanned="$(find "$_scan_dir" -mindepth 1 -maxdepth 1 -type d ! -name '.*' -printf '%f\n' 2>/dev/null | sort)"
        if [ -n "$_scanned" ]; then
            PLUGINS_JSON="$(printf '%s' "$_scanned" | jq -Rc '[., inputs] | map({key: ., value: "installed"}) | from_entries' 2>>"$LOG_FILE" || echo "{}")"
            echo "✓ 从目录扫描获取 skills: $(echo "$PLUGINS_JSON" | jq 'length' 2>/dev/null)" >>"$LOG_FILE"
        fi
    fi
fi

# ========== 3. 输出 JSON ==========
RESULT=$(printf '{"agent_version":"%s","agent_type":"hermes","plugins":%s}' \
  "${HERMES_VERSION}" \
  "${PLUGINS_JSON}")
echo "$RESULT"
