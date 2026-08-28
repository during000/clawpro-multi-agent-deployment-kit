package controller

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"hatchery/common"
	"hatchery/controller/provider"
	"hatchery/i18n"
	"hatchery/model"
)

// usageLogEntry 封装 usage log 条目和对应的请求 ctx，确保消费端能使用正确的租户上下文写 DB。
type usageLogEntry struct {
	ctx   context.Context
	entry model.LLMUsageLog
}

var (
	usageLogCh   chan usageLogEntry
	usageLogDone chan struct{}
)

// InitUsageLogger starts the background goroutine that writes usage logs to the database.
func InitUsageLogger() {
	usageLogCh = make(chan usageLogEntry, 256)
	usageLogDone = make(chan struct{})
	go func() {
		defer close(usageLogDone)
		for msg := range usageLogCh {
			if err := model.DB(msg.ctx).Create(&msg.entry).Error; err != nil {
				slog.Error("[LLM Proxy] Failed to write usage log", "error", err)
			}
			// Also upsert into daily summary for efficient quota queries.
			model.UpsertDailyUsage(msg.ctx, msg.entry.UserID, msg.entry.InstanceID, msg.entry.AIModelID, msg.entry.PromptTokens, msg.entry.CompletionTokens, msg.entry.TotalTokens, msg.entry.GroupID, msg.entry.PromptCacheReadTokens, msg.entry.PromptCacheWriteTokens)
		}
	}()
}

// FlushUsageLogs closes the usage log channel and waits for all pending entries to be written.
func FlushUsageLogs() {
	if usageLogCh != nil {
		close(usageLogCh)
		<-usageLogDone
		usageLogCh = nil
		slog.Info("Usage log channel drained")
	}
}

// llmErrorResponse writes an OpenAI-compatible error JSON response.
func llmErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"message": message,
			"type":    "error",
			"code":    statusCode,
		},
	})
}

// extractBearerToken extracts the token from "Authorization: Bearer xxx".
func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(auth, "Bearer ")
}

// extractRequestedModel 从请求 body 里拿 "model" 字段，空字符串表示客户端未指定。
func extractRequestedModel(reqBody map[string]interface{}) string {
	if v, ok := reqBody["model"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

// extractCustomHeaders 从入站 HTTP 请求中提取需要透传给上游模型的自定义头。
// customHeaders 是数据库配置的允许透传的请求头键值对
func extractCustomHeaders(r *http.Request, customHeaders map[string]string) map[string]string {
	if len(customHeaders) == 0 {
		return nil
	}
	result := make(map[string]string, len(customHeaders))
	for key := range customHeaders {
		if v := r.Header.Get(key); v != "" && v == customHeaders[key] {
			result[http.CanonicalHeaderKey(key)] = v
		} else {
			slog.Warn("[LLM Proxy] 自定义请求头的值与用户指定不匹配，不进行透传", "key", key, "expected", customHeaders[key], "actual", v)
			return nil
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func HandleLLMProxy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		llmErrorResponse(w, http.StatusMethodNotAllowed, i18n.T(r.Context(), i18n.MsgMethodNotAllowed))
		return
	}

	start := time.Now()

	// 1. Extract and validate proxy token
	token := extractBearerToken(r)
	if token == "" {
		llmErrorResponse(w, http.StatusUnauthorized, i18n.T(r.Context(), i18n.MsgMissingAPIKey))
		return
	}

	var instance model.Instance
	if model.DBGlobal(r.Context()).Where("proxy_token = ?", token).First(&instance).Error != nil {
		llmErrorResponse(w, http.StatusUnauthorized, i18n.T(r.Context(), i18n.MsgInvalidAPIKey))
		return
	}

	// 2. Read request body first so we can resolve the model by its name.
	body, err := io.ReadAll(r.Body)
	if err != nil {
		llmErrorResponse(w, http.StatusBadRequest, i18n.T(r.Context(), i18n.MsgReadRequestBody))
		return
	}

	var reqBody map[string]interface{}
	if err := json.Unmarshal(body, &reqBody); err != nil {
		llmErrorResponse(w, http.StatusBadRequest, i18n.T(r.Context(), i18n.MsgInvalidJSON))
		return
	}

	reqModelName := extractRequestedModel(reqBody)

	// 3. Resolve target model: by requested name → fallback to instance primary.
	resolved, err := model.ResolveModelForRequest(r.Context(), &instance, reqModelName)
	if err != nil {
		if errors.Is(err, model.ErrNoResolvableModel) {
			// 区分两种错误：
			//   - 客户端显式指定了 model 但绑定表里没有 → 返回"未绑定"
			//   - 未指定 model 且实例无任何可用配置 → 返回"未配置可用模型"
			if reqModelName != "" {
				llmErrorResponse(w, http.StatusBadRequest, i18n.T(r.Context(), i18n.MsgModelNotFound))
				return
			}
			llmErrorResponse(w, http.StatusBadRequest, i18n.T(r.Context(), i18n.MsgNoActiveModel))
			return
		}
		llmErrorResponse(w, http.StatusInternalServerError, i18n.T(r.Context(), i18n.MsgInternalServerError))
		return
	}

	// 4. Check global token quota rules.
	// Use 403 instead of 429 to prevent downstream agent/embedded from
	// classifying this as "rate_limit" and replacing our localized message
	// with a hardcoded English string.
	siteConfig := model.GetSiteConfig(r.Context())
	quotaPeriod := siteConfig.NormalizedGlobalTokenQuotaPeriod()
	globalQuotaScope := resolveEffectiveGlobalTokenQuotaScope(r.Context(), siteConfig, instance.GroupID)
	if exceeded, _, _ := model.CheckGlobalTokenQuota(r.Context(), globalQuotaScope.UsageGroupID, globalQuotaScope.Rules); exceeded {
		quotaErr := i18n.T(r.Context(), i18n.MsgGlobalQuotaExceeded)
		if quotaPeriod == model.GlobalTokenQuotaPeriodMonth {
			quotaErr = i18n.T(r.Context(), i18n.MsgGlobalMonthlyQuotaExceeded)
		}
		llmErrorResponse(w, http.StatusForbidden, quotaErr)
		go notifyQuotaExceeded(common.DetachContext(r.Context()), instance.UserID, instance.ID, instance.Name, quotaErr)
		return
	}

	// 5. Check model-level daily quota (只对内置模型生效；自定义模型 UsageBucketKey=0 无意义，跳过)
	if !resolved.IsCustom && resolved.QuotaDay >= 0 && resolved.UsageBucketKey > 0 {
		used := model.ModelDailyTokenUsage(r.Context(), resolved.UsageBucketKey)
		if used >= int64(resolved.QuotaDay) {
			llmErrorResponse(w, http.StatusForbidden, i18n.T(r.Context(), i18n.MsgModelQuotaExceeded))
			go notifyQuotaExceeded(common.DetachContext(r.Context()), instance.UserID, instance.ID, instance.Name, i18n.T(r.Context(), i18n.MsgModelQuotaExceeded))
			return
		}
	}

	// 6. Check per-user token quota (rules-based, reads llm_usage_logs)
	// 有分组时：运行时从组策略解析 rules（管理员改策略即时生效，不需要重新烙印）
	// 无分组时：读用户字段（创建时烙印的 site config 默认值）
	var user model.User
	if err := model.DB(r.Context()).First(&user, instance.UserID).Error; err != nil {
		llmErrorResponse(w, http.StatusInternalServerError, i18n.T(r.Context(), i18n.MsgLoadUserFailed))
		return
	}
	userQuotaRules := resolveEffectiveUserTokenQuotaRules(r.Context(), user, instance.GroupID)
	if len(userQuotaRules) > 0 {
		if exceeded, _, _ := model.CheckUserTokenQuota(r.Context(), user.ID, instance.GroupID, userQuotaRules); exceeded {
			llmErrorResponse(w, http.StatusForbidden, i18n.T(r.Context(), i18n.MsgUserQuotaExceeded))
			go notifyQuotaExceeded(common.DetachContext(r.Context()), user.ID, instance.ID, instance.Name, i18n.T(r.Context(), i18n.MsgUserQuotaExceeded))
			return
		}
	}

	// 标记 AI 活跃状态（用于云端浏览器接管判断）
	MarkAIActive(instance.ID)
	// hasToolCalls 用于判断 Agent Loop 是否还在继续（LLM 返回 tool_calls 表示后续还有工具执行 + LLM 请求）
	hasToolCalls := false
	defer func() {
		MarkAIInactiveWithContext(instance.ID, hasToolCalls)
	}()

	// Enforce max_tokens if set on the instance.
	enforceMaxTokens(reqBody, instance.MaxTokens)

	// 请求透传给上游时，强制使用 resolved.ModelID（而非客户端原始请求值），
	// 避免大小写偏差、客户端误填 display name 等情况导致上游报 model not found。
	reqBody["model"] = resolved.ModelID

	isStreaming := false
	if s, ok := reqBody["stream"]; ok {
		if sb, ok := s.(bool); ok && sb {
			isStreaming = true
		}
	}

	modifiedBody, err := json.Marshal(reqBody)
	if err != nil {
		llmErrorResponse(w, http.StatusInternalServerError, i18n.T(r.Context(), i18n.MsgInvalidJSON))
		return
	}

	// 7. Route to the correct provider implementation
	p := provider.GetProvider(resolved.ModelType)
	ctx := r.Context()

	if isStreaming {
		flusher, ok := w.(http.Flusher)
		if !ok {
			llmErrorResponse(w, http.StatusInternalServerError, i18n.T(r.Context(), i18n.MsgStreamNotSupported))
			return
		}

		result, statusCode, err := p.ChatCompletionStream(ctx, resolved.APIKey, resolved.URL, resolved.ModelID, modifiedBody, w, flusher, extractCustomHeaders(r, resolved.CustomHTTPHeaders))
		if err != nil {
			slog.Error("[LLM Proxy] Stream error", "error", err)
			return
		}
		if result != nil {
			hasToolCalls = result.HasToolCalls
			logUsage(ctx, result.Usage, instance, resolved, statusCode, start)
		}
	} else {
		result, statusCode, err := p.ChatCompletion(ctx, resolved.APIKey, resolved.URL, resolved.ModelID, modifiedBody, extractCustomHeaders(r, resolved.CustomHTTPHeaders))
		if err != nil {
			llmErrorResponse(w, http.StatusBadGateway, i18n.T(r.Context(), i18n.MsgLLMBackendConnect)+": "+err.Error())
			return
		}
		respBody := result.Body

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		w.Write(respBody)

		logUsage(ctx, result.Usage, instance, resolved, statusCode, start)
		var respResult map[string]interface{}
		if json.Unmarshal(respBody, &respResult) == nil {
			hasToolCalls = detectToolCallsInResponse(respResult)
		}
	}
}

func logUsage(ctx context.Context, usage *provider.Usage, instance model.Instance, resolved *model.ResolvedModel, statusCode int, start time.Time) {
	entry := model.LLMUsageLog{
		InstanceID: instance.ID,
		UserID:     instance.UserID,
		GroupID:    instance.GroupID,
		AIModelID:  resolved.UsageBucketKey,
		Model:      resolved.ModelID,
		Provider:   resolved.Provider,
		StatusCode: statusCode,
		Latency:    int(time.Since(start).Milliseconds()),
		CreatedAt:  time.Now().UTC(),
	}
	if usage != nil {
		entry.PromptTokens = usage.PromptTokens
		entry.CompletionTokens = usage.CompletionTokens
		entry.TotalTokens = usage.TotalTokens
		entry.PromptCacheReadTokens = usage.PromptCacheReadTokens
		entry.PromptCacheWriteTokens = usage.PromptCacheWriteTokens
	}

	if usageLogCh != nil {
		select {
		case usageLogCh <- usageLogEntry{ctx: common.DetachContext(ctx), entry: entry}:
		default:
			slog.Warn("[LLM Proxy] Usage log channel full, dropping entry")
		}
	}
}

// enforceMaxTokens 对请求体中的 max_tokens / max_completion_tokens 施加实例配额限制。
// 客户端可能发其中一个或两个，都需要限制。都没发则注入 max_tokens。
// provider 层会按模型统一转化参数名。
func enforceMaxTokens(reqBody map[string]interface{}, limit int) {
	if limit <= 0 {
		return
	}
	cap := func(key string) {
		if v, ok := reqBody[key]; ok {
			if vf, ok := v.(float64); ok && int(vf) > limit {
				reqBody[key] = float64(limit)
			}
		}
	}
	cap("max_tokens")
	cap("max_completion_tokens")
	if _, ok := reqBody["max_tokens"]; !ok {
		if _, ok := reqBody["max_completion_tokens"]; !ok {
			reqBody["max_tokens"] = float64(limit)
		}
	}
}

// detectToolCallsInResponse 检测非流式 LLM 响应中是否包含 tool_calls。
// 用于判断 Agent Loop 是否还在继续（有 tool_calls = 后续还有工具执行 + LLM 请求）。
func detectToolCallsInResponse(data map[string]interface{}) bool {
	choices, ok := data["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return false
	}
	for _, choice := range choices {
		choiceMap, ok := choice.(map[string]interface{})
		if !ok {
			continue
		}
		// 检查 finish_reason == "tool_calls"
		if fr, ok := choiceMap["finish_reason"].(string); ok && fr == "tool_calls" {
			return true
		}
		// 检查 message.tool_calls 字段
		if msg, ok := choiceMap["message"].(map[string]interface{}); ok {
			if tc, ok := msg["tool_calls"].([]interface{}); ok && len(tc) > 0 {
				return true
			}
		}
	}
	return false
}

func HandleLLMModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		llmErrorResponse(w, http.StatusMethodNotAllowed, i18n.T(r.Context(), i18n.MsgMethodNotAllowed))
		return
	}

	token := extractBearerToken(r)
	if token == "" {
		llmErrorResponse(w, http.StatusUnauthorized, i18n.T(r.Context(), i18n.MsgMissingAPIKey))
		return
	}

	var instance model.Instance
	if model.DBGlobal(r.Context()).Where("proxy_token = ?", token).First(&instance).Error != nil {
		llmErrorResponse(w, http.StatusUnauthorized, i18n.T(r.Context(), i18n.MsgInvalidAPIKey))
		return
	}

	// 列出实例全部绑定模型（primary + fallback；去重；自定义模型也包含）
	list := model.ListInstanceModels(r.Context(), instance.ID)
	modelList := make([]map[string]interface{}, 0, len(list))
	seen := make(map[string]struct{}, len(list))
	for _, m := range list {
		key := strings.ToLower(m.ModelID)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		modelList = append(modelList, map[string]interface{}{
			"id":       strings.ToLower(m.ModelID),
			"object":   "model",
			"created":  time.Now().Unix(),
			"owned_by": m.Provider,
		})
	}

	// Fallback: 未迁移到 instance_models 的老实例，从 instance 主字段取 primary
	if len(modelList) == 0 {
		if resolved, err := model.ResolveModelForRequest(r.Context(), &instance, ""); err == nil {
			modelList = append(modelList, map[string]interface{}{
				"id":       resolved.ModelID,
				"object":   "model",
				"created":  time.Now().Unix(),
				"owned_by": resolved.Provider,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"object": "list",
		"data":   modelList,
	})
}
