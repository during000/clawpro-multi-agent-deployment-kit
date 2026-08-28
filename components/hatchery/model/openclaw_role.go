package model

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"sync"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"

	"gorm.io/gorm"
)

// OpenClawRole 角色预设（管理员维护）
// 使用硬删除（非 gorm.Model），删除后名称可复用。
type OpenClawRole struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	Identifier     string    `gorm:"uniqueIndex:idx_role_name_identifier;index;default:''" json:"-"` // 多租户标识，MySQL 模式下自动填充和过滤
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Name           string    `gorm:"uniqueIndex:idx_role_name_identifier;not null" json:"name"` // 角色名称，业务上限 30 个字（见 controller.maxRoleNameRunes）
	Description    string    `gorm:"type:text;not null" json:"description"`                     // 角色描述
	Soul           string    `gorm:"type:text;not null" json:"soul"`                            // 角色灵魂（System Prompt）
	Visible        bool      `gorm:"not null;default:true" json:"visible"`                      // 是否对员工可见
	SortOrder      int       `gorm:"not null;default:0;index" json:"sort_order"`                // 排序序号，越小越靠前
	VisibilityType string    `gorm:"not null;default:'all'" json:"visibility_type"`             // 可见范围：all=全部用户, group=按分组
	Version        string    `gorm:"type:varchar(16);not null;default:'1.0'" json:"version"`    // 版本号，X.Y 两段式（如 1.0、2.0）
}

// OpenClawRoleSkill 角色关联的技能
type OpenClawRoleSkill struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	Identifier     string    `gorm:"index;default:''" json:"-"` // 多租户标识，MySQL 模式下自动填充和过滤
	CreatedAt      time.Time `json:"created_at"`
	OpenClawRoleID uint      `gorm:"not null;index" json:"openclaw_role_id"`  // 关联角色 ID
	Name           string    `gorm:"not null;default:''" json:"name"`         // 技能名称
	Slug           string    `gorm:"not null;default:''" json:"slug"`         // 技能标识
	Version        string    `gorm:"not null;default:''" json:"version"`      // 版本号
	Source         string    `gorm:"not null;default:'public'" json:"source"` // public / enterprise
	CosZipKey      string    `gorm:"not null;default:''" json:"cos_zip_key"`  // SMH common space 中的 zip 路径
}

// OpenClawRolePlugin 角色关联的插件
type OpenClawRolePlugin struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	Identifier     string    `gorm:"index;default:''" json:"-"`
	CreatedAt      time.Time `json:"created_at"`
	OpenClawRoleID uint      `gorm:"not null;index" json:"openclaw_role_id"`
	Name           string    `gorm:"not null;default:''" json:"name"`
	Slug           string    `gorm:"not null;default:''" json:"slug"`
	PluginID       string    `gorm:"not null;default:''" json:"plugin_id"`
	Version        string    `gorm:"not null;default:''" json:"version"`
	Source         string    `gorm:"not null;default:'enterprise'" json:"source"` // enterprise / npm
	CosZipKey      string    `gorm:"not null;default:''" json:"cos_zip_key"`
	NpmPackage     string    `gorm:"not null;default:''" json:"npm_package"`
	InstallMode    string    `gorm:"not null;default:'smh'" json:"install_mode"` // smh / npm
	Kind           string    `gorm:"not null;default:''" json:"kind"`
}

// ── 预设角色数据 ────────────────────────────────────────────────────

// defaultRoleDef 预设角色定义
type defaultRoleJSON struct {
	Name          string                `json:"name"`
	NameEn        string                `json:"name_en"`
	Description   string                `json:"description"`
	DescriptionEn string                `json:"description_en"`
	Soul          string                `json:"soul"`
	SoulEn        string                `json:"soul_en"`
	Visible       bool                  `json:"visible"`
	Skills        []defaultRoleSkillDef `json:"skills"`
}

type defaultRoleDef struct {
	Name        string
	Description string
	Soul        string
	Visible     bool
	Skills      []defaultRoleSkillDef
}

type defaultRoleSkillDef struct {
	Name    string `json:"name"`
	Slug    string `json:"slug"`
	Version string `json:"version"`
	Source  string `json:"source"`
}

// DefaultRolesJSON 由外部注入的预设角色 JSON 数据。
// release 模式下通过 go:embed 嵌入 config/default_roles.json，
// dev 模式下从磁盘读取 config/default_roles.json。
var DefaultRolesJSON []byte

var (
	defaultRolesJSON []defaultRoleJSON
	defaultRolesOnce sync.Once
)

// getDefaultRoles 懒加载并返回预设角色配置
func getDefaultRoles(ctx context.Context) []defaultRoleDef {
	defaultLang := hcommon.DefaultLangFromCtx(ctx)

	// defaultRolesJSON 仅初始化一次
	defaultRolesOnce.Do(func() {
		data := DefaultRolesJSON
		if len(data) == 0 {
			// 兜底：尝试从磁盘读取
			var err error
			data, err = os.ReadFile("config/default_roles.json")
			if err != nil {
				slog.Error("读取预设角色配置文件失败", "error", err)
				return
			}
		}
		if err := json.Unmarshal(data, &defaultRolesJSON); err != nil {
			slog.Error("解析预设角色配置 JSON 失败", "error", err)
		}
	})

	// 根据语言选择 defaultRolesJSON 中的对应的字段构造 defaultRoles
	defaultRoles := make([]defaultRoleDef, 0, len(defaultRolesJSON))
	for _, j := range defaultRolesJSON {
		if defaultLang == "zh" {
			defaultRoles = append(defaultRoles, defaultRoleDef{
				Name:        j.Name,
				Description: j.Description,
				Soul:        j.Soul,
				Visible:     j.Visible,
				Skills:      j.Skills,
			})
		} else {
			defaultRoles = append(defaultRoles, defaultRoleDef{
				Name:        j.NameEn,
				Description: j.DescriptionEn,
				Soul:        j.SoulEn,
				Visible:     j.Visible,
				Skills:      j.Skills,
			})
		}
	}
	return defaultRoles
}

// ── 种子函数 ────────────────────────────────────────────────────────

// SeedDefaultRoles 初始化预置角色。
// 通过 SiteConfig.DefaultRolesSeeded 标记判断是否已初始化过，
// 一旦标记为 true，即使角色被管理员删除，重启也不会重新创建。
// tx 为调用方传入的事务句柄(通常是 InitTenant 的外层事务)。
// 注意：`syncDefaultRoleSkills` 不再在此处调用，而是由启动阶段作为一次性任务独立执行。
func SeedDefaultRoles(ctx context.Context, tx *gorm.DB) error {
	var config SiteConfig
	if err := tx.First(&config).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgRoleSeedReadSiteConfig)
	}

	if config.DefaultRolesSeeded {
		return nil
	}

	// ── 首次初始化：创建角色和技能 ──
	var count int64
	if err := tx.Model(&OpenClawRole{}).Count(&count).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgRoleSeedCountRoles)
	}
	if count > 0 {
		// 兼容旧数据升级：已有角色则只补标记
		if err := tx.Model(&config).Update("default_roles_seeded", true).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgRoleSeedSetSeededIfExists)
		}
		slog.Info("角色已存在，补设 DefaultRolesSeeded 标记")
		return nil
	}

	roles := getDefaultRoles(ctx)
	for i, def := range roles {
		role := OpenClawRole{
			Name:        def.Name,
			Description: def.Description,
			Soul:        def.Soul,
			Visible:     def.Visible,
			SortOrder:   i,
			Version:     "1.0",
		}
		if err := tx.Create(&role).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgRoleSeedCreateRole, def.Name)
		}

		for _, s := range def.Skills {
			skill := OpenClawRoleSkill{
				OpenClawRoleID: role.ID,
				Name:           s.Name,
				Slug:           s.Slug,
				Version:        s.Version,
				Source:         s.Source,
			}
			if err := tx.Create(&skill).Error; err != nil {
				return hcommon.I18nRichError(err, i18n.MsgRoleSeedCreateSkill, def.Name, s.Slug)
			}
		}
	}

	if err := tx.Model(&config).Update("default_roles_seeded", true).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgRoleSeedUpdateSeededFlag)
	}
	slog.Info("预置角色初始化成功", "count", len(roles))
	return nil
}

// SeedDefaultRoleSkills 根据 defaultRoles 配置，使已有角色的预置技能（source=public）与配置完全一致。
// 每次启动都会执行：补齐缺失的预置技能，删除配置中已移除的预置技能。
// 注意：只管理 source="public" 的技能，不会影响用户自定义添加的技能（如 enterprise 来源）。
func SeedDefaultRoleSkills(ctx context.Context, tx *gorm.DB) error {
	added, removed := 0, 0
	for _, def := range getDefaultRoles(ctx) {
		var role OpenClawRole
		if tx.Where("name = ?", def.Name).First(&role).Error != nil {
			continue // 角色不存在（可能被管理员删除），跳过
		}

		// 构建配置中期望的预置技能集合 slug:version
		expectedSet := make(map[string]defaultRoleSkillDef, len(def.Skills))
		for _, s := range def.Skills {
			src := s.Source
			if src == "" {
				src = "public"
			}
			// 只收集 public 来源的技能作为期望集合
			if src == "public" {
				expectedSet[s.Slug+":"+s.Version] = s
			}
		}

		// 只查询该角色已有的 public 来源技能（不涉及用户自定义技能）
		var existingPublicSkills []OpenClawRoleSkill
		tx.Where("open_claw_role_id = ? AND source = 'public'", role.ID).Find(&existingPublicSkills)
		existingSet := make(map[string]bool, len(existingPublicSkills))
		for _, s := range existingPublicSkills {
			existingSet[s.Slug+":"+s.Version] = true
		}

		// 1. 删除配置中已移除的预置技能（仅 public 来源）
		for _, s := range existingPublicSkills {
			key := s.Slug + ":" + s.Version
			if _, ok := expectedSet[key]; !ok {
				if err := tx.Delete(&s).Error; err != nil {
					slog.Error("删除多余预置技能失败", "role", def.Name, "slug", s.Slug, "error", err)
					continue
				}
				removed++
				slog.Info("删除多余预置技能", "role", def.Name, "slug", s.Slug, "version", s.Version)
			}
		}

		// 2. 补齐缺失的预置技能
		for _, s := range def.Skills {
			src := s.Source
			if src == "" {
				src = "public"
			}
			// 只同步 public 来源的技能
			if src != "public" {
				continue
			}
			key := s.Slug + ":" + s.Version
			if existingSet[key] {
				continue // 已存在，跳过
			}

			skill := OpenClawRoleSkill{
				OpenClawRoleID: role.ID,
				Name:           s.Name,
				Slug:           s.Slug,
				Version:        s.Version,
				Source:         "public",
			}
			if err := tx.Create(&skill).Error; err != nil {
				slog.Error("补齐预置技能失败", "role", def.Name, "slug", s.Slug, "error", err)
				continue
			}
			added++
			slog.Info("补齐预置技能", "role", def.Name, "slug", s.Slug, "version", s.Version)
		}
	}

	if added > 0 || removed > 0 {
		slog.Info("角色预置技能同步完成", "added", added, "removed", removed)
	}
	return nil
}

// MigrateRenamedDefaultRoles 兼容预置角色重命名场景。
// 当 default_roles.json 中某个预置角色被改名时（如 "开发工程师" → "程序员"），
// 老租户库里仍是旧名称记录，由于 SeedDefaultRoles 通过 SiteConfig.DefaultRolesSeeded
// 标记跳过初始化，且 SeedDefaultRoleSkills 按 name 查角色，会导致：
//  1. DB 里仍保留旧角色名，前端展示与配置不一致
//  2. 新名字找不到对应角色，技能升降级被跳过
//
// 本函数把老名称记录原地 rename 为新名称，并刷新 description/soul，
// 后续 SeedDefaultRoleSkills 会按新名找到角色，自动 diff 出 skill 升降级。
//
// 设计原则：
//   - 启动时每次跑（幂等：目标名已存在则直接跳过）
//   - in-place rename，保留 ID/SortOrder/CreatedAt 等所有字段
//   - 不动任何 Instance（不清 soul_set_at），老实例保留旧 Soul，仅新建实例享受新 Soul
//   - 同时兼容国内站（zh）与国际站（en）：旧名同时声明中英两种候选，找到哪个就迁哪个；
//     新名直接从 default_roles.json（getDefaultRoles 解析后）按当前语言取，不在代码里硬编码。
//     将来再改名时只需：① 改 JSON；② 在 renames 表中追加一行旧名映射。
//
// 当前迁移映射：
//
//	zh: "开发工程师"        → 当前 default_roles.json 中 newKey="程序员" 对应的中文 Name
//	en: "Software Engineer" → 当前 default_roles.json 中 newKey="程序员" 对应的英文 NameEn
//
// 来源：tapd story 1020422209135471495
func MigrateRenamedDefaultRoles(ctx context.Context, tx *gorm.DB) error {
	// renames 按"角色身份"声明：
	//   - oldNameZH/oldNameEN：DB 中可能残留的旧名（来自上一版 default_roles.json，已从配置中删除，
	//     只能在代码中保留以完成一次性迁移）
	//   - newKey：用于在当前 default_roles.json 中定位"新角色"的稳定 key，
	//     这里用中文新名作为 key（JSON 里的 zh 名称），匹配后取该角色在当前语言下的实际 Name。
	renames := []struct {
		oldNameZH string
		oldNameEN string
		newKey    string // 对应 default_roles.json 中的中文 name
	}{
		{oldNameZH: "开发工程师", oldNameEN: "Software Engineer", newKey: "程序员"},
	}

	defaultLang := hcommon.DefaultLangFromCtx(ctx)
	roles := getDefaultRoles(ctx)

	// 解析 JSON 原始记录，便于按 newKey（中文 name）定位当前语言下的"新角色名"。
	rolesByZHName := make(map[string]defaultRoleJSON, len(defaultRolesJSON))
	for _, j := range defaultRolesJSON {
		rolesByZHName[j.Name] = j
	}

	for _, r := range renames {
		// 1. 计算"新角色名"：从 default_roles.json 中按 newKey（中文 name）找到对应角色，
		//    再按当前语言取 Name / NameEn。这样保证将来再改名时只改 JSON，无需再改本函数。
		var newName string
		if def, ok := rolesByZHName[r.newKey]; ok {
			if defaultLang == "zh" {
				newName = def.Name
			} else {
				newName = def.NameEn
			}
		}
		if newName == "" {
			// JSON 中已不存在该新角色（理论不应发生），跳过本条 rename
			slog.Warn("迁移所需的新角色未在 default_roles.json 中找到，跳过",
				"newKey", r.newKey, "lang", defaultLang)
			continue
		}

		// 2. 目标名已存在 → 已迁移过，幂等跳过
		var newCnt int64
		if err := tx.Model(&OpenClawRole{}).Where("name = ?", newName).Count(&newCnt).Error; err != nil {
			slog.Error("查询新角色名失败", "newName", newName, "error", err)
			continue
		}
		if newCnt > 0 {
			continue
		}

		// 3. 在 zh/en 两个旧名中任选其一找到老角色 —— 不依赖 defaultLang 判定，
		//    国内站老库只会有 zh 名记录，国际站老库只会有 en 名记录，
		//    遍历两条候选可一次代码兼容两种环境。
		var old OpenClawRole
		var matchedOld string
		for _, candidate := range []string{r.oldNameZH, r.oldNameEN} {
			if candidate == "" {
				continue
			}
			if err := tx.Where("name = ?", candidate).First(&old).Error; err == nil {
				matchedOld = candidate
				break
			}
		}
		if matchedOld == "" {
			// 老角色不存在（全新部署）→ 安全跳过
			continue
		}

		// 4. 同步刷新 description/soul 为 JSON 中当前语言的最新值
		updates := map[string]any{"name": newName}
		for _, def := range roles {
			if def.Name == newName {
				updates["description"] = def.Description
				updates["soul"] = def.Soul
				break
			}
		}

		if err := tx.Model(&old).Updates(updates).Error; err != nil {
			slog.Error("重命名预置角色失败", "from", matchedOld, "to", newName, "error", err)
			continue
		}
		slog.Info("预置角色重命名完成", "from", matchedOld, "to", newName, "role_id", old.ID, "lang", defaultLang)
	}
	return nil
}
