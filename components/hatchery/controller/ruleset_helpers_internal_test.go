package controller

import (
	"context"
	"errors"
	"strings"
	"testing"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
)

// ============================================================================
// UpdateRuleSetRulesInternal / ImportRulesFromSGInternal 非云 API 路径
//
// 覆盖验证失败、RuleSet 未找到、无 ACTIVE SG 的分支——这些不需要云端调用。
// 真实 fan-out 失败 + rollback 路径需要 HTTP-level 集成测试，不在此覆盖。
// ============================================================================

func TestUpdateRuleSetRulesInternal_RuleSetNotFound(t *testing.T) {
	_ = setupSGPoolTestDB(t)
	// DB 里没有 RuleSet → GetRuleSetByName 返回 err
	_, _, _, err := UpdateRuleSetRulesInternal(context.Background(), "missing-name", nil, true)
	if err == nil {
		t.Error("expected error when rule_set not found")
	}
}

func TestUpdateRuleSetRulesInternal_NoActiveSGs_DirectDBCommit(t *testing.T) {
	db := setupSGPoolTestDB(t)
	rs := &model.RuleSet{
		Name:        model.DefaultRuleSetName,
		Description: "x", Rules: "[]", Version: 1, IsDefault: true,
	}
	db.Create(rs)
	// 无 ACTIVE SG → fan-out 无目标 → 直接 commit DB
	fakeRules := []Rule{
		{Direction: "INGRESS", Protocol: "TCP", Port: "22", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"},
	}
	version, synced, _, err := UpdateRuleSetRulesInternal(context.Background(),
		model.DefaultRuleSetName, fakeRules, true)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if version != 2 {
		t.Errorf("version should bump to 2, got %d", version)
	}
	if synced != 0 {
		t.Errorf("no ACTIVE → synced=0, got %d", synced)
	}
}

func TestImportRulesFromSGInternal_RejectClawproManagedSource(t *testing.T) {
	db := setupSGPoolTestDB(t)
	rs := &model.RuleSet{
		Name: model.DefaultRuleSetName, Rules: "[]", Version: 1, IsDefault: true,
	}
	db.Create(rs)
	// sg-in-pool 是 clawpro 托管的 → 应拒绝
	db.Create(&model.ManagedSGPool{SGID: "sg-in-pool", RuleSetID: rs.ID, Status: model.SGStatusActive})

	_, _, _, err := ImportRulesFromSGInternal(context.Background(),
		model.DefaultRuleSetName, "sg-in-pool", false)
	if err == nil {
		t.Error("should reject import from managed pool SG")
	}
}

// TestUpdateRuleSetRulesInternal_SkipMergeBranch 覆盖 SiteConfig 全关 + autoFixRules=false 的"不 merge"分支。
func TestUpdateRuleSetRulesInternal_SkipMergeBranch(t *testing.T) {
	db := setupSGPoolTestDB(t)
	rs := &model.RuleSet{
		Name: model.DefaultRuleSetName, Description: "x", Rules: "[]", Version: 1, IsDefault: true,
	}
	db.Create(rs)
	// SiteConfig 全关（默认 setupSGPoolTestDB 没注入 SiteConfig 行 → 零值即全关）
	// + autoFixRules=false → 走 skip merge 分支
	userRules := []Rule{
		{Direction: "INGRESS", Protocol: "TCP", Port: "80", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"},
	}
	version, _, _, err := UpdateRuleSetRulesInternal(context.Background(),
		model.DefaultRuleSetName, userRules, false /* autoFixRules */)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if version != 2 {
		t.Errorf("version should bump to 2, got %d", version)
	}
	// 验证落盘的 rules 严格等于用户传入（没注入 builtin）
	var saved model.RuleSet
	if err := db.First(&saved, rs.ID).Error; err != nil {
		t.Fatalf("read rs: %v", err)
	}
	// builtin 含 SSH 22；如果误注入了，会出现 22 端口规则。这里我们传的是 80 端口，落盘也应该只有 80
	if !strings.Contains(saved.Rules, `"port":"80"`) {
		t.Errorf("expected port 80 in saved rules, got: %s", saved.Rules)
	}
	if strings.Contains(saved.Rules, `"port":"22"`) {
		t.Errorf("port 22 (builtin SSH) should NOT be merged when autoFixRules=false, got: %s", saved.Rules)
	}
}

// TestImportRulesFromSGInternal_AutoCreateRuleSet 覆盖"规则组不存在 → 自动创建"分支。
// 注意：这条路径会走云 API 创建 SG，但 setupSGPoolTestDB 没注入 fake 客户端，
// 所以会失败在 createCloudSG 阶段 —— 这正好覆盖到 L405-L417 的"不存在分支"和错误处理。
func TestImportRulesFromSGInternal_AutoCreateRuleSet_FailsAtCloudCreate(t *testing.T) {
	_ = setupSGPoolTestDB(t)
	// DB 里没有 RuleSet → 走"自动创建"分支
	// 没有 fake 客户端 → newVpcClientForSGFn 默认实现会失败 → createInitialRuleSetAndSG 报错
	_, _, _, err := ImportRulesFromSGInternal(context.Background(),
		model.DefaultRuleSetName, "sg-external", false /* autoFixRules */)
	if err == nil {
		t.Error("expected error when no real cloud client and rule_set must be created")
	}
}

// TestSiteConfigRequiresRecommendedRules 覆盖三种 SiteConfig 状态的判断。
func TestSiteConfigRequiresRecommendedRules(t *testing.T) {
	db := setupSGPoolTestDB(t)
	t.Run("全关", func(t *testing.T) {
		// SiteConfig 表已建但无数据 → GetSiteConfig 返回零值 → 全关
		if siteConfigRequiresRecommendedRules(context.Background()) {
			t.Error("empty SiteConfig should return false")
		}
	})
	t.Run("启用GatewayUI", func(t *testing.T) {
		db.Where("1=1").Delete(&model.SiteConfig{})
		db.Create(&model.SiteConfig{GatewayUIEnable: true, GatewayUIPort: 1080})
		if !siteConfigRequiresRecommendedRules(context.Background()) {
			t.Error("GatewayUIEnable+Port>0 should return true")
		}
	})
	t.Run("GatewayUI启用但端口0", func(t *testing.T) {
		db.Where("1=1").Delete(&model.SiteConfig{})
		db.Create(&model.SiteConfig{GatewayUIEnable: true, GatewayUIPort: 0})
		if siteConfigRequiresRecommendedRules(context.Background()) {
			t.Error("GatewayUIEnable but Port=0 should return false")
		}
	})
	t.Run("GatewayUI启用但addr_type=private", func(t *testing.T) {
		// 私网模式下用户走 VPC 内网通道访问 Gateway UI，不需要在 SG 上对公网放通端口。
		// 应等价于"未启用 recommended"，避免 import/merge 路径强制注入 0.0.0.0/0:port 规则。
		db.Where("1=1").Delete(&model.SiteConfig{})
		db.Create(&model.SiteConfig{
			GatewayUIEnable:   true,
			GatewayUIPort:     1080,
			GatewayUIAddrType: "private",
		})
		if siteConfigRequiresRecommendedRules(context.Background()) {
			t.Error("GatewayUIAddrType=private should return false (no SG injection needed)")
		}
	})
	t.Run("GatewayUI启用且addr_type=public", func(t *testing.T) {
		// public 模式下需要在 SG 上对公网放通端口，应正常报告"需要 recommended"。
		db.Where("1=1").Delete(&model.SiteConfig{})
		db.Create(&model.SiteConfig{
			GatewayUIEnable:   true,
			GatewayUIPort:     1080,
			GatewayUIAddrType: "public",
		})
		if !siteConfigRequiresRecommendedRules(context.Background()) {
			t.Error("GatewayUIAddrType=public should return true")
		}
	})
	t.Run("启用BrowserVNC", func(t *testing.T) {
		db.Where("1=1").Delete(&model.SiteConfig{})
		db.Create(&model.SiteConfig{BrowserVNCEnable: true})
		if !siteConfigRequiresRecommendedRules(context.Background()) {
			t.Error("BrowserVNCEnable should return true")
		}
	})
}

// ============================================================================
// ImportRulesFromSGInternal 完整路径覆盖（含 fake 云客户端）
// ============================================================================

// TestImportRulesFromSGInternal_DescribeSourceFails 覆盖"读取源 SG 规则失败"分支。
func TestImportRulesFromSGInternal_DescribeSourceFails(t *testing.T) {
	_ = setupSGPoolTestDB(t)
	fake := &fakeSGPoolVpcClient{
		describePoliciesErr: errors.New("source sg not found"),
	}
	restore := withFakeSGPoolVpcClient(fake)
	defer restore()

	_, _, _, err := ImportRulesFromSGInternal(context.Background(),
		model.DefaultRuleSetName, "sg-external", false)
	wanted := hcommon.I18nError(i18n.MsgImportDescribeSGPoliciesFail)
	if err == nil || !errors.Is(err, wanted) {
		t.Errorf("expected describe error, got %v", err)
	}
}

// TestImportRulesFromSGInternal_NilSourcePolicySet 覆盖"源 SG 规则为空"返回错误的分支。
func TestImportRulesFromSGInternal_NilSourcePolicySet(t *testing.T) {
	_ = setupSGPoolTestDB(t)
	fake := &fakeSGPoolVpcClient{
		describePoliciesResp: &vpc.DescribeSecurityGroupPoliciesResponse{
			Response: &vpc.DescribeSecurityGroupPoliciesResponseParams{
				// SecurityGroupPolicySet 为 nil
			},
		},
	}
	restore := withFakeSGPoolVpcClient(fake)
	defer restore()

	_, _, _, err := ImportRulesFromSGInternal(context.Background(),
		model.DefaultRuleSetName, "sg-external", false)
	if err == nil || !strings.Contains(err.Error(), "源安全组规则为空") {
		t.Errorf("expected empty source error, got %v", err)
	}
}

// TestImportRulesFromSGInternal_AutoCreate_Success 覆盖"规则组不存在 → 自动创建成功"完整路径。
// 验证 createInitialRuleSetAndSG 被调用、新 RuleSet 落盘、autoFixRules 透传正确。
func TestImportRulesFromSGInternal_AutoCreate_Success(t *testing.T) {
	db := setupSGPoolTestDB(t)
	idx0 := int64(0)
	fake := &fakeSGPoolVpcClient{
		// 源 SG 含一条规则
		describePoliciesResp: &vpc.DescribeSecurityGroupPoliciesResponse{
			Response: &vpc.DescribeSecurityGroupPoliciesResponseParams{
				SecurityGroupPolicySet: &vpc.SecurityGroupPolicySet{
					Ingress: []*vpc.SecurityGroupPolicy{
						{
							PolicyIndex: &idx0,
							Protocol:    common.StringPtr("TCP"),
							Port:        common.StringPtr("80"),
							CidrBlock:   common.StringPtr("0.0.0.0/0"),
							Action:      common.StringPtr("ACCEPT"),
						},
					},
				},
			},
		},
		// 自动建组需要的云 API
		createResp: &vpc.CreateSecurityGroupResponse{
			Response: &vpc.CreateSecurityGroupResponseParams{
				SecurityGroup: &vpc.SecurityGroup{SecurityGroupId: common.StringPtr("sg-new")},
			},
		},
	}
	restore := withFakeSGPoolVpcClient(fake)
	defer restore()

	// SiteConfig 全关 + autoFixRules=false → 不 merge，userRules 原样落盘
	version, synced, _, err := ImportRulesFromSGInternal(context.Background(),
		model.DefaultRuleSetName, "sg-external", false)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if version != 1 || synced != 1 {
		t.Errorf("expected version=1 synced=1, got version=%d synced=%d", version, synced)
	}
	// 验证 RuleSet 落盘
	var rs model.RuleSet
	if err := db.Where("name = ?", model.DefaultRuleSetName).First(&rs).Error; err != nil {
		t.Fatalf("rule_set should be created: %v", err)
	}
	// SiteConfig 全关 + autoFixRules=false → 应该只有源 SG 的 80 端口，没注入 SSH 22
	if !strings.Contains(rs.Rules, `"port":"80"`) {
		t.Errorf("expected port 80 in saved rules, got: %s", rs.Rules)
	}
	if strings.Contains(rs.Rules, `"port":"22"`) {
		t.Errorf("port 22 (builtin SSH) should NOT be merged when autoFixRules=false and SiteConfig disabled, got: %s", rs.Rules)
	}
}

// TestImportRulesFromSGInternal_AutoCreate_AutoFixRulesTrue 验证 autoFixRules=true 时
// 即使 SiteConfig 全关也会注入 builtin 规则。
func TestImportRulesFromSGInternal_AutoCreate_AutoFixRulesTrue(t *testing.T) {
	db := setupSGPoolTestDB(t)
	// 注入 builtin SSH 规则配置（替代磁盘文件）
	origJSON := ClawproRequiredSGRulesJSON
	defer func() { ClawproRequiredSGRulesJSON = origJSON }()
	ClawproRequiredSGRulesJSON = []byte(`{
		"categories": [{"type":"builtin","label":"内置","rule_groups":[
			{"key":"allow_ssh","name":"SSH","rules":[
				{"direction":"ingress","protocol":"TCP","port":"22","cidr_block":"0.0.0.0/0","action":"ACCEPT","description":"SSH"}
			]}
		]}]
	}`)

	idx0 := int64(0)
	fake := &fakeSGPoolVpcClient{
		describePoliciesResp: &vpc.DescribeSecurityGroupPoliciesResponse{
			Response: &vpc.DescribeSecurityGroupPoliciesResponseParams{
				SecurityGroupPolicySet: &vpc.SecurityGroupPolicySet{
					Ingress: []*vpc.SecurityGroupPolicy{
						{PolicyIndex: &idx0, Protocol: common.StringPtr("TCP"),
							Port: common.StringPtr("80"), CidrBlock: common.StringPtr("0.0.0.0/0"),
							Action: common.StringPtr("ACCEPT")},
					},
				},
			},
		},
		createResp: &vpc.CreateSecurityGroupResponse{
			Response: &vpc.CreateSecurityGroupResponseParams{
				SecurityGroup: &vpc.SecurityGroup{SecurityGroupId: common.StringPtr("sg-new")},
			},
		},
	}
	restore := withFakeSGPoolVpcClient(fake)
	defer restore()

	_, _, _, err := ImportRulesFromSGInternal(context.Background(),
		model.DefaultRuleSetName, "sg-external", true /* autoFixRules */)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	var rs model.RuleSet
	if err := db.Where("name = ?", model.DefaultRuleSetName).First(&rs).Error; err != nil {
		t.Fatalf("rule_set should be created: %v", err)
	}
	// autoFixRules=true → builtin 必需规则被注入（应包含 SSH 22）
	if !strings.Contains(rs.Rules, `"port":"22"`) {
		t.Errorf("port 22 (builtin SSH) should be merged when autoFixRules=true, got: %s", rs.Rules)
	}
	// 源 SG 的 80 也应该保留
	if !strings.Contains(rs.Rules, `"port":"80"`) {
		t.Errorf("source rule (port 80) should be preserved, got: %s", rs.Rules)
	}
}

// TestImportRulesFromSGInternal_RuleSetExists_DelegatesToUpdate 验证"规则组已存在"分支
// 走 UpdateRuleSetRulesInternal，autoFixRules 透传正确。
func TestImportRulesFromSGInternal_RuleSetExists_DelegatesToUpdate(t *testing.T) {
	db := setupSGPoolTestDB(t)
	// 预先建好 RuleSet
	rs := &model.RuleSet{
		Name: model.DefaultRuleSetName, Description: "x", Rules: "[]", Version: 1, IsDefault: true,
	}
	db.Create(rs)
	idx0 := int64(0)
	fake := &fakeSGPoolVpcClient{
		describePoliciesResp: &vpc.DescribeSecurityGroupPoliciesResponse{
			Response: &vpc.DescribeSecurityGroupPoliciesResponseParams{
				SecurityGroupPolicySet: &vpc.SecurityGroupPolicySet{
					Ingress: []*vpc.SecurityGroupPolicy{
						{PolicyIndex: &idx0, Protocol: common.StringPtr("TCP"),
							Port: common.StringPtr("443"), CidrBlock: common.StringPtr("0.0.0.0/0"),
							Action: common.StringPtr("ACCEPT")},
					},
				},
			},
		},
	}
	restore := withFakeSGPoolVpcClient(fake)
	defer restore()

	// SiteConfig 全关 + autoFixRules=false → 走 update 路径，不 merge
	version, _, _, err := ImportRulesFromSGInternal(context.Background(),
		model.DefaultRuleSetName, "sg-external", false)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if version != 2 {
		t.Errorf("version should bump to 2, got %d", version)
	}
	// 验证 RuleSet 已更新为源 SG 规则（不含 SSH 22）
	var saved model.RuleSet
	db.First(&saved, rs.ID)
	if !strings.Contains(saved.Rules, `"port":"443"`) {
		t.Errorf("expected port 443 from import, got: %s", saved.Rules)
	}
	if strings.Contains(saved.Rules, `"port":"22"`) {
		t.Errorf("port 22 should NOT be merged when autoFixRules=false, got: %s", saved.Rules)
	}
}
