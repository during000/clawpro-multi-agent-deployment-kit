package controller

import (
	"context"
	"strings"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
	"gorm.io/gorm"
)

// Rule.Fingerprint 的指纹归一化与稳定性测试。

func TestRuleFingerprint_Normalization(t *testing.T) {
	cases := []struct {
		name string
		a, b Rule
		same bool
	}{
		{
			name: "direction_case_ingress",
			a:    Rule{Direction: "ingress", Protocol: "TCP", Port: "22", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"},
			b:    Rule{Direction: "INGRESS", Protocol: "TCP", Port: "22", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"},
			same: true,
		},
		{
			name: "direction_alias_in_egress",
			a:    Rule{Direction: "IN", Protocol: "TCP", Port: "22", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"},
			b:    Rule{Direction: "INGRESS", Protocol: "TCP", Port: "22", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"},
			same: true,
		},
		{
			name: "port_22_vs_22-22",
			a:    Rule{Direction: "INGRESS", Protocol: "TCP", Port: "22-22", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"},
			b:    Rule{Direction: "INGRESS", Protocol: "TCP", Port: "22", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"},
			same: true,
		},
		{
			name: "port_ALL_vs_empty",
			a:    Rule{Direction: "INGRESS", Protocol: "TCP", Port: "", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"},
			b:    Rule{Direction: "INGRESS", Protocol: "TCP", Port: "ALL", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"},
			same: true,
		},
		{
			name: "cidr_ipv4_no_prefix",
			a:    Rule{Direction: "INGRESS", Protocol: "TCP", Port: "22", CidrBlock: "10.0.0.1", Action: "ACCEPT"},
			b:    Rule{Direction: "INGRESS", Protocol: "TCP", Port: "22", CidrBlock: "10.0.0.1/32", Action: "ACCEPT"},
			same: true,
		},
		{
			name: "cidr_ipv6_case",
			a:    Rule{Direction: "INGRESS", Protocol: "TCP", Port: "22", CidrBlock: "::1", Action: "ACCEPT"},
			b:    Rule{Direction: "INGRESS", Protocol: "TCP", Port: "22", CidrBlock: "::1/128", Action: "ACCEPT"},
			same: true,
		},
		{
			name: "ingress_vs_egress_different",
			a:    Rule{Direction: "INGRESS", Protocol: "TCP", Port: "22", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"},
			b:    Rule{Direction: "EGRESS", Protocol: "TCP", Port: "22", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"},
			same: false,
		},
		{
			name: "action_accept_vs_drop_different",
			a:    Rule{Direction: "INGRESS", Protocol: "TCP", Port: "22", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"},
			b:    Rule{Direction: "INGRESS", Protocol: "TCP", Port: "22", CidrBlock: "0.0.0.0/0", Action: "DROP"},
			same: false,
		},
		{
			name: "description_not_in_fingerprint",
			a:    Rule{Direction: "INGRESS", Protocol: "TCP", Port: "22", CidrBlock: "0.0.0.0/0", Action: "ACCEPT", PolicyDescription: "a"},
			b:    Rule{Direction: "INGRESS", Protocol: "TCP", Port: "22", CidrBlock: "0.0.0.0/0", Action: "ACCEPT", PolicyDescription: "b"},
			same: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := (tc.a.Fingerprint() == tc.b.Fingerprint())
			if got != tc.same {
				t.Errorf("expected same=%v, got a=%q b=%q", tc.same, tc.a.Fingerprint(), tc.b.Fingerprint())
			}
		})
	}
}

// MergeRequiredRules：必需规则优先，冲突时覆盖 Description；保留用户规则原序，仅必需的追加在末尾。
func TestMergeRequiredRules(t *testing.T) {
	user := []Rule{
		{Direction: "EGRESS", Protocol: "ALL", Port: "ALL", CidrBlock: "0.0.0.0/0", Action: "ACCEPT", PolicyDescription: "user egress"},
		{Direction: "INGRESS", Protocol: "TCP", Port: "443", CidrBlock: "2.2.2.2/32", Action: "ACCEPT", PolicyDescription: "stale prepend"},
		{Direction: "INGRESS", Protocol: "TCP", Port: "22", CidrBlock: "1.1.1.1/32", Action: "ACCEPT", PolicyDescription: "user ssh"},
	}
	required := []Rule{
		{Direction: "INGRESS", Protocol: "TCP", Port: "443", CidrBlock: "2.2.2.2/32", Action: "ACCEPT", PolicyDescription: "REQUIRED prepend", Prepend: true},
		{Direction: "INGRESS", Protocol: "TCP", Port: "22", CidrBlock: "1.1.1.1/32", Action: "ACCEPT", PolicyDescription: "REQUIRED ssh"},
		{Direction: "INGRESS", Protocol: "TCP", Port: "8080", CidrBlock: "0.0.0.0/0", Action: "ACCEPT", PolicyDescription: "REQUIRED api"},
	}
	merged := MergeRequiredRules(user, required)
	if len(merged) != 4 {
		t.Fatalf("expected 4 rules after merge, got %d", len(merged))
	}
	if merged[0].Port != "443" || merged[0].PolicyDescription != "REQUIRED prepend" || !merged[0].IsRequired {
		t.Errorf("merged[0] should be prepended required port 443, got port=%q desc=%q required=%v",
			merged[0].Port, merged[0].PolicyDescription, merged[0].IsRequired)
	}
	if merged[1].Direction != "EGRESS" || merged[1].PolicyDescription != "user egress" || merged[1].IsRequired {
		t.Errorf("merged[1] should be true user rule, got direction=%q desc=%q required=%v",
			merged[1].Direction, merged[1].PolicyDescription, merged[1].IsRequired)
	}
	if merged[2].Port != "22" || merged[2].PolicyDescription != "REQUIRED ssh" || !merged[2].IsRequired {
		t.Errorf("merged[2] should be non-prepend required port 22 at user position, got port=%q desc=%q required=%v",
			merged[2].Port, merged[2].PolicyDescription, merged[2].IsRequired)
	}
	if merged[3].Port != "8080" || merged[3].PolicyDescription != "REQUIRED api" || !merged[3].IsRequired {
		t.Errorf("merged[3] should be missing non-prepend required port 8080, got port=%q desc=%q required=%v",
			merged[3].Port, merged[3].PolicyDescription, merged[3].IsRequired)
	}
}

// hasPlaceholder 保险丝测试：识别 `{{...}}` 未替换占位符。
func TestHasPlaceholder(t *testing.T) {
	cases := map[string]bool{
		"":                    false,
		"22":                  false,
		"TCP":                 false,
		"10.0.0.0/16":         false,
		"{{GATEWAY_UI_PORT}}": true,
		"{{VPC_CIDR}}":        true,
		"prefix{{X}}suffix":   true,
	}
	for in, want := range cases {
		if got := hasPlaceholder(in); got != want {
			t.Errorf("hasPlaceholder(%q) = %v, want %v", in, got, want)
		}
	}
}

// TestLoadClawproRequiredRules_NoPlaceholdersLeak 覆盖生产环境 sg-init 崩溃的场景：
// config/clawpro_required_sg_rules.json 里带 `{{GATEWAY_UI_PORT}}` / `{{VPC_CIDR}}` 占位符；
// LoadClawproRequiredRules 必须先做条件解析和占位符替换，避免把
//
//	{"port": "{{GATEWAY_UI_PORT}}"}
//
// 原样发给腾讯云 API 触发 InvalidParameterValue。
//
// 本测试不依赖真实 VPC 查询（resolveVpcCidr 返回空时直接保留占位符），只保证最终展开的
// []Rule 里没有任何字段残留 `{{...}}`。
//
// 需要一个空的 site_configs 行让 resolveConditionalRules/resolveVpcCidr 不 panic。
// TestRuleToPolicy_IPv6GoesToIpv6CidrBlock 覆盖生产 sg-init 第二次崩溃：
//
//	参数 `.SecurityGroupPolicySet.Egress.1.CidrBlock` 值 `::/0` 是无效的。
//
// 腾讯云 VPC SDK 的 SecurityGroupPolicy 有独立的 CidrBlock / Ipv6CidrBlock 字段；
// IPv6 CIDR（含冒号）塞到 CidrBlock 会被拒。ruleToPolicy 必须按冒号分流。
func TestRuleToPolicy_IPv6GoesToIpv6CidrBlock(t *testing.T) {
	ipv6 := Rule{Direction: "EGRESS", Protocol: "ALL", Port: "ALL", CidrBlock: "::/0", Action: "ACCEPT"}
	p := ruleToPolicy(&ipv6)
	if p.CidrBlock != nil {
		t.Errorf("IPv6 rule should not populate CidrBlock; got %q", *p.CidrBlock)
	}
	if p.Ipv6CidrBlock == nil || *p.Ipv6CidrBlock != "::/0" {
		t.Errorf("IPv6 rule should populate Ipv6CidrBlock=::/0; got %+v", p.Ipv6CidrBlock)
	}

	ipv4 := Rule{Direction: "INGRESS", Protocol: "TCP", Port: "22", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"}
	p = ruleToPolicy(&ipv4)
	if p.Ipv6CidrBlock != nil {
		t.Errorf("IPv4 rule should not populate Ipv6CidrBlock; got %q", *p.Ipv6CidrBlock)
	}
	if p.CidrBlock == nil || *p.CidrBlock != "0.0.0.0/0" {
		t.Errorf("IPv4 rule should populate CidrBlock=0.0.0.0/0; got %+v", p.CidrBlock)
	}
}

func TestLoadClawproRequiredRules_NoPlaceholdersLeak(t *testing.T) {
	origJSON := ClawproRequiredSGRulesJSON
	defer func() {
		ClawproRequiredSGRulesJSON = origJSON
	}()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.SiteConfig{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// 默认 SiteConfig：GatewayUIEnable=false / BrowserVNCEnable=false / VpcId=""
	// 这对应刚部署、gateway_ui 关闭的场景——正是用户现场的崩溃条件。
	if err := db.Create(&model.SiteConfig{}).Error; err != nil {
		t.Fatalf("create site_config: %v", err)
	}
	restoreDB := model.UseDBForTest(db)
	defer restoreDB()

	// 内联 production 配置的相关片段（含占位符），绕过文件系统依赖
	ClawproRequiredSGRulesJSON = []byte(`{
	  "categories": [
	    {"type":"builtin","label":"内置","rule_groups":[
	      {"key":"restrict_vpc","name":"vpc","rules":[
	        {"direction":"ingress","protocol":"ALL","port":"ALL","cidr_block":"{{VPC_CIDR}}","ipv6_cidr":"","action":"DROP","description":"限 VPC 互访"}
	      ]},
	      {"key":"allow_ssh","name":"ssh","rules":[
	        {"direction":"ingress","protocol":"TCP","port":"22","cidr_block":"0.0.0.0/0","ipv6_cidr":"","action":"ACCEPT","description":"SSH v4"}
	      ]}
	    ]},
	    {"type":"recommended","label":"推荐","rule_groups":[
	      {"key":"gw","name":"gw","condition":"gateway_ui_enable","rules":[
	        {"direction":"ingress","protocol":"TCP","port":"{{GATEWAY_UI_PORT}}","cidr_block":"0.0.0.0/0","ipv6_cidr":"","action":"ACCEPT","description":"gw"}
	      ]},
	      {"key":"vnc","name":"vnc","condition":"browser_vnc_enable","rules":[
	        {"direction":"ingress","protocol":"TCP","port":"6080","cidr_block":"1.2.3.4/32","ipv6_cidr":"","action":"ACCEPT","description":"vnc"}
	      ]}
	    ]}
	  ]
	}`)

	rules := LoadClawproRequiredRules(context.Background())
	if len(rules) == 0 {
		t.Fatal("expected some rules (ssh), got 0")
	}
	for i, r := range rules {
		for name, field := range map[string]string{
			"Direction":         r.Direction,
			"Protocol":          r.Protocol,
			"Port":              r.Port,
			"CidrBlock":         r.CidrBlock,
			"Action":            r.Action,
			"PolicyDescription": r.PolicyDescription,
		} {
			if strings.Contains(field, "{{") || strings.Contains(field, "}}") {
				t.Errorf("rule[%d].%s contains unresolved placeholder: %q (full rule: %+v)", i, name, field, r)
			}
		}
	}
	// 额外断言：SSH 规则应在（无条件保留）；GATEWAY_UI_PORT 规则应被整组剔除（condition 关闭）；
	// VPC_CIDR 规则应被保险丝过滤掉（VpcId 为空 → resolveVpcCidr="" → 占位符未替换 → hasPlaceholder 拦截）
	var sawSSH, sawGW, sawVPCDrop bool
	for _, r := range rules {
		if r.Port == "22" {
			sawSSH = true
		}
		if r.PolicyDescription == "gw" {
			sawGW = true
		}
		if r.Action == "DROP" && r.Protocol == "ALL" {
			sawVPCDrop = true
		}
	}
	if !sawSSH {
		t.Error("SSH rule (port 22) should survive")
	}
	if sawGW {
		t.Error("gateway_ui rule should be filtered out when gateway_ui_enable=false")
	}
	if sawVPCDrop {
		t.Error("VPC_CIDR rule should be filtered by placeholder guard when VpcId is empty")
	}
}

// ============================================================================
// Change: sg-rule-support-sg-and-address-template
// ----------------------------------------------------------------------------
// 覆盖 Rule.CidrBlock 字段语义扩展：允许存 sg-xxx / ipm-xxx / ipmg-xxx
// 新增函数：classifyRuleSource / isValidRuleSource
// 升级函数：normalizeCIDR / policyToRule / policyToRuleSkippable
// ============================================================================

// TestClassifyRuleSource 验证前缀分类的所有分支。
// ⚠️ 关键回归：ipmg- 必须先于 ipm- 匹配（字符串前缀包含关系，顺序错了会导致
// "ipmg-abc" 被错误分类为 srcAddressTpl 而非 srcAddressGroup）。
func TestClassifyRuleSource(t *testing.T) {
	cases := []struct {
		in   string
		want sourceKind
	}{
		// 空值 / 未知
		{"", srcUnknown},
		{"   ", srcUnknown},

		// IPv4 CIDR（默认分支）
		{"0.0.0.0/0", srcIPv4CIDR},
		{"10.0.0.0/8", srcIPv4CIDR},
		{"1.2.3.4/32", srcIPv4CIDR},
		{"1.2.3.4", srcIPv4CIDR}, // 裸 IP 也归 IPv4（normalizeCIDR 会补 /32）

		// IPv6 CIDR
		{"::/0", srcIPv6CIDR},
		{"2001:db8::/64", srcIPv6CIDR},
		{"fe80::1", srcIPv6CIDR}, // 裸 IPv6

		// 安全组引用
		{"sg-2f25udyn", srcSG},
		{"sg-abcdef12", srcSG},

		// IP 地址模板组（⚠️ 必须优先于 ipm- 匹配）
		{"ipmg-abcdef12", srcAddressGroup},
		{"ipmg-xxxx", srcAddressGroup},

		// IP 地址模板
		{"ipm-aw8st7ni", srcAddressTpl},
		{"ipm-xxxx", srcAddressTpl},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got := classifyRuleSource(c.in)
			if got != c.want {
				t.Errorf("classifyRuleSource(%q) = %d; want %d", c.in, got, c.want)
			}
		})
	}
}

// TestIsValidRuleSource 合法/非法来源字符串校验（注：当前 apply 阶段不在写入路径调用此函数，
// 保留为公共工具便于后续需要时启用。见 tasks.md 任务 5 架构澄清）。
func TestIsValidRuleSource(t *testing.T) {
	valid := []string{
		"0.0.0.0/0", "10.0.0.0/8", "1.2.3.4/32", "1.2.3.4",
		"::/0", "2001:db8::/64", "fe80::1",
		"sg-2f25udyn", "ipm-aw8st7ni", "ipmg-abcdef12",
	}
	for _, s := range valid {
		if !isValidRuleSource(s) {
			t.Errorf("isValidRuleSource(%q) = false; want true", s)
		}
	}
	invalid := []string{
		"", "   ",
		"—",         // 前端 "—" bug 触发的脏数据
		"invalid",   // 既不是 CIDR 也不是已知前缀
		"sg-",       // 前缀后无 ID
		"ipm-",      // 前缀后无 ID
		"ipmg-",     // 前缀后无 ID
		"256.0.0.0", // 非法 IPv4
		"not.a.cidr",
	}
	for _, s := range invalid {
		if isValidRuleSource(s) {
			t.Errorf("isValidRuleSource(%q) = true; want false", s)
		}
	}
}

// TestNormalizeCIDR_NonCIDRPrefixPassThrough 验证 normalizeCIDR 对三类资源标识前缀原样返回，
// 不走 net.ParseCIDR/ParseIP。这是 Fingerprint 正确的关键：如果走了 IP 解析分支，
// "sg-xxx" 最终返回值虽然等于原串（ParseCIDR/ParseIP 都 fail 的 fallback），
// 但依赖"失败分支"不明确，本测试锁定显式前缀分支行为。
func TestNormalizeCIDR_NonCIDRPrefixPassThrough(t *testing.T) {
	cases := []string{
		"sg-2f25udyn",
		"sg-abcdef12",
		"ipm-aw8st7ni",
		"ipmg-abcdef12",
	}
	for _, s := range cases {
		if got := normalizeCIDR(s); got != s {
			t.Errorf("normalizeCIDR(%q) = %q; want %q (pass through)", s, got, s)
		}
	}
}

// TestFingerprint_NonCIDRSourceStable 验证含非 CIDR 来源的 Rule 生成稳定指纹。
// Guardian drift 检测依赖两侧指纹严格相等，本测试锁定指纹字符串形态。
func TestFingerprint_NonCIDRSourceStable(t *testing.T) {
	cases := []struct {
		name string
		r    Rule
		want string
	}{
		{
			"sg reference",
			Rule{Direction: "INGRESS", Protocol: "ALL", Port: "ALL", CidrBlock: "sg-2f25udyn", Action: "ACCEPT"},
			"INGRESS|ALL|ALL|sg-2f25udyn|ACCEPT",
		},
		{
			"address template",
			Rule{Direction: "INGRESS", Protocol: "ALL", Port: "ALL", CidrBlock: "ipm-aw8st7ni", Action: "ACCEPT"},
			"INGRESS|ALL|ALL|ipm-aw8st7ni|ACCEPT",
		},
		{
			"address group",
			Rule{Direction: "EGRESS", Protocol: "TCP", Port: "443", CidrBlock: "ipmg-abcdef12", Action: "DROP"},
			"EGRESS|TCP|443|ipmg-abcdef12|DROP",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.r.Fingerprint(); got != c.want {
				t.Errorf("Fingerprint() = %q; want %q", got, c.want)
			}
		})
	}
}

// TestPolicyToRule_SecurityGroupIdPriority 验证 SDK policy 只含 SecurityGroupId 时，
// 反向转换写入 Rule.CidrBlock=sg-xxx（不再 skip）。
func TestPolicyToRule_SecurityGroupIdPriority(t *testing.T) {
	sgid := "sg-2f25udyn"
	all := "ALL"
	accept := "ACCEPT"
	p := &vpc.SecurityGroupPolicy{
		Protocol:        &all,
		Port:            &all,
		SecurityGroupId: &sgid,
		Action:          &accept,
	}
	r, ok := policyToRuleSkippable(p, "INGRESS")
	if !ok {
		t.Fatal("policyToRuleSkippable returned ok=false for SG-reference rule; expected true")
	}
	if r.CidrBlock != sgid {
		t.Errorf("CidrBlock = %q; want %q", r.CidrBlock, sgid)
	}
	if r.Direction != "INGRESS" || r.Protocol != "ALL" || r.Action != "ACCEPT" {
		t.Errorf("other fields not populated correctly: %+v", r)
	}
}

// TestPolicyToRule_AddressTemplatePaths 验证 AddressTemplate 的两个子字段（AddressId / AddressGroupId）
// 各自单独填充时都被正确识别并写入 Rule.CidrBlock。
func TestPolicyToRule_AddressTemplatePaths(t *testing.T) {
	t.Run("AddressId only", func(t *testing.T) {
		addrID := "ipm-aw8st7ni"
		p := &vpc.SecurityGroupPolicy{
			AddressTemplate: &vpc.AddressTemplateSpecification{AddressId: &addrID},
		}
		r, ok := policyToRuleSkippable(p, "INGRESS")
		if !ok || r.CidrBlock != addrID {
			t.Errorf("got CidrBlock=%q ok=%v; want %q true", r.CidrBlock, ok, addrID)
		}
	})
	t.Run("AddressGroupId only", func(t *testing.T) {
		groupID := "ipmg-abcdef12"
		p := &vpc.SecurityGroupPolicy{
			AddressTemplate: &vpc.AddressTemplateSpecification{AddressGroupId: &groupID},
		}
		r, ok := policyToRuleSkippable(p, "EGRESS")
		if !ok || r.CidrBlock != groupID {
			t.Errorf("got CidrBlock=%q ok=%v; want %q true", r.CidrBlock, ok, groupID)
		}
	})
}

// TestPolicyToRule_PriorityWhenMultipleSet 验证多个来源字段同时非空（异常输入）时，
// 按 D4 优先级链取第一个：SecurityGroupId > AddressGroupId > AddressId > Ipv6CidrBlock > CidrBlock。
//
// 这种输入理论不会出现（腾讯云 API 层面互斥），但反向链条必须有明确优先级作安全降级，
// 避免不同版本/场景下行为漂移。
func TestPolicyToRule_PriorityWhenMultipleSet(t *testing.T) {
	sgid := "sg-highest"
	groupID := "ipmg-second"
	addrID := "ipm-third"
	ipv6 := "::/0"
	ipv4 := "0.0.0.0/0"
	p := &vpc.SecurityGroupPolicy{
		SecurityGroupId: &sgid,
		AddressTemplate: &vpc.AddressTemplateSpecification{
			AddressId:      &addrID,
			AddressGroupId: &groupID,
		},
		Ipv6CidrBlock: &ipv6,
		CidrBlock:     &ipv4,
	}
	r, ok := policyToRuleSkippable(p, "INGRESS")
	if !ok || r.CidrBlock != sgid {
		t.Errorf("SecurityGroupId has highest priority; got CidrBlock=%q ok=%v; want %q true", r.CidrBlock, ok, sgid)
	}
}

// TestPolicyToRuleSkippable_AllEmptyStillSkipped 验证所有来源字段都空时仍然 skip
// （仅 Protocol 非空的残缺规则，防御性保留）。
func TestPolicyToRuleSkippable_AllEmptyStillSkipped(t *testing.T) {
	tcp := "TCP"
	port := "22"
	empty := ""
	p := &vpc.SecurityGroupPolicy{
		Protocol:  &tcp,
		Port:      &port,
		CidrBlock: &empty, // 所有来源字段都空
	}
	_, ok := policyToRuleSkippable(p, "INGRESS")
	if ok {
		t.Error("all source fields empty should skip (return ok=false)")
	}
}

// TestPolicyToRuleSkippable_SGRefNotSkipped 回归：在本 change 之前的版本，
// 这种"仅 SecurityGroupId 非空"的规则会被 skip。升级后必须不再 skip。
func TestPolicyToRuleSkippable_SGRefNotSkipped(t *testing.T) {
	sgid := "sg-2f25udyn"
	p := &vpc.SecurityGroupPolicy{SecurityGroupId: &sgid}
	_, ok := policyToRuleSkippable(p, "INGRESS")
	if !ok {
		t.Error("SG-reference rule must NOT be skipped after this change")
	}
}
