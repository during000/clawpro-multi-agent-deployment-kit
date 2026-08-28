package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"hatchery/controller/usergroup"
	"hatchery/model"
)

func quotaDataReq(t *testing.T, path, username string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Accept", "application/json")
	session, err := Store.Get(req, "hatchery-session")
	if err != nil {
		t.Fatalf("获取 session 失败: %v", err)
	}
	session.Values["username"] = username
	rr := httptest.NewRecorder()
	if err := session.Save(req, rr); err != nil {
		t.Fatalf("保存 session 失败: %v", err)
	}
	for _, cookie := range rr.Result().Cookies() {
		req.AddCookie(cookie)
	}
	return req
}

func TestHandleQuotaData_OrderByDefault(t *testing.T) {
	cleanup := initUsageDataTestEnv(t)
	defer cleanup()

	var alice, bob model.User
	if err := model.DB(context.Background()).Where("username = ?", "alice").First(&alice).Error; err != nil {
		t.Fatalf("查询 alice 失败: %v", err)
	}
	if err := model.DB(context.Background()).Where("username = ?", "bob").First(&bob).Error; err != nil {
		t.Fatalf("查询 bob 失败: %v", err)
	}

	modelA := model.AIModel{ModelID: "model-a", ModelName: "Model A", Provider: "test"}
	modelB := model.AIModel{ModelID: "model-b", ModelName: "Model B", Provider: "test"}
	model.DB(context.Background()).Create(&modelA)
	model.DB(context.Background()).Create(&modelB)

	day := time.Date(2026, 5, 8, 0, 0, 0, 0, time.Local)
	model.DB(context.Background()).Create(&model.DailyUsageSummary{Date: day, UserID: alice.ID, AIModelID: modelA.ID, TotalTokens: 100, RequestCount: 5, PromptTokens: 60, CompletionTokens: 40})
	model.DB(context.Background()).Create(&model.DailyUsageSummary{Date: day, UserID: alice.ID, AIModelID: modelB.ID, TotalTokens: 200, RequestCount: 2, PromptTokens: 120, CompletionTokens: 80})
	model.DB(context.Background()).Create(&model.DailyUsageSummary{Date: day, UserID: bob.ID, AIModelID: modelB.ID, TotalTokens: 999, RequestCount: 9, PromptTokens: 600, CompletionTokens: 399})

	req := quotaDataReq(t, "/quota/data?start_date=2026-05-08&end_date=2026-05-08&group_by=model&order=desc", "alice")
	w := httptest.NewRecorder()
	HandleQuotaData(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200, got %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Rows []struct {
			AIModelID    uint  `json:"ai_model_id"`
			TotalTokens  int64 `json:"total_tokens"`
			RequestCount int64 `json:"request_count"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是有效 JSON: %v", err)
	}
	if len(resp.Rows) != 2 {
		t.Fatalf("应只返回当前用户的 2 行模型用量, got %d", len(resp.Rows))
	}
	if resp.Rows[0].AIModelID != modelB.ID || resp.Rows[0].TotalTokens != 200 {
		t.Errorf("默认应按 total_tokens 降序，第一行 = %+v", resp.Rows[0])
	}
}

func TestHandleQuotaData_ReturnsTokenQuotaUsagesPerRule(t *testing.T) {
	cleanup := initUsageDataTestEnv(t)
	defer cleanup()

	var alice model.User
	if err := model.DB(context.Background()).Where("username = ?", "alice").First(&alice).Error; err != nil {
		t.Fatalf("查询 alice 失败: %v", err)
	}
	alice.TokenQuotaDay = -1
	alice.TokenQuotaRules = `[{"mode":"day","limit":1000},{"mode":"custom","limit":5000,"start":1,"refresh":"none"}]`
	model.DB(context.Background()).Save(&alice)

	now := timeInCurrentTokenQuotaDayWindow(t)
	model.DB(context.Background()).Create(&model.LLMUsageLog{
		UserID:      alice.ID,
		TotalTokens: 250,
		CreatedAt:   now,
	})

	req := quotaDataReq(t, "/quota/data", "alice")
	w := httptest.NewRecorder()
	HandleQuotaData(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200, got %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		TokenQuotaRules  []model.TokenQuotaRule `json:"token_quota_rules"`
		TokenQuotaUsages []struct {
			RuleIndex   int    `json:"rule_index"`
			Used        int64  `json:"used"`
			PeriodStart *int64 `json:"period_start"`
			PeriodEnd   *int64 `json:"period_end"`
			Active      bool   `json:"active"`
		} `json:"token_quota_usages"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是有效 JSON: %v", err)
	}
	if len(resp.TokenQuotaRules) != 2 {
		t.Fatalf("期望返回 2 条 token_quota_rules, got %d", len(resp.TokenQuotaRules))
	}
	if len(resp.TokenQuotaUsages) != 2 {
		t.Fatalf("期望返回 2 条 token_quota_usages, got %d", len(resp.TokenQuotaUsages))
	}
	if resp.TokenQuotaUsages[0].RuleIndex != 0 || resp.TokenQuotaUsages[0].Used != 250 {
		t.Fatalf("第一条 usage = %+v, want rule_index=0 used=250", resp.TokenQuotaUsages[0])
	}
	if !resp.TokenQuotaUsages[0].Active {
		t.Fatalf("第一条 usage should be active: %+v", resp.TokenQuotaUsages[0])
	}
	from, to, active := resp.TokenQuotaRules[0].QuotaWindow(time.Now())
	if !active {
		t.Fatalf("day rule should be active")
	}
	if to == nil {
		t.Fatalf("day rule should have an end")
	}
	if resp.TokenQuotaUsages[0].PeriodStart == nil || *resp.TokenQuotaUsages[0].PeriodStart != from.Unix() || resp.TokenQuotaUsages[0].PeriodEnd == nil || *resp.TokenQuotaUsages[0].PeriodEnd != to.Unix() {
		t.Fatalf("第一条 usage period = %v-%v, want %d-%d", resp.TokenQuotaUsages[0].PeriodStart, resp.TokenQuotaUsages[0].PeriodEnd, from.Unix(), to.Unix())
	}
}

func TestHandleQuotaData_ReturnsOpenEndedCustomTokenQuotaUsage(t *testing.T) {
	cleanup := initUsageDataTestEnv(t)
	defer cleanup()

	var alice model.User
	if err := model.DB(context.Background()).Where("username = ?", "alice").First(&alice).Error; err != nil {
		t.Fatalf("查询 alice 失败: %v", err)
	}
	startTime := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)
	start := startTime.Unix()
	alice.TokenQuotaDay = -1
	alice.TokenQuotaRules = `[{"mode":"custom","limit":700000,"start":` + strconv.FormatInt(start, 10) + `,"refresh":"none"}]`
	model.DB(context.Background()).Save(&alice)
	model.DB(context.Background()).Where("user_id = ?", alice.ID).Delete(&model.LLMUsageLog{})

	model.DB(context.Background()).Create(&model.LLMUsageLog{UserID: alice.ID, TotalTokens: 15874, CreatedAt: startTime.Add(time.Minute)})
	model.DB(context.Background()).Create(&model.LLMUsageLog{UserID: alice.ID, TotalTokens: 999, CreatedAt: startTime.Add(-time.Hour)})

	req := quotaDataReq(t, "/quota/data", "alice")
	w := httptest.NewRecorder()
	HandleQuotaData(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200, got %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		TokenQuotaUsages []struct {
			RuleIndex   int    `json:"rule_index"`
			Used        int64  `json:"used"`
			PeriodStart *int64 `json:"period_start"`
			PeriodEnd   *int64 `json:"period_end"`
			Active      bool   `json:"active"`
		} `json:"token_quota_usages"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是有效 JSON: %v", err)
	}
	if len(resp.TokenQuotaUsages) != 1 {
		t.Fatalf("期望返回 1 条 token_quota_usages, got %d", len(resp.TokenQuotaUsages))
	}
	usage := resp.TokenQuotaUsages[0]
	if usage.RuleIndex != 0 || usage.Used != 15874 || !usage.Active || usage.PeriodStart == nil || *usage.PeriodStart != start || usage.PeriodEnd != nil {
		var gotStart any
		if usage.PeriodStart != nil {
			gotStart = *usage.PeriodStart
		}
		t.Fatalf("open-ended usage 不符合预期: rule_index=%d used=%d active=%v period_start=%v period_end=%v want_start=%d",
			usage.RuleIndex, usage.Used, usage.Active, gotStart, usage.PeriodEnd, start)
	}
}

func TestTokenQuotaUsages_InactiveRuleReturnsZeroWindow(t *testing.T) {
	start := time.Now().UTC().Add(-48 * time.Hour).Unix()
	end := time.Now().UTC().Add(-24 * time.Hour).Unix()
	usages := userTokenQuotaUsages(context.Background(), 1, 0, []model.TokenQuotaRule{{
		Mode:    model.QuotaModeCustom,
		Limit:   1000,
		Start:   &start,
		End:     &end,
		Refresh: model.QuotaRefreshNone,
	}})
	if len(usages) != 1 {
		t.Fatalf("期望返回 1 条 usage, got %d", len(usages))
	}
	usage := usages[0]
	if usage.Active || usage.Used != 0 || usage.PeriodStart == nil || usage.PeriodEnd == nil || *usage.PeriodStart != 0 || *usage.PeriodEnd != 0 {
		t.Fatalf("inactive usage 不符合预期: %+v", usage)
	}
}

func TestHandleQuotaData_ReturnsYearTokenQuotaUsage(t *testing.T) {
	cleanup := initUsageDataTestEnv(t)
	defer cleanup()

	var alice model.User
	if err := model.DB(context.Background()).Where("username = ?", "alice").First(&alice).Error; err != nil {
		t.Fatalf("查询 alice 失败: %v", err)
	}
	alice.TokenQuotaDay = -1
	alice.TokenQuotaRules = `[{"mode":"year","limit":-1}]`
	model.DB(context.Background()).Save(&alice)

	rule := model.TokenQuotaRule{Mode: model.QuotaModeYear, Limit: -1}
	inWindow := timeInCurrentTokenQuotaWindow(t, rule)
	from, _, _ := rule.QuotaWindow(time.Now())
	model.DB(context.Background()).Create(&model.LLMUsageLog{UserID: alice.ID, TotalTokens: 700, CreatedAt: inWindow})
	model.DB(context.Background()).Create(&model.LLMUsageLog{UserID: alice.ID, TotalTokens: 900, CreatedAt: from.Add(-time.Second)})

	req := quotaDataReq(t, "/quota/data", "alice")
	w := httptest.NewRecorder()
	HandleQuotaData(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200, got %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		TokenQuotaRules  []model.TokenQuotaRule `json:"token_quota_rules"`
		TokenQuotaUsages []struct {
			RuleIndex int   `json:"rule_index"`
			Used      int64 `json:"used"`
		} `json:"token_quota_usages"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是有效 JSON: %v", err)
	}
	if len(resp.TokenQuotaRules) != 1 || resp.TokenQuotaRules[0].Mode != model.QuotaModeYear {
		t.Fatalf("期望返回 year token_quota_rules, got %+v", resp.TokenQuotaRules)
	}
	if resp.TokenQuotaRules[0].Limit != -1 {
		t.Fatalf("期望 year rule limit=-1, got %+v", resp.TokenQuotaRules)
	}
	if len(resp.TokenQuotaUsages) != 1 || resp.TokenQuotaUsages[0].RuleIndex != 0 || resp.TokenQuotaUsages[0].Used != 700 {
		t.Fatalf("期望 year usage used=700, got %+v", resp.TokenQuotaUsages)
	}
}

func TestHandleQuotaData_GroupIDUsesGroupTokenQuotaRulesAndUsage(t *testing.T) {
	cleanup := initUsageDataTestEnv(t)
	defer cleanup()

	var alice model.User
	if err := model.DB(context.Background()).Where("username = ?", "alice").First(&alice).Error; err != nil {
		t.Fatalf("查询 alice 失败: %v", err)
	}
	alice.TokenQuotaDay = -1
	alice.TokenQuotaRules = `[{"mode":"day","limit":100}]`
	model.DB(context.Background()).Save(&alice)

	group := model.UserGroup{Name: "Group A", FullPath: "Group A", Source: model.GroupSourceManual}
	model.DB(context.Background()).Create(&group)
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: group.ID, DescendantID: group.ID, Depth: 0})
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: model.ConfigTypePolicy,
		ConfigKey:  usergroup.PolicyKeyTokenQuotaRules,
		GroupID:    group.ID,
		ValueJSON:  `{"value":"[{\"mode\":\"day\",\"limit\":1000}]"}`,
	})

	now := timeInCurrentTokenQuotaDayWindow(t)
	model.DB(context.Background()).Create(&model.LLMUsageLog{UserID: alice.ID, GroupID: group.ID, TotalTokens: 250, CreatedAt: now})
	model.DB(context.Background()).Create(&model.LLMUsageLog{UserID: alice.ID, GroupID: group.ID + 100, TotalTokens: 900, CreatedAt: now})

	req := quotaDataReq(t, "/quota/data?group_id="+strconv.FormatUint(uint64(group.ID), 10), "alice")
	w := httptest.NewRecorder()
	HandleQuotaData(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200, got %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		TokenQuotaRules  []model.TokenQuotaRule `json:"token_quota_rules"`
		TokenQuotaUsages []struct {
			RuleIndex int   `json:"rule_index"`
			Used      int64 `json:"used"`
		} `json:"token_quota_usages"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是有效 JSON: %v", err)
	}
	if len(resp.TokenQuotaRules) != 1 || resp.TokenQuotaRules[0].Limit != 1000 {
		t.Fatalf("期望使用分组 token_quota_rules limit=1000, got %+v", resp.TokenQuotaRules)
	}
	if len(resp.TokenQuotaUsages) != 1 || resp.TokenQuotaUsages[0].Used != 250 {
		t.Fatalf("期望只统计该分组 used=250, got %+v", resp.TokenQuotaUsages)
	}
}

func TestHandleQuotaData_GroupIDNoPolicyUsesSiteDefaultTokenQuotaRules(t *testing.T) {
	cleanup := initUsageDataTestEnv(t)
	defer cleanup()
	ctx := context.Background()

	var cfg model.SiteConfig
	if err := model.DB(ctx).First(&cfg).Error; err != nil {
		t.Fatalf("查询 site config 失败: %v", err)
	}
	cfg.DefaultTokenQuotaDay = -1
	cfg.DefaultTokenQuotaRules = `[{"mode":"day","limit":777}]`
	if err := model.DB(ctx).Save(&cfg).Error; err != nil {
		t.Fatalf("保存 site config 失败: %v", err)
	}

	var alice model.User
	if err := model.DB(ctx).Where("username = ?", "alice").First(&alice).Error; err != nil {
		t.Fatalf("查询 alice 失败: %v", err)
	}
	alice.TokenQuotaDay = -1
	alice.TokenQuotaRules = `[{"mode":"day","limit":100}]`
	model.DB(ctx).Save(&alice)

	group := model.UserGroup{Name: "Group No Policy", FullPath: "Group No Policy", Source: model.GroupSourceManual}
	model.DB(ctx).Create(&group)
	model.DB(ctx).Create(&model.GroupClosure{AncestorID: group.ID, DescendantID: group.ID, Depth: 0})

	now := timeInCurrentTokenQuotaDayWindow(t)
	model.DB(ctx).Create(&model.LLMUsageLog{UserID: alice.ID, GroupID: group.ID, TotalTokens: 250, CreatedAt: now})

	req := quotaDataReq(t, "/quota/data?group_id="+strconv.FormatUint(uint64(group.ID), 10), "alice")
	w := httptest.NewRecorder()
	HandleQuotaData(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200, got %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		TokenQuotaRules  []model.TokenQuotaRule `json:"token_quota_rules"`
		TokenQuotaUsages []struct {
			RuleIndex int   `json:"rule_index"`
			Used      int64 `json:"used"`
		} `json:"token_quota_usages"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是有效 JSON: %v", err)
	}
	if len(resp.TokenQuotaRules) != 1 || resp.TokenQuotaRules[0].Limit != 777 {
		t.Fatalf("期望 group 无策略时 fallback site default limit=777, got %+v", resp.TokenQuotaRules)
	}
	if len(resp.TokenQuotaUsages) != 1 || resp.TokenQuotaUsages[0].Used != 250 {
		t.Fatalf("期望只统计该分组 used=250, got %+v", resp.TokenQuotaUsages)
	}
}

func timeInCurrentTokenQuotaDayWindow(t *testing.T) time.Time {
	t.Helper()
	return timeInCurrentTokenQuotaWindow(t, model.TokenQuotaRule{Mode: model.QuotaModeDay, Limit: 1})
}

func timeInCurrentTokenQuotaWindow(t *testing.T, rule model.TokenQuotaRule) time.Time {
	t.Helper()
	from, to, active := rule.QuotaWindow(time.Now())
	if !active {
		t.Fatalf("%s quota rule should be active", rule.Mode)
	}
	if to == nil {
		return from.Add(time.Second)
	}
	ts := from.Add(time.Second)
	if !ts.Before(*to) {
		t.Fatalf("invalid %s quota window: from=%s to=%s", rule.Mode, from, to)
	}
	return ts
}

func TestHandleQuotaData_OrderByTotalTokens(t *testing.T) {
	cleanup := initUsageDataTestEnv(t)
	defer cleanup()

	var alice model.User
	if err := model.DB(context.Background()).Where("username = ?", "alice").First(&alice).Error; err != nil {
		t.Fatalf("查询 alice 失败: %v", err)
	}

	modelA := model.AIModel{ModelID: "model-a", ModelName: "Model A", Provider: "test"}
	modelB := model.AIModel{ModelID: "model-b", ModelName: "Model B", Provider: "test"}
	model.DB(context.Background()).Create(&modelA)
	model.DB(context.Background()).Create(&modelB)

	day := time.Date(2026, 5, 8, 0, 0, 0, 0, time.Local)
	model.DB(context.Background()).Create(&model.DailyUsageSummary{Date: day, UserID: alice.ID, AIModelID: modelA.ID, TotalTokens: 100, RequestCount: 5, PromptTokens: 60, CompletionTokens: 40})
	model.DB(context.Background()).Create(&model.DailyUsageSummary{Date: day, UserID: alice.ID, AIModelID: modelB.ID, TotalTokens: 200, RequestCount: 2, PromptTokens: 120, CompletionTokens: 80})

	req := quotaDataReq(t, "/quota/data?start_date=2026-05-08&end_date=2026-05-08&group_by=model&order_by=total_tokens&order=desc", "alice")
	w := httptest.NewRecorder()
	HandleQuotaData(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200, got %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Rows []struct {
			AIModelID   uint  `json:"ai_model_id"`
			TotalTokens int64 `json:"total_tokens"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是有效 JSON: %v", err)
	}
	if len(resp.Rows) != 2 {
		t.Fatalf("期望 2 行, got %d", len(resp.Rows))
	}
	if resp.Rows[0].AIModelID != modelB.ID || resp.Rows[0].TotalTokens != 200 {
		t.Errorf("应按 total_tokens 降序，第一行 = %+v", resp.Rows[0])
	}
}

func TestHandleQuotaData_OrderByRequestCount(t *testing.T) {
	cleanup := initUsageDataTestEnv(t)
	defer cleanup()

	var alice model.User
	if err := model.DB(context.Background()).Where("username = ?", "alice").First(&alice).Error; err != nil {
		t.Fatalf("查询 alice 失败: %v", err)
	}

	modelA := model.AIModel{ModelID: "model-a", ModelName: "Model A", Provider: "test"}
	modelB := model.AIModel{ModelID: "model-b", ModelName: "Model B", Provider: "test"}
	model.DB(context.Background()).Create(&modelA)
	model.DB(context.Background()).Create(&modelB)

	day := time.Date(2026, 5, 8, 0, 0, 0, 0, time.Local)
	model.DB(context.Background()).Create(&model.DailyUsageSummary{Date: day, UserID: alice.ID, AIModelID: modelA.ID, TotalTokens: 100, RequestCount: 5, PromptTokens: 60, CompletionTokens: 40})
	model.DB(context.Background()).Create(&model.DailyUsageSummary{Date: day, UserID: alice.ID, AIModelID: modelB.ID, TotalTokens: 200, RequestCount: 2, PromptTokens: 120, CompletionTokens: 80})

	req := quotaDataReq(t, "/quota/data?start_date=2026-05-08&end_date=2026-05-08&group_by=model&order_by=request_count&order=desc", "alice")
	w := httptest.NewRecorder()
	HandleQuotaData(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200, got %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Rows []struct {
			AIModelID    uint  `json:"ai_model_id"`
			RequestCount int64 `json:"request_count"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是有效 JSON: %v", err)
	}
	if len(resp.Rows) != 2 {
		t.Fatalf("期望 2 行, got %d", len(resp.Rows))
	}
	if resp.Rows[0].AIModelID != modelA.ID || resp.Rows[0].RequestCount != 5 {
		t.Errorf("应按 request_count 降序，第一行 = %+v", resp.Rows[0])
	}
}

func TestHandleQuotaData_OrderByInvalid(t *testing.T) {
	cleanup := initUsageDataTestEnv(t)
	defer cleanup()

	req := quotaDataReq(t, "/quota/data?order_by=invalid_field&order=desc", "alice")
	w := httptest.NewRecorder()
	HandleQuotaData(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("无效 order_by 应返回 400, got %d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是有效 JSON: %v", err)
	}
	if resp["error"] != "order_by 参数无效，仅支持 total_tokens 或 request_count" {
		t.Errorf("错误消息不匹配, got %q", resp["error"])
	}
}

// TestHandleQuotaData_GroupIDIncludesUngroupedLegacyAgents 覆盖 bug
// 1020422209158742647：用户最初无分组创建 agent，之后被加入分组 X，
// 在分组 X 视图下应能看到这些旧 agent（group_id=0）的用量与配额已用量。
//
// 涉及改动文件：
//   - controller/llm_quota.go: HandleQuotaData 设置 IncludeUserUngrouped=groupID>0，
//     并将 token_quota_usages 切到 userTokenQuotaUsagesCompat。
//   - controller/admin_usage.go: queryUsageData 在 IncludeUserUngrouped+FilterUserID 时
//     使用 group_id IN (X, 0) 过滤。
//   - controller/token_quota_usage.go: userTokenQuotaUsagesCompat 透传 includeUngrouped。
//   - model/llm.go: UserTokenUsageInWindowCompat 兼容分支。
func TestHandleQuotaData_GroupIDIncludesUngroupedLegacyAgents(t *testing.T) {
	cleanup := initUsageDataTestEnv(t)
	defer cleanup()
	ctx := context.Background()

	var alice model.User
	if err := model.DB(ctx).Where("username = ?", "alice").First(&alice).Error; err != nil {
		t.Fatalf("查询 alice 失败: %v", err)
	}
	alice.TokenQuotaDay = -1
	alice.TokenQuotaRules = `[{"mode":"day","limit":1000}]`
	model.DB(ctx).Save(&alice)

	group := model.UserGroup{Name: "Group X", FullPath: "Group X", Source: model.GroupSourceManual}
	model.DB(ctx).Create(&group)
	model.DB(ctx).Create(&model.GroupClosure{AncestorID: group.ID, DescendantID: group.ID, Depth: 0})

	modelA := model.AIModel{ModelID: "model-a", ModelName: "Model A", Provider: "test"}
	model.DB(ctx).Create(&modelA)

	day := time.Date(2026, 5, 8, 0, 0, 0, 0, time.Local)
	// 分组 X 下的新 agent 用量
	model.DB(ctx).Create(&model.DailyUsageSummary{
		Date: day, UserID: alice.ID, InstanceID: 101, GroupID: group.ID, AIModelID: modelA.ID,
		TotalTokens: 200, RequestCount: 2, PromptTokens: 120, CompletionTokens: 80,
	})
	// 同一用户的"无分组旧 agent"用量（group_id=0）—— 兼容分支应把它一并展示
	model.DB(ctx).Create(&model.DailyUsageSummary{
		Date: day, UserID: alice.ID, InstanceID: 102, GroupID: 0, AIModelID: modelA.ID,
		TotalTokens: 100, RequestCount: 5, PromptTokens: 60, CompletionTokens: 40,
	})
	// 其它分组的用量不应被并入
	model.DB(ctx).Create(&model.DailyUsageSummary{
		Date: day, UserID: alice.ID, InstanceID: 103, GroupID: group.ID + 99, AIModelID: modelA.ID,
		TotalTokens: 999, RequestCount: 9, PromptTokens: 600, CompletionTokens: 399,
	})

	// 配额已用量：分组 X 内 250 + 无分组 50 = 300，应通过兼容路径全部计入
	now := timeInCurrentTokenQuotaDayWindow(t)
	model.DB(ctx).Create(&model.LLMUsageLog{UserID: alice.ID, GroupID: group.ID, TotalTokens: 250, CreatedAt: now})
	model.DB(ctx).Create(&model.LLMUsageLog{UserID: alice.ID, GroupID: 0, TotalTokens: 50, CreatedAt: now})
	// 干扰项：其它分组的日志，不应计入
	model.DB(ctx).Create(&model.LLMUsageLog{UserID: alice.ID, GroupID: group.ID + 99, TotalTokens: 9999, CreatedAt: now})

	req := quotaDataReq(t,
		"/quota/data?start_date=2026-05-08&end_date=2026-05-08&group_by=model&group_id="+
			strconv.FormatUint(uint64(group.ID), 10), "alice")
	w := httptest.NewRecorder()
	HandleQuotaData(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200, got %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Rows []struct {
			AIModelID   uint  `json:"ai_model_id"`
			TotalTokens int64 `json:"total_tokens"`
		} `json:"rows"`
		TokenQuotaUsages []struct {
			RuleIndex int   `json:"rule_index"`
			Used      int64 `json:"used"`
		} `json:"token_quota_usages"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是有效 JSON: %v", err)
	}

	// rows: 分组 X(200) + 无分组(100) = 300，应聚合到 model-a 这一行
	if len(resp.Rows) != 1 {
		t.Fatalf("期望 1 行 model 聚合, got %d, body=%s", len(resp.Rows), w.Body.String())
	}
	if resp.Rows[0].AIModelID != modelA.ID || resp.Rows[0].TotalTokens != 300 {
		t.Fatalf("期望 model-a total_tokens=300（200 分组+100 无分组），got %+v", resp.Rows[0])
	}

	// token_quota_usages: 分组 X(250) + 无分组(50) = 300
	if len(resp.TokenQuotaUsages) != 1 {
		t.Fatalf("期望 1 条 quota usage, got %d", len(resp.TokenQuotaUsages))
	}
	if resp.TokenQuotaUsages[0].Used != 300 {
		t.Fatalf("期望 token_quota_usages.used=300（250+50 兼容并入），got %d", resp.TokenQuotaUsages[0].Used)
	}
}

// TestHandleQuotaData_NoGroupIDDoesNotMixOtherGroups 验证不传 group_id
// 时不会启用兼容分支（IncludeUserUngrouped=false），用户全量统计仍为该
// 用户在所有分组的合计，且 token_quota_usages.used 走默认路径（groupID=0
// 表示统计全部）。
func TestHandleQuotaData_NoGroupIDDoesNotMixOtherGroups(t *testing.T) {
	cleanup := initUsageDataTestEnv(t)
	defer cleanup()
	ctx := context.Background()

	var alice model.User
	if err := model.DB(ctx).Where("username = ?", "alice").First(&alice).Error; err != nil {
		t.Fatalf("查询 alice 失败: %v", err)
	}
	alice.TokenQuotaDay = -1
	alice.TokenQuotaRules = `[{"mode":"day","limit":1000}]`
	model.DB(ctx).Save(&alice)

	now := timeInCurrentTokenQuotaDayWindow(t)
	model.DB(ctx).Create(&model.LLMUsageLog{UserID: alice.ID, GroupID: 7, TotalTokens: 80, CreatedAt: now})
	model.DB(ctx).Create(&model.LLMUsageLog{UserID: alice.ID, GroupID: 0, TotalTokens: 20, CreatedAt: now})

	req := quotaDataReq(t, "/quota/data", "alice")
	w := httptest.NewRecorder()
	HandleQuotaData(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200, got %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		TokenQuotaUsages []struct {
			Used int64 `json:"used"`
		} `json:"token_quota_usages"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是有效 JSON: %v", err)
	}
	if len(resp.TokenQuotaUsages) != 1 || resp.TokenQuotaUsages[0].Used != 100 {
		t.Fatalf("不传 group_id 时应统计全部 used=100, got %+v", resp.TokenQuotaUsages)
	}
}

// TestHandleQuotaLogs_GroupIDIncludesUngrouped 验证 HandleQuotaLogs 在
// group_id 过滤下兼容地把 group_id=0 的"无分组旧 agent"日志一并展示。
// 涉及改动：controller/llm_quota.go HandleQuotaLogs 中 group_id IN (X, 0)。
func TestHandleQuotaLogs_GroupIDIncludesUngrouped(t *testing.T) {
	cleanup := initUsageDataTestEnv(t)
	defer cleanup()
	ctx := context.Background()

	var alice, bob model.User
	if err := model.DB(ctx).Where("username = ?", "alice").First(&alice).Error; err != nil {
		t.Fatalf("查询 alice 失败: %v", err)
	}
	if err := model.DB(ctx).Where("username = ?", "bob").First(&bob).Error; err != nil {
		t.Fatalf("查询 bob 失败: %v", err)
	}

	now := time.Now()
	// alice@group=5: 命中
	model.DB(ctx).Create(&model.LLMUsageLog{UserID: alice.ID, GroupID: 5, TotalTokens: 10, CreatedAt: now})
	// alice@group=0: 兼容命中
	model.DB(ctx).Create(&model.LLMUsageLog{UserID: alice.ID, GroupID: 0, TotalTokens: 11, CreatedAt: now})
	// alice@group=9: 不应命中
	model.DB(ctx).Create(&model.LLMUsageLog{UserID: alice.ID, GroupID: 9, TotalTokens: 12, CreatedAt: now})
	// bob@group=0: user_id 隔离，不应命中（HandleQuotaLogs 已强制 user_id=alice）
	model.DB(ctx).Create(&model.LLMUsageLog{UserID: bob.ID, GroupID: 0, TotalTokens: 99, CreatedAt: now})

	today := now.Local().Format("2006-01-02")
	req := quotaDataReq(t, "/quota/logs?start_date="+today+"&end_date="+today+"&group_id=5", "alice")
	w := httptest.NewRecorder()
	HandleQuotaLogs(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200, got %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Total int64 `json:"total"`
		Logs  []struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"logs"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是有效 JSON: %v", err)
	}
	if resp.Total != 2 {
		t.Fatalf("期望命中 2 条（group=5 + group=0），got total=%d", resp.Total)
	}
	got := map[int]bool{}
	for _, lg := range resp.Logs {
		got[lg.TotalTokens] = true
	}
	if !got[10] || !got[11] {
		t.Fatalf("期望同时包含 group=5(10) 和 group=0(11) 两条记录, got %+v", resp.Logs)
	}
	if got[12] || got[99] {
		t.Fatalf("不应包含其它分组或其它用户的记录, got %+v", resp.Logs)
	}
}

// TestUserTokenQuotaUsagesCompat_IncludeUngrouped 直接测 controller 层
// userTokenQuotaUsagesCompat 包装：includeUngrouped=true 时把 group_id=0
// 的旧 agent 用量并入指定分组的"已用量"，false 时维持默认。
func TestUserTokenQuotaUsagesCompat_IncludeUngrouped(t *testing.T) {
	cleanup := initUsageDataTestEnv(t)
	defer cleanup()
	ctx := context.Background()

	var alice model.User
	if err := model.DB(ctx).Where("username = ?", "alice").First(&alice).Error; err != nil {
		t.Fatalf("查询 alice 失败: %v", err)
	}

	now := timeInCurrentTokenQuotaDayWindow(t)
	model.DB(ctx).Create(&model.LLMUsageLog{UserID: alice.ID, GroupID: 5, TotalTokens: 70, CreatedAt: now})
	model.DB(ctx).Create(&model.LLMUsageLog{UserID: alice.ID, GroupID: 0, TotalTokens: 30, CreatedAt: now})
	model.DB(ctx).Create(&model.LLMUsageLog{UserID: alice.ID, GroupID: 9, TotalTokens: 999, CreatedAt: now})

	rules := []model.TokenQuotaRule{{Mode: model.QuotaModeDay, Limit: 1000}}

	// 默认（与原 userTokenQuotaUsages 等价）：仅统计分组 5
	defaults := userTokenQuotaUsages(ctx, alice.ID, 5, rules)
	if len(defaults) != 1 || defaults[0].Used != 70 {
		t.Fatalf("默认路径仅分组 5，期望 used=70, got %+v", defaults)
	}

	// 兼容：分组 5 + 无分组（旧 agent）
	compat := userTokenQuotaUsagesCompat(ctx, alice.ID, 5, rules, true)
	if len(compat) != 1 || compat[0].Used != 100 {
		t.Fatalf("兼容路径应并入 group_id=0，期望 used=100, got %+v", compat)
	}

	// includeUngrouped=false 应等同于默认
	compatOff := userTokenQuotaUsagesCompat(ctx, alice.ID, 5, rules, false)
	if len(compatOff) != 1 || compatOff[0].Used != 70 {
		t.Fatalf("includeUngrouped=false 应与默认一致，got %+v", compatOff)
	}
}
