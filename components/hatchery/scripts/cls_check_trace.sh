#!/bin/bash
set -uo pipefail

# cls_check_trace.sh
# 检查 ~/.openclaw/openclaw.json 中 CLS trace 配置是否完整。
# 输出 JSON：
#   {"trace_enabled": true/false, "trace_topic_id": "xxx", "configured": true/false}
# configured=true 表示 enabled=true 且 traceTopicId 非空。

CONFIG_FILE="$HOME/.openclaw/openclaw.json"

if [ ! -f "$CONFIG_FILE" ]; then
    echo '{"trace_enabled":false,"trace_topic_id":"","configured":false,"reason":"config_not_found"}'
    exit 0
fi

# 使用 python3 解析 JSON（CVM 实例上通常有 python3）
RESULT=$(python3 - "$CONFIG_FILE" <<'PYEOF'
import sys, json

config_path = sys.argv[1]
try:
    with open(config_path, 'r') as f:
        config = json.load(f)
except Exception as e:
    print(json.dumps({"trace_enabled": False, "trace_topic_id": "", "configured": False, "reason": "parse_error"}))
    sys.exit(0)

plugins = config.get("plugins", {})
entries = plugins.get("entries", {})
cls_plugin = entries.get("clawpro-diagnostics-metrics-cls", {})
cls_config = cls_plugin.get("config", {})
trace_config = cls_config.get("trace", {})

trace_enabled = trace_config.get("enabled", False)
trace_topic_id = trace_config.get("traceTopicId", "")

configured = bool(trace_enabled) and bool(trace_topic_id)

print(json.dumps({
    "trace_enabled": bool(trace_enabled),
    "trace_topic_id": str(trace_topic_id),
    "configured": configured
}))
PYEOF
)

echo "$RESULT"
