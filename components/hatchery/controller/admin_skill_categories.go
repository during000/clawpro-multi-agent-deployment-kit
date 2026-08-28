package controller

import (
	"log/slog"
	"net/http"
	"strconv"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// HandleAdminSkillCategories 查询技能分类列表（分页）
func HandleAdminSkillCategories(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	page, pageSize := parsePagination(r)
	slog.Info("查询技能分类列表", "page", page, "page_size", pageSize)

	var total int64
	if err := model.DB(r.Context()).Model(&model.SkillCategory{}).Count(&total).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryCategoryCountFailed))
		return
	}

	var categories []model.SkillCategory
	if err := model.DB(r.Context()).Order("id asc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&categories).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryCategoryListFailed))
		return
	}

	// 批量查询每个分类关联的技能数量
	type catCount struct {
		CategoryID uint
		Cnt        int64
	}
	var counts []catCount
	if len(categories) > 0 {
		catIDs := make([]uint, len(categories))
		for i, c := range categories {
			catIDs[i] = c.ID
		}
		if err := model.DB(r.Context()).Model(&model.SkillCategoryMapping{}).
			Select("skill_category_mappings.category_id, count(*) as cnt").
			Joins("JOIN skills ON skills.id = skill_category_mappings.skill_id AND skills.deleted_at IS NULL").
			Where("skill_category_mappings.category_id IN ?", catIDs).
			Group("skill_category_mappings.category_id").
			Scan(&counts).Error; err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryCategorySkillCountFailed))
			return
		}
	}
	countMap := make(map[uint]int64, len(counts))
	for _, c := range counts {
		countMap[c.CategoryID] = c.Cnt
	}

	// 构造响应，附加 skill_count
	type categoryResp struct {
		model.SkillCategory
		SkillCount int64 `json:"skill_count"`
	}
	resp := make([]categoryResp, len(categories))
	for i, c := range categories {
		resp[i] = categoryResp{
			SkillCategory: c,
			SkillCount:    countMap[c.ID],
		}
	}

	jsonOK(w, map[string]interface{}{
		"categories": resp,
		"page":       page,
		"page_size":  pageSize,
		"total":      total,
	})
}

// HandleCreateSkillCategory 创建技能分类
func HandleCreateSkillCategory(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	name := r.FormValue("name")
	slog.Info("开始创建技能分类", "name", name)
	if name == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgCategoryNameRequired))
		return
	}

	// 检查名称唯一
	var count int64
	if err := model.DB(r.Context()).Model(&model.SkillCategory{}).Where("name = ?", name).Count(&count).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryCategoryNameFailed))
		return
	}
	if count > 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgCategoryNameExists))
		return
	}

	cat := model.SkillCategory{
		Name:        name,
		Description: r.FormValue("description"),
	}
	if err := model.DB(r.Context()).Create(&cat).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgCreateCategoryFailed))
		return
	}

	slog.Info("技能分类创建成功", "id", cat.ID, "name", name)
	jsonOK(w, map[string]interface{}{"ok": true, "id": cat.ID})
}

// HandleUpdateSkillCategory 更新技能分类
func HandleUpdateSkillCategory(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgCategoryIDMustBeNumber))
		return
	}
	slog.Info("开始更新技能分类", "id", id)
	if id == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgCategoryIDRequired))
		return
	}

	var cat model.SkillCategory
	if model.DB(r.Context()).First(&cat, id).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgCategoryNotFound))
		return
	}

	updates := map[string]interface{}{}
	if name := r.FormValue("name"); name != "" {
		// 检查名称唯一（排除自身）
		var count int64
		if err := model.DB(r.Context()).Model(&model.SkillCategory{}).Where("name = ? AND id != ?", name, id).Count(&count).Error; err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryCategoryNameFailed))
			return
		}
		if count > 0 {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgCategoryNameExists))
			return
		}
		updates["name"] = name
	}
	if desc := r.FormValue("description"); desc != "" {
		updates["description"] = desc
	}

	if len(updates) > 0 {
		if err := model.DB(r.Context()).Model(&cat).Updates(updates).Error; err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgUpdateCategoryFailed))
			return
		}
	}

	slog.Info("技能分类更新成功", "id", id)
	jsonOK(w, map[string]interface{}{"ok": true})
}

// HandleDeleteSkillCategory 删除技能分类
func HandleDeleteSkillCategory(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgCategoryIDMustBeNumber))
		return
	}
	slog.Info("开始删除技能分类", "id", id)
	if id == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgCategoryIDRequired))
		return
	}

	var cat model.SkillCategory
	if model.DB(r.Context()).First(&cat, id).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgCategoryNotFound))
		return
	}

	// 清理分类与技能的关联关系
	if err := model.DB(r.Context()).Where("category_id = ?", id).Delete(&model.SkillCategoryMapping{}).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgCleanupCategoryMappingFailed))
		return
	}

	if err := model.DB(r.Context()).Delete(&cat).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgDeleteCategoryFailed))
		return
	}
	slog.Info("技能分类删除成功", "id", id, "name", cat.Name)
	jsonOK(w, map[string]interface{}{"ok": true})
}
