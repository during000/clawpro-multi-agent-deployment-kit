package task

import (
	"context"
	"errors"
	"testing"
	"time"

	"hatchery/model"

	tcerr "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
)

// ============================================================================
// 目标：提升 task/sg_guardian_task.go 增量覆盖率。
//
// 本文件集中补测以下函数的未覆盖分支：
//   - isInstanceNotFoundErr（4 种错误码路径）
//   - guardianMigrateInstances（happy / NotFound detach / generic error / DB 更新失败 / 零进度兜底）
//   - guardianRescueEmptySGInstances（空 instance_id / 有 instance_id happy / NotFound / 普通 error）
//   - guardianHealMissingSG（复用 ACTIVE / Insert 唯一约束失败 / 残留转 DRAINING）
//
// 所有云 API 调用通过 migrateInstanceSGsFn / createCloudSGWithRetryFn 等已有 hook 拦截，
// 不触发真实 SDK；setupGuardianDB 已自动备份/还原所有 hook。
// ============================================================================

// ---------------------------------------------------------------------------
// isInstanceNotFoundErr
// ---------------------------------------------------------------------------

func TestIsInstanceNotFoundErr_NilErrorReturnsFalse(t *testing.T) {
	if isInstanceNotFoundErr(nil) {
		t.Error("nil error 应返回 false")
	}
}

func TestIsInstanceNotFoundErr_InvalidInstanceIdNotFoundCode(t *testing.T) {
	err := &tcerr.TencentCloudSDKError{Code: "InvalidInstanceId.NotFound", Message: "instance not found"}
	if !isInstanceNotFoundErr(err) {
		t.Error("InvalidInstanceId.NotFound 应返回 true")
	}
}

func TestIsInstanceNotFoundErr_InvalidInstanceNotFoundCode(t *testing.T) {
	err := &tcerr.TencentCloudSDKError{Code: "InvalidInstance.NotFound", Message: "instance not found"}
	if !isInstanceNotFoundErr(err) {
		t.Error("InvalidInstance.NotFound 应返回 true")
	}
}

func TestIsInstanceNotFoundErr_InvalidInstanceIdPrefixCode(t *testing.T) {
	err := &tcerr.TencentCloudSDKError{Code: "InvalidInstanceId.Unknown", Message: "?"}
	if !isInstanceNotFoundErr(err) {
		t.Error("任意 InvalidInstanceId.* 前缀的 code 应返回 true")
	}
}

func TestIsInstanceNotFoundErr_OtherSDKErrorReturnsFalse(t *testing.T) {
	err := &tcerr.TencentCloudSDKError{Code: "AuthFailure.SecretIdNotFound", Message: "bad key"}
	if isInstanceNotFoundErr(err) {
		t.Error("非 InvalidInstance* 的 SDK 错误应返回 false")
	}
}

func TestIsInstanceNotFoundErr_StringMatchFallback(t *testing.T) {
	// 不是 SDK 类型，但消息里包含 "invalidinstanceid" —— 兜底路径
	err := errors.New("some wrapping: InvalidInstanceId.NotFound: boom")
	if !isInstanceNotFoundErr(err) {
		t.Error("错误消息含 'invalidinstanceid' 应返回 true")
	}
}

// ---------------------------------------------------------------------------
// guardianMigrateInstances
// ---------------------------------------------------------------------------

func TestGuardianMigrateInstances_HappyPath(t *testing.T) {
	db := setupGuardianDB(t)
	// 2 条实例绑在 sg-old
	db.Create(&model.Instance{InstanceId: "ins-a", SecurityGroupId: "sg-old"})
	db.Create(&model.Instance{InstanceId: "ins-b", SecurityGroupId: "sg-old"})

	called := 0
	migrateInstanceSGsFn = func(ctx context.Context, _ string, sgs []string) error {
		called++
		if len(sgs) != 1 || sgs[0] != "sg-new" {
			t.Errorf("expected sg-new in call, got %v", sgs)
		}
		return nil
	}

	migrated, failed := guardianMigrateInstances(context.Background(), "sg-old", "sg-new")
	if migrated != 2 || failed != 0 {
		t.Errorf("expected migrated=2 failed=0, got %d/%d", migrated, failed)
	}
	if called != 2 {
		t.Errorf("expected 2 cloud calls, got %d", called)
	}

	// DB 应已全部切到 sg-new
	var cnt int64
	db.Model(&model.Instance{}).Where("security_group_id = ?", "sg-new").Count(&cnt)
	if cnt != 2 {
		t.Errorf("expected 2 rows on sg-new, got %d", cnt)
	}
}

func TestGuardianMigrateInstances_CloudNotFoundDetachesToNewSG(t *testing.T) {
	db := setupGuardianDB(t)
	db.Create(&model.Instance{InstanceId: "ins-gone", SecurityGroupId: "sg-old"})

	migrateInstanceSGsFn = func(ctx context.Context, _ string, _ []string) error {
		return &tcerr.TencentCloudSDKError{Code: "InvalidInstanceId.NotFound", Message: "gone"}
	}

	migrated, failed := guardianMigrateInstances(context.Background(), "sg-old", "sg-new")
	if migrated != 1 || failed != 0 {
		t.Errorf("NotFound 应视为已迁移（detach），got migrated=%d failed=%d", migrated, failed)
	}
	var inst model.Instance
	db.Where("instance_id = ?", "ins-gone").First(&inst)
	if inst.SecurityGroupId != "sg-new" {
		t.Errorf("NotFound 场景应把 DB 的 sg 改到 newSGID，got %q", inst.SecurityGroupId)
	}
}

func TestGuardianMigrateInstances_GenericErrorFailsCountsFailures(t *testing.T) {
	db := setupGuardianDB(t)
	// 1 条普通报错 + 1 条成功；验证失败计数和"成功那条仍然迁成功"
	db.Create(&model.Instance{InstanceId: "ins-err", SecurityGroupId: "sg-old"})
	db.Create(&model.Instance{InstanceId: "ins-ok", SecurityGroupId: "sg-old"})

	migrateInstanceSGsFn = func(ctx context.Context, id string, _ []string) error {
		if id == "ins-err" {
			return errors.New("cloud busy")
		}
		return nil
	}

	migrated, failed := guardianMigrateInstances(context.Background(), "sg-old", "sg-new")
	// 预期：
	//  - migrated=1（ins-ok 迁走）
	//  - failed=2（ins-err 在第一批失败 1 次，第二批 ins-ok 已走只剩 ins-err，再失败一次后
	//    batchProgress==0 触发零进度退出；这是死循环保护的设计行为，不是 bug）
	if migrated != 1 || failed != 2 {
		t.Errorf("expected migrated=1 failed=2, got %d/%d", migrated, failed)
	}

	// 失败那条应仍在 sg-old，成功的应到 sg-new
	var errInst, okInst model.Instance
	db.Where("instance_id = ?", "ins-err").First(&errInst)
	db.Where("instance_id = ?", "ins-ok").First(&okInst)
	if errInst.SecurityGroupId != "sg-old" {
		t.Errorf("失败实例应仍在 sg-old，got %q", errInst.SecurityGroupId)
	}
	if okInst.SecurityGroupId != "sg-new" {
		t.Errorf("成功实例应切到 sg-new，got %q", okInst.SecurityGroupId)
	}
}

func TestGuardianMigrateInstances_ZeroProgressAborts(t *testing.T) {
	db := setupGuardianDB(t)
	// 全部云端报错，1 轮后 batchProgress==0 必须退出，不能无限循环
	for i := 0; i < 5; i++ {
		db.Create(&model.Instance{
			InstanceId:      "ins-busy-" + string(rune('a'+i)),
			SecurityGroupId: "sg-old",
		})
	}
	migrateInstanceSGsFn = func(ctx context.Context, _ string, _ []string) error {
		return errors.New("cloud busy")
	}

	done := make(chan struct{})
	var migrated, failed int
	go func() {
		migrated, failed = guardianMigrateInstances(context.Background(), "sg-old", "sg-new")
		close(done)
	}()

	select {
	case <-done:
		// ok
	case <-time.After(3 * time.Second):
		t.Fatal("guardianMigrateInstances 未在 3 秒内退出（可能死循环）")
	}

	if migrated != 0 {
		t.Errorf("全部失败不应计 migrated，got %d", migrated)
	}
	if failed != 5 {
		t.Errorf("expected failed=5, got %d", failed)
	}
}

func TestGuardianMigrateInstances_ContextCancelledExitsEarly(t *testing.T) {
	db := setupGuardianDB(t)
	db.Create(&model.Instance{InstanceId: "ins-a", SecurityGroupId: "sg-old"})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立刻取消

	migrated, failed := guardianMigrateInstances(ctx, "sg-old", "sg-new")
	if migrated != 0 || failed != 0 {
		t.Errorf("ctx 预先取消应立刻返回 0/0，got %d/%d", migrated, failed)
	}
}

// ---------------------------------------------------------------------------
// guardianRescueEmptySGInstances
// ---------------------------------------------------------------------------

func TestGuardianRescueEmptySGInstances_EmptyInstanceIDUpdatedDBOnly(t *testing.T) {
	db := setupGuardianDB(t)
	// 2 条 sg='' + instance_id=''（早期 placeholder 残留）
	db.Create(&model.Instance{InstanceId: "", SecurityGroupId: ""})
	db.Create(&model.Instance{InstanceId: "", SecurityGroupId: ""})

	cloudCalled := false
	migrateInstanceSGsFn = func(ctx context.Context, _ string, _ []string) error {
		cloudCalled = true
		return nil
	}

	migrated, failed := guardianRescueEmptySGInstances(context.Background(), "sg-new")
	if migrated != 2 || failed != 0 {
		t.Errorf("expected migrated=2 failed=0, got %d/%d", migrated, failed)
	}
	if cloudCalled {
		t.Error("空 instance_id 不应调用云 API")
	}
	var cnt int64
	db.Model(&model.Instance{}).Where("security_group_id = ?", "sg-new").Count(&cnt)
	if cnt != 2 {
		t.Errorf("DB 中应有 2 行被改到 sg-new，got %d", cnt)
	}
}

func TestGuardianRescueEmptySGInstances_WithInstanceIDHappyPath(t *testing.T) {
	db := setupGuardianDB(t)
	db.Create(&model.Instance{InstanceId: "ins-x", SecurityGroupId: ""})

	cloudCalled := 0
	migrateInstanceSGsFn = func(ctx context.Context, _ string, sgs []string) error {
		cloudCalled++
		if len(sgs) != 1 || sgs[0] != "sg-new" {
			t.Errorf("expected target sg-new, got %v", sgs)
		}
		return nil
	}

	migrated, failed := guardianRescueEmptySGInstances(context.Background(), "sg-new")
	if migrated != 1 || failed != 0 {
		t.Errorf("expected migrated=1 failed=0, got %d/%d", migrated, failed)
	}
	if cloudCalled != 1 {
		t.Errorf("expected 1 cloud call, got %d", cloudCalled)
	}
}

func TestGuardianRescueEmptySGInstances_CloudNotFoundOnlyUpdatesDB(t *testing.T) {
	db := setupGuardianDB(t)
	db.Create(&model.Instance{InstanceId: "ins-gone", SecurityGroupId: ""})

	migrateInstanceSGsFn = func(ctx context.Context, _ string, _ []string) error {
		return &tcerr.TencentCloudSDKError{Code: "InvalidInstanceId.NotFound"}
	}

	migrated, failed := guardianRescueEmptySGInstances(context.Background(), "sg-new")
	if migrated != 1 || failed != 0 {
		t.Errorf("NotFound 应视为已救援，got migrated=%d failed=%d", migrated, failed)
	}
	var inst model.Instance
	db.Where("instance_id = ?", "ins-gone").First(&inst)
	if inst.SecurityGroupId != "sg-new" {
		t.Errorf("NotFound 场景应在 DB 中把 sg 改到 sg-new，got %q", inst.SecurityGroupId)
	}
}

func TestGuardianRescueEmptySGInstances_GenericErrorCountsFailure(t *testing.T) {
	db := setupGuardianDB(t)
	db.Create(&model.Instance{InstanceId: "ins-busy", SecurityGroupId: ""})

	migrateInstanceSGsFn = func(ctx context.Context, _ string, _ []string) error {
		return errors.New("cloud busy")
	}

	migrated, failed := guardianRescueEmptySGInstances(context.Background(), "sg-new")
	if migrated != 0 || failed != 1 {
		t.Errorf("expected migrated=0 failed=1, got %d/%d", migrated, failed)
	}
	var inst model.Instance
	db.Where("instance_id = ?", "ins-busy").First(&inst)
	if inst.SecurityGroupId != "" {
		t.Errorf("失败实例 DB 不应被改动，got %q", inst.SecurityGroupId)
	}
}

func TestGuardianRescueEmptySGInstances_EmptyPoolReturnsEarly(t *testing.T) {
	_ = setupGuardianDB(t)
	// DB 没有任何 sg='' 实例 → 空查询 → 直接 0/0
	migrated, failed := guardianRescueEmptySGInstances(context.Background(), "sg-new")
	if migrated != 0 || failed != 0 {
		t.Errorf("expected 0/0, got %d/%d", migrated, failed)
	}
}

// ---------------------------------------------------------------------------
// guardianHealMissingSG 剩余分支
// ---------------------------------------------------------------------------

func TestGuardianHealMissingSG_ReusesExistingActiveSG(t *testing.T) {
	db := setupGuardianDB(t)
	rs := seedGuardianRuleSet(t, db)

	// 失踪的老 SG
	old := model.ManagedSGPool{SGID: "sg-missing", RuleSetID: rs.ID, Status: model.SGStatusActive, CVMCount: 3}
	db.Create(&old)
	// 同 RuleSet 下已有一个 cvm_count=1 的 ACTIVE SG（低于默认阈值 1500），应被复用
	reusable := model.ManagedSGPool{SGID: "sg-reuse-me", RuleSetID: rs.ID, Status: model.SGStatusActive, CVMCount: 1}
	db.Create(&reusable)

	// 老 SG 上挂 2 条实例，迁移过去后应转到 sg-reuse-me
	db.Create(&model.Instance{InstanceId: "ins-a", SecurityGroupId: "sg-missing"})
	db.Create(&model.Instance{InstanceId: "ins-b", SecurityGroupId: "sg-missing"})

	// 禁止创建新 SG —— 验证走的是"复用"分支
	createCalled := false
	createCloudSGWithRetryFn = func(ctx context.Context, _, _ string) (string, error) {
		createCalled = true
		return "sg-should-not-create", nil
	}
	migrateInstanceSGsFn = func(ctx context.Context, _ string, _ []string) error { return nil }

	guardianHealMissingSG(context.Background(), old, rs)

	if createCalled {
		t.Error("已有可复用的 ACTIVE SG 时，不应再创建新 SG")
	}
	// 两条实例应被迁到 sg-reuse-me
	var cnt int64
	db.Model(&model.Instance{}).Where("security_group_id = ?", "sg-reuse-me").Count(&cnt)
	if cnt != 2 {
		t.Errorf("expected 2 rows migrated to sg-reuse-me, got %d", cnt)
	}
	// 老 SG 应被标 RETIRED（全部迁完 + 0 残留）
	var afterOld model.ManagedSGPool
	db.Where("sg_id = ?", "sg-missing").First(&afterOld)
	if afterOld.Status != model.SGStatusRetired {
		t.Errorf("老 SG 应标 RETIRED，got %s", afterOld.Status)
	}
}

func TestGuardianHealMissingSG_RemainingInstancesMarksDraining(t *testing.T) {
	db := setupGuardianDB(t)
	rs := seedGuardianRuleSet(t, db)

	old := model.ManagedSGPool{SGID: "sg-partial", RuleSetID: rs.ID, Status: model.SGStatusActive, CVMCount: 2}
	db.Create(&old)
	// 2 条实例，一条成功、一条失败 → 还有 1 条残留
	db.Create(&model.Instance{InstanceId: "ins-ok", SecurityGroupId: "sg-partial"})
	db.Create(&model.Instance{InstanceId: "ins-fail", SecurityGroupId: "sg-partial"})

	createCloudSGWithRetryFn = func(ctx context.Context, _, _ string) (string, error) { return "sg-healed", nil }
	applyRulesToCloudSGWithRetryFn = func(ctx context.Context, _, _ string) error { return nil }
	migrateInstanceSGsFn = func(ctx context.Context, id string, _ []string) error {
		if id == "ins-fail" {
			return errors.New("cloud busy")
		}
		return nil
	}

	guardianHealMissingSG(context.Background(), old, rs)

	// 老 SG 必须变成 DRAINING（不是 RETIRED），cvm_count 应回到 1
	var afterOld model.ManagedSGPool
	db.Where("sg_id = ?", "sg-partial").First(&afterOld)
	if afterOld.Status != model.SGStatusDraining {
		t.Errorf("有残留实例时应标 DRAINING，got %s", afterOld.Status)
	}
	if afterOld.CVMCount != 1 {
		t.Errorf("DRAINING 时 cvm_count 应对齐实际残留数=1，got %d", afterOld.CVMCount)
	}
}
