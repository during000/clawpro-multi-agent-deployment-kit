package model

import (
	"testing"
	"time"
)

// ========== dateOnlyUTC 测试 ==========

// TestDateOnlyUTC_Basic 测试基本的日期截断功能
func TestDateOnlyUTC_Basic(t *testing.T) {
	tests := []struct {
		name  string
		input time.Time
		want  time.Time
	}{
		{
			name:  "带时分秒的时间",
			input: time.Date(2024, 3, 15, 14, 30, 45, 123456789, time.UTC),
			want:  time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "已经是零时的时间",
			input: time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
			want:  time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "接近午夜的时间",
			input: time.Date(2024, 3, 15, 23, 59, 59, 999999999, time.UTC),
			want:  time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "非 UTC 时区（东八区）",
			input: time.Date(2024, 3, 16, 2, 0, 0, 0, time.FixedZone("CST", 8*3600)),
			// 东八区 3月16日 02:00 = UTC 3月15日 18:00，截断后为 UTC 3月15日
			want: time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "非 UTC 时区跨日（西部时区）",
			input: time.Date(2024, 3, 14, 20, 0, 0, 0, time.FixedZone("EST", -5*3600)),
			// EST 3月14日 20:00 = UTC 3月15日 01:00，截断后为 UTC 3月15日
			want: time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "零值时间",
			input: time.Time{},
			want:  time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dateOnlyUTC(tt.input)
			if !got.Equal(tt.want) {
				t.Errorf("dateOnlyUTC(%v) = %v, want %v", tt.input, got, tt.want)
			}
			// 验证结果始终为 UTC
			if got.Location() != time.UTC {
				t.Errorf("结果时区应为 UTC，实际为 %v", got.Location())
			}
			// 验证时分秒纳秒均为零
			if got.Hour() != 0 || got.Minute() != 0 || got.Second() != 0 || got.Nanosecond() != 0 {
				t.Errorf("时分秒纳秒应均为零，实际为 %02d:%02d:%02d.%09d",
					got.Hour(), got.Minute(), got.Second(), got.Nanosecond())
			}
		})
	}
}

// ========== normalizeMigratedDailyUsageSummaries 测试 ==========

// TestNormalize_EmptyInput 测试空输入
func TestNormalize_EmptyInput(t *testing.T) {
	result := normalizeMigratedDailyUsageSummaries(nil, "tenant-1")
	if len(result) != 0 {
		t.Errorf("空输入应返回空切片，实际长度为 %d", len(result))
	}

	result = normalizeMigratedDailyUsageSummaries([]DailyUsageSummary{}, "tenant-1")
	if len(result) != 0 {
		t.Errorf("空切片输入应返回空切片，实际长度为 %d", len(result))
	}
}

// TestNormalize_SingleRecord 测试单条记录
func TestNormalize_SingleRecord(t *testing.T) {
	records := []DailyUsageSummary{
		{
			ID:               100,
			Date:             time.Date(2024, 3, 15, 14, 30, 0, 0, time.UTC),
			UserID:           1,
			InstanceID:       2,
			AIModelID:        3,
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
			RequestCount:     5,
		},
	}

	result := normalizeMigratedDailyUsageSummaries(records, "tenant-1")
	if len(result) != 1 {
		t.Fatalf("应返回 1 条记录，实际为 %d", len(result))
	}

	r := result[0]
	// ID 应被清零
	if r.ID != 0 {
		t.Errorf("ID 应被清零，实际为 %d", r.ID)
	}
	// 日期应被截断
	expectedDate := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	if !r.Date.Equal(expectedDate) {
		t.Errorf("日期应被截断为 %v，实际为 %v", expectedDate, r.Date)
	}
	// Token 数据应保持不变
	if r.PromptTokens != 100 || r.CompletionTokens != 50 || r.TotalTokens != 150 || r.RequestCount != 5 {
		t.Errorf("Token 数据不应改变")
	}
}

// TestNormalize_MergesDuplicates 测试合并重复记录
func TestNormalize_MergesDuplicates(t *testing.T) {
	// 两条记录：同一天（但时间不同）、同一用户、同一实例、同一模型
	records := []DailyUsageSummary{
		{
			ID:               1,
			Date:             time.Date(2024, 3, 15, 10, 0, 0, 0, time.UTC),
			UserID:           1,
			InstanceID:       2,
			AIModelID:        3,
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
			RequestCount:     5,
		},
		{
			ID:               2,
			Date:             time.Date(2024, 3, 15, 22, 0, 0, 0, time.UTC),
			UserID:           1,
			InstanceID:       2,
			AIModelID:        3,
			PromptTokens:     200,
			CompletionTokens: 100,
			TotalTokens:      300,
			RequestCount:     10,
		},
	}

	result := normalizeMigratedDailyUsageSummaries(records, "tenant-1")
	if len(result) != 1 {
		t.Fatalf("重复记录应合并为 1 条，实际为 %d", len(result))
	}

	r := result[0]
	if r.PromptTokens != 300 {
		t.Errorf("PromptTokens 应为 300，实际为 %d", r.PromptTokens)
	}
	if r.CompletionTokens != 150 {
		t.Errorf("CompletionTokens 应为 150，实际为 %d", r.CompletionTokens)
	}
	if r.TotalTokens != 450 {
		t.Errorf("TotalTokens 应为 450，实际为 %d", r.TotalTokens)
	}
	if r.RequestCount != 15 {
		t.Errorf("RequestCount 应为 15，实际为 %d", r.RequestCount)
	}
}

// TestNormalize_DifferentDays 测试不同日期不合并
func TestNormalize_DifferentDays(t *testing.T) {
	records := []DailyUsageSummary{
		{
			ID:           1,
			Date:         time.Date(2024, 3, 15, 10, 0, 0, 0, time.UTC),
			UserID:       1,
			InstanceID:   2,
			AIModelID:    3,
			PromptTokens: 100,
			RequestCount: 5,
		},
		{
			ID:           2,
			Date:         time.Date(2024, 3, 16, 10, 0, 0, 0, time.UTC),
			UserID:       1,
			InstanceID:   2,
			AIModelID:    3,
			PromptTokens: 200,
			RequestCount: 10,
		},
	}

	result := normalizeMigratedDailyUsageSummaries(records, "tenant-1")
	if len(result) != 2 {
		t.Fatalf("不同日期不应合并，应为 2 条，实际为 %d", len(result))
	}
}

// TestNormalize_DifferentUsers 测试不同用户不合并
func TestNormalize_DifferentUsers(t *testing.T) {
	records := []DailyUsageSummary{
		{
			ID:           1,
			Date:         time.Date(2024, 3, 15, 10, 0, 0, 0, time.UTC),
			UserID:       1,
			InstanceID:   2,
			AIModelID:    3,
			PromptTokens: 100,
			RequestCount: 5,
		},
		{
			ID:           2,
			Date:         time.Date(2024, 3, 15, 22, 0, 0, 0, time.UTC),
			UserID:       99, // 不同用户
			InstanceID:   2,
			AIModelID:    3,
			PromptTokens: 200,
			RequestCount: 10,
		},
	}

	result := normalizeMigratedDailyUsageSummaries(records, "tenant-1")
	if len(result) != 2 {
		t.Fatalf("不同用户不应合并，应为 2 条，实际为 %d", len(result))
	}
}

// TestNormalize_DifferentInstances 测试不同实例不合并
func TestNormalize_DifferentInstances(t *testing.T) {
	records := []DailyUsageSummary{
		{
			ID:           1,
			Date:         time.Date(2024, 3, 15, 10, 0, 0, 0, time.UTC),
			UserID:       1,
			InstanceID:   2,
			AIModelID:    3,
			PromptTokens: 100,
			RequestCount: 5,
		},
		{
			ID:           2,
			Date:         time.Date(2024, 3, 15, 22, 0, 0, 0, time.UTC),
			UserID:       1,
			InstanceID:   99, // 不同实例
			AIModelID:    3,
			PromptTokens: 200,
			RequestCount: 10,
		},
	}

	result := normalizeMigratedDailyUsageSummaries(records, "tenant-1")
	if len(result) != 2 {
		t.Fatalf("不同实例不应合并，应为 2 条，实际为 %d", len(result))
	}
}

// TestNormalize_DifferentModels 测试不同模型不合并
func TestNormalize_DifferentModels(t *testing.T) {
	records := []DailyUsageSummary{
		{
			ID:           1,
			Date:         time.Date(2024, 3, 15, 10, 0, 0, 0, time.UTC),
			UserID:       1,
			InstanceID:   2,
			AIModelID:    3,
			PromptTokens: 100,
			RequestCount: 5,
		},
		{
			ID:           2,
			Date:         time.Date(2024, 3, 15, 22, 0, 0, 0, time.UTC),
			UserID:       1,
			InstanceID:   2,
			AIModelID:    99, // 不同模型
			PromptTokens: 200,
			RequestCount: 10,
		},
	}

	result := normalizeMigratedDailyUsageSummaries(records, "tenant-1")
	if len(result) != 2 {
		t.Fatalf("不同模型不应合并，应为 2 条，实际为 %d", len(result))
	}
}

// TestNormalize_MultipleGroups 测试多组合并
func TestNormalize_MultipleGroups(t *testing.T) {
	records := []DailyUsageSummary{
		// 组1：用户1 + 实例2 + 模型3 + 3月15日（3条合并为1条）
		{ID: 1, Date: time.Date(2024, 3, 15, 8, 0, 0, 0, time.UTC), UserID: 1, InstanceID: 2, AIModelID: 3, PromptTokens: 10, RequestCount: 1},
		{ID: 2, Date: time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC), UserID: 1, InstanceID: 2, AIModelID: 3, PromptTokens: 20, RequestCount: 2},
		{ID: 3, Date: time.Date(2024, 3, 15, 18, 0, 0, 0, time.UTC), UserID: 1, InstanceID: 2, AIModelID: 3, PromptTokens: 30, RequestCount: 3},
		// 组2：用户1 + 实例2 + 模型3 + 3月16日（2条合并为1条）
		{ID: 4, Date: time.Date(2024, 3, 16, 10, 0, 0, 0, time.UTC), UserID: 1, InstanceID: 2, AIModelID: 3, PromptTokens: 40, RequestCount: 4},
		{ID: 5, Date: time.Date(2024, 3, 16, 20, 0, 0, 0, time.UTC), UserID: 1, InstanceID: 2, AIModelID: 3, PromptTokens: 50, RequestCount: 5},
		// 组3：用户2 + 实例2 + 模型3 + 3月15日（1条不合并）
		{ID: 6, Date: time.Date(2024, 3, 15, 10, 0, 0, 0, time.UTC), UserID: 2, InstanceID: 2, AIModelID: 3, PromptTokens: 60, RequestCount: 6},
	}

	result := normalizeMigratedDailyUsageSummaries(records, "tenant-1")
	if len(result) != 3 {
		t.Fatalf("应合并为 3 组，实际为 %d", len(result))
	}

	// 验证组1合并结果
	found := false
	for _, r := range result {
		if r.UserID == 1 && r.Date.Equal(time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)) {
			found = true
			if r.PromptTokens != 60 {
				t.Errorf("组1 PromptTokens 应为 60，实际为 %d", r.PromptTokens)
			}
			if r.RequestCount != 6 {
				t.Errorf("组1 RequestCount 应为 6，实际为 %d", r.RequestCount)
			}
		}
	}
	if !found {
		t.Error("未找到组1的合并结果")
	}
}

// TestNormalize_CrossTimezoneDate 测试跨时区日期合并
func TestNormalize_CrossTimezoneDate(t *testing.T) {
	// 两条记录在不同时区下看起来是不同日期，但 UTC 日期相同
	cst := time.FixedZone("CST", 8*3600)
	records := []DailyUsageSummary{
		{
			ID:           1,
			Date:         time.Date(2024, 3, 15, 20, 0, 0, 0, time.UTC), // UTC 3月15日
			UserID:       1,
			InstanceID:   2,
			AIModelID:    3,
			PromptTokens: 100,
			RequestCount: 5,
		},
		{
			ID:           2,
			Date:         time.Date(2024, 3, 16, 2, 0, 0, 0, cst), // CST 3月16日 02:00 = UTC 3月15日 18:00
			UserID:       1,
			InstanceID:   2,
			AIModelID:    3,
			PromptTokens: 200,
			RequestCount: 10,
		},
	}

	result := normalizeMigratedDailyUsageSummaries(records, "tenant-1")
	if len(result) != 1 {
		t.Fatalf("UTC 同一天的记录应合并为 1 条，实际为 %d", len(result))
	}
	if result[0].PromptTokens != 300 {
		t.Errorf("PromptTokens 应为 300，实际为 %d", result[0].PromptTokens)
	}
}

// TestNormalize_SetsIdentifier 测试 identifier 参数设置
func TestNormalize_SetsIdentifier(t *testing.T) {
	records := []DailyUsageSummary{
		{
			ID:           1,
			Identifier:   "old-tenant",
			Date:         time.Date(2024, 3, 15, 10, 0, 0, 0, time.UTC),
			UserID:       1,
			InstanceID:   2,
			AIModelID:    3,
			PromptTokens: 100,
			RequestCount: 5,
		},
	}

	result := normalizeMigratedDailyUsageSummaries(records, "test-tenant")
	if len(result) != 1 {
		t.Fatalf("应返回 1 条记录，实际为 %d", len(result))
	}
	if result[0].Identifier != "test-tenant" {
		t.Errorf("Identifier 应为 'test-tenant'，实际为 '%s'", result[0].Identifier)
	}
}

// TestNormalize_EmptyIdentifier 测试 identifier 参数为空时不覆盖
func TestNormalize_EmptyIdentifier(t *testing.T) {
	records := []DailyUsageSummary{
		{
			ID:           1,
			Identifier:   "original",
			Date:         time.Date(2024, 3, 15, 10, 0, 0, 0, time.UTC),
			UserID:       1,
			InstanceID:   2,
			AIModelID:    3,
			PromptTokens: 100,
			RequestCount: 5,
		},
	}

	result := normalizeMigratedDailyUsageSummaries(records, "")
	if len(result) != 1 {
		t.Fatalf("应返回 1 条记录，实际为 %d", len(result))
	}
	if result[0].Identifier != "original" {
		t.Errorf("identifier 为空时不应覆盖，应为 'original'，实际为 '%s'", result[0].Identifier)
	}
}

// TestNormalize_IdentifierAffectsMerge 测试不同 Identifier 的记录不合并
func TestNormalize_IdentifierAffectsMerge(t *testing.T) {
	records := []DailyUsageSummary{
		{
			ID:           1,
			Identifier:   "tenant-a",
			Date:         time.Date(2024, 3, 15, 10, 0, 0, 0, time.UTC),
			UserID:       1,
			InstanceID:   2,
			AIModelID:    3,
			PromptTokens: 100,
			RequestCount: 5,
		},
		{
			ID:           2,
			Identifier:   "tenant-b",
			Date:         time.Date(2024, 3, 15, 22, 0, 0, 0, time.UTC),
			UserID:       1,
			InstanceID:   2,
			AIModelID:    3,
			PromptTokens: 200,
			RequestCount: 10,
		},
	}

	result := normalizeMigratedDailyUsageSummaries(records, "")
	if len(result) != 2 {
		t.Fatalf("不同 Identifier 不应合并，应为 2 条，实际为 %d", len(result))
	}
}

// ========== idMapping 测试 ==========

// TestIdMapping_Basic 测试 idMapping 的基本功能
func TestIdMapping_Basic(t *testing.T) {
	ids := make(idMapping)

	// 设置映射
	ids.set("users", 1, 100)
	ids.set("users", 2, 200)
	ids.set("instances", 1, 300)

	// 获取映射
	if got := ids.get("users", 1); got != 100 {
		t.Errorf("users[1] 应为 100，实际为 %d", got)
	}
	if got := ids.get("users", 2); got != 200 {
		t.Errorf("users[2] 应为 200，实际为 %d", got)
	}
	if got := ids.get("instances", 1); got != 300 {
		t.Errorf("instances[1] 应为 300，实际为 %d", got)
	}
}

// TestIdMapping_ZeroID 测试 ID 为 0 的特殊处理
func TestIdMapping_ZeroID(t *testing.T) {
	ids := make(idMapping)

	// oldID 为 0 应直接返回 0
	if got := ids.get("users", 0); got != 0 {
		t.Errorf("oldID 为 0 应返回 0，实际为 %d", got)
	}
}

// TestIdMapping_NotFound 测试未找到映射时返回 0
func TestIdMapping_NotFound(t *testing.T) {
	// 重置 missingIDs 避免影响其他测试
	oldMissing := missingIDs
	missingIDs = make(map[string]map[uint]bool)
	defer func() { missingIDs = oldMissing }()

	ids := make(idMapping)
	ids.set("users", 1, 100)

	// 不存在的 ID
	if got := ids.get("users", 999); got != 0 {
		t.Errorf("不存在的 ID 应返回 0，实际为 %d", got)
	}

	// 不存在的表
	if got := ids.get("nonexistent", 1); got != 0 {
		t.Errorf("不存在的表应返回 0，实际为 %d", got)
	}
}

// TestIdMapping_MissingIDDedup 测试缺失 ID 日志去重
func TestIdMapping_MissingIDDedup(t *testing.T) {
	// 重置 missingIDs
	oldMissing := missingIDs
	missingIDs = make(map[string]map[uint]bool)
	defer func() { missingIDs = oldMissing }()

	ids := make(idMapping)

	// 第一次查找缺失 ID
	ids.get("users", 999)

	// 验证 missingIDs 已记录
	if missingIDs["users"] == nil || !missingIDs["users"][999] {
		t.Error("missingIDs 应记录 users[999]")
	}

	// 第二次查找同一个缺失 ID（不应重复记录日志，但 missingIDs 中已存在）
	ids.get("users", 999)

	// 验证仍然只有一条记录
	if !missingIDs["users"][999] {
		t.Error("missingIDs 应仍包含 users[999]")
	}
}

// TestIdMapping_OverwriteMapping 测试覆盖映射
func TestIdMapping_OverwriteMapping(t *testing.T) {
	ids := make(idMapping)

	ids.set("users", 1, 100)
	if got := ids.get("users", 1); got != 100 {
		t.Errorf("初始映射应为 100，实际为 %d", got)
	}

	// 覆盖
	ids.set("users", 1, 200)
	if got := ids.get("users", 1); got != 200 {
		t.Errorf("覆盖后应为 200，实际为 %d", got)
	}
}
