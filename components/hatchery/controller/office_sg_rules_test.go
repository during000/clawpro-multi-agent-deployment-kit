package controller

import (
	"encoding/json"
	"testing"

	"hatchery/model"
)

func TestRefreshOfficeIngressRulesForTenant_AddsRulesAndDedupes(t *testing.T) {
	db := setupSGPoolTestDB(t)
	rs := seedRuleSetAndSGs(t, db, []model.ManagedSGPool{{SGID: "sg-office-active"}})
	seedRules := []Rule{{
		Direction: "ingress",
		Protocol:  "all",
		Port:      "ALL",
		CidrBlock: "219.133.41.27",
		Action:    "accept",
	}}
	raw, err := json.Marshal(seedRules)
	if err != nil {
		t.Fatalf("marshal seed rules: %v", err)
	}
	if err := db.Model(rs).Updates(map[string]interface{}{"rules": string(raw), "version": 1}).Error; err != nil {
		t.Fatalf("seed ruleset rules: %v", err)
	}
	// 内部账号判定走 IsInternalAccount → ResolveCloudUin → CVMUinFromCtx，
	// 通过 ctxWithUin 注入 UIN（命中内部白名单 3205597606）。
	ctx := ctxWithUin("3205597606")

	fake := &fakeSGPoolVpcClient{}
	restore := withFakeSGPoolVpcClient(fake)
	defer restore()

	if err := RefreshOfficeIngressRulesForTenant(ctx); err != nil {
		t.Fatalf("refresh office rules: %v", err)
	}
	if len(fake.modifyReqs) != 1 {
		t.Fatalf("expected one fan-out call, got %d", len(fake.modifyReqs))
	}

	var fresh model.RuleSet
	if err := db.First(&fresh, rs.ID).Error; err != nil {
		t.Fatalf("load ruleset: %v", err)
	}
	if fresh.Version != 2 {
		t.Fatalf("version = %d, want 2", fresh.Version)
	}
	var rules []Rule
	if err := json.Unmarshal([]byte(fresh.Rules), &rules); err != nil {
		t.Fatalf("unmarshal rules: %v", err)
	}
	wantRuleCount := len(officeIngressCIDRs) + 2
	if len(rules) != wantRuleCount {
		t.Fatalf("rules count = %d, want %d", len(rules), wantRuleCount)
	}
	for _, r := range rules {
		if !r.IsRequired {
			t.Fatalf("office rule %s was not marked required", r.CidrBlock)
		}
	}
	if rules[0].CidrBlock != "219.133.41.27/32" || rules[0].Action != "ACCEPT" {
		t.Fatalf("deduped first rule = %+v, want normalized office ACCEPT", rules[0])
	}
	if rules[len(officeIngressCIDRs)].Action != "DROP" || rules[len(officeIngressCIDRs)].CidrBlock != "0.0.0.0/0" {
		t.Fatalf("IPv4 drop rule is not after office allowlist: %+v", rules[len(officeIngressCIDRs)])
	}
	if rules[len(officeIngressCIDRs)+1].Action != "DROP" || rules[len(officeIngressCIDRs)+1].CidrBlock != "::/0" {
		t.Fatalf("IPv6 drop rule is not after office allowlist: %+v", rules[len(officeIngressCIDRs)+1])
	}

	var pool model.ManagedSGPool
	if err := db.Where("sg_id = ?", "sg-office-active").First(&pool).Error; err != nil {
		t.Fatalf("load pool: %v", err)
	}
	if pool.RuleVersion != 2 {
		t.Fatalf("pool rule_version = %d, want 2", pool.RuleVersion)
	}

	if err := RefreshOfficeIngressRulesForTenant(ctx); err != nil {
		t.Fatalf("second refresh office rules: %v", err)
	}
	if len(fake.modifyReqs) != 1 {
		t.Fatalf("second ensure should be no-op, fan-out calls = %d", len(fake.modifyReqs))
	}
}

func TestRefreshOfficeIngressRulesForTenant_SkipsOtherUIN(t *testing.T) {
	db := setupSGPoolTestDB(t)
	rs := seedRuleSetAndSGs(t, db, []model.ManagedSGPool{{SGID: "sg-non-office-active"}})
	// 非内部账号 UIN（不在 internalAccountUins 白名单中）：
	// 走 IsInternalAccount 应判定为 false，进而 office ingress 规则不下发。
	ctx := ctxWithUin("1000000000")

	fake := &fakeSGPoolVpcClient{}
	restore := withFakeSGPoolVpcClient(fake)
	defer restore()

	if err := RefreshOfficeIngressRulesForTenant(ctx); err != nil {
		t.Fatalf("refresh office rules: %v", err)
	}
	if len(fake.modifyReqs) != 0 {
		t.Fatalf("unexpected fan-out calls: %d", len(fake.modifyReqs))
	}
	var fresh model.RuleSet
	if err := db.First(&fresh, rs.ID).Error; err != nil {
		t.Fatalf("load ruleset: %v", err)
	}
	if fresh.Rules != "[]" || fresh.Version != 1 {
		t.Fatalf("unexpected ruleset mutation: rules=%s version=%d", fresh.Rules, fresh.Version)
	}
}

func TestRefreshOfficeIngressRulesForTenant_PreservesExistingRuleAfterGuard(t *testing.T) {
	db := setupSGPoolTestDB(t)
	rs := seedRuleSetAndSGs(t, db, []model.ManagedSGPool{{SGID: "sg-office-preserve"}})
	seedRules := []Rule{{
		Direction:         "INGRESS",
		Protocol:          "TCP",
		Port:              "443",
		CidrBlock:         "10.0.0.0/8",
		Action:            "ACCEPT",
		PolicyDescription: "custom rule",
	}}
	raw, err := json.Marshal(seedRules)
	if err != nil {
		t.Fatalf("marshal seed rules: %v", err)
	}
	if err := db.Model(rs).Updates(map[string]interface{}{"rules": string(raw), "version": 1}).Error; err != nil {
		t.Fatalf("seed ruleset rules: %v", err)
	}

	fake := &fakeSGPoolVpcClient{}
	restore := withFakeSGPoolVpcClient(fake)
	defer restore()

	if err := RefreshOfficeIngressRulesForTenant(ctxWithUin("3205597606")); err != nil {
		t.Fatalf("refresh office rules: %v", err)
	}

	var fresh model.RuleSet
	if err := db.First(&fresh, rs.ID).Error; err != nil {
		t.Fatalf("load ruleset: %v", err)
	}
	var rules []Rule
	if err := json.Unmarshal([]byte(fresh.Rules), &rules); err != nil {
		t.Fatalf("unmarshal rules: %v", err)
	}
	wantRuleCount := len(officeIngressCIDRs) + 3
	if len(rules) != wantRuleCount {
		t.Fatalf("rules count = %d, want office guard + custom: %d", len(rules), wantRuleCount)
	}
	if rules[0].PolicyDescription != "办公网入站白名单" || rules[0].Action != "ACCEPT" {
		t.Fatalf("office allowlist was not placed at the front: %+v", rules[0])
	}
	if rules[len(officeIngressCIDRs)].Action != "DROP" || rules[len(officeIngressCIDRs)+1].Action != "DROP" {
		t.Fatalf("office drop rules were not placed before custom rules: %+v / %+v", rules[len(officeIngressCIDRs)], rules[len(officeIngressCIDRs)+1])
	}
	custom := rules[len(officeIngressCIDRs)+2]
	if custom.PolicyDescription != "custom rule" || custom.IsRequired {
		t.Fatalf("custom rule was not preserved after office guard: %+v", custom)
	}
}
