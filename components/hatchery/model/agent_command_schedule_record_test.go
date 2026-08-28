package model

import (
	"context"
	"testing"
)

func TestCreateAndListScheduleRecords(t *testing.T) {
	defer setupScheduleDB(t)()
	ctx := context.Background()

	// schedule 1 插 3 条，schedule 2 插 1 条
	for i := 0; i < 3; i++ {
		if err := CreateScheduleRecord(ctx, 1, "slug-1-"+string(rune('a'+i))); err != nil {
			t.Fatalf("create record: %v", err)
		}
	}
	if err := CreateScheduleRecord(ctx, 2, "slug-2-a"); err != nil {
		t.Fatalf("create record: %v", err)
	}

	// schedule 1：total=3，倒序（最后插入的 id 最大排最前）
	rows, total, err := ListScheduleRecords(ctx, 1, 1, 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 3 || len(rows) != 3 {
		t.Fatalf("schedule 1: total=%d len=%d, want 3/3", total, len(rows))
	}
	if rows[0].ID < rows[1].ID {
		t.Fatalf("expected desc by id, got %d before %d", rows[0].ID, rows[1].ID)
	}

	// 分页：pageSize=2 → 第 1 页 2 条，第 2 页 1 条
	page1, total, _ := ListScheduleRecords(ctx, 1, 1, 2)
	if total != 3 || len(page1) != 2 {
		t.Fatalf("page1: total=%d len=%d, want 3/2", total, len(page1))
	}
	page2, _, _ := ListScheduleRecords(ctx, 1, 2, 2)
	if len(page2) != 1 {
		t.Fatalf("page2 len=%d, want 1", len(page2))
	}

	// schedule 2：total=1
	rows2, total2, _ := ListScheduleRecords(ctx, 2, 1, 10)
	if total2 != 1 || len(rows2) != 1 || rows2[0].DispatchSlug != "slug-2-a" {
		t.Fatalf("schedule 2 unexpected: total=%d rows=%+v", total2, rows2)
	}
}
