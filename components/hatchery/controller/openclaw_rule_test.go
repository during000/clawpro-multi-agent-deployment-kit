package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// 企业规范库单测：admin/instances/rules
// ---------------------------------------------------------------------------

// initRuleCommandsTestDB 初始化 rule commands 测试用 DB。
// 与 initLocalCommandsTestDB 对称，额外 AutoMigrate 企业规范库相关的 4 张表。
func initRuleCommandsTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if sqlDB, e := db.DB(); e == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.Instance{},
		&model.LocalInstanceInfo{},
		&model.EnterpriseRule{},
		&model.RuleDistributionTask{},
		&model.RuleDistributionRecord{},
		&model.LocalInstanceRule{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	origDB := model.UseDBForTestWithDriver(db, "sqlite")
	if Store == nil {
		Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	}
	t.Cleanup(func() {
		origDB()
	})
}

// seedRuleCommandsFixture 建用户 + 本地 agent 实例 + 心跳行（与 seedLocalCommandsFixture 对称）
func seedRuleCommandsFixture(t *testing.T) (*model.User, *model.Instance) {
	t.Helper()
	ctx := context.Background()
	user := &model.User{Username: "u-rule-cmd", Password: "x", Role: "user"}
	if err := model.DB(ctx).Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	inst := &model.Instance{
		Name: "local-rule-agent", InstanceId: "local-rule-agent-abcdef",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	if err := model.DB(ctx).Create(inst).Error; err != nil {
		t.Fatalf("create inst: %v", err)
	}
	now := time.Now()
	if err := model.DB(ctx).Create(&model.LocalInstanceInfo{
		InstanceID: inst.ID, LastReportAt: &now,
	}).Error; err != nil {
		t.Fatalf("create info: %v", err)
	}
	return user, inst
}

// ─── HandleAdminInstanceRules ────────────────────────────────────────────

// TestHandleAdminInstanceRules_LocalInstance_ReturnsInstalledRules
// 本地实例有已安装规范时，返回规范列表。
func TestHandleAdminInstanceRules_LocalInstance_ReturnsInstalledRules(t *testing.T) {
	initRuleCommandsTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	t.Cleanup(func() { AdminToken = origToken })
	_, inst := seedRuleCommandsFixture(t)

	// 模拟已安装的规范
	now := time.Now()
	rules := []model.LocalInstanceRule{
		{InstanceID: inst.ID, Slug: "coding-standards", Version: "1.2.0",
			DisplayName: "编码规范", RuleType: "rule", Source: "enterprise", InstalledAt: &now},
		{InstanceID: inst.ID, Slug: "codebuddy-prompt", Version: "0.5.0",
			DisplayName: "CodeBuddy Prompt", RuleType: "prompt", Source: "enterprise", InstalledAt: &now},
	}
	for _, r := range rules {
		if err := model.DB(context.Background()).Create(&r).Error; err != nil {
			t.Fatalf("create local_instance_rule: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/admin/instances/rules?id=%d", inst.ID), nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	rr := httptest.NewRecorder()
	HandleAdminInstanceRules(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		InstanceID uint                     `json:"instance_id"`
		Rules      []map[string]interface{} `json:"rules"`
		Total      int                      `json:"total"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v body=%s", err, rr.Body.String())
	}
	if resp.Total != 2 {
		t.Fatalf("应返回 2 条规范，total=%d body=%s", resp.Total, rr.Body.String())
	}

	// 验证第一条规范字段
	item := resp.Rules[0]
	if got, _ := item["slug"].(string); got != "codebuddy-prompt" {
		t.Errorf("slug 应=codebuddy-prompt，实际=%v", item["slug"])
	}
	if got, _ := item["name"].(string); got != "CodeBuddy Prompt" {
		t.Errorf("name 应=CodeBuddy Prompt，实际=%v", item["name"])
	}
	if got, _ := item["type"].(string); got != "prompt" {
		t.Errorf("type 应=prompt，实际=%v", item["type"])
	}
	if got, _ := item["version"].(string); got != "0.5.0" {
		t.Errorf("version 应=0.5.0，实际=%v", item["version"])
	}
	if got, _ := item["source"].(string); got != "enterprise" {
		t.Errorf("source 应=enterprise，实际=%v", item["source"])
	}
}

// TestHandleAdminInstanceRules_NoRules_ReturnsEmptyList
// 本地实例无已安装规范时，返回空列表。
func TestHandleAdminInstanceRules_NoRules_ReturnsEmptyList(t *testing.T) {
	initRuleCommandsTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	t.Cleanup(func() { AdminToken = origToken })
	_, inst := seedRuleCommandsFixture(t)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/admin/instances/rules?id=%d", inst.ID), nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	rr := httptest.NewRecorder()
	HandleAdminInstanceRules(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Rules []map[string]interface{} `json:"rules"`
		Total int                      `json:"total"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v body=%s", err, rr.Body.String())
	}
	if resp.Total != 0 {
		t.Fatalf("无规范时应返回空列表，total=%d", resp.Total)
	}
}

// TestHandleAdminInstanceRules_RejectsNonLocalInstance
// 非本地实例（CVM）应拒绝。
func TestHandleAdminInstanceRules_RejectsNonLocalInstance(t *testing.T) {
	initRuleCommandsTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	t.Cleanup(func() { AdminToken = origToken })
	_, inst := seedRuleCommandsFixture(t)
	// 把实例改成 CVM
	model.DB(context.Background()).Model(&inst).Update("source", model.InstanceSourceCVM)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/admin/instances/rules?id=%d", inst.ID), nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	rr := httptest.NewRecorder()
	HandleAdminInstanceRules(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("CVM 实例应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}
