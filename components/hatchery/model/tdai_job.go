package model

import (
	"context"
	"log/slog"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"

	"gorm.io/gorm"
)

// TdaiJob 状态常量
const (
	TdaiJobStatePending   = "PENDING"
	TdaiJobStateRunning   = "RUNNING"
	TdaiJobStateSucceeded = "SUCCEEDED"
	TdaiJobStateFailed    = "FAILED"
	TdaiJobStateCanceled  = "CANCELED"
)

// TdaiJob 任务类型常量
const (
	TdaiJobTypeSwitchToFree = "SWITCH_TO_FREE"
	TdaiJobTypeSwitchToOff  = "SWITCH_TO_OFF"
	TdaiJobTypeSwitchToPro  = "SWITCH_TO_PRO"
)

const (
	TdaiJobDefaultMaxAttempts   = 10  // 默认最大执行次数（含首次）
	TdaiJobDefaultBackoffBase   = 5   // 秒
	TdaiJobDefaultBackoffFactor = 6   // 指数退避因子：5s, 30s, 180s, 180s, ...
	TdaiJobDefaultBackoffMax    = 180 // 退避上限（秒），避免等待过长
)

// TdaiJob 轻量任务流主表。
type TdaiJob struct {
	ID         uint           `gorm:"primarykey" json:"id"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
	Identifier string         `gorm:"size:191;index;default:''" json:"-"`

	// 业务标识（size 与 sql/init.sql varchar(N) 保持一致）
	JobType    string `gorm:"size:64;not null;index" json:"job_type"`
	BizKey     string `gorm:"size:191;index:idx_identifier_biz_key;not null" json:"biz_key"`
	InstanceID string `gorm:"size:191;index:idx_instance_state;not null;default:''" json:"instance_id"`

	// 状态
	State       string `gorm:"size:32;index:idx_instance_state;index:idx_state_run_at;not null;default:PENDING" json:"state"`
	CurrentStep string `gorm:"size:64;not null;default:''" json:"current_step"`
	Progress    int    `gorm:"not null;default:0" json:"progress"` // 0~100

	// 调度
	RunAt       time.Time `gorm:"index:idx_state_run_at;not null" json:"run_at"`
	Attempt     int       `gorm:"not null;default:0" json:"attempt"`
	MaxAttempts int       `gorm:"not null;default:3" json:"max_attempts"`

	// 租约（多实例抢占）
	LeaseOwner string     `gorm:"size:191;default:''" json:"-"`
	LeaseUntil *time.Time `json:"-"`

	// 数据
	PayloadJSON string `gorm:"type:text" json:"payload_json,omitempty"`
	ResultJSON  string `gorm:"type:text" json:"result_json,omitempty"`
	LastError   string `gorm:"type:text" json:"last_error,omitempty"`
	ErrorCode   string `gorm:"size:64;default:''" json:"error_code,omitempty"`

	// 审计
	Operator   string     `gorm:"size:191;default:''" json:"operator,omitempty"`
	TraceID    string     `gorm:"size:191;default:''" json:"trace_id,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

// SubmitJob 幂等提交任务。
// 若同 biz_key 已存在 PENDING/RUNNING 状态的任务，直接返回已有任务（不重复创建）。
func SubmitJob(ctx context.Context, jobType, bizKey, instanceID, payloadJSON, operator, traceID string) (*TdaiJob, error) {
	if bizKey == "" {
		return nil, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "biz_key")
	}

	// 幂等检查：同 biz_key 是否已有活跃任务
	var existing TdaiJob
	err := DB(ctx).Where("biz_key = ? AND state IN ?", bizKey, []string{TdaiJobStatePending, TdaiJobStateRunning}).First(&existing).Error
	if err == nil {
		return &existing, nil
	}

	job := TdaiJob{
		JobType:     jobType,
		BizKey:      bizKey,
		InstanceID:  instanceID,
		State:       TdaiJobStatePending,
		CurrentStep: "",
		RunAt:       time.Now(),
		Attempt:     0,
		MaxAttempts: TdaiJobDefaultMaxAttempts,
		PayloadJSON: payloadJSON,
		Operator:    operator,
		TraceID:     traceID,
	}
	if err := DB(ctx).Create(&job).Error; err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgFailedToCreateTask)
	}
	slog.Info("[SubmitJob] 任务已创建",
		"job_id", job.ID, "job_type", jobType, "biz_key", bizKey, "instance_id", instanceID)
	return &job, nil
}

// RetryJob 将失败任务重置为 PENDING，供重新调度。
func RetryJob(ctx context.Context, jobID uint) error {
	result := DB(ctx).Model(&TdaiJob{}).
		Where("id = ? AND state = ?", jobID, TdaiJobStateFailed).
		Updates(map[string]any{
			"state":      TdaiJobStatePending,
			"run_at":     time.Now(),
			"last_error": "",
			"error_code": "",
		})
	if result.RowsAffected == 0 {
		return hcommon.I18nError(i18n.MsgTDAIJobRetryFailed, jobID)
	}
	return nil
}

// CancelJob 取消 PENDING 状态的任务。
func CancelJob(ctx context.Context, jobID uint) error {
	now := time.Now()
	result := DB(ctx).Model(&TdaiJob{}).
		Where("id = ? AND state = ?", jobID, TdaiJobStatePending).
		Updates(map[string]any{
			"state":       TdaiJobStateCanceled,
			"finished_at": &now,
		})
	if result.RowsAffected == 0 {
		return hcommon.I18nError(i18n.MsgTDAIJobCancelFailed, jobID)
	}
	return nil
}
