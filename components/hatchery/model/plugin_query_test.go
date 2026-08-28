package model

import (
	"context"
	"os"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupPluginQueryTestDB 创建测试数据库用于 BuildPluginInstanceQuery 测试
func setupPluginQueryTestDB(t *testing.T) (cleanup func()) {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "plugin_query_test_*.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	tmpFile.Close()

	dsn := tmpFile.Name() + "?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
	testDB, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("open test db: %v", err)
	}

	origDB := gdb
	gdb = testDB

	if err := gdb.AutoMigrate(&Instance{}, &User{}, &Plugin{}, &PluginDistributionRecord{}); err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("auto migrate: %v", err)
	}

	return func() {
		sqlDB, _ := gdb.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
		os.Remove(tmpFile.Name())
		os.Remove(tmpFile.Name() + "-wal")
		os.Remove(tmpFile.Name() + "-shm")
		gdb = origDB
	}
}

func TestBuildPluginInstanceQuery_FiltersUnsupportedTypes(t *testing.T) {
	cleanup := setupPluginQueryTestDB(t)
	defer cleanup()

	// 创建用户
	user := User{Username: "testuser", Password: "test"}
	gdb.Create(&user)

	// 创建不同类型的实例
	instances := []Instance{
		{Name: "openclaw-inst", InstanceId: "ins-oc-001", UserID: user.ID, AgentType: "openclaw"},
		{Name: "hermes-inst", InstanceId: "ins-hm-001", UserID: user.ID, AgentType: "hermes"},
		{Name: "lightclaw-inst", InstanceId: "ins-lc-001", UserID: user.ID, AgentType: "lightclawace"},
		{Name: "legacy-inst", InstanceId: "ins-lg-001", UserID: user.ID, AgentType: ""}, // 存量空类型
	}
	for _, inst := range instances {
		gdb.Create(&inst)
	}

	type instResp struct {
		InstanceID   uint   `gorm:"column:instance_id"`
		InstanceName string `gorm:"column:instance_name"`
		InstanceType string `gorm:"column:instance_type"`
	}

	var results []instResp
	pluginIDs := []uint{0} // 用 0 只是为了构建查询
	q, err := BuildPluginInstanceQuery(context.Background(), pluginIDs, "1.0.0")
	if err != nil {
		t.Fatalf("构造查询失败: %v", err)
	}
	q.Scan(&results)

	// openclaw 和 空类型（兼容）应该被包含，hermes 和 lightclawace 应被过滤
	foundOpenclaw := false
	foundLegacy := false
	for _, r := range results {
		if r.InstanceName == "openclaw-inst" {
			foundOpenclaw = true
		}
		if r.InstanceName == "legacy-inst" {
			foundLegacy = true
		}
		if r.InstanceName == "hermes-inst" {
			t.Error("hermes instance should be filtered out (does not support plugin)")
		}
		if r.InstanceName == "lightclaw-inst" {
			t.Error("lightclawace instance should be filtered out (does not support plugin)")
		}
	}

	if !foundOpenclaw {
		t.Error("openclaw instance should be included")
	}
	if !foundLegacy {
		t.Error("legacy instance (empty agent_type) should be included")
	}
}

func TestBuildPluginInstanceQuery_ExcludesEmptyInstanceId(t *testing.T) {
	cleanup := setupPluginQueryTestDB(t)
	defer cleanup()

	user := User{Username: "testuser", Password: "test"}
	gdb.Create(&user)

	// 实例没有 CVM InstanceId
	inst := Instance{Name: "no-cvm", InstanceId: "", UserID: user.ID, AgentType: "openclaw"}
	gdb.Create(&inst)

	type instResp struct {
		InstanceID   uint   `gorm:"column:instance_id"`
		InstanceName string `gorm:"column:instance_name"`
	}

	var results []instResp
	pluginIDs := []uint{0}
	q, err := BuildPluginInstanceQuery(context.Background(), pluginIDs, "1.0.0")
	if err != nil {
		t.Fatalf("构造查询失败: %v", err)
	}
	q.Scan(&results)

	if len(results) != 0 {
		t.Errorf("expected 0 results for instance without CVM ID, got %d", len(results))
	}
}
