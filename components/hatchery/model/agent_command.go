package model

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	hcommon "hatchery/common"
	"hatchery/i18n"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"
)

// ============================================================================
// 常量
// ============================================================================

// AgentCommandType 命令类型枚举。
const (
	AgentCommandTypeShell = "SHELL"
)

// AgentCommandVisibility 可见性枚举（决策 Q11 预留扩展点）。
const (
	AgentCommandVisibilityTenant = "tenant"
)

// AgentCommandSlugPrefix 命令模板 slug 前缀，与 identifier 的 clp- 风格对齐。
const AgentCommandSlugPrefix = "cmd-"

// AgentCommand* 各类硬约束。
//
// 长度上限统一按 **Unicode 字符数（rune）** 计，与产品 spec / 前端输入框 maxLength 对齐。
// MySQL utf8mb4 `varchar(N)` 同样按字符计；DDL 已经匹配。
const (
	MaxAgentCommandsPerTenant   = 500  // 决策 R-NEW-4：单租户活跃命令上限
	AgentCommandNameMaxChars    = 60   // 决策 R-NEW-3
	AgentCommandDescMaxChars    = 512  // 描述上限
	AgentCommandContentMaxChars = 8192 // 命令正文上限
	AgentCommandTimeoutMin      = uint(1)
	AgentCommandTimeoutMax      = uint(86400) // 1 天，覆盖长跑任务（如全量数据迁移、大文件压缩）
	AgentCommandTimeoutDefault  = uint(60)
	AgentCommandRunUserMaxChars = 64  // 与 agent_commands.run_user varchar(64) 对齐
	AgentCommandWorkdirMaxChars = 255 // 与 agent_commands.workdir varchar(255) 对齐
	AgentCommandParamsMax       = 10  // 决策 R-NEW-2：单命令参数上限
	AgentCommandParamNameMin    = 1
	AgentCommandParamNameMax    = 32
	AgentCommandParamDefaultMax = 128 // params[].default 字符上限
	AgentCommandParamDescMax    = 200
	AgentCommandSlugRandLen     = 8 // "cmd-{8 位随机}"
)

// agentCommandSlugCharset slug 随机字符集（小写字母 + 数字）。
var agentCommandSlugCharset = []byte("abcdefghijklmnopqrstuvwxyz0123456789")

// agentCommandNamePattern 命令名称合法字符：中文/英文/数字/下划线/-/.（决策 R-NEW-3）。
// 注意：长度上限按字符数（rune）单独校验，见 AgentCommandNameMaxChars。
var agentCommandNamePattern = regexp.MustCompile(`^[\p{Han}A-Za-z0-9_\-.]+$`)

// agentCommandParamNamePattern 参数名 shell 标识符规则。
var agentCommandParamNamePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// ============================================================================
// 错误
// ============================================================================

var (
	ErrAgentCommandNotFound            = hcommon.I18nError(i18n.MsgCommandNotFound)
	ErrAgentCommandQuotaExceeded       = hcommon.I18nError(i18n.MsgAgentCmdQuotaExceeded)
	ErrAgentCommandSlugConflict        = hcommon.I18nError(i18n.MsgAgentCmdSlugConflict)
	ErrAgentCommandNameInvalid         = hcommon.I18nError(i18n.MsgAgentCmdNameInvalidChars)
	ErrAgentCommandNameTooLong         = hcommon.I18nError(i18n.MsgAgentCmdNameTooLong)
	ErrAgentCommandDescTooLong         = hcommon.I18nError(i18n.MsgAgentCmdDescriptionTooLong)
	ErrAgentCommandContentEmpty        = hcommon.I18nError(i18n.MsgAgentCmdContentRequired)
	ErrAgentCommandContentTooLong      = hcommon.I18nError(i18n.MsgAgentCmdContentTooLong)
	ErrAgentCommandTimeoutOOR          = hcommon.I18nError(i18n.MsgAgentCmdTimeoutOutOfRange)
	ErrAgentCommandRunUserTooLong      = hcommon.I18nError(i18n.MsgAgentCmdRunUserTooLong)
	ErrAgentCommandWorkdirTooLong      = hcommon.I18nError(i18n.MsgAgentCmdWorkdirTooLong)
	ErrAgentCommandTypeInvalid         = hcommon.I18nError(i18n.MsgAgentCmdInvalidType)
	ErrAgentCommandParamsTooMany       = hcommon.I18nError(i18n.MsgAgentCmdParamsTooMany)
	ErrAgentCommandParamNameInvalid    = hcommon.I18nError(i18n.MsgAgentCmdParamNameInvalid)
	ErrAgentCommandParamNameDup        = hcommon.I18nError(i18n.MsgAgentCmdParamNameDuplicated)
	ErrAgentCommandParamDescTooLong    = hcommon.I18nError(i18n.MsgAgentCmdParamDescriptionTooLong)
	ErrAgentCommandParamDefaultTooLong = hcommon.I18nError(i18n.MsgAgentCmdParamDefaultTooLong)
)

// ============================================================================
// 模型
// ============================================================================

// AgentCommandParam 命令模板参数定义项，用于 ParamsJSON 反序列化与 API 响应。
type AgentCommandParam struct {
	Name        string `json:"name"`
	Default     string `json:"default"`
	Description string `json:"description"`
}

// AgentCommand 命令模板。
//
// 存储约定：Content 为 raw 原文 UTF-8 文本，base64 仅在 controller/tat.go 调用 TAT
// SDK 前完成，参见 openspec/changes/agent-command-execution/design.md §8.1。
//
// 多租户：依赖项目 GORM identifier 回调自动注入 WHERE / 写入 identifier 字段。
//
// 软删除：使用 gorm.DeletedAt，DELETE 端点只把 deleted_at 置为 NOW()，保留 fk
// 引用以便历史 invocation/task 反查。配额仅计 deleted_at IS NULL 的活跃行。
type AgentCommand struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Identifier      string `gorm:"uniqueIndex:idx_agent_command_ident_slug,priority:1;index;default:''" json:"-"`
	Slug            string `gorm:"uniqueIndex:idx_agent_command_ident_slug,priority:2;type:varchar(32);not null;default:''" json:"slug"`
	Name            string `gorm:"type:varchar(60);not null;default:''" json:"name"`
	Description     string `gorm:"type:varchar(512);not null;default:''" json:"description"`
	Type            string `gorm:"type:varchar(16);not null;default:'SHELL'" json:"type"`
	Content         string `gorm:"type:text;not null" json:"content"`
	TimeoutSec      uint   `gorm:"not null;default:60" json:"timeout_sec"`
	RunUser         string `gorm:"type:varchar(64);not null;default:'root'" json:"run_user"`
	Workdir         string `gorm:"type:varchar(255);not null;default:'/root'" json:"workdir"`
	ParamsJSON      string `gorm:"type:varchar(8192);not null;default:'[]'" json:"-"`
	VisibilityType  string `gorm:"type:varchar(16);not null;default:'tenant'" json:"-"`
	CreatedByUserID uint   `gorm:"not null;default:0;index" json:"created_by_user_id"`
}

// TableName 固定表名（避免 GORM 自动复数化 agent_commands → agent_commands，反正一致，但显式声明更清晰）。
func (AgentCommand) TableName() string { return "agent_commands" }

// Params 把 ParamsJSON 反序列化为 []AgentCommandParam。
// 错误时返回空切片（容错；上层校验已经保证写入时合法，读取时极少出错）。
func (c *AgentCommand) Params() []AgentCommandParam {
	if strings.TrimSpace(c.ParamsJSON) == "" {
		return nil
	}
	var params []AgentCommandParam
	if err := json.Unmarshal([]byte(c.ParamsJSON), &params); err != nil {
		return nil
	}
	return params
}

// SetParams 把 []AgentCommandParam 序列化写回 ParamsJSON。
func (c *AgentCommand) SetParams(params []AgentCommandParam) error {
	if params == nil {
		params = []AgentCommandParam{}
	}
	b, err := json.Marshal(params)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgAgentCmdMarshalParamsFailed)
	}
	c.ParamsJSON = string(b)
	return nil
}

// ============================================================================
// 校验
// ============================================================================

// ValidateAgentCommandName 校验命令名称（决策 R-NEW-3）。
//
// 长度按 Unicode 字符数计：1 个汉字 = 1 字符。
func ValidateAgentCommandName(name string) error {
	if name == "" {
		return ErrAgentCommandNameInvalid
	}
	if utf8.RuneCountInString(name) > AgentCommandNameMaxChars {
		return ErrAgentCommandNameTooLong
	}
	if !agentCommandNamePattern.MatchString(name) {
		return ErrAgentCommandNameInvalid
	}
	return nil
}

// ValidateAgentCommandContent 校验命令正文。
//
// 长度按 Unicode 字符数计；内容含中文（如 echo 中文）按字符算 1 个。
func ValidateAgentCommandContent(content string) error {
	if strings.TrimSpace(content) == "" {
		return ErrAgentCommandContentEmpty
	}
	if utf8.RuneCountInString(content) > AgentCommandContentMaxChars {
		return ErrAgentCommandContentTooLong
	}
	return nil
}

// ValidateAgentCommandDescription 校验命令描述长度。空字符串视为合法。
//
// 长度按 Unicode 字符数计。超过上限直接报错（不再静默截断），让调用方明确感知。
func ValidateAgentCommandDescription(desc string) error {
	if utf8.RuneCountInString(desc) > AgentCommandDescMaxChars {
		return ErrAgentCommandDescTooLong
	}
	return nil
}

// ValidateAgentCommandType 校验命令类型枚举。
func ValidateAgentCommandType(t string) error {
	switch t {
	case AgentCommandTypeShell:
		return nil
	}
	return ErrAgentCommandTypeInvalid
}

// ValidateAgentCommandTimeout 校验超时秒数范围。
func ValidateAgentCommandTimeout(sec uint) error {
	if sec < AgentCommandTimeoutMin || sec > AgentCommandTimeoutMax {
		return ErrAgentCommandTimeoutOOR
	}
	return nil
}

// ValidateAgentCommandRunUser 校验执行用户名长度。空字符串视为合法（调用方会回落 default "root"）。
//
// 长度按 Unicode 字符数计；与 agent_commands.run_user varchar(64) 列宽对齐。
func ValidateAgentCommandRunUser(s string) error {
	if utf8.RuneCountInString(s) > AgentCommandRunUserMaxChars {
		return ErrAgentCommandRunUserTooLong
	}
	return nil
}

// ValidateAgentCommandWorkdir 校验工作目录长度。空字符串视为合法（调用方会回落 default "/root"）。
//
// 长度按 Unicode 字符数计；与 agent_commands.workdir varchar(255) 列宽对齐。
func ValidateAgentCommandWorkdir(s string) error {
	if utf8.RuneCountInString(s) > AgentCommandWorkdirMaxChars {
		return ErrAgentCommandWorkdirTooLong
	}
	return nil
}

// ValidateAgentCommandParams 校验参数列表（数量、name 格式、name 重名、description 长度）。
// 返回 (offendingName, err)，其中 offendingName 在 name / description 相关错误时给出具体冲突项，便于错误响应携带细节。
//
// 不做任何 in-place 修改：超长 description 直接 ErrAgentCommandParamDescTooLong，
// 调用方应当报 400 让用户感知。
func ValidateAgentCommandParams(params []AgentCommandParam) (string, error) {
	if len(params) > AgentCommandParamsMax {
		return "", ErrAgentCommandParamsTooMany
	}
	seen := make(map[string]struct{}, len(params))
	for i := range params {
		p := &params[i]
		// param.Name 受 agentCommandParamNamePattern 限制为 ASCII 标识符，
		// byte 长度 == 字符长度，无需 RuneCountInString。
		if l := len(p.Name); l < AgentCommandParamNameMin || l > AgentCommandParamNameMax {
			return p.Name, ErrAgentCommandParamNameInvalid
		}
		if !agentCommandParamNamePattern.MatchString(p.Name) {
			return p.Name, ErrAgentCommandParamNameInvalid
		}
		if _, dup := seen[p.Name]; dup {
			return p.Name, ErrAgentCommandParamNameDup
		}
		seen[p.Name] = struct{}{}
		if utf8.RuneCountInString(p.Default) > AgentCommandParamDefaultMax {
			return p.Name, ErrAgentCommandParamDefaultTooLong
		}
		if utf8.RuneCountInString(p.Description) > AgentCommandParamDescMax {
			return p.Name, ErrAgentCommandParamDescTooLong
		}
	}
	return "", nil
}

// ============================================================================
// slug 生成
// ============================================================================

// GenerateAgentCommandSlug 生成 "cmd-{8 位随机}" slug。字符集 [a-z0-9]，不暴露给前端冲突感知。
func GenerateAgentCommandSlug() string {
	return AgentCommandSlugPrefix + randomLowerAlnum(AgentCommandSlugRandLen)
}

// TruncateRunes 按 Unicode 字符数（rune）截断字符串，绝不在 UTF-8 字节中间切断。
//
// 适用于「描述类字段超长静默截断」语义。如果 s 长度 ≤ maxRunes 直接返回原串。
func TruncateRunes(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	count := 0
	for i := range s {
		if count == maxRunes {
			return s[:i]
		}
		count++
	}
	return s
}

// randomLowerAlnum 生成 n 位小写字母+数字的随机串，使用 crypto/rand 保证均匀分布。
func randomLowerAlnum(n int) string {
	if n <= 0 {
		return ""
	}
	buf := make([]byte, n)
	r := make([]byte, n)
	if _, err := rand.Read(r); err != nil {
		// rand.Read 在受支持平台上几乎不会失败；万一失败 fallback 到时间戳避免阻断业务。
		ts := time.Now().UnixNano()
		for i := range buf {
			buf[i] = agentCommandSlugCharset[(ts>>uint(i*4))&0x1f%int64(len(agentCommandSlugCharset))]
		}
		return string(buf)
	}
	for i, b := range r {
		buf[i] = agentCommandSlugCharset[int(b)%len(agentCommandSlugCharset)]
	}
	return string(buf)
}

// ============================================================================
// CRUD helper
// ============================================================================

// CountActiveAgentCommands 统计当前租户活跃命令数（不含软删）。用于配额校验。
func CountActiveAgentCommands(ctx context.Context) (int64, error) {
	var n int64
	if err := DB(ctx).Model(&AgentCommand{}).Count(&n).Error; err != nil {
		return 0, hcommon.I18nRichError(err, i18n.MsgCmdFailedToCountActiveCmds)
	}
	return n, nil
}

// FindAgentCommandByID 按主键查找当前租户内的命令；软删行不返回。
func FindAgentCommandByID(ctx context.Context, id uint) (*AgentCommand, error) {
	var c AgentCommand
	err := DB(ctx).Where("id = ?", id).First(&c).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAgentCommandNotFound
		}
		return nil, hcommon.I18nRichError(err, i18n.MsgAgentCmdFindByIDFailed)
	}
	return &c, nil
}

// FindAgentCommandBySlug 按 slug 查找当前租户内的命令（含软删时通过 Unscoped 选项控制）。
//   - includeDeleted=true：用于 slug 冲突检测，包含已软删的行（避免随机串被软删行占用）。
//   - includeDeleted=false：API 层正常查询，仅返回活跃行。
func FindAgentCommandBySlug(ctx context.Context, slug string, includeDeleted bool) (*AgentCommand, error) {
	q := DB(ctx)
	if includeDeleted {
		q = q.Unscoped()
	}
	var c AgentCommand
	err := q.Where("slug = ?", slug).First(&c).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAgentCommandNotFound
		}
		return nil, hcommon.I18nRichError(err, i18n.MsgAgentCmdFindBySlugFailed)
	}
	return &c, nil
}

// CreateAgentCommandWithSlugRetry 在租户内为命令生成不冲突的 slug 并落库。
//
// 策略：在事务外先随机生成 → 通过 Unscoped 检查（含软删行）→ 不冲突即 Create → 冲突重试。
// retries 控制最大重试次数；超过仍冲突返回 ErrAgentCommandSlugConflict（理论 8 字符 36^8≈2.8 万亿空间，
// 单租户 500 上限下，碰撞概率近似 0；retries 主要兜底极端并发竞态）。
func CreateAgentCommandWithSlugRetry(ctx context.Context, cmd *AgentCommand, retries int) error {
	if retries <= 0 {
		retries = 5
	}
	for i := 0; i < retries; i++ {
		cmd.Slug = GenerateAgentCommandSlug()
		// Unscoped 检查避免软删行占用同 slug
		exists, err := agentCommandSlugExists(ctx, cmd.Slug)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		if err := DB(ctx).Create(cmd).Error; err != nil {
			// MySQL/SQLite 都可能返回 unique violation，直接走重试
			if isUniqueConflict(err) {
				continue
			}
			return hcommon.I18nRichError(err, i18n.MsgAgentCmdCreateFailed)
		}
		return nil
	}
	return ErrAgentCommandSlugConflict
}

// agentCommandSlugExists 通过 Unscoped 查询同租户内 slug 是否已被任何行（含软删）占用。
func agentCommandSlugExists(ctx context.Context, slug string) (bool, error) {
	var n int64
	if err := DB(ctx).Unscoped().Model(&AgentCommand{}).
		Where("slug = ?", slug).Count(&n).Error; err != nil {
		return false, hcommon.I18nRichError(err, i18n.MsgAgentCmdSlugCheckFailed)
	}
	return n > 0, nil
}

// isUniqueConflict 粗粒度判断 GORM 错误是否是 UNIQUE 索引冲突。
// 不依赖具体驱动错误码（SQLite "UNIQUE constraint failed" / MySQL "Duplicate entry"），
// 用关键字匹配兼容两端。
func isUniqueConflict(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") || strings.Contains(msg, "duplicate")
}
