package controller

// 本地 Agent scope 请求类型、分组解析与用户级分组切换。

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"hatchery/model"

	"gorm.io/gorm"
)

const localScopeBatchPrefix = "local-scope:"

func localScopeBatchID(id uint) string {
	return localScopeBatchPrefix + strconv.FormatUint(uint64(id), 10)
}

func parseLocalScopeBatchID(value string) uint {
	if !strings.HasPrefix(value, localScopeBatchPrefix) {
		return 0
	}
	id, _ := strconv.ParseUint(strings.TrimPrefix(value, localScopeBatchPrefix), 10, 64)
	return uint(id)
}

// normalizeLocalAgentCommandScope 将下发命令归一为 TeamAI 当前支持的 scope。
// 二期新增的任务会通过 batch_id 精确关联到 user / workspace 事实行；早期按实例
// 直接下发的任务没有该关联，按 user 兼容，避免把已废弃的 instance scope 暴露给客户端。
func normalizeLocalAgentCommandScope(scope, workspacePath string) (string, string) {
	if scope == model.LocalSkillScopeWorkspace && workspacePath != "" {
		return scope, workspacePath
	}
	return model.LocalSkillScopeUser, ""
}

// ---- 二期 request/response 类型 -----------------------------------------

// reportUserLevel 二期 report body 的 user_level 段。
type reportUserLevel struct {
	GroupID uint               `json:"group_id"`
	Skills  []reportSkillEntry `json:"skills"`
	Rules   []reportRuleEntry  `json:"rules,omitempty"` // 用户级已装规范（scope='user'）
}

// reportWorkspace 二期 report body 的 workspaces[] 单项。
type reportWorkspace struct {
	Path      string             `json:"path"`
	Name      string             `json:"name"`
	IDEType   string             `json:"ide_type"`
	ProjectID *uint              `json:"project_id,omitempty"`
	Skills    []reportSkillEntry `json:"skills"`
	Rules     []reportRuleEntry  `json:"rules,omitempty"` // 项目级已装规范（scope='workspace'）
}

// syncWorkspace 二期 sync body 的 workspaces[] 单项（不含 skills，只有身份 + 分组）。
type syncWorkspace struct {
	Path      string `json:"path"`
	Name      string `json:"name"`
	IDEType   string `json:"ide_type"`
	ProjectID *uint  `json:"project_id,omitempty"`
}

// ---- computeGroupActive --------------------------------------------------

// computeGroupActive 判断 groupID 是否在用户当前分组列表中。
// 不在列表里 → group_active=false（用户被移出该分组）。
func computeGroupActive(ctx context.Context, userID uint, groupID uint) bool {
	return computeGroupActiveWithDB(model.DB(ctx), ctx, userID, groupID)
}

// computeGroupActiveWithDB 是 computeGroupActive 的事务安全变体，调用方传入事务句柄 tx。
func computeGroupActiveWithDB(tx *gorm.DB, ctx context.Context, userID uint, groupID uint) bool {
	if groupID == 0 {
		return false
	}
	ids, err := model.GetUserGroupIDsWithDB(tx, userID)
	if err != nil {
		return false
	}
	for _, id := range ids {
		if id == groupID {
			return true
		}
	}
	return false
}

// ---- 用户主分组解析 -------------------------------------------------------

// resolveUserPrimaryGroup 根据用户的 OneID 分组信息，解析默认主分组 ID。
// 规则（对齐产品需求）：
//  1. 有 OneID 主分组（IsMain=true AND Source=oneid_dept）→ 返回该分组
//  2. 有 OneID 分组但无主分组 → 仅 1 个时返回该分组；多个时返回 0
//  3. 只有 manual 分组或无分组 → 返回 0
func resolveUserPrimaryGroup(ctx context.Context, tx *gorm.DB, userID uint) uint {
	var members []model.UserGroupMember
	if err := tx.Where("user_id = ? AND source = ?",
		userID, model.MemberSourceOneIDDept).Find(&members).Error; err != nil {
		return 0
	}
	if len(members) == 0 {
		return 0
	}
	// 优先 IsMain=true
	for _, m := range members {
		if m.IsMain {
			return m.UserGroupID
		}
	}
	// 仅 1 个 OneID 分组 → 用它
	if len(members) == 1 {
		return members[0].UserGroupID
	}
	// 多个 OneID 分组但无主 → 不自动选择
	return 0
}

// ---- 用户级分组切换（公用）-------------------------------------------------

// applyUserLevelGroupSwitch 是用户级主分组切换的核心逻辑，被以下入口复用：
//   - HandleSwitchUserLevelGroup（保留的前端手动切换入口）
//   - ensureUserLevelGroup（report 时按服务端用户关系检测被动切换）
//
// 行为：更新 resources.UserLevel、instances.group_id + 序列化回 instances + diffAndQueue(scope='user')。
// newGroupID=0 表示切到无分组（只更新 resources，不触发 diffAndQueue）。
// 返回新增 pending 数。
func applyUserLevelGroupSwitch(ctx context.Context, tx *gorm.DB, inst *model.Instance,
	resources *model.LocalAgentResources, newGroupID uint) (int, error) {

	groupName, err := model.GetUserGroupNameWithDB(tx, newGroupID)
	if err != nil {
		return 0, fmt.Errorf("查询用户级分组名称: %w", err)
	}
	resources.UserLevel.GroupID = newGroupID
	resources.UserLevel.GroupName = groupName

	// 序列化回 instances.local_agent_resources
	jsonBytes, err := json.Marshal(resources)
	if err != nil {
		return 0, fmt.Errorf("序列化 local_agent_resources: %w", err)
	}
	if err := tx.Model(&model.Instance{}).Where("id = ?", inst.ID).
		Updates(map[string]any{"local_agent_resources": string(jsonBytes), "group_id": newGroupID}).Error; err != nil {
		return 0, fmt.Errorf("更新本地 Agent 用户级组织: %w", err)
	}
	// 管控端 Agent 列表从 instances.group_id 回填组织路径，保持内存对象同步以供后续 report 处理复用。
	inst.GroupID = newGroupID
	inst.LocalAgentResources = resources
	if err := upsertUserScopeBinding(tx, inst.ID, newGroupID, time.Now()); err != nil {
		return 0, fmt.Errorf("更新用户级分组绑定: %w", err)
	}

	// diffAndQueue（newGroupID=0 时不触发）
	if newGroupID == 0 {
		return 0, nil
	}
	count, err := diffAndQueue(ctx, tx, model.LocalSkillScopeUser, inst.ID, newGroupID, "")
	if err != nil {
		return 0, fmt.Errorf("applyUserLevelGroupSwitch diffAndQueue: %w", err)
	}
	return count, nil
}

// syncLocalAgentInstanceGroup 修复本地 Agent 的实例组织冗余字段。
// instances.group_id 是管理端列表回填组织路径的来源，必须与用户级资源快照保持一致。
// 仅修复存储不一致，不触发资源 diff 或下发。
func syncLocalAgentInstanceGroup(tx *gorm.DB, inst *model.Instance, groupID uint) error {
	if inst.GroupID == groupID {
		return nil
	}
	if err := tx.Model(&model.Instance{}).Where("id = ?", inst.ID).Update("group_id", groupID).Error; err != nil {
		return fmt.Errorf("回填本地 Agent 实例组织: %w", err)
	}
	inst.GroupID = groupID
	return nil
}

// ensureUserLevelGroup 在 report 时依据服务端用户关系校验实例的用户级主分组：
//   - 新 agent（group_id=0）→ 自动分配用户主分组 + diffAndQueue
//   - 已有 agent（group_id!=0）→ 检查分组是否仍有效，失效则切到新主分组
//   - 用户无主分组且当前分组失效 → group_id 置 0（无分组）
//
// report.user_level.group_id 不参与本判断，TeamAI 上报该字段也不会改变主分组。
// 该逻辑用于用户被动切换主组织后，在下一次 report 触发一次用户级资源对账；
// 主组织未变化时不重复扫描用户级资产目录。
//
// 返回 groupChanged（分组是否发生变化）。
func ensureUserLevelGroup(ctx context.Context, tx *gorm.DB, inst *model.Instance,
	user *model.User, resources *model.LocalAgentResources) (groupChanged bool, err error) {

	currentGroupID := resources.UserLevel.GroupID

	// 已有分组且仍有效 → 不变（规则7：用户手动选择的分组，只要仍有效就保留）
	if currentGroupID != 0 && computeGroupActiveWithDB(tx, ctx, user.ID, currentGroupID) {
		if err := syncLocalAgentInstanceGroup(tx, inst, currentGroupID); err != nil {
			return false, err
		}
		return false, nil
	}

	// 新 agent 或分组失效 → 解析主分组（规则8：重新按默认规则判断）
	newGroupID := resolveUserPrimaryGroup(ctx, tx, user.ID)

	if newGroupID == currentGroupID {
		// 都是 0：新 agent 且用户无主分组；仍修复可能遗留的 instances.group_id。
		if err := syncLocalAgentInstanceGroup(tx, inst, newGroupID); err != nil {
			return false, err
		}
		return false, nil
	}

	// 分组变化 → 更新 resources + diffAndQueue
	if _, err := applyUserLevelGroupSwitch(ctx, tx, inst, resources, newGroupID); err != nil {
		return false, fmt.Errorf("ensureUserLevelGroup: %w", err)
	}
	return true, nil
}

// ---- diffAndQueue（方案 A · 严格按 catalog，含降级覆盖）------------------
//
// 触发时机：
//   - 前端 POST /openclaw/local/user-group 切换用户级分组
//   - 用户主分组由服务端切换后触发 user scope 对账
//   - report/sync 检测到 Workspace 的 project_id 或列表变化后触发 workspace scope 对账
//
// 参数：
//   - scope: "user" | "workspace"
//   - workspacePath: scope="workspace" 时传 workspace 路径；scope="user" 时传 ""
//
// 返回新增 pending 数。
//
// 清理策略（切换分组时取消旧分组"还没安装成功"的记录）：
//   - local_instance_skills 中 failed 行 → 硬删
//   - local_instance_skills 中 distributing 行 → 硬删（还没装完，取消）
//   - skill_distribution_records 中 pending/failed 行 → 标 cancelled（不删历史流水）
//   - skill_distribution_records 中 success 行 → 不动（历史保留）
//
// 清理后，catalog diff 只对比 distributed 状态的行，确保"已装"语义正确。
