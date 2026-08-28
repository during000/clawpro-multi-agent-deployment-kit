package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	hcommon "hatchery/common"
	"hatchery/controller/usergroup"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func initLLMProxyQuotaTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.Instance{},
		&model.SiteConfig{},
		&model.AIModel{},
		&model.InstanceModel{},
		&model.DailyUsageSummary{},
		&model.LLMUsageLog{},
		&model.Notification{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	restore := model.UseDBForTest(db)
	restoreTZ := hcommon.SetBusinessLocation(time.FixedZone("TEST", 8*60*60))
	return func() {
		restoreTZ()
		restore()
	}
}

func TestHandleLLMProxy_GlobalMonthlyQuotaExceededUsesLocalMonth(t *testing.T) {
	cleanup := initLLMProxyQuotaTestDB(t)
	defer cleanup()

	if err := model.DB(context.Background()).Create(&model.SiteConfig{
		GlobalTokenQuotaDay:    10,
		GlobalTokenQuotaPeriod: model.GlobalTokenQuotaPeriodMonth,
	}).Error; err != nil {
		t.Fatalf("seed site config: %v", err)
	}
	user := model.User{Username: "quota-user", TokenQuotaDay: -1}
	if err := model.DB(context.Background()).Create(&user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	aiModel := model.AIModel{
		Provider:  "openai",
		ModelID:   "gpt-test",
		ModelType: "openai-completions",
		QuotaDay:  -1,
		Enabled:   true,
	}
	if err := model.DB(context.Background()).Create(&aiModel).Error; err != nil {
		t.Fatalf("seed ai model: %v", err)
	}
	proxyToken := "sk-month-quota-test"
	inst := model.Instance{
		Name:       "quota-inst",
		InstanceId: "ins-month-quota-test",
		UserID:     user.ID,
		ProxyToken: &proxyToken,
		AIModelID:  aiModel.ID,
	}
	if err := model.DB(context.Background()).Create(&inst).Error; err != nil {
		t.Fatalf("seed instance: %v", err)
	}

	monthRule := model.TokenQuotaRule{Mode: model.QuotaModeMonth, Limit: 10}
	monthStart, monthEnd, active := monthRule.QuotaWindow(time.Now())
	if !active {
		t.Fatal("month rule should be active")
	}
	if monthEnd == nil {
		t.Fatal("month rule should have an end")
	}
	usageRows := []model.LLMUsageLog{
		{CreatedAt: monthStart.Add(-time.Minute), UserID: user.ID, InstanceID: inst.ID, AIModelID: aiModel.ID, TotalTokens: 100},
		{CreatedAt: monthStart.Add(time.Minute), UserID: user.ID, InstanceID: inst.ID, AIModelID: aiModel.ID, TotalTokens: 6},
		{CreatedAt: monthEnd.Add(-time.Minute), UserID: user.ID, InstanceID: inst.ID, AIModelID: aiModel.ID, TotalTokens: 4},
		{CreatedAt: *monthEnd, UserID: user.ID, InstanceID: inst.ID, AIModelID: aiModel.ID, TotalTokens: 100},
	}
	if err := model.DB(context.Background()).Create(&usageRows).Error; err != nil {
		t.Fatalf("seed usage: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-test"}`))
	req.Header.Set("Authorization", "Bearer "+proxyToken)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	HandleLLMProxy(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "全局每月令牌配额已用尽") {
		t.Fatalf("expected monthly quota message, got %s", rr.Body.String())
	}
}

func TestHandleLLMProxy_GlobalTokenQuotaRulesExceeded(t *testing.T) {
	cleanup := initLLMProxyQuotaTestDB(t)
	defer cleanup()
	ctx := context.Background()

	if err := model.DB(ctx).Create(&model.SiteConfig{
		GlobalTokenQuotaDay:   -1,
		GlobalTokenQuotaRules: `[{"mode":"day","limit":100}]`,
	}).Error; err != nil {
		t.Fatalf("seed site config: %v", err)
	}
	user := model.User{Username: "global-rules-user", TokenQuotaDay: -1}
	model.DB(ctx).Create(&user)
	aiModel := model.AIModel{Provider: "openai", ModelID: "gpt-test", ModelType: "openai-completions", QuotaDay: -1, Enabled: true}
	model.DB(ctx).Create(&aiModel)

	proxyToken := "sk-global-rules-test"
	inst := model.Instance{Name: "global-rules-inst", InstanceId: "ins-global-rules", UserID: user.ID, ProxyToken: &proxyToken, AIModelID: aiModel.ID}
	model.DB(ctx).Create(&inst)
	model.DB(ctx).Create(&model.LLMUsageLog{UserID: user.ID, InstanceID: inst.ID, AIModelID: aiModel.ID, TotalTokens: 100, CreatedAt: time.Now().UTC()})

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-test"}`))
	req.Header.Set("Authorization", "Bearer "+proxyToken)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	HandleLLMProxy(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "全局每日令牌配额已用尽") {
		t.Fatalf("expected global quota message, got %s", rr.Body.String())
	}
}

func TestHandleLLMProxy_ExplicitGroupGlobalTokenQuotaRulesExceeded(t *testing.T) {
	cleanup := initLLMProxyGroupQuotaTestDB(t)
	defer cleanup()
	ctx := context.Background()

	if err := model.DB(ctx).Create(&model.SiteConfig{
		GlobalTokenQuotaDay:   -1,
		GlobalTokenQuotaRules: `[{"mode":"day","limit":1000000}]`,
	}).Error; err != nil {
		t.Fatalf("seed site config: %v", err)
	}
	user := model.User{Username: "group-global-user", TokenQuotaDay: -1}
	model.DB(ctx).Create(&user)
	group := model.UserGroup{Name: "group-global", Source: "manual", FullPath: "group-global"}
	model.DB(ctx).Create(&group)
	model.DB(ctx).Create(&model.GroupClosure{AncestorID: group.ID, DescendantID: group.ID, Depth: 0})
	model.DB(ctx).Create(&model.GroupConfigBinding{
		ConfigType: model.ConfigTypePolicy,
		ConfigKey:  usergroup.PolicyKeyGlobalTokenQuotaRules,
		GroupID:    group.ID,
		ValueJSON:  `{"value":"[{\"mode\":\"day\",\"limit\":50}]"}`,
	})
	aiModel := model.AIModel{Provider: "openai", ModelID: "gpt-test", ModelType: "openai-completions", QuotaDay: -1, Enabled: true}
	model.DB(ctx).Create(&aiModel)

	proxyToken := "sk-group-global-rules"
	inst := model.Instance{Name: "group-global-inst", InstanceId: "ins-group-global", UserID: user.ID, ProxyToken: &proxyToken, AIModelID: aiModel.ID, GroupID: group.ID}
	model.DB(ctx).Create(&inst)
	model.DB(ctx).Create(&model.LLMUsageLog{UserID: user.ID, InstanceID: inst.ID, AIModelID: aiModel.ID, GroupID: group.ID, TotalTokens: 60, CreatedAt: time.Now().UTC()})

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-test"}`))
	req.Header.Set("Authorization", "Bearer "+proxyToken)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	HandleLLMProxy(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "全局每日令牌配额已用尽") {
		t.Fatalf("expected global quota message, got %s", rr.Body.String())
	}
}

func TestHandleLLMProxy_ExplicitGroupGlobalTokenQuotaRulesOverrideSiteGlobalRules(t *testing.T) {
	cleanup := initLLMProxyGroupQuotaTestDB(t)
	defer cleanup()
	ctx := context.Background()

	if err := model.DB(ctx).Create(&model.SiteConfig{
		GlobalTokenQuotaDay:   -1,
		GlobalTokenQuotaRules: `[{"mode":"day","limit":50}]`,
	}).Error; err != nil {
		t.Fatalf("seed site config: %v", err)
	}
	user := model.User{Username: "group-global-override-user", TokenQuotaDay: -1}
	model.DB(ctx).Create(&user)
	group := model.UserGroup{Name: "group-global-override", Source: "manual", FullPath: "group-global-override"}
	model.DB(ctx).Create(&group)
	model.DB(ctx).Create(&model.GroupClosure{AncestorID: group.ID, DescendantID: group.ID, Depth: 0})
	model.DB(ctx).Create(&model.GroupConfigBinding{
		ConfigType: model.ConfigTypePolicy,
		ConfigKey:  usergroup.PolicyKeyGlobalTokenQuotaRules,
		GroupID:    group.ID,
		ValueJSON:  `{"value":"[{\"mode\":\"day\",\"limit\":1000}]"}`,
	})
	aiModel := model.AIModel{Provider: "openai", ModelID: "gpt-test", ModelType: "openai-completions", QuotaDay: -1, Enabled: true}
	model.DB(ctx).Create(&aiModel)

	proxyToken := "sk-group-global-override-rules"
	inst := model.Instance{Name: "group-global-override-inst", InstanceId: "ins-group-global-override", UserID: user.ID, ProxyToken: &proxyToken, AIModelID: aiModel.ID, GroupID: group.ID}
	model.DB(ctx).Create(&inst)
	model.DB(ctx).Create(&model.LLMUsageLog{UserID: user.ID, InstanceID: inst.ID, AIModelID: aiModel.ID, GroupID: group.ID, TotalTokens: 60, CreatedAt: time.Now().UTC()})

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-test"}`))
	req.Header.Set("Authorization", "Bearer "+proxyToken)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	HandleLLMProxy(rr, req)

	if rr.Code == http.StatusForbidden && strings.Contains(rr.Body.String(), "全局每日令牌配额已用尽") {
		t.Fatalf("explicit group global rules should override site global rules, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleLLMProxy_GroupGlobalQuotaFallbackUsesGroupUsage(t *testing.T) {
	cleanup := initLLMProxyGroupQuotaTestDB(t)
	defer cleanup()
	ctx := context.Background()

	if err := model.DB(ctx).Create(&model.SiteConfig{
		GlobalTokenQuotaDay:   -1,
		GlobalTokenQuotaRules: `[{"mode":"day","limit":50}]`,
	}).Error; err != nil {
		t.Fatalf("seed site config: %v", err)
	}
	user := model.User{Username: "group-global-fallback-user", TokenQuotaDay: -1}
	model.DB(ctx).Create(&user)
	group := model.UserGroup{Name: "group-global-fallback", Source: "manual", FullPath: "group-global-fallback"}
	model.DB(ctx).Create(&group)
	model.DB(ctx).Create(&model.GroupClosure{AncestorID: group.ID, DescendantID: group.ID, Depth: 0})
	aiModel := model.AIModel{Provider: "openai", ModelID: "gpt-test", ModelType: "openai-completions", QuotaDay: -1, Enabled: true}
	model.DB(ctx).Create(&aiModel)

	proxyToken := "sk-group-global-fallback"
	inst := model.Instance{Name: "group-global-fallback-inst", InstanceId: "ins-group-global-fallback", UserID: user.ID, ProxyToken: &proxyToken, AIModelID: aiModel.ID, GroupID: group.ID}
	model.DB(ctx).Create(&inst)
	model.DB(ctx).Create(&model.LLMUsageLog{UserID: user.ID, InstanceID: inst.ID, AIModelID: aiModel.ID, GroupID: group.ID, TotalTokens: 40, CreatedAt: time.Now().UTC()})
	model.DB(ctx).Create(&model.LLMUsageLog{UserID: user.ID, InstanceID: inst.ID, AIModelID: aiModel.ID, GroupID: group.ID + 100, TotalTokens: 20, CreatedAt: time.Now().UTC()})

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-test"}`))
	req.Header.Set("Authorization", "Bearer "+proxyToken)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	HandleLLMProxy(rr, req)

	if rr.Code == http.StatusForbidden && strings.Contains(rr.Body.String(), "全局每日令牌配额已用尽") {
		t.Fatalf("group fallback global quota should count only current group usage, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleLLMProxy_UserTokenQuotaRulesExceeded(t *testing.T) {
	cleanup := initLLMProxyQuotaTestDB(t)
	defer cleanup()

	// Create site config with no global limit
	if err := model.DB(context.Background()).Create(&model.SiteConfig{
		GlobalTokenQuotaDay: -1,
	}).Error; err != nil {
		t.Fatalf("seed site config: %v", err)
	}

	// Create user with token_quota_rules (day limit = 100 tokens)
	user := model.User{
		Username:        "rules-user",
		TokenQuotaDay:   100,
		TokenQuotaRules: `[{"mode":"day","limit":100}]`,
	}
	if err := model.DB(context.Background()).Create(&user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	aiModel := model.AIModel{
		Provider:  "openai",
		ModelID:   "gpt-test",
		ModelType: "openai-completions",
		QuotaDay:  -1,
		Enabled:   true,
	}
	if err := model.DB(context.Background()).Create(&aiModel).Error; err != nil {
		t.Fatalf("seed ai model: %v", err)
	}

	proxyToken := "sk-rules-quota-test"
	inst := model.Instance{
		Name:       "rules-inst",
		InstanceId: "ins-rules-quota-test",
		UserID:     user.ID,
		ProxyToken: &proxyToken,
		AIModelID:  aiModel.ID,
	}
	if err := model.DB(context.Background()).Create(&inst).Error; err != nil {
		t.Fatalf("seed instance: %v", err)
	}

	// Seed usage logs that exceed the daily limit
	now := time.Now().UTC()
	for i := 0; i < 5; i++ {
		model.DB(context.Background()).Create(&model.LLMUsageLog{
			UserID:      user.ID,
			InstanceID:  inst.ID,
			AIModelID:   aiModel.ID,
			TotalTokens: 30,
			CreatedAt:   now.Add(-time.Duration(i) * time.Minute),
		})
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-test"}`))
	req.Header.Set("Authorization", "Bearer "+proxyToken)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	HandleLLMProxy(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "用户每日令牌配额已用尽") {
		t.Fatalf("expected user quota message, got %s", rr.Body.String())
	}
}

func TestHandleLLMProxy_UserTokenQuotaRulesNotExceeded(t *testing.T) {
	cleanup := initLLMProxyQuotaTestDB(t)
	defer cleanup()

	// Create site config with no global limit
	if err := model.DB(context.Background()).Create(&model.SiteConfig{
		GlobalTokenQuotaDay: -1,
	}).Error; err != nil {
		t.Fatalf("seed site config: %v", err)
	}

	// Create user with generous quota
	user := model.User{
		Username:        "generous-user",
		TokenQuotaDay:   1000000,
		TokenQuotaRules: `[{"mode":"day","limit":1000000}]`,
	}
	if err := model.DB(context.Background()).Create(&user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	aiModel := model.AIModel{
		Provider:  "openai",
		ModelID:   "gpt-test",
		ModelType: "openai-completions",
		QuotaDay:  -1,
		Enabled:   true,
	}
	if err := model.DB(context.Background()).Create(&aiModel).Error; err != nil {
		t.Fatalf("seed ai model: %v", err)
	}

	proxyToken := "sk-generous-quota-test"
	inst := model.Instance{
		Name:       "generous-inst",
		InstanceId: "ins-generous-quota-test",
		UserID:     user.ID,
		ProxyToken: &proxyToken,
		AIModelID:  aiModel.ID,
	}
	if err := model.DB(context.Background()).Create(&inst).Error; err != nil {
		t.Fatalf("seed instance: %v", err)
	}

	// Seed minimal usage
	model.DB(context.Background()).Create(&model.LLMUsageLog{
		UserID:      user.ID,
		InstanceID:  inst.ID,
		AIModelID:   aiModel.ID,
		TotalTokens: 10,
		CreatedAt:   time.Now().UTC(),
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-test"}`))
	req.Header.Set("Authorization", "Bearer "+proxyToken)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	HandleLLMProxy(rr, req)

	// Should NOT be quota-blocked (will fail at upstream connection, which is fine)
	if rr.Code == http.StatusForbidden {
		t.Fatalf("should not be quota-blocked, got 403 body=%s", rr.Body.String())
	}
}

// initLLMProxyGroupQuotaTestDB 初始化包含分组相关模型的测试 DB
func initLLMProxyGroupQuotaTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.Instance{},
		&model.SiteConfig{},
		&model.AIModel{},
		&model.InstanceModel{},
		&model.DailyUsageSummary{},
		&model.LLMUsageLog{},
		&model.Notification{},
		&model.UserGroup{},
		&model.GroupClosure{},
		&model.UserGroupMember{},
		&model.GroupConfigBinding{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	restore := model.UseDBForTest(db)
	restoreTZ := hcommon.SetBusinessLocation(time.FixedZone("TEST", 8*60*60))
	return func() {
		restoreTZ()
		restore()
	}
}

// TestHandleLLMProxy_GroupQuotaRulesResolved 验证有分组时运行时解析组的 token_quota_rules 策略
// 场景：用户表 TokenQuotaDay=100（很低），但组策略 token_quota_rules 限制为 500万
// 用量 200 tokens → 不应被限额
func TestHandleLLMProxy_GroupQuotaRulesResolved(t *testing.T) {
	cleanup := initLLMProxyGroupQuotaTestDB(t)
	defer cleanup()
	ctx := context.Background()

	if err := model.DB(ctx).Create(&model.SiteConfig{GlobalTokenQuotaDay: -1}).Error; err != nil {
		t.Fatalf("seed site config: %v", err)
	}

	user := model.User{Username: "grp-rules-user", TokenQuotaDay: 100}
	model.DB(ctx).Create(&user)

	group := model.UserGroup{Name: "high-rules-group", Source: "manual", FullPath: "high-rules-group"}
	model.DB(ctx).Create(&group)
	model.DB(ctx).Create(&model.GroupClosure{AncestorID: group.ID, DescendantID: group.ID, Depth: 0})

	// 组策略：token_quota_rules = day limit 5000000
	now := time.Now().Unix()
	model.DB(ctx).Create(&model.GroupConfigBinding{
		ConfigType: model.ConfigTypePolicy,
		ConfigKey:  usergroup.PolicyKeyTokenQuotaRules,
		GroupID:    group.ID,
		ValueJSON:  `{"value":"[{\"mode\":\"day\",\"limit\":5000000}]"}`,
	})
	_ = now

	aiModel := model.AIModel{Provider: "openai", ModelID: "gpt-test", ModelType: "openai-completions", QuotaDay: -1, Enabled: true}
	model.DB(ctx).Create(&aiModel)

	proxyToken := "sk-grp-rules-test"
	inst := model.Instance{Name: "grp-inst", InstanceId: "ins-grp-rules", UserID: user.ID, ProxyToken: &proxyToken, AIModelID: aiModel.ID, GroupID: group.ID}
	model.DB(ctx).Create(&inst)

	// 用量：200 tokens（远低于组策略 500万）
	model.DB(ctx).Create(&model.LLMUsageLog{UserID: user.ID, InstanceID: inst.ID, AIModelID: aiModel.ID, TotalTokens: 200, CreatedAt: time.Now().UTC()})

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-test"}`))
	req.Header.Set("Authorization", "Bearer "+proxyToken)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	HandleLLMProxy(rr, req)

	// 不应被用户配额拦截（组策略 500万 > 用量 200）
	if rr.Code == http.StatusForbidden && strings.Contains(rr.Body.String(), "用户每日令牌配额已用尽") {
		t.Fatalf("应走组策略（500万），不应被用户表限额（100）拦截。got: %s", rr.Body.String())
	}
}

// TestHandleLLMProxy_GroupQuotaDayFallback 验证有分组但只配了 token_quota_day policy 时的 fallback
// 场景：组策略 token_quota_day=50，用量 100 → 应被限额
func TestHandleLLMProxy_GroupQuotaDayFallback(t *testing.T) {
	cleanup := initLLMProxyGroupQuotaTestDB(t)
	defer cleanup()
	ctx := context.Background()

	if err := model.DB(ctx).Create(&model.SiteConfig{GlobalTokenQuotaDay: -1}).Error; err != nil {
		t.Fatalf("seed site config: %v", err)
	}

	user := model.User{Username: "grp-day-user", TokenQuotaDay: -1}
	model.DB(ctx).Create(&user)

	group := model.UserGroup{Name: "low-day-group", Source: "manual", FullPath: "low-day-group"}
	model.DB(ctx).Create(&group)
	model.DB(ctx).Create(&model.GroupClosure{AncestorID: group.ID, DescendantID: group.ID, Depth: 0})

	// 组策略：token_quota_day = 50（旧格式）
	model.DB(ctx).Create(&model.GroupConfigBinding{
		ConfigType: model.ConfigTypePolicy,
		ConfigKey:  usergroup.PolicyKeyTokenQuotaDay,
		GroupID:    group.ID,
		ValueJSON:  `{"value":50}`,
	})

	aiModel := model.AIModel{Provider: "openai", ModelID: "gpt-test", ModelType: "openai-completions", QuotaDay: -1, Enabled: true}
	model.DB(ctx).Create(&aiModel)

	proxyToken := "sk-grp-day-test"
	inst := model.Instance{Name: "grp-day-inst", InstanceId: "ins-grp-day", UserID: user.ID, ProxyToken: &proxyToken, AIModelID: aiModel.ID, GroupID: group.ID}
	model.DB(ctx).Create(&inst)

	// 用量超过组策略
	model.DB(ctx).Create(&model.LLMUsageLog{UserID: user.ID, InstanceID: inst.ID, AIModelID: aiModel.ID, TotalTokens: 100, CreatedAt: time.Now().UTC(), GroupID: group.ID})

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-test"}`))
	req.Header.Set("Authorization", "Bearer "+proxyToken)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	HandleLLMProxy(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "用户每日令牌配额已用尽") {
		t.Fatalf("expected user quota exceeded, got %s", rr.Body.String())
	}
}

// TestHandleLLMProxy_GroupLegacyUnlimitedDayDoesNotFallbackToUserRules 验证旧分组策略
// token_quota_day=-1 且无 token_quota_rules 时表示该组显式无限制，不应 fallback 到用户限额。
func TestHandleLLMProxy_GroupLegacyUnlimitedDayDoesNotFallbackToUserRules(t *testing.T) {
	cleanup := initLLMProxyGroupQuotaTestDB(t)
	defer cleanup()
	ctx := context.Background()

	if err := model.DB(ctx).Create(&model.SiteConfig{GlobalTokenQuotaDay: -1}).Error; err != nil {
		t.Fatalf("seed site config: %v", err)
	}

	user := model.User{Username: "grp-legacy-unlimited-user", TokenQuotaDay: 100}
	model.DB(ctx).Create(&user)

	group := model.UserGroup{Name: "legacy-unlimited-group", Source: "manual", FullPath: "legacy-unlimited-group"}
	model.DB(ctx).Create(&group)
	model.DB(ctx).Create(&model.GroupClosure{AncestorID: group.ID, DescendantID: group.ID, Depth: 0})

	// 旧格式组策略：token_quota_day=-1，没有 token_quota_rules。
	model.DB(ctx).Create(&model.GroupConfigBinding{
		ConfigType: model.ConfigTypePolicy,
		ConfigKey:  usergroup.PolicyKeyTokenQuotaDay,
		GroupID:    group.ID,
		ValueJSON:  `{"value":-1}`,
	})

	aiModel := model.AIModel{Provider: "openai", ModelID: "gpt-test", ModelType: "openai-completions", QuotaDay: -1, Enabled: true}
	model.DB(ctx).Create(&aiModel)

	proxyToken := "sk-grp-legacy-unlimited-test"
	inst := model.Instance{Name: "legacy-unlimited-inst", InstanceId: "ins-legacy-unlimited", UserID: user.ID, ProxyToken: &proxyToken, AIModelID: aiModel.ID, GroupID: group.ID}
	model.DB(ctx).Create(&inst)

	// 用户个人配额是 100；若错误 fallback 用户配额，这里会被拦截。
	model.DB(ctx).Create(&model.LLMUsageLog{UserID: user.ID, InstanceID: inst.ID, AIModelID: aiModel.ID, TotalTokens: 200, CreatedAt: time.Now().UTC(), GroupID: group.ID})

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-test"}`))
	req.Header.Set("Authorization", "Bearer "+proxyToken)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	HandleLLMProxy(rr, req)

	if rr.Code == http.StatusForbidden && strings.Contains(rr.Body.String(), "用户每日令牌配额已用尽") {
		t.Fatalf("legacy group token_quota_day=-1 should be explicit unlimited, got user quota block: %s", rr.Body.String())
	}
}

// TestHandleLLMProxy_NoGroupFallbackToUserField 验证无分组时 fallback 到用户字段
func TestHandleLLMProxy_NoGroupFallbackToUserField(t *testing.T) {
	cleanup := initLLMProxyGroupQuotaTestDB(t)
	defer cleanup()
	ctx := context.Background()

	if err := model.DB(ctx).Create(&model.SiteConfig{GlobalTokenQuotaDay: -1}).Error; err != nil {
		t.Fatalf("seed site config: %v", err)
	}

	// 用户 TokenQuotaDay=100，无分组实例
	user := model.User{Username: "no-grp-user", TokenQuotaDay: 100}
	model.DB(ctx).Create(&user)

	aiModel := model.AIModel{Provider: "openai", ModelID: "gpt-test", ModelType: "openai-completions", QuotaDay: -1, Enabled: true}
	model.DB(ctx).Create(&aiModel)

	proxyToken := "sk-no-grp-test"
	inst := model.Instance{Name: "no-grp-inst", InstanceId: "ins-no-grp", UserID: user.ID, ProxyToken: &proxyToken, AIModelID: aiModel.ID, GroupID: 0}
	model.DB(ctx).Create(&inst)

	// 用量超过用户字段限额
	model.DB(ctx).Create(&model.LLMUsageLog{UserID: user.ID, InstanceID: inst.ID, AIModelID: aiModel.ID, TotalTokens: 200, CreatedAt: time.Now().UTC()})

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-test"}`))
	req.Header.Set("Authorization", "Bearer "+proxyToken)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	HandleLLMProxy(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "用户每日令牌配额已用尽") {
		t.Fatalf("expected user quota exceeded, got %s", rr.Body.String())
	}
}

// TestHandleLLMProxy_GroupNoPolicyFallbackToSiteDefaultRules 验证有分组但组无配额策略时，fallback 到站点默认用户 rules
// 场景：用户自身 rules 很高，site default rules={day,50}，组无策略
// 用量 100 > 50 → 应按 site default 被限额
func TestHandleLLMProxy_GroupNoPolicyFallbackToSiteDefaultRules(t *testing.T) {
	cleanup := initLLMProxyGroupQuotaTestDB(t)
	defer cleanup()
	ctx := context.Background()

	if err := model.DB(ctx).Create(&model.SiteConfig{
		GlobalTokenQuotaDay:    -1,
		DefaultTokenQuotaDay:   -1,
		DefaultTokenQuotaRules: `[{"mode":"day","limit":50}]`,
	}).Error; err != nil {
		t.Fatalf("seed site config: %v", err)
	}

	// 用户自身 rules 不应在有分组且组无策略时生效。
	user := model.User{Username: "migrated-user", TokenQuotaDay: -1, TokenQuotaRules: `[{"mode":"day","limit":1000000}]`}
	model.DB(ctx).Create(&user)

	// 组存在但没有任何配额策略
	group := model.UserGroup{Name: "no-policy-group", Source: "manual", FullPath: "no-policy-group"}
	model.DB(ctx).Create(&group)
	model.DB(ctx).Create(&model.GroupClosure{AncestorID: group.ID, DescendantID: group.ID, Depth: 0})

	aiModel := model.AIModel{Provider: "openai", ModelID: "gpt-test", ModelType: "openai-completions", QuotaDay: -1, Enabled: true}
	model.DB(ctx).Create(&aiModel)

	proxyToken := "sk-no-policy-grp"
	inst := model.Instance{Name: "np-inst", InstanceId: "ins-no-policy", UserID: user.ID, ProxyToken: &proxyToken, AIModelID: aiModel.ID, GroupID: group.ID}
	model.DB(ctx).Create(&inst)

	// 用量 100 > site default 限额 50，但远低于用户自身限额。
	model.DB(ctx).Create(&model.LLMUsageLog{UserID: user.ID, InstanceID: inst.ID, AIModelID: aiModel.ID, TotalTokens: 100, CreatedAt: time.Now().UTC(), GroupID: group.ID})

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-test"}`))
	req.Header.Set("Authorization", "Bearer "+proxyToken)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	HandleLLMProxy(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 (site default rules fallback), got %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "用户每日令牌配额已用尽") {
		t.Fatalf("expected user quota exceeded, got %s", rr.Body.String())
	}
}

// TestHandleLLMProxy_GroupedUserUngroupedInstanceUsesSiteConfig 验证用户有分组、
// 但实例 group_id=0 时，运行时应回落到 SiteConfig.DefaultTokenQuotaRules，
// 而不是继续使用创建时烙印在用户表上的 token 配额。
//
// 这是"用户被加入分组后，无分组旧实例跟随站点默认单用户配额"的核心回归用例。
func TestHandleLLMProxy_GroupedUserUngroupedInstanceUsesSiteConfig(t *testing.T) {
	cleanup := initLLMProxyGroupQuotaTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// 站点默认配额：每日 100 tokens；无全局限额。
	if err := model.DB(ctx).Create(&model.SiteConfig{
		GlobalTokenQuotaDay:    -1,
		DefaultTokenQuotaDay:   -1,
		DefaultTokenQuotaRules: `[{"mode":"day","limit":100}]`,
	}).Error; err != nil {
		t.Fatalf("seed site config: %v", err)
	}

	// 用户字段全空（模拟修复后新建的无分组用户）。
	user := model.User{Username: "no-grp-empty-user", TokenQuotaDay: -1, TokenQuotaRules: ""}
	if err := model.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	group := model.UserGroup{Name: "grouped-user", FullPath: "grouped-user", Source: model.GroupSourceManual}
	if err := model.DB(ctx).Create(&group).Error; err != nil {
		t.Fatalf("seed group: %v", err)
	}
	if err := model.DB(ctx).Create(&model.UserGroupMember{UserID: user.ID, UserGroupID: group.ID, Source: model.MemberSourceManual}).Error; err != nil {
		t.Fatalf("seed membership: %v", err)
	}

	aiModel := model.AIModel{Provider: "openai", ModelID: "gpt-test", ModelType: "openai-completions", QuotaDay: -1, Enabled: true}
	if err := model.DB(ctx).Create(&aiModel).Error; err != nil {
		t.Fatalf("seed ai model: %v", err)
	}

	proxyToken := "sk-no-grp-empty-user"
	inst := model.Instance{Name: "no-grp-empty-inst", InstanceId: "ins-no-grp-empty", UserID: user.ID, ProxyToken: &proxyToken, AIModelID: aiModel.ID, GroupID: 0}
	if err := model.DB(ctx).Create(&inst).Error; err != nil {
		t.Fatalf("seed instance: %v", err)
	}

	// 用量 150 > 站点默认 100 → 应被站点默认配额拦截。
	model.DB(ctx).Create(&model.LLMUsageLog{UserID: user.ID, InstanceID: inst.ID, AIModelID: aiModel.ID, TotalTokens: 150, CreatedAt: time.Now().UTC()})

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-test"}`))
	req.Header.Set("Authorization", "Bearer "+proxyToken)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	HandleLLMProxy(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 (site config fallback for grouped user with ungrouped instance), got %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "用户每日令牌配额已用尽") {
		t.Fatalf("expected user quota exceeded, got %s", rr.Body.String())
	}
}

// TestHandleLLMProxy_NoGroupNoUserFieldNoSiteDefault 验证无分组用户字段为
// TokenQuotaDay=-1 / TokenQuotaRules="" 时，应按用户自身的无限制配置放行。
//
// 边界保障：避免把 token_quota_day=-1 误判为"未配置"后回落到站点默认。
func TestHandleLLMProxy_NoGroupNoUserFieldNoSiteDefault(t *testing.T) {
	cleanup := initLLMProxyQuotaTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// 即使站点默认有低限额，真正无分组用户也应按用户自身 token_quota_day=-1 放行。
	if err := model.DB(ctx).Create(&model.SiteConfig{
		GlobalTokenQuotaDay:    -1,
		DefaultTokenQuotaDay:   -1,
		DefaultTokenQuotaRules: `[{"mode":"day","limit":100}]`,
	}).Error; err != nil {
		t.Fatalf("seed site config: %v", err)
	}

	user := model.User{Username: "fully-unlimited-user", TokenQuotaDay: -1, TokenQuotaRules: ""}
	if err := model.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	aiModel := model.AIModel{Provider: "openai", ModelID: "gpt-test", ModelType: "openai-completions", QuotaDay: -1, Enabled: true}
	model.DB(ctx).Create(&aiModel)

	proxyToken := "sk-fully-unlimited"
	inst := model.Instance{Name: "fu-inst", InstanceId: "ins-fully-unlimited", UserID: user.ID, ProxyToken: &proxyToken, AIModelID: aiModel.ID, GroupID: 0}
	model.DB(ctx).Create(&inst)

	// 即使有大量用量，也不应被配额拦截（仅可能因上游连接失败而非 403）。
	model.DB(ctx).Create(&model.LLMUsageLog{UserID: user.ID, InstanceID: inst.ID, AIModelID: aiModel.ID, TotalTokens: 999999, CreatedAt: time.Now().UTC()})

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-test"}`))
	req.Header.Set("Authorization", "Bearer "+proxyToken)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	HandleLLMProxy(rr, req)

	if rr.Code == http.StatusForbidden {
		t.Fatalf("should not be quota-blocked when both user and site config are empty, got 403 body=%s", rr.Body.String())
	}
}
