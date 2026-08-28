package task

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"hatchery/common"
	"hatchery/controller"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm"
)

// jobCtxRegistry 保存正在执行的 job 的 task context，
// 供 handler 通过 jobLogger / jobCtx 取回，使 handler 内的所有日志都能携带统一的
// request_id / trace_id / job_id 等链路字段，从而支持按 job_id 追溯一次任务的所有日志。
//
// key: jobID (uint), value: context.Context（由 NewTaskContext 创建并附加 job 维度字段）
var jobCtxRegistry sync.Map

// registerJobCtx 在 executeJob 入口注册当前 job 的 ctx；execute 结束后必须调用 unregisterJobCtx 清理。
func registerJobCtx(jobID uint, ctx context.Context) {
	jobCtxRegistry.Store(jobID, ctx)
}

func unregisterJobCtx(jobID uint) {
	jobCtxRegistry.Delete(jobID)
}

// lookupJobCtx 取出当前正在执行的 job 对应的 task ctx，未注册时回退到 context.Background()。
func lookupJobCtx(jobID uint) context.Context {
	if v, ok := jobCtxRegistry.Load(jobID); ok {
		if ctx, ok := v.(context.Context); ok {
			return ctx
		}
	}
	return context.Background()
}

// NonRetryableError 标记不可重试的业务错误（参数非法、状态冲突等）。
// handler 返回此错误时，任务直接标记 FAILED，不走退避重试。
type NonRetryableError struct {
	Msg string
}

func (e *NonRetryableError) Error() string { return e.Msg }

func NewNonRetryableError(msg string) error {
	return &NonRetryableError{Msg: msg}
}

const (
	// 调度周期：每个租户、每个 Pod 每 tick 触发一次 dispatchPendingJobs。
	// 取值原则：
	//   - 太小（如 3s）→ N 租户 × K Pod × QPS 放大 → DB 频繁 GET_LOCK + 空扫
	//   - 太大 → 用户 plan 切换感知延迟增大
	// 10s 兼顾管控延迟可接受度（任务最坏 +9s）与 DB 负载。
	tdaiDispatchInterval  = 10 * time.Second
	tdaiDispatchBatchSize = 20
	tdaiLeaseDuration     = 5 * time.Minute
)

var tdaiDispatcherRunning tenantBool
var tdaiHostname string

// 启动轻量任务流调度器（默认启动）。
func init() {
	// 环境变量开关：DISABLE_TDAI_TASK_ENGINE=true/1/on 时跳过注册
	if v := os.Getenv("DISABLE_TDAI_TASK_ENGINE"); v != "" {
		switch strings.ToLower(v) {
		case "true", "1", "on":
			return
		}
	}

	tdaiHostname, _ = os.Hostname()

	RegisterTask(TaskDef{
		Name:         "tdai-dispatcher",
		Interval:     tdaiDispatchInterval,
		RunFunc:      runTdaiDispatcherEntry,
		NeedDistLock: false, // 内部用 lease 机制
		PerTenant:    true,
		InitialDelay: 5 * time.Second,
	})
}

// runTdaiDispatcherEntry 是 scheduler 调用的入口。
func runTdaiDispatcherEntry(ctx context.Context) {
	safeTdaiDispatch(ctx, tdaiHostname)
}

func safeTdaiDispatch(ctx context.Context, hostname string) {
	identifier := model.CurrentIdentifier(ctx)
	if !tdaiDispatcherRunning.CompareAndSwap(identifier, false, true) {
		return
	}
	defer tdaiDispatcherRunning.Store(identifier, false)

	// 每一轮调度独立 ctx：dispatch 内部及其触发的 executeJob 都会派生自己的 ctx，
	// 但本轮调度的概览日志（扫描数量、抢占失败等）共享这条 trace。
	inCtx := common.WithTaskTrace(common.DetachContext(ctx), "tdai_dispatch_round")

	dispatchPendingJobs(inCtx, hostname)
}

// dispatchPendingJobs 扫描 PENDING 任务，通过条件更新抢占后执行。
//
// 调度协议（解决 DB 连接打满问题，2026-06-17）：
//  1. 无锁预扫：先判断是否有可执行的 PENDING 任务，没有则直接返回。
//     绝大多数租户常态没有任务，避免无谓的 GET_LOCK/RELEASE_LOCK 往返
//     与 advisory lock 钉住的连接占用。
//  2. 仅在"扫描+抢占"期间持锁（毫秒级）：lease + 条件更新已保证 job 维度互斥，
//     调度锁只是降低多实例同时空扫的浪费，无需覆盖 executeJob。
//  3. executeJob 在锁外串行执行：单个任务可能跑数十秒（脚本下发、TAT 等），
//     若在锁内会让 advisory lock 的物理连接被钉住几十秒；移出锁外后
//     钉连接时长降到毫秒级。
//
// SQLite 模式下锁为空操作，不影响单实例部署。
func dispatchPendingJobs(ctx context.Context, hostname string) {
	log := controller.Logger(ctx).With("hostname", hostname)
	now := time.Now()

	// 1) 无锁预扫：仅取 id，确认是否有任务可处理。
	//    Pluck 只读 PRIMARY KEY，命中现有 idx_state_run_at 复合索引，毫秒级返回。
	var probeIDs []uint
	if err := model.DB(ctx).Model(&model.TdaiJob{}).
		Where("state = ? AND run_at <= ?", model.TdaiJobStatePending, now).
		Order("run_at ASC").
		Limit(tdaiDispatchBatchSize).
		Pluck("id", &probeIDs).Error; err != nil {
		log.Error("[TaskEngine] 预扫 PENDING 任务失败", "error", err)
		return
	}
	if len(probeIDs) == 0 {
		// 空轮次静默返回，避免日志噪声。
		return
	}

	// 2) 取锁 + 重扫 + 条件更新抢占。锁与 executeJob 解耦：仅覆盖这一小段。
	//    用闭包包住，确保 lock.Release 与连接归还发生在 executeJob 之前。
	acquiredIDs := acquireDispatchedJobs(ctx, log, hostname, now)
	if len(acquiredIDs) == 0 {
		return
	}

	// 3) 锁外执行：lease_owner / lease_until 已写入 DB，其它实例不会重复抢占。
	for _, id := range acquiredIDs {
		executeJob(ctx, id)
	}
}

// acquireDispatchedJobs 持锁完成"扫描 PENDING + 条件更新抢占"，返回成功抢到的 job ID 列表。
// 抢占成功后立即释放 advisory lock（毫秒级持有），后续 executeJob 在锁外执行。
func acquireDispatchedJobs(ctx context.Context, log *slog.Logger, hostname string, now time.Time) []uint {
	// timeout=0：MySQL GET_LOCK 立即返回（成功或失败），不进入 1s 等待。
	// 多 Pod 部署下，未抢到锁的 Pod 立刻退出而不是 hold 一条连接等 1s，
	// 进一步降低 advisory lock 引入的连接占用。下一轮调度（10s 后）会自然重试。
	lock, err := model.AcquireLock(ctx, "tdai:task-engine:dispatch", 0)
	if err != nil {
		log.Debug("[TaskEngine] 未获取到调度锁，其他实例正在调度，跳过本轮", "error", err)
		return nil
	}
	defer lock.Release()

	leaseUntil := now.Add(tdaiLeaseDuration)
	scanStart := time.Now()

	// 持锁后重扫一次：预扫到现在可能已被其它实例抢占，重扫确保数据最新。
	var jobs []model.TdaiJob
	if err := model.DB(ctx).
		Where("state = ? AND run_at <= ?", model.TdaiJobStatePending, now).
		Order("run_at ASC").
		Limit(tdaiDispatchBatchSize).
		Find(&jobs).Error; err != nil {
		log.Error("[TaskEngine] 扫描 PENDING 任务失败", "error", err)
		return nil
	}
	if len(jobs) == 0 {
		return nil
	}

	log.Info("[TaskEngine] 本轮扫描到 PENDING 任务",
		"count", len(jobs),
		"scan_cost", time.Since(scanStart).String(),
	)

	var (
		acquiredIDs []uint
		skipped     int
	)
	for _, job := range jobs {
		// 条件更新抢占：state 必须仍为 PENDING
		result := model.DB(ctx).Model(&model.TdaiJob{}).
			Where("id = ? AND state = ?", job.ID, model.TdaiJobStatePending).
			Updates(map[string]any{
				"state":       model.TdaiJobStateRunning,
				"lease_owner": hostname,
				"lease_until": &leaseUntil,
				"attempt":     job.Attempt + 1,
			})
		if result.RowsAffected == 0 {
			skipped++
			log.Info("[TaskEngine] 任务已被其他实例抢占，跳过",
				"job_id", job.ID,
				"job_type", job.JobType,
				"instance_id", job.InstanceID,
			)
			continue
		}

		acquiredIDs = append(acquiredIDs, job.ID)
		log.Info("[TaskEngine] 抢占任务成功",
			"job_id", job.ID,
			"job_type", job.JobType,
			"instance_id", job.InstanceID,
			"attempt", job.Attempt+1,
			"lease_until", leaseUntil,
		)
	}

	log.Info("[TaskEngine] 本轮抢占结束",
		"scanned", len(jobs),
		"acquired", len(acquiredIDs),
		"skipped", skipped,
		"round_cost", time.Since(scanStart).String(),
	)
	return acquiredIDs
}

// executeJob 加载任务并执行对应 handler。
// 为本次执行建立独立的 task ctx，注入 job_id / job_type / instance_id / attempt 等链路字段，
// 注册到 jobCtxRegistry 以便 handler 通过 jobLogger / jobCtx 取回，从而保证一次 job 执行
// 期间所有日志（含 handler、DB、SDK 调用）都能通过同一 request_id / job_id 关联。
func executeJob(dispatchCtx context.Context, jobID uint) {
	// 整个 executeJob 只用一份 task ctx，保证"开始执行→加载→handler→标记成功/失败"
	// 都串联在同一 request_id 下，便于按 request_id 一次拉出整段 trace。
	ctx := common.WithTaskTrace(common.DetachContext(dispatchCtx), "tdai_job_exec")
	registerJobCtx(jobID, ctx)
	defer unregisterJobCtx(jobID)

	log := controller.Logger(ctx).With("job_id", jobID)
	log.Info("[TaskEngine] 开始执行任务")

	var job model.TdaiJob
	if err := model.DB(ctx).First(&job, jobID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("[TaskEngine] 任务不存在，可能已被取消或删除")
		} else {
			log.Error("[TaskEngine] 加载任务失败", "error", err)
		}
		return
	}

	// 加载到 job 后，给 logger 补齐完整 job 维度字段（job_type/instance_id/attempt）
	log = controller.Logger(ctx).With(
		"job_id", job.ID,
		"job_type", job.JobType,
		"instance_id", job.InstanceID,
		"attempt", job.Attempt,
	)
	log.Info("[TaskEngine] 任务加载完成，准备分发到 handler",
		"current_step", job.CurrentStep,
		"progress", job.Progress,
		"max_attempts", job.MaxAttempts,
	)

	handlerStart := time.Now()
	var err error
	switch job.JobType {
	case model.TdaiJobTypeSwitchToFree:
		err = handleSwitchToFree(&job)
	case model.TdaiJobTypeSwitchToOff:
		err = handleSwitchToOff(&job)
	case model.TdaiJobTypeSwitchToPro:
		err = handleSwitchToPro(&job)
	default:
		err = common.I18nError(i18n.MsgTDAIUnknownJobType, job.JobType)
	}
	handlerCost := time.Since(handlerStart)

	if err != nil {
		log.Warn("[TaskEngine] handler 返回错误，进入失败处理",
			"handler_cost", handlerCost.String(),
			"error", err,
		)
		markJobFailed(&job, err)
	} else {
		log.Info("[TaskEngine] handler 执行成功，进入成功处理",
			"handler_cost", handlerCost.String(),
		)
		markJobSucceeded(&job)
	}
}

// markJobSucceeded 标记任务成功。
func markJobSucceeded(job *model.TdaiJob) {
	ctx := lookupJobCtx(job.ID)
	log := jobLogger(job)
	now := time.Now()
	if err := model.DB(ctx).Model(job).Updates(map[string]any{
		"state":        model.TdaiJobStateSucceeded,
		"current_step": "done",
		"progress":     100,
		"finished_at":  &now,
		"last_error":   "",
		"error_code":   "",
	}).Error; err != nil {
		log.Error("[TaskEngine] 更新任务为 SUCCEEDED 失败", "error", err)
	}
	log.Info("[TaskEngine] 任务终态",
		"state", model.TdaiJobStateSucceeded,
		"current_step", "done",
		"progress", 100,
	)
}

// markJobFailed 标记任务失败，若未超重试上限且非不可重试错误则安排退避重试。
func markJobFailed(job *model.TdaiJob, jobErr error) {
	ctx := lookupJobCtx(job.ID)
	log := jobLogger(job)
	errMsg := jobErr.Error()
	now := time.Now()

	// 不可重试错误：直接终态
	var nonRetryable *NonRetryableError
	forceTerminal := errors.As(jobErr, &nonRetryable)

	if !forceTerminal && job.Attempt < job.MaxAttempts {
		// 指数退避：5s, 30s, 180s, 180s, 180s, ...（封顶 180s）
		backoff := time.Duration(model.TdaiJobDefaultBackoffBase) * time.Second
		for i := 1; i < job.Attempt; i++ {
			backoff *= time.Duration(model.TdaiJobDefaultBackoffFactor)
		}
		maxBackoff := time.Duration(model.TdaiJobDefaultBackoffMax) * time.Second
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
		nextRun := now.Add(backoff)
		if err := model.DB(ctx).Model(job).Updates(map[string]any{
			"state":      model.TdaiJobStatePending,
			"run_at":     nextRun,
			"last_error": errMsg,
		}).Error; err != nil {
			log.Error("[TaskEngine] 更新任务为 PENDING（退避重试）失败", "error", err)
		}
		log.Warn("[TaskEngine] 任务失败，将退避重试",
			"state", model.TdaiJobStatePending,
			"current_step", job.CurrentStep,
			"progress", job.Progress,
			"max_attempts", job.MaxAttempts,
			"next_run", nextRun,
			"backoff", backoff.String(),
			"error", errMsg,
			"non_retryable", forceTerminal,
		)
	} else {
		if err := model.DB(ctx).Model(job).Updates(map[string]any{
			"state":       model.TdaiJobStateFailed,
			"last_error":  errMsg,
			"finished_at": &now,
		}).Error; err != nil {
			log.Error("[TaskEngine] 更新任务为 FAILED 失败", "error", err)
		}
		log.Error("[TaskEngine] 任务终态",
			"state", model.TdaiJobStateFailed,
			"current_step", job.CurrentStep,
			"progress", job.Progress,
			"max_attempts", job.MaxAttempts,
			"non_retryable", forceTerminal,
			"error", errMsg,
		)

		// SWITCH_TO_PRO 失败时：释放已分配的远端记忆库，避免 used 泄漏
		if job.JobType == model.TdaiJobTypeSwitchToPro {
			log.Info("[TaskEngine] SWITCH_TO_PRO 最终失败，开始回滚远端记忆库")
			rollbackProMemSpace(ctx, job.InstanceID)

			// OpenClaw 实例：回滚 offload 配置（switch_pro.sh 可能已写入 offload 配置）
			if model.GetAgentRuntimeType(ctx, controller.LookupAgentType(ctx, job.InstanceID)) == model.AgentTypeOpenClaw {
				log.Info("[TaskEngine] 回滚 OpenClaw offload 配置")
				script := `
SCRIPT_DIR="$(dirname "$(readlink -f "$0")" 2>/dev/null || echo /usr/local/bin)"
# setup-offload.sh 在 switch_pro.sh 的同级目录（scripts/），TAT 脚本执行时不一定能找到
# 直接用 python3 修改 openclaw.json 关闭 offload（与 setup-offload.sh --disable 逻辑一致）
OPENCLAW_JSON="$HOME/.openclaw/openclaw.json"
if [ -f "$OPENCLAW_JSON" ]; then
  python3 -c "
import json
with open('$OPENCLAW_JSON') as f:
    cfg = json.load(f)
entry = cfg.get('plugins', {}).get('entries', {}).get('memory-tencentdb', {})
offload = entry.get('config', {}).get('offload', {})
offload['enabled'] = False
slots = cfg.get('plugins', {}).get('slots', {})
if 'contextEngine' in slots:
    del slots['contextEngine']
with open('$OPENCLAW_JSON', 'w') as f:
    json.dump(cfg, f, indent=2, ensure_ascii=False)
    f.write('\n')
print('offload disabled')
" 2>&1 || echo "WARN: offload rollback failed"
else
  echo "openclaw.json not found, skip offload rollback"
fi
`
				if _, err := controller.RunInlineScript(ctx, job.InstanceID, script, 30); err != nil {
					log.Warn("[TaskEngine] offload 回滚失败（非阻断）", "error", err)
				}
			}
		}

		// 回滚 switch_status
		log.Info("[TaskEngine] 回滚 switch_status", "instance_id", job.InstanceID)
		if err := model.DB(ctx).Model(&model.MemoryTDAIPlugin{}).
			Where("instance_id = ?", job.InstanceID).
			Updates(map[string]any{
				"switch_status": model.MemorySwitchStatusNone,
				"last_task_id":  job.ID,
			}).Error; err != nil {
			log.Error("[TaskEngine] 回滚 switch_status 失败", "error", err)
		}
	}
}
