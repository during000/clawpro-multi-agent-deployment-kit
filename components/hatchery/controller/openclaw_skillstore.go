package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm"
)

// HandleTenantSkills 技能广场 — 查询用户可见的技能列表（分页，每个 slug 只返回最新版本）
func HandleSkillStore(w http.ResponseWriter, r *http.Request) {
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	page, pageSize := parsePagination(r)

	// 基础查询：每个 slug 取最新版本（仅显示已上架的技能）
	db := model.DB(r.Context()).Model(&model.Skill{}).
		Where("id IN (?)", model.LatestVersionSkillIDs(r.Context())).
		Where("status = ?", model.SkillStatusPublished)

	// ── 可见范围过滤（用户端核心差异） ──
	userGroupIDs, _ := model.GetUserGroupIDs(r.Context(), user.ID)
	if len(userGroupIDs) > 0 {
		visSubQ := model.DB(r.Context()).Model(&model.SkillVisibilityGroup{}).
			Select("skill_id").Where("group_id IN ?", userGroupIDs)
		db = db.Where("visibility_type = 'all' OR id IN (?)", visSubQ)
	} else {
		db = db.Where("visibility_type = 'all'")
	}

	// ── 筛选条件 ──
	if keyword := r.URL.Query().Get("keyword"); keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("name LIKE ? OR description LIKE ?", like, like)
	}
	if catIDs := r.URL.Query().Get("category_ids"); catIDs != "" {
		var intIDs []int
		for _, s := range strings.Split(catIDs, ",") {
			if id, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
				intIDs = append(intIDs, id)
			}
		}
		if len(intIDs) > 0 {
			subQuery := model.DB(r.Context()).Model(&model.SkillCategoryMapping{}).
				Select("skill_id").
				Where("category_id IN ?", intIDs)
			db = db.Where("id IN (?)", subQuery)
		}
	}

	// ── 排序 ──
	sortMode := r.URL.Query().Get("sort")
	if sortMode == "downloads" {
		db = db.Order("distribute_count DESC, id DESC")
	} else {
		db = db.Order("id DESC")
	}

	var total int64
	db.Count(&total)

	var skills []model.Skill
	db.Offset((page - 1) * pageSize).Limit(pageSize).Find(&skills)

	// ── 批量加载 last_task ──
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
	if len(skills) > 0 {
		skillIDs := make([]uint, len(skills))
		for i, s := range skills {
			skillIDs[i] = s.ID
		}

		// 用户端 last_task 限定为当前用户实例相关的 task
		userInstSubQ := model.DB(r.Context()).Model(&model.Instance{}).Select("id").Where("user_id = ?", user.ID)
		userTaskSubQ := model.DB(r.Context()).Model(&model.SkillDistributionRecord{}).
			Select("DISTINCT task_id").Where("instance_id IN (?)", userInstSubQ)

		var tasks []model.SkillDistributionTask
		subQuery := model.DB(r.Context()).Model(&model.SkillDistributionTask{}).
			Select("MAX(id)").
			Where("skill_id IN ? AND id IN (?)", skillIDs, userTaskSubQ).
			Group("skill_id")
		model.DB(r.Context()).Where("id IN (?)", subQuery).Find(&tasks)

		if len(tasks) > 0 {
			taskIDs := make([]uint, len(tasks))
			for i, t := range tasks {
				taskIDs[i] = t.ID
			}
			// 聚合 record 计数（仅当前用户实例）
			type taskStatusCount struct {
				TaskID uint
				Status string
				Count  int
			}
			var counts []taskStatusCount
			if err := model.DB(r.Context()).Model(&model.SkillDistributionRecord{}).
				Select("task_id, status, COUNT(*) as count").
				Where("task_id IN ? AND instance_id IN (?)", taskIDs, userInstSubQ).
				Group("task_id, status").
				Scan(&counts).Error; err != nil {
				slog.Error("[SkillStore] 查询下发记录聚合失败", "error", err)
			}
			type counters struct{ Success, Failed int }
			countMap := make(map[uint]*counters)
			for _, c := range counts {
				if countMap[c.TaskID] == nil {
					countMap[c.TaskID] = &counters{}
				}
				switch c.Status {
				case "success":
					countMap[c.TaskID].Success = c.Count
				case "failed":
					countMap[c.TaskID].Failed = c.Count
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
				if c := countMap[t.ID]; c != nil {
					lt.Success = c.Success
					lt.Failed = c.Failed
				}
				taskMap[t.SkillID] = lt
			}
		}
	}

	// ── 批量加载分类 ──
	categoryMap := make(map[uint][]map[string]interface{})
	if len(skills) > 0 {
		skillIDs := make([]uint, len(skills))
		for i, s := range skills {
			skillIDs[i] = s.ID
		}
		var allMappings []model.SkillCategoryMapping
		model.DB(r.Context()).Where("skill_id IN ?", skillIDs).Find(&allMappings)

		catIDSet := make(map[uint]struct{})
		for _, m := range allMappings {
			catIDSet[m.CategoryID] = struct{}{}
		}
		catIDs := make([]uint, 0, len(catIDSet))
		for id := range catIDSet {
			catIDs = append(catIDs, id)
		}
		catMap := make(map[uint]model.SkillCategory)
		if len(catIDs) > 0 {
			var cats []model.SkillCategory
			model.DB(r.Context()).Where("id IN ?", catIDs).Find(&cats)
			for _, c := range cats {
				catMap[c.ID] = c
			}
		}
		for _, m := range allMappings {
			if cat, ok := catMap[m.CategoryID]; ok {
				categoryMap[m.SkillID] = append(categoryMap[m.SkillID], map[string]interface{}{"id": cat.ID, "name": cat.Name})
			}
		}
	}

	// ── 组装响应 ──
	type skillResp struct {
		model.Skill
		Categories     []map[string]interface{} `json:"categories"`
		LastTask       *lastTask                `json:"last_task"`
		SecurityStatus string                   `json:"security_status"`
	}

	// 批量查询安全检测状态
	var scanSkillIDs []uint
	for _, s := range skills {
		scanSkillIDs = append(scanSkillIDs, s.ID)
	}
	scanMap, scanErr := model.GetSkillsSecurityStatus(r.Context(), scanSkillIDs)
	if scanErr != nil {
		slog.Error("[SkillStore] 批量查询安全检测状态失败", "error", scanErr)
		scanMap = make(map[uint]*model.SkillSecurityScan)
	}

	// 与项目其他列表接口对齐：空切片须序列化为 []，不能为 null。
	// 否则前端/集成测试断言 isinstance(skills, list) 会失败。
	result := make([]skillResp, 0, len(skills))
	for _, s := range skills {
		sr := skillResp{Skill: s, LastTask: taskMap[s.ID]}
		sr.Categories = categoryMap[s.ID]
		if sr.Categories == nil {
			sr.Categories = []map[string]interface{}{}
		}
		sr.SecurityStatus = mapToScanStatusString(scanMap[s.ID])
		result = append(result, sr)
	}

	jsonOK(w, map[string]interface{}{
		"skills":    result,
		"page":      page,
		"page_size": pageSize,
		"total":     total,
	})
}

// HandleSkillStoreDetail 技能广场 — 查询技能详情
func HandleSkillStoreDetail(w http.ResponseWriter, r *http.Request) {
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	slug := r.URL.Query().Get("slug")
	if slug == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillStoreSlugRequired))
		return
	}

	version := r.URL.Query().Get("version")
	var skill model.Skill
	if version == "" || version == "latest" {
		if model.DB(r.Context()).Where("slug = ? AND status = ?", slug, model.SkillStatusPublished).Order("version_major DESC, version_minor DESC, version_patch DESC").First(&skill).Error != nil {
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgSkillNotExist))
			return
		}
	} else {
		if model.DB(r.Context()).Where("slug = ? AND version = ? AND status = ?", slug, version, model.SkillStatusPublished).First(&skill).Error != nil {
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgPluginVersionNotFoundDetail, slug, version))
			return
		}
	}

	// 可见性校验
	visible, err := model.IsSkillVisibleToUser(r.Context(), &skill, user.ID)
	if err != nil {
		slog.Error("[SkillStoreDetail] 可见性检查失败", "slug", slug, "user_id", user.ID, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgSkillStoreVisibilityCheckFail))
		return
	}
	if !visible {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgSkillNotExist))
		return
	}

	// 查询所有版本列表
	var allVersions []model.Skill
	model.DB(r.Context()).Where("slug = ?", slug).Order("version_major DESC, version_minor DESC, version_patch DESC").Find(&allVersions)

	// 批量查询所有版本的安全检测状态
	var versionSkillIDs []uint
	for _, v := range allVersions {
		versionSkillIDs = append(versionSkillIDs, v.ID)
	}
	versionScanMap, scanMapErr := model.GetSkillsSecurityStatus(r.Context(), versionSkillIDs)
	if scanMapErr != nil {
		slog.Error("[SkillStoreDetail] 批量查询版本安全检测状态失败", "slug", slug, "error", scanMapErr)
		versionScanMap = make(map[uint]*model.SkillSecurityScan)
	}

	type versionInfo struct {
		Version      string          `json:"version"`
		CreatedAt    time.Time       `json:"created_at"`
		SecurityScan *scanStatusResp `json:"security_scan"`
	}
	var versions []versionInfo
	for _, v := range allVersions {
		versions = append(versions, versionInfo{
			Version:      v.Version,
			CreatedAt:    v.CreatedAt,
			SecurityScan: buildScanStatusResp(versionScanMap[v.ID]),
		})
	}

	// 查询分类
	var categories []map[string]interface{}
	var mappings []model.SkillCategoryMapping
	model.DB(r.Context()).Where("skill_id = ?", skill.ID).Find(&mappings)
	if len(mappings) > 0 {
		catIDs := make([]uint, 0, len(mappings))
		for _, m := range mappings {
			catIDs = append(catIDs, m.CategoryID)
		}
		var cats []model.SkillCategory
		model.DB(r.Context()).Where("id IN ?", catIDs).Find(&cats)
		for _, c := range cats {
			categories = append(categories, map[string]interface{}{"id": c.ID, "name": c.Name})
		}
	}
	if categories == nil {
		categories = []map[string]interface{}{}
	}

	// 解析文件列表
	var fileList []string
	if skill.FileList != "" {
		json.Unmarshal([]byte(skill.FileList), &fileList)
	}
	if fileList == nil {
		fileList = []string{}
	}

	// 返回 SMH 只读 Token（供前端预览技能文件）
	var smhInfo map[string]interface{}
	smhConfig := model.GetSMHConfig(r.Context())
	if smhConfig.IsConfigured() && smhConfig.SkillhubSpace != "" {
		if token, err := GetSkillhubSpaceReadToken(r.Context()); err == nil {
			smhInfo = map[string]interface{}{
				"access_token": token,
				"space_id":     smhConfig.SkillhubSpace,
				"library_id":   smhConfig.LibraryId,
				"endpoint":     smhConfig.Endpoint,
			}
		}
	}

	// 查询安全检测状态
	latestScan, scanErr := model.GetLatestSkillSecurityScan(r.Context(), skill.ID)
	if scanErr != nil {
		slog.Error("[SkillStore] 查询技能安全检测状态失败", "skill_id", skill.ID, "error", scanErr)
	}
	securityScan := buildScanStatusResp(latestScan)

	jsonOK(w, map[string]interface{}{
		"skill": map[string]interface{}{
			"id":               skill.ID,
			"slug":             skill.Slug,
			"name":             skill.Name,
			"version":          skill.Version,
			"description":      skill.Description,
			"changelog":        skill.Changelog,
			"categories":       categories,
			"file_list":        fileList,
			"file_size":        skill.FileSize,
			"cos_zip_key":      skill.COSZipKey,
			"cos_dir_key":      skill.COSDirKey,
			"distribute_count": skill.DistributeCount,
			"created_at":       skill.CreatedAt,
			"security_scan":    securityScan,
		},
		"versions": versions,
		"smh":      smhInfo,
	})
}

// HandleSkillStoreCategories 技能广场 — 查询分类列表
func HandleSkillStoreCategories(w http.ResponseWriter, r *http.Request) {
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	jsonAPI(w)

	var categories []model.SkillCategory
	model.DB(r.Context()).Order("id ASC").Find(&categories)

	type catResp struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	}
	var result []catResp
	for _, c := range categories {
		result = append(result, catResp{ID: c.ID, Name: c.Name})
	}

	jsonOK(w, map[string]interface{}{
		"categories": result,
	})
}

// HandleSkillStoreInstances 技能广场 — 查询当前用户实例的安装状态
func HandleSkillStoreInstances(w http.ResponseWriter, r *http.Request) {
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	slug := r.URL.Query().Get("slug")
	if slug == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillStoreSlugRequired))
		return
	}

	// 可见性校验（取最新版本）
	var latestSkill model.Skill
	if model.DB(r.Context()).Where("slug = ?", slug).Order("version_major DESC, version_minor DESC, version_patch DESC").First(&latestSkill).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgSkillNotExist))
		return
	}
	visible, err := model.IsSkillVisibleToUser(r.Context(), &latestSkill, user.ID)
	if err != nil || !visible {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgSkillNotExist))
		return
	}

	// 查找该 slug 的所有 skill ID
	var skillIDs []uint
	model.DB(r.Context()).Model(&model.Skill{}).Where("slug = ?", slug).Pluck("id", &skillIDs)

	// 查询最新版本号
	latestVersion := latestSkill.Version

	page, pageSize := parsePagination(r, 500)

	type instResp struct {
		InstanceID            uint       `json:"instance_id"              gorm:"column:instance_id"`
		CVMInstanceID         string     `json:"cvm_instance_id"          gorm:"column:cvm_instance_id"`
		InstanceName          string     `json:"instance_name"            gorm:"column:instance_name"`
		InstanceType          string     `json:"instance_type"            gorm:"column:instance_type"`
		UserID                uint       `json:"-"                        gorm:"column:user_id"`
		LastCVMState          string     `json:"-"                        gorm:"column:last_cvm_state"`
		LastStableState       string     `json:"-"                        gorm:"column:last_stable_state"`
		CurrentOperation      string     `json:"-"                        gorm:"column:current_operation"`
		CurrentOperationState string     `json:"-"                        gorm:"column:current_operation_state"`
		AgentReady            int        `json:"-"                        gorm:"column:agent_ready"`
		CLSAgentStatus        int        `json:"-"                        gorm:"column:cls_agent_status"`
		CLSAgentStatusAt      *time.Time `json:"-"                        gorm:"column:cls_agent_status_at"`
		Status                string     `json:"status"                   gorm:"column:install_status"`
		Version               string     `json:"version"                  gorm:"column:version"`
		LatestVersion         string     `json:"latest_version"           gorm:"column:latest_version"`
	}

	// 构造基础查询 + 限定当前用户
	baseQuery := model.BuildSkillInstanceQuery(r.Context(), skillIDs, latestVersion, slug)
	baseQuery = baseQuery.Where("instances.user_id = ?", user.ID)

	// 搜索
	if search := r.URL.Query().Get("search"); search != "" {
		baseQuery = baseQuery.Where("instances.name LIKE ? OR instances.instance_id LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// 安装状态筛选
	if statusFilter := r.URL.Query().Get("status"); statusFilter != "" {
		statuses := strings.Split(statusFilter, ",")
		baseQuery = baseQuery.Scopes(model.FilterSkillInstallStatuses(latestVersion, statuses))
	}

	// 全量查询（后续需计算 CVM 语义状态过滤 running）
	var allResults []instResp
	if err := baseQuery.Order("instances.created_at DESC").Scan(&allResults).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgSkillStoreQueryInstancesFail, err))
		return
	}
	if allResults == nil {
		allResults = []instResp{}
	}

	// 批量查询 CVM 实时状态
	var cvmIDs []string
	for _, inst := range allResults {
		if inst.CVMInstanceID != "" {
			cvmIDs = append(cvmIDs, inst.CVMInstanceID)
		}
	}
	cvmInfoMap := batchFetchCVMInfoMap(r.Context(), cvmIDs)

	// ── 批量预查：消除循环内 N+1 ──
	siteConfig := model.GetSiteConfig(r.Context())
	allAgentTypes := model.GetAllAgentTypesMap(r.Context())
	preInstIDs := make([]uint, 0, len(allResults))
	for _, inst := range allResults {
		preInstIDs = append(preInstIDs, inst.InstanceID)
	}
	installingSkillMap := batchHasInstallingSkillInstallations(r.Context(), preInstIDs)
	batch := &InstanceStatusBatchLookup{SiteConfig: siteConfig, InstallingSkillMap: installingSkillMap}

	// 计算语义状态，只保留 running 且支持 Skill 的实例
	type instFinalResp struct {
		InstanceID          uint   `json:"instance_id"`
		CVMInstanceID       string `json:"cvm_instance_id"`
		InstanceName        string `json:"instance_name"`
		InstanceType        string `json:"instance_type"`
		Status              string `json:"status"`
		Version             string `json:"version"`
		LatestVersion       string `json:"latest_version"`
		InstanceStatus      string `json:"instance_status"`
		InstanceStatusLabel string `json:"instance_status_label"`
	}
	var runningResults []instFinalResp
	for _, inst := range allResults {
		// 过滤不支持技能的实例类型
		if !model.AgentTypeSupportsSkillByMap(inst.InstanceType, allAgentTypes) {
			continue
		}
		tmpInst := model.Instance{
			LastCVMState:          inst.LastCVMState,
			LastStableState:       inst.LastStableState,
			CurrentOperation:      inst.CurrentOperation,
			CurrentOperationState: inst.CurrentOperationState,
			AgentReady:            inst.AgentReady,
			CLSAgentStatus:        inst.CLSAgentStatus,
			CLSAgentStatusAt:      inst.CLSAgentStatusAt,
			InstanceId:            inst.CVMInstanceID,
		}
		tmpInst.ID = inst.InstanceID
		cvmInfo := cvmInfoMap[inst.CVMInstanceID]
		statusResp := ResolveInstanceStatus(r.Context(), &tmpInst, cvmInfo, batch)
		if statusResp.Status != model.StatusRunning {
			continue
		}
		runningResults = append(runningResults, instFinalResp{
			InstanceID:          inst.InstanceID,
			CVMInstanceID:       inst.CVMInstanceID,
			InstanceName:        inst.InstanceName,
			InstanceType:        inst.InstanceType,
			Status:              inst.Status,
			Version:             inst.Version,
			LatestVersion:       inst.LatestVersion,
			InstanceStatus:      statusResp.Status,
			InstanceStatusLabel: statusResp.Label,
		})
	}

	// 内存分页
	total := int64(len(runningResults))
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > len(runningResults) {
		start = len(runningResults)
	}
	if end > len(runningResults) {
		end = len(runningResults)
	}
	pageResults := runningResults[start:end]

	jsonOK(w, map[string]interface{}{
		"instances": pageResults,
		"page":      page,
		"page_size": pageSize,
		"total":     total,
	})
}

// HandleSkillStoreTasks 技能广场 — 查询下发记录（仅当前用户实例相关）
func HandleSkillStoreTasks(w http.ResponseWriter, r *http.Request) {
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	slug := r.URL.Query().Get("slug")
	if slug == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillStoreSlugRequired))
		return
	}

	// 可见性校验
	var latestSkill model.Skill
	if model.DB(r.Context()).Where("slug = ?", slug).Order("version_major DESC, version_minor DESC, version_patch DESC").First(&latestSkill).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgSkillNotExist))
		return
	}
	visible, err := model.IsSkillVisibleToUser(r.Context(), &latestSkill, user.ID)
	if err != nil || !visible {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgSkillNotExist))
		return
	}

	// 查找该 slug 的所有 skill ID
	var skillIDs []uint
	model.DB(r.Context()).Model(&model.Skill{}).Where("slug = ?", slug).Pluck("id", &skillIDs)

	page, pageSize := parsePagination(r)

	// 筛选：只查涉及当前用户实例的 task
	userInstSubQ := model.DB(r.Context()).Model(&model.Instance{}).Select("id").Where("user_id = ?", user.ID)
	taskSubQ := model.DB(r.Context()).Model(&model.SkillDistributionRecord{}).
		Select("DISTINCT task_id").Where("instance_id IN (?) AND skill_id IN ?", userInstSubQ, skillIDs)

	var total int64
	taskListQuery := model.DB(r.Context()).Model(&model.SkillDistributionTask{}).Where("id IN (?)", taskSubQ)
	// 支持按 type 筛选（all / distribute / uninstall）
	if typeFilter := r.URL.Query().Get("type"); typeFilter != "" && typeFilter != "all" {
		taskListQuery = taskListQuery.Where("type = ?", typeFilter)
	}
	taskListQuery.Count(&total)

	var tasks []model.SkillDistributionTask
	taskListQuery.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&tasks)

	type recordResp struct {
		InstanceID    uint   `json:"instance_id"`
		CVMInstanceID string `json:"cvm_instance_id"`
		InstanceName  string `json:"instance_name"`
		Status        string `json:"status"`
		Error         string `json:"error"`
	}
	type taskResp struct {
		ID        uint         `json:"id"`
		CreatedAt interface{}  `json:"created_at"`
		Operator  string       `json:"operator"`
		Version   string       `json:"version"`
		Total     int          `json:"total"`
		Success   int          `json:"success"`
		Failed    int          `json:"failed"`
		Pending   int          `json:"pending"`
		Status    string       `json:"status"`
		Type      string       `json:"type"`
		Records   []recordResp `json:"records"`
	}

	var result []taskResp
	if len(tasks) > 0 {
		taskIDs := make([]uint, len(tasks))
		operatorIDs := make(map[uint]struct{})
		for i, t := range tasks {
			taskIDs[i] = t.ID
			if t.OperatorID > 0 {
				operatorIDs[t.OperatorID] = struct{}{}
			}
		}

		// 查询操作人
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

		// 查询 records（仅当前用户实例）
		var allRecords []model.SkillDistributionRecord
		model.DB(r.Context()).Where("task_id IN ? AND instance_id IN (?)", taskIDs, userInstSubQ).Find(&allRecords)

		// 批量查实例名称
		instIDSet := make(map[uint]struct{})
		for _, rec := range allRecords {
			instIDSet[rec.InstanceID] = struct{}{}
		}
		instIDs := make([]uint, 0, len(instIDSet))
		for id := range instIDSet {
			instIDs = append(instIDs, id)
		}
		type instDetail struct {
			ID   uint
			Name string
		}
		instMap := make(map[uint]instDetail)
		if len(instIDs) > 0 {
			var insts []instDetail
			model.DB(r.Context()).Model(&model.Instance{}).Select("id, name").Where("id IN ?", instIDs).Scan(&insts)
			for _, inst := range insts {
				instMap[inst.ID] = inst
			}
		}

		// 按 task_id 分组并聚合
		recordsByTask := make(map[uint][]model.SkillDistributionRecord)
		for _, rec := range allRecords {
			recordsByTask[rec.TaskID] = append(recordsByTask[rec.TaskID], rec)
		}

		for _, t := range tasks {
			records := recordsByTask[t.ID]
			tr := taskResp{
				ID:        t.ID,
				CreatedAt: t.CreatedAt,
				Version:   t.Version,
				Status:    t.Status,
				Type:      t.Type,
				Operator:  userMap[t.OperatorID],
			}
			// 统计当前用户实例的计数
			for _, rec := range records {
				switch rec.Status {
				case "success":
					tr.Success++
				case "failed":
					tr.Failed++
				case "pending":
					tr.Pending++
				}
			}
			tr.Total = len(records)
			for _, rec := range records {
				rr := recordResp{
					InstanceID:    rec.InstanceID,
					CVMInstanceID: rec.InstanceCID,
					Status:        rec.Status,
					Error:         rec.Error,
				}
				if inst, ok := instMap[rec.InstanceID]; ok {
					rr.InstanceName = inst.Name
				}
				tr.Records = append(tr.Records, rr)
			}
			if tr.Records == nil {
				tr.Records = []recordResp{}
			}
			result = append(result, tr)
		}
	}

	jsonOK(w, map[string]interface{}{
		"tasks":     result,
		"page":      page,
		"page_size": pageSize,
		"total":     total,
	})
}

// HandleSkillStoreDistribute 技能广场 — 下发技能到用户自己的实例
func HandleSkillStoreDistribute(w http.ResponseWriter, r *http.Request) {
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	var req struct {
		Slug        string `json:"slug"`
		Version     string `json:"version"`
		InstanceIDs []uint `json:"instance_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgPluginRequestFormatErr, err))
		return
	}
	if req.Slug == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "slug"))
		return
	}
	if len(req.InstanceIDs) == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInstanceIdsCannotBeEmpty))
		return
	}

	// 去重
	seen := make(map[uint]bool, len(req.InstanceIDs))
	var uniqueIDs []uint
	for _, id := range req.InstanceIDs {
		if !seen[id] {
			seen[id] = true
			uniqueIDs = append(uniqueIDs, id)
		}
	}
	req.InstanceIDs = uniqueIDs

	// 查找技能（仅允许下发已上架的技能）
	var skill model.Skill
	if req.Version == "" || req.Version == "latest" {
		if model.DB(r.Context()).Where("slug = ? AND status = ?", req.Slug, model.SkillStatusPublished).Order("version_major DESC, version_minor DESC, version_patch DESC").First(&skill).Error != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillNotExist))
			return
		}
	} else {
		if model.DB(r.Context()).Where("slug = ? AND version = ? AND status = ?", req.Slug, req.Version, model.SkillStatusPublished).First(&skill).Error != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgPluginVersionNotFoundDetail, req.Slug, req.Version))
			return
		}
	}

	// 可见性校验
	visible, visErr := model.IsSkillVisibleToUser(r.Context(), &skill, user.ID)
	if visErr != nil || !visible {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgSkillNotExist))
		return
	}

	// ── 安全校验：所有 instance_ids 必须属于当前用户 ──
	var ownedCount int64
	model.DB(r.Context()).Model(&model.Instance{}).
		Where("id IN ? AND user_id = ?", req.InstanceIDs, user.ID).
		Count(&ownedCount)
	if int(ownedCount) != len(req.InstanceIDs) {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgSkillStoreNotOwnInstance))
		return
	}

	// 获取分布式锁
	lockKey := fmt.Sprintf("skill_dist:%d", skill.ID)
	lock, lockErr := model.AcquireLock(hcommon.WithTaskTrace(hcommon.DetachContext(r.Context()), "skillstore_distribute"), lockKey, 30*time.Minute)
	if lockErr != nil {
		slog.Warn("[SkillStoreDistribute] 获取锁失败", "slug", req.Slug, "version", skill.Version, "error", lockErr)
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgSkillStoreVersionLocked))
		return
	}

	// 批量查询实例信息（额外拉 source，区分 cvm/local）
	type instInfo struct {
		ID          uint
		InstanceId  string
		RuntimeUser string
		AgentType   string
		Source      string
	}
	var instInfos []instInfo
	if err := model.DB(r.Context()).Model(&model.Instance{}).
		Select("id, instance_id, runtime_user, agent_type, source").
		Where("id IN ?", req.InstanceIDs).
		Scan(&instInfos).Error; err != nil {
		lock.Release()
		slog.Error("[SkillStoreDistribute] 查询实例信息失败", "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgSkillStoreQueryInstanceFail))
		return
	}

	// 过滤不支持技能的实例类型
	allAgentTypes := model.GetAllAgentTypesMap(r.Context())
	cidMap := make(map[uint]string, len(instInfos))
	ruMap := make(map[uint]string, len(instInfos))
	atMap := make(map[uint]string, len(instInfos))
	srcMap := make(map[uint]string, len(instInfos))
	var validIDs []uint
	var localIDs []uint // 本地实例单独记录，不走 TAT
	for _, info := range instInfos {
		if !model.AgentTypeSupportsSkillByMap(info.AgentType, allAgentTypes) {
			continue
		}
		cidMap[info.ID] = info.InstanceId
		ruMap[info.ID] = info.RuntimeUser
		atMap[info.ID] = info.AgentType
		srcMap[info.ID] = info.Source
		validIDs = append(validIDs, info.ID)
		if info.Source == model.InstanceSourceLocal {
			localIDs = append(localIDs, info.ID)
		}
	}
	if len(validIDs) == 0 {
		lock.Release()
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillStoreNoValidInstall))
		return
	}
	req.InstanceIDs = validIDs

	// 创建下发任务
	task := model.SkillDistributionTask{
		SkillID:    skill.ID,
		Version:    skill.Version,
		OperatorID: user.ID,
		Total:      len(req.InstanceIDs),
		Status:     "running",
		Type:       "distribute",
	}
	model.DB(r.Context()).Create(&task)

	// 批量创建下发记录（CVM + local 共用同一任务、同一 records 表）
	records := make([]model.SkillDistributionRecord, 0, len(req.InstanceIDs))
	for _, instID := range req.InstanceIDs {
		records = append(records, model.SkillDistributionRecord{
			TaskID:      task.ID,
			SkillID:     skill.ID,
			InstanceID:  instID,
			InstanceCID: cidMap[instID],
			Version:     skill.Version,
			Status:      "pending",
			Type:        "distribute",
		})
	}
	if err := model.DB(r.Context()).Create(&records).Error; err != nil {
		lock.Release()
		slog.Error("[SkillStoreDistribute] 创建下发记录失败", "task_id", task.ID, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgSkillStoreCreateRecordFail, err))
		return
	}

	// 拆分：CVM record 走 executor（TAT）；local record 仅落表等 reporter sync
	cvmRecords := make([]model.SkillDistributionRecord, 0, len(records))
	for _, rec := range records {
		if srcMap[rec.InstanceID] == model.InstanceSourceLocal {
			continue
		}
		cvmRecords = append(cvmRecords, rec)
	}
	_ = localIDs // 仅作调试/读性示例，local records 已随 records 创建

	// 生成下载 URL
	cosZipKey := fmt.Sprintf("%s/%s-%s.zip", req.Slug, req.Slug, skill.Version)
	downloadURL, urlErr := buildSMHDownloadURL(r.Context(), cosZipKey, true)

	// 异步执行下发（仅 CVM records）
	// local records 保持 status='pending'，由 reporter sync 取走后走 ack 接口回写。
	// 可能存在纯 local 批次——跳过异步执行、直接释锁。
	if len(cvmRecords) == 0 {
		lock.Release()
		jsonOK(w, map[string]interface{}{
			"ok":      true,
			"task_id": task.ID,
			"version": skill.Version,
		})
		return
	}

	executeSkillTaskAsync(SkillTaskConfig{
		Ctx:      hcommon.DetachContext(r.Context()),
		Task:     task,
		Records:  cvmRecords,
		Lock:     lock,
		Slug:     req.Slug,
		SkillIDs: []uint{skill.ID},
		OnFailed: func(ctx context.Context, record model.SkillDistributionRecord) string {
			return model.ResolveDistributeFailedStatus(ctx, record.InstanceID, []uint{skill.ID})
		},
		OnComplete: func(ctx context.Context, successCount, _ int) {
			if successCount > 0 {
				model.DB(ctx).Model(&model.Skill{}).Where("id = ?", skill.ID).
					UpdateColumn("distribute_count", gorm.Expr("distribute_count + ?", successCount))
			}
		},
	}, func(ctx context.Context, record model.SkillDistributionRecord) error {
		if urlErr != nil {
			return hcommon.I18nError(i18n.MsgSkillDownloadURLGenFail, urlErr.Error())
		}

		agentType := atMap[record.InstanceID]
		scriptName, resolveErr := ResolveScript(ctx, "install_skill_from_smh", agentType)
		if resolveErr != nil {
			return hcommon.I18nError(i18n.MsgUnsupportedAgentType, agentType)
		}

		_, err := RunScript(ctx, record.InstanceCID, scriptName, 120, ruMap[record.InstanceID], nil, map[string]string{
			"download_url":  downloadURL,
			"skill_slug":    req.Slug,
			"skill_version": skill.Version,
		})
		return err
	})

	jsonOK(w, map[string]interface{}{
		"ok":      true,
		"task_id": task.ID,
		"version": skill.Version,
	})
}

// HandleSkillStoreDownload 技能广场 — 下载技能 zip 包（302 跳转到 SMH）
// 同时原子递增 distribute_count。
// GET /tenant/skills/download?slug=xxx&version=1.0.0
func HandleSkillStoreDownload(w http.ResponseWriter, r *http.Request) {
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	if !requireSMHEnabled(w, r) {
		return
	}

	slug := r.URL.Query().Get("slug")
	if slug == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillStoreSlugRequired))
		return
	}

	// 查找技能（仅允许下载已上架的技能）
	version := r.URL.Query().Get("version")
	var skill model.Skill
	if version == "" || version == "latest" {
		if model.DB(r.Context()).Where("slug = ? AND status = ?", slug, model.SkillStatusPublished).Order("version_major DESC, version_minor DESC, version_patch DESC").First(&skill).Error != nil {
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgSkillNotExist))
			return
		}
	} else {
		if model.DB(r.Context()).Where("slug = ? AND version = ? AND status = ?", slug, version, model.SkillStatusPublished).First(&skill).Error != nil {
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgPluginVersionNotFoundDetail, slug, version))
			return
		}
	}

	// 可见性校验
	visible, err := model.IsSkillVisibleToUser(r.Context(), &skill, user.ID)
	if err != nil || !visible {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgSkillNotExist))
		return
	}

	// 构建 SMH 下载 URL：{slug}/{slug}-{version}.zip
	cosZipKey := fmt.Sprintf("%s/%s-%s.zip", skill.Slug, skill.Slug, skill.Version)
	downloadURL, err := buildSMHDownloadURL(r.Context(), cosZipKey, false)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgSkillStoreGenDownloadURLFail, err))
		return
	}

	// 原子递增下载计数
	model.DB(r.Context()).Model(&model.Skill{}).Where("id = ?", skill.ID).
		UpdateColumn("distribute_count", gorm.Expr("distribute_count + 1"))

	http.Redirect(w, r, downloadURL, http.StatusFound)
}

// HandleSkillStoreUninstall 技能广场 — 卸载技能（从用户自己的实例上移除）
// POST /openclaw/skillstore/uninstall
func HandleSkillStoreUninstall(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	user := requireLogin(w, r)
	if user == nil {
		return
	}

	var req struct {
		Slug        string `json:"slug"`
		InstanceIDs []uint `json:"instance_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgPluginRequestFormatErr, err))
		return
	}
	if req.Slug == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "slug"))
		return
	}
	if len(req.InstanceIDs) == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInstanceIdsCannotBeEmpty))
		return
	}

	// 去重
	seen := make(map[uint]bool, len(req.InstanceIDs))
	var uniqueIDs []uint
	for _, id := range req.InstanceIDs {
		if !seen[id] {
			seen[id] = true
			uniqueIDs = append(uniqueIDs, id)
		}
	}
	req.InstanceIDs = uniqueIDs

	// 查找技能（取最新版本）
	var skill model.Skill
	if model.DB(r.Context()).Where("slug = ?", req.Slug).Order("version_major DESC, version_minor DESC, version_patch DESC").First(&skill).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgSkillNotExist))
		return
	}

	// 可见性校验
	visible, visErr := model.IsSkillVisibleToUser(r.Context(), &skill, user.ID)
	if visErr != nil || !visible {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgSkillNotExist))
		return
	}

	// ── 安全校验：所有 instance_ids 必须属于当前用户 ──
	var ownedCount int64
	model.DB(r.Context()).Model(&model.Instance{}).
		Where("id IN ? AND user_id = ?", req.InstanceIDs, user.ID).
		Count(&ownedCount)
	if int(ownedCount) != len(req.InstanceIDs) {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgSkillStoreNotOwnInstance))
		return
	}

	// 获取分布式锁（与下发共用同一把锁，确保互斥）
	lockKey := fmt.Sprintf("skill_dist:%d", skill.ID)
	lock, lockErr := model.AcquireLock(hcommon.WithTaskTrace(hcommon.DetachContext(r.Context()), "skillstore_uninstall"), lockKey, 30*time.Minute)
	if lockErr != nil {
		slog.Warn("[SkillStoreUninstall] 获取锁失败", "slug", req.Slug, "error", lockErr)
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgSkillStoreSkillLocked))
		return
	}

	// 批量查询实例信息（额外拉 source）
	type instInfo struct {
		ID          uint
		InstanceId  string
		RuntimeUser string
		AgentType   string
		Source      string
	}
	var instInfos []instInfo
	if err := model.DB(r.Context()).Model(&model.Instance{}).
		Select("id, instance_id, runtime_user, agent_type, source").
		Where("id IN ?", req.InstanceIDs).
		Scan(&instInfos).Error; err != nil {
		lock.Release()
		slog.Error("[SkillStoreUninstall] 查询实例信息失败", "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgSkillStoreQueryInstanceFail))
		return
	}

	// 过滤不支持技能的实例类型
	allAgentTypes := model.GetAllAgentTypesMap(r.Context())
	cidMap := make(map[uint]string, len(instInfos))
	ruMap := make(map[uint]string, len(instInfos))
	atMap := make(map[uint]string, len(instInfos))
	srcMap := make(map[uint]string, len(instInfos))
	var validIDs []uint
	for _, info := range instInfos {
		if !model.AgentTypeSupportsSkillByMap(info.AgentType, allAgentTypes) {
			continue
		}
		cidMap[info.ID] = info.InstanceId
		ruMap[info.ID] = info.RuntimeUser
		atMap[info.ID] = info.AgentType
		srcMap[info.ID] = info.Source
		validIDs = append(validIDs, info.ID)
	}
	if len(validIDs) == 0 {
		lock.Release()
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillStoreNoValidUninstall))
		return
	}
	req.InstanceIDs = validIDs

	// 创建卸载任务
	task := model.SkillDistributionTask{
		SkillID:    skill.ID,
		Version:    skill.Version,
		OperatorID: user.ID,
		Total:      len(req.InstanceIDs),
		Status:     "running",
		Type:       model.TaskTypeUninstall,
	}
	if err := model.DB(r.Context()).Create(&task).Error; err != nil {
		lock.Release()
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgSkillStoreCreateUninstallTask, err))
		return
	}

	// 批量构造并插入卸载记录
	records := make([]model.SkillDistributionRecord, 0, len(req.InstanceIDs))
	for _, instID := range req.InstanceIDs {
		records = append(records, model.SkillDistributionRecord{
			TaskID:      task.ID,
			SkillID:     skill.ID,
			InstanceID:  instID,
			InstanceCID: cidMap[instID],
			Version:     skill.Version,
			Status:      "pending",
			Type:        model.TaskTypeUninstall,
		})
	}
	if err := model.DB(r.Context()).Create(&records).Error; err != nil {
		lock.Release()
		slog.Error("[SkillStoreUninstall] 创建卸载记录失败", "task_id", task.ID, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgSkillStoreCreateUninstallRec, err))
		return
	}

	// 拆分：CVM record 走 executor（TAT）；local record 仅落表等 reporter sync
	cvmRecords := make([]model.SkillDistributionRecord, 0, len(records))
	for _, rec := range records {
		if srcMap[rec.InstanceID] == model.InstanceSourceLocal {
			continue
		}
		cvmRecords = append(cvmRecords, rec)
	}
	if len(cvmRecords) == 0 {
		lock.Release()
		jsonOK(w, map[string]interface{}{
			"ok":      true,
			"task_id": task.ID,
		})
		return
	}

	// 异步并发执行卸载（仅 CVM records）
	executeSkillTaskAsync(SkillTaskConfig{
		Ctx:      hcommon.DetachContext(r.Context()),
		Task:     task,
		Records:  cvmRecords,
		Lock:     lock,
		Slug:     req.Slug,
		SkillIDs: []uint{skill.ID},
		OnFailed: func(ctx context.Context, record model.SkillDistributionRecord) string {
			return model.ResolveUninstallFailedStatus(ctx, record.InstanceID, []uint{skill.ID}, skill.Version)
		},
	}, func(ctx context.Context, record model.SkillDistributionRecord) error {
		agentType := atMap[record.InstanceID]
		scriptName, resolveErr := ResolveScript(ctx, "uninstall_skill", agentType)
		if resolveErr != nil {
			return hcommon.I18nError(i18n.MsgUnsupportedAgentType, agentType)
		}
		_, err := RunScript(ctx, record.InstanceCID, scriptName, 60, ruMap[record.InstanceID], nil, map[string]string{
			"skill_slug": req.Slug,
		})
		return err
	})

	jsonOK(w, map[string]interface{}{
		"ok":      true,
		"task_id": task.ID,
	})
}
