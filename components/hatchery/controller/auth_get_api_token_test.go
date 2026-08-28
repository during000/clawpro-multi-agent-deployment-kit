package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"hatchery/model"
)

// authReqWithSession 构造一个登录态的 GET 请求（复用 reportReq 的 session 注入手法）。
func authReqWithSession(t *testing.T, username, url string) *http.Request {
	t.Helper()
	ensureSessionStore()
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Accept", "application/json")
	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = username
	rr := httptest.NewRecorder()
	session.Save(req, rr)
	for _, cookie := range rr.Result().Cookies() {
		req.AddCookie(cookie)
	}
	return req
}

// TestHandleGetAPIToken_DefaultMasksOnly
// 默认（无 reveal 参数 / reveal=false）只返 mask，不返 token；不写 audit。
func TestHandleGetAPIToken_DefaultMasksOnly(t *testing.T) {
	setupSkillInstancesDB(t)
	if err := model.DB(context.Background()).AutoMigrate(&model.AuditLog{}); err != nil {
		t.Fatalf("migrate AuditLog: %v", err)
	}
	ctx := context.Background()

	tok := "hk-1234567890abcdef1234567890abcdef1234567890abcdef"
	user := model.User{Username: "tok-default", Role: "user", APIToken: &tok}
	if err := model.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	for _, q := range []string{"", "?reveal=false", "?reveal=anything-not-true"} {
		t.Run("query="+q, func(t *testing.T) {
			rr := httptest.NewRecorder()
			HandleGetAPIToken(rr, authReqWithSession(t, "tok-default", "/api-token"+q))
			if rr.Code != http.StatusOK {
				t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
			}
			var resp map[string]any
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Fatalf("decode resp: %v", err)
			}
			if resp["exists"] != true {
				t.Errorf("exists 应 true，实际=%v", resp["exists"])
			}
			if mask, ok := resp["mask"].(string); !ok || mask == tok {
				t.Errorf("mask 应是掩码版本（非明文），实际=%q", resp["mask"])
			}
			if _, ok := resp["token"]; ok {
				t.Errorf("默认不应返 token 字段，实际=%v", resp["token"])
			}
		})
	}

	// 默认路径不应写 audit log
	time.Sleep(20 * time.Millisecond) // 等可能的异步 audit goroutine
	var n int64
	model.DB(ctx).Model(&model.AuditLog{}).
		Where("action = ? AND user_id = ?", "token_reveal", user.ID).Count(&n)
	if n != 0 {
		t.Errorf("默认请求不应写 token_reveal audit，实际写了 %d 条", n)
	}
}

// TestHandleGetAPIToken_RevealReturnsPlaintextAndAudits
// reveal=true 时同时返回 mask + token；写一条 token_reveal audit。
func TestHandleGetAPIToken_RevealReturnsPlaintextAndAudits(t *testing.T) {
	setupSkillInstancesDB(t)
	if err := model.DB(context.Background()).AutoMigrate(&model.AuditLog{}); err != nil {
		t.Fatalf("migrate AuditLog: %v", err)
	}
	ctx := context.Background()

	tok := "hk-deadbeefcafebabe1234567890abcdef1234567890abcd00"
	user := model.User{Username: "tok-reveal", Role: "user", APIToken: &tok}
	if err := model.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	rr := httptest.NewRecorder()
	HandleGetAPIToken(rr, authReqWithSession(t, "tok-reveal", "/api-token?reveal=true"))
	if rr.Code != http.StatusOK {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode resp: %v", err)
	}
	if got, _ := resp["token"].(string); got != tok {
		t.Errorf("token 应=明文，实际=%q", got)
	}
	if mask, _ := resp["mask"].(string); mask == tok {
		t.Errorf("mask 不应等于明文（仍应是掩码），实际=%q", mask)
	}
	if resp["exists"] != true {
		t.Errorf("exists 应 true")
	}

	// 等异步 audit goroutine 落库（model.LogAudit 是 go func）
	deadline := time.Now().Add(2 * time.Second)
	var n int64
	for time.Now().Before(deadline) {
		model.DB(ctx).Model(&model.AuditLog{}).
			Where("action = ? AND user_id = ?", "token_reveal", user.ID).Count(&n)
		if n >= 1 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if n < 1 {
		t.Errorf("应写 1 条 token_reveal audit，实际=%d", n)
	}
}

// TestHandleGetAPIToken_RevealNoTokenStillExistsFalse
// 用户没有 token 时，reveal=true 也只返 {exists:false}，不返 token 字段，不写 audit。
func TestHandleGetAPIToken_RevealNoTokenStillExistsFalse(t *testing.T) {
	setupSkillInstancesDB(t)
	if err := model.DB(context.Background()).AutoMigrate(&model.AuditLog{}); err != nil {
		t.Fatalf("migrate AuditLog: %v", err)
	}
	ctx := context.Background()

	user := model.User{Username: "tok-empty", Role: "user"}
	if err := model.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	rr := httptest.NewRecorder()
	HandleGetAPIToken(rr, authReqWithSession(t, "tok-empty", "/api-token?reveal=true"))
	if rr.Code != http.StatusOK {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode resp: %v", err)
	}
	if resp["exists"] != false {
		t.Errorf("exists 应 false，实际=%v", resp["exists"])
	}
	if _, ok := resp["token"]; ok {
		t.Errorf("无 token 时不应返 token 字段")
	}

	time.Sleep(20 * time.Millisecond)
	var n int64
	model.DB(ctx).Model(&model.AuditLog{}).
		Where("action = ? AND user_id = ?", "token_reveal", user.ID).Count(&n)
	if n != 0 {
		t.Errorf("无 token 时不应写 audit，实际=%d", n)
	}
}

// TestHandleGetAPIToken_RevealBlockedByAllowlist
// type='local-agent' 白名单已配置（≥1 条），但当前用户的 identifier 不在表里
// → reveal=true 应**退化为默认掩码路径**（不返 token、不写 audit），且 HTTP 仍 200。
//
// 设计意图：明文 token 接口与 local-agent 功能配套；不在白名单内的租户无此场景需求。
// 退化而不是 403 是为避免前端需要区分「未登录 / 未启用 / 真实错误」三种状态。
func TestHandleGetAPIToken_RevealBlockedByAllowlist(t *testing.T) {
	setupSkillInstancesDB(t)
	if err := model.DB(context.Background()).AutoMigrate(&model.AuditLog{}); err != nil {
		t.Fatalf("migrate AuditLog: %v", err)
	}
	ctx := context.Background()

	// 白名单只放行 allowed-tenant；当前用户 identifier=blocked-tenant
	if err := model.DB(ctx).Create(&model.FeatureAllowlist{
		Type: model.FeatureAllowlistTypeLocalAgent, Identifier: "allowed-tenant",
	}).Error; err != nil {
		t.Fatalf("create allowlist: %v", err)
	}
	tok := "hk-blocked0000000000000000000000000000000000000000"
	user := model.User{
		Username: "tok-blocked", Role: "user",
		Identifier: "blocked-tenant", APIToken: &tok,
	}
	if err := model.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	rr := httptest.NewRecorder()
	HandleGetAPIToken(rr, authReqWithSession(t, "tok-blocked", "/api-token?reveal=true"))
	if rr.Code != http.StatusOK {
		t.Fatalf("应 200（退化为掩码路径），实际=%d body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode resp: %v", err)
	}
	if _, ok := resp["token"]; ok {
		t.Errorf("白名单未命中不应返 token 字段，实际=%v", resp["token"])
	}
	if mask, _ := resp["mask"].(string); mask == tok {
		t.Errorf("mask 不应等于明文，实际=%q", mask)
	}

	time.Sleep(50 * time.Millisecond)
	var n int64
	model.DB(ctx).Model(&model.AuditLog{}).
		Where("action = ? AND user_id = ?", "token_reveal", user.ID).Count(&n)
	if n != 0 {
		t.Errorf("白名单未命中不应写 token_reveal audit，实际=%d", n)
	}
}

// TestHandleGetAPIToken_RevealAllowedByAllowlist
// 白名单已配置且当前用户的 identifier 命中 → reveal=true 正常返明文 + 写 audit。
func TestHandleGetAPIToken_RevealAllowedByAllowlist(t *testing.T) {
	setupSkillInstancesDB(t)
	if err := model.DB(context.Background()).AutoMigrate(&model.AuditLog{}); err != nil {
		t.Fatalf("migrate AuditLog: %v", err)
	}
	ctx := context.Background()

	if err := model.DB(ctx).Create(&model.FeatureAllowlist{
		Type: model.FeatureAllowlistTypeLocalAgent, Identifier: "allowed-tenant",
	}).Error; err != nil {
		t.Fatalf("create allowlist: %v", err)
	}
	tok := "hk-allowed1111111111111111111111111111111111111111"
	user := model.User{
		Username: "tok-allowed", Role: "user",
		Identifier: "allowed-tenant", APIToken: &tok,
	}
	if err := model.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	rr := httptest.NewRecorder()
	HandleGetAPIToken(rr, authReqWithSession(t, "tok-allowed", "/api-token?reveal=true"))
	if rr.Code != http.StatusOK {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode resp: %v", err)
	}
	if got, _ := resp["token"].(string); got != tok {
		t.Errorf("白名单命中应返明文 token，实际=%q", got)
	}

	deadline := time.Now().Add(2 * time.Second)
	var n int64
	for time.Now().Before(deadline) {
		model.DB(ctx).Model(&model.AuditLog{}).
			Where("action = ? AND user_id = ?", "token_reveal", user.ID).Count(&n)
		if n >= 1 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if n < 1 {
		t.Errorf("白名单命中应写 1 条 token_reveal audit，实际=%d", n)
	}
}
