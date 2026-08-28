package model

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// 本文件覆盖 sg_drain_state.go 的所有 helper：
//   GetDrainState / IncrDrainFail / MarkDrainStuck / ClearDrainState / ListStuckDrainStates

func setupDrainStateTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&SGDrainState{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	orig := UseDBForTest(db)
	return func() { orig() }
}

func TestSGDrainStateTableName(t *testing.T) {
	if name := (SGDrainState{}).TableName(); name != "sg_drain_state" {
		t.Errorf("TableName = %q, want sg_drain_state", name)
	}
}

func TestGetDrainState_NotFoundReturnsNilNil(t *testing.T) {
	defer setupDrainStateTestDB(t)()
	s, err := GetDrainState(context.Background(), "ins-missing")
	if err != nil {
		t.Fatalf("Get missing: %v", err)
	}
	if s != nil {
		t.Errorf("missing instance MUST return nil, got %+v", s)
	}
}

func TestIncrDrainFail_InsertThenUpdate(t *testing.T) {
	defer setupDrainStateTestDB(t)()

	// 第一次：插入 fail_count=1
	n, err := IncrDrainFail(context.Background(), "ins-1", "", "transient error")
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	if n != 1 {
		t.Errorf("first fail_count = %d, want 1", n)
	}

	// 确认插入了
	s, err := GetDrainState(context.Background(), "ins-1")
	if err != nil || s == nil {
		t.Fatalf("GetDrainState after insert: err=%v row=%+v", err, s)
	}
	if s.FailCount != 1 {
		t.Errorf("row FailCount = %d, want 1", s.FailCount)
	}
	if s.LastError != "transient error" {
		t.Errorf("LastError = %q", s.LastError)
	}

	// 第二次：递增到 2
	n, err = IncrDrainFail(context.Background(), "ins-1", "", "another err")
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if n != 2 {
		t.Errorf("second fail_count = %d, want 2", n)
	}

	// 确认 LastError 被覆盖
	s2, _ := GetDrainState(context.Background(), "ins-1")
	if s2.LastError != "another err" {
		t.Errorf("LastError not overwritten: %q", s2.LastError)
	}
}

func TestMarkDrainStuck(t *testing.T) {
	defer setupDrainStateTestDB(t)()
	// 先插入一行
	if _, err := IncrDrainFail(context.Background(), "ins-stuck", "", "err"); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := MarkDrainStuck(context.Background(), "ins-stuck"); err != nil {
		t.Fatalf("MarkDrainStuck: %v", err)
	}

	s, err := GetDrainState(context.Background(), "ins-stuck")
	if err != nil || s == nil {
		t.Fatalf("Get: err=%v", err)
	}
	if s.StuckAt == nil {
		t.Error("StuckAt MUST be set after MarkDrainStuck")
	}
}

func TestClearDrainState(t *testing.T) {
	defer setupDrainStateTestDB(t)()
	if _, err := IncrDrainFail(context.Background(), "ins-clear", "", "err"); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := ClearDrainState(context.Background(), "ins-clear"); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	s, _ := GetDrainState(context.Background(), "ins-clear")
	if s != nil {
		t.Error("drain state row MUST be deleted after Clear")
	}
}

func TestListStuckDrainStates(t *testing.T) {
	defer setupDrainStateTestDB(t)()
	// 两个 stuck、一个 active（未 stuck）
	if _, err := IncrDrainFail(context.Background(), "ins-a", "", "e"); err != nil {
		t.Fatal(err)
	}
	if err := MarkDrainStuck(context.Background(), "ins-a"); err != nil {
		t.Fatal(err)
	}
	if _, err := IncrDrainFail(context.Background(), "ins-b", "", "e"); err != nil {
		t.Fatal(err)
	}
	if err := MarkDrainStuck(context.Background(), "ins-b"); err != nil {
		t.Fatal(err)
	}
	if _, err := IncrDrainFail(context.Background(), "ins-c", "", "e"); err != nil {
		t.Fatal(err)
	}

	stuck, err := ListStuckDrainStates(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(stuck) != 2 {
		t.Errorf("got %d stuck, want 2", len(stuck))
	}
	for _, s := range stuck {
		if s.StuckAt == nil {
			t.Errorf("returned row %s has nil StuckAt", s.InstanceID)
		}
	}
}
