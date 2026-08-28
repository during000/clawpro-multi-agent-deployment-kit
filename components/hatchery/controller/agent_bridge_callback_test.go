// agent_bridge_callback_test.go
//
// 单元测试覆盖 controller/agent_bridge_callback.go 的 6 个函数：
//   - resolveAgentBridgeIdentity   （鉴权分流：sk- ProxyToken / hk- API Token / 异常）
//   - HandleAgentBridgeSTS         （长期密钥 / 凭据未配置 / 鉴权失败 / 非 POST / Bad JSON）
//   - HandleAgentBridgeAuth        （默认放行 / desktop:* + BrowserVNCEnable / sk- 资源不一致仅告警）
//   - HandleAgentBridgeInstances   （sk- 仅返回绑定实例 / hk- 返回全部 / region 过滤 / 空 DB / 非 POST）
//   - extractInstanceIDFromResource（纯字符串解析）
//   - agentBridgeBatchDescribeCVM  （getCredential 失败时返回空 map）
//
// 设计要点：
//  1. 所有用例在内存 SQLite 中跑（model.UseDBForTest），不触达真实腾讯云 API；
//  2. 涉及 CVM API 的路径（HandleAgentBridgeInstances）通过"不预置 CVMSecretId"
//     使 GetCVMClient 返回 err，agentBridgeBatchDescribeCVM 走"返回空 map"分支，
//     主流程不阻断（status 仍为 UNKNOWN），从而覆盖 DB 查询/实例锁/region 过滤；
//  3. STS 模式（uin != ""）需要 stub RefreshSTSCredentials，工作量大，仅覆盖
//     "长期密钥模式"（uin == ""，即默认 TenantSnapshot）；
//  4. main_test.go 已 AutoMigrate 全部所需表，本文件直接 reset 默认 testSafeDB
//     即可使用。

package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hatchery/i18n"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// ===== 测试 fixture =====

// agentBridgeStrPtr 返回 *string，用于给 Instance.ProxyToken 这种指针字段赋值。
func agentBridgeStrPtr(s string) *string { return &s }

// initAgentBridgeTestDB 初始化一个干净的内存 SQLite 数据库供本组测试使用。
//
// 关键设计点（防止异步 goroutine 引起的 race / panic）：
//
//  1. 用独立的 :memory: SQLite + AutoMigrate 本测试涉及的三张表，
//     与 main_test.go 中的 testSafeDB 隔离，避免用例间数据污染。
//
//  2. 通过 useDBForTestWithSafeRestore 切换全局 gdb，cleanup 时不会回到 nil
//     而是切到 testSafeDB（main_test.go 中已 AutoMigrate 全表）；
//     避免 hk- 鉴权成功后 getUserFromToken 启动的异步 goroutine
//     （UpdateAPITokenLastUsed，走 common.DetachContext 故意脱离请求 ctx）
//     在测试退出后撞上 nil gdb 触发 nil-pointer panic。
//
//  3. cleanup 内 sleep 200ms 让在飞的异步 goroutine 完成对 gdb 的读
//     再切 DB，避免 -race 报 DATA RACE（仓库惯例：openclaw_handler_guards_test.go）。
//
//  4. 确保 controller.Store（gorilla CookieStore）已初始化，避免无 Bearer
//     请求走到 getSession 时 nil-deref。
func initAgentBridgeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存 DB 失败: %v", err)
	}
	// 关键：SQLite :memory: 数据库是每条连接独立的。当 handler 内的异步 goroutine
	// （如 UpdateAPITokenLastUsed）和主线程同时向连接池借连接时，可能拿到不同的
	// 物理连接，导致看到不同的空 sqlite 实例（no such table）。
	// 强制 MaxOpenConns(1) 确保所有查询共用同一条物理连接。
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("获取 sql.DB 失败: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&model.User{}, &model.Instance{}, &model.SiteConfig{}); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	restoreSafe := useDBForTestWithSafeRestore(db)
	t.Cleanup(func() {
		// 等异步 goroutine（UpdateAPITokenLastUsed）完成对 gdb 的读再恢复，
		// 避免 -race 报 DATA RACE。仓库惯例参考 openclaw_handler_guards_test.go::35
		time.Sleep(200 * time.Millisecond)
		restoreSafe()
	})

	if Store == nil {
		Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	}
	return db
}

// seedUserWithToken 创建一个未封禁的用户，并设置 hk- API Token；返回 user 指针。
func seedUserWithToken(t *testing.T, db *gorm.DB, username, hkToken string) *model.User {
	t.Helper()
	u := &model.User{
		Username: username,
		Password: "x",
		Role:     "user",
		APIToken: agentBridgeStrPtr(hkToken),
	}
	if err := db.Create(u).Error; err != nil {
		t.Fatalf("创建用户 %q 失败: %v", username, err)
	}
	return u
}

// seedInstance 创建一台带 ProxyToken 的实例，返回 instance 指针。
func seedInstance(t *testing.T, db *gorm.DB, name string, userID uint, instanceID, proxyToken string) *model.Instance {
	t.Helper()
	inst := &model.Instance{
		Name:       name,
		UserID:     userID,
		InstanceId: instanceID,
		ProxyToken: agentBridgeStrPtr(proxyToken),
	}
	if err := db.Create(inst).Error; err != nil {
		t.Fatalf("创建实例 %q 失败: %v", name, err)
	}
	return inst
}

// withCVMRegion 临时设置全局 CVMRegion，t.Cleanup 还原。
func withCVMRegion(t *testing.T, region string) {
	t.Helper()
	orig := CVMRegion
	CVMRegion = region
	t.Cleanup(func() { CVMRegion = orig })
}

// newJSONReq 构造一个 POST + Bearer Token 的 JSON 请求；同时打上 openAPIHeader
// 模拟 WithOpenAPI 中间件已包过的状态（hk- 路径下 getUserFromToken 才会查 DB）。
func newJSONReq(t *testing.T, path, bearer string, body interface{}) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("编码请求体失败: %v", err)
		}
	}
	req := httptest.NewRequest(http.MethodPost, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	// 模拟 WithOpenAPI 注入的标记，让 getUserFromToken 走 OpenAPI 分支
	req.Header.Set(openAPIHeader, "1")
	return req
}

// ============================================================================
// 1. resolveAgentBridgeIdentity ── 8 个用例
// ============================================================================

// TestResolveAgentBridgeIdentity_SkValidToken：sk- 命中实例，user 未封禁。
func TestResolveAgentBridgeIdentity_SkValidToken(t *testing.T) {
	db := initAgentBridgeTestDB(t)
	u := seedUserWithToken(t, db, "alice", "hk-alice-token")
	inst := seedInstance(t, db, "alice-srv", u.ID, "ins-aaa", "sk-alice-001")

	req := newJSONReq(t, "/agent-bridge/sts", "sk-alice-001", nil)
	w := httptest.NewRecorder()

	user, bound, ok := resolveAgentBridgeIdentity(w, req)
	if !ok {
		t.Fatalf("期望鉴权通过，实际失败；HTTP=%d body=%s", w.Code, w.Body.String())
	}
	if user == nil || user.ID != u.ID {
		t.Errorf("user 不匹配：want id=%d, got=%+v", u.ID, user)
	}
	if bound == nil || bound.ID != inst.ID || bound.InstanceId != "ins-aaa" {
		t.Errorf("boundInstance 不匹配：want id=%d ins-aaa, got=%+v", inst.ID, bound)
	}
}

// TestResolveAgentBridgeIdentity_SkInvalidToken：sk- 但 DB 查不到 → 401。
func TestResolveAgentBridgeIdentity_SkInvalidToken(t *testing.T) {
	initAgentBridgeTestDB(t)

	req := newJSONReq(t, "/agent-bridge/sts", "sk-not-exist", nil)
	w := httptest.NewRecorder()

	user, bound, ok := resolveAgentBridgeIdentity(w, req)
	if ok {
		t.Fatalf("期望鉴权失败，实际通过 user=%+v bound=%+v", user, bound)
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望 401，实际 %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), i18n.T(req.Context(), i18n.MsgABInvalidProxyToken)) {
		t.Errorf("响应体未包含 '%s'：%q", i18n.T(req.Context(), i18n.MsgABInvalidProxyToken), w.Body.String())
	}
}

// TestResolveAgentBridgeIdentity_SkOrphanInstance：sk- 命中实例但 user 不存在 → 401。
func TestResolveAgentBridgeIdentity_SkOrphanInstance(t *testing.T) {
	db := initAgentBridgeTestDB(t)
	// 实例的 UserID 指向一个不存在的 user_id (99999)
	seedInstance(t, db, "orphan", 99999, "ins-orphan", "sk-orphan-001")

	req := newJSONReq(t, "/agent-bridge/sts", "sk-orphan-001", nil)
	w := httptest.NewRecorder()

	if _, _, ok := resolveAgentBridgeIdentity(w, req); ok {
		t.Fatalf("期望鉴权失败")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望 401，实际 %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), i18n.T(req.Context(), i18n.MsgABOrphanProxyToken)) {
		t.Errorf("响应体未包含 '%s'：%q", i18n.T(req.Context(), i18n.MsgABOrphanProxyToken), w.Body.String())
	}
}

// TestResolveAgentBridgeIdentity_SkBannedUser：sk- 命中实例，但 user 已被软删（封禁）→ 403。
func TestResolveAgentBridgeIdentity_SkBannedUser(t *testing.T) {
	db := initAgentBridgeTestDB(t)
	u := seedUserWithToken(t, db, "banned", "hk-banned")
	seedInstance(t, db, "banned-srv", u.ID, "ins-ban", "sk-banned-001")

	// 软删除（封禁）该用户
	if err := db.Delete(&model.User{}, u.ID).Error; err != nil {
		t.Fatalf("软删除用户失败: %v", err)
	}

	req := newJSONReq(t, "/agent-bridge/sts", "sk-banned-001", nil)
	w := httptest.NewRecorder()

	if _, _, ok := resolveAgentBridgeIdentity(w, req); ok {
		t.Fatalf("期望鉴权失败")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("期望 403，实际 %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), i18n.T(req.Context(), i18n.MsgABUserBanned)) {
		t.Errorf("响应体未包含 '%s'：%q", i18n.T(req.Context(), i18n.MsgABUserBanned), w.Body.String())
	}
}

// TestResolveAgentBridgeIdentity_HkFallsBackToRequireLogin：hk- 走 requireLogin 分支。
func TestResolveAgentBridgeIdentity_HkFallsBackToRequireLogin(t *testing.T) {
	db := initAgentBridgeTestDB(t)
	u := seedUserWithToken(t, db, "bob", "hk-bob-token-001")

	req := newJSONReq(t, "/agent-bridge/sts", "hk-bob-token-001", nil)
	w := httptest.NewRecorder()

	user, bound, ok := resolveAgentBridgeIdentity(w, req)
	if !ok {
		t.Fatalf("期望鉴权通过；HTTP=%d body=%s", w.Code, w.Body.String())
	}
	if user == nil || user.ID != u.ID {
		t.Errorf("user 不匹配：want id=%d, got=%+v", u.ID, user)
	}
	if bound != nil {
		t.Errorf("hk- 模式 boundInstance 必须为 nil，实际=%+v", bound)
	}
}

// TestResolveAgentBridgeIdentity_NoBearer：未携带 Authorization → requireLogin
// 写入 401（JSON 模式），整体返回 false。
func TestResolveAgentBridgeIdentity_NoBearer(t *testing.T) {
	initAgentBridgeTestDB(t)

	req := newJSONReq(t, "/agent-bridge/sts", "", nil)
	// 显式去掉 openAPIHeader 也行，但保留即可：getUserFromToken 没有 Bearer 直接返回 nil
	req.Header.Set("Accept", "application/json") // 写 JSON 401
	w := httptest.NewRecorder()

	if _, _, ok := resolveAgentBridgeIdentity(w, req); ok {
		t.Fatalf("期望鉴权失败")
	}
	if w.Code == 0 || w.Code == http.StatusOK {
		t.Errorf("期望写入非 200 错误码，实际 %d", w.Code)
	}
}

// TestResolveAgentBridgeIdentity_AdminToken：AdminToken 在 hatchery 中
// 通过 getUserFromToken 返回 ID=0 的虚拟 admin-token 用户，但 getLoginUser
// 把 ID==0 视为未登录 → resolveAgentBridgeIdentity 鉴权失败。
//
// 这是项目当前的实际行为：agent-bridge 回调走"用户级"凭证（hk-/sk-），
// AdminToken 是后台管理凭证，不参与回调链路。本用例锁定该行为，
// 防止后续重构无意中放开 AdminToken 直通 agent-bridge。
func TestResolveAgentBridgeIdentity_AdminToken(t *testing.T) {
	initAgentBridgeTestDB(t)

	origAdmin := AdminToken
	AdminToken = "super-admin-secret"
	t.Cleanup(func() { AdminToken = origAdmin })

	req := newJSONReq(t, "/agent-bridge/sts", "super-admin-secret", nil)
	w := httptest.NewRecorder()

	user, bound, ok := resolveAgentBridgeIdentity(w, req)
	if ok {
		t.Fatalf("AdminToken 不应通过 agent-bridge 鉴权，但实际通过了 user=%+v bound=%+v", user, bound)
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望 401，实际 %d", w.Code)
	}
}

// TestResolveAgentBridgeIdentity_HkInvalidToken：hk- 但 DB 中无对应 token → requireLogin 401。
func TestResolveAgentBridgeIdentity_HkInvalidToken(t *testing.T) {
	initAgentBridgeTestDB(t)

	req := newJSONReq(t, "/agent-bridge/sts", "hk-does-not-exist", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	if _, _, ok := resolveAgentBridgeIdentity(w, req); ok {
		t.Fatalf("期望鉴权失败")
	}
	// requireLogin 对未识别 Token 视为 BannedError 之外的普通错误 → 403
	// （getLoginUser 返回 err 时走 default 分支：账户已被封禁 / 403）
	if w.Code != http.StatusForbidden && w.Code != http.StatusUnauthorized {
		t.Errorf("期望 401 或 403，实际 %d", w.Code)
	}
}

// ============================================================================
// 2. HandleAgentBridgeSTS ── 4 个用例
// ============================================================================

// TestHandleAgentBridgeSTS_LongTermMode：无 UIN 配置 → 返回长期 AK/SK，token=""，expired_time=0。
func TestHandleAgentBridgeSTS_LongTermMode(t *testing.T) {
	db := initAgentBridgeTestDB(t)
	if err := db.Create(&model.SiteConfig{
		Name:         "Test",
		CVMSecretId:  "AKID-test-001",
		CVMSecretKey: "secret-key-001",
	}).Error; err != nil {
		t.Fatalf("创建 SiteConfig 失败: %v", err)
	}
	u := seedUserWithToken(t, db, "alice", "hk-alice")
	seedInstance(t, db, "srv", u.ID, "ins-1", "sk-alice-001")

	req := newJSONReq(t, "/agent-bridge/sts", "sk-alice-001", map[string]string{
		"platform_id": "hatchery",
		"user_id":     "1",
	})
	w := httptest.NewRecorder()

	HandleAgentBridgeSTS(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		TmpSecretID  string      `json:"tmp_secret_id"`
		TmpSecretKey string      `json:"tmp_secret_key"`
		Token        string      `json:"token"`
		ExpiredTime  json.Number `json:"expired_time"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v body=%s", err, w.Body.String())
	}
	if resp.TmpSecretID != "AKID-test-001" || resp.TmpSecretKey != "secret-key-001" {
		t.Errorf("AK/SK 不匹配：%+v", resp)
	}
	if resp.Token != "" {
		t.Errorf("长期密钥模式 token 必须为空，实际=%q", resp.Token)
	}
	if resp.ExpiredTime.String() != "0" {
		t.Errorf("长期密钥模式 expired_time 必须为 0，实际=%s", resp.ExpiredTime.String())
	}
}

// TestHandleAgentBridgeSTS_NoCredentials：SiteConfig 未配置 AK/SK → 500 + "credentials not configured"。
func TestHandleAgentBridgeSTS_NoCredentials(t *testing.T) {
	db := initAgentBridgeTestDB(t)
	if err := db.Create(&model.SiteConfig{Name: "Test"}).Error; err != nil {
		t.Fatalf("创建 SiteConfig 失败: %v", err)
	}
	u := seedUserWithToken(t, db, "alice", "hk-alice")
	seedInstance(t, db, "srv", u.ID, "ins-1", "sk-alice-001")

	req := newJSONReq(t, "/agent-bridge/sts", "sk-alice-001", map[string]string{
		"platform_id": "hatchery", "user_id": "1",
	})
	w := httptest.NewRecorder()

	HandleAgentBridgeSTS(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("期望 500，实际 %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), i18n.T(req.Context(), i18n.MsgABCredsNotConfigured)) {
		t.Errorf("响应体应包含 'credentials not configured'，实际=%q", w.Body.String())
	}
}

// TestHandleAgentBridgeSTS_AuthFailed：错误的 sk- token → 401，不进入业务逻辑。
func TestHandleAgentBridgeSTS_AuthFailed(t *testing.T) {
	initAgentBridgeTestDB(t)

	req := newJSONReq(t, "/agent-bridge/sts", "sk-wrong-token", map[string]string{
		"platform_id": "hatchery", "user_id": "1",
	})
	w := httptest.NewRecorder()

	HandleAgentBridgeSTS(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望 401，实际 %d", w.Code)
	}
}

// TestHandleAgentBridgeSTS_MethodNotAllowed：非 POST → 405。
func TestHandleAgentBridgeSTS_MethodNotAllowed(t *testing.T) {
	initAgentBridgeTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/agent-bridge/sts", nil)
	w := httptest.NewRecorder()

	HandleAgentBridgeSTS(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望 405，实际 %d", w.Code)
	}
}

// ============================================================================
// 3. HandleAgentBridgeAuth ── 5 个用例
// ============================================================================

// TestHandleAgentBridgeAuth_AllowedDefault：未触发 desktop:* 开关 → allowed=true。
func TestHandleAgentBridgeAuth_AllowedDefault(t *testing.T) {
	db := initAgentBridgeTestDB(t)
	if err := db.Create(&model.SiteConfig{Name: "Test"}).Error; err != nil {
		t.Fatalf("创建 SiteConfig 失败: %v", err)
	}
	u := seedUserWithToken(t, db, "alice", "hk-alice")
	seedInstance(t, db, "srv", u.ID, "ins-1", "sk-alice-001")

	req := newJSONReq(t, "/agent-bridge/auth", "sk-alice-001", map[string]string{
		"platform_id": "hatchery", "user_id": "1",
		"action": "instance:list", "resource": "hatchery",
	})
	w := httptest.NewRecorder()

	HandleAgentBridgeAuth(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Allowed bool   `json:"allowed"`
		Reason  string `json:"reason"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if !resp.Allowed {
		t.Errorf("默认应放行，实际 allowed=%v reason=%q", resp.Allowed, resp.Reason)
	}
}

// TestHandleAgentBridgeAuth_DesktopActionFeatureOff：desktop:install + BrowserVNCEnable=false → allowed=false。
func TestHandleAgentBridgeAuth_DesktopActionFeatureOff(t *testing.T) {
	db := initAgentBridgeTestDB(t)
	if err := db.Create(&model.SiteConfig{Name: "Test", BrowserVNCEnable: false}).Error; err != nil {
		t.Fatalf("创建 SiteConfig 失败: %v", err)
	}
	u := seedUserWithToken(t, db, "alice", "hk-alice")
	seedInstance(t, db, "srv", u.ID, "ins-1", "sk-alice-001")

	req := newJSONReq(t, "/agent-bridge/auth", "sk-alice-001", map[string]string{
		"platform_id": "hatchery", "user_id": "1",
		"action": "desktop:install", "resource": "hatchery",
	})
	w := httptest.NewRecorder()

	HandleAgentBridgeAuth(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200（业务层 allowed=false 也是 200），实际 %d", w.Code)
	}
	var resp struct {
		Allowed bool   `json:"allowed"`
		Reason  string `json:"reason"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Allowed {
		t.Errorf("BrowserVNCEnable=false 时应拒绝 desktop:* 操作")
	}
	if !strings.Contains(resp.Reason, "browser vnc") {
		t.Errorf("拒绝原因应说明 vnc 功能未开启，实际=%q", resp.Reason)
	}
}

// TestHandleAgentBridgeAuth_DesktopActionFeatureOn：desktop:install + BrowserVNCEnable=true → allowed=true。
func TestHandleAgentBridgeAuth_DesktopActionFeatureOn(t *testing.T) {
	db := initAgentBridgeTestDB(t)
	if err := db.Create(&model.SiteConfig{Name: "Test", BrowserVNCEnable: true}).Error; err != nil {
		t.Fatalf("创建 SiteConfig 失败: %v", err)
	}
	u := seedUserWithToken(t, db, "alice", "hk-alice")
	seedInstance(t, db, "srv", u.ID, "ins-1", "sk-alice-001")

	req := newJSONReq(t, "/agent-bridge/auth", "sk-alice-001", map[string]string{
		"platform_id": "hatchery", "user_id": "1",
		"action": "desktop:install", "resource": "hatchery",
	})
	w := httptest.NewRecorder()

	HandleAgentBridgeAuth(w, req)

	var resp struct {
		Allowed bool `json:"allowed"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.Allowed {
		t.Errorf("BrowserVNCEnable=true 时应放行 desktop:* 操作，body=%s", w.Body.String())
	}
}

// TestHandleAgentBridgeAuth_SkResourceMismatch：变体 A（保守模式）：sk- 模式下
// resource 携带的 instance 与绑定实例不一致，仅打 warn，不拒绝。
func TestHandleAgentBridgeAuth_SkResourceMismatch(t *testing.T) {
	db := initAgentBridgeTestDB(t)
	if err := db.Create(&model.SiteConfig{Name: "Test", BrowserVNCEnable: true}).Error; err != nil {
		t.Fatalf("创建 SiteConfig 失败: %v", err)
	}
	u := seedUserWithToken(t, db, "alice", "hk-alice")
	seedInstance(t, db, "srv", u.ID, "ins-bound", "sk-alice-001")

	req := newJSONReq(t, "/agent-bridge/auth", "sk-alice-001", map[string]string{
		"platform_id": "hatchery", "user_id": "1",
		"action": "desktop:install", "resource": "instance:ins-OTHER",
	})
	w := httptest.NewRecorder()

	HandleAgentBridgeAuth(w, req)

	var resp struct {
		Allowed bool `json:"allowed"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.Allowed {
		t.Errorf("变体 A 保守模式下，resource 不一致只 warn 不拒绝；实际 allowed=%v", resp.Allowed)
	}
}

// TestHandleAgentBridgeAuth_BadJSON：body 非法 JSON → 400。
func TestHandleAgentBridgeAuth_BadJSON(t *testing.T) {
	db := initAgentBridgeTestDB(t)
	if err := db.Create(&model.SiteConfig{Name: "Test"}).Error; err != nil {
		t.Fatalf("创建 SiteConfig 失败: %v", err)
	}
	u := seedUserWithToken(t, db, "alice", "hk-alice")
	seedInstance(t, db, "srv", u.ID, "ins-1", "sk-alice-001")

	req := httptest.NewRequest(http.MethodPost, "/agent-bridge/auth",
		bytes.NewBufferString("{not-json"))
	req.Header.Set("Authorization", "Bearer sk-alice-001")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(openAPIHeader, "1")
	w := httptest.NewRecorder()

	HandleAgentBridgeAuth(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际 %d body=%s", w.Code, w.Body.String())
	}
}

// ============================================================================
// 4. HandleAgentBridgeInstances ── 5 个用例
// ============================================================================

// 这些用例不预置 CVMSecretId/Key，使 GetCVMClient 返回 err，
// agentBridgeBatchDescribeCVM 走"返回空 map"分支（status 维持 UNKNOWN），
// 主流程不阻断 → 可独立验证 SQL 过滤、实例锁、region 过滤、空列表等核心逻辑。

// TestHandleAgentBridgeInstances_SkOnlyReturnsBoundInstance：sk- 模式下
// 用户名下有多台实例，但回调只能返回 ProxyToken 绑定的那一台（实例锁）。
func TestHandleAgentBridgeInstances_SkOnlyReturnsBoundInstance(t *testing.T) {
	db := initAgentBridgeTestDB(t)
	if err := db.Create(&model.SiteConfig{Name: "Test"}).Error; err != nil {
		t.Fatalf("创建 SiteConfig 失败: %v", err)
	}
	withCVMRegion(t, "ap-guangzhou")

	u := seedUserWithToken(t, db, "alice", "hk-alice")
	bound := seedInstance(t, db, "bound", u.ID, "ins-bound", "sk-alice-001")
	seedInstance(t, db, "extra-1", u.ID, "ins-extra-1", "sk-alice-002")
	seedInstance(t, db, "extra-2", u.ID, "ins-extra-2", "sk-alice-003")

	req := newJSONReq(t, "/agent-bridge/instances", "sk-alice-001", map[string]string{
		"platform_id": "hatchery", "user_id": "1", "region": "ap-guangzhou",
	})
	w := httptest.NewRecorder()

	HandleAgentBridgeInstances(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Instances []map[string]string `json:"instances"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if len(resp.Instances) != 1 {
		t.Fatalf("sk- 模式应只返回绑定实例 1 台，实际返回 %d 台：%+v", len(resp.Instances), resp.Instances)
	}
	if resp.Instances[0]["instance_id"] != bound.InstanceId {
		t.Errorf("返回的实例不是 boundInstance：%+v", resp.Instances[0])
	}
}

// TestHandleAgentBridgeInstances_HkReturnsAllUserInstances：hk- 模式下返回用户名下所有实例。
func TestHandleAgentBridgeInstances_HkReturnsAllUserInstances(t *testing.T) {
	db := initAgentBridgeTestDB(t)
	if err := db.Create(&model.SiteConfig{Name: "Test"}).Error; err != nil {
		t.Fatalf("创建 SiteConfig 失败: %v", err)
	}
	withCVMRegion(t, "ap-guangzhou")

	u := seedUserWithToken(t, db, "alice", "hk-alice-001")
	seedInstance(t, db, "i1", u.ID, "ins-1", "sk-001")
	seedInstance(t, db, "i2", u.ID, "ins-2", "sk-002")
	seedInstance(t, db, "i3", u.ID, "ins-3", "sk-003")

	req := newJSONReq(t, "/agent-bridge/instances", "hk-alice-001", map[string]string{
		"platform_id": "hatchery", "user_id": "1", "region": "ap-guangzhou",
	})
	w := httptest.NewRecorder()

	HandleAgentBridgeInstances(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Instances []map[string]string `json:"instances"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Instances) != 3 {
		t.Errorf("hk- 模式应返回用户全部实例 3 台，实际 %d 台", len(resp.Instances))
	}
}

// TestHandleAgentBridgeInstances_RegionFilter：region 与 CVMRegion 不一致时过滤掉。
func TestHandleAgentBridgeInstances_RegionFilter(t *testing.T) {
	db := initAgentBridgeTestDB(t)
	if err := db.Create(&model.SiteConfig{Name: "Test"}).Error; err != nil {
		t.Fatalf("创建 SiteConfig 失败: %v", err)
	}
	withCVMRegion(t, "ap-guangzhou")

	u := seedUserWithToken(t, db, "alice", "hk-alice-001")
	seedInstance(t, db, "i1", u.ID, "ins-1", "sk-001")

	req := newJSONReq(t, "/agent-bridge/instances", "hk-alice-001", map[string]string{
		"platform_id": "hatchery", "user_id": "1",
		"region": "ap-shanghai", // 与 CVMRegion 不一致
	})
	w := httptest.NewRecorder()

	HandleAgentBridgeInstances(w, req)

	var resp struct {
		Instances []map[string]string `json:"instances"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Instances) != 0 {
		t.Errorf("region 不匹配时应过滤为空，实际返回 %d 台", len(resp.Instances))
	}
}

// TestHandleAgentBridgeInstances_EmptyDB：用户名下没有实例 → instances:[]，不调用 CVM。
func TestHandleAgentBridgeInstances_EmptyDB(t *testing.T) {
	db := initAgentBridgeTestDB(t)
	if err := db.Create(&model.SiteConfig{Name: "Test"}).Error; err != nil {
		t.Fatalf("创建 SiteConfig 失败: %v", err)
	}
	withCVMRegion(t, "ap-guangzhou")

	seedUserWithToken(t, db, "alice", "hk-alice-001")
	// 注意：故意不创建实例

	req := newJSONReq(t, "/agent-bridge/instances", "hk-alice-001", map[string]string{
		"platform_id": "hatchery", "user_id": "1",
	})
	w := httptest.NewRecorder()

	HandleAgentBridgeInstances(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"instances":[]`) {
		t.Errorf("响应体应为 instances:[]，实际=%q", w.Body.String())
	}
}

// TestHandleAgentBridgeInstances_MethodNotAllowed：非 POST → 405。
func TestHandleAgentBridgeInstances_MethodNotAllowed(t *testing.T) {
	initAgentBridgeTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/agent-bridge/instances", nil)
	w := httptest.NewRecorder()

	HandleAgentBridgeInstances(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望 405，实际 %d", w.Code)
	}
}

// ============================================================================
// 5. extractInstanceIDFromResource ── 4 个用例（纯函数）
// ============================================================================

func TestExtractInstanceIDFromResource(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"标准格式", "instance:ins-xxx", "ins-xxx"},
		{"空字符串", "", ""},
		{"其他形态", "hatchery", ""},
		{"仅前缀（仍按格式提取，得到空 id）", "instance:", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractInstanceIDFromResource(tc.in)
			if got != tc.want {
				t.Errorf("input=%q want=%q got=%q", tc.in, tc.want, got)
			}
		})
	}
}

// ============================================================================
// 6. agentBridgeBatchDescribeCVM ── getCredential 失败时返回空 map
// ============================================================================

// TestAgentBridgeBatchDescribeCVM_NoCredential：SiteConfig 未配置 AK/SK →
// getCredential 失败 → GetCVMClient 失败 → 函数返回空 map（降级），不 panic。
func TestAgentBridgeBatchDescribeCVM_NoCredential(t *testing.T) {
	db := initAgentBridgeTestDB(t)
	// 故意不预置 CVMSecretId / CVMSecretKey
	if err := db.Create(&model.SiteConfig{Name: "Test"}).Error; err != nil {
		t.Fatalf("创建 SiteConfig 失败: %v", err)
	}
	withCVMRegion(t, "ap-guangzhou")

	got := agentBridgeBatchDescribeCVM(context.Background(), []string{"ins-1", "ins-2"})
	if got == nil {
		t.Fatalf("函数应永远返回非 nil map，实际 nil")
	}
	if len(got) != 0 {
		t.Errorf("getCredential 失败时应返回空 map，实际=%+v", got)
	}
}

// TestAgentBridgeBatchDescribeCVM_EmptyInput：空输入 → 空输出。
func TestAgentBridgeBatchDescribeCVM_EmptyInput(t *testing.T) {
	initAgentBridgeTestDB(t)
	got := agentBridgeBatchDescribeCVM(context.Background(), nil)
	if got == nil {
		t.Fatalf("函数应永远返回非 nil map")
	}
	if len(got) != 0 {
		t.Errorf("空输入应得到空 map，实际=%+v", got)
	}
}
