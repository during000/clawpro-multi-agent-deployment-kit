package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// setupPhase2DB 初始化二期测试 DB，包含所需的所有表。
func setupPhase2DB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if sqlDB, e := db.DB(); e == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	if err := db.AutoMigrate(
		&model.Instance{},
		&model.User{},
		&model.Skill{},
		&model.SkillDistributionRecord{},
		&model.SkillDistributionTask{},
		&model.SiteConfig{},
		&model.UserGroup{},
		&model.UserGroupMember{},
		&model.GroupClosure{},
		&model.GroupConfigBinding{},
		&model.Project{},
		&model.ProjectMember{},
		&model.ProjectConfigBinding{},
		&model.LocalAgentScopeBinding{},
		&model.LocalInstanceInfo{},
		&model.LocalInstanceSkill{},
		&model.LocalInstanceRule{},
		&model.EnterpriseRule{},
		&model.RuleDistributionRecord{},
		&model.RuleDistributionTask{},
		&model.FeatureAllowlist{},
		&model.SkillVisibilityGroup{},
		&model.RuleVisibilityGroup{},
	); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	// 替换全局 DB
	t.Cleanup(useDBForTestWithSafeRestore(db))
	// 开启本地 Agent 功能
	model.DBGlobal(context.Background()).Model(&model.SiteConfig{}).
		Where("1 = 1").Update("local_agent_enabled", true)
	return db
}

// createTestUserAndGroup 创建测试用户 + 分组 + 分组成员关系。
func createTestUserAndGroup(t *testing.T, db *gorm.DB) (*model.User, *model.UserGroup) {
	t.Helper()
	user := &model.User{Username: "testuser", Password: "x", Email: "t@t.com"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	group := &model.UserGroup{Name: "G1", Description: "test group"}
	if err := db.Create(group).Error; err != nil {
		t.Fatalf("创建分组失败: %v", err)
	}
	member := &model.UserGroupMember{UserGroupID: group.ID, UserID: user.ID}
	if err := db.Create(member).Error; err != nil {
		t.Fatalf("创建分组成员失败: %v", err)
	}
	return user, group
}

// createTestLocalInstance 创建一个 source=local 的实例。
func createTestLocalInstance(t *testing.T, db *gorm.DB, userID uint) *model.Instance {
	t.Helper()
	inst := &model.Instance{
		UserID:     userID,
		InstanceId: "local-workbuddy-abc123",
		Name:       "test-instance",
		AgentType:  "workbuddy",
		Source:     model.InstanceSourceLocal,
	}
	if err := db.Create(inst).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}
	return inst
}

// createTestSkillWithVisibility 创建一个 skill + 分组资产绑定。
// 函数名为兼容已有测试保留；可见范围不参与本地 Agent 下发。
func createTestSkillWithVisibility(t *testing.T, db *gorm.DB, slug, version string, groupID uint) *model.Skill {
	t.Helper()
	skill := &model.Skill{
		Slug:           slug,
		Version:        version,
		Name:           slug + "-name",
		COSZipKey:      "cos/" + slug + ".zip",
		VisibilityType: "group",
	}
	if err := db.Create(skill).Error; err != nil {
		t.Fatalf("创建 skill 失败: %v", err)
	}
	binding := &model.GroupConfigBinding{GroupID: groupID, ConfigType: model.AssetBindingTypeSkill, ConfigKey: slug}
	if err := db.Create(binding).Error; err != nil {
		t.Fatalf("创建 skill 资产绑定失败: %v", err)
	}
	return skill
}

// ---- computeGroupActive 测试 ----

func TestComputeGroupActive_InGroup(t *testing.T) {
	db := setupPhase2DB(t)

	user, group := createTestUserAndGroup(t, db)

	if !computeGroupActive(context.Background(), user.ID, group.ID) {
		t.Errorf("用户在分组里，期望 true")
	}
}

func TestComputeGroupActive_NotInGroup(t *testing.T) {
	db := setupPhase2DB(t)

	user, _ := createTestUserAndGroup(t, db)

	if computeGroupActive(context.Background(), user.ID, 99999) {
		t.Errorf("用户不在分组里，期望 false")
	}
}

func TestComputeGroupActive_GroupIDZero(t *testing.T) {
	setupPhase2DB(t)

	if computeGroupActive(context.Background(), 1, 0) {
		t.Errorf("groupID=0，期望 false")
	}
}

// ---- diffAndQueue 测试 ----

func TestDiffAndQueue_NewSkill(t *testing.T) {
	db := setupPhase2DB(t)

	user, group := createTestUserAndGroup(t, db)
	inst := createTestLocalInstance(t, db, user.ID)
	createTestSkillWithVisibility(t, db, "skill-a", "1.0.0", group.ID)

	// 执行 diffAndQueue
	ctx := context.Background()
	var pendingCount int
	err := db.Transaction(func(tx *gorm.DB) error {
		c, e := diffAndQueue(ctx, tx, model.LocalSkillScopeUser, inst.ID, group.ID, "")
		pendingCount = c
		return e
	})
	if err != nil {
		t.Fatalf("diffAndQueue 失败: %v", err)
	}
	if pendingCount != 1 {
		t.Errorf("期望 1 个 pending，实际 %d", pendingCount)
	}

	// 验证 local_instance_skills
	var lis model.LocalInstanceSkill
	if err := db.Where("instance_id = ? AND scope = ?", inst.ID, model.LocalSkillScopeUser).First(&lis).Error; err != nil {
		t.Fatalf("local_instance_skills 未创建: %v", err)
	}
	if lis.InstallStatus != model.LocalSkillInstallStatusDistributing {
		t.Errorf("期望 install_status=distributing，实际 %s", lis.InstallStatus)
	}
	if lis.Version != "1.0.0" {
		t.Errorf("期望 version=1.0.0，实际 %s", lis.Version)
	}

	// 验证 skill_distribution_records
	var rec model.SkillDistributionRecord
	if err := db.Where("instance_id = ? AND status = ?", inst.ID, "pending").First(&rec).Error; err != nil {
		t.Fatalf("pending record 未创建: %v", err)
	}
}

func TestDiffAndQueue_SameVersionSkip(t *testing.T) {
	db := setupPhase2DB(t)

	user, group := createTestUserAndGroup(t, db)
	inst := createTestLocalInstance(t, db, user.ID)
	createTestSkillWithVisibility(t, db, "skill-a", "1.0.0", group.ID)

	ctx := context.Background()

	// 第一次 diffAndQueue → 写 pending
	err := db.Transaction(func(tx *gorm.DB) error {
		_, e := diffAndQueue(ctx, tx, model.LocalSkillScopeUser, inst.ID, group.ID, "")
		return e
	})
	if err != nil {
		t.Fatalf("第一次 diffAndQueue 失败: %v", err)
	}

	// 模拟 ack 成功 → install_status=distributed
	db.Model(&model.LocalInstanceSkill{}).
		Where("instance_id = ? AND scope = ?", inst.ID, model.LocalSkillScopeUser).
		Update("install_status", model.LocalSkillInstallStatusDistributed)

	// 模拟 record 终态
	db.Model(&model.SkillDistributionRecord{}).
		Where("instance_id = ? AND status = ?", inst.ID, "pending").
		Update("status", "success")

	// 第二次 diffAndQueue → 版本相同，应跳过
	var pendingCount int
	err = db.Transaction(func(tx *gorm.DB) error {
		c, e := diffAndQueue(ctx, tx, model.LocalSkillScopeUser, inst.ID, group.ID, "")
		pendingCount = c
		return e
	})
	if err != nil {
		t.Fatalf("第二次 diffAndQueue 失败: %v", err)
	}
	if pendingCount != 0 {
		t.Errorf("版本相同应跳过，期望 0 pending，实际 %d", pendingCount)
	}
}

func TestDiffAndQueue_VersionUpgrade(t *testing.T) {
	db := setupPhase2DB(t)

	user, group := createTestUserAndGroup(t, db)
	inst := createTestLocalInstance(t, db, user.ID)
	createTestSkillWithVisibility(t, db, "skill-a", "1.0.0", group.ID)

	ctx := context.Background()

	// 第一次 → 写 pending + distributing
	err := db.Transaction(func(tx *gorm.DB) error {
		_, e := diffAndQueue(ctx, tx, model.LocalSkillScopeUser, inst.ID, group.ID, "")
		return e
	})
	if err != nil {
		t.Fatalf("第一次 diffAndQueue 失败: %v", err)
	}

	// 模拟 ack 成功 → distributed
	db.Model(&model.LocalInstanceSkill{}).
		Where("instance_id = ? AND scope = ?", inst.ID, model.LocalSkillScopeUser).
		Update("install_status", model.LocalSkillInstallStatusDistributed)
	db.Model(&model.SkillDistributionRecord{}).
		Where("instance_id = ? AND status = ?", inst.ID, "pending").
		Update("status", "success")

	// 升级 skill 版本
	db.Model(&model.Skill{}).Where("slug = ?", "skill-a").Update("version", "2.0.0")

	// 第二次 → 版本不同，应写 pending
	var pendingCount int
	err = db.Transaction(func(tx *gorm.DB) error {
		c, e := diffAndQueue(ctx, tx, model.LocalSkillScopeUser, inst.ID, group.ID, "")
		pendingCount = c
		return e
	})
	if err != nil {
		t.Fatalf("第二次 diffAndQueue 失败: %v", err)
	}
	if pendingCount != 1 {
		t.Errorf("版本升级应写 pending，期望 1，实际 %d", pendingCount)
	}

	// 验证 install_status 回到 distributing
	var lis model.LocalInstanceSkill
	if err := db.Where("instance_id = ? AND scope = ?", inst.ID, model.LocalSkillScopeUser).First(&lis).Error; err != nil {
		t.Fatalf("local_instance_skills 未找到: %v", err)
	}
	if lis.InstallStatus != model.LocalSkillInstallStatusDistributing {
		t.Errorf("期望 install_status=distributing，实际 %s", lis.InstallStatus)
	}
}

func TestDiffAndQueue_CleansFailed(t *testing.T) {
	db := setupPhase2DB(t)

	user, group := createTestUserAndGroup(t, db)
	inst := createTestLocalInstance(t, db, user.ID)
	createTestSkillWithVisibility(t, db, "skill-a", "1.0.0", group.ID)

	// 预置一条 failed local_instance_skill
	now := time.Now()
	db.Create(&model.LocalInstanceSkill{
		InstanceID:    inst.ID,
		Slug:          "old-failed",
		Version:       "0.9",
		Source:        model.LocalSkillSourceEnterprise,
		Scope:         model.LocalSkillScopeUser,
		WorkspacePath: "",
		InstallStatus: model.LocalSkillInstallStatusFailed,
		LastSeenAt:    &now,
	})

	// 预置一条 failed record
	db.Create(&model.SkillDistributionRecord{
		InstanceID: inst.ID,
		Type:       "distribute",
		Version:    "0.9",
		Status:     "failed",
	})

	ctx := context.Background()
	err := db.Transaction(func(tx *gorm.DB) error {
		_, e := diffAndQueue(ctx, tx, model.LocalSkillScopeUser, inst.ID, group.ID, "")
		return e
	})
	if err != nil {
		t.Fatalf("diffAndQueue 失败: %v", err)
	}

	// failed local_instance_skills 应被硬删
	var count int64
	db.Model(&model.LocalInstanceSkill{}).
		Where("instance_id = ? AND install_status = ?", inst.ID, model.LocalSkillInstallStatusFailed).
		Count(&count)
	if count != 0 {
		t.Errorf("failed local_instance_skills 应被清理，剩余 %d", count)
	}

	// failed records 应标 cancelled
	var rec model.SkillDistributionRecord
	if err := db.Where("instance_id = ? AND status = ?", inst.ID, "cancelled").First(&rec).Error; err != nil {
		t.Errorf("failed record 应标 cancelled: %v", err)
	}
}

// ---- HandleSwitchUserLevelGroup 测试 ----

// switchUserGroupReq 构造切换分组请求。
func switchUserGroupReq(t *testing.T, username string, body any) *http.Request {
	t.Helper()
	encoded, _ := json.Marshal(body)
	return switchUserGroupRawReq(t, username, string(encoded))
}

func switchUserGroupRawReq(t *testing.T, username, body string) *http.Request {
	t.Helper()
	ensureSessionStore()
	req := httptest.NewRequest(http.MethodPost, "/openclaw/local-agent/user-group", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
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

func TestHandleSwitchUserLevelGroup_Success(t *testing.T) {
	db := setupPhase2DB(t)

	user, group := createTestUserAndGroup(t, db)
	inst := createTestLocalInstance(t, db, user.ID)
	createTestSkillWithVisibility(t, db, "skill-x", "1.0.0", group.ID)

	req := switchUserGroupReq(t, user.Username, map[string]any{
		"group_id":    group.ID,
		"instance_id": inst.ID,
	})
	rr := httptest.NewRecorder()
	HandleSwitchUserLevelGroup(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp["ok"] != true {
		t.Errorf("期望 ok=true, 实际 %v", resp["ok"])
	}

	// 验证 instances.local_agent_resources 被更新
	var updatedInst model.Instance
	db.First(&updatedInst, inst.ID)
	if updatedInst.LocalAgentResources == nil {
		t.Fatal("local_agent_resources 未更新")
	}
	if updatedInst.LocalAgentResources.UserLevel.GroupID != group.ID {
		t.Errorf("期望 group_id=%d, 实际 %d", group.ID, updatedInst.LocalAgentResources.UserLevel.GroupID)
	}
	var binding model.LocalAgentScopeBinding
	if err := db.Where("instance_id = ? AND scope = ? AND scope_key = ?", inst.ID, model.LocalAgentScopeUser, "").First(&binding).Error; err != nil {
		t.Fatalf("用户级 scope binding 未写入: %v", err)
	}
	if binding.GroupID != group.ID {
		t.Errorf("scope binding group_id 期望 %d，实际 %d", group.ID, binding.GroupID)
	}
}

func TestHandleSwitchUserLevelGroup_MethodNotAllowed(t *testing.T) {
	setupPhase2DB(t)

	req := httptest.NewRequest(http.MethodGet, "/openclaw/local-agent/user-group", nil)
	rr := httptest.NewRecorder()
	HandleSwitchUserLevelGroup(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望 405，实际 %d", rr.Code)
	}
}

func TestHandleSwitchUserLevelGroup_GroupNotBelongToUser(t *testing.T) {
	db := setupPhase2DB(t)

	user, _ := createTestUserAndGroup(t, db)
	inst := createTestLocalInstance(t, db, user.ID)

	// 创建一个用户不在的分组
	otherGroup := &model.UserGroup{Name: "OtherGroup", Description: ""}
	db.Create(otherGroup)

	req := switchUserGroupReq(t, user.Username, map[string]any{
		"group_id":    otherGroup.ID,
		"instance_id": inst.ID,
	})
	rr := httptest.NewRecorder()
	HandleSwitchUserLevelGroup(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("用户不在该分组，期望 400，实际 %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleSwitchUserLevelGroup_InstanceNotFound(t *testing.T) {
	db := setupPhase2DB(t)

	user, group := createTestUserAndGroup(t, db)

	req := switchUserGroupReq(t, user.Username, map[string]any{
		"group_id":    group.ID,
		"instance_id": 99999,
	})
	rr := httptest.NewRecorder()
	HandleSwitchUserLevelGroup(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("实例不存在，期望 404，实际 %d", rr.Code)
	}
}

func TestHandleSwitchUserLevelGroup_MissingGroupID(t *testing.T) {
	db := setupPhase2DB(t)

	user, _ := createTestUserAndGroup(t, db)
	inst := createTestLocalInstance(t, db, user.ID)

	req := switchUserGroupReq(t, user.Username, map[string]any{
		"instance_id": inst.ID,
	})
	rr := httptest.NewRecorder()
	HandleSwitchUserLevelGroup(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("缺少 group_id，期望 400，实际 %d", rr.Code)
	}
}

func TestHandleSwitchUserLevelGroup_RequestValidation(t *testing.T) {
	db := setupPhase2DB(t)
	user, group := createTestUserAndGroup(t, db)

	t.Run("unauthenticated", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/openclaw/local-agent/user-group",
			strings.NewReader(`{"group_id":1,"instance_id":1}`))
		req.Header.Set("Accept", "application/json")
		rr := httptest.NewRecorder()
		HandleSwitchUserLevelGroup(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("未登录期望 401，实际 %d body=%s", rr.Code, rr.Body.String())
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		req := switchUserGroupRawReq(t, user.Username, "{")
		rr := httptest.NewRecorder()
		HandleSwitchUserLevelGroup(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("非法 JSON 期望 400，实际 %d body=%s", rr.Code, rr.Body.String())
		}
	})

	t.Run("missing instance ID", func(t *testing.T) {
		req := switchUserGroupReq(t, user.Username, map[string]any{"group_id": group.ID})
		rr := httptest.NewRecorder()
		HandleSwitchUserLevelGroup(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("缺少 instance_id 期望 400，实际 %d body=%s", rr.Code, rr.Body.String())
		}
	})
}

func TestHandleSwitchUserLevelGroup_TransactionFailure(t *testing.T) {
	db := setupPhase2DB(t)
	user, group := createTestUserAndGroup(t, db)
	inst := createTestLocalInstance(t, db, user.ID)
	if err := db.Migrator().DropTable(&model.LocalAgentScopeBinding{}); err != nil {
		t.Fatalf("删除 scope binding 表: %v", err)
	}

	req := switchUserGroupReq(t, user.Username, map[string]any{
		"group_id":    group.ID,
		"instance_id": inst.ID,
	})
	rr := httptest.NewRecorder()
	HandleSwitchUserLevelGroup(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("事务失败期望 500，实际 %d body=%s", rr.Code, rr.Body.String())
	}

	var reloaded model.Instance
	if err := db.First(&reloaded, inst.ID).Error; err != nil {
		t.Fatalf("重新查询实例: %v", err)
	}
	if reloaded.LocalAgentResources != nil && reloaded.LocalAgentResources.UserLevel.GroupID != 0 {
		t.Fatalf("事务失败后资源更新未回滚: %#v", reloaded.LocalAgentResources)
	}
}

// ---- alignLocalSkills 测试 ----

func TestAlignLocalSkills_NewSkill(t *testing.T) {
	db := setupPhase2DB(t)

	user, _ := createTestUserAndGroup(t, db)
	inst := createTestLocalInstance(t, db, user.ID)

	now := time.Now()
	reported := []reportSkillEntry{
		{Slug: "new-skill", Version: "1.0", DisplayName: "New", Source: "enterprise"},
	}

	synced, err := alignLocalSkills(db, inst.ID, model.LocalSkillScopeUser, "", reported, now)
	if err != nil {
		t.Fatalf("alignLocalSkills 失败: %v", err)
	}
	if synced != 1 {
		t.Errorf("期望 synced=1，实际 %d", synced)
	}

	var lis model.LocalInstanceSkill
	if err := db.Where("instance_id = ? AND slug = ?", inst.ID, "new-skill").First(&lis).Error; err != nil {
		t.Fatalf("local_instance_skills 未创建: %v", err)
	}
	if lis.InstallStatus != model.LocalSkillInstallStatusDistributed {
		t.Errorf("期望 distributed，实际 %s", lis.InstallStatus)
	}
}

func TestAlignLocalSkills_DisappearDelete(t *testing.T) {
	db := setupPhase2DB(t)

	user, _ := createTestUserAndGroup(t, db)
	inst := createTestLocalInstance(t, db, user.ID)

	// 预置一条已装 skill
	now := time.Now()
	db.Create(&model.LocalInstanceSkill{
		InstanceID:    inst.ID,
		Slug:          "old-skill",
		Version:       "1.0",
		Source:        model.LocalSkillSourceEnterprise,
		Scope:         model.LocalSkillScopeUser,
		WorkspacePath: "",
		InstallStatus: model.LocalSkillInstallStatusDistributed,
		LastSeenAt:    &now,
	})

	// report 不含 old-skill → 应被删除
	reported := []reportSkillEntry{}
	synced, err := alignLocalSkills(db, inst.ID, model.LocalSkillScopeUser, "", reported, now)
	if err != nil {
		t.Fatalf("alignLocalSkills 失败: %v", err)
	}
	if synced != 0 {
		t.Errorf("期望 synced=0，实际 %d", synced)
	}

	var count int64
	db.Model(&model.LocalInstanceSkill{}).Where("instance_id = ? AND slug = ?", inst.ID, "old-skill").Count(&count)
	if count != 0 {
		t.Errorf("消失的 skill 应被删除，剩余 %d", count)
	}
}

// ---- rule catalog 测试 ----

// createTestRuleWithVisibility 创建一个 enterprise rule + 可见性分组绑定。
func createTestRuleWithVisibility(t *testing.T, db *gorm.DB, slug, version, ruleType string, groupID uint) *model.EnterpriseRule {
	t.Helper()
	rule := &model.EnterpriseRule{
		Slug:           slug,
		Version:        version,
		Name:           slug + "-name",
		Type:           ruleType,
		COSKey:         "cos/" + slug + ".md",
		VisibilityType: "group",
	}
	rule.ParseVersion()
	if err := db.Create(rule).Error; err != nil {
		t.Fatalf("创建 rule 失败: %v", err)
	}
	binding := &model.GroupConfigBinding{GroupID: groupID, ConfigType: model.AssetBindingTypeRule, ConfigKey: slug}
	if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(binding).Error; err != nil {
		t.Fatalf("创建 rule 资产绑定失败: %v", err)
	}
	return rule
}

func TestDiffAndQueue_NewRule(t *testing.T) {
	db := setupPhase2DB(t)

	user, group := createTestUserAndGroup(t, db)
	inst := createTestLocalInstance(t, db, user.ID)
	createTestRuleWithVisibility(t, db, "rule-a", "1.0.0", "prompt", group.ID)

	ctx := context.Background()
	var pendingCount int
	err := db.Transaction(func(tx *gorm.DB) error {
		c, e := diffAndQueue(ctx, tx, model.LocalSkillScopeUser, inst.ID, group.ID, "")
		pendingCount = c
		return e
	})
	if err != nil {
		t.Fatalf("diffAndQueue 失败: %v", err)
	}
	if pendingCount != 1 {
		t.Errorf("期望 1 个 pending（rule），实际 %d", pendingCount)
	}

	// 验证 local_instance_rules
	var lir model.LocalInstanceRule
	if err := db.Where("instance_id = ? AND slug = ?", inst.ID, "rule-a").First(&lir).Error; err != nil {
		t.Fatalf("local_instance_rules 未创建: %v", err)
	}
	if lir.InstallStatus != model.LocalSkillInstallStatusDistributing {
		t.Errorf("期望 install_status=distributing，实际 %s", lir.InstallStatus)
	}
	if lir.RuleType != "prompt" {
		t.Errorf("期望 rule_type=prompt，实际 %s", lir.RuleType)
	}

	// 验证 rule_distribution_records
	var rec model.RuleDistributionRecord
	if err := db.Where("instance_id = ? AND status = ?", inst.ID, model.RuleRecordStatusPending).First(&rec).Error; err != nil {
		t.Fatalf("pending rule record 未创建: %v", err)
	}

	// 验证 rule_distribution_task status=completed
	var task model.RuleDistributionTask
	if err := db.First(&task, rec.TaskID).Error; err != nil {
		t.Fatalf("rule distribution task 未创建: %v", err)
	}
	if task.Status != model.TaskStatusCompleted {
		t.Errorf("期望 task.status=completed，实际 %s", task.Status)
	}
}

func TestDiffAndQueue_RuleSameVersionSkip(t *testing.T) {
	db := setupPhase2DB(t)

	user, group := createTestUserAndGroup(t, db)
	inst := createTestLocalInstance(t, db, user.ID)
	createTestRuleWithVisibility(t, db, "rule-a", "1.0.0", "prompt", group.ID)

	ctx := context.Background()

	// 第一次 diffAndQueue → 写 pending
	err := db.Transaction(func(tx *gorm.DB) error {
		_, e := diffAndQueue(ctx, tx, model.LocalSkillScopeUser, inst.ID, group.ID, "")
		return e
	})
	if err != nil {
		t.Fatalf("第一次 diffAndQueue 失败: %v", err)
	}

	// 模拟 ack 成功：install_status → distributed
	db.Model(&model.LocalInstanceRule{}).Where("instance_id = ? AND slug = ?", inst.ID, "rule-a").
		Update("install_status", model.LocalSkillInstallStatusDistributed)

	// 第二次 diffAndQueue → 版本相同，应跳过
	var pendingCount int
	err = db.Transaction(func(tx *gorm.DB) error {
		c, e := diffAndQueue(ctx, tx, model.LocalSkillScopeUser, inst.ID, group.ID, "")
		pendingCount = c
		return e
	})
	if err != nil {
		t.Fatalf("第二次 diffAndQueue 失败: %v", err)
	}
	if pendingCount != 0 {
		t.Errorf("版本相同应跳过，期望 0 pending，实际 %d", pendingCount)
	}
}

func TestDiffAndQueue_RuleVersionUpgrade(t *testing.T) {
	db := setupPhase2DB(t)

	user, group := createTestUserAndGroup(t, db)
	inst := createTestLocalInstance(t, db, user.ID)

	// 先创建 v1.0.0 并分发 + ack 成功
	createTestRuleWithVisibility(t, db, "rule-a", "1.0.0", "prompt", group.ID)
	ctx := context.Background()
	db.Transaction(func(tx *gorm.DB) error {
		_, e := diffAndQueue(ctx, tx, model.LocalSkillScopeUser, inst.ID, group.ID, "")
		return e
	})
	db.Model(&model.LocalInstanceRule{}).Where("instance_id = ? AND slug = ?", inst.ID, "rule-a").
		Update("install_status", model.LocalSkillInstallStatusDistributed)

	// 升级到 v2.0.0（创建新版本 + 绑定可见性）
	createTestRuleWithVisibility(t, db, "rule-a", "2.0.0", "prompt", group.ID)

	var pendingCount int
	err := db.Transaction(func(tx *gorm.DB) error {
		c, e := diffAndQueue(ctx, tx, model.LocalSkillScopeUser, inst.ID, group.ID, "")
		pendingCount = c
		return e
	})
	if err != nil {
		t.Fatalf("diffAndQueue 失败: %v", err)
	}
	if pendingCount != 1 {
		t.Errorf("版本不同应写 pending，期望 1，实际 %d", pendingCount)
	}

	// 验证 local_instance_rules 版本更新为 2.0.0 且 install_status=distributing
	var lir model.LocalInstanceRule
	if err := db.Where("instance_id = ? AND slug = ?", inst.ID, "rule-a").First(&lir).Error; err != nil {
		t.Fatalf("local_instance_rules 未找到: %v", err)
	}
	if lir.Version != "2.0.0" {
		t.Errorf("期望 version=2.0.0，实际 %s", lir.Version)
	}
	if lir.InstallStatus != model.LocalSkillInstallStatusDistributing {
		t.Errorf("期望 install_status=distributing，实际 %s", lir.InstallStatus)
	}
}

func TestListRulesByGroupWithDB_GroupIDZero(t *testing.T) {
	db := setupPhase2DB(t)
	ctx := context.Background()

	items, err := model.ListRulesByGroupWithDB(db, ctx, 0)
	if err != nil {
		t.Fatalf("ListRulesByGroupWithDB 失败: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("groupID=0 应返回空，实际 %d 条", len(items))
	}
}

func TestListRulesByGroupWithDB_HappyPath(t *testing.T) {
	db := setupPhase2DB(t)
	ctx := context.Background()

	_, group := createTestUserAndGroup(t, db)
	createTestRuleWithVisibility(t, db, "rule-a", "1.0.0", "prompt", group.ID)
	createTestRuleWithVisibility(t, db, "rule-b", "2.0.0", "rule", group.ID)

	items, err := model.ListRulesByGroupWithDB(db, ctx, group.ID)
	if err != nil {
		t.Fatalf("ListRulesByGroupWithDB 失败: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("期望 2 条 rule，实际 %d", len(items))
	}
	slugs := map[string]bool{}
	for _, item := range items {
		slugs[item.Slug] = true
	}
	if !slugs["rule-a"] || !slugs["rule-b"] {
		t.Errorf("期望 rule-a 和 rule-b，实际 %v", slugs)
	}
}

func TestAssetCatalogUsesBindingsInsteadOfVisibility(t *testing.T) {
	db := setupPhase2DB(t)
	ctx := context.Background()
	_, group := createTestUserAndGroup(t, db)

	bound := createTestSkillWithVisibility(t, db, "bound-skill", "1.0.0", group.ID)
	visibleOnly := &model.Skill{Slug: "visible-only", Version: "1.0.0", Name: "visible-only", COSZipKey: "cos/visible-only.zip"}
	if err := db.Create(visibleOnly).Error; err != nil {
		t.Fatalf("创建仅可见 skill: %v", err)
	}
	if err := db.Create(&model.SkillVisibilityGroup{SkillID: visibleOnly.ID, GroupID: group.ID}).Error; err != nil {
		t.Fatalf("创建仅可见绑定: %v", err)
	}

	catalog, err := model.ListAssetCatalogByGroupWithDB(db, ctx, group.ID)
	if err != nil {
		t.Fatalf("查询分组资产目录: %v", err)
	}
	if len(catalog.Skills) != 1 || catalog.Skills[0].Slug != bound.Slug {
		t.Fatalf("目录应只包含资产绑定 skill，实际 %+v", catalog.Skills)
	}
}

func TestListAssetBindingTargetsAndLocalAgentTargets(t *testing.T) {
	db := setupPhase2DB(t)
	ctx := context.Background()
	user, group := createTestUserAndGroup(t, db)
	createTestSkillWithVisibility(t, db, "shared-skill", "1.0.0", group.ID)
	project := &model.Project{Name: "项目 A"}
	if err := db.Create(project).Error; err != nil {
		t.Fatalf("创建项目: %v", err)
	}
	if err := db.Create(&model.ProjectConfigBinding{ProjectID: project.ID, ConfigType: model.AssetBindingTypeSkill, ConfigKey: "shared-skill"}).Error; err != nil {
		t.Fatalf("创建项目资产绑定: %v", err)
	}

	targets, err := model.ListAssetBindingTargets(ctx, model.AssetTypeSkill, "shared-skill")
	if err != nil {
		t.Fatalf("反查资产绑定目标: %v", err)
	}
	if len(targets) != 2 || targets[0].TargetType != "group" || targets[1].TargetType != "project" {
		t.Fatalf("资产绑定目标不符合预期: %+v", targets)
	}

	inst := createTestLocalInstance(t, db, user.ID)
	cloud := &model.Instance{UserID: user.ID, InstanceId: "cvm-abc", Name: "cloud", Source: model.InstanceSourceCVM}
	if err := db.Create(cloud).Error; err != nil {
		t.Fatalf("创建云端实例: %v", err)
	}
	bindings := []model.LocalAgentScopeBinding{
		{InstanceID: inst.ID, Scope: model.LocalAgentScopeUser, GroupID: group.ID},
		{InstanceID: inst.ID, Scope: model.LocalAgentScopeWorkspace, ScopeKey: "/workspace/a", ProjectID: project.ID},
		{InstanceID: cloud.ID, Scope: model.LocalAgentScopeUser, GroupID: group.ID},
	}
	if err := db.Create(&bindings).Error; err != nil {
		t.Fatalf("创建本地 Agent scope 绑定: %v", err)
	}
	groupInstances, err := model.ListLocalAgentInstancesByScope(ctx, model.LocalAgentScopeUser, group.ID)
	if err != nil || len(groupInstances) != 1 || groupInstances[0].Instance.ID != inst.ID {
		t.Fatalf("查询分组本地 Agent 失败: items=%+v err=%v", groupInstances, err)
	}
	projectInstances, err := model.ListLocalAgentInstancesByScope(ctx, model.LocalAgentScopeWorkspace, project.ID)
	if err != nil || len(projectInstances) != 1 || projectInstances[0].ScopeBindings[0].Scope != model.LocalAgentScopeWorkspace {
		t.Fatalf("查询项目本地 Agent 失败: items=%+v err=%v", projectInstances, err)
	}
}

func TestListInstalledRulesWithDB_HappyPath(t *testing.T) {
	db := setupPhase2DB(t)

	_, _ = createTestUserAndGroup(t, db)
	inst := createTestLocalInstance(t, db, 1)

	now := time.Now()
	db.Create(&model.LocalInstanceRule{
		InstanceID: inst.ID, Slug: "rule-x", Version: "1.0.0", Scope: model.LocalSkillScopeUser,
		InstallStatus: model.LocalSkillInstallStatusDistributed, InstalledAt: &now, LastSeenAt: &now,
	})

	rows, err := model.ListInstalledRulesWithDB(db, model.LocalSkillScopeUser, inst.ID, "")
	if err != nil {
		t.Fatalf("ListInstalledRulesWithDB 失败: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("期望 1 条，实际 %d", len(rows))
	}
	if rows[0].Slug != "rule-x" {
		t.Errorf("期望 slug=rule-x，实际 %s", rows[0].Slug)
	}
}

// ---- buildLocalAgentResourcesView 测试 ----

func TestBuildLocalAgentResourcesView_Nil(t *testing.T) {
	setupPhase2DB(t)

	inst := &model.Instance{
		Source:              model.InstanceSourceLocal,
		LocalAgentResources: nil,
	}
	result := buildLocalAgentResourcesView(context.Background(), inst, 1)
	// nil → deserializeLocalAgentResources 返回空结构（非 nil），但 UserLevel.GroupID=0
	// buildLocalAgentResourcesView 对 GroupID=0 不补充 group_active
	// 返回值取决于实现，但不应 panic
	if result == nil {
		t.Fatal("不应返回 nil（deserializeLocalAgentResources 返回空结构）")
	}
}

func TestBuildLocalAgentResourcesView_WithGroup(t *testing.T) {
	db := setupPhase2DB(t)

	user, group := createTestUserAndGroup(t, db)

	// 构造已有序列化 resources
	resources := &model.LocalAgentResources{
		UserLevel: model.UserLevelResources{
			GroupID: group.ID,
		},
	}

	inst := &model.Instance{
		Source:              model.InstanceSourceLocal,
		LocalAgentResources: resources,
	}

	result := buildLocalAgentResourcesView(context.Background(), inst, user.ID)
	if result == nil {
		t.Fatal("不应返回 nil")
	}
	if !result.UserLevel.GroupActive {
		t.Errorf("用户在分组里，期望 group_active=true")
	}
	if result.UserLevel.GroupName != group.Name {
		t.Errorf("期望 group_name=%s，实际 %s", group.Name, result.UserLevel.GroupName)
	}
}

// TestBuildLocalAgentResourcesView_PendingSkillFromDistributionRecord 验证：
// 管控端下发 skill（只写 skill_distribution_records pending，不写 local_instance_skills）后，
// 视图能立即从 skill_distribution_records 补查到 distributing 状态（与 rule 的处理对称）。
func TestBuildLocalAgentResourcesView_PendingSkillFromDistributionRecord(t *testing.T) {
	db := setupPhase2DB(t)
	ctx := context.Background()

	user, group := createTestUserAndGroup(t, db)
	resources := &model.LocalAgentResources{UserLevel: model.UserLevelResources{GroupID: group.ID}}
	inst := &model.Instance{Source: model.InstanceSourceLocal, LocalAgentResources: resources}
	if err := db.Create(inst).Error; err != nil {
		t.Fatalf("create inst: %v", err)
	}

	// 已安装的技能（distributed），视图应正常展示
	db.Create(&model.LocalInstanceSkill{
		InstanceID: inst.ID, Slug: "installed-skill", Version: "1.0.0",
		Scope: model.LocalSkillScopeUser, InstallStatus: model.LocalSkillInstallStatusDistributed,
	})

	// 管控端下发的技能：只写 skill_distribution_records(pending)，不写 local_instance_skills
	skill := model.Skill{Slug: "pending-skill", Name: "待下发技能"}
	if err := db.Create(&skill).Error; err != nil {
		t.Fatalf("create skill: %v", err)
	}
	task := model.SkillDistributionTask{Slug: "pending-skill", Type: model.TaskTypeDistribute, Status: "running"}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}
	db.Create(&model.SkillDistributionRecord{
		TaskID: task.ID, SkillID: skill.ID, InstanceID: inst.ID,
		Version: "2.0.0", Status: model.RecordStatusPending, Type: model.TaskTypeDistribute,
	})

	result := buildLocalAgentResourcesView(ctx, inst, user.ID)
	if result == nil {
		t.Fatal("不应返回 nil")
	}

	// 收集视图里的 skill slug + status
	got := map[string]string{}
	for _, s := range result.UserLevel.Skills {
		got[s.Slug] = s.InstallStatus
	}
	if _, ok := got["installed-skill"]; !ok {
		t.Errorf("已安装技能应展示，实际 skills=%+v", result.UserLevel.Skills)
	}
	// 管控端下发的 pending 技能应通过 skill_distribution_records 补查到
	if st, ok := got["pending-skill"]; !ok {
		t.Errorf("管控端下发的 pending 技能应在视图中（从 skill_distribution_records 补查），实际 skills=%+v", result.UserLevel.Skills)
	} else if st != model.LocalSkillInstallStatusDistributing {
		t.Errorf("pending 技能状态应为 distributing，实际 %s", st)
	}
}

// TestBuildLocalAgentResourcesView_FailedSkillFromDistributionRecord 验证：
// 下发失败的技能（skill_distribution_records status=failed）在视图中展示为 failed。
func TestBuildLocalAgentResourcesView_FailedSkillFromDistributionRecord(t *testing.T) {
	db := setupPhase2DB(t)
	ctx := context.Background()

	user, group := createTestUserAndGroup(t, db)
	resources := &model.LocalAgentResources{UserLevel: model.UserLevelResources{GroupID: group.ID}}
	inst := &model.Instance{Source: model.InstanceSourceLocal, LocalAgentResources: resources}
	if err := db.Create(inst).Error; err != nil {
		t.Fatalf("create inst: %v", err)
	}

	skill := model.Skill{Slug: "failed-skill", Name: "失败技能"}
	db.Create(&skill)
	task := model.SkillDistributionTask{Slug: "failed-skill", Type: model.TaskTypeDistribute, Status: "running"}
	db.Create(&task)
	db.Create(&model.SkillDistributionRecord{
		TaskID: task.ID, SkillID: skill.ID, InstanceID: inst.ID,
		Version: "1.0.0", Status: model.RecordStatusFailed, Type: model.TaskTypeDistribute,
	})

	result := buildLocalAgentResourcesView(ctx, inst, user.ID)
	if result == nil {
		t.Fatal("不应返回 nil")
	}
	found := false
	for _, s := range result.UserLevel.Skills {
		if s.Slug == "failed-skill" {
			found = true
			if s.InstallStatus != model.LocalSkillInstallStatusFailed {
				t.Errorf("failed 技能状态应为 failed，实际 %s", s.InstallStatus)
			}
		}
	}
	if !found {
		t.Errorf("失败的技能应在视图中，实际 skills=%+v", result.UserLevel.Skills)
	}
}
