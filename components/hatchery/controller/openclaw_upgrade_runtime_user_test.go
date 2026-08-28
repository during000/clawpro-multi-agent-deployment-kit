// openclaw_upgrade_runtime_user_test.go
//
// 针对 ensureOfficialImageRuntimeUser 的单元测试。
// 该函数是 OpenClaw 一键升级 / 升级重试的前置校正步骤，仅在以下情况下
// 才会把 DB 里的 runtime_user/runtime_home 强制对齐为 root / /root：
//  1. instance.AgentType == openclaw（含空字符串存量兼容）
//  2. imageId 命中 hcommon.CandidateImages（官方公共镜像）
//
// 本测试覆盖以下分支：
//   - instance == nil / imageId == "" → 无操作
//   - AgentType 为 hermes / lightclawace → 跳过（防御性判断）
//   - AgentType 为 openclaw + 非官方镜像 → 跳过
//   - AgentType 为 openclaw + 官方镜像 + 已符合预期 → 幂等
//   - AgentType 为 openclaw + 官方镜像 + 不一致 → 覆盖并同步内存
//   - AgentType 为空（存量）+ 官方镜像 → 按 openclaw 处理
//   - DB 更新失败 → 返回错误
package controller

import (
	"testing"

	hcommon "hatchery/common"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// initRuntimeUserTestDB 为 ensureOfficialImageRuntimeUser 构造一个纯内存 SQLite 库。
// 返回 cleanup，测试结束时恢复原 model.DB。
func initRuntimeUserTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&model.Instance{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	restore := model.UseDBForTest(db)
	return restore
}

// pickCandidateImageByType 从 hcommon.CandidateImages 中取出指定 AgentType 的第一个官方镜像 ID。
// 若找不到，测试直接 Fatal，防止候选镜像列表缩减后测试静默失效。
func pickCandidateImageByType(t *testing.T, agentType string) string {
	t.Helper()
	for _, c := range hcommon.CandidateImages {
		if c.AgentType == agentType {
			return c.ImageId
		}
	}
	t.Fatalf("CandidateImages 未包含 agent_type=%s 的镜像，测试无法继续", agentType)
	return ""
}

// ───────────────────────────────────────────────────────────────────
// 早退分支：instance == nil / imageId == ""
// ───────────────────────────────────────────────────────────────────

func TestEnsureOfficialImageRuntimeUser_NilInstance(t *testing.T) {
	t.Skip("ensureOfficialImageRuntimeUser removed in Release")
}

func TestEnsureOfficialImageRuntimeUser_EmptyImageId(t *testing.T) {
	t.Skip("ensureOfficialImageRuntimeUser removed in Release")
}

// ───────────────────────────────────────────────────────────────────
// AgentType 防御性判断：非 OpenClaw（hermes / lightclawace）即便命中官方镜像也必须跳过
// ───────────────────────────────────────────────────────────────────

func TestEnsureOfficialImageRuntimeUser_SkipHermes(t *testing.T) {
	t.Skip("ensureOfficialImageRuntimeUser removed in Release")
}

func TestEnsureOfficialImageRuntimeUser_SkipLightclawACE(t *testing.T) {
	t.Skip("ensureOfficialImageRuntimeUser removed in Release")
}

// ───────────────────────────────────────────────────────────────────
// OpenClaw + 非官方镜像：跳过
// ───────────────────────────────────────────────────────────────────

func TestEnsureOfficialImageRuntimeUser_OpenClawNonCandidateImage(t *testing.T) {
	t.Skip("ensureOfficialImageRuntimeUser removed in Release")
}

// ───────────────────────────────────────────────────────────────────
// OpenClaw + 官方镜像 + 已符合预期：幂等，无更新
// ───────────────────────────────────────────────────────────────────

func TestEnsureOfficialImageRuntimeUser_OfficialImageAlreadyRoot(t *testing.T) {
	t.Skip("ensureOfficialImageRuntimeUser removed in Release")
}

// ───────────────────────────────────────────────────────────────────
// OpenClaw + 官方镜像 + 不一致：强制覆盖为 root/root，并同步内存对象
// ───────────────────────────────────────────────────────────────────

func TestEnsureOfficialImageRuntimeUser_OfficialImageOverwrite(t *testing.T) {
	t.Skip("ensureOfficialImageRuntimeUser removed in Release")
}

// 仅 runtime_user 不一致（RuntimeHome 恰好是 /root）也必须进入校正分支，一起写回。
func TestEnsureOfficialImageRuntimeUser_OnlyUserDrift(t *testing.T) {
	t.Skip("ensureOfficialImageRuntimeUser removed in Release")
}

// ───────────────────────────────────────────────────────────────────
// AgentType 为空（存量兼容）：应按 OpenClaw 处理，官方镜像校正生效
// ───────────────────────────────────────────────────────────────────

func TestEnsureOfficialImageRuntimeUser_LegacyEmptyAgentType(t *testing.T) {
	t.Skip("ensureOfficialImageRuntimeUser removed in Release")
}

// ───────────────────────────────────────────────────────────────────
// DB 更新失败分支：关闭底层 *sql.DB 制造错误，验证错误被包装后返回
// ───────────────────────────────────────────────────────────────────

func TestEnsureOfficialImageRuntimeUser_DBUpdateFailure(t *testing.T) {
	t.Skip("ensureOfficialImageRuntimeUser removed in Release")
}

// ───────────────────────────────────────────────────────────────────
// ensureOfficialImageRuntimeUserForUpgrade：入口侧薄包装
// 目的：保证 HandleUpgrade / HandleUpgradeRetry 复用的"日志前缀 + 错误透传"分支
// 在单元测试层面可被直接覆盖，无需启动整套 HTTP + CVM + TAT 依赖。
// ───────────────────────────────────────────────────────────────────

// 传入 nil 实例：底层 ensureOfficialImageRuntimeUser 会立即 return nil，
// 包装函数应同样返回 nil，不得 panic。
func TestEnsureOfficialImageRuntimeUserForUpgrade_NilInstance(t *testing.T) {
	t.Skip("ensureOfficialImageRuntimeUser removed in Release")
}

// 官方镜像 + drift：包装函数应调用底层函数完成校正并返回 nil；
// 实例内存/DB 的 runtime_user/home 都应被改写为 root / /root。
func TestEnsureOfficialImageRuntimeUserForUpgrade_SuccessPath(t *testing.T) {
	t.Skip("ensureOfficialImageRuntimeUser removed in Release")
}

// 官方镜像 + DB 失败：期望底层 error 原样透传（不被吞或重新包装），
// 同时 wrap 的错误前缀仍应保留（说明走到了底层的 fmt.Errorf 分支）。
// 这个用例正对应 HandleUpgrade / HandleUpgradeRetry 中"ensureOfficialImageRuntimeUserForUpgrade 返回非 nil
// → writeError(500) + return"的失败路径。
func TestEnsureOfficialImageRuntimeUserForUpgrade_PropagatesError(t *testing.T) {
	t.Skip("ensureOfficialImageRuntimeUser removed in Release")
}
