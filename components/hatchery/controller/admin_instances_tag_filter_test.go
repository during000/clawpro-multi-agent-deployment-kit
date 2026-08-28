package controller

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"testing"

	hcommon "hatchery/common"
)

// TestParseAdminInstancesTagFilters_TagKeysHappyPath 多个 tag_keys 被解析为 OR 集合。
func TestParseAdminInstancesTagFilters_TagKeysHappyPath(t *testing.T) {
	q := url.Values{}
	q.Set("tag_keys", "env, team ,owner")
	var f adminQueryFilter
	if err := parseAdminInstancesTagFilters(q, &f); err != nil {
		t.Fatalf("不期望报错: %v", err)
	}
	if !equalStrings(f.TagKeys, []string{"env", "team", "owner"}) {
		t.Errorf("TagKeys 期望 [env team owner], 实际 %v", f.TagKeys)
	}
	if f.TagKey != "" || len(f.TagValues) != 0 {
		t.Errorf("不应填充 TagKey/TagValues: key=%q values=%v", f.TagKey, f.TagValues)
	}
}

// TestParseAdminInstancesTagFilters_TagKeysDedup 同名 key 去重。
func TestParseAdminInstancesTagFilters_TagKeysDedup(t *testing.T) {
	q := url.Values{}
	q.Set("tag_keys", "env,env, team ,team")
	var f adminQueryFilter
	if err := parseAdminInstancesTagFilters(q, &f); err != nil {
		t.Fatalf("不期望报错: %v", err)
	}
	if !equalStrings(f.TagKeys, []string{"env", "team"}) {
		t.Errorf("TagKeys 期望去重 [env team], 实际 %v", f.TagKeys)
	}
}

// TestParseAdminInstancesTagFilters_TagKeyValuesHappyPath tag_key + tag_values 正确解析。
func TestParseAdminInstancesTagFilters_TagKeyValuesHappyPath(t *testing.T) {
	q := url.Values{}
	q.Set("tag_key", "env")
	q.Set("tag_values", "prod, staging")
	var f adminQueryFilter
	if err := parseAdminInstancesTagFilters(q, &f); err != nil {
		t.Fatalf("不期望报错: %v", err)
	}
	if f.TagKey != "env" {
		t.Errorf("TagKey 期望 env, 实际 %q", f.TagKey)
	}
	if !equalStrings(f.TagValues, []string{"prod", "staging"}) {
		t.Errorf("TagValues 期望 [prod staging], 实际 %v", f.TagValues)
	}
}

// TestParseAdminInstancesTagFilters_BothPresent 同时传两组时,优先 tag_key + tag_values,忽略 tag_keys。
func TestParseAdminInstancesTagFilters_BothPresent(t *testing.T) {
	q := url.Values{}
	q.Set("tag_keys", "owner,team")
	q.Set("tag_key", "env")
	q.Set("tag_values", "prod")
	var f adminQueryFilter
	if err := parseAdminInstancesTagFilters(q, &f); err != nil {
		t.Fatalf("不期望报错: %v", err)
	}
	if f.TagKey != "env" || !equalStrings(f.TagValues, []string{"prod"}) {
		t.Errorf("应使用 tag_key=env tag_values=[prod],实际 key=%q values=%v", f.TagKey, f.TagValues)
	}
	if len(f.TagKeys) != 0 {
		t.Errorf("tag_keys 应被忽略,实际 %v", f.TagKeys)
	}
}

// TestParseAdminInstancesTagFilters_TagKeyOnlyNoValues 只传 tag_key 不传 tag_values 时不生效(避免误匹配 key 全集)。
func TestParseAdminInstancesTagFilters_TagKeyOnlyNoValues(t *testing.T) {
	q := url.Values{}
	q.Set("tag_key", "env")
	var f adminQueryFilter
	if err := parseAdminInstancesTagFilters(q, &f); err != nil {
		t.Fatalf("不期望报错: %v", err)
	}
	if f.TagKey != "" || len(f.TagValues) != 0 {
		t.Errorf("缺 tag_values 应不填充: key=%q values=%v", f.TagKey, f.TagValues)
	}
}

// TestParseAdminInstancesTagFilters_TagValuesTooMany 超限报错。
func TestParseAdminInstancesTagFilters_TagValuesTooMany(t *testing.T) {
	parts := make([]string, 0, adminInstancesQueryMaxTagValues+1)
	for i := 0; i <= adminInstancesQueryMaxTagValues; i++ {
		parts = append(parts, "v"+strconv.Itoa(i))
	}
	q := url.Values{}
	q.Set("tag_key", "env")
	q.Set("tag_values", strings.Join(parts, ","))
	var f adminQueryFilter
	err := parseAdminInstancesTagFilters(q, &f)
	if err == nil || !strings.Contains(hcommon.ErrorMessageWithCtx(context.Background(), err), "tag_values") {
		t.Errorf("期望 tag_values 超限错误,实际: %v", err)
	}
}

// TestParseAdminInstancesTagFilters_Empty 空入参不报错也不填充。
func TestParseAdminInstancesTagFilters_Empty(t *testing.T) {
	q := url.Values{}
	var f adminQueryFilter
	if err := parseAdminInstancesTagFilters(q, &f); err != nil {
		t.Fatalf("不期望报错: %v", err)
	}
	if f.hasTagFilter() {
		t.Errorf("空参数不应启用 tag filter")
	}
}

// TestMatchTagFilter_NoFilter 未启用过滤时一律放行。
func TestMatchTagFilter_NoFilter(t *testing.T) {
	var f adminQueryFilter
	if !f.matchTagFilter(nil) {
		t.Errorf("未启用过滤,nil 也应放行")
	}
	if !f.matchTagFilter(&CVMInstanceInfo{Tags: []CVMTag{{Key: "x", Value: "y"}}}) {
		t.Errorf("未启用过滤应放行任意实例")
	}
}

// TestMatchTagFilter_NilOrAPIError 启用过滤后,nil 或 API_ERROR 实例被排除。
func TestMatchTagFilter_NilOrAPIError(t *testing.T) {
	f := adminQueryFilter{TagKeys: []string{"env"}}
	if f.matchTagFilter(nil) {
		t.Errorf("nil cvmInfo 应被排除")
	}
	if f.matchTagFilter(&CVMInstanceInfo{State: "API_ERROR"}) {
		t.Errorf("API_ERROR 应被排除")
	}
}

// TestMatchTagFilter_TagKeysOR 多键 OR 命中。
func TestMatchTagFilter_TagKeysOR(t *testing.T) {
	f := adminQueryFilter{TagKeys: []string{"env", "owner"}}
	hit := &CVMInstanceInfo{Tags: []CVMTag{{Key: "team", Value: "infra"}, {Key: "owner", Value: "alice"}}}
	if !f.matchTagFilter(hit) {
		t.Errorf("应命中 owner")
	}
	miss := &CVMInstanceInfo{Tags: []CVMTag{{Key: "team", Value: "infra"}}}
	if f.matchTagFilter(miss) {
		t.Errorf("无 env/owner 不应命中")
	}
}

// TestMatchTagFilter_TagKeyValuesOR 单键多值 OR 命中。
func TestMatchTagFilter_TagKeyValuesOR(t *testing.T) {
	f := adminQueryFilter{TagKey: "env", TagValues: []string{"prod", "staging"}}
	if !f.matchTagFilter(&CVMInstanceInfo{Tags: []CVMTag{{Key: "env", Value: "staging"}}}) {
		t.Errorf("env=staging 应命中")
	}
	if f.matchTagFilter(&CVMInstanceInfo{Tags: []CVMTag{{Key: "env", Value: "dev"}}}) {
		t.Errorf("env=dev 不在候选值中,不应命中")
	}
	if f.matchTagFilter(&CVMInstanceInfo{Tags: []CVMTag{{Key: "team", Value: "prod"}}}) {
		t.Errorf("不同 key 不应命中")
	}
}

// ── escapeSQLLike 测试 ──

func TestEscapeSQLLike(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"normal", "normal"},
		{"has%percent", "has\\%percent"},
		{"has_underscore", "has\\_underscore"},
		{"nospace", "nospace"},
		{"just_letters", "just\\_letters"},
	}
	for _, tt := range tests {
		got := escapeSQLLike(tt.input)
		if got != tt.want {
			t.Errorf("escapeSQLLike(%q)=%q, want %q", tt.input, got, tt.want)
		}
	}
}

// ── parseStatusFilterValues 测试 ──

func TestParseStatusFilterValues(t *testing.T) {
	tests := []struct {
		filter string
		want   []string
	}{
		{"running", []string{"running"}},
		{"running,stopped", []string{"running", "stopped"}},
		{"", nil},
		{",,", nil},
	}
	for _, tt := range tests {
		got := parseStatusFilterValues(tt.filter)
		if !equalStrings(got, tt.want) {
			t.Errorf("parseStatusFilterValues(%q)=%v, want %v", tt.filter, got, tt.want)
		}
	}
}

func TestParseStatusFilterValues_Other(t *testing.T) {
	got := parseStatusFilterValues("other")
	if len(got) < 6 {
		t.Errorf("other 应展开多个状态, 实际长度=%d: %v", len(got), got)
	}
}
