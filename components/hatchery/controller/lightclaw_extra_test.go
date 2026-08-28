package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"hatchery/model"

	sdkerrors "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
)

// ============================================================================
// 本文件聚焦 lightclaw.go 里未覆盖的 handler / 工具函数：
//   1. HandleLightClawAuth              (L173) — 纯 DB + JSON body，完全可测
//   2. HandleLightClawDescribeInvocations      (L232) — 仅测 newTATClient 之前的校验分支
//   3. HandleLightClawDescribeInvocationTasks  (L303) — 同上
//   4. HandleLightClawRunCommand        (L374) — 同上
//   5. isSDKError                       (L472) — 纯类型断言，trivial
// ============================================================================

// ─── HandleLightClawAuth（纯 DB，完整路径可测） ──────────────────────────

func lightclawAuthReq(method string, body []byte) *http.Request {
	req := httptest.NewRequest(method, "/openclaw/lightclaw/auth", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return req
}

func TestHandleLightClawAuth_MethodNotAllowed(t *testing.T) {
	cleanup := initPluginTestDB(t)
	defer cleanup()

	req := lightclawAuthReq(http.MethodGet, nil)
	rr := httptest.NewRecorder()
	HandleLightClawAuth(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 应 405，实际=%d", rr.Code)
	}
}

func TestHandleLightClawAuth_InvalidBody(t *testing.T) {
	cleanup := initPluginTestDB(t)
	defer cleanup()

	req := lightclawAuthReq(http.MethodPost, []byte("{not json"))
	rr := httptest.NewRecorder()
	HandleLightClawAuth(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("非法 body 应 400，实际=%d", rr.Code)
	}
}

func TestHandleLightClawAuth_InvalidProduct(t *testing.T) {
	cleanup := initPluginTestDB(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]string{
		"product":     "wrong-product",
		"accessToken": "any",
	})
	req := lightclawAuthReq(http.MethodPost, body)
	rr := httptest.NewRecorder()
	HandleLightClawAuth(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("product 不匹配应 400，实际=%d", rr.Code)
	}
}

func TestHandleLightClawAuth_MissingAccessToken(t *testing.T) {
	cleanup := initPluginTestDB(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]string{
		"product":     lightClawProductCode,
		"accessToken": "",
	})
	req := lightclawAuthReq(http.MethodPost, body)
	rr := httptest.NewRecorder()
	HandleLightClawAuth(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("空 token 应 401，实际=%d", rr.Code)
	}
}

func TestHandleLightClawAuth_UnknownToken(t *testing.T) {
	cleanup := initPluginTestDB(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]string{
		"product":     lightClawProductCode,
		"accessToken": "non-existent-token",
	})
	req := lightclawAuthReq(http.MethodPost, body)
	rr := httptest.NewRecorder()
	HandleLightClawAuth(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("未知 token 应 401，实际=%d", rr.Code)
	}
}

func TestHandleLightClawAuth_Success(t *testing.T) {
	cleanup := initPluginTestDB(t)
	defer cleanup()

	user := &model.User{Username: "lightclaw-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	token := "tok-abcdef123"
	inst := &model.Instance{
		Name:       "inst",
		InstanceId: "ins-lc-auth",
		UserID:     user.ID,
		AgentType:  model.AgentTypeLightclawACE,
		ProxyToken: &token,
	}
	model.DB(context.Background()).Create(inst)

	body, _ := json.Marshal(map[string]string{
		"product":     lightClawProductCode,
		"accessToken": token,
	})
	req := lightclawAuthReq(http.MethodPost, body)
	rr := httptest.NewRecorder()
	HandleLightClawAuth(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Username   string `json:"username"`
			InstanceID string `json:"instance_id"`
			UserID     string `json:"user_id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("响应解析失败: %v", err)
	}
	if resp.Data.Username != "lightclaw-user" {
		t.Errorf("username 错：%q", resp.Data.Username)
	}
	if resp.Data.InstanceID != "ins-lc-auth" {
		t.Errorf("instance_id 错：%q", resp.Data.InstanceID)
	}
	if resp.Data.UserID == "" {
		t.Errorf("user_id 应被生成，实际为空")
	}
}

func TestHandleLightClawAuth_UserDataMissing(t *testing.T) {
	// 构造 Instance 指向不存在的 UserID，验证 "用户数据异常" 分支
	cleanup := initPluginTestDB(t)
	defer cleanup()

	token := "tok-orphan"
	inst := &model.Instance{
		Name:       "orphan",
		InstanceId: "ins-orphan",
		UserID:     9999, // 不存在
		AgentType:  model.AgentTypeLightclawACE,
		ProxyToken: &token,
	}
	model.DB(context.Background()).Create(inst)

	body, _ := json.Marshal(map[string]string{
		"product":     lightClawProductCode,
		"accessToken": token,
	})
	req := lightclawAuthReq(http.MethodPost, body)
	rr := httptest.NewRecorder()
	HandleLightClawAuth(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("user 不存在应 404，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ─── HandleLightClawDescribeInvocations 等：仅测 newTATClient 之前的分支 ──

func TestHandleLightClawDescribeInvocations_MethodNotAllowed(t *testing.T) {
	cleanup := initPluginTestDB(t)
	defer cleanup()
	req := httptest.NewRequest(http.MethodGet, "/openclaw/lightclaw/describe-invocations", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleLightClawDescribeInvocations(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 应 405，实际=%d", rr.Code)
	}
}

func TestHandleLightClawDescribeInvocations_Unauthorized(t *testing.T) {
	cleanup := initPluginTestDB(t)
	defer cleanup()
	req := httptest.NewRequest(http.MethodPost, "/openclaw/lightclaw/describe-invocations", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleLightClawDescribeInvocations(rr, req)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未登录应 401/403，实际=%d", rr.Code)
	}
}

func TestHandleLightClawDescribeInvocations_InstanceNotFound(t *testing.T) {
	cleanup := initPluginTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := pluginReqWithSession(t, http.MethodPost, "/openclaw/lightclaw/describe-invocations?id=9999", "u1", "")
	rr := httptest.NewRecorder()
	HandleLightClawDescribeInvocations(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("实例不存在应 404，实际=%d", rr.Code)
	}
}

func TestHandleLightClawDescribeInvocations_EmptyInvocationIds(t *testing.T) {
	cleanup := initPluginTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-lc-emptyids",
		UserID: user.ID, AgentType: model.AgentTypeLightclawACE,
	}
	model.DB(context.Background()).Create(inst)

	// body 传空数组
	req := pluginReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/lightclaw/describe-invocations?id=%d", inst.ID), "u1",
		`{"InvocationIds":[]}`)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	HandleLightClawDescribeInvocations(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("InvocationIds 为空应 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// HandleLightClawDescribeInvocationTasks 与 HandleLightClawDescribeInvocations
// 结构一致，覆盖前置校验。
func TestHandleLightClawDescribeInvocationTasks_MethodNotAllowed(t *testing.T) {
	cleanup := initPluginTestDB(t)
	defer cleanup()
	req := httptest.NewRequest(http.MethodGet, "/openclaw/lightclaw/describe-invocation-tasks", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleLightClawDescribeInvocationTasks(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 应 405，实际=%d", rr.Code)
	}
}

func TestHandleLightClawDescribeInvocationTasks_Unauthorized(t *testing.T) {
	cleanup := initPluginTestDB(t)
	defer cleanup()
	req := httptest.NewRequest(http.MethodPost, "/openclaw/lightclaw/describe-invocation-tasks", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleLightClawDescribeInvocationTasks(rr, req)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未登录应 401/403，实际=%d", rr.Code)
	}
}

func TestHandleLightClawDescribeInvocationTasks_InstanceNotFound(t *testing.T) {
	cleanup := initPluginTestDB(t)
	defer cleanup()
	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	req := pluginReqWithSession(t, http.MethodPost, "/openclaw/lightclaw/describe-invocation-tasks?id=9999", "u1", "")
	rr := httptest.NewRecorder()
	HandleLightClawDescribeInvocationTasks(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("实例不存在应 404，实际=%d", rr.Code)
	}
}

func TestHandleLightClawRunCommand_MethodNotAllowed(t *testing.T) {
	cleanup := initPluginTestDB(t)
	defer cleanup()
	req := httptest.NewRequest(http.MethodGet, "/openclaw/lightclaw/run-command", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleLightClawRunCommand(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 应 405，实际=%d", rr.Code)
	}
}

func TestHandleLightClawRunCommand_Unauthorized(t *testing.T) {
	cleanup := initPluginTestDB(t)
	defer cleanup()
	req := httptest.NewRequest(http.MethodPost, "/openclaw/lightclaw/run-command", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleLightClawRunCommand(rr, req)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未登录应 401/403，实际=%d", rr.Code)
	}
}

func TestHandleLightClawRunCommand_InstanceNotFound(t *testing.T) {
	cleanup := initPluginTestDB(t)
	defer cleanup()
	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	req := pluginReqWithSession(t, http.MethodPost, "/openclaw/lightclaw/run-command?id=9999", "u1", "")
	rr := httptest.NewRecorder()
	HandleLightClawRunCommand(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("实例不存在应 404，实际=%d", rr.Code)
	}
}

// ─── isSDKError 纯类型断言 ────────────────────────────────────────────────

func TestIsSDKError_MatchesSDKError(t *testing.T) {
	sdkErr := &sdkerrors.TencentCloudSDKError{
		Code:      "InvalidParameter",
		Message:   "bad param",
		RequestId: "req-1",
	}
	var out *sdkerrors.TencentCloudSDKError
	if !isSDKError(sdkErr, &out) {
		t.Errorf("SDK 错误应返回 true")
	}
	if out == nil || out.Code != "InvalidParameter" {
		t.Errorf("out 未正确赋值，实际=%+v", out)
	}
}

func TestIsSDKError_GenericErrorReturnsFalse(t *testing.T) {
	err := errors.New("plain error")
	var out *sdkerrors.TencentCloudSDKError
	if isSDKError(err, &out) {
		t.Errorf("非 SDK 错误应返回 false")
	}
	if out != nil {
		t.Errorf("out 不应被赋值，实际=%+v", out)
	}
}
