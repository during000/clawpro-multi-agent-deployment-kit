package usergroup

import (
	"fmt"
	"testing"
)

func TestParseUint(t *testing.T) {
	tests := []struct {
		input   string
		wantVal uint
		wantErr bool
	}{
		{"123", 123, false},
		{"0", 0, false},
		{"99999", 99999, false},
		{"abc", 0, true},
		{"", 0, true},
	}
	for _, tt := range tests {
		var val uint
		_, err := parseUintStr(tt.input, &val)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseUintStr(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && val != tt.wantVal {
			t.Errorf("parseUintStr(%q) = %d, want %d", tt.input, val, tt.wantVal)
		}
	}
}

func TestUintToString(t *testing.T) {
	tests := []struct {
		input uint
		want  string
	}{
		{0, "0"},
		{1, "1"},
		{12345, "12345"},
	}
	for _, tt := range tests {
		got := fmt.Sprintf("%d", tt.input)
		if got != tt.want {
			t.Errorf("fmt.Sprintf(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
