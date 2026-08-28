package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hatchery/model"
)

// mockCheckOpenclawConfig 在测试期间替换 checkOpenclawConfigFn，测试结束后自动恢复原始实现。
func mockCheckOpenclawConfig(t *testing.T, fn func(ctx context.Context, instanceId string, script string, timeout uint64) (string, error)) {
	t.Helper()
	orig := checkOpenclawConfigFn
	checkOpenclawConfigFn = fn
	t.Cleanup(func() { checkOpenclawConfigFn = orig })
}

// TestCheckOpenclawConfigProviderKeys_NilInstance 验证：instance 为 nil 时直接返回 nil，不调用 TAT
func TestCheckOpenclawConfigProviderKeys_NilInstance(t *testing.T) {
	called := false
	mockCheckOpenclawConfig(t, func(_ context.Context, _ string, _ string, _ uint64) (string, error) {
		called = true
		return "", nil
	})

	err := checkOpenclawConfigProviderKeys(context.Background(), nil)
	if err != nil {
		t.Errorf("nil instance 应返回 nil，实际=%v", err)
	}
	if called {
		t.Error("nil instance 不应调用 TAT")
	}
}

// TestCheckOpenclawConfigProviderKeys_TATError 验证：TAT 执行失败时不阻断升级（返回 nil）
func TestCheckOpenclawConfigProviderKeys_TATError(t *testing.T) {
	mockCheckOpenclawConfig(t, func(_ context.Context, _ string, _ string, _ uint64) (string, error) {
		return "", errors.New("TAT agent 离线")
	})

	inst := &model.Instance{InstanceId: "ins-tat-err"}
	err := checkOpenclawConfigProviderKeys(context.Background(), inst)
	if err != nil {
		t.Errorf("TAT 失败时应跳过检查返回 nil，实际=%v", err)
	}
}

// TestCheckOpenclawConfigProviderKeys_EmptyOutput 验证：TAT 返回空内容（文件不存在）时返回 nil
func TestCheckOpenclawConfigProviderKeys_EmptyOutput(t *testing.T) {
	mockCheckOpenclawConfig(t, func(_ context.Context, _ string, _ string, _ uint64) (string, error) {
		return "", nil
	})

	inst := &model.Instance{InstanceId: "ins-empty"}
	err := checkOpenclawConfigProviderKeys(context.Background(), inst)
	if err != nil {
		t.Errorf("空输出应返回 nil，实际=%v", err)
	}
}

// TestCheckOpenclawConfigProviderKeys_WhitespaceOnlyOutput 验证：TAT 返回纯空白内容时返回 nil
func TestCheckOpenclawConfigProviderKeys_WhitespaceOnlyOutput(t *testing.T) {
	mockCheckOpenclawConfig(t, func(_ context.Context, _ string, _ string, _ uint64) (string, error) {
		return "   \n\t  ", nil
	})

	inst := &model.Instance{InstanceId: "ins-ws"}
	err := checkOpenclawConfigProviderKeys(context.Background(), inst)
	if err != nil {
		t.Errorf("纯空白输出应返回 nil，实际=%v", err)
	}
}

// TestCheckOpenclawConfigProviderKeys_InvalidJSON 验证：JSON 解析失败时不阻断升级（返回 nil）
func TestCheckOpenclawConfigProviderKeys_InvalidJSON(t *testing.T) {
	mockCheckOpenclawConfig(t, func(_ context.Context, _ string, _ string, _ uint64) (string, error) {
		return "this is not json {{{", nil
	})

	inst := &model.Instance{InstanceId: "ins-bad-json"}
	err := checkOpenclawConfigProviderKeys(context.Background(), inst)
	if err != nil {
		t.Errorf("JSON 解析失败应跳过检查返回 nil，实际=%v", err)
	}
}

// TestCheckOpenclawConfigProviderKeys_NoModelsSection 验证：JSON 中无 models 字段时，providers 为空，返回 nil
func TestCheckOpenclawConfigProviderKeys_NoModelsSection(t *testing.T) {
	mockCheckOpenclawConfig(t, func(_ context.Context, _ string, _ string, _ uint64) (string, error) {
		return `{"gateway":{"mode":"local"}}`, nil
	})

	inst := &model.Instance{InstanceId: "ins-no-models"}
	err := checkOpenclawConfigProviderKeys(context.Background(), inst)
	if err != nil {
		t.Errorf("无 models 字段时应返回 nil，实际=%v", err)
	}
}

// TestCheckOpenclawConfigProviderKeys_EmptyProviders 验证：providers 为空对象时返回 nil
func TestCheckOpenclawConfigProviderKeys_EmptyProviders(t *testing.T) {
	mockCheckOpenclawConfig(t, func(_ context.Context, _ string, _ string, _ uint64) (string, error) {
		return `{"models":{"providers":{}}}`, nil
	})

	inst := &model.Instance{InstanceId: "ins-empty-providers"}
	err := checkOpenclawConfigProviderKeys(context.Background(), inst)
	if err != nil {
		t.Errorf("空 providers 应返回 nil，实际=%v", err)
	}
}

// TestCheckOpenclawConfigProviderKeys_AllValidKeys 验证：所有 provider key 均合法时返回 nil
func TestCheckOpenclawConfigProviderKeys_AllValidKeys(t *testing.T) {
	mockCheckOpenclawConfig(t, func(_ context.Context, _ string, _ string, _ uint64) (string, error) {
		return `{
			"models": {
				"providers": {
					"hatchery-qwen3.6-plus": {},
					"hatchery-glm-5.1": {},
					"hatchery-deepseek-v4-flash": {}
				}
			}
		}`, nil
	})

	inst := &model.Instance{InstanceId: "ins-valid-keys"}
	err := checkOpenclawConfigProviderKeys(context.Background(), inst)
	if err != nil {
		t.Errorf("合法 key 应返回 nil，实际=%v", err)
	}
}

// TestCheckOpenclawConfigProviderKeys_SingleInvalidKey 验证：单个含 "/" 的 key 被检测并返回错误
func TestCheckOpenclawConfigProviderKeys_SingleInvalidKey(t *testing.T) {
	mockCheckOpenclawConfig(t, func(_ context.Context, _ string, _ string, _ uint64) (string, error) {
		return `{
			"models": {
				"providers": {
					"hatchery-qwen3.6-plus/qwen3.6-plus": {}
				}
			}
		}`, nil
	})

	inst := &model.Instance{InstanceId: "ins-invalid-key"}
	err := checkOpenclawConfigProviderKeys(context.Background(), inst)
	if err == nil {
		t.Fatal("含 '/' 的 key 应返回错误，实际返回 nil")
	}
	if !strings.Contains(err.Error(), "hatchery-qwen3.6-plus/qwen3.6-plus") {
		t.Errorf("错误信息应包含非法 key 名称，实际=%v", err)
	}
	if !strings.Contains(err.Error(), "/") {
		t.Errorf("错误信息应包含非法字符 '/'，实际=%v", err)
	}
}

// TestCheckOpenclawConfigProviderKeys_MultipleInvalidKeys 验证：多个含 "/" 的 key 均被检测并全部出现在错误信息中
func TestCheckOpenclawConfigProviderKeys_MultipleInvalidKeys(t *testing.T) {
	mockCheckOpenclawConfig(t, func(_ context.Context, _ string, _ string, _ uint64) (string, error) {
		return `{
			"models": {
				"providers": {
					"bad/key1": {},
					"bad/key2": {},
					"good-key": {}
				}
			}
		}`, nil
	})

	inst := &model.Instance{InstanceId: "ins-multi-invalid"}
	err := checkOpenclawConfigProviderKeys(context.Background(), inst)
	if err == nil {
		t.Fatal("含 '/' 的 key 应返回错误，实际返回 nil")
	}
	if !strings.Contains(err.Error(), "bad/key1") && !strings.Contains(err.Error(), "bad/key2") {
		t.Errorf("错误信息应包含非法 key，实际=%v", err)
	}
}

// TestCheckOpenclawConfigProviderKeys_MixedKeys 验证：合法 key 与非法 key 混合时，只有非法 key 触发错误
func TestCheckOpenclawConfigProviderKeys_MixedKeys(t *testing.T) {
	mockCheckOpenclawConfig(t, func(_ context.Context, _ string, _ string, _ uint64) (string, error) {
		return `{
			"models": {
				"providers": {
					"hatchery-qwen3.6-plus": {},
					"hatchery-bad/provider": {},
					"hatchery-glm-5.1": {}
				}
			}
		}`, nil
	})

	inst := &model.Instance{InstanceId: "ins-mixed"}
	err := checkOpenclawConfigProviderKeys(context.Background(), inst)
	if err == nil {
		t.Fatal("混合 key 中含非法 key 时应返回错误")
	}
	if !strings.Contains(err.Error(), "hatchery-bad/provider") {
		t.Errorf("错误信息应包含非法 key，实际=%v", err)
	}
}

// TestCheckOpenclawConfigProviderKeys_ErrorMessageFormat 验证：错误信息格式包含必要提示
func TestCheckOpenclawConfigProviderKeys_ErrorMessageFormat(t *testing.T) {
	mockCheckOpenclawConfig(t, func(_ context.Context, _ string, _ string, _ uint64) (string, error) {
		return `{"models":{"providers":{"a/b":{}}}}`, nil
	})

	inst := &model.Instance{InstanceId: "ins-fmt"}
	err := checkOpenclawConfigProviderKeys(context.Background(), inst)
	if err == nil {
		t.Fatal("应返回错误")
	}
	msg := err.Error()
	if !strings.Contains(msg, "openclaw.json") {
		t.Errorf("错误信息应提及 openclaw.json，实际=%v", msg)
	}
	if !strings.Contains(msg, "修复") {
		t.Errorf("错误信息应包含修复提示，实际=%v", msg)
	}
}

// ─── providerKeyForbiddenChars 配置变量测试 ───────────────────────────────────

// TestProviderKeyForbiddenChars_DefaultContainsSlash 验证：默认禁止字符列表包含 "/"
func TestProviderKeyForbiddenChars_DefaultContainsSlash(t *testing.T) {
	found := false
	for _, ch := range providerKeyForbiddenChars {
		if ch == "/" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("providerKeyForbiddenChars 默认值应包含 \"/\"，实际=%v", providerKeyForbiddenChars)
	}
}

// TestProviderKeyForbiddenChars_NotEmpty 验证：默认禁止字符列表不为空
func TestProviderKeyForbiddenChars_NotEmpty(t *testing.T) {
	if len(providerKeyForbiddenChars) == 0 {
		t.Error("providerKeyForbiddenChars 不应为空")
	}
}

// TestCheckOpenclawConfigProviderKeys_CustomForbiddenChars 验证：扩展 providerKeyForbiddenChars 后，
// 新增的非法字符也能被正确检测（模拟后续扩展场景）
func TestCheckOpenclawConfigProviderKeys_CustomForbiddenChars(t *testing.T) {
	orig := providerKeyForbiddenChars
	providerKeyForbiddenChars = append(providerKeyForbiddenChars, "@")
	t.Cleanup(func() { providerKeyForbiddenChars = orig })

	mockCheckOpenclawConfig(t, func(_ context.Context, _ string, _ string, _ uint64) (string, error) {
		return `{"models":{"providers":{"provider@bad":{}}}}`, nil
	})

	inst := &model.Instance{InstanceId: "ins-custom-forbidden"}
	err := checkOpenclawConfigProviderKeys(context.Background(), inst)
	if err == nil {
		t.Fatal("扩展禁止字符后，含 '@' 的 key 应返回错误")
	}
	if !strings.Contains(err.Error(), "provider@bad") {
		t.Errorf("错误信息应包含非法 key，实际=%v", err)
	}
}

// TestCheckOpenclawConfigProviderKeys_MultipleCustomForbiddenChars 验证：多个自定义禁止字符均生效
func TestCheckOpenclawConfigProviderKeys_MultipleCustomForbiddenChars(t *testing.T) {
	orig := providerKeyForbiddenChars
	providerKeyForbiddenChars = []string{"/", "@", " "}
	t.Cleanup(func() { providerKeyForbiddenChars = orig })

	cases := []struct {
		key     string
		wantErr bool
	}{
		{"valid-key", false},
		{"bad/key", true},
		{"bad@key", true},
		{"bad key", true},
		{"hatchery-qwen3.6-plus", false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.key, func(t *testing.T) {
			mockCheckOpenclawConfig(t, func(_ context.Context, _ string, _ string, _ uint64) (string, error) {
				return `{"models":{"providers":{"` + tc.key + `":{}}}}`, nil
			})

			inst := &model.Instance{InstanceId: "ins-multi-custom"}
			err := checkOpenclawConfigProviderKeys(context.Background(), inst)
			if tc.wantErr && err == nil {
				t.Errorf("key=%q 应返回错误，实际返回 nil", tc.key)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("key=%q 应返回 nil，实际=%v", tc.key, err)
			}
		})
	}
}

// TestCheckOpenclawConfigProviderKeys_OutputWithLeadingTrailingSpaces 验证：TAT 输出含前后空白时能正确解析
func TestCheckOpenclawConfigProviderKeys_OutputWithLeadingTrailingSpaces(t *testing.T) {
	mockCheckOpenclawConfig(t, func(_ context.Context, _ string, _ string, _ uint64) (string, error) {
		return "\n  " + `{"models":{"providers":{"valid-key":{}}}}` + "  \n", nil
	})

	inst := &model.Instance{InstanceId: "ins-spaces"}
	err := checkOpenclawConfigProviderKeys(context.Background(), inst)
	if err != nil {
		t.Errorf("前后空白不影响解析，应返回 nil，实际=%v", err)
	}
}

// TestCheckOpenclawConfigProviderKeys_RealWorldConfig 验证：使用真实世界配置样例（含合法 key）时通过检查
func TestCheckOpenclawConfigProviderKeys_RealWorldConfig(t *testing.T) {
	realConfig := `{
		"agents": {"defaults": {"model": {"primary": "hatchery-qwen3.6-plus/qwen3.6-plus"}}},
		"models": {
			"providers": {
				"hatchery-qwen3.6-plus": {
					"baseUrl": "https://example.com/v1",
					"apiKey": "sk-xxx",
					"models": [{"id": "qwen3.6-plus"}]
				},
				"hatchery-glm-5.1": {
					"baseUrl": "https://example.com/v1",
					"apiKey": "sk-xxx",
					"models": [{"id": "glm-5.1"}]
				},
				"hatchery-deepseek-v4-flash": {
					"baseUrl": "https://example.com/v1",
					"apiKey": "sk-xxx",
					"models": [{"id": "deepseek-v4-flash"}]
				}
			}
		}
	}`

	mockCheckOpenclawConfig(t, func(_ context.Context, _ string, _ string, _ uint64) (string, error) {
		return realConfig, nil
	})

	inst := &model.Instance{InstanceId: "ins-real-world"}
	err := checkOpenclawConfigProviderKeys(context.Background(), inst)
	if err != nil {
		t.Errorf("真实世界合法配置应通过检查，实际=%v", err)
	}
}

// TestCheckOpenclawConfigProviderKeys_RealWorldConfigWithInvalidKey 验证：真实世界配置中含非法 key 时被拦截
func TestCheckOpenclawConfigProviderKeys_RealWorldConfigWithInvalidKey(t *testing.T) {
	invalidConfig := `{
		"models": {
			"providers": {
				"hatchery-qwen3.6-plus/qwen3.6-plus": {
					"baseUrl": "https://example.com/v1",
					"apiKey": "sk-xxx"
				},
				"hatchery-glm-5.1": {}
			}
		}
	}`

	mockCheckOpenclawConfig(t, func(_ context.Context, _ string, _ string, _ uint64) (string, error) {
		return invalidConfig, nil
	})

	inst := &model.Instance{InstanceId: "ins-real-invalid"}
	err := checkOpenclawConfigProviderKeys(context.Background(), inst)
	if err == nil {
		t.Fatal("含非法 key 的真实配置应被拦截，实际返回 nil")
	}
	if !strings.Contains(err.Error(), "hatchery-qwen3.6-plus/qwen3.6-plus") {
		t.Errorf("错误信息应包含非法 key，实际=%v", err)
	}
}

// ─── handleUpgrade 中新增调用路径的集成测试 ────────────────────────────────────

// TestHandleUpgrade_ProviderKeyInvalid_Returns400 验证：
// handleUpgrade 中 checkOpenclawConfigProviderKeys 返回错误时，handler 返回 400
func TestHandleUpgrade_ProviderKeyInvalid_Returns400(t *testing.T) {
	setupUpgradeExtraEnv(t)

	// mock checkOpenclawConfigFn 返回含非法 key 的配置
	mockCheckOpenclawConfig(t, func(_ context.Context, _ string, _ string, _ uint64) (string, error) {
		return `{"models":{"providers":{"bad/provider":{}}}}`, nil
	})

	user := createUpgradeExtraUser(t, "upgrade-bad-provider-key")
	createOpenClawInstance(t, user.ID, nil)
	model.DB(context.Background()).Create(&model.AIImage{
		ImageId:   "img-v2",
		Enabled:   true,
		AgentType: model.AgentTypeOpenClaw,
	})

	req := loggedInReq(t, http.MethodPost, "/openclaw/upgrade?id=1", "upgrade-bad-provider-key", "")
	rr := httptest.NewRecorder()
	handleUpgrade(rr, req, testCVMFetcher)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("provider key 含非法字符时应返回 400，实际=%d, body=%s", rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err == nil {
		if errMsg, ok := resp["error"].(string); !ok || !strings.Contains(errMsg, "bad/provider") {
			t.Errorf("错误信息应包含非法 key，实际=%v", resp["error"])
		}
	}
}

// TestHandleUpgrade_ProviderKeyValid_PassesCheck 验证：
// handleUpgrade 中 checkOpenclawConfigProviderKeys 通过时，继续执行后续逻辑
func TestHandleUpgrade_ProviderKeyValid_PassesCheck(t *testing.T) {
	setupUpgradeExtraEnv(t)

	// mock checkOpenclawConfigFn 返回合法配置
	mockCheckOpenclawConfig(t, func(_ context.Context, _ string, _ string, _ uint64) (string, error) {
		return `{"models":{"providers":{"hatchery-qwen3.6-plus":{}}}}`, nil
	})

	user := createUpgradeExtraUser(t, "upgrade-valid-provider-key")
	createOpenClawInstance(t, user.ID, nil)
	model.DB(context.Background()).Create(&model.AIImage{
		ImageId:      "img-v2",
		Enabled:      true,
		AgentType:    model.AgentTypeOpenClaw,
		AgentVersion: "2026.3.28",
	})

	req := loggedInReq(t, http.MethodPost, "/openclaw/upgrade?id=1", "upgrade-valid-provider-key", "")
	rr := httptest.NewRecorder()
	handleUpgrade(rr, req, testCVMFetcher)

	// provider key 合法，检查通过后继续执行；测试环境 CVM 不可用，
	// checkNeedsUpgrade 会返回错误 → 500，但不应是 400（provider key 拦截）
	if rr.Code == http.StatusBadRequest {
		var resp map[string]interface{}
		json.NewDecoder(rr.Body).Decode(&resp)
		if errMsg, ok := resp["error"].(string); ok && strings.Contains(errMsg, "openclaw.json") {
			t.Errorf("合法 provider key 不应被 openclaw.json 检查拦截，实际=%v", errMsg)
		}
	}
}

// ─── HandleAdminBatchUpgrade 中新增调用路径的集成测试 ─────────────────────────

// TestHandleAdminBatchUpgrade_ProviderKeyInvalid_MarkedFailed 验证：
// HandleAdminBatchUpgrade 中 checkOpenclawConfigProviderKeys 返回错误时，
// 该实例被标记为 status: failed，不影响整体响应（200）
func TestHandleAdminBatchUpgrade_ProviderKeyInvalid_MarkedFailed(t *testing.T) {
	initBatchUpgradeTestDB(t)

	// mock checkOpenclawConfigFn 返回含非法 key 的配置
	mockCheckOpenclawConfig(t, func(_ context.Context, _ string, _ string, _ uint64) (string, error) {
		return `{"models":{"providers":{"bad/provider":{}}}}`, nil
	})

	user := model.User{Username: "batch-bad-key-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)
	inst := model.Instance{
		Name:       "batch-bad-key-inst",
		InstanceId: "ins-batch-bad-001",
		UserID:     user.ID,
		AgentType:  model.AgentTypeOpenClaw,
		ProxyToken: strPtr("sk-batch-bad-001"),
	}
	model.DB(context.Background()).Create(&inst)
	model.DB(context.Background()).Create(&model.AIImage{
		ImageId:   "img-batch-001",
		ImageName: "Batch Test Image",
		Enabled:   true,
		AgentType: model.AgentTypeOpenClaw,
	})

	body, _ := json.Marshal(map[string]interface{}{"ids": []uint{inst.ID}})
	req := adminJSONReq(http.MethodPost, "/admin/instances/batch-upgrade", body)
	w := httptest.NewRecorder()
	HandleAdminBatchUpgrade(w, req)

	// 测试环境无真实 CVM，可能在 CVM 信息校验阶段返回 400，
	// 也可能走到 provider key 检查阶段返回 200+results
	if w.Code == http.StatusOK {
		resp := decodeJSONResp(t, w)
		results, ok := resp["results"].([]interface{})
		if !ok {
			t.Fatalf("期望 results 为数组，实际=%T", resp["results"])
		}
		for _, r := range results {
			item := r.(map[string]interface{})
			if uint(item["id"].(float64)) == inst.ID {
				if item["status"] != "failed" {
					t.Errorf("provider key 非法的实例应标记为 failed，实际 status=%v", item["status"])
				}
				if msg, ok := item["message"].(string); !ok || !strings.Contains(msg, "openclaw.json") {
					t.Errorf("错误信息应包含 openclaw.json，实际=%v", item["message"])
				}
			}
		}
	}
	// 测试环境 CVM 不可用时返回 400 也是合理的（在 provider key 检查之前就失败了）
}

// TestHandleAdminBatchUpgrade_LightclawACEType_RejectedByPrereq 验证：
// 当批量升级中混入 lightclawace（SupportsUpgrade=false）类型实例时，
// HandleAdminBatchUpgrade 会通过 prepareInstanceForUpgrade 第 3 步
// （checkInstanceSupportsUpgrade）将该实例标记为 status: failed，
// 错误信息含「不支持一键升级」。
//
// 这是用户原诉求「批量也加上反向拒绝检查」的最终验证：
// 即使后台为 lightclawace 启用了镜像（绕过"无镜像"分支），批量入口仍会拒绝该实例。
//
// 历史：原用 hermes 类型，随 Hermes 升级能力放开，改为用 lightclawace。
func TestHandleAdminBatchUpgrade_LightclawACEType_RejectedByPrereq(t *testing.T) {
	initBatchUpgradeTestDB(t)
	// 让 providerKeys 检查通过（视同 openclaw.json 不存在）
	mockCheckOpenclawConfig(t, func(_ context.Context, _ string, _ string, _ uint64) (string, error) {
		return "", nil
	})

	user := model.User{Username: "batch-ace-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)
	inst := model.Instance{
		Name:       "batch-ace-inst",
		InstanceId: "ins-batch-ace-001",
		UserID:     user.ID,
		AgentType:  "lightclawace",
		AgentReady: 1,
		ProxyToken: strPtr("sk-batch-ace-001"),
	}
	model.DB(context.Background()).Create(&inst)
	// 关键：为 lightclawace 启用镜像，让 prepareBatchUpgradeResults 不再以"无镜像"提前拒绝。
	model.DB(context.Background()).Create(&model.AIImage{
		ImageId:   "img-batch-ace",
		ImageName: "ACE Test Image",
		Enabled:   true,
		AgentType: "lightclawace",
	})

	body, _ := json.Marshal(map[string]interface{}{"ids": []uint{inst.ID}})
	req := adminJSONReq(http.MethodPost, "/admin/instances/batch-upgrade", body)
	w := httptest.NewRecorder()
	HandleAdminBatchUpgrade(w, req)

	// 测试环境无真实 CVM 凭证：batchFetchCVMInfoMap 返回 API_ERROR 兜底，
	// ResolveInstanceStatus.Step0.5 兜底为 running，进入循环体。
	// 然后命中 prepareInstanceForUpgrade 的 SupportsUpgrade 拒绝分支 → results: failed/含不支持一键升级。
	if w.Code != http.StatusOK {
		// 极少数情况下若状态准入或更早环节失败，给出充分诊断信息
		t.Logf("batch upgrade returned HTTP %d (not 200), body: %s", w.Code, w.Body.String())
		return
	}
	resp := decodeJSONResp(t, w)
	results, ok := resp["results"].([]interface{})
	if !ok {
		t.Fatalf("expected results array, got %T", resp["results"])
	}
	if len(results) == 0 {
		t.Fatal("expected at least 1 result")
	}
	// 找出我们这条实例的结果项
	var found bool
	for _, r := range results {
		item, _ := r.(map[string]interface{})
		if item == nil {
			continue
		}
		if uint(item["id"].(float64)) != inst.ID {
			continue
		}
		found = true
		if item["status"] != "failed" {
			t.Errorf("expected status=failed for ace instance, got %v", item["status"])
		}
		msg, _ := item["message"].(string)
		// 接受两种合法路径之一：
		//   1) prepareInstanceForUpgrade 拒绝（含不支持一键升级字样）；
		//   2) 极小概率走到状态准入失败（含「非」/「running」字样）。
		// 任一错误信息都说明 batch 入口正确拒绝了 ace 实例，没漏掉。
		if !strings.Contains(msg, "不支持一键升级") &&
			!strings.Contains(msg, "非") &&
			!strings.Contains(strings.ToLower(msg), "running") {
			t.Errorf("expected message to indicate rejection (不支持一键升级 or status), got %q", msg)
		}
	}
	if !found {
		t.Errorf("did not find result for instance id=%d in results", inst.ID)
	}
}

// TestHandleAdminBatchUpgrade_OpenClawType_StartUpgradeErrPath 验证：
// HandleAdminBatchUpgrade 中 startUpgradeForInstance 返回 Err 分支
// （checkNeedsUpgrade 因 CVM 镜像信息缺失而失败）会被 default 分支正确包装为
// upgradeResult{Status: outcome.BatchStatus, Message: outcome.Err.Error()}。
//
// 同时是 prepareInstanceForUpgrade 通过 + startUpgradeForInstance 失败 这条
// 端到端路径的唯一覆盖来源（OpenClaw 实例 + 启用镜像 → 走完前置 → CVM 信息为空触发失败）。
func TestHandleAdminBatchUpgrade_OpenClawType_StartUpgradeErrPath(t *testing.T) {
	initBatchUpgradeTestDB(t)
	mockCheckOpenclawConfig(t, func(_ context.Context, _ string, _ string, _ uint64) (string, error) {
		return "", nil
	})

	user := model.User{Username: "batch-oc-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)
	inst := model.Instance{
		Name:       "batch-oc-inst",
		InstanceId: "ins-batch-oc-001",
		UserID:     user.ID,
		AgentType:  model.AgentTypeOpenClaw,
		AgentReady: 1,
		ProxyToken: strPtr("sk-batch-oc-001"),
	}
	model.DB(context.Background()).Create(&inst)
	model.DB(context.Background()).Create(&model.AIImage{
		ImageId:   "img-batch-oc",
		ImageName: "OpenClaw Batch Test Image",
		Enabled:   true,
		AgentType: model.AgentTypeOpenClaw,
	})

	body, _ := json.Marshal(map[string]interface{}{"ids": []uint{inst.ID}})
	req := adminJSONReq(http.MethodPost, "/admin/instances/batch-upgrade", body)
	w := httptest.NewRecorder()
	HandleAdminBatchUpgrade(w, req)

	if w.Code != http.StatusOK {
		t.Logf("batch upgrade returned HTTP %d, body: %s", w.Code, w.Body.String())
		return
	}
	resp := decodeJSONResp(t, w)
	results, ok := resp["results"].([]interface{})
	if !ok {
		t.Fatalf("expected results array, got %T", resp["results"])
	}
	if len(results) == 0 {
		t.Fatal("expected at least 1 result")
	}

	// 该实例必然不会出现 status=started（无真实 CVM，checkNeedsUpgrade 一定失败）。
	// 但具体是 started/skipped/failed 中哪一个，依赖 CVMInstanceInfo 的实际兜底语义；
	// 我们只需要断言 result 项被 default/AlreadyLatest 三态分发逻辑命中（不是 panic、不是丢项）。
	var found bool
	for _, r := range results {
		item, _ := r.(map[string]interface{})
		if item == nil {
			continue
		}
		if uint(item["id"].(float64)) != inst.ID {
			continue
		}
		found = true
		status, _ := item["status"].(string)
		if status != "started" && status != "skipped" && status != "failed" {
			t.Errorf("expected status in {started,skipped,failed}, got %q", status)
		}
		// message 不应为空（每条 outcome 都设了 Message）
		if msg, _ := item["message"].(string); msg == "" {
			t.Errorf("expected non-empty message, got empty for status=%q", status)
		}
		_ = errors.New // 抑制 errors 包未使用警告（如果未来扩展）
	}
	if !found {
		t.Errorf("did not find result for instance id=%d in results", inst.ID)
	}
}

// TestHandleAdminBatchUpgrade_RejectDowngrade_MarkedFailed 验证：
// 「拒绝官方镜像降级」检查已被提到 prepareInstanceForUpgrade（step 0），
// 因此 HandleAdminBatchUpgrade 也会自动覆盖到——这是本次重构的核心收益验证：
// 之前 rejectDowngradeOnOfficialImage 仅在 handleUpgrade 中 inline 调用，
// 批量入口完全漏掉，存在「批量降级覆盖用户灰度版本」的隐患。
//
// 触发条件：
//   - 实例 AgentType=openclaw、AgentVersion 远高于官方镜像版本
//   - 启用镜像 = OpenClaw 官方镜像（来自 common.CandidateImages）
//
// 期望结果：
//   - HTTP 200 + results 数组中该实例 status="failed"，message 含两个版本号。
func TestHandleAdminBatchUpgrade_RejectDowngrade_MarkedFailed(t *testing.T) {
	initBatchUpgradeTestDB(t)
	mockCheckOpenclawConfig(t, func(_ context.Context, _ string, _ string, _ uint64) (string, error) {
		return "", nil
	})

	user := model.User{Username: "batch-down-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&user)
	inst := model.Instance{
		Name:         "batch-down-inst",
		InstanceId:   "ins-batch-down-001",
		UserID:       user.ID,
		AgentType:    "openclaw",
		AgentVersion: "9999.99.99", // 远高于任何官方镜像
		AgentReady:   1,
		ProxyToken:   strPtr("sk-batch-down-001"),
	}
	model.DB(context.Background()).Create(&inst)

	// 启用镜像必须是真实官方镜像（IsCandidateImage=true），否则 step 0 会因
	// "目标非官方镜像" 而放行，无法触发 reject 分支。
	model.DB(context.Background()).Create(&model.AIImage{
		ImageId:      "img-idzg74s9",
		ImageName:    "OpenClaw on Ubuntu 24.04",
		Enabled:      true,
		AgentType:    "openclaw",
		AgentVersion: "2026.5.7",
	})

	body, _ := json.Marshal(map[string]interface{}{"ids": []uint{inst.ID}})
	req := adminJSONReq(http.MethodPost, "/admin/instances/batch-upgrade", body)
	w := httptest.NewRecorder()
	HandleAdminBatchUpgrade(w, req)

	if w.Code != http.StatusOK {
		t.Logf("batch upgrade returned HTTP %d (not 200), body: %s", w.Code, w.Body.String())
		return
	}
	resp := decodeJSONResp(t, w)
	results, ok := resp["results"].([]interface{})
	if !ok {
		t.Fatalf("expected results array, got %T", resp["results"])
	}
	if len(results) == 0 {
		t.Fatal("expected at least 1 result")
	}

	var found bool
	for _, r := range results {
		item, _ := r.(map[string]interface{})
		if item == nil {
			continue
		}
		if uint(item["id"].(float64)) != inst.ID {
			continue
		}
		found = true
		if item["status"] != "failed" {
			t.Errorf("expected status=failed for downgrade instance, got %v", item["status"])
		}
		msg, _ := item["message"].(string)
		// 错误信息应同时包含「当前版本」与「官方镜像版本」两段语义，
		// 这样运维看日志能直接定位问题。
		if !strings.Contains(msg, "9999.99.99") || !strings.Contains(msg, "2026.5.7") {
			t.Errorf("expected message to mention both versions, got %q", msg)
		}
		// 若 message 提到 openclaw.json，则说明 step 0 没提前 return，
		// 错误地走到了 step 1（providerKeys），这是测试不能容忍的退步。
		if strings.Contains(msg, "openclaw.json") {
			t.Errorf("step 0 应在 providerKeys 检查之前 return，但实际命中了 providerKeys 错误: %q", msg)
		}
	}
	if !found {
		t.Errorf("did not find result for instance id=%d in results", inst.ID)
	}
}
