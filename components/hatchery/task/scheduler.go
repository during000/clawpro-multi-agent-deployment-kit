package task

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"hatchery/common"
	"hatchery/model"
)

// TaskPriority 任务优先级。
type TaskPriority int

const (
	// PriorityNormal 普通优先级（默认），与其他任务并发调度。
	PriorityNormal TaskPriority = 0
	// PriorityStartup 启动优先级，在所有普通任务之前同步执行完毕。
	// 用于 Seed、一次性迁移等定时任务依赖的前置操作。
	PriorityStartup TaskPriority = 1
)

// TaskDef 定义一个后台任务。
type TaskDef struct {
	Name         string                    // 任务唯一名称
	Interval     time.Duration             // 调度间隔；0 表示一次性任务（启动执行一次后不再调度）
	RunFunc      func(ctx context.Context) // 执行函数，ctx 已注入 TenantSnapshot
	NeedDistLock bool                      // true → 执行前 TryLock，拿不到则跳过
	PerTenant    bool                      // true → 遍历所有租户分发
	InitialDelay time.Duration             // 首次执行前的延迟
	Priority     TaskPriority              // 任务优先级；PriorityStartup 在普通任务前同步执行
}

// taskRegistry 全局任务注册表，由各 task 文件的 init() 写入。
var taskRegistry []TaskDef

// RegisterTask 注册一个后台任务到全局注册表。
// 必须在 StartScheduler 之前调用（通常在 init() 中）。
func RegisterTask(def TaskDef) {
	taskRegistry = append(taskRegistry, def)
}

const defaultWorkerCount = 100

// taskItem 是投入 worker pool 的工作单元。
type taskItem struct {
	def      TaskDef
	snapshot common.TenantSnapshot
}

var (
	taskCh      chan taskItem
	stopCh      chan struct{} // 通知 scheduler goroutines 退出
	workerWg    sync.WaitGroup
	schedulerWg sync.WaitGroup
)

// StartScheduler 启动后台任务调度器（立即返回，不阻塞调用方）。
// 内部启动 goroutine：先执行所有 PriorityStartup 任务，完成后再启动普通定时任务。
// 这样 HTTP 服务可以立即就绪，但定时任务仍等 Seed/迁移完成后才开始调度。
func StartScheduler(workerCount int) {
	if workerCount <= 0 {
		workerCount = defaultWorkerCount
	}

	// 分离启动任务和普通任务
	var startupTasks, normalTasks []TaskDef
	for _, def := range taskRegistry {
		if def.Priority == PriorityStartup {
			startupTasks = append(startupTasks, def)
		} else {
			normalTasks = append(normalTasks, def)
		}
	}

	slog.Info("[Scheduler] 启动后台任务调度器",
		"startup_tasks", len(startupTasks),
		"normal_tasks", len(normalTasks),
		"worker_count", workerCount,
	)

	taskCh = make(chan taskItem, 256)
	stopCh = make(chan struct{})

	// 启动固定数量的 worker goroutine
	for i := 0; i < workerCount; i++ {
		workerWg.Add(1)
		go func(workerID int) {
			defer workerWg.Done()
			for item := range taskCh {
				executeTask(item)
			}
		}(i)
	}

	// 异步：先执行 startup 任务，完成后再启动普通定时任务调度
	go func() {
		if len(startupTasks) > 0 {
			runStartupTasks(startupTasks)
		}
		// startup 任务全部完成后，启动普通定时任务
		for _, def := range normalTasks {
			schedulerWg.Add(1)
			go runTaskScheduler(def, taskCh, stopCh)
		}
	}()
}

// runStartupTasks 通过 worker pool 并发执行所有启动优先级任务。
// 每个 TaskDef 按顺序处理，但同一 TaskDef 的多个租户可并发执行。
// 全部完成后返回。
func runStartupTasks(tasks []TaskDef) {
	slog.Info("[Scheduler] 开始执行启动期任务", "count", len(tasks))
	for _, def := range tasks {
		if def.PerTenant {
			tenants, err := model.ListAllTenants()
			if err != nil {
				slog.Error("[Scheduler] 启动任务获取租户列表失败", "task", def.Name, "error", err)
				continue
			}
			// 通过 taskCh 投递到 worker pool 并发执行，WaitGroup 等待全部完成
			var wg sync.WaitGroup
			for _, snap := range tenants {
				wg.Add(1)
				go func(item taskItem) {
					defer wg.Done()
					executeTask(item)
				}(taskItem{def: def, snapshot: snap})
			}
			wg.Wait()
		} else {
			executeTask(taskItem{def: def})
		}
		slog.Info("[Scheduler] 启动期任务完成", "task", def.Name)
	}
	slog.Info("[Scheduler] 所有启动期任务执行完毕")
}

// StopScheduler 停止所有后台任务，等待 worker 退出。
// 先通知 scheduler goroutines 停止投递，再关闭 taskCh 让 worker 退出。
func StopScheduler() {
	if stopCh != nil {
		close(stopCh)      // 通知所有 scheduler goroutines 退出
		schedulerWg.Wait() // 等所有 scheduler goroutines 退出（不再写 taskCh）
		close(taskCh)      // 安全关闭 channel
		workerWg.Wait()    // 等所有 worker 消费完剩余任务并退出
		slog.Info("[Scheduler] 所有后台任务已停止")
	}
}

// runTaskScheduler 是单个 TaskDef 的调度循环。
func runTaskScheduler(def TaskDef, ch chan<- taskItem, done <-chan struct{}) {
	defer schedulerWg.Done()

	// 初始延迟
	if def.InitialDelay > 0 {
		select {
		case <-done:
			return
		case <-time.After(def.InitialDelay):
		}
	}

	// 立即执行一次
	if !sendTask(def, ch, done) {
		return
	}

	// 一次性任务：执行完即退出
	if def.Interval <= 0 {
		return
	}

	// 周期执行
	ticker := time.NewTicker(def.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if !sendTask(def, ch, done) {
				return
			}
		}
	}
}

// sendTask 将任务投递到 channel，返回 false 表示收到停止信号。
func sendTask(def TaskDef, ch chan<- taskItem, done <-chan struct{}) bool {
	if def.PerTenant {
		tenants, err := model.ListAllTenants()
		if err != nil {
			slog.Error("[Scheduler] 获取租户列表失败", "task", def.Name, "error", err)
			return true // 错误不退出，等下次 tick
		}
		for _, snap := range tenants {
			select {
			case <-done:
				return false
			case ch <- taskItem{def: def, snapshot: snap}:
			}
		}
	} else {
		select {
		case <-done:
			return false
		case ch <- taskItem{def: def}:
		}
	}
	return true
}

// executeTask 执行单个任务实例（一个租户 × 一个任务）。
func executeTask(item taskItem) {
	// panic recovery
	defer func() {
		if r := recover(); r != nil {
			slog.Error("[Scheduler] 任务 panic 已恢复",
				"task", item.def.Name,
				"identifier", item.snapshot.Identifier,
				"panic", fmt.Sprintf("%v", r),
			)
		}
	}()

	// 构造 ctx：注入租户信息 + task trace
	ctx := context.Background()
	if item.snapshot.Identifier != "" {
		ctx = common.InjectTenant(ctx, item.snapshot)
		ctx = common.InjectI18nPrinter(ctx)
	}
	ctx = common.WithTaskTrace(ctx, item.def.Name)

	// 分布式锁
	if item.def.NeedDistLock {
		lockKey := fmt.Sprintf("task:%s", item.def.Name)
		lock, err := model.TryLock(ctx, lockKey)
		if err != nil {
			slog.Debug("[Scheduler] 未获取到锁，跳过本次执行",
				"task", item.def.Name,
				"identifier", item.snapshot.Identifier,
			)
			return
		}
		defer lock.Release()
	}

	slog.Info("[Scheduler] 开始执行任务",
		"task", item.def.Name,
		"identifier", item.snapshot.Identifier,
	)
	item.def.RunFunc(ctx)
}

// TaskCount 返回已注册的任务数量（测试用）。
func TaskCount() int {
	return len(taskRegistry)
}

// ResetRegistry 清空注册表（仅测试用）。
func ResetRegistry() {
	taskRegistry = nil
}

// RandomDuration 返回 [0, max) 之间的随机 Duration（用于错峰延迟）。
func RandomDuration(max time.Duration) time.Duration {
	if max <= 0 {
		return 0
	}
	return time.Duration(rand.Int63n(int64(max)))
}
