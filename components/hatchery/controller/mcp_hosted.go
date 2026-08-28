package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// ========== 占位符检测 ==========

// IsPlaceholder 判断 header value 是否是占位符（被 <> 包裹）
func IsPlaceholder(value string) bool {
	v := strings.TrimSpace(value)
	return len(v) >= 2 && strings.HasPrefix(v, "<") && strings.HasSuffix(v, ">")
}

// ExtractPlaceholders 从 config_json 中提取所有占位符（headers + url query）
// 返回 map[字段名]占位符值（如 "Authorization" → "<your-token>"，"key1" → "<api-key>"）
func ExtractPlaceholders(configJSON string) map[string]string {
	var cfg map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return nil
	}

	hosted := make(map[string]string)

	// 提取 headers 中的占位符
	if headers, ok := cfg["headers"].(map[string]interface{}); ok {
		for fieldName, v := range headers {
			if s, ok := v.(string); ok && IsPlaceholder(s) {
				hosted[fieldName] = s
			}
		}
	}

	// 提取 URL query 中的占位符
	if urlStr, ok := cfg["url"].(string); ok {
		if idx := strings.Index(urlStr, "?"); idx >= 0 {
			queryStr := urlStr[idx+1:]
			parts := strings.Split(queryStr, "&")
			for _, part := range parts {
				if eqIdx := strings.Index(part, "="); eqIdx >= 0 {
					paramName := part[:eqIdx]
					val := part[eqIdx+1:]
					if IsPlaceholder(val) {
						hosted[paramName] = val
					}
				}
			}
		}
	}

	if len(hosted) == 0 {
		return nil
	}
	return hosted
}

// ========== URL 生成 ==========

// SecurityGatewayBaseURL 安全网关的基础地址（与 LLM 代理共用 --domain）。
func SecurityGatewayBaseURL(ctx context.Context) string {
	if d := hcommon.DomainFromCtx(ctx); d != "" {
		return strings.TrimSuffix(d, "/")
	}
	return ""
}

// McpGatewayProxyURL 根据 transportType 和 mcpID 生成安全网关的访问 URL
func McpGatewayProxyURL(ctx context.Context, transportType string, mcpID uint) string {
	base := SecurityGatewayBaseURL(ctx)
	if base == "" {
		slog.Warn("[MCP Credential] 安全网关地址未配置")
		return ""
	}
	switch transportType {
	case "sse":
		return fmt.Sprintf("%s/clawpro/sse/%d", base, mcpID)
	default:
		return fmt.Sprintf("%s/clawpro/mcp/%d", base, mcpID)
	}
}

// ========== 下发 config 构建 ==========

// BuildDeployConfigJSON 为凭据托管的 MCP 构造下发到实例的 config_json：
// 1. URL 替换为安全网关代理地址（/clawpro/mcp/{mcp_id}）
// 2. 托管 header 移除（由安全网关从 mcp_installations.hosted_values 注入）
// 3. 非托管 header 保留
// 4. 注入实例 proxyToken 用于安全网关身份验证
func BuildDeployConfigJSON(ctx context.Context, originalConfigJSON string, transportType string, mcpID uint, inst *model.Instance) string {
	proxyURL := McpGatewayProxyURL(ctx, transportType, mcpID)
	if proxyURL == "" {
		slog.Warn("[MCP Credential] 无法生成代理 URL，原样下发")
		return originalConfigJSON
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal([]byte(originalConfigJSON), &cfg); err != nil {
		return originalConfigJSON
	}

	// 替换 URL
	cfg["url"] = proxyURL

	// 处理 headers：移除托管字段（占位符），保留非托管字段，注入 proxyToken
	headers, _ := cfg["headers"].(map[string]interface{})
	newHeaders := make(map[string]interface{})

	// 保留非托管 header
	for k, v := range headers {
		if s, ok := v.(string); ok && IsPlaceholder(s) {
			continue // 托管字段，不下发
		}
		newHeaders[k] = v
	}

	// 注入 proxyToken（安全网关通过 Authorization: Bearer 做身份验证）
	if inst.ProxyToken != nil && *inst.ProxyToken != "" {
		newHeaders["Authorization"] = "Bearer " + *inst.ProxyToken
	}

	if len(newHeaders) > 0 {
		cfg["headers"] = newHeaders
	} else {
		delete(cfg, "headers")
	}

	result, err := json.Marshal(cfg)
	if err != nil {
		return originalConfigJSON
	}
	return string(result)
}

// ========== 展示 config 构建 ==========

// BuildDisplayConfigJSON 为用户端列表展示构造 config_json：
// 1. URL 反转为真实地址（从 mcp_versions.config_json 获取）
// 2. 托管 header 恢复为占位符显示（仅恢复版本原始 headers 中存在的 key，避免把 URL query 占位符误加为 header）
// 3. 移除 proxyToken（Authorization）
func BuildDisplayConfigJSON(ctx context.Context, configJSON string, server *model.McpServer) string {
	var cfg map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return configJSON
	}

	// URL 反转：从 mcp_versions 获取真实 URL，同时提取版本原始 headers 用于区分占位符来源
	var versionHeaders map[string]interface{}
	if server.LatestVersionID > 0 {
		var version model.McpVersion
		if err := model.DB(ctx).Where("id = ?", server.LatestVersionID).First(&version).Error; err == nil {
			var versionCfg map[string]interface{}
			if err := json.Unmarshal([]byte(version.ConfigJSON), &versionCfg); err == nil {
				if realURL, ok := versionCfg["url"].(string); ok {
					cfg["url"] = realURL
				}
				versionHeaders, _ = versionCfg["headers"].(map[string]interface{})
			}
		}
	}

	// 恢复托管 header 为占位符，移除 Authorization（proxyToken）
	var credentials []model.McpHostedKey
	model.DB(ctx).Where("mcp_id = ?", server.ID).Find(&credentials)

	headers, _ := cfg["headers"].(map[string]interface{})
	if headers == nil {
		headers = make(map[string]interface{})
	}

	// 只恢复版本原始 headers 中存在的托管字段（URL query 占位符已包含在恢复的 URL 中，不应加到 headers）
	for _, cred := range credentials {
		if _, exists := versionHeaders[cred.Key]; exists {
			headers[cred.Key] = cred.Placeholder
		}
	}

	// 移除系统注入的 proxyToken（Authorization 始终是网关注入的身份凭证，非用户配置）
	//delete(headers, "Authorization")
	//delete(headers, "authorization")

	if len(headers) > 0 {
		cfg["headers"] = headers
	} else {
		delete(cfg, "headers")
	}

	result, err := json.Marshal(cfg)
	if err != nil {
		return configJSON
	}
	return string(result)
}

// ========== 管理端辅助 ==========

// SaveHostedKeys 解析 config_json 中的占位符 header，保存到 mcp_hosted_keys 表。
// hostedDefaults: 管理员提供的默认值（可为空 map）
// 返回是否存在托管字段
func SaveHostedKeys(ctx context.Context, mcpID uint, configJSON string, hostedDefaults map[string]string) (bool, error) {
	placeholders := ExtractPlaceholders(configJSON)
	if len(placeholders) == 0 {
		return false, nil
	}

	for headerKey, placeholder := range placeholders {
		defaultVal := ""
		if hostedDefaults != nil {
			defaultVal = hostedDefaults[headerKey]
		}
		cred := model.McpHostedKey{
			MCPID:        mcpID,
			Key:          headerKey,
			Placeholder:  placeholder,
			DefaultValue: defaultVal,
		}
		if err := model.DB(ctx).
			Where("mcp_id = ? AND `key` = ?", mcpID, headerKey).
			Assign(map[string]interface{}{
				"placeholder":   placeholder,
				"default_value": defaultVal,
			}).
			FirstOrCreate(&cred).Error; err != nil {
			return false, hcommon.I18nRichError(err, i18n.MsgMcpSaveHostedKeyFailed, headerKey)
		}
	}

	return true, nil
}

// GetHostedKeys 获取 MCP 的所有托管字段定义
func GetHostedKeys(ctx context.Context, mcpID uint) []model.McpHostedKey {
	var creds []model.McpHostedKey
	model.DB(ctx).Where("mcp_id = ?", mcpID).Find(&creds)
	return creds
}

// ResolveHostedValues 解析实例的最终托管字段值：
// 优先用户提供的值 > 管理员默认值。
// 返回最终值 map 和缺失字段列表
func ResolveHostedValues(credentials []model.McpHostedKey, userValues map[string]string) (resolved map[string]string, missing []string) {
	resolved = make(map[string]string, len(credentials))
	for _, cred := range credentials {
		if v, ok := userValues[cred.Key]; ok && v != "" {
			resolved[cred.Key] = v
			continue
		}
		if cred.DefaultValue != "" {
			resolved[cred.Key] = cred.DefaultValue
			continue
		}
		missing = append(missing, cred.Key)
	}
	return
}

// ExtractTargetURL 从 config_json 中提取真实的 MCP 服务 URL
func ExtractTargetURL(configJSON string) string {
	var cfg map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return ""
	}
	url, _ := cfg["url"].(string)
	return url
}

// removeDefaultedPlaceholders 从 config_json 中移除有默认值的占位符：
// 1. headers 中：直接删除该 key
// 2. url query 中：移除对应的 query 参数
func removeDefaultedPlaceholders(configJSON string, keysToRemove map[string]bool) string {
	var cfg map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return configJSON
	}

	// 处理 headers：按 header name 匹配
	if headers, ok := cfg["headers"].(map[string]interface{}); ok {
		for fieldName, v := range headers {
			if s, ok := v.(string); ok && IsPlaceholder(s) {
				if keysToRemove[fieldName] {
					delete(headers, fieldName)
				}
			}
		}
		if len(headers) == 0 {
			delete(cfg, "headers")
		} else {
			cfg["headers"] = headers
		}
	}

	// 处理 url 中的 query 占位符：按 query 参数名匹配
	if urlStr, ok := cfg["url"].(string); ok {
		if idx := strings.Index(urlStr, "?"); idx >= 0 {
			basePath := urlStr[:idx]
			queryStr := urlStr[idx+1:]

			// 按 & 分割，逐个检查
			parts := strings.Split(queryStr, "&")
			var kept []string
			for _, part := range parts {
				// part 形如 key=<placeholder> 或 key=value
				eqIdx := strings.Index(part, "=")
				if eqIdx < 0 {
					kept = append(kept, part)
					continue
				}
				paramName := part[:eqIdx]
				val := part[eqIdx+1:]
				if IsPlaceholder(val) {
					if keysToRemove[paramName] {
						continue // 移除该 query 参数
					}
				}
				kept = append(kept, part)
			}

			if len(kept) > 0 {
				cfg["url"] = basePath + "?" + strings.Join(kept, "&")
			} else {
				cfg["url"] = basePath
			}
		}
	}

	result, err := json.Marshal(cfg)
	if err != nil {
		return configJSON
	}
	return string(result)
}

// DiffConfigPlaceholders 比对原始 config_json（含占位符）和用户提交的 config_json（已填值），
// 提取占位符位置被替换的值。同时校验：
// 1. 非占位符字段不能被修改
// 2. 有默认值的占位符不允许用户修改（应保持原占位符或不传）
// 3. URL path 不能改，只有 query 中的占位符可以被填值
// 返回 map[字段名]用户填入的值。
func DiffConfigPlaceholders(originalJSON, submittedJSON string) (map[string]string, error) {
	var original, submitted map[string]interface{}
	if err := json.Unmarshal([]byte(originalJSON), &original); err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgMcpConfigParseFailed)
	}
	if err := json.Unmarshal([]byte(submittedJSON), &submitted); err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgMcpConfigParseFailed)
	}

	filledValues := make(map[string]string)

	// 校验非 url/headers 字段不能被修改
	for key := range original {
		if key == "headers" || key == "url" {
			continue
		}
		origVal, _ := json.Marshal(original[key])
		subVal, _ := json.Marshal(submitted[key])
		if string(origVal) != string(subVal) {
			return nil, hcommon.I18nError(i18n.MsgMcpModifyFieldNotAllowed, key)
		}
	}
	for key := range submitted {
		if key == "headers" || key == "url" {
			continue
		}
		if _, exists := original[key]; !exists {
			return nil, hcommon.I18nError(i18n.MsgMcpAddFieldNotAllowed, key)
		}
	}

	// === 比对 URL ===
	origURL, _ := original["url"].(string)
	subURL, _ := submitted["url"].(string)

	// 分离 path 和 query
	origPath, origQuery := splitURL(origURL)
	subPath, subQuery := splitURL(subURL)

	// path 不允许修改
	if origPath != subPath {
		return nil, hcommon.I18nError(i18n.MsgMcpModifyURLPathNotAllowed)
	}

	// 比对 query 参数
	origParams := parseQueryParams(origQuery)
	subParams := parseQueryParams(subQuery)

	for _, op := range origParams {
		if IsPlaceholder(op.value) {
			// 查找用户提交中对应的参数
			subVal := findQueryParam(subParams, op.key)
			if subVal == "" || IsPlaceholder(subVal) {
				continue // 用户没填或保持占位符，后续用默认值
			}
			filledValues[op.key] = subVal // 按 query 参数名存
		} else {
			// 非占位符 query 参数：不允许修改
			subVal := findQueryParam(subParams, op.key)
			if subVal != op.value {
				return nil, hcommon.I18nError(i18n.MsgMcpModifyURLParamNotAllowed, op.key)
			}
		}
	}

	// === 比对 headers ===
	origHeaders, _ := original["headers"].(map[string]interface{})
	subHeaders, _ := submitted["headers"].(map[string]interface{})

	for key, origVal := range origHeaders {
		origStr, ok := origVal.(string)
		if !ok {
			continue
		}
		if IsPlaceholder(origStr) {
			subVal, exists := subHeaders[key]
			if !exists {
				continue // 用户没填，后续用默认值
			}
			subStr, ok := subVal.(string)
			if !ok {
				return nil, hcommon.I18nError(i18n.MsgMcpHeaderValueMustBeString, key)
			}
			if IsPlaceholder(subStr) {
				continue // 用户没改，还是占位符
			}
			filledValues[key] = subStr // 按 header name 存
		} else {
			// 非占位符字段：不允许修改
			subVal, _ := subHeaders[key]
			subStr, _ := subVal.(string)
			if subStr != origStr {
				return nil, hcommon.I18nError(i18n.MsgMcpModifyHeaderNotAllowed, key)
			}
		}
	}

	// 检查用户是否新增了 header 字段
	for key := range subHeaders {
		if _, exists := origHeaders[key]; !exists {
			return nil, hcommon.I18nError(i18n.MsgMcpAddHeaderNotAllowed, key)
		}
	}

	return filledValues, nil
}

// queryParam 表示一个 query 参数
type queryParam struct {
	key   string
	value string
}

// splitURL 分离 URL 的 path 和 query 部分
func splitURL(urlStr string) (path, query string) {
	if idx := strings.Index(urlStr, "?"); idx >= 0 {
		return urlStr[:idx], urlStr[idx+1:]
	}
	return urlStr, ""
}

// parseQueryParams 解析 query string 为参数列表
func parseQueryParams(query string) []queryParam {
	if query == "" {
		return nil
	}
	parts := strings.Split(query, "&")
	params := make([]queryParam, 0, len(parts))
	for _, part := range parts {
		if eqIdx := strings.Index(part, "="); eqIdx >= 0 {
			params = append(params, queryParam{key: part[:eqIdx], value: part[eqIdx+1:]})
		} else {
			params = append(params, queryParam{key: part, value: ""})
		}
	}
	return params
}

// findQueryParam 在参数列表中查找指定 key 的值
func findQueryParam(params []queryParam, key string) string {
	for _, p := range params {
		if p.key == key {
			return p.value
		}
	}
	return ""
}

// buildProbeConfigJSON 为 probe 构造带真实凭据的 config_json：
// 将占位符 header 替换为真实值，保留其他字段不变。
func buildProbeConfigJSON(originalConfigJSON string, resolvedValues map[string]string) string {
	var cfg map[string]interface{}
	if err := json.Unmarshal([]byte(originalConfigJSON), &cfg); err != nil {
		return originalConfigJSON
	}

	// 替换 headers 中的占位符值（resolvedValues key = header name）
	if headers, ok := cfg["headers"].(map[string]interface{}); ok {
		for fieldName, hv := range headers {
			if s, ok := hv.(string); ok && IsPlaceholder(s) {
				if realVal, exists := resolvedValues[fieldName]; exists {
					headers[fieldName] = realVal
				}
			}
		}
	}

	// 替换 URL query 中的占位符值（resolvedValues key = query param name）
	if urlStr, ok := cfg["url"].(string); ok {
		if idx := strings.Index(urlStr, "?"); idx >= 0 {
			basePath := urlStr[:idx]
			queryStr := urlStr[idx+1:]
			parts := strings.Split(queryStr, "&")
			var newParts []string
			for _, part := range parts {
				if eqIdx := strings.Index(part, "="); eqIdx >= 0 {
					paramName := part[:eqIdx]
					val := part[eqIdx+1:]
					if IsPlaceholder(val) {
						if realVal, exists := resolvedValues[paramName]; exists {
							part = paramName + "=" + realVal
						}
					}
				}
				newParts = append(newParts, part)
			}
			cfg["url"] = basePath + "?" + strings.Join(newParts, "&")
		}
	}

	result, err := json.Marshal(cfg)
	if err != nil {
		return originalConfigJSON
	}
	return string(result)
}
