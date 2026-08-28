package controller

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// initSpaceTestDB 初始化内存 SQLite 数据库，迁移个人空间相关的表。
func initSpaceTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.User{},
		&model.Instance{},
		&model.SMHPersonalSpace{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
}

// createTestInstanceAndSpace 创建测试用的用户、实例和个人空间（在回收站中）。
func createTestInstanceAndSpace(t *testing.T) (model.Instance, model.SMHPersonalSpace) {
	t.Helper()
	user := model.User{Username: "testuser", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)

	ins := model.Instance{Name: "test-box", InstanceId: "ins-test-001", UserID: user.ID}
	model.DB(context.Background()).Create(&ins)

	deleteAt := time.Now().Add(7 * 24 * time.Hour)
	space := model.SMHPersonalSpace{
		SpaceId:       "sp-test",
		UserId:        user.ID,
		InstanceId:    ins.ID,
		UserName:      user.Username,
		InstanceName:  ins.Name,
		CVMInstanceId: ins.InstanceId,
		ToBeDeletedAt: &deleteAt,
	}
	model.DB(context.Background()).Create(&space)
	return ins, space
}

// ---------- RestorePersonalSpace ----------

func TestRestorePersonalSpace_Success(t *testing.T) {
	initSpaceTestDB(t)
	_, space := createTestInstanceAndSpace(t)

	changed, err := RestorePersonalSpace(context.Background(), &space)
	if err != nil {
		t.Fatalf("RestorePersonalSpace 返回错误: %v", err)
	}
	if !changed {
		t.Fatal("实例存在时期望恢复成功（changed=true）")
	}

	// 验证数据库状态
	var updated model.SMHPersonalSpace
	model.DB(context.Background()).First(&updated, space.ID)
	if updated.ToBeDeletedAt != nil {
		t.Error("恢复后 to_be_deleted_at 应为 nil")
	}
	if updated.EnvInitialized {
		t.Error("恢复后 env_initialized 应为 false")
	}
}

func TestRestorePersonalSpace_KeepsProvisionRev(t *testing.T) {
	initSpaceTestDB(t)
	_, space := createTestInstanceAndSpace(t)
	if err := model.DB(context.Background()).Model(&space).Updates(map[string]interface{}{
		"env_initialized":   true,
		"env_provision_rev": CurrentSMHProvisionRev,
	}).Error; err != nil {
		t.Fatalf("预置个人空间环境状态失败: %v", err)
	}

	changed, err := RestorePersonalSpace(context.Background(), &space)
	if err != nil {
		t.Fatalf("RestorePersonalSpace 返回错误: %v", err)
	}
	if !changed {
		t.Fatal("实例存在时期望恢复成功（changed=true）")
	}

	var updated model.SMHPersonalSpace
	if err := model.DB(context.Background()).First(&updated, space.ID).Error; err != nil {
		t.Fatalf("查询个人空间失败: %v", err)
	}
	if updated.EnvProvisionRev != CurrentSMHProvisionRev {
		t.Errorf("恢复空间不应修改 env_provision_rev，期望 %d，实际 %d", CurrentSMHProvisionRev, updated.EnvProvisionRev)
	}
	if updated.EnvInitialized {
		t.Error("恢复空间应将 env_initialized 置为 false")
	}
}

func TestRestorePersonalSpace_InstanceDeleted(t *testing.T) {
	initSpaceTestDB(t)
	ins, space := createTestInstanceAndSpace(t)

	// 软删除实例
	model.DB(context.Background()).Delete(&ins)

	changed, err := RestorePersonalSpace(context.Background(), &space)
	if err != nil {
		t.Fatalf("RestorePersonalSpace 返回错误: %v", err)
	}
	if changed {
		t.Fatal("实例已删除时期望恢复被拒绝（changed=false）")
	}

	// 验证空间仍在回收站
	var unchanged model.SMHPersonalSpace
	model.DB(context.Background()).Unscoped().First(&unchanged, space.ID)
	if unchanged.ToBeDeletedAt == nil {
		t.Error("实例已删除时空间不应被恢复，to_be_deleted_at 应非空")
	}
}

func TestRestorePersonalSpace_AlreadyActive(t *testing.T) {
	initSpaceTestDB(t)

	user := model.User{Username: "active-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)
	ins := model.Instance{Name: "active-box", InstanceId: "ins-active", UserID: user.ID}
	model.DB(context.Background()).Create(&ins)
	// 创建活跃空间（to_be_deleted_at 为空）
	space := model.SMHPersonalSpace{
		SpaceId:       "sp-active",
		UserId:        user.ID,
		InstanceId:    ins.ID,
		UserName:      user.Username,
		InstanceName:  ins.Name,
		CVMInstanceId: ins.InstanceId,
	}
	model.DB(context.Background()).Create(&space)

	changed, err := RestorePersonalSpace(context.Background(), &space)
	if err != nil {
		t.Fatalf("RestorePersonalSpace 返回错误: %v", err)
	}
	if changed {
		t.Fatal("已活跃的空间期望幂等返回（changed=false）")
	}
}

// ---------- RecyclePersonalSpace ----------

func TestRecyclePersonalSpace_Success(t *testing.T) {
	initSpaceTestDB(t)

	user := model.User{Username: "recycle-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)
	ins := model.Instance{Name: "recycle-box", InstanceId: "ins-recycle", UserID: user.ID}
	model.DB(context.Background()).Create(&ins)
	space := model.SMHPersonalSpace{
		SpaceId:       "sp-recycle",
		UserId:        user.ID,
		InstanceId:    ins.ID,
		UserName:      user.Username,
		InstanceName:  ins.Name,
		CVMInstanceId: ins.InstanceId,
	}
	model.DB(context.Background()).Create(&space)

	changed, err := RecyclePersonalSpace(context.Background(), &space)
	if err != nil {
		t.Fatalf("RecyclePersonalSpace 返回错误: %v", err)
	}
	if !changed {
		t.Fatal("活跃空间期望回收成功（changed=true）")
	}

	var updated model.SMHPersonalSpace
	model.DB(context.Background()).First(&updated, space.ID)
	if updated.ToBeDeletedAt == nil {
		t.Error("回收后 to_be_deleted_at 应非空")
	}
}

func TestRecyclePersonalSpace_AlreadyRecycled(t *testing.T) {
	initSpaceTestDB(t)
	_, space := createTestInstanceAndSpace(t) // 已在回收站

	changed, err := RecyclePersonalSpace(context.Background(), &space)
	if err != nil {
		t.Fatalf("RecyclePersonalSpace 返回错误: %v", err)
	}
	if changed {
		t.Fatal("已在回收站的空间期望幂等返回（changed=false）")
	}
}

// ---------- MarkPersonalSpaceToBeDeleted ----------

func TestMarkPersonalSpaceToBeDeleted_ActiveSpace(t *testing.T) {
	initSpaceTestDB(t)

	user := model.User{Username: "mark-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)
	ins := model.Instance{Name: "mark-box", InstanceId: "ins-mark", UserID: user.ID}
	model.DB(context.Background()).Create(&ins)
	space := model.SMHPersonalSpace{
		SpaceId:       "sp-mark",
		UserId:        user.ID,
		InstanceId:    ins.ID,
		UserName:      user.Username,
		InstanceName:  ins.Name,
		CVMInstanceId: ins.InstanceId,
	}
	model.DB(context.Background()).Create(&space)

	MarkPersonalSpaceToBeDeleted(context.Background(), ins.ID)

	var updated model.SMHPersonalSpace
	model.DB(context.Background()).First(&updated, space.ID)
	if updated.ToBeDeletedAt == nil {
		t.Error("Mark 后 to_be_deleted_at 应非空")
	}
}

func TestMarkPersonalSpaceToBeDeleted_AfterInstanceDeleted(t *testing.T) {
	initSpaceTestDB(t)

	user := model.User{Username: "postdel-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)
	ins := model.Instance{Name: "postdel-box", InstanceId: "ins-postdel", UserID: user.ID}
	model.DB(context.Background()).Create(&ins)
	space := model.SMHPersonalSpace{
		SpaceId:       "sp-postdel",
		UserId:        user.ID,
		InstanceId:    ins.ID,
		UserName:      user.Username,
		InstanceName:  ins.Name,
		CVMInstanceId: ins.InstanceId,
	}
	model.DB(context.Background()).Create(&space)

	// 先删实例，再 Mark（模拟先 Delete 后 Mark 的路径）
	model.DB(context.Background()).Delete(&ins)
	MarkPersonalSpaceToBeDeleted(context.Background(), ins.ID)

	var updated model.SMHPersonalSpace
	model.DB(context.Background()).First(&updated, space.ID)
	if updated.ToBeDeletedAt == nil {
		t.Error("实例已删除后 Mark 仍应成功标记空间")
	}
}

// ---------- 竞争场景模拟 ----------

func TestRace_RestoreThenMarkAfterDelete(t *testing.T) {
	// 模拟：空间在回收站 → Restore 先于 Delete 恢复成功 → Delete 后 Mark 兜底
	initSpaceTestDB(t)
	ins, space := createTestInstanceAndSpace(t)

	// 1. Restore 成功（实例仍存在）
	changed, err := RestorePersonalSpace(context.Background(), &space)
	if err != nil {
		t.Fatalf("Restore 返回错误: %v", err)
	}
	if !changed {
		t.Fatal("Restore 期望成功")
	}

	// 2. 删除实例
	model.DB(context.Background()).Delete(&ins)

	// 3. Delete 后 Mark 兜底
	MarkPersonalSpaceToBeDeleted(context.Background(), ins.ID)

	// 验证空间被重新标记回收
	var final model.SMHPersonalSpace
	model.DB(context.Background()).First(&final, space.ID)
	if final.ToBeDeletedAt == nil {
		t.Error("Delete 后 Mark 应将恢复的空间重新标记为待删除")
	}
}

func TestRace_RestoreAfterDelete_Blocked(t *testing.T) {
	// 模拟：实例先被删除 → Restore 因子查询检测到实例不存在而被拒绝
	initSpaceTestDB(t)
	ins, space := createTestInstanceAndSpace(t)

	// 1. 删除实例
	model.DB(context.Background()).Delete(&ins)

	// 2. 尝试 Restore
	changed, err := RestorePersonalSpace(context.Background(), &space)
	if err != nil {
		t.Fatalf("Restore 返回错误: %v", err)
	}
	if changed {
		t.Fatal("实例已删除后 Restore 应被拒绝（changed=false）")
	}

	// 验证空间仍在回收站
	var final model.SMHPersonalSpace
	model.DB(context.Background()).Unscoped().First(&final, space.ID)
	if final.ToBeDeletedAt == nil {
		t.Error("空间应仍在回收站")
	}
}

// ---------- refreshPersonalSpaceToken ----------

// initSMHRefreshTestDB 初始化测试所需的全部表（User/Instance/SMHPersonalSpace/SiteConfig/SMHSpace），
// 并返回 cleanup：清空 personalSpaceTokenCache 与 SMH 状态，避免测试间串扰。
func initSMHRefreshTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.User{},
		&model.Instance{},
		&model.SMHPersonalSpace{},
		&model.SiteConfig{},
		&model.SMHSpace{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	origDB := model.UseDBForTest(db)
	// 清理 token 缓存（sync.Map 无 clear 方法，用 Range + Delete）
	personalSpaceTokenCache.Range(func(k, _ interface{}) bool {
		personalSpaceTokenCache.Delete(k)
		return true
	})

	return func() {
		personalSpaceTokenCache.Range(func(k, _ interface{}) bool {
			personalSpaceTokenCache.Delete(k)
			return true
		})
		origDB()
	}
}

// seedSMHConfigured 写入完整的 SMH 配置，使 model.GetSMHConfig().IsConfigured() 返回 true。
func seedSMHConfigured(t *testing.T) {
	t.Helper()
	if err := model.DB(context.Background()).Create(&model.SiteConfig{
		SMHEnabled:       1,
		SMHEndpoint:      "https://smh.example.com",
		SMHLibraryId:     "lib-test",
		SMHLibrarySecret: "secret-test",
	}).Error; err != nil {
		t.Fatalf("写入 SiteConfig 失败: %v", err)
	}
	if err := model.DB(context.Background()).Create(&model.SMHSpace{SpaceTag: "common", SpaceId: "sp-common", LibraryId: "lib-test", Purpose: "common"}).Error; err != nil {
		t.Fatalf("写入 common Space 失败: %v", err)
	}
	if err := model.DB(context.Background()).Create(&model.SMHSpace{SpaceTag: "skillhub", SpaceId: "sp-skillhub", LibraryId: "lib-test", Purpose: "skillhub"}).Error; err != nil {
		t.Fatalf("写入 skillhub Space 失败: %v", err)
	}
}

// primeTokenCache 预先往缓存中塞入一个"未到刷新时间"的 token，
// 让 ensurePersonalSpaceToken 命中缓存，跳过对真实 SMH API 的调用。
func primeTokenCache(spaceId, token string) {
	val, _ := personalSpaceTokenCache.LoadOrStore(spaceId, &cachedSpaceToken{})
	c := val.(*cachedSpaceToken)
	c.mu.Lock()
	c.accessToken = token
	c.expiresAt = time.Now().Add(personalSpaceTokenTTL) // 剩余 24h > 18h，直接命中缓存
	c.mu.Unlock()
}

// createSpaceAndInstance 创建一条 Instance + SMHPersonalSpace，便于测试复用。
func createSpaceAndInstance(t *testing.T, agentType string) *model.SMHPersonalSpace {
	t.Helper()
	user := model.User{Username: "refresh-user-" + agentType, Password: "x", Role: "user"}
	if err := model.DB(context.Background()).Create(&user).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	ins := model.Instance{
		Name:        "refresh-box",
		UserID:      user.ID,
		InstanceId:  "ins-refresh-" + agentType,
		AgentType:   agentType,
		RuntimeUser: "agentuser",
	}
	if err := model.DB(context.Background()).Create(&ins).Error; err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}
	space := model.SMHPersonalSpace{
		SpaceId:       "sp-refresh-" + agentType,
		UserId:        user.ID,
		InstanceId:    ins.ID,
		UserName:      user.Username,
		InstanceName:  ins.Name,
		CVMInstanceId: ins.InstanceId,
	}
	if err := model.DB(context.Background()).Create(&space).Error; err != nil {
		t.Fatalf("创建个人空间失败: %v", err)
	}
	return &space
}

// TestRefreshPersonalSpaceToken_EnsureTokenFailed 覆盖第一步 ensurePersonalSpaceToken 失败分支。
// 场景：缓存为空且 SMH 客户端未初始化 → ensurePersonalSpaceToken 直接报错 →
//
//	refreshPersonalSpaceToken 包装为 "获取 token 失败"。
func TestRefreshPersonalSpaceToken_EnsureTokenFailed(t *testing.T) {
	cleanup := initSMHRefreshTestDB(t)
	defer cleanup()

	// 不写 SiteConfig，且 SMH 未配置 → fetchPersonalSpaceToken 将返回错误
	space := createSpaceAndInstance(t, model.AgentTypeOpenClaw)

	err := refreshPersonalSpaceToken(context.Background(), space)
	if err == nil {
		t.Fatal("期望返回错误，实际 nil")
	}
	if !strings.Contains(err.Error(), "获取 token 失败") {
		t.Errorf("错误信息应包含 '获取 token 失败'，实际=%q", err.Error())
	}
}

// TestRefreshPersonalSpaceToken_SMHNotConfigured 覆盖 SMH 未配置分支。
// 场景：缓存命中 token（绕过 ensurePersonalSpaceToken 的 SMH API 调用），
//
//	但 SiteConfig 中 SMH 配置不完整 → IsConfigured 为 false → 返回 "SMH 未配置"。
func TestRefreshPersonalSpaceToken_SMHNotConfigured(t *testing.T) {
	cleanup := initSMHRefreshTestDB(t)
	defer cleanup()

	space := createSpaceAndInstance(t, model.AgentTypeOpenClaw)
	// 预填缓存，让 ensurePersonalSpaceToken 直接命中
	primeTokenCache(space.SpaceId, "cached-token")

	err := refreshPersonalSpaceToken(context.Background(), space)
	if err == nil {
		t.Fatal("期望返回错误，实际 nil")
	}
	if !strings.Contains(err.Error(), "SMH 未配置") {
		t.Errorf("错误信息应包含 'SMH 未配置'，实际=%q", err.Error())
	}
}

// TestRefreshPersonalSpaceToken_ResolveScriptFailed 覆盖 ResolveScript 失败分支。
// 场景：缓存命中 + SMH 已配置，但 agent_type 未在 scriptResolveTable 注册 →
//
//	ResolveScript 返回 error → 返回 "解析 set_smh_token 脚本失败"。
func TestRefreshPersonalSpaceToken_ResolveScriptFailed(t *testing.T) {
	cleanup := initSMHRefreshTestDB(t)
	defer cleanup()
	seedSMHConfigured(t)

	space := createSpaceAndInstance(t, "some_unknown_agent_type")
	primeTokenCache(space.SpaceId, "cached-token")

	err := refreshPersonalSpaceToken(context.Background(), space)
	if err == nil {
		t.Fatal("期望返回错误，实际 nil")
	}

	var re *hcommon.RichError
	if !errors.As(err, &re) {
		t.Fatalf("错误不是 RichError，实际=%T", err)
	}

	wanted := hcommon.I18nError(i18n.MsgSmhResolveScriptFailed, "set_smh_token", "some_unknown_agent_type")
	if !errors.Is(re, wanted) {
		t.Errorf("错误信息应包含 '解析 set_smh_token 脚本失败'，实际=%q", err.Error())
	}
}

// TestRefreshPersonalSpaceToken_RunScriptFailed 覆盖 RunScript 失败分支。
// 场景：缓存命中 + SMH 已配置 + ResolveScript 成功，但 runScriptFn mock 返回 error →
//
//	refreshPersonalSpaceToken 返回 "注入 SMH 环境变量失败"。
func TestRefreshPersonalSpaceToken_RunScriptFailed(t *testing.T) {
	cleanup := initSMHRefreshTestDB(t)
	defer cleanup()
	seedSMHConfigured(t)

	space := createSpaceAndInstance(t, model.AgentTypeOpenClaw)
	primeTokenCache(space.SpaceId, "cached-token")

	restore := mockRunScript(func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		return "", hcommon.I18nError(i18n.MsgTATFailed)
	})
	defer restore()

	err := refreshPersonalSpaceToken(context.Background(), space)
	if err == nil {
		t.Fatal("期望返回错误，实际 nil")
	}
	if !strings.Contains(err.Error(), "注入 SMH 环境变量失败") {
		t.Errorf("错误信息应包含 '注入 SMH 环境变量失败'，实际=%q", err.Error())
	}
}

// TestRefreshPersonalSpaceToken_Success 覆盖成功路径。
// 场景：缓存命中 + SMH 已配置 + ResolveScript 成功 + runScriptFn mock 返回 OK →
//
//	refreshPersonalSpaceToken 返回 nil，且 runScriptFn 应被以正确参数调用。
func TestRefreshPersonalSpaceToken_Success(t *testing.T) {
	cleanup := initSMHRefreshTestDB(t)
	defer cleanup()
	seedSMHConfigured(t)

	space := createSpaceAndInstance(t, model.AgentTypeOpenClaw)
	const cachedToken = "cached-success-token"
	primeTokenCache(space.SpaceId, cachedToken)

	var (
		gotInstanceId  string
		gotScriptName  string
		gotTimeout     uint64
		gotRuntimeUser string
		gotParams      map[string]string
		called         int
	)
	restore := mockRunScript(func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		called++
		gotInstanceId = instanceId
		gotScriptName = scriptName
		gotTimeout = timeout
		gotRuntimeUser = runtimeUser
		gotParams = params
		return "ok", nil
	})
	defer restore()

	err := refreshPersonalSpaceToken(context.Background(), space)
	if err != nil {
		t.Fatalf("期望成功，实际返回错误: %v", err)
	}
	// 成功路径下 controller 层应已直接回写 DB 的 LastPushedTokenExpiresAt 字段，
	// 避免下一轮 task 扫描重复下发。
	var persisted model.SMHPersonalSpace
	if err := model.DB(context.Background()).First(&persisted, space.ID).Error; err != nil {
		t.Fatalf("读取回写后的个人空间失败: %v", err)
	}
	if persisted.LastPushedTokenExpiresAt == nil || persisted.LastPushedTokenExpiresAt.IsZero() {
		t.Error("成功下发后 LastPushedTokenExpiresAt 应被回写，实际仍为空")
	}

	if called != 1 {
		t.Fatalf("runScriptFn 应被调用 1 次，实际 %d", called)
	}
	if gotInstanceId != space.CVMInstanceId {
		t.Errorf("instanceId 不匹配: 期望 %q, 实际 %q", space.CVMInstanceId, gotInstanceId)
	}
	if gotTimeout != 60 {
		t.Errorf("timeout 应为 60s，实际 %d", gotTimeout)
	}
	// runtimeUser 由 LookupRuntimeUser 从 Instance 表查出，createSpaceAndInstance 中设为 "agentuser"
	if gotRuntimeUser != "agentuser" {
		t.Errorf("runtimeUser 应为 'agentuser'，实际 %q", gotRuntimeUser)
	}
	// scriptName 由 ResolveScript("set_smh_token", AgentTypeOpenClaw) 解析，至少不应为空
	if gotScriptName == "" {
		t.Errorf("scriptName 不应为空")
	}
	// 校验核心 params
	if gotParams["accessToken"] != cachedToken {
		t.Errorf("accessToken 应为 %q，实际 %q", cachedToken, gotParams["accessToken"])
	}
	if gotParams["spaceId"] != space.SpaceId {
		t.Errorf("spaceId 应为 %q，实际 %q", space.SpaceId, gotParams["spaceId"])
	}
	if gotParams["basePath"] != "https://smh.example.com" {
		t.Errorf("basePath 应为 'https://smh.example.com'，实际 %q", gotParams["basePath"])
	}
	if gotParams["libraryId"] != "lib-test" {
		t.Errorf("libraryId 应为 'lib-test'，实际 %q", gotParams["libraryId"])
	}
}

// ---------- SyncPersonalSpaceEnv ----------
//
// 复用已建立的测试基础设施：
//   - initSMHRefreshTestDB：内存 SQLite + 清空 token 缓存
//   - seedSMHConfigured：写入完整 SMH 配置（SiteConfig + 两个 SMHSpace）
//   - createSpaceAndInstance：创建 Instance(AgentType=xxx, RuntimeUser=agentuser) + SMHPersonalSpace
//   - primeTokenCache：预置 token 绕过真实 SMH API 调用
//   - mockRunScript：替换 runScriptFn（版本同步测试已用）
//
// 注意：SyncPersonalSpaceEnv 内部会调用 ensureRuntimeUser / LookupAgentType，
// 它们从 DB 读 Instance.RuntimeUser / Instance.AgentType。因为 createSpaceAndInstance
// 已把这两个字段分别置为 "agentuser" 和入参 agentType，所以 ensureRuntimeUser 会直接命中
// 第一步的"DB 已有值"分支返回 "agentuser"，不会触发 detect_install 脚本，
// 从而测试只需断言 runScriptFn 的一次 init_smh_env / remove_smh_env 调用。

// TestSyncPersonalSpaceEnv_Install_Success 覆盖 install=true 成功路径：
// token 缓存命中 + SMH 已配置 + ResolveScript("init_smh_env") 成功 + runScriptFn 成功，
// 预期 runScriptFn 被以 init_smh_env.sh / 180s / agentuser / 全量 params 调用，
// 且 DB 中 SMHPersonalSpace.env_initialized 被置为 true。
func TestSyncPersonalSpaceEnv_Install_Success(t *testing.T) {
	cleanup := initSMHRefreshTestDB(t)
	defer cleanup()
	seedSMHConfigured(t)

	space := createSpaceAndInstance(t, model.AgentTypeOpenClaw)
	const cachedToken = "cached-install-token"
	primeTokenCache(space.SpaceId, cachedToken)

	var (
		called         int
		gotInstanceId  string
		gotScriptName  string
		gotTimeout     uint64
		gotRuntimeUser string
		gotParams      map[string]string
	)
	restore := mockRunScript(func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		called++
		gotInstanceId = instanceId
		gotScriptName = scriptName
		gotTimeout = timeout
		gotRuntimeUser = runtimeUser
		gotParams = params
		return "ok", nil
	})
	defer restore()

	if err := SyncPersonalSpaceEnv(context.Background(), space, true); err != nil {
		t.Fatalf("期望成功，实际返回错误: %v", err)
	}

	if called != 1 {
		t.Fatalf("runScriptFn 应被调用 1 次，实际 %d", called)
	}
	if gotInstanceId != space.CVMInstanceId {
		t.Errorf("instanceId 不匹配: 期望 %q, 实际 %q", space.CVMInstanceId, gotInstanceId)
	}
	if gotScriptName != "init_smh_env.sh" {
		t.Errorf("scriptName 应为 init_smh_env.sh，实际 %q", gotScriptName)
	}
	if gotTimeout != 180 {
		t.Errorf("timeout 应为 180s，实际 %d", gotTimeout)
	}
	if gotRuntimeUser != "agentuser" {
		t.Errorf("runtimeUser 应为 'agentuser'，实际 %q", gotRuntimeUser)
	}
	// 核心 params 校验
	if gotParams["accessToken"] != cachedToken {
		t.Errorf("accessToken 应为 %q，实际 %q", cachedToken, gotParams["accessToken"])
	}
	if gotParams["spaceId"] != space.SpaceId {
		t.Errorf("spaceId 应为 %q，实际 %q", space.SpaceId, gotParams["spaceId"])
	}
	if gotParams["basePath"] != "https://smh.example.com" {
		t.Errorf("basePath 应为 'https://smh.example.com'，实际 %q", gotParams["basePath"])
	}
	if gotParams["libraryId"] != "lib-test" {
		t.Errorf("libraryId 应为 'lib-test'，实际 %q", gotParams["libraryId"])
	}
	if gotParams["skill_name"] != "tencent-agent-storage" {
		t.Errorf("skill_name 应为 'tencent-agent-storage'，实际 %q", gotParams["skill_name"])
	}
	if gotParams["agent_type"] != model.AgentTypeOpenClaw {
		t.Errorf("agent_type 应为 %q，实际 %q", model.AgentTypeOpenClaw, gotParams["agent_type"])
	}

	// 验证 DB 中 env_initialized 被置为 true
	var updated model.SMHPersonalSpace
	if err := model.DB(context.Background()).First(&updated, space.ID).Error; err != nil {
		t.Fatalf("查询个人空间失败: %v", err)
	}
	if !updated.EnvInitialized {
		t.Error("install 成功后 env_initialized 应为 true")
	}
	if updated.EnvProvisionRev != CurrentSMHProvisionRev {
		t.Errorf("install 成功后 env_provision_rev 应为 %d，实际 %d", CurrentSMHProvisionRev, updated.EnvProvisionRev)
	}
	if !space.EnvInitialized {
		t.Error("install 成功后内存对象 EnvInitialized 应同步为 true")
	}
	if space.EnvProvisionRev != CurrentSMHProvisionRev {
		t.Errorf("install 成功后内存对象 EnvProvisionRev 应为 %d，实际 %d", CurrentSMHProvisionRev, space.EnvProvisionRev)
	}
}

// TestSyncPersonalSpaceEnv_Install_EnsureTokenFailed 覆盖 install 分支中 ensurePersonalSpaceToken 失败：
// 缓存为空 + SMH 未配置 → fetchPersonalSpaceToken 报错 →
// 期望错误包含 "获取 token 失败"，且 runScriptFn 不应被调用。
func TestSyncPersonalSpaceEnv_Install_EnsureTokenFailed(t *testing.T) {
	cleanup := initSMHRefreshTestDB(t)
	defer cleanup()
	// 不调用 seedSMHConfigured；不预置 token 缓存
	space := createSpaceAndInstance(t, model.AgentTypeOpenClaw)

	var called int
	restore := mockRunScript(func(ctx context.Context, _, _ string, _ uint64, _ string, _ func(chunk string), _ map[string]string) (string, error) {
		called++
		return "ok", nil
	})
	defer restore()

	err := SyncPersonalSpaceEnv(context.Background(), space, true)
	if err == nil {
		t.Fatal("期望返回错误，实际 nil")
	}
	if !strings.Contains(err.Error(), "获取 token 失败") {
		t.Errorf("错误信息应包含 '获取 token 失败'，实际=%q", err.Error())
	}
	if called != 0 {
		t.Errorf("token 获取失败时不应调用 runScriptFn，实际调用 %d 次", called)
	}

	var updated model.SMHPersonalSpace
	model.DB(context.Background()).First(&updated, space.ID)
	if updated.EnvInitialized {
		t.Error("失败路径不应将 env_initialized 置为 true")
	}
}

// TestSyncPersonalSpaceEnv_Install_SMHNotConfigured 覆盖 SMH 未配置分支：
// 缓存命中绕过 ensurePersonalSpaceToken 的 SMH API 调用，
// 但 SiteConfig 中 SMH 配置不完整 → 返回 "SMH 未配置"。
func TestSyncPersonalSpaceEnv_Install_SMHNotConfigured(t *testing.T) {
	cleanup := initSMHRefreshTestDB(t)
	defer cleanup()
	// 不调用 seedSMHConfigured
	space := createSpaceAndInstance(t, model.AgentTypeOpenClaw)
	primeTokenCache(space.SpaceId, "cached-token")

	var called int
	restore := mockRunScript(func(ctx context.Context, _, _ string, _ uint64, _ string, _ func(chunk string), _ map[string]string) (string, error) {
		called++
		return "ok", nil
	})
	defer restore()

	err := SyncPersonalSpaceEnv(context.Background(), space, true)
	if err == nil {
		t.Fatal("期望返回错误，实际 nil")
	}
	if !strings.Contains(err.Error(), "SMH 未配置") {
		t.Errorf("错误信息应包含 'SMH 未配置'，实际=%q", err.Error())
	}
	if called != 0 {
		t.Errorf("SMH 未配置时不应调用 runScriptFn，实际调用 %d 次", called)
	}
}

// TestSyncPersonalSpaceEnv_Install_ResolveScriptFailed 覆盖 ResolveScript 失败分支：
// agent_type 未在 scriptResolveTable 注册 → ResolveScript 返回 error →
// 期望错误包含 "解析 init_smh_env 脚本失败"。
func TestSyncPersonalSpaceEnv_Install_ResolveScriptFailed(t *testing.T) {
	ctx := context.Background()

	cleanup := initSMHRefreshTestDB(t)
	defer cleanup()
	seedSMHConfigured(t)

	space := createSpaceAndInstance(t, "some_unknown_agent_type")
	primeTokenCache(space.SpaceId, "cached-token")

	var called int
	restore := mockRunScript(func(ctx context.Context, _, _ string, _ uint64, _ string, _ func(chunk string), _ map[string]string) (string, error) {
		called++
		return "ok", nil
	})
	defer restore()

	err := SyncPersonalSpaceEnv(context.Background(), space, true)
	if err == nil {
		t.Fatal("期望返回错误，实际 nil")
	}

	wanted := hcommon.I18nError(i18n.MsgSmhResolveScriptFailed, "remove_smh_env", "some_unknown_agent_type")
	if !errors.Is(err, wanted) {
		t.Errorf("错误应为 %s，实际 %s", wanted.ErrorMessage(ctx), hcommon.ErrorMessageWithCtx(ctx, err))
	}

	if called != 0 {
		t.Errorf("ResolveScript 失败时不应调用 runScriptFn，实际调用 %d 次", called)
	}
}

// TestSyncPersonalSpaceEnv_Install_RunScriptFailed 覆盖 install 分支 runScriptFn 失败：
// 脚本执行返回 error → 期望错误包含 "初始化 SMH 环境失败"，且 env_initialized 不被置为 true。
func TestSyncPersonalSpaceEnv_Install_RunScriptFailed(t *testing.T) {
	cleanup := initSMHRefreshTestDB(t)
	defer cleanup()
	seedSMHConfigured(t)

	space := createSpaceAndInstance(t, model.AgentTypeOpenClaw)
	primeTokenCache(space.SpaceId, "cached-token")

	restore := mockRunScript(func(ctx context.Context, _, _ string, _ uint64, _ string, _ func(chunk string), _ map[string]string) (string, error) {
		return "", hcommon.I18nError(i18n.MsgTATFailed)
	})
	defer restore()

	err := SyncPersonalSpaceEnv(context.Background(), space, true)
	if err == nil {
		t.Fatal("期望返回错误，实际 nil")
	}

	wanted := hcommon.I18nError(i18n.MsgSmhInitEnvFailed)
	if !errors.Is(err, wanted) {
		t.Errorf("错误信息应包含 '初始化 SMH 环境失败'，实际=%q", err.Error())
	}

	var updated model.SMHPersonalSpace
	model.DB(context.Background()).First(&updated, space.ID)
	if updated.EnvInitialized {
		t.Error("RunScript 失败时不应将 env_initialized 置为 true")
	}
	if updated.EnvProvisionRev != 0 {
		t.Errorf("RunScript 失败时不应更新 env_provision_rev，实际 %d", updated.EnvProvisionRev)
	}
}

// TestSyncPersonalSpaceEnv_Install_DispatchesScriptByAgentType 验证 install 分支按 agent_type
// 分派不同脚本（openclaw/hermes/ace）。
// 背景：scriptResolveTable 中 init_smh_env 对三类 agent_type 都登记了同名脚本 init_smh_env.sh；
// 本用例同时充当回归锁，若未来按 agent_type 拆分为 init_smh_env_hermes.sh 等，此断言会提醒更新契约。
func TestSyncPersonalSpaceEnv_Install_DispatchesScriptByAgentType(t *testing.T) {
	cases := []struct {
		name       string
		agentType  string
		wantScript string
	}{
		{"openclaw", model.AgentTypeOpenClaw, "init_smh_env.sh"},
		{"hermes", model.AgentTypeHermes, "init_smh_env.sh"},
		{"ace", model.AgentTypeLightclawACE, "init_smh_env.sh"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := initSMHRefreshTestDB(t)
			defer cleanup()
			seedSMHConfigured(t)

			space := createSpaceAndInstance(t, tc.agentType)
			primeTokenCache(space.SpaceId, "cached-token-"+tc.name)

			var gotScript string
			var gotAgentTypeParam string
			restore := mockRunScript(func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
				gotScript = scriptName
				gotAgentTypeParam = params["agent_type"]
				return "ok", nil
			})
			defer restore()

			if err := SyncPersonalSpaceEnv(context.Background(), space, true); err != nil {
				t.Fatalf("期望成功，实际错误: %v", err)
			}
			if gotScript != tc.wantScript {
				t.Errorf("agent_type=%s 期望脚本=%s 实际=%s", tc.agentType, tc.wantScript, gotScript)
			}
			if gotAgentTypeParam != tc.agentType {
				t.Errorf("agent_type 参数透传错误: 期望 %q 实际 %q", tc.agentType, gotAgentTypeParam)
			}
		})
	}
}

// TestSyncPersonalSpaceEnv_Uninstall_Success 覆盖 install=false 成功路径：
// ResolveScript("remove_smh_env") 成功 + runScriptFn 成功，
// 期望 runScriptFn 被以 remove_smh_env.sh / 120s / agentuser 调用，
// 且 env_initialized 被置为 false（从 true 回落）。
func TestSyncPersonalSpaceEnv_Uninstall_Success(t *testing.T) {
	cleanup := initSMHRefreshTestDB(t)
	defer cleanup()
	// uninstall 分支不需要 SMH 配置；但也不影响，这里保持默认不 seed

	space := createSpaceAndInstance(t, model.AgentTypeOpenClaw)
	// 先把 env_initialized 置为 true，用于验证 uninstall 后会回落
	if err := model.DB(context.Background()).Model(space).Update("env_initialized", true).Error; err != nil {
		t.Fatalf("预置 env_initialized=true 失败: %v", err)
	}
	if err := model.DB(context.Background()).Model(space).Update("env_provision_rev", CurrentSMHProvisionRev).Error; err != nil {
		t.Fatalf("预置 env_provision_rev 失败: %v", err)
	}

	var (
		called         int
		gotScriptName  string
		gotTimeout     uint64
		gotRuntimeUser string
		gotParams      map[string]string
	)
	restore := mockRunScript(func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		called++
		gotScriptName = scriptName
		gotTimeout = timeout
		gotRuntimeUser = runtimeUser
		gotParams = params
		return "ok", nil
	})
	defer restore()

	if err := SyncPersonalSpaceEnv(context.Background(), space, false); err != nil {
		t.Fatalf("期望成功，实际返回错误: %v", err)
	}

	if called != 1 {
		t.Fatalf("runScriptFn 应被调用 1 次，实际 %d", called)
	}
	if gotScriptName != "remove_smh_env.sh" {
		t.Errorf("scriptName 应为 remove_smh_env.sh，实际 %q", gotScriptName)
	}
	if gotTimeout != 120 {
		t.Errorf("timeout 应为 120s，实际 %d", gotTimeout)
	}
	if gotRuntimeUser != "agentuser" {
		t.Errorf("runtimeUser 应为 'agentuser'，实际 %q", gotRuntimeUser)
	}
	// uninstall 仅透传 agent_type，不应带 spaceId/accessToken 等敏感信息
	if gotParams["agent_type"] != model.AgentTypeOpenClaw {
		t.Errorf("agent_type 应为 %q，实际 %q", model.AgentTypeOpenClaw, gotParams["agent_type"])
	}
	if _, ok := gotParams["accessToken"]; ok {
		t.Errorf("uninstall 不应透传 accessToken，实际 params=%v", gotParams)
	}
	if _, ok := gotParams["spaceId"]; ok {
		t.Errorf("uninstall 不应透传 spaceId，实际 params=%v", gotParams)
	}

	// 验证 DB 中 env_initialized 已回落为 false
	var updated model.SMHPersonalSpace
	if err := model.DB(context.Background()).First(&updated, space.ID).Error; err != nil {
		t.Fatalf("查询个人空间失败: %v", err)
	}
	if updated.EnvInitialized {
		t.Error("uninstall 成功后 env_initialized 应为 false")
	}
	if updated.EnvProvisionRev != CurrentSMHProvisionRev {
		t.Errorf("uninstall 成功后 env_provision_rev 应保留为 %d，实际 %d", CurrentSMHProvisionRev, updated.EnvProvisionRev)
	}
	if space.EnvInitialized {
		t.Error("uninstall 成功后内存对象 EnvInitialized 应同步为 false")
	}
}

func TestSyncPersonalSpaceEnv_Install_DBUpdateFailed(t *testing.T) {
	cleanup := initSMHRefreshTestDB(t)
	defer cleanup()
	seedSMHConfigured(t)

	space := createSpaceAndInstance(t, model.AgentTypeOpenClaw)
	primeTokenCache(space.SpaceId, "cached-token")

	restore := mockRunScript(func(ctx context.Context, _, _ string, _ uint64, _ string, _ func(chunk string), _ map[string]string) (string, error) {
		model.DB(context.Background()).Exec("PRAGMA query_only = ON")
		return "ok", nil
	})
	defer restore()
	defer model.DB(context.Background()).Exec("PRAGMA query_only = OFF")

	err := SyncPersonalSpaceEnv(context.Background(), space, true)
	if err == nil {
		t.Fatal("期望返回 DB 写回错误，实际 nil")
	}
	if !strings.Contains(err.Error(), "更新 SMH 环境状态失败") {
		t.Errorf("错误信息应包含 '更新 SMH 环境状态失败'，实际=%q", err.Error())
	}

	model.DB(context.Background()).Exec("PRAGMA query_only = OFF")
	var updated model.SMHPersonalSpace
	if err := model.DB(context.Background()).First(&updated, space.ID).Error; err != nil {
		t.Fatalf("查询个人空间失败: %v", err)
	}
	if updated.EnvInitialized {
		t.Error("DB 写回失败时不应将 env_initialized 置为 true")
	}
	if updated.EnvProvisionRev != 0 {
		t.Errorf("DB 写回失败时不应更新 env_provision_rev，实际 %d", updated.EnvProvisionRev)
	}
}

// TestSyncPersonalSpaceEnv_Uninstall_ResolveScriptFailed 覆盖 uninstall 分支 ResolveScript 失败：
// 未知 agent_type → 期望错误包含 "解析 remove_smh_env 脚本失败"。
func TestSyncPersonalSpaceEnv_Uninstall_ResolveScriptFailed(t *testing.T) {
	cleanup := initSMHRefreshTestDB(t)
	defer cleanup()

	space := createSpaceAndInstance(t, "some_unknown_agent_type")

	var called int
	restore := mockRunScript(func(ctx context.Context, _, _ string, _ uint64, _ string, _ func(chunk string), _ map[string]string) (string, error) {
		called++
		return "ok", nil
	})
	defer restore()

	err := SyncPersonalSpaceEnv(context.Background(), space, false)
	if err == nil {
		t.Fatal("期望返回错误，实际 nil")
	}

	wanted := hcommon.I18nError(i18n.MsgSmhResolveScriptFailed, "remove_smh_env", "some_unknown_agent_type")
	if !errors.Is(err, wanted) {
		t.Errorf("错误信息应包含 '解析 remove_smh_env 脚本失败'，实际=%q", err.Error())
	}

	if called != 0 {
		t.Errorf("ResolveScript 失败时不应调用 runScriptFn，实际调用 %d 次", called)
	}
}

// TestSyncPersonalSpaceEnv_Uninstall_RunScriptFailed 覆盖 uninstall 分支 runScriptFn 失败：
// 脚本执行返回 error → 期望错误包含 "卸载 SMH 环境失败"，
// 且 env_initialized 不应被改写（保持原先的 true）。
func TestSyncPersonalSpaceEnv_Uninstall_RunScriptFailed(t *testing.T) {
	cleanup := initSMHRefreshTestDB(t)
	defer cleanup()

	space := createSpaceAndInstance(t, model.AgentTypeOpenClaw)
	// 预置 env_initialized=true 以便验证失败时不被错误地改成 false
	if err := model.DB(context.Background()).Model(space).Update("env_initialized", true).Error; err != nil {
		t.Fatalf("预置 env_initialized=true 失败: %v", err)
	}

	restore := mockRunScript(func(ctx context.Context, _, _ string, _ uint64, _ string, _ func(chunk string), _ map[string]string) (string, error) {
		return "", hcommon.I18nError(i18n.MsgTATFailed)
	})
	defer restore()

	err := SyncPersonalSpaceEnv(context.Background(), space, false)
	if err == nil {
		t.Fatal("期望返回错误，实际 nil")
	}

	wanted := hcommon.I18nError(i18n.MsgSmhRemoveEnvFailed)
	if !errors.Is(err, wanted) {
		t.Errorf("错误信息应包含 '卸载 SMH 环境失败'，实际=%q", err.Error())
	}

	var updated model.SMHPersonalSpace
	model.DB(context.Background()).First(&updated, space.ID)
	if !updated.EnvInitialized {
		t.Error("卸载失败时不应修改 env_initialized（应保持 true）")
	}
}

// TestSyncPersonalSpaceEnv_Uninstall_DBUpdateFailed 覆盖 uninstall 分支 runScriptFn 成功、
// 但写回 env_initialized=false 时 DB 写操作失败的分支（smh.go: "更新 SMH 环境卸载状态失败"）。
//
// 构造方式：与 install 分支的 TestSyncPersonalSpaceEnv_Install_DBUpdateFailed 对齐 —
// 在 mock 的 runScriptFn 内对 SQLite 执行 `PRAGMA query_only = ON`，
// 使后续的 UPDATE 语句返回 "attempt to write a readonly database"，
// 从而命中错误分支；测试结束前再关掉 query_only，避免影响后续断言查询。
func TestSyncPersonalSpaceEnv_Uninstall_DBUpdateFailed(t *testing.T) {
	cleanup := initSMHRefreshTestDB(t)
	defer cleanup()

	space := createSpaceAndInstance(t, model.AgentTypeOpenClaw)
	// 预置 env_initialized=true，以便校验失败路径不会将其改为 false
	if err := model.DB(context.Background()).Model(space).Update("env_initialized", true).Error; err != nil {
		t.Fatalf("预置 env_initialized=true 失败: %v", err)
	}

	restore := mockRunScript(func(ctx context.Context, _, _ string, _ uint64, _ string, _ func(chunk string), _ map[string]string) (string, error) {
		// 进入脚本后立刻把 DB 置为只读，让紧随其后的 UPDATE 报错
		model.DB(context.Background()).Exec("PRAGMA query_only = ON")
		return "ok", nil
	})
	defer restore()
	defer model.DB(context.Background()).Exec("PRAGMA query_only = OFF")

	err := SyncPersonalSpaceEnv(context.Background(), space, false)
	if err == nil {
		t.Fatal("期望返回 DB 写回错误，实际 nil")
	}

	wanted := hcommon.I18nError(i18n.MsgSmhUpdateEnvStatusFailed)
	if !errors.Is(err, wanted) {
		t.Errorf("错误信息应包含 '更新 SMH 环境卸载状态失败'，实际=%q", err.Error())
	}

	// 断言前恢复写权限
	model.DB(context.Background()).Exec("PRAGMA query_only = OFF")
	var updated model.SMHPersonalSpace
	if err := model.DB(context.Background()).First(&updated, space.ID).Error; err != nil {
		t.Fatalf("查询个人空间失败: %v", err)
	}
	// DB 写回失败 → env_initialized 应保持预置的 true（UPDATE 未真正生效）
	if !updated.EnvInitialized {
		t.Error("DB 写回失败时 env_initialized 应保持 true（UPDATE 未生效）")
	}
	// 注意：此处不断言 space.EnvInitialized ——
	// GORM 的 Update 即便 SQL 失败也会把值写入 struct 字段，属于通用行为，
	// 与本分支（return error 前不执行 `space.EnvInitialized = false`）的预期无关，
	// 避免耦合 GORM 内部实现。
}

// TestRefreshPersonalSpaceToken_DBWriteBackFailed 覆盖 refreshPersonalSpaceToken 成功路径中
// 回写 last_pushed_token_expires_at 失败的分支（smh.go: "回写 last_pushed_token_expires_at 失败"）。
//
// 该分支仅 slog.Warn 记录日志、不影响函数返回值 —— 即便回写失败 refreshPersonalSpaceToken
// 仍应返回 nil。因此断言方式为：
//  1. 函数返回 nil（TAT 已成功 → 业务结果成功）；
//  2. DB 中 LastPushedTokenExpiresAt 仍为空 / 零值（UPDATE 实际未生效，证明确实走到 Warn 分支）。
//
// 构造方式与上面一致：mock runScriptFn 进入后把 DB 置为 query_only。
func TestRefreshPersonalSpaceToken_DBWriteBackFailed(t *testing.T) {
	cleanup := initSMHRefreshTestDB(t)
	defer cleanup()
	seedSMHConfigured(t)

	space := createSpaceAndInstance(t, model.AgentTypeOpenClaw)
	const cachedToken = "cached-writeback-failure-token"
	primeTokenCache(space.SpaceId, cachedToken)

	restore := mockRunScript(func(ctx context.Context, _, _ string, _ uint64, _ string, _ func(chunk string), _ map[string]string) (string, error) {
		// TAT 注入成功，但让紧随其后的 UPDATE last_pushed_token_expires_at 失败
		model.DB(context.Background()).Exec("PRAGMA query_only = ON")
		return "ok", nil
	})
	defer restore()
	defer model.DB(context.Background()).Exec("PRAGMA query_only = OFF")

	// TAT 已成功 → 即便回写失败，函数也应返回 nil（由下一轮任务自愈）
	if err := refreshPersonalSpaceToken(context.Background(), space); err != nil {
		t.Fatalf("TAT 成功时即便回写失败也应返回 nil，实际错误=%v", err)
	}

	// 恢复写权限后断言：UPDATE 未生效，LastPushedTokenExpiresAt 仍为空
	model.DB(context.Background()).Exec("PRAGMA query_only = OFF")
	var persisted model.SMHPersonalSpace
	if err := model.DB(context.Background()).First(&persisted, space.ID).Error; err != nil {
		t.Fatalf("读取个人空间失败: %v", err)
	}
	if persisted.LastPushedTokenExpiresAt != nil && !persisted.LastPushedTokenExpiresAt.IsZero() {
		t.Errorf("回写失败时 LastPushedTokenExpiresAt 不应被更新，实际=%v", persisted.LastPushedTokenExpiresAt)
	}
}
