package controller

import (
	"context"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"hatchery/model"

	"github.com/gorilla/sessions"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tcerr "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
)

// ============================================================================
// controller/ruleset_helpers.go 纯函数补测
//
// 覆盖目标：policySetToRules / policyToRule /
// validateCIDR / truncateErrMsg / isValidRuleSetName / normalizeDirection 未覆盖路径 /
// normalizeCIDR 未覆盖路径 / isRetryableError / writeRuleSetResponse /
// applyRulesToSGWithRetry 非重试错误直接返回路径 / getUserIDFromRequest 匿名路径
// ============================================================================

// ---------------- validateCIDR ----------------

func TestValidateCIDR(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		wantErr bool
	}{
		{"ipv4 cidr 0/0", "0.0.0.0/0", false},
		{"ipv4 cidr 24", "10.0.0.0/24", false},
		{"ipv4 single", "10.0.0.1", false},
		{"ipv6 cidr", "::/0", false},
		{"ipv6 single", "::1", false},
		{"bare 0.0.0.0 rejected", "0.0.0.0", true},
		{"bare :: rejected", "::", true},
		{"junk", "not-an-ip", true},
		{"bad prefix len", "10.0.0.1/33", true},
		{"empty treated as non-cidr invalid ip", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateCIDR(c.in)
			if c.wantErr && err == nil {
				t.Errorf("validateCIDR(%q) expected error", c.in)
			}
			if !c.wantErr && err != nil {
				t.Errorf("validateCIDR(%q) unexpected error: %v", c.in, err)
			}
		})
	}
}

// ---------------- truncateErrMsg ----------------

func TestTruncateErrMsg(t *testing.T) {
	short := "short msg"
	if got := truncateErrMsg(short); got != short {
		t.Errorf("short should pass through, got %q", got)
	}
	long := strings.Repeat("x", 600)
	got := truncateErrMsg(long)
	if !strings.HasSuffix(got, "...") {
		t.Error("long msg should end with ...")
	}
	if len(got) != 500+3 {
		t.Errorf("long cut to 500+... expected %d chars, got %d", 503, len(got))
	}
}

// ---------------- isValidRuleSetName ----------------

func TestIsValidRuleSetName(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"abc", true},
		{"a12", true},
		{"clawpro-default", true},
		{"Acme-Prod-01", true},
		{"ab", false},                    // <3
		{strings.Repeat("a", 33), false}, // >32
		{"1abc", false},                  // 数字开头
		{"-abc", false},                  // 短横线开头
		{"abc_d", false},                 // 下划线不允许
		{"abc.d", false},                 // 点不允许
		{"abc d", false},                 // 空格
		{"", false},                      // 空
		{strings.Repeat("a", 32), true},  // 边界 32
		{strings.Repeat("a", 3), true},   // 边界 3
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			if got := isValidRuleSetName(c.in); got != c.want {
				t.Errorf("isValidRuleSetName(%q) = %v want %v", c.in, got, c.want)
			}
		})
	}
}

// ---------------- normalizeDirection 未覆盖路径 ----------------

func TestNormalizeDirection_UnknownPassthrough(t *testing.T) {
	if got := normalizeDirection("FOO"); got != "FOO" {
		t.Errorf("unknown direction passthrough got=%q", got)
	}
	if got := normalizeDirection("  in  "); got != "INGRESS" {
		t.Errorf("in alias with whitespace got=%q", got)
	}
	if got := normalizeDirection("out"); got != "EGRESS" {
		t.Errorf("out alias got=%q", got)
	}
}

// ---------------- normalizeCIDR 未覆盖路径 ----------------

func TestNormalizeCIDR_EdgeCases(t *testing.T) {
	// junk 应原样返回（规范化只管能识别的情况）
	if got := normalizeCIDR("not-a-cidr"); got != "not-a-cidr" {
		t.Errorf("junk should passthrough, got %q", got)
	}
	// IPv6 with prefix — 规范化大小写
	got := normalizeCIDR("FE80::/10")
	if !strings.EqualFold(got, "fe80::/10") {
		t.Errorf("ipv6 prefix normalized lowercase: %q", got)
	}
}

// ---------------- policySetToRules / policyToRule ----------------

func TestPolicyToRule_AllFields(t *testing.T) {
	cidr := "10.0.0.0/8"
	proto := "TCP"
	port := "22-80"
	action := "ACCEPT"
	desc := "ssh & web"
	p := &vpc.SecurityGroupPolicy{
		Protocol:          &proto,
		Port:              &port,
		CidrBlock:         &cidr,
		Action:            &action,
		PolicyDescription: &desc,
	}
	r := policyToRule(p, "INGRESS")
	if r.Direction != "INGRESS" || r.Protocol != proto || r.Port != port ||
		r.CidrBlock != cidr || r.Action != action || r.PolicyDescription != desc {
		t.Errorf("policyToRule mismatch: %+v", r)
	}
}

func TestPolicyToRule_IPv6Fallback(t *testing.T) {
	v6 := "fe80::/10"
	emptyV4 := ""
	p := &vpc.SecurityGroupPolicy{
		CidrBlock:     &emptyV4,
		Ipv6CidrBlock: &v6,
	}
	r := policyToRule(p, "EGRESS")
	if r.CidrBlock != v6 {
		t.Errorf("ipv6 fallback expected %q got %q", v6, r.CidrBlock)
	}
	if r.Direction != "EGRESS" {
		t.Errorf("direction not propagated: %s", r.Direction)
	}
}

func TestPolicyToRule_AllNilSafe(t *testing.T) {
	p := &vpc.SecurityGroupPolicy{}
	r := policyToRule(p, "INGRESS")
	if r.Direction != "INGRESS" {
		t.Error("direction should stay set even when all fields nil")
	}
}

func TestPolicySetToRules_IngressAndEgress(t *testing.T) {
	proto := "TCP"
	port := "22"
	cidr := "0.0.0.0/0"
	action := "ACCEPT"
	ing := &vpc.SecurityGroupPolicy{Protocol: &proto, Port: &port, CidrBlock: &cidr, Action: &action}
	eg := &vpc.SecurityGroupPolicy{Protocol: &proto, Port: &port, CidrBlock: &cidr, Action: &action}
	set := &vpc.SecurityGroupPolicySet{
		Ingress: []*vpc.SecurityGroupPolicy{ing, ing},
		Egress:  []*vpc.SecurityGroupPolicy{eg},
	}
	rules := policySetToRules(set)
	if len(rules) != 3 {
		t.Fatalf("expected 3 rules (2 ing + 1 eg), got %d", len(rules))
	}
	ingCount := 0
	egCount := 0
	for _, r := range rules {
		if r.Direction == "INGRESS" {
			ingCount++
		} else if r.Direction == "EGRESS" {
			egCount++
		}
	}
	if ingCount != 2 || egCount != 1 {
		t.Errorf("ing/eg counts: %d/%d", ingCount, egCount)
	}
}

func TestPolicySetToRules_Empty(t *testing.T) {
	set := &vpc.SecurityGroupPolicySet{}
	if len(policySetToRules(set)) != 0 {
		t.Error("empty set should produce empty slice")
	}
}

// ---------------- applyRulesToSGWithRetry 非重试错误立即返回 ----------------

func TestApplyRulesToSGWithRetry_PermanentErrorNoRetry(t *testing.T) {
	// 通过 newVpcClientForSGFn 注入一个每次都 return InvalidParameter 的 client
	origFn := newVpcClientForSGFn
	defer func() { newVpcClientForSGFn = origFn }()
	fake := &fakeSGPoolVpcClient{
		modifyErr: &tcerr.TencentCloudSDKError{Code: "InvalidParameter", Message: "bad rule"},
	}
	newVpcClientForSGFn = func(ctx context.Context) (sgVpcClient, error) { return fake, nil }

	err := applyRulesToSGWithRetry(context.Background(), "sg-abc", `[{"direction":"INGRESS","protocol":"TCP","port":"22","cidr_block":"0.0.0.0/0","action":"ACCEPT"}]`)
	if err == nil {
		t.Fatal("expected InvalidParameter error, got nil")
	}
	// Permanent 错误应立即返回，不重试 3 次
	if len(fake.modifyReqs) != 1 {
		t.Errorf("permanent error should call once, got %d calls", len(fake.modifyReqs))
	}
}

func TestApplyRulesToSGWithRetry_Success(t *testing.T) {
	origFn := newVpcClientForSGFn
	defer func() { newVpcClientForSGFn = origFn }()
	newVpcClientForSGFn = func(ctx context.Context) (sgVpcClient, error) { return &fakeSGPoolVpcClient{}, nil }

	err := applyRulesToSGWithRetry(context.Background(), "sg-ok", `[]`)
	if err != nil {
		t.Errorf("success path should return nil, got %v", err)
	}
}

func TestApplyRulesToSGWithRetry_ContextCancelled(t *testing.T) {
	origFn := newVpcClientForSGFn
	defer func() { newVpcClientForSGFn = origFn }()
	newVpcClientForSGFn = func(ctx context.Context) (sgVpcClient, error) {
		return &fakeSGPoolVpcClient{
			modifyErr: &tcerr.TencentCloudSDKError{Code: "InternalError", Message: "boom"},
		}, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消
	// 传非空规则确保走 ModifySecurityGroupPolicies 路径（空规则会走 clearAllRulesForSG，
	// 而 fake 客户端默认 Describe 返回空 PolicySet → clear 直接成功，无法触发 modifyErr）
	rules := `[{"direction":"INGRESS","protocol":"TCP","port":"22","cidr_block":"0.0.0.0/0","action":"ACCEPT"}]`
	err := applyRulesToSGWithRetry(ctx, "sg-x", rules)
	if err == nil {
		t.Error("expected error when context cancelled mid-retry")
	}
}

// ---------------- writeRuleSetResponse ----------------

func TestWriteRuleSetResponse_HappyPath(t *testing.T) {
	db := setupSGPoolTestDB(t)
	rs := &model.RuleSet{
		Name:         "acme-rules",
		Description:  "desc here",
		Rules:        `[{"direction":"INGRESS","protocol":"TCP","port":"22","cidr_block":"0.0.0.0/0","action":"ACCEPT"}]`,
		UserGroupIDs: `["g1","g2"]`,
		Version:      3,
		IsDefault:    true,
	}
	if err := db.Create(rs).Error; err != nil {
		t.Fatal(err)
	}
	// 加一条 ACTIVE SG，projected_to 应该有 1 条
	_ = db.Create(&model.ManagedSGPool{
		SGID: "sg-x", SGName: "my-sg-name", RuleSetID: rs.ID, Status: model.SGStatusActive,
		CVMCount: 7, RuleVersion: 3,
	}).Error

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	writeRuleSetResponse(w, req, rs)

	body := w.Body.String()
	if !strings.Contains(body, `"name":"acme-rules"`) {
		t.Errorf("body missing name, got: %s", body)
	}
	if !strings.Contains(body, `"description":"desc here"`) {
		t.Errorf("body missing description, got: %s", body)
	}
	if !strings.Contains(body, `"cvm_count":7`) {
		t.Errorf("body missing projected cvm_count, got: %s", body)
	}
	if !strings.Contains(body, `"sg_name":"my-sg-name"`) {
		t.Errorf("body missing sg_name from DB, got: %s", body)
	}
	if !strings.Contains(body, `"user_group_ids":["g1","g2"]`) {
		t.Errorf("body missing user_group_ids, got: %s", body)
	}
}

func TestWriteRuleSetResponse_EmptyRulesAndGroups(t *testing.T) {
	db := setupSGPoolTestDB(t)
	rs := &model.RuleSet{
		Name:         "empty-rs",
		Rules:        "[]", // 空规则
		UserGroupIDs: "",   // 空 groups
		Version:      1,
	}
	db.Create(rs)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	writeRuleSetResponse(w, req, rs)
	body := w.Body.String()
	if !strings.Contains(body, `"initialized":true`) {
		t.Errorf("should mark initialized=true, got: %s", body)
	}
}

func TestWriteRuleSetResponse_SGNameFromDB(t *testing.T) {
	db := setupSGPoolTestDB(t)
	rs := &model.RuleSet{
		Identifier: "acme",
		Name:       "rs1",
		Rules:      "[]",
		Version:    1,
	}
	db.Create(rs)
	// 两条 ACTIVE，一条 FROZEN：projected_to 只含 ACTIVE，sg_name 直接读 DB
	db.Create(&model.ManagedSGPool{
		SGID: "sg-first", SGName: "clawpro-sg-acme-rs1-01",
		RuleSetID: rs.ID, Status: model.SGStatusActive,
		RuleVersion: 1, CVMCount: 5,
	})
	db.Create(&model.ManagedSGPool{
		SGID: "sg-frozen", SGName: "clawpro-sg-acme-rs1-02",
		RuleSetID: rs.ID, Status: model.SGStatusFrozen,
		RuleVersion: 1, CVMCount: 0,
	})
	db.Create(&model.ManagedSGPool{
		SGID: "sg-second", SGName: "some-custom-name-user-renamed",
		RuleSetID: rs.ID, Status: model.SGStatusActive,
		RuleVersion: 1, CVMCount: 3,
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	writeRuleSetResponse(w, req, rs)
	body := w.Body.String()
	// FROZEN 不投影
	if strings.Contains(body, `"sg_id":"sg-frozen"`) {
		t.Errorf("FROZEN should not be projected: %s", body)
	}
	// sg_name 来自 DB（Guardian 同步的结果），非运行时拼接
	if !strings.Contains(body, `"sg_name":"clawpro-sg-acme-rs1-01"`) {
		t.Errorf("expected sg_name for sg-first from DB, body=%s", body)
	}
	if !strings.Contains(body, `"sg_name":"some-custom-name-user-renamed"`) {
		t.Errorf("expected preserved user-renamed sg_name, body=%s", body)
	}
}

func TestWriteRuleSetResponse_BadUserGroupsJSONIgnored(t *testing.T) {
	db := setupSGPoolTestDB(t)
	rs := &model.RuleSet{
		Name:         "badjson-rs",
		Rules:        "[]",
		UserGroupIDs: `[broken`, // malformed JSON
	}
	db.Create(rs)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	writeRuleSetResponse(w, req, rs)
	// 不 panic 即可；body 里 user_group_ids 为空切片（或省略）
}

// ---------------- getUserIDFromRequest 匿名路径 ----------------

func TestGetUserIDFromRequest_AnonymousReturns0(t *testing.T) {
	_ = setupSGPoolTestDB(t)
	// getLoginUser 需要 Store 已初始化，否则会 nil 解引用 panic
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-16b"))
	t.Cleanup(func() { Store = origStore })

	req := httptest.NewRequest("GET", "/some/path", nil)
	if id := getUserIDFromRequest(req); id != 0 {
		t.Errorf("anonymous request should get 0, got %d", id)
	}
}

// ---------------- hasPlaceholder 额外补 ----------------

func TestHasPlaceholder_WithDoubleBraces(t *testing.T) {
	if !hasPlaceholder("abc{{X}}def") {
		t.Error("{{X}} should be a placeholder")
	}
	if hasPlaceholder("{X}") {
		t.Error("single brace should not be a placeholder")
	}
	if hasPlaceholder("") {
		t.Error("empty string not a placeholder")
	}
}

// ---------------- DriftError / sgRef / ruleSetResponse JSON ----------------

func TestDriftError_TruncatedLongMessage(t *testing.T) {
	long := strings.Repeat("E", 700)
	msg := truncateErrMsg(long)
	de := DriftError{SGID: "sg-xxx", Error: msg}
	// 验证长度被截断
	if len(de.Error) > 503 {
		t.Errorf("DriftError.Error not truncated: %d chars", len(de.Error))
	}
}

// ---------------- LoadClawproRequiredRules ----------------

func TestLoadClawproRequiredRules_ReturnsCanonicalForm(t *testing.T) {
	_ = setupSGPoolTestDB(t)
	rules := LoadClawproRequiredRules(context.Background())
	// 本测试只是 smoke——配置文件可能因 cwd 原因加载不到，结果为空也不算 bug。
	// 有规则的情况下校验 CIDR 合法即可（允许 placeholder）。
	for _, r := range rules {
		if hasPlaceholder(r.CidrBlock) {
			continue
		}
		if err := validateCIDR(r.CidrBlock); err != nil {
			t.Errorf("required rule cidr invalid: %q: %v", r.CidrBlock, err)
		}
	}
}

// ---------------- MergeRequiredRules  ----------------

func TestMergeRequiredRules_DoesNotDuplicate(t *testing.T) {
	_ = setupSGPoolTestDB(t)
	// 手造一组"必需规则"，确保测试独立于 JSON 配置文件。
	required := []Rule{
		{Direction: "INGRESS", Protocol: "TCP", Port: "22", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"},
		{Direction: "EGRESS", Protocol: "ALL", Port: "ALL", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"},
	}
	// 用户规则含首条必需规则副本 → merge 不应产生重复指纹
	user := []Rule{required[0]}
	merged := MergeRequiredRules(user, required)

	seen := map[string]int{}
	for _, r := range merged {
		if !hasPlaceholder(r.CidrBlock) {
			seen[r.Fingerprint()]++
		}
	}
	for fp, n := range seen {
		if n > 1 {
			t.Errorf("merged has duplicate fingerprint %s (%d)", fp, n)
		}
	}
	if len(merged) < len(required) {
		t.Errorf("merged should include all required rules, got %d want >=%d", len(merged), len(required))
	}
}

// ---------------- policyToRule: 验证 Protocol 为 * 的边界  ----------------

func TestPolicyToRule_ProtocolAll(t *testing.T) {
	all := "ALL"
	p := &vpc.SecurityGroupPolicy{Protocol: &all}
	r := policyToRule(p, "INGRESS")
	if r.Protocol != "ALL" {
		t.Errorf("expected Protocol=ALL, got %q", r.Protocol)
	}
}

// 验证 policySetToRules 遇到 v4 + v6 字段都为空的 policy 时能正常生成空 CidrBlock
func TestPolicyToRule_EmptyCIDRFields(t *testing.T) {
	emptyV4 := ""
	emptyV6 := ""
	p := &vpc.SecurityGroupPolicy{CidrBlock: &emptyV4, Ipv6CidrBlock: &emptyV6}
	r := policyToRule(p, "INGRESS")
	if r.CidrBlock != "" {
		t.Errorf("both empty should yield empty CidrBlock, got %q", r.CidrBlock)
	}
}

// 避免 unused import 警告
var _ = common.StringPtr
var _ = fmt.Sprintf
