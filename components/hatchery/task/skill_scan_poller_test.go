package task

import (
	"context"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupSkillScanPollerTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.AutoMigrate(&model.SkillSecurityScan{}, &model.SkillScanViolation{}); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
}

func TestRunSkillScanPoller_NoScanTasks(t *testing.T) {
	setupSkillScanPollerTestDB(t)
	// 没有待轮询的扫描任务，应正常返回不 panic
	runSkillScanPoller(context.Background())
}
