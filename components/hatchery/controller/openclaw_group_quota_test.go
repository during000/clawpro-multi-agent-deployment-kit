package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"hatchery/controller/usergroup"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

var groupQuotaDBSeq int64

func initGroupQuotaTestDB(t *testing.T) func() {
	t.Helper()
	dsn := fmt.Sprintf("file:gqtest_%d?mode=memory&cache=shared", atomic.AddInt64(&groupQuotaDBSeq, 1))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.User{}, &model.Instance{}, &model.SiteConfig{},
		&model.AIImage{}, &model.AuditLog{}, &model.Notification{},
		&model.GroupConfigBinding{}, &model.OpenClawRole{},
		&model.RoleVisibilityGroup{}, &model.UserGroup{}, &model.GroupClosure{},
		&model.AIModel{}, &model.AIChannel{},
		&model.Skill{}, &model.SkillBundle{},
		&model.BundleSkill{}, &model.OpenClawRoleSkill{},
		&model.Tag{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	origDB := model.UseDBForTestWithDriver(db, "sqlite")
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	AdminToken = "test-admin-token"

	return func() {
		origDB()
		Store = origStore
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}
}

func seedGroupQuotaFixtures(t *testing.T, username string, userQuota int, groupName string, groupQuota int, existingCount int) (*model.User, *model.UserGroup) {
	t.Helper()
	ctx := context.Background()

	user := &model.User{Username: username, Password: "x", Role: "user", InstanceQuota: userQuota}
	if err := model.DB(ctx).Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	group := &model.UserGroup{Name: groupName, FullPath: groupName, Source: "manual"}
	if err := model.DB(ctx).Create(group).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}
	if err := model.DB(ctx).Create(&model.GroupClosure{
		AncestorID: group.ID, DescendantID: group.ID, Depth: 0,
	}).Error; err != nil {
		t.Fatalf("create closure: %v", err)
	}

	if err := model.DB(ctx).Create(&model.GroupConfigBinding{
		ConfigType: model.ConfigTypePolicy,
		ConfigKey:  usergroup.PolicyKeyInstanceQuota,
		GroupID:    group.ID,
		ValueJSON:  fmt.Sprintf(`{"value":%d}`, groupQuota),
	}).Error; err != nil {
		t.Fatalf("create policy binding: %v", err)
	}

	for i := 0; i < existingCount; i++ {
		ins := &model.Instance{
			Name:       fmt.Sprintf("%s-existing-%d", groupName, i),
			InstanceId: fmt.Sprintf("ins-%s-existing-%d", groupName, i),
			UserID:     user.ID, GroupID: group.ID, AgentType: model.AgentTypeOpenClaw,
		}
		if err := model.DB(ctx).Create(ins).Error; err != nil {
			t.Fatalf("create existing instance: %v", err)
		}
	}

	img := &model.AIImage{
		ImageId:   fmt.Sprintf("img-%s", groupName),
		ImageName: groupName,
		ImageType: "PRIVATE_IMAGE",
		AgentType: model.AgentTypeOpenClaw, AgentVersion: "1.0.0", Enabled: true,
	}
	if err := model.DB(ctx).Create(img).Error; err != nil {
		t.Fatalf("create image: %v", err)
	}

	return user, group
}

func TestHandleCreateInstance_GroupQuotaExceeded(t *testing.T) {
	cleanup := initGroupQuotaTestDB(t)
	defer cleanup()

	user, group := seedGroupQuotaFixtures(t, "u1", 10, "qgroup", 1, 1)

	form := url.Values{}
	form.Set("name", "quota-exceed-grp")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	form.Set("group_id", fmt.Sprintf("%d", group.ID))
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u1", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("group quota exceeded should return 403, got=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "实例数量已达上限") {
		t.Errorf("error body should contain group-quota i18n message, got=%s", rr.Body.String())
	}

	var count int64
	model.DB(context.Background()).Model(&model.Instance{}).
		Where("user_id = ? AND group_id = ?", user.ID, group.ID).Count(&count)
	if count != 1 {
		t.Errorf("should still have 1 instance (placeholder rolled back), got=%d", count)
	}
}

func TestHandleCreateInstance_GroupQuotaBoundaryEqual(t *testing.T) {
	cleanup := initGroupQuotaTestDB(t)
	defer cleanup()

	user, group := seedGroupQuotaFixtures(t, "u-boundary", 100, "boundary-grp", 2, 2)

	form := url.Values{}
	form.Set("name", "boundary-new")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	form.Set("group_id", fmt.Sprintf("%d", group.ID))
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u-boundary", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("group quota boundary (count == quota) should return 403, got=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "实例数量已达上限") {
		t.Errorf("error body should contain quota-reached i18n message, got=%s", rr.Body.String())
	}

	var groupCount int64
	model.DB(context.Background()).Model(&model.Instance{}).
		Where("user_id = ? AND group_id = ?", user.ID, group.ID).Count(&groupCount)
	if groupCount != 2 {
		t.Errorf("group instance count should remain 2 (placeholder rolled back), got=%d", groupCount)
	}
}

func TestHandleCreateInstance_GroupQuotaIndependentFromGlobal(t *testing.T) {
	cleanup := initGroupQuotaTestDB(t)
	defer cleanup()

	user, group := seedGroupQuotaFixtures(t, "u-indep", 999, "indep-grp", 1, 1)

	otherGroup := &model.UserGroup{Name: "other-grp", FullPath: "other-grp", Source: "manual"}
	if err := model.DB(context.Background()).Create(otherGroup).Error; err != nil {
		t.Fatalf("create other group: %v", err)
	}
	if err := model.DB(context.Background()).Create(&model.GroupClosure{
		AncestorID: otherGroup.ID, DescendantID: otherGroup.ID, Depth: 0,
	}).Error; err != nil {
		t.Fatalf("create other closure: %v", err)
	}
	for i := 0; i < 5; i++ {
		ins := &model.Instance{
			Name:       fmt.Sprintf("other-%d", i),
			InstanceId: fmt.Sprintf("ins-other-%d", i),
			UserID:     user.ID, GroupID: otherGroup.ID, AgentType: model.AgentTypeOpenClaw,
		}
		if err := model.DB(context.Background()).Create(ins).Error; err != nil {
			t.Fatalf("create other instance: %v", err)
		}
	}

	form := url.Values{}
	form.Set("name", "indep-new")
	form.Set("agent_type", model.AgentTypeOpenClaw)
	form.Set("group_id", fmt.Sprintf("%d", group.ID))
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "u-indep", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("group quota should be evaluated independently of global quota, expect 403, got=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "实例数量已达上限") {
		t.Errorf("error body should contain quota-reached i18n message, got=%s", rr.Body.String())
	}

	var totalCount int64
	model.DB(context.Background()).Model(&model.Instance{}).Where("user_id = ?", user.ID).Count(&totalCount)
	if totalCount != 6 {
		t.Errorf("total instance count should remain 6 (no placeholder leaked), got=%d", totalCount)
	}

	var targetGroupCount int64
	model.DB(context.Background()).Model(&model.Instance{}).
		Where("user_id = ? AND group_id = ?", user.ID, group.ID).Count(&targetGroupCount)
	if targetGroupCount != 1 {
		t.Errorf("target group instance count should remain 1, got=%d", targetGroupCount)
	}
}
