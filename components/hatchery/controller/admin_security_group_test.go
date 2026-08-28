package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
	"gorm.io/gorm"
)

// init 在测试构建中将安全组 handler 后台 goroutine 使用的 listInstanceIdsFn
// 替换为空实现。生产代码里该函数会查询 model.DB(context.Background()) 的 instances 表；测试里
// handler 发起的异步换绑 goroutine 会泄漏到下一个测试，若下一个测试替换了
// model.DB(context.Background())(不同的表集合)，会造成 SQLite 并发错误导致 flaky。
// 此处默认空实现让异步 goroutine 立刻 return，单个测试如需覆盖此行为可再
// 临时替换。不影响生产行为（仅在 *_test.go 的构建单元中生效）。
func init() {
	listInstanceIdsFn = func(ctx context.Context) ([]string, error) { return nil, nil }
}

// ==================== sgPolicyMatchesRule 纯函数测试 ====================

func TestSgPolicyMatchesRule_ExactMatch(t *testing.T) {
	policy := &vpc.SecurityGroupPolicy{
		Protocol:  common.StringPtr("TCP"),
		Port:      common.StringPtr("22"),
		CidrBlock: common.StringPtr("0.0.0.0/0"),
		Action:    common.StringPtr("ACCEPT"),
	}
	rule := requiredSGRule{
		Direction: "ingress",
		Protocol:  "TCP",
		Port:      "22",
		CidrBlock: "0.0.0.0/0",
		Action:    "ACCEPT",
	}
	if !sgPolicyMatchesRule(policy, rule) {
		t.Error("完全匹配的规则应返回 true")
	}
}

func TestSgPolicyMatchesRule_NilPolicy(t *testing.T) {
	rule := requiredSGRule{Protocol: "TCP", Port: "22", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"}
	if sgPolicyMatchesRule(nil, rule) {
		t.Error("nil policy 应返回 false")
	}
}

func TestSgPolicyMatchesRule_ProtocolMismatch(t *testing.T) {
	policy := &vpc.SecurityGroupPolicy{
		Protocol:  common.StringPtr("UDP"),
		Port:      common.StringPtr("22"),
		CidrBlock: common.StringPtr("0.0.0.0/0"),
		Action:    common.StringPtr("ACCEPT"),
	}
	rule := requiredSGRule{Protocol: "TCP", Port: "22", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"}
	if sgPolicyMatchesRule(policy, rule) {
		t.Error("协议不匹配应返回 false")
	}
}

func TestSgPolicyMatchesRule_PortMismatch(t *testing.T) {
	policy := &vpc.SecurityGroupPolicy{
		Protocol:  common.StringPtr("TCP"),
		Port:      common.StringPtr("80"),
		CidrBlock: common.StringPtr("0.0.0.0/0"),
		Action:    common.StringPtr("ACCEPT"),
	}
	rule := requiredSGRule{Protocol: "TCP", Port: "22", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"}
	if sgPolicyMatchesRule(policy, rule) {
		t.Error("端口不匹配应返回 false")
	}
}

func TestSgPolicyMatchesRule_ActionMismatch(t *testing.T) {
	policy := &vpc.SecurityGroupPolicy{
		Protocol:  common.StringPtr("TCP"),
		Port:      common.StringPtr("22"),
		CidrBlock: common.StringPtr("0.0.0.0/0"),
		Action:    common.StringPtr("DROP"),
	}
	rule := requiredSGRule{Protocol: "TCP", Port: "22", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"}
	if sgPolicyMatchesRule(policy, rule) {
		t.Error("动作不匹配应返回 false")
	}
}

func TestSgPolicyMatchesRule_CidrMismatch(t *testing.T) {
	policy := &vpc.SecurityGroupPolicy{
		Protocol:  common.StringPtr("TCP"),
		Port:      common.StringPtr("22"),
		CidrBlock: common.StringPtr("10.0.0.0/8"),
		Action:    common.StringPtr("ACCEPT"),
	}
	rule := requiredSGRule{Protocol: "TCP", Port: "22", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"}
	if sgPolicyMatchesRule(policy, rule) {
		t.Error("CIDR 不匹配应返回 false")
	}
}

func TestSgPolicyMatchesRule_CidrNilInPolicy(t *testing.T) {
	policy := &vpc.SecurityGroupPolicy{
		Protocol: common.StringPtr("TCP"),
		Port:     common.StringPtr("22"),
		Action:   common.StringPtr("ACCEPT"),
		// CidrBlock 为 nil
	}
	rule := requiredSGRule{Protocol: "TCP", Port: "22", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"}
	if sgPolicyMatchesRule(policy, rule) {
		t.Error("规则要求 CidrBlock 但 policy 中为 nil 应返回 false")
	}
}

func TestSgPolicyMatchesRule_Ipv6Match(t *testing.T) {
	policy := &vpc.SecurityGroupPolicy{
		Protocol:      common.StringPtr("TCP"),
		Port:          common.StringPtr("22"),
		Ipv6CidrBlock: common.StringPtr("::/0"),
		Action:        common.StringPtr("ACCEPT"),
	}
	rule := requiredSGRule{Protocol: "TCP", Port: "22", Ipv6Cidr: "::/0", Action: "ACCEPT"}
	if !sgPolicyMatchesRule(policy, rule) {
		t.Error("IPv6 CIDR 匹配应返回 true")
	}
}

func TestSgPolicyMatchesRule_Ipv6Mismatch(t *testing.T) {
	policy := &vpc.SecurityGroupPolicy{
		Protocol:      common.StringPtr("TCP"),
		Port:          common.StringPtr("22"),
		Ipv6CidrBlock: common.StringPtr("fe80::/10"),
		Action:        common.StringPtr("ACCEPT"),
	}
	rule := requiredSGRule{Protocol: "TCP", Port: "22", Ipv6Cidr: "::/0", Action: "ACCEPT"}
	if sgPolicyMatchesRule(policy, rule) {
		t.Error("IPv6 CIDR 不匹配应返回 false")
	}
}

func TestSgPolicyMatchesRule_Ipv6NilInPolicy(t *testing.T) {
	policy := &vpc.SecurityGroupPolicy{
		Protocol: common.StringPtr("TCP"),
		Port:     common.StringPtr("22"),
		Action:   common.StringPtr("ACCEPT"),
		// Ipv6CidrBlock 为 nil
	}
	rule := requiredSGRule{Protocol: "TCP", Port: "22", Ipv6Cidr: "::/0", Action: "ACCEPT"}
	if sgPolicyMatchesRule(policy, rule) {
		t.Error("规则要求 Ipv6Cidr 但 policy 中为 nil 应返回 false")
	}
}

func TestSgPolicyMatchesRule_ALLProtocol(t *testing.T) {
	policy := &vpc.SecurityGroupPolicy{
		Protocol:  common.StringPtr("ALL"),
		Port:      common.StringPtr("ALL"),
		CidrBlock: common.StringPtr("0.0.0.0/0"),
		Action:    common.StringPtr("ACCEPT"),
	}
	rule := requiredSGRule{Protocol: "ALL", Port: "ALL", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"}
	if !sgPolicyMatchesRule(policy, rule) {
		t.Error("ALL 协议和端口应匹配")
	}
}

func TestSgPolicyMatchesRule_DropAction(t *testing.T) {
	policy := &vpc.SecurityGroupPolicy{
		Protocol:  common.StringPtr("ALL"),
		Port:      common.StringPtr("ALL"),
		CidrBlock: common.StringPtr("10.0.0.0/8"),
		Action:    common.StringPtr("DROP"),
	}
	rule := requiredSGRule{Protocol: "ALL", Port: "ALL", CidrBlock: "10.0.0.0/8", Action: "DROP"}
	if !sgPolicyMatchesRule(policy, rule) {
		t.Error("DROP 动作匹配应返回 true")
	}
}

func TestSgPolicyMatchesRule_NoCidrRequirement(t *testing.T) {
	// 规则不要求 CidrBlock 和 Ipv6Cidr 时，只要协议/端口/动作匹配即可
	policy := &vpc.SecurityGroupPolicy{
		Protocol:  common.StringPtr("TCP"),
		Port:      common.StringPtr("80"),
		CidrBlock: common.StringPtr("10.0.0.0/8"),
		Action:    common.StringPtr("ACCEPT"),
	}
	rule := requiredSGRule{Protocol: "TCP", Port: "80", Action: "ACCEPT"}
	if !sgPolicyMatchesRule(policy, rule) {
		t.Error("规则不要求 CIDR 时，只要其他字段匹配应返回 true")
	}
}

func TestSgPolicyMatchesRule_NilFields(t *testing.T) {
	// policy 所有字段为 nil，规则有具体要求
	policy := &vpc.SecurityGroupPolicy{}
	rule := requiredSGRule{Protocol: "TCP", Port: "22", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"}
	if sgPolicyMatchesRule(policy, rule) {
		t.Error("policy 字段全为 nil 但规则有要求时应返回 false")
	}
}

func TestSgPolicyMatchesRule_AllNilFieldsNoRequirement(t *testing.T) {
	// policy 所有字段为 nil，规则也无具体要求
	policy := &vpc.SecurityGroupPolicy{}
	rule := requiredSGRule{}
	if !sgPolicyMatchesRule(policy, rule) {
		t.Error("policy 和规则都无具体要求时应返回 true")
	}
}

// ==================== sgPolicyMatchesRule 表驱动测试 ====================

func TestSgPolicyMatchesRule_TableDriven(t *testing.T) {
	tests := []struct {
		name   string
		policy *vpc.SecurityGroupPolicy
		rule   requiredSGRule
		want   bool
	}{
		{
			"SSH IPv4 完全匹配",
			&vpc.SecurityGroupPolicy{
				Protocol: common.StringPtr("TCP"), Port: common.StringPtr("22"),
				CidrBlock: common.StringPtr("0.0.0.0/0"), Action: common.StringPtr("ACCEPT"),
			},
			requiredSGRule{Protocol: "TCP", Port: "22", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"},
			true,
		},
		{
			"SSH IPv6 完全匹配",
			&vpc.SecurityGroupPolicy{
				Protocol: common.StringPtr("TCP"), Port: common.StringPtr("22"),
				Ipv6CidrBlock: common.StringPtr("::/0"), Action: common.StringPtr("ACCEPT"),
			},
			requiredSGRule{Protocol: "TCP", Port: "22", Ipv6Cidr: "::/0", Action: "ACCEPT"},
			true,
		},
		{
			"出站全放通 IPv4",
			&vpc.SecurityGroupPolicy{
				Protocol: common.StringPtr("ALL"), Port: common.StringPtr("ALL"),
				CidrBlock: common.StringPtr("0.0.0.0/0"), Action: common.StringPtr("ACCEPT"),
			},
			requiredSGRule{Protocol: "ALL", Port: "ALL", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"},
			true,
		},
		{
			"ICMP 匹配",
			&vpc.SecurityGroupPolicy{
				Protocol: common.StringPtr("ICMP"), Port: common.StringPtr("ALL"),
				CidrBlock: common.StringPtr("0.0.0.0/0"), Action: common.StringPtr("ACCEPT"),
			},
			requiredSGRule{Protocol: "ICMP", Port: "ALL", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"},
			true,
		},
		{
			"RDP 端口匹配",
			&vpc.SecurityGroupPolicy{
				Protocol: common.StringPtr("TCP"), Port: common.StringPtr("3389"),
				CidrBlock: common.StringPtr("0.0.0.0/0"), Action: common.StringPtr("ACCEPT"),
			},
			requiredSGRule{Protocol: "TCP", Port: "3389", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"},
			true,
		},
		{
			"DNS UDP 出站",
			&vpc.SecurityGroupPolicy{
				Protocol: common.StringPtr("UDP"), Port: common.StringPtr("53"),
				CidrBlock: common.StringPtr("0.0.0.0/0"), Action: common.StringPtr("ACCEPT"),
			},
			requiredSGRule{Protocol: "UDP", Port: "53", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"},
			true,
		},
		{
			"协议不同",
			&vpc.SecurityGroupPolicy{
				Protocol: common.StringPtr("UDP"), Port: common.StringPtr("22"),
				CidrBlock: common.StringPtr("0.0.0.0/0"), Action: common.StringPtr("ACCEPT"),
			},
			requiredSGRule{Protocol: "TCP", Port: "22", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"},
			false,
		},
		{
			"端口不同",
			&vpc.SecurityGroupPolicy{
				Protocol: common.StringPtr("TCP"), Port: common.StringPtr("443"),
				CidrBlock: common.StringPtr("0.0.0.0/0"), Action: common.StringPtr("ACCEPT"),
			},
			requiredSGRule{Protocol: "TCP", Port: "22", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sgPolicyMatchesRule(tt.policy, tt.rule)
			if got != tt.want {
				t.Errorf("sgPolicyMatchesRule() = %v, 期望 %v", got, tt.want)
			}
		})
	}
}

// ==================== clawproRequiredRuleSet JSON 解析测试 ====================

func TestClawproRequiredRuleSet_ValidJSON(t *testing.T) {
	// 保存原始值并在测试结束后恢复
	origJSON := ClawproRequiredSGRulesJSON
	defer func() { ClawproRequiredSGRulesJSON = origJSON }()

	ClawproRequiredSGRulesJSON = []byte(`{
		"categories": [
			{
				"type": "builtin",
				"label": "内置规则",
				"rule_groups": [
					{
						"key": "allow_ssh",
						"name": "允许SSH登录",
						"rules": [
							{
								"direction": "ingress",
								"protocol": "TCP",
								"port": "22",
								"cidr_block": "0.0.0.0/0",
								"action": "ACCEPT",
								"description": "放通SSH"
							}
						]
					}
				]
			},
			{
				"type": "recommended",
				"label": "推荐规则",
				"rule_groups": [
					{
						"key": "allow_rdp",
						"name": "允许RDP",
						"rules": [
							{
								"direction": "ingress",
								"protocol": "TCP",
								"port": "3389",
								"cidr_block": "0.0.0.0/0",
								"action": "ACCEPT",
								"description": "放通RDP"
							}
						]
					}
				]
			}
		]
	}`)

	ruleSet := clawproRequiredRuleSet()

	if len(ruleSet.Categories) != 2 {
		t.Fatalf("期望 2 个分类，实际=%d", len(ruleSet.Categories))
	}

	// 验证 builtin 分类
	builtin := ruleSet.Categories[0]
	if builtin.Type != "builtin" {
		t.Errorf("第一个分类 Type 期望 builtin，实际=%s", builtin.Type)
	}
	if builtin.Label != "内置规则" {
		t.Errorf("第一个分类 Label 期望 '内置规则'，实际=%s", builtin.Label)
	}
	if len(builtin.RuleGroups) != 1 {
		t.Fatalf("builtin 分类期望 1 个规则组，实际=%d", len(builtin.RuleGroups))
	}
	if builtin.RuleGroups[0].Key != "allow_ssh" {
		t.Errorf("规则组 Key 期望 allow_ssh，实际=%s", builtin.RuleGroups[0].Key)
	}
	if builtin.RuleGroups[0].Name != "允许SSH登录" {
		t.Errorf("规则组 Name 期望 '允许SSH登录'，实际=%s", builtin.RuleGroups[0].Name)
	}
	if len(builtin.RuleGroups[0].Rules) != 1 {
		t.Fatalf("allow_ssh 规则组期望 1 条规则，实际=%d", len(builtin.RuleGroups[0].Rules))
	}

	rule := builtin.RuleGroups[0].Rules[0]
	if rule.Direction != "ingress" || rule.Protocol != "TCP" || rule.Port != "22" {
		t.Errorf("规则内容不匹配: %+v", rule)
	}

	// 验证 recommended 分类
	recommended := ruleSet.Categories[1]
	if recommended.Type != "recommended" {
		t.Errorf("第二个分类 Type 期望 recommended，实际=%s", recommended.Type)
	}
	if len(recommended.RuleGroups) != 1 || recommended.RuleGroups[0].Key != "allow_rdp" {
		t.Errorf("recommended 分类规则组不匹配")
	}
}

func TestClawproRequiredRuleSet_EmptyJSON(t *testing.T) {
	origJSON := ClawproRequiredSGRulesJSON
	defer func() { ClawproRequiredSGRulesJSON = origJSON }()

	ClawproRequiredSGRulesJSON = []byte(`{}`)
	ruleSet := clawproRequiredRuleSet()
	if len(ruleSet.Categories) != 0 {
		t.Errorf("空 JSON 期望 0 个分类，实际=%d", len(ruleSet.Categories))
	}
}

func TestClawproRequiredRuleSet_InvalidJSON(t *testing.T) {
	origJSON := ClawproRequiredSGRulesJSON
	defer func() { ClawproRequiredSGRulesJSON = origJSON }()

	ClawproRequiredSGRulesJSON = []byte(`{invalid json}`)
	ruleSet := clawproRequiredRuleSet()
	if len(ruleSet.Categories) != 0 {
		t.Errorf("无效 JSON 期望返回空 ruleSet，实际分类数=%d", len(ruleSet.Categories))
	}
}

func TestClawproRequiredRuleSet_NilJSON(t *testing.T) {
	origJSON := ClawproRequiredSGRulesJSON
	defer func() { ClawproRequiredSGRulesJSON = origJSON }()

	ClawproRequiredSGRulesJSON = nil
	// 当 JSON 为 nil 时，会尝试从磁盘读取，如果文件不存在则返回空
	ruleSet := clawproRequiredRuleSet()
	// 不 panic 即可，结果取决于磁盘上是否有文件
	_ = ruleSet
}

// ==================== createSGRequest QuickRules 规则组匹配测试 ====================

func TestQuickRulesGroupMapping(t *testing.T) {
	origJSON := ClawproRequiredSGRulesJSON
	defer func() { ClawproRequiredSGRulesJSON = origJSON }()

	ClawproRequiredSGRulesJSON = []byte(`{
		"categories": [
			{
				"type": "builtin", "label": "内置",
				"rule_groups": [
					{
						"key": "allow_ssh", "name": "SSH",
						"rules": [
							{"direction":"ingress","protocol":"TCP","port":"22","cidr_block":"0.0.0.0/0","action":"ACCEPT","description":"SSH IPv4"}
						]
					},
					{
						"key": "allow_internet", "name": "公网",
						"rules": [
							{"direction":"egress","protocol":"ALL","port":"ALL","cidr_block":"0.0.0.0/0","action":"ACCEPT","description":"出站"}
						]
					}
				]
			},
			{
				"type": "recommended", "label": "推荐",
				"rule_groups": [
					{
						"key": "allow_rdp", "name": "RDP",
						"rules": [
							{"direction":"ingress","protocol":"TCP","port":"3389","cidr_block":"0.0.0.0/0","action":"ACCEPT","description":"RDP"}
						]
					}
				]
			}
		]
	}`)

	// 模拟 HandleCreateSecurityGroup 中的规则组匹配逻辑
	ruleSet := clawproRequiredRuleSet()
	ruleGroupMap := make(map[string][]requiredSGRule)
	for _, category := range ruleSet.Categories {
		for _, group := range category.RuleGroups {
			ruleGroupMap[group.Key] = group.Rules
		}
	}

	tests := []struct {
		name      string
		quickRule string
		found     bool
		ruleCount int
	}{
		{"builtin 规则 allow_ssh", "allow_ssh", true, 1},
		{"builtin 规则 allow_internet", "allow_internet", true, 1},
		{"recommended 规则 allow_rdp", "allow_rdp", true, 1},
		{"不存在的规则", "nonexistent", false, 0},
		{"空字符串", "", false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules, ok := ruleGroupMap[tt.quickRule]
			if ok != tt.found {
				t.Errorf("规则 %q 查找结果期望 %v，实际=%v", tt.quickRule, tt.found, ok)
			}
			if len(rules) != tt.ruleCount {
				t.Errorf("规则 %q 期望 %d 条规则，实际=%d", tt.quickRule, tt.ruleCount, len(rules))
			}
		})
	}
}

func TestQuickRulesDirectionSplit(t *testing.T) {
	origJSON := ClawproRequiredSGRulesJSON
	defer func() { ClawproRequiredSGRulesJSON = origJSON }()

	ClawproRequiredSGRulesJSON = []byte(`{
		"categories": [{
			"type": "builtin", "label": "内置",
			"rule_groups": [
				{
					"key": "allow_ssh", "name": "SSH",
					"rules": [
						{"direction":"ingress","protocol":"TCP","port":"22","cidr_block":"0.0.0.0/0","action":"ACCEPT","description":"SSH"}
					]
				},
				{
					"key": "allow_internet", "name": "公网",
					"rules": [
						{"direction":"egress","protocol":"ALL","port":"ALL","cidr_block":"0.0.0.0/0","action":"ACCEPT","description":"出站"}
					]
				}
			]
		}]
	}`)

	// 模拟 HandleCreateSecurityGroup 中的入站/出站分离逻辑
	ruleSet := clawproRequiredRuleSet()
	ruleGroupMap := make(map[string][]requiredSGRule)
	for _, category := range ruleSet.Categories {
		for _, group := range category.RuleGroups {
			ruleGroupMap[group.Key] = group.Rules
		}
	}

	quickRules := []string{"allow_ssh", "allow_internet"}
	var ingressCount, egressCount int

	for _, ruleName := range quickRules {
		rules, ok := ruleGroupMap[ruleName]
		if !ok {
			continue
		}
		for _, rule := range rules {
			if rule.Direction == "ingress" {
				ingressCount++
			} else {
				egressCount++
			}
		}
	}

	if ingressCount != 1 {
		t.Errorf("期望 1 条入站规则，实际=%d", ingressCount)
	}
	if egressCount != 1 {
		t.Errorf("期望 1 条出站规则，实际=%d", egressCount)
	}
}

// ==================== sgRuleSet 结构体序列化测试 ====================

func TestSgRuleSetJSON_Serialization(t *testing.T) {
	ruleSet := sgRuleSet{
		Categories: []sgRuleCategory{
			{
				Type:  "builtin",
				Label: "内置规则",
				RuleGroups: []sgRuleGroup{
					{
						Key:  "allow_ssh",
						Name: "允许SSH登录",
						Rules: []requiredSGRule{
							{
								Direction:   "ingress",
								Protocol:    "TCP",
								Port:        "22",
								CidrBlock:   "0.0.0.0/0",
								Action:      "ACCEPT",
								Description: "放通SSH",
							},
						},
					},
				},
			},
		},
	}

	data, err := json.Marshal(ruleSet)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}

	var decoded sgRuleSet
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}

	if len(decoded.Categories) != 1 {
		t.Fatalf("期望 1 个分类，实际=%d", len(decoded.Categories))
	}
	if decoded.Categories[0].Type != "builtin" {
		t.Errorf("Type 期望 builtin，实际=%s", decoded.Categories[0].Type)
	}
	if decoded.Categories[0].RuleGroups[0].Key != "allow_ssh" {
		t.Errorf("Key 期望 allow_ssh，实际=%s", decoded.Categories[0].RuleGroups[0].Key)
	}
	if decoded.Categories[0].RuleGroups[0].Name != "允许SSH登录" {
		t.Errorf("Name 期望 '允许SSH登录'，实际=%s", decoded.Categories[0].RuleGroups[0].Name)
	}
}

func TestSgRuleGroupJSON_KeyAndName(t *testing.T) {
	group := sgRuleGroup{
		Key:   "allow_ssh",
		Name:  "允许LinuxSSH登录",
		Rules: []requiredSGRule{},
	}

	data, err := json.Marshal(group)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}

	var m map[string]interface{}
	json.Unmarshal(data, &m)

	if m["key"] != "allow_ssh" {
		t.Errorf("JSON key 字段期望 allow_ssh，实际=%v", m["key"])
	}
	if m["name"] != "允许LinuxSSH登录" {
		t.Errorf("JSON name 字段期望 '允许LinuxSSH登录'，实际=%v", m["name"])
	}
}

// ==================== DefaultChecked 字段测试 ====================

// TestSgRuleGroup_DefaultCheckedOmitEmpty 验证 DefaultChecked 字段在为 false 时会被省略。
func TestSgRuleGroup_DefaultCheckedOmitEmpty(t *testing.T) {
	group := sgRuleGroup{
		Key:   "allow_ssh",
		Name:  "允许SSH",
		Rules: []requiredSGRule{},
		// DefaultChecked 默认为 false
	}

	data, err := json.Marshal(group)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}

	if _, ok := m["default_checked"]; ok {
		t.Errorf("DefaultChecked 为 false 时不应出现在 JSON 中，实际=%v", m["default_checked"])
	}
}

// TestSgRuleGroup_DefaultCheckedTrue 验证 DefaultChecked 字段在为 true 时会被序列化。
func TestSgRuleGroup_DefaultCheckedTrue(t *testing.T) {
	group := sgRuleGroup{
		Key:            "allow_ssh",
		Name:           "允许SSH",
		DefaultChecked: true,
		Rules:          []requiredSGRule{},
	}

	data, err := json.Marshal(group)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}

	v, ok := m["default_checked"]
	if !ok {
		t.Fatal("DefaultChecked 为 true 时应出现在 JSON 中")
	}
	if b, _ := v.(bool); !b {
		t.Errorf("default_checked 期望 true，实际=%v", v)
	}
}

// TestSgRuleGroup_DefaultCheckedParsing 验证 JSON 中 default_checked 字段能够正确解析。
func TestSgRuleGroup_DefaultCheckedParsing(t *testing.T) {
	origJSON := ClawproRequiredSGRulesJSON
	defer func() { ClawproRequiredSGRulesJSON = origJSON }()

	ClawproRequiredSGRulesJSON = []byte(`{
		"categories": [{
			"type": "recommended", "label": "推荐规则",
			"rule_groups": [
				{
					"key": "allow_ssh", "name": "SSH",
					"default_checked": true,
					"rules": []
				},
				{
					"key": "allow_rdp", "name": "RDP",
					"default_checked": false,
					"rules": []
				},
				{
					"key": "allow_icmp", "name": "ICMP",
					"rules": []
				}
			]
		}]
	}`)

	ruleSet := clawproRequiredRuleSet()
	if len(ruleSet.Categories) != 1 {
		t.Fatalf("期望 1 个分类，实际=%d", len(ruleSet.Categories))
	}
	groups := ruleSet.Categories[0].RuleGroups
	if len(groups) != 3 {
		t.Fatalf("期望 3 个规则组，实际=%d", len(groups))
	}

	if !groups[0].DefaultChecked {
		t.Errorf("groups[0](allow_ssh) DefaultChecked 期望 true，实际=false")
	}
	if groups[1].DefaultChecked {
		t.Errorf("groups[1](allow_rdp) DefaultChecked 期望 false，实际=true")
	}
	if groups[2].DefaultChecked {
		t.Errorf("groups[2](allow_icmp) 未设置时 DefaultChecked 期望 false，实际=true")
	}
}

// ==================== replaceVpcCidrPlaceholder 测试 ====================

// buildRuleSetWithPlaceholder 辅助构造含 {{VPC_CIDR}} 占位符的规则集。
func buildRuleSetWithPlaceholder() sgRuleSet {
	return sgRuleSet{
		Categories: []sgRuleCategory{
			{
				Type:  "builtin",
				Label: "内置",
				RuleGroups: []sgRuleGroup{
					{
						Key:  "allow_internal_all",
						Name: "内网全放通",
						Rules: []requiredSGRule{
							{
								Direction:   "ingress",
								Protocol:    "ALL",
								Port:        "ALL",
								CidrBlock:   "{{VPC_CIDR}}",
								Action:      "ACCEPT",
								Description: "VPC 内网全放通",
							},
							{
								Direction:   "ingress",
								Protocol:    "TCP",
								Port:        "22",
								CidrBlock:   "0.0.0.0/0",
								Action:      "ACCEPT",
								Description: "SSH",
							},
						},
					},
					{
						Key:  "allow_internal_tcp",
						Name: "内网TCP",
						Rules: []requiredSGRule{
							{
								Direction:   "ingress",
								Protocol:    "TCP",
								Port:        "ALL",
								CidrBlock:   "{{VPC_CIDR}}",
								Action:      "ACCEPT",
								Description: "VPC 内 TCP 全放通",
							},
						},
					},
				},
			},
			{
				Type:  "recommended",
				Label: "推荐",
				RuleGroups: []sgRuleGroup{
					{
						Key:  "allow_rdp",
						Name: "RDP",
						Rules: []requiredSGRule{
							{
								Direction:   "ingress",
								Protocol:    "TCP",
								Port:        "3389",
								CidrBlock:   "0.0.0.0/0",
								Action:      "ACCEPT",
								Description: "RDP",
							},
						},
					},
				},
			},
		},
	}
}

// TestReplaceVpcCidrPlaceholder_ReplacesAllOccurrences 验证所有 {{VPC_CIDR}} 占位符都会被替换。
func TestReplaceVpcCidrPlaceholder_ReplacesAllOccurrences(t *testing.T) {
	ruleSet := buildRuleSetWithPlaceholder()
	replaceVpcCidrPlaceholder(&ruleSet, "10.0.0.0/16")

	// 规则组 0 第 0 条：原占位符 -> 10.0.0.0/16
	if got := ruleSet.Categories[0].RuleGroups[0].Rules[0].CidrBlock; got != "10.0.0.0/16" {
		t.Errorf("第一条规则 CidrBlock 期望 10.0.0.0/16，实际=%s", got)
	}
	// 规则组 0 第 1 条：非占位符 -> 保持原值
	if got := ruleSet.Categories[0].RuleGroups[0].Rules[1].CidrBlock; got != "0.0.0.0/0" {
		t.Errorf("非占位符规则不应被修改，期望 0.0.0.0/0，实际=%s", got)
	}
	// 规则组 1 第 0 条：原占位符 -> 10.0.0.0/16
	if got := ruleSet.Categories[0].RuleGroups[1].Rules[0].CidrBlock; got != "10.0.0.0/16" {
		t.Errorf("第二个规则组的占位符 CidrBlock 期望 10.0.0.0/16，实际=%s", got)
	}
	// 分类 1 规则组 0 第 0 条：非占位符 -> 保持原值
	if got := ruleSet.Categories[1].RuleGroups[0].Rules[0].CidrBlock; got != "0.0.0.0/0" {
		t.Errorf("recommended 分类规则不应被修改，期望 0.0.0.0/0，实际=%s", got)
	}
}

// TestReplaceVpcCidrPlaceholder_EmptyCidrNoop 验证 vpcCidr 为空时不修改任何规则。
func TestReplaceVpcCidrPlaceholder_EmptyCidrNoop(t *testing.T) {
	ruleSet := buildRuleSetWithPlaceholder()
	replaceVpcCidrPlaceholder(&ruleSet, "")

	if got := ruleSet.Categories[0].RuleGroups[0].Rules[0].CidrBlock; got != "{{VPC_CIDR}}" {
		t.Errorf("vpcCidr 为空时占位符应保留，实际=%s", got)
	}
	if got := ruleSet.Categories[0].RuleGroups[1].Rules[0].CidrBlock; got != "{{VPC_CIDR}}" {
		t.Errorf("vpcCidr 为空时占位符应保留，实际=%s", got)
	}
}

// TestReplaceVpcCidrPlaceholder_PartialMatchNotReplaced 验证仅当整个字段完全等于占位符时才替换。
func TestReplaceVpcCidrPlaceholder_PartialMatchNotReplaced(t *testing.T) {
	ruleSet := sgRuleSet{
		Categories: []sgRuleCategory{{
			Type: "builtin", Label: "内置",
			RuleGroups: []sgRuleGroup{{
				Key: "partial", Name: "部分",
				Rules: []requiredSGRule{
					{CidrBlock: "prefix-{{VPC_CIDR}}-suffix"},
					{CidrBlock: "{{VPC_CIDR}}/24"},
					{CidrBlock: "{{VPC_CIDR}}"},
				},
			}},
		}},
	}
	replaceVpcCidrPlaceholder(&ruleSet, "172.16.0.0/16")

	rules := ruleSet.Categories[0].RuleGroups[0].Rules
	if rules[0].CidrBlock != "prefix-{{VPC_CIDR}}-suffix" {
		t.Errorf("非完全匹配不应被替换，实际=%s", rules[0].CidrBlock)
	}
	if rules[1].CidrBlock != "{{VPC_CIDR}}/24" {
		t.Errorf("非完全匹配不应被替换，实际=%s", rules[1].CidrBlock)
	}
	if rules[2].CidrBlock != "172.16.0.0/16" {
		t.Errorf("完全匹配应被替换为 172.16.0.0/16，实际=%s", rules[2].CidrBlock)
	}
}

// TestReplaceVpcCidrPlaceholder_EmptyRuleSet 验证空规则集不会导致 panic。
func TestReplaceVpcCidrPlaceholder_EmptyRuleSet(t *testing.T) {
	ruleSet := sgRuleSet{}
	// 只要不 panic 即可
	replaceVpcCidrPlaceholder(&ruleSet, "10.0.0.0/16")
	if len(ruleSet.Categories) != 0 {
		t.Errorf("空 ruleSet 不应被修改")
	}
}

// TestReplaceVpcCidrPlaceholder_EmptyRulesInGroup 验证规则组无规则时不会导致 panic。
func TestReplaceVpcCidrPlaceholder_EmptyRulesInGroup(t *testing.T) {
	ruleSet := sgRuleSet{
		Categories: []sgRuleCategory{{
			Type: "builtin", Label: "内置",
			RuleGroups: []sgRuleGroup{{
				Key: "empty", Name: "空", Rules: []requiredSGRule{},
			}},
		}},
	}
	replaceVpcCidrPlaceholder(&ruleSet, "10.0.0.0/16")
	if len(ruleSet.Categories[0].RuleGroups[0].Rules) != 0 {
		t.Errorf("空规则列表不应被修改")
	}
}

// TestReplaceVpcCidrPlaceholder_DoesNotTouchIpv6 验证仅替换 CidrBlock，不会误改 Ipv6Cidr。
func TestReplaceVpcCidrPlaceholder_DoesNotTouchIpv6(t *testing.T) {
	ruleSet := sgRuleSet{
		Categories: []sgRuleCategory{{
			Type: "builtin", Label: "内置",
			RuleGroups: []sgRuleGroup{{
				Key: "ipv6", Name: "IPv6",
				Rules: []requiredSGRule{
					{CidrBlock: "{{VPC_CIDR}}", Ipv6Cidr: "{{VPC_CIDR}}"},
				},
			}},
		}},
	}
	replaceVpcCidrPlaceholder(&ruleSet, "10.0.0.0/16")

	rule := ruleSet.Categories[0].RuleGroups[0].Rules[0]
	if rule.CidrBlock != "10.0.0.0/16" {
		t.Errorf("CidrBlock 期望 10.0.0.0/16，实际=%s", rule.CidrBlock)
	}
	// 当前实现只替换 CidrBlock，Ipv6Cidr 不应被触碰
	if rule.Ipv6Cidr != "{{VPC_CIDR}}" {
		t.Errorf("Ipv6Cidr 不应被替换，实际=%s", rule.Ipv6Cidr)
	}
}

// TestReplaceVpcCidrPlaceholder_MutatesInPlace 验证函数确实通过指针修改了原对象。
func TestReplaceVpcCidrPlaceholder_MutatesInPlace(t *testing.T) {
	ruleSet := sgRuleSet{
		Categories: []sgRuleCategory{{
			Type: "builtin", Label: "内置",
			RuleGroups: []sgRuleGroup{{
				Key: "t", Name: "t",
				Rules: []requiredSGRule{{CidrBlock: "{{VPC_CIDR}}"}},
			}},
		}},
	}
	// 对原 ruleSet 取地址调用
	replaceVpcCidrPlaceholder(&ruleSet, "192.168.0.0/16")
	if ruleSet.Categories[0].RuleGroups[0].Rules[0].CidrBlock != "192.168.0.0/16" {
		t.Errorf("应该就地修改 ruleSet，实际=%s", ruleSet.Categories[0].RuleGroups[0].Rules[0].CidrBlock)
	}
}

// TestReplaceVpcCidrPlaceholder_MultipleCidrs 验证可多次调用使用不同 CIDR。
func TestReplaceVpcCidrPlaceholder_MultipleCidrs(t *testing.T) {
	tests := []struct {
		name    string
		vpcCidr string
	}{
		{"10网段", "10.0.0.0/16"},
		{"172网段", "172.16.0.0/12"},
		{"192网段", "192.168.0.0/16"},
		{"小网段", "10.1.2.0/24"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ruleSet := sgRuleSet{
				Categories: []sgRuleCategory{{
					Type: "builtin", Label: "内置",
					RuleGroups: []sgRuleGroup{{
						Key: "t", Name: "t",
						Rules: []requiredSGRule{{CidrBlock: "{{VPC_CIDR}}"}},
					}},
				}},
			}
			replaceVpcCidrPlaceholder(&ruleSet, tt.vpcCidr)
			if got := ruleSet.Categories[0].RuleGroups[0].Rules[0].CidrBlock; got != tt.vpcCidr {
				t.Errorf("CidrBlock 期望 %s，实际 %s", tt.vpcCidr, got)
			}
		})
	}
}

// ==================== HandleBindSecurityGroup 绑定自身安全组检查测试 ====================

// initBindSGTestDB 为 HandleBindSecurityGroup 相关测试初始化内存数据库，
// 并写入一条 SiteConfig 记录，允许通过 currentSgId 指定当前已配置的安全组 ID。
func initBindSGTestDB(t *testing.T, currentSgId string) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.User{}, &model.Instance{}, &model.SiteConfig{}, &model.ManagedSGPool{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	cfg := model.SiteConfig{SecurityGroupId: currentSgId}
	if err := db.Create(&cfg).Error; err != nil {
		t.Fatalf("创建 SiteConfig 失败: %v", err)
	}
}

// newBindSGRequest 构造一个带 AdminToken 认证的绑定安全组 JSON 请求。
func newBindSGRequest(t *testing.T, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/admin/config/security-group/bind",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req
}

// TestHandleBindSecurityGroup_RejectBindSelf 验证当请求绑定的安全组与当前已配置的
// 安全组一致时，接口直接拒绝并给出中文提示。
func TestHandleBindSecurityGroup_RejectBindSelf(t *testing.T) {
	initBindSGTestDB(t, "sg-aaa111")

	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := newBindSGRequest(t, `{"security_group_id":"sg-aaa111"}`)
	w := httptest.NewRecorder()

	HandleBindSecurityGroup(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v, body=%s", err, w.Body.String())
	}
	if !strings.Contains(resp["error"].(string), "当前使用的安全组") {
		t.Errorf("错误信息应包含 '当前使用的安全组'，实际=%q", resp["error"])
	}
}

// TestHandleBindSecurityGroup_EmptySecurityGroupId 验证 security_group_id 为空时
// 返回 400，且提示空值错误（而非自身安全组错误）。
func TestHandleBindSecurityGroup_EmptySecurityGroupId(t *testing.T) {
	initBindSGTestDB(t, "sg-aaa111")

	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := newBindSGRequest(t, `{"security_group_id":""}`)
	w := httptest.NewRecorder()

	HandleBindSecurityGroup(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v, body=%s", err, w.Body.String())
	}
	if !strings.Contains(resp["error"].(string), "不能为空") {
		t.Errorf("错误信息应包含 '不能为空'，实际=%q", resp["error"])
	}
	if strings.Contains(resp["error"].(string), "当前使用的安全组") {
		t.Errorf("空值场景不应触发自身安全组检查，实际=%q", resp["error"])
	}
}

// TestHandleBindSecurityGroup_InvalidJSON 验证请求体非法 JSON 时返回 400。
// 保证自身安全组检查位于 JSON 解析之后，不会掩盖参数格式错误。
func TestHandleBindSecurityGroup_InvalidJSON(t *testing.T) {
	initBindSGTestDB(t, "sg-aaa111")

	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := newBindSGRequest(t, `{invalid json`)
	w := httptest.NewRecorder()

	HandleBindSecurityGroup(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v, body=%s", err, w.Body.String())
	}
	if !strings.Contains(resp["error"].(string), "请求体格式错误") {
		t.Errorf("错误信息应包含 '请求体格式错误'，实际=%q", resp["error"])
	}
}

// TestHandleBindSecurityGroup_DifferentSgIdPassesSelfCheck 验证当请求的
// security_group_id 与当前已配置的安全组不同时，自身安全组检查放行（不会返回
// "当前使用的安全组" 的错误提示）。由于测试环境未配置真实的腾讯云凭证，后续
// 流程会因创建 VPC 客户端失败而返回 5xx，这里只断言不被自身检查拦截即可。
func TestHandleBindSecurityGroup_DifferentSgIdPassesSelfCheck(t *testing.T) {
	initBindSGTestDB(t, "sg-aaa111")

	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := newBindSGRequest(t, `{"security_group_id":"sg-bbb222"}`)
	w := httptest.NewRecorder()

	HandleBindSecurityGroup(w, req)

	// 无论后续 VPC 调用结果如何，只要响应正文不包含 "当前使用的安全组" 即说明
	// 自身安全组检查已放行。
	if strings.Contains(w.Body.String(), "当前使用的安全组") {
		t.Errorf("不同安全组 ID 不应触发自身安全组检查，实际 body=%s", w.Body.String())
	}
}

// TestHandleBindSecurityGroup_EmptyCurrentSgPassesSelfCheck 验证当前未配置安全组
// （SecurityGroupId 为空）时，自身安全组检查不会拦截任何绑定请求。
func TestHandleBindSecurityGroup_EmptyCurrentSgPassesSelfCheck(t *testing.T) {
	initBindSGTestDB(t, "")

	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := newBindSGRequest(t, `{"security_group_id":"sg-ccc333"}`)
	w := httptest.NewRecorder()

	HandleBindSecurityGroup(w, req)

	if strings.Contains(w.Body.String(), "当前使用的安全组") {
		t.Errorf("未配置安全组时不应触发自身安全组检查，实际 body=%s", w.Body.String())
	}
}

// ==================== HTTP Handler 覆盖率补充测试 ====================
//
// 以下测试聚焦于提升覆盖率：通过 requireAdmin 失败 / newVpcClient 失败（无凭据）/
// 参数校验失败等路径，驱动 handler 执行到相应分支。
//
// 测试环境中 model.SiteConfig 无 CVMSecretId/CVMSecretKey，因此 getCredential
// 会返回 "凭据未配置" 错误，所有经过 newVpcClient() 的 handler 都会走到
// "创建 VPC 客户端失败" 分支（HTTP 500），这正好覆盖了 handler 前中段代码。

// initSGHandlerTestDB 为通用 handler 测试初始化内存数据库（无任何凭据配置）。
func initSGHandlerTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.User{}, &model.Instance{}, &model.SiteConfig{}, &model.ManagedSGPool{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	// 空 SiteConfig（不填 CVMSecretId/CVMSecretKey，使 getCredential 返回 error）
	if err := db.Create(&model.SiteConfig{}).Error; err != nil {
		t.Fatalf("创建 SiteConfig 失败: %v", err)
	}
}

// withAdminToken 保存并临时设置 AdminToken，返回 teardown 函数。
func withAdminToken(token string) func() {
	orig := AdminToken
	AdminToken = token
	return func() { AdminToken = orig }
}

// newAdminRequest 构造带 AdminToken 鉴权头的请求。
func newAdminRequest(method, target, body string) *http.Request {
	var reader *strings.Reader
	if body != "" {
		reader = strings.NewReader(body)
	} else {
		reader = strings.NewReader("")
	}
	req := httptest.NewRequest(method, target, reader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req
}

// setSiteConfigSG 在已存在的 SiteConfig 上更新 SecurityGroupId。
func setSiteConfigSG(t *testing.T, sgId string) {
	t.Helper()
	if err := model.DB(context.Background()).Model(&model.SiteConfig{}).
		Where("1 = 1").
		Update("security_group_id", sgId).Error; err != nil {
		t.Fatalf("更新 SiteConfig 失败: %v", err)
	}
}

// ---------- HandleGetSecurityGroup ----------

func TestHandleGetSecurityGroup_EmptySGId_Returns(t *testing.T) {
	// 当 SiteConfig.SecurityGroupId 为空时，handler 直接 return（不调用 VPC）
	initSGHandlerTestDB(t)
	defer withAdminToken("test-admin-token")()

	req := newAdminRequest(http.MethodGet, "/admin/config/security-group", "")
	w := httptest.NewRecorder()
	HandleGetSecurityGroup(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200（无配置直接 return），实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleGetSecurityGroup_VpcClientFailure(t *testing.T) {
	// 配置了 SecurityGroupId 但无凭据 -> newVpcClient 失败 -> 500
	initSGHandlerTestDB(t)
	setSiteConfigSG(t, "sg-get-test")
	defer withAdminToken("test-admin-token")()

	req := newAdminRequest(http.MethodGet, "/admin/config/security-group", "")
	w := httptest.NewRecorder()
	HandleGetSecurityGroup(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("期望 500，实际=%d, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "创建 VPC 客户端失败") {
		t.Errorf("错误信息应包含 '创建 VPC 客户端失败'，实际=%s", w.Body.String())
	}
}

// ---------- HandleCreateSecurityGroup ----------

func TestHandleCreateSecurityGroup_InvalidJSON(t *testing.T) {
	initSGHandlerTestDB(t)
	defer withAdminToken("test-admin-token")()

	req := newAdminRequest(http.MethodPost, "/admin/config/security-group/create", `{invalid json`)
	w := httptest.NewRecorder()
	HandleCreateSecurityGroup(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "请求体格式错误") {
		t.Errorf("错误信息应包含 '请求体格式错误'，实际=%s", w.Body.String())
	}
}

func TestHandleCreateSecurityGroup_EmptyGroupName(t *testing.T) {
	initSGHandlerTestDB(t)
	defer withAdminToken("test-admin-token")()

	req := newAdminRequest(http.MethodPost, "/admin/config/security-group/create",
		`{"GroupName":"","GroupDescription":"desc"}`)
	w := httptest.NewRecorder()
	HandleCreateSecurityGroup(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "安全组名称不能为空") {
		t.Errorf("错误信息应包含 '安全组名称不能为空'，实际=%s", w.Body.String())
	}
}

func TestHandleCreateSecurityGroup_VpcClientFailure(t *testing.T) {
	initSGHandlerTestDB(t)
	defer withAdminToken("test-admin-token")()

	req := newAdminRequest(http.MethodPost, "/admin/config/security-group/create",
		`{"GroupName":"sg-new","GroupDescription":"desc","quick_rules":["allow_internet","restrict_vpc_access","unknown_rule"]}`)
	w := httptest.NewRecorder()
	HandleCreateSecurityGroup(w, req)

	// 无凭据 -> newVpcClient 失败 -> 500
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("期望 500，实际=%d, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "创建 VPC 客户端失败") {
		t.Errorf("错误信息应包含 '创建 VPC 客户端失败'，实际=%s", w.Body.String())
	}
}

// ---------- HandleBindSecurityGroup（已有自身检查测试，这里补充 VPC 失败分支）----------

func TestHandleBindSecurityGroup_VpcClientFailure(t *testing.T) {
	initSGHandlerTestDB(t)
	setSiteConfigSG(t, "sg-current")
	defer withAdminToken("test-admin-token")()

	// 绑定一个不同的 SG，通过自身检查后因无凭据失败
	req := newAdminRequest(http.MethodPost, "/admin/config/security-group/bind",
		`{"security_group_id":"sg-different"}`)
	w := httptest.NewRecorder()
	HandleBindSecurityGroup(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("期望 500，实际=%d, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "创建 VPC 客户端失败") {
		t.Errorf("错误信息应包含 '创建 VPC 客户端失败'，实际=%s", w.Body.String())
	}
}

func TestHandleBindSecurityGroup_AutoFixRules_VpcClientFailure(t *testing.T) {
	initSGHandlerTestDB(t)
	setSiteConfigSG(t, "sg-current")
	defer withAdminToken("test-admin-token")()

	req := newAdminRequest(http.MethodPost, "/admin/config/security-group/bind",
		`{"security_group_id":"sg-different","auto_fix_rules":true}`)
	w := httptest.NewRecorder()
	HandleBindSecurityGroup(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("期望 500，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// ---------- HandleUpdateSecurityGroup ----------

func TestHandleUpdateSecurityGroup_NoSGConfigured(t *testing.T) {
	initSGHandlerTestDB(t)
	// SecurityGroupId 为空
	defer withAdminToken("test-admin-token")()

	req := newAdminRequest(http.MethodPost, "/admin/config/security-group/update",
		`{"GroupName":"sg-new"}`)
	w := httptest.NewRecorder()
	HandleUpdateSecurityGroup(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "未配置安全组") {
		t.Errorf("错误信息应包含 '未配置安全组'，实际=%s", w.Body.String())
	}
}

func TestHandleUpdateSecurityGroup_VpcClientFailure(t *testing.T) {
	initSGHandlerTestDB(t)
	setSiteConfigSG(t, "sg-update-test")
	defer withAdminToken("test-admin-token")()

	req := newAdminRequest(http.MethodPost, "/admin/config/security-group/update",
		`{"GroupName":"sg-new"}`)
	w := httptest.NewRecorder()
	HandleUpdateSecurityGroup(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("期望 500，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// ---------- HandleDescribeSecurityGroupPolicies ----------

func TestHandleDescribeSecurityGroupPolicies_NoSGConfigured(t *testing.T) {
	initSGHandlerTestDB(t)
	defer withAdminToken("test-admin-token")()

	req := newAdminRequest(http.MethodGet, "/admin/config/security-group/policies", "")
	w := httptest.NewRecorder()
	HandleDescribeSecurityGroupPolicies(w, req)

	// 未配置时直接 return，响应码为 200
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200（无配置直接 return），实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleDescribeSecurityGroupPolicies_VpcClientFailure(t *testing.T) {
	initSGHandlerTestDB(t)
	setSiteConfigSG(t, "sg-desc-test")
	defer withAdminToken("test-admin-token")()

	req := newAdminRequest(http.MethodGet, "/admin/config/security-group/policies", "")
	w := httptest.NewRecorder()
	HandleDescribeSecurityGroupPolicies(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("期望 500，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// ---------- HandleCreateSecurityGroupPolicies ----------

// ---------- HandleReplaceSecurityGroupPolicy ----------

// ---------- HandleDeleteSecurityGroupPolicies ----------

// ---------- HandleListCloudSecurityGroups ----------

func TestHandleListCloudSecurityGroups_VpcClientFailure(t *testing.T) {
	initSGHandlerTestDB(t)
	defer withAdminToken("test-admin-token")()

	req := newAdminRequest(http.MethodGet,
		"/admin/config/security-group/list?offset=0&limit=20&keyword=test&security_group_id=sg-1,sg-2",
		"")
	w := httptest.NewRecorder()
	HandleListCloudSecurityGroups(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("期望 500，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleListCloudSecurityGroups_DefaultParams_VpcClientFailure(t *testing.T) {
	// 不带任何 query 参数，走默认值分支
	initSGHandlerTestDB(t)
	defer withAdminToken("test-admin-token")()

	req := newAdminRequest(http.MethodGet, "/admin/config/security-group/list", "")
	w := httptest.NewRecorder()
	HandleListCloudSecurityGroups(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("期望 500，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// ---------- HandleDescribeCloudSecurityGroupPolicies ----------

// ---------- HandleGetRequiredSGRules ----------

func TestHandleGetRequiredSGRules_DefaultType(t *testing.T) {
	initSGHandlerTestDB(t)
	defer withAdminToken("test-admin-token")()

	origJSON := ClawproRequiredSGRulesJSON
	defer func() { ClawproRequiredSGRulesJSON = origJSON }()
	ClawproRequiredSGRulesJSON = []byte(`{
		"categories": [
			{"type":"builtin","label":"内置","rule_groups":[
				{"key":"allow_ssh","name":"SSH","rules":[
					{"direction":"ingress","protocol":"TCP","port":"22","cidr_block":"0.0.0.0/0","action":"ACCEPT","description":"SSH"}
				]}
			]},
			{"type":"recommended","label":"推荐","rule_groups":[
				{"key":"allow_rdp","name":"RDP","rules":[]}
			]}
		]
	}`)

	req := newAdminRequest(http.MethodGet, "/admin/config/security-group/required-rules", "")
	w := httptest.NewRecorder()
	HandleGetRequiredSGRules(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	// 默认 type=builtin，只应返回 builtin 分类
	var resp struct {
		Data sgRuleSet `json:"data"`
	}
	// 响应格式依据 jsonOK；若结构不同，退而用字符串断言
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err == nil && len(resp.Data.Categories) > 0 {
		for _, cat := range resp.Data.Categories {
			if cat.Type != "builtin" {
				t.Errorf("默认 type=builtin 时不应出现 type=%s", cat.Type)
			}
		}
	} else if !strings.Contains(w.Body.String(), "builtin") {
		t.Errorf("响应应包含 builtin 分类，实际=%s", w.Body.String())
	}
}

func TestHandleGetRequiredSGRules_TypeRecommended(t *testing.T) {
	initSGHandlerTestDB(t)
	defer withAdminToken("test-admin-token")()

	origJSON := ClawproRequiredSGRulesJSON
	defer func() { ClawproRequiredSGRulesJSON = origJSON }()
	ClawproRequiredSGRulesJSON = []byte(`{
		"categories": [
			{"type":"builtin","label":"内置","rule_groups":[]},
			{"type":"recommended","label":"推荐","rule_groups":[
				{"key":"allow_rdp","name":"RDP","rules":[]}
			]}
		]
	}`)

	req := newAdminRequest(http.MethodGet,
		"/admin/config/security-group/required-rules?type=recommended", "")
	w := httptest.NewRecorder()
	HandleGetRequiredSGRules(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleGetRequiredSGRules_TypeAll(t *testing.T) {
	initSGHandlerTestDB(t)
	defer withAdminToken("test-admin-token")()

	origJSON := ClawproRequiredSGRulesJSON
	defer func() { ClawproRequiredSGRulesJSON = origJSON }()
	ClawproRequiredSGRulesJSON = []byte(`{
		"categories": [
			{"type":"builtin","label":"内置","rule_groups":[]},
			{"type":"recommended","label":"推荐","rule_groups":[]}
		]
	}`)

	req := newAdminRequest(http.MethodGet,
		"/admin/config/security-group/required-rules?type=all", "")
	w := httptest.NewRecorder()
	HandleGetRequiredSGRules(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	// type=all 应同时返回两个分类
	if !strings.Contains(w.Body.String(), "builtin") ||
		!strings.Contains(w.Body.String(), "recommended") {
		t.Errorf("type=all 应同时返回 builtin 和 recommended，实际=%s", w.Body.String())
	}
}

// ---------- HandleCheckSecurityGroupRules ----------

// ==================== 内部函数快速返回分支测试 ====================

// TestRebindAllInstancesToSingleSG_EmptyInstanceIds 验证空 instanceIds 直接返回 nil。
func TestRebindAllInstancesToSingleSG_EmptyInstanceIds(t *testing.T) {
	if err := rebindAllInstancesToSingleSG(context.Background(), nil, "sg-target"); err != nil {
		t.Errorf("空 instanceIds 应返回 nil，实际=%v", err)
	}
	if err := rebindAllInstancesToSingleSG(context.Background(), []string{}, "sg-target"); err != nil {
		t.Errorf("空 instanceIds 应返回 nil，实际=%v", err)
	}
}

// TestAddMissingSGRules_EmptyMissingRules 验证空规则列表直接返回 (0, nil)。
func TestAddMissingSGRules_EmptyMissingRules(t *testing.T) {
	// 传 nil vpcClient 也没关系，函数在 len(missingRules)==0 时提前返回
	n, err := addMissingSGRules(nil, "sg-test", nil)
	if err != nil {
		t.Errorf("空规则列表应返回 nil error，实际=%v", err)
	}
	if n != 0 {
		t.Errorf("空规则列表应返回 0，实际=%d", n)
	}

	n, err = addMissingSGRules(nil, "sg-test", []requiredSGRule{})
	if err != nil {
		t.Errorf("空规则列表应返回 nil error，实际=%v", err)
	}
	if n != 0 {
		t.Errorf("空规则列表应返回 0，实际=%d", n)
	}
}

// ==================== resolveConditionalRules 测试 ====================

// TestResolveConditionalRules_GatewayUIEnabled_KeepsAndReplacesPort 验证当
// GatewayUI 开启时，带 "gateway_ui_enable" 条件的规则组会被保留，且规则中
// 的 {{GATEWAY_UI_PORT}} 端口占位符会被替换为实际端口。
func TestResolveConditionalRules_GatewayUIEnabled_KeepsAndReplacesPort(t *testing.T) {
	// 准备数据库，SiteConfig.GatewayUIEnable=true, GatewayUIPort=8443
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.SiteConfig{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	if err := db.Create(&model.SiteConfig{
		GatewayUIEnable: true,
		GatewayUIPort:   8443,
	}).Error; err != nil {
		t.Fatalf("创建 SiteConfig 失败: %v", err)
	}

	ruleSet := sgRuleSet{
		Categories: []sgRuleCategory{{
			Type: "builtin", Label: "内置",
			RuleGroups: []sgRuleGroup{
				{
					Key: "allow_gateway_ui", Name: "GatewayUI",
					Condition: "gateway_ui_enable",
					Rules: []requiredSGRule{
						{Direction: "ingress", Protocol: "TCP", Port: "{{GATEWAY_UI_PORT}}",
							CidrBlock: "0.0.0.0/0", Action: "ACCEPT", Description: "GatewayUI"},
						{Direction: "ingress", Protocol: "TCP", Port: "22",
							CidrBlock: "0.0.0.0/0", Action: "ACCEPT", Description: "SSH"},
					},
				},
				{
					Key: "allow_ssh", Name: "SSH",
					// 无条件，应保留
					Rules: []requiredSGRule{
						{Direction: "ingress", Protocol: "TCP", Port: "22",
							CidrBlock: "0.0.0.0/0", Action: "ACCEPT", Description: "SSH"},
					},
				},
			},
		}},
	}
	resolveConditionalRules(context.Background(), &ruleSet)

	groups := ruleSet.Categories[0].RuleGroups
	if len(groups) != 2 {
		t.Fatalf("期望保留 2 个规则组，实际=%d", len(groups))
	}
	// 第一个规则组：端口占位符被替换为 "8443"
	if got := groups[0].Rules[0].Port; got != "8443" {
		t.Errorf("{{GATEWAY_UI_PORT}} 应被替换为 8443，实际=%s", got)
	}
	// 非占位符的端口不应被修改
	if got := groups[0].Rules[1].Port; got != "22" {
		t.Errorf("非占位符端口不应被修改，实际=%s", got)
	}
	// Condition 应被清除
	if groups[0].Condition != "" {
		t.Errorf("Condition 应被清空，实际=%s", groups[0].Condition)
	}
}

// TestResolveConditionalRules_GatewayUIDisabled_RemovesGroup 验证当 GatewayUI
// 关闭时，带条件的规则组被移除；无条件的规则组保留。
func TestResolveConditionalRules_GatewayUIDisabled_RemovesGroup(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.SiteConfig{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	// GatewayUIEnable=false
	if err := db.Create(&model.SiteConfig{GatewayUIEnable: false}).Error; err != nil {
		t.Fatalf("创建 SiteConfig 失败: %v", err)
	}

	ruleSet := sgRuleSet{
		Categories: []sgRuleCategory{{
			Type: "builtin", Label: "内置",
			RuleGroups: []sgRuleGroup{
				{Key: "gw", Name: "Gateway", Condition: "gateway_ui_enable",
					Rules: []requiredSGRule{{Port: "{{GATEWAY_UI_PORT}}"}}},
				{Key: "ssh", Name: "SSH",
					Rules: []requiredSGRule{{Port: "22"}}},
			},
		}},
	}
	resolveConditionalRules(context.Background(), &ruleSet)

	groups := ruleSet.Categories[0].RuleGroups
	if len(groups) != 1 {
		t.Fatalf("期望只保留 1 个规则组（无条件的 SSH），实际=%d", len(groups))
	}
	if groups[0].Key != "ssh" {
		t.Errorf("期望保留 ssh 规则组，实际=%s", groups[0].Key)
	}
}

// TestResolveConditionalRules_UnknownCondition_Skipped 验证未知条件的规则组会被跳过。
func TestResolveConditionalRules_UnknownCondition_Skipped(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.SiteConfig{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	if err := db.Create(&model.SiteConfig{}).Error; err != nil {
		t.Fatalf("创建 SiteConfig 失败: %v", err)
	}

	ruleSet := sgRuleSet{
		Categories: []sgRuleCategory{{
			Type: "builtin", Label: "内置",
			RuleGroups: []sgRuleGroup{
				{Key: "known", Name: "known", Condition: "",
					Rules: []requiredSGRule{{Port: "22"}}},
				{Key: "unknown", Name: "unknown", Condition: "some_unknown_condition",
					Rules: []requiredSGRule{{Port: "80"}}},
			},
		}},
	}
	resolveConditionalRules(context.Background(), &ruleSet)

	groups := ruleSet.Categories[0].RuleGroups
	if len(groups) != 1 {
		t.Fatalf("期望只保留 1 个规则组，实际=%d", len(groups))
	}
	if groups[0].Key != "known" {
		t.Errorf("期望保留无条件的规则组，实际=%s", groups[0].Key)
	}
}

// TestResolveConditionalRules_GatewayUIPrivateMode_DropsGroup 验证当 GatewayUI
// 开启且 addr_type=private 时，allow_gateway_ui 规则组应被整组剔除（与 enable=false 等价）。
//
// 背景：addr_type=private 表示用户走 VPC 内网通道访问 Gateway UI，不需要在 SG 上对公网
// 放通端口。如果不剔除，必需规则会注入 0.0.0.0/0:port 入站放通规则到所有 ACTIVE SG，
// 且任何 refresh 都会还原（用户手动删了也回来），即"无法关闭"问题。
func TestResolveConditionalRules_GatewayUIPrivateMode_DropsGroup(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&model.SiteConfig{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	// GatewayUIEnable=true + Port>0 + addr_type=private（关键场景）
	if err := db.Create(&model.SiteConfig{
		GatewayUIEnable:   true,
		GatewayUIPort:     8443,
		GatewayUIAddrType: "private",
	}).Error; err != nil {
		t.Fatalf("创建 SiteConfig 失败: %v", err)
	}

	ruleSet := sgRuleSet{
		Categories: []sgRuleCategory{{
			Type: "recommended", Label: "推荐",
			RuleGroups: []sgRuleGroup{
				{Key: "allow_gateway_ui", Name: "GatewayUI", Condition: "gateway_ui_enable",
					Rules: []requiredSGRule{{Port: "{{GATEWAY_UI_PORT}}", CidrBlock: "0.0.0.0/0"}}},
				{Key: "allow_ssh", Name: "SSH",
					Rules: []requiredSGRule{{Port: "22", CidrBlock: "0.0.0.0/0"}}},
			},
		}},
	}
	resolveConditionalRules(context.Background(), &ruleSet)

	groups := ruleSet.Categories[0].RuleGroups
	if len(groups) != 1 {
		t.Fatalf("期望只保留 1 个规则组（无条件的 SSH），私网模式下 allow_gateway_ui 应被剔除，实际=%d", len(groups))
	}
	if groups[0].Key != "allow_ssh" {
		t.Errorf("期望保留 allow_ssh 规则组，实际=%s", groups[0].Key)
	}
}

// TestResolveConditionalRules_GatewayUIPublicMode_KeepsGroup 验证当 GatewayUI
// 开启且 addr_type=public（或为空，向后兼容）时，allow_gateway_ui 规则组应被保留并替换占位符。
func TestResolveConditionalRules_GatewayUIPublicMode_KeepsGroup(t *testing.T) {
	cases := []struct {
		name     string
		addrType string
	}{
		{"显式public", "public"},
		{"空值视为非private", ""}, // 向后兼容：未升级到带 addr_type 的存量数据
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
			if err != nil {
				t.Fatalf("打开内存数据库失败: %v", err)
			}
			if err := db.AutoMigrate(&model.SiteConfig{}); err != nil {
				t.Fatalf("迁移失败: %v", err)
			}
			t.Cleanup(model.UseDBForTest(db))
			if err := db.Create(&model.SiteConfig{
				GatewayUIEnable:   true,
				GatewayUIPort:     8443,
				GatewayUIAddrType: tc.addrType,
			}).Error; err != nil {
				t.Fatalf("创建 SiteConfig 失败: %v", err)
			}

			ruleSet := sgRuleSet{
				Categories: []sgRuleCategory{{
					Type: "recommended", Label: "推荐",
					RuleGroups: []sgRuleGroup{
						{Key: "allow_gateway_ui", Name: "GatewayUI", Condition: "gateway_ui_enable",
							Rules: []requiredSGRule{{Port: "{{GATEWAY_UI_PORT}}", CidrBlock: "0.0.0.0/0"}}},
					},
				}},
			}
			resolveConditionalRules(context.Background(), &ruleSet)

			groups := ruleSet.Categories[0].RuleGroups
			if len(groups) != 1 {
				t.Fatalf("非 private 模式应保留 allow_gateway_ui，实际剩=%d", len(groups))
			}
			if got := groups[0].Rules[0].Port; got != "8443" {
				t.Errorf("端口占位符应被替换为 8443，实际=%s", got)
			}
		})
	}
}

// ==================== resolveVpcCidr 测试 ====================

// TestResolveVpcCidr_NoVpcId 验证 SiteConfig.VpcId 为空时返回空字符串。
func TestResolveVpcCidr_NoVpcId(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.SiteConfig{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	if err := db.Create(&model.SiteConfig{}).Error; err != nil {
		t.Fatalf("创建 SiteConfig 失败: %v", err)
	}

	if got := resolveVpcCidr(context.Background()); got != "" {
		t.Errorf("VpcId 为空应返回空字符串，实际=%s", got)
	}
}

// TestResolveVpcCidr_VpcIdSet_NoCredentials 验证 VpcId 已设置但无凭据时，
// newVpcClient 失败并返回空字符串（不 panic）。
func TestResolveVpcCidr_VpcIdSet_NoCredentials(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.SiteConfig{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	if err := db.Create(&model.SiteConfig{VpcId: "vpc-xxx"}).Error; err != nil {
		t.Fatalf("创建 SiteConfig 失败: %v", err)
	}

	// 无 CVMSecretId/Key，newVpcClient 失败，返回 ""
	if got := resolveVpcCidr(context.Background()); got != "" {
		t.Errorf("无凭据时应返回空字符串，实际=%s", got)
	}
}

// ==================== fake 实现（基于接口） ====================

// fakeSGPolicyClient 同时实现 sgPolicyQuerier 和 sgPolicyWriter。
type fakeSGPolicyClient struct {
	describeResp *vpc.DescribeSecurityGroupPoliciesResponse
	describeErr  error
	// createSeq 用于控制每次调用的返回；如果用尽则复用最后一项
	createSeq []error
	createIdx int
	// createCalls 记录每次传入的请求，便于断言
	createCalls []*vpc.CreateSecurityGroupPoliciesRequest
}

func (f *fakeSGPolicyClient) DescribeSecurityGroupPolicies(request *vpc.DescribeSecurityGroupPoliciesRequest) (*vpc.DescribeSecurityGroupPoliciesResponse, error) {
	if f.describeErr != nil {
		return nil, f.describeErr
	}
	if f.describeResp != nil {
		return f.describeResp, nil
	}
	// 默认空返回
	return &vpc.DescribeSecurityGroupPoliciesResponse{
		Response: &vpc.DescribeSecurityGroupPoliciesResponseParams{
			SecurityGroupPolicySet: &vpc.SecurityGroupPolicySet{},
		},
	}, nil
}

func (f *fakeSGPolicyClient) CreateSecurityGroupPolicies(request *vpc.CreateSecurityGroupPoliciesRequest) (*vpc.CreateSecurityGroupPoliciesResponse, error) {
	f.createCalls = append(f.createCalls, request)
	if len(f.createSeq) == 0 {
		return &vpc.CreateSecurityGroupPoliciesResponse{}, nil
	}
	var err error
	if f.createIdx < len(f.createSeq) {
		err = f.createSeq[f.createIdx]
	} else {
		err = f.createSeq[len(f.createSeq)-1]
	}
	f.createIdx++
	if err != nil {
		return nil, err
	}
	return &vpc.CreateSecurityGroupPoliciesResponse{}, nil
}

// fakeCVMClient 实现 cvmSgBinder。
type fakeCVMClient struct {
	// associateErrs 按顺序返回 Associate 的 error；用尽后返回 nil
	associateErrs []error
	associateIdx  int
	associated    []string // 记录已绑定的 instanceId
	// disassociateErrs 按顺序返回 Disassociate 的 error
	disassociateErrs []error
	disassociateIdx  int
	disassociated    []string // 记录已解绑的 instanceId+sg
}

func (f *fakeCVMClient) AssociateSecurityGroups(req *cvm.AssociateSecurityGroupsRequest) (*cvm.AssociateSecurityGroupsResponse, error) {
	if req != nil && len(req.InstanceIds) > 0 && req.InstanceIds[0] != nil {
		f.associated = append(f.associated, *req.InstanceIds[0])
	}
	var err error
	if f.associateIdx < len(f.associateErrs) {
		err = f.associateErrs[f.associateIdx]
	}
	f.associateIdx++
	if err != nil {
		return nil, err
	}
	return &cvm.AssociateSecurityGroupsResponse{}, nil
}

func (f *fakeCVMClient) DisassociateSecurityGroups(req *cvm.DisassociateSecurityGroupsRequest) (*cvm.DisassociateSecurityGroupsResponse, error) {
	if req != nil && len(req.InstanceIds) > 0 && req.InstanceIds[0] != nil && len(req.SecurityGroupIds) > 0 && req.SecurityGroupIds[0] != nil {
		f.disassociated = append(f.disassociated, *req.InstanceIds[0]+"|"+*req.SecurityGroupIds[0])
	}
	var err error
	if f.disassociateIdx < len(f.disassociateErrs) {
		err = f.disassociateErrs[f.disassociateIdx]
	}
	f.disassociateIdx++
	if err != nil {
		return nil, err
	}
	return &cvm.DisassociateSecurityGroupsResponse{}, nil
}

// resetSGPolicyCreateSeqFake 初始化一组"前 N-1 次失败，第 N 次成功"的序列。
func failThenOk(failCount int) []error {
	seq := make([]error, failCount+1)
	for i := 0; i < failCount; i++ {
		seq[i] = fmt.Errorf("mock api error #%d", i+1)
	}
	seq[failCount] = nil
	return seq
}

// commonPolicySet 构造一个带 ingress/egress 策略的返回体，用于 describe 结果匹配。
func commonPolicySet(ingress, egress []*vpc.SecurityGroupPolicy) *vpc.DescribeSecurityGroupPoliciesResponse {
	return &vpc.DescribeSecurityGroupPoliciesResponse{
		Response: &vpc.DescribeSecurityGroupPoliciesResponseParams{
			SecurityGroupPolicySet: &vpc.SecurityGroupPolicySet{
				Ingress: ingress,
				Egress:  egress,
			},
		},
	}
}

// pickBuiltinIngressRules 从内置规则集合中取出所有 ingress 规则，用于构造"全部匹配"的 describe 响应。
func pickBuiltinIngressRules(t *testing.T) []requiredSGRule {
	t.Helper()
	// 用最小化的内置规则集，避免与大 JSON 耦合
	ClawproRequiredSGRulesJSON = []byte(`{
		"categories": [
			{"type":"builtin","label":"内置","rule_groups":[
				{"key":"allow_ssh","name":"SSH","rules":[
					{"direction":"ingress","protocol":"TCP","port":"22","cidr_block":"0.0.0.0/0","action":"ACCEPT","description":"SSH"}
				]}
			]},
			{"type":"recommended","label":"推荐","rule_groups":[
				{"key":"allow_http","name":"HTTP","rules":[
					{"direction":"ingress","protocol":"TCP","port":"80","cidr_block":"0.0.0.0/0","action":"ACCEPT","description":"HTTP"}
				]}
			]}
		]
	}`)
	ruleSet := clawproRequiredRuleSet()
	resolveConditionalRules(context.Background(), &ruleSet)
	var all []requiredSGRule
	for _, cat := range ruleSet.Categories {
		for _, g := range cat.RuleGroups {
			all = append(all, g.Rules...)
		}
	}
	return all
}

// buildPolicyFromRule 把 requiredSGRule 转成腾讯云返回的 policy 格式，用于构造"已匹配"响应。
func buildPolicyFromRule(rule requiredSGRule) *vpc.SecurityGroupPolicy {
	p := &vpc.SecurityGroupPolicy{
		Protocol: common.StringPtr(rule.Protocol),
		Port:     common.StringPtr(rule.Port),
		Action:   common.StringPtr(rule.Action),
	}
	if rule.CidrBlock != "" {
		p.CidrBlock = common.StringPtr(rule.CidrBlock)
	}
	if rule.Ipv6Cidr != "" {
		p.Ipv6CidrBlock = common.StringPtr(rule.Ipv6Cidr)
	}
	return p
}

// ==================== checkMissingSGRules 深度覆盖 ====================

func TestCheckMissingSGRules_DescribeError(t *testing.T) {
	initSGHandlerTestDB(t)
	fake := &fakeSGPolicyClient{describeErr: errors.New("mock describe error")}
	_, err := checkMissingSGRules(context.Background(), fake, "sg-x")
	if err == nil {
		t.Fatalf("期望错误，实际 nil")
	}
	if !strings.Contains(err.Error(), "查询安全组规则失败") {
		t.Errorf("错误应包含 '查询安全组规则失败'，实际=%v", err)
	}
}

func TestCheckMissingSGRules_AllRulesMissing(t *testing.T) {
	initSGHandlerTestDB(t)
	origJSON := ClawproRequiredSGRulesJSON
	defer func() { ClawproRequiredSGRulesJSON = origJSON }()
	_ = pickBuiltinIngressRules(t) // 设置规则集（含 1 条 builtin + 1 条 recommended ingress）

	// describe 返回空 policy set -> recommended 规则全部缺失
	fake := &fakeSGPolicyClient{
		describeResp: commonPolicySet(nil, nil),
	}
	missing, err := checkMissingSGRules(context.Background(), fake, "sg-x")
	if err != nil {
		t.Fatalf("不应返回错误，实际=%v", err)
	}
	// checkMissingSGRules 仅检查 recommended 分类，builtin 规则应被忽略
	if len(missing) < 1 {
		t.Errorf("应返回 recommended 缺失规则，实际=%d", len(missing))
	}
	for _, r := range missing {
		if r.Port == "22" {
			t.Errorf("不应返回 builtin 规则（port=22），实际=%+v", r)
		}
	}
}

func TestCheckMissingSGRules_AllRulesMatched(t *testing.T) {
	initSGHandlerTestDB(t)
	origJSON := ClawproRequiredSGRulesJSON
	defer func() { ClawproRequiredSGRulesJSON = origJSON }()
	allRules := pickBuiltinIngressRules(t)

	// 把所有规则构造成 policy 放进 Ingress -> 全部匹配
	var ingress []*vpc.SecurityGroupPolicy
	for _, r := range allRules {
		if r.Direction == "ingress" {
			ingress = append(ingress, buildPolicyFromRule(r))
		}
	}
	fake := &fakeSGPolicyClient{
		describeResp: commonPolicySet(ingress, nil),
	}
	missing, err := checkMissingSGRules(context.Background(), fake, "sg-x")
	if err != nil {
		t.Fatalf("不应返回错误，实际=%v", err)
	}
	if len(missing) != 0 {
		t.Errorf("应返回 0 条缺失，实际=%d (%+v)", len(missing), missing)
	}
}

func TestCheckMissingSGRules_EgressDirection(t *testing.T) {
	initSGHandlerTestDB(t)
	origJSON := ClawproRequiredSGRulesJSON
	defer func() { ClawproRequiredSGRulesJSON = origJSON }()
	// checkMissingSGRules 只检查 recommended 分类，故此处使用 recommended 分类的 egress 规则
	ClawproRequiredSGRulesJSON = []byte(`{
		"categories": [{"type":"recommended","label":"推荐","rule_groups":[
			{"key":"allow_out","name":"Out","rules":[
				{"direction":"egress","protocol":"ALL","port":"ALL","cidr_block":"0.0.0.0/0","action":"ACCEPT","description":"全部允许出"}
			]}
		]}]
	}`)

	// 构造一条 egress policy 完全匹配
	fake := &fakeSGPolicyClient{
		describeResp: commonPolicySet(nil, []*vpc.SecurityGroupPolicy{
			{
				Protocol:  common.StringPtr("ALL"),
				Port:      common.StringPtr("ALL"),
				CidrBlock: common.StringPtr("0.0.0.0/0"),
				Action:    common.StringPtr("ACCEPT"),
			},
		}),
	}
	missing, err := checkMissingSGRules(context.Background(), fake, "sg-x")
	if err != nil {
		t.Fatalf("不应返回错误，实际=%v", err)
	}
	if len(missing) != 0 {
		t.Errorf("egress 匹配应无缺失，实际=%d (%+v)", len(missing), missing)
	}
}

func TestCheckMissingSGRules_NilResponse(t *testing.T) {
	initSGHandlerTestDB(t)
	origJSON := ClawproRequiredSGRulesJSON
	defer func() { ClawproRequiredSGRulesJSON = origJSON }()
	_ = pickBuiltinIngressRules(t)

	// describeResp = nil Response
	fake := &fakeSGPolicyClient{
		describeResp: &vpc.DescribeSecurityGroupPoliciesResponse{Response: nil},
	}
	missing, err := checkMissingSGRules(context.Background(), fake, "sg-x")
	if err != nil {
		t.Fatalf("不应返回错误，实际=%v", err)
	}
	if len(missing) == 0 {
		t.Errorf("Response 为 nil 时应全部缺失，实际=%d", len(missing))
	}
}

// ==================== checkMissingRecommendedSGRules 深度覆盖 ====================

func TestCheckMissingRecommendedSGRules_DescribeError(t *testing.T) {
	initSGHandlerTestDB(t)
	fake := &fakeSGPolicyClient{describeErr: errors.New("mock error")}
	_, err := checkMissingRecommendedSGRules(context.Background(), fake, "sg-x")
	if err == nil || !strings.Contains(err.Error(), "查询安全组规则失败") {
		t.Errorf("应返回查询失败错误，实际=%v", err)
	}
}

func TestCheckMissingRecommendedSGRules_OnlyRecommendedChecked(t *testing.T) {
	initSGHandlerTestDB(t)
	origJSON := ClawproRequiredSGRulesJSON
	defer func() { ClawproRequiredSGRulesJSON = origJSON }()
	ClawproRequiredSGRulesJSON = []byte(`{
		"categories": [
			{"type":"builtin","label":"内置","rule_groups":[
				{"key":"allow_ssh","name":"SSH","rules":[
					{"direction":"ingress","protocol":"TCP","port":"22","cidr_block":"0.0.0.0/0","action":"ACCEPT","description":"SSH"}
				]}
			]},
			{"type":"recommended","label":"推荐","rule_groups":[
				{"key":"allow_http","name":"HTTP","rules":[
					{"direction":"ingress","protocol":"TCP","port":"80","cidr_block":"0.0.0.0/0","action":"ACCEPT","description":"HTTP"}
				]}
			]}
		]
	}`)

	// 空 policy -> builtin(SSH) 应被忽略，只有 recommended(HTTP) 缺失
	fake := &fakeSGPolicyClient{describeResp: commonPolicySet(nil, nil)}
	missing, err := checkMissingRecommendedSGRules(context.Background(), fake, "sg-x")
	if err != nil {
		t.Fatalf("不应返回错误，实际=%v", err)
	}
	if len(missing) != 1 {
		t.Fatalf("应只有 1 条推荐规则缺失，实际=%d (%+v)", len(missing), missing)
	}
	if missing[0].Port != "80" {
		t.Errorf("应只缺失 HTTP(80)，实际 port=%s", missing[0].Port)
	}
}

func TestCheckMissingRecommendedSGRules_MatchedNoMissing(t *testing.T) {
	initSGHandlerTestDB(t)
	origJSON := ClawproRequiredSGRulesJSON
	defer func() { ClawproRequiredSGRulesJSON = origJSON }()
	ClawproRequiredSGRulesJSON = []byte(`{
		"categories": [
			{"type":"recommended","label":"推荐","rule_groups":[
				{"key":"allow_http","name":"HTTP","rules":[
					{"direction":"ingress","protocol":"TCP","port":"80","cidr_block":"0.0.0.0/0","action":"ACCEPT","description":"HTTP"}
				]}
			]}
		]
	}`)
	fake := &fakeSGPolicyClient{
		describeResp: commonPolicySet([]*vpc.SecurityGroupPolicy{
			{
				Protocol:  common.StringPtr("tcp"), // 大小写不敏感测试
				Port:      common.StringPtr("80"),
				CidrBlock: common.StringPtr("0.0.0.0/0"),
				Action:    common.StringPtr("accept"),
			},
		}, nil),
	}
	missing, err := checkMissingRecommendedSGRules(context.Background(), fake, "sg-x")
	if err != nil {
		t.Fatalf("不应返回错误，实际=%v", err)
	}
	if len(missing) != 0 {
		t.Errorf("应全部匹配，实际缺失=%d", len(missing))
	}
}

// ==================== sgPolicyMatchesRule 覆盖 ====================

func TestSGPolicyMatchesRule_NilPolicy(t *testing.T) {
	if sgPolicyMatchesRule(nil, requiredSGRule{Protocol: "TCP"}) {
		t.Error("nil policy 应不匹配")
	}
}

func TestSGPolicyMatchesRule_ProtocolMismatch(t *testing.T) {
	p := &vpc.SecurityGroupPolicy{Protocol: common.StringPtr("UDP")}
	if sgPolicyMatchesRule(p, requiredSGRule{Protocol: "TCP"}) {
		t.Error("协议不一致应不匹配")
	}
}

func TestSGPolicyMatchesRule_PortMismatch(t *testing.T) {
	p := &vpc.SecurityGroupPolicy{
		Protocol: common.StringPtr("TCP"),
		Port:     common.StringPtr("80"),
	}
	if sgPolicyMatchesRule(p, requiredSGRule{Protocol: "TCP", Port: "22"}) {
		t.Error("端口不一致应不匹配")
	}
}

func TestSGPolicyMatchesRule_ActionMismatch(t *testing.T) {
	p := &vpc.SecurityGroupPolicy{
		Protocol: common.StringPtr("TCP"),
		Port:     common.StringPtr("22"),
		Action:   common.StringPtr("DROP"),
	}
	if sgPolicyMatchesRule(p, requiredSGRule{Protocol: "TCP", Port: "22", Action: "ACCEPT"}) {
		t.Error("动作不一致应不匹配")
	}
}

func TestSGPolicyMatchesRule_CidrMismatch(t *testing.T) {
	p := &vpc.SecurityGroupPolicy{
		Protocol:  common.StringPtr("TCP"),
		Port:      common.StringPtr("22"),
		Action:    common.StringPtr("ACCEPT"),
		CidrBlock: common.StringPtr("10.0.0.0/8"),
	}
	if sgPolicyMatchesRule(p, requiredSGRule{
		Protocol: "TCP", Port: "22", Action: "ACCEPT", CidrBlock: "0.0.0.0/0",
	}) {
		t.Error("CIDR 不一致应不匹配")
	}
}

func TestSGPolicyMatchesRule_NilCidrButRuleRequires(t *testing.T) {
	p := &vpc.SecurityGroupPolicy{
		Protocol: common.StringPtr("TCP"),
		Port:     common.StringPtr("22"),
		Action:   common.StringPtr("ACCEPT"),
		// CidrBlock 为 nil
	}
	if sgPolicyMatchesRule(p, requiredSGRule{
		Protocol: "TCP", Port: "22", Action: "ACCEPT", CidrBlock: "0.0.0.0/0",
	}) {
		t.Error("policy.CidrBlock 为 nil 但 rule 要求 CIDR 时应不匹配")
	}
}

func TestSGPolicyMatchesRule_Ipv6Mismatch(t *testing.T) {
	p := &vpc.SecurityGroupPolicy{
		Protocol:      common.StringPtr("TCP"),
		Port:          common.StringPtr("22"),
		Action:        common.StringPtr("ACCEPT"),
		Ipv6CidrBlock: common.StringPtr("fe80::/10"),
	}
	if sgPolicyMatchesRule(p, requiredSGRule{
		Protocol: "TCP", Port: "22", Action: "ACCEPT", Ipv6Cidr: "::/0",
	}) {
		t.Error("IPv6 CIDR 不一致应不匹配")
	}
}

func TestSGPolicyMatchesRule_Ipv6NilButRuleRequires(t *testing.T) {
	p := &vpc.SecurityGroupPolicy{
		Protocol: common.StringPtr("TCP"),
		Port:     common.StringPtr("22"),
		Action:   common.StringPtr("ACCEPT"),
	}
	if sgPolicyMatchesRule(p, requiredSGRule{
		Protocol: "TCP", Port: "22", Action: "ACCEPT", Ipv6Cidr: "::/0",
	}) {
		t.Error("policy.Ipv6CidrBlock 为 nil 但 rule 要求时应不匹配")
	}
}

func TestSGPolicyMatchesRule_IPv6Match(t *testing.T) {
	p := &vpc.SecurityGroupPolicy{
		Protocol:      common.StringPtr("TCP"),
		Port:          common.StringPtr("22"),
		Action:        common.StringPtr("ACCEPT"),
		Ipv6CidrBlock: common.StringPtr("::/0"),
	}
	if !sgPolicyMatchesRule(p, requiredSGRule{
		Protocol: "TCP", Port: "22", Action: "ACCEPT", Ipv6Cidr: "::/0",
	}) {
		t.Error("IPv6 完全匹配应通过")
	}
}

// ==================== addMissingSGRules 深度覆盖 ====================

func withZeroRetryInterval() func() {
	orig := addMissingSGRulesRetryInterval
	addMissingSGRulesRetryInterval = 0
	return func() { addMissingSGRulesRetryInterval = orig }
}

func TestAddMissingSGRules_IngressOnly_Success(t *testing.T) {
	defer withZeroRetryInterval()()
	fake := &fakeSGPolicyClient{}
	n, err := addMissingSGRules(fake, "sg-x", []requiredSGRule{
		{Direction: "ingress", Protocol: "TCP", Port: "22", CidrBlock: "0.0.0.0/0", Action: "ACCEPT", Description: "SSH"},
	})
	if err != nil {
		t.Fatalf("不应返回错误，实际=%v", err)
	}
	if n != 1 {
		t.Errorf("应添加 1 条规则，实际=%d", n)
	}
	if len(fake.createCalls) != 1 {
		t.Errorf("应调用 1 次 Create（仅 ingress），实际=%d", len(fake.createCalls))
	}
}

func TestAddMissingSGRules_EgressOnly_Success(t *testing.T) {
	defer withZeroRetryInterval()()
	fake := &fakeSGPolicyClient{}
	n, err := addMissingSGRules(fake, "sg-x", []requiredSGRule{
		{Direction: "egress", Protocol: "ALL", Port: "ALL", CidrBlock: "0.0.0.0/0", Action: "ACCEPT", Description: "out"},
	})
	if err != nil {
		t.Fatalf("不应返回错误，实际=%v", err)
	}
	if n != 1 {
		t.Errorf("应添加 1 条，实际=%d", n)
	}
	if len(fake.createCalls) != 1 {
		t.Errorf("应调用 1 次 Create（仅 egress），实际=%d", len(fake.createCalls))
	}
}

func TestAddMissingSGRules_IngressAndEgress_Success(t *testing.T) {
	defer withZeroRetryInterval()()
	fake := &fakeSGPolicyClient{}
	n, err := addMissingSGRules(fake, "sg-x", []requiredSGRule{
		{Direction: "ingress", Protocol: "TCP", Port: "22", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"},
		{Direction: "egress", Protocol: "ALL", Port: "ALL", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"},
		{Direction: "ingress", Protocol: "TCP", Port: "443", Ipv6Cidr: "::/0", Action: "ACCEPT"},
	})
	if err != nil {
		t.Fatalf("不应返回错误，实际=%v", err)
	}
	if n != 3 {
		t.Errorf("应返回 3，实际=%d", n)
	}
	if len(fake.createCalls) != 2 {
		t.Errorf("应调用 2 次 Create（ingress+egress 分开），实际=%d", len(fake.createCalls))
	}
}

func TestAddMissingSGRules_RetrySucceed(t *testing.T) {
	defer withZeroRetryInterval()()
	// 第 1 次失败，第 2 次成功
	fake := &fakeSGPolicyClient{createSeq: failThenOk(1)}
	n, err := addMissingSGRules(fake, "sg-x", []requiredSGRule{
		{Direction: "ingress", Protocol: "TCP", Port: "22", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"},
	})
	if err != nil {
		t.Fatalf("重试成功应返回 nil，实际=%v", err)
	}
	if n != 1 {
		t.Errorf("应返回 1，实际=%d", n)
	}
	if len(fake.createCalls) != 2 {
		t.Errorf("应调用 2 次（第 1 次失败重试），实际=%d", len(fake.createCalls))
	}
}

func TestAddMissingSGRules_IngressFailAfterMaxRetry(t *testing.T) {
	defer withZeroRetryInterval()()
	// 连续 3 次都失败
	fake := &fakeSGPolicyClient{createSeq: []error{
		errors.New("e1"), errors.New("e2"), errors.New("e3"),
	}}
	n, err := addMissingSGRules(fake, "sg-x", []requiredSGRule{
		{Direction: "ingress", Protocol: "TCP", Port: "22", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"},
	})
	if err == nil {
		t.Fatalf("应返回错误")
	}
	if n != 0 {
		t.Errorf("失败时应返回 0，实际=%d", n)
	}
	if !strings.Contains(err.Error(), "添加缺失入站安全组规则失败") {
		t.Errorf("错误信息应指明入站失败，实际=%v", err)
	}
}

func TestAddMissingSGRules_EgressFailAfterMaxRetry(t *testing.T) {
	defer withZeroRetryInterval()()
	// 只有 egress 且全部失败
	fake := &fakeSGPolicyClient{createSeq: []error{
		errors.New("e1"), errors.New("e2"), errors.New("e3"),
	}}
	_, err := addMissingSGRules(fake, "sg-x", []requiredSGRule{
		{Direction: "egress", Protocol: "ALL", Port: "ALL", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"},
	})
	if err == nil {
		t.Fatalf("应返回错误")
	}
	if !strings.Contains(err.Error(), "添加缺失出站安全组规则失败") {
		t.Errorf("错误信息应指明出站失败，实际=%v", err)
	}
}

// ==================== rebindAllInstancesToSingleSG 深度覆盖 ====================

// overrideDescribeSGFn 临时替换 describeInstancesSecurityGroupsFn。
func overrideDescribeSGFn(fn func(context.Context, []string) (map[string][]string, error)) func() {
	orig := describeInstancesSecurityGroupsFn
	describeInstancesSecurityGroupsFn = fn
	return func() { describeInstancesSecurityGroupsFn = orig }
}

// overrideNewCVMClientFn 临时替换 newCVMClientFn。
func overrideNewCVMClientFn(fn func(context.Context) (cvmSgBinder, error)) func() {
	orig := newCVMClientFn
	newCVMClientFn = fn
	return func() { newCVMClientFn = orig }
}

func TestRebindAllInstancesToSingleSG_DescribeFails(t *testing.T) {
	defer overrideDescribeSGFn(func(_ context.Context, ids []string) (map[string][]string, error) {
		return nil, errors.New("mock describe error")
	})()
	err := rebindAllInstancesToSingleSG(context.Background(), []string{"ins-1"}, "sg-target")
	if err == nil || !strings.Contains(err.Error(), "查询实例安全组列表失败") {
		t.Errorf("应返回查询失败错误，实际=%v", err)
	}
}

func TestRebindAllInstancesToSingleSG_NewCVMClientFails(t *testing.T) {
	defer overrideDescribeSGFn(func(_ context.Context, ids []string) (map[string][]string, error) {
		return map[string][]string{"ins-1": {"sg-old"}}, nil
	})()
	defer overrideNewCVMClientFn(func(context.Context) (cvmSgBinder, error) {
		return nil, errors.New("mock cvm client error")
	})()
	err := rebindAllInstancesToSingleSG(context.Background(), []string{"ins-1"}, "sg-target")
	if err == nil || !strings.Contains(err.Error(), "创建 CVM 客户端失败") {
		t.Errorf("应返回 CVM 创建失败错误，实际=%v", err)
	}
}

func TestRebindAllInstancesToSingleSG_AlreadySingleTargetSG(t *testing.T) {
	// 实例已经只绑定目标 SG，应跳过
	defer overrideDescribeSGFn(func(_ context.Context, ids []string) (map[string][]string, error) {
		return map[string][]string{"ins-1": {"sg-target"}}, nil
	})()
	fc := &fakeCVMClient{}
	defer overrideNewCVMClientFn(func(context.Context) (cvmSgBinder, error) { return fc, nil })()

	err := rebindAllInstancesToSingleSG(context.Background(), []string{"ins-1"}, "sg-target")
	if err != nil {
		t.Errorf("不应返回错误，实际=%v", err)
	}
	if len(fc.associated) != 0 {
		t.Errorf("已绑定目标 SG 时不应再绑定，实际=%+v", fc.associated)
	}
	if len(fc.disassociated) != 0 {
		t.Errorf("已绑定目标 SG 时不应解绑，实际=%+v", fc.disassociated)
	}
}

func TestRebindAllInstancesToSingleSG_AlreadyBoundButHasOthers(t *testing.T) {
	// 已绑定目标 SG 但还绑了其他 SG -> 不再 associate，但需要解绑其他
	defer overrideDescribeSGFn(func(_ context.Context, ids []string) (map[string][]string, error) {
		return map[string][]string{"ins-1": {"sg-target", "sg-old-1", "sg-old-2"}}, nil
	})()
	fc := &fakeCVMClient{}
	defer overrideNewCVMClientFn(func(context.Context) (cvmSgBinder, error) { return fc, nil })()

	err := rebindAllInstancesToSingleSG(context.Background(), []string{"ins-1"}, "sg-target")
	if err != nil {
		t.Errorf("不应返回错误，实际=%v", err)
	}
	if len(fc.associated) != 0 {
		t.Errorf("目标已绑定时不应 associate，实际=%+v", fc.associated)
	}
	if len(fc.disassociated) != 2 {
		t.Errorf("应解绑 2 个旧 SG，实际=%+v", fc.disassociated)
	}
}

func TestRebindAllInstancesToSingleSG_NewTargetThenDisassociateOld(t *testing.T) {
	// 典型：先绑新，再解绑旧
	defer overrideDescribeSGFn(func(_ context.Context, ids []string) (map[string][]string, error) {
		return map[string][]string{"ins-1": {"sg-old-1", "sg-old-2"}}, nil
	})()
	fc := &fakeCVMClient{}
	defer overrideNewCVMClientFn(func(context.Context) (cvmSgBinder, error) { return fc, nil })()

	err := rebindAllInstancesToSingleSG(context.Background(), []string{"ins-1"}, "sg-target")
	if err != nil {
		t.Errorf("不应返回错误，实际=%v", err)
	}
	if len(fc.associated) != 1 || fc.associated[0] != "ins-1" {
		t.Errorf("应 associate ins-1 一次，实际=%+v", fc.associated)
	}
	if len(fc.disassociated) != 2 {
		t.Errorf("应解绑 2 个旧 SG，实际=%+v", fc.disassociated)
	}
}

func TestRebindAllInstancesToSingleSG_AssociateFailSkipInstance(t *testing.T) {
	// associate 失败 -> 跳过该实例，不解绑，lastErr 被保留
	defer overrideDescribeSGFn(func(_ context.Context, ids []string) (map[string][]string, error) {
		return map[string][]string{
			"ins-1": {"sg-old-1"},
			"ins-2": {"sg-target"}, // 这个无需 associate，直接跳过
		}, nil
	})()
	fc := &fakeCVMClient{
		associateErrs: []error{errors.New("mock associate error")},
	}
	defer overrideNewCVMClientFn(func(context.Context) (cvmSgBinder, error) { return fc, nil })()

	err := rebindAllInstancesToSingleSG(context.Background(), []string{"ins-1", "ins-2"}, "sg-target")
	if err == nil {
		t.Errorf("应返回最后一个错误")
	}
	// ins-1 associate 失败 -> 不应解绑 ins-1 的旧 SG
	for _, d := range fc.disassociated {
		if strings.HasPrefix(d, "ins-1|") {
			t.Errorf("associate 失败的实例不应解绑，实际解绑了 %s", d)
		}
	}
}

func TestRebindAllInstancesToSingleSG_DisassociateFails(t *testing.T) {
	// associate 成功但 disassociate 失败 -> lastErr 保留，成功计数仍 +1
	defer overrideDescribeSGFn(func(_ context.Context, ids []string) (map[string][]string, error) {
		return map[string][]string{"ins-1": {"sg-old-1"}}, nil
	})()
	fc := &fakeCVMClient{
		disassociateErrs: []error{errors.New("mock disassociate error")},
	}
	defer overrideNewCVMClientFn(func(context.Context) (cvmSgBinder, error) { return fc, nil })()

	err := rebindAllInstancesToSingleSG(context.Background(), []string{"ins-1"}, "sg-target")
	if err == nil {
		t.Errorf("disassociate 失败应返回 lastErr")
	}
	if len(fc.associated) != 1 {
		t.Errorf("associate 应成功 1 次，实际=%+v", fc.associated)
	}
}

// ==================== replaceVpcCidrPlaceholder 覆盖 ====================

func TestReplaceVpcCidrPlaceholder_EmptyCidr_NoOp(t *testing.T) {
	ruleSet := sgRuleSet{
		Categories: []sgRuleCategory{{
			RuleGroups: []sgRuleGroup{{
				Rules: []requiredSGRule{
					{CidrBlock: "{{VPC_CIDR}}"},
				},
			}},
		}},
	}
	replaceVpcCidrPlaceholder(&ruleSet, "")
	if ruleSet.Categories[0].RuleGroups[0].Rules[0].CidrBlock != "{{VPC_CIDR}}" {
		t.Error("空 cidr 时不应替换")
	}
}

func TestReplaceVpcCidrPlaceholder_WithCidr(t *testing.T) {
	ruleSet := sgRuleSet{
		Categories: []sgRuleCategory{{
			RuleGroups: []sgRuleGroup{{
				Rules: []requiredSGRule{
					{CidrBlock: "{{VPC_CIDR}}"},
					{CidrBlock: "0.0.0.0/0"}, // 非占位符，不变
				},
			}},
		}},
	}
	replaceVpcCidrPlaceholder(&ruleSet, "10.0.0.0/16")
	rules := ruleSet.Categories[0].RuleGroups[0].Rules
	if rules[0].CidrBlock != "10.0.0.0/16" {
		t.Errorf("应替换为 10.0.0.0/16，实际=%s", rules[0].CidrBlock)
	}
	if rules[1].CidrBlock != "0.0.0.0/0" {
		t.Errorf("非占位符不应变化，实际=%s", rules[1].CidrBlock)
	}
}

// ==================== clawproRequiredRuleSet 覆盖 ====================

func TestClawproRequiredRuleSet_FromEmbedded(t *testing.T) {
	orig := ClawproRequiredSGRulesJSON
	defer func() { ClawproRequiredSGRulesJSON = orig }()
	ClawproRequiredSGRulesJSON = []byte(`{"categories":[{"type":"builtin","label":"内置","rule_groups":[]}]}`)
	got := clawproRequiredRuleSet()
	if len(got.Categories) != 1 || got.Categories[0].Type != "builtin" {
		t.Errorf("应从嵌入 JSON 正确解析，实际=%+v", got)
	}
}

// ==================== HandleBindSecurityGroup 补充（placeholder）====================
// 下方同名测试已在文件前半部分定义，这里不重复。

// ============================================================================
// 以下为基于 mock sgVpcClient 的深度覆盖测试，用于覆盖 handler 的主成功路径
// 以及 VPC 调用失败的分支。通过注入 newVpcClientForSGFn，无需真实凭据即可驱动
// 整条代码路径运行，从而显著提升增量覆盖率。
// ============================================================================

// fakeSGVpcClient 聚合实现了 sgVpcClient 接口，用于 handler 级深度测试。
type fakeSGVpcClient struct {
	// DescribeSecurityGroups
	descSGResp *vpc.DescribeSecurityGroupsResponse
	descSGErr  error
	// CreateSecurityGroup
	createSGResp *vpc.CreateSecurityGroupResponse
	createSGErr  error
	// ModifySecurityGroupAttribute
	modifySGResp *vpc.ModifySecurityGroupAttributeResponse
	modifySGErr  error
	// DescribeSecurityGroupPolicies
	descPoliciesResp *vpc.DescribeSecurityGroupPoliciesResponse
	descPoliciesErr  error
	// CreateSecurityGroupPolicies（可多次调用，按序返回）
	createPoliciesErrs []error
	createPoliciesIdx  int
	createPoliciesReqs []*vpc.CreateSecurityGroupPoliciesRequest
	// ReplaceSecurityGroupPolicy
	replacePolicyResp *vpc.ReplaceSecurityGroupPolicyResponse
	replacePolicyErr  error
	// ModifySecurityGroupPolicies（整包替换，切换 base 时重写 shard 用）
	modifyPoliciesResp *vpc.ModifySecurityGroupPoliciesResponse
	modifyPoliciesErr  error
	// DeleteSecurityGroupPolicies
	deletePoliciesResp *vpc.DeleteSecurityGroupPoliciesResponse
	deletePoliciesErr  error
	// DescribeVpcs
	descVpcsResp *vpc.DescribeVpcsResponse
	descVpcsErr  error
}

func (f *fakeSGVpcClient) DescribeSecurityGroups(req *vpc.DescribeSecurityGroupsRequest) (*vpc.DescribeSecurityGroupsResponse, error) {
	if f.descSGErr != nil {
		return nil, f.descSGErr
	}
	if f.descSGResp != nil {
		return f.descSGResp, nil
	}
	return &vpc.DescribeSecurityGroupsResponse{Response: &vpc.DescribeSecurityGroupsResponseParams{}}, nil
}

func (f *fakeSGVpcClient) CreateSecurityGroup(req *vpc.CreateSecurityGroupRequest) (*vpc.CreateSecurityGroupResponse, error) {
	if f.createSGErr != nil {
		return nil, f.createSGErr
	}
	if f.createSGResp != nil {
		return f.createSGResp, nil
	}
	return &vpc.CreateSecurityGroupResponse{Response: &vpc.CreateSecurityGroupResponseParams{}}, nil
}

// DeleteSecurityGroup 默认返回空响应；sg-ruleset-projection 加入接口方法后补上。
func (f *fakeSGVpcClient) DeleteSecurityGroup(req *vpc.DeleteSecurityGroupRequest) (*vpc.DeleteSecurityGroupResponse, error) {
	return &vpc.DeleteSecurityGroupResponse{Response: &vpc.DeleteSecurityGroupResponseParams{}}, nil
}

func (f *fakeSGVpcClient) ModifySecurityGroupAttribute(req *vpc.ModifySecurityGroupAttributeRequest) (*vpc.ModifySecurityGroupAttributeResponse, error) {
	if f.modifySGErr != nil {
		return nil, f.modifySGErr
	}
	if f.modifySGResp != nil {
		return f.modifySGResp, nil
	}
	return &vpc.ModifySecurityGroupAttributeResponse{}, nil
}

func (f *fakeSGVpcClient) DescribeSecurityGroupPolicies(req *vpc.DescribeSecurityGroupPoliciesRequest) (*vpc.DescribeSecurityGroupPoliciesResponse, error) {
	if f.descPoliciesErr != nil {
		return nil, f.descPoliciesErr
	}
	if f.descPoliciesResp != nil {
		return f.descPoliciesResp, nil
	}
	return &vpc.DescribeSecurityGroupPoliciesResponse{
		Response: &vpc.DescribeSecurityGroupPoliciesResponseParams{
			SecurityGroupPolicySet: &vpc.SecurityGroupPolicySet{},
		},
	}, nil
}

func (f *fakeSGVpcClient) CreateSecurityGroupPolicies(req *vpc.CreateSecurityGroupPoliciesRequest) (*vpc.CreateSecurityGroupPoliciesResponse, error) {
	f.createPoliciesReqs = append(f.createPoliciesReqs, req)
	var err error
	if f.createPoliciesIdx < len(f.createPoliciesErrs) {
		err = f.createPoliciesErrs[f.createPoliciesIdx]
	}
	f.createPoliciesIdx++
	if err != nil {
		return nil, err
	}
	return &vpc.CreateSecurityGroupPoliciesResponse{}, nil
}

func (f *fakeSGVpcClient) ReplaceSecurityGroupPolicy(req *vpc.ReplaceSecurityGroupPolicyRequest) (*vpc.ReplaceSecurityGroupPolicyResponse, error) {
	if f.replacePolicyErr != nil {
		return nil, f.replacePolicyErr
	}
	if f.replacePolicyResp != nil {
		return f.replacePolicyResp, nil
	}
	return &vpc.ReplaceSecurityGroupPolicyResponse{}, nil
}

func (f *fakeSGVpcClient) ModifySecurityGroupPolicies(req *vpc.ModifySecurityGroupPoliciesRequest) (*vpc.ModifySecurityGroupPoliciesResponse, error) {
	if f.modifyPoliciesErr != nil {
		return nil, f.modifyPoliciesErr
	}
	if f.modifyPoliciesResp != nil {
		return f.modifyPoliciesResp, nil
	}
	return &vpc.ModifySecurityGroupPoliciesResponse{}, nil
}

func (f *fakeSGVpcClient) DeleteSecurityGroupPolicies(req *vpc.DeleteSecurityGroupPoliciesRequest) (*vpc.DeleteSecurityGroupPoliciesResponse, error) {
	if f.deletePoliciesErr != nil {
		return nil, f.deletePoliciesErr
	}
	if f.deletePoliciesResp != nil {
		return f.deletePoliciesResp, nil
	}
	return &vpc.DeleteSecurityGroupPoliciesResponse{}, nil
}

func (f *fakeSGVpcClient) DescribeVpcs(req *vpc.DescribeVpcsRequest) (*vpc.DescribeVpcsResponse, error) {
	if f.descVpcsErr != nil {
		return nil, f.descVpcsErr
	}
	if f.descVpcsResp != nil {
		return f.descVpcsResp, nil
	}
	return &vpc.DescribeVpcsResponse{Response: &vpc.DescribeVpcsResponseParams{}}, nil
}

func (f *fakeSGVpcClient) DescribeSecurityGroupAssociationStatistics(req *vpc.DescribeSecurityGroupAssociationStatisticsRequest) (*vpc.DescribeSecurityGroupAssociationStatisticsResponse, error) {
	return &vpc.DescribeSecurityGroupAssociationStatisticsResponse{}, nil
}

// withFakeSGVpcClient 替换 newVpcClientForSGFn 为返回给定 fake 客户端的工厂，
// 并返回 teardown 函数以恢复原工厂。
func withFakeSGVpcClient(fake sgVpcClient) func() {
	orig := newVpcClientForSGFn
	newVpcClientForSGFn = func(ctx context.Context) (sgVpcClient, error) { return fake, nil }
	return func() { newVpcClientForSGFn = orig }
}

// withFakeSGVpcClientError 让 newVpcClientForSGFn 返回指定错误（模拟客户端创建失败，
// 但不依赖真实凭据）。
func withFakeSGVpcClientError(err error) func() {
	orig := newVpcClientForSGFn
	newVpcClientForSGFn = func(ctx context.Context) (sgVpcClient, error) { return nil, err }
	return func() { newVpcClientForSGFn = orig }
}

// withNoRetrySleep 将 addMissingSGRulesRetryInterval 设为 0，使重试不 sleep。
func withNoRetrySleep() func() {
	orig := addMissingSGRulesRetryInterval
	addMissingSGRulesRetryInterval = 0
	return func() { addMissingSGRulesRetryInterval = orig }
}

// ---------- HandleGetSecurityGroup 成功 ----------

func TestHandleGetSecurityGroup_Success(t *testing.T) {
	initSGHandlerTestDB(t)
	setSiteConfigSG(t, "sg-abc")
	defer withAdminToken("test-admin-token")()

	fake := &fakeSGVpcClient{
		descSGResp: &vpc.DescribeSecurityGroupsResponse{
			Response: &vpc.DescribeSecurityGroupsResponseParams{
				SecurityGroupSet: []*vpc.SecurityGroup{
					{SecurityGroupId: common.StringPtr("sg-abc")},
				},
			},
		},
	}
	defer withFakeSGVpcClient(fake)()

	req := newAdminRequest(http.MethodGet, "/admin/config/security-group", "")
	w := httptest.NewRecorder()
	HandleGetSecurityGroup(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d，body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "sg-abc") {
		t.Errorf("响应应包含安全组 ID，实际=%s", w.Body.String())
	}
}

func TestHandleGetSecurityGroup_DescribeError(t *testing.T) {
	initSGHandlerTestDB(t)
	setSiteConfigSG(t, "sg-err")
	defer withAdminToken("test-admin-token")()

	fake := &fakeSGVpcClient{descSGErr: errors.New("mock describe error")}
	defer withFakeSGVpcClient(fake)()

	req := newAdminRequest(http.MethodGet, "/admin/config/security-group", "")
	w := httptest.NewRecorder()
	HandleGetSecurityGroup(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("期望 500，实际=%d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "查询安全组失败") {
		t.Errorf("body 应包含 '查询安全组失败'，实际=%s", w.Body.String())
	}
}

// ---------- HandleCreateSecurityGroup 主成功路径 ----------

func TestHandleCreateSecurityGroup_Success_NoQuickRules(t *testing.T) {
	initSGHandlerTestDB(t)
	defer withAdminToken("test-admin-token")()
	defer withNoRetrySleep()()
	// 禁止异步 goroutine 误跑到真实 listInstanceIds；返回空 -> 异步分支快速 return
	orig := describeInstancesSecurityGroupsFn
	describeInstancesSecurityGroupsFn = func(_ context.Context, ids []string) (map[string][]string, error) {
		return map[string][]string{}, nil
	}
	defer func() { describeInstancesSecurityGroupsFn = orig }()

	fake := &fakeSGVpcClient{
		createSGResp: &vpc.CreateSecurityGroupResponse{
			Response: &vpc.CreateSecurityGroupResponseParams{
				SecurityGroup: &vpc.SecurityGroup{SecurityGroupId: common.StringPtr("sg-new-1")},
			},
		},
	}
	defer withFakeSGVpcClient(fake)()

	body := `{"GroupName":"my-sg","GroupDescription":"desc"}`
	req := newAdminRequest(http.MethodPost, "/admin/config/security-group", body)
	w := httptest.NewRecorder()
	HandleCreateSecurityGroup(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "sg-new-1") {
		t.Errorf("响应应包含新安全组 ID，实际=%s", w.Body.String())
	}
}

func TestHandleCreateSecurityGroup_ReadBodyError(t *testing.T) {
	initSGHandlerTestDB(t)
	defer withAdminToken("test-admin-token")()

	// 构造一个 body read 失败的请求：Reader 返回 error
	req := httptest.NewRequest(http.MethodPost, "/admin/config/security-group", errReader{})
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	HandleCreateSecurityGroup(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// errReader 模拟 io.Reader 读取失败
type errReader struct{}

func (errReader) Read(p []byte) (int, error) { return 0, errors.New("mock read error") }

func TestHandleCreateSecurityGroup_CreateReturnError(t *testing.T) {
	initSGHandlerTestDB(t)
	defer withAdminToken("test-admin-token")()

	fake := &fakeSGVpcClient{createSGErr: errors.New("mock create sg error")}
	defer withFakeSGVpcClient(fake)()

	body := `{"GroupName":"x"}`
	req := newAdminRequest(http.MethodPost, "/admin/config/security-group", body)
	w := httptest.NewRecorder()
	HandleCreateSecurityGroup(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("期望 500，实际=%d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "创建安全组失败") {
		t.Errorf("body 应包含 '创建安全组失败'")
	}
}

func TestHandleCreateSecurityGroup_AbnormalResponse(t *testing.T) {
	initSGHandlerTestDB(t)
	defer withAdminToken("test-admin-token")()

	// Response 为 nil 时触发"返回数据异常"
	fake := &fakeSGVpcClient{
		createSGResp: &vpc.CreateSecurityGroupResponse{Response: nil},
	}
	defer withFakeSGVpcClient(fake)()

	body := `{"GroupName":"x"}`
	req := newAdminRequest(http.MethodPost, "/admin/config/security-group", body)
	w := httptest.NewRecorder()
	HandleCreateSecurityGroup(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("期望 500，实际=%d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "创建安全组返回数据异常") {
		t.Errorf("body 应包含 '创建安全组返回数据异常'")
	}
}

func TestHandleCreateSecurityGroup_WithQuickRules_BuiltinAndRestrictVPC(t *testing.T) {
	initSGHandlerTestDB(t)
	defer withAdminToken("test-admin-token")()
	defer withNoRetrySleep()()
	origDISG := describeInstancesSecurityGroupsFn
	describeInstancesSecurityGroupsFn = func(_ context.Context, ids []string) (map[string][]string, error) { return nil, nil }
	defer func() { describeInstancesSecurityGroupsFn = origDISG }()

	// 设置 VpcId 使 restrict_vpc_access 可以查询 CIDR
	model.DB(context.Background()).Model(&model.SiteConfig{}).Where("1=1").Updates(map[string]interface{}{"vpc_id": "vpc-1"})

	// 注入规则集
	origJSON := ClawproRequiredSGRulesJSON
	ClawproRequiredSGRulesJSON = []byte(`{
		"categories":[{"type":"builtin","label":"内置","rule_groups":[
			{"key":"allow_ssh","name":"SSH","rules":[
				{"direction":"ingress","protocol":"TCP","port":"22","cidr_block":"0.0.0.0/0","action":"ACCEPT","description":"ssh"}
			]},
			{"key":"allow_out","name":"OUT","rules":[
				{"direction":"egress","protocol":"ALL","port":"ALL","cidr_block":"0.0.0.0/0","action":"ACCEPT","description":"out"}
			]}
		]}]
	}`)
	defer func() { ClawproRequiredSGRulesJSON = origJSON }()

	fake := &fakeSGVpcClient{
		createSGResp: &vpc.CreateSecurityGroupResponse{
			Response: &vpc.CreateSecurityGroupResponseParams{
				SecurityGroup: &vpc.SecurityGroup{SecurityGroupId: common.StringPtr("sg-qr")},
			},
		},
		descVpcsResp: &vpc.DescribeVpcsResponse{
			Response: &vpc.DescribeVpcsResponseParams{
				VpcSet: []*vpc.Vpc{{CidrBlock: common.StringPtr("10.0.0.0/16")}},
			},
		},
		// Ingress + Egress 各一次 CreateSecurityGroupPolicies，都成功
		createPoliciesErrs: []error{nil, nil},
	}
	defer withFakeSGVpcClient(fake)()

	body := `{"GroupName":"x","quick_rules":["restrict_vpc_access","allow_ssh","allow_out","unknown_rule"]}`
	req := newAdminRequest(http.MethodPost, "/admin/config/security-group", body)
	w := httptest.NewRecorder()
	HandleCreateSecurityGroup(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	if len(fake.createPoliciesReqs) < 2 {
		t.Errorf("期望 ingress + egress 至少 2 次 CreateSecurityGroupPolicies 调用，实际=%d", len(fake.createPoliciesReqs))
	}
}

func TestHandleCreateSecurityGroup_QuickRules_CreatePoliciesRetrySuccess(t *testing.T) {
	initSGHandlerTestDB(t)
	defer withAdminToken("test-admin-token")()
	defer withNoRetrySleep()()
	origDISG := describeInstancesSecurityGroupsFn
	describeInstancesSecurityGroupsFn = func(_ context.Context, ids []string) (map[string][]string, error) { return nil, nil }
	defer func() { describeInstancesSecurityGroupsFn = origDISG }()

	origJSON := ClawproRequiredSGRulesJSON
	ClawproRequiredSGRulesJSON = []byte(`{
		"categories":[{"type":"builtin","label":"内置","rule_groups":[
			{"key":"allow_ssh","name":"SSH","rules":[
				{"direction":"ingress","protocol":"TCP","port":"22","cidr_block":"0.0.0.0/0","action":"ACCEPT","description":"ssh"}
			]}
		]}]
	}`)
	defer func() { ClawproRequiredSGRulesJSON = origJSON }()

	fake := &fakeSGVpcClient{
		createSGResp: &vpc.CreateSecurityGroupResponse{
			Response: &vpc.CreateSecurityGroupResponseParams{
				SecurityGroup: &vpc.SecurityGroup{SecurityGroupId: common.StringPtr("sg-r")},
			},
		},
		// 第 1 次失败，第 2 次成功
		createPoliciesErrs: []error{errors.New("first attempt fail"), nil},
	}
	defer withFakeSGVpcClient(fake)()

	body := `{"GroupName":"x","quick_rules":["allow_ssh"]}`
	req := newAdminRequest(http.MethodPost, "/admin/config/security-group", body)
	w := httptest.NewRecorder()
	HandleCreateSecurityGroup(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleCreateSecurityGroup_QuickRules_CreatePoliciesAllFail(t *testing.T) {
	initSGHandlerTestDB(t)
	defer withAdminToken("test-admin-token")()
	defer withNoRetrySleep()()
	origDISG := describeInstancesSecurityGroupsFn
	describeInstancesSecurityGroupsFn = func(_ context.Context, ids []string) (map[string][]string, error) { return nil, nil }
	defer func() { describeInstancesSecurityGroupsFn = origDISG }()

	origJSON := ClawproRequiredSGRulesJSON
	ClawproRequiredSGRulesJSON = []byte(`{
		"categories":[{"type":"builtin","label":"内置","rule_groups":[
			{"key":"allow_out","name":"OUT","rules":[
				{"direction":"egress","protocol":"ALL","port":"ALL","cidr_block":"0.0.0.0/0","action":"ACCEPT","description":"out"}
			]}
		]}]
	}`)
	defer func() { ClawproRequiredSGRulesJSON = origJSON }()

	fake := &fakeSGVpcClient{
		createSGResp: &vpc.CreateSecurityGroupResponse{
			Response: &vpc.CreateSecurityGroupResponseParams{
				SecurityGroup: &vpc.SecurityGroup{SecurityGroupId: common.StringPtr("sg-fail")},
			},
		},
		createPoliciesErrs: []error{
			errors.New("e1"), errors.New("e2"), errors.New("e3"),
			errors.New("e1"), errors.New("e2"), errors.New("e3"),
		},
	}
	defer withFakeSGVpcClient(fake)()

	body := `{"GroupName":"x","quick_rules":["allow_out"]}`
	req := newAdminRequest(http.MethodPost, "/admin/config/security-group", body)
	w := httptest.NewRecorder()
	HandleCreateSecurityGroup(w, req)

	// quickRuleErr 不阻断流程，仍返回 200（安全组已创建）
	if w.Code != http.StatusOK {
		t.Fatalf("quick rule 失败不阻断 handler，期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleCreateSecurityGroup_QuickRules_DescribeVpcsFails(t *testing.T) {
	initSGHandlerTestDB(t)
	defer withAdminToken("test-admin-token")()
	defer withNoRetrySleep()()
	origDISG := describeInstancesSecurityGroupsFn
	describeInstancesSecurityGroupsFn = func(_ context.Context, ids []string) (map[string][]string, error) { return nil, nil }
	defer func() { describeInstancesSecurityGroupsFn = origDISG }()
	model.DB(context.Background()).Model(&model.SiteConfig{}).Where("1=1").Update("vpc_id", "vpc-x")

	fake := &fakeSGVpcClient{
		createSGResp: &vpc.CreateSecurityGroupResponse{
			Response: &vpc.CreateSecurityGroupResponseParams{
				SecurityGroup: &vpc.SecurityGroup{SecurityGroupId: common.StringPtr("sg-dv")},
			},
		},
		descVpcsErr: errors.New("mock describe vpcs error"),
	}
	defer withFakeSGVpcClient(fake)()

	body := `{"GroupName":"x","quick_rules":["restrict_vpc_access"]}`
	req := newAdminRequest(http.MethodPost, "/admin/config/security-group", body)
	w := httptest.NewRecorder()
	HandleCreateSecurityGroup(w, req)

	// DescribeVpcs 失败只会 warn 跳过规则，handler 仍返回 200
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// ---------- HandleBindSecurityGroup 主成功路径 ----------

func TestHandleBindSecurityGroup_Success_NoAutoFix(t *testing.T) {
	initSGHandlerTestDB(t)
	defer withAdminToken("test-admin-token")()
	origDISG := describeInstancesSecurityGroupsFn
	describeInstancesSecurityGroupsFn = func(_ context.Context, ids []string) (map[string][]string, error) { return nil, nil }
	defer func() { describeInstancesSecurityGroupsFn = origDISG }()

	fake := &fakeSGVpcClient{
		descSGResp: &vpc.DescribeSecurityGroupsResponse{
			Response: &vpc.DescribeSecurityGroupsResponseParams{
				SecurityGroupSet: []*vpc.SecurityGroup{{SecurityGroupId: common.StringPtr("sg-target")}},
			},
		},
	}
	defer withFakeSGVpcClient(fake)()

	body := `{"security_group_id":"sg-target","auto_fix_rules":false}`
	req := newAdminRequest(http.MethodPost, "/admin/config/security-group/bind", body)
	w := httptest.NewRecorder()
	HandleBindSecurityGroup(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleBindSecurityGroup_DescribeFail(t *testing.T) {
	initSGHandlerTestDB(t)
	defer withAdminToken("test-admin-token")()

	fake := &fakeSGVpcClient{descSGErr: errors.New("mock describe sg error")}
	defer withFakeSGVpcClient(fake)()

	body := `{"security_group_id":"sg-xxx"}`
	req := newAdminRequest(http.MethodPost, "/admin/config/security-group/bind", body)
	w := httptest.NewRecorder()
	HandleBindSecurityGroup(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("期望 500，实际=%d", w.Code)
	}
}

func TestHandleBindSecurityGroup_SGNotExist(t *testing.T) {
	initSGHandlerTestDB(t)
	defer withAdminToken("test-admin-token")()

	fake := &fakeSGVpcClient{
		descSGResp: &vpc.DescribeSecurityGroupsResponse{
			Response: &vpc.DescribeSecurityGroupsResponseParams{SecurityGroupSet: nil},
		},
	}
	defer withFakeSGVpcClient(fake)()

	body := `{"security_group_id":"sg-not-exist"}`
	req := newAdminRequest(http.MethodPost, "/admin/config/security-group/bind", body)
	w := httptest.NewRecorder()
	HandleBindSecurityGroup(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "安全组不存在") {
		t.Errorf("body 应包含 '安全组不存在'")
	}
}

func TestHandleBindSecurityGroup_AutoFix_NoMissingRules(t *testing.T) {
	initSGHandlerTestDB(t)
	defer withAdminToken("test-admin-token")()
	origDISG := describeInstancesSecurityGroupsFn
	describeInstancesSecurityGroupsFn = func(_ context.Context, ids []string) (map[string][]string, error) { return nil, nil }
	defer func() { describeInstancesSecurityGroupsFn = origDISG }()

	// 空规则配置 -> 没有缺失规则
	origJSON := ClawproRequiredSGRulesJSON
	ClawproRequiredSGRulesJSON = []byte(`{"categories":[]}`)
	defer func() { ClawproRequiredSGRulesJSON = origJSON }()

	fake := &fakeSGVpcClient{
		descSGResp: &vpc.DescribeSecurityGroupsResponse{
			Response: &vpc.DescribeSecurityGroupsResponseParams{
				SecurityGroupSet: []*vpc.SecurityGroup{{SecurityGroupId: common.StringPtr("sg-target2")}},
			},
		},
		descPoliciesResp: &vpc.DescribeSecurityGroupPoliciesResponse{
			Response: &vpc.DescribeSecurityGroupPoliciesResponseParams{
				SecurityGroupPolicySet: &vpc.SecurityGroupPolicySet{},
			},
		},
	}
	defer withFakeSGVpcClient(fake)()

	body := `{"security_group_id":"sg-target2","auto_fix_rules":true}`
	req := newAdminRequest(http.MethodPost, "/admin/config/security-group/bind", body)
	w := httptest.NewRecorder()
	HandleBindSecurityGroup(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleBindSecurityGroup_AutoFix_WithMissingRulesAndFix(t *testing.T) {
	initSGHandlerTestDB(t)
	defer withAdminToken("test-admin-token")()
	defer withNoRetrySleep()()
	origDISG := describeInstancesSecurityGroupsFn
	describeInstancesSecurityGroupsFn = func(_ context.Context, ids []string) (map[string][]string, error) { return nil, nil }
	defer func() { describeInstancesSecurityGroupsFn = origDISG }()

	origJSON := ClawproRequiredSGRulesJSON
	ClawproRequiredSGRulesJSON = []byte(`{
		"categories":[{"type":"recommended","label":"推荐","rule_groups":[
			{"key":"allow_ssh","name":"SSH","rules":[
				{"direction":"ingress","protocol":"TCP","port":"22","cidr_block":"0.0.0.0/0","action":"ACCEPT","description":"ssh"}
			]}
		]}]
	}`)
	defer func() { ClawproRequiredSGRulesJSON = origJSON }()

	fake := &fakeSGVpcClient{
		descSGResp: &vpc.DescribeSecurityGroupsResponse{
			Response: &vpc.DescribeSecurityGroupsResponseParams{
				SecurityGroupSet: []*vpc.SecurityGroup{{SecurityGroupId: common.StringPtr("sg-target3")}},
			},
		},
		descPoliciesResp: &vpc.DescribeSecurityGroupPoliciesResponse{
			Response: &vpc.DescribeSecurityGroupPoliciesResponseParams{
				SecurityGroupPolicySet: &vpc.SecurityGroupPolicySet{},
			},
		},
	}
	defer withFakeSGVpcClient(fake)()

	body := `{"security_group_id":"sg-target3","auto_fix_rules":true}`
	req := newAdminRequest(http.MethodPost, "/admin/config/security-group/bind", body)
	w := httptest.NewRecorder()
	HandleBindSecurityGroup(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleBindSecurityGroup_AutoFix_CheckRulesError(t *testing.T) {
	initSGHandlerTestDB(t)
	defer withAdminToken("test-admin-token")()

	fake := &fakeSGVpcClient{
		descSGResp: &vpc.DescribeSecurityGroupsResponse{
			Response: &vpc.DescribeSecurityGroupsResponseParams{
				SecurityGroupSet: []*vpc.SecurityGroup{{SecurityGroupId: common.StringPtr("sg-4")}},
			},
		},
		descPoliciesErr: errors.New("mock describe policies error"),
	}
	defer withFakeSGVpcClient(fake)()

	body := `{"security_group_id":"sg-4","auto_fix_rules":true}`
	req := newAdminRequest(http.MethodPost, "/admin/config/security-group/bind", body)
	w := httptest.NewRecorder()
	HandleBindSecurityGroup(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("期望 500，实际=%d", w.Code)
	}
}

func TestHandleBindSecurityGroup_AutoFix_AddRulesError(t *testing.T) {
	initSGHandlerTestDB(t)
	defer withAdminToken("test-admin-token")()
	defer withNoRetrySleep()()

	origJSON := ClawproRequiredSGRulesJSON
	ClawproRequiredSGRulesJSON = []byte(`{
		"categories":[{"type":"recommended","label":"推荐","rule_groups":[
			{"key":"allow_ssh","name":"SSH","rules":[
				{"direction":"ingress","protocol":"TCP","port":"22","cidr_block":"0.0.0.0/0","action":"ACCEPT","description":"ssh"}
			]}
		]}]
	}`)
	defer func() { ClawproRequiredSGRulesJSON = origJSON }()

	fake := &fakeSGVpcClient{
		descSGResp: &vpc.DescribeSecurityGroupsResponse{
			Response: &vpc.DescribeSecurityGroupsResponseParams{
				SecurityGroupSet: []*vpc.SecurityGroup{{SecurityGroupId: common.StringPtr("sg-5")}},
			},
		},
		descPoliciesResp: &vpc.DescribeSecurityGroupPoliciesResponse{
			Response: &vpc.DescribeSecurityGroupPoliciesResponseParams{
				SecurityGroupPolicySet: &vpc.SecurityGroupPolicySet{},
			},
		},
		createPoliciesErrs: []error{errors.New("e1"), errors.New("e2"), errors.New("e3")},
	}
	defer withFakeSGVpcClient(fake)()

	body := `{"security_group_id":"sg-5","auto_fix_rules":true}`
	req := newAdminRequest(http.MethodPost, "/admin/config/security-group/bind", body)
	w := httptest.NewRecorder()
	HandleBindSecurityGroup(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("期望 500，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// ---------- HandleUpdateSecurityGroup ----------

func TestHandleUpdateSecurityGroup_Success(t *testing.T) {
	initSGHandlerTestDB(t)
	setSiteConfigSG(t, "sg-upd")
	defer withAdminToken("test-admin-token")()

	fake := &fakeSGVpcClient{}
	defer withFakeSGVpcClient(fake)()

	body := `{"GroupName":"new-name","GroupDescription":"new-desc"}`
	req := newAdminRequest(http.MethodPut, "/admin/config/security-group", body)
	w := httptest.NewRecorder()
	HandleUpdateSecurityGroup(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleUpdateSecurityGroup_InvalidJSON(t *testing.T) {
	initSGHandlerTestDB(t)
	setSiteConfigSG(t, "sg-upd2")
	defer withAdminToken("test-admin-token")()

	fake := &fakeSGVpcClient{}
	defer withFakeSGVpcClient(fake)()

	body := `not-a-json`
	req := newAdminRequest(http.MethodPut, "/admin/config/security-group", body)
	w := httptest.NewRecorder()
	HandleUpdateSecurityGroup(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleUpdateSecurityGroup_ModifyError(t *testing.T) {
	initSGHandlerTestDB(t)
	setSiteConfigSG(t, "sg-upd3")
	defer withAdminToken("test-admin-token")()

	fake := &fakeSGVpcClient{modifySGErr: errors.New("mock modify error")}
	defer withFakeSGVpcClient(fake)()

	body := `{"GroupName":"n"}`
	req := newAdminRequest(http.MethodPut, "/admin/config/security-group", body)
	w := httptest.NewRecorder()
	HandleUpdateSecurityGroup(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("期望 500，实际=%d", w.Code)
	}
}

func TestHandleUpdateSecurityGroup_ReadBodyError(t *testing.T) {
	initSGHandlerTestDB(t)
	setSiteConfigSG(t, "sg-upd4")
	defer withAdminToken("test-admin-token")()

	req := httptest.NewRequest(http.MethodPut, "/admin/config/security-group", errReader{})
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	HandleUpdateSecurityGroup(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d", w.Code)
	}
}

// ---------- HandleDescribeSecurityGroupPolicies ----------

func TestHandleDescribeSecurityGroupPolicies_Success(t *testing.T) {
	initSGHandlerTestDB(t)
	setSiteConfigSG(t, "sg-dp")
	defer withAdminToken("test-admin-token")()

	fake := &fakeSGVpcClient{}
	defer withFakeSGVpcClient(fake)()

	req := newAdminRequest(http.MethodGet, "/admin/config/security-group/policies", "")
	w := httptest.NewRecorder()
	HandleDescribeSecurityGroupPolicies(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}
}

func TestHandleDescribeSecurityGroupPolicies_EmptySG(t *testing.T) {
	initSGHandlerTestDB(t)
	defer withAdminToken("test-admin-token")()

	req := newAdminRequest(http.MethodGet, "/admin/config/security-group/policies", "")
	w := httptest.NewRecorder()
	HandleDescribeSecurityGroupPolicies(w, req)

	// 空 SG 时 handler 直接 return（不调用 VPC）-> 仍 200
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}
}

func TestHandleDescribeSecurityGroupPolicies_DescribeError(t *testing.T) {
	initSGHandlerTestDB(t)
	setSiteConfigSG(t, "sg-dpe")
	defer withAdminToken("test-admin-token")()

	fake := &fakeSGVpcClient{descPoliciesErr: errors.New("mock")}
	defer withFakeSGVpcClient(fake)()

	req := newAdminRequest(http.MethodGet, "/admin/config/security-group/policies", "")
	w := httptest.NewRecorder()
	HandleDescribeSecurityGroupPolicies(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("期望 500，实际=%d", w.Code)
	}
}

func TestHandleDescribeSecurityGroupPolicies_VpcClientFailure_Mock(t *testing.T) {
	initSGHandlerTestDB(t)
	setSiteConfigSG(t, "sg-dpf")
	defer withAdminToken("test-admin-token")()
	defer withFakeSGVpcClientError(errors.New("mock client"))()

	req := newAdminRequest(http.MethodGet, "/admin/config/security-group/policies", "")
	w := httptest.NewRecorder()
	HandleDescribeSecurityGroupPolicies(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("期望 500，实际=%d", w.Code)
	}
}

// ---------- HandleCreateSecurityGroupPolicies ----------

// ---------- HandleReplaceSecurityGroupPolicy ----------

// ---------- HandleDeleteSecurityGroupPolicies ----------

// ---------- HandleListCloudSecurityGroups ----------

func TestHandleListCloudSecurityGroups_Success_Default(t *testing.T) {
	initSGHandlerTestDB(t)
	defer withAdminToken("test-admin-token")()

	fake := &fakeSGVpcClient{
		descSGResp: &vpc.DescribeSecurityGroupsResponse{
			Response: &vpc.DescribeSecurityGroupsResponseParams{
				TotalCount: common.Uint64Ptr(2),
				SecurityGroupSet: []*vpc.SecurityGroup{
					{
						SecurityGroupId:   common.StringPtr("sg-1"),
						SecurityGroupName: common.StringPtr("n1"),
						SecurityGroupDesc: common.StringPtr("d1"),
						IsDefault:         common.BoolPtr(true),
					},
					{
						SecurityGroupId:   common.StringPtr("sg-2"),
						SecurityGroupName: common.StringPtr("n2"),
					},
				},
			},
		},
	}
	defer withFakeSGVpcClient(fake)()

	req := newAdminRequest(http.MethodGet, "/admin/config/security-groups", "")
	w := httptest.NewRecorder()
	HandleListCloudSecurityGroups(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		SecurityGroups []map[string]interface{} `json:"security_groups"`
		TotalCount     uint64                   `json:"total_count"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		// 最外层可能是 jsonOK 包装，尝试兼容
		var outer map[string]interface{}
		if err2 := json.Unmarshal(w.Body.Bytes(), &outer); err2 != nil {
			t.Fatalf("解析响应失败: %v / %v", err, err2)
		}
		if data, ok := outer["data"].(map[string]interface{}); ok {
			if sgs, ok := data["security_groups"].([]interface{}); ok {
				if len(sgs) != 2 {
					t.Errorf("期望返回 2 条，实际=%d", len(sgs))
				}
			}
		}
		return
	}
	if len(resp.SecurityGroups) != 2 {
		t.Errorf("期望返回 2 条，实际=%d", len(resp.SecurityGroups))
	}
}

func TestHandleListCloudSecurityGroups_WithParams(t *testing.T) {
	initSGHandlerTestDB(t)
	defer withAdminToken("test-admin-token")()

	fake := &fakeSGVpcClient{}
	defer withFakeSGVpcClient(fake)()

	req := newAdminRequest(http.MethodGet, "/admin/config/security-groups?offset=10&limit=5&keyword=foo&security_group_id=sg-a,sg-b", "")
	w := httptest.NewRecorder()
	HandleListCloudSecurityGroups(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}
}

func TestHandleListCloudSecurityGroups_Error(t *testing.T) {
	initSGHandlerTestDB(t)
	defer withAdminToken("test-admin-token")()

	fake := &fakeSGVpcClient{descSGErr: errors.New("mock")}
	defer withFakeSGVpcClient(fake)()

	req := newAdminRequest(http.MethodGet, "/admin/config/security-groups", "")
	w := httptest.NewRecorder()
	HandleListCloudSecurityGroups(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("期望 500，实际=%d", w.Code)
	}
}

func TestHandleListCloudSecurityGroups_EmptyResponse(t *testing.T) {
	initSGHandlerTestDB(t)
	defer withAdminToken("test-admin-token")()

	// Response 为 nil
	fake := &fakeSGVpcClient{
		descSGResp: &vpc.DescribeSecurityGroupsResponse{Response: nil},
	}
	defer withFakeSGVpcClient(fake)()

	req := newAdminRequest(http.MethodGet, "/admin/config/security-groups", "")
	w := httptest.NewRecorder()
	HandleListCloudSecurityGroups(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// ---------- HandleDescribeCloudSecurityGroupPolicies ----------

// ---------- HandleGetRequiredSGRules ----------

func TestHandleGetRequiredSGRules_DefaultBuiltin(t *testing.T) {
	initSGHandlerTestDB(t)
	defer withAdminToken("test-admin-token")()

	origJSON := ClawproRequiredSGRulesJSON
	ClawproRequiredSGRulesJSON = []byte(`{
		"categories":[
			{"type":"builtin","label":"内置","rule_groups":[
				{"key":"allow_ssh","name":"SSH","rules":[]}
			]},
			{"type":"recommended","label":"推荐","rule_groups":[
				{"key":"allow_http","name":"HTTP","rules":[]}
			]}
		]
	}`)
	defer func() { ClawproRequiredSGRulesJSON = origJSON }()

	req := newAdminRequest(http.MethodGet, "/admin/config/security-group/required-rules", "")
	w := httptest.NewRecorder()
	HandleGetRequiredSGRules(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	// 默认只返回 builtin
	if !strings.Contains(w.Body.String(), "builtin") {
		t.Errorf("body 应包含 builtin 分类")
	}
	if strings.Contains(w.Body.String(), "recommended") {
		t.Errorf("默认 type=builtin 时不应包含 recommended 分类：%s", w.Body.String())
	}
}

func TestHandleGetRequiredSGRules_All(t *testing.T) {
	initSGHandlerTestDB(t)
	defer withAdminToken("test-admin-token")()

	origJSON := ClawproRequiredSGRulesJSON
	ClawproRequiredSGRulesJSON = []byte(`{
		"categories":[
			{"type":"builtin","label":"内置","rule_groups":[]},
			{"type":"recommended","label":"推荐","rule_groups":[]}
		]
	}`)
	defer func() { ClawproRequiredSGRulesJSON = origJSON }()

	req := newAdminRequest(http.MethodGet, "/admin/config/security-group/required-rules?type=all", "")
	w := httptest.NewRecorder()
	HandleGetRequiredSGRules(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "builtin") || !strings.Contains(w.Body.String(), "recommended") {
		t.Errorf("type=all 应返回所有分类：%s", w.Body.String())
	}
}

func TestHandleGetRequiredSGRules_Recommended(t *testing.T) {
	initSGHandlerTestDB(t)
	defer withAdminToken("test-admin-token")()

	origJSON := ClawproRequiredSGRulesJSON
	ClawproRequiredSGRulesJSON = []byte(`{
		"categories":[
			{"type":"builtin","label":"内置","rule_groups":[]},
			{"type":"recommended","label":"推荐","rule_groups":[]}
		]
	}`)
	defer func() { ClawproRequiredSGRulesJSON = origJSON }()

	req := newAdminRequest(http.MethodGet, "/admin/config/security-group/required-rules?type=recommended", "")
	w := httptest.NewRecorder()
	HandleGetRequiredSGRules(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}
}

// ---------- HandleCheckSecurityGroupRules ----------

// ---------- resolveVpcCidr 成功路径（使用 sgVpcClient mock）----------

func TestResolveVpcCidr_SuccessWithMock(t *testing.T) {
	initSGHandlerTestDB(t)
	model.DB(context.Background()).Model(&model.SiteConfig{}).Where("1=1").Update("vpc_id", "vpc-ok")

	fake := &fakeSGVpcClient{
		descVpcsResp: &vpc.DescribeVpcsResponse{
			Response: &vpc.DescribeVpcsResponseParams{
				VpcSet: []*vpc.Vpc{{CidrBlock: common.StringPtr("10.20.0.0/16")}},
			},
		},
	}
	defer withFakeSGVpcClient(fake)()

	if got := resolveVpcCidr(context.Background()); got != "10.20.0.0/16" {
		t.Errorf("期望 10.20.0.0/16，实际=%s", got)
	}
}

func TestResolveVpcCidr_DescribeError(t *testing.T) {
	initSGHandlerTestDB(t)
	model.DB(context.Background()).Model(&model.SiteConfig{}).Where("1=1").Update("vpc_id", "vpc-err")

	fake := &fakeSGVpcClient{descVpcsErr: errors.New("mock")}
	defer withFakeSGVpcClient(fake)()

	if got := resolveVpcCidr(context.Background()); got != "" {
		t.Errorf("失败时应返回空字符串，实际=%s", got)
	}
}

func TestResolveVpcCidr_EmptyVpcSet(t *testing.T) {
	initSGHandlerTestDB(t)
	model.DB(context.Background()).Model(&model.SiteConfig{}).Where("1=1").Update("vpc_id", "vpc-empty")

	fake := &fakeSGVpcClient{
		descVpcsResp: &vpc.DescribeVpcsResponse{Response: &vpc.DescribeVpcsResponseParams{}},
	}
	defer withFakeSGVpcClient(fake)()

	if got := resolveVpcCidr(context.Background()); got != "" {
		t.Errorf("空 VpcSet 时应返回空字符串，实际=%s", got)
	}
}

// ---------- rebindAllInstancesToSingleSG 深度覆盖 ----------

func TestRebindAllInstancesToSingleSG_Empty(t *testing.T) {
	if err := rebindAllInstancesToSingleSG(context.Background(), nil, "sg-x"); err != nil {
		t.Errorf("空列表应直接返回 nil，实际=%v", err)
	}
}

func TestRebindAllInstancesToSingleSG_DescribeError(t *testing.T) {
	orig := describeInstancesSecurityGroupsFn
	describeInstancesSecurityGroupsFn = func(_ context.Context, ids []string) (map[string][]string, error) {
		return nil, errors.New("mock describe error")
	}
	defer func() { describeInstancesSecurityGroupsFn = orig }()

	err := rebindAllInstancesToSingleSG(context.Background(), []string{"i-1"}, "sg-x")
	if err == nil {
		t.Fatalf("期望返回错误")
	}
	if !strings.Contains(err.Error(), "查询实例安全组列表失败") {
		t.Errorf("错误应包含 '查询实例安全组列表失败'，实际=%v", err)
	}
}

func TestRebindAllInstancesToSingleSG_CVMClientError(t *testing.T) {
	origDISG := describeInstancesSecurityGroupsFn
	describeInstancesSecurityGroupsFn = func(_ context.Context, ids []string) (map[string][]string, error) {
		return map[string][]string{"i-1": {"sg-old"}}, nil
	}
	defer func() { describeInstancesSecurityGroupsFn = origDISG }()

	origCVM := newCVMClientFn
	newCVMClientFn = func(context.Context) (cvmSgBinder, error) { return nil, errors.New("mock cvm error") }
	defer func() { newCVMClientFn = origCVM }()

	err := rebindAllInstancesToSingleSG(context.Background(), []string{"i-1"}, "sg-x")
	if err == nil || !strings.Contains(err.Error(), "创建 CVM 客户端失败") {
		t.Errorf("期望 CVM 创建错误，实际=%v", err)
	}
}

func TestRebindAllInstancesToSingleSG_AlreadyBoundOnlyTarget(t *testing.T) {
	origDISG := describeInstancesSecurityGroupsFn
	describeInstancesSecurityGroupsFn = func(_ context.Context, ids []string) (map[string][]string, error) {
		return map[string][]string{"i-1": {"sg-target"}}, nil
	}
	defer func() { describeInstancesSecurityGroupsFn = origDISG }()

	origCVM := newCVMClientFn
	fakeCVM := &fakeCVMClient{}
	newCVMClientFn = func(context.Context) (cvmSgBinder, error) { return fakeCVM, nil }
	defer func() { newCVMClientFn = origCVM }()

	if err := rebindAllInstancesToSingleSG(context.Background(), []string{"i-1"}, "sg-target"); err != nil {
		t.Errorf("已只绑定目标 SG 时应 noop，实际 err=%v", err)
	}
	if len(fakeCVM.associated) != 0 || len(fakeCVM.disassociated) != 0 {
		t.Errorf("noop 场景不应触发任何 CVM 调用")
	}
}

func TestRebindAllInstancesToSingleSG_BindAndUnbind(t *testing.T) {
	origDISG := describeInstancesSecurityGroupsFn
	describeInstancesSecurityGroupsFn = func(_ context.Context, ids []string) (map[string][]string, error) {
		return map[string][]string{
			"i-a": {"sg-old1", "sg-old2"},   // 绑定新 + 解绑 2 个旧
			"i-b": {"sg-target", "sg-old3"}, // 已绑定目标，仅解绑 1 个
			"i-c": {"sg-target"},            // noop
		}, nil
	}
	defer func() { describeInstancesSecurityGroupsFn = origDISG }()

	origCVM := newCVMClientFn
	fakeCVM := &fakeCVMClient{}
	newCVMClientFn = func(context.Context) (cvmSgBinder, error) { return fakeCVM, nil }
	defer func() { newCVMClientFn = origCVM }()

	err := rebindAllInstancesToSingleSG(context.Background(), []string{"i-a", "i-b", "i-c"}, "sg-target")
	if err != nil {
		t.Errorf("正常流程应成功，实际 err=%v", err)
	}
	// i-a 触发 1 次 Associate；i-b/i-c 不触发（已绑定目标）
	if len(fakeCVM.associated) != 1 || fakeCVM.associated[0] != "i-a" {
		t.Errorf("Associate 调用不符合预期：%v", fakeCVM.associated)
	}
	// i-a 解绑 2 个旧；i-b 解绑 1 个旧；i-c 不解绑 -> 共 3 次
	if len(fakeCVM.disassociated) != 3 {
		t.Errorf("Disassociate 次数应为 3，实际=%d (%v)", len(fakeCVM.disassociated), fakeCVM.disassociated)
	}
}

func TestRebindAllInstancesToSingleSG_AssociateFail(t *testing.T) {
	origDISG := describeInstancesSecurityGroupsFn
	describeInstancesSecurityGroupsFn = func(_ context.Context, ids []string) (map[string][]string, error) {
		return map[string][]string{"i-1": {"sg-old"}}, nil
	}
	defer func() { describeInstancesSecurityGroupsFn = origDISG }()

	origCVM := newCVMClientFn
	fakeCVM := &fakeCVMClient{associateErrs: []error{errors.New("mock associate")}}
	newCVMClientFn = func(context.Context) (cvmSgBinder, error) { return fakeCVM, nil }
	defer func() { newCVMClientFn = origCVM }()

	err := rebindAllInstancesToSingleSG(context.Background(), []string{"i-1"}, "sg-target")
	if err == nil {
		t.Errorf("Associate 失败时应返回 lastErr")
	}
	// 绑定失败 -> 不应继续解绑
	if len(fakeCVM.disassociated) != 0 {
		t.Errorf("绑定失败后不应解绑旧 SG，实际=%v", fakeCVM.disassociated)
	}
}

func TestRebindAllInstancesToSingleSG_DisassociateFail(t *testing.T) {
	origDISG := describeInstancesSecurityGroupsFn
	describeInstancesSecurityGroupsFn = func(_ context.Context, ids []string) (map[string][]string, error) {
		return map[string][]string{"i-1": {"sg-old"}}, nil
	}
	defer func() { describeInstancesSecurityGroupsFn = origDISG }()

	origCVM := newCVMClientFn
	fakeCVM := &fakeCVMClient{disassociateErrs: []error{errors.New("mock dis")}}
	newCVMClientFn = func(context.Context) (cvmSgBinder, error) { return fakeCVM, nil }
	defer func() { newCVMClientFn = origCVM }()

	err := rebindAllInstancesToSingleSG(context.Background(), []string{"i-1"}, "sg-target")
	if err == nil || !strings.Contains(err.Error(), "mock dis") {
		t.Errorf("Disassociate 失败时 lastErr 应透传，实际=%v", err)
	}
}

// 避免 fmt 包未使用的错误（如果前面 block 里没用到 fmt 的话）
var _ = fmt.Sprintf

func TestSGPoliciesHandlers_MissingPolicies(t *testing.T) {
	tests := []struct {
		name    string
		handler func(w http.ResponseWriter, r *http.Request)
	}{
		{"CreatePolicies", HandleCreateSecurityGroupPolicies},
		{"ReplacePolicy", HandleReplaceSecurityGroupPolicy},
		{"DeletePolicies", HandleDeleteSecurityGroupPolicies},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initSGHandlerTestDB(t)
			setSiteConfigSG(t, "sg-test")
			cleanupToken := withAdminToken("test-admin-token")
			defer cleanupToken()
			cleanupVPC := withFakeSGVpcClient(&fakeSGVpcClient{})
			defer cleanupVPC()

			// 发送合法 JSON 但缺少 SecurityGroupPolicySet 字段，触发 nil 检查
			req := newAdminRequest(http.MethodPost, "/admin/sg/policies", `{}`)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			tt.handler(w, req)
			if w.Code != http.StatusBadRequest {
				t.Errorf("%s: expected 400, got %d body=%s", tt.name, w.Code, w.Body.String())
			}
			if !strings.Contains(w.Body.String(), "缺少必填参数 policies") {
				t.Errorf("%s: expected 缺少必填参数 policies, got body=%s", tt.name, w.Body.String())
			}
		})
	}
}
