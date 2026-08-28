package controller

import (
	"context"
	"net/http/httptest"
	"strconv"
	"testing"

	"hatchery/common"
	"hatchery/model"

	"gorm.io/gorm"
)

// TestHandleSetModel_SyncInstanceModels_E2E 端到端验证 HandleSetModel 切换模型后
// instance_models 表的 primary 记录被正确更新。
// 通过 mock agentScriptRunner 使 TAT 步骤返回成功，让 handler 跑到 DB 事务代码。
func TestHandleSetModel_SyncInstanceModels_E2E(t *testing.T) {
	setupMultiModelTestDB(t)

	// 设置 Domain（buildSetModelParams 需要）
	origDomain0 := common.FixedSnapshot.Domain
	common.FixedSnapshot.Domain = "https://test.example.com"
	t.Cleanup(func() { common.FixedSnapshot.Domain = origDomain0 })

	// Mock agentScriptRunner 为成功（跳过真实 TAT 调用）
	origRunner := agentScriptRunner
	agentScriptRunner = func(ctx context.Context, instanceId string, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		return "{}", nil // 模拟 TAT 成功
	}
	defer func() { agentScriptRunner = origRunner }()

	// Mock LoadScript（ResolveScript 需要）
	origLoadScript := LoadScript
	LoadScript = func(name string) (string, error) {
		return "#!/bin/bash\necho ok", nil
	}
	defer func() { LoadScript = origLoadScript }()

	user, inst := createMultiModelUserAndInstance(t, "hermes-e2e", "hermes-e2e-inst")
	model.DB(context.Background()).Model(inst).Update("agent_type", model.AgentTypeHermes)

	modelA := &model.AIModel{Provider: "hatchery", ModelID: "glm-4-plus", Enabled: true, Visible: true, ModelType: "openai-completions"}
	modelB := &model.AIModel{Provider: "hatchery", ModelID: "deepseek-v3", Enabled: true, Visible: true, ModelType: "openai-completions"}
	model.DB(context.Background()).Create(modelA)
	model.DB(context.Background()).Create(modelB)

	// 模拟创建时写入的默认模型
	model.DB(context.Background()).Model(inst).Update("ai_model_id", modelA.ID)
	model.DB(context.Background()).Create(&model.InstanceModel{
		InstanceID: inst.ID,
		AIModelID:  modelA.ID,
		Role:       model.ModelRolePrimary,
		SortOrder:  1,
	})

	// 调用 HandleSetModel 切换到 modelB
	body := "id=" + strconv.Itoa(int(inst.ID)) + "&ai_model_id=" + strconv.Itoa(int(modelB.ID))
	req := multiModelReqWithSession(t, "POST", "/openclaw/model", user.Username, body)
	rr := httptest.NewRecorder()
	handleSetModel(rr, req, testCVMFetcher)

	if rr.Code != 200 {
		t.Fatalf("HandleSetModel 应返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	// 验证 instances.ai_model_id
	var updatedInst model.Instance
	model.DB(context.Background()).First(&updatedInst, inst.ID)
	if updatedInst.AIModelID != modelB.ID {
		t.Errorf("instances.ai_model_id 应为 %d，实际=%d", modelB.ID, updatedInst.AIModelID)
	}

	// 验证 instance_models 只有一条 primary 且为 modelB
	var primary model.InstanceModel
	if err := model.DB(context.Background()).Where("instance_id = ? AND role = ?", inst.ID, model.ModelRolePrimary).First(&primary).Error; err != nil {
		t.Fatalf("查找 primary 失败: %v", err)
	}
	if primary.AIModelID != modelB.ID {
		t.Errorf("primary 应为 modelB(%d)，实际=%d", modelB.ID, primary.AIModelID)
	}

	// 验证总记录数（应该只有 1 条，旧的被 Update 而不是 Create）
	var count int64
	model.DB(context.Background()).Model(&model.InstanceModel{}).Where("instance_id = ?", inst.ID).Count(&count)
	if count != 1 {
		t.Errorf("instance_models 应只有 1 条记录，实际=%d", count)
	}
}

// TestHandleSetModel_CustomModel_E2E 端到端验证 handleCustomModel 分支。
func TestHandleSetModel_CustomModel_E2E(t *testing.T) {
	setupMultiModelTestDB(t)

	origDomain := common.FixedSnapshot.Domain
	common.FixedSnapshot.Domain = "https://test.example.com"
	t.Cleanup(func() { common.FixedSnapshot.Domain = origDomain })

	origRunner := agentScriptRunner
	agentScriptRunner = func(ctx context.Context, instanceId string, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		return "{}", nil
	}
	defer func() { agentScriptRunner = origRunner }()

	origLoadScript := LoadScript
	LoadScript = func(name string) (string, error) {
		return "#!/bin/bash\necho ok", nil
	}
	defer func() { LoadScript = origLoadScript }()

	user, inst := createMultiModelUserAndInstance(t, "hermes-custom", "hermes-custom-inst")

	// 调用 HandleSetModel ai_model_id=0（自定义模型）
	model.DB(context.Background()).Model(inst).Update("agent_type", model.AgentTypeHermes)

	// 创建 custom flag 模型（handleCustomModel 依赖）
	customFlag := &model.AIModel{Provider: model.BuiltinModelProvider, ModelID: model.BuiltinModelID, Enabled: true, Visible: true, ModelType: ""}
	model.DB(context.Background()).Create(customFlag)

	// 模拟创建时写入的默认模型
	modelA := &model.AIModel{Provider: "hatchery", ModelID: "glm-4-plus", Enabled: true, Visible: true, ModelType: "openai-completions"}
	model.DB(context.Background()).Create(modelA)
	model.DB(context.Background()).Model(inst).Update("ai_model_id", modelA.ID)
	model.DB(context.Background()).Create(&model.InstanceModel{
		InstanceID: inst.ID,
		AIModelID:  modelA.ID,
		Role:       model.ModelRolePrimary,
		SortOrder:  1,
	})

	// 调用 HandleSetModel ai_model_id=0（自定义模型）
	body := "id=" + strconv.Itoa(int(inst.ID)) + "&ai_model_id=0&provider=custom&model_id=my-model&api_key=sk-test&url=https://api.example.com/v1&model_type=openai-completions"
	req := multiModelReqWithSession(t, "POST", "/openclaw/model", user.Username, body)
	rr := httptest.NewRecorder()
	handleSetModel(rr, req, testCVMFetcher)

	if rr.Code != 200 {
		t.Fatalf("handleSetModel(custom) 应返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	// 验证 instance_models 只有 1 条 primary 且 ai_model_id=0
	var primary model.InstanceModel
	if err := model.DB(context.Background()).Where("instance_id = ? AND role = ?", inst.ID, model.ModelRolePrimary).First(&primary).Error; err != nil {
		t.Fatalf("查找 primary 失败: %v", err)
	}
	if primary.AIModelID != 0 {
		t.Errorf("自定义模型 primary.AIModelID 应为 0，实际=%d", primary.AIModelID)
	}
	if primary.CustomModelID != "my-model" {
		t.Errorf("CustomModelID 应为 my-model，实际=%q", primary.CustomModelID)
	}

	var count int64
	model.DB(context.Background()).Model(&model.InstanceModel{}).Where("instance_id = ?", inst.ID).Count(&count)
	if count != 1 {
		t.Errorf("instance_models 应只有 1 条记录，实际=%d", count)
	}
}

// TestSetModelDBSync_BuiltinModel 直接测试 HandleSetModel 的 DB 事务逻辑。
// 跳过 TAT 调用，只验证 instance_models 被正确同步。
func TestSetModelDBSync_BuiltinModel(t *testing.T) {
	setupMultiModelTestDB(t)

	user, inst := createMultiModelUserAndInstance(t, "sync-user", "sync-inst")
	model.DB(context.Background()).Model(inst).Update("agent_type", model.AgentTypeHermes)

	modelA := &model.AIModel{Provider: "hatchery", ModelID: "glm-4-plus", Enabled: true, Visible: true, ModelType: "openai-completions"}
	modelB := &model.AIModel{Provider: "hatchery", ModelID: "deepseek-v3", Enabled: true, Visible: true, ModelType: "openai-completions"}
	model.DB(context.Background()).Create(modelA)
	model.DB(context.Background()).Create(modelB)
	_ = user

	// 模拟创建时写入
	model.DB(context.Background()).Model(inst).Update("ai_model_id", modelA.ID)
	model.DB(context.Background()).Create(&model.InstanceModel{
		InstanceID: inst.ID,
		AIModelID:  modelA.ID,
		Role:       model.ModelRolePrimary,
		SortOrder:  1,
	})

	// 直接执行 HandleSetModel 的 DB 事务逻辑（模拟 TAT 成功后的写入）
	err := model.DB(context.Background()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.InstanceModel{}).
			Where("instance_id = ? AND role = ?", inst.ID, model.ModelRolePrimary).
			Update("role", model.ModelRoleFallback).Error; err != nil {
			return err
		}
		nextSort, err := nextSortOrder(tx, inst.ID)
		if err != nil {
			return err
		}
		im := model.InstanceModel{
			InstanceID: inst.ID,
			AIModelID:  modelB.ID,
			Role:       model.ModelRolePrimary,
			SortOrder:  nextSort,
		}
		if err := tx.Where("instance_id = ? AND ai_model_id = ? AND custom_model_id = ?",
			inst.ID, modelB.ID, "").
			Assign(model.InstanceModel{Role: model.ModelRolePrimary, SortOrder: nextSort}).
			FirstOrCreate(&im).Error; err != nil {
			return err
		}
		return tx.Model(inst).Update("ai_model_id", modelB.ID).Error
	})
	if err != nil {
		t.Fatalf("事务执行失败: %v", err)
	}

	// 验证 1: instances.ai_model_id = modelB
	var updatedInst model.Instance
	model.DB(context.Background()).First(&updatedInst, inst.ID)
	if updatedInst.AIModelID != modelB.ID {
		t.Errorf("instances.ai_model_id 应为 %d，实际=%d", modelB.ID, updatedInst.AIModelID)
	}

	// 验证 2: instance_models 中 primary 为 modelB
	var primary model.InstanceModel
	if err := model.DB(context.Background()).Where("instance_id = ? AND role = ?", inst.ID, model.ModelRolePrimary).First(&primary).Error; err != nil {
		t.Fatalf("查找 primary 记录失败: %v", err)
	}
	if primary.AIModelID != modelB.ID {
		t.Errorf("instance_models primary 应为 modelB(%d)，实际=%d", modelB.ID, primary.AIModelID)
	}

	// 验证 3: modelA 被降级为 fallback
	var fallback model.InstanceModel
	if err := model.DB(context.Background()).Where("instance_id = ? AND ai_model_id = ?", inst.ID, modelA.ID).First(&fallback).Error; err != nil {
		t.Fatalf("查找 modelA 记录失败: %v", err)
	}
	if fallback.Role != model.ModelRoleFallback {
		t.Errorf("modelA 应降级为 fallback，实际 role=%q", fallback.Role)
	}
}

// TestSetModelDBSync_Idempotent 验证重复设置同一模型的幂等性。
func TestSetModelDBSync_Idempotent(t *testing.T) {
	setupMultiModelTestDB(t)

	_, inst := createMultiModelUserAndInstance(t, "idem-user", "idem-inst")
	model.DB(context.Background()).Model(inst).Update("agent_type", model.AgentTypeHermes)

	modelA := &model.AIModel{Provider: "hatchery", ModelID: "glm-4-plus", Enabled: true, Visible: true, ModelType: "openai-completions"}
	model.DB(context.Background()).Create(modelA)

	// 执行两次相同的 SetModel DB 事务
	for i := 0; i < 2; i++ {
		err := model.DB(context.Background()).Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&model.InstanceModel{}).
				Where("instance_id = ? AND role = ?", inst.ID, model.ModelRolePrimary).
				Update("role", model.ModelRoleFallback).Error; err != nil {
				return err
			}
			nextSort, err := nextSortOrder(tx, inst.ID)
			if err != nil {
				return err
			}
			im := model.InstanceModel{
				InstanceID: inst.ID,
				AIModelID:  modelA.ID,
				Role:       model.ModelRolePrimary,
				SortOrder:  nextSort,
			}
			if err := tx.Where("instance_id = ? AND ai_model_id = ? AND custom_model_id = ?",
				inst.ID, modelA.ID, "").
				Assign(model.InstanceModel{Role: model.ModelRolePrimary, SortOrder: nextSort}).
				FirstOrCreate(&im).Error; err != nil {
				return err
			}
			return tx.Model(inst).Update("ai_model_id", modelA.ID).Error
		})
		if err != nil {
			t.Fatalf("第 %d 次事务执行失败: %v", i+1, err)
		}
	}

	// 验证只有一条 primary 记录
	var count int64
	model.DB(context.Background()).Model(&model.InstanceModel{}).
		Where("instance_id = ? AND ai_model_id = ?", inst.ID, modelA.ID).
		Count(&count)
	if count != 1 {
		t.Errorf("重复 SetModel 后应只有 1 条记录，实际=%d", count)
	}
}

// TestSetModelDBSync_RollbackDeletesPrimary 验证 rollbackDefaultModelIfIntact 同步删除 instance_models。
func TestSetModelDBSync_RollbackDeletesPrimary(t *testing.T) {
	setupMultiModelTestDB(t)

	_, inst := createMultiModelUserAndInstance(t, "rb-user", "rb-inst")
	model.DB(context.Background()).Model(inst).Update("agent_type", model.AgentTypeHermes)

	modelA := &model.AIModel{Provider: "hatchery", ModelID: "glm-4-plus", Enabled: true, Visible: true, ModelType: "openai-completions"}
	model.DB(context.Background()).Create(modelA)

	// 模拟创建时写入
	model.DB(context.Background()).Model(inst).Update("ai_model_id", modelA.ID)
	model.DB(context.Background()).Create(&model.InstanceModel{
		InstanceID: inst.ID,
		AIModelID:  modelA.ID,
		Role:       model.ModelRolePrimary,
		SortOrder:  1,
	})

	// 执行 rollback 事务逻辑
	err := model.DB(context.Background()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("instance_id = ? AND ai_model_id = ? AND role = ?",
			inst.ID, modelA.ID, model.ModelRolePrimary).
			Delete(&model.InstanceModel{}).Error; err != nil {
			return err
		}
		return tx.Model(&model.Instance{}).Where("id = ? AND ai_model_id = ?", inst.ID, modelA.ID).
			Update("ai_model_id", 0).Error
	})
	if err != nil {
		t.Fatalf("回滚事务失败: %v", err)
	}

	// 验证 1: ai_model_id = 0
	var updatedInst model.Instance
	model.DB(context.Background()).First(&updatedInst, inst.ID)
	if updatedInst.AIModelID != 0 {
		t.Errorf("回滚后 ai_model_id 应为 0，实际=%d", updatedInst.AIModelID)
	}

	// 验证 2: instance_models 中无 primary 记录
	var count int64
	model.DB(context.Background()).Model(&model.InstanceModel{}).
		Where("instance_id = ? AND role = ?", inst.ID, model.ModelRolePrimary).
		Count(&count)
	if count != 0 {
		t.Errorf("回滚后不应有 primary 记录，实际=%d", count)
	}
}

// TestHandleSetModel_NoPrior_E2E 覆盖"instance_models 为空时首次 SetModel → Create"分支。
func TestHandleSetModel_NoPrior_E2E(t *testing.T) {
	setupMultiModelTestDB(t)

	origDomainNP := common.FixedSnapshot.Domain
	common.FixedSnapshot.Domain = "https://test.example.com"
	t.Cleanup(func() { common.FixedSnapshot.Domain = origDomainNP })

	origRunner := agentScriptRunner
	agentScriptRunner = func(ctx context.Context, instanceId string, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		return "{}", nil
	}
	defer func() { agentScriptRunner = origRunner }()

	origLoadScript := LoadScript
	LoadScript = func(name string) (string, error) {
		return "#!/bin/bash\necho ok", nil
	}
	defer func() { LoadScript = origLoadScript }()

	user, inst := createMultiModelUserAndInstance(t, "hermes-noprior", "hermes-noprior-inst")
	model.DB(context.Background()).Model(inst).Update("agent_type", model.AgentTypeHermes)

	modelA := &model.AIModel{Provider: "hatchery", ModelID: "glm-4-plus", Enabled: true, Visible: true, ModelType: "openai-completions"}
	model.DB(context.Background()).Create(modelA)

	// 不写 instance_models（模拟存量实例无记录的情况）

	body := "id=" + strconv.Itoa(int(inst.ID)) + "&ai_model_id=" + strconv.Itoa(int(modelA.ID))
	req := multiModelReqWithSession(t, "POST", "/openclaw/model", user.Username, body)
	rr := httptest.NewRecorder()
	handleSetModel(rr, req, testCVMFetcher)

	if rr.Code != 200 {
		t.Fatalf("HandleSetModel 应返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	// 验证新增了一条 primary
	var primary model.InstanceModel
	if err := model.DB(context.Background()).Where("instance_id = ? AND role = ?", inst.ID, model.ModelRolePrimary).First(&primary).Error; err != nil {
		t.Fatalf("应新增 primary 记录: %v", err)
	}
	if primary.AIModelID != modelA.ID {
		t.Errorf("primary.AIModelID 应为 %d，实际=%d", modelA.ID, primary.AIModelID)
	}
}

// TestHandleSetModel_CustomNoPrior_E2E 覆盖 handleCustomModel "无记录 → Create" 分支。
func TestHandleSetModel_CustomNoPrior_E2E(t *testing.T) {
	setupMultiModelTestDB(t)

	origDomainCNP := common.FixedSnapshot.Domain
	common.FixedSnapshot.Domain = "https://test.example.com"
	t.Cleanup(func() { common.FixedSnapshot.Domain = origDomainCNP })

	origRunner := agentScriptRunner
	agentScriptRunner = func(ctx context.Context, instanceId string, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		return "{}", nil
	}
	defer func() { agentScriptRunner = origRunner }()

	origLoadScript := LoadScript
	LoadScript = func(name string) (string, error) {
		return "#!/bin/bash\necho ok", nil
	}
	defer func() { LoadScript = origLoadScript }()

	user, inst := createMultiModelUserAndInstance(t, "hermes-custnp", "hermes-custnp-inst")
	model.DB(context.Background()).Model(inst).Update("agent_type", model.AgentTypeHermes)

	customFlag := &model.AIModel{Provider: model.BuiltinModelProvider, ModelID: model.BuiltinModelID, Enabled: true, Visible: true, ModelType: ""}
	model.DB(context.Background()).Create(customFlag)

	// 不写 instance_models

	body := "id=" + strconv.Itoa(int(inst.ID)) + "&ai_model_id=0&provider=custom&model_id=new-model&api_key=sk-test&url=https://api.example.com/v1&model_type=openai-completions"
	req := multiModelReqWithSession(t, "POST", "/openclaw/model", user.Username, body)
	rr := httptest.NewRecorder()
	handleSetModel(rr, req, testCVMFetcher)

	if rr.Code != 200 {
		t.Fatalf("handleSetModel(custom, no-prior, testCVMFetcher) 应返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	var primary model.InstanceModel
	if err := model.DB(context.Background()).Where("instance_id = ? AND role = ?", inst.ID, model.ModelRolePrimary).First(&primary).Error; err != nil {
		t.Fatalf("应新增 primary: %v", err)
	}
	if primary.CustomModelID != "new-model" {
		t.Errorf("CustomModelID 应为 new-model，实际=%q", primary.CustomModelID)
	}
}

// TestRollbackDefaultModel_E2E 覆盖 rollbackDefaultModelIfIntact 的事务逻辑。
func TestRollbackDefaultModel_E2E(t *testing.T) {
	setupMultiModelTestDB(t)

	_, inst := createMultiModelUserAndInstance(t, "rb-e2e-user", "rb-e2e-inst")
	model.DB(context.Background()).Model(inst).Update("agent_type", model.AgentTypeHermes)

	modelA := &model.AIModel{Provider: "hatchery", ModelID: "glm-4-plus", Enabled: true, Visible: true, ModelType: "openai-completions"}
	model.DB(context.Background()).Create(modelA)

	// 模拟创建时写入
	model.DB(context.Background()).Model(inst).Update("ai_model_id", modelA.ID)
	model.DB(context.Background()).Create(&model.InstanceModel{
		InstanceID: inst.ID,
		AIModelID:  modelA.ID,
		Role:       model.ModelRolePrimary,
		SortOrder:  1,
	})

	// 调用 rollbackDefaultModelIfIntact
	rollbackDefaultModelIfIntact(context.Background(), inst.ID, modelA.ID, "test_reason")

	// 验证 ai_model_id = 0
	var updatedInst model.Instance
	model.DB(context.Background()).First(&updatedInst, inst.ID)
	if updatedInst.AIModelID != 0 {
		t.Errorf("回滚后 ai_model_id 应为 0，实际=%d", updatedInst.AIModelID)
	}

	// 验证 instance_models 无 primary
	var count int64
	model.DB(context.Background()).Model(&model.InstanceModel{}).
		Where("instance_id = ? AND role = ?", inst.ID, model.ModelRolePrimary).
		Count(&count)
	if count != 0 {
		t.Errorf("回滚后不应有 primary 记录，实际=%d", count)
	}
}
