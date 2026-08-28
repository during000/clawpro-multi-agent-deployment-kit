package task

import (
	"sync"
	"sync/atomic"
)

// casInt32 原子 CompareAndSwap 操作（用于 per-tenant 状态管理）。
func casInt32(addr *int32, old, new int32) bool {
	return atomic.CompareAndSwapInt32(addr, old, new)
}

// storeInt32 原子 Store 操作。
func storeInt32(addr *int32, val int32) {
	atomic.StoreInt32(addr, val)
}

// loadInt32 原子 Load 操作。
func loadInt32(addr *int32) int32 {
	return atomic.LoadInt32(addr)
}

// tenantBool 提供 per-tenant 的 atomic.Bool 语义。
type tenantBool struct {
	m sync.Map // key: string(identifier), value: *int32 (0=false, 1=true)
}

func (tb *tenantBool) get(identifier string) *int32 {
	val, _ := tb.m.LoadOrStore(identifier, new(int32))
	return val.(*int32)
}

// CompareAndSwap 对指定租户执行 CAS。
func (tb *tenantBool) CompareAndSwap(identifier string, old, new bool) bool {
	var o, n int32
	if old {
		o = 1
	}
	if new {
		n = 1
	}
	return atomic.CompareAndSwapInt32(tb.get(identifier), o, n)
}

// Store 对指定租户 Store。
func (tb *tenantBool) Store(identifier string, val bool) {
	var v int32
	if val {
		v = 1
	}
	atomic.StoreInt32(tb.get(identifier), v)
}

// tenantInt64 提供 per-tenant 的 atomic.Int64 语义。
type tenantInt64 struct {
	m sync.Map // key: string(identifier), value: *int64
}

func (ti *tenantInt64) get(identifier string) *int64 {
	val, _ := ti.m.LoadOrStore(identifier, new(int64))
	return val.(*int64)
}

// Add 对指定租户加 delta，返回新值。
func (ti *tenantInt64) Add(identifier string, delta int64) int64 {
	return atomic.AddInt64(ti.get(identifier), delta)
}
