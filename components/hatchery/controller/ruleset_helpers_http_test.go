package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hatchery/model"

	"github.com/gorilla/sessions"
)

// ============================================================================
// controller/ruleset_helpers.go HTTP handler 快速测试
//
// 目标：走通 HandleGetRuleSet / HandleCreateRuleSet / HandleUpdateRuleSetRules /
// HandleImportRulesFromSG 的参数校验和未授权路径，把覆盖率从 0% 拉到可见水平。
// 深层集成路径（实际 sg-init / fan-out）走另外的集成测试，不在此覆盖。
// ============================================================================

// rulesetHandlerSetup 最小 handler 测试环境：设置 Admin Token + Store + 空白 DB。
func rulesetHandlerSetup(t *testing.T) {
	t.Helper()
	_ = setupSGPoolTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	origStore := Store
	Store = sessions.NewCookieStore([]byte("ruleset-handler-test-key"))
	t.Cleanup(func() {
		AdminToken = origToken
		Store = origStore
	})
}

// rsAdminReq 构造 admin Bearer 请求。
func rsAdminReq(method, path, body string) *http.Request {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer test-admin-token")
	return r
}

// ---------------- HandleGetRuleSet ----------------

func TestHandleGetRuleSet_Unauthorized(t *testing.T) {
	rulesetHandlerSetup(t)
	req := httptest.NewRequest("GET", "/admin/config/security-group/ruleset", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	HandleGetRuleSet(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestHandleGetRuleSet_UninitializedReturnsInitializedFalse(t *testing.T) {
	rulesetHandlerSetup(t)
	w := httptest.NewRecorder()
	HandleGetRuleSet(w, rsAdminReq("GET", "/admin/config/security-group/ruleset", ""))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var resp map[string]any
	_ = json.NewDecoder(w.Body).Decode(&resp)
	// initialized=false 在空 DB 时应该是这样
	if resp["initialized"] != false {
		t.Errorf("expected initialized=false for empty DB, got %v", resp["initialized"])
	}
}

func TestHandleGetRuleSet_ReturnsExistingRuleSet(t *testing.T) {
	rulesetHandlerSetup(t)
	rs := &model.RuleSet{
		Name: model.DefaultRuleSetName, Description: "hello", Rules: "[]",
		Version: 2, IsDefault: true,
	}
	model.DB(context.Background()).Create(rs)

	w := httptest.NewRecorder()
	HandleGetRuleSet(w, rsAdminReq("GET", "/admin/config/security-group/ruleset", ""))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, `"initialized":true`) {
		t.Errorf("body should set initialized=true, got: %s", body)
	}
	if !strings.Contains(body, `"description":"hello"`) {
		t.Errorf("body should surface description, got: %s", body)
	}
}

// ---------------- HandleCreateRuleSet ----------------

func TestHandleCreateRuleSet_Unauthorized(t *testing.T) {
	rulesetHandlerSetup(t)
	req := httptest.NewRequest("POST", "/admin/config/security-group/rulesets",
		strings.NewReader(`{"name":"acme"}`))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	HandleCreateRuleSet(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestHandleCreateRuleSet_IdempotentWhenExists(t *testing.T) {
	rulesetHandlerSetup(t)
	// 已存在 DefaultRuleSetName 的 RuleSet → 幂等返回现有（GetDefaultRuleSet 查的是这个名字）
	existing := &model.RuleSet{
		Name: model.DefaultRuleSetName, Description: "x",
		Rules: "[]", Version: 1, IsDefault: true,
	}
	model.DB(context.Background()).Create(existing)

	w := httptest.NewRecorder()
	HandleCreateRuleSet(w, rsAdminReq("POST", "/admin/config/security-group/rulesets",
		`{"name":"`+model.DefaultRuleSetName+`","rules":[]}`))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 (idempotent), got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"name":"`+model.DefaultRuleSetName+`"`) {
		t.Errorf("body missing existing name: %s", w.Body.String())
	}
}

func TestHandleCreateRuleSet_InvalidName(t *testing.T) {
	rulesetHandlerSetup(t)
	w := httptest.NewRecorder()
	HandleCreateRuleSet(w, rsAdminReq("POST", "/admin/config/security-group/rulesets",
		`{"name":"1abc","rules":[]}`)) // 数字开头
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid name, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestHandleCreateRuleSet_BadJSON(t *testing.T) {
	rulesetHandlerSetup(t)
	w := httptest.NewRecorder()
	HandleCreateRuleSet(w, rsAdminReq("POST", "/admin/config/security-group/rulesets",
		`{broken`))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad json, got %d", w.Code)
	}
}

// ---------------- HandleUpdateRuleSetRules ----------------

func TestHandleUpdateRuleSetRules_Unauthorized(t *testing.T) {
	rulesetHandlerSetup(t)
	req := httptest.NewRequest("POST", "/admin/config/security-group/ruleset/rules",
		strings.NewReader(`{"rules":[]}`))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	HandleUpdateRuleSetRules(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestHandleUpdateRuleSetRules_BadJSON(t *testing.T) {
	rulesetHandlerSetup(t)
	w := httptest.NewRecorder()
	HandleUpdateRuleSetRules(w, rsAdminReq("POST", "/admin/config/security-group/ruleset/rules",
		`{oops`))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleUpdateRuleSetRules_InvalidName(t *testing.T) {
	rulesetHandlerSetup(t)
	w := httptest.NewRecorder()
	HandleUpdateRuleSetRules(w, rsAdminReq("POST", "/admin/config/security-group/ruleset/rules",
		`{"name":"_bad","rules":[]}`))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad name, got %d body=%s", w.Code, w.Body.String())
	}
}

// ---------------- HandleImportRulesFromSG ----------------

func TestHandleImportRulesFromSG_Unauthorized(t *testing.T) {
	rulesetHandlerSetup(t)
	req := httptest.NewRequest("POST", "/admin/config/security-group/ruleset/import-from-sg",
		strings.NewReader(`{"source_sg_id":"sg-x"}`))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	HandleImportRulesFromSG(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestHandleImportRulesFromSG_BadJSON(t *testing.T) {
	rulesetHandlerSetup(t)
	w := httptest.NewRecorder()
	HandleImportRulesFromSG(w, rsAdminReq("POST", "/admin/config/security-group/ruleset/import-from-sg",
		`{broken`))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleImportRulesFromSG_MissingSourceID(t *testing.T) {
	rulesetHandlerSetup(t)
	w := httptest.NewRecorder()
	HandleImportRulesFromSG(w, rsAdminReq("POST", "/admin/config/security-group/ruleset/import-from-sg",
		`{}`))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing source_sg_id, got %d body=%s", w.Code, w.Body.String())
	}
}

// ---------------- HandleReorderRuleSetRules ----------------

func TestHandleReorderRuleSetRules_Unauthorized(t *testing.T) {
	rulesetHandlerSetup(t)
	req := httptest.NewRequest("POST", "/admin/config/security-group/ruleset/rules/reorder",
		strings.NewReader(`{"ordered_fingerprints":["fp"]}`))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	HandleReorderRuleSetRules(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestHandleReorderRuleSetRules_BadJSON(t *testing.T) {
	rulesetHandlerSetup(t)
	w := httptest.NewRecorder()
	HandleReorderRuleSetRules(w, rsAdminReq("POST",
		"/admin/config/security-group/ruleset/rules/reorder", `{oops`))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad json, got %d", w.Code)
	}
}

func TestHandleReorderRuleSetRules_InvalidName(t *testing.T) {
	rulesetHandlerSetup(t)
	w := httptest.NewRecorder()
	HandleReorderRuleSetRules(w, rsAdminReq("POST",
		"/admin/config/security-group/ruleset/rules/reorder",
		`{"name":"_bad","ordered_fingerprints":["x"]}`))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid name, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestHandleReorderRuleSetRules_EmptyFingerprints(t *testing.T) {
	rulesetHandlerSetup(t)
	w := httptest.NewRecorder()
	HandleReorderRuleSetRules(w, rsAdminReq("POST",
		"/admin/config/security-group/ruleset/rules/reorder",
		`{"ordered_fingerprints":[]}`))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty fingerprints, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestHandleReorderRuleSetRules_DuplicateFingerprints(t *testing.T) {
	rulesetHandlerSetup(t)
	w := httptest.NewRecorder()
	HandleReorderRuleSetRules(w, rsAdminReq("POST",
		"/admin/config/security-group/ruleset/rules/reorder",
		`{"ordered_fingerprints":["x","x"]}`))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for duplicate fingerprints, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestHandleReorderRuleSetRules_RuleSetNotFound(t *testing.T) {
	rulesetHandlerSetup(t)
	// DB 里没有 RuleSet → 404
	w := httptest.NewRecorder()
	HandleReorderRuleSetRules(w, rsAdminReq("POST",
		"/admin/config/security-group/ruleset/rules/reorder",
		`{"ordered_fingerprints":["INGRESS|TCP|22|0.0.0.0/0|ACCEPT"]}`))
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 when rule_set absent, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestHandleReorderRuleSetRules_UnknownFingerprint(t *testing.T) {
	rulesetHandlerSetup(t)
	rules := []Rule{
		{Direction: "INGRESS", Protocol: "TCP", Port: "22", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"},
	}
	rulesJSON, _ := json.Marshal(rules)
	rs := &model.RuleSet{
		Name: model.DefaultRuleSetName, Rules: string(rulesJSON),
		Version: 1, IsDefault: true,
	}
	model.DB(context.Background()).Create(rs)

	w := httptest.NewRecorder()
	HandleReorderRuleSetRules(w, rsAdminReq("POST",
		"/admin/config/security-group/ruleset/rules/reorder",
		`{"ordered_fingerprints":["INGRESS|UDP|53|0.0.0.0/0|DROP"]}`))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for unknown fingerprint, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestHandleReorderRuleSetRules_EmptyExistingRules(t *testing.T) {
	rulesetHandlerSetup(t)
	rs := &model.RuleSet{
		Name: model.DefaultRuleSetName, Rules: "[]",
		Version: 1, IsDefault: true,
	}
	model.DB(context.Background()).Create(rs)

	w := httptest.NewRecorder()
	HandleReorderRuleSetRules(w, rsAdminReq("POST",
		"/admin/config/security-group/ruleset/rules/reorder",
		`{"ordered_fingerprints":["INGRESS|TCP|22|0.0.0.0/0|ACCEPT"]}`))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty existing rules, got %d body=%s", w.Code, w.Body.String())
	}
}

// TestHandleReorderRuleSetRules_HappyPath_NoActiveSGs 验证 happy path：
// 现有规则 [A,B,C]，请求顺序 [C,A] → 期望落盘 [C,A,B]（B 漏列被追加到末尾）。
// 没有 ACTIVE SG → UpdateRuleSetRulesInternal 直接走 DB commit，无云 API 副作用。
func TestHandleReorderRuleSetRules_HappyPath_NoActiveSGs(t *testing.T) {
	rulesetHandlerSetup(t)
	rA := Rule{Direction: "INGRESS", Protocol: "TCP", Port: "22", CidrBlock: "0.0.0.0/0", Action: "ACCEPT", PolicyDescription: "A"}
	rB := Rule{Direction: "INGRESS", Protocol: "TCP", Port: "80", CidrBlock: "0.0.0.0/0", Action: "ACCEPT", PolicyDescription: "B"}
	rC := Rule{Direction: "EGRESS", Protocol: "ALL", Port: "ALL", CidrBlock: "0.0.0.0/0", Action: "ACCEPT", PolicyDescription: "C"}
	rulesJSON, _ := json.Marshal([]Rule{rA, rB, rC})
	rs := &model.RuleSet{
		Name: model.DefaultRuleSetName, Rules: string(rulesJSON),
		Version: 5, IsDefault: true,
	}
	if err := model.DB(context.Background()).Create(rs).Error; err != nil {
		t.Fatalf("seed rule_set: %v", err)
	}

	body := `{"ordered_fingerprints":["` + rC.Fingerprint() + `","` + rA.Fingerprint() + `"]}`
	w := httptest.NewRecorder()
	HandleReorderRuleSetRules(w, rsAdminReq("POST",
		"/admin/config/security-group/ruleset/rules/reorder", body))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	// 校验 DB 中规则顺序：[C, A, B]
	var got model.RuleSet
	if err := model.DB(context.Background()).Where("name = ?", model.DefaultRuleSetName).First(&got).Error; err != nil {
		t.Fatalf("reload rule_set: %v", err)
	}
	if got.Version <= 5 {
		t.Errorf("version should bump after reorder, before=5 after=%d", got.Version)
	}
	var stored []Rule
	if err := json.Unmarshal([]byte(got.Rules), &stored); err != nil {
		t.Fatalf("unmarshal stored rules: %v", err)
	}
	// SiteConfig 全关 + autoFixRules=false → 不会 merge 任何 recommended，
	// 落盘应严格等于 [C,A,B]（B 漏列被追加在末尾）
	if len(stored) != 3 {
		t.Fatalf("expected 3 stored rules, got %d (%+v)", len(stored), stored)
	}
	if stored[0].PolicyDescription != "C" || stored[1].PolicyDescription != "A" || stored[2].PolicyDescription != "B" {
		t.Errorf("expected order [C,A,B], got [%s,%s,%s]",
			stored[0].PolicyDescription, stored[1].PolicyDescription, stored[2].PolicyDescription)
	}
}

// TestHandleReorderRuleSetRules_BlankFingerprint 验证 ordered_fingerprints 内出现空字符串时返回 400。
func TestHandleReorderRuleSetRules_BlankFingerprint(t *testing.T) {
	rulesetHandlerSetup(t)
	w := httptest.NewRecorder()
	HandleReorderRuleSetRules(w, rsAdminReq("POST",
		"/admin/config/security-group/ruleset/rules/reorder",
		`{"ordered_fingerprints":["INGRESS|TCP|22|0.0.0.0/0|ACCEPT","   "]}`))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for blank fingerprint, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "含空字符串") {
		t.Errorf("error message should mention 含空字符串, got: %s", w.Body.String())
	}
}

// TestHandleReorderRuleSetRules_CorruptedRulesJSON 验证 RuleSet.Rules 字段值非法 JSON 时返回 500。
func TestHandleReorderRuleSetRules_CorruptedRulesJSON(t *testing.T) {
	rulesetHandlerSetup(t)
	rs := &model.RuleSet{
		Name:      model.DefaultRuleSetName,
		Rules:     `{not-an-array`, // 故意写坏 JSON
		Version:   1,
		IsDefault: true,
	}
	if err := model.DB(context.Background()).Create(rs).Error; err != nil {
		t.Fatalf("seed rule_set: %v", err)
	}
	w := httptest.NewRecorder()
	HandleReorderRuleSetRules(w, rsAdminReq("POST",
		"/admin/config/security-group/ruleset/rules/reorder",
		`{"ordered_fingerprints":["INGRESS|TCP|22|0.0.0.0/0|ACCEPT"]}`))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for corrupted rules json, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "解析现有规则失败") {
		t.Errorf("error message should mention 解析现有规则失败, got: %s", w.Body.String())
	}
}

// TestHandleReorderRuleSetRules_NamedRuleSet 验证非默认 RuleSet（name 显式指定）的 reorder 路径，
// 同时验证：ordered_fingerprints 中重复条目在前置校验已被 400 拦截，但合法路径下，
// "未列出的规则按原顺序追加末尾"的行为对多条规则也成立。
func TestHandleReorderRuleSetRules_NamedRuleSet(t *testing.T) {
	rulesetHandlerSetup(t)
	rA := Rule{Direction: "INGRESS", Protocol: "TCP", Port: "22", CidrBlock: "0.0.0.0/0", Action: "ACCEPT", PolicyDescription: "A"}
	rB := Rule{Direction: "INGRESS", Protocol: "TCP", Port: "80", CidrBlock: "0.0.0.0/0", Action: "ACCEPT", PolicyDescription: "B"}
	rC := Rule{Direction: "INGRESS", Protocol: "TCP", Port: "443", CidrBlock: "0.0.0.0/0", Action: "ACCEPT", PolicyDescription: "C"}
	rD := Rule{Direction: "EGRESS", Protocol: "ALL", Port: "ALL", CidrBlock: "0.0.0.0/0", Action: "ACCEPT", PolicyDescription: "D"}
	rulesJSON, _ := json.Marshal([]Rule{rA, rB, rC, rD})
	rs := &model.RuleSet{
		Name:      "acme-prod",
		Rules:     string(rulesJSON),
		Version:   3,
		IsDefault: false,
	}
	if err := model.DB(context.Background()).Create(rs).Error; err != nil {
		t.Fatalf("seed rule_set: %v", err)
	}

	// 只列出 [D, C]，期望落盘 [D, C, A, B]（A、B 漏列按原相对顺序追加）
	body := `{"name":"acme-prod","ordered_fingerprints":["` + rD.Fingerprint() + `","` + rC.Fingerprint() + `"]}`
	w := httptest.NewRecorder()
	HandleReorderRuleSetRules(w, rsAdminReq("POST",
		"/admin/config/security-group/ruleset/rules/reorder", body))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var got model.RuleSet
	if err := model.DB(context.Background()).Where("name = ?", "acme-prod").First(&got).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got.Version <= 3 {
		t.Errorf("version should bump after reorder, before=3 after=%d", got.Version)
	}
	var stored []Rule
	if err := json.Unmarshal([]byte(got.Rules), &stored); err != nil {
		t.Fatalf("unmarshal stored: %v", err)
	}
	if len(stored) != 4 {
		t.Fatalf("expected 4 stored rules, got %d", len(stored))
	}
	wantOrder := []string{"D", "C", "A", "B"}
	for i, want := range wantOrder {
		if stored[i].PolicyDescription != want {
			t.Errorf("stored[%d].desc=%q, want=%q (full=[%s,%s,%s,%s])",
				i, stored[i].PolicyDescription, want,
				stored[0].PolicyDescription, stored[1].PolicyDescription,
				stored[2].PolicyDescription, stored[3].PolicyDescription)
		}
	}
}
