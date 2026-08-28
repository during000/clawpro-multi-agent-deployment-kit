package controller

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupEnsureRuntimeUserTestDB 初始化内存 SQLite 测试 DB。
func setupEnsureRuntimeUserTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.Instance{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	return db
}

// withFastEnsureRuntimeUserRetry 把重试间隔缩短到 1ms，避免单测阻塞。
func withFastEnsureRuntimeUserRetry(t *testing.T) {
	t.Helper()
	origInterval := ensureRuntimeUserRetryInterval
	ensureRuntimeUserRetryInterval = time.Millisecond
	t.Cleanup(func() { ensureRuntimeUserRetryInterval = origInterval })
}

// withMockLoadScriptError 让 LoadScript 始终返回错误，从而让 RunScript 失败。
func withMockLoadScriptError(t *testing.T) {
	t.Helper()
	orig := LoadScript
	LoadScript = func(name string) (string, error) {
		return "", errors.New("mock: script load failure")
	}
	t.Cleanup(func() { LoadScript = orig })
}

// TestEnsureRuntimeUser_DBHasValue 验证 DB 已有 runtime_user 时直接返回，不再触发探测。
func TestEnsureRuntimeUser_DBHasValue(t *testing.T) {
	db := setupEnsureRuntimeUserTestDB(t)
	inst := &model.Instance{
		Name: "inst-has-value", InstanceId: "ins-eru-001",
		AgentType: model.AgentTypeHermes, RuntimeUser: "ubuntu",
	}
	db.Create(inst)

	got := ensureRuntimeUser(context.Background(), inst.ID, inst.InstanceId, model.AgentTypeHermes)
	if got != "ubuntu" {
		t.Errorf("DB 已有值时应直接返回，期望 ubuntu，实际=%q", got)
	}
}

// TestEnsureRuntimeUser_InstanceNotFound 验证查询失败时 fallback root。
func TestEnsureRuntimeUser_InstanceNotFound(t *testing.T) {
	setupEnsureRuntimeUserTestDB(t)

	// 不创建任何实例，instance_pk=99999 查询会返回 record not found
	got := ensureRuntimeUser(context.Background(), 99999, "ins-not-exist", model.AgentTypeHermes)
	if got != "root" {
		t.Errorf("实例不存在时应 fallback root，实际=%q", got)
	}
}

// TestEnsureRuntimeUser_ScriptResolveFailed 验证 ResolveScript 失败时 fallback root。
func TestEnsureRuntimeUser_ScriptResolveFailed(t *testing.T) {
	db := setupEnsureRuntimeUserTestDB(t)
	inst := &model.Instance{
		Name: "inst-unknown-type", InstanceId: "ins-eru-002",
		AgentType: "future_unknown_type", RuntimeUser: "",
	}
	db.Create(inst)

	got := ensureRuntimeUser(context.Background(), inst.ID, inst.InstanceId, "future_unknown_type")
	if got != "root" {
		t.Errorf("脚本解析失败时应 fallback root，实际=%q", got)
	}
	// 验证 DB 未被写入
	var reload model.Instance
	db.First(&reload, inst.ID)
	if reload.RuntimeUser != "" {
		t.Errorf("脚本解析失败时不应写入 DB，实际 runtime_user=%q", reload.RuntimeUser)
	}
}

// TestEnsureRuntimeUser_AllRetriesFail 验证所有重试失败时不写入 DB，返回临时 root。
func TestEnsureRuntimeUser_AllRetriesFail(t *testing.T) {
	withFastEnsureRuntimeUserRetry(t)
	withMockLoadScriptError(t)
	db := setupEnsureRuntimeUserTestDB(t)
	inst := &model.Instance{
		Name: "inst-all-fail", InstanceId: "ins-eru-003",
		AgentType: model.AgentTypeHermes, RuntimeUser: "",
	}
	db.Create(inst)

	got := ensureRuntimeUser(context.Background(), inst.ID, inst.InstanceId, model.AgentTypeHermes)
	if got != "root" {
		t.Errorf("所有重试失败时应返回临时 root，实际=%q", got)
	}
	// 验证 DB 没有被写入兜底值
	var reload model.Instance
	db.First(&reload, inst.ID)
	if reload.RuntimeUser != "" {
		t.Errorf("重试失败时不应写入 DB，实际 runtime_user=%q", reload.RuntimeUser)
	}
}

// TestEnsureRuntimeUser_ConcurrentDBWrite 验证：探测在重试间隙时如果 DB 已被并发写入，
// 函数应直接返回该值（模拟 detectAndSaveRuntimeUser 在并发中先完成的场景）。
func TestEnsureRuntimeUser_ConcurrentDBWrite(t *testing.T) {
	withFastEnsureRuntimeUserRetry(t)
	db := setupEnsureRuntimeUserTestDB(t)
	inst := &model.Instance{
		Name: "inst-concurrent", InstanceId: "ins-eru-004",
		AgentType: model.AgentTypeHermes, RuntimeUser: "",
	}
	db.Create(inst)

	// LoadScript 永远失败 → RunScript 永远失败 → 进入重试循环 → 进入"重试前再查 DB"分支
	// 用 sync.Once 保证只在第一次失败后写入 DB
	var once sync.Once
	orig := LoadScript
	LoadScript = func(name string) (string, error) {
		once.Do(func() {
			// 第一次 RunScript 失败后，立即由"另一个 goroutine"写入 DB
			db.Model(&model.Instance{}).Where("id = ?", inst.ID).
				Updates(map[string]interface{}{"runtime_user": "ubuntu", "runtime_home": "/home/ubuntu"})
		})
		return "", errors.New("mock: script load failure")
	}
	t.Cleanup(func() { LoadScript = orig })

	got := ensureRuntimeUser(context.Background(), inst.ID, inst.InstanceId, model.AgentTypeHermes)
	if got != "ubuntu" {
		t.Errorf("重试间隙 DB 被并发写入时应返回该值，期望 ubuntu，实际=%q", got)
	}
}

// TestEnsureRuntimeUser_RetryParamsZero 验证 max_attempts=0 时直接进入失败兜底（边界场景）。
func TestEnsureRuntimeUser_RetryParamsZero(t *testing.T) {
	origAttempts := ensureRuntimeUserMaxAttempts
	ensureRuntimeUserMaxAttempts = 0
	t.Cleanup(func() { ensureRuntimeUserMaxAttempts = origAttempts })

	db := setupEnsureRuntimeUserTestDB(t)
	inst := &model.Instance{
		Name: "inst-zero-attempt", InstanceId: "ins-eru-005",
		AgentType: model.AgentTypeHermes, RuntimeUser: "",
	}
	db.Create(inst)

	// max_attempts=0 时循环不执行，lastErr 为 nil → 走"成功"分支但 output 为空 → 解析失败 → root
	got := ensureRuntimeUser(context.Background(), inst.ID, inst.InstanceId, model.AgentTypeHermes)
	if got != "root" {
		t.Errorf("max_attempts=0 时应返回 root（解析失败兜底），实际=%q", got)
	}
}
