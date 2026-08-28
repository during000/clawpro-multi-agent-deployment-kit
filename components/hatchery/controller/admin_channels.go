package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"

	hcommon "hatchery/common"
	"hatchery/controller/usergroup"
	"hatchery/i18n"
	"hatchery/model"
)

// adminChannelWithAgentTypes 在 AIChannel 基础上追加 agent_types + visibility 字段，
// 用于 /admin/channels JSON 响应，告知前端每个 channel 被哪些 agent_type 支持。
// - 自定义 channel（Custom=true）返回所有 SupportsChannel=true 的类型；
// - 内置 channel 按 AgentTypeChannelAllowed 反查。
type adminChannelWithAgentTypes struct {
	model.AIChannel
	AgentTypes       []string                       `json:"agent_types"`
	VisibilityGroups []usergroup.VisibilityGroupRef `json:"visibility_groups,omitempty"` // visibility_type='group' 时返回绑定的组列表
}

func queryAllChannels(ctx context.Context) []model.AIChannel {
	var channels []model.AIChannel
	model.DB(ctx).Order("id asc").Find(&channels)
	return model.SortChannelsByPredefined(channels)
}

// enrichChannelsWithAgentTypes 把 []AIChannel 映射为含 agent_types + visibility 字段的响应切片。
func enrichChannelsWithAgentTypes(ctx context.Context, channels []model.AIChannel) []adminChannelWithAgentTypes {
	// 批量获取所有 visibility_type='group' 的通道绑定信息
	groupChannelIDs := make([]uint, 0)
	for _, ch := range channels {
		if ch.VisibilityType == usergroup.VisibilityGroup {
			groupChannelIDs = append(groupChannelIDs, ch.ID)
		}
	}
	bindingMap := usergroup.GetVisibilityGroupRefs(ctx, model.ConfigTypeChannel, groupChannelIDs)

	out := make([]adminChannelWithAgentTypes, 0, len(channels))
	for _, ch := range channels {
		var ats []string
		if ch.Custom {
			ats = model.GetChannelSupportedAgentTypes(ctx)
		} else {
			ats = model.SupportedAgentTypesByChannel(ctx, ch.ChannelID)
		}
		item := adminChannelWithAgentTypes{
			AIChannel:  ch,
			AgentTypes: ats,
		}
		if ch.VisibilityType == usergroup.VisibilityGroup {
			item.VisibilityGroups = bindingMap[ch.ID]
		}
		out = append(out, item)
	}
	return out
}

func HandleAdminChannels(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	// final：响应追加 agent_types 字段，告知前端每个 channel 支持的 agent 类型
	jsonOK(w, map[string]interface{}{
		"channels": enrichChannelsWithAgentTypes(r.Context(), filterChannelsByCurrentSiteScope(r.Context(), queryAllChannels(r.Context()))),
	})
}

func HandleToggleChannel(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	id := r.URL.Query().Get("id")
	var ch model.AIChannel
	if model.DB(r.Context()).Where("id = ?", id).First(&ch).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgChannelNotExist))
		return
	}

	newVal := ch.Enabled == nil || !*ch.Enabled
	model.DB(r.Context()).Model(&ch).Update("enabled", newVal)

	jsonOK(w, map[string]interface{}{"ok": true})
}

// channelIDRegex 通道标识只允许英文字母、数字和下划线
var channelIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

// addChannelRequest 添加自定义通道的请求体
type addChannelRequest struct {
	ChannelID    string          `json:"channel_id"`
	Name         string          `json:"name"`
	CustomConfig json.RawMessage `json:"custom_config"`
}

// HandleAddChannel 添加自定义通道
func HandleAddChannel(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	jsonAPI(w)

	var req addChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidRequestFormat))
		return
	}

	// 校验 channel_id
	if req.ChannelID == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgChannelIDCannotBeEmpty))
		return
	}
	if !channelIDRegex.MatchString(req.ChannelID) {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgChannelIDInvalidChars))
		return
	}

	// 校验 name
	if req.Name == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgChannelNameCannotBeEmpty))
		return
	}

	// 校验 custom_config
	if len(req.CustomConfig) == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgCustomChannelConfigRequired))
		return
	}

	// 解析 custom_config 进行结构校验
	var cfg model.CustomChannelConfig
	if err := json.Unmarshal(req.CustomConfig, &cfg); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgCustomChannelConfigFormatError))
		return
	}

	// 校验 server：允许为空或 {}（DefaultsForChannel 会在运行时填充默认值），
	// 但如果提供了 server 则必须是合法 JSON 对象。
	if len(cfg.Server) > 0 && string(cfg.Server) != "null" && string(cfg.Server) != "{}" {
		// 校验 server 是合法 JSON 对象
		var serverObj map[string]interface{}
		if err := json.Unmarshal(cfg.Server, &serverObj); err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgIMServerConfigMustBeJSON))
			return
		}
	}

	// 校验 cred_fields
	credKeySet := make(map[string]bool)
	for _, field := range cfg.CredFields {
		if field.Key == "" || !channelIDRegex.MatchString(field.Key) {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgCredKeyInvalidChars))
			return
		}
		if field.Label == "" {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgCredLabelRequired))
			return
		}
		if credKeySet[field.Key] {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgCredKeyDuplicate))
			return
		}
		credKeySet[field.Key] = true
	}

	// 校验 channel_id 不重复
	var existing model.AIChannel
	if model.DB(r.Context()).Where("channel_id = ?", req.ChannelID).First(&existing).Error == nil {
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgChannelIDExists))
		return
	}

	// 创建通道（默认禁用，需管理员验证后启用）
	channel := model.AIChannel{
		ChannelID:    req.ChannelID,
		Name:         req.Name,
		Enabled:      model.BoolPtr(false),
		Custom:       true,
		CustomConfig: string(req.CustomConfig),
	}
	if err := model.DB(r.Context()).Create(&channel).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgCreateChannelFailed))
		return
	}

	jsonOK(w, map[string]interface{}{"ok": true, "channel": channel})
}

// HandleDeleteChannel 删除自定义通道
func HandleDeleteChannel(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMissingParamID))
		return
	}

	var ch model.AIChannel
	if model.DB(r.Context()).Where("id = ?", id).First(&ch).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgChannelNotExist))
		return
	}

	if !ch.Custom {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgPredefinedChannelCannotDelete))
		return
	}

	model.DB(r.Context()).Unscoped().Delete(&ch)

	jsonOK(w, map[string]interface{}{"ok": true})
}
