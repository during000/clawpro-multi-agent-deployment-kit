// Package controller 单元测试：覆盖 SlugifyModelID 白名单方案下
// buildSetModelParams 与 resolveBindingRef 的 providerKey 一致性。
package controller

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"hatchery/common"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupSlugTestDB 初始化 slug 测试所需的内存 SQLite 数据库。
func setupSlugTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.Instance{},
		&model.AIModel{},
		&model.InstanceModel{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	origDB := model.UseDBForTest(db)
	return origDB
}

// TestBuildSetModelParams_ProviderKeyConsistency 验证：
// buildSetModelParams 生成的 providerKey 与 resolveBindingRef 生成的 ref 前缀一致。
// 这是方案 B 的核心断言——providers key 和 fallbacks ref 必须对得上。
func TestBuildSetModelParams_ProviderKeyConsistency(t *testing.T) {
	cleanup := setupSlugTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户和实例
	user := model.User{Username: "slug-test-user", Password: "x"}
	model.DB(ctx).Create(&user)
	inst := model.Instance{Name: "slug-test-inst", UserID: user.ID, InstanceId: "ins-slug-test"}
	model.DB(ctx).Create(&inst)

	tests := []struct {
		name              string
		provider          string // AIModel.Provider
		modelID           string // AIModel.ModelID
		isUserCustomModel bool   // AIModelID == 0 (用户侧自定义模型)
		wantPrefix        string // 期望的 providerKey 前缀
		wantSlugID        string // 期望的 slugified model id
	}{
		{
			name:              "普通内置模型",
			provider:          "hatchery",
			modelID:           "deepseek-v3.2",
			isUserCustomModel: false,
			wantPrefix:        "hatchery",
			wantSlugID:        "deepseek-v3.2",
		},
		{
			name:              "qcloudlkeap 内置模型",
			provider:          "qcloudlkeap",
			modelID:           "deepseek-v3.2",
			isUserCustomModel: false,
			wantPrefix:        "qcloudlkeap",
			wantSlugID:        "deepseek-v3.2",
		},
		{
			name:              "管理侧自定义模型（含斜杠和冒号）",
			provider:          common.CustomModelProvider,
			modelID:           "minimax/minimax-m2.5:free",
			isUserCustomModel: false,
			wantPrefix:        model.BuiltinModelProvider, // "hatchery"
			wantSlugID:        "minimax-minimax-m2.5-free",
		},
		{
			name:              "管理侧自定义模型（含斜杠）",
			provider:          common.CustomModelProvider,
			modelID:           "ZhiJia/GLM-5.1-Plus",
			isUserCustomModel: false,
			wantPrefix:        model.BuiltinModelProvider,
			wantSlugID:        "zhijia-glm-5.1-plus",
		},
		{
			name:              "用户侧自定义模型",
			provider:          common.CustomModelProvider,
			modelID:           "minimax/minimax-m2.5:free",
			isUserCustomModel: true,
			wantPrefix:        "custom",
			wantSlugID:        "minimax-minimax-m2.5-free",
		},
		{
			name:              "含大写的内置模型",
			provider:          "hatchery",
			modelID:           "GLM-5.1",
			isUserCustomModel: false,
			wantPrefix:        "hatchery",
			wantSlugID:        "glm-5.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedProviderKey := fmt.Sprintf("%s-%s", tt.wantPrefix, tt.wantSlugID)
			// 【方案 C】用户侧自定义模型 ref 后段保留原始 ModelID（不 slug 化）；
			// 其他场景仍后段用 slug 化后的 modelId。
			var expectedRef string
			if tt.isUserCustomModel {
				expectedRef = fmt.Sprintf("%s/%s", expectedProviderKey, tt.modelID)
			} else {
				expectedRef = fmt.Sprintf("%s/%s", expectedProviderKey, tt.wantSlugID)
			}

			if tt.isUserCustomModel {
				// 用户侧自定义模型：AIModelID=0
				im := model.InstanceModel{
					InstanceID:    inst.ID,
					AIModelID:     0,
					CustomModelID: tt.modelID,
					CustomModelConfig: fmt.Sprintf(
						`{"provider":"%s","model_id":"%s","api_key":"sk-test","url":"https://api.example.com","model_type":"openai-completions","context_len":128000}`,
						tt.provider, tt.modelID),
					Role:      model.ModelRoleFallback,
					SortOrder: 99,
				}
				model.DB(ctx).Create(&im)
				defer model.DB(ctx).Unscoped().Delete(&im)

				// 验证 resolveBindingRef
				ref := resolveBindingRef(ctx, im)
				if ref != expectedRef {
					t.Errorf("resolveBindingRef: got %q, want %q", ref, expectedRef)
				}

				// 验证 buildSetModelParams（用户侧自定义模型 ID=0）
				aim := model.AIModel{
					Provider:  tt.provider,
					ModelID:   tt.modelID,
					URL:       "https://api.example.com",
					APIKey:    "sk-test",
					ModelType: "openai-completions",
				}
				// 注意：aim.ID 保持 0（未 Create），模拟用户侧自定义模型
				params, err := buildSetModelParams(ctx, aim, inst.ID, true)
				if err != nil {
					t.Fatalf("buildSetModelParams 失败: %v", err)
				}
				if params["provider"] != expectedProviderKey {
					t.Errorf("buildSetModelParams provider: got %q, want %q", params["provider"], expectedProviderKey)
				}

				// 验证 provider key 是 ref 的前缀（保持与内置分支对称的断言）
				if !strings.HasPrefix(ref, params["provider"]+"/") {
					t.Errorf("ref %q 应以 providerKey %q + '/' 为前缀", ref, params["provider"])
				}
			} else {
				// 内置/管理侧自定义模型：AIModelID > 0
				aim := model.AIModel{
					Provider:  tt.provider,
					ModelID:   tt.modelID,
					URL:       "https://api.example.com",
					APIKey:    "sk-test",
					ModelType: "openai-completions",
					Enabled:   true,
					QuotaDay:  -1,
				}
				model.DB(ctx).Create(&aim)
				defer model.DB(ctx).Unscoped().Delete(&aim)

				im := model.InstanceModel{
					InstanceID: inst.ID,
					AIModelID:  aim.ID,
					Role:       model.ModelRoleFallback,
					SortOrder:  99,
				}
				model.DB(ctx).Create(&im)
				defer model.DB(ctx).Unscoped().Delete(&im)

				// 验证 resolveBindingRef
				ref := resolveBindingRef(ctx, im)
				if ref != expectedRef {
					t.Errorf("resolveBindingRef: got %q, want %q", ref, expectedRef)
				}

				// 验证 buildSetModelParams（aim.ID > 0）
				params, err := buildSetModelParams(ctx, aim, inst.ID, false)
				if err != nil {
					t.Fatalf("buildSetModelParams 失败: %v", err)
				}
				if params["provider"] != expectedProviderKey {
					t.Errorf("buildSetModelParams provider: got %q, want %q", params["provider"], expectedProviderKey)
				}

				// 验证 provider key 是 ref 的前缀（ref = providerKey/slugID）
				if !strings.HasPrefix(ref, params["provider"]+"/") {
					t.Errorf("ref %q 应以 providerKey %q + '/' 为前缀", ref, params["provider"])
				}
			}
		})
	}
}

// TestBuildSetModelParams_ModelIDPreserved 验证：
// buildSetModelParams 生成的 provider JSON 中 models[].id 保留原始 ModelID（保真路径）。
func TestBuildSetModelParams_ModelIDPreserved(t *testing.T) {
	cleanup := setupSlugTestDB(t)
	defer cleanup()

	ctx := context.Background()

	user := model.User{Username: "preserve-user", Password: "x"}
	model.DB(ctx).Create(&user)
	inst := model.Instance{Name: "preserve-inst", UserID: user.ID, InstanceId: "ins-preserve"}
	model.DB(ctx).Create(&inst)

	tests := []struct {
		name    string
		modelID string
	}{
		{name: "含斜杠", modelID: "minimax/minimax-m2.5:free"},
		{name: "含冒号", modelID: "model:v1.0"},
		{name: "普通模型", modelID: "deepseek-v3.2"},
		{name: "含大写", modelID: "GLM-5.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aim := model.AIModel{
				Provider:  "hatchery",
				ModelID:   tt.modelID,
				URL:       "https://api.example.com",
				APIKey:    "sk-test",
				ModelType: "openai-completions",
			}
			model.DB(ctx).Create(&aim)
			defer model.DB(ctx).Unscoped().Delete(&aim)

			params, richErr := buildSetModelParams(ctx, aim, inst.ID, false)
			if richErr != nil {
				t.Fatalf("buildSetModelParams 失败: %v", richErr)
			}

			// 解码 valueb64 并检查 models[0].id
			valueJSON, err := base64.StdEncoding.DecodeString(params["valueb64"])
			if err != nil {
				t.Fatalf("base64 解码失败: %v", err)
			}
			var valueObj struct {
				Models []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"models"`
			}
			if err := json.Unmarshal(valueJSON, &valueObj); err != nil {
				t.Fatalf("JSON 解析失败: %v", err)
			}
			if len(valueObj.Models) == 0 {
				t.Fatal("models 数组为空")
			}
			// models[].id 应保留原始 ModelID（保真路径）
			if valueObj.Models[0].ID != tt.modelID {
				t.Errorf("models[0].id: got %q, want %q (原始值保真)", valueObj.Models[0].ID, tt.modelID)
			}
		})
	}
}

// TestBuildSetModelParams_SlugModel 验证：
// buildSetModelParams 返回的 "model" 参数是 slugified 后的值。
func TestBuildSetModelParams_SlugModel(t *testing.T) {
	cleanup := setupSlugTestDB(t)
	defer cleanup()

	ctx := context.Background()

	user := model.User{Username: "slug-model-user", Password: "x"}
	model.DB(ctx).Create(&user)
	inst := model.Instance{Name: "slug-model-inst", UserID: user.ID, InstanceId: "ins-slug-model"}
	model.DB(ctx).Create(&inst)

	tests := []struct {
		name      string
		modelID   string
		wantModel string
	}{
		{name: "含斜杠和冒号", modelID: "minimax/minimax-m2.5:free", wantModel: "minimax-minimax-m2.5-free"},
		{name: "含大写", modelID: "GLM-5.1", wantModel: "glm-5.1"},
		{name: "普通模型", modelID: "deepseek-v3.2", wantModel: "deepseek-v3.2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aim := model.AIModel{
				Provider:  "hatchery",
				ModelID:   tt.modelID,
				URL:       "https://api.example.com",
				APIKey:    "sk-test",
				ModelType: "openai-completions",
			}
			model.DB(ctx).Create(&aim)
			defer model.DB(ctx).Unscoped().Delete(&aim)

			params, err := buildSetModelParams(ctx, aim, inst.ID, false)
			if err != nil {
				t.Fatalf("buildSetModelParams 失败: %v", err)
			}
			if params["model"] != tt.wantModel {
				t.Errorf("params[model]: got %q, want %q", params["model"], tt.wantModel)
			}
		})
	}
}

// TestInjectModelConfigToCVM_ModelType 验证：
// injectModelConfigToCVM 对内置模型分支强制设置 ModelType = "openai-completions"。
func TestInjectModelConfigToCVM_ModelType(t *testing.T) {
	cleanup := setupSlugTestDB(t)
	defer cleanup()

	// 注入 Domain 到 context
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		Domain: "http://test.example.com",
	})

	user := model.User{Username: "modeltype-user", Password: "x"}
	model.DB(ctx).Create(&user)
	proxyToken := "test-proxy-token"
	inst := model.Instance{
		Name:       "modeltype-inst",
		UserID:     user.ID,
		InstanceId: "ins-modeltype",
		ProxyToken: &proxyToken,
	}
	model.DB(ctx).Create(&inst)

	// Stub injectModelScriptRunner 捕获参数
	var capturedParams map[string]string
	cleanupRunner := withInjectScriptRunner(func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		capturedParams = params
		return "{}", nil
	})
	defer cleanupRunner()

	// 内置模型，ModelType 为 anthropic-messages（应被覆写为 openai-completions）
	aim := model.AIModel{
		Provider:  "hatchery",
		ModelID:   "claude-3-opus",
		URL:       "https://api.anthropic.com",
		APIKey:    "sk-ant-test",
		ModelType: "anthropic-messages",
		Enabled:   true,
	}
	model.DB(ctx).Create(&aim)

	err := injectModelConfigToCVM(ctx, &inst, &aim, false)
	if err != nil {
		t.Fatalf("injectModelConfigToCVM 失败: %v", err)
	}

	// 验证 valueb64 中的 api 字段是 openai-completions（被覆写）
	valueJSON, err := base64.StdEncoding.DecodeString(capturedParams["valueb64"])
	if err != nil {
		t.Fatalf("base64 解码失败: %v", err)
	}
	var valueObj struct {
		API string `json:"api"`
	}
	if err := json.Unmarshal(valueJSON, &valueObj); err != nil {
		t.Fatalf("JSON 解析失败: %v", err)
	}
	if valueObj.API != "openai-completions" {
		t.Errorf("内置模型的 api 应被覆写为 openai-completions, 实际=%q", valueObj.API)
	}
}

// TestInjectModelConfigToCVM_UserCustomModelType 验证：
// injectModelConfigToCVM 对用户侧自定义模型分支保留原始 ModelType。
func TestInjectModelConfigToCVM_UserCustomModelType(t *testing.T) {
	cleanup := setupSlugTestDB(t)
	defer cleanup()

	ctx := context.Background()

	user := model.User{Username: "custom-modeltype-user", Password: "x"}
	model.DB(ctx).Create(&user)
	inst := model.Instance{
		Name:       "custom-modeltype-inst",
		UserID:     user.ID,
		InstanceId: "ins-custom-modeltype",
	}
	model.DB(ctx).Create(&inst)

	// Stub injectModelScriptRunner 捕获参数
	var capturedParams map[string]string
	cleanupRunner := withInjectScriptRunner(func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
		capturedParams = params
		return "{}", nil
	})
	defer cleanupRunner()

	// 自定义模型，ModelType 为 anthropic-messages（应保留）
	aim := model.AIModel{
		Provider:  common.CustomModelProvider,
		ModelID:   "claude-3-opus",
		URL:       "https://api.anthropic.com",
		APIKey:    "sk-ant-test",
		ModelType: "anthropic-messages",
	}
	// 注意：不 Create 到 DB，ID 保持 0（用户侧自定义模型）

	err := injectModelConfigToCVM(ctx, &inst, &aim, true)
	if err != nil {
		t.Fatalf("injectModelConfigToCVM 失败: %v", err)
	}

	// 验证 valueb64 中的 api 字段保留 anthropic-messages
	valueJSON, err := base64.StdEncoding.DecodeString(capturedParams["valueb64"])
	if err != nil {
		t.Fatalf("base64 解码失败: %v", err)
	}
	var valueObj struct {
		API string `json:"api"`
	}
	if err := json.Unmarshal(valueJSON, &valueObj); err != nil {
		t.Fatalf("JSON 解析失败: %v", err)
	}
	if valueObj.API != "anthropic-messages" {
		t.Errorf("自定义模型的 api 应保留原值 anthropic-messages, 实际=%q", valueObj.API)
	}
}

// TestBuildInstanceModelListItem_BindingIDConsistency 验证：
// buildInstanceModelListItem 生成的 bindingID 与 resolveBindingRef 完全一致。
// 这是修复点 3 的核心断言：避免前端通过 bindingID 操作时与后端 ref 不匹配。
func TestBuildInstanceModelListItem_BindingIDConsistency(t *testing.T) {
	cleanup := setupSlugTestDB(t)
	defer cleanup()

	ctx := context.Background()

	user := model.User{Username: "binding-user", Password: "x"}
	model.DB(ctx).Create(&user)
	inst := model.Instance{Name: "binding-inst", UserID: user.ID, InstanceId: "ins-binding"}
	model.DB(ctx).Create(&inst)

	tests := []struct {
		name              string
		isUserCustomModel bool
		provider          string
		modelID           string
	}{
		{
			name:              "内置模型_普通",
			isUserCustomModel: false,
			provider:          "hatchery",
			modelID:           "deepseek-v3.2",
		},
		{
			name:              "内置模型_含斜杠",
			isUserCustomModel: false,
			provider:          "hatchery",
			modelID:           "openrouter/deepseek-v3",
		},
		{
			name:              "内置模型_含冒号",
			isUserCustomModel: false,
			provider:          "hatchery",
			modelID:           "minimax-m2.5:free",
		},
		{
			name:              "管理侧自定义模型_含斜杠和冒号",
			isUserCustomModel: false,
			provider:          common.CustomModelProvider,
			modelID:           "minimax/minimax-m2.5:free",
		},
		{
			name:              "用户侧自定义模型_含斜杠",
			isUserCustomModel: true,
			provider:          common.CustomModelProvider,
			modelID:           "openrouter/auto",
		},
		{
			name:              "用户侧自定义模型_含冒号",
			isUserCustomModel: true,
			provider:          common.CustomModelProvider,
			modelID:           "minimax-m2.5:free",
		},
		{
			name:              "用户侧自定义模型_含斜杠和冒号",
			isUserCustomModel: true,
			provider:          common.CustomModelProvider,
			modelID:           "minimax/minimax-m2.5:free",
		},
		{
			name:              "用户侧自定义模型_含大写",
			isUserCustomModel: true,
			provider:          common.CustomModelProvider,
			modelID:           "MyModel/V1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var im model.InstanceModel
			aimMap := make(map[uint]model.AIModel)

			if tt.isUserCustomModel {
				cfg := fmt.Sprintf(
					`{"provider":"%s","model_id":"%s","api_key":"sk-test","url":"https://api.example.com","model_type":"openai-completions","context_len":128000}`,
					tt.provider, tt.modelID)
				im = model.InstanceModel{
					InstanceID:        inst.ID,
					AIModelID:         0,
					CustomModelID:     tt.modelID,
					CustomModelConfig: cfg,
					Role:              model.ModelRoleFallback,
					SortOrder:         99,
				}
				model.DB(ctx).Create(&im)
				defer model.DB(ctx).Unscoped().Delete(&im)
			} else {
				aim := model.AIModel{
					Provider:  tt.provider,
					ModelID:   tt.modelID,
					ModelName: tt.modelID,
					ModelType: "openai-completions",
					Enabled:   true,
				}
				model.DB(ctx).Create(&aim)
				defer model.DB(ctx).Unscoped().Delete(&aim)

				im = model.InstanceModel{
					InstanceID: inst.ID,
					AIModelID:  aim.ID,
					Role:       model.ModelRoleFallback,
					SortOrder:  99,
				}
				model.DB(ctx).Create(&im)
				defer model.DB(ctx).Unscoped().Delete(&im)

				aimMap[aim.ID] = aim
			}

			// 生成两个 ID
			itemBindingID := buildInstanceModelListItem(im, aimMap).BindingID
			refBindingID := resolveBindingRef(ctx, im)

			// 核心断言：两者完全一致
			if itemBindingID != refBindingID {
				t.Errorf("bindingID 不一致: buildInstanceModelListItem=%q, resolveBindingRef=%q",
					itemBindingID, refBindingID)
			}

			// 同时验证 bindingID 不含非法字符（除了 / 作为 key 与 modelId 的分隔符）
			// 【方案 C】用户侧自定义模型的 bindingID 后段保留原始 ModelID，可能
			// 含大小写 / 斜杠 / 冒号等非 [a-z0-9._-] 字符，这是设计中的预期行为
			// （上游 LLM API 需要原始 model 名识别）。所以字符集校验仅限内置/管理侧场景。
			// 对于用户侧自定义模型，只验证前段 providerKey 字符集（后段仅验证顺为原始 ModelID）。
			parts := strings.SplitN(itemBindingID, "/", 2)
			if len(parts) != 2 {
				t.Errorf("bindingID %q 应是 \"providerKey/modelId\" 格式", itemBindingID)
				return
			}
			if tt.isUserCustomModel {
				// 仅校验 providerKey 前段的字符集
				for _, r := range parts[0] {
					if !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '.' && r != '_' && r != '-' {
						t.Errorf("bindingID providerKey 部分 %q 含非法字符 %q（应只含 [a-z0-9._-]）", parts[0], r)
					}
				}
				// 后段验证与 CustomModelID 原值完全一致
				if parts[1] != tt.modelID {
					t.Errorf("bindingID 后段应为原始 ModelID %q，实际 %q", tt.modelID, parts[1])
				}
			} else {
				// 内置/管理侧场景：两段都必须是 slug 化后的字符集
				for _, part := range parts {
					for _, r := range part {
						if !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '.' && r != '_' && r != '-' {
							t.Errorf("bindingID 部分 %q 含非法字符 %q（应只含 [a-z0-9._-]）", part, r)
						}
					}
				}
			}
		})
	}
}

// TestBuildInstanceModelListItem_AIModelMissing 验证：
// 内置模型在 aimMap 中找不到时（ai_models 记录被删除），bindingID 仍能优雅生成。
func TestBuildInstanceModelListItem_AIModelMissing(t *testing.T) {
	im := model.InstanceModel{
		InstanceID: 1,
		AIModelID:  99999, // 不存在的 ID
		Role:       model.ModelRoleFallback,
	}
	aimMap := make(map[uint]model.AIModel) // 空 map

	item := buildInstanceModelListItem(im, aimMap)
	// 当 modelID 为空时（aim 是 zero value），bindingID 应回退到 "{prefix}-{aiModelID}/unknown"
	expected := fmt.Sprintf("-%d/unknown", im.AIModelID)
	if !strings.HasSuffix(item.BindingID, expected) {
		t.Errorf("bindingID 应以 %q 结尾，实际=%q", expected, item.BindingID)
	}
}

// TestBuildSetModelParams_PlanC_PreserveCustomModelIDCase 是方案 C 的核心回归测试。
//
// 背景：
//
//	用户侧自定义模型（Provider==CustomModelProvider 且 AIModelID==0）的请求由
//	OpenClaw Agent 直接透传到上游 BaseURL，不经过 hatchery 的 llm_proxy 兜底覆盖。
//	OpenClaw 解析 fallbacks/primary 的 ref 时取 "/" 之后的部分作为 body.model
//	发给上游，必须保留用户原始填写的大小写（如 "DeepSeek-V3.1"），否则
//	上游会返回 model_not_found 404。
//
// 本测试断言：
//  1. 用户侧自定义模型 ref 后段保留原始 ModelID（含大小写、含 "/"、含 ":"）。
//  2. 内置模型 ref 后段仍 slug 化（保持存量行为，因为有 llm_proxy 兜底覆盖）。
//  3. 同一实例下用户侧自定义 + 内置模型并存时，fallbacks JSON 数组中
//     两类 ref 各自的大小写处理符合预期。
//  4. 提交给 set_model.sh 的 provider 字段（白名单字符集）始终是 slug 化的，
//     不会引入非法字符。
func TestBuildSetModelParams_PlanC_PreserveCustomModelIDCase(t *testing.T) {
	cleanup := setupSlugTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// 准备 user + instance
	user := model.User{Username: "planc-user", Password: "x"}
	model.DB(ctx).Create(&user)
	inst := model.Instance{Name: "planc-inst", UserID: user.ID, InstanceId: "ins-planc"}
	model.DB(ctx).Create(&inst)

	// 1) 内置模型作为 fallback（应 slug 化）
	builtinAIM := model.AIModel{
		Provider:  "hatchery",
		ModelID:   "GLM-4-Plus",
		ModelName: "GLM-4-Plus",
		ModelType: "openai-completions",
		URL:       "https://api.example.com",
		APIKey:    "sk-test",
		Enabled:   true,
		QuotaDay:  -1,
	}
	model.DB(ctx).Create(&builtinAIM)
	model.DB(ctx).Create(&model.InstanceModel{
		InstanceID: inst.ID,
		AIModelID:  builtinAIM.ID,
		Role:       model.ModelRoleFallback,
		SortOrder:  1,
	})

	// 2) 用户侧自定义模型作为 primary（应保留原始大小写）
	customCfg := `{"provider":"自定义模型","model_id":"DeepSeek-V3.1","api_key":"sk-test","url":"https://api.example.com","model_type":"openai-completions","context_len":128000}`
	model.DB(ctx).Create(&model.InstanceModel{
		InstanceID:        inst.ID,
		AIModelID:         0,
		CustomModelID:     "DeepSeek-V3.1",
		CustomModelConfig: customCfg,
		Role:              model.ModelRolePrimary,
		SortOrder:         0,
	})

	// 3) 调用 buildSetModelParams 模拟"再次添加一个用户侧自定义模型"
	newCustom := model.AIModel{
		Provider:   common.CustomModelProvider,
		ModelID:    "Qwen2.5-72B-Instruct",
		ModelName:  "Qwen2.5-72B-Instruct",
		ModelType:  "openai-completions",
		URL:        "https://api.example.com",
		APIKey:     "sk-test",
		ContextLen: 128000,
		// 关键：ID 保持 0，模拟用户侧自定义
	}
	params, richErr := buildSetModelParams(ctx, newCustom, inst.ID, true)
	if richErr != nil {
		t.Fatalf("buildSetModelParams 失败: %v", richErr)
	}

	// 断言 A：provider 字段（写入 models.providers 的 key）必须 slug 化
	wantProviderKey := "custom-qwen2.5-72b-instruct"
	if params["provider"] != wantProviderKey {
		t.Errorf("provider 应为 %q（slug 化），实际=%q", wantProviderKey, params["provider"])
	}
	// model 字段同样
	if params["model"] != "qwen2.5-72b-instruct" {
		t.Errorf("model 应为 %q（slug 化），实际=%q", "qwen2.5-72b-instruct", params["model"])
	}

	// 断言 B：primary 仍是 DB 里已有的用户侧自定义模型 ref，且后段保留原始大小写
	wantPrimary := "custom-deepseek-v3.1/DeepSeek-V3.1"
	if params["primary"] != wantPrimary {
		t.Errorf("primary 应为 %q（用户侧自定义后段保留大小写），实际=%q", wantPrimary, params["primary"])
	}

	// 断言 C：fallbacks 数组里包含内置模型 ref（应 slug 化）
	fallbacksB64 := params["fallbacksb64"]
	fallbacksRaw, err := base64.StdEncoding.DecodeString(fallbacksB64)
	if err != nil {
		t.Fatalf("fallbacksb64 base64 解码失败: %v", err)
	}
	var fallbacksArr []string
	if err := json.Unmarshal(fallbacksRaw, &fallbacksArr); err != nil {
		t.Fatalf("fallbacks JSON 解析失败: %v\n--- raw ---\n%s", err, fallbacksRaw)
	}
	wantFallback := "hatchery-glm-4-plus/glm-4-plus"
	found := false
	for _, ref := range fallbacksArr {
		if ref == wantFallback {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("fallbacks 应包含内置模型 %q（slug 化），实际=%v", wantFallback, fallbacksArr)
	}

	// 断言 D：本次 newCustom 模型如果作为 primary 兜底（DB 中无 primary 时），
	// thisModelRef 后段也必须保留原始 ModelID。
	// 这里通过新建一个空实例验证兜底分支。
	emptyInst := model.Instance{Name: "planc-empty", UserID: user.ID, InstanceId: "ins-planc-empty"}
	model.DB(ctx).Create(&emptyInst)
	params2, rerr := buildSetModelParams(ctx, newCustom, emptyInst.ID, true)
	if rerr != nil {
		t.Fatalf("buildSetModelParams（空实例）失败: %v", rerr)
	}
	wantThisRef := "custom-qwen2.5-72b-instruct/Qwen2.5-72B-Instruct"
	if params2["primary"] != wantThisRef {
		t.Errorf("空实例下 primary 应为 %q（thisModelRef 后段保留原始大小写），实际=%q",
			wantThisRef, params2["primary"])
	}

	// 断言 E：内置模型走 buildSetModelParams 时，thisModelRef 后段仍 slug 化（存量行为）
	builtinNew := model.AIModel{
		Provider:   "hatchery",
		ModelID:    "DeepSeek-V3.1", // 故意用大写
		ModelName:  "DeepSeek-V3.1",
		ModelType:  "openai-completions",
		URL:        "https://api.example.com",
		APIKey:     "sk-test",
		ContextLen: 128000,
	}
	model.DB(ctx).Create(&builtinNew)
	emptyInst2 := model.Instance{Name: "planc-empty2", UserID: user.ID, InstanceId: "ins-planc-empty2"}
	model.DB(ctx).Create(&emptyInst2)
	params3, rerr := buildSetModelParams(ctx, builtinNew, emptyInst2.ID, false)
	if rerr != nil {
		t.Fatalf("buildSetModelParams（内置模型）失败: %v", rerr)
	}
	wantBuiltinRef := "hatchery-deepseek-v3.1/deepseek-v3.1"
	if params3["primary"] != wantBuiltinRef {
		t.Errorf("内置模型 primary 应为 %q（slug 化，存量行为），实际=%q",
			wantBuiltinRef, params3["primary"])
	}
}
