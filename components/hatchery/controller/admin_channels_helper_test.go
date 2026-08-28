package controller

import (
	"context"
	"sort"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// initAdminChannelsHelperTestDB 为不依赖业务表的能力测试准备最小 DB（仅 CustomAgentType）。
// model.GetAllAgentTypes 会查询自定义 Agent 类型表，因此即使测试只读内置类型也需要先建表。
func initAdminChannelsHelperTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.CustomAgentType{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
}

// TestEnrichChannelsWithAgentTypes_CustomChannelAllThree 自定义 channel (Custom=true)
// 应绕过白名单，agent_types 返回三端全量。
func TestEnrichChannelsWithAgentTypes_CustomChannelAllThree(t *testing.T) {
	initAdminChannelsHelperTestDB(t)
	channels := []model.AIChannel{
		{
			ChannelID: "custom_channel_xxx",
			Name:      "MyCustom",
			Custom:    true,
		},
	}

	enriched := enrichChannelsWithAgentTypes(context.Background(), channels)
	if len(enriched) != 1 {
		t.Fatalf("期望 1 条，实际=%d", len(enriched))
	}

	ats := enriched[0].AgentTypes
	set := map[string]bool{}
	for _, code := range ats {
		set[code] = true
	}
	if !set[model.AgentTypeOpenClaw] || !set[model.AgentTypeHermes] || !set[model.AgentTypeLightclawACE] {
		t.Errorf("自定义 channel 应返回三端全量，实际=%v", ats)
	}
}

// TestEnrichChannelsWithAgentTypes_BuiltinChannelFollowsWhitelist 内置 channel 依白名单返回。
func TestEnrichChannelsWithAgentTypes_BuiltinChannelFollowsWhitelist(t *testing.T) {
	tests := []struct {
		name      string
		channelID string
		want      []string
	}{
		{"feishu_openclaw_and_ace", "feishu", []string{model.AgentTypeOpenClaw, model.AgentTypeHermes, model.AgentTypeLightclawACE}},
		{"openclaw_weixin_only_openclaw", "openclaw-weixin", []string{model.AgentTypeOpenClaw, model.AgentTypeHermes, model.AgentTypeLightclawACE}},
		{"wecom_app_only_openclaw", "wecom_app", []string{model.AgentTypeOpenClaw}},
		{"ddingtalk_only_openclaw", "ddingtalk", []string{model.AgentTypeOpenClaw, model.AgentTypeHermes}},
		{"slack_openclaw_and_hermes", "slack", []string{model.AgentTypeOpenClaw, model.AgentTypeHermes}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initAdminChannelsHelperTestDB(t)
			channels := []model.AIChannel{{ChannelID: tt.channelID, Name: tt.name}}
			enriched := enrichChannelsWithAgentTypes(context.Background(), channels)
			if len(enriched) != 1 {
				t.Fatalf("expected 1, got %d", len(enriched))
			}
			got := enriched[0].AgentTypes

			sortedGot := append([]string(nil), got...)
			sortedWant := append([]string(nil), tt.want...)
			sort.Strings(sortedGot)
			sort.Strings(sortedWant)

			if len(sortedGot) != len(sortedWant) {
				t.Fatalf("len: got=%v want=%v", got, tt.want)
			}
			for i := range sortedGot {
				if sortedGot[i] != sortedWant[i] {
					t.Errorf("[%d]: got=%q want=%q", i, sortedGot[i], sortedWant[i])
				}
			}
		})
	}
}

// TestEnrichChannelsWithAgentTypes_EmptyInput 空输入返回非 nil 的空切片（JSON 序列化为 []）。
func TestEnrichChannelsWithAgentTypes_EmptyInput(t *testing.T) {
	got := enrichChannelsWithAgentTypes(context.Background(), []model.AIChannel{})
	if got == nil {
		t.Errorf("空输入应返回非 nil 的空 slice（JSON 序列化为 []），实际为 nil")
	}
	if len(got) != 0 {
		t.Errorf("空输入长度应为 0，实际=%d", len(got))
	}
}

// TestEnrichChannelsWithAgentTypes_UnknownBuiltinChannel 未知内置 channel 返回空 agent_types。
func TestEnrichChannelsWithAgentTypes_UnknownBuiltinChannel(t *testing.T) {
	initAdminChannelsHelperTestDB(t)
	channels := []model.AIChannel{
		{ChannelID: "some_future_channel", Name: "future", Custom: false},
	}
	enriched := enrichChannelsWithAgentTypes(context.Background(), channels)
	if len(enriched) != 1 {
		t.Fatalf("expected 1, got %d", len(enriched))
	}
	if enriched[0].AgentTypes == nil {
		t.Error("agent_types 应为非 nil 空 slice，实际为 nil")
	}
	if len(enriched[0].AgentTypes) != 0 {
		t.Errorf("未知 channel 的 agent_types 应为空，实际=%v", enriched[0].AgentTypes)
	}
}

// TestQueryAllChannels_EmptyDB 空表时返回空列表。
func TestQueryAllChannels_EmptyDB(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.AIChannel{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	defer model.UseDBForTest(db)()

	channels := queryAllChannels(context.Background())
	if channels == nil {
		t.Error("queryAllChannels 应返回非 nil 空 slice")
	}
	if len(channels) != 0 {
		t.Errorf("空表应返回 0 条，实际=%d", len(channels))
	}
}

// TestQueryAllChannels_WithData 插入数据后能查到并按预定义顺序排序。
func TestQueryAllChannels_WithData(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.AIChannel{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	defer model.UseDBForTest(db)()

	// 插入两条
	db.Create(&model.AIChannel{ChannelID: "feishu", Name: "Feishu"})
	db.Create(&model.AIChannel{ChannelID: "wecom", Name: "WeCom"})

	channels := queryAllChannels(context.Background())
	if len(channels) != 2 {
		t.Errorf("期望 2 条，实际=%d", len(channels))
	}
}
