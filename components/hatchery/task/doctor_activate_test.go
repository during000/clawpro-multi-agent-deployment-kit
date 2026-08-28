package task

import (
	"context"
	"testing"
	"time"

	"hatchery/model"
)

// ─── 测试 ───────────────────────────────────────────────────────────────────

func TestActivateDoctorSessions_Creating超时标记Failed(t *testing.T) {
	cleanup := initCleanupTestDB(t)
	defer cleanup()

	session := model.DoctorSession{
		UserID: 1, TargetInstanceID: 1,
		Status: model.DoctorStatusCreating,
	}
	model.DB(context.Background()).Create(&session)
	model.DB(context.Background()).Exec(
		"UPDATE doctor_sessions SET created_at = ? WHERE id = ?",
		time.Now().Add(-15*time.Minute), session.ID)

	processDoctorActivate(context.Background())

	var updated model.DoctorSession
	model.DB(context.Background()).First(&updated, session.ID)
	if updated.Status != model.DoctorStatusFailed {
		t.Errorf("status = %s, want failed", updated.Status)
	}
}
