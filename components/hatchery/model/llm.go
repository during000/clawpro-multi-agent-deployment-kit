package model

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	hcommon "hatchery/common"
	"hatchery/i18n"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// localDateToUTC converts a time to its local date (in the business timezone)
// represented as UTC midnight. This produces a pure date label: e.g. Beijing time
// 2026-04-03 14:00 +08:00 becomes 2026-04-03 00:00:00 UTC.
func localDateToUTC(t time.Time) time.Time {
	y, m, d := t.In(hcommon.BusinessLocation()).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// LocalToday returns today's date (in hcommon.BusinessLocation()) as UTC midnight.
func LocalToday() time.Time {
	return localDateToUTC(time.Now())
}

// LocalMonthRange returns the month containing t in hcommon.BusinessLocation() as [start, end),
// represented using the same UTC-midnight date labels as DailyUsageSummary.Date.
func LocalMonthRange(t time.Time) (start, end time.Time) {
	local := t.In(hcommon.BusinessLocation())
	y, m, _ := local.Date()
	monthStart := time.Date(y, m, 1, 0, 0, 0, 0, hcommon.BusinessLocation())
	nextMonthStart := monthStart.AddDate(0, 1, 0)
	return localDateToUTC(monthStart), localDateToUTC(nextMonthStart)
}

// LocalCurrentMonthRange returns the current local month as [start, end),
// represented using the same UTC-midnight date labels as DailyUsageSummary.Date.
func LocalCurrentMonthRange() (start, end time.Time) {
	return LocalMonthRange(time.Now())
}

// ParseLocalDate parses a date string (e.g. "2025-04-14") in the format
// "2006-01-02" and converts it to a UTC midnight time using the business timezone.
// This ensures frontend-provided dates are interpreted in the business timezone,
// consistent with LocalToday().
func ParseLocalDate(dateStr string) (time.Time, error) {
	t, err := time.ParseInLocation("2006-01-02", dateStr, hcommon.BusinessLocation())
	if err != nil {
		return time.Time{}, err
	}
	return localDateToUTC(t), nil
}

// ══════════════════════════════════════════════════════════════════════════════
// Token Quota Rules — 灵活配额规则引擎
// ══════════════════════════════════════════════════════════════════════════════

// Quota mode constants.
const (
	QuotaModeDay    = "day"    // 自然日 (00:00 ~ 次日 00:00)
	QuotaModeMonth  = "month"  // 自然月 (1号 ~ 下月1号)
	QuotaModeYear   = "year"   // 自然年 (1月1号 ~ 次年1月1号)
	QuotaModeCustom = "custom" // 自定义锚点时间窗口
)

// Quota refresh constants (for custom mode).
const (
	QuotaRefreshDaily   = "daily"
	QuotaRefreshMonthly = "monthly"
	QuotaRefreshYearly  = "yearly"
	QuotaRefreshNone    = "none"
)

// TokenQuotaRule 单条配额规则。
type TokenQuotaRule struct {
	Mode    string `json:"mode"`              // day / month / year / custom
	Limit   int    `json:"limit"`             // token 上限，-1=不限
	Start   *int64 `json:"start,omitempty"`   // Unix 秒，custom 模式的锚点起始时间
	End     *int64 `json:"end,omitempty"`     // Unix 秒，custom+none 的硬截止；nil/0=无截止
	Refresh string `json:"refresh,omitempty"` // daily / monthly / yearly / none（仅 custom 模式有效）
}

// ParseTokenQuotaRules 从 DB 存储的 JSON 字符串解析规则数组。
// 返回值 ok 区分"未设置"和"有值"：
//   - ""：未设置 → nil, false
//   - "[]" / "[{...}]"：有值 → rules, true
//   - 非法 JSON：静默容错 → nil, false
func ParseTokenQuotaRules(raw string) ([]TokenQuotaRule, bool) {
	if raw == "" {
		return nil, false
	}
	var rules []TokenQuotaRule
	if err := json.Unmarshal([]byte(raw), &rules); err != nil {
		return nil, false
	}
	return rules, true
}

// MarshalTokenQuotaRules 将规则数组序列化为 JSON 字符串。
// 序列化前自动 normalize：
//   - custom 模式的空 refresh 转为 "none"
//   - 非 custom 模式清除 start/end/refresh（无意义字段）
func MarshalTokenQuotaRules(rules []TokenQuotaRule) string {
	for i := range rules {
		if rules[i].Mode == QuotaModeCustom {
			if rules[i].Refresh == "" {
				rules[i].Refresh = QuotaRefreshNone
			}
		} else {
			// 非 custom 模式：start/end/refresh 无意义，清除
			rules[i].Start = nil
			rules[i].End = nil
			rules[i].Refresh = ""
		}
	}
	b, err := json.Marshal(rules)
	if err != nil {
		return ""
	}
	return string(b)
}

// QuotaWindow 计算该规则在给定时刻的有效时间窗口。
// 返回 (from, to, active)：active=false 表示规则当前不生效（未开始/已过期）。
// 返回的 from/to 是真正的 UTC 时间点（用于查询 llm_usage_logs.created_at）；to=nil 表示无终止。
func (rule TokenQuotaRule) QuotaWindow(now time.Time) (from time.Time, to *time.Time, active bool) {
	switch rule.Mode {
	case QuotaModeDay:
		local := now.In(hcommon.BusinessLocation())
		y, m, d := local.Date()
		dayStart := time.Date(y, m, d, 0, 0, 0, 0, hcommon.BusinessLocation())
		end := dayStart.Add(24 * time.Hour).UTC()
		return dayStart.UTC(), &end, true

	case QuotaModeMonth:
		local := now.In(hcommon.BusinessLocation())
		y, m, _ := local.Date()
		monthStart := time.Date(y, m, 1, 0, 0, 0, 0, hcommon.BusinessLocation())
		nextMonthStart := monthStart.AddDate(0, 1, 0)
		end := nextMonthStart.UTC()
		return monthStart.UTC(), &end, true

	case QuotaModeYear:
		local := now.In(hcommon.BusinessLocation())
		y, _, _ := local.Date()
		yearStart := time.Date(y, 1, 1, 0, 0, 0, 0, hcommon.BusinessLocation())
		nextYearStart := yearStart.AddDate(1, 0, 0)
		end := nextYearStart.UTC()
		return yearStart.UTC(), &end, true

	case QuotaModeCustom:
		return customQuotaWindow(rule, now)

	default:
		return time.Time{}, nil, false
	}
}

// customQuotaWindow 计算 custom 模式的时间窗口。
func customQuotaWindow(rule TokenQuotaRule, now time.Time) (from time.Time, to *time.Time, active bool) {
	if rule.Start == nil {
		return time.Time{}, nil, false
	}
	anchor := time.Unix(*rule.Start, 0).In(hcommon.BusinessLocation())
	if now.Before(anchor) {
		return time.Time{}, nil, false // 还没到开始时间
	}

	switch rule.Refresh {
	case QuotaRefreshNone, "":
		// 固定窗口：start → end（end 为空则无截止）
		if rule.End != nil && *rule.End > 0 {
			end := time.Unix(*rule.End, 0).In(hcommon.BusinessLocation())
			if now.After(end) || now.Equal(end) {
				return time.Time{}, nil, false // 已过期
			}
			windowEnd := end.UTC()
			return anchor.UTC(), &windowEnd, true
		}
		return anchor.UTC(), nil, true

	case QuotaRefreshDaily:
		// 按 anchor 的 HH:MM:SS 切割 24h 周期
		h, m, s := anchor.Clock()
		loc := hcommon.BusinessLocation()
		todayAnchor := time.Date(now.In(loc).Year(), now.In(loc).Month(), now.In(loc).Day(), h, m, s, 0, loc)
		if now.In(loc).Before(todayAnchor) {
			todayAnchor = todayAnchor.Add(-24 * time.Hour)
		}
		// 检查 end 截止
		var hardEnd *time.Time
		if rule.End != nil && *rule.End > 0 {
			end := time.Unix(*rule.End, 0).In(loc)
			if now.After(end) || now.Equal(end) {
				return time.Time{}, nil, false
			}
			utcEnd := end.UTC()
			hardEnd = &utcEnd
		}
		windowEnd := todayAnchor.Add(24 * time.Hour).UTC()
		if hardEnd != nil && hardEnd.Before(windowEnd) {
			windowEnd = *hardEnd
		}
		return todayAnchor.UTC(), &windowEnd, true

	case QuotaRefreshMonthly:
		// 固定 31 天周期，从 anchor 开始每 31 天为一个窗口
		const monthDuration = 31 * 24 * time.Hour
		elapsed := now.Sub(anchor)
		periods := int(elapsed / monthDuration)
		windowStart := anchor.Add(time.Duration(periods) * monthDuration)
		windowEnd := windowStart.Add(monthDuration)
		// 检查 end 截止
		var hardEnd *time.Time
		if rule.End != nil && *rule.End > 0 {
			end := time.Unix(*rule.End, 0).In(hcommon.BusinessLocation())
			if now.After(end) || now.Equal(end) {
				return time.Time{}, nil, false
			}
			utcEnd := end.UTC()
			hardEnd = &utcEnd
		}
		utcEnd := windowEnd.UTC()
		if hardEnd != nil && hardEnd.Before(utcEnd) {
			utcEnd = *hardEnd
		}
		return windowStart.UTC(), &utcEnd, true

	case QuotaRefreshYearly:
		// 固定 365 天周期，从 anchor 开始每 365 天为一个窗口
		const yearDuration = 365 * 24 * time.Hour
		elapsed := now.Sub(anchor)
		periods := int(elapsed / yearDuration)
		windowStart := anchor.Add(time.Duration(periods) * yearDuration)
		windowEnd := windowStart.Add(yearDuration)
		// 检查 end 截止
		var hardEnd *time.Time
		if rule.End != nil && *rule.End > 0 {
			end := time.Unix(*rule.End, 0).In(hcommon.BusinessLocation())
			if now.After(end) || now.Equal(end) {
				return time.Time{}, nil, false
			}
			utcEnd := end.UTC()
			hardEnd = &utcEnd
		}
		utcEnd := windowEnd.UTC()
		if hardEnd != nil && hardEnd.Before(utcEnd) {
			utcEnd = *hardEnd
		}
		return windowStart.UTC(), &utcEnd, true

	default:
		return time.Time{}, nil, false
	}
}

// CheckUserTokenQuota 检查用户是否超出任一配额规则。
// 返回 (exceeded, currentUsed, effectiveLimit)；rules 为空时视为不限制。
// groupID > 0 时只统计该用户在指定分组下的用量；groupID == 0 统计全部用量。
func CheckUserTokenQuota(ctx context.Context, userID uint, groupID uint, rules []TokenQuotaRule) (exceeded bool, currentUsed int64, effectiveLimit int) {
	if len(rules) == 0 {
		return false, 0, -1
	}
	now := time.Now()
	for _, rule := range rules {
		if rule.Limit < 0 {
			continue // -1 = unlimited for this rule
		}
		from, to, active := rule.QuotaWindow(now)
		if !active {
			// 规则不生效（未开始/已过期）→ 视为超限（阻止访问）
			return true, 0, rule.Limit
		}
		used := UserTokenUsageInWindow(ctx, userID, groupID, from, to)
		if used >= int64(rule.Limit) {
			return true, used, rule.Limit
		}
	}
	return false, 0, -1
}

// UserTokenUsageInWindow 查询 llm_usage_logs 表中指定用户在 [from, to) 时间窗口内的 token 用量。
// groupID > 0 时只统计该分组下的用量；groupID == 0 统计全部用量。
func UserTokenUsageInWindow(ctx context.Context, userID uint, groupID uint, from time.Time, to *time.Time) int64 {
	return UserTokenUsageInWindowCompat(ctx, userID, groupID, from, to, false)
}

// UserTokenUsageInWindowCompat 查询用户在 [from, to) 时间窗口内的 token 用量，
// 在 groupID > 0 时支持 includeUngrouped 兼容分支：把该用户名下 group_id=0
// 的"无分组创建的旧 agent"产生的用量一并计入。
//
// 兼容分支用于解决"用户最初无分组创建 agent，之后被加入分组 X 后，
// X 视图看不到这些旧 agent 用量"的展示问题（不修改历史数据）。
// 仅供用户端 /quota/data 入口使用，管理端按分组维度统计请勿启用，
// 以免污染分组聚合口径。
func UserTokenUsageInWindowCompat(ctx context.Context, userID uint, groupID uint, from time.Time, to *time.Time, includeUngrouped bool) int64 {
	var total int64
	q := DB(ctx).Model(&LLMUsageLog{}).
		Where("user_id = ? AND created_at >= ?", userID, from)
	if to != nil {
		q = q.Where("created_at < ?", *to)
	}
	if groupID > 0 {
		if includeUngrouped {
			q = q.Where("group_id IN ?", []uint{groupID, 0})
		} else {
			q = q.Where("group_id = ?", groupID)
		}
	}
	q.Select("COALESCE(SUM(total_tokens), 0)").Scan(&total)
	return total
}

// GlobalTokenUsageInWindow 查询 llm_usage_logs 中全站在 [from, to) 时间窗口内的 token 用量。
func GlobalTokenUsageInWindow(ctx context.Context, from time.Time, to *time.Time) int64 {
	var total int64
	q := DB(ctx).Model(&LLMUsageLog{}).Where("created_at >= ?", from)
	if to != nil {
		q = q.Where("created_at < ?", *to)
	}
	q.Select("COALESCE(SUM(total_tokens), 0)").Scan(&total)
	return total
}

// GroupTokenUsageInWindow 查询 llm_usage_logs 中指定分组在 [from, to) 时间窗口内的 token 用量。
func GroupTokenUsageInWindow(ctx context.Context, groupID uint, from time.Time, to *time.Time) int64 {
	var total int64
	q := DB(ctx).Model(&LLMUsageLog{}).
		Where("group_id = ? AND created_at >= ?", groupID, from)
	if to != nil {
		q = q.Where("created_at < ?", *to)
	}
	q.Select("COALESCE(SUM(total_tokens), 0)").Scan(&total)
	return total
}

// CheckGlobalTokenQuota 检查站点或分组维度是否超出任一全局配额规则。
// groupID == 0 时统计全站；groupID > 0 时统计该分组。
func CheckGlobalTokenQuota(ctx context.Context, groupID uint, rules []TokenQuotaRule) (exceeded bool, currentUsed int64, effectiveLimit int) {
	if len(rules) == 0 {
		return false, 0, -1
	}
	now := time.Now()
	for _, rule := range rules {
		if rule.Limit < 0 {
			continue
		}
		from, to, active := rule.QuotaWindow(now)
		if !active {
			return true, 0, rule.Limit
		}
		var used int64
		if groupID > 0 {
			used = GroupTokenUsageInWindow(ctx, groupID, from, to)
		} else {
			used = GlobalTokenUsageInWindow(ctx, from, to)
		}
		if used >= int64(rule.Limit) {
			return true, used, rule.Limit
		}
	}
	return false, 0, -1
}

// ValidateTokenQuotaRules 校验配额规则数组的合法性。
// 空数组合法（表示显式无限制）。
// start 允许为空（会在后续处理中自动填充当前时间）。
func ValidateTokenQuotaRules(rules []TokenQuotaRule) error {
	if len(rules) == 0 {
		return nil // 空数组 = 显式无限制
	}
	seenModes := make(map[string]bool)
	for _, rule := range rules {
		// 禁止重复 mode
		if seenModes[rule.Mode] {
			return hcommon.I18nError(i18n.MsgQuotaModeDuplicate)
		}
		seenModes[rule.Mode] = true

		// limit=-1 表示保留该规则窗口但不限制；其他负数非法。
		if rule.Limit < -1 {
			return hcommon.I18nError(i18n.MsgTokenQuotaMustBeValid)
		}

		switch rule.Mode {
		case QuotaModeDay, QuotaModeMonth, QuotaModeYear:
			// 只需要 limit
		case QuotaModeCustom:
			// start 允许为空（自动填充当前时间）
			// refresh 合法值为 daily/monthly/yearly/none，空默认 none
			if rule.Refresh != "" && rule.Refresh != QuotaRefreshDaily && rule.Refresh != QuotaRefreshMonthly && rule.Refresh != QuotaRefreshYearly && rule.Refresh != QuotaRefreshNone {
				return hcommon.I18nError(i18n.MsgQuotaRefreshModeInvalid)
			}
			// end 如果有值，必须晚于 start
			if rule.End != nil && *rule.End > 0 && rule.Start != nil && *rule.End <= *rule.Start {
				return hcommon.I18nError(i18n.MsgQuotaEndBeforeStart)
			}
		default:
			return hcommon.I18nError(i18n.MsgQuotaModeInvalid)
		}
	}
	return nil
}

// NormalizeTokenQuotaRules 统一处理 rules 输入：解析 → 校验 → 自动填 start → normalize → 序列化。
// 用于所有写入场景（设置用户配额、站点默认值、分组策略）。
// custom 模式的 start 为空时自动填充为当前时间。
func NormalizeTokenQuotaRules(raw string) (string, error) {
	if raw == "" {
		return "", hcommon.I18nError(i18n.MsgQuotaRulesCannotBeEmpty)
	}
	var rules []TokenQuotaRule
	if err := json.Unmarshal([]byte(raw), &rules); err != nil {
		return "", hcommon.I18nError(i18n.MsgInvalidQuotaRulesJSON, err)
	}
	if err := ValidateTokenQuotaRules(rules); err != nil {
		return "", err
	}
	FillCustomStartIfEmpty(rules)
	return MarshalTokenQuotaRules(rules), nil
}

// StampTokenQuotaRulesStart 用户创建时烙印：强制覆盖 custom 规则的 start 为当前时间。
// 每个用户独立计时，start 不继承策略/模板的值。
func StampTokenQuotaRulesStart(raw string) string {
	rules, ok := ParseTokenQuotaRules(raw)
	if !ok {
		return raw
	}
	StampCustomStart(rules)
	return MarshalTokenQuotaRules(rules)
}

// TokenQuotaDayFromRules 从 rules 中提取 day 模式的 limit 值（用于同步旧字段）。
// 如果没有 day 规则，返回 -1。
func TokenQuotaDayFromRules(rules []TokenQuotaRule) int {
	for _, r := range rules {
		if r.Mode == QuotaModeDay {
			return r.Limit
		}
	}
	return -1
}

// EffectiveTokenQuotaDay 返回对外展示的有效 day 配额值。
// 当 day==-1 且 rulesJSON 中包含 day 规则时，从 rules 反推该值；否则原样返回 day。
// 用于 API 响应中兼容旧客户端读取 token_quota_day 字段。
func EffectiveTokenQuotaDay(day int, rulesJSON string) int {
	if day == -1 {
		if rules, ok := ParseTokenQuotaRules(rulesJSON); ok {
			if d := TokenQuotaDayFromRules(rules); d >= 0 {
				return d
			}
		}
	}
	return day
}

// GlobalRulesFromLegacyQuota 将旧全局配额字段转换为规则数组。
func GlobalRulesFromLegacyQuota(limit int, period string) []TokenQuotaRule {
	if limit < 0 {
		return []TokenQuotaRule{}
	}
	mode := QuotaModeDay
	if NormalizeGlobalTokenQuotaPeriod(period) == GlobalTokenQuotaPeriodMonth {
		mode = QuotaModeMonth
	}
	return []TokenQuotaRule{{Mode: mode, Limit: limit}}
}

// TokenQuotaLimitFromRules 按 mode 从 rules 中提取 limit；没有对应规则返回 -1。
func TokenQuotaLimitFromRules(rules []TokenQuotaRule, mode string) int {
	for _, r := range rules {
		if r.Mode == mode {
			return r.Limit
		}
	}
	return -1
}

// EffectiveGlobalTokenQuotaLegacyFields 返回旧 API 字段 global_token_quota_day/global_token_quota_period 的兼容展示值。
// rules 有值时，rules 是权威配置，旧字段只从第一条 day/month 兼容规则反推；
// rules 为空或非法时，才 fallback 到旧 day + period 字段。
func EffectiveGlobalTokenQuotaLegacyFields(day int, period string, rulesJSON string) (int, string) {
	rules, ok := ParseTokenQuotaRules(rulesJSON)
	if !ok {
		return day, NormalizeGlobalTokenQuotaPeriod(period)
	}
	for _, rule := range rules {
		switch rule.Mode {
		case QuotaModeDay:
			return rule.Limit, GlobalTokenQuotaPeriodDay
		case QuotaModeMonth:
			return rule.Limit, GlobalTokenQuotaPeriodMonth
		}
	}
	return -1, GlobalTokenQuotaPeriodDay
}

// UpsertGlobalPeriodRule 在现有 rules 中插入或更新全局配额当前 period 对应的规则。
// limit=-1 表示删除该 period 对应规则；如果删除后无规则，返回 "[]"。
func UpsertGlobalPeriodRule(existingRulesJSON string, period string, limit int) string {
	mode := QuotaModeDay
	if NormalizeGlobalTokenQuotaPeriod(period) == GlobalTokenQuotaPeriodMonth {
		mode = QuotaModeMonth
	}
	rules, _ := ParseTokenQuotaRules(existingRulesJSON)
	if limit < 0 {
		filtered := rules[:0]
		for _, r := range rules {
			if r.Mode != mode {
				filtered = append(filtered, r)
			}
		}
		if len(filtered) == 0 {
			return "[]"
		}
		return MarshalTokenQuotaRules(filtered)
	}
	found := false
	for i := range rules {
		if rules[i].Mode == mode {
			rules[i].Limit = limit
			found = true
			break
		}
	}
	if !found {
		rules = append(rules, TokenQuotaRule{Mode: mode, Limit: limit})
	}
	return MarshalTokenQuotaRules(rules)
}

// UpsertDayRule 在现有 rules 中插入或更新 day 规则，保留其他规则不变。
// 用于旧 API `token_quota_day` 写入时：只影响 day 规则，不覆盖 custom/month 等。
// dayLimit=-1 表示"无限制"，此时删除已有的 day 规则。
func UpsertDayRule(existingRulesJSON string, dayLimit int) string {
	rules, _ := ParseTokenQuotaRules(existingRulesJSON)
	if dayLimit < 0 {
		// 删除 day 规则
		filtered := rules[:0]
		for _, r := range rules {
			if r.Mode != QuotaModeDay {
				filtered = append(filtered, r)
			}
		}
		if len(filtered) == 0 {
			return "[]"
		}
		return MarshalTokenQuotaRules(filtered)
	}
	found := false
	for i := range rules {
		if rules[i].Mode == QuotaModeDay {
			rules[i].Limit = dayLimit
			found = true
			break
		}
	}
	if !found {
		rules = append(rules, TokenQuotaRule{Mode: QuotaModeDay, Limit: dayLimit})
	}
	return MarshalTokenQuotaRules(rules)
}

// FillCustomStartIfEmpty 为 rules 中 start 为空的 custom 规则自动填充当前时间。
// 用于管理员设置策略/站点默认值时：custom 规则的计时起点默认为配置时间。
// 返回是否有修改。
func FillCustomStartIfEmpty(rules []TokenQuotaRule) bool {
	modified := false
	now := time.Now().Unix()
	for i := range rules {
		if rules[i].Mode == QuotaModeCustom && rules[i].Start == nil {
			rules[i].Start = &now
			modified = true
		}
	}
	return modified
}

// StampCustomStart 为 rules 中所有 custom 规则强制设置 start 为当前时间。
// 用于用户创建烙印时：每个用户的计时起点是各自的创建时间，而非策略配置时间。
func StampCustomStart(rules []TokenQuotaRule) {
	now := time.Now().Unix()
	for i := range rules {
		if rules[i].Mode == QuotaModeCustom {
			rules[i].Start = &now
		}
	}
}

// LLMUsageLog is an append-only request log for LLM proxy usage.
// Kept for audit purposes; statistics use DailyUsageSummary instead.
type LLMUsageLog struct {
	ID                     uint   `gorm:"primaryKey"`
	Identifier             string `gorm:"index;default:''"` // 多租户标识，MySQL 模式下自动填充和过滤
	InstanceID             uint   `gorm:"index;not null;default:0"`
	UserID                 uint   `gorm:"index;not null;default:0"`
	GroupID                uint   `gorm:"not null;default:0;index"` // agent 绑定的分组 ID
	AIModelID              uint
	Model                  string
	Provider               string
	PromptTokens           int
	CompletionTokens       int
	TotalTokens            int
	PromptCacheReadTokens  int `gorm:"not null;default:0"`
	PromptCacheWriteTokens int `gorm:"not null;default:0"`
	StatusCode             int
	Latency                int
	CreatedAt              time.Time `gorm:"index;not null"`
}

// DailyUsageSummary aggregates token usage per (date, user, instance, model).
// Updated via UPSERT on every proxy request instead of scanning LLMUsageLog.
type DailyUsageSummary struct {
	ID                     uint      `gorm:"primaryKey"`
	Identifier             string    `gorm:"uniqueIndex:idx_daily_summary_identifier;default:''"`                             // 多租户标识，加入复合唯一索引
	Date                   time.Time `gorm:"uniqueIndex:idx_daily_summary_identifier;not null;default:'1970-01-01 00:00:00'"` // date label, truncated to day
	UserID                 uint      `gorm:"uniqueIndex:idx_daily_summary_identifier;not null;default:0"`
	InstanceID             uint      `gorm:"uniqueIndex:idx_daily_summary_identifier;not null;default:0"`
	AIModelID              uint      `gorm:"uniqueIndex:idx_daily_summary_identifier;not null;default:0"`
	GroupID                uint      `gorm:"not null;default:0;index"` // agent 绑定的分组 ID，用于按组统计
	PromptTokens           int64     `gorm:"not null;default:0"`
	CompletionTokens       int64     `gorm:"not null;default:0"`
	TotalTokens            int64     `gorm:"not null;default:0"`
	PromptCacheReadTokens  int64     `gorm:"not null;default:0"`
	PromptCacheWriteTokens int64     `gorm:"not null;default:0"`
	RequestCount           int64     `gorm:"not null;default:0"`
}

// GenerateProxyToken generates a cryptographically random proxy token with "sk-" prefix.
func GenerateProxyToken() (string, error) {
	b := make([]byte, 24) // 24 bytes = 48 hex chars
	if _, err := rand.Read(b); err != nil {
		return "", hcommon.I18nRichError(err, i18n.MsgAPITokenGenerateFailed)
	}
	return "sk-" + hex.EncodeToString(b), nil
}

// UpsertDailyUsage increments (or inserts) the daily usage summary for the given
// (user, instance, model) combination. Called from the proxy after each request.
// Uses GORM clause.OnConflict for cross-database (SQLite + MySQL) compatibility.
func UpsertDailyUsage(ctx context.Context, userID, instanceID, aiModelID uint, promptTokens, completionTokens, totalTokens int, groupID uint, promptCacheReadTokens, promptCacheWriteTokens int) {
	if totalTokens <= 0 {
		return
	}
	today := LocalToday()

	DB(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "identifier"}, {Name: "date"}, {Name: "user_id"}, {Name: "instance_id"}, {Name: "ai_model_id"},
		},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"prompt_tokens":             gorm.Expr("prompt_tokens + ?", promptTokens),
			"completion_tokens":         gorm.Expr("completion_tokens + ?", completionTokens),
			"total_tokens":              gorm.Expr("total_tokens + ?", totalTokens),
			"prompt_cache_read_tokens":  gorm.Expr("prompt_cache_read_tokens + ?", promptCacheReadTokens),
			"prompt_cache_write_tokens": gorm.Expr("prompt_cache_write_tokens + ?", promptCacheWriteTokens),
			"request_count":             gorm.Expr("request_count + 1"),
		}),
	}).Create(&DailyUsageSummary{
		Date:                   today,
		UserID:                 userID,
		InstanceID:             instanceID,
		AIModelID:              aiModelID,
		GroupID:                groupID,
		PromptTokens:           int64(promptTokens),
		CompletionTokens:       int64(completionTokens),
		TotalTokens:            int64(totalTokens),
		PromptCacheReadTokens:  int64(promptCacheReadTokens),
		PromptCacheWriteTokens: int64(promptCacheWriteTokens),
		RequestCount:           1,
	})
}

// --- Query helpers using DailyUsageSummary ---

// GroupDailyTokenUsage returns today's total tokens for a specific group across all users and models.
func GroupDailyTokenUsage(ctx context.Context, groupID uint) int64 {
	var total int64
	today := LocalToday()
	DB(ctx).Model(&DailyUsageSummary{}).
		Where("group_id = ? AND date = ?", groupID, today).
		Select("COALESCE(SUM(total_tokens), 0)").
		Scan(&total)
	return total
}

func groupMonthlyTokenUsage(ctx context.Context, groupID uint) int64 {
	var total int64
	start, end := LocalCurrentMonthRange()
	DB(ctx).Model(&DailyUsageSummary{}).
		Where("group_id = ? AND date >= ? AND date < ?", groupID, start, end).
		Select("COALESCE(SUM(total_tokens), 0)").
		Scan(&total)
	return total
}

// GroupTokenUsageByPeriod returns token usage for a group using the configured global quota period.
func GroupTokenUsageByPeriod(ctx context.Context, groupID uint, period string) int64 {
	if NormalizeGlobalTokenQuotaPeriod(period) == GlobalTokenQuotaPeriodMonth {
		return groupMonthlyTokenUsage(ctx, groupID)
	}
	return GroupDailyTokenUsage(ctx, groupID)
}

// UserGroupDailyTokenUsage 返回指定用户在指定分组下今日所有 agent 的 Token 用量总和。
func UserGroupDailyTokenUsage(ctx context.Context, userID uint, groupID uint) int64 {
	var total int64
	today := LocalToday()
	DB(ctx).Model(&DailyUsageSummary{}).
		Where("user_id = ? AND group_id = ? AND date = ?", userID, groupID, today).
		Select("COALESCE(SUM(total_tokens), 0)").
		Scan(&total)
	return total
}

// ModelDailyTokenUsage returns today's total tokens across all instances for the given model.
func ModelDailyTokenUsage(ctx context.Context, aiModelID uint) int64 {
	var total int64
	today := LocalToday()
	DB(ctx).Model(&DailyUsageSummary{}).
		Where("ai_model_id = ? AND date = ?", aiModelID, today).
		Select("COALESCE(SUM(total_tokens), 0)").
		Scan(&total)
	return total
}

// UsageRow is a single row returned by a grouped usage query.
type UsageRow struct {
	InstanceID             uint
	PromptTokens           int64
	CompletionTokens       int64
	TotalTokens            int64
	PromptCacheReadTokens  int64
	PromptCacheWriteTokens int64
}

// UserDailyUsageByInstance returns today's usage grouped by instance_id for the given user.
func UserDailyUsageByInstance(ctx context.Context, userID uint) []UsageRow {
	today := LocalToday()
	var rows []UsageRow
	DB(ctx).Model(&DailyUsageSummary{}).
		Where("user_id = ? AND date = ?", userID, today).
		Select("instance_id, COALESCE(SUM(prompt_tokens), 0) as prompt_tokens, COALESCE(SUM(completion_tokens), 0) as completion_tokens, COALESCE(SUM(total_tokens), 0) as total_tokens, COALESCE(SUM(prompt_cache_read_tokens), 0) as prompt_cache_read_tokens, COALESCE(SUM(prompt_cache_write_tokens), 0) as prompt_cache_write_tokens").
		Group("instance_id").
		Scan(&rows)
	return rows
}

func thisMonday() time.Time {
	now := time.Now().In(hcommon.BusinessLocation())
	weekday := now.Weekday()
	if weekday == time.Sunday {
		weekday = 7
	}
	return localDateToUTC(now.AddDate(0, 0, -int(weekday-time.Monday)))
}
