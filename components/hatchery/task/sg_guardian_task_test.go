package task

import (
	"context"
	"errors"
	"testing"
	"time"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// ============================================================================
// task/sg_guardian_task.go 单元测试
//
// 覆盖目标：guardianTick / guardianReconcileCVMCount / guardianHealDrift /
// guardianDetectOrphans / guardianHealMissingSG / guardianMigrateInstances
// 靠 test hook（describeAssocStatsFn / describeSGNamesFn / createCloudSGWithRetryFn
// / applyRulesToCloudSGWithRetryFn / guardianApplyRulesFn / guardianTryDeleteFn
// / guardianGetDefaultRuleSetFn / sgGuardianLogAuditFn）拦截所有云 API 与
// 异步审计，避免真实 SDK 调用和 goroutine 写旧 DB。
// ============================================================================

// setupGuardianDB 在内存 SQLite 建齐 Guardian 需要的表，返回 teardown 闭包。
// 同时把所有外部副作用（云 API / 异步审计）替换为可控实现或 no-op。
func setupGuardianDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.RuleSet{},
		&model.ManagedSGPool{},
		&model.SiteConfig{},
		&model.Instance{},
		&model.SGDrainState{},
		&model.AuditLog{},
	); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	restoreDB := model.UseDBForTest(db)

	// 备份并替换所有 Guardian 外部依赖，cleanup 时还原。
	origDescribe := describeAssocStatsFn
	origCreate := createCloudSGWithRetryFn
	origApplyRetry := applyRulesToCloudSGWithRetryFn
	origApply := guardianApplyRulesFn
	origTryDelete := guardianTryDeleteFn
	origGetRS := guardianGetDefaultRuleSetFn
	origAudit := sgGuardianLogAuditFn
	origDescribeNames := describeSGNamesFn
	origMigrate := migrateInstanceSGsFn
	origConfirmMissing := confirmSGMissingFn

	// 默认 stub：空返回 / 成功
	describeAssocStatsFn = func(_ context.Context, _ []string) (map[string]int, error) {
		return map[string]int{}, nil
	}
	createCloudSGWithRetryFn = func(_ context.Context, _, _ string) (string, error) { return "sg-new", nil }
	applyRulesToCloudSGWithRetryFn = func(_ context.Context, _, _ string) error { return nil }
	guardianApplyRulesFn = func(_ context.Context, _, _ string) error { return nil }
	guardianTryDeleteFn = func(_ context.Context, _ string) {}
	guardianGetDefaultRuleSetFn = func(_ context.Context) (*model.RuleSet, error) {
		var rs model.RuleSet
		if err := model.DB(context.Background()).Where("is_default = ?", true).First(&rs).Error; err != nil {
			return nil, err
		}
		return &rs, nil
	}
	describeSGNamesFn = func(_ context.Context, _ []string) (map[string]string, error) { return map[string]string{}, nil }
	// 默认信任 describeSGNamesFn 的判定（老测试通过空 cloudSet 即视为失踪）。
	// 需要单独验证"二次确认"逻辑的测试请覆盖 confirmSGMissingFn。
	confirmSGMissingFn = func(_ context.Context, _ string) (bool, error) { return true, nil }
	// 同步 no-op，避免 goroutine 访问 cleanup 后的 DB
	sgGuardianLogAuditFn = func(_ context.Context, _ time.Time, _ uint, _, _, _, _, _ string) {}
	// 默认：云换绑直接成功；每个测试按需再覆盖
	migrateInstanceSGsFn = func(_ context.Context, _ string, _ []string) error { return nil }

	t.Cleanup(func() {
		describeAssocStatsFn = origDescribe
		createCloudSGWithRetryFn = origCreate
		applyRulesToCloudSGWithRetryFn = origApplyRetry
		guardianApplyRulesFn = origApply
		guardianTryDeleteFn = origTryDelete
		guardianGetDefaultRuleSetFn = origGetRS
		sgGuardianLogAuditFn = origAudit
		describeSGNamesFn = origDescribeNames
		migrateInstanceSGsFn = origMigrate
		confirmSGMissingFn = origConfirmMissing
		restoreDB()
	})
	return db
}

// seedGuardianRuleSet 插入一行 default RuleSet，返回其指针。
func seedGuardianRuleSet(t *testing.T, db *gorm.DB) *model.RuleSet {
	t.Helper()
	rs := &model.RuleSet{
		Name:         model.DefaultRuleSetName,
		Description:  "guardian test",
		Rules:        "[]",
		Version:      2,
		UserGroupIDs: "[]",
		IsDefault:    true,
	}
	if err := db.Create(rs).Error; err != nil {
		t.Fatalf("seed rule set: %v", err)
	}
	return rs
}

// ------------------------------------------------------------
// poolSGIDs / isManagedPoolSG / truncateID
// ------------------------------------------------------------

func TestPoolSGIDs(t *testing.T) {
	in := []model.ManagedSGPool{
		{SGID: "sg-a"}, {SGID: "sg-b"}, {SGID: "sg-c"},
	}
	got := poolSGIDs(in)
	if len(got) != 3 || got[0] != "sg-a" || got[2] != "sg-c" {
		t.Errorf("poolSGIDs unexpected: %v", got)
	}
	if len(poolSGIDs(nil)) != 0 {
		t.Error("empty input should return empty slice")
	}
}

func TestIsManagedPoolSG(t *testing.T) {
	if !isManagedPoolSG(model.ManagedSGPool{RuleSetID: 1}) {
		t.Error("RuleSetID>0 should be managed")
	}
	if isManagedPoolSG(model.ManagedSGPool{RuleSetID: 0}) {
		t.Error("RuleSetID=0 should be treated as legacy (not managed)")
	}
}

func TestTruncateID(t *testing.T) {
	if got := truncateID("abcdefghij", 5); got != "abcde" {
		t.Errorf("truncateID cut: got %q", got)
	}
	if got := truncateID("abc", 10); got != "abc" {
		t.Errorf("truncateID short: got %q", got)
	}
}

// ------------------------------------------------------------
// guardianReconcileCVMCount
// ------------------------------------------------------------

func TestGuardianReconcileCVMCount_UsesMaxOfDBAndCloud(t *testing.T) {
	db := setupGuardianDB(t)
	rs := seedGuardianRuleSet(t, db)

	// 池里 sg-x cvm_count=5，但 DB 中绑着 7 个实例，云端统计是 6 → 应取 max=7
	sg := model.ManagedSGPool{
		SGID: "sg-x", RuleSetID: rs.ID, Status: model.SGStatusActive, CVMCount: 5,
	}
	db.Create(&sg)
	for i := 0; i < 7; i++ {
		db.Create(&model.Instance{
			InstanceId:      "ins-" + string(rune('a'+i)),
			SecurityGroupId: "sg-x",
		})
	}
	describeAssocStatsFn = func(_ context.Context, sgIDs []string) (map[string]int, error) {
		return map[string]int{"sg-x": 6}, nil
	}

	guardianReconcileCVMCount(context.Background())

	var after model.ManagedSGPool
	db.Where("sg_id = ?", "sg-x").First(&after)
	if after.CVMCount != 7 {
		t.Errorf("expected cvm_count=7 (max(db=7, cloud=6)), got %d", after.CVMCount)
	}
}

func TestGuardianReconcileCVMCount_NoChange(t *testing.T) {
	db := setupGuardianDB(t)
	rs := seedGuardianRuleSet(t, db)
	sg := model.ManagedSGPool{
		SGID: "sg-y", RuleSetID: rs.ID, Status: model.SGStatusActive, CVMCount: 3,
	}
	db.Create(&sg)
	for i := 0; i < 3; i++ {
		db.Create(&model.Instance{
			InstanceId:      "iy-" + string(rune('a'+i)),
			SecurityGroupId: "sg-y",
		})
	}
	describeAssocStatsFn = func(_ context.Context, _ []string) (map[string]int, error) {
		return map[string]int{"sg-y": 3}, nil
	}

	before := time.Now()
	guardianReconcileCVMCount(context.Background())
	_ = before

	var after model.ManagedSGPool
	db.Where("sg_id = ?", "sg-y").First(&after)
	if after.CVMCount != 3 {
		t.Errorf("expected cvm_count unchanged=3, got %d", after.CVMCount)
	}
}

func TestGuardianReconcileCVMCount_EmptyPoolNoOp(t *testing.T) {
	_ = setupGuardianDB(t)
	// 不建任何 SG
	guardianReconcileCVMCount(context.Background()) // 不应 panic
}

func TestGuardianReconcileCVMCount_CloudAPIFailStillUsesDB(t *testing.T) {
	db := setupGuardianDB(t)
	rs := seedGuardianRuleSet(t, db)
	sg := model.ManagedSGPool{
		SGID: "sg-z", RuleSetID: rs.ID, Status: model.SGStatusActive, CVMCount: 0,
	}
	db.Create(&sg)
	for i := 0; i < 2; i++ {
		db.Create(&model.Instance{
			InstanceId:      "iz-" + string(rune('a'+i)),
			SecurityGroupId: "sg-z",
		})
	}
	describeAssocStatsFn = func(_ context.Context, _ []string) (map[string]int, error) {
		return nil, errors.New("cloud api down")
	}

	guardianReconcileCVMCount(context.Background())

	var after model.ManagedSGPool
	db.Where("sg_id = ?", "sg-z").First(&after)
	if after.CVMCount != 2 {
		t.Errorf("cloud fail fallback to DB=2, got %d", after.CVMCount)
	}
}

// ------------------------------------------------------------
// guardianHealDrift
// ------------------------------------------------------------

func TestGuardianHealDrift_UpdatesRuleVersion(t *testing.T) {
	db := setupGuardianDB(t)
	rs := seedGuardianRuleSet(t, db)

	// 一个 ACTIVE 但 rule_version 落后的 SG
	sg := model.ManagedSGPool{
		SGID: "sg-drift1", RuleSetID: rs.ID, Status: model.SGStatusActive,
		RuleVersion: 1, // 落后于 rs.Version=2
	}
	db.Create(&sg)

	var applied []string
	guardianApplyRulesFn = func(_ context.Context, sgID, rulesJSON string) error {
		applied = append(applied, sgID)
		return nil
	}

	guardianHealDrift(context.Background())

	if len(applied) != 1 || applied[0] != "sg-drift1" {
		t.Errorf("expected ApplyRules called for sg-drift1, got %v", applied)
	}
	var after model.ManagedSGPool
	db.First(&after, sg.ID)
	if after.RuleVersion != rs.Version {
		t.Errorf("expected rule_version=%d, got %d", rs.Version, after.RuleVersion)
	}
}

func TestGuardianHealDrift_ApplyFailsMarksDrift(t *testing.T) {
	db := setupGuardianDB(t)
	rs := seedGuardianRuleSet(t, db)

	sg := model.ManagedSGPool{
		SGID: "sg-drift-fail", RuleSetID: rs.ID, Status: model.SGStatusActive,
		RuleVersion: 1,
	}
	db.Create(&sg)

	guardianApplyRulesFn = func(_ context.Context, _, _ string) error { return errors.New("policy API down") }

	guardianHealDrift(context.Background())

	var after model.ManagedSGPool
	db.First(&after, sg.ID)
	if after.DriftAt == nil {
		t.Error("expected drift_at to be set after apply failure")
	}
	if after.RuleVersion != 1 {
		t.Errorf("rule_version should not advance on failure, got %d", after.RuleVersion)
	}
}

func TestGuardianHealDrift_NoCandidatesReturnsFast(t *testing.T) {
	db := setupGuardianDB(t)
	seedGuardianRuleSet(t, db)
	// 不建任何 drift SG
	called := 0
	guardianApplyRulesFn = func(_ context.Context, _, _ string) error { called++; return nil }
	guardianHealDrift(context.Background())
	if called != 0 {
		t.Errorf("expected no apply calls, got %d", called)
	}
}

func TestGuardianHealDrift_GetRuleSetFails(t *testing.T) {
	db := setupGuardianDB(t)
	_ = db
	guardianGetDefaultRuleSetFn = func(_ context.Context) (*model.RuleSet, error) {
		return nil, errors.New("ruleset not ready")
	}
	called := 0
	guardianApplyRulesFn = func(_ context.Context, _, _ string) error { called++; return nil }
	guardianHealDrift(context.Background()) // 应 warn + return，无 panic
	if called != 0 {
		t.Error("apply should not be called when rule set unavailable")
	}
}

// ------------------------------------------------------------
// guardianDetectOrphans
// ------------------------------------------------------------

// 新模型下不再有"云端 orphan 告警"：DB 是唯一真相源，不再反向扫全账号 SG。
// 此测试验证：即使云端有 DB 不认识的 SG，guardianDetectOrphans 也不应触发任何
// orphan 告警。
func TestGuardianDetectOrphans_NoOrphanAlertByNewModel(t *testing.T) {
	db := setupGuardianDB(t)
	rs := seedGuardianRuleSet(t, db)
	// DB 里只有 sg-a；describeSGNames 回答 sg-a 存在
	db.Create(&model.ManagedSGPool{SGID: "sg-a", RuleSetID: rs.ID, Status: model.SGStatusActive})

	describeSGNamesFn = func(_ context.Context, ids []string) (map[string]string, error) {
		out := map[string]string{}
		for _, id := range ids {
			if id == "sg-a" {
				out[id] = "clawpro-sg-acme-default-01"
			}
		}
		return out, nil
	}
	auditActions := []string{}
	sgGuardianLogAuditFn = func(_ context.Context, _ time.Time, _ uint, _, action, _, _, _ string) {
		auditActions = append(auditActions, action)
	}

	guardianDetectOrphans(context.Background())

	for _, a := range auditActions {
		if a == "alert_orphan_sg" {
			t.Errorf("orphan alert should be obsolete in new model, got actions=%v", auditActions)
		}
	}
}

func TestGuardianDetectOrphans_DBActiveMissingTriggersHeal(t *testing.T) {
	db := setupGuardianDB(t)
	rs := seedGuardianRuleSet(t, db)
	// DB 里有 ACTIVE sg-missing；describeSGNames 返回空（云端查不到）→ 触发自愈
	db.Create(&model.ManagedSGPool{
		SGID:      "sg-missing",
		RuleSetID: rs.ID,
		Status:    model.SGStatusActive,
	})

	describeSGNamesFn = func(_ context.Context, _ []string) (map[string]string, error) { return map[string]string{}, nil }
	// 自愈流程：创建新 SG + apply rules + 迁移
	createCloudSGWithRetryFn = func(_ context.Context, _, _ string) (string, error) { return "sg-healed", nil }
	applyRulesToCloudSGWithRetryFn = func(_ context.Context, _, _ string) error { return nil }

	guardianDetectOrphans(context.Background())

	// 验证新 SG 已入池
	var newSG model.ManagedSGPool
	if err := db.Where("sg_id = ?", "sg-healed").First(&newSG).Error; err != nil {
		t.Fatalf("expected sg-healed to be inserted: %v", err)
	}
	// 旧 SG 应被标记 RETIRED
	var old model.ManagedSGPool
	db.Where("sg_id = ?", "sg-missing").First(&old)
	if old.Status != model.SGStatusRetired {
		t.Errorf("expected old SG RETIRED, got %s", old.Status)
	}
}

func TestGuardianDetectOrphans_FrozenMissingMarkedRetired(t *testing.T) {
	db := setupGuardianDB(t)
	rs := seedGuardianRuleSet(t, db)
	db.Create(&model.ManagedSGPool{
		SGID: "sg-frozen-gone", RuleSetID: rs.ID, Status: model.SGStatusFrozen,
	})
	describeSGNamesFn = func(_ context.Context, _ []string) (map[string]string, error) { return map[string]string{}, nil }

	guardianDetectOrphans(context.Background())

	var after model.ManagedSGPool
	db.Where("sg_id = ?", "sg-frozen-gone").First(&after)
	if after.Status != model.SGStatusRetired {
		t.Errorf("expected FROZEN missing → RETIRED, got %s", after.Status)
	}
}

func TestGuardianDetectOrphans_DrainingMissingMarkedRetired(t *testing.T) {
	db := setupGuardianDB(t)
	rs := seedGuardianRuleSet(t, db)
	db.Create(&model.ManagedSGPool{
		SGID: "sg-draining-gone", RuleSetID: rs.ID, Status: model.SGStatusDraining,
	})
	describeSGNamesFn = func(_ context.Context, _ []string) (map[string]string, error) { return map[string]string{}, nil }

	guardianDetectOrphans(context.Background())

	var after model.ManagedSGPool
	db.Where("sg_id = ?", "sg-draining-gone").First(&after)
	if after.Status != model.SGStatusRetired {
		t.Errorf("expected DRAINING missing → RETIRED, got %s", after.Status)
	}
}

// 老 base SG 名字不以 clawpro-sg-* 开头但云端存在 → 不应被误判为消失
func TestGuardianDetectOrphans_FrozenOldBaseNotMisjudgedMissing(t *testing.T) {
	db := setupGuardianDB(t)
	rs := seedGuardianRuleSet(t, db)
	// 模拟初始化导入的老 base SG，名字不匹配 clawpro-sg-* 前缀
	db.Create(&model.ManagedSGPool{
		SGID: "sg-old-base", SGName: "clawpro-create",
		RuleSetID: rs.ID, Status: model.SGStatusFrozen, CVMCount: 53,
	})

	// describeSGNamesFn 按 sg_id 精确查 → 老 base 云端存在
	// 名字前缀不再作为判断依据，只要 describe 能返回即判存活
	describeSGNamesFn = func(_ context.Context, sgIDs []string) (map[string]string, error) {
		out := map[string]string{}
		for _, id := range sgIDs {
			if id == "sg-old-base" {
				out[id] = "clawpro-create"
			}
		}
		return out, nil
	}

	guardianDetectOrphans(context.Background())

	// 应仍为 FROZEN，不被误杀为 RETIRED
	var after model.ManagedSGPool
	db.Where("sg_id = ?", "sg-old-base").First(&after)
	if after.Status != model.SGStatusFrozen {
		t.Errorf("expected old base SG to remain FROZEN, got %s", after.Status)
	}
	if after.CVMCount != 53 {
		t.Errorf("expected cvm_count unchanged=53, got %d", after.CVMCount)
	}
}

func TestGuardianDetectOrphans_LegacyWithoutRuleSetIDSkipped(t *testing.T) {
	db := setupGuardianDB(t)
	seedGuardianRuleSet(t, db)
	// 存量数据（无 RuleSetID）不应参与自愈/告警
	db.Create(&model.ManagedSGPool{SGID: "sg-legacy", RuleSetID: 0, Status: model.SGStatusActive})
	describeSGNamesFn = func(_ context.Context, _ []string) (map[string]string, error) { return map[string]string{}, nil }

	healCalled := 0
	createCloudSGWithRetryFn = func(_ context.Context, _, _ string) (string, error) {
		healCalled++
		return "sg-x", nil
	}

	guardianDetectOrphans(context.Background())

	if healCalled != 0 {
		t.Errorf("legacy SG should not trigger heal, got %d heals", healCalled)
	}
}

// ------------------------------------------------------------
// guardianHealMissingSG 失败路径：createCloudSG 失败 → fallback (RETIRED)
// ------------------------------------------------------------

func TestGuardianHealMissingSG_CreateFailsFallbackRetires(t *testing.T) {
	db := setupGuardianDB(t)
	rs := seedGuardianRuleSet(t, db)
	old := model.ManagedSGPool{SGID: "sg-fail", RuleSetID: rs.ID, Status: model.SGStatusActive}
	db.Create(&old)

	createCloudSGWithRetryFn = func(_ context.Context, _, _ string) (string, error) { return "", errors.New("quota exceeded") }

	guardianHealMissingSG(context.Background(), old, rs)

	var after model.ManagedSGPool
	db.First(&after, old.ID)
	if after.Status != model.SGStatusRetired {
		t.Errorf("expected RETIRED fallback, got %s", after.Status)
	}
}

func TestGuardianHealMissingSG_ApplyFailsTriesDeleteAndRetires(t *testing.T) {
	db := setupGuardianDB(t)
	rs := seedGuardianRuleSet(t, db)
	old := model.ManagedSGPool{SGID: "sg-apply-fail", RuleSetID: rs.ID, Status: model.SGStatusActive}
	db.Create(&old)

	createCloudSGWithRetryFn = func(_ context.Context, _, _ string) (string, error) { return "sg-new-zzz", nil }
	applyRulesToCloudSGWithRetryFn = func(_ context.Context, _, _ string) error { return errors.New("rule conflict") }

	deleted := ""
	guardianTryDeleteFn = func(_ context.Context, sgID string) { deleted = sgID }

	guardianHealMissingSG(context.Background(), old, rs)

	if deleted != "sg-new-zzz" {
		t.Errorf("expected TryDelete called for new sg, got %q", deleted)
	}
	var after model.ManagedSGPool
	db.First(&after, old.ID)
	if after.Status != model.SGStatusRetired {
		t.Errorf("expected RETIRED after apply fail, got %s", after.Status)
	}
}

// ------------------------------------------------------------
// guardianMigrateInstances
// ------------------------------------------------------------

func TestGuardianMigrateInstances_EmptyPool(t *testing.T) {
	_ = setupGuardianDB(t)
	migrated, failed := guardianMigrateInstances(context.Background(), "sg-old", "sg-new")
	if migrated != 0 || failed != 0 {
		t.Errorf("empty should return 0,0 got %d,%d", migrated, failed)
	}
}

func TestGuardianMigrateInstances_SkipsEmptyInstanceID(t *testing.T) {
	db := setupGuardianDB(t)
	// 插入 2 条无 instance_id 的实例（异常数据），绑在 sg-old 上。
	// 这类脏数据没有办法定位到云端实例，Guardian 不应该尝试迁移——
	// guardianMigrateInstances 的 DB 查询显式过滤了 instance_id != ''，
	// 让这 2 行原封不动地留在 sg-old 上，由其他清理路径处理。
	for i := 0; i < 2; i++ {
		db.Create(&model.Instance{InstanceId: "", SecurityGroupId: "sg-old"})
	}

	migrated, failed := guardianMigrateInstances(context.Background(), "sg-old", "sg-new")
	if migrated != 0 || failed != 0 {
		t.Errorf("expected migrated=0 failed=0 (empty instance_id must be skipped), got %d/%d", migrated, failed)
	}

	// DB 中这 2 行应当仍然绑在 sg-old 上，不应被改到 sg-new
	var stayOld int64
	db.Model(&model.Instance{}).Where("security_group_id = ?", "sg-old").Count(&stayOld)
	if stayOld != 2 {
		t.Errorf("expected 2 empty-instance_id rows to stay on sg-old, got %d", stayOld)
	}
	var movedNew int64
	db.Model(&model.Instance{}).Where("security_group_id = ?", "sg-new").Count(&movedNew)
	if movedNew != 0 {
		t.Errorf("expected 0 rows on sg-new, got %d", movedNew)
	}
}

// ------------------------------------------------------------
// guardianCheckZeroActiveSG
// ------------------------------------------------------------

func TestGuardianCheckZeroActiveSG_Alerts(t *testing.T) {
	db := setupGuardianDB(t)
	rs := seedGuardianRuleSet(t, db)
	// 无 ACTIVE SG，应告警

	var auditActions []string
	sgGuardianLogAuditFn = func(_ context.Context, _ time.Time, _ uint, _, action, _, _, _ string) {
		auditActions = append(auditActions, action)
	}

	guardianCheckZeroActiveSG(context.Background(), rs.ID)

	if len(auditActions) == 0 || auditActions[0] != "alert_no_active_sg" {
		t.Errorf("expected alert_no_active_sg, got %v", auditActions)
	}
}

func TestGuardianCheckZeroActiveSG_HasActiveNoAlert(t *testing.T) {
	db := setupGuardianDB(t)
	rs := seedGuardianRuleSet(t, db)
	db.Create(&model.ManagedSGPool{SGID: "sg-ok", RuleSetID: rs.ID, Status: model.SGStatusActive})

	count := 0
	sgGuardianLogAuditFn = func(_ context.Context, _ time.Time, _ uint, _, _, _, _, _ string) { count++ }

	guardianCheckZeroActiveSG(context.Background(), rs.ID)

	if count != 0 {
		t.Errorf("should not alert when ACTIVE exists, got %d audits", count)
	}
}

// ------------------------------------------------------------
// guardianSyncSGNames
// ------------------------------------------------------------

func TestGuardianSyncSGNames_UpdatesChangedAndFillsEmpty(t *testing.T) {
	db := setupGuardianDB(t)
	rs := seedGuardianRuleSet(t, db)
	// 一条 SGName 为空（初始化导入老 base） + 一条 stale
	db.Create(&model.ManagedSGPool{
		SGID: "sg-empty", SGName: "", RuleSetID: rs.ID,
		Status: model.SGStatusFrozen, RuleVersion: 0,
	})
	db.Create(&model.ManagedSGPool{
		SGID: "sg-stale", SGName: "old-name", RuleSetID: rs.ID,
		Status: model.SGStatusActive, RuleVersion: rs.Version,
	})
	describeSGNamesFn = func(_ context.Context, ids []string) (map[string]string, error) {
		return map[string]string{
			"sg-empty": "云端真实名字",
			"sg-stale": "new-name-user-renamed",
		}, nil
	}

	guardianSyncSGNames(context.Background())

	var a, b model.ManagedSGPool
	db.Where("sg_id = ?", "sg-empty").First(&a)
	db.Where("sg_id = ?", "sg-stale").First(&b)
	if a.SGName != "云端真实名字" {
		t.Errorf("sg-empty SGName not filled: %q", a.SGName)
	}
	if b.SGName != "new-name-user-renamed" {
		t.Errorf("sg-stale SGName not updated: %q", b.SGName)
	}
}

func TestGuardianSyncSGNames_NoChangeWhenSame(t *testing.T) {
	db := setupGuardianDB(t)
	rs := seedGuardianRuleSet(t, db)
	db.Create(&model.ManagedSGPool{
		SGID: "sg-same", SGName: "my-sg", RuleSetID: rs.ID,
		Status: model.SGStatusActive, RuleVersion: rs.Version,
	})
	describeSGNamesFn = func(_ context.Context, _ []string) (map[string]string, error) {
		return map[string]string{"sg-same": "my-sg"}, nil
	}

	guardianSyncSGNames(context.Background())
	// 不应 panic；值仍是 my-sg
	var got model.ManagedSGPool
	db.Where("sg_id = ?", "sg-same").First(&got)
	if got.SGName != "my-sg" {
		t.Errorf("SGName drifted: %q", got.SGName)
	}
}

func TestGuardianSyncSGNames_APIError_NoOp(t *testing.T) {
	db := setupGuardianDB(t)
	rs := seedGuardianRuleSet(t, db)
	db.Create(&model.ManagedSGPool{
		SGID: "sg-x", SGName: "orig", RuleSetID: rs.ID,
		Status: model.SGStatusActive, RuleVersion: rs.Version,
	})
	describeSGNamesFn = func(_ context.Context, _ []string) (map[string]string, error) {
		return nil, errors.New("cloud api down")
	}

	guardianSyncSGNames(context.Background())

	var got model.ManagedSGPool
	db.Where("sg_id = ?", "sg-x").First(&got)
	if got.SGName != "orig" {
		t.Errorf("SGName should remain on API failure, got %q", got.SGName)
	}
}

func TestGuardianSyncSGNames_EmptyPoolNoOp(t *testing.T) {
	_ = setupGuardianDB(t)
	guardianSyncSGNames(context.Background()) // 不 panic
}


// ------------------------------------------------------------
// guardianTick 集成：验证 panic recover + 完整 tick 路径
// ------------------------------------------------------------

func TestGuardianTick_EndToEndSmoke(t *testing.T) {
	db := setupGuardianDB(t)
	rs := seedGuardianRuleSet(t, db)
	db.Create(&model.ManagedSGPool{
		SGID: "sg-a", RuleSetID: rs.ID, Status: model.SGStatusActive,
		RuleVersion: rs.Version,
	})
	// 不应 panic
	guardianTick(context.Background())
}

// TestGuardianTick_PanicRecovered 已移除：panic recovery 由 scheduler.executeTask 统一处理，
// 参见 scheduler_test.go 中的相关测试。

// ------------------------------------------------------------
// guardianDrainOrphanInstances
// ------------------------------------------------------------

// 孤儿实例（security_group_id 不在 ACTIVE pool 中）应被迁移到 ACTIVE SG
func TestGuardianDrainOrphanInstances_MigratesOrphans(t *testing.T) {
	db := setupGuardianDB(t)
	rs := seedGuardianRuleSet(t, db)

	// 创建一个 ACTIVE SG
	db.Create(&model.ManagedSGPool{
		SGID: "sg-active-01", SGName: "clawpro-sg-test-01",
		RuleSetID: rs.ID, RuleVersion: 1, Status: model.SGStatusActive, CVMCount: 5,
	})

	// 创建孤儿实例：security_group_id 是老的 base SG，不在 ACTIVE pool
	db.Create(&model.Instance{
		Identifier:      "",
		Name:            "orphan1",
		InstanceId:      "ins-orphan1",
		SecurityGroupId: "sg-old-base",
	})
	db.Create(&model.Instance{
		Identifier:      "",
		Name:            "orphan2",
		InstanceId:      "ins-orphan2",
		SecurityGroupId: "sg-old-base",
	})
	// 创建一个已在 ACTIVE pool 的正常实例，不应被迁移
	db.Create(&model.Instance{
		Identifier:      "",
		Name:            "normal",
		InstanceId:      "ins-normal",
		SecurityGroupId: "sg-active-01",
	})

	// stub drainSelectSGFn
	origDrainSelect := drainSelectSGFn
	drainSelectSGFn = func(_ context.Context, _ string, _ uint) (string, bool, error) {
		return "sg-active-01", false, nil
	}
	t.Cleanup(func() { drainSelectSGFn = origDrainSelect })

	// stub modifyInstanceSGs
	var modifiedInstances []string
	origModify := drainModifyFn
	drainModifyFn = func(_ context.Context, instanceID string, newSGs []string) error {
		modifiedInstances = append(modifiedInstances, instanceID)
		if len(newSGs) != 1 || newSGs[0] != "sg-active-01" {
			t.Errorf("expected newSGs=[sg-active-01], got %v", newSGs)
		}
		return nil
	}
	t.Cleanup(func() { drainModifyFn = origModify })

	guardianDrainOrphanInstances(context.Background())

	// 验证：两个孤儿实例被迁移
	if len(modifiedInstances) != 2 {
		t.Errorf("expected 2 instances migrated, got %d: %v", len(modifiedInstances), modifiedInstances)
	}

	// 验证 DB 更新
	var inst1 model.Instance
	db.Where("instance_id = ?", "ins-orphan1").First(&inst1)
	if inst1.SecurityGroupId != "sg-active-01" {
		t.Errorf("orphan1 sg expected sg-active-01, got %s", inst1.SecurityGroupId)
	}

	var inst2 model.Instance
	db.Where("instance_id = ?", "ins-orphan2").First(&inst2)
	if inst2.SecurityGroupId != "sg-active-01" {
		t.Errorf("orphan2 sg expected sg-active-01, got %s", inst2.SecurityGroupId)
	}

	// 验证正常实例没被修改
	var normal model.Instance
	db.Where("instance_id = ?", "ins-normal").First(&normal)
	if normal.SecurityGroupId != "sg-active-01" {
		t.Errorf("normal instance sg should remain sg-active-01, got %s", normal.SecurityGroupId)
	}

	// 验证 cvm_count 增加了 2
	var pool model.ManagedSGPool
	db.Where("sg_id = ?", "sg-active-01").First(&pool)
	if pool.CVMCount != 7 { // 5 + 2
		t.Errorf("expected cvm_count=7, got %d", pool.CVMCount)
	}
}

// 无可用 ACTIVE SG 时不做任何操作
func TestGuardianDrainOrphanInstances_NoActiveSG_Noop(t *testing.T) {
	db := setupGuardianDB(t)
	seedGuardianRuleSet(t, db)

	// 无 ACTIVE SG，只有 FROZEN
	db.Create(&model.ManagedSGPool{
		SGID: "sg-frozen", SGName: "clawpro-create",
		RuleSetID: 1, Status: model.SGStatusFrozen, CVMCount: 10,
	})

	// 孤儿实例
	db.Create(&model.Instance{
		Identifier:      "",
		Name:            "orphan",
		InstanceId:      "ins-orphan",
		SecurityGroupId: "sg-old",
	})

	// 不应调用 modifyInstanceSGs
	origModify := drainModifyFn
	drainModifyFn = func(_ context.Context, instanceID string, _ []string) error {
		t.Errorf("modifyInstanceSGs should not be called, but called for %s", instanceID)
		return nil
	}
	t.Cleanup(func() { drainModifyFn = origModify })

	guardianDrainOrphanInstances(context.Background())
}

// 云 API 换绑失败时跳过该实例，不影响其他
func TestGuardianDrainOrphanInstances_CloudAPIFails_SkipsInstance(t *testing.T) {
	db := setupGuardianDB(t)
	rs := seedGuardianRuleSet(t, db)

	db.Create(&model.ManagedSGPool{
		SGID: "sg-active", SGName: "clawpro-sg-test-01",
		RuleSetID: rs.ID, RuleVersion: 1, Status: model.SGStatusActive, CVMCount: 0,
	})

	db.Create(&model.Instance{
		Identifier: "", Name: "fail-orphan", InstanceId: "ins-fail",
		SecurityGroupId: "sg-old",
	})

	origDrainSelect := drainSelectSGFn
	drainSelectSGFn = func(_ context.Context, _ string, _ uint) (string, bool, error) {
		return "sg-active", false, nil
	}
	t.Cleanup(func() { drainSelectSGFn = origDrainSelect })

	origModify := drainModifyFn
	drainModifyFn = func(_ context.Context, _ string, _ []string) error {
		return errors.New("cloud API timeout")
	}
	t.Cleanup(func() { drainModifyFn = origModify })

	guardianDrainOrphanInstances(context.Background())

	// DB 应未变
	var inst model.Instance
	db.Where("instance_id = ?", "ins-fail").First(&inst)
	if inst.SecurityGroupId != "sg-old" {
		t.Errorf("expected sg unchanged=sg-old, got %s", inst.SecurityGroupId)
	}
}

// ------------------------------------------------------------
// createCloudSGWithRetry / applyRulesToCloudSGWithRetry
// （测试重试逻辑；注意：retry 使用真实 controller.CreateCloudSG / ApplyRulesToCloudSG，
// 它们依赖云 SDK client，我们无法直接 stub——所以这里只测 happy path 的单次调用）
// ------------------------------------------------------------
