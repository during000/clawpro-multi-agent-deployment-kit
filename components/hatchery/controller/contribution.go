package controller

import (
	"encoding/json"
	"net/http"
	"strconv"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// ── 管理员端：审核管理 ────────────────────────────────────────────────

// HandleAdminContributions 管理员查看所有审核申请列表
// GET /admin/contributions?resource_type=skill&action_type=publish&status=pending&page=1&page_size=20
func HandleAdminContributions(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	page, pageSize := parsePagination(r)

	db := model.DB(r.Context()).Model(&model.ReviewRequest{})

	if rt := r.URL.Query().Get("resource_type"); rt != "" {
		db = db.Where("resource_type = ?", rt)
	}
	if at := r.URL.Query().Get("action_type"); at != "" {
		db = db.Where("action_type = ?", at)
	}
	if st := r.URL.Query().Get("status"); st != "" {
		db = db.Where("status = ?", st)
	}
	// 按技能 slug 或名称搜索
	if keyword := r.URL.Query().Get("keyword"); keyword != "" {
		like := "%" + keyword + "%"
		var matchedSlugs []string
		_ = model.DB(r.Context()).Model(&model.Skill{}).
			Where("slug LIKE ? OR name LIKE ?", like, like).
			Pluck("DISTINCT slug", &matchedSlugs)
		if len(matchedSlugs) > 0 {
			db = db.Where("slug LIKE ? OR slug IN ?", like, matchedSlugs)
		} else {
			db = db.Where("slug LIKE ?", like)
		}
	}

	var total int64
	_ = db.Count(&total).Error

	var requests []model.ReviewRequest
	_ = db.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&requests).Error

	// 批量查询申请人 username
	requesterMap := make(map[uint]string)
	if len(requests) > 0 {
		ids := make([]uint, 0, len(requests))
		for _, req := range requests {
			if req.RequesterID > 0 {
				ids = append(ids, req.RequesterID)
			}
		}
		if len(ids) > 0 {
			var users []model.User
			_ = model.DB(r.Context()).Where("id IN ?", ids).Find(&users).Error
			for _, u := range users {
				requesterMap[u.ID] = u.Username
			}
		}
	}

	// 批量查询关联 Skill 名称
	skillNameMap := make(map[uint]string)
	if len(requests) > 0 {
		skillIDs := make([]uint, 0, len(requests))
		for _, req := range requests {
			if req.ResourceID > 0 {
				skillIDs = append(skillIDs, req.ResourceID)
			}
		}
		if len(skillIDs) > 0 {
			var skills []model.Skill
			_ = model.DB(r.Context()).Unscoped().Select("id, name").Where("id IN ?", skillIDs).Find(&skills).Error
			for _, s := range skills {
				skillNameMap[s.ID] = s.Name
			}
		}
	}

	// 查询 pending 总数
	var pendingTotal int64
	_ = model.DB(r.Context()).Model(&model.ReviewRequest{}).
		Where("status = ?", model.ReviewStatusPending).
		Count(&pendingTotal).Error

	type respItem struct {
		model.ReviewRequest
		RequesterName string `json:"requester_name"`
		SkillName     string `json:"skill_name"`
	}
	items := make([]respItem, len(requests))
	for i, req := range requests {
		items[i] = respItem{
			ReviewRequest: req,
			RequesterName: requesterMap[req.RequesterID],
			SkillName:     skillNameMap[req.ResourceID],
		}
	}

	jsonOK(w, map[string]interface{}{
		"requests":      items,
		"total":         total,
		"pending_total": pendingTotal,
		"page":          page,
		"page_size":     pageSize,
	})
}

// HandleAdminContributionDetail 管理员查看审核申请详情
// GET /admin/contributions/detail?id=123
func HandleAdminContributionDetail(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	id, err := strconv.ParseUint(r.URL.Query().Get("id"), 10, 64)
	if err != nil || id == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "id"))
		return
	}

	var req model.ReviewRequest
	if model.DB(r.Context()).Where("id = ?", id).First(&req).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgReviewRequestNotFound))
		return
	}

	// 查询申请人 username
	var requesterName string
	if req.RequesterID > 0 {
		var u model.User
		if model.DB(r.Context()).Select("username").Where("id = ?", req.RequesterID).First(&u).Error == nil {
			requesterName = u.Username
		}
	}

	// 如果是 skill 类型，附带 Skill 信息
	var skill *model.Skill
	if req.ResourceType == model.ResourceTypeSkill && req.ResourceID > 0 {
		var s model.Skill
		if model.DB(r.Context()).Where("id = ?", req.ResourceID).First(&s).Error == nil {
			skill = &s
		}
	}

	jsonOK(w, map[string]interface{}{
		"request":        req,
		"requester_name": requesterName,
		"skill":          skill,
	})
}

// HandleApproveContribution 管理员审核通过
// POST /admin/contributions/approve {"id": 123}
func HandleApproveContribution(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	var body struct {
		ID uint `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgPluginRequestFormatErr, err))
		return
	}
	if body.ID == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "id"))
		return
	}

	var req model.ReviewRequest
	if model.DB(r.Context()).Where("id = ?", body.ID).First(&req).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgReviewRequestNotFound))
		return
	}
	if req.Status != model.ReviewStatusPending {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgReviewRequestNotPending))
		return
	}

	// dispatch by resource_type
	var txErr error
	switch req.ResourceType {
	case model.ResourceTypeSkill:
		txErr = approveSkillContribution(r, &req)
	default:
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "resource_type"))
		return
	}

	if txErr != nil {
		Logger(r.Context()).Error("审核通过失败", "request_id", req.ID, "resource_type", req.ResourceType, "error", txErr)
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(txErr))
		return
	}

	if req.ResourceType == model.ResourceTypeSkill {
		notifySkillReviewResult(r.Context(), &req, true, "")
	}

	jsonOK(w, map[string]interface{}{"ok": true})
}

// HandleRejectContribution 管理员审核拒绝
// POST /admin/contributions/reject {"id": 123, "review_comment": "原因"}
func HandleRejectContribution(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	var body struct {
		ID            uint   `json:"id"`
		ReviewComment string `json:"review_comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgPluginRequestFormatErr, err))
		return
	}
	if body.ID == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "id"))
		return
	}
	if body.ReviewComment == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgReviewRejectCommentRequired))
		return
	}

	var req model.ReviewRequest
	if model.DB(r.Context()).Where("id = ?", body.ID).First(&req).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgReviewRequestNotFound))
		return
	}
	if req.Status != model.ReviewStatusPending {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgReviewRequestNotPending))
		return
	}

	// dispatch by resource_type
	var txErr error
	switch req.ResourceType {
	case model.ResourceTypeSkill:
		txErr = rejectSkillContribution(r, &req, body.ReviewComment)
	default:
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalid, "resource_type"))
		return
	}

	if txErr != nil {
		Logger(r.Context()).Error("审核拒绝失败", "request_id", req.ID, "resource_type", req.ResourceType, "error", txErr)
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(txErr))
		return
	}

	if req.ResourceType == model.ResourceTypeSkill {
		notifySkillReviewResult(r.Context(), &req, false, body.ReviewComment)
	}

	jsonOK(w, map[string]interface{}{"ok": true})
}
