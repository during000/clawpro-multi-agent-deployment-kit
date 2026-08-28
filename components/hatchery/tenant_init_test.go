package main

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"hatchery/common"
	"hatchery/model"
)

var backfillTestDBCounter int64

func setupBackfillDB(t *testing.T) {
	t.Helper()
	id := atomic.AddInt64(&backfillTestDBCounter, 1)
	dsn := fmt.Sprintf("file:backfillTest%d?mode=memory&cache=shared", id)
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite mem: %v", err)
	}
	if err := gdb.AutoMigrate(&model.SiteConfig{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	restore := model.UseDBForTest(gdb)
	t.Cleanup(func() {
		restore()
		common.FixedSnapshot = nil
	})
}

// ==================== backfillSiteConfig 单测 ====================

func TestBackfillSiteConfig_FillsEmptyFields(t *testing.T) {
	setupBackfillDB(t)
	model.DB(context.Background()).Create(&model.SiteConfig{Name: "ClawPro", Identifier: "tenant-a"})

	backfillSiteConfig(CmdlineConfig{Identifier: "tenant-a", Uin: "uin-1", Domain: "a.example.com", InternalSecret: "secret-1", Lang: "zh"})

	var got model.SiteConfig
	model.DB(context.Background()).First(&got)
	if got.Uin != "uin-1" {
		t.Fatalf("uin 未回填: %q", got.Uin)
	}
	if got.Domain != "a.example.com" {
		t.Fatalf("domain 未回填: %q", got.Domain)
	}
	if got.InternalSecret != "secret-1" {
		t.Fatalf("internal_secret 未回填: %q", got.InternalSecret)
	}
}

func TestBackfillSiteConfig_FillsUserLimit(t *testing.T) {
	t.Skip("多租户阶段一：user_limit 不再回填，始终返回无限制")
}

func TestBackfillSiteConfig_PreservesExistingFields(t *testing.T) {
	setupBackfillDB(t)
	model.DB(context.Background()).Create(&model.SiteConfig{
		Name:           "ClawPro",
		Uin:            "db-uin",
		Domain:         "db-domain",
		InternalSecret: "db-secret",
	})

	// DB 字段已有值，不应被覆盖
	backfillSiteConfig(CmdlineConfig{Identifier: "tenant-b", Uin: "arg-uin", Domain: "arg-domain", InternalSecret: "arg-secret", Lang: "zh"})

	var got model.SiteConfig
	model.DB(context.Background()).First(&got)
	if got.Uin != "db-uin" {
		t.Fatalf("DB 已有值不应被覆盖: uin=%q", got.Uin)
	}
	if got.Domain != "db-domain" {
		t.Fatalf("DB 已有值不应被覆盖: domain=%q", got.Domain)
	}
	if got.InternalSecret != "db-secret" {
		t.Fatalf("DB 已有值不应被覆盖: internal_secret=%q", got.InternalSecret)
	}
}

func TestBackfillSiteConfig_PartialFill(t *testing.T) {
	setupBackfillDB(t)
	model.DB(context.Background()).Create(&model.SiteConfig{
		Name:       "ClawPro",
		Identifier: "tenant-c",
		Uin:        "db-uin",
		Domain:     "db-domain",
	})

	backfillSiteConfig(CmdlineConfig{Identifier: "tenant-c", Uin: "arg-uin", Domain: "arg-domain", InternalSecret: "arg-secret", Lang: "zh"})

	var got model.SiteConfig
	model.DB(context.Background()).First(&got)
	if got.Uin != "db-uin" || got.Domain != "db-domain" {
		t.Fatalf("已有值不应被覆盖: %+v", got)
	}
	if got.InternalSecret != "arg-secret" {
		t.Fatalf("空字段应被回填: internal_secret=%q", got.InternalSecret)
	}
}

func TestBackfillSiteConfig_EnvOneID(t *testing.T) {
	setupBackfillDB(t)
	model.DB(context.Background()).Create(&model.SiteConfig{Name: "ClawPro", Identifier: "tenant-d"})

	t.Setenv("ONEID_ACCOUNT_ID", "env-acc")
	backfillSiteConfig(CmdlineConfig{Identifier: "tenant-d", Uin: "", Domain: "", InternalSecret: "", Lang: "zh"})

	var got model.SiteConfig
	model.DB(context.Background()).First(&got)
	if got.OneIDAccountID != "env-acc" {
		t.Fatalf("OneID 环境变量未回填: %q", got.OneIDAccountID)
	}
}

func TestBackfillSiteConfig_EnvOneID_PreservesExisting(t *testing.T) {
	setupBackfillDB(t)
	model.DB(context.Background()).Create(&model.SiteConfig{Name: "ClawPro", OneIDAccountID: "db-oneid"})

	t.Setenv("ONEID_ACCOUNT_ID", "env-oneid")
	backfillSiteConfig(CmdlineConfig{Identifier: "tenant-e", Uin: "", Domain: "", InternalSecret: "", Lang: "zh"})

	var got model.SiteConfig
	model.DB(context.Background()).First(&got)
	if got.OneIDAccountID != "db-oneid" {
		t.Fatalf("DB 已有值不应被环境变量覆盖: %q", got.OneIDAccountID)
	}
}

func TestBackfillSiteConfig_EnvSecrets(t *testing.T) {
	setupBackfillDB(t)
	model.DB(context.Background()).Create(&model.SiteConfig{Name: "ClawPro", Identifier: "tenant-f"})

	t.Setenv("SECRET_ID", "env-secret-id")
	t.Setenv("SECRET_KEY", "env-secret-key")
	backfillSiteConfig(CmdlineConfig{Identifier: "tenant-f", Uin: "", Domain: "", InternalSecret: "", Lang: "zh"})

	var got model.SiteConfig
	model.DB(context.Background()).First(&got)
	if got.CVMSecretId != "env-secret-id" || got.CVMSecretKey != "env-secret-key" {
		t.Fatalf("AKSK 环境变量未回填: id=%q key=%q", got.CVMSecretId, got.CVMSecretKey)
	}
}

func TestBackfillSiteConfig_EnvAgentCreds(t *testing.T) {
	setupBackfillDB(t)
	model.DB(context.Background()).Create(&model.SiteConfig{Name: "ClawPro", Identifier: "tenant-g"})

	t.Setenv("AGENT_CAM_ROLE_SECRET_ID", "env-agent-id")
	t.Setenv("AGENT_CAM_ROLE_SECRET_KEY", "env-agent-key")
	backfillSiteConfig(CmdlineConfig{Identifier: "tenant-g", Uin: "", Domain: "", InternalSecret: "", Lang: "zh"})

	var got model.SiteConfig
	model.DB(context.Background()).First(&got)
	if got.AgentCamRoleSecretId != "env-agent-id" || got.AgentCamRoleSecretKey != "env-agent-key" {
		t.Fatalf("Agent CAM 凭证环境变量未回填: %+v", got)
	}
}

func TestBackfillSiteConfig_PreservesExistingCreds(t *testing.T) {
	setupBackfillDB(t)
	model.DB(context.Background()).Create(&model.SiteConfig{
		Name:                  "ClawPro",
		CVMSecretId:           "db-secret-id",
		CVMSecretKey:          "db-secret-key",
		AgentCamRoleSecretId:  "db-agent-id",
		AgentCamRoleSecretKey: "db-agent-key",
	})

	t.Setenv("SECRET_ID", "env-secret-id")
	t.Setenv("SECRET_KEY", "env-secret-key")
	t.Setenv("AGENT_CAM_ROLE_SECRET_ID", "env-agent-id")
	t.Setenv("AGENT_CAM_ROLE_SECRET_KEY", "env-agent-key")

	backfillSiteConfig(CmdlineConfig{Identifier: "tenant-h", Uin: "", Domain: "", InternalSecret: "", Lang: "zh"})

	var got model.SiteConfig
	model.DB(context.Background()).First(&got)
	if got.CVMSecretId != "db-secret-id" || got.CVMSecretKey != "db-secret-key" {
		t.Fatalf("DB 中的 AKSK 不应被环境变量覆盖: %+v", got)
	}
	if got.AgentCamRoleSecretId != "db-agent-id" || got.AgentCamRoleSecretKey != "db-agent-key" {
		t.Fatalf("DB 中的 Agent CAM 凭证不应被环境变量覆盖: %+v", got)
	}
}

func TestBackfillSiteConfig_NoUpdatesNeeded(t *testing.T) {
	setupBackfillDB(t)
	model.DB(context.Background()).Create(&model.SiteConfig{
		Name:   "ClawPro",
		Uin:    "uin-1",
		Domain: "domain-1",
	})

	// 所有字段都已存在，不应该进行 updates
	backfillSiteConfig(CmdlineConfig{Identifier: "tenant-i", Uin: "", Domain: "", InternalSecret: "", Lang: "zh"})

	var got model.SiteConfig
	model.DB(context.Background()).First(&got)
	if got.Uin != "uin-1" || got.Domain != "domain-1" {
		t.Fatalf("存在的值应保持不变: %+v", got)
	}
}

func TestBackfillSiteConfig_ErrorHandling(t *testing.T) {
	setupBackfillDB(t)
	model.DB(context.Background()).Create(&model.SiteConfig{Name: "ClawPro"})

	// 关闭数据库以模拟 Updates 失败
	if err := model.CloseUnderlyingDBForTest(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	// 不应 panic
	backfillSiteConfig(CmdlineConfig{Identifier: "tenant-err", Uin: "uin-err", Domain: "", InternalSecret: "", Lang: "zh"})
}

// ==================== buildFixedSnapshot 单测 ====================

func TestBuildFixedSnapshot_UsesParamsDirectly(t *testing.T) {
	setupBackfillDB(t)
	model.DB(context.Background()).Create(&model.SiteConfig{
		Name:                  "ClawPro",
		Identifier:            "my-tenant",
		Uin:                   "db-uin",
		Domain:                "db-domain",
		InternalSecret:        "db-secret",
		OneIDAccountID:        "db-oneid",
		CVMSecretId:           "db-cvm-id",
		CVMSecretKey:          "db-cvm-key",
		AgentCamRoleSecretId:  "db-agent-id",
		AgentCamRoleSecretKey: "db-agent-key",
	})

	// 传入启动参数，snapshot 应直接使用参数值（非 AKSK 字段）
	buildFixedSnapshot(CmdlineConfig{Identifier: "my-tenant", Uin: "arg-uin", Domain: "arg-domain", InternalSecret: "arg-secret"})

	snap := common.FixedSnapshot
	if snap == nil {
		t.Fatal("FixedSnapshot 未构造")
	}
	if snap.Identifier != "my-tenant" {
		t.Fatalf("identifier 应为参数值: %q", snap.Identifier)
	}
	if snap.Uin != "arg-uin" {
		t.Fatalf("uin 应为参数值: %q", snap.Uin)
	}
	if snap.Domain != "arg-domain" {
		t.Fatalf("domain 应为参数值: %q", snap.Domain)
	}
	if snap.InternalSecret != "arg-secret" {
		t.Fatalf("internal_secret 应为参数值: %q", snap.InternalSecret)
	}
	// AKSK 应从 DB 读取
	if snap.CVMSecretId != "db-cvm-id" || snap.CVMSecretKey != "db-cvm-key" {
		t.Fatalf("AKSK 应从 DB 读取: id=%q key=%q", snap.CVMSecretId, snap.CVMSecretKey)
	}
	if snap.AgentCamRoleSecretId != "db-agent-id" || snap.AgentCamRoleSecretKey != "db-agent-key" {
		t.Fatalf("Agent AKSK 应从 DB 读取: id=%q key=%q", snap.AgentCamRoleSecretId, snap.AgentCamRoleSecretKey)
	}
	// OneIDAccountID 也从 DB 读取
	if snap.OneIDAccountID != "db-oneid" {
		t.Fatalf("OneIDAccountID 应从 DB 读取: %q", snap.OneIDAccountID)
	}
}

func TestBuildFixedSnapshot_FallbackToDB_WhenParamsEmpty(t *testing.T) {
	setupBackfillDB(t)
	model.DB(context.Background()).Create(&model.SiteConfig{
		Name:           "ClawPro",
		Identifier:     "my-tenant",
		Uin:            "db-uin",
		Domain:         "db-domain",
		InternalSecret: "db-secret",
	})

	// 启动参数为空，应回退到 DB 值
	buildFixedSnapshot(CmdlineConfig{Identifier: "my-tenant", Uin: "", Domain: "", InternalSecret: ""})

	snap := common.FixedSnapshot
	if snap == nil {
		t.Fatal("FixedSnapshot 未构造")
	}
	if snap.Uin != "db-uin" {
		t.Fatalf("参数为空时应回退到 DB 值: uin=%q", snap.Uin)
	}
	if snap.Domain != "db-domain" {
		t.Fatalf("参数为空时应回退到 DB 值: domain=%q", snap.Domain)
	}
	if snap.InternalSecret != "db-secret" {
		t.Fatalf("参数为空时应回退到 DB 值: internal_secret=%q", snap.InternalSecret)
	}
}

func TestBuildFixedSnapshot_SQLiteBlankIdentifier(t *testing.T) {
	setupBackfillDB(t)
	model.DB(context.Background()).Create(&model.SiteConfig{Name: "ClawPro", Uin: "db-uin"})

	// SQLite 本地开发：identifier 为空
	buildFixedSnapshot(CmdlineConfig{Identifier: "", Uin: "uin-sqlite", Domain: "", InternalSecret: ""})

	snap := common.FixedSnapshot
	if snap == nil {
		t.Fatal("FixedSnapshot 应该被构造")
	}
	if snap.Identifier != "" {
		t.Fatalf("SQLite 模式 identifier 应为空: %q", snap.Identifier)
	}
	if snap.Uin != "uin-sqlite" {
		t.Fatalf("uin 应为参数值: %q", snap.Uin)
	}
}

// ==================== backfillSiteConfig + buildFixedSnapshot 集成测试 ====================

func TestBackfillAndBuild_FillsEmpty(t *testing.T) {
	setupBackfillDB(t)
	model.DB(context.Background()).Create(&model.SiteConfig{Name: "ClawPro", Identifier: "tenant-a"})

	backfillSiteConfig(CmdlineConfig{Identifier: "tenant-a", Uin: "uin-1", Domain: "a.example.com", InternalSecret: "secret-1", Lang: "zh"})
	buildFixedSnapshot(CmdlineConfig{Identifier: "tenant-a", Uin: "uin-1", Domain: "a.example.com", InternalSecret: "secret-1"})

	var got model.SiteConfig
	model.DB(context.Background()).First(&got)
	if got.Uin != "uin-1" || got.Domain != "a.example.com" || got.InternalSecret != "secret-1" {
		t.Fatalf("backfill 未生效: %+v", got)
	}
	// 但 snapshot 直接使用参数值 2000
	snap := common.FixedSnapshot
	if snap == nil {
		t.Fatal("FixedSnapshot 未构造")
	}
	if snap.Identifier != "tenant-a" || snap.Uin != "uin-1" {
		t.Fatalf("snapshot unexpected: %+v", *snap)
	}
}

func TestBackfillAndBuild_PreservesExisting(t *testing.T) {
	setupBackfillDB(t)
	model.DB(context.Background()).Create(&model.SiteConfig{
		Name:       "ClawPro",
		Identifier: "tenant-b",
		Uin:        "db-uin",
		Domain:     "db-domain",
	})

	// DB 字段已有值不覆盖，InternalSecret 为空会被回填
	backfillSiteConfig(CmdlineConfig{Identifier: "tenant-b", Uin: "", Domain: "", InternalSecret: "arg-secret", Lang: "zh"})
	buildFixedSnapshot(CmdlineConfig{Identifier: "tenant-b", Uin: "", Domain: "", InternalSecret: "arg-secret"})

	var got model.SiteConfig
	model.DB(context.Background()).First(&got)
	if got.Uin != "db-uin" || got.Domain != "db-domain" {
		t.Fatalf("DB 已有值不应被覆盖: %+v", got)
	}
	if got.InternalSecret != "arg-secret" {
		t.Fatalf("InternalSecret 空字段应被回填: %q", got.InternalSecret)
	}
	snap := common.FixedSnapshot
	// 参数为空时回退到 DB 值
	if snap.Uin != "db-uin" {
		t.Fatalf("snapshot 参数为空时应回退到 DB 值: %+v", *snap)
	}
}

func TestBackfillAndBuild_DBHasValue_ParamNotOverride(t *testing.T) {
	setupBackfillDB(t)
	model.DB(context.Background()).Create(&model.SiteConfig{
		Name:   "ClawPro",
		Uin:    "db-uin",
		Domain: "db-domain",
	})

	// DB 已有值，传入非空参数也不会覆盖 DB（回填逻辑：数据库为空才写入）
	// 但 snapshot 直接使用参数值
	backfillSiteConfig(CmdlineConfig{Identifier: "tenant-c", Uin: "cli-uin", Domain: "cli-domain", InternalSecret: "cli-secret", Lang: "zh"})
	buildFixedSnapshot(CmdlineConfig{Identifier: "tenant-c", Uin: "cli-uin", Domain: "cli-domain", InternalSecret: "cli-secret"})

	var got model.SiteConfig
	model.DB(context.Background()).First(&got)
	// DB 值不变
	if got.Uin != "db-uin" {
		t.Fatalf("DB 已有值不应被覆盖: uin=%q", got.Uin)
	}
	if got.Domain != "db-domain" {
		t.Fatalf("DB 已有值不应被覆盖: domain=%q", got.Domain)
	}
	// snapshot 使用参数值
	snap := common.FixedSnapshot
	if snap.Uin != "cli-uin" {
		t.Fatalf("snapshot 应使用参数值: uin=%q", snap.Uin)
	}
	if snap.Domain != "cli-domain" {
		t.Fatalf("snapshot 应使用参数值: domain=%q", snap.Domain)
	}
}

func TestBackfillAndBuild_AllEnvVarsSet(t *testing.T) {
	setupBackfillDB(t)
	model.DB(context.Background()).Create(&model.SiteConfig{Name: "ClawPro", Identifier: "tenant-all"})

	t.Setenv("ONEID_ACCOUNT_ID", "env-oneid")
	t.Setenv("SECRET_ID", "env-id")
	t.Setenv("SECRET_KEY", "env-key")
	t.Setenv("AGENT_CAM_ROLE_SECRET_ID", "env-agent-id")
	t.Setenv("AGENT_CAM_ROLE_SECRET_KEY", "env-agent-key")

	backfillSiteConfig(CmdlineConfig{Identifier: "tenant-all", Uin: "uin-arg", Domain: "domain-arg", InternalSecret: "secret-arg", Lang: "zh"})
	buildFixedSnapshot(CmdlineConfig{Identifier: "tenant-all", Uin: "uin-arg", Domain: "domain-arg", InternalSecret: "secret-arg"})

	var got model.SiteConfig
	model.DB(context.Background()).First(&got)

	// 回填验证
	if got.Uin != "uin-arg" || got.Domain != "domain-arg" || got.InternalSecret != "secret-arg" {
		t.Fatalf("启动参数未被正确回填: %+v", got)
	}
	if got.OneIDAccountID != "env-oneid" {
		t.Fatalf("OneID 环境变量未回填: %+v", got)
	}
	if got.CVMSecretId != "env-id" || got.CVMSecretKey != "env-key" {
		t.Fatalf("AKSK 环境变量未回填: %+v", got)
	}
	if got.AgentCamRoleSecretId != "env-agent-id" || got.AgentCamRoleSecretKey != "env-agent-key" {
		t.Fatalf("Agent CAM 凭证环境变量未回填: %+v", got)
	}

	// snapshot 验证
	snap := common.FixedSnapshot
	if snap.Uin != "uin-arg" || snap.Domain != "domain-arg" || snap.InternalSecret != "secret-arg" {
		t.Fatalf("snapshot 应使用参数值: %+v", *snap)
	}
	if snap.CVMSecretId != "env-id" || snap.CVMSecretKey != "env-key" {
		t.Fatalf("snapshot AKSK 应从 DB 读取: %+v", *snap)
	}
}

func TestBackfillAndBuild_ErrorHandling(t *testing.T) {
	setupBackfillDB(t)
	model.DB(context.Background()).Create(&model.SiteConfig{Name: "ClawPro"})

	// 关闭数据库以模拟失败
	if err := model.CloseUnderlyingDBForTest(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	// 不应 panic
	backfillSiteConfig(CmdlineConfig{Identifier: "tenant-error-test", Uin: "uin-err", Domain: "", InternalSecret: "", Lang: "zh"})
	buildFixedSnapshot(CmdlineConfig{Identifier: "tenant-error-test", Uin: "uin-err", Domain: "", InternalSecret: ""})

	// FixedSnapshot 应该仍然被构造
	snap := common.FixedSnapshot
	if snap == nil {
		t.Fatal("FixedSnapshot 应该被构造，即使 DB 操作失败")
	}
}

// TestBackfillSiteConfig_InternalSecretFromEnv 覆盖参数为空 + 环境变量兜底的分支。
func TestBackfillSiteConfig_InternalSecretFromEnv(t *testing.T) {
	setupBackfillDB(t)
	model.DB(context.Background()).Create(&model.SiteConfig{Name: "ClawPro", Identifier: "tenant-env-secret"})

	t.Setenv("INTERNAL_SECRET", "env-internal-secret")
	// 参数为空，应从环境变量兜底
	backfillSiteConfig(CmdlineConfig{Identifier: "tenant-env-secret", Uin: "", Domain: "", InternalSecret: "", Lang: "zh"})

	var got model.SiteConfig
	model.DB(context.Background()).First(&got)
	if got.InternalSecret != "env-internal-secret" {
		t.Fatalf("INTERNAL_SECRET 环境变量未兜底回填: %q", got.InternalSecret)
	}
}

// TestBackfillSiteConfig_InternalSecretParamOverridesEnv 覆盖参数优先于环境变量的分支。
func TestBackfillSiteConfig_InternalSecretParamOverridesEnv(t *testing.T) {
	setupBackfillDB(t)
	model.DB(context.Background()).Create(&model.SiteConfig{Name: "ClawPro", Identifier: "tenant-param-secret"})

	t.Setenv("INTERNAL_SECRET", "env-should-not-use")
	// 参数非空，应优先使用参数
	backfillSiteConfig(CmdlineConfig{Identifier: "tenant-param-secret", Uin: "", Domain: "", InternalSecret: "param-secret", Lang: "zh"})

	var got model.SiteConfig
	model.DB(context.Background()).First(&got)
	if got.InternalSecret != "param-secret" {
		t.Fatalf("参数应优先于环境变量: got=%q", got.InternalSecret)
	}
}

// TestBackfillSiteConfig_InternalSecretDBNotEmpty_SkipBackfill 覆盖 DB 已有值不覆盖的分支。
func TestBackfillSiteConfig_InternalSecretDBNotEmpty_SkipBackfill(t *testing.T) {
	setupBackfillDB(t)
	model.DB(context.Background()).Create(&model.SiteConfig{
		Name:           "ClawPro",
		Identifier:     "tenant-db-secret",
		InternalSecret: "db-existing-secret",
	})

	t.Setenv("INTERNAL_SECRET", "env-should-not-override")
	backfillSiteConfig(CmdlineConfig{Identifier: "tenant-db-secret", Uin: "", Domain: "", InternalSecret: "param-should-not-override", Lang: "zh"})

	var got model.SiteConfig
	model.DB(context.Background()).First(&got)
	if got.InternalSecret != "db-existing-secret" {
		t.Fatalf("DB 已有值不应被覆盖: got=%q", got.InternalSecret)
	}
}

// 兜底防止 os.Setenv 在 goroutine 并发时串扰：明确环境变量恢复
func init() {
	os.Unsetenv("ONEID_ACCOUNT_ID")
	os.Unsetenv("SECRET_ID")
	os.Unsetenv("SECRET_KEY")
	os.Unsetenv("AGENT_CAM_ROLE_SECRET_ID")
	os.Unsetenv("AGENT_CAM_ROLE_SECRET_KEY")
}
