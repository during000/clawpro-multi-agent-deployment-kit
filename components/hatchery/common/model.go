package common

import (
	"encoding/json"
	"net/url"
	"regexp"
	"strings"

	"hatchery/i18n"
)

// 支持的模型接口类型常量
const (
	ModelTypeOpenAICompletions = "openai-completions"
	ModelTypeAnthropicMessages = "anthropic-messages"
)

// 支持的模型输入类型白名单
var validInputTypes = map[string]struct{}{
	"text":  {},
	"image": {},
}

// 自定义模型相关常量
const (
	CustomModelProvider = "自定义模型"
)

// ValidateModelType 校验 model_type 是否为合法枚举值。
func ValidateModelType(modelType string) error {
	switch modelType {
	case ModelTypeOpenAICompletions, ModelTypeAnthropicMessages:
		return nil
	}
	return I18nError(i18n.MsgModelTypeInvalid)
}

// ValidateInputTypes 校验 input_types 是否全部为合法值，返回去重后的列表。
func ValidateInputTypes(types []string) ([]string, error) {
	if len(types) == 0 {
		return []string{"text"}, nil
	}
	seen := make(map[string]bool)
	var result []string
	for _, t := range types {
		if _, ok := validInputTypes[t]; !ok {
			return nil, I18nError(i18n.MsgModelInputTypeInvalid)
		}
		if !seen[t] {
			seen[t] = true
			result = append(result, t)
		}
	}
	return result, nil
}

// customModelIDPattern 限制自定义模型 ID 仅允许字母、数字、`.`、`_`、`-`、`:`、`/` 字符，长度 1~128。
// 该值会作为 body.model 原样下发给真实 LLM（保真）；同时经 SlugifyModelID 转成 slug
// 后拼接到 shell 脚本 TAT 参数（provider/model/primary/fallbacks）。
//
// 关于 `/` 的安全性说明：
//   - `/` 不是 shell 元字符（不会闭合引号、不会触发命令替换）。
//   - 内部分发路径会先经 SlugifyModelID("/" → "-") 归一化后再拼接 shell 参数。
//   - body.model 作为 JSON 字段，被 json.Marshal 编码，不经 shell 插值。
//
// 因此允许 `/` 无命令注入风险。
var customModelIDPattern = regexp.MustCompile(`^[a-zA-Z0-9._:\-/]{1,128}$`)

// ValidateCustomModelID 校验自定义模型的 model_id 字段。
// 由于该值会作为 shell 参数替换进 TAT 脚本，仅允许安全字符集，拒绝任何 shell 元字符。
func ValidateCustomModelID(modelID string) error {
	if modelID == "" {
		return I18nError(i18n.MsgModelCustomIDEmpty)
	}
	if !customModelIDPattern.MatchString(modelID) {
		return I18nError(i18n.MsgModelCustomIDInvalidChars)
	}
	return nil
}

// ValidateHTTPURL 校验 URL 必须是合法的 http 或 https 地址。
func ValidateHTTPURL(rawURL string) error {
	u, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return I18nError(i18n.MsgModelURLInvalid)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return I18nError(i18n.MsgModelURLSchemeNotHTTP)
	}
	if strings.TrimSpace(u.Host) == "" {
		return I18nError(i18n.MsgModelURLHostEmpty)
	}
	return nil
}

// 自定义 HTTP 头部限制常量
const (
	MaxCustomHTTPHeadersCount = 10       // 最多允许 10 个自定义头部
	MaxCustomHTTPHeadersSize  = 4 * 1024 // JSON 最大 4KB
)

// reservedHTTPHeaderKeys 禁止用户通过 custom_http_headers 覆盖的保留头部键名（小写）。
// 包括安全敏感头（Authorization、Cookie）、协议帧头（Content-Length、Transfer-Encoding）、
// 跳跃级头（Connection、Upgrade）以及代理/隐私相关头。
var reservedHTTPHeaderKeys = map[string]struct{}{
	"authorization":       {}, // 认证凭据，禁止覆盖
	"host":                {}, // 路由关键头
	"cookie":              {}, // 安全敏感
	"set-cookie":          {}, // 安全敏感
	"proxy-authorization": {}, // 代理认证
	"proxy-authenticate":  {}, // 代理认证
	"content-length":      {}, // 协议帧
	"content-type":        {}, // 协议帧
	"transfer-encoding":   {}, // 协议帧
	"connection":          {}, // 跳跃级
	"upgrade":             {}, // 跳跃级
	"origin":              {}, // CORS
	"referer":             {}, // 隐私泄漏
	"user-agent":          {}, // 指纹伪装
}

// ValidateAndParseCustomHTTPHeaders 校验并解析用户自定义 HTTP 头部 JSON 字符串。
// 输入为空时返回 nil, nil；校验失败返回 error。
// RFC 7230 键名仅允许字母、数字、'-'、'_'，值允许任意非换行字符串，且键名不得为空。
// 最多允许 MaxCustomHTTPHeadersCount 个头部，JSON 大小不超过 MaxCustomHTTPHeadersSize 字节。
// 键名不得为保留头部（Authorization、Host 等）。
func ValidateAndParseCustomHTTPHeaders(raw string) (map[string]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if len(raw) > MaxCustomHTTPHeadersSize {
		return nil, I18nError(i18n.MsgModelCustomHeadersSizeExceeded, MaxCustomHTTPHeadersSize)
	}
	var result map[string]string
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, I18nError(i18n.MsgModelCustomHeadersInvalidFormat)
	}
	if len(result) > MaxCustomHTTPHeadersCount {
		return nil, I18nError(i18n.MsgModelCustomHeadersCountExceeded, MaxCustomHTTPHeadersCount, len(result))
	}
	headerKeyPattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	for k, v := range result {
		if k == "" {
			return nil, I18nError(i18n.MsgModelCustomHeadersKeyEmpty)
		}
		if !headerKeyPattern.MatchString(k) {
			return nil, I18nError(i18n.MsgModelCustomHeadersKeyInvalidChars, k)
		}
		if _, reserved := reservedHTTPHeaderKeys[strings.ToLower(k)]; reserved {
			return nil, I18nError(i18n.MsgModelCustomHeadersKeyReserved, k)
		}
		if strings.ContainsAny(v, "\n\r") {
			return nil, I18nError(i18n.MsgModelCustomHeadersValueNewline, k)
		}
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}
