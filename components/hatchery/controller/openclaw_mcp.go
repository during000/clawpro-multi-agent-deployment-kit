package controller

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	hcommon "hatchery/common"
	"hatchery/controller/usergroup"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm/clause"
)

// mcpAsyncWG 可选的 WaitGroup，用于测试等待异步探测 goroutine 完成。
// 正式运行时为 nil（不阻塞），测试中赋值以避免 goroutine 泄漏导致 race。
var mcpAsyncWG *sync.WaitGroup

// ========== 端点 1: GET /openclaw/mcp/available — 企业可选 MCP 列表 ==========

func HandleUserMcpAvailable(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 版本门控
	if err := checkInstanceSupportsMcp(r.Context(), instance); err != nil {
		writeError(w, r, http.StatusForbidden, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	search := r.URL.Query().Get("q")

	// 查询企业已配置的 MCP 列表
	db := model.DB(r.Context()).Model(&model.McpServer{})
	if search != "" {
		escaped := strings.ReplaceAll(search, "%", "\\%")
		escaped = strings.ReplaceAll(escaped, "_", "\\_")
		like := "%" + escaped + "%"
		db = db.Where("name LIKE ? OR description LIKE ?", like, like)
	}

	var servers []model.McpServer
	db.Order("created_at DESC").Find(&servers)

	// 获取用户组（用于 visibility 过滤）
	userGroupIDs, _ := usergroup.GetUserAllGroupAndAncestorIDs(r.Context(), user.ID)

	// 获取该实例已安装的 MCP service_id 集合
	var installedServiceIDs []string
	model.DB(r.Context()).Model(&model.McpInstallation{}).
		Where("instance_id = ?", instance.ID).
		Pluck("service_id", &installedServiceIDs)
	installedSet := make(map[string]bool, len(installedServiceIDs))
	for _, sid := range installedServiceIDs {
		installedSet[sid] = true
	}

	type availableItem struct {
		ID            uint   `json:"id"`
		ServiceID     string `json:"service_id"`
		Name          string `json:"name"`
		Description   string `json:"description"`
		TransportType string `json:"transport_type"`
		ConfigJSON    string `json:"config_json"`
	}

	// 批量查询有默认值的托管字段（用于从 config_json 中移除）
	var allHostedKeys []model.McpHostedKey
	model.DB(r.Context()).Where("default_value != ''").Find(&allHostedKeys)
	// map[mcpID] → 有默认值的占位符 key 集合（如 "token"、"api-key"）
	defaultedKeys := make(map[uint]map[string]bool)
	for _, c := range allHostedKeys {
		if defaultedKeys[c.MCPID] == nil {
			defaultedKeys[c.MCPID] = make(map[string]bool)
		}
		defaultedKeys[c.MCPID][c.Key] = true
	}

	items := make([]availableItem, 0)
	for _, s := range servers {
		// 排除已安装的
		if installedSet[s.ServiceID] {
			continue
		}

		// visibility 过滤
		if s.VisibilityType == usergroup.VisibilityGroup {
			visible, err := usergroup.IsResourceVisible(r.Context(), model.ConfigTypeMCP, s.ID, s.VisibilityType, userGroupIDs)
			if err != nil || !visible {
				continue
			}
		}

		// 获取最新版本的 config_json
		var configJSON string
		var transportType string
		if s.LatestVersionID > 0 {
			var version model.McpVersion
			if err := model.DB(r.Context()).Where("id = ?", s.LatestVersionID).First(&version).Error; err == nil {
				configJSON = version.ConfigJSON
				transportType = version.TransportType
			}
		}
		if transportType == "" {
			transportType = s.TransportType
		}

		// 如果有默认值的托管字段，从 config_json 中移除（用户不需要填）
		displayConfig := configJSON
		if keys, ok := defaultedKeys[s.ID]; ok && len(keys) > 0 && configJSON != "" {
			displayConfig = removeDefaultedPlaceholders(configJSON, keys)
		}

		items = append(items, availableItem{
			ID:            s.ID,
			ServiceID:     s.ServiceID,
			Name:          s.Name,
			Description:   s.Description,
			TransportType: transportType,
			ConfigJSON:    displayConfig,
		})
	}

	jsonOK(w, map[string]interface{}{
		"items": items,
		"total": len(items),
	})
}

// ========== 端点 2: POST /openclaw/mcp/add — 添加 MCP 到实例（支持单个/批量） ==========

func HandleUserMcpAdd(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		return
	}

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	var req struct {
		InstanceIDs []uint `json:"instance_ids"`
		ServiceID   string `json:"service_id"`
		ConfigJSON  string `json:"config_json"`
		Restart     bool   `json:"restart"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}

	// 参数校验
	if len(req.InstanceIDs) == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInstanceIdsCannotBeEmpty))
		return
	}
	if len(req.InstanceIDs) > 50 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMcpMaxInstancesPerCall))
		return
	}
	if req.ServiceID == "" {
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgBadRequestParamRequired, "service_id"))
		return
	}
	if req.ConfigJSON == "" {
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgBadRequestParamRequired, "config_json"))
		return
	}

	// JSON 格式校验
	var configMap map[string]interface{}
	if err := json.Unmarshal([]byte(req.ConfigJSON), &configMap); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMcpConfigJsonInvalid, err))
		return
	}
	if _, hasURL := configMap["url"]; !hasURL {
		if _, hasCmd := configMap["command"]; !hasCmd {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMcpConfigJsonMissingURLOrCommand))
			return
		}
	}

	// 查询 MCP
	var server model.McpServer
	if err := model.DB(r.Context()).Where("service_id = ?", req.ServiceID).First(&server).Error; err != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgMcpNotFound))
		return
	}

	// 获取原始 config_json（用于比对）
	var originalConfigJSON string
	if server.LatestVersionID > 0 {
		var version model.McpVersion
		if err := model.DB(r.Context()).Where("id = ?", server.LatestVersionID).First(&version).Error; err == nil {
			originalConfigJSON = version.ConfigJSON
		}
	}

	// 凭据托管时：比对用户提交的 config 和原始 config，提取用户填写的占位符值
	var filledValues map[string]string
	if server.KeyHosted && originalConfigJSON != "" {
		var err error
		filledValues, err = DiffConfigPlaceholders(originalConfigJSON, req.ConfigJSON)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
			return
		}
	}

	// 已知 transport_type 做字段级强校验
	if knownTransportTypes[server.TransportType] {
		if err := validateConfigJSON(server.TransportType, configMap); err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
			return
		}
	}

	// visibility 校验
	if server.VisibilityType == usergroup.VisibilityGroup {
		userGroupIDs, _ := usergroup.GetUserAllGroupAndAncestorIDs(r.Context(), user.ID)
		visible, err := usergroup.IsResourceVisible(r.Context(), model.ConfigTypeMCP, server.ID, server.VisibilityType, userGroupIDs)
		if err != nil || !visible {
			writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgMcpNoAccess))
			return
		}
	}

	// 获取最新版本号
	var latestVersion string
	if server.LatestVersionID > 0 {
		var ver model.McpVersion
		if err := model.DB(r.Context()).Where("id = ?", server.LatestVersionID).First(&ver).Error; err == nil {
			latestVersion = ver.Version
		}
	}

	// 去重
	seen := make(map[uint]bool, len(req.InstanceIDs))
	var uniqueIDs []uint
	for _, id := range req.InstanceIDs {
		if !seen[id] {
			seen[id] = true
			uniqueIDs = append(uniqueIDs, id)
		}
	}

	// 查询所有实例（仅当前用户名下）
	var instances []model.Instance
	if err := model.DB(r.Context()).Where("id IN ? AND user_id = ?", uniqueIDs, user.ID).Find(&instances).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgQueryInstanceFailed))
		return
	}

	// 构造实例 map
	instanceMap := make(map[uint]*model.Instance, len(instances))
	for i := range instances {
		instanceMap[instances[i].ID] = &instances[i]
	}

	type resultItem struct {
		InstanceID uint   `json:"instance_id"`
		Status     string `json:"status"` // success / failed / skipped
		Error      string `json:"error,omitempty"`
	}

	results := make([]resultItem, 0, len(uniqueIDs))

	// 根据 key_hosted 分流处理
	var resolvedValues map[string]string
	var sharedConfigB64 string

	// 统一校验：用户提交的 config 不允许含有未替换的占位符
	if remaining := ExtractPlaceholders(req.ConfigJSON); len(remaining) > 0 {
		keys := make([]string, 0, len(remaining))
		for k := range remaining {
			keys = append(keys, k)
		}
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMcpPlaceholderUnfilled, strings.Join(keys, ", ")))
		return
	}

	if server.KeyHosted {
		// 凭据托管：从 filledValues 中提取托管字段值（用户填的 > 管理员默认值）
		hostedKeys := GetHostedKeys(r.Context(), server.ID)
		if len(hostedKeys) > 0 {
			var missing []string
			resolvedValues, missing = ResolveHostedValues(hostedKeys, filledValues)
			if len(missing) > 0 {
				writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMcpMissingHostedFields, strings.Join(missing, ", ")))
				return
			}
		}
	} else {
		// 非托管：直接下发
		sharedConfigB64 = base64.StdEncoding.EncodeToString([]byte(req.ConfigJSON))
	}

	for _, instID := range uniqueIDs {
		inst, ok := instanceMap[instID]
		if !ok {
			results = append(results, resultItem{
				InstanceID: instID,
				Status:     "skipped",
				Error:      i18n.T(r.Context(), i18n.MsgInstanceNotFoundOrNoPerm),
			})
			continue
		}

		if err := requireNoResourceAdjustment(inst); err != nil {
			results = append(results, resultItem{
				InstanceID: instID,
				Status:     "skipped",
				Error:      i18n.T(r.Context(), i18n.MsgOperationInProgress),
			})
			continue
		}
		// 版本门控
		if err := checkInstanceSupportsMcp(r.Context(), inst); err != nil {
			results = append(results, resultItem{
				InstanceID: instID,
				Status:     "skipped",
				Error:      err.Error(),
			})
			continue
		}

		// 状态检查：CVM 实际运行中 + Agent 就绪
		if inst.LastCVMState != "RUNNING" || inst.AgentReady != 1 {
			errMsg := i18n.T(r.Context(), i18n.MsgInstanceNotRunningWithState, inst.LastCVMState)
			if inst.LastCVMState == "RUNNING" && inst.AgentReady != 1 {
				errMsg = i18n.T(r.Context(), i18n.MsgInstanceStartingAgentNotReady)
			}
			results = append(results, resultItem{
				InstanceID: instID,
				Status:     "skipped",
				Error:      errMsg,
			})
			continue
		}

		if inst.InstanceId == "" {
			results = append(results, resultItem{
				InstanceID: instID,
				Status:     "skipped",
				Error:      i18n.T(r.Context(), i18n.MsgInstanceNoCVM),
			})
			continue
		}

		// UPSERT McpInstallation
		// 凭据托管模式下保存版本原始 config_json（由系统管理，不受前端提交影响）
		saveConfigJSON := req.ConfigJSON
		if server.KeyHosted && originalConfigJSON != "" {
			saveConfigJSON = originalConfigJSON
		}
		installation := model.McpInstallation{
			InstanceID:    inst.ID,
			MCPID:         server.ID,
			ServiceID:     server.ServiceID,
			Name:          server.Name,
			Version:       latestVersion,
			InstallStatus: model.McpInstalling,
			ConfigJSON:    saveConfigJSON,
			Source:        "user",
		}
		if err := model.DB(r.Context()).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "identifier"}, {Name: "instance_id"}, {Name: "service_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"mcp_id", "name", "version", "install_status", "config_json", "source", "error_message", "updated_at"}),
		}).Create(&installation).Error; err != nil {
			results = append(results, resultItem{
				InstanceID: instID,
				Status:     "failed",
				Error:      i18n.T(r.Context(), i18n.MsgMcpSaveConfigFailed),
			})
			continue
		}

		// TAT 下发
		var configB64 string
		if server.KeyHosted && len(resolvedValues) > 0 {
			// 凭据托管：保存 hosted_values 到 mcp_installations
			hostedJSON, _ := json.Marshal(resolvedValues)
			model.DB(r.Context()).Model(&model.McpInstallation{}).
				Where("instance_id = ? AND service_id = ?", inst.ID, server.ServiceID).
				Update("hosted_values", string(hostedJSON))
			// 构建部署 config（URL 替换为网关地址，托管 header 移除，注入 proxyToken）
			configB64 = base64.StdEncoding.EncodeToString([]byte(BuildDeployConfigJSON(r.Context(), originalConfigJSON, server.TransportType, server.ID, inst)))
		} else {
			configB64 = sharedConfigB64
		}
		params := map[string]string{
			"service_id":         server.ServiceID,
			"config_json_base64": configB64,
		}
		_, runErr := runScriptFn(r.Context(), inst.InstanceId, "mcp_upsert.sh", 120, inst.RuntimeUser, nil, params)

		if runErr != nil {
			model.DB(r.Context()).Model(&model.McpInstallation{}).
				Where("instance_id = ? AND service_id = ?", inst.ID, server.ServiceID).
				Updates(map[string]interface{}{
					"install_status": model.McpInstallFailed,
					"error_message":  runErr.Error(),
				})
			results = append(results, resultItem{
				InstanceID: instID,
				Status:     "failed",
				Error:      runErr.Error(),
			})
			continue
		}

		// 更新安装状态为成功
		model.DB(r.Context()).Model(&model.McpInstallation{}).
			Where("instance_id = ? AND service_id = ?", inst.ID, server.ServiceID).
			Updates(map[string]interface{}{
				"install_status": model.McpInstallSuccess,
				"error_message":  "",
			})

		// 重启（由用户决定是否重启，失败仅记日志）
		if req.Restart {
			if _, restartErr := runScriptFn(r.Context(), inst.InstanceId, "restart_gateway.sh", 60, inst.RuntimeUser, nil, nil); restartErr != nil {
				slog.Warn("restart_gateway.sh 执行失败", "error", restartErr, "instance_id", inst.ID)
			}
		}

		// 异步探测连通性
		var probeConfig string
		if server.KeyHosted && originalConfigJSON != "" {
			// 凭据托管：用原始模板 + 解析后的真实值构造探测 config，直连真实 MCP 地址
			probeConfig = buildProbeConfigJSON(originalConfigJSON, resolvedValues)
		} else {
			// 非托管：直接用用户提交的 config 探测
			probeConfig = req.ConfigJSON
		}
		if mcpAsyncWG != nil {
			mcpAsyncWG.Add(1)
		}
		go func(ctx context.Context, cfg string) {
			if mcpAsyncWG != nil {
				defer mcpAsyncWG.Done()
			}
			probeAndUpdate(ctx, inst.ID, server.ServiceID, server.TransportType, cfg)
		}(hcommon.DetachContext(r.Context()), probeConfig)

		results = append(results, resultItem{
			InstanceID: instID,
			Status:     "success",
		})
	}

	// 统计
	successCount := 0
	failedCount := 0
	skippedCount := 0
	for _, r := range results {
		switch r.Status {
		case "success":
			successCount++
		case "failed":
			failedCount++
		case "skipped":
			skippedCount++
		}
	}

	jsonOK(w, map[string]interface{}{
		"total":   len(results),
		"success": successCount,
		"failed":  failedCount,
		"skipped": skippedCount,
		"items":   results,
	})
}

// ========== 端点 3: GET /openclaw/mcp/list — 实例已添加的 MCP 列表 ==========

func HandleUserMcpList(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	search := r.URL.Query().Get("q")

	// 查询已安装的 MCP（Success 或 Failed，让用户能看到操作失败的记录并重试）
	db := model.DB(r.Context()).Where("instance_id = ? AND install_status IN ?", instance.ID, []int{model.McpInstallSuccess, model.McpInstallFailed})
	if search != "" {
		escaped := strings.ReplaceAll(search, "%", "\\%")
		escaped = strings.ReplaceAll(escaped, "_", "\\_")
		like := "%" + escaped + "%"
		db = db.Where("name LIKE ? OR service_id LIKE ?", like, like)
	}

	var installations []model.McpInstallation
	if err := db.Order("created_at DESC").Find(&installations).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgQueryMcpInstallationsFailed))
		return
	}
	mcpIDs := make([]uint, 0, len(installations))
	for _, inst := range installations {
		if inst.MCPID > 0 {
			mcpIDs = append(mcpIDs, inst.MCPID)
		}
	}
	serverMap := make(map[uint]model.McpServer)
	if len(mcpIDs) > 0 {
		var servers []model.McpServer
		model.DB(r.Context()).Where("id IN ?", mcpIDs).Find(&servers)
		for _, s := range servers {
			serverMap[s.ID] = s
		}
	}

	type listItem struct {
		ID               uint     `json:"id"`
		ServiceID        string   `json:"service_id"`
		Name             string   `json:"name"`
		Description      string   `json:"description"`
		TransportType    string   `json:"transport_type"`
		Source           string   `json:"source"`
		Version          string   `json:"version"`
		InstallStatus    int      `json:"install_status"`
		ErrorMessage     string   `json:"error_message"`
		ConfigJSON       string   `json:"config_json"`
		ConnectionStatus string   `json:"connection_status"`
		ConnectionError  string   `json:"connection_error"`
		Tools            []string `json:"tools"`
		ProbedAt         *string  `json:"probed_at"`
		UpdatedAt        string   `json:"updated_at"`
		KeyHosted        bool     `json:"key_hosted"`
	}

	items := make([]listItem, 0, len(installations))
	for _, inst := range installations {
		displayConfigJSON := inst.ConfigJSON
		var credHosted bool

		// 凭据托管：展示时反转 URL + 恢复占位符 + 移除敏感 header
		if s, ok := serverMap[inst.MCPID]; ok && s.KeyHosted {
			credHosted = true
			displayConfigJSON = BuildDisplayConfigJSON(r.Context(), inst.ConfigJSON, &s)
		}

		item := listItem{
			ID:               inst.ID,
			ServiceID:        inst.ServiceID,
			Name:             inst.Name,
			Source:           inst.Source,
			Version:          inst.Version,
			InstallStatus:    inst.InstallStatus,
			ErrorMessage:     inst.ErrorMessage,
			ConfigJSON:       displayConfigJSON,
			ConnectionStatus: inst.ConnectionStatus,
			ConnectionError:  inst.ConnectionError,
			UpdatedAt:        inst.UpdatedAt.Format("2006-01-02 15:04:05"),
			KeyHosted:        credHosted,
		}

		// 从 McpServer 获取 description 和 transport_type
		if s, ok := serverMap[inst.MCPID]; ok {
			item.Description = s.Description
			item.TransportType = s.TransportType
		}

		// 解析 tools_json
		item.Tools = []string{}
		if inst.ToolsJSON != "" {
			var tools []string
			if json.Unmarshal([]byte(inst.ToolsJSON), &tools) == nil {
				item.Tools = tools
			}
		}

		// probed_at
		if inst.ProbedAt != nil {
			t := inst.ProbedAt.Format("2006-01-02 15:04:05")
			item.ProbedAt = &t
		}

		items = append(items, item)
	}

	jsonOK(w, map[string]interface{}{
		"items": items,
		"total": len(items),
	})
}

// ========== 端点 4: POST /openclaw/mcp/refresh-status — 刷新连接状态 ==========

func HandleUserMcpRefreshStatus(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		return
	}

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	var req struct {
		ID         uint     `json:"id"`
		ServiceIDs []string `json:"service_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}

	instance, err := getInstanceForMcp(&w, r, user, req.ID)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if err := requireNoResourceAdjustment(instance); err != nil {
		writeAgentGuardError(w, r, err)
		return
	}

	// 实例级防重
	if !mcpProber.TryAcquireInstance(instance.ID) {
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgMcpInstanceRefreshing))
		return
	}
	defer mcpProber.ReleaseInstance(instance.ID)

	// 查询目标 MCP 安装记录
	db := model.DB(r.Context()).Where("instance_id = ? AND install_status = ?", instance.ID, model.McpInstallSuccess)
	if len(req.ServiceIDs) > 0 {
		db = db.Where("service_id IN ?", req.ServiceIDs)
	}

	var installations []model.McpInstallation
	if err := db.Find(&installations).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgQueryMcpInstallationsFailed))
		return
	}

	if len(installations) == 0 {
		jsonOK(w, map[string]interface{}{"items": []McpProbeResult{}})
		return
	}

	// 批量获取 transport_type
	mcpIDs := make([]uint, 0, len(installations))
	for _, inst := range installations {
		if inst.MCPID > 0 {
			mcpIDs = append(mcpIDs, inst.MCPID)
		}
	}
	transportMap := make(map[uint]string)
	if len(mcpIDs) > 0 {
		var servers []model.McpServer
		model.DB(r.Context()).Select("id, transport_type").Where("id IN ?", mcpIDs).Find(&servers)
		for _, s := range servers {
			transportMap[s.ID] = s.TransportType
		}
	}

	// 构造探测输入
	inputs := make([]McpProbeInput, 0, len(installations))
	for _, inst := range installations {
		transportType := transportMap[inst.MCPID]
		inputs = append(inputs, McpProbeInput{
			ServiceID:     inst.ServiceID,
			TransportType: transportType,
			ConfigJSON:    inst.ConfigJSON,
		})
	}

	// 执行探测
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	results := mcpProber.Probe(ctx, inputs)

	// 写入 DB
	now := time.Now()
	for i, result := range results {
		toolsJSON, _ := json.Marshal(result.Tools)
		model.DB(r.Context()).Model(&model.McpInstallation{}).
			Where("id = ?", installations[i].ID).
			Updates(map[string]interface{}{
				"connection_status": result.ConnectionStatus,
				"tools_json":        string(toolsJSON),
				"connection_error":  result.Error,
				"probed_at":         now,
			})
	}

	jsonOK(w, map[string]interface{}{"items": results})
}

// ========== 端点 5: POST /openclaw/mcp/update-config — 编辑 MCP 配置 ==========

func HandleUserMcpUpdateConfig(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		return
	}

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	var req struct {
		ID         uint   `json:"id"`
		ServiceID  string `json:"service_id"`
		ConfigJSON string `json:"config_json"`
		Restart    bool   `json:"restart"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}

	instance, err := getInstanceForMcp(&w, r, user, req.ID)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if err := requireNoResourceAdjustment(instance); err != nil {
		writeAgentGuardError(w, r, err)
		return
	}

	// 版本门控
	if err := checkInstanceSupportsMcp(r.Context(), instance); err != nil {
		writeError(w, r, http.StatusForbidden, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 状态准入：CVM 运行中 + Agent 就绪
	if instance.LastCVMState != "RUNNING" || instance.AgentReady != 1 {
		errMsg := hcommon.I18nError(i18n.MsgInstanceNotRunningWithState, instance.LastCVMState)
		if instance.LastCVMState == "RUNNING" && instance.AgentReady != 1 {
			errMsg = hcommon.I18nError(i18n.MsgInstanceStartingAgentNotReady)
		}
		writeError(w, r, http.StatusConflict, errMsg)
		return
	}

	// 参数校验
	if req.ServiceID == "" {
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgBadRequestParamRequired, "service_id"))
		return
	}
	if req.ConfigJSON == "" {
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgBadRequestParamRequired, "config_json"))
		return
	}

	// JSON 格式校验
	var configMap map[string]interface{}
	if err := json.Unmarshal([]byte(req.ConfigJSON), &configMap); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMcpConfigJsonInvalid, err))
		return
	}
	// 校验内层配置格式（必须包含 url 或 command）
	if _, hasURL := configMap["url"]; !hasURL {
		if _, hasCmd := configMap["command"]; !hasCmd {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMcpConfigJsonMissingURLOrCommand))
			return
		}
	}

	// 查找安装记录
	var installation model.McpInstallation
	if err := model.DB(r.Context()).Where("instance_id = ? AND service_id = ?", instance.ID, req.ServiceID).First(&installation).Error; err != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgMcpNotInstalled))
		return
	}

	// 凭据托管的 MCP 不允许用户修改 config_json（URL 和凭据由系统管理）
	var server model.McpServer
	if err := model.DB(r.Context()).Where("id = ?", installation.MCPID).First(&server).Error; err == nil {
		if server.KeyHosted {
			writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgMcpKeyHostedNoManualEdit))
			return
		}
	}

	// 已知 transport_type 做字段级强校验
	if err := model.DB(r.Context()).Where("id = ?", installation.MCPID).First(&server).Error; err == nil {
		if knownTransportTypes[server.TransportType] {
			if err := validateConfigJSON(server.TransportType, configMap); err != nil {
				writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
				return
			}
		}
	}

	// 更新 DB
	model.DB(r.Context()).Model(&installation).Updates(map[string]interface{}{
		"config_json": req.ConfigJSON,
		"source":      "user", // 用户编辑过的标记为 user 来源
	})

	// TAT 下发
	configB64 := base64.StdEncoding.EncodeToString([]byte(req.ConfigJSON))
	params := map[string]string{
		"service_id":         req.ServiceID,
		"config_json_base64": configB64,
	}

	_, runErr := runScriptFn(r.Context(), instance.InstanceId, "mcp_upsert.sh", 120, instance.RuntimeUser, nil, params)

	syncStatus := "success"
	restarted := false
	if runErr != nil {
		syncStatus = "failed"
		// 标记下发失败状态
		model.DB(r.Context()).Model(&installation).Updates(map[string]interface{}{
			"install_status": model.McpInstallFailed,
			"error_message":  i18n.T(r.Context(), i18n.MsgMcpConfigDeployFailed, runErr.Error()),
		})
	} else {
		// 下发成功，确保状态为 Success
		model.DB(r.Context()).Model(&installation).Updates(map[string]interface{}{
			"install_status": model.McpInstallSuccess,
			"error_message":  "",
		})
		if req.Restart {
			_, restartErr := runScriptFn(r.Context(), instance.InstanceId, "restart_gateway.sh", 60, instance.RuntimeUser, nil, nil)
			restarted = restartErr == nil
		}

		// 异步探测
		if mcpAsyncWG != nil {
			mcpAsyncWG.Add(1)
		}
		go func(ctx context.Context) {
			if mcpAsyncWG != nil {
				defer mcpAsyncWG.Done()
			}
			time.Sleep(3 * time.Second)
			transportType := ""
			var server model.McpServer
			if err := model.DB(ctx).Where("id = ?", installation.MCPID).First(&server).Error; err == nil {
				transportType = server.TransportType
			}
			probeAndUpdate(ctx, instance.ID, req.ServiceID, transportType, req.ConfigJSON)
		}(hcommon.DetachContext(r.Context()))
	}

	resp := map[string]interface{}{
		"ok":          runErr == nil,
		"sync_status": syncStatus,
		"restarted":   restarted,
	}
	if runErr != nil {
		resp["error"] = runErr.Error()
	}
	jsonOK(w, resp)
}

// ========== 端点 6: POST /openclaw/mcp/delete — 删除 MCP ==========

func HandleUserMcpDelete(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		return
	}

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	var req struct {
		ID        uint   `json:"id"`
		ServiceID string `json:"service_id"`
		Restart   bool   `json:"restart"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}

	instance, err := getInstanceForMcp(&w, r, user, req.ID)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if err := requireNoResourceAdjustment(instance); err != nil {
		writeAgentGuardError(w, r, err)
		return
	}

	// 状态准入：CVM 运行中 + Agent 就绪
	if instance.LastCVMState != "RUNNING" || instance.AgentReady != 1 {
		errMsg := hcommon.I18nError(i18n.MsgInstanceNotRunningWithState, instance.LastCVMState)
		if instance.LastCVMState == "RUNNING" && instance.AgentReady != 1 {
			errMsg = hcommon.I18nError(i18n.MsgInstanceStartingAgentNotReady)
		}
		writeError(w, r, http.StatusConflict, errMsg)
		return
	}

	if req.ServiceID == "" {
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgBadRequestParamRequired, "service_id"))
		return
	}

	// 查找安装记录
	var installation model.McpInstallation
	if err := model.DB(r.Context()).Where("instance_id = ? AND service_id = ?", instance.ID, req.ServiceID).First(&installation).Error; err != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgMcpNotInstalled))
		return
	}

	// TAT 删除
	params := map[string]string{
		"service_id": req.ServiceID,
	}
	_, runErr := runScriptFn(r.Context(), instance.InstanceId, "mcp_del.sh", 120, instance.RuntimeUser, nil, params)

	if runErr != nil {
		// 删除失败：不删 DB，标记失败状态
		slog.Warn("mcp_del.sh 执行失败", "error", runErr, "service_id", req.ServiceID, "instance_id", instance.ID)
		model.DB(r.Context()).Model(&installation).Updates(map[string]interface{}{
			"install_status": model.McpInstallFailed,
			"error_message":  i18n.T(r.Context(), i18n.MsgMcpDeleteFailedWithDetail, runErr.Error()),
		})
		jsonOK(w, map[string]interface{}{
			"ok":        false,
			"restarted": false,
			"error":     runErr.Error(),
		})
		return
	}

	// 删除成功：硬删除 DB 记录
	model.DB(r.Context()).Where("id = ?", installation.ID).Delete(&model.McpInstallation{})

	// 重启
	restarted := false
	if req.Restart {
		_, restartErr := runScriptFn(r.Context(), instance.InstanceId, "restart_gateway.sh", 60, instance.RuntimeUser, nil, nil)
		restarted = restartErr == nil
	}

	jsonOK(w, map[string]interface{}{
		"ok":        true,
		"restarted": restarted,
	})
}

// ========== 端点 7: POST /openclaw/mcp/toggle — 开启/关闭 MCP ==========

func HandleUserMcpToggle(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		return
	}

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	var req struct {
		ID        uint   `json:"id"`
		ServiceID string `json:"service_id"`
		Disabled  bool   `json:"disabled"`
		Restart   bool   `json:"restart"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}

	instance, err := getInstanceForMcp(&w, r, user, req.ID)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if err := requireNoResourceAdjustment(instance); err != nil {
		writeAgentGuardError(w, r, err)
		return
	}

	// 状态准入：CVM 运行中 + Agent 就绪
	if instance.LastCVMState != "RUNNING" || instance.AgentReady != 1 {
		errMsg := hcommon.I18nError(i18n.MsgInstanceNotRunningWithState, instance.LastCVMState)
		if instance.LastCVMState == "RUNNING" && instance.AgentReady != 1 {
			errMsg = hcommon.I18nError(i18n.MsgInstanceStartingAgentNotReady)
		}
		writeError(w, r, http.StatusConflict, errMsg)
		return
	}

	if req.ServiceID == "" {
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgBadRequestParamRequired, "service_id"))
		return
	}

	// 查找安装记录
	var installation model.McpInstallation
	if err := model.DB(r.Context()).Where("instance_id = ? AND service_id = ?", instance.ID, req.ServiceID).First(&installation).Error; err != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgMcpNotInstalled))
		return
	}

	// 修改 config_json 中的 disabled 字段
	var configMap map[string]interface{}
	if err := json.Unmarshal([]byte(installation.ConfigJSON), &configMap); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgMcpConfigParseFailed))
		return
	}

	if req.Disabled {
		configMap["disabled"] = true
	} else {
		delete(configMap, "disabled")
	}

	newConfigBytes, err := json.Marshal(configMap)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgMcpConfigMarshalFailed))
		return
	}
	newConfigJSON := string(newConfigBytes)

	// 更新 DB
	model.DB(r.Context()).Model(&installation).Update("config_json", newConfigJSON)

	// TAT 下发
	configB64 := base64.StdEncoding.EncodeToString(newConfigBytes)
	params := map[string]string{
		"service_id":         req.ServiceID,
		"config_json_base64": configB64,
	}
	_, runErr := runScriptFn(r.Context(), instance.InstanceId, "mcp_upsert.sh", 120, instance.RuntimeUser, nil, params)

	restarted := false
	if runErr != nil {
		// 标记下发失败状态
		model.DB(r.Context()).Model(&installation).Updates(map[string]interface{}{
			"install_status": model.McpInstallFailed,
			"error_message":  i18n.T(r.Context(), i18n.MsgMcpConfigDeployFailed, runErr.Error()),
		})
	} else {
		// 下发成功，确保状态为 Success
		model.DB(r.Context()).Model(&installation).Updates(map[string]interface{}{
			"install_status": model.McpInstallSuccess,
			"error_message":  "",
		})
		if req.Restart {
			_, restartErr := runScriptFn(r.Context(), instance.InstanceId, "restart_gateway.sh", 60, instance.RuntimeUser, nil, nil)
			restarted = restartErr == nil
		}
	}

	resp := map[string]interface{}{
		"ok":        runErr == nil,
		"restarted": restarted,
	}
	if runErr != nil {
		resp["error"] = runErr.Error()
	}
	jsonOK(w, resp)
}

// ========== 辅助函数 ==========

// getInstanceForMcp 通过解析后的 ID 直接查询实例，校验所有权并注入 instanceId header。
// 避免修改 r.URL.RawQuery，适用于 POST 请求中 body 传递 id 的场景。
func getInstanceForMcp(w *http.ResponseWriter, r *http.Request, user *model.User, id uint) (*model.Instance, error) {
	if id == 0 {
		return nil, hcommon.I18nError(i18n.MsgMissingParamID)
	}
	var instance model.Instance
	if model.DB(r.Context()).Where("id = ? AND user_id = ?", id, user.ID).First(&instance).Error != nil {
		return nil, hcommon.I18nError(i18n.MsgInstanceNotFound)
	}
	*w = WrapInstanceId(*w, instance.InstanceId)
	return &instance, nil
}

// probeAndUpdate 异步探测单个 MCP 并更新 DB
func probeAndUpdate(parentCtx context.Context, instanceID uint, serviceID, transportType, configJSON string) {
	ctx, cancel := context.WithTimeout(parentCtx, 20*time.Second)
	defer cancel()

	input := McpProbeInput{
		ServiceID:     serviceID,
		TransportType: transportType,
		ConfigJSON:    configJSON,
	}

	results := mcpProber.Probe(ctx, []McpProbeInput{input})
	if len(results) == 0 {
		return
	}

	result := results[0]
	toolsJSON, _ := json.Marshal(result.Tools)
	now := time.Now()

	if err := model.DB(ctx).Model(&model.McpInstallation{}).
		Where("instance_id = ? AND service_id = ?", instanceID, serviceID).
		Updates(map[string]interface{}{
			"connection_status": result.ConnectionStatus,
			"tools_json":        string(toolsJSON),
			"connection_error":  result.Error,
			"probed_at":         now,
		}).Error; err != nil {
		slog.Error("更新 MCP 探测结果失败", "error", err, "service_id", serviceID, "instance_id", instanceID)
	}
}
