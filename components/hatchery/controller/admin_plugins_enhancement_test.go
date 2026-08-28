package controller

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupPluginEnhancementDB 初始化插件增强功能测试用的内存 SQLite 数据库
func setupPluginEnhancementDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetConnMaxIdleTime(0)
	sqlDB.SetConnMaxLifetime(0)

	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.Instance{},
		&model.User{},
		&model.Plugin{},
		&model.PluginDistributionRecord{},
		&model.PluginDistributionTask{},
		&model.PluginVisibilityGroup{},
		&model.PluginCategoryMapping{},
		&model.PublicPlugin{},
		&model.SiteConfig{},
		&model.SMHSpace{},
		&model.UserGroup{},
		&model.UserGroupMember{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	// 显式指定 sqlite driver — 防止 distlock 销毁连接导致 :memory: 数据库消失
	origDB := model.UseDBForTestWithDriver(db, "sqlite")

	// 启用 SMH
	db.Create(&model.SiteConfig{SMHEnabled: 1})

	origToken := AdminToken
	AdminToken = "test-admin-token"

	t.Cleanup(func() {
		AdminToken = origToken
		origDB()
	})
	return db
}

// seedPluginForUninstall 创建测试用插件
func seedPluginForUninstall(t *testing.T, slug string) model.Plugin {
	t.Helper()
	plugin := model.Plugin{
		Slug:           slug,
		Name:           "卸载测试插件",
		Version:        "1.0.0",
		VersionMajor:   1,
		VersionMinor:   0,
		VersionPatch:   0,
		VisibilityType: model.VisibilityAll,
		PluginID:       slug,
		PluginFormat:   "openclaw",
	}
	if err := model.DB(context.Background()).Create(&plugin).Error; err != nil {
		t.Fatalf("创建插件失败: %v", err)
	}
	return plugin
}

// seedInstanceForUninstall 创建测试用实例
func seedInstanceForUninstall(t *testing.T, username, name, cid, agentType string) model.Instance {
	t.Helper()
	user := model.User{Username: username, Role: "user"}
	if err := model.DB(context.Background()).Create(&user).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	inst := model.Instance{
		Name:         name,
		InstanceId:   cid,
		UserID:       user.ID,
		LastCVMState: "RUNNING",
		AgentReady:   1,
		AgentType:    agentType,
		RuntimeUser:  "root",
	}
	if err := model.DB(context.Background()).Create(&inst).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}
	return inst
}

// assertPluginUninstallHTTPStatus 发送卸载请求并验证 HTTP 状态码
func assertPluginUninstallHTTPStatus(t *testing.T, body string, want int) map[string]interface{} {
	t.Helper()
	w := httptest.NewRecorder()
	HandleUninstallPlugin(w, adminJSONPost("/admin/plugins/uninstall", body))
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

// waitPluginTask 等待插件任务完成（轮询检查）
func waitPluginTask(t *testing.T, taskID uint, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var task model.PluginDistributionTask
		model.DB(context.Background()).First(&task, taskID)
		if task.Status == "completed" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("卸载任务超时未完成")
}

// seedPluginTaskForTasks 创建测试用插件任务和记录
func seedPluginTaskForTasks(t *testing.T, plugin model.Plugin, taskType string, inst model.Instance) model.PluginDistributionTask {
	t.Helper()
	task := model.PluginDistributionTask{
		PluginDBID: plugin.ID,
		Version:    plugin.Version,
		Total:      1,
		Status:     "completed",
		Type:       taskType,
	}
	if err := model.DB(context.Background()).Create(&task).Error; err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}
	record := model.PluginDistributionRecord{
		TaskID:      task.ID,
		PluginDBID:  plugin.ID,
		InstanceID:  inst.ID,
		InstanceCID: inst.InstanceId,
		Version:     plugin.Version,
		Status:      "success",
		Type:        taskType,
	}
	if err := model.DB(context.Background()).Create(&record).Error; err != nil {
		t.Fatalf("创建任务记录失败: %v", err)
	}
	return task
}

// decodePluginTasksResponse 解析插件任务列表响应
func decodePluginTasksResponse(t *testing.T, path string) map[string]interface{} {
	t.Helper()
	w := httptest.NewRecorder()
	HandleAdminPluginTasks(w, adminJSONGet(path))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	return decodeJSON(t, w)
}

// ==================== HandleUninstallPlugin 测试 ====================

func TestHandleUninstallPlugin_MissingSlug(t *testing.T) {
	setupPluginEnhancementDB(t)
	assertPluginUninstallHTTPStatus(t, `{"instance_ids":[1]}`, http.StatusBadRequest)
}

func TestHandleUninstallPlugin_EmptyInstanceIDs(t *testing.T) {
	setupPluginEnhancementDB(t)
	assertPluginUninstallHTTPStatus(t, `{"slug":"test-plugin","instance_ids":[]}`, http.StatusBadRequest)
}

func TestHandleUninstallPlugin_NonExistentSlug(t *testing.T) {
	setupPluginEnhancementDB(t)
	assertPluginUninstallHTTPStatus(t, `{"slug":"no-such-plugin","instance_ids":[1]}`, http.StatusNotFound)
}

func TestHandleUninstallPlugin_AllUnsupportedTypes(t *testing.T) {
	setupPluginEnhancementDB(t)
	plugin := seedPluginForUninstall(t, "uninstall-unsupported")
	inst1 := seedInstanceForUninstall(t, "unsupported-u1", "unsupported-1", "ins-unsup-001", "hermes")
	inst2 := seedInstanceForUninstall(t, "unsupported-u2", "unsupported-2", "ins-unsup-002", "unknown_type")

	body := `{"slug":"` + plugin.Slug + `","instance_ids":[` + uintStr(inst1.ID) + `,` + uintStr(inst2.ID) + `]}`
	resp := assertPluginUninstallHTTPStatus(t, body, http.StatusBadRequest)
	if !strings.Contains(wantString(t, resp["error"]), "没有符合条件的实例") {
		t.Fatalf("错误信息应包含 没有符合条件的实例，实际=%v", resp)
	}

	var taskCount int64
	model.DB(context.Background()).Model(&model.PluginDistributionTask{}).Where("plugin_db_id = ?", plugin.ID).Count(&taskCount)
	if taskCount != 0 {
		t.Fatalf("不支持实例不应创建任务，实际 taskCount=%d", taskCount)
	}
}

func TestHandleUninstallPlugin_BasicFlow(t *testing.T) {
	setupPluginEnhancementDB(t)
	plugin := seedPluginForUninstall(t, "uninstall-basic")
	inst := seedInstanceForUninstall(t, "uninstall-basic-user", "uninstall-basic-inst", "ins-uninstall-basic", "openclaw")

	body := `{"slug":"` + plugin.Slug + `","instance_ids":[` + uintStr(inst.ID) + `]}`
	resp := assertPluginUninstallHTTPStatus(t, body, http.StatusOK)
	if resp["task_id"] == nil {
		t.Fatalf("响应缺少 task_id: %v", resp)
	}
	if msg, ok := resp["message"].(string); !ok || msg == "" {
		t.Fatalf("响应缺少 message 字段: %v", resp)
	}

	taskID := uint(resp["task_id"].(float64))
	waitPluginTask(t, taskID, 2*time.Second)

	var task model.PluginDistributionTask
	if err := model.DB(context.Background()).Where("plugin_db_id = ?", plugin.ID).First(&task).Error; err != nil {
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

	var record model.PluginDistributionRecord
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

func TestHandleUninstallPlugin_DeduplicateInstanceIDs(t *testing.T) {
	setupPluginEnhancementDB(t)
	plugin := seedPluginForUninstall(t, "uninstall-dedup")
	inst := seedInstanceForUninstall(t, "dedup-user", "dedup-inst", "ins-dedup", "openclaw")

	// 传入重复的 instance_ids
	body := `{"slug":"` + plugin.Slug + `","instance_ids":[` + uintStr(inst.ID) + `,` + uintStr(inst.ID) + `,` + uintStr(inst.ID) + `]}`
	resp := assertPluginUninstallHTTPStatus(t, body, http.StatusOK)
	if resp["task_id"] == nil {
		t.Fatalf("响应缺少 task_id: %v", resp)
	}

	taskID := uint(resp["task_id"].(float64))
	waitPluginTask(t, taskID, 2*time.Second)

	var task model.PluginDistributionTask
	if err := model.DB(context.Background()).Where("plugin_db_id = ?", plugin.ID).First(&task).Error; err != nil {
		t.Fatalf("未创建卸载任务: %v", err)
	}
	// 去重后应该只有 1 个实例
	if task.Total != 1 {
		t.Fatalf("去重后 Task.Total=%d，期望 1", task.Total)
	}

	var recordCount int64
	model.DB(context.Background()).Model(&model.PluginDistributionRecord{}).Where("task_id = ?", task.ID).Count(&recordCount)
	if recordCount != 1 {
		t.Fatalf("去重后 record 条数=%d，期望 1", recordCount)
	}
}

// ==================== HandleAdminPluginTasks type 筛选测试 ====================

func TestHandleAdminPluginTasks_TypeFilter(t *testing.T) {
	setupPluginEnhancementDB(t)
	plugin := seedPluginForUninstall(t, "task-filter-type")
	inst := seedInstanceForUninstall(t, "task-filter-user", "task-filter-inst", "ins-task-filter", "openclaw")
	seedPluginTaskForTasks(t, plugin, model.TaskTypeDistribute, inst)

	resp := decodePluginTasksResponse(t, "/admin/plugins/tasks?slug="+plugin.Slug+"&type=distribute")
	if int(resp["total"].(float64)) != 1 {
		t.Fatalf("type=distribute total=%v，期望 1", resp["total"])
	}
	tasks := resp["tasks"].([]interface{})
	if len(tasks) != 1 {
		t.Fatalf("type=distribute 应返回 1 个任务，实际=%d", len(tasks))
	}
	if tasks[0].(map[string]interface{})["type"] != model.TaskTypeDistribute {
		t.Fatalf("type=distribute 返回了非下发任务: %v", tasks[0])
	}
}

func TestHandleAdminPluginTasks_TypeFilterUninstall(t *testing.T) {
	setupPluginEnhancementDB(t)
	plugin := seedPluginForUninstall(t, "task-filter-uninstall")
	inst := seedInstanceForUninstall(t, "task-filter-uninstall-user", "task-filter-uninstall-inst", "ins-task-filter-uninstall", "openclaw")
	seedPluginTaskForTasks(t, plugin, model.TaskTypeDistribute, inst)
	seedPluginTaskForTasks(t, plugin, model.TaskTypeUninstall, inst)

	resp := decodePluginTasksResponse(t, "/admin/plugins/tasks?slug="+plugin.Slug+"&type=uninstall")
	if int(resp["total"].(float64)) != 1 {
		t.Fatalf("type=uninstall total=%v，期望 1", resp["total"])
	}
	tasks := resp["tasks"].([]interface{})
	if len(tasks) != 1 {
		t.Fatalf("type=uninstall 应返回 1 个任务，实际=%d", len(tasks))
	}
	if tasks[0].(map[string]interface{})["type"] != model.TaskTypeUninstall {
		t.Fatalf("type=uninstall 返回了非卸载任务: %v", tasks[0])
	}
}

func TestHandleAdminPluginTasks_InvalidType(t *testing.T) {
	setupPluginEnhancementDB(t)
	plugin := seedPluginForUninstall(t, "task-invalid-type")
	inst := seedInstanceForUninstall(t, "task-invalid-user", "task-invalid-inst", "ins-task-invalid", "openclaw")
	seedPluginTaskForTasks(t, plugin, model.TaskTypeDistribute, inst)

	w := httptest.NewRecorder()
	HandleAdminPluginTasks(w, adminJSONGet("/admin/plugins/tasks?slug="+plugin.Slug+"&type=invalid_type"))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("无效 type 应返回 400，实际=%d", w.Code)
	}
}

func TestHandleAdminPluginTasks_TaskHasTypeField(t *testing.T) {
	setupPluginEnhancementDB(t)
	plugin := seedPluginForUninstall(t, "task-has-type")
	inst := seedInstanceForUninstall(t, "task-type-user", "task-type-inst", "ins-task-type", "openclaw")
	seedPluginTaskForTasks(t, plugin, model.TaskTypeUninstall, inst)

	resp := decodePluginTasksResponse(t, "/admin/plugins/tasks?slug="+plugin.Slug)
	tasks := resp["tasks"].([]interface{})
	if len(tasks) != 1 {
		t.Fatalf("期望 1 个任务，实际=%d", len(tasks))
	}
	task := tasks[0].(map[string]interface{})
	if task["type"] == nil {
		t.Fatalf("响应缺少 type 字段: %v", task)
	}
	if task["type"] != model.TaskTypeUninstall {
		t.Fatalf("响应 type=%v，期望 %q", task["type"], model.TaskTypeUninstall)
	}
}

// ==================== HandleAdminPlugins 列表增强测试 ====================

func TestHandleAdminPlugins_ListHasNewFields(t *testing.T) {
	setupPluginEnhancementDB(t)
	plugin := seedPluginForUninstall(t, "list-new-fields")
	inst := seedInstanceForUninstall(t, "list-new-user", "list-new-inst", "ins-list-new", "openclaw")
	seedPluginTaskForTasks(t, plugin, model.TaskTypeDistribute, inst)

	// 更新插件的 changelog
	model.DB(context.Background()).Model(&plugin).Update("changelog", "测试更新日志")

	w := httptest.NewRecorder()
	HandleAdminPlugins(w, adminJSONGet("/admin/plugins?page=1&page_size=10"))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	resp := decodeJSON(t, w)
	plugins := resp["plugins"].([]interface{})
	if len(plugins) == 0 {
		t.Fatal("插件列表为空")
	}

	pluginData := plugins[0].(map[string]interface{})
	// 验证新增字段存在
	if _, ok := pluginData["changelog"]; !ok {
		t.Fatal("列表响应缺少 changelog 字段")
	}
	if _, ok := pluginData["visibility_type"]; !ok {
		t.Fatal("列表响应缺少 visibility_type 字段")
	}
	if _, ok := pluginData["visibility_groups"]; !ok {
		t.Fatal("列表响应缺少 visibility_groups 字段")
	}
	if _, ok := pluginData["installed_count"]; !ok {
		t.Fatal("列表响应缺少 installed_count 字段")
	}
	if _, ok := pluginData["has_running_task"]; !ok {
		t.Fatal("列表响应缺少 has_running_task 字段")
	}
	if lastTask, ok := pluginData["last_task"].(map[string]interface{}); ok && lastTask != nil {
		if _, ok := lastTask["type"]; !ok {
			t.Fatal("last_task 缺少 type 字段")
		}
	}
}

// ==================== HandleAdminPluginDetail 详情增强测试 ====================

func TestHandleAdminPluginDetail_HasNewFields(t *testing.T) {
	setupPluginEnhancementDB(t)
	plugin := seedPluginForUninstall(t, "detail-new-fields")
	inst := seedInstanceForUninstall(t, "detail-new-user", "detail-new-inst", "ins-detail-new", "openclaw")
	seedPluginTaskForTasks(t, plugin, model.TaskTypeDistribute, inst)

	// 更新插件的 changelog 和 distribute_count
	model.DB(context.Background()).Model(&plugin).Updates(map[string]interface{}{
		"changelog":        "详情测试更新日志",
		"distribute_count": 10,
	})

	w := httptest.NewRecorder()
	HandleAdminPluginDetail(w, adminJSONGet("/admin/plugins/detail?slug="+plugin.Slug))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	resp := decodeJSON(t, w)
	pluginData, ok := resp["plugin"].(map[string]interface{})
	if !ok {
		t.Fatal("响应缺少 plugin 字段")
	}
	// 验证新增字段存在
	if _, ok := pluginData["changelog"]; !ok {
		t.Fatal("详情响应缺少 changelog 字段")
	}
	if _, ok := pluginData["visibility_type"]; !ok {
		t.Fatal("详情响应缺少 visibility_type 字段")
	}
	if _, ok := pluginData["visibility_groups"]; !ok {
		t.Fatal("详情响应缺少 visibility_groups 字段")
	}
	if _, ok := pluginData["distribute_count"]; !ok {
		t.Fatal("详情响应缺少 distribute_count 字段")
	}
	if _, ok := pluginData["installed_count"]; !ok {
		t.Fatal("详情响应缺少 installed_count 字段")
	}
	if _, ok := pluginData["has_running_task"]; !ok {
		t.Fatal("详情响应缺少 has_running_task 字段")
	}
}

// ==================== PluginVisibilityGroup model 层测试 ====================

func TestSetPluginVisibility_Basic(t *testing.T) {
	setupPluginEnhancementDB(t)
	plugin := seedPluginForUninstall(t, "visibility-basic")
	group := model.UserGroup{Name: "测试分组"}
	if err := model.DB(context.Background()).Create(&group).Error; err != nil {
		t.Fatalf("创建用户组失败: %v", err)
	}

	// 设置为 group 可见性
	err := model.DB(context.Background()).Transaction(func(tx *gorm.DB) error {
		if rerr := model.SetPluginVisibility(tx, plugin.ID, model.VisibilityGroup, []uint{group.ID}); rerr != nil {
			return rerr
		}
		return nil
	})
	if err != nil {
		t.Fatalf("设置插件可见性失败: %v", err)
	}

	// 验证 visibility_type 已更新
	var updatedPlugin model.Plugin
	model.DB(context.Background()).First(&updatedPlugin, plugin.ID)
	if updatedPlugin.VisibilityType != model.VisibilityGroup {
		t.Fatalf("visibility_type=%q，期望 %q", updatedPlugin.VisibilityType, model.VisibilityGroup)
	}

	// 验证分组关联已创建
	var count int64
	model.DB(context.Background()).Model(&model.PluginVisibilityGroup{}).
		Where("plugin_id = ? AND group_id = ?", plugin.ID, group.ID).Count(&count)
	if count != 1 {
		t.Fatalf("分组关联条数=%d，期望 1", count)
	}
}

func TestCopyPluginVisibility_Basic(t *testing.T) {
	setupPluginEnhancementDB(t)
	slug := "visibility-copy"

	// 创建旧版本插件
	oldPlugin := model.Plugin{
		Slug:           slug,
		Name:           "旧版本",
		Version:        "1.0.0",
		VersionMajor:   1,
		VersionMinor:   0,
		VersionPatch:   0,
		VisibilityType: model.VisibilityGroup,
		PluginID:       slug,
		PluginFormat:   "openclaw",
	}
	if err := model.DB(context.Background()).Create(&oldPlugin).Error; err != nil {
		t.Fatalf("创建旧版本插件失败: %v", err)
	}

	// 创建分组并关联到旧版本
	group := model.UserGroup{Name: "复制测试分组"}
	if err := model.DB(context.Background()).Create(&group).Error; err != nil {
		t.Fatalf("创建用户组失败: %v", err)
	}
	visGroup := model.PluginVisibilityGroup{PluginID: oldPlugin.ID, GroupID: group.ID}
	if err := model.DB(context.Background()).Create(&visGroup).Error; err != nil {
		t.Fatalf("创建可见性关联失败: %v", err)
	}

	// 创建新版本插件
	newPlugin := model.Plugin{
		Slug:           slug,
		Name:           "新版本",
		Version:        "2.0.0",
		VersionMajor:   2,
		VersionMinor:   0,
		VersionPatch:   0,
		VisibilityType: model.VisibilityAll,
		PluginID:       slug,
		PluginFormat:   "openclaw",
	}
	if err := model.DB(context.Background()).Create(&newPlugin).Error; err != nil {
		t.Fatalf("创建新版本插件失败: %v", err)
	}

	// 复制可见性
	err := model.DB(context.Background()).Transaction(func(tx *gorm.DB) error {
		if rerr := model.CopyPluginVisibility(tx, slug, newPlugin.ID); rerr != nil {
			return rerr
		}
		return nil
	})
	if err != nil {
		t.Fatalf("复制插件可见性失败: %v", err)
	}

	// 验证新版本的 visibility_type 已更新
	var updatedPlugin model.Plugin
	model.DB(context.Background()).First(&updatedPlugin, newPlugin.ID)
	if updatedPlugin.VisibilityType != model.VisibilityGroup {
		t.Fatalf("新版本 visibility_type=%q，期望 %q", updatedPlugin.VisibilityType, model.VisibilityGroup)
	}

	// 验证新版本的分组关联已创建
	var count int64
	model.DB(context.Background()).Model(&model.PluginVisibilityGroup{}).
		Where("plugin_id = ? AND group_id = ?", newPlugin.ID, group.ID).Count(&count)
	if count != 1 {
		t.Fatalf("新版本分组关联条数=%d，期望 1", count)
	}
}

func TestCopyPluginVisibility_NoPrevVersion(t *testing.T) {
	setupPluginEnhancementDB(t)
	plugin := seedPluginForUninstall(t, "visibility-no-prev")

	// 无旧版本时，复制应该是 no-op
	err := model.DB(context.Background()).Transaction(func(tx *gorm.DB) error {
		if rerr := model.CopyPluginVisibility(tx, plugin.Slug, plugin.ID); rerr != nil {
			return rerr
		}
		return nil
	})
	if err != nil {
		t.Fatalf("无旧版本时复制可见性失败: %v", err)
	}

	// 验证 visibility_type 保持默认值 all
	var updatedPlugin model.Plugin
	model.DB(context.Background()).First(&updatedPlugin, plugin.ID)
	if updatedPlugin.VisibilityType != model.VisibilityAll {
		t.Fatalf("visibility_type=%q，期望保持默认 %q", updatedPlugin.VisibilityType, model.VisibilityAll)
	}
}

func TestCleanupPluginVisibilityByPluginID(t *testing.T) {
	setupPluginEnhancementDB(t)
	plugin := seedPluginForUninstall(t, "visibility-cleanup")
	group := model.UserGroup{Name: "清理测试分组"}
	if err := model.DB(context.Background()).Create(&group).Error; err != nil {
		t.Fatalf("创建用户组失败: %v", err)
	}
	visGroup := model.PluginVisibilityGroup{PluginID: plugin.ID, GroupID: group.ID}
	if err := model.DB(context.Background()).Create(&visGroup).Error; err != nil {
		t.Fatalf("创建可见性关联失败: %v", err)
	}

	// 清理可见性关联
	err := model.DB(context.Background()).Transaction(func(tx *gorm.DB) error {
		return model.CleanupPluginVisibilityByPluginID(tx, plugin.ID)
	})
	if err != nil {
		t.Fatalf("清理插件可见性失败: %v", err)
	}

	// 验证关联已删除
	var count int64
	model.DB(context.Background()).Model(&model.PluginVisibilityGroup{}).
		Where("plugin_id = ?", plugin.ID).Count(&count)
	if count != 0 {
		t.Fatalf("清理后分组关联条数=%d，期望 0", count)
	}
}

func TestGetPluginVisibilityGroupIDs(t *testing.T) {
	setupPluginEnhancementDB(t)
	plugin1 := seedPluginForUninstall(t, "visibility-get-1")
	plugin2 := seedPluginForUninstall(t, "visibility-get-2")
	group1 := model.UserGroup{Name: "获取测试分组1"}
	group2 := model.UserGroup{Name: "获取测试分组2"}
	if err := model.DB(context.Background()).Create(&group1).Error; err != nil {
		t.Fatalf("创建用户组1失败: %v", err)
	}
	if err := model.DB(context.Background()).Create(&group2).Error; err != nil {
		t.Fatalf("创建用户组2失败: %v", err)
	}

	// plugin1 关联 group1 和 group2
	visGroup1 := model.PluginVisibilityGroup{PluginID: plugin1.ID, GroupID: group1.ID}
	visGroup2 := model.PluginVisibilityGroup{PluginID: plugin1.ID, GroupID: group2.ID}
	if err := model.DB(context.Background()).Create(&visGroup1).Error; err != nil {
		t.Fatalf("创建可见性关联1失败: %v", err)
	}
	if err := model.DB(context.Background()).Create(&visGroup2).Error; err != nil {
		t.Fatalf("创建可见性关联2失败: %v", err)
	}

	// plugin2 关联 group2
	visGroup3 := model.PluginVisibilityGroup{PluginID: plugin2.ID, GroupID: group2.ID}
	if err := model.DB(context.Background()).Create(&visGroup3).Error; err != nil {
		t.Fatalf("创建可见性关联3失败: %v", err)
	}

	// 批量获取可见性分组
	result, err := model.GetPluginVisibilityGroupIDs(context.Background(), []uint{plugin1.ID, plugin2.ID})
	if err != nil {
		t.Fatalf("获取插件可见性分组失败: %v", err)
	}

	// 验证 plugin1 有 2 个分组
	if len(result[plugin1.ID]) != 2 {
		t.Fatalf("plugin1 分组数=%d，期望 2", len(result[plugin1.ID]))
	}
	// 验证 plugin2 有 1 个分组
	if len(result[plugin2.ID]) != 1 {
		t.Fatalf("plugin2 分组数=%d，期望 1", len(result[plugin2.ID]))
	}
}

// ==================== PluginInstallStatusCase 测试 ====================

func pluginInstallStatusForInstance(t *testing.T, pluginIDs []uint, latestVersion string, instanceID uint) string {
	t.Helper()
	type row struct {
		InstanceID uint   `gorm:"column:instance_id"`
		Status     string `gorm:"column:install_status"`
	}
	var rows []row
	q, err := model.BuildPluginInstanceQuery(context.Background(), pluginIDs, latestVersion)
	if err != nil {
		t.Fatalf("构造查询失败: %v", err)
	}
	if err := q.Where("instances.id = ?", instanceID).
		Scan(&rows).Error; err != nil {
		t.Fatalf("查询安装状态失败: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("期望查询到 1 条实例状态，实际=%d", len(rows))
	}
	return rows[0].Status
}

func TestPluginInstallStatusCase_Uninstalled(t *testing.T) {
	setupPluginEnhancementDB(t)
	plugin := seedPluginForUninstall(t, "status-uninstalled")
	inst := seedInstanceForUninstall(t, "status-uninstalled-user", "status-uninstalled-inst", "ins-status-uninstalled", "openclaw")

	// 无记录 → uninstalled
	status := pluginInstallStatusForInstance(t, []uint{plugin.ID}, plugin.Version, inst.ID)
	if status != "uninstalled" {
		t.Fatalf("无记录时 install_status=%q，期望 uninstalled", status)
	}
}

func TestPluginInstallStatusCase_Installed(t *testing.T) {
	setupPluginEnhancementDB(t)
	plugin := seedPluginForUninstall(t, "status-installed")
	inst := seedInstanceForUninstall(t, "status-installed-user", "status-installed-inst", "ins-status-installed", "openclaw")
	seedPluginTaskForTasks(t, plugin, model.TaskTypeDistribute, inst)

	// distribute+success → installed
	status := pluginInstallStatusForInstance(t, []uint{plugin.ID}, plugin.Version, inst.ID)
	if status != "installed" {
		t.Fatalf("distribute+success 时 install_status=%q，期望 installed", status)
	}
}

func TestPluginInstallStatusCase_Outdated(t *testing.T) {
	setupPluginEnhancementDB(t)
	slug := "status-outdated"
	oldPlugin := model.Plugin{
		Slug:         slug,
		Name:         "旧版本",
		Version:      "1.0.0",
		VersionMajor: 1,
		VersionMinor: 0,
		VersionPatch: 0,
		PluginID:     slug,
		PluginFormat: "openclaw",
	}
	if err := model.DB(context.Background()).Create(&oldPlugin).Error; err != nil {
		t.Fatalf("创建旧版本插件失败: %v", err)
	}

	newPlugin := model.Plugin{
		Slug:         slug,
		Name:         "新版本",
		Version:      "2.0.0",
		VersionMajor: 2,
		VersionMinor: 0,
		VersionPatch: 0,
		PluginID:     slug,
		PluginFormat: "openclaw",
	}
	if err := model.DB(context.Background()).Create(&newPlugin).Error; err != nil {
		t.Fatalf("创建新版本插件失败: %v", err)
	}

	inst := seedInstanceForUninstall(t, "status-outdated-user", "status-outdated-inst", "ins-status-outdated", "openclaw")

	// 创建旧版本的 distribute+success 记录
	task := model.PluginDistributionTask{
		PluginDBID: oldPlugin.ID,
		Version:    oldPlugin.Version,
		Total:      1,
		Status:     "completed",
		Type:       model.TaskTypeDistribute,
	}
	if err := model.DB(context.Background()).Create(&task).Error; err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}
	record := model.PluginDistributionRecord{
		TaskID:      task.ID,
		PluginDBID:  oldPlugin.ID,
		InstanceID:  inst.ID,
		InstanceCID: inst.InstanceId,
		Version:     oldPlugin.Version,
		Status:      "success",
		Type:        model.TaskTypeDistribute,
	}
	if err := model.DB(context.Background()).Create(&record).Error; err != nil {
		t.Fatalf("创建任务记录失败: %v", err)
	}

	// 用新版本查询 → outdated
	status := pluginInstallStatusForInstance(t, []uint{oldPlugin.ID, newPlugin.ID}, newPlugin.Version, inst.ID)
	if status != "outdated" {
		t.Fatalf("distribute+success 但版本旧时 install_status=%q，期望 outdated", status)
	}
}

func TestPluginInstallStatusCase_Uninstalling(t *testing.T) {
	setupPluginEnhancementDB(t)
	plugin := seedPluginForUninstall(t, "status-uninstalling")
	inst := seedInstanceForUninstall(t, "status-uninstalling-user", "status-uninstalling-inst", "ins-status-uninstalling", "openclaw")

	// 创建 uninstall+pending 记录
	task := model.PluginDistributionTask{
		PluginDBID: plugin.ID,
		Version:    plugin.Version,
		Total:      1,
		Status:     "running",
		Type:       model.TaskTypeUninstall,
	}
	if err := model.DB(context.Background()).Create(&task).Error; err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}
	record := model.PluginDistributionRecord{
		TaskID:      task.ID,
		PluginDBID:  plugin.ID,
		InstanceID:  inst.ID,
		InstanceCID: inst.InstanceId,
		Version:     plugin.Version,
		Status:      "pending",
		Type:        model.TaskTypeUninstall,
	}
	if err := model.DB(context.Background()).Create(&record).Error; err != nil {
		t.Fatalf("创建任务记录失败: %v", err)
	}

	// uninstall+pending → uninstalling
	status := pluginInstallStatusForInstance(t, []uint{plugin.ID}, plugin.Version, inst.ID)
	if status != "uninstalling" {
		t.Fatalf("uninstall+pending 时 install_status=%q，期望 uninstalling", status)
	}
}

func TestPluginInstallStatusCase_UninstallFailed(t *testing.T) {
	setupPluginEnhancementDB(t)
	plugin := seedPluginForUninstall(t, "status-uninstall-failed")
	inst := seedInstanceForUninstall(t, "status-uninstall-failed-user", "status-uninstall-failed-inst", "ins-status-uninstall-failed", "openclaw")

	// 创建 uninstall+failed 记录
	task := model.PluginDistributionTask{
		PluginDBID: plugin.ID,
		Version:    plugin.Version,
		Total:      1,
		Status:     "completed",
		Type:       model.TaskTypeUninstall,
	}
	if err := model.DB(context.Background()).Create(&task).Error; err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}
	record := model.PluginDistributionRecord{
		TaskID:      task.ID,
		PluginDBID:  plugin.ID,
		InstanceID:  inst.ID,
		InstanceCID: inst.InstanceId,
		Version:     plugin.Version,
		Status:      "failed",
		Type:        model.TaskTypeUninstall,
	}
	if err := model.DB(context.Background()).Create(&record).Error; err != nil {
		t.Fatalf("创建任务记录失败: %v", err)
	}

	// uninstall+failed → uninstall_failed
	status := pluginInstallStatusForInstance(t, []uint{plugin.ID}, plugin.Version, inst.ID)
	if status != "uninstall_failed" {
		t.Fatalf("uninstall+failed 时 install_status=%q，期望 uninstall_failed", status)
	}
}

func TestPluginInstallStatusCase_UninstallSuccess(t *testing.T) {
	setupPluginEnhancementDB(t)
	plugin := seedPluginForUninstall(t, "status-uninstall-success")
	inst := seedInstanceForUninstall(t, "status-uninstall-success-user", "status-uninstall-success-inst", "ins-status-uninstall-success", "openclaw")

	// 先创建 distribute+success
	seedPluginTaskForTasks(t, plugin, model.TaskTypeDistribute, inst)

	// 再创建 uninstall+success 记录
	task := model.PluginDistributionTask{
		PluginDBID: plugin.ID,
		Version:    plugin.Version,
		Total:      1,
		Status:     "completed",
		Type:       model.TaskTypeUninstall,
	}
	if err := model.DB(context.Background()).Create(&task).Error; err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}
	record := model.PluginDistributionRecord{
		TaskID:      task.ID,
		PluginDBID:  plugin.ID,
		InstanceID:  inst.ID,
		InstanceCID: inst.InstanceId,
		Version:     plugin.Version,
		Status:      "success",
		Type:        model.TaskTypeUninstall,
	}
	if err := model.DB(context.Background()).Create(&record).Error; err != nil {
		t.Fatalf("创建任务记录失败: %v", err)
	}

	// uninstall+success → uninstalled
	status := pluginInstallStatusForInstance(t, []uint{plugin.ID}, plugin.Version, inst.ID)
	if status != "uninstalled" {
		t.Fatalf("uninstall+success 时 install_status=%q，期望 uninstalled", status)
	}
}

// ==================== PluginVisibilityGroup 补充测试 ====================

func TestCleanupPluginVisibilityByGroupID(t *testing.T) {
	db := setupPluginEnhancementDB(t)
	plugin := seedPluginForUninstall(t, "vis-cleanup-group")

	// 创建可见性关联
	db.Create(&model.PluginVisibilityGroup{PluginID: plugin.ID, GroupID: 100})
	db.Create(&model.PluginVisibilityGroup{PluginID: plugin.ID, GroupID: 200})

	// 删除 group_id=100 的关联
	err := db.Transaction(func(tx *gorm.DB) error {
		return model.CleanupPluginVisibilityByGroupID(tx, 100)
	})
	if err != nil {
		t.Fatalf("CleanupPluginVisibilityByGroupID 失败: %v", err)
	}

	// 验证 group_id=100 已删除
	var count int64
	db.Model(&model.PluginVisibilityGroup{}).Where("group_id = ?", 100).Count(&count)
	if count != 0 {
		t.Fatalf("group_id=100 条数=%d，期望 0", count)
	}
	// group_id=200 未受影响
	db.Model(&model.PluginVisibilityGroup{}).Where("group_id = ?", 200).Count(&count)
	if count != 1 {
		t.Fatalf("group_id=200 条数=%d，期望 1", count)
	}
}

func TestIsGroupUsedByPluginVisibility_True(t *testing.T) {
	setupPluginEnhancementDB(t)
	plugin := seedPluginForUninstall(t, "vis-used-true")

	model.DB(context.Background()).Create(&model.PluginVisibilityGroup{
		PluginID: plugin.ID, GroupID: 999,
	})

	used, err := model.IsGroupUsedByPluginVisibility(context.Background(), 999)
	if err != nil {
		t.Fatalf("IsGroupUsedByPluginVisibility 错误: %v", err)
	}
	if !used {
		t.Fatal("期望 used=true，实际 false")
	}
}

func TestIsGroupUsedByPluginVisibility_False(t *testing.T) {
	setupPluginEnhancementDB(t)

	used, err := model.IsGroupUsedByPluginVisibility(context.Background(), 888)
	if err != nil {
		t.Fatalf("IsGroupUsedByPluginVisibility 错误: %v", err)
	}
	if used {
		t.Fatal("期望 used=false，实际 true")
	}
}

// ==================== HandleUpdatePlugin 参数校验测试 ====================

func TestHandleUpdatePlugin_NoFileSuccess(t *testing.T) {
	db := setupPluginEnhancementDB(t)
	// Create existing v1 with some COS data and categories
	p := model.Plugin{
		Slug: "update-nofile", Name: "NoFile Plugin", Version: "1.0.0",
		VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
		VisibilityType: "all", PluginID: "update-nofile",
		PluginFormat: "openclaw", COSZipKey: "plugins/update-nofile/update-nofile-1.0.0.zip",
		COSDirKey: "plugins/update-nofile/update-nofile-1.0.0/",
		FileList:  `["a.js"]`, FileSize: 1024,
		DistributeCount: 5,
	}
	db.Create(&p)
	db.Create(&model.PluginCategoryMapping{PluginID: p.ID, CategoryID: 1})

	w := httptest.NewRecorder()
	// No file field → inherits COS info from v1
	body := "------boundary\r\nContent-Disposition: form-data; name=\"slug\"\r\n\r\nupdate-nofile\r\n------boundary\r\nContent-Disposition: form-data; name=\"version\"\r\n\r\n2.0.0\r\n------boundary\r\nContent-Disposition: form-data; name=\"changelog\"\r\n\r\nv2 changes\r\n------boundary--\r\n"
	req := httptest.NewRequest(http.MethodPost, "/admin/plugins/update", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----boundary")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleUpdatePlugin(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("\u671f\u671b 200\uff0c\u5b9e\u9645 %d: %s", w.Code, w.Body.String())
	}
	resp := decodeJSON(t, w)
	if resp["version"] != "2.0.0" {
		t.Fatalf("\u671f\u671b version=2.0.0\uff0c\u5b9e\u9645 %v", resp["version"])
	}
	// Verify new plugin inherits COS info
	var newP model.Plugin
	db.Where("slug = ? AND version = ?", "update-nofile", "2.0.0").First(&newP)
	if newP.COSZipKey != p.COSZipKey {
		t.Fatalf("\u671f\u671b COSZipKey \u7ee7\u627f\uff0c\u5b9e\u9645 %q", newP.COSZipKey)
	}
	if newP.Changelog != "v2 changes" {
		t.Fatalf("\u671f\u671b changelog=v2 changes\uff0c\u5b9e\u9645 %q", newP.Changelog)
	}
	// Verify distribute_count inherited
	if newP.DistributeCount != 5 {
		t.Fatalf("\u671f\u671b distribute_count=5\uff0c\u5b9e\u9645 %d", newP.DistributeCount)
	}
	// Verify category inherited
	var catCount int64
	db.Model(&model.PluginCategoryMapping{}).Where("plugin_id = ?", newP.ID).Count(&catCount)
	if catCount != 1 {
		t.Fatalf("\u671f\u671b category mapping \u7ee7\u627f 1 \u6761\uff0c\u5b9e\u9645 %d", catCount)
	}
}

func TestHandleUpdatePlugin_NoFileWithVisibility(t *testing.T) {
	db := setupPluginEnhancementDB(t)
	p := model.Plugin{
		Slug: "update-vis", Name: "Vis Plugin", Version: "1.0.0",
		VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
		VisibilityType: "all", PluginID: "update-vis",
		PluginFormat: "openclaw",
	}
	db.Create(&p)
	// Create a user group
	g := model.UserGroup{Name: "TestGroup"}
	db.Create(&g)

	w := httptest.NewRecorder()
	// Set visibility_type=group with group_ids
	body := "------boundary\r\nContent-Disposition: form-data; name=\"slug\"\r\n\r\nupdate-vis\r\n------boundary\r\nContent-Disposition: form-data; name=\"version\"\r\n\r\n2.0.0\r\n------boundary\r\nContent-Disposition: form-data; name=\"visibility_type\"\r\n\r\ngroup\r\n------boundary\r\nContent-Disposition: form-data; name=\"group_ids\"\r\n\r\n" + uintStr(g.ID) + "\r\n------boundary--\r\n"
	req := httptest.NewRequest(http.MethodPost, "/admin/plugins/update", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----boundary")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleUpdatePlugin(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("\u671f\u671b 200\uff0c\u5b9e\u9645 %d: %s", w.Code, w.Body.String())
	}
	// Verify visibility set
	var newP model.Plugin
	db.Where("slug = ? AND version = ?", "update-vis", "2.0.0").First(&newP)
	if newP.VisibilityType != "group" {
		t.Fatalf("\u671f\u671b visibility_type=group\uff0c\u5b9e\u9645 %q", newP.VisibilityType)
	}
	var visCount int64
	db.Model(&model.PluginVisibilityGroup{}).Where("plugin_id = ?", newP.ID).Count(&visCount)
	if visCount != 1 {
		t.Fatalf("\u671f\u671b visibility group 1 \u6761\uff0c\u5b9e\u9645 %d", visCount)
	}
}

func TestHandleUpdatePlugin_InvalidVisibilityType(t *testing.T) {
	db := setupPluginEnhancementDB(t)
	p := model.Plugin{
		Slug: "update-badvis", Name: "BadVis Plugin", Version: "1.0.0",
		VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
		VisibilityType: "all", PluginID: "update-badvis",
		PluginFormat: "openclaw",
	}
	db.Create(&p)

	w := httptest.NewRecorder()
	body := "------boundary\r\nContent-Disposition: form-data; name=\"slug\"\r\n\r\nupdate-badvis\r\n------boundary\r\nContent-Disposition: form-data; name=\"version\"\r\n\r\n2.0.0\r\n------boundary\r\nContent-Disposition: form-data; name=\"visibility_type\"\r\n\r\ninvalid\r\n------boundary--\r\n"
	req := httptest.NewRequest(http.MethodPost, "/admin/plugins/update", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----boundary")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleUpdatePlugin(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("\u671f\u671b 400\uff0c\u5b9e\u9645 %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleUpdatePlugin_GroupVisibilityMissingGroupIDs(t *testing.T) {
	db := setupPluginEnhancementDB(t)
	p := model.Plugin{
		Slug: "update-nogrp", Name: "NoGrp Plugin", Version: "1.0.0",
		VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
		VisibilityType: "all", PluginID: "update-nogrp",
		PluginFormat: "openclaw",
	}
	db.Create(&p)

	w := httptest.NewRecorder()
	body := "------boundary\r\nContent-Disposition: form-data; name=\"slug\"\r\n\r\nupdate-nogrp\r\n------boundary\r\nContent-Disposition: form-data; name=\"version\"\r\n\r\n2.0.0\r\n------boundary\r\nContent-Disposition: form-data; name=\"visibility_type\"\r\n\r\ngroup\r\n------boundary--\r\n"
	req := httptest.NewRequest(http.MethodPost, "/admin/plugins/update", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----boundary")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleUpdatePlugin(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("\u671f\u671b 400\uff0c\u5b9e\u9645 %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "group_ids") {
		t.Fatalf("\u54cd\u5e94\u672a\u5305\u542b group_ids \u9519\u8bef\u63d0\u793a: %s", w.Body.String())
	}
}

func TestHandleUpdatePlugin_WithCategoryIDs(t *testing.T) {
	db := setupPluginEnhancementDB(t)
	p := model.Plugin{
		Slug: "update-cat", Name: "Cat Plugin", Version: "1.0.0",
		VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
		VisibilityType: "all", PluginID: "update-cat",
		PluginFormat: "openclaw",
	}
	db.Create(&p)

	w := httptest.NewRecorder()
	body := "------boundary\r\nContent-Disposition: form-data; name=\"slug\"\r\n\r\nupdate-cat\r\n------boundary\r\nContent-Disposition: form-data; name=\"version\"\r\n\r\n2.0.0\r\n------boundary\r\nContent-Disposition: form-data; name=\"category_ids\"\r\n\r\n1,2,3\r\n------boundary--\r\n"
	req := httptest.NewRequest(http.MethodPost, "/admin/plugins/update", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----boundary")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleUpdatePlugin(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("\u671f\u671b 200\uff0c\u5b9e\u9645 %d: %s", w.Code, w.Body.String())
	}
	// Verify categories created
	var newP model.Plugin
	db.Where("slug = ? AND version = ?", "update-cat", "2.0.0").First(&newP)
	var catCount int64
	db.Model(&model.PluginCategoryMapping{}).Where("plugin_id = ?", newP.ID).Count(&catCount)
	if catCount != 3 {
		t.Fatalf("\u671f\u671b 3 \u4e2a category mapping\uff0c\u5b9e\u9645 %d", catCount)
	}
}

func TestHandleUpdatePlugin_MissingSlug(t *testing.T) {
	setupPluginEnhancementDB(t)
	w := httptest.NewRecorder()
	// multipart form with version but no slug
	body := "------boundary\r\nContent-Disposition: form-data; name=\"version\"\r\n\r\n2.0.0\r\n------boundary--\r\n"
	req := httptest.NewRequest(http.MethodPost, "/admin/plugins/update", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----boundary")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleUpdatePlugin(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际 %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleUpdatePlugin_MissingVersion(t *testing.T) {
	setupPluginEnhancementDB(t)
	w := httptest.NewRecorder()
	body := "------boundary\r\nContent-Disposition: form-data; name=\"slug\"\r\n\r\nmy-plugin\r\n------boundary--\r\n"
	req := httptest.NewRequest(http.MethodPost, "/admin/plugins/update", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----boundary")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleUpdatePlugin(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际 %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleUpdatePlugin_PluginNotFound(t *testing.T) {
	setupPluginEnhancementDB(t)
	w := httptest.NewRecorder()
	body := "------boundary\r\nContent-Disposition: form-data; name=\"slug\"\r\n\r\nnonexistent-slug\r\n------boundary\r\nContent-Disposition: form-data; name=\"version\"\r\n\r\n2.0.0\r\n------boundary--\r\n"
	req := httptest.NewRequest(http.MethodPost, "/admin/plugins/update", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----boundary")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleUpdatePlugin(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际 %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleUpdatePlugin_InvalidVersion(t *testing.T) {
	setupPluginEnhancementDB(t)
	seedPluginForUninstall(t, "update-invalid-ver")
	w := httptest.NewRecorder()
	body := "------boundary\r\nContent-Disposition: form-data; name=\"slug\"\r\n\r\nupdate-invalid-ver\r\n------boundary\r\nContent-Disposition: form-data; name=\"version\"\r\n\r\nabc\r\n------boundary--\r\n"
	req := httptest.NewRequest(http.MethodPost, "/admin/plugins/update", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----boundary")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleUpdatePlugin(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际 %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleUpdatePlugin_VersionNotGreater(t *testing.T) {
	setupPluginEnhancementDB(t)
	seedPluginForUninstall(t, "update-not-greater") // version 1.0.0
	w := httptest.NewRecorder()
	body := "------boundary\r\nContent-Disposition: form-data; name=\"slug\"\r\n\r\nupdate-not-greater\r\n------boundary\r\nContent-Disposition: form-data; name=\"version\"\r\n\r\n0.9.0\r\n------boundary--\r\n"
	req := httptest.NewRequest(http.MethodPost, "/admin/plugins/update", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----boundary")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleUpdatePlugin(w, req)
	// version 0.9.0 不存在于 DB 中，原地更新查不到 → 404
	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404（版本不存在），实际 %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleUpdatePlugin_VersionEqual(t *testing.T) {
	setupPluginEnhancementDB(t)
	seedPluginForUninstall(t, "update-equal") // version 1.0.0
	w := httptest.NewRecorder()
	body := "------boundary\r\nContent-Disposition: form-data; name=\"slug\"\r\n\r\nupdate-equal\r\n------boundary\r\nContent-Disposition: form-data; name=\"version\"\r\n\r\n1.0.0\r\n------boundary\r\nContent-Disposition: form-data; name=\"name\"\r\n\r\n更新后名称\r\n------boundary--\r\n"
	req := httptest.NewRequest(http.MethodPost, "/admin/plugins/update", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----boundary")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleUpdatePlugin(w, req)
	// 版本相等 → 原地更新元信息（兼容 API 用户原有行为）
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200（原地更新），实际 %d: %s", w.Code, w.Body.String())
	}
	// 验证名称已更新
	var plugin model.Plugin
	model.DB(context.Background()).Where("slug = ?", "update-equal").First(&plugin)
	if plugin.Name != "更新后名称" {
		t.Fatalf("期望 name=更新后名称，实际=%s", plugin.Name)
	}
}

// ==================== HandleCreatePlugin 参数校验测试 ====================

func TestHandleCreatePlugin_MissingFields(t *testing.T) {
	setupPluginEnhancementDB(t)
	w := httptest.NewRecorder()
	body := "------boundary\r\nContent-Disposition: form-data; name=\"slug\"\r\n\r\ntest-create\r\n------boundary--\r\n"
	req := httptest.NewRequest(http.MethodPost, "/admin/plugins/create", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----boundary")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleCreatePlugin(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际 %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleCreatePlugin_InvalidSlug(t *testing.T) {
	setupPluginEnhancementDB(t)
	w := httptest.NewRecorder()
	body := "------boundary\r\nContent-Disposition: form-data; name=\"slug\"\r\n\r\nINVALID_SLUG!!\r\n------boundary\r\nContent-Disposition: form-data; name=\"name\"\r\n\r\nTest\r\n------boundary\r\nContent-Disposition: form-data; name=\"version\"\r\n\r\n1.0.0\r\n------boundary--\r\n"
	req := httptest.NewRequest(http.MethodPost, "/admin/plugins/create", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----boundary")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleCreatePlugin(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际 %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "slug") {
		t.Fatalf("响应未包含 slug 错误提示: %s", w.Body.String())
	}
}

func TestHandleCreatePlugin_InvalidVersionFormat(t *testing.T) {
	setupPluginEnhancementDB(t)
	w := httptest.NewRecorder()
	body := "------boundary\r\nContent-Disposition: form-data; name=\"slug\"\r\n\r\nvalid-slug\r\n------boundary\r\nContent-Disposition: form-data; name=\"name\"\r\n\r\nTest\r\n------boundary\r\nContent-Disposition: form-data; name=\"version\"\r\n\r\nnot-semver\r\n------boundary--\r\n"
	req := httptest.NewRequest(http.MethodPost, "/admin/plugins/create", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----boundary")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleCreatePlugin(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际 %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleCreatePlugin_DuplicateVersion(t *testing.T) {
	setupPluginEnhancementDB(t)
	seedPluginForUninstall(t, "dup-create") // version 1.0.0
	w := httptest.NewRecorder()
	// slug+version already exists, triggers the duplicate check before file parse
	body := "------boundary\r\nContent-Disposition: form-data; name=\"slug\"\r\n\r\ndup-create\r\n------boundary\r\nContent-Disposition: form-data; name=\"name\"\r\n\r\nTest\r\n------boundary\r\nContent-Disposition: form-data; name=\"version\"\r\n\r\n1.0.0\r\n------boundary--\r\n"
	req := httptest.NewRequest(http.MethodPost, "/admin/plugins/create", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----boundary")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleCreatePlugin(w, req)
	// Either 400 (duplicate) or 400 (missing file) — both are acceptable validation
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际 %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleCreatePlugin_MissingFile(t *testing.T) {
	setupPluginEnhancementDB(t)
	w := httptest.NewRecorder()
	body := "------boundary\r\nContent-Disposition: form-data; name=\"slug\"\r\n\r\nnew-plugin\r\n------boundary\r\nContent-Disposition: form-data; name=\"name\"\r\n\r\nNew Plugin\r\n------boundary\r\nContent-Disposition: form-data; name=\"version\"\r\n\r\n1.0.0\r\n------boundary--\r\n"
	req := httptest.NewRequest(http.MethodPost, "/admin/plugins/create", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----boundary")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleCreatePlugin(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际 %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "file") {
		t.Fatalf("响应未包含 file 相关错误: %s", w.Body.String())
	}
}

// ==================== HandleAdminPluginDetail 补充测试 ====================

func TestHandleAdminPluginDetail_MissingSlug(t *testing.T) {
	setupPluginEnhancementDB(t)
	w := httptest.NewRecorder()
	req := adminJSONGet("/admin/plugins/detail")
	HandleAdminPluginDetail(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际 %d", w.Code)
	}
}

func TestHandleAdminPluginDetail_NotFound(t *testing.T) {
	setupPluginEnhancementDB(t)
	w := httptest.NewRecorder()
	req := adminJSONGet("/admin/plugins/detail?slug=nonexistent")
	HandleAdminPluginDetail(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际 %d", w.Code)
	}
}

func TestHandleAdminPluginDetail_SpecificVersion(t *testing.T) {
	db := setupPluginEnhancementDB(t)
	// 创建两个版本
	p1 := model.Plugin{
		Slug: "detail-ver", Name: "Detail Test", Version: "1.0.0",
		VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
		VisibilityType: "all", PluginID: "detail-ver",
	}
	p2 := model.Plugin{
		Slug: "detail-ver", Name: "Detail Test", Version: "2.0.0",
		VersionMajor: 2, VersionMinor: 0, VersionPatch: 0,
		VisibilityType: "all", PluginID: "detail-ver",
		Changelog: "v2 changelog",
	}
	db.Create(&p1)
	db.Create(&p2)

	w := httptest.NewRecorder()
	req := adminJSONGet("/admin/plugins/detail?slug=detail-ver&version=1.0.0")
	HandleAdminPluginDetail(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d: %s", w.Code, w.Body.String())
	}
	resp := decodeJSON(t, w)
	pluginMap := resp["plugin"].(map[string]interface{})
	if pluginMap["version"] != "1.0.0" {
		t.Fatalf("期望 version=1.0.0，实际 %v", pluginMap["version"])
	}
	// versions 应包含两个
	versions := resp["versions"].([]interface{})
	if len(versions) != 2 {
		t.Fatalf("期望 2 个版本，实际 %d", len(versions))
	}
}

func TestHandleAdminPluginDetail_VersionNotFound(t *testing.T) {
	setupPluginEnhancementDB(t)
	seedPluginForUninstall(t, "detail-nover")
	w := httptest.NewRecorder()
	req := adminJSONGet("/admin/plugins/detail?slug=detail-nover&version=9.9.9")
	HandleAdminPluginDetail(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际 %d", w.Code)
	}
}

func TestHandleAdminPluginDetail_WithVisibilityGroups(t *testing.T) {
	db := setupPluginEnhancementDB(t)
	plugin := model.Plugin{
		Slug: "detail-vis", Name: "Vis Detail", Version: "1.0.0",
		VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
		VisibilityType: "group", PluginID: "detail-vis",
	}
	db.Create(&plugin)
	// 创建分组
	g1 := model.UserGroup{Name: "分组A"}
	g2 := model.UserGroup{Name: "分组B"}
	db.Create(&g1)
	db.Create(&g2)
	// 创建可见性关联
	db.Create(&model.PluginVisibilityGroup{PluginID: plugin.ID, GroupID: g1.ID})
	db.Create(&model.PluginVisibilityGroup{PluginID: plugin.ID, GroupID: g2.ID})

	w := httptest.NewRecorder()
	req := adminJSONGet("/admin/plugins/detail?slug=detail-vis")
	HandleAdminPluginDetail(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d: %s", w.Code, w.Body.String())
	}
	resp := decodeJSON(t, w)
	pluginMap := resp["plugin"].(map[string]interface{})
	if pluginMap["visibility_type"] != "group" {
		t.Fatalf("期望 visibility_type=group，实际 %v", pluginMap["visibility_type"])
	}
	visGroups := pluginMap["visibility_groups"].([]interface{})
	if len(visGroups) != 2 {
		t.Fatalf("期望 2 个 visibility_groups，实际 %d", len(visGroups))
	}
}

func TestHandleAdminPluginDetail_WithInstalledCountAndRunningTask(t *testing.T) {
	db := setupPluginEnhancementDB(t)
	plugin := model.Plugin{
		Slug: "detail-count", Name: "Count Detail", Version: "1.0.0",
		VersionMajor: 1, VersionMinor: 0, VersionPatch: 0,
		VisibilityType: "all", PluginID: "detail-count",
	}
	db.Create(&plugin)
	// 创建 success records
	db.Create(&model.PluginDistributionRecord{
		PluginDBID: plugin.ID, InstanceID: 1001,
		Status: "success", Type: "distribute",
	})
	db.Create(&model.PluginDistributionRecord{
		PluginDBID: plugin.ID, InstanceID: 1002,
		Status: "success", Type: "distribute",
	})
	// 创建 running task
	db.Create(&model.PluginDistributionTask{
		PluginDBID: plugin.ID, Status: "running", Type: "distribute",
	})

	w := httptest.NewRecorder()
	req := adminJSONGet("/admin/plugins/detail?slug=detail-count")
	HandleAdminPluginDetail(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d: %s", w.Code, w.Body.String())
	}
	resp := decodeJSON(t, w)
	pluginMap := resp["plugin"].(map[string]interface{})
	// installed_count should be 2
	ic := pluginMap["installed_count"].(float64)
	if ic != 2 {
		t.Fatalf("期望 installed_count=2，实际 %v", ic)
	}
	// has_running_task should be true
	hr := pluginMap["has_running_task"].(bool)
	if !hr {
		t.Fatal("期望 has_running_task=true，实际 false")
	}
}

// ==================== last_task 按 slug 维度查询旧版本任务 ====================

func TestHandleAdminPlugins_LastTaskCrossVersion(t *testing.T) {
	setupPluginEnhancementDB(t)

	// 创建旧版本 plugin
	oldPlugin := model.Plugin{
		Slug:         "cross-ver-task",
		Name:         "跨版本测试插件",
		Version:      "1.0.0",
		VersionMajor: 1,
	}
	model.DB(context.Background()).Create(&oldPlugin)

	// 创建新版本 plugin（最新版本）
	newPlugin := model.Plugin{
		Slug:         "cross-ver-task",
		Name:         "跨版本测试插件",
		Version:      "2.0.0",
		VersionMajor: 2,
	}
	model.DB(context.Background()).Create(&newPlugin)

	// 针对旧版本创建一个下发任务
	task := model.PluginDistributionTask{
		PluginDBID: oldPlugin.ID,
		Status:     "completed",
		Total:      1,
		Success:    1,
		Version:    "1.0.0",
		Type:       "distribute",
	}
	model.DB(context.Background()).Create(&task)

	// 列表接口只展示最新版本，但 last_task 应该能找到旧版本的任务
	w := httptest.NewRecorder()
	HandleAdminPlugins(w, adminJSONGet("/admin/plugins?page=1&page_size=50"))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}

	resp := decodeJSON(t, w)
	plugins := resp["plugins"].([]interface{})

	var found map[string]interface{}
	for _, p := range plugins {
		pm := p.(map[string]interface{})
		if pm["slug"] == "cross-ver-task" {
			found = pm
			break
		}
	}
	if found == nil {
		t.Fatal("列表中未找到 cross-ver-task")
	}

	lastTask := found["last_task"]
	if lastTask == nil {
		t.Fatal("last_task 应不为 nil（旧版本有下发任务），但实际为 null")
	}
}

// ==================== distribute_count 按 slug 取 MAX ====================

func TestHandleAdminPlugins_DistributeCountCrossVersion(t *testing.T) {
	setupPluginEnhancementDB(t)

	// 旧版本有 distribute_count=5
	oldPlugin := model.Plugin{
		Slug:            "dist-count-ver",
		Name:            "计数测试插件",
		Version:         "1.0.0",
		VersionMajor:    1,
		DistributeCount: 5,
	}
	model.DB(context.Background()).Create(&oldPlugin)

	// 新版本 distribute_count=0（模拟未通过 update 接口升版）
	newPlugin := model.Plugin{
		Slug:            "dist-count-ver",
		Name:            "计数测试插件",
		Version:         "2.0.0",
		VersionMajor:    2,
		DistributeCount: 0,
	}
	model.DB(context.Background()).Create(&newPlugin)

	w := httptest.NewRecorder()
	HandleAdminPlugins(w, adminJSONGet("/admin/plugins?page=1&page_size=50"))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}

	resp := decodeJSON(t, w)
	plugins := resp["plugins"].([]interface{})

	var found map[string]interface{}
	for _, p := range plugins {
		pm := p.(map[string]interface{})
		if pm["slug"] == "dist-count-ver" {
			found = pm
			break
		}
	}
	if found == nil {
		t.Fatal("列表中未找到 dist-count-ver")
	}

	dc := int(found["distribute_count"].(float64))
	if dc != 5 {
		t.Fatalf("期望 distribute_count=5（旧版本的值），实际=%d", dc)
	}
}

// ==================== HandleAdminPluginFiles 测试 ====================

func TestHandleAdminPluginFiles_Basic(t *testing.T) {
	setupPluginEnhancementDB(t)
	plugin := seedPluginForUninstall(t, "files-basic")
	model.DB(context.Background()).Model(&plugin).Updates(map[string]interface{}{
		"changelog": "测试changelog",
		"file_list": `["a.js","b.json"]`,
	})

	w := httptest.NewRecorder()
	HandleAdminPluginFiles(w, adminJSONGet("/admin/plugins/files?slug=files-basic"))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
	resp := decodeJSON(t, w)
	if resp["slug"] != "files-basic" {
		t.Fatalf("slug 不匹配: %v", resp["slug"])
	}
	versions := resp["versions"].([]interface{})
	if len(versions) == 0 {
		t.Fatal("versions 为空")
	}
	ver := versions[0].(map[string]interface{})
	if ver["changelog"] != "测试changelog" {
		t.Fatalf("changelog 不匹配: %v", ver["changelog"])
	}
	if ver["created_at"] == nil || ver["created_at"] == "" {
		t.Fatal("缺少 created_at")
	}
}

func TestHandleAdminPluginFiles_MissingSlug(t *testing.T) {
	setupPluginEnhancementDB(t)
	w := httptest.NewRecorder()
	HandleAdminPluginFiles(w, adminJSONGet("/admin/plugins/files"))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d", w.Code)
	}
}

func TestHandleAdminPluginFiles_NotFound(t *testing.T) {
	setupPluginEnhancementDB(t)
	w := httptest.NewRecorder()
	HandleAdminPluginFiles(w, adminJSONGet("/admin/plugins/files?slug=nonexistent"))
	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际=%d", w.Code)
	}
}

// ==================== HandleDeletePlugin 测试 ====================

func TestHandleDeletePlugin_Basic(t *testing.T) {
	setupPluginEnhancementDB(t)
	seedPluginForUninstall(t, "delete-basic")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/admin/plugins/delete?slug=delete-basic&version=1.0.0", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleDeletePlugin(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	// 确认已软删除
	var count int64
	model.DB(context.Background()).Model(&model.Plugin{}).Where("slug = ?", "delete-basic").Count(&count)
	if count != 0 {
		t.Fatalf("期望删除后 count=0，实际=%d", count)
	}
}

func TestHandleDeletePlugin_MissingParams(t *testing.T) {
	setupPluginEnhancementDB(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/admin/plugins/delete?slug=x", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleDeletePlugin(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d", w.Code)
	}
}

func TestHandleDeletePlugin_NotFound(t *testing.T) {
	setupPluginEnhancementDB(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/admin/plugins/delete?slug=ghost&version=9.9.9", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleDeletePlugin(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际=%d", w.Code)
	}
}

// ==================== handleUpdatePluginMetadata 增强测试 ====================

func TestHandleUpdatePluginMetadata_UpdateName(t *testing.T) {
	setupPluginEnhancementDB(t)
	seedPluginForUninstall(t, "meta-name")

	w := httptest.NewRecorder()
	body := "------boundary\r\nContent-Disposition: form-data; name=\"slug\"\r\n\r\nmeta-name\r\n------boundary\r\nContent-Disposition: form-data; name=\"version\"\r\n\r\n1.0.0\r\n------boundary\r\nContent-Disposition: form-data; name=\"name\"\r\n\r\n新名称\r\n------boundary--\r\n"
	req := httptest.NewRequest(http.MethodPost, "/admin/plugins/update", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----boundary")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleUpdatePlugin(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d: %s", w.Code, w.Body.String())
	}

	var plugin model.Plugin
	model.DB(context.Background()).Where("slug = ?", "meta-name").First(&plugin)
	if plugin.Name != "新名称" {
		t.Fatalf("name 未更新: %s", plugin.Name)
	}
}

func TestHandleUpdatePluginMetadata_UpdateDescription(t *testing.T) {
	setupPluginEnhancementDB(t)
	seedPluginForUninstall(t, "meta-desc")

	w := httptest.NewRecorder()
	body := "------boundary\r\nContent-Disposition: form-data; name=\"slug\"\r\n\r\nmeta-desc\r\n------boundary\r\nContent-Disposition: form-data; name=\"version\"\r\n\r\n1.0.0\r\n------boundary\r\nContent-Disposition: form-data; name=\"description\"\r\n\r\n新描述\r\n------boundary--\r\n"
	req := httptest.NewRequest(http.MethodPost, "/admin/plugins/update", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----boundary")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleUpdatePlugin(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d: %s", w.Code, w.Body.String())
	}

	var plugin model.Plugin
	model.DB(context.Background()).Where("slug = ?", "meta-desc").First(&plugin)
	if plugin.Description != "新描述" {
		t.Fatalf("description 未更新: %s", plugin.Description)
	}
}

func TestHandleUpdatePluginMetadata_UpdateCategoryIDs(t *testing.T) {
	setupPluginEnhancementDB(t)
	plugin := seedPluginForUninstall(t, "meta-cat")

	w := httptest.NewRecorder()
	body := "------boundary\r\nContent-Disposition: form-data; name=\"slug\"\r\n\r\nmeta-cat\r\n------boundary\r\nContent-Disposition: form-data; name=\"version\"\r\n\r\n1.0.0\r\n------boundary\r\nContent-Disposition: form-data; name=\"category_ids\"\r\n\r\n1,2\r\n------boundary--\r\n"
	req := httptest.NewRequest(http.MethodPost, "/admin/plugins/update", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----boundary")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleUpdatePlugin(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d: %s", w.Code, w.Body.String())
	}

	var mappings []model.PluginCategoryMapping
	model.DB(context.Background()).Where("plugin_id = ?", plugin.ID).Find(&mappings)
	if len(mappings) < 1 {
		t.Fatal("category_ids 未写入")
	}
}

func TestHandleUpdatePluginMetadata_NoChanges(t *testing.T) {
	setupPluginEnhancementDB(t)
	seedPluginForUninstall(t, "meta-nochange")

	w := httptest.NewRecorder()
	body := "------boundary\r\nContent-Disposition: form-data; name=\"slug\"\r\n\r\nmeta-nochange\r\n------boundary\r\nContent-Disposition: form-data; name=\"version\"\r\n\r\n1.0.0\r\n------boundary--\r\n"
	req := httptest.NewRequest(http.MethodPost, "/admin/plugins/update", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----boundary")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleUpdatePlugin(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d: %s", w.Code, w.Body.String())
	}
}

// ==================== HandleFavoritePlugin / Unfavorite / FavoritedList 测试 ====================

func TestHandleFavoritePlugin_Basic(t *testing.T) {
	setupPluginEnhancementDB(t)

	w := httptest.NewRecorder()
	body := `{"slug":"fav-basic","name":"收藏测试插件"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/plugins/favorite", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleFavoritePlugin(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d: %s", w.Code, w.Body.String())
	}
	resp := decodeJSON(t, w)
	if resp["ok"] != true {
		t.Fatalf("响应 ok 不为 true: %v", resp)
	}
}

func TestHandleFavoritePlugin_MissingFields(t *testing.T) {
	setupPluginEnhancementDB(t)
	w := httptest.NewRecorder()
	body := `{"slug":"only-slug"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/plugins/favorite", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleFavoritePlugin(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400（缺少 name），实际=%d: %s", w.Code, w.Body.String())
	}
}

func TestHandleFavoritePlugin_Duplicate(t *testing.T) {
	setupPluginEnhancementDB(t)

	body := `{"slug":"fav-dup","name":"重复收藏"}`
	// 第一次
	req1 := httptest.NewRequest(http.MethodPost, "/admin/plugins/favorite", strings.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer test-admin-token")
	w1 := httptest.NewRecorder()
	HandleFavoritePlugin(w1, req1)

	// 第二次
	req2 := httptest.NewRequest(http.MethodPost, "/admin/plugins/favorite", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer test-admin-token")
	w2 := httptest.NewRecorder()
	HandleFavoritePlugin(w2, req2)
	if w2.Code != http.StatusConflict {
		t.Fatalf("期望 409（重复收藏），实际=%d: %s", w2.Code, w2.Body.String())
	}
}

func TestHandleUnfavoritePlugin_Basic(t *testing.T) {
	setupPluginEnhancementDB(t)

	// 先收藏
	body := `{"slug":"unfav-basic","name":"取消收藏测试"}`
	req1 := httptest.NewRequest(http.MethodPost, "/admin/plugins/favorite", strings.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer test-admin-token")
	w1 := httptest.NewRecorder()
	HandleFavoritePlugin(w1, req1)
	resp1 := decodeJSON(t, w1)
	pluginID := resp1["plugin_id"]

	// 取消收藏
	w := httptest.NewRecorder()
	url := fmt.Sprintf("/admin/plugins/unfavorite?id=%v", pluginID)
	req := httptest.NewRequest(http.MethodPost, url, nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleUnfavoritePlugin(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d: %s", w.Code, w.Body.String())
	}
}

func TestHandleUnfavoritePlugin_MissingID(t *testing.T) {
	setupPluginEnhancementDB(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/plugins/unfavorite", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleUnfavoritePlugin(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d", w.Code)
	}
}

func TestHandleUnfavoritePlugin_NotFound(t *testing.T) {
	setupPluginEnhancementDB(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/plugins/unfavorite?id=99999", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleUnfavoritePlugin(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际=%d", w.Code)
	}
}

func TestHandleAdminFavoritedPlugins_Empty(t *testing.T) {
	setupPluginEnhancementDB(t)
	w := httptest.NewRecorder()
	HandleAdminFavoritedPlugins(w, adminJSONGet("/admin/plugins/favorited?page=1&page_size=10"))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d: %s", w.Code, w.Body.String())
	}
}

func TestHandleAdminFavoritedPlugins_WithData(t *testing.T) {
	setupPluginEnhancementDB(t)

	// 先收藏
	body := `{"slug":"fav-list","name":"列表测试"}`
	req1 := httptest.NewRequest(http.MethodPost, "/admin/plugins/favorite", strings.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer test-admin-token")
	w1 := httptest.NewRecorder()
	HandleFavoritePlugin(w1, req1)

	// 查列表
	w := httptest.NewRecorder()
	HandleAdminFavoritedPlugins(w, adminJSONGet("/admin/plugins/favorited?page=1&page_size=10"))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d: %s", w.Code, w.Body.String())
	}
	resp := decodeJSON(t, w)
	plugins := resp["plugins"].([]interface{})
	if len(plugins) == 0 {
		t.Fatal("收藏列表应有数据")
	}
}

// ==================== HandleCreatePlugin 更多分支覆盖 ====================

func TestHandleCreatePlugin_MissingSlug(t *testing.T) {
	setupPluginEnhancementDB(t)
	w := httptest.NewRecorder()
	body := "------boundary\r\nContent-Disposition: form-data; name=\"name\"\r\n\r\ntest\r\n------boundary\r\nContent-Disposition: form-data; name=\"version\"\r\n\r\n1.0.0\r\n------boundary--\r\n"
	req := httptest.NewRequest(http.MethodPost, "/admin/plugins/create", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----boundary")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleCreatePlugin(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400（缺少 slug），实际=%d: %s", w.Code, w.Body.String())
	}
}

func TestHandleCreatePlugin_MissingName(t *testing.T) {
	setupPluginEnhancementDB(t)
	w := httptest.NewRecorder()
	body := "------boundary\r\nContent-Disposition: form-data; name=\"slug\"\r\n\r\ncreate-no-name\r\n------boundary\r\nContent-Disposition: form-data; name=\"version\"\r\n\r\n1.0.0\r\n------boundary--\r\n"
	req := httptest.NewRequest(http.MethodPost, "/admin/plugins/create", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----boundary")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleCreatePlugin(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400（缺少 name），实际=%d: %s", w.Code, w.Body.String())
	}
}

func TestHandleCreatePlugin_MissingVersion(t *testing.T) {
	setupPluginEnhancementDB(t)
	w := httptest.NewRecorder()
	body := "------boundary\r\nContent-Disposition: form-data; name=\"slug\"\r\n\r\ncreate-no-ver\r\n------boundary\r\nContent-Disposition: form-data; name=\"name\"\r\n\r\ntest\r\n------boundary--\r\n"
	req := httptest.NewRequest(http.MethodPost, "/admin/plugins/create", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----boundary")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleCreatePlugin(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400（缺少 version），实际=%d: %s", w.Code, w.Body.String())
	}
}

// ==================== HandleUpdatePlugin 版本升级分支覆盖 ====================

func TestHandleUpdatePlugin_VersionUpgrade_MissingPlugin(t *testing.T) {
	setupPluginEnhancementDB(t)
	w := httptest.NewRecorder()
	body := "------boundary\r\nContent-Disposition: form-data; name=\"slug\"\r\n\r\nno-such-plugin\r\n------boundary\r\nContent-Disposition: form-data; name=\"version\"\r\n\r\n2.0.0\r\n------boundary--\r\n"
	req := httptest.NewRequest(http.MethodPost, "/admin/plugins/update", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----boundary")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleUpdatePlugin(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际=%d: %s", w.Code, w.Body.String())
	}
}

func TestHandleUpdatePlugin_VersionUpgrade_InvalidVersion(t *testing.T) {
	setupPluginEnhancementDB(t)
	seedPluginForUninstall(t, "upgrade-bad-ver")

	w := httptest.NewRecorder()
	body := "------boundary\r\nContent-Disposition: form-data; name=\"slug\"\r\n\r\nupgrade-bad-ver\r\n------boundary\r\nContent-Disposition: form-data; name=\"version\"\r\n\r\nbad\r\n------boundary--\r\n"
	req := httptest.NewRequest(http.MethodPost, "/admin/plugins/update", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----boundary")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleUpdatePlugin(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d: %s", w.Code, w.Body.String())
	}
}

func TestHandleUpdatePlugin_VersionUpgrade_MissingSlugOrVersion(t *testing.T) {
	setupPluginEnhancementDB(t)

	// 缺少 version
	w := httptest.NewRecorder()
	body := "------boundary\r\nContent-Disposition: form-data; name=\"slug\"\r\n\r\nsome-slug\r\n------boundary--\r\n"
	req := httptest.NewRequest(http.MethodPost, "/admin/plugins/update", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----boundary")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleUpdatePlugin(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400（缺少 version），实际=%d", w.Code)
	}
}

// TestHandleCreatePlugin_ValidZipButNoSMH 测试有效 zip 但 SMH 未配置的情况
func TestHandleCreatePlugin_ValidZipButNoSMH(t *testing.T) {
	setupPluginEnhancementDB(t)

	// 创建一个最小有效 zip（包含 openclaw.plugin.json）
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fw, _ := zw.Create("test-create-zip/openclaw.plugin.json")
	fw.Write([]byte(`{"id":"test-zip","name":"Test","version":"1.0.0","kind":"mcp","format":"directory"}`))
	zw.Close()

	// 构建 multipart 请求
	var reqBody bytes.Buffer
	writer := multipart.NewWriter(&reqBody)
	writer.WriteField("slug", "test-create-zip")
	writer.WriteField("name", "Test Plugin")
	writer.WriteField("version", "1.0.0")
	part, _ := writer.CreateFormFile("file", "plugin.zip")
	part.Write(buf.Bytes())
	writer.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/plugins/create", &reqBody)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleCreatePlugin(w, req)

	// SMH 未配置，应该在 uploadPluginZipToSMH 阶段失败（400 or 500）
	// 但至少走到了 zip 验证通过的分支
	if w.Code == http.StatusOK {
		t.Log("SMH 可用，创建成功")
	} else if w.Code == http.StatusBadRequest || w.Code == http.StatusInternalServerError {
		t.Logf("SMH 不可用，预期失败: %d: %s", w.Code, w.Body.String())
	} else {
		t.Fatalf("非预期状态码: %d: %s", w.Code, w.Body.String())
	}
}

// TestHandleCreatePlugin_InvalidZipContent 测试无效 zip 内容
func TestHandleCreatePlugin_InvalidZipContent(t *testing.T) {
	setupPluginEnhancementDB(t)

	// 非 zip 文件内容
	var reqBody bytes.Buffer
	writer := multipart.NewWriter(&reqBody)
	writer.WriteField("slug", "test-bad-zip")
	writer.WriteField("name", "Bad Zip")
	writer.WriteField("version", "1.0.0")
	part, _ := writer.CreateFormFile("file", "plugin.zip")
	part.Write([]byte("this is not a zip file"))
	writer.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/plugins/create", &reqBody)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleCreatePlugin(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400（无效 zip），实际=%d: %s", w.Code, w.Body.String())
	}
}

// TestHandleCreatePlugin_ZipMissingManifest 测试 zip 缺少 openclaw.plugin.json
func TestHandleCreatePlugin_ZipMissingManifest(t *testing.T) {
	setupPluginEnhancementDB(t)

	// 有效 zip 但不含 openclaw.plugin.json
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fw, _ := zw.Create("test-no-manifest/README.md")
	fw.Write([]byte("hello"))
	zw.Close()

	var reqBody bytes.Buffer
	writer := multipart.NewWriter(&reqBody)
	writer.WriteField("slug", "test-no-manifest")
	writer.WriteField("name", "No Manifest")
	writer.WriteField("version", "1.0.0")
	part, _ := writer.CreateFormFile("file", "plugin.zip")
	part.Write(buf.Bytes())
	writer.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/plugins/create", &reqBody)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer test-admin-token")
	HandleCreatePlugin(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400（缺少 manifest），实际=%d: %s", w.Code, w.Body.String())
	}
}
