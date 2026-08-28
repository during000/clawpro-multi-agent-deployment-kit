#!/bin/bash
set -euo pipefail

export PATH="$HOME/.npm-global/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u)

# 依赖检查
for cmd in jq systemctl flock; do
    command -v "$cmd" >/dev/null 2>&1 || { echo "错误: 缺少命令 $cmd"; exit 1; }
done

# remove_model_provider.sh — 从 openclaw.json 的 models.providers 中移除指定 provider，
# 同时从 agents.defaults.model.fallbacks 数组中剔除相关引用；若该 provider 恰好是
# 当前 primary，则一并清空 primary（由调用方负责后续再 set 新的 primary）。
# 同步处理 agents.defaults.imageModel：剔除 fallbacks 中前缀匹配项；若 imageModel.primary
# 指向被删 provider，则整个 imageModel 字段删除（5.7+），避免残留无效 ref。
#
# Parameters (substituted by TAT before execution):
#   {{provider}} - openclaw provider key（不含 /modelId 后缀）
#                  e.g. "hatchery-glm-4-plus", "custom-my-model"

PROVIDER="{{provider}}"
if [ -z "$PROVIDER" ]; then
    echo "错误: 参数 provider 不能为空"
    exit 1
fi

CONFIG="$HOME/.openclaw/openclaw.json"
LOCKFILE="$HOME/.openclaw/openclaw.lock"

# 确保配置目录存在，避免锁文件、备份文件或临时文件创建失败
mkdir -p "$HOME/.openclaw"

if [ ! -f "$CONFIG" ]; then
    echo "[remove_model_provider] openclaw.json 不存在，无需处理"
    exit 0
fi

# 使用 flock 文件锁防止与 switch_model.sh / set_model.sh 并发修改 openclaw.json
# -w 30 表示最多等待 30 秒获取锁，避免无限阻塞
(
flock -x -w 30 200 || { echo "错误: 获取文件锁超时"; exit 1; }

# 备份
cp "$CONFIG" "${CONFIG}.bak.$(date +%Y-%m-%dT%H:%M:%S)"

# 使用 mktemp 避免并发冲突，临时文件放在用户私有目录下防止被其他用户读取
TMPFILE="$(mktemp -p "$HOME/.openclaw" openclaw.XXXXXX.json)"
trap 'rm -f "$TMPFILE"' EXIT

# 一次 jq 完成：删 provider + 清 fallbacks + 清悬空 primary + 同步 imageModel
jq --arg provider "$PROVIDER" --arg prefix "${PROVIDER}/" '
    # 1) 删 providers 下的 key（若存在）
    (if (.models.providers // {}) | has($provider) then .models.providers |= del(.[$provider]) else . end)
    # 2) 过滤 fallbacks：去掉前缀匹配的项
    | (if .agents.defaults.model.fallbacks then
        .agents.defaults.model.fallbacks |= map(select(startswith($prefix) | not))
      else . end)
    # 3) 若当前 primary 以该 provider 开头，则置空，避免 gateway 加载到无效 primary
    | (if (.agents.defaults.model.primary // "" | startswith($prefix)) then
        .agents.defaults.model.primary = ""
      else . end)
    # 4) + 5) imageModel 处理：先判断类型，字符串格式（旧）直接 del，
    #    对象格式走原逻辑（剔 fallbacks + 清悬空 primary），null/不存在则不变
    | (if (.agents.defaults.imageModel | type) == "string" then
        (if (.agents.defaults.imageModel | startswith($prefix)) then
            del(.agents.defaults.imageModel)
          else . end)
      elif (.agents.defaults.imageModel | type) == "object" then
        (if .agents.defaults.imageModel.fallbacks then
            .agents.defaults.imageModel.fallbacks |= map(select(startswith($prefix) | not))
          else . end)
        | (if (.agents.defaults.imageModel.primary // "" | startswith($prefix)) then
            del(.agents.defaults.imageModel)
          else . end)
      else . end)
' "$CONFIG" > "$TMPFILE"

# mv 原子替换，避免 cp 截断导致文件损坏
mv "$TMPFILE" "$CONFIG"
trap - EXIT

) 200>"$LOCKFILE"

# 重启 gateway 使变更生效
systemctl --user restart openclaw-gateway || { echo "错误: openclaw-gateway 重启失败"; exit 1; }
