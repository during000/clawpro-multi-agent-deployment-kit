package controller

import (
	"context"
	"encoding/json"
	"hatchery/controller/usergroup"
	"hatchery/model"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ── HandleChannelVisibility 测试 ────────────────────────────────────────────

func TestCoverageChannelVisibility_Success(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	ch := model.AIChannel{Name: "微信通道", ChannelID: "wechat-001", VisibilityType: "all"}
	model.DB(context.Background()).Create(&ch)

	var group model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&group)

	body, _ := json.Marshal(map[string]interface{}{
		"channel_id":      "wechat-001",
		"visibility_type": "group",
		"group_ids":       []uint{group.ID},
	})

	w := httptest.NewRecorder()
	HandleChannelVisibility(w, adminTreeReq(http.MethodPost, "/admin/channels/visibility", body))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	// 验证更新
	var updated model.AIChannel
	model.DB(context.Background()).First(&updated, ch.ID)
	if updated.VisibilityType != "group" {
		t.Errorf("期望 visibility_type=group，实际=%s", updated.VisibilityType)
	}
}

func TestCoverageChannelVisibility_ByID(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	ch := model.AIChannel{Name: "QQ通道", ChannelID: "qq-001", VisibilityType: "all"}
	model.DB(context.Background()).Create(&ch)

	body, _ := json.Marshal(map[string]interface{}{
		"channel_id":      ch.ID,
		"visibility_type": "all",
		"group_ids":       []uint{},
	})

	w := httptest.NewRecorder()
	HandleChannelVisibility(w, adminTreeReq(http.MethodPost, "/admin/channels/visibility", body))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestCoverageChannelVisibility_MethodNotAllowed(t *testing.T) {
	setupTreeTestDB(t)

	w := httptest.NewRecorder()
	HandleChannelVisibility(w, adminTreeReq(http.MethodGet, "/admin/channels/visibility", nil))

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望 405，实际=%d", w.Code)
	}
}

func TestCoverageChannelVisibility_InvalidVisibilityType(t *testing.T) {
	setupTreeTestDB(t)

	body, _ := json.Marshal(map[string]interface{}{
		"channel_id":      "ch-001",
		"visibility_type": "invalid",
	})

	w := httptest.NewRecorder()
	HandleChannelVisibility(w, adminTreeReq(http.MethodPost, "/admin/channels/visibility", body))

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w.Code)
	}
}

func TestCoverageChannelVisibility_GroupWithoutGroupIDs(t *testing.T) {
	setupTreeTestDB(t)

	body, _ := json.Marshal(map[string]interface{}{
		"channel_id":      "ch-001",
		"visibility_type": "group",
		"group_ids":       []uint{},
	})

	w := httptest.NewRecorder()
	HandleChannelVisibility(w, adminTreeReq(http.MethodPost, "/admin/channels/visibility", body))

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w.Code)
	}
}

func TestCoverageChannelVisibility_ChannelNotFound(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	var group model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&group)

	body, _ := json.Marshal(map[string]interface{}{
		"channel_id":      "nonexist-channel",
		"visibility_type": "group",
		"group_ids":       []uint{group.ID},
	})

	w := httptest.NewRecorder()
	HandleChannelVisibility(w, adminTreeReq(http.MethodPost, "/admin/channels/visibility", body))

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("期望 422，实际=%d", w.Code)
	}
}

func TestCoverageChannelVisibility_InvalidGroupIDs(t *testing.T) {
	setupTreeTestDB(t)

	ch := model.AIChannel{Name: "通道", ChannelID: "ch-ok", VisibilityType: "all"}
	model.DB(context.Background()).Create(&ch)

	body, _ := json.Marshal(map[string]interface{}{
		"channel_id":      "ch-ok",
		"visibility_type": "group",
		"group_ids":       []uint{99999},
	})

	w := httptest.NewRecorder()
	HandleChannelVisibility(w, adminTreeReq(http.MethodPost, "/admin/channels/visibility", body))

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w.Code)
	}
}

// ── HandleMCPVisibility 测试 ─────────────────────────────────────────────────

func TestCoverageMCPVisibility_Success(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	mcp := model.McpServer{Name: "天气MCP", VisibilityType: "all"}
	model.DB(context.Background()).Create(&mcp)

	body, _ := json.Marshal(map[string]interface{}{
		"mcp_id":          mcp.ID,
		"visibility_type": "all",
	})

	w := httptest.NewRecorder()
	HandleMCPVisibility(w, adminTreeReq(http.MethodPost, "/admin/mcp/visibility", body))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestCoverageMCPVisibility_NotFound(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	var group model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&group)

	body, _ := json.Marshal(map[string]interface{}{
		"mcp_id":          uint(99999),
		"visibility_type": "group",
		"group_ids":       []uint{group.ID},
	})

	w := httptest.NewRecorder()
	HandleMCPVisibility(w, adminTreeReq(http.MethodPost, "/admin/mcp/visibility", body))

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("期望 422，实际=%d", w.Code)
	}
}

// ── HandleImageTypeVisibility 测试 ────────────────────────────────────────────

func TestCoverageImageTypeVisibility_Success(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	model.DB(context.Background()).Create(&model.AIImage{AgentType: "openclaw", Enabled: true})

	body, _ := json.Marshal(map[string]interface{}{
		"agent_type":      "openclaw",
		"visibility_type": "all",
	})

	w := httptest.NewRecorder()
	HandleImageTypeVisibility(w, adminTreeReq(http.MethodPost, "/admin/images/type-visibility", body))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestCoverageImageTypeVisibility_InvalidAgentType(t *testing.T) {
	setupTreeTestDB(t)

	body, _ := json.Marshal(map[string]interface{}{
		"agent_type":      "invalid_type_xyz",
		"visibility_type": "all",
	})

	w := httptest.NewRecorder()
	HandleImageTypeVisibility(w, adminTreeReq(http.MethodPost, "/admin/images/type-visibility", body))

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w.Code)
	}
}

func TestCoverageImageTypeVisibility_InvalidVisibilityType(t *testing.T) {
	setupTreeTestDB(t)

	body, _ := json.Marshal(map[string]interface{}{
		"agent_type":      "openclaw",
		"visibility_type": "invalid",
	})

	w := httptest.NewRecorder()
	HandleImageTypeVisibility(w, adminTreeReq(http.MethodPost, "/admin/images/type-visibility", body))

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w.Code)
	}
}

func TestCoverageImageTypeVisibility_GroupWithoutIDs(t *testing.T) {
	setupTreeTestDB(t)

	body, _ := json.Marshal(map[string]interface{}{
		"agent_type":      "openclaw",
		"visibility_type": "group",
		"group_ids":       []uint{},
	})

	w := httptest.NewRecorder()
	HandleImageTypeVisibility(w, adminTreeReq(http.MethodPost, "/admin/images/type-visibility", body))

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w.Code)
	}
}

func TestCoverageImageTypeVisibility_MethodNotAllowed(t *testing.T) {
	setupTreeTestDB(t)

	w := httptest.NewRecorder()
	HandleImageTypeVisibility(w, adminTreeReq(http.MethodGet, "/admin/images/type-visibility", nil))

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望 405，实际=%d", w.Code)
	}
}

// ── HandleGroupConfigGroups 测试 ──────────────────────────────────────────────

func TestCoverageGroupConfigGroups_Success(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	var group model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&group)

	// 创建 channel 绑定
	ch := model.AIChannel{Name: "微信", VisibilityType: "group"}
	model.DB(context.Background()).Create(&ch)
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: usergroup.ConfigTypeChannel,
		ConfigKey:  itoa(ch.ID),
		GroupID:    group.ID,
	})

	queries, _ := json.Marshal([]configQuery{{ConfigType: "channel", ConfigKey: itoa(ch.ID)}})
	w := httptest.NewRecorder()
	HandleGroupConfigGroups(w, adminTreeReq(http.MethodGet,
		"/admin/group-config/groups?queries="+string(queries), nil))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	results := resp["results"].([]interface{})
	if len(results) != 1 {
		t.Errorf("期望 1 个结果，实际=%d", len(results))
	}
}

func TestCoverageGroupConfigGroups_PolicyWithValue(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	var group model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&group)

	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: "policy",
		ConfigKey:  "token_quota_day",
		GroupID:    group.ID,
		ValueJSON:  `{"value": 10000}`,
	})

	queries, _ := json.Marshal([]configQuery{{ConfigType: "policy", ConfigKey: "token_quota_day"}})
	w := httptest.NewRecorder()
	HandleGroupConfigGroups(w, adminTreeReq(http.MethodGet,
		"/admin/group-config/groups?queries="+string(queries), nil))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestCoverageGroupConfigGroups_MissingQueries(t *testing.T) {
	setupTreeTestDB(t)

	w := httptest.NewRecorder()
	HandleGroupConfigGroups(w, adminTreeReq(http.MethodGet,
		"/admin/group-config/groups", nil))

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w.Code)
	}
}

func TestCoverageGroupConfigGroups_InvalidJSON(t *testing.T) {
	setupTreeTestDB(t)

	w := httptest.NewRecorder()
	HandleGroupConfigGroups(w, adminTreeReq(http.MethodGet,
		"/admin/group-config/groups?queries=not-json", nil))

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w.Code)
	}
}

func TestCoverageGroupConfigGroups_EmptyArray(t *testing.T) {
	setupTreeTestDB(t)

	w := httptest.NewRecorder()
	HandleGroupConfigGroups(w, adminTreeReq(http.MethodGet,
		"/admin/group-config/groups?queries=[]", nil))

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w.Code)
	}
}

func TestCoverageGroupConfigGroups_InvalidConfigType(t *testing.T) {
	setupTreeTestDB(t)

	queries, _ := json.Marshal([]configQuery{{ConfigType: "invalid_type", ConfigKey: "1"}})
	w := httptest.NewRecorder()
	HandleGroupConfigGroups(w, adminTreeReq(http.MethodGet,
		"/admin/group-config/groups?queries="+string(queries), nil))

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w.Code)
	}
}

func TestCoverageGroupConfigGroups_MethodNotAllowed(t *testing.T) {
	setupTreeTestDB(t)

	w := httptest.NewRecorder()
	HandleGroupConfigGroups(w, adminTreeReq(http.MethodPost,
		"/admin/group-config/groups", nil))

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望 405，实际=%d", w.Code)
	}
}

// ── HandleSetGroupPolicy 测试 ─────────────────────────────────────────────────

func TestCoverageSetGroupPolicy_Success(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	var group model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&group)

	body, _ := json.Marshal(map[string]interface{}{
		"group_id":   group.ID,
		"config_key": "token_quota_day",
		"value_json": `{"value": 5000}`,
	})

	w := httptest.NewRecorder()
	HandleSetGroupPolicy(w, adminTreeReq(http.MethodPost, "/admin/group-config/policy", body))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestCoverageSetGroupPolicy_MissingGroupID(t *testing.T) {
	setupTreeTestDB(t)

	body, _ := json.Marshal(map[string]interface{}{
		"group_id":   0,
		"config_key": "token_quota_day",
		"value_json": `{"value": 5000}`,
	})

	w := httptest.NewRecorder()
	HandleSetGroupPolicy(w, adminTreeReq(http.MethodPost, "/admin/group-config/policy", body))

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w.Code)
	}
}

func TestCoverageSetGroupPolicy_InvalidConfigKey(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	var group model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&group)

	body, _ := json.Marshal(map[string]interface{}{
		"group_id":   group.ID,
		"config_key": "invalid_key",
		"value_json": `{"value": 5000}`,
	})

	w := httptest.NewRecorder()
	HandleSetGroupPolicy(w, adminTreeReq(http.MethodPost, "/admin/group-config/policy", body))

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w.Code)
	}
}

func TestCoverageSetGroupPolicy_MissingValueJSON(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	var group model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&group)

	body, _ := json.Marshal(map[string]interface{}{
		"group_id":   group.ID,
		"config_key": "token_quota_day",
		"value_json": "",
	})

	w := httptest.NewRecorder()
	HandleSetGroupPolicy(w, adminTreeReq(http.MethodPost, "/admin/group-config/policy", body))

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w.Code)
	}
}

func TestCoverageSetGroupPolicy_InvalidValueJSON(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	var group model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&group)

	body, _ := json.Marshal(map[string]interface{}{
		"group_id":   group.ID,
		"config_key": "token_quota_day",
		"value_json": "not-valid-json",
	})

	w := httptest.NewRecorder()
	HandleSetGroupPolicy(w, adminTreeReq(http.MethodPost, "/admin/group-config/policy", body))

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w.Code)
	}
}

func TestCoverageSetGroupPolicy_GroupNotFound(t *testing.T) {
	setupTreeTestDB(t)

	body, _ := json.Marshal(map[string]interface{}{
		"group_id":   uint(99999),
		"config_key": "token_quota_day",
		"value_json": `{"value": 5000}`,
	})

	w := httptest.NewRecorder()
	HandleSetGroupPolicy(w, adminTreeReq(http.MethodPost, "/admin/group-config/policy", body))

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("期望 422，实际=%d", w.Code)
	}
}

func TestCoverageSetGroupPolicy_MethodNotAllowed(t *testing.T) {
	setupTreeTestDB(t)

	w := httptest.NewRecorder()
	HandleSetGroupPolicy(w, adminTreeReq(http.MethodGet, "/admin/group-config/policy", nil))

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望 405，实际=%d", w.Code)
	}
}

// ── HandleDeleteGroupPolicy 测试 ──────────────────────────────────────────────

func TestCoverageDeleteGroupPolicy_Success(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	var group model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&group)

	// 先设置策略
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: "policy",
		ConfigKey:  "token_quota_day",
		GroupID:    group.ID,
		ValueJSON:  `{"value": 5000}`,
	})

	body, _ := json.Marshal(map[string]interface{}{
		"group_id":   group.ID,
		"config_key": "token_quota_day",
	})

	w := httptest.NewRecorder()
	HandleDeleteGroupPolicy(w, adminTreeReq(http.MethodPost, "/admin/group-config/policy/delete", body))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestCoverageDeleteGroupPolicy_MissingGroupID(t *testing.T) {
	setupTreeTestDB(t)

	body, _ := json.Marshal(map[string]interface{}{
		"group_id":   0,
		"config_key": "token_quota_day",
	})

	w := httptest.NewRecorder()
	HandleDeleteGroupPolicy(w, adminTreeReq(http.MethodPost, "/admin/group-config/policy/delete", body))

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w.Code)
	}
}

func TestCoverageDeleteGroupPolicy_InvalidKey(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	var group model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&group)

	body, _ := json.Marshal(map[string]interface{}{
		"group_id":   group.ID,
		"config_key": "invalid_key",
	})

	w := httptest.NewRecorder()
	HandleDeleteGroupPolicy(w, adminTreeReq(http.MethodPost, "/admin/group-config/policy/delete", body))

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w.Code)
	}
}

func TestCoverageDeleteGroupPolicy_BindingNotExist(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	var group model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&group)

	body, _ := json.Marshal(map[string]interface{}{
		"group_id":   group.ID,
		"config_key": "token_quota_day",
	})

	w := httptest.NewRecorder()
	HandleDeleteGroupPolicy(w, adminTreeReq(http.MethodPost, "/admin/group-config/policy/delete", body))

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("期望 422，实际=%d", w.Code)
	}
}

func TestCoverageDeleteGroupPolicy_GroupNotFound(t *testing.T) {
	setupTreeTestDB(t)

	body, _ := json.Marshal(map[string]interface{}{
		"group_id":   uint(99999),
		"config_key": "token_quota_day",
	})

	w := httptest.NewRecorder()
	HandleDeleteGroupPolicy(w, adminTreeReq(http.MethodPost, "/admin/group-config/policy/delete", body))

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("期望 422，实际=%d", w.Code)
	}
}

func TestCoverageDeleteGroupPolicy_MethodNotAllowed(t *testing.T) {
	setupTreeTestDB(t)

	w := httptest.NewRecorder()
	HandleDeleteGroupPolicy(w, adminTreeReq(http.MethodGet, "/admin/group-config/policy/delete", nil))

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望 405，实际=%d", w.Code)
	}
}

// ── 配额 day↔rules 兼容清理测试 ──────────────────────────────────────────────

// countPolicyBindings 统计某组某 policy key 的 binding 数量
func countPolicyBindings(t *testing.T, groupID uint, configKey string) int64 {
	t.Helper()
	var count int64
	model.DB(context.Background()).
		Model(&model.GroupConfigBinding{}).
		Where("config_type = ? AND config_key = ? AND group_id = ?", "policy", configKey, groupID).
		Count(&count)
	return count
}

// 用例 1：设置 token_quota_rules，组原有 token_quota_day binding → rules 写入，day 被清理
func TestCoverageSetGroupPolicy_RulesClearsLegacyDay(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	var group model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&group)

	// 预置 legacy day binding
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: "policy", ConfigKey: "token_quota_day", GroupID: group.ID, ValueJSON: `{"value": 5000}`,
	})

	body, _ := json.Marshal(map[string]interface{}{
		"group_id":   group.ID,
		"config_key": "token_quota_rules",
		"value_json": `{"value": "[{\"mode\":\"day\",\"limit\":8000}]"}`,
	})

	w := httptest.NewRecorder()
	HandleSetGroupPolicy(w, adminTreeReq(http.MethodPost, "/admin/group-config/policy", body))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	if got := countPolicyBindings(t, group.ID, "token_quota_rules"); got != 1 {
		t.Errorf("期望 rules binding 数=1，实际=%d", got)
	}
	if got := countPolicyBindings(t, group.ID, "token_quota_day"); got != 0 {
		t.Errorf("期望 day binding 已清理=0，实际=%d", got)
	}
}

// 用例 2：设置 global_token_quota_rules，组原有 global_token_quota_day binding → rules 写入，day 被清理
func TestCoverageSetGroupPolicy_GlobalRulesClearsLegacyDay(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	var group model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&group)

	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: "policy", ConfigKey: "global_token_quota_day", GroupID: group.ID, ValueJSON: `{"value": 100000}`,
	})

	body, _ := json.Marshal(map[string]interface{}{
		"group_id":   group.ID,
		"config_key": "global_token_quota_rules",
		"value_json": `{"value": "[{\"mode\":\"day\",\"limit\":200000}]"}`,
	})

	w := httptest.NewRecorder()
	HandleSetGroupPolicy(w, adminTreeReq(http.MethodPost, "/admin/group-config/policy", body))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	if got := countPolicyBindings(t, group.ID, "global_token_quota_rules"); got != 1 {
		t.Errorf("期望 global rules binding 数=1，实际=%d", got)
	}
	if got := countPolicyBindings(t, group.ID, "global_token_quota_day"); got != 0 {
		t.Errorf("期望 global day binding 已清理=0，实际=%d", got)
	}
}

// 用例 3：设置 token_quota_rules，组无任何配额 binding → rules 写入，幂等删 day 不报错
func TestCoverageSetGroupPolicy_RulesNoLegacyDay(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	var group model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&group)

	body, _ := json.Marshal(map[string]interface{}{
		"group_id":   group.ID,
		"config_key": "token_quota_rules",
		"value_json": `{"value": "[{\"mode\":\"day\",\"limit\":8000}]"}`,
	})

	w := httptest.NewRecorder()
	HandleSetGroupPolicy(w, adminTreeReq(http.MethodPost, "/admin/group-config/policy", body))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	if got := countPolicyBindings(t, group.ID, "token_quota_rules"); got != 1 {
		t.Errorf("期望 rules binding 数=1，实际=%d", got)
	}
}

// 用例 4：删除 token_quota_rules，组仅有 token_quota_day binding → fallback 删 day
func TestCoverageDeleteGroupPolicy_FallbackToDay(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	var group model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&group)

	// 组仅有 legacy day binding，无 rules binding
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: "policy", ConfigKey: "token_quota_day", GroupID: group.ID, ValueJSON: `{"value": 5000}`,
	})

	body, _ := json.Marshal(map[string]interface{}{
		"group_id":   group.ID,
		"config_key": "token_quota_rules",
	})

	w := httptest.NewRecorder()
	HandleDeleteGroupPolicy(w, adminTreeReq(http.MethodPost, "/admin/group-config/policy/delete", body))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	if got := countPolicyBindings(t, group.ID, "token_quota_day"); got != 0 {
		t.Errorf("期望 day binding 已被 fallback 删除=0，实际=%d", got)
	}
	if got := countPolicyBindings(t, group.ID, "token_quota_rules"); got != 0 {
		t.Errorf("期望 rules binding 数=0，实际=%d", got)
	}
}

// 用例 5：删除 global_token_quota_rules，组仅有 global_token_quota_day binding → fallback 删 day
func TestCoverageDeleteGroupPolicy_FallbackToGlobalDay(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	var group model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&group)

	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: "policy", ConfigKey: "global_token_quota_day", GroupID: group.ID, ValueJSON: `{"value": 100000}`,
	})

	body, _ := json.Marshal(map[string]interface{}{
		"group_id":   group.ID,
		"config_key": "global_token_quota_rules",
	})

	w := httptest.NewRecorder()
	HandleDeleteGroupPolicy(w, adminTreeReq(http.MethodPost, "/admin/group-config/policy/delete", body))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	if got := countPolicyBindings(t, group.ID, "global_token_quota_day"); got != 0 {
		t.Errorf("期望 global day binding 已被 fallback 删除=0，实际=%d", got)
	}
}

// 用例 6：删除 token_quota_day，组仅有 token_quota_rules binding → 反向 fallback 删 rules
func TestCoverageDeleteGroupPolicy_FallbackToRules(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	var group model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&group)

	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: "policy", ConfigKey: "token_quota_rules", GroupID: group.ID, ValueJSON: `{"value": "[{\"mode\":\"day\",\"limit\":8000}]"}`,
	})

	body, _ := json.Marshal(map[string]interface{}{
		"group_id":   group.ID,
		"config_key": "token_quota_day",
	})

	w := httptest.NewRecorder()
	HandleDeleteGroupPolicy(w, adminTreeReq(http.MethodPost, "/admin/group-config/policy/delete", body))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	if got := countPolicyBindings(t, group.ID, "token_quota_rules"); got != 0 {
		t.Errorf("期望 rules binding 已被 fallback 删除=0，实际=%d", got)
	}
}

// 用例 7：删除 token_quota_rules，组既无 rules 也无 day → 422
func TestCoverageDeleteGroupPolicy_BothAbsent(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	var group model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&group)

	body, _ := json.Marshal(map[string]interface{}{
		"group_id":   group.ID,
		"config_key": "token_quota_rules",
	})

	w := httptest.NewRecorder()
	HandleDeleteGroupPolicy(w, adminTreeReq(http.MethodPost, "/admin/group-config/policy/delete", body))
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("期望 422，实际=%d", w.Code)
	}
}

// 用例 8：删除非配额 key（instance_quota）不存在时 → 422，不触发 fallback
func TestCoverageDeleteGroupPolicy_NonQuotaKeyNoFallback(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	var group model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&group)

	body, _ := json.Marshal(map[string]interface{}{
		"group_id":   group.ID,
		"config_key": "instance_quota",
	})

	w := httptest.NewRecorder()
	HandleDeleteGroupPolicy(w, adminTreeReq(http.MethodPost, "/admin/group-config/policy/delete", body))
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("期望 422，实际=%d", w.Code)
	}
}

// 用例 9：删除 token_quota_rules，组同时有 rules 和 day（legacy 脏数据）→ 仅删 rules，day 保留
func TestCoverageDeleteGroupPolicy_BothPresentOnlyDeletesTarget(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	var group model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&group)

	// legacy 脏数据：两个 binding 共存
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: "policy", ConfigKey: "token_quota_rules", GroupID: group.ID, ValueJSON: `{"value": "[{\"mode\":\"day\",\"limit\":8000}]"}`,
	})
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: "policy", ConfigKey: "token_quota_day", GroupID: group.ID, ValueJSON: `{"value": 5000}`,
	})

	body, _ := json.Marshal(map[string]interface{}{
		"group_id":   group.ID,
		"config_key": "token_quota_rules",
	})

	w := httptest.NewRecorder()
	HandleDeleteGroupPolicy(w, adminTreeReq(http.MethodPost, "/admin/group-config/policy/delete", body))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	if got := countPolicyBindings(t, group.ID, "token_quota_rules"); got != 0 {
		t.Errorf("期望 rules binding 已删除=0，实际=%d", got)
	}
	if got := countPolicyBindings(t, group.ID, "token_quota_day"); got != 1 {
		t.Errorf("期望 day binding 保留=1（目标存在时只删目标），实际=%d", got)
	}
}

// ── batchGetGroupNames 测试 ───────────────────────────────────────────────────

func TestCoverageBatchGetGroupNames_Empty(t *testing.T) {
	setupTreeTestDB(t)

	result := batchGetGroupNames(context.Background(), nil)
	if len(result) != 0 {
		t.Errorf("期望空 map，实际=%d", len(result))
	}
}

func TestCoverageBatchGetGroupNames_WithData(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	var groups []model.UserGroup
	model.DB(context.Background()).Find(&groups)

	ids := make([]uint, len(groups))
	for i, g := range groups {
		ids[i] = g.ID
	}

	result := batchGetGroupNames(context.Background(), ids)
	if len(result) != len(groups) {
		t.Errorf("期望 %d 个名称，实际=%d", len(groups), len(result))
	}
}

// ── HandleGroupConfigGroups token_quota_rules 兼容测试 ───────────────────────

func TestCoverageGroupConfigGroups_RulesCompat_DayFromRules(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	var group model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&group)

	// 只有 token_quota_rules binding（模拟新代码设置 day 后转为 rules）
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: "policy",
		ConfigKey:  "token_quota_rules",
		GroupID:    group.ID,
		ValueJSON:  `{"value":"[{\"mode\":\"day\",\"limit\":300000}]"}`,
	})

	queries, _ := json.Marshal([]configQuery{{ConfigType: "policy", ConfigKey: "token_quota_day"}})
	w := httptest.NewRecorder()
	HandleGroupConfigGroups(w, adminTreeReq(http.MethodGet,
		"/admin/group-config/groups?queries="+string(queries), nil))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Results []struct {
			Groups []struct {
				GroupID uint                   `json:"group_id"`
				Value   map[string]interface{} `json:"value"`
			} `json:"groups"`
		} `json:"results"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	if len(resp.Results) != 1 || len(resp.Results[0].Groups) != 1 {
		t.Fatalf("期望 1 个组，实际=%+v", resp.Results)
	}
	g := resp.Results[0].Groups[0]
	if g.GroupID != group.ID {
		t.Errorf("期望 group_id=%d，实际=%d", group.ID, g.GroupID)
	}
	if v, ok := g.Value["value"].(float64); !ok || int(v) != 300000 {
		t.Errorf("期望 value=300000，实际=%v", g.Value)
	}
}

func TestCoverageGroupConfigGroups_RulesCompat_SkipExistingDay(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	var groups []model.UserGroup
	model.DB(context.Background()).Find(&groups)
	if len(groups) < 2 {
		t.Fatal("需要至少 2 个组")
	}
	g1, g2 := groups[0], groups[1]

	// g1: 仍有旧 token_quota_day binding，不应从 rules 反推
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: "policy",
		ConfigKey:  "token_quota_day",
		GroupID:    g1.ID,
		ValueJSON:  `{"value":100000}`,
	})
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: "policy",
		ConfigKey:  "token_quota_rules",
		GroupID:    g1.ID,
		ValueJSON:  `{"value":"[{\"mode\":\"day\",\"limit\":999999}]"}`,
	})
	// g2: 只有 token_quota_rules（从 rules 反推）
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: "policy",
		ConfigKey:  "token_quota_rules",
		GroupID:    g2.ID,
		ValueJSON:  `{"value":"[{\"mode\":\"day\",\"limit\":200000}]"}`,
	})

	queries, _ := json.Marshal([]configQuery{{ConfigType: "policy", ConfigKey: "token_quota_day"}})
	w := httptest.NewRecorder()
	HandleGroupConfigGroups(w, adminTreeReq(http.MethodGet,
		"/admin/group-config/groups?queries="+string(queries), nil))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Results []struct {
			Groups []struct {
				GroupID uint                   `json:"group_id"`
				Value   map[string]interface{} `json:"value"`
			} `json:"groups"`
		} `json:"results"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	if len(resp.Results) != 1 {
		t.Fatalf("期望 1 个 result，实际=%d", len(resp.Results))
	}
	resultGroups := resp.Results[0].Groups
	if len(resultGroups) != 2 {
		t.Fatalf("期望 2 个组，实际=%d", len(resultGroups))
	}

	for _, rg := range resultGroups {
		if rg.GroupID == g1.ID {
			// 旧 binding 优先，不被 rules 的 999999 覆盖
			if v, ok := rg.Value["value"].(float64); !ok || int(v) != 100000 {
				t.Errorf("g1 期望 value=100000（旧 binding），实际=%v", rg.Value)
			}
		}
		if rg.GroupID == g2.ID {
			if v, ok := rg.Value["value"].(float64); !ok || int(v) != 200000 {
				t.Errorf("g2 期望 value=200000（从 rules 反推），实际=%v", rg.Value)
			}
		}
	}
}

func TestCoverageGroupConfigGroups_RulesCompat_EmptyRules(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	var group model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&group)

	// token_quota_rules = "[]" 表示无限制
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: "policy",
		ConfigKey:  "token_quota_rules",
		GroupID:    group.ID,
		ValueJSON:  `{"value":"[]"}`,
	})

	queries, _ := json.Marshal([]configQuery{{ConfigType: "policy", ConfigKey: "token_quota_day"}})
	w := httptest.NewRecorder()
	HandleGroupConfigGroups(w, adminTreeReq(http.MethodGet,
		"/admin/group-config/groups?queries="+string(queries), nil))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Results []struct {
			Groups []struct {
				GroupID uint                   `json:"group_id"`
				Value   map[string]interface{} `json:"value"`
			} `json:"groups"`
		} `json:"results"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	if len(resp.Results) != 1 || len(resp.Results[0].Groups) != 1 {
		t.Fatalf("期望 1 个组，实际=%+v", resp.Results)
	}
	g := resp.Results[0].Groups[0]
	// rules=[] → TokenQuotaDayFromRules 返回 -1
	if v, ok := g.Value["value"].(float64); !ok || int(v) != -1 {
		t.Errorf("期望 value=-1（无限制），实际=%v", g.Value)
	}
}

func TestCoverageGroupConfigGroups_RulesCompat_RulesFromLegacyDay(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	var group model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&group)

	// 只有旧 token_quota_day binding；查询新 token_quota_rules 时应反推，避免平台策略页漏显旧策略。
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: "policy",
		ConfigKey:  "token_quota_day",
		GroupID:    group.ID,
		ValueJSON:  `{"value":200000}`,
	})

	queries, _ := json.Marshal([]configQuery{{ConfigType: "policy", ConfigKey: "token_quota_rules"}})
	w := httptest.NewRecorder()
	HandleGroupConfigGroups(w, adminTreeReq(http.MethodGet,
		"/admin/group-config/groups?queries="+string(queries), nil))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Results []struct {
			Groups []struct {
				GroupID uint                   `json:"group_id"`
				Value   map[string]interface{} `json:"value"`
			} `json:"groups"`
		} `json:"results"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	if len(resp.Results) != 1 || len(resp.Results[0].Groups) != 1 {
		t.Fatalf("期望 1 个组，实际=%+v", resp.Results)
	}
	g := resp.Results[0].Groups[0]
	if g.GroupID != group.ID {
		t.Fatalf("期望 group_id=%d，实际=%d", group.ID, g.GroupID)
	}
	rulesRaw, ok := g.Value["value"].(string)
	if !ok {
		t.Fatalf("期望 value 为 rules JSON 字符串，实际=%v", g.Value)
	}
	rules, ok := model.ParseTokenQuotaRules(rulesRaw)
	if !ok || len(rules) != 1 || rules[0].Mode != model.QuotaModeDay || rules[0].Limit != 200000 {
		t.Fatalf("期望从旧 day 反推 day limit=200000，实际 raw=%s rules=%+v", rulesRaw, rules)
	}
}

func TestCoverageGroupConfigGroups_RulesCompat_RulesPreferExplicitRulesOverLegacyDay(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	var group model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&group)

	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: "policy",
		ConfigKey:  "token_quota_day",
		GroupID:    group.ID,
		ValueJSON:  `{"value":200000}`,
	})
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: "policy",
		ConfigKey:  "token_quota_rules",
		GroupID:    group.ID,
		ValueJSON:  `{"value":"[{\"mode\":\"month\",\"limit\":900000}]"}`,
	})

	queries, _ := json.Marshal([]configQuery{{ConfigType: "policy", ConfigKey: "token_quota_rules"}})
	w := httptest.NewRecorder()
	HandleGroupConfigGroups(w, adminTreeReq(http.MethodGet,
		"/admin/group-config/groups?queries="+string(queries), nil))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Results []struct {
			Groups []struct {
				Value map[string]interface{} `json:"value"`
			} `json:"groups"`
		} `json:"results"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	if len(resp.Results) != 1 || len(resp.Results[0].Groups) != 1 {
		t.Fatalf("期望 1 个组，实际=%+v", resp.Results)
	}
	rulesRaw, ok := resp.Results[0].Groups[0].Value["value"].(string)
	if !ok {
		t.Fatalf("期望 value 为 rules JSON 字符串，实际=%v", resp.Results[0].Groups[0].Value)
	}
	rules, ok := model.ParseTokenQuotaRules(rulesRaw)
	if !ok || len(rules) != 1 || rules[0].Mode != model.QuotaModeMonth || rules[0].Limit != 900000 {
		t.Fatalf("期望显式 rules 优先于旧 day，实际 raw=%s rules=%+v", rulesRaw, rules)
	}
}

func TestCoverageGroupConfigGroups_GlobalRulesCompat_RulesFromLegacyDayMonthAndSkipExisting(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	ctx := context.Background()
	if err := model.DB(ctx).Model(&model.SiteConfig{}).Where("id > 0").
		Update("global_token_quota_period", model.GlobalTokenQuotaPeriodMonth).Error; err != nil {
		t.Fatalf("更新全局配额周期失败: %v", err)
	}

	var groups []model.UserGroup
	model.DB(ctx).Find(&groups)
	if len(groups) < 2 {
		t.Fatal("需要至少 2 个组")
	}
	g1, g2 := groups[0], groups[1]

	model.DB(ctx).Create(&model.GroupConfigBinding{
		ConfigType: usergroup.ConfigTypePolicy,
		ConfigKey:  usergroup.PolicyKeyGlobalTokenQuotaRules,
		GroupID:    g1.ID,
		ValueJSON:  `{"value":"[{\"mode\":\"month\",\"limit\":900000}]"}`,
	})
	model.DB(ctx).Create(&model.GroupConfigBinding{
		ConfigType: usergroup.ConfigTypePolicy,
		ConfigKey:  usergroup.PolicyKeyGlobalTokenQuotaDay,
		GroupID:    g1.ID,
		ValueJSON:  `{"value":111111}`,
	})
	model.DB(ctx).Create(&model.GroupConfigBinding{
		ConfigType: usergroup.ConfigTypePolicy,
		ConfigKey:  usergroup.PolicyKeyGlobalTokenQuotaDay,
		GroupID:    g2.ID,
		ValueJSON:  `{"value":123456}`,
	})

	queries, _ := json.Marshal([]configQuery{{ConfigType: usergroup.ConfigTypePolicy, ConfigKey: usergroup.PolicyKeyGlobalTokenQuotaRules}})
	w := httptest.NewRecorder()
	HandleGroupConfigGroups(w, adminTreeReq(http.MethodGet,
		"/admin/group-config/groups?queries="+string(queries), nil))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Results []struct {
			Groups []struct {
				GroupID uint                   `json:"group_id"`
				Value   map[string]interface{} `json:"value"`
			} `json:"groups"`
		} `json:"results"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if len(resp.Results) != 1 || len(resp.Results[0].Groups) != 2 {
		t.Fatalf("期望 2 个组，实际=%+v", resp.Results)
	}

	for _, rg := range resp.Results[0].Groups {
		rulesRaw, ok := rg.Value["value"].(string)
		if !ok {
			t.Fatalf("group %d 期望 value 为 rules JSON 字符串，实际=%v", rg.GroupID, rg.Value)
		}
		rules, ok := model.ParseTokenQuotaRules(rulesRaw)
		if !ok || len(rules) != 1 || rules[0].Mode != model.QuotaModeMonth {
			t.Fatalf("group %d 期望 month rules，实际 raw=%s rules=%+v", rg.GroupID, rulesRaw, rules)
		}
		switch rg.GroupID {
		case g1.ID:
			if rules[0].Limit != 900000 {
				t.Fatalf("g1 应保持显式 global_token_quota_rules，实际=%+v", rules)
			}
		case g2.ID:
			if rules[0].Limit != 123456 {
				t.Fatalf("g2 应从旧 global_token_quota_day 按月反推，实际=%+v", rules)
			}
		default:
			t.Fatalf("返回了未知 group_id=%d", rg.GroupID)
		}
	}
}
