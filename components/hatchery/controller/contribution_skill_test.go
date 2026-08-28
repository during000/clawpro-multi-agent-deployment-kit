package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// setupContributionTestDB 初始化测试 DB，创建管理员和员工用户，返回 cleanup。
func setupContributionTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)

	if err := db.AutoMigrate(
		&model.User{},
		&model.Skill{},
		&model.SkillCategoryMapping{},
		&model.SkillCategory{},
		&model.SkillDistributionTask{},
		&model.SkillDistributionRecord{},
		&model.SkillVisibilityGroup{},
		&model.SiteConfig{},
		&model.SMHSpace{},
		&model.ReviewRequest{},
		&model.Notification{},
		&model.Instance{},
		&model.ProjectConfigBinding{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	origDB := model.UseDBForTestWithDriver(db, "sqlite")
	db.Create(&model.SiteConfig{SMHEnabled: 1})

	origToken := AdminToken
	AdminToken = "test-admin-token"

	if Store == nil {
		Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	}

	// 创建管理员用户
	db.Create(&model.User{Username: "admin", Role: "admin", Password: "x"})

	t.Cleanup(func() {
		AdminToken = origToken
		origDB()
	})
}

// createEmployeeSessionReq 创建带员工 session 的请求
func createEmployeeSessionReq(t *testing.T, method, url, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = "employee"
	rr := httptest.NewRecorder()
	session.Save(req, rr)
	for _, cookie := range rr.Result().Cookies() {
		req.AddCookie(cookie)
	}
	return req
}

// createAdminReq 创建带 admin token 的请求
func createAdminReq(method, url, body string) *http.Request {
	req := httptest.NewRequest(method, url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req
}

func assertReviewNotification(t *testing.T, userID uint, notifyType string, messageParts ...string) {
	t.Helper()
	var notif model.Notification
	if err := model.DB(context.Background()).
		Where("user_id = ? AND type = ?", userID, notifyType).
		First(&notif).Error; err != nil {
		t.Fatalf("未找到审核结果通知 type=%s: %v", notifyType, err)
	}
	for _, part := range messageParts {
		if !strings.Contains(notif.Message, part) {
			t.Errorf("通知内容 %q 不包含 %q", notif.Message, part)
		}
	}
}

// seedEmployeeUser 创建员工用户并返回
func seedEmployeeUser(t *testing.T) *model.User {
	u := &model.User{Username: "employee", Role: "user", Password: "x"}
	if err := model.DB(context.Background()).Create(u).Error; err != nil {
		t.Fatalf("创建员工用户失败: %v", err)
	}
	return u
}

// seedPublishedSkill 创建已上架技能
func seedPublishedSkill(t *testing.T, slug, version string, uploaderID uint) *model.Skill {
	s := &model.Skill{
		Slug:         slug,
		Name:         "Test Skill",
		Version:      version,
		VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
		Status:     model.SkillStatusPublished,
		UploaderID: uploaderID,
	}
	if err := model.DB(context.Background()).Create(s).Error; err != nil {
		t.Fatalf("创建技能失败: %v", err)
	}
	return s
}

// seedPendingReviewSkill 创建待审核技能
func seedPendingReviewSkill(t *testing.T, slug, version string, uploaderID uint) *model.Skill {
	s := &model.Skill{
		Slug:         slug,
		Name:         "Pending Skill",
		Version:      version,
		VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
		Status:     model.SkillStatusPendingReview,
		UploaderID: uploaderID,
	}
	if err := model.DB(context.Background()).Create(s).Error; err != nil {
		t.Fatalf("创建技能失败: %v", err)
	}
	return s
}

// ── 下架流程测试 ─────────────────────────────────────────────────────

func TestHandleTakedownSkill_Success(t *testing.T) {
	setupContributionTestDB(t)
	emp := seedEmployeeUser(t)
	seedPublishedSkill(t, "my-skill", "1.0.0", emp.ID)

	body := `{"slug":"my-skill","reason":"不再需要"}`
	req := createEmployeeSessionReq(t, "POST", "/openclaw/skills/takedown", body)
	w := httptest.NewRecorder()
	HandleTakedownSkill(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["ok"] != true {
		t.Errorf("期望 ok=true，实际=%v", resp["ok"])
	}

	// 验证 ReviewRequest 已创建
	var req2 model.ReviewRequest
	if model.DB(context.Background()).Where("slug = ? AND action_type = ?", "my-skill", model.ActionTypeTakedown).First(&req2).Error != nil {
		t.Fatal("ReviewRequest 未创建")
	}
	if req2.Status != model.ReviewStatusPending {
		t.Errorf("期望 status=pending，实际=%s", req2.Status)
	}
	if req2.RequesterID != emp.ID {
		t.Errorf("期望 requester_id=%d，实际=%d", emp.ID, req2.RequesterID)
	}
}

func TestHandleTakedownSkill_NotOwner(t *testing.T) {
	setupContributionTestDB(t)
	seedEmployeeUser(t)
	other := &model.User{Username: "other", Role: "user", Password: "x"}
	model.DB(context.Background()).Create(other)
	// other 上传的技能
	seedPublishedSkill(t, "other-skill", "1.0.0", other.ID)

	body := `{"slug":"other-skill","reason":"test"}`
	req := createEmployeeSessionReq(t, "POST", "/openclaw/skills/takedown", body)
	w := httptest.NewRecorder()
	HandleTakedownSkill(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("期望 403（非上传者），实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleTakedownSkill_AdminUploaded(t *testing.T) {
	setupContributionTestDB(t)
	seedEmployeeUser(t)
	// 管理员上传的技能（UploaderID=0）
	seedPublishedSkill(t, "admin-skill", "1.0.0", 0)

	body := `{"slug":"admin-skill","reason":"test"}`
	req := createEmployeeSessionReq(t, "POST", "/openclaw/skills/takedown", body)
	w := httptest.NewRecorder()
	HandleTakedownSkill(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("期望 403（管理员上传），实际=%d", w.Code)
	}
}

func TestHandleTakedownSkill_NotExist(t *testing.T) {
	setupContributionTestDB(t)
	seedEmployeeUser(t)

	body := `{"slug":"nonexistent","reason":"test"}`
	req := createEmployeeSessionReq(t, "POST", "/openclaw/skills/takedown", body)
	w := httptest.NewRecorder()
	HandleTakedownSkill(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际=%d", w.Code)
	}
}

func TestHandleTakedownSkill_MissingReason(t *testing.T) {
	setupContributionTestDB(t)
	emp := seedEmployeeUser(t)
	seedPublishedSkill(t, "my-skill", "1.0.0", emp.ID)

	body := `{"slug":"my-skill","reason":""}`
	req := createEmployeeSessionReq(t, "POST", "/openclaw/skills/takedown", body)
	w := httptest.NewRecorder()
	HandleTakedownSkill(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400（缺少 reason），实际=%d", w.Code)
	}
}

func TestHandleTakedownSkill_MutexConflict(t *testing.T) {
	setupContributionTestDB(t)
	emp := seedEmployeeUser(t)
	skill := seedPublishedSkill(t, "my-skill", "1.0.0", emp.ID)
	model.DB(context.Background()).Create(&model.ReviewRequest{
		RequesterID:  emp.ID,
		ResourceType: model.ResourceTypeSkill,
		ResourceID:   skill.ID,
		ActionType:   model.ActionTypeTakedown,
		Slug:         "my-skill",
		Status:       model.ReviewStatusPending,
	})

	body := `{"slug":"my-skill","reason":"test"}`
	req := createEmployeeSessionReq(t, "POST", "/openclaw/skills/takedown", body)
	w := httptest.NewRecorder()
	HandleTakedownSkill(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400（互斥），实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleTakedownSkill_PendingReviewSkill(t *testing.T) {
	setupContributionTestDB(t)
	emp := seedEmployeeUser(t)
	// pending_review 状态的技能不允许下架
	seedPendingReviewSkill(t, "pending-skill", "1.0.0", emp.ID)

	body := `{"slug":"pending-skill","reason":"test"}`
	req := createEmployeeSessionReq(t, "POST", "/openclaw/skills/takedown", body)
	w := httptest.NewRecorder()
	HandleTakedownSkill(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404（pending_review 不可下架），实际=%d", w.Code)
	}
}

// ── 审核流程测试 ─────────────────────────────────────────────────────

func TestHandleApprove_Publish(t *testing.T) {
	setupContributionTestDB(t)
	emp := seedEmployeeUser(t)
	skill := seedPendingReviewSkill(t, "my-skill", "1.0.0", emp.ID)
	req := &model.ReviewRequest{
		RequesterID:  emp.ID,
		ResourceType: model.ResourceTypeSkill,
		ResourceID:   skill.ID,
		ActionType:   model.ActionTypePublish,
		Slug:         "my-skill",
		Status:       model.ReviewStatusPending,
	}
	model.DB(context.Background()).Create(req)

	body := `{"id":1}`
	r := createAdminReq("POST", "/admin/contributions/approve", body)
	w := httptest.NewRecorder()
	HandleApproveContribution(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	// 验证 Skill 状态变为 published
	var updated model.Skill
	model.DB(context.Background()).Where("id = ?", skill.ID).First(&updated)
	if updated.Status != model.SkillStatusPublished {
		t.Errorf("期望 skill status=published，实际=%s", updated.Status)
	}
	// 验证 ReviewRequest 状态变为 approved
	var updatedReq model.ReviewRequest
	model.DB(context.Background()).Where("id = ?", req.ID).First(&updatedReq)
	if updatedReq.Status != model.ReviewStatusApproved {
		t.Errorf("期望 request status=approved，实际=%s", updatedReq.Status)
	}
	assertReviewNotification(t, emp.ID, "skill_review_approved", "my-skill")
}

func TestHandleApprove_Takedown(t *testing.T) {
	setupContributionTestDB(t)
	emp := seedEmployeeUser(t)
	skill := seedPublishedSkill(t, "my-skill", "1.0.0", emp.ID)
	req := &model.ReviewRequest{
		RequesterID:  emp.ID,
		ResourceType: model.ResourceTypeSkill,
		ResourceID:   skill.ID,
		ActionType:   model.ActionTypeTakedown,
		Slug:         "my-skill",
		Status:       model.ReviewStatusPending,
	}
	model.DB(context.Background()).Create(req)

	body := `{"id":1}`
	r := createAdminReq("POST", "/admin/contributions/approve", body)
	w := httptest.NewRecorder()
	HandleApproveContribution(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	// 验证 Skill 状态变为 offline
	var updated model.Skill
	model.DB(context.Background()).Where("id = ?", skill.ID).First(&updated)
	if updated.Status != model.SkillStatusOffline {
		t.Errorf("期望 skill status=offline，实际=%s", updated.Status)
	}
	assertReviewNotification(t, emp.ID, "skill_takedown_approved", "my-skill")
}

func TestHandleReject_Publish(t *testing.T) {
	setupContributionTestDB(t)
	emp := seedEmployeeUser(t)
	skill := seedPendingReviewSkill(t, "my-skill", "1.0.0", emp.ID)
	req := &model.ReviewRequest{
		RequesterID:  emp.ID,
		ResourceType: model.ResourceTypeSkill,
		ResourceID:   skill.ID,
		ActionType:   model.ActionTypePublish,
		Slug:         "my-skill",
		Status:       model.ReviewStatusPending,
	}
	model.DB(context.Background()).Create(req)

	body := `{"id":1,"review_comment":"内容不符合要求"}`
	r := createAdminReq("POST", "/admin/contributions/reject", body)
	w := httptest.NewRecorder()
	HandleRejectContribution(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	// 验证 Skill 已被软删除
	var count int64
	model.DB(context.Background()).Model(&model.Skill{}).Where("id = ?", skill.ID).Count(&count)
	if count != 0 {
		t.Errorf("期望 Skill 已软删除，实际 count=%d", count)
	}
	// 验证 ReviewRequest 状态变为 rejected
	var updatedReq model.ReviewRequest
	model.DB(context.Background()).Where("id = ?", req.ID).First(&updatedReq)
	if updatedReq.Status != model.ReviewStatusRejected {
		t.Errorf("期望 request status=rejected，实际=%s", updatedReq.Status)
	}
	if updatedReq.ReviewComment == "" {
		t.Error("期望 review_comment 非空")
	}
	assertReviewNotification(t, emp.ID, "skill_review_rejected", "my-skill", "内容不符合要求")
}

func TestHandleReject_Takedown(t *testing.T) {
	setupContributionTestDB(t)
	emp := seedEmployeeUser(t)
	skill := seedPublishedSkill(t, "my-skill", "1.0.0", emp.ID)
	req := &model.ReviewRequest{
		RequesterID:  emp.ID,
		ResourceType: model.ResourceTypeSkill,
		ResourceID:   skill.ID,
		ActionType:   model.ActionTypeTakedown,
		Slug:         "my-skill",
		Status:       model.ReviewStatusPending,
	}
	model.DB(context.Background()).Create(req)

	body := `{"id":1,"review_comment":"技能仍需保留"}`
	r := createAdminReq("POST", "/admin/contributions/reject", body)
	w := httptest.NewRecorder()
	HandleRejectContribution(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	// 验证 Skill 状态不变（仍为 published）
	var updated model.Skill
	model.DB(context.Background()).Where("id = ?", skill.ID).First(&updated)
	if updated.Status != model.SkillStatusPublished {
		t.Errorf("期望 skill status=published（不变），实际=%s", updated.Status)
	}
	assertReviewNotification(t, emp.ID, "skill_takedown_rejected", "my-skill", "技能仍需保留")
}

func TestHandleApprove_AlreadyReviewed(t *testing.T) {
	setupContributionTestDB(t)
	emp := seedEmployeeUser(t)
	skill := seedPendingReviewSkill(t, "my-skill", "1.0.0", emp.ID)
	req := &model.ReviewRequest{
		RequesterID:  emp.ID,
		ResourceType: model.ResourceTypeSkill,
		ResourceID:   skill.ID,
		ActionType:   model.ActionTypePublish,
		Slug:         "my-skill",
		Status:       model.ReviewStatusApproved, // 已审核
	}
	model.DB(context.Background()).Create(req)

	body := `{"id":1}`
	r := createAdminReq("POST", "/admin/contributions/approve", body)
	w := httptest.NewRecorder()
	HandleApproveContribution(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400（已审核），实际=%d", w.Code)
	}
}

func TestHandleReject_MissingComment(t *testing.T) {
	setupContributionTestDB(t)
	emp := seedEmployeeUser(t)
	skill := seedPendingReviewSkill(t, "my-skill", "1.0.0", emp.ID)
	req := &model.ReviewRequest{
		RequesterID:  emp.ID,
		ResourceType: model.ResourceTypeSkill,
		ResourceID:   skill.ID,
		ActionType:   model.ActionTypePublish,
		Slug:         "my-skill",
		Status:       model.ReviewStatusPending,
	}
	model.DB(context.Background()).Create(req)

	body := `{"id":1,"review_comment":""}`
	r := createAdminReq("POST", "/admin/contributions/reject", body)
	w := httptest.NewRecorder()
	HandleRejectContribution(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400（缺少 comment），实际=%d", w.Code)
	}
}

func TestHandleApprove_NotFound(t *testing.T) {
	setupContributionTestDB(t)

	body := `{"id":999}`
	r := createAdminReq("POST", "/admin/contributions/approve", body)
	w := httptest.NewRecorder()
	HandleApproveContribution(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际=%d", w.Code)
	}
}

// ── 查询接口测试 ─────────────────────────────────────────────────────

func TestHandleMyContributions(t *testing.T) {
	setupContributionTestDB(t)
	emp := seedEmployeeUser(t)
	skill := seedPublishedSkill(t, "my-skill", "1.0.0", emp.ID)

	model.DB(context.Background()).Create(&model.ReviewRequest{
		RequesterID:  emp.ID,
		ResourceType: model.ResourceTypeSkill,
		ResourceID:   skill.ID,
		ActionType:   model.ActionTypeTakedown,
		Slug:         "my-skill",
		Status:       model.ReviewStatusPending,
	})

	req := createEmployeeSessionReq(t, "GET", "/openclaw/skills/contributions", "")
	w := httptest.NewRecorder()
	HandleMyContributions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	skills := resp["skills"].([]interface{})
	if len(skills) != 1 {
		t.Fatalf("期望 1 个技能组，实际=%d", len(skills))
	}
	group := skills[0].(map[string]interface{})
	requests := group["requests"].([]interface{})
	if len(requests) != 1 {
		t.Errorf("期望 1 条申请，实际=%d", len(requests))
	}
}

func TestHandleMyContributionDetail_NotOwner(t *testing.T) {
	setupContributionTestDB(t)
	emp := seedEmployeeUser(t)
	other := &model.User{Username: "other", Role: "user", Password: "x"}
	model.DB(context.Background()).Create(other)
	skill := seedPublishedSkill(t, "my-skill", "1.0.0", emp.ID)

	req2 := &model.ReviewRequest{
		RequesterID:  emp.ID, // emp 的申请
		ResourceType: model.ResourceTypeSkill,
		ResourceID:   skill.ID,
		ActionType:   model.ActionTypePublish,
		Slug:         "my-skill",
		Status:       model.ReviewStatusPending,
	}
	model.DB(context.Background()).Create(req2)

	// other 登录，尝试查看 emp 的申请
	req := httptest.NewRequest("GET", "/openclaw/skills/contributions/detail?id=1", nil)
	req.Header.Set("Accept", "application/json")
	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = "other"
	rr := httptest.NewRecorder()
	session.Save(req, rr)
	for _, cookie := range rr.Result().Cookies() {
		req.AddCookie(cookie)
	}

	w := httptest.NewRecorder()
	HandleMyContributionDetail(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("期望 403（非申请人），实际=%d", w.Code)
	}
}

func TestHandleAdminContributions(t *testing.T) {
	setupContributionTestDB(t)
	emp := seedEmployeeUser(t)
	skill := seedPendingReviewSkill(t, "my-skill", "1.0.0", emp.ID)
	model.DB(context.Background()).Create(&model.ReviewRequest{
		RequesterID:  emp.ID,
		ResourceType: model.ResourceTypeSkill,
		ResourceID:   skill.ID,
		ActionType:   model.ActionTypePublish,
		Slug:         "my-skill",
		Status:       model.ReviewStatusPending,
	})

	req := createAdminReq("GET", "/admin/contributions?status=pending", "")
	w := httptest.NewRecorder()
	HandleAdminContributions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	requests := resp["requests"].([]interface{})
	if len(requests) != 1 {
		t.Errorf("期望 1 条 pending 申请，实际=%d", len(requests))
	}
}

// ── 状态机一致性测试 ─────────────────────────────────────────────────

func TestStateMachine_RejectThenResubmit(t *testing.T) {
	setupContributionTestDB(t)
	emp := seedEmployeeUser(t)
	skill := seedPendingReviewSkill(t, "my-skill", "1.0.0", emp.ID)
	oldReq := &model.ReviewRequest{
		RequesterID:  emp.ID,
		ResourceType: model.ResourceTypeSkill,
		ResourceID:   skill.ID,
		ActionType:   model.ActionTypePublish,
		Slug:         "my-skill",
		Status:       model.ReviewStatusRejected, // 已拒绝
	}
	model.DB(context.Background()).Create(oldReq)

	// 拒绝后旧 Skill 被软删除，可以重新提交
	// 验证 HasPendingRequest 返回 false（rejected 不阻塞）
	if model.HasPendingRequest(context.Background(), model.ResourceTypeSkill, "my-skill") {
		t.Error("rejected 申请不应阻塞新申请")
	}
}

func TestStateMachine_ApprovePublishThenTakedown(t *testing.T) {
	setupContributionTestDB(t)
	emp := seedEmployeeUser(t)
	skill := seedPendingReviewSkill(t, "my-skill", "1.0.0", emp.ID)
	publishReq := &model.ReviewRequest{
		RequesterID:  emp.ID,
		ResourceType: model.ResourceTypeSkill,
		ResourceID:   skill.ID,
		ActionType:   model.ActionTypePublish,
		Slug:         "my-skill",
		Status:       model.ReviewStatusPending,
	}
	model.DB(context.Background()).Create(publishReq)

	// 审核通过 publish
	body := `{"id":1}`
	r := createAdminReq("POST", "/admin/contributions/approve", body)
	w := httptest.NewRecorder()
	HandleApproveContribution(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("审核通过失败: %d, %s", w.Code, w.Body.String())
	}

	// Skill 现在是 published，可以申请下架
	if model.HasPendingRequest(context.Background(), model.ResourceTypeSkill, "my-skill") {
		t.Fatal("publish 通过后不应有 pending 申请")
	}

	// 员工申请下架
	takedownBody := `{"slug":"my-skill","reason":"不再需要"}`
	tr := createEmployeeSessionReq(t, "POST", "/openclaw/skills/takedown", takedownBody)
	tw := httptest.NewRecorder()
	HandleTakedownSkill(tw, tr)
	if tw.Code != http.StatusOK {
		t.Fatalf("下架申请失败: %d, %s", tw.Code, tw.Body.String())
	}
}

// ── Model 层测试 ────────────────────────────────────────────────────

func TestHasPendingRequest(t *testing.T) {
	setupContributionTestDB(t)
	emp := seedEmployeeUser(t)
	skill := seedPublishedSkill(t, "my-skill", "1.0.0", emp.ID)

	// 无 pending 申请
	if model.HasPendingRequest(context.Background(), model.ResourceTypeSkill, "my-skill") {
		t.Error("不应有 pending 申请")
	}

	// 创建 pending 申请
	model.DB(context.Background()).Create(&model.ReviewRequest{
		RequesterID:  emp.ID,
		ResourceType: model.ResourceTypeSkill,
		ResourceID:   skill.ID,
		ActionType:   model.ActionTypePublish,
		Slug:         "my-skill",
		Status:       model.ReviewStatusPending,
	})

	// 有 pending 申请
	if !model.HasPendingRequest(context.Background(), model.ResourceTypeSkill, "my-skill") {
		t.Error("应有 pending 申请")
	}

	// 其他 slug 无 pending
	if model.HasPendingRequest(context.Background(), model.ResourceTypeSkill, "other-slug") {
		t.Error("other-slug 不应有 pending 申请")
	}
}

// ── 技能广场 status 过滤测试 ─────────────────────────────────────────

func TestSkillStore_ExcludesPendingReview(t *testing.T) {
	setupContributionTestDB(t)
	emp := seedEmployeeUser(t)
	seedPendingReviewSkill(t, "pending-skill", "1.0.0", emp.ID)
	seedPublishedSkill(t, "published-skill", "1.0.0", emp.ID)

	// 查询已上架技能，应只返回 published-skill
	var skills []model.Skill
	model.DB(context.Background()).
		Model(&model.Skill{}).
		Where("id IN (?)", model.LatestVersionSkillIDs(context.Background())).
		Where("status = ?", model.SkillStatusPublished).
		Find(&skills)

	if len(skills) != 1 {
		t.Fatalf("期望 1 个 published 技能，实际=%d", len(skills))
	}
	if skills[0].Slug != "published-skill" {
		t.Errorf("期望 slug=published-skill，实际=%s", skills[0].Slug)
	}
}

// ── 员工撤回测试 ──────────────────────────────────────────────────────

func TestHandleWithdrawContribution_Success(t *testing.T) {
	setupContributionTestDB(t)
	emp := seedEmployeeUser(t)
	skill := seedPendingReviewSkill(t, "my-skill", "1.0.0", emp.ID)
	model.DB(context.Background()).Create(&model.ReviewRequest{
		RequesterID:  emp.ID,
		ResourceType: model.ResourceTypeSkill,
		ResourceID:   skill.ID,
		ActionType:   model.ActionTypePublish,
		Slug:         "my-skill",
		Status:       model.ReviewStatusPending,
	})

	body := `{"id":1}`
	req := createEmployeeSessionReq(t, "POST", "/openclaw/skills/contributions/withdraw", body)
	w := httptest.NewRecorder()
	HandleWithdrawContribution(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	// 验证 ReviewRequest 状态变为 withdrawn
	var updatedReq model.ReviewRequest
	model.DB(context.Background()).Where("id = ?", 1).First(&updatedReq)
	if updatedReq.Status != model.ReviewStatusWithdrawn {
		t.Errorf("期望 status=withdrawn，实际=%s", updatedReq.Status)
	}
	// 验证 Skill 已被软删除（publish 撤回）
	var count int64
	model.DB(context.Background()).Model(&model.Skill{}).Where("id = ?", skill.ID).Count(&count)
	if count != 0 {
		t.Errorf("期望 Skill 已软删除，实际 count=%d", count)
	}
}

func TestHandleWithdrawContribution_Takedown(t *testing.T) {
	setupContributionTestDB(t)
	emp := seedEmployeeUser(t)
	skill := seedPublishedSkill(t, "my-skill", "1.0.0", emp.ID)
	model.DB(context.Background()).Create(&model.ReviewRequest{
		RequesterID:  emp.ID,
		ResourceType: model.ResourceTypeSkill,
		ResourceID:   skill.ID,
		ActionType:   model.ActionTypeTakedown,
		Slug:         "my-skill",
		Status:       model.ReviewStatusPending,
	})

	body := `{"id":1}`
	req := createEmployeeSessionReq(t, "POST", "/openclaw/skills/contributions/withdraw", body)
	w := httptest.NewRecorder()
	HandleWithdrawContribution(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	// takedown 撤回：Skill 状态不变
	var updated model.Skill
	model.DB(context.Background()).Where("id = ?", skill.ID).First(&updated)
	if updated.Status != model.SkillStatusPublished {
		t.Errorf("期望 skill status=published（不变），实际=%s", updated.Status)
	}
}

func TestHandleWithdrawContribution_NotOwner(t *testing.T) {
	setupContributionTestDB(t)
	seedEmployeeUser(t)
	other := &model.User{Username: "other", Role: "user", Password: "x"}
	model.DB(context.Background()).Create(other)
	skill := seedPublishedSkill(t, "my-skill", "1.0.0", other.ID)
	model.DB(context.Background()).Create(&model.ReviewRequest{
		RequesterID:  other.ID,
		ResourceType: model.ResourceTypeSkill,
		ResourceID:   skill.ID,
		ActionType:   model.ActionTypeTakedown,
		Slug:         "my-skill",
		Status:       model.ReviewStatusPending,
	})

	body := `{"id":1}`
	req := createEmployeeSessionReq(t, "POST", "/openclaw/skills/contributions/withdraw", body)
	w := httptest.NewRecorder()
	HandleWithdrawContribution(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("期望 403（非申请人），实际=%d", w.Code)
	}
}

func TestHandleWithdrawContribution_NotPending(t *testing.T) {
	setupContributionTestDB(t)
	emp := seedEmployeeUser(t)
	skill := seedPublishedSkill(t, "my-skill", "1.0.0", emp.ID)
	model.DB(context.Background()).Create(&model.ReviewRequest{
		RequesterID:  emp.ID,
		ResourceType: model.ResourceTypeSkill,
		ResourceID:   skill.ID,
		ActionType:   model.ActionTypeTakedown,
		Slug:         "my-skill",
		Status:       model.ReviewStatusApproved,
	})

	body := `{"id":1}`
	req := createEmployeeSessionReq(t, "POST", "/openclaw/skills/contributions/withdraw", body)
	w := httptest.NewRecorder()
	HandleWithdrawContribution(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400（非 pending 不可撤回），实际=%d", w.Code)
	}
}

func TestHandleWithdrawContribution_NotFound(t *testing.T) {
	setupContributionTestDB(t)
	seedEmployeeUser(t)

	body := `{"id":999}`
	req := createEmployeeSessionReq(t, "POST", "/openclaw/skills/contributions/withdraw", body)
	w := httptest.NewRecorder()
	HandleWithdrawContribution(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际=%d", w.Code)
	}
}

// ── 贡献列表增强测试 ──────────────────────────────────────────────────

func TestHandleMyContributions_WithFilters(t *testing.T) {
	setupContributionTestDB(t)
	emp := seedEmployeeUser(t)
	s1 := seedPublishedSkill(t, "skill-a", "1.0.0", emp.ID)
	s2 := seedPublishedSkill(t, "skill-b", "1.0.0", emp.ID)

	model.DB(context.Background()).Create(&model.ReviewRequest{
		RequesterID:  emp.ID,
		ResourceType: model.ResourceTypeSkill,
		ResourceID:   s1.ID,
		ActionType:   model.ActionTypeTakedown,
		Slug:         "skill-a",
		Status:       model.ReviewStatusPending,
	})
	model.DB(context.Background()).Create(&model.ReviewRequest{
		RequesterID:  emp.ID,
		ResourceType: model.ResourceTypeSkill,
		ResourceID:   s2.ID,
		ActionType:   model.ActionTypePublish,
		Slug:         "skill-b",
		Status:       model.ReviewStatusApproved,
	})

	// 按 status=pending 过滤
	req := createEmployeeSessionReq(t, "GET", "/openclaw/skills/contributions?status=pending", "")
	w := httptest.NewRecorder()
	HandleMyContributions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	skills := resp["skills"].([]interface{})
	if len(skills) != 1 {
		t.Fatalf("期望 1 个技能组，实际=%d", len(skills))
	}
	group := skills[0].(map[string]interface{})
	if group["slug"] != "skill-a" {
		t.Errorf("期望 slug=skill-a，实际=%v", group["slug"])
	}
}

func TestHandleMyContributions_KeywordSearch(t *testing.T) {
	setupContributionTestDB(t)
	emp := seedEmployeeUser(t)
	s1 := seedPublishedSkill(t, "search-me", "1.0.0", emp.ID)
	s2 := seedPublishedSkill(t, "other", "1.0.0", emp.ID)
	model.DB(context.Background()).Create(&model.ReviewRequest{
		RequesterID: emp.ID, ResourceType: model.ResourceTypeSkill,
		ResourceID: s1.ID, ActionType: model.ActionTypePublish,
		Slug: "search-me", Status: model.ReviewStatusPending,
	})
	model.DB(context.Background()).Create(&model.ReviewRequest{
		RequesterID: emp.ID, ResourceType: model.ResourceTypeSkill,
		ResourceID: s2.ID, ActionType: model.ActionTypePublish,
		Slug: "other", Status: model.ReviewStatusPending,
	})

	req := createEmployeeSessionReq(t, "GET", "/openclaw/skills/contributions?keyword=search", "")
	w := httptest.NewRecorder()
	HandleMyContributions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	skills := resp["skills"].([]interface{})
	if len(skills) != 1 {
		t.Fatalf("期望 1 个技能组（过滤后），实际=%d", len(skills))
	}
	group := skills[0].(map[string]interface{})
	if group["slug"] != "search-me" {
		t.Errorf("期望 slug=search-me，实际=%v", group["slug"])
	}
}

func TestHandleMyContributions_MultipleRequestsSameSlug(t *testing.T) {
	setupContributionTestDB(t)
	emp := seedEmployeeUser(t)
	s := seedPublishedSkill(t, "multi", "1.0.0", emp.ID)
	model.DB(context.Background()).Create(&model.ReviewRequest{
		RequesterID: emp.ID, ResourceType: model.ResourceTypeSkill,
		ResourceID: s.ID, ActionType: model.ActionTypePublish,
		Slug: "multi", Status: model.ReviewStatusRejected,
	})
	// 再来一个新的 pending
	model.DB(context.Background()).Create(&model.ReviewRequest{
		RequesterID: emp.ID, ResourceType: model.ResourceTypeSkill,
		ResourceID: s.ID, ActionType: model.ActionTypePublish,
		Slug: "multi", Status: model.ReviewStatusPending,
	})

	req := createEmployeeSessionReq(t, "GET", "/openclaw/skills/contributions", "")
	w := httptest.NewRecorder()
	HandleMyContributions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	skills := resp["skills"].([]interface{})
	if len(skills) != 1 {
		t.Fatalf("期望 1 个技能组（同一 slug 聚合），实际=%d", len(skills))
	}
	group := skills[0].(map[string]interface{})
	requests := group["requests"].([]interface{})
	if len(requests) != 2 {
		t.Errorf("期望 2 条申请（同 slug），实际=%d", len(requests))
	}
}

func TestHandleMyContributions_EmptyList(t *testing.T) {
	setupContributionTestDB(t)
	seedEmployeeUser(t)

	req := createEmployeeSessionReq(t, "GET", "/openclaw/skills/contributions", "")
	w := httptest.NewRecorder()
	HandleMyContributions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	skills := resp["skills"].([]interface{})
	if len(skills) != 0 {
		t.Errorf("期望空列表，实际=%d", len(skills))
	}
}

func TestHandleMyContributionDetail_Success(t *testing.T) {
	setupContributionTestDB(t)
	emp := seedEmployeeUser(t)
	skill := seedPublishedSkill(t, "my-skill", "1.0.0", emp.ID)
	model.DB(context.Background()).Create(&model.ReviewRequest{
		RequesterID: emp.ID, ResourceType: model.ResourceTypeSkill,
		ResourceID: skill.ID, ActionType: model.ActionTypeTakedown,
		Slug: "my-skill", Status: model.ReviewStatusPending,
	})

	req := createEmployeeSessionReq(t, "GET", "/openclaw/skills/contributions/detail?id=1", "")
	w := httptest.NewRecorder()
	HandleMyContributionDetail(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}
}

// ── 管理员查询测试 ────────────────────────────────────────────────────

func TestHandleAdminContributions_KeywordSearch(t *testing.T) {
	setupContributionTestDB(t)
	emp := seedEmployeeUser(t)
	s1 := seedPendingReviewSkill(t, "target-skill", "1.0.0", emp.ID)
	s2 := seedPendingReviewSkill(t, "other", "1.0.0", emp.ID)
	model.DB(context.Background()).Create(&model.ReviewRequest{
		RequesterID: emp.ID, ResourceType: model.ResourceTypeSkill,
		ResourceID: s1.ID, ActionType: model.ActionTypePublish,
		Slug: "target-skill", Status: model.ReviewStatusPending,
	})
	model.DB(context.Background()).Create(&model.ReviewRequest{
		RequesterID: emp.ID, ResourceType: model.ResourceTypeSkill,
		ResourceID: s2.ID, ActionType: model.ActionTypePublish,
		Slug: "other", Status: model.ReviewStatusPending,
	})

	req := createAdminReq("GET", "/admin/contributions?keyword=target", "")
	w := httptest.NewRecorder()
	HandleAdminContributions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	requests := resp["requests"].([]interface{})
	if len(requests) != 1 {
		t.Errorf("期望 1 条结果（keyword 过滤），实际=%d", len(requests))
	}
}

func TestHandleAdminContributions_FilterByActionType(t *testing.T) {
	setupContributionTestDB(t)
	emp := seedEmployeeUser(t)
	s1 := seedPublishedSkill(t, "s1", "1.0.0", emp.ID)
	s2 := seedPendingReviewSkill(t, "s2", "1.0.0", emp.ID)
	model.DB(context.Background()).Create(&model.ReviewRequest{
		RequesterID: emp.ID, ResourceType: model.ResourceTypeSkill,
		ResourceID: s1.ID, ActionType: model.ActionTypeTakedown,
		Slug: "s1", Status: model.ReviewStatusPending,
	})
	model.DB(context.Background()).Create(&model.ReviewRequest{
		RequesterID: emp.ID, ResourceType: model.ResourceTypeSkill,
		ResourceID: s2.ID, ActionType: model.ActionTypePublish,
		Slug: "s2", Status: model.ReviewStatusPending,
	})

	req := createAdminReq("GET", "/admin/contributions?action_type=takedown", "")
	w := httptest.NewRecorder()
	HandleAdminContributions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	requests := resp["requests"].([]interface{})
	if len(requests) != 1 {
		t.Errorf("期望 1 条 takedown，实际=%d", len(requests))
	}
}

func TestHandleAdminContributionDetail_Success(t *testing.T) {
	setupContributionTestDB(t)
	emp := seedEmployeeUser(t)
	skill := seedPendingReviewSkill(t, "my-skill", "1.0.0", emp.ID)
	model.DB(context.Background()).Create(&model.ReviewRequest{
		RequesterID: emp.ID, ResourceType: model.ResourceTypeSkill,
		ResourceID: skill.ID, ActionType: model.ActionTypePublish,
		Slug: "my-skill", Status: model.ReviewStatusPending,
	})

	req := createAdminReq("GET", "/admin/contributions/detail?id=1", "")
	w := httptest.NewRecorder()
	HandleAdminContributionDetail(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["request"] == nil {
		t.Fatal("期望 response 包含 request")
	}
	if resp["requester_name"] != "employee" {
		t.Errorf("期望 requester_name=employee，实际=%v", resp["requester_name"])
	}
}

func TestHandleAdminContributionDetail_NotFound(t *testing.T) {
	setupContributionTestDB(t)

	req := createAdminReq("GET", "/admin/contributions/detail?id=999", "")
	w := httptest.NewRecorder()
	HandleAdminContributionDetail(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际=%d", w.Code)
	}
}

// ── 员工提交参数校验测试 ──────────────────────────────────────────────

func TestHandleContributeSkill_MissingFields(t *testing.T) {
	setupContributionTestDB(t)
	seedEmployeeUser(t)

	req := createEmployeeSessionReq(t, "POST", "/openclaw/skills/contribute", "")
	// 不设置 Content-Type 为 multipart，ParseMultipartForm 会失败，触发 i18n MsgRequestBodyTooLargeWithError
	w := httptest.NewRecorder()
	HandleContributeSkill(w, req)

	// 非 multipart 请求导致 ParseMultipartForm 错误
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleContributeSkill_NotLoggedIn(t *testing.T) {
	setupContributionTestDB(t)

	req := httptest.NewRequest("POST", "/openclaw/skills/contribute", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	HandleContributeSkill(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("期望 401，实际=%d", w.Code)
	}
}

func TestHandleContributeSkill_MutexConflict(t *testing.T) {
	setupContributionTestDB(t)
	emp := seedEmployeeUser(t)
	model.DB(context.Background()).Create(&model.ReviewRequest{
		RequesterID:  emp.ID,
		ResourceType: model.ResourceTypeSkill,
		ResourceID:   1,
		ActionType:   model.ActionTypePublish,
		Slug:         "existing-slug",
		Status:       model.ReviewStatusPending,
	})

	// 构造简单的 multipart 请求（无文件，但参数合法），会在 mutex 检查阶段被拦截
	bodyStr := "--boundary\r\nContent-Disposition: form-data; name=\"slug\"\r\n\r\nexisting-slug\r\n--boundary\r\nContent-Disposition: form-data; name=\"name\"\r\n\r\nTest\r\n--boundary\r\nContent-Disposition: form-data; name=\"version\"\r\n\r\n2.0.0\r\n--boundary--\r\n"
	req := httptest.NewRequest("POST", "/openclaw/skills/contribute", strings.NewReader(bodyStr))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	req.Header.Set("Accept", "application/json")
	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = "employee"
	rr := httptest.NewRecorder()
	session.Save(req, rr)
	for _, cookie := range rr.Result().Cookies() {
		req.AddCookie(cookie)
	}

	w := httptest.NewRecorder()
	HandleContributeSkill(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400（互斥），实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// ── 管理员直接下架/上架测试 ───────────────────────────────────────────

func TestHandleAdminSkillOffline_Success(t *testing.T) {
	setupContributionTestDB(t)
	emp := seedEmployeeUser(t)
	skill := seedPublishedSkill(t, "test-skill", "1.0.0", emp.ID)

	body := "slug=test-skill"
	req := httptest.NewRequest("POST", "/admin/skills/offline", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleAdminSkillOffline(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	var updated model.Skill
	model.DB(context.Background()).Where("id = ?", skill.ID).First(&updated)
	if updated.Status != model.SkillStatusOffline {
		t.Errorf("期望 status=offline，实际=%s", updated.Status)
	}
}

func TestHandleAdminSkillOnline_Success(t *testing.T) {
	setupContributionTestDB(t)
	emp := seedEmployeeUser(t)
	skill := &model.Skill{
		Slug: "test-skill", Name: "Test", Version: "1.0.0",
		VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
		Status: model.SkillStatusOffline, UploaderID: emp.ID,
	}
	model.DB(context.Background()).Create(skill)

	body := "slug=test-skill"
	req := httptest.NewRequest("POST", "/admin/skills/online", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleAdminSkillOnline(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	var updated model.Skill
	model.DB(context.Background()).Where("id = ?", skill.ID).First(&updated)
	if updated.Status != model.SkillStatusPublished {
		t.Errorf("期望 status=published，实际=%s", updated.Status)
	}
}

func TestHandleAdminSkillOffline_NotFound(t *testing.T) {
	setupContributionTestDB(t)

	body := "slug=nonexistent"
	req := httptest.NewRequest("POST", "/admin/skills/offline", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleAdminSkillOffline(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际=%d", w.Code)
	}
}

func TestHandleAdminSkillOnline_NotFound(t *testing.T) {
	setupContributionTestDB(t)

	body := "slug=nonexistent"
	req := httptest.NewRequest("POST", "/admin/skills/online", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleAdminSkillOnline(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际=%d", w.Code)
	}
}

// ── 员工提交更多校验测试 ──────────────────────────────────────────────

func TestHandleContributeSkill_InvalidSlug(t *testing.T) {
	setupContributionTestDB(t)
	seedEmployeeUser(t)

	bodyStr := "--boundary\r\nContent-Disposition: form-data; name=\"slug\"\r\n\r\nINVALID_SLUG!\r\n--boundary\r\nContent-Disposition: form-data; name=\"name\"\r\n\r\nTest\r\n--boundary\r\nContent-Disposition: form-data; name=\"version\"\r\n\r\n1.0.0\r\n--boundary--\r\n"
	req := httptest.NewRequest("POST", "/openclaw/skills/contribute", strings.NewReader(bodyStr))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	req.Header.Set("Accept", "application/json")
	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = "employee"
	rr := httptest.NewRecorder()
	session.Save(req, rr)
	for _, cookie := range rr.Result().Cookies() {
		req.AddCookie(cookie)
	}

	w := httptest.NewRecorder()
	HandleContributeSkill(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400（非法 slug），实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleContributeSkill_VersionNotIncremented(t *testing.T) {
	setupContributionTestDB(t)
	emp := seedEmployeeUser(t)
	seedPublishedSkill(t, "test-skill", "2.0.0", emp.ID)

	// 提交一个更低的版本号
	bodyStr := "--boundary\r\nContent-Disposition: form-data; name=\"slug\"\r\n\r\ntest-skill\r\n--boundary\r\nContent-Disposition: form-data; name=\"name\"\r\n\r\nTest\r\n--boundary\r\nContent-Disposition: form-data; name=\"version\"\r\n\r\n1.0.0\r\n--boundary--\r\n"
	req := httptest.NewRequest("POST", "/openclaw/skills/contribute", strings.NewReader(bodyStr))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	req.Header.Set("Accept", "application/json")
	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = "employee"
	rr := httptest.NewRecorder()
	session.Save(req, rr)
	for _, cookie := range rr.Result().Cookies() {
		req.AddCookie(cookie)
	}

	w := httptest.NewRecorder()
	HandleContributeSkill(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400（版本号未递增），实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleContributeSkill_DuplicateVersion(t *testing.T) {
	setupContributionTestDB(t)
	emp := seedEmployeeUser(t)
	seedPublishedSkill(t, "test-skill", "1.0.0", emp.ID)

	// 提交重复版本号（非删除记录）
	bodyStr := "--boundary\r\nContent-Disposition: form-data; name=\"slug\"\r\n\r\ntest-skill\r\n--boundary\r\nContent-Disposition: form-data; name=\"name\"\r\n\r\nTest\r\n--boundary\r\nContent-Disposition: form-data; name=\"version\"\r\n\r\n1.0.0\r\n--boundary--\r\n"
	req := httptest.NewRequest("POST", "/openclaw/skills/contribute", strings.NewReader(bodyStr))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	req.Header.Set("Accept", "application/json")
	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = "employee"
	rr := httptest.NewRecorder()
	session.Save(req, rr)
	for _, cookie := range rr.Result().Cookies() {
		req.AddCookie(cookie)
	}

	w := httptest.NewRecorder()
	HandleContributeSkill(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400（版本重复），实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleContributeSkill_MissingSlug(t *testing.T) {
	setupContributionTestDB(t)
	seedEmployeeUser(t)

	bodyStr := "--boundary\r\nContent-Disposition: form-data; name=\"name\"\r\n\r\nTest\r\n--boundary\r\nContent-Disposition: form-data; name=\"version\"\r\n\r\n1.0.0\r\n--boundary--\r\n"
	req := httptest.NewRequest("POST", "/openclaw/skills/contribute", strings.NewReader(bodyStr))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	req.Header.Set("Accept", "application/json")
	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = "employee"
	rr := httptest.NewRecorder()
	session.Save(req, rr)
	for _, cookie := range rr.Result().Cookies() {
		req.AddCookie(cookie)
	}

	w := httptest.NewRecorder()
	HandleContributeSkill(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400（缺少 slug），实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// ── 管理员贡献列表更多测试 ─────────────────────────────────────────────

func TestHandleAdminContributions_EmptyList(t *testing.T) {
	setupContributionTestDB(t)

	req := createAdminReq("GET", "/admin/contributions", "")
	w := httptest.NewRecorder()
	HandleAdminContributions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	requests := resp["requests"].([]interface{})
	if len(requests) != 0 {
		t.Errorf("期望空列表，实际=%d", len(requests))
	}
}

// ── 最后缺口补齐 ──────────────────────────────────────────────────────

func TestHandleTakedownSkill_MissingSlug(t *testing.T) {
	setupContributionTestDB(t)
	seedEmployeeUser(t)

	body := `{"slug":"","reason":"test"}`
	req := createEmployeeSessionReq(t, "POST", "/openclaw/skills/takedown", body)
	w := httptest.NewRecorder()
	HandleTakedownSkill(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400（slug 为空），实际=%d", w.Code)
	}
}

func TestHandleAdminContributionDetail_MissingID(t *testing.T) {
	setupContributionTestDB(t)

	req := createAdminReq("GET", "/admin/contributions/detail", "")
	w := httptest.NewRecorder()
	HandleAdminContributionDetail(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400（缺少 id），实际=%d", w.Code)
	}
}

func TestHandleContributeSkill_InvalidVersionFormat(t *testing.T) {
	setupContributionTestDB(t)
	seedEmployeeUser(t)

	bodyStr := "--boundary\r\nContent-Disposition: form-data; name=\"slug\"\r\n\r\ntest-skill\r\n--boundary\r\nContent-Disposition: form-data; name=\"name\"\r\n\r\nTest\r\n--boundary\r\nContent-Disposition: form-data; name=\"version\"\r\n\r\nnot-a-version\r\n--boundary--\r\n"
	req := httptest.NewRequest("POST", "/openclaw/skills/contribute", strings.NewReader(bodyStr))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	req.Header.Set("Accept", "application/json")
	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = "employee"
	rr := httptest.NewRecorder()
	session.Save(req, rr)
	for _, cookie := range rr.Result().Cookies() {
		req.AddCookie(cookie)
	}

	w := httptest.NewRecorder()
	HandleContributeSkill(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400（版本号格式非法），实际=%d, body=%s", w.Code, w.Body.String())
	}
}

// seedPublishedSkillAtVersion 创建指定语义化版本的 published 技能（正确填充 VersionMajor/Minor/Patch）。
func seedPublishedSkillAtVersion(t *testing.T, slug, version string, uploaderID uint) *model.Skill {
	t.Helper()
	s := &model.Skill{
		Slug:       slug,
		Name:       "Test Skill",
		Version:    version,
		Status:     model.SkillStatusPublished,
		UploaderID: uploaderID,
	}
	if err := s.ParseVersion(); err != nil {
		t.Fatalf("解析版本失败: %v", err)
	}
	if err := model.DB(context.Background()).Create(s).Error; err != nil {
		t.Fatalf("创建技能失败: %v", err)
	}
	return s
}

// ── 多版本下架修复 ───────────────────────────────────────────────────

func TestHandleTakedownSkill_MultiVersion_BindsLatest(t *testing.T) {
	setupContributionTestDB(t)
	emp := seedEmployeeUser(t)
	old := seedPublishedSkillAtVersion(t, "multi-skill", "1.0.0", emp.ID)
	latest := seedPublishedSkillAtVersion(t, "multi-skill", "1.0.1", emp.ID)
	if old.ID >= latest.ID {
		t.Fatalf("期望旧版本 id < 新版本 id，got old=%d latest=%d", old.ID, latest.ID)
	}

	body := `{"slug":"multi-skill","reason":"下架多版本"}`
	req := createEmployeeSessionReq(t, "POST", "/openclaw/skills/takedown", body)
	w := httptest.NewRecorder()
	HandleTakedownSkill(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	var rr model.ReviewRequest
	if model.DB(context.Background()).Where("slug = ? AND action_type = ?", "multi-skill", model.ActionTypeTakedown).First(&rr).Error != nil {
		t.Fatal("ReviewRequest 未创建")
	}
	if rr.ResourceID != latest.ID {
		t.Errorf("期望 resource_id=最新版 %d，实际=%d（旧版=%d）", latest.ID, rr.ResourceID, old.ID)
	}
}

func TestHandleAdminSkills_PendingReview_BySlug_OldResourceID(t *testing.T) {
	setupContributionTestDB(t)
	emp := seedEmployeeUser(t)
	old := seedPublishedSkillAtVersion(t, "multi-skill", "1.0.0", emp.ID)
	latest := seedPublishedSkillAtVersion(t, "multi-skill", "1.0.1", emp.ID)

	// 模拟线上存量：pending 挂在旧版本 resource_id
	pr := &model.ReviewRequest{
		RequesterID:  emp.ID,
		ResourceType: model.ResourceTypeSkill,
		ResourceID:   old.ID,
		ActionType:   model.ActionTypeTakedown,
		Slug:         "multi-skill",
		Status:       model.ReviewStatusPending,
		Reason:       "存量挂旧 id",
	}
	if err := model.DB(context.Background()).Create(pr).Error; err != nil {
		t.Fatalf("创建 pending 失败: %v", err)
	}

	req := createAdminReq("GET", "/admin/skills?slug=multi-skill&page=1&page_size=10", "")
	w := httptest.NewRecorder()
	HandleAdminSkills(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Skills []struct {
			ID            uint   `json:"ID"`
			Slug          string `json:"slug"`
			Version       string `json:"version"`
			PendingReview *struct {
				RequestID  uint   `json:"request_id"`
				ActionType string `json:"action_type"`
			} `json:"pending_review"`
		} `json:"skills"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v body=%s", err, w.Body.String())
	}
	if len(resp.Skills) != 1 {
		t.Fatalf("期望列表仅最新 1 条，实际=%d", len(resp.Skills))
	}
	row := resp.Skills[0]
	if row.Version != "1.0.1" && row.ID != latest.ID {
		// GORM json 可能用小写 id；以 version 为准
		if row.Version != "1.0.1" {
			t.Fatalf("期望展示最新版 1.0.1，实际 version=%s id=%d", row.Version, row.ID)
		}
	}
	if row.PendingReview == nil {
		t.Fatal("期望 pending_review 非空（按 slug 关联旧 resource_id）")
	}
	if row.PendingReview.ActionType != model.ActionTypeTakedown {
		t.Errorf("期望 action_type=takedown，实际=%s", row.PendingReview.ActionType)
	}
	if row.PendingReview.RequestID != pr.ID {
		t.Errorf("期望 request_id=%d，实际=%d", pr.ID, row.PendingReview.RequestID)
	}
}

func TestHandleApprove_Takedown_MultiVersion_AllOffline(t *testing.T) {
	setupContributionTestDB(t)
	emp := seedEmployeeUser(t)
	old := seedPublishedSkillAtVersion(t, "multi-skill", "1.0.0", emp.ID)
	latest := seedPublishedSkillAtVersion(t, "multi-skill", "1.0.1", emp.ID)

	req := &model.ReviewRequest{
		RequesterID:  emp.ID,
		ResourceType: model.ResourceTypeSkill,
		ResourceID:   old.ID, // 故意挂旧版
		ActionType:   model.ActionTypeTakedown,
		Slug:         "multi-skill",
		Status:       model.ReviewStatusPending,
		Reason:       "整 slug 下架",
	}
	model.DB(context.Background()).Create(req)

	body := `{"id":1}`
	r := createAdminReq("POST", "/admin/contributions/approve", body)
	w := httptest.NewRecorder()
	HandleApproveContribution(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	var skills []model.Skill
	model.DB(context.Background()).Where("slug = ?", "multi-skill").Find(&skills)
	if len(skills) != 2 {
		t.Fatalf("期望 2 个版本，实际=%d", len(skills))
	}
	for _, s := range skills {
		if s.Status != model.SkillStatusOffline {
			t.Errorf("期望 id=%d version=%s 为 offline，实际=%s", s.ID, s.Version, s.Status)
		}
	}
	_ = latest
	assertReviewNotification(t, emp.ID, "skill_takedown_approved", "multi-skill")
}

func TestHandleReject_Takedown_MultiVersion_StayPublished(t *testing.T) {
	setupContributionTestDB(t)
	emp := seedEmployeeUser(t)
	seedPublishedSkillAtVersion(t, "multi-skill", "1.0.0", emp.ID)
	seedPublishedSkillAtVersion(t, "multi-skill", "1.0.1", emp.ID)

	req := &model.ReviewRequest{
		RequesterID:  emp.ID,
		ResourceType: model.ResourceTypeSkill,
		ResourceID:   1,
		ActionType:   model.ActionTypeTakedown,
		Slug:         "multi-skill",
		Status:       model.ReviewStatusPending,
		Reason:       "拒绝应保持 published",
	}
	model.DB(context.Background()).Create(req)

	body := `{"id":1,"review_comment":"暂不下架"}`
	r := createAdminReq("POST", "/admin/contributions/reject", body)
	w := httptest.NewRecorder()
	HandleRejectContribution(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	var skills []model.Skill
	model.DB(context.Background()).Where("slug = ?", "multi-skill").Find(&skills)
	for _, s := range skills {
		if s.Status != model.SkillStatusPublished {
			t.Errorf("拒绝后期望仍 published，id=%d status=%s", s.ID, s.Status)
		}
	}
}

func TestHandleTakedownSkill_MultiVersion_OwnerOK(t *testing.T) {
	setupContributionTestDB(t)
	emp := seedEmployeeUser(t)
	seedPublishedSkillAtVersion(t, "multi-skill", "1.0.0", emp.ID)
	seedPublishedSkillAtVersion(t, "multi-skill", "1.0.1", emp.ID)

	body := `{"slug":"multi-skill","reason":"本人多版本"}`
	req := createEmployeeSessionReq(t, "POST", "/openclaw/skills/takedown", body)
	w := httptest.NewRecorder()
	HandleTakedownSkill(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

