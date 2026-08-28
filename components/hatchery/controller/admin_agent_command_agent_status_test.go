package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tat "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tat/v20201028"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// withMockTATAgentStatus 临时替换 describeTATAgentStatusFn，返回 cleanup。
func withMockTATAgentStatus(fn func(ctx context.Context, ids []string) ([]*tat.AutomationAgentInfo, error)) func() {
	prev := describeTATAgentStatusFn
	describeTATAgentStatusFn = fn
	return func() { describeTATAgentStatusFn = prev }
}

func TestHandleAgentCommandAgentStatus_MethodNotAllowed(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	makeAdminUser(t, context.Background(), "u1")
	req := adminSessionReq(t, http.MethodGet, "/admin/agent-commands/agent-status", nil, "u1")
	rr := httptest.NewRecorder()
	HandleAgentCommandAgentStatus(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("status=%d, want 405; body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleAgentCommandAgentStatus_EmptyIDs(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	makeAdminUser(t, context.Background(), "u1")

	cases := []struct {
		name string
		body any
	}{
		{"empty_array", map[string]any{"instance_ids": []string{}}},
		{"all_blank", map[string]any{"instance_ids": []string{"", "  "}}},
		{"missing_field", map[string]any{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := adminSessionReq(t, http.MethodPost,
				"/admin/agent-commands/agent-status", c.body, "u1")
			rr := httptest.NewRecorder()
			HandleAgentCommandAgentStatus(rr, req)
			if rr.Code != http.StatusBadRequest ||
				!strings.Contains(rr.Body.String(), i18n.T(req.Context(), i18n.MsgBadRequestMissingParamWithKey, "instance_ids")) {
				t.Errorf("got %d %s, want 400 %s",
					rr.Code, rr.Body.String(), i18n.T(req.Context(), i18n.MsgBadRequestMissingParamWithKey, "instance_ids"))
			}
		})
	}
}

func TestHandleAgentCommandAgentStatus_EmptyIDs_I18n(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	makeAdminUser(t, context.Background(), "u1")

	cases := []struct {
		name string
		body any
	}{
		{"empty_array", map[string]any{"instance_ids": []string{}}},
		{"all_blank", map[string]any{"instance_ids": []string{"", "  "}}},
		{"missing_field", map[string]any{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			wrapped := I18nMiddleware(http.HandlerFunc(HandleAgentCommandAgentStatus))
			req := adminSessionReq(t, http.MethodPost,
				"/admin/agent-commands/agent-status", c.body, "u1")
			req.Header.Set("Accept-Language", "en")
			rr := httptest.NewRecorder()
			wrapped.ServeHTTP(rr, req)
			if rr.Code != http.StatusBadRequest ||
				!strings.Contains(rr.Body.String(), "Missing request parameter: instance_ids") {
				t.Errorf("got %d %s, want 400 %s",
					rr.Code, rr.Body.String(), "Missing request parameter: instance_ids")
			}
		})
	}
}

func TestHandleAgentCommandAgentStatus_TooManyIDs(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	makeAdminUser(t, context.Background(), "u1")
	ids := make([]string, model.AgentDispatchMaxTargets+1)
	for i := range ids {
		ids[i] = fmt.Sprintf("ins-%04d", i)
	}
	req := adminSessionReq(t, http.MethodPost,
		"/admin/agent-commands/agent-status",
		map[string]any{"instance_ids": ids}, "u1")
	rr := httptest.NewRecorder()
	HandleAgentCommandAgentStatus(rr, req)
	if rr.Code != http.StatusBadRequest ||
		!strings.Contains(rr.Body.String(), "too_many_instance_ids") {
		t.Errorf("got %d %s, want 400 too_many_instance_ids",
			rr.Code, rr.Body.String())
	}
}

func TestHandleAgentCommandAgentStatus_TATError(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	makeAdminUser(t, context.Background(), "u1")
	defer withMockTATAgentStatus(func(ctx context.Context, ids []string) ([]*tat.AutomationAgentInfo, error) {
		return nil, hcommon.I18nError(i18n.MsgUpstreamError)
	})()

	req := adminSessionReq(t, http.MethodPost,
		"/admin/agent-commands/agent-status",
		map[string]any{"instance_ids": []string{"ins-x"}}, "u1")
	rr := httptest.NewRecorder()
	HandleAgentCommandAgentStatus(rr, req)
	if rr.Code != http.StatusBadGateway ||
		!strings.Contains(rr.Body.String(), "tat_describe_agent_failed") {
		t.Errorf("got %d %s, want 502 tat_describe_agent_failed",
			rr.Code, rr.Body.String())
	}
}

func TestHandleAgentCommandAgentStatus_HappyPathWithUnknown(t *testing.T) {
	defer initAgentCommandsTestDB(t)()
	makeAdminUser(t, context.Background(), "u1")

	// 入参 3 个：A 在线、B 离线、C 不在 TAT 返回中（应补 Unknown）
	var capturedIDs []string
	defer withMockTATAgentStatus(func(ctx context.Context, ids []string) ([]*tat.AutomationAgentInfo, error) {
		capturedIDs = append([]string(nil), ids...)
		return []*tat.AutomationAgentInfo{
			{
				InstanceId:        common.StringPtr("ins-A"),
				AgentStatus:       common.StringPtr("Online"),
				Version:           common.StringPtr("1.0.0"),
				LastHeartbeatTime: common.StringPtr("2026-05-22T10:00:00Z"),
				Environment:       common.StringPtr("Linux"),
			},
			{
				InstanceId:  common.StringPtr("ins-B"),
				AgentStatus: common.StringPtr("Offline"),
				Environment: common.StringPtr("Linux"),
			},
		}, nil
	})()

	// 故意带空白 + 重复，验证去空 + 去重 + 顺序保留
	req := adminSessionReq(t, http.MethodPost,
		"/admin/agent-commands/agent-status",
		map[string]any{"instance_ids": []string{"ins-A", " ins-A ", "ins-B", "ins-C"}},
		"u1")
	rr := httptest.NewRecorder()
	HandleAgentCommandAgentStatus(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	// 上游应该只收到去重后的 3 个 ID
	if len(capturedIDs) != 3 {
		t.Errorf("describeTATAgentStatusFn 收到 %d 个 ID，want 3 (去重后)；got=%v",
			len(capturedIDs), capturedIDs)
	}

	var resp agentStatusResp
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v body=%s", err, rr.Body.String())
	}
	if len(resp.Agents) != 3 {
		t.Fatalf("agents len=%d, want 3 (A/B/C 各一行); body=%s",
			len(resp.Agents), rr.Body.String())
	}
	want := []struct {
		id, status, version, env string
	}{
		{"ins-A", "Online", "1.0.0", "Linux"},
		{"ins-B", "Offline", "", "Linux"},
		{"ins-C", "Unknown", "", ""},
	}
	for i, w := range want {
		got := resp.Agents[i]
		if got.InstanceID != w.id || got.AgentStatus != w.status ||
			got.Version != w.version || got.Environment != w.env {
			t.Errorf("agents[%d]=%+v, want id=%s status=%s version=%s env=%s",
				i, got, w.id, w.status, w.version, w.env)
		}
	}
}
