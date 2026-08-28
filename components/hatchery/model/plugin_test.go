package model

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupPluginTestDB 创建临时 SQLite 数据库用于插件相关测试。
func setupPluginTestDB(t *testing.T) (cleanup func()) {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "hatchery_plugin_test_*.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	tmpFile.Close()

	dsn := tmpFile.Name() + "?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
	testDB, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("open test db: %v", err)
	}

	origDB := gdb
	gdb = testDB

	if err := gdb.AutoMigrate(
		&SiteConfig{},
		&Plugin{},
		&PluginDistributionTask{},
		&PluginDistributionRecord{},
		&PluginCategory{},
		&PluginCategoryMapping{},
		&PluginBundle{},
		&BundlePlugin{},
		&PublicPlugin{},
		&PluginInstallation{},
		&OpenClawRole{},
		&OpenClawRolePlugin{},
		&Instance{},
		&User{},
	); err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("auto migrate: %v", err)
	}

	// 创建默认 SiteConfig
	gdb.Create(&SiteConfig{Name: "test"})

	return func() {
		sqlDB, _ := gdb.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
		os.Remove(tmpFile.Name())
		os.Remove(tmpFile.Name() + "-wal")
		os.Remove(tmpFile.Name() + "-shm")
		gdb = origDB
	}
}

// ── Plugin.ParseVersion 测试 ──────────────────────────────────────

func TestPluginParseVersion(t *testing.T) {
	tests := []struct {
		version   string
		wantMajor int
		wantMinor int
		wantPatch int
		wantErr   bool
	}{
		{"1.0.0", 1, 0, 0, false},
		{"2.10.3", 2, 10, 3, false},
		{"0.0.1", 0, 0, 1, false},
		{"10.20.30", 10, 20, 30, false},
		{"100.200.300", 100, 200, 300, false},

		// 错误用例
		{"abc", 0, 0, 0, true},
		{"1.0", 0, 0, 0, true},
		{"1", 0, 0, 0, true},
		{"1.0.0.0", 0, 0, 0, true},
		{"a.b.c", 0, 0, 0, true},
		{"", 0, 0, 0, true},
		{"..", 0, 0, 0, true},
		{"1..0", 0, 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			p := &Plugin{Version: tt.version}
			err := p.ParseVersion()
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseVersion(%q) expected error", tt.version)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseVersion(%q) unexpected error: %v", tt.version, err)
				return
			}
			if p.VersionMajor != tt.wantMajor || p.VersionMinor != tt.wantMinor || p.VersionPatch != tt.wantPatch {
				t.Errorf("ParseVersion(%q) = %d.%d.%d, want %d.%d.%d",
					tt.version, p.VersionMajor, p.VersionMinor, p.VersionPatch,
					tt.wantMajor, tt.wantMinor, tt.wantPatch)
			}
		})
	}
}

// ── SeedPluginCategories 测试 ─────────────────────────────────────

func TestSeedPluginCategories(t *testing.T) {
	cleanup := setupPluginTestDB(t)
	defer cleanup()

	// 首次调用应创建所有预置分类
	SeedPluginCategories(gdb)

	var categories []PluginCategory
	gdb.Find(&categories)
	if len(categories) != len(predefinedPluginCategories) {
		t.Errorf("SeedPluginCategories(gdb) created %d categories, want %d",
			len(categories), len(predefinedPluginCategories))
	}

	// 验证分类名称
	nameSet := make(map[string]bool)
	for _, c := range categories {
		nameSet[c.Name] = true
	}
	for _, expected := range predefinedPluginCategories {
		if !nameSet[expected.Name] {
			t.Errorf("SeedPluginCategories(gdb) missing category: %s", expected.Name)
		}
	}

	// 重复调用应幂等（不会创建重复记录）
	SeedPluginCategories(gdb)

	var count int64
	gdb.Model(&PluginCategory{}).Count(&count)
	if int(count) != len(predefinedPluginCategories) {
		t.Errorf("SeedPluginCategories(gdb) idempotency failed: got %d, want %d",
			count, len(predefinedPluginCategories))
	}
}

func TestSeedPluginCategories_PartialExisting(t *testing.T) {
	cleanup := setupPluginTestDB(t)
	defer cleanup()

	// 预先创建部分分类
	gdb.Create(&PluginCategory{Name: "AI 模型提供商", Description: "已存在"})

	SeedPluginCategories(gdb)

	var count int64
	gdb.Model(&PluginCategory{}).Count(&count)
	if int(count) != len(predefinedPluginCategories) {
		t.Errorf("SeedPluginCategories(gdb) with partial existing: got %d, want %d",
			count, len(predefinedPluginCategories))
	}

	// 验证已存在的分类描述未被覆盖
	var existing PluginCategory
	gdb.Where("name = ?", "AI 模型提供商").First(&existing)
	if existing.Description != "已存在" {
		t.Errorf("SeedPluginCategories(gdb) overwrote existing description: got %q", existing.Description)
	}
}

// ── SeedDefaultPluginBundle 测试 ──────────────────────────────────

func TestSeedDefaultPluginBundle(t *testing.T) {
	cleanup := setupPluginTestDB(t)
	defer cleanup()

	// 首次调用应创建默认插件包
	SeedDefaultPluginBundle(gdb)

	var bundle PluginBundle
	if err := gdb.Where("name = ?", DefaultPluginBundleName).First(&bundle).Error; err != nil {
		t.Fatalf("SeedDefaultPluginBundle(gdb) did not create bundle: %v", err)
	}

	if bundle.Enabled {
		t.Error("SeedDefaultPluginBundle(gdb) should create disabled bundle by default")
	}
	if bundle.PluginCount != 0 {
		t.Errorf("SeedDefaultPluginBundle(gdb) plugin_count = %d, want 0", bundle.PluginCount)
	}

	// 验证标记已设置
	config := GetSiteConfig(context.Background())
	if !config.DefaultPluginBundleSeeded {
		t.Error("SeedDefaultPluginBundle(gdb) did not set DefaultPluginBundleSeeded flag")
	}
}

func TestSeedDefaultPluginBundle_Idempotent(t *testing.T) {
	cleanup := setupPluginTestDB(t)
	defer cleanup()

	// 调用两次
	SeedDefaultPluginBundle(gdb)
	SeedDefaultPluginBundle(gdb)

	var count int64
	gdb.Model(&PluginBundle{}).Where("name = ?", DefaultPluginBundleName).Count(&count)
	if count != 1 {
		t.Errorf("SeedDefaultPluginBundle(gdb) idempotency failed: got %d bundles, want 1", count)
	}
}

func TestSeedDefaultPluginBundle_ExistingBundle(t *testing.T) {
	cleanup := setupPluginTestDB(t)
	defer cleanup()

	// 预先创建同名插件包（模拟旧版本升级）
	gdb.Create(&PluginBundle{Name: DefaultPluginBundleName, PluginCount: 5, Enabled: true})

	SeedDefaultPluginBundle(gdb)

	// 应该只补标记，不创建新的
	var count int64
	gdb.Model(&PluginBundle{}).Where("name = ?", DefaultPluginBundleName).Count(&count)
	if count != 1 {
		t.Errorf("SeedDefaultPluginBundle(gdb) with existing bundle: got %d, want 1", count)
	}

	// 验证标记已设置
	config := GetSiteConfig(context.Background())
	if !config.DefaultPluginBundleSeeded {
		t.Error("SeedDefaultPluginBundle(gdb) did not set flag for existing bundle")
	}
}

// ── PluginInstallation 状态枚举测试 ──────────────────────────────

func TestPluginInstallStatusConstants(t *testing.T) {
	// 验证状态枚举值不重复且符合预期
	statuses := map[int]string{
		PluginInstallNone:      "未安装",
		PluginInstalling:       "安装中",
		PluginInstallSuccess:   "安装成功",
		PluginInstallFailed:    "安装失败",
		PluginInstallCancelled: "已取消",
	}

	if len(statuses) != 5 {
		t.Errorf("expected 5 unique status values, got %d", len(statuses))
	}

	if PluginInstallNone != 0 {
		t.Errorf("PluginInstallNone = %d, want 0", PluginInstallNone)
	}
	if PluginInstallSuccess != 2 {
		t.Errorf("PluginInstallSuccess = %d, want 2", PluginInstallSuccess)
	}
	if PluginInstallFailed != 3 {
		t.Errorf("PluginInstallFailed = %d, want 3", PluginInstallFailed)
	}
}

// ── Plugin CRUD 集成测试 ──────────────────────────────────────────

func TestPluginCRUD(t *testing.T) {
	cleanup := setupPluginTestDB(t)
	defer cleanup()

	// 创建插件
	plugin := Plugin{
		Slug:         "test-plugin",
		Name:         "Test Plugin",
		Description:  "A test plugin",
		Version:      "1.0.0",
		VersionMajor: 1,
		VersionMinor: 0,
		VersionPatch: 0,
		PluginID:     "test-plugin",
		PluginFormat: "openclaw",
		Kind:         "",
	}
	if err := gdb.Create(&plugin).Error; err != nil {
		t.Fatalf("Create plugin failed: %v", err)
	}
	if plugin.ID == 0 {
		t.Error("Plugin ID should be non-zero after creation")
	}

	// 读取
	var found Plugin
	if err := gdb.First(&found, plugin.ID).Error; err != nil {
		t.Fatalf("Find plugin failed: %v", err)
	}
	if found.Slug != "test-plugin" {
		t.Errorf("Plugin slug = %q, want %q", found.Slug, "test-plugin")
	}
	if found.PluginFormat != "openclaw" {
		t.Errorf("Plugin format = %q, want %q", found.PluginFormat, "openclaw")
	}

	// 更新
	gdb.Model(&found).Update("description", "Updated description")
	var updated Plugin
	gdb.First(&updated, plugin.ID)
	if updated.Description != "Updated description" {
		t.Errorf("Plugin description = %q, want %q", updated.Description, "Updated description")
	}

	// 软删除
	gdb.Delete(&updated)
	var deleted Plugin
	err := gdb.First(&deleted, plugin.ID).Error
	if err == nil {
		t.Error("Plugin should not be found after soft delete")
	}

	// Unscoped 查询应能找到
	var unscoped Plugin
	if err := gdb.Unscoped().First(&unscoped, plugin.ID).Error; err != nil {
		t.Errorf("Unscoped find failed: %v", err)
	}
}

func TestPluginSlugVersionUniqueness(t *testing.T) {
	cleanup := setupPluginTestDB(t)
	defer cleanup()

	p1 := Plugin{Slug: "my-plugin", Name: "P1", Version: "1.0.0", VersionMajor: 1}
	gdb.Create(&p1)

	// 同 slug+version 应失败
	p2 := Plugin{Slug: "my-plugin", Name: "P2", Version: "1.0.0", VersionMajor: 1}
	err := gdb.Create(&p2).Error
	if err == nil {
		t.Error("expected unique constraint violation for same slug+version")
	}

	// 不同 version 应成功
	p3 := Plugin{Slug: "my-plugin", Name: "P3", Version: "2.0.0", VersionMajor: 2}
	if err := gdb.Create(&p3).Error; err != nil {
		t.Errorf("different version should succeed: %v", err)
	}
}

// ── PluginCategoryMapping 测试 ────────────────────────────────────

func TestPluginCategoryMapping(t *testing.T) {
	cleanup := setupPluginTestDB(t)
	defer cleanup()

	cat := PluginCategory{Name: "测试分类"}
	gdb.Create(&cat)

	plugin := Plugin{Slug: "cat-test", Name: "Cat Test", Version: "1.0.0", VersionMajor: 1}
	gdb.Create(&plugin)

	mapping := PluginCategoryMapping{PluginID: plugin.ID, CategoryID: cat.ID}
	if err := gdb.Create(&mapping).Error; err != nil {
		t.Fatalf("Create mapping failed: %v", err)
	}

	// 重复映射应失败
	dup := PluginCategoryMapping{PluginID: plugin.ID, CategoryID: cat.ID}
	if err := gdb.Create(&dup).Error; err == nil {
		t.Error("expected unique constraint violation for duplicate mapping")
	}
}

// ── OpenClawRolePlugin 测试 ──────────────────────────────────────

func TestOpenClawRolePluginCRUD(t *testing.T) {
	cleanup := setupPluginTestDB(t)
	defer cleanup()

	role := OpenClawRole{Name: "测试角色", Visible: true}
	gdb.Create(&role)

	rp := OpenClawRolePlugin{
		OpenClawRoleID: role.ID,
		Name:           "Test Plugin",
		Slug:           "test-plugin",
		PluginID:       "test-plugin",
		Version:        "1.0.0",
		Source:         "enterprise",
		InstallMode:    "smh",
		Kind:           "memory",
	}
	if err := gdb.Create(&rp).Error; err != nil {
		t.Fatalf("Create role plugin failed: %v", err)
	}

	var found []OpenClawRolePlugin
	gdb.Where("open_claw_role_id = ?", role.ID).Find(&found)
	if len(found) != 1 {
		t.Errorf("expected 1 role plugin, got %d", len(found))
	}
	if found[0].Kind != "memory" {
		t.Errorf("role plugin kind = %q, want %q", found[0].Kind, "memory")
	}
	if found[0].InstallMode != "smh" {
		t.Errorf("role plugin install_mode = %q, want %q", found[0].InstallMode, "smh")
	}

	// 级联删除测试
	gdb.Where("open_claw_role_id = ?", role.ID).Delete(&OpenClawRolePlugin{})
	var afterDelete []OpenClawRolePlugin
	gdb.Where("open_claw_role_id = ?", role.ID).Find(&afterDelete)
	if len(afterDelete) != 0 {
		t.Errorf("expected 0 role plugins after delete, got %d", len(afterDelete))
	}
}

// ── PluginBundle + BundlePlugin 测试 ─────────────────────────────

func TestPluginBundleCRUD(t *testing.T) {
	cleanup := setupPluginTestDB(t)
	defer cleanup()

	bundle := PluginBundle{Name: "测试插件包", PluginCount: 0, Enabled: false}
	gdb.Create(&bundle)

	bp := BundlePlugin{
		PluginBundleID: bundle.ID,
		Name:           "Test Plugin",
		Slug:           "test-plugin",
		PluginID:       "test-plugin",
		Version:        "1.0.0",
		Source:         "enterprise",
		InstallMode:    "smh",
		Kind:           "",
	}
	gdb.Create(&bp)

	var plugins []BundlePlugin
	gdb.Where("plugin_bundle_id = ?", bundle.ID).Find(&plugins)
	if len(plugins) != 1 {
		t.Errorf("expected 1 bundle plugin, got %d", len(plugins))
	}

	// 更新 plugin_count
	gdb.Model(&bundle).Update("plugin_count", 1)
	var updated PluginBundle
	gdb.First(&updated, bundle.ID)
	if updated.PluginCount != 1 {
		t.Errorf("plugin_count = %d, want 1", updated.PluginCount)
	}
}

// ── PluginInstallation 测试 ──────────────────────────────────────

func TestPluginInstallationCRUD(t *testing.T) {
	cleanup := setupPluginTestDB(t)
	defer cleanup()

	// 创建实例
	instance := Instance{Name: "test-instance"}
	gdb.Create(&instance)

	installation := PluginInstallation{
		InstanceID:    instance.ID,
		Name:          "Test Plugin",
		Slug:          "test-plugin",
		PluginID:      "test-plugin",
		Version:       "1.0.0",
		InstallMode:   "smh",
		Kind:          "memory",
		InstallStatus: PluginInstallNone,
	}
	gdb.Create(&installation)

	// 更新状态
	gdb.Model(&installation).Update("install_status", PluginInstallSuccess)

	var found PluginInstallation
	gdb.First(&found, installation.ID)
	if found.InstallStatus != PluginInstallSuccess {
		t.Errorf("install_status = %d, want %d", found.InstallStatus, PluginInstallSuccess)
	}

	// 按实例查询
	var instPlugins []PluginInstallation
	gdb.Where("instance_id = ?", instance.ID).Find(&instPlugins)
	if len(instPlugins) != 1 {
		t.Errorf("expected 1 installation, got %d", len(instPlugins))
	}
}

// ── PluginDistributionTask/Record 测试 ───────────────────────────

func TestPluginDistributionTaskRecord(t *testing.T) {
	cleanup := setupPluginTestDB(t)
	defer cleanup()

	plugin := Plugin{Slug: "dist-test", Name: "Dist Test", Version: "1.0.0", VersionMajor: 1}
	gdb.Create(&plugin)

	task := PluginDistributionTask{
		PluginDBID: plugin.ID,
		Version:    "1.0.0",
		OperatorID: 1,
		Total:      3,
		Status:     "running",
	}
	gdb.Create(&task)

	// 创建记录
	for i := uint(1); i <= 3; i++ {
		record := PluginDistributionRecord{
			TaskID:      task.ID,
			PluginDBID:  plugin.ID,
			InstanceID:  i,
			InstanceCID: "ins-test",
			Version:     "1.0.0",
			Status:      "pending",
		}
		gdb.Create(&record)
	}

	// 更新记录状态
	gdb.Model(&PluginDistributionRecord{}).Where("task_id = ? AND instance_id = ?", task.ID, 1).
		Update("status", "success")
	gdb.Model(&PluginDistributionRecord{}).Where("task_id = ? AND instance_id = ?", task.ID, 2).
		Update("status", "success")
	gdb.Model(&PluginDistributionRecord{}).Where("task_id = ? AND instance_id = ?", task.ID, 3).
		Updates(map[string]interface{}{"status": "failed", "error": "timeout"})

	// 统计
	var successCount, failedCount int64
	gdb.Model(&PluginDistributionRecord{}).Where("task_id = ? AND status = ?", task.ID, "success").Count(&successCount)
	gdb.Model(&PluginDistributionRecord{}).Where("task_id = ? AND status = ?", task.ID, "failed").Count(&failedCount)

	if successCount != 2 {
		t.Errorf("success count = %d, want 2", successCount)
	}
	if failedCount != 1 {
		t.Errorf("failed count = %d, want 1", failedCount)
	}

	// 更新 task 统计
	gdb.Model(&task).Updates(map[string]interface{}{
		"success": int(successCount),
		"failed":  int(failedCount),
		"status":  "completed",
	})

	var updatedTask PluginDistributionTask
	gdb.First(&updatedTask, task.ID)
	if updatedTask.Success != 2 || updatedTask.Failed != 1 || updatedTask.Status != "completed" {
		t.Errorf("task stats: success=%d failed=%d status=%s, want 2/1/completed",
			updatedTask.Success, updatedTask.Failed, updatedTask.Status)
	}
}

// ── PublicPlugin 测试 ────────────────────────────────────────────

func TestPublicPluginCRUD(t *testing.T) {
	cleanup := setupPluginTestDB(t)
	defer cleanup()

	pp := PublicPlugin{
		Name:           "Public Test",
		Slug:           "public-test",
		PluginID:       "public-test",
		Version:        "1.0.0",
		Description:    "A public plugin",
		NpmPackage:     "@test/public-test",
		TotalDownloads: 100,
		TotalFavorites: 10,
	}
	gdb.Create(&pp)

	var found PublicPlugin
	gdb.First(&found, pp.ID)
	if found.NpmPackage != "@test/public-test" {
		t.Errorf("npm_package = %q, want %q", found.NpmPackage, "@test/public-test")
	}
	if found.TotalDownloads != 100 {
		t.Errorf("total_downloads = %d, want 100", found.TotalDownloads)
	}
}

// ==================== ResolvePluginUninstallFailedStatus 测试 ====================

func TestResolvePluginUninstallFailedStatus_NoRecord(t *testing.T) {
	cleanup := setupPluginTestDB(t)
	defer cleanup()

	// 没有成功记录 → 返回 "failed"
	status := ResolvePluginUninstallFailedStatus(
		context.Background(), 999, []uint{1}, "2.0.0")
	if status != "failed" {
		t.Fatalf("期望 failed，实际 %q", status)
	}
}

func TestResolvePluginUninstallFailedStatus_SameVersion(t *testing.T) {
	cleanup := setupPluginTestDB(t)
	defer cleanup()

	// 创建一条下发成功记录，版本与最新版本相同
	gdb.Create(&PluginDistributionRecord{
		TaskID:     1,
		PluginDBID: 10,
		InstanceID: 100,
		Version:    "1.0.0",
		Status:     "success",
		Type:       "distribute",
	})

	// 已安装版本 == 最新版本 → 返回 "failed"
	status := ResolvePluginUninstallFailedStatus(
		context.Background(), 100, []uint{10}, "1.0.0")
	if status != "failed" {
		t.Fatalf("期望 failed，实际 %q", status)
	}
}

func TestResolvePluginUninstallFailedStatus_OldVersion(t *testing.T) {
	cleanup := setupPluginTestDB(t)
	defer cleanup()

	// 创建一条下发成功记录，版本为旧版本
	gdb.Create(&PluginDistributionRecord{
		TaskID:     1,
		PluginDBID: 10,
		InstanceID: 100,
		Version:    "1.0.0",
		Status:     "success",
		Type:       "distribute",
	})

	// 已安装版本 != 最新版本 → 返回 "uninstall_failed_old"
	status := ResolvePluginUninstallFailedStatus(
		context.Background(), 100, []uint{10}, "2.0.0")
	if status != "uninstall_failed_old" {
		t.Fatalf("期望 uninstall_failed_old，实际 %q", status)
	}
}

func TestResolvePluginUninstallFailedStatus_UninstallRecordIgnored(t *testing.T) {
	cleanup := setupPluginTestDB(t)
	defer cleanup()

	// 创建一条卸载类型的 success 记录（应被忽略）
	gdb.Create(&PluginDistributionRecord{
		TaskID:     1,
		PluginDBID: 10,
		InstanceID: 100,
		Version:    "1.0.0",
		Status:     "success",
		Type:       "uninstall",
	})

	// 无 distribute success 记录 → 返回 "failed"
	status := ResolvePluginUninstallFailedStatus(
		context.Background(), 100, []uint{10}, "2.0.0")
	if status != "failed" {
		t.Fatalf("期望 failed，实际 %q", status)
	}
}

func TestPluginInstallStatusCase_SQLInjection(t *testing.T) {
	// 正常版本号应正常返回
	result, err := PluginInstallStatusCase("1.2.3")
	if err != nil {
		t.Fatalf("正常版本号不应报错: %v", err)
	}
	if !strings.Contains(result, "1.2.3") {
		t.Fatalf("正常版本号未包含在结果中: %s", result)
	}

	// 恶意版本号应返回 error
	if _, err := PluginInstallStatusCase("1.0.0' OR '1'='1"); err == nil {
		t.Fatal("恶意版本号应返回 error，但未发生")
	}
}

func TestSanitizeVersion(t *testing.T) {
	// 合法版本号
	result, err := sanitizeVersion("1.2.3")
	if err != nil || result != "1.2.3" {
		t.Fatalf("期望 1.2.3, nil; 实际=%q, %v", result, err)
	}
	// 带前导零返回规范化后的值
	result, err = sanitizeVersion("01.02.03")
	if err != nil || result != "1.2.3" {
		t.Fatalf("期望 1.2.3, nil; 实际=%q, %v", result, err)
	}
	// 格式不合法
	if _, err := sanitizeVersion("1.2"); err == nil {
		t.Fatal("期望返回 error")
	}
	if _, err := sanitizeVersion("a.b.c"); err == nil {
		t.Fatal("非数字应返回 error")
	}
	if _, err := sanitizeVersion("1.0.0' OR '1'='1"); err == nil {
		t.Fatal("注入字符应返回 error")
	}
}

func TestGetPluginVisibilityGroupIDs(t *testing.T) {
	cleanup := setupPluginTestDB(t)
	defer cleanup()

	gdb.AutoMigrate(&PluginVisibilityGroup{})
	gdb.Create(&PluginVisibilityGroup{PluginID: 1, GroupID: 10})
	gdb.Create(&PluginVisibilityGroup{PluginID: 1, GroupID: 20})

	result, err := GetPluginVisibilityGroupIDs(context.Background(), []uint{1})
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(result[1]) != 2 {
		t.Fatalf("期望 2 个 group ID，实际=%d", len(result[1]))
	}
}

func TestSetPluginVisibility(t *testing.T) {
	cleanup := setupPluginTestDB(t)
	defer cleanup()

	gdb.AutoMigrate(&PluginVisibilityGroup{})
	p := Plugin{Slug: "vis-test", Name: "test", Version: "1.0.0", VersionMajor: 1, VisibilityType: "all"}
	gdb.Create(&p)

	err := SetPluginVisibility(gdb, p.ID, "group", []uint{1, 2, 3})
	if err != nil {
		t.Fatalf("设置可见性失败: %v", err)
	}

	var updated Plugin
	gdb.First(&updated, p.ID)
	if updated.VisibilityType != "group" {
		t.Fatalf("期望 visibility_type=group，实际=%s", updated.VisibilityType)
	}

	idsMap, _ := GetPluginVisibilityGroupIDs(context.Background(), []uint{p.ID})
	ids := idsMap[p.ID]
	if len(ids) != 3 {
		t.Fatalf("期望 3 个分组，实际=%d", len(ids))
	}
}

func TestCopyPluginVisibility(t *testing.T) {
	cleanup := setupPluginTestDB(t)
	defer cleanup()

	gdb.AutoMigrate(&PluginVisibilityGroup{})
	old := Plugin{Slug: "copy-vis", Name: "old", Version: "1.0.0", VersionMajor: 1, VisibilityType: "group"}
	gdb.Create(&old)
	gdb.Create(&PluginVisibilityGroup{PluginID: old.ID, GroupID: 5})
	gdb.Create(&PluginVisibilityGroup{PluginID: old.ID, GroupID: 6})

	new := Plugin{Slug: "copy-vis", Name: "new", Version: "2.0.0", VersionMajor: 2}
	gdb.Create(&new)

	err := CopyPluginVisibility(gdb, "copy-vis", new.ID)
	if err != nil {
		t.Fatalf("复制可见性失败: %v", err)
	}

	var updated Plugin
	gdb.First(&updated, new.ID)
	if updated.VisibilityType != "group" {
		t.Fatalf("期望 visibility_type=group，实际=%s", updated.VisibilityType)
	}

	idsMap2, _ := GetPluginVisibilityGroupIDs(context.Background(), []uint{new.ID})
	ids := idsMap2[new.ID]
	if len(ids) != 2 {
		t.Fatalf("期望 2 个分组，实际=%d", len(ids))
	}
}

func TestCleanupPluginVisibilityByGroupID(t *testing.T) {
	cleanup := setupPluginTestDB(t)
	defer cleanup()

	gdb.AutoMigrate(&PluginVisibilityGroup{})
	gdb.Create(&PluginVisibilityGroup{PluginID: 1, GroupID: 99})

	CleanupPluginVisibilityByGroupID(gdb, 99)

	var count int64
	gdb.Model(&PluginVisibilityGroup{}).Where("group_id = ?", 99).Count(&count)
	if count != 0 {
		t.Fatalf("期望删除完毕，实际 count=%d", count)
	}
}

func TestCleanupPluginVisibilityByPluginID(t *testing.T) {
	cleanup := setupPluginTestDB(t)
	defer cleanup()

	gdb.AutoMigrate(&PluginVisibilityGroup{})
	gdb.Create(&PluginVisibilityGroup{PluginID: 88, GroupID: 1})

	CleanupPluginVisibilityByPluginID(gdb, 88)

	var count int64
	gdb.Model(&PluginVisibilityGroup{}).Where("plugin_id = ?", 88).Count(&count)
	if count != 0 {
		t.Fatalf("期望删除完毕，实际 count=%d", count)
	}
}

func TestIsGroupUsedByPluginVisibility(t *testing.T) {
	cleanup := setupPluginTestDB(t)
	defer cleanup()

	gdb.AutoMigrate(&PluginVisibilityGroup{})
	gdb.Create(&PluginVisibilityGroup{PluginID: 1, GroupID: 77})

	used, err := IsGroupUsedByPluginVisibility(context.Background(), 77)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if !used {
		t.Fatal("期望 group 77 被使用")
	}
	used2, _ := IsGroupUsedByPluginVisibility(context.Background(), 999)
	if used2 {
		t.Fatal("期望 group 999 未被使用")
	}
}
