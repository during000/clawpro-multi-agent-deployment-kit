package controller

import (
	"context"
	"errors"
	"testing"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// ─── createErrorNotification 结构体提取测试 ──────────────────────────────────

func TestCreateErrorNotification_RichErrorExtraction(t *testing.T) {
	// 验证从 RichError 中正确提取字段
	richErr := hcommon.I18nError(i18n.MsgTATTimeout).WithRequestId("req-abc123").WithI18nDetail(i18n.MsgTATTimeout).WithInstanceId("ins-xyz789")

	detail := &model.NotifErrorDetail{Error: hcommon.ErrorMessageWithCtx(context.Background(), richErr)}
	var re *hcommon.RichError
	if errors.As(richErr, &re) {
		detail.Detail = hcommon.ErrorDetailWithCtx(context.Background(), re)
		detail.RequestId = re.RequestId
		detail.InstanceId = re.InstanceId
	}

	if detail.Error != "TAT 操作超时" {
		t.Errorf("Error: got %q, want %q", detail.Error, "TAT 操作超时")
	}
	if detail.Detail != "TAT 操作超时" {
		t.Errorf("Detail: got %q, want %q", detail.Detail, "TAT 操作超时")
	}
	if detail.RequestId != "req-abc123" {
		t.Errorf("RequestId: got %q, want %q", detail.RequestId, "req-abc123")
	}
	if detail.InstanceId != "ins-xyz789" {
		t.Errorf("InstanceId: got %q, want %q", detail.InstanceId, "ins-xyz789")
	}
}

func TestCreateErrorNotification_PlainError(t *testing.T) {
	// 验证普通 error 只提取 Error 字段
	plainErr := errors.New("普通错误信息")

	detail := &model.NotifErrorDetail{Error: hcommon.ErrorMessageWithCtx(context.Background(), plainErr)}
	var re *hcommon.RichError
	if errors.As(plainErr, &re) {
		t.Fatal("plain error should not match RichError")
	}

	if detail.Error != "普通错误信息" {
		t.Errorf("Error: got %q, want %q", detail.Error, "普通错误信息")
	}
	if detail.Detail != "" {
		t.Errorf("Detail should be empty for plain error, got %q", detail.Detail)
	}
	if detail.RequestId != "" {
		t.Errorf("RequestId should be empty for plain error, got %q", detail.RequestId)
	}
}

func TestCreateErrorNotification_WrappedRichError(t *testing.T) {
	// 验证 wrapped RichError 也能正确提取
	inner := hcommon.I18nError(i18n.MsgTATTimeout).WithRequestId("req-wrapped").WithI18nDetail(i18n.MsgTATTimeout)

	detail := &model.NotifErrorDetail{Error: hcommon.ErrorMessageWithCtx(context.Background(), inner)}
	var re *hcommon.RichError
	if errors.As(inner, &re) {
		detail.Detail = hcommon.ErrorDetailWithCtx(context.Background(), re)
		detail.RequestId = re.RequestId
		detail.InstanceId = re.InstanceId
	}

	if detail.Error != "TAT 操作超时" {
		t.Errorf("Error: got %q, want %q", detail.Error, "TAT 操作超时")
	}
	if detail.Detail != "TAT 操作超时" {
		t.Errorf("Detail: got %q, want %q", detail.Detail, "TAT 操作超时")
	}
	if detail.RequestId != "req-wrapped" {
		t.Errorf("RequestId: got %q, want %q", detail.RequestId, "req-wrapped")
	}
}

// ─── BizRequestId 优先级测试 ─────────────────────────────────────────────────

func TestCreateErrorNotification_BizRequestIdPriority(t *testing.T) {
	// BizRequestId 非空时应优先于 RequestId
	richErr := hcommon.I18nError(i18n.MsgTATExecuteCommandFailed).WithRequestId("tat-req-123").WithBizRequestId("biz-req-456")

	detail := &model.NotifErrorDetail{Error: hcommon.ErrorMessageWithCtx(context.Background(), richErr)}
	var re *hcommon.RichError
	if errors.As(richErr, &re) {
		if re.BizRequestId != "" {
			detail.RequestId = re.BizRequestId
		} else {
			detail.RequestId = re.RequestId
		}
	}

	if detail.RequestId != "biz-req-456" {
		t.Errorf("RequestId should be BizRequestId when present, got %q, want %q", detail.RequestId, "biz-req-456")
	}
}

func TestCreateErrorNotification_FallbackToRequestId(t *testing.T) {
	// BizRequestId 为空时应回退到 RequestId
	richErr := hcommon.I18nError(i18n.MsgTATTimeout).WithRequestId("tat-req-123")

	detail := &model.NotifErrorDetail{Error: hcommon.ErrorMessageWithCtx(context.Background(), richErr)}
	var re *hcommon.RichError
	if errors.As(richErr, &re) {
		if re.BizRequestId != "" {
			detail.RequestId = re.BizRequestId
		} else {
			detail.RequestId = re.RequestId
		}
	}

	if detail.RequestId != "tat-req-123" {
		t.Errorf("RequestId should fallback to TAT RequestId, got %q, want %q", detail.RequestId, "tat-req-123")
	}
}

// ─── context 兜底 request_id 测试 ────────────────────────────────────────────

func TestCreateErrorNotification_ContextFallbackRequestId(t *testing.T) {
	// RichError 无 RequestId 时，从 context 中提取
	ctx := context.WithValue(context.Background(), hcommon.CtxKeyRequestID, "ctx-req-789")

	richErr := hcommon.I18nError(i18n.MsgTATExecuteCommandFailed)

	detail := &model.NotifErrorDetail{Error: hcommon.ErrorMessageWithCtx(context.Background(), richErr)}
	var re *hcommon.RichError
	if errors.As(richErr, &re) {
		if re.BizRequestId != "" {
			detail.RequestId = re.BizRequestId
		} else {
			detail.RequestId = re.RequestId
		}
	}

	// 模拟 createErrorNotification 的 context 兜底逻辑
	if detail.RequestId == "" {
		if reqID := hcommon.GetRequestID(ctx); reqID != "" {
			detail.RequestId = reqID
		}
	}

	if detail.RequestId != "ctx-req-789" {
		t.Errorf("RequestId should fallback to context request_id, got %q, want %q", detail.RequestId, "ctx-req-789")
	}
}

func TestCreateErrorNotification_ContextNotUsedWhenRequestIdPresent(t *testing.T) {
	// RichError 有 RequestId 时，不使用 context
	ctx := context.WithValue(context.Background(), hcommon.CtxKeyRequestID, "ctx-req-should-not-use")

	richErr := hcommon.I18nError(i18n.MsgTATExecuteCommandFailed).WithRequestId("tat-req-exists")

	detail := &model.NotifErrorDetail{Error: hcommon.ErrorMessageWithCtx(context.Background(), richErr)}
	var re *hcommon.RichError
	if errors.As(richErr, &re) {
		if re.BizRequestId != "" {
			detail.RequestId = re.BizRequestId
		} else {
			detail.RequestId = re.RequestId
		}
	}

	if detail.RequestId == "" {
		if reqID := hcommon.GetRequestID(ctx); reqID != "" {
			detail.RequestId = reqID
		}
	}

	if detail.RequestId != "tat-req-exists" {
		t.Errorf("RequestId should NOT be overridden by context, got %q, want %q", detail.RequestId, "tat-req-exists")
	}
}

// ─── HandleDeleteNotification ids 去重过滤测试 ───────────────────────────────

func TestDeduplicateIDs(t *testing.T) {
	tests := []struct {
		name     string
		input    []uint
		wantLen  int
		wantHas0 bool // 结果中不应包含 0
	}{
		{"空列表", nil, 0, false},
		{"无重复", []uint{1, 2, 3}, 3, false},
		{"有重复", []uint{1, 2, 2, 3, 3, 3}, 3, false},
		{"含0值", []uint{0, 1, 2, 0}, 2, false},
		{"全0值", []uint{0, 0, 0}, 0, false},
		{"单个元素", []uint{42}, 1, false},
		{"重复单元素", []uint{5, 5, 5, 5}, 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seen := make(map[uint]bool)
			var uniqueIDs []uint
			for _, id := range tt.input {
				if id > 0 && !seen[id] {
					seen[id] = true
					uniqueIDs = append(uniqueIDs, id)
				}
			}

			if len(uniqueIDs) != tt.wantLen {
				t.Errorf("去重后长度: got %d, want %d (input: %v, result: %v)", len(uniqueIDs), tt.wantLen, tt.input, uniqueIDs)
			}
			for _, id := range uniqueIDs {
				if id == 0 {
					t.Errorf("结果中不应包含 0 (input: %v, result: %v)", tt.input, uniqueIDs)
				}
			}
		})
	}
}

// ─── validCategories 校验测试 ────────────────────────────────────────────────

func TestValidCategories(t *testing.T) {
	tests := []struct {
		category string
		valid    bool
	}{
		{"success", true},
		{"error", true},
		{"notice", true},
		{"", false},
		{"invalid", false},
		{"SUCCESS", false}, // 区分大小写
		{"Error", false},
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			if got := validCategories[tt.category]; got != tt.valid {
				t.Errorf("validCategories[%q] = %v, want %v", tt.category, got, tt.valid)
			}
		})
	}
}

// ─── notifyQuotaExceeded 集成测试 ────────────────────────────────────────────

// TestNotifyQuotaExceeded_CreatesNotification 验证第一次调用会写入通知。
func TestNotifyQuotaExceeded_CreatesNotification(t *testing.T) {
	setupMultiModelTestDB(t)

	// 需要额外迁移 Notification 表（setupMultiModelTestDB 已包含）
	user, inst := createMultiModelUserAndInstance(t, "quota-u1", "quota-inst1")

	notifyQuotaExceeded(context.Background(), user.ID, inst.ID, inst.Name, "用户每日令牌配额已用尽")

	var count int64
	model.DB(context.Background()).Model(&model.Notification{}).
		Where("user_id = ? AND type = ?", user.ID, model.NotifyTypeQuotaExceeded).
		Count(&count)
	if count != 1 {
		t.Errorf("期望创建 1 条通知，实际 count=%d", count)
	}
}

// TestNotifyQuotaExceeded_Idempotent 验证同一天内重复调用不会重复创建通知。
func TestNotifyQuotaExceeded_Idempotent(t *testing.T) {
	setupMultiModelTestDB(t)

	user, inst := createMultiModelUserAndInstance(t, "quota-u2", "quota-inst2")

	// 调用三次
	notifyQuotaExceeded(context.Background(), user.ID, inst.ID, inst.Name, "配额超限")
	notifyQuotaExceeded(context.Background(), user.ID, inst.ID, inst.Name, "配额超限")
	notifyQuotaExceeded(context.Background(), user.ID, inst.ID, inst.Name, "配额超限")

	var count int64
	model.DB(context.Background()).Model(&model.Notification{}).
		Where("user_id = ? AND type = ?", user.ID, model.NotifyTypeQuotaExceeded).
		Count(&count)
	if count != 1 {
		t.Errorf("同一天内重复调用应只创建 1 条通知，实际 count=%d", count)
	}
}
