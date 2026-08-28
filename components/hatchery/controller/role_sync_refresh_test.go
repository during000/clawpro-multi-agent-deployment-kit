package controller

import (
	"context"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupRoleSyncRefreshTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&model.RoleDistributionRecord{},
		&model.Instance{},
		&model.SkillInstallation{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	origDB := model.UseDBForTest(db)
	t.Cleanup(func() {
		origDB()
		if testSafeDB != nil {
			model.SetDBForTest(testSafeDB)
		}
	})
	return db
}

func TestRefreshRoleRecord_NoUpdatingRecord(t *testing.T) {
	setupRoleSyncRefreshTestDB(t)
	// 无 record 时应 no-op
	refreshRoleRecord(context.Background(), 999)
}

func TestRefreshRoleRecord_EmptyDiffSoulSuccess_Updated(t *testing.T) {
	db := setupRoleSyncRefreshTestDB(t)
	ctx := context.Background()
	// 差集为空 + soul=success → status=updated
	db.Create(&model.RoleDistributionRecord{
		InstanceID: 1, RoleID: 1, Version: "1.0",
		Status:      model.RoleRecordStatusUpdating,
		SoulStatus:  model.RoleSubStatusSuccess,
		SkillStatus: model.RoleSubStatusPending,
	})
	refreshRoleRecord(ctx, 1)
	var rec model.RoleDistributionRecord
	db.Where("instance_id = ?", 1).First(&rec)
	if rec.Status != model.RoleRecordStatusUpdated {
		t.Errorf("期望 status=updated，实际=%s", rec.Status)
	}
	if rec.SkillStatus != model.RoleSubStatusSuccess {
		t.Errorf("期望 skill_status=success，实际=%s", rec.SkillStatus)
	}
	if rec.SkillSetAt == nil {
		t.Error("期望 skill_set_at 非 nil")
	}
}

func TestRefreshRoleRecord_SoulFailed_Failed(t *testing.T) {
	db := setupRoleSyncRefreshTestDB(t)
	ctx := context.Background()
	db.Create(&model.RoleDistributionRecord{
		InstanceID: 1, RoleID: 1, Version: "1.0",
		Status:      model.RoleRecordStatusUpdating,
		SoulStatus:  model.RoleSubStatusFailed,
		SoulError:   "TAT timeout",
		SkillStatus: model.RoleSubStatusPending,
	})
	refreshRoleRecord(ctx, 1)
	var rec model.RoleDistributionRecord
	db.Where("instance_id = ?", 1).First(&rec)
	if rec.Status != model.RoleRecordStatusFailed {
		t.Errorf("期望 status=failed，实际=%s", rec.Status)
	}
}

func TestRefreshRoleRecord_SkillFailed_Failed(t *testing.T) {
	db := setupRoleSyncRefreshTestDB(t)
	ctx := context.Background()
	// 创建 record + skill installations
	db.Create(&model.RoleDistributionRecord{
		InstanceID: 1, RoleID: 1, Version: "1.0",
		Status:      model.RoleRecordStatusUpdating,
		SoulStatus:  model.RoleSubStatusSuccess,
		SkillStatus: model.RoleSubStatusPending,
		SkillInstallationIDs: "[1,2]",
	})
	db.Create(&model.SkillInstallation{InstanceID: 1, Slug: "s1", InstallStatus: model.SkillInstallFailed, ErrorMessage: "timeout"})
	db.Create(&model.SkillInstallation{InstanceID: 1, Slug: "s2", InstallStatus: model.SkillInstallSuccess})
	refreshRoleRecord(ctx, 1)
	var rec model.RoleDistributionRecord
	db.Where("instance_id = ?", 1).First(&rec)
	if rec.Status != model.RoleRecordStatusFailed {
		t.Errorf("期望 status=failed，实际=%s", rec.Status)
	}
	if rec.SkillStatus != model.RoleSubStatusFailed {
		t.Errorf("期望 skill_status=failed，实际=%s", rec.SkillStatus)
	}
	if rec.SkillError != "timeout" {
		t.Errorf("期望 skill_error=timeout，实际=%s", rec.SkillError)
	}
}

func TestRefreshRoleRecord_MultiSkillFailed_Failed(t *testing.T) {
	db := setupRoleSyncRefreshTestDB(t)
	ctx := context.Background()
	db.Create(&model.RoleDistributionRecord{
		InstanceID: 1, RoleID: 1, Version: "1.0",
		Status:      model.RoleRecordStatusUpdating,
		SoulStatus:  model.RoleSubStatusSuccess,
		SkillStatus: model.RoleSubStatusPending,
		SkillInstallationIDs: "[1,2,3]",
	})
	db.Create(&model.SkillInstallation{InstanceID: 1, Slug: "s1", InstallStatus: model.SkillInstallFailed, ErrorMessage: "timeout"})
	db.Create(&model.SkillInstallation{InstanceID: 1, Slug: "s2", InstallStatus: model.SkillInstallFailed, ErrorMessage: "network error"})
	db.Create(&model.SkillInstallation{InstanceID: 1, Slug: "s3", InstallStatus: model.SkillInstallSuccess})
	refreshRoleRecord(ctx, 1)
	var rec model.RoleDistributionRecord
	db.Where("instance_id = ?", 1).First(&rec)
	if rec.Status != model.RoleRecordStatusFailed {
		t.Errorf("期望 status=failed，实际=%s", rec.Status)
	}
	if rec.SkillStatus != model.RoleSubStatusFailed {
		t.Errorf("期望 skill_status=failed，实际=%s", rec.SkillStatus)
	}
	// 汇总全部失败技能的错误（不再拼 slug 前缀）
	if rec.SkillError != "timeout; network error" {
		t.Errorf("期望 skill_error='timeout; network error'，实际=%s", rec.SkillError)
	}
}

func TestRefreshRoleRecord_AllSuccess_Updated(t *testing.T) {
	db := setupRoleSyncRefreshTestDB(t)
	ctx := context.Background()
	db.Create(&model.RoleDistributionRecord{
		InstanceID: 1, RoleID: 1, Version: "1.0",
		Status:      model.RoleRecordStatusUpdating,
		SoulStatus:  model.RoleSubStatusSuccess,
		SkillStatus: model.RoleSubStatusPending,
		SkillInstallationIDs: "[1,2]",
	})
	db.Create(&model.SkillInstallation{InstanceID: 1, Slug: "s1", InstallStatus: model.SkillInstallSuccess})
	db.Create(&model.SkillInstallation{InstanceID: 1, Slug: "s2", InstallStatus: model.SkillInstallSuccess})
	refreshRoleRecord(ctx, 1)
	var rec model.RoleDistributionRecord
	db.Where("instance_id = ?", 1).First(&rec)
	if rec.Status != model.RoleRecordStatusUpdated {
		t.Errorf("期望 status=updated，实际=%s", rec.Status)
	}
	if rec.SkillStatus != model.RoleSubStatusSuccess {
		t.Errorf("期望 skill_status=success，实际=%s", rec.SkillStatus)
	}
}

func TestRefreshRoleRecord_SkillRunning_StayUpdating(t *testing.T) {
	db := setupRoleSyncRefreshTestDB(t)
	ctx := context.Background()
	db.Create(&model.RoleDistributionRecord{
		InstanceID: 1, RoleID: 1, Version: "1.0",
		Status:      model.RoleRecordStatusUpdating,
		SoulStatus:  model.RoleSubStatusSuccess,
		SkillStatus: model.RoleSubStatusPending,
		SkillInstallationIDs: "[1]",
	})
	db.Create(&model.SkillInstallation{InstanceID: 1, Slug: "s1", InstallStatus: model.SkillInstalling})
	refreshRoleRecord(ctx, 1)
	var rec model.RoleDistributionRecord
	db.Where("instance_id = ?", 1).First(&rec)
	if rec.Status != model.RoleRecordStatusUpdating {
		t.Errorf("期望 status=updating，实际=%s", rec.Status)
	}
}

func TestDeriveRecordStatus(t *testing.T) {
	tests := []struct {
		soul, skill, want string
	}{
		{model.RoleSubStatusSuccess, model.RoleSubStatusSuccess, model.RoleRecordStatusUpdated},
		{model.RoleSubStatusFailed, model.RoleSubStatusSuccess, model.RoleRecordStatusFailed},
		{model.RoleSubStatusSuccess, model.RoleSubStatusFailed, model.RoleRecordStatusFailed},
		{model.RoleSubStatusFailed, model.RoleSubStatusFailed, model.RoleRecordStatusFailed},
		{model.RoleSubStatusPending, model.RoleSubStatusSuccess, model.RoleRecordStatusUpdating},
		{model.RoleSubStatusRunning, model.RoleSubStatusRunning, model.RoleRecordStatusUpdating},
	}
	for _, tt := range tests {
		got := deriveRecordStatus(tt.soul, tt.skill)
		if got != tt.want {
			t.Errorf("deriveRecordStatus(%s,%s)=%s, want %s", tt.soul, tt.skill, got, tt.want)
		}
	}
}

func TestParseSkillInstallationIDs(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"[1,2,3]", 3},
		{"invalid", 0},
		{"[]", 0},
	}
	for _, tt := range tests {
		got := parseSkillInstallationIDs(tt.input)
		if len(got) != tt.want {
			t.Errorf("parseSkillInstallationIDs(%q)=%v, want len=%d", tt.input, got, tt.want)
		}
	}
}

func TestClassifySkillInstalls(t *testing.T) {
	tests := []struct {
		name     string
		installs []model.SkillInstallation
		expected int
		want     string
	}{
		{"all_success", []model.SkillInstallation{
			{InstallStatus: model.SkillInstallSuccess},
			{InstallStatus: model.SkillInstallSuccess},
		}, 2, model.RoleSubStatusSuccess},
		{"has_failed", []model.SkillInstallation{
			{InstallStatus: model.SkillInstallSuccess},
			{InstallStatus: model.SkillInstallFailed, ErrorMessage: "err1"},
		}, 2, model.RoleSubStatusFailed},
		{"has_installing", []model.SkillInstallation{
			{InstallStatus: model.SkillInstallSuccess},
			{InstallStatus: model.SkillInstalling},
		}, 2, model.RoleSubStatusRunning},
		{"missing_records", []model.SkillInstallation{
			{InstallStatus: model.SkillInstallSuccess},
		}, 2, model.RoleSubStatusRunning},
		{"multi_failed_with_slug", []model.SkillInstallation{
			{Slug: "s1", InstallStatus: model.SkillInstallFailed, ErrorMessage: "timeout"},
			{Slug: "s2", InstallStatus: model.SkillInstallFailed, ErrorMessage: "network"},
			{Slug: "s3", InstallStatus: model.SkillInstallSuccess},
		}, 3, model.RoleSubStatusFailed},
		{"failed_no_slug", []model.SkillInstallation{
			{InstallStatus: model.SkillInstallFailed, ErrorMessage: "err"},
		}, 1, model.RoleSubStatusFailed},
		{"failed_empty_error", []model.SkillInstallation{
			{Slug: "s1", InstallStatus: model.SkillInstallFailed, ErrorMessage: ""},
		}, 1, model.RoleSubStatusFailed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, _ := classifySkillInstalls(tt.installs, tt.expected)
			if status != tt.want {
				t.Errorf("classifySkillInstalls()=%s, want %s", status, tt.want)
			}
		})
	}
}

func TestClassifySkillInstalls_MultiErrorAggregation(t *testing.T) {
	installs := []model.SkillInstallation{
		{Slug: "s1", InstallStatus: model.SkillInstallFailed, ErrorMessage: "timeout"},
		{Slug: "s2", InstallStatus: model.SkillInstallFailed, ErrorMessage: "network error"},
	}
	_, errMsg := classifySkillInstalls(installs, 2)
	expected := "timeout; network error"
	if errMsg != expected {
		t.Errorf("期望 %q，实际 %q", expected, errMsg)
	}
}

func TestClassifySkillInstalls_ErrorTruncation(t *testing.T) {
	// 构造超长错误消息验证截断
	var installs []model.SkillInstallation
	for i := 0; i < 50; i++ {
		installs = append(installs, model.SkillInstallation{
			Slug:           "skill-" + string(rune('a'+i%26)),
			InstallStatus:  model.SkillInstallFailed,
			ErrorMessage:   "very long error message that repeats over and over again to fill space",
		})
	}
	_, errMsg := classifySkillInstalls(installs, len(installs))
	if len(errMsg) > 1004 { // 1000 + "..."
		t.Errorf("期望截断到 1003 字符，实际 %d", len(errMsg))
	}
	if errMsg[len(errMsg)-3:] != "..." {
		t.Errorf("期望以 ... 结尾，实际 %q", errMsg[len(errMsg)-3:])
	}
}

func TestFinalizeActiveRecordAsCancelled(t *testing.T) {
	db := setupRoleSyncRefreshTestDB(t)
	db.Create(&model.RoleDistributionRecord{
		InstanceID: 1, RoleID: 1, Status: model.RoleRecordStatusUpdating,
	})
	db.Create(&model.RoleDistributionRecord{
		InstanceID: 1, RoleID: 1, Status: model.RoleRecordStatusUpdated,
	})
	id := finalizeActiveRecordAsCancelled(db, 1)
	if id == 0 {
		t.Error("期望返回被 finalize 的 record ID")
	}
	var rec model.RoleDistributionRecord
	db.First(&rec, id)
	if rec.Status != model.RoleRecordStatusCancelled {
		t.Errorf("期望 status=cancelled，实际=%s", rec.Status)
	}
	// updated 的 record 不应被改
	var updated model.RoleDistributionRecord
	db.Where("instance_id = ? AND status = ?", 1, model.RoleRecordStatusUpdated).First(&updated)
	if updated.ID == 0 {
		t.Error("updated record 不应被 finalize")
	}
}

func TestUpdateRecordSoulStatus(t *testing.T) {
	db := setupRoleSyncRefreshTestDB(t)
	ctx := context.Background()
	db.Create(&model.RoleDistributionRecord{
		InstanceID: 1, RoleID: 1, Status: model.RoleRecordStatusUpdating,
	})
	var rec model.RoleDistributionRecord
	db.First(&rec)

	// recordID=0 应 no-op
	updateRecordSoulStatus(ctx, 0, model.RoleSubStatusSuccess, "")

	// 正常更新
	updateRecordSoulStatus(ctx, rec.ID, model.RoleSubStatusSuccess, "")
	var updated model.RoleDistributionRecord
	db.First(&updated, rec.ID)
	if updated.SoulStatus != model.RoleSubStatusSuccess {
		t.Errorf("期望 soul_status=success，实际=%s", updated.SoulStatus)
	}
	if updated.SoulSetAt == nil {
		t.Error("期望 soul_set_at 非 nil")
	}
}

func TestFindLatestUpdatingRecordID(t *testing.T) {
	db := setupRoleSyncRefreshTestDB(t)
	ctx := context.Background()
	// 无 record
	id := findLatestUpdatingRecordID(ctx, 999)
	if id != 0 {
		t.Errorf("期望 0，实际 %d", id)
	}
	// 有 updating record
	db.Create(&model.RoleDistributionRecord{InstanceID: 1, Status: model.RoleRecordStatusUpdating})
	db.Create(&model.RoleDistributionRecord{InstanceID: 1, Status: model.RoleRecordStatusUpdating})
	id = findLatestUpdatingRecordID(ctx, 1)
	if id == 0 {
		t.Error("期望非 0")
	}
}
