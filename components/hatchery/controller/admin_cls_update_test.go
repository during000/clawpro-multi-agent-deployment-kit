package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// setupCLSUpdateTestEnv 初始化 CLS update handler 测试所需的内存数据库和全局状态。
func setupCLSUpdateTestEnv(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("数据库初始化失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.SiteConfig{},
		&model.GroupConfigBinding{},
		&model.GroupClosure{},
		&model.UserGroupMember{},
		&model.UserGroup{},
		&model.Instance{},
		&model.User{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	restore := model.UseDBForTest(db)

	origToken := AdminToken
	AdminToken = "test-admin-token"

	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))

	t.Cleanup(func() {
		restore()
		AdminToken = origToken
		Store = origStore
	})
}

// clsUpdateAdminReq 创建带 admin Bearer Token 的 JSON 请求。
func clsUpdateAdminReq(method, path, body string) (*http.Request, *httptest.ResponseRecorder) {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req, httptest.NewRecorder()
}

// parseCLSUpdateResp 解析 JSON 响应。
func parseCLSUpdateResp(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("响应 JSON 解析失败: %v, body=%s", err, w.Body.String())
	}
	return result
}

// ─── HandleAdminGetCLSUpdateStats 测试 ─────────────────────────────────────

// TestHandleAdminGetCLSUpdateStats_Unauthorized 验证未授权请求返回 401/403。
func TestHandleAdminGetCLSUpdateStats_Unauthorized(t *testing.T) {
	setupCLSUpdateTestEnv(t)

	req := httptest.NewRequest("GET", "/admin/cls/update", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	HandleAdminGetCLSUpdateStats(w, req)

	if w.Code != http.StatusForbidden && w.Code != http.StatusUnauthorized {
		t.Errorf("未授权请求期望 401/403，实际 %d", w.Code)
	}
}

// TestHandleAdminGetCLSUpdateStats_NoInstances 验证无已安装实例时返回空列表。
func TestHandleAdminGetCLSUpdateStats_NoInstances(t *testing.T) {
	setupCLSUpdateTestEnv(t)

	req, w := clsUpdateAdminReq("GET", "/admin/cls/update", "")
	HandleAdminGetCLSUpdateStats(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d: %s", w.Code, w.Body.String())
	}
	result := parseCLSUpdateResp(t, w)
	if result["ok"] != true {
		t.Errorf("期望 ok=true，实际 %v", result["ok"])
	}
	if int(result["v1_count"].(float64)) != 0 {
		t.Errorf("期望 v1_count=0，实际 %v", result["v1_count"])
	}
	if int(result["v2_count"].(float64)) != 0 {
		t.Errorf("期望 v2_count=0，实际 %v", result["v2_count"])
	}
	instances, ok := result["instances"].([]interface{})
	if !ok {
		t.Fatalf("instances 应为数组，实际 %T", result["instances"])
	}
	if len(instances) != 0 {
		t.Errorf("期望 instances 为空，实际 %d 条", len(instances))
	}
}

// TestHandleAdminGetCLSUpdateStats_WithInstances 验证有已安装实例时返回正确的版本统计。
func TestHandleAdminGetCLSUpdateStats_WithInstances(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	// v1.0 实例
	model.DB(ctx).Create(&model.Instance{
		InstanceId:       "ins-v1-001",
		Name:             "v1实例1",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionV1,
		LastCVMState:     "RUNNING",
	})
	// v2.0 实例
	model.DB(ctx).Create(&model.Instance{
		InstanceId:       "ins-v2-001",
		Name:             "v2实例1",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionV2,
		LastCVMState:     "RUNNING",
	})
	// updating 实例（归入 v1_count）
	model.DB(ctx).Create(&model.Instance{
		InstanceId:       "ins-updating-001",
		Name:             "升级中实例1",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionUpdating,
		LastCVMState:     "RUNNING",
	})
	// 版本为空的实例（归一化为 1.0，归入 v1_count）
	model.DB(ctx).Create(&model.Instance{
		InstanceId:     "ins-empty-001",
		Name:           "空版本实例1",
		CLSAgentStatus: model.CLSAgentInstalled,
		LastCVMState:   "RUNNING",
	})
	// 未安装实例（不应出现在结果中）
	model.DB(ctx).Create(&model.Instance{
		InstanceId:     "ins-not-installed",
		Name:           "未安装实例",
		CLSAgentStatus: model.CLSAgentNotInstalled,
	})
	// doctor 节点（不应出现在结果中）
	model.DB(ctx).Create(&model.Instance{
		InstanceId:     "ins-doctor",
		Name:           "医生节点",
		CLSAgentStatus: model.CLSAgentInstalled,
		IsDoctorNode:   true,
	})

	req, w := clsUpdateAdminReq("GET", "/admin/cls/update", "")
	HandleAdminGetCLSUpdateStats(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d: %s", w.Code, w.Body.String())
	}
	result := parseCLSUpdateResp(t, w)

	// v1_count = 3（v1.0 + updating + 空版本）
	if int(result["v1_count"].(float64)) != 3 {
		t.Errorf("期望 v1_count=3，实际 %v", result["v1_count"])
	}
	// v2_count = 1
	if int(result["v2_count"].(float64)) != 1 {
		t.Errorf("期望 v2_count=1，实际 %v", result["v2_count"])
	}
	// instances 共 4 条（排除未安装和 doctor 节点）
	instances, ok := result["instances"].([]interface{})
	if !ok {
		t.Fatalf("instances 应为数组，实际 %T", result["instances"])
	}
	if len(instances) != 4 {
		t.Errorf("期望 4 条实例，实际 %d", len(instances))
	}
}

// TestHandleAdminGetCLSUpdateStats_NonRunningInstanceExcluded 验证非 RUNNING 状态（SHUTDOWN、TERMINATING 等）
// 的实例不出现在统计结果中，只有 last_cvm_state=RUNNING 的实例才被统计。
func TestHandleAdminGetCLSUpdateStats_NonRunningInstanceExcluded(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	// 正常运行中的 v1.0 实例（应出现在结果中）
	model.DB(ctx).Create(&model.Instance{
		InstanceId:       "ins-running-v1",
		Name:             "运行中v1实例",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionV1,
		LastCVMState:     "RUNNING",
	})
	// 待回收的 v1.0 实例（应被过滤）
	model.DB(ctx).Create(&model.Instance{
		InstanceId:       "ins-shutdown-v1",
		Name:             "待回收v1实例",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionV1,
		LastCVMState:     "SHUTDOWN",
	})
	// 销毁中的 v2.0 实例（应被过滤）
	model.DB(ctx).Create(&model.Instance{
		InstanceId:       "ins-terminating-v2",
		Name:             "销毁中v2实例",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionV2,
		LastCVMState:     "TERMINATING",
	})

	req, w := clsUpdateAdminReq("GET", "/admin/cls/update", "")
	HandleAdminGetCLSUpdateStats(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d: %s", w.Code, w.Body.String())
	}
	result := parseCLSUpdateResp(t, w)

	// 只有运行中的 v1.0 实例，v1_count=1，v2_count=0
	if int(result["v1_count"].(float64)) != 1 {
		t.Errorf("期望 v1_count=1（非 RUNNING 实例应被过滤），实际 %v", result["v1_count"])
	}
	if int(result["v2_count"].(float64)) != 0 {
		t.Errorf("期望 v2_count=0（非 RUNNING 实例应被过滤），实际 %v", result["v2_count"])
	}
	instances, ok := result["instances"].([]interface{})
	if !ok {
		t.Fatalf("instances 应为数组，实际 %T", result["instances"])
	}
	if len(instances) != 1 {
		t.Errorf("期望 1 条实例（非 RUNNING 实例应被过滤），实际 %d", len(instances))
	}
	inst := instances[0].(map[string]interface{})
	if inst["instance_id"] != "ins-running-v1" {
		t.Errorf("期望 ins-running-v1，实际 %v", inst["instance_id"])
	}
}

// TestHandleAdminGetCLSUpdateStats_EmptyVersionNormalized 验证版本为空时归一化为 "1.0"。
func TestHandleAdminGetCLSUpdateStats_EmptyVersionNormalized(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	model.DB(ctx).Create(&model.Instance{
		InstanceId:     "ins-empty-ver",
		Name:           "空版本",
		CLSAgentStatus: model.CLSAgentInstalled,
		LastCVMState:   "RUNNING",
	})
	// GORM default:'1.0' 会写入默认值，需手动清空以触发 v=="" 归一化路径（行 560-561）
	model.DB(ctx).Exec("UPDATE instances SET cls_plugin_version = '' WHERE instance_id = 'ins-empty-ver'")

	req, w := clsUpdateAdminReq("GET", "/admin/cls/update", "")
	HandleAdminGetCLSUpdateStats(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d: %s", w.Code, w.Body.String())
	}
	result := parseCLSUpdateResp(t, w)
	instances := result["instances"].([]interface{})
	if len(instances) != 1 {
		t.Fatalf("期望 1 条实例，实际 %d", len(instances))
	}
	inst := instances[0].(map[string]interface{})
	if inst["cls_plugin_version"] != "1.0" {
		t.Errorf("空版本应归一化为 1.0，实际 %v", inst["cls_plugin_version"])
	}
}

// ─── HandleAdminUpdateCLSPlugin 测试 ─────────────────────────────────────

// TestHandleAdminUpdateCLSPlugin_Unauthorized 验证未授权请求返回 401/403。
func TestHandleAdminUpdateCLSPlugin_Unauthorized(t *testing.T) {
	setupCLSUpdateTestEnv(t)

	req := httptest.NewRequest("POST", "/admin/cls/update", strings.NewReader(`{"scope_type":"all"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	HandleAdminUpdateCLSPlugin(w, req)

	if w.Code != http.StatusForbidden && w.Code != http.StatusUnauthorized {
		t.Errorf("未授权请求期望 401/403，实际 %d", w.Code)
	}
}

// TestHandleAdminUpdateCLSPlugin_CLSNotEnabled 验证 CLS 未开启时返回 400。
func TestHandleAdminUpdateCLSPlugin_CLSNotEnabled(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	model.DB(ctx).Create(&model.SiteConfig{CLSEnabled: 0})

	req, w := clsUpdateAdminReq("POST", "/admin/cls/update", `{"scope_type":"all"}`)
	HandleAdminUpdateCLSPlugin(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("CLS 未开启时期望 400，实际 %d: %s", w.Code, w.Body.String())
	}
}

// TestHandleAdminUpdateCLSPlugin_InvalidJSON 验证无效 JSON 请求体返回 400。
func TestHandleAdminUpdateCLSPlugin_InvalidJSON(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	model.DB(ctx).Create(&model.SiteConfig{CLSEnabled: 1})

	req, w := clsUpdateAdminReq("POST", "/admin/cls/update", `{invalid json}`)
	HandleAdminUpdateCLSPlugin(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("无效 JSON 期望 400，实际 %d: %s", w.Code, w.Body.String())
	}
}

// TestHandleAdminUpdateCLSPlugin_NoTargetInstances 验证无待升级实例时返回空结果。
func TestHandleAdminUpdateCLSPlugin_NoTargetInstances(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	model.DB(ctx).Create(&model.SiteConfig{CLSEnabled: 1})

	// 所有实例已是 v2.0，无需升级
	model.DB(ctx).Create(&model.Instance{
		InstanceId:       "ins-v2-only",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionV2,
	})

	req, w := clsUpdateAdminReq("POST", "/admin/cls/update", `{"scope_type":"all"}`)
	HandleAdminUpdateCLSPlugin(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d: %s", w.Code, w.Body.String())
	}
	result := parseCLSUpdateResp(t, w)
	if result["ok"] != true {
		t.Errorf("期望 ok=true，实际 %v", result["ok"])
	}
	if int(result["total"].(float64)) != 0 {
		t.Errorf("期望 total=0，实际 %v", result["total"])
	}
	if result["message"] == nil {
		t.Errorf("无待升级实例时期望有 message 字段")
	}
}

// TestHandleAdminUpdateCLSPlugin_GroupModeIgnoresGroupIDs 验证 group 模式下 group_ids 被忽略，升级全部机器。
func TestHandleAdminUpdateCLSPlugin_GroupModeIgnoresGroupIDs(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	model.DB(ctx).Create(&model.SiteConfig{CLSEnabled: 1})

	// 无待升级实例，验证 group 模式携带 group_ids 不报错
	req, w := clsUpdateAdminReq("POST", "/admin/cls/update", `{"scope_type":"group","group_ids":[1,2,3]}`)
	HandleAdminUpdateCLSPlugin(w, req)

	// group_ids 被忽略，不应返回 400
	if w.Code == http.StatusBadRequest {
		t.Errorf("group 模式下 group_ids 应被忽略，不应返回 400，实际 body: %s", w.Body.String())
	}
}

// TestHandleAdminUpdateCLSPlugin_UpdatingInstancesFiltered 验证 updating 状态的实例不被重复处理。
func TestHandleAdminUpdateCLSPlugin_UpdatingInstancesFiltered(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	model.DB(ctx).Create(&model.SiteConfig{CLSEnabled: 1})

	// updating 状态的实例（应被过滤）
	model.DB(ctx).Create(&model.Instance{
		InstanceId:       "ins-updating",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionUpdating,
	})

	req, w := clsUpdateAdminReq("POST", "/admin/cls/update", `{"scope_type":"all"}`)
	HandleAdminUpdateCLSPlugin(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d: %s", w.Code, w.Body.String())
	}
	result := parseCLSUpdateResp(t, w)
	// updating 实例被过滤，total=0
	if int(result["total"].(float64)) != 0 {
		t.Errorf("updating 实例应被过滤，期望 total=0，实际 %v", result["total"])
	}
}

// TestHandleAdminUpdateCLSPlugin_NonRunningInstancesFiltered 验证非 RUNNING 状态（SHUTDOWN、TERMINATING 等）
// 的实例在 update 执行路径中被过滤，不计入 total。
func TestHandleAdminUpdateCLSPlugin_NonRunningInstancesFiltered(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	model.DB(ctx).Create(&model.SiteConfig{CLSEnabled: 1})

	// 待回收的 v1.0 实例（应被过滤）
	model.DB(ctx).Create(&model.Instance{
		InstanceId:       "ins-shutdown-v1",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionV1,
		LastCVMState:     "SHUTDOWN",
	})
	// 销毁中的 v1.0 实例（应被过滤）
	model.DB(ctx).Create(&model.Instance{
		InstanceId:       "ins-terminating-v1",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionV1,
		LastCVMState:     "TERMINATING",
	})

	req, w := clsUpdateAdminReq("POST", "/admin/cls/update", `{"scope_type":"all"}`)
	HandleAdminUpdateCLSPlugin(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d: %s", w.Code, w.Body.String())
	}
	result := parseCLSUpdateResp(t, w)
	// SHUTDOWN/TERMINATING 实例均被过滤，total=0
	if int(result["total"].(float64)) != 0 {
		t.Errorf("非 RUNNING 实例应被过滤，期望 total=0，实际 %v", result["total"])
	}
}

// ─── queryCLSUpdateTargetInstances 测试 ─────────────────────────────────────

// TestQueryCLSUpdateTargetInstances_AllMode 验证全量模式下查询所有待升级实例。
func TestQueryCLSUpdateTargetInstances_AllMode(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	// v1.0 实例（应被查到）
	model.DB(ctx).Create(&model.Instance{
		InstanceId:       "ins-q-v1",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionV1,
		LastCVMState:     "RUNNING",
	})
	// v2.0 实例（应被过滤）
	model.DB(ctx).Create(&model.Instance{
		InstanceId:       "ins-q-v2",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionV2,
	})
	// updating 实例（应被过滤）
	model.DB(ctx).Create(&model.Instance{
		InstanceId:       "ins-q-updating",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionUpdating,
	})
	// 未安装实例（应被过滤）
	model.DB(ctx).Create(&model.Instance{
		InstanceId:     "ins-q-not-installed",
		CLSAgentStatus: model.CLSAgentNotInstalled,
	})
	// doctor 节点（应被过滤）
	model.DB(ctx).Create(&model.Instance{
		InstanceId:     "ins-q-doctor",
		CLSAgentStatus: model.CLSAgentInstalled,
		IsDoctorNode:   true,
	})
	// 空 instance_id（应被过滤）
	model.DB(ctx).Create(&model.Instance{
		InstanceId:     "",
		CLSAgentStatus: model.CLSAgentInstalled,
	})
	// 待回收实例（应被过滤）
	model.DB(ctx).Create(&model.Instance{
		InstanceId:       "ins-q-shutdown",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionV1,
		LastCVMState:     "SHUTDOWN",
	})
	// 销毁中实例（应被过滤）
	model.DB(ctx).Create(&model.Instance{
		InstanceId:       "ins-q-terminating",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionV1,
		LastCVMState:     "TERMINATING",
	})

	req := clsUpdateRequest{ScopeType: "all"}
	instances, err := queryCLSUpdateTargetInstances(ctx, req)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(instances) != 1 {
		t.Errorf("期望 1 条待升级实例，实际 %d", len(instances))
	}
	if instances[0].InstanceId != "ins-q-v1" {
		t.Errorf("期望 ins-q-v1，实际 %s", instances[0].InstanceId)
	}
}

// TestQueryCLSUpdateTargetInstances_GroupModeIgnoresGroupIDs 验证 group 模式下 group_ids 被忽略，
// 查询结果与全量模式一致。
func TestQueryCLSUpdateTargetInstances_GroupModeIgnoresGroupIDs(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	// 两个不同分组的实例，group 模式下均应被查到
	model.DB(ctx).Create(&model.Instance{
		InstanceId:       "ins-q-group-a",
		GroupID:          1,
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionV1,
		LastCVMState:     "RUNNING",
	})
	model.DB(ctx).Create(&model.Instance{
		InstanceId:       "ins-q-group-b",
		GroupID:          2,
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionV1,
		LastCVMState:     "RUNNING",
	})

	// 传入 group_ids 也不影响结果
	req := clsUpdateRequest{ScopeType: "group", GroupIDs: []uint{1}}
	instances, err := queryCLSUpdateTargetInstances(ctx, req)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	// group_ids 被忽略，两个实例均应被查到
	if len(instances) != 2 {
		t.Errorf("group_ids 应被忽略，期望查到 2 条实例，实际 %d", len(instances))
	}
}

// ─── parseTraceCheckOutput 测试 ─────────────────────────────────────

// TestParseTraceCheckOutput_ValidJSON 验证正常 JSON 输出解析成功。
func TestParseTraceCheckOutput_ValidJSON(t *testing.T) {
	output := `some log line
another log line
{"trace_enabled":true,"trace_topic_id":"topic-123","configured":true,"reason":"ok"}`

	result, err := parseTraceCheckOutput(output)
	if err != nil {
		t.Fatalf("期望解析成功，实际 %v", err)
	}
	if !result.Configured {
		t.Errorf("期望 configured=true，实际 %v", result.Configured)
	}
	if !result.TraceEnabled {
		t.Errorf("期望 trace_enabled=true，实际 %v", result.TraceEnabled)
	}
	if result.TraceTopicID != "topic-123" {
		t.Errorf("期望 trace_topic_id=topic-123，实际 %v", result.TraceTopicID)
	}
}

// TestParseTraceCheckOutput_EmptyOutput 验证空输出返回错误。
func TestParseTraceCheckOutput_EmptyOutput(t *testing.T) {
	_, err := parseTraceCheckOutput("")
	if err == nil {
		t.Error("空输出期望返回错误")
	}
}

// TestParseTraceCheckOutput_OnlyWhitespace 验证仅含空白字符的输出返回错误。
func TestParseTraceCheckOutput_OnlyWhitespace(t *testing.T) {
	_, err := parseTraceCheckOutput("   \n\t\n  ")
	if err == nil {
		t.Error("仅含空白字符的输出期望返回错误")
	}
}

// TestParseTraceCheckOutput_InvalidJSON 验证最后一行非 JSON 时返回错误。
func TestParseTraceCheckOutput_InvalidJSON(t *testing.T) {
	output := `log line 1
log line 2
not a json`

	_, err := parseTraceCheckOutput(output)
	if err == nil {
		t.Error("非 JSON 最后一行期望返回错误")
	}
}

// TestParseTraceCheckOutput_NotConfigured 验证 configured=false 的输出解析正确。
func TestParseTraceCheckOutput_NotConfigured(t *testing.T) {
	output := `{"trace_enabled":false,"trace_topic_id":"","configured":false,"reason":"trace not enabled"}`

	result, err := parseTraceCheckOutput(output)
	if err != nil {
		t.Fatalf("期望解析成功，实际 %v", err)
	}
	if result.Configured {
		t.Errorf("期望 configured=false，实际 %v", result.Configured)
	}
	if result.TraceEnabled {
		t.Errorf("期望 trace_enabled=false，实际 %v", result.TraceEnabled)
	}
}

// TestParseTraceCheckOutput_TrailingNewlines 验证末尾有多个空行时取最后一行非空内容。
func TestParseTraceCheckOutput_TrailingNewlines(t *testing.T) {
	output := `{"configured":true,"trace_enabled":true,"trace_topic_id":"t1","reason":""}

`

	result, err := parseTraceCheckOutput(output)
	if err != nil {
		t.Fatalf("期望解析成功，实际 %v", err)
	}
	if !result.Configured {
		t.Errorf("期望 configured=true，实际 %v", result.Configured)
	}
}

// ─── updateCLSPluginVersion / rollbackCLSPluginVersion 测试 ─────────────────────────────────────

// TestUpdateCLSPluginVersion_Success 验证更新版本成功。
func TestUpdateCLSPluginVersion_Success(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	model.DB(ctx).Create(&model.Instance{
		InstanceId:       "ins-update-ver",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionV1,
	})

	if err := updateCLSPluginVersion(ctx, "ins-update-ver", CLSPluginVersionV2); err != nil {
		t.Fatalf("更新版本失败: %v", err)
	}

	var inst model.Instance
	model.DB(ctx).Where("instance_id = ?", "ins-update-ver").First(&inst)
	if inst.CLSPluginVersion != CLSPluginVersionV2 {
		t.Errorf("期望版本 %s，实际 %s", CLSPluginVersionV2, inst.CLSPluginVersion)
	}
	// cls_agent_status_at 应被清空
	if inst.CLSAgentStatusAt != nil {
		t.Errorf("更新版本后 cls_agent_status_at 应为 nil，实际 %v", inst.CLSAgentStatusAt)
	}
}

// TestRollbackCLSPluginVersion_Success 验证回滚版本成功。
func TestRollbackCLSPluginVersion_Success(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	model.DB(ctx).Create(&model.Instance{
		InstanceId:       "ins-rollback-ver",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionUpdating,
	})

	if err := rollbackCLSPluginVersion(ctx, "ins-rollback-ver"); err != nil {
		t.Fatalf("回滚版本失败: %v", err)
	}

	var inst model.Instance
	model.DB(ctx).Where("instance_id = ?", "ins-rollback-ver").First(&inst)
	if inst.CLSPluginVersion != CLSPluginVersionV1 {
		t.Errorf("期望回滚到 %s，实际 %s", CLSPluginVersionV1, inst.CLSPluginVersion)
	}
	// cls_agent_status_at 应被清空
	if inst.CLSAgentStatusAt != nil {
		t.Errorf("回滚版本后 cls_agent_status_at 应为 nil，实际 %v", inst.CLSAgentStatusAt)
	}
}

// TestUpdateCLSPluginVersion_NonExistentInstance 验证更新不存在的实例不报错（RowsAffected=0）。
func TestUpdateCLSPluginVersion_NonExistentInstance(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	// 不存在的实例 ID，更新应静默成功（不报错）
	if err := updateCLSPluginVersion(ctx, "ins-nonexistent", CLSPluginVersionV2); err != nil {
		t.Errorf("更新不存在实例不应报错，实际 %v", err)
	}
}

// TestRollbackCLSPluginVersion_NonExistentInstance 验证回滚不存在的实例不报错。
func TestRollbackCLSPluginVersion_NonExistentInstance(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	if err := rollbackCLSPluginVersion(ctx, "ins-nonexistent"); err != nil {
		t.Errorf("回滚不存在实例不应报错，实际 %v", err)
	}
}

// ─── runCLSPluginUpdate 测试 ─────────────────────────────────────

// TestRunCLSPluginUpdate_GetCVMClientFails 验证 GetCVMClient 失败时所有实例计入 failed。
// 无 CVM 配置时 GetCVMClient 会返回错误。
func TestRunCLSPluginUpdate_GetCVMClientFails(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	instances := []model.Instance{
		{InstanceId: "ins-cvm-fail-1", Name: "实例1", CLSAgentStatus: model.CLSAgentInstalled, CLSPluginVersion: CLSPluginVersionV1},
		{InstanceId: "ins-cvm-fail-2", Name: "实例2", CLSAgentStatus: model.CLSAgentInstalled, CLSPluginVersion: CLSPluginVersionV1},
	}

	clawResult := &CLSClawServiceResult{
		MetricTopicId: "metric-topic",
		TopicId:       "log-topic",
		TraceTopicId:  "trace-topic",
	}

	// 无 CVM 配置，GetCVMClient 会失败，所有实例应计入 failed
	stats := runCLSPluginUpdate(ctx, instances, clawResult)

	if stats.total != 2 {
		t.Errorf("期望 total=2，实际 %d", stats.total)
	}
	if stats.failed != 2 {
		t.Errorf("期望 failed=2（CVM 客户端失败），实际 %d", stats.failed)
	}
}

// TestRunCLSPluginUpdate_EmptyInstances 验证空实例列表时直接返回零统计。
func TestRunCLSPluginUpdate_EmptyInstances(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	clawResult := &CLSClawServiceResult{
		MetricTopicId: "metric-topic",
		TopicId:       "log-topic",
		TraceTopicId:  "trace-topic",
	}

	stats := runCLSPluginUpdate(ctx, []model.Instance{}, clawResult)

	if stats.total != 0 {
		t.Errorf("期望 total=0，实际 %d", stats.total)
	}
	if stats.failed != 0 {
		t.Errorf("期望 failed=0，实际 %d", stats.failed)
	}
}

// ─── runSingleCLSPluginUpdate 测试 ─────────────────────────────────────

// TestRunSingleCLSPluginUpdate_AlreadyV2Skipped 验证已是 v2.0 的实例被跳过。
func TestRunSingleCLSPluginUpdate_AlreadyV2Skipped(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	inst := model.Instance{
		InstanceId:       "ins-already-v2",
		Name:             "已升级实例",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionV2,
	}
	model.DB(ctx).Create(&inst)

	stats := &clsUpdateStats{total: 1}
	clawResult := &CLSClawServiceResult{
		MetricTopicId: "metric-topic",
		TopicId:       "log-topic",
		TraceTopicId:  "trace-topic",
	}

	runSingleCLSPluginUpdate(ctx, inst, clawResult, stats, defaultCLSScriptRunner, mockFilterRunner([]string{inst.InstanceId}, nil))

	if stats.skipped != 1 {
		t.Errorf("已是 v2.0 的实例期望 skipped=1，实际 %d", stats.skipped)
	}
	if stats.failed != 0 {
		t.Errorf("期望 failed=0，实际 %d", stats.failed)
	}
}

// TestRunSingleCLSPluginUpdate_CASRaceConditionSkipped 验证 CAS 失败（RowsAffected=0）时实例被跳过。
// 模拟并发场景：实例已被其他请求抢先处理（版本已不是 1.0 或空字符串）。
func TestRunSingleCLSPluginUpdate_CASRaceConditionSkipped(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	// 创建一个 updating 状态的实例（CAS 条件 cls_plugin_version IN ('1.0', '') 不匹配）
	inst := model.Instance{
		InstanceId:       "ins-cas-race",
		Name:             "并发竞争实例",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionUpdating, // 已被其他请求标记为 updating
	}
	model.DB(ctx).Create(&inst)

	stats := &clsUpdateStats{total: 1}
	clawResult := &CLSClawServiceResult{
		MetricTopicId: "metric-topic",
		TopicId:       "log-topic",
		TraceTopicId:  "trace-topic",
	}

	runSingleCLSPluginUpdate(ctx, inst, clawResult, stats, defaultCLSScriptRunner, mockFilterRunner([]string{inst.InstanceId}, nil))

	if stats.skipped != 1 {
		t.Errorf("CAS 失败时期望 skipped=1，实际 %d", stats.skipped)
	}
	if stats.failed != 0 {
		t.Errorf("期望 failed=0，实际 %d", stats.failed)
	}
}

// ─── runSingleCLSPluginUpdate（mock runner）测试 ─────────────────────────────────────

// mockScriptRunner 创建一个返回固定输出的 mock 脚本执行器。
func mockScriptRunner(output string, err error) clsScriptRunner {
	return func(_ context.Context, _, _ string, _ uint64, _ string, _ func(string), _ map[string]string) (string, error) {
		return output, err
	}
}

// mockScriptRunnerByName 创建一个根据脚本名称返回不同结果的 mock 脚本执行器。
func mockScriptRunnerByName(results map[string]struct {
	output string
	err    error
}) clsScriptRunner {
	return func(_ context.Context, _, scriptName string, _ uint64, _ string, _ func(string), _ map[string]string) (string, error) {
		if r, ok := results[scriptName]; ok {
			return r.output, r.err
		}
		return "", fmt.Errorf("未知脚本: %s", scriptName)
	}
}

// TestRunSingleCLSPluginUpdate_RunScriptFails 验证 RunScript（cls_check_trace.sh）失败时回滚版本并计入 failed。
func TestRunSingleCLSPluginUpdate_RunScriptFails(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	inst := model.Instance{
		InstanceId:       "ins-script-fail",
		Name:             "脚本失败实例",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionV1,
	}
	model.DB(ctx).Create(&inst)

	stats := &clsUpdateStats{total: 1}
	clawResult := &CLSClawServiceResult{
		MetricTopicId: "metric-topic",
		TopicId:       "log-topic",
		TraceTopicId:  "trace-topic",
	}

	runner := mockScriptRunner("", fmt.Errorf("TAT 执行失败"))
	runSingleCLSPluginUpdate(ctx, inst, clawResult, stats, runner, mockFilterRunner([]string{inst.InstanceId}, nil))

	if stats.failed != 1 {
		t.Errorf("RunScript 失败时期望 failed=1，实际 %d", stats.failed)
	}
	// 版本应被回滚为 1.0
	var got model.Instance
	model.DB(ctx).Where("instance_id = ?", "ins-script-fail").First(&got)
	if got.CLSPluginVersion != CLSPluginVersionV1 {
		t.Errorf("RunScript 失败后版本应回滚为 1.0，实际 %s", got.CLSPluginVersion)
	}
}

// TestRunSingleCLSPluginUpdate_ParseOutputFails 验证脚本输出解析失败时回滚版本并计入 failed。
func TestRunSingleCLSPluginUpdate_ParseOutputFails(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	inst := model.Instance{
		InstanceId:       "ins-parse-fail",
		Name:             "解析失败实例",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionV1,
	}
	model.DB(ctx).Create(&inst)

	stats := &clsUpdateStats{total: 1}
	clawResult := &CLSClawServiceResult{
		MetricTopicId: "metric-topic",
		TopicId:       "log-topic",
		TraceTopicId:  "trace-topic",
	}

	// 返回无效 JSON，解析会失败
	runner := mockScriptRunner("not a json output", nil)
	runSingleCLSPluginUpdate(ctx, inst, clawResult, stats, runner, mockFilterRunner([]string{inst.InstanceId}, nil))

	if stats.failed != 1 {
		t.Errorf("解析失败时期望 failed=1，实际 %d", stats.failed)
	}
	// 版本应被回滚为 1.0
	var got model.Instance
	model.DB(ctx).Where("instance_id = ?", "ins-parse-fail").First(&got)
	if got.CLSPluginVersion != CLSPluginVersionV1 {
		t.Errorf("解析失败后版本应回滚为 1.0，实际 %s", got.CLSPluginVersion)
	}
}

// TestRunSingleCLSPluginUpdate_ConfiguredTrue_UpgradeOnly 验证 configured=true 时仅更新版本为 2.0，不重新安装。
func TestRunSingleCLSPluginUpdate_ConfiguredTrue_UpgradeOnly(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	inst := model.Instance{
		InstanceId:       "ins-configured-true",
		Name:             "已配置 trace 实例",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionV1,
	}
	model.DB(ctx).Create(&inst)

	stats := &clsUpdateStats{total: 1}
	clawResult := &CLSClawServiceResult{
		MetricTopicId: "metric-topic",
		TopicId:       "log-topic",
		TraceTopicId:  "trace-topic",
	}

	// configured=true，trace 已配置，仅更新版本
	runner := mockScriptRunner(`{"trace_enabled":true,"trace_topic_id":"t1","configured":true,"reason":"ok"}`, nil)
	runSingleCLSPluginUpdate(ctx, inst, clawResult, stats, runner, mockFilterRunner([]string{inst.InstanceId}, nil))

	if stats.upgraded != 1 {
		t.Errorf("configured=true 时期望 upgraded=1，实际 %d", stats.upgraded)
	}
	if stats.reinstalled != 0 {
		t.Errorf("configured=true 时期望 reinstalled=0，实际 %d", stats.reinstalled)
	}
	// 版本应更新为 2.0
	var got model.Instance
	model.DB(ctx).Where("instance_id = ?", "ins-configured-true").First(&got)
	if got.CLSPluginVersion != CLSPluginVersionV2 {
		t.Errorf("configured=true 后版本应为 2.0，实际 %s", got.CLSPluginVersion)
	}
}

// TestRunSingleCLSPluginUpdate_ConfiguredFalse_Reinstall 验证 configured=false 时执行重新安装并更新版本为 2.0。
func TestRunSingleCLSPluginUpdate_ConfiguredFalse_Reinstall(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	inst := model.Instance{
		InstanceId:       "ins-configured-false",
		Name:             "未配置 trace 实例",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionV1,
	}
	model.DB(ctx).Create(&inst)

	stats := &clsUpdateStats{total: 1}
	clawResult := &CLSClawServiceResult{
		MetricTopicId: "metric-topic",
		TopicId:       "log-topic",
		TraceTopicId:  "trace-topic",
	}

	// configured=false，需要重新安装
	runner := mockScriptRunnerByName(map[string]struct {
		output string
		err    error
	}{
		"cls_check_trace.sh":      {output: `{"trace_enabled":false,"trace_topic_id":"","configured":false,"reason":"not configured"}`, err: nil},
		"cls_plugin_reinstall.sh": {output: "reinstall success", err: nil},
	})
	runSingleCLSPluginUpdate(ctx, inst, clawResult, stats, runner, mockFilterRunner([]string{inst.InstanceId}, nil))

	if stats.reinstalled != 1 {
		t.Errorf("configured=false 时期望 reinstalled=1，实际 %d", stats.reinstalled)
	}
	if stats.upgraded != 0 {
		t.Errorf("configured=false 时期望 upgraded=0，实际 %d", stats.upgraded)
	}
	// 版本应更新为 2.0
	var got model.Instance
	model.DB(ctx).Where("instance_id = ?", "ins-configured-false").First(&got)
	if got.CLSPluginVersion != CLSPluginVersionV2 {
		t.Errorf("重新安装后版本应为 2.0，实际 %s", got.CLSPluginVersion)
	}
}

// TestRunSingleCLSPluginUpdate_ReinstallFails 验证重新安装失败时回滚版本并计入 failed。
func TestRunSingleCLSPluginUpdate_ReinstallFails(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	inst := model.Instance{
		InstanceId:       "ins-reinstall-fail",
		Name:             "重新安装失败实例",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionV1,
	}
	model.DB(ctx).Create(&inst)

	stats := &clsUpdateStats{total: 1}
	clawResult := &CLSClawServiceResult{
		MetricTopicId: "metric-topic",
		TopicId:       "log-topic",
		TraceTopicId:  "trace-topic",
	}

	// configured=false，重新安装失败
	runner := mockScriptRunnerByName(map[string]struct {
		output string
		err    error
	}{
		"cls_check_trace.sh":      {output: `{"trace_enabled":false,"trace_topic_id":"","configured":false,"reason":"not configured"}`, err: nil},
		"cls_plugin_reinstall.sh": {output: "", err: fmt.Errorf("npx 安装失败")},
	})
	runSingleCLSPluginUpdate(ctx, inst, clawResult, stats, runner, mockFilterRunner([]string{inst.InstanceId}, nil))

	if stats.failed != 1 {
		t.Errorf("重新安装失败时期望 failed=1，实际 %d", stats.failed)
	}
	// 版本应被回滚为 1.0
	var got model.Instance
	model.DB(ctx).Where("instance_id = ?", "ins-reinstall-fail").First(&got)
	if got.CLSPluginVersion != CLSPluginVersionV1 {
		t.Errorf("重新安装失败后版本应回滚为 1.0，实际 %s", got.CLSPluginVersion)
	}
}

// ─── runCLSPluginUpdateWithRunner 测试 ─────────────────────────────────────

// mockFilterRunner 创建一个返回固定结果的 mock 过滤器。
func mockFilterRunner(ids []string, err error) clsFilterRunner {
	return func(_ context.Context, _ []string) ([]string, error) {
		return ids, err
	}
}

// TestRunCLSPluginUpdateWithRunner_CVMClientFails 验证 CVM 过滤器失败时所有实例计入 failed。
func TestRunCLSPluginUpdateWithRunner_CVMClientFails(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	instances := []model.Instance{
		{InstanceId: "ins-runner-1", Name: "实例1", CLSAgentStatus: model.CLSAgentInstalled, CLSPluginVersion: CLSPluginVersionV1},
	}
	clawResult := &CLSClawServiceResult{
		MetricTopicId: "metric-topic",
		TopicId:       "log-topic",
		TraceTopicId:  "trace-topic",
	}

	runner := mockScriptRunner(`{"configured":true,"trace_enabled":true,"trace_topic_id":"t1","reason":""}`, nil)
	filter := mockFilterRunner(nil, fmt.Errorf("CVM 查询失败"))
	stats := runCLSPluginUpdateWithRunner(ctx, instances, clawResult, runner, filter)

	// 过滤器失败，所有实例计入 failed
	if stats.failed != 1 {
		t.Errorf("CVM 过滤器失败时期望 failed=1，实际 %d", stats.failed)
	}
}

// TestRunCLSPluginUpdateWithRunner_AllNotRunningSkipped 验证所有实例非运行中时全部计入 skipped。
func TestRunCLSPluginUpdateWithRunner_AllNotRunningSkipped(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	instances := []model.Instance{
		{InstanceId: "ins-not-running-1", Name: "实例1", CLSAgentStatus: model.CLSAgentInstalled, CLSPluginVersion: CLSPluginVersionV1},
		{InstanceId: "ins-not-running-2", Name: "实例2", CLSAgentStatus: model.CLSAgentInstalled, CLSPluginVersion: CLSPluginVersionV1},
	}
	clawResult := &CLSClawServiceResult{
		MetricTopicId: "metric-topic",
		TopicId:       "log-topic",
		TraceTopicId:  "trace-topic",
	}

	runner := mockScriptRunner("", nil)
	// 过滤器返回空列表，所有实例均非运行中
	filter := mockFilterRunner([]string{}, nil)
	stats := runCLSPluginUpdateWithRunner(ctx, instances, clawResult, runner, filter)

	if stats.skipped != 2 {
		t.Errorf("所有实例非运行中时期望 skipped=2，实际 %d", stats.skipped)
	}
	if stats.failed != 0 {
		t.Errorf("期望 failed=0，实际 %d", stats.failed)
	}
}

// TestRunCLSPluginUpdateWithRunner_PartialRunning 验证部分实例运行中时正确分类。
func TestRunCLSPluginUpdateWithRunner_PartialRunning(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	// 创建两个实例，一个运行中一个不运行
	model.DB(ctx).Create(&model.Instance{
		InstanceId:       "ins-partial-running",
		Name:             "运行中实例",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionV1,
	})
	model.DB(ctx).Create(&model.Instance{
		InstanceId:       "ins-partial-stopped",
		Name:             "停止实例",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionV1,
	})

	instances := []model.Instance{
		{InstanceId: "ins-partial-running", Name: "运行中实例", CLSAgentStatus: model.CLSAgentInstalled, CLSPluginVersion: CLSPluginVersionV1},
		{InstanceId: "ins-partial-stopped", Name: "停止实例", CLSAgentStatus: model.CLSAgentInstalled, CLSPluginVersion: CLSPluginVersionV1},
	}
	clawResult := &CLSClawServiceResult{
		MetricTopicId: "metric-topic",
		TopicId:       "log-topic",
		TraceTopicId:  "trace-topic",
	}

	// 只有 ins-partial-running 是运行中
	filter := mockFilterRunner([]string{"ins-partial-running"}, nil)
	// configured=true，直接升级版本
	runner := mockScriptRunner(`{"configured":true,"trace_enabled":true,"trace_topic_id":"t1","reason":"ok"}`, nil)
	stats := runCLSPluginUpdateWithRunner(ctx, instances, clawResult, runner, filter)

	if stats.skipped != 1 {
		t.Errorf("停止实例期望 skipped=1，实际 %d", stats.skipped)
	}
	if stats.upgraded != 1 {
		t.Errorf("运行中实例期望 upgraded=1，实际 %d", stats.upgraded)
	}
}

// TestRunCLSPluginUpdateWithRunner_ConfiguredTrueUpdateVersionFails 验证 configured=true 时
// updateCLSPluginVersion 失败时回滚版本并计入 failed。
func TestRunCLSPluginUpdateWithRunner_ConfiguredTrueUpdateVersionFails(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	model.DB(ctx).Create(&model.Instance{
		InstanceId:       "ins-update-ver-fail",
		Name:             "版本更新失败实例",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionV1,
	})

	clawResult := &CLSClawServiceResult{
		MetricTopicId: "metric-topic",
		TopicId:       "log-topic",
		TraceTopicId:  "trace-topic",
	}

	// 先执行一次正常流程，标记为 updating
	// 然后关闭 DB，使 updateCLSPluginVersion 失败
	// 注意：CAS 操作也需要 DB，所以需要在 CAS 之后关闭 DB
	// 这里通过直接调用 runSingleCLSPluginUpdate 并在 runner 中关闭 DB 来模拟
	stats := &clsUpdateStats{total: 1}

	// 使用特殊 runner：在返回 configured=true 之前关闭 DB
	var dbClosed bool
	specialRunner := func(_ context.Context, _, _ string, _ uint64, _ string, _ func(string), _ map[string]string) (string, error) {
		if !dbClosed {
			dbClosed = true
			_ = model.CloseUnderlyingDBForTest()
		}
		return `{"configured":true,"trace_enabled":true,"trace_topic_id":"t1","reason":"ok"}`, nil
	}

	inst := model.Instance{
		InstanceId:       "ins-update-ver-fail",
		Name:             "版本更新失败实例",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionV1,
	}
	runSingleCLSPluginUpdate(ctx, inst, clawResult, stats, specialRunner, mockFilterRunner([]string{inst.InstanceId}, nil))

	// DB 关闭后 updateCLSPluginVersion 失败，应计入 failed
	if stats.failed != 1 {
		t.Errorf("updateCLSPluginVersion 失败时期望 failed=1，实际 %d", stats.failed)
	}
}

// TestRunCLSPluginUpdateWithRunner_ReinstallUpdateVersionFails 验证重新安装后
// updateCLSPluginVersion 失败时回滚版本并计入 failed。
func TestRunCLSPluginUpdateWithRunner_ReinstallUpdateVersionFails(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	model.DB(ctx).Create(&model.Instance{
		InstanceId:       "ins-reinstall-ver-fail",
		Name:             "重装后版本更新失败实例",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionV1,
	})

	clawResult := &CLSClawServiceResult{
		MetricTopicId: "metric-topic",
		TopicId:       "log-topic",
		TraceTopicId:  "trace-topic",
	}

	stats := &clsUpdateStats{total: 1}

	// 第一次调用（cls_check_trace.sh）返回 configured=false
	// 第二次调用（cls_plugin_reinstall.sh）成功后关闭 DB，使 updateCLSPluginVersion 失败
	callCount := 0
	specialRunner := func(_ context.Context, _, scriptName string, _ uint64, _ string, _ func(string), _ map[string]string) (string, error) {
		callCount++
		switch scriptName {
		case "cls_check_trace.sh":
			return `{"configured":false,"trace_enabled":false,"trace_topic_id":"","reason":"not configured"}`, nil
		case "cls_plugin_reinstall.sh":
			// 重装成功后关闭 DB，使后续 updateCLSPluginVersion 失败
			_ = model.CloseUnderlyingDBForTest()
			return "reinstall success", nil
		default:
			return "", fmt.Errorf("未知脚本: %s", scriptName)
		}
	}

	inst := model.Instance{
		InstanceId:       "ins-reinstall-ver-fail",
		Name:             "重装后版本更新失败实例",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionV1,
	}
	runSingleCLSPluginUpdate(ctx, inst, clawResult, stats, specialRunner, mockFilterRunner([]string{inst.InstanceId}, nil))

	// updateCLSPluginVersion 失败，应计入 failed
	if stats.failed != 1 {
		t.Errorf("重装后 updateCLSPluginVersion 失败时期望 failed=1，实际 %d", stats.failed)
	}
}

// ─── queryCLSUpdateTargetInstances 超时回滚测试 ─────────────────────────────────────

// TestQueryCLSUpdateTargetInstances_TimeoutUpdatingRollback 验证超时的 updating 实例被回滚为 1.0
// 并重新纳入待处理队列。
func TestQueryCLSUpdateTargetInstances_TimeoutUpdatingRollback(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	staleAt := time.Now().Add(-15 * time.Minute) // 早于 10 分钟超时阈值

	// 超时的 updating 实例（应被回滚为 1.0，重新纳入查询结果）
	staleInst := model.Instance{
		InstanceId:       "ins-timeout-updating",
		Name:             "超时升级中实例",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionUpdating,
		LastCVMState:     "RUNNING",
	}
	model.DB(ctx).Create(&staleInst)
	model.DB(ctx).Model(&model.Instance{}).
		Where("instance_id = ?", "ins-timeout-updating").
		Update("cls_agent_status_at", staleAt)

	// 未超时的 updating 实例（不应被回滚，也不应出现在查询结果中）
	freshInst := model.Instance{
		InstanceId:       "ins-fresh-updating",
		Name:             "未超时升级中实例",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionUpdating,
	}
	model.DB(ctx).Create(&freshInst)
	model.DB(ctx).Model(&model.Instance{}).
		Where("instance_id = ?", "ins-fresh-updating").
		Update("cls_agent_status_at", time.Now().Add(-1*time.Minute))

	req := clsUpdateRequest{ScopeType: "all"}
	instances, err := queryCLSUpdateTargetInstances(ctx, req)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}

	// 超时的 updating 实例被回滚为 1.0，应出现在查询结果中
	found := false
	for _, inst := range instances {
		if inst.InstanceId == "ins-timeout-updating" {
			found = true
		}
		if inst.InstanceId == "ins-fresh-updating" {
			t.Errorf("未超时的 updating 实例不应出现在查询结果中")
		}
	}
	if !found {
		t.Errorf("超时的 updating 实例应被回滚并出现在查询结果中")
	}

	// 验证超时实例的版本已被回滚为 1.0
	var rolledBack model.Instance
	model.DB(ctx).Where("instance_id = ?", "ins-timeout-updating").First(&rolledBack)
	if rolledBack.CLSPluginVersion != CLSPluginVersionV1 {
		t.Errorf("超时 updating 实例应被回滚为 1.0，实际 %s", rolledBack.CLSPluginVersion)
	}
}

// TestQueryCLSUpdateTargetInstances_NullStatusAtUpdatingNotRolledBack 验证 cls_agent_status_at 为 NULL
// 的 updating 实例不被回滚（NULL 表示尚未记录开始时间）。
func TestQueryCLSUpdateTargetInstances_NullStatusAtUpdatingNotRolledBack(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	// cls_agent_status_at 为 NULL 的 updating 实例（不应被回滚）
	nullInst := model.Instance{
		InstanceId:       "ins-null-updating",
		Name:             "NULL状态时间升级中实例",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionUpdating,
	}
	model.DB(ctx).Create(&nullInst)
	// 确保 cls_agent_status_at 为 NULL
	model.DB(ctx).Model(&model.Instance{}).
		Where("instance_id = ?", "ins-null-updating").
		Update("cls_agent_status_at", nil)

	req := clsUpdateRequest{ScopeType: "all"}
	_, err := queryCLSUpdateTargetInstances(ctx, req)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}

	// 验证 NULL status_at 的 updating 实例未被回滚
	var inst model.Instance
	model.DB(ctx).Where("instance_id = ?", "ins-null-updating").First(&inst)
	if inst.CLSPluginVersion != CLSPluginVersionUpdating {
		t.Errorf("NULL status_at 的 updating 实例不应被回滚，实际版本 %s", inst.CLSPluginVersion)
	}
}

// ─── HandleAdminGetCLSUpdateStats DB 失败测试 ─────────────────────────────────────

// TestHandleAdminGetCLSUpdateStats_DBError 验证 DB 查询失败时返回 500。
func TestHandleAdminGetCLSUpdateStats_DBError(t *testing.T) {
	setupCLSUpdateTestEnv(t)

	// 关闭底层 DB 连接，使后续 DB 操作失败
	if err := model.CloseUnderlyingDBForTest(); err != nil {
		t.Fatalf("关闭 DB 失败: %v", err)
	}

	req, w := clsUpdateAdminReq("GET", "/admin/cls/update", "")
	HandleAdminGetCLSUpdateStats(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("DB 失败时期望 500，实际 %d: %s", w.Code, w.Body.String())
	}
}

// ─── HandleAdminUpdateCLSPlugin DB 失败测试 ─────────────────────────────────────

// TestHandleAdminUpdateCLSPlugin_DBClosedReturnsBadRequest 验证 DB 关闭时 GetSiteConfig 返回默认值
// （CLSEnabled=0），HandleAdminUpdateCLSPlugin 返回 400（CLS 未开启）。
func TestHandleAdminUpdateCLSPlugin_DBClosedReturnsBadRequest(t *testing.T) {
	setupCLSUpdateTestEnv(t)

	// 关闭底层 DB 连接，GetSiteConfig 会返回默认值（CLSEnabled=0）
	if err := model.CloseUnderlyingDBForTest(); err != nil {
		t.Fatalf("关闭 DB 失败: %v", err)
	}

	req, w := clsUpdateAdminReq("POST", "/admin/cls/update", `{"scope_type":"all"}`)
	HandleAdminUpdateCLSPlugin(w, req)

	// DB 关闭时 GetSiteConfig 返回默认值 CLSEnabled=0，应返回 400
	if w.Code != http.StatusBadRequest {
		t.Errorf("DB 关闭时期望 400（CLS 未开启），实际 %d: %s", w.Code, w.Body.String())
	}
}

// TestHandleAdminUpdateCLSPlugin_QueryInstancesDBError 验证 queryCLSUpdateTargetInstances
// 失败时返回 500（行 98-100）。
func TestHandleAdminUpdateCLSPlugin_QueryInstancesDBError(t *testing.T) {
	setupCLSUpdateTestEnv(t)

	// 注入一个返回错误的 queryRunner，模拟 queryCLSUpdateTargetInstances 失败
	errorQueryRunner := func(_ context.Context, _ clsUpdateRequest) ([]model.Instance, error) {
		return nil, fmt.Errorf("DB 查询失败")
	}
	errorClawRunner := func(_ context.Context) (*CLSClawServiceResult, error) {
		return nil, fmt.Errorf("不应被调用")
	}

	// 需要 CLSEnabled=1 才能走到 queryRunner
	ctx := context.Background()
	model.DB(ctx).Create(&model.SiteConfig{CLSEnabled: 1})

	req, w := clsUpdateAdminReq("POST", "/admin/cls/update", `{"scope_type":"all"}`)
	handleAdminUpdateCLSPluginWithRunners(w, req, errorQueryRunner, errorClawRunner)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("queryCLSUpdateTargetInstances 失败时期望 500，实际 %d: %s", w.Code, w.Body.String())
	}
}

// TestHandleAdminUpdateCLSPlugin_NewCLSCommonClientFails 验证 clawRunner 失败时返回 500
// （行 116-125 / 127-141）。
func TestHandleAdminUpdateCLSPlugin_NewCLSCommonClientFails(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	model.DB(ctx).Create(&model.SiteConfig{CLSEnabled: 1})

	// queryRunner 返回一个待升级实例，使流程走到 clawRunner
	mockQueryRunner := func(_ context.Context, _ clsUpdateRequest) ([]model.Instance, error) {
		return []model.Instance{
			{InstanceId: "ins-cls-client-fail", CLSAgentStatus: model.CLSAgentInstalled, CLSPluginVersion: CLSPluginVersionV1},
		}, nil
	}
	// clawRunner 返回错误，模拟 newCLSCommonClient 或 openClawService 失败
	errorClawRunner := func(_ context.Context) (*CLSClawServiceResult, error) {
		return nil, fmt.Errorf("凭据未配置")
	}

	req, w := clsUpdateAdminReq("POST", "/admin/cls/update", `{"scope_type":"all"}`)
	handleAdminUpdateCLSPluginWithRunners(w, req, mockQueryRunner, errorClawRunner)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("clawRunner 失败时期望 500，实际 %d: %s", w.Code, w.Body.String())
	}
}

// TestRunSingleCLSPluginUpdate_CASDBError 验证 CAS UPDATE 失败（DB 错误）时计入 failed（行 214/311-314）。
// 通过在调用 runSingleCLSPluginUpdate 前关闭 DB 来触发 CAS DB 错误。
func TestRunSingleCLSPluginUpdate_CASDBError(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	model.DB(ctx).Create(&model.Instance{
		InstanceId:       "ins-cas-db-err",
		Name:             "CAS DB 错误实例",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionV1,
	})

	// 关闭底层 DB 连接，使 CAS UPDATE 失败
	if err := model.CloseUnderlyingDBForTest(); err != nil {
		t.Fatalf("关闭 DB 失败: %v", err)
	}

	stats := &clsUpdateStats{total: 1}
	clawResult := &CLSClawServiceResult{
		MetricTopicId: "metric-topic",
		TopicId:       "log-topic",
		TraceTopicId:  "trace-topic",
	}

	inst := model.Instance{
		InstanceId:       "ins-cas-db-err",
		Name:             "CAS DB 错误实例",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionV1,
	}
	runner := mockScriptRunner("", nil)
	runSingleCLSPluginUpdate(ctx, inst, clawResult, stats, runner, mockFilterRunner([]string{inst.InstanceId}, nil))

	// CAS DB 错误，应计入 failed
	if stats.failed != 1 {
		t.Errorf("CAS DB 错误时期望 failed=1，实际 %d", stats.failed)
	}
}

// ─── handleAdminUpdateCLSPluginWithRunners 成功路径 ────────────────────────

// TestHandleAdminUpdateCLSPlugin_SuccessPath 验证有实例且 clawRunner 成功时返回 200
// 并包含统计字段（行 148-165）。
func TestHandleAdminUpdateCLSPlugin_SuccessPath(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	model.DB(ctx).Create(&model.SiteConfig{CLSEnabled: 1})
	model.DB(ctx).Create(&model.Instance{
		InstanceId:       "ins-success-path",
		Name:             "成功路径实例",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionV1,
	})

	mockQueryRunner := func(_ context.Context, _ clsUpdateRequest) ([]model.Instance, error) {
		return []model.Instance{
			{InstanceId: "ins-success-path", CLSAgentStatus: model.CLSAgentInstalled, CLSPluginVersion: CLSPluginVersionV1},
		}, nil
	}
	mockClawRunner := func(_ context.Context) (*CLSClawServiceResult, error) {
		return &CLSClawServiceResult{
			MetricTopicId: "metric-topic",
			TopicId:       "log-topic",
			TraceTopicId:  "trace-topic",
		}, nil
	}

	req, w := clsUpdateAdminReq("POST", "/admin/cls/update", `{"scope_type":"all"}`)
	handleAdminUpdateCLSPluginWithRunners(w, req, mockQueryRunner, mockClawRunner)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d: %s", w.Code, w.Body.String())
	}
	result := parseCLSUpdateResp(t, w)
	if result["ok"] != true {
		t.Errorf("期望 ok=true，实际 %v", result["ok"])
	}
	if _, ok := result["total"]; !ok {
		t.Errorf("响应中缺少 total 字段")
	}
}

// ─── batchRollbackTimedOutUpdating / batchQueryCLSTargetInstances 错误路径 ─

// TestBatchRollbackTimedOutUpdating_UpdateFails 验证 UPDATE 失败时返回错误（行 198-199）。
func TestBatchRollbackTimedOutUpdating_UpdateFails(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	// 创建一个超时的 updating 实例，使 Pluck 能查到 ids
	staleAt := time.Now().Add(-20 * time.Minute)
	inst := model.Instance{
		InstanceId:       "ins-rollback-fail",
		Name:             "回滚失败实例",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionUpdating,
	}
	model.DB(ctx).Create(&inst)
	model.DB(ctx).Model(&model.Instance{}).
		Where("id = ?", inst.ID).
		Update("cls_agent_status_at", staleAt)

	// 关闭 DB，使 UPDATE 失败
	if err := model.CloseUnderlyingDBForTest(); err != nil {
		t.Fatalf("关闭 DB 失败: %v", err)
	}

	err := batchRollbackTimedOutUpdating(ctx, time.Now().Add(-10*time.Minute))
	if err == nil {
		t.Error("DB 关闭后期望 batchRollbackTimedOutUpdating 返回错误，实际 nil")
	}
}

// TestBatchQueryCLSTargetInstances_DBError 验证 Find 失败时返回错误（行 217-218）。
func TestBatchQueryCLSTargetInstances_DBError(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	// 关闭 DB，使 Find 失败
	if err := model.CloseUnderlyingDBForTest(); err != nil {
		t.Fatalf("关闭 DB 失败: %v", err)
	}

	_, err := batchQueryCLSTargetInstances(ctx)
	if err == nil {
		t.Error("DB 关闭后期望 batchQueryCLSTargetInstances 返回错误，实际 nil")
	}
}

// TestBatchQueryCLSTargetInstances_MultiBatch 验证多批次查询能正确汇总结果（行 228-229 / 247-248）。
// 创建超过 clsQueryBatchSize 条实例，验证分页游标逻辑正确。
func TestBatchQueryCLSTargetInstances_MultiBatch(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	// 创建 clsQueryBatchSize+5 条待升级实例，触发多批次查询
	total := clsQueryBatchSize + 5
	for i := range total {
		model.DB(ctx).Create(&model.Instance{
			InstanceId:       fmt.Sprintf("ins-batch-%04d", i),
			Name:             fmt.Sprintf("批次实例%d", i),
			CLSAgentStatus:   model.CLSAgentInstalled,
			CLSPluginVersion: CLSPluginVersionV1,
			LastCVMState:     "RUNNING",
		})
	}

	result, err := batchQueryCLSTargetInstances(ctx)
	if err != nil {
		t.Fatalf("多批次查询失败: %v", err)
	}
	if len(result) != total {
		t.Errorf("期望 %d 条实例，实际 %d", total, len(result))
	}
}

// ─── rollback 失败日志路径 ─────────────────────────────────────────────────

// TestRunSingleCLSPluginUpdate_ScriptFailRollbackFails 验证脚本失败且 rollback 也失败时
// 正确记录日志并计入 failed（行 391-392）。
func TestRunSingleCLSPluginUpdate_ScriptFailRollbackFails(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	inst := model.Instance{
		InstanceId:       "ins-script-rb-fail",
		Name:             "脚本失败回滚失败",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionV1,
	}
	model.DB(ctx).Create(&inst)

	stats := &clsUpdateStats{total: 1}
	clawResult := &CLSClawServiceResult{
		MetricTopicId: "metric-topic",
		TopicId:       "log-topic",
		TraceTopicId:  "trace-topic",
	}

	// runner 在被调用时关闭 DB 并返回错误：
	// CAS 成功（DB 开着）→ runner 关闭 DB 并返回错误 → rollback 失败（DB 已关）
	failRunner := func(_ context.Context, _, _ string, _ uint64, _ string, _ func(string), _ map[string]string) (string, error) {
		_ = model.CloseUnderlyingDBForTest()
		return "", fmt.Errorf("脚本执行失败")
	}

	runSingleCLSPluginUpdate(ctx, inst, clawResult, stats, failRunner, mockFilterRunner([]string{inst.InstanceId}, nil))

	if stats.failed != 1 {
		t.Errorf("期望 failed=1，实际 %d", stats.failed)
	}
}

// TestRunSingleCLSPluginUpdate_ParseFailRollbackFails 验证解析失败且 rollback 也失败时
// 正确记录日志并计入 failed（行 402-403）。
func TestRunSingleCLSPluginUpdate_ParseFailRollbackFails(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	inst := model.Instance{
		InstanceId:       "ins-parse-rb-fail",
		Name:             "解析失败回滚失败",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionV1,
	}
	model.DB(ctx).Create(&inst)

	stats := &clsUpdateStats{total: 1}
	clawResult := &CLSClawServiceResult{
		MetricTopicId: "metric-topic",
		TopicId:       "log-topic",
		TraceTopicId:  "trace-topic",
	}

	// runner 在被调用时关闭 DB 并返回无效 JSON：
	// CAS 成功（DB 开着）→ runner 关闭 DB 并返回无效 JSON → 解析失败 → rollback 失败（DB 已关）
	invalidJSONRunner := func(_ context.Context, _, _ string, _ uint64, _ string, _ func(string), _ map[string]string) (string, error) {
		_ = model.CloseUnderlyingDBForTest()
		return "not-valid-json", nil
	}

	runSingleCLSPluginUpdate(ctx, inst, clawResult, stats, invalidJSONRunner, mockFilterRunner([]string{inst.InstanceId}, nil))

	// 解析失败，应计入 failed
	if stats.failed != 1 {
		t.Errorf("解析失败时期望 failed=1，实际 %d", stats.failed)
	}
}

// TestRunSingleCLSPluginUpdate_ConfiguredUpdateVersionFailRollbackFails 验证
// configured=true 时 updateVersion 失败且 rollback 也失败时正确处理（行 441-442）。
func TestRunSingleCLSPluginUpdate_ConfiguredUpdateVersionFailRollbackFails(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	inst := model.Instance{
		InstanceId:       "ins-cfg-upd-rb-fail",
		Name:             "配置更新回滚失败",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionV1,
	}
	model.DB(ctx).Create(&inst)

	stats := &clsUpdateStats{total: 1}
	clawResult := &CLSClawServiceResult{
		MetricTopicId: "metric-topic",
		TopicId:       "log-topic",
		TraceTopicId:  "trace-topic",
	}

	// runner 在被调用时关闭 DB 并返回 configured=true 的 JSON：
	// CAS 成功（DB 开着）→ runner 关闭 DB 并返回 configured=true
	// → updateCLSPluginVersion 失败（DB 已关）→ rollback 也失败
	configuredRunner := func(_ context.Context, _, scriptName string, _ uint64, _ string, _ func(string), _ map[string]string) (string, error) {
		if scriptName == "cls_check_trace.sh" {
			_ = model.CloseUnderlyingDBForTest()
			return `{"configured":true,"trace_enabled":true,"trace_topic_id":"topic-xxx"}`, nil
		}
		return "", nil
	}

	runSingleCLSPluginUpdate(ctx, inst, clawResult, stats, configuredRunner, mockFilterRunner([]string{inst.InstanceId}, nil))

	if stats.failed != 1 {
		t.Errorf("期望 failed=1，实际 %d", stats.failed)
	}
}

// ─── runSingleCLSPluginUpdate 二次确认测试 ─────────────────────────────────────────

// TestRunSingleCLSPluginUpdate_SecondCheckInstanceNotRunning 验证 CAS 成功后二次确认发现实例
// 已不在运行中时，回滚版本并计入 skipped（不计入 failed）。
func TestRunSingleCLSPluginUpdate_SecondCheckInstanceNotRunning(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	inst := model.Instance{
		InstanceId:       "ins-second-check-stopped",
		Name:             "二次确认停机实例",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionV1,
	}
	model.DB(ctx).Create(&inst)

	stats := &clsUpdateStats{total: 1}
	clawResult := &CLSClawServiceResult{
		MetricTopicId: "metric-topic",
		TopicId:       "log-topic",
		TraceTopicId:  "trace-topic",
	}

	// 二次确认返回空列表，表示实例已不在运行中
	runner := mockScriptRunner(`{"configured":true,"trace_enabled":true,"trace_topic_id":"t1","reason":"ok"}`, nil)
	filter := mockFilterRunner([]string{}, nil)
	runSingleCLSPluginUpdate(ctx, inst, clawResult, stats, runner, filter)

	// 应计入 skipped，不计入 failed
	if stats.skipped != 1 {
		t.Errorf("二次确认实例停机时期望 skipped=1，实际 %d", stats.skipped)
	}
	if stats.failed != 0 {
		t.Errorf("二次确认实例停机时期望 failed=0，实际 %d", stats.failed)
	}
	// 版本应被回滚为 1.0
	var got model.Instance
	model.DB(ctx).Where("instance_id = ?", "ins-second-check-stopped").First(&got)
	if got.CLSPluginVersion != CLSPluginVersionV1 {
		t.Errorf("二次确认停机后版本应回滚为 1.0，实际 %s", got.CLSPluginVersion)
	}
}

// TestRunSingleCLSPluginUpdate_SecondCheckFilterError 验证 CAS 成功后二次确认 CVM API 失败时
// 继续尝试下发脚本（降级处理，不中断流程）。
func TestRunSingleCLSPluginUpdate_SecondCheckFilterError(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	inst := model.Instance{
		InstanceId:       "ins-second-check-filter-err",
		Name:             "二次确认过滤器失败实例",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionV1,
	}
	model.DB(ctx).Create(&inst)

	stats := &clsUpdateStats{total: 1}
	clawResult := &CLSClawServiceResult{
		MetricTopicId: "metric-topic",
		TopicId:       "log-topic",
		TraceTopicId:  "trace-topic",
	}

	// 二次确认 CVM API 失败，但流程应继续（降级处理）
	// runner 返回 configured=true，最终应升级成功
	runner := mockScriptRunner(`{"configured":true,"trace_enabled":true,"trace_topic_id":"t1","reason":"ok"}`, nil)
	filter := mockFilterRunner(nil, fmt.Errorf("CVM API 抖动"))
	runSingleCLSPluginUpdate(ctx, inst, clawResult, stats, runner, filter)

	// 二次确认失败时降级继续，脚本成功则应升级
	if stats.upgraded != 1 {
		t.Errorf("二次确认失败时应降级继续，期望 upgraded=1，实际 %d", stats.upgraded)
	}
	if stats.skipped != 0 {
		t.Errorf("期望 skipped=0，实际 %d", stats.skipped)
	}
	// 版本应更新为 2.0
	var got model.Instance
	model.DB(ctx).Where("instance_id = ?", "ins-second-check-filter-err").First(&got)
	if got.CLSPluginVersion != CLSPluginVersionV2 {
		t.Errorf("二次确认失败降级后版本应为 2.0，实际 %s", got.CLSPluginVersion)
	}
}

// TestRunSingleCLSPluginUpdate_SecondCheckNotRunningRollbackFails 验证 CAS 成功后二次确认
// 发现实例停机，且 rollback 也失败时，正确记录日志并计入 skipped。
func TestRunSingleCLSPluginUpdate_SecondCheckNotRunningRollbackFails(t *testing.T) {
	setupCLSUpdateTestEnv(t)
	ctx := context.Background()

	inst := model.Instance{
		InstanceId:       "ins-second-check-rb-fail",
		Name:             "二次确认停机回滚失败实例",
		CLSAgentStatus:   model.CLSAgentInstalled,
		CLSPluginVersion: CLSPluginVersionV1,
	}
	model.DB(ctx).Create(&inst)

	stats := &clsUpdateStats{total: 1}
	clawResult := &CLSClawServiceResult{
		MetricTopicId: "metric-topic",
		TopicId:       "log-topic",
		TraceTopicId:  "trace-topic",
	}

	// 二次确认返回空列表（实例停机），同时关闭 DB 使 rollback 失败
	runner := mockScriptRunner(`{"configured":true,"trace_enabled":true,"trace_topic_id":"t1","reason":"ok"}`, nil)
	filter := func(_ context.Context, _ []string) ([]string, error) {
		_ = model.CloseUnderlyingDBForTest()
		return []string{}, nil
	}
	runSingleCLSPluginUpdate(ctx, inst, clawResult, stats, runner, filter)

	// rollback 失败，但仍应计入 skipped（不是 failed）
	if stats.skipped != 1 {
		t.Errorf("二次确认停机且 rollback 失败时期望 skipped=1，实际 %d", stats.skipped)
	}
	if stats.failed != 0 {
		t.Errorf("期望 failed=0，实际 %d", stats.failed)
	}
}
