package controller

import (
	"net/http"
	"time"

	hcommon "hatchery/common"
	"hatchery/controller/usergroup"
	"hatchery/i18n"
	"hatchery/model"
)

type imageUpdateNoticeItem struct {
	ImageID      string     `json:"image_id"`
	ImageName    string     `json:"image_name"`
	AgentType    string     `json:"agent_type"`
	AgentVersion string     `json:"agent_version"`
	PublishedAt  *time.Time `json:"published_at,omitempty"`
}

// HandleUserImageUpdateNotices returns current official image update notices visible to the login user.
func HandleUserImageUpdateNotices(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	ids := officialImageIDs()
	if len(ids) == 0 {
		jsonOK(w, map[string]any{"items": []imageUpdateNoticeItem{}})
		return
	}
	var images []model.AIImage
	if err := model.DB(r.Context()).Where("update_notice_enabled = ? AND enabled = ? AND image_id IN ?", true, true, ids).
		Order("id desc").Find(&images).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgImgUpdQueryNotice))
		return
	}
	if len(images) == 0 {
		jsonOK(w, map[string]any{"items": []imageUpdateNoticeItem{}})
		return
	}

	allTypes := make([]string, 0, len(model.GetAllAgentTypes(r.Context())))
	for _, t := range model.GetAllAgentTypes(r.Context()) {
		if t != nil {
			allTypes = append(allTypes, t.Code)
		}
	}
	groupIDs, err := model.GetUserGroupIDs(r.Context(), user.ID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	ancestorSet := make(map[uint]struct{})
	for _, gid := range groupIDs {
		ancestors, err := usergroup.GetAncestorIDs(r.Context(), gid)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgUgDBQueryFailed))
			return
		}
		for _, id := range ancestors {
			ancestorSet[id] = struct{}{}
		}
	}
	allGroupIDs := make([]uint, 0, len(ancestorSet))
	for id := range ancestorSet {
		allGroupIDs = append(allGroupIDs, id)
	}
	visibleTypes, err := usergroup.ResolveImageTypes(r.Context(), allGroupIDs, allTypes)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgResolveImageVisibilityFail))
		return
	}
	visibleSet := make(map[string]struct{}, len(visibleTypes))
	for _, t := range visibleTypes {
		visibleSet[t] = struct{}{}
	}

	imageIDs := make([]string, 0, len(images))
	for _, img := range images {
		imageIDs = append(imageIDs, img.ImageId)
	}
	latestHistory, err := model.LatestImageHistoriesByImageID(r.Context(), imageIDs)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgImgUpdQueryHistoryFail))
		return
	}

	items := make([]imageUpdateNoticeItem, 0, len(images))
	for _, img := range images {
		agentType := model.NormalizeAgentType(img.AgentType)
		if _, ok := visibleSet[agentType]; !ok {
			continue
		}
		var publishedAt *time.Time
		if h, ok := latestHistory[img.ImageId]; ok {
			publishedAt = &h.PublishedAt
		}
		items = append(items, imageUpdateNoticeItem{
			ImageID:      img.ImageId,
			ImageName:    img.ImageName,
			AgentType:    agentType,
			AgentVersion: img.AgentVersion,
			PublishedAt:  publishedAt,
		})
	}
	jsonOK(w, map[string]any{"items": items})
}
