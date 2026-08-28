// agent_bridge_audit_test.go
//
// 单元测试覆盖 HandleAgentBridgeAudit 函数的所有分支：
//   - 非 POST 方法 → 405
//   - 鉴权失败 → 401
//   - 请求体非法 JSON → 400
//   - action 为空 → 400
//   - status 为空 → 400
//   - action 缺少 "agent_bridge_" 前缀 → 400
//   - status 不在白名单中 → 400
//   - sk- 模式下 resource_id 与绑定实例不一致 → 403
//   - hk- 模式正常写入 → 200 + 审计记录落库
//   - sk- 模式正常写入（resource_id 自动补充）→ 200 + 审计记录落库
//   - started_at 超出合理范围时静默修正为当前时间
//   - 请求体超过 4KB 限制 → 400

package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// initAuditCallbackTestDB 初始化一个干净的内存 SQLite 数据库供审计回调测试使用。
// 与 initAgentBridgeTestDB 类似，但额外 AutoMigrate AuditLog 表。
func initAuditCallbackTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存 DB 失败: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("获取 sql.DB 失败: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&model.User{}, &model.Instance{}, &model.SiteConfig{}, &model.AuditLog{}); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	restoreSafe := useDBForTestWithSafeRestore(db)
	t.Cleanup(func() {
		time.Sleep(200 * time.Millisecond)
		restoreSafe()
	})

	if Store == nil {
		Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	}
	return db
}

// newAuditReq 构造一个 POST + Bearer Token 的 JSON 请求，用于审计回调测试。
func newAuditReq(t *testing.T, bearer string, body interface{}) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("编码请求体失败: %v", err)
		}
	}
	req := httptest.NewRequest(http.MethodPost, "/agent-bridge/audit", &buf)
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	req.Header.Set(openAPIHeader, "1")
	return req
}

// ============================================================================
// HandleAgentBridgeAudit ── 13 个用例
// ============================================================================

// TestHandleAgentBridgeAudit_MethodNotAllowed：非 POST → 405。
func TestHandleAgentBridgeAudit_MethodNotAllowed(t *testing.T) {
	initAuditCallbackTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/agent-bridge/audit", nil)
	w := httptest.NewRecorder()

	HandleAgentBridgeAudit(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望 405，实际 %d", w.Code)
	}
}

// TestHandleAgentBridgeAudit_AuthFailed：无效 token → 401。
func TestHandleAgentBridgeAudit_AuthFailed(t *testing.T) {
	initAuditCallbackTestDB(t)

	req := newAuditReq(t, "sk-invalid-token", map[string]string{
		"action": "agent_bridge_test",
		"status": "success",
	})
	w := httptest.NewRecorder()

	HandleAgentBridgeAudit(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望 401，实际 %d body=%s", w.Code, w.Body.String())
	}
}

// TestHandleAgentBridgeAudit_BadJSON：请求体非法 JSON → 400。
func TestHandleAgentBridgeAudit_BadJSON(t *testing.T) {
	db := initAuditCallbackTestDB(t)
	u := seedUserWithToken(t, db, "alice", "hk-alice-audit")
	seedInstance(t, db, "srv", u.ID, "ins-1", "sk-alice-audit")

	req := httptest.NewRequest(http.MethodPost, "/agent-bridge/audit",
		bytes.NewBufferString("{not-json"))
	req.Header.Set("Authorization", "Bearer sk-alice-audit")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(openAPIHeader, "1")
	w := httptest.NewRecorder()

	HandleAgentBridgeAudit(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际 %d body=%s", w.Code, w.Body.String())
	}
}

// TestHandleAgentBridgeAudit_ActionRequired：action 为空 → 400。
func TestHandleAgentBridgeAudit_ActionRequired(t *testing.T) {
	db := initAuditCallbackTestDB(t)
	u := seedUserWithToken(t, db, "alice", "hk-alice-audit2")
	seedInstance(t, db, "srv", u.ID, "ins-1", "sk-alice-audit2")

	req := newAuditReq(t, "sk-alice-audit2", map[string]string{
		"status": "success",
	})
	w := httptest.NewRecorder()

	HandleAgentBridgeAudit(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际 %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "action") {
		t.Errorf("响应体应包含 'action' 相关错误信息，实际=%q", w.Body.String())
	}
}

// TestHandleAgentBridgeAudit_StatusRequired：status 为空 → 400。
func TestHandleAgentBridgeAudit_StatusRequired(t *testing.T) {
	db := initAuditCallbackTestDB(t)
	u := seedUserWithToken(t, db, "alice", "hk-alice-audit3")
	seedInstance(t, db, "srv", u.ID, "ins-1", "sk-alice-audit3")

	req := newAuditReq(t, "sk-alice-audit3", map[string]string{
		"action": "agent_bridge_test",
	})
	w := httptest.NewRecorder()

	HandleAgentBridgeAudit(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际 %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "status") {
		t.Errorf("响应体应包含 'status' 相关错误信息，实际=%q", w.Body.String())
	}
}

// TestHandleAgentBridgeAudit_ActionPrefixRequired：action 缺少 "agent_bridge_" 前缀 → 400。
func TestHandleAgentBridgeAudit_ActionPrefixRequired(t *testing.T) {
	db := initAuditCallbackTestDB(t)
	u := seedUserWithToken(t, db, "alice", "hk-alice-audit4")
	seedInstance(t, db, "srv", u.ID, "ins-1", "sk-alice-audit4")

	req := newAuditReq(t, "sk-alice-audit4", map[string]string{
		"action": "desktop_install", // 缺少 agent_bridge_ 前缀
		"status": "success",
	})
	w := httptest.NewRecorder()

	HandleAgentBridgeAudit(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际 %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "agent_bridge_") {
		t.Errorf("响应体应提及 'agent_bridge_' 前缀要求，实际=%q", w.Body.String())
	}
}

// TestHandleAgentBridgeAudit_InvalidStatus：status 不在白名单中 → 400。
func TestHandleAgentBridgeAudit_InvalidStatus(t *testing.T) {
	db := initAuditCallbackTestDB(t)
	u := seedUserWithToken(t, db, "alice", "hk-alice-audit5")
	seedInstance(t, db, "srv", u.ID, "ins-1", "sk-alice-audit5")

	req := newAuditReq(t, "sk-alice-audit5", map[string]string{
		"action": "agent_bridge_test",
		"status": "unknown_status", // 非法 status
	})
	w := httptest.NewRecorder()

	HandleAgentBridgeAudit(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际 %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "status") {
		t.Errorf("响应体应提及合法 status 值，实际=%q", w.Body.String())
	}
}

// TestHandleAgentBridgeAudit_SkResourceMismatch：sk- 模式下 resource_id 与绑定实例不一致 → 403。
func TestHandleAgentBridgeAudit_SkResourceMismatch(t *testing.T) {
	db := initAuditCallbackTestDB(t)
	u := seedUserWithToken(t, db, "alice", "hk-alice-audit6")
	seedInstance(t, db, "srv", u.ID, "ins-bound-001", "sk-alice-audit6")

	req := newAuditReq(t, "sk-alice-audit6", map[string]string{
		"action":      "agent_bridge_desktop_install",
		"status":      "success",
		"resource_id": "ins-OTHER-999", // 与绑定实例 ins-bound-001 不一致
	})
	w := httptest.NewRecorder()

	HandleAgentBridgeAudit(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("期望 403，实际 %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "resource_id") {
		t.Errorf("响应体应包含 'resource_id' 相关错误信息，实际=%q", w.Body.String())
	}
}

// TestHandleAgentBridgeAudit_HkSuccess：hk- 模式正常写入审计记录 → 200 + 数据库有记录。
func TestHandleAgentBridgeAudit_HkSuccess(t *testing.T) {
	db := initAuditCallbackTestDB(t)
	seedUserWithToken(t, db, "bob", "hk-bob-audit-001")

	startedAt := time.Now().Add(-10 * time.Minute).Unix()
	req := newAuditReq(t, "hk-bob-audit-001", map[string]interface{}{
		"platform_id":   "hatchery",
		"action":        "agent_bridge_desktop_install",
		"resource":      "instance",
		"resource_id":   "ins-target-001",
		"invocation_id": "inv-abc123",
		"script_name":   "install.sh",
		"status":        "success",
		"trace_id":      "trace-xyz",
		"started_at":    startedAt,
	})
	w := httptest.NewRecorder()

	HandleAgentBridgeAudit(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp["ok"] != true {
		t.Errorf("响应应包含 ok:true，实际=%+v", resp)
	}

	// 等待异步写入完成
	time.Sleep(300 * time.Millisecond)

	// 验证审计记录已落库
	var logs []model.AuditLog
	db.Find(&logs)
	if len(logs) == 0 {
		t.Fatalf("审计记录未写入数据库")
	}

	found := false
	for _, log := range logs {
		if log.Action == "agent_bridge_desktop_install" &&
			log.ResourceID == "ins-target-001" &&
			log.Status == "success" &&
			log.Resource == "instance" {
			found = true
			// 验证 started_at 被正确设置（应接近我们传入的时间）
			expectedStart := time.Unix(startedAt, 0)
			if log.StartedAt.Sub(expectedStart) > time.Second {
				t.Errorf("started_at 不匹配：期望接近 %v，实际 %v", expectedStart, log.StartedAt)
			}
			break
		}
	}
	if !found {
		t.Errorf("未找到预期的审计记录，实际记录=%+v", logs)
	}
}

// TestHandleAgentBridgeAudit_SkSuccessAutoResourceID：sk- 模式下 resource_id 为空时自动补充绑定实例 ID。
func TestHandleAgentBridgeAudit_SkSuccessAutoResourceID(t *testing.T) {
	db := initAuditCallbackTestDB(t)
	u := seedUserWithToken(t, db, "charlie", "hk-charlie-audit")
	seedInstance(t, db, "srv", u.ID, "ins-auto-fill", "sk-charlie-audit")

	req := newAuditReq(t, "sk-charlie-audit", map[string]string{
		"action": "agent_bridge_desktop_check",
		"status": "failed",
		// 注意：不传 resource_id，应自动补充为 ins-auto-fill
	})
	w := httptest.NewRecorder()

	HandleAgentBridgeAudit(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d body=%s", w.Code, w.Body.String())
	}

	// 等待异步写入完成
	time.Sleep(300 * time.Millisecond)

	// 验证 resource_id 被自动补充
	var log model.AuditLog
	if err := db.Where("action = ?", "agent_bridge_desktop_check").First(&log).Error; err != nil {
		t.Fatalf("查询审计记录失败: %v", err)
	}
	if log.ResourceID != "ins-auto-fill" {
		t.Errorf("resource_id 应自动补充为 'ins-auto-fill'，实际=%q", log.ResourceID)
	}
	if log.Status != "failed" {
		t.Errorf("status 应为 'failed'，实际=%q", log.Status)
	}
}

// TestHandleAgentBridgeAudit_SkResourceIDMatch：sk- 模式下 resource_id 与绑定实例一致 → 正常通过。
func TestHandleAgentBridgeAudit_SkResourceIDMatch(t *testing.T) {
	db := initAuditCallbackTestDB(t)
	u := seedUserWithToken(t, db, "dave", "hk-dave-audit")
	seedInstance(t, db, "srv", u.ID, "ins-match-001", "sk-dave-audit")

	req := newAuditReq(t, "sk-dave-audit", map[string]string{
		"action":      "agent_bridge_desktop_install",
		"status":      "dispatched",
		"resource_id": "ins-match-001", // 与绑定实例一致
	})
	w := httptest.NewRecorder()

	HandleAgentBridgeAudit(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d body=%s", w.Code, w.Body.String())
	}
}

// TestHandleAgentBridgeAudit_StartedAtOutOfRange：started_at 超出合理范围时静默修正为当前时间。
func TestHandleAgentBridgeAudit_StartedAtOutOfRange(t *testing.T) {
	db := initAuditCallbackTestDB(t)
	seedUserWithToken(t, db, "eve", "hk-eve-audit")

	// started_at 设为 48 小时前（超出 24 小时限制）
	oldTime := time.Now().Add(-48 * time.Hour).Unix()
	req := newAuditReq(t, "hk-eve-audit", map[string]interface{}{
		"action":     "agent_bridge_desktop_timeout",
		"status":     "timeout",
		"started_at": oldTime,
	})
	w := httptest.NewRecorder()

	HandleAgentBridgeAudit(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200（静默修正，不拒绝），实际 %d body=%s", w.Code, w.Body.String())
	}

	// 等待异步写入完成
	time.Sleep(300 * time.Millisecond)

	// 验证 started_at 被修正为接近当前时间（而非 48 小时前）
	var log model.AuditLog
	if err := db.Where("action = ?", "agent_bridge_desktop_timeout").First(&log).Error; err != nil {
		t.Fatalf("查询审计记录失败: %v", err)
	}
	// 修正后的 started_at 应在最近 5 秒内
	if time.Since(log.StartedAt) > 5*time.Second {
		t.Errorf("started_at 应被修正为当前时间，实际=%v（距今 %v）", log.StartedAt, time.Since(log.StartedAt))
	}
}

// TestHandleAgentBridgeAudit_AllValidStatuses：验证所有合法 status 值都能通过。
func TestHandleAgentBridgeAudit_AllValidStatuses(t *testing.T) {
	db := initAuditCallbackTestDB(t)
	seedUserWithToken(t, db, "frank", "hk-frank-audit")

	validStatuses := []string{"success", "failed", "timeout", "dispatched"}
	for _, status := range validStatuses {
		t.Run(status, func(t *testing.T) {
			req := newAuditReq(t, "hk-frank-audit", map[string]string{
				"action": "agent_bridge_test_" + status,
				"status": status,
			})
			w := httptest.NewRecorder()

			HandleAgentBridgeAudit(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("status=%q 应通过校验，实际 HTTP %d body=%s", status, w.Code, w.Body.String())
			}
		})
	}
}

// TestHandleAgentBridgeAudit_BodyTooLarge：请求体超过 4KB 限制 → 400。
func TestHandleAgentBridgeAudit_BodyTooLarge(t *testing.T) {
	db := initAuditCallbackTestDB(t)
	u := seedUserWithToken(t, db, "grace", "hk-grace-audit")
	seedInstance(t, db, "srv", u.ID, "ins-1", "sk-grace-audit")

	// 构造一个超过 4KB 的请求体
	largeBody := strings.Repeat("x", 5000)
	body := map[string]string{
		"action": "agent_bridge_test",
		"status": "success",
		"extra":  largeBody,
	}
	req := newAuditReq(t, "sk-grace-audit", body)
	w := httptest.NewRecorder()

	HandleAgentBridgeAudit(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400（body 超限），实际 %d body=%s", w.Code, w.Body.String())
	}
}
