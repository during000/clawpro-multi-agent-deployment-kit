#!/bin/bash
set -uo pipefail

# ========== OpenClaw 备份包分块上传脚本 ==========
# 功能：在 CVM 上将备份压缩包的指定分块通过 HTTP PUT 上传到 SMH
#
# 参数传递方式（TAT 模板变量，由 Go 侧 RunScript params 注入）：
#   archivepath   - CVM 上的备份压缩包路径（必填）
#   uploadurlb64 - 当前分块的 SMH PUT 上传 URL 的 base64 编码（必填，含 partNumber 和 uploadId）
#                   使用 base64 编码是为了避免 URL 中的特殊字符（& 等）干扰 TAT 模板变量替换
#   offset        - 当前分块在文件中的字节偏移量（必填）
#   partsize      - 当前分块的最大字节数（必填）
#   partnumber    - 当前分块序号，从 1 开始（必填）
#   totalparts    - 总分块数（必填）

# ========== TAT 模板变量 ==========
ARCHIVE_PATH="{{archivepath}}"
UPLOAD_URL_B64="{{uploadurlb64}}"
OFFSET="{{offset}}"
PART_SIZE="{{partsize}}"
PART_NUMBER="{{partnumber}}"
TOTAL_PARTS="{{totalparts}}"
# SMH 分块 PUT 所需额外 Header
# Go 侧注入：每个 Header 的 key 直接赋值（ASCII 安全），value 单独 base64 编码
# 格式：HEADER_COUNT=N，HEADER_0_KEY=xxx，HEADER_0_VAL_B64=yyy，...
HEADER_COUNT={{headercount}}
{{headerkvlines}}
# 构建 curl -H 参数数组
EXTRA_HEADER_ARGS=()
for (( _i=0; _i<HEADER_COUNT; _i++ )); do
    _key_var="HEADER_${_i}_KEY"
    _val_var="HEADER_${_i}_VAL_B64"
    _key="${!_key_var}"
    _val=$(printf '%s' "${!_val_var}" | base64 -d 2>/dev/null)
    echo "  Header[$_i]: ${_key} = ${_val:0:60}..."
    EXTRA_HEADER_ARGS+=(-H "${_key}: ${_val}")
done
echo "额外 Header 数量: ${#EXTRA_HEADER_ARGS[@]}"

# ========== 参数校验 ==========
if [ -z "$ARCHIVE_PATH" ] || [ "$ARCHIVE_PATH" = "{{archivepath}}" ]; then
    echo "✗ 错误: archivepath 参数未注入"; exit 1
fi
if [ -z "$UPLOAD_URL_B64" ] || [ "$UPLOAD_URL_B64" = "{{uploadurlb64}}" ]; then
    echo "✗ 错误: uploadurlb64 参数未注入"; exit 1
fi
if [ -z "$OFFSET" ] || [ "$OFFSET" = "{{offset}}" ]; then
    echo "✗ 错误: offset 参数未注入"; exit 1
fi
if [ -z "$PART_SIZE" ] || [ "$PART_SIZE" = "{{partsize}}" ]; then
    echo "✗ 错误: partsize 参数未注入"; exit 1
fi
if [ -z "$PART_NUMBER" ] || [ "$PART_NUMBER" = "{{partnumber}}" ]; then
    echo "✗ 错误: partnumber 参数未注入"; exit 1
fi
if [ -z "$TOTAL_PARTS" ] || [ "$TOTAL_PARTS" = "{{totalparts}}" ]; then
    echo "✗ 错误: totalparts 参数未注入"; exit 1
fi

# base64 解码得到真实上传 URL
UPLOAD_URL=$(echo "$UPLOAD_URL_B64" | base64 -d 2>/dev/null)
if [ -z "$UPLOAD_URL" ]; then
    echo "✗ 错误: uploadurl_b64 base64 解码失败"; exit 1
fi

echo "=== OpenClaw 备份包分块上传到 SMH ==="
echo "压缩包路径: $ARCHIVE_PATH"
echo "分块: $PART_NUMBER / $TOTAL_PARTS"
echo "偏移量: $OFFSET 字节，分块大小上限: $PART_SIZE 字节"

if [ ! -f "$ARCHIVE_PATH" ]; then
    echo "✗ 错误: 备份压缩包不存在: $ARCHIVE_PATH"
    exit 1
fi

ARCHIVE_SIZE=$(stat -c%s "$ARCHIVE_PATH" 2>/dev/null || stat -f%z "$ARCHIVE_PATH" 2>/dev/null)
echo "压缩包总大小: ${ARCHIVE_SIZE} bytes"

# ========== 计算实际分块大小（最后一块可能更小）==========
REMAINING=$(( ARCHIVE_SIZE - OFFSET ))
if [ "$REMAINING" -le 0 ]; then
    echo "✗ 错误: 偏移量 $OFFSET 超出文件大小 $ARCHIVE_SIZE"
    exit 1
fi
ACTUAL_SIZE=$(( REMAINING < PART_SIZE ? REMAINING : PART_SIZE ))
echo "实际分块大小: ${ACTUAL_SIZE} bytes"

# ========== 用 dd 截取分块并通过 curl 上传 ==========
# dd 从文件指定偏移量读取 ACTUAL_SIZE 字节，通过管道传给 curl
# 使用 bs=1M 提升读取性能，skip/count 单位均为 1MB
echo "开始上传分块 $PART_NUMBER / $TOTAL_PARTS ..."

BLOCK_SIZE=$((1024 * 1024))  # 1MB
SKIP_BLOCKS=$(( OFFSET / BLOCK_SIZE ))
# OFFSET 不是 1MB 整数倍时，dd 实际读取起点比 OFFSET 早 SKIP_REMAINDER 字节
# 需要用 tail -c + 跳过多读的部分，确保精确从 OFFSET 处开始
SKIP_REMAINDER=$(( OFFSET % BLOCK_SIZE ))
# 向上取整：确保 count 覆盖 SKIP_REMAINDER + ACTUAL_SIZE 字节
COUNT_BLOCKS=$(( (SKIP_REMAINDER + ACTUAL_SIZE + BLOCK_SIZE - 1) / BLOCK_SIZE ))

RESP_BODY=$(mktemp)
HTTP_CODE=$(dd if="$ARCHIVE_PATH" bs="${BLOCK_SIZE}" skip="${SKIP_BLOCKS}" count="${COUNT_BLOCKS}" 2>/dev/null \
    | tail -c +"$(( SKIP_REMAINDER + 1 ))" \
    | head -c "${ACTUAL_SIZE}" \
    | curl -sS -o "$RESP_BODY" -w "%{http_code}" \
        -X PUT \
        -H "Content-Length: ${ACTUAL_SIZE}" \
        "${EXTRA_HEADER_ARGS[@]}" \
        --data-binary @- \
        --retry 3 \
        --retry-delay 5 \
        "$UPLOAD_URL")
CURL_EXIT=$?

if [ "$CURL_EXIT" -ne 0 ] || [ "$HTTP_CODE" -lt 200 ] || [ "$HTTP_CODE" -ge 300 ]; then
    echo "✗ 分块 $PART_NUMBER 上传失败 (HTTP $HTTP_CODE, curl_exit=$CURL_EXIT)"
    echo "响应内容: $(cat "$RESP_BODY")"
    rm -f "$RESP_BODY"
    exit 1
fi
rm -f "$RESP_BODY"

echo "✓ 分块 $PART_NUMBER 上传成功 (HTTP $HTTP_CODE)"
echo "=== 分块上传完成 ==="
