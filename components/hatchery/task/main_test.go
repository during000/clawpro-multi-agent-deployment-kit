package task

import (
	"os"
	"testing"

	"hatchery/common"
)

// TestMain 是 task 包的测试入口。
func TestMain(m *testing.M) {
	// 这是在 init 函数或测试中使用的，因此必须在 TestMain 中预先初始化。
	if common.FixedSnapshot == nil {
		common.FixedSnapshot = &common.TenantSnapshot{}
	}

	os.Exit(m.Run())
}
