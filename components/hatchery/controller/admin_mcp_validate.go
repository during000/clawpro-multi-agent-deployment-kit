package controller

import (
	"encoding/json"
	"fmt"
	hcommon "hatchery/common"
	"hatchery/i18n"
	"regexp"
	"strings"
)

// mcpServiceIDRegex 服务ID格式校验：仅支持英文字母、数字、中划线、下划线
var mcpServiceIDRegex = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// 已知的 transport type
var knownTransportTypes = map[string]bool{
	"sse":             true,
	"streamable-http": true,
	"stdio":           true,
}

// mcpValidationResult 校验结果
type mcpValidationResult struct {
	Warnings []string `json:"warnings,omitempty"`
}

// validateMCPInput 校验 MCP 创建/版本新增的输入参数
// serviceID: 服务ID（创建时必填，版本新增时可为空由调用方处理）
// transportType: 连接方式
// configJSON: 服务配置 JSON 字符串
// isCreate: 是否为创建操作（创建时需要校验 serviceID）
func validateMCPInput(serviceID, transportType, configJSON string, isCreate bool) (*mcpValidationResult, error) {
	result := &mcpValidationResult{}

	// 1. 校验 service_id（仅创建时）
	if isCreate {
		if serviceID == "" {
			return nil, hcommon.I18nError(i18n.MsgMcpServiceIDRequired)
		}
		if !mcpServiceIDRegex.MatchString(serviceID) {
			return nil, hcommon.I18nError(i18n.MsgMcpServiceIDInvalidChars)
		}
		if len(serviceID) > 48 {
			return nil, hcommon.I18nError(i18n.MsgMcpServiceIDTooLong)
		}
	}

	// 2. 校验 transport_type
	if transportType == "" {
		return nil, hcommon.I18nError(i18n.MsgMcpTransportTypeRequired)
	}

	// 3. 校验 config_json
	if configJSON == "" {
		return nil, hcommon.I18nError(i18n.MsgMcpServiceConfigRequired)
	}
	if len(configJSON) > 16*1024 {
		return nil, hcommon.I18nError(i18n.MsgMcpConfigJsonTooLarge)
	}

	// 解析 JSON
	var configMap map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &configMap); err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgMcpConfigJsonParseError, err)
	}
	if len(configMap) == 0 {
		return nil, hcommon.I18nError(i18n.MsgMcpConfigAtLeastOneServer)
	}

	// 4. 已知类型做强校验
	if knownTransportTypes[transportType] {
		if err := validateConfigJSON(transportType, configMap); err != nil {
			return nil, err
		}
	} else {
		// 未知类型：接受但返回 warning
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("unknown transport type '%s', accepted without schema check", transportType))
	}

	return result, nil
}

// validateConfigJSON 对已知 transport type 做字段级校验
func validateConfigJSON(transportType string, config map[string]interface{}) error {
	// 检查 transportType 字段是否与所选连接方式一致
	if tt, ok := config["transportType"]; ok {
		ttStr, _ := tt.(string)
		if ttStr != "" && ttStr != transportType {
			return hcommon.I18nError(i18n.MsgMcpTransportTypeMismatch, transportType, ttStr)
		}
	}

	switch transportType {
	case "sse", "streamable-http":
		// url 必须以 http 或 https 开头
		urlVal, ok := config["url"]
		if !ok {
			return hcommon.I18nError(i18n.MsgMcpURLRequired)
		}
		urlStr, _ := urlVal.(string)
		if urlStr == "" {
			return hcommon.I18nError(i18n.MsgMcpURLRequired)
		}
		if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
			return hcommon.I18nError(i18n.MsgMcpURLMustStartWithHTTP)
		}
	case "stdio":
		// command 不能为空
		cmdVal, ok := config["command"]
		if !ok {
			return hcommon.I18nError(i18n.MsgMcpCommandRequired)
		}
		cmdStr, _ := cmdVal.(string)
		if cmdStr == "" {
			return hcommon.I18nError(i18n.MsgMcpCommandRequired)
		}
	}

	return nil
}
