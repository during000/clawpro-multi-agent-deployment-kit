package controller

import (
	"testing"
	"time"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// initTestDB 初始化内存 SQLite 数据库
func initOperationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.User{}, &model.Instance{}, &model.InstanceAdjustment{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	return db
}

// createTestUser 创建测试用户
func createTestUser(t *testing.T, db *gorm.DB) *model.User {
	t.Helper()
	user := &model.User{Username: "testuser", Password: "x", Role: "user"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	return user
}

// createTestInstance 创建测试实例
func createTestInstance(t *testing.T, db *gorm.DB, userID uint, name string) *model.Instance {
	t.Helper()
	instance := &model.Instance{
		Name:   name,
		UserID: userID,
	}
	if err := db.Create(instance).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}
	return instance
}

// ─── setOperation 乐观锁测试 ───────────────────────────────────────────────

func TestSetOperation_Success(t *testing.T) {
	db := initOperationTestDB(t)
	user := createTestUser(t, db)
	instance := createTestInstance(t, db, user.ID, "test-instance")

	err := setOperation(db, instance, model.OpReboot)
	if err != nil {
		t.Errorf("期望设置操作成功，实际错误: %v", err)
	}

	// 验证数据库中的值
	var updated model.Instance
	db.First(&updated, instance.ID)
	if updated.CurrentOperation != model.OpReboot {
		t.Errorf("期望 CurrentOperation=%s，实际=%s", model.OpReboot, updated.CurrentOperation)
	}
	if updated.CurrentOperationState != model.OpStateProcessing {
		t.Errorf("期望 CurrentOperationState=%s，实际=%s", model.OpStateProcessing, updated.CurrentOperationState)
	}
	if updated.CurrentOperationUpdatedAt == nil {
		t.Error("期望 CurrentOperationUpdatedAt 非空")
	}
}

func TestSetOperation_Conflict(t *testing.T) {
	db := initOperationTestDB(t)
	user := createTestUser(t, db)
	instance := createTestInstance(t, db, user.ID, "test-instance")

	// 第一次设置操作
	err1 := setOperation(db, instance, model.OpReboot)
	if err1 != nil {
		t.Fatalf("第一次设置操作失败: %v", err1)
	}

	// 第二次设置不同操作应返回冲突错误
	err2 := setOperation(db, instance, model.OpReinstall)
	if err2 == nil {
		t.Error("期望返回冲突错误，实际为 nil")
	}
	if err2 != ErrOperationInProgress {
		t.Errorf("期望 ErrOperationInProgress，实际: %v", err2)
	}
}

func TestSetOperation_SameOperationAllowed(t *testing.T) {
	db := initOperationTestDB(t)
	user := createTestUser(t, db)
	instance := createTestInstance(t, db, user.ID, "test-instance")

	// 第一次设置
	err1 := setOperation(db, instance, model.OpReboot)
	if err1 != nil {
		t.Fatalf("第一次设置操作失败: %v", err1)
	}

	// 再次设置相同操作应该允许（幂等）
	err2 := setOperation(db, instance, model.OpReboot)
	if err2 != nil {
		t.Errorf("期望相同操作允许，实际错误: %v", err2)
	}
}

func TestSetOperation_AfterClear(t *testing.T) {
	db := initOperationTestDB(t)
	user := createTestUser(t, db)
	instance := createTestInstance(t, db, user.ID, "test-instance")

	// 设置操作
	rerr := setOperation(db, instance, model.OpReboot)
	if rerr != nil {
		t.Fatalf("设置操作失败: %v", rerr)
	}

	// 清除操作
	err := clearOperation(db, instance, model.OpStateSuccess)
	if err != nil {
		t.Fatalf("清除操作失败: %v", err)
	}

	// 现在应该可以设置新操作
	rerr = setOperation(db, instance, model.OpReinstall)
	if rerr != nil {
		t.Errorf("清除后设置新操作应该成功，实际错误: %v", rerr)
	}
}

// ─── clearOperation 清除操作测试 ───────────────────────────────────────────

func TestClearOperation_Success(t *testing.T) {
	db := initOperationTestDB(t)
	user := createTestUser(t, db)
	instance := createTestInstance(t, db, user.ID, "test-instance")

	// 先设置操作
	setOperation(db, instance, model.OpReboot)

	// 清除操作
	err := clearOperation(db, instance, model.OpStateSuccess)
	if err != nil {
		t.Errorf("期望清除成功，实际错误: %v", err)
	}

	// 验证
	var updated model.Instance
	db.First(&updated, instance.ID)
	if updated.CurrentOperation != model.OpNone {
		t.Errorf("期望 CurrentOperation 为空，实际=%s", updated.CurrentOperation)
	}
	if updated.CurrentOperationState != model.OpStateSuccess {
		t.Errorf("期望 CurrentOperationState=%s，实际=%s", model.OpStateSuccess, updated.CurrentOperationState)
	}
}

func TestClearOperation_NoOperation(t *testing.T) {
	db := initOperationTestDB(t)
	user := createTestUser(t, db)
	instance := createTestInstance(t, db, user.ID, "test-instance")

	// 不设置操作直接清除应该成功（幂等）
	err := clearOperation(db, instance, model.OpStateSuccess)
	if err != nil {
		t.Errorf("清除空操作应成功，实际错误: %v", err)
	}
}

// ─── canOperate 可操作性测试 ───────────────────────────────────────────────

func TestCanOperate_NoOperation(t *testing.T) {
	db := initOperationTestDB(t)
	user := createTestUser(t, db)
	instance := createTestInstance(t, db, user.ID, "test-instance")

	err := canOperate(instance, model.OpReboot)
	if err != nil {
		t.Errorf("无操作时应可执行，实际错误: %v", err)
	}
}

func TestCanOperate_OperationInProgress(t *testing.T) {
	db := initOperationTestDB(t)
	user := createTestUser(t, db)
	instance := createTestInstance(t, db, user.ID, "test-instance")

	// 设置一个操作
	setOperation(db, instance, model.OpReboot)

	// 检查其他操作应该被拒绝
	err := canOperate(instance, model.OpReinstall)
	if err == nil {
		t.Error("有操作进行中时应拒绝其他操作")
	}
}

func TestCanOperate_DeleteAlwaysAllowed(t *testing.T) {
	db := initOperationTestDB(t)
	user := createTestUser(t, db)
	instance := createTestInstance(t, db, user.ID, "test-instance")

	// 设置一个操作
	setOperation(db, instance, model.OpReboot)

	// 删除操作应该始终允许（无状态参数时）
	err := canOperate(instance, model.OpDelete)
	if err != nil {
		t.Errorf("删除操作应始终允许，实际错误: %v", err)
	}
}

// ─── canOperate 基于 OpenClaw 状态的删除校验测试（Gap 1）─────────────────────

func TestCanOperate_DeleteForbiddenWhenLoading(t *testing.T) {
	db := initOperationTestDB(t)
	user := createTestUser(t, db)
	instance := createTestInstance(t, db, user.ID, "test-instance")

	// loading 状态下禁止删除
	err := canOperate(instance, model.OpDelete, model.StatusLoading)
	if err == nil {
		t.Error("loading 状态下应禁止删除")
	}
}

func TestCanOperate_DeleteForbiddenWhenDestroying(t *testing.T) {
	db := initOperationTestDB(t)
	user := createTestUser(t, db)
	instance := createTestInstance(t, db, user.ID, "test-instance")

	// destroying 状态下禁止删除
	err := canOperate(instance, model.OpDelete, model.StatusDestroying)
	if err == nil {
		t.Error("destroying 状态下应禁止删除")
	}
}

func TestCanOperate_DeleteForbiddenWhenPending(t *testing.T) {
	db := initOperationTestDB(t)
	user := createTestUser(t, db)
	instance := createTestInstance(t, db, user.ID, "test-instance")

	// pending 状态下禁止删除
	err := canOperate(instance, model.OpDelete, model.StatusPending)
	if err == nil {
		t.Error("pending 状态下应禁止删除")
	}
}

func TestCanOperate_DeleteForbiddenWhenCreating(t *testing.T) {
	db := initOperationTestDB(t)
	user := createTestUser(t, db)
	instance := createTestInstance(t, db, user.ID, "test-instance")

	// creating 状态下禁止删除
	err := canOperate(instance, model.OpDelete, model.StatusCreating)
	if err == nil {
		t.Error("creating 状态下应禁止删除")
	}
}

func TestCanOperate_DeleteAllowedWhenRunning(t *testing.T) {
	db := initOperationTestDB(t)
	user := createTestUser(t, db)
	instance := createTestInstance(t, db, user.ID, "test-instance")

	// running 状态下允许删除
	err := canOperate(instance, model.OpDelete, model.StatusRunning)
	if err != nil {
		t.Errorf("running 状态下应允许删除，实际错误: %v", err)
	}
}

func TestCanOperate_DeleteAllowedWhenLoadFailed(t *testing.T) {
	db := initOperationTestDB(t)
	user := createTestUser(t, db)
	instance := createTestInstance(t, db, user.ID, "test-instance")

	// load_failed 状态下允许删除
	err := canOperate(instance, model.OpDelete, model.StatusLoadFailed)
	if err != nil {
		t.Errorf("load_failed 状态下应允许删除，实际错误: %v", err)
	}
}

func TestCanOperate_DeleteAllowedWhenMaintaining(t *testing.T) {
	db := initOperationTestDB(t)
	user := createTestUser(t, db)
	instance := createTestInstance(t, db, user.ID, "test-instance")

	// maintaining 状态下允许删除
	err := canOperate(instance, model.OpDelete, model.StatusMaintaining)
	if err != nil {
		t.Errorf("maintaining 状态下应允许删除，实际错误: %v", err)
	}
}

func TestCanOperate_DeleteAllowedWhenCreateFailed(t *testing.T) {
	db := initOperationTestDB(t)
	user := createTestUser(t, db)
	instance := createTestInstance(t, db, user.ID, "test-instance")

	// create_failed 状态下允许删除
	err := canOperate(instance, model.OpDelete, model.StatusCreateFailed)
	if err != nil {
		t.Errorf("create_failed 状态下应允许删除，实际错误: %v", err)
	}
}

func TestCanOperate_DeleteAllowedWhenDestroyed(t *testing.T) {
	db := initOperationTestDB(t)
	user := createTestUser(t, db)
	instance := createTestInstance(t, db, user.ID, "test-instance")

	// destroyed 状态下允许删除（用于清理 DB 记录）
	err := canOperate(instance, model.OpDelete, model.StatusDestroyed)
	if err != nil {
		t.Errorf("destroyed 状态下应允许删除，实际错误: %v", err)
	}
}

func TestCanOperate_NonDeleteNotAffectedByStatus(t *testing.T) {
	db := initOperationTestDB(t)
	user := createTestUser(t, db)
	instance := createTestInstance(t, db, user.ID, "test-instance")

	// 非删除操作不受 OpenClaw 状态影响（只受 currentOperation 影响）
	err := canOperate(instance, model.OpReboot, model.StatusLoading)
	if err != nil {
		t.Errorf("无操作进行中时 reboot 应允许，实际错误: %v", err)
	}
}

// ─── isOperationTimedOut 超时检测测试 ──────────────────────────────────────

func TestIsOperationTimedOut_NoOperation(t *testing.T) {
	db := initOperationTestDB(t)
	user := createTestUser(t, db)
	instance := createTestInstance(t, db, user.ID, "test-instance")

	if isOperationTimedOut(instance) {
		t.Error("无操作时不应超时")
	}
}

func TestIsOperationTimedOut_NotTimedOut(t *testing.T) {
	db := initOperationTestDB(t)
	user := createTestUser(t, db)
	instance := createTestInstance(t, db, user.ID, "test-instance")

	// 设置操作，当前时间
	now := time.Now()
	instance.CurrentOperation = model.OpReboot
	instance.CurrentOperationUpdatedAt = &now

	if isOperationTimedOut(instance) {
		t.Error("刚刚设置的操作不应超时")
	}
}

func TestIsOperationTimedOut_Expired(t *testing.T) {
	db := initOperationTestDB(t)
	user := createTestUser(t, db)
	instance := createTestInstance(t, db, user.ID, "test-instance")

	// 设置操作为 10 分钟前（超时阈值是 5 分钟）
	oldTime := time.Now().Add(-10 * time.Minute)
	instance.CurrentOperation = model.OpReboot
	instance.CurrentOperationUpdatedAt = &oldTime

	if !isOperationTimedOut(instance) {
		t.Error("10 分钟前的 reboot 操作应该超时（阈值 5 分钟）")
	}
}

func TestIsOperationTimedOut_CreateTimeout(t *testing.T) {
	db := initOperationTestDB(t)
	user := createTestUser(t, db)
	instance := createTestInstance(t, db, user.ID, "test-instance")

	// 创建操作超时阈值是 10 分钟，设置 15 分钟前
	oldTime := time.Now().Add(-15 * time.Minute)
	instance.CurrentOperation = model.OpCreate
	instance.CurrentOperationUpdatedAt = &oldTime

	if !isOperationTimedOut(instance) {
		t.Error("15 分钟前的 create 操作应该超时（阈值 10 分钟）")
	}
}

// ─── markOperationFailed 标记失败测试 ────────────────────────────────────

func TestMarkOperationFailed(t *testing.T) {
	db := initOperationTestDB(t)
	user := createTestUser(t, db)
	instance := createTestInstance(t, db, user.ID, "test-instance")

	// 先设置操作
	setOperation(db, instance, model.OpReboot)

	// 标记失败
	err := markOperationFailed(db, instance, "操作超时")
	if err != nil {
		t.Errorf("标记失败失败: %v", err)
	}

	// 验证
	var updated model.Instance
	db.First(&updated, instance.ID)
	if updated.CurrentOperationState != model.OpStateFailed {
		t.Errorf("期望 CurrentOperationState=%s，实际=%s", model.OpStateFailed, updated.CurrentOperationState)
	}
}

// ─── setOperation 删除操作特殊处理测试 ──────────────────────────────────────

func TestSetOperation_DeleteOverridesOtherOperation(t *testing.T) {
	db := initOperationTestDB(t)
	user := createTestUser(t, db)
	instance := createTestInstance(t, db, user.ID, "test-instance")

	// 先设置 reboot 操作
	err := setOperation(db, instance, model.OpReboot)
	if err != nil {
		t.Fatalf("设置 reboot 操作失败: %v", err)
	}

	// 删除操作应该能覆盖 reboot 操作
	err = setOperation(db, instance, model.OpDelete)
	if err != nil {
		t.Errorf("删除操作应该能覆盖其他操作，实际错误: %v", err)
	}

	// 验证操作已被改为 delete
	var updated model.Instance
	db.First(&updated, instance.ID)
	if updated.CurrentOperation != model.OpDelete {
		t.Errorf("期望 CurrentOperation=%s，实际=%s", model.OpDelete, updated.CurrentOperation)
	}
}

func TestSetOperation_DeleteOperationAlwaysSucceeds(t *testing.T) {
	db := initOperationTestDB(t)
	user := createTestUser(t, db)
	instance := createTestInstance(t, db, user.ID, "test-instance")

	// 不设置任何操作，直接设置删除操作
	err := setOperation(db, instance, model.OpDelete)
	if err != nil {
		t.Errorf("删除操作应该总是成功，实际错误: %v", err)
	}
}

// ─── setOperationForRetry 重试场景乐观锁测试 ───────────────────────────────

// TestSetOperationForRetry_FromEmpty 覆盖 current_operation 为空时成功设置
func TestSetOperationForRetry_FromEmpty(t *testing.T) {
	db := initOperationTestDB(t)
	user := createTestUser(t, db)
	instance := createTestInstance(t, db, user.ID, "retry-empty")
	instance.AgentReady = 1
	db.Save(instance)

	err := setOperationForRetry(db, instance, model.OpUpgrade)
	if err != nil {
		t.Fatalf("空操作态 setOperationForRetry 应成功: %v", err)
	}
	// 验证内存对象
	if instance.CurrentOperation != model.OpUpgrade {
		t.Errorf("期望内存 CurrentOperation=%s，实际=%s", model.OpUpgrade, instance.CurrentOperation)
	}
	if instance.CurrentOperationState != model.OpStateProcessing {
		t.Errorf("期望内存 state=%s，实际=%s", model.OpStateProcessing, instance.CurrentOperationState)
	}
	if instance.AgentReady != 0 {
		t.Errorf("重试应重置 AgentReady=0，实际=%d", instance.AgentReady)
	}
	// 验证 DB
	var updated model.Instance
	db.First(&updated, instance.ID)
	if updated.CurrentOperation != model.OpUpgrade {
		t.Errorf("DB CurrentOperation=%s", updated.CurrentOperation)
	}
	if updated.CurrentOperationState != model.OpStateProcessing {
		t.Errorf("DB state=%s", updated.CurrentOperationState)
	}
	if updated.AgentReady != 0 {
		t.Errorf("DB AgentReady=%d", updated.AgentReady)
	}
}

// TestSetOperationForRetry_OverrideSameOperationFailed 覆盖 "相同操作 + failed 状态"
// 允许重试覆盖：典型的升级失败重试路径
func TestSetOperationForRetry_OverrideSameOperationFailed(t *testing.T) {
	db := initOperationTestDB(t)
	user := createTestUser(t, db)
	instance := createTestInstance(t, db, user.ID, "retry-failed")

	// 模拟一次升级失败的状态
	now := time.Now().Add(-1 * time.Hour)
	instance.CurrentOperation = model.OpUpgrade
	instance.CurrentOperationState = model.OpStateFailed
	instance.CurrentOperationUpdatedAt = &now
	db.Save(instance)

	err := setOperationForRetry(db, instance, model.OpUpgrade)
	if err != nil {
		t.Fatalf("upgrade+failed 态应允许重试覆盖: %v", err)
	}
	if instance.CurrentOperationState != model.OpStateProcessing {
		t.Errorf("期望 state=processing，实际=%s", instance.CurrentOperationState)
	}
}

// TestSetOperationForRetry_OverrideDifferentOpFailed 覆盖 "不同操作 + failed 状态"
// 只要 current_operation_state = failed 就允许覆盖
func TestSetOperationForRetry_OverrideDifferentOpFailed(t *testing.T) {
	db := initOperationTestDB(t)
	user := createTestUser(t, db)
	instance := createTestInstance(t, db, user.ID, "retry-diff-failed")

	instance.CurrentOperation = model.OpReboot
	instance.CurrentOperationState = model.OpStateFailed
	db.Save(instance)

	err := setOperationForRetry(db, instance, model.OpUpgrade)
	if err != nil {
		t.Fatalf("任意 failed 态均应允许重试覆盖: %v", err)
	}
	if instance.CurrentOperation != model.OpUpgrade {
		t.Errorf("期望 CurrentOperation=upgrade，实际=%s", instance.CurrentOperation)
	}
}

// TestSetOperationForRetry_ConflictWhenOtherOpProcessing 覆盖
// "不同操作 + processing 状态" 应冲突
func TestSetOperationForRetry_ConflictWhenOtherOpProcessing(t *testing.T) {
	db := initOperationTestDB(t)
	user := createTestUser(t, db)
	instance := createTestInstance(t, db, user.ID, "retry-conflict")

	instance.CurrentOperation = model.OpReboot
	instance.CurrentOperationState = model.OpStateProcessing
	db.Save(instance)

	err := setOperationForRetry(db, instance, model.OpUpgrade)
	if err == nil {
		t.Error("不同操作 processing 态应返回冲突错误")
	}
	if err != nil && err != ErrOperationConflict {
		t.Errorf("期望 ErrOperationConflict，实际=%v", err)
	}
}

// ─── clearOperation 补充分支：success 状态带 last_stable_state ─────────────

// TestClearOperation_SuccessWithStableState 覆盖 state=success 且 LastCVMState 非空
// → updates["last_stable_state"] 分支
func TestClearOperation_SuccessWithStableState(t *testing.T) {
	db := initOperationTestDB(t)
	user := createTestUser(t, db)
	instance := createTestInstance(t, db, user.ID, "clear-stable")

	// 预置操作态 + LastCVMState
	instance.CurrentOperation = model.OpUpgrade
	instance.CurrentOperationState = model.OpStateProcessing
	instance.LastCVMState = "RUNNING"
	db.Save(instance)

	err := clearOperation(db, instance, model.OpStateSuccess)
	if err != nil {
		t.Fatalf("clearOperation(success) 应成功: %v", err)
	}
	// 内存对象
	if instance.CurrentOperation != model.OpNone {
		t.Errorf("CurrentOperation 应被清空，实际=%s", instance.CurrentOperation)
	}
	if instance.CurrentOperationState != model.OpStateSuccess {
		t.Errorf("CurrentOperationState 应为 success，实际=%s", instance.CurrentOperationState)
	}
	if instance.LastStableState != "RUNNING" {
		t.Errorf("LastStableState 应同步为 RUNNING，实际=%s", instance.LastStableState)
	}
	// DB
	var updated model.Instance
	db.First(&updated, instance.ID)
	if updated.LastStableState != "RUNNING" {
		t.Errorf("DB LastStableState=%s", updated.LastStableState)
	}
}
