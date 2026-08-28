package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// ── guard helper 单测 ─────────────────────────────────────────────────

func TestAgentStatusRejectMessage(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{model.StatusStopped, "实例已关机，请先开机并等待实例恢复运行中后再操作"},
		{model.StatusCreating, "实例当前为创建中，请等待实例恢复运行中后再操作"},
		{model.StatusLoading, "实例当前为加载中，请等待实例恢复运行中后再操作"},
		{model.StatusUpgrading, "实例当前为升级中，请等待实例恢复运行中后再操作"},
		{model.StatusMaintaining, "实例当前为维护中，请等待实例恢复运行中后再操作"},
		{model.StatusPending, "实例当前为待处理，请等待实例恢复运行中后再操作"},
		{model.StatusLoadFailed, "实例当前为加载失败，无法执行该操作"},
		{model.StatusCreateFailed, "实例当前为创建失败，无法执行该操作"},
		{model.StatusUpgradeFailed, "实例当前为升级失败，无法执行该操作"},
		{model.StatusDestroyed, "实例当前为已销毁，无法执行该操作"},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			resp := InstanceStatusResponse{Status: tt.status}
			key, args := agentStatusRejectMessage(resp)
			got := i18n.T(context.Background(), key, args...)
			if got != tt.want {
				t.Errorf("status=%s\n  got:  %s\n  want: %s", tt.status, got, tt.want)
			}
		})
	}
}

func TestRequireActionAllowedForUser(t *testing.T) {
	inst := &model.Instance{InstanceId: "ins-test"}

	tests := []struct {
		name      string
		status    string
		action    string
		wantAllow bool
	}{
		{"running+reboot=allow", model.StatusRunning, "reboot", true},
		{"running+restart_gateway=allow", model.StatusRunning, "restart_gateway", true},
		{"running+terminal=allow", model.StatusRunning, "terminal", true},
		{"running+delete=allow", model.StatusRunning, "delete", true},
		{"stopped+delete=allow", model.StatusStopped, "delete", true},
		{"stopped+reboot=deny", model.StatusStopped, "reboot", false},
		{"creating+reboot=deny", model.StatusCreating, "reboot", false},
		{"load_failed+retry=allow", model.StatusLoadFailed, "retry", true},
		{"load_failed+reboot=deny", model.StatusLoadFailed, "reboot", false},
		{"upgrading+delete=deny", model.StatusUpgrading, "delete", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &mockStatusResolverWithStatus{status: tt.status, label: ""}
			_, err := requireActionAllowedForUser(context.Background(), inst, tt.action, resolver)
			if tt.wantAllow && err != nil {
				t.Errorf("expected allow, got err=%v", err)
			}
			if !tt.wantAllow && err == nil {
				t.Errorf("expected deny, got nil")
			}
		})
	}
}

func TestRequireActionAllowedForAdmin(t *testing.T) {
	inst := &model.Instance{InstanceId: "ins-test"}

	tests := []struct {
		name      string
		status    string
		action    string
		wantAllow bool
	}{
		{"running+stop=allow", model.StatusRunning, "stop", true},
		{"running+reboot=allow", model.StatusRunning, "reboot", true},
		{"running+restart_gateway=allow", model.StatusRunning, "restart_gateway", true},
		{"running+terminal=allow", model.StatusRunning, "terminal", true},
		{"stopped+start=allow", model.StatusStopped, "start", true},
		{"stopped+reinstall=allow", model.StatusStopped, "reinstall", true},
		{"stopped+reboot=deny", model.StatusStopped, "reboot", false},
		{"creating+delete=deny", model.StatusCreating, "delete", false},
		{"load_failed+delete=allow", model.StatusLoadFailed, "delete", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &mockStatusResolverWithStatus{status: tt.status, label: ""}
			_, err := requireActionAllowedForAdmin(context.Background(), inst, tt.action, resolver)
			if tt.wantAllow && err != nil {
				t.Errorf("expected allow, got err=%v", err)
			}
			if !tt.wantAllow && err == nil {
				t.Errorf("expected deny, got nil")
			}
		})
	}
}

func TestRequireInstanceRunning(t *testing.T) {
	inst := &model.Instance{InstanceId: "ins-test"}

	tests := []struct {
		name      string
		status    string
		wantAllow bool
	}{
		{"running=allow", model.StatusRunning, true},
		{"stopped=deny", model.StatusStopped, false},
		{"creating=deny", model.StatusCreating, false},
		{"loading=deny", model.StatusLoading, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &mockStatusResolverWithStatus{status: tt.status, label: ""}
			_, err := requireInstanceRunning(context.Background(), inst, resolver)
			if tt.wantAllow && err != nil {
				t.Errorf("expected allow, got err=%v", err)
			}
			if !tt.wantAllow && err == nil {
				t.Errorf("expected deny, got nil")
			}
		})
	}
}

func TestStatusGuardsRejectResourceAdjustment(t *testing.T) {
	for _, operation := range []string{model.OpAdjustInstanceType, model.OpAdjustSystemDisk} {
		instance := &model.Instance{InstanceId: "ins-test", CurrentOperation: operation}
		resolver := &mockStatusResolverWithStatus{status: model.StatusRunning}
		checks := []struct {
			name string
			run  func() error
		}{
			{"user action", func() error {
				_, err := requireActionAllowedForUser(context.Background(), instance, "delete", resolver)
				return err
			}},
			{"admin action", func() error {
				_, err := requireActionAllowedForAdmin(context.Background(), instance, "delete", resolver)
				return err
			}},
			{"running operation", func() error {
				_, err := requireInstanceRunning(context.Background(), instance, resolver)
				return err
			}},
		}
		for _, check := range checks {
			t.Run(operation+"/"+check.name, func(t *testing.T) {
				if err := check.run(); !errors.Is(err, ErrOperationInProgress) {
					t.Fatalf("error=%v, want ErrOperationInProgress", err)
				}
			})
		}
	}
}

func TestWriteAgentGuardError_409(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Accept", "application/json")

	status := InstanceStatusResponse{Status: model.StatusStopped}
	err := newAgentNotAllowedError(status)
	writeAgentGuardError(rr, req, err)

	if rr.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "已关机") {
		t.Errorf("expected 关机 message, got %s", rr.Body.String())
	}
}

func TestWriteAgentGuardError_500(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Accept", "application/json")

	err := hcommon.I18nRichError(fmt.Errorf("CVM 查询失败: connection refused"), i18n.MsgAgentNotAllowed)
	writeAgentGuardError(rr, req, err)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

// ── handler 拒绝路径集成测试 ──────────────────────────────────────────

func initGuardTestDB(t *testing.T) func() {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	db.AutoMigrate(&model.User{}, &model.Instance{}, &model.InstanceAdjustment{}, &model.SiteConfig{}, &model.AIModel{}, &model.InstanceModel{})
	restore := model.UseDBForTest(db)

	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))

	return func() {
		restore()
		Store = origStore
	}
}

// stoppedResolver 模拟 stopped 状态
var stoppedResolver = &mockStatusResolverWithStatus{status: model.StatusStopped, label: "已关机"}

func guardTestSetup(t *testing.T) (*model.User, *model.Instance, func()) {
	cleanup := initGuardTestDB(t)
	user := &model.User{Username: "guard-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "guard-inst", InstanceId: "ins-guard-test",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)
	return user, inst, cleanup
}

func guardJSONPost(t *testing.T, path string, userID uint, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = "guard-user"

	rr := httptest.NewRecorder()
	session.Save(req, rr)
	for _, c := range rr.Result().Cookies() {
		req.AddCookie(c)
	}
	return req
}

func TestHandleSetEnv_Stopped_409(t *testing.T) {
	_, inst, cleanup := guardTestSetup(t)
	defer cleanup()

	body := fmt.Sprintf(`{"id":%d,"env":{"FOO":"bar"}}`, inst.ID)
	req := httptest.NewRequest(http.MethodPost, "/openclaw/set-env", strings.NewReader(body))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = "guard-user"
	rr := httptest.NewRecorder()
	session.Save(req, rr)
	for _, c := range rr.Result().Cookies() {
		req.AddCookie(c)
	}

	rr = httptest.NewRecorder()
	handleSetEnv(rr, req, stoppedResolver)

	if rr.Code != http.StatusConflict {
		t.Errorf("stopped 状态 set-env 应 409，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if !strings.Contains(resp["error"].(string), "已关机") {
		t.Errorf("文案应包含'已关机'，实际=%s", resp["error"])
	}
}

func TestHandleAddSkill_Stopped_409(t *testing.T) {
	user, inst, cleanup := guardTestSetup(t)
	defer cleanup()

	form := url.Values{"id": {fmt.Sprintf("%d", inst.ID)}, "skill_name": {"test-skill"}}
	req := guardJSONPost(t, "/openclaw/add-skill", user.ID, form.Encode())
	rr := httptest.NewRecorder()
	handleAddSkill(rr, req, stoppedResolver)

	if rr.Code != http.StatusConflict {
		t.Errorf("stopped 状态 add-skill 应 409，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleAddPlugin_Stopped_409(t *testing.T) {
	user, inst, cleanup := guardTestSetup(t)
	defer cleanup()

	form := url.Values{"id": {fmt.Sprintf("%d", inst.ID)}, "plugin": {"test-plugin"}}
	req := guardJSONPost(t, "/openclaw/add-plugin", user.ID, form.Encode())
	rr := httptest.NewRecorder()
	handleAddPlugin(rr, req, stoppedResolver)

	if rr.Code != http.StatusConflict {
		t.Errorf("stopped 状态 add-plugin 应 409，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleSetChannel_Stopped_409(t *testing.T) {
	user, inst, cleanup := guardTestSetup(t)
	defer cleanup()

	form := url.Values{"id": {fmt.Sprintf("%d", inst.ID)}, "channel": {"qqbot"}, "key": {"k"}, "value": {"v"}}
	req := guardJSONPost(t, "/openclaw/set-channel", user.ID, form.Encode())
	rr := httptest.NewRecorder()
	handleSetChannel(rr, req, stoppedResolver)

	if rr.Code != http.StatusConflict {
		t.Errorf("stopped 状态 set-channel 应 409，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleSetModel_Stopped_409(t *testing.T) {
	user, inst, cleanup := guardTestSetup(t)
	defer cleanup()

	form := url.Values{"id": {fmt.Sprintf("%d", inst.ID)}, "ai_model_id": {"1"}}
	req := guardJSONPost(t, "/openclaw/set-model", user.ID, form.Encode())
	rr := httptest.NewRecorder()
	handleSetModel(rr, req, stoppedResolver)

	if rr.Code != http.StatusConflict {
		t.Errorf("stopped 状态 set-model 应 409，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleUpgrade_Stopped_409(t *testing.T) {
	user, inst, cleanup := guardTestSetup(t)
	defer cleanup()

	form := url.Values{"id": {fmt.Sprintf("%d", inst.ID)}}
	req := guardJSONPost(t, "/openclaw/upgrade", user.ID, form.Encode())
	rr := httptest.NewRecorder()
	handleUpgrade(rr, req, stoppedResolver)

	if rr.Code != http.StatusConflict {
		t.Errorf("stopped 状态 upgrade 应 409，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleSetGatewayUi_Stopped_409(t *testing.T) {
	_, inst, cleanup := guardTestSetup(t)
	defer cleanup()
	// gateway 需要 siteConfig
	model.DB(context.Background()).Create(&model.SiteConfig{GatewayUIEnable: true, GatewayUIPort: 8888})

	form := url.Values{"id": {fmt.Sprintf("%d", inst.ID)}}
	req := guardJSONPost(t, "/openclaw/set-gateway-ui", 0, form.Encode())
	rr := httptest.NewRecorder()
	handleSetGatewayUi(rr, req, stoppedResolver)

	if rr.Code != http.StatusConflict {
		t.Errorf("stopped 状态 set-gateway-ui 应 409，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ── 管控端 handler 拒绝路径 ───────────────────────────────────────────

func adminGuardTestSetup(t *testing.T) (*model.Instance, func()) {
	cleanup := initGuardTestDB(t)
	admin := &model.User{Username: "admin", Password: "x", Role: "admin"}
	model.DB(context.Background()).Create(admin)
	inst := &model.Instance{
		Name: "admin-guard-inst", InstanceId: "ins-admin-guard",
		UserID: admin.ID, AgentType: model.AgentTypeOpenClaw,
	}
	model.DB(context.Background()).Create(inst)
	return inst, cleanup
}

func adminJSONPostWithSession(t *testing.T, path, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = "admin"
	rr := httptest.NewRecorder()
	session.Save(req, rr)
	for _, c := range rr.Result().Cookies() {
		req.AddCookie(c)
	}
	return req
}

func TestHandleAdminStartInstance_Running_409(t *testing.T) {
	inst, cleanup := adminGuardTestSetup(t)
	defer cleanup()
	// running 状态下不允许 start（只有 stopped 可以）
	runningResolver := &mockStatusResolverWithStatus{status: model.StatusRunning, label: "运行中"}

	form := url.Values{"id": {fmt.Sprintf("%d", inst.ID)}}
	req := adminJSONPostWithSession(t, "/admin/instances/start", form.Encode())
	rr := httptest.NewRecorder()
	handleAdminStartInstance(rr, req, runningResolver)

	if rr.Code != http.StatusConflict {
		t.Errorf("running 状态 start 应 409，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleAdminStopInstance_Stopped_409(t *testing.T) {
	inst, cleanup := adminGuardTestSetup(t)
	defer cleanup()

	form := url.Values{"id": {fmt.Sprintf("%d", inst.ID)}}
	req := adminJSONPostWithSession(t, "/admin/instances/stop", form.Encode())
	rr := httptest.NewRecorder()
	handleAdminStopInstance(rr, req, stoppedResolver)

	if rr.Code != http.StatusConflict {
		t.Errorf("stopped 状态 stop 应 409，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleAdminRebootInstance_Stopped_409(t *testing.T) {
	inst, cleanup := adminGuardTestSetup(t)
	defer cleanup()

	form := url.Values{"id": {fmt.Sprintf("%d", inst.ID)}}
	req := adminJSONPostWithSession(t, "/admin/instances/reboot", form.Encode())
	rr := httptest.NewRecorder()
	handleAdminRebootInstance(rr, req, stoppedResolver)

	if rr.Code != http.StatusConflict {
		t.Errorf("stopped 状态 reboot 应 409，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleAdminResetInstance_Creating_409(t *testing.T) {
	inst, cleanup := adminGuardTestSetup(t)
	defer cleanup()
	creatingResolver := &mockStatusResolverWithStatus{status: model.StatusCreating, label: "创建中"}

	form := url.Values{"id": {fmt.Sprintf("%d", inst.ID)}}
	req := adminJSONPostWithSession(t, "/admin/instances/reset", form.Encode())
	rr := httptest.NewRecorder()
	handleAdminResetInstance(rr, req, creatingResolver)

	if rr.Code != http.StatusConflict {
		t.Errorf("creating 状态 reset 应 409，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleAdminInstanceTerminal_Stopped_409(t *testing.T) {
	inst, cleanup := adminGuardTestSetup(t)
	defer cleanup()

	form := url.Values{"id": {fmt.Sprintf("%d", inst.ID)}}
	req := adminJSONPostWithSession(t, "/admin/instances/terminal-url", form.Encode())
	rr := httptest.NewRecorder()
	handleAdminInstanceTerminal(rr, req, stoppedResolver)

	if rr.Code != http.StatusConflict {
		t.Errorf("stopped 状态 terminal 应 409，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleAdminDeleteInstance_Creating_409(t *testing.T) {
	inst, cleanup := adminGuardTestSetup(t)
	defer cleanup()
	creatingResolver := &mockStatusResolverWithStatus{status: model.StatusCreating, label: "创建中"}

	form := url.Values{"id": {fmt.Sprintf("%d", inst.ID)}}
	req := adminJSONPostWithSession(t, "/admin/instances/delete", form.Encode())
	rr := httptest.NewRecorder()
	handleAdminDeleteInstance(rr, req, creatingResolver)

	if rr.Code != http.StatusConflict {
		t.Errorf("creating 状态 delete 应 409，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleAdminDeleteInstance_Adjustment_409(t *testing.T) {
	inst, cleanup := adminGuardTestSetup(t)
	defer cleanup()
	if err := model.DB(context.Background()).Model(inst).
		Update("current_operation", model.OpAdjustInstanceType).Error; err != nil {
		t.Fatalf("set adjustment operation: %v", err)
	}
	inst.CurrentOperation = model.OpAdjustInstanceType
	runningResolver := &mockStatusResolverWithStatus{status: model.StatusRunning, label: "运行中"}

	form := url.Values{"id": {fmt.Sprintf("%d", inst.ID)}}
	req := adminJSONPostWithSession(t, "/admin/instances/delete", form.Encode())
	rr := httptest.NewRecorder()
	handleAdminDeleteInstance(rr, req, runningResolver)

	if rr.Code != http.StatusConflict {
		t.Fatalf("adjustment delete 应 409，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	var count int64
	if err := model.DB(context.Background()).Model(&model.Instance{}).
		Where("id = ?", inst.ID).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("adjustment delete removed instance: count=%d err=%v", count, err)
	}
}

// ── openclaw.go reboot/reset/terminal 拒绝路径 ───────────────────────

func TestHandleRebootInstance_Stopped_409(t *testing.T) {
	_, inst, cleanup := guardTestSetup(t)
	defer cleanup()

	form := url.Values{"id": {fmt.Sprintf("%d", inst.ID)}}
	req := guardJSONPost(t, "/openclaw/reboot", 0, form.Encode())
	rr := httptest.NewRecorder()
	handleRebootInstance(rr, req, stoppedResolver)

	if rr.Code != http.StatusConflict {
		t.Errorf("stopped 状态 reboot 应 409，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleResetInstance_Stopped_409(t *testing.T) {
	_, inst, cleanup := guardTestSetup(t)
	defer cleanup()

	form := url.Values{"id": {fmt.Sprintf("%d", inst.ID)}}
	req := guardJSONPost(t, "/openclaw/reset", 0, form.Encode())
	rr := httptest.NewRecorder()
	handleResetInstance(rr, req, stoppedResolver)

	if rr.Code != http.StatusConflict {
		t.Errorf("stopped 状态 reset 应 409，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleInstanceTerminal_Stopped_409(t *testing.T) {
	_, inst, cleanup := guardTestSetup(t)
	defer cleanup()
	model.DB(context.Background()).Create(&model.SiteConfig{TerminalEnabled: true})

	form := url.Values{"id": {fmt.Sprintf("%d", inst.ID)}}
	req := guardJSONPost(t, "/openclaw/terminal-url", 0, form.Encode())
	rr := httptest.NewRecorder()
	handleInstanceTerminal(rr, req, stoppedResolver)

	if rr.Code != http.StatusConflict {
		t.Errorf("stopped 状态 terminal 应 409，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ── browser_vnc 拒绝路径 ─────────────────────────────────────────────

func TestHandleBrowserTakeover_Stopped_409(t *testing.T) {
	_, inst, cleanup := guardTestSetup(t)
	defer cleanup()

	form := url.Values{"id": {fmt.Sprintf("%d", inst.ID)}}
	req := guardJSONPost(t, "/openclaw/browser-takeover", 0, form.Encode())
	rr := httptest.NewRecorder()
	handleBrowserTakeover(rr, req, stoppedResolver)

	if rr.Code != http.StatusConflict {
		t.Errorf("stopped 状态 browser-takeover 应 409，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleBrowserVNCInstall_Stopped_409(t *testing.T) {
	_, inst, cleanup := guardTestSetup(t)
	defer cleanup()

	form := url.Values{"id": {fmt.Sprintf("%d", inst.ID)}}
	req := guardJSONPost(t, "/openclaw/browser-vnc-install", 0, form.Encode())
	rr := httptest.NewRecorder()
	handleBrowserVNCInstall(rr, req, stoppedResolver)

	if rr.Code != http.StatusConflict {
		t.Errorf("stopped 状态 browser-vnc-install 应 409，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ── openclaw_channel 拒绝路径 ────────────────────────────────────────

func TestHandleDelChannel_Stopped_409(t *testing.T) {
	_, inst, cleanup := guardTestSetup(t)
	defer cleanup()

	form := url.Values{"id": {fmt.Sprintf("%d", inst.ID)}, "channel": {"qqbot"}}
	req := guardJSONPost(t, "/openclaw/del-channel", 0, form.Encode())
	rr := httptest.NewRecorder()
	handleDelChannel(rr, req, stoppedResolver)

	if rr.Code != http.StatusConflict {
		t.Errorf("stopped 状态 del-channel 应 409，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleAutoChannel_Stopped_409(t *testing.T) {
	_, inst, cleanup := guardTestSetup(t)
	defer cleanup()

	req := guardJSONPost(t, fmt.Sprintf("/openclaw/auto-channel?id=%d&channel=qqbot", inst.ID), 0, "")
	rr := httptest.NewRecorder()
	handleAutoChannel(rr, req, stoppedResolver)

	if rr.Code != http.StatusConflict {
		t.Errorf("stopped 状态 auto-channel 应 409，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}
