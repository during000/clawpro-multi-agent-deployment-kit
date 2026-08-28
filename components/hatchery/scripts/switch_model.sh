#!/bin/bash
set -euo pipefail

export PATH="$HOME/.npm-global/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u)

# 依赖检查
for cmd in openclaw jq base64 systemctl flock; do
    command -v "$cmd" >/dev/null 2>&1 || { echo "错误: 缺少命令 $cmd"; exit 1; }
done

# switch_model.sh — 切换主模型（不写入 provider 配置，仅切换 primary + 更新 fallbacks 并重启 gateway）
#
# Parameters (substituted by TAT before execution):
#   {{primary}}           - 新主模型 ref "providerKey/modelId"（字符集受 Go 侧白名单约束）
#                            e.g. "hatchery-glm-4-plus/glm-4-plus"
#   {{fallbacksb64}}      - base64(fallbacks JSON 数组). decode 后 jq 直接写入 openclaw.json。
#                            openclaw CLI 当前不支持 `models set fallbacks`，走 jq。
#                            e.g. base64 of '["hatchery-qwen-max/qwen-max","hatchery-deepseek-chat/deepseek-chat"]'
#   {{imageprimary}}      - imageModel.primary ref（OpenClaw 5.7+，可选）。
#                            空字符串表示实例当前无 image-capable 模型，脚本侧执行
#                            del(.agents.defaults.imageModel)。旧版 hatchery 不传时为空，
#                            走删除分支，向下兼容。
#   {{imagefallbacksb64}} - base64(imageModel.fallbacks JSON 数组). 同上，可选。
#
# 前置：provider 配置已存在于 openclaw.json 中（通过 add-model 接口的 set_model.sh 写入）
# 因此本脚本只需用 jq 更新 agents.defaults.model（primary + fallbacks）并重启 gateway。

CONFIG="$HOME/.openclaw/openclaw.json"
LOCKFILE="$HOME/.openclaw/openclaw.lock"

# base64 解码 fallbacks，解码失败立即报错退出，避免以空值覆盖配置
FALLBACKS_JSON="$(printf '%s' '{{fallbacksb64}}' | base64 -d)" || {
    echo "错误: fallbacksb64 base64 解码失败，中止执行"
    exit 1
}
# 校验解码结果是合法的 JSON（防止乱码或截断内容写入配置）
echo "$FALLBACKS_JSON" | jq empty || { echo "错误: fallbacks JSON 无效，中止执行"; exit 1; }

# imageModel.fallbacks 解码：imagefallbacksb64 缺省（旧 hatchery）或空时回退到空数组，
# 与 imageprimary 为空一同走 del 分支
IMAGE_FALLBACKS_B64='{{imagefallbacksb64}}'
if [ -z "$IMAGE_FALLBACKS_B64" ]; then
    IMAGE_FALLBACKS_JSON='[]'
else
    IMAGE_FALLBACKS_JSON="$(printf '%s' "$IMAGE_FALLBACKS_B64" | base64 -d)" || {
        echo "错误: imagefallbacksb64 base64 解码失败，中止执行"
        exit 1
    }
    echo "$IMAGE_FALLBACKS_JSON" | jq empty || { echo "错误: imageModel.fallbacks JSON 无效，中止执行"; exit 1; }
fi

# 提前检查配置文件是否存在，避免 primary 已切换后才发现 fallbacks 无法更新
if [ ! -f "$CONFIG" ]; then
    echo "错误: $CONFIG 不存在，中止执行"
    exit 1
fi

# 使用 flock 文件锁防止多实例并发修改 openclaw.json
# 带 30s 超时，避免无限阻塞放大 DB 与 openclaw.json 的不一致窗口
# （与 set_model.sh / remove_model_provider.sh 对齐）。
(
flock -x -w 30 200 || { echo "错误: 获取文件锁超时（30s），中止执行"; exit 1; }

# 1) 用 jq 直接写入 primary 和 fallbacks（不走 openclaw models set，避免 CLI 把 provider key 转小写
#    以及覆盖/重写 models.providers 配置）
#    同时把 providers 里的旧格式 key（zhipu/qcloudlkeap 等）规范化为新格式（hatchery-{ModelID}）：
#    从 primary 和 fallbacks 里解析 {newKey, modelId}，找 providers 里包含该 modelId 的旧 key，
#    若 key 不同则迁移内容到新 key 并删除旧 key（幂等，存量兼容）。
cp "$CONFIG" "${CONFIG}.bak.$(date +%Y-%m-%dT%H:%M:%S)"
TMPFILE="$(mktemp /tmp/openclaw.XXXXXX.json)"
trap 'rm -f "$TMPFILE"' EXIT
jq --arg primary "{{primary}}" \
   --argjson fallbacks "$FALLBACKS_JSON" \
   --arg imageprimary '{{imageprimary}}' \
   --argjson imagefallbacks "$IMAGE_FALLBACKS_JSON" \
   '
   # 从 "providerKey/modelId" 格式解析出 {key, modelId}
   # modelId 取第一个 "/" 之后的全部内容，兼容 modelId 本身含 "/" 的情况
   def parse_ref(r):
     (r | ascii_downcase) as $r_lower |
     ($r_lower | index("/")) as $i |
     if $i then {key: $r_lower[0:$i], modelId: $r_lower[$i+1:]}
     else {key: $r_lower, modelId: $r_lower}
     end;

   # 判断一个 provider key 是否属于"旧格式裸 key"（不含 "-"）。
   # 新格式 key 形如 hatchery-deepseek-v3.1 / custom-deepseek-v3.1，必含 "-"。
   def is_legacy_bare_key(k): (k | index("-")) == null;

   # 保存文档根，防止 map 管道改变 . 的指向
   . as $doc |

   # 收集 primary + fallbacks 里所有非空 ref 解析结果
   ([$primary] + $fallbacks) | map(select(. != null and . != "")) | map(parse_ref(.)) as $refs |

   # 回到文档根开始 reduce，对每个 ref 规范化旧格式 provider key：
   # 1) 旧裸 key（不含 "-"，如 "hatchery"/"zhipu"）→ 新格式 key
   # 2) 旧 prefix 变更的 key（如 "hatchery-deepseek-chat" → "deepseek-deepseek-chat"）：
   #    当 ref 中的 newKey 在 providers 中不存在，但有另一个 key 包含相同 modelId 时，
   #    将其 rename 为 newKey。仅在 newKey 不存在时触发，避免误改合法并存的多通道配置。
   $doc |
   reduce $refs[] as $ref (
     .;
     ($ref.key) as $newKey |
     ($ref.modelId) as $mid |
     # 优先匹配旧裸 key（不含 "-"）
     (
       .models.providers // {} | to_entries |
       map(select(
        is_legacy_bare_key(.key) and
         (.value.models? // [] | map(.id | ascii_downcase | gsub("[^a-z0-9._-]"; "-") == $mid) | any)
       )) |
       if length > 0 then .[0] else null end
     ) as $old_bare |
     if $old_bare != null and $old_bare.key != $newKey then
       .models.providers[$newKey] = $old_bare.value |
       del(.models.providers[$old_bare.key])
     else
       # 裸 key 未匹配：检查 newKey 是否已存在于 providers 中，
       # 若不存在则查找包含相同 modelId 的其他 key 进行 rename（prefix 变更场景）
       if (.models.providers[$newKey] == null) then
         (
           .models.providers // {} | to_entries |
           map(select(
             .key != $newKey and
             (.value.models? // [] | map(.id | ascii_downcase | gsub("[^a-z0-9._-]"; "-") == $mid) | any)
           )) |
           if length > 0 then .[0] else null end
         ) as $old_prefix |
         if $old_prefix != null then
           .models.providers[$newKey] = $old_prefix.value |
           del(.models.providers[$old_prefix.key])
         else . end
       else . end
     end
   )
   | .agents.defaults.model.primary = $primary
   | .agents.defaults.model.fallbacks = $fallbacks
   # imageModel 同步：与 model 写入在同一 jq pipeline 内完成，保证原子性。
   # imageprimary 为空（候选为空 / 旧 hatchery 不传）→ 删除整个 imageModel 字段。
   | if $imageprimary == "" then
       del(.agents.defaults.imageModel)
     else
       # 兼容旧格式：imageModel 为字符串时先重置为空对象，避免 jq "Cannot index string" 报错
       (if (.agents.defaults.imageModel | type) == "string" then
          .agents.defaults.imageModel = {}
        else . end)
       | .agents.defaults.imageModel.primary = $imageprimary
       | .agents.defaults.imageModel.fallbacks = $imagefallbacks
     end
   ' \
   "$CONFIG" > "$TMPFILE"
mv "$TMPFILE" "$CONFIG"
trap - EXIT

) 200>"$LOCKFILE"

# 3) 重启 gateway 使变更生效
systemctl --user restart openclaw-gateway || { echo "错误: 重启服务失败"; exit 1; }
