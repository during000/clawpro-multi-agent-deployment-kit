package task

import (
	"context"
	"sync"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"hatchery/model"
)

// ─── 辅助 ─────────────────────────────────────────────────────────────────

func setupSGInitTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.RuleSet{},
		&model.SiteConfig{},
		&model.User{},
		&model.Notification{},
		&model.AuditLog{},
		&model.Instance{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	// 清理全局 per-tenant 状态
	t.Cleanup(func() { sgInitStates = sync.Map{} })
	return db
}

// ─── getSGInitState ────────────────────────────────────────────────────────

func TestGetSGInitState_CreatesNew(t *testing.T) {
	sgInitStates = sync.Map{}
	defer func() { sgInitStates = sync.Map{} }()

	s := getSGInitState("tenant-x")
	if s == nil {
		t.Fatal("getSGInitState 应返回非 nil")
	}
	if loadInt32(&s.completed) != 0 {
		t.Error("初始 completed 应为 0")
	}
}

func TestGetSGInitState_ReturnsSame(t *testing.T) {
	sgInitStates = sync.Map{}
	defer func() { sgInitStates = sync.Map{} }()

	s1 := getSGInitState("t1")
	s2 := getSGInitState("t1")
	if s1 != s2 {
		t.Error("同一 identifier 应返回同一 state 指针")
	}
}

// ─── IsSGInitCompleted ─────────────────────────────────────────────────────

func TestIsSGInitCompleted_Empty(t *testing.T) {
	sgInitStates = sync.Map{}
	defer func() { sgInitStates = sync.Map{} }()

	// 无任何 state 时，Range 不执行，allDone=true
	if !IsSGInitCompleted() {
		t.Error("无 state 时 IsSGInitCompleted 应返回 true")
	}
}

func TestIsSGInitCompleted_OneIncomplete(t *testing.T) {
	sgInitStates = sync.Map{}
	defer func() { sgInitStates = sync.Map{} }()

	s := getSGInitState("t1")
	storeInt32(&s.completed, 0)

	if IsSGInitCompleted() {
		t.Error("有未完成 state 时应返回 false")
	}
}

func TestIsSGInitCompleted_AllComplete(t *testing.T) {
	sgInitStates = sync.Map{}
	defer func() { sgInitStates = sync.Map{} }()

	s := getSGInitState("t1")
	storeInt32(&s.completed, 1)

	if !IsSGInitCompleted() {
		t.Error("全部完成时应返回 true")
	}
}

// ─── GetSGInitError ────────────────────────────────────────────────────────

func TestGetSGInitError_Empty(t *testing.T) {
	sgInitStates = sync.Map{}
	defer func() { sgInitStates = sync.Map{} }()

	if got := GetSGInitError(); got != "" {
		t.Errorf("无 state 时应返回空，实际 %q", got)
	}
}

func TestGetSGInitError_WithError(t *testing.T) {
	sgInitStates = sync.Map{}
	defer func() { sgInitStates = sync.Map{} }()

	s := getSGInitState("t1")
	s.mu.Lock()
	s.lastErr = "quota exceeded"
	s.mu.Unlock()

	if got := GetSGInitError(); got != "quota exceeded" {
		t.Errorf("期望 'quota exceeded'，实际 %q", got)
	}
}

func TestGetSGInitError_NoError(t *testing.T) {
	sgInitStates = sync.Map{}
	defer func() { sgInitStates = sync.Map{} }()

	s := getSGInitState("t1")
	s.mu.Lock()
	s.lastErr = ""
	s.mu.Unlock()

	if got := GetSGInitError(); got != "" {
		t.Errorf("期望空，实际 %q", got)
	}
}

// ─── runSGRuleSetInit ──────────────────────────────────────────────────────

func TestRunSGRuleSetInit_AlreadyCompleted(t *testing.T) {
	setupSGInitTestDB(t)

	s := getSGInitState("")
	storeInt32(&s.completed, 1)

	// 已完成，直接返回，不 panic
	runSGRuleSetInit(context.Background())
}

func TestRunSGRuleSetInit_FreshTenantNoSG(t *testing.T) {
	db := setupSGInitTestDB(t)
	db.Create(&model.SiteConfig{Name: "Test", SecurityGroupId: ""})

	runSGRuleSetInit(context.Background())

	// 完成后 state.completed 应为 1
	s := getSGInitState("")
	if loadInt32(&s.completed) != 1 {
		t.Error("fresh tenant 无 SG 应成功完成（直接返回 nil）")
	}
}

func TestRunSGRuleSetInit_FailStoresError(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// 不迁移 rule_sets 表，迫使 GetDefaultRuleSet 返回非 ErrRecordNotFound 错误
	db.AutoMigrate(&model.User{}, &model.Notification{})
	t.Cleanup(model.UseDBForTest(db))
	t.Cleanup(func() { sgInitStates = sync.Map{} })

	runSGRuleSetInit(context.Background())

	s := getSGInitState("")
	s.mu.RLock()
	errStr := s.lastErr
	s.mu.RUnlock()
	if errStr == "" {
		t.Error("initSGRuleSet 失败时应记录错误")
	}
}

// ─── initSGRuleSet ─────────────────────────────────────────────────────────

func TestInitSGRuleSet_RuleSetExists(t *testing.T) {
	db := setupSGInitTestDB(t)
	db.Create(&model.RuleSet{Name: model.DefaultRuleSetName, IsDefault: true})

	err := initSGRuleSet(context.Background())
	if err != nil {
		t.Errorf("rule_set 已存在时应返回 nil，实际: %v", err)
	}
}

func TestInitSGRuleSet_FreshTenant_NoSG(t *testing.T) {
	db := setupSGInitTestDB(t)
	db.Create(&model.SiteConfig{Name: "Fresh", SecurityGroupId: ""})

	err := initSGRuleSet(context.Background())
	if err != nil {
		t.Errorf("fresh tenant 无 SG 应返回 nil，实际: %v", err)
	}
}

func TestInitSGRuleSet_WithSiteConfigNonEmptySG(t *testing.T) {
	db := setupSGInitTestDB(t)
	db.Create(&model.SiteConfig{Name: "With-SG", SecurityGroupId: "sg-test-001"})

	err := initSGRuleSet(context.Background())
	if err == nil {
		t.Error("有 SecurityGroupId 但无云凭据时应返回 error")
	}
}

// ─── notifyAdminsOfSGMigration ─────────────────────────────────────────────

func TestNotifyAdminsOfSGMigration_NoAdmins(t *testing.T) {
	setupSGInitTestDB(t)
	notifyAdminsOfSGMigration(context.Background(), 1, "sg-old", "sg-new")
}

func TestNotifyAdminsOfSGMigration_WithAdmins(t *testing.T) {
	db := setupSGInitTestDB(t)
	db.Create(&model.User{Username: "admin1", Password: "x", Role: "admin"})

	notifyAdminsOfSGMigration(context.Background(), 1, "sg-old", "sg-new")

	var count int64
	db.Model(&model.Notification{}).Count(&count)
	if count == 0 {
		t.Error("期望为 admin 创建迁移通知")
	}
}

// ─── notifyAdminsOfSGInitFailure ───────────────────────────────────────────

func TestNotifyAdminsOfSGInitFailure_NoAdmins(t *testing.T) {
	setupSGInitTestDB(t)
	notifyAdminsOfSGInitFailure(context.Background(), simpleErr{"quota exceeded"})
}

func TestNotifyAdminsOfSGInitFailure_HasUnreadNotif(t *testing.T) {
	db := setupSGInitTestDB(t)
	db.Create(&model.User{Username: "admin1", Password: "x", Role: "admin"})
	db.Create(&model.Notification{
		UserID: 1, Type: "sg_bootstrap_failed", IsRead: false, Title: "旧通知",
	})

	notifyAdminsOfSGInitFailure(context.Background(), simpleErr{"quota exceeded"})

	var count int64
	db.Model(&model.Notification{}).Where("`type` = ? AND is_read = ?", "sg_bootstrap_failed", false).Count(&count)
	if count != 1 {
		t.Errorf("已有未读通知时不应新增，期望 1，实际 %d", count)
	}
}

func TestNotifyAdminsOfSGInitFailure_NoUnread(t *testing.T) {
	db := setupSGInitTestDB(t)
	db.Create(&model.User{Username: "admin1", Password: "x", Role: "admin"})

	notifyAdminsOfSGInitFailure(context.Background(), simpleErr{"quota exceeded"})

	var count int64
	db.Model(&model.Notification{}).Where("`type` = ?", "sg_bootstrap_failed").Count(&count)
	if count == 0 {
		t.Error("无未读通知时应创建失败通知")
	}
}

// ─── notifyAdminsOfSGInitRecovery ──────────────────────────────────────────

func TestNotifyAdminsOfSGInitRecovery_NoAdmins(t *testing.T) {
	setupSGInitTestDB(t)
	notifyAdminsOfSGInitRecovery(context.Background())
}

func TestNotifyAdminsOfSGInitRecovery_WithAdmins(t *testing.T) {
	db := setupSGInitTestDB(t)
	db.Create(&model.User{Username: "admin1", Password: "x", Role: "admin"})

	notifyAdminsOfSGInitRecovery(context.Background())

	var count int64
	db.Model(&model.Notification{}).Where("`type` = ?", "sg_bootstrap_recovered").Count(&count)
	if count == 0 {
		t.Error("期望创建恢复通知")
	}
}

type simpleErr struct{ msg string }

func (e simpleErr) Error() string { return e.msg }

var _ sync.Map // 确保 sync 包被使用
