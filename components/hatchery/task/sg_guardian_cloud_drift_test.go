package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"hatchery/controller"
	"hatchery/model"
)

// ============================================================================
// guardianDetectCloudRuleDrift 单元测试
//
// 场景覆盖：
//   1. 云端规则与 RuleSet 一致 → 不打 drift
//   2. 云端规则被人改了（少一条/多一条）→ MarkSGDrift
//   3. SG 在云端不存在（errSGGone）→ 跳过，不打 drift（由 detectOrphans 处理）
//   4. drift_at IS NOT NULL（已经待处理）→ 跳过，不重复调云 API
//   5. status != ACTIVE → 完全不扫
//   6. describe 失败（非 SG-gone）→ 跳过，不打 drift（下一轮重试）
// ============================================================================

// stubDescribeCloudRules 装饰 guardianDescribeCloudRulesFn，按 sgID 返回预设结果。
// teardown 闭包负责还原原值。
func stubDescribeCloudRules(t *testing.T, byID map[string]struct {
	rules []controller.Rule
	err   error
}) {
	t.Helper()
	orig := guardianDescribeCloudRulesFn
	guardianDescribeCloudRulesFn = func(ctx context.Context, sgID string) ([]controller.Rule, error) {
		v, ok := byID[sgID]
		if !ok {
			return nil, fmt.Errorf("unexpected sgID in stub: %s", sgID)
		}
		return v.rules, v.err
	}
	t.Cleanup(func() { guardianDescribeCloudRulesFn = orig })
}

func ruleAccept22() controller.Rule {
	return controller.Rule{
		Direction: "INGRESS", Protocol: "TCP", Port: "22",
		CidrBlock: "0.0.0.0/0", Action: "ACCEPT",
	}
}

func ruleAccept80() controller.Rule {
	return controller.Rule{
		Direction: "INGRESS", Protocol: "TCP", Port: "80",
		CidrBlock: "0.0.0.0/0", Action: "ACCEPT",
	}
}

// seedSGWithRules 写一条 ACTIVE 的 SG 记录到 DB，driftAt 默认 nil。
func seedSGWithRules(t *testing.T, sgID string, ruleVersion int) *model.ManagedSGPool {
	t.Helper()
	sg := &model.ManagedSGPool{
		SGID:        sgID,
		Status:      model.SGStatusActive,
		RuleVersion: ruleVersion,
	}
	if err := model.DB(context.Background()).Create(sg).Error; err != nil {
		t.Fatalf("seed sg %s: %v", sgID, err)
	}
	return sg
}

// setRuleSetRules 把 default RuleSet 的 Rules 字段设置成给定规则的 JSON。
func setRuleSetRules(t *testing.T, rs *model.RuleSet, rules []controller.Rule) {
	t.Helper()
	b, err := json.Marshal(rules)
	if err != nil {
		t.Fatalf("marshal rules: %v", err)
	}
	rs.Rules = string(b)
	if err := model.DB(context.Background()).Save(rs).Error; err != nil {
		t.Fatalf("save rule set: %v", err)
	}
}

// 1. 云端与 RuleSet 一致 → 不标 drift
func TestGuardianDetectCloudRuleDrift_InSync_NoMark(t *testing.T) {
	db := setupGuardianDB(t)
	rs := seedGuardianRuleSet(t, db)
	expected := []controller.Rule{ruleAccept22(), ruleAccept80()}
	setRuleSetRules(t, rs, expected)
	sg := seedSGWithRules(t, "sg-aaa", 2)

	stubDescribeCloudRules(t, map[string]struct {
		rules []controller.Rule
		err   error
	}{
		"sg-aaa": {rules: expected, err: nil},
	})

	guardianDetectCloudRuleDrift(context.Background())

	var got model.ManagedSGPool
	if err := db.First(&got, sg.ID).Error; err != nil {
		t.Fatalf("reload sg: %v", err)
	}
	if got.DriftAt != nil {
		t.Fatalf("expected drift_at NIL, got %v", got.DriftAt)
	}
}

// 2a. 云端少一条规则（被人删了）→ 标 drift
func TestGuardianDetectCloudRuleDrift_CloudMissingRule_Marks(t *testing.T) {
	db := setupGuardianDB(t)
	rs := seedGuardianRuleSet(t, db)
	expected := []controller.Rule{ruleAccept22(), ruleAccept80()}
	setRuleSetRules(t, rs, expected)
	sg := seedSGWithRules(t, "sg-bbb", 2)

	// 云端只有 22 端口；80 被人删了
	stubDescribeCloudRules(t, map[string]struct {
		rules []controller.Rule
		err   error
	}{
		"sg-bbb": {rules: []controller.Rule{ruleAccept22()}, err: nil},
	})

	guardianDetectCloudRuleDrift(context.Background())

	var got model.ManagedSGPool
	if err := db.First(&got, sg.ID).Error; err != nil {
		t.Fatalf("reload sg: %v", err)
	}
	if got.DriftAt == nil {
		t.Fatalf("expected drift_at set, got nil")
	}
}

// 2b. 云端多一条规则（被人加了）→ 标 drift
func TestGuardianDetectCloudRuleDrift_CloudExtraRule_Marks(t *testing.T) {
	db := setupGuardianDB(t)
	rs := seedGuardianRuleSet(t, db)
	expected := []controller.Rule{ruleAccept22()}
	setRuleSetRules(t, rs, expected)
	sg := seedSGWithRules(t, "sg-ccc", 2)

	// 云端被人多加了 80 端口
	stubDescribeCloudRules(t, map[string]struct {
		rules []controller.Rule
		err   error
	}{
		"sg-ccc": {rules: []controller.Rule{ruleAccept22(), ruleAccept80()}, err: nil},
	})

	guardianDetectCloudRuleDrift(context.Background())

	var got model.ManagedSGPool
	if err := db.First(&got, sg.ID).Error; err != nil {
		t.Fatalf("reload sg: %v", err)
	}
	if got.DriftAt == nil {
		t.Fatalf("expected drift_at set when cloud has extra rule, got nil")
	}
}

// 3. SG 在云端不存在 → 跳过，不打 drift（detectOrphans 接管）
func TestGuardianDetectCloudRuleDrift_SGGone_Skips(t *testing.T) {
	db := setupGuardianDB(t)
	rs := seedGuardianRuleSet(t, db)
	setRuleSetRules(t, rs, []controller.Rule{ruleAccept22()})
	sg := seedSGWithRules(t, "sg-gone", 2)

	stubDescribeCloudRules(t, map[string]struct {
		rules []controller.Rule
		err   error
	}{
		"sg-gone": {rules: nil, err: fmt.Errorf("wrap: %w", errSGGone)},
	})

	guardianDetectCloudRuleDrift(context.Background())

	var got model.ManagedSGPool
	if err := db.First(&got, sg.ID).Error; err != nil {
		t.Fatalf("reload sg: %v", err)
	}
	if got.DriftAt != nil {
		t.Fatalf("expected drift_at NIL for sg-gone, got %v", got.DriftAt)
	}
}

// 4. drift_at 已置位的 SG 跳过（avoid 重复云 API 调用）
func TestGuardianDetectCloudRuleDrift_AlreadyDrifted_Skips(t *testing.T) {
	db := setupGuardianDB(t)
	rs := seedGuardianRuleSet(t, db)
	setRuleSetRules(t, rs, []controller.Rule{ruleAccept22()})
	now := time.Now()
	sg := &model.ManagedSGPool{
		SGID:        "sg-already-drift",
		Status:      model.SGStatusActive,
		RuleVersion: 2,
		DriftAt:     &now,
	}
	if err := db.Create(sg).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	called := false
	orig := guardianDescribeCloudRulesFn
	guardianDescribeCloudRulesFn = func(ctx context.Context, sgID string) ([]controller.Rule, error) {
		called = true
		return nil, nil
	}
	t.Cleanup(func() { guardianDescribeCloudRulesFn = orig })

	guardianDetectCloudRuleDrift(context.Background())

	if called {
		t.Fatalf("expected describe NOT called for already-drifted sg")
	}
}

// 5. 非 ACTIVE 的 SG 不参与扫描
func TestGuardianDetectCloudRuleDrift_NonActiveIgnored(t *testing.T) {
	db := setupGuardianDB(t)
	rs := seedGuardianRuleSet(t, db)
	setRuleSetRules(t, rs, []controller.Rule{ruleAccept22()})

	frozen := &model.ManagedSGPool{
		SGID:        "sg-frozen",
		Status:      model.SGStatusFrozen,
		RuleVersion: 2,
	}
	if err := db.Create(frozen).Error; err != nil {
		t.Fatalf("seed frozen: %v", err)
	}

	called := 0
	orig := guardianDescribeCloudRulesFn
	guardianDescribeCloudRulesFn = func(ctx context.Context, sgID string) ([]controller.Rule, error) {
		called++
		return nil, nil
	}
	t.Cleanup(func() { guardianDescribeCloudRulesFn = orig })

	guardianDetectCloudRuleDrift(context.Background())

	if called != 0 {
		t.Fatalf("expected 0 cloud calls for non-ACTIVE sg, got %d", called)
	}
}

// 6. describe 一般错误（非 SG-gone）→ 跳过该 SG，不打 drift
func TestGuardianDetectCloudRuleDrift_DescribeError_Skips(t *testing.T) {
	db := setupGuardianDB(t)
	rs := seedGuardianRuleSet(t, db)
	setRuleSetRules(t, rs, []controller.Rule{ruleAccept22()})
	sg := seedSGWithRules(t, "sg-err", 2)

	stubDescribeCloudRules(t, map[string]struct {
		rules []controller.Rule
		err   error
	}{
		"sg-err": {rules: nil, err: errors.New("transient cloud 5xx")},
	})

	guardianDetectCloudRuleDrift(context.Background())

	var got model.ManagedSGPool
	if err := db.First(&got, sg.ID).Error; err != nil {
		t.Fatalf("reload sg: %v", err)
	}
	if got.DriftAt != nil {
		t.Fatalf("expected NO drift mark on transient describe error, got %v", got.DriftAt)
	}
}
