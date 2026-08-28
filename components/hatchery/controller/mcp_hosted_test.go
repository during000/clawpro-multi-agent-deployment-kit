package controller

import (
	"context"
	"encoding/json"
	"testing"

	hcommon "hatchery/common"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestExtractPlaceholders(t *testing.T) {
	tests := []struct {
		name     string
		config   string
		wantKeys []string // 字段名（header name / query param name）
	}{
		{
			name:     "header placeholder",
			config:   `{"url":"http://x.com/mcp","headers":{"Authorization":"<token>"}}`,
			wantKeys: []string{"Authorization"},
		},
		{
			name:     "url query placeholder",
			config:   `{"url":"http://x.com/mcp?api_key=<api-key>","headers":{}}`,
			wantKeys: []string{"api_key"},
		},
		{
			name:     "both header and query placeholders",
			config:   `{"url":"http://x.com/mcp?api_key=<api-key>","headers":{"Authorization":"<token>","X-Trace":"fixed"}}`,
			wantKeys: []string{"api_key", "Authorization"},
		},
		{
			name:     "no placeholders",
			config:   `{"url":"http://x.com/mcp","headers":{"Authorization":"Bearer real-token"}}`,
			wantKeys: nil,
		},
		{
			name:     "invalid json",
			config:   `not json`,
			wantKeys: nil,
		},
		{
			name:     "no headers no url",
			config:   `{"transportType":"sse"}`,
			wantKeys: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractPlaceholders(tt.config)
			if tt.wantKeys == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}
			if len(result) != len(tt.wantKeys) {
				t.Errorf("expected %d keys, got %d: %v", len(tt.wantKeys), len(result), result)
				return
			}
			for _, key := range tt.wantKeys {
				if _, ok := result[key]; !ok {
					t.Errorf("expected key %q not found in %v", key, result)
				}
			}
		})
	}
}

func TestIsPlaceholder(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"<token>", true},
		{"<api-key>", true},
		{"< spaced >", true},
		{"Bearer token", false},
		{"<>", true},
		{"", false},
		{"no-angle", false},
		{"<only-start", false},
		{"only-end>", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := IsPlaceholder(tt.input); got != tt.want {
				t.Errorf("IsPlaceholder(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestDiffConfigPlaceholders(t *testing.T) {
	original := `{"transportType":"streamable-http","url":"http://x.com/mcp?api_key=<api-key>","headers":{"Authorization":"<token>","X-Trace":"fixed"}}`

	t.Run("user fills Authorization, keeps api_key placeholder", func(t *testing.T) {
		submitted := `{"transportType":"streamable-http","url":"http://x.com/mcp?api_key=<api-key>","headers":{"Authorization":"Bearer xxx","X-Trace":"fixed"}}`
		filled, err := DiffConfigPlaceholders(original, submitted)
		if err != nil {
			t.Fatal(err)
		}
		if filled["Authorization"] != "Bearer xxx" {
			t.Errorf("expected Authorization='Bearer xxx', got %q", filled["Authorization"])
		}
		if _, ok := filled["api_key"]; ok {
			t.Errorf("api_key should not be in filled (still placeholder)")
		}
	})

	t.Run("user fills both", func(t *testing.T) {
		submitted := `{"transportType":"streamable-http","url":"http://x.com/mcp?api_key=my-key","headers":{"Authorization":"Bearer xxx","X-Trace":"fixed"}}`
		filled, err := DiffConfigPlaceholders(original, submitted)
		if err != nil {
			t.Fatal(err)
		}
		if filled["Authorization"] != "Bearer xxx" {
			t.Errorf("expected Authorization='Bearer xxx', got %q", filled["Authorization"])
		}
		if filled["api_key"] != "my-key" {
			t.Errorf("expected api_key='my-key', got %q", filled["api_key"])
		}
	})

	t.Run("reject modified non-placeholder field", func(t *testing.T) {
		submitted := `{"transportType":"streamable-http","url":"http://x.com/mcp?api_key=<api-key>","headers":{"Authorization":"<token>","X-Trace":"modified"}}`
		_, err := DiffConfigPlaceholders(original, submitted)
		if err == nil {
			t.Error("expected error for modified X-Trace")
		}
	})

	t.Run("reject modified URL path", func(t *testing.T) {
		submitted := `{"transportType":"streamable-http","url":"http://evil.com/mcp?api_key=<api-key>","headers":{"Authorization":"<token>","X-Trace":"fixed"}}`
		_, err := DiffConfigPlaceholders(original, submitted)
		if err == nil {
			t.Error("expected error for modified URL path")
		}
	})

	t.Run("reject new header", func(t *testing.T) {
		submitted := `{"transportType":"streamable-http","url":"http://x.com/mcp?api_key=<api-key>","headers":{"Authorization":"<token>","X-Trace":"fixed","X-New":"hack"}}`
		_, err := DiffConfigPlaceholders(original, submitted)
		if err == nil {
			t.Error("expected error for new header")
		}
	})

	t.Run("reject modified transportType", func(t *testing.T) {
		submitted := `{"transportType":"sse","url":"http://x.com/mcp?api_key=<api-key>","headers":{"Authorization":"<token>","X-Trace":"fixed"}}`
		_, err := DiffConfigPlaceholders(original, submitted)
		if err == nil {
			t.Error("expected error for modified transportType")
		}
	})

	t.Run("reject new field", func(t *testing.T) {
		submitted := `{"transportType":"streamable-http","url":"http://x.com/mcp?api_key=<api-key>","headers":{"Authorization":"<token>","X-Trace":"fixed"},"extra":"field"}`
		_, err := DiffConfigPlaceholders(original, submitted)
		if err == nil {
			t.Error("expected error for new field")
		}
	})

	t.Run("reject invalid original json", func(t *testing.T) {
		_, err := DiffConfigPlaceholders("not json", `{"url":"http://x.com/mcp"}`)
		if err == nil {
			t.Error("expected error for invalid original json")
		}
	})

	t.Run("reject invalid submitted json", func(t *testing.T) {
		_, err := DiffConfigPlaceholders(`{"url":"http://x.com/mcp"}`, "not json")
		if err == nil {
			t.Error("expected error for invalid submitted json")
		}
	})

	t.Run("user leaves placeholder untouched", func(t *testing.T) {
		submitted := `{"transportType":"streamable-http","url":"http://x.com/mcp?api_key=<api-key>","headers":{"Authorization":"<token>","X-Trace":"fixed"}}`
		filled, err := DiffConfigPlaceholders(original, submitted)
		if err != nil {
			t.Fatal(err)
		}
		if len(filled) != 0 {
			t.Errorf("expected empty filled values, got %v", filled)
		}
	})

	t.Run("reject non-string header value in submitted", func(t *testing.T) {
		submitted := `{"transportType":"streamable-http","url":"http://x.com/mcp?api_key=<api-key>","headers":{"Authorization":123,"X-Trace":"fixed"}}`
		_, err := DiffConfigPlaceholders(original, submitted)
		if err == nil {
			t.Error("expected error for non-string header value")
		}
	})

	t.Run("reject modified non-placeholder url param", func(t *testing.T) {
		orig := `{"url":"http://x.com/mcp?fixed=val","headers":{}}`
		sub := `{"url":"http://x.com/mcp?fixed=changed","headers":{}}`
		_, err := DiffConfigPlaceholders(orig, sub)
		if err == nil {
			t.Error("expected error for modified non-placeholder url param")
		}
	})

	t.Run("no url field", func(t *testing.T) {
		orig := `{"transportType":"sse","headers":{"Authorization":"<token>"}}`
		sub := `{"transportType":"sse","headers":{"Authorization":"Bearer xxx"}}`
		filled, err := DiffConfigPlaceholders(orig, sub)
		if err != nil {
			t.Fatal(err)
		}
		if filled["Authorization"] != "Bearer xxx" {
			t.Errorf("expected Authorization='Bearer xxx', got %q", filled["Authorization"])
		}
	})

	t.Run("no headers field", func(t *testing.T) {
		orig := `{"url":"http://x.com/mcp?key=<val>"}`
		sub := `{"url":"http://x.com/mcp?key=filled"}`
		filled, err := DiffConfigPlaceholders(orig, sub)
		if err != nil {
			t.Fatal(err)
		}
		if filled["key"] != "filled" {
			t.Errorf("expected key='filled', got %q", filled["key"])
		}
	})
}

func TestRemoveDefaultedPlaceholders(t *testing.T) {
	config := `{"url":"http://x.com/mcp?api_key=<api-key>&other=<other>","headers":{"Authorization":"<token>","X-Trace":"fixed"}}`

	t.Run("remove api_key from url query", func(t *testing.T) {
		result := removeDefaultedPlaceholders(config, map[string]bool{"api_key": true})
		placeholders := ExtractPlaceholders(result)
		if _, ok := placeholders["api_key"]; ok {
			t.Error("api_key should have been removed")
		}
		if _, ok := placeholders["Authorization"]; !ok {
			t.Error("Authorization should still be present")
		}
		if _, ok := placeholders["other"]; !ok {
			t.Error("other should still be present")
		}
	})

	t.Run("remove Authorization from headers", func(t *testing.T) {
		result := removeDefaultedPlaceholders(config, map[string]bool{"Authorization": true})
		placeholders := ExtractPlaceholders(result)
		if _, ok := placeholders["Authorization"]; ok {
			t.Error("Authorization should have been removed")
		}
		if _, ok := placeholders["api_key"]; !ok {
			t.Error("api_key should still be present")
		}
	})

	t.Run("remove multiple", func(t *testing.T) {
		result := removeDefaultedPlaceholders(config, map[string]bool{"api_key": true, "Authorization": true})
		placeholders := ExtractPlaceholders(result)
		if _, ok := placeholders["api_key"]; ok {
			t.Error("api_key should have been removed")
		}
		if _, ok := placeholders["Authorization"]; ok {
			t.Error("Authorization should have been removed")
		}
		if _, ok := placeholders["other"]; !ok {
			t.Error("other should still be present")
		}
	})

	t.Run("remove all placeholders from url query", func(t *testing.T) {
		result := removeDefaultedPlaceholders(config, map[string]bool{"api_key": true, "other": true})
		placeholders := ExtractPlaceholders(result)
		if _, ok := placeholders["api_key"]; ok {
			t.Error("api_key should have been removed")
		}
		if _, ok := placeholders["other"]; ok {
			t.Error("other should have been removed")
		}
		// url should no longer have query params
		var cfg map[string]interface{}
		json.Unmarshal([]byte(result), &cfg)
		url := cfg["url"].(string)
		if url != "http://x.com/mcp" {
			t.Errorf("expected url without query, got %q", url)
		}
	})

	t.Run("remove all placeholders from headers", func(t *testing.T) {
		result := removeDefaultedPlaceholders(config, map[string]bool{"Authorization": true})
		var cfg map[string]interface{}
		json.Unmarshal([]byte(result), &cfg)
		// X-Trace is not a placeholder so it remains, headers key should still exist
		headers, _ := cfg["headers"].(map[string]interface{})
		if _, ok := headers["X-Trace"]; !ok {
			t.Error("X-Trace should still be present")
		}
		if _, ok := headers["Authorization"]; ok {
			t.Error("Authorization should be removed")
		}
	})

	t.Run("remove all headers leaving only non-placeholder", func(t *testing.T) {
		cfg := `{"headers":{"Authorization":"<token>"}}`
		result := removeDefaultedPlaceholders(cfg, map[string]bool{"Authorization": true})
		var parsed map[string]interface{}
		json.Unmarshal([]byte(result), &parsed)
		if _, ok := parsed["headers"]; ok {
			t.Error("headers should be removed when all placeholders are removed and no non-placeholder remains")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		result := removeDefaultedPlaceholders("not json", map[string]bool{"key": true})
		if result != "not json" {
			t.Errorf("invalid json should be returned as-is, got %q", result)
		}
	})

	t.Run("no placeholders in url", func(t *testing.T) {
		cfg := `{"url":"http://x.com/mcp?fixed=val","headers":{"Authorization":"<token>"}}`
		result := removeDefaultedPlaceholders(cfg, map[string]bool{"Authorization": true})
		placeholders := ExtractPlaceholders(result)
		if _, ok := placeholders["Authorization"]; ok {
			t.Error("Authorization should have been removed")
		}
	})

	t.Run("url query param without equals", func(t *testing.T) {
		cfg := `{"url":"http://x.com/mcp?flag","headers":{}}`
		result := removeDefaultedPlaceholders(cfg, map[string]bool{})
		if result == "" {
			t.Error("should return valid result")
		}
	})
}

func TestBuildProbeConfigJSON(t *testing.T) {
	original := `{"transportType":"streamable-http","url":"http://x.com/mcp?api_key=<key>","headers":{"Authorization":"<token>","X-Trace":"fixed"}}`

	t.Run("replace all by field name", func(t *testing.T) {
		resolved := map[string]string{
			"Authorization": "Bearer real-token",
			"api_key":       "real-api-key",
		}
		result := buildProbeConfigJSON(original, resolved)
		placeholders := ExtractPlaceholders(result)
		if _, ok := placeholders["Authorization"]; ok {
			t.Error("Authorization should have been replaced")
		}
		if _, ok := placeholders["api_key"]; ok {
			t.Error("api_key should have been replaced")
		}
	})

	t.Run("partial replace", func(t *testing.T) {
		resolved := map[string]string{
			"Authorization": "Bearer real-token",
		}
		result := buildProbeConfigJSON(original, resolved)
		placeholders := ExtractPlaceholders(result)
		if _, ok := placeholders["Authorization"]; ok {
			t.Error("Authorization should have been replaced")
		}
		if _, ok := placeholders["api_key"]; !ok {
			t.Error("api_key should still be placeholder (not in resolved)")
		}
	})

	t.Run("empty resolved", func(t *testing.T) {
		result := buildProbeConfigJSON(original, nil)
		placeholders := ExtractPlaceholders(result)
		if len(placeholders) != 2 {
			t.Errorf("expected 2 placeholders, got %d", len(placeholders))
		}
	})

	t.Run("invalid json returns original", func(t *testing.T) {
		result := buildProbeConfigJSON("not json", map[string]string{"key": "val"})
		if result != "not json" {
			t.Errorf("invalid json should be returned as-is, got %q", result)
		}
	})

	t.Run("no url with query", func(t *testing.T) {
		cfg := `{"headers":{"Authorization":"<token>"}}`
		resolved := map[string]string{"Authorization": "Bearer xxx"}
		result := buildProbeConfigJSON(cfg, resolved)
		placeholders := ExtractPlaceholders(result)
		if _, ok := placeholders["Authorization"]; ok {
			t.Error("Authorization should have been replaced")
		}
	})

	t.Run("no headers", func(t *testing.T) {
		cfg := `{"url":"http://x.com/mcp?key=<val>"}`
		resolved := map[string]string{"key": "filled"}
		result := buildProbeConfigJSON(cfg, resolved)
		placeholders := ExtractPlaceholders(result)
		if _, ok := placeholders["key"]; ok {
			t.Error("key should have been replaced")
		}
	})
}

// ── 数据库相关测试辅助 ──────────────────────────────────────────────

func setupHostedTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.McpServer{},
		&model.McpVersion{},
		&model.McpHostedKey{},
		&model.McpInstallation{},
		&model.Instance{},
		&model.SiteConfig{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	db.Create(&model.SiteConfig{})
}

func TestSaveHostedKeys(t *testing.T) {
	setupHostedTestDB(t)
	ctx := context.Background()

	t.Run("save and retrieve", func(t *testing.T) {
		configJSON := `{"url":"http://x.com/mcp","headers":{"Authorization":"<token>","X-Key":"<api-key>"}}`
		hasHosted, err := SaveHostedKeys(ctx, 1, configJSON, map[string]string{"Authorization": "default-token"})
		if err != nil {
			t.Fatal(err)
		}
		if !hasHosted {
			t.Error("expected hasHosted=true")
		}
		creds := GetHostedKeys(ctx, 1)
		if len(creds) != 2 {
			t.Fatalf("expected 2 keys, got %d", len(creds))
		}
		// Check defaults
		for _, c := range creds {
			if c.Key == "Authorization" && c.DefaultValue != "default-token" {
				t.Errorf("expected default 'default-token', got %q", c.DefaultValue)
			}
		}
	})

	t.Run("no placeholders", func(t *testing.T) {
		configJSON := `{"url":"http://x.com/mcp","headers":{"Authorization":"Bearer fixed"}}`
		hasHosted, err := SaveHostedKeys(ctx, 99, configJSON, nil)
		if err != nil {
			t.Fatal(err)
		}
		if hasHosted {
			t.Error("expected hasHosted=false")
		}
	})

	t.Run("nil hostedDefaults", func(t *testing.T) {
		configJSON := `{"headers":{"Authorization":"<token>"}}`
		hasHosted, err := SaveHostedKeys(ctx, 100, configJSON, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !hasHosted {
			t.Error("expected hasHosted=true")
		}
		creds := GetHostedKeys(ctx, 100)
		for _, c := range creds {
			if c.DefaultValue != "" {
				t.Errorf("expected empty default, got %q", c.DefaultValue)
			}
		}
	})

	t.Run("upsert existing", func(t *testing.T) {
		configJSON := `{"headers":{"Authorization":"<token>"}}`
		SaveHostedKeys(ctx, 200, configJSON, map[string]string{"Authorization": "old-default"})
		SaveHostedKeys(ctx, 200, configJSON, map[string]string{"Authorization": "new-default"})
		creds := GetHostedKeys(ctx, 200)
		if len(creds) != 1 {
			t.Fatalf("expected 1 key, got %d", len(creds))
		}
		if creds[0].DefaultValue != "new-default" {
			t.Errorf("expected 'new-default', got %q", creds[0].DefaultValue)
		}
	})
}

func TestGetHostedKeys(t *testing.T) {
	setupHostedTestDB(t)
	ctx := context.Background()

	creds := GetHostedKeys(ctx, 999)
	if len(creds) != 0 {
		t.Errorf("expected 0 keys for non-existent mcp, got %d", len(creds))
	}
}

func TestResolveHostedValues(t *testing.T) {
	creds := []model.McpHostedKey{
		{Key: "Authorization", DefaultValue: "default-token"},
		{Key: "api-key", DefaultValue: ""},
		{Key: "X-Custom", DefaultValue: "custom-default"},
	}

	t.Run("user value overrides default", func(t *testing.T) {
		userVals := map[string]string{"Authorization": "Bearer user-token"}
		resolved, missing := ResolveHostedValues(creds, userVals)
		if resolved["Authorization"] != "Bearer user-token" {
			t.Errorf("expected user value, got %q", resolved["Authorization"])
		}
		if resolved["X-Custom"] != "custom-default" {
			t.Errorf("expected default, got %q", resolved["X-Custom"])
		}
		if len(missing) != 1 || missing[0] != "api-key" {
			t.Errorf("expected missing=['api-key'], got %v", missing)
		}
	})

	t.Run("all defaults used when no user values", func(t *testing.T) {
		resolved, missing := ResolveHostedValues(creds, nil)
		if resolved["Authorization"] != "default-token" {
			t.Errorf("expected default, got %q", resolved["Authorization"])
		}
		if resolved["X-Custom"] != "custom-default" {
			t.Errorf("expected default, got %q", resolved["X-Custom"])
		}
		if len(missing) != 1 || missing[0] != "api-key" {
			t.Errorf("expected missing=['api-key'], got %v", missing)
		}
	})

	t.Run("empty user value falls back to default", func(t *testing.T) {
		userVals := map[string]string{"Authorization": ""}
		resolved, _ := ResolveHostedValues(creds, userVals)
		if resolved["Authorization"] != "default-token" {
			t.Errorf("expected default fallback, got %q", resolved["Authorization"])
		}
	})

	t.Run("all missing with no defaults", func(t *testing.T) {
		noDefaultCreds := []model.McpHostedKey{
			{Key: "key1", DefaultValue: ""},
			{Key: "key2", DefaultValue: ""},
		}
		_, missing := ResolveHostedValues(noDefaultCreds, nil)
		if len(missing) != 2 {
			t.Errorf("expected 2 missing, got %d", len(missing))
		}
	})
}

func TestExtractTargetURL(t *testing.T) {
	tests := []struct {
		name   string
		config string
		want   string
	}{
		{"with url", `{"url":"http://x.com/mcp","headers":{}}`, "http://x.com/mcp"},
		{"no url", `{"headers":{}}`, ""},
		{"invalid json", "not json", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractTargetURL(tt.config); got != tt.want {
				t.Errorf("ExtractTargetURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSecurityGatewayBaseURL(t *testing.T) {
	t.Run("with domain in context", func(t *testing.T) {
		ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{Domain: "http://example.com"})
		if got := SecurityGatewayBaseURL(ctx); got != "http://example.com" {
			t.Errorf("expected 'http://example.com', got %q", got)
		}
	})

	t.Run("with trailing slash", func(t *testing.T) {
		ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{Domain: "http://example.com/"})
		if got := SecurityGatewayBaseURL(ctx); got != "http://example.com" {
			t.Errorf("expected 'http://example.com', got %q", got)
		}
	})

	t.Run("empty context", func(t *testing.T) {
		if got := SecurityGatewayBaseURL(context.Background()); got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})
}

func TestMcpGatewayProxyURL(t *testing.T) {
	t.Run("sse transport", func(t *testing.T) {
		ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{Domain: "http://gw.example.com"})
		if got := McpGatewayProxyURL(ctx, "sse", 42); got != "http://gw.example.com/clawpro/sse/42" {
			t.Errorf("unexpected URL: %q", got)
		}
	})

	t.Run("streamable-http transport", func(t *testing.T) {
		ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{Domain: "http://gw.example.com"})
		if got := McpGatewayProxyURL(ctx, "streamable-http", 42); got != "http://gw.example.com/clawpro/mcp/42" {
			t.Errorf("unexpected URL: %q", got)
		}
	})

	t.Run("empty base URL", func(t *testing.T) {
		if got := McpGatewayProxyURL(context.Background(), "sse", 42); got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})
}

func TestBuildDeployConfigJSON(t *testing.T) {
	setupHostedTestDB(t)
	ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{Domain: "http://gw.example.com"})

	t.Run("replace url and remove hosted headers, inject proxyToken", func(t *testing.T) {
		proxyToken := "test-proxy-token"
		inst := &model.Instance{ProxyToken: &proxyToken}
		configJSON := `{"transportType":"streamable-http","url":"http://real.mcp.com/api","headers":{"Authorization":"<token>","X-Trace":"fixed"}}`
		result := BuildDeployConfigJSON(ctx, configJSON, "streamable-http", 5, inst)

		var cfg map[string]interface{}
		json.Unmarshal([]byte(result), &cfg)

		// URL should be replaced
		if cfg["url"] != "http://gw.example.com/clawpro/mcp/5" {
			t.Errorf("expected proxy URL, got %v", cfg["url"])
		}

		// Headers: Authorization should be proxyToken, X-Trace should be kept, no placeholder
		headers := cfg["headers"].(map[string]interface{})
		if headers["Authorization"] != "Bearer test-proxy-token" {
			t.Errorf("expected proxyToken, got %v", headers["Authorization"])
		}
		if headers["X-Trace"] != "fixed" {
			t.Errorf("X-Trace should be kept, got %v", headers["X-Trace"])
		}
	})

	t.Run("no proxy token", func(t *testing.T) {
		inst := &model.Instance{}
		configJSON := `{"url":"http://real.mcp.com/api","headers":{"X-Custom":"<token>"}}`
		result := BuildDeployConfigJSON(ctx, configJSON, "sse", 5, inst)

		var cfg map[string]interface{}
		json.Unmarshal([]byte(result), &cfg)
		// All headers are placeholders → no headers
		if _, ok := cfg["headers"]; ok {
			t.Error("headers should be removed when all are placeholders and no proxyToken")
		}
	})

	t.Run("empty base URL returns original", func(t *testing.T) {
		inst := &model.Instance{}
		configJSON := `{"url":"http://real.mcp.com/api"}`
		result := BuildDeployConfigJSON(context.Background(), configJSON, "sse", 5, inst)
		if result != configJSON {
			t.Errorf("expected original, got %q", result)
		}
	})

	t.Run("invalid json returns original", func(t *testing.T) {
		inst := &model.Instance{}
		result := BuildDeployConfigJSON(ctx, "not json", "sse", 5, inst)
		if result != "not json" {
			t.Errorf("expected original, got %q", result)
		}
	})
}

func TestBuildDisplayConfigJSON(t *testing.T) {
	setupHostedTestDB(t)
	ctx := context.Background()

	// Create test data: server, version, hosted keys
	server := model.McpServer{ServiceID: "test-svc", KeyHosted: true}
	model.DB(ctx).Create(&server)

	version := model.McpVersion{MCPID: server.ID, ConfigJSON: `{"url":"http://real.mcp.com/api","headers":{"Authorization":"<token>"}}`}
	model.DB(ctx).Create(&version)
	server.LatestVersionID = version.ID
	model.DB(ctx).Save(&server)

	model.DB(ctx).Create(&model.McpHostedKey{MCPID: server.ID, Key: "Authorization", Placeholder: "<token>"})

	t.Run("invalid json returns as-is", func(t *testing.T) {
		result := BuildDisplayConfigJSON(ctx, "not json", &server)
		if result != "not json" {
			t.Errorf("expected original, got %q", result)
		}
	})

	t.Run("no latest version", func(t *testing.T) {
		s := model.McpServer{ServiceID: "no-version", KeyHosted: true}
		model.DB(ctx).Create(&s)
		configJSON := `{"url":"http://original.com","headers":{"Authorization":"Bearer token"}}`
		result := BuildDisplayConfigJSON(ctx, configJSON, &s)

		var cfg map[string]interface{}
		json.Unmarshal([]byte(result), &cfg)
		// URL should not change (no version to look up)
		if cfg["url"] != "http://original.com" {
			t.Errorf("URL should stay the same, got %v", cfg["url"])
		}
	})

	t.Run("URL query placeholders should not be added to headers", func(t *testing.T) {
		s := model.McpServer{ServiceID: "query-test", KeyHosted: true}
		model.DB(ctx).Create(&s)

		v := model.McpVersion{MCPID: s.ID, ConfigJSON: `{"url":"http://real.mcp.com/api?key1=<>&key2=<>","headers":{"X-Custom":"<>"}}`}
		model.DB(ctx).Create(&v)
		s.LatestVersionID = v.ID
		model.DB(ctx).Save(&s)

		model.DB(ctx).Create(&model.McpHostedKey{MCPID: s.ID, Key: "key1", Placeholder: "<>"})
		model.DB(ctx).Create(&model.McpHostedKey{MCPID: s.ID, Key: "key2", Placeholder: "<>"})
		model.DB(ctx).Create(&model.McpHostedKey{MCPID: s.ID, Key: "X-Custom", Placeholder: "<>"})

		configJSON := `{"url":"http://gw.example.com/clawpro/mcp/1","headers":{"X-Custom":"real-value"}}`
		result := BuildDisplayConfigJSON(ctx, configJSON, &s)

		var cfg map[string]interface{}
		json.Unmarshal([]byte(result), &cfg)

		headers, _ := cfg["headers"].(map[string]interface{})
		// key1/key2 are URL query params, should NOT appear in headers
		if _, ok := headers["key1"]; ok {
			t.Error("key1 should not be in headers (it's a URL query param)")
		}
		if _, ok := headers["key2"]; ok {
			t.Error("key2 should not be in headers (it's a URL query param)")
		}
		// X-Custom is a version header, placeholder should be restored
		if headers["X-Custom"] != "<>" {
			t.Errorf("X-Custom placeholder should be restored, got %v", headers["X-Custom"])
		}
	})
}

func TestSplitURL(t *testing.T) {
	tests := []struct {
		input    string
		wantPath string
		wantQry  string
	}{
		{"http://x.com/mcp?key=val", "http://x.com/mcp", "key=val"},
		{"http://x.com/mcp", "http://x.com/mcp", ""},
		{"", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			p, q := splitURL(tt.input)
			if p != tt.wantPath || q != tt.wantQry {
				t.Errorf("splitURL(%q) = (%q, %q), want (%q, %q)", tt.input, p, q, tt.wantPath, tt.wantQry)
			}
		})
	}
}

func TestParseQueryParams(t *testing.T) {
	t.Run("empty query", func(t *testing.T) {
		if got := parseQueryParams(""); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("single param", func(t *testing.T) {
		params := parseQueryParams("key=val")
		if len(params) != 1 || params[0].key != "key" || params[0].value != "val" {
			t.Errorf("unexpected params: %v", params)
		}
	})

	t.Run("multiple params", func(t *testing.T) {
		params := parseQueryParams("k1=v1&k2=v2")
		if len(params) != 2 {
			t.Errorf("expected 2 params, got %d", len(params))
		}
	})

	t.Run("param without value", func(t *testing.T) {
		params := parseQueryParams("flag")
		if len(params) != 1 || params[0].key != "flag" || params[0].value != "" {
			t.Errorf("unexpected params: %v", params)
		}
	})
}

func TestFindQueryParam(t *testing.T) {
	params := []queryParam{{key: "k1", value: "v1"}, {key: "k2", value: "v2"}}
	if got := findQueryParam(params, "k1"); got != "v1" {
		t.Errorf("expected 'v1', got %q", got)
	}
	if got := findQueryParam(params, "missing"); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}
