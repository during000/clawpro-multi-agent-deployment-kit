package model

import (
	"context"
	"testing"
	"time"

	"hatchery/common"
)

// TestLockName_ShortResource 短资源标识符直接生成锁名
func TestLockName_ShortResource(t *testing.T) {
	snap := common.TenantSnapshot{Identifier: "my-tenant"}
	ctx := common.InjectTenant(context.Background(), snap)
	resource := "sg-bootstrap-my-tenant"
	finalKey := lockName(ctx, resource)
	if len(finalKey) > 64 {
		t.Errorf("lockName = %q (len=%d), exceeds 64", finalKey, len(finalKey))
	}
	expected := "my-tenant:sg-bootstrap-my-tenant"
	if finalKey != expected {
		t.Errorf("expected %q, got %q", expected, finalKey)
	}
}

// TestLockName_LongResource 长资源标识符的锁名也不超过64字符（前缀+resource截断）
func TestLockName_LongResource(t *testing.T) {
	ident := "abcdefghij-1234567890-abcde"
	snap := common.TenantSnapshot{Identifier: ident}
	ctx := common.InjectTenant(context.Background(), snap)
	resource := "sg-bootstrap-short"
	finalKey := lockName(ctx, resource)
	if len(finalKey) > 64 {
		t.Errorf("lockName = %q (len=%d), exceeds 64", finalKey, len(finalKey))
	}
}

// TestLockName_DifferentResources 不同资源标识符产生不同的锁名
func TestLockName_DifferentResources(t *testing.T) {
	snap := common.TenantSnapshot{Identifier: "t1"}
	ctx := common.InjectTenant(context.Background(), snap)
	r1 := lockName(ctx, "resource-aaa")
	r2 := lockName(ctx, "resource-bbb")
	if r1 == r2 {
		t.Errorf("different resources produced same lock name: %q", r1)
	}
}

// TestLockNameWithIdentifier 测试 lockName 生成带前缀的锁名
func TestLockNameWithIdentifier(t *testing.T) {
	snap := common.TenantSnapshot{Identifier: "tenant-1"}
	ctx := common.InjectTenant(context.Background(), snap)

	name := lockName(ctx, "resource:42")
	expected := "tenant-1:resource:42"
	if name != expected {
		t.Fatalf("expected %q, got %q", expected, name)
	}
}

// TestLockNameWithoutIdentifier 测试 lockName 不带前缀
func TestLockNameWithoutIdentifier(t *testing.T) {
	ctx := context.Background()
	name := lockName(ctx, "resource:42")
	expected := "resource:42"
	if name != expected {
		t.Fatalf("expected %q, got %q", expected, name)
	}
}

// TestLockNameWithSkipIdentifier 测试 WithSkipIdentifier 时 lockName 不带前缀
func TestLockNameWithSkipIdentifier(t *testing.T) {
	snap := common.TenantSnapshot{Identifier: "tenant-1"}
	ctx := common.WithSkipIdentifier(common.InjectTenant(context.Background(), snap))

	name := lockName(ctx, "resource:42")
	expected := "resource:42"
	if name != expected {
		t.Fatalf("expected %q, got %q", expected, name)
	}
}

// TestAcquireLockSQLiteMode 测试 SQLite 模式下的 AcquireLock（空操作）
func TestAcquireLockSQLiteMode(t *testing.T) {
	old := dbDriver
	dbDriver = "sqlite"
	defer func() { dbDriver = old }()

	ctx := context.Background()
	lock, err := AcquireLock(ctx, "test", 5*time.Second)

	if err != nil {
		t.Fatalf("SQLite mode should not return error: %v", err)
	}
	if lock == nil {
		t.Fatal("SQLite mode should return non-nil DistLock")
	}
	if lock.conn != nil {
		t.Fatal("SQLite mode DistLock should have nil conn")
	}

	lock.Release()
	lock.Release() // 多次调用不应 panic
}

// TestTryLockSQLiteMode 测试 SQLite 模式下的 TryLock（空操作）
func TestTryLockSQLiteMode(t *testing.T) {
	old := dbDriver
	dbDriver = "sqlite"
	defer func() { dbDriver = old }()

	ctx := context.Background()
	lock, err := TryLock(ctx, "test")

	if err != nil {
		t.Fatalf("SQLite TryLock should not return error: %v", err)
	}
	if lock == nil {
		t.Fatal("SQLite TryLock should return non-nil DistLock")
	}
}

// TestReleaseNilLock 测试释放 nil lock
func TestReleaseNilLock(t *testing.T) {
	var lock *DistLock
	lock.Release()
}

// TestReleaseEmptyLock 测试释放空 lock
func TestReleaseEmptyLock(t *testing.T) {
	lock := &DistLock{}
	lock.Release()
}

// TestIsLockHeldSQLiteMode 测试 SQLite 模式下的 IsLockHeld
func TestIsLockHeldSQLiteMode(t *testing.T) {
	old := dbDriver
	dbDriver = "sqlite"
	defer func() { dbDriver = old }()

	ctx := context.Background()
	held, err := IsLockHeld(ctx, "test")

	if err != nil {
		t.Fatalf("SQLite IsLockHeld should not return error: %v", err)
	}
	if held {
		t.Fatal("SQLite IsLockHeld should always return false")
	}
}

// TestWithLockSQLiteMode 测试 SQLite 模式下的 WithLock
func TestWithLockSQLiteMode(t *testing.T) {
	old := dbDriver
	dbDriver = "sqlite"
	defer func() { dbDriver = old }()

	ctx := context.Background()
	executed := false

	err := WithLock(ctx, "test", 5*time.Second, func(ctx context.Context) error {
		executed = true
		return nil
	})

	if err != nil {
		t.Fatalf("SQLite WithLock should not return error: %v", err)
	}
	if !executed {
		t.Fatal("WithLock should execute the function")
	}
}

// TestWithLockSQLiteModeWithError 测试 SQLite 模式下 WithLock 中函数返回错误
func TestWithLockSQLiteModeWithError(t *testing.T) {
	old := dbDriver
	dbDriver = "sqlite"
	defer func() { dbDriver = old }()

	ctx := context.Background()
	executed := false

	err := WithLock(ctx, "test", 5*time.Second, func(ctx context.Context) error {
		executed = true
		return context.DeadlineExceeded
	})

	if err != context.DeadlineExceeded {
		t.Fatalf("WithLock should propagate function error: %v", err)
	}
	if !executed {
		t.Fatal("WithLock should execute the function even if it fails")
	}
}
