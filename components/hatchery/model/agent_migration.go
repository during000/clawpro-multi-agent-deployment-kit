package model

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// 迁移状态常量
const (
	MigrationStatusPendingUpload = "pending_upload"
	MigrationStatusImporting     = "importing"
	MigrationStatusDone          = "done"
	MigrationStatusFailed        = "failed"
)

// 迁移阶段名称常量
const (
	MigrationStepDownloading   = "downloading"
	MigrationStepBackingUp     = "backing_up"
	MigrationStepExtracting    = "extracting"
	MigrationStepRestarting    = "restarting"
	MigrationStepSyncingModels = "syncing_models"
	MigrationStepSyncingSMH    = "syncing_smh"
)

// 迁移阶段状态常量
const (
	MigrationStepStatusPending = "pending"
	MigrationStepStatusRunning = "running"
	MigrationStepStatusSuccess = "success"
	MigrationStepStatusFailed  = "failed"
)

// MigrationStep 单个迁移阶段的状态快照。
type MigrationStep struct {
	Step   string                 `json:"step"`
	Status string                 `json:"status"`
	Ts     *string                `json:"ts"`              // RFC3339，nil 表示尚未开始
	Detail map[string]interface{} `json:"detail,omitempty"` // 阶段附加信息，按需填充
}

type AgentMigration struct {
	gorm.Model
	Identifier    string `gorm:"column:identifier;index;default:''"`
	InstanceID    uint   `gorm:"column:instance_id;not null;index"`
	CVMInstanceID string `gorm:"column:cvm_instance_id;type:varchar(64);not null;default:''"`
	FileKey       string `gorm:"column:file_key;type:varchar(255);not null;default:''"`
	Status        string `gorm:"column:status;type:varchar(32);not null;default:'pending_upload';index"`
	FailReason    string `gorm:"column:fail_reason;type:text"`
	StepsJSON     string `gorm:"column:steps_json;type:text"`
}

// MigrationStepsForAgentType 返回指定 agent type 应有的迁移阶段列表。
func MigrationStepsForAgentType(ctx context.Context, agentType string) []string {
	steps := []string{
		MigrationStepDownloading,
		MigrationStepBackingUp,
		MigrationStepExtracting,
		MigrationStepRestarting,
		MigrationStepSyncingModels,
	}
	if AgentTypeSupportsSMH(ctx, agentType) {
		steps = append(steps, MigrationStepSyncingSMH)
	}
	return steps
}

// InitMigrationSteps 将 StepsJSON 初始化为全部 pending 的阶段列表并写库。
func InitMigrationSteps(ctx context.Context, db *gorm.DB, migration *AgentMigration, agentType string) {
	allSteps := MigrationStepsForAgentType(ctx, agentType)
	steps := make([]MigrationStep, len(allSteps))
	for i, s := range allSteps {
		steps[i] = MigrationStep{Step: s, Status: MigrationStepStatusPending, Ts: nil}
	}
	b, _ := json.Marshal(steps)
	migration.StepsJSON = string(b)
	db.Model(migration).Update("steps_json", migration.StepsJSON)
}

// UpdateMigrationStep 更新指定阶段的 status 和 ts 并写库。detail 为 nil 时不覆盖原有值。
func UpdateMigrationStep(db *gorm.DB, migration *AgentMigration, stepName, status string, detail map[string]interface{}) {
	var steps []MigrationStep
	if err := json.Unmarshal([]byte(migration.StepsJSON), &steps); err != nil {
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for i := range steps {
		if steps[i].Step == stepName {
			steps[i].Status = status
			steps[i].Ts = &now
			if detail != nil {
				steps[i].Detail = detail
			}
			break
		}
	}
	b, _ := json.Marshal(steps)
	migration.StepsJSON = string(b)
	db.Model(migration).Update("steps_json", migration.StepsJSON)
}

// ParseMigrationSteps 解析 StepsJSON，出错时返回空切片。
func ParseMigrationSteps(migration *AgentMigration) []MigrationStep {
	if migration.StepsJSON == "" {
		return nil
	}
	var steps []MigrationStep
	if err := json.Unmarshal([]byte(migration.StepsJSON), &steps); err != nil {
		return nil
	}
	return steps
}
