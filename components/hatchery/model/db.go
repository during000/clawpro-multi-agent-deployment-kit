package model

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"hatchery/common"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// gdb 是进程级 GORM 句柄，仅 model 包内部使用（"g" 意为 global，避免与函数参数
// 惯用名 db 冲突）。对外必须通过 DB(ctx) / DBGlobal(ctx) 函数入口访问，以保证
// GORM 回调能从 ctx 中读取 TenantSnapshot 完成租户隔离。
var gdb *gorm.DB
var savedDBPath string
var dbDriver string // "sqlite" or "mysql"

// DB 返回一个绑定到指定 ctx 的 *gorm.DB 句柄。
// 所有包外调用方必须通过 DB(ctx) 访问数据库，使 GORM 回调能从 ctx 中读取
// TenantSnapshot 完成租户隔离。桥接期内若 ctx 无 snapshot，回调回退到
// currentIdentifier 保证行为等价（参见 design.md §"渐进桥接方案"）。
func DB(ctx context.Context) *gorm.DB {
	if gdb == nil || ctx == nil {
		return gdb
	}
	return gdb.WithContext(ctx)
}

// SaveSelectedFields 只更新 GORM model 上显式传入的 Go 结构体字段。
//
// fields 必须是 Go 结构体字段名（例如 "Name"、"SkillHub"、"CVMTemplate"），
// 不能写数据库列名。函数会先用 GORM schema 校验字段名，再生成 SQL；
// 这样字段拼错会在运行时明确报错，而不是被 GORM 静默忽略或退化成数据库侧 unknown column。
func SaveSelectedFields(ctx context.Context, value interface{}, fields ...string) error {
	if len(fields) == 0 {
		return nil
	}
	if value == nil {
		return fmt.Errorf("nil model")
	}

	db := DB(ctx)
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(value); err != nil {
		return err
	}
	for _, field := range fields {
		if stmt.Schema.LookUpField(field) == nil {
			return fmt.Errorf("unknown %s field %q", stmt.Schema.Name, field)
		}
	}

	return db.Model(value).Select(fields).Updates(value).Error
}

// DBGlobal 返回一个绑定到 ctx 且跳过 identifier 过滤的 *gorm.DB 句柄。
// 用于访问没有 Identifier 字段的全局表(如未来的 tenant_domains、分布式锁辅助表等)。
// 注意：即使目标 model 没有 Identifier 字段，回调也能安全跳过，但显式使用 DBGlobal
// 让意图在调用点处清晰可见。
func DBGlobal(ctx context.Context) *gorm.DB {
	if gdb == nil {
		return gdb
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return gdb.WithContext(common.WithSkipIdentifier(ctx))
}

// ================================ 测试相关函数 ================================
var testDBLock sync.Mutex

func UseDBForTestWithDriver(newDB *gorm.DB, driver string) func() {
	oldDriver := dbDriver
	dbDriver = driver
	restore := UseDBForTest(newDB)
	return func() {
		dbDriver = oldDriver
		restore()
	}
}

// UseDBForTest 在单测中替换进程级 GORM 句柄，返回 restore 函数用于 defer 恢复。
// 非测试代码不得调用。
//
// db := openTestSQLite(t)
// t.Cleanup(model.UseDBForTest(db))
func UseDBForTest(newDB *gorm.DB) func() {
	testDBLock.Lock()
	old := gdb
	gdb = newDB
	return func() {
		if gdb != newDB {
			// FIXME: 防止双重释放导致 panic
			// 但是双重释放本就不应该出现
			// 需要在测试用例编写侧面杜绝
			return
		}
		gdb = old
		testDBLock.Unlock()
	}
}

func SetDBForTest(newDB *gorm.DB) {
	testDBLock.Lock()
	defer testDBLock.Unlock()
	gdb = newDB
}

// AllModelsForTest 返回所有 GORM model 列表，供外部测试包 AutoMigrate 使用。
func AllModelsForTest() []any {
	return allModels
}

// UseNilDBForTest 将进程级 GORM 句柄置为 nil，返回 restore 函数用于 defer 恢复。
// 用于测试 gdb == nil 的防御分支。非测试代码不得调用。
//
//	t.Cleanup(model.UseNilDBForTest())
func UseNilDBForTest() func() {
	return UseDBForTest(nil)
}

// CloseUnderlyingDBForTest 关闭底层 sql.DB 连接，仅用于单测模拟 DB 不可用场景。
// 非测试代码不得调用。
func CloseUnderlyingDBForTest() error {
	if gdb == nil {
		return nil
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// =================================================================

// resolveIdentifier 从 ctx 中解析当前有效 identifier。
// InjectTenant 与 WithSkipIdentifier 互斥（注入时互相清除），因此不会同时存在。
//   - ctx 有 TenantSnapshot → 返回 snapshot.Identifier
//   - ctx 有 WithSkipIdentifier 标记 → 返回空(跳过过滤)
//   - ctx 两者都无 → panic（调用方未正确注入）
func resolveIdentifier(ctx context.Context) string {
	if ctx == nil {
		ctx = context.Background()
	}
	if snap, ok := common.GetTenantSnapshot(ctx); ok {
		return snap.Identifier
	}
	if common.ShouldSkipIdentifier(ctx) {
		return ""
	}
	// ctx 无 snapshot：调用方未正确注入 TenantSnapshot，直接 panic 防止数据泄露
	var caller string
	for skip := 2; skip < 10; skip++ {
		_, file, line, ok := runtime.Caller(skip)
		if !ok {
			break
		}
		if strings.Contains(file, "gorm.io") || strings.Contains(file, "model/db.go") {
			continue
		}
		caller = fmt.Sprintf("%s:%d", file, line)
		break
	}
	panic(fmt.Sprintf("model: DB operation without TenantSnapshot in context, caller: %s", caller))
}

// CurrentIdentifier 返回当前实例的多租户标识。
// SQLite 单租户模式下返回空串。
func CurrentIdentifier(ctx context.Context) string {
	if snap, ok := common.GetTenantSnapshot(ctx); ok {
		return snap.Identifier
	}
	return ""
}

// allModels 是所有 GORM model 的完整列表，AutoMigrate 和迁移校验共用。
// 新增 model 时必须加到此列表中，否则 SQLite AutoMigrate 和 MySQL 迁移都不会覆盖。
var allModels = []any{
	&User{},
	&PasswordlessLoginToken{},
	&SiteConfig{},
	&Instance{},
	&InstanceAdjustment{},
	&AgentProxyRoute{},
	&AIModel{},
	&AIChannel{},
	&LLMUsageLog{},
	&DailyUsageSummary{},
	&AIImage{},
	&ImageHistory{},
	&CustomAgentType{},
	&AuditLog{},
	&SessionBlacklist{},
	&OneIDUserProfile{},
	&OneIDDepartmentRecord{},
	&SkillCategory{},
	&SkillCategoryMapping{},
	&Skill{},
	&SkillDistributionTask{},
	&SkillDistributionRecord{},
	&RoleDistributionRecord{},
	&SMHSpace{},
	&SMHPersonalSpace{},
	&Notification{},
	&SkillBundle{},
	&BundleSkill{},
	&PublicSkill{},
	&PublicSkillSet{},
	&SkillInstallation{},
	&MemoryTDAIPlugin{},
	&MemoryPlanGroupPolicy{},
	&TdaiJob{},
	&OpenClawRole{},
	&OpenClawRoleSkill{},
	&Plugin{},
	&PluginDistributionTask{},
	&PluginDistributionRecord{},
	&PluginCategory{},
	&PluginCategoryMapping{},
	&PluginBundle{},
	&BundlePlugin{},
	&PublicPlugin{},
	&PluginInstallation{},
	&OpenClawRolePlugin{},
	&UserGroup{},
	&ResourcePolicy{},
	&UserGroupMember{},
	&GroupClosure{},       // 🆕 v6.12 P1: 分组闭包表
	&GroupConfigBinding{}, // 🆕 分组配置绑定统一表
	&VpcConfig{},          // 🆕 VPC 配置表
	&Tag{},
	&TagVisibilityGroup{},
	&ModelVisibilityGroup{},
	&SkillVisibilityGroup{},
	&PluginVisibilityGroup{},
	&SkillBundleVisibilityGroup{},
	&RoleVisibilityGroup{},
	&DoctorSession{},
	&DoctorAuthorization{},
	// MCP 企业库（6 张表）
	&McpServer{},
	&McpHostedKey{},
	&McpVersion{},
	&McpDistributionTask{},
	&McpDistributionRecord{},
	&McpInstallation{},
	// 安全组规则组 + 池（sg-ruleset-projection 方案）
	&RuleSet{},
	&ManagedSGPool{},
	&SGDrainState{},
	&InstanceModel{},
	&AgentMigration{},
	// Skill 安全检测
	&SkillSecurityScan{},
	&SkillScanViolation{},
	// Agent 命令执行
	&AgentCommand{},
	&AgentCommandDispatch{},
	&AgentCommandInvocation{},
	&AgentCommandTask{},
	&AgentCommandSchedule{},
	&AgentCommandScheduleRecord{},
	// 多租户阶段二：域名→租户映射（全局表）
	&TenantDomain{},
	// 功能白名单（全局表，跨租户，按 type 分样；首期用于 local-agent）
	&FeatureAllowlist{},
	// 本地 agent（clawpro 一期）
	&LocalInstanceInfo{},
	&LocalInstanceSkill{},
	&LocalAgentCLSCredential{},
	&LocalAgentTask{}, // 通用本地实例任务表（本地 agent 三期）
	// 企业规范库（本地 agent 二期）
	&EnterpriseRule{},
	&RuleDistributionTask{},
	&RuleDistributionRecord{},
	&RuleVisibilityGroup{},
	&LocalInstanceRule{},
	// 项目资产管理
	&Project{},
	&ProjectMember{},
	&ProjectConfigBinding{},
	&LocalAgentScopeBinding{},
	&AssetVersionRecord{}, // 资产版本记录（资产管理版本记录子任务）
	// 存量实例分组归属处理（stale-instances v1.0）
	&InstanceFlag{},
	&InstanceChangeGroupRecord{},
	// 技能共建审核（通用审批表）
	&ReviewRequest{},
}

// InitDB opens the database, runs migrations, seeds defaults, and
// creates the initial admin user when the database is empty.
// dbType: "sqlite" (default) or "mysql".
// identifier: optional, used for multi-tenant isolation (auto-injects WHERE identifier=? on queries).
// universe: true for universe multi-tenant mode (skips seed/admin, each tenant inits via /tenants/init API).
// dbMigrate: optional, path to source SQLite DB for one-time migration.
func InitDB(dbPath, dbType, identifier, initUser, initPass, dbMigrate string, universe bool) {
	savedDBPath = dbPath
	dbDriver = dbType
	var dialector gorm.Dialector

	switch dbType {
	case "mysql":
		dialector = mysql.New(mysql.Config{
			DSN:               dbPath,
			DefaultStringSize: 191, // MySQL 下无显式 size 的 string 默认为 varchar(191)，避免 longtext 无法做索引
		})
	default: // "sqlite"
		dbDriver = "sqlite"
		dsn := dbPath + "?" +
			"_txlock=immediate" +
			"&_pragma=journal_mode(WAL)" +
			"&_pragma=busy_timeout(5000)" +
			"&_pragma=synchronous(NORMAL)" +
			"&_pragma=cache_size(-20000)" +
			"&_pragma=foreign_keys(ON)" +
			"&_pragma=temp_store(MEMORY)" +
			"&_pragma=mmap_size(268435456)"
		dialector = sqlite.Open(dsn)
	}

	var err error
	// 不使用 GORM 内置 SlogLogger，避免与 RegisterDBLogger 的自定义回调产双重日志。
	// 所有 DB 操作日志（含慢查询）统一由 controller.RegisterDBLogger 的 Before/After 回调记录。
	gdb, err = gorm.Open(dialector, &gorm.Config{
		Logger: nil,
	})
	if err != nil {
		slog.Error("failed to connect database", "error", err)
		os.Exit(1)
	}

	// MySQL connection pool configuration
	if dbType == "mysql" {
		configureConnectionPool(gdb, universe)
	}

	// 注册多租户隔离回调（自动为所有含 Identifier 字段的模型注入 WHERE identifier = ?）
	// universe 模式同样需要：identifier 从请求 ctx 中动态获取。
	if identifier != "" || universe {
		registerIdentifierCallbacks(gdb)
	}

	// 启动期 ctx：MySQL 模式下需要注入 identifier，以便 GORM 回调（distlock、seed 等）正常工作。
	// FixedSnapshot 此时尚未构造，手动注入最小 TenantSnapshot。
	initCtx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: identifier})

	// SQLite 模式：AutoMigrate 自动建表/加字段。
	// MySQL 模式：跳过 AutoMigrate，表结构由外部 SQL 脚本管理（sql/init.sql）。
	//             删字段、改类型等复杂变更，生产环境应通过 DBA 执行 DDL 变更。
	if dbDriver == "sqlite" {
		if err := gdb.WithContext(initCtx).AutoMigrate(allModels...); err != nil {
			slog.Error("auto migrate failed", "error", err)
			os.Exit(1)
		}
	}

	// Universe 模式：不执行 seed 和 admin 创建。
	// 每个租户通过 /tenants/init API 独立初始化。
	if universe {
		slog.Info("[InitDB] Universe 模式，跳过 seed 和初始管理员创建")
		return
	}

	// ===== 以下仅在单租户模式下执行 =====

	// MySQL 多实例部署时，用分布式锁保证 seed 阶段串行执行，避免 check-then-create 竞态导致重复数据。
	// SQLite 模式下 AcquireLock 为空操作，不影响单实例部署。
	lock, err := AcquireLock(initCtx, "db:seed", 30*time.Second)
	if err != nil {
		slog.Error("[InitDB] 获取 seed 锁失败", "error", err)
		os.Exit(1)
	}
	defer lock.Release()

	// --db-migrate 模式：从 SQLite 导入初始数据
	if dbMigrate != "" {
		MigrateFromSQLite(initCtx, dbMigrate, identifier)
	}

	// Seed default site config
	var config SiteConfig
	initDB := gdb.WithContext(initCtx)
	if initDB.First(&config).Error != nil {
		config = SiteConfig{}
		ApplySiteConfigDefaults(&config)
		initDB.Create(&config)
	}
	// 已有配置的 MemoryTDAISupportedVersions 同步由 startup task 统一处理（SyncMemoryTDAISupportedVersions）

	var count int64
	initDB.Model(&User{}).Count(&count)
	if count == 0 {
		if initUser == "" {
			// OneID 模式：无本地用户时跳过初始管理员创建，用户通过 SSO 登录后自动建立
			slog.Info("No users found and --init-user not set; skipping initial admin creation (OneID SSO mode)")
		} else {
			if initPass == "" {
				slog.Error("--init-pass is required when --init-user is set")
				os.Exit(1)
			}
			hash, _ := bcrypt.GenerateFromPassword([]byte(initPass), bcrypt.DefaultCost)
			admin := User{Username: initUser, Password: string(hash), Role: "admin"}
			initDB.Create(&admin)
			slog.Info("Created initial admin user", "username", initUser)
		}
	}
}

// configureConnectionPool 设置 MySQL 连接池参数。
// universe 模式下多租户 + task scheduler 并发高，需要更大连接池。
func configureConnectionPool(db *gorm.DB, universe bool) {
	sqlDB, err := db.DB()
	if err != nil {
		panic(fmt.Sprintf("failed to get underlying sql.DB for pool configuration: %v", err))
	}
	if universe {
		sqlDB.SetMaxIdleConns(50)
		sqlDB.SetMaxOpenConns(500)
	} else {
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)
	}
	sqlDB.SetConnMaxLifetime(time.Hour)
}

// registerIdentifierCallbacks 注册 GORM 全局回调，自动为所有含 Identifier 字段的模型注入多租户隔离条件。
// Create 时自动填充 identifier，Query/Update/Delete/RowQuery 时自动追加 WHERE identifier = ?。
// 阶段一桥接期：优先从 tx.Statement.Context 中的 TenantSnapshot 读取 identifier，读不到时回退到 currentIdentifier。
func registerIdentifierCallbacks(db *gorm.DB) {
	// Create 时自动填充 identifier
	db.Callback().Create().Before("gorm:create").Register("set_identifier", func(tx *gorm.DB) {
		id := resolveIdentifier(tx.Statement.Context)
		if id == "" {
			return
		}
		if tx.Statement.Schema != nil {
			if field := tx.Statement.Schema.LookUpField("Identifier"); field != nil {
				tx.Statement.SetColumn("Identifier", id, true)
			}
		}
	})

	// Query 时自动加 WHERE identifier = ?
	db.Callback().Query().Before("gorm:query").Register("filter_identifier_query", applyIdentifierFilter)
	db.Callback().Update().Before("gorm:update").Register("filter_identifier_update", applyIdentifierFilter)
	db.Callback().Delete().Before("gorm:delete").Register("filter_identifier_delete", applyIdentifierFilter)
	db.Callback().Row().Before("gorm:row").Register("filter_identifier_row", applyIdentifierFilter)
}

// applyIdentifierFilter 为当前 GORM 语句追加 WHERE identifier = ? 条件（仅当模型含 Identifier 字段时）。
func applyIdentifierFilter(tx *gorm.DB) {
	id := resolveIdentifier(tx.Statement.Context)
	if id == "" {
		return
	}
	if tx.Statement.Schema != nil {
		if field := tx.Statement.Schema.LookUpField("Identifier"); field != nil {
			tx.Statement.AddClause(clause.Where{
				Exprs: []clause.Expression{
					clause.Eq{
						Column: clause.Column{Table: tx.Statement.Schema.Table, Name: "identifier"},
						Value:  id,
					},
				},
			})
		}
	}
}

// CloseDB checkpoints WAL (SQLite only) to flush all data back to the main database file,
// then closes connections and removes leftover -wal/-shm files.
func CloseDB() {
	if gdb == nil {
		return
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		slog.Error("failed to get underlying sql.DB", "error", err)
		return
	}

	if dbDriver == "sqlite" {
		// wal_checkpoint(TRUNCATE) returns: busy, walLog, checkpointed.
		// If walLog == checkpointed (and both >= 0), all WAL pages are flushed;
		// it is then safe to remove the -wal and -shm files after closing.
		var busy, walLog, checkpointed int
		canRemove := false
		row := sqlDB.QueryRow("PRAGMA wal_checkpoint(TRUNCATE)")
		if err := row.Scan(&busy, &walLog, &checkpointed); err != nil {
			slog.Error("WAL checkpoint failed", "error", err)
		} else {
			slog.Info("WAL checkpoint completed", "busy", busy, "log", walLog, "checkpointed", checkpointed)
			if busy == 0 && walLog >= 0 && walLog == checkpointed {
				canRemove = true
			}
		}

		// Close all database connections.
		if err := sqlDB.Close(); err != nil {
			slog.Error("failed to close database", "error", err)
		} else {
			slog.Info("Database connections closed")
		}

		// After a successful TRUNCATE checkpoint all WAL content is in the main gdb.
		// The -wal file is truncated to 0 bytes and -shm is a stale wal-index.
		// SQLite normally removes them when the last connection closes, but the
		// pure-Go driver (modernc) does not always do so. Clean up manually.
		if canRemove {
			os.Remove(savedDBPath + "-wal")
			os.Remove(savedDBPath + "-shm")
			slog.Info("WAL/SHM files removed")
		}
	} else {
		// MySQL: just close connections
		if err := sqlDB.Close(); err != nil {
			slog.Error("failed to close database", "error", err)
		} else {
			slog.Info("Database connections closed")
		}
	}
}
