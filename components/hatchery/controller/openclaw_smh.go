package controller

import (
	"net/http"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// HandleOpenClawSMHStatus 查询个人空间功能状态及实例是否已绑定
// GET /openclaw/smh-status?id=xxx
func HandleOpenClawSMHStatus(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgInstanceNotFound))
		return
	}

	config := model.GetSiteConfig(r.Context())
	enabled := config.SMHEnabled == 1

	var hasSpace bool
	if enabled {
		var count int64
		model.DB(r.Context()).Model(&model.SMHPersonalSpace{}).
			Where("instance_id = ? AND to_be_deleted_at IS NULL", instance.ID).
			Count(&count)
		hasSpace = count > 0
	}

	jsonOK(w, map[string]interface{}{
		"enabled":   enabled,
		"has_space": hasSpace,
	})
}

// HandleOpenClawSMHToken 获取实例对应个人空间的临时访问 Token
// GET /openclaw/smh-token?id=xxx
func HandleOpenClawSMHToken(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	if !requireSMHEnabled(w, r) {
		return
	}

	instance, richErr := getInstanceByID(&w, r, user)
	if richErr != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgInstanceNotFound))
		return
	}

	var space model.SMHPersonalSpace
	if model.DB(r.Context()).Where("instance_id = ? AND to_be_deleted_at IS NULL", instance.ID).First(&space).Error != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInstanceNoSMH))
		return
	}

	smhConfig := model.GetSMHConfig(r.Context())

	accessToken, expiresAt, _, err := ensurePersonalSpaceToken(r.Context(), space.SpaceId)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgGetSMHTokenFailed))
		return
	}

	jsonOK(w, map[string]interface{}{
		"token":      accessToken,
		"space_id":   space.SpaceId,
		"library_id": smhConfig.LibraryId,
		"endpoint":   smhConfig.Endpoint,
		"expires_at": expiresAt.UnixMilli(),
	})
}
