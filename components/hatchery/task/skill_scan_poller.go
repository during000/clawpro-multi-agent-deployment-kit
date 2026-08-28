package task

import (
	"context"
	"log/slog"
	"time"

	"hatchery/controller"
)

// 启动技能安全扫描后台轮询任务。
// 首次延迟 5 分钟再轮询（API 文档建议上传后 5 分钟再查询），之后每分钟轮询一次。
func init() {
	RegisterTask(TaskDef{
		Name:         "skill-scan-poller",
		Interval:     1 * time.Minute,
		RunFunc:      runSkillScanPoller,
		NeedDistLock: true,
		PerTenant:    true,
		InitialDelay: 5 * time.Minute,
	})
}

// runSkillScanPoller 轮询技能安全扫描结果。
func runSkillScanPoller(ctx context.Context) {
	pollCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := controller.PollSkillSecurityScanResults(pollCtx); err != nil {
		slog.Error("[SkillScan] 后台轮询出错", "error", err)
	}
}
