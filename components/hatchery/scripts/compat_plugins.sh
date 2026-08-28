#!/bin/bash
# =============================================================================
# OpenClaw 插件兼容修复脚本
# -----------------------------------------------------------------------------
# 各版本升级后需要执行的插件兼容修复逻辑，每个修复项独立为一个函数。
# 新增修复项：追加函数 + 在 main() 中调用即可。
#
# 设计原则：
#   - 幂等：重复执行结果一致
#   - 容错：单项失败不影响其他项，整体以 0 退出（不阻断升级主流程）
# =============================================================================

set -u  # 不开 -e：单项失败不终止全局

export PATH="$HOME/.npm-global/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u)

# ========== 日志系统初始化 ==========
LOG_DIR="${HOME}/.openclaw/logs"
if [ ! -d "$LOG_DIR" ]; then
    mkdir -p "$LOG_DIR"
fi
SCRIPT_NAME="compat_plugins"
LOG_FILE="$LOG_DIR/${SCRIPT_NAME}.log"
exec > >(tee -a "$LOG_FILE") 2>&1
echo ""
echo "========== 日志开始: $(date '+%Y-%m-%d %H:%M:%S') =========="

LOG_PREFIX="[compat-plugins]"

# ========== 公共：加载 openclaw 运行环境 ==========
load_env() {
    export PNPM_HOME="${PNPM_HOME:-$HOME/.local/share/pnpm}"
    export PATH="$PNPM_HOME:$HOME/.npm-global/bin:$HOME/.local/bin:$PATH"
    export NVM_DIR="$HOME/.nvm"
    [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
    hash -r || true
}

# ========== 公共：获取 openclaw 版本号 ==========
# 输出形如 "2026.5.7"；获取失败输出空字符串。兼容 `--version` / `version` 两种 CLI。
get_openclaw_version() {
    load_env
    { openclaw --version; openclaw version; } 2>&1 \
        | grep -oP '[0-9]{4}\.[0-9]+\.[0-9]+' | head -1 || true
}

# ========== 公共：判断当前 openclaw 版本是否 >= 指定版本 ==========
# 用法：openclaw_version_ge "2026.5.28"，按三段语义版本号比较。
# 返回 0 表示满足；返回 1 表示不满足或无法获取当前版本。
openclaw_version_ge() {
    local target="$1"
    if [ -z "$target" ]; then
        return 1
    fi

    local current
    current="$(get_openclaw_version)"
    if [ -z "$current" ]; then
        return 1
    fi

    local cur_major cur_minor cur_patch tgt_major tgt_minor tgt_patch
    cur_major="$(echo "$current" | awk -F. '{print $1+0}')"
    cur_minor="$(echo "$current" | awk -F. '{print $2+0}')"
    cur_patch="$(echo "$current" | awk -F. '{print $3+0}')"
    tgt_major="$(echo "$target"  | awk -F. '{print $1+0}')"
    tgt_minor="$(echo "$target"  | awk -F. '{print $2+0}')"
    tgt_patch="$(echo "$target"  | awk -F. '{print $3+0}')"

    if [ "$cur_major" -gt "$tgt_major" ] 2>/dev/null; then
        return 0
    elif [ "$cur_major" -lt "$tgt_major" ] 2>/dev/null; then
        return 1
    fi
    if [ "$cur_minor" -gt "$tgt_minor" ] 2>/dev/null; then
        return 0
    elif [ "$cur_minor" -lt "$tgt_minor" ] 2>/dev/null; then
        return 1
    fi
    if [ "$cur_patch" -ge "$tgt_patch" ] 2>/dev/null; then
        return 0
    fi
    return 1
}

# =============================================================================
# 修复项：wecom 老名清理 + channel 配置格式迁移（openclaw >= 2026.5.0 生效）
#   - 老名 "wecom" 已废弃，统一用新名 "wecom-openclaw-plugin"（< 2026.7.01）：
#     >= 2026.7.01: @sunnoy/wecom 替换 @wecom/wecom-openclaw-plugin，plugin_id 回归 "wecom"
#     移除 plugins.entries.wecom 与 plugins.allow 中的 "wecom"
#   - channels.wecom 旧格式（botId/secret 在 bot 内）→ 新格式（提升到顶层）
# =============================================================================
fix_wecom_legacy_names() {
    local cfg="$HOME/.openclaw/openclaw.json"
    echo ""
    echo "${LOG_PREFIX} [fix_wecom_legacy_names] 开始执行 wecom 老名清理"

    if [ ! -f "$cfg" ] || ! command -v jq >/dev/null 2>&1; then
        echo "${LOG_PREFIX} [fix_wecom_legacy_names] 配置文件不存在或 jq 未安装，跳过"
        return
    fi

    local openclaw_version
    openclaw_version="$(get_openclaw_version)"

    # --- >= 2026.7.01: @sunnoy/wecom 优先，plugin_id 回归 "wecom" ---
    if [ -n "$openclaw_version" ] && openclaw_version_ge "2026.7.1"; then
        if compgen -G "$HOME/.openclaw/npm/projects/sunnoy-wecom-*" >/dev/null 2>&1; then
            echo "${LOG_PREFIX} [fix_wecom_legacy_names] openclaw ${openclaw_version}，检测到 @sunnoy/wecom"
            # 清理 wecom-openclaw-plugin 残留（@wecom/wecom-openclaw-plugin 已废弃）
            if jq -e '.plugins.entries["wecom-openclaw-plugin"]' "$cfg" >/dev/null 2>&1; then
                jq 'del(.plugins.entries["wecom-openclaw-plugin"])' "$cfg" > "$cfg.tmp" \
                    && mv "$cfg.tmp" "$cfg" \
                    && echo "${LOG_PREFIX} [fix_wecom_legacy_names] ✓ 已移除 plugins.entries[\"wecom-openclaw-plugin\"]"
            fi
            if jq -e '(.plugins.allow // []) | index("wecom-openclaw-plugin")' "$cfg" >/dev/null 2>&1; then
                jq '.plugins.allow = ((.plugins.allow // []) | map(select(. != "wecom-openclaw-plugin")))' \
                    "$cfg" > "$cfg.tmp" \
                    && mv "$cfg.tmp" "$cfg" \
                    && echo "${LOG_PREFIX} [fix_wecom_legacy_names] ✓ plugins.allow 已移除 \"wecom-openclaw-plugin\""
            fi
            # 确保 wecom 在 plugins.allow 中（@sunnoy/wecom 的 plugin_id 是 "wecom"）
            jq '.plugins.allow = ((.plugins.allow // []) | if index("wecom") then . else . + ["wecom"] end)' \
                "$cfg" > "$cfg.tmp" \
                && mv "$cfg.tmp" "$cfg" \
                && echo "${LOG_PREFIX} [fix_wecom_legacy_names] ✓ plugins.allow 已确保包含 \"wecom\""

            # channels.wecom → @sunnoy/wecom 扁平格式迁移
            # 将嵌套 bot 对象的字段全部提升到 channels.wecom 顶层：
            #   bot.botId/secret            → 顶层 botId/secret
            #   bot.dm.policy               → 顶层 dmPolicy
            #   bot.welcomeText             → 顶层 welcomeMessage
            #   bot.streamPlaceholderContent → 顶层 sendThinkingMessage=true
            #   完全删除 bot 对象（@sunnoy/wecom 不使用 bot 嵌套）
            if jq -e '.channels.wecom' "$cfg" >/dev/null 2>&1; then
                if jq -e '.channels.wecom.bot' "$cfg" >/dev/null 2>&1; then
                    echo "${LOG_PREFIX} [fix_wecom_legacy_names] channels.wecom 嵌套格式，迁移为 @sunnoy/wecom 扁平格式"
                    jq '.channels.wecom.botId = (.channels.wecom.botId // .channels.wecom.bot.botId // null)
                        | .channels.wecom.secret = (.channels.wecom.secret // .channels.wecom.bot.secret // null)
                        | .channels.wecom.dmPolicy = (.channels.wecom.dmPolicy // .channels.wecom.bot.dm.policy // "open")
                        | .channels.wecom.welcomeMessage = (.channels.wecom.welcomeMessage // .channels.wecom.bot.welcomeText // "你好！我是 AI 助手")
                        | .channels.wecom.sendThinkingMessage = (.channels.wecom.sendThinkingMessage // true)
                        | del(.channels.wecom.bot)' \
                        "$cfg" > "$cfg.tmp" \
                        && mv "$cfg.tmp" "$cfg" \
                        && echo "${LOG_PREFIX} [fix_wecom_legacy_names] ✓ 已迁移为 @sunnoy/wecom 扁平格式"
                fi
                openclaw plugins enable wecom || true
                echo "${LOG_PREFIX} [fix_wecom_legacy_names] ✓ wecom（@sunnoy/wecom）已启用"
            fi

            echo "${LOG_PREFIX} [fix_wecom_legacy_names] 完成（@sunnoy/wecom 模式）"
            return
        fi
    fi

    # --- < 2026.7.01: 原逻辑（wecom-openclaw-plugin 是官方新版）---

    # 版本门槛：>= 2026.5.0 才清理老名
    local need_legacy_cleanup=false
    if [ -z "$openclaw_version" ]; then
        echo "${LOG_PREFIX} [fix_wecom_legacy_names] 无法获取 openclaw 版本，跳过老名清理"
    elif openclaw_version_ge "2026.5.0"; then
        need_legacy_cleanup=true
        echo "${LOG_PREFIX} [fix_wecom_legacy_names] openclaw ${openclaw_version} >= 2026.5.0，清理 wecom 老名残留"
    else
        echo "${LOG_PREFIX} [fix_wecom_legacy_names] openclaw ${openclaw_version} < 2026.5.0，跳过老名清理"
    fi

    if $need_legacy_cleanup; then
        if jq -e '.plugins.entries.wecom' "$cfg" >/dev/null 2>&1; then
            jq 'del(.plugins.entries.wecom)' "$cfg" > "$cfg.tmp" \
                && mv "$cfg.tmp" "$cfg" \
                && echo "${LOG_PREFIX} [fix_wecom_legacy_names] ✓ 已移除 plugins.entries.wecom"
        fi
        if jq -e '(.plugins.allow // []) | index("wecom")' "$cfg" >/dev/null 2>&1; then
            jq '.plugins.allow = ((.plugins.allow // []) | map(select(. != "wecom")))' \
                "$cfg" > "$cfg.tmp" \
                && mv "$cfg.tmp" "$cfg" \
                && echo "${LOG_PREFIX} [fix_wecom_legacy_names] ✓ plugins.allow 已移除 \"wecom\""
        fi
    fi

    # channels.wecom 旧格式（botId/secret 在 bot 内）→ 新格式（提升到顶层）
    if jq -e '.channels.wecom' "$cfg" >/dev/null 2>&1; then
        if jq -e '.channels.wecom.bot.botId != null and (.channels.wecom.botId == null)' "$cfg" >/dev/null 2>&1; then
            echo "${LOG_PREFIX} [fix_wecom_legacy_names] channels.wecom 旧格式，迁移 botId/secret 到顶层"
            jq '.channels.wecom.botId = .channels.wecom.bot.botId
                | .channels.wecom.secret = .channels.wecom.bot.secret
                | del(.channels.wecom.bot.botId)
                | del(.channels.wecom.bot.secret)' \
                "$cfg" > "$cfg.tmp" \
                && mv "$cfg.tmp" "$cfg" \
                && echo "${LOG_PREFIX} [fix_wecom_legacy_names] ✓ botId/secret 已提升到 channels.wecom 顶层"
        fi

        # 用户已配置 wecom 通道，确保新插件启用
        openclaw plugins enable wecom-openclaw-plugin || true
        echo "${LOG_PREFIX} [fix_wecom_legacy_names] ✓ wecom-openclaw-plugin 已启用"
    fi

    echo "${LOG_PREFIX} [fix_wecom_legacy_names] 完成"
}

# =============================================================================
# 修复项：ddingtalk → @dingtalk-real-ai/dingtalk-connector（新版镜像生效）
#   判据：磁盘上是否存在新版插件目录（plugins.installs 可能从旧实例迁移过来，
#         不反映当前镜像真实安装状态，与 set_channel.sh 保持一致）。
# =============================================================================
fix_ddingtalk_legacy_names() {
    local cfg="$HOME/.openclaw/openclaw.json"
    # 新版插件可能存在于两种路径（与 set_channel.sh / del_channel.sh 保持一致）：
    #   1) node_modules/@dingtalk-real-ai/dingtalk-connector            —— npm 全局 scoped 包形态
    #   2) projects/dingtalk-real-ai-dingtalk-connector-<hash>/         —— 新版 npm 子工程隔离安装形态（目录名带哈希后缀，需用 glob 匹配）
    local new_plugin_dir="$HOME/.openclaw/npm/node_modules/@dingtalk-real-ai/dingtalk-connector"
    local new_plugin_glob="$HOME/.openclaw/npm/projects/dingtalk-real-ai-dingtalk-connector-*"
    echo ""
    echo "${LOG_PREFIX} [fix_ddingtalk_legacy_names] 开始执行 ddingtalk 老名迁移"

    if [ ! -f "$cfg" ] || ! command -v jq >/dev/null 2>&1; then
        echo "${LOG_PREFIX} [fix_ddingtalk_legacy_names] 配置文件不存在或 jq 未安装，跳过"
        return
    fi

    # 新版插件目录探测：优先 node_modules 老路径；未命中再 glob 匹配 projects/ 新路径
    local matched_dir=""
    if [ -d "$new_plugin_dir" ]; then
        matched_dir="$new_plugin_dir"
    else
        # shellcheck disable=SC2086
        for d in $new_plugin_glob; do
            if [ -d "$d" ]; then
                matched_dir="$d"
                break
            fi
        done
    fi
    if [ -z "$matched_dir" ]; then
        echo "${LOG_PREFIX} [fix_ddingtalk_legacy_names] 新版插件目录不存在，跳过"
        return
    fi
    echo "${LOG_PREFIX} [fix_ddingtalk_legacy_names] 命中新版插件目录：${matched_dir}"

    # plugins.entries：迁移 ddingtalk → dingtalk-connector（设为 enabled）
    if jq -e '.plugins.entries.ddingtalk' "$cfg" >/dev/null 2>&1; then
        jq '.plugins.entries["dingtalk-connector"] = ((.plugins.entries["dingtalk-connector"] // {}) + {"enabled": true})
            | del(.plugins.entries.ddingtalk)' \
            "$cfg" > "$cfg.tmp" \
            && mv "$cfg.tmp" "$cfg" \
            && echo "${LOG_PREFIX} [fix_ddingtalk_legacy_names] ✓ plugins.entries.ddingtalk → dingtalk-connector"
    fi

    # channels：迁移 ddingtalk → dingtalk-connector
    if jq -e '.channels.ddingtalk' "$cfg" >/dev/null 2>&1; then
        jq '.channels["dingtalk-connector"] = ((.channels["dingtalk-connector"] // {}) + .channels.ddingtalk)
            | del(.channels.ddingtalk)' \
            "$cfg" > "$cfg.tmp" \
            && mv "$cfg.tmp" "$cfg" \
            && echo "${LOG_PREFIX} [fix_ddingtalk_legacy_names] ✓ channels.ddingtalk → channels[\"dingtalk-connector\"]"
    fi

    # plugins.allow：移除 ddingtalk，加入 dingtalk-connector
    jq '.plugins.allow = ((.plugins.allow // []) | map(select(. != "ddingtalk")))
        | .plugins.allow = ((.plugins.allow // []) | if index("dingtalk-connector") then . else . + ["dingtalk-connector"] end)' \
        "$cfg" > "$cfg.tmp" \
        && mv "$cfg.tmp" "$cfg" \
        && echo "${LOG_PREFIX} [fix_ddingtalk_legacy_names] ✓ plugins.allow 已更新"

    # 按 channel 配置开关 dingtalk-connector：已配置 → enable；未配置 → disable
    if jq -e '.channels["dingtalk-connector"]' "$cfg" >/dev/null 2>&1; then
        openclaw plugins enable dingtalk-connector || true
        echo "${LOG_PREFIX} [fix_ddingtalk_legacy_names] ✓ dingtalk-connector 已启用（channel 已配置）"
    else
        if jq -e '.plugins.entries["dingtalk-connector"]' "$cfg" >/dev/null 2>&1; then
            jq '.plugins.entries["dingtalk-connector"].enabled = false' \
                "$cfg" > "$cfg.tmp" && mv "$cfg.tmp" "$cfg"
        fi
        openclaw plugins disable dingtalk-connector 2>/dev/null || true
        echo "${LOG_PREFIX} [fix_ddingtalk_legacy_names] ✓ dingtalk-connector 已禁用（channel 未配置）"
    fi

    echo "${LOG_PREFIX} [fix_ddingtalk_legacy_names] 完成"
}

# =============================================================================
# 修复项：feishu → openclaw-lark（openclaw >= 2026.5.28 + 新版插件目录存在时生效）
#   - 判据目录使用 glob 匹配（projects/larksuite-openclaw-lark-* 带版本/哈希后缀），
#     路径在 projects/ 而非 node_modules/，是飞书插件的特殊安装位置。
#   - channels.feishu 保持原名不动：Go 层 channel id 仍是 "feishu"，
#     openclaw-lark 通过 channels.feishu 读取配置。
#   - plugins.entries.feishu：直接删除老名；若原本 enabled=true 则同步启用
#     plugins.entries["openclaw-lark"]，否则不强制启用（尊重用户原开关状态）。
# =============================================================================
fix_lark_legacy_names() {
    local cfg="$HOME/.openclaw/openclaw.json"
    local new_plugin_glob="$HOME/.openclaw/npm/projects/larksuite-openclaw-lark-*"
    echo ""
    echo "${LOG_PREFIX} [fix_lark_legacy_names] 开始执行 feishu 老名迁移"

    if [ ! -f "$cfg" ] || ! command -v jq >/dev/null 2>&1; then
        echo "${LOG_PREFIX} [fix_lark_legacy_names] 配置文件不存在或 jq 未安装，跳过"
        return
    fi

    # 版本门槛：openclaw < 2026.5.28 时新插件不存在，强行删除 feishu 老名会导致飞书功能失效
    local openclaw_version
    openclaw_version="$(get_openclaw_version)"
    if [ -z "$openclaw_version" ]; then
        echo "${LOG_PREFIX} [fix_lark_legacy_names] 无法获取 openclaw 版本，跳过"
        return
    fi
    if ! openclaw_version_ge "2026.5.28"; then
        echo "${LOG_PREFIX} [fix_lark_legacy_names] openclaw ${openclaw_version} < 2026.5.28，跳过"
        return
    fi

    # 新版插件目录探测（glob 匹配，路径含通配符不能直接 [ -d ]）
    local matched_dir=""
    # shellcheck disable=SC2086
    for d in $new_plugin_glob; do
        if [ -d "$d" ]; then
            matched_dir="$d"
            break
        fi
    done
    if [ -z "$matched_dir" ]; then
        echo "${LOG_PREFIX} [fix_lark_legacy_names] 新版插件目录不存在，跳过"
        return
    fi
    echo "${LOG_PREFIX} [fix_lark_legacy_names] openclaw ${openclaw_version}，命中新版插件目录：${matched_dir}"

    # plugins.entries：必须先读 enabled 再 del，否则原状态丢失
    local feishu_enabled="false"
    if jq -e '.plugins.entries.feishu' "$cfg" >/dev/null 2>&1; then
        feishu_enabled=$(jq -r '.plugins.entries.feishu.enabled // false' "$cfg" 2>/dev/null)
        if [ "$feishu_enabled" = "true" ]; then
            jq 'del(.plugins.entries.feishu)
                | .plugins.entries["openclaw-lark"] = ((.plugins.entries["openclaw-lark"] // {}) + {"enabled": true})' \
                "$cfg" > "$cfg.tmp" \
                && mv "$cfg.tmp" "$cfg" \
                && echo "${LOG_PREFIX} [fix_lark_legacy_names] ✓ 删除 feishu，启用 openclaw-lark"
        else
            jq 'del(.plugins.entries.feishu)' "$cfg" > "$cfg.tmp" \
                && mv "$cfg.tmp" "$cfg" \
                && echo "${LOG_PREFIX} [fix_lark_legacy_names] ✓ 删除 feishu（原未启用，不启用 openclaw-lark）"
        fi
    fi

    # plugins.allow：移除 feishu，加入 openclaw-lark
    jq '.plugins.allow = ((.plugins.allow // []) | map(select(. != "feishu")))
        | .plugins.allow = ((.plugins.allow // []) | if index("openclaw-lark") then . else . + ["openclaw-lark"] end)' \
        "$cfg" > "$cfg.tmp" \
        && mv "$cfg.tmp" "$cfg" \
        && echo "${LOG_PREFIX} [fix_lark_legacy_names] ✓ plugins.allow 已更新"

    # 按 channel 配置开关 openclaw-lark：已配置 → enable；未配置 → disable
    # （restore_post_reinstall.sh 的 mappings 只能处理老名 feishu，新名 openclaw-lark
    #   的 enable/disable 状态在这里统一修正）
    if jq -e '.channels.feishu' "$cfg" >/dev/null 2>&1; then
        openclaw plugins enable openclaw-lark || true
        echo "${LOG_PREFIX} [fix_lark_legacy_names] ✓ openclaw-lark 已启用（channel 已配置）"
    else
        if jq -e '.plugins.entries["openclaw-lark"]' "$cfg" >/dev/null 2>&1; then
            jq '.plugins.entries["openclaw-lark"].enabled = false' \
                "$cfg" > "$cfg.tmp" && mv "$cfg.tmp" "$cfg"
        fi
        openclaw plugins disable openclaw-lark 2>/dev/null || true
        echo "${LOG_PREFIX} [fix_lark_legacy_names] ✓ openclaw-lark 已禁用（channel 未配置）"
    fi

    echo "${LOG_PREFIX} [fix_lark_legacy_names] 完成"
}

# =============================================================================
# 修复项：memory-tencentdb contextEngine slot 版本对齐
#   plugins.slots.contextEngine 的值随插件版本而异：
#     >= 0.3.6 → "memory-tencentdb"
#     <  0.3.6 → "openclaw-context-offload"
#   一键升级时 openclaw.json 从备份恢复，slot 值可能与新镜像插件版本不匹配，
#   导致 doctor 将其重置为 "legacy"，Pro 记忆功能失效。
#   仅在 offload.enabled=true（Pro 实例）时处理；版本读不到时跳过不动。
# =============================================================================
fix_memory_context_engine_slot() {
    local cfg="$HOME/.openclaw/openclaw.json"
    # 老路径：openclaw < 2026.5.28 时 memory-tencentdb 安装在 node_modules 下
    local pkg_json="$HOME/.openclaw/npm/node_modules/@tencentdb-agent-memory/memory-tencentdb/package.json"
    # 新路径：openclaw >= 2026.5.28 后改为 projects/ 下带哈希后缀的目录（glob 匹配）
    local pkg_json_glob="$HOME/.openclaw/npm/projects/tencentdb-agent-memory-memory-tencentdb-*/package.json"
    echo ""
    echo "${LOG_PREFIX} [fix_memory_context_engine_slot] 开始执行 memory-tencentdb contextEngine slot 修复"

    if [ ! -f "$cfg" ] || ! command -v jq >/dev/null 2>&1; then
        echo "${LOG_PREFIX} [fix_memory_context_engine_slot] 配置文件不存在或 jq 未安装，跳过"
        return
    fi

    # 前置条件：offload 已开启（Pro 模式才有 contextEngine slot）
    local offload_enabled
    offload_enabled=$(jq -r '.plugins.entries["memory-tencentdb"].config.offload.enabled // false' "$cfg" 2>/dev/null)
    if [ "$offload_enabled" != "true" ]; then
        echo "${LOG_PREFIX} [fix_memory_context_engine_slot] offload 未开启，跳过"
        return
    fi

    # 版本读取：优先 plugins.installs，回退磁盘 package.json
    # （plugins.installs 可能是从旧实例迁移过来的，磁盘文件最反映真实安装状态）
    # 磁盘 package.json 有两个候选位置：
    #   1) 老路径 node_modules/@tencentdb-agent-memory/memory-tencentdb/package.json
    #   2) 新路径 projects/tencentdb-agent-memory-memory-tencentdb-*/package.json（glob，>= 2026.5.28）
    # 老的没取到就尝试新的，两个都没取到则跳过。
    local plugin_version
    plugin_version=$(jq -r '.plugins.installs["memory-tencentdb"].version // ""' "$cfg" 2>/dev/null)
    if [ -z "$plugin_version" ] && [ -f "$pkg_json" ]; then
        plugin_version=$(jq -r '.version // ""' "$pkg_json" 2>/dev/null)
    fi
    if [ -z "$plugin_version" ]; then
        # 新路径下 package.json 是 pnpm project 容器，插件版本写在 dependencies 字段里，
        # 而不是顶层 .version；版本字符串可能带 ^ ~ 等前缀，需要剥掉再用。
        # shellcheck disable=SC2086
        for p in $pkg_json_glob; do
            if [ -f "$p" ]; then
                plugin_version=$(jq -r '.dependencies["@tencentdb-agent-memory/memory-tencentdb"] // ""' "$p" 2>/dev/null)
                # 剥离 ^ ~ >= 等版本范围前缀，只保留数字与点
                plugin_version=$(echo "$plugin_version" | sed -E 's/^[^0-9]*//')
                if [ -n "$plugin_version" ]; then
                    echo "${LOG_PREFIX} [fix_memory_context_engine_slot] 命中新版 package.json：$p"
                    break
                fi
            fi
        done
    fi
    if [ -z "$plugin_version" ]; then
        echo "${LOG_PREFIX} [fix_memory_context_engine_slot] 版本信息缺失，跳过"
        return
    fi
    echo "${LOG_PREFIX} [fix_memory_context_engine_slot] memory-tencentdb 版本: $plugin_version"

    # 版本比较（三段数字）：>= 0.3.6 用新名，否则用老名
    local expected_slot ver_major ver_minor ver_patch
    ver_major=$(echo "$plugin_version" | awk -F. '{print $1+0}')
    ver_minor=$(echo "$plugin_version" | awk -F. '{print $2+0}')
    ver_patch=$(echo "$plugin_version" | awk -F. '{print $3+0}')
    if [ "$ver_major" -gt 0 ] 2>/dev/null \
        || { [ "$ver_major" -eq 0 ] && [ "$ver_minor" -gt 3 ]; } 2>/dev/null \
        || { [ "$ver_major" -eq 0 ] && [ "$ver_minor" -eq 3 ] && [ "$ver_patch" -ge 6 ]; } 2>/dev/null; then
        expected_slot="memory-tencentdb"
    else
        expected_slot="openclaw-context-offload"
    fi

    local current_slot
    current_slot=$(jq -r '.plugins.slots.contextEngine // ""' "$cfg" 2>/dev/null)
    if [ "$current_slot" = "$expected_slot" ]; then
        echo "${LOG_PREFIX} [fix_memory_context_engine_slot] contextEngine=\"$current_slot\" 已正确"
        return
    fi

    jq --arg slot "$expected_slot" '
        .plugins.slots = (.plugins.slots // {}) | .plugins.slots.contextEngine = $slot
    ' "$cfg" > "$cfg.tmp" \
        && mv "$cfg.tmp" "$cfg" \
        && echo "${LOG_PREFIX} [fix_memory_context_engine_slot] ✓ contextEngine: \"$current_slot\" → \"$expected_slot\""

    echo "${LOG_PREFIX} [fix_memory_context_engine_slot] 完成"
}

# ========== 主执行逻辑 ==========
main() {
    echo "${LOG_PREFIX} 开始执行插件兼容修复"

    # 停 gateway 防止其 Config overwrite 与本脚本 jq 写入产生并发竞态。
    # 注意不能用 `pkill -f "openclaw"`：会误杀本脚本（路径含 .openclaw）与 tee 子进程。
    echo "${LOG_PREFIX} 停止 openclaw-gateway..."
    export XDG_RUNTIME_DIR="${XDG_RUNTIME_DIR:-/run/user/$(id -u)}"
    systemctl --user stop openclaw-gateway 2>/dev/null || true
    pkill -f "openclaw-gateway" 2>/dev/null || true
    pkill -f "node.*openclaw/.*gateway" 2>/dev/null || true
    sleep 1

    fix_wecom_legacy_names
    fix_ddingtalk_legacy_names
    fix_lark_legacy_names
    fix_memory_context_engine_slot

    echo ""
    echo "${LOG_PREFIX} 重启 openclaw-gateway..."
    systemctl --user restart openclaw-gateway 2>/dev/null || true

    echo ""
    echo "${LOG_PREFIX} 所有插件兼容修复执行完毕"
    echo "========== 日志结束: $(date '+%Y-%m-%d %H:%M:%S') =========="
    echo ""
}

main "$@"

# 兼容脚本永远以 0 退出，避免阻断升级主流程；问题通过日志体现
exit 0
