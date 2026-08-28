package task

import (
	"context"
	"time"

	"hatchery/controller"
)

// agent-command dispatch reconcile 任务：定期对未终态 dispatch 调用 recalcDispatchStatus，
// 兜底事件式状态推进可能漏掉的边界场景。
//
// 设计要点：
//   - PerTenant=true：scheduler 遍历所有租户 + 注入完整 TenantSnapshot，
//     下游 controller.RunAgentCommandReconcileOnce 内的 model.DB(ctx) 走
//     identifier 回调按当前租户过滤。
//   - NeedDistLock=true：多实例部署同一租户由分布式锁仲裁，避免重复扫描。
//   - Interval=60s：dispatch 状态变化是稀疏事件，不需要高频；事件式更新
//     已经覆盖 99% 路径，本任务仅作兜底。
//   - InitialDelay=15s：服务起来 15s 后开始首轮，让 agent-command-poller
//     先跑一轮（5s 间隔）。
func init() {
	RegisterTask(TaskDef{
		Name:         "agent-command-dispatch-reconcile",
		Interval:     60 * time.Second,
		RunFunc:      runAgentCommandDispatchReconcile,
		NeedDistLock: true,
		PerTenant:    true,
		InitialDelay: 15 * time.Second,
	})
}

func runAgentCommandDispatchReconcile(ctx context.Context) {
	controller.RunAgentCommandDispatchReconcileOnce(ctx)
}
