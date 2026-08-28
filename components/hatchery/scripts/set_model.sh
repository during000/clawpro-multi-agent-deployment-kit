#!/bin/bash
set -euo pipefail

export PATH="$HOME/.npm-global/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u)

# Parameters (substituted by TAT before execution):
#   {{valueb64}}          - base64(provider JSON object). 在 shell 里 decode 为 JSON 字符串，
#                            避免原始 JSON 中的单引号/反引号/$ 等字符污染 shell 语法。
#   {{provider}}          - openclaw provider key, e.g. hatchery-glm-4-plus, custom-my-model
#                            由后端 SlugifyModelID 生成，字符集 [a-z0-9._-]
#   {{model}}             - slugified model id, e.g. glm-4-plus（同上，白名单字符集）
#   {{primary}}           - primary model ref "providerKey/modelId"，字符集 [a-z0-9._-/]
#   {{fallbacksb64}}      - base64(fallbacks JSON array). decode 后 jq 直接写入
#                            agents.defaults.model.fallbacks。openclaw CLI 目前不支持
#                            `models set fallbacks`，必须走 jq 写 openclaw.json。
#   {{imageprimary}}      - imageModel.primary ref（OpenClaw 5.7+，可选）。
#                            空字符串表示实例当前无 image-capable 模型，脚本侧执行
#                            del(.agents.defaults.imageModel)。旧版 hatchery 不传时为空，
#                            走删除分支，向下兼容。
#   {{imagefallbacksb64}} - base64(imageModel.fallbacks JSON array). 同上，可选。

# 依赖检查
for cmd in jq base64 systemctl flock; do
    command -v "$cmd" >/dev/null 2>&1 || { echo "错误: 缺少命令 $cmd"; exit 1; }
done

# ========== 日志系统初始化 ==========
LOG_DIR="${HOME}/.openclaw/logs"
if [ ! -d "$LOG_DIR" ]; then
    mkdir -p "$LOG_DIR"
fi
SCRIPT_NAME="set_model"
LOG_FILE="$LOG_DIR/${SCRIPT_NAME}.log"
# 将 stdout 和 stderr 同时输出到终端和日志文件
exec > >(tee -a "$LOG_FILE") 2>&1
echo ""
echo "========== 日志开始: $(date '+%Y-%m-%d %H:%M:%S') =========="

echo "=== 设置模型配置 ==="
echo "provider: {{provider}}"
echo "model:    {{model}}"
echo "primary:  {{primary}}"

mkdir -p "$HOME/.openclaw"
CONFIG="$HOME/.openclaw/openclaw.json"
LOCKFILE="$HOME/.openclaw/openclaw.lock"

# 将 base64 占位符 decode 为原始 JSON 字符串（避免 shell 元字符解析风险）
# 在 flock 前 decode，decode 失败立即退出，不占用锁
echo ""
echo ">>> [步骤 1/3] 解码并校验参数"
NEW_VALUE="$(printf '%s' '{{valueb64}}' | base64 -d)" || {
    echo "✗ valueb64 base64 解码失败，中止执行"; exit 1
}
FALLBACKS_JSON="$(printf '%s' '{{fallbacksb64}}' | base64 -d)" || {
    echo "✗ fallbacksb64 base64 解码失败，中止执行"; exit 1
}
echo "$NEW_VALUE" | jq empty || { echo "✗ provider JSON 无效，中止执行"; exit 1; }
echo "$FALLBACKS_JSON" | jq empty || { echo "✗ fallbacks JSON 无效，中止执行"; exit 1; }

# valueb64 兼容两种格式：
#   - legacy：单个 provider 配置对象（无 mode 字段）
#   - batch：{mode:"batch",providers:[{provider,model,value},...]}
MODE="$(printf '%s' "$NEW_VALUE" | jq -r '.mode // ""')" || {
    echo "✗ 无法识别模型设置模式，中止执行"; exit 1
}
case "$MODE" in
    "")
        ;;
    "batch")
        printf '%s' "$NEW_VALUE" | jq -e '
          (.providers | type) == "array"
          and (.providers | length) > 0
          and all(.providers[];
            ((.provider | type) == "string" and (.provider | length) > 0)
            and ((.model | type) == "string" and (.model | length) > 0)
            and ((.value | type) == "object")
          )
          and (([.providers[].provider] | length) == ([.providers[].provider] | unique | length))
        ' >/dev/null || {
            echo "✗ batch providers 格式无效或 provider 重复，中止执行"; exit 1
        }
        ;;
    *)
        echo "✗ 不支持的模型设置模式: $MODE"; exit 1
        ;;
esac

# imageModel.fallbacks 解码：imagefallbacksb64 缺省（旧 hatchery）或空时回退到空数组，
# 与 imageprimary 为空一同走 del 分支
IMAGE_FALLBACKS_B64='{{imagefallbacksb64}}'
if [ -z "$IMAGE_FALLBACKS_B64" ]; then
    IMAGE_FALLBACKS_JSON='[]'
else
    IMAGE_FALLBACKS_JSON="$(printf '%s' "$IMAGE_FALLBACKS_B64" | base64 -d)" || {
        echo "✗ imagefallbacksb64 base64 解码失败，中止执行"; exit 1
    }
    echo "$IMAGE_FALLBACKS_JSON" | jq empty || { echo "✗ imageModel.fallbacks JSON 无效，中止执行"; exit 1; }
fi
echo "✓ 参数解码校验通过"

# 使用 flock 文件锁防止多实例并发修改 openclaw.json（与 switch_model.sh 共用同一锁文件）
(
# 先尝试非阻塞获取锁，失败则带超时等待（最多 30s），超时即报错退出，
# 避免无限阻塞放大 DB 与 openclaw.json 的不一致窗口（与 remove_model_provider.sh 对齐）。
if ! flock -n 200; then
    echo "[INFO] 🔒 openclaw.json 正被其他脚本修改，等待锁释放（最多 30s）..."
    flock -x -w 30 200 || { echo "✗ 获取文件锁超时（30s），中止执行"; exit 1; }
    echo "[INFO] 🔓 锁已获取，继续执行"
fi

echo ""
echo ">>> [步骤 2/3] 更新 openclaw.json 配置"

# Backup + 读取当前配置（在锁内读，确保读到最新版本）
if [ -f "$CONFIG" ]; then
    cp "$CONFIG" "${CONFIG}.bak.$(date +%Y-%m-%dT%H:%M:%S)"
    BASE_JSON="$(cat "$CONFIG")"
    echo "✓ 已备份当前配置"
else
    BASE_JSON='{"models":{"providers":{}}}'
    echo "⚠ 配置文件不存在，使用默认空配置"
fi

# 1) 写入/更新 provider 配置 + 同步写入 fallbacks（一次 jq 完成，原子性更好）
#    legacy value 和 batch providers 均通过 --argjson 传入，不会被 shell 再次解析
TMPFILE="$(mktemp /tmp/openclaw.XXXXXX.json)"
trap 'rm -f "$TMPFILE"' EXIT

echo "$BASE_JSON" \
    | jq --arg provider "{{provider}}" \
         --arg modelid "{{model}}" \
         --arg primary "{{primary}}" \
         --argjson newval "$NEW_VALUE" \
         --argjson fallbacks "$FALLBACKS_JSON" \
         --arg imageprimary '{{imageprimary}}' \
         --argjson imagefallbacks "$IMAGE_FALLBACKS_JSON" \
        '
        def parse_ref(r):
          (r | ascii_downcase) as $r_lower |
          ($r_lower | index("/")) as $i |
          if $i then {key: $r_lower[0:$i], modelId: $r_lower[$i+1:]}
          else {key: $r_lower, modelId: $r_lower}
          end;

        # 判断一个 provider key 是否属于"旧格式裸 provider key"。
        def is_legacy_bare_key(k): (k | index("-")) == null;

        # legacy 模式传入一个 provider；batch 模式逐项调用本函数。
        # 每项先清理持有相同 modelId 的旧裸 key，再写入新 provider。
        def upsert_provider($entry):
          ($entry.model) as $entry_model |
          .models.providers |= (. // {}) |
          .models.providers |= with_entries(
            select(
              (
                is_legacy_bare_key(.key) and
                (.value.models? // [] | map(.id | ascii_downcase | gsub("[^a-z0-9._-]"; "-") == $entry_model) | any)
              ) | not
            )
          ) |
          .models.providers[$entry.provider] = $entry.value;

        .models.providers |= (. // {})
        | if ($newval.mode // "") == "batch" then
            reduce $newval.providers[] as $entry (
              .;
              upsert_provider($entry)
            )
          else
            upsert_provider({provider: $provider, model: $modelid, value: $newval})
          end

        # 规范化 primary + fallbacks 里所有模型的旧格式 provider key：
        # 1) 旧裸 key（不含 "-"，如 "hatchery"/"zhipu"）→ 新格式 key
        # 2) 旧 prefix 变更的 key：当目标 key 不存在，但另一 key 包含同一 modelId 时 rename。
        | . as $doc |
          (([$primary] + $fallbacks) | map(select(. != null and . != "")) | map(parse_ref(.))) as $refs |
          $doc |
          reduce $refs[] as $ref (
            .;
            ($ref.key) as $newKey |
            ($ref.modelId) as $mid |
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
    > "$TMPFILE"

mv "$TMPFILE" "$CONFIG"
trap - EXIT
echo "✓ provider 配置已写入: {{provider}}"
echo "✓ primary 已设置为: {{primary}}"
echo "✓ fallbacks 已更新"

) 200>"$LOCKFILE"

# 2) 重启 gateway 使配置（尤其是 primary/fallbacks）生效
echo ""
echo ">>> [步骤 3/3] 重启 gateway"
echo "重启 openclaw-gateway..."
systemctl --user restart openclaw-gateway
echo "✓ openclaw-gateway 已重启"

echo ""
echo "=== 模型配置完成 ==="
echo "========== 日志结束: $(date '+%Y-%m-%d %H:%M:%S') =========="
