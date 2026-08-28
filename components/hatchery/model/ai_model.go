package model

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"

	hcommon "hatchery/common"
	"hatchery/i18n"

	"gorm.io/gorm"
)

// 内置占位记录的常量
const (
	BuiltinModelProvider = "hatchery"
	BuiltinModelID       = "custom"
)

type AIModel struct {
	gorm.Model
	Identifier string `gorm:"index;default:''"`    // 多租户标识，MySQL 模式下自动填充和过滤
	Provider   string `gorm:"not null;default:''"` // 模型提供商名称（用作 openclaw provider key）
	// ModelID 存储用户/运营创建时填入的**原始模型 ID**（保真，支持 "/" 等字符）。
	// 下发到真实 LLM 的 body.model / body.models[].id 等字段直接使用本字段，
	// 以保证上游 API（OpenAI / Anthropic / 腾讯云 TokenPlan 等）识别原始模型名。
	// 当需要作为 openclaw/hermes/ACE 的 providerKey / ref / TAT 参数下发时，
	// 调用方必须使用 SlugifyModelID(ModelID) 得到 shell 安全的 slug 形式。
	ModelID           string `gorm:"not null;default:''"`
	ModelName         string `gorm:"not null;default:''"`           // 模型显示名称，为空时使用 ModelID
	APIKey            string `gorm:"not null;default:''" json:"-"`  // API Key
	URL               string `gorm:"not null;default:''"`           // URL
	ModelType         string `gorm:"not null;default:''"`           // 接口类型：openai-completions / anthropic-messages
	InputTypes        string `gorm:"not null;default:'[\"text\"]'"` // 支持的输入类型，JSON array 或逗号分隔
	ContextLen        int    `gorm:"not null;default:0"`            // 上下文长度
	MaxTokens         int    `gorm:"not null;default:0"`            // 最大输出 Token 数，0=不限
	CustomHTTPHeaders string `gorm:"not null;default:''"`           // 自定义 HTTP 请求头，JSON 对象字符串
	QuotaDay          int    `gorm:"not null;default:-1"`           // 每日Token上限，-1=不限
	// Enabled / Visible 不声明 json tag，由 MarshalJSON 显式接管输出：
	//   - 对外 JSON 中的 `Enabled` 字段输出的是 Visible 的真实值，
	//     用于兼容旧版 React 管控台（client/src/pages/admin/ModelConfig.tsx），
	//     该前端把 `model.Enabled` 当作"用户可见"开关读取；
	//   - 真实的 Enabled（"是否启用 / 开启关闭"）通过额外的 `EnabledStatus` 字段
	//     输出，供新前端读取。字段名与"是否开启/关闭"的业务语义保持一致；
	//   - `Visible` 字段不直接对外输出（MarshalJSON 用 json:"-" 显式隐藏），
	//     避免与 Enabled 字段在 UI 语义上重复，前端统一通过 Enabled 读取
	//     "用户可见"语义、通过 EnabledStatus 读取真实启用状态。
	Enabled bool `gorm:"not null;default:false"` // 是否启用：控制模型是否可用于 LLM 路由（关闭后已绑定的 agent 也无法使用）。新建模型由应用层显式置为 true。
	Visible bool `gorm:"not null;default:false"` // 是否对用户可见：控制用户端模型列表是否展示该模型；不影响存量已绑定 agent 的可用性
	// VisibilityType 与 Visible 的关系：
	//   - Visible=false → 直接对所有用户隐藏
	//   - Visible=true 且 VisibilityType=all → 对所有用户可见
	//   - Visible=true 且 VisibilityType=group → 仅对绑定的分组（含祖先链）可见
	VisibilityType string `gorm:"not null;default:'all'"` // 可见范围类型：all=全部用户, group=按分组
}

// MarshalJSON 自定义 AIModel 的 JSON 序列化输出，目标是同时满足：
//   - 旧版 React 管控台（无前端改动权限）按 `model.Enabled` 读取"用户可见"开关
//     → 输出 `Enabled` 字段时，回填的是真实的 `Visible` 值；
//   - 新前端按 `EnabledStatus` 读取真实的"是否启用（开启/关闭）"状态
//     → 额外输出 `EnabledStatus` 字段，对应真实的 `Enabled`；
//   - 内部 `Visible` 字段不再对外暴露（输出后从 map 中删除），
//     避免与 Enabled 字段在 UI 语义上重复；
//   - 其它字段保持原 Go 默认大写名（ID/Provider/ModelID 等），不影响其他业务调用方。
//
// 实现细节：先借助类型别名 alias（剥离 MarshalJSON 方法以规避无限递归）
// marshal 一遍拿到 base JSON，再 unmarshal 成 map 注入 Enabled / EnabledStatus
// 并删除原始 Visible 键。这种方式相比"匿名结构体内嵌 + json tag 覆盖"更可靠：
// encoding/json 的字段提升 collision 规则要求外层与内层在 **JSON 名** 上同名才
// 能遮蔽，外层声明 `json:"-"` 的字段不参与 collision 判定，无法去掉内嵌字段。
//
// 注意：AIModel 自身不会序列化 APIKey（字段带 json:"-"）；需要展示凭据时，
// controller 层必须显式注入脱敏后的展示字段，禁止输出明文。
// CreatedAt/UpdatedAt/DeletedAt 由 gorm.Model 嵌入，走默认的 JSON 行为。
func (m AIModel) MarshalJSON() ([]byte, error) {
	type alias AIModel // 借助类型别名规避无限递归
	base, err := json.Marshal(alias(m))
	if err != nil {
		return nil, err
	}
	var obj map[string]interface{}
	if err := json.Unmarshal(base, &obj); err != nil {
		return nil, err
	}
	obj["Enabled"] = m.Visible       // 兼容旧前端：值=Visible（"用户可见"开关）
	obj["EnabledStatus"] = m.Enabled // 真实"是否启用（开启/关闭）"状态，供新前端读取
	delete(obj, "Visible")           // 不再对外暴露 Visible 字段，避免与 Enabled 语义重复
	return json.Marshal(obj)
}

// slugifyDashRe 用于合并连续的 "-"，预编译避免每次调用时重复编译。
var slugifyDashRe = regexp.MustCompile(`-+`)

// SlugifyModelID 把 ModelID 归一化为 openclaw providerKey / ref / TAT
// 参数可安全使用的 slug 形式。
//
// 规则（白名单方式）：
//   - 统一转为小写；
//   - 只保留 [a-z0-9._-] 字符，其他字符（包括 "/" ":" 等）一律替换为 "-"；
//   - 合并连续的 "-"；
//   - 去除首尾的 "-"。
//
// 转换示例：
//   - "deepseek-v3.2"           → "deepseek-v3.2"（无变化）
//   - "minimax/minimax-m2.5:free" → "minimax-minimax-m2.5-free"
//   - "GLM-5.1"                 → "glm-5.1"
//
// 该函数不负责校验长度/白名单，调用方已有 ValidateXxx 约束。
func SlugifyModelID(id string) string {
	s := strings.ToLower(id)
	var buf strings.Builder
	buf.Grow(len(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '_' {
			buf.WriteRune(r)
		} else {
			buf.WriteByte('-')
		}
	}
	result := slugifyDashRe.ReplaceAllString(buf.String(), "-")
	return strings.Trim(result, "-")
}

// GetInputTypes parses InputTypes into a string slice.
// Supports both JSON array format (e.g. '["text","image"]') and
// comma-separated format (e.g. 'text,image' or bare 'text').
func (m AIModel) GetInputTypes() []string {
	s := strings.TrimSpace(m.InputTypes)
	if s == "" {
		return []string{"text"}
	}
	if strings.HasPrefix(s, "[") {
		var result []string
		if err := json.Unmarshal([]byte(s), &result); err == nil {
			return result
		}
	}
	// Comma-separated or bare value
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	if len(result) == 0 {
		return []string{"text"}
	}
	return result
}

// GetCustomHTTPHeaders parses CustomHTTPHeaders into a map.
// Returns nil if empty or invalid JSON.
func (m AIModel) GetCustomHTTPHeaders() map[string]string {
	s := strings.TrimSpace(m.CustomHTTPHeaders)
	if s == "" {
		return nil
	}
	var result map[string]string
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		return nil
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// DisplayName returns "Provider/ModelID".
func (m AIModel) DisplayName() string {
	return m.Provider + "/" + m.ModelID
}

// SeedModels ensures the built-in custom model enablement record exists.
// tx 为调用方传入的事务句柄，用于在 InitTenant 流程中串联多个 Seed。
func SeedModels(tx *gorm.DB) error {
	var count int64
	if err := tx.Model(&AIModel{}).Where("provider = ? AND model_id = ?", BuiltinModelProvider, BuiltinModelID).Count(&count).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgAICountBuiltinModelFailed)
	}
	if count == 0 {
		// 内置占位记录的 Enabled 表示"是否允许用户配置自定义模型"（全局开关），
		// 默认关闭，需要管理员主动打开。
		// Visible 表示该占位记录在用户端列表中是否可见，默认 false。
		if err := tx.Create(&AIModel{Provider: BuiltinModelProvider, ModelID: BuiltinModelID, APIKey: "", URL: "", ModelType: "", Enabled: false, Visible: false}).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgAISeedBuiltinModelFailed)
		}
	}
	return nil
}

// IsCustomModelEnabled 查询自定义模型功能是否对用户开放。
//
// hatchery/custom 是一条内置占位记录，不代表真实可用模型，只用于控制
// "是否允许用户配置自定义模型"。其 Enabled 字段不对 LLM 路由产生影响，
// 因此仅需判断 Visible——管理员通过「用户可见」开关统一控制此功能。
func IsCustomModelEnabled(ctx context.Context) bool {
	var m AIModel
	err := DB(ctx).Where("provider = ? AND model_id = ?", BuiltinModelProvider, BuiltinModelID).First(&m).Error
	if err != nil {
		return false
	}
	return m.Visible
}
