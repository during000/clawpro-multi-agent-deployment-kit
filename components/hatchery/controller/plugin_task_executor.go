package controller

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"

	"hatchery/model"
)

// PluginTaskExecutor 定义插件任务中每个实例的执行逻辑。
// 参数：
//   - ctx: 异步上下文（DetachContext 后的）

// pluginDistributeWG 可选的 WaitGroup，用于测试等待插件任务的后台 goroutine 完成。
// 仅在测试中设置，生产环境为 nil。
var pluginDistributeWG *sync.WaitGroup
//   - record: 当前要处理的下发/卸载记录
//
// 返回值：
//   - error: nil 表示成功，非 nil 表示失败（error.Error() 会写入 record.Error）
type PluginTaskExecutor func(ctx context.Context, record model.PluginDistributionRecord) error

// PluginTaskConfig 异步执行插件任务的配置
type PluginTaskConfig struct {
	Ctx     context.Context                   // 异步上下文（已 DetachContext）
	Task    model.PluginDistributionTask      // 任务对象
	Records []model.PluginDistributionRecord  // 所有待处理记录
	Lock    *model.DistLock                   // 分布式锁，任务完成后释放
	Slug    string                            // 插件 slug（仅用于日志）

	// OnSuccess 可选回调，每条 record 执行成功后调用。
	// 注意：该回调在各 record 的独立 goroutine 中调用，实现方须自行保证并发安全。
	OnSuccess func(ctx context.Context, record model.PluginDistributionRecord)

	// OnFailed 可选回调，每条 record 执行失败时调用，返回应写入的 status 字符串。
	// 用于区分不同类型的失败（如 skipped、failed）。
	// 若为 nil，默认写入 "failed"。
	OnFailed func(ctx context.Context, record model.PluginDistributionRecord) string

	// OnComplete 可选回调，所有 record 处理完成后调用（在更新 Task 统计之后、释放锁之前）。
	// 参数 successCount 和 failedCount 为本次任务的成功/失败计数。
	// 用于批量操作（如一次性递增 distribute_count）。
	OnComplete func(ctx context.Context, successCount, failedCount int)
}

// executePluginTaskAsync 异步并发执行插件任务（下发/卸载共用）。
//
// 流程：
//  1. 从配置读取并发数（默认 100）
//  2. 并发执行 executor，每个 record 独立 goroutine
//  3. 更新 record 状态（success/failed）
//  4. 完成后更新 task 统计
//  5. 调用 OnComplete 回调（如有）
//  6. 释放分布式锁
//
// 该函数在独立的 goroutine 中运行，不阻塞 HTTP handler。
func executePluginTaskAsync(cfg PluginTaskConfig, executor PluginTaskExecutor) {
	if pluginDistributeWG != nil {
		pluginDistributeWG.Add(1)
	}
	go func() {
		if pluginDistributeWG != nil {
			defer pluginDistributeWG.Done()
		}
		defer cfg.Lock.Release()

		// 从配置读取并发数
		siteConfig := model.GetSiteConfig(cfg.Ctx)
		maxConcurrency := siteConfig.SkillDistributeConcurrency
		if maxConcurrency <= 0 {
			maxConcurrency = 100
		}
		sem := make(chan struct{}, maxConcurrency)
		var wg sync.WaitGroup
		var successCount int64
		var failedCount int64

		for _, rec := range cfg.Records {
			wg.Add(1)
			go func(record model.PluginDistributionRecord) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				defer func() {
					if r := recover(); r != nil {
						slog.Error("插件任务执行 panic",
							"task_id", cfg.Task.ID,
							"instance_id", record.InstanceCID,
							"slug", cfg.Slug,
							"panic", r)
						failedStatus := "failed"
						if cfg.OnFailed != nil {
							failedStatus = cfg.OnFailed(cfg.Ctx, record)
						}
						if dbErr := model.DB(cfg.Ctx).Model(&record).Updates(map[string]interface{}{
							"status": failedStatus,
							"error":  fmt.Sprintf("panic: %v", r),
						}).Error; dbErr != nil {
							slog.Error("更新 record 状态失败(panic恢复)", "record_id", record.ID, "error", dbErr)
						}
						atomic.AddInt64(&failedCount, 1)
					}
				}()

				err := executor(cfg.Ctx, record)

				if err != nil {
					slog.Error("插件任务执行失败",
						"task_id", cfg.Task.ID,
						"task_type", cfg.Task.Type,
						"instance_id", record.InstanceCID,
						"slug", cfg.Slug,
						"error", err)
					failedStatus := "failed"
					if cfg.OnFailed != nil {
						failedStatus = cfg.OnFailed(cfg.Ctx, record)
					}
					if dbErr := model.DB(cfg.Ctx).Model(&record).Updates(map[string]interface{}{
						"status": failedStatus,
						"error":  err.Error(),
					}).Error; dbErr != nil {
						slog.Error("更新 record 状态失败", "record_id", record.ID, "error", dbErr)
					}
					atomic.AddInt64(&failedCount, 1)
				} else {
					if dbErr := model.DB(cfg.Ctx).Model(&record).Update("status", "success").Error; dbErr != nil {
						slog.Error("更新 record 状态失败", "record_id", record.ID, "error", dbErr)
					}
					atomic.AddInt64(&successCount, 1)
					if cfg.OnSuccess != nil {
						cfg.OnSuccess(cfg.Ctx, record)
					}
				}
			}(rec)
		}

		wg.Wait()

		// 一次性写入 Task 统计
		sc := int(atomic.LoadInt64(&successCount))
		fc := int(atomic.LoadInt64(&failedCount))
		if dbErr := model.DB(cfg.Ctx).Model(&cfg.Task).Updates(map[string]interface{}{
			"status":  "completed",
			"success": sc,
			"failed":  fc,
		}).Error; dbErr != nil {
			slog.Error("更新 task 统计失败", "task_id", cfg.Task.ID, "error", dbErr)
		}

		// 任务完成后回调（如批量递增 distribute_count）
		if cfg.OnComplete != nil {
			cfg.OnComplete(cfg.Ctx, sc, fc)
		}

		slog.Info("插件任务完成",
			"task_id", cfg.Task.ID,
			"task_type", cfg.Task.Type,
			"slug", cfg.Slug,
			"success", sc,
			"failed", fc)
	}()
}
