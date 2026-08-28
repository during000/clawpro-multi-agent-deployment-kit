package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	hcommon "hatchery/common"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// ─── 测试 DB 初始化 ───────────────────────────────────────────────────────────

func initInstanceIDParamTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.User{}, &model.Instance{}, &model.AIImage{}, &model.SiteConfig{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	origDB := model.UseDBForTest(db)
	AdminToken = "test-admin-token"
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	return func() {
		origDB()
		Store = origStore
	}
}

// adminInstanceIDReq 构造带 admin token 的 form 请求。
func adminInstanceIDReq(method, path, body string) *http.Request {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req
}

// adminInstanceIDReqJSON 构造带 admin token 的 JSON 请求。
func adminInstanceIDReqJSON(method, path string, body []byte) *http.Request {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req
}

// userInstanceIDReqWithSession 构造带用户 session 的请求。
func userInstanceIDReqWithSession(t *testing.T, method, path, username string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
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

// ─── parseAdminDeleteRequest: instance_id / instance_ids 新路径 ───────────────

// TestParseAdminDeleteRequest_FormInstanceID form 传 instance_id 单删。
func TestParseAdminDeleteRequest_FormInstanceID(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	// 先创建实例
	inst := &model.Instance{Name: "test", InstanceId: "ins-abc123"}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("instance_id", "ins-abc123")
	req := httptest.NewRequest(http.MethodPost, "/admin/instances/delete", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ids, isBatch, err := parseAdminDeleteRequest(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if isBatch {
		t.Fatal("form instance_id 不应走 batch 分支")
	}
	if len(ids) != 1 || ids[0] != inst.ID {
		t.Fatalf("ids 错: %v, 期望 [%d]", ids, inst.ID)
	}
}

// TestParseAdminDeleteRequest_FormInstanceID_NotFound form instance_id 不存在应报错。
func TestParseAdminDeleteRequest_FormInstanceID_NotFound(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	form := url.Values{}
	form.Set("instance_id", "ins-notexist")
	req := httptest.NewRequest(http.MethodPost, "/admin/instances/delete", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	_, _, err := parseAdminDeleteRequest(req)
	if err == nil {
		t.Fatal("instance_id 不存在应报错")
	}
	if !strings.Contains(hcommon.ErrorMessageWithCtx(req.Context(), err), "不存在") {
		t.Fatalf("错误消息应含 '不存在'，实际: %v", err)
	}
}

// TestParseAdminDeleteRequest_JSONInstanceID JSON body 传 instance_id 单删。
func TestParseAdminDeleteRequest_JSONInstanceID(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	inst := &model.Instance{Name: "test", InstanceId: "ins-json-single"}
	model.DB(context.Background()).Create(inst)

	req := httptest.NewRequest(http.MethodPost, "/admin/instances/delete",
		strings.NewReader(`{"instance_id": "ins-json-single"}`))
	req.Header.Set("Content-Type", "application/json")

	ids, isBatch, err := parseAdminDeleteRequest(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if isBatch {
		t.Fatal("JSON instance_id 不应走 batch 分支")
	}
	if len(ids) != 1 || ids[0] != inst.ID {
		t.Fatalf("ids 错: %v", ids)
	}
}

// TestParseAdminDeleteRequest_JSONInstanceIDs JSON body 传 instance_ids 走批量。
func TestParseAdminDeleteRequest_JSONInstanceIDs(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	inst1 := &model.Instance{Name: "inst1", InstanceId: "ins-batch-1"}
	inst2 := &model.Instance{Name: "inst2", InstanceId: "ins-batch-2"}
	model.DB(context.Background()).Create(inst1)
	model.DB(context.Background()).Create(inst2)

	body, _ := json.Marshal(map[string]interface{}{
		"instance_ids": []string{"ins-batch-1", "ins-batch-2"},
	})
	req := httptest.NewRequest(http.MethodPost, "/admin/instances/delete", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ids, isBatch, err := parseAdminDeleteRequest(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !isBatch {
		t.Fatal("instance_ids 应走 batch 分支")
	}
	if len(ids) != 2 {
		t.Fatalf("期望 2 个 id，实际: %v", ids)
	}
}

// TestParseAdminDeleteRequest_JSONInstanceIDs_Empty instance_ids=[] 应报错。
func TestParseAdminDeleteRequest_JSONInstanceIDs_Empty(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/admin/instances/delete",
		strings.NewReader(`{"instance_ids": []}`))
	req.Header.Set("Content-Type", "application/json")

	_, isBatch, err := parseAdminDeleteRequest(req)
	if err == nil {
		t.Fatal("instance_ids=[] 应报错")
	}
	if !isBatch {
		t.Fatal("instance_ids 字段存在应走 batch 分支")
	}
	if !strings.Contains(hcommon.ErrorMessageWithCtx(req.Context(), err), "空列表") {
		t.Fatalf("错误消息应含 '空列表'，实际: %v", err)
	}
}

// TestParseAdminDeleteRequest_JSONInstanceIDs_TooMany instance_ids > 100 应报错。
func TestParseAdminDeleteRequest_JSONInstanceIDs_TooMany(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	ids := make([]string, 101)
	for i := range ids {
		ids[i] = "ins-x"
	}
	body, _ := json.Marshal(map[string]interface{}{"instance_ids": ids})
	req := httptest.NewRequest(http.MethodPost, "/admin/instances/delete", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	_, _, err := parseAdminDeleteRequest(req)
	if err == nil {
		t.Fatal("instance_ids>100 应报错")
	}
	if !strings.Contains(hcommon.ErrorMessageWithCtx(req.Context(), err), "上限") {
		t.Fatalf("错误消息应含 '上限'，实际: %v", err)
	}
}

// TestParseAdminDeleteRequest_JSONInstanceIDs_NotFound instance_ids 全不存在应报错。
func TestParseAdminDeleteRequest_JSONInstanceIDs_NotFound(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{
		"instance_ids": []string{"ins-notexist-1", "ins-notexist-2"},
	})
	req := httptest.NewRequest(http.MethodPost, "/admin/instances/delete", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	_, _, err := parseAdminDeleteRequest(req)
	if err == nil {
		t.Fatal("instance_ids 全不存在应报错")
	}
	if !strings.Contains(hcommon.ErrorMessageWithCtx(req.Context(), err), "不存在") {
		t.Fatalf("错误消息应含 '不存在'，实际: %v", err)
	}
}

// TestParseAdminDeleteRequest_IDsPriorityOverInstanceIDs ids 优先于 instance_ids。
func TestParseAdminDeleteRequest_IDsPriorityOverInstanceIDs(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{
		"ids":          []uint{1, 2},
		"instance_ids": []string{"ins-should-be-ignored"},
	})
	req := httptest.NewRequest(http.MethodPost, "/admin/instances/delete", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ids, isBatch, err := parseAdminDeleteRequest(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !isBatch {
		t.Fatal("应走 batch 分支")
	}
	// ids 优先，instance_ids 被忽略
	if len(ids) != 2 || ids[0] != 1 || ids[1] != 2 {
		t.Fatalf("ids 应优先，实际: %v", ids)
	}
}

// ─── getAdminInstanceByIDOrInstanceID ────────────────────────────────────────

// TestGetAdminInstanceByIDOrInstanceID_ByInstanceID 通过 instance_id 查询。
func TestGetAdminInstanceByIDOrInstanceID_ByInstanceID(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	inst := &model.Instance{Name: "test", InstanceId: "ins-query-test"}
	model.DB(context.Background()).Create(inst)

	req := adminInstanceIDReq(http.MethodPost, "/admin/instances/start?instance_id=ins-query-test", "")
	result, err := getAdminInstanceByIDOrInstanceID(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if result.InstanceId != "ins-query-test" {
		t.Errorf("期望 ins-query-test，实际 %s", result.InstanceId)
	}
}

// TestGetAdminInstanceByIDOrInstanceID_ByID 通过 id 查询。
func TestGetAdminInstanceByIDOrInstanceID_ByID(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	inst := &model.Instance{Name: "test", InstanceId: "ins-by-id"}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("id", "1")
	req := httptest.NewRequest(http.MethodPost, "/admin/instances/start", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer test-admin-token")

	result, err := getAdminInstanceByIDOrInstanceID(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if result.ID != inst.ID {
		t.Errorf("期望 ID=%d，实际 %d", inst.ID, result.ID)
	}
}

// TestGetAdminInstanceByIDOrInstanceID_IDPriority id 优先于 instance_id。
func TestGetAdminInstanceByIDOrInstanceID_IDPriority(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	inst1 := &model.Instance{Name: "inst1", InstanceId: "ins-priority-1"}
	inst2 := &model.Instance{Name: "inst2", InstanceId: "ins-priority-2"}
	model.DB(context.Background()).Create(inst1)
	model.DB(context.Background()).Create(inst2)

	// 同时传 id 和 instance_id，id 优先
	form := url.Values{}
	form.Set("id", "1")
	form.Set("instance_id", "ins-priority-2")
	req := httptest.NewRequest(http.MethodPost, "/admin/instances/start", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	result, err := getAdminInstanceByIDOrInstanceID(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	// id=1 对应 inst1
	if result.InstanceId != "ins-priority-1" {
		t.Errorf("id 应优先，期望 ins-priority-1，实际 %s", result.InstanceId)
	}
}

// TestGetAdminInstanceByIDOrInstanceID_InvalidID 无效 id 应报错。
func TestGetAdminInstanceByIDOrInstanceID_InvalidID(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	form := url.Values{}
	form.Set("id", "abc")
	req := httptest.NewRequest(http.MethodPost, "/admin/instances/start", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	_, err := getAdminInstanceByIDOrInstanceID(req)
	if err == nil {
		t.Fatal("无效 id 应报错")
	}
	if !strings.Contains(hcommon.ErrorMessageWithCtx(req.Context(), err), "无效") {
		t.Fatalf("错误消息应含 '无效'，实际: %v", err)
	}
}

// TestGetAdminInstanceByIDOrInstanceID_InstanceIDNotFound instance_id 不存在应报错。
func TestGetAdminInstanceByIDOrInstanceID_InstanceIDNotFound(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	form := url.Values{}
	form.Set("instance_id", "ins-notexist")
	req := httptest.NewRequest(http.MethodPost, "/admin/instances/start", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	_, err := getAdminInstanceByIDOrInstanceID(req)
	if err == nil {
		t.Fatal("instance_id 不存在应报错")
	}
	if !strings.Contains(err.Error(), "不存在") {
		t.Fatalf("错误消息应含 '不存在'，实际: %v", err)
	}
}

// TestGetAdminInstanceByIDOrInstanceID_NeitherParam 两者都不传应报错。
func TestGetAdminInstanceByIDOrInstanceID_NeitherParam(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/admin/instances/start", nil)

	_, err := getAdminInstanceByIDOrInstanceID(req)
	if err == nil {
		t.Fatal("两者都不传应报错")
	}
	if !strings.Contains(hcommon.ErrorMessageWithCtx(context.Background(), err), "缺少参数") {
		t.Fatalf("错误消息应含 '缺少参数'，实际: %v", err)
	}
}

// TestGetAdminInstanceByIDOrInstanceID_QueryParam 通过 URL query 参数传 instance_id。
func TestGetAdminInstanceByIDOrInstanceID_QueryParam(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	inst := &model.Instance{Name: "test", InstanceId: "ins-query-param"}
	model.DB(context.Background()).Create(inst)

	req := httptest.NewRequest(http.MethodGet, "/admin/instances/status?instance_id=ins-query-param", nil)
	result, err := getAdminInstanceByIDOrInstanceID(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if result.InstanceId != "ins-query-param" {
		t.Errorf("期望 ins-query-param，实际 %s", result.InstanceId)
	}
}

// ─── handleAdminStartInstance: instance_id 参数错误路径 ──────────────────────

// TestHandleAdminStartInstance_MissingParam 缺少 id 和 instance_id 应返回 400。
func TestHandleAdminStartInstance_MissingParam(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	req := adminInstanceIDReq(http.MethodPost, "/admin/instances/start", "")
	rr := httptest.NewRecorder()
	handleAdminStartInstance(rr, req, testCVMFetcher)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("缺少参数应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// TestHandleAdminStartInstance_ByInstanceID_NotFound instance_id 不存在应返回 404。
func TestHandleAdminStartInstance_ByInstanceID_NotFound(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	form := url.Values{}
	form.Set("instance_id", "ins-notexist")
	req := httptest.NewRequest(http.MethodPost, "/admin/instances/start", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer test-admin-token")

	rr := httptest.NewRecorder()
	handleAdminStartInstance(rr, req, testCVMFetcher)
	if rr.Code != http.StatusNotFound {
		t.Errorf("instance_id 不存在应返回 404，实际=%d", rr.Code)
	}
}

// ─── handleAdminStopInstance: instance_id 参数错误路径 ───────────────────────

// TestHandleAdminStopInstance_MissingParam 缺少 id 和 instance_id 应返回 400。
func TestHandleAdminStopInstance_MissingParam(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	req := adminInstanceIDReq(http.MethodPost, "/admin/instances/stop", "")
	rr := httptest.NewRecorder()
	handleAdminStopInstance(rr, req, testCVMFetcher)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("缺少参数应返回 400，实际=%d", rr.Code)
	}
}

// TestHandleAdminStopInstance_ByInstanceID_NotFound instance_id 不存在应返回 404。
func TestHandleAdminStopInstance_ByInstanceID_NotFound(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	form := url.Values{}
	form.Set("instance_id", "ins-notexist")
	req := httptest.NewRequest(http.MethodPost, "/admin/instances/stop", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer test-admin-token")

	rr := httptest.NewRecorder()
	handleAdminStopInstance(rr, req, testCVMFetcher)
	if rr.Code != http.StatusNotFound {
		t.Errorf("instance_id 不存在应返回 404，实际=%d", rr.Code)
	}
}

// ─── handleAdminRebootInstance: instance_id 参数错误路径 ─────────────────────

// TestHandleAdminRebootInstance_MissingParam 缺少 id 和 instance_id 应返回 400。
func TestHandleAdminRebootInstance_MissingParam(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	req := adminInstanceIDReq(http.MethodPost, "/admin/instances/reboot", "")
	rr := httptest.NewRecorder()
	handleAdminRebootInstance(rr, req, testCVMFetcher)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("缺少参数应返回 400，实际=%d", rr.Code)
	}
}

// TestHandleAdminRebootInstance_ByInstanceID_NotFound instance_id 不存在应返回 404。
func TestHandleAdminRebootInstance_ByInstanceID_NotFound(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	form := url.Values{}
	form.Set("instance_id", "ins-notexist")
	req := httptest.NewRequest(http.MethodPost, "/admin/instances/reboot", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer test-admin-token")

	rr := httptest.NewRecorder()
	handleAdminRebootInstance(rr, req, testCVMFetcher)
	if rr.Code != http.StatusNotFound {
		t.Errorf("instance_id 不存在应返回 404，实际=%d", rr.Code)
	}
}

// ─── handleAdminResetInstance: instance_id 参数错误路径 ──────────────────────

// TestHandleAdminResetInstance_MissingParam 缺少 id 和 instance_id 应返回 400。
func TestHandleAdminResetInstance_MissingParam(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	req := adminInstanceIDReq(http.MethodPost, "/admin/instances/reset", "")
	rr := httptest.NewRecorder()
	handleAdminResetInstance(rr, req, testCVMFetcher)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("缺少参数应返回 400，实际=%d", rr.Code)
	}
}

// TestHandleAdminResetInstance_ByInstanceID_NotFound instance_id 不存在应返回 404。
func TestHandleAdminResetInstance_ByInstanceID_NotFound(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	form := url.Values{}
	form.Set("instance_id", "ins-notexist")
	req := httptest.NewRequest(http.MethodPost, "/admin/instances/reset", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer test-admin-token")

	rr := httptest.NewRecorder()
	handleAdminResetInstance(rr, req, testCVMFetcher)
	if rr.Code != http.StatusNotFound {
		t.Errorf("instance_id 不存在应返回 404，实际=%d", rr.Code)
	}
}

// ─── HandleAdminRefreshInstanceVersion: instance_id 参数错误路径 ─────────────

// TestHandleAdminRefreshInstanceVersion_MissingParam 缺少 id 和 instance_id 应返回 400。
func TestHandleAdminRefreshInstanceVersion_MissingParam(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	req := adminInstanceIDReq(http.MethodPost, "/admin/instances/refresh-version", "")
	rr := httptest.NewRecorder()
	HandleAdminRefreshInstanceVersion(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("缺少参数应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// TestHandleAdminRefreshInstanceVersion_ByInstanceID_NotFound instance_id 不存在应返回 404。
func TestHandleAdminRefreshInstanceVersion_ByInstanceID_NotFound(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	form := url.Values{}
	form.Set("instance_id", "ins-notexist")
	req := httptest.NewRequest(http.MethodPost, "/admin/instances/refresh-version", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer test-admin-token")

	rr := httptest.NewRecorder()
	HandleAdminRefreshInstanceVersion(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("instance_id 不存在应返回 404，实际=%d", rr.Code)
	}
}

// ─── HandleAdminBatchUpgrade: instance_ids 新路径 ────────────────────────────

// TestHandleAdminBatchUpgrade_InstanceIDs_Empty instance_ids 为空应报错。
func TestHandleAdminBatchUpgrade_InstanceIDs_Empty(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{"instance_ids": []string{}})
	req := adminInstanceIDReqJSON(http.MethodPost, "/admin/instances/batch-upgrade", body)
	rr := httptest.NewRecorder()
	HandleAdminBatchUpgrade(rr, req)
	// instance_ids 为空列表，走 else 分支（缺少参数）
	if rr.Code != http.StatusBadRequest {
		t.Errorf("instance_ids=[] 应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// TestHandleAdminBatchUpgrade_InstanceIDs_TooMany instance_ids > 20 应报错。
func TestHandleAdminBatchUpgrade_InstanceIDs_TooMany(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	ids := make([]string, 21)
	for i := range ids {
		ids[i] = "ins-x"
	}
	body, _ := json.Marshal(map[string]interface{}{"instance_ids": ids})
	req := adminInstanceIDReqJSON(http.MethodPost, "/admin/instances/batch-upgrade", body)
	rr := httptest.NewRecorder()
	HandleAdminBatchUpgrade(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("instance_ids>20 应返回 400，实际=%d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "20") {
		t.Errorf("错误消息应含 '20'，实际: %s", rr.Body.String())
	}
}

// TestHandleAdminBatchUpgrade_MissingBothParams 两者都不传应报错。
func TestHandleAdminBatchUpgrade_MissingBothParams(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{})
	req := adminInstanceIDReqJSON(http.MethodPost, "/admin/instances/batch-upgrade", body)
	rr := httptest.NewRecorder()
	HandleAdminBatchUpgrade(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("两者都不传应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "缺少参数") {
		t.Errorf("错误消息应含 '缺少参数'，实际: %s", rr.Body.String())
	}
}

// TestHandleAdminBatchUpgrade_InstanceIDs_Found instance_ids 存在时正常查询到实例。
func TestHandleAdminBatchUpgrade_InstanceIDs_Found(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	// 创建实例（使用官方公共镜像，避免非官方镜像检查失败）
	inst := &model.Instance{Name: "inst1", InstanceId: "ins-upgrade-1", AgentType: "openclaw"}
	model.DB(context.Background()).Create(inst)

	// 创建启用的镜像（公共镜像）
	img := &model.AIImage{
		ImageId:   "img-public",
		ImageName: "Public Image",
		ImageType: "PUBLIC_IMAGE",
		AgentType: "openclaw",
		Enabled:   true,
	}
	model.DB(context.Background()).Create(img)

	body, _ := json.Marshal(map[string]interface{}{
		"instance_ids": []string{"ins-upgrade-1"},
	})
	req := adminInstanceIDReqJSON(http.MethodPost, "/admin/instances/batch-upgrade", body)
	rr := httptest.NewRecorder()
	HandleAdminBatchUpgrade(rr, req)
	// 实例没有 CVM instance_id 绑定，会在后续步骤失败，但参数解析应通过（不是 400 缺少参数）
	if rr.Code == http.StatusBadRequest {
		body := rr.Body.String()
		if strings.Contains(body, "缺少参数") {
			t.Errorf("instance_ids 存在时不应报缺少参数，实际: %s", body)
		}
	}
}

// ─── HandleAdminDetectInstall: instance_id / instance_ids 新路径 ─────────────

// TestHandleAdminDetectInstall_ByInstanceID_NotFound instance_id 不存在时返回
// 200 + 空 results（与批量模式 ids/instance_ids 行为一致），而非 404。
func TestHandleAdminDetectInstall_ByInstanceID_NotFound(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	form := url.Values{}
	form.Set("instance_id", "ins-notexist")
	req := httptest.NewRequest(http.MethodPost, "/admin/instances/detect-install", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer test-admin-token")

	rr := httptest.NewRecorder()
	HandleAdminDetectInstall(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("instance_id 不存在应返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "\"results\"") {
		t.Errorf("响应体应包含 results 字段，实际=%s", rr.Body.String())
	}
}

// TestHandleAdminDetectInstall_ByInstanceID_Found instance_id 存在时正常进入后续逻辑。
func TestHandleAdminDetectInstall_ByInstanceID_Found(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	inst := &model.Instance{Name: "test", InstanceId: "ins-detect-1", AgentType: "openclaw"}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("instance_id", "ins-detect-1")
	req := httptest.NewRequest(http.MethodPost, "/admin/instances/detect-install", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer test-admin-token")

	rr := httptest.NewRecorder()
	HandleAdminDetectInstall(rr, req)
	// 实例没有 CVM，后续 TAT 调用会失败，但参数解析应通过（不是 404 实例不存在）
	if rr.Code == http.StatusNotFound {
		t.Errorf("instance_id 存在时不应返回 404，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// TestHandleAdminDetectInstall_ByInstanceIDs_JSON instance_ids JSON body 路径。
func TestHandleAdminDetectInstall_ByInstanceIDs_JSON(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	inst := &model.Instance{Name: "test", InstanceId: "ins-detect-batch-1", AgentType: "openclaw"}
	model.DB(context.Background()).Create(inst)

	body, _ := json.Marshal(map[string]interface{}{
		"instance_ids": []string{"ins-detect-batch-1"},
	})
	req := adminInstanceIDReqJSON(http.MethodPost, "/admin/instances/detect-install", body)
	rr := httptest.NewRecorder()
	HandleAdminDetectInstall(rr, req)
	// 参数解析应通过，不应返回 400 缺少参数
	if rr.Code == http.StatusBadRequest {
		if strings.Contains(rr.Body.String(), "缺少参数") {
			t.Errorf("instance_ids 存在时不应报缺少参数，实际: %s", rr.Body.String())
		}
	}
}

// TestHandleAdminDetectInstall_ByInstanceIDs_TooMany instance_ids > 50 应报错。
func TestHandleAdminDetectInstall_ByInstanceIDs_TooMany(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	ids := make([]string, 51)
	for i := range ids {
		ids[i] = "ins-x"
	}
	body, _ := json.Marshal(map[string]interface{}{"instance_ids": ids})
	req := adminInstanceIDReqJSON(http.MethodPost, "/admin/instances/detect-install", body)
	rr := httptest.NewRecorder()
	HandleAdminDetectInstall(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("instance_ids>50 应返回 400，实际=%d", rr.Code)
	}
}

// TestHandleAdminDetectInstall_MissingAllParams 所有参数都不传应报错。
func TestHandleAdminDetectInstall_MissingAllParams(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{})
	req := adminInstanceIDReqJSON(http.MethodPost, "/admin/instances/detect-install", body)
	rr := httptest.NewRecorder()
	HandleAdminDetectInstall(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("所有参数都不传应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// ─── getInstanceByIDRaw: instance_id 路径（用户侧）──────────────────────────

// TestGetInstanceByIDRaw_ByInstanceID 通过 instance_id 查询用户实例。
func TestGetInstanceByIDRaw_ByInstanceID(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	user := &model.User{Username: "testuser2", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)

	inst := &model.Instance{Name: "test", InstanceId: "ins-user-query", UserID: user.ID}
	model.DB(context.Background()).Create(inst)

	req := userInstanceIDReqWithSession(t, http.MethodGet, "/openclaw/status?instance_id=ins-user-query", "testuser2")
	rr := httptest.NewRecorder()
	w := http.ResponseWriter(rr)

	result, err := getInstanceByIDRaw(&w, req, user.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if result.InstanceId != "ins-user-query" {
		t.Errorf("期望 ins-user-query，实际 %s", result.InstanceId)
	}
}

// TestGetInstanceByIDRaw_ByInstanceID_WrongUser instance_id 属于其他用户应报错。
func TestGetInstanceByIDRaw_ByInstanceID_WrongUser(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	user1 := &model.User{Username: "user1-raw", Password: "test", Role: "user"}
	user2 := &model.User{Username: "user2-raw", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user1)
	model.DB(context.Background()).Create(user2)

	inst := &model.Instance{Name: "test", InstanceId: "ins-user1-only", UserID: user1.ID}
	model.DB(context.Background()).Create(inst)

	// user2 尝试访问 user1 的实例
	req := userInstanceIDReqWithSession(t, http.MethodGet, "/openclaw/status?instance_id=ins-user1-only", "user2-raw")
	rr := httptest.NewRecorder()
	w := http.ResponseWriter(rr)

	_, err := getInstanceByIDRaw(&w, req, user2.ID)
	if err == nil {
		t.Fatal("跨用户访问应报错")
	}
	if !strings.Contains(err.Error(), "不存在") {
		t.Fatalf("错误消息应含 '不存在'，实际: %v", err)
	}
}

// TestGetInstanceByIDRaw_MissingBothParams 两者都不传应报错。
func TestGetInstanceByIDRaw_MissingBothParams(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/openclaw/status", nil)
	rr := httptest.NewRecorder()
	w := http.ResponseWriter(rr)

	_, err := getInstanceByIDRaw(&w, req, 1)
	if err == nil {
		t.Fatal("两者都不传应报错")
	}
	if !strings.Contains(hcommon.ErrorMessageWithCtx(req.Context(), err), "缺少参数") {
		t.Fatalf("错误消息应含 '缺少参数'，实际: %v", err)
	}
}

// TestGetInstanceByIDRaw_AdminMode_ByInstanceID userID=0 时不限所有者，通过 instance_id 查询。
func TestGetInstanceByIDRaw_AdminMode_ByInstanceID(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	user := &model.User{Username: "someuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)

	inst := &model.Instance{Name: "test", InstanceId: "ins-admin-mode", UserID: user.ID}
	model.DB(context.Background()).Create(inst)

	req := httptest.NewRequest(http.MethodGet, "/admin/instances/status?instance_id=ins-admin-mode", nil)
	rr := httptest.NewRecorder()
	w := http.ResponseWriter(rr)

	// userID=0 表示管理员模式，不限所有者
	result, err := getInstanceByIDRaw(&w, req, 0)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if result.InstanceId != "ins-admin-mode" {
		t.Errorf("期望 ins-admin-mode，实际 %s", result.InstanceId)
	}
}

// ─── HandleInstanceDeniedActions: instance_ids 路径 ──────────────────────────

// TestHandleInstanceDeniedActions_MethodNotAllowed GET 应返回 405。
func TestHandleInstanceDeniedActions_MethodNotAllowed(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	user := &model.User{Username: "da-user", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := userInstanceIDReqWithSession(t, http.MethodGet, "/openclaw/denied-actions", "da-user")
	rr := httptest.NewRecorder()
	HandleInstanceDeniedActions(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 应返回 405，实际=%d", rr.Code)
	}
}

// TestHandleInstanceDeniedActions_ByIDs ids 路径正常返回。
func TestHandleInstanceDeniedActions_ByIDs(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	user := &model.User{Username: "da-user2", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)

	inst := &model.Instance{Name: "test", InstanceId: "", UserID: user.ID}
	model.DB(context.Background()).Create(inst)

	body, _ := json.Marshal(map[string]interface{}{"ids": []uint{inst.ID}})
	req := userInstanceIDReqWithSession(t, http.MethodPost, "/openclaw/denied-actions", "da-user2")
	req.Body = io.NopCloser(bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	HandleInstanceDeniedActions(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("ids 路径应返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// TestHandleInstanceDeniedActions_ByInstanceIDs instance_ids 路径正常返回。
func TestHandleInstanceDeniedActions_ByInstanceIDs(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	user := &model.User{Username: "da-user3", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)

	// InstanceId 为空，避免触发 CVM API 调用（无凭据）
	inst := &model.Instance{Name: "test", InstanceId: "", UserID: user.ID}
	model.DB(context.Background()).Create(inst)

	// 用 instance_id 字段查询（数据库中 instance_id 为空，但 instance_ids 参数匹配空字符串）
	// 改为用 ids 路径验证 instance_ids 参数解析逻辑：先创建有 instance_id 的实例，再用 instance_ids 查
	inst2 := &model.Instance{Name: "test2", InstanceId: "ins-da-test2", UserID: user.ID}
	model.DB(context.Background()).Create(inst2)

	// 用 instance_ids 查询，但实例 InstanceId 为空（不会触发 CVM API）
	body, _ := json.Marshal(map[string]interface{}{"instance_ids": []string{"ins-da-test2"}})
	req := userInstanceIDReqWithSession(t, http.MethodPost, "/openclaw/denied-actions", "da-user3")
	req.Body = io.NopCloser(bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	HandleInstanceDeniedActions(rr, req)
	// inst2 的 InstanceId 非空，会尝试调用 CVM API，凭据未配置会返回 500
	// 只验证参数解析通过（不是 400 缺少参数）
	if rr.Code == http.StatusBadRequest && strings.Contains(rr.Body.String(), "缺少参数") {
		t.Errorf("instance_ids 存在时不应报缺少参数，实际: %s", rr.Body.String())
	}
}

// TestHandleInstanceDeniedActions_LocalInstance 本地 agent 实例虽然有非空
// instance_id（host CID 如 "local-codebuddy-001"），但不是 CVM 格式。应被过滤掉，
// 不会被 CVM API 报「实例ID不合要求」。
func TestHandleInstanceDeniedActions_LocalInstance(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	user := &model.User{Username: "da-local", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)

	inst := &model.Instance{
		Name:       "local-codebuddy",
		InstanceId: "local-codebuddy-001",
		Source:     model.InstanceSourceLocal,
		UserID:     user.ID,
	}
	model.DB(context.Background()).Create(inst)

	body, _ := json.Marshal(map[string]interface{}{"ids": []uint{inst.ID}})
	req := userInstanceIDReqWithSession(t, http.MethodPost, "/openclaw/denied-actions", "da-local")
	req.Body = io.NopCloser(bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	HandleInstanceDeniedActions(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("本地实例调 denied-actions 应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "实例ID") || strings.Contains(rr.Body.String(), "不合要求") {
		t.Errorf("本地实例不应报 CVM ID 格式错，实际=%s", rr.Body.String())
	}
}

// TestHandleInstanceDeniedActions_MissingParams 两者都不传应返回空结果。
func TestHandleInstanceDeniedActions_MissingParams(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	user := &model.User{Username: "da-user4", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)

	body, _ := json.Marshal(map[string]interface{}{})
	req := userInstanceIDReqWithSession(t, http.MethodPost, "/openclaw/denied-actions", "da-user4")
	req.Body = io.NopCloser(bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	HandleInstanceDeniedActions(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("两者都不传应返回 200，实际=%d", rr.Code)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	instances, ok := result["instances"].([]interface{})
	if !ok {
		t.Errorf("响应中应包含 instances 数组")
	}
	if len(instances) != 0 {
		t.Errorf("缺少参数时应返回空实例列表，实际长度=%d", len(instances))
	}
}

// TestHandleInstanceDeniedActions_InvalidJSON 无效 JSON 应返回 400。
func TestHandleInstanceDeniedActions_InvalidJSON(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	user := &model.User{Username: "da-user5", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := userInstanceIDReqWithSession(t, http.MethodPost, "/openclaw/denied-actions", "da-user5")
	req.Body = io.NopCloser(strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	HandleInstanceDeniedActions(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("无效 JSON 应返回 400，实际=%d", rr.Code)
	}
}

// ─── handleAdminStartInstance: instance_id 正常路径（实例存在但 CVM 状态不对）──

// TestHandleAdminStartInstance_ByInstanceID_Found instance_id 存在时进入后续逻辑。
func TestHandleAdminStartInstance_ByInstanceID_Found(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	inst := &model.Instance{Name: "test", InstanceId: "ins-start-found", AgentType: "openclaw"}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("instance_id", "ins-start-found")
	req := httptest.NewRequest(http.MethodPost, "/admin/instances/start", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer test-admin-token")

	rr := httptest.NewRecorder()
	handleAdminStartInstance(rr, req, testCVMFetcher)
	// 实例存在，参数解析通过，后续 CVM 状态检查（testCVMFetcher 返回 RUNNING）→ 409
	if rr.Code == http.StatusBadRequest && strings.Contains(rr.Body.String(), "缺少参数") {
		t.Errorf("instance_id 存在时不应报缺少参数，实际: %s", rr.Body.String())
	}
}

// TestHandleAdminStopInstance_ByInstanceID_Found instance_id 存在时进入后续逻辑。
func TestHandleAdminStopInstance_ByInstanceID_Found(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	inst := &model.Instance{Name: "test", InstanceId: "ins-stop-found", AgentType: "openclaw"}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("instance_id", "ins-stop-found")
	req := httptest.NewRequest(http.MethodPost, "/admin/instances/stop", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer test-admin-token")

	rr := httptest.NewRecorder()
	handleAdminStopInstance(rr, req, testCVMFetcher)
	if rr.Code == http.StatusBadRequest && strings.Contains(rr.Body.String(), "缺少参数") {
		t.Errorf("instance_id 存在时不应报缺少参数，实际: %s", rr.Body.String())
	}
}

// TestHandleAdminRebootInstance_ByInstanceID_Found instance_id 存在时进入后续逻辑。
func TestHandleAdminRebootInstance_ByInstanceID_Found(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	inst := &model.Instance{Name: "test", InstanceId: "ins-reboot-found", AgentType: "openclaw"}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("instance_id", "ins-reboot-found")
	req := httptest.NewRequest(http.MethodPost, "/admin/instances/reboot", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer test-admin-token")

	rr := httptest.NewRecorder()
	handleAdminRebootInstance(rr, req, testCVMFetcher)
	if rr.Code == http.StatusBadRequest && strings.Contains(rr.Body.String(), "缺少参数") {
		t.Errorf("instance_id 存在时不应报缺少参数，实际: %s", rr.Body.String())
	}
}

// TestHandleAdminResetInstance_ByInstanceID_Found instance_id 存在时进入后续逻辑。
func TestHandleAdminResetInstance_ByInstanceID_Found(t *testing.T) {
	cleanup := initInstanceIDParamTestDB(t)
	defer cleanup()

	inst := &model.Instance{Name: "test", InstanceId: "ins-reset-found", AgentType: "openclaw"}
	model.DB(context.Background()).Create(inst)

	form := url.Values{}
	form.Set("instance_id", "ins-reset-found")
	req := httptest.NewRequest(http.MethodPost, "/admin/instances/reset", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer test-admin-token")

	rr := httptest.NewRecorder()
	handleAdminResetInstance(rr, req, testCVMFetcher)
	if rr.Code == http.StatusBadRequest && strings.Contains(rr.Body.String(), "缺少参数") {
		t.Errorf("instance_id 存在时不应报缺少参数，实际: %s", rr.Body.String())
	}
}
