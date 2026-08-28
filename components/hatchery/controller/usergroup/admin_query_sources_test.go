package usergroup

import (
	"testing"
)

// TestNormalizeSources 覆盖 /admin/user-groups/tree 的 sources 白名单清洗。
// 文档规定：未知值应被过滤；合法值为 manual / oneid_dept；去重；输入空或清洗后为空返回 nil（表示"不加过滤"）。
func TestNormalizeSources(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{"Nil", nil, nil},
		{"Empty", []string{}, nil},
		{"AllWhitespace", []string{"   ", "\t"}, nil},
		{"AllUnknown", []string{"oneid_group", "foo", "bar"}, nil},
		{"Single", []string{"manual"}, []string{"manual"}},
		{"Two", []string{"manual", "oneid_dept"}, []string{"manual", "oneid_dept"}},
		{"PreservesOrder", []string{"oneid_dept", "manual"}, []string{"oneid_dept", "manual"}},
		{"TrimsWhitespace", []string{" manual ", " oneid_dept "}, []string{"manual", "oneid_dept"}},
		{"LowerCase", []string{"MANUAL", "OneID_Dept"}, []string{"manual", "oneid_dept"}},
		{"Dedup", []string{"manual", "manual", "oneid_dept"}, []string{"manual", "oneid_dept"}},
		{"MixValidAndUnknown", []string{"manual", "oneid_group", "bogus", "oneid_dept"}, []string{"manual", "oneid_dept"}},
		{"DropsEmpty", []string{"", "manual", "", "oneid_dept"}, []string{"manual", "oneid_dept"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeSources(tc.in)
			if !equalStr(got, tc.want) {
				t.Fatalf("normalizeSources(%#v) = %#v, want %#v", tc.in, got, tc.want)
			}
		})
	}
}

func equalStr(a, b []string) bool {
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
