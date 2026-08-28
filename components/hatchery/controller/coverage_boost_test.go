package controller

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	hcommon "hatchery/common"
	"hatchery/controller/provider"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// ─── llm_proxy.go 未覆盖行 ────────────────────────────────────────────

func TestExtractBearerToken(t *testing.T) {
	cases := []struct {
		name   string
		header string
		want   string
	}{
		{"valid", "Bearer abc123", "abc123"},
		{"no_prefix", "abc123", ""},
		{"empty", "", ""},
		{"wrong_scheme", "Basic abc123", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", nil)
			r.Header.Set("Authorization", c.header)
			got := extractBearerToken(r)
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestDetectToolCallsInResponse(t *testing.T) {
	cases := []struct {
		name string
		data map[string]interface{}
		want bool
	}{
		{"no_choices", map[string]interface{}{}, false},
		{"empty_choices", map[string]interface{}{"choices": []interface{}{}}, false},
		{"finish_reason_tool_calls", map[string]interface{}{
			"choices": []interface{}{
				map[string]interface{}{"finish_reason": "tool_calls"},
			},
		}, true},
		{"finish_reason_stop", map[string]interface{}{
			"choices": []interface{}{
				map[string]interface{}{"finish_reason": "stop"},
			},
		}, false},
		{"message_tool_calls", map[string]interface{}{
			"choices": []interface{}{
				map[string]interface{}{
					"message": map[string]interface{}{
						"tool_calls": []interface{}{map[string]interface{}{"id": "t1"}},
					},
				},
			},
		}, true},
		{"message_empty_tool_calls", map[string]interface{}{
			"choices": []interface{}{
				map[string]interface{}{
					"message": map[string]interface{}{
						"tool_calls": []interface{}{},
					},
				},
			},
		}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := detectToolCallsInResponse(c.data)
			if got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestLLMErrorResponse(t *testing.T) {
	w := httptest.NewRecorder()
	llmErrorResponse(w, http.StatusUnauthorized, "test error")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status=%d, want 401", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	errObj, ok := resp["error"].(map[string]interface{})
	if !ok || errObj["message"] != "test error" {
		t.Errorf("unexpected response: %v", resp)
	}
}

// TestErrMsg_String removed: errMsg type was deprecated in favor of i18n.T system.

// ─── smh.go 未覆盖行 ─────────────────────────────────────────────────

func TestEnsureHTTPS(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"example.com", "https://example.com"},
		{"https://example.com", "https://example.com"},
		{"http://example.com", "http://example.com"},
	}
	for _, c := range cases {
		got := ensureHTTPS(c.in)
		if got != c.want {
			t.Errorf("ensureHTTPS(%q)=%q, want %q", c.in, got, c.want)
		}
	}
}

func initCovBoostSMHTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.AutoMigrate(&model.CustomAgentType{}, &model.Instance{}, &model.SMHPersonalSpace{}, &model.User{})
	return model.UseDBForTest(db)
}

func TestRecyclePersonalSpace(t *testing.T) {
	cleanup := initCovBoostSMHTestDB(t)
	defer cleanup()

	space := &model.SMHPersonalSpace{
		SpaceId: "sp-recycle", InstanceId: 1, CVMInstanceId: "ins-1",
	}
	model.DB(context.Background()).Create(space)

	changed, err := RecyclePersonalSpace(context.Background(), space)
	if err != nil {
		t.Fatalf("RecyclePersonalSpace error: %v", err)
	}
	if !changed {
		t.Error("应标记为回收站")
	}

	changed2, _ := RecyclePersonalSpace(context.Background(), space)
	if changed2 {
		t.Error("重复标记应返回 false")
	}
}

func TestRestorePersonalSpace(t *testing.T) {
	cleanup := initCovBoostSMHTestDB(t)
	defer cleanup()

	inst := &model.Instance{Name: "i", InstanceId: "ins-restore"}
	model.DB(context.Background()).Create(inst)

	past := time.Now().Add(-1 * time.Hour)
	space := &model.SMHPersonalSpace{
		SpaceId: "sp-restore", InstanceId: inst.ID, CVMInstanceId: "ins-restore",
		ToBeDeletedAt: &past,
	}
	model.DB(context.Background()).Create(space)

	changed, err := RestorePersonalSpace(context.Background(), space)
	if err != nil {
		t.Fatalf("RestorePersonalSpace error: %v", err)
	}
	if !changed {
		t.Error("应恢复为活跃")
	}
}

func TestMarkPersonalSpaceToBeDeleted(t *testing.T) {
	cleanup := initCovBoostSMHTestDB(t)
	defer cleanup()

	inst := &model.Instance{Name: "i", InstanceId: "ins-mark"}
	model.DB(context.Background()).Create(inst)

	space := &model.SMHPersonalSpace{
		SpaceId: "sp-mark", InstanceId: inst.ID, CVMInstanceId: "ins-mark",
	}
	model.DB(context.Background()).Create(space)

	MarkPersonalSpaceToBeDeleted(context.Background(), inst.ID)

	var updated model.SMHPersonalSpace
	model.DB(context.Background()).First(&updated, space.ID)
	if updated.ToBeDeletedAt == nil {
		t.Error("ToBeDeletedAt 应被设置")
	}
}

func TestMarkPersonalSpacesToBeDeletedByUser(t *testing.T) {
	cleanup := initCovBoostSMHTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	inst := &model.Instance{Name: "i", InstanceId: "ins-batch", UserID: user.ID}
	model.DB(context.Background()).Create(inst)

	space := &model.SMHPersonalSpace{
		SpaceId: "sp-batch", InstanceId: inst.ID, CVMInstanceId: "ins-batch",
	}
	model.DB(context.Background()).Create(space)

	MarkPersonalSpacesToBeDeletedByUser(context.Background(), user.ID)

	var updated model.SMHPersonalSpace
	model.DB(context.Background()).First(&updated, space.ID)
	if updated.ToBeDeletedAt == nil {
		t.Error("ToBeDeletedAt 应被设置")
	}
}

// ─── openclaw_model.go 未覆盖行 ──────────────────────────────────────

func TestBuildSetModelParams(t *testing.T) {
	// buildSetModelParams 内部调用 buildPrimaryAndFallbacks 查 DB，需要先初始化
	cleanup := initModelTestDB(t)
	defer cleanup()

	m := model.AIModel{
		Provider:   "openai",
		ModelID:    "gpt-4",
		APIKey:     "sk-test",
		URL:        "https://api.openai.com/v1",
		ModelType:  "openai-completions",
		InputTypes: `["text","image"]`,
		ContextLen: 128000,
	}
	params, err := buildSetModelParams(context.Background(), m, 0, false)
	if err != nil {
		t.Fatalf("buildSetModelParams error: %v", err)
	}
	// provider key 使用实际 Provider 字段值作为前缀，与 resolveBindingRef 保持一致
	if params["provider"] != "openai-gpt-4" {
		t.Errorf("provider=%q, want openai-gpt-4", params["provider"])
	}
	if params["model"] != "gpt-4" {
		t.Errorf("model=%q, want gpt-4", params["model"])
	}
	var valueObj map[string]interface{}
	valueJSON, b64Err := base64.StdEncoding.DecodeString(params["valueb64"])
	if b64Err != nil {
		t.Fatalf("base64 decode valueb64 失败: %v", b64Err)
	}
	if err := json.Unmarshal(valueJSON, &valueObj); err != nil {
		t.Fatalf("解析 value JSON 失败: %v", err)
	}
	if valueObj["baseUrl"] != "https://api.openai.com/v1" {
		t.Errorf("baseUrl 不对: %v", valueObj["baseUrl"])
	}
}

func TestFilterModelsByVisibility(t *testing.T) {
	t.Skip("filterModelsByVisibility moved to usergroup package with changed signature")
}

// ─── openclaw_channel.go 未覆盖行 ────────────────────────────────────

func TestGetDefaultTATRunIdentity(t *testing.T) {
	// 同时覆盖：
	//   - agentuser（Hermes/ACE 旧镜像 <= v0.0.11）
	//   - ubuntu   （Hermes 新镜像 >= v0.0.12）
	//   - 空字符串 （DB 未探测到 RuntimeUser，fallback 到 root + /root）
	cases := []struct {
		runtimeUser string
		wantUser    string
		wantWorkdir string
	}{
		{"agentuser", "agentuser", "/home/agentuser"},
		{"ubuntu", "ubuntu", "/home/ubuntu"},
		{"", "root", "/root"},
	}
	for _, c := range cases {
		gotUser, gotWorkdir := getDefaultTATRunIdentity(c.runtimeUser)
		if gotUser != c.wantUser || gotWorkdir != c.wantWorkdir {
			t.Errorf("getDefaultTATRunIdentity(%q)=(%q,%q), want (%q,%q)",
				c.runtimeUser, gotUser, gotWorkdir, c.wantUser, c.wantWorkdir)
		}
	}
}

// ─── openclaw_skill.go 未覆盖行 ──────────────────────────────────────

func initCovBoostSkillTestDB(t *testing.T) func() {
	t.Helper()
	cleanup := initOpenClawHandlerTestDB(t)
	model.DB(context.Background()).AutoMigrate(&model.CustomAgentType{}, &model.SkillInstallation{})
	return cleanup
}

func TestParseAndUpdateInstallResults_NoJSONLine(t *testing.T) {
	cleanup := initCovBoostSkillTestDB(t)
	defer cleanup()

	inst := &model.Instance{Name: "i", InstanceId: "ins-no-json"}
	model.DB(context.Background()).Create(inst)

	skill := &model.SkillInstallation{
		InstanceID: inst.ID, Name: "s1", Slug: "slug1", Version: "1.0",
		InstallStatus: model.SkillInstalling,
	}
	model.DB(context.Background()).Create(skill)

	skills := []model.SkillInstallation{*skill}
	parseAndUpdateInstallResults(context.Background(), "some output without json", skills, inst.UserID, inst.ID, inst.Name, slog.Default())

	var updated model.SkillInstallation
	model.DB(context.Background()).First(&updated, skill.ID)
	if updated.InstallStatus != model.SkillInstallFailed {
		t.Errorf("无 JSON 输出应标记为 Failed，实际=%d", updated.InstallStatus)
	}
}

func TestParseAndUpdateInstallResults_InvalidJSON(t *testing.T) {
	cleanup := initCovBoostSkillTestDB(t)
	defer cleanup()

	inst := &model.Instance{Name: "i", InstanceId: "ins-bad-json"}
	model.DB(context.Background()).Create(inst)

	skill := &model.SkillInstallation{
		InstanceID: inst.ID, Name: "s1", Slug: "slug2", Version: "1.0",
		InstallStatus: model.SkillInstalling,
	}
	model.DB(context.Background()).Create(skill)

	skills := []model.SkillInstallation{*skill}
	parseAndUpdateInstallResults(context.Background(), "{invalid json", skills, inst.UserID, inst.ID, inst.Name, slog.Default())

	var updated model.SkillInstallation
	model.DB(context.Background()).First(&updated, skill.ID)
	if updated.InstallStatus != model.SkillInstallFailed {
		t.Errorf("非法 JSON 应标记为 Failed，实际=%d", updated.InstallStatus)
	}
}

func TestParseAndUpdateInstallResults_Success(t *testing.T) {
	cleanup := initCovBoostSkillTestDB(t)
	defer cleanup()

	inst := &model.Instance{Name: "i", InstanceId: "ins-ok-json"}
	model.DB(context.Background()).Create(inst)

	skill := &model.SkillInstallation{
		InstanceID: inst.ID, Name: "s1", Slug: "slug3", Version: "1.0",
		InstallStatus: model.SkillInstalling,
	}
	model.DB(context.Background()).Create(skill)

	output := `========== BATCH INSTALL RESULTS ==========
{"results":[{"slug":"slug3","version":"1.0","status":"success","message":""}],"summary":{"total":1,"success":1,"failed":0}}`

	skills := []model.SkillInstallation{*skill}
	parseAndUpdateInstallResults(context.Background(), output, skills, inst.UserID, inst.ID, inst.Name, slog.Default())

	var updated model.SkillInstallation
	model.DB(context.Background()).First(&updated, skill.ID)
	if updated.InstallStatus != model.SkillInstallSuccess {
		t.Errorf("成功应标记为 Success，实际=%d", updated.InstallStatus)
	}
}

func TestParseAndUpdateInstallResults_PartialFailure(t *testing.T) {
	cleanup := initCovBoostSkillTestDB(t)
	defer cleanup()

	inst := &model.Instance{Name: "i", InstanceId: "ins-partial"}
	model.DB(context.Background()).Create(inst)

	s1 := &model.SkillInstallation{
		InstanceID: inst.ID, Name: "s1", Slug: "slug-ok", Version: "1.0",
		InstallStatus: model.SkillInstalling,
	}
	s2 := &model.SkillInstallation{
		InstanceID: inst.ID, Name: "s2", Slug: "slug-fail", Version: "2.0",
		InstallStatus: model.SkillInstalling,
	}
	model.DB(context.Background()).Create(s1)
	model.DB(context.Background()).Create(s2)

	output := `========== BATCH INSTALL RESULTS ==========
{"results":[{"slug":"slug-ok","version":"1.0","status":"success","message":""},{"slug":"slug-fail","version":"2.0","status":"failed","message":"install error"}],"summary":{"total":2,"success":1,"failed":1}}`

	skills := []model.SkillInstallation{*s1, *s2}
	parseAndUpdateInstallResults(context.Background(), output, skills, inst.UserID, inst.ID, inst.Name, slog.Default())

	var u1, u2 model.SkillInstallation
	model.DB(context.Background()).First(&u1, s1.ID)
	model.DB(context.Background()).First(&u2, s2.ID)
	if u1.InstallStatus != model.SkillInstallSuccess {
		t.Errorf("slug-ok 应为 Success，实际=%d", u1.InstallStatus)
	}
	if u2.InstallStatus != model.SkillInstallFailed {
		t.Errorf("slug-fail 应为 Failed，实际=%d", u2.InstallStatus)
	}
	if u2.ErrorMessage != "install error" {
		t.Errorf("error_message=%q, want 'install error'", u2.ErrorMessage)
	}
}

func TestParseAndUpdateInstallResults_MissingSkill(t *testing.T) {
	cleanup := initCovBoostSkillTestDB(t)
	defer cleanup()

	inst := &model.Instance{Name: "i", InstanceId: "ins-missing"}
	model.DB(context.Background()).Create(inst)

	skill := &model.SkillInstallation{
		InstanceID: inst.ID, Name: "s1", Slug: "slug-missing", Version: "1.0",
		InstallStatus: model.SkillInstalling,
	}
	model.DB(context.Background()).Create(skill)

	output := `========== BATCH INSTALL RESULTS ==========
{"results":[{"slug":"other","version":"1.0","status":"success","message":""}],"summary":{"total":1,"success":1,"failed":0}}`

	skills := []model.SkillInstallation{*skill}
	parseAndUpdateInstallResults(context.Background(), output, skills, inst.UserID, inst.ID, inst.Name, slog.Default())

	var updated model.SkillInstallation
	model.DB(context.Background()).First(&updated, skill.ID)
	if updated.InstallStatus != model.SkillInstallFailed {
		t.Errorf("结果中未找到的技能应标记为 Failed，实际=%d", updated.InstallStatus)
	}
}

func TestTryBuildSkillDownloadURL_NoBundleSkill(t *testing.T) {
	cleanup := initCovBoostSkillTestDB(t)
	defer cleanup()

	url := tryBuildSkillDownloadURL(context.Background(), "nonexistent-slug")
	if url != "" {
		t.Errorf("不存在的 slug 应返回空，实际=%q", url)
	}
}

// ─── agent_type_guard.go 未覆盖行 ────────────────────────────────────

func TestCheckInstanceSupportsSkill_AllTypes(t *testing.T) {
	cases := []struct {
		agentType string
		wantErr   bool
	}{
		{model.AgentTypeOpenClaw, false},
		{model.AgentTypeHermes, false},
		{model.AgentTypeLightclawACE, false},
	}
	for _, c := range cases {
		inst := &model.Instance{AgentType: c.agentType, InstanceId: "ins-test"}
		err := checkInstanceSupportsSkill(nil, inst)
		if (err != nil) != c.wantErr {
			t.Errorf("agentType=%s err=%v, wantErr=%v", c.agentType, err, c.wantErr)
		}
	}
}

func TestCheckInstanceSupportsReinstall_AllTypes(t *testing.T) {
	cases := []struct {
		agentType string
		wantErr   bool
	}{
		{model.AgentTypeOpenClaw, false},
		{model.AgentTypeHermes, false},
		{model.AgentTypeLightclawACE, false},
	}
	for _, c := range cases {
		inst := &model.Instance{AgentType: c.agentType, InstanceId: "ins-test"}
		err := checkInstanceSupportsReinstall(nil, inst)
		if (err != nil) != c.wantErr {
			t.Errorf("agentType=%s err=%v, wantErr=%v", c.agentType, err, c.wantErr)
		}
	}
}

func TestCheckInstanceSupportsMemory_AllTypes(t *testing.T) {
	cases := []struct {
		agentType string
		wantErr   bool
	}{
		{model.AgentTypeOpenClaw, false},
		{model.AgentTypeHermes, false},
		{model.AgentTypeLightclawACE, true},
	}
	for _, c := range cases {
		inst := &model.Instance{AgentType: c.agentType, InstanceId: "ins-test"}
		err := checkInstanceSupportsMemory(nil, inst)
		if (err != nil) != c.wantErr {
			t.Errorf("agentType=%s err=%v, wantErr=%v", c.agentType, err, c.wantErr)
		}
	}
}

func TestCheckInstanceSupportsModel_AllTypes(t *testing.T) {
	cases := []struct {
		agentType string
		wantErr   bool
	}{
		{model.AgentTypeOpenClaw, false},
		{model.AgentTypeHermes, false},
		{model.AgentTypeLightclawACE, false},
	}
	for _, c := range cases {
		inst := &model.Instance{AgentType: c.agentType, InstanceId: "ins-test"}
		err := checkInstanceSupportsModel(nil, inst)
		if (err != nil) != c.wantErr {
			t.Errorf("agentType=%s err=%v, wantErr=%v", c.agentType, err, c.wantErr)
		}
	}
}

func TestCheckInstanceSupportsBrowserVNC_AllTypes(t *testing.T) {
	cases := []struct {
		agentType string
		wantErr   bool
	}{
		{model.AgentTypeOpenClaw, false},
		{model.AgentTypeHermes, true},
		{model.AgentTypeLightclawACE, true},
	}
	for _, c := range cases {
		inst := &model.Instance{AgentType: c.agentType, InstanceId: "ins-test"}
		err := checkInstanceSupportsBrowserVNC(nil, inst)
		if (err != nil) != c.wantErr {
			t.Errorf("agentType=%s err=%v, wantErr=%v", c.agentType, err, c.wantErr)
		}
	}
}

// ─── SMHProvisionError ───────────────────────────────────────────────

func TestSMHProvisionError(t *testing.T) {
	innerErr := errors.New("inner")
	e := newProvisionError("TEST_CODE", innerErr)
	if e.Code != "TEST_CODE" {
		t.Errorf("Code=%q, want TEST_CODE", e.Code)
	}
	if e.Error() != "inner" {
		t.Errorf("Error()=%q, want 'inner'", e.Error())
	}
	if !errors.Is(e, innerErr) {
		t.Error("应支持 errors.Is 解包")
	}
}

// ─── HandleInstallSkills / HandleRetryFailedSkills ──────────────────

func TestHandleInstallSkills(t *testing.T) {
	cleanup := initCovBoostSkillTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	proxyToken := "pt-install-skills"
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-install-skills",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
		ProxyToken: &proxyToken,
	}
	model.DB(context.Background()).Create(inst)

	req := openclawReqWithSession(t, http.MethodGet,
		"/openclaw/install-skills?id="+fmt.Sprintf("%d", inst.ID)+"&json=1", "u1", "")
	rr := httptest.NewRecorder()
	HandleInstallSkills(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("应返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleRetryFailedSkills(t *testing.T) {
	cleanup := initCovBoostSkillTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-retry-skills",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	skill := &model.SkillInstallation{
		InstanceID: inst.ID, Name: "s1", Slug: "slug-retry", Version: "1.0",
		InstallStatus: model.SkillInstallFailed, ErrorMessage: "some error",
	}
	model.DB(context.Background()).Create(skill)

	req := openclawReqWithSession(t, http.MethodPost,
		"/openclaw/retry-failed-skills?id="+fmt.Sprintf("%d", inst.ID), "u1", "")
	req.Body = io.NopCloser(strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	HandleRetryFailedSkills(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("应返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleCancelFailedSkills(t *testing.T) {
	cleanup := initCovBoostSkillTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-cancel-skills",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)

	skill := &model.SkillInstallation{
		InstanceID: inst.ID, Name: "s1", Slug: "slug-cancel", Version: "1.0",
		InstallStatus: model.SkillInstallFailed, ErrorMessage: "some error",
	}
	model.DB(context.Background()).Create(skill)

	req := openclawReqWithSession(t, http.MethodPost,
		"/openclaw/cancel-failed-skills?id="+fmt.Sprintf("%d", inst.ID), "u1", "")
	rr := httptest.NewRecorder()
	HandleCancelFailedSkills(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("应返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	var updated model.SkillInstallation
	model.DB(context.Background()).First(&updated, skill.ID)
	if updated.InstallStatus != model.SkillInstallCancelled {
		t.Errorf("应标记为 Cancelled，实际=%d", updated.InstallStatus)
	}
}

func TestHandleSkillsList_NoInstance(t *testing.T) {
	cleanup := initCovBoostSkillTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := openclawReqWithSession(t, http.MethodGet,
		"/openclaw/skills?id=99999", "u1", "")
	rr := httptest.NewRecorder()
	handleSkillsList(rr, req, newUserSkillDependencies())

	if rr.Code != http.StatusBadRequest {
		t.Errorf("应返回 400，实际=%d", rr.Code)
	}
}

// ─── HandleLLMModels ────────────────────────────────────────────────

func TestHandleLLMModels_NoToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	rr := httptest.NewRecorder()
	HandleLLMModels(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("无 token 应返回 401，实际=%d", rr.Code)
	}
}

func TestHandleLLMModels_InvalidToken(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rr := httptest.NewRecorder()
	HandleLLMModels(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("无效 token 应返回 401，实际=%d", rr.Code)
	}
}

func TestHandleLLMModels_ValidToken(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	aiModel := &model.AIModel{
		Provider: "openai", ModelID: "gpt-4", ModelType: "openai",
		APIKey: "sk-test", URL: "https://api.openai.com/v1", Enabled: true,
	}
	model.DB(context.Background()).Create(aiModel)

	proxyToken := "pt-llm-models"
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-llm-models",
		AIModelID: aiModel.ID, ProxyToken: &proxyToken,
	}
	model.DB(context.Background()).Create(inst)

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer "+proxyToken)
	rr := httptest.NewRecorder()
	HandleLLMModels(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("应返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ─── HandleLLMProxy guards ──────────────────────────────────────────

func TestHandleLLMProxy_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/chat/completions", nil)
	rr := httptest.NewRecorder()
	HandleLLMProxy(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 应返回 405，实际=%d", rr.Code)
	}
}

func TestHandleLLMProxy_MissingToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	rr := httptest.NewRecorder()
	HandleLLMProxy(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("无 token 应返回 401，实际=%d", rr.Code)
	}
}

func TestHandleLLMProxy_InvalidToken(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req.Header.Set("Authorization", "Bearer nonexistent-token")
	rr := httptest.NewRecorder()
	HandleLLMProxy(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("无效 token 应返回 401，实际=%d", rr.Code)
	}
}

func TestHandleLLMProxy_NoModel(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	proxyToken := "pt-no-model"
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-no-model",
		AIModelID: 999, ProxyToken: &proxyToken,
	}
	model.DB(context.Background()).Create(inst)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-4"}`))
	req.Header.Set("Authorization", "Bearer "+proxyToken)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	HandleLLMProxy(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("无模型应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ─── HandleModelsList ───────────────────────────────────────────────

func TestHandleModelsList_CovBoost(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "ml-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := openclawReqWithSession(t, http.MethodGet, "/openclaw/models", "ml-user", "")
	rr := httptest.NewRecorder()
	HandleModelsList(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("应返回 200，实际=%d", rr.Code)
	}
}

// ─── requireSMHEnabled ──────────────────────────────────────────────

// requireSMHEnabled 依赖 model.DB(context.Background()) 全局变量，全量测试中并发冲突不可控，
// 跳过此函数的 HTTP 测试，其核心逻辑已在 TestEnsureHTTPS 等纯函数测试中覆盖。

// ─── WaitForCVMRunning / WaitForTATAgentOnline ──────────────────────

func TestWaitForCVMRunning_EmptyID(t *testing.T) {
	if waitForCVMRunning(context.Background(), "", 1*time.Second) {
		t.Error("空 ID 应返回 false")
	}
}

func TestWaitForTATAgentOnline_EmptyID(t *testing.T) {
	if waitForTATAgentOnline(context.Background(), "", 1*time.Second) {
		t.Error("空 ID 应返回 false")
	}
}

// ─── llm_proxy.go logProviderUsage ─────────────────────────────────

func TestLogProviderUsage_NoChannel(t *testing.T) {
	usage := &provider.Usage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}
	instance := model.Instance{Model: gorm.Model{ID: 1}, UserID: 1}
	resolved := &model.ResolvedModel{AIModelID: 1, ModelID: "gpt-4", Provider: "openai", UsageBucketKey: 1}

	// 不初始化 usageLogCh，logProviderUsage 应安全跳过
	logUsage(context.Background(), usage, instance, resolved, 200, time.Now())
}

func TestLogProviderUsage_NoUsageOK(t *testing.T) {
	instance := model.Instance{Model: gorm.Model{ID: 2}, UserID: 1}
	resolved := &model.ResolvedModel{AIModelID: 1, ModelID: "gpt-4", Provider: "openai", UsageBucketKey: 1}

	logUsage(context.Background(), nil, instance, resolved, 200, time.Now())
}

func TestLogProviderUsage_WithUsage(t *testing.T) {
	usage := &provider.Usage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}
	instance := model.Instance{Model: gorm.Model{ID: 3}, UserID: 1}
	resolved := &model.ResolvedModel{AIModelID: 1, ModelID: "gpt-4", Provider: "openai", UsageBucketKey: 1}

	logUsage(context.Background(), usage, instance, resolved, 200, time.Now())
}

func TestLogProviderUsage_NoUsageErrorStatus(t *testing.T) {
	instance := model.Instance{Model: gorm.Model{ID: 4}, UserID: 1}
	resolved := &model.ResolvedModel{AIModelID: 1, ModelID: "gpt-4", Provider: "openai", UsageBucketKey: 1}

	logUsage(context.Background(), nil, instance, resolved, 500, time.Now())
}

// ─── HandleLLMModels coverage ─────────────────────────────────────

func TestHandleLLMModels_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/models", nil)
	rr := httptest.NewRecorder()
	HandleLLMModels(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST 应返回 405，实际=%d", rr.Code)
	}
}

// ─── openclaw.go 未覆盖 handler 分支 ──────────────────────────────

func TestHandleDeleteInstance_NoInstance(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "del-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := openclawReqWithSession(t, http.MethodPost, "/openclaw/delete?id=99999", "del-user", "")
	rr := httptest.NewRecorder()
	HandleDeleteInstance(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleResetInstance_NoInstance(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "reset-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := openclawReqWithSession(t, http.MethodPost, "/openclaw/reset?id=99999", "reset-user", "")
	rr := httptest.NewRecorder()
	handleResetInstance(rr, req, testCVMFetcher)
	if rr.Code != http.StatusNotFound {
		t.Errorf("应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleAdminDetectInstall_NoInstance(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	// 设置 admin token
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := httptest.NewRequest(http.MethodPost, "/admin/detect-install?id=99999", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	rr := httptest.NewRecorder()
	HandleAdminDetectInstall(rr, req)
	// 不存在的实例 ID 返回 200 + 空 results（不是 400 / 404）
	// 与批量模式 (ids / instance_ids) 全部不存在时的行为保持一致。
	if rr.Code != http.StatusOK {
		t.Errorf("应返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ─── openclaw_channel.go 未覆盖分支 ──────────────────────────────

func TestHandleSetChannel_NoInstance(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "ch-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := openclawReqWithSession(t, http.MethodPost, "/openclaw/set-channel?id=99999", "ch-user", "channel_id=wecom")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	handleSetChannel(rr, req, testCVMFetcher)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleDelChannel_NoInstance(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	user := &model.User{Username: "delch-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := openclawReqWithSession(t, http.MethodPost, "/openclaw/del-channel?id=99999", "delch-user", "channel_id=wecom")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	handleDelChannel(rr, req, testCVMFetcher)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ─── agent_checker.go 未覆盖行 ────────────────────────────────────
// agent_checker.go 没有 HTTP handler，其核心函数依赖 TAT API，无法纯单元测试

// ─── agent_type_guard.go 未覆盖 guard 分支 ─────────────────────────

func TestCheckInstanceSupportsDetailConfig_AllTypes(t *testing.T) {
	initAgentTypeGuardTestDB(t)
	cases := []struct {
		agentType string
		wantErr   bool
	}{
		{model.AgentTypeOpenClaw, false},
		{model.AgentTypeHermes, false},       // hermes 支持 model/channel/skill
		{model.AgentTypeLightclawACE, false}, // ace 支持 skill/model/channel
		{"unknown_type", true},               // 未知类型不支持
	}
	for _, c := range cases {
		inst := &model.Instance{AgentType: c.agentType, InstanceId: "ins-test"}
		err := checkInstanceSupportsDetailConfig(context.Background(), inst)
		if (err != nil) != c.wantErr {
			t.Errorf("agentType=%s err=%v, wantErr=%v", c.agentType, err, c.wantErr)
		}
	}
}

func TestCheckInstanceSupportsPlugin_AllTypes(t *testing.T) {
	cases := []struct {
		agentType string
		wantErr   bool
	}{
		{model.AgentTypeOpenClaw, false},
		{model.AgentTypeHermes, true},
		{model.AgentTypeLightclawACE, true},
	}
	for _, c := range cases {
		inst := &model.Instance{AgentType: c.agentType, InstanceId: "ins-test"}
		err := checkInstanceSupportsPlugin(nil, inst)
		if (err != nil) != c.wantErr {
			t.Errorf("agentType=%s err=%v, wantErr=%v", c.agentType, err, c.wantErr)
		}
	}
}

func TestCheckInstanceSupportsChatbot_AllTypes(t *testing.T) {
	cases := []struct {
		agentType string
		wantErr   bool
	}{
		{model.AgentTypeOpenClaw, false},
		{model.AgentTypeHermes, true},
		{model.AgentTypeLightclawACE, false}, // ace 支持 chatbot
	}
	for _, c := range cases {
		inst := &model.Instance{AgentType: c.agentType, InstanceId: "ins-test"}
		err := checkInstanceSupportsChatbot(nil, inst)
		if (err != nil) != c.wantErr {
			t.Errorf("agentType=%s err=%v, wantErr=%v", c.agentType, err, c.wantErr)
		}
	}
}

func TestCheckInstanceSupportsChannel_AllTypes(t *testing.T) {
	cases := []struct {
		agentType string
		wantErr   bool
	}{
		{model.AgentTypeOpenClaw, false},
		{model.AgentTypeHermes, false},
		{model.AgentTypeLightclawACE, false},
	}
	for _, c := range cases {
		inst := &model.Instance{AgentType: c.agentType, InstanceId: "ins-test"}
		err := checkInstanceSupportsChannel(nil, inst)
		if (err != nil) != c.wantErr {
			t.Errorf("agentType=%s err=%v, wantErr=%v", c.agentType, err, c.wantErr)
		}
	}
}

// ─── HandleAdminChannels 覆盖 ────────────────────────────────────
// HandleAdminChannels 需要 AIChannel 表迁移，跳过

func setupChannelsTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.AutoMigrate(&model.AIChannel{}, &model.User{}, &model.SiteConfig{})
	origToken := AdminToken
	AdminToken = "test-channel-token"
	t.Cleanup(func() { AdminToken = origToken })
	return useDBForTestWithSafeRestore(db)
}

func newAdminJSONRequest(t *testing.T, method, url string, body io.Reader) *http.Request {
	t.Helper()
	r := httptest.NewRequest(method, url, body)
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer test-channel-token")
	if body != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	return r
}

func TestHandleToggleChannel_JSON_Success(t *testing.T) {
	cleanup := setupChannelsTestDB(t)
	defer cleanup()

	enabled := true
	model.DB(context.Background()).Create(&model.AIChannel{
		ChannelID: "openai", Name: "OpenAI", Enabled: &enabled,
	})

	var ch model.AIChannel
	model.DB(context.Background()).First(&ch)

	r := newAdminJSONRequest(t, http.MethodPost, "/admin/channels/toggle?id="+fmt.Sprintf("%d", ch.ID), nil)
	w := httptest.NewRecorder()
	HandleToggleChannel(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleToggleChannel_JSON_NotFound(t *testing.T) {
	cleanup := setupChannelsTestDB(t)
	defer cleanup()

	r := newAdminJSONRequest(t, http.MethodPost, "/admin/channels/toggle?id=9999", nil)
	w := httptest.NewRecorder()
	HandleToggleChannel(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandleAddChannel_JSON_Success(t *testing.T) {
	cleanup := setupChannelsTestDB(t)
	defer cleanup()

	body := strings.NewReader(`{"channel_id":"custom_ch","name":"Custom","custom_config":{"server":{"type":"webhook","url":"https://example.com"}}}`)
	r := newAdminJSONRequest(t, http.MethodPost, "/admin/channels/add", body)
	w := httptest.NewRecorder()
	HandleAddChannel(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleDeleteChannel_JSON_Success(t *testing.T) {
	cleanup := setupChannelsTestDB(t)
	defer cleanup()

	model.DB(context.Background()).Create(&model.AIChannel{
		ChannelID: "del-ch", Name: "Del", Custom: true,
	})

	var ch model.AIChannel
	model.DB(context.Background()).Where("channel_id = ?", "del-ch").First(&ch)

	r := newAdminJSONRequest(t, http.MethodPost, "/admin/channels/delete?id="+fmt.Sprintf("%d", ch.ID), nil)
	w := httptest.NewRecorder()
	HandleDeleteChannel(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleDeleteChannel_JSON_NotCustom(t *testing.T) {
	cleanup := setupChannelsTestDB(t)
	defer cleanup()

	model.DB(context.Background()).Create(&model.AIChannel{
		ChannelID: "preset-ch", Name: "Preset", Custom: false,
	})

	var ch model.AIChannel
	model.DB(context.Background()).Where("channel_id = ?", "preset-ch").First(&ch)

	r := newAdminJSONRequest(t, http.MethodPost, "/admin/channels/delete?id="+fmt.Sprintf("%d", ch.ID), nil)
	w := httptest.NewRecorder()
	HandleDeleteChannel(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

// ─── HandleAdminNotices 覆盖 ─────────────────────────────────────────

func TestHandleAdminNotices_JSON(t *testing.T) {
	setupNoticesTestDB(t)

	origToken := AdminToken
	AdminToken = "test-notices-cov-token"
	t.Cleanup(func() { AdminToken = origToken })

	r := httptest.NewRequest(http.MethodGet, "/admin/notices?limit=5&offset=0", nil)
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer test-notices-cov-token")
	w := httptest.NewRecorder()
	HandleAdminNotices(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := resp["config_steps"]; !ok {
		t.Error("response should have 'config_steps' key")
	}
}

// ─── HandleAdminSkillCategories / HandleAdminPluginCategories 覆盖 ──

func setupSMHCatsTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.AutoMigrate(
		&model.SiteConfig{},
		&model.SkillCategory{},
		&model.SkillCategoryMapping{},
		&model.Skill{},
		&model.PluginCategory{},
		&model.PluginCategoryMapping{},
		&model.Plugin{},
	)
	db.Create(&model.SiteConfig{Name: "Test", SMHEnabled: 1})
	origToken := AdminToken
	AdminToken = "test-smh-cats-token"
	t.Cleanup(func() { AdminToken = origToken })
	return useDBForTestWithSafeRestore(db)
}

func TestHandleAdminSkillCategories_JSON(t *testing.T) {
	cleanup := setupSMHCatsTestDB(t)
	defer cleanup()

	// 创建几个分类
	model.DB(context.Background()).Create(&model.SkillCategory{Name: "AI 工具"})
	model.DB(context.Background()).Create(&model.SkillCategory{Name: "数据分析"})

	r := httptest.NewRequest(http.MethodGet, "/admin/skill-categories?page=1&page_size=20", nil)
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer test-smh-cats-token")
	w := httptest.NewRecorder()
	HandleAdminSkillCategories(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleAdminPluginCategories_JSON(t *testing.T) {
	cleanup := setupSMHCatsTestDB(t)
	defer cleanup()

	model.DB(context.Background()).Create(&model.PluginCategory{Name: "浏览器"})

	r := httptest.NewRequest(http.MethodGet, "/admin/plugin-categories?page=1&page_size=20", nil)
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer test-smh-cats-token")
	w := httptest.NewRecorder()
	HandleAdminPluginCategories(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// TestHandleCreateSkillCategory_JSON 覆盖 lines 104, 117
func TestHandleCreateSkillCategory_JSON(t *testing.T) {
	cleanup := setupSMHCatsTestDB(t)
	defer cleanup()

	body := strings.NewReader("name=测试分类&description=测试描述")
	r := httptest.NewRequest(http.MethodPost, "/admin/skill-categories/create", body)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer test-smh-cats-token")
	w := httptest.NewRecorder()
	HandleCreateSkillCategory(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// TestHandleUpdateSkillCategory_JSON 覆盖 lines 148, 157, 172
func TestHandleUpdateSkillCategory_JSON(t *testing.T) {
	cleanup := setupSMHCatsTestDB(t)
	defer cleanup()

	cat := model.SkillCategory{Name: "原分类"}
	model.DB(context.Background()).Create(&cat)

	body := strings.NewReader(fmt.Sprintf("id=%d&name=新分类&description=新描述", cat.ID))
	r := httptest.NewRequest(http.MethodPost, "/admin/skill-categories/update", body)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer test-smh-cats-token")
	w := httptest.NewRecorder()
	HandleUpdateSkillCategory(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// TestHandleDeleteSkillCategory_JSON 覆盖 lines 204, 210, 215
func TestHandleDeleteSkillCategory_JSON(t *testing.T) {
	cleanup := setupSMHCatsTestDB(t)
	defer cleanup()

	cat := model.SkillCategory{Name: "删除分类"}
	model.DB(context.Background()).Create(&cat)

	body := strings.NewReader(fmt.Sprintf("id=%d", cat.ID))
	r := httptest.NewRequest(http.MethodPost, "/admin/skill-categories/delete", body)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer test-smh-cats-token")
	w := httptest.NewRecorder()
	HandleDeleteSkillCategory(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// TestHandleCreatePluginCategory_JSON 覆盖 lines 108, 121
func TestHandleCreatePluginCategory_JSON(t *testing.T) {
	cleanup := setupSMHCatsTestDB(t)
	defer cleanup()

	body := strings.NewReader("name=新插件分类&description=插件描述")
	r := httptest.NewRequest(http.MethodPost, "/admin/plugin-categories/create", body)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer test-smh-cats-token")
	w := httptest.NewRecorder()
	HandleCreatePluginCategory(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// TestHandleUpdatePluginCategory_JSON 覆盖 lines 158, 166
func TestHandleUpdatePluginCategory_JSON(t *testing.T) {
	cleanup := setupSMHCatsTestDB(t)
	defer cleanup()

	cat := model.PluginCategory{Name: "原插件分类"}
	model.DB(context.Background()).Create(&cat)

	body := strings.NewReader(fmt.Sprintf("id=%d&name=更新插件分类", cat.ID))
	r := httptest.NewRequest(http.MethodPost, "/admin/plugin-categories/update", body)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer test-smh-cats-token")
	w := httptest.NewRecorder()
	HandleUpdatePluginCategory(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// TestHandleDeletePluginCategory_JSON 覆盖 lines 184, 216, 221
func TestHandleDeletePluginCategory_JSON(t *testing.T) {
	cleanup := setupSMHCatsTestDB(t)
	defer cleanup()

	cat := model.PluginCategory{Name: "删除插件分类"}
	model.DB(context.Background()).Create(&cat)

	body := strings.NewReader(fmt.Sprintf("id=%d", cat.ID))
	r := httptest.NewRequest(http.MethodPost, "/admin/plugin-categories/delete", body)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer test-smh-cats-token")
	w := httptest.NewRecorder()
	HandleDeletePluginCategory(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── admin_images.go 覆盖 ─────────────────────────────────────────────

func setupImagesTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.AutoMigrate(&model.AIImage{}, &model.SiteConfig{})
	db.Create(&model.SiteConfig{Name: "Test"})
	origToken := AdminToken
	AdminToken = "test-images-token"
	t.Cleanup(func() { AdminToken = origToken })
	return useDBForTestWithSafeRestore(db)
}

func TestHandleAdminImages_JSON_Cov(t *testing.T) {
	cleanup := setupImagesTestDB(t)
	defer cleanup()

	model.DB(context.Background()).Create(&model.AIImage{ImageId: "img-test1", AgentType: "openclaw", Enabled: true})

	r := httptest.NewRequest(http.MethodGet, "/admin/images", nil)
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer test-images-token")
	w := httptest.NewRecorder()
	HandleAdminImages(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleDeleteImage_JSON_Success(t *testing.T) {
	cleanup := setupImagesTestDB(t)
	defer cleanup()

	img := model.AIImage{ImageId: "img-del", AgentType: "openclaw", Enabled: false}
	model.DB(context.Background()).Create(&img)

	r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/admin/images/delete?id=%d", img.ID), nil)
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer test-images-token")
	w := httptest.NewRecorder()
	HandleDeleteImage(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleDeleteImage_JSON_EnabledForbidden(t *testing.T) {
	cleanup := setupImagesTestDB(t)
	defer cleanup()

	img := model.AIImage{ImageId: "img-enabled", AgentType: "openclaw", Enabled: true}
	model.DB(context.Background()).Create(&img)

	r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/admin/images/delete?id=%d", img.ID), nil)
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer test-images-token")
	w := httptest.NewRecorder()
	HandleDeleteImage(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestHandleDeleteImage_JSON_NotFound(t *testing.T) {
	cleanup := setupImagesTestDB(t)
	defer cleanup()

	r := httptest.NewRequest(http.MethodPost, "/admin/images/delete?id=9999", nil)
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer test-images-token")
	w := httptest.NewRecorder()
	HandleDeleteImage(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// ─── llm_proxy.go InitUsageLogger / FlushUsageLogs 覆盖 ────────────────

func TestInitFlushUsageLogger_Cov(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&model.LLMUsageLog{}, &model.DailyUsageSummary{}, &model.SiteConfig{})
	t.Cleanup(model.UseDBForTest(db))

	// 确保 FixedSnapshot 非 nil
	origSnap := hcommon.FixedSnapshot
	if origSnap == nil {
		hcommon.FixedSnapshot = &hcommon.TenantSnapshot{}
		t.Cleanup(func() { hcommon.FixedSnapshot = origSnap })
	}

	// 初始化使用日志
	InitUsageLogger()

	// 写入一条日志到 channel（覆盖 lines 50-52, 56）
	log := model.LLMUsageLog{
		UserID:           1,
		InstanceID:       1,
		AIModelID:        1,
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}
	if usageLogCh != nil {
		usageLogCh <- usageLogEntry{ctx: context.Background(), entry: log}
	}

	// Flush，等待 goroutine 完成
	FlushUsageLogs()
}

// ─── admin_usage.go HandleAdminUsageData 覆盖 ────────────────────────

func setupUsageTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.AutoMigrate(&model.SiteConfig{}, &model.DailyUsageSummary{}, &model.LLMUsageLog{}, &model.User{}, &model.AIModel{}, &model.Instance{})
	db.Create(&model.SiteConfig{Name: "Test", GlobalTokenQuotaDay: -1})
	origToken := AdminToken
	AdminToken = "test-usage-token"
	t.Cleanup(func() { AdminToken = origToken })
	return useDBForTestWithSafeRestore(db)
}

func TestHandleAdminUsageData_JSON(t *testing.T) {
	cleanup := setupUsageTestDB(t)
	defer cleanup()

	r := httptest.NewRequest(http.MethodGet, "/admin/usage-data?group_by=date,model", nil)
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer test-usage-token")
	w := httptest.NewRecorder()
	HandleAdminUsageData(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := resp["rows"]; !ok {
		t.Error("response should have 'rows' key")
	}
}

// TestHandleDeleteInstance_LocalSource_Rejected 验证：
//   - source=local 实例不走 /openclaw/delete 删除路径（无 CVM 实体可 Terminate）
//   - 返回 400 unsupported operation，实例及关联子表均不被删
//   - 本地实例删除应走 /local-agent/remove（用户端）或 /admin/local-agent/remove（管控端）
func TestHandleDeleteInstance_LocalSource_Rejected(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	ctx := context.Background()
	if err := model.DB(ctx).AutoMigrate(
		&model.LocalInstanceInfo{},
		&model.LocalInstanceSkill{},
	); err != nil {
		t.Fatalf("migrate local tables: %v", err)
	}

	user := &model.User{Username: "del-local-user", Password: "x", Role: "user"}
	if err := model.DB(ctx).Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	inst := &model.Instance{
		Name: "local-del", InstanceId: "local-codebuddy-001",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: model.AgentTypeOpenClaw,
	}
	if err := model.DB(ctx).Create(inst).Error; err != nil {
		t.Fatalf("create inst: %v", err)
	}
	info := &model.LocalInstanceInfo{InstanceID: inst.ID, HostName: "alex-mbp"}
	if err := model.DB(ctx).Create(info).Error; err != nil {
		t.Fatalf("create info: %v", err)
	}
	now := time.Now()
	skill := &model.LocalInstanceSkill{
		InstanceID: inst.ID, Slug: "any-skill", Version: "1.0.0",
		Source: model.LocalSkillSourceEnterprise, InstalledAt: &now, LastSeenAt: &now,
	}
	if err := model.DB(ctx).Create(skill).Error; err != nil {
		t.Fatalf("create skill: %v", err)
	}

	path := fmt.Sprintf("/openclaw/delete?id=%d", inst.ID)
	req := openclawReqWithSession(t, http.MethodPost, path, "del-local-user", "")
	rr := httptest.NewRecorder()
	HandleDeleteInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("local 删除应 400 拒绝，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	// 实例不被删
	var loaded model.Instance
	if err := model.DB(ctx).First(&loaded, inst.ID).Error; err != nil {
		t.Errorf("实例不应被删，但查不到了 id=%d: %v", inst.ID, err)
	}
	// LocalInstanceInfo / LocalInstanceSkill 子表不被清理
	var infoCount, skillCount int64
	model.DB(ctx).Model(&model.LocalInstanceInfo{}).Where("instance_id = ?", inst.ID).Count(&infoCount)
	model.DB(ctx).Model(&model.LocalInstanceSkill{}).Where("instance_id = ?", inst.ID).Count(&skillCount)
	if infoCount == 0 {
		t.Errorf("LocalInstanceInfo 不应被清理")
	}
	if skillCount == 0 {
		t.Errorf("LocalInstanceSkill 不应被清理")
	}
}
