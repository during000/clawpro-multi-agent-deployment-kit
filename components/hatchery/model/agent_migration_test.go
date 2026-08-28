package model

import (
	"encoding/json"
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func initAgentMigrationTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&AgentMigration{}, &CustomAgentType{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	gdb = db
}

func createTestMigration(t *testing.T, agentType string) *AgentMigration {
	t.Helper()
	m := &AgentMigration{
		InstanceID:    1,
		CVMInstanceID: "ins-test",
		Status:        MigrationStatusImporting,
	}
	if err := gdb.Create(m).Error; err != nil {
		t.Fatalf("create migration: %v", err)
	}
	InitMigrationSteps(context.Background(), gdb, m, agentType)
	return m
}

// ---------- MigrationStepsForAgentType ----------

func TestMigrationStepsForAgentType_OpenClaw(t *testing.T) {
	initAgentMigrationTestDB(t)
	steps := MigrationStepsForAgentType(context.Background(), AgentTypeOpenClaw)
	if len(steps) == 0 {
		t.Fatal("steps should not be empty")
	}
	// openclaw 支持 SMH，syncing_smh 应在列表中
	found := false
	for _, s := range steps {
		if s == MigrationStepSyncingSMH {
			found = true
		}
	}
	if !found {
		t.Errorf("openclaw should include syncing_smh, got %v", steps)
	}
}

func TestMigrationStepsForAgentType_NoSMH(t *testing.T) {
	initAgentMigrationTestDB(t)
	// 未知 agent type → AgentTypeSupportsSMH 返回 false，不含 syncing_smh
	steps := MigrationStepsForAgentType(context.Background(), "unknown_type")
	for _, s := range steps {
		if s == MigrationStepSyncingSMH {
			t.Errorf("unknown_type should not include syncing_smh")
		}
	}
	// 必须包含固定的5个阶段
	required := []string{
		MigrationStepDownloading,
		MigrationStepExtracting,
		MigrationStepRestarting,
		// waiting_ready step removed
		MigrationStepSyncingModels,
	}
	for _, r := range required {
		found := false
		for _, s := range steps {
			if s == r {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("step %q should always be present, got %v", r, steps)
		}
	}
}

func TestMigrationStepsForAgentType_EmptyType(t *testing.T) {
	initAgentMigrationTestDB(t)
	// 空 agentType 走 openclaw 默认路径
	steps := MigrationStepsForAgentType(context.Background(), "")
	if len(steps) == 0 {
		t.Fatal("steps should not be empty for empty agentType")
	}
}

// ---------- InitMigrationSteps ----------

func TestInitMigrationSteps_AllPending(t *testing.T) {
	initAgentMigrationTestDB(t)
	m := createTestMigration(t, AgentTypeOpenClaw)

	steps := ParseMigrationSteps(m)
	if len(steps) == 0 {
		t.Fatal("steps should not be empty after init")
	}
	for _, s := range steps {
		if s.Status != MigrationStepStatusPending {
			t.Errorf("step %q should be pending, got %q", s.Step, s.Status)
		}
		if s.Ts != nil {
			t.Errorf("step %q ts should be nil, got %v", s.Step, s.Ts)
		}
	}
}

func TestInitMigrationSteps_WritesToDB(t *testing.T) {
	initAgentMigrationTestDB(t)
	m := createTestMigration(t, AgentTypeOpenClaw)

	var fromDB AgentMigration
	gdb.First(&fromDB, m.ID)
	if fromDB.StepsJSON == "" {
		t.Error("steps_json should be persisted in DB")
	}
}

// ---------- UpdateMigrationStep ----------

func TestUpdateMigrationStep_SetsStatusAndTs(t *testing.T) {
	initAgentMigrationTestDB(t)
	m := createTestMigration(t, AgentTypeOpenClaw)

	UpdateMigrationStep(gdb, m, MigrationStepDownloading, MigrationStepStatusRunning, nil)

	steps := ParseMigrationSteps(m)
	for _, s := range steps {
		if s.Step == MigrationStepDownloading {
			if s.Status != MigrationStepStatusRunning {
				t.Errorf("expected running, got %q", s.Status)
			}
			if s.Ts == nil {
				t.Error("ts should be set after update")
			}
			return
		}
	}
	t.Error("downloading step not found")
}

func TestUpdateMigrationStep_WithDetail(t *testing.T) {
	initAgentMigrationTestDB(t)
	m := createTestMigration(t, AgentTypeOpenClaw)

	detail := map[string]interface{}{"is_primary_model_valid": true}
	UpdateMigrationStep(gdb, m, MigrationStepSyncingModels, MigrationStepStatusSuccess, detail)

	steps := ParseMigrationSteps(m)
	for _, s := range steps {
		if s.Step == MigrationStepSyncingModels {
			if s.Detail == nil {
				t.Fatal("detail should be set")
			}
			v, ok := s.Detail["is_primary_model_valid"]
			if !ok {
				t.Fatal("is_primary_model_valid key missing")
			}
			// JSON round-trip: bool becomes bool
			if v != true {
				t.Errorf("expected true, got %v", v)
			}
			return
		}
	}
	t.Error("syncing_models step not found")
}

func TestUpdateMigrationStep_NilDetailPreservesExisting(t *testing.T) {
	initAgentMigrationTestDB(t)
	m := createTestMigration(t, AgentTypeOpenClaw)

	// 先写入 detail
	detail := map[string]interface{}{"foo": "bar"}
	UpdateMigrationStep(gdb, m, MigrationStepDownloading, MigrationStepStatusRunning, detail)

	// 再以 nil detail 更新
	UpdateMigrationStep(gdb, m, MigrationStepDownloading, MigrationStepStatusSuccess, nil)

	steps := ParseMigrationSteps(m)
	for _, s := range steps {
		if s.Step == MigrationStepDownloading {
			if s.Status != MigrationStepStatusSuccess {
				t.Errorf("status should be success, got %q", s.Status)
			}
			// detail 应保留原有值
			if s.Detail == nil {
				t.Error("detail should be preserved when nil is passed")
			}
			return
		}
	}
	t.Error("downloading step not found")
}

func TestUpdateMigrationStep_UnknownStepNoOp(t *testing.T) {
	initAgentMigrationTestDB(t)
	m := createTestMigration(t, AgentTypeOpenClaw)
	orig := m.StepsJSON

	UpdateMigrationStep(gdb, m, "nonexistent_step", MigrationStepStatusSuccess, nil)

	// StepsJSON 内容应不变（步骤未找到，循环不触发修改）
	if m.StepsJSON != orig {
		t.Error("unknown step should not modify steps")
	}
}

func TestUpdateMigrationStep_PersistsToDb(t *testing.T) {
	initAgentMigrationTestDB(t)
	m := createTestMigration(t, AgentTypeOpenClaw)

	UpdateMigrationStep(gdb, m, MigrationStepExtracting, MigrationStepStatusSuccess, nil)

	var fromDB AgentMigration
	gdb.First(&fromDB, m.ID)
	steps := ParseMigrationSteps(&fromDB)
	for _, s := range steps {
		if s.Step == MigrationStepExtracting {
			if s.Status != MigrationStepStatusSuccess {
				t.Errorf("gdb should have success, got %q", s.Status)
			}
			return
		}
	}
	t.Error("extracting step not found in DB")
}

// ---------- ParseMigrationSteps ----------

func TestParseMigrationSteps_Empty(t *testing.T) {
	m := &AgentMigration{}
	steps := ParseMigrationSteps(m)
	if steps != nil {
		t.Error("empty StepsJSON should return nil")
	}
}

func TestParseMigrationSteps_InvalidJSON(t *testing.T) {
	m := &AgentMigration{StepsJSON: "not-json"}
	steps := ParseMigrationSteps(m)
	if steps != nil {
		t.Error("invalid JSON should return nil")
	}
}

func TestParseMigrationSteps_ValidJSON(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	s := []MigrationStep{
		{Step: "downloading", Status: "running", Ts: &now},
		{Step: "extracting", Status: "pending", Ts: nil},
	}
	b, _ := json.Marshal(s)
	m := &AgentMigration{StepsJSON: string(b)}

	steps := ParseMigrationSteps(m)
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps))
	}
	if steps[0].Step != "downloading" || steps[0].Status != "running" {
		t.Errorf("unexpected step[0]: %+v", steps[0])
	}
	if steps[1].Ts != nil {
		t.Errorf("step[1] ts should be nil")
	}
}

// ---------- MigrationStep constants ----------

func TestMigrationStatusConstants(t *testing.T) {
	for _, s := range []string{
		MigrationStatusPendingUpload,
		MigrationStatusImporting,
		MigrationStatusDone,
		MigrationStatusFailed,
	} {
		if s == "" {
			t.Errorf("status constant should not be empty")
		}
	}
}

func TestUpdateMigrationStep_InvalidJSON(t *testing.T) {
	// StepsJSON 不合法时，UpdateMigrationStep 应直接 return 不 panic
	m := &AgentMigration{StepsJSON: "invalid-json"}
	// 不应 panic
	UpdateMigrationStep(nil, m, MigrationStepDownloading, MigrationStepStatusRunning, nil)
	// StepsJSON 不变
	if m.StepsJSON != "invalid-json" {
		t.Error("invalid JSON should leave StepsJSON unchanged")
	}
}

func TestMigrationStepConstants(t *testing.T) {
	for _, s := range []string{
		MigrationStepDownloading,
		MigrationStepExtracting,
		MigrationStepRestarting,
		// waiting_ready step removed
		MigrationStepSyncingModels,
		MigrationStepSyncingSMH,
	} {
		if s == "" {
			t.Errorf("step constant should not be empty")
		}
	}
}
