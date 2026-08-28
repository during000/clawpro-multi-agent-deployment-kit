package model

import (
	"testing"
	"time"
)

func TestDateOnlyUTC(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected time.Time
	}{
		{
			name:     "already UTC midnight",
			input:    time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC),
			expected: time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "UTC with time portion",
			input:    time.Date(2026, 4, 15, 13, 30, 45, 123456789, time.UTC),
			expected: time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "non-UTC timezone (CST +8) same day in UTC",
			input:    time.Date(2026, 4, 15, 6, 0, 0, 0, time.FixedZone("CST", 8*3600)),
			expected: time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC), // 06:00 CST = 22:00 UTC prev day
		},
		{
			name:     "non-UTC timezone crosses date boundary",
			input:    time.Date(2026, 4, 15, 23, 59, 59, 0, time.FixedZone("EST", -5*3600)),
			expected: time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC), // 23:59 EST = 04:59 next day UTC
		},
		{
			name:     "zero time",
			input:    time.Time{},
			expected: time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "year boundary",
			input:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			expected: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "leap year Feb 29",
			input:    time.Date(2024, 2, 29, 15, 30, 0, 0, time.UTC),
			expected: time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dateOnlyUTC(tt.input)
			if !result.Equal(tt.expected) {
				t.Errorf("dateOnlyUTC(%v) = %v, want %v", tt.input, result, tt.expected)
			}
			// 验证返回值始终是 UTC
			if result.Location() != time.UTC {
				t.Errorf("dateOnlyUTC() location = %v, want UTC", result.Location())
			}
			// 验证时间部分为零
			if result.Hour() != 0 || result.Minute() != 0 || result.Second() != 0 || result.Nanosecond() != 0 {
				t.Errorf("dateOnlyUTC() has non-zero time portion: %v", result)
			}
		})
	}
}

func TestNormalizeMigratedDailyUsageSummaries_EmptyInput(t *testing.T) {
	result := normalizeMigratedDailyUsageSummaries(nil, "tenant-1")
	if len(result) != 0 {
		t.Errorf("expected empty result for nil input, got %d records", len(result))
	}

	result = normalizeMigratedDailyUsageSummaries([]DailyUsageSummary{}, "tenant-1")
	if len(result) != 0 {
		t.Errorf("expected empty result for empty input, got %d records", len(result))
	}
}

func TestNormalizeMigratedDailyUsageSummaries_SingleRecord(t *testing.T) {
	input := []DailyUsageSummary{
		{
			ID:               42,
			Date:             time.Date(2026, 4, 15, 13, 30, 0, 0, time.UTC),
			UserID:           1,
			InstanceID:       2,
			AIModelID:        3,
			PromptTokens:     100,
			CompletionTokens: 200,
			TotalTokens:      300,
			RequestCount:     5,
		},
	}

	result := normalizeMigratedDailyUsageSummaries(input, "tenant-1")

	if len(result) != 1 {
		t.Fatalf("expected 1 record, got %d", len(result))
	}

	r := result[0]
	// ID 应该被清零
	if r.ID != 0 {
		t.Errorf("ID should be reset to 0, got %d", r.ID)
	}
	// Date 应该被截断到 UTC 日期
	expected := time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)
	if !r.Date.Equal(expected) {
		t.Errorf("Date = %v, want %v", r.Date, expected)
	}
	// 其他字段保持不变
	if r.PromptTokens != 100 || r.CompletionTokens != 200 || r.TotalTokens != 300 || r.RequestCount != 5 {
		t.Errorf("token counts should be unchanged, got prompt=%d completion=%d total=%d request=%d",
			r.PromptTokens, r.CompletionTokens, r.TotalTokens, r.RequestCount)
	}
}

func TestNormalizeMigratedDailyUsageSummaries_MergesDuplicates(t *testing.T) {
	// 两条记录具有相同的 key（日期+用户+实例+模型），但时间不同
	input := []DailyUsageSummary{
		{
			ID:               1,
			Date:             time.Date(2026, 4, 15, 8, 0, 0, 0, time.UTC),
			UserID:           1,
			InstanceID:       2,
			AIModelID:        3,
			PromptTokens:     100,
			CompletionTokens: 200,
			TotalTokens:      300,
			RequestCount:     5,
		},
		{
			ID:               2,
			Date:             time.Date(2026, 4, 15, 16, 30, 0, 0, time.UTC),
			UserID:           1,
			InstanceID:       2,
			AIModelID:        3,
			PromptTokens:     50,
			CompletionTokens: 100,
			TotalTokens:      150,
			RequestCount:     3,
		},
	}

	result := normalizeMigratedDailyUsageSummaries(input, "tenant-1")

	if len(result) != 1 {
		t.Fatalf("expected 1 merged record, got %d", len(result))
	}

	r := result[0]
	if r.PromptTokens != 150 {
		t.Errorf("PromptTokens = %d, want 150", r.PromptTokens)
	}
	if r.CompletionTokens != 300 {
		t.Errorf("CompletionTokens = %d, want 300", r.CompletionTokens)
	}
	if r.TotalTokens != 450 {
		t.Errorf("TotalTokens = %d, want 450", r.TotalTokens)
	}
	if r.RequestCount != 8 {
		t.Errorf("RequestCount = %d, want 8", r.RequestCount)
	}
}

func TestNormalizeMigratedDailyUsageSummaries_DifferentKeysNotMerged(t *testing.T) {
	input := []DailyUsageSummary{
		{
			Date:         time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC),
			UserID:       1,
			InstanceID:   2,
			AIModelID:    3,
			RequestCount: 1,
		},
		{
			Date:         time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC),
			UserID:       2, // 不同用户
			InstanceID:   2,
			AIModelID:    3,
			RequestCount: 1,
		},
		{
			Date:         time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC), // 不同日期
			UserID:       1,
			InstanceID:   2,
			AIModelID:    3,
			RequestCount: 1,
		},
		{
			Date:         time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC),
			UserID:       1,
			InstanceID:   5, // 不同实例
			AIModelID:    3,
			RequestCount: 1,
		},
		{
			Date:         time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC),
			UserID:       1,
			InstanceID:   2,
			AIModelID:    7, // 不同模型
			RequestCount: 1,
		},
	}

	result := normalizeMigratedDailyUsageSummaries(input, "tenant-1")

	if len(result) != 5 {
		t.Errorf("expected 5 distinct records, got %d", len(result))
	}
}

func TestNormalizeMigratedDailyUsageSummaries_MergeWithThreeRecords(t *testing.T) {
	// 3条相同key的记录应该合并为1条
	input := []DailyUsageSummary{
		{Date: time.Date(2026, 4, 15, 1, 0, 0, 0, time.UTC), UserID: 1, InstanceID: 1, AIModelID: 1, PromptTokens: 10, CompletionTokens: 20, TotalTokens: 30, RequestCount: 1},
		{Date: time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC), UserID: 1, InstanceID: 1, AIModelID: 1, PromptTokens: 30, CompletionTokens: 40, TotalTokens: 70, RequestCount: 2},
		{Date: time.Date(2026, 4, 15, 23, 0, 0, 0, time.UTC), UserID: 1, InstanceID: 1, AIModelID: 1, PromptTokens: 60, CompletionTokens: 40, TotalTokens: 100, RequestCount: 3},
	}

	result := normalizeMigratedDailyUsageSummaries(input, "tenant-1")

	if len(result) != 1 {
		t.Fatalf("expected 1 merged record, got %d", len(result))
	}

	r := result[0]
	if r.PromptTokens != 100 {
		t.Errorf("PromptTokens = %d, want 100", r.PromptTokens)
	}
	if r.CompletionTokens != 100 {
		t.Errorf("CompletionTokens = %d, want 100", r.CompletionTokens)
	}
	if r.TotalTokens != 200 {
		t.Errorf("TotalTokens = %d, want 200", r.TotalTokens)
	}
	if r.RequestCount != 6 {
		t.Errorf("RequestCount = %d, want 6", r.RequestCount)
	}
}

func TestNormalizeMigratedDailyUsageSummaries_IDsResetToZero(t *testing.T) {
	input := []DailyUsageSummary{
		{ID: 100, Date: time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC), UserID: 1, InstanceID: 1, AIModelID: 1},
		{ID: 200, Date: time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC), UserID: 1, InstanceID: 1, AIModelID: 1},
	}

	result := normalizeMigratedDailyUsageSummaries(input, "tenant-1")

	for i, r := range result {
		if r.ID != 0 {
			t.Errorf("result[%d].ID = %d, want 0", i, r.ID)
		}
	}
}

// ==================== idMapping Tests ====================

func TestIdMapping_SetAndGet(t *testing.T) {
	m := make(idMapping)
	m.set("users", 1, 100)
	m.set("users", 2, 200)
	m.set("instances", 10, 1000)

	if got := m.get("users", 1); got != 100 {
		t.Errorf("get(users, 1) = %d, want 100", got)
	}
	if got := m.get("users", 2); got != 200 {
		t.Errorf("get(users, 2) = %d, want 200", got)
	}
	if got := m.get("instances", 10); got != 1000 {
		t.Errorf("get(instances, 10) = %d, want 1000", got)
	}
}

func TestIdMapping_GetZeroReturnsZero(t *testing.T) {
	m := make(idMapping)
	// oldID=0 应该直接返回 0，不查表
	if got := m.get("users", 0); got != 0 {
		t.Errorf("get(users, 0) = %d, want 0", got)
	}
}

func TestIdMapping_GetNotFound(t *testing.T) {
	// 清理全局 missingIDs 避免测试互相干扰
	missingIDs = make(map[string]map[uint]bool)

	m := make(idMapping)
	m.set("users", 1, 100)

	// 查询不存在的ID应返回0
	if got := m.get("users", 999); got != 0 {
		t.Errorf("get(users, 999) = %d, want 0", got)
	}

	// 查询不存在的表应返回0
	if got := m.get("nonexistent", 1); got != 0 {
		t.Errorf("get(nonexistent, 1) = %d, want 0", got)
	}
}

func TestIdMapping_GetDeduplicatesWarnings(t *testing.T) {
	// 清理全局 missingIDs
	missingIDs = make(map[string]map[uint]bool)

	m := make(idMapping)

	// 多次查询相同的不存在ID，只应记录一次
	_ = m.get("test_table", 42)
	_ = m.get("test_table", 42)
	_ = m.get("test_table", 42)

	// missingIDs 中应该记录了这个缺失
	if !missingIDs["test_table"][42] {
		t.Error("expected missingIDs to track test_table/42")
	}
}
