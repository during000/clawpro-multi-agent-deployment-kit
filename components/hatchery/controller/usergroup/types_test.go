package usergroup

import (
	"testing"
)

func TestIsValidConfigType(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"channel", true},
		{"plugin_bundle", true},
		{"mcp", true},
		{"image_type", true},
		{"policy", true},
		{"invalid", false},
		{"", false},
	}
	for _, tt := range tests {
		got := IsValidConfigType(tt.input)
		if got != tt.want {
			t.Errorf("IsValidConfigType(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestIsValidPolicyKey(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"token_quota_day", true},
		{"instance_quota", true},
		{"agent_terminal", true},
		{"browser_vnc", true},
		{"nonexistent_key", false},
		{"", false},
	}
	for _, tt := range tests {
		got := IsValidPolicyKey(tt.input)
		if got != tt.want {
			t.Errorf("IsValidPolicyKey(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
