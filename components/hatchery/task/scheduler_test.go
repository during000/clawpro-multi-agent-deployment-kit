package task

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"hatchery/common"
)

func TestRegisterTask(t *testing.T) {
	ResetRegistry()
	defer ResetRegistry()

	RegisterTask(TaskDef{Name: "test-1", Interval: time.Second, RunFunc: func(ctx context.Context) {}})
	RegisterTask(TaskDef{Name: "test-2", Interval: time.Second, RunFunc: func(ctx context.Context) {}})

	if TaskCount() != 2 {
		t.Fatalf("expected 2 tasks registered, got %d", TaskCount())
	}
}

func TestStartScheduler_ExecutesTask(t *testing.T) {
	ResetRegistry()
	defer ResetRegistry()

	old := common.FixedSnapshot
	common.FixedSnapshot = &common.TenantSnapshot{Identifier: "test-tenant"}
	defer func() { common.FixedSnapshot = old }()

	var count atomic.Int32
	RegisterTask(TaskDef{
		Name:      "counter",
		Interval:  50 * time.Millisecond,
		RunFunc:   func(ctx context.Context) { count.Add(1) },
		PerTenant: true,
	})

	StartScheduler(0)
	time.Sleep(200 * time.Millisecond)
	StopScheduler()

	// 至少执行了 1 次（立即执行） + ticker 触发
	if c := count.Load(); c < 2 {
		t.Fatalf("expected at least 2 executions, got %d", c)
	}
}

func TestStartScheduler_OneShot(t *testing.T) {
	ResetRegistry()
	defer ResetRegistry()

	old := common.FixedSnapshot
	common.FixedSnapshot = &common.TenantSnapshot{Identifier: "test"}
	defer func() { common.FixedSnapshot = old }()

	var count atomic.Int32
	RegisterTask(TaskDef{
		Name:      "oneshot",
		Interval:  0,
		RunFunc:   func(ctx context.Context) { count.Add(1) },
		PerTenant: true,
	})

	StartScheduler(0)
	time.Sleep(100 * time.Millisecond)
	StopScheduler()

	if c := count.Load(); c != 1 {
		t.Fatalf("expected exactly 1 execution for one-shot task, got %d", c)
	}
}

func TestStartScheduler_InitialDelay(t *testing.T) {
	ResetRegistry()
	defer ResetRegistry()

	old := common.FixedSnapshot
	common.FixedSnapshot = &common.TenantSnapshot{Identifier: "test"}
	defer func() { common.FixedSnapshot = old }()

	var count atomic.Int32
	RegisterTask(TaskDef{
		Name:         "delayed",
		Interval:     0,
		RunFunc:      func(ctx context.Context) { count.Add(1) },
		PerTenant:    true,
		InitialDelay: 150 * time.Millisecond,
	})

	StartScheduler(0)

	// 50ms 后还没执行
	time.Sleep(50 * time.Millisecond)
	if c := count.Load(); c != 0 {
		t.Fatalf("expected 0 executions before delay, got %d", c)
	}

	// 等待延迟结束
	time.Sleep(200 * time.Millisecond)
	StopScheduler()

	if c := count.Load(); c != 1 {
		t.Fatalf("expected 1 execution after delay, got %d", c)
	}
}

func TestStartScheduler_PanicRecovery(t *testing.T) {
	ResetRegistry()
	defer ResetRegistry()

	old := common.FixedSnapshot
	common.FixedSnapshot = &common.TenantSnapshot{Identifier: "test"}
	defer func() { common.FixedSnapshot = old }()

	var count atomic.Int32
	RegisterTask(TaskDef{
		Name:     "panicker",
		Interval: 50 * time.Millisecond,
		RunFunc: func(ctx context.Context) {
			count.Add(1)
			if count.Load() == 1 {
				panic("test panic")
			}
		},
		PerTenant: true,
	})

	StartScheduler(0)
	time.Sleep(200 * time.Millisecond)
	StopScheduler()

	// 即使第一次 panic，后续仍然继续执行
	if c := count.Load(); c < 2 {
		t.Fatalf("expected at least 2 executions despite panic, got %d", c)
	}
}

func TestStartScheduler_GracefulStop(t *testing.T) {
	ResetRegistry()
	defer ResetRegistry()

	old := common.FixedSnapshot
	common.FixedSnapshot = &common.TenantSnapshot{Identifier: "test"}
	defer func() { common.FixedSnapshot = old }()

	var count atomic.Int32
	RegisterTask(TaskDef{
		Name:      "stopper",
		Interval:  10 * time.Millisecond,
		RunFunc:   func(ctx context.Context) { count.Add(1) },
		PerTenant: true,
	})

	StartScheduler(0)
	time.Sleep(50 * time.Millisecond)
	StopScheduler()

	// 停止后不再增长
	afterStop := count.Load()
	time.Sleep(50 * time.Millisecond)
	if count.Load() != afterStop {
		t.Fatal("task continued executing after StopScheduler")
	}
}

func TestStartScheduler_PerTenantInjectsCtx(t *testing.T) {
	ResetRegistry()
	defer ResetRegistry()

	old := common.FixedSnapshot
	common.FixedSnapshot = &common.TenantSnapshot{Identifier: "my-tenant", Uin: "12345"}
	defer func() { common.FixedSnapshot = old }()

	var gotIdentifier string
	RegisterTask(TaskDef{
		Name:     "ctx-check",
		Interval: 0,
		RunFunc: func(ctx context.Context) {
			if snap, ok := common.GetTenantSnapshot(ctx); ok {
				gotIdentifier = snap.Identifier
			}
		},
		PerTenant: true,
	})

	StartScheduler(0)
	time.Sleep(50 * time.Millisecond)
	StopScheduler()

	if gotIdentifier != "my-tenant" {
		t.Fatalf("expected identifier 'my-tenant', got '%s'", gotIdentifier)
	}
}

func TestStartScheduler_NonPerTenant(t *testing.T) {
	ResetRegistry()
	defer ResetRegistry()

	old := common.FixedSnapshot
	common.FixedSnapshot = &common.TenantSnapshot{Identifier: "tenant-x"}
	defer func() { common.FixedSnapshot = old }()

	var executed atomic.Int32
	RegisterTask(TaskDef{
		Name:      "global-task",
		Interval:  0,
		RunFunc:   func(ctx context.Context) { executed.Add(1) },
		PerTenant: false,
	})

	StartScheduler(0)
	time.Sleep(50 * time.Millisecond)
	StopScheduler()

	if c := executed.Load(); c != 1 {
		t.Fatalf("expected 1 execution for non-PerTenant task, got %d", c)
	}
}
