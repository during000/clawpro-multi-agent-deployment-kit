package task

import (
	"context"
	"log/slog"
	"time"

	"hatchery/common"
	"hatchery/controller"
)

func init() {
	RegisterTask(TaskDef{
		Name:         "sts-refresh",
		Interval:     100 * time.Minute,
		RunFunc:      runSTSRefresh,
		NeedDistLock: true,
		PerTenant:    true,
	})
}

// runSTSRefresh 刷新 STS 临时密钥。
// 若 UIN 为空则跳过（无需 STS）。
func runSTSRefresh(ctx context.Context) {
	uin := common.CVMUinFromCtx(ctx)
	if uin == "" {
		return
	}
	if err := controller.RefreshSTSCredentials(ctx); err != nil {
		slog.Error("STS 临时密钥刷新失败", "error", err)
	}
}
