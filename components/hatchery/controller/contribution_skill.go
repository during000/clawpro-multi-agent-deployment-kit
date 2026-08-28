package controller

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm"
)

// ── 员工端：技能共建 ────────────────────────────────────────────────

// HandleContributeSkill 员工提交技能/新版本
// POST /openclaw/skills/contribute (multipart/form-data)
// 参数与 /admin/skills/create 一致，额外设置 status=pending_review, uploader_id=员工
func HandleContributeSkill(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	if !requireSMHEnabled(w, r) {
		return
	}

	// 先解析表单做字段/互斥校验，避免无谓的 zip 处理与软删清理
	if err := r.ParseMultipartForm(skillUploadMaxSize); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgRequestBodyTooLargeWithError, err))
		return
	}
	slug := r.FormValue("slug")
	name := r.FormValue("name")
	version := r.FormValue("version")
	if slug == "" || name == "" || version == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgPluginSlugNameVerRequired))
		return
	}
	if !isValidSlug(slug) {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgPluginInvalidSlug))
		return
	}
	if model.HasPendingRequest(r.Context(), model.ResourceTypeSkill, slug) {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillContributePendingExists))
		return
	}

	prep, upErr := prepareSkillUploadFromForm(r, user.ID)
	if upErr != nil {
		writeSkillUploadError(w, r, upErr)
		return
	}
	skill := prep.Skill
	slug, version = skill.Slug, skill.Version
	slog.Info("员工提交技能", "slug", slug, "name", skill.Name, "version", version, "user_id", user.ID)

	skill.Status = model.SkillStatusPendingReview
	skill.UploaderID = user.ID

	// ── 事务：创建 Skill + ReviewRequest ──
	tx := model.DB(r.Context()).Begin()

	if err := tx.Create(&skill).Error; err != nil {
		tx.Rollback()
		if isDuplicateKeyError(err) {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillVersionExist, slug, version))
			return
		}
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSkillCreateRecordFail))
		return
	}

	createSkillCategoryMappings(tx, skill.ID, r.FormValue("category_ids"))

	// 处理可见范围
	visType, visGroupIDs, visProjectIDs, hasScope, visErr := parseVisibilityParams(r)
	if visErr != nil {
		tx.Rollback()
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(visErr))
		return
	}
	if hasScope {
		if err := model.SetSkillVisibility(tx, skill.ID, visType, visGroupIDs); err != nil {
			tx.Rollback()
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSkillSetVisibilityFail))
			return
		}
		if err := model.ReplaceResourceProjectBindings(tx, model.ProjectConfigTypeSkill, skill.Slug, visProjectIDs); err != nil {
			tx.Rollback()
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSkillSetVisibilityFail))
			return
		}
	} else {
		if err := model.CopySkillVisibility(tx, slug, skill.ID); err != nil {
			tx.Rollback()
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSkillInheritVisibilityFail))
			return
		}
	}

	// 继承 distribute_count
	if err := model.InheritSkillDistributeCount(tx, slug, skill.ID); err != nil {
		slog.Error("继承 distribute_count 失败", "slug", slug, "version", version, "error", err)
	}

	// 创建 ReviewRequest
	reviewReq := model.ReviewRequest{
		RequesterID:  user.ID,
		ResourceType: model.ResourceTypeSkill,
		ResourceID:   skill.ID,
		ActionType:   model.ActionTypePublish,
		Slug:         slug,
		Status:       model.ReviewStatusPending,
	}
	if err := tx.Create(&reviewReq).Error; err != nil {
		tx.Rollback()
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSkillCreateRecordFail))
		return
	}

	// ── COS 上传（事务内，失败回滚）──
	if upErr := uploadSkillPackageToStorage(r.Context(), prep.ZipData, prep.CosZipKey, prep.CosDirKey, prep.SlugPrefix); upErr != nil {
		tx.Rollback()
		writeSkillUploadError(w, r, upErr)
		return
	}

	tx.Commit()
	slog.Info("员工技能提交成功", "slug", slug, "version", version, "skill_id", skill.ID, "request_id", reviewReq.ID, "user_id", user.ID)

	// 通知管理员
	notifyAdminsNewReviewRequest(r.Context(), user.Username, slug, i18n.T(r.Context(), i18n.MsgSkillContributeSuccess))

	scanSubmitted, scanSkipReason := maybeSubmitSkillSecurityScan(
		r, prep.ZipData, skill.ID, skill.Version, slug+"-"+version+".zip")

	jsonOK(w, map[string]interface{}{
		"ok":               true,
		"skill_id":         skill.ID,
		"request_id":       reviewReq.ID,
		"scan_submitted":   scanSubmitted,
		"scan_skip_reason": scanSkipReason,
	})
}

// HandleTakedownSkill 员工申请下架技能
// POST /openclaw/skills/takedown {"slug": "xxx", "reason": "xxx"}
func HandleTakedownSkill(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		return
	}

	var body struct {
		Slug   string `json:"slug"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgPluginRequestFormatErr, err))
		return
	}
	if body.Slug == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillSlugRequired))
		return
	}
	if body.Reason == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillTakedownReasonRequired))
		return
	}

	// 归属校验：该 slug 下存在「本人上传 + published」的版本即可申请
	var owned model.Skill
	if model.DB(r.Context()).Where("slug = ? AND status = ? AND uploader_id = ?",
		body.Slug, model.SkillStatusPublished, user.ID).First(&owned).Error != nil {
		// 区分：有 published 但非本人 → 403；完全无 published → 404
		var anyPublished int64
		model.DB(r.Context()).Model(&model.Skill{}).
			Where("slug = ? AND status = ?", body.Slug, model.SkillStatusPublished).
			Count(&anyPublished)
		if anyPublished > 0 {
			writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgSkillTakedownNotOwner))
			return
		}
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgSkillNotExist))
		return
	}

	// 互斥校验
	if model.HasPendingRequest(r.Context(), model.ResourceTypeSkill, body.Slug) {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSkillContributePendingExists))
		return
	}

	// resource_id 绑定最新 published 版本（下架语义按 slug，列表也只展示最新版）
	var latest model.Skill
	if model.DB(r.Context()).Where("slug = ? AND status = ?", body.Slug, model.SkillStatusPublished).
		Order("version_major DESC, version_minor DESC, version_patch DESC").
		First(&latest).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgSkillNotExist))
		return
	}

	// 创建 ReviewRequest
	reviewReq := model.ReviewRequest{
		RequesterID:  user.ID,
		ResourceType: model.ResourceTypeSkill,
		ResourceID:   latest.ID,
		ActionType:   model.ActionTypeTakedown,
		Slug:         body.Slug,
		Status:       model.ReviewStatusPending,
		Reason:       body.Reason,
	}
	if err := model.DB(r.Context()).Create(&reviewReq).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSkillCreateRecordFail))
		return
	}

	slog.Info("员工申请下架技能", "slug", body.Slug, "skill_id", latest.ID, "request_id", reviewReq.ID, "user_id", user.ID)

	// 通知管理员
	notifyAdminsNewReviewRequest(r.Context(), user.Username, body.Slug, i18n.T(r.Context(), i18n.MsgSkillTakedownSuccess))

	jsonOK(w, map[string]interface{}{
		"ok":         true,
		"request_id": reviewReq.ID,
	})
}

// HandleMyContributions 员工查看自己的申请列表（按技能聚合）
// GET /openclaw/skills/contributions?status=pending&action_type=publish&keyword=xxx&page=1&page_size=20
func HandleMyContributions(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		return
	}

	page, pageSize := parsePagination(r)

	// 查询该用户所有申请的 slug 去重列表，用于分页
	slugQuery := model.DB(r.Context()).Model(&model.ReviewRequest{}).
		Where("requester_id = ?", user.ID)

	if st := r.URL.Query().Get("status"); st != "" {
		slugQuery = slugQuery.Where("status = ?", st)
	}
	if at := r.URL.Query().Get("action_type"); at != "" {
		slugQuery = slugQuery.Where("action_type = ?", at)
	}
	if keyword := r.URL.Query().Get("keyword"); keyword != "" {
		like := "%" + keyword + "%"
		var matchedSlugs []string
		model.DB(r.Context()).Model(&model.Skill{}).
			Where("slug LIKE ? OR name LIKE ?", like, like).
			Pluck("DISTINCT slug", &matchedSlugs)
		if len(matchedSlugs) > 0 {
			slugQuery = slugQuery.Where("slug LIKE ? OR slug IN ?", like, matchedSlugs)
		} else {
			slugQuery = slugQuery.Where("slug LIKE ?", like)
		}
	}

	// 获取去重的 slug 列表（按最新申请时间排序），用于分页
	var slugRows []struct {
		Slug string
	}
	model.DB(r.Context()).Model(&model.ReviewRequest{}).
		Select("slug").
		Where("id IN (?)",
			slugQuery.Select("MAX(id)").Group("slug"),
		).
		Order("id DESC").
		Scan(&slugRows)

	total := len(slugRows)

	// 分页截取 slug
	start := (page - 1) * pageSize
	if start > len(slugRows) {
		start = len(slugRows)
	}
	end := start + pageSize
	if end > len(slugRows) {
		end = len(slugRows)
	}
	pageSlugs := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		pageSlugs = append(pageSlugs, slugRows[i].Slug)
	}

	// 查询当前页所有 slug 的申请单
	var allRequests []model.ReviewRequest
	if len(pageSlugs) > 0 {
		reqDB := model.DB(r.Context()).Model(&model.ReviewRequest{}).
			Where("requester_id = ? AND slug IN ?", user.ID, pageSlugs)
		if st := r.URL.Query().Get("status"); st != "" {
			reqDB = reqDB.Where("status = ?", st)
		}
		if at := r.URL.Query().Get("action_type"); at != "" {
			reqDB = reqDB.Where("action_type = ?", at)
		}
		reqDB.Order("id DESC").Find(&allRequests)
	}

	// 按 slug 分组
	reqGroupMap := make(map[string][]model.ReviewRequest)
	for _, req := range allRequests {
		reqGroupMap[req.Slug] = append(reqGroupMap[req.Slug], req)
	}

	// 批量查询技能信息（含已被软删除的）
	skillInfoMap := make(map[string]struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	})
	if len(pageSlugs) > 0 {
		var skills []model.Skill
		model.DB(r.Context()).Unscoped().
			Select("slug, name, status").
			Where("slug IN ?", pageSlugs).
			Find(&skills)
		for _, s := range skills {
			skillInfoMap[s.Slug] = struct {
				Name   string `json:"name"`
				Status string `json:"status"`
			}{Name: s.Name, Status: s.Status}
		}
	}

	// 查询 pending 总数
	var pendingTotal int64
	model.DB(r.Context()).Model(&model.ReviewRequest{}).
		Where("requester_id = ? AND status = ?", user.ID, model.ReviewStatusPending).
		Count(&pendingTotal)

	// 按 pageSlugs 顺序组装响应
	type groupResp struct {
		Slug     string                `json:"slug"`
		Name     string                `json:"name"`
		Status   string                `json:"status"`
		Requests []model.ReviewRequest `json:"requests"`
	}
	groups := make([]groupResp, 0, len(pageSlugs))
	for _, slug := range pageSlugs {
		info := skillInfoMap[slug]
		requests := reqGroupMap[slug]
		if requests == nil {
			requests = []model.ReviewRequest{}
		}
		groups = append(groups, groupResp{
			Slug:     slug,
			Name:     info.Name,
			Status:   info.Status,
			Requests: requests,
		})
	}

	jsonOK(w, map[string]interface{}{
		"skills":        groups,
		"total":         total,
		"pending_total": pendingTotal,
		"page":          page,
		"page_size":     pageSize,
	})
}

// HandleMyContributionDetail 员工查看申请详情
// GET /openclaw/skills/contributions/detail?id=123
func HandleMyContributionDetail(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		return
	}

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

	// 权限校验：申请人本人或管理员可查看
	if req.RequesterID != user.ID && user.Role != "admin" {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgReviewNotOwner))
		return
	}

	// 附带 Skill 信息
	var skill *model.Skill
	if req.ResourceType == model.ResourceTypeSkill && req.ResourceID > 0 {
		var s model.Skill
		if model.DB(r.Context()).Where("id = ?", req.ResourceID).First(&s).Error == nil {
			skill = &s
		}
	}

	jsonOK(w, map[string]interface{}{
		"request": req,
		"skill":   skill,
	})
}

// ── Skill 审核逻辑（被 contribution.go dispatch 调用）────────────────

// approveSkillContribution 审核通过 skill 类型的申请
func approveSkillContribution(r *http.Request, req *model.ReviewRequest) error {
	admin, _ := RequestUser(r)
	var reviewerID uint
	if admin != nil {
		reviewerID = admin.ID
	}
	now := time.Now()

	return model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		switch req.ActionType {
		case model.ActionTypePublish:
			var skill model.Skill
			if err := tx.Where("id = ?", req.ResourceID).First(&skill).Error; err != nil {
				return hcommon.I18nError(i18n.MsgReviewSkillNotExist)
			}
			// 再次校验 (slug, version) 唯一性（防并发）
			var conflict model.Skill
			if tx.Where("slug = ? AND version = ? AND id != ? AND status = ?",
				skill.Slug, skill.Version, skill.ID, model.SkillStatusPublished).First(&conflict).Error == nil {
				return hcommon.I18nError(i18n.MsgSkillVersionExist, skill.Slug, skill.Version)
			}
			// pending_review → published
			if err := tx.Model(&model.Skill{}).Where("id = ?", skill.ID).
				Update("status", model.SkillStatusPublished).Error; err != nil {
				return hcommon.I18nRichError(err, i18n.MsgSkillCreateRecordFail)
			}
		case model.ActionTypeTakedown:
			// 整 slug：所有 published → offline（对齐 /admin/skills/offline；用 req.Slug 兼容旧 resource_id）
			slug := req.Slug
			if slug == "" {
				var skill model.Skill
				if err := tx.Where("id = ?", req.ResourceID).First(&skill).Error; err != nil {
					return hcommon.I18nError(i18n.MsgReviewSkillNotExist)
				}
				slug = skill.Slug
			}
			if err := tx.Model(&model.Skill{}).
				Where("slug = ? AND status = ?", slug, model.SkillStatusPublished).
				Update("status", model.SkillStatusOffline).Error; err != nil {
				return hcommon.I18nRichError(err, i18n.MsgSkillCreateRecordFail)
			}
		}

		// 更新 ReviewRequest
		if err := tx.Model(&model.ReviewRequest{}).Where("id = ?", req.ID).
			Updates(map[string]interface{}{
				"status":      model.ReviewStatusApproved,
				"reviewer_id": reviewerID,
				"reviewed_at": now,
			}).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgSkillCreateRecordFail)
		}
		return nil
	})
}

// rejectSkillContribution 审核拒绝 skill 类型的申请
func rejectSkillContribution(r *http.Request, req *model.ReviewRequest, comment string) error {
	admin, _ := RequestUser(r)
	var reviewerID uint
	if admin != nil {
		reviewerID = admin.ID
	}
	now := time.Now()

	return model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		switch req.ActionType {
		case model.ActionTypePublish:
			// 软删除 Skill（审核拒绝 = 技能不会上架）
			var skill model.Skill
			if err := tx.Where("id = ?", req.ResourceID).First(&skill).Error; err != nil {
				return hcommon.I18nError(i18n.MsgReviewSkillNotExist)
			}
			if err := tx.Delete(&model.Skill{}, skill.ID).Error; err != nil {
				return hcommon.I18nRichError(err, i18n.MsgSkillCreateRecordFail)
			}
		case model.ActionTypeTakedown:
			// Skill 不变，仅更新 ReviewRequest
		}

		// 更新 ReviewRequest
		if err := tx.Model(&model.ReviewRequest{}).Where("id = ?", req.ID).
			Updates(map[string]interface{}{
				"status":         model.ReviewStatusRejected,
				"reviewer_id":    reviewerID,
				"reviewed_at":    now,
				"review_comment": comment,
			}).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgSkillCreateRecordFail)
		}
		return nil
	})
}

// ── 通知辅助 ────────────────────────────────────────────────────────

// notifyAdminsNewReviewRequest 通知所有管理员有新的审核申请
func notifyAdminsNewReviewRequest(ctx context.Context, requesterName, slug, actionDesc string) {
	var admins []model.User
	model.DB(ctx).Where("role = ?", "admin").Find(&admins)
	for _, admin := range admins {
		_ = model.CreateNotificationWithCategory(
			ctx, admin.ID, 0, "",
			"new_review_request",
			model.NotifCategoryNotice,
			i18n.T(ctx, i18n.NotifTitleNewReviewRequest),
			i18n.T(ctx, i18n.NotifMsgNewReviewRequest, requesterName, slug, actionDesc),
			nil,
		)
	}
}

// notifySkillReviewResult 通知申请人技能发布/下架申请的审核结果。
// 通知失败不回滚已完成的审核事务，仅记录错误供排查。
func notifySkillReviewResult(ctx context.Context, req *model.ReviewRequest, approved bool, comment string) {
	if req == nil || req.RequesterID == 0 {
		return
	}

	var notifyType, category, title, message string
	category = model.NotifCategoryNotice

	switch req.ActionType {
	case model.ActionTypePublish:
		if approved {
			notifyType = "skill_review_approved"
			category = model.NotifCategorySuccess
			title = i18n.T(ctx, i18n.NotifTitleSkillReviewApproved)
			message = i18n.T(ctx, i18n.NotifMsgSkillReviewApproved, req.Slug)
		} else {
			notifyType = "skill_review_rejected"
			title = i18n.T(ctx, i18n.NotifTitleSkillReviewRejected)
			message = i18n.T(ctx, i18n.NotifMsgSkillReviewRejected, req.Slug, comment)
		}
	case model.ActionTypeTakedown:
		if approved {
			notifyType = "skill_takedown_approved"
			category = model.NotifCategorySuccess
			title = i18n.T(ctx, i18n.NotifTitleSkillTakedownApproved)
			message = i18n.T(ctx, i18n.NotifMsgSkillTakedownApproved, req.Slug)
		} else {
			notifyType = "skill_takedown_rejected"
			title = i18n.T(ctx, i18n.NotifTitleSkillTakedownRejected)
			message = i18n.T(ctx, i18n.NotifMsgSkillTakedownRejected, req.Slug, comment)
		}
	default:
		slog.Warn("未知技能审核动作，跳过通知", "request_id", req.ID, "action_type", req.ActionType)
		return
	}

	if err := model.CreateNotificationWithCategory(
		ctx, req.RequesterID, 0, "",
		notifyType, category, title, message, nil,
	); err != nil {
		slog.Error("发送技能审核结果通知失败", "request_id", req.ID, "requester_id", req.RequesterID, "error", err)
	}
}

// HandleWithdrawContribution 员工撤回自己的审核申请
// POST /openclaw/skills/contributions/withdraw {"id": 123}
func HandleWithdrawContribution(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		return
	}

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

	// 权限校验：只有申请人本人可以撤回
	if req.RequesterID != user.ID {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgReviewNotOwner))
		return
	}
	// 状态校验：只有 pending 状态可以撤回
	if req.Status != model.ReviewStatusPending {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgReviewRequestNotPending))
		return
	}

	now := time.Now()
	txErr := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		switch req.ActionType {
		case model.ActionTypePublish:
			// publish 撤回 → 软删除 Skill（技能不会上架）
			if err := tx.Delete(&model.Skill{}, req.ResourceID).Error; err != nil {
				return hcommon.I18nRichError(err, i18n.MsgSkillCreateRecordFail)
			}
		case model.ActionTypeTakedown:
			// takedown 撤回 → Skill 不变，仅更新 ReviewRequest
		}

		// 更新 ReviewRequest 状态为 withdrawn
		if err := tx.Model(&model.ReviewRequest{}).Where("id = ?", req.ID).
			Updates(map[string]interface{}{
				"status":      model.ReviewStatusWithdrawn,
				"reviewed_at": now,
			}).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgSkillCreateRecordFail)
		}
		return nil
	})

	if txErr != nil {
		slog.Error("撤回申请失败", "request_id", req.ID, "error", txErr)
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(txErr))
		return
	}

	slog.Info("员工撤回申请", "request_id", req.ID, "action_type", req.ActionType, "slug", req.Slug, "user_id", user.ID)
	jsonOK(w, map[string]interface{}{"ok": true})
}
