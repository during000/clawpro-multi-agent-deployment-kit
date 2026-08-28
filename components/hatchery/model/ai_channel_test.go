package model

import (
	"context"
	"fmt"
	hcommon "hatchery/common"
	"sync/atomic"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var aiChannelTestDBCounter int64

// setupAIChannelTestDB creates an isolated SQLite memory database for testing.
func setupAIChannelTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	id := atomic.AddInt64(&aiChannelTestDBCounter, 1)
	dsn := fmt.Sprintf("file:aiChannelTest%d?mode=memory&cache=shared", id)
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite mem: %v", err)
	}
	if err := gdb.AutoMigrate(&AIChannel{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	return gdb
}

// TestSeedChannels_CreatesPredefinedChannels verifies that SeedChannels creates predefined channels.
func TestSeedChannels_CreatesPredefinedChannels(t *testing.T) {
	gdb := setupAIChannelTestDB(t)

	ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{
		Identifier:  "",
		DefaultLang: "zh",
	})

	err := gdb.Transaction(func(tx *gorm.DB) error {
		return SeedChannels(ctx, tx)
	})
	if err != nil {
		t.Fatalf("SeedChannels: %v", err)
	}

	var count int64
	if err := gdb.Model(&AIChannel{}).Count(&count).Error; err != nil {
		t.Fatalf("count channels: %v", err)
	}
	if count == 0 {
		t.Fatalf("expected channels to be created, got count %d", count)
	}

	var slack AIChannel
	if err := gdb.Where("channel_id = ?", "slack").First(&slack).Error; err != nil {
		t.Fatalf("expected slack predefined channel: %v", err)
	}
	if slack.Name != "Slack" {
		t.Fatalf("slack channel name = %q, want Slack", slack.Name)
	}
}

// TestSeedChannels_IdempotentCall verifies that calling SeedChannels twice doesn't create duplicates.
func TestSeedChannels_IdempotentCall(t *testing.T) {
	gdb := setupAIChannelTestDB(t)

	ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{
		Identifier:  "",
		DefaultLang: "zh",
	})

	// First call
	err := gdb.Transaction(func(tx *gorm.DB) error {
		return SeedChannels(ctx, tx)
	})
	if err != nil {
		t.Fatalf("SeedChannels first call: %v", err)
	}

	var count1 int64
	if err := gdb.Model(&AIChannel{}).Count(&count1).Error; err != nil {
		t.Fatalf("count first call: %v", err)
	}

	// Second call should not create duplicates
	err = gdb.Transaction(func(tx *gorm.DB) error {
		return SeedChannels(ctx, tx)
	})
	if err != nil {
		t.Fatalf("SeedChannels second call: %v", err)
	}

	var count2 int64
	if err := gdb.Model(&AIChannel{}).Count(&count2).Error; err != nil {
		t.Fatalf("count second call: %v", err)
	}
	if count1 != count2 {
		t.Fatalf("expected same count after second call, first=%d second=%d", count1, count2)
	}
}

func TestChannelParams_Slack(t *testing.T) {
	params := ChannelParams["slack"]
	if len(params) != 2 {
		t.Fatalf("slack params len=%d, want 2", len(params))
	}
	if params[0].Key != "app_token" || params[0].Label != "App-Level Token" {
		t.Fatalf("slack app token param=%+v", params[0])
	}
	if params[1].Key != "bot_token" || params[1].Label != "Bot User OAuth Token" {
		t.Fatalf("slack bot token param=%+v", params[1])
	}
}

func TestPredefinedChannels_HaveSiteScope(t *testing.T) {
	for _, ch := range predefinedChannels {
		scope, ok := ChannelSiteScopeFor(ch.ChannelID)
		if !ok {
			t.Fatalf("predefined channel %s missing site scope", ch.ChannelID)
		}
		if scope == 0 {
			t.Fatalf("predefined channel %s has empty site scope", ch.ChannelID)
		}
	}
}

func TestFilterChannelsBySiteScope(t *testing.T) {
	channels := []AIChannel{
		{ChannelID: "feishu", Name: "Feishu"},
		{ChannelID: "msteams", Name: "Microsoft Teams"},
		{ChannelID: "line", Name: "LINE"},
		{ChannelID: "slack", Name: "Slack"},
		{ChannelID: "custom_x", Name: "Custom", Custom: true},
	}

	domestic := FilterChannelsBySiteScope(channels, false)
	if got := channelIDsForTest(domestic); fmt.Sprint(got) != "[feishu msteams custom_x]" {
		t.Fatalf("domestic visible channels=%v, want [feishu msteams custom_x]", got)
	}

	overseas := FilterChannelsBySiteScope(channels, true)
	if got := channelIDsForTest(overseas); fmt.Sprint(got) != "[msteams line slack custom_x]" {
		t.Fatalf("overseas visible channels=%v, want [msteams line slack custom_x]", got)
	}
}

func TestFilterChannelIDsBySiteScope(t *testing.T) {
	ids := []string{"feishu", "msteams", "line", "slack", "qqbot", "unknown"}
	if got := FilterChannelIDsBySiteScope(ids, false); fmt.Sprint(got) != "[feishu msteams qqbot unknown]" {
		t.Fatalf("domestic visible ids=%v, want [feishu msteams qqbot unknown]", got)
	}
	if got := FilterChannelIDsBySiteScope(ids, true); fmt.Sprint(got) != "[msteams line slack qqbot unknown]" {
		t.Fatalf("overseas visible ids=%v, want [msteams line slack qqbot unknown]", got)
	}
}

func channelIDsForTest(channels []AIChannel) []string {
	out := make([]string, 0, len(channels))
	for _, ch := range channels {
		out = append(out, ch.ChannelID)
	}
	return out
}

// TestSeedChannels_ChannelsHaveSensibleDefaults verifies that channels have proper defaults.
func TestSeedChannels_ChannelsHaveSensibleDefaults(t *testing.T) {
	gdb := setupAIChannelTestDB(t)

	ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{
		Identifier:  "",
		DefaultLang: "zh",
	})

	err := gdb.Transaction(func(tx *gorm.DB) error {
		return SeedChannels(ctx, tx)
	})
	if err != nil {
		t.Fatalf("SeedChannels: %v", err)
	}

	var channels []AIChannel
	if err := gdb.Find(&channels).Error; err != nil {
		t.Fatalf("find channels: %v", err)
	}

	for _, ch := range channels {
		if ch.ChannelID == "" {
			t.Fatalf("channel has empty channel_id")
		}
		if ch.Name == "" {
			t.Fatalf("channel %s has empty name", ch.ChannelID)
		}
		if ch.Enabled == nil {
			t.Fatalf("channel %s has nil enabled flag", ch.ChannelID)
		}
		if ch.Custom {
			t.Fatalf("predefined channel %s should have custom=false", ch.ChannelID)
		}
	}
}

// TestSeedChannels_WithClosedDatabase verifies error handling when database operations fail.
func TestSeedChannels_WithClosedDatabase(t *testing.T) {
	gdb := setupAIChannelTestDB(t)

	ctx := hcommon.InjectTenant(context.Background(), hcommon.TenantSnapshot{
		Identifier:  "",
		DefaultLang: "zh",
	})

	// Get the underlying sqlDB and close it to simulate database error
	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatalf("get sqlDB: %v", err)
	}
	sqlDB.Close()

	// Now attempt to seed channels on the closed database
	// This should fail during the create operation
	err = gdb.Transaction(func(tx *gorm.DB) error {
		return SeedChannels(ctx, tx)
	})
	if err == nil {
		t.Fatalf("expected error with closed database, got nil")
	}
}
