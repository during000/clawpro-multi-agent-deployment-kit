package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hatchery/model"
)

// reportReq 构造 reporter 上报请求（带 session）。
func reportReq(t *testing.T, username string, body any) *http.Request {
	t.Helper()
	ensureSessionStore()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("encode body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/local-agent/report", strings.NewReader(string(encoded)))
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

// migrateLocalAgentTables 给 reporter 测试补齐 local_instance_infos 和
// local_instance_skills（setupSkillInstancesDB 默认不带）。
//
// 同时打开 SiteConfig.LocalAgentEnabled，避免 reporter 测试被第 ② 层 SiteConfig
// 守卫拦下（属改动未被测试覆盖的默认项，为保留原业务路径 happy-path
// 需要补上）。测 SiteConfig 本身关闭场景走 disableLocalAgentSiteConfig。
func migrateLocalAgentTables(t *testing.T) {
	t.Helper()
	if err := model.DB(context.Background()).AutoMigrate(
		&model.LocalInstanceInfo{},
		&model.LocalInstanceSkill{},
		&model.LocalAgentCLSCredential{},
		&model.LocalInstanceRule{},
		&model.EnterpriseRule{},
		&model.RuleDistributionRecord{},
		&model.RuleDistributionTask{},
		&model.RuleVisibilityGroup{},
		// local_agent_scope_bindings：report/sync 路径会查询用户级与工作区级绑定
		&model.LocalAgentScopeBinding{},
		&model.LocalAgentTask{},
		// catalog 查询依赖：group_closure + group_config_bindings
		&model.GroupClosure{},
		&model.GroupConfigBinding{},
		// workspace 路径查询依赖：projects + project_members + project_config_bindings
		&model.Project{},
		&model.ProjectMember{},
		&model.ProjectConfigBinding{},
	); err != nil {
		t.Fatalf("migrate local agent tables: %v", err)
	}
	if err := model.DBGlobal(context.Background()).Model(&model.SiteConfig{}).
		Where("1 = 1").Update("local_agent_enabled", true).Error; err != nil {
		t.Fatalf("enable LocalAgentEnabled: %v", err)
	}
}

// seedUserGroupCatalog 创建 OneID 分组 + 用户成员关系 + 分组绑定的 skill/rule 资产，
// 供 report 后的用户级资产下发对账使用。
// 返回创建的 groupID。
//
// skillSlugs / ruleSlugs 为要放入 catalog 的 slug 列表；
// 对应的 model.Skill / model.EnterpriseRule 行也会被创建（catalog 查询需要 join 这两张表）。
func seedUserGroupCatalog(t *testing.T, userID uint, skillSlugs, ruleSlugs []string) uint {
	t.Helper()
	ctx := context.Background()
	db := model.DB(ctx)

	g := &model.UserGroup{Name: "G-" + fmt.Sprintf("%d", userID), Source: model.GroupSourceOneIDDept, FullPath: "/G"}
	if err := db.Create(g).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}
	db.Create(&model.GroupClosure{AncestorID: g.ID, DescendantID: g.ID, Depth: 0})
	db.Create(&model.UserGroupMember{
		UserGroupID: g.ID, UserID: userID,
		Source: model.MemberSourceOneIDDept, IsMain: true,
	})

	for _, slug := range skillSlugs {
		db.Create(&model.Skill{Slug: slug, Name: slug, Version: "1.0.0", COSZipKey: "cos/" + slug + ".zip"})
		db.Create(&model.GroupConfigBinding{
			GroupID: g.ID, ConfigType: model.AssetBindingTypeSkill, ConfigKey: slug, ValueJSON: "{}",
		})
	}
	for _, slug := range ruleSlugs {
		db.Create(&model.EnterpriseRule{Slug: slug, Name: slug, Version: "1.0.0", Type: "prompt", COSKey: "cos/" + slug + ".md"})
		db.Create(&model.GroupConfigBinding{
			GroupID: g.ID, ConfigType: model.AssetBindingTypeRule, ConfigKey: slug, ValueJSON: "{}",
		})
	}
	return g.ID
}

// disableLocalAgentSiteConfig 关闭第 ② 层 SiteConfig.LocalAgentEnabled，仅用于测试那
// 一层守卫本身。默认 migrateLocalAgentTables 会打开。
func disableLocalAgentSiteConfig(t *testing.T) {
	t.Helper()
	if err := model.DBGlobal(context.Background()).Model(&model.SiteConfig{}).
		Where("1 = 1").Update("local_agent_enabled", false).Error; err != nil {
		t.Fatalf("disable LocalAgentEnabled: %v", err)
	}
}

// TestHandleLocalAgentReport_MethodNotAllowed POST 以外的方法应被拒绝。
func TestHandleLocalAgentReport_MethodNotAllowed(t *testing.T) {
	setupSkillInstancesDB(t)
	ensureSessionStore()

	req := httptest.NewRequest(http.MethodGet, "/local-agent/report", nil)
	rr := httptest.NewRecorder()
	HandleLocalAgentReport(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("应 405，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// TestHandleLocalAgentReport_Unauthorized 未登录返回 401。
func TestHandleLocalAgentReport_Unauthorized(t *testing.T) {
	setupSkillInstancesDB(t)
	ensureSessionStore()

	req := httptest.NewRequest(http.MethodPost, "/local-agent/report", strings.NewReader("{}"))
	rr := httptest.NewRecorder()
	HandleLocalAgentReport(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("应 401，实际=%d", rr.Code)
	}
}

// TestHandleLocalAgentReport_InvalidJSON body 不合法 JSON 返回 400。
func TestHandleLocalAgentReport_InvalidJSON(t *testing.T) {
	setupSkillInstancesDB(t)
	ensureSessionStore()
	user := model.User{Username: "report-bad-json", Role: "user"}
	if err := model.DB(context.Background()).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/local-agent/report", strings.NewReader("{not json"))
	req.Header.Set("Content-Type", "application/json")
	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = "report-bad-json"
	rr := httptest.NewRecorder()
	session.Save(req, rr)
	for _, cookie := range rr.Result().Cookies() {
		req.AddCookie(cookie)
	}
	rr2 := httptest.NewRecorder()
	HandleLocalAgentReport(rr2, req)
	if rr2.Code != http.StatusBadRequest {
		t.Errorf("非法 JSON 应 400，实际=%d body=%s", rr2.Code, rr2.Body.String())
	}
}

// TestHandleLocalAgentReport_BadAgentType 非白名单 agent_type 返回 400。
func TestHandleLocalAgentReport_BadAgentType(t *testing.T) {
	setupSkillInstancesDB(t)
	user := model.User{Username: "report-bad-type", Role: "user"}
	if err := model.DB(context.Background()).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	body := map[string]any{
		"agent_type":     "unknown-bot",
		"local_agent_id": "0123456789abcdef",
		"host_name":      "alex-mbp",
	}
	rr := httptest.NewRecorder()
	HandleLocalAgentReport(rr, reportReq(t, "report-bad-type", body))
	if rr.Code != http.StatusBadRequest {
		t.Errorf("非法 agent_type 应 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// TestHandleLocalAgentReport_BadLocalAgentID local_agent_id 不是 16 hex 返回 400。
func TestHandleLocalAgentReport_BadLocalAgentID(t *testing.T) {
	setupSkillInstancesDB(t)
	user := model.User{Username: "report-bad-id", Role: "user"}
	if err := model.DB(context.Background()).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	body := map[string]any{
		"agent_type":     "codebuddy",
		"local_agent_id": "tooshort",
		"host_name":      "alex-mbp",
	}
	rr := httptest.NewRecorder()
	HandleLocalAgentReport(rr, reportReq(t, "report-bad-id", body))
	if rr.Code != http.StatusBadRequest {
		t.Errorf("非法 local_agent_id 应 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// TestHandleLocalAgentReport_NewInstance_HappyPath 首次上报：
//   - upsert instances 行（source=local）
//   - upsert local_instance_infos
//   - per-skill upsert local_instance_skills（source=local 默认）
func TestHandleLocalAgentReport_NewInstance_HappyPath(t *testing.T) {
	setupSkillInstancesDB(t)
	migrateLocalAgentTables(t)
	ctx := context.Background()

	user := model.User{Username: "report-happy", Role: "user"}
	if err := model.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	// 预建 OneID 分组 + catalog 绑定，供用户级下发对账使用。
	seedUserGroupCatalog(t, user.ID, []string{"translator", "weather"}, nil)

	body := map[string]any{
		"agent_type":     "codebuddy",
		"agent_version":  "0.1.0",
		"local_agent_id": "deadbeefcafebabe",
		"host_name":      "alex-mbp",
		"os":             "darwin",
		"user_level": map[string]any{
			"skills": []map[string]any{
				{"slug": "translator", "version": "1.0.0", "display_name": "翻译器"},
				{"slug": "weather", "version": "2.1.0"},
			},
		},
	}
	rr := httptest.NewRecorder()
	HandleLocalAgentReport(rr, reportReq(t, "report-happy", body))
	if rr.Code != http.StatusOK {
		t.Fatalf("happy path 应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	// 1. instances 行被创建
	var inst model.Instance
	if err := model.DB(ctx).Where("user_id = ? AND source = ?", user.ID, model.InstanceSourceLocal).
		First(&inst).Error; err != nil {
		t.Fatalf("应创建 instance，实际无：%v", err)
	}
	if inst.AgentType != "codebuddy" {
		t.Errorf("agent_type 应=codebuddy，实际=%q", inst.AgentType)
	}
	if !strings.HasPrefix(inst.InstanceId, "local-codebuddy-") {
		t.Errorf("instance_id 应以 local-codebuddy- 开头，实际=%q", inst.InstanceId)
	}

	// 2. local_instance_infos 被创建
	var info model.LocalInstanceInfo
	if err := model.DB(ctx).Where("instance_id = ?", inst.ID).First(&info).Error; err != nil {
		t.Fatalf("应创建 LocalInstanceInfo，实际无：%v", err)
	}
	if info.HostName != "alex-mbp" {
		t.Errorf("host_name 应=alex-mbp，实际=%q", info.HostName)
	}
	if info.OS != "darwin" {
		t.Errorf("os 应=darwin，实际=%q", info.OS)
	}
	if info.LastReportAt == nil {
		t.Errorf("last_report_at 应非空")
	}

	// 3. local_instance_skills 两条
	var skills []model.LocalInstanceSkill
	if err := model.DB(ctx).Where("instance_id = ?", inst.ID).
		Order("slug asc").Find(&skills).Error; err != nil {
		t.Fatalf("query skills: %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("应有 2 条 skill，实际=%d", len(skills))
	}
	gotSlugs := []string{skills[0].Slug, skills[1].Slug}
	if gotSlugs[0] != "translator" || gotSlugs[1] != "weather" {
		t.Errorf("slug 应=translator/weather，实际=%v", gotSlugs)
	}
	for _, sk := range skills {
		if sk.Source != model.LocalSkillSourceLocal {
			t.Errorf("未传 source 时应默认 local，实际=%q (slug=%q)", sk.Source, sk.Slug)
		}
	}
}

// TestHandleLocalAgentReport_DisappearedSkillsRemoved
// reporter 上报是实例 skill 的全量真相：上报里没有的 slug 全部硬删，
// 不再区分 source（enterprise / public / local 均参与消失即删）。
func TestHandleLocalAgentReport_DisappearedSkillsRemoved(t *testing.T) {
	setupSkillInstancesDB(t)
	migrateLocalAgentTables(t)
	ctx := context.Background()

	user := model.User{Username: "report-disappear", Role: "user"}
	if err := model.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	// 预建 catalog：只含 skill-a（让 diffAndQueue 不会重新创建 skill-b）。
	seedUserGroupCatalog(t, user.ID, []string{"skill-a"}, nil)

	// 第一次上报：A；report 以本地事实为准，不受 catalog 过滤。
	body1 := map[string]any{
		"agent_type":     "codebuddy",
		"local_agent_id": "0123456789abcdef",
		"host_name":      "host1",
		"os":             "darwin",
		"user_level": map[string]any{
			"skills": []map[string]any{
				{"slug": "skill-a", "version": "1.0.0"},
			},
		},
	}
	rr := httptest.NewRecorder()
	HandleLocalAgentReport(rr, reportReq(t, "report-disappear", body1))
	if rr.Code != http.StatusOK {
		t.Fatalf("第一次上报应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	var inst model.Instance
	if err := model.DB(ctx).Where("user_id = ? AND source = ?", user.ID, model.InstanceSourceLocal).
		First(&inst).Error; err != nil {
		t.Fatalf("查 instance 失败: %v", err)
	}

	// 手动塞 skill-b（模拟之前装着但不在 catalog 中）和 skill-ent（模拟 enterprise 下发）。
	// 新语义下，下一次上报如果未包含它们，也会被一起硬删。
	now := time.Now()
	for _, slug := range []string{"skill-b", "skill-ent"} {
		ent := &model.LocalInstanceSkill{
			InstanceID: inst.ID, Slug: slug, Version: "1.0.0",
			Source: model.LocalSkillSourceEnterprise, InstalledAt: &now, LastSeenAt: &now,
		}
		if err := model.DB(ctx).Create(ent).Error; err != nil {
			t.Fatalf("create enterprise skill: %v", err)
		}
	}

	// 第二次上报：只有 A（B 、ent 都消失）
	body2 := map[string]any{
		"agent_type":     "codebuddy",
		"local_agent_id": "0123456789abcdef",
		"host_name":      "host1",
		"os":             "darwin",
		"user_level": map[string]any{
			"skills": []map[string]any{
				{"slug": "skill-a", "version": "1.0.0"},
			},
		},
	}
	rr2 := httptest.NewRecorder()
	HandleLocalAgentReport(rr2, reportReq(t, "report-disappear", body2))
	if rr2.Code != http.StatusOK {
		t.Fatalf("第二次上报应 200，实际=%d body=%s", rr2.Code, rr2.Body.String())
	}

	// 期望：只剩 skill-a（skill-b 和 enterprise skill-ent 均被硬删）
	var rows []model.LocalInstanceSkill
	if err := model.DB(ctx).Where("instance_id = ?", inst.ID).
		Order("slug asc").Find(&rows).Error; err != nil {
		t.Fatalf("query: %v", err)
	}
	got := map[string]string{}
	for _, r := range rows {
		got[r.Slug] = r.Source
	}
	if len(got) != 1 {
		t.Fatalf("期望只剩 1 条，实际 %d 条：%+v", len(got), got)
	}
	if got["skill-a"] != model.LocalSkillSourceLocal {
		t.Errorf("skill-a 未传 source 时应保留为 local，实际=%q", got["skill-a"])
	}
	if _, ok := got["skill-b"]; ok {
		t.Errorf("skill-b 应被硬删")
	}
	if _, ok := got["skill-ent"]; ok {
		t.Errorf("enterprise skill-ent 上报未包含应被硬删")
	}
}

// TestHandleLocalAgentReport_Idempotent 同 (user, instance_id) 第二次上报是 update：
// 不会创建新 instances 行，host_name / os 被更新。
func TestHandleLocalAgentReport_Idempotent(t *testing.T) {
	setupSkillInstancesDB(t)
	migrateLocalAgentTables(t)
	ctx := context.Background()

	user := model.User{Username: "report-idem", Role: "user"}
	if err := model.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	body := map[string]any{
		"agent_type":     "workbuddy",
		"local_agent_id": "abcdef0123456789",
		"host_name":      "old-host",
		"os":             "linux",
		"user_level":     map[string]any{"skills": []map[string]any{}},
	}
	rr := httptest.NewRecorder()
	HandleLocalAgentReport(rr, reportReq(t, "report-idem", body))
	if rr.Code != http.StatusOK {
		t.Fatalf("第一次应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	// 第二次：换 host_name + os
	body["host_name"] = "new-host"
	body["os"] = "darwin"
	rr2 := httptest.NewRecorder()
	HandleLocalAgentReport(rr2, reportReq(t, "report-idem", body))
	if rr2.Code != http.StatusOK {
		t.Fatalf("第二次应 200，实际=%d body=%s", rr2.Code, rr2.Body.String())
	}

	// 同 user 同 instance_id 只应有一行 instance
	var count int64
	if err := model.DB(ctx).Model(&model.Instance{}).
		Where("user_id = ? AND source = ?", user.ID, model.InstanceSourceLocal).
		Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Errorf("应只有 1 条 instance，实际=%d", count)
	}

	// host_name / os 应被更新
	var inst model.Instance
	if err := model.DB(ctx).Where("user_id = ? AND source = ?", user.ID, model.InstanceSourceLocal).
		First(&inst).Error; err != nil {
		t.Fatalf("first: %v", err)
	}
	var info model.LocalInstanceInfo
	if err := model.DB(ctx).Where("instance_id = ?", inst.ID).First(&info).Error; err != nil {
		t.Fatalf("info: %v", err)
	}
	if info.HostName != "new-host" {
		t.Errorf("host_name 应被更新为 new-host，实际=%q", info.HostName)
	}
	if info.OS != "darwin" {
		t.Errorf("os 应被更新为 darwin，实际=%q", info.OS)
	}

	// 卸载链路对称性回归：ack success 会软删 local_instance_infos（写 deleted_at）。
	// reporter 再 report（用户本地重新接入）应通过 upsert 的 DO UPDATE 把 deleted_at
	// 清零，与 instances 行的 Unscoped 复活对齐；否则 infos 仍是软删态，
	// 状态机取不到 LastReportAt 会把复活实例误判为 stopped。
	if err := model.DB(ctx).Where("instance_id = ?", inst.ID).Delete(&model.LocalInstanceInfo{}).Error; err != nil {
		t.Fatalf("软删 infos: %v", err)
	}
	var softInfo model.LocalInstanceInfo
	if err := model.DB(ctx).Unscoped().Where("instance_id = ?", inst.ID).First(&softInfo).Error; err != nil {
		t.Fatalf("unscoped 查软删 infos: %v", err)
	}
	if !softInfo.DeletedAt.Valid {
		t.Fatalf("软删后 deleted_at 应非空")
	}

	// 再 report 一次（模拟本地重新接入）
	rr3 := httptest.NewRecorder()
	HandleLocalAgentReport(rr3, reportReq(t, "report-idem", body))
	if rr3.Code != http.StatusOK {
		t.Fatalf("第三次 report 应 200，实际=%d body=%s", rr3.Code, rr3.Body.String())
	}
	var revived model.LocalInstanceInfo
	if err := model.DB(ctx).Where("instance_id = ?", inst.ID).First(&revived).Error; err != nil {
		t.Fatalf("复活后 infos 应默认可见（deleted_at 已清零），实际 err=%v", err)
	}
	if revived.DeletedAt.Valid {
		t.Errorf("reporter 再 report 后 infos deleted_at 应被清零，实际仍软删")
	}
	if revived.HostName != "new-host" {
		t.Errorf("复活后 host_name 应保持 new-host，实际=%q", revived.HostName)
	}
}

// TestHandleLocalAgentReport_DisappearReinstall_HardDelete 验证：
// local_instance_skills 不使用软删除，slug 消失后再次上报，能被正常 INSERT
// 而不是命中残留软删行更新失败。
//
// 历史背景：早期版本 LocalInstanceSkill 嵌了 gorm.Model（带 DeletedAt），
// UNIQUE (instance_id, slug) 不含 deleted_at；slug 软删后再上报会 ON DUPLICATE
// 命中那条软删行更新但 deleted_at 不被重置，导致前端「装了但查不到」。
// 现在改为 hard delete，本测试守住该回归。
func TestHandleLocalAgentReport_DisappearReinstall_HardDelete(t *testing.T) {
	setupSkillInstancesDB(t)
	migrateLocalAgentTables(t)
	ctx := context.Background()

	user := model.User{Username: "report-revive", Role: "user"}
	if err := model.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	inst := model.Instance{
		Name: "local-revive", InstanceId: "local-codebuddy-revive",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	if err := model.DB(ctx).Create(&inst).Error; err != nil {
		t.Fatalf("create inst: %v", err)
	}

	mkReported := func(slugs ...string) []reportSkillEntry {
		items := make([]reportSkillEntry, 0, len(slugs))
		for _, s := range slugs {
			items = append(items, reportSkillEntry{Slug: s, Version: "1.0.0"})
		}
		return items
	}

	// 直接调用 alignLocalSkills，绕过 catalog 过滤与 diffAndQueue，
	// 聚焦验证 hard delete 语义（消失即删是物理删除，不是软删）。

	// 第 1 次：装 skill-x
	if _, err := alignLocalSkills(model.DB(ctx), inst.ID, model.LocalSkillScopeUser, "", mkReported("skill-x"), time.Now()); err != nil {
		t.Fatalf("第 1 次 alignLocalSkills: %v", err)
	}

	// 第 2 次：skill-x 消失（被「消失即删」逻辑物理删除）
	if _, err := alignLocalSkills(model.DB(ctx), inst.ID, model.LocalSkillScopeUser, "", mkReported(), time.Now()); err != nil {
		t.Fatalf("第 2 次 alignLocalSkills: %v", err)
	}

	// 验证物理删除（不是软删）：表里查不到、Unscoped 也查不到。
	var alive int64
	model.DB(ctx).Model(&model.LocalInstanceSkill{}).
		Where("instance_id = ? AND slug = ?", inst.ID, "skill-x").Count(&alive)
	if alive != 0 {
		t.Fatalf("常规查询应查不到 skill-x，实际 count=%d", alive)
	}
	var anyRow int64
	model.DB(ctx).Unscoped().Model(&model.LocalInstanceSkill{}).
		Where("instance_id = ? AND slug = ?", inst.ID, "skill-x").Count(&anyRow)
	if anyRow != 0 {
		t.Fatalf("hard delete 后 Unscoped 也应查不到 skill-x，实际 count=%d（说明仍是软删）", anyRow)
	}

	// 第 3 次：skill-x 又装回来——必须能正常出现（hard delete 后 INSERT 不会被残留行卡住）
	if _, err := alignLocalSkills(model.DB(ctx), inst.ID, model.LocalSkillScopeUser, "", mkReported("skill-x"), time.Now()); err != nil {
		t.Fatalf("第 3 次 alignLocalSkills: %v", err)
	}

	var got model.LocalInstanceSkill
	if err := model.DB(ctx).Where("instance_id = ? AND slug = ?", inst.ID, "skill-x").
		First(&got).Error; err != nil {
		t.Fatalf("第 3 次后应查到 skill-x，实际错=%v", err)
	}
	if got.Source != model.LocalSkillSourceLocal {
		t.Errorf("source 应为 local，实际=%q", got.Source)
	}
}

// TestHandleLocalAgentReport_ConsistentRowSkipsUpdate
// 信息一致（version / display_name / source 全等）的行：updated_at 不刷新；
// 仅 last_seen_at 刷新（不进入"信息一致"判断）。
func TestHandleLocalAgentReport_ConsistentRowSkipsUpdate(t *testing.T) {
	setupSkillInstancesDB(t)
	migrateLocalAgentTables(t)
	ctx := context.Background()

	user := model.User{Username: "report-consistent", Role: "user"}
	if err := model.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	inst := model.Instance{
		Name: "local-consistent", InstanceId: "local-codebuddy-cons",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	if err := model.DB(ctx).Create(&inst).Error; err != nil {
		t.Fatalf("create inst: %v", err)
	}

	// 直接调用 alignLocalSkills，绕过 catalog 过滤与 diffAndQueue，
	// 聚焦验证信息一致时 updated_at 不刷新、last_seen_at 刷新。
	reported := []reportSkillEntry{{Slug: "skill-a", Version: "1.0.0", DisplayName: "A"}}

	if _, err := alignLocalSkills(model.DB(ctx), inst.ID, model.LocalSkillScopeUser, "", reported, time.Now()); err != nil {
		t.Fatalf("第一次 alignLocalSkills: %v", err)
	}

	var first model.LocalInstanceSkill
	if err := model.DB(ctx).Where("slug = ?", "skill-a").First(&first).Error; err != nil {
		t.Fatalf("first query: %v", err)
	}
	firstUpdated := first.UpdatedAt
	firstSeen := first.LastSeenAt

	// 等一小段，避免时间戳碰巧相等
	time.Sleep(20 * time.Millisecond)

	// 第二次完全一致的上报
	if _, err := alignLocalSkills(model.DB(ctx), inst.ID, model.LocalSkillScopeUser, "", reported, time.Now()); err != nil {
		t.Fatalf("第二次 alignLocalSkills: %v", err)
	}

	var second model.LocalInstanceSkill
	if err := model.DB(ctx).Where("slug = ?", "skill-a").First(&second).Error; err != nil {
		t.Fatalf("second query: %v", err)
	}
	if !second.UpdatedAt.Equal(firstUpdated) {
		t.Errorf("信息一致时 updated_at 不应变化，first=%v second=%v", firstUpdated, second.UpdatedAt)
	}
	if second.LastSeenAt == nil || firstSeen == nil || !second.LastSeenAt.After(*firstSeen) {
		t.Errorf("last_seen_at 应被刷新，first=%v second=%v", firstSeen, second.LastSeenAt)
	}
}

// TestHandleLocalAgentReport_SourceFollowsReporter
// 新语义下 source 跟随 reporter 上报：上报标 enterprise 就落 enterprise；
// 后续上报改成 local 就跟着改。
func TestHandleLocalAgentReport_SourceFollowsReporter(t *testing.T) {
	setupSkillInstancesDB(t)
	migrateLocalAgentTables(t)
	ctx := context.Background()

	user := model.User{Username: "report-source", Role: "user"}
	if err := model.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	inst := model.Instance{
		Name: "local-source", InstanceId: "local-codebuddy-src",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	if err := model.DB(ctx).Create(&inst).Error; err != nil {
		t.Fatalf("create inst: %v", err)
	}

	// 直接调用 alignLocalSkills，绕过 catalog 过滤与 diffAndQueue，
	// 聚焦验证 source 跟随 reporter 上报值。

	// 第一次上报：source=enterprise
	reported1 := []reportSkillEntry{{Slug: "skill-a", Version: "1.0.0", Source: "enterprise"}}
	if _, err := alignLocalSkills(model.DB(ctx), inst.ID, model.LocalSkillScopeUser, "", reported1, time.Now()); err != nil {
		t.Fatalf("第一次 alignLocalSkills: %v", err)
	}

	var row1 model.LocalInstanceSkill
	if err := model.DB(ctx).Where("slug = ?", "skill-a").First(&row1).Error; err != nil {
		t.Fatalf("query1: %v", err)
	}
	if row1.Source != model.LocalSkillSourceEnterprise {
		t.Errorf("第一次 source 应=enterprise，实际=%q", row1.Source)
	}

	// 第二次上报：source=local（信息变化）
	reported2 := []reportSkillEntry{{Slug: "skill-a", Version: "1.0.0", Source: "local"}}
	if _, err := alignLocalSkills(model.DB(ctx), inst.ID, model.LocalSkillScopeUser, "", reported2, time.Now()); err != nil {
		t.Fatalf("第二次 alignLocalSkills: %v", err)
	}

	var row2 model.LocalInstanceSkill
	if err := model.DB(ctx).Where("slug = ?", "skill-a").First(&row2).Error; err != nil {
		t.Fatalf("query2: %v", err)
	}
	if row2.Source != model.LocalSkillSourceLocal {
		t.Errorf("第二次 source 应=local，实际=%q", row2.Source)
	}
	if row1.InstalledAt == nil || row2.InstalledAt == nil || !row2.InstalledAt.Equal(*row1.InstalledAt) {
		t.Errorf("installed_at 应保留首次值，first=%v second=%v", row1.InstalledAt, row2.InstalledAt)
	}
}
