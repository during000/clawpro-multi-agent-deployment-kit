package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	hcommon "hatchery/common"
	"hatchery/controller/provider"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm"
)

func maskModelAPIKey(apiKey string) string {
	if apiKey == "" {
		return ""
	}
	chars := []rune(apiKey)
	if len(chars) < 12 {
		return strings.Repeat("*", len(chars))
	}
	return string(chars[:4]) + strings.Repeat("*", len(chars)-8) + string(chars[len(chars)-4:])
}

func queryAllModels(ctx context.Context) []model.AIModel {
	var models []model.AIModel
	model.DB(ctx).Order("id desc").Find(&models)
	return models
}

// modelWithVisibility 是 /admin/models 列表响应中单条模型记录的对外结构，
// 在嵌入 model.AIModel 的基础上追加 visibility_groups 字段。
//
// 注意：由于 model.AIModel 自己实现了 MarshalJSON（用于兼容旧前端的
// Enabled / Visible 字段语义），Go encoding/json 会把内嵌类型的
// json.Marshaler 接口"提升"到外层结构体，从而直接调用 AIModel.MarshalJSON
// 并**忽略外层并列字段**（包括这里的 VisibilityGroups），导致响应里看不到
// visibility_groups。为此，本结构体必须显式实现 MarshalJSON，先按 AIModel
// 的规则序列化内嵌部分，再把 visibility_groups 注入到结果 map 中。
type modelWithVisibility struct {
	model.AIModel
	VisibilityGroups []visibilityGroupInfo `json:"visibility_groups"`
}

// MarshalJSON 自定义序列化：合并 AIModel.MarshalJSON 的输出 + visibility_groups。
//
// 实现思路：先调用 AIModel 自己的 MarshalJSON 拿到 base JSON（含旧前端兼容字段），
// 再 unmarshal 成 map，注入 visibility_groups，最后重新 marshal 输出。
// admin 列表数据量小，额外的 marshal/unmarshal 开销可忽略。
func (mv modelWithVisibility) MarshalJSON() ([]byte, error) {
	base, err := json.Marshal(mv.AIModel)
	if err != nil {
		return nil, err
	}
	var obj map[string]interface{}
	if err := json.Unmarshal(base, &obj); err != nil {
		return nil, err
	}
	obj["APIKey"] = maskModelAPIKey(mv.APIKey)
	// 即使为空也输出 [] 而非缺失字段，避免前端判空逻辑出错。
	if mv.VisibilityGroups == nil {
		obj["visibility_groups"] = []visibilityGroupInfo{}
	} else {
		obj["visibility_groups"] = mv.VisibilityGroups
	}
	return json.Marshal(obj)
}

func HandleAdminModels(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	config := model.GetSiteConfig(r.Context())
	models := queryAllModels(r.Context())

	// 批量查询可见性分组信息（固定 3 次 DB 查询，无 N+1 问题）
	visibilityData := buildVisibilityData(r.Context(), models)

	result := make([]modelWithVisibility, 0, len(models))
	for _, m := range models {
		mv := modelWithVisibility{
			AIModel:          m,
			VisibilityGroups: make([]visibilityGroupInfo, 0),
		}
		if groups, ok := visibilityData[m.ID]; ok {
			mv.VisibilityGroups = groups
		}
		result = append(result, mv)
	}

	jsonOK(w, map[string]interface{}{
		"models":           result,
		"default_model_id": config.DefaultModelID,
	})
}

func HandleCreateModel(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	r.ParseForm()

	provider := r.FormValue("provider")
	if provider == "" {
		provider = hcommon.CustomModelProvider
	}
	modelID := r.FormValue("model_id")
	modelName := r.FormValue("model_name")
	if modelName == "" {
		modelName = modelID
	}
	apiKey := r.FormValue("api_key")
	url := r.FormValue("url")
	modelType := r.FormValue("model_type")
	quotaDay, err := strconv.Atoi(r.FormValue("quota_day"))
	if err != nil || quotaDay < -1 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgQuotaDayInvalid))
		return
	}
	contextLen, _ := strconv.Atoi(r.FormValue("context_len"))
	if contextLen <= 0 {
		contextLen = 128000
	}
	maxTokensStr := r.FormValue("max_tokens")
	maxTokens, err := strconv.Atoi(maxTokensStr)
	if maxTokensStr != "" && err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgMaxTokensMustBeInteger))
		return
	}
	// max_tokens 选填：不传或传 0 表示不限，传正值用用户值，负数非法。
	if maxTokens < 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMaxTokensMustBeNonNegative))
		return
	}

	customHTTPHeaders, err := hcommon.ValidateAndParseCustomHTTPHeaders(r.FormValue("custom_http_headers"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	var customHTTPHeadersJSON string
	if customHTTPHeaders != nil {
		b, _ := json.Marshal(customHTTPHeaders)
		customHTTPHeadersJSON = string(b)
	}

	inputTypes, err := hcommon.ValidateInputTypes(r.Form["input_types"])
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	if modelID == "" || apiKey == "" || url == "" || modelType == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgModelRequiredFieldsCreate))
		return
	}

	if err = hcommon.ValidateHTTPURL(url); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	if err = hcommon.ValidateModelType(modelType); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	inputTypesJSON, _ := json.Marshal(inputTypes)

	// 新建模型默认 Enabled=true（启用）+ Visible=false（不向用户展示），
	// 由管理员确认配置无误后再手动开启「用户可见」。
	m := model.AIModel{
		Provider:          provider,
		ModelID:           modelID,
		ModelName:         modelName,
		APIKey:            apiKey,
		URL:               url,
		ModelType:         modelType,
		InputTypes:        string(inputTypesJSON),
		ContextLen:        contextLen,
		MaxTokens:         maxTokens,
		CustomHTTPHeaders: customHTTPHeadersJSON,
		QuotaDay:          quotaDay,
		Enabled:           true,
		Visible:           false,
	}
	if result := model.DB(r.Context()).Create(&m); result.Error != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgCreateModelFailed))
		return
	}

	jsonOK(w, map[string]interface{}{"ok": true})
}

func HandleUpdateModel(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	r.ParseForm()

	id := r.URL.Query().Get("id")
	var m model.AIModel
	if model.DB(r.Context()).Where("id = ?", id).First(&m).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgAdminModelNotFound))
		return
	}

	if isCustomModelRecord(&m) {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgBuiltinModelCannotModify))
		return
	}

	modelID := r.FormValue("model_id")
	modelName := r.FormValue("model_name")
	if modelName == "" {
		modelName = modelID
	}
	apiKey := r.FormValue("api_key")
	url := r.FormValue("url")
	modelType := r.FormValue("model_type")
	quotaDay, err := strconv.Atoi(r.FormValue("quota_day"))
	if err != nil || quotaDay < -1 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgQuotaDayInvalid))
		return
	}
	contextLen, _ := strconv.Atoi(r.FormValue("context_len"))
	if contextLen <= 0 {
		contextLen = 128000
	}
	maxTokensStr := r.FormValue("max_tokens")
	// 区分「未传」与「显式传值」：未传则保留原值；显式传值（含 0）则更新，0 表示不限输出。
	maxTokensSet := maxTokensStr != ""
	var maxTokens int
	if maxTokensSet {
		maxTokens, err = strconv.Atoi(maxTokensStr)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgMaxTokensMustBeInteger))
			return
		}
		if maxTokens < 0 {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMaxTokensMustBeNonNegative))
			return
		}
	}
	// 区分「未传」与「显式传值」：原始串为空表示未传，保留原值；
	// 否则按解析结果写入（空对象序列化为空串，即「无自定义头」的规范表示，与 create 一致）。
	rawCustomHeaders := r.FormValue("custom_http_headers")
	customHeadersSet := strings.TrimSpace(rawCustomHeaders) != ""
	customHTTPHeaders, err := hcommon.ValidateAndParseCustomHTTPHeaders(rawCustomHeaders)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	var customHTTPHeadersJSON string
	if customHTTPHeaders != nil {
		b, _ := json.Marshal(customHTTPHeaders)
		customHTTPHeadersJSON = string(b)
	}
	inputTypes, err := hcommon.ValidateInputTypes(r.Form["input_types"])
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	if modelID == "" || url == "" || modelType == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgModelRequiredFieldsUpdate))
		return
	}

	if err := hcommon.ValidateHTTPURL(url); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	if err := hcommon.ValidateModelType(modelType); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	inputTypesJSON, _ := json.Marshal(inputTypes)

	updates := map[string]interface{}{
		"model_id":    modelID,
		"model_name":  modelName,
		"url":         url,
		"model_type":  modelType,
		"input_types": string(inputTypesJSON),
		"context_len": contextLen,
		"quota_day":   quotaDay,
	}
	// 显式传入自定义 HTTP 头才更新，未传则保留原值；空对象写入空串，与「无自定义头」等价。
	if customHeadersSet {
		updates["custom_http_headers"] = customHTTPHeadersJSON
	}
	// 显式传入 max_tokens（含 0）才更新；0 表示不限输出，未传则保留原值。
	if maxTokensSet {
		updates["max_tokens"] = maxTokens
	}
	if apiKey != "" {
		updates["api_key"] = apiKey
	}

	if err := model.DB(r.Context()).Model(&m).Updates(updates).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgUpdateFailed))
		return
	}

	updatedModel := m
	updatedModel.ModelID = modelID
	updatedModel.ModelName = modelName
	updatedModel.URL = url
	updatedModel.ModelType = modelType
	updatedModel.InputTypes = string(inputTypesJSON)
	updatedModel.ContextLen = contextLen
	updatedModel.QuotaDay = quotaDay
	if customHeadersSet {
		updatedModel.CustomHTTPHeaders = customHTTPHeadersJSON
	}
	if maxTokensSet {
		updatedModel.MaxTokens = maxTokens
	}
	if apiKey != "" {
		updatedModel.APIKey = apiKey
	}

	// 更新 model 后，同步到所有绑定该 model 的实例（含 primary 与 fallback），
	// 确保实例 CVM 侧 openclaw.json 的 provider 配置（URL/APIKey/ModelType 等）与 DB 一致。
	// 复用 injectModelConfigToCVM 下发 set_model.sh，全量重写 provider 配置 + primary/fallback 链。
	// 异步执行，失败仅记日志（与 HandleDeleteModel 行为对齐），不回滚 DB。
	var affectedInstanceIDs []uint
	if err := model.DB(r.Context()).Model(&model.InstanceModel{}).
		Where("ai_model_id = ?", m.ID).
		Distinct("instance_id").
		Pluck("instance_id", &affectedInstanceIDs).Error; err != nil {
		// 查询失败不影响 DB 更新成功，仅记日志（实例配置可能滞后，可通过重新更新触发同步）
		slog.Error("[AdminUpdateModel] 查询受影响实例失败，跳过同步", "model_id", m.ID, "error", err)
		jsonOK(w, map[string]interface{}{"ok": true})
		return
	}

	if len(affectedInstanceIDs) > 0 {
		slog.Info("[AdminUpdateModel] 同步到已绑定实例",
			"model_id", updatedModel.ID, "model_name", updatedModel.ModelName, "affected_instances", len(affectedInstanceIDs))

		go func(ctx context.Context, aim model.AIModel) {
			const maxConcurrency = 10
			sem := make(chan struct{}, maxConcurrency)
			var wg sync.WaitGroup
			// HandleUpdateModel 只处理 ai_models 中的管控端模型，必须统一走 hatchery proxy。
			for _, instID := range affectedInstanceIDs {
				wg.Add(1)
				sem <- struct{}{}
				go func(id uint) {
					defer wg.Done()
					defer func() { <-sem }()
					var instance model.Instance
					if err := model.DB(ctx).First(&instance, id).Error; err != nil {
						slog.Error("[AdminUpdateModel] 查询实例失败，跳过 TAT 下发",
							"instance_id", id, "error", err)
						return
					}
					if err := injectModelConfigToCVM(ctx, &instance, &aim, false); err != nil {
						slog.Error("[AdminUpdateModel] CVM 同步失败",
							"error", err, "instance_id", instance.InstanceId)
					}
				}(instID)
			}
			wg.Wait()
		}(hcommon.DetachContext(r.Context()), updatedModel)
	}

	jsonOK(w, map[string]interface{}{"ok": true})
}

func isCustomModelRecord(m *model.AIModel) bool {
	return m.Provider == model.BuiltinModelProvider && m.ModelID == model.BuiltinModelID
}

func HandleDeleteModel(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	id := r.URL.Query().Get("id")
	var m model.AIModel
	if model.DB(r.Context()).Where("id = ?", id).First(&m).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgAdminModelNotFound))
		return
	}

	if isCustomModelRecord(&m) {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgBuiltinModelCannotDelete))
		return
	}

	// 删除前，若该模型是默认模型则联动清除
	config := model.GetSiteConfig(r.Context())
	if config.DefaultModelID == m.ID {
		model.DB(r.Context()).Model(&config).Update("default_model_id", 0)
	}

	// 在事务中删除模型及其关联数据
	var affectedInstanceIDs []uint
	if err := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		// 1) 清理可见性关联
		if err := model.CleanupVisibilityByModelID(tx, m.ID); err != nil {
			return err
		}
		// 2) 清理 instance_models 绑定记录，自动处理 primary 提升和 instances.ai_model_id 同步
		var err error
		affectedInstanceIDs, err = model.CleanupInstanceModelsByAIModelID(tx, m.ID)
		if err != nil {
			return hcommon.I18nRichError(err, i18n.MsgCleanupInstanceModelBindingFailed).WithDetail(err.Error())
		}
		// 3) 删除模型本身
		if err := tx.Delete(&m).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		slog.Error("[AdminDeleteModel] 删除模型事务失败", "model_id", m.ID, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgDeleteModelFailed))
		return
	}

	// 【新增】异步同步 CVM 配置，限制最大并发数避免资源耗尽
	go func(ctx context.Context) {
		const maxConcurrency = 10
		// providerKey 必须与 buildSetModelParams / resolveBindingRef 保持完全一致
		providerKey := setModelProviderKey(m, false)
		sem := make(chan struct{}, maxConcurrency)
		var wg sync.WaitGroup
		for _, instID := range affectedInstanceIDs {
			wg.Add(1)
			sem <- struct{}{}
			go func(id uint) {
				defer wg.Done()
				defer func() { <-sem }()
				var instance model.Instance
				if model.DB(ctx).First(&instance, id).Error != nil {
					slog.Error("[AdminDeleteModel] 查询实例失败，跳过 TAT 下发",
						"instance_id", id)
					return
				}
				if err := syncInstanceModelsToCVM(ctx, &instance, providerKey); err != nil {
					slog.Error("[AdminDeleteModel] CVM 同步失败",
						"error", err, "instance_id", instance.InstanceId)
				}
			}(instID)
		}
		wg.Wait()
	}(hcommon.DetachContext(r.Context()))

	slog.Info("[AdminDeleteModel] 模型已删除", "model_id", m.ID, "model_name", m.ModelName)

	jsonOK(w, map[string]interface{}{"ok": true})
}

// HandleToggleModel 切换模型「用户可见」状态（visible 字段）。
//
// 注意：此开关仅控制用户端列表是否展示该模型，不影响已绑定 agent 的可用性。
// 模型是否可用由 Enabled 字段控制（HandleToggleModelEnabled）。
func HandleToggleModel(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	id := r.URL.Query().Get("id")
	var m model.AIModel
	if model.DB(r.Context()).Where("id = ?", id).First(&m).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgAdminModelNotFound))
		return
	}

	// 注意：GORM 的 Model(&m).Update(col, v) 会回写 m 的字段为新值，
	// 因此必须先把旧值拷出来，后续判断「本次是否为关闭操作」要用旧值。
	wasVisible := m.Visible
	model.DB(r.Context()).Model(&m).Update("visible", !wasVisible)

	// 关闭可见时，若该模型是默认模型则联动清除（避免新建实例自动注入用户已不可见的模型）
	if wasVisible {
		config := model.GetSiteConfig(r.Context())
		if config.DefaultModelID == m.ID {
			model.DB(r.Context()).Model(&config).Update("default_model_id", 0)
		}
	}

	jsonOK(w, map[string]interface{}{"ok": true})
}

// HandleToggleModelEnabled 切换模型「启用」状态（enabled 字段）。
//
// 关闭后，已绑定该模型的 agent 在 LLM 代理路由时会被拒绝（与历史行为一致）；
// 同时该模型也无法被新用户绑定。如果该模型是默认模型，会联动清除默认状态。
func HandleToggleModelEnabled(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	var m model.AIModel
	if model.DB(r.Context()).Where("id = ?", id).First(&m).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgAdminModelNotFound))
		return
	}

	// 注意：GORM 的 Model(&m).Update(col, v) 会回写 m 的字段为新值，
	// 因此必须先把旧值拷出来，后续判断「本次是否为关闭操作」要用旧值。
	wasEnabled := m.Enabled
	model.DB(r.Context()).Model(&m).Update("enabled", !wasEnabled)

	// 关闭启用时，若该模型是默认模型则联动清除
	if wasEnabled {
		config := model.GetSiteConfig(r.Context())
		if config.DefaultModelID == m.ID {
			model.DB(r.Context()).Model(&config).Update("default_model_id", 0)
		}
	}

	jsonOK(w, map[string]interface{}{"ok": true})
}

func HandleToggleDefault(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	id := r.URL.Query().Get("id")
	var m model.AIModel
	if model.DB(r.Context()).Where("id = ?", id).First(&m).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgAdminModelNotFound))
		return
	}

	config := model.GetSiteConfig(r.Context())

	if config.DefaultModelID == m.ID {
		// 当前是默认 → 取消默认
		model.DB(r.Context()).Model(&config).Update("default_model_id", 0)
	} else {
		// 设为默认 → 校验模型必须可用（启用）且对用户可见
		if !m.Enabled {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgEnableEnabledBeforeDefault))
			return
		}
		if !m.Visible {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgEnableModelBeforeDefault))
			return
		}
		model.DB(r.Context()).Model(&config).Update("default_model_id", m.ID)
	}

	jsonOK(w, map[string]interface{}{"ok": true})
}

// HandleUpdateModelVisibility 更新模型的可见范围。
// POST /admin/models/visibility?id=N
func HandleUpdateModelVisibility(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	// 获取当前操作的管理员用户名（用于日志）
	adminUser := ""
	if user, err := RequestUser(r); user != nil && err == nil {
		adminUser = user.Username
	}

	id := r.URL.Query().Get("id")
	var m model.AIModel
	if model.DB(r.Context()).Where("id = ?", id).First(&m).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgAdminModelNotFound))
		return
	}

	var req struct {
		VisibilityType string `json:"visibility_type"`
		GroupIDs       []uint `json:"group_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidRequestFormat))
		return
	}

	if req.VisibilityType != model.VisibilityAll && req.VisibilityType != model.VisibilityGroup {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidVisibilityForModel))
		return
	}

	if req.VisibilityType == model.VisibilityGroup {
		if len(req.GroupIDs) == 0 {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgGroupRequiredForVisibility))
			return
		}
		// 校验所有 group_ids 在分组表中存在
		groups, err := model.GetGroupsByIDs(r.Context(), req.GroupIDs)
		if err != nil {
			slog.Error("[ModelVisibility] 查询分组失败", "error", err)
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgQueryGroupFailed))
			return
		}
		existIDs := make(map[uint]bool)
		for _, g := range groups {
			existIDs[g.ID] = true
		}
		var missing []uint
		for _, gid := range req.GroupIDs {
			if !existIDs[gid] {
				missing = append(missing, gid)
			}
		}
		if len(missing) > 0 {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgGroupNotFoundList, missing))
			return
		}
	}

	// 在同一个事务中执行：删除旧关联 + 创建新关联 + 更新 VisibilityType
	if err := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		// 删除该模型的所有旧关联
		if err := tx.Where("ai_model_id = ?", m.ID).Delete(&model.ModelVisibilityGroup{}).Error; err != nil {
			return err
		}
		// 如果是 group 类型，批量创建新关联
		if req.VisibilityType == model.VisibilityGroup {
			for _, gid := range req.GroupIDs {
				if err := tx.Create(&model.ModelVisibilityGroup{
					AIModelID: m.ID,
					GroupID:   gid,
				}).Error; err != nil {
					return err
				}
			}
		}
		// 更新 VisibilityType
		if err := tx.Model(&m).Update("visibility_type", req.VisibilityType).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		slog.Error("[ModelVisibility] 更新失败", "model_id", m.ID, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgUpdateVisibilityFailed))
		return
	}

	slog.Info("[ModelVisibility] 可见范围已更新",
		"model_id", m.ID,
		"visibility_type", req.VisibilityType,
		"group_ids", fmt.Sprintf("%v", req.GroupIDs),
		"admin", adminUser,
	)

	jsonOK(w, map[string]interface{}{"ok": true})
}

// visibilityGroupInfo 用于管理端模型列表响应中的可见性分组信息。
type visibilityGroupInfo struct {
	GroupID   uint   `json:"group_id"`
	GroupName string `json:"group_name"`
}

// buildVisibilityData 批量构建模型列表的可见性分组数据。
// 返回 map[modelID][]visibilityGroupInfo（含 group_id + group_name）。
// 固定 2 次额外 DB 查询（查关联 + 查分组名称），无 N+1 问题。
func buildVisibilityData(ctx context.Context, models []model.AIModel) map[uint][]visibilityGroupInfo {
	result := make(map[uint][]visibilityGroupInfo)

	// 筛出 visibility_type="group" 的模型 ID
	var groupModelIDs []uint
	for _, m := range models {
		if m.VisibilityType == model.VisibilityGroup {
			groupModelIDs = append(groupModelIDs, m.ID)
		}
	}
	if len(groupModelIDs) == 0 {
		return result
	}

	// 批量查出所有关联
	modelGroupMap, rerr := model.GetModelVisibilityGroupIDs(ctx, groupModelIDs)
	if rerr != nil {
		slog.Error("[ModelVisibility] 批量查询模型分组关联失败", "error", rerr)
		return result
	}

	// 收集去重的 group_id
	groupIDSet := make(map[uint]bool)
	for _, gids := range modelGroupMap {
		for _, gid := range gids {
			groupIDSet[gid] = true
		}
	}
	if len(groupIDSet) == 0 {
		return result
	}
	allGroupIDs := make([]uint, 0, len(groupIDSet))
	for gid := range groupIDSet {
		allGroupIDs = append(allGroupIDs, gid)
	}

	// 批量查分组名称
	groups, err := model.GetGroupsByIDs(ctx, allGroupIDs)
	if err != nil {
		slog.Error("[ModelVisibility] 批量查询分组名称失败", "error", err)
		return result
	}
	groupNameMap := make(map[uint]string)
	for _, g := range groups {
		groupNameMap[g.ID] = g.Name
	}

	// 组装结果
	for modelID, gids := range modelGroupMap {
		for _, gid := range gids {
			name, ok := groupNameMap[gid]
			if !ok {
				// 防御性处理：分组已被软删除但关联未清理，跳过并记录 Warn 日志
				slog.Warn("[ModelVisibility] 分组已不存在，跳过", "group_id", gid, "model_id", modelID)
				continue
			}
			result[modelID] = append(result[modelID], visibilityGroupInfo{GroupID: gid, GroupName: name})
		}
	}
	return result
}

// ── 模型连通性检测 ────────────────────────────────────────────────────────
//
// HandleAdminModelConnectivity 探测某个模型上游的连通性与 API Key 是否有效。
//
// 路由：POST /admin/models/connectivity
//
// 入参（两种用法，二选一）：
//
//  1. 按已保存模型 ID 探测——常用于"已存在模型的健康检查"：
//     POST /admin/model/connectivity?id=42
//
//  2. 用临时凭证探测——常用于"新增/编辑模型表单"未保存即试连：
//     POST /admin/model/connectivity
//     Content-Type: application/json
//     {"url":"https://api.openai.com/v1","api_key":"sk-...","model_type":"openai-completions"}
//
// 响应（HTTP 200 始终代表后端逻辑正常完成）：
//
//	成功：{"ok":true, "latency_ms":234}
//	失败：{"ok":false, "kind":"invalid_api_key", "message":"...",
//	       "status_code":401, "snippet":"...", "latency_ms":120}
//
// 仅当本地参数错误时才返回 4xx；上游连不通/凭证无效不会返回 4xx，由 ok=false
// 表达，便于前端根据 kind 字段本地化错误文案。
func HandleAdminModelConnectivity(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	handleModelConnectivity(w, r, nil)
}

// user 为 nil 时，表示是管理员端探测；否则是用户端探测。
func handleModelConnectivity(w http.ResponseWriter, r *http.Request, user *model.User) {
	apiBase, apiKey, modelType, modelID, err := resolveConnectivityArgs(r, user)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			writeError(w, r, http.StatusForbidden, ErrForbidden)
		} else {
			writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		}
		return
	}

	if err := hcommon.ValidateModelType(modelType); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if err := hcommon.ValidateHTTPURL(apiBase); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 用稍长的 ctx timeout 兜底，防止上游半死状态拖死调用方。
	// provider 内部本身已有 10~15s 超时，这里再加 30s 上限即可。
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	p := provider.GetProvider(modelType)

	// 全部采用 chat 探活，确保 model id 正确性也能被验证
	latency, probeErr := p.CheckConnectivityWithChat(ctx, apiKey, apiBase, modelID)

	latencyMs := latency.Milliseconds()

	if probeErr == nil {
		jsonOK(w, map[string]interface{}{
			"ok":         true,
			"latency_ms": latencyMs,
		})
		return
	}

	kind, message := classifyConnectivityError(r, probeErr)
	resp := map[string]interface{}{
		"ok":         false,
		"kind":       kind,
		"message":    message,
		"latency_ms": latencyMs,
	}
	var ce *provider.ConnectivityError
	if errors.As(probeErr, &ce) {
		if ce.StatusCode != 0 {
			resp["status_code"] = ce.StatusCode
		}
		if ce.Snippet != "" {
			resp["snippet"] = ce.Snippet
		}
	}
	slog.Info("[模型连通性探测] probe failed",
		"kind", kind,
		"model_type", modelType,
		"api_base", apiBase,
		"model", modelID,
		"latency_ms", latencyMs,
		"err", probeErr.Error(),
	)
	jsonOK(w, resp)
}

// resolveConnectivityArgs 从请求中解析连通性探测所需的四要素：
// apiBase（URL）、apiKey、modelType、modelID
func resolveConnectivityArgs(r *http.Request, user *model.User) (apiBase, apiKey, modelType, modelID string, err error) {
	var body struct {
		URL       string `json:"url"`
		APIKey    string `json:"api_key"`
		ModelType string `json:"model_type"`
		Model     string `json:"model"`
	}

	if r.Body != nil && r.ContentLength != 0 {
		if decErr := json.NewDecoder(r.Body).Decode(&body); decErr != nil && decErr != io.EOF {
			return "", "", "", "", hcommon.I18nError(i18n.MsgRequestBodyShouldBeJSON)
		}
	}

	if idStr := r.URL.Query().Get("id"); idStr != "" {
		id, convErr := strconv.ParseUint(idStr, 10, 64)
		if convErr != nil {
			return "", "", "", "", hcommon.I18nError(i18n.MsgBadRequestParamFormatError, "id")
		}
		var m model.AIModel
		if dbErr := model.DB(r.Context()).Where("id = ?", uint(id)).First(&m).Error; dbErr != nil {
			return "", "", "", "", hcommon.I18nError(i18n.MsgAdminModelNotFound)
		}
		if user != nil {
			// 用户端探测：模型必须既启用又对用户可见
			if !m.Enabled || !m.Visible {
				return "", "", "", "", ErrForbidden
			}
			visible, err := model.IsModelVisibleToUser(r.Context(), &m, user.ID)
			if err != nil {
				return "", "", "", "", err
			}
			if !visible {
				return "", "", "", "", ErrForbidden
			}
		}
		apiBase = m.URL
		apiKey = m.APIKey
		modelType = m.ModelType
		modelID = m.ModelID
	} else {
		apiBase = body.URL
		apiKey = body.APIKey
		modelType = body.ModelType
		modelID = body.Model
	}

	if apiBase == "" {
		return "", "", "", "", hcommon.I18nError(i18n.MsgBadRequestParamRequired, "URL")
	}
	if modelType == "" {
		return "", "", "", "", hcommon.I18nError(i18n.MsgBadRequestParamRequired, "model_type")
	}
	if apiKey == "" {
		return "", "", "", "", hcommon.I18nError(i18n.MsgBadRequestParamRequired, "api_key")
	}
	if modelID == "" {
		return "", "", "", "", hcommon.I18nError(i18n.MsgBadRequestParamRequired, "model")
	}
	return apiBase, apiKey, modelType, modelID, nil
}

// classifyConnectivityError 把 provider.CheckConnectivity 返回的错误映射为
// 稳定的 kind 标识与中文提示。kind 命名采用 snake_case，前端可据此做文案本地化。
func classifyConnectivityError(r *http.Request, err error) (kind, message string) {
	switch {
	case errors.Is(err, provider.ErrNetworkUnreachable):
		return "network_unreachable", i18n.T(r.Context(), i18n.MsgNetworkUnreachable)
	case errors.Is(err, provider.ErrInvalidAPIKey):
		return "invalid_api_key", i18n.T(r.Context(), i18n.MsgInvalidAPIKey)
	case errors.Is(err, provider.ErrForbidden):
		return "forbidden", i18n.T(r.Context(), i18n.MsgForbidden)
	case errors.Is(err, provider.ErrRateLimited):
		return "rate_limited", i18n.T(r.Context(), i18n.MsgRateLimitExceeded)
	case errors.Is(err, provider.ErrUpstreamServer):
		return "upstream_server_error", i18n.T(r.Context(), i18n.MsgUpstreamError)
	case errors.Is(err, provider.ErrUpstreamClient):
		return "upstream_client_error", i18n.T(r.Context(), i18n.MsgUpstreamDenied)
	default:
		return "unknown", i18n.T(r.Context(), i18n.MsgConnectivityCheckError)
	}
}
