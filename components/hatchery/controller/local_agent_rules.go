package controller

// ============================================================
// 本地 Agent 资源分发：rule（规范）ack + sync 查询
//
// 同事已完成企业规范库基础设施（model/enterprise_rule.go 等），
// 本文件接入 local-agent sync/ack 的 rule 路径：
//   - handleLocalAgentRuleAck：处理 install_prompt_rule / install_rule_rule /
//     uninstall_prompt_rule / uninstall_rule_rule 的 ack 回写
//   - queryRuleCommands：sync 时查询 rule 的 pending/failed records
//
// skill 路径在 local_agent.go，rule 路径在本文件。
// ============================================================

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ---- rule ack ---------------------------------------------------------------

type localAgentRuleAckState struct {
	record        model.RuleDistributionRecord
	task          model.RuleDistributionTask
	rule          model.EnterpriseRule
	instance      model.Instance
	ruleSlug      string
	localRuleID   uint
	scope         string
	workspacePath string
	now           time.Time
}

// handleLocalAgentRuleAck 处理 rule 类型的 ack 回写。
// type ∈ {install_prompt_rule, install_rule_rule, uninstall_prompt_rule, uninstall_rule_rule}
func handleLocalAgentRuleAck(w http.ResponseWriter, r *http.Request, user *model.User,
	recordID uint, ackType, status, errMsg, ackVersion string) {
	ctx := r.Context()
	state := localAgentRuleAckState{now: time.Now()}

	txErr := model.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := loadLocalAgentRuleAckState(tx, &state, user.ID, recordID); err != nil {
			return err
		}
		if !shouldApplyLocalAgentRuleAck(state.record.Status, status) {
			return nil
		}
		if err := updateLocalAgentRuleAckRecord(tx, &state, status, errMsg, ackVersion); err != nil {
			return err
		}
		if err := updateLocalAgentRuleAckTask(tx, &state, status); err != nil {
			return err
		}
		if err := resolveLocalAgentRuleAckTarget(tx, &state); err != nil {
			return err
		}
		return applyLocalAgentRuleAck(tx, &state, status, ackVersion)
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
		"record_id": state.record.ID,
		"status":    state.record.Status,
	})
}

func loadLocalAgentRuleAckState(tx *gorm.DB, state *localAgentRuleAckState, userID, recordID uint) error {
	if err := tx.First(&state.record, recordID).Error; err != nil {
		return err
	}
	if err := tx.First(&state.instance, state.record.InstanceID).Error; err != nil {
		return err
	}
	if state.instance.UserID != userID || state.instance.Source != model.InstanceSourceLocal {
		return gorm.ErrRecordNotFound
	}
	if state.record.TaskID != 0 {
		if err := tx.First(&state.task, state.record.TaskID).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}
	if state.record.RuleID != 0 {
		if err := tx.First(&state.rule, state.record.RuleID).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}
	return nil
}

func shouldApplyLocalAgentRuleAck(currentStatus, ackStatus string) bool {
	if currentStatus != model.RuleRecordStatusPending && currentStatus != model.RuleRecordStatusFailed {
		return false
	}
	return currentStatus != model.RuleRecordStatusFailed || ackStatus != model.RuleRecordStatusFailed
}

func updateLocalAgentRuleAckRecord(tx *gorm.DB, state *localAgentRuleAckState, status, errMsg, ackVersion string) error {
	updates := map[string]any{"status": status, "error": errMsg, "updated_at": state.now}
	if status == model.RuleRecordStatusSuccess {
		version := strings.TrimSpace(ackVersion)
		if version == "" {
			version = state.record.Version
		}
		updates["version"] = version
	}
	return tx.Model(&state.record).Updates(updates).Error
}

func updateLocalAgentRuleAckTask(tx *gorm.DB, state *localAgentRuleAckState, status string) error {
	if state.record.TaskID == 0 {
		return nil
	}
	countCol := model.RuleRecordStatusSuccess
	if status == model.RuleRecordStatusFailed {
		countCol = model.RuleRecordStatusFailed
	}
	if err := tx.Model(&model.RuleDistributionTask{}).Where("id = ?", state.record.TaskID).
		Update(countCol, gorm.Expr(countCol+" + ?", 1)).Error; err != nil {
		return err
	}
	var task model.RuleDistributionTask
	if err := tx.Select("id", "total", "success", "failed", "status").First(&task, state.record.TaskID).Error; err != nil {
		return err
	}
	if task.Status != model.TaskStatusRunning || task.Total <= 0 || task.Success+task.Failed < task.Total {
		return nil
	}
	return tx.Model(&model.RuleDistributionTask{}).
		Where("id = ? AND status = ?", state.record.TaskID, model.TaskStatusRunning).
		Update("status", model.TaskStatusCompleted).Error
}

func resolveLocalAgentRuleAckTarget(tx *gorm.DB, state *localAgentRuleAckState) error {
	state.ruleSlug = state.rule.Slug
	if state.ruleSlug == "" {
		state.ruleSlug = state.task.Slug
	}
	state.scope = model.LocalSkillScopeUser
	var localRule model.LocalInstanceRule
	if localID := parseLocalScopeBatchID(state.task.BatchID); localID > 0 {
		if err := tx.Where("id = ? AND instance_id = ?", localID, state.instance.ID).First(&localRule).Error; err == nil {
			state.scope, state.workspacePath, state.localRuleID = localRule.Scope, localRule.WorkspacePath, localRule.ID
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return nil
	}
	if state.ruleSlug == "" {
		return nil
	}
	err := tx.Where("instance_id = ? AND slug = ? AND install_status IN ?", state.instance.ID, state.ruleSlug,
		[]string{model.LocalSkillInstallStatusDistributing, model.LocalSkillInstallStatusFailed}).First(&localRule).Error
	if err == nil {
		state.scope, state.workspacePath, state.localRuleID = localRule.Scope, localRule.WorkspacePath, localRule.ID
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	return err
}

func applyLocalAgentRuleAck(tx *gorm.DB, state *localAgentRuleAckState, status, ackVersion string) error {
	switch {
	case status == model.RuleRecordStatusSuccess && state.record.Type == model.RuleTaskTypeDistribute:
		return upsertLocalInstanceRuleFromAck(tx, state, ackVersion)
	case status == model.RuleRecordStatusSuccess && state.record.Type == model.RuleTaskTypeUninstall:
		if state.ruleSlug == "" {
			return nil
		}
		return tx.Where("instance_id = ? AND slug = ? AND scope = ?", state.instance.ID, state.ruleSlug, state.scope).
			Delete(&model.LocalInstanceRule{}).Error
	case status == model.RuleRecordStatusFailed:
		if state.ruleSlug == "" || state.localRuleID == 0 {
			return nil
		}
		return tx.Model(&model.LocalInstanceRule{}).Where("id = ?", state.localRuleID).
			Update("install_status", model.LocalSkillInstallStatusFailed).Error
	default:
		return nil
	}
}

func upsertLocalInstanceRuleFromAck(tx *gorm.DB, state *localAgentRuleAckState, ackVersion string) error {
	displayName := state.rule.Name
	if displayName == "" {
		displayName = state.ruleSlug
	}
	ruleType := state.task.RuleType
	if ruleType == "" {
		ruleType = state.rule.Type
	}
	row := model.LocalInstanceRule{
		InstanceID: state.instance.ID, Slug: state.ruleSlug,
		Version: pickVersion(ackVersion, state.record.Version, state.rule.Version), DisplayName: displayName,
		RuleType: ruleType, Source: model.LocalRuleSourceEnterprise, Scope: state.scope,
		WorkspacePath: state.workspacePath, InstallStatus: model.LocalSkillInstallStatusDistributed,
		InstalledAt: &state.now, LastSeenAt: &state.now,
	}
	if state.localRuleID != 0 {
		return tx.Model(&model.LocalInstanceRule{}).Where("id = ?", state.localRuleID).Updates(map[string]any{
			"version": row.Version, "display_name": row.DisplayName, "rule_type": row.RuleType, "source": row.Source,
			"install_status": model.LocalSkillInstallStatusDistributed, "installed_at": &state.now, "last_seen_at": &state.now,
		}).Error
	}
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "scope"}, {Name: "instance_id"}, {Name: "workspace_path"}, {Name: "slug"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"version", "display_name", "rule_type", "source", "install_status", "installed_at", "last_seen_at", "updated_at",
		}),
	}).Create(&row).Error
}

// ---- rule sync 查询 ---------------------------------------------------------

// ruleCommandItem sync 返回的 rule 命令项。
type ruleCommandItem struct {
	ID            uint   `json:"id"`
	Type          string `json:"type"` // install_prompt_rule / install_rule_rule / uninstall_prompt_rule / uninstall_rule_rule / install_hook_rule / uninstall_hook_rule
	RuleSlug      string `json:"rule_slug"`
	RuleVersion   string `json:"rule_version"`
	RuleType      string `json:"rule_type,omitempty"` // prompt / rule / hook
	DownloadURL   string `json:"download_url,omitempty"`
	Event         string `json:"event,omitempty"` // hook 专属触发时机
	Cmd           string `json:"cmd,omitempty"`   // hook 专属执行命令
	Scope         string `json:"scope"`
	WorkspacePath string `json:"workspace_path,omitempty"`
	ProjectID     uint   `json:"project_id,omitempty"`
}

// queryRuleCommands 查询 rule 的 pending/failed records，组装成 sync commands。
func queryRuleCommands(ctx context.Context, inst *model.Instance) ([]ruleCommandItem, error) {
	type row struct {
		RecordID      uint
		Type          string
		Version       string
		TaskSlug      string
		RuleSlug      string
		COSKey        string
		RuleType      string
		Scope         string
		WorkspacePath string
		BatchID       string
		Event         string
		Cmd           string
	}

	var rows []row
	if err := model.DB(ctx).
		Model(&model.RuleDistributionRecord{}).
		Select(`rule_distribution_records.id AS record_id,
		        rule_distribution_records.type AS type,
		        rule_distribution_records.version AS version,
		        rule_distribution_tasks.slug AS task_slug,
		        enterprise_rules.slug AS rule_slug,
		        enterprise_rules.cos_key AS cos_key,
		        rule_distribution_tasks.rule_type AS rule_type,
		        rule_distribution_tasks.batch_id AS batch_id,
		        enterprise_rules.event AS event,
		        enterprise_rules.cmd AS cmd,
		        local_instance_rules.scope AS scope,
		        local_instance_rules.workspace_path AS workspace_path`).
		Joins("JOIN rule_distribution_tasks ON rule_distribution_tasks.id = rule_distribution_records.task_id").
		Joins("LEFT JOIN enterprise_rules ON enterprise_rules.id = rule_distribution_records.rule_id").
		Joins("LEFT JOIN local_instance_rules ON local_instance_rules.instance_id = rule_distribution_records.instance_id AND local_instance_rules.slug = COALESCE(enterprise_rules.slug, rule_distribution_tasks.slug) AND local_instance_rules.install_status IN ('distributing','failed')").
		Where("rule_distribution_records.instance_id = ? AND rule_distribution_records.status = ?",
			inst.ID, model.RuleRecordStatusPending).
		Order("rule_distribution_records.id ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	localRuleIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		if id := parseLocalScopeBatchID(row.BatchID); id > 0 {
			localRuleIDs = append(localRuleIDs, id)
		}
	}
	localRuleByID := make(map[uint]model.LocalInstanceRule)
	if len(localRuleIDs) > 0 {
		var localRows []model.LocalInstanceRule
		if err := model.DB(ctx).Where("id IN ?", uniqueUintIDs(localRuleIDs)).Find(&localRows).Error; err != nil {
			return nil, err
		}
		for _, row := range localRows {
			localRuleByID[row.ID] = row
		}
	}
	projectByPath := make(map[string]uint)
	var scopeBindings []model.LocalAgentScopeBinding
	if err := model.DB(ctx).Where("instance_id = ? AND scope = ?", inst.ID, model.LocalAgentScopeWorkspace).Find(&scopeBindings).Error; err != nil {
		return nil, err
	}
	for _, binding := range scopeBindings {
		projectByPath[binding.ScopeKey] = binding.ProjectID
	}

	// 去重（同一条 record 可能 JOIN 到多个 local_instance_rules 行导致重复）
	seen := make(map[uint]bool, len(rows))
	cmds := make([]ruleCommandItem, 0, len(rows))
	for _, rr := range rows {
		if seen[rr.RecordID] {
			continue
		}
		seen[rr.RecordID] = true

		slug := rr.RuleSlug
		if slug == "" {
			slug = rr.TaskSlug
		}

		cmdType := model.RuleTypeCommandName(rr.Type, rr.RuleType)
		if cmdType == "" {
			continue
		}

		if localRow, ok := localRuleByID[parseLocalScopeBatchID(rr.BatchID)]; ok {
			rr.Scope, rr.WorkspacePath = localRow.Scope, localRow.WorkspacePath
		}
		rr.Scope, rr.WorkspacePath = normalizeLocalAgentCommandScope(rr.Scope, rr.WorkspacePath)
		item := ruleCommandItem{
			ID:            rr.RecordID,
			Type:          cmdType,
			RuleSlug:      slug,
			RuleVersion:   rr.Version,
			RuleType:      rr.RuleType,
			Scope:         rr.Scope,
			WorkspacePath: rr.WorkspacePath,
			ProjectID:     projectByPath[rr.WorkspacePath],
		}
		// hook 专属字段（仅 type=hook 时前端需要 event + cmd）
		if rr.RuleType == model.EnterpriseRuleTypeHook {
			item.Event = rr.Event
			item.Cmd = rr.Cmd
		}

		// distribute 才需要下载 URL
		if rr.Type == model.RuleTaskTypeDistribute && rr.COSKey != "" {
			url, urlErr := buildSMHDownloadURL(ctx, rr.COSKey, false)
			if urlErr == nil {
				item.DownloadURL = url
			}
		}

		cmds = append(cmds, item)
	}

	return cmds, nil
}
