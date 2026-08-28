package controller

import (
	"context"
	"testing"
	"time"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupRoleApplyTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&model.Instance{},
		&model.InstanceAdjustment{},
		&model.OpenClawRole{},
		&model.OpenClawRoleSkill{},
		&model.SkillInstallation{},
		&model.RoleDistributionRecord{},
		&model.User{},
		&model.SiteConfig{},
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

func TestBuildRoleFieldUpdates_WithRole(t *testing.T) {
	role := &model.OpenClawRole{ID: 5, Version: "2.0"}
	updates := buildRoleFieldUpdates(role)
	if updates["role_id"] != uint(5) {
		t.Errorf("期望 role_id=5，实际=%v", updates["role_id"])
	}
	if updates["distributed_role_version"] != "2.0" {
		t.Errorf("期望 distributed_role_version=2.0，实际=%v", updates["distributed_role_version"])
	}
	if updates["role_sync_status"] != model.RoleSyncStatusUpdating {
		t.Errorf("期望 role_sync_status=updating，实际=%v", updates["role_sync_status"])
	}
	if updates["soul_set_at"] != nil {
		t.Errorf("期望 soul_set_at=nil，实际=%v", updates["soul_set_at"])
	}
}

func TestBuildRoleFieldUpdates_NilRole(t *testing.T) {
	updates := buildRoleFieldUpdates(nil)
	if updates["role_id"] != uint(0) {
		t.Errorf("期望 role_id=0，实际=%v", updates["role_id"])
	}
	if updates["distributed_role_version"] != "" {
		t.Errorf("期望 distributed_role_version=空，实际=%v", updates["distributed_role_version"])
	}
	if updates["role_sync_status"] != "" {
		t.Errorf("期望 role_sync_status=空，实际=%v", updates["role_sync_status"])
	}
}

func TestPickRoleID(t *testing.T) {
	if pickRoleID(nil) != 0 {
		t.Error("nil role 应返回 0")
	}
	if pickRoleID(&model.OpenClawRole{ID: 7}) != 7 {
		t.Error("应返回 7")
	}
}

func TestPickRoleVersion(t *testing.T) {
	if pickRoleVersion(nil) != "" {
		t.Error("nil role 应返回空串")
	}
	if pickRoleVersion(&model.OpenClawRole{Version: "1.5"}) != "1.5" {
		t.Error("应返回 1.5")
	}
}

func TestPickInitialSyncStatus(t *testing.T) {
	if pickInitialSyncStatus(nil) != model.RoleSyncStatusEmpty {
		t.Error("nil role 应返回空串")
	}
	if pickInitialSyncStatus(&model.OpenClawRole{}) != model.RoleSyncStatusUpdating {
		t.Error("非 nil role 应返回 updating")
	}
}

func TestMarshalSkillInstallationIDs(t *testing.T) {
	tests := []struct {
		ids  []uint
		want string
	}{
		{nil, ""},
		{[]uint{}, ""},
		{[]uint{1}, "[1]"},
		{[]uint{1, 2, 3}, "[1,2,3]"},
	}
	for _, tt := range tests {
		got, err := marshalSkillInstallationIDs(tt.ids)
		if err != nil {
			t.Fatalf("marshalSkillInstallationIDs(%v) err=%v", tt.ids, err)
		}
		if got != tt.want {
			t.Errorf("marshalSkillInstallationIDs(%v)=%q, want %q", tt.ids, got, tt.want)
		}
	}
}

func TestWriteRoleFieldsTx_WithRole(t *testing.T) {
	db := setupRoleApplyTestDB(t)
	ctx := context.Background()
	inst := model.Instance{Name: "test", InstanceId: "ins-1", AgentType: "openclaw"}
	db.Create(&inst)
	role := &model.OpenClawRole{ID: 1, Name: "analyst", Version: "1.0"}
	db.Create(role)
	db.Create(&model.OpenClawRoleSkill{OpenClawRoleID: 1, Slug: "s1", Version: "1.0"})

	recordID, err := writeRoleFieldsTx(ctx, &inst, role, applyOptions{OperatorID: 1, Source: model.RoleRecordSourceSwitch})
	if err != nil {
		t.Fatalf("writeRoleFieldsTx 失败: %v", err)
	}
	if recordID == 0 {
		t.Fatal("期望 recordID > 0")
	}

	var updated model.Instance
	db.First(&updated, inst.ID)
	if updated.RoleID != 1 {
		t.Errorf("期望 role_id=1，实际=%d", updated.RoleID)
	}
	if updated.RoleSyncStatus != model.RoleSyncStatusUpdating {
		t.Errorf("期望 role_sync_status=updating，实际=%s", updated.RoleSyncStatus)
	}

	var rec model.RoleDistributionRecord
	db.First(&rec, recordID)
	if rec.Status != model.RoleRecordStatusUpdating {
		t.Errorf("期望 record status=updating，实际=%s", rec.Status)
	}
	if rec.Source != model.RoleRecordSourceSwitch {
		t.Errorf("期望 source=switch，实际=%s", rec.Source)
	}
}

func TestWriteRoleFieldsTx_NilRole(t *testing.T) {
	db := setupRoleApplyTestDB(t)
	ctx := context.Background()
	inst := model.Instance{Name: "test", InstanceId: "ins-1", RoleID: 1, DistributedRoleVersion: "1.0", RoleSyncStatus: model.RoleSyncStatusUpdated}
	db.Create(&inst)

	recordID, err := writeRoleFieldsTx(ctx, &inst, nil, applyOptions{})
	if err != nil {
		t.Fatalf("writeRoleFieldsTx 失败: %v", err)
	}
	if recordID != 0 {
		t.Errorf("nil role 应返回 recordID=0，实际=%d", recordID)
	}

	var updated model.Instance
	db.First(&updated, inst.ID)
	if updated.RoleID != 0 {
		t.Errorf("期望 role_id=0，实际=%d", updated.RoleID)
	}
	if updated.RoleSyncStatus != "" {
		t.Errorf("期望 role_sync_status=空，实际=%s", updated.RoleSyncStatus)
	}
}

func TestCreateSkillDiffInstallations(t *testing.T) {
	db := setupRoleApplyTestDB(t)
	db.Create(&model.OpenClawRole{ID: 1, Name: "test", Version: "1.0"})
	db.Create(&model.OpenClawRoleSkill{OpenClawRoleID: 1, Slug: "s1", Version: "1.0"})
	db.Create(&model.OpenClawRoleSkill{OpenClawRoleID: 1, Slug: "s2", Version: "2.0"})

	ids, err := createSkillDiffInstallations(db, 100, 1)
	if err != nil {
		t.Fatalf("createSkillDiffInstallations 失败: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("期望 2 个 skill installation，实际 %d", len(ids))
	}

	var count int64
	db.Model(&model.SkillInstallation{}).Where("instance_id = ?", 100).Count(&count)
	if count != 2 {
		t.Errorf("期望 2 条 SkillInstallation，实际 %d", count)
	}
}

func TestCreateSkillDiffInstallations_NoSkills(t *testing.T) {
	db := setupRoleApplyTestDB(t)
	db.Create(&model.OpenClawRole{ID: 1, Name: "test", Version: "1.0"})

	ids, err := createSkillDiffInstallations(db, 100, 1)
	if err != nil {
		t.Fatalf("createSkillDiffInstallations 失败: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("期望 0 个 skill installation，实际 %d", len(ids))
	}
}

func TestCreateRoleDistributionRecord(t *testing.T) {
	db := setupRoleApplyTestDB(t)
	inst := &model.Instance{InstanceId: "ins-1"}
	role := &model.OpenClawRole{ID: 1, Name: "analyst", Version: "1.0"}
	opts := applyOptions{OperatorID: 5, Source: model.RoleRecordSourceDistribute}

	rec, err := createRoleDistributionRecord(db, inst, role, opts, []uint{1, 2, 3})
	if err != nil {
		t.Fatalf("createRoleDistributionRecord 失败: %v", err)
	}
	if rec.ID == 0 {
		t.Fatal("期望 ID > 0")
	}
	if rec.InstanceID != 0 {
		t.Errorf("期望 instance_id=0（inst 无 ID），实际=%d", rec.InstanceID)
	}
	if rec.InstanceCID != "ins-1" {
		t.Errorf("期望 instance_cid=ins-1，实际=%s", rec.InstanceCID)
	}
	if rec.RoleID != 1 {
		t.Errorf("期望 role_id=1，实际=%d", rec.RoleID)
	}
	if rec.RoleName != "analyst" {
		t.Errorf("期望 role_name=analyst，实际=%s", rec.RoleName)
	}
	if rec.Version != "1.0" {
		t.Errorf("期望 version=1.0，实际=%s", rec.Version)
	}
	if rec.OperatorID != 5 {
		t.Errorf("期望 operator_id=5，实际=%d", rec.OperatorID)
	}
	if rec.Source != model.RoleRecordSourceDistribute {
		t.Errorf("期望 source=distribute，实际=%s", rec.Source)
	}
	if rec.Status != model.RoleRecordStatusUpdating {
		t.Errorf("期望 status=updating，实际=%s", rec.Status)
	}
	if rec.SoulStatus != model.RoleSubStatusPending {
		t.Errorf("期望 soul_status=pending，实际=%s", rec.SoulStatus)
	}
	if rec.SkillStatus != model.RoleSubStatusPending {
		t.Errorf("期望 skill_status=pending，实际=%s", rec.SkillStatus)
	}
	if rec.SkillInstallationIDs != "[1,2,3]" {
		t.Errorf("期望 skill_installation_ids=[1,2,3]，实际=%s", rec.SkillInstallationIDs)
	}
}

func TestPipelineCheckInstance_UpdatingInProgress(t *testing.T) {
	// updating_in_progress 检查在 not_running 之后，所以需要先通过 running 检查
	// 用一个返回 running 的 mock resolver
	mockResolver := &mockStatusResolver{status: model.StatusRunning}
	inst := &model.Instance{AgentType: "openclaw", RoleSyncStatus: model.RoleSyncStatusUpdating}
	reason := pipelineCheckInstance(context.Background(), inst, &model.OpenClawRole{ID: 1}, applyModeDistribute, mockResolver)
	if reason != skipReasonUpdatingInProgress {
		t.Errorf("期望 updating_in_progress，实际=%s", reason)
	}
}

// mockStatusResolver 返回固定状态，不调 CVM API。
type mockStatusResolver struct {
	status string
}

func (m *mockStatusResolver) ResolveStatus(ctx context.Context, inst *model.Instance) (InstanceStatusResponse, error) {
	return InstanceStatusResponse{Status: m.status}, nil
}

// ─── waitForRoleSkillsSynced ────────────────────────────────────────────────

func TestWaitForRoleSkillsSynced_AllReady(t *testing.T) {
	db := setupRoleApplyTestDB(t)
	db.Create(&model.OpenClawRoleSkill{OpenClawRoleID: 1, Slug: "s1", CosZipKey: "key1"})
	db.Create(&model.OpenClawRoleSkill{OpenClawRoleID: 1, Slug: "s2", CosZipKey: "key2"})

	got := waitForRoleSkillsSynced(context.Background(), 1)
	if !got {
		t.Error("期望 true（全部就绪），实际 false")
	}
}

func TestWaitForRoleSkillsSynced_NotReadyThenReady(t *testing.T) {
	db := setupRoleApplyTestDB(t)
	// 技能 cos_zip_key 为空
	db.Create(&model.OpenClawRoleSkill{OpenClawRoleID: 1, Slug: "s1", CosZipKey: ""})

	// 缩短轮询间隔
	origInterval := roleSkillsSyncPollInterval
	roleSkillsSyncPollInterval = 50 * time.Millisecond
	t.Cleanup(func() { roleSkillsSyncPollInterval = origInterval })

	// 异步：短暂等待后填充 cos_zip_key
	go func() {
		time.Sleep(100 * time.Millisecond)
		db.Model(&model.OpenClawRoleSkill{}).Where("open_claw_role_id = ?", 1).Update("cos_zip_key", "filled")
	}()

	got := waitForRoleSkillsSynced(context.Background(), 1)
	if !got {
		t.Error("期望 true（轮询后就绪），实际 false")
	}
}

func TestWaitForRoleSkillsSynced_Timeout(t *testing.T) {
	db := setupRoleApplyTestDB(t)
	db.Create(&model.OpenClawRoleSkill{OpenClawRoleID: 1, Slug: "s1", CosZipKey: ""})

	// 缩短超时
	origMax := roleSkillsSyncMaxWait
	origInterval := roleSkillsSyncPollInterval
	roleSkillsSyncMaxWait = 200 * time.Millisecond
	roleSkillsSyncPollInterval = 50 * time.Millisecond
	t.Cleanup(func() {
		roleSkillsSyncMaxWait = origMax
		roleSkillsSyncPollInterval = origInterval
	})

	got := waitForRoleSkillsSynced(context.Background(), 1)
	if got {
		t.Error("期望 false（超时未就绪），实际 true")
	}
}

// ─── refreshSkillInstallationsCosZipKey ──────────────────────────────────────

func TestRefreshSkillInstallationsCosZipKey_UpdatesEmpty(t *testing.T) {
	db := setupRoleApplyTestDB(t)
	ctx := context.Background()

	// 角色技能有 cos_zip_key
	db.Create(&model.OpenClawRoleSkill{OpenClawRoleID: 1, Slug: "s1", CosZipKey: "key1"})
	db.Create(&model.OpenClawRoleSkill{OpenClawRoleID: 1, Slug: "s2", CosZipKey: "key2"})

	// SkillInstallation 快照时 cos_zip_key 为空
	db.Create(&model.SkillInstallation{InstanceID: 100, Slug: "s1", CosZipKey: "", InstallStatus: model.SkillInstallNone})
	db.Create(&model.SkillInstallation{InstanceID: 100, Slug: "s2", CosZipKey: "", InstallStatus: model.SkillInstallNone})

	refreshSkillInstallationsCosZipKey(ctx, 100, 1)

	var installs []model.SkillInstallation
	db.Where("instance_id = ?", 100).Find(&installs)
	for _, si := range installs {
		if si.CosZipKey == "" {
			t.Errorf("slug=%s 的 cos_zip_key 应被刷新，但仍为空", si.Slug)
		}
	}
}

func TestRefreshSkillInstallationsCosZipKey_NoopWhenFilled(t *testing.T) {
	db := setupRoleApplyTestDB(t)
	ctx := context.Background()

	db.Create(&model.OpenClawRoleSkill{OpenClawRoleID: 1, Slug: "s1", CosZipKey: "new-key"})
	// SkillInstallation 已有 cos_zip_key（不应被覆盖）
	db.Create(&model.SkillInstallation{InstanceID: 100, Slug: "s1", CosZipKey: "old-key", InstallStatus: model.SkillInstallNone})

	refreshSkillInstallationsCosZipKey(ctx, 100, 1)

	var si model.SkillInstallation
	db.Where("instance_id = ? AND slug = ?", 100, "s1").First(&si)
	if si.CosZipKey != "old-key" {
		t.Errorf("已填充的 cos_zip_key 不应被覆盖，期望 old-key，实际=%s", si.CosZipKey)
	}
}

func TestRefreshSkillInstallationsCosZipKey_NoMatchingRoleSkill(t *testing.T) {
	db := setupRoleApplyTestDB(t)
	ctx := context.Background()

	// 角色技能 slug 不匹配 SkillInstallation
	db.Create(&model.OpenClawRoleSkill{OpenClawRoleID: 1, Slug: "s1", CosZipKey: "key1"})
	db.Create(&model.SkillInstallation{InstanceID: 100, Slug: "s-other", CosZipKey: "", InstallStatus: model.SkillInstallNone})

	refreshSkillInstallationsCosZipKey(ctx, 100, 1)

	var si model.SkillInstallation
	db.Where("instance_id = ? AND slug = ?", 100, "s-other").First(&si)
	if si.CosZipKey != "" {
		t.Errorf("无匹配角色技能时 cos_zip_key 不应被修改，期望空，实际=%s", si.CosZipKey)
	}
}
