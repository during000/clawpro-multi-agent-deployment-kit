package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hatchery/model"

	"gorm.io/gorm"
)

// TestHandleLocalAgentReport_WithUserLevel
// report body 含 user_level（skills + rules；group_id 不作为切换输入）时，应走
// processReportUserLevelAndWorkspaces，把用户级资源写入 local_agent_resources。
func TestHandleLocalAgentReport_WithUserLevel(t *testing.T) {
	setupSkillInstancesDB(t)
	migrateLocalAgentTables(t)
	// report 成功路径会校验服务端主分组并可能触发 diffAndQueue，需 LocalInstanceRule 表。
	ctx := context.Background()
	if err := model.DB(ctx).AutoMigrate(&model.LocalInstanceRule{}); err != nil {
		t.Fatalf("migrate LocalInstanceRule: %v", err)
	}

	user := model.User{Username: "report-ul", Role: "user"}
	if err := model.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	// 预建 OneID 分组与资产绑定，供用户级下发对账使用。
	seedUserGroupCatalog(t, user.ID, []string{"translator", "weather"}, []string{"rule-a"})

	body := map[string]any{
		"agent_type":     "codebuddy",
		"local_agent_id": "cafef00dcafef00d",
		"skills": []map[string]any{
			{"slug": "translator", "version": "1.0.0"},
		},
		"user_level": map[string]any{
			"group_id": 7,
			"skills": []map[string]any{
				{"slug": "translator", "version": "1.0.0"},
				{"slug": "weather", "version": "2.0.0"},
			},
			"rules": []map[string]any{
				{"slug": "rule-a", "version": "1.0.0"},
			},
		},
	}
	rr := httptest.NewRecorder()
	HandleLocalAgentReport(rr, reportReq(t, "report-ul", body))
	if rr.Code != http.StatusOK {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	var inst model.Instance
	if err := model.DB(ctx).Where("user_id = ? AND source = ?", user.ID, model.InstanceSourceLocal).
		First(&inst).Error; err != nil {
		t.Fatalf("应创建 instance: %v", err)
	}
	res := deserializeLocalAgentResources(inst.LocalAgentResources)
	if res == nil {
		t.Fatalf("local_agent_resources 不应为空")
	}
	// group_id 由 ensureUserLevelGroup 按服务端用户关系统一管理，不采纳 reporter 上报值；
	// 本用例预建了 OneID 分组，group_id 应为该分组 ID（>0）。重点验证 user_level 的 skills/rules 对齐。
	if res.UserLevel.GroupID == 0 {
		t.Errorf("user_level.group_id 期望 >0（有 OneID 分组），实际=0")
	}
	// 管控端 Agent 列表读取 instances.group_id 回填组织路径；report 自动分配主组织时
	// 必须与 user_level 资源快照保持一致，不能只写 local_agent_resources。
	if inst.GroupID != res.UserLevel.GroupID {
		t.Errorf("instances.group_id=%d，应与用户级主组织=%d 一致", inst.GroupID, res.UserLevel.GroupID)
	}
	// user_level 的 skills/rules 通过 local_instance_skills/rules(scope='user')存储。
	var lis []model.LocalInstanceSkill
	if err := model.DB(ctx).Where("instance_id = ? AND scope = ?", inst.ID, model.LocalSkillScopeUser).
		Order("slug asc").Find(&lis).Error; err != nil {
		t.Fatalf("query user skills: %v", err)
	}
	if len(lis) != 2 {
		t.Errorf("用户级 local_instance_skills 期望 2 条，实际=%d", len(lis))
	}
	var lir []model.LocalInstanceRule
	if err := model.DB(ctx).Where("instance_id = ? AND scope = ?", inst.ID, model.LocalSkillScopeUser).
		Find(&lir).Error; err != nil {
		t.Fatalf("query user rules: %v", err)
	}
	if len(lir) != 1 {
		t.Errorf("用户级 local_instance_rules 期望 1 条，实际=%d", len(lir))
	}
}

// TestHandleLocalAgentReport_WithWorkspaces
// report body 含 workspaces[] 时，应走 processReportWorkspaces，按 workspace 写入
// 项目级（scope='workspace'）资源。
func TestHandleLocalAgentReport_WithWorkspaces(t *testing.T) {
	setupSkillInstancesDB(t)
	migrateLocalAgentTables(t)
	ctx := context.Background()
	if err := model.DB(ctx).AutoMigrate(&model.LocalInstanceRule{}); err != nil {
		t.Fatalf("migrate LocalInstanceRule: %v", err)
	}
	// 预建项目、成员与资产绑定，供 Workspace 下发对账使用。
	proj := &model.Project{Identifier: "proj1", Name: "proj1", Description: "", SyncMode: "continuous"}
	if err := model.DB(ctx).Create(proj).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}

	user := model.User{Username: "report-ws", Role: "user"}
	if err := model.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := model.DB(ctx).Create(&model.ProjectMember{
		Identifier: "", ProjectID: proj.ID, UserID: user.ID,
	}).Error; err != nil {
		t.Fatalf("create project member: %v", err)
	}
	if err := model.DB(ctx).Create(&model.Skill{Slug: "translator", Name: "translator", Version: "1.0.0", COSZipKey: "cos/translator.zip"}).Error; err != nil {
		t.Fatalf("create skill: %v", err)
	}
	if err := model.DB(ctx).Create(&model.ProjectConfigBinding{
		ProjectID: proj.ID, ConfigType: model.AssetBindingTypeSkill, ConfigKey: "translator", ValueJSON: "{}",
	}).Error; err != nil {
		t.Fatalf("create project skill binding: %v", err)
	}
	if err := model.DB(ctx).Create(&model.EnterpriseRule{Slug: "rule-p", Name: "rule-p", Version: "1.0.0", Type: "prompt", COSKey: "cos/rule-p.md"}).Error; err != nil {
		t.Fatalf("create rule: %v", err)
	}
	if err := model.DB(ctx).Create(&model.ProjectConfigBinding{
		ProjectID: proj.ID, ConfigType: model.AssetBindingTypeRule, ConfigKey: "rule-p", ValueJSON: "{}",
	}).Error; err != nil {
		t.Fatalf("create project rule binding: %v", err)
	}

	body := map[string]any{
		"agent_type":     "codebuddy",
		"local_agent_id": "feedfacefeedface",
		"skills":         []map[string]any{},
		"workspaces": []map[string]any{
			{
				"path":       "/home/alex/proj1",
				"name":       "proj1",
				"ide_type":   "vscode",
				"project_id": proj.ID,
				"skills": []map[string]any{
					{"slug": "translator", "version": "1.0.0"},
				},
				"rules": []map[string]any{
					{"slug": "rule-p", "version": "1.0.0"},
				},
			},
		},
	}
	rr := httptest.NewRecorder()
	HandleLocalAgentReport(rr, reportReq(t, "report-ws", body))
	if rr.Code != http.StatusOK {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	var inst model.Instance
	if err := model.DB(ctx).Where("user_id = ? AND source = ?", user.ID, model.InstanceSourceLocal).
		First(&inst).Error; err != nil {
		t.Fatalf("应创建 instance: %v", err)
	}
	res := deserializeLocalAgentResources(inst.LocalAgentResources)
	if res == nil {
		t.Fatalf("local_agent_resources 不应为空")
	}
	if len(res.Workspaces) != 1 {
		t.Fatalf("期望 1 个 workspace，实际=%d", len(res.Workspaces))
	}
	ws := res.Workspaces[0]
	if ws.Path != "/home/alex/proj1" {
		t.Errorf("workspace.path 期望 /home/alex/proj1，实际=%q", ws.Path)
	}
	if ws.ProjectID != proj.ID {
		t.Errorf("workspace.project_id 期望 %d，实际=%d", proj.ID, ws.ProjectID)
	}
	// workspace 的 skills/rules 通过 local_instance_skills(scope='workspace')存储，不进结构体。
	var lis []model.LocalInstanceSkill
	if err := model.DB(ctx).Where("instance_id = ? AND scope = ? AND workspace_path = ?",
		inst.ID, model.LocalSkillScopeWorkspace, "/home/alex/proj1").Find(&lis).Error; err != nil {
		t.Fatalf("query project skills: %v", err)
	}
	if len(lis) != 1 {
		t.Errorf("项目级 local_instance_skills 期望 1 条，实际=%d", len(lis))
	}
	var lir []model.LocalInstanceRule
	if err := model.DB(ctx).Where("instance_id = ? AND scope = ? AND workspace_path = ?",
		inst.ID, model.LocalSkillScopeWorkspace, "/home/alex/proj1").Find(&lir).Error; err != nil {
		t.Fatalf("query project rules: %v", err)
	}
	if len(lir) != 1 {
		t.Errorf("项目级 local_instance_rules 期望 1 条，实际=%d", len(lir))
	}
	var skillRecords, ruleRecords int64
	if err := model.DB(ctx).Model(&model.SkillDistributionRecord{}).Where("instance_id = ?", inst.ID).Count(&skillRecords).Error; err != nil {
		t.Fatalf("count skill records: %v", err)
	}
	if err := model.DB(ctx).Model(&model.RuleDistributionRecord{}).Where("instance_id = ?", inst.ID).Count(&ruleRecords).Error; err != nil {
		t.Fatalf("count rule records: %v", err)
	}
	if skillRecords != 0 || ruleRecords != 0 {
		t.Fatalf("report 只记录事实快照，不应直接入队，skills=%d rules=%d", skillRecords, ruleRecords)
	}
	if err := processSyncWorkspaces(ctx, &inst, &user, []syncWorkspace{{Path: "/home/alex/proj1", ProjectID: &proj.ID}}); err != nil {
		t.Fatalf("sync 项目目录对账失败: %v", err)
	}
	if err := model.DB(ctx).Model(&model.SkillDistributionRecord{}).Where("instance_id = ?", inst.ID).Count(&skillRecords).Error; err != nil {
		t.Fatalf("count sync skill records: %v", err)
	}
	if err := model.DB(ctx).Model(&model.RuleDistributionRecord{}).Where("instance_id = ?", inst.ID).Count(&ruleRecords).Error; err != nil {
		t.Fatalf("count sync rule records: %v", err)
	}
	if skillRecords != 0 || ruleRecords != 0 {
		t.Fatalf("本地已上报的项目资产不应重复下发，skills=%d rules=%d", skillRecords, ruleRecords)
	}
}

// TestHandleLocalAgentReport_UnboundAssetsAreTracked 复现真实 reporter 上报：
// 无论资源是否属于当前分组/项目资产，report 都是本地安装事实的唯一来源，必须按 scope 入库。
func TestHandleLocalAgentReport_UnboundAssetsAreTracked(t *testing.T) {
	setupSkillInstancesDB(t)
	migrateLocalAgentTables(t)
	ctx := context.Background()
	user := model.User{Username: "report-unbound-assets", Role: "user"}
	if err := model.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	body := map[string]any{
		"agent_type":     "codebuddy",
		"local_agent_id": "185f445ae8c49622",
		"host_name":      "MacBook-Pro.local",
		"os":             "darwin",
		"user_level": map[string]any{
			"skills": []map[string]any{
				{"slug": "dragon-doctor", "source": "enterprise"},
				{"slug": "fd-find", "source": "local"},
				{"slug": "find-skills", "source": "local"},
				{"slug": "self-improvement", "source": "local"},
				{"slug": "tencent-meeting-mcp", "source": "enterprise"},
				{"slug": "xinchao-kaigong", "source": "enterprise"},
			},
			"rules": []map[string]any{
				{"slug": "frontend-project-entry-1-3-0", "source": "enterprise"},
				{"slug": "frontend-react-rules-1-4-0", "source": "enterprise"},
				{"slug": "readme", "source": "enterprise"},
			},
		},
		"workspaces": []map[string]any{{
			"path": "/Users/alex/project/hatchery", "name": "hatchery", "ide_type": "codebuddy",
			"skills": []map[string]any{{"slug": "clarify", "source": "local"}},
			"rules": []map[string]any{
				{"slug": "code", "source": "local"},
				{"slug": "unittest", "source": "local"},
			},
		}},
	}
	rr := httptest.NewRecorder()
	HandleLocalAgentReport(rr, reportReq(t, user.Username, body))
	if rr.Code != http.StatusOK {
		t.Fatalf("report 应成功，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"user_level_synced":6`) ||
		!strings.Contains(rr.Body.String(), `"project_synced":1`) ||
		!strings.Contains(rr.Body.String(), `"rules_synced":5`) {
		t.Fatalf("所有上报资产应按 scope 入库，实际响应=%s", rr.Body.String())
	}

	var inst model.Instance
	if err := model.DB(ctx).Where("user_id = ? AND instance_id = ?", user.ID, "local-codebuddy-c49622").First(&inst).Error; err != nil {
		t.Fatalf("query instance: %v", err)
	}
	var skillCount, ruleCount int64
	if err := model.DB(ctx).Model(&model.LocalInstanceSkill{}).Where("instance_id = ?", inst.ID).Count(&skillCount).Error; err != nil {
		t.Fatalf("count local skills: %v", err)
	}
	if err := model.DB(ctx).Model(&model.LocalInstanceRule{}).Where("instance_id = ?", inst.ID).Count(&ruleCount).Error; err != nil {
		t.Fatalf("count local rules: %v", err)
	}
	if skillCount != 7 || ruleCount != 5 {
		t.Fatalf("未绑定资源也应按 report 入库，skills=%d rules=%d", skillCount, ruleCount)
	}
}

// TestHandleLocalAgentSync_WithWorkspaces
// sync body 含 workspaces[] 时，应走 processSyncWorkspaces，不因缺 workspace 处理崩溃。
func TestHandleLocalAgentSync_WithWorkspaces(t *testing.T) {
	setupSkillInstancesDB(t)
	migrateLocalAgentTables(t)
	ctx := context.Background()
	if err := model.DB(ctx).AutoMigrate(
		&model.LocalInstanceRule{},
		&model.RuleDistributionRecord{},
		&model.RuleDistributionTask{},
		&model.EnterpriseRule{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	user := model.User{Username: "sync-ws", Role: "user"}
	if err := model.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	inst := model.Instance{
		Name: "local-sync-ws", InstanceId: "local-sync-ws-001",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	if err := model.DB(ctx).Create(&inst).Error; err != nil {
		t.Fatalf("create inst: %v", err)
	}
	if err := model.DB(ctx).Create(&model.LocalInstanceInfo{
		InstanceID: inst.ID, LastReportAt: timePtr(),
	}).Error; err != nil {
		t.Fatalf("create info: %v", err)
	}

	body := map[string]any{
		"agent_type":     "codebuddy",
		"local_agent_id": "facefeedfacefac1",
		"workspaces": []map[string]any{
			{"path": "/p1", "name": "p1", "ide_type": "vscode"},
		},
	}
	rr := httptest.NewRecorder()
	HandleLocalAgentSync(rr, syncReq(t, "sync-ws", body))
	if rr.Code != http.StatusOK {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	// 命令列表应正常返回（含 skill/rule 命令，本用例未建 pending 记录，期望空命令但 ok=true）。
	if !strings.Contains(rr.Body.String(), "\"ok\"") {
		t.Errorf("响应应含 ok 字段，body=%s", rr.Body.String())
	}
}

func TestProcessSyncWorkspaces_SameProjectQueuesMissingAssets(t *testing.T) {
	setupSkillInstancesDB(t)
	migrateLocalAgentTables(t)
	ctx := context.Background()

	user := model.User{Username: "sync-same-project", Role: "user"}
	if err := model.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	project := model.Project{Name: "sync-same-project", SyncMode: "continuous"}
	if err := model.DB(ctx).Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}
	if err := model.DB(ctx).Create(&model.ProjectMember{ProjectID: project.ID, UserID: user.ID}).Error; err != nil {
		t.Fatalf("create project member: %v", err)
	}
	if err := model.DB(ctx).Create(&model.Skill{Slug: "same-project-skill", Name: "same-project-skill", Version: "1.0.0"}).Error; err != nil {
		t.Fatalf("create skill: %v", err)
	}
	if err := model.DB(ctx).Create(&model.ProjectConfigBinding{ProjectID: project.ID, ConfigType: model.AssetBindingTypeSkill, ConfigKey: "same-project-skill"}).Error; err != nil {
		t.Fatalf("create project binding: %v", err)
	}
	projectID := project.ID
	inst := model.Instance{
		Name: "sync-same-project", InstanceId: "sync-same-project-001", UserID: user.ID,
		Source: model.InstanceSourceLocal, LocalAgentResources: &model.LocalAgentResources{Workspaces: []model.WorkspaceResource{{Path: "/same", ProjectID: projectID}}},
	}
	if err := model.DB(ctx).Create(&inst).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}

	if err := processSyncWorkspaces(ctx, &inst, &user, []syncWorkspace{{Path: "/same", ProjectID: &projectID}}); err != nil {
		t.Fatalf("process sync workspaces: %v", err)
	}
	var records []model.SkillDistributionRecord
	if err := model.DB(ctx).Where("instance_id = ?", inst.ID).Find(&records).Error; err != nil {
		t.Fatalf("query distribution records: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("相同 project_id 的 sync 应补齐缺失资产，实际下发记录=%d", len(records))
	}
	var localSkill model.LocalInstanceSkill
	if err := model.DB(ctx).Where("instance_id = ? AND scope = ? AND workspace_path = ? AND slug = ?", inst.ID, model.LocalSkillScopeWorkspace, "/same", "same-project-skill").First(&localSkill).Error; err != nil {
		t.Fatalf("query queued local skill: %v", err)
	}
	if localSkill.Source != model.LocalSkillSourceEnterprise {
		t.Fatalf("项目资产下发必须显式标记 enterprise，实际=%q", localSkill.Source)
	}
}

func TestProcessReportWorkspaces_DeletedProjectKeepsSnapshot(t *testing.T) {
	setupSkillInstancesDB(t)
	migrateLocalAgentTables(t)
	ctx := context.Background()
	user := model.User{Username: "report-deleted-project", Role: "user"}
	if err := model.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	staleProjectID := uint(99999)
	resources := &model.LocalAgentResources{Workspaces: []model.WorkspaceResource{{Path: "/deleted", Name: "deleted", ProjectID: staleProjectID}}}
	inst := model.Instance{Name: "report-deleted-project", InstanceId: "report-deleted-project-001", UserID: user.ID, Source: model.InstanceSourceLocal, LocalAgentResources: resources}
	if err := model.DB(ctx).Create(&inst).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}
	if err := model.DB(ctx).Create(&model.LocalInstanceSkill{InstanceID: inst.ID, Scope: model.LocalSkillScopeWorkspace, WorkspacePath: "/deleted", Slug: "historical-skill", Version: "1.0.0", InstallStatus: model.LocalSkillInstallStatusDistributed}).Error; err != nil {
		t.Fatalf("create installed skill snapshot: %v", err)
	}

	err := model.DB(ctx).Transaction(func(tx *gorm.DB) error {
		_, _, err := processReportWorkspaces(ctx, tx, &inst, &user, resources, []reportWorkspace{{Path: "/deleted", Name: "deleted", ProjectID: &staleProjectID}}, time.Now())
		return err
	})
	if err != nil {
		t.Fatalf("失效项目的 report 不应报错: %v", err)
	}
	var snapshot model.LocalInstanceSkill
	if err := model.DB(ctx).Where("instance_id = ? AND scope = ? AND workspace_path = ? AND slug = ?", inst.ID, model.LocalSkillScopeWorkspace, "/deleted", "historical-skill").First(&snapshot).Error; err != nil {
		t.Fatalf("失效项目应保留已装资产快照: %v", err)
	}
	if len(resources.Workspaces) != 1 || resources.Workspaces[0].ProjectID != staleProjectID {
		t.Fatalf("失效项目应保留 workspace project_id，实际=%+v", resources.Workspaces)
	}
	view := buildLocalAgentResourcesView(ctx, &inst, user.ID)
	if view == nil || len(view.Workspaces) != 1 {
		t.Fatalf("应返回失效项目 workspace 视图，实际=%+v", view)
	}
	workspace := view.Workspaces[0]
	if workspace.ProjectExists || workspace.ProjectName != "" || workspace.ProjectID != staleProjectID {
		t.Fatalf("失效项目视图应标记 project_exists=false，实际=%+v", workspace)
	}
}

func TestLoadWorkspaceProjectSets_BatchResolution(t *testing.T) {
	db := setupPhase2DB(t)
	user := model.User{Username: "report-project-batch", Role: "user"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	projects := []model.Project{{Name: "batch-p1"}, {Name: "batch-p2"}, {Name: "batch-forbidden"}}
	if err := db.Create(&projects).Error; err != nil {
		t.Fatalf("create projects: %v", err)
	}
	if err := db.Create([]model.ProjectMember{
		{ProjectID: projects[0].ID, UserID: user.ID},
		{ProjectID: projects[1].ID, UserID: user.ID},
	}).Error; err != nil {
		t.Fatalf("create memberships: %v", err)
	}
	workspaces := []reportWorkspace{
		{Path: "/a", ProjectID: &projects[0].ID},
		{Path: "/b", ProjectID: &projects[1].ID},
		{Path: "/forbidden", ProjectID: &projects[2].ID},
	}
	members, existing, err := loadWorkspaceProjectSets(db, user.ID, map[string]uint{}, workspaces)
	if err != nil {
		t.Fatalf("load project sets: %v", err)
	}
	if len(members) != 2 || len(existing) != 3 {
		t.Fatalf("unexpected sets: members=%v existing=%v", members, existing)
	}
	if _, err := resolveWorkspaceProjectIDFromSet(0, workspaces[0].ProjectID, members); err != nil {
		t.Fatalf("member project should resolve: %v", err)
	}
	if _, err := resolveWorkspaceProjectIDFromSet(0, workspaces[2].ProjectID, members); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("non-member project should be rejected, got %v", err)
	}
}

// syncReq 构造 /local-agent/sync 请求（带登录态 + JSON body）。
func syncReq(t *testing.T, username string, body any) *http.Request {
	t.Helper()
	ensureSessionStore()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("syncReq encode: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/local-agent/sync", strings.NewReader(string(encoded)))
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

// timePtr 小工具。
func timePtr() *time.Time {
	now := time.Now()
	return &now
}
