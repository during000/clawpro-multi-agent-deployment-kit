package task

import (
	"context"
	"testing"
	"time"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupRuntimeUserTestDB 创建内存 SQLite，迁移必要的表。
func setupRuntimeUserTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.User{}, &model.Instance{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	return db
}

// TestDetectAndSaveRuntimeUser_UnknownAgentTypeSkips 验证当 agent_type 不在 scriptResolveTable
// 的 detect_install 分派中时，函数应静默返回，不修改数据库、不 panic。
func TestDetectAndSaveRuntimeUser_UnknownAgentTypeSkips(t *testing.T) {
	db := setupRuntimeUserTestDB(t)
	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	db.Create(user)
	inst := &model.Instance{
		Name:       "unknown-agent-inst",
		UserID:     user.ID,
		InstanceId: "ins-unknown-detect",
		AgentType:  "future_agent_type_xxx",
	}
	db.Create(inst)

	// 直接调用（不走 goroutine，便于同步验证）
	detectAndSaveRuntimeUser(context.Background(), inst.ID, inst.InstanceId, "future_agent_type_xxx")

	// 验证：runtime_user / runtime_home 均未被修改（应保持空）
	var got model.Instance
	db.First(&got, inst.ID)
	if got.RuntimeUser != "" {
		t.Errorf("未知 agent_type 不应修改 runtime_user，实际=%q", got.RuntimeUser)
	}
	if got.RuntimeHome != "" {
		t.Errorf("未知 agent_type 不应修改 runtime_home，实际=%q", got.RuntimeHome)
	}
}

// TestDetectAndSaveRuntimeUser_PanicRecovered 验证 goroutine panic 被 recover 兜住，
// 不影响主流程，不引起测试进程崩溃。
//
// 背景：detectAndSaveRuntimeUser 内部 defer recover 是最后一道防线，
// 保证异步 goroutine 的 panic 不会传播导致进程退出。
func TestDetectAndSaveRuntimeUser_PanicRecovered(t *testing.T) {
	// 此测试无法直接构造 panic 触发点（RunScript 无 mock 入口），
	// 但我们可以验证至少正常路径（resolveErr 分支）不会 panic。
	// 之前的 TestDetectAndSaveRuntimeUser_UnknownAgentTypeSkips 已经覆盖。
	//
	// 此处作为防御断言，保证函数在空参数下也不 panic。
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("detectAndSaveRuntimeUser 不应 panic，实际 recover=%v", r)
		}
	}()

	_ = setupRuntimeUserTestDB(t) // 确保 model.DB(context.Background()) 非 nil

	// 未知 agent_type → 走 resolveErr 分支，然后优雅返回
	detectAndSaveRuntimeUser(context.Background(), 0, "", "some_unknown_type")
}

// TestCheckAllAgents_SkipsRecentOperation 验证当 current_operation_updated_at 在 30s 内时，
// 实例不应被 checkAllAgents 处理（避免 CVM API 还未生效时误判 Agent 就绪）。
func TestCheckAllAgents_SkipsRecentOperation(t *testing.T) {
	db := setupAgentTestDB(t)
	user := createAgentTestUser(t, db)

	// 刚刚设置 current_operation_updated_at（5s 前），应被排除
	recent := time.Now().Add(-5 * time.Second)
	inst := &model.Instance{
		Name:                      "recent-op-inst",
		UserID:                    user.ID,
		InstanceId:                "ins-recent-op",
		LastCVMState:              "RUNNING",
		AgentReady:                0,
		CurrentOperation:          model.OpReboot,
		CurrentOperationState:     model.OpStateProcessing,
		CurrentOperationUpdatedAt: &recent,
	}
	if err := db.Create(inst).Error; err != nil {
		t.Skipf("创建实例失败: %v", err)
	}
	// SQLite AutoMigrate 对 varchar 不完全支持，补齐状态
	db.Exec("UPDATE instances SET last_cvm_state = 'RUNNING', current_operation = ?, current_operation_updated_at = ? WHERE id = ?",
		model.OpReboot, &recent, inst.ID)

	// mock checker 返回就绪
	agentCheckers = nil
	RegisterAgentChecker(&mockChecker{
		name:   "mock",
		result: map[string]CheckResult{"ins-recent-op": CheckReady},
	})

	checkAllAgents(context.Background())

	// 由于实例在 30s 内有过操作，本轮检测应将它过滤掉 → agent_ready 保持 0
	var got int
	db.Raw("SELECT agent_ready FROM instances WHERE id = ?", inst.ID).Scan(&got)
	if got != 0 {
		t.Errorf("30s 内有过操作的实例不应被更新 agent_ready，实际=%d", got)
	}
}

// TestCheckAllAgents_CheckerFailureNonFatal 验证单个 checker 返回错误时整体检测不崩溃：
//   - checker 整体报错 → 跳过此 checker，其他 checker 继续
//   - 若所有 checker 均不 Ready → agent_ready 保持 0
func TestCheckAllAgents_CheckerFailureNonFatal(t *testing.T) {
	db := setupAgentTestDB(t)
	user := createAgentTestUser(t, db)

	inst := &model.Instance{
		Name:         "checker-err-inst",
		UserID:       user.ID,
		InstanceId:   "ins-cerr",
		LastCVMState: "RUNNING",
		AgentReady:   0,
	}
	if err := db.Create(inst).Error; err != nil {
		t.Skipf("创建实例失败: %v", err)
	}
	db.Exec("UPDATE instances SET last_cvm_state = 'RUNNING' WHERE id = ?", inst.ID)

	// 一个 checker 返回 error，整体不应 panic
	agentCheckers = nil
	RegisterAgentChecker(&mockChecker{
		name: "err-checker",
		err:  context.DeadlineExceeded,
	})

	// 不应 panic
	checkAllAgents(context.Background())

	// agent_ready 保持 0（所有 checker 均未返回 Ready）
	var got int
	db.Raw("SELECT agent_ready FROM instances WHERE id = ?", inst.ID).Scan(&got)
	if got != 0 {
		t.Errorf("checker 全部失败时 agent_ready 不应被置 1，实际=%d", got)
	}
}

// TestCheckAllAgents_MultipleCheckersPartialReady 多 checker 场景：
// - checkerA 标记 Ready
// - checkerB 标记 NotReady（整体应保持 NotReady）
// 语义：所有 checker 均 Ready 才算就绪。
func TestCheckAllAgents_MultipleCheckersPartialReady(t *testing.T) {
	db := setupAgentTestDB(t)
	user := createAgentTestUser(t, db)

	inst := &model.Instance{
		Name:         "multi-checker-inst",
		UserID:       user.ID,
		InstanceId:   "ins-multi",
		LastCVMState: "RUNNING",
		AgentReady:   0,
	}
	if err := db.Create(inst).Error; err != nil {
		t.Skipf("创建实例失败: %v", err)
	}
	db.Exec("UPDATE instances SET last_cvm_state = 'RUNNING' WHERE id = ?", inst.ID)

	agentCheckers = nil
	RegisterAgentChecker(&mockChecker{
		name:   "a",
		result: map[string]CheckResult{"ins-multi": CheckReady},
	})
	RegisterAgentChecker(&mockChecker{
		name:   "b",
		result: map[string]CheckResult{"ins-multi": CheckNotReady},
	})

	checkAllAgents(context.Background())

	var got int
	db.Raw("SELECT agent_ready FROM instances WHERE id = ?", inst.ID).Scan(&got)
	if got != 0 {
		t.Errorf("一个 checker NotReady 时整体应保持未就绪，实际=%d", got)
	}
}

// TestCheckAllAgents_NoInstanceId 实例 instance_id 为空时不参与检测，
// 避免 TAT 请求因 "" 报错。
func TestCheckAllAgents_NoInstanceId(t *testing.T) {
	db := setupAgentTestDB(t)
	user := createAgentTestUser(t, db)

	inst := &model.Instance{
		Name:         "no-cvm-inst",
		UserID:       user.ID,
		InstanceId:   "",
		LastCVMState: "RUNNING",
		AgentReady:   0,
	}
	if err := db.Create(inst).Error; err != nil {
		t.Skipf("创建实例失败: %v", err)
	}
	db.Exec("UPDATE instances SET last_cvm_state = 'RUNNING', instance_id = '' WHERE id = ?", inst.ID)

	agentCheckers = nil
	callCount := 0
	RegisterAgentChecker(&mockChecker{
		name:   "count",
		result: map[string]CheckResult{},
	})

	// 用自定义 checker 检测是否被调用
	agentCheckers = nil
	RegisterAgentChecker(&countingChecker{n: &callCount})

	checkAllAgents(context.Background())

	// 实例 instance_id 为空时，不应触发 checker.Check（WHERE ... 过滤 + ids 列表空）
	// 注意：由于 DB 其实仍可能返回这条记录（SQL 没过滤 instance_id），
	// 但内部会 if inst.InstanceId != "" 过滤空 id → len(ids)==0 → 直接 return
	if callCount != 0 {
		t.Errorf("instance_id 为空的实例不应触发 checker.Check，实际调用 %d 次", callCount)
	}
}

type countingChecker struct {
	n *int
}

func (c *countingChecker) Name() string { return "counting" }
func (c *countingChecker) Check(ctx context.Context, ids []string) (map[string]CheckResult, error) {
	*c.n++
	res := make(map[string]CheckResult, len(ids))
	for _, id := range ids {
		res[id] = CheckReady
	}
	return res, nil
}
