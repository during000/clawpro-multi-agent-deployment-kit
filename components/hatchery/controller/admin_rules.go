package controller

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm"
)

// -----------------------------------------------------------------------------
// 企业规范库 admin CRUD（本地 agent 二期）
//
// 覆盖 5 条 REST 接口（对齐 skill 侧命名 + 精简）：
//   GET  /admin/rules
//   GET  /admin/rules/detail
//   POST /admin/rules/create
//   POST /admin/rules/delete
//   GET  /admin/rules/files
//
// 与 skill CRUD 的差异：
//   - 无 category / visibility_type / security_scan / changelog（一期决定）
//   - 单 md 文件上传（≤ 1 MiB），不打 zip、不解压
//   - 新增 type（prompt / rule）+ 首版决定后不可变约束
//
// 未覆盖接口（PR 3+）：
//   POST /admin/rules/distribute
//   POST /admin/rules/uninstall
//   GET  /admin/rules/tasks
//   GET  /admin/rules/instances
//   GET  /admin/instances/rules
// -----------------------------------------------------------------------------

// maxRuleUploadSize 规范文件大小上限：1 MiB。
const maxRuleUploadSize = 1 << 20

// enterpriseRuleCOSPrefix SMH 上规范文件的目录前缀。
const enterpriseRuleCOSPrefix = "enterprise-rules"

// errRuleHasRunningTask 用于 delete 前置检查（保持事务外错误分支简洁）。
var errRuleHasRunningTask = hcommon.I18nError(i18n.MsgRuleHasRunningTask)

// buildRuleCOSKey 生成规范在 SMH 上的 object key：`enterprise-rules/{slug}/{version}.md`。
// 与 skill 的 `{slug}/{slug}-{version}.zip` 结构不同——单文件、无目录展开。
func buildRuleCOSKey(slug, version string) string {
	return fmt.Sprintf("%s/%s/%s.md", enterpriseRuleCOSPrefix, slug, version)
}

// validateRuleType 校验 type 参数合法且必填。
func validateRuleType(t string) error {
	switch t {
	case "":
		return hcommon.I18nError(i18n.MsgRuleTypeRequired)
	case model.EnterpriseRuleTypePrompt, model.EnterpriseRuleTypeRule, model.EnterpriseRuleTypeHook:
		return nil
	default:
		return hcommon.I18nError(i18n.MsgRuleTypeInvalid)
	}
}

// validateRuleFileContent 校验 md 内容：非空、UTF-8、无 \x00。
func validateRuleFileContent(data []byte) error {
	if len(data) == 0 {
		return hcommon.I18nError(i18n.MsgRuleFileContentEmpty)
	}
	if !utf8.Valid(data) {
		return hcommon.I18nError(i18n.MsgRuleFileContentInvalid)
	}
	for _, b := range data {
		if b == 0x00 {
			return hcommon.I18nError(i18n.MsgRuleFileContentInvalid)
		}
	}
	return nil
}

// -----------------------------------------------------------------------------
// helpers：rule visibility 镶位包装（镜像 skill 侧 buildSkillVisibilityData）
// -----------------------------------------------------------------------------

// ruleVisibilityGroupInfo 用于规范列表响应中的可见性分组信息。
type ruleVisibilityGroupInfo struct {
	GroupID   uint   `json:"group_id"`
	GroupName string `json:"group_name"`
}

// buildRuleVisibilityData 批量构建规范列表的可见性分组数据。
// 返回 map[ruleID][]ruleVisibilityGroupInfo（含 group_id + group_name）。
// 固定 2 次额外 DB 查询（查关联 + 查分组名称），无 N+1 问题。
func buildRuleVisibilityData(ctx context.Context, rules []model.EnterpriseRule) map[uint][]ruleVisibilityGroupInfo {
	result := make(map[uint][]ruleVisibilityGroupInfo)

	// 筛出 visibility_type="group" 的 rule ID
	var groupRuleIDs []uint
	for _, ru := range rules {
		if ru.VisibilityType == model.VisibilityGroup {
			groupRuleIDs = append(groupRuleIDs, ru.ID)
		}
	}
	if len(groupRuleIDs) == 0 {
		return result
	}

	ruleGroupMap, err := model.GetRuleVisibilityGroupIDs(ctx, groupRuleIDs)
	if err != nil {
		slog.Error("[RuleVisibility] 批量查询规范分组关联失败", "error", err)
		return result
	}

	groupIDSet := make(map[uint]bool)
	for _, gids := range ruleGroupMap {
		for _, gid := range gids {
			groupIDSet[gid] = true
		}
	}
	if len(groupIDSet) == 0 {
		return result
	}
	allGroupIDs := make([]uint, 0, len(groupIDSet))
	for gid := range groupIDSet {
		allGroupIDs = append(allGroupIDs, gid)
	}

	groups, rerr := model.GetGroupsByIDs(ctx, allGroupIDs)
	if rerr != nil {
		slog.Error("[RuleVisibility] 批量查询分组名称失败", "error", rerr)
		return result
	}
	groupNameMap := make(map[uint]string)
	for _, g := range groups {
		groupNameMap[g.ID] = g.Name
	}

	for ruleID, gids := range ruleGroupMap {
		for _, gid := range gids {
			name, ok := groupNameMap[gid]
			if !ok {
				slog.Warn("[RuleVisibility] 分组已不存在，跳过", "group_id", gid, "rule_id", ruleID)
				continue
			}
			result[ruleID] = append(result[ruleID], ruleVisibilityGroupInfo{GroupID: gid, GroupName: name})
		}
	}
	return result
}

// -----------------------------------------------------------------------------
// GET /admin/rules
// -----------------------------------------------------------------------------

// HandleAdminRules 查询规范列表（分页，每个 slug 只返回最新版本）。
//
// 与 HandleAdminSkills 的差异：
//   - 无 category_ids / visibility_type / group_id 三组筛选
//   - 新增 type / source 筛选
//   - `last_task` 结构一致（对应前端已有渲染）
func HandleAdminRules(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	page, pageSize := parsePagination(r)
	slog.Info("查询规范列表",
		"keyword", r.URL.Query().Get("keyword"),
		"type", r.URL.Query().Get("type"),
		"source", r.URL.Query().Get("source"),
		"page", page, "page_size", pageSize)

	// 基础查询：每个 slug 取最新版本
	db := model.DB(r.Context()).Model(&model.EnterpriseRule{}).
		Where("id IN (?)", model.LatestVersionRuleIDs(r.Context()))

	if keyword := r.URL.Query().Get("keyword"); keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("name LIKE ? OR description LIKE ?", like, like)
	}
	if name := r.URL.Query().Get("name"); name != "" {
		db = db.Where("name LIKE ?", "%"+name+"%")
	}
	if desc := r.URL.Query().Get("description"); desc != "" {
		db = db.Where("description LIKE ?", "%"+desc+"%")
	}
	if t := r.URL.Query().Get("type"); t != "" {
		db = db.Where("type = ?", t)
	}
	if src := r.URL.Query().Get("source"); src != "" {
		db = db.Where("source = ?", src)
	}

	// 应用范围筛选：分组与项目同时传入时取并集，避免遗漏任一目标可见的资源。
	vtFilter := r.URL.Query().Get("visibility_type")
	var parsedGIDs []int
	if gidStr := r.URL.Query().Get("group_id"); gidStr != "" {
		for _, s := range strings.Split(gidStr, ",") {
			if id, err := strconv.Atoi(strings.TrimSpace(s)); err == nil && id > 0 {
				parsedGIDs = append(parsedGIDs, id)
			}
		}
	}
	projectIDs, _ := parseUintCSV(r.URL.Query().Get("project_id"))
	db = applyRuleScopeFilter(r.Context(), db, vtFilter, parsedGIDs, projectIDs)

	var total int64
	db.Count(&total)

	var rules []model.EnterpriseRule
	db.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rules)

	// last_task 聚合（与 skill 侧结构对齐）
	type lastTask struct {
		TaskID    uint      `json:"task_id"`
		Status    string    `json:"status"`
		Type      string    `json:"type"`
		Total     int       `json:"total"`
		Success   int       `json:"success"`
		Failed    int       `json:"failed"`
		Version   string    `json:"version"`
		CreatedAt time.Time `json:"created_at"`
	}
	taskMap := make(map[uint]*lastTask)
	if len(rules) > 0 {
		ruleIDs := make([]uint, len(rules))
		for i, ru := range rules {
			ruleIDs[i] = ru.ID
		}
		var tasks []model.RuleDistributionTask
		subQuery := model.DB(r.Context()).Model(&model.RuleDistributionTask{}).
			Select("MAX(id)").Where("rule_id IN ?", ruleIDs).Group("rule_id")
		model.DB(r.Context()).Where("id IN (?)", subQuery).Find(&tasks)

		if len(tasks) > 0 {
			taskIDs := make([]uint, len(tasks))
			for i, t := range tasks {
				taskIDs[i] = t.ID
			}
			type taskStatusCount struct {
				TaskID uint
				Status string
				Count  int
			}
			var counts []taskStatusCount
			if err := model.DB(r.Context()).Model(&model.RuleDistributionRecord{}).
				Select("task_id, status, COUNT(*) as count").
				Where("task_id IN ?", taskIDs).
				Group("task_id, status").
				Scan(&counts).Error; err != nil {
				slog.Error("查询规范下发记录聚合失败", "error", err)
			}
			type counters struct{ Success, Failed int }
			cm := make(map[uint]*counters)
			for _, c := range counts {
				if cm[c.TaskID] == nil {
					cm[c.TaskID] = &counters{}
				}
				switch c.Status {
				case model.RuleRecordStatusSuccess:
					cm[c.TaskID].Success = c.Count
				case model.RuleRecordStatusFailed,
					model.RuleRecordStatusUpgradeFailed,
					model.RuleRecordStatusUninstallFailedOld:
					cm[c.TaskID].Failed += c.Count
				}
			}
			for _, t := range tasks {
				lt := &lastTask{
					TaskID:    t.ID,
					Status:    t.Status,
					Type:      t.Type,
					Total:     t.Total,
					Version:   t.Version,
					CreatedAt: t.CreatedAt,
				}
				if c := cm[t.ID]; c != nil {
					lt.Success = c.Success
					lt.Failed = c.Failed
				}
				taskMap[t.RuleID] = lt
			}
		}
	}

	type ruleResp struct {
		model.EnterpriseRule
		LastTask           *lastTask                 `json:"last_task"`
		VisibilityGroups   []ruleVisibilityGroupInfo `json:"visibility_groups"`
		VisibilityProjects []projectVisibilityInfo   `json:"visibility_projects"`
	}
	visibilityMap := buildRuleVisibilityData(r.Context(), rules)
	ruleSlugs := make([]string, 0, len(rules))
	for _, rule := range rules {
		ruleSlugs = append(ruleSlugs, rule.Slug)
	}
	projectVisibilityMap, err := buildProjectVisibilityData(r.Context(), model.ProjectConfigTypeRule, ruleSlugs)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
		return
	}
	result := make([]ruleResp, 0, len(rules))
	for _, ru := range rules {
		visGroups := visibilityMap[ru.ID]
		if visGroups == nil {
			visGroups = []ruleVisibilityGroupInfo{}
		}
		visProjects := projectVisibilityMap[ru.Slug]
		if visProjects == nil {
			visProjects = []projectVisibilityInfo{}
		}
		result = append(result, ruleResp{
			EnterpriseRule:     ru,
			LastTask:           taskMap[ru.ID],
			VisibilityGroups:   visGroups,
			VisibilityProjects: visProjects,
		})
	}

	jsonOK(w, map[string]interface{}{
		"rules":     result,
		"page":      page,
		"page_size": pageSize,
		"total":     total,
	})
}

func applyRuleScopeFilter(ctx context.Context, db *gorm.DB, visibilityType string, groupIDs []int, projectIDs []uint) *gorm.DB {
	var groupQuery *gorm.DB
	if len(groupIDs) > 0 {
		groupQuery = model.DB(ctx).Model(&model.RuleVisibilityGroup{}).
			Select("rule_id").Where("group_id IN ?", groupIDs)
	}
	var projectQuery *gorm.DB
	if len(projectIDs) > 0 {
		projectQuery = model.DB(ctx).Model(&model.ProjectConfigBinding{}).
			Select("config_key").Where("config_type = ? AND project_id IN ?", model.ProjectConfigTypeRule, projectIDs)
	}
	switch {
	case groupQuery != nil && projectQuery != nil && visibilityType != "":
		return db.Where("visibility_type = ? OR id IN (?) OR slug IN (?)", visibilityType, groupQuery, projectQuery)
	case groupQuery != nil && projectQuery != nil:
		return db.Where("id IN (?) OR slug IN (?)", groupQuery, projectQuery)
	case groupQuery != nil && visibilityType != "":
		return db.Where("visibility_type = ? OR id IN (?)", visibilityType, groupQuery)
	case groupQuery != nil:
		return db.Where("id IN (?)", groupQuery)
	case projectQuery != nil:
		return db.Where("slug IN (?)", projectQuery)
	case visibilityType != "":
		return db.Where("visibility_type = ?", visibilityType)
	default:
		return db
	}
}

// -----------------------------------------------------------------------------
// GET /admin/rules/detail
// -----------------------------------------------------------------------------

// HandleAdminRuleDetail 查询规范详情。
func HandleAdminRuleDetail(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	slug := r.URL.Query().Get("slug")
	versionParam := r.URL.Query().Get("version")
	slog.Info("查询规范详情", "slug", slug, "version", versionParam)
	if slug == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillSlugRequired))
		return
	}

	var rule model.EnterpriseRule
	if versionParam == "" || versionParam == "latest" {
		if model.DB(r.Context()).Where("slug = ?", slug).
			Order("version_major DESC, version_minor DESC, version_patch DESC").
			First(&rule).Error != nil {
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgRuleNotExist))
			return
		}
	} else {
		if model.DB(r.Context()).Where("slug = ? AND version = ?", slug, versionParam).
			First(&rule).Error != nil {
			writeError(w, r, http.StatusNotFound,
				hcommon.I18nError(i18n.MsgRuleVersionNotFound, slug, versionParam))
			return
		}
	}

	// 全部版本
	var all []model.EnterpriseRule
	model.DB(r.Context()).Where("slug = ?", slug).
		Order("version_major DESC, version_minor DESC, version_patch DESC").Find(&all)
	versions := make([]string, 0, len(all))
	for _, v := range all {
		versions = append(versions, v.Version)
	}

	// visibility_groups（仅 group 类型展开）
	var visGroups []ruleVisibilityGroupInfo
	if rule.VisibilityType == model.VisibilityGroup {
		groupMap, _ := model.GetRuleVisibilityGroupIDs(r.Context(), []uint{rule.ID})
		if gids, ok := groupMap[rule.ID]; ok && len(gids) > 0 {
			if groups, gerr := model.GetGroupsByIDs(r.Context(), gids); gerr == nil {
				nameMap := make(map[uint]string, len(groups))
				for _, g := range groups {
					nameMap[g.ID] = g.Name
				}
				for _, gid := range gids {
					if name, ok := nameMap[gid]; ok {
						visGroups = append(visGroups, ruleVisibilityGroupInfo{GroupID: gid, GroupName: name})
					}
				}
			}
		}
	}
	if visGroups == nil {
		visGroups = []ruleVisibilityGroupInfo{}
	}
	projectVisibilityMap, err := buildProjectVisibilityData(r.Context(), model.ProjectConfigTypeRule, []string{rule.Slug})
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
		return
	}
	visProjects := projectVisibilityMap[rule.Slug]
	if visProjects == nil {
		visProjects = []projectVisibilityInfo{}
	}

	jsonOK(w, map[string]interface{}{
		"rule":                rule,
		"versions":            versions,
		"visibility_groups":   visGroups,
		"visibility_projects": visProjects,
	})
}

// -----------------------------------------------------------------------------
// POST /admin/rules/create
// -----------------------------------------------------------------------------

// HandleCreateRule 上传规范新版本（首次插入或递增版本）。
func HandleCreateRule(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	if err := r.ParseMultipartForm(maxRuleUploadSize + (1 << 15)); err != nil {
		slog.Warn("规范上传请求解析失败", "error", err)
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nRichError(err, i18n.MsgRequestBodyTooLargeWithError, err))
		return
	}

	slug := r.FormValue("slug")
	name := r.FormValue("name")
	version := r.FormValue("version")
	ruleType := r.FormValue("type")
	slog.Info("开始创建规范", "slug", slug, "name", name, "version", version, "type", ruleType)

	if slug == "" || name == "" || version == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgPluginSlugNameVerRequired))
		return
	}
	if !isValidSlug(slug) {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgPluginInvalidSlug))
		return
	}
	if err := validateRuleType(ruleType); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 可见范围参数（可选）：parseVisibilityParams 依赖 r.Form，必须在 ParseMultipartForm 之后调用
	visType, visGroupIDs, visProjectIDs, hasScope, visErr := parseVisibilityParams(r)
	if visErr != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(visErr))
		return
	}
	// 类型一致性：同 slug 后续版本 type 必须与首版一致。
	// 用 First(order asc) 取首版；用 Unscoped 保留软删版本作参照，避免删除历史后允许重定 type。
	var firstVersion model.EnterpriseRule
	firstErr := model.DB(r.Context()).Unscoped().Where("slug = ?", slug).
		Order("version_major ASC, version_minor ASC, version_patch ASC").
		First(&firstVersion).Error
	if firstErr == nil && firstVersion.Type != ruleType {
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgRuleTypeMismatch, slug, firstVersion.Type))
		return
	}

	// 组装 model 并解析版本号
	rule := model.EnterpriseRule{
		Slug:        slug,
		Name:        name,
		Description: r.FormValue("description"),
		Type:        ruleType,
		Source:      model.EnterpriseRuleSourceEnterprise,
		Version:     version,
		Changelog:   r.FormValue("changelog"),
	}
	if err := rule.ParseVersion(); err != nil {
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nRichError(err, i18n.MsgBadRequestParamInvalid, "version"))
		return
	}

	// 版本递增校验
	var maxExisting model.EnterpriseRule
	if model.DB(r.Context()).Where("slug = ?", slug).
		Order("version_major DESC, version_minor DESC, version_patch DESC").
		First(&maxExisting).Error == nil {
		newScore := rule.VersionMajor*1_000_000 + rule.VersionMinor*1_000 + rule.VersionPatch
		existingScore := maxExisting.VersionMajor*1_000_000 + maxExisting.VersionMinor*1_000 + maxExisting.VersionPatch
		if newScore <= existingScore {
			writeError(w, r, http.StatusBadRequest,
				hcommon.I18nError(i18n.MsgRuleNewVersionMustBeGreater, version, maxExisting.Version))
			return
		}
	}

	// 唯一 (slug, version) 检查（Unscoped 复活软删记录，避免 DB 唯一索引冲突）
	var existing model.EnterpriseRule
	if err := model.DB(r.Context()).Unscoped().
		Where("slug = ? AND version = ?", slug, version).First(&existing).Error; err == nil {
		if existing.DeletedAt.Valid {
			// 复活：先物理删除软删记录，然后走正常 create
			model.DB(r.Context()).Unscoped().Delete(&model.EnterpriseRule{}, existing.ID)
		} else {
			writeError(w, r, http.StatusBadRequest,
				hcommon.I18nError(i18n.MsgRuleVersionExist, slug, version))
			return
		}
	}

	// 三期：Hook 资源走无文件创建路径（event + cmd 表单字段，不传 md 文件）
	if ruleType == model.EnterpriseRuleTypeHook {
		handleCreateHookRule(w, r, &rule)
		return
	}

	// 读取 md 文件
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgRuleFileFieldMissing))
		return
	}
	defer file.Close()

	if header.Size <= 0 || header.Size > maxRuleUploadSize {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgRuleFileMustBeMD))
		return
	}
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".md") {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgRuleFileMustBeMD))
		return
	}

	data, err := io.ReadAll(io.LimitReader(file, maxRuleUploadSize+1))
	if err != nil {
		slog.Error("读取上传规范失败", "slug", slug, "version", version, "error", err)
		writeError(w, r, http.StatusInternalServerError,
			hcommon.I18nRichError(err, i18n.MsgRuleReadUploadFail, err))
		return
	}
	if int64(len(data)) > maxRuleUploadSize {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgRuleFileMustBeMD))
		return
	}
	if err := validateRuleFileContent(data); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	rule.COSKey = buildRuleCOSKey(slug, version)
	rule.FileSize = header.Size
	rule.ContentSHA256 = sha256Hex(data)

	// ── 先写 DB，再上传 SMH ──
	tx := model.DB(r.Context()).Begin()
	if err := tx.Create(&rule).Error; err != nil {
		tx.Rollback()
		if isDuplicateKeyError(err) {
			writeError(w, r, http.StatusBadRequest,
				hcommon.I18nError(i18n.MsgRuleVersionExist, slug, version))
			return
		}
		slog.Error("规范写入数据库失败", "slug", slug, "version", version, "error", err)
		writeError(w, r, http.StatusInternalServerError,
			hcommon.I18nRichError(err, i18n.MsgRuleCreateRecordFail, err))
		return
	}

	// 从同 slug 旧版本继承 distribute_count
	if err := model.InheritRuleDistributeCount(tx, slug, rule.ID); err != nil {
		slog.Warn("继承规范 distribute_count 失败", "slug", slug, "version", version, "error", err)
	}

	// 可见范围：
	//   - 显式传了 visibility_type → SetRuleVisibility
	//   - 未传 → CopyRuleVisibility（从同 slug 旧版本复制）
	var removedProjectIDs, removedGroupIDs []uint
	if hasScope {
		oldGroupIDs, oldProjectIDs, dErr := loadRuleScopeIDs(tx, rule.ID, slug)
		if dErr != nil {
			slog.Error("查询规范旧可见范围失败", "slug", slug, "version", version, "error", dErr)
			tx.Rollback()
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(dErr, i18n.MsgInternalError))
			return
		}
		if err := model.SetRuleVisibility(tx, rule.ID, visType, visGroupIDs); err != nil {
			slog.Error("设置规范可见范围失败", "slug", slug, "version", version, "error", err)
			tx.Rollback()
			writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
			return
		}
		if err := model.ReplaceResourceProjectBindings(tx, model.ProjectConfigTypeRule, rule.Slug, visProjectIDs); err != nil {
			tx.Rollback()
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
			return
		}
		removedGroupIDs, removedProjectIDs = diffRemovedScope(visType, oldGroupIDs, oldProjectIDs, visGroupIDs, visProjectIDs)
	} else {
		if err := model.CopyRuleVisibility(tx, slug, rule.ID); err != nil {
			slog.Warn("复制规范可见范围失败", "slug", slug, "version", version, "error", err)
			// 不阻断主流程：默认 all 可见，后续可手动修正
		}
	}

	// 上传 SMH
	storageClient, storageErr := getStorageClient(r.Context())
	if storageErr != nil {
		slog.Error("获取存储客户端失败", "slug", slug, "version", version, "error", storageErr)
		tx.Rollback()
		writeError(w, r, http.StatusInternalServerError,
			hcommon.I18nRichError(storageErr, i18n.MsgPluginSMHUnavailable, storageErr))
		return
	}
	if err := storageClient.Upload(rule.COSKey, data, "text/markdown"); err != nil {
		slog.Error("规范上传 SMH 失败", "slug", slug, "version", version, "cos_key", rule.COSKey, "error", err)
		tx.Rollback()
		writeError(w, r, http.StatusInternalServerError,
			hcommon.I18nRichError(err, i18n.MsgRuleUploadSMHFail, err))
		return
	}

	if err := publishScopeRemoval(r.Context(), tx, model.AssetTypeRule, slug, removedProjectIDs, removedGroupIDs); err != nil {
		tx.Rollback()
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgRuleUpdateFail, err))
		return
	}
	// 范围缩小会先移除直接资产绑定，避免新版记录或下发落到已移出范围的目标。
	// 首次创建（slug 无旧版本）不触发：此时尚未关联到任何具体项目/分组。
	if maxExisting.ID != 0 {
		fromVer := maxExisting.Version
		publishAssetVersionForChange(r.Context(), tx, model.AssetTypeRule, slug, fromVer, version, model.TriggerReasonAssetPublished)
	}

	if err := tx.Commit().Error; err != nil {
		// 事务提交失败：SMH 已上传，异步清理，避免孤儿文件
		slog.Error("规范事务提交失败", "slug", slug, "version", version, "error", err)
		go func(ctx context.Context, key string) {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("规范事务提交失败后清理 SMH 时 panic", "recover", r)
				}
			}()
			if c, e := getStorageClient(ctx); e == nil {
				_ = c.Delete(key, true)
			}
		}(hcommon.DetachContext(r.Context()), rule.COSKey)
		writeError(w, r, http.StatusInternalServerError,
			hcommon.I18nRichError(err, i18n.MsgRuleCreateRecordFail, err))
		return
	}

	slog.Info("规范创建成功", "slug", slug, "version", version, "id", rule.ID, "size", header.Size)
	jsonOK(w, map[string]interface{}{
		"ok":      true,
		"id":      rule.ID,
		"slug":    slug,
		"version": version,
	})
}

// handleCreateHookRule 处理 type=hook 的规范创建（三期）。
//
// 与 prompt/rule 不同：hook 不需要上传 md 文件，改用表单字段 event + cmd 定义
// 触发时机与执行命令。其余可见范围校验与 prompt/rule 完全一致（复用调用方已解析的
// visType/visGroupIDs/visProjectIDs/hasScope）。
//
// 调用前已完成：requireAdmin / ParseMultipartForm / 基础字段校验 / 类型一致性 /
// 版本递增 / 唯一 (slug,version) 检查。rule 已填充 Slug/Name/Description/Type/Version/Changelog。
func handleCreateHookRule(w http.ResponseWriter, r *http.Request, rule *model.EnterpriseRule) {
	event := r.FormValue("event")
	cmd := r.FormValue("cmd")
	if !model.IsValidHookEvent(event) {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgHookEventInvalid, event))
		return
	}
	if strings.TrimSpace(cmd) == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgHookCmdRequired))
		return
	}
	rule.Event = event
	rule.Cmd = cmd
	// hook 无文件：COSKey / FileSize / ContentSHA256 保持零值

	// 可见范围（与 prompt/rule 一致）：显式传 visibility_type → SetRuleVisibility，否则从旧版本复制
	visType, visGroupIDs, visProjectIDs, hasScope, visErr := parseVisibilityParams(r)
	if visErr != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(visErr))
		return
	}

	tx := model.DB(r.Context()).Begin()
	if err := tx.Create(&rule).Error; err != nil {
		tx.Rollback()
		if isDuplicateKeyError(err) {
			writeError(w, r, http.StatusBadRequest,
				hcommon.I18nError(i18n.MsgRuleVersionExist, rule.Slug, rule.Version))
			return
		}
		slog.Error("[AdminRules] Hook 规范写入数据库失败", "slug", rule.Slug, "version", rule.Version, "error", err)
		writeError(w, r, http.StatusInternalServerError,
			hcommon.I18nRichError(err, i18n.MsgRuleCreateRecordFail, err))
		return
	}

	if hasScope {
		if err := model.SetRuleVisibility(tx, rule.ID, visType, visGroupIDs); err != nil {
			tx.Rollback()
			slog.Error("[AdminRules] 设置 Hook 可见范围失败", "slug", rule.Slug, "error", err)
			writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
			return
		}
		if err := model.ReplaceResourceProjectBindings(tx, model.ProjectConfigTypeRule, rule.Slug, visProjectIDs); err != nil {
			tx.Rollback()
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
			return
		}
	} else if err := model.CopyRuleVisibility(tx, rule.Slug, rule.ID); err != nil {
		tx.Rollback()
		slog.Error("[AdminRules] 复制 Hook 可见范围失败", "slug", rule.Slug, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
		return
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		slog.Error("[AdminRules] Hook 规范提交事务失败", "slug", rule.Slug, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
		return
	}

	slog.Info("[AdminRules] Hook 规范创建成功", "slug", rule.Slug, "version", rule.Version, "id", rule.ID, "event", event)
	jsonOK(w, map[string]interface{}{
		"ok":      true,
		"id":      rule.ID,
		"slug":    rule.Slug,
		"version": rule.Version,
	})
}

// -----------------------------------------------------------------------------
// POST /admin/rules/delete
// -----------------------------------------------------------------------------

// HandleDeleteRule 删除规范（不级联卸载已装实例）。
//
// 语义严格对齐 HandleDeleteSkill：
//   - 不传 version → 全版本删除；传 version → 单版本删除
//   - 前置检查：如果该 slug 涉及的 rule 存在 running task 则拒绝
//   - 事务内软删；事务外异步删 SMH 上对应版本的 md
func HandleDeleteRule(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	slug := r.FormValue("slug")
	version := r.FormValue("version")
	slog.Info("开始删除规范", "slug", slug, "version", version)
	if slug == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillSlugRequired))
		return
	}

	var rules []model.EnterpriseRule
	if version != "" {
		var one model.EnterpriseRule
		if model.DB(r.Context()).Where("slug = ? AND version = ?", slug, version).
			First(&one).Error != nil {
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgRuleNotExist))
			return
		}
		rules = append(rules, one)
	} else {
		model.DB(r.Context()).Where("slug = ?", slug).Find(&rules)
		if len(rules) == 0 {
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgRuleNotExist))
			return
		}
	}

	ruleIDs := make([]uint, len(rules))
	deletedVersions := make([]string, len(rules))
	for i, ru := range rules {
		ruleIDs[i] = ru.ID
		deletedVersions[i] = ru.Version
	}

	txErr := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		var runningCount int64
		if err := tx.Model(&model.RuleDistributionTask{}).
			Where("rule_id IN ? AND status = ?", ruleIDs, "running").
			Set("gorm:query_option", "FOR UPDATE").
			Count(&runningCount).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgRuleCheckTaskFail, err)
		}
		if runningCount > 0 {
			return errRuleHasRunningTask
		}
		if err := tx.Where("id IN ?", ruleIDs).Delete(&model.EnterpriseRule{}).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgRuleDeleteRecordFail, err)
		}
		// 同事务清理可见性关联（对齐 skill 侧 delete 路径）
		for _, id := range ruleIDs {
			if err := model.CleanupRuleVisibilityByRuleID(tx, id); err != nil {
				return err
			}
		}
		// 旁路：删除规范前先记录版本历史（asset_deleted，同事务）。
		// 必须在 CleanupProjectBindings 之前调用，否则绑定已删、查不到受影响目标。
		publishAssetVersionForChange(r.Context(), tx, model.AssetTypeRule, slug, "", "", model.TriggerReasonAssetDeleted)

		var remaining int64
		if err := tx.Model(&model.EnterpriseRule{}).Where("slug = ?", slug).Count(&remaining).Error; err != nil {
			return err
		}
		if remaining == 0 {
			if err := model.CleanupProjectBindings(tx, model.ProjectConfigTypeRule, slug); err != nil {
				return err
			}
		}
		return nil
	})

	if txErr != nil {
		if errors.Is(txErr, errRuleHasRunningTask) {
			writeError(w, r, http.StatusBadRequest, errRuleHasRunningTask)
		} else {
			slog.Error("删除规范事务失败", "slug", slug, "version", version, "error", txErr)
			writeError(w, r, http.StatusInternalServerError,
				hcommon.I18nRichError(txErr, i18n.MsgRuleDeleteFail, txErr))
		}
		return
	}

	// 事务外异步删 SMH
	go func(ctx context.Context, versions []string) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("规范 SMH 清理 panic", "slug", slug, "recover", r)
			}
		}()
		client, err := getStorageClient(ctx)
		if err != nil {
			slog.Warn("规范 SMH 清理跳过：客户端不可用", "slug", slug, "error", err)
			return
		}
		for _, v := range versions {
			key := buildRuleCOSKey(slug, v)
			if err := client.Delete(key, true); err != nil {
				slog.Warn("规范 SMH 清理失败", "slug", slug, "version", v, "cos_key", key, "error", err)
			}
		}
		slog.Info("规范 SMH 清理完成", "slug", slug, "versions", versions)
	}(hcommon.DetachContext(r.Context()), deletedVersions)

	slog.Info("规范删除成功", "slug", slug, "version", version, "deleted_rules", len(rules))
	jsonOK(w, map[string]interface{}{
		"ok":            true,
		"deleted_rules": len(rules),
	})
}

// -----------------------------------------------------------------------------
// GET /admin/rules/files
// -----------------------------------------------------------------------------

// HandleAdminRuleFiles 查询某 slug 全部版本的下载 URL 与元信息。
// URL 通过 buildSMHDownloadURL(cosKey, false) 生成（走公网，reporter 在用户机器上下载）。
func HandleAdminRuleFiles(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	slug := r.URL.Query().Get("slug")
	slog.Info("查询规范文件列表", "slug", slug)
	if slug == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillSlugRequired))
		return
	}

	var rules []model.EnterpriseRule
	model.DB(r.Context()).Where("slug = ?", slug).
		Order("version_major DESC, version_minor DESC, version_patch DESC").Find(&rules)
	if len(rules) == 0 {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgRuleNotExist))
		return
	}

	type versionInfo struct {
		Version       string `json:"version"`
		DownloadURL   string `json:"download_url"`
		ContentSHA256 string `json:"content_sha256"`
		FileSize      int64  `json:"file_size"`
	}
	result := make([]versionInfo, 0, len(rules))
	for _, ru := range rules {
		vi := versionInfo{
			Version:       ru.Version,
			ContentSHA256: ru.ContentSHA256,
			FileSize:      ru.FileSize,
		}
		if url, err := buildSMHDownloadURL(r.Context(), ru.COSKey, false); err == nil {
			vi.DownloadURL = url
		} else {
			slog.Warn("规范下载 URL 生成失败", "slug", slug, "version", ru.Version, "error", err)
		}
		result = append(result, vi)
	}

	jsonOK(w, map[string]interface{}{
		"slug":     slug,
		"versions": result,
	})
}

// -----------------------------------------------------------------------------
// POST /admin/rules/update
// -----------------------------------------------------------------------------

// HandleAdminRuleUpdate 更新规范元信息（name / description / visibility）。
func HandleAdminRuleUpdate(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	slug := r.FormValue("slug")
	version := r.FormValue("version")
	slog.Info("开始更新规范", "slug", slug, "version", version)
	if slug == "" || version == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgRuleSlugVersionRequired))
		return
	}

	var rule model.EnterpriseRule
	if model.DB(r.Context()).Where("slug = ? AND version = ?", slug, version).First(&rule).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgRuleNotExist))
		return
	}

	updates := map[string]interface{}{}
	if name := r.FormValue("name"); name != "" {
		updates["name"] = name
	}
	if desc := r.FormValue("description"); desc != "" {
		updates["description"] = desc
	}

	// 预解析可见范围参数
	visType, visGroupIDs, visProjectIDs, hasScope, visErr := parseVisibilityParams(r)
	if visErr != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(visErr))
		return
	}
	var removedProjectIDs, removedGroupIDs []uint
	if err := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		if len(updates) > 0 {
			if err := tx.Model(&rule).Updates(updates).Error; err != nil {
				return err
			}
		}
		if hasScope {
			// Set 之前先查旧范围，用于 diff 出被移出的存量分组/项目
			oldGroupIDs, oldProjectIDs, dErr := loadRuleScopeIDs(tx, rule.ID, rule.Slug)
			if dErr != nil {
				return dErr
			}
			if err := model.SetRuleVisibility(tx, rule.ID, visType, visGroupIDs); err != nil {
				return err
			}
			if err := model.ReplaceResourceProjectBindings(tx, model.ProjectConfigTypeRule, rule.Slug, visProjectIDs); err != nil {
				return err
			}
			removedGroupIDs, removedProjectIDs = diffRemovedScope(visType, oldGroupIDs, oldProjectIDs, visGroupIDs, visProjectIDs)
		}
		return publishScopeRemoval(r.Context(), tx, model.AssetTypeRule, slug, removedProjectIDs, removedGroupIDs)
	}); err != nil {
		slog.Error("规范更新事务失败", "slug", slug, "version", version, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgRuleUpdateFail, err))
		return
	}

	slog.Info("规范更新成功", "slug", slug, "version", version, "id", rule.ID)
	jsonOK(w, map[string]interface{}{"ok": true})
}

// loadRuleScopeIDs 查询某 rule 当前的应用范围（分组可见性 + 项目绑定）ID 列表。
func loadRuleScopeIDs(tx *gorm.DB, ruleID uint, slug string) (groupIDs, projectIDs []uint, err error) {
	if e := tx.Model(&model.RuleVisibilityGroup{}).
		Where("rule_id = ?", ruleID).Pluck("group_id", &groupIDs).Error; e != nil {
		return nil, nil, e
	}
	if e := tx.Model(&model.ProjectConfigBinding{}).
		Where("config_type = ? AND config_key = ?", model.ProjectConfigTypeRule, slug).
		Pluck("project_id", &projectIDs).Error; e != nil {
		return nil, nil, e
	}
	return groupIDs, projectIDs, nil
}

// -----------------------------------------------------------------------------
// GET /admin/rules/tasks
// -----------------------------------------------------------------------------

// ruleRecordResp 与 skill 接口对齐：每条下发/卸载记录明细。
type ruleRecordResp struct {
	InstanceID    uint   `json:"instance_id"`
	CVMInstanceID string `json:"cvm_instance_id"`
	InstanceName  string `json:"instance_name"`
	Username      string `json:"username"`
	Status        string `json:"status"`
	Error         string `json:"error"`
}

// ruleTaskResp 在任务基础上附加 records 明细（与 /admin/skills/tasks 对齐）。
type ruleTaskResp struct {
	model.RuleDistributionTask
	Records []ruleRecordResp `json:"records"`
}

// HandleAdminRuleTasks 查询规范的分发/卸载任务列表。
func HandleAdminRuleTasks(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	slug := strings.TrimSpace(r.URL.Query().Get("slug"))
	batchID := strings.TrimSpace(r.URL.Query().Get("batch_id"))
	slog.Info("查询规范下发任务列表", "slug", slug, "batch_id", batchID)
	if slug == "" && batchID == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgRuleSlugOrBatchIDRequired))
		return
	}

	page, pageSize := parsePagination(r)
	typeFilter := r.URL.Query().Get("type")

	var total int64
	taskQuery := model.DB(r.Context()).Model(&model.RuleDistributionTask{})
	if batchID != "" {
		taskQuery = taskQuery.Where("batch_id = ?", batchID)
	} else {
		var ruleIDs []uint
		model.DB(r.Context()).Model(&model.EnterpriseRule{}).Where("slug = ?", slug).Pluck("id", &ruleIDs)
		if len(ruleIDs) == 0 {
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgRuleNotExist))
			return
		}
		taskQuery = taskQuery.Where("rule_id IN ?", ruleIDs)
	}
	if typeFilter != "" && typeFilter != "all" {
		taskQuery = taskQuery.Where("type = ?", typeFilter)
	}
	taskQuery.Count(&total)

	var tasks []model.RuleDistributionTask
	taskQuery.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&tasks)

	// 聚合 record 统计
	type recordCount struct {
		TaskID uint
		Status string
		Count  int
	}
	if len(tasks) > 0 {
		taskIDs := make([]uint, len(tasks))
		for i, t := range tasks {
			taskIDs[i] = t.ID
		}
		var counts []recordCount
		if err := model.DB(r.Context()).Model(&model.RuleDistributionRecord{}).
			Select("task_id, status, COUNT(*) as count").
			Where("task_id IN ?", taskIDs).
			Group("task_id, status").
			Scan(&counts).Error; err != nil {
			slog.Error("查询规范下发记录聚合失败", "error", err)
		}
		cm := make(map[uint]map[string]int)
		for _, c := range counts {
			if cm[c.TaskID] == nil {
				cm[c.TaskID] = make(map[string]int)
			}
			cm[c.TaskID][c.Status] = c.Count
		}
		for i := range tasks {
			if c, ok := cm[tasks[i].ID]; ok {
				tasks[i].Success = c[model.RuleRecordStatusSuccess]
				tasks[i].Failed = c[model.RuleRecordStatusFailed] +
					c[model.RuleRecordStatusUpgradeFailed] +
					c[model.RuleRecordStatusUninstallFailedOld]
			}
		}

		// 查询每条 record 明细，按 task_id 分组（与 /admin/skills/tasks 对齐）
		var allRecords []model.RuleDistributionRecord
		model.DB(r.Context()).Where("task_id IN ?", taskIDs).Find(&allRecords)
		recordsByTask := make(map[uint][]model.RuleDistributionRecord)
		instIDSet := make(map[uint]struct{})
		for _, rec := range allRecords {
			recordsByTask[rec.TaskID] = append(recordsByTask[rec.TaskID], rec)
			if rec.InstanceID > 0 {
				instIDSet[rec.InstanceID] = struct{}{}
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
			var instUsers []model.User
			model.DB(r.Context()).Where("id IN ?", instUserIDList).Find(&instUsers)
			for _, u := range instUsers {
				instUserMap[u.ID] = u.Username
			}
		}

		result := make([]ruleTaskResp, 0, len(tasks))
		for _, t := range tasks {
			tr := ruleTaskResp{RuleDistributionTask: t}
			for _, rec := range recordsByTask[t.ID] {
				rr := ruleRecordResp{
					InstanceID:    rec.InstanceID,
					CVMInstanceID: rec.InstanceCID,
					Status:        rec.Status,
					Error:         rec.Error,
				}
				if inst, ok := instMap[rec.InstanceID]; ok {
					rr.InstanceName = inst.Name
					rr.Username = instUserMap[inst.UserID]
				}
				tr.Records = append(tr.Records, rr)
			}
			if tr.Records == nil {
				tr.Records = []ruleRecordResp{}
			}
			result = append(result, tr)
		}
		jsonOK(w, map[string]interface{}{
			"tasks":     result,
			"page":      page,
			"page_size": pageSize,
			"total":     total,
		})
		return
	}

	jsonOK(w, map[string]interface{}{
		"tasks":     []ruleTaskResp{},
		"page":      page,
		"page_size": pageSize,
		"total":     total,
	})
}

// -----------------------------------------------------------------------------
// GET /admin/rules/instances
// -----------------------------------------------------------------------------

// HandleAdminRuleInstances 查询某规范 slug 的已安装实例列表。
func HandleAdminRuleInstances(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	slug := strings.TrimSpace(r.URL.Query().Get("slug"))
	statusFilter := r.URL.Query().Get("status")
	search := r.URL.Query().Get("search")
	groupIDStr := r.URL.Query().Get("group_id")
	slog.Info("查询规范安装实例", "slug", slug, "status", statusFilter, "group_id", groupIDStr)
	if slug == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillSlugRequired))
		return
	}

	var ruleIDs []uint
	latestVersion := ""
	model.DB(r.Context()).Model(&model.EnterpriseRule{}).Where("slug = ?", slug).Pluck("id", &ruleIDs)
	if len(ruleIDs) == 0 {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgRuleNotExist))
		return
	}
	model.DB(r.Context()).Model(&model.EnterpriseRule{}).Where("id IN ?", ruleIDs).
		Order("version_major DESC, version_minor DESC, version_patch DESC").
		Limit(1).Pluck("version", &latestVersion)

	page, pageSize := parsePagination(r, 500)

	type instResp struct {
		InstanceID   uint      `json:"instance_id"              gorm:"column:instance_id"`
		InstanceCID  string    `json:"cvm_instance_id"          gorm:"column:cvm_instance_id"`
		InstanceName string    `json:"instance_name"            gorm:"column:instance_name"`
		Source       string    `json:"source"                   gorm:"column:source"`
		Username     string    `json:"username"                 gorm:"column:username"`
		Version      string    `json:"version"                  gorm:"column:version"`
		Status       string    `json:"status"                   gorm:"column:status"`
		Error        string    `json:"error"                    gorm:"column:error"`
		CreatedAt    time.Time `json:"created_at"            gorm:"column:created_at"`
	}

	var total int64
	var items []instResp

	// 主表改为 instances + LEFT JOIN 最新下发记录 + 本地实例事实快照，在 SQL 层推导安装状态。
	// 这样从未下发过的本地实例（无 record 行）也能以 status=uninstalled 返回，
	// 与 skill 版 HandleAdminSkillInstances 的查询模型对齐。
	baseQuery := model.BuildRuleInstanceQuery(r.Context(), ruleIDs, latestVersion, slug)

	if search != "" {
		like := "%" + search + "%"
		// 与 skill 版 HandleAdminSkillInstances 对齐：只按实例名 / 实例 ID 匹配。
		baseQuery = baseQuery.Where("instances.name LIKE ? OR instances.instance_id LIKE ?", like, like)
	}

	// 安装状态筛选（支持逗号分隔多状态，如 status=uninstalled,failed）。
	// 复用状态 CASE 确保与 SELECT 中的 CASE 逻辑一致。
	if statusFilter != "" {
		statuses := strings.Split(statusFilter, ",")
		trimmed := make([]string, 0, len(statuses))
		for _, s := range statuses {
			if v := strings.TrimSpace(s); v != "" {
				trimmed = append(trimmed, v)
			}
		}
		if len(trimmed) > 0 {
			baseQuery = baseQuery.Where(model.RuleInstallStatusCase()+" IN ?", trimmed)
		}
	}

	// 按用户组筛选实例（辅助筛选，支持逗号分隔多个 group_id）。
	// group_id=0 表示未分组用户的实例，可与正常 group_id 组合使用，如 group_id=0,1,3。
	// 使用半连接（WHERE user_id IN / NOT IN 子查询）而非 JOIN user_group_members，
	// 因为一个用户可属于多个组，JOIN 会把同一实例扇出成多行导致结果重复（即 skill/mcp 的已知老 bug）。
	// rule 为本次新增代码，按正确实现来，不受 skill/mcp 老 bug 影响。
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
			// 未分组 + 指定分组：OR 语义
			ungroupedSubQ := model.DB(r.Context()).Model(&model.UserGroupMember{}).Select("DISTINCT user_id")
			groupedSubQ := model.DB(r.Context()).Model(&model.UserGroupMember{}).Select("DISTINCT user_id").Where("user_group_id IN ?", groupIDs)
			baseQuery = baseQuery.Where("instances.user_id NOT IN (?) OR instances.user_id IN (?)", ungroupedSubQ, groupedSubQ)
		} else if includeUngrouped {
			// 仅未分组
			ungroupedSubQ := model.DB(r.Context()).Model(&model.UserGroupMember{}).Select("DISTINCT user_id")
			baseQuery = baseQuery.Where("instances.user_id NOT IN (?)", ungroupedSubQ)
		} else if len(groupIDs) > 0 {
			// 仅指定分组（半连接，避免跨组用户被 JOIN 扇出重复）
			groupedSubQ := model.DB(r.Context()).Model(&model.UserGroupMember{}).Select("DISTINCT user_id").Where("user_group_id IN ?", groupIDs)
			baseQuery = baseQuery.Where("instances.user_id IN (?)", groupedSubQ)
		}
	}

	baseQuery.Count(&total)
	baseQuery.Order("instances.id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Scan(&items)

	if items == nil {
		items = []instResp{}
	}

	jsonOK(w, map[string]interface{}{
		"slug":           slug,
		"latest_version": latestVersion,
		"instances":      items,
		"page":           page,
		"page_size":      pageSize,
		"total":          total,
	})
}

// -----------------------------------------------------------------------------
// POST /admin/rules/distribute
// -----------------------------------------------------------------------------

// HandleDistributeRule 批量下发规范。
// POST /admin/rules/distribute
func HandleDistributeRule(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireSMHEnabled(w, r) {
		return
	}

	var req struct {
		Slug        string `json:"slug"`
		Version     string `json:"version"`
		InstanceIDs []uint `json:"instance_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("规范下发请求解析失败", "error", err)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgPluginRequestFormatErr))
		return
	}
	slog.Info("开始批量下发规范", "slug", req.Slug, "version", req.Version, "instance_count", len(req.InstanceIDs))

	if req.Slug == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillSlugRequired))
		return
	}
	if len(req.InstanceIDs) == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInstanceIdsCannotBeEmpty))
		return
	}

	// 取操作员并复用纯函数执行批量下发
	var operatorID uint
	if user, err := RequestUser(r); user != nil && err == nil {
		operatorID = user.ID
	}
	_, results, err := distributeRuleBatch(r.Context(), operatorID, req.InstanceIDs, []distributeRuleRequestItem{{Slug: req.Slug, Version: req.Version}})
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgRuleNoValidInstance))
		return
	}
	var taskID uint
	var version string
	if len(results) > 0 && results[0].Status == "submitted" {
		taskID = results[0].TaskID
		version = results[0].Version
	}
	jsonOK(w, map[string]interface{}{
		"ok":      true,
		"task_id": taskID,
		"slug":    req.Slug,
		"version": version,
	})
}

// distributeRuleRequestItem 单条规范下发请求项（批量下发使用）。
type distributeRuleRequestItem struct {
	Slug    string `json:"slug"`
	Version string `json:"version"`
}

// ruleBatchResultItem 规范批量下发结果项（镜像 skillBatchResultItem）。
type ruleBatchResultItem struct {
	Index         int    `json:"index"`
	Slug          string `json:"slug"`
	Version       string `json:"version,omitempty"`
	Status        string `json:"status"`
	TaskID        uint   `json:"task_id,omitempty"`
	InstanceCount int    `json:"instance_count,omitempty"`
	Reason        string `json:"reason,omitempty"`
	Message       string `json:"message,omitempty"`
}

func ruleBatchResultItemFailed(slug, version string, index int, reason string, richErr *hcommon.RichError, ctx context.Context) ruleBatchResultItem {
	return ruleBatchResultItem{
		Index:   index,
		Slug:    slug,
		Version: version,
		Status:  "failed",
		Reason:  reason,
		Message: hcommon.ErrorMessageWithCtx(ctx, richErr),
	}
}

// newRuleTaskBatchID 生成规范下发批次 ID（镜像 newSkillTaskBatchID）。
func newRuleTaskBatchID(now time.Time) string {
	var suffix [6]byte
	if _, err := rand.Read(suffix[:]); err == nil {
		return fmt.Sprintf("ruledist-%s-%x", now.UTC().Format("20060102150405"), suffix)
	}
	return fmt.Sprintf("ruledist-%d", now.UTC().UnixNano())
}

// distributeRuleBatch 批量下发规范的纯函数（无 HTTP 依赖）。
// 由 HandleDistributeRule（HTTP 接口）与本模块资产版本自动同步（controller/asset_version.go）共用。
// 单个规范准备/加锁失败只记入 results，不影响其他规范；返回 err 仅用于实例过滤等致命错误。
func distributeRuleBatch(ctx context.Context, operatorID uint, instanceIDs []uint, rules []distributeRuleRequestItem) (string, []ruleBatchResultItem, error) {
	if len(instanceIDs) == 0 {
		return "", nil, fmt.Errorf("instance_ids empty")
	}
	instanceIDs = hcommon.Unique(instanceIDs)
	validIDs, infoMap, skippedCount, err := loadInstancesSupportingRuleTasks(ctx, instanceIDs)
	if err != nil {
		return "", nil, err
	}
	if skippedCount > 0 {
		slog.Info("规范下发跳过非 local 实例", "skipped", skippedCount)
	}
	if len(validIDs) == 0 {
		return "", nil, fmt.Errorf("no valid instances")
	}

	now := time.Now()
	batchID := newRuleTaskBatchID(now)
	results := make([]ruleBatchResultItem, 0, len(rules))

	for i, raw := range rules {
		var rule model.EnterpriseRule
		q := model.DB(ctx).Where("slug = ?", raw.Slug)
		if raw.Version != "" && raw.Version != "latest" {
			q = q.Where("version = ?", raw.Version)
		} else {
			q = q.Order("version_major DESC, version_minor DESC, version_patch DESC")
		}
		if q.First(&rule).Error != nil {
			results = append(results, ruleBatchResultItemFailed(raw.Slug, raw.Version, i, "rule_not_found", hcommon.I18nError(i18n.MsgRuleNotExist), ctx))
			continue
		}

		lockKey := fmt.Sprintf("enterprise_rule:distribute:%s", sha256Hex([]byte(rule.Slug))[:16])
		lock, lockErr := model.AcquireLock(hcommon.WithTaskTrace(hcommon.DetachContext(ctx), "rule_distribute"), lockKey, 30*time.Minute)
		if lockErr != nil {
			results = append(results, ruleBatchResultItemFailed(rule.Slug, rule.Version, i, "locked", hcommon.I18nError(i18n.MsgSkillStoreVersionLocked), ctx))
			continue
		}

		if err := failPreviousPendingRuleDistribute(ctx, rule.Slug, []uint{rule.ID}, validIDs); err != nil {
			lock.Release()
			results = append(results, ruleBatchResultItemFailed(rule.Slug, rule.Version, i, "fail_prev_failed", hcommon.I18nRichError(err, i18n.MsgRuleDistributeFailed), ctx))
			continue
		}

		task, records, cerr := createRuleTaskAndRecords(ctx, rule, model.RuleTaskTypeDistribute, operatorID, validIDs, infoMap, batchID, now)
		if cerr != nil {
			lock.Release()
			results = append(results, ruleBatchResultItemFailed(rule.Slug, rule.Version, i, "create_task_failed", hcommon.I18nRichError(err, i18n.MsgRuleCreateRecordFail), ctx))
			continue
		}
		runRuleDistributeTask(ctx, rule, task, records, lock, infoMap)

		results = append(results, ruleBatchResultItem{
			Index:         i,
			Slug:          rule.Slug,
			Version:       rule.Version,
			Status:        "submitted",
			TaskID:        task.ID,
			InstanceCount: len(validIDs),
		})
	}
	return batchID, results, nil
}

// -----------------------------------------------------------------------------
// POST /admin/rules/uninstall

// -----------------------------------------------------------------------------
// POST /admin/rules/uninstall
// -----------------------------------------------------------------------------

// HandleUninstallRule 批量卸载规范。
// POST /admin/rules/uninstall
func HandleUninstallRule(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	var req struct {
		Slug        string `json:"slug"`
		InstanceIDs []uint `json:"instance_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("规范卸载请求解析失败", "error", err)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgPluginRequestFormatErr))
		return
	}
	slog.Info("开始批量卸载规范", "slug", req.Slug, "instance_count", len(req.InstanceIDs))

	if req.Slug == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillSlugRequired))
		return
	}
	if len(req.InstanceIDs) == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInstanceIdsCannotBeEmpty))
		return
	}

	// 去重
	req.InstanceIDs = hcommon.Unique(req.InstanceIDs)

	// 查找规范
	var rule model.EnterpriseRule
	if model.DB(r.Context()).Where("slug = ?", req.Slug).
		Order("version_major DESC, version_minor DESC, version_patch DESC").
		First(&rule).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgRuleNotExist))
		return
	}

	// 过滤非 local 实例
	validIDs, infoMap, skippedCount, err := loadInstancesSupportingRuleTasks(r.Context(), req.InstanceIDs)
	if err != nil {
		slog.Error("[UninstallRule] 查询实例信息失败", "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgPluginQueryInstanceInfo))
		return
	}
	if skippedCount > 0 {
		slog.Info("规范卸载跳过非 local 实例", "slug", req.Slug, "skipped", skippedCount)
	}
	if len(validIDs) == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgRuleNoValidInstance))
		return
	}

	// 分布式锁：与 distribute 同理，slug 用 sha256 前 16 位 hex 避免 lock key 超 64 字符。
	lockKey := fmt.Sprintf("enterprise_rule:uninstall:%s", sha256Hex([]byte(req.Slug))[:16])
	lock, lockErr := model.AcquireLock(hcommon.WithTaskTrace(hcommon.DetachContext(r.Context()), "rule_uninstall"), lockKey, 30*time.Minute)
	if lockErr != nil {
		slog.Warn("规范卸载获取锁失败", "slug", req.Slug, "error", lockErr)
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgSkillStoreSkillLocked))
		return
	}

	// 获取操作人
	var operatorID uint
	if user, err := RequestUser(r); user != nil && err == nil {
		operatorID = user.ID
	}

	// 创建卸载任务和记录
	task, records, err := createRuleTaskAndRecords(r.Context(), rule, model.RuleTaskTypeUninstall, operatorID, validIDs, infoMap, "", time.Now())
	if err != nil {
		lock.Release()
		slog.Error("创建规范卸载记录失败", "slug", req.Slug, "version", rule.Version, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgRuleCreateRecordFail))
		return
	}

	// 异步执行卸载
	runRuleDistributeTask(r.Context(), rule, task, records, lock, infoMap)

	jsonOK(w, map[string]interface{}{
		"ok":      true,
		"task_id": task.ID,
		"slug":    req.Slug,
		"version": rule.Version,
	})
}

// -----------------------------------------------------------------------------
// helpers 内部私有
// -----------------------------------------------------------------------------

// -----------------------------------------------------------------------------
// 分发/卸载辅助函数
// -----------------------------------------------------------------------------

// ruleInstanceInfo 本地实例信息（对称 skillInstanceInfo）。
type ruleInstanceInfo struct {
	ID         uint   `gorm:"column:id"`
	InstanceID string `gorm:"column:instance_id"`
}

// loadInstancesSupportingRuleTasks 过滤出支持规范操作的本地实例。
func loadInstancesSupportingRuleTasks(ctx context.Context, instanceIDs []uint) ([]uint, map[uint]ruleInstanceInfo, int, error) {
	var infos []ruleInstanceInfo
	if err := model.DB(ctx).Model(&model.Instance{}).
		Select("id, instance_id").
		Where("id IN ? AND source = ?", instanceIDs, model.InstanceSourceLocal).
		Scan(&infos).Error; err != nil {
		return nil, nil, 0, err
	}
	infoMap := make(map[uint]ruleInstanceInfo, len(infos))
	validIDs := make([]uint, 0, len(infos))
	for _, info := range infos {
		infoMap[info.ID] = info
		validIDs = append(validIDs, info.ID)
	}
	skipped := len(instanceIDs) - len(infos)
	return validIDs, infoMap, skipped, nil
}

// failPreviousPendingRuleDistribute 在发起新一次下发前，检查同一 slug 上一次 distribute
// 任务是否仍有 pending 记录（即上次下发给某些本地 agent 还没被 reporter 拉走处理完）。
// 若存在，则只把上一次任务中本次请求涉及的相同 instance（instanceIDs）的 pending record
// 置为 failed（失败原因：已下发新的版本），同时把这些记录的数量累加到上一次任务的 failed 计数。
//
// 只处理与本次请求相同 instance_id 的 pending 记录——一个 task 可能给多个本地 agent 下发，
// 其它实例可能早已安装成功，不能因个别实例还 pending 就把整个 task 判失败，也不能把本次请求
// 没覆盖到的实例的 pending 误判。
//
// 调用方须已持有该 slug 的 distribute 分布式锁（HandleDistributeRule 里已加），
// 因此这里不需要额外的并发保护。
//
// 只处理 distribute 类型；uninstall 的 pending 不在本函数职责内。
func failPreviousPendingRuleDistribute(ctx context.Context, slug string, ruleIDs []uint, instanceIDs []uint) error {
	if len(instanceIDs) == 0 {
		return nil
	}
	var prevTask model.RuleDistributionTask
	// 取该 slug 最新的 distribute 任务（type=distribute），不论其 task 状态——
	// 因为 runRuleDistributeTask 会同步把 task 标 completed，但 record 可能仍 pending（等 reporter ack）。
	if err := model.DB(ctx).Where("slug = ? AND type = ?", slug, model.RuleTaskTypeDistribute).
		Order("id DESC").Limit(1).First(&prevTask).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	// 该任务中，本次请求涉及的相同实例是否还有 pending 记录
	var pendingCnt int64
	model.DB(ctx).Model(&model.RuleDistributionRecord{}).
		Where("task_id = ? AND rule_id IN ? AND type = ? AND status = ? AND instance_id IN ?",
			prevTask.ID, ruleIDs, model.RuleTaskTypeDistribute, model.RuleRecordStatusPending, instanceIDs).
		Count(&pendingCnt)
	if pendingCnt == 0 {
		return nil
	}

	// 只把本次请求涉及的相同实例的 pending record 置为 failed。
	now := time.Now()
	if err := model.DB(ctx).Model(&model.RuleDistributionRecord{}).
		Where("task_id = ? AND rule_id IN ? AND type = ? AND status = ? AND instance_id IN ?",
			prevTask.ID, ruleIDs, model.RuleTaskTypeDistribute, model.RuleRecordStatusPending, instanceIDs).
		Updates(map[string]interface{}{
			"status":     model.RuleRecordStatusFailed,
			"error":      i18n.T(ctx, i18n.MsgSkillNewVersionDistributed),
			"updated_at": now,
		}).Error; err != nil {
		return err
	}
	// 把判失败的记录数量累加到上一次任务的 failed 计数。
	if err := model.DB(ctx).Model(&model.RuleDistributionTask{}).
		Where("id = ?", prevTask.ID).
		UpdateColumn("failed", gorm.Expr("failed + ?", pendingCnt)).Error; err != nil {
		return err
	}
	slog.Info("规范下发：上一次 pending 记录已判失败", "slug", slug, "prev_task_id", prevTask.ID, "pending", pendingCnt)
	return nil
}

// createRuleTaskAndRecords 创建分发/卸载任务和记录。
func createRuleTaskAndRecords(ctx context.Context, rule model.EnterpriseRule, action string, operatorID uint, instanceIDs []uint, infoMap map[uint]ruleInstanceInfo, batchID string, createdAt time.Time) (model.RuleDistributionTask, []model.RuleDistributionRecord, error) {
	var task model.RuleDistributionTask
	var records []model.RuleDistributionRecord
	err := model.DB(ctx).Transaction(func(tx *gorm.DB) error {
		task = model.RuleDistributionTask{
			CreatedAt:  createdAt,
			UpdatedAt:  createdAt,
			RuleID:     rule.ID,
			Slug:       rule.Slug,
			RuleType:   rule.Type,
			Version:    rule.Version,
			BatchID:    batchID,
			OperatorID: operatorID,
			Total:      len(instanceIDs),
			Status:     "running",
			Type:       action,
		}
		if err := tx.Create(&task).Error; err != nil {
			return err
		}

		records = make([]model.RuleDistributionRecord, 0, len(instanceIDs))
		for _, instID := range instanceIDs {
			info := infoMap[instID]
			records = append(records, model.RuleDistributionRecord{
				TaskID:      task.ID,
				RuleID:      rule.ID,
				InstanceID:  instID,
				InstanceCID: info.InstanceID,
				Version:     rule.Version,
				Status:      model.RuleRecordStatusPending,
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

// runRuleDistributeTask 异步执行规范下发/卸载。
// 本地实例的 record 保留 pending 给 reporter sync 拉取，不需要像 skill 那样后端直接下发。
func runRuleDistributeTask(ctx context.Context, rule model.EnterpriseRule, task model.RuleDistributionTask, records []model.RuleDistributionRecord, lock *model.DistLock, infoMap map[uint]ruleInstanceInfo) {
	defer lock.Release()

	// 规范的分发/卸载由 reporter 在 sync 时拉取 pending 记录并执行，
	// 服务端不需要像 skill 那样直接调用 CVM API 下发。
	// 这里只需要把 task 标记为 completed 即可。
	now := time.Now()
	if err := model.DB(ctx).Model(&task).Updates(map[string]interface{}{
		"status":     "completed",
		"updated_at": now,
	}).Error; err != nil {
		slog.Error("规范分发任务标记完成失败", "task_id", task.ID, "error", err)
	}
}

// sha256Hex 返回小写 16 进制的 SHA-256（32 字节 → 64 字符）。
func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
