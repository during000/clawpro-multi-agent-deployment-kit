package task

import (
	"sync"
	"testing"
)

// ─── casInt32 / storeInt32 / loadInt32 ───────────────────────────────────

func TestCasInt32_Success(t *testing.T) {
	var v int32 = 0
	if !casInt32(&v, 0, 1) {
		t.Error("CAS(0→1) 应成功")
	}
	if loadInt32(&v) != 1 {
		t.Errorf("CAS 后应为 1，实际 %d", loadInt32(&v))
	}
}

func TestCasInt32_Fail(t *testing.T) {
	var v int32 = 1
	if casInt32(&v, 0, 2) {
		t.Error("CAS(0→2) 在当前值=1 时应失败")
	}
	if loadInt32(&v) != 1 {
		t.Errorf("CAS 失败后值应保持 1，实际 %d", loadInt32(&v))
	}
}

func TestStoreInt32(t *testing.T) {
	var v int32
	storeInt32(&v, 42)
	if loadInt32(&v) != 42 {
		t.Errorf("Store 后应为 42，实际 %d", loadInt32(&v))
	}
}

func TestLoadInt32_Initial(t *testing.T) {
	var v int32
	if loadInt32(&v) != 0 {
		t.Errorf("零值应为 0，实际 %d", loadInt32(&v))
	}
}

// ─── tenantBool ───────────────────────────────────────────────────────────

func TestTenantBool_StoreAndLoad(t *testing.T) {
	tb := &tenantBool{}
	tb.Store("tenant-a", true)
	if loadInt32(tb.get("tenant-a")) != 1 {
		t.Error("Store(true) 后应为 1")
	}
	tb.Store("tenant-a", false)
	if loadInt32(tb.get("tenant-a")) != 0 {
		t.Error("Store(false) 后应为 0")
	}
}

func TestTenantBool_CompareAndSwap_Success(t *testing.T) {
	tb := &tenantBool{}
	// 初始 false → true
	if !tb.CompareAndSwap("t1", false, true) {
		t.Error("CAS(false→true) 应成功")
	}
	if loadInt32(tb.get("t1")) != 1 {
		t.Error("CAS 后应为 true")
	}
}

func TestTenantBool_CompareAndSwap_Fail(t *testing.T) {
	tb := &tenantBool{}
	tb.Store("t2", true)
	// 当前 true，期望 false→true 应失败
	if tb.CompareAndSwap("t2", false, true) {
		t.Error("CAS(false→true) 在当前值=true 时应失败")
	}
}

func TestTenantBool_CompareAndSwap_TrueToFalse(t *testing.T) {
	tb := &tenantBool{}
	tb.Store("t3", true)
	if !tb.CompareAndSwap("t3", true, false) {
		t.Error("CAS(true→false) 应成功")
	}
	if loadInt32(tb.get("t3")) != 0 {
		t.Error("CAS 后应为 false")
	}
}

func TestTenantBool_Isolation(t *testing.T) {
	tb := &tenantBool{}
	tb.Store("a", true)
	tb.Store("b", false)
	if loadInt32(tb.get("a")) != 1 {
		t.Error("租户 a 应为 true")
	}
	if loadInt32(tb.get("b")) != 0 {
		t.Error("租户 b 应为 false")
	}
}

func TestTenantBool_Concurrent(t *testing.T) {
	tb := &tenantBool{}
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tb.Store("x", true)
			tb.CompareAndSwap("x", true, false)
		}()
	}
	wg.Wait()
}

// ─── tenantInt64 ──────────────────────────────────────────────────────────

func TestTenantInt64_Add(t *testing.T) {
	ti := &tenantInt64{}
	v := ti.Add("tenant-a", 5)
	if v != 5 {
		t.Errorf("Add(5) 后应为 5，实际 %d", v)
	}
	v = ti.Add("tenant-a", 3)
	if v != 8 {
		t.Errorf("Add(3) 后应为 8，实际 %d", v)
	}
}

func TestTenantInt64_NegativeDelta(t *testing.T) {
	ti := &tenantInt64{}
	ti.Add("t", 10)
	v := ti.Add("t", -3)
	if v != 7 {
		t.Errorf("Add(-3) 后应为 7，实际 %d", v)
	}
}

func TestTenantInt64_Isolation(t *testing.T) {
	ti := &tenantInt64{}
	ti.Add("a", 10)
	ti.Add("b", 20)
	if *ti.get("a") != 10 {
		t.Errorf("租户 a 应为 10，实际 %d", *ti.get("a"))
	}
	if *ti.get("b") != 20 {
		t.Errorf("租户 b 应为 20，实际 %d", *ti.get("b"))
	}
}

func TestTenantInt64_Concurrent(t *testing.T) {
	ti := &tenantInt64{}
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ti.Add("shared", 1)
		}()
	}
	wg.Wait()
	if *ti.get("shared") != 100 {
		t.Errorf("并发 Add 后应为 100，实际 %d", *ti.get("shared"))
	}
}
