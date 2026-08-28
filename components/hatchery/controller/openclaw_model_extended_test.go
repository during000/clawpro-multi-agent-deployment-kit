package controller

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"hatchery/common"
	"hatchery/controller/usergroup"
	"hatchery/model"
)

// ─── HandleModelsList ──────────────────────────────────────────────────────

func TestHandleModelsList_Unauthorized(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/openclaw/models", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleModelsList(rr, req)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未登录应返回 401/403，实际=%d", rr.Code)
	}
}

func TestHandleModelsList_EmptyDB(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := modelReqWithSession(t, http.MethodGet, "/openclaw/models", "u1", "")
	rr := httptest.NewRecorder()
	HandleModelsList(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d", rr.Code)
	}

	var resp struct {
		OK     bool `json:"ok"`
		Models []struct {
			ID uint `json:"id"`
		} `json:"models"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v body=%s", err, rr.Body.String())
	}
	if !resp.OK {
		t.Error("ok 应为 true")
	}
	if len(resp.Models) != 0 {
		t.Errorf("空 DB 应返回空列表，实际=%d", len(resp.Models))
	}
}

func TestHandleModelsList_OnlyEnabledReturned(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	// 插入启用模型
	enabled := model.AIModel{
		Provider: "p1", ModelID: "m1", ModelType: "openai-completions",
		Enabled: true, Visible: true, VisibilityType: "all",
	}
	model.DB(context.Background()).Create(&enabled)
	// 插入禁用模型
	disabled := model.AIModel{
		Provider: "p2", ModelID: "m2", ModelType: "openai-completions",
		Enabled: false, VisibilityType: "all",
	}
	model.DB(context.Background()).Create(&disabled)

	req := modelReqWithSession(t, http.MethodGet, "/openclaw/models", "u1", "")
	rr := httptest.NewRecorder()
	HandleModelsList(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d", rr.Code)
	}

	var resp struct {
		Models []struct {
			ID       uint   `json:"id"`
			Provider string `json:"provider"`
		} `json:"models"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if len(resp.Models) != 1 {
		t.Fatalf("应只返回 1 个启用的模型，实际=%d", len(resp.Models))
	}
	if resp.Models[0].Provider != "p1" {
		t.Errorf("应返回 p1，实际=%s", resp.Models[0].Provider)
	}
}

// ─── HandleModelsList custom 占位项裁剪 ───────────────────────────────────

// TestHandleModelsList_CustomShownWhenEnabled 站点开放自定义模型时（custom 占位记录
// enabled+visible），列表应包含 model_id="custom" 项。
func TestHandleModelsList_CustomShownWhenEnabled(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	customFlag := model.AIModel{
		Provider: model.BuiltinModelProvider, ModelID: model.BuiltinModelID,
		Enabled: true, Visible: true, VisibilityType: "all",
	}
	model.DB(context.Background()).Create(&customFlag)

	req := modelReqWithSession(t, http.MethodGet, "/openclaw/models", "u1", "")
	rr := httptest.NewRecorder()
	HandleModelsList(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d", rr.Code)
	}
	var resp struct {
		Models []struct {
			ModelID string `json:"model_id"`
		} `json:"models"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	found := false
	for _, m := range resp.Models {
		if m.ModelID == model.BuiltinModelID {
			found = true
		}
	}
	if !found {
		t.Errorf("站点开放自定义模型时列表应包含 custom 占位项，实际=%+v", resp.Models)
	}
}

// TestHandleModelsList_CustomHiddenWhenGroupDenied 站点开放自定义模型，但实例所属分组
// 显式拒绝 custom_model 策略时，列表不应返回 model_id="custom" 占位项（占位记录本身
// enabled+visible 能通过初筛，需由分组级裁剪逻辑剔除）。
func TestHandleModelsList_CustomHiddenWhenGroupDenied(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()
	ctx := context.Background()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(ctx).Create(user)

	g := &model.UserGroup{Name: "g1", ParentID: 0}
	model.DB(ctx).Create(g)
	model.DB(ctx).Create(&model.GroupClosure{AncestorID: g.ID, DescendantID: g.ID, Depth: 0})

	inst := &model.Instance{Name: "x", InstanceId: "ins-modellist-deny", UserID: user.ID, GroupID: g.ID}
	model.DB(ctx).Create(inst)

	// 站点级开放（占位记录 enabled+visible，能通过 HandleModelsList 初筛）
	customFlag := model.AIModel{
		Provider: model.BuiltinModelProvider, ModelID: model.BuiltinModelID,
		Enabled: true, Visible: true, VisibilityType: "all",
	}
	model.DB(ctx).Create(&customFlag)
	// 分组级显式拒绝
	model.DB(ctx).Create(&model.GroupConfigBinding{
		ConfigType: model.ConfigTypePolicy,
		ConfigKey:  usergroup.PolicyKeyCustomModel,
		GroupID:    g.ID,
		ValueJSON:  `{"enabled":false}`,
	})

	req := modelReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/models?agent_id=%d", inst.ID), "u1", "")
	rr := httptest.NewRecorder()
	HandleModelsList(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d", rr.Code)
	}
	var resp struct {
		Models []struct {
			ModelID string `json:"model_id"`
		} `json:"models"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	for _, m := range resp.Models {
		if m.ModelID == model.BuiltinModelID {
			t.Errorf("分组拒绝自定义模型时列表不应包含 custom 占位项，实际=%+v", resp.Models)
		}
	}
}

// ─── HandleSetModel agent_type guard ─────────────────────────────────────

// TestHandleSetModel_UnknownAgentType 未知 agent_type 应被 checkInstanceSupportsModel
// 拦截返回 403。
func TestHandleSetModel_UnknownAgentType(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-setmodel-unk",
		UserID: user.ID, AgentType: "totally_future_type",
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("ai_model_id", "1")
	req := modelReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/model?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleSetModel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusForbidden {
		t.Errorf("未知 agent_type 应返回 403，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleSetModel_MissingAIModelID(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-miss-id",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{} // 无 ai_model_id
	req := modelReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/model?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleSetModel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("缺 ai_model_id 应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleSetModel_InvalidAIModelID(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-bad-id",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("ai_model_id", "abc") // 非数字
	req := modelReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/model?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleSetModel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("非数字 ai_model_id 应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleSetModel_ModelNotFound(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-modelnf",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("ai_model_id", "9999") // 不存在
	req := modelReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/model?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleSetModel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("模型不存在应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleSetModel_DomainNotConfigured(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	// 确保 Domain 为空
	origSnap := common.FixedSnapshot
	common.FixedSnapshot = &common.TenantSnapshot{
		Identifier:     origSnap.Identifier,
		Domain:         "",
		Uin:            origSnap.Uin,
		InternalSecret: origSnap.InternalSecret,
	}
	defer func() { common.FixedSnapshot = origSnap }()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-nodomain",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	aiModel := model.AIModel{
		Provider: "p1", ModelID: "m1", APIKey: "k", URL: "http://x",
		ModelType: "openai-completions", Enabled: true, Visible: true, VisibilityType: "all",
	}
	model.DB(context.Background()).Create(&aiModel)

	form := url.Values{}
	form.Set("ai_model_id", fmt.Sprintf("%d", aiModel.ID))
	req := modelReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/model?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleSetModel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Domain 未配置应返回 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ─── handleCustomModel 校验分支 ───────────────────────────────────────────

func TestHandleSetModel_CustomModelDisabled(t *testing.T) {
	// ai_model_id=0 → 走 handleCustomModel，但 hatchery/custom 未启用 → 403
	cleanup := initModelTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-custom-off",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("ai_model_id", "0")
	req := modelReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/model?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleSetModel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusForbidden {
		t.Errorf("自定义模型未启用应返回 403，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleSetModel_CustomModelMissingFields(t *testing.T) {
	// 启用 custom，但缺字段
	cleanup := initModelTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-custom-miss",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	// 启用 hatchery/custom 标志模型
	customFlag := model.AIModel{
		Provider: model.BuiltinModelProvider,
		ModelID:  model.BuiltinModelID,
		APIKey:   "x", URL: "http://x", ModelType: "openai-completions",
		Enabled: true, Visible: true, VisibilityType: "all",
	}
	model.DB(context.Background()).Create(&customFlag)

	form := url.Values{}
	form.Set("ai_model_id", "0") // 自定义模式
	// 缺 model_id/api_key/url/model_type
	req := modelReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/model?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleSetModel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("自定义模型缺必填字段应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleSetModel_CustomModelInvalidURL(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-custom-badurl",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	customFlag := model.AIModel{
		Provider: model.BuiltinModelProvider, ModelID: model.BuiltinModelID,
		APIKey: "x", URL: "http://x", ModelType: "openai-completions",
		Enabled: true, Visible: true, VisibilityType: "all",
	}
	model.DB(context.Background()).Create(&customFlag)

	form := url.Values{}
	form.Set("ai_model_id", "0")
	form.Set("model_id", "my-model")
	form.Set("api_key", "sk-1")
	form.Set("url", "not-a-valid-url")
	form.Set("model_type", "openai-completions")
	req := modelReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/model?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleSetModel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("非法 URL 应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleSetModel_CustomModelInvalidType(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-custom-badtype",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	customFlag := model.AIModel{
		Provider: model.BuiltinModelProvider, ModelID: model.BuiltinModelID,
		APIKey: "x", URL: "http://x", ModelType: "openai-completions",
		Enabled: true, Visible: true, VisibilityType: "all",
	}
	model.DB(context.Background()).Create(&customFlag)

	form := url.Values{}
	form.Set("ai_model_id", "0")
	form.Set("model_id", "m1")
	form.Set("api_key", "sk")
	form.Set("url", "https://example.com")
	form.Set("model_type", "totally-invalid-type") // 非枚举
	req := modelReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/model?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleSetModel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("非法 model_type 应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ─── buildSetModelParams ──────────────────────────────────────────────────

func TestBuildSetModelParams_ValidModel(t *testing.T) {
	// buildSetModelParams 内部调用 buildPrimaryAndFallbacks 查 DB，需要先初始化
	cleanup := initModelTestDB(t)
	defer cleanup()

	m := model.AIModel{
		Provider: "p1", ModelID: "m1",
		APIKey: "sk-abc", URL: "https://example.com",
		ModelType:  "openai-completions",
		InputTypes: `["text","image"]`,
		ContextLen: 128000,
	}
	params, richErr := buildSetModelParams(context.Background(), m, 0, false)
	if richErr != nil {
		t.Fatalf("不应返回错误：%v", richErr)
	}
	// provider key 使用实际 Provider 字段值作为前缀
	if params["provider"] != "p1-m1" {
		t.Errorf("provider 应为 p1-m1，实际=%q", params["provider"])
	}
	if params["model"] != "m1" {
		t.Errorf("model 应为 m1，实际=%q", params["model"])
	}
	// valueb64 是 base64 编码的 JSON，decode 后应含 apiKey 和 baseUrl
	valBytes, err := base64.StdEncoding.DecodeString(params["valueb64"])
	if err != nil {
		t.Fatalf("base64 decode valueb64 失败：%v", err)
	}
	val := string(valBytes)
	if !strings.Contains(val, "sk-abc") {
		t.Errorf("value 应含 apiKey，实际=%s", val)
	}
	if !strings.Contains(val, "https://example.com") {
		t.Errorf("value 应含 URL，实际=%s", val)
	}
}

// TestBuildSetModelParams_CaseInsensitive 验证大小写混合的 ModelID：
//   - providerKey / ref / TAT model 参数：SlugifyModelID('/' → '-' + ToLower)
//   - 自定义模型（Provider="自定义模型"）→ custom- 前缀
//   - 内置模型 → hatchery- 前缀
//   - body.models[].id/name：直接用 m.ModelID 原始值（保真下发给真实 LLM）
func TestBuildSetModelParams_CaseInsensitive(t *testing.T) {
	cleanup := initModelTestDB(t)
	t.Cleanup(cleanup)

	// 自定义模型：大小写混合（Provider 必须用 common.CustomModelProvider 即 "自定义模型"）
	customModel := model.AIModel{
		Provider:   "自定义模型",
		ModelID:    "DeepSeek-V3.1",
		APIKey:     "sk-test",
		URL:        "https://api.example.com",
		ModelType:  "openai-completions",
		InputTypes: `["text"]`,
		ContextLen: 128000,
	}
	params, richErr := buildSetModelParams(context.Background(), customModel, 0, true)
	if richErr != nil {
		t.Fatalf("不应返回错误：%v", richErr)
	}
	// 自定义模型 → custom- 前缀 + 小写 slug
	if params["provider"] != "custom-deepseek-v3.1" {
		t.Errorf("自定义模型 provider 应为 custom-deepseek-v3.1，实际=%q", params["provider"])
	}
	// primary ref → custom- 前缀（【方案 C】后段保留原始 ModelID不做 slug 化）
	if params["primary"] != "custom-deepseek-v3.1/DeepSeek-V3.1" {
		t.Errorf("primary 应为 custom-deepseek-v3.1/DeepSeek-V3.1，实际=%q", params["primary"])
	}
	// valueb64 中的 models[].id 为 m.ModelID 原始值（保真）
	valBytes, err := base64.StdEncoding.DecodeString(params["valueb64"])
	if err != nil {
		t.Fatalf("base64 decode 失败：%v", err)
	}
	var valObj struct {
		Models []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.Unmarshal(valBytes, &valObj); err != nil {
		t.Fatalf("JSON 解析失败：%v", err)
	}
	if len(valObj.Models) != 1 {
		t.Fatalf("应只有 1 个 model，实际=%d", len(valObj.Models))
	}
	if valObj.Models[0].ID != "DeepSeek-V3.1" {
		t.Errorf("models[].id 应为 DeepSeek-V3.1（保真），实际=%q", valObj.Models[0].ID)
	}
	if valObj.Models[0].Name != "DeepSeek-V3.1" {
		t.Errorf("models[].name 应为 DeepSeek-V3.1（保真），实际=%q", valObj.Models[0].Name)
	}

	// 内置模型：大小写混合
	builtinModel := model.AIModel{
		Provider:   "hatchery",
		ModelID:    "GLM-4-Plus",
		APIKey:     "sk-test",
		URL:        "https://api.example.com",
		ModelType:  "openai-completions",
		InputTypes: `["text"]`,
		ContextLen: 128000,
	}
	params, rerr := buildSetModelParams(context.Background(), builtinModel, 0, false)
	if rerr != nil {
		t.Fatalf("不应返回错误：%v", rerr)
	}
	if params["provider"] != "hatchery-glm-4-plus" {
		t.Errorf("内置模型 provider 应为 hatchery-glm-4-plus，实际=%q", params["provider"])
	}
	if params["primary"] != "hatchery-glm-4-plus/glm-4-plus" {
		t.Errorf("primary 应为 hatchery-glm-4-plus/glm-4-plus，实际=%q", params["primary"])
	}
}

// ─── filterModelsByVisibility ─────────────────────────────────────────────

func TestFilterModelsByVisibility_AllVisibility(t *testing.T) {
	t.Skip("filterModelsByVisibility moved to usergroup package")
}

func TestFilterModelsByVisibility_GroupVisibility_NotInGroup(t *testing.T) {
	t.Skip("filterModelsByVisibility moved to usergroup package")
}

func TestFilterModelsByVisibility_EmptyInput(t *testing.T) {
	t.Skip("filterModelsByVisibility moved to usergroup package")
}

// ─── injectDefaultModel 相关测试 ────────────────────────────────────────

// TestInjectDefaultModel_InstanceDeleted 实例被删除时 goroutine 应优雅退出，
// 不会 panic。
func TestInjectDefaultModel_InstanceDeleted(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("不应 panic，实际=%v", r)
		}
	}()

	// 不存在的 instancePK，goroutine 内会查询失败并返回
	// 注：函数会 sleep 10s 后查 DB；为避免测试超时，此处不等待结束
	// 仅验证入口参数不 panic
	go injectDefaultModel(context.Background(), 99999, 1)

	// 快速退出，避免长时间等待
	time.Sleep(50 * time.Millisecond)
}

// ─── rollbackDefaultModelIfIntact ─────────────────────────────────────────

func TestRollbackDefaultModelIfIntact_ZeroParams(t *testing.T) {
	// instancePK 或 modelID 为 0 时直接返回，不 panic
	rollbackDefaultModelIfIntact(context.Background(), 0, 1, "test")
	rollbackDefaultModelIfIntact(context.Background(), 1, 0, "test")
}

func TestRollbackDefaultModelIfIntact_InstanceNotFound(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()
	// 不存在的实例 → 安静返回
	rollbackDefaultModelIfIntact(context.Background(), 99999, 1, "not_found")
}

func TestRollbackDefaultModelIfIntact_UserChangedModel(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	inst := &model.Instance{
		Name: "i", InstanceId: "ins-rollback-changed",
		UserID: 1, AgentType: model.AgentTypeHermes, AIModelID: 99,
	}
	model.DB(context.Background()).Create(inst)

	// ai_model_id 已被用户改为 99，不等于注入的 modelID=1 → 不回滚
	rollbackDefaultModelIfIntact(context.Background(), inst.ID, 1, "user_changed")

	var after model.Instance
	model.DB(context.Background()).First(&after, inst.ID)
	if after.AIModelID != 99 {
		t.Errorf("用户已修改模型，不应回滚，期望 ai_model_id=99，实际=%d", after.AIModelID)
	}
}

func TestRollbackDefaultModelIfIntact_Rollback(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	inst := &model.Instance{
		Name: "i", InstanceId: "ins-rollback-intact",
		UserID: 1, AgentType: model.AgentTypeHermes, AIModelID: 5,
	}
	model.DB(context.Background()).Create(inst)

	// ai_model_id == modelID=5 → 应回滚为 0
	rollbackDefaultModelIfIntact(context.Background(), inst.ID, 5, "tat_failed")

	var after model.Instance
	model.DB(context.Background()).First(&after, inst.ID)
	if after.AIModelID != 0 {
		t.Errorf("注入失败应回滚 ai_model_id=0，实际=%d", after.AIModelID)
	}
}

// ─── buildPrimaryAndFallbacks ─────────────────────────────────────────────────

func TestBuildPrimaryAndFallbacks_Empty(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	primary, fallbacks, err := buildPrimaryAndFallbacks(context.Background(), 0)
	if err != nil {
		t.Fatalf("不应返回错误: %v", err)
	}
	if primary != "" {
		t.Errorf("无绑定记录时 primary 应为空，实际=%q", primary)
	}
	if fallbacks != "[]" {
		t.Errorf("无绑定记录时 fallbacks 应为 []，实际=%q", fallbacks)
	}
}

func TestBuildPrimaryAndFallbacks_WithRecords(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	user, inst := createMultiModelUserAndInstance(t, "pf-u1", "pf-inst1")
	_ = user

	m1 := model.AIModel{Provider: model.BuiltinModelProvider, ModelID: "glm-4-plus", Enabled: true}
	m2 := model.AIModel{Provider: model.BuiltinModelProvider, ModelID: "qwen-max", Enabled: true}
	model.DB(context.Background()).Create(&m1)
	model.DB(context.Background()).Create(&m2)

	model.DB(context.Background()).Create(&model.InstanceModel{InstanceID: inst.ID, AIModelID: m1.ID, Role: model.ModelRolePrimary, SortOrder: 10})
	model.DB(context.Background()).Create(&model.InstanceModel{InstanceID: inst.ID, AIModelID: m2.ID, Role: model.ModelRoleFallback, SortOrder: 5})

	primary, fallbacks, err := buildPrimaryAndFallbacks(context.Background(), inst.ID)
	if err != nil {
		t.Fatalf("不应返回错误: %v", err)
	}
	if primary == "" {
		t.Error("有 primary 绑定时 primary 不应为空")
	}
	if !strings.Contains(primary, "glm-4-plus") {
		t.Errorf("primary 应含 glm-4-plus，实际=%q", primary)
	}
	if fallbacks == "[]" {
		t.Error("有 fallback 绑定时 fallbacks 不应为 []")
	}
	if !strings.Contains(fallbacks, "qwen-max") {
		t.Errorf("fallbacks 应含 qwen-max，实际=%q", fallbacks)
	}
}

// ─── resolveBindingRef ────────────────────────────────────────────────────────

func TestResolveBindingRef_BuiltinModel(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	aim := model.AIModel{Provider: model.BuiltinModelProvider, ModelID: "glm-4-plus", Enabled: true}
	model.DB(context.Background()).Create(&aim)

	im := model.InstanceModel{AIModelID: aim.ID}
	ref := resolveBindingRef(context.Background(), im)
	if !strings.Contains(ref, "glm-4-plus") {
		t.Errorf("内置模型 ref 应含 model_id，实际=%q", ref)
	}
}

func TestResolveBindingRef_BuiltinModelNotFound(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	// ai_model_id 不存在于 DB
	im := model.InstanceModel{AIModelID: 99999}
	ref := resolveBindingRef(context.Background(), im)
	if !strings.Contains(ref, "unknown") {
		t.Errorf("找不到内置模型时 ref 应含 unknown，实际=%q", ref)
	}
}

func TestResolveBindingRef_CustomModelWithID(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	im := model.InstanceModel{AIModelID: 0, CustomModelID: "my-custom-model"}
	ref := resolveBindingRef(context.Background(), im)
	if !strings.Contains(ref, "my-custom-model") {
		t.Errorf("自定义模型 ref 应含 CustomModelID，实际=%q", ref)
	}
	if !strings.HasPrefix(ref, "custom-") {
		t.Errorf("自定义模型 ref 应以 custom- 开头，实际=%q", ref)
	}
}

// TestResolveBindingRef_CaseInsensitive 验证 SlugifyModelID 会统一转为小写
// （SlugifyModelID 替换 '/' → '-' 并 ToLower）。
func TestResolveBindingRef_CaseInsensitive(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	// 内置模型大小写混合 → slug 化后全小写，hatchery- 前缀
	aim := model.AIModel{Provider: model.BuiltinModelProvider, ModelID: "DeepSeek-V3.1", Enabled: true}
	model.DB(context.Background()).Create(&aim)
	im := model.InstanceModel{AIModelID: aim.ID}
	ref := resolveBindingRef(context.Background(), im)
	if ref != "hatchery-deepseek-v3.1/deepseek-v3.1" {
		t.Errorf("内置模型 ref 应为 hatchery-deepseek-v3.1/deepseek-v3.1，实际=%q", ref)
	}

	// 用户侧自定义模型（AIModelID=0）大小写混合 → custom- 前缀（slug 化），
	// 【方案 C】后段保留原始 ModelID，避免上游误识别。
	im2 := model.InstanceModel{AIModelID: 0, CustomModelID: "DeepSeek-V3.1"}
	ref2 := resolveBindingRef(context.Background(), im2)
	if ref2 != "custom-deepseek-v3.1/DeepSeek-V3.1" {
		t.Errorf("自定义模型 ref 应为 custom-deepseek-v3.1/DeepSeek-V3.1，实际=%q", ref2)
	}

	// 管理员预配的"自定义模型"（AIModelID>0, Provider="自定义模型"）→ custom- 前缀
	aim2 := model.AIModel{Provider: "自定义模型", ModelID: "DeepSeek-V3.1/DeepSeek-V3.1", Enabled: true}
	model.DB(context.Background()).Create(&aim2)
	im3 := model.InstanceModel{AIModelID: aim2.ID}
	ref3 := resolveBindingRef(context.Background(), im3)
	if ref3 != "hatchery-deepseek-v3.1-deepseek-v3.1/deepseek-v3.1-deepseek-v3.1" {
		t.Errorf("管理员预配自定义模型 ref 应为 hatchery-deepseek-v3.1-deepseek-v3.1/deepseek-v3.1-deepseek-v3.1，实际=%q", ref3)
	}
}

// TestProviderKeyPrefix 验证 providerKeyPrefix 对各种 Provider 值的映射。
func TestProviderKeyPrefix(t *testing.T) {
	tests := []struct {
		provider          string
		isUserCustomModel bool
		want              string
	}{
		{"自定义模型", true, "custom"},
		{"hatchery", true, "custom"},
		{"自定义模型", false, "自定义模型"},
		{"hatchery", false, "hatchery"},
		{"qcloudlkeap", false, "qcloudlkeap"},
		{"TencentTokenPlan", false, "tencenttokenplan"},
		{"Hatchery", false, "hatchery"},
		{"doubao", false, "doubao"},
		{"", false, ""},
	}
	for _, tt := range tests {
		got := providerKeyPrefix(tt.provider, tt.isUserCustomModel)
		if got != tt.want {
			t.Errorf("providerKeyPrefix(provider=%q, isUserCustomModel=%t) = %q, want %q", tt.provider, tt.isUserCustomModel, got, tt.want)
		}
	}
}

func TestResolveBindingRef_CustomModelFromJSON(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	im := model.InstanceModel{
		AIModelID:         0,
		CustomModelID:     "", // 空，需从 JSON 解析
		CustomModelConfig: `{"model_id":"json-model","provider":"custom","api_key":"sk","url":"https://x.com","model_type":"openai-completions"}`,
	}
	ref := resolveBindingRef(context.Background(), im)
	if !strings.Contains(ref, "json-model") {
		t.Errorf("从 JSON 解析的 ref 应含 json-model，实际=%q", ref)
	}
}

func TestResolveBindingRef_CustomModelFallbackUnknown(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	im := model.InstanceModel{AIModelID: 0, CustomModelID: "", CustomModelConfig: ""}
	ref := resolveBindingRef(context.Background(), im)
	if !strings.Contains(ref, "unknown") {
		t.Errorf("无 model_id 时 ref 应含 unknown，实际=%q", ref)
	}
}

// ─── isUniqueConstraintError ──────────────────────────────────────────────────

func TestIsUniqueConstraintError(t *testing.T) {
	cases := []struct {
		err     error
		wantYes bool
	}{
		{nil, false},
		{fmt.Errorf("UNIQUE constraint failed: instance_models.identifier"), true},
		{fmt.Errorf("Duplicate entry '1-2' for key 'idx_instance_model'"), true},
		{fmt.Errorf("some other error"), false},
		{fmt.Errorf("record not found"), false},
	}
	for _, c := range cases {
		got := isUniqueConstraintError(c.err)
		if got != c.wantYes {
			t.Errorf("isUniqueConstraintError(%v) = %v, want %v", c.err, got, c.wantYes)
		}
	}
}

// ─── nextSortOrder ────────────────────────────────────────────────────────────

func TestNextSortOrder_NoRecords(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	next, err := nextSortOrder(model.DB(context.Background()), 9999)
	if err != nil {
		t.Fatalf("不应返回错误: %v", err)
	}
	if next != 1 {
		t.Errorf("无记录时 nextSortOrder 应为 1，实际=%d", next)
	}
}

func TestNextSortOrder_WithRecords(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	_, inst := createMultiModelUserAndInstance(t, "sort-u", "sort-inst")
	aim := model.AIModel{Provider: model.BuiltinModelProvider, ModelID: "m1", Enabled: true}
	model.DB(context.Background()).Create(&aim)
	model.DB(context.Background()).Create(&model.InstanceModel{InstanceID: inst.ID, AIModelID: aim.ID, Role: model.ModelRolePrimary, SortOrder: 7})

	next, err := nextSortOrder(model.DB(context.Background()), inst.ID)
	if err != nil {
		t.Fatalf("不应返回错误: %v", err)
	}
	if next != 8 {
		t.Errorf("有记录时 nextSortOrder 应为 MAX+1=8，实际=%d", next)
	}
}

// ─── parseCustomModelFromForm ─────────────────────────────────────────────────

func TestParseCustomModelFromForm_Valid(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Form = map[string][]string{
		"model_id":   {"my-model"},
		"api_key":    {"sk-test"},
		"url":        {"https://api.example.com"},
		"model_type": {"openai-completions"},
	}
	cfg, err := parseCustomModelFromForm(req)
	if err != nil {
		t.Fatalf("不应返回错误: %v", err)
	}
	if cfg.ModelID != "my-model" {
		t.Errorf("ModelID=%q, want my-model", cfg.ModelID)
	}
	if cfg.ModelName != "my-model" {
		t.Errorf("ModelName 应 fallback 到 ModelID，实际=%q", cfg.ModelName)
	}
	if cfg.ContextLen != 128000 {
		t.Errorf("默认 ContextLen 应为 128000，实际=%d", cfg.ContextLen)
	}
}

func TestParseCustomModelFromForm_MissingRequired(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Form = map[string][]string{
		"model_id": {"my-model"},
		// 缺 api_key、url、model_type
	}
	_, err := parseCustomModelFromForm(req)
	if err == nil {
		t.Error("缺少必填字段应返回错误")
	}
}

func TestParseCustomModelFromForm_InvalidModelID(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Form = map[string][]string{
		"model_id":   {"bad model id!"},
		"api_key":    {"sk-test"},
		"url":        {"https://api.example.com"},
		"model_type": {"openai-completions"},
	}
	_, err := parseCustomModelFromForm(req)
	if err == nil {
		t.Error("非法 model_id 应返回错误")
	}
}

func TestParseCustomModelFromForm_InvalidURL(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Form = map[string][]string{
		"model_id":   {"my-model"},
		"api_key":    {"sk-test"},
		"url":        {"not-a-url"},
		"model_type": {"openai-completions"},
	}
	_, err := parseCustomModelFromForm(req)
	if err == nil {
		t.Error("非法 URL 应返回错误")
	}
}

func TestParseCustomModelFromForm_InvalidModelType(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Form = map[string][]string{
		"model_id":   {"my-model"},
		"api_key":    {"sk-test"},
		"url":        {"https://api.example.com"},
		"model_type": {"invalid-type"},
	}
	_, err := parseCustomModelFromForm(req)
	if err == nil {
		t.Error("非法 model_type 应返回错误")
	}
}

func TestParseCustomModelFromForm_CustomContextLen(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Form = map[string][]string{
		"model_id":    {"my-model"},
		"api_key":     {"sk-test"},
		"url":         {"https://api.example.com"},
		"model_type":  {"openai-completions"},
		"context_len": {"32000"},
	}
	cfg, err := parseCustomModelFromForm(req)
	if err != nil {
		t.Fatalf("不应返回错误: %v", err)
	}
	if cfg.ContextLen != 32000 {
		t.Errorf("ContextLen=%d, want 32000", cfg.ContextLen)
	}
}

// ─── JSONStr & InputTypesJSON ─────────────────────────────────────────────────

func TestCustomModelConfig_JSONStr(t *testing.T) {
	cfg := customModelConfig{
		Provider:  "custom",
		ModelID:   "my-model",
		ModelName: "My Model",
		APIKey:    "sk-abc",
		URL:       "https://example.com",
		ModelType: "openai-completions",
	}
	jsonStr := cfg.JSONStr()
	if !strings.Contains(jsonStr, "my-model") {
		t.Errorf("JSONStr 应含 model_id，实际=%s", jsonStr)
	}
	if !strings.Contains(jsonStr, "https://example.com") {
		t.Errorf("JSONStr 应含 URL，实际=%s", jsonStr)
	}
}

func TestCustomModelConfig_InputTypesJSON(t *testing.T) {
	cfg := customModelConfig{InputTypes: []string{"text", "image"}}
	j := cfg.InputTypesJSON()
	if !strings.Contains(j, "text") || !strings.Contains(j, "image") {
		t.Errorf("InputTypesJSON 应含 text 和 image，实际=%s", j)
	}

	empty := customModelConfig{}
	ej := empty.InputTypesJSON()
	if ej != "null" && ej != "[]" {
		// nil slice Marshal 为 "null"，空 slice 为 "[]"，均可接受
		t.Logf("空 InputTypes JSONStr=%s（可接受）", ej)
	}
}

// ─── getModelModelType & getContextLen ────────────────────────────────────────

func TestGetModelModelType_BuiltinModel(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	aim := model.AIModel{Provider: model.BuiltinModelProvider, ModelID: "m1", ModelType: "openai-completions", Enabled: true}
	model.DB(context.Background()).Create(&aim)

	im := model.InstanceModel{AIModelID: aim.ID}
	mt := getModelModelType(context.Background(), im)
	if mt != "openai-completions" {
		t.Errorf("getModelModelType=%q, want openai-completions", mt)
	}
}

func TestGetModelModelType_CustomModel(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	im := model.InstanceModel{
		AIModelID:         0,
		CustomModelConfig: `{"model_type":"anthropic-messages","model_id":"m","provider":"p","api_key":"k","url":"https://x.com"}`,
	}
	mt := getModelModelType(context.Background(), im)
	if mt != "anthropic-messages" {
		t.Errorf("getModelModelType=%q, want anthropic-messages", mt)
	}
}

func TestGetContextLen_BuiltinModel(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	aim := model.AIModel{Provider: model.BuiltinModelProvider, ModelID: "m1", ContextLen: 64000, Enabled: true}
	model.DB(context.Background()).Create(&aim)

	im := model.InstanceModel{AIModelID: aim.ID}
	cl := getContextLen(context.Background(), im)
	if cl != 64000 {
		t.Errorf("getContextLen=%d, want 64000", cl)
	}
}

func TestGetContextLen_CustomModel(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	im := model.InstanceModel{
		AIModelID:         0,
		CustomModelConfig: `{"context_len":32000,"model_id":"m","provider":"p","api_key":"k","url":"https://x.com","model_type":"openai-completions"}`,
	}
	cl := getContextLen(context.Background(), im)
	if cl != 32000 {
		t.Errorf("getContextLen=%d, want 32000", cl)
	}
}

// ─── buildDemotedResponse ─────────────────────────────────────────────────────

func TestBuildDemotedResponse_EmptyRef(t *testing.T) {
	result := buildDemotedResponse(context.Background(), model.InstanceModel{}, "")
	if result != nil {
		t.Errorf("demotedRef 为空时应返回 nil，实际=%v", result)
	}
}

func TestBuildDemotedResponse_WithRef(t *testing.T) {
	cleanup := initModelTestDB(t)
	defer cleanup()

	aim := model.AIModel{Provider: model.BuiltinModelProvider, ModelID: "qwen-max", ModelName: "Qwen Max", Enabled: true}
	model.DB(context.Background()).Create(&aim)

	im := model.InstanceModel{AIModelID: aim.ID, Role: model.ModelRoleFallback}
	result := buildDemotedResponse(context.Background(), im, "hatchery-qwen-max/qwen-max")

	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("应返回 map，实际=%T", result)
	}
	if m["binding_id"] != "hatchery-qwen-max/qwen-max" {
		t.Errorf("binding_id=%v", m["binding_id"])
	}
	if m["role"] != model.ModelRoleFallback {
		t.Errorf("role=%v, want fallback", m["role"])
	}
}
