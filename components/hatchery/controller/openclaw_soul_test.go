package controller

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// ---------- DB 初始化 ----------

func initSoulControllerTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.Instance{},
		&model.OpenClawRole{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
}

// ---------- SetInstanceSoul Tests ----------

func TestSetInstanceSoul_Success(t *testing.T) {
	initSoulControllerTestDB(t)

	role := model.OpenClawRole{Name: "测试角色", Soul: "你是一个测试助手", Visible: true}
	model.DB(context.Background()).Create(&role)

	inst := model.Instance{
		Name:        "test",
		InstanceId:  "ins-test",
		RoleID:      role.ID,
		RuntimeUser: "testuser",
		AgentType:   model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(&inst)

	// Mock RunScript 避免真实 TAT 调用
	origRunner := agentScriptRunner
	t.Cleanup(func() { agentScriptRunner = origRunner })
	agentScriptRunner = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		// 验证参数正确性
		if scriptName != "set_soul.sh" {
			t.Errorf("期望脚本 set_soul.sh, 实际 %s", scriptName)
		}
		if runtimeUser != "testuser" {
			t.Errorf("期望 runtimeUser testuser, 实际 %s", runtimeUser)
		}
		soulB64 := params["soul_b64"]
		decoded, err := base64.StdEncoding.DecodeString(soulB64)
		if err != nil {
			t.Fatalf("soul_b64 解码失败: %v", err)
		}
		if string(decoded) != "你是一个测试助手" {
			t.Errorf("期望 soul 内容不正确: %s", string(decoded))
		}
		return "ok", nil
	}

	err := SetInstanceSoul(context.Background(), inst.ID, 0)
	if err != nil {
		t.Fatalf("SetInstanceSoul 失败: %v", err)
	}

	// 验证 soul_set_at 已设置
	var updated model.Instance
	model.DB(context.Background()).First(&updated, inst.ID)
	if updated.SoulSetAt == nil {
		t.Error("soul_set_at 应为非 NULL")
	}
}

func TestSetInstanceSoul_NoRole(t *testing.T) {
	initSoulControllerTestDB(t)

	inst := model.Instance{
		Name:        "test",
		InstanceId:  "ins-test",
		RoleID:      0,
		RuntimeUser: "testuser",
		AgentType:   model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(&inst)

	runCalled := false
	origRunner := agentScriptRunner
	t.Cleanup(func() { agentScriptRunner = origRunner })
	agentScriptRunner = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		runCalled = true
		return "ok", nil
	}

	err := SetInstanceSoul(context.Background(), inst.ID, 0)
	if err != nil {
		t.Fatalf("无角色时不应报错: %v", err)
	}
	if runCalled {
		t.Error("无角色时不应执行脚本")
	}
}

func TestSetInstanceSoul_NoRuntimeUser(t *testing.T) {
	initSoulControllerTestDB(t)

	role := model.OpenClawRole{Name: "测试角色", Soul: "test", Visible: true}
	model.DB(context.Background()).Create(&role)

	inst := model.Instance{
		Name:        "test",
		InstanceId:  "ins-test",
		RoleID:      role.ID,
		RuntimeUser: "", // 未检测
		AgentType:   model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(&inst)

	runCalled := false
	origRunner := agentScriptRunner
	t.Cleanup(func() { agentScriptRunner = origRunner })
	agentScriptRunner = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		runCalled = true
		return "ok", nil
	}

	err := SetInstanceSoul(context.Background(), inst.ID, 0)
	if err != nil {
		t.Fatalf("无 RuntimeUser 时不应报错: %v", err)
	}
	if runCalled {
		t.Error("无 RuntimeUser 时不应执行脚本，应等下次轮询")
	}
}

func TestSetInstanceSoul_EmptySoul(t *testing.T) {
	initSoulControllerTestDB(t)

	role := model.OpenClawRole{Name: "空灵魂角色", Soul: "", Visible: true}
	model.DB(context.Background()).Create(&role)

	inst := model.Instance{
		Name:        "test",
		InstanceId:  "ins-test",
		RoleID:      role.ID,
		RuntimeUser: "testuser",
		AgentType:   model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(&inst)

	runCalled := false
	origRunner := agentScriptRunner
	t.Cleanup(func() { agentScriptRunner = origRunner })
	agentScriptRunner = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		runCalled = true
		return "ok", nil
	}

	err := SetInstanceSoul(context.Background(), inst.ID, 0)
	if err != nil {
		t.Fatalf("空 Soul 时不应报错: %v", err)
	}
	if runCalled {
		t.Error("空 Soul 时不应执行脚本")
	}
}

func TestSetInstanceSoul_TATFailure(t *testing.T) {
	initSoulControllerTestDB(t)

	role := model.OpenClawRole{Name: "测试角色", Soul: "test", Visible: true}
	model.DB(context.Background()).Create(&role)

	inst := model.Instance{
		Name:        "test",
		InstanceId:  "ins-test",
		RoleID:      role.ID,
		RuntimeUser: "testuser",
		AgentType:   model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(&inst)

	origRunner := agentScriptRunner
	t.Cleanup(func() { agentScriptRunner = origRunner })
	agentScriptRunner = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		return "", hcommon.I18nError(i18n.MsgTATCommandDispatchFailed)
	}

	err := SetInstanceSoul(context.Background(), inst.ID, 0)
	if err == nil {
		t.Fatal("TAT 失败时应返回 error")
	}

	// soul_set_at 不应被设置
	var updated model.Instance
	model.DB(context.Background()).First(&updated, inst.ID)
	if updated.SoulSetAt != nil {
		t.Error("TAT 失败时 soul_set_at 应为 NULL")
	}
}

func TestSetInstanceSoul_NonExistentRole(t *testing.T) {
	initSoulControllerTestDB(t)

	inst := model.Instance{
		Name:        "test",
		InstanceId:  "ins-test",
		RoleID:      999, // 不存在的角色
		RuntimeUser: "testuser",
		AgentType:   model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(&inst)

	err := SetInstanceSoul(context.Background(), inst.ID, 0)
	if err == nil {
		t.Fatal("角色不存在时应返回 error")
	}
}

func TestSetInstanceSoul_AgentTypeNotSupportsRole(t *testing.T) {
	initSoulControllerTestDB(t)

	role := model.OpenClawRole{Name: "测试角色", Soul: "test", Visible: true}
	model.DB(context.Background()).Create(&role)

	inst := model.Instance{
		Name:        "test",
		InstanceId:  "ins-test",
		RoleID:      role.ID,
		RuntimeUser: "testuser",
		AgentType:   "unknown_type",
	}
	model.DB(context.Background()).Create(&inst)

	runCalled := false
	origRunner := agentScriptRunner
	t.Cleanup(func() { agentScriptRunner = origRunner })
	agentScriptRunner = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		runCalled = true
		return "ok", nil
	}

	err := SetInstanceSoul(context.Background(), inst.ID, 0)
	if err != nil {
		t.Fatalf("不支持的 agent_type 不应报错: %v", err)
	}
	if runCalled {
		t.Error("不支持的 agent_type 不应执行脚本")
	}
}

// ---------- RemoveInstanceSoul Tests ----------

func TestRemoveInstanceSoul_Success(t *testing.T) {
	initSoulControllerTestDB(t)

	now := time.Now()
	inst := model.Instance{
		Name:        "test",
		InstanceId:  "ins-test",
		RoleID:      1,
		RuntimeUser: "testuser",
		AgentType:   model.AgentTypeOpenClaw,
		SoulSetAt:   &now,
	}
	model.DB(context.Background()).Create(&inst)

	origRunner := agentScriptRunner
	t.Cleanup(func() { agentScriptRunner = origRunner })
	agentScriptRunner = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		if scriptName != "remove_soul.sh" {
			t.Errorf("期望脚本 remove_soul.sh, 实际 %s", scriptName)
		}
		return "ok", nil
	}

	err := RemoveInstanceSoul(context.Background(), inst.ID, 0)
	if err != nil {
		t.Fatalf("RemoveInstanceSoul 不应失败: %v", err)
	}

	// 验证 soul_set_at 已清除
	var updated model.Instance
	model.DB(context.Background()).First(&updated, inst.ID)
	if updated.SoulSetAt != nil {
		t.Error("移除后 soul_set_at 应为 NULL")
	}
}

func TestRemoveInstanceSoul_NoRuntimeUser(t *testing.T) {
	initSoulControllerTestDB(t)

	inst := model.Instance{
		Name:        "test",
		InstanceId:  "ins-test",
		RuntimeUser: "",
		AgentType:   model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(&inst)

	runCalled := false
	origRunner := agentScriptRunner
	t.Cleanup(func() { agentScriptRunner = origRunner })
	agentScriptRunner = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		runCalled = true
		return "ok", nil
	}

	err := RemoveInstanceSoul(context.Background(), inst.ID, 0)
	if err != nil {
		t.Fatalf("无 RuntimeUser 时应静默跳过，不应报错: %v", err)
	}
	if runCalled {
		t.Error("无 RuntimeUser 时不应执行脚本")
	}
}

func TestRemoveInstanceSoul_TATFailure(t *testing.T) {
	initSoulControllerTestDB(t)

	inst := model.Instance{
		Name:        "test",
		InstanceId:  "ins-test",
		RoleID:      1,
		RuntimeUser: "testuser",
		AgentType:   model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(&inst)

	origRunner := agentScriptRunner
	t.Cleanup(func() { agentScriptRunner = origRunner })
	agentScriptRunner = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		return "", hcommon.I18nError(i18n.MsgTATCommandDispatchFailed)
	}

	err := RemoveInstanceSoul(context.Background(), inst.ID, 0)
	if err == nil {
		t.Fatal("TAT 失败时应返回 error")
	}
}

// ---------- setInstanceSoulWhenReady Tests ----------

func TestSetInstanceSoulWhenReady_WaitsForRuntimeUser(t *testing.T) {
	initSoulControllerTestDB(t)

	role := model.OpenClawRole{Name: "测试角色", Soul: "test", Visible: true}
	model.DB(context.Background()).Create(&role)

	// 创建时 RuntimeUser 为空
	inst := model.Instance{
		Name:        "test",
		InstanceId:  "ins-test",
		RoleID:      role.ID,
		RuntimeUser: "",
		AgentType:   model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(&inst)

	// 在另一个 goroutine 中 1 秒后设置 RuntimeUser
	go func() {
		time.Sleep(100 * time.Millisecond)
		model.DB(context.Background()).Model(&inst).Update("runtime_user", "testuser")
	}()

	origRunner := agentScriptRunner
	t.Cleanup(func() { agentScriptRunner = origRunner })
	runCalled := false
	agentScriptRunner = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		runCalled = true
		return "ok", nil
	}

	setInstanceSoulWhenReady(context.Background(), inst.ID, "ins-test")

	if !runCalled {
		t.Error("RuntimeUser 就绪后应执行下发")
	}

	var updated model.Instance
	model.DB(context.Background()).First(&updated, inst.ID)
	if updated.SoulSetAt == nil {
		t.Error("下发成功后 soul_set_at 应为非 NULL")
	}
}

func TestSetInstanceSoulWhenReady_RoleRemovedWhileWaiting(t *testing.T) {
	initSoulControllerTestDB(t)

	role := model.OpenClawRole{Name: "测试角色", Soul: "test", Visible: true}
	model.DB(context.Background()).Create(&role)

	inst := model.Instance{
		Name:        "test",
		InstanceId:  "ins-test",
		RoleID:      role.ID,
		RuntimeUser: "",
		AgentType:   model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(&inst)

	// 在等待期间移除角色
	go func() {
		time.Sleep(100 * time.Millisecond)
		model.DB(context.Background()).Model(&inst).Updates(map[string]interface{}{
			"role_id":      0,
			"runtime_user": "testuser",
		})
	}()

	runCalled := false
	origRunner := agentScriptRunner
	t.Cleanup(func() { agentScriptRunner = origRunner })
	agentScriptRunner = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		runCalled = true
		return "ok", nil
	}

	setInstanceSoulWhenReady(context.Background(), inst.ID, "ins-test")

	if runCalled {
		t.Error("角色已移除后不应执行下发")
	}
}

func TestRemoveInstanceSoul_AgentTypeNotSupportsRole(t *testing.T) {
	initSoulControllerTestDB(t)

	inst := model.Instance{
		Name:        "test",
		InstanceId:  "ins-test",
		RuntimeUser: "testuser",
		AgentType:   "unknown_type",
	}
	model.DB(context.Background()).Create(&inst)

	runCalled := false
	origRunner := agentScriptRunner
	t.Cleanup(func() { agentScriptRunner = origRunner })
	agentScriptRunner = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		runCalled = true
		return "ok", nil
	}

	err := RemoveInstanceSoul(context.Background(), inst.ID, 0)
	if err != nil {
		t.Fatalf("不支持的 agent_type 不应报错: %v", err)
	}
	if runCalled {
		t.Error("不支持的 agent_type 不应执行脚本")
	}
}
