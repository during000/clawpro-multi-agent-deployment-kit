package controller

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// HandleFavoriteSkillSet 收藏一个公共技能包
// POST /admin/skillsets/favorite
func HandleFavoriteSkillSet(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	var req struct {
		Slug string `json:"slug"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}

	if req.Slug == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "slug"))
		return
	}

	skillset := model.PublicSkillSet{
		Slug: req.Slug,
	}
	if err := model.DB(r.Context()).Create(&skillset).Error; err != nil {
		slog.Error("收藏技能包失败", "slug", req.Slug, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgFavoriteSkillSetFailed))
		return
	}

	slog.Info("技能包收藏成功", "slug", req.Slug, "id", skillset.ID)
	jsonOK(w, map[string]interface{}{"ok": true, "skillset_id": skillset.ID})
}

// HandleUnfavoriteSkillSet 取消收藏公共技能包
// POST /admin/skillsets/unfavorite?id=xxx
func HandleUnfavoriteSkillSet(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
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
	slug := r.URL.Query().Get("slug")
	if slug == "" {
		slug = r.FormValue("slug")
	}
	if idStr == "" && slug == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgIDOrSlugRequired))
		return
	}
	if idStr != "" && slug != "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgIDAndSlugConflict))
		return
	}

	var skillset model.PublicSkillSet
	if idStr != "" {
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil || id == 0 {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidIDFormat))
			return
		}
		if model.DB(r.Context()).First(&skillset, id).Error != nil {
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgSkillsetNotFound))
			return
		}
	} else {
		if model.DB(r.Context()).Where("slug = ?", slug).First(&skillset).Error != nil {
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgSkillsetNotFound))
			return
		}
	}

	model.DB(r.Context()).Delete(&skillset)
	slog.Info("技能包取消收藏", "id", skillset.ID, "slug", skillset.Slug)
	jsonOK(w, map[string]interface{}{"ok": true})
}

// HandleAdminFavoritedSkillSets 获取已收藏技能包列表
// GET /admin/skillsets/favorited
func HandleAdminFavoritedSkillSets(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	page, pageSize := parsePagination(r)

	var total int64
	model.DB(r.Context()).Model(&model.PublicSkillSet{}).Count(&total)

	var skillsets []model.PublicSkillSet
	model.DB(r.Context()).Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&skillsets)

	if skillsets == nil {
		skillsets = []model.PublicSkillSet{}
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPages++
	}

	jsonOK(w, map[string]interface{}{
		"skillsets":   skillsets,
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": totalPages,
	})
}
