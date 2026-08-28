#!/bin/bash
set -euo pipefail

export PATH="$HOME/.npm-global/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u)
export NO_COLOR=1

# ========== 日志系统初始化 ==========
LOG_DIR="/var/log/clawpro"
if [ ! -d "$LOG_DIR" ]; then
    mkdir -p "$LOG_DIR"
fi
SCRIPT_NAME="add_plugin"
LOG_FILE="$LOG_DIR/${SCRIPT_NAME}.log"
# 将 stdout 和 stderr 同时输出到终端和日志文件
exec > >(tee -a "$LOG_FILE") 2>&1
echo ""
echo "========== 日志开始: $(date '+%Y-%m-%d %H:%M:%S') =========="

# ========== 参数解析 ==========
fullname="{{plugin}}"
id=$(basename "$fullname")
install_target="{{install_target}}"
config_file="$HOME/.openclaw/openclaw.json"
ext_base="$HOME/.openclaw/extensions"
ext_dir="$ext_base/$id"

echo "=== OpenClaw 插件安装/更新 ==="
echo "插件全名(fullname): $fullname"
echo "插件 ID: $id"
echo "安装目标(install_target): $install_target"
echo "配置文件: $config_file"
echo "插件目录: $ext_dir"

# ========== 第一步：清理配置中的脏数据 ==========
echo ""
echo ">>> [步骤 1/5] 清理配置中的脏数据（allowlist / entries）"
if [ -f "$config_file" ]; then
    valid_allow="[]"
    del_entries="[]"
    removed_allow=()
    removed_entries=()
    for p in $(jq -r '.plugins.allow // [] | .[]' "$config_file" 2>/dev/null); do
        if [ -d "$ext_base/$p" ]; then
            valid_allow=$(echo "$valid_allow" | jq --arg p "$p" '. + [$p]')
        else
            removed_allow+=("$p")
        fi
    done
    for p in $(jq -r '.plugins.entries // {} | keys[]' "$config_file" 2>/dev/null); do
        if [ ! -d "$ext_base/$p" ]; then
            del_entries=$(echo "$del_entries" | jq --arg p "$p" '. + [$p]')
            removed_entries+=("$p")
        fi
    done
    if [ ${#removed_allow[@]} -gt 0 ]; then
        echo "⚠ 从 plugins.allow 移除不存在的插件: ${removed_allow[*]}"
    fi
    if [ ${#removed_entries[@]} -gt 0 ]; then
        echo "⚠ 从 plugins.entries 移除不存在的插件: ${removed_entries[*]}"
    fi
    if jq --argjson allow "$valid_allow" --argjson del "$del_entries" '
            .plugins.allow = $allow | reduce $del[] as $k (.; del(.plugins.entries[$k]))
        ' "$config_file" > /tmp/openclaw_fix.json && mv /tmp/openclaw_fix.json "$config_file"; then
        echo "✓ 配置清理完成"
    else
        echo "⚠ 配置清理失败（已忽略）"
    fi
else
    echo "⚠ 配置文件不存在: $config_file，跳过清理"
fi

# ========== 第二步：检查插件是否已安装 ==========
echo ""
echo ">>> [步骤 2/5] 检查插件安装状态"
installed=false
if openclaw plugins list --json 2>/dev/null | sed -n '/^{/,/^}/p' | jq -e ".plugins[] | select(.id == \"$id\")" > /dev/null 2>&1; then
    installed=true
    echo "✓ 插件 $id 已安装"
else
    echo "插件 $id 未安装"
fi

# ========== 第三步：安装或更新插件 ==========
echo ""
echo ">>> [步骤 3/5] 安装/更新插件"
if [ "$installed" = true ] && [ "$install_target" = "$fullname" ]; then
    # 未指定版本，正常更新
    echo "未指定版本，执行更新: openclaw plugins update $id"
    if openclaw plugins update "$id"; then
        echo "✓ 插件 $id 更新成功"
    else
        echo "✗ 插件 $id 更新失败"
        exit 1
    fi
else
    # 新安装或指定版本重装：清理后安装
    if [ -d "$ext_dir" ]; then
        echo "清理旧插件目录: $ext_dir"
        rm -rf "$ext_dir"
    fi
    if [ -f "$config_file" ]; then
        jq --arg id "$id" \
            '.plugins.allow = ((.plugins.allow // []) | map(select(. != $id)))' \
            "$config_file" > /tmp/openclaw_tmp.json && mv /tmp/openclaw_tmp.json "$config_file"
        echo "✓ 已从 plugins.allow 中临时移除 $id"
    fi
    echo "执行安装: openclaw plugins install $install_target"
    if openclaw plugins install "$install_target"; then
        echo "✓ 插件 $install_target 安装成功"
    else
        echo "✗ 插件 $install_target 安装失败"
        exit 1
    fi
fi

echo "install/update success"

# ========== 第四步：更新 allowlist 并启用插件 ==========
echo ""
echo ">>> [步骤 4/5] 更新 allowlist 并启用插件"
backup_file="${config_file}.bak.$(date +%Y-%m-%dT%H:%M:%S)"
cp "$config_file" "$backup_file"
echo "✓ 配置文件已备份: $backup_file"

jq --arg id "$id" '.plugins.allow = ((.plugins.allow // []) | if index($id) then . else . + [$id] end)' \
    "$config_file" > /tmp/openclaw.json && mv /tmp/openclaw.json "$config_file"
echo "✓ 插件 $id 已加入 plugins.allow 白名单"

echo "启用插件: openclaw plugins enable $id"
if openclaw plugins enable "$id"; then
    echo "✓ 插件 $id 已启用"
else
    echo "⚠ 插件 $id 启用失败（已忽略，可手动执行）"
fi

# ========== 第五步：重启 gateway ==========
echo ""
echo ">>> [步骤 5/5] 重启 openclaw-gateway 服务"
if systemctl --user restart openclaw-gateway; then
    echo "✓ openclaw-gateway 已重启"
else
    echo "⚠ openclaw-gateway 重启失败，请手动执行: systemctl --user restart openclaw-gateway"
fi

echo ""
echo "=== 插件安装/更新完成: $id ==="
echo "========== 日志结束: $(date '+%Y-%m-%d %H:%M:%S') =========="
