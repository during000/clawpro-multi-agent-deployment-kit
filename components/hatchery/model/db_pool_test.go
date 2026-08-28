package model

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// openPoolTestDB 创建一个仅用于 configureConnectionPool 测试的临时 DB。
// 不注入全局句柄（不调用 UseDBForTest），因为测试只验证连接池参数设置，不做 DB 操作。
func openPoolTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	// 遵循 testing.md 规范：:memory: SQLite 设置单连接
	sqlDB, _ := db.DB()
	sqlDB.SetConnMaxIdleTime(0)
	sqlDB.SetConnMaxLifetime(0)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		sqlDB.Close()
	})
	return db
}

func TestConfigureConnectionPool_Universe(t *testing.T) {
	db := openPoolTestDB(t)
	configureConnectionPool(db, true)

	sqlDB, _ := db.DB()
	stats := sqlDB.Stats()
	if stats.MaxOpenConnections != 500 {
		t.Errorf("universe MaxOpenConns = %d, want 500", stats.MaxOpenConnections)
	}
}

func TestConfigureConnectionPool_SingleTenant(t *testing.T) {
	db := openPoolTestDB(t)
	configureConnectionPool(db, false)

	sqlDB, _ := db.DB()
	stats := sqlDB.Stats()
	if stats.MaxOpenConnections != 100 {
		t.Errorf("single-tenant MaxOpenConns = %d, want 100", stats.MaxOpenConnections)
	}
}

func TestConfigureConnectionPool_PanicOnError(t *testing.T) {
	// 构造一个没有 connPool 的空 gorm.DB，使 db.DB() 返回 error
	db := &gorm.DB{Config: &gorm.Config{}}

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when db.DB() fails, got none")
		}
	}()
	configureConnectionPool(db, false)
}
