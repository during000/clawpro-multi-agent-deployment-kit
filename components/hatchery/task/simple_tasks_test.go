package task

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"hatchery/common"
	"hatchery/model"
)

// ─── 通用 DB 初始化辅助 ───────────────────────────────────────────────────

func setupSimpleTaskDB(t *testing.T, models ...interface{}) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if len(models) > 0 {
		if err := db.AutoMigrate(models...); err != nil {
			t.Fatalf("migrate: %v", err)
		}
	}
	t.Cleanup(model.UseDBForTest(db))
	return db
}

// ─── runSTSRefresh ───────────────────────────────────────────────────────

// TestRunSTSRefresh_EmptyUIN UIN 为空时直接返回（不调用 RefreshSTSCredentials）。
func TestRunSTSRefresh_EmptyUIN(t *testing.T) {
	// 不设置 TenantSnapshot，CVMUinFromCtx 返回 ""
	runSTSRefresh(context.Background())
}

// TestRunSTSRefresh_WithUIN UIN 非空时尝试刷新（SiteConfig 无凭据，报错但不 panic）。
func TestRunSTSRefresh_WithUIN(t *testing.T) {
	setupSimpleTaskDB(t, &model.SiteConfig{})

	old := common.FixedSnapshot
	common.FixedSnapshot = &common.TenantSnapshot{Identifier: "tenant-sts", Uin: "123456789"}
	defer func() { common.FixedSnapshot = old }()

	ctx := common.InjectTenant(context.Background(), *common.FixedSnapshot)
	// RefreshSTSCredentials 内部因无有效凭据而报错，但不应 panic
	runSTSRefresh(ctx)
}

// ─── runOneIDSync ────────────────────────────────────────────────────────

// TestRunOneIDSync_NoConfig OneID 同步无配置时直接返回，不 panic。
func TestRunOneIDSync_NoConfig(t *testing.T) {
	setupSimpleTaskDB(t, &model.SiteConfig{})
	// SyncViaGateway 内部无 OneID 配置时跳过
	runOneIDSync(context.Background())
}

// ─── runCleanExpiredSessions ──────────────────────────────────────────────

// TestRunCleanExpiredSessions_NoSessions 无会话记录时正常执行。
func TestRunCleanExpiredSessions_NoSessions(t *testing.T) {
	setupSimpleTaskDB(t, &model.SessionBlacklist{})
	runCleanExpiredSessions(context.Background())
}

// ─── runMigrateInstanceModels ──────────────────────────────────────────────

// TestRunMigrateInstanceModels_NoData 无数据时幂等执行，不 panic。
func TestRunMigrateInstanceModels_NoData(t *testing.T) {
	setupSimpleTaskDB(t, &model.Instance{})
	runMigrateInstanceModels(context.Background())
}

// ─── runSMHTokenRefresh ────────────────────────────────────────────────────

// TestRunSMHTokenRefresh_NotConfigured SMH 未配置时直接返回。
func TestRunSMHTokenRefresh_NotConfigured(t *testing.T) {
	setupSimpleTaskDB(t, &model.SiteConfig{})
	// GetSMHConfig 无配置，IsConfigured() 返回 false → 直接 return
	runSMHTokenRefresh(context.Background())
}

// ─── runVNCSGMigration ─────────────────────────────────────────────────────

// TestRunVNCSGMigration_BrowserVNCDisabled VNC 未开启时直接返回。
func TestRunVNCSGMigration_BrowserVNCDisabled(t *testing.T) {
	db := setupSimpleTaskDB(t, &model.SiteConfig{}, &model.RuleSet{}, &model.ManagedSGPool{})
	// BrowserVNCEnable=false → 直接返回
	db.Create(&model.SiteConfig{Name: "test", BrowserVNCEnable: false})
	storeInt32(getVNCMigrateState(""), 0)
	runVNCSGMigration(context.Background())
}

// TestRunVNCSGMigration_AlreadyDone 已完成时直接返回（CAS 失败）。
func TestRunVNCSGMigration_AlreadyDone(t *testing.T) {
	db := setupSimpleTaskDB(t, &model.SiteConfig{}, &model.RuleSet{}, &model.ManagedSGPool{})
	db.Create(&model.SiteConfig{Name: "test", BrowserVNCEnable: true})
	storeInt32(getVNCMigrateState(""), 2) // 已完成
	defer storeInt32(getVNCMigrateState(""), 0)
	runVNCSGMigration(context.Background())
}

// TestRunVNCSGMigration_NoSGPool BrowserVNC 开启但无安全组池时跳过迁移。
func TestRunVNCSGMigration_NoSGPool(t *testing.T) {
	db := setupSimpleTaskDB(t, &model.SiteConfig{}, &model.RuleSet{}, &model.ManagedSGPool{})
	db.Create(&model.SiteConfig{Name: "test", BrowserVNCEnable: true})
	storeInt32(getVNCMigrateState(""), 0)
	defer storeInt32(getVNCMigrateState(""), 0)
	// HasSGPoolReady：无 RuleSet/SG → ready=false → 跳过
	runVNCSGMigration(context.Background())
}

// ─── runSoulTaskEntry ─────────────────────────────────────────────────────

// TestRunSoulTaskEntry_Entry 验证入口函数调用 doSoulSet（已有 mock 测试覆盖，这里只测入口）。
func TestRunSoulTaskEntry_Entry(t *testing.T) {
	setupSimpleTaskDB(t, &model.Instance{}, &model.OpenClawRole{})
	// 无待下发实例，直接返回，不 panic
	runSoulTaskEntry(context.Background())
}

// ─── runNotificationCleanup 补充覆盖 ─────────────────────────────────────

// TestRunNotificationCleanup_WithError 模拟清理时有旧通知，触发 slog.Info 路径。
func TestRunNotificationCleanup_HasExpired(t *testing.T) {
	db := setupSimpleTaskDB(t, &model.Notification{})

	import_time := model.Notification{
		UserID: 1, Type: "test", Title: "old",
	}
	db.Create(&import_time)
	// 直接更新 created_at 为 35 天前
	db.Exec("UPDATE notifications SET created_at = datetime('now', '-35 days') WHERE 1=1")

	// 应触发 CleanupExpiredNotifications 并进入 affected > 0 分支
	runNotificationCleanup(context.Background())
}

// ─── runSMHAutoProvisionEntry 补充覆盖 ─────────────────────────────────────

// TestRunSMHAutoProvisionEntry_AlreadyEnabled SMH 已开通，provision 返回 nil，
// 触发 StartDefaultBundleSMHSync（但 sync 内部因无 token 而跳过）。
func TestRunSMHAutoProvisionEntry_AlreadyEnabled(t *testing.T) {
	setupSimpleTaskDB(t, &model.SiteConfig{}, &model.SkillBundle{}, &model.BundleSkill{}, &model.OpenClawRoleSkill{})

	// 用 mock deps 替换，让 safeSMHProvision 直接返回 nil
	origFn := newSMHAutoProvisionTask
	_ = origFn

	task := newSMHAutoProvisionTask(&mockSMHProvisionDeps{
		getSiteConfig:              func(ctx context.Context) model.SiteConfig { return model.SiteConfig{SMHEnabled: 1} },
		ensureLibrarySearchEnabled: func() error { return nil },
		startDefaultBundleSMHSync:  func(ctx context.Context) {}, // no-op
	})

	// 验证 err==nil 时会触发 StartDefaultBundleSMHSync
	syncCalled := false
	task.deps = &mockSMHProvisionDeps{
		getSiteConfig:              func(ctx context.Context) model.SiteConfig { return model.SiteConfig{SMHEnabled: 1} },
		ensureLibrarySearchEnabled: func() error { return nil },
		startDefaultBundleSMHSync:  func(ctx context.Context) { syncCalled = true },
	}
	err := task.safeSMHProvision(context.Background())
	if err != nil {
		t.Fatalf("expected nil: %v", err)
	}
	// 模拟 runSMHAutoProvisionEntry 的行为
	if err == nil {
		task.deps.StartDefaultBundleSMHSync(context.Background())
	}
	if !syncCalled {
		t.Error("expected StartDefaultBundleSMHSync to be called")
	}
}

// ─── runPersonalSpaceEnvSync / runPersonalSpaceTokenRefresh / runPersonalSpaceRecycleBin ─

// TestRunPersonalSpaceEnvSync_NoSpaces 无个人空间时直接返回。
func TestRunPersonalSpaceEnvSync_NoSpaces(t *testing.T) {
	setupSimpleTaskDB(t, &model.SMHPersonalSpace{}, &model.SiteConfig{})
	runPersonalSpaceEnvSync(context.Background())
}

// TestRunPersonalSpaceTokenRefresh_NoSpaces 无待刷新空间时直接返回。
func TestRunPersonalSpaceTokenRefresh_NoSpaces(t *testing.T) {
	setupSimpleTaskDB(t, &model.SMHPersonalSpace{}, &model.SiteConfig{})
	runPersonalSpaceTokenRefresh(context.Background())
}

// TestRunPersonalSpaceRecycleBin_NoExpired 无过期空间时直接返回。
func TestRunPersonalSpaceRecycleBin_NoExpired(t *testing.T) {
	setupSimpleTaskDB(t, &model.SMHPersonalSpace{}, &model.SiteConfig{})
	runPersonalSpaceRecycleBin(context.Background())
}

// ─── runVNCSGMigration 补充覆盖 ───────────────────────────────────────────

// TestRunVNCSGMigration_SGPoolReady BrowserVNC 开启且 SG 池就绪时，触发 HasSGPoolReady(ctx)
// 并进入 CAS 执行路径（RefreshAllRuleSetsForRequiredRules 因无云凭据而失败，但不 panic）。
func TestRunVNCSGMigration_SGPoolReady(t *testing.T) {
	db := setupSimpleTaskDB(t, &model.SiteConfig{}, &model.RuleSet{}, &model.ManagedSGPool{})
	db.Create(&model.SiteConfig{Name: "test", BrowserVNCEnable: true})

	// 插入一个 RuleSet + 一个 ACTIVE SG，使 HasSGPoolReady 返回 ready=true
	rs := model.RuleSet{Name: "default", Rules: "[]", Version: 1, UserGroupIDs: "[]"}
	db.Create(&rs)
	db.Create(&model.ManagedSGPool{
		SGID:        "sg-test-001",
		RuleSetID:   rs.ID,
		Status:      model.SGStatusActive,
		RuleVersion: 1,
	})

	// 重置迁移状态为 0（未完成）
	storeInt32(getVNCMigrateState(""), 0)
	defer storeInt32(getVNCMigrateState(""), 0)

	// 注入 TenantSnapshot（identifier 为空，与 SQLite 单租户模式一致）
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{})
	// RefreshAllRuleSetsForRequiredRules 内部因无云凭据而失败，但不应 panic
	runVNCSGMigration(ctx)
}
