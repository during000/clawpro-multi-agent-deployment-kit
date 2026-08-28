package controller

import (
	"log/slog"
	"net/http"
	"strconv"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm"
)

// HandleAdminPluginCategories 查询插件分类列表（分页）
func HandleAdminPluginCategories(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	page, pageSize := parsePagination(r)
	slog.Debug("查询插件分类列表", "page", page, "page_size", pageSize)

	var total int64
	if err := model.DB(r.Context()).Model(&model.PluginCategory{}).Count(&total).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryCategoryCountFailed))
		return
	}

	var categories []model.PluginCategory
	if err := model.DB(r.Context()).Order("id asc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&categories).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryCategoryListFailed))
		return
	}

	// 批量查询每个分类关联的插件数量
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
		if err := model.DB(r.Context()).Model(&model.PluginCategoryMapping{}).
			Select("plugin_category_mappings.category_id, count(*) as cnt").
			Joins("JOIN plugins ON plugins.id = plugin_category_mappings.plugin_id AND plugins.deleted_at IS NULL").
			Where("plugin_category_mappings.category_id IN ?", catIDs).
			Group("plugin_category_mappings.category_id").
			Scan(&counts).Error; err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryCategoryPluginCountFailed))
			return
		}
	}
	countMap := make(map[uint]int64, len(counts))
	for _, c := range counts {
		countMap[c.CategoryID] = c.Cnt
	}

	type categoryResp struct {
		model.PluginCategory
		PluginCount int64 `json:"plugin_count"`
	}
	resp := make([]categoryResp, len(categories))
	for i, c := range categories {
		resp[i] = categoryResp{
			PluginCategory: c,
			PluginCount:    countMap[c.ID],
		}
	}

	jsonOK(w, map[string]interface{}{
		"categories": resp,
		"page":       page,
		"page_size":  pageSize,
		"total":      total,
	})
}

// HandleCreatePluginCategory 创建插件分类
func HandleCreatePluginCategory(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)
	if !requireSMHEnabled(w, r) {
		return
	}

	name := r.FormValue("name")
	slog.Info("开始创建插件分类", "name", name)
	if name == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgCategoryNameRequired))
		return
	}
	if len(name) > 100 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgCategoryNameTooLong))
		return
	}

	var count int64
	if err := model.DB(r.Context()).Model(&model.PluginCategory{}).Where("name = ?", name).Count(&count).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryCategoryNameFailed))
		return
	}
	if count > 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgCategoryNameExists))
		return
	}

	cat := model.PluginCategory{
		Name:        name,
		Description: r.FormValue("description"),
	}
	if err := model.DB(r.Context()).Create(&cat).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgCreateCategoryFailed))
		return
	}

	slog.Info("插件分类创建成功", "id", cat.ID, "name", name)
	jsonOK(w, map[string]interface{}{"ok": true, "id": cat.ID})
}

// HandleUpdatePluginCategory 更新插件分类
func HandleUpdatePluginCategory(w http.ResponseWriter, r *http.Request) {
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
	slog.Info("开始更新插件分类", "id", id)
	if id == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgCategoryIDRequired))
		return
	}

	// 显式调用 ParseForm，确保 r.Form 被初始化
	if err := r.ParseForm(); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgParseFormFailed))
		return
	}

	var cat model.PluginCategory
	if model.DB(r.Context()).First(&cat, id).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgCategoryNotFound))
		return
	}

	updates := map[string]interface{}{}
	if name := r.FormValue("name"); name != "" {
		var count int64
		if err := model.DB(r.Context()).Model(&model.PluginCategory{}).Where("name = ? AND id != ?", name, id).Count(&count).Error; err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryCategoryNameFailed))
			return
		}
		if count > 0 {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgCategoryNameExists))
			return
		}
		updates["name"] = name
	}
	// 使用 r.Form 检查字段是否存在，允许清空为空字符串
	if r.Form != nil {
		if _, exists := r.Form["description"]; exists {
			updates["description"] = r.FormValue("description")
		}
	}

	if len(updates) > 0 {
		if err := model.DB(r.Context()).Model(&cat).Updates(updates).Error; err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgUpdateCategoryFailed))
			return
		}
	}

	slog.Info("插件分类更新成功", "id", id)
	jsonOK(w, map[string]interface{}{"ok": true})
}

// HandleDeletePluginCategory 删除插件分类
func HandleDeletePluginCategory(w http.ResponseWriter, r *http.Request) {
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
	slog.Info("开始删除插件分类", "id", id)
	if id == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgCategoryIDRequired))
		return
	}

	var cat model.PluginCategory
	if model.DB(r.Context()).First(&cat, id).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgCategoryNotFound))
		return
	}

	if err := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("category_id = ?", id).Delete(&model.PluginCategoryMapping{}).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgCleanupCategoryMappingFailed)
		}
		if err := tx.Delete(&cat).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgDeleteCategoryFailed)
		}
		return nil
	}); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	slog.Info("插件分类删除成功", "id", id, "name", cat.Name)
	jsonOK(w, map[string]interface{}{"ok": true})
}
