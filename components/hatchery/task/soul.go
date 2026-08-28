package task

import (
	"context"
	"log/slog"
	"time"

	"hatchery/controller"
	"hatchery/model"
)

const soulScanInterval = 2 * time.Minute

// SoulDependencies 是 Soul 下发任务的外部依赖接口，测试时可替换。
type SoulDependencies interface {
	SetSoul(ctx context.Context, instancePK uint) error
}

// defaultSoulDeps 是生产环境使用的真实实现。
type defaultSoulDeps struct{}

// SetSoul 调用 controller 层下发 SOUL。
// 反查该实例最新的 updating record ID 一起传给 SetInstanceSoul，
// 便于把 SOUL 侧成败结果写回 record。若无 updating record（老实例 / 没走 apply 流程），
// recordID=0，SetInstanceSoul 内部会跳过 record 写入。
func (defaultSoulDeps) SetSoul(ctx context.Context, instancePK uint) error {
	recordID := controller.FindLatestUpdatingRoleRecordID(ctx, instancePK)
	return controller.SetInstanceSoul(ctx, instancePK, recordID)
}

// 启动 Soul 下发后台任务，每 2 分钟扫描一次。
func init() {
	RegisterTask(TaskDef{
		Name:         "soul-set",
		Interval:     soulScanInterval,
		RunFunc:      runSoulTaskEntry,
		NeedDistLock: true,
		PerTenant:    true,
		InitialDelay: 30 * time.Second,
	})
}

// runSoulTaskEntry 是 scheduler 调用的入口。
func runSoulTaskEntry(ctx context.Context) {
	deps := defaultSoulDeps{}
	doSoulSet(ctx, deps)
}

// doSoulSet 执行一轮 Soul 下发扫描。
func doSoulSet(ctx context.Context, deps SoulDependencies) {
	logger := slog.With("task", "SoulSet")

	var instances []model.Instance
	if err := model.DB(ctx).Where(
		"role_id > 0 AND soul_set_at IS NULL AND runtime_user != '' AND deleted_at IS NULL",
	).Limit(50).Find(&instances).Error; err != nil {
		logger.Error("查询待下发实例失败", "error", err)
		return
	}

	if len(instances) == 0 {
		return
	}

	logger.Info("发现待下发 Soul 的实例", "count", len(instances))

	success := 0
	failed := 0
	for _, inst := range instances {
		if err := deps.SetSoul(ctx, inst.ID); err != nil {
			logger.Warn("Soul 下发失败，下次重试", "instance_pk", inst.ID, "error", err)
			failed++
		} else {
			success++
		}
	}

	logger.Info("Soul 下发轮次完成", "total", len(instances), "success", success, "failed", failed)
}
