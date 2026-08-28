#!/bin/bash
set -euo pipefail

export PATH="$HOME/.npm-global/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u)

ENV_JSON='{{env_json}}'
DROPIN_DIR="$HOME/.config/systemd/user/openclaw-gateway.service.d"
ENV_FILE="$DROPIN_DIR/.openclaw_env"
DROPIN_FILE="$DROPIN_DIR/env.conf"
LOCK_FILE="/tmp/.openclaw_set_env.lock"

# 使用文件锁避免并发重启服务触发 systemd start rate limit
exec 200>"$LOCK_FILE"
if ! flock -n 200; then
    echo "环境变量设置操作正在进行中，请勿重复提交" >&2
    exit 1
fi

# 读取现有环境变量为 jq 对象
CURRENT='{}'
if [ -f "$ENV_FILE" ]; then
    while IFS= read -r line || [ -n "$line" ]; do
        # 跳过空行和注释
        [[ -z "$line" || "$line" == \#* ]] && continue
        # 取第一个 = 之前为 key，之后全部为 value（value 中可能包含 =）
        key="${line%%=*}"
        val="${line#*=}"
        CURRENT=$(echo "$CURRENT" | jq --arg k "$key" --arg v "$val" '.[$k] = $v')
    done < "$ENV_FILE"
fi

# 合并：遍历传入的 env_json，null 值删除，string 值更新/新增
MERGED=$(echo "$CURRENT" | jq --argjson input "$ENV_JSON" '
    reduce ($input | to_entries[]) as $e (.;
        if $e.value == null then del(.[$e.key])
        else .[$e.key] = $e.value
        end
    )
')

# 写入环境变量文件
mkdir -p "$DROPIN_DIR"
echo "$MERGED" | jq -r 'to_entries[] | "\(.key)=\(.value)"' > "$ENV_FILE"

# 幂等创建 drop-in 配置
if [ ! -f "$DROPIN_FILE" ]; then
    cat > "$DROPIN_FILE" << CONF
[Service]
EnvironmentFile=-${HOME}/.config/systemd/user/openclaw-gateway.service.d/.openclaw_env
CONF
fi

# 重载并重启服务
systemctl --user daemon-reload
systemctl --user restart openclaw-gateway

echo "ok"
