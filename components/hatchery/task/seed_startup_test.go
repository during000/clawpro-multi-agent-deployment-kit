package task

import (
	"context"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupFullTestDB 创建包含所有表的内存 SQLite 测试库。
func setupFullTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetConnMaxIdleTime(0)
	sqlDB.SetConnMaxLifetime(0)

	if err := db.AutoMigrate(model.AllModelsForTest()...); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	t.Cleanup(model.UseDBForTestWithDriver(db, "sqlite"))
}

// TestRunSeedStartup_NoData 验证 runSeedStartup 在空库上正常执行。
func TestRunSeedStartup_NoData(t *testing.T) {
	setupFullTestDB(t)
	runSeedStartup(context.Background())

	// 验证 Seed 数据已写入
	var count int64
	model.DB(context.Background()).Model(&model.AIChannel{}).Count(&count)
	if count == 0 {
		t.Error("runSeedStartup 应创建预置渠道")
	}
}

// TestRunStartupMigrations_NoData 验证 runStartupMigrations 在空库上不 panic。
func TestRunStartupMigrations_NoData(t *testing.T) {
	setupFullTestDB(t)
	runStartupMigrations(context.Background())
}

// TestRunStartupMigrations_WithRecoverData 验证迁移中的 recover 逻辑正常工作。
func TestRunStartupMigrations_WithRecoverData(t *testing.T) {
	setupFullTestDB(t)

	// 插入需要恢复的 running 任务
	task := model.SkillDistributionTask{SkillID: 1, Version: "1.0.0", Total: 1, Status: "running"}
	model.DB(context.Background()).Create(&task)
	model.DB(context.Background()).Create(&model.SkillDistributionRecord{
		TaskID: task.ID, SkillID: 1, InstanceID: 1, Status: "pending",
	})

	runStartupMigrations(context.Background())

	var updated model.SkillDistributionTask
	model.DB(context.Background()).First(&updated, task.ID)
	if updated.Status != "completed" {
		t.Errorf("expected completed, got %s", updated.Status)
	}
}
