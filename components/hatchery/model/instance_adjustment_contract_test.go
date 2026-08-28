package model

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestAI2InstanceTypeRank_FourTiers(t *testing.T) {
	tiers := []string{"Ai2.MEDIUM2", "Ai2.MEDIUM4", "Ai2.LARGE8", "Ai2.2XLARGE16"}
	for currentRank, current := range tiers {
		rank, ok := AI2InstanceTypeRank(current)
		if !ok || rank != currentRank {
			t.Fatalf("AI2InstanceTypeRank(%q) = (%d, %v), want (%d, true)", current, rank, ok, currentRank)
		}
	}

	if _, ok := AI2InstanceTypeRank("Ai2.4XLARGE32"); ok {
		t.Fatal("unknown AI2 tier unexpectedly has a rank")
	}
}

func TestResourceAdjustmentOperations_ClassificationAndTimeout(t *testing.T) {
	for _, operation := range []string{OpAdjustInstanceType, OpAdjustSystemDisk} {
		if !IsResourceAdjustmentOperation(operation) {
			t.Errorf("%q is not classified as a resource adjustment", operation)
		}
		if got := OperationTimeouts[operation]; got != 900 {
			t.Errorf("OperationTimeouts[%q] = %d, want 900", operation, got)
		}
		if _, ok := OperationTransitStatus[operation]; ok {
			t.Errorf("resource adjustment %q must not use generic lifecycle transit state", operation)
		}
	}
	for _, operation := range []string{OpNone, OpCreate, OpReboot, OpDelete} {
		if IsResourceAdjustmentOperation(operation) {
			t.Errorf("ordinary operation %q classified as resource adjustment", operation)
		}
	}
}

func TestInstanceAdjustmentSchema_AutoMigrateColumnsDefaultsAndIndexes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Instance{}, &InstanceAdjustment{}); err != nil {
		t.Fatalf("AutoMigrate adjustment schema: %v", err)
	}

	for _, column := range []string{
		"cvm_instance_type", "cvm_cpu", "cvm_memory_gb", "system_disk_type", "system_disk_size",
		"cvm_public_ip", "cvm_internet_charge_type", "cvm_internet_max_bandwidth_out",
	} {
		if !db.Migrator().HasColumn(&Instance{}, column) {
			t.Errorf("AutoMigrate missing instances.%s", column)
		}
	}
	for _, column := range []string{
		"resource_synced_at", "adjustment_status", "adjustment_type", "adjustment_phase",
		"adjustment_target_instance_type", "adjustment_target_disk_size", "adjustment_resize_mode",
		"adjustment_original_cvm_state", "adjustment_original_stop_charging_mode", "adjustment_request_id",
		"adjustment_error_code", "adjustment_error_message", "adjustment_started_at",
		"adjustment_updated_at", "adjustment_next_poll_at", "adjustment_reconcile_count",
	} {
		if db.Migrator().HasColumn(&Instance{}, column) {
			t.Errorf("obsolete instances.%s still exists", column)
		}
	}
	for _, field := range []string{"CVMInstanceType", "SystemDiskSize"} {
		if !db.Migrator().HasIndex(&Instance{}, field) {
			t.Errorf("AutoMigrate missing instances index for %s", field)
		}
	}

	for _, column := range []string{
		"id", "created_at", "updated_at", "finished_at", "execution_started_at", "identifier", "instance_id",
		"status", "adjustment_type", "phase", "payload_json", "request_id", "run_at", "attempt", "error_code",
	} {
		if !db.Migrator().HasColumn(&InstanceAdjustment{}, column) {
			t.Errorf("AutoMigrate missing instance_adjustments.%s", column)
		}
	}
	for _, index := range []string{"uk_instance_adjustment_instance", "idx_instance_adjustment_due"} {
		if !db.Migrator().HasIndex(&InstanceAdjustment{}, index) {
			t.Errorf("AutoMigrate missing instance_adjustments index %s", index)
		}
	}

	instance := Instance{Name: "schema-defaults", InstanceId: "ins-schema-defaults"}
	if err := db.Create(&instance).Error; err != nil {
		t.Fatalf("insert instance defaults row: %v", err)
	}
	var stored Instance
	if err := db.First(&stored, instance.ID).Error; err != nil {
		t.Fatalf("reload instance defaults row: %v", err)
	}
	if stored.CVMInstanceType != "" || stored.CVMCPU != 0 || stored.CVMMemoryGB != 0 ||
		stored.SystemDiskType != "" || stored.SystemDiskSize != 0 || stored.CVMPublicIP != "" ||
		stored.CVMInternetChargeType != "" || stored.CVMInternetMaxBandwidthOut != 0 {
		t.Fatalf("unexpected resource defaults: %+v", stored)
	}

	task := InstanceAdjustment{
		InstanceID: instance.ID,
		Type:       "instance_type",
		RunAt:      time.Now(),
	}
	if err := task.SetPayload(InstanceAdjustmentPayload{TargetInstanceType: "Ai2.LARGE8"}); err != nil {
		t.Fatalf("encode adjustment payload: %v", err)
	}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("insert adjustment defaults row: %v", err)
	}
	var storedTask InstanceAdjustment
	if err := db.First(&storedTask, task.ID).Error; err != nil {
		t.Fatalf("reload adjustment defaults row: %v", err)
	}
	if storedTask.Status != "processing" || storedTask.Phase != "queued" ||
		storedTask.RequestID != "" || storedTask.Attempt != 0 ||
		storedTask.ErrorCode != "" || storedTask.FinishedAt != nil ||
		storedTask.ExecutionStartedAt != nil {
		t.Fatalf("unexpected adjustment defaults: %+v", storedTask)
	}
	payload, err := storedTask.Payload()
	if err != nil || payload.TargetInstanceType != "Ai2.LARGE8" {
		t.Fatalf("stored payload=%+v err=%v", payload, err)
	}
}
