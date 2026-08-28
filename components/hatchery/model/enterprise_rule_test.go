package model

import (
	"context"
	"os"
	"testing"

	"hatchery/common"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupRuleTestDB 独立于 setupTestDB，只挂载 rule 相关 4 张表（+ Instance 供 JOIN 用）。
// 与一期 skill 侧独立 test DB 的做法一致，避免污染全局 setupTestDB。
func setupRuleTestDB(t *testing.T) (cleanup func()) {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "hatchery_rule_test_*.db")
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
	registerIdentifierCallbacks(gdb)

	migrateDB := gdb.WithContext(common.WithSkipIdentifier(context.Background()))
	if err := migrateDB.AutoMigrate(
		&Instance{}, // JOIN target for status queries
		&User{},
		&EnterpriseRule{},
		&RuleDistributionTask{},
		&RuleDistributionRecord{},
		&LocalInstanceRule{},
	); err != nil {
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

func TestBuildRuleInstanceQuery_VersionIsBoundAndOutdatedNeedsLatest(t *testing.T) {
	cleanup := setupRuleTestDB(t)
	defer cleanup()
	ctx := common.WithSkipIdentifier(context.Background())
	db := DB(ctx)

	user := User{Username: "rule-query-user", Password: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	instance := Instance{
		InstanceId: "local-rule-query-1", Name: "Rule Query", UserID: user.ID,
		Source: InstanceSourceLocal,
	}
	if err := db.Create(&instance).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}
	rule := EnterpriseRule{Slug: "query-rule", Name: "Query Rule", Type: EnterpriseRuleTypeRule, Version: "1.0.0"}
	if err := db.Create(&rule).Error; err != nil {
		t.Fatalf("create rule: %v", err)
	}
	task := RuleDistributionTask{RuleID: rule.ID, Slug: rule.Slug, Version: rule.Version, Type: RuleTaskTypeDistribute}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}
	record := RuleDistributionRecord{
		TaskID: task.ID, RuleID: rule.ID, InstanceID: instance.ID, InstanceCID: instance.InstanceId,
		Version: rule.Version, Status: RuleRecordStatusSuccess, Type: RuleTaskTypeDistribute,
	}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("create record: %v", err)
	}
	if err := db.Create(&LocalInstanceRule{
		InstanceID: instance.ID, Slug: rule.Slug, Version: rule.Version,
		RuleType: rule.Type, Source: LocalRuleSourceEnterprise, Scope: LocalSkillScopeUser,
		InstallStatus: LocalSkillInstallStatusDistributed,
	}).Error; err != nil {
		t.Fatalf("create local rule: %v", err)
	}

	type row struct {
		InstanceID uint
		Status     string
	}
	for _, tc := range []struct {
		name          string
		latestVersion string
		wantStatus    string
	}{
		{"newer version", "2.0.0", "outdated"},
		{"empty version", "", "installed"},
		{"invalid version", "1.0.0' OR 1=1 --", "installed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var rows []row
			if err := BuildRuleInstanceQuery(ctx, []uint{rule.ID}, tc.latestVersion, rule.Slug).Scan(&rows).Error; err != nil {
				t.Fatalf("query: %v", err)
			}
			if len(rows) != 1 || rows[0].InstanceID != instance.ID || rows[0].Status != tc.wantStatus {
				t.Fatalf("rows=%#v want status=%s", rows, tc.wantStatus)
			}
		})
	}
}

// TestEnterpriseRule_ParseVersion_HappyPath 覆盖 x.y.z semver 正确解析。
func TestEnterpriseRule_ParseVersion_HappyPath(t *testing.T) {
	r := &EnterpriseRule{Version: "1.2.3"}
	if err := r.ParseVersion(); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if r.VersionMajor != 1 || r.VersionMinor != 2 || r.VersionPatch != 3 {
		t.Fatalf("got major=%d minor=%d patch=%d, want 1/2/3", r.VersionMajor, r.VersionMinor, r.VersionPatch)
	}
}

// TestEnterpriseRule_ParseVersion_BadFormats 覆盖非法输入的三种拒绝分支。
func TestEnterpriseRule_ParseVersion_BadFormats(t *testing.T) {
	cases := []string{"", "1.2", "1.2.3.4", "a.2.3", "1.b.3", "1.2.c"}
	for _, c := range cases {
		r := &EnterpriseRule{Version: c}
		if err := r.ParseVersion(); err == nil {
			t.Fatalf("case %q: want error, got nil", c)
		}
	}
}

// TestRuleVersionScore 覆盖 semver → int 打分。
func TestRuleVersionScore(t *testing.T) {
	cases := []struct {
		v    string
		want int
	}{
		{"1.2.3", 1_002_003},
		{"0.0.0", 0},
		{"9.9.9", 9_009_009},
		{"", 0},
		{"1.2", 0},
		{"a.b.c", 0},
	}
	for _, c := range cases {
		if got := RuleVersionScore(c.v); got != c.want {
			t.Fatalf("RuleVersionScore(%q)=%d, want %d", c.v, got, c.want)
		}
	}
}

// TestRuleTypeCommandName 覆盖 4 种合法组合 + 若干拒绝分支。
func TestRuleTypeCommandName(t *testing.T) {
	cases := []struct {
		recordType, ruleType, want string
	}{
		{"distribute", "prompt", "install_prompt_rule"},
		{"distribute", "rule", "install_rule_rule"},
		{"uninstall", "prompt", "uninstall_prompt_rule"},
		{"uninstall", "rule", "uninstall_rule_rule"},
		// 拒绝分支：未知 recordType
		{"unknown", "prompt", ""},
		{"", "rule", ""},
		// 拒绝分支：未知 ruleType（兼容脏数据）
		{"distribute", "", ""},
		{"distribute", "unknown", ""},
	}
	for _, c := range cases {
		if got := RuleTypeCommandName(c.recordType, c.ruleType); got != c.want {
			t.Fatalf("RuleTypeCommandName(%q,%q)=%q, want %q",
				c.recordType, c.ruleType, got, c.want)
		}
	}
}

// TestLatestVersionRuleIDs 覆盖同 slug 多版本时选出最高版本 id 的行为。
func TestLatestVersionRuleIDs(t *testing.T) {
	cleanup := setupRuleTestDB(t)
	defer cleanup()

	ctx := common.WithSkipIdentifier(context.Background())

	seed := []EnterpriseRule{
		{Slug: "a", Name: "A", Type: "rule", Version: "1.0.0", VersionMajor: 1},
		{Slug: "a", Name: "A", Type: "rule", Version: "1.2.0", VersionMajor: 1, VersionMinor: 2},
		{Slug: "a", Name: "A", Type: "rule", Version: "2.0.0", VersionMajor: 2},
		{Slug: "b", Name: "B", Type: "prompt", Version: "0.5.0", VersionMinor: 5},
		{Slug: "b", Name: "B", Type: "prompt", Version: "0.5.1", VersionMinor: 5, VersionPatch: 1},
	}
	for i := range seed {
		if err := DB(ctx).Create(&seed[i]).Error; err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}

	sub := LatestVersionRuleIDs(ctx)
	var ids []uint
	if err := DB(ctx).Model(&EnterpriseRule{}).
		Where("id IN (?)", sub).Order("id ASC").Pluck("id", &ids).Error; err != nil {
		t.Fatalf("query latest ids: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("want 2 latest ids, got %d: %v", len(ids), ids)
	}
	// 期望：slug=a 最新是 2.0.0（第 3 条 seed），slug=b 最新是 0.5.1（第 5 条 seed）
	if ids[0] != seed[2].ID || ids[1] != seed[4].ID {
		t.Fatalf("want [%d, %d], got %v", seed[2].ID, seed[4].ID, ids)
	}
}

// TestInheritRuleDistributeCount 覆盖从旧版本继承 distribute_count 三种分支：
//   - 有旧版且 count>0 → 继承
//   - 无旧版 → 不继承（保持默认 0）
//   - 有旧版但 count=0 → 不继承（保持默认 0）
func TestInheritRuleDistributeCount(t *testing.T) {
	cleanup := setupRuleTestDB(t)
	defer cleanup()

	ctx := common.WithSkipIdentifier(context.Background())
	tx := DB(ctx)

	// case 1: has previous with count>0
	prev1 := &EnterpriseRule{Slug: "s1", Name: "S1", Type: "rule", Version: "1.0.0", DistributeCount: 7}
	if err := tx.Create(prev1).Error; err != nil {
		t.Fatalf("seed prev1: %v", err)
	}
	newer1 := &EnterpriseRule{Slug: "s1", Name: "S1", Type: "rule", Version: "1.1.0", DistributeCount: 0}
	if err := tx.Create(newer1).Error; err != nil {
		t.Fatalf("seed newer1: %v", err)
	}
	if err := InheritRuleDistributeCount(tx, "s1", newer1.ID); err != nil {
		t.Fatalf("inherit s1: %v", err)
	}
	var got1 EnterpriseRule
	if err := tx.First(&got1, newer1.ID).Error; err != nil {
		t.Fatalf("reload s1: %v", err)
	}
	if got1.DistributeCount != 7 {
		t.Fatalf("want inherited count=7, got %d", got1.DistributeCount)
	}

	// case 2: no previous version
	newer2 := &EnterpriseRule{Slug: "s2", Name: "S2", Type: "rule", Version: "1.0.0", DistributeCount: 0}
	if err := tx.Create(newer2).Error; err != nil {
		t.Fatalf("seed newer2: %v", err)
	}
	if err := InheritRuleDistributeCount(tx, "s2", newer2.ID); err != nil {
		t.Fatalf("inherit s2: %v", err)
	}
	var got2 EnterpriseRule
	if err := tx.First(&got2, newer2.ID).Error; err != nil {
		t.Fatalf("reload s2: %v", err)
	}
	if got2.DistributeCount != 0 {
		t.Fatalf("want 0 (no prev), got %d", got2.DistributeCount)
	}

	// case 3: prev exists but count=0
	prev3 := &EnterpriseRule{Slug: "s3", Name: "S3", Type: "rule", Version: "1.0.0", DistributeCount: 0}
	if err := tx.Create(prev3).Error; err != nil {
		t.Fatalf("seed prev3: %v", err)
	}
	newer3 := &EnterpriseRule{Slug: "s3", Name: "S3", Type: "rule", Version: "1.1.0", DistributeCount: 0}
	if err := tx.Create(newer3).Error; err != nil {
		t.Fatalf("seed newer3: %v", err)
	}
	if err := InheritRuleDistributeCount(tx, "s3", newer3.ID); err != nil {
		t.Fatalf("inherit s3: %v", err)
	}
	var got3 EnterpriseRule
	if err := tx.First(&got3, newer3.ID).Error; err != nil {
		t.Fatalf("reload s3: %v", err)
	}
	if got3.DistributeCount != 0 {
		t.Fatalf("want 0 (prev count=0), got %d", got3.DistributeCount)
	}
}

// TestResolveRuleDistributeFailedStatus 覆盖有历史成功 → upgrade_failed；无 → failed。
func TestResolveRuleDistributeFailedStatus(t *testing.T) {
	cleanup := setupRuleTestDB(t)
	defer cleanup()

	ctx := common.WithSkipIdentifier(context.Background())
	tx := DB(ctx)

	// 无 record → failed（也覆盖 ruleIDs 为空的守卫）
	if got := ResolveRuleDistributeFailedStatus(ctx, 42, nil); got != RuleRecordStatusFailed {
		t.Fatalf("empty ruleIDs: want failed, got %s", got)
	}
	if got := ResolveRuleDistributeFailedStatus(ctx, 42, []uint{1}); got != RuleRecordStatusFailed {
		t.Fatalf("no history: want failed, got %s", got)
	}

	// 有历史 success → upgrade_failed
	if err := tx.Create(&RuleDistributionRecord{
		TaskID: 1, RuleID: 1, InstanceID: 42, Status: RuleRecordStatusSuccess, Type: RuleTaskTypeDistribute,
	}).Error; err != nil {
		t.Fatalf("seed success: %v", err)
	}
	if got := ResolveRuleDistributeFailedStatus(ctx, 42, []uint{1}); got != RuleRecordStatusUpgradeFailed {
		t.Fatalf("has history: want upgrade_failed, got %s", got)
	}

	// 别的实例的 success 不影响
	if got := ResolveRuleDistributeFailedStatus(ctx, 99, []uint{1}); got != RuleRecordStatusFailed {
		t.Fatalf("other instance: want failed, got %s", got)
	}
}

// TestResolveRuleUninstallFailedStatus 覆盖旧版本仍在 → uninstall_failed_old；
// 已装版本==最新版本 → failed；无历史 → failed；ruleIDs 为空 → failed。
func TestResolveRuleUninstallFailedStatus(t *testing.T) {
	cleanup := setupRuleTestDB(t)
	defer cleanup()

	ctx := common.WithSkipIdentifier(context.Background())
	tx := DB(ctx)

	// ruleIDs 为空
	if got := ResolveRuleUninstallFailedStatus(ctx, 1, nil, "1.2.0"); got != RuleRecordStatusFailed {
		t.Fatalf("empty ruleIDs: want failed, got %s", got)
	}

	// 无历史 success
	if got := ResolveRuleUninstallFailedStatus(ctx, 1, []uint{1}, "1.2.0"); got != RuleRecordStatusFailed {
		t.Fatalf("no history: want failed, got %s", got)
	}

	// 有历史 success，Version == latest → failed
	if err := tx.Create(&RuleDistributionRecord{
		TaskID: 1, RuleID: 1, InstanceID: 1, Version: "1.2.0",
		Status: RuleRecordStatusSuccess, Type: RuleTaskTypeDistribute,
	}).Error; err != nil {
		t.Fatalf("seed same-version: %v", err)
	}
	if got := ResolveRuleUninstallFailedStatus(ctx, 1, []uint{1}, "1.2.0"); got != RuleRecordStatusFailed {
		t.Fatalf("same version: want failed, got %s", got)
	}

	// 有历史 success，Version != latest → uninstall_failed_old
	if err := tx.Create(&RuleDistributionRecord{
		TaskID: 2, RuleID: 1, InstanceID: 2, Version: "1.1.0",
		Status: RuleRecordStatusSuccess, Type: RuleTaskTypeDistribute,
	}).Error; err != nil {
		t.Fatalf("seed old-version: %v", err)
	}
	if got := ResolveRuleUninstallFailedStatus(ctx, 2, []uint{1}, "1.2.0"); got != RuleRecordStatusUninstallFailedOld {
		t.Fatalf("old version: want uninstall_failed_old, got %s", got)
	}
}

// TestLocalInstanceRule_UniqueIndex 覆盖 (instance_id, slug) 唯一约束。
func TestLocalInstanceRule_UniqueIndex(t *testing.T) {
	cleanup := setupRuleTestDB(t)
	defer cleanup()

	ctx := common.WithSkipIdentifier(context.Background())
	tx := DB(ctx)

	first := &LocalInstanceRule{InstanceID: 1, Slug: "a", Version: "1.0.0", RuleType: "rule"}
	if err := tx.Create(first).Error; err != nil {
		t.Fatalf("first insert: %v", err)
	}

	// 相同 (instance_id, slug) 再插一条应报错
	dup := &LocalInstanceRule{InstanceID: 1, Slug: "a", Version: "1.1.0", RuleType: "rule"}
	if err := tx.Create(dup).Error; err == nil {
		t.Fatalf("want unique-index error, got nil")
	}

	// 换 slug 或换 instance_id 都不冲突
	ok1 := &LocalInstanceRule{InstanceID: 1, Slug: "b", RuleType: "prompt"}
	if err := tx.Create(ok1).Error; err != nil {
		t.Fatalf("different slug: %v", err)
	}
	ok2 := &LocalInstanceRule{InstanceID: 2, Slug: "a", RuleType: "rule"}
	if err := tx.Create(ok2).Error; err != nil {
		t.Fatalf("different instance_id: %v", err)
	}
}

// TestEnterpriseRule_UniqueIndex 覆盖 (identifier, slug, version) 唯一约束。
func TestEnterpriseRule_UniqueIndex(t *testing.T) {
	cleanup := setupRuleTestDB(t)
	defer cleanup()

	ctx := common.WithSkipIdentifier(context.Background())
	tx := DB(ctx)

	first := &EnterpriseRule{Slug: "s", Name: "S", Type: "rule", Version: "1.0.0"}
	if err := tx.Create(first).Error; err != nil {
		t.Fatalf("first: %v", err)
	}

	dup := &EnterpriseRule{Slug: "s", Name: "S", Type: "rule", Version: "1.0.0"}
	if err := tx.Create(dup).Error; err == nil {
		t.Fatalf("want unique-index error, got nil")
	}

	// 版本不同即可
	ok := &EnterpriseRule{Slug: "s", Name: "S", Type: "rule", Version: "1.1.0"}
	if err := tx.Create(ok).Error; err != nil {
		t.Fatalf("different version: %v", err)
	}
}
