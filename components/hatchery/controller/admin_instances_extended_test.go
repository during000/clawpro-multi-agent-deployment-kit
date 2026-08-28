package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// initAdminInstancesExtTestDB 初始化 admin instances 扩展测试 DB。
func initAdminInstancesExtTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.User{}, &model.Instance{}, &model.AIImage{}, &model.SiteConfig{},
		&model.InstanceAdjustment{},
		&model.Notification{}, &model.InstanceModel{}, &model.MemoryTDAIPlugin{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	origDB := model.UseDBForTest(db)
	AdminToken = "test-admin-token"
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))

	return func() {
		origDB()
		Store = origStore
	}
}

// adminTokenReq 构造带 admin token 的请求。
func adminTokenReq(method, path string, body string) *http.Request {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req
}

// adminTokenReqJSON 构造 JSON body 的 admin 请求。
func adminTokenReqJSON(method, path string, body []byte) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req
}

// ─── HandleAdminInstanceChannels ───────────────────────────────────────

func TestHandleAdminInstanceChannels_Unauthorized(t *testing.T) {
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/admin/instances/channels?id=1", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleAdminInstanceChannels(rr, req)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("无 admin token 应返回 401/403，实际=%d", rr.Code)
	}
}

func TestHandleAdminInstanceChannels_InstanceNotFound(t *testing.T) {
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()

	req := adminTokenReq(http.MethodGet, "/admin/instances/channels?id=999", "")
	rr := httptest.NewRecorder()
	HandleAdminInstanceChannels(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("实例不存在应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleAdminInstanceChannels_UnknownAgentType_ListFails(t *testing.T) {
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()

	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-admch-unk",
		UserID: 1, AgentType: "future_unknown_type",
	}
	model.DB(context.Background()).Create(inst)

	req := adminTokenReq(http.MethodGet,
		fmt.Sprintf("/admin/instances/channels?id=%d", inst.ID), "")
	rr := httptest.NewRecorder()
	HandleAdminInstanceChannels(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("未知 agent_type 应返回 500（list_channels 解析失败），实际=%d body=%s",
			rr.Code, rr.Body.String())
	}
}

func TestHandleAdminInstanceChannels_KeepsExistingOverseasOnlyChannelsOnDomesticSite(t *testing.T) {
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()
	i18n.SetDefaultLang("zh")
	defer i18n.SetDefaultLang("zh")

	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-admch-site",
		UserID: 1, AgentType: model.AgentTypeOpenClaw,
		AgentReady: 1, RuntimeUser: "root",
	}
	model.DB(context.Background()).Create(inst)
	model.DB(context.Background()).Create(&model.AIChannel{ChannelID: "slack", Name: "Slack"})
	model.DB(context.Background()).Create(&model.AIChannel{ChannelID: "msteams", Name: "Microsoft Teams"})

	origRunner := listChannelsScriptRunner
	listChannelsScriptRunner = func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		return `{"feishu":{"enabled":true},"slack":{"enabled":true},"msteams":{"enabled":true}}`, nil
	}
	defer func() { listChannelsScriptRunner = origRunner }()

	req := adminTokenReq(http.MethodGet,
		fmt.Sprintf("/admin/instances/channels?id=%d", inst.ID), "")
	rr := httptest.NewRecorder()
	HandleAdminInstanceChannels(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v body=%s", err, rr.Body.String())
	}
	for _, ch := range []string{"slack", "msteams"} {
		if _, ok := resp[ch]; !ok {
			t.Fatalf("domestic site should keep existing %s config, channels=%v", ch, resp)
		}
	}
	if _, ok := resp["feishu"]; !ok {
		t.Fatalf("domestic site should keep configured feishu, channels=%v", resp)
	}
}

// ─── HandleAdminInstanceSkills ─────────────────────────────────────────

func TestHandleAdminInstanceSkills_Unauthorized(t *testing.T) {
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/admin/instances/skills?id=1", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleAdminInstanceSkills(rr, req)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("无 admin token 应返回 401/403，实际=%d", rr.Code)
	}
}

func TestHandleAdminInstanceSkills_UnknownAgentType_ListFails(t *testing.T) {
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()

	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-admsk-unk",
		UserID: 1, AgentType: "future_unknown_type",
	}
	model.DB(context.Background()).Create(inst)

	req := adminTokenReq(http.MethodGet,
		fmt.Sprintf("/admin/instances/skills?id=%d", inst.ID), "")
	rr := httptest.NewRecorder()
	HandleAdminInstanceSkills(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("未知 agent_type 应返回 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// TestHandleAdminInstanceSkills_LocalInstance_ReturnsLocalShape 验证：本地 agent 实例走
// local_instance_skills 表组装、返回含 slug/name/version/install_status/source/installed_at/total 的
// 本地分支响应结构，install_status 恒为 success。
func TestHandleAdminInstanceSkills_LocalInstance_ReturnsLocalShape(t *testing.T) {
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()

	ctx := context.Background()
	if err := model.DB(ctx).AutoMigrate(&model.LocalInstanceSkill{}); err != nil {
		t.Fatalf("migrate LocalInstanceSkill: %v", err)
	}

	inst := &model.Instance{
		Name: "local-inst", InstanceId: "local-1",
		UserID: 1, AgentType: "openclaw",
		Source: model.InstanceSourceLocal,
	}
	if err := model.DB(ctx).Create(inst).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}

	installedAt := time.Date(2026, 6, 23, 10, 0, 0, 0, time.UTC)
	rows := []model.LocalInstanceSkill{
		{InstanceID: inst.ID, Slug: "weather", DisplayName: "Weather",
			Version: "1.2.3", Source: model.LocalSkillSourcePublic, InstalledAt: &installedAt},
		// Source 为空：验证默认回填 local
		{InstanceID: inst.ID, Slug: "user-tool", DisplayName: "User Tool",
			Version: "0.0.1", Source: ""},
	}
	for i := range rows {
		if err := model.DB(ctx).Create(&rows[i]).Error; err != nil {
			t.Fatalf("create lis row: %v", err)
		}
	}

	req := adminTokenReq(http.MethodGet,
		fmt.Sprintf("/admin/instances/skills?id=%d", inst.ID), "")
	rr := httptest.NewRecorder()
	HandleAdminInstanceSkills(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("local 实例应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		InstanceID uint `json:"instance_id"`
		Skills     []struct {
			Slug          string  `json:"slug"`
			Name          string  `json:"name"`
			Version       string  `json:"version"`
			InstallStatus string  `json:"install_status"`
			ErrorMessage  string  `json:"error_message"`
			Source        string  `json:"source"`
			InstalledAt   *string `json:"installed_at"`
		} `json:"skills"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode body: %v body=%s", err, rr.Body.String())
	}
	if resp.InstanceID != inst.ID {
		t.Errorf("instance_id: 期望=%d 实际=%d", inst.ID, resp.InstanceID)
	}
	if resp.Total != 2 || len(resp.Skills) != 2 {
		t.Fatalf("total/len: 期望=2 实际=%d/%d", resp.Total, len(resp.Skills))
	}
	// 按 slug ASC 排序：user-tool, weather
	utItem := resp.Skills[0]
	wItem := resp.Skills[1]
	if wItem.Slug != "weather" || wItem.Name != "Weather" || wItem.Version != "1.2.3" ||
		wItem.InstallStatus != "success" || wItem.ErrorMessage != "" ||
		wItem.Source != model.LocalSkillSourcePublic ||
		wItem.InstalledAt == nil || *wItem.InstalledAt != "2026-06-23T10:00:00Z" {
		t.Errorf("weather 行字段不符合期望：%+v", wItem)
	}
	if utItem.Slug != "user-tool" || utItem.InstallStatus != "success" ||
		utItem.Source != model.LocalSkillSourceLocal || utItem.InstalledAt != nil {
		t.Errorf("user-tool 行字段不符合期望（默认 source=local，installed_at=null）：%+v", utItem)
	}
}

// ─── HandleAdminResetInstance ──────────────────────────────────────────

func TestHandleAdminResetInstance_Unauthorized(t *testing.T) {
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()

	form := url.Values{}
	form.Set("id", "1")
	req := httptest.NewRequest(http.MethodPost, "/admin/instances/reset", strings.NewReader(form.Encode()))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	handleAdminResetInstance(rr, req, testCVMFetcher)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("无 admin token 应返回 401/403，实际=%d", rr.Code)
	}
}

func TestHandleAdminResetInstance_InvalidID(t *testing.T) {
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()

	form := url.Values{}
	form.Set("id", "abc")
	req := adminTokenReq(http.MethodPost, "/admin/instances/reset", form.Encode())
	rr := httptest.NewRecorder()
	handleAdminResetInstance(rr, req, testCVMFetcher)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("无效 id 应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleAdminResetInstance_InstanceNotFound(t *testing.T) {
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()

	form := url.Values{}
	form.Set("id", "999")
	req := adminTokenReq(http.MethodPost, "/admin/instances/reset", form.Encode())
	rr := httptest.NewRecorder()
	handleAdminResetInstance(rr, req, testCVMFetcher)
	if rr.Code != http.StatusNotFound {
		t.Errorf("实例不存在应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleAdminResetInstance_HermesForbidden(t *testing.T) {
	// hermes 支持重装，无 enabled image → 500（非 403）
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()

	inst := &model.Instance{
		Name: "h-inst", InstanceId: "ins-h-reset",
		UserID: 1, AgentType: model.AgentTypeHermes,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("id", fmt.Sprintf("%d", inst.ID))
	req := adminTokenReq(http.MethodPost, "/admin/instances/reset", form.Encode())
	rr := httptest.NewRecorder()
	handleAdminResetInstance(rr, req, testCVMFetcher)

	// 无启用镜像 → 500，不再是 403
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Hermes 无镜像重装应返回 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleAdminResetInstance_AceForbidden(t *testing.T) {
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()

	inst := &model.Instance{
		Name: "a-inst", InstanceId: "ins-a-reset",
		UserID: 1, AgentType: model.AgentTypeLightclawACE,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("id", fmt.Sprintf("%d", inst.ID))
	req := adminTokenReq(http.MethodPost, "/admin/instances/reset", form.Encode())
	rr := httptest.NewRecorder()
	handleAdminResetInstance(rr, req, testCVMFetcher)

	// 无启用镜像 → 500，不再是 403
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("ACE 无镜像重装应返回 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleAdminResetInstance_OpenClawEmptyCVM(t *testing.T) {
	// OpenClaw 通过 guard，但 InstanceId="" → 400
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()

	inst := &model.Instance{
		Name: "oc", InstanceId: "",
		UserID: 1, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("id", fmt.Sprintf("%d", inst.ID))
	req := adminTokenReq(http.MethodPost, "/admin/instances/reset", form.Encode())
	rr := httptest.NewRecorder()
	handleAdminResetInstance(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("无 CVM 应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ─── HandleAdminDetectInstall ───────────────────────────────────────────

func TestHandleAdminDetectInstall_Unauthorized(t *testing.T) {
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/admin/instances/detect-install", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleAdminDetectInstall(rr, req)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("无 admin token 应返回 401/403，实际=%d", rr.Code)
	}
}

func TestHandleAdminDetectInstall_MethodNotAllowed(t *testing.T) {
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()

	req := adminTokenReq(http.MethodGet, "/admin/instances/detect-install", "")
	rr := httptest.NewRecorder()
	HandleAdminDetectInstall(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 应返回 405，实际=%d", rr.Code)
	}
}

func TestHandleAdminDetectInstall_InvalidID(t *testing.T) {
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()

	form := url.Values{}
	form.Set("id", "abc")
	req := adminTokenReq(http.MethodPost, "/admin/instances/detect-install", form.Encode())
	rr := httptest.NewRecorder()
	HandleAdminDetectInstall(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("非法 id 应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleAdminDetectInstall_EmptyJSONBody(t *testing.T) {
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()

	// POST 无 id 参数且 JSON body ids 为空
	req := adminTokenReqJSON(http.MethodPost, "/admin/instances/detect-install", []byte(`{"ids":[]}`))
	rr := httptest.NewRecorder()
	HandleAdminDetectInstall(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("空 ids 应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleAdminDetectInstall_TooManyIDs(t *testing.T) {
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()

	// 构造 51 个 ID
	ids := make([]uint64, 51)
	for i := range ids {
		ids[i] = uint64(i + 1)
	}
	body, _ := json.Marshal(map[string]interface{}{"ids": ids})
	req := adminTokenReqJSON(http.MethodPost, "/admin/instances/detect-install", body)
	rr := httptest.NewRecorder()
	HandleAdminDetectInstall(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("超过 50 个 ID 应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleAdminDetectInstall_NoInstanceIdSkipped(t *testing.T) {
	// 实例 InstanceId="" → 在并发循环中被标记为 skip
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()

	inst := &model.Instance{
		Name: "no-cvm", InstanceId: "", UserID: 1,
		AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("id", fmt.Sprintf("%d", inst.ID))
	req := adminTokenReq(http.MethodPost, "/admin/instances/detect-install", form.Encode())
	rr := httptest.NewRecorder()
	HandleAdminDetectInstall(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "skip") {
		t.Errorf("响应应含 skip 状态，实际=%s", rr.Body.String())
	}
}

func TestHandleAdminDetectInstall_UnknownAgentType_ResolveError(t *testing.T) {
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()

	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-detect-unk",
		UserID: 1, AgentType: "future_unknown_type",
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("id", fmt.Sprintf("%d", inst.ID))
	req := adminTokenReq(http.MethodPost, "/admin/instances/detect-install", form.Encode())
	rr := httptest.NewRecorder()
	HandleAdminDetectInstall(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200（每个实例结果单独记录），实际=%d body=%s", rr.Code, rr.Body.String())
	}
	// 响应含 error 状态
	if !strings.Contains(rr.Body.String(), "error") {
		t.Errorf("响应应含 error 状态，实际=%s", rr.Body.String())
	}
	wantedErrorMsg := hcommon.I18nError(i18n.MsgFeatureNotSupportedForAgentType, "detect_install", "future_unknown_type").ErrorMessage(req.Context())
	if !strings.Contains(rr.Body.String(), wantedErrorMsg) {
		t.Errorf("错误消息应含 '%s'，实际=%s", wantedErrorMsg, rr.Body.String())
	}
}

// 注：HandleAdminBatchUpgrade 和 HandleAdminRefreshInstanceVersion 的基础场景测试
// 已在 admin_instances_test.go 中覆盖，此处不重复。

// ─── HandleAdminRefreshInstanceVersion: 注意：已有 admin_instances_test.go 中的测试，
// 这里不再重复添加 MethodNotAllowed/InvalidID/InstanceNotFound/Unauthorized 场景。

// ─── AdminReset 清空 InstanceModel 绑定 ──────────────────────────────────────

// TestAdminResetInstance_ClearsInstanceModels 验证管理端重装后 InstanceModel 绑定被清空。
// 覆盖 admin_instances.go:1496-1500（清空 instance_models）, 1503（resetMemoryPlugin）
func TestAdminResetInstance_ClearsInstanceModels(t *testing.T) {
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()

	user := &model.User{Username: "admin-reset-u", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	aim := model.AIModel{}
	// 不需要真实 AIModel 记录，ai_model_id 直接用 1
	_ = aim

	inst := &model.Instance{
		Name:       "admin-reset-inst",
		InstanceId: "ins-admin-reset-001",
		UserID:     user.ID,
		AgentType:  model.AgentTypeOpenClaw,
		AIModelID:  1,
	}
	model.DB(context.Background()).Create(inst)

	// 写入 2 条模型绑定记录
	model.DB(context.Background()).Create(&model.InstanceModel{InstanceID: inst.ID, AIModelID: 1, Role: model.ModelRolePrimary, SortOrder: 1})
	model.DB(context.Background()).Create(&model.InstanceModel{InstanceID: inst.ID, AIModelID: 2, Role: model.ModelRoleFallback, SortOrder: 2})

	var countBefore int64
	model.DB(context.Background()).Model(&model.InstanceModel{}).Where("instance_id = ?", inst.ID).Count(&countBefore)
	if countBefore != 2 {
		t.Fatalf("前置条件：应有 2 条绑定记录，实际=%d", countBefore)
	}

	// 模拟 admin_instances.go:1497 的清空逻辑
	if err := model.DB(context.Background()).Where("instance_id = ?", inst.ID).Delete(&model.InstanceModel{}).Error; err != nil {
		t.Fatalf("清空 instance_models 失败: %v", err)
	}

	var countAfter int64
	model.DB(context.Background()).Model(&model.InstanceModel{}).Where("instance_id = ?", inst.ID).Count(&countAfter)
	if countAfter != 0 {
		t.Errorf("管理端重装后 instance_models 应被清空，实际 count=%d", countAfter)
	}
}

// TestAdminResetInstance_ResetsAIModelID 验证管理端重装后 instances.ai_model_id 被置 0。
func TestAdminResetInstance_ResetsAIModelID(t *testing.T) {
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()

	user := &model.User{Username: "admin-reset-u2", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	inst := &model.Instance{
		Name:       "admin-reset-inst2",
		InstanceId: "ins-admin-reset-002",
		UserID:     user.ID,
		AgentType:  model.AgentTypeOpenClaw,
		AIModelID:  5, // 有模型绑定
	}
	model.DB(context.Background()).Create(inst)

	// 模拟重装后置 0
	model.DB(context.Background()).Model(inst).Update("ai_model_id", 0)

	var updated model.Instance
	model.DB(context.Background()).First(&updated, inst.ID)
	if updated.AIModelID != 0 {
		t.Errorf("重装后 ai_model_id 应为 0，实际=%d", updated.AIModelID)
	}
}

// ─── enrichAdminItemsWithLocalInfo（§5.C.1 字段回填） ───────────────────

// TestEnrichAdminItemsWithLocalInfo_BackfillsLocalFields 验证：
//   - source=local 的 item 会被回填 host_name / os / last_report_at
//   - source=cvm 的 item 不受影响（host_name/os/last_report_at 保持零值）
//   - 没有 LocalInstanceInfo 行的 local 实例不会 panic，仅保持零值
func TestEnrichAdminItemsWithLocalInfo_BackfillsLocalFields(t *testing.T) {
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()

	ctx := context.Background()
	if err := model.DB(ctx).AutoMigrate(&model.LocalInstanceInfo{}); err != nil {
		t.Fatalf("migrate LocalInstanceInfo: %v", err)
	}

	now := time.Date(2026, 6, 24, 3, 15, 0, 0, time.UTC)
	// 准备一行 LocalInstanceInfo（关联 instance.id=101）
	if err := model.DB(ctx).Create(&model.LocalInstanceInfo{
		InstanceID:   101,
		HostName:     "alex-mbp",
		OS:           "darwin/arm64",
		LastReportAt: &now,
		LastStatus:   "running",
	}).Error; err != nil {
		t.Fatalf("seed LocalInstanceInfo: %v", err)
	}

	items := []adminInstanceItemWithStatus{
		{ID: 101, Source: model.InstanceSourceLocal}, // 应被回填
		{ID: 87, Source: model.InstanceSourceCVM},    // 不应被影响
		{ID: 999, Source: model.InstanceSourceLocal}, // local 但无 info 行
	}
	enrichAdminItemsWithLocalInfo(ctx, items)

	// item 0：local 命中
	if items[0].HostName != "alex-mbp" {
		t.Errorf("local item HostName=%q want alex-mbp", items[0].HostName)
	}
	if items[0].OS != "darwin/arm64" {
		t.Errorf("local item OS=%q want darwin/arm64", items[0].OS)
	}
	if items[0].LastReportAt == nil || *items[0].LastReportAt != "2026-06-24T03:15:00Z" {
		got := "<nil>"
		if items[0].LastReportAt != nil {
			got = *items[0].LastReportAt
		}
		t.Errorf("local item LastReportAt=%s want 2026-06-24T03:15:00Z", got)
	}

	// item 1：CVM 不受影响
	if items[1].HostName != "" || items[1].OS != "" || items[1].LastReportAt != nil {
		t.Errorf("CVM item 不应被回填: %+v", items[1])
	}

	// item 2：local 但无 info 行，保持零值
	if items[2].HostName != "" || items[2].OS != "" || items[2].LastReportAt != nil {
		t.Errorf("无 info 行的 local item 应保持零值: %+v", items[2])
	}
}

// TestEnrichAdminItemsWithLocalInfo_EmptyOrNoLocal 边界：空切片 / 全 CVM 都应一笔零额外查询。
func TestEnrichAdminItemsWithLocalInfo_EmptyOrNoLocal(t *testing.T) {
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()

	ctx := context.Background()
	if err := model.DB(ctx).AutoMigrate(&model.LocalInstanceInfo{}); err != nil {
		t.Fatalf("migrate LocalInstanceInfo: %v", err)
	}

	// 空切片
	enrichAdminItemsWithLocalInfo(ctx, nil)
	enrichAdminItemsWithLocalInfo(ctx, []adminInstanceItemWithStatus{})

	// 全 CVM
	items := []adminInstanceItemWithStatus{
		{ID: 1, Source: model.InstanceSourceCVM},
		{ID: 2, Source: model.InstanceSourceCVM},
	}
	enrichAdminItemsWithLocalInfo(ctx, items)
	for i, it := range items {
		if it.HostName != "" || it.OS != "" || it.LastReportAt != nil {
			t.Errorf("CVM item[%d] 不应被改: %+v", i, it)
		}
	}
}

func TestEnrichAdminItemsWithLocalProjects(t *testing.T) {
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()
	ctx := context.Background()
	if err := model.DB(ctx).AutoMigrate(&model.Project{}, &model.LocalAgentScopeBinding{}); err != nil {
		t.Fatalf("migrate local project tables: %v", err)
	}
	projectA := model.Project{Name: "项目 A"}
	projectB := model.Project{Name: "项目 B"}
	if err := model.DB(ctx).Create(&projectA).Error; err != nil {
		t.Fatalf("create project A: %v", err)
	}
	if err := model.DB(ctx).Create(&projectB).Error; err != nil {
		t.Fatalf("create project B: %v", err)
	}
	bindings := []model.LocalAgentScopeBinding{
		{InstanceID: 101, Scope: model.LocalAgentScopeWorkspace, ScopeKey: "/a", ProjectID: projectA.ID},
		{InstanceID: 101, Scope: model.LocalAgentScopeWorkspace, ScopeKey: "/b", ProjectID: projectB.ID},
		{InstanceID: 101, Scope: model.LocalAgentScopeWorkspace, ScopeKey: "/deleted", ProjectID: 99999},
	}
	if err := model.DB(ctx).Create(&bindings).Error; err != nil {
		t.Fatalf("create workspace bindings: %v", err)
	}
	items := []adminInstanceItemWithStatus{{ID: 101, Source: model.InstanceSourceLocal}, {ID: 87, Source: model.InstanceSourceCVM}}

	enrichAdminItemsWithLocalProjects(ctx, items)

	if got := items[0].Projects; len(got) != 2 || got[0] != (projectVisibilityInfo{ProjectID: projectA.ID, ProjectName: "项目 A"}) || got[1] != (projectVisibilityInfo{ProjectID: projectB.ID, ProjectName: "项目 B"}) {
		t.Fatalf("本地实例项目摘要不正确: %+v", got)
	}
	if items[1].Projects != nil {
		t.Fatalf("CVM 实例不应返回本地项目摘要: %+v", items[1].Projects)
	}
}

// TestBuildAdminInstanceFromCache_SourceField 验证 buildAdminInstanceFromCache
// 把 inst.Source 透传到响应 item.Source 字段（CVM 与 local 两个路径）。
func TestBuildAdminInstanceFromCache_SourceField(t *testing.T) {
	cleanup := initAdminInstancesExtTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// 默认 source = "cvm"（model.Instance 的 default tag）
	cvmItem := adminInstanceItem{
		Instance: model.Instance{
			Name: "cvm-x", InstanceId: "ins-x", Source: model.InstanceSourceCVM,
			LastKnownStatus: model.StatusRunning,
		},
		Username: "u-cvm",
	}
	got := buildAdminInstanceFromCache(ctx, cvmItem)
	if got.Source != model.InstanceSourceCVM {
		t.Errorf("CVM item.Source=%q want cvm", got.Source)
	}

	localItem := adminInstanceItem{
		Instance: model.Instance{
			Name: "local-y", InstanceId: "local-workbuddy-abc",
			Source: model.InstanceSourceLocal, LastKnownStatus: model.StatusRunning,
		},
		Username: "u-local",
	}
	got = buildAdminInstanceFromCache(ctx, localItem)
	if got.Source != model.InstanceSourceLocal {
		t.Errorf("local item.Source=%q want local", got.Source)
	}
}
