package usergroup

import (
	"context"
	"testing"
)

func TestResolvePolicyRaw_EmptyAncestors(t *testing.T) {
	raw, source, err := resolvePolicyRaw(context.Background(), "token_quota_day", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if raw != "" {
		t.Errorf("expected empty raw, got %q", raw)
	}
	if source.Type != SourceSiteDefault {
		t.Errorf("expected source type %q, got %q", SourceSiteDefault, source.Type)
	}
}

func TestResolvePolicyRaw_EmptyAncestorSlice(t *testing.T) {
	raw, source, err := resolvePolicyRaw(context.Background(), "agent_terminal", []uint{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if raw != "" {
		t.Errorf("expected empty raw, got %q", raw)
	}
	if source.Type != SourceSiteDefault {
		t.Errorf("expected source type %q, got %q", SourceSiteDefault, source.Type)
	}
}
