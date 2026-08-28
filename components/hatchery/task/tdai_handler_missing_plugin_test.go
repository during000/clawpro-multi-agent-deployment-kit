package task

import (
	"context"
	"strings"
	"testing"

	"hatchery/model"
)

func TestHandleSwitchToFree_PluginMissing(t *testing.T) {
	setupMemoryProTestDB(t)

	model.DB(context.Background()).Create(&model.Instance{InstanceId: "inst-free-no-plugin", AgentType: model.AgentTypeOpenClaw})
	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToFree, "switch:inst-free-no-plugin", "inst-free-no-plugin", "{}", "u", "")

	err := handleSwitchToFree(job)
	if err == nil {
		t.Fatal("expected error when plugin row is missing")
	}
	if !strings.Contains(err.Error(), "plugin 行失败") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHandleSwitchToOff_PluginMissing(t *testing.T) {
	setupMemoryProTestDB(t)

	model.DB(context.Background()).Create(&model.Instance{InstanceId: "inst-off-no-plugin", AgentType: model.AgentTypeOpenClaw})
	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToOff, "switch:inst-off-no-plugin", "inst-off-no-plugin", "{}", "u", "")

	err := handleSwitchToOff(job)
	if err == nil {
		t.Fatal("expected error when plugin row is missing")
	}
	if !strings.Contains(err.Error(), "plugin 行失败") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHandleSwitchToPro_PluginMissing(t *testing.T) {
	setupMemoryProTestDB(t)

	model.DB(context.Background()).Create(&model.Instance{InstanceId: "inst-pro-no-plugin", AgentType: model.AgentTypeOpenClaw})
	job, _ := model.SubmitJob(context.Background(), model.TdaiJobTypeSwitchToPro, "switch:inst-pro-no-plugin", "inst-pro-no-plugin", "{}", "u", "")

	err := handleSwitchToPro(job)
	if err == nil {
		t.Fatal("expected error when plugin row is missing")
	}
	if !strings.Contains(err.Error(), "plugin 行失败") {
		t.Fatalf("unexpected error: %v", err)
	}
}
