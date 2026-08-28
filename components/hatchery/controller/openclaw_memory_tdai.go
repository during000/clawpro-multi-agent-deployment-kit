package controller

import (
	"net/http"

	hcommon "hatchery/common"
	"hatchery/model"
)

// HandleOpenClawMemoryTDAIStatus 查询实例的记忆插件安装状态及全局开关。
// GET /openclaw/memory-tdai-status?id=xxx
func HandleOpenClawMemoryTDAIStatus(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		return
	}

	log := Logger(r.Context())

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	if err := checkInstanceSupportsMemory(r.Context(), instance); err != nil {
		writeError(w, r, http.StatusForbidden, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	config := model.GetSiteConfig(r.Context())

	var plugin model.MemoryTDAIPlugin
	status := model.MemoryTDAIPluginStatusNotInstalled
	var lastError string
	if err := model.DB(r.Context()).Where("instance_id = ?", instance.InstanceId).First(&plugin).Error; err != nil {
		log.Error("[MemoryTDAIStatus] 查询 plugin 行失败", "instance_id", instance.InstanceId, "error", err)
	} else {
		status = plugin.Status
		lastError = plugin.LastError
	}

	resp := map[string]interface{}{
		"memory_tdai_enable": config.MemoryTDAIEnable,
		"status":             status,
	}
	if lastError != "" {
		resp["last_error"] = lastError
	}
	jsonOK(w, resp)
}
