package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"hatchery/model"
)

// 注：内部测试账号 UIN 白名单 internalAccountUins 与判定函数 IsInternalAccount /
// isInternalAccountUin 统一收敛在 internal_account.go 中，本文件只负责
// 办公网 IP CIDR 列表与基于该白名单的规则刷新逻辑。
var officeIngressCIDRs = []string{
	"219.133.41.27/32",
	"14.17.3.190/32",
	"14.17.22.0/25",
	"219.133.41.59/32",
	"14.17.3.159/32",
	"59.37.124.10/32",
	"113.108.77.0/25",
	"58.250.8.0/25",
	"14.17.22.128/25",
	"59.37.124.0/25",
	"183.60.2.26/32",
	"219.133.41.17/32",
	"119.147.10.128/25",
	"58.251.102.0/25",
	"14.17.22.140/32",
	"14.17.3.210/32",
	"117.88.121.64/27",
	"14.116.239.32/27",
	"14.22.11.160/27",
	"180.101.244.64/27",
	"43.128.123.32/27",
	"43.128.91.192/27",
	"43.132.141.0/27",
	"170.106.101.170/32",
	"119.147.35.47/32",
	"218.17.71.122/32",
	"119.147.35.115/32",
	"59.37.125.0/25",
	"119.147.35.183/32",
	"157.255.187.0/25",
	"59.36.126.128/25",
	"157.255.208.0/25",
	"59.36.126.0/25",
	"101.226.154.128/25",
	"101.226.154.253/32",
	"220.196.140.253/32",
	"61.151.228.144/29",
	"101.226.143.240/29",
	"180.153.219.124/32",
	"180.153.219.0/25",
	"101.226.154.254/32",
	"180.153.219.126/32",
	"220.196.140.128/25",
	"220.196.140.254/32",
	"111.206.145.100/32",
	"111.206.145.0/25",
	"61.135.194.253/32",
	"111.206.145.126/32",
	"61.135.194.128/25",
	"111.206.96.144/29",
	"61.135.194.254/32",
	"111.206.94.144/29",
	"113.142.183.254/32",
	"113.142.183.253/32",
	"113.142.183.128/26",
	"113.56.176.126/32",
	"113.56.176.0/25",
	"116.211.195.126/32",
	"116.211.195.0/25",
	"2402:4e00:d028::/45",
	"2402:4e00:d000::/45",
	"2402:4e00:d030::/45",
	"2402:4e00:d020::/45",
	"2402:4e00:d038::/45",
	"2402:4e00:d018::/45",
	"2402:4e00:d008::/45",
	"2402:4e00:d010::/45",
}

func shouldApplyOfficeIngressRules(ctx context.Context) bool {
	// 走统一的内部账号判定入口 IsInternalAccount：内部封装"ctx → CAM 接口"
	// 的 UIN 解析与白名单匹配，避免本文件再单独维护一份 UIN 解析逻辑。
	// 与 HandleCreateInstance 中 image_id 路径保持一致：CAM 异常时按"非内部"
	// 静默降级，宁可少加规则，也不要因外部依赖抖动让整条 ruleset 刷新链路失败。
	isInternal, err := IsInternalAccount(ctx)
	if err != nil {
		return false
	}
	return isInternal
}

func loadOfficeIngressRules(ctx context.Context) []Rule {
	if !shouldApplyOfficeIngressRules(ctx) {
		return nil
	}
	rules := make([]Rule, 0, len(officeIngressCIDRs)+2)
	for _, cidr := range officeIngressCIDRs {
		rules = append(rules, Rule{
			Direction:         "INGRESS",
			Protocol:          "ALL",
			Port:              "ALL",
			CidrBlock:         cidr,
			Action:            "ACCEPT",
			PolicyDescription: "办公网入站白名单",
			IsRequired:        true,
			Prepend:           true,
		})
	}
	rules = append(rules,
		Rule{
			Direction:         "INGRESS",
			Protocol:          "ALL",
			Port:              "ALL",
			CidrBlock:         "0.0.0.0/0",
			Action:            "DROP",
			PolicyDescription: "办公网入站兜底拒绝",
			IsRequired:        true,
			Prepend:           true,
		},
		Rule{
			Direction:         "INGRESS",
			Protocol:          "ALL",
			Port:              "ALL",
			CidrBlock:         "::/0",
			Action:            "DROP",
			PolicyDescription: "办公网入站兜底拒绝",
			IsRequired:        true,
			Prepend:           true,
		},
	)
	return rules
}

// RefreshOfficeIngressRulesForTenant refreshes RuleSet projections for internal-account tenants
// when the office ingress allowlist is missing from any RuleSet.
func RefreshOfficeIngressRulesForTenant(ctx context.Context) error {
	officeRules := loadOfficeIngressRules(ctx)
	if len(officeRules) == 0 {
		return nil
	}
	needsRefresh, err := officeIngressRulesNeedRefresh(ctx, officeRules)
	if err != nil {
		return err
	}
	if !needsRefresh {
		return nil
	}
	return RefreshAllRuleSetsForRequiredRules(ctx)
}

func officeIngressRulesNeedRefresh(ctx context.Context, officeRules []Rule) (bool, error) {
	var ruleSets []model.RuleSet
	if err := model.DB(ctx).Order("id ASC").Find(&ruleSets).Error; err != nil {
		return false, fmt.Errorf("list rule_sets: %w", err)
	}
	for _, rs := range ruleSets {
		var existing []Rule
		if strings.TrimSpace(rs.Rules) != "" {
			if err := json.Unmarshal([]byte(rs.Rules), &existing); err != nil {
				return false, fmt.Errorf("parse rule_set %d rules: %w", rs.ID, err)
			}
		}
		merged := MergeRequiredRules(existing, officeRules)
		oldJSON, err := json.Marshal(existing)
		if err != nil {
			return false, fmt.Errorf("marshal existing rules: %w", err)
		}
		newJSON, err := json.Marshal(merged)
		if err != nil {
			return false, fmt.Errorf("marshal merged rules: %w", err)
		}
		if string(oldJSON) != string(newJSON) {
			return true, nil
		}
	}
	return false, nil
}
