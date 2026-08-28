package model

import (
	"context"
	"encoding/json"

	hcommon "hatchery/common"
	"hatchery/i18n"

	"gorm.io/gorm"
)

type ChannelParam struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

type ChannelSiteScope uint8

const (
	ChannelScopeDomestic ChannelSiteScope = 1 << iota
	ChannelScopeOverseas

	ChannelScopeAll = ChannelScopeDomestic | ChannelScopeOverseas
)

// CustomChannelConfig 表示自定义通道的完整配置信息。
// 序列化为 JSON 后存储在 AIChannel.CustomConfig 字段中。
type CustomChannelConfig struct {
	Server     json.RawMessage `json:"server"`      // IM 服务器配置，原样存储 JSON，不做解析
	CredFields []ChannelParam  `json:"cred_fields"` // 用户凭证字段定义
}

// ServerConfig 表示 CustomChannelConfig.Server 中可被 Go 层识别的语义字段。
// 未识别的字段仍以 json.RawMessage 透传，不做解析。
//
// server 字段本身即作为 openclaw.json 的模板：
//   - {{key}} 占位符会被替换为用户提交的凭证值
//   - 顶层自动补 enabled=true（如果 server 中没有 enabled 字段）
//   - 管理员负责写完整的目标 JSON 结构
type ServerConfig struct {
	PairingMode    *bool  `json:"pairingMode,omitempty"`    // true=配对码模式（如 WhatsApp）—— *bool 区分 false 与缺失
	DeleteFeature  string `json:"deleteFeature,omitempty"`  // 删除通道时的 feature 名（如 "del_whatsapp_channel"）
	AutoFeature    string `json:"autoFeature,omitempty"`    // 自动配置的 feature 名（如 "whatsapp_pairing"）
	AutoTimeout    uint64 `json:"autoTimeout,omitempty"`    // 自动配置超时秒数（如 180）
	DmPolicy       string `json:"dmPolicy,omitempty"`       // DM 策略（如 "allowlist"）
	SelfChatMode   *bool  `json:"selfChatMode,omitempty"`   // 自聊模式（*bool 区分 false 与缺失）
	PhoneRequired  *bool  `json:"phoneRequired,omitempty"`  // 是否需要手机号输入—— *bool 区分 false 与缺失
	PhonePattern   string `json:"phonePattern,omitempty"`   // 手机号正则（缺失时使用内置 whatsappPhoneRegexp）
	EgressRequired *bool  `json:"egressRequired,omitempty"` // 是否需要出站网络访问—— *bool 区分 false 与缺失
}

// ParseServerConfig 解析 server JSON 为结构化配置。
func (c CustomChannelConfig) ParseServerConfig() (ServerConfig, error) {
	var cfg ServerConfig
	if len(c.Server) == 0 {
		return cfg, nil
	}
	err := json.Unmarshal(c.Server, &cfg)
	return cfg, err
}

// channelDefaults 存储已知自定义通道的预设默认值。
// 当管理员未显式配置某个字段时，按 channel_id 匹配此表填充默认值。
var channelDefaults = map[string]ServerConfig{
	"openclaw_whatsapp": {
		PairingMode:    BoolPtr(true),
		AutoFeature:    "whatsapp_pairing",
		AutoTimeout:    180,
		DeleteFeature:  "del_whatsapp_channel",
		DmPolicy:       "allowlist",
		SelfChatMode:   BoolPtr(true),
		PhoneRequired:  BoolPtr(true),
		PhonePattern:   `^[1-9]\d{6,14}$`,
		EgressRequired: BoolPtr(true),
	},
}

// DefaultsForChannel 根据 channel_id 为 ServerConfig 的零值字段填充默认值。
// 显式配置的字段（非零值）保持不变，零值字段按 channelDefaults 表匹配预设默认值。
// 未匹配到预设的 channel_id 时，使用通用默认值兜底。
func (s *ServerConfig) DefaultsForChannel(channelID string) {
	// 查找该 channel_id 的预设默认值
	defaults, hasDefaults := channelDefaults[channelID]

	// 四个 *bool 字段（PairingMode / SelfChatMode / PhoneRequired / EgressRequired）
	// 使用统一的填充规则：
	//  - nil（未配置）时，仅当 channelDefaults 中有预设才用预设值填充
	//  - nil 且无预设：保持 nil，由调用方按"未配置"语义处理
	//  - 非 nil（管理员显式配置）：保持原值，不做任何覆盖
	// 这样 channelDefaults 表的预设能真正起作用，且能区分"未配置"vs"显式 false"。
	if s.PairingMode == nil && hasDefaults {
		s.PairingMode = defaults.PairingMode
	}
	if s.AutoFeature == "" {
		if hasDefaults && defaults.AutoFeature != "" {
			s.AutoFeature = defaults.AutoFeature
		} else {
			s.AutoFeature = channelID + "_pairing"
		}
	}
	if s.DeleteFeature == "" {
		if hasDefaults && defaults.DeleteFeature != "" {
			s.DeleteFeature = defaults.DeleteFeature
		} else {
			s.DeleteFeature = "del_channel" // 通用删除脚本
		}
	}
	if s.AutoTimeout == 0 {
		if hasDefaults && defaults.AutoTimeout > 0 {
			s.AutoTimeout = defaults.AutoTimeout
		} else {
			s.AutoTimeout = 180 // 配对码流程默认超时 180 秒
		}
	}
	if s.DmPolicy == "" {
		if hasDefaults && defaults.DmPolicy != "" {
			s.DmPolicy = defaults.DmPolicy
		} else {
			s.DmPolicy = "allowlist"
		}
	}
	if s.SelfChatMode == nil && hasDefaults {
		s.SelfChatMode = defaults.SelfChatMode
	}
	if s.PhoneRequired == nil && hasDefaults {
		s.PhoneRequired = defaults.PhoneRequired
	}
	if s.PhonePattern == "" {
		if hasDefaults && defaults.PhonePattern != "" {
			s.PhonePattern = defaults.PhonePattern
		} else {
			s.PhonePattern = `^[1-9]\d{6,14}$`
		}
	}
	if s.EgressRequired == nil && hasDefaults {
		s.EgressRequired = defaults.EgressRequired
	}
}

type AIChannel struct {
	gorm.Model
	Identifier     string `gorm:"uniqueIndex:idx_channel_identifier;index;default:''" json:"-"`            // 多租户标识，MySQL 模式下自动填充和过滤
	ChannelID      string `gorm:"uniqueIndex:idx_channel_identifier;not null;default:''" json:"ChannelID"` // 通道标识
	Name           string `gorm:"not null;default:''" json:"Name"`                                         // 显示名称
	Enabled        *bool  `gorm:"not null;default:true" json:"Enabled"`
	Custom         bool   `gorm:"not null;default:false" json:"Custom"`                  // true=自定义通道
	CustomConfig   string `gorm:"type:text;default:''" json:"CustomConfig"`              // 自定义通道配置 JSON
	VisibilityType string `gorm:"size:16;not null;default:'all'" json:"visibility_type"` // 应用范围：'all'=全部用户, 'group'=按组可见
}

// Params returns the parameter definitions for this channel type.
// For predefined channels, params come from the hardcoded ChannelParams map.
// For custom channels, params come from the CredFields in CustomConfig JSON.
func (c AIChannel) Params() []ChannelParam {
	if !c.Custom {
		return ChannelParams[c.ChannelID]
	}
	if c.CustomConfig == "" {
		return nil
	}
	var cfg CustomChannelConfig
	if json.Unmarshal([]byte(c.CustomConfig), &cfg) != nil {
		return nil
	}
	return cfg.CredFields
}

// ParseCustomConfig parses the CustomConfig JSON string into a CustomChannelConfig struct.
func (c AIChannel) ParseCustomConfig() (CustomChannelConfig, error) {
	var cfg CustomChannelConfig
	if c.CustomConfig == "" {
		return cfg, nil
	}
	err := json.Unmarshal([]byte(c.CustomConfig), &cfg)
	return cfg, err
}

var ChannelParams = map[string][]ChannelParam{
	"qqbot": {
		{Key: "app_id", Label: "机器人App ID"},
		{Key: "app_secret", Label: "机器人App Secret"},
	},
	"wecom": {
		{Key: "bot_id", Label: "机器人botId"},
		{Key: "secret", Label: "机器人secret"},
	},
	"feishu": {
		{Key: "app_id", Label: "应用App ID"},
		{Key: "app_secret", Label: "应用App Secret"},
	},
	"lark": {
		{Key: "app_id", Label: "App ID"},
		{Key: "app_secret", Label: "App Secret"},
	},
	"slack": {
		{Key: "app_token", Label: "App-Level Token"},
		{Key: "bot_token", Label: "Bot User OAuth Token"},
	},
	"discord": {
		{Key: "bot_token", Label: "Discord App Bot Token"},
		{Key: "user_id", Label: "Discord User Id"},
	},
	"ddingtalk": {
		{Key: "client_id", Label: "应用Client ID"},
		{Key: "client_secret", Label: "应用Client Secret"},
	},
	"msteams": {
		{Key: "app_id", Label: "Azure App Client ID"},
		{Key: "app_secret", Label: "Azure Client Secret"},
		{Key: "tenant_id", Label: "Azure Tenant ID"},
	},
	"line": {
		{Key: "channel_token", Label: "Channel Access Token"},
		{Key: "channel_secret", Label: "Channel Secret"},
	},
	"wecom_app": {
		{Key: "corp_id", Label: "Corp ID"},
		{Key: "corp_secret", Label: "Corp Secret"},
		{Key: "agent_id", Label: "Agent ID"},
		{Key: "token", Label: "Token"},
		{Key: "encoding_aes_key", Label: "Encoding AES Key"},
	},
	"openclaw-weixin": {},
	"whatsapp":        {},
}

var predefinedChannelSiteScopes = map[string]ChannelSiteScope{
	"openclaw-weixin":    ChannelScopeAll,
	"wecom":              ChannelScopeAll,
	"wecom_app":          ChannelScopeAll,
	"feishu":             ChannelScopeDomestic,
	"lark":               ChannelScopeOverseas,
	"ddingtalk":          ChannelScopeAll,
	"qqbot":              ChannelScopeAll,
	"dingtalk-connector": ChannelScopeAll,
	"msteams":            ChannelScopeAll,
	"line":               ChannelScopeOverseas,
	"slack":              ChannelScopeOverseas,
	"discord":            ChannelScopeOverseas,
	"whatsapp":           ChannelScopeOverseas,
}

type aiChannelDef struct {
	ChannelID string
	Name      string
	NameEn    string
}

// predefinedChannels 的数组顺序即为最终展示顺序，调整顺序只需调整数组元素位置
var predefinedChannels = []aiChannelDef{
	{ChannelID: "openclaw-weixin", Name: "微信", NameEn: "WeChat"},
	{ChannelID: "wecom", Name: "企业微信", NameEn: "WeCom"},
	{ChannelID: "wecom_app", Name: "企业微信应用", NameEn: "WeCom App"},
	{ChannelID: "feishu", Name: "飞书", NameEn: "Feishu"},
	{ChannelID: "ddingtalk", Name: "钉钉", NameEn: "DingTalk"},
	{ChannelID: "msteams", Name: "Microsoft Teams", NameEn: "Microsoft Teams"},
	{ChannelID: "line", Name: "LINE", NameEn: "LINE"},
	{ChannelID: "qqbot", Name: "QQ", NameEn: "QQ"},
	{ChannelID: "slack", Name: "Slack", NameEn: "Slack"},
	{ChannelID: "discord", Name: "Discord", NameEn: "Discord"},
	{ChannelID: "whatsapp", Name: "WhatsApp", NameEn: "WhatsApp"},
	{ChannelID: "lark", Name: "Lark", NameEn: "Lark"},
}

// SortChannelsByPredefined 根据 predefinedChannels 的数组顺序对通道列表排序
// 不在预定义列表中的通道排到最后
func SortChannelsByPredefined(channels []AIChannel) []AIChannel {
	orderMap := make(map[string]int, len(predefinedChannels))
	for i, ch := range predefinedChannels {
		orderMap[ch.ChannelID] = i
	}

	sorted := make([]AIChannel, 0, len(channels))
	// 先按预定义顺序添加
	for _, pre := range predefinedChannels {
		for _, ch := range channels {
			if ch.ChannelID == pre.ChannelID {
				sorted = append(sorted, ch)
				break
			}
		}
	}
	// 再追加不在预定义列表中的通道
	for _, ch := range channels {
		if _, ok := orderMap[ch.ChannelID]; !ok {
			sorted = append(sorted, ch)
		}
	}
	return sorted
}

func ChannelSiteScopeFor(channelID string) (ChannelSiteScope, bool) {
	v, ok := predefinedChannelSiteScopes[channelID]
	return v, ok
}

func ChannelInSiteScope(channelID string, isOverseas bool) bool {
	scope, ok := ChannelSiteScopeFor(channelID)
	if !ok {
		return true
	}
	if isOverseas {
		return scope&ChannelScopeOverseas != 0
	}
	return scope&ChannelScopeDomestic != 0
}

func FilterChannelsBySiteScope(channels []AIChannel, isOverseas bool) []AIChannel {
	out := make([]AIChannel, 0, len(channels))
	for _, ch := range channels {
		if ChannelInSiteScope(ch.ChannelID, isOverseas) {
			out = append(out, ch)
		}
	}
	return out
}

func FilterChannelIDsBySiteScope(channelIDs []string, isOverseas bool) []string {
	out := make([]string, 0, len(channelIDs))
	for _, channelID := range channelIDs {
		if ChannelInSiteScope(channelID, isOverseas) {
			out = append(out, channelID)
		}
	}
	return out
}

// BoolPtr 返回 bool 值的指针，用于 GORM *bool 字段赋值
func BoolPtr(b bool) *bool {
	return &b
}

// defaultDisabledChannels 默认关闭的预定义通道集合
var defaultDisabledChannels = map[string]bool{
	"wecom_app": true,
}

// SeedChannels 初始化预定义通道。
// tx 为调用方传入的事务句柄(或 *gorm.DB 自身)，使得该函数可复用到 InitTenant 多租户场景。
func SeedChannels(ctx context.Context, tx *gorm.DB) error {
	defaultLang := hcommon.DefaultLangFromCtx(ctx)

	for _, ch := range predefinedChannels {
		var existing AIChannel
		if tx.Where("channel_id = ?", ch.ChannelID).First(&existing).Error != nil {
			enabled := !defaultDisabledChannels[ch.ChannelID]
			var channel *AIChannel
			if defaultLang == "zh" {
				channel = &AIChannel{
					ChannelID: ch.ChannelID,
					Name:      ch.Name,
					Enabled:   BoolPtr(enabled),
					Custom:    false,
				}
			} else {
				channel = &AIChannel{
					ChannelID: ch.ChannelID,
					Name:      ch.NameEn,
					Enabled:   BoolPtr(enabled),
					Custom:    false,
				}
			}
			if err := tx.Create(channel).Error; err != nil {
				return hcommon.I18nRichError(err, i18n.MsgSeedChannelFailed, ch.ChannelID)
			}
		}
	}
	return nil
}
