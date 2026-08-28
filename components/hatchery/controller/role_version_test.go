package controller

import (
	"testing"
)

func TestCompareRoleVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.0", "1.0", 0},
		{"2.0", "1.0", 1},
		{"1.0", "2.0", -1},
		{"1.10", "1.2", 1},
		{"1.0", "", 1},
		{"", "1.0", -1},
		{"", "", 0},
		{"invalid", "1.0", -1},
	}
	for _, tt := range tests {
		got := compareRoleVersions(tt.a, tt.b)
		if (tt.want == 0 && got != 0) || (tt.want > 0 && got <= 0) || (tt.want < 0 && got >= 0) {
			t.Errorf("compareRoleVersions(%q,%q)=%d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestParseRoleVersion(t *testing.T) {
	tests := []struct {
		input       string
		wantMajor   int
		wantMinor   int
	}{
		{"1.0", 1, 0},
		{"2.5", 2, 5},
		{"", -1, -1},
		{"invalid", 0, 0},
		{"1", 0, 0},
		{"1.0.0", 0, 0},
		{"v1.0", 0, 0},
	}
	for _, tt := range tests {
		major, minor := parseRoleVersion(tt.input)
		if major != tt.wantMajor || minor != tt.wantMinor {
			t.Errorf("parseRoleVersion(%q)=(%d,%d), want (%d,%d)", tt.input, major, minor, tt.wantMajor, tt.wantMinor)
		}
	}
}

func TestValidateRoleVersionFormat(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"1.0", false},
		{"2.5", false},
		{"", true},
		{"1", true},
		{"1.0.0", true},
		{"v1.0", true},
		{"abc", true},
		{"1.0a", true},
	}
	for _, tt := range tests {
		err := validateRoleVersionFormat(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("validateRoleVersionFormat(%q) err=%v, wantErr=%v", tt.input, err, tt.wantErr)
		}
	}
}
