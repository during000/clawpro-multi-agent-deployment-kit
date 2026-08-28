package model

import (
	"context"
	"testing"

	"hatchery/common"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		version   string
		wantMajor int
		wantMinor int
		wantPatch int
		wantErr   bool
	}{
		{"1.0.0", 1, 0, 0, false},
		{"2.10.3", 2, 10, 3, false},
		{"0.0.1", 0, 0, 1, false},
		{"10.20.30", 10, 20, 30, false},

		{"abc", 0, 0, 0, true},     // 非数字
		{"1.0", 0, 0, 0, true},     // 缺少 patch
		{"1", 0, 0, 0, true},       // 只有 major
		{"1.0.0.0", 0, 0, 0, true}, // 多余字段（SplitN 3 后 "0.0" 不是数字）
		{"a.b.c", 0, 0, 0, true},   // 全部非数字
		{"", 0, 0, 0, true},        // 空字符串
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			s := &Skill{Version: tt.version}
			err := s.ParseVersion()
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseVersion(%q) expected error", tt.version)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseVersion(%q) unexpected error: %v", tt.version, err)
				return
			}
			if s.VersionMajor != tt.wantMajor || s.VersionMinor != tt.wantMinor || s.VersionPatch != tt.wantPatch {
				t.Errorf("ParseVersion(%q) = %d.%d.%d, want %d.%d.%d",
					tt.version, s.VersionMajor, s.VersionMinor, s.VersionPatch,
					tt.wantMajor, tt.wantMinor, tt.wantPatch)
			}
		})
	}
}

func TestNormalizeSkillVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
		wantErr bool
	}{
		{name: "canonical", version: "1.2.3", want: "1.2.3"},
		{name: "leading zeros", version: "01.002.0003", want: "1.2.3"},
		{name: "invalid format", version: "1.2", wantErr: true},
		{name: "sql payload", version: "1.0.0' OR '1'='1", wantErr: true},
		{name: "negative component", version: "-1.2.3", wantErr: true},
		{name: "signed component", version: "1.+2.3", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeSkillVersion(tt.version)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("NormalizeSkillVersion(%q) expected error, got %q", tt.version, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeSkillVersion(%q) unexpected error: %v", tt.version, err)
			}
			if got != tt.want {
				t.Fatalf("NormalizeSkillVersion(%q) = %q, want %q", tt.version, got, tt.want)
			}
		})
	}
}

func TestFilterSkillInstallStatusesPreservesTenantAndSoftDeleteHooks(t *testing.T) {
	defer setupTestDB(t, "tenant-a")()

	ctxA := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "tenant-a"})
	ctxB := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "tenant-b"})

	active := Instance{Name: "active-a", InstanceId: "ins-active-a"}
	deleted := Instance{Name: "deleted-a", InstanceId: "ins-deleted-a"}
	otherTenant := Instance{Name: "active-b", InstanceId: "ins-active-b"}
	if err := DB(ctxA).Create(&active).Error; err != nil {
		t.Fatalf("create active tenant-a instance: %v", err)
	}
	if err := DB(ctxA).Create(&deleted).Error; err != nil {
		t.Fatalf("create deleted tenant-a instance: %v", err)
	}
	if err := DB(ctxB).Create(&otherTenant).Error; err != nil {
		t.Fatalf("create tenant-b instance: %v", err)
	}
	if err := DB(ctxA).Delete(&deleted).Error; err != nil {
		t.Fatalf("soft delete tenant-a instance: %v", err)
	}

	var rows []struct {
		InstanceID uint `gorm:"column:instance_id"`
	}
	err := BuildSkillInstanceQuery(ctxA, []uint{999}, "1.0.0", "missing-skill").
		Scopes(FilterSkillInstallStatuses("1.0.0", []string{"uninstalled"})).
		Scan(&rows).Error
	if err != nil {
		t.Fatalf("query filtered instances: %v", err)
	}
	if len(rows) != 1 || rows[0].InstanceID != active.ID {
		t.Fatalf("rows=%v, want only active tenant-a instance %d", rows, active.ID)
	}
}

func setupSkillStatusTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&SkillDistributionTask{}, &SkillDistributionRecord{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return UseDBForTest(db)
}

func seedSkillStatusRecord(t *testing.T, skillID, instanceID uint, source, slug, taskType, status string) SkillDistributionRecord {
	t.Helper()
	task := SkillDistributionTask{SkillID: skillID, Source: source, Slug: slug, Version: "1.0.0", Type: taskType, Status: "completed", Total: 1}
	if err := DB(context.Background()).Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}
	record := SkillDistributionRecord{TaskID: task.ID, SkillID: skillID, InstanceID: instanceID, Version: "1.0.0", Type: taskType, Status: status}
	if err := DB(context.Background()).Create(&record).Error; err != nil {
		t.Fatalf("create record: %v", err)
	}
	return record
}

func TestResolveDistributeFailedStatus_AfterUninstallSuccess(t *testing.T) {
	defer setupSkillStatusTestDB(t)()
	const skillID uint = 11
	const instanceID uint = 22
	seedSkillStatusRecord(t, skillID, instanceID, SkillSourceEnterprise, "enterprise-skill", TaskTypeDistribute, RecordStatusSuccess)
	seedSkillStatusRecord(t, skillID, instanceID, SkillSourceEnterprise, "enterprise-skill", TaskTypeUninstall, RecordStatusSuccess)

	if got := ResolveDistributeFailedStatus(context.Background(), instanceID, []uint{skillID}); got != RecordStatusUpgradeFailed {
		t.Fatalf("status after uninstall success = %q, want %q", got, RecordStatusUpgradeFailed)
	}
}

func TestResolvePublicSkillDistributeFailedStatus_AfterUninstallSuccess(t *testing.T) {
	defer setupSkillStatusTestDB(t)()
	const instanceID uint = 33
	seedSkillStatusRecord(t, 0, instanceID, SkillSourcePublic, "public-skill", TaskTypeDistribute, RecordStatusSuccess)
	seedSkillStatusRecord(t, 0, instanceID, SkillSourcePublic, "public-skill", TaskTypeUninstall, RecordStatusSuccess)

	if got := ResolvePublicSkillDistributeFailedStatus(context.Background(), instanceID, "public-skill", 0); got != RecordStatusFailed {
		t.Fatalf("public status after uninstall success = %q, want %q", got, RecordStatusFailed)
	}
}

func TestResolvePublicSkillDistributeFailedStatus_WhenLatestRecordStillInstalled(t *testing.T) {
	defer setupSkillStatusTestDB(t)()
	const instanceID uint = 44
	seedSkillStatusRecord(t, 0, instanceID, SkillSourcePublic, "public-still-installed", TaskTypeDistribute, RecordStatusSuccess)
	seedSkillStatusRecord(t, 0, instanceID, SkillSourcePublic, "public-still-installed", TaskTypeUninstall, RecordStatusFailed)

	if got := ResolvePublicSkillDistributeFailedStatus(context.Background(), instanceID, "public-still-installed", 0); got != RecordStatusUpgradeFailed {
		t.Fatalf("public status with failed uninstall = %q, want %q", got, RecordStatusUpgradeFailed)
	}
}

func TestResolvePublicSkillDistributeFailedStatus_ExcludesCurrentPendingRecord(t *testing.T) {
	defer setupSkillStatusTestDB(t)()
	const instanceID uint = 77
	seedSkillStatusRecord(t, 0, instanceID, SkillSourcePublic, "public-upgrade", TaskTypeDistribute, RecordStatusSuccess)
	current := seedSkillStatusRecord(t, 0, instanceID, SkillSourcePublic, "public-upgrade", TaskTypeDistribute, RecordStatusPending)

	if got := ResolvePublicSkillDistributeFailedStatus(context.Background(), instanceID, "public-upgrade", current.ID); got != RecordStatusUpgradeFailed {
		t.Fatalf("public status excluding current pending = %q, want %q", got, RecordStatusUpgradeFailed)
	}
}
