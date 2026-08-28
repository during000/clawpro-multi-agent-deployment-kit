package controller

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

var pluginNameRe = regexp.MustCompile(`^@?[\w.\-]+(\/[\w.\-]+)?$`)

func HandleAddPlugin(w http.ResponseWriter, r *http.Request) {
	handleAddPlugin(w, r, defaultStatusResolver)
}

func handleAddPlugin(w http.ResponseWriter, r *http.Request, resolver instanceStatusResolver) {
	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		return
	}

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 【关键防护】校验实例是否支持插件安装
	if err := checkInstanceSupportsPlugin(r.Context(), instance); err != nil {
		writeError(w, r, http.StatusForbidden, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 本地实例：插件安装依赖 CVM Agent 下发，本地 agent 不支持。
	if rejectLocalOrWrite(w, r, instance) {
		return
	}
	// 状态准入：仅 running 状态允许安装插件
	if _, err := requireInstanceRunning(r.Context(), instance, resolver); err != nil {
		writeAgentGuardError(w, r, err)
		return
	}

	plugin := r.FormValue("plugin")
	if plugin == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "plugin"))
		return
	}
	if !pluginNameRe.MatchString(plugin) {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidPluginName))
		return
	}

	// 微信插件版本从环境变量获取，拼接到插件名后面
	var version string
	if strings.Contains(plugin, "openclaw-weixin") {
		version = os.Getenv("WEIXIN_VERSION")
	}
	installTarget := plugin
	if strings.TrimSpace(version) != "" {
		installTarget = plugin + "@" + version
	}
	params := map[string]string{"plugin": plugin, "install_target": installTarget}
	_, err = RunScript(r.Context(), instance.InstanceId, "add_plugin.sh", 120, instance.RuntimeUser, nil, params)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	jsonOK(w, i18n.T(r.Context(), i18n.MsgPluginInstallSuccess))
}

// ── 插件安装核心逻辑（与 openclaw_skill.go 对称） ─────────────────

// batchPluginInstallResult 表示 batch_install_plugins_from_smh.sh 输出的 JSON 结构
type batchPluginInstallResult struct {
	Results []struct {
		Slug    string `json:"slug"`
		Version string `json:"version"`
		Status  string `json:"status"`
		Message string `json:"message"`
	} `json:"results"`
	Summary struct {
		Total   int `json:"total"`
		Success int `json:"success"`
		Failed  int `json:"failed"`
	} `json:"summary"`
}

// createPluginInstallTasks 为实例创建插件安装任务记录。
// 查询角色关联的插件（优先级高）和启用的插件包中的插件，合并去重后写入 plugin_installations 表。
func createPluginInstallTasks(ctx context.Context, instanceID uint, roleID uint) {
	// 【关键防护】获取实例的 agent_type，非插件支持类型跳过
	var instance model.Instance
	if err := model.DB(ctx).Select("agent_type").First(&instance, instanceID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			slog.Warn("[PluginInstall] 实例不存在，跳过插件安装", "instance_id", instanceID)
		} else {
			slog.Error("[PluginInstall] 查询实例失败", "instance_id", instanceID, "error", err)
		}
		return
	}
	if !model.AgentTypeSupportsPlugin(ctx, instance.AgentType) {
		slog.Info("[Plugin] 跳过插件安装，该类型不支持",
			"instance_id", instanceID, "agent_type", instance.AgentType)
		return
	}

	seen := make(map[string]bool)

	type pluginEntry struct {
		Name        string
		Slug        string
		PluginID    string
		Version     string
		CosZipKey   string
		NpmPackage  string
		InstallMode string
		Kind        string
	}
	var allPlugins []pluginEntry

	// ① 角色插件（优先级高，先加入）
	if roleID > 0 {
		var rolePlugins []model.OpenClawRolePlugin
		model.DB(ctx).Where("open_claw_role_id = ?", roleID).Find(&rolePlugins)
		for _, rp := range rolePlugins {
			if !seen[rp.Slug] {
				seen[rp.Slug] = true
				allPlugins = append(allPlugins, pluginEntry{
					Name:        rp.Name,
					Slug:        rp.Slug,
					PluginID:    rp.PluginID,
					Version:     rp.Version,
					CosZipKey:   rp.CosZipKey,
					NpmPackage:  rp.NpmPackage,
					InstallMode: rp.InstallMode,
					Kind:        rp.Kind,
				})
			}
		}
	}

	// ② 全局插件包插件
	var bundles []model.PluginBundle
	model.DB(ctx).Where("enabled = ?", true).Find(&bundles)
	if len(bundles) > 0 {
		var bundleIDs []uint
		for _, b := range bundles {
			bundleIDs = append(bundleIDs, b.ID)
		}
		var bundlePlugins []model.BundlePlugin
		model.DB(ctx).Where("plugin_bundle_id IN ?", bundleIDs).Find(&bundlePlugins)
		for _, bp := range bundlePlugins {
			if !seen[bp.Slug] {
				seen[bp.Slug] = true
				allPlugins = append(allPlugins, pluginEntry{
					Name:        bp.Name,
					Slug:        bp.Slug,
					PluginID:    bp.PluginID,
					Version:     bp.Version,
					CosZipKey:   bp.CosZipKey,
					NpmPackage:  bp.NpmPackage,
					InstallMode: bp.InstallMode,
					Kind:        bp.Kind,
				})
			}
		}
	}

	if len(allPlugins) == 0 {
		return
	}

	for _, p := range allPlugins {
		// 先查询是否已存在（处理重装场景下唯一索引冲突）
		var existing model.PluginInstallation
		err := model.DB(ctx).Where("instance_id = ? AND slug = ?", instanceID, p.Slug).First(&existing).Error
		if err == nil {
			// 已存在，更新为重新安装状态
			model.DB(ctx).Model(&existing).Updates(map[string]interface{}{
				"name":           p.Name,
				"plugin_id":      p.PluginID,
				"version":        p.Version,
				"cos_zip_key":    p.CosZipKey,
				"npm_package":    p.NpmPackage,
				"install_mode":   p.InstallMode,
				"kind":           p.Kind,
				"install_status": model.PluginInstallNone,
				"error_message":  "",
			})
		} else {
			installation := model.PluginInstallation{
				InstanceID:    instanceID,
				Name:          p.Name,
				Slug:          p.Slug,
				PluginID:      p.PluginID,
				Version:       p.Version,
				CosZipKey:     p.CosZipKey,
				NpmPackage:    p.NpmPackage,
				InstallMode:   p.InstallMode,
				Kind:          p.Kind,
				InstallStatus: model.PluginInstallNone,
			}
			if err := model.DB(ctx).Create(&installation).Error; err != nil {
				slog.Error("[PluginInstall] 创建安装记录失败", "instance_id", instanceID, "slug", p.Slug, "error", err)
			}
		}
	}

	slog.Info("[PluginInstall] 插件安装任务已创建", "instance_id", instanceID, "plugin_count", len(allPlugins), "role_id", roleID)
}

// installPluginsAsync 异步安装插件到 CVM 实例。
// waitMode 控制等待策略（与 installSkillsAsync 相同）。
func installPluginsAsync(ctx context.Context, instanceID uint, cvmInstanceId string, waitMode int) {
	logger := slog.With("task", "installPluginsAsync", "instance_id", instanceID, "cvm_instance_id", cvmInstanceId)

	// final：防御式 guard。上游 createPluginInstallTasks 已按 agent_type 过滤了
	// 不支持插件的实例（不会生成 PluginInstallation 行），这里再核验一次，
	// 避免未来新的入口绕过预创建逻辑直接调用本函数。
	var inst model.Instance
	if err := model.DB(ctx).Select("agent_type").First(&inst, instanceID).Error; err != nil {
		logger.Warn("查询实例 agent_type 失败，跳过插件安装", "error", err)
		return
	}
	if !model.AgentTypeSupportsPlugin(ctx, inst.AgentType) {
		logger.Info("实例类型不支持插件，跳过", "agent_type", inst.AgentType)
		return
	}

	// 查询待安装的插件（status = None）
	var plugins []model.PluginInstallation
	model.DB(ctx).Where("instance_id = ? AND install_status = ?", instanceID, model.PluginInstallNone).Find(&plugins)
	if len(plugins) == 0 {
		logger.Info("无待安装插件")
		return
	}

	// ── 阶段1：等待 CVM 就绪（复用 Skill 的等待逻辑） ──

	if waitMode == waitModeReinstall {
		logger.Info("重装场景：等待 CVM 进入重装状态", "plugin_count", len(plugins))
		reinstallStarted := false
		for attempt := 1; attempt <= 24; attempt++ {
			state, err := fetchCVMState(ctx, cvmInstanceId)
			if err != nil {
				time.Sleep(5 * time.Second)
				continue
			}
			if state != "RUNNING" {
				reinstallStarted = true
				break
			}
			time.Sleep(5 * time.Second)
		}
		if !reinstallStarted {
			logger.Warn("等待 CVM 进入非 RUNNING 状态超时，继续执行")
		}
	}

	if waitMode != waitModeRetry {
		logger.Info("等待 CVM 就绪", "plugin_count", len(plugins))
		cvmReady := false
		for attempt := 1; attempt <= 60; attempt++ {
			state, err := fetchCVMState(ctx, cvmInstanceId)
			if err != nil {
				time.Sleep(10 * time.Second)
				continue
			}
			if state == "RUNNING" {
				cvmReady = true
				break
			}
			time.Sleep(10 * time.Second)
		}
		if !cvmReady {
			logger.Error("等待 CVM 就绪超时，标记所有插件为失败")
			model.DB(ctx).Model(&model.PluginInstallation{}).
				Where("instance_id = ? AND install_status = ?", instanceID, model.PluginInstallNone).
				Updates(map[string]interface{}{
					"install_status": model.PluginInstallFailed,
					"error_message":  "等待 CVM 就绪超时",
				})
			return
		}
	}

	// ── 阶段2：按 InstallMode 分组 ──
	model.DB(ctx).Model(&model.PluginInstallation{}).
		Where("instance_id = ? AND install_status = ?", instanceID, model.PluginInstallNone).
		Update("install_status", model.PluginInstalling)

	model.DB(ctx).Where("instance_id = ? AND install_status = ?", instanceID, model.PluginInstalling).Find(&plugins)

	var smhPlugins, npmPlugins []model.PluginInstallation
	for _, p := range plugins {
		if p.InstallMode == "npm" && p.NpmPackage != "" {
			npmPlugins = append(npmPlugins, p)
		} else {
			smhPlugins = append(smhPlugins, p)
		}
	}

	// ── SMH 模式安装 ──
	if len(smhPlugins) > 0 {
		installPluginsSMH(ctx, instanceID, cvmInstanceId, smhPlugins, logger)
	}

	// ── npm 模式安装 ──
	if len(npmPlugins) > 0 {
		installPluginsNPM(ctx, instanceID, cvmInstanceId, npmPlugins, logger)
	}
}

// installPluginsSMH 通过 SMH 下载安装插件
func installPluginsSMH(ctx context.Context, instanceID uint, cvmInstanceId string, plugins []model.PluginInstallation, logger *slog.Logger) {
	var lines []string
	var validPlugins []model.PluginInstallation

	for _, p := range plugins {
		if p.CosZipKey == "" {
			logger.Warn("插件 cos_zip_key 为空，跳过安装", "slug", p.Slug)
			model.DB(ctx).Model(&p).Updates(map[string]interface{}{
				"install_status": model.PluginInstallFailed,
				"error_message":  "插件包尚未完成 SMH 同步",
			})
			continue
		}

		downloadURL, err := BuildCommonSMHDownloadURL(ctx, p.CosZipKey, true)
		if err != nil {
			logger.Error("生成下载 URL 失败", "slug", p.Slug, "error", err)
			model.DB(ctx).Model(&p).Updates(map[string]interface{}{
				"install_status": model.PluginInstallFailed,
				"error_message":  "生成下载 URL 失败: " + err.Error(),
			})
			continue
		}

		// 每行格式：download_url<TAB>plugin_slug<TAB>plugin_id<TAB>plugin_version<TAB>plugin_kind
		lines = append(lines, fmt.Sprintf("%s\t%s\t%s\t%s\t%s", downloadURL, p.Slug, p.PluginID, p.Version, p.Kind))
		validPlugins = append(validPlugins, p)
	}

	if len(lines) == 0 {
		return
	}

	pluginsList := strings.Join(lines, "\n")
	pluginsListB64 := base64.StdEncoding.EncodeToString([]byte(pluginsList))

	timeout := uint64((len(validPlugins)/10 + 1) * 180)
	if timeout > 1800 {
		timeout = 1800
	}

	var output string
	var tatErr error
	for retry := 1; retry <= 5; retry++ {
		output, tatErr = RunScript(ctx, cvmInstanceId, "batch_install_plugins_from_smh.sh", timeout, LookupRuntimeUser(ctx, cvmInstanceId), nil, map[string]string{
			"plugins_list": pluginsListB64,
		})
		if tatErr == nil {
			break
		}
		logger.Warn("TAT 执行失败，重试", "retry", retry, "error", tatErr)
		time.Sleep(10 * time.Second)
	}

	if tatErr != nil {
		logger.Error("TAT 执行失败", "error", tatErr)
		for _, p := range validPlugins {
			model.DB(ctx).Model(&p).Updates(map[string]interface{}{
				"install_status": model.PluginInstallFailed,
				"error_message":  "TAT 执行失败: " + tatErr.Error(),
			})
		}
		return
	}

	parseAndUpdatePluginInstallResults(ctx, output, validPlugins, logger)
}

// installPluginsNPM 通过 npm 安装插件
func installPluginsNPM(ctx context.Context, instanceID uint, cvmInstanceId string, plugins []model.PluginInstallation, logger *slog.Logger) {
	var lines []string
	var validPlugins []model.PluginInstallation

	for _, p := range plugins {
		if p.NpmPackage == "" {
			model.DB(ctx).Model(&p).Updates(map[string]interface{}{
				"install_status": model.PluginInstallFailed,
				"error_message":  "npm 包名为空",
			})
			continue
		}
		// 每行格式：npm_package<TAB>plugin_id<TAB>plugin_version<TAB>plugin_kind
		lines = append(lines, fmt.Sprintf("%s\t%s\t%s\t%s", p.NpmPackage, p.PluginID, p.Version, p.Kind))
		validPlugins = append(validPlugins, p)
	}

	if len(lines) == 0 {
		return
	}

	pluginsList := strings.Join(lines, "\n")
	pluginsListB64 := base64.StdEncoding.EncodeToString([]byte(pluginsList))

	timeout := uint64((len(validPlugins)/5 + 1) * 300)
	if timeout > 1800 {
		timeout = 1800
	}

	var output string
	var tatErr error
	for retry := 1; retry <= 3; retry++ {
		output, tatErr = RunScript(ctx, cvmInstanceId, "batch_install_plugins_npm.sh", timeout, LookupRuntimeUser(ctx, cvmInstanceId), nil, map[string]string{
			"plugins_list": pluginsListB64,
		})
		if tatErr == nil {
			break
		}
		logger.Warn("npm TAT 执行失败，重试", "retry", retry, "error", tatErr)
		time.Sleep(15 * time.Second)
	}

	if tatErr != nil {
		logger.Error("npm TAT 执行失败", "error", tatErr)
		for _, p := range validPlugins {
			model.DB(ctx).Model(&p).Updates(map[string]interface{}{
				"install_status": model.PluginInstallFailed,
				"error_message":  "npm TAT 执行失败: " + tatErr.Error(),
			})
		}
		return
	}

	parseAndUpdatePluginInstallResults(ctx, output, validPlugins, logger)
}

// parseAndUpdatePluginInstallResults 解析 TAT 输出并更新插件安装状态
func parseAndUpdatePluginInstallResults(ctx context.Context, output string, plugins []model.PluginInstallation, logger *slog.Logger) {
	lines := strings.Split(output, "\n")
	var jsonLine string

	for i, line := range lines {
		if strings.Contains(line, "BATCH INSTALL RESULTS") && i+1 < len(lines) {
			jsonLine = strings.TrimSpace(lines[i+1])
			break
		}
	}

	if jsonLine == "" {
		for i := len(lines) - 1; i >= 0; i-- {
			trimmed := strings.TrimSpace(lines[i])
			if strings.HasPrefix(trimmed, "{") {
				jsonLine = trimmed
				break
			}
		}
	}

	if jsonLine == "" {
		logger.Error("TAT 输出中未找到 JSON 结果")
		for _, p := range plugins {
			model.DB(ctx).Model(&p).Updates(map[string]interface{}{
				"install_status": model.PluginInstallFailed,
				"error_message":  "TAT 输出中未找到安装结果",
			})
		}
		return
	}

	var result batchPluginInstallResult
	if err := json.Unmarshal([]byte(jsonLine), &result); err != nil {
		logger.Error("解析 JSON 结果失败", "error", err)
		for _, p := range plugins {
			model.DB(ctx).Model(&p).Updates(map[string]interface{}{
				"install_status": model.PluginInstallFailed,
				"error_message":  "解析安装结果 JSON 失败",
			})
		}
		return
	}

	resultMap := make(map[string]struct {
		Status  string
		Message string
	})
	for _, r := range result.Results {
		resultMap[r.Slug] = struct {
			Status  string
			Message string
		}{r.Status, r.Message}
	}

	for _, p := range plugins {
		if r, ok := resultMap[p.Slug]; ok {
			if r.Status == "success" {
				model.DB(ctx).Model(&p).Updates(map[string]interface{}{
					"install_status": model.PluginInstallSuccess,
					"error_message":  "",
				})
			} else {
				model.DB(ctx).Model(&p).Updates(map[string]interface{}{
					"install_status": model.PluginInstallFailed,
					"error_message":  r.Message,
				})
			}
		} else {
			model.DB(ctx).Model(&p).Updates(map[string]interface{}{
				"install_status": model.PluginInstallFailed,
				"error_message":  "安装结果中未找到该插件",
			})
		}
	}

	logger.Info("插件安装结果已更新",
		"total", result.Summary.Total,
		"success", result.Summary.Success,
		"failed", result.Summary.Failed)
}
