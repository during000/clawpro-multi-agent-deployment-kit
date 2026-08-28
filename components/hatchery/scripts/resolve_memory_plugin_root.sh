#!/bin/bash
# 探测记忆插件安装路径（按优先级返回 $HOME/... 形式）
#
# 参数（由 TAT EnableParameter 机制注入）：
#   {{memory_plugin_id}}  - 插件 ID（如 memory-tencentdb）
#   {{memory_npm_pkg}}    - npm 包名（如 @tencentdb-agent-memory/memory-tencentdb）
#
# 探测优先级：
#   1. ~/.openclaw/npm/projects/tencentdb-agent-memory-<id>-<hash>/node_modules/<pkg>  (OpenClaw 5.28+)
#   2. ~/.openclaw/npm/node_modules/<pkg>                                              (OpenClaw 5.2 ~ 5.7)
#   3. ~/.openclaw/extensions/<id>                                                     (OpenClaw ≤ 5.1)

_proj=$(ls -d $HOME/.openclaw/npm/projects/tencentdb-agent-memory-{{memory_plugin_id}}-*/node_modules/{{memory_npm_pkg}} 2>/dev/null | head -1)
_npm="$HOME/.openclaw/npm/node_modules/{{memory_npm_pkg}}"
_ext="$HOME/.openclaw/extensions/{{memory_plugin_id}}"

if [ -n "$_proj" ] && [ -d "$_proj" ]; then
  echo "$_proj" | sed "s|^$HOME|\$HOME|"
elif [ -d "$_npm" ]; then
  echo '$HOME/.openclaw/npm/node_modules/{{memory_npm_pkg}}'
elif [ -d "$_ext" ]; then
  echo '$HOME/.openclaw/extensions/{{memory_plugin_id}}'
else
  echo ""
fi
