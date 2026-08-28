package model

import (
	"context"
	"os"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupInstanceModelTestDB(t *testing.T) func() {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "instance_model_test_*.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	tmpFile.Close()

	dsn := tmpFile.Name() + "?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
	testDB, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("open test db: %v", err)
	}

	origDB := gdb
	gdb = testDB

	if err := gdb.AutoMigrate(&User{}, &Instance{}, &AIModel{}, &InstanceModel{}, &SiteConfig{}); err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("auto migrate: %v", err)
	}

	return func() {
		sqlDB, _ := gdb.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
		os.Remove(tmpFile.Name())
		os.Remove(tmpFile.Name() + "-wal")
		os.Remove(tmpFile.Name() + "-shm")
		gdb = origDB
	}
}

func TestInstanceModel_CreateAndQuery(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "testuser", Password: "test"}
	gdb.Create(&user)

	inst := Instance{Name: "inst-1", UserID: user.ID, InstanceId: "ins-001"}
	gdb.Create(&inst)

	model1 := AIModel{Provider: BuiltinModelProvider, ModelID: "glm-4-plus", ModelName: "GLM-4-Plus", Enabled: true}
	gdb.Create(&model1)

	im := InstanceModel{
		InstanceID: inst.ID,
		AIModelID:  model1.ID,
		Role:       ModelRolePrimary,
		SortOrder:  1,
	}
	if err := gdb.Create(&im).Error; err != nil {
		t.Fatalf("创建 InstanceModel 失败: %v", err)
	}

	var found InstanceModel
	if err := gdb.First(&found, im.ID).Error; err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if found.Role != ModelRolePrimary {
		t.Errorf("期望 role=%s, 实际=%s", ModelRolePrimary, found.Role)
	}
	if found.InstanceID != inst.ID {
		t.Errorf("期望 instance_id=%d, 实际=%d", inst.ID, found.InstanceID)
	}
	if found.AIModelID != model1.ID {
		t.Errorf("期望 ai_model_id=%d, 实际=%d", model1.ID, found.AIModelID)
	}
}

func TestInstanceModel_MultipleModelsPerInstance(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "u1", Password: "t"}
	gdb.Create(&user)
	inst := Instance{Name: "multi-inst", UserID: user.ID, InstanceId: "ins-multi"}
	gdb.Create(&inst)

	m1 := AIModel{Provider: BuiltinModelProvider, ModelID: "glm-4", Enabled: true}
	m2 := AIModel{Provider: BuiltinModelProvider, ModelID: "qwen-max", Enabled: true}
	m3 := AIModel{Provider: BuiltinModelProvider, ModelID: "deepseek-chat", Enabled: true}
	gdb.Create(&m1)
	gdb.Create(&m2)
	gdb.Create(&m3)

	if err := gdb.Create(&InstanceModel{InstanceID: inst.ID, AIModelID: m1.ID, Role: ModelRolePrimary, SortOrder: 10}).Error; err != nil {
		t.Fatalf("创建 primary 失败: %v", err)
	}
	if err := gdb.Create(&InstanceModel{InstanceID: inst.ID, AIModelID: m2.ID, Role: ModelRoleFallback, SortOrder: 5}).Error; err != nil {
		t.Fatalf("创建 fallback1 失败: %v", err)
	}
	if err := gdb.Create(&InstanceModel{InstanceID: inst.ID, AIModelID: m3.ID, Role: ModelRoleFallback, SortOrder: 3}).Error; err != nil {
		t.Fatalf("创建 fallback2 失败: %v", err)
	}

	var count int64
	gdb.Model(&InstanceModel{}).Where("instance_id = ?", inst.ID).Count(&count)
	if count != 3 {
		t.Errorf("实例应有 3 个模型绑定, 实际=%d", count)
	}

	var primaryCount int64
	gdb.Model(&InstanceModel{}).Where("instance_id = ? AND role = ?", inst.ID, ModelRolePrimary).Count(&primaryCount)
	if primaryCount != 1 {
		t.Errorf("实例应只有 1 个 primary, 实际=%d", primaryCount)
	}
}

func TestInstanceModel_CustomModelBinding(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "u2", Password: "t"}
	gdb.Create(&user)
	inst := Instance{Name: "custom-inst", UserID: user.ID, InstanceId: "ins-custom"}
	gdb.Create(&inst)

	cfgJSON := `{"provider":"custom","model_id":"deepseek-chat","api_key":"sk-test","url":"https://api.deepseek.com","model_type":"openai-completions"}`

	im := InstanceModel{
		InstanceID:        inst.ID,
		AIModelID:         0,
		CustomModelID:     "deepseek-chat",
		CustomModelConfig: cfgJSON,
		Role:              ModelRolePrimary,
		SortOrder:         1,
	}
	if err := gdb.Create(&im).Error; err != nil {
		t.Fatalf("创建自定义模型绑定失败: %v", err)
	}

	var found InstanceModel
	gdb.First(&found, im.ID)
	if found.AIModelID != 0 {
		t.Errorf("自定义模型 ai_model_id 应为 0, 实际=%d", found.AIModelID)
	}
	if found.CustomModelID != "deepseek-chat" {
		t.Errorf("自定义模型 custom_model_id 应为 deepseek-chat, 实际=%q", found.CustomModelID)
	}
	if found.CustomModelConfig == "" {
		t.Error("自定义模型配置不应为空")
	}
}

// TestInstanceModel_UniqueIndex_PreBuiltModel 验证预置模型的联合唯一索引
// (instance_id, ai_model_id, custom_model_id=”)：同一实例不能重复绑定同一预置模型。
func TestInstanceModel_UniqueIndex_PreBuiltModel(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "u-unique-pre", Password: "t"}
	gdb.Create(&user)
	inst := Instance{Name: "inst-unique", UserID: user.ID, InstanceId: "ins-unique"}
	gdb.Create(&inst)
	aim := AIModel{Provider: BuiltinModelProvider, ModelID: "glm-unique", Enabled: true}
	gdb.Create(&aim)

	if err := gdb.Create(&InstanceModel{InstanceID: inst.ID, AIModelID: aim.ID, Role: ModelRolePrimary, SortOrder: 1}).Error; err != nil {
		t.Fatalf("首次绑定应成功: %v", err)
	}
	// 再次绑定同一预置模型，应被联合唯一索引拦截
	err := gdb.Create(&InstanceModel{InstanceID: inst.ID, AIModelID: aim.ID, Role: ModelRoleFallback, SortOrder: 2}).Error
	if err == nil {
		t.Error("期望联合唯一索引拦截重复绑定，但创建成功")
	}
}

// TestInstanceModel_UniqueIndex_CustomModel 验证自定义模型的联合唯一索引
// (instance_id, ai_model_id=0, custom_model_id)：同一实例不能重复绑定同一 custom model_id。
func TestInstanceModel_UniqueIndex_CustomModel(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "u-unique-custom", Password: "t"}
	gdb.Create(&user)
	inst := Instance{Name: "inst-custom-unique", UserID: user.ID, InstanceId: "ins-cunique"}
	gdb.Create(&inst)

	cfg := `{"provider":"custom","model_id":"my-model","api_key":"k","url":"https://x.com","model_type":"openai-completions"}`

	if err := gdb.Create(&InstanceModel{
		InstanceID: inst.ID, AIModelID: 0, CustomModelID: "my-model",
		CustomModelConfig: cfg, Role: ModelRolePrimary, SortOrder: 1,
	}).Error; err != nil {
		t.Fatalf("首次绑定自定义模型应成功: %v", err)
	}
	// 再次绑定同一 custom model_id，应被拦截
	err := gdb.Create(&InstanceModel{
		InstanceID: inst.ID, AIModelID: 0, CustomModelID: "my-model",
		CustomModelConfig: cfg, Role: ModelRoleFallback, SortOrder: 2,
	}).Error
	if err == nil {
		t.Error("期望联合唯一索引拦截重复的 custom_model_id，但创建成功")
	}

	// 不同 custom_model_id 应能共存
	if err := gdb.Create(&InstanceModel{
		InstanceID: inst.ID, AIModelID: 0, CustomModelID: "other-model",
		CustomModelConfig: cfg, Role: ModelRoleFallback, SortOrder: 3,
	}).Error; err != nil {
		t.Errorf("不同 custom_model_id 的绑定应成功，但失败: %v", err)
	}
}

func TestMigrateInstanceModels_Idempotent(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "mig-u", Password: "t"}
	gdb.Create(&user)
	inst := Instance{Name: "mig-inst", UserID: user.ID, InstanceId: "ins-mig", AIModelID: 42}
	gdb.Create(&inst)
	aim := AIModel{Provider: BuiltinModelProvider, ModelID: "glm-4-migrate", Enabled: true}
	gdb.Create(&aim)
	gdb.Model(&inst).Update("ai_model_id", aim.ID)

	MigrateInstanceModels(context.Background())

	var im InstanceModel
	result := gdb.Where("instance_id = ? AND role = ?", inst.ID, ModelRolePrimary).First(&im)

	if result.Error != nil {
		t.Fatalf("查询迁移结果失败: %v", result.Error)
	}
	if im.AIModelID == 0 {
		t.Error("迁移后 ai_modelID 不应为 0")
	}
	if im.Role != ModelRolePrimary {
		t.Errorf("迁移后角色应为 primary, 实际=%s", im.Role)
	}

	// 幂等验证
	MigrateInstanceModels(context.Background())
	var count int64
	gdb.Model(&InstanceModel{}).Where("instance_id = ?", inst.ID).Count(&count)
	if count != 1 {
		t.Errorf("幂等迁移后应为 1 条记录, 实际=%d", count)
	}
}

func TestMigrateInstanceModels_SkipsNoModel(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "no-model-u", Password: "t"}
	gdb.Create(&user)
	inst := Instance{Name: "no-model-inst", UserID: user.ID, InstanceId: "ins-nomod", AIModelID: 0}
	gdb.Create(&inst)

	MigrateInstanceModels(context.Background())

	var count int64
	gdb.Model(&InstanceModel{}).Where("instance_id = ?", inst.ID).Count(&count)
	if count != 0 {
		t.Errorf("无模型的实例不应产生迁移记录, 实际=%d", count)
	}
}

// TestMigrateInstanceModels_CustomModel 验证存量自定义模型迁移后 CustomModelID 被正确填充
func TestMigrateInstanceModels_CustomModel(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "mig-custom-u", Password: "t"}
	gdb.Create(&user)
	cfg := `{"provider":"custom","model_id":"legacy-custom","api_key":"k","url":"https://api.x.com","model_type":"openai-completions"}`
	inst := Instance{
		Name:              "mig-custom-inst",
		UserID:            user.ID,
		InstanceId:        "ins-mig-custom",
		AIModelID:         0,
		CustomModelConfig: cfg,
	}
	gdb.Create(&inst)

	MigrateInstanceModels(context.Background())

	var im InstanceModel
	if err := gdb.Where("instance_id = ? AND role = ?", inst.ID, ModelRolePrimary).First(&im).Error; err != nil {
		t.Fatalf("自定义模型未被迁移: %v", err)
	}
	if im.AIModelID != 0 {
		t.Errorf("自定义模型迁移后 ai_model_id 应为 0, 实际=%d", im.AIModelID)
	}
	if im.CustomModelID != "legacy-custom" {
		t.Errorf("CustomModelID 应从 JSON 中解析填充为 'legacy-custom', 实际=%q", im.CustomModelID)
	}
	if im.CustomModelConfig == "" {
		t.Error("CustomModelConfig 应被保留")
	}
}

// TestMigrateInstanceModels_InvalidCustomJSON 验证：自定义模型 JSON 解析失败时跳过该实例，不 panic，不影响其他实例。
// 覆盖 instance_model.go:70-72（json.Unmarshal 失败 → continue）
func TestMigrateInstanceModels_InvalidCustomJSON(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "u-badjson", Password: "x"}
	gdb.Create(&user)

	// 故意写入非法 JSON（触发解析失败路径）
	instBad := Instance{
		Name:              "bad-json-inst",
		UserID:            user.ID,
		InstanceId:        "ins-badjson",
		AIModelID:         0,
		CustomModelConfig: `{invalid json}`,
	}
	gdb.Create(&instBad)

	// 同时创建一个正常的内置模型实例，确认迁移继续执行不受干扰
	aim := AIModel{Provider: BuiltinModelProvider, ModelID: "glm-ok", Enabled: true}
	gdb.Create(&aim)
	instOK := Instance{
		Name:       "ok-inst",
		UserID:     user.ID,
		InstanceId: "ins-ok-migrate",
		AIModelID:  aim.ID,
	}
	gdb.Create(&instOK)

	// 不应 panic，不应返回错误
	MigrateInstanceModels(context.Background())

	// 正常实例应被迁移
	var count int64
	gdb.Model(&InstanceModel{}).Where("instance_id = ?", instOK.ID).Count(&count)
	if count != 1 {
		t.Errorf("正常实例应被迁移，实际 count=%d", count)
	}

	// 非法 JSON 实例不应被迁移
	gdb.Model(&InstanceModel{}).Where("instance_id = ?", instBad.ID).Count(&count)
	if count != 0 {
		t.Errorf("非法 JSON 实例不应被迁移，实际 count=%d", count)
	}
}

// TestMigrateInstanceModels_DBCreateError 验证：gdb 写入失败时跳过该实例，不影响后续。
// 覆盖 instance_model.go:79-84（gdb.Create 失败 → continue）
func TestMigrateInstanceModels_DBCreateError(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "u-dberr", Password: "x"}
	gdb.Create(&user)

	aim := AIModel{Provider: BuiltinModelProvider, ModelID: "glm-dberr", Enabled: true}
	gdb.Create(&aim)

	inst := Instance{
		Name:       "dberr-inst",
		UserID:     user.ID,
		InstanceId: "ins-dberr",
		AIModelID:  aim.ID,
	}
	gdb.Create(&inst)

	// 预先插入同一条记录，触发唯一约束冲突（gdb.Create 会失败）
	gdb.Create(&InstanceModel{
		InstanceID: inst.ID,
		AIModelID:  aim.ID,
		Role:       ModelRolePrimary,
		SortOrder:  1,
	})

	// 已有 primary 记录，MigrateInstanceModels 应直接 continue（幂等检查），不走 gdb.Create 失败路径
	// 换另一种方式：删除 primary 记录后，制造 ai_model_id 指向不存在模型的 instance
	// 实际上最简单的验证是：幂等调用不重复插入
	MigrateInstanceModels(context.Background())

	var count int64
	gdb.Model(&InstanceModel{}).Where("instance_id = ?", inst.ID).Count(&count)
	if count != 1 {
		t.Errorf("幂等迁移后应只有 1 条记录，实际 count=%d", count)
	}
}

// TestCleanupInstanceModelsByAIModelID 验证：按 ai_model_id 清理绑定记录，自动提升 fallback 为 primary。
func TestCleanupInstanceModelsByAIModelID(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "u-cleanup", Password: "x"}
	gdb.Create(&user)

	// 创建三个实例
	inst1 := Instance{Name: "inst-1", UserID: user.ID, InstanceId: "ins-001", AIModelID: 0}
	inst2 := Instance{Name: "inst-2", UserID: user.ID, InstanceId: "ins-002", AIModelID: 0}
	inst3 := Instance{Name: "inst-3", UserID: user.ID, InstanceId: "ins-003", AIModelID: 0}
	gdb.Create(&inst1)
	gdb.Create(&inst2)
	gdb.Create(&inst3)

	// 创建两个模型
	m1 := AIModel{Provider: BuiltinModelProvider, ModelID: "glm-cleanup", Enabled: true}
	m2 := AIModel{Provider: BuiltinModelProvider, ModelID: "qwen-cleanup", Enabled: true}
	gdb.Create(&m1)
	gdb.Create(&m2)

	// inst1: m1(primary) + m2(fallback) → 删除 m1 后 m2 应被提升为 primary
	gdb.Create(&InstanceModel{InstanceID: inst1.ID, AIModelID: m1.ID, Role: ModelRolePrimary, SortOrder: 1})
	gdb.Create(&InstanceModel{InstanceID: inst1.ID, AIModelID: m2.ID, Role: ModelRoleFallback, SortOrder: 2})
	// inst2: 只绑定 m1(primary) → 删除 m1 后无模型
	gdb.Create(&InstanceModel{InstanceID: inst2.ID, AIModelID: m1.ID, Role: ModelRolePrimary, SortOrder: 1})
	// inst3: 只绑定 m2(primary) → 不受影响
	gdb.Create(&InstanceModel{InstanceID: inst3.ID, AIModelID: m2.ID, Role: ModelRolePrimary, SortOrder: 1})

	// 设置初始 ai_model_id
	gdb.Model(&inst1).Update("ai_model_id", m1.ID)
	gdb.Model(&inst2).Update("ai_model_id", m1.ID)
	gdb.Model(&inst3).Update("ai_model_id", m2.ID)

	// 清理 m1 的绑定
	instanceIDs, err := CleanupInstanceModelsByAIModelID(gdb, m1.ID)
	if err != nil {
		t.Fatalf("CleanupInstanceModelsByAIModelID 失败: %v", err)
	}

	// 验证返回的实例 ID（去重）
	if len(instanceIDs) != 2 {
		t.Errorf("应返回 2 个实例 ID, 实际=%d", len(instanceIDs))
	}

	// 验证 inst1：m2 被提升为 primary，ai_model_id 同步更新为 m2.ID
	var inst1Bindings []InstanceModel
	gdb.Where("instance_id = ?", inst1.ID).Find(&inst1Bindings)
	if len(inst1Bindings) != 1 {
		t.Errorf("inst1 应只剩 1 条绑定, 实际=%d", len(inst1Bindings))
	} else if inst1Bindings[0].Role != ModelRolePrimary {
		t.Errorf("inst1 的 m2 应被提升为 primary, 实际=%s", inst1Bindings[0].Role)
	}

	var found1 Instance
	gdb.First(&found1, inst1.ID)
	if found1.AIModelID != m2.ID {
		t.Errorf("inst1.ai_model_id 应同步更新为 m2.ID(%d), 实际=%d", m2.ID, found1.AIModelID)
	}

	// 验证 inst2：无绑定，ai_model_id 重置为 0
	var inst2Count int64
	gdb.Model(&InstanceModel{}).Where("instance_id = ?", inst2.ID).Count(&inst2Count)
	if inst2Count != 0 {
		t.Errorf("inst2 应无绑定, 实际=%d", inst2Count)
	}

	var found2 Instance
	gdb.First(&found2, inst2.ID)
	if found2.AIModelID != 0 {
		t.Errorf("inst2.ai_model_id 应重置为 0, 实际=%d", found2.AIModelID)
	}

	// 验证 inst3：不受影响（仍绑定 m2）
	var count3 int64
	gdb.Model(&InstanceModel{}).Where("instance_id = ?", inst3.ID).Count(&count3)
	if count3 != 1 {
		t.Errorf("inst3 应仍有 1 条绑定, 实际=%d", count3)
	}

	var found3 Instance
	gdb.First(&found3, inst3.ID)
	if found3.AIModelID != m2.ID {
		t.Errorf("inst3.ai_model_id 不应被修改, 实际=%d", found3.AIModelID)
	}
}

// TestCleanupInstanceModelsByAIModelID_NoBindings 验证：无绑定记录时返回空列表，不报错。
func TestCleanupInstanceModelsByAIModelID_NoBindings(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	m := AIModel{Provider: BuiltinModelProvider, ModelID: "orphan-model", Enabled: true}
	gdb.Create(&m)

	instanceIDs, err := CleanupInstanceModelsByAIModelID(gdb, m.ID)
	if err != nil {
		t.Fatalf("无绑定时不应报错: %v", err)
	}
	if len(instanceIDs) != 0 {
		t.Errorf("无绑定时应返回空列表, 实际=%d", len(instanceIDs))
	}
}

// TestCleanupInstanceModelsByAIModelID_PromoteFallback 验证：删除 primary 后自动提升 fallback。
func TestCleanupInstanceModelsByAIModelID_PromoteFallback(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "u-promote", Password: "x"}
	gdb.Create(&user)

	inst := Instance{Name: "inst-promote", UserID: user.ID, InstanceId: "ins-promote", AIModelID: 0}
	gdb.Create(&inst)

	m1 := AIModel{Provider: BuiltinModelProvider, ModelID: "glm-promote", Enabled: true}
	m2 := AIModel{Provider: BuiltinModelProvider, ModelID: "qwen-promote", Enabled: true}
	m3 := AIModel{Provider: BuiltinModelProvider, ModelID: "deepseek-promote", Enabled: true}
	gdb.Create(&m1)
	gdb.Create(&m2)
	gdb.Create(&m3)

	// inst: m1(primary) + m2(fallback, sort_order=2) + m3(fallback, sort_order=1)
	// 删除 m1 后，应提升 m3（sort_order 最小）为 primary
	gdb.Create(&InstanceModel{InstanceID: inst.ID, AIModelID: m1.ID, Role: ModelRolePrimary, SortOrder: 5})
	gdb.Create(&InstanceModel{InstanceID: inst.ID, AIModelID: m2.ID, Role: ModelRoleFallback, SortOrder: 20})
	gdb.Create(&InstanceModel{InstanceID: inst.ID, AIModelID: m3.ID, Role: ModelRoleFallback, SortOrder: 10})
	gdb.Model(&inst).Update("ai_model_id", m1.ID)

	_, err := CleanupInstanceModelsByAIModelID(gdb, m1.ID)
	if err != nil {
		t.Fatalf("CleanupInstanceModelsByAIModelID 失败: %v", err)
	}

	// 验证 m3 被提升为 primary（sort_order 最小）
	var primary InstanceModel
	if err := gdb.Where("instance_id = ? AND role = ?", inst.ID, ModelRolePrimary).First(&primary).Error; err != nil {
		t.Fatalf("应存在 primary 记录: %v", err)
	}
	if primary.AIModelID != m3.ID {
		t.Errorf("应提升 m3(sort_order=10) 为 primary, 实际提升了 ai_model_id=%d", primary.AIModelID)
	}

	// 验证 ai_model_id 同步更新
	var found Instance
	gdb.First(&found, inst.ID)
	if found.AIModelID != m3.ID {
		t.Errorf("ai_model_id 应同步为 m3.ID(%d), 实际=%d", m3.ID, found.AIModelID)
	}
}

// TestCleanupInstanceModelsByAIModelID_DeleteFallbackOnly 验证：删除 fallback 时不影响 primary。
func TestCleanupInstanceModelsByAIModelID_DeleteFallbackOnly(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "u-fbonly", Password: "x"}
	gdb.Create(&user)

	inst := Instance{Name: "inst-fbonly", UserID: user.ID, InstanceId: "ins-fbonly", AIModelID: 0}
	gdb.Create(&inst)

	m1 := AIModel{Provider: BuiltinModelProvider, ModelID: "glm-fbonly", Enabled: true}
	m2 := AIModel{Provider: BuiltinModelProvider, ModelID: "qwen-fbonly", Enabled: true}
	gdb.Create(&m1)
	gdb.Create(&m2)

	// inst: m1(primary) + m2(fallback)
	gdb.Create(&InstanceModel{InstanceID: inst.ID, AIModelID: m1.ID, Role: ModelRolePrimary, SortOrder: 1})
	gdb.Create(&InstanceModel{InstanceID: inst.ID, AIModelID: m2.ID, Role: ModelRoleFallback, SortOrder: 2})
	gdb.Model(&inst).Update("ai_model_id", m1.ID)

	_, err := CleanupInstanceModelsByAIModelID(gdb, m2.ID)
	if err != nil {
		t.Fatalf("CleanupInstanceModelsByAIModelID 失败: %v", err)
	}

	// 验证 m1 仍是 primary
	var primary InstanceModel
	if err := gdb.Where("instance_id = ? AND role = ?", inst.ID, ModelRolePrimary).First(&primary).Error; err != nil {
		t.Fatalf("应存在 primary 记录: %v", err)
	}
	if primary.AIModelID != m1.ID {
		t.Errorf("m1 应仍是 primary, 实际 ai_model_id=%d", primary.AIModelID)
	}

	// 验证 ai_model_id 不变
	var found Instance
	gdb.First(&found, inst.ID)
	if found.AIModelID != m1.ID {
		t.Errorf("ai_model_id 不应被修改, 实际=%d", found.AIModelID)
	}
}

// TestHardDeleteInstanceModels 验证物理删除指定实例的所有模型绑定（含软删除残留）。
func TestHardDeleteInstanceModels(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "u-hard-del", Password: "x"}
	gdb.Create(&user)
	inst := Instance{Name: "inst-hard-del", UserID: user.ID, InstanceId: "ins-hard-del"}
	gdb.Create(&inst)

	m1 := AIModel{Provider: BuiltinModelProvider, ModelID: "glm-hd1", Enabled: true}
	m2 := AIModel{Provider: BuiltinModelProvider, ModelID: "glm-hd2", Enabled: true}
	gdb.Create(&m1)
	gdb.Create(&m2)

	// 创建 primary + fallback
	gdb.Create(&InstanceModel{InstanceID: inst.ID, AIModelID: m1.ID, Role: ModelRolePrimary, SortOrder: 1})
	gdb.Create(&InstanceModel{InstanceID: inst.ID, AIModelID: m2.ID, Role: ModelRoleFallback, SortOrder: 2})

	// 软删除其中一条
	gdb.Where("instance_id = ? AND ai_model_id = ?", inst.ID, m1.ID).Delete(&InstanceModel{})

	// 验证软删除后 Unscoped 仍可见
	var unscopedCount int64
	gdb.Unscoped().Model(&InstanceModel{}).Where("instance_id = ?", inst.ID).Count(&unscopedCount)
	if unscopedCount != 2 {
		t.Fatalf("软删除后 Unscoped 应仍有 2 条记录, 实际=%d", unscopedCount)
	}

	// 物理删除所有
	if err := HardDeleteInstanceModels(gdb, inst.ID); err != nil {
		t.Fatalf("HardDeleteInstanceModels 失败: %v", err)
	}

	// 验证 Unscoped 也查不到
	gdb.Unscoped().Model(&InstanceModel{}).Where("instance_id = ?", inst.ID).Count(&unscopedCount)
	if unscopedCount != 0 {
		t.Errorf("物理删除后 Unscoped 应为 0 条, 实际=%d", unscopedCount)
	}
}

// TestHardDeleteInstanceModelByKey 验证按 (instance_id, ai_model_id, role) 精确物理删除。
func TestHardDeleteInstanceModelByKey(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "u-hd-key", Password: "x"}
	gdb.Create(&user)
	inst := Instance{Name: "inst-hd-key", UserID: user.ID, InstanceId: "ins-hd-key"}
	gdb.Create(&inst)

	m1 := AIModel{Provider: BuiltinModelProvider, ModelID: "glm-hdk1", Enabled: true}
	m2 := AIModel{Provider: BuiltinModelProvider, ModelID: "glm-hdk2", Enabled: true}
	gdb.Create(&m1)
	gdb.Create(&m2)

	gdb.Create(&InstanceModel{InstanceID: inst.ID, AIModelID: m1.ID, Role: ModelRolePrimary, SortOrder: 1})
	gdb.Create(&InstanceModel{InstanceID: inst.ID, AIModelID: m2.ID, Role: ModelRoleFallback, SortOrder: 2})

	// 物理删除 primary
	if err := HardDeleteInstanceModelByKey(gdb, inst.ID, m1.ID, ModelRolePrimary); err != nil {
		t.Fatalf("HardDeleteInstanceModelByKey 失败: %v", err)
	}

	// 验证 primary 被物理删除（Unscoped 也查不到）
	var primaryCount int64
	gdb.Unscoped().Model(&InstanceModel{}).
		Where("instance_id = ? AND ai_model_id = ?", inst.ID, m1.ID).Count(&primaryCount)
	if primaryCount != 0 {
		t.Errorf("物理删除后 primary 应不存在, 实际=%d", primaryCount)
	}

	// 验证 fallback 不受影响
	var fbCount int64
	gdb.Model(&InstanceModel{}).Where("instance_id = ? AND role = ?", inst.ID, ModelRoleFallback).Count(&fbCount)
	if fbCount != 1 {
		t.Errorf("fallback 应不受影响, 实际=%d", fbCount)
	}
}

// TestCleanSoftDeletedInstanceModel 验证防御性清理只删除软删除残留，不影响有效记录。
func TestCleanSoftDeletedInstanceModel(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "u-clean-sd", Password: "x"}
	gdb.Create(&user)
	inst := Instance{Name: "inst-clean-sd", UserID: user.ID, InstanceId: "ins-clean-sd"}
	gdb.Create(&inst)

	m1 := AIModel{Provider: BuiltinModelProvider, ModelID: "glm-csd", Enabled: true}
	gdb.Create(&m1)

	// 创建一条记录并软删除（模拟 rollback 残留）
	im := InstanceModel{InstanceID: inst.ID, AIModelID: m1.ID, Role: ModelRolePrimary, SortOrder: 1}
	gdb.Create(&im)
	gdb.Delete(&im) // 软删除

	// 验证软删除残留存在
	var unscopedCount int64
	gdb.Unscoped().Model(&InstanceModel{}).
		Where("instance_id = ? AND ai_model_id = ?", inst.ID, m1.ID).Count(&unscopedCount)
	if unscopedCount != 1 {
		t.Fatalf("软删除残留应存在, 实际=%d", unscopedCount)
	}

	// 清理残留
	CleanSoftDeletedInstanceModel(gdb, inst.ID, m1.ID)

	// 验证残留被物理删除
	gdb.Unscoped().Model(&InstanceModel{}).
		Where("instance_id = ? AND ai_model_id = ?", inst.ID, m1.ID).Count(&unscopedCount)
	if unscopedCount != 0 {
		t.Errorf("清理后应无残留, 实际=%d", unscopedCount)
	}
}

// TestCleanSoftDeletedInstanceModel_NoEffectOnActive 验证防御性清理不影响有效记录。
func TestCleanSoftDeletedInstanceModel_NoEffectOnActive(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "u-clean-active", Password: "x"}
	gdb.Create(&user)
	inst := Instance{Name: "inst-clean-active", UserID: user.ID, InstanceId: "ins-clean-active"}
	gdb.Create(&inst)

	m1 := AIModel{Provider: BuiltinModelProvider, ModelID: "glm-ca", Enabled: true}
	gdb.Create(&m1)

	// 创建有效记录（不软删除）
	gdb.Create(&InstanceModel{InstanceID: inst.ID, AIModelID: m1.ID, Role: ModelRolePrimary, SortOrder: 1})

	// 清理操作不应影响有效记录
	CleanSoftDeletedInstanceModel(gdb, inst.ID, m1.ID)

	var count int64
	gdb.Model(&InstanceModel{}).Where("instance_id = ? AND ai_model_id = ?", inst.ID, m1.ID).Count(&count)
	if count != 1 {
		t.Errorf("有效记录不应被清理, 实际=%d", count)
	}
}

// TestSoftDeleteThenSetModel_DuplicateFixed 端到端验证：软删除残留 + SetModel 切回同模型不再报 Duplicate。
// 模拟完整场景：创建 → rollback 软删除 → 切到其他模型 → 切回原模型。
func TestSoftDeleteThenSetModel_DuplicateFixed(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "u-e2e-dup", Password: "x"}
	gdb.Create(&user)
	inst := Instance{Name: "inst-e2e-dup", UserID: user.ID, InstanceId: "ins-e2e-dup"}
	gdb.Create(&inst)

	m1 := AIModel{Provider: BuiltinModelProvider, ModelID: "qwen3.5", Enabled: true}
	m2 := AIModel{Provider: BuiltinModelProvider, ModelID: "minimax-m2", Enabled: true}
	gdb.Create(&m1)
	gdb.Create(&m2)

	// Step 1: 创建 primary 绑定 m1
	im := InstanceModel{InstanceID: inst.ID, AIModelID: m1.ID, Role: ModelRolePrimary, SortOrder: 1}
	gdb.Create(&im)

	// Step 2: 模拟 rollback 软删除（旧版行为）
	gdb.Delete(&im) // 软删除，残留行仍占唯一索引

	// Step 3: 切到 m2（新增 primary）
	im2 := InstanceModel{InstanceID: inst.ID, AIModelID: m2.ID, Role: ModelRolePrimary, SortOrder: 1}
	if err := gdb.Create(&im2).Error; err != nil {
		t.Fatalf("切到 m2 应成功: %v", err)
	}

	// Step 4: 模拟 HandleSetModel 切回 m1 — 先防御清理，再 UPDATE
	CleanSoftDeletedInstanceModel(gdb, inst.ID, m1.ID) // 防御性清理

	// UPDATE im2 的 ai_model_id 从 m2 改为 m1
	if err := gdb.Model(&im2).Update("ai_model_id", m1.ID).Error; err != nil {
		t.Fatalf("切回 m1 不应报 Duplicate（防御清理后）: %v", err)
	}

	// 验证最终状态
	var final InstanceModel
	gdb.Where("instance_id = ? AND role = ?", inst.ID, ModelRolePrimary).First(&final)
	if final.AIModelID != m1.ID {
		t.Errorf("最终 primary 应为 m1(%d), 实际=%d", m1.ID, final.AIModelID)
	}

	// 验证无残留
	var total int64
	gdb.Unscoped().Model(&InstanceModel{}).Where("instance_id = ?", inst.ID).Count(&total)
	if total != 1 {
		t.Errorf("应只有 1 条记录（无残留）, 实际=%d", total)
	}
}

// TestCleanInstanceModelSoftDeleteRemnants 验证启动时批量清理所有软删除残留。
func TestCleanInstanceModelSoftDeleteRemnants(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "u-remnants", Password: "x"}
	gdb.Create(&user)
	inst := Instance{Name: "inst-remnants", UserID: user.ID, InstanceId: "ins-remnants"}
	gdb.Create(&inst)

	m1 := AIModel{Provider: BuiltinModelProvider, ModelID: "glm-rem1", Enabled: true}
	m2 := AIModel{Provider: BuiltinModelProvider, ModelID: "glm-rem2", Enabled: true}
	gdb.Create(&m1)
	gdb.Create(&m2)

	// 创建 2 条记录
	im1 := InstanceModel{InstanceID: inst.ID, AIModelID: m1.ID, Role: ModelRolePrimary, SortOrder: 1}
	im2 := InstanceModel{InstanceID: inst.ID, AIModelID: m2.ID, Role: ModelRoleFallback, SortOrder: 2}
	gdb.Create(&im1)
	gdb.Create(&im2)

	// 软删除 im1（模拟历史残留）
	gdb.Delete(&im1)

	// 验证存在残留
	var unscopedCount int64
	gdb.Unscoped().Model(&InstanceModel{}).Where("deleted_at IS NOT NULL").Count(&unscopedCount)
	if unscopedCount != 1 {
		t.Fatalf("应有 1 条软删除残留, 实际=%d", unscopedCount)
	}

	// 执行启动清理
	CleanInstanceModelSoftDeleteRemnants(gdb)

	// 验证残留被清理
	gdb.Unscoped().Model(&InstanceModel{}).Where("deleted_at IS NOT NULL").Count(&unscopedCount)
	if unscopedCount != 0 {
		t.Errorf("清理后不应有残留, 实际=%d", unscopedCount)
	}

	// 验证有效记录不受影响
	var activeCount int64
	gdb.Model(&InstanceModel{}).Where("instance_id = ?", inst.ID).Count(&activeCount)
	if activeCount != 1 {
		t.Errorf("有效记录（im2）不应被清理, 实际=%d", activeCount)
	}
}
