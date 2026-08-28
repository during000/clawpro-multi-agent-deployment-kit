package model

import (
	"context"
	"testing"
)

// ============ ResolveModelForRequest 场景覆盖 ============

// 场景 1：请求指定的内置模型（大小写不敏感命中 instance_models 绑定）
func TestResolveModelForRequest_BuiltinCaseInsensitive(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "u1", Password: "x"}
	gdb.Create(&user)
	inst := Instance{Name: "i1", UserID: user.ID, InstanceId: "ins-1", AIModelID: 0}
	gdb.Create(&inst)

	m := AIModel{
		Provider: "hatchery", ModelID: "deepseek-chat", ModelName: "DeepSeek-Chat",
		URL: "https://api.deepseek.com", APIKey: "sk-builtin", ModelType: "openai-completions",
		Enabled: true, QuotaDay: -1,
	}
	gdb.Create(&m)
	gdb.Create(&InstanceModel{InstanceID: inst.ID, AIModelID: m.ID, Role: ModelRolePrimary, SortOrder: 1})

	// 请求名大小写混合，应能命中
	r, err := ResolveModelForRequest(context.Background(), &inst, "DeepSeek-Chat")
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if r.AIModelID != m.ID {
		t.Errorf("AIModelID 期望=%d 实际=%d", m.ID, r.AIModelID)
	}
	if r.ModelID != "deepseek-chat" {
		t.Errorf("ModelID 应透传为 gdb 原值 deepseek-chat, 实际=%q", r.ModelID)
	}
	if r.IsCustom {
		t.Error("内置模型 IsCustom 应为 false")
	}
	if r.UsageBucketKey != m.ID {
		t.Errorf("内置模型 UsageBucketKey 应等于 AIModelID=%d, 实际=%d", m.ID, r.UsageBucketKey)
	}
}

// 场景 2：请求指定的自定义模型（大小写不敏感命中 instance_models 绑定）
func TestResolveModelForRequest_CustomCaseInsensitive(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "u2", Password: "x"}
	gdb.Create(&user)
	inst := Instance{Name: "i2", UserID: user.ID, InstanceId: "ins-2", AIModelID: 0}
	gdb.Create(&inst)

	cfg := `{"provider":"custom-deepseek-v3.1","model_id":"deepseek-v3.1","model_name":"DeepSeek-V3.1","api_key":"sk-custom","url":"https://api.x.com","model_type":"openai-completions"}`
	gdb.Create(&InstanceModel{
		InstanceID: inst.ID, AIModelID: 0, CustomModelID: "deepseek-v3.1",
		CustomModelConfig: cfg, Role: ModelRolePrimary, SortOrder: 1,
	})

	// 客户端传大写 DeepSeek-V3.1
	r, err := ResolveModelForRequest(context.Background(), &inst, "DeepSeek-V3.1")
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if !r.IsCustom {
		t.Error("自定义模型 IsCustom 应为 true")
	}
	if r.AIModelID != 0 {
		t.Errorf("自定义模型 AIModelID 应为 0, 实际=%d", r.AIModelID)
	}
	if r.ModelID != "deepseek-v3.1" {
		t.Errorf("ModelID 应为 deepseek-v3.1, 实际=%q", r.ModelID)
	}
	if r.APIKey != "sk-custom" || r.URL != "https://api.x.com" {
		t.Errorf("APIKey/URL 未正确透传: key=%q url=%q", r.APIKey, r.URL)
	}
	if r.QuotaDay != -1 {
		t.Errorf("自定义模型 QuotaDay 固定 -1, 实际=%d", r.QuotaDay)
	}
	if r.UsageBucketKey != 0 {
		t.Errorf("自定义模型 UsageBucketKey 应为 0, 实际=%d", r.UsageBucketKey)
	}
}

// 场景 3：请求名未指定，回退到 instance.AIModelID 指向的内置 primary
func TestResolveModelForRequest_EmptyFallbackToInstanceBuiltin(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "u3", Password: "x"}
	gdb.Create(&user)
	m := AIModel{
		Provider: "hatchery", ModelID: "glm-4-plus", Enabled: true, QuotaDay: -1,
		URL: "https://api.z.com", APIKey: "sk-b", ModelType: "openai-completions",
	}
	gdb.Create(&m)
	inst := Instance{Name: "i3", UserID: user.ID, InstanceId: "ins-3", AIModelID: m.ID}
	gdb.Create(&inst)
	// 注意：这里故意不建 instance_models，验证"存量实例"兼容路径

	r, err := ResolveModelForRequest(context.Background(), &inst, "")
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if r.AIModelID != m.ID {
		t.Errorf("应回退到 instance.AIModelID=%d, 实际=%d", m.ID, r.AIModelID)
	}
	if r.ModelID != "glm-4-plus" {
		t.Errorf("ModelID 错误: %q", r.ModelID)
	}
}

// 场景 4：请求名未指定，instance.AIModelID=0 但有 CustomModelConfig，回退到自定义
func TestResolveModelForRequest_EmptyFallbackToInstanceCustom(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "u4", Password: "x"}
	gdb.Create(&user)
	cfg := `{"provider":"custom","model_id":"legacy","api_key":"k","url":"https://u.com","model_type":"openai-completions"}`
	inst := Instance{
		Name: "i4", UserID: user.ID, InstanceId: "ins-4",
		AIModelID:         0,
		CustomModelConfig: cfg,
	}
	gdb.Create(&inst)
	// 无 instance_models

	r, err := ResolveModelForRequest(context.Background(), &inst, "")
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if !r.IsCustom || r.ModelID != "legacy" {
		t.Errorf("应回退到 instance.CustomModelConfig, 实际: %+v", r)
	}
}

// 场景 5：内置模型被 disabled，不应被路由到
func TestResolveModelForRequest_DisabledBuiltinSkipped(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "u5", Password: "x"}
	gdb.Create(&user)
	m := AIModel{Provider: "hatchery", ModelID: "retired", Enabled: false}
	gdb.Create(&m)
	inst := Instance{Name: "i5", UserID: user.ID, InstanceId: "ins-5", AIModelID: m.ID}
	gdb.Create(&inst)
	gdb.Create(&InstanceModel{InstanceID: inst.ID, AIModelID: m.ID, Role: ModelRolePrimary, SortOrder: 1})

	_, err := ResolveModelForRequest(context.Background(), &inst, "retired")
	if err == nil {
		t.Error("disabled 模型不应可解析，但 err=nil")
	}
}

// 场景 6：请求名指定了不存在的绑定，且实例也无 primary，应返回 ErrNoResolvableModel
func TestResolveModelForRequest_NotBound(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "u6", Password: "x"}
	gdb.Create(&user)
	inst := Instance{Name: "i6", UserID: user.ID, InstanceId: "ins-6", AIModelID: 0}
	gdb.Create(&inst)

	_, err := ResolveModelForRequest(context.Background(), &inst, "nonexistent-model")
	if err == nil {
		t.Error("不存在的模型应返回错误")
	}
}

// 场景 7：客户端传入未绑定的模型，但实例有 primary —— 当前设计是"找不到就回退到 primary"
// 这里验证此行为，确保 agent 指定某个 fallback 名但绑定表里没有时不会直接 400。
func TestResolveModelForRequest_UnknownModelFallbackToPrimary(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "u7", Password: "x"}
	gdb.Create(&user)
	m := AIModel{Provider: "hatchery", ModelID: "primary-m", Enabled: true, QuotaDay: -1}
	gdb.Create(&m)
	inst := Instance{Name: "i7", UserID: user.ID, InstanceId: "ins-7", AIModelID: m.ID}
	gdb.Create(&inst)
	// instance_models 里只有 primary-m 的绑定
	gdb.Create(&InstanceModel{InstanceID: inst.ID, AIModelID: m.ID, Role: ModelRolePrimary, SortOrder: 1})

	// 客户端传了一个根本不在绑定表里的名字
	r, err := ResolveModelForRequest(context.Background(), &inst, "ghost-model")
	if err != nil {
		t.Fatalf("期望回退到 primary, 但返回错误: %v", err)
	}
	if r.ModelID != "primary-m" {
		t.Errorf("应回退到 primary-m, 实际=%q", r.ModelID)
	}
}

// 场景 8：primary 是自定义模型，fallback 是内置模型。按请求名路由到 fallback
func TestResolveModelForRequest_MixedPrimaryCustomFallbackBuiltin(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "u8", Password: "x"}
	gdb.Create(&user)

	cfg := `{"provider":"custom-ds","model_id":"deepseek-v3.1","api_key":"k","url":"https://u.com","model_type":"openai-completions"}`
	inst := Instance{
		Name: "i8", UserID: user.ID, InstanceId: "ins-8",
		AIModelID: 0, CustomModelConfig: cfg,
	}
	gdb.Create(&inst)

	builtin := AIModel{Provider: "hatchery", ModelID: "deepseek-chat", Enabled: true, QuotaDay: -1,
		URL: "https://api.deepseek.com", APIKey: "sk-b", ModelType: "openai-completions"}
	gdb.Create(&builtin)

	gdb.Create(&InstanceModel{InstanceID: inst.ID, AIModelID: 0, CustomModelID: "deepseek-v3.1",
		CustomModelConfig: cfg, Role: ModelRolePrimary, SortOrder: 1})
	gdb.Create(&InstanceModel{InstanceID: inst.ID, AIModelID: builtin.ID,
		Role: ModelRoleFallback, SortOrder: 2})

	// 请求名 = 内置模型 → 走内置
	r, err := ResolveModelForRequest(context.Background(), &inst, "deepseek-chat")
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if r.IsCustom || r.AIModelID != builtin.ID {
		t.Errorf("应命中内置 fallback, 实际: %+v", r)
	}

	// 请求名 = 自定义模型 → 走自定义
	r2, err := ResolveModelForRequest(context.Background(), &inst, "deepseek-v3.1")
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if !r2.IsCustom || r2.ModelID != "deepseek-v3.1" {
		t.Errorf("应命中自定义 primary, 实际: %+v", r2)
	}
}

// 场景 9：nil 实例安全
func TestResolveModelForRequest_NilInstance(t *testing.T) {
	_, err := ResolveModelForRequest(context.Background(), nil, "any")
	if err == nil {
		t.Error("nil instance 应返回错误")
	}
}

// ============ ListInstanceModels 场景覆盖 ============

// 验证 ListInstanceModels 返回所有绑定，primary 在前，fallback 按 sort_order 排序
func TestListInstanceModels_OrderAndDedup(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "u-list", Password: "x"}
	gdb.Create(&user)
	inst := Instance{Name: "i-list", UserID: user.ID, InstanceId: "ins-list", AIModelID: 0}
	gdb.Create(&inst)

	primary := AIModel{Provider: "hatchery", ModelID: "p-model", Enabled: true}
	fb := AIModel{Provider: "hatchery", ModelID: "fb-model", Enabled: true}
	gdb.Create(&primary)
	gdb.Create(&fb)

	gdb.Create(&InstanceModel{InstanceID: inst.ID, AIModelID: fb.ID, Role: ModelRoleFallback, SortOrder: 10})
	gdb.Create(&InstanceModel{InstanceID: inst.ID, AIModelID: primary.ID, Role: ModelRolePrimary, SortOrder: 5})

	list := ListInstanceModels(context.Background(), inst.ID)
	if len(list) != 2 {
		t.Fatalf("应返回 2 个绑定, 实际=%d", len(list))
	}
	if list[0].ModelID != "p-model" {
		t.Errorf("primary 应排第一, 实际=%q", list[0].ModelID)
	}
	if list[1].ModelID != "fb-model" {
		t.Errorf("fallback 应排第二, 实际=%q", list[1].ModelID)
	}
}

// 场景 12：instance 主字段为空，但 instance_models 表有 primary 绑定（OpenClaw gateway 场景）
func TestResolveModelForRequest_PrimaryFromBindings(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "u-primary-binding", Password: "x"}
	gdb.Create(&user)
	// 注意：AIModelID=0 且 CustomModelConfig=""，模拟 gateway 配置方式
	inst := Instance{Name: "i-primary-binding", UserID: user.ID, InstanceId: "ins-primary-binding", AIModelID: 0, CustomModelConfig: ""}
	gdb.Create(&inst)

	m := AIModel{
		Provider: "hatchery", ModelID: "deepseek-chat", ModelName: "DeepSeek-Chat",
		URL: "https://api.deepseek.com", APIKey: "sk-binding", ModelType: "openai-completions",
		Enabled: true, QuotaDay: 100,
	}
	gdb.Create(&m)
	gdb.Create(&InstanceModel{InstanceID: inst.ID, AIModelID: m.ID, Role: ModelRolePrimary, SortOrder: 1})

	// reqModelName 为空，应回退到 instance_models 表的 primary 绑定
	r, err := ResolveModelForRequest(context.Background(), &inst, "")
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if r.AIModelID != m.ID {
		t.Errorf("AIModelID 期望=%d 实际=%d", m.ID, r.AIModelID)
	}
	if r.ModelID != "deepseek-chat" {
		t.Errorf("ModelID 期望=deepseek-chat 实际=%q", r.ModelID)
	}
	if r.IsCustom {
		t.Error("内置模型 IsCustom 应为 false")
	}
}

// 场景 13：instance 主字段为空，instance_models 表有 primary 自定义模型绑定
func TestResolveModelForRequest_CustomPrimaryFromBindings(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "u-custom-primary", Password: "x"}
	gdb.Create(&user)
	inst := Instance{Name: "i-custom-primary", UserID: user.ID, InstanceId: "ins-custom-primary", AIModelID: 0, CustomModelConfig: ""}
	gdb.Create(&inst)

	cfgJSON := `{"provider":"custom","model_id":"deepseek-v3.1","model_name":"DeepSeek-V3.1","api_key":"sk-custom","url":"https://api.custom.com","model_type":"openai-completions"}`
	gdb.Create(&InstanceModel{
		InstanceID:        inst.ID,
		AIModelID:         0,
		CustomModelID:     "deepseek-v3.1",
		CustomModelConfig: cfgJSON,
		Role:              ModelRolePrimary,
		SortOrder:         1,
	})

	// reqModelName 为空，应回退到 instance_models 表的 primary 自定义绑定
	r, err := ResolveModelForRequest(context.Background(), &inst, "")
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if !r.IsCustom {
		t.Error("自定义模型 IsCustom 应为 true")
	}
	if r.ModelID != "deepseek-v3.1" {
		t.Errorf("ModelID 期望=deepseek-v3.1 实际=%q", r.ModelID)
	}
	if r.Provider != "custom" {
		t.Errorf("Provider 期望=custom 实际=%q", r.Provider)
	}
}

// 禁用的内置模型不应出现在列表里
func TestListInstanceModels_DisabledBuiltinExcluded(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "u-list2", Password: "x"}
	gdb.Create(&user)
	inst := Instance{Name: "i-list2", UserID: user.ID, InstanceId: "ins-list2", AIModelID: 0}
	gdb.Create(&inst)

	disabled := AIModel{Provider: "hatchery", ModelID: "dead-model", Enabled: false}
	gdb.Create(&disabled)
	gdb.Create(&InstanceModel{InstanceID: inst.ID, AIModelID: disabled.ID, Role: ModelRolePrimary, SortOrder: 1})

	list := ListInstanceModels(context.Background(), inst.ID)
	if len(list) != 0 {
		t.Errorf("禁用模型应被过滤, 实际返回=%d 条", len(list))
	}
}

// ============ slug 化兼容匹配测试 ============

// TestResolveModelForRequest_BuiltinSlugCompat 验证：
// 请求的 model name 是 slug 化后的值（'/' → '-'），DB 中 model_id 含 '/'，
// 精确匹配失败后通过 slug 化兼容匹配命中。
func TestResolveModelForRequest_BuiltinSlugCompat(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "slug-u1", Password: "x"}
	DB(context.Background()).Create(&user)
	inst := Instance{Name: "slug-i1", UserID: user.ID, InstanceId: "ins-slug1", AIModelID: 0}
	DB(context.Background()).Create(&inst)

	// DB 中 model_id 含 '/'
	m := AIModel{
		Provider: "自定义模型", ModelID: "ZhiJia/GLM-5.1-Plus",
		URL: "https://api.example.com", APIKey: "sk-test", ModelType: "openai-completions",
		Enabled: true, QuotaDay: -1,
	}
	DB(context.Background()).Create(&m)
	DB(context.Background()).Create(&InstanceModel{InstanceID: inst.ID, AIModelID: m.ID, Role: ModelRolePrimary, SortOrder: 1})

	// 请求用 slug 化的名字（'/' → '-'，全小写）—— 精确匹配失败，slug 兼容匹配命中
	r, err := ResolveModelForRequest(context.Background(), &inst, "zhijia-glm-5.1-plus")
	if err != nil {
		t.Fatalf("slug 兼容匹配应成功, 错误: %v", err)
	}
	if r.AIModelID != m.ID {
		t.Errorf("AIModelID 期望=%d 实际=%d", m.ID, r.AIModelID)
	}
	if r.ModelID != "ZhiJia/GLM-5.1-Plus" {
		t.Errorf("ModelID 应透传 DB 原值, 实际=%q", r.ModelID)
	}
}

// TestResolveModelForRequest_CustomSlugCompat 验证：
// 自定义模型的 slug 化兼容匹配（custom_model_id 含 '/'）。
func TestResolveModelForRequest_CustomSlugCompat(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "slug-u2", Password: "x"}
	DB(context.Background()).Create(&user)
	inst := Instance{Name: "slug-i2", UserID: user.ID, InstanceId: "ins-slug2", AIModelID: 0}
	DB(context.Background()).Create(&inst)

	// 自定义模型 custom_model_id 含 '/'
	customCfg := `{"provider":"自定义模型","model_id":"openrouter/auto","api_key":"sk-test","url":"https://openrouter.ai/api/v1","model_type":"openai-completions","context_len":128000}`
	DB(context.Background()).Create(&InstanceModel{
		InstanceID:        inst.ID,
		AIModelID:         0,
		CustomModelID:     "openrouter/auto",
		CustomModelConfig: customCfg,
		Role:              ModelRolePrimary,
		SortOrder:         1,
	})

	// 请求用 slug 化名字（'/' → '-'）
	r, err := ResolveModelForRequest(context.Background(), &inst, "openrouter-auto")
	if err != nil {
		t.Fatalf("自定义模型 slug 兼容匹配应成功, 错误: %v", err)
	}
	if r.CustomModelID != "openrouter/auto" {
		t.Errorf("CustomModelID 应透传 DB 原值, 实际=%q", r.CustomModelID)
	}
	if r.ModelID != "openrouter/auto" {
		t.Errorf("ModelID 应为 openrouter/auto, 实际=%q", r.ModelID)
	}
	if !r.IsCustom {
		t.Error("自定义模型 IsCustom 应为 true")
	}
}

// TestResolveModelForRequest_BuiltinSlugCompat_Colon 验证：
// 内置模型的 slug 化兼容匹配（model_id 含 ':'）。
func TestResolveModelForRequest_BuiltinSlugCompat_Colon(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "slug-colon-u1", Password: "x"}
	DB(context.Background()).Create(&user)
	inst := Instance{Name: "slug-colon-i1", UserID: user.ID, InstanceId: "ins-slug-colon1", AIModelID: 0}
	DB(context.Background()).Create(&inst)

	// DB 中 model_id 含 ':'
	m := AIModel{
		Provider: "hatchery", ModelID: "minimax-m2.5:free",
		URL: "https://api.example.com", APIKey: "sk-test", ModelType: "openai-completions",
		Enabled: true, QuotaDay: -1,
	}
	DB(context.Background()).Create(&m)
	DB(context.Background()).Create(&InstanceModel{InstanceID: inst.ID, AIModelID: m.ID, Role: ModelRolePrimary, SortOrder: 1})

	// 请求用 slug 化的名字（':' → '-'，全小写）—— 精确匹配失败，slug 兼容匹配命中
	r, err := ResolveModelForRequest(context.Background(), &inst, "minimax-m2.5-free")
	if err != nil {
		t.Fatalf("含冒号的 slug 兼容匹配应成功, 错误: %v", err)
	}
	if r.AIModelID != m.ID {
		t.Errorf("AIModelID 期望=%d 实际=%d", m.ID, r.AIModelID)
	}
	if r.ModelID != "minimax-m2.5:free" {
		t.Errorf("ModelID 应透传 DB 原值, 实际=%q", r.ModelID)
	}
}

// TestResolveModelForRequest_BuiltinSlugCompat_SlashAndColon 验证：
// 内置模型的 slug 化兼容匹配（model_id 同时含 '/' 和 ':'）。
func TestResolveModelForRequest_BuiltinSlugCompat_SlashAndColon(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "slug-both-u1", Password: "x"}
	DB(context.Background()).Create(&user)
	inst := Instance{Name: "slug-both-i1", UserID: user.ID, InstanceId: "ins-slug-both1", AIModelID: 0}
	DB(context.Background()).Create(&inst)

	// DB 中 model_id 同时含 '/' 和 ':'
	m := AIModel{
		Provider: "自定义模型", ModelID: "minimax/minimax-m2.5:free",
		URL: "https://api.minimax.chat/v1", APIKey: "sk-test", ModelType: "openai-completions",
		Enabled: true, QuotaDay: -1,
	}
	DB(context.Background()).Create(&m)
	DB(context.Background()).Create(&InstanceModel{InstanceID: inst.ID, AIModelID: m.ID, Role: ModelRolePrimary, SortOrder: 1})

	// 请求用 slug 化的名字（'/' → '-'，':' → '-'，全小写）
	r, err := ResolveModelForRequest(context.Background(), &inst, "minimax-minimax-m2.5-free")
	if err != nil {
		t.Fatalf("含斜杠和冒号的 slug 兼容匹配应成功, 错误: %v", err)
	}
	if r.AIModelID != m.ID {
		t.Errorf("AIModelID 期望=%d 实际=%d", m.ID, r.AIModelID)
	}
	if r.ModelID != "minimax/minimax-m2.5:free" {
		t.Errorf("ModelID 应透传 DB 原值, 实际=%q", r.ModelID)
	}
}

// TestResolveModelForRequest_CustomSlugCompat_Colon 验证：
// 自定义模型的 slug 化兼容匹配（custom_model_id 含 ':'）。
func TestResolveModelForRequest_CustomSlugCompat_Colon(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "slug-colon-u2", Password: "x"}
	DB(context.Background()).Create(&user)
	inst := Instance{Name: "slug-colon-i2", UserID: user.ID, InstanceId: "ins-slug-colon2", AIModelID: 0}
	DB(context.Background()).Create(&inst)

	// 自定义模型 custom_model_id 含 ':'
	customCfg := `{"provider":"自定义模型","model_id":"minimax/minimax-m2.5:free","api_key":"sk-test","url":"https://api.minimax.chat/v1","model_type":"openai-completions","context_len":128000}`
	DB(context.Background()).Create(&InstanceModel{
		InstanceID:        inst.ID,
		AIModelID:         0,
		CustomModelID:     "minimax/minimax-m2.5:free",
		CustomModelConfig: customCfg,
		Role:              ModelRolePrimary,
		SortOrder:         1,
	})

	// 请求用 slug 化名字（'/' → '-'，':' → '-'）
	r, err := ResolveModelForRequest(context.Background(), &inst, "minimax-minimax-m2.5-free")
	if err != nil {
		t.Fatalf("自定义模型含冒号的 slug 兼容匹配应成功, 错误: %v", err)
	}
	if r.CustomModelID != "minimax/minimax-m2.5:free" {
		t.Errorf("CustomModelID 应透传 DB 原值, 实际=%q", r.CustomModelID)
	}
	if r.ModelID != "minimax/minimax-m2.5:free" {
		t.Errorf("ModelID 应为 minimax/minimax-m2.5:free, 实际=%q", r.ModelID)
	}
	if !r.IsCustom {
		t.Error("自定义模型 IsCustom 应为 true")
	}
}

// ─── CustomHTTPHeaders 透传测试 ───────────────────────────────────────────

// TestResolveModelForRequest_BuiltinWithCustomHTTPHeaders 验证：
// 内置模型的 CustomHTTPHeaders 能正确透传到 ResolvedModel。
func TestResolveModelForRequest_BuiltinWithCustomHTTPHeaders(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "u-headers-1", Password: "x"}
	DB(context.Background()).Create(&user)
	inst := Instance{Name: "i-headers-1", UserID: user.ID, InstanceId: "ins-headers1", AIModelID: 0}
	DB(context.Background()).Create(&inst)

	m := AIModel{
		Provider: "hatchery", ModelID: "deepseek-chat", ModelName: "DeepSeek-Chat",
		URL: "https://api.deepseek.com", APIKey: "sk-builtin", ModelType: "openai-completions",
		Enabled: true, QuotaDay: -1,
		CustomHTTPHeaders: `{"X-Api-Key":"sk-123","X-Request-Id":"req-1"}`,
	}
	DB(context.Background()).Create(&m)
	DB(context.Background()).Create(&InstanceModel{InstanceID: inst.ID, AIModelID: m.ID, Role: ModelRolePrimary, SortOrder: 1})

	r, err := ResolveModelForRequest(context.Background(), &inst, "deepseek-chat")
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if r.CustomHTTPHeaders == nil {
		t.Fatal("CustomHTTPHeaders 不应为 nil")
	}
	if r.CustomHTTPHeaders["X-Api-Key"] != "sk-123" {
		t.Errorf("X-Api-Key = %q, want %q", r.CustomHTTPHeaders["X-Api-Key"], "sk-123")
	}
	if r.CustomHTTPHeaders["X-Request-Id"] != "req-1" {
		t.Errorf("X-Request-Id = %q, want %q", r.CustomHTTPHeaders["X-Request-Id"], "req-1")
	}
}

// TestResolveModelForRequest_BuiltinWithoutCustomHTTPHeaders 验证：
// 内置模型无 CustomHTTPHeaders 时 ResolvedModel.CustomHTTPHeaders 为 nil。
func TestResolveModelForRequest_BuiltinWithoutCustomHTTPHeaders(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "u-headers-2", Password: "x"}
	DB(context.Background()).Create(&user)
	inst := Instance{Name: "i-headers-2", UserID: user.ID, InstanceId: "ins-headers2", AIModelID: 0}
	DB(context.Background()).Create(&inst)

	m := AIModel{
		Provider: "hatchery", ModelID: "glm-4", Enabled: true, QuotaDay: -1,
		URL: "https://api.z.com", APIKey: "sk-b", ModelType: "openai-completions",
		CustomHTTPHeaders: "",
	}
	DB(context.Background()).Create(&m)
	DB(context.Background()).Create(&InstanceModel{InstanceID: inst.ID, AIModelID: m.ID, Role: ModelRolePrimary, SortOrder: 1})

	r, err := ResolveModelForRequest(context.Background(), &inst, "glm-4")
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if r.CustomHTTPHeaders != nil {
		t.Errorf("无 CustomHTTPHeaders 时应为 nil, 实际=%v", r.CustomHTTPHeaders)
	}
}

// TestResolveModelForRequest_CustomWithCustomHTTPHeaders 验证：
// 自定义模型的 CustomHTTPHeaders 能正确透传到 ResolvedModel。
func TestResolveModelForRequest_CustomWithCustomHTTPHeaders(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "u-headers-3", Password: "x"}
	DB(context.Background()).Create(&user)
	inst := Instance{Name: "i-headers-3", UserID: user.ID, InstanceId: "ins-headers3", AIModelID: 0}
	DB(context.Background()).Create(&inst)

	cfgJSON := `{"provider":"custom","model_id":"my-model","model_name":"My Model","api_key":"sk-custom","url":"https://api.custom.com","model_type":"openai-completions","custom_http_headers":{"X-Token":"abc123"}}`
	DB(context.Background()).Create(&InstanceModel{
		InstanceID:        inst.ID,
		AIModelID:         0,
		CustomModelID:     "my-model",
		CustomModelConfig: cfgJSON,
		Role:              ModelRolePrimary,
		SortOrder:         1,
	})

	r, err := ResolveModelForRequest(context.Background(), &inst, "my-model")
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if r.CustomHTTPHeaders == nil {
		t.Fatal("自定义模型 CustomHTTPHeaders 不应为 nil")
	}
	if r.CustomHTTPHeaders["X-Token"] != "abc123" {
		t.Errorf("X-Token = %q, want %q", r.CustomHTTPHeaders["X-Token"], "abc123")
	}
}

// TestResolveModelForRequest_BuiltinWithInvalidCustomHTTPHeaders 验证：
// 内置模型 CustomHTTPHeaders 为无效 JSON 时，GetCustomHTTPHeaders 返回 nil，
// ResolvedModel.CustomHTTPHeaders 应为 nil。
func TestResolveModelForRequest_BuiltinWithInvalidCustomHTTPHeaders(t *testing.T) {
	cleanup := setupInstanceModelTestDB(t)
	defer cleanup()

	user := User{Username: "u-headers-4", Password: "x"}
	DB(context.Background()).Create(&user)
	inst := Instance{Name: "i-headers-4", UserID: user.ID, InstanceId: "ins-headers4", AIModelID: 0}
	DB(context.Background()).Create(&inst)

	m := AIModel{
		Provider: "hatchery", ModelID: "bad-headers", Enabled: true, QuotaDay: -1,
		URL: "https://api.z.com", APIKey: "sk-b", ModelType: "openai-completions",
		CustomHTTPHeaders: "not-valid-json",
	}
	DB(context.Background()).Create(&m)
	DB(context.Background()).Create(&InstanceModel{InstanceID: inst.ID, AIModelID: m.ID, Role: ModelRolePrimary, SortOrder: 1})

	r, err := ResolveModelForRequest(context.Background(), &inst, "bad-headers")
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if r.CustomHTTPHeaders != nil {
		t.Errorf("无效 JSON 的 CustomHTTPHeaders 应为 nil, 实际=%v", r.CustomHTTPHeaders)
	}
}
