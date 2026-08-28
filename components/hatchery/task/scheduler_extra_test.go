package task

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"hatchery/common"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ─── scheduler 补充覆盖 ───────────────────────────────────────────────────

// TestSendTask_Done_PerTenant 测试 PerTenant=true 时 done 通道触发提前退出。
// 使用无缓冲 ch + 已关闭 done，确保 select 选择 done 分支。
func TestSendTask_Done_PerTenant(t *testing.T) {
	old := common.FixedSnapshot
	common.FixedSnapshot = &common.TenantSnapshot{Identifier: "tenant-x"}
	defer func() { common.FixedSnapshot = old }()

	def := TaskDef{Name: "test-done", Interval: 0, RunFunc: func(ctx context.Context) {}, PerTenant: true}
	// 无缓冲 channel：无法写入，select 只能走 done 分支
	ch := make(chan taskItem)
	done := make(chan struct{})
	close(done)

	result := sendTask(def, ch, done)
	if result {
		t.Error("done 已关闭时 sendTask 应返回 false")
	}
}

// TestSendTask_Done_NonPerTenant 测试 PerTenant=false 时 done 通道触发提前退出。
// 通过填满 channel 强制 select 只能走 done 分支。
func TestSendTask_Done_NonPerTenant(t *testing.T) {
	def := TaskDef{Name: "test-done-np", Interval: 0, RunFunc: func(ctx context.Context) {}, PerTenant: false}
	// ch 容量为 0，无法写入
	ch := make(chan taskItem)
	done := make(chan struct{})
	close(done)

	// done 已关闭，ch 无缓冲无法写，sendTask 应选择 done 分支返回 false
	result := sendTask(def, ch, done)
	if result {
		t.Error("done 已关闭时 sendTask(非PerTenant) 应返回 false")
	}
}

// TestSendTask_ListTenantsError 测试 ListAllTenants 失败时 sendTask 返回 true（继续）。
func TestSendTask_ListTenantsError(t *testing.T) {
	// 使用空内存数据库（没有 site_configs 表），ListAllTenants 会返回错误
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))

	// 清空 FixedSnapshot 使 ListAllTenants 走 DB 查询路径（会因为表不存在而报错）
	old := common.FixedSnapshot
	common.FixedSnapshot = nil
	defer func() { common.FixedSnapshot = old }()

	def := TaskDef{Name: "test-list-err", Interval: 0, RunFunc: func(ctx context.Context) {}, PerTenant: true}
	ch := make(chan taskItem, 10)
	done := make(chan struct{})

	result := sendTask(def, ch, done)
	// ListAllTenants 出错时返回 true（不退出，等下次 tick）
	if !result {
		t.Error("ListAllTenants 失败时 sendTask 应返回 true")
	}
}

// TestExecuteTask_NeedDistLock_SQLite SQLite 模式下 TryLock 总是成功，任务应被执行。
func TestExecuteTask_NeedDistLock_SQLite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))

	var executed atomic.Bool
	item := taskItem{
		def: TaskDef{
			Name:         "dist-lock-task",
			Interval:     0,
			RunFunc:      func(ctx context.Context) { executed.Store(true) },
			NeedDistLock: true,
			PerTenant:    false,
		},
	}
	executeTask(item)
	if !executed.Load() {
		t.Error("SQLite 模式下 NeedDistLock 任务应被执行")
	}
}

// TestExecuteTask_WithIdentifier 注入 identifier 后任务能通过 ctx 读到。
func TestExecuteTask_WithIdentifier(t *testing.T) {
	var gotIdentifier string
	item := taskItem{
		def: TaskDef{
			Name:     "id-task",
			Interval: 0,
			RunFunc: func(ctx context.Context) {
				if snap, ok := common.GetTenantSnapshot(ctx); ok {
					gotIdentifier = snap.Identifier
				}
			},
			PerTenant: true,
		},
		snapshot: common.TenantSnapshot{Identifier: "my-tenant"},
	}
	executeTask(item)
	if gotIdentifier != "my-tenant" {
		t.Errorf("期望 identifier=my-tenant，实际=%s", gotIdentifier)
	}
}

// TestStartScheduler_NeedDistLock_Coverage 覆盖 NeedDistLock=true 的调度路径。
func TestStartScheduler_NeedDistLock_Coverage(t *testing.T) {
	ResetRegistry()
	defer ResetRegistry()

	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	t.Cleanup(model.UseDBForTest(db))

	old := common.FixedSnapshot
	common.FixedSnapshot = &common.TenantSnapshot{Identifier: "dist-tenant"}
	defer func() { common.FixedSnapshot = old }()

	var count atomic.Int32
	RegisterTask(TaskDef{
		Name:         "dist-task",
		Interval:     0,
		RunFunc:      func(ctx context.Context) { count.Add(1) },
		NeedDistLock: true,
		PerTenant:    true,
	})

	StartScheduler(1)
	time.Sleep(100 * time.Millisecond)
	StopScheduler()

	if count.Load() == 0 {
		t.Error("NeedDistLock=true 任务应至少执行一次")
	}
}

// ─── sg_ruleset_init_task 补充覆盖 ────────────────────────────────────────

// TestStartScheduler_PriorityStartup_RunsBeforeNormal 验证 PriorityStartup 任务
// 在普通任务之前同步执行完毕。
func TestStartScheduler_PriorityStartup_RunsBeforeNormal(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))

	// 保存并清空注册表
	oldRegistry := taskRegistry
	taskRegistry = nil
	defer func() { taskRegistry = oldRegistry }()

	var order []string
	var mu sync.Mutex

	RegisterTask(TaskDef{
		Name:     "normal-task",
		Interval: 1 * time.Hour,
		RunFunc: func(ctx context.Context) {
			mu.Lock()
			order = append(order, "normal")
			mu.Unlock()
		},
	})
	RegisterTask(TaskDef{
		Name:     "startup-task",
		Interval: 0,
		Priority: PriorityStartup,
		RunFunc: func(ctx context.Context) {
			mu.Lock()
			order = append(order, "startup")
			mu.Unlock()
		},
	})

	StartScheduler(2)
	// 给 normal task 一点时间执行首次
	time.Sleep(100 * time.Millisecond)
	StopScheduler()

	mu.Lock()
	defer mu.Unlock()

	if len(order) < 2 {
		t.Fatalf("expected at least 2 executions, got %d: %v", len(order), order)
	}
	if order[0] != "startup" {
		t.Errorf("startup task should run first, got order: %v", order)
	}
}

// TestStartScheduler_PriorityStartup_PerTenant 验证 PriorityStartup + PerTenant 任务
// 遍历所有租户执行。
func TestStartScheduler_PriorityStartup_PerTenant(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))

	oldRegistry := taskRegistry
	taskRegistry = nil
	defer func() { taskRegistry = oldRegistry }()

	// 设置 FixedSnapshot 模拟单租户
	oldSnap := common.FixedSnapshot
	common.FixedSnapshot = &common.TenantSnapshot{Identifier: "test-tenant"}
	defer func() { common.FixedSnapshot = oldSnap }()

	var executed atomic.Bool
	RegisterTask(TaskDef{
		Name:      "startup-per-tenant",
		Interval:  0,
		Priority:  PriorityStartup,
		PerTenant: true,
		RunFunc: func(ctx context.Context) {
			snap, ok := common.GetTenantSnapshot(ctx)
			if ok && snap.Identifier == "test-tenant" {
				executed.Store(true)
			}
		},
	})

	StartScheduler(2)
	// 等待异步 startup 任务完成
	time.Sleep(200 * time.Millisecond)
	StopScheduler()

	if !executed.Load() {
		t.Error("PriorityStartup PerTenant task should execute with correct tenant context")
	}
}
