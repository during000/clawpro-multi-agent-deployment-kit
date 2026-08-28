package controller

import "testing"

func TestNormalizePagingParams(t *testing.T) {
	cases := []struct {
		name                string
		offsetIn, limitIn   string
		offsetOut, limitOut string
	}{
		// ── 默认值 ──
		{"both empty → defaults", "", "", "0", "20"},
		{"offset empty, limit ok", "", "50", "0", "50"},
		{"limit empty, offset ok", "10", "", "10", "20"},

		// ── 合法区间 ──
		{"typical ok", "30", "25", "30", "25"},
		{"offset zero explicit", "0", "20", "0", "20"},
		{"limit at upper bound 100", "0", "100", "0", "100"},
		{"limit at lower bound 1", "0", "1", "0", "1"},

		// ── offset 非法 ──
		{"offset negative → 0", "-5", "20", "0", "20"},
		{"offset garbage → 0", "abc", "20", "0", "20"},
		{"offset float → 0", "1.5", "20", "0", "20"},

		// ── limit 非法 / 越界 ──
		{"limit zero → 20", "0", "0", "0", "20"},
		{"limit negative → 20", "0", "-3", "0", "20"},
		{"limit garbage → 20", "0", "xyz", "0", "20"},
		{"limit over max → 100", "0", "500", "0", "100"},
		{"limit exactly over 100 → 100", "0", "101", "0", "100"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotOffset, gotLimit := normalizePagingParams(c.offsetIn, c.limitIn)
			if gotOffset != c.offsetOut || gotLimit != c.limitOut {
				t.Errorf("normalizePagingParams(%q,%q) = (%q,%q), want (%q,%q)",
					c.offsetIn, c.limitIn, gotOffset, gotLimit, c.offsetOut, c.limitOut)
			}
		})
	}
}
