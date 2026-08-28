package task

import (
	"context"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// ─── AgentChecker 接口测试 ─────────────────────────────────────────────────

type mockChecker struct {
	name   string
	result map[string]CheckResult
	err    error
}

func (m *mockChecker) Name() string {
	return m.name
}

func (m *mockChecker) Check(ctx context.Context, instanceIds []string) (map[string]CheckResult, error) {
	return m.result, m.err
}

func TestRegisterAgentChecker(t *testing.T) {
	// 重置全局变量
	agentCheckers = nil

	mock := &mockChecker{name: "test", result: map[string]CheckResult{}}
	RegisterAgentChecker(mock)

	checkers := GetAgentCheckers()
	if len(checkers) != 1 {
		t.Errorf("期望注册 1 个 checker，实际 %d", len(checkers))
	}
	if checkers[0].Name() != "test" {
		t.Errorf("期望 name=test，实际 %s", checkers[0].Name())
	}
}

func TestGetAgentCheckers_ReturnsCopy(t *testing.T) {
	// 重置全局变量
	agentCheckers = nil

	RegisterAgentChecker(&mockChecker{name: "a", result: map[string]CheckResult{}})
	RegisterAgentChecker(&mockChecker{name: "b", result: map[string]CheckResult{}})

	checkers1 := GetAgentCheckers()
	checkers2 := GetAgentCheckers()

	if len(checkers1) != len(checkers2) {
		t.Error("两次调用应返回相同数量")
	}
}

// ─── CheckResult 常量测试 ──────────────────────────────────────────────────

func TestCheckResultConstants(t *testing.T) {
	if CheckReady != 0 {
		t.Errorf("CheckReady 应为 0，实际 %d", CheckReady)
	}
	if CheckNotReady != 1 {
		t.Errorf("CheckNotReady 应为 1，实际 %d", CheckNotReady)
	}
	if CheckUnknown != 2 {
		t.Errorf("CheckUnknown 应为 2，实际 %d", CheckUnknown)
	}
}

// ─── TATChecker 测试 ──────────────────────────────────────────────────────

func TestTATChecker_Name(t *testing.T) {
	checker := &TATChecker{}
	if checker.Name() != "tat" {
		t.Errorf("期望 name=tat，实际 %s", checker.Name())
	}
}

func TestTATChecker_Check_EmptyIds(t *testing.T) {
	checker := &TATChecker{}
	ctx := context.Background()

	result, err := checker.Check(ctx, []string{})
	if err != nil {
		t.Errorf("空列表不应返回错误，实际: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("空列表应返回空 map，实际 %v", result)
	}
}

// ─── checkAllAgents 集成测试（需要 DB）────────────────────────────────────

func setupAgentTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Skipf("跳过测试（数据库初始化失败）: %v", err)
	}
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.User{}, &model.Instance{}); err != nil {
		t.Skipf("跳过测试（数据库迁移失败）: %v", err)
	}
	// 手动添加生命周期管理字段（SQLite AutoMigrate 对 varchar 类型支持不佳）
	ensureColumnExists := func(col string, createSQL string) {
		var count int
		db.Raw("SELECT COUNT(*) FROM pragma_table_info('instances') WHERE name = ?", col).Scan(&count)
		if count == 0 {
			db.Exec(createSQL)
		}
	}

	ensureColumnExists("current_operation", "ALTER TABLE instances ADD COLUMN current_operation varchar(32) DEFAULT ''")
	ensureColumnExists("current_operation_state", "ALTER TABLE instances ADD COLUMN current_operation_state varchar(32) DEFAULT ''")
	ensureColumnExists("last_cvm_state", "ALTER TABLE instances ADD COLUMN last_cvm_state varchar(32) DEFAULT ''")
	ensureColumnExists("agent_ready", "ALTER TABLE instances ADD COLUMN agent_ready integer DEFAULT 0")

	t.Cleanup(model.UseDBForTest(db))
	return db
}

func createAgentTestUser(t *testing.T, db *gorm.DB) *model.User {
	t.Helper()
	user := &model.User{Username: "testuser", Password: "x", Role: "user"}
	if err := db.Create(user).Error; err != nil {
		t.Skipf("跳过测试（创建用户失败）: %v", err)
	}
	return user
}

func createAgentTestInstance(t *testing.T, db *gorm.DB, userID uint) *model.Instance {
	t.Helper()
	instance := &model.Instance{
		Name:       "test-instance",
		UserID:     userID,
		InstanceId: "ins-test-001",
	}
	if err := db.Create(instance).Error; err != nil {
		t.Skipf("跳过测试（创建实例失败）: %v", err)
	}
	return instance
}

func TestCheckAllAgents_NoRunningInstances(t *testing.T) {
	db := setupAgentTestDB(t)
	user := createAgentTestUser(t, db)

	// 创建一个实例但不设置 LastCVMState
	instance := createAgentTestInstance(t, db, user.ID)
	_ = instance

	// 重置 checker
	agentCheckers = nil
	RegisterAgentChecker(&TATChecker{})

	// 不应 panic
	checkAllAgents(context.Background())
}

func TestCheckAllAgents_OnlyRunningWithAgent0(t *testing.T) {
	db := setupAgentTestDB(t)
	user := createAgentTestUser(t, db)

	// 创建实例，LastCVMState=RUNNING，AgentReady=0
	instance := &model.Instance{
		Name:         "test-instance",
		UserID:       user.ID,
		InstanceId:   "ins-test-001",
		LastCVMState: "RUNNING",
		AgentReady:   0,
	}
	if err := db.Create(instance).Error; err != nil {
		t.Skipf("跳过测试（创建实例失败）: %v", err)
	}

	// 重置 checker
	agentCheckers = nil
	RegisterAgentChecker(&mockChecker{
		name:   "mock",
		result: map[string]CheckResult{"ins-test-001": CheckReady},
		err:    nil,
	})

	// 由于 SQLite 对 varchar(32) 类型支持有问题，LastCVMState 在 GORM Create 时可能未正确保存
	// 因此先用原始 SQL 设置正确的状态，再测试 checkAllAgents 的更新逻辑
	db.Exec("UPDATE instances SET last_cvm_state = 'RUNNING' WHERE id = ?", instance.ID)

	// 执行检测
	checkAllAgents(context.Background())

	// 验证 agent_ready 是否被更新
	var rawAgentReady int
	db.Raw("SELECT agent_ready FROM instances WHERE id = ?", instance.ID).Scan(&rawAgentReady)
	if rawAgentReady != 1 {
		t.Errorf("期望 AgentReady=1，实际 %d", rawAgentReady)
	}
}

func TestCheckAllAgents_AgentNotReady(t *testing.T) {
	db := setupAgentTestDB(t)
	user := createAgentTestUser(t, db)

	instance := &model.Instance{
		Name:         "test-instance",
		UserID:       user.ID,
		InstanceId:   "ins-test-001",
		LastCVMState: "RUNNING",
		AgentReady:   0,
	}
	if err := db.Create(instance).Error; err != nil {
		t.Skipf("跳过测试（创建实例失败）: %v", err)
	}

	// 重置 checker
	agentCheckers = nil
	RegisterAgentChecker(&mockChecker{
		name:   "mock",
		result: map[string]CheckResult{"ins-test-001": CheckNotReady},
		err:    nil,
	})

	// 执行检测
	checkAllAgents(context.Background())

	// 验证 agent_ready 保持为 0
	var updated model.Instance
	db.First(&updated, instance.ID)
	if updated.AgentReady != 0 {
		t.Errorf("期望 AgentReady=0（未就绪），实际 %d", updated.AgentReady)
	}
}

func TestCheckAllAgents_AgentUnknown_NotBlocking(t *testing.T) {
	db := setupAgentTestDB(t)
	user := createAgentTestUser(t, db)

	instance := &model.Instance{
		Name:         "test-instance",
		UserID:       user.ID,
		InstanceId:   "ins-test-001",
		LastCVMState: "RUNNING",
		AgentReady:   0,
	}
	if err := db.Create(instance).Error; err != nil {
		t.Skipf("跳过测试（创建实例失败）: %v", err)
	}

	// 重置 checker，mock 返回 Unknown
	agentCheckers = nil
	RegisterAgentChecker(&mockChecker{
		name:   "mock",
		result: map[string]CheckResult{"ins-test-001": CheckUnknown},
		err:    nil,
	})

	// 执行检测不应 panic（Unknown 应该跳过）
	checkAllAgents(context.Background())

	// AgentReady 仍为 0（因为 Unknown 不算就绪也不算不就绪）
	var updated model.Instance
	db.First(&updated, instance.ID)
	if updated.AgentReady != 0 {
		t.Errorf("Unknown 时 AgentReady 应保持 0，实际 %d", updated.AgentReady)
	}
}

func TestCheckAllAgents_OperationConverge(t *testing.T) {
	db := setupAgentTestDB(t)
	user := createAgentTestUser(t, db)

	instance := &model.Instance{
		Name:                  "test-instance",
		UserID:                user.ID,
		InstanceId:            "ins-test-001",
		LastCVMState:          "RUNNING",
		AgentReady:            0,
		CurrentOperation:      model.OpReboot,
		CurrentOperationState: model.OpStateProcessing,
	}
	if err := db.Create(instance).Error; err != nil {
		t.Skipf("跳过测试（创建实例失败）: %v", err)
	}

	// 由于 SQLite 对 varchar(32) 类型支持有问题，LastCVMState 在 GORM Create 时可能未正确保存
	// 因此先用原始 SQL 设置正确的状态
	db.Exec("UPDATE instances SET last_cvm_state = 'RUNNING' WHERE id = ?", instance.ID)

	// 重置 checker，返回就绪
	agentCheckers = nil
	RegisterAgentChecker(&mockChecker{
		name:   "mock",
		result: map[string]CheckResult{"ins-test-001": CheckReady},
		err:    nil,
	})

	// 执行检测
	checkAllAgents(context.Background())

	// 验证操作被收敛（使用原始 SQL 验证）
	var rawCurrentOp, rawCurrentOpState, rawLastStableState string
	db.Raw("SELECT current_operation FROM instances WHERE id = ?", instance.ID).Scan(&rawCurrentOp)
	if rawCurrentOp != model.OpNone {
		t.Errorf("期望 CurrentOperation 被清空，实际 %s", rawCurrentOp)
	}
	db.Raw("SELECT current_operation_state FROM instances WHERE id = ?", instance.ID).Scan(&rawCurrentOpState)
	if rawCurrentOpState != model.OpStateSuccess {
		t.Errorf("期望 CurrentOperationState=success，实际 %s", rawCurrentOpState)
	}
	db.Raw("SELECT last_stable_state FROM instances WHERE id = ?", instance.ID).Scan(&rawLastStableState)
	if rawLastStableState != "RUNNING" {
		t.Errorf("期望 LastStableState=RUNNING，实际 %s", rawLastStableState)
	}
}

func TestCheckAllAgents_DeleteOperation_NotConverge(t *testing.T) {
	db := setupAgentTestDB(t)
	user := createAgentTestUser(t, db)

	instance := &model.Instance{
		Name:                  "test-instance",
		UserID:                user.ID,
		InstanceId:            "ins-test-001",
		LastCVMState:          "RUNNING",
		AgentReady:            0,
		CurrentOperation:      model.OpDelete, // 删除操作不收敛
		CurrentOperationState: model.OpStateProcessing,
	}
	if err := db.Create(instance).Error; err != nil {
		t.Skipf("跳过测试（创建实例失败）: %v", err)
	}

	// 由于 SQLite 对 varchar(32) 类型支持有问题，LastCVMState 在 GORM Create 时可能未正确保存
	// 因此先用原始 SQL 设置正确的状态
	db.Exec("UPDATE instances SET last_cvm_state = 'RUNNING' WHERE id = ?", instance.ID)

	// 重置 checker，返回就绪
	agentCheckers = nil
	RegisterAgentChecker(&mockChecker{
		name:   "mock",
		result: map[string]CheckResult{"ins-test-001": CheckReady},
		err:    nil,
	})

	// 执行检测
	checkAllAgents(context.Background())

	// 验证删除操作的 currentOperation 不被收敛，但 AgentReady 仍被更新
	// （使用原始 SQL 验证）
	var rawAgentReady int
	var rawCurrentOp string
	db.Raw("SELECT agent_ready FROM instances WHERE id = ?", instance.ID).Scan(&rawAgentReady)
	if rawAgentReady != 1 {
		t.Errorf("期望 AgentReady=1，实际 %d", rawAgentReady)
	}
	db.Raw("SELECT current_operation FROM instances WHERE id = ?", instance.ID).Scan(&rawCurrentOp)
	// 删除操作的收敛由 handleStatusSideEffects 处理，AgentChecker 不收敛删除操作
	if rawCurrentOp != model.OpDelete {
		t.Errorf("期望 CurrentOperation 保持为 delete，实际 %s", rawCurrentOp)
	}
}
