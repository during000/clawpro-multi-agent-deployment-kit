package common

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// ─── ValidateModelType ────────────────────────────────────────────────────────

func TestValidateModelType_Valid(t *testing.T) {
	for _, mt := range []string{ModelTypeOpenAICompletions, ModelTypeAnthropicMessages} {
		if err := ValidateModelType(mt); err != nil {
			t.Errorf("期望合法，实际错误: %v (type=%s)", err, mt)
		}
	}
}

func TestValidateModelType_Invalid(t *testing.T) {
	for _, mt := range []string{"", "gpt-4", "openai", "unknown-type"} {
		if err := ValidateModelType(mt); err == nil {
			t.Errorf("期望返回错误，实际 nil (type=%s)", mt)
		}
	}
}

// ─── ValidateInputTypes ───────────────────────────────────────────────────────

func TestValidateInputTypes_Empty_DefaultsToText(t *testing.T) {
	result, err := ValidateInputTypes(nil)
	if err != nil {
		t.Fatalf("不应返回错误: %v", err)
	}
	if len(result) != 1 || result[0] != "text" {
		t.Errorf("空输入应返回 [text]，实际=%v", result)
	}
}

func TestValidateInputTypes_Valid(t *testing.T) {
	result, err := ValidateInputTypes([]string{"text", "image"})
	if err != nil {
		t.Fatalf("不应返回错误: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("期望 2 个元素，实际=%d", len(result))
	}
}

func TestValidateInputTypes_Dedup(t *testing.T) {
	result, err := ValidateInputTypes([]string{"text", "text", "image"})
	if err != nil {
		t.Fatalf("不应返回错误: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("期望去重后 2 个元素，实际=%d", len(result))
	}
}

func TestValidateInputTypes_Invalid(t *testing.T) {
	_, err := ValidateInputTypes([]string{"text", "video"})
	if err == nil {
		t.Error("无效类型 video 应返回错误")
	}
}

// ─── ValidateCustomModelID ────────────────────────────────────────────────────

func TestValidateCustomModelID_Valid(t *testing.T) {
	cases := []string{
		"gpt-4", "my_model", "model.v2", "model:latest", "a", "GPT-4-turbo",
		// P1 方案：允许 '/'，支持 anthropic/claude-3 等原始上游 ID
		"anthropic/claude-3", "openai/gpt-4", "a/b/c",
	}
	for _, id := range cases {
		if err := ValidateCustomModelID(id); err != nil {
			t.Errorf("期望合法 ID=%s，实际错误: %v", id, err)
		}
	}
}

func TestValidateCustomModelID_Empty(t *testing.T) {
	if err := ValidateCustomModelID(""); err == nil {
		t.Error("空 ID 应返回错误")
	}
}

func TestValidateCustomModelID_InvalidChars(t *testing.T) {
	cases := []string{"model id", "model;id", "model$id", "model`id", "model$(cmd)", "model'id", "model\"id", "model\nid"}
	for _, id := range cases {
		if err := ValidateCustomModelID(id); err == nil {
			t.Errorf("含危险字符的 ID=%q 应返回错误", id)
		}
	}
}

func TestValidateCustomModelID_TooLong(t *testing.T) {
	long := make([]byte, 129)
	for i := range long {
		long[i] = 'a'
	}
	if err := ValidateCustomModelID(string(long)); err == nil {
		t.Error("超长 ID (>128) 应返回错误")
	}
}

// ─── ValidateHTTPURL ──────────────────────────────────────────────────────────

func TestValidateHTTPURL_Valid(t *testing.T) {
	cases := []string{
		"http://example.com",
		"https://api.openai.com/v1",
		"https://192.168.1.1:8080/path",
	}
	for _, u := range cases {
		if err := ValidateHTTPURL(u); err != nil {
			t.Errorf("期望合法 URL=%s，实际错误: %v", u, err)
		}
	}
}

func TestValidateHTTPURL_InvalidScheme(t *testing.T) {
	cases := []string{"ftp://example.com", "ws://example.com", "//example.com"}
	for _, u := range cases {
		if err := ValidateHTTPURL(u); err == nil {
			t.Errorf("非 http/https URL=%s 应返回错误", u)
		}
	}
}

func TestValidateHTTPURL_Malformed(t *testing.T) {
	cases := []string{"", "not-a-url", "http://", "https://"}
	for _, u := range cases {
		if err := ValidateHTTPURL(u); err == nil {
			t.Errorf("格式无效 URL=%q 应返回错误", u)
		}
	}
}

// ─── ValidateAndParseCustomHTTPHeaders ──────────────────────────────────────

func TestValidateAndParseCustomHTTPHeaders_Empty(t *testing.T) {
	result, err := ValidateAndParseCustomHTTPHeaders("")
	if err != nil {
		t.Fatalf("空输入不应返回错误: %v", err)
	}
	if result != nil {
		t.Errorf("空输入应返回 nil, 实际=%v", result)
	}
}

func TestValidateAndParseCustomHTTPHeaders_WhitespaceOnly(t *testing.T) {
	result, err := ValidateAndParseCustomHTTPHeaders("   ")
	if err != nil {
		t.Fatalf("纯空白输入不应返回错误: %v", err)
	}
	if result != nil {
		t.Errorf("纯空白输入应返回 nil, 实际=%v", result)
	}
}

func TestValidateAndParseCustomHTTPHeaders_ValidSingle(t *testing.T) {
	result, err := ValidateAndParseCustomHTTPHeaders(`{"X-Custom":"value123"}`)
	if err != nil {
		t.Fatalf("合法输入不应返回错误: %v", err)
	}
	if len(result) != 1 || result["X-Custom"] != "value123" {
		t.Errorf("期望 {X-Custom:value123}, 实际=%v", result)
	}
}

func TestValidateAndParseCustomHTTPHeaders_ValidMultiple(t *testing.T) {
	result, err := ValidateAndParseCustomHTTPHeaders(`{"X-Api-Key":"abc","X-Request-Id":"req-1"}`)
	if err != nil {
		t.Fatalf("合法输入不应返回错误: %v", err)
	}
	if len(result) != 2 || result["X-Api-Key"] != "abc" || result["X-Request-Id"] != "req-1" {
		t.Errorf("期望两个头部, 实际=%v", result)
	}
}

func TestValidateAndParseCustomHTTPHeaders_EmptyJSONObject(t *testing.T) {
	result, err := ValidateAndParseCustomHTTPHeaders(`{}`)
	if err != nil {
		t.Fatalf("空 JSON 对象不应返回错误: %v", err)
	}
	if result != nil {
		t.Errorf("空 JSON 对象应返回 nil, 实际=%v", result)
	}
}

func TestValidateAndParseCustomHTTPHeaders_InvalidJSON(t *testing.T) {
	_, err := ValidateAndParseCustomHTTPHeaders(`not-json`)
	if err == nil {
		t.Error("非法 JSON 应返回错误")
	}
}

func TestValidateAndParseCustomHTTPHeaders_NonObjectJSON(t *testing.T) {
	_, err := ValidateAndParseCustomHTTPHeaders(`[1,2,3]`)
	if err == nil {
		t.Error("JSON 数组应返回错误")
	}
}

func TestValidateAndParseCustomHTTPHeaders_InvalidKeyName(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"含空格", `{"X Custom":"v"}`},
		{"含冒号", `{"X:Custom":"v"}`},
		{"含点号", `{"X.Custom":"v"}`},
		{"含中文", `{"自定义":"v"}`},
		{"含特殊字符", `{"X@Header":"v"}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := ValidateAndParseCustomHTTPHeaders(c.input)
			if err == nil {
				t.Errorf("非法键名 %q 应返回错误", c.input)
			}
		})
	}
}

func TestValidateAndParseCustomHTTPHeaders_ValidKeyName(t *testing.T) {
	cases := []struct {
		name  string
		key   string
		value string
	}{
		{"字母数字", "XApiKey123", "val"},
		{"含连字符", "X-Custom-Header", "val"},
		{"含下划线", "X_Custom_Header", "val"},
		{"全小写", "x-api-key", "val"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			input := `{"` + c.key + `":"` + c.value + `"}`
			result, err := ValidateAndParseCustomHTTPHeaders(input)
			if err != nil {
				t.Fatalf("合法键名 %q 不应返回错误: %v", c.key, err)
			}
			if result[c.key] != c.value {
				t.Errorf("期望 %s=%s, 实际=%s", c.key, c.value, result[c.key])
			}
		})
	}
}

func TestValidateAndParseCustomHTTPHeaders_ValueWithNewline(t *testing.T) {
	_, err := ValidateAndParseCustomHTTPHeaders(`{"X-Key":"line1` + "\n" + `line2"}`)
	if err == nil {
		t.Error("值含换行符应返回错误")
	}
}

func TestValidateAndParseCustomHTTPHeaders_ValueWithCarriageReturn(t *testing.T) {
	_, err := ValidateAndParseCustomHTTPHeaders(`{"X-Key":"line1` + "\r" + `line2"}`)
	if err == nil {
		t.Error("值含回车符应返回错误")
	}
}

func TestValidateAndParseCustomHTTPHeaders_TrimsWhitespace(t *testing.T) {
	result, err := ValidateAndParseCustomHTTPHeaders(`  {"X-Key":"val"}  `)
	if err != nil {
		t.Fatalf("前后空白应被 trim: %v", err)
	}
	if result["X-Key"] != "val" {
		t.Errorf("期望 X-Key=val, 实际=%s", result["X-Key"])
	}
}

func TestValidateAndParseCustomHTTPHeaders_ExceedsMaxCount(t *testing.T) {
	// 构造 11 个头部，超过 MaxCustomHTTPHeadersCount=10
	headers := make(map[string]string)
	for i := 0; i < MaxCustomHTTPHeadersCount+1; i++ {
		headers[fmt.Sprintf("X-Header-%02d", i)] = fmt.Sprintf("val%d", i)
	}
	raw, _ := json.Marshal(headers)
	_, err := ValidateAndParseCustomHTTPHeaders(string(raw))
	if err == nil {
		t.Errorf("超过 %d 个头部应返回错误", MaxCustomHTTPHeadersCount)
	}
}

func TestValidateAndParseCustomHTTPHeaders_ExactlyMaxCount(t *testing.T) {
	// 恰好 10 个头部应通过
	headers := make(map[string]string)
	for i := 0; i < MaxCustomHTTPHeadersCount; i++ {
		headers[fmt.Sprintf("X-Header-%02d", i)] = fmt.Sprintf("val%d", i)
	}
	raw, _ := json.Marshal(headers)
	result, err := ValidateAndParseCustomHTTPHeaders(string(raw))
	if err != nil {
		t.Fatalf("恰好 %d 个头部不应返回错误: %v", MaxCustomHTTPHeadersCount, err)
	}
	if len(result) != MaxCustomHTTPHeadersCount {
		t.Errorf("期望 %d 个头部, 实际=%d", MaxCustomHTTPHeadersCount, len(result))
	}
}

func TestValidateAndParseCustomHTTPHeaders_ExceedsMaxSize(t *testing.T) {
	// 构造超过 4KB 的 JSON
	headers := make(map[string]string)
	longVal := strings.Repeat("a", MaxCustomHTTPHeadersSize)
	headers["X-Key"] = longVal
	raw, _ := json.Marshal(headers)
	if len(raw) <= MaxCustomHTTPHeadersSize {
		t.Fatalf("测试数据未超过 %d 字节，无法验证", MaxCustomHTTPHeadersSize)
	}
	_, err := ValidateAndParseCustomHTTPHeaders(string(raw))
	if err == nil {
		t.Errorf("超过 %d 字节应返回错误", MaxCustomHTTPHeadersSize)
	}
}

func TestValidateAndParseCustomHTTPHeaders_JustUnderMaxSize(t *testing.T) {
	// 构造恰好等于 4KB 的 JSON
	prefix := `{"X-Key":"`
	suffix := `"}`
	valLen := MaxCustomHTTPHeadersSize - len(prefix) - len(suffix)
	raw := prefix + strings.Repeat("a", valLen) + suffix
	if len(raw) != MaxCustomHTTPHeadersSize {
		t.Fatalf("测试数据应为 %d 字节，实际=%d", MaxCustomHTTPHeadersSize, len(raw))
	}
	result, err := ValidateAndParseCustomHTTPHeaders(raw)
	if err != nil {
		t.Fatalf("恰好 %d 字节不应返回错误: %v", MaxCustomHTTPHeadersSize, err)
	}
	if result["X-Key"] != strings.Repeat("a", valLen) {
		t.Error("值不匹配")
	}
}

func TestValidateAndParseCustomHTTPHeaders_ReservedHeaderKeys(t *testing.T) {
	// 所有大写/小写/混合大小写的保留头都应被拒绝
	reserved := []string{
		"Authorization", "authorization", "AUTHORIZATION", "AuThOrIzAtIoN",
		"Host", "HOST", "host",
		"Cookie", "COOKIE", "cookie",
		"Set-Cookie", "SET-COOKIE", "set-cookie",
		"Content-Type", "content-type", "CONTENT-TYPE",
		"Content-Length", "content-length",
		"Transfer-Encoding", "transfer-encoding",
		"Connection", "connection",
		"Upgrade", "upgrade",
		"Origin", "origin",
		"Referer", "referer",
		"User-Agent", "user-agent",
		"Proxy-Authorization", "proxy-authorization",
		"Proxy-Authenticate", "proxy-authenticate",
	}
	for _, key := range reserved {
		t.Run(key, func(t *testing.T) {
			input := `{"` + key + `":"value"}`
			_, err := ValidateAndParseCustomHTTPHeaders(input)
			if err == nil {
				t.Errorf("保留头部键名 %q 应返回错误", key)
			}
		})
	}
}

func TestValidateAndParseCustomHTTPHeaders_NonReservedHeaderKeys(t *testing.T) {
	// 非保留的 X- 前缀头部应正常通过
	cases := []string{
		"X-Custom-Header",
		"X-Api-Key",
		"X-Request-Id",
		"X-Trace-Id",
	}
	for _, key := range cases {
		t.Run(key, func(t *testing.T) {
			input := `{"` + key + `":"value"}`
			result, err := ValidateAndParseCustomHTTPHeaders(input)
			if err != nil {
				t.Fatalf("非保留键名 %q 不应返回错误: %v", key, err)
			}
			if result[key] != "value" {
				t.Errorf("期望 %s=value, 实际=%s", key, result[key])
			}
		})
	}
}
