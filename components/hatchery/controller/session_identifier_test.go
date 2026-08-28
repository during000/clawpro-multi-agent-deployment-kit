package controller

import (
	"context"
	"testing"

	"github.com/gorilla/sessions"

	"hatchery/common"
)

// ─── setSessionIdentifier ─────────────────────────────────────────────────

func TestSetSessionIdentifier_NilSession(t *testing.T) {
	// 不应 panic
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "t1"})
	setSessionIdentifier(nil, ctx)
}

func TestSetSessionIdentifier_WritesIdentifier(t *testing.T) {
	session := sessions.NewSession(nil, "test")
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "session-tenant"})
	setSessionIdentifier(session, ctx)

	got, ok := session.Values[sessionKeyIdentifier].(string)
	if !ok || got != "session-tenant" {
		t.Fatalf("expected session-tenant, got %q", got)
	}
}

func TestSetSessionIdentifier_EmptyIdentifier(t *testing.T) {
	session := sessions.NewSession(nil, "test")
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: ""})
	setSessionIdentifier(session, ctx)

	// 空 identifier 不应写入
	if _, ok := session.Values[sessionKeyIdentifier]; ok {
		t.Fatal("empty identifier should not be stored in session")
	}
}

func TestSetSessionIdentifier_NoSnapshot(t *testing.T) {
	session := sessions.NewSession(nil, "test")
	setSessionIdentifier(session, context.Background())

	if _, ok := session.Values[sessionKeyIdentifier]; ok {
		t.Fatal("no snapshot ctx should not write identifier")
	}
}

// ─── validateSessionIdentifier ─────────────────────────────────────────────

func TestValidateSessionIdentifier_NilSession(t *testing.T) {
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "t"})
	if validateSessionIdentifier(nil, ctx) {
		t.Fatal("nil session should return false")
	}
}

func TestValidateSessionIdentifier_NoSnapshotInCtx(t *testing.T) {
	session := sessions.NewSession(nil, "test")
	// 无 TenantSnapshot → 放行
	if !validateSessionIdentifier(session, context.Background()) {
		t.Fatal("no snapshot should allow pass")
	}
}

func TestValidateSessionIdentifier_EmptyIdentifierInSnapshot(t *testing.T) {
	session := sessions.NewSession(nil, "test")
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: ""})
	// 空 identifier（SQLite模式）→ 放行
	if !validateSessionIdentifier(session, ctx) {
		t.Fatal("empty identifier should allow pass")
	}
}

func TestValidateSessionIdentifier_NonUniverseMode_NoSessionId(t *testing.T) {
	oldSnap := common.FixedSnapshot
	defer func() { common.FixedSnapshot = oldSnap }()
	common.FixedSnapshot = &common.TenantSnapshot{Identifier: "fixed"}

	session := sessions.NewSession(nil, "test")
	// session 中没有 identifier
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "fixed"})
	// 非 universe 模式：兼容放行旧 cookie
	if !validateSessionIdentifier(session, ctx) {
		t.Fatal("non-universe mode should allow empty session identifier (backward compat)")
	}
}

func TestValidateSessionIdentifier_NonUniverseMode_MatchingSessionId(t *testing.T) {
	oldSnap := common.FixedSnapshot
	defer func() { common.FixedSnapshot = oldSnap }()
	common.FixedSnapshot = &common.TenantSnapshot{Identifier: "fixed"}

	session := sessions.NewSession(nil, "test")
	session.Values[sessionKeyIdentifier] = "fixed"
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "fixed"})
	if !validateSessionIdentifier(session, ctx) {
		t.Fatal("matching identifiers should pass")
	}
}

func TestValidateSessionIdentifier_NonUniverseMode_MismatchSessionId(t *testing.T) {
	oldSnap := common.FixedSnapshot
	defer func() { common.FixedSnapshot = oldSnap }()
	common.FixedSnapshot = &common.TenantSnapshot{Identifier: "fixed"}

	session := sessions.NewSession(nil, "test")
	session.Values[sessionKeyIdentifier] = "other-tenant"
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "fixed"})
	if validateSessionIdentifier(session, ctx) {
		t.Fatal("mismatching identifiers should fail")
	}
}

func TestValidateSessionIdentifier_UniverseMode_NoSessionId(t *testing.T) {
	oldSnap := common.FixedSnapshot
	defer func() { common.FixedSnapshot = oldSnap }()
	common.FixedSnapshot = nil // universe mode

	session := sessions.NewSession(nil, "test")
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "uni-tenant"})
	// Universe 模式：session 无 identifier → 拒绝
	if validateSessionIdentifier(session, ctx) {
		t.Fatal("universe mode should reject empty session identifier")
	}
}

func TestValidateSessionIdentifier_UniverseMode_MatchingSessionId(t *testing.T) {
	oldSnap := common.FixedSnapshot
	defer func() { common.FixedSnapshot = oldSnap }()
	common.FixedSnapshot = nil // universe mode

	session := sessions.NewSession(nil, "test")
	session.Values[sessionKeyIdentifier] = "uni-tenant"
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "uni-tenant"})
	if !validateSessionIdentifier(session, ctx) {
		t.Fatal("universe mode should allow matching session identifier")
	}
}

func TestValidateSessionIdentifier_UniverseMode_MismatchSessionId(t *testing.T) {
	oldSnap := common.FixedSnapshot
	defer func() { common.FixedSnapshot = oldSnap }()
	common.FixedSnapshot = nil // universe mode

	session := sessions.NewSession(nil, "test")
	session.Values[sessionKeyIdentifier] = "wrong-tenant"
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "uni-tenant"})
	if validateSessionIdentifier(session, ctx) {
		t.Fatal("universe mode should reject mismatching session identifier")
	}
}

// ─── currentIdentifierFromCtx ─────────────────────────────────────────────

func TestCurrentIdentifierFromCtx_WithSnapshot(t *testing.T) {
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "ctx-id"})
	if got := currentIdentifierFromCtx(ctx); got != "ctx-id" {
		t.Fatalf("expected ctx-id, got %q", got)
	}
}

func TestCurrentIdentifierFromCtx_WithoutSnapshot(t *testing.T) {
	if got := currentIdentifierFromCtx(context.Background()); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}
