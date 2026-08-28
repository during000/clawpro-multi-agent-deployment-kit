package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	hcommon "hatchery/common"
	"hatchery/controller/usergroup"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

func initUsageTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.Instance{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	origDB := model.UseDBForTest(db)
	return func() {
		origDB()
	}
}

func TestLoadInstanceMap_EmptyIDs(t *testing.T) {
	cleanup := initUsageTestDB(t)
	defer cleanup()

	m := loadInstanceMap(context.Background(), nil)
	if len(m) != 0 {
		t.Errorf("nil IDs 应返回空 map, got %d entries", len(m))
	}

	m = loadInstanceMap(context.Background(), map[uint]struct{}{})
	if len(m) != 0 {
		t.Errorf("空 IDs 应返回空 map, got %d entries", len(m))
	}
}

func TestLoadInstanceMap_ExistingInstances(t *testing.T) {
	cleanup := initUsageTestDB(t)
	defer cleanup()

	// 创建两个实例
	inst1 := model.Instance{Name: "test-instance-1", InstanceId: "ins-aaa111"}
	inst2 := model.Instance{Name: "test-instance-2", InstanceId: "ins-bbb222"}
	model.DB(context.Background()).Create(&inst1)
	model.DB(context.Background()).Create(&inst2)

	m := loadInstanceMap(context.Background(), map[uint]struct{}{
		inst1.ID:  {},
		inst2.ID:  {},
		uint(999): {}, // 不存在的 ID
	})

	if len(m) != 2 {
		t.Errorf("应返回 2 个实例, got %d", len(m))
	}
	if m[inst1.ID].Name != "test-instance-1" {
		t.Errorf("实例 1 Name 应为 test-instance-1, got %q", m[inst1.ID].Name)
	}
	if m[inst1.ID].InstanceCVMId != "ins-aaa111" {
		t.Errorf("实例 1 InstanceCVMId 应为 ins-aaa111, got %q", m[inst1.ID].InstanceCVMId)
	}
	if m[inst2.ID].Name != "test-instance-2" {
		t.Errorf("实例 2 Name 应为 test-instance-2, got %q", m[inst2.ID].Name)
	}
	if m[inst2.ID].InstanceCVMId != "ins-bbb222" {
		t.Errorf("实例 2 InstanceCVMId 应为 ins-bbb222, got %q", m[inst2.ID].InstanceCVMId)
	}
	// 不存在的 ID 不在 map 中
	if _, ok := m[uint(999)]; ok {
		t.Error("不存在的 ID 不应在 map 中")
	}
}

func TestLoadInstanceMap_SoftDeletedInstances(t *testing.T) {
	cleanup := initUsageTestDB(t)
	defer cleanup()

	// 创建并软删除一个实例
	inst := model.Instance{Name: "deleted-instance", InstanceId: "ins-deleted"}
	model.DB(context.Background()).Create(&inst)
	model.DB(context.Background()).Delete(&inst) // 软删除

	// 使用 Unscoped 查询，软删除的实例也应返回
	m := loadInstanceMap(context.Background(), map[uint]struct{}{inst.ID: {}})

	if len(m) != 1 {
		t.Fatalf("软删除实例应被 Unscoped 查询到, got %d entries", len(m))
	}
	if m[inst.ID].Name != "deleted-instance" {
		t.Errorf("软删除实例 Name 应为 deleted-instance, got %q", m[inst.ID].Name)
	}
	if m[inst.ID].InstanceCVMId != "ins-deleted" {
		t.Errorf("软删除实例 InstanceCVMId 应为 ins-deleted, got %q", m[inst.ID].InstanceCVMId)
	}
}

func TestInstanceInfoStruct(t *testing.T) {
	info := instanceInfo{
		Name:          "my-instance",
		InstanceCVMId: "ins-xyz789",
	}
	if info.Name != "my-instance" {
		t.Errorf("Name 应为 my-instance, got %q", info.Name)
	}
	if info.InstanceCVMId != "ins-xyz789" {
		t.Errorf("InstanceCVMId 应为 ins-xyz789, got %q", info.InstanceCVMId)
	}
}

// ── order_by 参数测试 ────────────────────────────────────────────────

func initUsageDataTestEnv(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.DailyUsageSummary{},
		&model.Instance{},
		&model.AIModel{},
		&model.User{},
		&model.SiteConfig{},
		&model.LLMUsageLog{},
		&model.UserGroup{},
		&model.GroupClosure{},
		&model.UserGroupMember{},
		&model.GroupConfigBinding{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	origDB := model.UseDBForTest(db)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))

	db.Create(&model.SiteConfig{})
	db.Create(&model.User{Username: "alice"})
	db.Create(&model.User{Username: "bob"})

	return func() {
		origDB()
		AdminToken = origToken
		Store = origStore
	}
}

func usageDataReq(path string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req
}

func TestHandleAdminUsageData_OrderByInvalid(t *testing.T) {
	cleanup := initUsageDataTestEnv(t)
	defer cleanup()

	req := usageDataReq("/admin/usage/data?order_by=invalid_field")
	w := httptest.NewRecorder()
	HandleAdminUsageData(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("无效 order_by 应返回 400, got %d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是有效 JSON: %v", err)
	}
	wanted := hcommon.I18nError(i18n.MsgInvalidOrderBy).ErrorMessage(req.Context())
	if resp["error"] != wanted {
		t.Errorf("错误消息不匹配, got %q", resp["error"])
	}
}

func TestHandleAdminUsageData_OrderByDefault(t *testing.T) {
	cleanup := initUsageDataTestEnv(t)
	defer cleanup()

	// 插入两条汇总数据以便测试排序；日期需与默认查询窗口（LocalToday）一致
	today := model.LocalToday()
	model.DB(context.Background()).Create(&model.DailyUsageSummary{Date: today, UserID: 1, InstanceID: 1, AIModelID: 1, TotalTokens: 100, RequestCount: 5, PromptTokens: 60, CompletionTokens: 40})
	model.DB(context.Background()).Create(&model.DailyUsageSummary{Date: today, UserID: 2, InstanceID: 1, AIModelID: 1, TotalTokens: 200, RequestCount: 2, PromptTokens: 120, CompletionTokens: 80})
	model.DB(context.Background()).Create(&model.AIModel{ModelID: "test-model", ModelName: "Test Model", Provider: "test"})

	// 不传 order_by，默认为 total_tokens
	req := usageDataReq("/admin/usage/data?group_by=user&order=desc")
	w := httptest.NewRecorder()
	HandleAdminUsageData(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200, got %d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是有效 JSON: %v", err)
	}
	rows := resp["rows"].([]interface{})
	if len(rows) < 2 {
		t.Fatalf("期望至少 2 行, got %d", len(rows))
	}
	// 按 total_tokens 降序：user2 (200) > user1 (100)
	first := rows[0].(map[string]interface{})
	if first["total_tokens"].(float64) != 200 {
		t.Errorf("降序第一行 total_tokens 应为 200, got %v", first["total_tokens"])
	}
}

func TestHandleAdminUsageData_OrderByRequestCount(t *testing.T) {
	cleanup := initUsageDataTestEnv(t)
	defer cleanup()

	today := model.LocalToday()
	model.DB(context.Background()).Create(&model.DailyUsageSummary{Date: today, UserID: 1, InstanceID: 1, AIModelID: 1, TotalTokens: 100, RequestCount: 5, PromptTokens: 60, CompletionTokens: 40})
	model.DB(context.Background()).Create(&model.DailyUsageSummary{Date: today, UserID: 2, InstanceID: 1, AIModelID: 1, TotalTokens: 200, RequestCount: 2, PromptTokens: 120, CompletionTokens: 80})
	model.DB(context.Background()).Create(&model.AIModel{ModelID: "test-model", ModelName: "Test Model", Provider: "test"})

	// order_by=request_count 降序：user1 (5) > user2 (2)
	req := usageDataReq("/admin/usage/data?group_by=user&order_by=request_count&order=desc")
	w := httptest.NewRecorder()
	HandleAdminUsageData(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200, got %d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是有效 JSON: %v", err)
	}
	rows := resp["rows"].([]interface{})
	if len(rows) < 2 {
		t.Fatalf("期望至少 2 行, got %d", len(rows))
	}
	first := rows[0].(map[string]interface{})
	if first["request_count"].(float64) != 5 {
		t.Errorf("按 request_count 降序第一行 request_count 应为 5, got %v", first["request_count"])
	}
}

func TestHandleAdminUsageData_UserRowsIncludeTokenQuotaUsages(t *testing.T) {
	cleanup := initUsageDataTestEnv(t)
	defer cleanup()

	var alice model.User
	if err := model.DB(context.Background()).Where("username = ?", "alice").First(&alice).Error; err != nil {
		t.Fatalf("查询 alice 失败: %v", err)
	}
	alice.TokenQuotaRules = `[{"mode":"day","limit":1000},{"mode":"custom","limit":5000,"start":1,"refresh":"none"}]`
	alice.TokenQuotaDay = -1
	model.DB(context.Background()).Save(&alice)

	now := timeInCurrentTokenQuotaDayWindow(t)
	today := model.LocalToday()
	model.DB(context.Background()).Create(&model.DailyUsageSummary{Date: today, UserID: alice.ID, TotalTokens: 120, RequestCount: 1})
	model.DB(context.Background()).Create(&model.LLMUsageLog{UserID: alice.ID, TotalTokens: 120, CreatedAt: now})

	req := usageDataReq("/admin/usage/data?group_by=user")
	w := httptest.NewRecorder()
	HandleAdminUsageData(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200, got %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Rows []struct {
			UserID           uint                   `json:"user_id"`
			TokenQuotaRules  []model.TokenQuotaRule `json:"token_quota_rules"`
			TokenQuotaUsages []map[string]any       `json:"token_quota_usages"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是有效 JSON: %v", err)
	}
	for _, row := range resp.Rows {
		if row.UserID != alice.ID {
			continue
		}
		if len(row.TokenQuotaRules) != 2 {
			t.Fatalf("期望返回 2 条 token_quota_rules, got %d", len(row.TokenQuotaRules))
		}
		if len(row.TokenQuotaUsages) != 2 {
			t.Fatalf("期望返回 2 条 token_quota_usages, got %d", len(row.TokenQuotaUsages))
		}
		if row.TokenQuotaUsages[0]["used"].(float64) != 120 {
			t.Fatalf("day rule used = %v, want 120", row.TokenQuotaUsages[0]["used"])
		}
		return
	}
	t.Fatalf("未找到 alice 的用量行")
}

func TestHandleAdminUsageData_UserRowsWithGroupIDUseGroupTokenQuotaRules(t *testing.T) {
	cleanup := initUsageDataTestEnv(t)
	defer cleanup()

	var alice model.User
	if err := model.DB(context.Background()).Where("username = ?", "alice").First(&alice).Error; err != nil {
		t.Fatalf("查询 alice 失败: %v", err)
	}
	alice.TokenQuotaRules = `[{"mode":"day","limit":100}]`
	alice.TokenQuotaDay = -1
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
	today := model.LocalToday()
	model.DB(context.Background()).Create(&model.DailyUsageSummary{Date: today, UserID: alice.ID, GroupID: group.ID, TotalTokens: 120, RequestCount: 1})
	model.DB(context.Background()).Create(&model.LLMUsageLog{UserID: alice.ID, GroupID: group.ID, TotalTokens: 120, CreatedAt: now})
	model.DB(context.Background()).Create(&model.LLMUsageLog{UserID: alice.ID, GroupID: group.ID + 100, TotalTokens: 800, CreatedAt: now})

	req := usageDataReq("/admin/usage/data?group_by=user&group_id=" + strconv.FormatUint(uint64(group.ID), 10))
	w := httptest.NewRecorder()
	HandleAdminUsageData(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200, got %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Rows []struct {
			UserID           uint                   `json:"user_id"`
			TokenQuotaRules  []model.TokenQuotaRule `json:"token_quota_rules"`
			TokenQuotaUsages []map[string]any       `json:"token_quota_usages"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是有效 JSON: %v", err)
	}
	for _, row := range resp.Rows {
		if row.UserID != alice.ID {
			continue
		}
		if len(row.TokenQuotaRules) != 1 || row.TokenQuotaRules[0].Limit != 1000 {
			t.Fatalf("期望使用分组 token_quota_rules limit=1000, got %+v", row.TokenQuotaRules)
		}
		if len(row.TokenQuotaUsages) != 1 || row.TokenQuotaUsages[0]["used"].(float64) != 120 {
			t.Fatalf("期望只统计该分组 used=120, got %+v", row.TokenQuotaUsages)
		}
		return
	}
	t.Fatalf("未找到 alice 的用量行")
}

func TestHandleAdminUsageData_UserRowsIncludeTokenQuotaGroups(t *testing.T) {
	cleanup := initUsageDataTestEnv(t)
	defer cleanup()
	ctx := context.Background()

	var alice model.User
	if err := model.DB(ctx).Where("username = ?", "alice").First(&alice).Error; err != nil {
		t.Fatalf("查询 alice 失败: %v", err)
	}
	alice.TokenQuotaRules = `[{"mode":"day","limit":100}]`
	alice.TokenQuotaDay = -1
	model.DB(ctx).Save(&alice)

	groupA := model.UserGroup{Name: "Group A", FullPath: "Z / Group A", Source: model.GroupSourceManual}
	groupB := model.UserGroup{Name: "Group B", FullPath: "A / Group B", Source: model.GroupSourceManual}
	model.DB(ctx).Create(&groupA)
	model.DB(ctx).Create(&groupB)
	model.DB(ctx).Create(&model.GroupClosure{AncestorID: groupA.ID, DescendantID: groupA.ID, Depth: 0})
	model.DB(ctx).Create(&model.GroupClosure{AncestorID: groupB.ID, DescendantID: groupB.ID, Depth: 0})
	model.DB(ctx).Create(&model.UserGroupMember{UserID: alice.ID, UserGroupID: groupA.ID, Source: model.MemberSourceManual})
	model.DB(ctx).Create(&model.UserGroupMember{UserID: alice.ID, UserGroupID: groupB.ID, Source: model.MemberSourceManual})
	model.DB(ctx).Create(&model.GroupConfigBinding{
		ConfigType: model.ConfigTypePolicy,
		ConfigKey:  usergroup.PolicyKeyTokenQuotaRules,
		GroupID:    groupA.ID,
		ValueJSON:  `{"value":"[{\"mode\":\"day\",\"limit\":1000}]"}`,
	})
	model.DB(ctx).Create(&model.GroupConfigBinding{
		ConfigType: model.ConfigTypePolicy,
		ConfigKey:  usergroup.PolicyKeyTokenQuotaRules,
		GroupID:    groupB.ID,
		ValueJSON:  `{"value":"[{\"mode\":\"day\",\"limit\":2000}]"}`,
	})

	now := timeInCurrentTokenQuotaDayWindow(t)
	today := model.LocalToday()
	model.DB(ctx).Create(&model.DailyUsageSummary{Date: today, UserID: alice.ID, InstanceID: 1, GroupID: groupA.ID, TotalTokens: 120, RequestCount: 1})
	model.DB(ctx).Create(&model.DailyUsageSummary{Date: today, UserID: alice.ID, InstanceID: 2, GroupID: groupB.ID, TotalTokens: 230, RequestCount: 1})
	model.DB(ctx).Create(&model.LLMUsageLog{UserID: alice.ID, GroupID: groupA.ID, TotalTokens: 120, CreatedAt: now})
	model.DB(ctx).Create(&model.LLMUsageLog{UserID: alice.ID, GroupID: groupB.ID, TotalTokens: 230, CreatedAt: now})

	req := usageDataReq("/admin/usage/data?group_by=user")
	w := httptest.NewRecorder()
	HandleAdminUsageData(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200, got %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Rows []struct {
			UserID           uint `json:"user_id"`
			TokenQuotaGroups []struct {
				GroupID          uint                   `json:"group_id"`
				GroupName        string                 `json:"group_name"`
				TokenQuotaRules  []model.TokenQuotaRule `json:"token_quota_rules"`
				TokenQuotaUsages []map[string]any       `json:"token_quota_usages"`
			} `json:"token_quota_groups"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是有效 JSON: %v", err)
	}
	for _, row := range resp.Rows {
		if row.UserID != alice.ID {
			continue
		}
		if len(row.TokenQuotaGroups) != 2 {
			t.Fatalf("期望返回 alice 的 2 个分组 token quota 信息, got %+v", row.TokenQuotaGroups)
		}
		if row.TokenQuotaGroups[0].GroupID != groupA.ID || row.TokenQuotaGroups[1].GroupID != groupB.ID {
			t.Fatalf("token_quota_groups 应按 group_id 升序返回, got %+v", row.TokenQuotaGroups)
		}
		got := map[string]struct {
			Limit int
			Used  float64
		}{}
		for _, group := range row.TokenQuotaGroups {
			if len(group.TokenQuotaRules) != 1 || len(group.TokenQuotaUsages) != 1 {
				t.Fatalf("分组 quota 信息不完整: %+v", group)
			}
			from, to, active := group.TokenQuotaRules[0].QuotaWindow(time.Now())
			if !active {
				t.Fatalf("分组 quota rule should be active: %+v", group.TokenQuotaRules[0])
			}
			if group.TokenQuotaUsages[0]["period_start"].(float64) != float64(from.Unix()) || group.TokenQuotaUsages[0]["period_end"].(float64) != float64(to.Unix()) {
				t.Fatalf("分组 quota period 不符合预期: %+v", group.TokenQuotaUsages[0])
			}
			got[group.GroupName] = struct {
				Limit int
				Used  float64
			}{Limit: group.TokenQuotaRules[0].Limit, Used: group.TokenQuotaUsages[0]["used"].(float64)}
		}
		if got["Group A"].Limit != 1000 || got["Group A"].Used != 120 {
			t.Fatalf("Group A quota 信息不符合预期: %+v", got["Group A"])
		}
		if got["Group B"].Limit != 2000 || got["Group B"].Used != 230 {
			t.Fatalf("Group B quota 信息不符合预期: %+v", got["Group B"])
		}
		return
	}
	t.Fatalf("未找到 alice 的用量行")
}

func TestHandleAdminUsageData_UserRowsWithGroupIDNoPolicyUseSiteDefaultTokenQuotaRules(t *testing.T) {
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
	alice.TokenQuotaRules = `[{"mode":"day","limit":100}]`
	alice.TokenQuotaDay = -1
	model.DB(ctx).Save(&alice)

	group := model.UserGroup{Name: "Group No Policy", FullPath: "Group No Policy", Source: model.GroupSourceManual}
	model.DB(ctx).Create(&group)
	model.DB(ctx).Create(&model.GroupClosure{AncestorID: group.ID, DescendantID: group.ID, Depth: 0})

	now := timeInCurrentTokenQuotaDayWindow(t)
	today := model.LocalToday()
	model.DB(ctx).Create(&model.DailyUsageSummary{Date: today, UserID: alice.ID, GroupID: group.ID, TotalTokens: 120, RequestCount: 1})
	model.DB(ctx).Create(&model.LLMUsageLog{UserID: alice.ID, GroupID: group.ID, TotalTokens: 120, CreatedAt: now})

	req := usageDataReq("/admin/usage/data?group_by=user&group_id=" + strconv.FormatUint(uint64(group.ID), 10))
	w := httptest.NewRecorder()
	HandleAdminUsageData(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200, got %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Rows []struct {
			UserID           uint                   `json:"user_id"`
			TokenQuotaRules  []model.TokenQuotaRule `json:"token_quota_rules"`
			TokenQuotaUsages []map[string]any       `json:"token_quota_usages"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是有效 JSON: %v", err)
	}
	for _, row := range resp.Rows {
		if row.UserID != alice.ID {
			continue
		}
		if len(row.TokenQuotaRules) != 1 || row.TokenQuotaRules[0].Limit != 777 {
			t.Fatalf("期望 group 无策略时 fallback site default limit=777, got %+v", row.TokenQuotaRules)
		}
		if len(row.TokenQuotaUsages) != 1 || row.TokenQuotaUsages[0]["used"].(float64) != 120 {
			t.Fatalf("期望只统计该分组 used=120, got %+v", row.TokenQuotaUsages)
		}
		return
	}
	t.Fatalf("未找到 alice 的用量行")
}

func TestHandleAdminUsageData_GroupRowsIncludeExplicitGlobalTokenQuotaUsages(t *testing.T) {
	cleanup := initUsageDataTestEnv(t)
	defer cleanup()

	group := model.UserGroup{Name: "Root", FullPath: "Root", Source: model.GroupSourceManual}
	model.DB(context.Background()).Create(&group)
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: group.ID, DescendantID: group.ID, Depth: 0})
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: model.ConfigTypePolicy,
		ConfigKey:  usergroup.PolicyKeyGlobalTokenQuotaRules,
		GroupID:    group.ID,
		ValueJSON:  `{"value":"[{\"mode\":\"day\",\"limit\":1000}]"}`,
	})

	now := timeInCurrentTokenQuotaDayWindow(t)
	today := model.LocalToday()
	model.DB(context.Background()).Create(&model.DailyUsageSummary{Date: today, GroupID: group.ID, TotalTokens: 300, RequestCount: 1})
	model.DB(context.Background()).Create(&model.LLMUsageLog{GroupID: group.ID, TotalTokens: 300, CreatedAt: now})

	req := usageDataReq("/admin/usage/data?group_by=group")
	w := httptest.NewRecorder()
	HandleAdminUsageData(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200, got %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Rows []struct {
			GroupID                uint                   `json:"group_id"`
			GlobalTokenQuotaRules  []model.TokenQuotaRule `json:"global_token_quota_rules"`
			GlobalTokenQuotaUsages []map[string]any       `json:"global_token_quota_usages"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是有效 JSON: %v", err)
	}
	for _, row := range resp.Rows {
		if row.GroupID != group.ID {
			continue
		}
		if len(row.GlobalTokenQuotaRules) != 1 {
			t.Fatalf("期望返回 1 条 global_token_quota_rules, got %d", len(row.GlobalTokenQuotaRules))
		}
		if len(row.GlobalTokenQuotaUsages) != 1 {
			t.Fatalf("期望返回 1 条 global_token_quota_usages, got %d", len(row.GlobalTokenQuotaUsages))
		}
		if row.GlobalTokenQuotaUsages[0]["used"].(float64) != 300 {
			t.Fatalf("group global used = %v, want 300", row.GlobalTokenQuotaUsages[0]["used"])
		}
		from, to, active := row.GlobalTokenQuotaRules[0].QuotaWindow(time.Now())
		if !active {
			t.Fatalf("global token quota rule should be active")
		}
		if row.GlobalTokenQuotaUsages[0]["period_start"].(float64) != float64(from.Unix()) || row.GlobalTokenQuotaUsages[0]["period_end"].(float64) != float64(to.Unix()) {
			t.Fatalf("group global period = %+v, want %d-%d", row.GlobalTokenQuotaUsages[0], from.Unix(), to.Unix())
		}
		return
	}
	t.Fatalf("未找到分组用量行")
}

func TestHandleAdminUsageData_GroupRowsIncludeCustomYearlyGlobalQuotaUsage(t *testing.T) {
	cleanup := initUsageDataTestEnv(t)
	defer cleanup()

	group := model.UserGroup{Name: "Root", FullPath: "Root", Source: model.GroupSourceManual}
	model.DB(context.Background()).Create(&group)
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: group.ID, DescendantID: group.ID, Depth: 0})

	start := time.Now().Add(-400 * 24 * time.Hour).Unix()
	rulesJSON := `[{"mode":"custom","limit":-1,"start":` + strconv.FormatInt(start, 10) + `,"refresh":"yearly"}]`
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: model.ConfigTypePolicy,
		ConfigKey:  usergroup.PolicyKeyGlobalTokenQuotaRules,
		GroupID:    group.ID,
		ValueJSON:  `{"value":` + strconv.Quote(rulesJSON) + `}`,
	})

	rule := model.TokenQuotaRule{Mode: model.QuotaModeCustom, Limit: -1, Start: &start, Refresh: model.QuotaRefreshYearly}
	inWindow := timeInCurrentTokenQuotaWindow(t, rule)
	from, _, _ := rule.QuotaWindow(time.Now())
	today := model.LocalToday()
	model.DB(context.Background()).Create(&model.DailyUsageSummary{Date: today, GroupID: group.ID, TotalTokens: 900, RequestCount: 1})
	model.DB(context.Background()).Create(&model.LLMUsageLog{GroupID: group.ID, TotalTokens: 900, CreatedAt: inWindow})
	model.DB(context.Background()).Create(&model.LLMUsageLog{GroupID: group.ID, TotalTokens: 700, CreatedAt: from.Add(-time.Second)})

	req := usageDataReq("/admin/usage/data?group_by=group")
	w := httptest.NewRecorder()
	HandleAdminUsageData(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200, got %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Rows []struct {
			GroupID                uint                   `json:"group_id"`
			GlobalTokenQuotaRules  []model.TokenQuotaRule `json:"global_token_quota_rules"`
			GlobalTokenQuotaUsages []map[string]any       `json:"global_token_quota_usages"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是有效 JSON: %v", err)
	}
	for _, row := range resp.Rows {
		if row.GroupID != group.ID {
			continue
		}
		if len(row.GlobalTokenQuotaRules) != 1 || row.GlobalTokenQuotaRules[0].Refresh != model.QuotaRefreshYearly {
			t.Fatalf("期望返回 custom yearly global_token_quota_rules, got %+v", row.GlobalTokenQuotaRules)
		}
		if row.GlobalTokenQuotaRules[0].Limit != -1 {
			t.Fatalf("期望 custom yearly global_token_quota_rules limit=-1, got %+v", row.GlobalTokenQuotaRules)
		}
		if len(row.GlobalTokenQuotaUsages) != 1 || row.GlobalTokenQuotaUsages[0]["used"].(float64) != 900 {
			t.Fatalf("期望 custom yearly group global used=900, got %+v", row.GlobalTokenQuotaUsages)
		}
		return
	}
	t.Fatalf("未找到分组用量行")
}

func TestHandleAdminUsageData_GroupRowsIncludeFallbackGlobalTokenQuotaUsages(t *testing.T) {
	cleanup := initUsageDataTestEnv(t)
	defer cleanup()
	ctx := context.Background()

	group := model.UserGroup{Name: "Root", FullPath: "Root", Source: model.GroupSourceManual}
	model.DB(ctx).Create(&group)
	model.DB(ctx).Create(&model.GroupClosure{AncestorID: group.ID, DescendantID: group.ID, Depth: 0})
	model.DB(ctx).Model(&model.SiteConfig{}).Where("id > 0").Updates(map[string]any{
		"global_token_quota_day":   -1,
		"global_token_quota_rules": `[{"mode":"day","limit":1000}]`,
	})

	now := timeInCurrentTokenQuotaDayWindow(t)
	today := model.LocalToday()
	model.DB(ctx).Create(&model.DailyUsageSummary{Date: today, GroupID: group.ID, TotalTokens: 300, RequestCount: 1})
	model.DB(ctx).Create(&model.LLMUsageLog{GroupID: group.ID, TotalTokens: 300, CreatedAt: now})
	model.DB(ctx).Create(&model.LLMUsageLog{GroupID: group.ID + 100, TotalTokens: 700, CreatedAt: now})

	req := usageDataReq("/admin/usage/data?group_by=group")
	w := httptest.NewRecorder()
	HandleAdminUsageData(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200, got %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Rows []struct {
			GroupID                uint                   `json:"group_id"`
			GlobalTokenQuotaRules  []model.TokenQuotaRule `json:"global_token_quota_rules"`
			GlobalTokenQuotaUsages []map[string]any       `json:"global_token_quota_usages"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是有效 JSON: %v", err)
	}
	for _, row := range resp.Rows {
		if row.GroupID != group.ID {
			continue
		}
		if len(row.GlobalTokenQuotaRules) != 1 || row.GlobalTokenQuotaRules[0].Limit != 1000 {
			t.Fatalf("期望无显式分组规则时返回站点 global_token_quota_rules, got %+v", row.GlobalTokenQuotaRules)
		}
		if len(row.GlobalTokenQuotaUsages) != 1 || row.GlobalTokenQuotaUsages[0]["used"].(float64) != 300 {
			t.Fatalf("期望无显式分组规则时按该分组统计 used=300, got %+v", row.GlobalTokenQuotaUsages)
		}
		return
	}
	t.Fatalf("未找到分组用量行")
}
