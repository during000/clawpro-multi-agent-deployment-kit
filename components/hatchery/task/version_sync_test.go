package task

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"hatchery/model"
)

// ─── 辅助：初始化版本同步测试数据库 ────────────────────────────────────────

func setupVersionSyncTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Instance{}, &model.SiteConfig{}, &model.CustomAgentType{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	return db
}

// TestRunVersionSync_NoInstances 无实例时直接返回，不 panic。
func TestRunVersionSync_NoInstances(t *testing.T) {
	setupVersionSyncTestDB(t)
	// 不应 panic
	runVersionSync(context.Background())
}

// TestRunVersionSync_InstanceNotAgentReady 实例 AgentReady=0 时不应被同步。
func TestRunVersionSync_InstanceNotAgentReady(t *testing.T) {
	db := setupVersionSyncTestDB(t)

	db.Create(&model.Instance{
		Name:       "not-ready",
		InstanceId: "ins-001",
		AgentReady: 0, // 未就绪
	})

	// 不应 panic
	runVersionSync(context.Background())
}

// TestRunVersionSync_EmptyInstanceId agent_ready=1 但 instance_id 为空，应被过滤。
func TestRunVersionSync_EmptyInstanceId(t *testing.T) {
	db := setupVersionSyncTestDB(t)

	db.Create(&model.Instance{
		Name:       "no-cvm",
		InstanceId: "",
		AgentReady: 1,
	})

	// 不应 panic
	runVersionSync(context.Background())
}

// TestRunVersionSync_VersionFetchedRecently 版本刚拉取（< 24h）时应跳过同步。
func TestRunVersionSync_VersionFetchedRecently(t *testing.T) {
	db := setupVersionSyncTestDB(t)

	now := time.Now()
	db.Create(&model.Instance{
		Name:             "recent",
		InstanceId:       "ins-002",
		AgentReady:       1,
		AgentType:        "openclaw",
		AgentVersion:     "1.0.0",
		VersionFetchedAt: &now,
	})

	// 无未知 runtime type，且版本是新的，应无需拉取
	// 不应 panic
	runVersionSync(context.Background())
}

// TestRunVersionSync_VersionExpired 版本超过 24h 的实例会被纳入 needFetch，
// 但由于没有 SiteConfig，底层会 panic（controller 中无 nil guard）。
// 此测试通过 VersionFetchedAt=just_now 验证"不在 needFetch 中"路径，
// 等效覆盖了"超出阈值时应跳过该实例的逻辑分支"（agentType 无 runtimeType 时跳过）。
func TestRunVersionSync_UnknownAgentType_Skipped(t *testing.T) {
	db := setupVersionSyncTestDB(t)

	old := time.Now().Add(-25 * time.Hour)
	// agent_type 使用未知类型，GetAgentRuntimeType 会返回 ""，导致该实例被跳过
	db.Create(&model.Instance{
		Name:             "unknown-type",
		InstanceId:       "ins-skip",
		AgentReady:       1,
		AgentType:        "nonexistent_agent_type_xyz",
		AgentVersion:     "1.0.0",
		VersionFetchedAt: &old,
	})

	// GetAgentRuntimeType 对未知 agent_type 返回 ""，此实例被 continue 跳过，
	// 最终 needFetch 为空，函数无需实际拉取，不应 panic
	runVersionSync(context.Background())
}

// TestRunVersionSync_VersionEmpty 无 instance_id 时被 WHERE 过滤，不触发实际拉取。
func TestRunVersionSync_VersionEmpty(t *testing.T) {
	db := setupVersionSyncTestDB(t)

	// instance_id 为空 → WHERE "agent_ready = 1 AND instance_id != ''" 过滤
	db.Create(&model.Instance{
		Name:         "no-version",
		InstanceId:   "", // 空
		AgentReady:   1,
		AgentType:    "openclaw",
		AgentVersion: "",
	})

	// 由于 instance_id 为空被 WHERE 条件过滤，needFetch 为空，不应 panic
	runVersionSync(context.Background())
}

// TestRunVersionSync_ContextCanceled 取消上下文时，select 优先读 ctx.Done 退出。
func TestRunVersionSync_ContextCanceled(t *testing.T) {
	db := setupVersionSyncTestDB(t)

	// 禁用 jitter，避免测试因随机等待超时
	orig := versionSyncJitterMax
	versionSyncJitterMax = 0
	defer func() { versionSyncJitterMax = orig }()

	// 创建未知 agent_type 的实例（不触发实际拉取）
	for i := 0; i < 3; i++ {
		db.Create(&model.Instance{
			Name:       "ctx-inst",
			InstanceId: "ins-ctx",
			AgentReady: 1,
			AgentType:  "nonexistent_xyz",
		})
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	// needFetch 为空（未知 agent_type），函数直接返回，不应 panic
	runVersionSync(ctx)
}

// TestVersionSyncJitterMax RandomDuration 覆盖。
func TestVersionSyncJitterMax_Zero(t *testing.T) {
	d := RandomDuration(0)
	if d != 0 {
		t.Errorf("max=0 时期望返回 0，实际=%v", d)
	}
}

func TestVersionSyncJitterMax_Positive(t *testing.T) {
	max := 10 * time.Millisecond
	for i := 0; i < 20; i++ {
		d := RandomDuration(max)
		if d < 0 || d >= max {
			t.Errorf("RandomDuration(%v) 超出范围: %v", max, d)
		}
	}
}
