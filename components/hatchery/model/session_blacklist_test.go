package model

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupSessionBlacklistTestDB 创建测试数据库用于 SessionBlacklist 测试
func setupSessionBlacklistTestDB(t *testing.T) (cleanup func()) {
	t.Helper()
	f, err := os.CreateTemp("", "test-session-blacklist-*.db")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	f.Close()
	db, err := gorm.Open(sqlite.Open(f.Name()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("打开数据库失败: %v", err)
	}
	db.AutoMigrate(&SessionBlacklist{})
	orig := UseDBForTest(db)
	dbDriver = "sqlite"
	return func() {
		orig()
		os.Remove(f.Name())
	}
}

func TestRevokeSession(t *testing.T) {
	cleanup := setupSessionBlacklistTestDB(t)
	defer cleanup()

	err := RevokeSession(context.Background(), "sid-1", "sub-1", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("RevokeSession failed: %v", err)
	}

	// 幂等性：重复调用不报错
	err = RevokeSession(context.Background(), "sid-1", "sub-1", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("RevokeSession(idempotent) failed: %v", err)
	}
}

func TestIsSessionRevoked_BySid(t *testing.T) {
	cleanup := setupSessionBlacklistTestDB(t)
	defer cleanup()

	RevokeSession(context.Background(), "sid-revoked", "sub-x", time.Now().Add(time.Hour))

	if !IsSessionRevoked(context.Background(), "sid-revoked", "", time.Time{}) {
		t.Error("expected revoked by sid")
	}
	if IsSessionRevoked(context.Background(), "sid-other", "", time.Time{}) {
		t.Error("expected not revoked for different sid")
	}
}

func TestIsSessionRevoked_BySub(t *testing.T) {
	cleanup := setupSessionBlacklistTestDB(t)
	defer cleanup()

	RevokeSession(context.Background(), "sid-sub", "sub-target", time.Now().Add(time.Hour))

	// sub 维度匹配：loginAt 在 blacklist 写入之前
	if !IsSessionRevoked(context.Background(), "", "sub-target", time.Now().Add(-time.Hour)) {
		t.Error("expected revoked by sub (loginAt before blacklist)")
	}
	// sub 维度不匹配：loginAt 在 blacklist 写入之后
	if IsSessionRevoked(context.Background(), "", "sub-target", time.Now().Add(time.Hour)) {
		t.Error("expected not revoked by sub (loginAt after blacklist)")
	}
}

func TestIsSessionRevoked_Empty(t *testing.T) {
	cleanup := setupSessionBlacklistTestDB(t)
	defer cleanup()

	if IsSessionRevoked(context.Background(), "", "", time.Time{}) {
		t.Error("empty sid and sub should not be revoked")
	}
}

func TestIsSidRevoked(t *testing.T) {
	cleanup := setupSessionBlacklistTestDB(t)
	defer cleanup()

	RevokeSession(context.Background(), "sid-check", "", time.Now().Add(time.Hour))
	if !IsSidRevoked(context.Background(), "sid-check") {
		t.Error("expected revoked")
	}
	if IsSidRevoked(context.Background(), "sid-unknown") {
		t.Error("expected not revoked")
	}
}

func TestCleanExpiredSessions(t *testing.T) {
	cleanup := setupSessionBlacklistTestDB(t)
	defer cleanup()

	// 创建一条已过期的记录和一条未过期的
	RevokeSession(context.Background(), "sid-expired", "", time.Now().Add(-time.Hour))
	RevokeSession(context.Background(), "sid-valid", "", time.Now().Add(time.Hour))

	// 手动修改过期时间为过去
	gdb.Model(&SessionBlacklist{}).Where("one_id_sid = ?", "sid-expired").Update("expire_at", time.Now().Add(-time.Hour))

	CleanExpiredSessions(context.Background())

	var count int64
	gdb.Model(&SessionBlacklist{}).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 remaining, got %d", count)
	}
}

func TestIsSessionRevoked_ExpiredSid(t *testing.T) {
	cleanup := setupSessionBlacklistTestDB(t)
	defer cleanup()

	// 已过期的 sid 不应该被认为是 revoked
	RevokeSession(context.Background(), "sid-exp", "", time.Now().Add(-time.Minute))
	// 手动设置为已过期
	gdb.Model(&SessionBlacklist{}).Where("one_id_sid = ?", "sid-exp").Update("expire_at", time.Now().Add(-time.Minute))

	if IsSessionRevoked(context.Background(), "sid-exp", "", time.Time{}) {
		t.Error("expired sid should not be revoked")
	}
}
