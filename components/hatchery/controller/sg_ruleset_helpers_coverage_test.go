package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// sg_ruleset_helpers_coverage_test.go —— 增量覆盖率补齐。
//
// 覆盖目标（tasks.md 1.10 之外的补丁）：
//   - ensureRuleInRuleSet 的 RuleSet 不存在 / Rules JSON 损坏 / required 剔除 3 条
//   - ensureRuleOnInstanceRuleSet 便捷包装的成功/失败 2 条
//   - ensureRuleInAllRuleSets 单条 RuleSet 失败导致整体报错 1 条
//   - resolveNetworkParams 新模型从 DefaultRuleSet ACTIVE 池读 sgIds 的 2 条
//   - HandleCheckGatewayAccess 新模型响应分支（实例未绑 SG / 全新租户 / 同步态放通+拒绝 / 不支持类型）

// ============================================================================
// ensureRuleInRuleSet 错误分支
// ============================================================================

func TestEnsureRuleInRuleSet_RuleSetNotFound(t *testing.T) {
	_ = setupSGPoolTestDB(t)
	newRule := Rule{Direction: "INGRESS", Protocol: "TCP", Port: "7540", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"}
	err := ensureRuleInRuleSet(context.Background(), 999999, newRule)
	if err == nil {
		t.Fatal("expected error for non-existent rule_set id")
	}
	if !errors.Is(err, hcommon.I18nError(i18n.MsgSGRulesetLoadRuleSetFailed)) {
		t.Errorf("expected 'load rule_set' error, got: %v", err)
	}
}

func TestEnsureRuleInRuleSet_BadRulesJSON(t *testing.T) {
	db := setupSGPoolTestDB(t)
	rs := model.RuleSet{
		Name: model.DefaultRuleSetName, IsDefault: true, Version: 1,
		Rules: `{not a JSON array`,
	}
	db.Create(&rs)

	newRule := Rule{Direction: "INGRESS", Protocol: "TCP", Port: "7540", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"}
	err := ensureRuleInRuleSet(context.Background(), rs.ID, newRule)
	if err == nil {
		t.Fatal("expected JSON parse error")
	}
	if !errors.Is(err, hcommon.I18nError(i18n.MsgSGPoolParseRulesJSONFailed)) {
		t.Errorf("expected 'parse rule_set' error, got: %v", err)
	}
}

// TestEnsureRuleInRuleSet_AppendsNewRule_WithRequiredStripped 确保已存在的 IsRequired 规则
// 在 Append 前被剔除（后续由 MergeRequiredRules 重新合入），而不是作为用户规则重复写入。
func TestEnsureRuleInRuleSet_AppendsNewRule_WithRequiredStripped(t *testing.T) {
	db := setupSGPoolTestDB(t)
	rs := model.RuleSet{
		Name: model.DefaultRuleSetName, IsDefault: true, Version: 1,
		// 先放一条 IsRequired=true 的规则，验证 append 时被剔除不重复
		Rules: `[{"direction":"INGRESS","protocol":"TCP","port":"22","cidr_block":"0.0.0.0/0","action":"ACCEPT","is_required":true}]`,
	}
	db.Create(&rs)

	newRule := Rule{Direction: "INGRESS", Protocol: "TCP", Port: "7540", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"}
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
// ensureRuleOnInstanceRuleSet 便捷包装
// ============================================================================

func TestEnsureRuleOnInstanceRuleSet_Success(t *testing.T) {
	db := setupSGPoolTestDB(t)
	rs := model.RuleSet{Name: model.DefaultRuleSetName, IsDefault: true, Version: 1, Rules: "[]"}
	db.Create(&rs)
	// ManagedSGPool 行用于 resolveRuleSetIDForInstance 反查，但状态设为 FROZEN
	// 避免被 ListActiveSGsForFanout 纳入下发目标（本测试无云 API mock）。
	db.Create(&model.ManagedSGPool{SGID: "sg-i-bound", RuleSetID: rs.ID, Status: model.SGStatusFrozen, RuleVersion: 1})

	inst := &model.Instance{InstanceId: "ins-x", SecurityGroupId: "sg-i-bound"}
	newRule := Rule{Direction: "INGRESS", Protocol: "TCP", Port: "8888", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"}
	if err := ensureRuleOnInstanceRuleSet(context.Background(), inst, newRule); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var reloaded model.RuleSet
	db.First(&reloaded, rs.ID)
	if reloaded.Version != 2 {
		t.Errorf("version should bump to 2, got %d", reloaded.Version)
	}
}

func TestEnsureRuleOnInstanceRuleSet_OrphanInstance(t *testing.T) {
	_ = setupSGPoolTestDB(t)
	inst := &model.Instance{InstanceId: "ins-orphan", SecurityGroupId: "sg-ghost"}
	newRule := Rule{Direction: "INGRESS", Protocol: "TCP", Port: "8888", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"}
	err := ensureRuleOnInstanceRuleSet(context.Background(), inst, newRule)
	if err == nil {
		t.Fatal("expected error for orphan instance")
	}
	if !errors.Is(err, hcommon.I18nError(i18n.MsgSGRulesetSGNotInPool)) {
		t.Errorf("expected 'not in any managed pool' error, got: %v", err)
	}
}

// ============================================================================
// ensureRuleInAllRuleSets 部分失败
// ============================================================================

// TestEnsureRuleInAllRuleSets_AbortOnFirstFailure 当某 RuleSet 的 Rules JSON 损坏导致
// ensureRuleInRuleSet 失败时，ensureRuleInAllRuleSets 应立即返回错误（已成功的前置 RuleSet
// 保留变更，后续未处理的 RuleSet 不再继续）。
func TestEnsureRuleInAllRuleSets_AbortOnFirstFailure(t *testing.T) {
	db := setupSGPoolTestDB(t)
	// rsA 正常，rsB 坏 JSON
	rsA := model.RuleSet{Name: "rs-a", IsDefault: true, Version: 1, Rules: "[]"}
	db.Create(&rsA)
	rsB := model.RuleSet{Name: "rs-b", Version: 1, Rules: `{bad`}
	db.Create(&rsB)

	newRule := Rule{Direction: "INGRESS", Protocol: "TCP", Port: "9999", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"}
	err := ensureRuleInAllRuleSets(context.Background(), newRule)
	if err == nil {
		t.Fatal("expected error because rs-b has bad JSON")
	}
	if !errors.Is(err, hcommon.I18nError(i18n.MsgSGRulesetEnsureRuleFailed)) {
		t.Errorf("expected 'ensure rule in ruleset' wrap, got: %v", err)
	}

	// rsA 应已成功写入（部分成功保留）
	var reloadedA model.RuleSet
	db.First(&reloadedA, rsA.ID)
	if reloadedA.Version != 2 {
		t.Errorf("rs-a should have bumped version to 2, got %d", reloadedA.Version)
	}
}

// ============================================================================
// resolveNetworkParams（admin_memory_pro.go）新路径
// ============================================================================

func TestResolveNetworkParams_UsesActiveSGsFromDefaultRuleSet(t *testing.T) {
	db := setupSGPoolTestDB(t)
	// 建 DefaultRuleSet + 2 个 ACTIVE SG + 1 个 FROZEN SG
	rs := model.RuleSet{Name: model.DefaultRuleSetName, IsDefault: true, Version: 1, Rules: "[]"}
	db.Create(&rs)
	db.Create(&model.ManagedSGPool{SGID: "sg-active-1", RuleSetID: rs.ID, Status: model.SGStatusActive, RuleVersion: 1})
	db.Create(&model.ManagedSGPool{SGID: "sg-active-2", RuleSetID: rs.ID, Status: model.SGStatusActive, RuleVersion: 1})
	db.Create(&model.ManagedSGPool{SGID: "sg-frozen", RuleSetID: rs.ID, Status: model.SGStatusFrozen, RuleVersion: 1})

	cfg := &model.SiteConfig{
		VpcId:     "vpc-1",
		SubnetIds: `{"ap-guangzhou-3":"subnet-1"}`,
	}
	_, _, sgIds, err := resolveNetworkParams(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sgIds) != 2 {
		t.Fatalf("expected 2 sgIds from ACTIVE pool, got %v", sgIds)
	}
	if sgIds[0] != "sg-active-1" || sgIds[1] != "sg-active-2" {
		t.Errorf("expected [sg-active-1 sg-active-2], got %v", sgIds)
	}
}

func TestResolveNetworkParams_NoRuleSetMakesEmptySGIds(t *testing.T) {
	_ = setupSGPoolTestDB(t)
	// 不建 RuleSet → GetDefaultRuleSet 报错 → sgIds 为空
	cfg := &model.SiteConfig{
		VpcId:     "vpc-1",
		SubnetIds: `{"ap-guangzhou-3":"subnet-1"}`,
	}
	_, _, sgIds, err := resolveNetworkParams(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sgIds) != 0 {
		t.Errorf("expected empty sgIds when no RuleSet, got %v", sgIds)
	}
}

// ============================================================================
// HandleCheckGatewayAccess 新模型响应分支
//
// 复用 openclaw_gateway_test.go 中的 initGatewayTestDB（只 migrate User/Instance/SiteConfig）
// 并手动补 migrate RuleSet / ManagedSGPool，插入测试数据。
// ============================================================================

// TestHandleCheckGatewayAccess_InstanceHasNoSecurityGroupId 覆盖"实例未绑定安全组"分支。
func TestHandleCheckGatewayAccess_InstanceHasNoSecurityGroupId(t *testing.T) {
	cleanup := initGatewayTestDB(t)
	defer cleanup()

	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)

	config := model.SiteConfig{ID: 1, GatewayUIEnable: true, GatewayUIPort: 8080}
	model.DB(context.Background()).Create(&config)

	proxyToken := "sk-test"
	inst := model.Instance{
		Name:            "no-sg",
		InstanceId:      "ins-no-sg",
		UserID:          user.ID,
		AgentType:       "openclaw",
		SecurityGroupId: "",
		ProxyToken:      &proxyToken,
	}
	model.DB(context.Background()).Create(&inst)

	req := userGatewayReqWithSession(t, http.MethodGet, "/openclaw/check-gateway-access?id=1", "testuser")
	rr := httptest.NewRecorder()
	HandleCheckGatewayAccess(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d, body=%s", rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["accessible"] != false {
		t.Errorf("expected accessible=false, got %v", resp["accessible"])
	}
	if resp["drifting"] != false {
		t.Errorf("expected drifting=false, got %v", resp["drifting"])
	}
	sgs, ok := resp["securityGroupIds"].([]interface{})
	if !ok || len(sgs) != 0 {
		t.Errorf("expected empty securityGroupIds array, got %v", resp["securityGroupIds"])
	}
	msg, _ := resp["message"].(string)
	if !strings.Contains(msg, "未绑定任何安全组") {
		t.Errorf("expected '未绑定任何安全组' in message, got: %s", msg)
	}
}

// TestHandleCheckGatewayAccess_FreshTenant 覆盖"全新租户无 RuleSet"分支。
func TestHandleCheckGatewayAccess_FreshTenant(t *testing.T) {
	cleanup := initGatewayTestDB(t)
	defer cleanup()

	if err := model.DB(context.Background()).AutoMigrate(&model.RuleSet{}, &model.ManagedSGPool{}); err != nil {
		t.Fatalf("migrate rule_set: %v", err)
	}

	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)
	config := model.SiteConfig{ID: 1, GatewayUIEnable: true, GatewayUIPort: 8080}
	model.DB(context.Background()).Create(&config)

	proxyToken := "sk-test"
	inst := model.Instance{
		Name: "fresh", InstanceId: "ins-fresh", UserID: user.ID, AgentType: "openclaw",
		SecurityGroupId: "sg-any", ProxyToken: &proxyToken,
	}
	model.DB(context.Background()).Create(&inst)

	req := userGatewayReqWithSession(t, http.MethodGet, "/openclaw/check-gateway-access?id=1", "testuser")
	rr := httptest.NewRecorder()
	HandleCheckGatewayAccess(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK (friendly degraded), got %d, body=%s", rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["accessible"] != false {
		t.Errorf("expected accessible=false, got %v", resp["accessible"])
	}
	msg, _ := resp["message"].(string)
	if !strings.Contains(msg, "安全组尚未初始化") {
		t.Errorf("expected bootstrap hint, got: %s", msg)
	}
	sgs, _ := resp["securityGroupIds"].([]interface{})
	if len(sgs) != 1 || sgs[0] != "sg-any" {
		t.Errorf("expected securityGroupIds=[sg-any], got %v", resp["securityGroupIds"])
	}
}

// TestHandleCheckGatewayAccess_AllowedSynced 覆盖"同步态 + 已放通"分支。
func TestHandleCheckGatewayAccess_AllowedSynced(t *testing.T) {
	cleanup := initGatewayTestDB(t)
	defer cleanup()

	if err := model.DB(context.Background()).AutoMigrate(&model.RuleSet{}, &model.ManagedSGPool{}); err != nil {
		t.Fatalf("migrate rule_set: %v", err)
	}

	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)
	config := model.SiteConfig{ID: 1, GatewayUIEnable: true, GatewayUIPort: 8080}
	model.DB(context.Background()).Create(&config)

	rs := model.RuleSet{
		Name: model.DefaultRuleSetName, IsDefault: true, Version: 3,
		Rules: `[{"direction":"INGRESS","protocol":"TCP","port":"8080","cidr_block":"0.0.0.0/0","action":"ACCEPT"}]`,
	}
	model.DB(context.Background()).Create(&rs)
	model.DB(context.Background()).Create(&model.ManagedSGPool{
		SGID: "sg-synced", RuleSetID: rs.ID, Status: model.SGStatusActive, RuleVersion: 3,
	})

	proxyToken := "sk-test"
	inst := model.Instance{
		Name: "synced", InstanceId: "ins-synced", UserID: user.ID, AgentType: "openclaw",
		SecurityGroupId: "sg-synced", ProxyToken: &proxyToken,
	}
	model.DB(context.Background()).Create(&inst)

	req := userGatewayReqWithSession(t, http.MethodGet, "/openclaw/check-gateway-access?id=1", "testuser")
	rr := httptest.NewRecorder()
	HandleCheckGatewayAccess(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["accessible"] != true {
		t.Errorf("expected accessible=true, got %v", resp["accessible"])
	}
	if resp["drifting"] != false {
		t.Errorf("expected drifting=false (synced), got %v", resp["drifting"])
	}
}

// TestHandleCheckGatewayAccess_DeniedSynced 覆盖"同步态 + 规则未放通"分支。
func TestHandleCheckGatewayAccess_DeniedSynced(t *testing.T) {
	cleanup := initGatewayTestDB(t)
	defer cleanup()

	if err := model.DB(context.Background()).AutoMigrate(&model.RuleSet{}, &model.ManagedSGPool{}); err != nil {
		t.Fatalf("migrate rule_set: %v", err)
	}

	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)
	config := model.SiteConfig{ID: 1, GatewayUIEnable: true, GatewayUIPort: 8080}
	model.DB(context.Background()).Create(&config)

	rs := model.RuleSet{
		Name: model.DefaultRuleSetName, IsDefault: true, Version: 2,
		Rules: `[{"direction":"INGRESS","protocol":"TCP","port":"22","cidr_block":"0.0.0.0/0","action":"ACCEPT"}]`,
	}
	model.DB(context.Background()).Create(&rs)
	model.DB(context.Background()).Create(&model.ManagedSGPool{
		SGID: "sg-denied", RuleSetID: rs.ID, Status: model.SGStatusActive, RuleVersion: 2,
	})

	proxyToken := "sk-test"
	inst := model.Instance{
		Name: "denied", InstanceId: "ins-denied", UserID: user.ID, AgentType: "openclaw",
		SecurityGroupId: "sg-denied", ProxyToken: &proxyToken,
	}
	model.DB(context.Background()).Create(&inst)

	req := userGatewayReqWithSession(t, http.MethodGet, "/openclaw/check-gateway-access?id=1", "testuser")
	rr := httptest.NewRecorder()
	HandleCheckGatewayAccess(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["accessible"] != false {
		t.Errorf("expected accessible=false, got %v", resp["accessible"])
	}
	msg, _ := resp["message"].(string)
	if !strings.Contains(msg, "尚未放通") {
		t.Errorf("expected '尚未放通' in message, got: %s", msg)
	}
}

func TestHandleCheckGatewayAccess_SourceIPWhitelistBeforeDrop(t *testing.T) {
	cleanup := initGatewayTestDB(t)
	defer cleanup()

	if err := model.DB(context.Background()).AutoMigrate(&model.RuleSet{}, &model.ManagedSGPool{}); err != nil {
		t.Fatalf("migrate rule_set: %v", err)
	}

	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)
	config := model.SiteConfig{ID: 1, GatewayUIEnable: true, GatewayUIPort: 8080}
	model.DB(context.Background()).Create(&config)

	rs := model.RuleSet{
		Name: model.DefaultRuleSetName, IsDefault: true, Version: 4,
		Rules: `[
			{"direction":"INGRESS","protocol":"ALL","port":"ALL","cidr_block":"10.0.0.0/8","action":"ACCEPT"},
			{"direction":"INGRESS","protocol":"ALL","port":"ALL","cidr_block":"0.0.0.0/0","action":"DROP"},
			{"direction":"INGRESS","protocol":"TCP","port":"8080","cidr_block":"0.0.0.0/0","action":"ACCEPT"}
		]`,
	}
	model.DB(context.Background()).Create(&rs)
	model.DB(context.Background()).Create(&model.ManagedSGPool{
		SGID: "sg-office", RuleSetID: rs.ID, Status: model.SGStatusActive, RuleVersion: 4,
	})

	proxyToken := "sk-test"
	inst := model.Instance{
		Name: "office", InstanceId: "ins-office", UserID: user.ID, AgentType: "openclaw",
		SecurityGroupId: "sg-office", ProxyToken: &proxyToken,
	}
	model.DB(context.Background()).Create(&inst)

	req := userGatewayReqWithSession(t, http.MethodGet, "/openclaw/check-gateway-access?id=1", "testuser")
	req.Header.Set("X-Forwarded-For", "10.1.2.3")
	rr := httptest.NewRecorder()
	HandleCheckGatewayAccess(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["accessible"] != true {
		t.Fatalf("office source should be accessible, got %v body=%s", resp["accessible"], rr.Body.String())
	}

	req = userGatewayReqWithSession(t, http.MethodGet, "/openclaw/check-gateway-access?id=1", "testuser")
	req.Header.Set("X-Forwarded-For", "203.0.113.10")
	rr = httptest.NewRecorder()
	HandleCheckGatewayAccess(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", rr.Code, rr.Body.String())
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["accessible"] != false {
		t.Fatalf("non-office source should be denied by fallback drop, got %v body=%s", resp["accessible"], rr.Body.String())
	}
}

// TestRefreshAllRuleSetsForRequiredRules_NoRuleSet 覆盖无 RuleSet 时的早期返回。
func TestRefreshAllRuleSetsForRequiredRules_NoRuleSet(t *testing.T) {
	_ = setupSGPoolTestDB(t)
	err := RefreshAllRuleSetsForRequiredRules(context.Background())
	if err == nil {
		t.Fatal("expected error when no rule_set")
	}
	if err != ErrSGBootstrapNotDone {
		t.Errorf("expected 'no rule_set found', got: %v", err)
	}
}

// TestRefreshAllRuleSetsForRequiredRules_HappyPath 覆盖正常遍历路径。
// 无 ACTIVE SG → UpdateRuleSetRulesInternal 直接 DB commit，避免云 API。
func TestRefreshAllRuleSetsForRequiredRules_HappyPath(t *testing.T) {
	db := setupSGPoolTestDB(t)
	rs := model.RuleSet{Name: model.DefaultRuleSetName, IsDefault: true, Version: 1, Rules: "[]"}
	db.Create(&rs)

	if err := RefreshAllRuleSetsForRequiredRules(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var reloaded model.RuleSet
	db.First(&reloaded, rs.ID)
	// 刷新后 version 应 bump（即使 Rules 内容可能与合并后相同，UpdateRuleSetRulesInternal 会 +1）
	if reloaded.Version != 2 {
		t.Errorf("version should bump to 2, got %d", reloaded.Version)
	}
}

// TestRefreshAllRuleSetsForRequiredRules_PartialFailureBestEffort 覆盖 best-effort 继续。
// rsA 正常，rsB 坏 JSON → 失败但不终止，rsC 继续处理。最终返回错误汇总。
func TestRefreshAllRuleSetsForRequiredRules_PartialFailureBestEffort(t *testing.T) {
	db := setupSGPoolTestDB(t)
	rsA := model.RuleSet{Name: "rs-a", IsDefault: true, Version: 1, Rules: "[]"}
	db.Create(&rsA)
	rsB := model.RuleSet{Name: "rs-b", Version: 1, Rules: `{bad`}
	db.Create(&rsB)
	rsC := model.RuleSet{Name: "rs-c", Version: 1, Rules: "[]"}
	db.Create(&rsC)

	err := RefreshAllRuleSetsForRequiredRules(context.Background())
	if err == nil {
		t.Fatal("expected error due to bad JSON in rs-b")
	}
	if !errors.Is(err, hcommon.I18nError(i18n.MsgSGRulesetRefreshFailed)) {
		t.Errorf("expected summary error, got: %v", err)
	}
	// rs-a 和 rs-c 应都已成功（best-effort 继续），version bump
	var a, c model.RuleSet
	db.First(&a, rsA.ID)
	db.First(&c, rsC.ID)
	if a.Version != 2 {
		t.Errorf("rs-a should bump to 2, got %d", a.Version)
	}
	if c.Version != 2 {
		t.Errorf("rs-c should bump to 2 (best-effort continues), got %d", c.Version)
	}
}

// TestHandleCheckGatewayAccess_UnsupportedAgentType 覆盖不支持的 agent type → 400。
func TestHandleCheckGatewayAccess_UnsupportedAgentType(t *testing.T) {
	cleanup := initGatewayTestDB(t)
	defer cleanup()

	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)
	config := model.SiteConfig{ID: 1, GatewayUIEnable: true, GatewayUIPort: 8080}
	model.DB(context.Background()).Create(&config)

	proxyToken := "sk-test"
	inst := model.Instance{
		Name: "unsupported", InstanceId: "ins-unsup", UserID: user.ID,
		AgentType: "unknown-type", ProxyToken: &proxyToken,
	}
	model.DB(context.Background()).Create(&inst)

	req := userGatewayReqWithSession(t, http.MethodGet, "/openclaw/check-gateway-access?id=1", "testuser")
	rr := httptest.NewRecorder()
	HandleCheckGatewayAccess(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for unsupported agent type, got %d, body=%s", rr.Code, rr.Body.String())
	}
}
