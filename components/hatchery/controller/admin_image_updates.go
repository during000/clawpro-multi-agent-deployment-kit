package controller

import (
	"context"
	"encoding/json"
	"errors"
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

type imageHistoryItem struct {
	ID                           uint       `json:"id"`
	ImageID                      string     `json:"image_id"`
	ImageName                    string     `json:"image_name"`
	AgentType                    string     `json:"agent_type"`
	AgentVersion                 string     `json:"agent_version"`
	PublishedAt                  time.Time  `json:"published_at"`
	CreatedAt                    time.Time  `json:"created_at"`
	UpdatedAt                    time.Time  `json:"updated_at"`
	DeletedAt                    *time.Time `json:"deleted_at,omitempty"`
	CanSetNotice                 bool       `json:"can_set_notice"`
	UpdateNoticeEnabled          bool       `json:"update_notice_enabled"`
	ImageEnabled                 bool       `json:"image_enabled"`
	OutdatedRunningInstanceCount *int64     `json:"outdated_running_instance_count,omitempty"`
}

func officialImageIDs() []string {
	ids := make([]string, 0, len(hcommon.CandidateImages))
	for _, c := range hcommon.CandidateImages {
		ids = append(ids, c.ImageId)
	}
	return ids
}

func tenantOfficialImageIDs(ctx context.Context) ([]string, error) {
	ids := officialImageIDs()
	if len(ids) == 0 {
		return nil, nil
	}
	var visibleIDs []string
	if err := model.DB(ctx).Model(&model.AIImage{}).
		Where("image_id IN ?", ids).
		Distinct().Pluck("image_id", &visibleIDs).Error; err != nil {
		return nil, err
	}
	return visibleIDs, nil
}

func enabledOfficialImageIDSet(ctx context.Context, imageIDs []string) (map[string]struct{}, error) {
	result := make(map[string]struct{})
	if len(imageIDs) == 0 {
		return result, nil
	}
	type imageRow struct {
		ImageId   string
		AgentType string
	}
	var rows []imageRow
	if err := model.DB(ctx).Model(&model.AIImage{}).
		Select("image_id, agent_type").
		Where("image_id IN ? AND enabled = ?", imageIDs, true).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	disabled := make(map[string]struct{})
	for _, t := range model.GetDisabledAgentTypes(ctx) {
		disabled[t] = struct{}{}
	}
	for _, row := range rows {
		agentType := model.NormalizeAgentType(strings.TrimSpace(row.AgentType))
		if _, ok := disabled[agentType]; ok {
			continue
		}
		result[row.ImageId] = struct{}{}
	}
	return result, nil
}

func latestImageHistory(ctx context.Context, imageID string) (*model.ImageHistory, error) {
	var history model.ImageHistory
	if err := model.DBGlobal(ctx).Where("image_id = ?", imageID).
		Order("published_at desc, id desc").First(&history).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &history, nil
}

func imageHistoryLatestChanged(before, after *model.ImageHistory) bool {
	if before == nil && after == nil {
		return false
	}
	if before == nil || after == nil {
		return true
	}
	return before.ID != after.ID || before.AgentVersion != after.AgentVersion || !before.PublishedAt.Equal(after.PublishedAt)
}

func syncImageVersionFromLatestHistory(ctx context.Context, imageID string, before *model.ImageHistory) (bool, int64, error) {
	after, err := latestImageHistory(ctx, imageID)
	if err != nil {
		return false, 0, err
	}
	if !imageHistoryLatestChanged(before, after) {
		return false, 0, nil
	}
	candidate, ok := hcommon.GetCandidateImage(imageID)
	if !ok {
		return false, 0, hcommon.I18nError(i18n.MsgImgUpdOnlyOfficialSync)
	}
	agentType := model.NormalizeAgentType(candidate.AgentType)
	agentVersion := candidate.AgentVersion
	if after != nil && after.AgentVersion != "" {
		agentType = after.AgentType
		if agentType == "" {
			agentType = model.NormalizeAgentType(candidate.AgentType)
		}
		agentVersion = after.AgentVersion
	}
	result := model.DBGlobal(ctx).Model(&model.AIImage{}).
		Where("image_id = ?", imageID).
		Updates(map[string]any{
			"image_name":            candidate.ImageName,
			"agent_type":            agentType,
			"agent_version":         agentVersion,
			"update_notice_enabled": false,
		})
	return true, result.RowsAffected, result.Error
}

func parsePublishedAt(ctx context.Context, value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "published_at")
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02", value); err == nil {
		return t, nil
	}
	return time.Time{}, hcommon.I18nError(i18n.MsgImgUpdPublishedAtInvalid)
}

func parsePublishImageUpdateRequest(r *http.Request) (imageID, version string, publishedAt time.Time, err error) {
	var body struct {
		ImageID      string `json:"image_id"`
		AgentVersion string `json:"agent_version"`
		PublishedAt  string `json:"published_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return "", "", time.Time{}, hcommon.I18nError(i18n.MsgInvalidJSON)
	}
	imageID = strings.TrimSpace(body.ImageID)
	version = strings.TrimSpace(body.AgentVersion)
	publishedAt, err = parsePublishedAt(r.Context(), body.PublishedAt)
	return imageID, version, publishedAt, err
}

func parseUpdateImageNoticeRequest(r *http.Request) (imageID string, enabled bool, err error) {
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		var body struct {
			ImageID string `json:"image_id"`
			Enabled *bool  `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			return "", false, hcommon.I18nError(i18n.MsgInvalidJSON)
		}
		if body.Enabled == nil {
			return "", false, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "enabled")
		}
		return strings.TrimSpace(body.ImageID), *body.Enabled, nil
	}
	imageID = strings.TrimSpace(r.FormValue("image_id"))
	raw := strings.TrimSpace(r.FormValue("enabled"))
	if raw == "" {
		return "", false, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "enabled")
	}
	enabled, parseErr := strconv.ParseBool(raw)
	if parseErr != nil {
		return "", false, hcommon.I18nError(i18n.MsgParamMustBeBool, "enabled")
	}
	return imageID, enabled, nil
}

func parseUpdateImageHistoryRequest(r *http.Request) (id uint, version string, publishedAt *time.Time, err error) {
	var body struct {
		ID           uint   `json:"id"`
		AgentVersion string `json:"agent_version"`
		PublishedAt  string `json:"published_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return 0, "", nil, hcommon.I18nError(i18n.MsgInvalidJSON)
	}
	version = strings.TrimSpace(body.AgentVersion)
	if strings.TrimSpace(body.PublishedAt) != "" {
		t, parseErr := parsePublishedAt(r.Context(), body.PublishedAt)
		if parseErr != nil {
			return 0, "", nil, parseErr
		}
		publishedAt = &t
	}
	return body.ID, version, publishedAt, nil
}

func parseDeleteImageHistoryRequest(r *http.Request) (id uint, imageID string, hard bool, err error) {
	var body struct {
		ID      uint   `json:"id"`
		ImageID string `json:"image_id"`
		Hard    bool   `json:"hard"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return 0, "", false, hcommon.I18nError(i18n.MsgInvalidJSON)
	}
	return body.ID, strings.TrimSpace(body.ImageID), body.Hard, nil
}

func parseRestoreImageHistoryRequest(r *http.Request) (id uint, err error) {
	var body struct {
		ID uint `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return 0, hcommon.I18nError(i18n.MsgInvalidJSON)
	}
	return body.ID, nil
}

// HandlePublishImageUpdate publishes official image update history.
// Only the process admin-token may call this endpoint.
func HandlePublishImageUpdate(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !isAdminTokenRequest(r) {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgImgUpdAdminTokenOnlyPublish))
		return
	}

	imageID, version, publishedAt, err := parsePublishImageUpdateRequest(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if imageID == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "image_id"))
		return
	}
	if version == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgAgentVersionCannotBeEmpty))
		return
	}
	candidate, ok := hcommon.GetCandidateImage(imageID)
	if !ok {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgImgUpdOnlyOfficialPublish))
		return
	}
	agentType := model.NormalizeAgentType(candidate.AgentType)
	if err := model.ValidateAgentVersion(r.Context(), agentType, version); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	beforeLatest, dbErr := latestImageHistory(r.Context(), imageID)
	if dbErr != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgImgUpdQueryHistoryFail))
		return
	}

	history := model.ImageHistory{
		ImageId:      imageID,
		AgentType:    agentType,
		AgentVersion: version,
		PublishedAt:  publishedAt,
	}
	if err := model.DBGlobal(r.Context()).Create(&history).Error; err != nil {
		slog.ErrorContext(r.Context(), i18n.T(r.Context(), i18n.MsgImgUpdLogPublishHistoryFail), "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgImgUpdSaveHistoryFail))
		return
	}

	latestChanged, updatedImages, dbErr := syncImageVersionFromLatestHistory(r.Context(), imageID, beforeLatest)
	if dbErr != nil {
		slog.ErrorContext(r.Context(), i18n.T(r.Context(), i18n.MsgImgUpdLogSyncVersionsFail), "image_id", imageID, "error", dbErr)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgImgUpdSyncVersionFail))
		return
	}

	jsonOK(w, map[string]any{"ok": true, "id": history.ID, "latest_changed": latestChanged, "updated_images": updatedImages})
}

// HandleUpdateImageHistory updates one non-deleted official image update history item.
// Only the process admin-token may call this endpoint.
func HandleUpdateImageHistory(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !isAdminTokenRequest(r) {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgImgUpdAdminTokenOnlyEdit))
		return
	}

	id, version, publishedAt, err := parseUpdateImageHistoryRequest(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if id == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgIDCannotBeEmpty))
		return
	}
	if version == "" && publishedAt == nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgImgUpdVersionOrTimeRequired))
		return
	}

	var history model.ImageHistory
	if err := model.DBGlobal(r.Context()).Where("id = ?", id).First(&history).Error; err != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgImgUpdHistoryNotFound))
		return
	}
	candidate, ok := hcommon.GetCandidateImage(history.ImageId)
	if !ok {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgImgUpdOnlyOfficialEdit))
		return
	}
	beforeLatest, dbErr := latestImageHistory(r.Context(), history.ImageId)
	if dbErr != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgImgUpdQueryHistoryFail))
		return
	}

	updates := make(map[string]any)
	if version != "" {
		agentType := model.NormalizeAgentType(candidate.AgentType)
		if err := model.ValidateAgentVersion(r.Context(), agentType, version); err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
			return
		}
		updates["agent_type"] = agentType
		updates["agent_version"] = version
	}
	if publishedAt != nil {
		updates["published_at"] = *publishedAt
	}
	if err := model.DBGlobal(r.Context()).Model(&history).Updates(updates).Error; err != nil {
		slog.ErrorContext(r.Context(), i18n.T(r.Context(), i18n.MsgImgUpdEditHistoryFail), "id", id, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgImgUpdEditHistoryFail))
		return
	}
	latestChanged, updatedImages, dbErr := syncImageVersionFromLatestHistory(r.Context(), history.ImageId, beforeLatest)
	if dbErr != nil {
		slog.ErrorContext(r.Context(), i18n.T(r.Context(), i18n.MsgImgUpdLogSyncVersionsFail), "image_id", history.ImageId, "error", dbErr)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgImgUpdSyncVersionFail))
		return
	}
	jsonOK(w, map[string]any{"ok": true, "latest_changed": latestChanged, "updated_images": updatedImages})
}

// HandleDeleteImageHistory deletes one official image update history item.
// It soft-deletes by default; set hard=true to physically delete the record.
// If id is omitted, image_id deletes the latest non-deleted history of that image.
// Only the process admin-token may call this endpoint.
func HandleDeleteImageHistory(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !isAdminTokenRequest(r) {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgImgUpdAdminTokenOnlyDelete))
		return
	}

	id, imageID, hard, err := parseDeleteImageHistoryRequest(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	var history model.ImageHistory
	if id != 0 {
		q := model.DBGlobal(r.Context())
		if hard {
			q = q.Unscoped()
		}
		if err := q.Where("id = ?", id).First(&history).Error; err != nil {
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgImgUpdHistoryNotFound))
			return
		}
	} else {
		if imageID == "" {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgImgUpdIDOrImageIDRequired))
			return
		}
		if !hcommon.IsCandidateImage(imageID) {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgImgUpdOnlyOfficialDelete))
			return
		}
		if err := model.DBGlobal(r.Context()).Where("image_id = ?", imageID).
			Order("published_at desc, id desc").First(&history).Error; err != nil {
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgImgUpdHistoryNotFound))
			return
		}
	}
	if !hcommon.IsCandidateImage(history.ImageId) {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgImgUpdOnlyOfficialDelete))
		return
	}
	beforeLatest, dbErr := latestImageHistory(r.Context(), history.ImageId)
	if dbErr != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgImgUpdQueryHistoryFail))
		return
	}
	deleteDB := model.DBGlobal(r.Context())
	if hard {
		deleteDB = deleteDB.Unscoped()
	}
	if err := deleteDB.Delete(&history).Error; err != nil {
		slog.ErrorContext(r.Context(), i18n.T(r.Context(), i18n.MsgImgUpdDeleteHistoryFail), "id", history.ID, "hard", hard, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgImgUpdDeleteHistoryFail))
		return
	}
	latestChanged, updatedImages, dbErr := syncImageVersionFromLatestHistory(r.Context(), history.ImageId, beforeLatest)
	if dbErr != nil {
		slog.ErrorContext(r.Context(), i18n.T(r.Context(), i18n.MsgImgUpdLogSyncVersionsFail), "image_id", history.ImageId, "error", dbErr)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgImgUpdSyncVersionFail))
		return
	}
	jsonOK(w, map[string]any{"ok": true, "deleted_id": history.ID, "hard": hard, "latest_changed": latestChanged, "updated_images": updatedImages})
}

// HandleRestoreImageHistory restores one soft-deleted official image update history item.
// Only the process admin-token may call this endpoint.
func HandleRestoreImageHistory(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !isAdminTokenRequest(r) {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgImgUpdAdminTokenOnlyEnable))
		return
	}

	id, err := parseRestoreImageHistoryRequest(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if id == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgIDCannotBeEmpty))
		return
	}

	var history model.ImageHistory
	if err := model.DBGlobal(r.Context()).Unscoped().Where("id = ?", id).First(&history).Error; err != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgImgUpdHistoryNotFound))
		return
	}
	if !hcommon.IsCandidateImage(history.ImageId) {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgImgUpdOnlyOfficialEnable))
		return
	}

	beforeLatest, dbErr := latestImageHistory(r.Context(), history.ImageId)
	if dbErr != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgImgUpdQueryHistoryFail))
		return
	}
	if history.DeletedAt.Valid {
		if err := model.DBGlobal(r.Context()).Unscoped().Model(&history).Update("deleted_at", nil).Error; err != nil {
			slog.ErrorContext(r.Context(), i18n.T(r.Context(), i18n.MsgImgUpdEnableHistoryFail), "id", history.ID, "error", err)
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgImgUpdEnableHistoryFail))
			return
		}
	}
	latestChanged, updatedImages, dbErr := syncImageVersionFromLatestHistory(r.Context(), history.ImageId, beforeLatest)
	if dbErr != nil {
		slog.ErrorContext(r.Context(), i18n.T(r.Context(), i18n.MsgImgUpdLogSyncVersionsFail), "image_id", history.ImageId, "error", dbErr)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgImgUpdSyncVersionFail))
		return
	}
	jsonOK(w, map[string]any{"ok": true, "restored_id": history.ID, "latest_changed": latestChanged, "updated_images": updatedImages})
}

// HandleUpdateImageNotice lets a tenant admin enable/disable a current official image update notice in this tenant.
func HandleUpdateImageNotice(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	imageID, enabled, err := parseUpdateImageNoticeRequest(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if imageID == "" {
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgBadRequestParamRequired, "image_id"))
		return
	}
	if !hcommon.IsCandidateImage(imageID) {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgImgUpdOnlyOfficialNotice))
		return
	}

	var img model.AIImage
	if err := model.DB(r.Context()).Where("image_id = ?", imageID).First(&img).Error; err != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgImageNotFound))
		return
	}
	if enabled {
		var count int64
		if err := model.DBGlobal(r.Context()).Model(&model.ImageHistory{}).Where("image_id = ?", imageID).Count(&count).Error; err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgImgUpdQueryHistoryFail))
			return
		}
		if count == 0 {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgImgUpdNoNoticeForImage))
			return
		}
	}
	if err := model.DB(r.Context()).Model(&img).Update("update_notice_enabled", enabled).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgImgUpdToggleNoticeFail))
		return
	}
	jsonOK(w, map[string]any{"ok": true})
}

// HandleImageUpdateHistory returns official image update history.
func HandleImageUpdateHistory(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	page, pageSize := parsePagination(r)
	offset := (page - 1) * pageSize
	limit := pageSize

	q := model.DBGlobal(r.Context()).Model(&model.ImageHistory{})
	if includeDeleted, _ := strconv.ParseBool(strings.TrimSpace(r.URL.Query().Get("include_deleted"))); includeDeleted {
		q = q.Unscoped()
	}
	if enabledOnly, _ := strconv.ParseBool(strings.TrimSpace(r.URL.Query().Get("enabled_only"))); enabledOnly {
		enabledImageSet, err := enabledOfficialImageIDSet(r.Context(), officialImageIDs())
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgImgUpdQueryEnabledImage))
			return
		}
		if len(enabledImageSet) == 0 {
			jsonOK(w, map[string]any{"items": []imageHistoryItem{}, "total": int64(0), "limit": limit, "offset": offset})
			return
		}
		enabledImageIDs := make([]string, 0, len(enabledImageSet))
		for id := range enabledImageSet {
			enabledImageIDs = append(enabledImageIDs, id)
		}
		q = q.Where("image_id IN ?", enabledImageIDs)
	}
	if !isAdminTokenRequest(r) {
		visibleImageIDs, err := tenantOfficialImageIDs(r.Context())
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgImgUpdQueryTenantImage))
			return
		}
		if len(visibleImageIDs) == 0 {
			jsonOK(w, map[string]any{"items": []imageHistoryItem{}, "total": int64(0), "limit": limit, "offset": offset})
			return
		}
		q = q.Where("image_id IN ?", visibleImageIDs)
	}
	if imageID := strings.TrimSpace(r.URL.Query().Get("image_id")); imageID != "" {
		q = q.Where("image_id = ?", imageID)
	}
	if agentType := strings.TrimSpace(r.URL.Query().Get("agent_type")); agentType != "" {
		q = q.Where("agent_type = ?", agentType)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgImgUpdQueryHistoryFail))
		return
	}
	var rows []model.ImageHistory
	if err := q.Order("published_at desc, id desc").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgImgUpdQueryHistoryFail))
		return
	}
	imageIDs := make([]string, 0, len(rows))
	seenImageIDs := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		if _, ok := seenImageIDs[row.ImageId]; ok {
			continue
		}
		seenImageIDs[row.ImageId] = struct{}{}
		imageIDs = append(imageIDs, row.ImageId)
	}
	latestHistory, err := model.LatestImageHistoriesByImageID(r.Context(), imageIDs)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgImgUpdQueryLatestHistory))
		return
	}
	noticeEnabledImages := make(map[string]struct{}, len(imageIDs))
	enabledImages := make(map[string]struct{}, len(imageIDs))
	if len(imageIDs) > 0 {
		var noticeImageIDs []string
		if err := model.DB(r.Context()).Model(&model.AIImage{}).
			Where("image_id IN ? AND update_notice_enabled = ?", imageIDs, true).
			Distinct().Pluck("image_id", &noticeImageIDs).Error; err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgImgUpdQueryNoticeStatus))
			return
		}
		for _, id := range noticeImageIDs {
			noticeEnabledImages[id] = struct{}{}
		}

		enabledImages, err = enabledOfficialImageIDSet(r.Context(), imageIDs)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgImgUpdQueryEnabledStatus))
			return
		}
	}

	outdatedCounts := make(map[string]int64)
	for _, latest := range latestHistory {
		agentType := strings.TrimSpace(latest.AgentType)
		if agentType == "" {
			if candidate, ok := hcommon.GetCandidateImage(latest.ImageId); ok {
				agentType = model.NormalizeAgentType(candidate.AgentType)
			}
		}
		latestVersion := strings.TrimSpace(latest.AgentVersion)
		if agentType == "" || latestVersion == "" {
			continue
		}
		key := agentType + "\x00" + latestVersion
		if _, exists := outdatedCounts[key]; exists {
			continue
		}
		q := model.DB(r.Context()).Model(&model.Instance{}).
			Where("instance_id <> '' AND last_cvm_state = ? AND agent_ready = ? AND agent_version <> '' AND agent_version <> ?", "RUNNING", 1, latestVersion)
		if agentType == model.AgentTypeOpenClaw {
			q = q.Where("agent_type = ? OR agent_type = ''", agentType)
		} else {
			q = q.Where("agent_type = ?", agentType)
		}
		var count int64
		if err := q.Count(&count).Error; err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgImgUpdQueryOutdatedCount))
			return
		}
		outdatedCounts[key] = count
	}

	items := make([]imageHistoryItem, 0, len(rows))
	for _, row := range rows {
		var deletedAt *time.Time
		if row.DeletedAt.Valid {
			t := row.DeletedAt.Time
			deletedAt = &t
		}
		candidate, _ := hcommon.GetCandidateImage(row.ImageId)
		latest, hasLatest := latestHistory[row.ImageId]
		isLatest := !row.DeletedAt.Valid && hasLatest && row.ID == latest.ID
		canSetNotice := !row.DeletedAt.Valid && hasLatest && row.PublishedAt.Equal(latest.PublishedAt)
		_, noticeEnabled := noticeEnabledImages[row.ImageId]
		_, imageEnabled := enabledImages[row.ImageId]
		var outdatedCount *int64
		if isLatest {
			agentType := strings.TrimSpace(row.AgentType)
			if agentType == "" {
				agentType = model.NormalizeAgentType(candidate.AgentType)
			}
			count := outdatedCounts[agentType+"\x00"+strings.TrimSpace(row.AgentVersion)]
			outdatedCount = &count
		}
		items = append(items, imageHistoryItem{
			ID:                           row.ID,
			ImageID:                      row.ImageId,
			ImageName:                    candidate.ImageName,
			AgentType:                    row.AgentType,
			AgentVersion:                 row.AgentVersion,
			PublishedAt:                  row.PublishedAt,
			CreatedAt:                    row.CreatedAt,
			UpdatedAt:                    row.UpdatedAt,
			DeletedAt:                    deletedAt,
			CanSetNotice:                 canSetNotice,
			UpdateNoticeEnabled:          isLatest && noticeEnabled,
			ImageEnabled:                 imageEnabled,
			OutdatedRunningInstanceCount: outdatedCount,
		})
	}
	jsonOK(w, map[string]any{"items": items, "total": total, "limit": limit, "offset": offset})
}
