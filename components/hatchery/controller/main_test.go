package controller

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"hatchery/common"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// testSafeDB 是 TestMain 初始化的全局安全 DB，供各测试的 cleanup 在恢复时使用，
// 确保恢复后的 gdb 包含所有必要的表，避免泄漏的 goroutine 触发 no-such-table 错误。
var testSafeDB *gorm.DB

// useDBForTestWithSafeRestore 包装 model.UseDBForTest，cleanup 时先恢复 origDB，
// 再强制把 gdb 切到 testSafeDB，防止泄漏的 goroutine 访问已失效的 testDB。
// 同时自动设置 MaxOpenConns(1)，防止 SQLite in-memory DB 多连接问题。
// 用法：t.Cleanup(model.UseDBForTest(db))
func useDBForTestWithSafeRestore(db *gorm.DB) func() {
	// SQLite in-memory DB 是连接私有的，强制单连接防止新连接看到空 DB。
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	origDB := model.UseDBForTest(db)
	return func() {
		origDB()
		if testSafeDB != nil {
			model.SetDBForTest(testSafeDB)
		}
	}
}

// TestMain 是 controller 包的测试入口。
// 必须放在 controller 包内，因为：
//  1. LoadScript 是 controller 包私有变量
//  2. SafeDB 需要在这里初始化，防止多包并行时异步 goroutine 访问无效 DB
func TestMain(m *testing.M) {
	// 确保 LoadScript 不为 nil，避免异步 goroutine 中 panic
	if LoadScript == nil {
		LoadScript = func(name string) (string, error) {
			return "", fmt.Errorf("test: script %s not available", name)
		}
	}

	// 全局打桩 startUpgradePerformFn：
	// startUpgradeForInstance 命中 Started 分支时会启动一个异步 goroutine 执行升级，
	// 这条路径在多个测试里被间接触发（例如 HandleAdminBatchUpgrade 的 OpenClaw 路径、
	// 单实例 handleUpgrade 在 needUpgrade=true 时等）。真实 performUpgrade 会调
	// backupAndUploadToSMH → RunScript → NewTATClient → getCredential →
	// model.GetSiteConfig，最终在测试结束、全局 DB 被切换/关闭后落到一个 nil/失效
	// 的 *gorm.DB 上 panic（gorm.(*DB).getInstance），并跨越测试边界污染后续用例。
	//
	// 单测里我们只关心 startUpgradeForInstance 的同步返回值（Started/AlreadyLatest/Err
	// 以及操作锁是否被设置），不关心异步 goroutine 的实际行为，因此默认让它直接
	// 返回 nil。需要观察 goroutine 入参的用例（如 TestStartUpgradeForInstance_
	// StartedAndOperationLocked）会在用例内部临时替换并在 t.Cleanup 中还原。
	startUpgradePerformFn = func(_ context.Context, _ *model.Instance, _, _ string) error {
		return nil
	}

	// 这是在 init 函数中调用的，因此必须在 TestMain 中预先初始化。
	if common.FixedSnapshot == nil {
		snap := common.TenantSnapshot{
			Domain: "https://example.com", // 默认域名用于测试
		}
		common.FixedSnapshot = &snap
	}

	// 初始化一个包含所有必要 schema 的安全 DB。
	// 多包并行运行时，model/task 包的测试会替换全局 gdb，
	// controller 内的异步 goroutine（如 syncRoleSkillsCosZipKey）仍会继续执行，
	// 需要一个有效的 DB，否则会 panic 或导致测试失败。
	var err error
		testSafeDB, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		if err == nil {
			// SQLite in-memory DB 是连接私有的，必须固定单连接，否则新连接看到空 DB。
			if sqlDB, sqlErr := testSafeDB.DB(); sqlErr == nil {
				sqlDB.SetMaxOpenConns(1)
			}
			// 迁移全部 model，作为安全兜底 DB：任何泄漏的 goroutine 或
			// 被污染的全局 gdb 落到 testSafeDB 时，都不会因缺表而报错。
			// 覆盖分支新增的 group_closure / catalog / project_config_binding /
			// project_member / enterprise_rule / local_instance_rule 等表。
			if migErr := testSafeDB.AutoMigrate(model.AllModelsForTest()...); migErr != nil {
				slog.Error("testSafeDB AutoMigrate failed", "error", migErr)
			}
			model.SetDBForTest(testSafeDB) // TestMain 进程级设置，无需 restore
		}

	os.Exit(m.Run())
}
