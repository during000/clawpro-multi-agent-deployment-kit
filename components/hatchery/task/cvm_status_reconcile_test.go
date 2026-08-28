package task

import (
	"context"
	"testing"
)

// TestRunCVMStatusReconcile_NoPanic 验证 runCVMStatusReconcile 在有 panic recovery 的情况下不会崩溃。
func TestRunCVMStatusReconcile_NoPanic(t *testing.T) {
	// 无需真实 DB，panic recovery 包装器能安全处理任意 ctx
	runCVMStatusReconcile(context.Background())
}
