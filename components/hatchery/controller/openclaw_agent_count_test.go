package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

func TestParseMultiAgentCheckResult(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    multiAgentCheckResult
		wantErr bool
	}{
		{
			name:  "single agent",
			input: `{"count":1}`,
			want:  multiAgentCheckResult{Count: 1},
		},
		{
			name:  "multiple agents",
			input: `{"count":2}`,
			want:  multiAgentCheckResult{Count: 2},
		},
		{name: "invalid json", input: `{`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMultiAgentCheckResult(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("result = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestHandleAgentCount_SingleAndMultiple(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		wantCnt  float64
		queryKey string
	}{
		{
			name:     "single agent by id",
			output:   `{"count":1}`,
			wantCnt:  1,
			queryKey: "id",
		},
		{
			name:     "multiple agents by instance_id",
			output:   `{"count":2}`,
			wantCnt:  2,
			queryKey: "instance_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := initEnvTestDB(t)
			defer cleanup()

			user := &model.User{Username: "u1", Password: "x", Role: "user"}
			if err := model.DB(context.Background()).Create(user).Error; err != nil {
				t.Fatalf("create user: %v", err)
			}
			inst := &model.Instance{
				Name:       "i1",
				InstanceId: "ins-multi-agent",
				UserID:     user.ID,
				AgentType:  model.AgentTypeOpenClaw,
			}
			if err := model.DB(context.Background()).Create(inst).Error; err != nil {
				t.Fatalf("create instance: %v", err)
			}

			origRunner := openclawMultiAgentScriptRunner
			openclawMultiAgentScriptRunner = func(ctx context.Context, instanceID, scriptName string, timeout uint64, runtimeUser string) (string, error) {
				if instanceID != inst.InstanceId {
					t.Fatalf("instanceID = %q, want %q", instanceID, inst.InstanceId)
				}
				if scriptName != "check_multi_agent.sh" {
					t.Fatalf("scriptName = %q, want check_multi_agent.sh", scriptName)
				}
				return tt.output, nil
			}
			defer func() { openclawMultiAgentScriptRunner = origRunner }()

			path := "/openclaw/agent-count?id=1"
			if tt.queryKey == "instance_id" {
				path = "/openclaw/agent-count?instance_id=" + inst.InstanceId
			} else {
				path = "/openclaw/agent-count?id=1"
			}
			req := envReqWithSession(t, http.MethodGet, path, "u1", "")
			rr := httptest.NewRecorder()

			HandleAgentCount(rr, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if _, ok := resp["has_multi_agent"]; ok {
				t.Fatalf("response should not include has_multi_agent: %v", resp)
			}
			if resp["count"] != tt.wantCnt {
				t.Fatalf("count = %v, want %v", resp["count"], tt.wantCnt)
			}
			if _, ok := resp["instance_id"]; ok {
				t.Fatalf("response should not include instance_id: %v", resp)
			}
		})
	}
}

func TestHandleAgentCount_ErrorsAndFallback(t *testing.T) {
	tests := []struct {
		name      string
		method    string
		agentType string
		output    string
		runErr    *hcommon.RichError
		wantCode  int
		wantText  string
		wantCount float64
	}{
		{name: "method not allowed", method: http.MethodPost, agentType: model.AgentTypeOpenClaw, wantCode: http.StatusMethodNotAllowed, wantText: "请求方法不允许"},
		{name: "unsupported agent type fallback", method: http.MethodGet, agentType: model.AgentTypeHermes, wantCode: http.StatusOK, wantCount: 1},
		{name: "runner failed", method: http.MethodGet, agentType: model.AgentTypeOpenClaw, runErr: hcommon.I18nError(i18n.MsgTATFailed), wantCode: http.StatusInternalServerError, wantText: "multi-agent 查询失败"},
		{name: "invalid output", method: http.MethodGet, agentType: model.AgentTypeOpenClaw, output: `{`, wantCode: http.StatusInternalServerError, wantText: "multi-agent 查询结果解析失败"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := initEnvTestDB(t)
			defer cleanup()

			user := &model.User{Username: "u1", Password: "x", Role: "user"}
			if err := model.DB(context.Background()).Create(user).Error; err != nil {
				t.Fatalf("create user: %v", err)
			}
			inst := &model.Instance{Name: "i1", InstanceId: "ins-error", UserID: user.ID, AgentType: tt.agentType}
			if err := model.DB(context.Background()).Create(inst).Error; err != nil {
				t.Fatalf("create instance: %v", err)
			}

			origRunner := openclawMultiAgentScriptRunner
			openclawMultiAgentScriptRunner = func(ctx context.Context, instanceID, scriptName string, timeout uint64, runtimeUser string) (string, error) {
				if tt.runErr != nil {
					return "", tt.runErr
				}
				return tt.output, nil
			}
			defer func() { openclawMultiAgentScriptRunner = origRunner }()

			req := envReqWithSession(t, tt.method, "/openclaw/agent-count?id=1", "u1", "")
			if tt.method == http.MethodPost {
				req = httptest.NewRequest(tt.method, "/openclaw/agent-count?id=1", strings.NewReader(""))
				req.Header.Set("Accept", "application/json")
			}
			rr := httptest.NewRecorder()

			HandleAgentCount(rr, req)
			if rr.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d body=%s", rr.Code, tt.wantCode, rr.Body.String())
			}
			if tt.wantCode == http.StatusOK {
				var resp map[string]interface{}
				if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
					t.Fatalf("decode response: %v", err)
				}
				if resp["count"] != tt.wantCount {
					t.Fatalf("count = %v, want %v", resp["count"], tt.wantCount)
				}
				if _, ok := resp["has_multi_agent"]; ok {
					t.Fatalf("response should not include has_multi_agent: %v", resp)
				}
				return
			}
			if !strings.Contains(rr.Body.String(), tt.wantText) {
				t.Fatalf("response %q should contain %q", rr.Body.String(), tt.wantText)
			}
		})
	}
}
