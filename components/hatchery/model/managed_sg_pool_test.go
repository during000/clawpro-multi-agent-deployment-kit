package model

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// 本文件覆盖 managed_sg_pool.go 的所有 helper 函数：
//   IncrementSGCVMCount / DecrementSGCVMCount / IsManagedSG
//   ListActiveSGsByRuleSet / ListActiveSGsForFanout / ListFrozenSGs
//   CountActiveSGsInRuleSet / UpdateSGRuleVersion / MarkSGDrift / UpdateSGCVMCount

func setupManagedSGPoolTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&ManagedSGPool{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	orig := UseDBForTest(db)
	return func() { orig() }
}

func seedSG(t *testing.T, sgID string, ruleSetID uint, status string, cvmCount int) *ManagedSGPool {
	t.Helper()
	ctx := context.Background()
	row := &ManagedSGPool{
		SGID:      sgID,
		RuleSetID: ruleSetID,
		Status:    status,
		CVMCount:  cvmCount,
	}
	if err := DB(ctx).Create(row).Error; err != nil {
		t.Fatalf("seed %s: %v", sgID, err)
	}
	return row
}

func getSG(t *testing.T, sgID string) ManagedSGPool {
	t.Helper()
	ctx := context.Background()
	var row ManagedSGPool
	if err := DB(ctx).Where("sg_id = ?", sgID).First(&row).Error; err != nil {
		t.Fatalf("get %s: %v", sgID, err)
	}
	return row
}

func TestManagedSGPoolTableName(t *testing.T) {
	if name := (ManagedSGPool{}).TableName(); name != "managed_sg_pool" {
		t.Errorf("TableName = %q, want managed_sg_pool", name)
	}
}

func TestIncrementSGCVMCount(t *testing.T) {
	defer setupManagedSGPoolTestDB(t)()
	seedSG(t, "sg-1", 1, SGStatusActive, 5)

	if err := IncrementSGCVMCount(context.Background(), "sg-1"); err != nil {
		t.Fatalf("Increment: %v", err)
	}
	if got := getSG(t, "sg-1").CVMCount; got != 6 {
		t.Errorf("CVMCount = %d, want 6", got)
	}

	// 对不存在的 sg_id 不报错但不影响
	if err := IncrementSGCVMCount(context.Background(), "sg-missing"); err != nil {
		t.Errorf("Increment on missing sg: %v", err)
	}
}

func TestDecrementSGCVMCount(t *testing.T) {
	defer setupManagedSGPoolTestDB(t)()
	seedSG(t, "sg-a", 1, SGStatusActive, 3)
	seedSG(t, "sg-zero", 1, SGStatusActive, 0)

	if err := DecrementSGCVMCount(context.Background(), "sg-a"); err != nil {
		t.Fatalf("Decrement: %v", err)
	}
	if got := getSG(t, "sg-a").CVMCount; got != 2 {
		t.Errorf("CVMCount = %d, want 2", got)
	}

	// cvm_count=0 时保护不跌破 0
	if err := DecrementSGCVMCount(context.Background(), "sg-zero"); err != nil {
		t.Fatalf("Decrement zero: %v", err)
	}
	if got := getSG(t, "sg-zero").CVMCount; got != 0 {
		t.Errorf("zero-cvm_count went to %d, want 0", got)
	}
}

func TestIsManagedSG(t *testing.T) {
	defer setupManagedSGPoolTestDB(t)()
	seedSG(t, "sg-managed", 1, SGStatusActive, 0)

	yes, err := IsManagedSG(context.Background(), "sg-managed")
	if err != nil {
		t.Fatalf("IsManagedSG yes: %v", err)
	}
	if !yes {
		t.Error("sg-managed MUST 被识别为 managed")
	}

	no, err := IsManagedSG(context.Background(), "sg-external")
	if err != nil {
		t.Fatalf("IsManagedSG no: %v", err)
	}
	if no {
		t.Error("sg-external MUST NOT 被识别为 managed")
	}
}

func TestListActiveSGsByRuleSet_ExcludesDriftAndOthers(t *testing.T) {
	defer setupManagedSGPoolTestDB(t)()
	ctx := context.Background()
	driftTime := time.Now()
	// rule_set=1 下三个 ACTIVE，一个 FROZEN，一个 drift
	seedSG(t, "sg-a", 1, SGStatusActive, 10)
	seedSG(t, "sg-b", 1, SGStatusActive, 5)
	seedSG(t, "sg-c", 1, SGStatusActive, 20)
	seedSG(t, "sg-frozen", 1, SGStatusFrozen, 0)
	driftRow := seedSG(t, "sg-drift", 1, SGStatusActive, 1)
	driftRow.DriftAt = &driftTime
	if err := DB(ctx).Save(driftRow).Error; err != nil {
		t.Fatalf("save drift: %v", err)
	}
	// rule_set=2 的不应返回
	seedSG(t, "sg-other", 2, SGStatusActive, 0)

	list, err := ListActiveSGsByRuleSet(ctx, 1)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("got %d rows, want 3 (exclude FROZEN + drift + rule_set=2)", len(list))
	}
	// 按 cvm_count ASC 应该是 b/a/c
	if list[0].SGID != "sg-b" || list[1].SGID != "sg-a" || list[2].SGID != "sg-c" {
		t.Errorf("order = %v, want [sg-b, sg-a, sg-c]", []string{list[0].SGID, list[1].SGID, list[2].SGID})
	}
}

func TestListActiveSGsForFanout_IncludesDrift(t *testing.T) {
	defer setupManagedSGPoolTestDB(t)()
	ctx := context.Background()
	driftTime := time.Now()
	seedSG(t, "sg-a", 1, SGStatusActive, 10)
	driftRow := seedSG(t, "sg-drift", 1, SGStatusActive, 1)
	driftRow.DriftAt = &driftTime
	if err := DB(ctx).Save(driftRow).Error; err != nil {
		t.Fatalf("save drift: %v", err)
	}
	seedSG(t, "sg-frozen", 1, SGStatusFrozen, 0)

	list, err := ListActiveSGsForFanout(ctx, 1)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	// 仅含 ACTIVE（含 drift），不含 FROZEN
	if len(list) != 2 {
		t.Fatalf("got %d rows, want 2 (ACTIVE incl. drift, excl. FROZEN)", len(list))
	}
}

func TestListFrozenSGs(t *testing.T) {
	defer setupManagedSGPoolTestDB(t)()
	seedSG(t, "sg-active", 1, SGStatusActive, 5)
	seedSG(t, "sg-frozen1", 1, SGStatusFrozen, 0)
	seedSG(t, "sg-frozen2", 2, SGStatusFrozen, 3)
	seedSG(t, "sg-draining", 1, SGStatusDraining, 0)

	list, err := ListFrozenSGs(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("got %d frozen, want 2", len(list))
	}
}

func TestCountActiveSGsInRuleSet(t *testing.T) {
	defer setupManagedSGPoolTestDB(t)()
	seedSG(t, "sg-a", 1, SGStatusActive, 0)
	seedSG(t, "sg-b", 1, SGStatusActive, 0)
	seedSG(t, "sg-c", 1, SGStatusFrozen, 0) // 不计入 ACTIVE
	seedSG(t, "sg-d", 2, SGStatusActive, 0) // 不同 rule_set

	n, err := CountActiveSGsInRuleSet(context.Background(), 1)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if n != 2 {
		t.Errorf("count = %d, want 2", n)
	}

	n2, err := CountActiveSGsInRuleSet(context.Background(), 999)
	if err != nil {
		t.Fatalf("Count empty: %v", err)
	}
	if n2 != 0 {
		t.Errorf("empty count = %d, want 0", n2)
	}
}

func TestUpdateSGRuleVersion_ClearsDrift(t *testing.T) {
	defer setupManagedSGPoolTestDB(t)()
	ctx := context.Background()
	driftTime := time.Now().Add(-time.Hour)
	row := seedSG(t, "sg-a", 1, SGStatusActive, 5)
	row.DriftAt = &driftTime
	row.RuleVersion = 3
	if err := DB(ctx).Save(row).Error; err != nil {
		t.Fatalf("save: %v", err)
	}

	if err := UpdateSGRuleVersion(ctx, "sg-a", 7); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got := getSG(t, "sg-a")
	if got.RuleVersion != 7 {
		t.Errorf("RuleVersion = %d, want 7", got.RuleVersion)
	}
	if got.DriftAt != nil {
		t.Errorf("DriftAt should be cleared, got %v", got.DriftAt)
	}
}

func TestMarkSGDrift(t *testing.T) {
	defer setupManagedSGPoolTestDB(t)()
	seedSG(t, "sg-a", 1, SGStatusActive, 5)

	before := time.Now().Add(-time.Second)
	if err := MarkSGDrift(context.Background(), "sg-a"); err != nil {
		t.Fatalf("MarkSGDrift: %v", err)
	}
	got := getSG(t, "sg-a")
	if got.DriftAt == nil {
		t.Fatal("DriftAt MUST be set")
	}
	if !got.DriftAt.After(before) {
		t.Errorf("DriftAt %v not after %v", got.DriftAt, before)
	}
}

func TestUpdateSGCVMCount_SetsAbsoluteValue(t *testing.T) {
	defer setupManagedSGPoolTestDB(t)()
	seedSG(t, "sg-a", 1, SGStatusActive, 10)

	if err := UpdateSGCVMCount(context.Background(), "sg-a", 42); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got := getSG(t, "sg-a")
	if got.CVMCount != 42 {
		t.Errorf("CVMCount = %d, want 42", got.CVMCount)
	}
	if got.CVMCountAt == nil {
		t.Error("CVMCountAt MUST be set when Update")
	}
}
