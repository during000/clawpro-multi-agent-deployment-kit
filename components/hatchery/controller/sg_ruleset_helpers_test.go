package controller

import (
	"context"
	"errors"
	"strings"
	"testing"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// sg_ruleset_helpers_test.go —— migrate-port-open-to-ruleset 的 helper 单测。
//
// 覆盖 tasks.md 1.10 列出的 10 个场景：
//   1. listActiveSGIDsByRuleSet 按 RuleSet 精确枚举
//   2. listAllActiveSGIDs 跨 RuleSet 并集
//   3. resolveRuleSetIDForInstance 孤儿实例报错
//   4. checkPortRuleOnInstanceSG 全新租户返回 ErrSGBootstrapNotDone
//   5. checkPortRuleOnInstanceSG 孤儿实例（有 RuleSet 但 SG 不在池）明确报错
//   6. checkPortRuleOnInstanceSG 同步态走 DB 快路径
//   7. ensureRuleInRuleSet 已存在等价规则幂等返回（不 bump version）
//   8. ensureRuleInRuleSet 空 ACTIVE 池走 UpdateRuleSetRulesInternal 直接 commit
//   9. ensureRuleInAllRuleSets 遍历多 RuleSet
//  10. ensureRuleInAllRuleSets 无 RuleSet 时明确报错
//
// 说明：drift 态走云 API 的路径需要 mock 云 SDK，不在本文件覆盖；由 HTTP 集成测试处理。

// ============================================================================
// 1. listActiveSGIDsByRuleSet / listAllActiveSGIDs
// ============================================================================

func TestListActiveSGIDsByRuleSet_FiltersByRuleSetAndStatus(t *testing.T) {
	db := setupSGPoolTestDB(t)

	// RuleSet A：2 个 ACTIVE，1 个 FROZEN
	rsA := model.RuleSet{Name: "rs-a", Rules: "[]", Version: 1, IsDefault: true}
	db.Create(&rsA)
	db.Create(&model.ManagedSGPool{SGID: "sg-a1", RuleSetID: rsA.ID, Status: model.SGStatusActive, RuleVersion: 1})
	db.Create(&model.ManagedSGPool{SGID: "sg-a2", RuleSetID: rsA.ID, Status: model.SGStatusActive, RuleVersion: 1})
	db.Create(&model.ManagedSGPool{SGID: "sg-afrozen", RuleSetID: rsA.ID, Status: model.SGStatusFrozen, RuleVersion: 1})

	// RuleSet B：1 个 ACTIVE（不应出现在 A 的查询结果里）
	rsB := model.RuleSet{Name: "rs-b", Rules: "[]", Version: 1}
	db.Create(&rsB)
	db.Create(&model.ManagedSGPool{SGID: "sg-b1", RuleSetID: rsB.ID, Status: model.SGStatusActive, RuleVersion: 1})

	got, err := listActiveSGIDsByRuleSet(context.Background(), rsA.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 SGs, got %d: %v", len(got), got)
	}
	if got[0] != "sg-a1" || got[1] != "sg-a2" {
		t.Errorf("expected [sg-a1 sg-a2] in id order, got %v", got)
	}
}

func TestListActiveSGIDsByRuleSet_EmptyReturnsNonNilSlice(t *testing.T) {
	db := setupSGPoolTestDB(t)
	rs := model.RuleSet{Name: "rs-empty", Rules: "[]", Version: 1, IsDefault: true}
	db.Create(&rs)

	got, err := listActiveSGIDsByRuleSet(context.Background(), rs.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil empty slice, got nil")
	}
	if len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
}

func TestListAllActiveSGIDs_UnionAcrossRuleSets(t *testing.T) {
	db := setupSGPoolTestDB(t)

	rsA := model.RuleSet{Name: "rs-a", Rules: "[]", Version: 1, IsDefault: true}
	db.Create(&rsA)
	db.Create(&model.ManagedSGPool{SGID: "sg-a1", RuleSetID: rsA.ID, Status: model.SGStatusActive, RuleVersion: 1})
	db.Create(&model.ManagedSGPool{SGID: "sg-a2", RuleSetID: rsA.ID, Status: model.SGStatusActive, RuleVersion: 1})

	rsB := model.RuleSet{Name: "rs-b", Rules: "[]", Version: 1}
	db.Create(&rsB)
	db.Create(&model.ManagedSGPool{SGID: "sg-b1", RuleSetID: rsB.ID, Status: model.SGStatusActive, RuleVersion: 1})
	db.Create(&model.ManagedSGPool{SGID: "sg-bfrozen", RuleSetID: rsB.ID, Status: model.SGStatusFrozen, RuleVersion: 1})

	got, err := listAllActiveSGIDs(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 SGs across rulesets, got %d: %v", len(got), got)
	}
	// FROZEN 的应该被过滤掉
	for _, id := range got {
		if strings.Contains(id, "frozen") {
			t.Errorf("FROZEN SG %s must be excluded", id)
		}
	}
}

// ============================================================================
// 2. resolveRuleSetIDForInstance
// ============================================================================

func TestResolveRuleSetIDForInstance_Found(t *testing.T) {
	db := setupSGPoolTestDB(t)
	rs := model.RuleSet{Name: "rs-x", Rules: "[]", Version: 1, IsDefault: true}
	db.Create(&rs)
	db.Create(&model.ManagedSGPool{SGID: "sg-x1", RuleSetID: rs.ID, Status: model.SGStatusActive, RuleVersion: 1})

	inst := &model.Instance{InstanceId: "ins-1", SecurityGroupId: "sg-x1"}
	got, err := resolveRuleSetIDForInstance(context.Background(), inst)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != rs.ID {
		t.Errorf("expected rsID=%d, got %d", rs.ID, got)
	}
}

func TestResolveRuleSetIDForInstance_OrphanSG(t *testing.T) {
	_ = setupSGPoolTestDB(t)
	inst := &model.Instance{InstanceId: "ins-orphan", SecurityGroupId: "sg-notinpool"}
	_, err := resolveRuleSetIDForInstance(context.Background(), inst)
	if err == nil {
		t.Fatal("expected error for orphan SG, got nil")
	}
	if !errors.Is(err, hcommon.I18nError(i18n.MsgSGRulesetSGNotInPool)) {
		t.Errorf("expected 'not in any managed pool' error, got: %v", err)
	}
}

func TestResolveRuleSetIDForInstance_EmptySG(t *testing.T) {
	_ = setupSGPoolTestDB(t)
	inst := &model.Instance{InstanceId: "ins-no-sg", SecurityGroupId: ""}
	_, err := resolveRuleSetIDForInstance(context.Background(), inst)
	if err == nil {
		t.Fatal("expected error for instance without SG, got nil")
	}
	if !errors.Is(err, hcommon.I18nError(i18n.MsgSGRulesetInstanceNoSG)) {
		t.Errorf("expected 'has no SecurityGroupId' error, got: %v", err)
	}
}

// ============================================================================
// 3. checkPortRuleOnInstanceSG
// ============================================================================

func TestCheckPortRuleOnInstanceSG_FreshTenantReturnsSentinel(t *testing.T) {
	_ = setupSGPoolTestDB(t)
	// 没建任何 RuleSet → 全新租户场景
	inst := &model.Instance{InstanceId: "ins-new", SecurityGroupId: "sg-any"}
	_, _, err := checkPortRuleOnInstanceSG(context.Background(), inst, 7540, "TCP")
	if !errors.Is(err, ErrSGBootstrapNotDone) {
		t.Errorf("expected ErrSGBootstrapNotDone, got: %v", err)
	}
}

func TestCheckPortRuleOnInstanceSG_OrphanInstance(t *testing.T) {
	db := setupSGPoolTestDB(t)
	// 租户已有 RuleSet，但实例绑的 SG 不在任何池里
	rs := model.RuleSet{Name: "rs-default", Rules: "[]", Version: 1, IsDefault: true}
	db.Create(&rs)

	inst := &model.Instance{InstanceId: "ins-orphan", SecurityGroupId: "sg-ghost"}
	_, _, err := checkPortRuleOnInstanceSG(context.Background(), inst, 7540, "TCP")
	if err == nil {
		t.Fatal("expected error for orphan SG")
	}
	if !errors.Is(err, hcommon.I18nError(i18n.MsgSGRulesetSGNotInPool)) {
		t.Errorf("expected 'not in any managed pool', got: %v", err)
	}
}

func TestCheckPortRuleOnInstanceSG_SyncedFastPathAllowed(t *testing.T) {
	db := setupSGPoolTestDB(t)
	rs := model.RuleSet{
		Name: "rs-default", IsDefault: true, Version: 5,
		// Rules 里有一条放通 7540 的规则
		Rules: `[{"direction":"INGRESS","protocol":"TCP","port":"7540","cidr_block":"0.0.0.0/0","action":"ACCEPT"}]`,
	}
	db.Create(&rs)
	db.Create(&model.ManagedSGPool{
		SGID: "sg-synced", RuleSetID: rs.ID, Status: model.SGStatusActive,
		RuleVersion: 5, // == rs.Version → 同步态
	})

	inst := &model.Instance{InstanceId: "ins-synced", SecurityGroupId: "sg-synced"}
	allowed, drifting, err := checkPortRuleOnInstanceSG(context.Background(), inst, 7540, "TCP")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Error("expected allowed=true, got false")
	}
	if drifting {
		t.Error("expected drifting=false in synced state")
	}
}

func TestCheckPortRuleOnInstanceSG_SyncedFastPathDenied(t *testing.T) {
	db := setupSGPoolTestDB(t)
	rs := model.RuleSet{
		Name: "rs-default", IsDefault: true, Version: 3,
		// Rules 里没有放通 7540 的规则
		Rules: `[{"direction":"INGRESS","protocol":"TCP","port":"22","cidr_block":"0.0.0.0/0","action":"ACCEPT"}]`,
	}
	db.Create(&rs)
	db.Create(&model.ManagedSGPool{
		SGID: "sg-synced", RuleSetID: rs.ID, Status: model.SGStatusActive,
		RuleVersion: 3,
	})

	inst := &model.Instance{InstanceId: "ins-synced", SecurityGroupId: "sg-synced"}
	allowed, _, err := checkPortRuleOnInstanceSG(context.Background(), inst, 7540, "TCP")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Error("expected allowed=false for uncovered port, got true")
	}
}

// ============================================================================
// 4. portCoveredByRules（纯函数，直接验语义）
// ============================================================================

func TestPortCoveredByRules(t *testing.T) {
	tests := []struct {
		name   string
		rules  []Rule
		port   int
		proto  string
		expect bool
	}{
		{
			name: "exact match ACCEPT",
			rules: []Rule{
				{Direction: "INGRESS", Protocol: "TCP", Port: "7540", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"},
			},
			port: 7540, proto: "TCP", expect: true,
		},
		{
			name: "port range covers target",
			rules: []Rule{
				{Direction: "INGRESS", Protocol: "TCP", Port: "7000-8000", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"},
			},
			port: 7540, proto: "TCP", expect: true,
		},
		{
			name: "ALL protocol matches",
			rules: []Rule{
				{Direction: "INGRESS", Protocol: "ALL", Port: "ALL", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"},
			},
			port: 7540, proto: "TCP", expect: true,
		},
		{
			name: "DROP before ACCEPT wins",
			rules: []Rule{
				{Direction: "INGRESS", Protocol: "TCP", Port: "7540", CidrBlock: "0.0.0.0/0", Action: "DROP"},
				{Direction: "INGRESS", Protocol: "TCP", Port: "7540", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"},
			},
			port: 7540, proto: "TCP", expect: false,
		},
		{
			name: "EGRESS rule is ignored",
			rules: []Rule{
				{Direction: "EGRESS", Protocol: "TCP", Port: "7540", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"},
			},
			port: 7540, proto: "TCP", expect: false,
		},
		{
			name: "non-0.0.0.0/0 CIDR is treated as not publicly open",
			rules: []Rule{
				{Direction: "INGRESS", Protocol: "TCP", Port: "7540", CidrBlock: "10.0.0.0/8", Action: "ACCEPT"},
			},
			port: 7540, proto: "TCP", expect: false,
		},
		{
			name:  "empty rules",
			rules: nil,
			port:  7540, proto: "TCP", expect: false,
		},
		{
			name: "wrong protocol",
			rules: []Rule{
				{Direction: "INGRESS", Protocol: "UDP", Port: "7540", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"},
			},
			port: 7540, proto: "TCP", expect: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := portCoveredByRules(tt.rules, tt.port, tt.proto)
			if got != tt.expect {
				t.Errorf("portCoveredByRules(%+v, %d, %s) = %v, want %v",
					tt.rules, tt.port, tt.proto, got, tt.expect)
			}
		})
	}
}

// ============================================================================
// 5. ensureRuleInRuleSet 幂等性
// ============================================================================

func TestEnsureRuleInRuleSet_IdempotentByCoverage(t *testing.T) {
	db := setupSGPoolTestDB(t)
	rs := model.RuleSet{
		Name: model.DefaultRuleSetName, IsDefault: true, Version: 5,
		Rules: `[{"direction":"INGRESS","protocol":"TCP","port":"7540","cidr_block":"0.0.0.0/0","action":"ACCEPT"}]`,
	}
	db.Create(&rs)

	// 追加同一条规则：应因覆盖语义而幂等跳过
	newRule := Rule{
		Direction: "INGRESS", Protocol: "TCP", Port: "7540",
		CidrBlock: "0.0.0.0/0", Action: "ACCEPT",
	}
	if err := ensureRuleInRuleSet(context.Background(), rs.ID, newRule); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 校验 version 没 bump
	var reloaded model.RuleSet
	db.First(&reloaded, rs.ID)
	if reloaded.Version != 5 {
		t.Errorf("version should stay 5 (idempotent), got %d", reloaded.Version)
	}
}

func TestEnsureRuleInRuleSet_IdempotentByFingerprint(t *testing.T) {
	db := setupSGPoolTestDB(t)
	// 规则是非 INGRESS ACCEPT（比如 EGRESS），不走 coverage 路径，走指纹去重
	rs := model.RuleSet{
		Name: model.DefaultRuleSetName, IsDefault: true, Version: 3,
		Rules: `[{"direction":"EGRESS","protocol":"TCP","port":"443","cidr_block":"0.0.0.0/0","action":"ACCEPT"}]`,
	}
	db.Create(&rs)

	sameRule := Rule{
		Direction: "EGRESS", Protocol: "TCP", Port: "443",
		CidrBlock: "0.0.0.0/0", Action: "ACCEPT",
	}
	if err := ensureRuleInRuleSet(context.Background(), rs.ID, sameRule); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var reloaded model.RuleSet
	db.First(&reloaded, rs.ID)
	if reloaded.Version != 3 {
		t.Errorf("version should stay 3 (fingerprint dedup), got %d", reloaded.Version)
	}
}

func TestEnsureRuleInRuleSet_AppendsNewRule_NoActiveSGs(t *testing.T) {
	db := setupSGPoolTestDB(t)
	// 无 ACTIVE SG → UpdateRuleSetRulesInternal 会直接 DB commit（无需调云 API）
	rs := model.RuleSet{
		Name: model.DefaultRuleSetName, IsDefault: true, Version: 1,
		Rules: `[]`,
	}
	db.Create(&rs)

	newRule := Rule{
		Direction: "INGRESS", Protocol: "TCP", Port: "7540",
		CidrBlock: "0.0.0.0/0", Action: "ACCEPT",
	}
	if err := ensureRuleInRuleSet(context.Background(), rs.ID, newRule); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var reloaded model.RuleSet
	db.First(&reloaded, rs.ID)
	if reloaded.Version != 2 {
		t.Errorf("version should bump to 2, got %d", reloaded.Version)
	}
	if !strings.Contains(reloaded.Rules, `"port":"7540"`) {
		t.Errorf("new rule not persisted; rules=%s", reloaded.Rules)
	}
}

// ============================================================================
// 6. ensureRuleInAllRuleSets
// ============================================================================

func TestEnsureRuleInAllRuleSets_TraversesAllRuleSets(t *testing.T) {
	db := setupSGPoolTestDB(t)
	rsA := model.RuleSet{Name: "rs-a", IsDefault: true, Version: 1, Rules: "[]"}
	db.Create(&rsA)
	rsB := model.RuleSet{Name: "rs-b", Version: 1, Rules: "[]"}
	db.Create(&rsB)
	// 均无 ACTIVE SG → 走 DB commit 路径

	newRule := Rule{
		Direction: "INGRESS", Protocol: "TCP", Port: "9999",
		CidrBlock: "0.0.0.0/0", Action: "ACCEPT",
	}
	if err := ensureRuleInAllRuleSets(context.Background(), newRule); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var reloadedA, reloadedB model.RuleSet
	db.First(&reloadedA, rsA.ID)
	db.First(&reloadedB, rsB.ID)
	if reloadedA.Version != 2 {
		t.Errorf("rs-a version should bump to 2, got %d", reloadedA.Version)
	}
	if reloadedB.Version != 2 {
		t.Errorf("rs-b version should bump to 2, got %d", reloadedB.Version)
	}
	if !strings.Contains(reloadedA.Rules, `"port":"9999"`) {
		t.Errorf("rule not persisted in rs-a; rules=%s", reloadedA.Rules)
	}
	if !strings.Contains(reloadedB.Rules, `"port":"9999"`) {
		t.Errorf("rule not persisted in rs-b; rules=%s", reloadedB.Rules)
	}
}

func TestEnsureRuleInAllRuleSets_NoRuleSetReturnsError(t *testing.T) {
	_ = setupSGPoolTestDB(t)
	// 无任何 RuleSet → 立即报错
	newRule := Rule{
		Direction: "INGRESS", Protocol: "TCP", Port: "9999",
		CidrBlock: "0.0.0.0/0", Action: "ACCEPT",
	}
	err := ensureRuleInAllRuleSets(context.Background(), newRule)
	if err == nil {
		t.Fatal("expected error when no rule_set exists")
	}
	if err != ErrSGBootstrapNotDone {
		t.Errorf("expected 'no rule_set found' error, got: %v", err)
	}
}

// ============================================================================
// 7. portCoveredByRulesAnyCIDR（纯函数，验证不检查 CidrBlock 的语义）
// ============================================================================

func TestPortCoveredByRulesAnyCIDR(t *testing.T) {
	tests := []struct {
		name   string
		rules  []Rule
		port   int
		proto  string
		expect bool
	}{
		{
			name: "白名单 IP /32 规则视为放通",
			rules: []Rule{
				{Direction: "INGRESS", Protocol: "TCP", Port: "6080", CidrBlock: "118.195.242.145/32", Action: "ACCEPT"},
			},
			port: 6080, proto: "TCP", expect: true,
		},
		{
			name: "0.0.0.0/0 全公网规则也视为放通",
			rules: []Rule{
				{Direction: "INGRESS", Protocol: "TCP", Port: "6080", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"},
			},
			port: 6080, proto: "TCP", expect: true,
		},
		{
			name: "内网 CIDR 规则也视为放通（不限制来源）",
			rules: []Rule{
				{Direction: "INGRESS", Protocol: "TCP", Port: "6080", CidrBlock: "10.0.0.0/8", Action: "ACCEPT"},
			},
			port: 6080, proto: "TCP", expect: true,
		},
		{
			name: "DROP 规则优先于后续 ACCEPT",
			rules: []Rule{
				{Direction: "INGRESS", Protocol: "TCP", Port: "6080", CidrBlock: "118.195.242.145/32", Action: "DROP"},
				{Direction: "INGRESS", Protocol: "TCP", Port: "6080", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"},
			},
			port: 6080, proto: "TCP", expect: false,
		},
		{
			name: "EGRESS 规则被忽略",
			rules: []Rule{
				{Direction: "EGRESS", Protocol: "TCP", Port: "6080", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"},
			},
			port: 6080, proto: "TCP", expect: false,
		},
		{
			name: "端口不匹配",
			rules: []Rule{
				{Direction: "INGRESS", Protocol: "TCP", Port: "22", CidrBlock: "118.195.242.145/32", Action: "ACCEPT"},
			},
			port: 6080, proto: "TCP", expect: false,
		},
		{
			name: "协议不匹配",
			rules: []Rule{
				{Direction: "INGRESS", Protocol: "UDP", Port: "6080", CidrBlock: "118.195.242.145/32", Action: "ACCEPT"},
			},
			port: 6080, proto: "TCP", expect: false,
		},
		{
			name: "ALL 协议匹配",
			rules: []Rule{
				{Direction: "INGRESS", Protocol: "ALL", Port: "ALL", CidrBlock: "118.195.242.145/32", Action: "ACCEPT"},
			},
			port: 6080, proto: "TCP", expect: true,
		},
		{
			name: "端口范围覆盖",
			rules: []Rule{
				{Direction: "INGRESS", Protocol: "TCP", Port: "6000-7000", CidrBlock: "129.204.9.86/32", Action: "ACCEPT"},
			},
			port: 6080, proto: "TCP", expect: true,
		},
		{
			name:  "空规则列表",
			rules: nil,
			port:  6080, proto: "TCP", expect: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := portCoveredByRulesAnyCIDR(tt.rules, tt.port, tt.proto)
			if got != tt.expect {
				t.Errorf("portCoveredByRulesAnyCIDR(%+v, %d, %s) = %v, want %v",
					tt.rules, tt.port, tt.proto, got, tt.expect)
			}
		})
	}
}

// ============================================================================
// 8. checkPortRuleOnInstanceSG 的 anyCIDR 可选参数测试
// ============================================================================

func TestCheckPortRuleOnInstanceSG_SyncedAnyCIDR_WhitelistIPAllowed(t *testing.T) {
	db := setupSGPoolTestDB(t)
	// 规则里只有白名单 IP（/32），不是 0.0.0.0/0
	rs := model.RuleSet{
		Name: "rs-default", IsDefault: true, Version: 5,
		Rules: `[{"direction":"INGRESS","protocol":"TCP","port":"6080","cidr_block":"118.195.242.145/32","action":"ACCEPT"}]`,
	}
	db.Create(&rs)
	db.Create(&model.ManagedSGPool{
		SGID: "sg-vnc", RuleSetID: rs.ID, Status: model.SGStatusActive,
		RuleVersion: 5, // 同步态
	})

	inst := &model.Instance{InstanceId: "ins-vnc", SecurityGroupId: "sg-vnc"}

	// 不传 anyCIDR（默认 false）：白名单 IP 不算放通
	allowed, drifting, err := checkPortRuleOnInstanceSG(context.Background(), inst, 6080, "TCP")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Error("without anyCIDR, whitelist IP should NOT be treated as allowed")
	}
	if drifting {
		t.Error("expected drifting=false in synced state")
	}

	// 传 anyCIDR=true：白名单 IP 视为放通
	allowed, drifting, err = checkPortRuleOnInstanceSG(context.Background(), inst, 6080, "TCP", portRuleCheckOptions{anyCIDR: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Error("with anyCIDR=true, whitelist IP should be treated as allowed")
	}
	if drifting {
		t.Error("expected drifting=false in synced state")
	}
}

func TestCheckPortRuleOnInstanceSG_SyncedAnyCIDR_PortNotCovered(t *testing.T) {
	db := setupSGPoolTestDB(t)
	// 规则里放通的是 22 端口，不是 6080
	rs := model.RuleSet{
		Name: "rs-default", IsDefault: true, Version: 3,
		Rules: `[{"direction":"INGRESS","protocol":"TCP","port":"22","cidr_block":"118.195.242.145/32","action":"ACCEPT"}]`,
	}
	db.Create(&rs)
	db.Create(&model.ManagedSGPool{
		SGID: "sg-vnc2", RuleSetID: rs.ID, Status: model.SGStatusActive,
		RuleVersion: 3,
	})

	inst := &model.Instance{InstanceId: "ins-vnc2", SecurityGroupId: "sg-vnc2"}

	// 即使 anyCIDR=true，端口不匹配也不算放通
	allowed, _, err := checkPortRuleOnInstanceSG(context.Background(), inst, 6080, "TCP", portRuleCheckOptions{anyCIDR: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Error("with anyCIDR=true but wrong port, should NOT be allowed")
	}
}

func TestCheckPortRuleOnInstanceSG_SyncedAnyCIDR_DropRuleWins(t *testing.T) {
	db := setupSGPoolTestDB(t)
	// DROP 规则在 ACCEPT 之前
	rs := model.RuleSet{
		Name: "rs-default", IsDefault: true, Version: 4,
		Rules: `[{"direction":"INGRESS","protocol":"TCP","port":"6080","cidr_block":"118.195.242.145/32","action":"DROP"},{"direction":"INGRESS","protocol":"TCP","port":"6080","cidr_block":"0.0.0.0/0","action":"ACCEPT"}]`,
	}
	db.Create(&rs)
	db.Create(&model.ManagedSGPool{
		SGID: "sg-vnc3", RuleSetID: rs.ID, Status: model.SGStatusActive,
		RuleVersion: 4,
	})

	inst := &model.Instance{InstanceId: "ins-vnc3", SecurityGroupId: "sg-vnc3"}

	// anyCIDR=true 但 DROP 优先
	allowed, _, err := checkPortRuleOnInstanceSG(context.Background(), inst, 6080, "TCP", portRuleCheckOptions{anyCIDR: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Error("DROP rule should take priority even with anyCIDR=true")
	}
}

func TestCheckPortRuleOnInstanceSG_GatewayStrictMode_WhitelistIPDenied(t *testing.T) {
	db := setupSGPoolTestDB(t)
	// Gateway 场景：白名单 IP 不算放通（需要 0.0.0.0/0）
	rs := model.RuleSet{
		Name: "rs-default", IsDefault: true, Version: 2,
		Rules: `[{"direction":"INGRESS","protocol":"TCP","port":"7540","cidr_block":"118.195.242.145/32","action":"ACCEPT"}]`,
	}
	db.Create(&rs)
	db.Create(&model.ManagedSGPool{
		SGID: "sg-gw", RuleSetID: rs.ID, Status: model.SGStatusActive,
		RuleVersion: 2,
	})

	inst := &model.Instance{InstanceId: "ins-gw", SecurityGroupId: "sg-gw"}

	// 不传 anyCIDR（Gateway 场景）：白名单 IP 不算放通
	allowed, _, err := checkPortRuleOnInstanceSG(context.Background(), inst, 7540, "TCP")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Error("Gateway strict mode: whitelist IP should NOT be treated as allowed")
	}
}

func TestCheckPortRuleOnInstanceSG_GatewayStrictMode_FullPublicAllowed(t *testing.T) {
	db := setupSGPoolTestDB(t)
	// Gateway 场景：0.0.0.0/0 才算放通
	rs := model.RuleSet{
		Name: "rs-default", IsDefault: true, Version: 2,
		Rules: `[{"direction":"INGRESS","protocol":"TCP","port":"7540","cidr_block":"0.0.0.0/0","action":"ACCEPT"}]`,
	}
	db.Create(&rs)
	db.Create(&model.ManagedSGPool{
		SGID: "sg-gw2", RuleSetID: rs.ID, Status: model.SGStatusActive,
		RuleVersion: 2,
	})

	inst := &model.Instance{InstanceId: "ins-gw2", SecurityGroupId: "sg-gw2"}

	// 不传 anyCIDR（Gateway 场景）：0.0.0.0/0 算放通
	allowed, _, err := checkPortRuleOnInstanceSG(context.Background(), inst, 7540, "TCP")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Error("Gateway strict mode: 0.0.0.0/0 should be treated as allowed")
	}
}
