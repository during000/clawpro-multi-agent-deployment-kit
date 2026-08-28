package task

import (
	"context"
	"log/slog"
	"sync"

	"hatchery/controller"
	"hatchery/model"
)

// vncMigrateDone per-tenant 迁移状态：0=未完成/失败，1=正在执行，2=已成功。
var vncMigrateDone sync.Map // key: identifier(string), value: *int32

func getVNCMigrateState(identifier string) *int32 {
	val, _ := vncMigrateDone.LoadOrStore(identifier, new(int32))
	return val.(*int32)
}

func init() {
	RegisterTask(TaskDef{
		Name:         "vnc-sg-migration",
		Interval:     0, // 一次性
		RunFunc:      runVNCSGMigration,
		NeedDistLock: true,
		PerTenant:    true,
	})
}

// runVNCSGMigration 对每个 RuleSet 触发一次规则刷新，让必需规则（含 VNC 白名单）
// 被合入并扇出到 ACTIVE 池。
func runVNCSGMigration(ctx context.Context) {
	identifier := model.CurrentIdentifier(ctx)
	state := getVNCMigrateState(identifier)

	config := model.GetSiteConfig(ctx)
	if !config.BrowserVNCEnable {
		return
	}
	ready, rsCount, activeSGCount, err := controller.HasSGPoolReady(ctx)
	if err != nil {
		slog.Warn("[BrowserVNC] 检查 SG 就绪状态失败，跳过启动时安全组迁移", "error", err)
		return
	}
	if !ready {
		slog.Info("[BrowserVNC] 无有效安全组（RuleSet 或 ACTIVE SG 缺失），跳过启动时安全组迁移",
			"ruleset_count", rsCount, "active_sg_count", activeSGCount)
		return
	}

	// CAS: 0→1 表示开始执行，已完成(2)或正在执行(1)则跳过
	if !casInt32(state, 0, 1) {
		return
	}
	slog.Info("[BrowserVNC] 启动时安全组白名单迁移开始", "identifier", identifier)
	if err := controller.RefreshAllRuleSetsForRequiredRules(ctx); err != nil {
		slog.Warn("[BrowserVNC] 启动时安全组白名单迁移失败，将在用户访问时重试", "error", err)
		storeInt32(state, 0)
	} else {
		slog.Info("[BrowserVNC] 启动时安全组白名单迁移成功", "identifier", identifier)
		storeInt32(state, 2)
	}
}
