package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// 本文件覆盖 HandleBrowserVNCCheck / handleBrowserVNCInstall 两个 handler 内部
// 内联的 runScript 闭包（browser_vnc.go 中的成功/失败两条 return 分支）。
//
// 这两个闭包封装 agentScriptRunner（默认绑定真实 RunScript）。真实 RunScript 在单测
// 环境下必然失败（无凭据 / 无在线 Agent），因此用 withAgentScriptRunner 注入桩：
//   - 成功桩：闭包走 `return output, nil`，handler 解析 JSON 后返回 200；
//   - 失败桩：闭包走 `return output, err`，handler 返回 500。

// newBrowserVNCOpenClawInstance 创建一个可通过 BrowserVNC 校验的 OpenClaw 实例，
// 并打开站点级 BrowserVNC 开关。
func newBrowserVNCOpenClawInstance(t *testing.T, cvmID string) (*model.User, *model.Instance) {
	t.Helper()
	ctx := context.Background()
	// 站点级开关：vncEnabled = ResolvePolicyBoolForGroup(..., GroupID=0, fallback=BrowserVNCEnable)
	if err := model.DB(ctx).Create(&model.SiteConfig{BrowserVNCEnable: true}).Error; err != nil {
		t.Fatalf("创建 SiteConfig 失败: %v", err)
	}
	user := &model.User{Username: "vnc-user", Password: "x", Role: "user"}
	model.DB(ctx).Create(user)
	inst := &model.Instance{
		Name:        "oc-vnc",
		InstanceId:  cvmID,
		UserID:      user.ID,
		AgentType:   model.AgentTypeOpenClaw,
		RuntimeUser: "root",
	}
	model.DB(ctx).Create(inst)
	return user, inst
}

// ─── HandleBrowserVNCCheck: runScript 闭包 ───

func TestHandleBrowserVNCCheck_RunScriptSuccess(t *testing.T) {
	cleanup := initBrowserVNCHandlerTestDB(t)
	defer cleanup()

	_, inst := newBrowserVNCOpenClawInstance(t, "ins-oc-chk-ok")

	withAgentScriptRunner(t, func(_ context.Context, _, _ string, _ uint64, _ string, _ func(chunk string), _ map[string]string) (string, error) {
		return `{"installed":true,"running":true}`, nil
	})

	req := browserVNCReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/browser-vnc-check?id=%d", inst.ID), "vnc-user")
	rr := httptest.NewRecorder()
	HandleBrowserVNCCheck(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("脚本成功路径应返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleBrowserVNCCheck_RunScriptError(t *testing.T) {
	cleanup := initBrowserVNCHandlerTestDB(t)
	defer cleanup()

	_, inst := newBrowserVNCOpenClawInstance(t, "ins-oc-chk-err")

	withAgentScriptRunner(t, func(_ context.Context, _, _ string, _ uint64, _ string, _ func(chunk string), _ map[string]string) (string, error) {
		return "", hcommon.I18nError(i18n.MsgTATExecuteCommandFailed)
	})

	req := browserVNCReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/browser-vnc-check?id=%d", inst.ID), "vnc-user")
	rr := httptest.NewRecorder()
	HandleBrowserVNCCheck(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("脚本失败路径应返回 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ─── handleBrowserVNCInstall: runScript 闭包 ───

func TestHandleBrowserVNCInstall_RunScriptSuccess(t *testing.T) {
	cleanup := initBrowserVNCHandlerTestDB(t)
	defer cleanup()

	_, inst := newBrowserVNCOpenClawInstance(t, "ins-oc-inst-ok")

	withAgentScriptRunner(t, func(_ context.Context, _, _ string, _ uint64, _ string, _ func(chunk string), _ map[string]string) (string, error) {
		return `{"ok":true,"installed":true}`, nil
	})

	req := browserVNCReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/browser-vnc-install?id=%d", inst.ID), "vnc-user")
	rr := httptest.NewRecorder()
	handleBrowserVNCInstall(rr, req, testCVMFetcher)

	if rr.Code != http.StatusOK {
		t.Fatalf("脚本成功路径应返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleBrowserVNCInstall_RunScriptError(t *testing.T) {
	cleanup := initBrowserVNCHandlerTestDB(t)
	defer cleanup()

	_, inst := newBrowserVNCOpenClawInstance(t, "ins-oc-inst-err")

	withAgentScriptRunner(t, func(_ context.Context, _, _ string, _ uint64, _ string, _ func(chunk string), _ map[string]string) (string, error) {
		return "", hcommon.I18nError(i18n.MsgTATExecuteCommandFailed)
	})

	req := browserVNCReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/browser-vnc-install?id=%d", inst.ID), "vnc-user")
	rr := httptest.NewRecorder()
	handleBrowserVNCInstall(rr, req, testCVMFetcher)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("脚本失败路径应返回 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}
