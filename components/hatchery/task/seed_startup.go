package task

import (
	"context"
	"log/slog"

	"hatchery/controller"
	"hatchery/model"
)

func init() {
	// 启动期 Seed（纯 DB 操作）：每次启动为每个租户补齐预置数据（幂等）。
	// PriorityStartup 保证在所有普通定时任务之前执行完毕。
	RegisterTask(TaskDef{
		Name:         "seed-startup",
		Interval:     0,
		RunFunc:      runSeedStartup,
		PerTenant:    true,
		NeedDistLock: true,
		Priority:     PriorityStartup,
	})

	// 启动期一次性迁移和恢复（纯 DB 操作）。
	RegisterTask(TaskDef{
		Name:         "startup-migrations",
		Interval:     0,
		RunFunc:      runStartupMigrations,
		PerTenant:    true,
		NeedDistLock: true,
		Priority:     PriorityStartup,
	})

	// 候选镜像初始化（调用 CVM DescribeImages API，可能耗时）。
	// 不设 PriorityStartup：有外部 API 调用，不应阻塞定时任务启动。
	RegisterTask(TaskDef{
		Name:         "seed-available-images",
		Interval:     0,
		RunFunc:      runSeedImages,
		PerTenant:    true,
		NeedDistLock: true,
	})
}

// runSeedStartup 为当前租户执行所有幂等 Seed 操作（纯 DB，不含外部 API 调用）。
func runSeedStartup(ctx context.Context) {
	model.RunAllSeeds(ctx, nil) // nil = 不执行镜像初始化
}

// runStartupMigrations 执行启动期的一次性迁移和恢复操作。
func runStartupMigrations(ctx context.Context) {
	// 同步环境变量 MEMORY_TDAI_SUPPORTED_VERSIONS 到已有站点配置
	model.SyncMemoryTDAISupportedVersions(ctx)

	// 分组闭包表自检 + full_path 重算
	model.ReconcileClosure(ctx)
	if err := model.RecomputeFullPathAll(ctx); err != nil {
		slog.Warn("[StartupMigration] RecomputeFullPathAll failed", "err", err)
	}

	// 记忆计划迁移（从旧 status 推导 current_plan）
	model.MigrateMemoryPlanFromStatus(ctx)

	// 用户分组存量数据迁移（闭包行 + full_path 回填）
	model.MigrateUserGroupClosureAndFullPath(model.DB(ctx))

	// 清理 instance_models 软删除残留
	model.CleanInstanceModelSoftDeleteRemnants(model.DB(ctx))

	// 恢复被中断的技能/插件下发任务
	recoverInterruptedSkillTasks(ctx)
	recoverInterruptedPluginTasks(ctx)

	// 恢复被中断的初始技能包/角色插件包安装（Installing/None → Failed，使用户可重试）
	recoverInterruptedSkillInitTasks(ctx)
	recoverInterruptedPluginInitTasks(ctx)

	// 恢复被中断的升级/迁移（重启丢失异步 goroutine → processing → failed）
	recoverInterruptedUpgradeAndMigrate(ctx)

	// 恢复被中断的角色下发任务（updating record → failed）
	recoverInterruptedRoleSyncTasks(ctx)
}

// runSeedImages 通过 CVM API 初始化候选镜像（异步执行，不阻塞启动）。
func runSeedImages(ctx context.Context) {
	model.SeedAvailableImages(ctx, controller.NewCVMClient)
}