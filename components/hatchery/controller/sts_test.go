package controller

import (
	"context"
	"testing"

	"hatchery/common"
)

func TestIsValidInstanceId(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{"ins-abcde12345", true},
		{"ins-ABCDE", true},
		{"ins-a1b2c3d4", true},
		{"ins-abc", false},           // too short
		{"ins-", false},              // empty suffix
		{"", false},                  // empty
		{"i-abcde12345", false},      // wrong prefix
		{"ins-abcde!@#$", false},     // special chars
		{"ins-abcde12345'; DROP", false}, // injection attempt
	}
	for _, tt := range tests {
		got := isValidInstanceId(tt.id)
		if got != tt.want {
			t.Errorf("isValidInstanceId(%q) = %v, want %v",
				tt.id, got, tt.want)
		}
	}
}

func TestRequestInstanceScopedSTS_InvalidInstanceId(t *testing.T) {
	setupMemoryProDB(t)
	ctx := common.InjectTenant(context.Background(),
		common.TenantSnapshot{Identifier: "x"})

	_, err := RequestInstanceScopedSTS(ctx, "'; DROP TABLE")
	if err == nil {
		t.Fatal("expected error for invalid instanceId")
	}
}

func TestRequestInstanceScopedSTS_NoCredential(t *testing.T) {
	setupMemoryProDB(t)
	ctx := common.InjectTenant(context.Background(),
		common.TenantSnapshot{Identifier: "x"})

	_, err := RequestInstanceScopedSTS(ctx, "ins-validtest01")
	if err == nil {
		t.Fatal("expected error for no credential, got nil")
	}
}
