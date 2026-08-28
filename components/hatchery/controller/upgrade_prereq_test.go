// Package controller 中 upgrade_prereq.go 的单元测试。
//
// 这个测试文件覆盖 prepareInstanceForUpgrade 和 startUpgradeForInstance 两个共用函数
// 的所有 outcome 分支，确保单实例与批量升级入口共用的核心逻辑（OpenClaw 唯一支持检查 /
// 防重入 / providerKeys / runtime_user 校正 / needUpgrade 判定 / 操作锁 / 异步启动）
// 在重构后行为完全正确。
package controller

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// ─── 测试辅助 ────────────────────────────────────────────────────────────────

// initPrereqTestDB 初始化一个内存 SQLite，并完成必要表迁移，返回 *gorm.DB。
// 与现有 initUpgradeTestDB 不同的是，本 helper 不会迁移 SiteConfig（本文件无此需求）。
func initPrereqTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.User{}, &model.Instance{}, &model.InstanceAdjustment{}, &model.AIImage{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	return db
}

// stubCheckOpenclawConfigFn 在测试期间替换 checkOpenclawConfigFn。
// 单测无法走真实 TAT，所以这里默认返回空字符串（视为 openclaw.json 不存在 → 跳过检查）。
// 当传入 emit 函数时，用 emit 决定该 stub 的行为，便于覆盖"读取失败/JSON 非法/含非法 key"等分支。
func stubCheckOpenclawConfigFn(t *testing.T, emit func(instanceId string) (string, error)) {
	t.Helper()
	orig := checkOpenclawConfigFn
	checkOpenclawConfigFn = func(_ context.Context, instanceId, _ string, _ uint64) (string, error) {
		if emit == nil {
			return "", nil
		}
		return emit(instanceId)
	}
	t.Cleanup(func() { checkOpenclawConfigFn = orig })
}

// makeRunningCVMInfo 构造一个让 ResolveInstanceStatus 判定为 running 的 CVMInstanceInfo。
// 调用方可通过 imageId 控制 needUpgrade 判定（与目标镜像 ID 不同时 needUpgrade=true）。
func makeRunningCVMInfo(imageId string) *CVMInstanceInfo {
	return &CVMInstanceInfo{
		State:         "RUNNING",
		RestrictState: "NORMAL",
		ImageId:       imageId,
	}
}

// runningInstance 创建一个能通过 needUpgrade 判定的 OpenClaw 实例（agent_type=openclaw,
// agent_ready=1, current_operation=空），便于复用。
func runningOpenClawInstance(t *testing.T, db *gorm.DB, name, instanceId string) *model.Instance {
	t.Helper()
	user := &model.User{Username: name + "-user", Password: "x", Role: "user"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	inst := &model.Instance{
		Name:       name,
		InstanceId: instanceId,
		UserID:     user.ID,
		AgentType:  "openclaw",
		AgentReady: 1,
	}
	if err := db.Create(inst).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}
	return inst
}

// ─── prepareInstanceForUpgrade 单元测试 ─────────────────────────────────────

// 1) 全部检查通过：OpenClaw 实例 + 无 providerKeys 问题 + current_operation 空 +
//    目标镜像非官方镜像（runtime_user 校正会因 IsCandidateImage=false 直接跳过）。
func TestPrepareInstanceForUpgrade_OK(t *testing.T) {
	db := initPrereqTestDB(t)
	stubCheckOpenclawConfigFn(t, nil) // 视同 openclaw.json 不存在 → providerKeys 通过
	inst := runningOpenClawInstance(t, db, "ok-inst", "ins-prereq-ok-001")

	outcome := prepareInstanceForUpgrade(context.Background(), inst, &model.AIImage{ImageId: "img-non-official"}, "[Test]")
	if !outcome.OK {
		t.Fatalf("expected OK=true, got outcome=%+v", outcome)
	}
	if outcome.Err != nil || outcome.HTTPCode != 0 || outcome.BatchStatus != "" {
		t.Errorf("expected zero-value error fields when OK, got %+v", outcome)
	}
}

// 2) providerKeys 检查不通过：返回 HTTP 400 / batch=failed。
//    通过 stub checkOpenclawConfigFn 返回一个含非法 key 的 JSON，触发 fmt.Errorf 失败路径。
func TestPrepareInstanceForUpgrade_ProviderKeysInvalid(t *testing.T) {
	db := initPrereqTestDB(t)
	// 返回一个含非法 key（带 "/"，与默认 providerKeyForbiddenChars 匹配）的 openclaw.json
	stubCheckOpenclawConfigFn(t, func(_ string) (string, error) {
		return `{"models":{"providers":{"bad/key":{}}}}`, nil
	})
	inst := runningOpenClawInstance(t, db, "bad-cfg", "ins-prereq-bad-001")

	outcome := prepareInstanceForUpgrade(context.Background(), inst, &model.AIImage{ImageId: "img-x"}, "[Test]")
	if outcome.OK {
		t.Fatalf("expected OK=false on invalid provider keys, got %+v", outcome)
	}
	if outcome.HTTPCode != http.StatusBadRequest {
		t.Errorf("expected HTTP 400, got %d", outcome.HTTPCode)
	}
	if outcome.BatchStatus != "failed" {
		t.Errorf("expected BatchStatus=failed, got %q", outcome.BatchStatus)
	}
	if outcome.Err == nil || !strings.Contains(outcome.Err.Error(), "openclaw.json") {
		t.Errorf("expected Err to mention openclaw.json, got %v", outcome.Err)
	}
}

// 3) 防重入：current_operation != "" && state == processing → HTTP 409 / batch=skipped。
func TestPrepareInstanceForUpgrade_OperationInProgress(t *testing.T) {
	db := initPrereqTestDB(t)
	stubCheckOpenclawConfigFn(t, nil)
	inst := runningOpenClawInstance(t, db, "busy-inst", "ins-prereq-busy-001")
	now := time.Now()
	inst.CurrentOperation = model.OpReinstall
	inst.CurrentOperationState = model.OpStateProcessing
	inst.CurrentOperationUpdatedAt = &now

	outcome := prepareInstanceForUpgrade(context.Background(), inst, &model.AIImage{ImageId: "img-x"}, "[Test]")
	if outcome.OK {
		t.Fatalf("expected OK=false when operation in progress, got %+v", outcome)
	}
	if outcome.HTTPCode != http.StatusConflict {
		t.Errorf("expected HTTP 409, got %d", outcome.HTTPCode)
	}
	if outcome.BatchStatus != "skipped" {
		t.Errorf("expected BatchStatus=skipped, got %q", outcome.BatchStatus)
	}
	if outcome.Err == nil || !strings.Contains(outcome.Err.Error(), "reinstall") {
		t.Errorf("expected Err to mention current operation 'reinstall', got %v", outcome.Err)
	}
}

// 4) Hermes 现已支持一键升级（backup_pre_reinstall_hermes.sh + restore_post_reinstall_hermes.sh）：
//    前置检查应当通过（OK=true），与 OpenClaw 走同一份 prepareInstanceForUpgrade 链路。
//    历史：原测试断言"Hermes 拒绝升级"，随 SupportsUpgrade 放开而调整。
func TestPrepareInstanceForUpgrade_HermesSupported(t *testing.T) {
	db := initPrereqTestDB(t)
	stubCheckOpenclawConfigFn(t, nil)
	user := &model.User{Username: "hermes-user", Password: "x", Role: "user"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	inst := &model.Instance{
		Name:       "hermes-inst",
		InstanceId: "ins-prereq-hermes-001",
		UserID:     user.ID,
		AgentType:  "hermes",
		AgentReady: 1,
	}
	if err := db.Create(inst).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}

	outcome := prepareInstanceForUpgrade(context.Background(), inst, &model.AIImage{ImageId: "img-x"}, "[Test]")
	if !outcome.OK {
		t.Fatalf("expected OK=true for hermes type (SupportsUpgrade=true), got %+v", outcome)
	}
	if outcome.Err != nil {
		t.Errorf("expected nil Err on success, got %v", outcome.Err)
	}
}

// 5) LightclawACE 类型同样不支持升级，与 hermes 走同一分支，再补一例做"广谱"验证。
func TestPrepareInstanceForUpgrade_LightclawACENotSupported(t *testing.T) {
	db := initPrereqTestDB(t)
	stubCheckOpenclawConfigFn(t, nil)
	user := &model.User{Username: "lightclaw-user", Password: "x", Role: "user"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	inst := &model.Instance{
		Name:       "lightclaw-inst",
		InstanceId: "ins-prereq-lc-001",
		UserID:     user.ID,
		AgentType:  "lightclawace",
		AgentReady: 1,
	}
	if err := db.Create(inst).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}

	outcome := prepareInstanceForUpgrade(context.Background(), inst, &model.AIImage{ImageId: "img-x"}, "[Test]")
	if outcome.OK {
		t.Fatalf("expected OK=false for lightclawace type, got %+v", outcome)
	}
	if outcome.HTTPCode != http.StatusBadRequest {
		t.Errorf("expected HTTP 400, got %d", outcome.HTTPCode)
	}
	if outcome.BatchStatus != "failed" {
		t.Errorf("expected BatchStatus=failed, got %q", outcome.BatchStatus)
	}
}

// 6) ensureOfficialImageRuntimeUserForUpgrade 路径：非官方镜像 → 直接 return nil（不写 DB），
//    OK=true。确保该步骤对"非官方镜像"幂等无副作用，符合"先检查后写"的原则。
//    （注意：DB 写失败的 500 分支需要构造 DB 错误，单测里 SQLite + In-Memory 难以复现，
//     但该分支的逻辑只是把 ensureOfficial... 的 error 包装一层，复杂度极低，
//     由 ensureOfficialImageRuntimeUser 自身的现有测试覆盖即可。）
func TestPrepareInstanceForUpgrade_NonOfficialImage_NoDBWrite(t *testing.T) {
	db := initPrereqTestDB(t)
	stubCheckOpenclawConfigFn(t, nil)
	inst := runningOpenClawInstance(t, db, "non-official", "ins-prereq-no-001")
	// 故意把 RuntimeUser 设成非 root，验证非官方镜像不会触发校正
	inst.RuntimeUser = "ubuntu"
	inst.RuntimeHome = "/home/ubuntu"
	if err := db.Save(inst).Error; err != nil {
		t.Fatalf("更新实例失败: %v", err)
	}

	outcome := prepareInstanceForUpgrade(context.Background(), inst, &model.AIImage{ImageId: "img-private-not-candidate"}, "[Test]")
	if !outcome.OK {
		t.Fatalf("expected OK=true on non-official image path, got %+v", outcome)
	}

	// DB 中 runtime_user 不应被改写
	var reloaded model.Instance
	if err := db.First(&reloaded, inst.ID).Error; err != nil {
		t.Fatalf("重新加载实例失败: %v", err)
	}
	if reloaded.RuntimeUser != "ubuntu" {
		t.Errorf("expected runtime_user 保持 ubuntu，got %q", reloaded.RuntimeUser)
	}
}

// ─── startUpgradeForInstance 单元测试 ───────────────────────────────────────

// 7) checkNeedsUpgrade 失败：单测里不传 cvmInfoMap → 内部 batchFetchCVMInfoMap 拿不到 info →
//    返回 "无法获取 CVM 实例信息" 错误 → 命中 Err 分支（HTTP 500 / batch=failed）。
func TestStartUpgradeForInstance_CheckNeedsUpgradeFailed(t *testing.T) {
	db := initPrereqTestDB(t)
	inst := runningOpenClawInstance(t, db, "check-fail", "ins-start-fail-001")
	img := &model.AIImage{ImageId: "img-target", Enabled: true, AgentType: "openclaw"}
	if err := db.Create(img).Error; err != nil {
		t.Fatalf("创建镜像失败: %v", err)
	}

	// 不传 cvmInfoMap，内部走真实 batchFetchCVMInfoMap，无 CVM client → API_ERROR / 空映射。
	outcome := startUpgradeForInstance(context.Background(), inst, img, nil, "[Test]")
	if outcome.Started || outcome.AlreadyLatest {
		t.Fatalf("expected fail outcome, got %+v", outcome)
	}
	if outcome.Err == nil {
		t.Fatal("expected Err to be set")
	}
	if outcome.HTTPCode != http.StatusInternalServerError {
		t.Errorf("expected HTTP 500, got %d", outcome.HTTPCode)
	}
	if outcome.BatchStatus != "failed" {
		t.Errorf("expected BatchStatus=failed, got %q", outcome.BatchStatus)
	}
	if !strings.Contains(outcome.Err.Error(), "检查升级状态失败") {
		t.Errorf("expected wrapped error message, got %v", outcome.Err)
	}
}

// 8) AlreadyLatest 分支：CVMInstanceInfo.ImageId == defaultImage.ImageId 且 agent_version 一致。
func TestStartUpgradeForInstance_AlreadyLatest(t *testing.T) {
	db := initPrereqTestDB(t)
	inst := runningOpenClawInstance(t, db, "latest-inst", "ins-start-latest-001")
	inst.AgentVersion = "1.0.0"
	if err := db.Save(inst).Error; err != nil {
		t.Fatalf("更新版本失败: %v", err)
	}
	img := &model.AIImage{
		ImageId:      "img-same",
		Enabled:      true,
		AgentType:    "openclaw",
		AgentVersion: "1.0.0",
	}
	if err := db.Create(img).Error; err != nil {
		t.Fatalf("创建镜像失败: %v", err)
	}
	cvmInfoMap := map[string]*CVMInstanceInfo{
		inst.InstanceId: makeRunningCVMInfo("img-same"), // 与 image.ImageId 一致
	}

	outcome := startUpgradeForInstance(context.Background(), inst, img, cvmInfoMap, "[Test]")
	if !outcome.AlreadyLatest {
		t.Fatalf("expected AlreadyLatest=true, got %+v", outcome)
	}
	if outcome.Started {
		t.Error("Started must be false when AlreadyLatest")
	}
	if outcome.Err != nil {
		t.Errorf("expected Err=nil on AlreadyLatest, got %v", outcome.Err)
	}
	if outcome.CurrentImageId != "img-same" || outcome.TargetImageId != "img-same" {
		t.Errorf("expected image ids reported, got current=%q target=%q",
			outcome.CurrentImageId, outcome.TargetImageId)
	}

	// DB 操作锁不应被设置（AlreadyLatest 不进入 setOperation 分支）
	var reloaded model.Instance
	if err := db.First(&reloaded, inst.ID).Error; err != nil {
		t.Fatalf("重新加载失败: %v", err)
	}
	if reloaded.CurrentOperation != "" {
		t.Errorf("expected current_operation 未被设置，got %q", reloaded.CurrentOperation)
	}
}

// 9) Started 分支：镜像 ID 不一致 → needUpgrade=true → 设置操作锁 → 启动异步 goroutine。
//    本用例的核心断言是"操作锁已设置 + 返回 Started"。为避免真实 performUpgrade 在
//    后台 goroutine 里访问未迁移的 notifications / site_configs / skill_installations
//    等表（goroutine 还可能跨越测试边界，把 SQL 落到后续测试的 DB 上，造成 "no such table"
//    级别的串扰），这里 stub 掉 startUpgradePerformFn，并通过 channel 同步等待 goroutine
//    被实际调用，确保测试退出前不留泄漏的后台执行。
func TestStartUpgradeForInstance_StartedAndOperationLocked(t *testing.T) {
	db := initPrereqTestDB(t)
	inst := runningOpenClawInstance(t, db, "start-inst", "ins-start-go-001")
	img := &model.AIImage{ImageId: "img-new", Enabled: true, AgentType: "openclaw"}
	if err := db.Create(img).Error; err != nil {
		t.Fatalf("创建镜像失败: %v", err)
	}
	cvmInfoMap := map[string]*CVMInstanceInfo{
		// CVM 当前镜像与目标不同 → needUpgrade=true
		inst.InstanceId: makeRunningCVMInfo("img-old"),
	}

	// stub 升级钩子：让异步 goroutine 立即返回 nil，并通过 channel 通知测试已被调用。
	// 这样既能验证"goroutine 真的被启动了"，又彻底避免真实 performUpgrade 在后台运行。
	called := make(chan struct {
		instanceId string
		target     string
		current    string
	}, 1)
	origPerformFn := startUpgradePerformFn
	startUpgradePerformFn = func(_ context.Context, inst *model.Instance, target, current string) error {
		called <- struct {
			instanceId string
			target     string
			current    string
		}{inst.InstanceId, target, current}
		return nil
	}
	t.Cleanup(func() { startUpgradePerformFn = origPerformFn })

	outcome := startUpgradeForInstance(context.Background(), inst, img, cvmInfoMap, "[Test]")
	if !outcome.Started {
		t.Fatalf("expected Started=true, got %+v", outcome)
	}
	if outcome.Err != nil {
		t.Errorf("expected Err=nil on Started, got %v", outcome.Err)
	}

	// 验证 DB 操作锁已设置为 OpUpgrade / processing
	var reloaded model.Instance
	if err := db.First(&reloaded, inst.ID).Error; err != nil {
		t.Fatalf("重新加载失败: %v", err)
	}
	if reloaded.CurrentOperation != model.OpUpgrade {
		t.Errorf("expected current_operation=upgrade, got %q", reloaded.CurrentOperation)
	}
	if reloaded.CurrentOperationState != model.OpStateProcessing {
		t.Errorf("expected current_operation_state=processing, got %q", reloaded.CurrentOperationState)
	}
	// 内存对象也应被同步
	if inst.CurrentOperation != model.OpUpgrade {
		t.Errorf("expected in-memory current_operation 已同步，got %q", inst.CurrentOperation)
	}

	// 同步等待异步 goroutine 真正调用了升级钩子，避免测试退出后还有后台执行。
	select {
	case got := <-called:
		if got.instanceId != inst.InstanceId {
			t.Errorf("expected stub 收到 instance_id=%q, got %q", inst.InstanceId, got.instanceId)
		}
		if got.target != img.ImageId {
			t.Errorf("expected stub 收到 target=%q, got %q", img.ImageId, got.target)
		}
		if got.current != "img-old" {
			t.Errorf("expected stub 收到 current=%q, got %q", "img-old", got.current)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("异步升级 goroutine 未在预期时间内调用 startUpgradePerformFn")
	}
}

// 10) setOperation 失败 → HTTP 409 / batch=failed。
//     构造方法：先把实例 current_operation 锁成 reinstall（非 upgrade），setOperation 的
//     WHERE 子句 (current_operation = '' OR current_operation = 'upgrade') 不命中 →
//     RowsAffected=0 → 返回 ErrOperationInProgress。
//     注意：要走到 setOperation，必须先绕过 prepareInstanceForUpgrade 的防重入检查。
//     因此此用例直接调 startUpgradeForInstance（它不做防重入），构造 DB 上有锁但内存对象
//     里 CurrentOperation 为空的不一致状态——这正好是"防重入是单独的检查、
//     setOperation 的 race-condition 兜底机制"这一设计语义。
func TestStartUpgradeForInstance_SetOperationConflict(t *testing.T) {
	db := initPrereqTestDB(t)
	inst := runningOpenClawInstance(t, db, "lock-inst", "ins-start-lock-001")
	img := &model.AIImage{ImageId: "img-new", Enabled: true, AgentType: "openclaw"}
	if err := db.Create(img).Error; err != nil {
		t.Fatalf("创建镜像失败: %v", err)
	}

	// 模拟 race：DB 端被另一进程占用为 reinstall 锁，但传入的 inst 内存对象还没拿到这个变更。
	now := time.Now()
	if err := db.Model(&model.Instance{}).Where("id = ?", inst.ID).Updates(map[string]interface{}{
		"current_operation":            model.OpReinstall,
		"current_operation_state":      model.OpStateProcessing,
		"current_operation_updated_at": &now,
	}).Error; err != nil {
		t.Fatalf("DB 直接占锁失败: %v", err)
	}

	cvmInfoMap := map[string]*CVMInstanceInfo{
		inst.InstanceId: makeRunningCVMInfo("img-old"), // needUpgrade=true
	}

	outcome := startUpgradeForInstance(context.Background(), inst, img, cvmInfoMap, "[Test]")
	if outcome.Started || outcome.AlreadyLatest {
		t.Fatalf("expected fail outcome, got %+v", outcome)
	}
	if outcome.HTTPCode != http.StatusConflict {
		t.Errorf("expected HTTP 409 on setOperation conflict, got %d", outcome.HTTPCode)
	}
	if outcome.BatchStatus != "failed" {
		t.Errorf("expected BatchStatus=failed, got %q", outcome.BatchStatus)
	}
	if outcome.Err == nil || !strings.Contains(outcome.Err.Error(), "设置升级操作锁失败") {
		t.Errorf("expected wrapped operation-lock error, got %v", outcome.Err)
	}
	if !errors.Is(outcome.Err, ErrOperationInProgress) {
		t.Errorf("expected wrapped error chain to include ErrOperationInProgress, got %v", outcome.Err)
	}
}

// 11) cvmInfoMap=nil 路径：单实例侧调用形态。
//     已由 TestStartUpgradeForInstance_CheckNeedsUpgradeFailed 覆盖（传 nil 时走自查路径并失败）。
//     这里再补一个 cvmInfoMap=nil 但显式构造一个让 checkNeedsUpgrade 走完整查询 fail 的用例，
//     断言"传 nil 与传空 map 都会让函数走自查分支并优雅失败"。
func TestStartUpgradeForInstance_NilCvmInfoMapTriggersSelfFetch(t *testing.T) {
	db := initPrereqTestDB(t)
	inst := runningOpenClawInstance(t, db, "self-fetch", "ins-start-self-001")
	img := &model.AIImage{ImageId: "img-y", Enabled: true, AgentType: "openclaw"}
	if err := db.Create(img).Error; err != nil {
		t.Fatalf("创建镜像失败: %v", err)
	}

	// 显式传 nil（单实例入口形态）
	outcomeA := startUpgradeForInstance(context.Background(), inst, img, nil, "[Test]")
	if outcomeA.Started || outcomeA.AlreadyLatest {
		t.Fatalf("expected fail outcome on nil map, got %+v", outcomeA)
	}

	// 显式传空 map（变体），走批量入口形态但 map 中没有该 instanceId
	outcomeB := startUpgradeForInstance(context.Background(), inst, img,
		map[string]*CVMInstanceInfo{}, "[Test]")
	if outcomeB.Started || outcomeB.AlreadyLatest {
		t.Fatalf("expected fail outcome on empty map, got %+v", outcomeB)
	}
}

// ─── step 0 (rejectDowngradeOnOfficialImage) 通过 prepare 入口生效的回归测试 ──────
//
// 这两条用例确认：「拒绝官方镜像降级」检查在 prepareInstanceForUpgrade 内部按
// step 0 顺序执行，单实例 / 批量入口都能自动覆盖到，不需要任何调用方再 inline 写一遍。
// rejectDowngradeOnOfficialImage 的纯函数级分支由 openclaw_upgrade_reject_downgrade_test.go
// 单独覆盖，本处只验证「集成到 prepare 入口后的输出语义」。

// 13) 命中降级拒绝：官方镜像 + OpenClaw 实例 + 当前版本 > 镜像版本 →
//     prepareInstanceForUpgrade 应在 step 0 立即返回 HTTP 400 / batch=failed，
//     且不应进入后续的 providerKeys / 防重入 / SupportsUpgrade / runtime_user 校正步骤。
func TestPrepareInstanceForUpgrade_DowngradeRejected(t *testing.T) {
	db := initPrereqTestDB(t)
	// 让 providerKeys 检查永不被触达：若实际触达，stub 返回非法 key 会让用例以错误的
	// 错误信息断言失败，从而暴露 step 0 没有提前 return。
	stubCheckOpenclawConfigFn(t, func(_ string) (string, error) {
		return `{"models":{"providers":{"unreachable/key":{}}}}`, nil
	})
	inst := runningOpenClawInstance(t, db, "downgrade-inst", "ins-prereq-down-001")
	inst.AgentVersion = "9999.99.99" // 远高于任何官方镜像声明版本
	if err := db.Save(inst).Error; err != nil {
		t.Fatalf("更新版本失败: %v", err)
	}

	// 选用 common.CandidateImages 中的真实官方镜像 ID（OpenClaw 系列）
	officialImage := &model.AIImage{
		ImageId:      "img-idzg74s9",
		AgentType:    "openclaw",
		AgentVersion: "2026.5.7",
	}

	outcome := prepareInstanceForUpgrade(context.Background(), inst, officialImage, "[Test]")
	if outcome.OK {
		t.Fatalf("expected OK=false on downgrade, got %+v", outcome)
	}
	if outcome.HTTPCode != http.StatusBadRequest {
		t.Errorf("expected HTTP 400, got %d", outcome.HTTPCode)
	}
	if outcome.BatchStatus != "failed" {
		t.Errorf("expected BatchStatus=failed, got %q", outcome.BatchStatus)
	}
	if outcome.Err == nil ||
		!strings.Contains(outcome.Err.Error(), "9999.99.99") ||
		!strings.Contains(outcome.Err.Error(), "2026.5.7") {
		t.Errorf("expected Err to mention both versions, got %v", outcome.Err)
	}
	// 关键反向断言：错误信息不应是 providerKeys 检查里的 "openclaw.json"，
	// 否则说明 step 0 没有提前 return，检查顺序不符合设计。
	if outcome.Err != nil && strings.Contains(outcome.Err.Error(), "openclaw.json") {
		t.Errorf("step 0 应在 providerKeys 检查之前 return，但实际命中了 providerKeys 错误: %v", outcome.Err)
	}
}

// 14) 同版本不拒绝：官方镜像 + OpenClaw 实例 + 当前版本 == 镜像版本 → step 0 放行，
//     继续走后续检查。这里用 stub 让 providerKeys 通过、runtime_user 校正幂等通过，
//     最终应得到 OK=true。
func TestPrepareInstanceForUpgrade_SameVersionNoDowngrade(t *testing.T) {
	db := initPrereqTestDB(t)
	stubCheckOpenclawConfigFn(t, nil)
	inst := runningOpenClawInstance(t, db, "same-ver", "ins-prereq-same-001")
	inst.AgentVersion = "2026.5.7"
	if err := db.Save(inst).Error; err != nil {
		t.Fatalf("更新版本失败: %v", err)
	}

	officialImage := &model.AIImage{
		ImageId:      "img-idzg74s9",
		AgentType:    "openclaw",
		AgentVersion: "2026.5.7",
	}

	outcome := prepareInstanceForUpgrade(context.Background(), inst, officialImage, "[Test]")
	if !outcome.OK {
		t.Fatalf("expected OK=true on same version, got %+v", outcome)
	}
}

// 15) defaultImage=nil 防御：调用方传 nil 时 prepare 不应 panic，且整体仍能完成
//     非降级类的检查（step 0 内部对 nil 容忍并 return nil）。
//     此用例同时构成对 admin_instances.go / openclaw_upgrade.go 调用方的契约保护——
//     即使未来某个调用方没拿到 defaultImage 就误传 nil，也只是跳过 image 相关判断，
//     不会导致进程崩溃。
func TestPrepareInstanceForUpgrade_NilImageDoesNotPanic(t *testing.T) {
	db := initPrereqTestDB(t)
	stubCheckOpenclawConfigFn(t, nil)
	inst := runningOpenClawInstance(t, db, "nil-img", "ins-prereq-nilimg-001")

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("nil defaultImage caused panic: %v", r)
		}
	}()

	outcome := prepareInstanceForUpgrade(context.Background(), inst, nil, "[Test]")
	if !outcome.OK {
		t.Fatalf("expected OK=true on nil image (other checks should still pass), got %+v", outcome)
	}
}

// ─── 并发安全性的快速烟雾测试 ────────────────────────────────────────────────
// 16) prepareInstanceForUpgrade 是否能被多协程并发调用而不破坏内部状态。
//     这是一项防御性测试：函数本身无全局可变状态，但通过 stub 的全局 var
//     checkOpenclawConfigFn 是有共享态的；并发执行相同 stub 行为应保持稳定。
func TestPrepareInstanceForUpgrade_ConcurrentSafe(t *testing.T) {
	db := initPrereqTestDB(t)
	stubCheckOpenclawConfigFn(t, nil)
	inst := runningOpenClawInstance(t, db, "concur-inst", "ins-prereq-concur-001")

	const goroutines = 16
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			out := prepareInstanceForUpgrade(context.Background(), inst, &model.AIImage{ImageId: "img-non-official"}, "[Test]")
			if !out.OK {
				t.Errorf("expected OK on concurrent call, got %+v", out)
			}
		}()
	}
	wg.Wait()
}
