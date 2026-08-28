package model

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupMemoryTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试 gdb 失败: %v", err)
	}
	gdb = db
	if err := db.AutoMigrate(
		&MemoryTDAIPlugin{},
		&TdaiJob{},
		&SiteConfig{},
	); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	db.Create(&SiteConfig{
		MemoryTDAIEnable:            false,
		MemoryTDAISupportedVersions: "[]",
	})
}

// --- 常量正确性 ---

func TestMemoryPlanConstants(t *testing.T) {
	if MemoryPlanOff != "OFF" {
		t.Errorf("MemoryPlanOff = %q, want OFF", MemoryPlanOff)
	}
	if MemoryPlanFree != "FREE" {
		t.Errorf("MemoryPlanFree = %q, want FREE", MemoryPlanFree)
	}
	if MemoryPlanPro != "PRO" {
		t.Errorf("MemoryPlanPro = %q, want PRO", MemoryPlanPro)
	}
}

func TestMemorySwitchStatusConstants(t *testing.T) {
	if MemorySwitchStatusNone != "" {
		t.Errorf("MemorySwitchStatusNone = %q, want empty", MemorySwitchStatusNone)
	}
	if MemorySwitchStatusSwitchingToOff != "SWITCHING_TO_OFF" {
		t.Errorf("got %q", MemorySwitchStatusSwitchingToOff)
	}
	if MemorySwitchStatusSwitchingToFree != "SWITCHING_TO_FREE" {
		t.Errorf("got %q", MemorySwitchStatusSwitchingToFree)
	}
	if MemorySwitchStatusSwitchingToPro != "SWITCHING_TO_PRO" {
		t.Errorf("got %q", MemorySwitchStatusSwitchingToPro)
	}
}

func TestTdaiJobStateConstants(t *testing.T) {
	states := map[string]string{
		"PENDING":   TdaiJobStatePending,
		"RUNNING":   TdaiJobStateRunning,
		"SUCCEEDED": TdaiJobStateSucceeded,
		"FAILED":    TdaiJobStateFailed,
		"CANCELED":  TdaiJobStateCanceled,
	}
	for want, got := range states {
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}

func TestTdaiJobTypeConstants(t *testing.T) {
	types := map[string]string{
		"SWITCH_TO_FREE": TdaiJobTypeSwitchToFree,
		"SWITCH_TO_OFF":  TdaiJobTypeSwitchToOff,
		"SWITCH_TO_PRO":  TdaiJobTypeSwitchToPro,
	}
	for want, got := range types {
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}

// --- NormalizeMemoryTDAISupportedVersions ---

func TestNormalize_Empty(t *testing.T) {
	norm, versions, err := NormalizeMemoryTDAISupportedVersions("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if norm != "[]" {
		t.Errorf("norm = %q, want []", norm)
	}
	if len(versions) != 0 {
		t.Errorf("versions should be empty, got %v", versions)
	}
}

func TestNormalize_ValidJSON(t *testing.T) {
	norm, versions, err := NormalizeMemoryTDAISupportedVersions(`["1.0","2.0","1.0"]`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if norm != `["1.0","2.0"]` {
		t.Errorf("norm = %q, want deduplicated", norm)
	}
	if len(versions) != 2 {
		t.Errorf("len(versions) = %d, want 2", len(versions))
	}
}

func TestNormalize_InvalidJSON(t *testing.T) {
	_, _, err := NormalizeMemoryTDAISupportedVersions("not-json")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestNormalize_EmptyStringElement(t *testing.T) {
	_, _, err := NormalizeMemoryTDAISupportedVersions(`["ok", ""]`)
	if err == nil {
		t.Error("expected error for empty string element")
	}
}

// --- EnsureMemoryTDAIPluginRow ---

func TestEnsurePluginRow_CreatesNew(t *testing.T) {
	setupMemoryTestDB(t)

	EnsureMemoryTDAIPluginRow(context.Background(), "ins-new-001")

	var count int64
	gdb.Model(&MemoryTDAIPlugin{}).Where("instance_id = ?", "ins-new-001").Count(&count)
	if count != 1 {
		t.Fatalf("expected 1 row, got %d", count)
	}
}

func TestEnsurePluginRow_Idempotent(t *testing.T) {
	setupMemoryTestDB(t)

	EnsureMemoryTDAIPluginRow(context.Background(), "ins-idem-001")
	EnsureMemoryTDAIPluginRow(context.Background(), "ins-idem-001")

	var count int64
	gdb.Model(&MemoryTDAIPlugin{}).Where("instance_id = ?", "ins-idem-001").Count(&count)
	if count != 1 {
		t.Fatalf("expected 1 row (idempotent), got %d", count)
	}
}

func TestEnsurePluginRow_EmptyInstanceID(t *testing.T) {
	setupMemoryTestDB(t)

	EnsureMemoryTDAIPluginRow(context.Background(), "")

	var count int64
	gdb.Model(&MemoryTDAIPlugin{}).Count(&count)
	if count != 0 {
		t.Fatalf("empty instance_id should not create row, got %d", count)
	}
}

func TestEnsurePluginRow_DefaultValues(t *testing.T) {
	setupMemoryTestDB(t)

	EnsureMemoryTDAIPluginRow(context.Background(), "ins-default-001")

	var plugin MemoryTDAIPlugin
	gdb.Where("instance_id = ?", "ins-default-001").First(&plugin)
	if plugin.CurrentPlan != MemoryPlanOff {
		t.Errorf("current_plan = %q, want OFF", plugin.CurrentPlan)
	}
	if plugin.DesiredPlan != MemoryPlanOff {
		t.Errorf("desired_plan = %q, want OFF", plugin.DesiredPlan)
	}
	if plugin.Status != MemoryTDAIPluginStatusNotInstalled {
		t.Errorf("status = %q, want NOT_INSTALLED", plugin.Status)
	}
}

// --- GetMemoryTDAIPlugin ---

func TestGetPlugin_Found(t *testing.T) {
	setupMemoryTestDB(t)
	gdb.Create(&MemoryTDAIPlugin{InstanceID: "ins-get-001", Status: MemoryTDAIPluginStatusEnabled, CurrentPlan: MemoryPlanFree})

	plugin := GetMemoryTDAIPlugin(context.Background(), "ins-get-001")
	if plugin == nil {
		t.Fatal("expected plugin, got nil")
	}
	if plugin.CurrentPlan != MemoryPlanFree {
		t.Errorf("current_plan = %q, want FREE", plugin.CurrentPlan)
	}
}

func TestGetPlugin_NotFound(t *testing.T) {
	setupMemoryTestDB(t)

	plugin := GetMemoryTDAIPlugin(context.Background(), "ins-nonexist")
	if plugin != nil {
		t.Fatal("expected nil for non-existing plugin")
	}
}

func TestGetPlugin_EmptyID(t *testing.T) {
	setupMemoryTestDB(t)

	plugin := GetMemoryTDAIPlugin(context.Background(), "")
	if plugin != nil {
		t.Fatal("expected nil for empty instance_id")
	}
}

// --- DeleteMemoryTDAIPluginRow ---

func TestDeletePluginRow(t *testing.T) {
	setupMemoryTestDB(t)
	gdb.Create(&MemoryTDAIPlugin{InstanceID: "ins-del-001", Status: MemoryTDAIPluginStatusEnabled})

	DeleteMemoryTDAIPluginRow(context.Background(), "ins-del-001")

	var count int64
	gdb.Model(&MemoryTDAIPlugin{}).Where("instance_id = ?", "ins-del-001").Count(&count)
	if count != 0 {
		t.Fatalf("expected 0 rows after delete, got %d", count)
	}
}

func TestDeletePluginRow_EmptyID(t *testing.T) {
	setupMemoryTestDB(t)
	gdb.Create(&MemoryTDAIPlugin{InstanceID: "ins-keep", Status: MemoryTDAIPluginStatusEnabled})

	DeleteMemoryTDAIPluginRow(context.Background(), "")

	var count int64
	gdb.Model(&MemoryTDAIPlugin{}).Count(&count)
	if count != 1 {
		t.Fatalf("empty id should not delete anything, got %d rows", count)
	}
}

// --- MigrateMemoryPlanFromStatus ---

func TestMigrate_EnabledToFree(t *testing.T) {
	setupMemoryTestDB(t)
	gdb.Create(&MemoryTDAIPlugin{
		InstanceID:  "ins-mig-001",
		Status:      MemoryTDAIPluginStatusEnabled,
		CurrentPlan: MemoryPlanOff,
		DesiredPlan: MemoryPlanOff,
	})

	MigrateMemoryPlanFromStatus(context.Background())

	var plugin MemoryTDAIPlugin
	gdb.Where("instance_id = ?", "ins-mig-001").First(&plugin)
	if plugin.CurrentPlan != MemoryPlanFree {
		t.Errorf("current_plan = %q, want FREE", plugin.CurrentPlan)
	}
	if plugin.DesiredPlan != MemoryPlanFree {
		t.Errorf("desired_plan = %q, want FREE", plugin.DesiredPlan)
	}
}

func TestMigrate_AlreadyMigrated(t *testing.T) {
	setupMemoryTestDB(t)
	gdb.Create(&MemoryTDAIPlugin{
		InstanceID:  "ins-mig-002",
		Status:      MemoryTDAIPluginStatusEnabled,
		CurrentPlan: MemoryPlanFree, // 已迁移过
		DesiredPlan: MemoryPlanFree,
	})

	MigrateMemoryPlanFromStatus(context.Background())

	var plugin MemoryTDAIPlugin
	gdb.Where("instance_id = ?", "ins-mig-002").First(&plugin)
	if plugin.CurrentPlan != MemoryPlanFree {
		t.Errorf("should remain FREE, got %q", plugin.CurrentPlan)
	}
}

func TestMigrate_DisabledNotMigrated(t *testing.T) {
	setupMemoryTestDB(t)
	gdb.Create(&MemoryTDAIPlugin{
		InstanceID:  "ins-mig-003",
		Status:      MemoryTDAIPluginStatusDisabled,
		CurrentPlan: MemoryPlanOff,
		DesiredPlan: MemoryPlanOff,
	})

	MigrateMemoryPlanFromStatus(context.Background())

	var plugin MemoryTDAIPlugin
	gdb.Where("instance_id = ?", "ins-mig-003").First(&plugin)
	if plugin.CurrentPlan != MemoryPlanOff {
		t.Errorf("DISABLED should stay OFF, got %q", plugin.CurrentPlan)
	}
}

// --- SubmitJob 边界用例 ---

func TestSubmitJob_EmptyBizKey(t *testing.T) {
	setupMemoryTestDB(t)

	_, err := SubmitJob(context.Background(), TdaiJobTypeSwitchToFree, "", "inst", "{}", "u", "t")
	if err == nil {
		t.Fatal("empty biz_key should error")
	}
}

func TestSubmitJob_DefaultMaxAttempts(t *testing.T) {
	setupMemoryTestDB(t)

	job, err := SubmitJob(context.Background(), TdaiJobTypeSwitchToFree, "bk:001", "inst", "{}", "u", "t")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if job.MaxAttempts != TdaiJobDefaultMaxAttempts {
		t.Errorf("max_attempts = %d, want %d", job.MaxAttempts, TdaiJobDefaultMaxAttempts)
	}
}

// --- RetryJob 边界用例 ---

func TestRetryJob_NotFailed(t *testing.T) {
	setupMemoryTestDB(t)

	job, _ := SubmitJob(context.Background(), TdaiJobTypeSwitchToFree, "bk:retry-nf", "inst", "{}", "u", "t")
	// job 仍是 PENDING
	err := RetryJob(context.Background(), job.ID)
	if err == nil {
		t.Fatal("RetryJob on PENDING should error")
	}
}

func TestRetryJob_NonexistentID(t *testing.T) {
	setupMemoryTestDB(t)

	err := RetryJob(context.Background(), 99999)
	if err == nil {
		t.Fatal("RetryJob on nonexistent ID should error")
	}
}

// --- CancelJob 边界用例 ---

func TestCancelJob_AlreadyRunning(t *testing.T) {
	setupMemoryTestDB(t)

	job, _ := SubmitJob(context.Background(), TdaiJobTypeSwitchToFree, "bk:cancel-run", "inst", "{}", "u", "t")
	gdb.Model(job).Update("state", TdaiJobStateRunning)

	err := CancelJob(context.Background(), job.ID)
	if err == nil {
		t.Fatal("CancelJob on RUNNING should error")
	}
}

func TestCancelJob_SetsFinishedAt(t *testing.T) {
	setupMemoryTestDB(t)

	job, _ := SubmitJob(context.Background(), TdaiJobTypeSwitchToFree, "bk:cancel-ts", "inst", "{}", "u", "t")
	if err := CancelJob(context.Background(), job.ID); err != nil {
		t.Fatalf("CancelJob failed: %v", err)
	}

	var updated TdaiJob
	gdb.First(&updated, job.ID)
	if updated.FinishedAt == nil {
		t.Error("finished_at should be set after cancel")
	}
	if updated.FinishedAt != nil && time.Since(*updated.FinishedAt) > 5*time.Second {
		t.Error("finished_at should be recent")
	}
}
