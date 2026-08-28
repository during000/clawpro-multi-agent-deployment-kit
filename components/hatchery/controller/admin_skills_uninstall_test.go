package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"hatchery/model"
)

func seedAdminUninstallSkill(t *testing.T, slug string) model.Skill {
	t.Helper()
	skill := model.Skill{
		Slug: slug, Name: "卸载测试技能", Version: "1.0.0",
		VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
		VisibilityType: model.VisibilityAll,
	}
	if err := model.DB(context.Background()).Create(&skill).Error; err != nil {
		t.Fatalf("创建技能失败: %v", err)
	}
	return skill
}

func seedAdminUninstallInstance(t *testing.T, username, name, cid, agentType string) model.Instance {
	t.Helper()
	user := model.User{Username: username, Role: "user"}
	if err := model.DB(context.Background()).Create(&user).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	inst := model.Instance{
		Name: name, InstanceId: cid, UserID: user.ID,
		LastCVMState: "RUNNING", AgentReady: 1,
		AgentType: agentType, RuntimeUser: "root",
	}
	if err := model.DB(context.Background()).Create(&inst).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}
	return inst
}

func seedSkillInstallRecord(t *testing.T, skillID uint, source, slug, version string, instance model.Instance, status string) {
	t.Helper()
	task := model.SkillDistributionTask{
		SkillID: skillID, Source: source, Slug: slug, Version: version,
		Total: 1, Status: model.TaskStatusCompleted, Type: model.TaskTypeDistribute,
	}
	if err := model.DB(context.Background()).Create(&task).Error; err != nil {
		t.Fatalf("创建历史技能任务失败: %v", err)
	}
	record := model.SkillDistributionRecord{
		TaskID: task.ID, SkillID: skillID, InstanceID: instance.ID, InstanceCID: instance.InstanceId,
		Version: version, Status: status, Type: model.TaskTypeDistribute,
	}
	if err := model.DB(context.Background()).Create(&record).Error; err != nil {
		t.Fatalf("创建历史技能记录失败: %v", err)
	}
}

func assertUninstallHTTPStatus(t *testing.T, body string, want int) map[string]interface{} {
	t.Helper()
	w := httptest.NewRecorder()
	HandleUninstallSkill(w, adminJSONPost("/admin/skills/uninstall", body))
	if w.Code != want {
		t.Fatalf("期望 HTTP %d，实际=%d, body=%s", want, w.Code, w.Body.String())
	}
	if w.Body.Len() == 0 {
		return nil
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("JSON 解析失败: %v, body=%s", err, w.Body.String())
	}
	return resp
}

func waitSkillTaskAsync(t *testing.T) {
	t.Helper()
	if skillDistributeWG != nil {
		skillDistributeWG.Wait()
	}
}

func installStatusForInstance(t *testing.T, skillIDs []uint, latestVersion string, instanceID uint) string {
	t.Helper()
	return installStatusForInstanceWithSlug(t, skillIDs, latestVersion, "", instanceID)
}

// installStatusForInstanceWithSlug 支持传入 slug，用于验证本地 agent 实例
// 的 lis JOIN 校验分支（source='local' + lis.id IS NULL → uninstalled）。
func installStatusForInstanceWithSlug(t *testing.T, skillIDs []uint, latestVersion, slug string, instanceID uint) string {
	t.Helper()
	type row struct {
		InstanceID uint   `gorm:"column:instance_id"`
		Status     string `gorm:"column:install_status"`
	}
	var rows []row
	if err := model.BuildSkillInstanceQuery(context.Background(), skillIDs, latestVersion, slug).
		Where("instances.id = ?", instanceID).
		Scan(&rows).Error; err != nil {
		t.Fatalf("查询安装状态失败: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("期望查询到 1 条实例状态，实际=%d", len(rows))
	}
	return rows[0].Status
}

func countInstancesByInstallStatus(t *testing.T, skillIDs []uint, latestVersion, status string) int64 {
	t.Helper()
	var rows []struct {
		InstanceID uint `gorm:"column:instance_id"`
	}
	if err := model.BuildSkillInstanceQuery(context.Background(), skillIDs, latestVersion, "").
		Scopes(model.FilterSkillInstallStatuses(latestVersion, []string{status})).
		Scan(&rows).Error; err != nil {
		t.Fatalf("按安装状态筛选失败: %v", err)
	}
	return int64(len(rows))
}

func seedSkillTaskForAdminTasks(t *testing.T, skill model.Skill, taskType string, inst model.Instance) model.SkillDistributionTask {
	t.Helper()
	task := model.SkillDistributionTask{
		SkillID: skill.ID, Version: skill.Version,
		Total: 1, Status: "completed", Type: taskType,
	}
	if err := model.DB(context.Background()).Create(&task).Error; err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}
	record := model.SkillDistributionRecord{
		TaskID: task.ID, SkillID: skill.ID, InstanceID: inst.ID,
		InstanceCID: inst.InstanceId, Version: skill.Version,
		Status: "success", Type: taskType,
	}
	if err := model.DB(context.Background()).Create(&record).Error; err != nil {
		t.Fatalf("创建任务记录失败: %v", err)
	}
	return task
}

func seedPublicSkillTaskForAdminTasks(t *testing.T, slug, version, sourceSkillsetSlug, batchID, taskType string, inst model.Instance, status string) model.SkillDistributionTask {
	t.Helper()
	task := model.SkillDistributionTask{
		Source: model.SkillSourcePublic, Slug: slug, Version: version,
		SourceSkillsetSlug: sourceSkillsetSlug, BatchID: batchID,
		Total: 1, Status: "completed", Type: taskType,
	}
	if err := model.DB(context.Background()).Create(&task).Error; err != nil {
		t.Fatalf("创建公共技能任务失败: %v", err)
	}
	record := model.SkillDistributionRecord{
		TaskID: task.ID, SkillID: 0, InstanceID: inst.ID,
		InstanceCID: inst.InstanceId, Version: version,
		Status: status, Type: taskType,
	}
	if err := model.DB(context.Background()).Create(&record).Error; err != nil {
		t.Fatalf("创建公共技能任务记录失败: %v", err)
	}
	return task
}

func decodeAdminTasksResponse(t *testing.T, path string) map[string]interface{} {
	t.Helper()
	w := httptest.NewRecorder()
	HandleAdminSkillTasks(w, adminJSONGet(path))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	return decodeJSON(t, w)
}

func TestHandleUninstallSkill_MissingSlug(t *testing.T) {
	setupSkillInstancesDB(t)
	assertUninstallHTTPStatus(t, `{"instance_ids":[1]}`, http.StatusBadRequest)
}

func TestHandleUninstallSkill_EmptyInstanceIDs(t *testing.T) {
	setupSkillInstancesDB(t)
	assertUninstallHTTPStatus(t, `{"slug":"test-skill","instance_ids":[]}`, http.StatusBadRequest)
}

func TestHandleUninstallSkill_NonExistentSlug(t *testing.T) {
	setupSkillInstancesDB(t)
	assertUninstallHTTPStatus(t, `{"slug":"no-such-skill","instance_ids":[1]}`, http.StatusNotFound)
}

func TestHandleUninstallSkill_AllUnsupportedTypes(t *testing.T) {
	setupSkillInstancesDB(t)
	skill := seedAdminUninstallSkill(t, "uninstall-unsupported")
	inst1 := seedAdminUninstallInstance(t, "unsupported-u1", "unsupported-1", "ins-unsup-001", "unknown_type_a")
	inst2 := seedAdminUninstallInstance(t, "unsupported-u2", "unsupported-2", "ins-unsup-002", "unknown_type_b")

	body := `{"slug":"` + skill.Slug + `","instance_ids":[` + uintStr(inst1.ID) + `,` + uintStr(inst2.ID) + `]}`
	resp := assertUninstallHTTPStatus(t, body, http.StatusBadRequest)
	if !strings.Contains(wantString(t, resp["error"]), "没有符合条件的实例") {
		t.Fatalf("错误信息应包含 没有符合条件的实例，实际=%v", resp)
	}

	var taskCount int64
	model.DB(context.Background()).Model(&model.SkillDistributionTask{}).Where("skill_id = ?", skill.ID).Count(&taskCount)
	if taskCount != 0 {
		t.Fatalf("不支持实例不应创建任务，实际 taskCount=%d", taskCount)
	}
}

func TestHandleUninstallSkill_BasicFlow(t *testing.T) {
	setupSkillInstancesDB(t)
	skill := seedAdminUninstallSkill(t, "uninstall-basic")
	inst := seedAdminUninstallInstance(t, "uninstall-basic-user", "uninstall-basic-inst", "ins-uninstall-basic", "openclaw")

	body := `{"slug":"` + skill.Slug + `","instance_ids":[` + uintStr(inst.ID) + `]}`
	resp := assertUninstallHTTPStatus(t, body, http.StatusOK)
	if resp["task_id"] == nil {
		t.Fatalf("响应缺少 task_id: %v", resp)
	}
	waitSkillTaskAsync(t)

	var task model.SkillDistributionTask
	if err := model.DB(context.Background()).Where("skill_id = ?", skill.ID).First(&task).Error; err != nil {
		t.Fatalf("未创建卸载任务: %v", err)
	}
	if task.Type != model.TaskTypeUninstall {
		t.Fatalf("Task.Type=%q，期望 %q", task.Type, model.TaskTypeUninstall)
	}
	if task.Status != "completed" {
		t.Fatalf("异步完成后 Task.Status=%q，期望 completed", task.Status)
	}
	if task.Success+task.Failed != task.Total {
		t.Fatalf("任务统计不一致: total=%d success=%d failed=%d", task.Total, task.Success, task.Failed)
	}

	var record model.SkillDistributionRecord
	if err := model.DB(context.Background()).Where("task_id = ? AND instance_id = ?", task.ID, inst.ID).First(&record).Error; err != nil {
		t.Fatalf("未创建卸载记录: %v", err)
	}
	if record.Type != model.TaskTypeUninstall {
		t.Fatalf("Record.Type=%q，期望 %q", record.Type, model.TaskTypeUninstall)
	}
	if record.Status == "pending" {
		t.Fatalf("异步完成后 Record.Status 不应仍为 pending")
	}
}

func TestNormalizeSkillUninstallStatuses(t *testing.T) {
	got, err := normalizeSkillUninstallStatuses(nil)
	if err != nil {
		t.Fatalf("normalizeSkillUninstallStatuses(nil) error=%v", err)
	}
	want := "installed,outdated,upgrade_failed,uninstall_failed,uninstall_failed_old"
	if strings.Join(got, ",") != want {
		t.Fatalf("默认卸载状态=%v，期望 %s", got, want)
	}
	for _, status := range []string{"uninstalled", "failed", "installing", "uninstalling"} {
		if _, err := normalizeSkillUninstallStatuses([]string{status}); err == nil {
			t.Errorf("normalizeSkillUninstallStatuses(%q) 未返回错误", status)
		}
	}
}

func TestHandleUninstallSkill_SelectAllEnterpriseByStatusAndGroup(t *testing.T) {
	setupSkillInstancesDB(t)
	db := model.DB(context.Background())
	skill := seedAdminUninstallSkill(t, "uninstall-select-all-enterprise")
	group := model.UserGroup{Name: "uninstall-select-all-group"}
	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("创建用户组失败: %v", err)
	}
	users := []model.User{
		{Username: "uninstall-select-all-grouped"},
		{Username: "uninstall-select-all-other"},
		{Username: "uninstall-select-all-failed"},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	for _, userID := range []uint{users[0].ID, users[2].ID} {
		if err := db.Create(&model.UserGroupMember{UserGroupID: group.ID, UserID: userID}).Error; err != nil {
			t.Fatalf("创建用户组成员失败: %v", err)
		}
	}
	instances := []model.Instance{
		{Name: "uninstall-select-all-match", InstanceId: "ins-uninstall-select-all-match", UserID: users[0].ID, AgentType: "openclaw"},
		{Name: "uninstall-select-all-other", InstanceId: "ins-uninstall-select-all-other", UserID: users[1].ID, AgentType: "openclaw"},
		{Name: "uninstall-select-all-failed", InstanceId: "ins-uninstall-select-all-failed", UserID: users[2].ID, AgentType: "openclaw"},
	}
	if err := db.Create(&instances).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}
	seedSkillInstallRecord(t, skill.ID, model.SkillSourceEnterprise, skill.Slug, skill.Version, instances[0], model.RecordStatusSuccess)
	seedSkillInstallRecord(t, skill.ID, model.SkillSourceEnterprise, skill.Slug, skill.Version, instances[1], model.RecordStatusSuccess)
	seedSkillInstallRecord(t, skill.ID, model.SkillSourceEnterprise, skill.Slug, skill.Version, instances[2], model.RecordStatusFailed)

	body := `{"source":"enterprise","slug":"` + skill.Slug + `","select_all":true,` +
		`"statuses":["installed"],"group_ids":[` + uintStr(group.ID) + `]}`
	w := httptest.NewRecorder()
	HandleUninstallSkill(w, adminJSONPost("/admin/skills/uninstall", body))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", w.Code, w.Body.String())
	}
	resp := decodeJSON(t, w)
	if got := int(resp["total"].(float64)); got != 1 {
		t.Fatalf("total=%d，期望 1", got)
	}
	taskID := uint(resp["task_id"].(float64))
	waitSkillTaskAsync(t)
	var records []model.SkillDistributionRecord
	if err := db.Where("task_id = ?", taskID).Find(&records).Error; err != nil {
		t.Fatalf("查询卸载记录失败: %v", err)
	}
	if len(records) != 1 || records[0].InstanceID != instances[0].ID || records[0].Type != model.TaskTypeUninstall {
		t.Fatalf("卸载记录=%+v，期望只包含实例 %d", records, instances[0].ID)
	}
}

func TestHandleUninstallSkill_SelectAllBatchPublic(t *testing.T) {
	setupSkillInstancesDB(t)
	db := model.DB(context.Background())
	instance := seedAdminUninstallInstance(t, "uninstall-select-all-public-user", "uninstall-select-all-public", "ins-uninstall-select-all-public", "openclaw")
	const slug = "uninstall-select-all-public"
	seedSkillInstallRecord(t, 0, model.SkillSourcePublic, slug, "1.0.0", instance, model.RecordStatusSuccess)

	body := `{"select_all":true,"statuses":["installed"],"skills":[{"source":"public","slug":"` + slug + `"}]}`
	w := httptest.NewRecorder()
	HandleUninstallSkill(w, adminJSONPost("/admin/skills/uninstall", body))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", w.Code, w.Body.String())
	}
	resp := decodeJSON(t, w)
	results := resp["results"].([]interface{})
	result := results[0].(map[string]interface{})
	if got := int(result["instance_count"].(float64)); got != 1 {
		t.Fatalf("instance_count=%d，期望 1", got)
	}
	taskID := uint(result["task_id"].(float64))
	waitSkillTaskAsync(t)
	var record model.SkillDistributionRecord
	if err := db.Where("task_id = ?", taskID).First(&record).Error; err != nil {
		t.Fatalf("查询公共技能卸载记录失败: %v", err)
	}
	if record.InstanceID != instance.ID || record.Type != model.TaskTypeUninstall {
		t.Fatalf("公共技能卸载记录=%+v，期望实例 %d", record, instance.ID)
	}
}

func TestHandleUninstallSkill_TaskTypeIsUninstall(t *testing.T) {
	setupSkillInstancesDB(t)
	skill := seedAdminUninstallSkill(t, "uninstall-type")
	inst := seedAdminUninstallInstance(t, "uninstall-type-user", "uninstall-type-inst", "ins-uninstall-type", "openclaw")

	body := `{"slug":"` + skill.Slug + `","instance_ids":[` + uintStr(inst.ID) + `]}`
	assertUninstallHTTPStatus(t, body, http.StatusOK)
	waitSkillTaskAsync(t)

	var task model.SkillDistributionTask
	if err := model.DB(context.Background()).Where("skill_id = ?", skill.ID).First(&task).Error; err != nil {
		t.Fatalf("未创建卸载任务: %v", err)
	}
	if task.Type != model.TaskTypeUninstall {
		t.Fatalf("Task.Type=%q，期望 %q", task.Type, model.TaskTypeUninstall)
	}

	var record model.SkillDistributionRecord
	if err := model.DB(context.Background()).Where("task_id = ?", task.ID).First(&record).Error; err != nil {
		t.Fatalf("未创建卸载记录: %v", err)
	}
	if record.Type != model.TaskTypeUninstall {
		t.Fatalf("Record.Type=%q，期望 %q", record.Type, model.TaskTypeUninstall)
	}
}

func TestHandleAdminSkillInstances_UninstallFailedStatus(t *testing.T) {
	setupSkillInstancesDB(t)
	skill := seedAdminUninstallSkill(t, "uninstall-failed-status")
	inst := seedAdminUninstallInstance(t, "uninstall-failed-user", "uninstall-failed-inst", "ins-uninstall-failed", "openclaw")
	task := seedSkillTaskForAdminTasks(t, skill, model.TaskTypeUninstall, inst)
	model.DB(context.Background()).Model(&model.SkillDistributionRecord{}).
		Where("task_id = ?", task.ID).
		Update("status", "failed")

	w := httptest.NewRecorder()
	HandleAdminSkillInstances(w, adminJSONGet("/admin/skills/instances?slug="+skill.Slug+"&status=uninstall_failed"))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	if got := countInstancesByInstallStatus(t, []uint{skill.ID}, skill.Version, "uninstall_failed"); got != 1 {
		t.Fatalf("status=uninstall_failed 应查到 1 条，实际=%d", got)
	}
}

func TestHandleAdminSkillInstances_UninstallingStatus(t *testing.T) {
	setupSkillInstancesDB(t)
	skill := seedAdminUninstallSkill(t, "uninstalling-status")
	inst := seedAdminUninstallInstance(t, "uninstalling-user", "uninstalling-inst", "ins-uninstalling", "openclaw")
	seedSkillTaskForAdminTasks(t, skill, model.TaskTypeUninstall, inst)
	model.DB(context.Background()).Model(&model.SkillDistributionRecord{}).
		Where("skill_id = ? AND instance_id = ?", skill.ID, inst.ID).
		Update("status", "pending")

	w := httptest.NewRecorder()
	HandleAdminSkillInstances(w, adminJSONGet("/admin/skills/instances?slug="+skill.Slug+"&status=uninstalling"))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	if got := countInstancesByInstallStatus(t, []uint{skill.ID}, skill.Version, "uninstalling"); got != 1 {
		t.Fatalf("status=uninstalling 应查到 1 条，实际=%d", got)
	}
}

func TestHandleAdminSkillInstances_AfterUninstall_ShowsUninstalled(t *testing.T) {
	setupSkillInstancesDB(t)
	skill := seedAdminUninstallSkill(t, "after-uninstall-status")
	inst := seedAdminUninstallInstance(t, "after-uninstall-user", "after-uninstall-inst", "ins-after-uninstall", "openclaw")
	seedSkillTaskForAdminTasks(t, skill, model.TaskTypeDistribute, inst)
	seedSkillTaskForAdminTasks(t, skill, model.TaskTypeUninstall, inst)

	if got := installStatusForInstance(t, []uint{skill.ID}, skill.Version, inst.ID); got != "uninstalled" {
		t.Fatalf("卸载成功后 install_status=%q，期望 uninstalled", got)
	}
}

func TestHandleAdminSkillInstances_ReinstallAfterUninstall(t *testing.T) {
	setupSkillInstancesDB(t)
	skill := seedAdminUninstallSkill(t, "reinstall-after-uninstall")
	inst := seedAdminUninstallInstance(t, "reinstall-user", "reinstall-inst", "ins-reinstall", "openclaw")
	seedSkillTaskForAdminTasks(t, skill, model.TaskTypeDistribute, inst)
	seedSkillTaskForAdminTasks(t, skill, model.TaskTypeUninstall, inst)
	seedSkillTaskForAdminTasks(t, skill, model.TaskTypeDistribute, inst)

	if got := installStatusForInstance(t, []uint{skill.ID}, skill.Version, inst.ID); got != "installed" {
		t.Fatalf("重新安装后 install_status=%q，期望 installed", got)
	}
}

func TestHandleAdminSkillTasks_TypeFieldInResponse(t *testing.T) {
	setupSkillInstancesDB(t)
	skill := seedAdminUninstallSkill(t, "task-type-field")
	inst := seedAdminUninstallInstance(t, "task-type-user", "task-type-inst", "ins-task-type", "openclaw")
	seedSkillTaskForAdminTasks(t, skill, model.TaskTypeUninstall, inst)

	resp := decodeAdminTasksResponse(t, "/admin/skills/tasks?slug="+skill.Slug)
	tasks := resp["tasks"].([]interface{})
	if len(tasks) != 1 {
		t.Fatalf("期望 1 个任务，实际=%d", len(tasks))
	}
	task := tasks[0].(map[string]interface{})
	if task["type"] != model.TaskTypeUninstall {
		t.Fatalf("响应 type=%v，期望 %q", task["type"], model.TaskTypeUninstall)
	}
}

func TestHandleAdminSkillTasks_TypeFilter_Uninstall(t *testing.T) {
	setupSkillInstancesDB(t)
	skill := seedAdminUninstallSkill(t, "task-filter-uninstall")
	inst := seedAdminUninstallInstance(t, "task-filter-uninstall-user", "task-filter-uninstall-inst", "ins-task-filter-uninstall", "openclaw")
	seedSkillTaskForAdminTasks(t, skill, model.TaskTypeDistribute, inst)
	seedSkillTaskForAdminTasks(t, skill, model.TaskTypeUninstall, inst)

	resp := decodeAdminTasksResponse(t, "/admin/skills/tasks?slug="+skill.Slug+"&type=uninstall")
	if int(resp["total"].(float64)) != 1 {
		t.Fatalf("type=uninstall total=%v，期望 1", resp["total"])
	}
	tasks := resp["tasks"].([]interface{})
	if tasks[0].(map[string]interface{})["type"] != model.TaskTypeUninstall {
		t.Fatalf("type=uninstall 返回了非卸载任务: %v", tasks[0])
	}
}

func TestHandleAdminSkillTasks_TypeFilter_Distribute(t *testing.T) {
	setupSkillInstancesDB(t)
	skill := seedAdminUninstallSkill(t, "task-filter-distribute")
	inst := seedAdminUninstallInstance(t, "task-filter-distribute-user", "task-filter-distribute-inst", "ins-task-filter-distribute", "openclaw")
	seedSkillTaskForAdminTasks(t, skill, model.TaskTypeDistribute, inst)
	seedSkillTaskForAdminTasks(t, skill, model.TaskTypeUninstall, inst)

	resp := decodeAdminTasksResponse(t, "/admin/skills/tasks?slug="+skill.Slug+"&type=distribute")
	if int(resp["total"].(float64)) != 1 {
		t.Fatalf("type=distribute total=%v，期望 1", resp["total"])
	}
	tasks := resp["tasks"].([]interface{})
	if tasks[0].(map[string]interface{})["type"] != model.TaskTypeDistribute {
		t.Fatalf("type=distribute 返回了非下发任务: %v", tasks[0])
	}
}

func TestHandleAdminSkillTasks_TypeFilter_All(t *testing.T) {
	setupSkillInstancesDB(t)
	skill := seedAdminUninstallSkill(t, "task-filter-all")
	inst := seedAdminUninstallInstance(t, "task-filter-all-user", "task-filter-all-inst", "ins-task-filter-all", "openclaw")
	seedSkillTaskForAdminTasks(t, skill, model.TaskTypeDistribute, inst)
	seedSkillTaskForAdminTasks(t, skill, model.TaskTypeUninstall, inst)

	respAll := decodeAdminTasksResponse(t, "/admin/skills/tasks?slug="+skill.Slug+"&type=all")
	if int(respAll["total"].(float64)) != 2 {
		t.Fatalf("type=all total=%v，期望 2", respAll["total"])
	}
	respDefault := decodeAdminTasksResponse(t, "/admin/skills/tasks?slug="+skill.Slug)
	if int(respDefault["total"].(float64)) != 2 {
		t.Fatalf("不传 type total=%v，期望 2", respDefault["total"])
	}
}

func TestHandleAdminSkillTasks_CountsUpgradeFailedAsFailed(t *testing.T) {
	setupSkillInstancesDB(t)
	skill := seedAdminUninstallSkill(t, "task-count-upgrade-failed")
	inst := seedAdminUninstallInstance(t, "task-count-user", "task-count-inst", "ins-task-count", "openclaw")
	task := seedSkillTaskForAdminTasks(t, skill, model.TaskTypeDistribute, inst)
	model.DB(context.Background()).Model(&model.SkillDistributionRecord{}).
		Where("task_id = ?", task.ID).
		Update("status", model.RecordStatusUpgradeFailed)

	resp := decodeAdminTasksResponse(t, "/admin/skills/tasks?slug="+skill.Slug)
	tasks := resp["tasks"].([]interface{})
	taskResp := tasks[0].(map[string]interface{})
	if int(taskResp["failed"].(float64)) != 1 {
		t.Fatalf("upgrade_failed 应计入 failed，实际 task=%v", taskResp)
	}
}

func TestHandleUninstallSkill_BatchPublicCreatesTasksWithoutLocalSkill(t *testing.T) {
	setupSkillInstancesDB(t)
	inst := seedAdminUninstallInstance(t, "public-uninstall-user", "public-uninstall-inst", "ins-public-uninstall", "openclaw")

	body := `{"instance_ids":[` + uintStr(inst.ID) + `],"skills":[` +
		`{"source":"public","slug":"public-uninstall-a","source_skillset_slug":"pkg-uninstall"},` +
		`{"source":"public","slug":"public-uninstall-b","source_skillset_slug":"pkg-uninstall"}]}`
	resp := assertUninstallHTTPStatus(t, body, http.StatusOK)
	batchID, ok := resp["batch_id"].(string)
	if !ok || batchID == "" {
		t.Fatalf("响应缺少 batch_id: %v", resp)
	}
	if int(resp["submitted"].(float64)) != 2 {
		t.Fatalf("submitted=%v, want 2", resp["submitted"])
	}
	waitSkillTaskAsync(t)

	var tasks []model.SkillDistributionTask
	if err := model.DB(context.Background()).Where("batch_id = ?", batchID).Order("id asc").Find(&tasks).Error; err != nil {
		t.Fatalf("查询公共卸载任务失败: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 public uninstall tasks, got %d", len(tasks))
	}
	for _, task := range tasks {
		if task.SkillID != 0 {
			t.Fatalf("public uninstall task skill_id=%d, want 0", task.SkillID)
		}
		if task.Source != model.SkillSourcePublic || task.SourceSkillsetSlug != "pkg-uninstall" || task.Type != model.TaskTypeUninstall {
			t.Fatalf("unexpected public uninstall task fields: %+v", task)
		}
	}
}

func TestHandleAdminSkillTasks_PublicSkillsetAggregatesByBatch(t *testing.T) {
	setupSkillInstancesDB(t)
	inst := seedAdminUninstallInstance(t, "public-task-user", "public-task-inst", "ins-public-task", "openclaw")
	seedPublicSkillTaskForAdminTasks(t, "public-task-a", "1.0.0", "pkg-task", "batch-public-task", model.TaskTypeDistribute, inst, "success")
	seedPublicSkillTaskForAdminTasks(t, "public-task-b", "1.0.0", "pkg-task", "batch-public-task", model.TaskTypeDistribute, inst, "success")

	resp := decodeAdminTasksResponse(t, "/admin/skills/tasks?source=public&source_skillset_slug=pkg-task&type=distribute")
	if int(resp["total"].(float64)) != 1 {
		t.Fatalf("public skillset task total=%v, want 1 batch", resp["total"])
	}
	tasks := resp["tasks"].([]interface{})
	task := tasks[0].(map[string]interface{})
	if task["batch_id"] != "batch-public-task" {
		t.Fatalf("batch_id=%v, want batch-public-task", task["batch_id"])
	}
	if len(task["task_ids"].([]interface{})) != 2 {
		t.Fatalf("task_ids=%v, want 2 ids", task["task_ids"])
	}
	unfilteredResp := decodeAdminTasksResponse(t, "/admin/skills/tasks?source=public&source_skillset_slug=pkg-task")
	unfilteredTask := unfilteredResp["tasks"].([]interface{})[0].(map[string]interface{})
	if unfilteredTask["type"] != model.TaskTypeDistribute {
		t.Fatalf("未传 type 过滤时 type=%v, want %q", unfilteredTask["type"], model.TaskTypeDistribute)
	}
	records := task["records"].([]interface{})
	if len(records) != 1 {
		t.Fatalf("records=%v, want one instance aggregate", records)
	}
	skillStatuses := records[0].(map[string]interface{})["skill_statuses"].([]interface{})
	if len(skillStatuses) != 2 {
		t.Fatalf("skill_statuses=%v, want 2 skill details", skillStatuses)
	}
}

func TestHandleAdminSkillTasks_PublicSkillsetLegacyTaskHasEmptyBatchID(t *testing.T) {
	setupSkillInstancesDB(t)
	inst := seedAdminUninstallInstance(t, "public-legacy-user", "public-legacy-inst", "ins-public-legacy", "openclaw")
	seedPublicSkillTaskForAdminTasks(t, "public-legacy-a", "1.0.0", "pkg-legacy", "", model.TaskTypeDistribute, inst, "success")

	resp := decodeAdminTasksResponse(t, "/admin/skills/tasks?source=public&source_skillset_slug=pkg-legacy")
	tasks := resp["tasks"].([]interface{})
	if len(tasks) != 1 {
		t.Fatalf("tasks=%v, want one legacy task aggregate", tasks)
	}
	task := tasks[0].(map[string]interface{})
	if task["batch_id"] != "" {
		t.Fatalf("legacy task batch_id=%v, want empty string", task["batch_id"])
	}
	if len(task["task_ids"].([]interface{})) != 1 {
		t.Fatalf("task_ids=%v, want one task id", task["task_ids"])
	}
}

func TestHandleAdminSkillInstances_PublicSkillsetAggregatesStatus(t *testing.T) {
	setupSkillInstancesDB(t)
	inst := seedAdminUninstallInstance(t, "public-inst-user", "public-inst", "ins-public-inst", "openclaw")
	seedPublicSkillTaskForAdminTasks(t, "public-inst-a", "1.0.0", "pkg-inst", "batch-public-inst", model.TaskTypeDistribute, inst, "success")
	seedPublicSkillTaskForAdminTasks(t, "public-inst-b", "1.0.0", "pkg-inst", "batch-public-inst", model.TaskTypeDistribute, inst, "success")

	body := `{"source":"public","source_skillset_slug":"pkg-inst","skills":[` +
		`{"source":"public","slug":"public-inst-a","version":"1.0.0"},` +
		`{"source":"public","slug":"public-inst-b","version":"1.0.0"}]}`
	w := httptest.NewRecorder()
	HandleAdminSkillInstances(w, adminJSONPost("/admin/skills/instances", body))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	resp := decodeJSON(t, w)
	if int(resp["total"].(float64)) != 1 {
		t.Fatalf("public skillset instances total=%v, want 1", resp["total"])
	}
	instances := resp["instances"].([]interface{})
	instance := instances[0].(map[string]interface{})
	if instance["status"] != "installed" {
		t.Fatalf("aggregate status=%v, want installed", instance["status"])
	}
	if len(instance["skill_statuses"].([]interface{})) != 2 {
		t.Fatalf("skill_statuses=%v, want 2", instance["skill_statuses"])
	}
}

func TestHandleAdminSkillInstances_PublicSkillsetValidationErrors(t *testing.T) {
	setupSkillInstancesDB(t)
	tests := []struct {
		name string
		body string
	}{
		{"unsupported source", `{"source":"enterprise","source_skillset_slug":"pkg","skills":[{"source":"public","slug":"a"}]}`},
		{"missing skillset", `{"source":"public","skills":[{"source":"public","slug":"a"}]}`},
		{"empty skills", `{"source":"public","source_skillset_slug":"pkg","skills":[]}`},
		{"mismatch skillset", `{"source":"public","source_skillset_slug":"pkg","skills":[{"source":"public","slug":"a","source_skillset_slug":"other"}]}`},
		{"invalid item source", `{"source":"public","source_skillset_slug":"pkg","skills":[{"source":"enterprise","slug":"a"}]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			HandleAdminSkillInstances(w, adminJSONPost("/admin/skills/instances", tt.body))
			if w.Code != http.StatusBadRequest {
				t.Fatalf("期望 400，实际=%d, body=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestHandleAdminSkillInstances_PublicSingleSkillStatus(t *testing.T) {
	setupSkillInstancesDB(t)
	inst := seedAdminUninstallInstance(t, "public-single-user", "public-single-inst", "ins-public-single", "openclaw")
	seedPublicSkillTaskForAdminTasks(t, "public-single", "1.0.0", "", "", model.TaskTypeDistribute, inst, "success")

	w := httptest.NewRecorder()
	HandleAdminSkillInstances(w, adminJSONGet("/admin/skills/instances?source=public&slug=public-single&version=1.0.0&status=installed"))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	resp := decodeJSON(t, w)
	if int(resp["total"].(float64)) != 1 {
		t.Fatalf("public single total=%v, want 1", resp["total"])
	}
	instances := resp["instances"].([]interface{})
	if instances[0].(map[string]interface{})["status"] != "installed" {
		t.Fatalf("status=%v, want installed", instances[0].(map[string]interface{})["status"])
	}
}

// TestHandleAdminSkillInstances_PublicSkillset_GroupIDFilter_MultiGroupNoDuplicate
// 验证公共技能集查询多分组筛选时实例不重复。
func TestHandleAdminSkillInstances_PublicSkillset_GroupIDFilter_MultiGroupNoDuplicate(t *testing.T) {
	setupSkillInstancesDB(t)

	// 创建用户 alice，同时属于 groupA 和 groupB
	user := model.User{Username: "alice"}
	model.DB(context.Background()).Create(&user)
	groupA := model.UserGroup{Name: "分组A"}
	model.DB(context.Background()).Create(&groupA)
	groupB := model.UserGroup{Name: "分组B"}
	model.DB(context.Background()).Create(&groupB)
	model.DB(context.Background()).Create(&model.UserGroupMember{UserGroupID: groupA.ID, UserID: user.ID})
	model.DB(context.Background()).Create(&model.UserGroupMember{UserGroupID: groupB.ID, UserID: user.ID})

	// alice 有 2 个实例
	inst1 := model.Instance{Name: "inst-alice-1", InstanceId: "ins-a1", UserID: user.ID, AgentType: "openclaw", LastCVMState: "RUNNING", AgentReady: 1}
	model.DB(context.Background()).Create(&inst1)
	inst2 := model.Instance{Name: "inst-alice-2", InstanceId: "ins-a2", UserID: user.ID, AgentType: "openclaw", LastCVMState: "RUNNING", AgentReady: 1}
	model.DB(context.Background()).Create(&inst2)

	// 为实例创建公共技能下发记录
	seedPublicSkillTaskForAdminTasks(t, "public-skill-a", "1.0.0", "test-pkg", "batch-test", model.TaskTypeDistribute, inst1, "success")
	seedPublicSkillTaskForAdminTasks(t, "public-skill-a", "1.0.0", "test-pkg", "batch-test", model.TaskTypeDistribute, inst2, "success")

	body := `{"source":"public","source_skillset_slug":"test-pkg","skills":[` +
		`{"source":"public","slug":"public-skill-a","version":"1.0.0"}]}`
	url := fmt.Sprintf("/admin/skills/instances?group_id=%d,%d", groupA.ID, groupB.ID)
	req := adminJSONPost(url, body)
	w := httptest.NewRecorder()
	HandleAdminSkillInstances(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	resp := decodeJSON(t, w)

	instances, _ := resp["instances"].([]interface{})
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

func TestHandleAdminSkillInstances_PublicVersionDoesNotReachSQLRaw(t *testing.T) {
	setupSkillInstancesDB(t)
	inst := seedAdminUninstallInstance(t, "public-version-user", "public-version-inst", "ins-public-version", "openclaw")
	seedPublicSkillTaskForAdminTasks(t, "public-version", "1.0.0", "", "", model.TaskTypeDistribute, inst, "success")

	payload := "1.0.0' OR '1'='1"
	target := "/admin/skills/instances?source=public&slug=public-version&status=installed&version=" + url.QueryEscape(payload)
	w := httptest.NewRecorder()
	HandleAdminSkillInstances(w, adminJSONGet(target))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	resp := decodeJSON(t, w)
	instances := resp["instances"].([]interface{})
	if len(instances) != 1 {
		t.Fatalf("instances=%v, want one public instance", instances)
	}
	instance := instances[0].(map[string]interface{})
	if instance["status"] != "installed" {
		t.Fatalf("status=%v, want installed", instance["status"])
	}
	if instance["latest_version"] != "" {
		t.Fatalf("latest_version=%v, want empty after invalid request version", instance["latest_version"])
	}
}

func TestPublicSkillAggregationHelpers(t *testing.T) {
	if got := aggregateRecordStatuses([]string{}); got != "pending" {
		t.Fatalf("aggregate empty=%q, want pending", got)
	}
	if got := aggregateRecordStatuses([]string{"success", "pending"}); got != "pending" {
		t.Fatalf("aggregate pending=%q, want pending", got)
	}
	if got := aggregateRecordStatuses([]string{"success", "failed"}); got != "failed" {
		t.Fatalf("aggregate failed=%q, want failed", got)
	}
	if got := aggregateRecordStatuses([]string{"success", "success"}); got != "success" {
		t.Fatalf("aggregate success=%q, want success", got)
	}
	if got := skillRecordInstallStatus(model.TaskTypeDistribute, "success", "1.0.0", "2.0.0"); got != "outdated" {
		t.Fatalf("distribute old status=%q, want outdated", got)
	}
	if got := skillRecordInstallStatus(model.TaskTypeDistribute, "pending", "1.0.0", "1.0.0"); got != "installing" {
		t.Fatalf("distribute pending status=%q, want installing", got)
	}
	if got := skillRecordInstallStatus(model.TaskTypeDistribute, "upgrade_failed", "1.0.0", "1.0.0"); got != "upgrade_failed" {
		t.Fatalf("distribute upgrade_failed status=%q, want upgrade_failed", got)
	}
	if got := skillRecordInstallStatus(model.TaskTypeUninstall, "success", "1.0.0", "1.0.0"); got != "uninstalled" {
		t.Fatalf("uninstall success status=%q, want uninstalled", got)
	}
	if got := skillRecordInstallStatus(model.TaskTypeUninstall, "pending", "1.0.0", "1.0.0"); got != "uninstalling" {
		t.Fatalf("uninstall pending status=%q, want uninstalling", got)
	}
	if got := skillRecordInstallStatus(model.TaskTypeUninstall, "failed", "1.0.0", "1.0.0"); got != "uninstall_failed" {
		t.Fatalf("uninstall failed status=%q, want uninstall_failed", got)
	}
	if got := aggregateSkillsetInstallStatus(nil); got != "uninstalled" {
		t.Fatalf("skillset empty=%q, want uninstalled", got)
	}
	if got := aggregateSkillsetInstallStatus([]string{"installed", "installing"}); got != "installing" {
		t.Fatalf("skillset installing=%q, want installing", got)
	}
	if got := aggregateSkillsetInstallStatus([]string{"installed", "outdated"}); got != "outdated" {
		t.Fatalf("skillset outdated=%q, want outdated", got)
	}
	if got := aggregateSkillsetInstallStatus([]string{"installed", "failed"}); got != "failed" {
		t.Fatalf("skillset failed=%q, want failed", got)
	}
	if got := aggregateSkillsetInstallStatus([]string{"installed", "uninstalled"}); got != "uninstalled" {
		t.Fatalf("skillset uninstalled=%q, want uninstalled", got)
	}
	if !statusFilterAllows("installed,failed", "failed") {
		t.Fatal("status filter should allow failed")
	}
	if statusFilterAllows("installed", "failed") {
		t.Fatal("status filter should reject failed")
	}
}

func wantString(t *testing.T, v interface{}) string {
	t.Helper()
	s, ok := v.(string)
	if !ok {
		t.Fatalf("期望 string，实际=%T(%v)", v, v)
	}
	return s
}

// TestHandleUninstallSkill_LocalInstance_KeepsPending /admin/skills/uninstall
// 对本地 agent 实例的处理与 distribute 对称：
//   - 不被 AgentTypeSupportsSkill 过滤掉（即便 agent_type=codebuddy 等本地类型）
//   - 创建 status=pending、type=uninstall 的 record
//   - 不交给 executeSkillTaskAsync 跑 RunScript
//   - record 保留 pending，等 reporter 来 ack 转 success/failed
func TestHandleUninstallSkill_LocalInstance_KeepsPending(t *testing.T) {
	setupSkillInstancesDB(t)
	testDB := model.DB(context.Background())

	skill := seedAdminUninstallSkill(t, "uninst-local")

	user := model.User{Username: "uninst-local-user"}
	testDB.Create(&user)
	localInst := model.Instance{
		Name: "local-codebuddy-uninst", InstanceId: "local-codebuddy-uninst-001",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	testDB.Create(&localInst)

	body := `{"slug":"uninst-local","instance_ids":[` + uintStr(localInst.ID) + `]}`
	resp := assertUninstallHTTPStatus(t, body, http.StatusOK)
	if resp == nil {
		t.Fatal("expected response body")
	}

	// 等任务+record 写入
	var task model.SkillDistributionTask
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if testDB.Where("skill_id = ? AND type = ?", skill.ID, model.TaskTypeUninstall).First(&task).Error == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if testDB.Where("skill_id = ? AND type = ?", skill.ID, model.TaskTypeUninstall).First(&task).Error != nil {
		t.Fatal("expected uninstall task to be created")
	}

	var rec model.SkillDistributionRecord
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if testDB.Where("task_id = ? AND instance_id = ?", task.ID, localInst.ID).First(&rec).Error == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if testDB.Where("task_id = ? AND instance_id = ?", task.ID, localInst.ID).First(&rec).Error != nil {
		t.Fatal("expected uninstall record to be created for local instance")
	}
	if rec.Type != model.TaskTypeUninstall {
		t.Errorf("record.Type 应为 uninstall，实际=%q", rec.Type)
	}

	// async finalize 跑完，本地 record 仍应是 pending
	waitSkillTaskAsync(t)
	testDB.Where("task_id = ? AND instance_id = ?", task.ID, localInst.ID).First(&rec)
	if rec.Status != "pending" {
		t.Errorf("本地实例 uninstall record 应保持 pending（等 reporter ack），实际=%q", rec.Status)
	}
}

// TestBuildSkillInstanceQuery_LocalInstance_UninstalledLocally 验证本次修复：
// 本地 agent 实例下发 skill 成功后，如果用户在本地把 skill 卸掉（reporter
// 上报中不再包含该 skill），hatchery 将对应 local_instance_skills 行硬删。
// 此时 install_status 应归为 uninstalled（而非仍看 lr.status='success' 担 installed）。
func TestBuildSkillInstanceQuery_LocalInstance_UninstalledLocally(t *testing.T) {
	setupSkillInstancesDB(t)
	testDB := model.DB(context.Background())

	skill := seedAdminUninstallSkill(t, "local-uninst")
	user := model.User{Username: "local-uninst-user"}
	testDB.Create(&user)

	localInst := model.Instance{
		Name: "local-codebuddy-1", InstanceId: "local-codebuddy-uninst-001",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	testDB.Create(&localInst)

	// 创建一条 distribute+success 的 record（表示当初下发成功）
	task := model.SkillDistributionTask{
		SkillID: skill.ID, Version: skill.Version, Total: 1, Status: "completed",
		Type: model.TaskTypeDistribute,
	}
	testDB.Create(&task)
	record := model.SkillDistributionRecord{
		TaskID: task.ID, SkillID: skill.ID, InstanceID: localInst.ID,
		InstanceCID: localInst.InstanceId, Version: skill.Version,
		Status: "success", Type: model.TaskTypeDistribute,
	}
	testDB.Create(&record)

	skillIDs := []uint{skill.ID}

	// 场景 1：local_instance_skills 中有对应行→ installed（本地确实装着）
	now := time.Now()
	lis := model.LocalInstanceSkill{
		InstanceID: localInst.ID, Slug: skill.Slug, Version: skill.Version,
		Source: model.LocalSkillSourceEnterprise, InstalledAt: &now, LastSeenAt: &now,
	}
	testDB.Create(&lis)

	if got := installStatusForInstanceWithSlug(t, skillIDs, skill.Version, skill.Slug, localInst.ID); got != "installed" {
		t.Fatalf("lis 存在时：期望 installed，实际=%q", got)
	}

	// 场景 2：user 本地卸掉，reporter 上报不再包含→ lis 行被硬删 → uninstalled
	if err := testDB.Where("instance_id = ? AND slug = ?", localInst.ID, skill.Slug).
		Delete(&model.LocalInstanceSkill{}).Error; err != nil {
		t.Fatalf("删 lis 行失败: %v", err)
	}
	if got := installStatusForInstanceWithSlug(t, skillIDs, skill.Version, skill.Slug, localInst.ID); got != "uninstalled" {
		t.Fatalf("lis 不存在时：期望 uninstalled（本地已卸），实际=%q", got)
	}
}

// TestBuildSkillInstanceQuery_LocalInstance_OutdatedNotMisjudgedAsUninstalled
// 本地 agent 实例旧版本 success（version=1.0.0，latest=2.0.0）+ lis 缺失
// 不应误判为 uninstalled，而应判 outdated（待更新），与 ack 路径
// skillRecordInstallStatus 的语义保持一致：record=success 且版本旧即 outdated。
// 只有当最新版（version==latest）success 且 lis 缺失时，才回退为 uninstalled。
func TestBuildSkillInstanceQuery_LocalInstance_OutdatedNotMisjudgedAsUninstalled(t *testing.T) {
	setupSkillInstancesDB(t)
	testDB := model.DB(context.Background())

	skillOld := seedAdminUninstallSkill(t, "local-outd-old")
	// 同 slug 创建一个新版本，构造 outdated 场景（latestVersion=2.0.0，record.version=1.0.0）
	skillNew := model.Skill{
		Slug: skillOld.Slug, Name: skillOld.Name, Version: "2.0.0",
		VersionMajor: 2, VersionMinor: 0, VersionPatch: 0, VisibilityType: "all",
	}
	testDB.Create(&skillNew)

	user := model.User{Username: "local-outd-user"}
	testDB.Create(&user)
	localInst := model.Instance{
		Name: "local-codebuddy-outd", InstanceId: "local-codebuddy-outd-001",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	testDB.Create(&localInst)

	task := model.SkillDistributionTask{
		SkillID: skillOld.ID, Version: skillOld.Version, Total: 1, Status: "completed",
		Type: model.TaskTypeDistribute,
	}
	testDB.Create(&task)
	record := model.SkillDistributionRecord{
		TaskID: task.ID, SkillID: skillOld.ID, InstanceID: localInst.ID,
		InstanceCID: localInst.InstanceId, Version: "1.0.0",
		Status: "success", Type: model.TaskTypeDistribute,
	}
	testDB.Create(&record)

	skillIDs := []uint{skillOld.ID, skillNew.ID}

	// 未插 lis：旧版本 success + lis 缺失 → 应为 outdated（待更新），
	// 不是 uninstalled。更新版本后，已装旧版的实例不能再被误判为未下发。
	if got := installStatusForInstanceWithSlug(t, skillIDs, "2.0.0", skillOld.Slug, localInst.ID); got != "outdated" {
		t.Fatalf("旧版本 + lis 缺失：期望 outdated（待更新），实际=%q", got)
	}

	// 插 lis（旧版本）→ 应为 outdated（本地确实装着旧版）
	now := time.Now()
	lis := model.LocalInstanceSkill{
		InstanceID: localInst.ID, Slug: skillOld.Slug, Version: "1.0.0",
		Source: model.LocalSkillSourceEnterprise, InstalledAt: &now, LastSeenAt: &now,
	}
	testDB.Create(&lis)
	if got := installStatusForInstanceWithSlug(t, skillIDs, "2.0.0", skillOld.Slug, localInst.ID); got != "outdated" {
		t.Fatalf("本地旧版本仍在：期望 outdated，实际=%q", got)
	}
}

// TestBuildSkillInstanceQuery_LocalInstance_MultiScopeLIS_NoDuplicate 回归用例：
// 同一 slug 在同一本地实例上可同时存在于 user / project 两个 scope 的 local_instance_skills
// （见 LocalInstanceSkill 唯一约束 (scope, instance_id, workspace_path, slug)）。
// BuildSkillInstanceQuery 之前的 JOIN 按 (instance_id, slug) 直接 LEFT JOIN，会把这个实例
// 扇出成多行，导致 /admin/skills/instances 返回重复数据（与 enterprise_rule 的 JOIN 扇出老 bug 同类）。
// 修复后改用 MAX(id) 聚合子查询，每个 (instance_id, slug) 至多匹配一行 lis，应返回 1 行且 installed。
// （installStatusForInstanceWithSlug 内部已断言 len(rows)==1，重复会直接失败）
func TestBuildSkillInstanceQuery_LocalInstance_MultiScopeLIS_NoDuplicate(t *testing.T) {
	setupSkillInstancesDB(t)
	testDB := model.DB(context.Background())

	skill := seedAdminUninstallSkill(t, "local-multiscope")
	user := model.User{Username: "local-multiscope-user"}
	testDB.Create(&user)
	localInst := model.Instance{
		Name: "local-codebuddy-ms", InstanceId: "local-codebuddy-ms-001",
		UserID: user.ID, Source: model.InstanceSourceLocal, AgentType: "codebuddy",
	}
	testDB.Create(&localInst)

	task := model.SkillDistributionTask{
		SkillID: skill.ID, Version: skill.Version, Total: 1, Status: "completed",
		Type: model.TaskTypeDistribute,
	}
	testDB.Create(&task)
	record := model.SkillDistributionRecord{
		TaskID: task.ID, SkillID: skill.ID, InstanceID: localInst.ID,
		InstanceCID: localInst.InstanceId, Version: skill.Version,
		Status: "success", Type: model.TaskTypeDistribute,
	}
	testDB.Create(&record)

	// 同一 slug 在同一实例上写两行不同 scope 的本地事实快照（user + project）。
	now := time.Now()
	for _, sc := range []string{model.LocalSkillScopeUser, model.LocalSkillScopeWorkspace} {
		wp := ""
		if sc == model.LocalSkillScopeWorkspace {
			wp = "/home/alex/proj1"
		}
		lis := model.LocalInstanceSkill{
			InstanceID:    localInst.ID,
			Slug:          skill.Slug,
			Version:       skill.Version,
			Source:        model.LocalSkillSourceEnterprise,
			Scope:         sc,
			WorkspacePath: wp,
			InstalledAt:   &now,
			LastSeenAt:    &now,
		}
		if err := testDB.Create(&lis).Error; err != nil {
			t.Fatalf("create lis row (scope=%s): %v", sc, err)
		}
	}

	skillIDs := []uint{skill.ID}
	// 返回 1 行（不被扇出），且本地确有装着 → installed。
	if got := installStatusForInstanceWithSlug(t, skillIDs, skill.Version, skill.Slug, localInst.ID); got != "installed" {
		t.Fatalf("多 scope lis 同 slug：期望 installed，实际=%q", got)
	}
}

// TestBuildSkillInstanceQuery_CVMInstance_UnaffectedByLisBranch
// 回归守护：CVM 实例（source != 'local'）不受 lis JOIN 影响，installed 仍为 installed。
func TestBuildSkillInstanceQuery_CVMInstance_UnaffectedByLisBranch(t *testing.T) {
	setupSkillInstancesDB(t)
	testDB := model.DB(context.Background())

	skill := seedAdminUninstallSkill(t, "cvm-not-affected")
	user := model.User{Username: "cvm-user"}
	testDB.Create(&user)
	cvmInst := model.Instance{
		Name: "cvm-inst-1", InstanceId: "ins-cvm-001",
		UserID: user.ID, AgentType: "openclaw", // Source 默认为非 local
	}
	testDB.Create(&cvmInst)

	task := model.SkillDistributionTask{
		SkillID: skill.ID, Version: skill.Version, Total: 1, Status: "completed",
		Type: model.TaskTypeDistribute,
	}
	testDB.Create(&task)
	record := model.SkillDistributionRecord{
		TaskID: task.ID, SkillID: skill.ID, InstanceID: cvmInst.ID,
		InstanceCID: cvmInst.InstanceId, Version: skill.Version,
		Status: "success", Type: model.TaskTypeDistribute,
	}
	testDB.Create(&record)

	skillIDs := []uint{skill.ID}

	// CVM 实例无 lis 行→ 仍为 installed（不因 lis IS NULL 而变 uninstalled）
	if got := installStatusForInstanceWithSlug(t, skillIDs, skill.Version, skill.Slug, cvmInst.ID); got != "installed" {
		t.Fatalf("CVM 实例无 lis 行：期望 installed，实际=%q", got)
	}
}
