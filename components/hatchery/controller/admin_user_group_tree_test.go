package controller

import (
	"net/http/httptest"
	"testing"
)

// TestParseBoolQueryDefault 覆盖 with_user_counts 默认值语义的核心单元。
// 文档 API-user-group-foundation.md §1 规定：with_user_counts 不传时默认为 true。
// parseBoolQueryDefault 必须在"不传"时返回传入的默认值，避免再次出现原 parseBoolQuery
// 始终返回 false 导致默认计数全为 0 的问题。
func TestParseBoolQueryDefault(t *testing.T) {
	cases := []struct {
		name       string
		query      string
		defaultVal bool
		want       bool
	}{
		{"NotProvided_DefaultTrue", "", true, true},
		{"NotProvided_DefaultFalse", "", false, false},
		{"ExplicitTrue_DefaultFalse", "?with_user_counts=true", false, true},
		{"ExplicitFalse_DefaultTrue", "?with_user_counts=false", true, false},
		{"Explicit1_DefaultFalse", "?with_user_counts=1", false, true},
		{"Explicit0_DefaultTrue", "?with_user_counts=0", true, false},
		{"ExplicitYes_DefaultFalse", "?with_user_counts=yes", false, true},
		{"ExplicitNo_DefaultTrue", "?with_user_counts=no", true, false},
		{"ExplicitOn_DefaultFalse", "?with_user_counts=on", false, true},
		{"ExplicitOff_DefaultTrue", "?with_user_counts=off", true, false},
		{"EmptyString_DefaultTrue", "?with_user_counts=", true, true},
		{"EmptyString_DefaultFalse", "?with_user_counts=", false, false},
		{"WhitespaceOnly_DefaultTrue", "?with_user_counts=%20%20", true, true},
		{"UnknownValue_FallbackToDefaultTrue", "?with_user_counts=maybe", true, true},
		{"UnknownValue_FallbackToDefaultFalse", "?with_user_counts=maybe", false, false},
		{"CaseInsensitiveTrue", "?with_user_counts=TRUE", false, true},
		{"CaseInsensitiveFalse", "?with_user_counts=FALSE", true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/admin/user-groups/tree"+tc.query, nil)
			got := parseBoolQueryDefault(r, "with_user_counts", tc.defaultVal)
			if got != tc.want {
				t.Fatalf("parseBoolQueryDefault(%q, default=%v) = %v, want %v",
					tc.query, tc.defaultVal, got, tc.want)
			}
		})
	}
}

// TestParseBoolQuery 保留原函数"未传=false"的语义（供 include_descendants 等调用者使用）。
// 这是回归测试：确保拆分新辅助后不影响原实现。
func TestParseBoolQuery(t *testing.T) {
	cases := []struct {
		name  string
		query string
		want  bool
	}{
		{"NotProvided", "", false},
		{"ExplicitTrue", "?include_descendants=true", true},
		{"ExplicitFalse", "?include_descendants=false", false},
		{"Explicit1", "?include_descendants=1", true},
		{"Explicit0", "?include_descendants=0", false},
		{"UnknownValue", "?include_descendants=maybe", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/test"+tc.query, nil)
			got := parseBoolQuery(r, "include_descendants")
			if got != tc.want {
				t.Fatalf("parseBoolQuery(%q) = %v, want %v", tc.query, got, tc.want)
			}
		})
	}
}

// TestParseCSVQuery 覆盖 /admin/user-groups/tree 的 sources 参数解析。
// 文档规定 sources 为"英文逗号分隔的 source 白名单"，默认（不传）返回全部来源。
func TestParseCSVQuery(t *testing.T) {
	cases := []struct {
		name  string
		query string
		want  []string
	}{
		{"NotProvided", "", nil},
		{"EmptyValue", "?sources=", nil},
		{"WhitespaceOnly", "?sources=%20%20%20", nil},
		{"CommasOnly", "?sources=,,,", nil},
		{"Single", "?sources=manual", []string{"manual"}},
		{"Multiple", "?sources=manual,oneid_dept", []string{"manual", "oneid_dept"}},
		{"TrimsInnerSpaces", "?sources=%20manual%20,%20oneid_dept%20", []string{"manual", "oneid_dept"}},
		{"DropsEmptySegments", "?sources=manual,,oneid_dept,", []string{"manual", "oneid_dept"}},
		{"PreservesOrder", "?sources=oneid_dept,manual", []string{"oneid_dept", "manual"}},
		// parseCSVQuery 只做"切分 + Trim"，不负责合法性（交给下游 normalizeSources）
		{"PassThroughUnknown", "?sources=foo,bar", []string{"foo", "bar"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/admin/user-groups/tree"+tc.query, nil)
			got := parseCSVQuery(r, "sources")
			if !equalStringSlice(got, tc.want) {
				t.Fatalf("parseCSVQuery(%q) = %#v, want %#v", tc.query, got, tc.want)
			}
		})
	}
}

func equalStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
