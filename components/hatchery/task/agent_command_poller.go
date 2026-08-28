package task

import (
	"context"
	"time"

	"hatchery/controller"
)

// agent-command 全局轮询任务：每 5s 把所有 status='in_progress' 的 task
// 拉到 TAT DescribeInvocationTasks，写回终态状态 / exit_code / elapsed_ms，
// 并触发其所属 invocation 的状态聚合。
//
// 设计要点：
//   - PerTenant=true：scheduler 遍历所有租户 + 注入完整 TenantSnapshot，
//     下游 controller.RunAgentCommandPollerOnce 内的 model.DB(ctx) 走
//     identifier 回调按当前租户过滤；getCredential 拿到完整 Uin / AKSK
//     走 STS 路径，TAT 调用正常返回。
//   - NeedDistLock=true：多实例部署同一租户由分布式锁仲裁，避免重复
//     polling TAT、重复刷写 DB（写本身幂等，但浪费 quota）。
//   - InitialDelay=5s：和老实现 agentPollerInterval 对齐；服务起来 5s
//     后开始首轮，避免和 SeedAvailableImages 等启动初始化抢 DB。
func init() {
	RegisterTask(TaskDef{
		Name:         "agent-command-poller",
		Interval:     5 * time.Second,
		RunFunc:      runAgentCommandPoller,
		NeedDistLock: true,
		PerTenant:    true,
		InitialDelay: 5 * time.Second,
	})
}

func runAgentCommandPoller(ctx context.Context) {
	controller.RunAgentCommandPollerOnce(ctx)
}
