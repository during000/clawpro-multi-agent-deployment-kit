package controller

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"

	"hatchery/model"
)

// SkillTaskExecutor 定义技能任务中每个实例的执行逻辑。
// 返回 nil 表示成功；非 nil 错误会写入 record.Error。
type SkillTaskExecutor func(ctx context.Context, record model.SkillDistributionRecord) error

// SkillTaskConfig 定义技能任务执行配置，供同步和异步入口共用。
type SkillTaskConfig struct {
	Ctx      context.Context
	Task     model.SkillDistributionTask
	Records  []model.SkillDistributionRecord
	Lock     *model.DistLock
	Slug     string
	SkillIDs []uint

	// OnSuccess 可选回调：单条记录成功落库后调用。
	// 并发执行时回调实现必须保证线程安全。
	OnSuccess func(ctx context.Context, record model.SkillDistributionRecord)

	// OnFailed 可选回调：单条记录失败时返回最终状态。
	// 未设置时使用 RecordStatusFailed。
	OnFailed func(ctx context.Context, record model.SkillDistributionRecord) string

	// OnComplete 可选回调：任务统计落库后、分布式锁释放前调用。
	OnComplete func(ctx context.Context, successCount, failedCount int)
}

// executeSkillTaskAsync 异步执行技能任务并立即返回。
func executeSkillTaskAsync(cfg SkillTaskConfig, executor SkillTaskExecutor) {
	if skillDistributeWG != nil {
		skillDistributeWG.Add(1)
	}
	go func() {
		if skillDistributeWG != nil {
			defer skillDistributeWG.Done()
		}
		_ = executeSkillTask(cfg, executor)
	}()
}

// executeSkillTask 同步执行技能任务，持久化最终状态并释放分布式锁。
func executeSkillTask(cfg SkillTaskConfig, executor SkillTaskExecutor) error {
	if cfg.Lock != nil {
		defer cfg.Lock.Release()
	}

	maxConcurrency := model.GetSiteConfig(cfg.Ctx).SkillDistributeConcurrency
	if maxConcurrency <= 0 {
		maxConcurrency = 100
	}
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	var successCount int64
	var failedCount int64
	var firstErr error
	var firstErrOnce sync.Once
	setFirstErr := func(err error) {
		if err != nil {
			firstErrOnce.Do(func() {
				firstErr = err
			})
		}
	}
	failedStatus := func(record model.SkillDistributionRecord) string {
		if cfg.OnFailed != nil {
			return cfg.OnFailed(cfg.Ctx, record)
		}
		return model.RecordStatusFailed
	}

	for _, rec := range cfg.Records {
		wg.Add(1)
		go func(record model.SkillDistributionRecord) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			err := executor(cfg.Ctx, record)
			if err != nil {
				slog.Error("[SkillTask] 技能任务执行失败",
					"task_id", cfg.Task.ID,
					"task_type", cfg.Task.Type,
					"instance_id", record.InstanceCID,
					"slug", cfg.Slug,
					"error", err)
				status := failedStatus(record)
				dbErr := model.DB(cfg.Ctx).Model(&record).Updates(map[string]interface{}{
					"status": status,
					"error":  err.Error(),
				}).Error
				if dbErr != nil {
					slog.Error("[SkillTask] 更新 record 状态失败", "record_id", record.ID, "error", dbErr)
				}
				setFirstErr(err)
				setFirstErr(dbErr)
				atomic.AddInt64(&failedCount, 1)
				return
			}

			dbErr := model.DB(cfg.Ctx).Model(&record).Update("status", model.RecordStatusSuccess).Error
			if dbErr != nil {
				slog.Error("[SkillTask] 更新 record 状态失败", "record_id", record.ID, "error", dbErr)
				setFirstErr(dbErr)
				terminalErr := model.DB(cfg.Ctx).Model(&record).Updates(map[string]interface{}{
					"status": failedStatus(record),
					"error":  dbErr.Error(),
				}).Error
				if terminalErr != nil {
					slog.Error("[SkillTask] 写入 record 失败终态失败", "record_id", record.ID, "error", terminalErr)
					setFirstErr(terminalErr)
				}
				atomic.AddInt64(&failedCount, 1)
				return
			}
			atomic.AddInt64(&successCount, 1)
			if cfg.OnSuccess != nil {
				cfg.OnSuccess(cfg.Ctx, record)
			}
		}(rec)
	}
	wg.Wait()

	sc := int(atomic.LoadInt64(&successCount))
	fc := int(atomic.LoadInt64(&failedCount))
	dbErr := model.DB(cfg.Ctx).Model(&cfg.Task).Updates(map[string]interface{}{
		"status":  model.TaskStatusCompleted,
		"success": sc,
		"failed":  fc,
	}).Error
	if dbErr != nil {
		slog.Error("[SkillTask] 更新 task 统计失败", "task_id", cfg.Task.ID, "error", dbErr)
		setFirstErr(dbErr)
	}

	if cfg.OnComplete != nil {
		cfg.OnComplete(cfg.Ctx, sc, fc)
	}
	slog.Info("[SkillTask] 技能任务完成",
		"task_id", cfg.Task.ID,
		"task_type", cfg.Task.Type,
		"slug", cfg.Slug,
		"success", sc,
		"failed", fc)
	return firstErr
}
