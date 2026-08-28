package model

import (
	"gorm.io/gorm"
	"log/slog"
	"time"
)

// MigrateUserGroupClosureAndFullPath 一次性数据迁移：
//  0. SQLite 后端先删除 0414 遗留的 idx_ug_identifier_name 唯一索引
//     （0506 migration 仅适用于 MySQL，SQLite 历史库会残留导致同名子组换父冲突，
//     报 "UNIQUE constraint failed: user_groups.identifier, user_groups.name"）
//     这一步在前面执行，与下方 closure / full_path 的"一次性"门槛解耦：
//     测试环境可能已经有 group_closure 数据但仍残留旧索引，放在 gate 之后会被跳过；
//     放最前面统一兜底，DROP IF EXISTS 幂等，对生产无副作用。
//  1. 为每个 user_groups 行写入 closure 自指 (a=d=id, depth=0)
//  2. 为根组 (parent_id=0) 且 full_path 为空的行 backfill full_path=name
//
// 1) 与 2) 的触发条件 (全部满足才执行)：
//   - count(group_closure) == 0
//   - count(user_groups) > 0
//
// 全新部署 / 已迁移过 / 没有任何分组 → 跳过 1)/2)，但 0) 仍执行（无副作用）。
//
// 函数不抛错中断启动；任何失败只打 ERROR 日志。调用方位于 InitDB 中
// SeedChannels 等 seeder 之后，由 db:seed 分布式锁保证多实例串行化。
func MigrateUserGroupClosureAndFullPath(tx *gorm.DB) {
	if tx == nil || gdb == nil {
		return
	}
	// 0) SQLite 后端：删除 0414 遗留的 idx_ug_identifier_name 唯一索引。
	//    放在 closure / full_path 的 gate 之前，确保即使该 gate 短路也能修复线上的索引残留。
	//    MySQL 后端不在此处处理（0506 SQL 已用 INFORMATION_SCHEMA 路径正确清理）。
	if gdb.Dialector.Name() == "sqlite" {
		const legacyIdx = "idx_ug_identifier_name"
		if err := tx.Exec("DROP INDEX IF EXISTS " + legacyIdx).Error; err != nil {
			slog.Warn("[MigrateUserGroup] DROP "+legacyIdx+" 失败（可忽略）", "err", err)
		} else {
			slog.Info("[MigrateUserGroup] 已尝试 DROP " + legacyIdx + "（SQLite，如存在）")
		}
	}

	var closureCount int64
	if err := tx.Table("group_closure").Count(&closureCount).Error; err != nil {
		slog.Error("[MigrateUserGroup] count group_closure 失败", "error", err)
		return
	}
	if closureCount > 0 {
		return // 已迁移过或本来就有数据
	}
	var groupCount int64
	if err := tx.Model(&UserGroup{}).Count(&groupCount).Error; err != nil {
		slog.Error("[MigrateUserGroup] count user_groups 失败", "error", err)
		return
	}
	if groupCount == 0 {
		return // 全新部署
	}

	slog.Info("[MigrateUserGroup] 启动一次性存量迁移",
		"user_groups", groupCount, "group_closure", closureCount)
	start := time.Now()

	// 1) 为每个分组写入闭包自指行
	if err := tx.Exec(`
		INSERT INTO group_closure (identifier, ancestor_id, descendant_id, depth)
		SELECT identifier, id, id, 0
		FROM user_groups
	`).Error; err != nil {
		slog.Error("[MigrateUserGroup] backfill group_closure 失败", "error", err)
		return
	}

	// 2) 根组 full_path = name (仅处理 parent_id=0 + full_path 为空的行)
	if err := tx.Exec(`
		UPDATE user_groups
		SET full_path = name
		WHERE parent_id = 0
		  AND full_path = ''
	`).Error; err != nil {
		slog.Error("[MigrateUserGroup] backfill user_groups.full_path 失败", "error", err)
		return
	}

	slog.Info("[MigrateUserGroup] 存量迁移完成", "elapsed", time.Since(start))
}
