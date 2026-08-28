package controller

import (
	"regexp"
	"testing"
)

// TestMergeFallbackCredentials 验证自定义通道凭证兜底逻辑：
// 用户提交但 server 模板未声明占位符的 key=value 会被补到最终 config 顶层，
// 模板已用占位符渲染的 key 跳过（避免重复/覆盖）。
func TestMergeFallbackCredentials(t *testing.T) {
	re := regexp.MustCompile(`\{\{\s*([^{}\s]+)\s*\}\}`)
	extract := func(tpl string) [][]string {
		return re.FindAllStringSubmatch(tpl, -1)
	}

	t.Run("form模式缺失凭证被兜底", func(t *testing.T) {
		cfg := map[string]interface{}{"enabled": true, "serverUrl": "x", "wsUrl": "y"}
		keys := []string{"appId", "uin"}
		values := []string{"appld", "23456"}
		ph := extract(`{"serverUrl":"x","wsUrl":"y"}`)
		mergeFallbackCredentials(cfg, keys, values, ph)
		if cfg["appId"] != "appld" {
			t.Fatalf("appId 期望 appld, 实际 %v", cfg["appId"])
		}
		if cfg["uin"] != "23456" {
			t.Fatalf("uin 期望 23456, 实际 %v", cfg["uin"])
		}
	})

	t.Run("模板已用占位符渲染的key跳过兜底", func(t *testing.T) {
		cfg := map[string]interface{}{"enabled": true, "appId": "renderedVal"}
		keys := []string{"appId"}
		values := []string{"userVal"}
		ph := extract(`{"appId":"{{appId}}"}`)
		mergeFallbackCredentials(cfg, keys, values, ph)
		// 模板已用占位符渲染，期望保留渲染结果，不被覆盖。
		if cfg["appId"] != "renderedVal" {
			t.Fatalf("appId 期望保留 renderedVal, 实际 %v", cfg["appId"])
		}
	})

	t.Run("嵌套占位符已渲染则不兜底到顶层", func(t *testing.T) {
		cfg := map[string]interface{}{
			"enabled": true,
			"accounts": map[string]interface{}{
				"default": map[string]interface{}{"appId": "nestedVal"},
			},
		}
		keys := []string{"appId"}
		values := []string{"userVal"}
		ph := extract(`{"accounts":{"default":{"appId":"{{appId}}"}}}`)
		mergeFallbackCredentials(cfg, keys, values, ph)
		if _, ok := cfg["appId"]; ok {
			t.Fatalf("嵌套占位符场景不应在顶层兜底 appId, 实际 %v", cfg["appId"])
		}
	})

	t.Run("json模式写死值且用户未提交同名则不新增", func(t *testing.T) {
		cfg := map[string]interface{}{"enabled": true, "serverUrl": "x", "appId": "fixedByAdmin"}
		keys := []string{"uin"}
		values := []string{"23456"}
		ph := extract(`{"serverUrl":"x","appId":"fixedByAdmin"}`)
		mergeFallbackCredentials(cfg, keys, values, ph)
		if cfg["uin"] != "23456" {
			t.Fatalf("uin 期望 23456, 实际 %v", cfg["uin"])
		}
		if cfg["appId"] != "fixedByAdmin" {
			t.Fatalf("appId 期望保留 fixedByAdmin, 实际 %v", cfg["appId"])
		}
	})

	t.Run("用户提交值与模板静态字段同名则覆盖为提交值", func(t *testing.T) {
		cfg := map[string]interface{}{"enabled": true, "serverUrl": "x", "appId": "adminFixed"}
		keys := []string{"appId"}
		values := []string{"userVal"}
		ph := extract(`{"serverUrl":"x","appId":"adminFixed"}`)
		mergeFallbackCredentials(cfg, keys, values, ph)
		if cfg["appId"] != "userVal" {
			t.Fatalf("appId 期望 userVal(覆盖静态字段), 实际 %v", cfg["appId"])
		}
	})

	t.Run("系统保留字段不被兜底覆盖", func(t *testing.T) {
		cfg := map[string]interface{}{"enabled": true, "serverUrl": "x"}
		keys := []string{"enabled"}
		values := []string{"false"}
		ph := extract(`{"serverUrl":"x"}`)
		mergeFallbackCredentials(cfg, keys, values, ph)
		if cfg["enabled"] != true {
			t.Fatalf("enabled 是保留字段不应被覆盖, 期望 true, 实际 %v", cfg["enabled"])
		}
	})
}
