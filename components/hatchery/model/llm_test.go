package model

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	hcommon "hatchery/common"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var llmTestDBCounter int64

// setupLLMTestDB creates an isolated SQLite memory database for LLM testing
func setupLLMTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	id := atomic.AddInt64(&llmTestDBCounter, 1)
	dsn := fmt.Sprintf("file:llmTest%d?mode=memory&cache=shared", id)
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite mem: %v", err)
	}
	if err := gdb.AutoMigrate(&DailyUsageSummary{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	t.Cleanup(UseDBForTest(gdb))
	return gdb
}

// TestModelDailyTokenUsage tests the ModelDailyTokenUsage query function
func TestModelDailyTokenUsage(t *testing.T) {
	db := setupLLMTestDB(t)

	today := LocalToday()

	// Create a test record with known tokens
	record := &DailyUsageSummary{
		AIModelID:        1,
		Date:             today,
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}
	if err := db.Create(record).Error; err != nil {
		t.Fatalf("create record: %v", err)
	}

	// Query for today's usage
	total := ModelDailyTokenUsage(context.Background(), 1)
	if total != 150 {
		t.Fatalf("expected 150 tokens, got %d", total)
	}

	// Query for different model (should be 0)
	total = ModelDailyTokenUsage(context.Background(), 999)
	if total != 0 {
		t.Fatalf("expected 0 tokens for different model, got %d", total)
	}
}

// TestUserDailyUsageByInstance tests the UserDailyUsageByInstance query function
func TestUserDailyUsageByInstance(t *testing.T) {
	db := setupLLMTestDB(t)

	today := LocalToday()

	// Create test records for same user, different instances
	records := []DailyUsageSummary{
		{
			UserID:           10,
			InstanceID:       100,
			AIModelID:        1,
			Date:             today,
			PromptTokens:     50,
			CompletionTokens: 30,
			TotalTokens:      80,
		},
		{
			UserID:           10,
			InstanceID:       101,
			AIModelID:        1,
			Date:             today,
			PromptTokens:     100,
			CompletionTokens: 60,
			TotalTokens:      160,
		},
	}

	for _, r := range records {
		r_copy := r // Make a copy to avoid pointer issues
		if err := db.Create(&r_copy).Error; err != nil {
			t.Fatalf("create record: %v", err)
		}
	}

	// Query usage by instance for this user today
	rows := UserDailyUsageByInstance(context.Background(), 10)
	if len(rows) != 2 {
		t.Fatalf("expected 2 instances, got %d", len(rows))
	}

	// Verify we got both instances
	instances := make(map[uint]UsageRow)
	for _, row := range rows {
		instances[row.InstanceID] = row
	}

	if _, ok := instances[100]; !ok {
		t.Fatalf("missing instance 100")
	}
	if _, ok := instances[101]; !ok {
		t.Fatalf("missing instance 101")
	}

	// Verify totals for instance 101
	row101 := instances[101]
	if row101.TotalTokens != 160 {
		t.Fatalf("expected 160 total tokens for instance 101, got %d", row101.TotalTokens)
	}
}

// TestUserDailyUsageByInstance_EmptyResult tests when user has no usage today
func TestUserDailyUsageByInstance_EmptyResult(t *testing.T) {
	setupLLMTestDB(t)

	// Query for a user with no records
	rows := UserDailyUsageByInstance(context.Background(), 999)
	if rows != nil && len(rows) > 0 {
		t.Fatalf("expected empty result for unknown user, got %d rows", len(rows))
	}
}

// TestModelDailyTokenUsage_MultipleRecords tests aggregation with multiple records for same model
func TestModelDailyTokenUsage_MultipleRecords(t *testing.T) {
	gdb := setupLLMTestDB(t)

	today := LocalToday()

	// Create multiple records for same model, same day, but different users
	for i := 0; i < 3; i++ {
		record := &DailyUsageSummary{
			UserID:           uint(10 + i),
			AIModelID:        1,
			Date:             today,
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		}
		if err := gdb.Create(record).Error; err != nil {
			t.Fatalf("create record %d: %v", i, err)
		}
	}

	// Total should be 3 * 150 = 450
	total := ModelDailyTokenUsage(context.Background(), 1)
	if total != 450 {
		t.Fatalf("expected 450 tokens from 3 records, got %d", total)
	}
}

// TestUserDailyUsageByInstance_PastDates tests that past dates are not included
func TestUserDailyUsageByInstance_PastDates(t *testing.T) {
	gdb := setupLLMTestDB(t)

	today := LocalToday()
	yesterday := today.AddDate(0, 0, -1)

	// Create record from yesterday
	record := &DailyUsageSummary{
		UserID:           10,
		InstanceID:       100,
		AIModelID:        1,
		Date:             yesterday,
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}
	if err := gdb.Create(record).Error; err != nil {
		t.Fatalf("create record: %v", err)
	}

	// Query today's usage should return empty
	rows := UserDailyUsageByInstance(context.Background(), 10)
	if rows != nil && len(rows) > 0 {
		t.Fatalf("expected no rows for today, got %d", len(rows))
	}

	// Create today's record
	record2 := &DailyUsageSummary{
		UserID:           10,
		InstanceID:       100,
		AIModelID:        1,
		Date:             today,
		PromptTokens:     50,
		CompletionTokens: 25,
		TotalTokens:      75,
	}
	if err := gdb.Create(record2).Error; err != nil {
		t.Fatalf("create today record: %v", err)
	}

	// Now should have 1 row for today
	rows = UserDailyUsageByInstance(context.Background(), 10)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row for today, got %d", len(rows))
	}
	if rows[0].TotalTokens != 75 {
		t.Fatalf("expected 75 tokens today, got %d", rows[0].TotalTokens)
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Token Quota Rules 测试
// ══════════════════════════════════════════════════════════════════════════════

func TestParseTokenQuotaRules(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantLen  int
		wantOK   bool
		wantMode string
	}{
		{"empty", "", 0, false, ""},
		{"invalid json", "not json", 0, false, ""},
		{"null", "null", 0, true, ""},
		{"empty array", "[]", 0, true, ""},
		{"single day", `[{"mode":"day","limit":100000}]`, 1, true, "day"},
		{"single month", `[{"mode":"month","limit":500000}]`, 1, true, "month"},
		{"single year", `[{"mode":"year","limit":5000000}]`, 1, true, "year"},
		{"custom with refresh", `[{"mode":"custom","limit":100000,"start":1747562460,"refresh":"monthly"}]`, 1, true, "custom"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules, ok := ParseTokenQuotaRules(tt.input)
			if ok != tt.wantOK {
				t.Fatalf("got ok=%v, want %v", ok, tt.wantOK)
			}
			if len(rules) != tt.wantLen {
				t.Fatalf("got %d rules, want %d", len(rules), tt.wantLen)
			}
			if tt.wantLen > 0 && rules[0].Mode != tt.wantMode {
				t.Fatalf("got mode=%s, want %s", rules[0].Mode, tt.wantMode)
			}
		})
	}
}

func TestMarshalTokenQuotaRules(t *testing.T) {
	// empty slice → "[]"
	if got := MarshalTokenQuotaRules([]TokenQuotaRule{}); got != "[]" {
		t.Fatalf("expected \"[]\", got %q", got)
	}
	// roundtrip
	rules := []TokenQuotaRule{{Mode: QuotaModeDay, Limit: 100000}}
	marshaled := MarshalTokenQuotaRules(rules)
	parsed, _ := ParseTokenQuotaRules(marshaled)
	if len(parsed) != 1 || parsed[0].Mode != QuotaModeDay || parsed[0].Limit != 100000 {
		t.Fatalf("roundtrip failed: %v", parsed)
	}
}

func TestQuotaWindow_Day(t *testing.T) {
	rule := TokenQuotaRule{Mode: QuotaModeDay, Limit: 100}
	from, to, active := rule.QuotaWindow(LocalToday().Add(12 * time.Hour))
	if !active {
		t.Fatal("day rule should be active")
	}
	if to == nil {
		t.Fatal("day rule should have an end")
	}
	if to.Sub(from) != 24*time.Hour {
		t.Fatalf("day window should be 24h, got %v", to.Sub(from))
	}
}

func TestQuotaWindow_Month(t *testing.T) {
	rule := TokenQuotaRule{Mode: QuotaModeMonth, Limit: 1000000}
	from, to, active := rule.QuotaWindow(time.Now())
	if !active {
		t.Fatal("month rule should be active")
	}
	if to == nil {
		t.Fatal("month rule should have an end")
	}
	if from.After(*to) {
		t.Fatal("from should be before to")
	}
	// month window should be 28-31 days
	days := to.Sub(from).Hours() / 24
	if days < 28 || days > 31 {
		t.Fatalf("month window should be 28-31 days, got %.1f", days)
	}
}

func TestQuotaWindow_Year(t *testing.T) {
	loc := hcommon.BusinessLocation()
	rule := TokenQuotaRule{Mode: QuotaModeYear, Limit: 10000000}
	now := time.Date(2026, 6, 4, 12, 0, 0, 0, loc)

	from, to, active := rule.QuotaWindow(now)
	if !active {
		t.Fatal("year rule should be active")
	}
	expectedFrom := time.Date(2026, 1, 1, 0, 0, 0, 0, loc).UTC()
	expectedTo := time.Date(2027, 1, 1, 0, 0, 0, 0, loc).UTC()
	if !from.Equal(expectedFrom) {
		t.Fatalf("from should be %v, got %v", expectedFrom, from)
	}
	if to == nil || !to.Equal(expectedTo) {
		t.Fatalf("to should be %v, got %v", expectedTo, to)
	}
}

func TestQuotaWindow_Custom_RefreshNone(t *testing.T) {
	start := time.Now().Add(-1 * time.Hour).Unix()
	end := time.Now().Add(24 * time.Hour).Unix()
	rule := TokenQuotaRule{Mode: QuotaModeCustom, Limit: 500, Start: &start, End: &end, Refresh: QuotaRefreshNone}

	from, to, active := rule.QuotaWindow(time.Now())
	if !active {
		t.Fatal("should be active (within window)")
	}
	if from.Unix() != start {
		t.Fatalf("from should equal start")
	}
	if to == nil || to.Unix() != end {
		t.Fatalf("to should equal end")
	}
}

func TestQuotaWindow_Custom_RefreshNone_OpenEnded(t *testing.T) {
	start := time.Now().Add(-1 * time.Hour).Unix()
	rule := TokenQuotaRule{Mode: QuotaModeCustom, Limit: 500, Start: &start, Refresh: QuotaRefreshNone}

	from, to, active := rule.QuotaWindow(time.Now())
	if !active {
		t.Fatal("should be active (open-ended window)")
	}
	if from.Unix() != start {
		t.Fatalf("from should equal start")
	}
	if to != nil {
		t.Fatalf("to should be nil for open-ended window, got %v", to)
	}
}

func TestQuotaWindow_Custom_RefreshNone_Expired(t *testing.T) {
	start := time.Now().Add(-48 * time.Hour).Unix()
	end := time.Now().Add(-1 * time.Hour).Unix()
	rule := TokenQuotaRule{Mode: QuotaModeCustom, Limit: 500, Start: &start, End: &end, Refresh: QuotaRefreshNone}

	_, _, active := rule.QuotaWindow(time.Now())
	if active {
		t.Fatal("should be inactive (expired)")
	}
}

func TestQuotaWindow_Custom_NotStarted(t *testing.T) {
	start := time.Now().Add(1 * time.Hour).Unix()
	rule := TokenQuotaRule{Mode: QuotaModeCustom, Limit: 500, Start: &start, Refresh: QuotaRefreshDaily}

	_, _, active := rule.QuotaWindow(time.Now())
	if active {
		t.Fatal("should be inactive (not started)")
	}
}

func TestQuotaWindow_Custom_RefreshDaily(t *testing.T) {
	// Anchor at 2 hours ago
	start := time.Now().Add(-26 * time.Hour).Unix()
	rule := TokenQuotaRule{Mode: QuotaModeCustom, Limit: 500, Start: &start, Refresh: QuotaRefreshDaily}

	from, to, active := rule.QuotaWindow(time.Now())
	if !active {
		t.Fatal("should be active")
	}
	if to == nil {
		t.Fatal("daily refresh rule should have an end")
	}
	if to.Sub(from) != 24*time.Hour {
		t.Fatalf("daily refresh window should be 24h, got %v", to.Sub(from))
	}
}

func TestQuotaWindow_Custom_RefreshDaily_CapsAtEnd(t *testing.T) {
	loc := hcommon.BusinessLocation()
	anchor := time.Date(2026, 6, 1, 10, 0, 0, 0, loc)
	end := time.Date(2026, 6, 2, 12, 0, 0, 0, loc).Unix()
	rule := TokenQuotaRule{Mode: QuotaModeCustom, Limit: 100, Start: ptrInt64(anchor.Unix()), End: &end, Refresh: QuotaRefreshDaily}

	now := time.Date(2026, 6, 2, 11, 0, 0, 0, loc)
	from, to, active := rule.QuotaWindow(now)
	if !active {
		t.Fatal("should be active")
	}
	expectedFrom := time.Date(2026, 6, 2, 10, 0, 0, 0, loc).UTC()
	expectedTo := time.Date(2026, 6, 2, 12, 0, 0, 0, loc).UTC()
	if !from.Equal(expectedFrom) {
		t.Fatalf("from should be %v, got %v", expectedFrom, from)
	}
	if to == nil || !to.Equal(expectedTo) {
		t.Fatalf("to should be capped at hard end %v, got %v", expectedTo, to)
	}
}

func TestQuotaWindow_Custom_RefreshMonthly(t *testing.T) {
	// Anchor at 2 months ago
	start := time.Now().AddDate(0, -2, 0).Unix()
	rule := TokenQuotaRule{Mode: QuotaModeCustom, Limit: 500000, Start: &start, Refresh: QuotaRefreshMonthly}

	from, to, active := rule.QuotaWindow(time.Now())
	if !active {
		t.Fatal("should be active")
	}
	if to == nil {
		t.Fatal("monthly refresh rule should have an end")
	}
	// Window should be roughly a month
	days := to.Sub(from).Hours() / 24
	if days < 28 || days > 31 {
		t.Fatalf("monthly refresh window should be 28-31 days, got %.1f", days)
	}
}

func TestCheckUserTokenQuota_NilRules(t *testing.T) {
	setupLLMTestDB(t)
	exceeded, _, _ := CheckUserTokenQuota(context.Background(), 1, 0, nil)
	if exceeded {
		t.Fatal("nil rules should mean unlimited")
	}
}

func TestCheckUserTokenQuota_UnlimitedRule(t *testing.T) {
	setupLLMTestDB(t)
	rules := []TokenQuotaRule{{Mode: QuotaModeDay, Limit: -1}}
	exceeded, _, _ := CheckUserTokenQuota(context.Background(), 1, 0, rules)
	if exceeded {
		t.Fatal("limit=-1 should mean unlimited")
	}
}

func TestCheckUserTokenQuota_Exceeded(t *testing.T) {
	gdb := setupLLMTestDB(t)
	gdb.AutoMigrate(&LLMUsageLog{})

	// Insert usage logs for today
	now := time.Now().UTC()
	for i := 0; i < 5; i++ {
		gdb.Create(&LLMUsageLog{
			UserID:      42,
			TotalTokens: 200,
			CreatedAt:   now.Add(-time.Duration(i) * time.Minute),
		})
	}

	rules := []TokenQuotaRule{{Mode: QuotaModeDay, Limit: 500}}
	exceeded, used, limit := CheckUserTokenQuota(context.Background(), 42, 0, rules)
	if !exceeded {
		t.Fatalf("should be exceeded: used=%d limit=%d", used, limit)
	}
	if used != 1000 {
		t.Fatalf("expected used=1000, got %d", used)
	}
	if limit != 500 {
		t.Fatalf("expected limit=500, got %d", limit)
	}
}

func TestCheckUserTokenQuota_NotExceeded(t *testing.T) {
	gdb := setupLLMTestDB(t)
	gdb.AutoMigrate(&LLMUsageLog{})

	now := time.Now().UTC()
	gdb.Create(&LLMUsageLog{
		UserID:      43,
		TotalTokens: 100,
		CreatedAt:   now,
	})

	rules := []TokenQuotaRule{{Mode: QuotaModeDay, Limit: 500}}
	exceeded, _, _ := CheckUserTokenQuota(context.Background(), 43, 0, rules)
	if exceeded {
		t.Fatal("should not be exceeded")
	}
}

func TestCheckUserTokenQuota_YearModeExceeded(t *testing.T) {
	gdb := setupLLMTestDB(t)
	gdb.AutoMigrate(&LLMUsageLog{})

	rule := TokenQuotaRule{Mode: QuotaModeYear, Limit: 500}
	from, _, active := rule.QuotaWindow(time.Now())
	if !active {
		t.Fatal("year rule should be active")
	}
	gdb.Create(&LLMUsageLog{UserID: 44, TotalTokens: 600, CreatedAt: from.Add(time.Hour)})
	gdb.Create(&LLMUsageLog{UserID: 44, TotalTokens: 999, CreatedAt: from.Add(-time.Hour)})

	exceeded, used, limit := CheckUserTokenQuota(context.Background(), 44, 0, []TokenQuotaRule{rule})
	if !exceeded {
		t.Fatalf("year rule should be exceeded: used=%d limit=%d", used, limit)
	}
	if used != 600 {
		t.Fatalf("expected current year used=600, got %d", used)
	}
	if limit != 500 {
		t.Fatalf("expected limit=500, got %d", limit)
	}
}

func TestCheckGlobalTokenQuota_CustomYearlyExceeded(t *testing.T) {
	gdb := setupLLMTestDB(t)
	gdb.AutoMigrate(&LLMUsageLog{})

	start := time.Now().Add(-400 * 24 * time.Hour).Unix()
	rule := TokenQuotaRule{Mode: QuotaModeCustom, Limit: 500, Start: &start, Refresh: QuotaRefreshYearly}
	from, _, active := rule.QuotaWindow(time.Now())
	if !active {
		t.Fatal("custom yearly rule should be active")
	}
	gdb.Create(&LLMUsageLog{TotalTokens: 600, CreatedAt: from.Add(time.Hour)})
	gdb.Create(&LLMUsageLog{TotalTokens: 999, CreatedAt: from.Add(-time.Hour)})

	exceeded, used, limit := CheckGlobalTokenQuota(context.Background(), 0, []TokenQuotaRule{rule})
	if !exceeded {
		t.Fatalf("custom yearly global rule should be exceeded: used=%d limit=%d", used, limit)
	}
	if used != 600 {
		t.Fatalf("expected current yearly window used=600, got %d", used)
	}
	if limit != 500 {
		t.Fatalf("expected limit=500, got %d", limit)
	}
}

func TestUserTokenUsageInWindow(t *testing.T) {
	gdb := setupLLMTestDB(t)
	gdb.AutoMigrate(&LLMUsageLog{})

	now := time.Now().UTC()
	// Insert 3 logs: 2 in window, 1 outside
	gdb.Create(&LLMUsageLog{UserID: 50, TotalTokens: 100, CreatedAt: now.Add(-1 * time.Hour)})
	gdb.Create(&LLMUsageLog{UserID: 50, TotalTokens: 200, CreatedAt: now.Add(-30 * time.Minute)})
	gdb.Create(&LLMUsageLog{UserID: 50, TotalTokens: 999, CreatedAt: now.Add(-3 * time.Hour)})

	// Window: 2 hours ago to now
	from := now.Add(-2 * time.Hour)
	to := now.Add(1 * time.Minute)
	total := UserTokenUsageInWindow(context.Background(), 50, 0, from, &to)
	if total != 300 {
		t.Fatalf("expected 300 (100+200), got %d", total)
	}
}

func TestUserTokenUsageInOpenEndedWindow(t *testing.T) {
	gdb := setupLLMTestDB(t)
	gdb.AutoMigrate(&LLMUsageLog{})

	from := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)
	gdb.Create(&LLMUsageLog{UserID: 51, TotalTokens: 15874, CreatedAt: from.Add(time.Minute)})
	gdb.Create(&LLMUsageLog{UserID: 51, TotalTokens: 999, CreatedAt: from.Add(-time.Hour)})

	total := UserTokenUsageInWindow(context.Background(), 51, 0, from, nil)
	if total != 15874 {
		t.Fatalf("expected open-ended usage 15874, got %d", total)
	}
}

func TestGlobalAndGroupTokenUsageInWindow(t *testing.T) {
	gdb := setupLLMTestDB(t)
	gdb.AutoMigrate(&LLMUsageLog{})

	now := time.Now().UTC()
	gdb.Create(&LLMUsageLog{GroupID: 1, TotalTokens: 100, CreatedAt: now.Add(-time.Hour)})
	gdb.Create(&LLMUsageLog{GroupID: 1, TotalTokens: 200, CreatedAt: now.Add(-30 * time.Minute)})
	gdb.Create(&LLMUsageLog{GroupID: 2, TotalTokens: 300, CreatedAt: now.Add(-15 * time.Minute)})
	gdb.Create(&LLMUsageLog{GroupID: 1, TotalTokens: 999, CreatedAt: now.Add(-3 * time.Hour)})

	from := now.Add(-2 * time.Hour)
	to := now.Add(time.Minute)
	if got := GlobalTokenUsageInWindow(context.Background(), from, &to); got != 600 {
		t.Fatalf("expected global usage 600, got %d", got)
	}
	if got := GroupTokenUsageInWindow(context.Background(), 1, from, &to); got != 300 {
		t.Fatalf("expected group usage 300, got %d", got)
	}
}

func TestCheckGlobalTokenQuota_GroupExceeded(t *testing.T) {
	gdb := setupLLMTestDB(t)
	gdb.AutoMigrate(&LLMUsageLog{})

	now := time.Now().UTC()
	gdb.Create(&LLMUsageLog{GroupID: 7, TotalTokens: 250, CreatedAt: now})
	gdb.Create(&LLMUsageLog{GroupID: 8, TotalTokens: 1000, CreatedAt: now})

	rules := []TokenQuotaRule{{Mode: QuotaModeDay, Limit: 200}}
	exceeded, used, limit := CheckGlobalTokenQuota(context.Background(), 7, rules)
	if !exceeded {
		t.Fatal("expected group quota to be exceeded")
	}
	if used != 250 || limit != 200 {
		t.Fatalf("expected used=250 limit=200, got used=%d limit=%d", used, limit)
	}
}

func TestValidateTokenQuotaRules(t *testing.T) {
	now := time.Now().Unix()
	tests := []struct {
		name    string
		rules   []TokenQuotaRule
		wantErr bool
	}{
		{"day valid", []TokenQuotaRule{{Mode: QuotaModeDay, Limit: 100}}, false},
		{"month valid", []TokenQuotaRule{{Mode: QuotaModeMonth, Limit: 1000}}, false},
		{"year valid", []TokenQuotaRule{{Mode: QuotaModeYear, Limit: 10000}}, false},
		{"empty array", []TokenQuotaRule{}, false},
		{"unlimited limit", []TokenQuotaRule{{Mode: QuotaModeDay, Limit: -1}}, false},
		{"invalid negative limit", []TokenQuotaRule{{Mode: QuotaModeDay, Limit: -2}}, true},
		{"duplicate mode", []TokenQuotaRule{{Mode: QuotaModeDay, Limit: 100}, {Mode: QuotaModeDay, Limit: 200}}, true},
		{"invalid mode", []TokenQuotaRule{{Mode: "invalid", Limit: 100}}, true},
		{"custom missing start ok", []TokenQuotaRule{{Mode: QuotaModeCustom, Limit: 100, Refresh: QuotaRefreshDaily}}, false},
		{"custom empty refresh ok", []TokenQuotaRule{{Mode: QuotaModeCustom, Limit: 100, Start: ptrInt64(now)}}, false},
		{"custom invalid refresh", []TokenQuotaRule{{Mode: QuotaModeCustom, Limit: 100, Start: ptrInt64(now), Refresh: "bad"}}, true},
		{"custom none no end ok", []TokenQuotaRule{{Mode: QuotaModeCustom, Limit: 100, Start: ptrInt64(now), Refresh: QuotaRefreshNone}}, false},
		{"custom none with end", []TokenQuotaRule{{Mode: QuotaModeCustom, Limit: 100, Start: ptrInt64(now), End: ptrInt64(now + 3600), Refresh: QuotaRefreshNone}}, false},
		{"custom end before start", []TokenQuotaRule{{Mode: QuotaModeCustom, Limit: 100, Start: ptrInt64(now), End: ptrInt64(now - 100), Refresh: QuotaRefreshNone}}, true},
		{"custom daily valid", []TokenQuotaRule{{Mode: QuotaModeCustom, Limit: 100, Start: ptrInt64(now - 3600), Refresh: QuotaRefreshDaily}}, false},
		{"custom monthly valid", []TokenQuotaRule{{Mode: QuotaModeCustom, Limit: 100, Start: ptrInt64(now - 86400), Refresh: QuotaRefreshMonthly}}, false},
		{"custom yearly valid", []TokenQuotaRule{{Mode: QuotaModeCustom, Limit: 100, Start: ptrInt64(now - 86400), Refresh: QuotaRefreshYearly}}, false},
		{"multi mode valid", []TokenQuotaRule{{Mode: QuotaModeDay, Limit: 100}, {Mode: QuotaModeMonth, Limit: 1000}, {Mode: QuotaModeYear, Limit: 10000}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTokenQuotaRules(tt.rules)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateTokenQuotaRules() err=%v, wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestTokenQuotaDayFromRules(t *testing.T) {
	// No day rule → -1
	rules := []TokenQuotaRule{{Mode: QuotaModeMonth, Limit: 1000}}
	if got := TokenQuotaDayFromRules(rules); got != -1 {
		t.Fatalf("expected -1, got %d", got)
	}
	// With day rule
	rules = []TokenQuotaRule{{Mode: QuotaModeDay, Limit: 500}}
	if got := TokenQuotaDayFromRules(rules); got != 500 {
		t.Fatalf("expected 500, got %d", got)
	}
}

func TestEffectiveTokenQuotaDay(t *testing.T) {
	// day != -1 → 直接返回 day，不看 rules
	if got := EffectiveTokenQuotaDay(500, `[{"mode":"day","limit":999}]`); got != 500 {
		t.Fatalf("expected 500, got %d", got)
	}
	// day == -1，rules 有 day 规则 → 从 rules 反推
	if got := EffectiveTokenQuotaDay(-1, `[{"mode":"day","limit":200}]`); got != 200 {
		t.Fatalf("expected 200, got %d", got)
	}
	// day == -1，rules 无 day 规则 → 返回 -1
	if got := EffectiveTokenQuotaDay(-1, `[{"mode":"month","limit":10000}]`); got != -1 {
		t.Fatalf("expected -1, got %d", got)
	}
	// day == -1，rules 为空 → 返回 -1
	if got := EffectiveTokenQuotaDay(-1, ""); got != -1 {
		t.Fatalf("expected -1, got %d", got)
	}
	// day == 0（无限额）→ 直接返回 0
	if got := EffectiveTokenQuotaDay(0, `[{"mode":"day","limit":100}]`); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}

func TestEffectiveGlobalTokenQuotaLegacyFields(t *testing.T) {
	cases := []struct {
		name       string
		day        int
		period     string
		rulesJSON  string
		wantDay    int
		wantPeriod string
	}{
		{name: "rules day", day: -1, period: GlobalTokenQuotaPeriodMonth, rulesJSON: `[{"mode":"day","limit":200}]`, wantDay: 200, wantPeriod: GlobalTokenQuotaPeriodDay},
		{name: "rules month", day: -1, period: GlobalTokenQuotaPeriodDay, rulesJSON: `[{"mode":"month","limit":5000}]`, wantDay: 5000, wantPeriod: GlobalTokenQuotaPeriodMonth},
		{name: "first compatible rule", day: 300, period: GlobalTokenQuotaPeriodMonth, rulesJSON: `[{"mode":"day","limit":200},{"mode":"month","limit":5000}]`, wantDay: 200, wantPeriod: GlobalTokenQuotaPeriodDay},
		{name: "explicit unlimited", day: 300, period: GlobalTokenQuotaPeriodMonth, rulesJSON: `[{"mode":"month","limit":-1}]`, wantDay: -1, wantPeriod: GlobalTokenQuotaPeriodMonth},
		{name: "rules not representable", day: 300, period: GlobalTokenQuotaPeriodMonth, rulesJSON: `[{"mode":"custom","limit":100,"start":1747562460}]`, wantDay: -1, wantPeriod: GlobalTokenQuotaPeriodDay},
		{name: "empty rules", day: 300, period: GlobalTokenQuotaPeriodMonth, rulesJSON: `[]`, wantDay: -1, wantPeriod: GlobalTokenQuotaPeriodDay},
		{name: "legacy fallback", day: 300, period: GlobalTokenQuotaPeriodMonth, rulesJSON: "", wantDay: 300, wantPeriod: GlobalTokenQuotaPeriodMonth},
	}
	for _, tc := range cases {
		gotDay, gotPeriod := EffectiveGlobalTokenQuotaLegacyFields(tc.day, tc.period, tc.rulesJSON)
		if gotDay != tc.wantDay || gotPeriod != tc.wantPeriod {
			t.Fatalf("%s: expected (%d, %s), got (%d, %s)", tc.name, tc.wantDay, tc.wantPeriod, gotDay, gotPeriod)
		}
	}
}

func TestUpsertGlobalPeriodRule(t *testing.T) {
	result := UpsertGlobalPeriodRule(`[{"mode":"custom","limit":100,"start":1747562460}]`, GlobalTokenQuotaPeriodMonth, 500)
	rules, _ := ParseTokenQuotaRules(result)
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d: %v", len(rules), rules)
	}
	if got := TokenQuotaLimitFromRules(rules, QuotaModeMonth); got != 500 {
		t.Fatalf("expected month limit 500, got %d", got)
	}

	result = UpsertGlobalPeriodRule(result, GlobalTokenQuotaPeriodMonth, -1)
	rules, _ = ParseTokenQuotaRules(result)
	if got := TokenQuotaLimitFromRules(rules, QuotaModeMonth); got != -1 {
		t.Fatalf("expected month rule removed, got %d", got)
	}
	if got := TokenQuotaLimitFromRules(rules, QuotaModeCustom); got != 100 {
		t.Fatalf("expected custom rule preserved, got %d", got)
	}
}

func TestUpsertDayRule_EmptyRules(t *testing.T) {
	result := UpsertDayRule("", 300)
	rules, _ := ParseTokenQuotaRules(result)
	if len(rules) != 1 || rules[0].Mode != QuotaModeDay || rules[0].Limit != 300 {
		t.Fatalf("expected [{day,300}], got %v", rules)
	}
}

func TestUpsertDayRule_UpdateExisting(t *testing.T) {
	existing := `[{"mode":"day","limit":500}]`
	result := UpsertDayRule(existing, 300)
	rules, _ := ParseTokenQuotaRules(result)
	if len(rules) != 1 || rules[0].Limit != 300 {
		t.Fatalf("expected [{day,300}], got %v", rules)
	}
}

func TestUpsertDayRule_PreservesOtherRules(t *testing.T) {
	existing := `[{"mode":"custom","limit":1000000,"start":1747562460,"refresh":"monthly"}]`
	result := UpsertDayRule(existing, 200)
	rules, _ := ParseTokenQuotaRules(result)
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d: %v", len(rules), rules)
	}
	// custom should be preserved
	hasCustom := false
	hasDay := false
	for _, r := range rules {
		if r.Mode == QuotaModeCustom && r.Limit == 1000000 {
			hasCustom = true
		}
		if r.Mode == QuotaModeDay && r.Limit == 200 {
			hasDay = true
		}
	}
	if !hasCustom {
		t.Fatal("custom rule should be preserved")
	}
	if !hasDay {
		t.Fatal("day rule should be upserted")
	}
}

func TestUpsertDayRule_UpdateDayPreservesCustom(t *testing.T) {
	existing := `[{"mode":"day","limit":100},{"mode":"custom","limit":500000,"start":1747562460,"refresh":"monthly"}]`
	result := UpsertDayRule(existing, 999)
	rules, _ := ParseTokenQuotaRules(result)
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}
	if TokenQuotaDayFromRules(rules) != 999 {
		t.Fatalf("day should be updated to 999")
	}
	// custom still there
	found := false
	for _, r := range rules {
		if r.Mode == QuotaModeCustom {
			found = true
		}
	}
	if !found {
		t.Fatal("custom rule should be preserved after day upsert")
	}
}

func TestUpsertDayRule_NegativeOneRemovesDayRule(t *testing.T) {
	// 只有 day 规则，-1 后变成 "[]"
	existing := `[{"mode":"day","limit":500}]`
	result := UpsertDayRule(existing, -1)
	if result != "[]" {
		t.Fatalf("expected \"[]\", got %q", result)
	}

	// 有 day + custom，-1 后只保留 custom
	existing = `[{"mode":"day","limit":100},{"mode":"custom","limit":500000,"start":1747562460,"refresh":"monthly"}]`
	result = UpsertDayRule(existing, -1)
	rules, _ := ParseTokenQuotaRules(result)
	if len(rules) != 1 || rules[0].Mode != QuotaModeCustom {
		t.Fatalf("expected only custom rule, got %v", rules)
	}

	// 空 rules，-1 返回 "[]"
	result = UpsertDayRule("", -1)
	if result != "[]" {
		t.Fatalf("expected \"[]\", got %q", result)
	}
}

func TestCheckUserTokenQuota_InactiveRuleBlocks(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	db.AutoMigrate(&LLMUsageLog{}, &User{})
	cleanup := UseDBForTest(db)
	defer cleanup()

	db.Create(&User{Username: "test", Password: "x", Role: "user"})

	// 一个已过期的 custom 规则（end 已过去），应阻止访问
	pastStart := time.Now().Add(-48 * time.Hour).Unix()
	pastEnd := time.Now().Add(-24 * time.Hour).Unix()
	rules := []TokenQuotaRule{{
		Mode:    QuotaModeCustom,
		Limit:   100,
		Start:   &pastStart,
		End:     &pastEnd,
		Refresh: QuotaRefreshNone,
	}}

	exceeded, _, _ := CheckUserTokenQuota(context.Background(), 1, 0, rules)
	if !exceeded {
		t.Fatal("expired rule should block access")
	}
}

func ptrInt64(v int64) *int64 { return &v }

func TestQuotaWindow_Custom_RefreshMonthly_Fixed31Days(t *testing.T) {
	loc := hcommon.BusinessLocation()
	// Start on Jan 31 10:00
	start := time.Date(2026, 1, 31, 10, 0, 0, 0, loc).Unix()
	rule := TokenQuotaRule{Mode: QuotaModeCustom, Limit: 100000, Start: &start, Refresh: QuotaRefreshMonthly}

	// Check from Feb 15 — elapsed = 15 days < 31 days, so still in first window
	now := time.Date(2026, 2, 15, 12, 0, 0, 0, loc)
	from, to, active := rule.QuotaWindow(now)
	if !active {
		t.Fatal("should be active")
	}
	// First window: Jan 31 10:00 → Jan 31 10:00 + 31 days = Mar 3 10:00
	if from.Month() != 1 || from.Day() != 31 {
		t.Fatalf("from should be Jan 31, got %v", from)
	}
	expectedTo := time.Date(2026, 1, 31, 10, 0, 0, 0, loc).Add(31 * 24 * time.Hour)
	if to == nil || !to.Equal(expectedTo.UTC()) {
		t.Fatalf("to should be %v, got %v", expectedTo.UTC(), to)
	}
}

func TestQuotaWindow_Custom_RefreshMonthly_SecondPeriod(t *testing.T) {
	loc := hcommon.BusinessLocation()
	// Start on Jan 15 16:00
	start := time.Date(2026, 1, 15, 16, 0, 0, 0, loc).Unix()
	rule := TokenQuotaRule{Mode: QuotaModeCustom, Limit: 100000, Start: &start, Refresh: QuotaRefreshMonthly}

	// Check from Mar 10 — elapsed = 54 days, 54/31 = 1 full period
	// Window #2: start + 31d = Feb 15 16:00 → start + 62d = Mar 18 16:00
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, loc)
	from, to, active := rule.QuotaWindow(now)
	if !active {
		t.Fatal("should be active")
	}
	anchor := time.Date(2026, 1, 15, 16, 0, 0, 0, loc)
	expectedFrom := anchor.Add(31 * 24 * time.Hour) // Feb 15 16:00
	expectedTo := anchor.Add(62 * 24 * time.Hour)   // Mar 18 16:00
	if !from.Equal(expectedFrom.UTC()) {
		t.Fatalf("from should be %v, got %v", expectedFrom.UTC(), from)
	}
	if to == nil || !to.Equal(expectedTo.UTC()) {
		t.Fatalf("to should be %v, got %v", expectedTo.UTC(), to)
	}
}

func TestQuotaWindow_Custom_RefreshYearly_Fixed365Days(t *testing.T) {
	loc := hcommon.BusinessLocation()
	// Start on leap day. yearly refresh is fixed 365 days, not natural year.
	start := time.Date(2024, 2, 29, 10, 0, 0, 0, loc).Unix()
	rule := TokenQuotaRule{Mode: QuotaModeCustom, Limit: 100000, Start: &start, Refresh: QuotaRefreshYearly}

	now := time.Date(2025, 2, 28, 12, 0, 0, 0, loc)
	from, to, active := rule.QuotaWindow(now)
	if !active {
		t.Fatal("should be active")
	}
	anchor := time.Date(2024, 2, 29, 10, 0, 0, 0, loc)
	expectedFrom := anchor.Add(365 * 24 * time.Hour)
	expectedTo := anchor.Add(730 * 24 * time.Hour)
	if !from.Equal(expectedFrom.UTC()) {
		t.Fatalf("from should be %v, got %v", expectedFrom.UTC(), from)
	}
	if to == nil || !to.Equal(expectedTo.UTC()) {
		t.Fatalf("to should be %v, got %v", expectedTo.UTC(), to)
	}
}

func TestUserResolvedTokenQuotaRules_FromRules(t *testing.T) {
	u := &User{
		TokenQuotaDay:   500,
		TokenQuotaRules: `[{"mode":"month","limit":1000000}]`,
	}
	rules := u.ResolvedTokenQuotaRules()
	if len(rules) != 1 || rules[0].Mode != QuotaModeMonth {
		t.Fatalf("expected month rule, got %v", rules)
	}
}

func TestUserResolvedTokenQuotaRules_FallbackToDay(t *testing.T) {
	u := &User{TokenQuotaDay: 500, TokenQuotaRules: ""}
	rules := u.ResolvedTokenQuotaRules()
	if len(rules) != 1 || rules[0].Mode != QuotaModeDay || rules[0].Limit != 500 {
		t.Fatalf("expected day rule with limit 500, got %v", rules)
	}
}

func TestUserResolvedTokenQuotaRules_Unlimited(t *testing.T) {
	u := &User{TokenQuotaDay: -1, TokenQuotaRules: ""}
	rules := u.ResolvedTokenQuotaRules()
	if len(rules) != 0 {
		t.Fatalf("expected empty rules (unlimited), got %v", rules)
	}
}

func TestSiteConfigResolvedDefaultTokenQuotaRules(t *testing.T) {
	cfg := &SiteConfig{DefaultTokenQuotaDay: 500000, DefaultTokenQuotaRules: ""}
	rules := cfg.ResolvedDefaultTokenQuotaRules()
	if len(rules) != 1 || rules[0].Limit != 500000 {
		t.Fatalf("expected day rule with limit 500000, got %v", rules)
	}

	cfg.DefaultTokenQuotaDay = -1
	cfg.DefaultTokenQuotaRules = ""
	rules = cfg.ResolvedDefaultTokenQuotaRules()
	if len(rules) != 0 {
		t.Fatalf("expected empty rules for default_token_quota_day=-1, got %v", rules)
	}

	cfg.DefaultTokenQuotaRules = `[{"mode":"month","limit":2000000}]`
	rules = cfg.ResolvedDefaultTokenQuotaRules()
	if len(rules) != 1 || rules[0].Mode != QuotaModeMonth {
		t.Fatalf("expected month rule, got %v", rules)
	}
}

func TestGlobalRulesFromLegacyQuota_Unlimited(t *testing.T) {
	rules := GlobalRulesFromLegacyQuota(-1, GlobalTokenQuotaPeriodDay)
	if len(rules) != 0 {
		t.Fatalf("expected empty rules for global_token_quota_day=-1, got %v", rules)
	}
}

func TestFillCustomStartIfEmpty(t *testing.T) {
	rules := []TokenQuotaRule{
		{Mode: QuotaModeDay, Limit: 100},
		{Mode: QuotaModeCustom, Limit: 500, Refresh: QuotaRefreshMonthly},
	}
	modified := FillCustomStartIfEmpty(rules)
	if !modified {
		t.Fatal("should modify custom rule with nil start")
	}
	if rules[1].Start == nil {
		t.Fatal("start should be filled")
	}
	if *rules[1].Start <= 0 {
		t.Fatalf("start should be a positive unix timestamp, got %d", *rules[1].Start)
	}
	// day rule should be unchanged
	if rules[0].Start != nil {
		t.Fatal("day rule should not have start set")
	}
}

func TestFillCustomStartIfEmpty_AlreadySet(t *testing.T) {
	start := int64(1000000)
	rules := []TokenQuotaRule{
		{Mode: QuotaModeCustom, Limit: 500, Start: &start, Refresh: QuotaRefreshDaily},
	}
	modified := FillCustomStartIfEmpty(rules)
	if modified {
		t.Fatal("should not modify when start is already set")
	}
	if *rules[0].Start != 1000000 {
		t.Fatal("start should remain unchanged")
	}
}

func TestProxyQuotaCheck_OldUserFallback(t *testing.T) {
	// Simulate an unmigrated user: day=500, rules=""
	// Proxy should use fallback and enforce the 500 token limit
	gdb := setupLLMTestDB(t)
	gdb.AutoMigrate(&LLMUsageLog{})

	u := &User{TokenQuotaDay: 500, TokenQuotaRules: ""}
	rules := u.ResolvedTokenQuotaRules()
	if len(rules) != 1 || rules[0].Mode != QuotaModeDay || rules[0].Limit != 500 {
		t.Fatalf("fallback should produce day rule, got %v", rules)
	}

	// Add usage exceeding the limit
	now := time.Now().UTC()
	gdb.Create(&LLMUsageLog{UserID: 1, TotalTokens: 600, CreatedAt: now})

	exceeded, used, limit := CheckUserTokenQuota(context.Background(), 1, 0, rules)
	if !exceeded {
		t.Fatalf("should be exceeded: used=%d limit=%d", used, limit)
	}
}

func TestProxyQuotaCheck_MigratedUser(t *testing.T) {
	// Migrated user: day=-1, rules set
	gdb := setupLLMTestDB(t)
	gdb.AutoMigrate(&LLMUsageLog{})

	u := &User{TokenQuotaDay: -1, TokenQuotaRules: `[{"mode":"day","limit":1000}]`}
	rules := u.ResolvedTokenQuotaRules()
	if len(rules) != 1 || rules[0].Limit != 1000 {
		t.Fatalf("should use rules directly, got %v", rules)
	}

	now := time.Now().UTC()
	gdb.Create(&LLMUsageLog{UserID: 2, TotalTokens: 500, CreatedAt: now})

	exceeded, _, _ := CheckUserTokenQuota(context.Background(), 2, 0, rules)
	if exceeded {
		t.Fatal("should not be exceeded (500 < 1000)")
	}
}

func TestProxyQuotaCheck_UnlimitedUser(t *testing.T) {
	// Unlimited: day=-1, rules=""
	setupLLMTestDB(t)

	u := &User{TokenQuotaDay: -1, TokenQuotaRules: ""}
	rules := u.ResolvedTokenQuotaRules()
	if len(rules) != 0 {
		t.Fatalf("unlimited user should have empty rules, got %v", rules)
	}

	exceeded, _, _ := CheckUserTokenQuota(context.Background(), 1, 0, rules)
	if exceeded {
		t.Fatal("empty rules should never be exceeded")
	}
}

func TestMarshalNormalizesRefresh(t *testing.T) {
	rules := []TokenQuotaRule{
		{Mode: QuotaModeCustom, Limit: 100, Refresh: ""},
	}
	marshaled := MarshalTokenQuotaRules(rules)
	parsed, _ := ParseTokenQuotaRules(marshaled)
	if len(parsed) != 1 || parsed[0].Refresh != QuotaRefreshNone {
		t.Fatalf("empty refresh should be normalized to 'none', got %q", parsed[0].Refresh)
	}
}

// TestUserTokenUsageInWindowCompat_IncludeUngrouped 验证 includeUngrouped=true
// 时把同一用户 group_id=0 的"无分组旧 agent"用量并入分组 X 的统计。
// 对应改动：model/llm.go 的 UserTokenUsageInWindowCompat 兼容分支。
func TestUserTokenUsageInWindowCompat_IncludeUngrouped(t *testing.T) {
	gdb := setupLLMTestDB(t)
	gdb.AutoMigrate(&LLMUsageLog{})

	now := time.Now().UTC()
	from := now.Add(-2 * time.Hour)
	to := now.Add(time.Minute)

	// 用户 60：分组 5 下 100 token
	gdb.Create(&LLMUsageLog{UserID: 60, GroupID: 5, TotalTokens: 100, CreatedAt: now.Add(-time.Hour)})
	// 用户 60：无分组（group_id=0）下 50 token —— 这是"老 agent"
	gdb.Create(&LLMUsageLog{UserID: 60, GroupID: 0, TotalTokens: 50, CreatedAt: now.Add(-30 * time.Minute)})
	// 用户 60：分组 9（其它分组）下 999 token，不应被计入
	gdb.Create(&LLMUsageLog{UserID: 60, GroupID: 9, TotalTokens: 999, CreatedAt: now.Add(-15 * time.Minute)})
	// 其它用户在 group_id=0 下 7 token，不应被计入（user_id 隔离）
	gdb.Create(&LLMUsageLog{UserID: 61, GroupID: 0, TotalTokens: 7, CreatedAt: now.Add(-10 * time.Minute)})

	// 默认行为：仅统计分组 5
	if got := UserTokenUsageInWindow(context.Background(), 60, 5, from, &to); got != 100 {
		t.Fatalf("default should count only group 5, expected 100, got %d", got)
	}
	// 兼容行为：把 group_id=0 的旧用量并入
	if got := UserTokenUsageInWindowCompat(context.Background(), 60, 5, from, &to, true); got != 150 {
		t.Fatalf("compat should count group 5 + group 0, expected 150, got %d", got)
	}
	// includeUngrouped=true 但 groupID=0：走"全部用量"分支（不限分组），应得 100+50+999=1149
	if got := UserTokenUsageInWindowCompat(context.Background(), 60, 0, from, &to, true); got != 1149 {
		t.Fatalf("groupID=0 should count all groups, expected 1149, got %d", got)
	}
	// includeUngrouped=false：默认行为
	if got := UserTokenUsageInWindowCompat(context.Background(), 60, 5, from, &to, false); got != 100 {
		t.Fatalf("includeUngrouped=false should count only group 5, expected 100, got %d", got)
	}
}

// TestUserTokenUsageInWindowCompat_OpenEnded 验证 to=nil 时兼容路径仍正确工作。
func TestUserTokenUsageInWindowCompat_OpenEnded(t *testing.T) {
	gdb := setupLLMTestDB(t)
	gdb.AutoMigrate(&LLMUsageLog{})

	from := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)
	gdb.Create(&LLMUsageLog{UserID: 70, GroupID: 3, TotalTokens: 200, CreatedAt: from.Add(time.Minute)})
	gdb.Create(&LLMUsageLog{UserID: 70, GroupID: 0, TotalTokens: 33, CreatedAt: from.Add(2 * time.Minute)})
	// 窗口外不计
	gdb.Create(&LLMUsageLog{UserID: 70, GroupID: 0, TotalTokens: 999, CreatedAt: from.Add(-time.Hour)})

	if got := UserTokenUsageInWindowCompat(context.Background(), 70, 3, from, nil, true); got != 233 {
		t.Fatalf("open-ended compat expected 233, got %d", got)
	}
}
