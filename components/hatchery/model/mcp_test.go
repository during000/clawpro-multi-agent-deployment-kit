package model

import "testing"

func TestCompareSemver(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want int
	}{
		{"a < b", "1.0.0", "1.0.1", -1},
		{"a > b", "2.0.0", "1.9.9", 1},
		{"相等", "1.2.3", "1.2.3", 0},
		{"major 不同", "2.0.0", "1.0.0", 1},
		{"minor 不同", "1.2.0", "1.1.0", 1},
		{"patch 不同", "1.0.2", "1.0.1", 1},
		{"空串 vs 空串", "", "", 0},
		{"空串 vs 有值", "", "1.0.0", -1},
		{"有值 vs 空串", "1.0.0", "", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CompareSemver(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("CompareSemver(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
