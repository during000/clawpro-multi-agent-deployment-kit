package controller

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	skillTaskMaxItems          = 50
	skillDistributionBatchSize = 200
)

type skillExecutionDependencies struct {
	runScript        func(context.Context, string, string, uint64, string, func(string), map[string]string) (string, error)
	buildDownloadURL func(context.Context, string, bool) (string, error)
}

func defaultSkillExecutionDependencies() skillExecutionDependencies {
	return skillExecutionDependencies{
		runScript:        RunScript,
		buildDownloadURL: buildSMHDownloadURL,
	}
}

// skillTaskItem 表示一次下发或卸载请求里要处理的一个技能。
// SkillID、COSZipKey、DownloadURL 是创建任务前补齐的执行信息。
type skillTaskItem struct {
	Index              int
	Source             string
	Slug               string
	Version            string
	SourceSkillsetSlug string
	SkillID            uint
	COSZipKey          string
	DownloadURL        string
}

type distributeSkillRequestItem struct {
	Source             string `json:"source"`
	Slug               string `json:"slug"`
	Version            string `json:"version"`
	SourceSkillsetSlug string `json:"source_skillset_slug"`
}

type skillDistributionTarget struct {
	InstanceID  uint   `gorm:"column:instance_id"`
	InstanceCID string `gorm:"column:cvm_instance_id"`
}

func normalizeSkillDistributionStatuses(statuses []string) ([]string, error) {
	return normalizeDistributionStatuses(
		statuses,
		[]string{"uninstalled", "installed", "outdated", "failed", "upgrade_failed", "uninstall_failed", "uninstall_failed_old"},
		[]string{"installing", "uninstalling"},
	)
}

func normalizeSkillUninstallStatuses(statuses []string) ([]string, error) {
	return normalizeDistributionStatuses(
		statuses,
		[]string{"installed", "outdated", "upgrade_failed", "uninstall_failed", "uninstall_failed_old"},
		[]string{"installing", "uninstalling"},
	)
}

type uninstallSkillRequestItem struct {
	Source             string `json:"source"`
	Slug               string `json:"slug"`
	SourceSkillsetSlug string `json:"source_skillset_slug"`
}

// skillBatchResultItem 是批量下发/卸载响应中单个 skills[] 条目的提交结果。
type skillBatchResultItem struct {
	Index         int    `json:"index"`
	Source        string `json:"source"`
	Slug          string `json:"slug"`
	Version       string `json:"version,omitempty"`
	Status        string `json:"status"`
	TaskID        uint   `json:"task_id,omitempty"`
	InstanceCount int    `json:"instance_count,omitempty"`
	Reason        string `json:"reason,omitempty"`
	Message       string `json:"message,omitempty"`
}

type skillInstanceInfo struct {
	ID               uint
	InstanceId       string
	RuntimeUser      string
	AgentType        string
	Source           string
	CurrentOperation string
}

// isLocalAgentInstance 判断当前实例是否为本地 agent 实例（workbuddy / codebuddy 等）。
// 本地实例不需要走 CVM、TAT 等云侧链路，也未注册到内置 agentTypesMap，
// 但其 skill 能力由 reporter ack 链路保证，所以不能因 agent_type 不支持 skill 而一刑切掉。
func (info skillInstanceInfo) isLocalAgentInstance() bool {
	return info.Source == model.InstanceSourceLocal
}

type publicSkillDistributeDeps struct {
	commonStorageClient func(context.Context) (StorageClient, error)
	commonDownloadURL   func(context.Context, string, bool) (string, error)
}

// createSkillTaskItem 创建 handler 内部使用的技能任务条目。
// 这里仅做字段 trim 和默认 source 填充；参数合法性由 validateSkillItem 负责。
func createSkillTaskItem(index int, source, slug, version, sourceSkillsetSlug string) skillTaskItem {
	source = strings.TrimSpace(source)
	if source == "" {
		source = model.SkillSourceEnterprise
	}
	return skillTaskItem{
		Index:              index,
		Source:             source,
		Slug:               strings.TrimSpace(slug),
		Version:            strings.TrimSpace(version),
		SourceSkillsetSlug: strings.TrimSpace(sourceSkillsetSlug),
	}
}

func validateSkillItem(item skillTaskItem) (reason string, err *hcommon.RichError) {
	if item.Source != model.SkillSourceEnterprise && item.Source != model.SkillSourcePublic {
		return "unsupported_source", hcommon.I18nError(i18n.MsgUnsupportedSkillSource, item.Source)
	}
	if item.Slug == "" {
		return "slug_required", hcommon.I18nError(i18n.MsgBadRequestParamRequired, "slug")
	}
	if item.Source == model.SkillSourceEnterprise && item.SourceSkillsetSlug != "" {
		return "invalid_source_skillset", hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "source_skillset_slug")
	}
	return "", nil
}

// loadInstancesSupportingSkillTasks 查询请求中的实例，并只保留支持技能任务的实例。
// 返回值依次为：可下发/卸载的实例 ID、实例执行信息映射、因 Agent 类型不支持技能而跳过的实例数量。
//
// 本地 agent 实例（source=local）目前包括 workbuddy / codebuddy 等，未注册到
// 内置 agentTypesMap，AgentTypeSupportsSkill 会返 false；但本地实例的 skill 能力
// 由 reporter ack 链路保证，不依赖该能力位。二期正式注册本地 agent 类型后可移除。
func loadInstancesSupportingSkillTasks(ctx context.Context, instanceIDs []uint) ([]uint, map[uint]skillInstanceInfo, int, error) {
	var instInfos []skillInstanceInfo
	if err := model.DB(ctx).Model(&model.Instance{}).
		Select("id, instance_id, runtime_user, agent_type, source, current_operation").
		Where("id IN ?", instanceIDs).
		Scan(&instInfos).Error; err != nil {
		return nil, nil, 0, err
	}
	allAgentTypes := model.GetAllAgentTypesMap(ctx)
	infoMap := make(map[uint]skillInstanceInfo, len(instInfos))
	validIDs := make([]uint, 0, len(instInfos))
	skipped := 0
	for _, info := range instInfos {
		if model.IsResourceAdjustmentOperation(info.CurrentOperation) {
			skipped++
			continue
		}
		if !info.isLocalAgentInstance() && !model.AgentTypeSupportsSkillByMap(info.AgentType, allAgentTypes) {
			skipped++
			continue
		}
		infoMap[info.ID] = info
		validIDs = append(validIDs, info.ID)
	}
	return validIDs, infoMap, skipped, nil
}

func newSkillTaskBatchID(now time.Time) string {
	var suffix [6]byte
	if _, err := rand.Read(suffix[:]); err == nil {
		return fmt.Sprintf("skilldist-%s-%x", now.UTC().Format("20060102150405"), suffix)
	}
	return fmt.Sprintf("skilldist-%d", now.UTC().UnixNano())
}

// prepareDistributeSkillItem 把请求里的技能条目补齐为可执行的下发条目。
// 企业技能从本地 Skill 表解析版本和 zip 路径；公共技能先下载公共 zip、转存到 common space，再生成实例可访问的下载 URL。
func prepareDistributeSkillItem(ctx context.Context, item skillTaskItem) (skillTaskItem, string, *hcommon.RichError) {
	if reason, richErr := validateSkillItem(item); richErr != nil {
		return skillTaskItem{}, reason, richErr
	}
	switch item.Source {
	case model.SkillSourceEnterprise:
		skill, richErr := resolveEnterpriseSkillForOperation(ctx, item.Slug, item.Version)
		if richErr != nil {
			return skillTaskItem{}, "skill_not_found", richErr
		}
		// 不允许下发 pending_review 的技能（offline 允许下发，管控端管理能力与正常一致）
		if skill.Status == model.SkillStatusPendingReview {
			return skillTaskItem{}, "skill_not_published", hcommon.I18nError(i18n.MsgSkillNotPublished)
		}
		item.SkillID = skill.ID
		item.Version = skill.Version
		item.COSZipKey = fmt.Sprintf("%s/%s-%s.zip", skill.Slug, skill.Slug, skill.Version)
		return item, "", nil
	case model.SkillSourcePublic:
		publicItem, reason, richErr := preparePublicSkillForDistribute(ctx, item, publicSkillDistributeDeps{
			commonStorageClient: GetCommonStorageClient,
			commonDownloadURL:   BuildCommonSMHDownloadURL,
		})
		if richErr != nil {
			return skillTaskItem{}, reason, richErr
		}
		return publicItem, "", nil
	}
	return skillTaskItem{}, "unsupported_source", hcommon.I18nError(i18n.MsgUnsupportedSkillSource, item.Source)
}

// prepareUninstallSkillItem 把请求里的技能条目补齐为可执行的卸载条目。
// 企业技能需要解析本地 SkillID 和当前版本用于记录/失败状态判断；公共技能没有本地 Skill 记录，保留 slug/version 维度处理。
func prepareUninstallSkillItem(ctx context.Context, item skillTaskItem) (skillTaskItem, string, *hcommon.RichError) {
	if reason, richErr := validateSkillItem(item); richErr != nil {
		return skillTaskItem{}, reason, richErr
	}
	switch item.Source {
	case model.SkillSourceEnterprise:
		skill, richErr := resolveEnterpriseSkillForOperation(ctx, item.Slug, "")
		if richErr != nil {
			return skillTaskItem{}, "skill_not_found", richErr
		}
		item.SkillID = skill.ID
		item.Version = skill.Version
		return item, "", nil
	case model.SkillSourcePublic:
		return item, "", nil
	}
	return skillTaskItem{}, "unsupported_source", hcommon.I18nError(i18n.MsgUnsupportedSkillSource, item.Source)
}

// resolveEnterpriseSkillForOperation 解析企业技能的具体版本。
// version 为空或 latest 时沿用 release 行为取本地最新版本；传具体版本时要求精确命中。
func resolveEnterpriseSkillForOperation(ctx context.Context, slug, version string) (model.Skill, *hcommon.RichError) {
	var skill model.Skill
	if version == "" || version == "latest" {
		if err := model.DB(ctx).Where("slug = ?", slug).Order("version_major DESC, version_minor DESC, version_patch DESC").First(&skill).Error; err != nil {
			return skill, hcommon.I18nError(i18n.MsgSkillNotExist)
		}
		return skill, nil
	}
	if err := model.DB(ctx).Where("slug = ? AND version = ?", slug, version).First(&skill).Error; err != nil {
		return skill, hcommon.I18nError(i18n.MsgPluginVersionNotFoundDetail, slug, version)
	}
	return skill, nil
}

// preparePublicSkillForDistribute 准备公共技能下发所需的 zip 和下载地址。
// 公共技能不能直接使用企业技能的本地 COS 路径，必须先从 SkillHub 下载，再上传到 common space，后续实例安装统一走 SMH 下载 URL。
func preparePublicSkillForDistribute(ctx context.Context, item skillTaskItem, deps publicSkillDistributeDeps) (skillTaskItem, string, *hcommon.RichError) {
	downloadURL := buildSkillHubPublicDownloadURL(item.Slug, item.Version)
	zipData, finalURL, richErr := downloadSkillZipWithFinalURL(downloadURL, i18n.MsgDownloadPublicZipFail, i18n.MsgReadPublicZipFail)
	if richErr != nil {
		return skillTaskItem{}, "download_public_zip_failed", richErr
	}
	resolvedVersion := item.Version
	if resolvedVersion == "latest" {
		resolvedVersion = ""
	}
	if resolvedVersion == "" {
		resolvedVersion = publicSkillVersionFromDownloadURL(finalURL)
		if resolvedVersion == "" {
			return skillTaskItem{}, "version_required", hcommon.I18nError(i18n.MsgBadRequestParamRequired, "version")
		}
	}

	commonClient, err := deps.commonStorageClient(ctx)
	if err != nil {
		return skillTaskItem{}, "common_storage_failed", hcommon.I18nRichError(err, i18n.MsgGetCommonStorageClientFail)
	}
	cosZipKey := fmt.Sprintf("public-skills/%s/%s-%s.zip", item.Slug, item.Slug, resolvedVersion)
	if err := commonClient.Upload(cosZipKey, zipData, "application/zip"); err != nil {
		return skillTaskItem{}, "upload_public_zip_failed", hcommon.I18nRichError(err, i18n.MsgUploadSkillZipFail)
	}
	zipData = nil
	smhURL, err := deps.commonDownloadURL(ctx, cosZipKey, true)
	if err != nil {
		return skillTaskItem{}, "download_url_failed", hcommon.I18nRichError(err, i18n.MsgSkillDownloadURLGenFail, err.Error())
	}
	item.Version = resolvedVersion
	item.DownloadURL = smhURL
	return item, "", nil
}

func skillTaskItemLockKey(item skillTaskItem) string {
	if item.Source == model.SkillSourceEnterprise && item.SkillID > 0 {
		return fmt.Sprintf("skill_dist:%d", item.SkillID)
	}
	return fmt.Sprintf("skill_dist:%s:%s", item.Source, item.Slug)
}

func skillBatchResultItemFailed(item skillTaskItem, reason string, richErr *hcommon.RichError, ctx context.Context) skillBatchResultItem {
	return skillBatchResultItem{
		Index:   item.Index,
		Source:  item.Source,
		Slug:    item.Slug,
		Version: item.Version,
		Status:  "failed",
		Reason:  reason,
		Message: hcommon.ErrorMessageWithCtx(ctx, richErr),
	}
}

// failPreviousPendingSkillDistribute 在发起新一次下发前，检查同一 slug+source 上一次
// distribute 任务是否仍有 pending 记录（reporter 尚未拉走处理）。若存在，则只把上一次
// 任务中本次请求涉及的相同 instance（instanceIDs）的 pending 记录置为 failed，避免残留
// pending 被后续 sync 重复处理；同时把这些记录的数量累加到上一次任务的 failed 计数。
//
// 只处理与本次请求相同 instance_id 的 pending 记录——一个 task 可能给多个实例下发，
// 其它实例可能早已安装成功，不能因个别实例还 pending 就把整个 task 判失败，也不能把
// 本次请求没覆盖到的实例的 pending 误判。
//
// 调用方须已持有该 item 的 distribute 分布式锁（HandleDistributeSkill / 批量路径里已加），
// 因此这里不需要额外的并发保护。
//
// 只处理 distribute 类型；uninstall 的 pending 不在本函数职责内。
func failPreviousPendingSkillDistribute(ctx context.Context, item skillTaskItem, instanceIDs []uint) error {
	if len(instanceIDs) == 0 {
		return nil
	}
	var prevTask model.SkillDistributionTask
	// 取该 slug+source 最新的 distribute 任务（不论其 task 状态）。
	q := model.DB(ctx).Where("slug = ? AND source = ? AND type = ?", item.Slug, item.Source, model.TaskTypeDistribute)
	if item.Source == model.SkillSourceEnterprise && item.SkillID > 0 {
		q = q.Where("skill_id = ?", item.SkillID)
	}
	if err := q.Order("id DESC").Limit(1).First(&prevTask).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	// 该任务中，本次请求涉及的相同实例是否还有 pending 记录
	var pendingCnt int64
	if err := model.DB(ctx).Model(&model.SkillDistributionRecord{}).
		Where("task_id = ? AND skill_id = ? AND type = ? AND status = ? AND instance_id IN ?",
			prevTask.ID, item.SkillID, model.TaskTypeDistribute, model.RecordStatusPending, instanceIDs).
		Count(&pendingCnt).Error; err != nil {
		return err
	}
	if pendingCnt == 0 {
		return nil
	}

	now := time.Now()
	// 只把本次请求涉及的相同实例的 pending record 置为 failed。
	if err := model.DB(ctx).Model(&model.SkillDistributionRecord{}).
		Where("task_id = ? AND skill_id = ? AND type = ? AND status = ? AND instance_id IN ?",
			prevTask.ID, item.SkillID, model.TaskTypeDistribute, model.RecordStatusPending, instanceIDs).
		Updates(map[string]interface{}{
			"status":     model.RecordStatusFailed,
			"error":      i18n.T(ctx, i18n.MsgSkillNewVersionDistributed),
			"updated_at": now,
		}).Error; err != nil {
		return err
	}
	// 把判失败的记录数量累加到上一次任务的 failed 计数。
	if err := model.DB(ctx).Model(&model.SkillDistributionTask{}).
		Where("id = ?", prevTask.ID).
		UpdateColumn("failed", gorm.Expr("failed + ?", pendingCnt)).Error; err != nil {
		return err
	}
	slog.Info("技能下发：上一次 pending 记录已判失败", "slug", item.Slug, "source", item.Source, "prev_task_id", prevTask.ID, "pending", pendingCnt)
	return nil
}

func createSkillTaskAndRecords(ctx context.Context, item skillTaskItem, action string, operatorID uint, instanceIDs []uint, infoMap map[uint]skillInstanceInfo, batchID string, createdAt time.Time) (model.SkillDistributionTask, []model.SkillDistributionRecord, error) {
	var task model.SkillDistributionTask
	var records []model.SkillDistributionRecord
	err := model.DB(ctx).Transaction(func(tx *gorm.DB) error {
		task = model.SkillDistributionTask{
			CreatedAt:          createdAt,
			UpdatedAt:          createdAt,
			SkillID:            item.SkillID,
			Version:            item.Version,
			Source:             item.Source,
			Slug:               item.Slug,
			SourceSkillsetSlug: item.SourceSkillsetSlug,
			BatchID:            batchID,
			OperatorID:         operatorID,
			Total:              len(instanceIDs),
			Status:             "running",
			Type:               action,
		}
		if err := tx.Create(&task).Error; err != nil {
			return err
		}

		records = make([]model.SkillDistributionRecord, 0, len(instanceIDs))
		for _, instID := range instanceIDs {
			info := infoMap[instID]
			records = append(records, model.SkillDistributionRecord{
				TaskID:      task.ID,
				SkillID:     item.SkillID,
				InstanceID:  instID,
				InstanceCID: info.InstanceId,
				Version:     item.Version,
				Status:      "pending",
				Type:        action,
			})
		}
		if err := tx.Create(&records).Error; err != nil {
			return err
		}
		return nil
	})
	return task, records, err
}

func createSkillSelectAllTask(
	ctx context.Context,
	item skillTaskItem,
	action string,
	operatorID uint,
	batchID string,
	selection distributionSelection,
	createdAt time.Time,
) (model.SkillDistributionTask, int, error) {
	var statuses []string
	var err error
	if action == model.TaskTypeUninstall {
		statuses, err = normalizeSkillUninstallStatuses(selection.Statuses)
	} else {
		statuses, err = normalizeSkillDistributionStatuses(selection.Statuses)
	}
	if err != nil {
		return model.SkillDistributionTask{}, 0, err
	}

	var baseQuery *gorm.DB
	switch {
	case item.Source == model.SkillSourceEnterprise:
		var skillIDs []uint
		if err := model.DB(ctx).Model(&model.Skill{}).
			Where("slug = ?", item.Slug).
			Pluck("id", &skillIDs).Error; err != nil {
			return model.SkillDistributionTask{}, 0, err
		}
		baseQuery = model.BuildSkillInstanceQuery(ctx, skillIDs, item.Version, item.Slug).
			Scopes(model.FilterSkillInstallStatuses(item.Version, statuses))
	case action == model.TaskTypeUninstall:
		baseQuery = buildPublicSkillInstanceQueryWithoutVersion(ctx, item.Slug).
			Scopes(filterPublicSkillInstallStatusesWithoutVersion(statuses))
	default:
		baseQuery = buildPublicSkillInstanceQuery(ctx, item.Slug, item.Version).
			Scopes(model.FilterSkillInstallStatuses(item.Version, statuses))
	}
	baseQuery = baseQuery.Where(
		"instances.source = ? OR instances.agent_type IN ?",
		model.InstanceSourceLocal,
		model.GetSkillSupportedAgentTypes(ctx),
	)
	baseQuery = model.FilterInstancesByUserGroups(ctx, baseQuery, selection.GroupIDs)
	baseQuery = applyDistributionSearch(baseQuery, selection.Search)

	var task model.SkillDistributionTask
	var afterID uint
	total := 0
	for {
		var rows []skillDistributionTarget
		if err := baseQuery.Session(&gorm.Session{}).
			Where("instances.id > ?", afterID).
			Order(clause.OrderByColumn{
				Column:  clause.Column{Table: "instances", Name: "id"},
				Reorder: true,
			}).
			Limit(skillDistributionBatchSize).
			Scan(&rows).Error; err != nil {
			cleanupSkillSelectAllTask(hcommon.DetachContext(ctx), task.ID)
			return model.SkillDistributionTask{}, 0, err
		}
		if len(rows) == 0 {
			break
		}

		targets := make([]skillDistributionTarget, 0, len(rows))
		seen := make(map[uint]struct{}, len(rows))
		for _, row := range rows {
			if row.InstanceID > afterID {
				afterID = row.InstanceID
			}
			if _, ok := seen[row.InstanceID]; ok {
				continue
			}
			seen[row.InstanceID] = struct{}{}
			targets = append(targets, row)
		}
		if len(targets) == 0 {
			continue
		}

		if task.ID == 0 {
			task = model.SkillDistributionTask{
				CreatedAt:          createdAt,
				UpdatedAt:          createdAt,
				SkillID:            item.SkillID,
				Version:            item.Version,
				Source:             item.Source,
				Slug:               item.Slug,
				SourceSkillsetSlug: item.SourceSkillsetSlug,
				BatchID:            batchID,
				OperatorID:         operatorID,
				Status:             model.TaskStatusRunning,
				Type:               action,
			}
			if err := model.DB(ctx).Create(&task).Error; err != nil {
				cleanupSkillSelectAllTask(hcommon.DetachContext(ctx), task.ID)
				return model.SkillDistributionTask{}, 0, err
			}
		}

		records := make([]model.SkillDistributionRecord, 0, len(targets))
		for _, target := range targets {
			records = append(records, model.SkillDistributionRecord{
				TaskID:      task.ID,
				SkillID:     item.SkillID,
				InstanceID:  target.InstanceID,
				InstanceCID: target.InstanceCID,
				Version:     item.Version,
				Status:      model.RecordStatusPending,
				Type:        action,
			})
		}
		if err := model.DB(ctx).Create(&records).Error; err != nil {
			cleanupSkillSelectAllTask(hcommon.DetachContext(ctx), task.ID)
			return model.SkillDistributionTask{}, 0, err
		}
		total += len(targets)
	}
	if total == 0 {
		if action == model.TaskTypeUninstall {
			return model.SkillDistributionTask{}, 0, hcommon.I18nError(i18n.MsgSkillStoreNoValidUninstall)
		}
		return model.SkillDistributionTask{}, 0, hcommon.I18nError(i18n.MsgSkillStoreNoValidInstall)
	}
	task.Total = total
	if err := model.DB(ctx).Model(&task).Update("total", total).Error; err != nil {
		cleanupSkillSelectAllTask(hcommon.DetachContext(ctx), task.ID)
		return model.SkillDistributionTask{}, 0, err
	}
	return task, total, nil
}

func cleanupSkillSelectAllTask(ctx context.Context, taskID uint) {
	if taskID == 0 {
		return
	}
	if err := model.DB(ctx).Where("task_id = ?", taskID).Delete(&model.SkillDistributionRecord{}).Error; err != nil {
		slog.Error("[SkillSelectAll] 清理下发记录失败", "task_id", taskID, "error", err)
	}
	if err := model.DB(ctx).Delete(&model.SkillDistributionTask{}, taskID).Error; err != nil {
		slog.Error("[SkillSelectAll] 清理下发任务失败", "task_id", taskID, "error", err)
	}
}

func failPendingSkillDistributionRecords(ctx context.Context, taskID uint, cause error) (int, error) {
	result := model.DB(ctx).Model(&model.SkillDistributionRecord{}).
		Where("task_id = ? AND status = ?", taskID, model.RecordStatusPending).
		Updates(map[string]interface{}{
			"status": model.RecordStatusFailed,
			"error":  hcommon.ErrorMessageWithCtx(ctx, cause),
		})
	return int(result.RowsAffected), result.Error
}

type skillSelectAllJob struct {
	record model.SkillDistributionRecord
	info   skillInstanceInfo
}

func recoverSkillSelectAllTaskPanic(ctx context.Context, task model.SkillDistributionTask) {
	recovered := recover()
	if recovered == nil {
		return
	}
	cause := fmt.Errorf("panic: %v", recovered)
	slog.Error("[SkillSelectAll] task panic", "task_id", task.ID, "panic", recovered, "stack", string(debug.Stack()))
	if _, err := failPendingSkillDistributionRecords(ctx, task.ID, cause); err != nil {
		slog.Error("[SkillSelectAll] 收敛 panic 任务失败", "task_id", task.ID, "error", err)
	}
	var success int64
	if err := model.DB(ctx).Model(&model.SkillDistributionRecord{}).
		Where("task_id = ? AND status = ?", task.ID, model.RecordStatusSuccess).
		Count(&success).Error; err != nil {
		slog.Error("[SkillSelectAll] 统计 panic 任务成功记录失败", "task_id", task.ID, "error", err)
	}
	failed := task.Total - int(success)
	if failed < 0 {
		failed = 0
	}
	if err := model.DB(ctx).Model(&task).Updates(map[string]interface{}{
		"status":  model.TaskStatusCompleted,
		"success": int(success),
		"failed":  failed,
	}).Error; err != nil {
		slog.Error("[SkillSelectAll] 更新 panic 任务状态失败", "task_id", task.ID, "error", err)
	}
}

func runSkillSelectAllTask(
	ctx context.Context,
	item skillTaskItem,
	task model.SkillDistributionTask,
) {
	var downloadURL string
	var downloadURLErr error
	var downloadURLOnce sync.Once
	resolveDownloadURL := func() (string, error) {
		if item.DownloadURL != "" {
			return item.DownloadURL, nil
		}
		downloadURLOnce.Do(func() {
			downloadURL, downloadURLErr = buildSMHDownloadURL(ctx, item.COSZipKey, true)
		})
		return downloadURL, downloadURLErr
	}

	maxConcurrency := model.GetSiteConfig(ctx).SkillDistributeConcurrency
	if maxConcurrency <= 0 {
		maxConcurrency = 100
	}
	successCount := 0
	failedCount := 0
	var afterID uint
	for {
		records, jobs, loadFailed, err := loadSkillSelectAllBatch(ctx, task.ID, afterID)
		if err != nil {
			slog.Error("[SkillSelectAll] 分批读取下发记录失败", "task_id", task.ID, "error", err)
			failed, convergeErr := failPendingSkillDistributionRecords(ctx, task.ID, err)
			if convergeErr != nil {
				slog.Error("[SkillSelectAll] 收敛未处理记录失败", "task_id", task.ID, "error", convergeErr)
			} else {
				failedCount += failed
			}
			break
		}
		if len(records) == 0 {
			break
		}
		afterID = records[len(records)-1].ID
		failedCount += loadFailed
		success, failed := executeSkillSelectAllBatch(ctx, item, task.Type, jobs, resolveDownloadURL, maxConcurrency)
		successCount += success
		failedCount += failed
	}

	if err := model.DB(ctx).Model(&task).Updates(map[string]interface{}{
		"status":  model.TaskStatusCompleted,
		"success": successCount,
		"failed":  failedCount,
	}).Error; err != nil {
		slog.Error("[SkillSelectAll] 更新任务统计失败", "task_id", task.ID, "error", err)
	}
	if task.Type == model.TaskTypeDistribute && successCount > 0 && item.Source == model.SkillSourceEnterprise && item.SkillID > 0 {
		if err := model.DB(ctx).Model(&model.Skill{}).Where("id = ?", item.SkillID).
			UpdateColumn("distribute_count", gorm.Expr("distribute_count + ?", successCount)).Error; err != nil {
			slog.Error("[SkillSelectAll] 更新下发计数失败", "skill_id", item.SkillID, "error", err)
		}
	}
}

func loadSkillSelectAllBatch(
	ctx context.Context,
	taskID uint,
	afterID uint,
) ([]model.SkillDistributionRecord, []skillSelectAllJob, int, error) {
	var records []model.SkillDistributionRecord
	if err := model.DB(ctx).
		Where("task_id = ? AND id > ? AND status = ?", taskID, afterID, model.RecordStatusPending).
		Order("id ASC").
		Limit(skillDistributionBatchSize).
		Find(&records).Error; err != nil {
		return nil, nil, 0, err
	}
	if len(records) == 0 {
		return nil, nil, 0, nil
	}

	instanceIDs := make([]uint, 0, len(records))
	for _, record := range records {
		instanceIDs = append(instanceIDs, record.InstanceID)
	}
	_, infoMap, _, err := loadInstancesSupportingSkillTasks(ctx, instanceIDs)
	if err != nil {
		slog.Error("[SkillSelectAll] 批量加载实例信息失败", "task_id", taskID, "error", err)
		for i := range records {
			if updateErr := model.DB(ctx).Model(&records[i]).Updates(map[string]interface{}{
				"status": model.RecordStatusFailed,
				"error":  hcommon.ErrorMessageWithCtx(ctx, hcommon.I18nRichError(err, i18n.MsgPluginQueryInstanceInfo)),
			}).Error; updateErr != nil {
				slog.Error("[SkillSelectAll] 更新实例查询失败记录失败", "record_id", records[i].ID, "error", updateErr)
			}
		}
		return records, nil, len(records), nil
	}

	jobs := make([]skillSelectAllJob, 0, len(records))
	failed := 0
	for i := range records {
		info, ok := infoMap[records[i].InstanceID]
		if !ok {
			if updateErr := model.DB(ctx).Model(&records[i]).Updates(map[string]interface{}{
				"status": model.RecordStatusFailed,
				"error":  i18n.T(ctx, i18n.MsgInstanceNotFound),
			}).Error; updateErr != nil {
				slog.Error("[SkillSelectAll] 更新缺失实例记录失败", "record_id", records[i].ID, "error", updateErr)
			}
			failed++
			continue
		}
		if info.isLocalAgentInstance() {
			continue
		}
		jobs = append(jobs, skillSelectAllJob{record: records[i], info: info})
	}
	return records, jobs, failed, nil
}

func resolveSkillSelectAllFailedStatus(ctx context.Context, item skillTaskItem, taskType string, record model.SkillDistributionRecord) string {
	if taskType == model.TaskTypeUninstall {
		if item.Source == model.SkillSourcePublic {
			return model.ResolvePublicSkillUninstallFailedStatus(ctx, record.InstanceID, item.Slug, item.Version)
		}
		return model.ResolveUninstallFailedStatus(ctx, record.InstanceID, []uint{item.SkillID}, item.Version)
	}
	if item.Source == model.SkillSourcePublic {
		return model.ResolvePublicSkillDistributeFailedStatus(ctx, record.InstanceID, item.Slug, record.ID)
	}
	return model.ResolveDistributeFailedStatus(ctx, record.InstanceID, []uint{item.SkillID})
}

func executeSkillSelectAllBatch(
	ctx context.Context,
	item skillTaskItem,
	taskType string,
	jobs []skillSelectAllJob,
	resolveDownloadURL func() (string, error),
	maxConcurrency int,
) (int, int) {
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	defer wg.Wait()
	var mu sync.Mutex
	successCount := 0
	failedCount := 0
	for _, current := range jobs {
		sem <- struct{}{}
		wg.Add(1)
		go func(current skillSelectAllJob) {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				recovered := recover()
				if recovered == nil {
					return
				}
				cause := fmt.Errorf("panic: %v", recovered)
				slog.Error("[SkillSelectAll] record panic", "record_id", current.record.ID, "panic", recovered, "stack", string(debug.Stack()))
				failedStatus := resolveSkillSelectAllFailedStatus(ctx, item, taskType, current.record)
				if err := model.DB(ctx).Model(&current.record).Updates(map[string]interface{}{
					"status": failedStatus,
					"error":  hcommon.ErrorMessageWithCtx(ctx, cause),
				}).Error; err != nil {
					slog.Error("[SkillSelectAll] 收敛 panic 记录失败", "record_id", current.record.ID, "error", err)
				}
				mu.Lock()
				failedCount++
				mu.Unlock()
			}()

			var err error
			if taskType == model.TaskTypeUninstall {
				var scriptName string
				scriptName, err = ResolveScript(ctx, "uninstall_skill", current.info.AgentType)
				if err == nil {
					_, err = RunScript(ctx, current.record.InstanceCID, scriptName, 60, current.info.RuntimeUser, nil, map[string]string{
						"skill_slug": item.Slug,
					})
				}
			} else {
				var url string
				url, err = resolveDownloadURL()
				if err == nil {
					var scriptName string
					scriptName, err = ResolveScript(ctx, "install_skill_from_smh", current.info.AgentType)
					if err == nil {
						_, err = RunScript(ctx, current.record.InstanceCID, scriptName, 120, current.info.RuntimeUser, nil, map[string]string{
							"download_url":  url,
							"skill_slug":    item.Slug,
							"skill_version": item.Version,
						})
					}
				}
			}
			if err != nil {
				failedStatus := resolveSkillSelectAllFailedStatus(ctx, item, taskType, current.record)
				if updateErr := model.DB(ctx).Model(&current.record).Updates(map[string]interface{}{
					"status": failedStatus,
					"error":  err.Error(),
				}).Error; updateErr != nil {
					slog.Error("[SkillSelectAll] 更新失败记录失败", "record_id", current.record.ID, "error", updateErr)
				}
				mu.Lock()
				failedCount++
				mu.Unlock()
				return
			}
			if updateErr := model.DB(ctx).Model(&current.record).Update("status", model.RecordStatusSuccess).Error; updateErr != nil {
				slog.Error("[SkillSelectAll] 更新成功记录失败", "record_id", current.record.ID, "error", updateErr)
			}
			mu.Lock()
			successCount++
			mu.Unlock()
		}(current)
	}
	wg.Wait()
	return successCount, failedCount
}

func buildSkillDistributeExecution(
	ctx context.Context,
	item skillTaskItem,
	task model.SkillDistributionTask,
	records []model.SkillDistributionRecord,
	lock *model.DistLock,
	infoMap map[uint]skillInstanceInfo,
	deps skillExecutionDependencies,
) (SkillTaskConfig, SkillTaskExecutor) {
	var downloadURL string
	var downloadURLErr error
	var downloadURLOnce sync.Once
	resolveDownloadURL := func(ctx context.Context) (string, error) {
		if item.DownloadURL != "" {
			return item.DownloadURL, nil
		}
		downloadURLOnce.Do(func() {
			downloadURL, downloadURLErr = deps.buildDownloadURL(ctx, item.COSZipKey, true)
		})
		return downloadURL, downloadURLErr
	}

	cfg := SkillTaskConfig{
		Ctx:     ctx,
		Task:    task,
		Records: records,
		Lock:    lock,
		Slug:    item.Slug,
		OnFailed: func(ctx context.Context, record model.SkillDistributionRecord) string {
			if item.Source == model.SkillSourcePublic {
				return model.ResolvePublicSkillDistributeFailedStatus(ctx, record.InstanceID, item.Slug, record.ID)
			}
			states, err := model.ListDistributedSkillStates(ctx, record.InstanceID, []string{item.Slug})
			if err == nil {
				if state, ok := states[item.Slug]; ok && state.Installed {
					return model.RecordStatusUpgradeFailed
				}
			}
			return model.RecordStatusFailed
		},
		OnComplete: func(ctx context.Context, successCount, _ int) {
			if successCount > 0 && item.Source == model.SkillSourceEnterprise && item.SkillID > 0 {
				model.DB(ctx).Model(&model.Skill{}).Where("id = ?", item.SkillID).
					UpdateColumn("distribute_count", gorm.Expr("distribute_count + ?", successCount))
			}
		},
	}
	executor := func(ctx context.Context, record model.SkillDistributionRecord) error {
		downloadURL, err := resolveDownloadURL(ctx)
		if err != nil {
			return hcommon.I18nRichError(err, i18n.MsgSkillDownloadURLGenFail, err.Error())
		}
		info := infoMap[record.InstanceID]
		scriptName, resolveErr := ResolveScript(ctx, "install_skill_from_smh", info.AgentType)
		if resolveErr != nil {
			return hcommon.I18nError(i18n.MsgUnsupportedAgentType, info.AgentType)
		}
		_, err = deps.runScript(ctx, record.InstanceCID, scriptName, 120, info.RuntimeUser, nil, map[string]string{
			"download_url":  downloadURL,
			"skill_slug":    item.Slug,
			"skill_version": item.Version,
		})
		return err
	}
	return cfg, executor
}

func runSkillDistributeTask(
	ctx context.Context,
	item skillTaskItem,
	task model.SkillDistributionTask,
	records []model.SkillDistributionRecord,
	lock *model.DistLock,
	infoMap map[uint]skillInstanceInfo,
	deps skillExecutionDependencies,
) {
	// 本地 agent 实例的 record 不交给 executeSkillTaskAsync。
	// executor 返 nil 时会将 record 改为 status=success，但本地实例这时 reporter 还
	// 未下载并安装，纯属误报；本地 record 保留 pending，reporter /local-agent/sync
	// 拉取、ack 后转 success/failed。
	asyncRecords := filterOutLocalAgentRecords(records, infoMap)
	cfg, executor := buildSkillDistributeExecution(
		hcommon.DetachContext(ctx),
		item,
		task,
		asyncRecords,
		lock,
		infoMap,
		deps,
	)
	executeSkillTaskAsync(cfg, executor)
}

func buildSkillUninstallExecution(
	ctx context.Context,
	item skillTaskItem,
	task model.SkillDistributionTask,
	records []model.SkillDistributionRecord,
	lock *model.DistLock,
	infoMap map[uint]skillInstanceInfo,
	deps skillExecutionDependencies,
) (SkillTaskConfig, SkillTaskExecutor) {
	cfg := SkillTaskConfig{
		Ctx:     ctx,
		Task:    task,
		Records: records,
		Lock:    lock,
		Slug:    item.Slug,
		OnFailed: func(ctx context.Context, record model.SkillDistributionRecord) string {
			if item.Source == model.SkillSourcePublic {
				return model.ResolvePublicSkillUninstallFailedStatus(ctx, record.InstanceID, item.Slug, item.Version)
			}
			return model.ResolveUninstallFailedStatus(ctx, record.InstanceID, []uint{item.SkillID}, item.Version)
		},
	}
	executor := func(ctx context.Context, record model.SkillDistributionRecord) error {
		info := infoMap[record.InstanceID]
		scriptName, resolveErr := ResolveScript(ctx, "uninstall_skill", info.AgentType)
		if resolveErr != nil {
			return hcommon.I18nError(i18n.MsgUnsupportedAgentType, info.AgentType)
		}
		_, err := deps.runScript(ctx, record.InstanceCID, scriptName, 60, info.RuntimeUser, nil, map[string]string{
			"skill_slug": item.Slug,
		})
		return err
	}
	return cfg, executor
}

func runSkillUninstallTask(
	ctx context.Context,
	item skillTaskItem,
	task model.SkillDistributionTask,
	records []model.SkillDistributionRecord,
	lock *model.DistLock,
	infoMap map[uint]skillInstanceInfo,
	deps skillExecutionDependencies,
) {
	// 本地 agent 实例同样不交给 executor（参见 runSkillDistributeTask 注释）：
	// executor 返 nil 就会把 record 改为 success，但本地实例真正的卸载结果要等
	// reporter ack 后才能确认。
	asyncRecords := filterOutLocalAgentRecords(records, infoMap)
	cfg, executor := buildSkillUninstallExecution(
		hcommon.DetachContext(ctx),
		item,
		task,
		asyncRecords,
		lock,
		infoMap,
		deps,
	)
	executeSkillTaskAsync(cfg, executor)
}

// filterOutLocalAgentRecords 从 records 中剔除本地 agent 实例（source=local）的行，
// 未在 infoMap 中命中的 record 保留（一般不会发生，入一参一完全保守）。
// 本地实例的 record 需要保留 pending 状态 —— reporter /local-agent/sync 将拉取后 ack。
func filterOutLocalAgentRecords(records []model.SkillDistributionRecord, infoMap map[uint]skillInstanceInfo) []model.SkillDistributionRecord {
	out := make([]model.SkillDistributionRecord, 0, len(records))
	for _, rec := range records {
		info, ok := infoMap[rec.InstanceID]
		if ok && info.isLocalAgentInstance() {
			continue
		}
		out = append(out, rec)
	}
	return out
}

func handleDistributeSkillBatch(w http.ResponseWriter, r *http.Request, source, slug, version, sourceSkillsetSlug string, selection distributionSelection, skills []distributeSkillRequestItem) {
	if strings.TrimSpace(source) != "" || strings.TrimSpace(slug) != "" || strings.TrimSpace(version) != "" || strings.TrimSpace(sourceSkillsetSlug) != "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillTopLevelFieldsWithSkills))
		return
	}
	if len(skills) > skillTaskMaxItems {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillBatchItemsCountLimit, skillTaskMaxItems))
		return
	}

	// 取操作员并复用纯函数执行批量下发
	var operatorID uint
	if user, err := RequestUser(r); user != nil && err == nil {
		operatorID = user.ID
	}
	var batchID string
	var results []skillBatchResultItem
	var err error
	if selection.SelectAll {
		batchID, results, err = distributeSkillBatchSelectAll(r.Context(), operatorID, selection, skills)
	} else {
		batchID, results, err = distributeSkillBatch(r.Context(), operatorID, selection.InstanceIDs, skills)
	}
	if err != nil {
		var richErr *hcommon.RichError
		if errors.As(err, &richErr) {
			writeError(w, r, http.StatusBadRequest, richErr)
		} else {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
		}
		return
	}

	// 响应结果
	submitted := 0
	taskIDs := make([]uint, 0, len(results))
	for _, res := range results {
		if res.Status == "submitted" {
			submitted++
			if res.TaskID > 0 {
				taskIDs = append(taskIDs, res.TaskID)
			}
		}
	}
	jsonOK(w, map[string]interface{}{
		"ok":        true,
		"batch_id":  batchID,
		"task_ids":  taskIDs,
		"total":     len(results),
		"submitted": submitted,
		"failed":    len(results) - submitted,
		"results":   results,
	})
}

// distributeSkillBatch 批量下发技能的纯函数（无 HTTP 依赖）。
// 由 handleDistributeSkillBatch（HTTP 接口）与本模块资产版本自动同步（controller/asset_version.go）共用。
// 单个 skill 准备/加锁失败只记入 results，不影响其他 skill；返回 err 仅用于实例过滤等致命错误。
func distributeSkillBatch(ctx context.Context, operatorID uint, instanceIDs []uint, skills []distributeSkillRequestItem) (string, []skillBatchResultItem, error) {
	if len(instanceIDs) == 0 {
		return "", nil, hcommon.I18nError(i18n.MsgInstanceIdsCannotBeEmpty)
	}
	if len(skills) > skillTaskMaxItems {
		return "", nil, hcommon.I18nError(i18n.MsgSkillBatchItemsCountLimit, skillTaskMaxItems)
	}

	// 构造技能列表
	items := make([]skillTaskItem, 0, len(skills))
	for i, raw := range skills {
		items = append(items, createSkillTaskItem(i, raw.Source, raw.Slug, raw.Version, raw.SourceSkillsetSlug))
	}

	// 过滤目标实例
	instanceIDs = hcommon.Unique(instanceIDs)
	validIDs, infoMap, skippedCount, err := loadInstancesSupportingSkillTasks(ctx, instanceIDs)
	if err != nil {
		return "", nil, hcommon.I18nRichError(err, i18n.MsgSkillStoreQueryInstancesFail, err)
	}
	if skippedCount > 0 {
		slog.Info("技能下发跳过不支持技能的实例类型", "skipped", skippedCount)
	}
	if len(validIDs) == 0 {
		return "", nil, hcommon.I18nError(i18n.MsgSkillStoreNoValidInstall)
	}

	now := time.Now()
	batchID := newSkillTaskBatchID(now)
	results := make([]skillBatchResultItem, 0, len(items))

	// 创建任务并提交
	for _, item := range items {
		item, reason, prepErr := prepareDistributeSkillItem(ctx, item)
		if prepErr != nil {
			results = append(results, skillBatchResultItemFailed(item, reason, prepErr, ctx))
			continue
		}

		lock, lockErr := model.AcquireLock(hcommon.WithTaskTrace(hcommon.DetachContext(ctx), "skill_distribute"), skillTaskItemLockKey(item), 30*time.Minute)
		if lockErr != nil {
			results = append(results, skillBatchResultItemFailed(item, "locked", hcommon.I18nError(i18n.MsgSkillStoreVersionLocked), ctx))
			continue
		}

		// 新下发前：若上一次同 slug+source 的 distribute 仍有 pending 记录，先把上一次任务判失败。
		if err := failPreviousPendingSkillDistribute(ctx, item, validIDs); err != nil {
			lock.Release()
			results = append(results, skillBatchResultItemFailed(item, "fail_prev_failed", hcommon.I18nRichError(err, i18n.MsgSkillStoreCreateRecordFail), ctx))
			continue
		}

		task, records, err := createSkillTaskAndRecords(ctx, item, model.TaskTypeDistribute, operatorID, validIDs, infoMap, batchID, now)
		if err != nil {
			lock.Release()
			results = append(results, skillBatchResultItemFailed(item, "create_task_failed", hcommon.I18nRichError(err, i18n.MsgSkillStoreCreateRecordFail), ctx))
			continue
		}
		runSkillDistributeTask(ctx, item, task, records, lock, infoMap, defaultSkillExecutionDependencies())

		results = append(results, skillBatchResultItem{
			Index:         item.Index,
			Source:        item.Source,
			Slug:          item.Slug,
			Version:       item.Version,
			Status:        "submitted",
			TaskID:        task.ID,
			InstanceCount: len(validIDs),
		})
	}
	return batchID, results, nil
}

func distributeSkillBatchSelectAll(
	ctx context.Context,
	operatorID uint,
	selection distributionSelection,
	skills []distributeSkillRequestItem,
) (string, []skillBatchResultItem, error) {
	if len(skills) > skillTaskMaxItems {
		return "", nil, hcommon.I18nError(i18n.MsgSkillBatchItemsCountLimit, skillTaskMaxItems)
	}
	if _, err := normalizeSkillDistributionStatuses(selection.Statuses); err != nil {
		return "", nil, err
	}

	items := make([]skillTaskItem, 0, len(skills))
	for i, raw := range skills {
		items = append(items, createSkillTaskItem(i, raw.Source, raw.Slug, raw.Version, raw.SourceSkillsetSlug))
	}

	now := time.Now()
	batchID := newSkillTaskBatchID(now)
	results := make([]skillBatchResultItem, 0, len(items))
	for _, item := range items {
		item, reason, prepErr := prepareDistributeSkillItem(ctx, item)
		if prepErr != nil {
			results = append(results, skillBatchResultItemFailed(item, reason, prepErr, ctx))
			continue
		}

		lock, lockErr := model.AcquireLock(
			hcommon.WithTaskTrace(hcommon.DetachContext(ctx), "skill_distribute"),
			skillTaskItemLockKey(item),
			30*time.Minute,
		)
		if lockErr != nil {
			results = append(results, skillBatchResultItemFailed(item, "locked", hcommon.I18nError(i18n.MsgSkillStoreVersionLocked), ctx))
			continue
		}

		task, total, createErr := createSkillSelectAllTask(ctx, item, model.TaskTypeDistribute, operatorID, batchID, selection, now)
		if createErr != nil {
			lock.Release()
			var richErr *hcommon.RichError
			if !errors.As(createErr, &richErr) {
				richErr = hcommon.I18nRichError(createErr, i18n.MsgSkillStoreCreateRecordFail, createErr)
			}
			results = append(results, skillBatchResultItemFailed(item, "create_task_failed", richErr, ctx))
			continue
		}
		taskCtx := i18n.WithPrinter(hcommon.DetachContext(ctx), ctx)
		wg := skillDistributeWG
		if wg != nil {
			wg.Add(1)
		}
		go func() {
			if wg != nil {
				defer wg.Done()
			}
			defer lock.Release()
			defer recoverSkillSelectAllTaskPanic(taskCtx, task)
			runSkillSelectAllTask(taskCtx, item, task)
		}()
		results = append(results, skillBatchResultItem{
			Index:         item.Index,
			Source:        item.Source,
			Slug:          item.Slug,
			Version:       item.Version,
			Status:        "submitted",
			TaskID:        task.ID,
			InstanceCount: total,
		})
	}
	return batchID, results, nil
}

func handleUninstallSkillBatch(w http.ResponseWriter, r *http.Request, source, slug, sourceSkillsetSlug string, selection distributionSelection, skills []uninstallSkillRequestItem) {
	if err := selection.validate(); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if selection.SelectAll {
		if _, err := normalizeSkillUninstallStatuses(selection.Statuses); err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
			return
		}
	}
	if strings.TrimSpace(source) != "" || strings.TrimSpace(slug) != "" || strings.TrimSpace(sourceSkillsetSlug) != "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillTopLevelFieldsWithSkills))
		return
	}
	if len(skills) > skillTaskMaxItems {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillBatchItemsCountLimit, skillTaskMaxItems))
		return
	}

	// 构造技能列表
	items := make([]skillTaskItem, 0, len(skills))
	for i, raw := range skills {
		items = append(items, createSkillTaskItem(i, raw.Source, raw.Slug, "", raw.SourceSkillsetSlug))
	}

	var validIDs []uint
	var infoMap map[uint]skillInstanceInfo
	if !selection.SelectAll {
		instanceIDs := hcommon.Unique(selection.InstanceIDs)
		var skippedCount int
		var err error
		validIDs, infoMap, skippedCount, err = loadInstancesSupportingSkillTasks(r.Context(), instanceIDs)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgPluginQueryInstanceInfo))
			return
		}
		if skippedCount > 0 {
			slog.Info("技能卸载跳过不支持技能的实例类型", "skipped", skippedCount)
		}
		if len(validIDs) == 0 {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillStoreNoValidUninstall))
			return
		}
	}

	// 初始化批次
	var operatorID uint
	if user, err := RequestUser(r); user != nil && err == nil {
		operatorID = user.ID
	}
	now := time.Now()
	batchID := newSkillTaskBatchID(now)
	results := make([]skillBatchResultItem, 0, len(items))
	taskIDs := make([]uint, 0, len(items))

	// 创建任务并提交
	for _, item := range items {
		item, reason, prepErr := prepareUninstallSkillItem(r.Context(), item)
		if prepErr != nil {
			results = append(results, skillBatchResultItemFailed(item, reason, prepErr, r.Context()))
			continue
		}

		lock, lockErr := model.AcquireLock(hcommon.WithTaskTrace(hcommon.DetachContext(r.Context()), "skill_uninstall"), skillTaskItemLockKey(item), 30*time.Minute)
		if lockErr != nil {
			results = append(results, skillBatchResultItemFailed(item, "locked", hcommon.I18nError(i18n.MsgSkillStoreSkillLocked), r.Context()))
			continue
		}

		if selection.SelectAll {
			task, total, createErr := createSkillSelectAllTask(r.Context(), item, model.TaskTypeUninstall, operatorID, batchID, selection, now)
			if createErr != nil {
				lock.Release()
				var richErr *hcommon.RichError
				if !errors.As(createErr, &richErr) {
					richErr = hcommon.I18nRichError(createErr, i18n.MsgSkillStoreCreateUninstallTask)
				}
				results = append(results, skillBatchResultItemFailed(item, "create_task_failed", richErr, r.Context()))
				continue
			}
			taskCtx := i18n.WithPrinter(hcommon.DetachContext(r.Context()), r.Context())
			wg := skillDistributeWG
			if wg != nil {
				wg.Add(1)
			}
			go func() {
				if wg != nil {
					defer wg.Done()
				}
				defer lock.Release()
				defer recoverSkillSelectAllTaskPanic(taskCtx, task)
				runSkillSelectAllTask(taskCtx, item, task)
			}()
			taskIDs = append(taskIDs, task.ID)
			results = append(results, skillBatchResultItem{
				Index:         item.Index,
				Source:        item.Source,
				Slug:          item.Slug,
				Status:        "submitted",
				TaskID:        task.ID,
				InstanceCount: total,
			})
			continue
		}

		task, records, err := createSkillTaskAndRecords(r.Context(), item, model.TaskTypeUninstall, operatorID, validIDs, infoMap, batchID, now)
		if err != nil {
			lock.Release()
			results = append(results, skillBatchResultItemFailed(item, "create_task_failed", hcommon.I18nRichError(err, i18n.MsgSkillStoreCreateUninstallTask), r.Context()))
			continue
		}
		runSkillUninstallTask(r.Context(), item, task, records, lock, infoMap, defaultSkillExecutionDependencies())

		taskIDs = append(taskIDs, task.ID)
		results = append(results, skillBatchResultItem{
			Index:         item.Index,
			Source:        item.Source,
			Slug:          item.Slug,
			Status:        "submitted",
			TaskID:        task.ID,
			InstanceCount: len(validIDs),
		})
	}

	// 响应结果
	submitted := len(taskIDs)
	jsonOK(w, map[string]interface{}{
		"ok":        true,
		"batch_id":  batchID,
		"task_ids":  taskIDs,
		"total":     len(items),
		"submitted": submitted,
		"failed":    len(items) - submitted,
		"results":   results,
	})
}

const publicSkillInstallStatusCaseWithoutVersion = `CASE
		WHEN lr.status IS NULL THEN 'uninstalled'
		WHEN COALESCE(lr.type, 'distribute') = 'uninstall' AND lr.status = 'success' THEN 'uninstalled'
		WHEN COALESCE(lr.type, 'distribute') = 'uninstall' AND lr.status = 'pending' THEN 'uninstalling'
		WHEN COALESCE(lr.type, 'distribute') = 'uninstall' AND lr.status = 'failed'  THEN 'uninstall_failed'
		WHEN lr.status = 'success' THEN 'installed'
		WHEN lr.status = 'pending' THEN 'installing'
		WHEN lr.status = 'upgrade_failed' THEN 'upgrade_failed'
		WHEN lr.status = 'uninstall_failed_old' THEN 'uninstall_failed_old'
		WHEN lr.status = 'failed'  THEN 'failed'
		ELSE 'uninstalled'
	END`

func filterPublicSkillInstallStatusesWithoutVersion(statuses []string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if len(statuses) == 0 {
			return db
		}
		return db.Where(`(CASE
			WHEN lr.status IS NULL THEN 'uninstalled'
			WHEN COALESCE(lr.type, 'distribute') = 'uninstall' AND lr.status = 'success' THEN 'uninstalled'
			WHEN COALESCE(lr.type, 'distribute') = 'uninstall' AND lr.status = 'pending' THEN 'uninstalling'
			WHEN COALESCE(lr.type, 'distribute') = 'uninstall' AND lr.status = 'failed' THEN 'uninstall_failed'
			WHEN lr.status = 'success' THEN 'installed'
			WHEN lr.status = 'pending' THEN 'installing'
			WHEN lr.status = 'upgrade_failed' THEN 'upgrade_failed'
			WHEN lr.status = 'uninstall_failed_old' THEN 'uninstall_failed_old'
			WHEN lr.status = 'failed' THEN 'failed'
			ELSE 'uninstalled'
		END) IN ?`, statuses)
	}
}

const publicSkillInstanceSelectFormat = `instances.id AS instance_id,
		instances.instance_id AS cvm_instance_id,
		instances.name AS instance_name,
		COALESCE(instances.agent_type, '') AS instance_type,
		instances.user_id AS user_id,
		instances.last_cvm_state AS last_cvm_state,
		instances.last_stable_state AS last_stable_state,
		instances.current_operation AS current_operation,
		instances.current_operation_state AS current_operation_state,
		instances.agent_ready AS agent_ready,
		instances.cls_agent_status AS cls_agent_status,
		instances.cls_agent_status_at AS cls_agent_status_at,
		COALESCE(u.username, '') AS username,
		%s AS install_status,
		COALESCE(lr.version, '') AS version,
		'%s' AS latest_version`

func buildPublicSkillInstanceQuery(ctx context.Context, slug, normalizedVersion string) *gorm.DB {
	// normalizedVersion 会以 SQL 字面值形式嵌入 SELECT，用于复用原企业技能查询的 CASE 逻辑。
	// 进入 fmt.Sprintf 前已在请求入口规范化；缺省或非法版本由无版本查询分支处理。
	selectClause := fmt.Sprintf(
		publicSkillInstanceSelectFormat,
		model.InstallStatusCase(normalizedVersion),
		normalizedVersion,
	)
	return newPublicSkillInstanceQuery(ctx, slug).Select(selectClause)
}

func buildPublicSkillInstanceQueryWithoutVersion(ctx context.Context, slug string) *gorm.DB {
	selectClause := fmt.Sprintf(
		publicSkillInstanceSelectFormat,
		publicSkillInstallStatusCaseWithoutVersion,
		"",
	)
	return newPublicSkillInstanceQuery(ctx, slug).Select(selectClause)
}

func newPublicSkillInstanceQuery(ctx context.Context, slug string) *gorm.DB {
	q := model.DB(ctx).Model(&model.Instance{})
	q = q.Joins("LEFT JOIN users u ON u.id = instances.user_id AND u.deleted_at IS NULL")
	q = q.Joins(`LEFT JOIN skill_distribution_records lr
		ON lr.instance_id = instances.id
		AND lr.deleted_at IS NULL
		AND lr.id = (
			SELECT MAX(r2.id) FROM skill_distribution_records r2
			JOIN skill_distribution_tasks t2 ON t2.id = r2.task_id AND t2.deleted_at IS NULL
			WHERE t2.source = ? AND t2.slug = ? AND r2.instance_id = instances.id AND r2.deleted_at IS NULL
		)`, model.SkillSourcePublic, slug)
	// local_instance_skills JOIN — 与企业技能分支保持对齐（参见 model.BuildSkillInstanceQuery）：
	// InstallStatusCase() 在 latestVersion 非空时会引用 lis.id（本地实例上报已卸载
	// 时将状态改判 uninstalled），公共技能分支同样需要这个 JOIN，否则 SQL 会
	// 报 "no such column: lis.id"。CVM 实例（source != 'local'）不会命中 lis 分支，
	// 行为保持不变。
	q = q.Joins("LEFT JOIN local_instance_skills lis ON lis.instance_id = instances.id AND lis.slug = ?", slug)
	q = q.Where("instances.instance_id != ''")
	return q
}

func skillRecordFailed(status string) bool {
	switch status {
	case "failed", "upgrade_failed", "uninstall_failed", "uninstall_failed_old":
		return true
	default:
		return false
	}
}

func skillRecordPending(status string) bool {
	switch status {
	case "pending", "installing", "uninstalling":
		return true
	default:
		return false
	}
}

// aggregateRecordStatuses 汇总同一实例在一个批量任务里的多条 record 状态。
// 任一技能失败则该实例的聚合状态为 failed；只要还有 pending 就保持 pending；全部成功才返回 success。
func aggregateRecordStatuses(statuses []string) string {
	if len(statuses) == 0 {
		return "pending"
	}
	hasPending := false
	for _, status := range statuses {
		if skillRecordFailed(status) {
			return "failed"
		}
		if skillRecordPending(status) {
			hasPending = true
		}
	}
	if hasPending {
		return "pending"
	}
	return "success"
}

func skillRecordInstallStatus(recordType, recordStatus, recordVersion, latestVersion string) string {
	if recordType == model.TaskTypeUninstall {
		switch recordStatus {
		case "success":
			return "uninstalled"
		case "pending":
			return "uninstalling"
		case "failed":
			return "uninstall_failed"
		default:
			return recordStatus
		}
	}
	switch recordStatus {
	case "success":
		if latestVersion != "" && model.VersionScore(latestVersion) != 0 && recordVersion != latestVersion {
			return "outdated"
		}
		return "installed"
	case "pending":
		return "installing"
	case "failed":
		return "failed"
	case "upgrade_failed", "uninstall_failed_old":
		return recordStatus
	default:
		return "uninstalled"
	}
}

func handlePublicSkillsetTaskBatches(w http.ResponseWriter, r *http.Request, sourceSkillsetSlug, typeFilter string, page, pageSize int) {
	taskQuery := model.DB(r.Context()).Model(&model.SkillDistributionTask{}).
		Where("source = ? AND source_skillset_slug = ?", model.SkillSourcePublic, sourceSkillsetSlug)
	if typeFilter != "" && typeFilter != "all" {
		taskQuery = taskQuery.Where("type = ?", typeFilter)
	}

	var tasks []model.SkillDistributionTask
	if err := taskQuery.Order("id desc").Find(&tasks).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQuerySkillInstallationFailed))
		return
	}

	type taskGroup struct {
		Key     string
		Tasks   []model.SkillDistributionTask
		TaskIDs []uint
	}
	groupByKey := make(map[string]*taskGroup)
	groups := make([]*taskGroup, 0)
	for _, task := range tasks {
		key := task.BatchID
		if key == "" {
			key = fmt.Sprintf("task:%d", task.ID)
		}
		group := groupByKey[key]
		if group == nil {
			group = &taskGroup{Key: key}
			groupByKey[key] = group
			groups = append(groups, group)
		}
		group.Tasks = append(group.Tasks, task)
		group.TaskIDs = append(group.TaskIDs, task.ID)
	}

	total := len(groups)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	pageGroups := groups[start:end]

	taskIDSet := make(map[uint]struct{})
	operatorIDs := make(map[uint]struct{})
	taskByID := make(map[uint]model.SkillDistributionTask)
	for _, group := range pageGroups {
		for _, task := range group.Tasks {
			taskIDSet[task.ID] = struct{}{}
			taskByID[task.ID] = task
			if task.OperatorID > 0 {
				operatorIDs[task.OperatorID] = struct{}{}
			}
		}
	}
	taskIDs := make([]uint, 0, len(taskIDSet))
	for id := range taskIDSet {
		taskIDs = append(taskIDs, id)
	}

	var allRecords []model.SkillDistributionRecord
	if len(taskIDs) > 0 {
		if err := model.DB(r.Context()).Where("task_id IN ?", taskIDs).Find(&allRecords).Error; err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQuerySkillInstallationFailed))
			return
		}
	}
	recordsByTask := make(map[uint][]model.SkillDistributionRecord)
	instIDSet := make(map[uint]struct{})
	for _, rec := range allRecords {
		recordsByTask[rec.TaskID] = append(recordsByTask[rec.TaskID], rec)
		instIDSet[rec.InstanceID] = struct{}{}
	}

	opIDs := make([]uint, 0, len(operatorIDs))
	for id := range operatorIDs {
		opIDs = append(opIDs, id)
	}
	userMap := make(map[uint]string)
	if len(opIDs) > 0 {
		var users []model.User
		model.DB(r.Context()).Where("id IN ?", opIDs).Find(&users)
		for _, u := range users {
			userMap[u.ID] = u.Username
		}
	}

	instIDs := make([]uint, 0, len(instIDSet))
	for id := range instIDSet {
		instIDs = append(instIDs, id)
	}
	type instDetail struct {
		ID     uint
		Name   string
		UserID uint
	}
	instMap := make(map[uint]instDetail)
	instUserIDs := make(map[uint]struct{})
	if len(instIDs) > 0 {
		var insts []instDetail
		model.DB(r.Context()).Model(&model.Instance{}).Select("id, name, user_id").Where("id IN ?", instIDs).Scan(&insts)
		for _, inst := range insts {
			instMap[inst.ID] = inst
			if inst.UserID > 0 {
				instUserIDs[inst.UserID] = struct{}{}
			}
		}
	}
	instUserIDList := make([]uint, 0, len(instUserIDs))
	for id := range instUserIDs {
		instUserIDList = append(instUserIDList, id)
	}
	instUserMap := make(map[uint]string)
	if len(instUserIDList) > 0 {
		var users []model.User
		model.DB(r.Context()).Where("id IN ?", instUserIDList).Find(&users)
		for _, u := range users {
			instUserMap[u.ID] = u.Username
		}
	}

	type skillStatusResp struct {
		TaskID  uint   `json:"task_id"`
		Slug    string `json:"slug"`
		Version string `json:"version"`
		Type    string `json:"type"`
		Status  string `json:"status"`
		Error   string `json:"error"`
	}
	type batchRecordResp struct {
		InstanceID    uint              `json:"instance_id"`
		CVMInstanceID string            `json:"cvm_instance_id"`
		InstanceName  string            `json:"instance_name"`
		Username      string            `json:"username"`
		Status        string            `json:"status"`
		Error         string            `json:"error"`
		SkillStatuses []skillStatusResp `json:"skill_statuses"`
	}
	type batchResp struct {
		BatchID            string            `json:"batch_id"`
		TaskIDs            []uint            `json:"task_ids"`
		CreatedAt          interface{}       `json:"created_at"`
		Operator           string            `json:"operator"`
		Source             string            `json:"source"`
		SourceSkillsetSlug string            `json:"source_skillset_slug"`
		Total              int               `json:"total"`
		Success            int               `json:"success"`
		Failed             int               `json:"failed"`
		Pending            int               `json:"pending"`
		Status             string            `json:"status"`
		Type               string            `json:"type"`
		Records            []batchRecordResp `json:"records"`
	}

	result := make([]batchResp, 0, len(pageGroups))
	for _, group := range pageGroups {
		first := group.Tasks[0]
		resp := batchResp{
			BatchID:            first.BatchID,
			TaskIDs:            group.TaskIDs,
			CreatedAt:          first.CreatedAt,
			Operator:           userMap[first.OperatorID],
			Source:             model.SkillSourcePublic,
			SourceSkillsetSlug: sourceSkillsetSlug,
			Status:             "completed",
			Type:               first.Type,
			Records:            []batchRecordResp{},
		}
		for _, task := range group.Tasks[1:] {
			if task.Type != resp.Type {
				resp.Type = "all"
				break
			}
		}
		recordIndexByInstance := make(map[uint]int)
		recordStatusByInstance := make(map[uint][]string)
		for _, task := range group.Tasks {
			if task.Status == "running" {
				resp.Status = "running"
			}
			for _, rec := range recordsByTask[task.ID] {
				resp.Total++
				switch {
				case rec.Status == "success":
					resp.Success++
				case rec.Status == "pending":
					resp.Pending++
					resp.Status = "running"
				default:
					resp.Failed++
				}
				idx, ok := recordIndexByInstance[rec.InstanceID]
				if !ok {
					resp.Records = append(resp.Records, batchRecordResp{InstanceID: rec.InstanceID, CVMInstanceID: rec.InstanceCID})
					idx = len(resp.Records) - 1
					recordIndexByInstance[rec.InstanceID] = idx
					if inst, ok := instMap[rec.InstanceID]; ok {
						resp.Records[idx].InstanceName = inst.Name
						resp.Records[idx].Username = instUserMap[inst.UserID]
					}
				}
				rr := &resp.Records[idx]
				taskInfo := taskByID[rec.TaskID]
				status := skillRecordInstallStatus(rec.Type, rec.Status, rec.Version, taskInfo.Version)
				if rec.Status == "pending" {
					status = "pending"
				}
				rr.SkillStatuses = append(rr.SkillStatuses, skillStatusResp{
					TaskID:  rec.TaskID,
					Slug:    taskInfo.Slug,
					Version: rec.Version,
					Type:    rec.Type,
					Status:  status,
					Error:   rec.Error,
				})
				if rec.Error != "" && rr.Error == "" {
					rr.Error = rec.Error
				}
				recordStatusByInstance[rec.InstanceID] = append(recordStatusByInstance[rec.InstanceID], rec.Status)
			}
		}
		for i := range resp.Records {
			instID := resp.Records[i].InstanceID
			resp.Records[i].Status = aggregateRecordStatuses(recordStatusByInstance[instID])
			if resp.Records[i].SkillStatuses == nil {
				resp.Records[i].SkillStatuses = []skillStatusResp{}
			}
		}
		result = append(result, resp)
	}

	jsonOK(w, map[string]interface{}{
		"tasks":     result,
		"page":      page,
		"page_size": pageSize,
		"total":     total,
	})
}

type publicSkillsetInstanceSkillRequest struct {
	Source             string `json:"source"`
	Slug               string `json:"slug"`
	Version            string `json:"version"`
	SourceSkillsetSlug string `json:"source_skillset_slug"`
}

type publicSkillsetInstancesRequest struct {
	Source             string                               `json:"source"`
	SourceSkillsetSlug string                               `json:"source_skillset_slug"`
	Skills             []publicSkillsetInstanceSkillRequest `json:"skills"`
	Status             string                               `json:"status"`
	Search             string                               `json:"search"`
	InstanceType       string                               `json:"instance_type"`
	GroupID            string                               `json:"group_id"`
}

// aggregateSkillsetInstallStatus 汇总一个公共技能包在单个实例上的整体安装状态。
// 失败优先级最高，其次是进行中、待更新、未完整安装；只有所有技能都已安装且非旧版时才返回 installed。
func aggregateSkillsetInstallStatus(statuses []string) string {
	if len(statuses) == 0 {
		return "uninstalled"
	}
	hasInstalling := false
	hasOutdated := false
	hasUninstalled := false
	for _, status := range statuses {
		switch status {
		case "failed", "upgrade_failed", "uninstall_failed", "uninstall_failed_old":
			return "failed"
		case "installing", "uninstalling", "pending":
			hasInstalling = true
		case "outdated":
			hasOutdated = true
		case "uninstalled":
			hasUninstalled = true
		}
	}
	if hasInstalling {
		return "installing"
	}
	if hasOutdated {
		return "outdated"
	}
	if hasUninstalled {
		return "uninstalled"
	}
	return "installed"
}

// statusFilterAllows 判断聚合后的状态是否命中用户传入的 status 过滤条件。
// 这里复用 GET 接口的逗号分隔约定，例如 status=installed,failed。
func statusFilterAllows(filter, status string) bool {
	if filter == "" {
		return true
	}
	for _, item := range strings.Split(filter, ",") {
		if strings.TrimSpace(item) == status {
			return true
		}
	}
	return false
}

func handlePublicSkillsetInstances(w http.ResponseWriter, r *http.Request) {
	// 解析请求
	var req publicSkillsetInstancesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgPluginRequestFormatErr))
		return
	}

	// 校验技能包参数
	req.Source = strings.TrimSpace(req.Source)
	if req.Source == "" {
		req.Source = model.SkillSourcePublic
	}
	req.SourceSkillsetSlug = strings.TrimSpace(req.SourceSkillsetSlug)
	if req.Source != model.SkillSourcePublic {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgUnsupportedSkillSource, req.Source))
		return
	}
	if req.SourceSkillsetSlug == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "source_skillset_slug"))
		return
	}
	if len(req.Skills) == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "skills"))
		return
	}
	if len(req.Skills) > skillTaskMaxItems {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillBatchItemsCountLimit, skillTaskMaxItems))
		return
	}

	// 校验技能列表
	items := make([]skillTaskItem, 0, len(req.Skills))
	slugSet := make(map[string]struct{}, len(req.Skills))
	slugs := make([]string, 0, len(req.Skills))
	for i, raw := range req.Skills {
		source := strings.TrimSpace(raw.Source)
		if source == "" {
			source = model.SkillSourcePublic
		}
		sourceSkillsetSlug := strings.TrimSpace(raw.SourceSkillsetSlug)
		if sourceSkillsetSlug == "" {
			sourceSkillsetSlug = req.SourceSkillsetSlug
		}
		item := createSkillTaskItem(i, source, raw.Slug, raw.Version, sourceSkillsetSlug)
		if reason, richErr := validateSkillItem(item); richErr != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalidWithDetail, reason, hcommon.ErrorMessageWithCtx(r.Context(), richErr)))
			return
		}
		if item.Source != model.SkillSourcePublic {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgUnsupportedSkillSource, item.Source))
			return
		}
		if item.SourceSkillsetSlug != req.SourceSkillsetSlug {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "source_skillset_slug"))
			return
		}
		items = append(items, item)
		if _, ok := slugSet[item.Slug]; !ok {
			slugSet[item.Slug] = struct{}{}
			slugs = append(slugs, item.Slug)
		}
	}

	// 读取筛选条件
	statusFilter := req.Status
	if statusFilter == "" {
		statusFilter = r.URL.Query().Get("status")
	}
	search := req.Search
	if search == "" {
		search = r.URL.Query().Get("search")
	}
	instanceType := req.InstanceType
	if instanceType == "" {
		instanceType = r.URL.Query().Get("instance_type")
	}
	groupIDStr := req.GroupID
	if groupIDStr == "" {
		groupIDStr = r.URL.Query().Get("group_id")
	}

	// 查询候选实例
	type instBaseResp struct {
		InstanceID            uint       `json:"instance_id"              gorm:"column:instance_id"`
		CVMInstanceID         string     `json:"cvm_instance_id"          gorm:"column:cvm_instance_id"`
		InstanceName          string     `json:"instance_name"            gorm:"column:instance_name"`
		InstanceType          string     `json:"instance_type"            gorm:"column:instance_type"`
		Source                string     `json:"-"                        gorm:"column:source"`
		UserID                uint       `json:"user_id"                  gorm:"column:user_id"`
		Username              string     `json:"username"                 gorm:"column:username"`
		LastCVMState          string     `json:"last_cvm_state"           gorm:"column:last_cvm_state"`
		LastStableState       string     `json:"-"                        gorm:"column:last_stable_state"`
		CurrentOperation      string     `json:"-"                        gorm:"column:current_operation"`
		CurrentOperationState string     `json:"-"                        gorm:"column:current_operation_state"`
		AgentReady            int        `json:"-"                        gorm:"column:agent_ready"`
		CLSAgentStatus        int        `json:"-"                        gorm:"column:cls_agent_status"`
		CLSAgentStatusAt      *time.Time `json:"-"                        gorm:"column:cls_agent_status_at"`
	}
	selectClause := `instances.id AS instance_id,
		instances.instance_id AS cvm_instance_id,
		instances.name AS instance_name,
		COALESCE(instances.agent_type, '') AS instance_type,
		COALESCE(instances.source, '') AS source,
		instances.user_id AS user_id,
		instances.last_cvm_state AS last_cvm_state,
		instances.last_stable_state AS last_stable_state,
		instances.current_operation AS current_operation,
		instances.current_operation_state AS current_operation_state,
		instances.agent_ready AS agent_ready,
		instances.cls_agent_status AS cls_agent_status,
		instances.cls_agent_status_at AS cls_agent_status_at,
		COALESCE(u.username, '') AS username`
	baseQuery := model.DB(r.Context()).Model(&model.Instance{}).Select(selectClause).
		Joins("LEFT JOIN users u ON u.id = instances.user_id AND u.deleted_at IS NULL").
		Where("instances.instance_id != ''")

	if groupIDStr != "" {
		var groupIDs []int
		includeUngrouped := false
		for _, s := range strings.Split(groupIDStr, ",") {
			id, err := strconv.Atoi(strings.TrimSpace(s))
			if err != nil {
				continue
			}
			if id == 0 {
				includeUngrouped = true
			} else if id > 0 {
				groupIDs = append(groupIDs, id)
			}
		}
		if includeUngrouped && len(groupIDs) > 0 {
			ungroupedSubQ := model.DB(r.Context()).Model(&model.UserGroupMember{}).Select("DISTINCT user_id")
			groupedSubQ := model.DB(r.Context()).Model(&model.UserGroupMember{}).Select("DISTINCT user_id").Where("user_group_id IN ?", groupIDs)
			baseQuery = baseQuery.Where("instances.user_id NOT IN (?) OR instances.user_id IN (?)", ungroupedSubQ, groupedSubQ)
		} else if includeUngrouped {
			ungroupedSubQ := model.DB(r.Context()).Model(&model.UserGroupMember{}).Select("DISTINCT user_id")
			baseQuery = baseQuery.Where("instances.user_id NOT IN (?)", ungroupedSubQ)
		} else if len(groupIDs) > 0 {
			// 仅指定分组（使用子查询避免 JOIN 产生重复行）
			groupedSubQ := model.DB(r.Context()).Model(&model.UserGroupMember{}).Select("DISTINCT user_id").Where("user_group_id IN ?", groupIDs)
			baseQuery = baseQuery.Where("instances.user_id IN (?)", groupedSubQ)
		}
	}
	if search != "" {
		baseQuery = baseQuery.Where("instances.name LIKE ? OR instances.instance_id LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if instanceType != "" {
		types := strings.Split(instanceType, ",")
		trimmed := make([]string, 0, len(types))
		for _, t := range types {
			if s := strings.TrimSpace(t); s != "" {
				trimmed = append(trimmed, s)
			}
		}
		if len(trimmed) > 0 {
			baseQuery = baseQuery.Where("instances.agent_type IN ?", trimmed)
		}
	}

	var allInstances []instBaseResp
	if err := baseQuery.Order("instances.created_at DESC").Scan(&allInstances).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSkillStoreQueryInstancesFail))
		return
	}

	// 过滤可安装的运行中实例
	var cvmIDs []string
	for _, inst := range allInstances {
		if inst.CVMInstanceID != "" {
			cvmIDs = append(cvmIDs, inst.CVMInstanceID)
		}
	}
	cvmInfoMap := batchFetchCVMInfoMap(r.Context(), cvmIDs)

	// 批量预查：消除循环内 N+1 DB 查询
	siteConfig := model.GetSiteConfig(r.Context())
	allAgentTypes := model.GetAllAgentTypesMap(r.Context())

	preInstIDs := make([]uint, 0, len(allInstances))
	localInstIDs := make([]uint, 0)
	for _, item := range allInstances {
		preInstIDs = append(preInstIDs, item.InstanceID)
		if item.Source == model.InstanceSourceLocal {
			localInstIDs = append(localInstIDs, item.InstanceID)
		}
	}
	installingSkillMap := batchHasInstallingSkillInstallations(r.Context(), preInstIDs)
	localInfoMap := batchResolveLocalInstanceStatus(r.Context(), localInstIDs)
	batch := &InstanceStatusBatchLookup{SiteConfig: siteConfig, InstallingSkillMap: installingSkillMap, LocalInfoMap: localInfoMap}

	type runningInst struct {
		instBaseResp
		InstanceStatus      string
		InstanceStatusLabel string
		Transient           bool
	}
	runningInstances := make([]runningInst, 0, len(allInstances))
	for _, item := range allInstances {
		// 本地实例放行（与 GET 路径保持一致）
		if item.Source != model.InstanceSourceLocal &&
			!model.AgentTypeSupportsSkillByMap(item.InstanceType, allAgentTypes) {
			continue
		}
		tmpInst := model.Instance{
			LastCVMState:          item.LastCVMState,
			LastStableState:       item.LastStableState,
			CurrentOperation:      item.CurrentOperation,
			CurrentOperationState: item.CurrentOperationState,
			AgentReady:            item.AgentReady,
			CLSAgentStatus:        item.CLSAgentStatus,
			CLSAgentStatusAt:      item.CLSAgentStatusAt,
			InstanceId:            item.CVMInstanceID,
			Source:                item.Source,
		}
		tmpInst.ID = item.InstanceID
		statusResp := ResolveInstanceStatus(r.Context(), &tmpInst, cvmInfoMap[item.CVMInstanceID], batch)
		if statusResp.Status != model.StatusRunning {
			continue
		}
		runningInstances = append(runningInstances, runningInst{
			instBaseResp:        item,
			InstanceStatus:      statusResp.Status,
			InstanceStatusLabel: statusResp.Label,
			Transient:           statusResp.Transient,
		})
	}

	// 查询技能安装记录
	instIDs := make([]uint, 0, len(runningInstances))
	for _, inst := range runningInstances {
		instIDs = append(instIDs, inst.InstanceID)
	}
	type publicRecordRow struct {
		RecordID      uint   `gorm:"column:record_id"`
		TaskID        uint   `gorm:"column:task_id"`
		InstanceID    uint   `gorm:"column:instance_id"`
		Slug          string `gorm:"column:slug"`
		Version       string `gorm:"column:version"`
		TargetVersion string `gorm:"column:target_version"`
		Type          string `gorm:"column:type"`
		Status        string `gorm:"column:status"`
		Error         string `gorm:"column:error"`
	}
	var recordRows []publicRecordRow
	if len(instIDs) > 0 {
		if err := model.DB(r.Context()).Model(&model.SkillDistributionRecord{}).
			Select(`skill_distribution_records.id AS record_id,
				skill_distribution_records.task_id AS task_id,
				skill_distribution_records.instance_id AS instance_id,
				skill_distribution_records.version AS version,
				skill_distribution_records.type AS type,
				skill_distribution_records.status AS status,
				skill_distribution_records.error AS error,
				t.slug AS slug,
				t.version AS target_version`).
			Joins("JOIN skill_distribution_tasks t ON t.id = skill_distribution_records.task_id AND t.deleted_at IS NULL").
			Where("t.source = ? AND t.source_skillset_slug = ? AND t.slug IN ? AND skill_distribution_records.instance_id IN ?", model.SkillSourcePublic, req.SourceSkillsetSlug, slugs, instIDs).
			Order("skill_distribution_records.id ASC").
			Scan(&recordRows).Error; err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQuerySkillInstallationFailed))
			return
		}
	}

	// 按实例聚合技能状态
	latestByInstanceSlug := make(map[uint]map[string]publicRecordRow)
	for _, row := range recordRows {
		if latestByInstanceSlug[row.InstanceID] == nil {
			latestByInstanceSlug[row.InstanceID] = make(map[string]publicRecordRow)
		}
		latestByInstanceSlug[row.InstanceID][row.Slug] = row
	}

	type skillStatusResp struct {
		Slug    string `json:"slug"`
		Version string `json:"version"`
		Status  string `json:"status"`
		TaskID  uint   `json:"task_id,omitempty"`
		Error   string `json:"error,omitempty"`
	}
	type groupInfo struct {
		GroupID   uint   `json:"group_id"`
		GroupName string `json:"group_name"`
	}
	type instFinalResp struct {
		instBaseResp
		Status              string            `json:"status"`
		Version             string            `json:"version"`
		LatestVersion       string            `json:"latest_version"`
		SourceSkillsetSlug  string            `json:"source_skillset_slug"`
		SkillStatuses       []skillStatusResp `json:"skill_statuses"`
		UserGroups          []groupInfo       `json:"user_groups"`
		InstanceStatus      string            `json:"instance_status"`
		InstanceStatusLabel string            `json:"instance_status_label"`
		Transient           bool              `json:"transient"`
	}
	filtered := make([]instFinalResp, 0, len(runningInstances))
	for _, inst := range runningInstances {
		perSkill := make([]skillStatusResp, 0, len(items))
		statuses := make([]string, 0, len(items))
		for _, item := range items {
			status := "uninstalled"
			version := item.Version
			var taskID uint
			var errText string
			if row, ok := latestByInstanceSlug[inst.InstanceID][item.Slug]; ok {
				latestVersion := item.Version
				if latestVersion == "latest" {
					latestVersion = ""
				}
				if latestVersion == "" {
					latestVersion = row.TargetVersion
				}
				status = skillRecordInstallStatus(row.Type, row.Status, row.Version, latestVersion)
				version = row.Version
				taskID = row.TaskID
				errText = row.Error
			}
			statuses = append(statuses, status)
			perSkill = append(perSkill, skillStatusResp{Slug: item.Slug, Version: version, Status: status, TaskID: taskID, Error: errText})
		}
		aggregateStatus := aggregateSkillsetInstallStatus(statuses)
		if !statusFilterAllows(statusFilter, aggregateStatus) {
			continue
		}
		filtered = append(filtered, instFinalResp{
			instBaseResp:        inst.instBaseResp,
			Status:              aggregateStatus,
			SourceSkillsetSlug:  req.SourceSkillsetSlug,
			SkillStatuses:       perSkill,
			UserGroups:          []groupInfo{},
			InstanceStatus:      inst.InstanceStatus,
			InstanceStatusLabel: inst.InstanceStatusLabel,
			Transient:           inst.Transient,
		})
	}

	// 分页
	page, pageSize := parsePagination(r, 500)
	total := int64(len(filtered))
	start := (page - 1) * pageSize
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	pageResults := filtered[start:end]

	// 补充用户组信息
	userIDSet := make(map[uint]bool)
	for _, item := range pageResults {
		if item.UserID > 0 {
			userIDSet[item.UserID] = true
		}
	}
	userGroupMap := make(map[uint][]model.UserGroup)
	if len(userIDSet) > 0 {
		userIDs := make([]uint, 0, len(userIDSet))
		for uid := range userIDSet {
			userIDs = append(userIDs, uid)
		}
		if m, err := model.GetUserGroupsByUserIDs(r.Context(), userIDs); err == nil {
			userGroupMap = m
		} else {
			slog.Error("[PublicSkillsetInstances] 批量查询用户分组失败", "error", err)
		}
	}
	for i := range pageResults {
		if groups, ok := userGroupMap[pageResults[i].UserID]; ok {
			for _, g := range groups {
				pageResults[i].UserGroups = append(pageResults[i].UserGroups, groupInfo{GroupID: g.ID, GroupName: g.Name})
			}
		}
		if pageResults[i].SkillStatuses == nil {
			pageResults[i].SkillStatuses = []skillStatusResp{}
		}
		if pageResults[i].UserGroups == nil {
			pageResults[i].UserGroups = []groupInfo{}
		}
	}

	// 响应结果
	jsonOK(w, map[string]interface{}{
		"instances": pageResults,
		"page":      page,
		"page_size": pageSize,
		"total":     total,
	})
}
