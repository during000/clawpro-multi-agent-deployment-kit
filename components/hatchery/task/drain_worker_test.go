package task

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	"gorm.io/gorm"
)

// ============================================================================
// task/drain_worker.go 单元测试
//
// 覆盖：drainTick / drainOneInstance / dedupStrings。
// 不覆盖：真实云 API（listInstancesBoundToSGs / modifyInstanceSGs）— 用 hook 替身。
// ============================================================================

// setupDrainDB 内存 SQLite 建齐表，替换云 API / RuleSet 相关 hook 为可控 stub。
func setupDrainDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.RuleSet{},
		&model.ManagedSGPool{},
		&model.SiteConfig{},
		&model.Instance{},
		&model.SGDrainState{},
		&model.AuditLog{},
	); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	origDB := model.UseDBForTest(db)

	origList := drainListFn
	origModify := drainModifyFn
	origGetRS := drainGetDefaultRuleSetFn
	origSelect := drainSelectSGFn
	origAudit := drainLogAuditFn

	// 默认 stub：无可搬实例 / 换绑成功
	drainListFn = func(_ context.Context, _ []string, _ int) ([]instanceToDrain, error) { return nil, nil }
	drainModifyFn = func(_ context.Context, _ string, _ []string) error { return nil }
	drainGetDefaultRuleSetFn = func(ctx context.Context) (*model.RuleSet, error) {
		var rs model.RuleSet
		if err := model.DB(ctx).Where("is_default = ?", true).First(&rs).Error; err != nil {
			return nil, err
		}
		return &rs, nil
	}
	drainSelectSGFn = func(_ context.Context, _ string, _ uint) (string, bool, error) {
		return "sg-target", false, nil
	}
	drainLogAuditFn = func(_ context.Context, _ time.Time, _ uint, _, _, _, _, _ string) {}

	t.Cleanup(func() {
		drainListFn = origList
		drainModifyFn = origModify
		drainGetDefaultRuleSetFn = origGetRS
		drainSelectSGFn = origSelect
		drainLogAuditFn = origAudit
		origDB()
	})
	return db
}

func seedDrainRuleSet(t *testing.T, db *gorm.DB) *model.RuleSet {
	t.Helper()
	rs := &model.RuleSet{
		Name:         model.DefaultRuleSetName,
		Description:  "drain test",
		Rules:        "[]",
		Version:      1,
		UserGroupIDs: "[]",
		IsDefault:    true,
	}
	if err := db.Create(rs).Error; err != nil {
		t.Fatalf("seed rs: %v", err)
	}
	return rs
}

// ------------------------------------------------------------
// dedupStrings
// ------------------------------------------------------------

func TestDedupStrings(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{"no dupes", []string{"a", "b", "c"}, []string{"a", "b", "c"}},
		{"dupes", []string{"a", "b", "a", "c", "b"}, []string{"a", "b", "c"}},
		{"empty filtered", []string{"", "a", "", "b"}, []string{"a", "b"}},
		{"all empty", []string{"", "", ""}, []string{}},
		{"nil", nil, []string{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := dedupStrings(c.in)
			if len(got) != len(c.want) {
				t.Errorf("len mismatch got=%v want=%v", got, c.want)
				return
			}
			for i, v := range got {
				if v != c.want[i] {
					t.Errorf("at %d got %q want %q", i, v, c.want[i])
				}
			}
		})
	}
}

// ------------------------------------------------------------
// drainTick 无 FROZEN → 快速 no-op
// ------------------------------------------------------------

func TestDrainTick_NoFrozenEarlyReturn(t *testing.T) {
	_ = setupDrainDB(t)
	listCalled := 0
	drainListFn = func(_ context.Context, _ []string, _ int) ([]instanceToDrain, error) {
		listCalled++
		return nil, nil
	}
	drainTick(context.Background())
	if listCalled != 0 {
		t.Errorf("no FROZEN should skip list: got %d calls", listCalled)
	}
}

// ------------------------------------------------------------
// drainTick 有 FROZEN 但云端无实例 → 不调换绑
// ------------------------------------------------------------

func TestDrainTick_FrozenButNoInstances(t *testing.T) {
	db := setupDrainDB(t)
	rs := seedDrainRuleSet(t, db)
	db.Create(&model.ManagedSGPool{
		SGID: "sg-frozen", RuleSetID: rs.ID, Status: model.SGStatusFrozen,
	})
	drainListFn = func(_ context.Context, _ []string, _ int) ([]instanceToDrain, error) { return nil, nil }

	modifyCalled := 0
	drainModifyFn = func(_ context.Context, _ string, _ []string) error { modifyCalled++; return nil }

	drainTick(context.Background())

	if modifyCalled != 0 {
		t.Errorf("no instances → no modify calls, got %d", modifyCalled)
	}
}

// ------------------------------------------------------------
// drainOneInstance 成功路径
// ------------------------------------------------------------

func TestDrainOneInstance_SuccessRebindsAndUpdatesCounts(t *testing.T) {
	db := setupDrainDB(t)
	rs := seedDrainRuleSet(t, db)
	// 目标 SG（cvm_count=0）和源 FROZEN（cvm_count=3）
	db.Create(&model.ManagedSGPool{
		SGID: "sg-target", RuleSetID: rs.ID, Status: model.SGStatusActive, CVMCount: 0,
	})
	db.Create(&model.ManagedSGPool{
		SGID: "sg-frozen", RuleSetID: rs.ID, Status: model.SGStatusFrozen, CVMCount: 3,
	})
	// 实例本来绑着 sg-frozen
	db.Create(&model.Instance{InstanceId: "ins-001", SecurityGroupId: "sg-frozen"})

	frozenSet := map[string]bool{"sg-frozen": true}
	inst := instanceToDrain{
		ID:               "ins-001",
		SecurityGroupIDs: []string{"sg-frozen"},
	}

	// 捕获 modify 调用参数
	var gotSGs []string
	drainModifyFn = func(_ context.Context, id string, sgs []string) error {
		gotSGs = sgs
		return nil
	}

	drainOneInstance(context.Background(), "acme", inst, frozenSet)

	if len(gotSGs) != 1 || gotSGs[0] != "sg-target" {
		t.Errorf("expected modify with [sg-target], got %v", gotSGs)
	}
	var target, frozen model.ManagedSGPool
	db.Where("sg_id = ?", "sg-target").First(&target)
	db.Where("sg_id = ?", "sg-frozen").First(&frozen)
	if target.CVMCount != 1 {
		t.Errorf("target cvm_count should be 1, got %d", target.CVMCount)
	}
	if frozen.CVMCount != 2 {
		t.Errorf("frozen cvm_count should drop to 2, got %d", frozen.CVMCount)
	}
	var afterInst model.Instance
	db.Where("instance_id = ?", "ins-001").First(&afterInst)
	if afterInst.SecurityGroupId != "sg-target" {
		t.Errorf("instance sg should be sg-target, got %q", afterInst.SecurityGroupId)
	}
}

// ------------------------------------------------------------
// drainOneInstance 修改失败 → fail_count++
// ------------------------------------------------------------

func TestDrainOneInstance_FailCountIncrement(t *testing.T) {
	db := setupDrainDB(t)
	rs := seedDrainRuleSet(t, db)
	db.Create(&model.ManagedSGPool{SGID: "sg-target", RuleSetID: rs.ID, Status: model.SGStatusActive})
	db.Create(&model.ManagedSGPool{SGID: "sg-frozen", RuleSetID: rs.ID, Status: model.SGStatusFrozen})
	db.Create(&model.Instance{InstanceId: "ins-fail", SecurityGroupId: "sg-frozen"})

	drainModifyFn = func(_ context.Context, _ string, _ []string) error { return errors.New("cloud busy") }

	frozenSet := map[string]bool{"sg-frozen": true}
	inst := instanceToDrain{ID: "ins-fail", SecurityGroupIDs: []string{"sg-frozen"}}

	drainOneInstance(context.Background(), "acme", inst, frozenSet)

	state, err := model.GetDrainState(context.Background(), "ins-fail")
	if err != nil {
		t.Fatalf("get drain state: %v", err)
	}
	if state == nil || state.FailCount != 1 {
		t.Errorf("expected fail_count=1, got %+v", state)
	}
}

// ------------------------------------------------------------
// drainOneInstance 已 stuck → 直接跳过
// ------------------------------------------------------------

func TestDrainOneInstance_StuckSkipped(t *testing.T) {
	db := setupDrainDB(t)
	rs := seedDrainRuleSet(t, db)
	db.Create(&model.ManagedSGPool{SGID: "sg-target", RuleSetID: rs.ID, Status: model.SGStatusActive})
	db.Create(&model.ManagedSGPool{SGID: "sg-frozen", RuleSetID: rs.ID, Status: model.SGStatusFrozen})

	// 人为放一个 stuck state
	now := time.Now()
	db.Create(&model.SGDrainState{
		InstanceID: "ins-stuck",
		FailCount:  10,
		StuckAt:    &now,
	})

	modifyCalled := 0
	drainModifyFn = func(_ context.Context, _ string, _ []string) error { modifyCalled++; return nil }

	drainOneInstance(context.Background(), "acme", instanceToDrain{
		ID: "ins-stuck", SecurityGroupIDs: []string{"sg-frozen"},
	}, map[string]bool{"sg-frozen": true})

	if modifyCalled != 0 {
		t.Errorf("stuck instance should skip modify, got %d calls", modifyCalled)
	}
}

// ------------------------------------------------------------
// drainOneInstance 没有 FROZEN 源（云端扫描到但已不绑）→ no-op
// ------------------------------------------------------------

func TestDrainOneInstance_NoFrozenSourceNoOp(t *testing.T) {
	db := setupDrainDB(t)
	rs := seedDrainRuleSet(t, db)
	db.Create(&model.ManagedSGPool{SGID: "sg-target", RuleSetID: rs.ID, Status: model.SGStatusActive})

	modifyCalled := 0
	drainModifyFn = func(_ context.Context, _ string, _ []string) error { modifyCalled++; return nil }

	// 传入的 inst SecurityGroupIDs 全是非 FROZEN
	drainOneInstance(context.Background(), "acme",
		instanceToDrain{ID: "ins-x", SecurityGroupIDs: []string{"sg-other"}},
		map[string]bool{"sg-frozen": true}, // sg-other 不在 frozenSet
	)

	if modifyCalled != 0 {
		t.Errorf("no frozen source → no modify, got %d", modifyCalled)
	}
}

// ------------------------------------------------------------
// drainOneInstance fail_count 达阈值 → 标 stuck
// ------------------------------------------------------------

func TestDrainOneInstance_ReachStuckThreshold(t *testing.T) {
	db := setupDrainDB(t)
	rs := seedDrainRuleSet(t, db)
	db.Create(&model.ManagedSGPool{SGID: "sg-target", RuleSetID: rs.ID, Status: model.SGStatusActive})
	db.Create(&model.ManagedSGPool{SGID: "sg-frozen", RuleSetID: rs.ID, Status: model.SGStatusFrozen})
	db.Create(&model.Instance{InstanceId: "ins-max", SecurityGroupId: "sg-frozen"})

	// 已有 fail_count = threshold - 1；这次失败应到阈值
	db.Create(&model.SGDrainState{
		InstanceID: "ins-max",
		FailCount:  drainMaxFails - 1,
	})
	drainModifyFn = func(_ context.Context, _ string, _ []string) error { return errors.New("api failure") }

	drainOneInstance(context.Background(), "acme",
		instanceToDrain{ID: "ins-max", SecurityGroupIDs: []string{"sg-frozen"}},
		map[string]bool{"sg-frozen": true})

	state, _ := model.GetDrainState(context.Background(), "ins-max")
	if state.StuckAt == nil {
		t.Error("expected stuck_at set after reaching threshold")
	}
}

// ------------------------------------------------------------
// drainTick panic recover
// ------------------------------------------------------------

// TestDrainTick_PanicRecovered 已移除：panic recovery 由 scheduler.executeTask 统一处理，
// 参见 scheduler_test.go 中的相关测试。

// ------------------------------------------------------------
// 并发安全：多个 goroutine 同时 drainTick（smoke test）。
// 在 setupDrainDB 的 t.Cleanup 之前把所有 goroutine join，
// 避免测试结束时底层 model.DB 被替换引发 "no such table" 警告。
// ------------------------------------------------------------

func TestDrainTick_Concurrent_NoRace(t *testing.T) {
	_ = setupDrainDB(t)
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			drainTick(context.Background())
		}()
	}
	wg.Wait()
}

// ------------------------------------------------------------
// filterManagedInstances：只保留 hatchery instances 表里有的 CVM。
// 防止 listInstancesBoundToSGs 误处理用户业务机器（同 SG 配给多机使用时）。
// ------------------------------------------------------------

func cvmInst(id string, sgs ...string) *cvm.Instance {
	sgPtrs := make([]*string, 0, len(sgs))
	for _, s := range sgs {
		v := s
		sgPtrs = append(sgPtrs, &v)
	}
	idCopy := id
	return &cvm.Instance{
		InstanceId:       &idCopy,
		SecurityGroupIds: sgPtrs,
	}
}

func TestFilterManagedInstances_KeepsOnlyManagedCVMs(t *testing.T) {
	db := setupDrainDB(t)
	// hatchery DB 里只有 ins-managed 这一台
	db.Create(&model.Instance{InstanceId: "ins-managed", SecurityGroupId: "sg-frozen"})

	cloud := []*cvm.Instance{
		cvmInst("ins-managed", "sg-frozen", "sg-other"),
		cvmInst("ins-customer-biz", "sg-frozen"), // 用户自己的业务机器
		nil,                                       // nil 也要安全跳过
		{InstanceId: nil},                         // InstanceId 为 nil 也跳过
	}

	got, err := filterManagedInstances(context.Background(), cloud)
	if err != nil {
		t.Fatalf("filterManagedInstances err: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected exactly 1 managed instance, got %d: %+v", len(got), got)
	}
	if got[0].ID != "ins-managed" {
		t.Errorf("expected ins-managed, got %s", got[0].ID)
	}
	if len(got[0].SecurityGroupIDs) != 2 {
		t.Errorf("expected 2 SGs, got %v", got[0].SecurityGroupIDs)
	}
}

func TestFilterManagedInstances_EmptyCloudInput(t *testing.T) {
	_ = setupDrainDB(t)
	got, err := filterManagedInstances(context.Background(), nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty result, got %v", got)
	}
}

func TestFilterManagedInstances_NoneManaged(t *testing.T) {
	_ = setupDrainDB(t)
	// hatchery DB 里没有任何这些 ID
	cloud := []*cvm.Instance{
		cvmInst("ins-foreign-1", "sg-frozen"),
		cvmInst("ins-foreign-2", "sg-frozen"),
	}
	got, err := filterManagedInstances(context.Background(), cloud)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 managed (all foreign), got %d", len(got))
	}
}
