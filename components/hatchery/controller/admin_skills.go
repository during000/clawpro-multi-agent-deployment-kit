package controller

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm"
)

// fileListMaxSize MySQL TEXT 类型最大存储 65535 字节
const fileListMaxSize = 65535

var slugRegexp = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,48}[a-z0-9]$`)

// skillDistributeWG 可选的 WaitGroup，用于测试等待 HandleDistributeSkill 的后台 goroutine 完成。
// 生产环境为 nil（不使用），测试中通过赋值启用。
var skillDistributeWG *sync.WaitGroup

// errSkillHasRunningTask 技能有进行中的下发任务时返回此错误，用于区分业务错误和系统错误
var errSkillHasRunningTask = hcommon.I18nError(i18n.MsgPluginVersionInUse)

func isValidSlug(s string) bool {
	return slugRegexp.MatchString(s)
}

// isFileListTooLarge 检查 JSON 序列化后的文件列表是否超过 MySQL TEXT 字段限制（65535 字节）。
// MySQL TEXT 类型最大存储 65535 字节，超出会导致 Error 1406: Data too long for column 'file_list'。
// 该函数在写入 DB 前做前置校验，提前返回友好错误而非让 MySQL 报错。
func isFileListTooLarge(fileListJSON []byte) bool {
	return len(fileListJSON) > fileListMaxSize
}

// parseVisibilityParams 从 multipart/form-data 中解析可见范围参数。
// 返回值：
//   - visibilityType: "all" 或 "group"
//   - groupIDs / projectIDs: 分组和项目应用范围
//   - hasScope: 请求是否传了 visibility_type、group_ids 或 project_ids（false 表示不变）
//   - err: 校验失败时返回错误（可见类型、分组或项目不存在）
//
// group_ids 和 project_ids 均为覆盖式参数：缺失或显式传空值均表示清空对应关联。
func parseVisibilityParams(r *http.Request) (visibilityType string, groupIDs, projectIDs []uint, hasScope bool, err error) {
	if r.Form == nil {
		return "", nil, nil, false, nil
	}
	_, hasVisibility := r.Form["visibility_type"]
	_, hasGroups := r.Form["group_ids"]
	_, hasProjects := r.Form["project_ids"]
	if !hasVisibility && !hasGroups && !hasProjects {
		return "", nil, nil, false, nil
	}
	if !hasVisibility {
		return "", nil, nil, true, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "visibility_type")
	}
	visibilityType = r.FormValue("visibility_type")
	if visibilityType == "" {
		visibilityType = model.VisibilityAll
	}
	if visibilityType != model.VisibilityAll && visibilityType != model.VisibilityGroup {
		return "", nil, nil, true, hcommon.I18nError(i18n.MsgInvalidVisibilityForModel)
	}
	if hasGroups {
		groupIDs, err = parseUintCSV(r.FormValue("group_ids"))
		if err != nil {
			return "", nil, nil, true, err
		}
	}
	if hasProjects {
		projectIDs, err = parseUintCSV(r.FormValue("project_ids"))
		if err != nil {
			return "", nil, nil, true, err
		}
	}
	if visibilityType == model.VisibilityGroup && len(groupIDs) == 0 && len(projectIDs) == 0 {
		return "", nil, nil, true, hcommon.I18nError(i18n.MsgGroupRequiredForVisibility)
	}
	if err := validateVisibilityGroupIDs(r.Context(), groupIDs); err != nil {
		return "", nil, nil, true, err
	}
	if err := validateVisibilityProjects(r, projectIDs); err != nil {
		return "", nil, nil, true, err
	}
	return visibilityType, groupIDs, projectIDs, true, nil
}

// validateVisibilityInput 校验可见范围参数的合法性。
// 供 form-data（parseVisibilityParams）和 JSON 解析路径共用。
// visibilityType 必须为 "all" 或 "group"；group 类型时 groupIDs 不能为空且必须在分组表中存在。
func validateVisibilityInput(ctx context.Context, visibilityType string, groupIDs []uint) error {
	if visibilityType != model.VisibilityAll && visibilityType != model.VisibilityGroup {
		return hcommon.I18nError(i18n.MsgInvalidVisibilityForModel)
	}
	if visibilityType == model.VisibilityGroup {
		if len(groupIDs) == 0 {
			return hcommon.I18nError(i18n.MsgGroupRequiredForVisibility)
		}
		return validateVisibilityGroupIDs(ctx, groupIDs)
	}
	return nil
}

// validateVisibilityGroupIDs 校验非空分组 ID 均存在。空数组用于覆盖清空旧分组关联。
func validateVisibilityGroupIDs(ctx context.Context, groupIDs []uint) error {
	if len(groupIDs) == 0 {
		return nil
	}
	groups, err := model.GetGroupsByIDs(ctx, groupIDs)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSkillVisQueryGroupFail)
	}
	existIDs := make(map[uint]bool, len(groups))
	for _, group := range groups {
		existIDs[group.ID] = true
	}
	var missing []uint
	for _, groupID := range groupIDs {
		if !existIDs[groupID] {
			missing = append(missing, groupID)
		}
	}
	if len(missing) > 0 {
		return hcommon.I18nError(i18n.MsgGroupNotFoundList, missing)
	}
	return nil
}

// skillVisibilityGroupInfo 用于技能列表响应中的可见性分组信息。
type skillVisibilityGroupInfo struct {
	GroupID   uint   `json:"group_id"`
	GroupName string `json:"group_name"`
}

// buildSkillVisibilityData 批量构建技能列表的可见性分组数据。
// 返回 map[skillID][]skillVisibilityGroupInfo（含 group_id + group_name）。
// 固定 2 次额外 DB 查询（查关联 + 查分组名称），无 N+1 问题。
func buildSkillVisibilityData(ctx context.Context, skills []model.Skill) map[uint][]skillVisibilityGroupInfo {
	result := make(map[uint][]skillVisibilityGroupInfo)

	// 筛出 visibility_type="group" 的技能 ID
	var groupSkillIDs []uint
	for _, s := range skills {
		if s.VisibilityType == model.VisibilityGroup {
			groupSkillIDs = append(groupSkillIDs, s.ID)
		}
	}
	if len(groupSkillIDs) == 0 {
		return result
	}

	// 批量查出所有关联
	skillGroupMap, err := model.GetSkillVisibilityGroupIDs(ctx, groupSkillIDs)
	if err != nil {
		slog.Error("[SkillVisibility] 批量查询技能分组关联失败", "error", err)
		return result
	}

	// 收集去重的 group_id
	groupIDSet := make(map[uint]bool)
	for _, gids := range skillGroupMap {
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

	// 批量查分组名称
	groups, rerr := model.GetGroupsByIDs(ctx, allGroupIDs)
	if rerr != nil {
		slog.Error("[SkillVisibility] 批量查询分组名称失败", "error", rerr)
		return result
	}
	groupNameMap := make(map[uint]string)
	for _, g := range groups {
		groupNameMap[g.ID] = g.Name
	}

	// 组装结果
	for skillID, gids := range skillGroupMap {
		for _, gid := range gids {
			name, ok := groupNameMap[gid]
			if !ok {
				slog.Warn("[SkillVisibility] 分组已不存在，跳过", "group_id", gid, "skill_id", skillID)
				continue
			}
			result[skillID] = append(result[skillID], skillVisibilityGroupInfo{GroupID: gid, GroupName: name})
		}
	}
	return result
}

// isDuplicateKeyError 判断是否为唯一索引冲突错误
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	// MySQL: Error 1062: Duplicate entry '...' for key '...'
	// SQLite: UNIQUE constraint failed: ...
	return strings.Contains(s, "Duplicate entry") ||
		strings.Contains(s, "UNIQUE constraint failed")
}

// findBadFileNameChar 检查文件路径中是否包含 SMH 存储不支持的特殊字符。
// 返回第一个匹配的非法字符，未找到返回空字符串。
func findBadFileNameChar(name string) string {
	// SMH / COS 文件名不支持的字符：冒号、反斜杠、竖线、星号、问号、双引号、尖括号
	for _, c := range []string{":", "\\", "|", "*", "?", "\"", "<", ">"} {
		if strings.Contains(name, c) {
			return c
		}
	}
	return ""
}

// validateSkillZip 校验 zip 包并以 SKILL.md 为锚点提取技能文件。
// 以 SKILL.md 所在目录为技能根目录，只保留该目录下的文件，其余（__MACOSX 等）全部忽略。
// 输出的 zip 内文件统一包裹在 {slug}/ 目录下：{slug}/SKILL.md、{slug}/src/...
// 返回文件列表（相对于 zip 根，如 {slug}/SKILL.md）和重新打包的 zip 数据。
func validateSkillZip(zipData []byte, slug string) ([]string, []byte, error) {
	r, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, nil, hcommon.I18nRichError(err, i18n.MsgZipParseFail, err)
	}
	if len(r.File) == 0 {
		return nil, nil, hcommon.I18nError(i18n.MsgZipEmpty)
	}
	// ZIP only records whether a name is UTF-8; it does not identify a legacy
	// code page. Guessing GB18030, Big5, Shift-JIS, etc. can silently corrupt
	// names, so reject non-UTF-8 bytes before any path processing or upload.
	for _, f := range r.File {
		if !utf8.ValidString(f.Name) {
			return nil, nil, hcommon.I18nError(i18n.MsgZipFileNameNotUTF8)
		}
	}

	const maxUncompressedSize = 200 << 20 // 200MB

	// 第一遍扫描：找到 SKILL.md，确定技能根目录
	var skillMdPath string
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		rawName := f.Name
		fileName := rawName
		if idx := strings.LastIndexAny(rawName, "/\\"); idx >= 0 {
			fileName = rawName[idx+1:]
		}
		if strings.EqualFold(fileName, "SKILL.md") {
			if skillMdPath != "" {
				return nil, nil, hcommon.I18nError(i18n.MsgZipMultiSkillMd)
			}
			skillMdPath = strings.ReplaceAll(rawName, "\\", "/")
		}
	}
	if skillMdPath == "" {
		return nil, nil, hcommon.I18nError(i18n.MsgZipNoSkillMd)
	}

	// 计算锚点前缀：SKILL.md 所在的目录
	// "SKILL.md" → anchorPrefix = ""（根目录）
	// "soulhub/SKILL.md" → anchorPrefix = "soulhub/"
	anchorPrefix := ""
	if idx := strings.LastIndex(skillMdPath, "/"); idx >= 0 {
		anchorPrefix = skillMdPath[:idx+1]
	}

	// 第二遍扫描：提取锚点目录下的文件，重新打包为扁平 zip
	var totalSize uint64
	var files []string
	var badFiles []string // 文件名含非法字符的文件
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		name := strings.ReplaceAll(f.Name, "\\", "/")
		// 只保留锚点目录下的文件
		if anchorPrefix != "" {
			if !strings.HasPrefix(name, anchorPrefix) {
				continue
			}
		}
		// Windows 打包的 zip 可能用反斜杠，统一替换为正斜杠
		if strings.Contains(name, "..") {
			return nil, nil, hcommon.I18nError(i18n.MsgZipInvalidPath, name)
		}
		// 检查文件名是否包含存储系统不支持的特殊字符
		if badChar := findBadFileNameChar(name); badChar != "" {
			relName := strings.TrimPrefix(name, anchorPrefix)
			badFiles = append(badFiles, fmt.Sprintf("%s（含字符 '%s'）", relName, badChar))
			continue
		}
		totalSize += f.UncompressedSize64
		if totalSize > maxUncompressedSize {
			return nil, nil, hcommon.I18nError(i18n.MsgZipTooLarge)
		}

		// 去掉锚点前缀，加上 {slug}/ 前缀
		flatName := slug + "/" + strings.TrimPrefix(name, anchorPrefix)

		rc, err := f.Open()
		if err != nil {
			return nil, nil, hcommon.I18nRichError(err, i18n.MsgZipReadEntryFail, err)
		}
		var fbuf bytes.Buffer
		fbuf.ReadFrom(rc)
		rc.Close()

		// 写入新 zip（扁平路径）
		newHeader := f.FileHeader
		newHeader.Name = flatName
		// The validated name is UTF-8. Clear the legacy encoding marker so
		// archive/zip writes the UTF-8 flag for non-ASCII names. Some ZIP tools
		// omit this flag even when the name bytes are already valid UTF-8.
		newHeader.NonUTF8 = false
		if !utf8.ValidString(newHeader.Comment) {
			newHeader.Comment = ""
		}
		fw, err := writer.CreateHeader(&newHeader)
		if err != nil {
			return nil, nil, hcommon.I18nRichError(err, i18n.MsgZipRepackFail, err)
		}
		fw.Write(fbuf.Bytes())

		files = append(files, flatName)
	}

	if err := writer.Close(); err != nil {
		return nil, nil, hcommon.I18nRichError(err, i18n.MsgZipFinishFail, err)
	}

	// 如果有文件名含非法字符，一次性列出所有问题文件
	if len(badFiles) > 0 {
		list := strings.Join(badFiles, "、")
		if len(badFiles) > 5 {
			list = strings.Join(badFiles[:5], "、") + fmt.Sprintf(" 等共 %d 个文件", len(badFiles))
		}
		return nil, nil, hcommon.I18nError(i18n.MsgZipBadFileName, list)
	}

	if len(files) == 0 {
		return nil, nil, hcommon.I18nError(i18n.MsgZipNoValidFile)
	}

	return files, buf.Bytes(), nil
}

// injectMetaIntoZip 将 _meta.json 注入到 zip 中 {slug}/ 目录下，与 SKILL.md 同级。
// 如果 zip 中已存在 _meta.json，会被覆盖。
func injectMetaIntoZip(zipData []byte, slug string, meta map[string]string) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, err
	}

	metaPath := slug + "/_meta.json"

	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)

	// 复制原有文件（跳过已有的 _meta.json）
	for _, f := range reader.File {
		if f.Name == metaPath {
			continue
		}
		fw, err := writer.CreateHeader(&f.FileHeader)
		if err != nil {
			return nil, err
		}
		if !f.FileInfo().IsDir() {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			var fbuf bytes.Buffer
			fbuf.ReadFrom(rc)
			rc.Close()
			fw.Write(fbuf.Bytes())
		}
	}

	// 写入 {slug}/_meta.json，与 SKILL.md 同级
	metaJSON, _ := json.Marshal(meta)
	mw, err := writer.Create(metaPath)
	if err != nil {
		return nil, err
	}
	mw.Write(metaJSON)

	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// HandleAdminSkills 查询技能列表（分页，每个 slug 只返回最新版本）
func HandleAdminSkills(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	page, pageSize := parsePagination(r)
	slog.Info("查询技能列表", "keyword", r.URL.Query().Get("keyword"), "name", r.URL.Query().Get("name"), "category_ids", r.URL.Query().Get("category_ids"), "page", page, "page_size", pageSize)

	// 基础查询：每个 slug 取最新版本（identifier 由 model 包内部注入）
	db := model.DB(r.Context()).Model(&model.Skill{}).Where("id IN (?)", model.LatestVersionSkillIDs(r.Context()))

	// 筛选条件
	// keyword：同时模糊匹配名称、描述或 slug
	if keyword := r.URL.Query().Get("keyword"); keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("name LIKE ? OR description LIKE ? OR slug LIKE ?", like, like, like)
	}
	if name := r.URL.Query().Get("name"); name != "" {
		db = db.Where("name LIKE ?", "%"+name+"%")
	}
	if slug := r.URL.Query().Get("slug"); slug != "" {
		db = db.Where("slug = ?", slug)
	}
	if desc := r.URL.Query().Get("description"); desc != "" {
		db = db.Where("description LIKE ?", "%"+desc+"%")
	}
	if catIDs := r.URL.Query().Get("category_ids"); catIDs != "" {
		ids := strings.Split(catIDs, ",")
		var intIDs []int
		for _, s := range ids {
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
	// 应用范围筛选：分组与项目同时传入时取并集，避免遗漏任一目标可见的资源。
	// group_id 支持逗号分隔多个，如 group_id=1,3。
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
	db = applySkillScopeFilter(r.Context(), db, vtFilter, parsedGIDs, projectIDs)

	// status 筛选（管理员可按状态过滤）
	if st := r.URL.Query().Get("status"); st != "" {
		db = db.Where("status = ?", st)
	}

	var total int64
	db.Count(&total)

	var skills []model.Skill
	db.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&skills)

	// 批量查询上传者 username
	uploaderMap := make(map[uint]string)
	if len(skills) > 0 {
		uploaderIDs := make([]uint, 0, len(skills))
		for _, s := range skills {
			if s.UploaderID > 0 {
				uploaderIDs = append(uploaderIDs, s.UploaderID)
			}
		}
		if len(uploaderIDs) > 0 {
			var uploaders []model.User
			model.DB(r.Context()).Select("id, username").Where("id IN ?", uploaderIDs).Find(&uploaders)
			for _, u := range uploaders {
				uploaderMap[u.ID] = u.Username
			}
		}
	}

	// 批量查询每个 skill 的待审核申请（按 slug 关联，兼容旧 pending 挂在旧版本 resource_id 的情况）
	type pendingReviewInfo struct {
		RequestID     uint   `json:"request_id"`
		ActionType    string `json:"action_type"`
		RequesterID   uint   `json:"requester_id"`
		RequesterName string `json:"requester_name"`
	}
	pendingReviewMap := make(map[string]*pendingReviewInfo)
	if len(skills) > 0 {
		slugs := make([]string, 0, len(skills))
		seenSlug := make(map[string]struct{}, len(skills))
		for _, s := range skills {
			if _, ok := seenSlug[s.Slug]; ok {
				continue
			}
			seenSlug[s.Slug] = struct{}{}
			slugs = append(slugs, s.Slug)
		}
		var pendingReqs []model.ReviewRequest
		model.DB(r.Context()).
			Where("resource_type = ? AND slug IN ? AND status = ?",
				model.ResourceTypeSkill, slugs, model.ReviewStatusPending).
			Order("id DESC").
			Find(&pendingReqs)
		if len(pendingReqs) > 0 {
			// 批量查申请人名称
			reqUserIDs := make([]uint, 0, len(pendingReqs))
			for _, pr := range pendingReqs {
				if pr.RequesterID > 0 {
					reqUserIDs = append(reqUserIDs, pr.RequesterID)
				}
			}
			reqUserMap := make(map[uint]string)
			if len(reqUserIDs) > 0 {
				var reqUsers []model.User
				model.DB(r.Context()).Select("id, username").Where("id IN ?", reqUserIDs).Find(&reqUsers)
				for _, u := range reqUsers {
					reqUserMap[u.ID] = u.Username
				}
			}
			for _, pr := range pendingReqs {
				// 同一 slug 只保留 id 最大的一条（查询已按 id DESC）
				if _, exists := pendingReviewMap[pr.Slug]; exists {
					continue
				}
				pendingReviewMap[pr.Slug] = &pendingReviewInfo{
					RequestID:     pr.ID,
					ActionType:    pr.ActionType,
					RequesterID:   pr.RequesterID,
					RequesterName: reqUserMap[pr.RequesterID],
				}
			}
		}
	}

	// 批量查询分类关联 + 最近一次下发任务
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
	type skillResp struct {
		model.Skill
		UploaderName       string                     `json:"uploader_name"`
		PendingReview      *pendingReviewInfo         `json:"pending_review"`
		Categories         []map[string]interface{}   `json:"categories"`
		LastTask           *lastTask                  `json:"last_task"`
		VisibilityGroups   []skillVisibilityGroupInfo `json:"visibility_groups"`
		VisibilityProjects []projectVisibilityInfo    `json:"visibility_projects"`
		SecurityScan       *scanStatusResp            `json:"security_scan"`
	}

	// 批量查询每个 skill 的最新 task，实时从 record 表聚合进度
	taskMap := make(map[uint]*lastTask)
	if len(skills) > 0 {
		skillIDs := make([]uint, len(skills))
		for i, s := range skills {
			skillIDs[i] = s.ID
		}
		var tasks []model.SkillDistributionTask
		// GORM 子查询：回调自动注入 identifier 条件
		subQuery := model.DB(r.Context()).Model(&model.SkillDistributionTask{}).
			Select("MAX(id)").
			Where("skill_id IN ?", skillIDs).
			Group("skill_id")
		model.DB(r.Context()).Where("id IN (?)", subQuery).Find(&tasks)

		if len(tasks) > 0 {
			// 批量聚合所有 task 的 record 计数（一条 SQL）
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
			if err := model.DB(r.Context()).Model(&model.SkillDistributionRecord{}).
				Select("task_id, status, COUNT(*) as count").
				Where("task_id IN ?", taskIDs).
				Group("task_id, status").
				Scan(&counts).Error; err != nil {
				slog.Error("查询下发记录聚合失败", "error", err)
			}

			// 按 task_id 汇总
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

	// 批量查询所有 skill 的分类关联（消除 N+1）
	categoryMap := make(map[uint][]map[string]interface{})
	if len(skills) > 0 {
		skillIDs := make([]uint, len(skills))
		for i, s := range skills {
			skillIDs[i] = s.ID
		}
		var allMappings []model.SkillCategoryMapping
		model.DB(r.Context()).Where("skill_id IN ?", skillIDs).Find(&allMappings)

		// 收集所有需要查的分类 ID
		catIDSet := make(map[uint]struct{})
		for _, m := range allMappings {
			catIDSet[m.CategoryID] = struct{}{}
		}
		catIDs := make([]uint, 0, len(catIDSet))
		for id := range catIDSet {
			catIDs = append(catIDs, id)
		}

		// 一次查询所有分类
		catMap := make(map[uint]model.SkillCategory)
		if len(catIDs) > 0 {
			var cats []model.SkillCategory
			model.DB(r.Context()).Where("id IN ?", catIDs).Find(&cats)
			for _, c := range cats {
				catMap[c.ID] = c
			}
		}

		// 按 skill_id 分组
		for _, m := range allMappings {
			if cat, ok := catMap[m.CategoryID]; ok {
				categoryMap[m.SkillID] = append(categoryMap[m.SkillID], map[string]interface{}{"id": cat.ID, "name": cat.Name})
			}
		}
	}

	// 批量加载可见性分组数据
	visibilityMap := buildSkillVisibilityData(r.Context(), skills)
	skillSlugs := make([]string, 0, len(skills))
	for _, skill := range skills {
		skillSlugs = append(skillSlugs, skill.Slug)
	}
	projectVisibilityMap, err := buildProjectVisibilityData(r.Context(), model.ProjectConfigTypeSkill, skillSlugs)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
		return
	}

	// 批量查询安全检测状态
	var scanSkillIDs []uint
	for _, s := range skills {
		scanSkillIDs = append(scanSkillIDs, s.ID)
	}
	scanMap, scanErr := model.GetSkillsSecurityStatus(r.Context(), scanSkillIDs)
	if scanErr != nil {
		slog.Error("批量查询安全检测状态失败", "error", scanErr)
		scanMap = make(map[uint]*model.SkillSecurityScan)
	}

	var result []skillResp
	for _, s := range skills {
		sr := skillResp{Skill: s, LastTask: taskMap[s.ID], UploaderName: uploaderMap[s.UploaderID], PendingReview: pendingReviewMap[s.Slug]}
		sr.Categories = categoryMap[s.ID]
		if sr.Categories == nil {
			sr.Categories = []map[string]interface{}{}
		}
		sr.VisibilityGroups = visibilityMap[s.ID]
		if sr.VisibilityGroups == nil {
			sr.VisibilityGroups = []skillVisibilityGroupInfo{}
		}
		sr.VisibilityProjects = projectVisibilityMap[s.Slug]
		if sr.VisibilityProjects == nil {
			sr.VisibilityProjects = []projectVisibilityInfo{}
		}
		sr.SecurityScan = buildScanStatusResp(scanMap[s.ID])
		result = append(result, sr)
	}

	jsonOK(w, map[string]interface{}{
		"skills":    result,
		"page":      page,
		"page_size": pageSize,
		"total":     total,
	})
}

func applySkillScopeFilter(ctx context.Context, db *gorm.DB, visibilityType string, groupIDs []int, projectIDs []uint) *gorm.DB {
	var groupQuery *gorm.DB
	if len(groupIDs) > 0 {
		groupQuery = model.DB(ctx).Model(&model.SkillVisibilityGroup{}).
			Select("skill_id").Where("group_id IN ?", groupIDs)
	}
	var projectQuery *gorm.DB
	if len(projectIDs) > 0 {
		projectQuery = model.DB(ctx).Model(&model.ProjectConfigBinding{}).
			Select("config_key").Where("config_type = ? AND project_id IN ?", model.ProjectConfigTypeSkill, projectIDs)
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

// HandleCreateSkill 创建技能
func HandleCreateSkill(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	var ownerID uint
	if user, err := RequestUser(r); user != nil && err == nil {
		ownerID = user.ID
	}

	uploadStart := time.Now()
	prep, upErr := prepareSkillUploadFromForm(r, ownerID)
	if upErr != nil {
		slog.Warn("技能创建预处理失败", "error", upErr.Err)
		writeSkillUploadError(w, r, upErr)
		return
	}
	skill := prep.Skill
	slug, version := skill.Slug, skill.Version
	maxExisting := prep.MaxExisting
	slog.Info("开始创建技能", "slug", slug, "name", skill.Name, "version", version,
		"file_count", len(prep.Files), "zip_size", len(prep.ZipData))

	// ── 先写 DB，再上传 COS ──
	// 确保 DB 成功后再上传，COS 上传失败则回滚 DB，避免孤儿文件。
	tx := model.DB(r.Context()).Begin()

	if err := tx.Create(&skill).Error; err != nil {
		tx.Rollback()
		if isDuplicateKeyError(err) {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillVersionExist, slug, version))
			return
		}
		slog.Error("技能写入数据库失败", "slug", slug, "version", version, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSkillCreateRecordFail))
		return
	}

	createSkillCategoryMappings(tx, skill.ID, r.FormValue("category_ids"))

	// 处理应用范围：显式传入 > 旧版本继承 > 默认 all
	var removedProjectIDs, removedGroupIDs []uint
	visType, visGroupIDs, visProjectIDs, hasScope, visErr := parseVisibilityParams(r)
	if visErr != nil {
		tx.Rollback()
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(visErr))
		return
	}
	if hasScope {
		oldGroupIDs, oldProjectIDs, dErr := loadSkillScopeIDs(tx, skill.ID, slug)
		if dErr != nil {
			tx.Rollback()
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(dErr, i18n.MsgSkillSetVisibilityFail))
			return
		}
		if err := model.SetSkillVisibility(tx, skill.ID, visType, visGroupIDs); err != nil {
			tx.Rollback()
			slog.Error("设置技能可见范围失败", "slug", slug, "version", version, "error", err)
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSkillSetVisibilityFail))
			return
		}
		if err := model.ReplaceResourceProjectBindings(tx, model.ProjectConfigTypeSkill, skill.Slug, visProjectIDs); err != nil {
			tx.Rollback()
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSkillSetVisibilityFail))
			return
		}
		removedGroupIDs, removedProjectIDs = diffRemovedScope(visType, oldGroupIDs, oldProjectIDs, visGroupIDs, visProjectIDs)
	} else {
		// 从同 slug 旧版本继承
		if err := model.CopySkillVisibility(tx, slug, skill.ID); err != nil {
			tx.Rollback()
			slog.Error("继承技能可见范围失败", "slug", slug, "version", version, "error", err)
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSkillInheritVisibilityFail))
			return
		}
	}

	// 从同 slug 旧版本继承 distribute_count（下载量是技能维度，跨版本累计）
	if err := model.InheritSkillDistributeCount(tx, slug, skill.ID); err != nil {
		slog.Error("继承 distribute_count 失败", "slug", slug, "version", version, "error", err)
	}

	if upErr := uploadSkillPackageToStorage(r.Context(), prep.ZipData, prep.CosZipKey, prep.CosDirKey, prep.SlugPrefix); upErr != nil {
		tx.Rollback()
		writeSkillUploadError(w, r, upErr)
		return
	}

	if err := publishScopeRemoval(r.Context(), tx, model.AssetTypeSkill, slug, removedProjectIDs, removedGroupIDs); err != nil {
		tx.Rollback()
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSkillSetVisibilityFail))
		return
	}
	// 范围缩小会先移除直接资产绑定，避免新版记录或下发落到已移出范围的目标。
	// 首次创建（slug 无旧版本）不触发：此时尚未关联到任何具体项目/分组。
	if maxExisting.ID != 0 {
		fromVer := maxExisting.Version
		publishAssetVersionForChange(r.Context(), tx, model.AssetTypeSkill, slug, fromVer, version, model.TriggerReasonAssetPublished)
	}

	tx.Commit()
	slog.Info("技能创建成功", "slug", slug, "version", version, "id", skill.ID,
		"file_count", len(prep.Files), "upload_ms", time.Since(uploadStart).Milliseconds())

	scanSubmitted, scanSkipReason := maybeSubmitSkillSecurityScan(
		r, prep.ZipData, skill.ID, skill.Version, slug+"-"+version+".zip")

	jsonOK(w, map[string]interface{}{
		"ok":               true,
		"id":               skill.ID,
		"slug":             slug,
		"version":          version,
		"scan_submitted":   scanSubmitted,
		"scan_skip_reason": scanSkipReason,
	})
}

// HandleUpdateSkill 编辑技能元信息（name/description/categories/visibility）
func HandleUpdateSkill(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	slug := r.FormValue("slug")
	version := r.FormValue("version")
	slog.Info("开始更新技能", "slug", slug, "version", version)
	if slug == "" || version == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgPluginSlugVersionRequired))
		return
	}

	var skill model.Skill
	if model.DB(r.Context()).Where("slug = ? AND version = ?", slug, version).First(&skill).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgSkillNotExist))
		return
	}

	updates := map[string]interface{}{}
	if name := r.FormValue("name"); name != "" {
		updates["name"] = name
	}
	if desc := r.FormValue("description"); desc != "" {
		updates["description"] = desc
	}

	// 预解析可见范围参数（在事务外校验，避免持锁时间过长）
	visType, visGroupIDs, visProjectIDs, hasScope, visErr := parseVisibilityParams(r)
	if visErr != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(visErr))
		return
	}
	// 在事务中原子化执行：更新元信息 + 更新分类 + 更新可见范围
	var removedProjectIDs, removedGroupIDs []uint
	if err := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		// 1. 更新 name/description
		if len(updates) > 0 {
			if err := tx.Model(&skill).Updates(updates).Error; err != nil {
				return err
			}
		}

		// 2. 更新分类关联（不传不变，传空清空，传值更新）
		if r.Form != nil {
			if _, exists := r.Form["category_ids"]; exists {
				tx.Where("skill_id = ?", skill.ID).Delete(&model.SkillCategoryMapping{})
				if catIDs := r.FormValue("category_ids"); catIDs != "" {
					for _, s := range strings.Split(catIDs, ",") {
						if id, err := strconv.Atoi(strings.TrimSpace(s)); err == nil && id > 0 {
							tx.Create(&model.SkillCategoryMapping{SkillID: skill.ID, CategoryID: uint(id)})
						}
					}
				}
			}
		}

		// 3. 更新可见范围（不传不变，传 all 清空关联，传 group 创建关联）
		if hasScope {
			// Set 之前先查旧范围，用于 diff 出被移出的存量分组/项目
			oldGroupIDs, oldProjectIDs, dErr := loadSkillScopeIDs(tx, skill.ID, skill.Slug)
			if dErr != nil {
				return dErr
			}
			if err := model.SetSkillVisibility(tx, skill.ID, visType, visGroupIDs); err != nil {
				return err
			}
			if err := model.ReplaceResourceProjectBindings(tx, model.ProjectConfigTypeSkill, skill.Slug, visProjectIDs); err != nil {
				return err
			}
			removedGroupIDs, removedProjectIDs = diffRemovedScope(visType, oldGroupIDs, oldProjectIDs, visGroupIDs, visProjectIDs)
		}
		return publishScopeRemoval(r.Context(), tx, model.AssetTypeSkill, slug, removedProjectIDs, removedGroupIDs)
	}); err != nil {
		slog.Error("技能更新事务失败", "slug", slug, "version", version, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSkillUpdateFail))
		return
	}

	slog.Info("技能更新成功", "slug", slug, "version", version, "id", skill.ID)
	jsonOK(w, map[string]interface{}{"ok": true})
}

// loadSkillScopeIDs 查询某 skill 当前的应用范围（分组可见性 + 项目绑定）ID 列表。
func loadSkillScopeIDs(tx *gorm.DB, skillID uint, slug string) (groupIDs, projectIDs []uint, err error) {
	if e := tx.Model(&model.SkillVisibilityGroup{}).
		Where("skill_id = ?", skillID).Pluck("group_id", &groupIDs).Error; e != nil {
		return nil, nil, e
	}
	if e := tx.Model(&model.ProjectConfigBinding{}).
		Where("config_type = ? AND config_key = ?", model.ProjectConfigTypeSkill, slug).
		Pluck("project_id", &projectIDs).Error; e != nil {
		return nil, nil, e
	}
	return groupIDs, projectIDs, nil
}

// HandleAdminSkillOffline 管理员下架技能（直接生效，不走审核）
// POST /admin/skills/offline  slug=xxx
func HandleAdminSkillOffline(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	slug := r.FormValue("slug")
	if slug == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillSlugRequired))
		return
	}

	result := model.DB(r.Context()).Model(&model.Skill{}).
		Where("slug = ? AND status = ?", slug, model.SkillStatusPublished).
		Update("status", model.SkillStatusOffline)
	if result.Error != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(result.Error, i18n.MsgSkillUpdateFail, result.Error))
		return
	}
	if result.RowsAffected == 0 {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgSkillNotExist))
		return
	}

	slog.Info("管理员下架技能", "slug", slug)
	jsonOK(w, map[string]interface{}{"ok": true})
}

// HandleAdminSkillOnline 管理员上架技能（直接生效，不走审核）
// POST /admin/skills/online  slug=xxx
func HandleAdminSkillOnline(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	slug := r.FormValue("slug")
	if slug == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillSlugRequired))
		return
	}

	result := model.DB(r.Context()).Model(&model.Skill{}).
		Where("slug = ? AND status = ?", slug, model.SkillStatusOffline).
		Update("status", model.SkillStatusPublished)
	if result.Error != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(result.Error, i18n.MsgSkillUpdateFail, result.Error))
		return
	}
	if result.RowsAffected == 0 {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgSkillNotExist))
		return
	}

	slog.Info("管理员上架技能", "slug", slug)
	jsonOK(w, map[string]interface{}{"ok": true})
}

// HandleDeleteSkill 删除指定 slug+version
func HandleDeleteSkill(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	slug := r.FormValue("slug")
	version := r.FormValue("version")
	cascade := r.FormValue("cascade") == "true"
	slog.Info("开始删除技能", "slug", slug, "version", version, "cascade", cascade)
	if slug == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillSlugRequired))
		return
	}

	// ── 确定删除范围 ──
	var skills []model.Skill
	if version != "" {
		// 单版本删除（向后兼容）
		var skill model.Skill
		if model.DB(r.Context()).Where("slug = ? AND version = ?", slug, version).First(&skill).Error != nil {
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgSkillNotExist))
			return
		}
		skills = append(skills, skill)
	} else {
		// 全版本删除
		model.DB(r.Context()).Where("slug = ?", slug).Find(&skills)
		if len(skills) == 0 {
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgSkillNotExist))
			return
		}
	}

	// 收集待删除 skill IDs 和版本列表
	skillIDs := make([]uint, len(skills))
	deletedVersions := make([]string, len(skills))
	for i, s := range skills {
		skillIDs[i] = s.ID
		deletedVersions[i] = s.Version
	}
	slog.Info("确定删除范围", "slug", slug, "versions", deletedVersions, "skill_ids", skillIDs, "cascade", cascade)

	// ── 事务内执行删除 ──
	var deletedBundleCount, deletedRoleCount int64
	var bundleCosKeys, roleCosKeys []string

	txErr := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		// 1. 检查 running tasks（FOR UPDATE 锁定，MySQL 模式下防幻读）
		var runningCount int64
		if err := tx.Model(&model.SkillDistributionTask{}).
			Where("skill_id IN ? AND status = ?", skillIDs, "running").
			Set("gorm:query_option", "FOR UPDATE").
			Count(&runningCount).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgSkillCheckTaskFail, err)
		}
		if runningCount > 0 {
			return errSkillHasRunningTask
		}

		// 2. cascade：级联删除技能包中的引用
		if cascade {
			bundleQ := tx.Where("slug = ?", slug)
			if version != "" {
				bundleQ = bundleQ.Where("version = ?", version)
			}
			var toDelete []model.BundleSkill
			bundleQ.Find(&toDelete)

			// 收集 cos_zip_key（事务内收集，事务外清理）
			for _, bs := range toDelete {
				if bs.CosZipKey != "" {
					bundleCosKeys = append(bundleCosKeys, bs.CosZipKey)
				}
			}
			deletedBundleCount = int64(len(toDelete))
			slog.Info("级联删除技能包技能", "slug", slug, "found", len(toDelete), "cos_keys", len(bundleCosKeys))

			if len(toDelete) > 0 {
				// 删除记录
				deleteIDs := make([]uint, len(toDelete))
				for i, bs := range toDelete {
					deleteIDs[i] = bs.ID
				}
				if err := tx.Where("id IN ?", deleteIDs).Delete(&model.BundleSkill{}).Error; err != nil {
					return hcommon.I18nRichError(err, i18n.MsgSkillCascadeDeleteBundleFail, err)
				}

				// 按 skill_bundle_id 分组，更新每个 bundle 的 skill_count
				affectedBundles := make(map[uint]bool)
				for _, bs := range toDelete {
					affectedBundles[bs.SkillBundleID] = true
				}
				for bundleID := range affectedBundles {
					var count int64
					tx.Model(&model.BundleSkill{}).Where("skill_bundle_id = ?", bundleID).Count(&count)
					tx.Model(&model.SkillBundle{}).Where("id = ?", bundleID).Update("skill_count", int(count))
				}
				slog.Info("级联删除技能包技能完成", "slug", slug, "deleted", len(toDelete), "affected_bundles", len(affectedBundles))
			}
		}

		// 3. cascade：级联删除角色中的引用
		if cascade {
			roleQ := tx.Where("slug = ?", slug)
			if version != "" {
				roleQ = roleQ.Where("version = ?", version)
			}
			var toDelete []model.OpenClawRoleSkill
			roleQ.Find(&toDelete)

			// 收集 cos_zip_key
			for _, rs := range toDelete {
				if rs.CosZipKey != "" {
					roleCosKeys = append(roleCosKeys, rs.CosZipKey)
				}
			}
			deletedRoleCount = int64(len(toDelete))
			slog.Info("级联删除角色技能", "slug", slug, "found", len(toDelete), "cos_keys", len(roleCosKeys))

			if len(toDelete) > 0 {
				deleteIDs := make([]uint, len(toDelete))
				for i, rs := range toDelete {
					deleteIDs[i] = rs.ID
				}
				if err := tx.Where("id IN ?", deleteIDs).Delete(&model.OpenClawRoleSkill{}).Error; err != nil {
					return hcommon.I18nRichError(err, i18n.MsgSkillCascadeDeleteRoleFail, err)
				}
				slog.Info("级联删除角色技能完成", "slug", slug, "deleted", len(toDelete))
			}
		}

		// 4. 删除分类关联
		tx.Where("skill_id IN ?", skillIDs).Delete(&model.SkillCategoryMapping{})

		// 5. 删除可见性关联
		tx.Where("skill_id IN ?", skillIDs).Delete(&model.SkillVisibilityGroup{})

		// 6. 批量软删除技能记录
		if err := tx.Where("id IN ?", skillIDs).Delete(&model.Skill{}).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgSkillDeleteRecordFail, err)
		}
		// 旁路：删除技能前先记录版本历史（asset_deleted，同事务）。
		// 必须在 CleanupProjectBindings 之前调用，否则绑定已删、查不到受影响目标。
		publishAssetVersionForChange(r.Context(), tx, model.AssetTypeSkill, slug, "", "", model.TriggerReasonAssetDeleted)

		var remaining int64
		if err := tx.Model(&model.Skill{}).Where("slug = ?", slug).Count(&remaining).Error; err != nil {
			return err
		}
		if remaining == 0 {
			if err := model.CleanupProjectBindings(tx, model.ProjectConfigTypeSkill, slug); err != nil {
				return err
			}
		}
		return nil
	})

	if txErr != nil {
		if errors.Is(txErr, errSkillHasRunningTask) {
			writeError(w, r, http.StatusBadRequest, errSkillHasRunningTask)
		} else {
			slog.Error("删除技能事务失败", "slug", slug, "version", version, "error", txErr)
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(txErr, i18n.MsgSkillDeleteFail))
		}
		return
	}

	// ── 事务外 COS 清理（异步） ──
	go func(ctx context.Context) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("异步 COS 清理 panic", "slug", slug, "recover", r)
			}
		}()
		slog.Info("开始异步 COS 清理", "slug", slug, "versions", deletedVersions, "bundle_cos_keys", len(bundleCosKeys), "role_cos_keys", len(roleCosKeys))
		// 清理 skills space 文件
		for _, v := range deletedVersions {
			cosDirPrefix := fmt.Sprintf("%s/%s-%s/", slug, slug, v)
			if err := deleteCOSPrefix(ctx, cosDirPrefix); err != nil {
				slog.Warn("COS cleanup failed (dir)", "prefix", cosDirPrefix, "error", err)
			}
			cosZipKey := fmt.Sprintf("%s/%s-%s.zip", slug, slug, v)
			if client, err := getStorageClient(ctx); err == nil {
				if err := client.Delete(cosZipKey, true); err != nil {
					slog.Warn("COS cleanup failed (zip)", "key", cosZipKey, "error", err)
				}
			}
		}
		// 清理 cascade 收集的 common space 文件
		if len(bundleCosKeys) > 0 || len(roleCosKeys) > 0 {
			commonClient, err := GetCommonStorageClient(ctx)
			if err != nil {
				slog.Warn("获取 common space 客户端失败，跳过级联 COS 清理", "error", err)
				return
			}
			for _, key := range bundleCosKeys {
				if err := commonClient.Delete(key, true); err != nil {
					slog.Warn("COS cleanup failed (bundle skill)", "key", key, "error", err)
				}
			}
			for _, key := range roleCosKeys {
				if err := commonClient.Delete(key, true); err != nil {
					slog.Warn("COS cleanup failed (role skill)", "key", key, "error", err)
				}
			}
		}
		slog.Info("异步 COS 清理完成", "slug", slug)
	}(hcommon.DetachContext(r.Context()))

	slog.Info("技能删除成功", "slug", slug, "version", version, "deleted_skills", len(skills),
		"deleted_bundle_skills", deletedBundleCount, "deleted_role_skills", deletedRoleCount)

	jsonOK(w, map[string]interface{}{
		"ok":             true,
		"deleted_skills": len(skills),
		"cascade_deleted": map[string]interface{}{
			"bundle_skills": deletedBundleCount,
			"role_skills":   deletedRoleCount,
		},
	})

}

// HandleSkillReferences 查询技能在技能包和角色中的关联引用
// GET /admin/skills/references?slug=xxx&version=xxx
func HandleSkillReferences(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	slug := r.URL.Query().Get("slug")
	if slug == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillStoreSlugRequired))
		return
	}
	version := r.URL.Query().Get("version")

	// ── 查询 BundleSkill 引用（两步查询，避免 JOIN 的多租户泄漏） ──
	bundleQuery := model.DB(r.Context()).Where("slug = ?", slug)
	if version != "" {
		bundleQuery = bundleQuery.Where("version = ?", version)
	}
	var bundleSkills []model.BundleSkill
	bundleQuery.Find(&bundleSkills)

	// 收集 bundle IDs → 批量查询 SkillBundle 获取 name
	bundleNameMap := make(map[uint]string)
	if len(bundleSkills) > 0 {
		bundleIDSet := make(map[uint]bool)
		for _, bs := range bundleSkills {
			bundleIDSet[bs.SkillBundleID] = true
		}
		bundleIDs := make([]uint, 0, len(bundleIDSet))
		for id := range bundleIDSet {
			bundleIDs = append(bundleIDs, id)
		}
		var bundles []model.SkillBundle
		model.DB(r.Context()).Where("id IN ?", bundleIDs).Find(&bundles)
		for _, b := range bundles {
			bundleNameMap[b.ID] = b.Name
		}
	}

	type bundleSkillRef struct {
		ID            uint   `json:"id"`
		SkillBundleID uint   `json:"skill_bundle_id"`
		BundleName    string `json:"bundle_name"`
		Version       string `json:"version"`
	}
	bundleRefs := make([]bundleSkillRef, 0, len(bundleSkills))
	for _, bs := range bundleSkills {
		bundleRefs = append(bundleRefs, bundleSkillRef{
			ID:            bs.ID,
			SkillBundleID: bs.SkillBundleID,
			BundleName:    bundleNameMap[bs.SkillBundleID],
			Version:       bs.Version,
		})
	}

	// ── 查询 OpenClawRoleSkill 引用 ──
	roleQuery := model.DB(r.Context()).Where("slug = ?", slug)
	if version != "" {
		roleQuery = roleQuery.Where("version = ?", version)
	}
	var roleSkills []model.OpenClawRoleSkill
	roleQuery.Find(&roleSkills)

	// 收集 role IDs → 批量查询 OpenClawRole 获取 name
	roleNameMap := make(map[uint]string)
	if len(roleSkills) > 0 {
		roleIDSet := make(map[uint]bool)
		for _, rs := range roleSkills {
			roleIDSet[rs.OpenClawRoleID] = true
		}
		roleIDs := make([]uint, 0, len(roleIDSet))
		for id := range roleIDSet {
			roleIDs = append(roleIDs, id)
		}
		var roles []model.OpenClawRole
		model.DB(r.Context()).Where("id IN ?", roleIDs).Find(&roles)
		for _, r := range roles {
			roleNameMap[r.ID] = r.Name
		}
	}

	type roleSkillRef struct {
		ID             uint   `json:"id"`
		OpenClawRoleID uint   `json:"openclaw_role_id"`
		RoleName       string `json:"role_name"`
		Version        string `json:"version"`
	}
	roleRefs := make([]roleSkillRef, 0, len(roleSkills))
	for _, rs := range roleSkills {
		roleRefs = append(roleRefs, roleSkillRef{
			ID:             rs.ID,
			OpenClawRoleID: rs.OpenClawRoleID,
			RoleName:       roleNameMap[rs.OpenClawRoleID],
			Version:        rs.Version,
		})
	}

	jsonOK(w, map[string]interface{}{
		"slug": slug,
		"references": map[string]interface{}{
			"bundle_skills": bundleRefs,
			"role_skills":   roleRefs,
		},
	})
}

// HandleAdminSkillDetail 查询技能详情
func HandleAdminSkillDetail(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	slug := r.URL.Query().Get("slug")
	versionParam := r.URL.Query().Get("version")
	slog.Info("查询技能详情", "slug", slug, "version", versionParam)
	if slug == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillStoreSlugRequired))
		return
	}

	version := r.URL.Query().Get("version")
	var skill model.Skill
	if version == "" || version == "latest" {
		// 取最新版本
		if model.DB(r.Context()).Where("slug = ?", slug).Order("version_major DESC, version_minor DESC, version_patch DESC").First(&skill).Error != nil {
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgSkillNotExist))
			return
		}
	} else {
		if model.DB(r.Context()).Where("slug = ? AND version = ?", slug, version).First(&skill).Error != nil {
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgPluginVersionNotFoundDetail, slug, version))
			return
		}
	}

	// 查询所有版本列表
	var allVersions []model.Skill
	model.DB(r.Context()).Where("slug = ?", slug).Order("version_major DESC, version_minor DESC, version_patch DESC").Find(&allVersions)
	var versions []string
	for _, v := range allVersions {
		versions = append(versions, v.Version)
	}

	// 查询分类
	var categories []map[string]interface{}
	var mappings []model.SkillCategoryMapping
	model.DB(r.Context()).Where("skill_id = ?", skill.ID).Find(&mappings)
	for _, m := range mappings {
		var cat model.SkillCategory
		if model.DB(r.Context()).First(&cat, m.CategoryID).Error == nil {
			categories = append(categories, map[string]interface{}{"id": cat.ID, "name": cat.Name})
		}
	}
	if categories == nil {
		categories = []map[string]interface{}{}
	}

	// 查询可见性分组
	var visGroups []skillVisibilityGroupInfo
	if skill.VisibilityType == model.VisibilityGroup {
		groupMap, _ := model.GetSkillVisibilityGroupIDs(r.Context(), []uint{skill.ID})
		if gids, ok := groupMap[skill.ID]; ok && len(gids) > 0 {
			groups, _ := model.GetGroupsByIDs(r.Context(), gids)
			nameMap := make(map[uint]string)
			for _, g := range groups {
				nameMap[g.ID] = g.Name
			}
			for _, gid := range gids {
				if name, ok := nameMap[gid]; ok {
					visGroups = append(visGroups, skillVisibilityGroupInfo{GroupID: gid, GroupName: name})
				}
			}
		}
	}
	if visGroups == nil {
		visGroups = []skillVisibilityGroupInfo{}
	}
	projectVisibilityMap, err := buildProjectVisibilityData(r.Context(), model.ProjectConfigTypeSkill, []string{skill.Slug})
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
		return
	}
	visProjects := projectVisibilityMap[skill.Slug]
	if visProjects == nil {
		visProjects = []projectVisibilityInfo{}
	}

	// 查询当前版本的安全检测状态
	latestScan, scanErr := model.GetLatestSkillSecurityScan(r.Context(), skill.ID)
	if scanErr != nil {
		slog.Error("查询技能安全检测状态失败", "skill_id", skill.ID, "error", scanErr)
	}
	securityScan := buildScanDetailResp(latestScan)

	jsonOK(w, map[string]interface{}{
		"skill": map[string]interface{}{
			"id":                  skill.ID,
			"slug":                skill.Slug,
			"name":                skill.Name,
			"version":             skill.Version,
			"description":         skill.Description,
			"changelog":           skill.Changelog,
			"categories":          categories,
			"visibility_type":     skill.VisibilityType,
			"visibility_groups":   visGroups,
			"visibility_projects": visProjects,
			"file_size":           skill.FileSize,
			"distribute_count":    skill.DistributeCount,
			"cos_zip_key":         skill.COSZipKey,
			"cos_dir_key":         skill.COSDirKey,
			"created_at":          skill.CreatedAt,
			"updated_at":          skill.UpdatedAt,
			"security_scan":       securityScan,
		},
		"versions": versions,
	})
}

// HandleAdminSkillFiles 查询技能文件列表（所有版本）
func HandleAdminSkillFiles(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	slug := r.URL.Query().Get("slug")
	slog.Info("查询技能文件列表", "slug", slug)
	if slug == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillStoreSlugRequired))
		return
	}

	var skills []model.Skill
	model.DB(r.Context()).Where("slug = ?", slug).Order("version_major DESC, version_minor DESC, version_patch DESC").Find(&skills)
	if len(skills) == 0 {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgSkillNotExist))
		return
	}

	type versionFiles struct {
		Version      string          `json:"version"`
		Files        []string        `json:"files"`
		SecurityScan *scanStatusResp `json:"security_scan"`
	}

	// 批量查询所有版本的安全检测状态
	var skillIDs []uint
	for _, s := range skills {
		skillIDs = append(skillIDs, s.ID)
	}
	scanMap, scanErr := model.GetSkillsSecurityStatus(r.Context(), skillIDs)
	if scanErr != nil {
		slog.Error("[AdminSkillFiles] 批量查询安全检测状态失败", "error", scanErr)
		scanMap = make(map[uint]*model.SkillSecurityScan)
	}

	var result []versionFiles
	for _, s := range skills {
		var files []string
		if s.FileList != "" {
			json.Unmarshal([]byte(s.FileList), &files)
		}
		if files == nil {
			files = []string{}
		}
		vf := versionFiles{Version: s.Version, Files: files}
		vf.SecurityScan = buildScanStatusResp(scanMap[s.ID])
		result = append(result, vf)
	}

	jsonOK(w, map[string]interface{}{
		"slug":     slug,
		"versions": result,
	})
}

// HandleAdminSkillTasks 查询下发任务列表（分页）
func HandleAdminSkillTasks(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	slug := strings.TrimSpace(r.URL.Query().Get("slug"))
	source := strings.TrimSpace(r.URL.Query().Get("source"))
	if source == "" {
		source = model.SkillSourceEnterprise
	}
	sourceSkillsetSlug := strings.TrimSpace(r.URL.Query().Get("source_skillset_slug"))
	batchID := strings.TrimSpace(r.URL.Query().Get("batch_id"))
	slog.Info("查询下发任务列表", "source", source, "slug", slug, "source_skillset_slug", sourceSkillsetSlug, "batch_id", batchID)
	if source != model.SkillSourceEnterprise && source != model.SkillSourcePublic {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgUnsupportedSkillSource, source))
		return
	}
	if slug == "" && sourceSkillsetSlug == "" && batchID == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillStoreSlugRequired))
		return
	}

	page, pageSize := parsePagination(r)
	typeFilter := r.URL.Query().Get("type")
	if source == model.SkillSourcePublic && sourceSkillsetSlug != "" && batchID == "" {
		handlePublicSkillsetTaskBatches(w, r, sourceSkillsetSlug, typeFilter, page, pageSize)
		return
	}
	var total int64

	taskQuery := model.DB(r.Context()).Model(&model.SkillDistributionTask{})
	var skillIDs []uint
	if batchID != "" {
		taskQuery = taskQuery.Where("batch_id = ?", batchID)
	} else if sourceSkillsetSlug != "" {
		if source != model.SkillSourcePublic {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "source_skillset_slug"))
			return
		}
		taskQuery = taskQuery.Where("source = ? AND source_skillset_slug = ?", model.SkillSourcePublic, sourceSkillsetSlug)
	} else if source == model.SkillSourcePublic {
		taskQuery = taskQuery.Where("source = ? AND slug = ?", model.SkillSourcePublic, slug)
	} else {
		model.DB(r.Context()).Model(&model.Skill{}).Where("slug = ?", slug).Pluck("id", &skillIDs)
		if len(skillIDs) == 0 {
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgSkillNotExist))
			return
		}
		taskQuery = taskQuery.Where("skill_id IN ?", skillIDs)
	}
	if typeFilter != "" && typeFilter != "all" {
		taskQuery = taskQuery.Where("type = ?", typeFilter)
	}
	taskQuery.Count(&total)

	var tasks []model.SkillDistributionTask
	taskQuery.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&tasks)

	type recordResp struct {
		InstanceID    uint   `json:"instance_id"`
		CVMInstanceID string `json:"cvm_instance_id"`
		InstanceName  string `json:"instance_name"`
		Username      string `json:"username"`
		Status        string `json:"status"`
		Error         string `json:"error"`
	}
	type taskResp struct {
		ID                 uint         `json:"id"`
		CreatedAt          interface{}  `json:"created_at"`
		Operator           string       `json:"operator"`
		Version            string       `json:"version"`
		Source             string       `json:"source"`
		Slug               string       `json:"slug"`
		SourceSkillsetSlug string       `json:"source_skillset_slug"`
		BatchID            string       `json:"batch_id"`
		Total              int          `json:"total"`
		Success            int          `json:"success"`
		Failed             int          `json:"failed"`
		Pending            int          `json:"pending"`
		Status             string       `json:"status"`
		Type               string       `json:"type"`
		Records            []recordResp `json:"records"`
	}

	result := make([]taskResp, 0, len(tasks))
	if len(tasks) > 0 {
		// 批量查询所有 task 的状态聚合（消除 N+1）
		taskIDs := make([]uint, len(tasks))
		operatorIDs := make(map[uint]struct{})
		for i, t := range tasks {
			taskIDs[i] = t.ID
			if t.OperatorID > 0 {
				operatorIDs[t.OperatorID] = struct{}{}
			}
		}

		// 一次 SQL 聚合所有 task 的 record 计数
		type taskStatusCount struct {
			TaskID uint
			Status string
			Count  int
		}
		var allCounts []taskStatusCount
		model.DB(r.Context()).Model(&model.SkillDistributionRecord{}).
			Select("task_id, status, COUNT(*) as count").
			Where("task_id IN ?", taskIDs).
			Group("task_id, status").
			Scan(&allCounts)
		type counters struct{ Success, Failed, Pending int }
		countMap := make(map[uint]*counters)
		for _, c := range allCounts {
			if countMap[c.TaskID] == nil {
				countMap[c.TaskID] = &counters{}
			}
			switch c.Status {
			case "success":
				countMap[c.TaskID].Success = c.Count
			case "pending":
				countMap[c.TaskID].Pending = c.Count
			default:
				countMap[c.TaskID].Failed += c.Count
			}
		}

		// 批量查询操作人
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

		// 批量查询所有 record
		var allRecords []model.SkillDistributionRecord
		model.DB(r.Context()).Where("task_id IN ?", taskIDs).Find(&allRecords)

		// 批量查询所有涉及的实例 + 用户
		instIDSet := make(map[uint]struct{})
		for _, rec := range allRecords {
			instIDSet[rec.InstanceID] = struct{}{}
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
		// 批量查询实例关联的用户名
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

		// 按 task_id 分组 records
		recordsByTask := make(map[uint][]model.SkillDistributionRecord)
		for _, rec := range allRecords {
			recordsByTask[rec.TaskID] = append(recordsByTask[rec.TaskID], rec)
		}

		for _, t := range tasks {
			c := countMap[t.ID]
			respSource := t.Source
			if respSource == "" {
				respSource = model.SkillSourceEnterprise
			}
			respSlug := t.Slug
			if respSlug == "" {
				respSlug = slug
			}
			tr := taskResp{
				ID:                 t.ID,
				CreatedAt:          t.CreatedAt,
				Version:            t.Version,
				Source:             respSource,
				Slug:               respSlug,
				SourceSkillsetSlug: t.SourceSkillsetSlug,
				BatchID:            t.BatchID,
				Total:              t.Total,
				Status:             t.Status,
				Type:               t.Type,
				Operator:           userMap[t.OperatorID],
			}
			if c != nil {
				tr.Success = c.Success
				tr.Failed = c.Failed
				tr.Pending = c.Pending
			}
			for _, rec := range recordsByTask[t.ID] {
				rr := recordResp{
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

// HandleAdminSkillInstances 查询实例安装情况（分页）
func HandleAdminSkillInstances(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}
	if r.Method == http.MethodPost {
		handlePublicSkillsetInstances(w, r)
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	slug := strings.TrimSpace(r.URL.Query().Get("slug"))
	source := strings.TrimSpace(r.URL.Query().Get("source"))
	if source == "" {
		source = model.SkillSourceEnterprise
	}
	statusFilter := r.URL.Query().Get("status")
	search := r.URL.Query().Get("search")
	instanceType := r.URL.Query().Get("instance_type")
	requestedVersion := strings.TrimSpace(r.URL.Query().Get("version"))
	slog.Info("查询实例安装情况", "source", source, "slug", slug, "status", statusFilter, "instance_type", instanceType)
	if source != model.SkillSourceEnterprise && source != model.SkillSourcePublic {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgUnsupportedSkillSource, source))
		return
	}
	if slug == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillStoreSlugRequired))
		return
	}

	// 构造基础查询：LEFT JOIN 最新下发记录 + 用户名 + 本地实例 skill 事实快照，在 SQL 层推导安装状态
	// 两个数据源分支分别构造查询：企业版本来自数据库，公共版本来自规范化后的请求参数。
	// 安装状态筛选（SQL 层预过滤，减少全量数据量）
	// 复用状态 CASE 确保与 SELECT 中的 CASE 逻辑一致。
	var statuses []string
	if statusFilter != "" {
		statuses = strings.Split(statusFilter, ",")
	}

	var (
		skillIDs      []uint
		latestVersion string
		baseQuery     *gorm.DB
	)
	if source == model.SkillSourceEnterprise {
		model.DB(r.Context()).Model(&model.Skill{}).Where("slug = ?", slug).Pluck("id", &skillIDs)
		if len(skillIDs) == 0 {
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgSkillNotExist))
			return
		}
		model.DB(r.Context()).Model(&model.Skill{}).Where("id IN ?", skillIDs).
			Order("version_major DESC, version_minor DESC, version_patch DESC").
			Limit(1).Pluck("version", &latestVersion)
		baseQuery = model.BuildSkillInstanceQuery(r.Context(), skillIDs, latestVersion, slug).
			Scopes(model.FilterSkillInstallStatuses(latestVersion, statuses))
	} else if normalizedVersion, err := model.NormalizeSkillVersion(requestedVersion); err == nil {
		baseQuery = buildPublicSkillInstanceQuery(r.Context(), slug, normalizedVersion).
			Scopes(model.FilterSkillInstallStatuses(normalizedVersion, statuses))
	} else {
		baseQuery = buildPublicSkillInstanceQueryWithoutVersion(r.Context(), slug).
			Scopes(filterPublicSkillInstallStatusesWithoutVersion(statuses))
	}

	page, pageSize := parsePagination(r, 500)

	type instResp struct {
		InstanceID            uint       `json:"instance_id"              gorm:"column:instance_id"`
		CVMInstanceID         string     `json:"cvm_instance_id"          gorm:"column:cvm_instance_id"`
		InstanceName          string     `json:"instance_name"            gorm:"column:instance_name"`
		InstanceType          string     `json:"instance_type"            gorm:"column:instance_type"`
		UserID                uint       `json:"user_id"                  gorm:"column:user_id"`
		Source                string     `json:"-"                        gorm:"column:source"`
		Username              string     `json:"username"                 gorm:"column:username"`
		LastCVMState          string     `json:"last_cvm_state"           gorm:"column:last_cvm_state"`
		LastStableState       string     `json:"-"                        gorm:"column:last_stable_state"`
		CurrentOperation      string     `json:"-"                        gorm:"column:current_operation"`
		CurrentOperationState string     `json:"-"                        gorm:"column:current_operation_state"`
		AgentReady            int        `json:"-"                        gorm:"column:agent_ready"`
		CLSAgentStatus        int        `json:"-"                        gorm:"column:cls_agent_status"`
		CLSAgentStatusAt      *time.Time `json:"-"                        gorm:"column:cls_agent_status_at"`
		Status                string     `json:"status"                   gorm:"column:install_status"`
		Version               string     `json:"version"                  gorm:"column:version"`
		LatestVersion         string     `json:"latest_version"            gorm:"column:latest_version"`
	}

	// 按用户组筛选实例（辅助筛选，支持逗号分隔多个 group_id）
	// group_id=0 表示未分组用户的实例，可与正常 group_id 组合使用，如 group_id=0,1,3
	if groupIDStr := r.URL.Query().Get("group_id"); groupIDStr != "" {
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
			// 仅指定分组（使用子查询避免 JOIN 产生重复行）
			groupedSubQ := model.DB(r.Context()).Model(&model.UserGroupMember{}).Select("DISTINCT user_id").Where("user_group_id IN ?", groupIDs)
			baseQuery = baseQuery.Where("instances.user_id IN (?)", groupedSubQ)
		}
	}

	if search != "" {
		baseQuery = baseQuery.Where("instances.name LIKE ? OR instances.instance_id LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// 按实例类型筛选（支持逗号分隔多类型，如 instance_type=openclaw,Hermes）
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

	// ── 第一步：全量查询（不分页），用于批量计算实例语义状态后内存过滤 ──
	var allResults []instResp
	if err := baseQuery.Order("instances.created_at DESC").Scan(&allResults).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSkillStoreQueryInstancesFail))
		return
	}
	if allResults == nil {
		allResults = []instResp{}
	}

	// ── 第二步：批量查询 CVM 实时状态 ──
	var cvmIDs []string
	for _, r := range allResults {
		if r.CVMInstanceID != "" {
			cvmIDs = append(cvmIDs, r.CVMInstanceID)
		}
	}
	cvmInfoMap := batchFetchCVMInfoMap(r.Context(), cvmIDs)

	// ── 第三步：计算每个实例的语义状态，过滤出 running 的实例 ──
	// 批量预查：消除循环内 N+1 DB 查询
	siteConfig := model.GetSiteConfig(r.Context())
	allAgentTypes := model.GetAllAgentTypesMap(r.Context())

	instIDs := make([]uint, 0, len(allResults))
	localInstIDs := make([]uint, 0)
	for _, item := range allResults {
		instIDs = append(instIDs, item.InstanceID)
		if item.Source == model.InstanceSourceLocal {
			localInstIDs = append(localInstIDs, item.InstanceID)
		}
	}
	installingSkillMap := batchHasInstallingSkillInstallations(r.Context(), instIDs)
	localInfoMap := batchResolveLocalInstanceStatus(r.Context(), localInstIDs)
	batch := &InstanceStatusBatchLookup{SiteConfig: siteConfig, InstallingSkillMap: installingSkillMap, LocalInfoMap: localInfoMap}

	type instWithStatus struct {
		instResp
		InstanceStatus      string
		InstanceStatusLabel string
		Transient           bool
	}
	var runningResults []instWithStatus
	for _, item := range allResults {
		// 过滤不支持技能的实例类型。
		// 本地 agent 实例（source=local，agent_type=workbuddy / codebuddy）未注册到内置
		// agentTypesMap，AgentTypeSupportsSkill 会返回 false；但本地实例的 skill 能力
		// 由 reporter ack 链路保证（skill_distribution_records / local_instance_skills
		// 同时会被更新），不依赖该能力位，所以这里放行。
		// 二期正式注册本地 agent 类型后可移除本口子。
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
		cvmInfo := cvmInfoMap[item.CVMInstanceID]
		statusResp := ResolveInstanceStatus(r.Context(), &tmpInst, cvmInfo, batch)
		// 只保留 instance_status=running 的实例
		if statusResp.Status != model.StatusRunning {
			continue
		}
		runningResults = append(runningResults, instWithStatus{
			instResp:            item,
			InstanceStatus:      statusResp.Status,
			InstanceStatusLabel: statusResp.Label,
			Transient:           statusResp.Transient,
		})
	}

	// ── 第四步：内存分页 ──
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

	// 批量加载用户所属分组
	userIDSet := make(map[uint]bool)
	for _, r := range pageResults {
		if r.UserID > 0 {
			userIDSet[r.UserID] = true
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
			slog.Error("[SkillInstances] 批量查询用户分组失败", "error", err)
		}
	}

	type groupInfo struct {
		GroupID   uint   `json:"group_id"`
		GroupName string `json:"group_name"`
	}
	type instFinalResp struct {
		instResp
		UserGroups          []groupInfo `json:"user_groups"`
		InstanceStatus      string      `json:"instance_status"`
		InstanceStatusLabel string      `json:"instance_status_label"`
		Transient           bool        `json:"transient"`
	}
	finalResults := make([]instFinalResp, 0, len(pageResults))
	for _, r := range pageResults {
		item := instFinalResp{
			instResp:            r.instResp,
			InstanceStatus:      r.InstanceStatus,
			InstanceStatusLabel: r.InstanceStatusLabel,
			Transient:           r.Transient,
		}
		if groups, ok := userGroupMap[r.UserID]; ok {
			for _, g := range groups {
				item.UserGroups = append(item.UserGroups, groupInfo{GroupID: g.ID, GroupName: g.Name})
			}
		}
		if item.UserGroups == nil {
			item.UserGroups = []groupInfo{}
		}
		finalResults = append(finalResults, item)
	}

	jsonOK(w, map[string]interface{}{
		"instances": finalResults,
		"page":      page,
		"page_size": pageSize,
		"total":     total,
	})
}

// HandleDistributeSkill 批量下发技能到实例
func HandleDistributeSkill(w http.ResponseWriter, r *http.Request) {
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
		Source             string                       `json:"source"`
		Slug               string                       `json:"slug"`
		Version            string                       `json:"version"`
		SourceSkillsetSlug string                       `json:"source_skillset_slug"`
		Skills             []distributeSkillRequestItem `json:"skills"`
		distributionSelection
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("技能下发请求解析失败", "error", err)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgPluginRequestFormatErr))
		return
	}
	if err := req.distributionSelection.validate(); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if req.SelectAll {
		if _, err := normalizeSkillDistributionStatuses(req.Statuses); err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
			return
		}
	}
	slog.Info("开始批量下发技能", "source", req.Source, "slug", req.Slug, "version", req.Version, "select_all", req.SelectAll, "instance_count", len(req.InstanceIDs), "skill_count", len(req.Skills))
	if len(req.Skills) > 0 {
		handleDistributeSkillBatch(w, r, req.Source, req.Slug, req.Version, req.SourceSkillsetSlug, req.distributionSelection, req.Skills)
		return
	}
	if req.Slug == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillStoreSlugRequired))
		return
	}

	item := createSkillTaskItem(0, req.Source, req.Slug, req.Version, req.SourceSkillsetSlug)
	item, _, richErr := prepareDistributeSkillItem(r.Context(), item)
	if richErr != nil {
		writeError(w, r, http.StatusBadRequest, richErr)
		return
	}

	lock, lockErr := model.AcquireLock(hcommon.WithTaskTrace(hcommon.DetachContext(r.Context()), "skill_distribute"), skillTaskItemLockKey(item), 30*time.Minute)
	if lockErr != nil {
		slog.Warn("技能下发获取锁失败", "source", item.Source, "slug", item.Slug, "version", item.Version, "error", lockErr)
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgSkillStoreVersionLocked))
		return
	}

	var operatorID uint
	if user, err := RequestUser(r); user != nil && err == nil {
		operatorID = user.ID
	}

	if req.SelectAll {
		task, total, err := createSkillSelectAllTask(r.Context(), item, model.TaskTypeDistribute, operatorID, "", req.distributionSelection, time.Now())
		if err != nil {
			lock.Release()
			var richErr *hcommon.RichError
			if errors.As(err, &richErr) {
				writeError(w, r, http.StatusBadRequest, richErr)
			} else {
				slog.Error("[SkillSelectAll] 创建下发任务失败", "slug", item.Slug, "version", item.Version, "error", err)
				writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSkillStoreCreateRecordFail, err))
			}
			return
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
		jsonOK(w, map[string]interface{}{
			"ok":                   true,
			"task_id":              task.ID,
			"source":               item.Source,
			"slug":                 item.Slug,
			"version":              item.Version,
			"source_skillset_slug": item.SourceSkillsetSlug,
			"total":                total,
		})
		return
	}

	req.InstanceIDs = hcommon.Unique(req.InstanceIDs)
	validIDs, infoMap, skippedCount, err := loadInstancesSupportingSkillTasks(r.Context(), req.InstanceIDs)
	if err != nil {
		lock.Release()
		slog.Error("[DistributeSkill] 查询实例信息失败", "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgPluginQueryInstanceInfo))
		return
	}
	if skippedCount > 0 {
		slog.Info("技能下发跳过不支持技能的实例类型", "slug", req.Slug, "skipped", skippedCount)
	}
	if len(validIDs) == 0 {
		lock.Release()
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillStoreNoValidInstall))
		return
	}

	if err := failPreviousPendingSkillDistribute(r.Context(), item, validIDs); err != nil {
		lock.Release()
		slog.Error("判失败上一次 pending 任务失败", "slug", item.Slug, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSkillStoreCreateRecordFail))
		return
	}

	task, records, err := createSkillTaskAndRecords(r.Context(), item, model.TaskTypeDistribute, operatorID, validIDs, infoMap, "", time.Now())
	if err != nil {
		lock.Release()
		slog.Error("创建下发记录失败", "slug", req.Slug, "version", item.Version, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSkillStoreCreateRecordFail))
		return
	}
	runSkillDistributeTask(r.Context(), item, task, records, lock, infoMap, defaultSkillExecutionDependencies())

	jsonOK(w, map[string]interface{}{
		"ok":                   true,
		"task_id":              task.ID,
		"source":               item.Source,
		"slug":                 item.Slug,
		"version":              item.Version,
		"source_skillset_slug": item.SourceSkillsetSlug,
		"total":                len(validIDs),
	})
}

// HandleAdminSkillDownload 管控端 — 下载技能 zip 包（302 跳转到 SMH）
// 同时原子递增 distribute_count。
// GET /admin/skills/download?slug=xxx&version=1.0.0
func HandleAdminSkillDownload(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	if !requireSMHEnabled(w, r) {
		return
	}

	slug := r.URL.Query().Get("slug")
	if slug == "" {
		jsonAPI(w)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillStoreSlugRequired))
		return
	}

	// 查找技能（支持指定版本，不传默认最新）
	version := r.URL.Query().Get("version")
	var skill model.Skill
	if version == "" || version == "latest" {
		if model.DB(r.Context()).Where("slug = ?", slug).Order("version_major DESC, version_minor DESC, version_patch DESC").First(&skill).Error != nil {
			jsonAPI(w)
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgSkillNotExist))
			return
		}
	} else {
		if model.DB(r.Context()).Where("slug = ? AND version = ?", slug, version).First(&skill).Error != nil {
			jsonAPI(w)
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgPluginVersionNotFoundDetail, slug, version))
			return
		}
	}

	// 构建 SMH 下载 URL：{slug}/{slug}-{version}.zip
	cosZipKey := fmt.Sprintf("%s/%s-%s.zip", skill.Slug, skill.Slug, skill.Version)
	downloadURL, err := buildSMHDownloadURL(r.Context(), cosZipKey, false)
	if err != nil {
		jsonAPI(w)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSkillStoreGenDownloadURLFail))
		return
	}

	// 原子递增下载计数
	model.DB(r.Context()).Model(&model.Skill{}).Where("id = ?", skill.ID).
		UpdateColumn("distribute_count", gorm.Expr("distribute_count + 1"))

	http.Redirect(w, r, downloadURL, http.StatusFound)
}

// HandleAdminSMHToken 返回 SMH 配置信息和各 Space 的 AccessToken
func HandleAdminSMHToken(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	siteConfig := model.GetSiteConfig(r.Context())
	smhConfig := model.GetSMHConfig(r.Context())

	result := map[string]interface{}{
		"smh_enabled":              siteConfig.SMHEnabled,
		"provision_error":          siteConfig.SMHProvisionError,
		"endpoint":                 smhConfig.Endpoint,
		"library_id":               smhConfig.LibraryId,
		"auto_provision_on_create": siteConfig.SMHAutoProvisionOnCreate,
	}

	// SMH 未开通或未配置完成时，直接返回状态信息
	if siteConfig.SMHEnabled != 1 || !smhConfig.IsConfigured() {
		jsonOK(w, result)
		return
	}

	slog.Info("查询 SMH 配置信息", "endpoint", smhConfig.Endpoint, "common_space", smhConfig.CommonSpace, "skillhub_space", smhConfig.SkillhubSpace)

	// common space（返回只读 Token）
	if smhConfig.CommonSpace != "" {
		entry := map[string]interface{}{"space_id": smhConfig.CommonSpace}
		if token, err := GetCommonSpaceReadToken(r.Context()); err == nil {
			entry["access_token"] = token
		} else {
			entry["error"] = err.Error()
		}
		result["common_space"] = entry
	}

	// skillhub space（返回只读 Token）
	if smhConfig.SkillhubSpace != "" {
		entry := map[string]interface{}{"space_id": smhConfig.SkillhubSpace}
		if token, err := GetSkillhubSpaceReadToken(r.Context()); err == nil {
			entry["access_token"] = token
		} else {
			entry["error"] = err.Error()
		}
		result["skillhub_space"] = entry
	}

	jsonOK(w, result)
}

// HandleUninstallSkill 批量卸载技能（从实例上移除）
// POST /admin/skills/uninstall
func HandleUninstallSkill(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}

	var req struct {
		Source             string                      `json:"source"`
		Slug               string                      `json:"slug"`
		SourceSkillsetSlug string                      `json:"source_skillset_slug"`
		Skills             []uninstallSkillRequestItem `json:"skills"`
		distributionSelection
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgPluginRequestFormatErr))
		return
	}
	if err := req.distributionSelection.validate(); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if req.SelectAll {
		if _, err := normalizeSkillUninstallStatuses(req.Statuses); err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
			return
		}
	}
	slog.Info("开始批量卸载技能", "source", req.Source, "slug", req.Slug, "select_all", req.SelectAll, "instance_count", len(req.InstanceIDs), "skill_count", len(req.Skills))
	if len(req.Skills) > 0 {
		handleUninstallSkillBatch(w, r, req.Source, req.Slug, req.SourceSkillsetSlug, req.distributionSelection, req.Skills)
		return
	}
	if req.Slug == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillStoreSlugRequired))
		return
	}

	// 去重 instance_ids
	req.InstanceIDs = hcommon.Unique(req.InstanceIDs)

	// 查找并准备技能
	item := createSkillTaskItem(0, req.Source, req.Slug, "", req.SourceSkillsetSlug)
	preparedItem, reason, richErr := prepareUninstallSkillItem(r.Context(), item)
	if richErr != nil {
		status := http.StatusBadRequest
		if reason == "skill_not_found" && item.Source == model.SkillSourceEnterprise {
			status = http.StatusNotFound
		}
		writeError(w, r, status, richErr)
		return
	}
	item = preparedItem

	// 获取分布式锁
	lock, lockErr := model.AcquireLock(hcommon.WithTaskTrace(hcommon.DetachContext(r.Context()), "skill_uninstall"), skillTaskItemLockKey(item), 30*time.Minute)
	if lockErr != nil {
		slog.Warn("技能卸载获取锁失败", "source", item.Source, "slug", item.Slug, "error", lockErr)
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgSkillStoreSkillLocked))
		return
	}

	var operatorID uint
	if user, err := RequestUser(r); user != nil && err == nil {
		operatorID = user.ID
	}
	if req.SelectAll {
		task, total, err := createSkillSelectAllTask(r.Context(), item, model.TaskTypeUninstall, operatorID, "", req.distributionSelection, time.Now())
		if err != nil {
			lock.Release()
			var richErr *hcommon.RichError
			if errors.As(err, &richErr) {
				writeError(w, r, http.StatusBadRequest, richErr)
			} else {
				slog.Error("[SkillUninstallSelectAll] 创建卸载任务失败", "slug", item.Slug, "error", err)
				writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSkillStoreCreateUninstallTask))
			}
			return
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
		jsonOK(w, map[string]interface{}{
			"ok":      true,
			"task_id": task.ID,
			"total":   total,
		})
		return
	}

	// 过滤不支持技能的实例类型（helper 内部已对 source=local 本地 agent 实例开后门）
	validIDs, infoMap, skippedCount, err := loadInstancesSupportingSkillTasks(r.Context(), req.InstanceIDs)
	if err != nil {
		lock.Release()
		slog.Error("[UninstallSkill] 查询实例信息失败", "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgPluginQueryInstanceInfo))
		return
	}
	if skippedCount > 0 {
		slog.Info("技能卸载跳过不支持技能的实例类型", "slug", req.Slug, "skipped", skippedCount)
	}
	if len(validIDs) == 0 {
		lock.Release()
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillStoreNoValidUninstall))
		return
	}

	// 创建卸载任务和记录
	task, records, err := createSkillTaskAndRecords(r.Context(), item, model.TaskTypeUninstall, operatorID, validIDs, infoMap, "", time.Now())
	if err != nil {
		lock.Release()
		slog.Error("创建卸载记录失败", "slug", req.Slug, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSkillStoreCreateUninstallRec))
		return
	}

	// 异步执行卸载（helper 内部会自动剔除 source=local 的 record，它们保留 pending 给
	// reporter /local-agent/sync 拉取 + ack 后才转 success/failed）
	runSkillUninstallTask(r.Context(), item, task, records, lock, infoMap, defaultSkillExecutionDependencies())

	jsonOK(w, map[string]interface{}{
		"ok":      true,
		"task_id": task.ID,
	})
}
