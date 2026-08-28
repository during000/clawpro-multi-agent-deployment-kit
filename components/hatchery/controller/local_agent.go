package controller

// ============================================================
// 本地 agent (clawpro 一期) reporter 接口
//
// 设计文档：https://iwiki.woa.com/p/4022150701
//
// reporter 是嵌在用户本地 agent 进程里的小模块（OnSessionCreate / OnUserMessage
// hook），通过用户的 hk- API Token 调用以下三个接口与 hatchery 同步状态：
//
//   POST /local-agent/report           上报本地实例 + 已装 skill
//   POST /local-agent/sync             拉待执行命令（pending records）+ 拍 last_report_at
//   POST /local-agent/commands/ack     回写命令执行结果（id 走 Body）
//
// 鉴权：WithOpenAPI + requireLogin（参考 controller/agent_bridge_callback.go）。
// 鉴权后能拿到 user.ID，再用 (user_id, instance_id) 复合 key 做多租户隔离。
//
// 一期白名单：agent_type ∈ {workbuddy, codebuddy}（写死，二期再扩展）。
// ============================================================

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ---- 常量 / 配置 -------------------------------------------------------------

// ClawHub 公共 skill 库 download API。
// 本地 agent 下发未注册的 public skill 时，reporter 拉 commands
// 拿不到 cos_zip_key，拼这个 URL + ?slug=<slug> 让 reporter 去公共仓库下载。
const clawHubDownloadAPI = "https://wry-manatee-359.convex.site/api/v1/download"

// SkillHub 是未注册 skill 重试下载用的备选源。
// 场景：某未注册 skill 上一次在同一本地 instance 上 ack 失败，用户重试安装时
// HandleLocalAgentSync 会改拼 skillhub.cn 的 URL 换个下载源，试试能不能装上。
const skillHubDownloadAPI = "https://api.skillhub.cn/api/v1/download"

// localAgentSyncProtocolVersion 是 /local-agent/sync 响应的协议版本标记。
// reporter 据此判断能力（当前 v2 同时返回 commands 老字段 + cmds 统一字段做灰度兼容）。
// 后续协议演进时在此升版，并在 reporter 侧按版本区分消费。
const localAgentSyncProtocolVersion = "v2"

// buildClawHubDownloadURL 拼接 ClawHub 下载链接。version 为空则拉最新。
func buildClawHubDownloadURL(slug string) string {
	return clawHubDownloadAPI + "?slug=" + slug
}

// buildSkillHubDownloadURL 拼接 SkillHub 下载链接。与 ClawHub 使用同样的 ?slug= 查询参数。
func buildSkillHubDownloadURL(slug string) string {
	return skillHubDownloadAPI + "?slug=" + slug
}

// LocalAgentTypeMeta 是本地 agent type 的描述性元数据。一期双类型写死，
// 二期考虑走表。`Code` 为 reporter 上报与 DB 存储的原始值。
type LocalAgentTypeMeta struct {
	Code        string
	Name        string
	Description string
}

// localAgentTypes 一期 agent_type 白名单单一真相源。
//
// reporter 接口校验、/admin/local-agent-types 返回列表都从此派生，
// 以防未来扩类型时漏改某一处。数组衰出保证 /admin 接口 JSON
// 顺序稳定。
var localAgentTypes = []LocalAgentTypeMeta{
	{
		Code:        "codebuddy",
		Name:        "CodeBuddy",
		Description: "本机代码助手 agent，由用户在本地安装并运维",
	},
	{
		Code:        "workbuddy",
		Name:        "WorkBuddy",
		Description: "本机工作流 agent，由用户在本地安装并运维",
	},
	{
		Code:        "claude",
		Name:        "Claude Code",
		Description: "本机ai agent，由用户在本地安装并运维",
	},
	{
		Code:        "codex",
		Name:        "Codex",
		Description: "本机ai agent，由用户在本地安装并运维",
	},
}

// localAgentTypeWhitelist 从 localAgentTypes 派生、供 reporter 接口 agent_type 校验使用。
var localAgentTypeWhitelist = func() map[string]bool {
	m := make(map[string]bool, len(localAgentTypes))
	for _, t := range localAgentTypes {
		m[t.Code] = true
	}
	return m
}()

// local_agent_id 必须为 16 位 hex
var localAgentIDRegexp = regexp.MustCompile(`^[a-fA-F0-9]{16}$`)

// formatLocalInstanceID 派生 instance_id：local-<agent_type>-<local_agent_id 末 6 位>。
// 调用方需先校验 agent_type 和 local_agent_id 合法性。
func formatLocalInstanceID(agentType, localAgentID string) string {
	const suffixLen = 6
	if len(localAgentID) < suffixLen {
		return ""
	}
	return fmt.Sprintf("local-%s-%s", agentType, strings.ToLower(localAgentID[len(localAgentID)-suffixLen:]))
}

// validateLocalAgentInputs 校验 agent_type / local_agent_id 并返回派生的 instance_id。
func validateLocalAgentInputs(agentType, localAgentID string) (string, *hcommon.RichError) {
	agentType = strings.ToLower(strings.TrimSpace(agentType))
	if !localAgentTypeWhitelist[agentType] {
		return "", hcommon.I18nError(i18n.MsgLocalAgentInvalidAgentType)
	}
	localAgentID = strings.TrimSpace(localAgentID)
	if !localAgentIDRegexp.MatchString(localAgentID) {
		return "", hcommon.I18nError(i18n.MsgLocalAgentInvalidLocalAgentID)
	}
	instanceID := formatLocalInstanceID(agentType, localAgentID)
	if instanceID == "" {
		return "", hcommon.I18nError(i18n.MsgLocalAgentInvalidLocalAgentID)
	}
	return instanceID, nil
}

// ensureLocalAgentAllowed 检查当前 user 所属租户是否可以使用本地 Agent 接入，两层守卫：
//
// 　　① feature_allowlist (type='local-agent', identifier=user.Identifier) 同意
// 　　　  表语义：某 type 下无记录 = 全开；有记录 = 仅表内放行。
// 　　② SiteConfig.LocalAgentEnabled = true
// 　　　  租户 admin 在 /admin/config 显式设置；默认 false（保守）。
//
// 任一层拒绝 → 直接写 403。调用方根据返回 bool 决定是否继续。
//
// 下期计划叠加第 ③ 层：分组策略 UserGroup.policy.local_agent，
// 联动 ResolvePolicyBoolForGroup，与 doctor_enabled 同模式。
func ensureLocalAgentAllowed(w http.ResponseWriter, r *http.Request, user *model.User) bool {
	if user == nil {
		return false
	}
	ctx := r.Context()

	// ① 跨租户白名单
	allowed, err := model.IsFeatureAllowed(ctx, model.FeatureAllowlistTypeLocalAgent, user.Identifier)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
		return false
	}
	if !allowed {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgLocalAgentNotAllowed))
		return false
	}

	// ② 租户全局预设
	if !model.GetSiteConfig(ctx).LocalAgentEnabled {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgLocalAgentNotAllowed))
		return false
	}

	return true
}

// ---- POST /local-agent/report -----------------------------------------

type reportRequest struct {
	AgentType    string     `json:"agent_type"`
	AgentVersion string     `json:"agent_version"`
	LocalAgentID string     `json:"local_agent_id"`
	HostName     string     `json:"host_name"`
	OS           string     `json:"os"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	LastStatus   string     `json:"last_status,omitempty"`

	// ─── 二期新增 ───
	UserLevel  *reportUserLevel  `json:"user_level,omitempty"` // 用户级资源（scope='user'）
	Workspaces []reportWorkspace `json:"workspaces,omitempty"` // 项目级 workspace 列表（scope='workspace'）
}

type reportSkillEntry struct {
	Slug        string `json:"slug"`
	Version     string `json:"version"`
	DisplayName string `json:"display_name,omitempty"`
	Source      string `json:"source,omitempty"` // public / enterprise / local；reporter 上报什么落什么
}

// reportRuleEntry report body 中 rules[] 单项（用户级和项目级均用此结构）。
type reportRuleEntry struct {
	Slug        string `json:"slug"`
	Version     string `json:"version"`
	DisplayName string `json:"display_name,omitempty"`
	RuleType    string `json:"rule_type,omitempty"` // prompt / rule
	Source      string `json:"source,omitempty"`    // enterprise / local
}

// HandleLocalAgentReport 处理 reporter 上报。
// 行为：
//
//  1. upsert instances（按 (user_id, instance_id) 复合 key），source=local
//
//  2. upsert local_instance_infos
//
//  3. 本地 skill 全量对齐（reporter 上报代表该实例当前 skill 的全量真相）：
//     - 本次上报里有、DB 里没有→ INSERT
//     - 本次上报里有、DB 里也有且信息一致 (slug, version, display_name, source) → 跳过（不刷 updated_at）
//     - 本次上报里有、DB 里也有但信息不一致 → UPDATE上述字段
//     - DB 里有、本次上报里没有 → DELETE（硬删，不分 source）
//     installed_at 仅首次插入时赋值；last_seen_at 仅作为上报心跳拍，不参与信息一致判定。
//
//     前提约定：reporter 是该实例 skill 状态的唯一真相源，ack=success 写入后下一轮上报一定会包含
//     新装的 skill（reporter 端同步扫描）。如未来发现“装完闪一下又消失”，可加安装完成后的
//     宽限期（installed_at < now-Ns才能被消失即删命中）。
func HandleLocalAgentReport(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	if !ensureLocalAgentAllowed(w, r, user) {
		return
	}

	var req reportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidJSON))
		return
	}
	defer r.Body.Close()

	instanceCID, vErr := validateLocalAgentInputs(req.AgentType, req.LocalAgentID)
	if vErr != nil {
		writeError(w, r, http.StatusBadRequest, vErr)
		return
	}

	now := time.Now()
	ctx := r.Context()

	// 二期计数（在事务外声明，供事务内赋值、事务外返回）
	var userLevelSynced, projectSynced, rulesSynced int

	// 1. upsert Instance（事务保证 instance + info 原子）
	var inst model.Instance
	txErr := model.DB(ctx).Transaction(func(tx *gorm.DB) error {
		// 查现有 instance（同 user + 同 instance_id 才视为同一行）。
		// 用 Unscoped 查：包含已软删行——移除本地 Agent 后实例被软删，
		// 同一 local_agent_id 重新 report 时应重新激活（恢复 deleted_at）而非新建。
		// 同时加 FOR UPDATE 锁：并发 report/sync 同一实例时，统一先锁实例行再写
		// local_agent_scope_bindings，避免 AB-BA 死锁。
		err := tx.Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND instance_id = ? AND source = ?",
				user.ID, instanceCID, model.InstanceSourceLocal).First(&inst).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			// 新增
			displayName := req.HostName
			if displayName == "" {
				displayName = instanceCID
			}
			inst = model.Instance{
				UserID:          user.ID,
				InstanceId:      instanceCID,
				Name:            displayName,
				AgentType:       strings.ToLower(req.AgentType),
				AgentVersion:    req.AgentVersion,
				Source:          model.InstanceSourceLocal,
				LastKnownStatus: model.StatusRunning,
			}
			if err := tx.Create(&inst).Error; err != nil {
				return err
			}
		case err != nil:
			return err
		default:
			// 更新版本号 + 状态缓存
			// 注意：若实例正处于本地 Agent 卸载中（current_operation=uninstall_local_agent），
			// 不要覆盖 last_known_status——保持 destroying，否则 reporter 心跳会把「销毁中」
			// 冲回 running，前端永远看不到卸载进度。卸载中间态由 uninstall_teamai 任务链路管理。
			lastKnownStatus := model.StatusRunning
			if inst.CurrentOperation == model.LocalAgentOpUninstall {
				lastKnownStatus = model.StatusDestroying
			}
			updates := map[string]any{
				"agent_version":     req.AgentVersion,
				"last_known_status": lastKnownStatus,
				"status_synced_at":  now,
			}
			// 软删实例重新接入：恢复 deleted_at（被移除本地 Agent 后通过 report 重新激活）
			if inst.DeletedAt.Valid {
				updates["deleted_at"] = nil
			}
			// host_name 变了的话顺手更新 name（仅在用户没改过时友好）
			if req.HostName != "" && (inst.Name == "" || inst.Name == inst.InstanceId) {
				updates["name"] = req.HostName
			}
			// 用 Unscoped 更新：避免 gorm 自动附加 deleted_at IS NULL 条件，
			// 否则软删行（恢复 deleted_at 场景）会更新 0 行。
			if err := tx.Unscoped().Model(&inst).Updates(updates).Error; err != nil {
				return err
			}
		}

		// 2. upsert LocalInstanceInfo
		info := model.LocalInstanceInfo{
			InstanceID:   inst.ID,
			HostName:     req.HostName,
			OS:           req.OS,
			StartedAt:    req.StartedAt,
			LastReportAt: &now,
			LastStatus:   req.LastStatus,
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "instance_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"host_name", "os", "started_at", "last_report_at", "last_status", "updated_at", "deleted_at",
			}),
		}).Create(&info).Error; err != nil {
			return err
		}

		// 3. 二期：服务端在 report 时校验用户级主分组（不依赖 request.user_level.group_id）。
		//    - 新 agent → 按服务端用户关系解析主分组并触发 diffAndQueue
		//    - 已有 agent → 当前主分组被动变化或失效时，切到服务端解析结果并触发 diffAndQueue
		// TeamAI 不负责传递或切换用户级 group_id；其传入值不会作为切换依据。
		resources := deserializeLocalAgentResources(inst.LocalAgentResources)
		if _, err := ensureUserLevelGroup(ctx, tx, &inst, user, resources); err != nil {
			return err
		}
		// 更新 inst.LocalAgentResources 供后续步骤使用
		inst.LocalAgentResources = resources

		// 4. 二期：用户级 + 项目级资源处理（仅当 report body 包含 user_level/workspaces 时）
		if req.UserLevel != nil || len(req.Workspaces) > 0 {
			us, ps, rs, err := processReportUserLevelAndWorkspaces(ctx, tx, &inst, user, &req)
			if err != nil {
				return err
			}
			userLevelSynced = us
			projectSynced = ps
			rulesSynced = rs
		}

		return nil
	})

	if txErr != nil {
		Logger(ctx).Error("[LocalAgent] report transaction failed", "error", txErr, "agent_id", req.LocalAgentID)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(txErr, i18n.MsgInternalError))
		return
	}

	jsonOK(w, map[string]any{
		"instance_id":       inst.InstanceId,
		"instance_pk":       inst.ID,
		"received_at":       now.Format(time.RFC3339),
		"rules_synced":      rulesSynced,
		"user_level_synced": userLevelSynced,
		"project_synced":    projectSynced,
	})
}

// ---- POST /local-agent/sync -------------------------------------------

type syncRequest struct {
	AgentType    string `json:"agent_type"`
	LocalAgentID string `json:"local_agent_id"`
	Status       string `json:"status"` // running / error

	// ─── 二期新增 ───
	Workspaces []syncWorkspace `json:"workspaces,omitempty"` // agent 上报当前 workspace 列表
}

type commandItem struct {
	ID           uint   `json:"id"`
	Type         string `json:"type"`                   // install_skill / uninstall_skill（本期）/ install_rule / uninstall_rule（rule 预留）/ uninstall_teamai（三期）
	SkillSlug    string `json:"skill_slug"`             // 优先 JOIN skills.slug，未注册走 task.slug
	SkillVersion string `json:"skill_version"`          // records.version
	DownloadURL  string `json:"download_url,omitempty"` // install_skill 才有
	Cmd          string `json:"cmd,omitempty"`          // 三期：uninstall_teamai 卸载命令
	// ─── 二期新增 ───
	Scope          string `json:"scope"`                    // user / workspace
	WorkspacePath  string `json:"workspace_path,omitempty"` // 仅 scope="workspace" 时有
	ProjectID      uint   `json:"project_id,omitempty"`
	AgentType      string `json:"agent_type,omitempty"` // execute_agent_task 专属
	Prompt         string `json:"prompt,omitempty"`     // execute_agent_task 专属
	Executor       string `json:"executor,omitempty"`
	TargetAgentID  string `json:"target_agent_id,omitempty"`
	IMateProjectID string `json:"imate_project_id,omitempty"`
}

// syncCmdItem 是 sync 返回的「统一字段」命令项（cmds 列表）。
//
// 三期协议演进：所有资产类型（skill / rule / hook / 卸载插件）共用一套字段，
// 不再各自带 skill_/rule_ 前缀。与老字段名的 commandItem / ruleCommandItem 并存，
// 由 sync 组装逻辑从同一份数据映射生成，保证两列表内容一致。
//
// omitempty 规则：
//   - skill / rule：slug + version + download_url（rule 另带 handle_type）
//   - hook：slug + version + handle_type + event + cmd（无 download_url）
//   - uninstall_teamai：仅 id + type + cmd
//
// type 取值（与 commands 数组完全一致）：
//
//	install_skill / uninstall_skill / install_prompt_rule / install_rule_rule /
//	uninstall_prompt_rule / uninstall_rule_rule / install_hook_rule /
//	uninstall_hook_rule / uninstall_teamai
type syncCmdItem struct {
	ID             uint   `json:"id"`
	Type           string `json:"type"`
	Slug           string `json:"slug,omitempty"`
	Version        string `json:"version,omitempty"`
	HandleType     string `json:"handle_type,omitempty"` // prompt / rule / hook
	Event          string `json:"event,omitempty"`       // hook 专属触发时机
	Cmd            string `json:"cmd,omitempty"`         // hook / uninstall_teamai 执行命令
	DownloadURL    string `json:"download_url,omitempty"`
	Scope          string `json:"scope"`
	WorkspacePath  string `json:"workspace_path,omitempty"`
	ProjectID      uint   `json:"project_id,omitempty"`
	AgentType      string `json:"agent_type,omitempty"` // execute_agent_task 专属
	Prompt         string `json:"prompt,omitempty"`     // execute_agent_task 专属
	Executor       string `json:"executor,omitempty"`
	TargetAgentID  string `json:"target_agent_id,omitempty"`
	IMateProjectID string `json:"imate_project_id,omitempty"`
}

// querySkillCommands 查询本实例 pending 的 skill 下发命令（install_skill / uninstall_skill）。
//
// 内部完成：JOIN skills 取 slug/cos_zip_key、LEFT JOIN local_instance_skills 反查 scope/workspace_path、
// 构建 workspace→project 映射、查询同 slug 历史终态 record 判定 ClawHub/SkillHub 备选源、
// 最终映射成 reporter 协议视角的 commandItem 列表。
//
// 返回老字段名列表（commands），供老版本 reporter 使用；统一字段列表 cmds 由调用方从
// 此处结果映射生成（与 queryRuleCommands / queryLocalAgentTaskCommands 形态对齐）。
func querySkillCommands(ctx context.Context, inst *model.Instance) ([]commandItem, error) {
	// 拉 pending + failed records，JOIN skills 取 slug / display_name / cos_zip_key
	// 二期：LEFT JOIN local_instance_skills 反查 scope + workspace_path
	type row struct {
		RecordID      uint
		Type          string
		Version       string
		TaskSlug      string
		SkillSlug     string
		COSZipKey     string
		Scope         string
		WorkspacePath string
		BatchID       string
	}
	var rows []row
	if err := model.DB(ctx).
		Model(&model.SkillDistributionRecord{}).
		Select(`skill_distribution_records.id AS record_id,
		        skill_distribution_records.type AS type,
		        skill_distribution_records.version AS version,
		        skill_distribution_tasks.slug AS task_slug,
		        skills.slug AS skill_slug,
		        skills.cos_zip_key AS cos_zip_key,
		        skill_distribution_tasks.batch_id AS batch_id,
		        local_instance_skills.scope AS scope,
		        local_instance_skills.workspace_path AS workspace_path`).
		Joins("JOIN skill_distribution_tasks ON skill_distribution_tasks.id = skill_distribution_records.task_id").
		Joins("LEFT JOIN skills ON skills.id = skill_distribution_records.skill_id").
		Joins("LEFT JOIN local_instance_skills ON local_instance_skills.instance_id = skill_distribution_records.instance_id AND local_instance_skills.slug = COALESCE(skills.slug, skill_distribution_tasks.slug) AND local_instance_skills.install_status IN ?",
			[]string{model.LocalSkillInstallStatusDistributing, model.LocalSkillInstallStatusFailed}).
		Where("skill_distribution_records.instance_id = ? AND skill_distribution_records.status = ?", inst.ID, model.RecordStatusPending).
		Order("skill_distribution_records.id ASC").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("query skill distribution records: %w", err)
	}
	localSkillIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		if id := parseLocalScopeBatchID(row.BatchID); id > 0 {
			localSkillIDs = append(localSkillIDs, id)
		}
	}
	localSkillByID := make(map[uint]model.LocalInstanceSkill)
	if len(localSkillIDs) > 0 {
		var localRows []model.LocalInstanceSkill
		if err := model.DB(ctx).Where("id IN ?", uniqueUintIDs(localSkillIDs)).Find(&localRows).Error; err != nil {
			return nil, fmt.Errorf("query local instance skills: %w", err)
		}
		for _, row := range localRows {
			localSkillByID[row.ID] = row
		}
	}
	projectByPath := make(map[string]uint)
	var scopeBindings []model.LocalAgentScopeBinding
	if err := model.DB(ctx).Where("instance_id = ? AND scope = ?", inst.ID, model.LocalAgentScopeWorkspace).Find(&scopeBindings).Error; err != nil {
		return nil, fmt.Errorf("query local agent scope bindings: %w", err)
	}
	for _, binding := range scopeBindings {
		projectByPath[binding.ScopeKey] = binding.ProjectID
	}

	// 查询本 instance 上、同一 task.slug 的最新一条终态 distribute record（排除当前的 pending）。
	// 用于判定未注册 skill 重试下发时是否要换成 SkillHub 备选源：
	//   最新一条终态 record 是 failed/upgrade_failed/uninstall_failed_old → 走 SkillHub
	//   是 success / 从未下过 → 走 ClawHub
	// 实现：拉本 instance 所有终态 distribute record（按 id DESC），内存里取每个 slug 首条。
	// pending record 数量本身少，所以同名历史终态 record 也不会高。避开 GROUP_CONCAT
	// 这类 MySQL/SQLite 语义不一致的能力，保证单元测试（sqlite）与生产（mysql）一致。
	type histRow struct {
		Slug   string
		Status string
	}
	var histRows []histRow
	if err := model.DB(ctx).
		Model(&model.SkillDistributionRecord{}).
		Select(`skill_distribution_tasks.slug AS slug,
		        skill_distribution_records.status AS status`).
		Joins("JOIN skill_distribution_tasks ON skill_distribution_tasks.id = skill_distribution_records.task_id").
		Where("skill_distribution_records.instance_id = ? AND skill_distribution_records.type = ? AND skill_distribution_records.status <> ?",
			inst.ID, model.TaskTypeDistribute, model.RecordStatusPending).
		Order("skill_distribution_records.id DESC").
		Scan(&histRows).Error; err != nil {
		return nil, fmt.Errorf("query skill distribution history: %w", err)
	}
	lastFailedSlugs := make(map[string]bool)
	seen := make(map[string]bool, len(histRows))
	for _, h := range histRows {
		if h.Slug == "" || seen[h.Slug] {
			continue
		}
		seen[h.Slug] = true
		switch h.Status {
		case model.RecordStatusFailed, model.RecordStatusUpgradeFailed, model.RecordStatusUninstallFailedOld:
			lastFailedSlugs[h.Slug] = true
		}
	}

	// 按 record_id 去重（同一条 record 可能 JOIN 到多个 local_instance_skills 行导致重复）
	seenRecordIDs := make(map[uint]bool, len(rows))
	cmds := make([]commandItem, 0, len(rows))
	for _, rr := range rows {
		if seenRecordIDs[rr.RecordID] {
			continue
		}
		seenRecordIDs[rr.RecordID] = true
		// slug 优先用 skills.slug（已注册的权威来源），fallback tasks.slug（兜底未注册场景）
		slug := rr.SkillSlug
		if slug == "" {
			slug = rr.TaskSlug
		}
		// DB type 映射到 reporter 视角的协议 type：
		//   distribute → install_skill（update_skill 不单独区分，reporter 端幂等覆盖）
		//   uninstall  → uninstall_skill
		// 遇到未知 type 跳过，避免 reporter 拿到不认识的价值。
		var protoType string
		switch rr.Type {
		case model.TaskTypeDistribute:
			protoType = "install_skill"
		case model.TaskTypeUninstall:
			protoType = "uninstall_skill"
		default:
			continue
		}
		if localRow, ok := localSkillByID[parseLocalScopeBatchID(rr.BatchID)]; ok {
			rr.Scope, rr.WorkspacePath = localRow.Scope, localRow.WorkspacePath
		}
		rr.Scope, rr.WorkspacePath = normalizeLocalAgentCommandScope(rr.Scope, rr.WorkspacePath)
		item := commandItem{
			ID:            rr.RecordID,
			Type:          protoType,
			SkillSlug:     slug,
			SkillVersion:  rr.Version,
			Scope:         rr.Scope,
			WorkspacePath: rr.WorkspacePath,
			ProjectID:     projectByPath[rr.WorkspacePath],
		}
		if rr.Type == model.TaskTypeDistribute {
			switch {
			case rr.COSZipKey != "":
				// 已注册 skill：走 SMH。本地 reporter 跑在用户机器上，必须用公网域名
				url, urlErr := buildSMHDownloadURL(ctx, rr.COSZipKey, false)
				if urlErr != nil {
					// SMH 配置不全等场景：跳过该条，不影响其它命令派发；让 reporter 下个周期再拉
					continue
				}
				item.DownloadURL = url
			case slug != "":
				// 未注册 skill 走外部备选源。默认 ClawHub；
				// 若同 (instance, slug) 最新一次终态 record 是失败（用户重装场景），换 SkillHub 试试。
				if lastFailedSlugs[slug] {
					item.DownloadURL = buildSkillHubDownloadURL(slug)
				} else {
					item.DownloadURL = buildClawHubDownloadURL(slug)
				}
			default:
				// slug 为空不可能拼出有效下载链：必须跳过，避免 reporter 拿到空 URL 安装全部技能
				continue
			}
		}
		cmds = append(cmds, item)
	}
	return cmds, nil
}

// queryLocalAgentTaskCommands 查询本实例 pending 的通用本地任务（local_agent_tasks），
// 组装成 sync 命令。本期仅 uninstall_teamai 一种。
//
// 返回：
//   - cmdItems：统一字段列表（cmds），仅含 id / type / cmd
//   - legacyItems：老字段名列表（commands），uninstall_teamai 用通用 commandItem 承载
//
// 老 reporter 遇到不认识的 type=uninstall_teamai 会跳过，不受影响。
func queryLocalAgentTaskCommands(ctx context.Context, inst *model.Instance) (cmdItems []syncCmdItem, legacyItems []commandItem, err error) {
	var tasks []model.LocalAgentTask
	if err = model.DB(ctx).
		Where("instance_id = ? AND status = ?", inst.ID, model.LocalAgentTaskStatusPending).
		Order("id ASC").
		Find(&tasks).Error; err != nil {
		return nil, nil, err
	}
	cmdItems = make([]syncCmdItem, 0, len(tasks))
	legacyItems = make([]commandItem, 0, len(tasks))
	for _, t := range tasks {
		cmdItem := syncCmdItem{
			ID:   t.ID,
			Type: t.Type,
			Cmd:  t.Cmd,
		}
		legacyItem := commandItem{
			ID:   t.ID,
			Type: t.Type,
			Cmd:  t.Cmd,
		}
		if t.Type == model.LocalAgentTaskTypeExecuteAgent {
			var payload agentTaskCommandPayload
			_ = json.Unmarshal([]byte(t.Cmd), &payload)
			cmdItem.Scope = model.LocalAgentScopeWorkspace
			cmdItem.ProjectID = t.ProjectID
			cmdItem.WorkspacePath = t.WorkspacePath
			cmdItem.AgentType = t.AgentType
			cmdItem.Prompt = t.Prompt
			cmdItem.Executor = payload.Executor
			cmdItem.TargetAgentID = payload.TargetAgentID
			cmdItem.IMateProjectID = payload.IMateProjectID
			legacyItem.Scope = model.LocalAgentScopeWorkspace
			legacyItem.ProjectID = t.ProjectID
			legacyItem.WorkspacePath = t.WorkspacePath
			legacyItem.AgentType = t.AgentType
			legacyItem.Prompt = t.Prompt
			legacyItem.Executor = payload.Executor
			legacyItem.TargetAgentID = payload.TargetAgentID
			legacyItem.IMateProjectID = payload.IMateProjectID
		}
		cmdItems = append(cmdItems, cmdItem)
		legacyItems = append(legacyItems, legacyItem)
	}
	return cmdItems, legacyItems, nil
}

// HandleLocalAgentSync 拉取本地实例的 pending 命令。
// POST /local-agent/sync
//
// Request body (JSON):
//
//	{"agent_type":"workbuddy","local_agent_id":"<id>"}
//
// 选用 POST 而非 GET 的原因：
//  1. 本接口顶背面会顺手更新 last_report_at（sync 也算心跳），有副作用，
//     不适合 GET 语义（GET 应为幂等、可被代理缓存）。
//  2. 与 /report、/commands/ack 统一都是 POST + JSON body，避免 reporter
//     身份参数出现在 access log URL 中。
func HandleLocalAgentSync(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	if !ensureLocalAgentAllowed(w, r, user) {
		return
	}

	var req syncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgBadRequest))
		return
	}
	agentType := strings.TrimSpace(req.AgentType)
	localAgentID := strings.TrimSpace(req.LocalAgentID)
	instanceCID, vErr := validateLocalAgentInputs(agentType, localAgentID)
	if vErr != nil {
		writeError(w, r, http.StatusBadRequest, vErr)
		return
	}

	// 找实例：必须是当前 user 的 local 实例
	var inst model.Instance
	err := model.DB(r.Context()).Where("user_id = ? AND instance_id = ? AND source = ?",
		user.ID, instanceCID, model.InstanceSourceLocal).First(&inst).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// reporter 还没 report 过，返回空命令列表（不算错）
			jsonOK(w, map[string]any{"ok": true, "version": localAgentSyncProtocolVersion, "commands": []commandItem{}})
			return
		}
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
		return
	}

	// 顺手记一笔 last_report_at（sync 也算心跳）
	now := time.Now()
	model.DB(r.Context()).Model(&model.LocalInstanceInfo{}).
		Where("instance_id = ?", inst.ID).
		Update("last_report_at", now)

	// 二期：处理 workspaces[] 上报（project_id 变化检测 + diffAndQueue）
	if len(req.Workspaces) > 0 {
		if err := processSyncWorkspaces(r.Context(), &inst, user, req.Workspaces); err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
			return
		}
	}

	// 三期前：查询本实例 pending skill 下发命令（封装到 querySkillCommands）
	cmds, qErr := querySkillCommands(r.Context(), &inst)
	if qErr != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(qErr, i18n.MsgInternalError))
		return
	}

	// 合并 skill + rule commands（老字段名，commands 列表，供老版本 reporter 使用）
	allCmds := make([]any, 0, len(cmds))
	for _, c := range cmds {
		allCmds = append(allCmds, c)
	}
	ruleCmds, rErr := queryRuleCommands(r.Context(), &inst)
	if rErr != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(rErr, i18n.MsgInternalError))
		return
	}
	for _, c := range ruleCmds {
		allCmds = append(allCmds, c)
	}

	// 三期：组装统一字段 cmds 列表（单一数据源，与 commands 内容一致）
	cmdItems := make([]syncCmdItem, 0, len(cmds)+len(ruleCmds))
	for _, c := range cmds {
		cmdItems = append(cmdItems, syncCmdItem{
			ID:            c.ID,
			Type:          c.Type,
			Slug:          c.SkillSlug,
			Version:       c.SkillVersion,
			DownloadURL:   c.DownloadURL,
			Scope:         c.Scope,
			WorkspacePath: c.WorkspacePath,
			ProjectID:     c.ProjectID,
		})
	}
	for _, c := range ruleCmds {
		cmdItems = append(cmdItems, syncCmdItem{
			ID:            c.ID,
			Type:          c.Type,
			Slug:          c.RuleSlug,
			Version:       c.RuleVersion,
			HandleType:    c.RuleType,
			Event:         c.Event,
			Cmd:           c.Cmd,
			DownloadURL:   c.DownloadURL,
			Scope:         c.Scope,
			WorkspacePath: c.WorkspacePath,
			ProjectID:     c.ProjectID,
		})
	}

	// 合并本地通用任务（本期 uninstall_teamai）到两个列表
	taskCmds, taskLegacy, taskErr := queryLocalAgentTaskCommands(r.Context(), &inst)
	if taskErr != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(taskErr, i18n.MsgInternalError))
		return
	}
	for _, c := range taskLegacy {
		allCmds = append(allCmds, c)
	}
	cmdItems = append(cmdItems, taskCmds...)

	jsonOK(w, map[string]any{"ok": true, "version": localAgentSyncProtocolVersion, "commands": allCmds, "cmds": cmdItems})
}

// ---- POST /local-agent/commands/ack ------------------------------

type ackRequest struct {
	ID        uint64 `json:"id"`                   // 命令 ID（= skill/rule distribution_records.id）
	Type      string `json:"type"`                 // install_skill / uninstall_skill / install_prompt_rule / install_rule_rule / uninstall_prompt_rule / uninstall_rule_rule
	Status    string `json:"status"`               // success / failed；execute_agent_task 另支持 running
	Error     string `json:"error,omitempty"`      // failed 时填
	Version   string `json:"version,omitempty"`    // success 时回报实际安装版本（可选）
	Result    string `json:"result,omitempty"`     // execute_agent_task 的增量或最终输出
	SessionID string `json:"session_id,omitempty"` // 本地 Agent 会话标识
}

// HandleLocalAgentAck 处理 reporter 回写命令执行结果。
//
// POST /local-agent/commands/ack
//
// 路由注册（普通 method+path，无 path 参数）：
//
//	http.HandleFunc("/local-agent/commands/ack", ...)
//
// id 通过 Request Body 字段读，与 report / sync 一致（所有信息都在 body 里）。
// type 也通过 Request Body 读（与 sync 返回的 command.type 一致），用于区分 skill/rule 记录。
//
// 行为：事务内
//  1. 校验 type（install_skill / uninstall_skill / install_rule / uninstall_rule）
//  2. 按 type 路由到对应表查记录（本期只实现 skill 路径，rule 预留）
//  3. owner = current user 通过 instance JOIN 校验
//  4. status='pending' 时才允许更新；否则幂等返回当前结果
//  5. 把 records.status 改成 success / failed，写 error / version
//  6. distribute + success：upsert local_instance_skills（source 从 skill.visibility_type 推断）
//     uninstall  + success：删除 local_instance_skills 对应行
//     failed                 ：local_instance_skills 不动
func HandleLocalAgentAck(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	if !ensureLocalAgentAllowed(w, r, user) {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxAgentTaskResultBytes+(64<<10))
	var req ackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidJSON))
		return
	}
	defer r.Body.Close()

	if req.ID == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "id"))
		return
	}
	recordID := uint(req.ID)

	// 校验 type：与 sync 返回的 command.type 一致，用于区分 skill/rule 记录。
	// 一期 agent 不传 type → 空串默认按 skill 处理（向后兼容）。
	ackType := strings.ToLower(strings.TrimSpace(req.Type))
	isRuleAck := false
	status := strings.ToLower(strings.TrimSpace(req.Status))
	switch ackType {
	case "", model.CommandTypeInstallSkill, model.CommandTypeUninstallSkill:
		// skill 路径（空串 = 一期 agent 向后兼容）
	case model.CommandTypeInstallPromptRule, model.CommandTypeInstallRuleRule, model.CommandTypeUninstallPromptRule, model.CommandTypeUninstallRuleRule,
		model.CommandTypeInstallHookRule, model.CommandTypeUninstallHookRule:
		// rule 路径（含三期 hook，复用 rule 下发/记录管道）
		isRuleAck = true
	case model.CommandTypeUninstallTeamai:
		// 三期：移除本地 Agent（卸载 clawpro-teamai 插件）走本地任务路径
		handleLocalAgentUninstallTeamaiAck(w, r, user, recordID, status, req.Error)
		return
	case model.LocalAgentTaskTypeExecuteAgent:
		handleLocalAgentExecutionAck(w, r, user, recordID, status, req.Result, req.Error, req.SessionID)
		return
	default:
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "type"))
		return
	}

	if status != model.RecordStatusSuccess && status != model.RecordStatusFailed {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "status"))
		return
	}

	// rule 路径走独立处理（同事基础设施已就绪）
	if isRuleAck {
		handleLocalAgentRuleAck(w, r, user, recordID, ackType, status, req.Error, req.Version)
		return
	}

	now := time.Now()
	ctx := r.Context()
	var (
		rec   model.SkillDistributionRecord
		task  model.SkillDistributionTask
		skill model.Skill
		inst  model.Instance
	)

	txErr := model.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&rec, recordID).Error; err != nil {
			return err
		}
		if err := tx.First(&inst, rec.InstanceID).Error; err != nil {
			return err
		}
		if inst.UserID != user.ID || inst.Source != model.InstanceSourceLocal {
			return gorm.ErrRecordNotFound
		}
		// 查 task 拿 slug 兜底（本地下发未注册 skill 时 task.Slug 是唯一可识别 slug 来源）。
		// rec.TaskID==0 的老数据走 silent fallback。
		if rec.TaskID != 0 {
			if err := tx.First(&task, rec.TaskID).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}
		// 本地下发未注册 skill 时 rec.SkillID=0；其他情况 skills 表理论上都有匹配。
		// skills 行被删的边界场景下 fall through 走 task.Slug 兜底，避免 ack 永远失败。
		if rec.SkillID != 0 {
			if err := tx.First(&skill, rec.SkillID).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}

		// 已结算的不重复改（幂等）。允许 pending → success/failed + failed → success（重试成功）
		if rec.Status != model.RecordStatusPending && rec.Status != model.RecordStatusFailed {
			return nil
		}
		// failed → failed 不重复改（幂等）
		if rec.Status == model.RecordStatusFailed && status == model.RecordStatusFailed {
			return nil
		}

		// 二期：反查 local_instance_skills 获取 scope + workspace_path
		// ack 请求不回传 scope/workspace_path，从 DB 反查（diffAndQueue 写 pending 时已 upsert 了 local_instance_skills）
		var lis model.LocalInstanceSkill
		lisScope := model.LocalSkillScopeUser
		lisWorkspacePath := ""
		lisID := uint(0)
		lisSlug := ""
		if rowSlug := skill.Slug; rowSlug == "" {
			lisSlug = task.Slug
		} else {
			lisSlug = rowSlug
		}
		if localID := parseLocalScopeBatchID(task.BatchID); localID > 0 {
			if err := tx.Where("id = ? AND instance_id = ?", localID, inst.ID).First(&lis).Error; err == nil {
				lisScope, lisWorkspacePath, lisID = lis.Scope, lis.WorkspacePath, lis.ID
			}
		} else if lisSlug != "" {
			if err := tx.Where("instance_id = ? AND slug = ? AND install_status IN ?",
				inst.ID, lisSlug,
				[]string{model.LocalSkillInstallStatusDistributing, model.LocalSkillInstallStatusFailed}).
				First(&lis).Error; err == nil {
				lisScope = lis.Scope
				lisWorkspacePath = lis.WorkspacePath
				lisID = lis.ID
			}
		}

		// 更新 records
		updates := map[string]any{
			"status":     status,
			"error":      req.Error,
			"updated_at": now,
		}
		if status == model.RecordStatusSuccess && strings.TrimSpace(req.Version) != "" {
			updates["version"] = strings.TrimSpace(req.Version)
		}
		if err := tx.Model(&rec).Updates(updates).Error; err != nil {
			return err
		}

		// 同步回写 task 计数。原因：executeSkillTaskAsync 给本地实例 record 裁出了
		// asyncRecords，其 finalize 阶段只统计 CVM 实例的 success/failed。本地实例的
		// 终态只能由 reporter ack 这里原子递增。使用 sql.Expr 避免丢失 update。
		// rec.TaskID == 0 的老数据跳过。
		if rec.TaskID != 0 {
			countCol := model.RecordStatusSuccess
			if status == model.RecordStatusFailed {
				countCol = model.RecordStatusFailed
			}
			if err := tx.Model(&model.SkillDistributionTask{}).
				Where("id = ?", rec.TaskID).
				Update(countCol, gorm.Expr(countCol+" + ?", 1)).Error; err != nil {
				return err
			}

			// finalize：success+failed >= total 且当前仍为 running 时，标为 completed。
			// CVM 同步路径里由 skill_task_executor 主流程一次性打“status=completed”，
			// 本地 agent 走异步 ack，需在这里逆向推导。仅按 (id, status='running')
			// 更新，保证重复 ack / 并发下不会把 completed 反复改。
			var t model.SkillDistributionTask
			if err := tx.Select("id", "total", "success", "failed", "status").
				First(&t, rec.TaskID).Error; err != nil {
				return err
			}
			if t.Status == model.TaskStatusRunning && t.Total > 0 && t.Success+t.Failed >= t.Total {
				if err := tx.Model(&model.SkillDistributionTask{}).
					Where("id = ? AND status = ?", rec.TaskID, model.TaskStatusRunning).
					Update("status", model.TaskStatusCompleted).Error; err != nil {
					return err
				}
			}
		}

		// 更新 local_instance_skills
		switch {
		case status == model.RecordStatusSuccess && rec.Type == model.TaskTypeDistribute:
			// 优先 skills 表（已注册）的字段，兜底 record.Slug（ClawHub 未注册 skill）
			rowSlug := skill.Slug
			if rowSlug == "" {
				rowSlug = task.Slug
			}
			displayName := skill.Name
			if displayName == "" {
				displayName = rowSlug
			}
			// source 判定只看 skill.ID：
			//   - skill.ID != 0：hatchery skills 表里有行 → 企业内部上传 → enterprise
			//   - skill.ID == 0：handleAddSkillLocal 公共 ClawHub 兜底 → public
			//
			// 不能用 skill.VisibilityType 推断。visibility 是「谁可见」的维度
			//（all/group），与「技能来源」是正交的：企业 skill 默认 visibility=all，
			// 这里会被误判为 public（这就是历史 bug）。
			rowSource := model.LocalSkillSourceEnterprise
			if skill.ID == 0 {
				rowSource = model.LocalSkillSourcePublic
			}
			row := model.LocalInstanceSkill{
				InstanceID:    inst.ID,
				Slug:          rowSlug,
				Version:       pickVersion(req.Version, rec.Version, skill.Version),
				DisplayName:   displayName,
				Source:        rowSource,
				Scope:         lisScope,         // 从 local_instance_skills 反查
				WorkspacePath: lisWorkspacePath, // 从 local_instance_skills 反查
				InstallStatus: model.LocalSkillInstallStatusDistributed,
				InstalledAt:   &now,
				LastSeenAt:    &now,
			}
			// 二期：优先更新已存在的 distributing 行（diffAndQueue 写的），否则 OnConflict upsert
			if lisID != 0 {
				if err := tx.Model(&model.LocalInstanceSkill{}).
					Where("id = ?", lisID).
					Updates(map[string]any{
						"version":        row.Version,
						"display_name":   row.DisplayName,
						"source":         row.Source,
						"install_status": model.LocalSkillInstallStatusDistributed,
						"installed_at":   &now,
						"last_seen_at":   &now,
					}).Error; err != nil {
					return err
				}
			} else {
				// OnConflict 用新唯一索引 (scope, instance_id, workspace_path, slug)
				if err := tx.Clauses(clause.OnConflict{
					Columns: []clause.Column{
						{Name: "scope"},
						{Name: "instance_id"},
						{Name: "workspace_path"},
						{Name: "slug"},
					},
					DoUpdates: clause.AssignmentColumns([]string{
						"version", "display_name", "source", "install_status",
						"installed_at", "last_seen_at", "updated_at",
					}),
				}).Create(&row).Error; err != nil {
					return err
				}
			}
		case status == model.RecordStatusSuccess && rec.Type == model.TaskTypeUninstall:
			// 优先 task.Slug（ClawHub 兜底场景必填），fallback skills 表
			delSlug := task.Slug
			if delSlug == "" {
				delSlug = skill.Slug
			}
			if delSlug == "" {
				// 没有任何可识别 slug：直接跳过删除，避免误删全部 slug='' 行
				return nil
			}
			if err := tx.Where("instance_id = ? AND slug = ? AND scope = ?",
				inst.ID, delSlug, lisScope).
				Delete(&model.LocalInstanceSkill{}).Error; err != nil {
				return err
			}
		case status == model.RecordStatusFailed:
			// 二期：ack 失败 → local_instance_skills.install_status='failed'
			// 保留行以便前端展示"下发失败"状态 + 下次 sync 拉走重试
			failedSlug := skill.Slug
			if failedSlug == "" {
				failedSlug = task.Slug
			}
			if failedSlug == "" || lisID == 0 {
				// 没有可识别 slug 或没找到对应的 distributing 行 → 跳过
				return nil
			}
			if err := tx.Model(&model.LocalInstanceSkill{}).
				Where("id = ?", lisID).
				Update("install_status", model.LocalSkillInstallStatusFailed).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if txErr != nil {
		if errors.Is(txErr, gorm.ErrRecordNotFound) {
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgInstanceNotFoundOrNoPerm))
			return
		}
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(txErr, i18n.MsgInternalError))
		return
	}

	jsonOK(w, map[string]any{
		"record_id": rec.ID,
		"status":    rec.Status,
	})
}

// handleLocalAgentExecutionAck 接收 TeamAI/Edge Runtime 的任务启动、增量输出和终态结果。
// running 可重复上报 result 全量快照；success/failed 为终态，重复 ACK 幂等返回。
func handleLocalAgentExecutionAck(w http.ResponseWriter, r *http.Request, user *model.User, taskID uint, status, result, ackErr, sessionID string) {
	if status != model.LocalAgentTaskStatusRunning &&
		status != model.LocalAgentTaskStatusSuccess &&
		status != model.LocalAgentTaskStatusFailed {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "status"))
		return
	}
	if len(result) > maxAgentTaskResultBytes {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "result"))
		return
	}
	if len([]rune(strings.TrimSpace(sessionID))) > 191 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "session_id"))
		return
	}

	now := time.Now()
	var task model.LocalAgentTask
	txErr := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND type = ?", taskID, model.LocalAgentTaskTypeExecuteAgent).
			First(&task).Error; err != nil {
			return err
		}
		var inst model.Instance
		if err := tx.Where("id = ? AND user_id = ? AND source = ?",
			task.InstanceID, user.ID, model.InstanceSourceLocal).First(&inst).Error; err != nil {
			return err
		}
		if task.Status == model.LocalAgentTaskStatusSuccess ||
			task.Status == model.LocalAgentTaskStatusFailed ||
			task.Status == model.LocalAgentTaskStatusCancelled {
			return nil
		}
		updates := map[string]any{
			"status": status,
		}
		if result != "" {
			updates["result"] = result
		}
		if task.StartedAt == nil {
			updates["started_at"] = now
		}
		if normalizedSessionID := strings.TrimSpace(sessionID); normalizedSessionID != "" {
			updates["session_id"] = normalizedSessionID
		}
		if status == model.LocalAgentTaskStatusSuccess || status == model.LocalAgentTaskStatusFailed {
			updates["finished_at"] = now
		}
		if status == model.LocalAgentTaskStatusFailed {
			updates["error"] = strings.TrimSpace(ackErr)
		} else if status == model.LocalAgentTaskStatusSuccess {
			updates["error"] = ""
		}
		if err := tx.Model(&task).Updates(updates).Error; err != nil {
			return err
		}
		return tx.First(&task, task.ID).Error
	})
	if txErr != nil {
		switch {
		case errors.Is(txErr, gorm.ErrRecordNotFound):
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgTaskNotFound, taskID))
		default:
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(txErr, i18n.MsgInternalError))
		}
		return
	}
	jsonOK(w, map[string]any{
		"record_id": task.ID,
		"status":    task.Status,
		"task":      newAgentTaskResponse(task),
	})
}

// handleLocalAgentUninstallTeamaiAck 处理 uninstall_teamai 命令回写。
//
// 路由：local_agent_tasks（id = task.ID）。
//   - success：任务置 success + 软删该本地实例（关联 skill/rule 数据保留）；
//     下次 reporter report 时通过 Unscoped 查询重新激活（恢复 deleted_at）。
//   - failed：任务置 failed + 记 error，保留任务可重试。
//   - 幂等：pending→success/failed、failed→success 允许；终态不覆盖。
func handleLocalAgentUninstallTeamaiAck(w http.ResponseWriter, r *http.Request, user *model.User, taskID uint, status, ackErr string) {
	ctx := r.Context()
	var task model.LocalAgentTask
	var inst model.Instance

	txErr := model.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&task, taskID).Error; err != nil {
			return err
		}
		if err := tx.First(&inst, task.InstanceID).Error; err != nil {
			return err
		}
		// owner 校验：必须是当前用户的 local 实例（防止越权 ack 他人任务）
		if inst.UserID != user.ID || inst.Source != model.InstanceSourceLocal {
			return gorm.ErrRecordNotFound
		}
		// 终态不覆盖（success/failed 已终态，除非从 failed 重试 success）
		if task.Status == model.LocalAgentTaskStatusSuccess {
			return nil
		}
		if status == model.LocalAgentTaskStatusSuccess {
			task.Status = model.LocalAgentTaskStatusSuccess
			task.Error = ""
			if err := tx.Save(&task).Error; err != nil {
				return err
			}
			// 先把实例置 destroyed 终态（同一事务内），确保即使后续软删行被 reporter
			// Unscoped 复活，状态机也能读到 destroyed 而非卡在 destroying。
			if err := tx.Model(&model.Instance{}).
				Where("id = ?", inst.ID).
				Update("last_known_status", model.StatusDestroyed).Error; err != nil {
				return err
			}
			// 清空 current_operation / current_operation_state：卸载完成，实例不再处于卸载中。
			// 否则 reporter 再 report 时 inst.CurrentOperation 仍为 uninstall_local_agent，会把
			// last_known_status 强行打回 destroying（甚至 Unscoped 复活软删行），导致实例卡死。
			if err := tx.Model(&model.Instance{}).
				Where("id = ?", inst.ID).
				Updates(map[string]any{
					"current_operation":       "",
					"current_operation_state": "",
				}).Error; err != nil {
				return err
			}
			// reporter 执行卸载成功：清理本地实例关联数据 + 软删 instances 行。
			// last_known_status 已显式置 destroyed、current_operation 已清空。
			// skills / rules 硬删（无 deleted_at，且有唯一索引 upsert 约束，不能软删）；
			// infos / instances 软删（有 deleted_at）。
			if err := tx.Where("instance_id = ?", inst.ID).Delete(&model.LocalInstanceSkill{}).Error; err != nil {
				return err
			}
			if err := tx.Where("instance_id = ?", inst.ID).Delete(&model.LocalInstanceRule{}).Error; err != nil {
				return err
			}
			if err := tx.Where("instance_id = ?", inst.ID).Delete(&model.LocalInstanceInfo{}).Error; err != nil {
				return err
			}
			if err := tx.Delete(&inst).Error; err != nil {
				return err
			}
			return nil
		}
		// failed：记录错误，保留任务支持重试；实例退出卸载中状态回到正常运行态
		task.Status = model.LocalAgentTaskStatusFailed
		task.Error = ackErr
		if err := tx.Model(&model.Instance{}).
			Where("id = ? AND current_operation = ?", inst.ID, model.LocalAgentOpUninstall).
			Updates(map[string]any{
				"last_known_status":       model.StatusRunning,
				"current_operation":       "",
				"current_operation_state": "",
				"status_synced_at":        time.Now(),
			}).Error; err != nil {
			return err
		}
		return tx.Save(&task).Error
	})
	if txErr != nil {
		if errors.Is(txErr, gorm.ErrRecordNotFound) {
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgInstanceNotFoundOrNoPerm))
			return
		}
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(txErr, i18n.MsgInternalError))
		return
	}

	jsonOK(w, map[string]any{
		"record_id": task.ID,
		"status":    task.Status,
	})
}

// pickVersion 优先用 ack 上报的版本；fallback 到 record / skill 表上的版本。
func pickVersion(candidates ...string) string {
	for _, c := range candidates {
		if v := strings.TrimSpace(c); v != "" {
			return v
		}
	}
	return ""
}

// ---- GET /local-agent/availability ------------------------------------

// availabilityResponse 即 GET /local-agent/availability 的响应体。
//
// 仅一个字段 enabled——前端只需要这个聚合判定。详见 iwiki §5.B.7。
// 一期：enabled = feature_allowlist 命中 AND SiteConfig.LocalAgentEnabled
// 下期：再 AND ResolvePolicyBoolForGroup(local_agent, user.GroupID, ...)
//
// 内部决策因子（白名单未命中 / 全局开关关闭）不外泄到 API 契约，
// 前端不需要、也不应该据此做差异化处理；普通用户也不应据此诊断"为什么我看不到"。
type availabilityResponse struct {
	Enabled bool `json:"enabled"`
}

// HandleLocalAgentAvailability 普通用户视角查询本地 Agent 是否可用。
//
// 鉴权：requireLogin（任何已登录用户均可）
// 入参：无
// 返回：{ "enabled": bool }
//
// 实现说明：
//   - 不复用 ensureLocalAgentAllowed——那个会在不允许时写 403；
//     availability 接口永远返 200 + bool，让前端用单一值决策 UI 展示。
//   - 任一层查询出错：返 500（不能假装 enabled=true 给前端，会误导后续 reporter 调用失败）。
//   - 出于性能考虑：第二层 SiteConfig 短路在第一层之后，避免白名单已拦下时的多余查询。
func HandleLocalAgentAvailability(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	ctx := r.Context()

	// ① 跨租户白名单
	allowed, err := model.IsFeatureAllowed(ctx, model.FeatureAllowlistTypeLocalAgent, user.Identifier)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
		return
	}

	// ② 租户全局预设（短路：白名单未命中直接返 false，不必查 SiteConfig）
	enabled := allowed && model.GetSiteConfig(ctx).LocalAgentEnabled

	jsonOK(w, availabilityResponse{Enabled: enabled})
}

// ---- GET /local-agent/get-config --------------------------------------

// 本地 agent 拉取 CLS 公网上报配置所需的固定值字段（全局固定值，由用户确定后写入常量）。
// run_cmd 为模板，含 ${endpoint}/${topic_id}/${secret_id}/${secret_key}/${user_name}/${user_id}
// 占位符，由 handler 实时替换为实际值；${local_agent_id} 为本地 agent 自身标识，
// 服务端在拉配置阶段未知，保留占位符下发给 agent 端自行填充。
const (
	localAgentCLSInstallCmd   = "curl -fsSL https://haihub-model-bj-1251001002.cos.ap-beijing.myqcloud.com/teamai/install_cls.sh  | bash -s  --  install --cls-endpoint  ${endpoint}  --cls-topic-id ${topic_id}  --cls-secret-id  ${secret_id}  --cls-secret-key ${secret_key} --user-id ${user_id}  --user-name  ${user_name}"
	localAgentCLSRunCmd       = "onesuite-pilot start"
	localAgentCLSUpdateCmd    = "curl -fsSL https://haihub-model-bj-1251001002.cos.ap-beijing.myqcloud.com/teamai/update_cls.sh  | bash -s  --  install --cls-endpoint  ${endpoint}  --cls-topic-id ${topic_id}  --cls-secret-id  ${secret_id}  --cls-secret-key ${secret_key} --user-id ${user_id}  --user-name  ${user_name}"
	localAgentCLSUninstallCmd = "curl -fsSL  https://haihub-model-bj-1251001002.cos.ap-beijing.myqcloud.com/teamai/install_cls.sh  | bash -s -- uninstall --purge"
)

// localAgentCheckCLSClawServiceOpened 是 CheckCLSClawServiceOpened 的测试 seam，
// 便于单测中替换返回值而无需真实调用腾讯云 CLS API。
var localAgentCheckCLSClawServiceOpened = CheckCLSClawServiceOpened

// HandleLocalAgentGetConfig 处理 GET /local-agent/get-config。
//
// 本地 agent（reporter / agent 自身）在需要往 CLS 公网上报日志/指标/Trace 前，
// 主动拉取接入配置：公网 endpoint + topic_id（实时查）+ 永久 AK/SK（按租户查表）。
// 与 §5.A.1~5.A.3 同鉴权前置（两层白名单），但本接口是 GET 只读查询，不写实例状态。
//
// Query: config_type（可选）。不传返回全量配置（一期仅 cls）；传 cls 等价于筛选 cls；
// 传其他类型返回 400。
func HandleLocalAgentGetConfig(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	if !ensureLocalAgentAllowed(w, r, user) {
		return
	}
	ctx := r.Context()

	configType := r.URL.Query().Get("config_type")
	// config_type 非必传：空（全量）/ cls（筛选 cls）都合法，其他类型 400
	if configType != "" && configType != "cls" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidConfigType, configType))
		return
	}

	// 公网 CLS endpoint（注意是 tencentcs.com，非 CVM 实例用的 tencentyun.com 内网域名）
	endpoint := fmt.Sprintf("%s.cls.tencentcs.com", CVMRegion)

	// topic_id 实时查（CLS OpenClawService），不落库；未开通 / topic 为空 → 4xx。
	clsResult, err := localAgentCheckCLSClawServiceOpened(ctx)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgInternalError))
		return
	}
	if clsResult == nil || clsResult.TraceTopicId == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgCLSServiceNotEnabled))
		return
	}

	// secret 按租户查（model.DB(ctx) 走 identifier 回调自动过滤）。
	// 一期仅有 cls；config_type 为空（全量）时也返回 cls 块。
	var cred model.LocalAgentCLSCredential
	if err := model.DB(ctx).Where("config_type = ?", "cls").First(&cred).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgLocalGetConfigCredentialNotReady))
		return
	}

	resp := map[string]any{
		"cls": map[string]any{
			"endpoint":      endpoint,
			"topic_id":      clsResult.TraceTopicId,
			"secret_id":     cred.SecretID,
			"secret_key":    cred.SecretKey,
			"user_id":       user.ID,
			"user_name":     user.Username,
			"install_cmd":   localAgentCLSInstallCmd,
			"run_cmd":       localAgentCLSRunCmd,
			"update_cmd":    localAgentCLSUpdateCmd,
			"uninstall_cmd": localAgentCLSUninstallCmd,
		},
	}
	jsonOK(w, resp)
}

// ---- 移除本地 Agent（三期） -----------------------------------------------

// createUninstallTeamaiTask 创建「卸载本地 Agent」本地任务（uninstall_teamai）。
//
// 幂等：同一实例已存在 pending 的 uninstall_teamai 任务时，不重复创建，直接返回已有任务。
// cmd 在创建时按实例 agent_type 拼好落表（审计可追溯）：teamai uninstall --force --agent <agent_type>。
//
// 注意：仅软任务，不立即删实例——由 reporter 下次 sync 拉取命令后本地执行，ack success 才软删。
func createUninstallTeamaiTask(ctx context.Context, tx *gorm.DB, inst *model.Instance, operatorID uint) (*model.LocalAgentTask, error) {
	var existing model.LocalAgentTask
	err := tx.
		Where("instance_id = ? AND type = ? AND status = ?", inst.ID, model.LocalAgentTaskTypeUninstallTeamai, model.LocalAgentTaskStatusPending).
		First(&existing).Error
	if err == nil {
		return &existing, nil // 已有 pending 任务，直接复用
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	cmd := fmt.Sprintf("teamai uninstall --force --agent %s", inst.AgentType)
	task := model.LocalAgentTask{
		Identifier:  model.CurrentIdentifier(ctx),
		InstanceID:  inst.ID,
		InstanceCID: inst.InstanceId,
		Type:        model.LocalAgentTaskTypeUninstallTeamai,
		Cmd:         cmd,
		Status:      model.LocalAgentTaskStatusPending,
		OperatorID:  operatorID,
	}
	// 下发任务即把实例标记为卸载中：
	//   - last_known_status = destroying：前端展示「销毁中」（该字段是真实状态，直接返回前端）
	//   - current_operation = uninstalling_local：防重入（重复下发会命中 pending 任务，不重复进此分支）
	// 不复用 setOperation/clearOperation（那是 CVM 删除专用状态机，会写 status_synced_at / last_stable_state 等本地 agent 不需要的字段）。
	// 注意：调用方需将本函数包在事务里（写 destroying + 创建任务原子提交），避免「destroying 但无任务」脏状态。
	if err := tx.Model(&model.Instance{}).
		Where("id = ?", inst.ID).
		Updates(map[string]any{
			"last_known_status":       model.StatusDestroying,
			"current_operation":       model.LocalAgentOpUninstall,
			"current_operation_state": model.OpStateProcessing,
			"status_synced_at":        time.Now(),
		}).Error; err != nil {
		return nil, err
	}
	if err := tx.Create(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// HandleLocalAgentRemove 用户端移除自己的本地 Agent。
//
// 仅能移除自己的 source=local 实例（防越权删他人实例）。创建 uninstall_teamai 任务后即返回，
// 实际卸载由 reporter 拉命令执行（离线场景等待下次连接）。
func HandleLocalAgentRemove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	if !ensureLocalAgentAllowed(w, r, user) {
		return
	}

	var req struct {
		InstanceID uint `json:"instance_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidJSON))
		return
	}
	if req.InstanceID == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidID))
		return
	}

	ctx := r.Context()
	var inst model.Instance
	if err := model.DB(ctx).Where("id = ? AND user_id = ? AND source = ?",
		req.InstanceID, user.ID, model.InstanceSourceLocal).First(&inst).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgInstanceNotFoundOrNoPerm))
			return
		}
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
		return
	}

	var task *model.LocalAgentTask
	txErr := model.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		task, err = createUninstallTeamaiTask(ctx, tx, &inst, user.ID)
		return err
	})
	if txErr != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(txErr, i18n.MsgInternalError))
		return
	}

	jsonOK(w, map[string]any{
		"ok":      true,
		"task_id": task.ID,
		"status":  task.Status,
	})
}
