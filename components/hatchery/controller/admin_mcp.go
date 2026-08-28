package controller

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"regexp"
	"strings"

	hcommon "hatchery/common"
	"hatchery/controller/usergroup"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm"
)

// mcpVersionRegex 版本号格式校验：x.y.z
var mcpVersionRegex = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// validateIPWhitelist 校验 IP 白名单格式，要求逗号分隔的 IP 或 CIDR，空字符串合法（表示不限制）
func validateIPWhitelist(raw string) error {
	if raw == "" {
		return nil
	}
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if strings.Contains(item, "/") {
			if _, _, err := net.ParseCIDR(item); err != nil {
				return hcommon.I18nError(i18n.MsgInvalidCIDRFormat, item)
			}
		} else {
			if net.ParseIP(item) == nil {
				return hcommon.I18nError(i18n.MsgInvalidIPAddress, item)
			}
		}
	}
	return nil
}

// ========== 端点 1: GET /admin/mcp — 列表 ==========

func HandleAdminMcpList(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	q := r.URL.Query()
	search := q.Get("q")
	transport := q.Get("transport")
	page, size := parsePagination(r)

	db := model.DB(r.Context()).Model(&model.McpServer{})

	if search != "" {
		escaped := strings.ReplaceAll(search, "%", "\\%")
		escaped = strings.ReplaceAll(escaped, "_", "\\_")
		like := "%" + escaped + "%"
		db = db.Where("name LIKE ? OR description LIKE ? OR service_id LIKE ?", like, like, like)
	}
	if transport != "" {
		db = db.Where("transport_type = ?", transport)
	}

	var total int64
	db.Count(&total)

	var servers []model.McpServer
	db.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&servers)

	type serverItem struct {
		ID                  uint                           `json:"id"`
		ServiceID           string                         `json:"service_id"`
		Name                string                         `json:"name"`
		Description         string                         `json:"description"`
		TransportType       string                         `json:"transport_type"`
		LatestVersion       string                         `json:"latest_version"`
		DistributionSummary map[string]interface{}         `json:"distribution_summary"`
		VisibilityType      string                         `json:"visibility_type"`
		VisibilityGroups    []usergroup.VisibilityGroupRef `json:"visibility_groups,omitempty"`
		CreatedAt           string                         `json:"created_at"`
		UpdatedAt           string                         `json:"updated_at"`
	}

	// 批量获取可见性绑定组
	groupServerIDs := make([]uint, 0)
	for _, s := range servers {
		if s.VisibilityType == usergroup.VisibilityGroup {
			groupServerIDs = append(groupServerIDs, s.ID)
		}
	}
	mcpBindingMap := usergroup.GetVisibilityGroupRefs(r.Context(), model.ConfigTypeMCP, groupServerIDs)

	items := make([]serverItem, 0, len(servers))
	for _, s := range servers {
		item := serverItem{
			ID:             s.ID,
			ServiceID:      s.ServiceID,
			Name:           s.Name,
			Description:    s.Description,
			TransportType:  s.TransportType,
			VisibilityType: s.VisibilityType,
			CreatedAt:      s.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:      s.UpdatedAt.Format("2006-01-02 15:04:05"),
		}

		// 可见性绑定组
		if s.VisibilityType == usergroup.VisibilityGroup {
			item.VisibilityGroups = mcpBindingMap[s.ID]
		}

		// 获取最新版本号
		if s.LatestVersionID > 0 {
			var latestVer model.McpVersion
			if err := model.DB(r.Context()).Where("id = ?", s.LatestVersionID).First(&latestVer).Error; err == nil {
				item.LatestVersion = latestVer.Version
			}
		}

		// 获取最近一次 task 的聚合信息
		var task model.McpDistributionTask
		if err := model.DB(r.Context()).Where("mcp_id = ?", s.ID).Order("created_at DESC").First(&task).Error; err == nil {
			item.DistributionSummary = map[string]interface{}{
				"total":   task.Total,
				"success": task.Success,
				"failed":  task.Failed,
				"pending": task.Total - task.Success - task.Failed,
			}
		}

		items = append(items, item)
	}

	jsonOK(w, map[string]interface{}{
		"total":     total,
		"page":      page,
		"page_size": size,
		"items":     items,
	})
}

// ========== 端点 2: POST /admin/mcp/create — 新增 MCP ==========

func HandleCreateMcp(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	var req struct {
		ServiceID      string            `json:"service_id"`
		Name           string            `json:"name"`
		Description    string            `json:"description"`
		TransportType  string            `json:"transport_type"`
		ConfigJSON     string            `json:"config_json"`
		UsageDocMD     string            `json:"usage_doc_md"`
		ToolDocMD      string            `json:"tool_doc_md"`
		KeyHosted      bool              `json:"key_hosted"`      // 是否开启凭据托管
		HostedDefaults map[string]string `json:"hosted_defaults"` // 托管字段默认值
		IPWhitelist    string            `json:"ip_whitelist"`    // IP 白名单（逗号分隔），仅 key_hosted 时生效
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}

	// 如果 config_json 为空
	if req.ConfigJSON == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMcpServiceConfigRequired))
		return
	}

	// 校验输入
	validResult, err := validateMCPInput(req.ServiceID, req.TransportType, req.ConfigJSON, true)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// name 默认 = service_id
	if req.Name == "" {
		req.Name = req.ServiceID
	}

	// 凭据托管：只有管理员显式开启时才检测占位符
	hasHosted := false
	if req.KeyHosted {
		if placeholders := ExtractPlaceholders(req.ConfigJSON); len(placeholders) > 0 {
			hasHosted = true
		} else {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMcpKeyHostedNeedPlaceholder))
			return
		}
	}

	// 校验 IP 白名单格式
	if err := validateIPWhitelist(req.IPWhitelist); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	username := getAdminUsername(r)

	var serverID uint
	var versionID uint
	txErr := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		server := model.McpServer{
			ServiceID:     req.ServiceID,
			Name:          req.Name,
			Description:   req.Description,
			TransportType: req.TransportType,
			CreatedBy:     username,
			KeyHosted:     hasHosted,
			IPWhitelist:   req.IPWhitelist,
		}
		if err := tx.Create(&server).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) || isDuplicateKeyError(err) {
				return hcommon.I18nError(i18n.MsgMcpServiceIDExists).WithPrefix("CONFLICT")
			}
			return hcommon.I18nRichError(err, i18n.MsgOperationFailed)
		}
		serverID = server.ID

		version := model.McpVersion{
			MCPID:         server.ID,
			Version:       "1.0.0",
			TransportType: req.TransportType,
			ConfigJSON:    req.ConfigJSON,
			UsageDocMD:    req.UsageDocMD,
			ToolDocMD:     req.ToolDocMD,
			CreatedBy:     username,
		}
		if err := tx.Create(&version).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgOperationFailed)
		}
		versionID = version.ID

		if err := tx.Model(&model.McpServer{}).Where("id = ?", server.ID).
			Update("latest_version_id", version.ID).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgOperationFailed)
		}

		return nil
	})

	if txErr != nil {
		if strings.HasPrefix(txErr.Error(), "CONFLICT:") {
			writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgMcpServiceIDExists))
			return
		}
		slog.Error("创建 MCP 失败", "error", txErr)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgCreateModelFailed))
		return
	}

	// 保存托管字段定义（事务外，非关键路径）
	if hasHosted {
		if _, err := SaveHostedKeys(r.Context(), serverID, req.ConfigJSON, req.HostedDefaults); err != nil {
			slog.Error("[MCP] 保存托管字段失败", "service_id", req.ServiceID, "error", err)
		}
	}

	resp := map[string]interface{}{
		"id":             serverID,
		"service_id":     req.ServiceID,
		"latest_version": "1.0.0",
		"version_id":     versionID,
	}
	if len(validResult.Warnings) > 0 {
		resp["warnings"] = validResult.Warnings
	}
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, resp)
}

// ========== 端点 3: POST /admin/mcp/meta — 修改元数据 ==========

func HandleUpdateMcpMeta(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	var req struct {
		ServiceID   string  `json:"service_id"`
		Name        *string `json:"name"`
		Description *string `json:"description"`
		IPWhitelist *string `json:"ip_whitelist"` // IP 白名单（逗号分隔），空字符串=取消限制
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}
	if req.ServiceID == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "service_id"))
		return
	}

	var server model.McpServer
	if err := model.DB(r.Context()).Where("service_id = ?", req.ServiceID).First(&server).Error; err != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgMcpNotFound))
		return
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = *req.Name
		server.Name = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
		server.Description = *req.Description
	}
	if req.IPWhitelist != nil {
		if err := validateIPWhitelist(*req.IPWhitelist); err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
			return
		}
		updates["ip_whitelist"] = *req.IPWhitelist
		server.IPWhitelist = *req.IPWhitelist
	}

	if len(updates) > 0 {
		if err := model.DB(r.Context()).Model(&server).Updates(updates).Error; err != nil {
			slog.Error("更新 MCP 元数据失败", "error", err)
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgUpdateFailed))
			return
		}
	}

	jsonOK(w, map[string]interface{}{
		"id":          server.ID,
		"service_id":  server.ServiceID,
		"name":        server.Name,
		"description": server.Description,
		"updated_at":  server.UpdatedAt,
	})
}

// ========== 端点 4: GET /admin/mcp/detail — 详情 ==========

func HandleAdminMcpDetail(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	serviceID := r.URL.Query().Get("service_id")
	if serviceID == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "service_id"))
		return
	}

	var server model.McpServer
	if err := model.DB(r.Context()).Where("service_id = ?", serviceID).First(&server).Error; err != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgMcpNotFound))
		return
	}

	versionStr := r.URL.Query().Get("version")
	var version model.McpVersion
	if versionStr != "" && versionStr != "latest" {
		if err := model.DB(r.Context()).Where("mcp_id = ? AND version = ?", server.ID, versionStr).First(&version).Error; err != nil {
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgMcpVersionNotFoundDetail, versionStr))
			return
		}
	} else if server.LatestVersionID > 0 {
		model.DB(r.Context()).Where("id = ?", server.LatestVersionID).First(&version)
	}

	var latestVersion string
	if server.LatestVersionID > 0 {
		var lv model.McpVersion
		if err := model.DB(r.Context()).Where("id = ?", server.LatestVersionID).First(&lv).Error; err == nil {
			latestVersion = lv.Version
		}
	}

	resp := map[string]interface{}{
		"id":           server.ID,
		"service_id":   server.ServiceID,
		"name":         server.Name,
		"description":  server.Description,
		"created_at":   server.CreatedAt,
		"created_by":   server.CreatedBy,
		"key_hosted":   server.KeyHosted,
		"ip_whitelist": server.IPWhitelist,
		"current_version": map[string]interface{}{
			"version":        version.Version,
			"transport_type": version.TransportType,
			"config_json":    version.ConfigJSON,
			"usage_doc_md":   version.UsageDocMD,
			"tool_doc_md":    version.ToolDocMD,
			"created_at":     version.CreatedAt,
			"created_by":     version.CreatedBy,
		},
		"latest_version": latestVersion,
	}

	// 凭据托管：返回所有托管字段（key → 默认值，无默认值则为空字符串）
	if server.KeyHosted {
		creds := GetHostedKeys(r.Context(), server.ID)
		hostedMap := make(map[string]string, len(creds))
		for _, c := range creds {
			hostedMap[c.Key] = c.DefaultValue // 无默认值时为 ""
		}
		resp["hosted_credentials"] = hostedMap
	}

	jsonOK(w, resp)
}

// ========== 端点 5: POST /admin/mcp/delete — 硬删 MCP ==========

func HandleDeleteMcp(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	var req struct {
		ServiceID string `json:"service_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}
	if req.ServiceID == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "service_id"))
		return
	}

	var server model.McpServer
	if err := model.DB(r.Context()).Where("service_id = ?", req.ServiceID).First(&server).Error; err != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgMcpNotFound))
		return
	}

	// 检查是否有正在运行的下发任务
	var runningCount int64
	model.DB(r.Context()).Model(&model.McpDistributionTask{}).
		Where("mcp_id = ? AND status = ?", server.ID, "running").
		Count(&runningCount)
	if runningCount > 0 {
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgMcpDistributionRunning))
		return
	}

	// 检查是否有 Installing 状态的 installation
	var installingCount int64
	model.DB(r.Context()).Model(&model.McpInstallation{}).
		Where("mcp_id = ? AND install_status = ?", server.ID, model.McpInstalling).
		Count(&installingCount)
	if installingCount > 0 {
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgMcpInstanceInstalling))
		return
	}

	// 事务内硬删：installations → versions → server
	txErr := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("mcp_id = ?", server.ID).Delete(&model.McpInstallation{}).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgOperationFailed)
		}
		if err := tx.Where("mcp_id = ?", server.ID).Delete(&model.McpVersion{}).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgOperationFailed)
		}
		if err := tx.Where("id = ?", server.ID).Delete(&model.McpServer{}).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgOperationFailed)
		}
		return nil
	})

	if txErr != nil {
		slog.Error("删除 MCP 失败", "error", txErr)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgDeleteModelFailed))
		return
	}

	jsonOK(w, map[string]interface{}{"message": i18n.T(r.Context(), i18n.MsgMcpDeleteSuccess)})
}

// ========== 端点 6: POST /admin/mcp/update — 新增版本 ==========

func HandleCreateMcpVersion(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	var req struct {
		ServiceID      string            `json:"service_id"`
		Version        string            `json:"version"`
		TransportType  string            `json:"transport_type"`
		ConfigJSON     string            `json:"config_json"`
		UsageDocMD     string            `json:"usage_doc_md"`
		ToolDocMD      string            `json:"tool_doc_md"`
		HostedDefaults map[string]string `json:"hosted_defaults"` // 托管字段默认值（可选，更新占位符定义时使用）
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}
	if req.ServiceID == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "service_id"))
		return
	}

	// 如果前端传入了版本号，校验格式
	if req.Version != "" && !mcpVersionRegex.MatchString(req.Version) {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMcpVersionFormatInvalid))
		return
	}

	validResult, err := validateMCPInput("", req.TransportType, req.ConfigJSON, false)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	var server model.McpServer
	if err := model.DB(r.Context()).Where("service_id = ?", req.ServiceID).First(&server).Error; err != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgMcpNotFound))
		return
	}

	// 凭据托管校验：新版本 config_json 必须包含占位符
	if server.KeyHosted {
		placeholders := ExtractPlaceholders(req.ConfigJSON)
		if len(placeholders) == 0 {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMcpKeyHostedRequirePlaceholder))
			return
		}
		// hosted_defaults 的 key 必须是有效占位符
		for k := range req.HostedDefaults {
			if _, ok := placeholders[k]; !ok {
				writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMcpHostedDefaultNotPlaceholder, k))
				return
			}
		}
	}

	username := getAdminUsername(r)

	var newVersion string
	txErr := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		if req.Version != "" {
			// 前端指定版本号：检查是否已存在
			var existing model.McpVersion
			if err := tx.Where("mcp_id = ? AND version = ?", server.ID, req.Version).First(&existing).Error; err == nil {
				return hcommon.I18nError(i18n.MsgMcpVersionAlreadyExists, req.Version)
			}
			// 前端指定版本号必须大于当前最大版本
			maxVer, err := model.MaxMcpVersion(tx, server.ID)
			if err != nil {
				return hcommon.I18nRichError(err, i18n.MsgOperationFailed)
			}
			if maxVer != "" && model.CompareSemver(req.Version, maxVer) <= 0 {
				return hcommon.I18nError(i18n.MsgMcpVersionMustBeGreaterMax, maxVer)
			}
			newVersion = req.Version
		} else {
			// 未传版本号：后端自增
			ver, err := model.NextMcpVersion(tx, server.ID)
			if err != nil {
				return hcommon.I18nRichError(err, i18n.MsgOperationFailed)
			}
			newVersion = ver
		}

		version := model.McpVersion{
			MCPID:         server.ID,
			Version:       newVersion,
			TransportType: req.TransportType,
			ConfigJSON:    req.ConfigJSON,
			UsageDocMD:    req.UsageDocMD,
			ToolDocMD:     req.ToolDocMD,
			CreatedBy:     username,
		}
		if err := tx.Create(&version).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgOperationFailed)
		}

		if err := tx.Model(&model.McpServer{}).Where("id = ?", server.ID).
			Updates(map[string]interface{}{
				"latest_version_id": version.ID,
				"transport_type":    req.TransportType,
			}).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgOperationFailed)
		}

		return nil
	})

	if txErr != nil {
		var re *hcommon.RichError
		if errors.As(txErr, &re) {
			if errors.Is(re, hcommon.I18nError(i18n.MsgMcpVersionAlreadyExists)) {
				writeError(w, r, http.StatusConflict, re)
			} else if errors.Is(re, hcommon.I18nError(i18n.MsgMcpVersionMustBeGreaterMax)) {
				writeError(w, r, http.StatusBadRequest, re)
			} else {
				writeError(w, r, http.StatusInternalServerError, re)
			}
			return
		}
		slog.Error("创建 MCP 版本失败", "error", txErr)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(txErr, i18n.MsgMcpCreateVersionFailed))
		return
	}

	resp := map[string]interface{}{
		"version": newVersion,
	}

	// 同步托管字段定义（仅 key_hosted，非关键路径，失败不阻塞）
	if server.KeyHosted {
		if _, err := SaveHostedKeys(r.Context(), server.ID, req.ConfigJSON, req.HostedDefaults); err != nil {
			slog.Warn("更新版本时同步托管字段失败", "error", err, "service_id", req.ServiceID)
		}
	}

	if len(validResult.Warnings) > 0 {
		resp["warnings"] = validResult.Warnings
	}
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, resp)
}

// ========== 端点 7: GET /admin/mcp/versions — 版本列表 ==========

func HandleAdminMcpVersions(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	serviceID := r.URL.Query().Get("service_id")
	if serviceID == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "service_id"))
		return
	}

	var server model.McpServer
	if err := model.DB(r.Context()).Where("service_id = ?", serviceID).First(&server).Error; err != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgMcpNotFound))
		return
	}

	var versions []model.McpVersion
	model.DB(r.Context()).Where("mcp_id = ?", server.ID).Order("id DESC").Find(&versions)

	type versionItem struct {
		Version       string `json:"version"`
		TransportType string `json:"transport_type"`
		CreatedAt     string `json:"created_at"`
		CreatedBy     string `json:"created_by"`
		IsLatest      bool   `json:"is_latest"`
	}

	items := make([]versionItem, 0, len(versions))
	for _, v := range versions {
		items = append(items, versionItem{
			Version:       v.Version,
			TransportType: v.TransportType,
			CreatedAt:     v.CreatedAt.Format("2006-01-02 15:04:05"),
			CreatedBy:     v.CreatedBy,
			IsLatest:      v.ID == server.LatestVersionID,
		})
	}

	jsonOK(w, map[string]interface{}{"versions": items})
}

// ========== 辅助函数 ==========

// getAdminUsername 从请求中获取管理员用户名
func getAdminUsername(r *http.Request) string {
	if user, err := RequestUser(r); user != nil && err == nil {
		return user.Username
	}
	return "admin"
}

// HandleAdminMcpVersionsRouter 路由分发：POST 创建版本（已迁移到 /admin/mcp/update），GET 版本列表
func HandleAdminMcpVersionsRouter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		HandleAdminMcpVersions(w, r)
	default:
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
	}
}
