package controller

import (
	"encoding/base64"
	"os"
	"strings"
	"testing"
)

// ─── 占位符替换逻辑单元测试 ────────────────────────────────────────────────────

// TestRunScript_ParamSubstitution 验证 Go 侧占位符替换逻辑是否正确
func TestRunScript_ParamSubstitution(t *testing.T) {
	scriptTemplate := `#!/bin/bash
ARCHIVE_PATH="{{archivepath}}"
UPLOAD_URL_B64="{{uploadurlb64}}"
OFFSET="{{offset}}"
PART_SIZE="{{partsize}}"
PART_NUMBER="{{partnumber}}"
TOTAL_PARTS="{{totalparts}}"

if [ -z "$ARCHIVE_PATH" ] || [ "$ARCHIVE_PATH" = "{{archivepath}}" ]; then
	echo "✗ 错误: archivepath 参数未注入"; exit 1
fi
echo "✓ archivepath=$ARCHIVE_PATH"
`

	uploadURL := "https://example.cos.ap-guangzhou.myqcloud.com/upload?partNumber=1&uploadId=abc123&token=xyz&sign=foo%2Bbar"
	uploadURLB64 := base64.StdEncoding.EncodeToString([]byte(uploadURL))

	params := map[string]string{
		"archivepath":  "/tmp/openclaw-state-20260403_205631.tgz",
		"uploadurlb64": uploadURLB64,
		"offset":       "0",
		"partsize":     "10485760",
		"partnumber":   "1",
		"totalparts":   "14",
	}

	// 模拟 RunScript 内部的替换逻辑
	finalScript := scriptTemplate
	for k, v := range params {
		finalScript = strings.ReplaceAll(finalScript, "{{"+k+"}}", v)
	}

	t.Logf("替换后脚本片段:\n%s", finalScript[:min(len(finalScript), 300)])

	// 1. 验证所有占位符都已被替换
	if strings.Contains(finalScript, "{{") {
		// 找出哪些占位符未被替换
		for _, line := range strings.Split(finalScript, "\n") {
			if strings.Contains(line, "{{") {
				t.Errorf("发现未替换的占位符: %s", line)
			}
		}
	}

	// 2. 验证各参数值已正确注入
	checks := map[string]string{
		"archivepath":  "/tmp/openclaw-state-20260403_205631.tgz",
		"uploadurlb64": uploadURLB64,
		"offset":       "0",
		"partsize":     "10485760",
		"partnumber":   "1",
		"totalparts":   "14",
	}
	for key, expected := range checks {
		if !strings.Contains(finalScript, expected) {
			t.Errorf("参数 %s 的值 %q 未出现在替换后的脚本中", key, expected)
		}
	}

	// 3. 验证脚本中的校验逻辑不会误报（替换后 archivepath 不等于占位符字符串）
	if strings.Contains(finalScript, `"{{archivepath}}"`) {
		t.Error("archivepath 占位符未被替换，脚本校验会报错")
	}

	t.Log("✅ 占位符替换验证通过")
}

// TestRunScript_ParamSubstitution_MissingParam 验证：缺少参数时占位符保留，脚本校验能检测到
func TestRunScript_ParamSubstitution_MissingParam(t *testing.T) {
	scriptTemplate := `ARCHIVE_PATH="{{archivepath}}"
UPLOAD_URL_B64="{{uploadurlb64}}"`

	// 故意不传 uploadurlb64
	params := map[string]string{
		"archivepath": "/tmp/test.tgz",
		// 缺少 uploadurlb64
	}

	finalScript := scriptTemplate
	for k, v := range params {
		finalScript = strings.ReplaceAll(finalScript, "{{"+k+"}}", v)
	}

	t.Logf("替换后:\n%s", finalScript)

	// archivepath 应已替换
	if strings.Contains(finalScript, "{{archivepath}}") {
		t.Error("archivepath 应该已被替换")
	}
	// uploadurlb64 应仍为占位符（未传入）
	if !strings.Contains(finalScript, "{{uploadurlb64}}") {
		t.Error("uploadurlb64 未传入，占位符应保留")
	}
	t.Log("✅ 缺少参数时占位符正确保留")
}

// TestRunScript_UploadParams_Base64Roundtrip 验证 uploadurlb64 的 base64 编解码完整性
func TestRunScript_UploadParams_Base64Roundtrip(t *testing.T) {
	// 模拟真实的 SMH 分块上传 URL（含特殊字符）
	testURLs := []string{
		"https://example.cos.ap-guangzhou.myqcloud.com/file.tgz?partNumber=1&uploadId=abc123XYZ&token=foo%2Bbar%3D",
		"https://smh.tencentcs.com/upload?partNumber=14&uploadId=xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx&sign=abc+def/ghi=",
		"https://example.com/path?a=1&b=2&c=3",
	}

	for _, originalURL := range testURLs {
		encoded := base64.StdEncoding.EncodeToString([]byte(originalURL))

		// 验证 base64 编码后不含 & 等特殊字符（不会干扰脚本）
		if strings.Contains(encoded, "&") || strings.Contains(encoded, "?") {
			t.Errorf("base64 编码后不应含 & 或 ?，实际: %s", encoded)
		}

		// 模拟脚本中的 base64 解码
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			t.Errorf("base64 解码失败: %v，原始 URL: %s", err, originalURL)
			continue
		}
		if string(decoded) != originalURL {
			t.Errorf("base64 往返不一致\n原始: %s\n解码: %s", originalURL, string(decoded))
		}
	}
	t.Log("✅ base64 编解码往返验证通过")
}

// TestRunScript_LoadAndSubstitute_RealScript 加载真实的 upload_to_smh.sh 脚本并验证替换
func TestRunScript_LoadAndSubstitute_RealScript(t *testing.T) {
	// 初始化 LoadScript 读取真实脚本文件
	LoadScript = func(name string) (string, error) {
		data, err := os.ReadFile("../scripts/" + name)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}

	scriptContent, err := LoadScript("upload_to_smh.sh")
	if err != nil {
		t.Fatalf("加载 upload_to_smh.sh 失败: %v", err)
	}

	uploadURL := "https://example.cos.ap-guangzhou.myqcloud.com/file.tgz?partNumber=1&uploadId=testUploadId123&token=testToken"
	uploadURLB64 := base64.StdEncoding.EncodeToString([]byte(uploadURL))

	params := map[string]string{
		"archivepath":   "/tmp/openclaw-state-20260403_205631.tgz",
		"uploadurlb64":  uploadURLB64,
		"offset":        "0",
		"partsize":      "10485760",
		"partnumber":    "1",
		"totalparts":    "14",
		"headercount":   "2",
		"headerkvlines": "HEADER_0_KEY=\"x-smh-meta-a\"\nHEADER_0_VAL_B64=\"dGVzdA==\"\nHEADER_1_KEY=\"x-smh-meta-b\"\nHEADER_1_VAL_B64=\"dGVzdDI=\"",
	}

	// 执行替换（与 RunScript 内部逻辑完全一致）
	finalScript := scriptContent
	for k, v := range params {
		finalScript = strings.ReplaceAll(finalScript, "{{"+k+"}}", v)
	}

	// 检查所有占位符是否都已替换
	remainingPlaceholders := []string{}
	for _, line := range strings.Split(finalScript, "\n") {
		if strings.Contains(line, "{{") && !strings.HasPrefix(strings.TrimSpace(line), "#") {
			remainingPlaceholders = append(remainingPlaceholders, strings.TrimSpace(line))
		}
	}

	if len(remainingPlaceholders) > 0 {
		t.Errorf("替换后仍有 %d 处未替换的占位符（非注释行）:", len(remainingPlaceholders))
		for _, p := range remainingPlaceholders {
			t.Errorf("  → %s", p)
		}
	} else {
		t.Log("✅ upload_to_smh.sh 所有占位符替换完毕")
	}

	// 验证关键参数值已注入
	if !strings.Contains(finalScript, "/tmp/openclaw-state-20260403_205631.tgz") {
		t.Error("archivepath 值未注入脚本")
	}
	if !strings.Contains(finalScript, uploadURLB64) {
		t.Error("uploadurlb64 值未注入脚本")
	}

	// 验证脚本中的参数校验逻辑不会误触发
	// 即：替换后不存在 ARCHIVE_PATH="{{archivepath}}" 这样的赋值
	if strings.Contains(finalScript, `"{{archivepath}}"`) {
		t.Error("archivepath 占位符未被替换，CVM 上执行时会报 '参数未注入'")
	}
	if strings.Contains(finalScript, `"{{uploadurlb64}}"`) {
		t.Error("uploadurlb64 占位符未被替换，CVM 上执行时会报 '参数未注入'")
	}

	t.Logf("替换后脚本前 400 字节:\n%s", finalScript[:min(len(finalScript), 400)])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
