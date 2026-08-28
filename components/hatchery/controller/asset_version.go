package controller

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm"
)

// ---- 输入/输出类型 ----

// SaveInput 是 RecordAssetSave 的入参（由 HandleAdminAssetSave 事务内传入）。
// 本函数纯消费，不自查 DB；旧绑定/旧 SyncMode 由调用方查出传入。
type SaveInput struct {
	TargetType   string // group | project
	TargetID     uint
	SyncMode     string             // 本次保存的同步模式（initial_only | continuous）
	OldSyncMode  string             // 调用方查出的旧同步模式
	Assets       []AssetBindingItem // 新绑定（全量）
	OldAssets    []AssetBindingItem // 旧绑定（用于 diff）
	OperatorID   uint               // 当前登录管理员 ID
	OperatorName string             // 当前登录管理员姓名（Username）
}

// AssetBindingItem 资产绑定项（save 接口的 assets[] 元素）。
type AssetBindingItem struct {
	AssetType string // skill | rule
	Slug      string
}

// PublishInput 是 PublishAssetVersion 的入参（工具库资产自动变更时由本模块胶水代码传入）。
type PublishInput struct {
	AssetType     string // skill | rule
	Slug          string
	FromVersion   string // 空=新增；删除/范围变更也可能无 from
	ToVersion     string // 空=删除
	TriggerReason string // asset_version_published | asset_deleted | asset_scope_changed
	// AffectedTargets 必填：受影响目标列表（调用方用 model.ListAssetBindingTargets 查好传入）。
	// 每项自带该目标的 SyncMode（调用方查出），本函数不自查。
	AffectedTargets []AssetTarget
}

// AssetTarget 受影响目标（PublishAssetVersion 遍历单位）。
type AssetTarget struct {
	TargetType string
	TargetID   uint
	SyncMode   string // 该目标的同步模式（调用方查出传入）
}

// ---- 版本记录生成（手动） ----

// RecordAssetSave 在 HandleAdminAssetSave 的事务 lambda 内调用，生成版本记录并按同步模式决定是否下发。
// 接收事务句柄，不自己开事务。首建目标 current_version 由 0 自增为 1（即 v1）。
func RecordAssetSave(ctx context.Context, tx *gorm.DB, in SaveInput) error {
	// 1. 原子自增目标 current_version，取更新后值作为本次版本号
	newVersion, err := bumpTargetVersion(tx, in.TargetType, in.TargetID)
	if err != nil {
		return err
	}

	// 2. 计算 diff
	changes := diffAssets(in.OldAssets, in.Assets)
	// 同步模式变化（单独改或连同资产一起改）都要在版本记录里体现
	if in.SyncMode != in.OldSyncMode {
		changes.SyncMode = in.SyncMode
	}

	// 3. 写 changes 明细
	changesJSON, err := json.Marshal(changes)
	if err != nil {
		return err
	}

	// 4. 落 AssetVersionRecord（手动保存）
	record := model.AssetVersionRecord{
		TargetType:    in.TargetType,
		TargetID:      in.TargetID,
		Version:       newVersion,
		TriggerType:   model.TriggerTypeManual,
		TriggerReason: model.TriggerReasonManualSave,
		OperatorType:  "admin",
		OperatorID:    in.OperatorID,
		OperatorName:  in.OperatorName,
		ChangesJSON:   string(changesJSON),
	}
	if err := tx.Create(&record).Error; err != nil {
		return err
	}

	// 5. 按同步模式决定是否下发
	return maybeInstall(ctx, tx, installContext{
		targetType:  in.TargetType,
		targetID:    in.TargetID,
		syncMode:    in.SyncMode,
		oldSyncMode: in.OldSyncMode,
		changes:     changes,
		useSaveAdded: true,      // save 场景按 Added 增量下发
		fullAssets:  in.Assets, // 跳变时全量当前绑定，覆盖 diff
	})
}

// ---- 版本记录生成（自动） ----

// PublishAssetVersion 工具库技能/规范版本更新（发布新版本、删除、范围变更）时由本模块胶水代码调用。
// 遍历 AffectedTargets，对每个目标原子自增版本并落记录；按 SyncMode + 是否有 added/updated 项决定是否下发。
// PublishAssetVersion 为受影响的各目标生成资产版本记录（系统触发）。
// 调用方负责在事务内调用并传入同一个 tx，保证「查受影响目标 → 落版本记录 → 下发」同事务原子。
func PublishAssetVersion(ctx context.Context, tx *gorm.DB, in PublishInput) (uint, error) {
	var lastRecordID uint
	for _, t := range in.AffectedTargets {
		newVersion, err := bumpTargetVersion(tx, t.TargetType, t.TargetID)
		if err != nil {
			return lastRecordID, err
		}
		// 计算该目标的 changes
		changes := assetChangesForPublish(in, t)
		changesJSON, err := json.Marshal(changes)
		if err != nil {
			return lastRecordID, err
		}
		record := model.AssetVersionRecord{
			TargetType:    t.TargetType,
			TargetID:      t.TargetID,
			Version:       newVersion,
			TriggerType:   model.TriggerTypeSystem,
			TriggerReason: in.TriggerReason,
			OperatorType:  "system",
			OperatorID:    0,
			OperatorName:  "",
			ChangesJSON:   string(changesJSON),
		}
		if err := tx.Create(&record).Error; err != nil {
			return lastRecordID, err
		}
		lastRecordID = record.ID
		// 下发判定：仅 continuous 且有 added/updated 项
		if err := maybeInstall(ctx, tx, installContext{
			targetType:  t.TargetType,
			targetID:    t.TargetID,
			syncMode:    t.SyncMode,
			oldSyncMode: "",
			changes:     changes,
			// 自动同步：走 else 分支（Added+Updated）；removed 由 hasNew 判定拦截，不下发
		}); err != nil {
			return lastRecordID, err
		}
	}
	return lastRecordID, nil
}

// ---- 内部：版本自增 ----

// bumpTargetVersion 原子自增目标的 current_version 字段（projects / user_groups），返回更新后的值。
// 用 SQL 表达式 current_version = current_version + 1 避免并发竞态。
// bumpTargetVersion 返回该目标下一个版本号：查 asset_versions 表当前最大 version + 1。
// 版本号是版本记录表的衍生值，不冗余存储在目标表（projects/user_groups）上，
// 避免双写一致性问题。调用方已在事务内，MAX(version) 读取与记录插入同事务提交。
func bumpTargetVersion(tx *gorm.DB, targetType string, targetID uint) (int, error) {
	if targetType != model.TargetTypeProject && targetType != model.TargetTypeGroup {
		return 0, ErrInvalidTargetType
	}
	var maxVer int
	if err := tx.Model(&model.AssetVersionRecord{}).
		Where("target_type = ? AND target_id = ?", targetType, targetID).
		Select("COALESCE(MAX(version), 0)").Scan(&maxVer).Error; err != nil {
		return 0, err
	}
	return maxVer + 1, nil
}

// ---- 内部：diff ----

func diffAssets(oldAssets, newAssets []AssetBindingItem) model.AssetChanges {
	oldMap := make(map[string]AssetBindingItem)
	for _, a := range oldAssets {
		oldMap[a.AssetType+"/"+a.Slug] = a
	}
	newMap := make(map[string]AssetBindingItem)
	for _, a := range newAssets {
		newMap[a.AssetType+"/"+a.Slug] = a
	}
	ch := model.AssetChanges{
		Added:   []model.AssetChangeItem{},
		Removed: []model.AssetChangeItem{},
		Updated: []model.AssetChangeItem{},
	}
	for k, na := range newMap {
		if _, ok := oldMap[k]; !ok {
			ch.Added = append(ch.Added, model.AssetChangeItem{
				AssetType: na.AssetType, Slug: na.Slug, Name: na.Slug,
			})
		}
	}
	for k, oa := range oldMap {
		if _, ok := newMap[k]; !ok {
			ch.Removed = append(ch.Removed, model.AssetChangeItem{
				AssetType: oa.AssetType, Slug: oa.Slug, Name: oa.Slug,
			})
		}
	}
	return ch
}

// assetChangesForPublish 根据 PublishInput 构造单个目标的 changes。
// 自动场景仅 added/updated/removed 之一有值（由 TriggerReason 决定）。
func assetChangesForPublish(in PublishInput, t AssetTarget) model.AssetChanges {
	ch := model.AssetChanges{
		Added:   []model.AssetChangeItem{},
		Removed: []model.AssetChangeItem{},
		Updated: []model.AssetChangeItem{},
	}
	item := model.AssetChangeItem{
		AssetType:   in.AssetType,
		Slug:        in.Slug,
		Name:        in.Slug,
		FromVersion: in.FromVersion,
		ToVersion:   in.ToVersion,
	}
	switch in.TriggerReason {
	case model.TriggerReasonAssetPublished:
		ch.Updated = append(ch.Updated, item) // 版本更新
	case model.TriggerReasonAssetDeleted, model.TriggerReasonScopeChanged:
		ch.Removed = append(ch.Removed, item) // 删除/移出应用范围
	}
	return ch
}

// ---- 安装（下发） ----

type installContext struct {
	ctx         context.Context
	targetType  string
	targetID    uint
	syncMode    string
	oldSyncMode string
	changes     model.AssetChanges
	useSaveAdded bool               // save 场景：按 diff 的 Added 增量下发（不同于跳变全量）
	fullAssets  []AssetBindingItem // 跳变(initial_only→continuous)时全量当前绑定，覆盖 diff
}

// maybeInstall 根据同步模式 + 变更内容决定是否触发下发。
// - initial_only：只记录不下发
// - continuous + 有 added/updated：触发下发
// - 仅 removed（删除/范围缩小）：不下发（仅追加不卸载）
func maybeInstall(ctx context.Context, tx *gorm.DB, ic installContext) error {
	ic.ctx = ctx
	if ic.syncMode != model.SyncModeContinuous {
		return nil // initial_only 或未知模式：只记录不下发
	}
	hasNew := len(ic.changes.Added) > 0 || len(ic.changes.Updated) > 0
	if !hasNew {
		return nil // 仅 removed：不下发
	}
	// 收集需要下发的实例（按 target_type 选 scope）
	// 用 tx.WithContext(ctx) 既保留事务底层连接（in-memory sqlite 测试表存在），
	// 又注入带租户快照的 ctx（避免 applyIdentifierFilter 触发多租户护栏 panic）。
	// 不能用 model.DB(ctx)：会取连接池里另一个连接，in-memory sqlite 表不可见。
	scope := scopeForTarget(ic.targetType)
	instances, err := model.ListLocalAgentInstancesByScopeWithDB(tx.WithContext(ctx), scope, ic.targetID)
	if err != nil {
		return err
	}
	if len(instances) == 0 {
		return nil
	}
	instanceIDs := make([]uint, 0, len(instances))
	for _, ins := range instances {
		instanceIDs = append(instanceIDs, ins.Instance.ID)
	}
	// 组装下发资产清单
	var items []model.AssetChangeItem
	if ic.oldSyncMode != model.SyncModeContinuous && ic.syncMode == model.SyncModeContinuous && len(ic.fullAssets) > 0 {
		// 跳变(initial_only→continuous)：无视 diff，全量重装当前绑定资产
		for _, a := range ic.fullAssets {
			items = append(items, model.AssetChangeItem{AssetType: a.AssetType, Name: a.Slug, Slug: a.Slug})
		}
	} else if ic.useSaveAdded {
		// save 连续模式：下发 added（updated 在 save 路径不产生，added 覆盖新增项）
		items = ic.changes.Added
	} else {
		items = append(items, ic.changes.Added...)
		items = append(items, ic.changes.Updated...)
	}
	if len(items) == 0 {
		return nil
	}
	// 调 seam 真正下发（默认实现见 dispatchInstall，需与现有资产下发通道对齐）
	return dispatchInstallFn(ctx, dispatchRequest{
		targetType:  ic.targetType,
		targetID:    ic.targetID,
		instanceIDs: instanceIDs,
		items:       items,
	})
}

// dispatchInstallFn 是下发动作的 seam（包级变量，测试可替换）。
// 默认实现 dispatchInstall 调用已有的资产下发通道（skill/rule 分发任务）。
var dispatchInstallFn = dispatchInstall

type dispatchRequest struct {
	targetType  string
	targetID    uint
	instanceIDs []uint
	items       []model.AssetChangeItem
}

// dispatchInstall 默认下发实现：按 asset_type 分流调用已有的批量下发纯函数
// distributeSkillBatch / distributeRuleBatch（与 HTTP 接口 HandleDistributeSkill/Rule 共用同一份逻辑）。
// items 中 added 项无 version（用 latest），updated 项带 to_version。
func dispatchInstall(ctx context.Context, req dispatchRequest) error {
	if len(req.instanceIDs) == 0 {
		return nil
	}
	var skillReqs []distributeSkillRequestItem
	var ruleReqs []distributeRuleRequestItem
	for _, it := range req.items {
		switch it.AssetType {
		case model.AssetTypeSkill:
			skillReqs = append(skillReqs, distributeSkillRequestItem{Source: model.SkillSourceEnterprise, Slug: it.Slug, Version: it.ToVersion})
		case model.AssetTypeRule:
			ruleReqs = append(ruleReqs, distributeRuleRequestItem{Slug: it.Slug, Version: it.ToVersion})
		}
	}
	if len(skillReqs) > 0 {
		if _, _, err := distributeSkillBatch(ctx, 0, req.instanceIDs, skillReqs); err != nil {
			return err
		}
	}
	if len(ruleReqs) > 0 {
		if _, _, err := distributeRuleBatch(ctx, 0, req.instanceIDs, ruleReqs); err != nil {
			return err
		}
	}
	return nil
}

// ---- 查询接口 ----

// HandleAdminAssetVersions GET /admin/assets/versions 分页返回版本记录。
// 响应只返回结构化 segments（items 列表），文案由前端拼接；operator 仅 type。
func HandleAdminAssetVersions(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	targetType := r.URL.Query().Get("target_type")
	targetID, _ := strconv.ParseUint(r.URL.Query().Get("target_id"), 10, 64)
	if targetType != model.TargetTypeProject && targetType != model.TargetTypeGroup {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "target_type"))
		return
	}
	if targetID == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "target_id"))
		return
	}
	page, pageSize := parsePagination(r)

	ctx := r.Context()
	tx := model.DB(ctx)
	var total int64
	if err := tx.Model(&model.AssetVersionRecord{}).
		Where("target_type = ? AND target_id = ?", targetType, targetID).
		Count(&total).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgInternalError))
		return
	}
	var records []model.AssetVersionRecord
	if err := tx.Where("target_type = ? AND target_id = ?", targetType, targetID).
		Order("version DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&records).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgInternalError))
		return
	}

	data := make([]map[string]any, 0, len(records))
	for _, rec := range records {
		segs, err := buildSegments(tx, rec)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgInternalError))
			return
		}
		data = append(data, map[string]any{
			"record_id":      rec.ID,
			"version":        rec.Version,
			"trigger_type":   rec.TriggerType,
			"trigger_reason": rec.TriggerReason,
			"operator":       map[string]any{"type": rec.OperatorType, "id": rec.OperatorID, "name": rec.OperatorName},
			"segments":       segs,
			"created_at":     rec.CreatedAt.Format("2006-01-02T15:04:05-07:00"),
		})
	}
	jsonOK(w, map[string]any{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"data":      data,
	})
}

// buildSegments 从 AssetVersionRecord 的 ChangesJSON 构造响应 segments（结构化 items，前端拼文案）。
// items 的 name 在响应时按 slug 现查 skills/enterprise_rules 表的真实名称（落库存 slug）。
func buildSegments(tx *gorm.DB, rec model.AssetVersionRecord) ([]map[string]any, error) {
	var ch model.AssetChanges
	if err := json.Unmarshal([]byte(rec.ChangesJSON), &ch); err != nil {
		return nil, err
	}
	segs := make([]map[string]any, 0, 4)
	switch rec.TriggerReason {
	case model.TriggerReasonManualSave:
		if len(ch.Added) > 0 {
			segs = append(segs, segItems(tx, "added", ch.Added))
		}
		if len(ch.Removed) > 0 {
			segs = append(segs, segItems(tx, "removed", ch.Removed))
		}
		if ch.SyncMode != "" {
			segs = append(segs, map[string]any{"type": "sync_mode", "items": []any{map[string]any{"name": ch.SyncMode}}})
		}
	case model.TriggerReasonAssetPublished:
		if len(ch.Updated) > 0 {
			segs = append(segs, segItems(tx, "version_published", ch.Updated))
		}
	case model.TriggerReasonAssetDeleted:
		if len(ch.Removed) > 0 {
			segs = append(segs, segItems(tx, "deleted", ch.Removed))
		}
	case model.TriggerReasonScopeChanged:
		if len(ch.Removed) > 0 {
			segs = append(segs, segItems(tx, "scope_changed", ch.Removed))
		}
	}
	return segs, nil
}

// assetName 按 (asset_type, slug) 查 skills/enterprise_rules 表的真实名称；查不到返回 slug。
func assetName(tx *gorm.DB, assetType, slug string) string {
	switch assetType {
	case model.AssetTypeSkill:
		var s model.Skill
		if err := tx.Where("slug = ?", slug).First(&s).Error; err == nil && s.Name != "" {
			return s.Name
		}
	case model.AssetTypeRule:
		var r model.EnterpriseRule
		if err := tx.Where("slug = ?", slug).First(&r).Error; err == nil && r.Name != "" {
			return r.Name
		}
	}
	return slug
}

func segItems(tx *gorm.DB, typ string, items []model.AssetChangeItem) map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		m := map[string]any{"asset_type": it.AssetType, "name": assetName(tx, it.AssetType, it.Slug)}
		if it.FromVersion != "" || it.ToVersion != "" {
			m["old_version"] = it.FromVersion
			m["new_version"] = it.ToVersion
		}
		out = append(out, m)
	}
	return map[string]any{"type": typ, "items": out}
}

// ---- 工具 ----

func scopeForTarget(targetType string) string {
	if targetType == model.TargetTypeProject {
		return model.LocalAgentScopeWorkspace
	}
	return model.LocalAgentScopeUser
}

// publishAssetVersionForChange 是工具库 skill/rule 变更（发布新版本、删除、应用范围变更）后
// 触发版本记录的胶水封装：查出受影响目标（带各自 SyncMode）并调用 PublishAssetVersion。
// 该调用为旁路动作——创建/删除主流程已成功，版本记录失败仅记录日志，不阻断响应。
// publishAssetVersionForChange 处理「资产发布新版本 / 删除」的自动版本记录触发。
// 调用方在事务内调用并传入同一个 tx（见各 handler）。
func publishAssetVersionForChange(ctx context.Context, tx *gorm.DB, assetType, slug, fromVersion, toVersion, triggerReason string) {
	if slug == "" {
		return
	}
	targets, err := model.ListAssetBindingTargetsWithDB(tx, assetType, slug)
	if err != nil {
		slog.Warn("查询资产受影响目标失败，跳过版本记录", "asset_type", assetType, "slug", slug, "error", err)
		return
	}
	affected := make([]AssetTarget, 0, len(targets))
	for _, t := range targets {
		syncMode := model.SyncModeContinuous
		if t.TargetType == model.TargetTypeGroup {
			var g model.UserGroup
			if dbErr := tx.Where("id = ?", t.TargetID).First(&g).Error; dbErr == nil {
				syncMode = g.SyncMode
			}
		}
		affected = append(affected, AssetTarget{
			TargetType: t.TargetType,
			TargetID:   t.TargetID,
			SyncMode:   syncMode,
		})
	}
	if _, err := PublishAssetVersion(ctx, tx, PublishInput{
		AssetType:       assetType,
		Slug:            slug,
		FromVersion:     fromVersion,
		ToVersion:       toVersion,
		TriggerReason:   triggerReason,
		AffectedTargets: affected,
	}); err != nil {
		slog.Warn("发布资产版本记录失败", "asset_type", assetType, "slug", slug, "reason", triggerReason, "error", err)
	}
}

// publishScopeRemoval 在应用范围缩小时，删除被移出目标的直接资产绑定并记录版本。
// 只有原本绑定过该资产的目标会产生记录；移除不生成卸载命令。
func publishScopeRemoval(ctx context.Context, tx *gorm.DB, assetType, slug string, removedProjectIDs, removedGroupIDs []uint) error {
	if slug == "" || (len(removedProjectIDs) == 0 && len(removedGroupIDs) == 0) {
		return nil
	}
	targets, err := model.RemoveAssetBindingsForTargets(tx, assetType, slug, removedProjectIDs, removedGroupIDs)
	if err != nil || len(targets) == 0 {
		return err
	}
	affected := make([]AssetTarget, 0, len(targets))
	for _, target := range targets {
		if target.TargetType == model.TargetTypeProject {
			affected = append(affected, AssetTarget{TargetType: target.TargetType, TargetID: target.TargetID, SyncMode: model.SyncModeContinuous})
			continue
		}
		syncMode := model.SyncModeContinuous
		var g model.UserGroup
		if dbErr := tx.Where("id = ?", target.TargetID).First(&g).Error; dbErr == nil {
			syncMode = g.SyncMode
		}
		affected = append(affected, AssetTarget{
			TargetType: target.TargetType,
			TargetID:   target.TargetID,
			SyncMode:   syncMode,
		})
	}
	_, err = PublishAssetVersion(ctx, tx, PublishInput{
		AssetType:       assetType,
		Slug:            slug,
		TriggerReason:   model.TriggerReasonScopeChanged,
		AffectedTargets: affected,
	})
	return err
}

// diffRemovedScope 计算应用范围变更中被移出的存量分组/项目（旧有、新无）。
// visType==all 视为范围扩大（对所有人可见），不算移出，返回空。
func diffRemovedScope(visType string, oldGroupIDs, oldProjectIDs, newGroupIDs, newProjectIDs []uint) (removedGroups, removedProjects []uint) {
	if visType == model.VisibilityAll {
		return nil, nil
	}
	newGroupSet := make(map[uint]bool, len(newGroupIDs))
	for _, id := range newGroupIDs {
		newGroupSet[id] = true
	}
	for _, id := range oldGroupIDs {
		if id != 0 && !newGroupSet[id] {
			removedGroups = append(removedGroups, id)
		}
	}
	newProjectSet := make(map[uint]bool, len(newProjectIDs))
	for _, id := range newProjectIDs {
		newProjectSet[id] = true
	}
	for _, id := range oldProjectIDs {
		if id != 0 && !newProjectSet[id] {
			removedProjects = append(removedProjects, id)
		}
	}
	return removedGroups, removedProjects
}

// ErrInvalidTargetType 表示非法目标类型。
var ErrInvalidTargetType = errors.New("invalid target type")
