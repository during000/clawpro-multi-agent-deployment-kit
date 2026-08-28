package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
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

type resolvedBundleSkill struct {
	Name               string
	Slug               string
	Version            string
	Source             string
	SourceSkillsetSlug string
	SourceSkillsetName string
	ZipData            []byte
}

type preparedBundleSkill struct {
	Name               string
	Slug               string
	Version            string
	Source             string
	SourceSkillsetSlug string
	SourceSkillsetName string
	CosZipKey          string
}

type bundleSkillAddResult struct {
	BundleID   uint   `json:"bundle_id"`
	BundleName string `json:"bundle_name"`
	Added      int    `json:"added"`
	SkillCount int    `json:"skill_count"`
}

type bundleSkillIdentity struct {
	slug    string
	version string
	source  string
}

// SkillAPIBaseURL 是 SkillHub 后端 API 地址（用于服务端下载技能 zip 等），
// 与 site_configs.skill_hub（前端/实例使用的 SkillHub 地址）不同。
const SkillAPIBaseURL = "https://lightmake.site"

const maxSkillBundleZipDownloadSize = 50 << 20

// SkillHTTPClient 是下载技能 zip 的专用 HTTP 客户端，带 30s 超时防止无限阻塞。
var SkillHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
}

func buildSkillHubPublicDownloadURL(slug, version string) string {
	values := url.Values{}
	values.Set("slug", slug)
	if version != "" && version != "latest" {
		values.Set("version", version)
	}
	return SkillAPIBaseURL + "/api/v1/download?" + values.Encode()
}

func publicSkillVersionFromDownloadURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	fileName := path.Base(u.Path)
	version := strings.TrimSuffix(fileName, ".zip")
	if version == fileName || version == "." || version == "/" {
		return ""
	}
	return version
}

func resolvePublicBundleSkillForAdd(slug, version, name string) (resolvedBundleSkill, int, *hcommon.RichError) {
	slug = strings.TrimSpace(slug)
	version = strings.TrimSpace(version)
	name = strings.TrimSpace(name)
	if slug == "" {
		return resolvedBundleSkill{}, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "slug")
	}

	downloadURL := buildSkillHubPublicDownloadURL(slug, version)
	zipData, finalURL, richErr := downloadSkillZipWithFinalURL(downloadURL, i18n.MsgDownloadPublicZipFail, i18n.MsgReadPublicZipFail)
	if richErr != nil {
		return resolvedBundleSkill{}, http.StatusInternalServerError, richErr
	}
	resolvedVersion := version
	if resolvedVersion == "latest" {
		resolvedVersion = ""
	}
	if resolvedVersion == "" {
		resolvedVersion = publicSkillVersionFromDownloadURL(finalURL)
		if resolvedVersion == "" {
			return resolvedBundleSkill{}, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "version")
		}
	}
	if name == "" {
		name = slug
	}

	return resolvedBundleSkill{
		Name:    name,
		Slug:    slug,
		Version: resolvedVersion,
		Source:  "public",
		ZipData: zipData,
	}, http.StatusOK, nil
}

func resolveBundleSkillForAdd(ctx context.Context, id uint, source, slug, version, name string) (resolvedBundleSkill, int, *hcommon.RichError) {
	source = strings.TrimSpace(source)
	if source == "" {
		return resolvedBundleSkill{}, http.StatusBadRequest, hcommon.I18nError(i18n.MsgAddSourceCannotBeEmpty)
	}

	switch source {
	case "enterprise":
		var skill model.Skill
		if model.DB(ctx).First(&skill, id).Error != nil {
			return resolvedBundleSkill{}, http.StatusBadRequest, hcommon.I18nError(i18n.MsgEnterpriseSkillNotFound, id)
		}
		downloadURL, err := buildSMHDownloadURL(ctx, skill.COSZipKey, false)
		if err != nil {
			return resolvedBundleSkill{}, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgGenEnterpriseDownloadURLFail)
		}
		zipData, richErr := downloadSkillZip(downloadURL, i18n.MsgDownloadEnterpriseZipFail, i18n.MsgReadEnterpriseZipFail)
		if richErr != nil {
			return resolvedBundleSkill{}, http.StatusInternalServerError, richErr
		}
		return resolvedBundleSkill{
			Name:    skill.Name,
			Slug:    skill.Slug,
			Version: skill.Version,
			Source:  source,
			ZipData: zipData,
		}, http.StatusOK, nil

	case "public":
		if strings.TrimSpace(slug) != "" {
			return resolvePublicBundleSkillForAdd(slug, version, name)
		}
		var pubSkill model.PublicSkill
		if model.DB(ctx).First(&pubSkill, id).Error != nil {
			return resolvedBundleSkill{}, http.StatusBadRequest, hcommon.I18nError(i18n.MsgPublicSkillNotFound, id)
		}
		return resolvePublicBundleSkillForAdd(pubSkill.Slug, pubSkill.Version, pubSkill.Name)

	default:
		return resolvedBundleSkill{}, http.StatusBadRequest, hcommon.I18nError(i18n.MsgUnsupportedSkillSource, source)
	}
}

func downloadSkillZip(downloadURL string, downloadKey, readKey i18n.Key) ([]byte, *hcommon.RichError) {
	zipData, _, richErr := downloadSkillZipWithFinalURL(downloadURL, downloadKey, readKey)
	return zipData, richErr
}

func downloadSkillZipWithFinalURL(downloadURL string, downloadKey, readKey i18n.Key) ([]byte, string, *hcommon.RichError) {
	resp, err := SkillHTTPClient.Get(downloadURL)
	if err != nil {
		return nil, "", hcommon.I18nRichError(err, downloadKey)
	}
	defer resp.Body.Close()
	finalURL := ""
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}
	if resp.StatusCode != http.StatusOK {
		return nil, finalURL, hcommon.I18nError(readKey, resp.StatusCode)
	}
	if resp.ContentLength > maxSkillBundleZipDownloadSize {
		return nil, finalURL, hcommon.I18nError(i18n.MsgSkillFileSizeTooLarge)
	}
	var zipBuf bytes.Buffer
	if _, readErr := zipBuf.ReadFrom(io.LimitReader(resp.Body, maxSkillBundleZipDownloadSize+1)); readErr != nil {
		return nil, finalURL, hcommon.I18nRichError(readErr, readKey, resp.StatusCode)
	}
	if zipBuf.Len() > maxSkillBundleZipDownloadSize {
		return nil, finalURL, hcommon.I18nError(i18n.MsgSkillFileSizeTooLarge)
	}
	return zipBuf.Bytes(), finalURL, nil
}

func prepareBundleSkillAdds(commonClient StorageClient, bundle model.SkillBundle, skills []resolvedBundleSkill) ([]preparedBundleSkill, *hcommon.RichError) {
	prepared := make([]preparedBundleSkill, 0, len(skills))
	for _, skill := range skills {
		cosZipKey := fmt.Sprintf("skill-bundles/%s/%s/%s-%s.zip", bundle.Name, skill.Slug, skill.Slug, skill.Version)
		if err := commonClient.Upload(cosZipKey, skill.ZipData, "application/zip"); err != nil {
			return nil, hcommon.I18nRichError(err, i18n.MsgUploadSkillZipFail)
		}
		prepared = append(prepared, preparedBundleSkill{
			Name:               skill.Name,
			Slug:               skill.Slug,
			Version:            skill.Version,
			Source:             skill.Source,
			SourceSkillsetSlug: skill.SourceSkillsetSlug,
			SourceSkillsetName: skill.SourceSkillsetName,
			CosZipKey:          cosZipKey,
		})
	}
	return prepared, nil
}

func addPreparedSkillsToBundle(tx *gorm.DB, bundleID uint, skills []preparedBundleSkill) (int, error) {
	for i, skill := range skills {
		var dupCount int64
		if err := tx.Model(&model.BundleSkill{}).Where(
			"skill_bundle_id = ? AND slug = ? AND version = ? AND source = ?",
			bundleID, skill.Slug, skill.Version, skill.Source,
		).Count(&dupCount).Error; err != nil {
			return i, err
		}
		if dupCount > 0 {
			return i, hcommon.I18nError(i18n.MsgSkillVersionConflictInBundle, skill.Slug, skill.Version)
		}

		bs := model.BundleSkill{
			SkillBundleID:      bundleID,
			Name:               skill.Name,
			Slug:               skill.Slug,
			Version:            skill.Version,
			Source:             skill.Source,
			SourceSkillsetSlug: skill.SourceSkillsetSlug,
			SourceSkillsetName: skill.SourceSkillsetName,
			CosZipKey:          skill.CosZipKey,
		}
		if err := tx.Create(&bs).Error; err != nil {
			return i, err
		}
	}

	var count int64
	if err := tx.Model(&model.BundleSkill{}).Where("skill_bundle_id = ?", bundleID).Count(&count).Error; err != nil {
		return len(skills), err
	}
	if err := tx.Model(&model.SkillBundle{}).Where("id = ?", bundleID).Update("skill_count", int(count)).Error; err != nil {
		return len(skills), err
	}
	return len(skills), nil
}

// ── 技能包 CRUD ────────────────────────────────────────────────────

// HandleCreateSkillBundle 创建技能包
// POST /admin/skill-bundles/create
func HandleCreateSkillBundle(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillBundleNameRequired))
		return
	}

	// 解析可见范围参数
	visibilityType := strings.TrimSpace(r.FormValue("visibility_type"))
	if visibilityType == "" {
		visibilityType = model.VisibilityAll
	}
	var groupIDs []uint
	if visibilityType == model.VisibilityGroup {
		gidStr := r.FormValue("group_ids")
		if gidStr != "" {
			for _, s := range strings.Split(gidStr, ",") {
				if id, e := strconv.Atoi(strings.TrimSpace(s)); e == nil && id > 0 {
					groupIDs = append(groupIDs, uint(id))
				}
			}
		}
	}
	if err := validateVisibilityInput(r.Context(), visibilityType, groupIDs); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	var bundle model.SkillBundle
	err := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		// 事务内唯一性检查
		var existing model.SkillBundle
		if tx.Where("name = ?", name).First(&existing).Error == nil {
			return fmt.Errorf("conflict")
		}

		bundle = model.SkillBundle{
			Name:           name,
			SkillCount:     0,
			Enabled:        false,
			VisibilityType: visibilityType,
		}
		if err := tx.Create(&bundle).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgCreateSkillBundleFailed)
		}

		// 设置可见性关联
		if visibilityType == model.VisibilityGroup && len(groupIDs) > 0 {
			if err := model.SetSkillBundleVisibility(tx, bundle.ID, visibilityType, groupIDs); err != nil {
				return err
			}
		}

		// 创建 SMH 目录（事务内执行，失败则回滚 DB）
		commonClient, err := GetCommonStorageClient(r.Context())
		if err != nil {
			return hcommon.I18nRichError(err, i18n.MsgGetCommonStorageClientFail)
		}
		dirKey := fmt.Sprintf("skill-bundles/%s/.keep", name)
		if err := commonClient.Upload(dirKey, []byte{}, "application/octet-stream"); err != nil {
			return hcommon.I18nRichError(err, i18n.MsgSBCreateSMHDirFailed)
		}

		return nil
	})

	if err != nil {
		if err.Error() == "conflict" {
			writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgSkillBundleConflict))
			return
		}
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgCreateSkillBundleFailed))
		return
	}

	jsonOK(w, map[string]interface{}{"ok": true, "id": bundle.ID})
}

// HandleAdminSkillBundles 查看技能包列表
// GET /admin/skill-bundles
func HandleAdminSkillBundles(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	page, pageSize := parsePagination(r)

	db := model.DB(r.Context()).Model(&model.SkillBundle{})

	if idStr := strings.TrimSpace(r.URL.Query().Get("id")); idStr != "" {
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil || id == 0 {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "id"))
			return
		}
		db = db.Where("id = ?", id)
	}
	if keyword := strings.TrimSpace(r.URL.Query().Get("keyword")); keyword != "" {
		db = db.Where("name LIKE ?", "%"+escapeSQLLike(keyword)+"%")
	}

	skillSource := strings.TrimSpace(r.URL.Query().Get("skill_source"))
	skillSlug := strings.TrimSpace(r.URL.Query().Get("skill_slug"))
	skillVersion := strings.TrimSpace(r.URL.Query().Get("skill_version"))
	sourceSkillsetSlug := strings.TrimSpace(r.URL.Query().Get("source_skillset_slug"))
	if sourceSkillsetSlug != "" && skillSlug != "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "source_skillset_slug"))
		return
	}
	if skillSource != "" {
		if skillSlug == "" || (skillSource != "public" && skillSource != "enterprise") {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "skill_source"))
			return
		}
	}
	if skillVersion != "" && skillSlug == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "skill_version"))
		return
	}
	reverseLookup := sourceSkillsetSlug != "" || skillSlug != ""
	applyBundleSkillLookup := func(q *gorm.DB) *gorm.DB {
		if sourceSkillsetSlug != "" {
			return q.Where("source_skillset_slug = ?", sourceSkillsetSlug)
		}
		if skillSource != "" {
			q = q.Where("source = ?", skillSource)
		}
		q = q.Where("slug = ?", skillSlug)
		if skillVersion != "" {
			q = q.Where("version = ?", skillVersion)
		}
		return q
	}
	if reverseLookup {
		subQ := applyBundleSkillLookup(model.DB(r.Context()).Model(&model.BundleSkill{}).Select("DISTINCT skill_bundle_id"))
		db = db.Where("id IN (?)", subQ)
	}

	// 应用范围筛选：
	// 1. 只传 visibility_type=all → 仅全局可见
	// 2. 只传 group_id → 仅匹配分组可见
	// 3. visibility_type=all + group_id → 全局 + 匹配分组
	// group_id 支持逗号分隔多个，如 group_id=1,3
	vtFilter := r.URL.Query().Get("visibility_type")
	var parsedGIDs []int
	if gidStr := r.URL.Query().Get("group_id"); gidStr != "" {
		for _, s := range strings.Split(gidStr, ",") {
			if id, err := strconv.Atoi(strings.TrimSpace(s)); err == nil && id > 0 {
				parsedGIDs = append(parsedGIDs, id)
			}
		}
	}
	if vtFilter != "" && len(parsedGIDs) > 0 {
		subQ := model.DB(r.Context()).Model(&model.SkillBundleVisibilityGroup{}).Select("skill_bundle_id").Where("group_id IN ?", parsedGIDs)
		db = db.Where("visibility_type = ? OR id IN (?)", vtFilter, subQ)
	} else if vtFilter != "" {
		db = db.Where("visibility_type = ?", vtFilter)
	} else if len(parsedGIDs) > 0 {
		subQ := model.DB(r.Context()).Model(&model.SkillBundleVisibilityGroup{}).Select("skill_bundle_id").Where("group_id IN ?", parsedGIDs)
		db = db.Where("id IN (?)", subQ)
	}

	var total int64
	db.Count(&total)

	var bundles []model.SkillBundle
	db.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&bundles)

	if bundles == nil {
		bundles = []model.SkillBundle{}
	}

	// 批量构建可见性分组数据
	visibilityMap := buildBundleVisibilityData(r.Context(), bundles)
	matchedSkillMap := make(map[uint][]model.BundleSkill)
	if reverseLookup && len(bundles) > 0 {
		bundleIDs := make([]uint, 0, len(bundles))
		for _, b := range bundles {
			bundleIDs = append(bundleIDs, b.ID)
		}
		var matchedSkills []model.BundleSkill
		lookupDB := applyBundleSkillLookup(model.DB(r.Context()).Where("skill_bundle_id IN ?", bundleIDs))
		lookupDB.Order("skill_bundle_id asc, id asc").Find(&matchedSkills)
		for _, skill := range matchedSkills {
			matchedSkillMap[skill.SkillBundleID] = append(matchedSkillMap[skill.SkillBundleID], skill)
		}
	}

	type bundleResp struct {
		model.SkillBundle
		VisibleGroups     []visibilityGroupInfo `json:"visible_groups"`
		MatchedSkillCount *int                  `json:"matched_skill_count,omitempty"`
		MatchedSkills     []model.BundleSkill   `json:"matched_skills,omitempty"`
	}
	result := make([]bundleResp, len(bundles))
	for i, b := range bundles {
		groups := visibilityMap[b.ID]
		if groups == nil {
			groups = []visibilityGroupInfo{}
		}
		resp := bundleResp{
			SkillBundle:   b,
			VisibleGroups: groups,
		}
		if reverseLookup {
			matchedSkills := matchedSkillMap[b.ID]
			if matchedSkills == nil {
				matchedSkills = []model.BundleSkill{}
			}
			matchedSkillCount := len(matchedSkills)
			resp.MatchedSkillCount = &matchedSkillCount
			resp.MatchedSkills = matchedSkills
		}
		result[i] = resp
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPages++
	}

	jsonOK(w, map[string]interface{}{
		"skill_bundles": result,
		"page":          page,
		"page_size":     pageSize,
		"total":         total,
		"total_pages":   totalPages,
	})
}

// buildBundleVisibilityData 批量构建技能包列表的可见性分组数据。
// 返回 map[bundleID][]visibilityGroupInfo（含 group_id + group_name）。
// 固定 2 次额外 DB 查询（查关联 + 查分组名称），无 N+1 问题。
func buildBundleVisibilityData(ctx context.Context, bundles []model.SkillBundle) map[uint][]visibilityGroupInfo {
	result := make(map[uint][]visibilityGroupInfo)

	// 筛出 visibility_type="group" 的技能包 ID
	var groupBundleIDs []uint
	for _, b := range bundles {
		if b.VisibilityType == model.VisibilityGroup {
			groupBundleIDs = append(groupBundleIDs, b.ID)
		}
	}
	if len(groupBundleIDs) == 0 {
		return result
	}

	// 批量查出所有关联
	bundleGroupMap, err := model.GetSkillBundleVisibilityGroupIDs(ctx, groupBundleIDs)
	if err != nil {
		slog.Error("[BundleVisibility] 批量查询技能包分组关联失败", "error", err)
		return result
	}

	// 收集去重的 group_id
	groupIDSet := make(map[uint]bool)
	for _, gids := range bundleGroupMap {
		for _, gid := range gids {
			groupIDSet[gid] = true
		}
	}
	if len(groupIDSet) == 0 {
		return result
	}
	uniqueGroupIDs := make([]uint, 0, len(groupIDSet))
	for gid := range groupIDSet {
		uniqueGroupIDs = append(uniqueGroupIDs, gid)
	}

	// 批量查分组名称
	groups, rerr := model.GetGroupsByIDs(ctx, uniqueGroupIDs)
	if rerr != nil {
		slog.Error("[BundleVisibility] 批量查询分组名称失败", "error", rerr)
		return result
	}
	groupNameMap := make(map[uint]string)
	for _, g := range groups {
		groupNameMap[g.ID] = g.Name
	}

	// 组装结果
	for bundleID, gids := range bundleGroupMap {
		infos := make([]visibilityGroupInfo, 0, len(gids))
		for _, gid := range gids {
			name := groupNameMap[gid]
			if name == "" {
				slog.Warn("[BundleVisibility] 分组已不存在，跳过", "group_id", gid, "bundle_id", bundleID)
				continue
			}
			infos = append(infos, visibilityGroupInfo{
				GroupID:   gid,
				GroupName: name,
			})
		}
		result[bundleID] = infos
	}
	return result
}

// HandleDeleteSkillBundle 删除技能包
// POST /admin/skill-bundles/delete?id=xxx
func HandleDeleteSkillBundle(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		idStr = r.FormValue("id")
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMissingParamID))
		return
	}

	var bundle model.SkillBundle
	if model.DB(r.Context()).First(&bundle, id).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgSkillBundleNotFound))
		return
	}

	if bundle.Enabled {
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgSkillBundleEnabledNeedsOff))
		return
	}

	// 删除 SMH 文件（删除失败仅记录日志）
	if commonClient, err := GetCommonStorageClient(r.Context()); err == nil {
		dirPrefix := fmt.Sprintf("skill-bundles/%s/", bundle.Name)
		if err := commonClient.DeletePrefix(dirPrefix, true); err != nil {
			slog.Warn("删除技能包 SMH 目录失败", "name", bundle.Name, "error", err)
		}
	}

	// 事务内级联删除 DB 记录
	if err := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("skill_bundle_id = ?", bundle.ID).Delete(&model.BundleSkill{}).Error; err != nil {
			return err
		}
		if err := model.CleanupSkillBundleVisibilityByBundleID(tx, bundle.ID); err != nil {
			return err
		}
		return tx.Delete(&bundle).Error
	}); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgDeleteSkillBundleFailed))
		return
	}

	jsonOK(w, map[string]interface{}{"ok": true})
}

// HandleToggleSkillBundle 启用/禁用技能包
// POST /admin/skill-bundles/toggle?id=xxx
func HandleToggleSkillBundle(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		idStr = r.FormValue("id")
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMissingParamID))
		return
	}

	txErr := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		var bundle model.SkillBundle
		// 加行锁防止多实例并发启用（MySQL 生效，SQLite 静默忽略）
		if tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&bundle, id).Error != nil {
			return fmt.Errorf("not_found")
		}

		if bundle.Enabled {
			// 当前启用 → 禁用
			return tx.Model(&bundle).Update("enabled", false).Error
		}

		// 当前禁用 → 尝试启用：仅 all 类型互斥
		if bundle.VisibilityType == model.VisibilityAll {
			var otherCount int64
			tx.Model(&model.SkillBundle{}).Where("enabled = ? AND id != ? AND visibility_type = ?", true, bundle.ID, model.VisibilityAll).Count(&otherCount)
			if otherCount > 0 {
				return fmt.Errorf("conflict")
			}
		}
		// group 类型不做互斥检查，可以多个共存
		return tx.Model(&bundle).Update("enabled", true).Error
	})

	if txErr != nil {
		switch txErr.Error() {
		case "not_found":
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgSkillBundleNotFound))
		case "conflict":
			writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgAllVisibilityBundleConflict))
		default:
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(txErr, i18n.MsgToggleSkillBundleFailed))
		}
		return
	}

	jsonOK(w, map[string]interface{}{"ok": true})
}

// HandleSkillBundleDetail 技能包详情
// GET /admin/skill-bundles/detail?id=xxx
func HandleSkillBundleDetail(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMissingParamID))
		return
	}

	var bundle model.SkillBundle
	if model.DB(r.Context()).First(&bundle, id).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgSkillBundleNotFound))
		return
	}

	var skills []model.BundleSkill
	model.DB(r.Context()).Where("skill_bundle_id = ?", bundle.ID).Find(&skills)
	if skills == nil {
		skills = []model.BundleSkill{}
	}

	// 查询可见性分组信息
	visGroups := []visibilityGroupInfo{}
	if bundle.VisibilityType == model.VisibilityGroup {
		visMap := buildBundleVisibilityData(r.Context(), []model.SkillBundle{bundle})
		if vg, ok := visMap[bundle.ID]; ok {
			visGroups = vg
		}
	}

	jsonOK(w, map[string]interface{}{
		"skill_bundle":   bundle,
		"skills":         skills,
		"visible_groups": visGroups,
	})
}

// HandleUpdateSkillBundleSkills 批量更新技能包内技能
// POST /admin/skill-bundles/update-skills?id=xxx
func HandleUpdateSkillBundleSkills(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		idStr = r.FormValue("id")
	}
	bundleID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || bundleID == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMissingParamID))
		return
	}

	var bundle model.SkillBundle
	if model.DB(r.Context()).First(&bundle, bundleID).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgSkillBundleNotFound))
		return
	}

	// 解析 JSON body
	var req struct {
		Add []struct {
			ID                 uint   `json:"id"`
			Source             string `json:"source"`
			Slug               string `json:"slug"`
			Name               string `json:"name"`
			Version            string `json:"version"`
			SourceSkillsetSlug string `json:"source_skillset_slug"`
			SourceSkillsetName string `json:"source_skillset_name"`
		} `json:"add"`
		Remove []uint `json:"remove"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}

	commonClient, err := GetCommonStorageClient(r.Context())
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgGetCommonStorageClientFail))
		return
	}

	// ── 阶段1：处理 remove 的 SMH 文件（事务外）──
	removedCount := 0
	if len(req.Remove) > 0 {
		var toRemove []model.BundleSkill
		model.DB(r.Context()).Where("id IN ? AND skill_bundle_id = ?", req.Remove, bundle.ID).Find(&toRemove)
		for _, skill := range toRemove {
			if skill.CosZipKey != "" {
				if err := commonClient.Delete(skill.CosZipKey, true); err != nil {
					slog.Warn("删除技能包技能 SMH 文件失败", "cos_zip_key", skill.CosZipKey, "error", err)
				}
			}
		}
		removedCount = len(toRemove)
	}

	// ── 阶段2：处理 add 的 SMH 文件（事务外）──
	resolvedSkills := make([]resolvedBundleSkill, 0, len(req.Add))
	for _, item := range req.Add {
		resolved, status, richErr := resolveBundleSkillForAdd(r.Context(), item.ID, item.Source, item.Slug, item.Version, item.Name)
		if richErr != nil {
			writeError(w, r, status, richErr)
			return
		}
		resolved.SourceSkillsetSlug = item.SourceSkillsetSlug
		resolved.SourceSkillsetName = item.SourceSkillsetName
		resolvedSkills = append(resolvedSkills, resolved)
	}
	addedSkills, richErr := prepareBundleSkillAdds(commonClient, bundle, resolvedSkills)
	if richErr != nil {
		writeError(w, r, http.StatusInternalServerError, richErr)
		return
	}
	// ── 阶段3：事务内批量更新 DB ──
	addedCount := 0
	txErr := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		// 删除 remove 对应的 bundle_skills
		if len(req.Remove) > 0 {
			if err := tx.Where("id IN ? AND skill_bundle_id = ?", req.Remove, bundle.ID).Delete(&model.BundleSkill{}).Error; err != nil {
				return err
			}
		}

		var err error
		addedCount, err = addPreparedSkillsToBundle(tx, bundle.ID, addedSkills)
		return err
	})
	if txErr != nil {
		var richErr *hcommon.RichError
		if errors.As(txErr, &richErr) {
			writeError(w, r, http.StatusConflict, richErr)
			return
		}
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(txErr, i18n.MsgUpdateSkillBundleSkillsFailed))
		return
	}

	// 重新查询最新 skill_count
	var finalCount int64
	model.DB(r.Context()).Model(&model.BundleSkill{}).Where("skill_bundle_id = ?", bundle.ID).Count(&finalCount)

	jsonOK(w, map[string]interface{}{
		"ok":          true,
		"skill_count": int(finalCount),
		"added":       addedCount,
		"removed":     removedCount,
	})
}

// HandleBatchAddSkillBundleSkills 将技能批量加入多个初始技能包。
// POST /admin/skill-bundles/batch-add-skills
func HandleBatchAddSkillBundleSkills(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	var req struct {
		BundleIDs []uint `json:"bundle_ids"`
		Skills    []struct {
			Slug               string `json:"slug"`
			Name               string `json:"name"`
			Version            string `json:"version"`
			SourceSkillsetSlug string `json:"source_skillset_slug"`
			SourceSkillsetName string `json:"source_skillset_name"`
		} `json:"skills"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}
	if len(req.BundleIDs) == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "bundle_ids"))
		return
	}
	if len(req.Skills) == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "skills"))
		return
	}

	bundleIDSet := make(map[uint]struct{}, len(req.BundleIDs))
	bundleIDs := make([]uint, 0, len(req.BundleIDs))
	for _, id := range req.BundleIDs {
		if id == 0 {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "bundle_ids"))
			return
		}
		if _, exists := bundleIDSet[id]; exists {
			continue
		}
		bundleIDSet[id] = struct{}{}
		bundleIDs = append(bundleIDs, id)
	}

	var bundles []model.SkillBundle
	if err := model.DB(r.Context()).Where("id IN ?", bundleIDs).Find(&bundles).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgUpdateSkillBundleSkillsFailed))
		return
	}
	if len(bundles) != len(bundleIDs) {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgSkillBundleNotFound))
		return
	}
	bundleMap := make(map[uint]model.SkillBundle, len(bundles))
	for _, b := range bundles {
		bundleMap[b.ID] = b
	}
	orderedBundles := make([]model.SkillBundle, 0, len(bundleIDs))
	for _, id := range bundleIDs {
		orderedBundles = append(orderedBundles, bundleMap[id])
	}

	commonClient, err := GetCommonStorageClient(r.Context())
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgGetCommonStorageClientFail))
		return
	}

	preparedByBundle := make(map[uint][]preparedBundleSkill, len(orderedBundles))
	seenByBundle := make(map[uint]map[bundleSkillIdentity]struct{}, len(orderedBundles))
	for _, bundle := range orderedBundles {
		var existing []model.BundleSkill
		if err := model.DB(r.Context()).Where("skill_bundle_id = ?", bundle.ID).Find(&existing).Error; err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgUpdateSkillBundleSkillsFailed))
			return
		}
		seen := make(map[bundleSkillIdentity]struct{}, len(existing)+len(req.Skills))
		for _, skill := range existing {
			seen[bundleSkillIdentity{slug: skill.Slug, version: skill.Version, source: skill.Source}] = struct{}{}
		}
		seenByBundle[bundle.ID] = seen
	}

	for _, item := range req.Skills {
		resolved, status, richErr := resolvePublicBundleSkillForAdd(item.Slug, item.Version, item.Name)
		if richErr != nil {
			writeError(w, r, status, richErr)
			return
		}
		resolved.SourceSkillsetSlug = item.SourceSkillsetSlug
		resolved.SourceSkillsetName = item.SourceSkillsetName
		for _, bundle := range orderedBundles {
			key := bundleSkillIdentity{slug: resolved.Slug, version: resolved.Version, source: resolved.Source}
			if _, ok := seenByBundle[bundle.ID][key]; ok {
				continue
			}
			seenByBundle[bundle.ID][key] = struct{}{}
			preparedSkills, richErr := prepareBundleSkillAdds(commonClient, bundle, []resolvedBundleSkill{resolved})
			if richErr != nil {
				writeError(w, r, http.StatusInternalServerError, richErr)
				return
			}
			preparedByBundle[bundle.ID] = append(preparedByBundle[bundle.ID], preparedSkills...)
		}
		resolved.ZipData = nil
	}

	addedByBundle := make(map[uint]int, len(orderedBundles))
	txErr := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		for _, bundle := range orderedBundles {
			added, err := addPreparedSkillsToBundle(tx, bundle.ID, preparedByBundle[bundle.ID])
			if err != nil {
				return err
			}
			addedByBundle[bundle.ID] = added
		}
		return nil
	})
	if txErr != nil {
		var richErr *hcommon.RichError
		if errors.As(txErr, &richErr) {
			writeError(w, r, http.StatusConflict, richErr)
			return
		}
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(txErr, i18n.MsgUpdateSkillBundleSkillsFailed))
		return
	}

	results := make([]bundleSkillAddResult, 0, len(orderedBundles))
	totalAdded := 0
	for _, bundle := range orderedBundles {
		var finalCount int64
		model.DB(r.Context()).Model(&model.BundleSkill{}).Where("skill_bundle_id = ?", bundle.ID).Count(&finalCount)
		added := addedByBundle[bundle.ID]
		totalAdded += added
		results = append(results, bundleSkillAddResult{
			BundleID:   bundle.ID,
			BundleName: bundle.Name,
			Added:      added,
			SkillCount: int(finalCount),
		})
	}

	jsonOK(w, map[string]interface{}{
		"ok":             true,
		"added":          totalAdded,
		"bundle_results": results,
	})
}

// ── 收藏/取消收藏 ──────────────────────────────────────────────────

// HandleFavoriteSkill 收藏公共技能
// POST /admin/skills/favorite
func HandleFavoriteSkill(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	var req struct {
		Name        string `json:"name"`
		Slug        string `json:"slug"`
		Version     string `json:"version"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}

	if req.Name == "" || req.Slug == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgNameSlugCannotBeEmpty))
		return
	}

	skill := model.PublicSkill{
		Name:        req.Name,
		Slug:        req.Slug,
		Version:     req.Version,
		Description: req.Description,
	}
	if err := model.DB(r.Context()).Create(&skill).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgFavoriteSkillFailed))
		return
	}

	jsonOK(w, map[string]interface{}{"ok": true, "skill_id": skill.ID})
}

// HandleUnfavoriteSkill 取消收藏公共技能
// POST /admin/skills/unfavorite?id=xxx
func HandleUnfavoriteSkill(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		idStr = r.FormValue("id")
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMissingParamID))
		return
	}

	var skill model.PublicSkill
	if model.DB(r.Context()).First(&skill, id).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgSkillNotExist))
		return
	}

	model.DB(r.Context()).Delete(&skill)
	jsonOK(w, map[string]interface{}{"ok": true})
}

// HandleAdminFavoritedSkills 获取已收藏技能列表
// GET /admin/skills/favorited
func HandleAdminFavoritedSkills(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	page, pageSize := parsePagination(r)

	var total int64
	model.DB(r.Context()).Model(&model.PublicSkill{}).Count(&total)

	var skills []model.PublicSkill
	model.DB(r.Context()).Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&skills)

	if skills == nil {
		skills = []model.PublicSkill{}
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPages++
	}

	jsonOK(w, map[string]interface{}{
		"skills":      skills,
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": totalPages,
	})
}

// ── 默认技能包 SMH 同步 ─────────────────────────────────────────────

var smhTokenInitOnce sync.Once

// EnsureSMHTokenReady 确保 SMH Token Refresher 已启动（幂等）。
// 用于 StartDefaultBundleSMHSync 中 SMH 开通后主动初始化 Token。
func EnsureSMHTokenReady(ctx context.Context) {
	smhTokenInitOnce.Do(func() {
		smhConfig := model.GetSMHConfig(ctx)
		if !smhConfig.IsConfigured() {
			return
		}
		InitSMHTokenRefresher(ctx, smhConfig)
	})
}

// HandleUpdateSkillBundleVisibility 更新技能包可见范围
// POST /admin/skill-bundles/update-visibility?id=xxx
func HandleUpdateSkillBundleVisibility(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		idStr = r.FormValue("id")
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMissingParamID))
		return
	}

	// 解析可见范围参数
	visibilityType := strings.TrimSpace(r.FormValue("visibility_type"))
	if visibilityType == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgVisibilityTypeCannotBeEmpty))
		return
	}
	var groupIDs []uint
	if visibilityType == model.VisibilityGroup {
		gidStr := r.FormValue("group_ids")
		if gidStr != "" {
			for _, s := range strings.Split(gidStr, ",") {
				if gid, e := strconv.Atoi(strings.TrimSpace(s)); e == nil && gid > 0 {
					groupIDs = append(groupIDs, uint(gid))
				}
			}
		}
	}
	if err := validateVisibilityInput(r.Context(), visibilityType, groupIDs); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	txErr := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		var bundle model.SkillBundle
		if tx.First(&bundle, id).Error != nil {
			return fmt.Errorf("not_found")
		}
		// 切换为 all 时，自动禁用技能包（避免与其他 all 包互斥冲突）
		if visibilityType == model.VisibilityAll && bundle.VisibilityType != model.VisibilityAll && bundle.Enabled {
			if err := tx.Model(&bundle).Update("enabled", false).Error; err != nil {
				return hcommon.I18nRichError(err, i18n.MsgSBDisableBundleFailed)
			}
		}
		return model.SetSkillBundleVisibility(tx, bundle.ID, visibilityType, groupIDs)
	})

	if txErr != nil {
		if txErr.Error() == "not_found" {
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgSkillBundleNotFound))
			return
		}
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(txErr, i18n.MsgUpdateSkillBundleVisibilityFailed))
		return
	}

	jsonOK(w, map[string]interface{}{"ok": true})
}
