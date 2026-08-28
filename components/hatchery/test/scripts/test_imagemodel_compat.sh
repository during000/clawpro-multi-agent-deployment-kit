#!/bin/bash
# test_imagemodel_compat.sh — 验证 imageModel 字符串/对象格式兼容性
# 覆盖 set_model.sh / switch_model.sh / remove_model_provider.sh 中修改后的 jq 表达式
set -euo pipefail

PASS=0
FAIL=0

# set_model / switch_model 的 imageModel 处理 jq 片段（与脚本中修改后的逻辑一致）
SET_JQ='if $imageprimary == "" then del(.agents.defaults.imageModel) else (if (.agents.defaults.imageModel | type) == "string" then .agents.defaults.imageModel = {} else . end) | .agents.defaults.imageModel.primary = $imageprimary | .agents.defaults.imageModel.fallbacks = $imagefallbacks end'

# remove_model_provider 的 imageModel 处理 jq 片段（与脚本中修改后的逻辑一致）
RM_JQ='(if (.agents.defaults.imageModel | type) == "string" then (if (.agents.defaults.imageModel | startswith($prefix)) then del(.agents.defaults.imageModel) else . end) elif (.agents.defaults.imageModel | type) == "object" then (if .agents.defaults.imageModel.fallbacks then .agents.defaults.imageModel.fallbacks |= map(select(startswith($prefix) | not)) else . end) | (if (.agents.defaults.imageModel.primary // "" | startswith($prefix)) then del(.agents.defaults.imageModel) else . end) else . end)'

assert_eq() {
    local name="$1" expected="$2" actual="$3"
    if [ "$expected" = "$actual" ]; then
        echo "PASS: $name"
        PASS=$((PASS+1))
    else
        echo "FAIL: $name"
        echo "  expected: $expected"
        echo "  actual:   $actual"
        FAIL=$((FAIL+1))
    fi
}

# ========== set_model / switch_model 测试 ==========

# TC-01: set_model imageModel=string → 重置为对象
result=$(echo '{"agents":{"defaults":{"imageModel":"old-ref"}}}' | jq -c --arg imageprimary "new-ref" --argjson imagefallbacks '[]' "$SET_JQ")
assert_eq "TC-01 set_model string" '{"agents":{"defaults":{"imageModel":{"primary":"new-ref","fallbacks":[]}}}}' "$result"

# TC-02: set_model imageModel=object → 正常更新
result=$(echo '{"agents":{"defaults":{"imageModel":{"primary":"old","fallbacks":["old-fb"]}}}}' | jq -c --arg imageprimary "new-ref" --argjson imagefallbacks '["new-fb"]' "$SET_JQ")
assert_eq "TC-02 set_model object" '{"agents":{"defaults":{"imageModel":{"primary":"new-ref","fallbacks":["new-fb"]}}}}' "$result"

# TC-03: set_model imageModel=missing → 自动创建
result=$(echo '{"agents":{"defaults":{}}}' | jq -c --arg imageprimary "new-ref" --argjson imagefallbacks '[]' "$SET_JQ")
assert_eq "TC-03 set_model missing" '{"agents":{"defaults":{"imageModel":{"primary":"new-ref","fallbacks":[]}}}}' "$result"

# TC-04: set_model imageprimary="" → del imageModel
result=$(echo '{"agents":{"defaults":{"imageModel":"old-ref"}}}' | jq -c --arg imageprimary "" --argjson imagefallbacks '[]' "$SET_JQ")
assert_eq "TC-04 set_model empty primary" '{"agents":{"defaults":{}}}' "$result"

# TC-05: switch_model imageModel=string（逻辑同 set_model）
result=$(echo '{"agents":{"defaults":{"imageModel":"old-ref"}}}' | jq -c --arg imageprimary "new-ref" --argjson imagefallbacks '[]' "$SET_JQ")
assert_eq "TC-05 switch_model string" '{"agents":{"defaults":{"imageModel":{"primary":"new-ref","fallbacks":[]}}}}' "$result"

# TC-06: switch_model imageModel=object（逻辑同 set_model）
result=$(echo '{"agents":{"defaults":{"imageModel":{"primary":"old","fallbacks":["old-fb"]}}}}' | jq -c --arg imageprimary "new-ref" --argjson imagefallbacks '["new-fb"]' "$SET_JQ")
assert_eq "TC-06 switch_model object" '{"agents":{"defaults":{"imageModel":{"primary":"new-ref","fallbacks":["new-fb"]}}}}' "$result"

# ========== remove_model_provider 测试 ==========

# TC-07: remove imageModel=string, 匹配 prefix → del
result=$(echo '{"agents":{"defaults":{"imageModel":"hatchery-glm-4-plus/glm-4-plus"}}}' | jq -c --arg provider "hatchery-glm-4-plus" --arg prefix "hatchery-glm-4-plus/" "$RM_JQ")
assert_eq "TC-07 remove string (match)" '{"agents":{"defaults":{}}}' "$result"

# TC-07b: remove imageModel=string, 不匹配 prefix → 保留
result=$(echo '{"agents":{"defaults":{"imageModel":"hatchery-qwen3.7-max/qwen3.7-max"}}}' | jq -c --arg provider "hatchery-glm-4-plus" --arg prefix "hatchery-glm-4-plus/" "$RM_JQ")
assert_eq "TC-07b remove string (no match)" '{"agents":{"defaults":{"imageModel":"hatchery-qwen3.7-max/qwen3.7-max"}}}' "$result"

# TC-08: remove imageModel=object, primary 匹配 → del imageModel
result=$(echo '{"agents":{"defaults":{"imageModel":{"primary":"hatchery-glm-4-plus/glm-4-plus","fallbacks":["hatchery-dall-e-3/dall-e-3"]}}}}' | jq -c --arg provider "hatchery-glm-4-plus" --arg prefix "hatchery-glm-4-plus/" "$RM_JQ")
assert_eq "TC-08 remove object (primary match)" '{"agents":{"defaults":{}}}' "$result"

# TC-08b: remove imageModel=object, primary 不匹配 → 仅过滤 fallbacks
result=$(echo '{"agents":{"defaults":{"imageModel":{"primary":"hatchery-dall-e-3/dall-e-3","fallbacks":["hatchery-glm-4-plus/glm-4-plus","hatchery-dall-e-3/dall-e-3"]}}}}' | jq -c --arg provider "hatchery-glm-4-plus" --arg prefix "hatchery-glm-4-plus/" "$RM_JQ")
assert_eq "TC-08b remove object (primary not match)" '{"agents":{"defaults":{"imageModel":{"primary":"hatchery-dall-e-3/dall-e-3","fallbacks":["hatchery-dall-e-3/dall-e-3"]}}}}' "$result"

# TC-09: remove imageModel=missing → 不变
result=$(echo '{"agents":{"defaults":{}}}' | jq -c --arg provider "hatchery-glm-4-plus" --arg prefix "hatchery-glm-4-plus/" "$RM_JQ")
assert_eq "TC-09 remove missing" '{"agents":{"defaults":{}}}' "$result"

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
[ $FAIL -eq 0 ]
