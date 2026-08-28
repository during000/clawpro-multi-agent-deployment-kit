package controller

import (
	"encoding/json"
	"net/http"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

type memoryTDAIConfigResponse struct {
	MemoryTDAIEnable            bool             `json:"memory_tdai_enable"`
	MemoryTDAISupportedVersions []string         `json:"memory_tdai_supported_versions"`
	Stats                       map[string]int64 `json:"stats"`
}

type updateMemoryTDAIConfigRequest struct {
	MemoryTDAIEnable *bool `json:"memory_tdai_enable"`
}

// HandleAdminMemoryTDAIConfig GET /admin/memory-tdai/config
func HandleAdminMemoryTDAIConfig(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, hcommon.I18nError(i18n.MsgOnlyGetMethod))
		return
	}

	cfg := model.GetSiteConfig(r.Context())
	stats := map[string]int64{}
	var rows []struct {
		Status string
		Count  int64
	}
	if err := model.DB(r.Context()).Model(&model.MemoryTDAIPlugin{}).
		Select("status, count(*) as count").
		Group("status").
		Find(&rows).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryMemoryTDAIStatsFailed))
		return
	}
	for _, row := range rows {
		stats[row.Status] = row.Count
	}

	_, supported, err := model.NormalizeMemoryTDAISupportedVersions(cfg.MemoryTDAISupportedVersions)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	jsonOK(w, map[string]interface{}{
		"config": memoryTDAIConfigResponse{
			MemoryTDAIEnable:            cfg.MemoryTDAIEnable,
			MemoryTDAISupportedVersions: supported,
			Stats:                       stats,
		},
	})
}

// HandleAdminUpdateMemoryTDAIConfig PUT /admin/memory-tdai/config
// 修改全局记忆开关。开关仅影响增量实例：
//   - 开启：新建实例默认启用 Free 版记忆插件
//   - 关闭：新建实例默认不启用记忆插件（OFF）
//
// 不影响存量实例，存量实例的记忆计划由用户在实例详情页自行切换。
func HandleAdminUpdateMemoryTDAIConfig(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}
	if r.Method != http.MethodPut {
		writeError(w, r, http.StatusMethodNotAllowed, hcommon.I18nError(i18n.MsgOnlyPutMethodSupported))
		return
	}

	var req updateMemoryTDAIConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgBadRequest))
		return
	}
	if req.MemoryTDAIEnable == nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMemoryTDAIEnableRequired))
		return
	}

	newEnable := *req.MemoryTDAIEnable

	if err := model.UpdateSiteConfig(r.Context(), map[string]interface{}{"memory_tdai_enable": newEnable}); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgUpdateMemoryTDAIFailed))
		return
	}

	jsonOK(w, map[string]interface{}{
		"ok":                 true,
		"memory_tdai_enable": newEnable,
	})
}
