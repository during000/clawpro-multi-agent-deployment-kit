package task

import (
	"context"
	"testing"
	"time"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestRunNotificationCleanup(t *testing.T) {
	// 初始化带 Notification 表的 DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.Notification{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	origDB := model.UseDBForTest(db)
	defer origDB()

	// runNotificationCleanup 应不 panic
	runNotificationCleanup(context.Background())
}

func TestCleanupExpiredNotifications_Basic(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.Notification{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	origDB := model.UseDBForTest(db)
	defer origDB()

	// 创建两条旧通知
	old := time.Now().AddDate(0, 0, -35)
	db.Create(&model.Notification{UserID: 1, InstanceID: 1, InstanceName: "inst1", Type: "test", Title: "old"})
	db.Model(&model.Notification{}).Where("1 = 1").Update("created_at", old)

	affected, err := model.CleanupExpiredNotifications(context.Background(), 30)
	if err != nil {
		t.Fatalf("CleanupExpiredNotifications: %v", err)
	}
	if affected == 0 {
		t.Error("expected to delete at least 1 old notification")
	}
}
