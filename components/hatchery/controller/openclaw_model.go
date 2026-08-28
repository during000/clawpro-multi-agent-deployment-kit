package controller

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	hcommon "hatchery/common"
	"hatchery/controller/usergroup"
	"hatchery/i18n"
	"hatchery/model"
)

// errModelAlreadyBound 用于事务内返回"已绑定"语义错误，避免依赖字符串比较。
var errModelAlreadyBound error = hcommon.I18nError(i18n.MsgModelAlreadyBound)

// isUniqueConstraintError 判断错误是否为数据库唯一约束冲突。
// 兼容 SQLite（"UNIQUE constraint failed"）和 MySQL（error 1062 / "Duplicate entry"）。
// 用于并发场景下 COUNT+INSERT 之间的竞态：两个请求同时通过 COUNT 检查，其中一个 INSERT 触发唯一索引冲突。
func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") || // SQLite
		strings.Contains(msg, "Duplicate entry") // MySQL
}

// nextSortOrder 返回实例下一个应该使用的 SortOrder 值（MAX(sort_order)+1）。
// 这样即使存量记录被删除后再添加，SortOrder 也不会回退复用导致排序错乱或主键碰撞。
func nextSortOrder(tx *gorm.DB, instanceID uint) (int, error) {
	var maxSort sql.NullInt64
	row := tx.Model(&model.InstanceModel{}).
		Where("instance_id = ?", instanceID).
		Select("MAX(sort_order)").Row()
	if err := row.Scan(&maxSort); err != nil {
		return 0, err
	}
	if !maxSort.Valid {
		return 1, nil
	}
	return int(maxSort.Int64) + 1, nil
}

// syncHermesLLMToTDAI 在 set-model 成功后，通过 TAT 下发 sync-hermes-llm-to-tdai.sh
// 将 hermes 的 LLM 配置同步到 TDAI Gateway。
// 脚本自己从 ~/.hermes/config.yaml 读 LLM 配置，不需要额外传参。
// 非阻断：失败只 warn，不影响 set-model 结果。
var syncHermesLLMToTDAI = func(ctx context.Context, instanceID string) {
	defer func() {
		if r := recover(); r != nil {
			slog.Warn("[SetModel] sync-hermes-llm-to-tdai panic recovered (non-fatal)",
				"instance_id", instanceID, "panic", r)
		}
	}()
	runtimeUser := LookupRuntimeUser(ctx, instanceID)
	_, err := RunScript(ctx, instanceID, "sync-hermes-llm-to-tdai.sh", 60, runtimeUser, nil,
		map[string]string{
			"mode":    "hermes",
			"restart": "true",
		})
	if err != nil {
		slog.Warn("[SetModel] sync-hermes-llm-to-tdai failed (non-fatal)",
			"instance_id", instanceID, "error", err)
	} else {
		slog.Info("[SetModel] sync-hermes-llm-to-tdai completed",
			"instance_id", instanceID)
	}
}

func setModelProviderKey(m model.AIModel, isUserCustomModel bool) string {
	slugID := model.SlugifyModelID(m.ModelID)
	if isUserCustomModel {
		return fmt.Sprintf("custom-%s", slugID)
	}

	// CustomModelProvider 只是管控端模型的 Provider 名称；它不表示用户自定义模型。
	// 是否为用户侧自定义模型由调用方通过 isUserCustomModel 显式传入，此处仅将该名称转换为 hatchery slug 前缀。
	prefix := providerKeyPrefix(m.Provider, false)
	if m.Provider == hcommon.CustomModelProvider {
		prefix = model.BuiltinModelProvider
	}
	return fmt.Sprintf("%s-%s", prefix, slugID)
}

type setModelProviderModel struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	Reasoning         bool              `json:"reasoning"`
	Input             []string          `json:"input"`
	ContextWindow     int               `json:"contextWindow"`
	MaxTokens         int               `json:"maxTokens,omitempty"`
	CustomHTTPHeaders map[string]string `json:"headers,omitempty"`
}

type setModelProviderValue struct {
	BaseURL string                  `json:"baseUrl"`
	APIKey  string                  `json:"apiKey"`
	Auth    string                  `json:"auth"`
	API     string                  `json:"api"`
	Models  []setModelProviderModel `json:"models"`
}

type batchSetModelProviderValue struct {
	Provider string          `json:"provider"`
	Model    string          `json:"model"`
	Value    json.RawMessage `json:"value"`
}

type batchSetModelValue struct {
	Mode      string                       `json:"mode"`
	Providers []batchSetModelProviderValue `json:"providers"`
}

func buildSetModelProviderValue(m model.AIModel, isUserCustomModel bool) (json.RawMessage, string, string, error) {
	slugID := model.SlugifyModelID(m.ModelID)
	valueJSON, err := json.Marshal(setModelProviderValue{
		BaseURL: m.URL,
		APIKey:  m.APIKey,
		Auth:    "api-key",
		API:     m.ModelType,
		Models: []setModelProviderModel{
			{
				ID:                m.ModelID,
				Name:              m.ModelID,
				Reasoning:         true,
				Input:             m.GetInputTypes(),
				ContextWindow:     m.ContextLen,
				MaxTokens:         m.MaxTokens,
				CustomHTTPHeaders: m.GetCustomHTTPHeaders(),
			},
		},
	})
	if err != nil {
		return nil, "", "", hcommon.I18nRichError(err, i18n.MsgMarshalParamsFailed)
	}
	return json.RawMessage(valueJSON), setModelProviderKey(m, isUserCustomModel), slugID, nil
}

func setModelRefForModel(m model.AIModel, providerKey, slugID string, isUserCustomModel bool) string {
	refModelID := slugID
	if isUserCustomModel {
		refModelID = m.ModelID
	}
	return fmt.Sprintf("%s/%s", providerKey, refModelID)
}

// buildSetModelParams builds the TAT params for set_model.sh.
// instanceID 用于查询该实例的所有模型绑定，生成 primary 和 fallbacks 参数。
func buildSetModelParams(ctx context.Context, m model.AIModel, instanceID uint, isUserCustomModel bool) (map[string]string, error) {
	valueJSON, providerKey, slugID, err := buildSetModelProviderValue(m, isUserCustomModel)
	if err != nil {
		return nil, err
	}

	primary, fallbacks, err := buildPrimaryAndFallbacks(ctx, instanceID)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgModelGenPrimaryFallbacksFailed)
	}
	if primary == "" {
		primary = setModelRefForModel(m, providerKey, slugID, isUserCustomModel)
	}

	imagePrimary, imageFallbacks, err := buildImageModelRefs(ctx, instanceID, primary)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgModelGenImageModelParamFailed)
	}

	return map[string]string{
		"valueb64":          base64.StdEncoding.EncodeToString(valueJSON),
		"provider":          providerKey,
		"model":             slugID,
		"primary":           primary,
		"fallbacksb64":      base64.StdEncoding.EncodeToString([]byte(fallbacks)),
		"imageprimary":      imagePrimary,
		"imagefallbacksb64": base64.StdEncoding.EncodeToString([]byte(imageFallbacks)),
	}, nil
}

func buildBatchSetModelParams(ctx context.Context, models []resolvedSetModelBinding, instanceID uint) (map[string]string, error) {
	if len(models) == 0 {
		return nil, hcommon.I18nError(i18n.MsgBadRequest)
	}

	providers := make([]batchSetModelProviderValue, 0, len(models))
	for _, binding := range models {
		value, providerKey, slugID, err := buildSetModelProviderValue(binding.InjectModel, binding.IsUserCustomModel)
		if err != nil {
			return nil, err
		}
		providers = append(providers, batchSetModelProviderValue{
			Provider: providerKey,
			Model:    slugID,
			Value:    value,
		})
	}

	valueJSON, err := json.Marshal(batchSetModelValue{
		Mode:      "batch",
		Providers: providers,
	})
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgMarshalParamsFailed)
	}

	primary, fallbacks, err := buildPrimaryAndFallbacks(ctx, instanceID)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgModelGenPrimaryFallbacksFailed)
	}
	if primary == "" {
		first := models[0]
		primary = setModelRefForModel(first.InjectModel, providers[0].Provider, providers[0].Model, first.IsUserCustomModel)
	}

	imagePrimary, imageFallbacks, err := buildImageModelRefs(ctx, instanceID, primary)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgModelGenImageModelParamFailed)
	}

	return map[string]string{
		"valueb64":          base64.StdEncoding.EncodeToString(valueJSON),
		"provider":          providers[0].Provider,
		"model":             providers[0].Model,
		"primary":           primary,
		"fallbacksb64":      base64.StdEncoding.EncodeToString([]byte(fallbacks)),
		"imageprimary":      imagePrimary,
		"imagefallbacksb64": base64.StdEncoding.EncodeToString([]byte(imageFallbacks)),
	}, nil
}

// buildPrimaryAndFallbacks 根据 instance_models 表生成 TAT 脚本的 primary 和 fallbacks 参数。
// fallbacks 按 sort_order ASC 排序（最早添加的优先，符合方案文档 §6.1）。
// 返回 error，DB 查询失败时调用方应中断操作，避免以空值下发 TAT 清空实例配置。
func buildPrimaryAndFallbacks(ctx context.Context, instanceID uint) (string, string, error) {
	var models []model.InstanceModel
	if err := model.DB(ctx).Where("instance_id = ?", instanceID).Order("sort_order ASC").Find(&models).Error; err != nil {
		return "", "", hcommon.I18nRichError(err, i18n.MsgModelQueryInstanceModelsFailed)
	}

	var primary string
	var fallbackList []string
	for _, m := range models {
		bindingRef := resolveBindingRef(ctx, m)
		// resolveBindingRef 在 ai_models 表记录缺失时会返回 "hatchery-N/unknown"，
		// 该值下发到实例会导致 gateway 报 Unknown model，应视为异常跳过。
		if strings.Contains(bindingRef, "/unknown") {
			slog.Warn("[buildPrimaryAndFallbacks] 跳过异常 bindingRef",
				"instance_id", instanceID, "instance_model_id", m.ID, "ref", bindingRef)
			continue
		}
		if m.Role == model.ModelRolePrimary {
			primary = bindingRef
		} else if m.Role == model.ModelRoleFallback {
			fallbackList = append(fallbackList, bindingRef)
		}
	}
	fallbacksJSON, _ := json.Marshal(fallbackList)
	if len(fallbackList) == 0 {
		fallbacksJSON = []byte("[]")
	}
	return primary, string(fallbacksJSON), nil
}

// buildImageModelRefs 根据实例当前绑定模型推导 agents.defaults.imageModel 的 primary 和
// fallbacks，用于 OpenClaw 5.7+ image 输入支持。
//
// 规则（与方案文档保持一致）：
//   - 候选 = 所有 InputTypes 含 "image" 的 InstanceModel，按 sort_order ASC
//   - 主模型 ∈ 候选：imageModel.primary = 主模型 ref，fallbacks = 候选 - 主模型
//   - 主模型 ∉ 候选 但候选非空：imageModel.primary = 候选[0]，fallbacks = 候选[1:]
//   - 候选为空：返回 ("", "[]", nil)，调用方约定脚本侧执行 del(.agents.defaults.imageModel)
//
// 模型是否支持 image：
//   - 内置模型（AIModelID > 0）：查 ai_models.input_types
//   - 自定义模型（AIModelID == 0）：解析 CustomModelConfig JSON 的 input_types；
//     JSON 异常 → 视为不支持，记 warn，不中断
//
// primaryRef 由调用方从 buildPrimaryAndFallbacks / resolveBindingRef 传入，用于在主模型支持
// image 时强制对齐（即使 sort_order 把别的 image 模型排在前面）。
func buildImageModelRefs(ctx context.Context, instanceID uint, primaryRef string) (string, string, error) {
	var models []model.InstanceModel
	if err := model.DB(ctx).Where("instance_id = ?", instanceID).Order("sort_order ASC").Find(&models).Error; err != nil {
		return "", "", hcommon.I18nRichError(err, i18n.MsgModelQueryInstanceModelsFailed)
	}

	// 一次性批量查询所有内置模型的 input_types/provider/model_id，避免循环里 N+1 查询：
	//   - input_types 用于判断是否支持 image
	//   - provider/model_id 用于本地拼 binding ref（跳过 resolveBindingRef 的逐条 First 查询）
	// 自定义模型的字段直接来自 InstanceModel.CustomModelConfig / CustomModelID，无需查 DB。
	var aiModelIDs []uint
	for _, im := range models {
		if im.AIModelID > 0 {
			aiModelIDs = append(aiModelIDs, im.AIModelID)
		}
	}
	aiCache := make(map[uint]model.AIModel, len(aiModelIDs))
	if len(aiModelIDs) > 0 {
		var aims []model.AIModel
		if err := model.DB(ctx).Select("id, input_types, provider, model_id").
			Where("id IN ?", aiModelIDs).Find(&aims).Error; err != nil {
			return "", "", hcommon.I18nRichError(err, i18n.MsgModelBatchQueryAIModelsFailed)
		}
		for _, a := range aims {
			aiCache[a.ID] = a
		}
	}

	// 收集所有支持 image 的候选 ref（按 sort_order ASC 顺序）
	var candidates []string
	for _, im := range models {
		if !instanceModelSupportsImageWithCache(im, aiCache) {
			continue
		}
		// 内置模型走缓存拼 ref（与 resolveBindingRef 内置分支等价），自定义模型 fallback。
		var ref string
		if im.AIModelID > 0 {
			if aim, ok := aiCache[im.AIModelID]; ok {
				slugID := model.SlugifyModelID(aim.ModelID)
				ref = fmt.Sprintf("%s/%s", setModelProviderKey(aim, false), slugID)
			} else {
				ref = fmt.Sprintf("%s-%d/unknown", model.BuiltinModelProvider, im.AIModelID)
			}
		} else {
			ref = resolveBindingRef(ctx, im)
		}
		// 跳过异常 ref（与 buildPrimaryAndFallbacks 一致）
		if strings.Contains(ref, "/unknown") {
			slog.Warn("[buildImageModelRefs] 跳过异常 bindingRef",
				"instance_id", instanceID, "instance_model_id", im.ID, "ref", ref)
			continue
		}
		candidates = append(candidates, ref)
	}

	if len(candidates) == 0 {
		return "", "[]", nil
	}

	var primary string
	var fallbacks []string
	// 主模型支持 image → 强制对齐，从候选里剔除主模型作为 fallbacks
	primaryInCandidates := false
	if primaryRef != "" {
		for _, c := range candidates {
			if c == primaryRef {
				primaryInCandidates = true
				break
			}
		}
	}

	if primaryInCandidates {
		primary = primaryRef
		for _, c := range candidates {
			if c != primaryRef {
				fallbacks = append(fallbacks, c)
			}
		}
	} else {
		primary = candidates[0]
		fallbacks = candidates[1:]
	}

	if len(fallbacks) == 0 {
		return primary, "[]", nil
	}
	fallbacksJSON, err := json.Marshal(fallbacks)
	if err != nil {
		return "", "", hcommon.I18nRichError(err, i18n.MsgModelMarshalFallbacksFailed)
	}
	return primary, string(fallbacksJSON), nil
}

// instanceModelSupportsImageWithCache 判断一条 InstanceModel 是否支持 image 输入，
// 内置模型从 aiCache（由调用方一次性批量查出）读取 input_types，避免 N+1。
// 自定义模型直接从 CustomModelConfig 解析。
func instanceModelSupportsImageWithCache(im model.InstanceModel, aiCache map[uint]model.AIModel) bool {
	if im.AIModelID > 0 {
		aim, ok := aiCache[im.AIModelID]
		if !ok {
			// ai_models 表中没找到该记录（已被删除）→ 视为不支持
			return false
		}
		// 复用 AIModel.GetInputTypes 的解析逻辑（兼容 JSON array / 逗号分隔 / 空字符串）
		for _, t := range aim.GetInputTypes() {
			if strings.EqualFold(t, "image") {
				return true
			}
		}
		return false
	}
	// 自定义模型：解析 CustomModelConfig
	if im.CustomModelConfig == "" {
		return false
	}
	var cfg customModelConfig
	if err := json.Unmarshal([]byte(im.CustomModelConfig), &cfg); err != nil {
		slog.Warn("[instanceModelSupportsImage] 解析 CustomModelConfig 失败，视为不支持 image",
			"instance_model_id", im.ID, "error", err)
		return false
	}
	for _, t := range cfg.InputTypes {
		if strings.EqualFold(t, "image") {
			return true
		}
	}
	return false
}

// providerKeyPrefix 根据模型来源生成 provider key 的前缀。
// 模型来源由 isUserCustomModel 显式传入，不从 Provider 展示值推断。
func providerKeyPrefix(provider string, isUserCustomModel bool) string {
	if isUserCustomModel {
		return "custom"
	}
	return strings.ToLower(provider)
}

// resolveBindingRef 生成一条绑定记录的引用标识，格式为 "providerKey/modelId"。
func resolveBindingRef(ctx context.Context, m model.InstanceModel) string {
	if m.AIModelID > 0 {
		// 内置 + 管理侧自定义模型
		var aim model.AIModel
		// NOCA:sql_injection(m.AIModelID 为 uint 主键，GORM First 内部参数化查询无注入风险)
		if model.DB(ctx).Select("provider, model_id").First(&aim, m.AIModelID).Error == nil {
			slugID := model.SlugifyModelID(aim.ModelID)
			return fmt.Sprintf("%s/%s", setModelProviderKey(aim, false), slugID)
		}
		return fmt.Sprintf("%s-%d/unknown", model.BuiltinModelProvider, m.AIModelID)
	}
	// 用户侧自定义模型：binding ref 格式为 "custom-{slug(model_id)}/{model_id}"
	// 优先使用 CustomModelID 字段（稳定业务键），兜底从 JSON 解析
	//
	// 【方案 C】ref 后段保留用户原始填写的 model_id（不做 slug 化）。
	// 原因：用户侧自定义模型的请求由 OpenClaw Agent 直接透传到上游 BaseURL，
	// 不经过 hatchery 的 llm_proxy 兜底覆盖。OpenClaw 解析 ref 时会用 "/" 之后
	// 的部分作为 body.model 发给上游，必须保留原始大小写，否则会出现
	// "DeepSeek-V3.1" → "deepseek-v3.1" → 上游 404 model_not_found 的问题。
	// providerKey 前段（custom-{slug}）仍 slug 化，保持白名单字符集要求不变。
	modelID := m.CustomModelID
	if modelID == "" {
		var cfg customModelConfig
		json.Unmarshal([]byte(m.CustomModelConfig), &cfg)
		modelID = cfg.ModelID
	}
	if modelID == "" {
		modelID = "unknown"
	}
	return fmt.Sprintf("custom-%s/%s", model.SlugifyModelID(modelID), modelID)
}

// HandleModelsList returns the list of enabled AI models available for instances.
func HandleModelsList(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		return
	}

	var models []model.AIModel
	// 用户端列表：只返回「启用 + 可见」的模型。
	//   - enabled=true 保证模型可用于 LLM 代理
	//   - visible=true 保证模型应该被用户看到
	//
	// 例外：内置的 hatchery/custom 占位记录不代表可用模型，而是表示
	// "是否允许用户自定义模型"的功能开关。其 Enabled 字段是 feature gate，
	// 而非"模型是否可用于路由"，因此该记录只需判 Visible=true 即可。
	model.DB(r.Context()).Where(
		"(enabled = ? AND visible = ?) OR (provider = ? AND model_id = ? AND visible = ?)",
		true, true,
		model.BuiltinModelProvider, model.BuiltinModelID, true,
	).Order("created_at DESC").Find(&models)

	// 按可见性过滤模型列表（支持 agent_id 参数指定实例，查其绑定的分组）
	totalCount := len(models)
	var agentGroupID uint
	if agentIDStr := r.URL.Query().Get("agent_id"); agentIDStr != "" {
		if instID, err := strconv.ParseUint(agentIDStr, 10, 64); err == nil && instID > 0 {
			var inst model.Instance
			if model.DB(r.Context()).Select("group_id").Where("id = ? AND user_id = ?", instID, user.ID).First(&inst).Error == nil {
				agentGroupID = inst.GroupID
			}
		}
	}
	models = usergroup.FilterModelsByVisibility(r.Context(), models, agentGroupID)

	// 判断分组是否打开自定义模型开关，否则过滤掉 custom 占位模型
	if !usergroup.ResolvePolicyBoolForGroup(r.Context(), usergroup.PolicyKeyCustomModel, agentGroupID, model.IsCustomModelEnabled(r.Context())) {
		kept := models[:0]
		for _, m := range models {
			if m.Provider == model.BuiltinModelProvider && m.ModelID == model.BuiltinModelID {
				continue
			}
			kept = append(kept, m)
		}
		models = kept
	}

	// 仅在有模型被过滤时记录日志（避免热路径刷屏）
	if len(models) < totalCount {
		slog.Info("[ModelVisibility] 模型列表已过滤",
			"user_id", user.ID,
			"total", totalCount,
			"visible", len(models),
		)
	}

	type modelItem struct {
		ID                uint              `json:"id"`
		Provider          string            `json:"provider"`
		ModelID           string            `json:"model_id"`
		ModelType         string            `json:"model_type"`
		InputTypes        []string          `json:"input_types"`
		ContextLen        int               `json:"context_len"`
		MaxTokens         int               `json:"max_tokens"`
		CustomHTTPHeaders map[string]string `json:"custom_http_headers"`
		ModelName         string            `json:"model_name"`
		Default           bool              `json:"default"`
	}

	config := model.GetSiteConfig(r.Context())
	items := make([]modelItem, 0, len(models))
	for _, m := range models {
		items = append(items, modelItem{
			ID:                m.ID,
			Provider:          m.Provider,
			ModelID:           m.ModelID,
			ModelType:         m.ModelType,
			InputTypes:        m.GetInputTypes(),
			ContextLen:        m.ContextLen,
			MaxTokens:         m.MaxTokens,
			CustomHTTPHeaders: m.GetCustomHTTPHeaders(),
			ModelName:         m.ModelName,
			Default:           config.DefaultModelID == m.ID,
		})
	}

	jsonOK(w, map[string]interface{}{
		"ok":     true,
		"models": items,
	})
}

// customModelConfig is the JSON structure stored in Instance.CustomModelConfig.
type customModelConfig struct {
	Provider          string            `json:"provider"`
	ModelID           string            `json:"model_id"`
	ModelName         string            `json:"model_name"`
	APIKey            string            `json:"api_key"`
	URL               string            `json:"url"`
	ModelType         string            `json:"model_type"`
	InputTypes        []string          `json:"input_types"`
	ContextLen        int               `json:"context_len"`
	MaxTokens         int               `json:"max_tokens,omitempty"`
	CustomHTTPHeaders map[string]string `json:"custom_http_headers,omitempty"`
}

// HandleSetModel binds an AIModel to an instance by database ID and injects config via TAT.
// When ai_model_id=0, it uses custom model configuration from form fields.
func HandleSetModel(w http.ResponseWriter, r *http.Request) {
	handleSetModel(w, r, defaultStatusResolver)
}

func handleSetModel(w http.ResponseWriter, r *http.Request, resolver instanceStatusResolver) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	setModel(w, r, instance, user.ID, resolver)
}

func parseSetModelInstanceModelID(r *http.Request) (uint, bool, *hcommon.RichError) {
	raw := strings.TrimSpace(r.FormValue("instance_model_id"))
	if raw == "" {
		return 0, false, nil
	}
	id, err := strconv.Atoi(raw)
	if err != nil || id <= 0 {
		return 0, false, hcommon.I18nError(i18n.MsgModelInvalidInstanceModelID)
	}
	return uint(id), true, nil
}

// setModelInput describes the primary set-model operation shared by HTTP adapters and admin batch calls.
type setModelInput struct {
	AIModelID         uint
	Provider          string
	ModelID           string
	ModelName         string
	APIKey            string
	URL               string
	ModelType         string
	InputTypes        []string
	ContextLen        int
	MaxTokens         int
	CustomHTTPHeaders map[string]string
}

type setModelApplyError struct {
	HTTPStatus int
	Err        *hcommon.RichError
}

func newSetModelApplyError(status int, err *hcommon.RichError) *setModelApplyError {
	return &setModelApplyError{HTTPStatus: status, Err: err}
}

func validateSetModelInstance(ctx context.Context, instance *model.Instance, resolver instanceStatusResolver) *setModelApplyError {
	// 【关键防护】校验实例是否支持模型配置
	if err := checkInstanceSupportsModel(ctx, instance); err != nil {
		return newSetModelApplyError(http.StatusForbidden, hcommon.EnsureRichErrorOrPanic(err))
	}

	// 本地实例：模型配置需下发到 CVM Agent，本地 agent 不支持。
	if rerr := rejectLocalInstance(instance); rerr != nil {
		return newSetModelApplyError(http.StatusBadRequest, rerr)
	}

	// 状态准入：仅 running 状态允许配置模型
	if _, err := requireInstanceRunning(ctx, instance, resolver); err != nil {
		if errors.Is(err, ErrAgentNotAllowed) || errors.Is(err, ErrOperationInProgress) {
			return newSetModelApplyError(http.StatusConflict, hcommon.EnsureRichErrorOrPanic(err))
		}
		return newSetModelApplyError(http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
	}

	return nil
}

func parseSetModelInputFromRequest(r *http.Request) (setModelInput, *setModelApplyError) {
	aiModelIdStr := r.FormValue("ai_model_id")
	if aiModelIdStr == "" {
		return setModelInput{}, newSetModelApplyError(http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
	}

	aiModelIDInt, err := strconv.Atoi(aiModelIdStr)
	if err != nil {
		return setModelInput{}, newSetModelApplyError(http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
	}

	in := setModelInput{AIModelID: uint(aiModelIDInt)}
	if aiModelIDInt != 0 {
		return in, nil
	}

	contextLen := 0
	if contextLenStr := r.FormValue("context_len"); contextLenStr != "" {
		contextLen, err = strconv.Atoi(contextLenStr)
		if err != nil {
			return setModelInput{}, newSetModelApplyError(http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgBadRequest))
		}
	}

	maxTokensStr := r.FormValue("max_tokens")
	maxTokens, err := strconv.Atoi(maxTokensStr)
	if maxTokensStr != "" && err != nil {
		return setModelInput{}, newSetModelApplyError(http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgMaxTokensMustBeInteger))
	}

	customHTTPHeaders, err := hcommon.ValidateAndParseCustomHTTPHeaders(r.FormValue("custom_http_headers"))
	if err != nil {
		return setModelInput{}, newSetModelApplyError(http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
	}

	in.Provider = r.FormValue("provider")
	in.ModelID = r.FormValue("model_id")
	in.ModelName = r.FormValue("model_name")
	in.APIKey = r.FormValue("api_key")
	in.URL = r.FormValue("url")
	in.ModelType = r.FormValue("model_type")
	in.InputTypes = r.Form["input_types"]
	in.ContextLen = contextLen
	in.MaxTokens = maxTokens
	in.CustomHTTPHeaders = customHTTPHeaders
	return in, nil
}

// setModel 设置/替换模型，用户端和管控端共享。默认设置 primary；传 instance_model_id 时更新指定绑定。
func setModel(w http.ResponseWriter, r *http.Request, instance *model.Instance, notifyUID uint, resolver instanceStatusResolver) {
	if applyErr := validateSetModelInstance(r.Context(), instance, resolver); applyErr != nil {
		writeError(w, r, applyErr.HTTPStatus, applyErr.Err)
		return
	}

	targetID, hasTarget, targetErr := parseSetModelInstanceModelID(r)
	if targetErr != nil {
		writeError(w, r, http.StatusBadRequest, targetErr)
		return
	}
	if hasTarget {
		setModelInstanceBinding(w, r, instance, notifyUID, targetID)
		return
	}

	input, applyErr := parseSetModelInputFromRequest(r)
	if applyErr != nil {
		writeError(w, r, applyErr.HTTPStatus, applyErr.Err)
		return
	}

	provider := input.Provider
	modelID := input.ModelID
	if input.AIModelID == 0 {
		if provider == "" {
			provider = hcommon.CustomModelProvider
		}
	} else {
		var aiModel model.AIModel
		if model.DB(r.Context()).Select("provider, model_id").
			Where("id = ? AND enabled = ? AND visible = ?", input.AIModelID, true, true).
			First(&aiModel).Error != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgModelNotFoundOrDisabled))
			return
		}
		provider = aiModel.Provider
		modelID = aiModel.ModelID
	}

	applyErr = setPrimaryModelForValidatedInstance(r.Context(), instance, notifyUID, input)
	if applyErr != nil {
		writeError(w, r, applyErr.HTTPStatus, applyErr.Err)
		return
	}

	jsonOK(w, map[string]interface{}{
		"ok":       true,
		"provider": provider,
		"model_id": modelID,
	})
}

func setPrimaryModelForValidatedInstance(ctx context.Context, instance *model.Instance, notifyUID uint, in setModelInput) *setModelApplyError {
	if in.AIModelID == 0 {
		return setCustomPrimaryModelForInstance(ctx, instance, notifyUID, in)
	}
	return setBuiltInPrimaryModelForInstance(ctx, instance, notifyUID, in)
}

type resolvedSetModelBinding struct {
	Row               model.InstanceModel
	InjectModel       model.AIModel
	IsUserCustomModel bool
}

type instanceModelBindingsSnapshot struct {
	OldInstanceAIModelID    uint
	OldInstanceCustomConfig string
	OldBindings             []model.InstanceModel
}

func resolveSetModelBindingForInstance(ctx context.Context, instance *model.Instance, in setModelInput, role string, sortOrder int) (resolvedSetModelBinding, *setModelApplyError) {
	if in.AIModelID > 0 {
		var aiModel model.AIModel
		if model.DB(ctx).Where("id = ? AND enabled = ? AND visible = ?", in.AIModelID, true, true).First(&aiModel).Error != nil {
			return resolvedSetModelBinding{}, newSetModelApplyError(http.StatusBadRequest, hcommon.I18nError(i18n.MsgModelNotFoundOrDisabled))
		}

		filtered := usergroup.FilterModelsByVisibility(ctx, []model.AIModel{aiModel}, instance.GroupID)
		if len(filtered) == 0 {
			slog.Warn("[BatchSetModel] 绑定被拒：模型不在实例分组可见范围",
				"model_id", aiModel.ID, "instance_group_id", instance.GroupID)
			return resolvedSetModelBinding{}, newSetModelApplyError(http.StatusForbidden, hcommon.I18nError(i18n.MsgModelNoAccess))
		}

		if hcommon.DomainFromCtx(ctx) == "" {
			return resolvedSetModelBinding{}, newSetModelApplyError(http.StatusInternalServerError, hcommon.I18nError(i18n.MsgModelDomainNotConfigured))
		}

		return resolvedSetModelBinding{
			Row: model.InstanceModel{
				InstanceID:        instance.ID,
				AIModelID:         aiModel.ID,
				CustomModelID:     "",
				Role:              role,
				SortOrder:         sortOrder,
				CustomModelConfig: "",
			},
			InjectModel:       aiModel,
			IsUserCustomModel: false,
		}, nil
	}

	if !usergroup.ResolvePolicyBoolForGroup(ctx, usergroup.PolicyKeyCustomModel, instance.GroupID, model.IsCustomModelEnabled(ctx)) {
		slog.Warn("[BatchSetModel] 绑定被拒：分组不允许添加自定义模型",
			"instance_group_id", instance.GroupID)
		return resolvedSetModelBinding{}, newSetModelApplyError(http.StatusForbidden, hcommon.I18nError(i18n.MsgModelCustomModelDisabled))
	}

	provider := in.Provider
	if provider == "" {
		provider = hcommon.CustomModelProvider
	}
	modelID := in.ModelID
	modelName := in.ModelName
	if modelName == "" {
		modelName = modelID
	}
	inputTypes, err := hcommon.ValidateInputTypes(in.InputTypes)
	if err != nil {
		return resolvedSetModelBinding{}, newSetModelApplyError(http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
	}

	if modelID == "" || in.APIKey == "" || in.URL == "" || in.ModelType == "" {
		return resolvedSetModelBinding{}, newSetModelApplyError(http.StatusBadRequest, hcommon.I18nError(i18n.MsgModelCustomModelFieldsRequired))
	}
	if err := hcommon.ValidateCustomModelID(modelID); err != nil {
		return resolvedSetModelBinding{}, newSetModelApplyError(http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
	}
	if err := hcommon.ValidateHTTPURL(in.URL); err != nil {
		return resolvedSetModelBinding{}, newSetModelApplyError(http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
	}
	if err := hcommon.ValidateModelType(in.ModelType); err != nil {
		return resolvedSetModelBinding{}, newSetModelApplyError(http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
	}

	contextLen := in.ContextLen
	if contextLen <= 0 {
		contextLen = 128000
	}
	maxTokens := in.MaxTokens
	if maxTokens < 0 {
		maxTokens = 0
	}
	customHTTPHeaders, err := validateCustomHTTPHeadersMap(in.CustomHTTPHeaders)
	if err != nil {
		return resolvedSetModelBinding{}, newSetModelApplyError(http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
	}

	inputTypesJSON, _ := json.Marshal(inputTypes)
	cfg := customModelConfig{
		Provider:          provider,
		ModelID:           modelID,
		ModelName:         modelName,
		APIKey:            in.APIKey,
		URL:               in.URL,
		ModelType:         in.ModelType,
		InputTypes:        inputTypes,
		ContextLen:        contextLen,
		MaxTokens:         maxTokens,
		CustomHTTPHeaders: customHTTPHeaders,
	}
	cfgJSON, marshalErr := json.Marshal(cfg)
	if marshalErr != nil {
		return resolvedSetModelBinding{}, newSetModelApplyError(http.StatusInternalServerError, hcommon.I18nError(i18n.MsgModelMarshalConfigFailed))
	}
	customHTTPHeadersJSON, err := json.Marshal(customHTTPHeaders)
	if err != nil {
		return resolvedSetModelBinding{}, newSetModelApplyError(http.StatusInternalServerError, hcommon.I18nError(i18n.MsgModelMarshalCustomHeadersFailed))
	}

	tempModel := model.AIModel{
		Provider:          cfg.Provider,
		ModelID:           cfg.ModelID,
		APIKey:            cfg.APIKey,
		URL:               cfg.URL,
		ModelType:         cfg.ModelType,
		InputTypes:        string(inputTypesJSON),
		ContextLen:        cfg.ContextLen,
		MaxTokens:         cfg.MaxTokens,
		CustomHTTPHeaders: string(customHTTPHeadersJSON),
	}

	return resolvedSetModelBinding{
		Row: model.InstanceModel{
			InstanceID:        instance.ID,
			AIModelID:         0,
			CustomModelID:     cfg.ModelID,
			Role:              role,
			SortOrder:         sortOrder,
			CustomModelConfig: string(cfgJSON),
		},
		InjectModel:       tempModel,
		IsUserCustomModel: true,
	}, nil
}

func replaceInstanceModelBindings(ctx context.Context, instance *model.Instance, desired []model.InstanceModel) (instanceModelBindingsSnapshot, *setModelApplyError) {
	st := instanceModelBindingsSnapshot{
		OldInstanceAIModelID:    instance.AIModelID,
		OldInstanceCustomConfig: instance.CustomModelConfig,
	}
	if err := model.DB(ctx).Where("instance_id = ?", instance.ID).Order("sort_order ASC").Find(&st.OldBindings).Error; err != nil {
		return st, newSetModelApplyError(http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgModelSaveBindingFailed))
	}
	if len(desired) == 0 {
		return st, newSetModelApplyError(http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
	}

	primary := desired[0]
	if err := model.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("instance_id = ?", instance.ID).Delete(&model.InstanceModel{}).Error; err != nil {
			return err
		}
		for i := range desired {
			row := desired[i]
			row.InstanceID = instance.ID
			row.SortOrder = i + 1
			if i == 0 {
				row.Role = model.ModelRolePrimary
			} else {
				row.Role = model.ModelRoleFallback
			}
			if row.AIModelID > 0 {
				row.CustomModelID = ""
				row.CustomModelConfig = ""
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return tx.Model(instance).Updates(map[string]interface{}{
			"ai_model_id":         primary.AIModelID,
			"custom_model_config": primary.CustomModelConfig,
		}).Error
	}); err != nil {
		return st, newSetModelApplyError(http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgModelSaveBindingFailed))
	}
	instance.AIModelID = primary.AIModelID
	instance.CustomModelConfig = primary.CustomModelConfig
	return st, nil
}

func restoreInstanceModelBindings(ctx context.Context, instance *model.Instance, st instanceModelBindingsSnapshot) bool {
	err := model.DB(hcommon.DetachContext(ctx)).Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("instance_id = ?", instance.ID).Delete(&model.InstanceModel{}).Error; err != nil {
			return err
		}
		for _, oldBinding := range st.OldBindings {
			restore := oldBinding
			if err := tx.Create(&restore).Error; err != nil {
				return err
			}
		}
		return tx.Model(instance).Updates(map[string]interface{}{
			"ai_model_id":         st.OldInstanceAIModelID,
			"custom_model_config": st.OldInstanceCustomConfig,
		}).Error
	})
	if err != nil {
		slog.Error("[BatchSetModel] 回滚 DB 失败", "error", err, "instance_id", instance.InstanceId)
		return false
	}
	instance.AIModelID = st.OldInstanceAIModelID
	instance.CustomModelConfig = st.OldInstanceCustomConfig
	return true
}

// restoredBindingModelForInjection 返回的 bool 仅表示该绑定是否为用户侧自定义模型（AIModelID == 0）。
func restoredBindingModelForInjection(ctx context.Context, binding model.InstanceModel) (model.AIModel, bool, error) {
	if binding.AIModelID > 0 {
		var aiModel model.AIModel
		if err := model.DB(ctx).First(&aiModel, binding.AIModelID).Error; err != nil {
			return model.AIModel{}, false, err
		}
		return aiModel, false, nil
	}

	var cfg customModelConfig
	if err := json.Unmarshal([]byte(binding.CustomModelConfig), &cfg); err != nil {
		return model.AIModel{}, true, err
	}
	if cfg.Provider == "" {
		cfg.Provider = hcommon.CustomModelProvider
	}
	inputTypesJSON, err := json.Marshal(cfg.InputTypes)
	if err != nil {
		return model.AIModel{}, true, err
	}
	customHTTPHeadersJSON, err := json.Marshal(cfg.CustomHTTPHeaders)
	if err != nil {
		return model.AIModel{}, true, err
	}
	return model.AIModel{
		Provider:          cfg.Provider,
		ModelID:           cfg.ModelID,
		APIKey:            cfg.APIKey,
		URL:               cfg.URL,
		ModelType:         cfg.ModelType,
		InputTypes:        string(inputTypesJSON),
		ContextLen:        cfg.ContextLen,
		MaxTokens:         cfg.MaxTokens,
		CustomHTTPHeaders: string(customHTTPHeadersJSON),
	}, true, nil
}

func restoreInstanceModelBindingsToCVM(ctx context.Context, instance *model.Instance, st instanceModelBindingsSnapshot) bool {
	for _, oldBinding := range st.OldBindings {
		aiModel, isUserCustomModel, err := restoredBindingModelForInjection(ctx, oldBinding)
		if err != nil {
			slog.Error("[BatchSetModel] 构造运行时回滚模型失败", "error", err, "instance_id", instance.InstanceId, "instance_model_id", oldBinding.ID)
			return false
		}
		if err := injectModelConfigToCVM(ctx, instance, &aiModel, isUserCustomModel); err != nil {
			slog.Error("[BatchSetModel] 回滚 CVM 模型配置失败", "error", err, "instance_id", instance.InstanceId, "instance_model_id", oldBinding.ID)
			return false
		}
	}
	return true
}

func cleanupDesiredModelBindingsFromCVM(ctx context.Context, instance *model.Instance, resolved []resolvedSetModelBinding) bool {
	seen := make(map[string]struct{}, len(resolved))
	for _, binding := range resolved {
		providerKey := setModelProviderKey(binding.InjectModel, binding.IsUserCustomModel)
		if _, ok := seen[providerKey]; ok {
			continue
		}
		seen[providerKey] = struct{}{}
		if _, err := syncScriptRunner(ctx, instance.InstanceId, "remove_model_provider.sh", 30, instance.RuntimeUser, nil, map[string]string{
			"provider": providerKey,
		}); err != nil {
			slog.Error("[BatchSetModel] 清理 CVM 新模型配置失败", "error", err, "instance_id", instance.InstanceId, "provider", providerKey)
			return false
		}
	}
	return true
}

func batchSetModelForInstance(ctx context.Context, instance *model.Instance, notifyUID uint, primaryInput setModelInput, fallbackInputs []setModelInput, resolver instanceStatusResolver) *setModelApplyError {
	if applyErr := validateSetModelInstance(ctx, instance, resolver); applyErr != nil {
		return applyErr
	}
	runtimeType := model.GetAgentRuntimeType(ctx, instance.AgentType)
	if runtimeType == "" {
		return newSetModelApplyError(http.StatusForbidden, hcommon.I18nError(i18n.MsgModelNotSupported))
	}
	if len(fallbackInputs) > 0 {
		if runtimeType != model.AgentTypeOpenClaw {
			return newSetModelApplyError(http.StatusBadRequest, hcommon.I18nError(i18n.MsgBatchSetModelFallbackUnsupported))
		}
		if strings.HasPrefix(instance.AgentVersion, "3.28") {
			return newSetModelApplyError(http.StatusConflict, hcommon.I18nError(i18n.MsgModelAgentFallbackUnsupported))
		}
	}

	resolved := make([]resolvedSetModelBinding, 0, 1+len(fallbackInputs))
	primary, applyErr := resolveSetModelBindingForInstance(ctx, instance, primaryInput, model.ModelRolePrimary, 1)
	if applyErr != nil {
		return applyErr
	}
	resolved = append(resolved, primary)
	for i, fallback := range fallbackInputs {
		binding, applyErr := resolveSetModelBindingForInstance(ctx, instance, fallback, model.ModelRoleFallback, i+2)
		if applyErr != nil {
			return applyErr
		}
		resolved = append(resolved, binding)
	}

	desired := make([]model.InstanceModel, 0, len(resolved))
	for _, binding := range resolved {
		desired = append(desired, binding.Row)
	}
	snapshot, applyErr := replaceInstanceModelBindings(ctx, instance, desired)
	if applyErr != nil {
		return applyErr
	}

	var injectErr error
	if runtimeType == model.AgentTypeOpenClaw {
		injectErr = injectModelConfigsToCVM(ctx, instance, resolved)
	} else {
		binding := resolved[0]
		injectErr = injectModelConfigToCVM(ctx, instance, &binding.InjectModel, binding.IsUserCustomModel)
	}
	if injectErr != nil {
		dbRestored := restoreInstanceModelBindings(ctx, instance, snapshot)
		if dbRestored {
			if len(snapshot.OldBindings) == 0 {
				if cleanupDesiredModelBindingsFromCVM(ctx, instance, resolved) {
					slog.Warn("[BatchSetModel] TAT 执行失败，已回滚 DB 并清理 CVM 新模型配置", "error", injectErr, "instance_id", instance.InstanceId)
				} else {
					slog.Error("[BatchSetModel] TAT 执行失败，DB 已回滚但 CVM 新模型配置清理未成功",
						"error", injectErr, "instance_id", instance.InstanceId)
				}
			} else if restoreInstanceModelBindingsToCVM(ctx, instance, snapshot) {
				slog.Warn("[BatchSetModel] TAT 执行失败，已回滚 DB 和 CVM", "error", injectErr, "instance_id", instance.InstanceId)
			} else {
				slog.Error("[BatchSetModel] TAT 执行失败，DB 已回滚但 CVM 回滚未成功",
					"error", injectErr, "instance_id", instance.InstanceId)
			}
		} else {
			slog.Error("[BatchSetModel] TAT 执行失败，且 DB 回滚未成功，DB 与 CVM 可能不一致",
				"error", injectErr, "instance_id", instance.InstanceId)
		}
		if errors.Is(injectErr, ErrScriptResolveFailed) {
			return newSetModelApplyError(http.StatusBadRequest, hcommon.I18nRichError(injectErr, i18n.MsgModelParseSetModelScriptFailed))
		}
		richErr := hcommon.I18nRichError(injectErr, i18n.MsgModelTATExecuteFailed)
		notifySetModelFailure(ctx, notifyUID, instance, richErr)
		return newSetModelApplyError(http.StatusInternalServerError, richErr)
	}

	if runtimeType == model.AgentTypeHermes {
		go syncHermesLLMToTDAI(hcommon.DetachContext(ctx), instance.InstanceId)
	}
	return nil
}

func setBuiltInPrimaryModelForInstance(ctx context.Context, instance *model.Instance, notifyUID uint, in setModelInput) *setModelApplyError {
	// Pre-configured model mode
	var aiModel model.AIModel
	// 用户主动绑定场景：必须同时启用 + 可见。
	// （存量已绑定的 agent 不走这里，在 LLM 代理路由时仅校验 enabled）
	if model.DB(ctx).Where("id = ? AND enabled = ? AND visible = ?", in.AIModelID, true, true).First(&aiModel).Error != nil {
		return newSetModelApplyError(http.StatusBadRequest, hcommon.I18nError(i18n.MsgModelNotFoundOrDisabled))
	}

	// 可见性校验：按实例绑定的分组过滤（与 HandleModelsList 保持一致）
	filtered := usergroup.FilterModelsByVisibility(ctx, []model.AIModel{aiModel}, instance.GroupID)
	if len(filtered) == 0 {
		slog.Warn("[SetModel] 绑定被拒：模型不在实例分组可见范围",
			"model_id", aiModel.ID, "instance_group_id", instance.GroupID)
		return newSetModelApplyError(http.StatusForbidden, hcommon.I18nError(i18n.MsgModelNoAccess))
	}

	if hcommon.DomainFromCtx(ctx) == "" {
		return newSetModelApplyError(http.StatusInternalServerError, hcommon.I18nError(i18n.MsgModelDomainNotConfigured))
	}

	proxyModel := aiModel
	if instance.ProxyToken != nil {
		proxyModel.APIKey = *instance.ProxyToken
	}
	proxyModel.URL = hcommon.DomainFromCtx(ctx) + "/v1"
	proxyModel.ModelType = "openai-completions"

	// 前置校验：若目标 ai_model_id 已作为 fallback 绑定到该实例，
	// set-model（仅设置 primary）会把当前 primary 行 UPDATE 成相同 (instance_id, ai_model_id, '')
	// 唯一键，撞上已存在的 fallback 行触发 1062。该场景语义上"已被绑定为 fallback"，
	// 直接返回 409，由调用方先 remove-model 解绑该 fallback 后再切主。
	var boundFallback model.InstanceModel
	if err := model.DB(ctx).Where(
		"instance_id = ? AND ai_model_id = ? AND custom_model_id = ? AND role = ?",
		instance.ID, aiModel.ID, "", model.ModelRoleFallback,
	).First(&boundFallback).Error; err == nil {
		return newSetModelApplyError(http.StatusConflict, hcommon.I18nError(i18n.MsgModelAlreadyBoundAsFallback))
	}

	// 保存事务前状态，供 TAT 失败时回滚 DB 使用。
	rb := setModelRollbackState{
		oldInstanceAIModelID:    instance.AIModelID,
		oldInstanceCustomConfig: instance.CustomModelConfig,
	}

	// 先在事务内写 DB（instance_models + instances.ai_model_id）并提交。
	// set-model 只维护 primary：有则更新、无则新增，不把旧 primary 降级为 fallback。
	if err := model.DB(ctx).Transaction(func(tx *gorm.DB) error {
		// 清理目标 ai_model_id 的软删除残留，避免唯一索引冲突。
		// 场景：injectDefaultModel 注入失败后 rollback 遗留的旧版本软删除记录。
		model.CleanSoftDeletedInstanceModel(tx, instance.ID, aiModel.ID)

		var existing model.InstanceModel
		err := tx.Where("instance_id = ? AND role = ?", instance.ID, model.ModelRolePrimary).
			First(&existing).Error

		if err == nil {
			// 已有 primary → 直接更新（先快照旧记录供 TAT 失败回滚）
			snap := existing
			rb.oldPrimary = &snap
			if err := tx.Model(&existing).Updates(map[string]interface{}{
				"ai_model_id":         aiModel.ID,
				"custom_model_id":     "",
				"custom_model_config": "",
			}).Error; err != nil {
				return err
			}
		} else {
			// 无记录 → 新增
			nextSort, err := nextSortOrder(tx, instance.ID)
			if err != nil {
				return err
			}
			im := model.InstanceModel{
				InstanceID: instance.ID,
				AIModelID:  aiModel.ID,
				Role:       model.ModelRolePrimary,
				SortOrder:  nextSort,
			}
			if err := tx.Create(&im).Error; err != nil {
				return err
			}
			rb.createdIMID = im.ID
		}
		// 同步更新 instances.ai_model_id
		if err := tx.Model(instance).Update("ai_model_id", aiModel.ID).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		// DB 落库失败（TAT 尚未下发），事务已回滚，DB 仍为旧状态。
		return newSetModelApplyError(http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgModelSaveBindingFailed))
	}

	// DB 已提交后再生成 TAT 参数，确保 buildSetModelParams 读取到新的 primary。
	// 若参数生成失败，补偿回滚 DB，避免出现"DB 已绑定但 CVM 未生效"的虚绑状态。
	params, modelErr := buildSetModelParams(ctx, proxyModel, instance.ID, false)
	if modelErr != nil {
		if rollbackSetModelDB(ctx, instance, rb) {
			slog.Warn("[SetModel] 生成 TAT 参数失败，已回滚 DB", "error", modelErr, "instance_id", instance.InstanceId)
		} else {
			slog.Error("[SetModel] 生成 TAT 参数失败，且 DB 回滚未成功，DB 与 CVM 可能不一致",
				"error", modelErr, "instance_id", instance.InstanceId)
		}
		return newSetModelApplyError(http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(modelErr))
	}

	// DB 已提交，事务外下发 set_model TAT。若失败则执行补偿事务把 DB 回滚到事务前状态。
	// 统一走 RunAgentScript：Resolve 失败 → 400；RunScript 失败 → 500 + 通知。
	if _, err := RunAgentScript(ctx, instance, "set_model", 60, nil, params); err != nil {
		// 【回滚】TAT 失败 → 把 DB 恢复到事务前：恢复/删除 primary 记录、恢复 instances。
		if rollbackSetModelDB(ctx, instance, rb) {
			slog.Warn("[SetModel] TAT 执行失败，已回滚 DB", "error", err, "instance_id", instance.InstanceId)
		} else {
			// 回滚未成功，DB 仍处于新状态
			slog.Error("[SetModel] TAT 执行失败，且 DB 回滚未成功，DB 与 CVM 可能不一致",
				"error", err, "instance_id", instance.InstanceId)
		}
		if errors.Is(err, ErrScriptResolveFailed) {
			return newSetModelApplyError(http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgModelParseSetModelScriptFailed))
		}
		richErr := hcommon.I18nRichError(err, i18n.MsgModelTATExecuteFailed)
		notifySetModelFailure(ctx, notifyUID, instance, richErr)
		return newSetModelApplyError(http.StatusInternalServerError, richErr)
	}

	// Hermes 场景：异步同步 LLM 配置到 TDAI Gateway（非阻断）
	// 兼容 hermes 的自定义类型走相同路径。
	if model.GetAgentRuntimeType(ctx, instance.AgentType) == model.AgentTypeHermes {
		go syncHermesLLMToTDAI(hcommon.DetachContext(ctx), instance.InstanceId)
	}

	return nil
}

func notifySetModelFailure(ctx context.Context, notifyUID uint, instance *model.Instance, richErr *hcommon.RichError) {
	notifyCtx := hcommon.DetachContext(ctx)
	go createErrorNotification(notifyUID, instance.ID, instance.Name, model.NotifyTypeModelConfigFailed, i18n.T(notifyCtx, i18n.MsgModelConfigFailedTitle), richErr, notifyCtx)
}

func setModelInstanceBinding(w http.ResponseWriter, r *http.Request, instance *model.Instance, notifyUID uint, targetID uint) {
	var target model.InstanceModel
	if model.DB(r.Context()).Where("id = ? AND instance_id = ?", targetID, instance.ID).First(&target).Error != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgModelTargetNotFoundOrNotInInstance))
		return
	}
	if target.Role != model.ModelRolePrimary && target.Role != model.ModelRoleFallback {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalidWithDetail, "role", target.Role))
		return
	}

	aiModelIdStr := r.FormValue("ai_model_id")
	if aiModelIdStr == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}
	aiModelIDInt, err := strconv.Atoi(aiModelIdStr)
	if err != nil || aiModelIDInt < 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}

	oldTarget := target
	oldInstanceAIModelID := instance.AIModelID
	oldInstanceCustomConfig := instance.CustomModelConfig
	updateInstance := target.Role == model.ModelRolePrimary

	if aiModelIDInt > 0 {
		aiModelID := uint(aiModelIDInt)
		var aim model.AIModel
		if model.DB(r.Context()).Where("id = ? AND enabled = ? AND visible = ?", aiModelID, true, true).First(&aim).Error != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgModelNotFoundOrDisabled))
			return
		}
		filtered := usergroup.FilterModelsByVisibility(r.Context(), []model.AIModel{aim}, instance.GroupID)
		if len(filtered) == 0 {
			slog.Warn("[SetModel] 绑定被拒：模型不在实例分组可见范围",
				"model_id", aim.ID, "instance_group_id", instance.GroupID)
			writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgModelNoAccess))
			return
		}

		txErr := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
			model.CleanSoftDeletedInstanceModel(tx, instance.ID, aiModelID)
			var dupCount int64
			if err := tx.Model(&model.InstanceModel{}).
				Where("instance_id = ? AND ai_model_id = ? AND custom_model_id = ? AND id <> ?", instance.ID, aiModelID, "", target.ID).
				Count(&dupCount).Error; err != nil {
				return err
			}
			if dupCount > 0 {
				return errModelAlreadyBound
			}
			if err := tx.Model(&model.InstanceModel{}).Where("id = ?", target.ID).Updates(map[string]interface{}{
				"ai_model_id":         aiModelID,
				"custom_model_id":     "",
				"custom_model_config": "",
			}).Error; err != nil {
				return err
			}
			if updateInstance {
				if err := tx.Model(instance).Update("ai_model_id", aiModelID).Error; err != nil {
					return err
				}
			}
			return nil
		})
		if txErr != nil {
			if errors.Is(txErr, errModelAlreadyBound) || isUniqueConstraintError(txErr) {
				writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgModelAlreadyBound))
				return
			}
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(txErr, i18n.MsgModelSaveBindingFailed))
			return
		}

		if err := injectModelConfigToCVM(r.Context(), instance, &aim, false); err != nil {
			if rollbackSetModelInstanceBinding(r.Context(), instance, oldTarget, oldInstanceAIModelID, oldInstanceCustomConfig, updateInstance) {
				slog.Warn("[SetModel] 指定绑定 TAT 执行失败，已回滚 DB", "error", err, "instance_id", instance.InstanceId, "instance_model_id", target.ID)
			} else {
				slog.Error("[SetModel] 指定绑定 TAT 执行失败，且 DB 回滚未成功，DB 与 CVM 可能不一致",
					"error", err, "instance_id", instance.InstanceId, "instance_model_id", target.ID)
			}
			richErr := hcommon.I18nRichError(err, i18n.MsgModelTATExecuteFailed)
			writeError(w, r, http.StatusInternalServerError, richErr)
			notifyCtx := hcommon.DetachContext(r.Context())
			go createErrorNotification(notifyUID, instance.ID, instance.Name,
				model.NotifyTypeModelConfigFailed, i18n.T(notifyCtx, i18n.MsgModelConfigFailedTitle), richErr, notifyCtx)
			return
		}

		target.AIModelID = aiModelID
		target.CustomModelID = ""
		target.CustomModelConfig = ""
		jsonOK(w, map[string]interface{}{
			"ok":                true,
			"role":              target.Role,
			"instance_model_id": target.ID,
			"binding_id":        resolveBindingRef(r.Context(), target),
			"provider":          aim.Provider,
			"model_id":          aim.ModelID,
			"model_name":        aim.ModelName,
		})
		return
	}

	if !usergroup.ResolvePolicyBoolForGroup(r.Context(), usergroup.PolicyKeyCustomModel, instance.GroupID, model.IsCustomModelEnabled(r.Context())) {
		slog.Warn("[ModelVisibility] 绑定被拒：分组不允许添加自定义模型",
			"instance_group_id", instance.GroupID)
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgModelCustomModelDisabled))
		return
	}
	cfg, parseErr := parseCustomModelFromForm(r)
	if parseErr != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(parseErr))
		return
	}
	if cfg.ModelID == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgModelCustomMissingModelID))
		return
	}

	txErr := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		var dupCount int64
		if err := tx.Model(&model.InstanceModel{}).
			Where("instance_id = ? AND ai_model_id = 0 AND custom_model_id = ? AND id <> ?", instance.ID, cfg.ModelID, target.ID).
			Count(&dupCount).Error; err != nil {
			return err
		}
		if dupCount > 0 {
			return errModelAlreadyBound
		}
		if err := tx.Model(&model.InstanceModel{}).Where("id = ?", target.ID).Updates(map[string]interface{}{
			"ai_model_id":         uint(0),
			"custom_model_id":     cfg.ModelID,
			"custom_model_config": cfg.JSONStr(),
		}).Error; err != nil {
			return err
		}
		if updateInstance {
			if err := tx.Model(instance).Updates(map[string]interface{}{
				"ai_model_id":         uint(0),
				"custom_model_config": cfg.JSONStr(),
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if txErr != nil {
		if errors.Is(txErr, errModelAlreadyBound) || isUniqueConstraintError(txErr) {
			writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgModelCustomAlreadyBound))
			return
		}
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(txErr, i18n.MsgModelSaveCustomBindingFailed))
		return
	}

	tempModel := model.AIModel{
		Provider:          cfg.Provider,
		ModelID:           cfg.ModelID,
		APIKey:            cfg.APIKey,
		URL:               cfg.URL,
		ModelType:         cfg.ModelType,
		InputTypes:        cfg.InputTypesJSON(),
		ContextLen:        cfg.ContextLen,
		MaxTokens:         cfg.MaxTokens,
		CustomHTTPHeaders: cfg.CustomHTTPHeadersJSON(),
	}
	if err := injectModelConfigToCVM(r.Context(), instance, &tempModel, true); err != nil {
		if rollbackSetModelInstanceBinding(r.Context(), instance, oldTarget, oldInstanceAIModelID, oldInstanceCustomConfig, updateInstance) {
			slog.Warn("[SetModel] 指定自定义绑定 TAT 执行失败，已回滚 DB", "error", err, "instance_id", instance.InstanceId, "instance_model_id", target.ID)
		} else {
			slog.Error("[SetModel] 指定自定义绑定 TAT 执行失败，且 DB 回滚未成功，DB 与 CVM 可能不一致",
				"error", err, "instance_id", instance.InstanceId, "instance_model_id", target.ID)
		}
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgModelTATExecuteFailed))
		return
	}

	target.AIModelID = 0
	target.CustomModelID = cfg.ModelID
	target.CustomModelConfig = cfg.JSONStr()
	jsonOK(w, map[string]interface{}{
		"ok":                true,
		"role":              target.Role,
		"instance_model_id": target.ID,
		"binding_id":        resolveBindingRef(r.Context(), target),
		"provider":          cfg.Provider,
		"model_id":          cfg.ModelID,
		"model_name":        cfg.ModelName,
	})
}

func rollbackSetModelInstanceBinding(ctx context.Context, instance *model.Instance, oldTarget model.InstanceModel, oldInstanceAIModelID uint, oldInstanceCustomConfig string, updateInstance bool) bool {
	err := model.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.InstanceModel{}).Where("id = ?", oldTarget.ID).Updates(map[string]interface{}{
			"ai_model_id":         oldTarget.AIModelID,
			"custom_model_id":     oldTarget.CustomModelID,
			"custom_model_config": oldTarget.CustomModelConfig,
			"role":                oldTarget.Role,
			"sort_order":          oldTarget.SortOrder,
		}).Error; err != nil {
			return err
		}
		if updateInstance {
			if err := tx.Model(instance).Updates(map[string]interface{}{
				"ai_model_id":         oldInstanceAIModelID,
				"custom_model_config": oldInstanceCustomConfig,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return err == nil
}

func validateCustomHTTPHeadersMap(headers map[string]string) (map[string]string, error) {
	if len(headers) == 0 {
		return nil, nil
	}
	raw, err := json.Marshal(headers)
	if err != nil {
		return nil, hcommon.I18nError(i18n.MsgModelMarshalCustomHeadersFailed)
	}
	return hcommon.ValidateAndParseCustomHTTPHeaders(string(raw))
}

func setCustomPrimaryModelForInstance(ctx context.Context, instance *model.Instance, notifyUID uint, in setModelInput) *setModelApplyError {
	// 校验是否允许自定义模型：有分组按 custom_model 策略解析，无分组回退站点级开关
	if !usergroup.ResolvePolicyBoolForGroup(ctx, usergroup.PolicyKeyCustomModel, instance.GroupID, model.IsCustomModelEnabled(ctx)) {
		slog.Warn("[ModelVisibility] 绑定被拒：分组不允许添加自定义模型",
			"instance_group_id", instance.GroupID)
		return newSetModelApplyError(http.StatusForbidden, hcommon.I18nError(i18n.MsgModelCustomModelDisabled))
	}

	provider := in.Provider
	if provider == "" {
		provider = hcommon.CustomModelProvider
	}
	modelID := in.ModelID
	modelName := in.ModelName
	if modelName == "" {
		modelName = modelID
	}
	inputTypes, err := hcommon.ValidateInputTypes(in.InputTypes)
	if err != nil {
		return newSetModelApplyError(http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
	}

	// Validate required fields
	if modelID == "" || in.APIKey == "" || in.URL == "" || in.ModelType == "" {
		return newSetModelApplyError(http.StatusBadRequest, hcommon.I18nError(i18n.MsgModelCustomModelFieldsRequired))
	}

	// 【安全】model_id 会被拼接到 shell TAT 参数中，必须白名单校验防命令注入
	if err := hcommon.ValidateCustomModelID(modelID); err != nil {
		return newSetModelApplyError(http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
	}

	// Validate URL format: must be a valid http/https URL
	if err := hcommon.ValidateHTTPURL(in.URL); err != nil {
		return newSetModelApplyError(http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
	}

	// Validate model_type enum
	if err := hcommon.ValidateModelType(in.ModelType); err != nil {
		return newSetModelApplyError(http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
	}

	contextLen := in.ContextLen
	if contextLen <= 0 {
		contextLen = 128000
	}

	maxTokens := in.MaxTokens
	if maxTokens < 0 {
		maxTokens = 0
	}

	customHTTPHeaders, err := validateCustomHTTPHeadersMap(in.CustomHTTPHeaders)
	if err != nil {
		return newSetModelApplyError(http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
	}

	inputTypesJSON, _ := json.Marshal(inputTypes)

	cfg := customModelConfig{
		Provider:          provider,
		ModelID:           modelID,
		ModelName:         modelName,
		APIKey:            in.APIKey,
		URL:               in.URL,
		ModelType:         in.ModelType,
		InputTypes:        inputTypes,
		ContextLen:        contextLen,
		MaxTokens:         maxTokens,
		CustomHTTPHeaders: customHTTPHeaders,
	}

	cfgJSON, marshalErr := json.Marshal(cfg)
	if marshalErr != nil {
		return newSetModelApplyError(http.StatusInternalServerError, hcommon.I18nError(i18n.MsgModelMarshalConfigFailed))
	}

	customHTTPHeadersJSON, err := json.Marshal(customHTTPHeaders)
	if err != nil {
		return newSetModelApplyError(http.StatusInternalServerError, hcommon.I18nError(i18n.MsgModelMarshalCustomHeadersFailed))
	}

	// Build a temporary AIModel for TAT script (not persisted)
	tempModel := model.AIModel{
		Provider:          cfg.Provider,
		ModelID:           cfg.ModelID,
		APIKey:            cfg.APIKey,
		URL:               cfg.URL,
		ModelType:         cfg.ModelType,
		InputTypes:        string(inputTypesJSON),
		ContextLen:        cfg.ContextLen,
		MaxTokens:         cfg.MaxTokens,
		CustomHTTPHeaders: string(customHTTPHeadersJSON),
	}

	// 调整后的执行顺序（与内置模型分支一致）：先在事务内写 DB（instance_models + instances）并提交，
	// 提交成功后再在事务外下发 set_model TAT。
	// 若 TAT 执行失败，则执行补偿事务把 DB 回滚到事务前状态，保证两表与 CVM 一致。
	// 与内置模型分支一致：hermes/ace 只保留一条 primary 记录。
	rb := setModelRollbackState{
		oldInstanceAIModelID:    instance.AIModelID,
		oldInstanceCustomConfig: instance.CustomModelConfig,
	}
	if err := model.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.InstanceModel
		err := tx.Where("instance_id = ? AND role = ?", instance.ID, model.ModelRolePrimary).
			First(&existing).Error

		if err == nil {
			// 已有 primary → 直接更新为自定义模型（先快照旧记录供 TAT 失败回滚）
			snap := existing
			rb.oldPrimary = &snap
			if err := tx.Model(&existing).Updates(map[string]interface{}{
				"ai_model_id":         0,
				"custom_model_id":     cfg.ModelID,
				"custom_model_config": string(cfgJSON),
			}).Error; err != nil {
				return err
			}
		} else {
			// 无记录 → 新增
			nextSort, err := nextSortOrder(tx, instance.ID)
			if err != nil {
				return err
			}
			im := model.InstanceModel{
				InstanceID:        instance.ID,
				AIModelID:         0,
				CustomModelID:     cfg.ModelID,
				Role:              model.ModelRolePrimary,
				SortOrder:         nextSort,
				CustomModelConfig: string(cfgJSON),
			}
			if err := tx.Create(&im).Error; err != nil {
				return err
			}
			rb.createdIMID = im.ID
		}
		// 同步更新 instances
		if err := tx.Model(instance).Updates(map[string]interface{}{
			"ai_model_id":         0,
			"custom_model_config": string(cfgJSON),
		}).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		// DB 落库失败（TAT 尚未下发），事务已回滚，DB 仍为旧状态。
		return newSetModelApplyError(http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgModelSaveCustomBindingFailed))
	}

	params, modelErr := buildSetModelParams(ctx, tempModel, instance.ID, true)
	if modelErr != nil {
		if rollbackSetModelDB(ctx, instance, rb) {
			slog.Warn("[SetModel] 生成自定义模型 TAT 参数失败，已回滚 DB", "error", modelErr, "instance_id", instance.InstanceId)
		} else {
			slog.Error("[SetModel] 生成自定义模型 TAT 参数失败，且 DB 回滚未成功，DB 与 CVM 可能不一致",
				"error", modelErr, "instance_id", instance.InstanceId)
		}
		return newSetModelApplyError(http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(modelErr))
	}

	// DB 已提交，事务外下发 set_model TAT（自定义模型分支）。若失败则补偿回滚 DB。
	if _, err := RunAgentScript(ctx, instance, "set_model", 60, nil, params); err != nil {
		// 【回滚】TAT 失败 → 把 DB 恢复到事务前：恢复/删除 primary 记录、恢复 instances。
		if rollbackSetModelDB(ctx, instance, rb) {
			slog.Warn("[SetModel] 设置自定义模型 TAT 执行失败，已回滚 DB", "error", err, "instance_id", instance.InstanceId)
		} else {
			slog.Error("[SetModel] 设置自定义模型 TAT 执行失败，且 DB 回滚未成功，DB 与 CVM 可能不一致",
				"error", err, "instance_id", instance.InstanceId)
		}
		if errors.Is(err, ErrScriptResolveFailed) {
			return newSetModelApplyError(http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgModelParseSetModelScriptFailed))
		}
		richErr := hcommon.I18nRichError(err, i18n.MsgModelTATExecuteFailed)
		notifySetModelFailure(ctx, notifyUID, instance, richErr)
		return newSetModelApplyError(http.StatusInternalServerError, richErr)
	}

	// Hermes 场景：异步同步 LLM 配置到 TDAI Gateway（非阻断）
	// 兼容 hermes 的自定义类型走相同路径。
	if model.GetAgentRuntimeType(ctx, instance.AgentType) == model.AgentTypeHermes {
		go syncHermesLLMToTDAI(hcommon.DetachContext(ctx), instance.InstanceId)
	}

	return nil
}

// setModelRollbackState 保存 set_model / customModel 事务前的状态，
// 供事务外下发 set_model TAT 失败时补偿回滚 DB 使用。
type setModelRollbackState struct {
	oldPrimary              *model.InstanceModel // 事务前 primary 记录快照；nil 表示本次为新增
	createdIMID             uint                 // 新增记录主键（oldPrimary == nil 时有效）
	oldInstanceAIModelID    uint                 // 旧 instances.ai_model_id
	oldInstanceCustomConfig string               // 旧 instances.custom_model_config
}

// rollbackSetModelDB 在事务外下发 set_model TAT 失败后，将 instance_models / instances
// 恢复到事务前状态，返回回滚是否成功。与 addModel / switchPrimaryModel 的回滚策略一致：
// 新增分支物理删除（避免软删除残留占用唯一索引导致重试报"已绑定"），更新分支还原旧字段。
//
// 内部统一 DetachContext：RunAgentScript 可能耗时较久，期间客户端若刷新/关闭页面会取消
// r.Context()，导致补偿事务因 "context canceled" 失败、回滚被静默跳过而 DB 停留在新状态。
// 补偿事务的生命周期不应依赖客户端连接。
func rollbackSetModelDB(ctx context.Context, instance *model.Instance, st setModelRollbackState) bool {
	err := model.DB(hcommon.DetachContext(ctx)).Transaction(func(tx *gorm.DB) error {
		if st.oldPrimary != nil {
			// 更新分支：还原旧 primary 字段
			if err := tx.Model(&model.InstanceModel{}).Where("id = ?", st.oldPrimary.ID).
				Updates(map[string]interface{}{
					"ai_model_id":         st.oldPrimary.AIModelID,
					"custom_model_id":     st.oldPrimary.CustomModelID,
					"custom_model_config": st.oldPrimary.CustomModelConfig,
				}).Error; err != nil {
				return err
			}
		} else {
			// 新增分支：物理删除新建记录
			if err := tx.Unscoped().Delete(&model.InstanceModel{}, st.createdIMID).Error; err != nil {
				return err
			}
		}
		return tx.Model(instance).Updates(map[string]interface{}{
			"ai_model_id":         st.oldInstanceAIModelID,
			"custom_model_config": st.oldInstanceCustomConfig,
		}).Error
	})
	if err != nil {
		slog.Error("[SetModel] 回滚 DB 失败", "error", err, "instance_id", instance.InstanceId)
		return false
	}
	return true
}

// injectDefaultModelPollInterval / injectDefaultModelMaxWait 是可被单测覆盖的
// 轮询参数。生产默认每 10 秒查一次、最多等 10 分钟；单测可以在 setup 里缩短。
var (
	injectDefaultModelPollInterval = 10 * time.Second
	injectDefaultModelMaxWait      = 10 * time.Minute
)

// injectDefaultModel 等待实例 Agent 就绪后自动注入默认模型配置。
// 在 goroutine 中执行，每 injectDefaultModelPollInterval 查一次 DB，
// 最多等 injectDefaultModelMaxWait。
//
// 所有失败分支统一通过 rollbackDefaultModelIfIntact 尝试精确回滚，避免
// "DB 已绑定但 TAT 侧未实际生效"的虚绑状态（三期一致性修复）。
func injectDefaultModel(ctx context.Context, instancePK uint, modelID uint) {
	failureReason := "agent_ready_timeout" // 默认为超时，成功或无需回滚时清空
	defer func() {
		if failureReason != "" {
			rollbackDefaultModelIfIntact(ctx, instancePK, modelID, failureReason)
		}
	}()

	deadline := time.Now().Add(injectDefaultModelMaxWait)
	for time.Now().Before(deadline) {
		time.Sleep(injectDefaultModelPollInterval)

		// 查询实例状态
		var inst model.Instance
		if err := model.DB(ctx).First(&inst, instancePK).Error; err != nil {
			slog.Warn("[DefaultModel] 实例不存在，放弃注入（无需回滚）",
				"instance_pk", instancePK, "error", err)
			failureReason = "" // 实例已删除，无需回滚
			return
		}
		// 校验实例类型是否允许"自动注入默认模型"。
		// 入口 CreateInstance 已做同样检查，这里是防御性二次 guard：
		// 防止实例创建后 agent_type 被修改的边界场景。
		if !model.AgentTypeSupportsDefaultModelInjection(ctx, inst.AgentType) {
			slog.Error("[DefaultModel] 实例类型不支持默认模型自动注入，触发回滚",
				"instance_pk", instancePK, "agent_type", inst.AgentType,
				"failure_reason", "agent_type_changed")
			failureReason = "agent_type_changed"
			return
		}
		if inst.AgentReady != 1 {
			continue
		}

		// Agent 就绪，验证模型仍存在且启用 + 可见
		// （默认模型注入属于主动绑定场景，与 setModel 一致）
		var aiModel model.AIModel
		if err := model.DB(ctx).Where("id = ? AND enabled = ? AND visible = ?", modelID, true, true).First(&aiModel).Error; err != nil {
			slog.Error("[DefaultModel] 默认模型已被删除、禁用或不可见，触发回滚",
				"model_id", modelID, "instance_pk", instancePK,
				"failure_reason", "model_disabled_or_deleted")
			failureReason = "model_disabled_or_deleted"
			return
		}

		// 构建代理模型参数（复用 HandleSetModel 的逻辑）
		proxyModel := aiModel
		if inst.ProxyToken != nil {
			proxyModel.APIKey = *inst.ProxyToken
		}
		proxyModel.URL = hcommon.DomainFromCtx(ctx) + "/v1"
		proxyModel.ModelType = "openai-completions"

		params, err := buildSetModelParams(ctx, proxyModel, inst.ID, false)
		if err != nil {
			slog.Error("[DefaultModel] 构建参数失败，触发回滚",
				"error", err, "instance_pk", instancePK,
				"failure_reason", "build_params_failed")
			failureReason = "build_params_failed"
			return
		}

		// 确保 runtimeUser 已探测：agent_ready=1 时 detectAndSaveRuntimeUser 异步执行，
		// 此处可能还未完成，同步调用 ensureRuntimeUser 避免以错误用户执行脚本。
		inst.RuntimeUser = ensureRuntimeUser(ctx, inst.ID, inst.InstanceId, inst.AgentType)

		// v7：按 agent_type 分派 set_model 脚本。
		// 上游已有 AgentTypeSupportsDefaultModelInjection guard，
		// Resolve 失败分支仅用于兜底日志（理论不可达）。
		if _, err := RunAgentScript(ctx, &inst, "set_model", 60, nil, params); err != nil {
			if errors.Is(err, ErrScriptResolveFailed) {
				slog.Error("[DefaultModel] 解析脚本失败（理论不可达，因已 guard），触发回滚",
					"agent_type", inst.AgentType, "error", err, "instance_pk", instancePK,
					"failure_reason", "script_resolve_failed")
				failureReason = "script_resolve_failed"
				return
			}
			slog.Error("[DefaultModel] TAT 注入失败，触发回滚",
				"error", errors.Unwrap(err), "instance_pk", instancePK,
				"instance_id", inst.InstanceId,
				"failure_reason", "tat_script_failed")
			failureReason = "tat_script_failed"
			return
		}

		slog.Info("[DefaultModel] 默认模型注入成功", "instance_pk", instancePK, "instance_id", inst.InstanceId, "model_id", aiModel.ModelID)
		failureReason = "" // 成功，不触发回滚
		return
	}

	// 等待超时：agent_ready 一直是 0，failureReason 保持默认 "agent_ready_timeout"
	slog.Error("[DefaultModel] 等待实例就绪超时，触发回滚",
		"instance_pk", instancePK, "model_id", modelID,
		"failure_reason", "agent_ready_timeout")
}

// rollbackDefaultModelIfIntact 精确回滚默认模型注入意图。
//
// 三期一致性修复：injectDefaultModel 是 HandleCreate 主事务之后异步 goroutine 运行，
// 失败时不能粗暴地把 ai_model_id 置零 —— 用户在 10 分钟 poll 窗口内可能已经
// 主动换过模型，粗暴回滚会覆盖用户选择。
//
// 精确条件：instances.ai_model_id 仍等于 modelID（上游事务写入的默认模型 ID）。
// 若用户已改过：ai_model_id != modelID，跳过回滚，保留用户意图。
func rollbackDefaultModelIfIntact(ctx context.Context, instancePK, modelID uint, reason string) {
	if instancePK == 0 || modelID == 0 {
		slog.Warn("[DefaultModel] 回滚参数缺失，跳过",
			"instance_pk", instancePK, "model_id", modelID, "reason", reason)
		return
	}

	var inst model.Instance
	if err := model.DB(ctx).First(&inst, instancePK).Error; err != nil {
		slog.Info("[DefaultModel] 实例已被删除或查询失败，无需回滚",
			"instance_pk", instancePK, "reason", reason, "error", err)
		return
	}

	// 条件：ai_model_id 仍等于我们注入的 modelID
	if inst.AIModelID != modelID {
		slog.Info("[DefaultModel] 用户已切换到其他模型，保留用户选择，不回滚",
			"instance_pk", instancePK, "current_model_id", inst.AIModelID,
			"intended_model_id", modelID, "reason", reason)
		return
	}

	if err := model.DB(ctx).Transaction(func(tx *gorm.DB) error {
		// 物理删除对应的 primary 记录（精确匹配，避免误删 fallback）。
		// 使用物理删除而非软删除，防止残留行占用唯一索引导致后续绑定同模型时 Duplicate entry。
		if err := model.HardDeleteInstanceModelByKey(tx, instancePK, modelID, model.ModelRolePrimary); err != nil {
			return err
		}
		return tx.Model(&model.Instance{}).Where("id = ? AND ai_model_id = ?", instancePK, modelID).
			Update("ai_model_id", 0).Error
	}); err != nil {
		slog.Error("[DefaultModel] 回滚 ai_model_id 失败",
			"instance_pk", instancePK, "model_id", modelID,
			"reason", reason, "error", err)
		return
	}
	slog.Warn("[DefaultModel] 注入失败，已精确回滚 ai_model_id=0",
		"instance_pk", instancePK, "model_id", modelID, "reason", reason)
}

// ========== 多模型 Fallback 接口（v2.0 新增） ==========

// applyInitialModels waits for the new Agent and then performs the exact same
// built-in add-model operation used when a user configures models manually.
// The first model becomes primary and later models become fallbacks. Each
// failed operation rolls back its own binding and stops the sequence.
func applyInitialModels(ctx context.Context, instanceID uint, modelIDs []uint) {
	instance, err := waitForCreatePresetInstance(ctx, instanceID)
	if err != nil {
		var persisted model.Instance
		if model.DB(ctx).First(&persisted, instanceID).Error == nil {
			notifyInitialModelFailure(ctx, &persisted, hcommon.I18nError(i18n.MsgModelTATExecuteFailed))
		}
		return
	}
	if !model.AgentTypeSupportsModel(ctx, instance.AgentType) {
		notifyInitialModelFailure(ctx, instance,
			hcommon.I18nError(i18n.MsgAgentTypeDoNotSupportModelConfigWithDetail,
				model.GetAgentTypeDisplayName(ctx, instance.AgentType)))
		return
	}

	for _, modelID := range modelIDs {
		if _, applyErr := applyBuiltinModel(ctx, instance, modelID); applyErr != nil {
			notifyInitialModelFailure(ctx, instance, applyErr.err)
			return
		}
	}
	slog.Info("[InitialModels] 初始模型配置下发成功",
		"instance_id", instance.ID,
		"model_count", len(modelIDs),
	)
}

func notifyInitialModelFailure(ctx context.Context, instance *model.Instance, err error) {
	if instance == nil {
		return
	}
	richErr := hcommon.EnsureRichErrorOrPanic(err)
	slog.Warn("[InitialModels] 初始模型配置下发失败", "instance_id", instance.ID)
	createErrorNotification(instance.UserID, instance.ID, instance.Name, model.NotifyTypeModelConfigFailed, i18n.T(ctx, i18n.MsgModelConfigFailedTitle), richErr, ctx)
}

type builtinModelApplyResult struct {
	model   model.AIModel
	binding model.InstanceModel
	role    string
}

type builtinModelApplyError struct {
	status int
	err    *hcommon.RichError
	notify bool
}

// applyBuiltinModel performs the same built-in-model operation for both
// add-model requests and create-time presets. With no existing primary, the
// first model becomes primary; later models become fallbacks.
func applyBuiltinModel(ctx context.Context, instance *model.Instance, aiModelID uint) (builtinModelApplyResult, *builtinModelApplyError) {
	var result builtinModelApplyResult
	if model.DB(ctx).Where("id = ? AND enabled = ? AND visible = ?", aiModelID, true, true).First(&result.model).Error != nil {
		return result, &builtinModelApplyError{
			status: http.StatusBadRequest,
			err:    hcommon.I18nError(i18n.MsgModelNotFoundDisabledOrInvisible),
		}
	}
	if len(usergroup.FilterModelsByVisibility(ctx, []model.AIModel{result.model}, instance.GroupID)) == 0 {
		return result, &builtinModelApplyError{
			status: http.StatusForbidden,
			err:    hcommon.I18nError(i18n.MsgModelNoAccess),
		}
	}

	result.role = model.ModelRoleFallback
	txErr := model.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var duplicateCount int64
		if err := tx.Model(&model.InstanceModel{}).
			Where("instance_id = ? AND ai_model_id = ? AND custom_model_id = ?", instance.ID, aiModelID, "").
			Count(&duplicateCount).Error; err != nil {
			return err
		}
		if duplicateCount > 0 {
			return errModelAlreadyBound
		}

		var primaryCount int64
		if err := tx.Model(&model.InstanceModel{}).
			Where("instance_id = ? AND role = ?", instance.ID, model.ModelRolePrimary).
			Count(&primaryCount).Error; err != nil {
			return err
		}
		if primaryCount == 0 {
			result.role = model.ModelRolePrimary
		}

		nextSort, err := nextSortOrder(tx, instance.ID)
		if err != nil {
			return err
		}
		result.binding = model.InstanceModel{
			InstanceID: instance.ID,
			AIModelID:  aiModelID,
			Role:       result.role,
			SortOrder:  nextSort,
		}
		return tx.Create(&result.binding).Error
	})
	if txErr != nil {
		if errors.Is(txErr, errModelAlreadyBound) || isUniqueConstraintError(txErr) {
			return result, &builtinModelApplyError{
				status: http.StatusConflict,
				err:    hcommon.I18nError(i18n.MsgModelAlreadyBound),
			}
		}
		return result, &builtinModelApplyError{
			status: http.StatusInternalServerError,
			err:    hcommon.I18nRichError(txErr, i18n.MsgModelCreateBindingFailed),
		}
	}

	if err := injectModelConfigToCVM(ctx, instance, &result.model, false); err != nil {
		if rbErr := model.DB(ctx).Unscoped().Delete(&result.binding).Error; rbErr != nil {
			slog.Error("[AddModel] TAT 失败后回滚 DB 删除 im 失败", "error", rbErr, "instance_model_id", result.binding.ID)
		}
		return result, &builtinModelApplyError{
			status: http.StatusInternalServerError,
			err:    hcommon.I18nRichError(err, i18n.MsgModelTATExecuteFailed),
			notify: true,
		}
	}

	if result.role == model.ModelRolePrimary {
		model.DB(ctx).Model(instance).Update("ai_model_id", aiModelID)
	}
	return result, nil
}

func modelFallbackSupportError(ctx context.Context, agentType, agentVersion string) *hcommon.RichError {
	if model.GetAgentRuntimeType(ctx, agentType) != model.AgentTypeOpenClaw {
		return hcommon.I18nError(
			i18n.MsgModelFallbackUnsupportedByAgentType,
			model.GetAgentTypeDisplayName(ctx, agentType),
		)
	}
	if strings.HasPrefix(agentVersion, "3.28") {
		return hcommon.I18nError(i18n.MsgModelAgentFallbackUnsupported)
	}
	return nil
}

// HandleAddModel 为实例添加一个模型。首个自动设为 primary，后续为 fallback。
func HandleAddModel(w http.ResponseWriter, r *http.Request) {
	handleAddModel(w, r, defaultStatusResolver)
}

func handleAddModel(w http.ResponseWriter, r *http.Request, resolver instanceStatusResolver) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	addModel(w, r, instance, user.ID, resolver)
}

// addModel 新增模型绑定（首个自动 primary，后续 fallback），用户端和管控端共享。
func addModel(w http.ResponseWriter, r *http.Request, instance *model.Instance, notifyUID uint, resolver instanceStatusResolver) {
	// 实例类型校验
	if err := checkInstanceSupportsModel(r.Context(), instance); err != nil {
		writeError(w, r, http.StatusForbidden, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 本地实例：模型配置需下发到 CVM Agent，本地 agent 不支持。
	if rejectLocalOrWrite(w, r, instance) {
		return
	}
	// 状态准入：仅 running 状态允许新增模型
	if _, err := requireInstanceRunning(r.Context(), instance, resolver); err != nil {
		writeAgentGuardError(w, r, err)
		return
	}

	// add-model 在已有绑定时会产生 fallback。Hermes/ACE 只有单个激活模型，
	// 仅允许在无绑定时通过该入口设置第一个模型；OpenClaw 3.28 保持原有禁用规则。
	if model.GetAgentRuntimeType(r.Context(), instance.AgentType) != model.AgentTypeOpenClaw {
		var bindingCount int64
		if err := model.DB(r.Context()).Model(&model.InstanceModel{}).
			Where("instance_id = ?", instance.ID).Count(&bindingCount).Error; err != nil {
			writeError(w, r, http.StatusInternalServerError,
				hcommon.I18nRichError(err, i18n.MsgModelQueryListFailed))
			return
		}
		if bindingCount > 0 {
			writeError(w, r, http.StatusConflict,
				hcommon.I18nError(
					i18n.MsgModelFallbackUnsupportedByAgentType,
					model.GetAgentTypeDisplayName(r.Context(), instance.AgentType),
				))
			return
		}
	} else if err := modelFallbackSupportError(r.Context(), instance.AgentType, instance.AgentVersion); err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}

	aiModelIdStr := r.FormValue("ai_model_id")
	if aiModelIdStr == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}
	aiModelIDInt, err := strconv.Atoi(aiModelIdStr)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}
	aiModelID := uint(aiModelIDInt)

	if aiModelID == 0 {
		customModelForAddModel(w, r, instance, notifyUID)
		return
	}

	result, applyErr := applyBuiltinModel(r.Context(), instance, aiModelID)
	if applyErr != nil {
		writeError(w, r, applyErr.status, applyErr.err)
		if applyErr.notify {
			notifyCtx := hcommon.DetachContext(r.Context())
			go createErrorNotification(notifyUID, instance.ID, instance.Name,
				model.NotifyTypeModelConfigFailed, i18n.T(notifyCtx, i18n.MsgModelAddFailedTitle), applyErr.err, notifyCtx)
		}
		return
	}

	jsonOK(w, map[string]interface{}{
		"ok":                true,
		"role":              result.role,
		"instance_model_id": result.binding.ID,
		"binding_id":        resolveBindingRef(r.Context(), result.binding),
		"provider":          result.model.Provider,
		"model_id":          result.model.ModelID,
		"model_name":        result.model.ModelName,
	})
}

// customModelForAddModel 处理自定义模型的 add-model 逻辑，用户端和管控端共享。
func customModelForAddModel(w http.ResponseWriter, r *http.Request, instance *model.Instance, notifyUID uint) {
	// 校验是否允许自定义模型：有分组按 custom_model 策略解析，无分组回退站点级开关
	if !usergroup.ResolvePolicyBoolForGroup(r.Context(), usergroup.PolicyKeyCustomModel, instance.GroupID, model.IsCustomModelEnabled(r.Context())) {
		slog.Warn("[ModelVisibility] 绑定被拒：分组不允许添加自定义模型",
			"instance_group_id", instance.GroupID)
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgModelCustomModelDisabled))
		return
	}

	// 复用自定义模型字段解析逻辑（与 handleCustomModel 一致）
	cfg, err := parseCustomModelFromForm(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 自定义模型的业务键为 cfg.ModelID（用户提交的 model_id 字段）
	if cfg.ModelID == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgModelCustomMissingModelID))
		return
	}

	var im model.InstanceModel
	var role = model.ModelRoleFallback

	// 事务：重复绑定校验 + role 判定 + SortOrder 计算 + 插入
	txErr := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		// 1) 通过联合唯一键 (instance_id, ai_model_id=0, custom_model_id=cfg.ModelID) 防重
		var dupCount int64
		if err := tx.Model(&model.InstanceModel{}).
			Where("instance_id = ? AND ai_model_id = 0 AND custom_model_id = ?", instance.ID, cfg.ModelID).
			Count(&dupCount).Error; err != nil {
			return err
		}
		if dupCount > 0 {
			return errModelAlreadyBound
		}

		// 2) role 判定：基于 primary 是否存在
		var primaryCount int64
		if err := tx.Model(&model.InstanceModel{}).
			Where("instance_id = ? AND role = ?", instance.ID, model.ModelRolePrimary).
			Count(&primaryCount).Error; err != nil {
			return err
		}
		if primaryCount == 0 {
			role = model.ModelRolePrimary
		}

		// 3) SortOrder = MAX+1
		nextSort, err := nextSortOrder(tx, instance.ID)
		if err != nil {
			return err
		}

		im = model.InstanceModel{
			InstanceID:        instance.ID,
			CustomModelID:     cfg.ModelID,
			CustomModelConfig: cfg.JSONStr(),
			Role:              role,
			SortOrder:         nextSort,
		}
		return tx.Create(&im).Error
	})
	if txErr != nil {
		if errors.Is(txErr, errModelAlreadyBound) || isUniqueConstraintError(txErr) {
			writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgModelCustomAlreadyBound))
			return
		}
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(txErr, i18n.MsgModelCreateCustomBindingFailed))
		return
	}

	// 构建 AIModel 并下发 TAT
	tempModel := model.AIModel{
		Provider:          cfg.Provider,
		ModelID:           cfg.ModelID,
		APIKey:            cfg.APIKey,
		URL:               cfg.URL,
		ModelType:         cfg.ModelType,
		InputTypes:        cfg.InputTypesJSON(),
		ContextLen:        cfg.ContextLen,
		MaxTokens:         cfg.MaxTokens,
		CustomHTTPHeaders: cfg.CustomHTTPHeadersJSON(),
	}
	// 用户侧自定义模型分支：isUserCustomModel=true，保留用户填写的 URL / APIKey，不走 hatchery 代理。
	if err := injectModelConfigToCVM(r.Context(), instance, &tempModel, true); err != nil {
		// 回滚 DB（物理删除，避免软删除记录占用唯一索引导致用户重试时报"已绑定"）
		model.DB(r.Context()).Unscoped().Delete(&im)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgModelTATExecuteFailed))
		return
	}

	bindingRef := resolveBindingRef(r.Context(), im)

	jsonOK(w, map[string]interface{}{
		"ok":                true,
		"role":              role,
		"instance_model_id": im.ID,
		"binding_id":        bindingRef,
		"provider":          cfg.Provider,
		"model_id":          cfg.ModelID,
		"model_name":        cfg.ModelName,
	})
}

// HandleSwitchPrimaryModel 将指定模型切换为主模型（原主模型降级为 fallback）。
func HandleSwitchPrimaryModel(w http.ResponseWriter, r *http.Request) {
	handleSwitchPrimaryModel(w, r, defaultStatusResolver)
}

func handleSwitchPrimaryModel(w http.ResponseWriter, r *http.Request, resolver instanceStatusResolver) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	switchPrimaryModel(w, r, instance, user.ID, resolver)
}

// switchPrimaryModel 切换主备模型，用户端和管控端共享。
func switchPrimaryModel(w http.ResponseWriter, r *http.Request, instance *model.Instance, notifyUID uint, resolver instanceStatusResolver) {
	// 本地实例：模型配置需下发到 CVM Agent，本地 agent 不支持。
	if rejectLocalOrWrite(w, r, instance) {
		return
	}
	// 状态准入：仅 running 状态允许切换主模型
	if _, err := requireInstanceRunning(r.Context(), instance, resolver); err != nil {
		writeAgentGuardError(w, r, err)
		return
	}

	idStr := r.FormValue("instance_model_id")
	targetID, err := strconv.Atoi(idStr)
	if err != nil || targetID <= 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgModelInvalidInstanceModelID))
		return
	}

	// 查找目标模型绑定
	var targetModel model.InstanceModel
	if model.DB(r.Context()).Where("id = ? AND instance_id = ?", targetID, instance.ID).First(&targetModel).Error != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgModelTargetNotFoundOrNotInInstance))
		return
	}

	if targetModel.Role == model.ModelRolePrimary {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgModelCannotSwitchToSelf))
		return
	}

	// 事务前保存旧 primary 的绑定记录，确保后续可以精确定位（尤其是自定义模型 ai_model_id=0 场景）
	var oldPrimary model.InstanceModel
	hasOldPrimary := model.DB(r.Context()).Where("instance_id = ? AND role = ?", instance.ID, model.ModelRolePrimary).
		First(&oldPrimary).Error == nil

	// 保存事务前状态用于 TAT 失败时回滚
	oldInstanceAIModelID := instance.AIModelID

	// 开启事务，任一 UPDATE 失败立即 Rollback
	tx := model.DB(r.Context()).Begin()
	if tx.Error != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(tx.Error, i18n.MsgModelBeginTxFailed))
		return
	}
	// 兜底：函数退出时若事务未提交，确保不留悬挂事务
	committed := false
	defer func() {
		if !committed {
			tx.Rollback()
		}
	}()

	// 1. 当前 primary 改为 fallback
	if err := tx.Model(&model.InstanceModel{}).
		Where("instance_id = ? AND role = ?", instance.ID, model.ModelRolePrimary).
		Update("role", model.ModelRoleFallback).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgModelDemoteOldPrimaryFailed))
		return
	}

	// 2. 目标模型改为 primary
	if err := tx.Model(&targetModel).Update("role", model.ModelRolePrimary).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgModelPromoteTargetFailed))
		return
	}

	// 3. 同步更新 instances.ai_model_id
	newAIModelID := uint(0)
	if targetModel.AIModelID > 0 {
		newAIModelID = targetModel.AIModelID
	}
	if err := tx.Model(instance).Update("ai_model_id", newAIModelID).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgModelUpdateInstancePrimaryFailed))
		return
	}

	if err := tx.Commit().Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgModelCommitTxFailed))
		return
	}
	committed = true

	// 重启 Gateway 生效（TAT 调用 switch_model.sh）
	primaryRef := resolveBindingRef(r.Context(), targetModel)
	imagePrimary, imageFallbacks, err := buildImageModelRefs(r.Context(), instance.ID, primaryRef)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgModelGenImageModelParamFailed))
		return
	}
	if _, err := RunScript(r.Context(), instance.InstanceId, "switch_model.sh", 60, instance.RuntimeUser, nil, map[string]string{
		"primary":           primaryRef,
		"fallbacksb64":      base64.StdEncoding.EncodeToString([]byte(getFallbackRefsJSON(r.Context(), instance))),
		"imageprimary":      imagePrimary,
		"imagefallbacksb64": base64.StdEncoding.EncodeToString([]byte(imageFallbacks)),
	}); err != nil {
		// 【回滚】TAT 失败 → 把 DB 状态恢复到事务前
		// 注：TAT 侧是否部分生效未知，但 DB 先恢复成最终一致状态，下一次切换会重新下发
		rbTx := model.DB(r.Context()).Begin()
		if rbTx.Error == nil {
			rbOK := true
			if err2 := rbTx.Model(&targetModel).Update("role", model.ModelRoleFallback).Error; err2 != nil {
				slog.Error("[SwitchModel] 回滚目标模型失败", "error", err2)
				rbTx.Rollback()
				rbOK = false
			}
			if rbOK && hasOldPrimary {
				if err2 := rbTx.Model(&model.InstanceModel{}).
					Where("id = ?", oldPrimary.ID).
					Update("role", model.ModelRolePrimary).Error; err2 != nil {
					slog.Error("[SwitchModel] 回滚原主模型失败", "error", err2)
					rbTx.Rollback()
					rbOK = false
				}
			}
			if rbOK {
				if err2 := rbTx.Model(instance).Update("ai_model_id", oldInstanceAIModelID).Error; err2 != nil {
					slog.Error("[SwitchModel] 回滚 ai_model_id 失败", "error", err2)
					rbTx.Rollback()
					rbOK = false
				}
			}
			if rbOK {
				if cErr := rbTx.Commit().Error; cErr != nil {
					slog.Error("[SwitchModel] TAT 失败后回滚事务提交失败",
						"error", cErr, "instance_id", instance.InstanceId)
					rbTx.Rollback()
				}
			}
		}
		richErr := hcommon.I18nRichError(err, i18n.MsgModelTATExecuteFailed)
		slog.Error("[SwitchModel] TAT 执行失败，已回滚 DB", "error", err, "instance_id", instance.InstanceId)
		writeError(w, r, http.StatusInternalServerError, richErr)
		notifyCtx := hcommon.DetachContext(r.Context())
		go createErrorNotification(notifyUID, instance.ID, instance.Name,
			model.NotifyTypeModelConfigFailed, i18n.T(notifyCtx, i18n.MsgModelSwitchFailedTitle), richErr, notifyCtx)
		return
	}

	// 构建响应：通过事务前保存的 oldPrimary.ID 精确定位降级后的记录
	demotedRef := ""
	var demotedModel model.InstanceModel
	if hasOldPrimary {
		if model.DB(r.Context()).First(&demotedModel, oldPrimary.ID).Error == nil {
			demotedRef = resolveBindingRef(r.Context(), demotedModel)
		}
	}

	jsonOK(w, map[string]interface{}{
		"ok": true,
		"new_primary": map[string]interface{}{
			"binding_id":        resolveBindingRef(r.Context(), targetModel),
			"instance_model_id": targetModel.ID,
			"provider":          getModelProviderName(r.Context(), targetModel),
			"model_id":          getModelModelID(r.Context(), targetModel),
			"model_name":        getModelModelName(r.Context(), targetModel),
			"role":              model.ModelRolePrimary,
		},
		"demoted_to_fallback": buildDemotedResponse(r.Context(), demotedModel, demotedRef),
	})
}

// HandleDelModel 从实例删除指定模型绑定。
func HandleDelModel(w http.ResponseWriter, r *http.Request) {
	handleDelModel(w, r, defaultStatusResolver)
}

func handleDelModel(w http.ResponseWriter, r *http.Request, resolver instanceStatusResolver) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	deleteModel(w, r, instance, user.ID, resolver)
}

// deleteModel 删除模型绑定，用户端和管控端共享。
func deleteModel(w http.ResponseWriter, r *http.Request, instance *model.Instance, notifyUID uint, resolver instanceStatusResolver) {
	// 本地实例：模型配置需下发到 CVM Agent，本地 agent 不支持。
	if rejectLocalOrWrite(w, r, instance) {
		return
	}
	// 状态准入：仅 running 状态允许删除模型
	if _, err := requireInstanceRunning(r.Context(), instance, resolver); err != nil {
		writeAgentGuardError(w, r, err)
		return
	}

	idStr := r.FormValue("instance_model_id")
	targetID, err := strconv.Atoi(idStr)
	if err != nil || targetID <= 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgModelInvalidInstanceModelID))
		return
	}

	var targetModel model.InstanceModel
	if model.DB(r.Context()).Where("id = ? AND instance_id = ?", targetID, instance.ID).First(&targetModel).Error != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgModelTargetNotFoundOrNotInInstance))
		return
	}

	wasPrimary := targetModel.Role == model.ModelRolePrimary
	var promotedModel *model.InstanceModel
	var totalCount int64

	// 被删模型的 provider key 需在删除前解析（删除后 instance_model 记录已不存在）。
	deletedProviderKey := resolveProviderKey(r.Context(), targetModel)
	// 保存事务前状态，供 TAT 同步失败时回滚 DB 使用。
	originalAIModelID := instance.AIModelID

	// DB 操作放入事务，保证原子性：totalCount 在事务内查询消除 TOCTOU，
	// 删除 + 提升 + 更新 ai_model_id 在同一事务中执行，任一失败整体回滚。
	tx := model.DB(r.Context()).Begin()
	if tx.Error != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(tx.Error, i18n.MsgModelBeginTxFailed))
		return
	}
	txCommitted := false
	defer func() {
		if !txCommitted {
			tx.Rollback()
		}
	}()

	// 在事务内统计总绑定数（避免事务外查询与后续操作间的竞态）
	if err := tx.Model(&model.InstanceModel{}).
		Where("instance_id = ?", instance.ID).
		Count(&totalCount).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgModelCountBindingFailed))
		return
	}

	// 物理删除目标记录，避免软删除记录占用唯一索引导致删除后重新添加同一模型时冲突
	if err := tx.Unscoped().Delete(&targetModel).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgModelDeleteBindingFailed))
		return
	}

	if wasPrimary && totalCount > 1 {
		// 场景 B：自动提升 sort_order 最小（最早添加）的 fallback 为 primary
		var nextPrimary model.InstanceModel
		if err := tx.Where("instance_id = ?", instance.ID).
			Order("sort_order ASC").
			First(&nextPrimary).Error; err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgModelQueryCandidatePrimaryFailed))
			return
		}
		if err := tx.Model(&nextPrimary).Update("role", model.ModelRolePrimary).Error; err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgModelPromoteNextPrimaryFailed))
			return
		}
		promotedModel = &nextPrimary

		// 同步更新 instances.ai_model_id
		if err := tx.Model(instance).Update("ai_model_id", promotedModel.AIModelID).Error; err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgModelUpdatePrimaryIDFailed))
			return
		}
	} else if wasPrimary && totalCount <= 1 {
		// 场景 C：删除最后一个模型，ai_model_id 置 0
		if err := tx.Model(instance).Update("ai_model_id", 0).Error; err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgModelClearPrimaryIDFailed))
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgModelCommitTxFailed))
		return
	}
	txCommitted = true

	// deleteModel 为低频操作：DB 提交后立即下发 TAT 同步 openclaw.json，使其反映删除
	// 后的真实绑定状态。若 TAT 失败，则把数据库回滚到删除前的状态（补偿事务），保证
	// DB 与 CVM 最终一致，避免"DB 已删除但 CVM 未生效"的不一致。
	if err := syncInstanceModelsToCVM(r.Context(), instance, deletedProviderKey); err != nil {
		// 【回滚】TAT 失败 → 把 DB 恢复到删除前：重建被删记录、撤销主模型提升、恢复 ai_model_id。
		// 注：CVM 侧是否部分生效未知，但 DB 先恢复成最终一致状态，下一次操作会重新下发。
		//
		// 必须使用 detached context：syncInstanceModelsToCVM 可能耗时较久（TAT 下发），
		// 期间客户端若刷新/关闭页面会取消 r.Context()，导致补偿事务的 Begin/Create/Commit
		// 全部因 "context canceled" 失败，回滚被静默跳过而 DB 仍停留在已删除状态。
		// 补偿事务的生命周期不应依赖客户端连接，故脱离请求 context 独立执行。
		rbCtx := hcommon.DetachContext(r.Context())
		rbDone := false
		rbTx := model.DB(rbCtx).Begin()
		if rbTx.Error != nil {
			slog.Error("[DelModel] 回滚开启事务失败", "error", rbTx.Error, "instance_id", instance.InstanceId)
		}
		if rbTx.Error == nil {
			rbOK := true
			// 重建被物理删除的绑定记录（保留原 ID/sort_order/role 等字段）
			restore := targetModel
			if err2 := rbTx.Create(&restore).Error; err2 != nil {
				slog.Error("[DelModel] 回滚重建被删模型失败", "error", err2)
				rbTx.Rollback()
				rbOK = false
			}
			// 撤销自动提升：把被提升的新主模型恢复为 fallback
			if rbOK && promotedModel != nil {
				if err2 := rbTx.Model(&model.InstanceModel{}).
					Where("id = ?", promotedModel.ID).
					Update("role", model.ModelRoleFallback).Error; err2 != nil {
					slog.Error("[DelModel] 回滚撤销主模型提升失败", "error", err2)
					rbTx.Rollback()
					rbOK = false
				}
			}
			// 恢复 instances.ai_model_id
			if rbOK {
				if err2 := rbTx.Model(instance).Update("ai_model_id", originalAIModelID).Error; err2 != nil {
					slog.Error("[DelModel] 回滚 ai_model_id 失败", "error", err2)
					rbTx.Rollback()
					rbOK = false
				}
			}
			if rbOK {
				if cErr := rbTx.Commit().Error; cErr != nil {
					slog.Error("[DelModel] TAT 失败后回滚事务提交失败",
						"error", cErr, "instance_id", instance.InstanceId)
					rbTx.Rollback()
				} else {
					rbDone = true
				}
			}
		}
		richErr := hcommon.I18nRichError(err, i18n.MsgModelTATExecuteFailed)
		if rbDone {
			slog.Error("[DelModel] CVM 同步失败，已回滚 DB", "error", err, "instance_id", instance.InstanceId)
		} else {
			// 回滚未成功（如开启事务/写入失败），DB 仍处于已删除状态，需告警人工或等待 reconcile 兜底
			slog.Error("[DelModel] CVM 同步失败，且 DB 回滚未成功，DB 与 CVM 可能不一致",
				"error", err, "instance_id", instance.InstanceId)
		}
		writeError(w, r, http.StatusInternalServerError, richErr)
		notifyCtx := hcommon.DetachContext(r.Context())
		go createErrorNotification(notifyUID, instance.ID, instance.Name,
			model.NotifyTypeModelConfigFailed, i18n.T(notifyCtx, i18n.MsgModelDeleteSyncFailedTitle),
			richErr, notifyCtx)
		return
	}

	resp := map[string]interface{}{
		"ok":          true,
		"was_primary": wasPrimary,
	}

	if !wasPrimary && totalCount > 1 {
		// 场景 A: 删除备选模型 — 返回当前 primary
		var currentPrimary model.InstanceModel
		if model.DB(r.Context()).Where("instance_id = ? AND role = ?", instance.ID, model.ModelRolePrimary).First(&currentPrimary).Error == nil {
			resp["current_primary"] = map[string]interface{}{
				"binding_id":        resolveBindingRef(r.Context(), currentPrimary),
				"instance_model_id": currentPrimary.ID,
				"provider":          getModelProviderName(r.Context(), currentPrimary),
				"model_id":          getModelModelID(r.Context(), currentPrimary),
				"model_name":        getModelModelName(r.Context(), currentPrimary),
				"role":              model.ModelRolePrimary,
			}
		}
	}

	if wasPrimary {
		if promotedModel != nil {
			// 场景 B: 删除主模型 — 返回被提升的新主模型
			resp["promoted_model"] = map[string]interface{}{
				"binding_id":        resolveBindingRef(r.Context(), *promotedModel),
				"instance_model_id": promotedModel.ID,
				"provider":          getModelProviderName(r.Context(), *promotedModel),
				"model_id":          getModelModelID(r.Context(), *promotedModel),
				"model_name":        getModelModelName(r.Context(), *promotedModel),
				"role":              model.ModelRolePrimary,
			}
		} else if totalCount <= 1 {
			// 场景 C: 删除最后一个模型
			resp["promoted_model"] = nil
		}
	}

	jsonOK(w, resp)
}

// HandleInstanceModels 返回实例的所有模型绑定列表。
func HandleInstanceModels(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	writeInstanceModelsResponse(r.Context(), w, instance)
}

// instanceModelListItem 表示模型绑定列表中的单条返回结构。
type instanceModelListItem struct {
	InstanceModelID   uint              `json:"instance_model_id"`
	BindingID         string            `json:"binding_id"`
	AIModelID         uint              `json:"ai_model_id"`
	Role              string            `json:"role"`
	Provider          string            `json:"provider"`
	ModelID           string            `json:"model_id"`
	ModelName         string            `json:"model_name"`
	ModelType         string            `json:"model_type"`
	ContextLen        int               `json:"context_len"`
	MaxTokens         int               `json:"max_tokens"`
	CustomHTTPHeaders map[string]string `json:"custom_http_headers"`
	IsUserCustomModel bool              `json:"is_custom"`
}

// writeInstanceModelsResponse 查询给定实例下的全部模型绑定并以 JSON 形式写出。
//
// 用户侧 HandleInstanceModels 与管控端 HandleAdminInstanceModels 仅在鉴权与
// 实例获取方式上不同，绑定查询与渲染逻辑完全一致，因此抽取到本函数复用：
//   - 查询 instance_models 表中归属该实例的全部记录，按 sort_order DESC 排序；
//   - 批量查询关联的管控端 ai_models（避免 N+1）；
//   - 区分管控端模型（AIModelID > 0，包括 Provider 为“自定义模型”的记录）与用户侧自定义模型（AIModelID == 0）；
//   - 写出 {ok, models} 形式的 JSON。
func writeInstanceModelsResponse(ctx context.Context, w http.ResponseWriter, instance *model.Instance) {
	var bindings []model.InstanceModel
	model.DB(ctx).Where("instance_id = ?", instance.ID).Order("sort_order DESC").Find(&bindings)

	// 批量查询所有内置模型信息，避免循环内 N+1 查询
	var builtinModelIDs []uint
	for _, m := range bindings {
		if m.AIModelID > 0 {
			builtinModelIDs = append(builtinModelIDs, m.AIModelID)
		}
	}
	aimMap := make(map[uint]model.AIModel)
	if len(builtinModelIDs) > 0 {
		var aims []model.AIModel
		model.DB(ctx).Where("id IN ?", builtinModelIDs).Find(&aims)
		for _, a := range aims {
			aimMap[a.ID] = a
		}
	}

	items := make([]instanceModelListItem, 0, len(bindings))
	for _, m := range bindings {
		items = append(items, buildInstanceModelListItem(m, aimMap))
	}

	jsonOK(w, map[string]interface{}{
		"ok":     true,
		"models": items,
	})
}

// buildInstanceModelListItem 将一条 InstanceModel 渲染为列表项。
// 拆分独立函数便于覆盖率统计与单元测试。
func buildInstanceModelListItem(m model.InstanceModel, aimMap map[uint]model.AIModel) instanceModelListItem {
	isUserCustomModel := m.AIModelID == 0
	var provider, modelID, modelName, modelType string
	var contextLen, maxTokens int
	var customHTTPHeaders map[string]string
	var bindingID string

	if !isUserCustomModel {
		aim := aimMap[m.AIModelID]
		provider = aim.Provider
		modelID = aim.ModelID
		modelName = aim.ModelName
		modelType = aim.ModelType
		contextLen = aim.ContextLen
		maxTokens = aim.MaxTokens
		customHTTPHeaders = aim.GetCustomHTTPHeaders()
		if modelID != "" {
			bindingID = fmt.Sprintf("%s/%s", setModelProviderKey(aim, false), model.SlugifyModelID(modelID))
		} else {
			bindingID = fmt.Sprintf("%s-%d/unknown", providerKeyPrefix(provider, false), m.AIModelID)
		}
	} else {
		var cfg customModelConfig
		unmarshalErr := json.Unmarshal([]byte(m.CustomModelConfig), &cfg)
		if unmarshalErr != nil {
			slog.Error("[buildInstanceModelListItem] 解析 CustomModelConfig 失败", "error", unmarshalErr, "instance_model_id", m.ID)
			provider = "custom"
			modelID = "unknown"
			modelName = i18n.T(context.Background(), i18n.MsgModelUnknownModel)
			modelType = "unknown"
			contextLen = 0
			maxTokens = 0
		} else {
			provider = cfg.Provider
			modelID = cfg.ModelID
			modelName = cfg.ModelName
			modelType = cfg.ModelType
			contextLen = cfg.ContextLen
			maxTokens = cfg.MaxTokens
			customHTTPHeaders = cfg.CustomHTTPHeaders
		}
		mid := m.CustomModelID
		if mid == "" && unmarshalErr == nil {
			mid = cfg.ModelID
		}
		if mid == "" {
			mid = "unknown"
		}
		// 【方案 C】用户侧自定义模型 ref 后段保留原始 ModelID（不做 slug 化），
		// 与 resolveBindingRef 保持完全一致，确保前端列表展示的 binding_id
		// 与 OpenClaw 实例上 openclaw.json 中的 fallbacks ref 完全相等，
		// 上游 API 才能正确识别（如 DeepSeek-V3.1 不会被误转为 deepseek-v3.1）。
		bindingID = fmt.Sprintf("custom-%s/%s", model.SlugifyModelID(mid), mid)
	}

	return instanceModelListItem{
		InstanceModelID:   m.ID,
		BindingID:         bindingID,
		AIModelID:         m.AIModelID,
		Role:              m.Role,
		Provider:          provider,
		ModelID:           modelID,
		ModelName:         modelName,
		ModelType:         modelType,
		ContextLen:        contextLen,
		MaxTokens:         maxTokens,
		CustomHTTPHeaders: customHTTPHeaders,
		IsUserCustomModel: isUserCustomModel,
	}
}

// HandleVersionInfo 查询实例 OpenClaw 版本信息（复用 detect_openclaw_install.sh）
func HandleVersionInfo(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	output, err := RunScript(r.Context(), instance.InstanceId, "detect_openclaw_install.sh", 30, instance.RuntimeUser, nil, nil)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgModelTATExecuteFailed))
		return
	}

	// 解析脚本输出 JSON
	var detectResult struct {
		RuntimeUser     string `json:"runtime_user"`
		OpenclawVersion string `json:"openclaw_version"`
	}
	if json.Unmarshal([]byte(output), &detectResult) != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgModelVersionParseFailed))
		return
	}

	jsonOK(w, map[string]interface{}{
		"ok":           true,
		"instance_id":  instance.InstanceId,
		"version":      detectResult.OpenclawVersion,
		"runtime_user": detectResult.RuntimeUser,
	})
}

// ========== 辅助函数 ==========

// injectModelScriptRunner 是 injectModelConfigToCVM 内部 TAT 调用的测试 hook，
// 默认绑定真实 RunScript。生产路径上不直接替换；测试路径通过注入 stub 验证
// proxyModel 改写后的参数传递与成功/失败语义。
//
// 与 script_registry.go 的 agentScriptRunner 同构（后者用于 RunAgentScript）。
var injectModelScriptRunner = RunScript

// syncScriptRunner 便于单测 mock 的函数变量，供 syncInstanceModelsToCVM 下发
// switch_model.sh / remove_model_provider.sh 使用。默认绑定真实 RunScript。
var syncScriptRunner = RunScript

// injectModelConfigToCVM 向实例下发单个模型的 TAT 配置。
//
// isUserCustomModel 表示模型是否来自用户侧自定义配置：
//   - false（管控端模型）：包括 Provider 为“自定义模型”的管控端记录。URL/APIKey 必须改写为
//     hatchery 代理地址（common.DomainFromCtx(ctx)+"/v1"）和实例 ProxyToken，绝不下发上游明文 Key。
//   - true（用户侧自定义模型，AIModelID == 0）：保留用户填写的真实 URL / APIKey，直连第三方服务。
//
// 注意：来源由调用方显式传入而非通过 aim.Provider 推断，避免上游修改 Provider
// 字段值（如 set-model 内置分支会重写为 "hatchery-{modelid}"）时误判。
func modelForInjection(ctx context.Context, instance *model.Instance, aim model.AIModel, isUserCustomModel bool) (model.AIModel, error) {
	if isUserCustomModel {
		return aim, nil
	}
	if hcommon.DomainFromCtx(ctx) == "" {
		return model.AIModel{}, hcommon.I18nError(i18n.MsgModelDomainNotConfiguredAlt)
	}
	if instance.ProxyToken != nil {
		aim.APIKey = *instance.ProxyToken
	}
	aim.URL = hcommon.DomainFromCtx(ctx) + "/v1"
	aim.ModelType = "openai-completions"
	return aim, nil
}

func injectModelConfigToCVM(ctx context.Context, instance *model.Instance, aim *model.AIModel, isUserCustomModel bool) error {
	injectModel, err := modelForInjection(ctx, instance, *aim, isUserCustomModel)
	if err != nil {
		return err
	}
	params, err := buildSetModelParams(ctx, injectModel, instance.ID, isUserCustomModel)
	if err != nil {
		return err
	}
	scriptName, rerr := ResolveScript(ctx, "set_model", instance.AgentType)
	if rerr != nil {
		return hcommon.I18nError(i18n.MsgScriptResolveFailedWrap, "set_model", instance.AgentType).
			WithCause(ErrScriptResolveFailed).WithCause(rerr)
	}
	if _, err := injectModelScriptRunner(ctx, instance.InstanceId, scriptName, 60, instance.RuntimeUser, nil, params); err != nil {
		return err
	}
	return nil
}

func injectModelConfigsToCVM(ctx context.Context, instance *model.Instance, bindings []resolvedSetModelBinding) error {
	injectBindings := make([]resolvedSetModelBinding, 0, len(bindings))
	for _, binding := range bindings {
		injectModel, err := modelForInjection(ctx, instance, binding.InjectModel, binding.IsUserCustomModel)
		if err != nil {
			return err
		}
		binding.InjectModel = injectModel
		injectBindings = append(injectBindings, binding)
	}
	params, err := buildBatchSetModelParams(ctx, injectBindings, instance.ID)
	if err != nil {
		return err
	}
	scriptName, rerr := ResolveScript(ctx, "set_model", instance.AgentType)
	if rerr != nil {
		return hcommon.I18nError(i18n.MsgScriptResolveFailedWrap, "set_model", instance.AgentType).
			WithCause(ErrScriptResolveFailed).WithCause(rerr)
	}
	if _, err := injectModelScriptRunner(ctx, instance.InstanceId, scriptName, 60, instance.RuntimeUser, nil, params); err != nil {
		return err
	}
	return nil
}

// parseCustomModelFromForm 从请求表单解析自定义模型配置。
func parseCustomModelFromForm(r *http.Request) (*customModelConfig, error) {
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
	modelUrl := r.FormValue("url")
	modelType := r.FormValue("model_type")
	contextLenStr := r.FormValue("context_len")
	inputTypes, rerr := hcommon.ValidateInputTypes(r.Form["input_types"])
	if rerr != nil {
		return nil, rerr
	}

	if modelID == "" || apiKey == "" || modelUrl == "" || modelType == "" {
		return nil, hcommon.I18nError(i18n.MsgModelCustomModelFieldsRequired)
	}
	// 【安全】model_id 会被拼接到 shell TAT 参数中，必须白名单校验防命令注入
	if err := hcommon.ValidateCustomModelID(modelID); err != nil {
		return nil, err
	}
	if err := hcommon.ValidateHTTPURL(modelUrl); err != nil {
		return nil, err
	}
	if err := hcommon.ValidateModelType(modelType); err != nil {
		return nil, err
	}

	contextLen := 128000
	if contextLenStr != "" {
		contextLen, _ = strconv.Atoi(contextLenStr)
	}
	if contextLen <= 0 {
		contextLen = 128000
	}

	maxTokensStr := r.FormValue("max_tokens")
	maxTokens, err := strconv.Atoi(maxTokensStr)
	if maxTokensStr != "" && err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgMaxTokensMustBeInteger)
	}
	if maxTokens < 0 {
		return nil, hcommon.I18nError(i18n.MsgMaxTokensMustBeNonNegative)
	}

	customHTTPHeaders, rerr := hcommon.ValidateAndParseCustomHTTPHeaders(r.FormValue("custom_http_headers"))
	if rerr != nil {
		return nil, rerr
	}

	cfg := &customModelConfig{
		Provider:          provider,
		ModelID:           modelID,
		ModelName:         modelName,
		APIKey:            apiKey,
		URL:               modelUrl,
		ModelType:         modelType,
		InputTypes:        inputTypes,
		ContextLen:        contextLen,
		MaxTokens:         maxTokens,
		CustomHTTPHeaders: customHTTPHeaders,
	}
	return cfg, nil
}

func (c customModelConfig) CustomHTTPHeadersJSON() string {
	j, _ := json.Marshal(c.CustomHTTPHeaders)
	return string(j)
}

// JSONStr 返回自定义模型配置的 JSON 序列化字符串。
func (c customModelConfig) JSONStr() string {
	j, _ := json.Marshal(c)
	return string(j)
}

// InputTypesJSON 返回输入类型的 JSON 字符串。
func (c customModelConfig) InputTypesJSON() string {
	j, _ := json.Marshal(c.InputTypes)
	return string(j)
}

// getFallbackRefsJSON 获取实例所有 fallback 模型的引用标识 JSON 数组。
func getFallbackRefsJSON(ctx context.Context, instance *model.Instance) string {
	var models []model.InstanceModel
	if err := model.DB(ctx).Where("instance_id = ? AND role = ?", instance.ID, model.ModelRoleFallback).
		Order("sort_order ASC").Find(&models).Error; err != nil {
		slog.Error("[getFallbackRefsJSON] 查询 fallback 模型失败，返回空列表", "instance_id", instance.ID, "error", err)
		return "[]"
	}
	var refs []string
	for _, m := range models {
		ref := resolveBindingRef(ctx, m)
		// 跳过异常 ref（ai_models 表记录缺失时 resolveBindingRef 返回 hatchery-N/unknown），
		// 与 buildPrimaryAndFallbacks 保持一致
		if strings.Contains(ref, "/unknown") {
			slog.Warn("[getFallbackRefsJSON] 跳过异常 ref",
				"instance_id", instance.ID, "instance_model_id", m.ID, "ref", ref)
			continue
		}
		refs = append(refs, ref)
	}
	j, _ := json.Marshal(refs)
	return string(j)
}

// filterFallbacks 从 fallbacksJSON 字符串里过滤掉与 primaryRef 相同 providerKey 的项，
// 避免同一模型同时出现在 primary 和 fallbacks 里。
// fallbacksJSON 是 JSON 数组字符串，如 `["hatchery-a/a","custom-b/b"]`。
// 过滤后返回新的 JSON 数组字符串；解析失败时原样返回。
func filterFallbacks(fallbacksJSON, primaryRef string) string {
	var refs []string
	if err := json.Unmarshal([]byte(fallbacksJSON), &refs); err != nil {
		return fallbacksJSON
	}
	// 取 primaryRef 的 providerKey 部分（/ 之前）
	primaryKey := primaryRef
	if i := strings.Index(primaryRef, "/"); i >= 0 {
		primaryKey = primaryRef[:i]
	}
	var filtered []string
	for _, ref := range refs {
		refKey := ref
		if i := strings.Index(ref, "/"); i >= 0 {
			refKey = ref[:i]
		}
		if refKey != primaryKey {
			filtered = append(filtered, ref)
		}
	}
	if len(filtered) == 0 {
		return "[]"
	}
	out, _ := json.Marshal(filtered)
	return string(out)
}

// resolveProviderKey 返回用于 TAT 的 provider key（不包含 /modelId 后缀）。
func resolveProviderKey(ctx context.Context, m model.InstanceModel) string {
	ref := resolveBindingRef(ctx, m)
	for i, c := range ref {
		if c == '/' {
			return ref[:i]
		}
	}
	return ref
}

// ======== 响应辅助函数（从 InstanceModel 提取展示信息） ========

func getModelProviderName(ctx context.Context, m model.InstanceModel) string {
	if m.AIModelID > 0 {
		var aim model.AIModel
		if model.DB(ctx).Select("provider").First(&aim, m.AIModelID).Error == nil {
			return aim.Provider
		}
		return "unknown"
	}
	var cfg customModelConfig
	json.Unmarshal([]byte(m.CustomModelConfig), &cfg)
	return cfg.Provider
}

func getModelModelID(ctx context.Context, m model.InstanceModel) string {
	if m.AIModelID > 0 {
		var aim model.AIModel
		if model.DB(ctx).Select("model_id").First(&aim, m.AIModelID).Error == nil {
			return aim.ModelID
		}
		return "unknown"
	}
	var cfg customModelConfig
	json.Unmarshal([]byte(m.CustomModelConfig), &cfg)
	return cfg.ModelID
}

func getModelModelName(ctx context.Context, m model.InstanceModel) string {
	if m.AIModelID > 0 {
		var aim model.AIModel
		if model.DB(ctx).Select("model_name").First(&aim, m.AIModelID).Error == nil {
			return aim.ModelName
		}
		return ""
	}
	var cfg customModelConfig
	json.Unmarshal([]byte(m.CustomModelConfig), &cfg)
	return cfg.ModelName
}

func getModelModelType(ctx context.Context, m model.InstanceModel) string {
	if m.AIModelID > 0 {
		var aim model.AIModel
		if model.DB(ctx).Select("model_type").First(&aim, m.AIModelID).Error == nil {
			return aim.ModelType
		}
		return ""
	}
	var cfg customModelConfig
	json.Unmarshal([]byte(m.CustomModelConfig), &cfg)
	return cfg.ModelType
}

func getContextLen(ctx context.Context, m model.InstanceModel) int {
	if m.AIModelID > 0 {
		var aim model.AIModel
		if model.DB(ctx).Select("context_len").First(&aim, m.AIModelID).Error == nil {
			return aim.ContextLen
		}
		return 0
	}
	var cfg customModelConfig
	if err := json.Unmarshal([]byte(m.CustomModelConfig), &cfg); err != nil {
		slog.Error("[getContextLen] 解析 CustomModelConfig 失败", "error", err, "instance_model_id", m.ID)
		return 0
	}
	return cfg.ContextLen
}

func buildDemotedResponse(ctx context.Context, demotedModel model.InstanceModel, demotedRef string) interface{} {
	if demotedRef == "" {
		return nil
	}
	return map[string]interface{}{
		"binding_id":        demotedRef,
		"instance_model_id": demotedModel.ID,
		"provider":          getModelProviderName(ctx, demotedModel),
		"model_id":          getModelModelID(ctx, demotedModel),
		"model_name":        getModelModelName(ctx, demotedModel),
		"role":              model.ModelRoleFallback,
	}
}

// syncInstanceModelsToCVM 将实例的 instance_models 状态同步到 CVM 的 openclaw.json。
// 被 HandleDelModel（用户侧）和 HandleDeleteModel（管理侧）复用。
// deletedProviderKey: 被删除模型的 provider key（用于清理 CVM 侧残留配置），可为空。
// 返回 error 表示 TAT 执行失败，调用方应记录日志或通知。
func syncInstanceModelsToCVM(ctx context.Context, instance *model.Instance, deletedProviderKey string) error {

	primary, fallbacks, err := buildPrimaryAndFallbacks(ctx, instance.ID)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgModelQueryListFailed)
	}

	if primary == "" {
		// 无模型了 → 清理 provider（如果知道被删的 provider key）
		if deletedProviderKey != "" {
			if _, err := syncScriptRunner(ctx, instance.InstanceId, "remove_model_provider.sh", 30, instance.RuntimeUser, nil, map[string]string{
				"provider": deletedProviderKey,
			}); err != nil {
				return hcommon.I18nRichError(err, i18n.MsgModelCleanProviderFailed)
			}
		}
		return nil
	}

	// 有模型 → switch_model.sh 重新下发 primary + fallbacks
	imagePrimary, imageFallbacks, err := buildImageModelRefs(ctx, instance.ID, primary)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgModelGenImageModelParamFailed)
	}
	if _, err := syncScriptRunner(ctx, instance.InstanceId, "switch_model.sh", 60, instance.RuntimeUser, nil, map[string]string{
		"primary":           primary,
		"fallbacksb64":      base64.StdEncoding.EncodeToString([]byte(fallbacks)),
		"imageprimary":      imagePrimary,
		"imagefallbacksb64": base64.StdEncoding.EncodeToString([]byte(imageFallbacks)),
	}); err != nil {
		return hcommon.I18nRichError(err, i18n.MsgModelSwitchTATFailed)
	}

	// 清理被删模型的 provider 配置（非关键，失败只记 warn）
	if deletedProviderKey != "" {
		if _, err := syncScriptRunner(ctx, instance.InstanceId, "remove_model_provider.sh", 30, instance.RuntimeUser, nil, map[string]string{
			"provider": deletedProviderKey,
		}); err != nil {
			slog.Warn("[syncInstanceModelsToCVM] 清理 provider 失败（非关键）",
				"error", err, "instance_id", instance.InstanceId, "provider", deletedProviderKey)
		}
	}
	return nil
}

func HandleModelConnectivity(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	handleModelConnectivity(w, r, user)
}
