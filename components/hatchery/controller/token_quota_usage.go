package controller

import (
	"context"
	"time"

	"hatchery/controller/usergroup"
	"hatchery/model"
)

type tokenQuotaUsage struct {
	RuleIndex   int    `json:"rule_index"`
	Used        int64  `json:"used"`
	PeriodStart *int64 `json:"period_start"`
	PeriodEnd   *int64 `json:"period_end"`
	Active      bool   `json:"active"`
}

type globalTokenQuotaScope struct {
	Rules        []model.TokenQuotaRule
	UsageGroupID uint
}

func newTokenQuotaUsage(ruleIndex int, used int64, from time.Time, to *time.Time, active bool) tokenQuotaUsage {
	usage := tokenQuotaUsage{RuleIndex: ruleIndex, Used: used}
	zero := int64(0)
	usage.PeriodStart = &zero
	usage.PeriodEnd = &zero
	if active {
		usage.Active = true
		start := from.Unix()
		usage.PeriodStart = &start
		if to != nil {
			end := to.Unix()
			usage.PeriodEnd = &end
		} else {
			usage.PeriodEnd = nil
		}
	}
	return usage
}

// resolveEffectiveUserTokenQuotaRules 计算用户在指定分组下生效的 Token 配额规则。
//
// 语义（按 groupID 分支严格隔离，避免不同维度互相干扰）：
//
//   - groupID == 0（无分组场景）
//     1) 用户当前属于任意分组 → 使用 SiteConfig.DefaultTokenQuotaRules
//     2) 用户当前不属于任何分组 → 使用用户表自身配置
//     这样：用户被加入分组后，其无分组旧实例不再使用创建时烙印在用户表上的旧配额；
//     真正无分组用户仍保留用户表语义，TokenQuotaDay=-1 表示显式无限制。
//
//   - groupID > 0（有分组场景）
//     完全按"组（含祖先）策略 → SiteConfig 默认"的链路解析，**不读用户表字段**。
//     这是历史既定语义：分组用户的配额由分组管理员控制，不被用户表的旧字段
//     干扰，否则旧的 user.TokenQuotaDay 烙印会污染分组的 token_quota_day=-1
//     "显式无限制"等强语义。改动这条假设会破坏向后兼容的多个回归用例。
func resolveEffectiveUserTokenQuotaRules(ctx context.Context, user model.User, groupID uint) []model.TokenQuotaRule {
	siteConfig := model.GetSiteConfig(ctx)

	// 分支 A：无分组实例 —— 有分组用户走站点默认；真正无分组用户走用户自身配置
	if groupID == 0 {
		if userHasAnyGroup(ctx, user) {
			return siteConfig.ResolvedDefaultTokenQuotaRules()
		}
		return user.ResolvedTokenQuotaRules()
	}

	// 分支 B：有分组 —— 严格按组策略 → SiteConfig 默认解析，不读用户表字段
	fallbackRules := model.MarshalTokenQuotaRules(siteConfig.ResolvedDefaultTokenQuotaRules())
	rulesJSON, _ := usergroup.ResolveTokenQuotaRulesForGroup(ctx, groupID, fallbackRules, -1)
	if rules, ok := model.ParseTokenQuotaRules(rulesJSON); ok {
		return rules
	}
	return siteConfig.ResolvedDefaultTokenQuotaRules()
}

func userHasAnyGroup(ctx context.Context, user model.User) bool {
	var count int64
	if err := model.DB(ctx).Model(&model.UserGroupMember{}).Where("user_id = ?", user.ID).Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}

func resolveEffectiveGlobalTokenQuotaScope(ctx context.Context, siteConfig model.SiteConfig, groupID uint) globalTokenQuotaScope {
	period := siteConfig.NormalizedGlobalTokenQuotaPeriod()
	if rulesJSON, ok := usergroup.ResolveExplicitGlobalTokenQuotaRulesForGroup(ctx, groupID, period); ok {
		if rules, ok := model.ParseTokenQuotaRules(rulesJSON); ok {
			return globalTokenQuotaScope{Rules: rules, UsageGroupID: groupID}
		}
	}
	return globalTokenQuotaScope{Rules: siteConfig.ResolvedGlobalTokenQuotaRules(), UsageGroupID: groupID}
}

func userTokenQuotaUsages(ctx context.Context, userID uint, groupID uint, rules []model.TokenQuotaRule) []tokenQuotaUsage {
	return userTokenQuotaUsagesCompat(ctx, userID, groupID, rules, false)
}

// userTokenQuotaUsagesCompat 是 userTokenQuotaUsages 的兼容版本。
// 当 includeUngrouped 为 true 且 groupID > 0 时，统计中会把该用户名下 group_id=0
// 的"无分组创建的旧 agent"用量一并计入，用于解决用户被加入分组后旧 agent 用量
// 在分组视图下"消失"的展示问题。仅用户端 /quota/data 启用，管理端不启用。
func userTokenQuotaUsagesCompat(ctx context.Context, userID uint, groupID uint, rules []model.TokenQuotaRule, includeUngrouped bool) []tokenQuotaUsage {
	now := time.Now()
	usages := make([]tokenQuotaUsage, 0, len(rules))
	for i, rule := range rules {
		used := int64(0)
		from, to, active := rule.QuotaWindow(now)
		if active {
			used = model.UserTokenUsageInWindowCompat(ctx, userID, groupID, from, to, includeUngrouped)
		}
		usages = append(usages, newTokenQuotaUsage(i, used, from, to, active))
	}
	return usages
}

func globalTokenQuotaUsages(ctx context.Context, groupID uint, rules []model.TokenQuotaRule) []tokenQuotaUsage {
	now := time.Now()
	usages := make([]tokenQuotaUsage, 0, len(rules))
	for i, rule := range rules {
		used := int64(0)
		from, to, active := rule.QuotaWindow(now)
		if active {
			if groupID > 0 {
				used = model.GroupTokenUsageInWindow(ctx, groupID, from, to)
			} else {
				used = model.GlobalTokenUsageInWindow(ctx, from, to)
			}
		}
		usages = append(usages, newTokenQuotaUsage(i, used, from, to, active))
	}
	return usages
}
