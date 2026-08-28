package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hatchery/common"
	"hatchery/model"
)

func jsonUnmarshal(b []byte, v any) error { return json.Unmarshal(b, v) }

// multipartBodyHook 构造无文件的 hook 创建 multipart body。
type multipartBody struct {
	buf         *bytes.Buffer
	contentType string
}

func multipartBodyHook(t *testing.T, slug, version, ruleType, event, cmd string) multipartBody {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	writer.WriteField("slug", slug)
	writer.WriteField("version", version)
	writer.WriteField("type", ruleType)
	writer.WriteField("name", slug)
	writer.WriteField("description", "hook test")
	writer.WriteField("event", event)
	writer.WriteField("cmd", cmd)
	writer.Close()
	return multipartBody{buf: &buf, contentType: writer.FormDataContentType()}
}

// ---------------------------------------------------------------------------
// F5: 列表 source 筛选 + 阈值常量
// ---------------------------------------------------------------------------

func TestLocalInstanceOfflineThreshold_Is7Days(t *testing.T) {
	if model.LocalInstanceOfflineThreshold.Hours() != 24*7 {
		t.Fatalf("阈值应为 7 天，实际 %v", model.LocalInstanceOfflineThreshold)
	}
}

func TestHandleAdminInstances_FilterBySource_Local(t *testing.T) {
	setupLocalAgentRemoveTestDB(t)
	userID, _ := seedLocalUserAndInstance(t, "alice", "codebuddy")
	ctx := context.Background()
	// 再建一个普通 CVM 实例用于对比
	cvmInst := model.Instance{
		Identifier:  "test-tenant",
		UserID:      userID,
		Name:        "cvm-1",
		InstanceId:  "ins-cvm-1",
		Source:      model.InstanceSourceCVM,
		AgentType:   "codebuddy",
	}
	model.DB(ctx).Create(&cvmInst)

	// 不带 source：应查到 2 个
	all := doAdminInstances(t, "")
	if len(all) != 2 {
		t.Fatalf("不带 source 应 2 个，实际 %d", len(all))
	}
	// source=local：应只 1 个
	local := doAdminInstances(t, "local")
	if len(local) != 1 {
		t.Fatalf("source=local 应 1 个，实际 %d", len(local))
	}
	if local[0].Source != model.InstanceSourceLocal {
		t.Fatalf("source=local 但返回 source=%q", local[0].Source)
	}
	// source=cvm：应只 1 个
	cvm := doAdminInstances(t, "cvm")
	if len(cvm) != 1 {
		t.Fatalf("source=cvm 应 1 个，实际 %d", len(cvm))
	}
	_ = model.InstanceSourceCVM
}

func doAdminInstances(t *testing.T, source string) []model.Instance {
	t.Helper()
	url := "/api/v1/admin/instances"
	if source != "" {
		url += "?source=" + source
	}
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Cookie", adminCookie(t))
	req = req.WithContext(common.InjectTenant(req.Context(), common.TenantSnapshot{Identifier: "test-tenant"}))
	rr := httptest.NewRecorder()
	HandleAdminInstances(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Instances []model.Instance `json:"instances"`
		Total      int64           `json:"total"`
	}
	if err := jsonUnmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return resp.Instances
}

// ---------------------------------------------------------------------------
// F6: Hook 资源创建（无文件 + event/cmd）+ sync 返回 event/cmd
// ---------------------------------------------------------------------------

func TestHandleAdminCreateRule_Hook_Success(t *testing.T) {
	setupLocalAgentRemoveTestDB(t)
	body := multipartBodyHook(t, "my-hook", "1.0.0", model.EnterpriseRuleTypeHook, model.EnterpriseRuleHookEventSessionStart, "echo hello")
	req := httptest.NewRequest(http.MethodPost, "/admin/rules/create", strings.NewReader(body.buf.String()))
	req.Header.Set("Content-Type", body.contentType)
	req.Header.Set("Cookie", adminCookie(t))
	rr := httptest.NewRecorder()
	HandleCreateRule(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Slug    string `json:"slug"`
		Version string `json:"version"`
	}
	jsonUnmarshal(rr.Body.Bytes(), &resp)
	if resp.Slug != "my-hook" {
		t.Fatalf("slug=%q", resp.Slug)
	}
	// 库里应存 event + cmd
	var rule model.EnterpriseRule
	model.DB(context.Background()).Where("slug = ?", "my-hook").First(&rule)
	if rule.Event != model.EnterpriseRuleHookEventSessionStart {
		t.Fatalf("event=%q", rule.Event)
	}
	if rule.Cmd != "echo hello" {
		t.Fatalf("cmd=%q", rule.Cmd)
	}
	if rule.Type != model.EnterpriseRuleTypeHook {
		t.Fatalf("type=%q", rule.Type)
	}
}

func TestHandleAdminCreateRule_Hook_InvalidEvent(t *testing.T) {
	setupLocalAgentRemoveTestDB(t)
	body := multipartBodyHook(t, "bad-hook", "1.0.0", model.EnterpriseRuleTypeHook, "NotARealEvent", "echo hi")
	req := httptest.NewRequest(http.MethodPost, "/admin/rules/create", strings.NewReader(body.buf.String()))
	req.Header.Set("Content-Type", body.contentType)
	req.Header.Set("Cookie", adminCookie(t))
	rr := httptest.NewRecorder()
	HandleCreateRule(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400 body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleAdminCreateRule_Hook_MissingCmd(t *testing.T) {
	setupLocalAgentRemoveTestDB(t)
	body := multipartBodyHook(t, "nohook", "1.0.0", model.EnterpriseRuleTypeHook, model.EnterpriseRuleHookEventStop, "")
	req := httptest.NewRequest(http.MethodPost, "/admin/rules/create", strings.NewReader(body.buf.String()))
	req.Header.Set("Content-Type", body.contentType)
	req.Header.Set("Cookie", adminCookie(t))
	rr := httptest.NewRecorder()
	HandleCreateRule(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400 body=%s", rr.Code, rr.Body.String())
	}
}

func TestRuleTypeCommandName_Hook(t *testing.T) {
	if got := model.RuleTypeCommandName(model.RuleTaskTypeDistribute, model.EnterpriseRuleTypeHook); got != "install_hook_rule" {
		t.Fatalf("install hook = %q", got)
	}
	if got := model.RuleTypeCommandName(model.RuleTaskTypeUninstall, model.EnterpriseRuleTypeHook); got != "uninstall_hook_rule" {
		t.Fatalf("uninstall hook = %q", got)
	}
}
