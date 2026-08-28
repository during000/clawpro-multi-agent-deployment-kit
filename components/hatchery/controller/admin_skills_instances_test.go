package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupSkillInstancesDB 初始化内存 SQLite 数据库，包含 HandleAdminSkillInstances 需要的全部表
func setupSkillInstancesDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if sqlDB, e := db.DB(); e == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.Instance{},
		&model.User{},
		&model.Skill{},
		&model.SkillDistributionRecord{},
		&model.SkillDistributionTask{},
		&model.SiteConfig{},
		&model.SMHSpace{},
		&model.UserGroup{},
		&model.UserGroupMember{},
		&model.FeatureAllowlist{},
		&model.LocalInstanceSkill{},
		&model.SkillVisibilityGroup{},
		// local_agent_scope_bindings：commands/sync 路径会查询工作区绑定
		&model.LocalAgentScopeBinding{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetConnMaxIdleTime(0)
	sqlDB.SetConnMaxLifetime(0)
	// 保证dbDriver是sqlite，防止distlock销毁连接
	origDB := model.UseDBForTestWithDriver(db, "sqlite")
	//origDB := model.UseDBForTest(db)
	// 启用 SMH；默认打开 LocalAgentEnabled，避免所有依赖 reporter 接口的测试被
	// 第 ② 层 SiteConfig 守卫拦下（该层仅是为保护 reporter 路径、不在大多数测试
	// 的主要验证点）。需要验证该层本身关闭路径的测试调 disableLocalAgentSiteConfig。
	db.Create(&model.SiteConfig{SMHEnabled: 1, LocalAgentEnabled: true})

	origToken := AdminToken
	AdminToken = "test-admin-token"

	// 启用 skillDistributeWG，让 HandleDistributeSkill 的后台 goroutine 在 cleanup 时可被等待，
	// 避免 -race -count=2 时后台 goroutine 与下一轮测试的 model.DB(context.Background()) 赋值产生数据竞态。
	var wg sync.WaitGroup
	skillDistributeWG = &wg

	t.Cleanup(func() {
		AdminToken = origToken
		wg.Wait()
		skillDistributeWG = nil
		origDB()
	})
	return db
}

// adminJSONGet 创建带 admin Bearer Token 和 Accept: application/json 的 GET 请求
func adminJSONGet(url string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req
}

// decodeJSON 解析响应 body 为 map
func decodeJSON(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("JSON 解析失败: %v, body=%s", err, w.Body.String())
	}
	return resp
}

// seedSkillAndInstances 创建测试用的技能和实例数据，返回 skill slug
func seedSkillAndInstances(t *testing.T) string {
	t.Helper()
	slug := "test-skill"

	// 创建技能
	skill := model.Skill{
		Slug: slug, Name: "测试技能", Version: "1.0.0",
		VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
		VisibilityType: "all",
	}
	model.DB(context.Background()).Create(&skill)

	// 创建用户
	user1 := model.User{Username: "alice"}
	model.DB(context.Background()).Create(&user1)
	user2 := model.User{Username: "bob"}
	model.DB(context.Background()).Create(&user2)
	user3 := model.User{Username: "charlie"}
	model.DB(context.Background()).Create(&user3)

	// 创建实例（带 instance_id，让 BuildSkillInstanceQuery 的 WHERE instance_id != '' 通过）
	instances := []model.Instance{
		{Name: "inst-alice-1", InstanceId: "ins-aaa111", UserID: user1.ID, LastCVMState: "RUNNING", AgentReady: 1},
		{Name: "inst-alice-2", InstanceId: "ins-aaa222", UserID: user1.ID, LastCVMState: "STOPPED", AgentReady: 0},
		{Name: "inst-bob-1", InstanceId: "ins-bbb111", UserID: user2.ID, LastCVMState: "RUNNING", AgentReady: 1},
		{Name: "inst-charlie-1", InstanceId: "ins-ccc111", UserID: user3.ID, LastCVMState: "RUNNING", AgentReady: 1},
	}
	for i := range instances {
		model.DB(context.Background()).Create(&instances[i])
	}

	// 创建下发记录（关联 skill）
	for _, inst := range instances {
		record := model.SkillDistributionRecord{
			TaskID: 1, SkillID: skill.ID, InstanceID: inst.ID,
			InstanceCID: inst.InstanceId, Version: "1.0.0", Status: "success",
		}
		model.DB(context.Background()).Create(&record)
	}

	return slug
}

func TestHandleAdminSkillInstances_MissingSlug(t *testing.T) {
	setupSkillInstancesDB(t)
	w := httptest.NewRecorder()
	HandleAdminSkillInstances(w, adminJSONGet("/admin/skills/instances"))
	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleAdminSkillInstances_NonExistentSlug(t *testing.T) {
	setupSkillInstancesDB(t)
	w := httptest.NewRecorder()
	HandleAdminSkillInstances(w, adminJSONGet("/admin/skills/instances?slug=no-such-skill"))
	if w.Code != http.StatusNotFound {
		t.Errorf("期望 404，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleAdminSkillInstances_BasicFlow(t *testing.T) {
	setupSkillInstancesDB(t)
	slug := seedSkillAndInstances(t)

	w := httptest.NewRecorder()
	HandleAdminSkillInstances(w, adminJSONGet("/admin/skills/instances?slug="+slug))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	resp := decodeJSON(t, w)
	// CVM 客户端创建失败时，batchFetchCVMInfoMap 返回 API_ERROR 标记，
	// ResolveInstanceStatus 使用缓存兜底返回 running，实例正常展示。
	instances := resp["instances"].([]interface{})
	total := int(resp["total"].(float64))
	if total != 4 {
		t.Errorf("期望 total=4（API_ERROR 缓存兜底），实际=%d", total)
	}
	if len(instances) != 4 {
		t.Errorf("期望 instances 有 4 条，实际有 %d 条", len(instances))
	}
	// 验证分页参数存在
	if _, ok := resp["page"]; !ok {
		t.Error("响应缺少 page 字段")
	}
	if _, ok := resp["page_size"]; !ok {
		t.Error("响应缺少 page_size 字段")
	}
}

func TestHandleAdminSkillInstances_SearchFilter(t *testing.T) {
	setupSkillInstancesDB(t)
	slug := seedSkillAndInstances(t)

	w := httptest.NewRecorder()
	HandleAdminSkillInstances(w, adminJSONGet("/admin/skills/instances?slug="+slug+"&search=alice"))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	resp := decodeJSON(t, w)
	// search 过滤在 SQL 层执行，结果仍为空（CVM 过滤），但代码路径已覆盖
	if resp["total"] == nil {
		t.Error("响应缺少 total 字段")
	}
}

func TestHandleAdminSkillInstances_StatusFilter(t *testing.T) {
	setupSkillInstancesDB(t)
	slug := seedSkillAndInstances(t)

	// 测试各种状态过滤
	for _, status := range []string{"installed", "uninstalled", "installing", "failed", "outdated", "installed,failed", "outdated,installed"} {
		w := httptest.NewRecorder()
		HandleAdminSkillInstances(w, adminJSONGet("/admin/skills/instances?slug="+slug+"&status="+status))
		if w.Code != http.StatusOK {
			t.Errorf("status=%s: 期望 200，实际=%d", status, w.Code)
		}
	}
}

func TestHandleAdminSkillInstances_GroupIDFilter_Ungrouped(t *testing.T) {
	setupSkillInstancesDB(t)
	slug := seedSkillAndInstances(t)

	// group_id=0 表示未分组用户
	w := httptest.NewRecorder()
	HandleAdminSkillInstances(w, adminJSONGet("/admin/skills/instances?slug="+slug+"&group_id=0"))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	decodeJSON(t, w) // 验证 JSON 合法
}

func TestHandleAdminSkillInstances_GroupIDFilter_NormalGroup(t *testing.T) {
	setupSkillInstancesDB(t)
	slug := seedSkillAndInstances(t)

	// 创建用户组并关联用户
	group := model.UserGroup{Name: "测试组"}
	model.DB(context.Background()).Create(&group)
	// 获取 alice 的 user_id
	var alice model.User
	model.DB(context.Background()).Where("username = ?", "alice").First(&alice)
	model.DB(context.Background()).Create(&model.UserGroupMember{UserGroupID: group.ID, UserID: alice.ID})

	w := httptest.NewRecorder()
	HandleAdminSkillInstances(w, adminJSONGet("/admin/skills/instances?slug="+slug+"&group_id="+uintStr(group.ID)))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	decodeJSON(t, w)
}

func TestHandleAdminSkillInstances_GroupIDFilter_UngroupedPlusNormal(t *testing.T) {
	setupSkillInstancesDB(t)
	slug := seedSkillAndInstances(t)

	// 创建用户组
	group := model.UserGroup{Name: "研发组"}
	model.DB(context.Background()).Create(&group)
	var bob model.User
	model.DB(context.Background()).Where("username = ?", "bob").First(&bob)
	model.DB(context.Background()).Create(&model.UserGroupMember{UserGroupID: group.ID, UserID: bob.ID})

	// group_id=0,<groupID> 表示未分组 + 指定分组（OR 语义）
	w := httptest.NewRecorder()
	HandleAdminSkillInstances(w, adminJSONGet("/admin/skills/instances?slug="+slug+"&group_id=0,"+uintStr(group.ID)))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	decodeJSON(t, w)
}

// TestHandleAdminSkillInstances_GroupIDFilter_MultiGroupNoDuplicate 验证用户属于多个分组时实例不重复
func TestHandleAdminSkillInstances_GroupIDFilter_MultiGroupNoDuplicate(t *testing.T) {
	setupSkillInstancesDB(t)
	slug := seedSkillAndInstances(t)

	// 创建两个分组
	groupA := model.UserGroup{Name: "分组A"}
	model.DB(context.Background()).Create(&groupA)
	groupB := model.UserGroup{Name: "分组B"}
	model.DB(context.Background()).Create(&groupB)

	// alice 同时属于两个分组
	var alice model.User
	model.DB(context.Background()).Where("username = ?", "alice").First(&alice)
	model.DB(context.Background()).Create(&model.UserGroupMember{UserGroupID: groupA.ID, UserID: alice.ID})
	model.DB(context.Background()).Create(&model.UserGroupMember{UserGroupID: groupB.ID, UserID: alice.ID})

	// 查询多分组
	w := httptest.NewRecorder()
	HandleAdminSkillInstances(w, adminJSONGet("/admin/skills/instances?slug="+slug+"&group_id="+uintStr(groupA.ID)+","+uintStr(groupB.ID)))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	resp := decodeJSON(t, w)

	// alice 有 2 个实例，每个只应出现一次
	total := int(resp["total"].(float64))
	if total != 2 {
		t.Errorf("用户属于多分组时不应重复，期望 total=2，实际=%d", total)
	}

	instances := resp["instances"].([]interface{})
	instanceIDs := make(map[float64]bool)
	for _, item := range instances {
		m := item.(map[string]interface{})
		id := m["instance_id"].(float64)
		if instanceIDs[id] {
			t.Errorf("发现重复实例 instance_id=%v", id)
		}
		instanceIDs[id] = true
	}
}

func TestHandleAdminSkillInstances_CombinedFilters(t *testing.T) {
	setupSkillInstancesDB(t)
	slug := seedSkillAndInstances(t)

	group := model.UserGroup{Name: "全栈组"}
	model.DB(context.Background()).Create(&group)

	// 同时使用 search + status + group_id 过滤
	w := httptest.NewRecorder()
	HandleAdminSkillInstances(w, adminJSONGet(
		"/admin/skills/instances?slug="+slug+
			"&search=inst&status=installed&group_id=0,"+uintStr(group.ID)+
			"&page=1&page_size=5"))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	resp := decodeJSON(t, w)
	if _, ok := resp["page_size"]; !ok {
		t.Error("响应缺少 page_size")
	}
}

func TestHandleAdminSkillInstances_InvalidGroupID(t *testing.T) {
	setupSkillInstancesDB(t)
	slug := seedSkillAndInstances(t)

	// group_id 含非法值 → 被忽略，不报错
	w := httptest.NewRecorder()
	HandleAdminSkillInstances(w, adminJSONGet("/admin/skills/instances?slug="+slug+"&group_id=abc,xyz"))
	if w.Code != http.StatusOK {
		t.Errorf("期望 200（非法 group_id 静默忽略），实际=%d", w.Code)
	}
}

func TestHandleAdminSkillInstances_OutdatedStatus(t *testing.T) {
	setupSkillInstancesDB(t)
	slug := "outdated-skill"

	// 创建两个版本的技能：v1.0.0 和 v2.0.0
	skillV1 := model.Skill{
		Slug: slug, Name: "待更新技能", Version: "1.0.0",
		VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
		VisibilityType: "all",
	}
	model.DB(context.Background()).Create(&skillV1)
	skillV2 := model.Skill{
		Slug: slug, Name: "待更新技能", Version: "2.0.0",
		VersionMajor: 2, VersionMinor: 0, VersionPatch: 0,
		VisibilityType: "all",
	}
	model.DB(context.Background()).Create(&skillV2)

	// 创建用户和实例
	user := model.User{Username: "outdated-user"}
	model.DB(context.Background()).Create(&user)
	inst := model.Instance{Name: "inst-outdated", InstanceId: "ins-outdated-001", UserID: user.ID, LastCVMState: "RUNNING", AgentReady: 1}
	model.DB(context.Background()).Create(&inst)

	// 下发记录为 v1.0.0（旧版本），状态 success
	record := model.SkillDistributionRecord{
		TaskID: 1, SkillID: skillV1.ID, InstanceID: inst.ID,
		InstanceCID: inst.InstanceId, Version: "1.0.0", Status: "success",
	}
	model.DB(context.Background()).Create(&record)

	// 查询 — 由于没有 CVM 客户端，实例会被过滤为非 running，
	// 但 SQL 层的 install_status 和 latest_version 逻辑可通过直接查询验证
	w := httptest.NewRecorder()
	HandleAdminSkillInstances(w, adminJSONGet("/admin/skills/instances?slug="+slug))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	// 验证 outdated 状态筛选不报错
	w = httptest.NewRecorder()
	HandleAdminSkillInstances(w, adminJSONGet("/admin/skills/instances?slug="+slug+"&status=outdated"))
	if w.Code != http.StatusOK {
		t.Fatalf("outdated 状态筛选: 期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	// 直接验证 SQL 层逻辑：通过 BuildSkillInstanceQuery 查询
	var skillIDs []uint
	model.DB(context.Background()).Model(&model.Skill{}).Where("slug = ?", slug).Pluck("id", &skillIDs)

	type queryResult struct {
		InstallStatus string `gorm:"column:install_status"`
		Version       string `gorm:"column:version"`
		LatestVersion string `gorm:"column:latest_version"`
	}
	var results []queryResult
	model.BuildSkillInstanceQuery(context.Background(), skillIDs, "2.0.0", slug).Scan(&results)

	if len(results) == 0 {
		t.Fatal("BuildSkillInstanceQuery 未返回结果")
	}
	r := results[0]
	if r.InstallStatus != "outdated" {
		t.Errorf("期望 install_status=outdated，实际=%s", r.InstallStatus)
	}
	if r.Version != "1.0.0" {
		t.Errorf("期望 version=1.0.0，实际=%s", r.Version)
	}
	if r.LatestVersion != "2.0.0" {
		t.Errorf("期望 latest_version=2.0.0，实际=%s", r.LatestVersion)
	}
}

func TestHandleAdminSkillInstances_Pagination(t *testing.T) {
	setupSkillInstancesDB(t)
	slug := seedSkillAndInstances(t)

	// 测试分页参数
	w := httptest.NewRecorder()
	HandleAdminSkillInstances(w, adminJSONGet("/admin/skills/instances?slug="+slug+"&page=2&page_size=1"))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}
	resp := decodeJSON(t, w)
	page := int(resp["page"].(float64))
	pageSize := int(resp["page_size"].(float64))
	if page != 2 {
		t.Errorf("期望 page=2，实际=%d", page)
	}
	if pageSize != 1 {
		t.Errorf("期望 page_size=1，实际=%d", pageSize)
	}
}

// uintStr 将 uint 转换为字符串
func uintStr(id uint) string {
	return fmt.Sprintf("%d", id)
}

// ==================== InstanceType 字段测试 ====================
// 覆盖 HandleAdminSkillInstances 中新增的 InstanceType 字段

// seedSkillAndInstancesWithAgentType 创建带有不同 agent_type 的测试实例
func seedSkillAndInstancesWithAgentType(t *testing.T) string {
	t.Helper()
	slug := "test-skill-agent-type"

	skill := model.Skill{
		Slug: slug, Name: "Agent类型测试技能", Version: "1.0.0",
		VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
		VisibilityType: "all",
	}
	model.DB(context.Background()).Create(&skill)

	user := model.User{Username: "agent_type_user"}
	model.DB(context.Background()).Create(&user)

	// 创建不同 agent_type 的实例
	instances := []model.Instance{
		{Name: "inst-openclaw", InstanceId: "ins-oc001", UserID: user.ID, LastCVMState: "RUNNING", AgentReady: 1, AgentType: "openclaw"},
		{Name: "inst-hermes", InstanceId: "ins-hm001", UserID: user.ID, LastCVMState: "RUNNING", AgentReady: 1, AgentType: "hermes"},
		{Name: "inst-lightclaw", InstanceId: "ins-lc001", UserID: user.ID, LastCVMState: "RUNNING", AgentReady: 1, AgentType: "lightclawace"},
		{Name: "inst-empty-type", InstanceId: "ins-et001", UserID: user.ID, LastCVMState: "RUNNING", AgentReady: 1, AgentType: ""},
	}
	for i := range instances {
		model.DB(context.Background()).Create(&instances[i])
	}

	// 创建下发记录
	for _, inst := range instances {
		record := model.SkillDistributionRecord{
			TaskID: 1, SkillID: skill.ID, InstanceID: inst.ID,
			InstanceCID: inst.InstanceId, Version: "1.0.0", Status: "success",
		}
		model.DB(context.Background()).Create(&record)
	}

	return slug
}

func TestHandleAdminSkillInstances_InstanceTypeFieldPresent(t *testing.T) {
	setupSkillInstancesDB(t)
	slug := seedSkillAndInstancesWithAgentType(t)

	w := httptest.NewRecorder()
	HandleAdminSkillInstances(w, adminJSONGet("/admin/skills/instances?slug="+slug))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	// 解析完整 JSON 响应
	var rawResp map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &rawResp); err != nil {
		t.Fatalf("JSON 解析失败: %v", err)
	}

	// 验证 instances 字段存在
	instancesRaw, ok := rawResp["instances"]
	if !ok {
		t.Fatal("响应缺少 instances 字段")
	}

	// 解析 instances 数组，每个元素应包含 instance_type 字段
	var instances []map[string]interface{}
	if err := json.Unmarshal(instancesRaw, &instances); err != nil {
		t.Fatalf("instances 字段解析失败: %v", err)
	}

	// 注意：由于没有 CVM 客户端，batchFetchCVMInfoMap 返回空 map，
	// ResolveInstanceStatus 会将实例标记为 destroyed 并过滤掉，
	// 所以这里主要验证 JSON 结构中 instance_type 字段的存在性。
	// 即使 instances 为空，也说明查询和序列化路径已覆盖。
	for i, inst := range instances {
		if _, hasField := inst["instance_type"]; !hasField {
			t.Errorf("instances[%d] 缺少 instance_type 字段", i)
		}
	}
}

func TestHandleAdminSkillInstances_InstanceTypeInSelectClause(t *testing.T) {
	setupSkillInstancesDB(t)
	slug := seedSkillAndInstancesWithAgentType(t)

	// 直接通过 BuildSkillInstanceQuery 验证 SQL 查询中 instance_type 字段的正确性
	var skillIDs []uint
	model.DB(context.Background()).Model(&model.Skill{}).Where("slug = ?", slug).Pluck("id", &skillIDs)
	if len(skillIDs) == 0 {
		t.Fatal("技能不存在")
	}

	type instRow struct {
		InstanceID   uint   `gorm:"column:instance_id"`
		InstanceName string `gorm:"column:instance_name"`
		InstanceType string `gorm:"column:instance_type"`
	}

	var rows []instRow
	q := model.BuildSkillInstanceQuery(context.Background(), skillIDs, "", slug)
	if err := q.Find(&rows).Error; err != nil {
		t.Fatalf("查询失败: %v", err)
	}

	if len(rows) != 4 {
		t.Fatalf("期望 4 条记录，实际=%d", len(rows))
	}

	// 验证 instance_type 值与创建时的 agent_type 一致
	typeMap := make(map[string]string) // instance_name → instance_type
	for _, row := range rows {
		typeMap[row.InstanceName] = row.InstanceType
	}

	expectedTypes := map[string]string{
		"inst-openclaw":   "openclaw",
		"inst-hermes":     "hermes",
		"inst-lightclaw":  "lightclawace",
		"inst-empty-type": "openclaw", // Instance model 的 AgentType 默认值为 'openclaw'
	}

	for name, expectedType := range expectedTypes {
		actual, ok := typeMap[name]
		if !ok {
			t.Errorf("未找到实例 %q", name)
			continue
		}
		if actual != expectedType {
			t.Errorf("实例 %q 的 instance_type = %q, 期望 %q", name, actual, expectedType)
		}
	}
}

// TestHandleAdminSkillInstances_LocalInstance 验证本地 agent 实例能出现在结果里：
//   - instances.source='local' 需要被 BuildSkillInstanceQuery SELECT 出来
//   - instResp.Source 要传给 ResolveInstanceStatus，走 Step -1 本地状态分支
//   - LocalInstanceInfo.last_report_at 新鲜 → StatusRunning → 不被第四步过滤
//
// 这是针对之前的 bug：tmpInst.Source 未传 → 走 CVM 状态机 → cvmInfo==nil
// → 状态被算为 load_failed/loading，不是 running → 被过滤掉。
func TestHandleAdminSkillInstances_LocalInstance(t *testing.T) {
	setupSkillInstancesDB(t)
	// LocalInstanceInfo 表不在 setupSkillInstancesDB 默认 migrate 列表，补一下
	if err := model.DB(context.Background()).AutoMigrate(&model.LocalInstanceInfo{}); err != nil {
		t.Fatalf("migrate LocalInstanceInfo: %v", err)
	}

	slug := "local-skill"
	skill := model.Skill{
		Slug: slug, Name: "本地 skill", Version: "1.0.0",
		VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
		VisibilityType: "all",
	}
	model.DB(context.Background()).Create(&skill)

	user := model.User{Username: "local-user"}
	model.DB(context.Background()).Create(&user)

	// 本地 agent 实例：instance_id 是 host CID，source=local，
	// agent_type=codebuddy（未注册到内置 agentTypesMap）。
	localInst := model.Instance{
		Name: "local-codebuddy", InstanceId: "local-codebuddy-001",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	model.DB(context.Background()).Create(&localInst)

	// 心跳新鲜→本地状态应该 running
	now := time.Now()
	model.DB(context.Background()).Create(&model.LocalInstanceInfo{
		InstanceID: localInst.ID, LastReportAt: &now, HostName: "codebuddy-host",
	})

	// 为这个本地实例写一条 status=failed 的分发记录，模拟 reporter ack
	model.DB(context.Background()).Create(&model.SkillDistributionRecord{
		TaskID: 1, SkillID: skill.ID, InstanceID: localInst.ID,
		InstanceCID: localInst.InstanceId, Version: "1.0.0",
		Status: "failed", Type: "distribute", Error: "mock failure",
	})

	// 查 status=failed 应能看到本地实例
	w := httptest.NewRecorder()
	HandleAdminSkillInstances(w, adminJSONGet("/admin/skills/instances?slug="+slug+"&status=failed"))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", w.Code, w.Body.String())
	}
	resp := decodeJSON(t, w)
	total := int(resp["total"].(float64))
	if total != 1 {
		t.Fatalf("本地实例应出现在 status=failed 过滤下，total 期望 1，实际=%d body=%s", total, w.Body.String())
	}
	instances := resp["instances"].([]interface{})
	if len(instances) != 1 {
		t.Fatalf("instances 应为 1 条，实际=%d", len(instances))
	}
	item := instances[0].(map[string]interface{})
	if name, _ := item["instance_name"].(string); name != "local-codebuddy" {
		t.Errorf("期望 instance_name=local-codebuddy，实际=%v", item["instance_name"])
	}
	if st, _ := item["status"].(string); st != "failed" {
		t.Errorf("期望 status=failed，实际=%v", item["status"])
	}
}

// TestHandleAdminSkillInstances_LocalInstance_Offline 本地实例心跳丢了应该被过滤掉。
func TestHandleAdminSkillInstances_LocalInstance_Offline(t *testing.T) {
	setupSkillInstancesDB(t)
	if err := model.DB(context.Background()).AutoMigrate(&model.LocalInstanceInfo{}); err != nil {
		t.Fatalf("migrate LocalInstanceInfo: %v", err)
	}
	slug := "local-skill-2"
	skill := model.Skill{
		Slug: slug, Name: "本地 skill 2", Version: "1.0.0",
		VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
		VisibilityType: "all",
	}
	model.DB(context.Background()).Create(&skill)
	user := model.User{Username: "local-user-2"}
	model.DB(context.Background()).Create(&user)
	localInst := model.Instance{
		Name: "local-stale", InstanceId: "local-stale-001",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: "openclaw",
	}
	model.DB(context.Background()).Create(&localInst)
	// last_report_at 超过阈值（并且 LocalInstanceOfflineThreshold = 24h）
	stale := time.Now().Add(-2 * model.LocalInstanceOfflineThreshold)
	model.DB(context.Background()).Create(&model.LocalInstanceInfo{
		InstanceID: localInst.ID, LastReportAt: &stale,
	})
	model.DB(context.Background()).Create(&model.SkillDistributionRecord{
		TaskID: 1, SkillID: skill.ID, InstanceID: localInst.ID,
		InstanceCID: localInst.InstanceId, Version: "1.0.0",
		Status: "failed", Type: "distribute",
	})

	w := httptest.NewRecorder()
	HandleAdminSkillInstances(w, adminJSONGet("/admin/skills/instances?slug="+slug+"&status=failed"))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}
	resp := decodeJSON(t, w)
	total := int(resp["total"].(float64))
	if total != 0 {
		t.Errorf("心跳丢失的本地实例应被过滤掉，total 期望 0，实际=%d body=%s", total, w.Body.String())
	}
}
