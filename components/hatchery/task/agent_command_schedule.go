package task

import (
	"context"
	"log/slog"
	"time"

	"hatchery/controller"
	"hatchery/i18n"
	"hatchery/model"
)

// agent-command 定时任务的两个后台协程，统一注册到 scheduler：
//
//  1. runner（1s）    ：扫描到期 schedule（enabled=true && next_run_at<=now），CAS 抢占推进
//     next_run_at 后通过 controller.TriggerScheduleDispatch 触发一次 dispatch。
//     - NeedDistLock=true：多实例同租户由分布式锁仲裁，schedule 层再用 next_run_at CAS 二次去重。
//     - Interval=1s：任务为分钟级精度，next_run_at 落在整分（HH:MM:00）；1s 扫描使触发尽量贴近
//     整点（最坏延迟 ~1s）。命中集合走 (enabled,next_run_at) 联合索引，非到期分钟近乎零成本。
//     - InitialDelay=5s：让 poller / reconcile 先就绪。
//
//  2. reconcile（1s）：扫描 is_running=true 的 schedule，查 last_dispatch_slug 对应 dispatch
//     是否终态后订正 is_running，供列表展示「执行中」状态。
//     - NeedDistLock=false：置 false 为幂等操作（多实例并发无副作用），免每秒抢锁开销。
//     - Interval=1s：is_running 是展示态需较实时收敛；命中集合很小，开销低。
//     - InitialDelay=6s：让 poller / runner 先就绪。
//
// 两者均 PerTenant=true：scheduler 遍历所有租户并注入 TenantSnapshot，下游 model.DB(ctx) 据此
// 按 identifier 过滤。编排逻辑（找到期/抢占/串行保护/记录/订正）在本包；实际派发（startDispatch
// + TAT）封装在 controller，通过 triggerScheduleDispatch seam 调用，便于单测替换。
func init() {
	RegisterTask(TaskDef{
		Name:         "agent-command-schedule-runner",
		Interval:     1 * time.Second,
		RunFunc:      runAgentCommandSchedule,
		NeedDistLock: true,
		PerTenant:    true,
		InitialDelay: 5 * time.Second,
	})
	RegisterTask(TaskDef{
		Name:         "agent-command-schedule-reconcile",
		Interval:     1 * time.Second,
		RunFunc:      runAgentCommandScheduleReconcile,
		NeedDistLock: false,
		PerTenant:    true,
		InitialDelay: 6 * time.Second,
	})
}

// agentScheduleRunBatchLimit 单次扫描处理的到期定时任务上限。
const agentScheduleRunBatchLimit = 200

// triggerScheduleDispatch 是「触发一次 dispatch」的 seam，默认指向 controller.TriggerScheduleDispatch；
// 单测可替换为 stub，从而把 runner 编排逻辑与真实派发 + TAT 解耦。
var triggerScheduleDispatch = controller.TriggerScheduleDispatch

// runAgentCommandSchedule 扫描当前租户内所有到期定时任务并触发。
//
// 由 task scheduler 每 1s 调一次（PerTenant + NeedDistLock）。流程：
//  1. 查 enabled=true 且 next_run_at<=now 的行
//  2. CAS 抢占推进 next_run_at（多实例去重，独占本周期触发权）
//  3. 串行保护：若上一轮 dispatch 仍未终态，跳过本次触发（next_run_at 已推进，不补跑）
//  4. 通过 seam 触发一次 dispatch，记录运行结果 + 追加执行记录
func runAgentCommandSchedule(ctx context.Context) {
	defer func() {
		if rec := recover(); rec != nil {
			slog.Error("[AgentCommandSchedule] tick panic", "error", rec)
		}
	}()

	now := time.Now()
	schedules, err := model.FindDueSchedules(ctx, now, agentScheduleRunBatchLimit)
	if err != nil {
		slog.Error("[AgentCommandSchedule] 查询到期定时任务失败", "error", err)
		return
	}
	for i := range schedules {
		s := &schedules[i]
		if s.NextRunAt == nil {
			continue
		}
		expected := *s.NextRunAt
		// 计算下一次（once → nil）；CAS 抢占成功才执行，避免多实例 / 重入重复触发
		next, cerr := s.ComputeNextRun(now)
		if cerr != nil {
			slog.Error("[AgentCommandSchedule] 计算下次触发失败", "schedule_id", s.ID, "error", cerr)
			next = nil
		}
		claimed, err := model.ClaimScheduleRun(ctx, s.ID, expected, next)
		if err != nil {
			slog.Error("[AgentCommandSchedule] 抢占失败", "schedule_id", s.ID, "error", err)
			continue
		}
		if !claimed {
			continue // 已被其它实例/并发抢走
		}

		// 串行保护：上一轮 dispatch 仍未终态 → 跳过本次（next_run_at 已推进，不补跑）
		if s.LastDispatchSlug != "" {
			if prev, perr := model.FindDispatchBySlug(ctx, s.LastDispatchSlug); perr == nil && !prev.IsTerminal() {
				slog.Info("[AgentCommandSchedule] 上一轮未完成，跳过本次",
					"schedule_id", s.ID, "last_dispatch", s.LastDispatchSlug)
				_ = model.MarkScheduleSkipped(ctx, s.ID, i18n.T(ctx, i18n.MsgScheduleSkippedPreviousRunning))
				continue
			}
		}

		slug, derr := triggerScheduleDispatch(ctx, s)
		ranAt := time.Now()
		if derr != nil {
			slog.Warn("[AgentCommandSchedule] 触发 dispatch 失败",
				"schedule_id", s.ID, "error", derr)
			_ = model.MarkScheduleRunResult(ctx, s.ID, ranAt, "", derr.Error(), false)
			continue
		}
		_ = model.MarkScheduleRunResult(ctx, s.ID, ranAt, slug, "", true)
		_ = model.CreateScheduleRecord(ctx, s.ID, slug)
	}
}

// runAgentCommandScheduleReconcile 订正当前租户内 is_running 标记。
//
// 由 task scheduler 高频（1s）调一次（PerTenant）。流程：
//  1. 查 is_running=true 的行（集合天然很小）
//  2. 批量查这些 last_dispatch_slug 对应 dispatch（避免 N+1）
//  3. 对应 dispatch 已终态（或 slug 缺失/查不到）→ 收集 id 批量置 is_running=false
//
// 幂等：多实例并发重复置 false 无副作用，故无需分布式锁。
func runAgentCommandScheduleReconcile(ctx context.Context) {
	defer func() {
		if rec := recover(); rec != nil {
			slog.Error("[AgentCommandSchedule] reconcile panic", "error", rec)
		}
	}()

	running, err := model.FindRunningSchedules(ctx)
	if err != nil {
		slog.Error("[AgentCommandSchedule] 查询执行中定时任务失败", "error", err)
		return
	}
	if len(running) == 0 {
		return
	}

	// 批量查 dispatch 终态
	slugs := make([]string, 0, len(running))
	for i := range running {
		if running[i].LastDispatchSlug != "" {
			slugs = append(slugs, running[i].LastDispatchSlug)
		}
	}
	terminalBySlug := make(map[string]bool, len(slugs))
	if len(slugs) > 0 {
		var rows []model.AgentCommandDispatch
		if err := model.DB(ctx).Where("slug IN ?", slugs).Find(&rows).Error; err != nil {
			slog.Error("[AgentCommandSchedule] reconcile 查询 dispatch 失败", "error", err)
			return
		}
		for i := range rows {
			terminalBySlug[rows[i].Slug] = rows[i].IsTerminal()
		}
	}

	// slug 为空、dispatch 查不到、或已终态 → 订正为非执行中
	var done []uint
	for i := range running {
		s := &running[i]
		if terminal, found := terminalBySlug[s.LastDispatchSlug]; s.LastDispatchSlug == "" || !found || terminal {
			done = append(done, s.ID)
		}
	}
	if err := model.ClearScheduleRunning(ctx, done); err != nil {
		slog.Error("[AgentCommandSchedule] 订正 is_running 失败", "error", err)
	}
}
