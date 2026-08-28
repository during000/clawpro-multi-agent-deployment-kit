package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// ──────────────────────────────────────────────────────────────────────────
// 公共辅助
// ──────────────────────────────────────────────────────────────────────────

// createNotifTestUser 在测试 DB 中快速创建一个普通用户。
func createNotifTestUser(t *testing.T, username string) *model.User {
	t.Helper()
	u := &model.User{Username: username, Password: "x", Role: "user"}
	if err := model.DB(context.Background()).Create(u).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	return u
}

// createNotif 在测试 DB 中快速写入一条通知。
func createNotif(t *testing.T, userID, instanceID uint, category string, isRead bool) *model.Notification {
	t.Helper()
	n := &model.Notification{
		UserID:       userID,
		InstanceID:   instanceID,
		InstanceName: "inst",
		Type:         "instance_create_success",
		Category:     category,
		Title:        "t",
		Message:      "m",
		IsRead:       isRead,
	}
	if err := model.DB(context.Background()).Create(n).Error; err != nil {
		t.Fatalf("create notification: %v", err)
	}
	return n
}

// jsonBodyReqWithSession 构造一个带 session 的 JSON body 请求（body 为结构体自动序列化）。
func jsonBodyReqWithSession(t *testing.T, method, path, username string, body interface{}) *http.Request {
	t.Helper()

	var req *http.Request
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		req = httptest.NewRequest(method, path, bytes.NewReader(buf))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Accept", "application/json")

	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = username

	rr := httptest.NewRecorder()
	_ = session.Save(req, rr)
	for _, cookie := range rr.Result().Cookies() {
		req.AddCookie(cookie)
	}
	return req
}

// jsonRawBodyReqWithSession 构造一个带 session 和原始（可能非法 JSON）body 的请求。
func jsonRawBodyReqWithSession(t *testing.T, method, path, username, rawBody string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(rawBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = username

	rr := httptest.NewRecorder()
	_ = session.Save(req, rr)
	for _, cookie := range rr.Result().Cookies() {
		req.AddCookie(cookie)
	}
	return req
}

// ──────────────────────────────────────────────────────────────────────────
// HandleGetNotifications
// ──────────────────────────────────────────────────────────────────────────

func TestHandleGetNotifications_Unauthorized(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/openclaw/notifications", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleGetNotifications(rr, req)

	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未登录应 401/403，实际=%d", rr.Code)
	}
}

func TestHandleGetNotifications_InvalidIsRead(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	u := createNotifTestUser(t, "u-get-1")
	_ = u

	req := openclawReqWithSession(t, http.MethodGet,
		"/openclaw/notifications?is_read=yes", "u-get-1", "")
	rr := httptest.NewRecorder()
	HandleGetNotifications(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("is_read 非法应 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "is_read") {
		t.Errorf("错误信息应包含 is_read，实际=%s", rr.Body.String())
	}
}

func TestHandleGetNotifications_InvalidCategory(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	createNotifTestUser(t, "u-get-2")

	req := openclawReqWithSession(t, http.MethodGet,
		"/openclaw/notifications?category=bad", "u-get-2", "")
	rr := httptest.NewRecorder()
	HandleGetNotifications(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("category 非法应 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "category") {
		t.Errorf("错误信息应包含 category，实际=%s", rr.Body.String())
	}
}

func TestHandleGetNotifications_SuccessAll(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	u := createNotifTestUser(t, "u-get-3")
	createNotif(t, u.ID, 1, model.NotifCategorySuccess, false)
	createNotif(t, u.ID, 2, model.NotifCategoryError, true)
	createNotif(t, u.ID, 3, model.NotifCategoryNotice, false)

	req := openclawReqWithSession(t, http.MethodGet, "/openclaw/notifications", "u-get-3", "")
	rr := httptest.NewRecorder()
	HandleGetNotifications(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Notifications []model.Notification `json:"notifications"`
		Total         int64                `json:"total"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, rr.Body.String())
	}
	if resp.Total != 3 {
		t.Errorf("total 应 3，实际=%d", resp.Total)
	}
	if len(resp.Notifications) != 3 {
		t.Errorf("notifications 应 3 条，实际=%d", len(resp.Notifications))
	}
}

func TestHandleGetNotifications_FilterUnreadByCategory(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	u := createNotifTestUser(t, "u-get-4")
	createNotif(t, u.ID, 1, model.NotifCategorySuccess, false) // 未读 success
	createNotif(t, u.ID, 2, model.NotifCategoryError, false)   // 未读 error
	createNotif(t, u.ID, 3, model.NotifCategorySuccess, true)  // 已读 success

	req := openclawReqWithSession(t, http.MethodGet,
		"/openclaw/notifications?is_read=false&category=success", "u-get-4", "")
	rr := httptest.NewRecorder()
	HandleGetNotifications(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Total int64 `json:"total"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Total != 1 {
		t.Errorf("未读 success 应只有 1 条，实际=%d", resp.Total)
	}
}

// TestHandleGetNotifications_DBError 通过删除表制造 DB 错误，覆盖 500 分支。
func TestHandleGetNotifications_DBError(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	createNotifTestUser(t, "u-get-err")
	if err := model.DB(context.Background()).Migrator().DropTable(&model.Notification{}); err != nil {
		t.Fatalf("drop table: %v", err)
	}

	req := openclawReqWithSession(t, http.MethodGet, "/openclaw/notifications", "u-get-err", "")
	rr := httptest.NewRecorder()
	HandleGetNotifications(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("DB 错误应 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ──────────────────────────────────────────────────────────────────────────
// HandleReadNotification
// ──────────────────────────────────────────────────────────────────────────

func TestHandleReadNotification_Unauthorized(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/openclaw/notifications/read", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleReadNotification(rr, req)

	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未登录应 401/403，实际=%d", rr.Code)
	}
}

func TestHandleReadNotification_BadBody(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	createNotifTestUser(t, "u-read-1")

	req := jsonRawBodyReqWithSession(t, http.MethodPost,
		"/openclaw/notifications/read", "u-read-1", "not-json")
	rr := httptest.NewRecorder()
	HandleReadNotification(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("非法 JSON 应 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleReadNotification_InvalidCategory(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	createNotifTestUser(t, "u-read-2")

	req := jsonBodyReqWithSession(t, http.MethodPost,
		"/openclaw/notifications/read", "u-read-2",
		map[string]interface{}{"id": 0, "category": "wrong"})
	rr := httptest.NewRecorder()
	HandleReadNotification(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("非法 category 应 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleReadNotification_SingleSuccess(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	u := createNotifTestUser(t, "u-read-3")
	n := createNotif(t, u.ID, 1, model.NotifCategorySuccess, false)

	req := jsonBodyReqWithSession(t, http.MethodPost,
		"/openclaw/notifications/read", "u-read-3",
		map[string]interface{}{"id": n.ID})
	rr := httptest.NewRecorder()
	HandleReadNotification(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	var fresh model.Notification
	model.DB(context.Background()).First(&fresh, n.ID)
	if !fresh.IsRead {
		t.Errorf("通知应变为已读")
	}
}

func TestHandleReadNotification_AllSuccess(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	u := createNotifTestUser(t, "u-read-4")
	createNotif(t, u.ID, 1, model.NotifCategorySuccess, false)
	createNotif(t, u.ID, 2, model.NotifCategoryError, false)
	createNotif(t, u.ID, 3, model.NotifCategoryNotice, false)

	// 按 category 全部已读：仅 error 被标记
	req := jsonBodyReqWithSession(t, http.MethodPost,
		"/openclaw/notifications/read", "u-read-4",
		map[string]interface{}{"id": 0, "category": model.NotifCategoryError})
	rr := httptest.NewRecorder()
	HandleReadNotification(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	var unread int64
	model.DB(context.Background()).Model(&model.Notification{}).
		Where("user_id = ? AND is_read = ?", u.ID, false).Count(&unread)
	if unread != 2 {
		t.Errorf("仅 error 应变已读，剩 2 条未读，实际=%d", unread)
	}
}

// TestHandleReadNotification_SingleDBError 通过删除 notifications 表制造 DB 错误，
// 覆盖 MarkNotificationRead 失败分支（500）。
func TestHandleReadNotification_SingleDBError(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	createNotifTestUser(t, "u-read-err1")
	// 移除底层表
	if err := model.DB(context.Background()).Migrator().DropTable(&model.Notification{}); err != nil {
		t.Fatalf("drop table: %v", err)
	}

	req := jsonBodyReqWithSession(t, http.MethodPost,
		"/openclaw/notifications/read", "u-read-err1",
		map[string]interface{}{"id": 1})
	rr := httptest.NewRecorder()
	HandleReadNotification(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("DB 错误应 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// TestHandleReadNotification_AllDBError 覆盖 MarkAllNotificationsRead 失败分支。
func TestHandleReadNotification_AllDBError(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	createNotifTestUser(t, "u-read-err2")
	if err := model.DB(context.Background()).Migrator().DropTable(&model.Notification{}); err != nil {
		t.Fatalf("drop table: %v", err)
	}

	req := jsonBodyReqWithSession(t, http.MethodPost,
		"/openclaw/notifications/read", "u-read-err2",
		map[string]interface{}{"id": 0})
	rr := httptest.NewRecorder()
	HandleReadNotification(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("DB 错误应 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ──────────────────────────────────────────────────────────────────────────
// HandleGetUnreadCount
// ──────────────────────────────────────────────────────────────────────────

func TestHandleGetUnreadCount_Unauthorized(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/openclaw/notifications/count", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleGetUnreadCount(rr, req)

	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未登录应 401/403，实际=%d", rr.Code)
	}
}

func TestHandleGetUnreadCount_Success(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	u := createNotifTestUser(t, "u-cnt-1")
	createNotif(t, u.ID, 1, model.NotifCategorySuccess, false)
	createNotif(t, u.ID, 2, model.NotifCategorySuccess, false)
	createNotif(t, u.ID, 3, model.NotifCategoryError, false)
	createNotif(t, u.ID, 4, model.NotifCategoryNotice, true) // 已读，不计入

	req := openclawReqWithSession(t, http.MethodGet,
		"/openclaw/notifications/count", "u-cnt-1", "")
	rr := httptest.NewRecorder()
	HandleGetUnreadCount(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		UnreadCount      int64            `json:"unread_count"`
		UnreadByCategory map[string]int64 `json:"unread_by_category"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, rr.Body.String())
	}
	if resp.UnreadCount != 3 {
		t.Errorf("unread_count 应 3，实际=%d", resp.UnreadCount)
	}
	if resp.UnreadByCategory[model.NotifCategorySuccess] != 2 {
		t.Errorf("success 未读应 2，实际=%d", resp.UnreadByCategory[model.NotifCategorySuccess])
	}
	if resp.UnreadByCategory[model.NotifCategoryError] != 1 {
		t.Errorf("error 未读应 1，实际=%d", resp.UnreadByCategory[model.NotifCategoryError])
	}
}

// TestHandleGetUnreadCount_DBError 覆盖 DB 错误分支。
func TestHandleGetUnreadCount_DBError(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	createNotifTestUser(t, "u-cnt-err")
	if err := model.DB(context.Background()).Migrator().DropTable(&model.Notification{}); err != nil {
		t.Fatalf("drop table: %v", err)
	}

	req := openclawReqWithSession(t, http.MethodGet,
		"/openclaw/notifications/count", "u-cnt-err", "")
	rr := httptest.NewRecorder()
	HandleGetUnreadCount(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("DB 错误应 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ──────────────────────────────────────────────────────────────────────────
// HandleDeleteNotification
// ──────────────────────────────────────────────────────────────────────────

func TestHandleDeleteNotification_Unauthorized(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/openclaw/notifications/delete", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleDeleteNotification(rr, req)

	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未登录应 401/403，实际=%d", rr.Code)
	}
}

func TestHandleDeleteNotification_MethodNotAllowed(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	createNotifTestUser(t, "u-del-0")

	req := openclawReqWithSession(t, http.MethodGet,
		"/openclaw/notifications/delete", "u-del-0", "")
	rr := httptest.NewRecorder()
	HandleDeleteNotification(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 应 405，实际=%d", rr.Code)
	}
}

func TestHandleDeleteNotification_BadBody(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	createNotifTestUser(t, "u-del-1")

	req := jsonRawBodyReqWithSession(t, http.MethodPost,
		"/openclaw/notifications/delete", "u-del-1", "not-json")
	rr := httptest.NewRecorder()
	HandleDeleteNotification(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("非法 JSON 应 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleDeleteNotification_InvalidCategory(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	createNotifTestUser(t, "u-del-2")

	req := jsonBodyReqWithSession(t, http.MethodPost,
		"/openclaw/notifications/delete", "u-del-2",
		map[string]interface{}{"category": "bad"})
	rr := httptest.NewRecorder()
	HandleDeleteNotification(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("非法 category 应 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleDeleteNotification_Single(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	u := createNotifTestUser(t, "u-del-3")
	n := createNotif(t, u.ID, 1, model.NotifCategorySuccess, false)

	req := jsonBodyReqWithSession(t, http.MethodPost,
		"/openclaw/notifications/delete", "u-del-3",
		map[string]interface{}{"id": n.ID})
	rr := httptest.NewRecorder()
	HandleDeleteNotification(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Deleted int64 `json:"deleted"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Deleted != 1 {
		t.Errorf("deleted 应 1，实际=%d", resp.Deleted)
	}
	var cnt int64
	model.DB(context.Background()).Unscoped().Model(&model.Notification{}).
		Where("id = ?", n.ID).Count(&cnt)
	if cnt != 0 {
		t.Errorf("通知应被物理删除")
	}
}

func TestHandleDeleteNotification_BatchDedup(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	u := createNotifTestUser(t, "u-del-4")
	n1 := createNotif(t, u.ID, 1, model.NotifCategoryError, false)
	n2 := createNotif(t, u.ID, 2, model.NotifCategoryError, false)

	// 包含 0、重复、非法 ID
	req := jsonBodyReqWithSession(t, http.MethodPost,
		"/openclaw/notifications/delete", "u-del-4",
		map[string]interface{}{"ids": []uint{0, n1.ID, n1.ID, n2.ID}})
	rr := httptest.NewRecorder()
	HandleDeleteNotification(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Deleted int64 `json:"deleted"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Deleted != 2 {
		t.Errorf("deleted 应 2，实际=%d", resp.Deleted)
	}
}

func TestHandleDeleteNotification_BatchAllZero(t *testing.T) {
	// ids 全为 0：去重后为空，应直接返回 deleted=0
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	createNotifTestUser(t, "u-del-5")

	req := jsonBodyReqWithSession(t, http.MethodPost,
		"/openclaw/notifications/delete", "u-del-5",
		map[string]interface{}{"ids": []uint{0, 0}})
	rr := httptest.NewRecorder()
	HandleDeleteNotification(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Deleted int64 `json:"deleted"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Deleted != 0 {
		t.Errorf("deleted 应 0，实际=%d", resp.Deleted)
	}
}

func TestHandleDeleteNotification_BatchTooMany(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	createNotifTestUser(t, "u-del-6")

	ids := make([]uint, 0, 101)
	for i := 1; i <= 101; i++ {
		ids = append(ids, uint(i))
	}
	req := jsonBodyReqWithSession(t, http.MethodPost,
		"/openclaw/notifications/delete", "u-del-6",
		map[string]interface{}{"ids": ids})
	rr := httptest.NewRecorder()
	HandleDeleteNotification(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("超 100 应 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleDeleteNotification_DeleteAll(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	u := createNotifTestUser(t, "u-del-7")
	createNotif(t, u.ID, 1, model.NotifCategorySuccess, false)
	createNotif(t, u.ID, 2, model.NotifCategoryError, true)
	createNotif(t, u.ID, 3, model.NotifCategoryNotice, false)

	// id=0 且 ids 空 + 无 category => 全部删除
	req := jsonBodyReqWithSession(t, http.MethodPost,
		"/openclaw/notifications/delete", "u-del-7",
		map[string]interface{}{})
	rr := httptest.NewRecorder()
	HandleDeleteNotification(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Deleted int64 `json:"deleted"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Deleted != 3 {
		t.Errorf("deleted 应 3，实际=%d", resp.Deleted)
	}
}

func TestHandleDeleteNotification_DeleteAllByCategory(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	u := createNotifTestUser(t, "u-del-8")
	createNotif(t, u.ID, 1, model.NotifCategorySuccess, false)
	createNotif(t, u.ID, 2, model.NotifCategoryError, false)
	createNotif(t, u.ID, 3, model.NotifCategoryError, true)

	req := jsonBodyReqWithSession(t, http.MethodPost,
		"/openclaw/notifications/delete", "u-del-8",
		map[string]interface{}{"category": model.NotifCategoryError})
	rr := httptest.NewRecorder()
	HandleDeleteNotification(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Deleted int64 `json:"deleted"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Deleted != 2 {
		t.Errorf("按 error 类别删除应 2，实际=%d", resp.Deleted)
	}
	var left int64
	model.DB(context.Background()).Unscoped().Model(&model.Notification{}).
		Where("user_id = ?", u.ID).Count(&left)
	if left != 1 {
		t.Errorf("剩余应 1（success），实际=%d", left)
	}
}

// TestHandleDeleteNotification_DBError 覆盖删除时 DB 失败的 500 分支。
func TestHandleDeleteNotification_DBError(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	createNotifTestUser(t, "u-del-err")
	if err := model.DB(context.Background()).Migrator().DropTable(&model.Notification{}); err != nil {
		t.Fatalf("drop table: %v", err)
	}

	req := jsonBodyReqWithSession(t, http.MethodPost,
		"/openclaw/notifications/delete", "u-del-err",
		map[string]interface{}{"id": 1})
	rr := httptest.NewRecorder()
	HandleDeleteNotification(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("DB 错误应 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ──────────────────────────────────────────────────────────────────────────
// HandleServiceStatus 补充测试（沿用 guards_test 已有 2 个用例，这里补 400 分支）
// ──────────────────────────────────────────────────────────────────────────

func TestHandleServiceStatus_InstanceNotFound(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	createNotifTestUser(t, "u-svc-1")

	// 不存在的 id
	req := openclawReqWithSession(t, http.MethodGet,
		"/openclaw/service-status?id=9999", "u-svc-1", "")
	rr := httptest.NewRecorder()
	HandleServiceStatus(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("实例不存在应 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// 注：RunScript 需要真实 TAT 客户端，单元测试环境无法走到成功路径；
// 已在 openclaw_handler_guards_test.go 中覆盖 Unauthorized / UnknownAgentType 分支，
// 这里仅补充 InstanceNotFound 400 分支。成功路径依赖 TAT，属于集成测试范畴。

// TestHandleServiceStatus_RunScriptFailed 通过 mock runScriptForServiceStatusFn
// 覆盖 RunScript 返回 error 的 500 分支。
func TestHandleServiceStatus_RunScriptFailed(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	orig := runScriptForServiceStatusFn
	runScriptForServiceStatusFn = func(ctx context.Context, instanceId, scriptName string, timeout uint64,
		runtimeUser string, onOutput func(chunk string), params map[string]string,
	) (string, error) {
		return "", hcommon.I18nError(i18n.MsgTATFailed)
	}
	defer func() { runScriptForServiceStatusFn = orig }()

	u := createNotifTestUser(t, "u-svc-2")
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-svc-2",
		UserID: u.ID, AgentType: model.AgentTypeOpenClaw,
	}
	if err := model.DB(context.Background()).Create(inst).Error; err != nil {
		t.Fatalf("create inst: %v", err)
	}

	req := openclawReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/service-status?id=%d", inst.ID), "u-svc-2", "")
	rr := httptest.NewRecorder()
	HandleServiceStatus(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("RunScript 失败应 500，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// TestHandleServiceStatus_Success 通过 mock runScriptForServiceStatusFn
// 覆盖 RunScript 成功后写出 JSON 响应的分支。
func TestHandleServiceStatus_Success(t *testing.T) {
	cleanup := initOpenClawHandlerTestDB(t)
	defer cleanup()

	orig := runScriptForServiceStatusFn
	runScriptForServiceStatusFn = func(ctx context.Context, instanceId, scriptName string, timeout uint64,
		runtimeUser string, onOutput func(chunk string), params map[string]string,
	) (string, error) {
		return `{"ok":true,"running":true}`, nil
	}
	defer func() { runScriptForServiceStatusFn = orig }()

	u := createNotifTestUser(t, "u-svc-3")
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-svc-3",
		UserID: u.ID, AgentType: model.AgentTypeOpenClaw,
	}
	if err := model.DB(context.Background()).Create(inst).Error; err != nil {
		t.Fatalf("create inst: %v", err)
	}

	req := openclawReqWithSession(t, http.MethodGet,
		fmt.Sprintf("/openclaw/service-status?id=%d", inst.ID), "u-svc-3", "")
	rr := httptest.NewRecorder()
	HandleServiceStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "running") {
		t.Errorf("响应应包含脚本输出，实际=%s", rr.Body.String())
	}
}
