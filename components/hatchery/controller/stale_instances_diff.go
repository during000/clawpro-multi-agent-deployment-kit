package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"hatchery/controller/usergroup"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tchttp "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/http"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

// stale_instances_diff.go — 配置差异比对
//
// 输出形态：响应里 target 配置和 instance 配置**分离**：
//   - 顶层 target_config.categories[]   — 目标分组的行集合（仅一份，所有实例共享）
//   - 顶层 instance_configs[].categories[] — 每个实例独立一份，仅含 instance_values + status
// 行通过共享的 key 字段一一对应（前端按 key join）。
//
// 语义：左侧 instance_values 的取值分三档：
//
// A. 实例级覆盖（读 Instance 行字段）——
//    - chargeType ← InstanceChargeType
//    - model      ← AIModelID（==0 视为未配置，instance_values 为空集）
//    - skill 角色 ← RoleID（==0 视为未配置）
//    - imageType  ← AgentType
//    - network    ← VpcId / SubnetId / SecurityGroupId + CVMInstanceInfo 公网三字段
//
// B. 跟随 inst.group_id 组配置（含继承链）——
//    - platformPolicy
//    此类实例上没有 override 字段，instance_values 用 inst.GroupID 解析后的组视图。
//
// C. 实例本身数据（不跟随组视图）——
//    - memory ← model.MemoryTDAIPlugin（按 CVM InstanceId）
//    - drive  ← model.SMHPersonalSpace（按 DB inst.ID）
//    - cls    ← inst.CLSAgentStatus 字段
//    - aiAgentSecurity ← CWP DescribeMachines ProtectType
//
// D. 空集——
//    - channel / agentTool / skill 的"初始技能包"/"技能安装来源"子项
//
// 行展示策略按 category 分类：
//   single_row     — 整个 category 一行（model / channel / memory / drive / cls / aiAgentSecurity / imageType）
//   by_sub_label   — 按 entry.SubLabel 拆多行（skill / agentTool / network）
//   by_entry       — 每个 entry 一行（platformPolicy，因每条策略是独立配置项）
//
// 子行/entry 的顺序以 target（after）为主，让 target_config 与 instance_configs 行 key 顺序对齐。

// configDiffRequest /admin/stale-instances/config-diff 请求体。
//
// TargetGroupID 用 *uint 区分"未传"和"传 0"两种语义：
//   - nil      → 未传，handler 返回 400
//   - 非 nil 0 → 显式指定未分组（全局默认配置视角）
//   - 非 nil >0 → 目标分组 ID
type configDiffRequest struct {
	InstanceIDs   []uint `json:"instance_ids"`
	TargetGroupID *uint  `json:"target_group_id"`
}

// configDiffValue 单个值的展示信息。多值类目里的一个项；单值类目（如 memory）就是 1 个 value。
// Status 仅在 instance_values 里出现（omitempty），target values 不填。
type configDiffValue struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Value  string `json:"value"`
	Status string `json:"status,omitempty"`
}

// targetConfigRow 目标分组维度的一行（值来自目标组解析视图）。
// 所有实例共享同一份 target_config，避免批量场景下重复序列化。
type targetConfigRow struct {
	Key            string            `json:"key"`                       // 行唯一 key（如 "model" / "skill.0" / "platformPolicy.instance_quota"）
	CategoryKey    string            `json:"category_key"`              // 所属 category
	CategoryLabel  string            `json:"category_label"`            // category 显示名
	SubLabel       string            `json:"sub_label,omitempty"`       // 子标签
	PolicyCategory string            `json:"policy_category,omitempty"` // T29：platformPolicy 行独有——分类（user_quota / model_quota / feature_toggle）
	Values         []configDiffValue `json:"values"`                    // 目标分组该行的取值集合
}

// instanceConfigRow 单个实例维度的一行。通过 key 与 targetConfigRow 一一对应。
type instanceConfigRow struct {
	Key            string            `json:"key"`
	CategoryKey    string            `json:"category_key"`
	CategoryLabel  string            `json:"category_label"`
	SubLabel       string            `json:"sub_label,omitempty"`
	PolicyCategory string            `json:"policy_category,omitempty"` // T29：platformPolicy 行独有
	InstanceValues []configDiffValue `json:"instance_values"`           // 实例自身存储的取值集合（每项含 status）
	Status         string            `json:"status"`                    // same / contained_in_target / different / not_check
}

// configDiffRow 内部使用：buildRowsForCategory 仍以"双侧合并"形式产出，
// 由 computeConfigDiff / buildTargetConfig 拆成两类行。
type configDiffRow struct {
	Key            string
	CategoryKey    string
	CategoryLabel  string
	SubLabel       string
	PolicyCategory string
	InstanceValues []configDiffValue
	TargetValues   []configDiffValue
	Status         string
}

// configDiffPerInstance 单个实例的结果（仅 instance 侧 + status，target 在外层共享）。
type configDiffPerInstance struct {
	InstanceID       uint                `json:"instance_id"`
	CurrentGroupID   uint                `json:"current_group_id"`
	CurrentGroupPath string              `json:"current_group_path,omitempty"`
	Categories       []instanceConfigRow `json:"categories"`
}

// targetConfigPayload 顶层 target_config 字段。
type targetConfigPayload struct {
	Categories []targetConfigRow `json:"categories"`
}

// 行展示策略
type diffRowStrategy int

const (
	stratSingleRow        diffRowStrategy = iota // 整个 category 一行
	stratBySubLabel                              // 按 SubLabel 拆多行
	stratByEntry                                 // 每个 entry 一行
	stratByPolicyCategory                        // 平台策略专用：按 policy_category 拆 3 行（用户配额 / 模型配额 / 功能权限开关）
)

var diffStrategyByCategory = map[string]diffRowStrategy{
	usergroup.CategoryKeyChargeType:      stratSingleRow,
	usergroup.CategoryKeyModel:           stratSingleRow,
	usergroup.CategoryKeyChannel:         stratSingleRow,
	usergroup.CategoryKeySkill:           stratBySubLabel,
	usergroup.CategoryKeyAgentTool:       stratBySubLabel,
	usergroup.CategoryKeyMemory:          stratSingleRow,
	usergroup.CategoryKeyDrive:           stratSingleRow,
	usergroup.CategoryKeyImageType:       stratSingleRow,
	usergroup.CategoryKeyNetwork:         stratBySubLabel,
	usergroup.CategoryKeyCLS:             stratSingleRow,
	usergroup.CategoryKeyAIAgentSecurity: stratSingleRow,
	usergroup.CategoryKeyPlatformPolicy:  stratByPolicyCategory,
}

// computeConfigDiff 计算单个实例的差异行。
//
// 调用方应预先调用 buildCategoriesForView 一次得到 targetCats、
// loadInstanceLookups 一次得到 lookups，再遍历每个实例复用，避免批量场景下
// 重复构建目标分组视图与 per-instance DB 查询。
//
// cvmInfo 用于填充实例侧公网 3 子行（T28）；批量场景下由 handler 一次性
// batchFetchCVMInfoMap 拿到 map 后按 InstanceId 索引传入。为 nil 时公网子行不参与对比
// （其他 category 正常输出）。
//
// siteConfig 用于解析实例所属组的 1 类"跟随组"category（platformPolicy）。为 nil 时这类走空集回退（等同旧行为）。
//
// groupCache 可选缓存实例所属组的 1 类 entries（避免每实例都查一次组视图）；
// 传 nil 时按需构建。
func computeConfigDiff(ctx context.Context, inst *model.Instance, targetCats []usergroup.ConfigCategoryResult, lookups instanceLookups, cvmInfo *CVMInstanceInfo, siteConfig *model.SiteConfig, groupCache instanceGroupInheritedCache, memoryPluginMap map[string]*model.MemoryTDAIPlugin, driveSpaceMap map[uint]bool, cwpSecurityMap map[string]string) configDiffPerInstance {
	beforeCats := buildInstanceCategoriesView(ctx, inst, lookups, cvmInfo, siteConfig, groupCache, memoryPluginMap, driveSpaceMap, cwpSecurityMap)

	beforeIdx := indexCategoriesByKey(beforeCats)
	afterIdx := indexCategoriesByKey(targetCats)

	rows := make([]instanceConfigRow, 0, 16)
	for _, meta := range usergroup.ConfigCategoryList {
		for _, r := range buildRowsForCategory(ctx, meta, beforeIdx[meta.Key], afterIdx[meta.Key], inst) {
			rows = append(rows, instanceConfigRow{
				Key:            r.Key,
				CategoryKey:    r.CategoryKey,
				CategoryLabel:  r.CategoryLabel,
				SubLabel:       r.SubLabel,
				PolicyCategory: r.PolicyCategory,
				InstanceValues: attachValueStatuses(r.Status, r.InstanceValues, r.TargetValues),
				Status:         r.Status,
			})
		}
	}
	return configDiffPerInstance{
		InstanceID:       inst.ID,
		CurrentGroupID:   inst.GroupID,
		CurrentGroupPath: groupFullPath(ctx, inst.GroupID),
		Categories:       rows,
	}
}

// buildTargetConfig 从目标组视图构建顶层 target_config。
// 走 buildRowsForCategory 用一个空 instance 触发同样的行展开逻辑，保证 target_config
// 的 row keys 与 instance_configs[].categories 中的 row keys 完全一致（前端 join 用 key）。
func buildTargetConfig(ctx context.Context, targetCats []usergroup.ConfigCategoryResult) targetConfigPayload {
	emptyInst := &model.Instance{}
	// 空实例 → AIModelID/RoleID 都为 0，lookups 不会被读到，传 nil map 即可
	// cvmInfo 传 nil：target_config 只关心 target 侧，实例侧的公网子行来自 target_config 的 sub_label 保序即可
	// siteConfig / groupCache 都传 nil：target_config 只需要 row 骨架，不需要解析 emptyInst.GroupID 的组视图
	emptyCats := buildInstanceCategoriesView(ctx, emptyInst, instanceLookups{}, nil, nil, nil, nil, nil, nil)
	beforeIdx := indexCategoriesByKey(emptyCats)
	afterIdx := indexCategoriesByKey(targetCats)

	rows := make([]targetConfigRow, 0, 16)
	for _, meta := range usergroup.ConfigCategoryList {
		for _, r := range buildRowsForCategory(ctx, meta, beforeIdx[meta.Key], afterIdx[meta.Key], emptyInst) {
			rows = append(rows, targetConfigRow{
				Key:            r.Key,
				CategoryKey:    r.CategoryKey,
				CategoryLabel:  r.CategoryLabel,
				SubLabel:       r.SubLabel,
				PolicyCategory: r.PolicyCategory,
				Values:         r.TargetValues,
			})
		}
	}
	return targetConfigPayload{Categories: rows}
}

// buildCategoriesForView 给定 groupID（0 = 未分组视角），返回 11 个 category 的 overview。
// config-diff 专用：公网部分使用 3 子行格式（公网 IP / 计费模式 / 带宽上限），
// 与 instance_configs 中的格式保持一致。config-overview 不走此路径，仍用单条 public-ip + meta。
func buildCategoriesForView(ctx context.Context, groupID uint, siteConfig *model.SiteConfig) []usergroup.ConfigCategoryResult {
	var ancestors []uint
	if groupID != 0 {
		if ancs, err := model.ClosureAncestors(ctx, groupID, true); err == nil {
			ancestors = ancs
		} else {
			slog.Warn("[ConfigDiff] ClosureAncestors failed for target group, policies will fall back to site defaults",
				"group_id", groupID, "err", err)
		}
		if len(ancestors) == 0 {
			slog.Warn("[ConfigDiff] empty ancestor chain for target group, policies will fall back to site defaults",
				"group_id", groupID)
		}
	}
	cats := buildCategoriesForGroup(ctx, groupID, ancestors, siteConfig, nil)

	// 把网络 category 中的公网从单条 public-ip + meta 转为 3 子行格式
	for i := range cats {
		if cats[i].Key != usergroup.CategoryKeyNetwork {
			continue
		}
		cats[i].Entries = expandPublicIPToSubRows(ctx, cats[i].Entries, siteConfig)
	}
	return cats
}

// expandPublicIPToSubRows 把网络 entries 中的单条 public-ip + meta 拆为 3 子行
// （公网 IP / 计费模式 / 带宽上限），其余 entries 保持不变。
func expandPublicIPToSubRows(ctx context.Context, entries []usergroup.ConfigEntry, cfg *model.SiteConfig) []usergroup.ConfigEntry {
	out := make([]usergroup.ConfigEntry, 0, len(entries)+2)
	globalSource := usergroup.Source{Type: usergroup.SourceGlobal}
	for _, e := range entries {
		if e.ID == "public-ip" {
			// 从 meta 中提取 InternetAccessible 信息，用 targetInternetEntries 重建 3 子行
			if cfg != nil {
				if overview, err := model.ParseCVMTemplateOverview(cfg.CVMTemplate); err == nil && overview != nil && overview.InternetAccessible != nil {
					out = append(out, targetInternetEntries(ctx, overview.InternetAccessible, globalSource)...)
					continue
				}
			}
			// 降级：解析不了就保留原条目
		}
		out = append(out, e)
	}
	return out
}

// instanceLookups 批量场景下预查的实例级覆盖字段查询结果，避免每个实例独立查 DB。
//   - ModelNames:  AIModel.id → 展示名（model_name 优先，回退 model_id）
//   - RoleNames:   OpenClawRole.id → 角色名
type instanceLookups struct {
	ModelNames map[uint]string
	RoleNames  map[uint]string
}

// loadInstanceLookups 在循环外做一次批量查询，把所有实例用到的 AIModelID/RoleID 一次拉到 map。
// 命中 0 个 ID 时跳过相应 SQL；最坏 2 条 SQL（不论实例数量）。
func loadInstanceLookups(ctx context.Context, instances []*model.Instance) instanceLookups {
	modelIDSet := make(map[uint]struct{})
	roleIDSet := make(map[uint]struct{})
	for _, inst := range instances {
		if inst == nil {
			continue
		}
		if inst.AIModelID != 0 {
			modelIDSet[inst.AIModelID] = struct{}{}
		}
		if inst.RoleID != 0 {
			roleIDSet[inst.RoleID] = struct{}{}
		}
	}

	out := instanceLookups{
		ModelNames: make(map[uint]string, len(modelIDSet)),
		RoleNames:  make(map[uint]string, len(roleIDSet)),
	}
	if len(modelIDSet) > 0 {
		ids := make([]uint, 0, len(modelIDSet))
		for id := range modelIDSet {
			ids = append(ids, id)
		}
		var rows []struct {
			ID   uint   `gorm:"column:id"`
			Name string `gorm:"column:name"`
		}
		_ = model.DB(ctx).Model(&model.AIModel{}).
			Select("id, CASE WHEN model_name != '' THEN model_name ELSE model_id END as name").
			Where("id IN ?", ids).
			Find(&rows).Error
		for _, r := range rows {
			out.ModelNames[r.ID] = r.Name
		}
	}
	if len(roleIDSet) > 0 {
		ids := make([]uint, 0, len(roleIDSet))
		for id := range roleIDSet {
			ids = append(ids, id)
		}
		var rows []model.OpenClawRole
		_ = model.DB(ctx).Select("id, name").Where("id IN ?", ids).Find(&rows).Error
		for _, r := range rows {
			out.RoleNames[r.ID] = r.Name
		}
	}
	return out
}

// instanceGroupInheritedCategories 1 类"实例侧跟随组配置（含继承链）"的 category。
// memory / drive / cls / aiAgentSecurity 已改为读实例本身数据，不再跟随组视图。
var instanceGroupInheritedCategories = map[string]bool{
	usergroup.CategoryKeyPlatformPolicy: true,
}

// instanceGroupInheritedCache 按 group_id 缓存 5 类跟随组 category 的 entries。
// 批量场景（enrich / config-diff 请求）预先按 unique group_id 构建一次，
// 避免 buildInstanceCategoriesView 里每个实例都触发一次 buildCategoriesForGroup。
type instanceGroupInheritedCache map[uint]map[string][]usergroup.ConfigEntry

// buildInstanceGroupInheritedCacheFromTargetCats 当"实例组视图 == 目标组视图"时
// （典型场景：/admin/instances 的 config-drift 检查），从已缓存的 targetCatsByGroup
// 直接过滤出 5 类 category，无须重复调 buildCategoriesForGroup。
func buildInstanceGroupInheritedCacheFromTargetCats(targetCatsByGroup map[uint][]usergroup.ConfigCategoryResult) instanceGroupInheritedCache {
	if len(targetCatsByGroup) == 0 {
		return nil
	}
	out := make(instanceGroupInheritedCache, len(targetCatsByGroup))
	for gid, cats := range targetCatsByGroup {
		byKey := make(map[string][]usergroup.ConfigEntry, len(instanceGroupInheritedCategories))
		for _, c := range cats {
			if instanceGroupInheritedCategories[c.Key] {
				byKey[c.Key] = c.Entries
			}
		}
		out[gid] = byKey
	}
	return out
}

// buildInstanceCategoriesView 按左侧语义构建 diff 的实例侧输入。
//
// 三档取值（详见文件头注释）：
//   A. 实例级覆盖：chargeType / model / skill 角色 / imageType / network（含公网子行）
//   B. 跟随组视图（含继承链）：platformPolicy / memory / drive / cls / aiAgentSecurity
//   C. 空集：channel / agentTool / skill 非角色子项
//
// lookups 由调用方通过 loadInstanceLookups 预先批量查得；批量场景下
// 不会触发任何 per-instance SQL。
//
// cvmInfo 用于产出网络 category 的公网 3 子行（T28）；nil 时不输出公网子行。
// siteConfig 用于 B 档 category 解析；nil 时 B 档退化为空集（等同旧行为，主要给
// buildTargetConfig 的 emptyInst 骨架路径使用）。
// groupCache 若命中 inst.GroupID 直接复用；未命中或为 nil 时按需查一次
// buildCategoriesForGroup。批量场景 caller 应传缓存以避免 N 次组视图构建。
// memoryPluginMap / driveSpaceMap / cwpSecurityMap 用于实例本身 C 档数据；传 nil 时对应 category 输出空集。
func buildInstanceCategoriesView(ctx context.Context, inst *model.Instance, lookups instanceLookups, cvmInfo *CVMInstanceInfo, siteConfig *model.SiteConfig, groupCache instanceGroupInheritedCache, memoryPluginMap map[string]*model.MemoryTDAIPlugin, driveSpaceMap map[uint]bool, cwpSecurityMap map[string]string) []usergroup.ConfigCategoryResult {
	// B 档：解析 inst.GroupID 的 5 类跟随组配置（含继承链）。
	// 优先命中缓存；未命中则按需构建（非空 siteConfig 前提下）。
	var groupInherited map[string][]usergroup.ConfigEntry
	if inst != nil {
		if cached, ok := groupCache[inst.GroupID]; ok {
			groupInherited = cached
		} else if siteConfig != nil {
			var ancestors []uint
			if inst.GroupID != 0 {
				if ancs, err := model.ClosureAncestors(ctx, inst.GroupID, true); err == nil {
					ancestors = ancs
				} else {
					slog.Warn("[ConfigDiff] ClosureAncestors failed for instance group, policies will fall back to site defaults",
						"instance_id", inst.ID, "group_id", inst.GroupID, "err", err)
				}
				if len(ancestors) == 0 {
					slog.Warn("[ConfigDiff] empty ancestor chain for instance group, policies will fall back to site defaults",
						"instance_id", inst.ID, "group_id", inst.GroupID)
				}
			}
			cats := buildCategoriesForGroup(ctx, inst.GroupID, ancestors, siteConfig, instanceGroupInheritedCategories)
			groupInherited = make(map[string][]usergroup.ConfigEntry, len(cats))
			for _, c := range cats {
				groupInherited[c.Key] = c.Entries
			}
		}
	}

	cats := make([]usergroup.ConfigCategoryResult, 0, len(usergroup.ConfigCategoryList))
	for _, meta := range usergroup.ConfigCategoryList {
		cat := usergroup.ConfigCategoryResult{
			Key:         meta.Key,
			Label:       meta.Label,
			Description: meta.Description,
			Icon:        meta.Icon,
			Entries:     []usergroup.ConfigEntry{},
		}
		switch meta.Key {
		case usergroup.CategoryKeyChargeType:
			cat.Entries = instanceChargeTypeEntries(ctx, inst)
		case usergroup.CategoryKeyModel:
			cat.Entries = instanceModelEntries(inst, lookups.ModelNames)
		case usergroup.CategoryKeySkill:
			cat.Entries = instanceSkillEntries(ctx, inst, lookups.RoleNames)
		case usergroup.CategoryKeyImageType:
			cat.Entries = instanceImageTypeEntries(inst)
		case usergroup.CategoryKeyNetwork:
			cat.Entries = instanceNetworkEntries(ctx, inst, cvmInfo)
		case usergroup.CategoryKeyMemory:
			if inst != nil {
				plugin := (*model.MemoryTDAIPlugin)(nil)
				if memoryPluginMap != nil {
					plugin = memoryPluginMap[inst.InstanceId]
				}
				enabled := plugin != nil && plugin.Status == model.MemoryTDAIPluginStatusEnabled
				plan := model.MemoryPlanOff
				if plugin != nil {
					plan = plugin.CurrentPlan
				}
				label := i18n.MsgGroupTreeLabelOff
				if enabled {
					if plan == model.MemoryPlanPro {
						label = i18n.MsgGroupTreeMemoryProEdition
					} else {
						label = i18n.MsgGroupTreeMemoryFreeEdition
					}
				}
				cat.Entries = []usergroup.ConfigEntry{{
					ID:     "tdai",
					Label:  i18n.T(ctx, label),
					Source: usergroup.Source{Type: usergroup.SourceGlobal},
					Meta:   map[string]interface{}{"enabled": enabled, "plan": plan},
				}}
			}
		case usergroup.CategoryKeyDrive:
			if inst != nil {
				enabled := driveSpaceMap != nil && driveSpaceMap[inst.ID]
				label := i18n.MsgGroupTreeLabelOff
				if enabled {
					label = i18n.MsgGroupTreeLabelOn
				}
				cat.Entries = []usergroup.ConfigEntry{{
					ID:     "smh",
					Label:  i18n.T(ctx, label),
					Source: usergroup.Source{Type: usergroup.SourceGlobal},
					Meta:   map[string]interface{}{"enabled": enabled},
				}}
			}
		case usergroup.CategoryKeyCLS:
			if inst != nil {
				enabled := inst.CLSAgentStatus == 1
				label := i18n.MsgGroupTreeLabelOff
				if enabled {
					label = i18n.MsgGroupTreeLabelOn
				}
				cat.Entries = []usergroup.ConfigEntry{{
					ID:     "cls",
					Label:  i18n.T(ctx, label),
					Source: usergroup.Source{Type: usergroup.SourceGlobal},
					Meta:   map[string]interface{}{"enabled": enabled, "scope_type": "instance"},
				}}
			}
		case usergroup.CategoryKeyAIAgentSecurity:
			if inst != nil {
				flagship := cwpSecurityMap != nil && cwpSecurityMap[inst.InstanceId] == "Flagship 旗舰版"
				protectLevel := 0
				if flagship {
					protectLevel = 2
				}
				label := i18n.MsgGroupTreeAIAgentSecurityBasic
				if flagship {
					label = i18n.MsgGroupTreeAIAgentSecurityFlagship
				}
				cat.Entries = []usergroup.ConfigEntry{{
					ID:     "aiAgentSecurity",
					Label:  i18n.T(ctx, label),
					Source: usergroup.Source{Type: usergroup.SourceGlobal},
					Meta:   map[string]interface{}{"protect_level": protectLevel},
				}}
			}
		default:
			// B 档跟随组视图；C 档 groupInherited 命中不到自然回退空集
			if entries, ok := groupInherited[meta.Key]; ok {
				cat.Entries = entries
			}
		}
		cats = append(cats, cat)
	}
	return cats
}

// instanceModelEntries 实例 model 覆盖。AIModelID==0 → 空集；
// 否则从批量预查结果 modelNames 中取展示名（命中不到 → 空集）。
func instanceModelEntries(inst *model.Instance, modelNames map[uint]string) []usergroup.ConfigEntry {
	if inst.AIModelID == 0 {
		return nil
	}
	name, ok := modelNames[inst.AIModelID]
	if !ok {
		return nil
	}
	return makeInstanceModelEntries(inst.AIModelID, name)
}

func makeInstanceModelEntries(aiModelID uint, name string) []usergroup.ConfigEntry {
	if aiModelID == 0 {
		return nil
	}
	return []usergroup.ConfigEntry{{
		ID:    strconv.FormatUint(uint64(aiModelID), 10),
		Label: name,
	}}
}

// instanceSkillEntries 实例 skill 覆盖。仅"角色"子标签有实例字段 RoleID；
// "初始技能包"/"技能安装来源" 没有实例字段，instance 侧不输出。
// 角色名从批量预查结果 roleNames 中取（命中不到 → 空集）。
func instanceSkillEntries(ctx context.Context, inst *model.Instance, roleNames map[uint]string) []usergroup.ConfigEntry {
	if inst.RoleID == 0 {
		return nil
	}
	name, ok := roleNames[inst.RoleID]
	if !ok {
		return nil
	}
	return makeInstanceSkillEntries(inst.RoleID, name, i18n.T(ctx, i18n.MsgGroupTreeSubLabelRole))
}

func makeInstanceSkillEntries(roleID uint, roleName, roleSubLabel string) []usergroup.ConfigEntry {
	if roleID == 0 {
		return nil
	}
	return []usergroup.ConfigEntry{{
		ID:       strconv.FormatUint(uint64(roleID), 10),
		Label:    roleName,
		SubLabel: roleSubLabel,
	}}
}

// chargeTypeDisplayKeys 计费模式代码到 i18n 键的映射（T25，i18n 化）。
// 未识别的原始值原样返回，避免响应缺失。
var chargeTypeDisplayKeys = map[string]i18n.Key{
	"PREPAID":          i18n.MsgInstanceChargeTypePrepaidLabel,
	"POSTPAID_BY_HOUR": i18n.MsgInstanceChargeTypePostpaidByHourLabel,
}

// chargeTypeDisplayName 把计费模式代码通过 i18n 翻译（未知值兜底为原字符串）。
func chargeTypeDisplayName(ctx context.Context, code string) string {
	if key, ok := chargeTypeDisplayKeys[code]; ok {
		return i18n.T(ctx, key)
	}
	return code
}

// internetChargeTypeDisplayKeys 公网计费类型代码到 i18n 键的映射（T28，i18n 化）。
var internetChargeTypeDisplayKeys = map[string]i18n.Key{
	"BANDWIDTH_POSTPAID_BY_HOUR": i18n.MsgInternetChargeBandwidthPostpaidLabel,
	"TRAFFIC_POSTPAID_BY_HOUR":   i18n.MsgInternetChargeTrafficPostpaidLabel,
	"BANDWIDTH_PACKAGE":          i18n.MsgInternetChargeBandwidthPackageLabel,
	"BANDWIDTH_PREPAID":          i18n.MsgInternetChargeBandwidthPrepaidLabel,
}

// internetChargeTypeDisplayName 把公网计费类型代码通过 i18n 翻译（未知或空值 → "-"）。
// 破折号 "-" 是 UI 通用占位符，无需 i18n。
func internetChargeTypeDisplayName(ctx context.Context, code string) string {
	if code == "" {
		return "-"
	}
	if key, ok := internetChargeTypeDisplayKeys[code]; ok {
		return i18n.T(ctx, key)
	}
	return code
}

// publicIPAssignedLabel 把"是否分配公网 IP" bool 值 i18n 化为 已分配/未分配（用于目标组视图）。
func publicIPAssignedLabel(ctx context.Context, assigned bool) string {
	if assigned {
		return i18n.T(ctx, i18n.MsgLabelAllocated)
	}
	return i18n.T(ctx, i18n.MsgLabelNotAllocated)
}

// publicIPAssignedFromInstance 根据实例真实公网 IP 反推"是否分配"的 i18n 标签（用于实例侧）。
// 语义与 publicIPAssignedLabel 对齐（同一 i18n 键），确保两侧同值时能算 same。
func publicIPAssignedFromInstance(ctx context.Context, publicIP string) string {
	if publicIP != "" {
		return i18n.T(ctx, i18n.MsgLabelAllocated)
	}
	return i18n.T(ctx, i18n.MsgLabelNotAllocated)
}

// bandwidthDisplayName 把带宽上限（Mbps）格式化为 "N Mbps"；0 用 "0 Mbps"（不留空，方便集合比较）。
// Mbps 是国际通用带宽单位，不做 i18n。
func bandwidthDisplayName(mbps int64) string {
	return strconv.FormatInt(mbps, 10) + " Mbps"
}

// instanceChargeTypeEntries 实例 chargeType 覆盖：读 Instance.InstanceChargeType（T25）。
// 空值 → 空集，让 diff 保持 "same"（组侧空则一致）。
func instanceChargeTypeEntries(ctx context.Context, inst *model.Instance) []usergroup.ConfigEntry {
	if inst.InstanceChargeType == "" {
		return nil
	}
	return []usergroup.ConfigEntry{{
		ID:    inst.InstanceChargeType,
		Label: chargeTypeDisplayName(ctx, inst.InstanceChargeType),
	}}
}

// instanceImageTypeEntries 实例 imageType 覆盖（AgentType 字段，正常情况下非空）。
func instanceImageTypeEntries(inst *model.Instance) []usergroup.ConfigEntry {
	if inst.AgentType == "" {
		return nil
	}
	displayName := inst.AgentType
	if name, ok := model.AgentTypeDisplayNames[inst.AgentType]; ok {
		displayName = name
	}
	return []usergroup.ConfigEntry{{
		ID:    inst.AgentType,
		Label: displayName,
	}}
}

// instanceNetworkEntries 实例 network 覆盖：
//   - 私有网络与子网：1 条 vpc + 1 条 subnet（空字符串显示为"自动分配"，与组视图保持一致）
//   - 安全组：0 或 1 条
//   - 公网：cvmInfo 非 nil 时输出 3 条子行（公网 IP / 计费模式 / 带宽上限，T28），否则跳过
func instanceNetworkEntries(ctx context.Context, inst *model.Instance, cvmInfo *CVMInstanceInfo) []usergroup.ConfigEntry {
	autoAssignLabel := i18n.T(ctx, i18n.MsgGroupTreeAutoAssign)
	subVPC := i18n.T(ctx, i18n.MsgGroupTreeSubLabelVpcSubnet)
	subSG := i18n.T(ctx, i18n.MsgGroupTreeSubLabelSecurityGroup)
	entries := makeInstanceNetworkEntries(inst.VpcId, inst.SubnetId, inst.SecurityGroupId, autoAssignLabel, subVPC, subSG)
	if cvmInfo != nil {
		entries = append(entries, instanceInternetEntries(ctx, cvmInfo)...)
	}
	return entries
}

// instanceInternetEntries 实例侧公网三子项（T28）—— 全部归属 SubLabel="公网"，
// 与 targetInternetEntries 对齐，让 buildSubLabelRows 合并成一行做集合比较。
// cvmInfo 从 CVM API 现拉，本函数为纯格式化（走 i18n 化的 label）。
func instanceInternetEntries(ctx context.Context, cvmInfo *CVMInstanceInfo) []usergroup.ConfigEntry {
	subInternet := i18n.T(ctx, i18n.MsgGroupTreeSubLabelInternet)
	idIP := i18n.T(ctx, i18n.MsgGroupTreeSubLabelInternetPublicIP)
	idCT := i18n.T(ctx, i18n.MsgGroupTreeSubLabelInternetChargeType)
	idBW := i18n.T(ctx, i18n.MsgGroupTreeSubLabelInternetBandwidth)
	ipLabel := publicIPAssignedFromInstance(ctx, cvmInfo.PublicIP)
	chargeLabel := internetChargeTypeDisplayName(ctx, cvmInfo.InternetChargeType)
	bwLabel := bandwidthDisplayName(cvmInfo.InternetMaxBandwidthOut)
	return []usergroup.ConfigEntry{
		{ID: idIP, Label: ipLabel, SubLabel: subInternet, NameHint: idIP},
		{ID: idCT, Label: chargeLabel, SubLabel: subInternet, NameHint: idCT},
		{ID: idBW, Label: bwLabel, SubLabel: subInternet, NameHint: idBW},
	}
}

func makeInstanceNetworkEntries(vpcID, subnetID, sgID, autoAssignLabel, subVPC, subSG string) []usergroup.ConfigEntry {
	entries := make([]usergroup.ConfigEntry, 0, 3)

	vpcDisplay := vpcID
	if vpcDisplay == "" {
		vpcDisplay = autoAssignLabel
	}
	entries = append(entries, usergroup.ConfigEntry{ID: vpcDisplay, Label: vpcDisplay, SubLabel: subVPC})

	subnetDisplay := subnetID
	if subnetDisplay == "" {
		subnetDisplay = autoAssignLabel
	}
	entries = append(entries, usergroup.ConfigEntry{ID: subnetDisplay, Label: subnetDisplay, SubLabel: subVPC})

	if sgID != "" {
		entries = append(entries, usergroup.ConfigEntry{ID: sgID, Label: sgID, SubLabel: subSG})
	}
	return entries
}

func indexCategoriesByKey(cats []usergroup.ConfigCategoryResult) map[string]*usergroup.ConfigCategoryResult {
	m := make(map[string]*usergroup.ConfigCategoryResult, len(cats))
	for i := range cats {
		m[cats[i].Key] = &cats[i]
	}
	return m
}

// buildRowsForCategory 按策略把一个 category 的 entries 切成 rows。
func buildRowsForCategory(ctx context.Context, meta usergroup.ConfigCategoryMeta, before, after *usergroup.ConfigCategoryResult, inst *model.Instance) []configDiffRow {
	strat, ok := diffStrategyByCategory[meta.Key]
	if !ok {
		strat = stratSingleRow
	}
	switch strat {
	case stratSingleRow:
		return []configDiffRow{buildSingleRow(meta, before, after, inst)}
	case stratBySubLabel:
		return buildSubLabelRows(ctx, meta, before, after, inst)
	case stratByEntry:
		return buildPolicyEntryRows(ctx, meta, before, after)
	case stratByPolicyCategory:
		return buildPolicyByCategoryRows(ctx, meta, before, after)
	}
	return nil
}

// buildSingleRow 整个 category 一行。
func buildSingleRow(meta usergroup.ConfigCategoryMeta, before, after *usergroup.ConfigCategoryResult, inst *model.Instance) configDiffRow {
	instanceValues := entriesToValues(getEntries(before))
	targetValues := entriesToValues(getEntries(after))
	row := configDiffRow{
		Key:            meta.Key,
		CategoryKey:    meta.Key,
		CategoryLabel:  meta.Label,
		InstanceValues: instanceValues,
		TargetValues:   targetValues,
	}
	row.Status = computeRowStatus(instanceValues, targetValues)
	return row
}

// statusNotCheck 表示"该行不参与对比"（T26 / T27）。
// - T26（skill 角色/安装来源、agentTool 企业技能/企业插件）：两侧 values 均清空
// - T27（网络私有网络与子网 + 目标组自动分配）：仅 target 清空，instance 原样返回
const statusNotCheck = "not_check"

// isNotCheckSubLabel 判断某个 (category, sub_label) 组合是否属于 T26 定义的"两侧都清空"集合：
//   - skill    → ""（角色经 diffSubLabelTransform 映射后） / 技能安装来源
//   - agentTool → 企业插件 / 企业MCP（企业技能已由 diffSubLabelTransform 跳过）
//
// sub_label 用 i18n 解析，避免硬编码中文导致多语言场景失效。
func isNotCheckSubLabel(ctx context.Context, categoryKey, subLabel string) bool {
	switch categoryKey {
	case usergroup.CategoryKeySkill:
		return subLabel == "" ||
			subLabel == i18n.T(ctx, i18n.MsgGroupTreeSubLabelSkillSource)
	case usergroup.CategoryKeyAgentTool:
		return subLabel == i18n.T(ctx, i18n.MsgGroupTreeSubLabelEnterprisePlugin) ||
			subLabel == i18n.T(ctx, i18n.MsgGroupTreeSubLabelEnterpriseMCP)
	}
	return false
}

// diffSubLabelTransform 对特定 (category, sub_label) 做重映射或跳过：
//   - skill/"初始技能包" → skip=true（整行不输出）
//   - skill/"角色"       → 重映射为 ""（not_check，但仍出现在输出中）
//   - agentTool/"企业技能" → skip=true（整行不输出）
//   - 其他              → 原样返回
func diffSubLabelTransform(ctx context.Context, categoryKey, subLabel string) (string, bool) {
	switch categoryKey {
	case usergroup.CategoryKeySkill:
		if subLabel == i18n.T(ctx, i18n.MsgGroupTreeSubLabelInitialSkillBundle) {
			return "", true
		}
		if subLabel == i18n.T(ctx, i18n.MsgGroupTreeSubLabelRole) {
			return "", false
		}
	case usergroup.CategoryKeyAgentTool:
		if subLabel == i18n.T(ctx, i18n.MsgGroupTreeSubLabelEnterpriseSkill) {
			return "", true
		}
	}
	return subLabel, false
}

// isTargetVpcAutoAssign 判断目标组的"私有网络与子网"配置是否等价于"自动分配"（T27）。
//
// 语义：目标 entries 全部为空、或每一项 ID 都等于 autoAssignLabel（VPC + Subnet 都是"自动分配"）
// 时视为自动分配 → 该行返回 not_check。
func isTargetVpcAutoAssign(entries []usergroup.ConfigEntry, autoAssignLabel string) bool {
	if len(entries) == 0 {
		return true
	}
	for _, e := range entries {
		if e.ID != autoAssignLabel {
			return false
		}
	}
	return true
}

// buildSubLabelRows 按 SubLabel 拆多行（skill / agentTool / network）。
func buildSubLabelRows(ctx context.Context, meta usergroup.ConfigCategoryMeta, before, after *usergroup.ConfigCategoryResult, inst *model.Instance) []configDiffRow {
	subLabels := collectSubLabelsInOrder(before, after)
	// agentTool：企业插件 / 企业MCP 两行始终出现（均为 not_check），
	// 已在列表中的按原顺序保留，缺失的追加到末尾。
	if meta.Key == usergroup.CategoryKeyAgentTool {
		subLabels = forceAgentToolSubLabels(ctx, subLabels)
	}
	rows := make([]configDiffRow, 0, len(subLabels))
	autoAssignLabel := i18n.T(ctx, i18n.MsgGroupTreeAutoAssign)
	subVPC := i18n.T(ctx, i18n.MsgGroupTreeSubLabelVpcSubnet)
	outputIdx := 0
	for _, sub := range subLabels {
		newSub, skip := diffSubLabelTransform(ctx, meta.Key, sub)
		if skip {
			continue
		}
		instEntries := filterEntriesBySubLabel(getEntries(before), sub)
		tgtEntries := filterEntriesBySubLabel(getEntries(after), sub)
		instanceValues := entriesToValues(instEntries)
		targetValues := entriesToValues(tgtEntries)

		// T26：命中不检查集合（skill 角色/安装来源、agentTool 企业插件）→
		//      两侧 values 都清空，status=not_check
		// T27：网络私有网络与子网 + 目标组"自动分配" →
		//      仅 target 清空，instance 原样返回（管理员仍能看到实例真实 VPC/Subnet），
		//      status=not_check
		row := configDiffRow{
			Key:           fmt.Sprintf("%s.%d", meta.Key, outputIdx),
			CategoryKey:   meta.Key,
			CategoryLabel: meta.Label,
			SubLabel:      newSub,
		}
		outputIdx++
		switch {
		case isNotCheckSubLabel(ctx, meta.Key, newSub):
			// T26：两侧都清空
			row.InstanceValues = []configDiffValue{}
			row.TargetValues = []configDiffValue{}
			row.Status = statusNotCheck
		case meta.Key == usergroup.CategoryKeyNetwork && newSub == subVPC && isTargetVpcAutoAssign(tgtEntries, autoAssignLabel):
			// T27：只清 target，instance 原样返回
			row.InstanceValues = instanceValues
			row.TargetValues = []configDiffValue{}
			row.Status = statusNotCheck
		default:
			row.InstanceValues = instanceValues
			row.TargetValues = targetValues
			row.Status = computeRowStatus(instanceValues, targetValues)
		}
		rows = append(rows, row)
	}
	return rows
}

// policyCategoryOrder 平台策略分类的固定展示顺序：用户配额 → 模型配额 → 功能权限开关。
var policyCategoryOrder = []string{
	usergroup.PolicyCategoryUserQuota,
	usergroup.PolicyCategoryModelQuota,
	usergroup.PolicyCategoryFeatureToggle,
}

// policyCategorySubLabel 分类枚举 → 中文 sub_label（i18n 化）。
func policyCategorySubLabel(ctx context.Context, category string) string {
	switch category {
	case usergroup.PolicyCategoryUserQuota:
		return i18n.T(ctx, i18n.MsgGroupTreeSubLabelUserQuota)
	case usergroup.PolicyCategoryModelQuota:
		return i18n.T(ctx, i18n.MsgGroupTreeSubLabelModelQuota)
	case usergroup.PolicyCategoryFeatureToggle:
		return i18n.T(ctx, i18n.MsgGroupTreeSubLabelFeatureToggle)
	}
	return category
}

// buildPolicyByCategoryRows 平台策略按 policy_category 分组，产出 3 行（T29 合并版）。
//
// 每行 SubLabel 为分类中文名（用户配额 / 模型配额 / 功能权限开关）；
// InstanceValues / TargetValues 为该分类下所有 policy 的 (ID=policy_key, Name=formatted_value) 集合。
// computeRowStatus 做集合比较：分类内所有 policy 都相同 → same，任一不同 → different + highlight_keys 含不同 policy 的 ID。
//
// 行 key 格式：`platformPolicy.<category>`（如 `platformPolicy.user_quota`）。
func buildPolicyByCategoryRows(ctx context.Context, meta usergroup.ConfigCategoryMeta, before, after *usergroup.ConfigCategoryResult) []configDiffRow {
	groupByCategory := func(entries []usergroup.ConfigEntry) map[string][]configDiffValue {
		out := make(map[string][]configDiffValue, len(policyCategoryOrder))
		for _, e := range entries {
			// 跳过 legacy _day key（只提取整数，custom 模式会丢失信息 → 误显"无限制"）
			// 保留 _rules key（含完整规则 JSON），在 policyEntryToValue 中格式化为可读字符串
			if e.ID == usergroup.PolicyKeyTokenQuotaDay || e.ID == usergroup.PolicyKeyGlobalTokenQuotaDay {
				continue
			}
			def, ok := usergroup.PolicyDefs[e.ID]
			if !ok || def.Category == "" {
				continue
			}
			out[def.Category] = append(out[def.Category], policyEntryToValue(ctx, e))
		}
		return out
	}
	beforeByCat := groupByCategory(getEntries(before))
	afterByCat := groupByCategory(getEntries(after))

	rows := make([]configDiffRow, 0, len(policyCategoryOrder))
	for _, category := range policyCategoryOrder {
		instanceValues := beforeByCat[category]
		if instanceValues == nil {
			instanceValues = []configDiffValue{}
		}
		targetValues := afterByCat[category]
		if targetValues == nil {
			targetValues = []configDiffValue{}
		}
		row := configDiffRow{
			Key:            meta.Key + "." + category,
			CategoryKey:    meta.Key,
			CategoryLabel:  meta.Label,
			SubLabel:       policyCategorySubLabel(ctx, category),
			PolicyCategory: category,
			InstanceValues: instanceValues,
			TargetValues:   targetValues,
		}
		row.Status = computeRowStatus(instanceValues, targetValues)
		rows = append(rows, row)
	}
	return rows
}

// buildPolicyEntryRows 每个 platform_policy entry 一行。
//
// policy entry 的 entry.Meta 通常含 {"value": <实际值>}，比对时把值放到 configDiffValue.Name
// 让 set 比较直接看到值不同。
func buildPolicyEntryRows(ctx context.Context, meta usergroup.ConfigCategoryMeta, before, after *usergroup.ConfigCategoryResult) []configDiffRow {
	beforeMap := make(map[string]usergroup.ConfigEntry)
	afterMap := make(map[string]usergroup.ConfigEntry)
	for _, e := range getEntries(before) {
		beforeMap[e.ID] = e
	}
	for _, e := range getEntries(after) {
		afterMap[e.ID] = e
	}
	// 保序：先 after（target）出现顺序，再 before 中独有的（让 row key 顺序与 target_config 对齐）
	seen := make(map[string]bool)
	ordered := make([]string, 0, len(beforeMap)+len(afterMap))
	for _, e := range getEntries(after) {
		if !seen[e.ID] {
			seen[e.ID] = true
			ordered = append(ordered, e.ID)
		}
	}
	for _, e := range getEntries(before) {
		if !seen[e.ID] {
			seen[e.ID] = true
			ordered = append(ordered, e.ID)
		}
	}
	rows := make([]configDiffRow, 0, len(ordered))
	for _, id := range ordered {
		// 显式初始化为非 nil 空切片，避免一侧缺失时 JSON 序列化为 null（与其他策略保持一致输出 []）。
		beforeVal := []configDiffValue{}
		afterVal := []configDiffValue{}
		var subLabel string
		if e, ok := beforeMap[id]; ok {
			beforeVal = []configDiffValue{policyEntryToValue(ctx, e)}
			subLabel = e.Label
		}
		if e, ok := afterMap[id]; ok {
			afterVal = []configDiffValue{policyEntryToValue(ctx, e)}
			if subLabel == "" {
				subLabel = e.Label
			}
		}
		row := configDiffRow{
			Key:            meta.Key + "." + id,
			CategoryKey:    meta.Key,
			CategoryLabel:  meta.Label,
			SubLabel:       subLabel,
			PolicyCategory: usergroup.PolicyDefs[id].Category, // T29：分类；未识别 key 返回零值 ""
			InstanceValues: beforeVal,
			TargetValues:   afterVal,
		}
		row.Status = computeRowStatus(beforeVal, afterVal)
		rows = append(rows, row)
	}
	return rows
}

// policyEntryToValue 把 policy entry 转 value（Name 和 Value 分离）：
//   - 用户配额 / 模型配额：Name = label（如"单用户 Agent 数量上限"），Value = 格式化数值（如"3"）
//   - 功能权限开关：Name = label（如"允许用户配置模型"），Value = "开启"/"关闭"
//   - 未识别 policy_key：Name = ""，Value = 格式化 meta["value"]
//
// buildPolicyEntries 里 bool 类 policy 用 Meta["enabled"]，其余走 Meta["value"]，
// 本函数两个键都识别。Meta 缺失时退化为纯 label（Value = ""）。
func policyEntryToValue(ctx context.Context, e usergroup.ConfigEntry) configDiffValue {
	metaMap, _ := e.Meta.(map[string]interface{})

	// token_quota_rules / global_token_quota_rules：用对应 _day key 的 ID + Label，
	// 值格式化为可读字符串（如 "2026/07/07 17:14 - 2026/07/08 00:00，按日刷新 100,000"）
	switch e.ID {
	case usergroup.PolicyKeyTokenQuotaRules:
		if dayDef, ok := usergroup.PolicyDefs[usergroup.PolicyKeyTokenQuotaDay]; ok {
			return configDiffValue{
				ID:    usergroup.PolicyKeyTokenQuotaDay,
				Name:  i18n.T(ctx, i18n.NewKey(dayDef.Label)),
				Value: formatTokenQuotaRulesValue(ctx, metaMap),
			}
		}
	case usergroup.PolicyKeyGlobalTokenQuotaRules:
		if dayDef, ok := usergroup.PolicyDefs[usergroup.PolicyKeyGlobalTokenQuotaDay]; ok {
			return configDiffValue{
				ID:    usergroup.PolicyKeyGlobalTokenQuotaDay,
				Name:  i18n.T(ctx, i18n.NewKey(dayDef.Label)),
				Value: formatTokenQuotaRulesValue(ctx, metaMap),
			}
		}
	}

	var policyName, policyValue string
	switch {
	case isFeatureTogglePolicy(e.ID):
		var boolVal interface{}
		if v, ok := metaMap["enabled"]; ok {
			boolVal = v
		} else if v, ok := metaMap["value"]; ok {
			boolVal = v
		}
		policyName = e.Label
		policyValue = featureToggleLabel(ctx, boolVal)
	case isQuotaPolicy(e.ID):
		policyName = e.Label
		if v, ok := metaMap["value"]; ok {
			policyValue = formatPolicyValue(v)
		}
	default:
		if v, ok := metaMap["value"]; ok {
			policyValue = formatPolicyValue(v)
		}
	}
	return configDiffValue{ID: e.ID, Name: policyName, Value: policyValue}
}

// formatTokenQuotaRulesValue 把 token_quota_rules / global_token_quota_rules 的
// meta.value（rules JSON 字符串）格式化为人类可读的字符串。
// 空规则或 -1 → "无限制"；day/month 模式 → 限制值；custom 模式 → "起始 - 截止，刷新方式 限制值"。
func formatTokenQuotaRulesValue(ctx context.Context, metaMap map[string]interface{}) string {
	raw, _ := metaMap["value"].(string)
	if raw == "" {
		return i18n.T(ctx, i18n.MsgGroupTreeMetaUnlimited)
	}
	rules, ok := model.ParseTokenQuotaRules(raw)
	if !ok || len(rules) == 0 {
		return i18n.T(ctx, i18n.MsgGroupTreeMetaUnlimited)
	}
	parts := make([]string, 0, len(rules))
	for _, r := range rules {
		parts = append(parts, formatSingleQuotaRule(ctx, r))
	}
	return strings.Join(parts, "；")
}

// formatSingleQuotaRule 格式化单条配额规则为可读字符串。
// 无限制（Limit < 0）时保留周期上下文：
//   - day/month → "每日 无限制" / "每月 无限制"
//   - custom    → "2026/07/09 20:07 - 无终止，按日刷新 无限制"
//
// 有限额时也保留周期前缀 + 千分位：
//   - day/month → "每日 30,000,000" / "每月 30,000,000"
//   - custom    → "2026/07/09 20:07 - 无终止，按日刷新 30,000,000"
func formatSingleQuotaRule(ctx context.Context, r model.TokenQuotaRule) string {
	unlimitedLabel := i18n.T(ctx, i18n.MsgGroupTreeMetaUnlimited)
	var limitStr string
	if r.Limit < 0 {
		limitStr = unlimitedLabel
	} else {
		limitStr = formatWithCommas(int64(r.Limit))
	}
	switch r.Mode {
	case model.QuotaModeDay:
		return i18n.T(ctx, i18n.MsgQuotaPeriodDaily) + " " + limitStr
	case model.QuotaModeMonth:
		return i18n.T(ctx, i18n.MsgQuotaPeriodMonthly) + " " + limitStr
	case model.QuotaModeCustom:
		var timeRange, refreshLabel string
		if r.Start != nil {
			timeRange = time.Unix(*r.Start, 0).Format("2006/01/02 15:04")
		}
		if r.End != nil {
			timeRange += " - " + time.Unix(*r.End, 0).Format("2006/01/02 15:04")
		} else {
			timeRange += " - " + i18n.T(ctx, i18n.MsgQuotaNoEndTime)
		}
		switch r.Refresh {
		case model.QuotaRefreshDaily:
			refreshLabel = i18n.T(ctx, i18n.MsgQuotaRefreshDaily)
		case model.QuotaRefreshMonthly:
			refreshLabel = i18n.T(ctx, i18n.MsgQuotaRefreshMonthly)
		case model.QuotaRefreshYearly:
			refreshLabel = i18n.T(ctx, i18n.MsgQuotaRefreshYearly)
		case model.QuotaRefreshNone, "":
			refreshLabel = i18n.T(ctx, i18n.MsgQuotaRefreshNone)
		}
		if refreshLabel != "" {
			return timeRange + ", " + refreshLabel + " " + limitStr
		}
		return timeRange + " " + limitStr
	default:
		return limitStr
	}
}

// formatWithCommas 将整数格式化为带千分位逗号的字符串（如 30000000 → "30,000,000"）。
func formatWithCommas(n int64) string {
	s := strconv.FormatInt(n, 10)
	neg := false
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	}
	var buf strings.Builder
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			buf.WriteByte(',')
		}
		buf.WriteByte(c)
	}
	if neg {
		return "-" + buf.String()
	}
	return buf.String()
}

// featureToggleLabel 把功能开关的 bool 值 i18n 化为 开启 / 关闭。
// 非 bool 或 nil 返回空串（走 policyEntryToValue 内部降级）。
func featureToggleLabel(ctx context.Context, v interface{}) string {
	b, ok := v.(bool)
	if !ok {
		return ""
	}
	if b {
		return i18n.T(ctx, i18n.MsgGroupTreeLabelOn)
	}
	return i18n.T(ctx, i18n.MsgGroupTreeLabelOff)
}

// isQuotaPolicy 判断 policy 是否属于配额类（用户配额 / 模型配额），
// 决定 policyEntryToValue 是否给 name 前置 label。
func isQuotaPolicy(policyKey string) bool {
	def, ok := usergroup.PolicyDefs[policyKey]
	if !ok {
		return false
	}
	return def.Category == usergroup.PolicyCategoryUserQuota ||
		def.Category == usergroup.PolicyCategoryModelQuota
}

// isFeatureTogglePolicy 判断 policy 是否属于功能权限开关类。
// name 直接用 label（如"允许用户配置模型"），不拼 value。
func isFeatureTogglePolicy(policyKey string) bool {
	def, ok := usergroup.PolicyDefs[policyKey]
	if !ok {
		return false
	}
	return def.Category == usergroup.PolicyCategoryFeatureToggle
}

func formatPolicyValue(v interface{}) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case uint:
		return strconv.FormatUint(uint64(t), 10)
	case float64:
		// JSON numbers Unmarshal 到 interface{} 时是 float64
		return strconv.FormatFloat(t, 'f', -1, 64)
	}
	return fmt.Sprintf("%v", v)
}

// entriesToValues 把 ConfigEntry 列表压缩为 configDiffValue 列表。
func entriesToValues(entries []usergroup.ConfigEntry) []configDiffValue {
	values := make([]configDiffValue, 0, len(entries))
	for _, e := range entries {
		values = append(values, configDiffValue{ID: e.ID, Name: e.NameHint, Value: e.Label})
	}
	return values
}

func getEntries(cat *usergroup.ConfigCategoryResult) []usergroup.ConfigEntry {
	if cat == nil {
		return nil
	}
	return cat.Entries
}

func filterEntriesBySubLabel(entries []usergroup.ConfigEntry, subLabel string) []usergroup.ConfigEntry {
	out := make([]usergroup.ConfigEntry, 0, len(entries))
	for _, e := range entries {
		if e.SubLabel == subLabel {
			out = append(out, e)
		}
	}
	return out
}

// forceAgentToolSubLabels 确保 企业插件 / 企业MCP 两个 sub_label 始终包含在列表中，
// 已存在的按原位置保留，缺失的依次追加到末尾。
func forceAgentToolSubLabels(ctx context.Context, subLabels []string) []string {
	required := []string{
		i18n.T(ctx, i18n.MsgGroupTreeSubLabelEnterprisePlugin),
		i18n.T(ctx, i18n.MsgGroupTreeSubLabelEnterpriseMCP),
	}
	seen := make(map[string]bool, len(subLabels))
	for _, s := range subLabels {
		seen[s] = true
	}
	result := make([]string, len(subLabels), len(subLabels)+len(required))
	copy(result, subLabels)
	for _, r := range required {
		if !seen[r] {
			result = append(result, r)
		}
	}
	return result
}

// collectSubLabelsInOrder 按出现顺序收集 before/after 两侧涉及的所有 sub_label（去重）。
// 优先使用 after（target）的顺序作为主序，让 target_config 与 per-instance 行 key 顺序对齐。
func collectSubLabelsInOrder(before, after *usergroup.ConfigCategoryResult) []string {
	seen := make(map[string]bool)
	ordered := make([]string, 0)
	for _, e := range getEntries(after) {
		if !seen[e.SubLabel] {
			seen[e.SubLabel] = true
			ordered = append(ordered, e.SubLabel)
		}
	}
	for _, e := range getEntries(before) {
		if !seen[e.SubLabel] {
			seen[e.SubLabel] = true
			ordered = append(ordered, e.SubLabel)
		}
	}
	return ordered
}

// computeRowStatus 计算两个值列表的关系。
//
//   - 都为空：same
//   - instance 任一项不在 target 内：different
//   - instance ⊊ target（包含且 target 更多）：contained_in_target
//   - 集合相等：same
//
// 比较 key 用 ID + Name 联合（同 ID 不同 Name 视作不同，便于 policy 行值比较）。
func computeRowStatus(instanceValues, targetValues []configDiffValue) string {
	targetSet := make(map[string]bool, len(targetValues))
	for _, v := range targetValues {
		targetSet[valueSetKey(v)] = true
	}
	instanceSet := make(map[string]bool, len(instanceValues))
	for _, v := range instanceValues {
		instanceSet[valueSetKey(v)] = true
	}
	if len(instanceSet) == 0 && len(targetSet) == 0 {
		return "same"
	}
	allContained := true
	for k := range instanceSet {
		if !targetSet[k] {
			allContained = false
			break
		}
	}
	if !allContained {
		return "different"
	}
	if len(targetSet) > len(instanceSet) {
		return "contained_in_target"
	}
	return "same"
}

// attachValueStatuses 根据行级 status 为每个 instance value 标注 per-value status。
//
// 语义：
//   - rowStatus ∈ {same, contained_in_target, not_check} → 所有 value 继承行 status
//   - rowStatus == different：value 在 targetSet 中 → same；不在 → different
func attachValueStatuses(rowStatus string, instanceValues, targetValues []configDiffValue) []configDiffValue {
	if len(instanceValues) == 0 {
		return instanceValues
	}
	if rowStatus != "different" {
		out := make([]configDiffValue, len(instanceValues))
		for i, v := range instanceValues {
			v.Status = rowStatus
			out[i] = v
		}
		return out
	}
	targetSet := make(map[string]bool, len(targetValues))
	for _, v := range targetValues {
		targetSet[valueSetKey(v)] = true
	}
	out := make([]configDiffValue, len(instanceValues))
	for i, v := range instanceValues {
		if targetSet[valueSetKey(v)] {
			v.Status = "same"
		} else {
			v.Status = "different"
		}
		out[i] = v
	}
	return out
}

func valueSetKey(v configDiffValue) string {
	// 用 \x00 做分隔，避免 ID/Value 内含 ":" 等字符引起冲突
	var b strings.Builder
	b.WriteString(v.ID)
	b.WriteByte(0)
	b.WriteString(v.Value)
	return b.String()
}

// groupFullPath 返回 group_id 对应的 full_path（0 → "未分组"）。
func groupFullPath(ctx context.Context, groupID uint) string {
	if groupID == 0 {
		return "未分组"
	}
	groups, err := model.GetGroupsByIDs(ctx, []uint{groupID})
	if err != nil || len(groups) == 0 {
		return ""
	}
	return groups[0].FullPath
}

// batchFetchMemoryPluginMap 批量查询 memory_tdai_plugins，返回 CVM instanceId → *MemoryTDAIPlugin。
func batchFetchMemoryPluginMap(ctx context.Context, cvmIDs []string) map[string]*model.MemoryTDAIPlugin {
	out := make(map[string]*model.MemoryTDAIPlugin, len(cvmIDs))
	if len(cvmIDs) == 0 {
		return out
	}
	var rows []model.MemoryTDAIPlugin
	if err := model.DB(ctx).Where("instance_id IN ?", cvmIDs).Find(&rows).Error; err != nil {
		slog.Warn("[ConfigDiff] memory plugin batch query failed", "err", err)
		return out
	}
	for i := range rows {
		out[rows[i].InstanceID] = &rows[i]
	}
	return out
}

// batchFetchDriveSpaceMap 批量查询 smh_personal_spaces，返回 DB instanceId → bool。
func batchFetchDriveSpaceMap(ctx context.Context, instanceIDs []uint) map[uint]bool {
	out := make(map[uint]bool, len(instanceIDs))
	if len(instanceIDs) == 0 {
		return out
	}
	var ids []uint
	if err := model.DB(ctx).Model(&model.SMHPersonalSpace{}).
		Select("instance_id").
		Where("instance_id IN ?", instanceIDs).
		Pluck("instance_id", &ids).Error; err != nil {
		slog.Warn("[ConfigDiff] smh_personal_spaces batch query failed", "err", err)
		return out
	}
	for _, id := range ids {
		out[id] = true
	}
	return out
}

// batchFetchCWPSecurityMap 批量查询 CWP DescribeMachines，返回 CVM instanceId → ProtectType。
func batchFetchCWPSecurityMap(ctx context.Context, cvmIDs []string) map[string]string {
	out := make(map[string]string, len(cvmIDs))
	if len(cvmIDs) == 0 {
		return out
	}
	credential, err := getCredential(ctx)
	if err != nil {
		slog.Warn("[ConfigDiff] CWP credential failed", "err", err)
		return out
	}
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "cwp.tencentcloudapi.com"
	cpf.HttpProfile.ReqMethod = "POST"
	client := common.NewCommonClient(credential, CVMRegion, cpf)
	req := tchttp.NewCommonRequest("cwp", "2018-02-28", "DescribeMachines")
	params := map[string]interface{}{
		"MachineType":   "CVM",
		"MachineRegion": CVMRegion,
		"Filters": []map[string]interface{}{
			{"Name": "instance-ids", "Values": cvmIDs},
		},
		"Limit": 100,
	}
	b, _ := json.Marshal(params)
	if err := req.SetActionParameters(string(b)); err != nil {
		slog.Warn("[ConfigDiff] CWP request payload failed", "err", err)
		return out
	}
	resp := tchttp.NewCommonResponse()
	if err := client.Send(req, resp); err != nil {
		slog.Warn("[ConfigDiff] CWP DescribeMachines failed", "err", err)
		return out
	}
	var result struct {
		Response struct {
			Machines []struct {
				InstanceID  string `json:"InstanceId"`
				ProtectType string `json:"ProtectType"`
			} `json:"Machines"`
		} `json:"Response"`
	}
	if err := json.Unmarshal(resp.GetBody(), &result); err != nil {
		slog.Warn("[ConfigDiff] CWP response unmarshal failed", "err", err)
		return out
	}
	for _, m := range result.Response.Machines {
		out[m.InstanceID] = m.ProtectType
	}
	return out
}
