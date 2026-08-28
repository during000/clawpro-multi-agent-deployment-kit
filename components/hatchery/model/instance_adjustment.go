package model

import (
	"encoding/json"
	"time"
)

// InstanceAdjustmentPayload stores operation-specific state that is not queried
// by the scheduler. Keeping it as one payload avoids expanding the instances
// table or the task table for every new adjustment type.
type InstanceAdjustmentPayload struct {
	TargetInstanceType       string `json:"target_instance_type,omitempty"`
	TargetDiskSize           int64  `json:"target_disk_size,omitempty"`
	ResizeMode               string `json:"resize_mode,omitempty"`
	OriginalCVMState         string `json:"original_cvm_state"`
	OriginalStopChargingMode string `json:"original_stop_charging_mode,omitempty"`
}

// InstanceAdjustment is the single current/latest resource-adjustment task for
// one instance. Successful tasks are deleted; failed tasks remain until the next
// accepted mutation so list/status can expose the latest failure.
type InstanceAdjustment struct {
	ID uint `gorm:"column:id;primaryKey" json:"id"`

	CreatedAt          time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	FinishedAt         *time.Time `gorm:"column:finished_at;default:null" json:"finished_at,omitempty"`
	ExecutionStartedAt *time.Time `gorm:"column:execution_started_at;default:null" json:"-"`

	Identifier string `gorm:"column:identifier;type:varchar(191);not null;default:'';uniqueIndex:uk_instance_adjustment_instance,priority:1" json:"-"`
	InstanceID uint   `gorm:"column:instance_id;not null;uniqueIndex:uk_instance_adjustment_instance,priority:2" json:"instance_id"`

	Status      string    `gorm:"column:status;type:varchar(16);not null;default:'processing';index:idx_instance_adjustment_due,priority:1" json:"status"`
	Type        string    `gorm:"column:adjustment_type;type:varchar(32);not null" json:"adjustment_type"`
	Phase       string    `gorm:"column:phase;type:varchar(32);not null;default:'queued'" json:"phase"`
	PayloadJSON string    `gorm:"column:payload_json;type:text;not null" json:"-"`
	RequestID   string    `gorm:"column:request_id;type:varchar(64);not null;default:''" json:"-"`
	RunAt       time.Time `gorm:"column:run_at;not null;index:idx_instance_adjustment_due,priority:2" json:"-"`
	Attempt     int       `gorm:"column:attempt;not null;default:0" json:"-"`
	ErrorCode   string    `gorm:"column:error_code;type:varchar(128);not null;default:''" json:"error_code,omitempty"`
}

func (InstanceAdjustment) TableName() string { return "instance_adjustments" }

func (a *InstanceAdjustment) Payload() (InstanceAdjustmentPayload, error) {
	var payload InstanceAdjustmentPayload
	err := json.Unmarshal([]byte(a.PayloadJSON), &payload)
	return payload, err
}

func (a *InstanceAdjustment) SetPayload(payload InstanceAdjustmentPayload) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	a.PayloadJSON = string(encoded)
	return nil
}
