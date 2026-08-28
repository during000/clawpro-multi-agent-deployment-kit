package task

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"hatchery/controller"
)

func init() {
	RegisterTask(TaskDef{
		Name:         "instance-adjustment-poller",
		Interval:     5 * time.Second,
		RunFunc:      runInstanceAdjustmentPoller,
		NeedDistLock: true,
		PerTenant:    true,
		InitialDelay: 5 * time.Second,
	})
}

func runInstanceAdjustmentPoller(ctx context.Context) {
	defer func() {
		if recovered := recover(); recovered != nil {
			slog.Error("[instance-adjustment-poller] panic recovered", "error", fmt.Sprintf("%v", recovered))
		}
	}()
	controller.RunInstanceAdjustmentWorkerOnce(ctx)
}
