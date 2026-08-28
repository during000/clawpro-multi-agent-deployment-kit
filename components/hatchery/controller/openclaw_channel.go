package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hatchery/common"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"hatchery/controller/usergroup"
	"hatchery/i18n"
	"hatchery/model"

	qrterminal "github.com/mdp/qrterminal/v3"
)

// ========== v7 新增：HandleAutoChannel 的 channel → feature / timeout 映射 ==========
//
// 真实事实：channel 值与脚本文件名**不一一对应**：
//   - qqbot           → qq_bot_creator.sh（无 bot 后缀简化）
//   - openclaw-weixin → weixin_bot_creator.sh（去掉 openclaw- 前缀）
//   - weixin          → weixin_bot_creator.sh（Hermes/ACE 别名）
//
// Hermes/ACE 白名单里用 "weixin"（不带 openclaw- 前缀），统一映射到同一 feature
// weixin_bot_creator，通过 ResolveScript(feature, agentType) 分派到对应脚本。
// 注意：WhatsApp 存在两种独立且互不冲突的接入方式，通过不同 channel_id 区分：
//   - "whatsapp"：内置扫码登录通道（baileys 扫码，无需手机号），走 whatsapp_bot_creator.sh
//   - "openclaw_whatsapp"（或管理员在自定义通道中配置的其他 channel_id）：
//     自定义通道框架下的配对码模式（需要手机号），由 CustomConfig.server 驱动，
//     走 set_channel_whatsapp.sh / del_channel_whatsapp.sh
//
// 两者是两套独立能力，channel_id 不同，互不覆盖，不做别名映射。
var autoChannelFeature = map[string]string{
	"qqbot":           "qq_bot_creator",
	"feishu":          "feishu_bot_creator",
	"lark":            "feishu_bot_creator", // 与飞书保持一致
	"openclaw-weixin": "weixin_bot_creator",
	"weixin":          "weixin_bot_creator", // Hermes/ACE 使用的 channel 名别名
	"whatsapp":        "whatsapp_bot_creator",
	// openclaw_whatsapp（及其他自定义通道）不在此表中，由自定义通道框架 CustomConfig.server.autoFeature 驱动
}

var autoChannelTimeout = map[string]uint64{
	"qqbot":           300,
	"feishu":          600, // 飞书需要安装 playwright + chromium，首次耗时较长
	"lark":            600, // 与飞书保持一致
	"openclaw-weixin": 600, // 微信扫码最长等待 10 分钟
	"weixin":          600,
	"whatsapp":        120,
	// openclaw_whatsapp（及其他自定义通道）不在此表中，由自定义通道框架 CustomConfig.server.autoTimeout 驱动
}

// egressDiagnosticChannelWhitelist 标识哪些通道在 Agent 实例出站被拒时会失败。
// 经测试只有个人微信与飞书需要 Agent 主动出站访问对方服务器，其他通道不受影响。
// 仅这些 channel 失败时才触发安全组 egress 诊断，避免误报。
var egressDiagnosticChannelWhitelist = map[string]bool{
	"openclaw-weixin":   true,
	"weixin":            true, // Hermes/ACE 别名
	"feishu":            true,
	"lark":              true, // 与飞书保持一致
	"openclaw_whatsapp": true, // 自定义通道框架下的 WhatsApp 配对码流程依赖 Agent 主动出站连接 WhatsApp 服务器
	// TODO: 最终将改为从 CustomConfig.server.egressRequired 驱动，消除硬编码白名单
}

// whatsappPhoneRegexp 是自定义通道框架下 WhatsApp 配对码手机号格式的内置默认正则。
// 当 ServerConfig.PhonePattern 未配置时使用此正则进行校验。
// 格式要求：1-9 开头，后跟 6-14 位数字，不含 + 号（如 85266803489）。
var whatsappPhoneRegexp = regexp.MustCompile(`^[1-9]\d{6,14}$`)
var channelScriptRunner = RunScript
var delChannelScriptRunner = RunScript

type manualChannelPreset struct {
	Channel string
	Config  map[string]string
}

type manualChannelApplyResult struct {
	Output        string
	ProxyRouteID  string
	ProxyEndpoint string
}

type manualChannelValidationError struct {
	status int
	err    *common.RichError
}

func (e *manualChannelValidationError) Error() string { return e.err.Error() }
func (e *manualChannelValidationError) Unwrap() error { return e.err }

func newManualChannelValidationError(status int, err *common.RichError) error {
	return &manualChannelValidationError{status: status, err: err}
}

func manualChannelValidationStatus(err error) (int, bool) {
	var validationErr *manualChannelValidationError
	if errors.As(err, &validationErr) {
		return validationErr.status, true
	}
	return http.StatusBadRequest, false
}

func manualChannelValidationHTTPStatus(err error) int {
	status, _ := manualChannelValidationStatus(err)
	return status
}

// validateManualChannelConfig applies the same availability, scope, agent-type,
// and group-visibility rules as the manual set-channel endpoint. strictParams
// additionally requires every configured channel parameter and is used before
// an admin create request can allocate a CVM.
func validateManualChannelConfig(ctx context.Context, instance *model.Instance, preset manualChannelPreset, strictParams bool) (*model.AIChannel, error) {
	if instance == nil || !model.AgentTypeSupportsChannel(ctx, instance.AgentType) {
		typeName := ""
		if instance != nil {
			typeName = model.GetAgentTypeDisplayName(ctx, instance.AgentType)
		}
		return nil, newManualChannelValidationError(http.StatusForbidden,
			common.I18nError(i18n.MsgChannelNotSupportedWithDetail, typeName))
	}
	channelID := strings.TrimSpace(preset.Channel)
	if channelID == "" {
		return nil, newManualChannelValidationError(http.StatusBadRequest,
			common.I18nError(i18n.MsgBadRequestParamRequired, "channel"))
	}

	var channel model.AIChannel
	channelErr := model.DB(ctx).Where("channel_id = ?", channelID).First(&channel).Error
	if channelErr != nil {
		if strictParams {
			return nil, newManualChannelValidationError(http.StatusBadRequest,
				common.I18nError(i18n.MsgChannelNotFound, channelID))
		}
		// Preserve the existing manual set-channel contract: predefined
		// channels are validated by site/agent allowlists even when a legacy
		// database has not seeded ai_channels yet.
		channel = model.AIChannel{ChannelID: channelID, VisibilityType: usergroup.VisibilityAll}
	} else if strictParams && (channel.Enabled == nil || !*channel.Enabled) {
		return nil, newManualChannelValidationError(http.StatusBadRequest,
			common.I18nError(i18n.MsgChannelNotFound, channelID))
	}
	if !channel.Custom && (!channelInCurrentSiteScope(ctx, channelID) || !model.AgentTypeChannelAllowed(ctx, instance.AgentType, channelID)) {
		return nil, newManualChannelValidationError(http.StatusBadRequest,
			common.I18nError(i18n.MsgAgentTypeNotSupportChannel, instance.AgentType, channelID))
	}
	if channel.VisibilityType == usergroup.VisibilityGroup {
		if instance.GroupID == 0 {
			return nil, newManualChannelValidationError(http.StatusForbidden,
				common.I18nError(i18n.MsgChannelOnlyForGroup, channelID))
		}
		ancestors, _ := usergroup.GetAncestorIDs(ctx, instance.GroupID)
		visible, _ := usergroup.IsResourceVisible(ctx, usergroup.ConfigTypeChannel, channel.ID, channel.VisibilityType, ancestors)
		if !visible {
			return nil, newManualChannelValidationError(http.StatusForbidden,
				common.I18nError(i18n.MsgChannelNotVisible, channelID))
		}
	}
	if len(preset.Config) == 0 {
		return nil, newManualChannelValidationError(http.StatusBadRequest,
			common.I18nError(i18n.MsgMissingConfigKeys))
	}
	for key, value := range preset.Config {
		if strings.TrimSpace(key) == "" {
			return nil, newManualChannelValidationError(http.StatusBadRequest,
				common.I18nError(i18n.MsgEmptyConfigKey, 1))
		}
		if strings.TrimSpace(value) == "" {
			return nil, newManualChannelValidationError(http.StatusBadRequest,
				common.I18nError(i18n.MsgEmptyConfigValue, key))
		}
	}
	if strictParams {
		for _, param := range channel.Params() {
			if strings.TrimSpace(preset.Config[param.Key]) == "" {
				return nil, newManualChannelValidationError(http.StatusBadRequest,
					common.I18nError(i18n.MsgBadRequestParamRequired, "channels[].config."+param.Key))
			}
		}
	}
	return &channel, nil
}

func applyManualChannelConfig(r *http.Request, instance *model.Instance, preset manualChannelPreset) (*manualChannelApplyResult, error) {
	ctx := r.Context()
	channel, err := validateManualChannelConfig(ctx, instance, preset, false)
	if err != nil {
		return nil, err
	}
	params := map[string]string{"channel": preset.Channel}
	for key, value := range preset.Config {
		params[key] = value
	}

	if channel.Custom {
		customConfig, err := channel.ParseCustomConfig()
		if err != nil {
			return nil, common.I18nError(i18n.MsgParseCustomChannelConfigFailed)
		}
		params["is_custom"] = "true"
		// 与 setChannel 对齐：server 字段即为最终 openclaw.json 的模板，
		// {{key}} 占位符会被替换为用户提交的凭证值。
		originalTpl := string(customConfig.Server)
		for key, value := range preset.Config {
			b, _ := json.Marshal(value)
			quoted := string(b)
			inner := quoted[1 : len(quoted)-1]
			originalTpl = strings.ReplaceAll(originalTpl, `"{{`+key+`}}"`, quoted)
			originalTpl = strings.ReplaceAll(originalTpl, `{{`+key+`}}`, inner)
		}
		var cfgMap map[string]interface{}
		if json.Unmarshal([]byte(originalTpl), &cfgMap) != nil {
			return nil, common.I18nError(i18n.MsgBadRequestParamInvalid, "server",
				"rendered config is not valid JSON")
		}
		if _, ok := cfgMap["enabled"]; !ok {
			cfgMap["enabled"] = true
		}
		cfgJSON, _ := json.Marshal(cfgMap)
		params["channel_config"] = string(cfgJSON)
	}

	result := &manualChannelApplyResult{}
	if preset.Channel == "msteams" {
		route, endpoint, err := ensureAgentProxyRoute(r, instance, model.AgentProxyRouteKindTeams)
		if err != nil {
			return nil, err
		}
		result.ProxyRouteID = route.RouteID
		result.ProxyEndpoint = endpoint
		params["proxy_route_id"] = route.RouteID
		params["proxy_endpoint"] = endpoint
		params["teams_endpoint"] = endpoint
		params["webhook_port"] = fmt.Sprintf("%d", route.TargetPort)
		params["webhook_path"] = route.TargetPath
	}
	if preset.Channel == "line" {
		route, endpoint, err := ensureAgentProxyRoute(r, instance, model.AgentProxyRouteKindLine)
		if err != nil {
			return nil, err
		}
		result.ProxyRouteID = route.RouteID
		result.ProxyEndpoint = endpoint
		params["proxy_route_id"] = route.RouteID
		params["proxy_endpoint"] = endpoint
		params["webhook_port"] = fmt.Sprintf("%d", route.TargetPort)
		params["webhook_path"] = route.TargetPath
	}
	if preset.Channel == "feishu" {
		params["feishu_domain"] = "feishu"
	} else if preset.Channel == "lark" {
		params["feishu_domain"] = "lark"
	}

	scriptName, err := ResolveScript(ctx, "set_channel", instance.AgentType)
	if err != nil {
		return nil, common.I18nRichError(err, i18n.MsgParseSetChannelScriptFailed)
	}
	result.Output, err = channelScriptRunner(ctx, instance.InstanceId, scriptName, 60, instance.RuntimeUser, nil, params)
	if err != nil {
		if egressDiagnosticChannelWhitelist[preset.Channel] {
			err = maybeWrapEgressBlocked(ctx, instance.InstanceId, err)
		}
		return nil, common.EnsureRichErrorOrPanic(err).WithI18nPrefix(i18n.MsgChannelConfigFailed)
	}
	return result, nil
}

func HandleSetChannel(w http.ResponseWriter, r *http.Request) {
	handleSetChannel(w, r, defaultStatusResolver)
}

func handleSetChannel(w http.ResponseWriter, r *http.Request, resolver instanceStatusResolver) {
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
		writeError(w, r, http.StatusBadRequest, common.EnsureRichErrorOrPanic(err))
		return
	}

	setChannel(w, r, instance, user.ID, resolver)
}

// setChannel 设置通道配置，用户端和管控端共享。
func setChannel(w http.ResponseWriter, r *http.Request, instance *model.Instance, notifyUID uint, resolver instanceStatusResolver) {
	// 【关键防护】校验实例是否支持通道配置
	if err := checkInstanceSupportsChannel(r.Context(), instance); err != nil {
		writeError(w, r, http.StatusForbidden, common.EnsureRichErrorOrPanic(err))
		return
	}
	if rejectLocalOrWrite(w, r, instance) {
		return
	}
	if _, err := requireInstanceRunning(r.Context(), instance, resolver); err != nil {
		writeAgentGuardError(w, r, err)
		return
	}

	channel := strings.TrimSpace(r.FormValue("channel"))
	if channel == "" {
		writeError(w, r, http.StatusBadRequest, common.I18nError(i18n.MsgBadRequestParamRequired, "channel"))
		return
	}
	keys := r.Form["key"]
	values := r.Form["value"]
	if len(keys) == 0 {
		writeError(w, r, http.StatusBadRequest, common.I18nError(i18n.MsgMissingConfigKeys))
		return
	}
	if len(values) == 0 {
		writeError(w, r, http.StatusBadRequest, common.I18nError(i18n.MsgMissingConfigValues))
		return
	}
	if len(keys) != len(values) {
		writeError(w, r, http.StatusBadRequest, common.I18nError(i18n.MsgKeyValueCountMismatch, len(keys), len(values)))
		return
	}
	config := make(map[string]string, len(keys))
	for i := range keys {
		if keys[i] == "" {
			writeError(w, r, http.StatusBadRequest, common.I18nError(i18n.MsgEmptyConfigKey, i+1))
			return
		}
		if values[i] == "" {
			writeError(w, r, http.StatusBadRequest, common.I18nError(i18n.MsgEmptyConfigValue, keys[i]))
			return
		}
		config[keys[i]] = values[i]
	}

	// v7：白名单校验（自定义 channel 豁免）
	var ch model.AIChannel
	chFound := model.DB(r.Context()).Where("channel_id = ?", channel).First(&ch).Error == nil
	isCustom := chFound && ch.Custom
	if !isCustom && (!channelInCurrentSiteScope(r.Context(), channel) || !model.AgentTypeChannelAllowed(r.Context(), instance.AgentType, channel)) {
		writeError(w, r, http.StatusBadRequest,
			common.I18nError(i18n.MsgAgentTypeNotSupportChannel, instance.AgentType, channel))
		return
	}
	// 可见性校验：通道设置为 group 可见时，实例必须在对应分组内
	if chFound && ch.VisibilityType == usergroup.VisibilityGroup {
		if instance.GroupID == 0 {
			writeError(w, r, http.StatusForbidden,
				common.I18nError(i18n.MsgChannelOnlyForGroup, channel))
			return
		}
		chGroupIDs, _ := usergroup.GetAncestorIDs(r.Context(), instance.GroupID)
		visible, _ := usergroup.IsResourceVisible(r.Context(), usergroup.ConfigTypeChannel, ch.ID, ch.VisibilityType, chGroupIDs)
		if !visible {
			writeError(w, r, http.StatusForbidden,
				common.I18nError(i18n.MsgChannelNotVisible, channel))
			return
		}
	}

	// Build params: channel + each key-value pair as individual template variable
	params := map[string]string{
		"channel": channel,
	}
	for i := range keys {
		params[keys[i]] = values[i]
	}

	// Custom channel: server 字段就是模板，{{key}} 占位符会被替换为用户提交的凭证值
	if isCustom {
		ccfg, err := ch.ParseCustomConfig()
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, common.I18nError(i18n.MsgParseCustomChannelConfigFailed))
			return
		}

		params["is_custom"] = "true"

		// 模板渲染：server 字段即为最终 openclaw.json 中该通道下的 JSON 结构
		// 管理员负责写完整结构，{{key}} 占位符会被替换为用户提交的凭证值
		// 顶层自动补 enabled=true（如果 server 中没有）
		originalTpl := string(ccfg.Server)

		// 步骤 1：占位符校验——扫描 server 中所有 {{...}} 占位符，
		// 确保每个占位符 key 都在用户提交的 keys 中，一次性报告所有未替换的占位符。
		// 用正则严格匹配 {{...}} 配对，避免 strings.Index 的配对错位问题。
		placeholderRe := regexp.MustCompile(`\{\{\s*([^{}\s]+)\s*\}\}`)
		allPlaceholders := placeholderRe.FindAllStringSubmatch(originalTpl, -1)

		keysSet := make(map[string]bool, len(keys))
		for _, k := range keys {
			keysSet[k] = true
		}

		var missing []string
		seenMissing := make(map[string]bool)
		for _, m := range allPlaceholders {
			name := strings.TrimSpace(m[1])
			if !keysSet[name] && !seenMissing[name] {
				missing = append(missing, "{{"+name+"}}")
				seenMissing[name] = true
			}
		}
		if len(missing) > 0 {
			writeError(w, r, http.StatusBadRequest, common.I18nError(
				i18n.MsgBadRequestParamInvalid, "server",
				"unresolved placeholder(s): "+strings.Join(missing, ", ")))
			return
		}

		// 步骤 2：JSON 转义替换——用 json.Marshal 对 value 转义后再替换，
		// 防止 value 中的 "、\、换行等 JSON 特殊字符逃逸出字符串边界造成 JSON 注入。
		// 支持两种占位符写法：
		//   (a) 独立字段值   "token": "{{token}}"        → 连引号整体替换（带引号形式）
		//   (b) 嵌入字符串   "url": "https://{{host}}/x" → 裸替换（去引号的转义内容）
		tplStr := originalTpl
		for i := range keys {
			b, mErr := json.Marshal(values[i])
			if mErr != nil {
				writeError(w, r, http.StatusInternalServerError, common.I18nError(i18n.MsgMarshalParamsFailed))
				return
			}
			quoted := string(b)                // 带引号形式：`"escaped value"`
			inner := quoted[1 : len(quoted)-1] // 去掉首尾引号，用于嵌入字符串场景
			// 先替换带引号的独立占位符，再替换裸占位符
			tplStr = strings.ReplaceAll(tplStr, `"{{`+keys[i]+`}}"`, quoted)
			tplStr = strings.ReplaceAll(tplStr, `{{`+keys[i]+`}}`, inner)
		}

		// 步骤 3：解析校验——渲染后必须是合法 JSON，否则 400（在 hatchery 层拦截，
		// 不把非法 JSON 传到实例端脚本层才失败）。同时用 map key 判断顶层 enabled，
		// 避免字符串 Contains 对字段值中含 "enabled" 子串的误判。
		var cfgMap map[string]interface{}
		if err := json.Unmarshal([]byte(tplStr), &cfgMap); err != nil {
			writeError(w, r, http.StatusBadRequest, common.I18nError(
				i18n.MsgBadRequestParamInvalid, "server", "rendered config is not valid JSON"))
			return
		}
		if _, ok := cfgMap["enabled"]; !ok {
			cfgMap["enabled"] = true
		}
		// 步骤 4：兜底 merge——将用户提交但 server 模板未声明占位符的 key=value
		// 补到最终 config 顶层。修复 form 模式（server 仅含 serverUrl/wsUrl）下，
		// 用户填写的凭证（如 appId/uin）因无对应 {{key}} 占位符被静默丢弃的问题。
		// 模板已用占位符渲染的 key 跳过，避免重复或覆盖模板渲染结果。
		mergeFallbackCredentials(cfgMap, keys, values, allPlaceholders)
		cfgJSON, err := json.Marshal(cfgMap)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, common.I18nError(
				i18n.MsgSerializeScriptFailedWithDetail, err))
			return
		}
		params["channel_config"] = string(cfgJSON)
	}

	var proxyRoute *model.AgentProxyRoute
	var proxyEndpoint string
	if channel == "msteams" {
		var err error
		proxyRoute, proxyEndpoint, err = ensureAgentProxyRoute(r, instance, model.AgentProxyRouteKindTeams)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, common.EnsureRichErrorOrPanic(err))
			return
		}
		params["proxy_route_id"] = proxyRoute.RouteID
		params["proxy_endpoint"] = proxyEndpoint
		params["teams_endpoint"] = proxyEndpoint
		params["webhook_port"] = fmt.Sprintf("%d", proxyRoute.TargetPort)
		params["webhook_path"] = proxyRoute.TargetPath
	}
	if channel == "line" {
		var err error
		proxyRoute, proxyEndpoint, err = ensureAgentProxyRoute(r, instance, model.AgentProxyRouteKindLine)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, common.EnsureRichErrorOrPanic(err))
			return
		}
		// line 通道需要 access_token 和 secret，否则 400
		if _, ok := params["channel_access_token"]; !ok {
			writeError(w, r, http.StatusBadRequest, common.I18nError(i18n.MsgLINEMissingAccessToken))
			return
		}
		if _, ok := params["channel_secret"]; !ok {
			writeError(w, r, http.StatusBadRequest, common.I18nError(i18n.MsgLINEMissingSecret))
			return
		}
		params["proxy_route_id"] = proxyRoute.RouteID
		params["proxy_endpoint"] = proxyEndpoint
		params["webhook_port"] = fmt.Sprintf("%d", proxyRoute.TargetPort)
		params["webhook_path"] = proxyRoute.TargetPath
	}

	if channel == "feishu" {
		params["feishu_domain"] = "feishu"
	}

	if channel == "lark" {
		params["feishu_domain"] = "lark"
	}

	// v7：按 agent_type 分派 set_channel 脚本
	scriptName, rerr := ResolveScript(r.Context(), "set_channel", instance.AgentType)
	if rerr != nil {
		writeError(w, r, http.StatusBadRequest, common.I18nRichError(rerr, i18n.MsgParseSetChannelScriptFailed))
		return
	}
	output, err := channelScriptRunner(r.Context(), instance.InstanceId, scriptName, 60, instance.RuntimeUser, nil, params)
	if err != nil {
		// 仅对已知依赖出站的通道做 egress 诊断，命中则替换面向用户文案。
		if egressDiagnosticChannelWhitelist[channel] {
			err = maybeWrapEgressBlocked(r.Context(), instance.InstanceId, err)
		}
		channelConfigFailedTitle := i18n.T(r.Context(), i18n.MsgChannelConfigFailed)
		richErr := common.EnsureRichErrorOrPanic(err).WithI18nPrefix(i18n.MsgChannelConfigFailed)
		writeError(w, r, http.StatusInternalServerError, richErr)
		notifyCtx := common.DetachContext(r.Context())
		go createErrorNotification(notifyUID, instance.ID, instance.Name, model.NotifyTypeChannelConfigFailed, channelConfigFailedTitle, richErr, notifyCtx)
		return
	}

	resp := map[string]interface{}{"ok": true, "output": output}
	if (channel == "msteams" || channel == "line") && proxyRoute != nil {
		resp["proxy_route_id"] = proxyRoute.RouteID
		resp["proxy_endpoint"] = proxyEndpoint
	}
	if channel == "msteams" && proxyRoute != nil {
		resp["teams_endpoint"] = proxyEndpoint
	}
	jsonOK(w, resp)
}

// reservedChannelConfigKeys 系统保留的通道配置字段，用户凭证值不应覆盖这些字段。
var reservedChannelConfigKeys = map[string]bool{
	"enabled": true,
	"channel": true,
}

// mergeFallbackCredentials 将用户提交的凭证（key=value）兜底写入自定义通道最终配置。
// 仅当该 key 未作为占位符出现在 server 模板中时才写入（避免覆盖模板已渲染的结果）。
// 用于修复：form 模式创建的通道 server 仅含 serverUrl/wsUrl，用户填写的 appId/uin 等
// 因无对应 {{key}} 占位符而在渲染环节被静默丢弃的问题。
// 注意：只有用户真正提交的 key 才会被兜底；模板中已声明并使用 {{key}} 渲染的字段不重复写入。
//
// tplPlaceholders: regexp.FindAllStringSubmatch 的返回值，每个元素 m 中 m[1] 为 {{...}} 内的占位符名。
func mergeFallbackCredentials(cfg map[string]interface{}, keys, values []string, tplPlaceholders [][]string) {
	if len(keys) != len(values) {
		return
	}
	declared := make(map[string]bool, len(tplPlaceholders))
	for _, m := range tplPlaceholders {
		declared[strings.TrimSpace(m[1])] = true
	}
	for i := range keys {
		if declared[keys[i]] || reservedChannelConfigKeys[keys[i]] {
			continue
		}
		// 用户提交的凭证优先：即使顶层已存在同名静态字段，也以用户提交值为准。
		cfg[keys[i]] = values[i]
	}
}

func HandleDelChannel(w http.ResponseWriter, r *http.Request) {
	handleDelChannel(w, r, defaultStatusResolver)
}

func handleDelChannel(w http.ResponseWriter, r *http.Request, resolver instanceStatusResolver) {
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
		writeError(w, r, http.StatusBadRequest, common.EnsureRichErrorOrPanic(err))
		return
	}

	deleteChannel(w, r, instance, resolver)
}

// deleteChannel 删除通道，用户端和管控端共享。
func deleteChannel(w http.ResponseWriter, r *http.Request, instance *model.Instance, resolver instanceStatusResolver) {
	// v7：补齐 ① 能力 guard（真实代码 HandleDelChannel 遗漏了此 guard，与 SetChannel 不对称）
	if err := checkInstanceSupportsChannel(r.Context(), instance); err != nil {
		writeError(w, r, http.StatusForbidden, common.EnsureRichErrorOrPanic(err))
		return
	}

	// 本地实例：通道配置属于 CVM 侧能力，本地 agent 不支持。
	if rejectLocalOrWrite(w, r, instance) {
		return
	}
	// 状态准入：仅 running 状态允许删除通道
	if _, err := requireInstanceRunning(r.Context(), instance, resolver); err != nil {
		writeAgentGuardError(w, r, err)
		return
	}

	channel := r.FormValue("channel")
	if channel == "" {
		writeError(w, r, http.StatusBadRequest, common.I18nError(i18n.MsgBadRequestParamRequired, "channel"))
		return
	}

	// v7：② 白名单校验（自定义 channel 豁免）
	var ch model.AIChannel
	chFound := model.DB(r.Context()).Where("channel_id = ?", channel).First(&ch).Error == nil
	isCustom := chFound && ch.Custom
	if !isCustom && !model.AgentTypeChannelAllowed(r.Context(), instance.AgentType, channel) {
		writeError(w, r, http.StatusBadRequest,
			common.I18nError(i18n.MsgAgentTypeNotSupportChannel, instance.AgentType, channel))
		return
	}

	// Build params: channel + each key-value pair as individual template variable
	params := map[string]string{
		"channel": channel,
	}

	// v7：③ 按 agent_type 分派 del_channel 脚本
	// 自定义通道从 DB CustomConfig.server.deleteFeature 读取 feature 名，
	// 缺失时 fallback 到 "del_channel"（通用删除脚本）。
	deleteFeature := "del_channel"
	if isCustom {
		ccfg, parseErr := ch.ParseCustomConfig()
		if parseErr != nil {
			writeError(w, r, http.StatusInternalServerError, common.I18nError(i18n.MsgParseCustomChannelConfigFailed))
			return
		}
		scfg, parseErr := ccfg.ParseServerConfig()
		if parseErr != nil {
			writeError(w, r, http.StatusInternalServerError, common.I18nError(i18n.MsgParseCustomChannelConfigFailed))
			return
		}
		// 为 ServerConfig 零值字段填充默认值（如 deleteFeature 等），
		// 显式配置的字段（非零值）保持不变。
		scfg.DefaultsForChannel(channel)
		if scfg.DeleteFeature != "" {
			deleteFeature = scfg.DeleteFeature
		}
	}
	scriptName, rerr := ResolveScript(r.Context(), deleteFeature, instance.AgentType)
	if rerr != nil {
		writeError(w, r, http.StatusBadRequest, common.I18nRichError(rerr, i18n.MsgParseDelChannelScriptFailed))
		return
	}
	output, err := delChannelScriptRunner(r.Context(), instance.InstanceId, scriptName, 60, instance.RuntimeUser, nil, params)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, common.EnsureRichErrorOrPanic(err))
		return
	}
	if channel == "msteams" {
		if err := model.DB(r.Context()).Model(&model.AgentProxyRoute{}).
			Where("instance_id = ? AND kind = ?", instance.InstanceId, model.AgentProxyRouteKindTeams).
			Update("enabled", false).Error; err != nil {
			writeError(w, r, http.StatusInternalServerError, common.I18nRichError(err, i18n.MsgDisableProxyRouteFailed))
			return
		}
		if err := RefreshAllRuleSetsForRequiredRules(r.Context()); err != nil {
			Logger(r.Context()).Warn("[Channel] msteams 已删除但安全组规则刷新失败", "instance_id", instance.ID, "cvm_id", instance.InstanceId, "error", err)
		}
	}
	if channel == "line" {
		if err := model.DB(r.Context()).Model(&model.AgentProxyRoute{}).
			Where("instance_id = ? AND kind = ?", instance.InstanceId, model.AgentProxyRouteKindLine).
			Update("enabled", false).Error; err != nil {
			writeError(w, r, http.StatusInternalServerError, common.I18nRichError(err, i18n.MsgDisableProxyRouteFailed))
			return
		}
		if err := RefreshAllRuleSetsForRequiredRules(r.Context()); err != nil {
			Logger(r.Context()).Warn("[Channel] line 已删除但安全组规则刷新失败", "instance_id", instance.ID, "cvm_id", instance.InstanceId, "error", err)
		}
	}

	jsonOK(w, map[string]interface{}{"ok": true, "output": output})
}

// listChannelsScriptRunner 是 listInstanceChannels 内部 TAT 调用的 hook，便于单测 mock。
var listChannelsScriptRunner = RunScript

// listInstanceChannels 查询实例已配置通道，归一化 wecom 字段，返回解析后的通道数据。
func listInstanceChannels(ctx context.Context, instance *model.Instance) (map[string]interface{}, error) {
	scriptName, rerr := ResolveScript(ctx, "list_channels", instance.AgentType)
	if rerr != nil {
		return nil, common.I18nRichError(rerr, i18n.MsgParseListChannelsScriptFailed)
	}
	output, err := listChannelsScriptRunner(ctx, instance.InstanceId, scriptName, 60, instance.RuntimeUser, nil, nil)
	if err != nil {
		return nil, err
	}

	output = normalizeWecomShape(output)

	var channels map[string]interface{}
	if err := json.Unmarshal([]byte(output), &channels); err != nil {
		return nil, common.I18nRichError(err, i18n.MsgParseChannelListFailed)
	}

	return channels, nil
}

func HandleChannelsList(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		return
	}

	// 不传 id 时，返回数据库中所有 channel 列表（含参数定义）
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		var channels []model.AIChannel
		model.DB(r.Context()).Find(&channels)
		channels = model.SortChannelsByPredefined(channels)
		channels = filterChannelsByCurrentSiteScope(r.Context(), channels)
		// 按可见性过滤通道列表（支持 agent_id 参数指定实例，查其绑定的分组）
		var agentGroupID uint
		if agentIDStr := r.URL.Query().Get("agent_id"); agentIDStr != "" {
			if instID, err := strconv.ParseUint(agentIDStr, 10, 64); err == nil && instID > 0 {
				var inst model.Instance
				if model.DB(r.Context()).Select("group_id").Where("id = ? AND user_id = ?", instID, user.ID).First(&inst).Error == nil {
					agentGroupID = inst.GroupID
				}
			}
		}
		channels = usergroup.FilterChannelsByVisibility(r.Context(), channels, agentGroupID)

		// final：追加 AgentTypes 字段，告知前端每个 channel 被哪些 agent_type 支持。
		// 语义与 /admin/channels 对齐（见 admin_channels.go::enrichChannelsWithAgentTypes）。
		// 前端据此在"添加通道"弹窗按实例 agent_type 置灰不兼容卡片。
		// 字段命名风格：与 AIChannel 其它导出字段（ChannelID/Name/Enabled/Custom/CustomConfig）
		// 保持一致的 Go 零改名（PascalCase），JSON 输出即 "AgentTypes"。
		type channelItem struct {
			model.AIChannel
			Params     []model.ChannelParam `json:"params"`
			AgentTypes []string
		}
		items := make([]channelItem, 0, len(channels))
		for _, ch := range channels {
			var ats []string
			if ch.Custom {
				ats = model.GetChannelSupportedAgentTypes(r.Context())
			} else {
				ats = model.SupportedAgentTypesByChannel(r.Context(), ch.ChannelID)
			}
			items = append(items, channelItem{
				AIChannel:  ch,
				Params:     ch.Params(),
				AgentTypes: ats,
			})
		}
		jsonOK(w, items)
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, common.EnsureRichErrorOrPanic(err))
		return
	}

	channels, err := listInstanceChannels(r.Context(), instance)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, common.EnsureRichErrorOrPanic(err))
		return
	}
	resp := map[string]interface{}{
		"agent_type":                    instance.AgentType,
		"agent_type_supported_channels": filterChannelIDsByCurrentSiteScope(r.Context(), model.SupportedChannelsByAgentType(r.Context(), instance.AgentType)),
		"channels":                      channels,
	}
	jsonOK(w, resp)
}

// HandleAutoChannel 通过 SSE 流式执行自动配置脚本（QQ / 飞书）。
// 脚本 stdout 输出结构化 JSON 行，Go 逐行解析后通过 SSE 转发给前端。
// SSE 事件类型：qrcode（展示二维码）、log（日志）、progress（进度）、done/fail（最终结果）。
func HandleAutoChannel(w http.ResponseWriter, r *http.Request) {
	handleAutoChannel(w, r, defaultStatusResolver)
}

func handleAutoChannel(w http.ResponseWriter, r *http.Request, resolver instanceStatusResolver) {
	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, common.EnsureRichErrorOrPanic(err))
		return
	}

	if rejectLocalOrWrite(w, r, instance) {
		return
	}

	// 状态准入：仅 running 状态允许自动配置通道
	// 放在 SSE header 设置之前，失败走标准 409 JSON 响应。
	if _, err := requireInstanceRunning(r.Context(), instance, resolver); err != nil {
		writeAgentGuardError(w, r, err)
		return
	}

	channel := r.URL.Query().Get("channel")
	if channel == "" {
		channel = "qqbot"
	}

	// 预解析：自定义通道需提前查 DB 校验 pairingMode 等，
	// 必须在 SSE header 设置之前执行，这样校验失败能走标准 400 JSON 响应。
	var scfg model.ServerConfig // 自定义通道的服务器配置
	var ch model.AIChannel      // DB 中的通道记录
	var chFound bool            // DB 是否查到通道记录
	if _, ok := autoChannelFeature[channel]; !ok {
		// 非预定义通道：从 DB 查询自定义通道配置
		chFound = model.DB(r.Context()).Where("channel_id = ?", channel).First(&ch).Error == nil
		if !chFound || !ch.Custom {
			writeError(w, r, http.StatusBadRequest, common.I18nError(i18n.MsgChannelNotSupportAutoConfig, channel))
			return
		}
		ccfg, parseErr := ch.ParseCustomConfig()
		if parseErr != nil {
			writeError(w, r, http.StatusInternalServerError, common.I18nError(i18n.MsgParseCustomChannelConfigFailed))
			return
		}
		scfg, parseErr = ccfg.ParseServerConfig()
		if parseErr != nil {
			writeError(w, r, http.StatusInternalServerError, common.I18nError(i18n.MsgParseCustomChannelConfigFailed))
			return
		}
		// 为 ServerConfig 零值字段填充默认值（如 pairingMode、autoFeature 等），
		// 这样管理员只需配置 channel_id + name + cred_fields，其余参数由默认值兜底。
		// 显式配置的字段（非零值）保持不变。
		scfg.DefaultsForChannel(channel)

		// 自定义通道未开启配对码模式 → 自动配置不适用
		// 当管理员未配置 pairingMode（nil）或显式设为 false 时，返回 400
		if scfg.PairingMode == nil || !*scfg.PairingMode {
			writeError(w, r, http.StatusBadRequest, common.I18nError(i18n.MsgChannelNotSupportAutoConfig, channel))
			return
		}
	}

	// feature/timeout 解析：预定义通道走 map，自定义通道走 DB ServerConfig
	feature, ok := autoChannelFeature[channel]
	if !ok {
		feature = scfg.AutoFeature
	}
	scriptName, err := ResolveScript(r.Context(), feature, instance.AgentType)
	if err != nil {
		writeError(w, r, http.StatusBadRequest,
			common.I18nRichError(err, i18n.MsgAgentTypeNotSupportChannelAutoConfig, instance.AgentType, channel))
		return
	}
	timeout, ok := autoChannelTimeout[channel]
	if !ok {
		if scfg.AutoTimeout > 0 {
			timeout = scfg.AutoTimeout
		} else {
			timeout = 600 // 自定义通道默认超时
		}
	}

	// 自定义通道 phone 参数校验（必须在 SSE header 之前完成，这样校验失败能走标准 400 JSON 响应）
	var phoneForCustom string
	if scfg.PhoneRequired != nil && *scfg.PhoneRequired {
		phoneForCustom = r.URL.Query().Get("phone")
		if phoneForCustom == "" {
			writeError(w, r, http.StatusBadRequest, common.I18nError(i18n.MsgBadRequestParamRequired, "phone"))
			return
		}
		// 选择正则：server.phonePattern 优先，否则 fallback 到内置 whatsappPhoneRegexp
		phoneRegexp := whatsappPhoneRegexp
		if scfg.PhonePattern != "" {
			compiled, compileErr := regexp.Compile(scfg.PhonePattern)
			if compileErr != nil {
				slog.Warn("[AutoChannel] invalid phonePattern in DB, fallback to default",
					"channel", channel, "pattern", scfg.PhonePattern, "err", compileErr)
			} else {
				phoneRegexp = compiled
			}
		}
		if !phoneRegexp.MatchString(phoneForCustom) {
			writeError(w, r, http.StatusBadRequest, common.I18nError(i18n.MsgInvalidPhoneFormat))
			return
		}
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, r, http.StatusInternalServerError, common.I18nError(i18n.MsgStreamNotSupported))
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	sendSSE := func(event string, data interface{}) {
		jsonData, _ := json.Marshal(data)
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, jsonData)
		flusher.Flush()
	}

	// newJSONLinesHandler 创建一个 JSON Lines 逐行解析的 onOutput 回调。
	// 脚本输出结构化 JSON 行，逐行解析 action 字段后原样转发对应的 SSE 事件。
	newJSONLinesHandler := func() func(string) {
		processedOffset := 0
		return func(fullOutput string) {
			remaining := fullOutput[processedOffset:]
			for {
				nlIdx := strings.IndexByte(remaining, '\n')
				if nlIdx < 0 {
					break // 最后一段没有换行符，可能是不完整的行，等下次再处理
				}
				line := strings.TrimSpace(remaining[:nlIdx])
				processedOffset += nlIdx + 1
				remaining = remaining[nlIdx+1:]

				if line == "" {
					continue
				}
				// 快速提取 action 字段，判断 SSE 事件类型
				var peek struct {
					Action string `json:"action"`
				}
				if json.Unmarshal([]byte(line), &peek) != nil {
					continue // 跳过非 JSON 行（如 DeprecationWarning）
				}
				switch peek.Action {
				case "show_qrcode":
					sendSSE("qrcode", normalizeShowQrcode(json.RawMessage(line)))
				case "show_pairing_code":
					sendSSE("pairing_code", json.RawMessage(line))
				case "log":
					sendSSE("log", json.RawMessage(line))
				case "progress":
					sendSSE("progress", json.RawMessage(line))
				case "finish":
					sendSSE("finish", json.RawMessage(line))
				}
			}
		}
	}

	var params map[string]string
	var onOutput func(string)

	switch channel {
	case "qqbot": //deprecated: 后续删除
		qrSent := false
		onOutput = func(fullOutput string) {
			if qrSent {
				return
			}
			// TAT 混合 stderr/stdout，URL 在 "[create] 二维码链接: " 后面
			const prefix = "[create] 二维码链接: "
			if idx := strings.Index(fullOutput, prefix); idx >= 0 {
				rest := fullOutput[idx+len(prefix):]
				url := strings.SplitN(rest, "\n", 2)[0]
				sendSSE("qrcode", map[string]string{"url": strings.TrimSpace(url)})
				qrSent = true
			}
		}
	case "feishu":
		onOutput = newJSONLinesHandler()
		// 飞书通道：构建包含动态欢迎词的参数
		siteConfig := model.GetSiteConfig(r.Context())
		greeting := fmt.Sprintf("Hi，我是你刚刚在 %s 创建的机器人，你现在可以跟我聊天了！", siteConfig.Name)
		params = map[string]string{"greeting": greeting, "feishu_domain": "feishu"}
	case "lark":
		onOutput = newJSONLinesHandler()
		// Lark 通道：构建包含动态欢迎词的参数
		siteConfig := model.GetSiteConfig(r.Context())
		greeting := fmt.Sprintf("Hi, I am the robot you just created on %s, you can chat with me now!", siteConfig.Name)
		// 在扫码时 feishu_domain 不要设置为 lark，否则会出现链接失效的错误
		// 扫码链接时直接请求 open.feishu.com 会自动转移到 open.larksuite.com
		params = map[string]string{"greeting": greeting, "feishu_domain": "feishu"}
	case "openclaw-weixin", "weixin": // v7：新增 Hermes/ACE 用的 weixin 别名
		onOutput = newJSONLinesHandler()
	case "whatsapp": // 内置扫码登录通道（与自定义通道框架下的 openclaw_whatsapp 配对码模式并存，互不影响）
		onOutput = newJSONLinesHandler()
	default:
		// 通用配对码模式（由 CustomConfig.server.pairingMode 驱动）
		// 自定义通道的参数从 DB 读取，而非硬编码
		// 注意：DB 查询和 pairingMode/AutoFeature 校验已在 SSE header 之前完成
		onOutput = newJSONLinesHandler()
		params = map[string]string{}

		// 手机号参数已在 SSE header 之前完成校验（见上方 phoneForCustom），直接使用
		if phoneForCustom != "" {
			params["phone_number"] = phoneForCustom
		}

		// 从 server 配置读取策略（有默认值兜底）
		if scfg.DmPolicy != "" {
			params["dm_policy"] = scfg.DmPolicy
		} else {
			params["dm_policy"] = "allowlist"
		}
		if scfg.SelfChatMode != nil {
			// 布尔值 "true"/"false" 与脚本中 --argjson 兼容
			params["self_chat_mode"] = strconv.FormatBool(*scfg.SelfChatMode)
		} else {
			params["self_chat_mode"] = "true"
		}
	}

	// 前置 egress 诊断：飞书/微信/WhatsApp 等脚本强依赖 Agent 出站访问外部服务器
	// （open.feishu.cn / micromsg.qq.com / g.whatsapp.net 等），出站被拒时脚本会跑 ~3 分钟才超时报错。
	// 在启动脚本前直接查一次实例的安全组出站规则，blocked 即 fast-fail,
	// 省掉用户的无效等待,也避免脚本层通过 finish(error) 上报业务失败再回溯诊断的复杂链路。
	// DiagnoseInstanceEgress 自带 5s 超时；诊断失败（云 API 不可用）不拦截,走正常脚本路径。
	//
	// egress 诊断触发条件：
	//   - 预定义通道：走硬编码白名单 egressDiagnosticChannelWhitelist
	//   - 自定义通道：走 DB ServerConfig.egressRequired
	needEgressDiag := egressDiagnosticChannelWhitelist[channel]
	if !needEgressDiag && ch.ID > 0 {
		needEgressDiag = scfg.EgressRequired != nil && *scfg.EgressRequired
	}
	if needEgressDiag {
		if blocked, diagErr := DiagnoseInstanceEgress(r.Context(), instance.InstanceId); diagErr != nil {
			slog.Warn("[AutoChannel] egress diagnostic skipped",
				"instance_id", instance.InstanceId, "channel", channel, "diag_err", diagErr)
		} else if blocked {
			slog.Info("[AutoChannel] egress blocked, short-circuit",
				"instance_id", instance.InstanceId, "channel", channel)
			sendSSE("fail", map[string]string{"message": i18n.T(r.Context(), EgressBlockedMessage)})
			return
		}
	}

	slog.Info("[AutoChannel] 开始自动配置", "instance", instance.ID, "user", user.Username, "channel", channel, "params", params)

	output, err := RunScript(r.Context(), instance.InstanceId, scriptName, timeout, instance.RuntimeUser, onOutput, params)
	if err != nil {
		sendSSE("fail", map[string]string{"message": err.Error()})
		return
	}

	if channel == "qqbot" {
		// QQ 通道：解析最终 JSON 结果（最后一行）
		lines := strings.Split(strings.TrimSpace(output), "\n")
		lastLine := lines[len(lines)-1]
		var result map[string]interface{}
		if json.Unmarshal([]byte(lastLine), &result) == nil {
			sendSSE("done", result)
		} else {
			sendSSE("done", map[string]string{"message": i18n.T(r.Context(), i18n.MsgAutoChannelDone)})
		}
	}
}

// normalizeWecomShape 统一 list_channels 返回的 wecom 字段形状：
//
//  1. openclaw 脚本历史契约：wecom 子字段名为 `agent`（企微应用），先改名为
//     `wecom_app`，与前端约定对齐。
//  2. ACE 兼容：ACE（lightclaw.json）原生将 botId/secret/name/websocketUrl/
//     dmPolicy/allowFrom 平铺在 wecom 根下（无 .bot 子对象），而前端契约要求
//     这些字段在 .bot 子对象中。list_channels_ace.sh 脚本层已有 jq 归一化逻辑，
//     但存量实例可能未部署最新脚本，或 jq 执行失败走了兜底路径，导致原始平铺
//     格式直接透传到 Go 层。因此 Go 层兜底：检测 wecom 根下是否有 bot 级字段，
//     有则搬运到 .bot 子对象（与已有内容合并）。
//  3. 结构对齐：缺 wecom_app 则补空对象 `{}`，让 hermes/ace（底层不支持企微
//     应用）与 openclaw 返回同一形状，前端无需针对不同 agent_type 走不同渲染分支。
//  4. 完全没有 wecom 字段的实例（未配置）→ 保持不存在，不无中生有。
//  5. wecom 非对象 / 整体 JSON 无效 → 原样返回（降级兼容）。
func normalizeWecomShape(jsonStr string) string {
	var channels map[string]interface{}
	if json.Unmarshal([]byte(jsonStr), &channels) != nil {
		return jsonStr
	}
	wecom, ok := channels["wecom"].(map[string]interface{})
	if !ok {
		// 包含两种降级情形：①wecom 不存在；②wecom 不是对象（如字符串/数字/null）
		return jsonStr
	}

	changed := false

	// ① openclaw 历史契约：agent → wecom_app 改名
	if agentVal, exists := wecom["agent"]; exists {
		wecom["wecom_app"] = agentVal
		delete(wecom, "agent")
		changed = true
	}

	// ② ACE 兼容：将 wecom 根下平铺的 bot 级字段搬运到 .bot 子对象。
	//    ACE（lightclaw.json）原生将 botId/secret/name 等字段平铺在 wecom 根下，
	//    而前端契约要求这些字段在 .bot 子对象中。list_channels_ace.sh 脚本层已有
	//    jq 归一化逻辑，但存量实例可能未部署最新脚本，或 jq 执行失败走了兜底路径，
	//    导致原始平铺格式直接透传到 Go 层。因此 Go 层也需要兜底搬运。
	//
	//    bot 级字段白名单（与 list_channels_ace.sh 对齐）：
	//      botId, secret, name, websocketUrl, dmPolicy, allowFrom
	botFieldKeys := []string{"botId", "secret", "name", "websocketUrl", "dmPolicy", "allowFrom"}
	botObj, hasBotKey := wecom["bot"].(map[string]interface{})
	if botObj == nil {
		botObj = map[string]interface{}{}
		if !hasBotKey {
			// bot key 原本不存在，补齐空对象
			changed = true
		}
	}
	for _, k := range botFieldKeys {
		if v, exists := wecom[k]; exists {
			botObj[k] = v
			delete(wecom, k)
			changed = true
		}
	}
	wecom["bot"] = botObj

	// ③ 结构对齐：wecom_app 缺则补空对象 {}
	//    ACE/Hermes 底层不支持企微应用，但产品要求前端统一渲染，所以兜底补齐。
	if _, hasApp := wecom["wecom_app"]; !hasApp {
		wecom["wecom_app"] = map[string]interface{}{}
		changed = true
	}

	if !changed {
		return jsonStr
	}

	channels["wecom"] = wecom
	result, err := json.Marshal(channels)
	if err != nil {
		return jsonStr
	}
	return string(result)
}

// normalizeShowQrcode 统一 show_qrcode 事件格式，新增 mode 字段供前端识别：
//
//   - mode="qrlogin"  ：content 为 JSON 字符串 {"qrlogin":{"token":"<short_token>"}}
//     前端用 token 值生成二维码，飞书 App 扫码后走 qrlogin polling（OpenClaw 飞书）
//   - mode="url"      ：content 为裸 URL 字符串
//     前端直接用 QRCodeCanvas 渲染此 URL（Hermes/ACE 飞书、Hermes/ACE 微信）
//   - mode="ascii_art"：content 为 UTF8 字符画字符串
//     前端用 <pre> 渲染（OpenClaw 微信）
//     当脚本输出 render_qr=true 时，Go 层将 content(裸URL) 转为字符画
//
// 兼容性：
//   - 原有 action/content 字段保持不变，mode 是增量新增字段
//   - Hermes 飞书脚本输出 content={"qrlogin":{"token":"https://..."}}，
//     此函数将其展开为裸 URL 并设 mode=url，前端不再需要检测 token.startsWith('http')
//   - OpenClaw 飞书 Device Code Flow 输出 content={"verification_uri":"https://..."}，
//     此函数将其展开为裸 URL 并设 mode=url
//   - OpenClaw 微信脚本输出 content=裸URL + render_qr=true，
//     Go 层将 URL 编码为 UTF8 字符画放 content，mode=ascii_art
//
// 注：此函数保留在 Go 层而非下沉到脚本，原因是 OpenClaw 飞书依赖远程 curl 执行的
// Python 脚本（feishu_bot_creator.py），其输出格式不受本仓库控制，只能在 Go 层统一处理。
func normalizeShowQrcode(raw json.RawMessage) interface{} {
	var evt struct {
		Action   string `json:"action"`
		Content  string `json:"content"`
		RenderQR bool   `json:"render_qr"`
	}
	if json.Unmarshal(raw, &evt) != nil {
		return raw
	}

	mode := ""
	content := evt.Content

	// 尝试解析 content 为 {"qrlogin":{"token":"..."}} 格式
	var qrloginWrapper struct {
		Qrlogin struct {
			Token string `json:"token"`
		} `json:"qrlogin"`
	}
	if json.Unmarshal([]byte(content), &qrloginWrapper) == nil && qrloginWrapper.Qrlogin.Token != "" {
		token := qrloginWrapper.Qrlogin.Token
		if strings.HasPrefix(token, "http") {
			// Hermes 飞书：token 实为 URL，展开为裸 URL
			mode = "url"
			content = token
		} else {
			// OpenClaw 飞书：真正的 qrlogin token
			mode = "qrlogin"
			// content 保持原 JSON 字符串不变（向后兼容）
		}
	} else if strings.HasPrefix(content, "http") {
		// 裸 URL（Hermes/ACE 微信、OpenClaw 微信）
		mode = "url"
	} else {
		// 尝试解析飞书 Device Code Flow 格式：{"verification_uri":"https://..."}
		var verificationWrapper struct {
			VerificationURI string `json:"verification_uri"`
		}
		if json.Unmarshal([]byte(content), &verificationWrapper) == nil &&
			strings.HasPrefix(verificationWrapper.VerificationURI, "http") {
			mode = "url"
			content = verificationWrapper.VerificationURI
		} else {
			// UTF8 字符画
			mode = "ascii_art"
		}
	}

	result := map[string]interface{}{
		"action":  evt.Action,
		"mode":    mode,
		"content": content,
	}

	// 当脚本输出 render_qr=true 时（目前仅 OpenClaw 微信），将 content 中的裸 URL
	// 转为 UTF8 字符画放入 content，mode 改为 ascii_art。
	// 其他通道（Hermes/ACE 飞书/微信）不带 render_qr 标记，完全不受影响。
	if evt.RenderQR && mode == "url" {
		var buf bytes.Buffer
		config := qrterminal.Config{
			HalfBlocks: true,
			Level:      qrterminal.L,
			Writer:     &buf,
			QuietZone:  1,
		}
		qrterminal.GenerateWithConfig(content, config)
		if buf.Len() > 0 {
			result["content"] = buf.String()
			result["mode"] = "ascii_art"
		}
	}

	return result
}
