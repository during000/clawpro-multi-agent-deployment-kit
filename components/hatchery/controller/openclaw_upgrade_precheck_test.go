// Package controller 中 openclaw_upgrade_precheck.go 的单元测试。
//
// 覆盖的关键行为：
//   - parseDiskPrecheckOutput：KV 输出的容错解析（正常 / 缺字段 / 非法值 / 空输出）
//   - humanKB：容量数值→人类可读字符串（KB / MB / GB 三档）
//   - diskPrecheckResult.OK：ok / unknown 一律放行，仅 insufficient 判为不足
//   - precheckUpgradeDiskSpaceStep：TAT 通道错误放行 / insufficient 拒绝 / ok 放行
//   - isUpgradeAbortedErr：wrap 后仍能识别
//   - buildAbortedByDiskInsufficient：文案含预估空间 & 可用空间
package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"hatchery/model"
)

// ─── parseDiskPrecheckOutput ─────────────────────────────────────────────────

func TestParseDiskPrecheckOutput_TableDriven(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		wantResult  string
		wantOK      bool
		wantSource  int64
		wantAvail   int64
		wantReason  string
	}{
		{
			name: "ok_full_kv",
			output: strings.Join([]string{
				"=== OpenClaw 升级前存储空间预探测 ===",
				"PRECHECK_SOURCE_KB:1024",
				"PRECHECK_ESTIMATED_KB:410",
				"PRECHECK_REQUIRED_KB:614",
				"PRECHECK_HOME_AVAIL_KB:2048000",
				"PRECHECK_HOME_FS:ext4",
				"PRECHECK_RESULT:ok",
				"PRECHECK_REASON:",
				"=== 探测完成 ===",
			}, "\n"),
			wantResult: "ok",
			wantOK:     true,
			wantSource: 1024,
			wantAvail:  2048000,
			wantReason: "",
		},
		{
			name: "insufficient_result",
			output: strings.Join([]string{
				"PRECHECK_SOURCE_KB:5000000",
				"PRECHECK_ESTIMATED_KB:2000000",
				"PRECHECK_REQUIRED_KB:3000000",
				"PRECHECK_HOME_AVAIL_KB:100000",
				"PRECHECK_HOME_FS:ext4",
				"PRECHECK_RESULT:insufficient",
				"PRECHECK_REASON:home_avail_lt_required",
			}, "\n"),
			wantResult: "insufficient",
			wantOK:     false,
			wantSource: 5000000,
			wantAvail:  100000,
			wantReason: "home_avail_lt_required",
		},
		{
			name:       "empty_output",
			output:     "",
			wantResult: "unknown",
			wantOK:     true, // unknown 归为放行
		},
		{
			name: "only_result_line",
			output: "PRECHECK_RESULT:unknown",
			wantResult: "unknown",
			wantOK:     true,
		},
		{
			name: "invalid_result_value_falls_back_to_unknown",
			output: strings.Join([]string{
				"PRECHECK_RESULT:weird_value",
				"PRECHECK_HOME_AVAIL_KB:100",
			}, "\n"),
			wantResult: "unknown",
			wantOK:     true,
			wantAvail:  100,
		},
		{
			name: "malformed_int_values_are_zero",
			output: strings.Join([]string{
				"PRECHECK_SOURCE_KB:not-a-number",
				"PRECHECK_HOME_AVAIL_KB: 4096 ",
				"PRECHECK_RESULT:ok",
			}, "\n"),
			wantResult: "ok",
			wantOK:     true,
			wantSource: 0,    // 非法数字 → 0
			wantAvail:  4096, // 前后空格能被 TrimSpace 处理
		},
		{
			name: "value_containing_colon_preserved",
			output: strings.Join([]string{
				"PRECHECK_REASON:foo:bar:baz",
				"PRECHECK_RESULT:insufficient",
			}, "\n"),
			wantResult: "insufficient",
			wantOK:     false,
			wantReason: "foo:bar:baz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := parseDiskPrecheckOutput(tt.output)
			if res == nil {
				t.Fatalf("parseDiskPrecheckOutput 返回 nil")
			}
			if res.Result != tt.wantResult {
				t.Errorf("Result = %q, want %q", res.Result, tt.wantResult)
			}
			if res.OK() != tt.wantOK {
				t.Errorf("OK() = %v, want %v", res.OK(), tt.wantOK)
			}
			if res.SourceKB != tt.wantSource {
				t.Errorf("SourceKB = %d, want %d", res.SourceKB, tt.wantSource)
			}
			if res.HomeAvailKB != tt.wantAvail {
				t.Errorf("HomeAvailKB = %d, want %d", res.HomeAvailKB, tt.wantAvail)
			}
			if res.Reason != tt.wantReason {
				t.Errorf("Reason = %q, want %q", res.Reason, tt.wantReason)
			}
		})
	}
}

// nil result 也应视为 OK（跟 err != nil 走同一放行分支的语义）
func TestDiskPrecheckResult_OK_NilReceiver(t *testing.T) {
	var res *diskPrecheckResult
	if !res.OK() {
		t.Errorf("nil *diskPrecheckResult.OK() 应为 true（放行）")
	}
}

// ─── humanKB ────────────────────────────────────────────────────────────────

func TestHumanKB_Formatting(t *testing.T) {
	tests := []struct {
		name string
		kb   int64
		want string
	}{
		{"zero", 0, "0 KB"},
		{"negative", -1, "0 KB"},
		{"pure_kb", 500, "500 KB"},
		{"kb_upper_boundary", 1023, "1023 KB"},
		{"just_mb", 1024, "1.0 MB"},
		{"mb_range", 2048, "2.0 MB"},
		{"mb_upper_boundary", 1024*1024 - 1, "1024.0 MB"},
		{"just_gb", 1024 * 1024, "1.00 GB"},
		{"gb_range", 2 * 1024 * 1024, "2.00 GB"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := humanKB(tt.kb)
			if got != tt.want {
				t.Errorf("humanKB(%d) = %q, want %q", tt.kb, got, tt.want)
			}
		})
	}
}

// ─── isUpgradeAbortedErr ────────────────────────────────────────────────────

func TestIsUpgradeAbortedErr_UnwrapChain(t *testing.T) {
	direct := &errUpgradeAborted{Reason: "disk_insufficient", UserMsg: "test"}
	if aborted, ok := isUpgradeAbortedErr(direct); !ok || aborted != direct {
		t.Errorf("直接传入应识别成功，got ok=%v aborted=%v", ok, aborted)
	}

	// wrap 后依然识别（errors.As 能顺着 %w 链找到）
	wrapped := fmt.Errorf("upstream failed: %w", direct)
	if aborted, ok := isUpgradeAbortedErr(wrapped); !ok || aborted != direct {
		t.Errorf("wrap 后应识别成功，got ok=%v aborted=%v", ok, aborted)
	}

	// 普通 error 不识别
	if aborted, ok := isUpgradeAbortedErr(errors.New("plain error")); ok || aborted != nil {
		t.Errorf("普通 error 应不识别，got ok=%v aborted=%v", ok, aborted)
	}

	// nil 不识别
	if aborted, ok := isUpgradeAbortedErr(nil); ok || aborted != nil {
		t.Errorf("nil error 应不识别，got ok=%v aborted=%v", ok, aborted)
	}
}

func TestErrUpgradeAborted_Error(t *testing.T) {
	e := &errUpgradeAborted{Reason: "disk_insufficient"}
	if got := e.Error(); !strings.Contains(got, "disk_insufficient") {
		t.Errorf("Error() 应包含 reason，got %q", got)
	}
	// nil 保护
	var nilE *errUpgradeAborted
	if got := nilE.Error(); got == "" {
		t.Errorf("nil *errUpgradeAborted.Error() 应有兜底文案")
	}
}

// ─── precheckUpgradeDiskSpaceStep ───────────────────────────────────────────

// stubPrecheckDiskFn 在测试期间替换 precheckUpgradeDiskSpace 变量。
func stubPrecheckDiskFn(t *testing.T, fn func(ctx context.Context, instance *model.Instance) (*diskPrecheckResult, error)) {
	t.Helper()
	orig := precheckUpgradeDiskSpace
	precheckUpgradeDiskSpace = fn
	t.Cleanup(func() { precheckUpgradeDiskSpace = orig })
}

// TAT 通道错误 → 放行（outcome.OK=true）
func TestPrecheckUpgradeDiskSpaceStep_TATError_PassThrough(t *testing.T) {
	stubPrecheckDiskFn(t, func(ctx context.Context, instance *model.Instance) (*diskPrecheckResult, error) {
		return nil, errors.New("tat rpc timeout")
	})
	inst := &model.Instance{InstanceId: "ins-1", RuntimeUser: "root"}
	outcome := precheckUpgradeDiskSpaceStep(context.Background(), inst, "[TestStep]")
	if !outcome.OK {
		t.Fatalf("TAT 通道错误应放行，got outcome=%+v", outcome)
	}
}

// insufficient → HTTP 400 + batch=failed
func TestPrecheckUpgradeDiskSpaceStep_Insufficient_Rejected(t *testing.T) {
	stubPrecheckDiskFn(t, func(ctx context.Context, instance *model.Instance) (*diskPrecheckResult, error) {
		return &diskPrecheckResult{
			SourceKB: 5_000_000, EstimatedKB: 2_000_000, RequiredKB: 3_000_000,
			HomeAvailKB: 100_000, HomeFS: "ext4",
			Result: "insufficient", Reason: "home_avail_lt_required",
		}, nil
	})
	inst := &model.Instance{InstanceId: "ins-2", RuntimeUser: "root"}
	outcome := precheckUpgradeDiskSpaceStep(context.Background(), inst, "[TestStep]")
	if outcome.OK {
		t.Fatalf("insufficient 应拒绝，got outcome.OK=true")
	}
	if outcome.HTTPCode != http.StatusBadRequest {
		t.Errorf("HTTPCode = %d, want %d", outcome.HTTPCode, http.StatusBadRequest)
	}
	if outcome.BatchStatus != "failed" {
		t.Errorf("BatchStatus = %q, want failed", outcome.BatchStatus)
	}
	if outcome.Err == nil {
		t.Fatalf("outcome.Err 不应为 nil")
	}
	// 错误文案应包含 required / avail 的可读字符串
	if !strings.Contains(outcome.Err.Error(), "MB") && !strings.Contains(outcome.Err.Error(), "GB") && !strings.Contains(outcome.Err.Error(), "KB") {
		t.Errorf("错误文案应含容量单位，got %q", outcome.Err.Error())
	}
}

// ok → 放行
func TestPrecheckUpgradeDiskSpaceStep_OK_PassThrough(t *testing.T) {
	stubPrecheckDiskFn(t, func(ctx context.Context, instance *model.Instance) (*diskPrecheckResult, error) {
		return &diskPrecheckResult{
			SourceKB: 100, EstimatedKB: 40, RequiredKB: 60,
			HomeAvailKB: 10_000_000, HomeFS: "ext4",
			Result: "ok",
		}, nil
	})
	inst := &model.Instance{InstanceId: "ins-3", RuntimeUser: "root"}
	outcome := precheckUpgradeDiskSpaceStep(context.Background(), inst, "[TestStep]")
	if !outcome.OK {
		t.Fatalf("ok 应放行，got outcome=%+v", outcome)
	}
}

// unknown → 放行（跟 TAT 错误同一策略）
func TestPrecheckUpgradeDiskSpaceStep_Unknown_PassThrough(t *testing.T) {
	stubPrecheckDiskFn(t, func(ctx context.Context, instance *model.Instance) (*diskPrecheckResult, error) {
		return &diskPrecheckResult{Result: "unknown", Reason: "openclaw_home_not_exist"}, nil
	})
	inst := &model.Instance{InstanceId: "ins-4", RuntimeUser: "root"}
	outcome := precheckUpgradeDiskSpaceStep(context.Background(), inst, "[TestStep]")
	if !outcome.OK {
		t.Fatalf("unknown 应放行，got outcome=%+v", outcome)
	}
}

// nil result → 放行
func TestPrecheckUpgradeDiskSpaceStep_NilResult_PassThrough(t *testing.T) {
	stubPrecheckDiskFn(t, func(ctx context.Context, instance *model.Instance) (*diskPrecheckResult, error) {
		return nil, nil
	})
	inst := &model.Instance{InstanceId: "ins-5", RuntimeUser: "root"}
	outcome := precheckUpgradeDiskSpaceStep(context.Background(), inst, "[TestStep]")
	if !outcome.OK {
		t.Fatalf("nil result 应放行，got outcome=%+v", outcome)
	}
}

// ─── buildAbortedByDiskInsufficient ─────────────────────────────────────────

func TestBuildAbortedByDiskInsufficient_ContainsCapacityInMsg(t *testing.T) {
	res := &diskPrecheckResult{
		RequiredKB:  3 * 1024 * 1024, // 3 GB
		HomeAvailKB: 100 * 1024,      // 100 MB
		Result:     "insufficient",
	}
	aborted := buildAbortedByDiskInsufficient(context.Background(), res)
	if aborted == nil {
		t.Fatalf("buildAbortedByDiskInsufficient 返回 nil")
	}
	if aborted.Reason != "disk_insufficient" {
		t.Errorf("Reason = %q, want disk_insufficient", aborted.Reason)
	}
	if !strings.Contains(aborted.UserMsg, "GB") {
		t.Errorf("UserMsg 应含 GB 单位（required=3GB），got %q", aborted.UserMsg)
	}
	if !strings.Contains(aborted.UserMsg, "MB") {
		t.Errorf("UserMsg 应含 MB 单位（avail=100MB），got %q", aborted.UserMsg)
	}
	if aborted.Detail != res {
		t.Errorf("Detail 应保留探测结果引用")
	}
}
