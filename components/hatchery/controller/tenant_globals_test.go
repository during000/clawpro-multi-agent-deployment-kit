package controller

import (
	"context"
	"testing"

	"hatchery/common"
)

func TestCVMUinFromCtx_ReturnsSnapshotUin(t *testing.T) {
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Uin: "snap-uin"})
	if got := common.CVMUinFromCtx(ctx); got != "snap-uin" {
		t.Fatalf("want snap-uin, got %q", got)
	}
}

func TestCVMUinFromCtx_ReturnsEmptyWithoutSnapshot(t *testing.T) {
	oldSnap := common.FixedSnapshot
	common.FixedSnapshot = nil
	defer func() { common.FixedSnapshot = oldSnap }()

	if got := common.CVMUinFromCtx(context.Background()); got != "" {
		t.Fatalf("want empty, got %q", got)
	}
}

func TestDomainFromCtx_ReturnsSnapshotDomain(t *testing.T) {
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Domain: "snap.example.com"})
	if got := common.DomainFromCtx(ctx); got != "snap.example.com" {
		t.Fatalf("want snap.example.com, got %q", got)
	}
}

func TestDomainFromCtx_ReturnsEmptyWithoutSnapshot(t *testing.T) {
	oldSnap := common.FixedSnapshot
	common.FixedSnapshot = nil
	defer func() { common.FixedSnapshot = oldSnap }()

	if got := common.DomainFromCtx(context.Background()); got != "" {
		t.Fatalf("want empty, got %q", got)
	}
}

func TestInternalSecretFromCtx(t *testing.T) {
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{InternalSecret: "snap-secret"})
	if got := common.InternalSecretFromCtx(ctx); got != "snap-secret" {
		t.Fatalf("want snap-secret, got %q", got)
	}

	oldSnap := common.FixedSnapshot
	common.FixedSnapshot = nil
	defer func() { common.FixedSnapshot = oldSnap }()

	if got := common.InternalSecretFromCtx(context.Background()); got != "" {
		t.Fatalf("want empty, got %q", got)
	}
}

func TestTenantIDFromCtx(t *testing.T) {
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{OneIDAccountID: "snap-tid"})
	if got := common.TenantIDFromCtx(ctx); got != "snap-tid" {
		t.Fatalf("want snap-tid, got %q", got)
	}

	oldSnap := common.FixedSnapshot
	common.FixedSnapshot = nil
	defer func() { common.FixedSnapshot = oldSnap }()

	if got := common.TenantIDFromCtx(context.Background()); got != "" {
		t.Fatalf("want empty, got %q", got)
	}
}
