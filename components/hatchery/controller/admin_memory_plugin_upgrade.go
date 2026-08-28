package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// ========== 接口一：查询待升级实例列表 ==========

// HandleAdminMemoryPluginUpgradeCandidates GET /admin/memory/plugin-upgrade/candidates
// 查询所有 Pro 版且插件版本低于 DefaultMemoryTDAIMinVersion 的 OpenClaw 实例。
// 接口会实时通过 TAT 查询 CVM 上的插件版本和 offload 状态，并回写 DB。
func HandleAdminMemoryPluginUpgradeCandidates(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}
	log := Logger(r.Context())
	log.Info("[PluginUpgradeCandidates] 收到查询请求")

	// Step 1: 从 DB 查出候选实例（Pro + 非转圈 + OpenClaw 类型）
	var plugins []model.MemoryTDAIPlugin
	if err := model.DB(r.Context()).
		Joins("JOIN instances ON instances.instance_id = memory_tda_iplugins.instance_id AND instances.deleted_at IS NULL").
		Where("memory_tda_iplugins.current_plan = ? AND memory_tda_iplugins.switch_status = ?",
			model.MemoryPlanPro, model.MemorySwitchStatusNone).
		Where("(instances.agent_type = ? OR instances.agent_type = ?)", model.AgentTypeOpenClaw, "").
		Find(&plugins).Error; err != nil {
		log.Error("[PluginUpgradeCandidates] 查询 DB 失败", "error", err)
		writeError(w, r, http.StatusInternalServerError, common.I18nRichError(err, i18n.MsgPluginUpgradeQueryFail))
		return
	}

	if len(plugins) == 0 {
		jsonOK(w, map[string]any{
			"min_version": model.DefaultMemoryTDAIMinVersion,
			"total":       0,
			"instances":   []any{},
		})
		return
	}

	// Step 2: 并发 TAT 查询每个实例的插件版本 + offload 状态（并发度上限 15）
	type probeResult struct {
		InstanceID string
		Version    string
		Offload    bool
		Err        error
	}

	var wg sync.WaitGroup
	results := make([]probeResult, len(plugins))
	const probeConcurrency = 15
	sem := make(chan struct{}, probeConcurrency)

	for i, p := range plugins {
		sem <- struct{}{}
		wg.Add(1)
		go func(ctx context.Context, idx int, instanceID string) {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					slog.Error("[PluginUpgradeCandidates] panic", "instance_id", instanceID, "recover", r)
					results[idx] = probeResult{InstanceID: instanceID, Err: common.I18nError(i18n.MsgPluginUpgradePanic, r)}
				}
			}()

			script := `jq -r '{ "version": (.plugins.installs["memory-tencentdb"].version // ""), "offload": (.plugins.entries["memory-tencentdb"].config.offload.enabled // false) }' ~/.openclaw/openclaw.json`
			output, err := runInlineScriptFn(ctx, instanceID, script, uint64(15))
			if err != nil {
				results[idx] = probeResult{InstanceID: instanceID, Err: err}
				return
			}

			var parsed struct {
				Version string `json:"version"`
				Offload bool   `json:"offload"`
			}
			if err := json.Unmarshal([]byte(output), &parsed); err != nil {
				results[idx] = probeResult{InstanceID: instanceID, Err: common.I18nRichError(err, i18n.MsgPluginUpgradeParseJSONFailed)}
				return
			}
			results[idx] = probeResult{
				InstanceID: instanceID,
				Version:    parsed.Version,
				Offload:    parsed.Offload,
			}
		}(common.DetachContext(r.Context()), i, p.InstanceID)
	}
	wg.Wait()

	// Step 3: 回写 DB + 筛选
	type candidateItem struct {
		InstanceID          string `json:"instance_id"`
		InstanceName        string `json:"instance_name"`
		CreatorName         string `json:"creator_name"`
		MemoryPluginVersion string `json:"memory_plugin_version"`
		OffloadEnabled      *bool  `json:"offload_enabled"`
	}

	var candidates []candidateItem
	minVersion := model.DefaultMemoryTDAIMinVersion

	for _, res := range results {
		if res.Err != nil {
			log.Warn("[PluginUpgradeCandidates] 实例探测失败，跳过",
				"instance_id", res.InstanceID, "error", res.Err)
			continue
		}

		// 回写 DB
		offloadVal := res.Offload
		if dbErr := model.DB(r.Context()).Model(&model.MemoryTDAIPlugin{}).
			Where("instance_id = ?", res.InstanceID).
			Updates(map[string]any{
				"memory_plugin_version": res.Version,
				"offload_enabled":       &offloadVal,
			}).Error; dbErr != nil {
			log.Warn("[PluginUpgradeCandidates] DB 回写失败",
				"instance_id", res.InstanceID, "error", dbErr)
		}

		// 筛选：版本低于最低要求 OR offload 未开启
		needsUpgrade := res.Version == "" || versionLessThan(res.Version, minVersion)
		needsOffload := !res.Offload
		if needsUpgrade || needsOffload {
			candidates = append(candidates, candidateItem{
				InstanceID:          res.InstanceID,
				MemoryPluginVersion: res.Version,
				OffloadEnabled:      &offloadVal,
			})
		}
	}

	// Step 4: 补充 instance_name 和 creator_name
	if len(candidates) > 0 {
		instanceIDs := make([]string, len(candidates))
		for i, c := range candidates {
			instanceIDs[i] = c.InstanceID
		}

		type instanceInfo struct {
			InstanceID   string
			InstanceName string
			CreatorName  string
		}
		var infos []instanceInfo
		model.DB(r.Context()).Model(&model.Instance{}).
			Select("instances.instance_id, instances.name as instance_name, COALESCE(users.username, '') as creator_name").
			Joins("LEFT JOIN users ON users.id = instances.user_id AND users.deleted_at IS NULL").
			Where("instances.instance_id IN ?", instanceIDs).
			Find(&infos)

		infoMap := make(map[string]instanceInfo, len(infos))
		for _, info := range infos {
			infoMap[info.InstanceID] = info
		}
		for i := range candidates {
			if info, ok := infoMap[candidates[i].InstanceID]; ok {
				candidates[i].InstanceName = info.InstanceName
				candidates[i].CreatorName = info.CreatorName
			}
		}
	}

	log.Info("[PluginUpgradeCandidates] 查询完成",
		"total_pro", len(plugins), "candidates", len(candidates))

	jsonOK(w, map[string]any{
		"min_version": minVersion,
		"total":       len(candidates),
		"instances":   candidates,
	})
}

// ========== 接口二：触发插件升级 + 开启 Offload ==========

type pluginUpgradeExecuteRequest struct {
	InstanceIDs []string `json:"instance_ids"`
}

// HandleAdminMemoryPluginUpgradeExecute POST /admin/memory/plugin-upgrade/execute
// 异步触发批量插件升级 + 开启 Offload。
func HandleAdminMemoryPluginUpgradeExecute(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, common.I18nError(i18n.MsgOnlyPostMethodSupported))
		return
	}

	var req pluginUpgradeExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, common.I18nRichError(err, i18n.MsgBadRequest))
		return
	}
	if len(req.InstanceIDs) == 0 {
		writeError(w, r, http.StatusBadRequest, common.I18nError(i18n.MsgInstanceIdsCannotBeEmpty))
		return
	}

	log := Logger(r.Context())
	log.Info("[PluginUpgradeExecute] 收到升级请求", "instance_count", len(req.InstanceIDs))

	// 校验每个实例并收集结果
	type submitResult struct {
		InstanceID string `json:"instance_id"`
		Status     string `json:"status"`
		Message    string `json:"message,omitempty"`
	}
	var results []submitResult
	var validIDs []string

	for _, instID := range req.InstanceIDs {
		instID = strings.TrimSpace(instID)
		if instID == "" {
			continue
		}

		// 校验实例存在且为 Pro
		var plugin model.MemoryTDAIPlugin
		if err := model.DB(r.Context()).Where("instance_id = ?", instID).First(&plugin).Error; err != nil {
			results = append(results, submitResult{InstanceID: instID, Status: "failed", Message: i18n.T(r.Context(), i18n.MsgInstanceNotFound)})
			continue
		}
		if plugin.CurrentPlan != model.MemoryPlanPro {
			results = append(results, submitResult{InstanceID: instID, Status: "failed", Message: i18n.T(r.Context(), i18n.MsgMemPluginInstanceNotPro)})
			continue
		}
		if plugin.SwitchStatus != "" {
			results = append(results, submitResult{InstanceID: instID, Status: "failed", Message: i18n.T(r.Context(), i18n.MsgMemPluginSwitchInProgress, plugin.SwitchStatus)})
			continue
		}

		// 校验实例类型支持插件（通过 SupportsPlugin 矩阵判断）
		var inst model.Instance
		if err := model.DB(r.Context()).Where("instance_id = ?", instID).First(&inst).Error; err != nil {
			results = append(results, submitResult{InstanceID: instID, Status: "failed", Message: i18n.T(r.Context(), i18n.MsgMemPluginInstanceRecordMissing)})
			continue
		}
		if !model.AgentTypeSupportsPlugin(r.Context(), inst.AgentType) {
			results = append(results, submitResult{InstanceID: instID, Status: "failed", Message: i18n.T(r.Context(), i18n.MsgMemPluginAgentTypeUnsupported)})
			continue
		}

		validIDs = append(validIDs, instID)
		results = append(results, submitResult{InstanceID: instID, Status: "submitted"})
	}

	// 异步执行升级（并发度上限 5）
	offloadURL := GetOffloadBackendURL()
	offloadUin := common.CVMUinFromCtx(r.Context())
	traceLog := log // 捕获带 trace_id 的 logger 传入异步 goroutine

	go func(ctx context.Context) {
		const maxConcurrency = 5
		sem := make(chan struct{}, maxConcurrency)
		var wg sync.WaitGroup

		for _, instanceID := range validIDs {
			sem <- struct{}{}
			wg.Add(1)
			go func(id string) {
				defer wg.Done()
				defer func() { <-sem }()
				defer func() {
					if r := recover(); r != nil {
						traceLog.Error("[PluginUpgrade] panic", "instance_id", id, "recover", r)
					}
				}()
				doPluginUpgrade(ctx, traceLog, id, offloadURL, offloadUin)
			}(instanceID)
		}
		wg.Wait()
		traceLog.Info("[PluginUpgradeExecute] 所有升级任务执行完毕", "count", len(validIDs))
	}(common.DetachContext(r.Context()))

	log.Info("[PluginUpgradeExecute] 升级任务已提交",
		"submitted", len(validIDs), "total", len(req.InstanceIDs))

	jsonOK(w, map[string]any{
		"min_version": model.DefaultMemoryTDAIMinVersion,
		"submitted":   len(validIDs),
		"results":     results,
	})
}

// doPluginUpgrade 对单个实例执行升级插件 + 开启 offload 的完整流程。
// 容错原则：升级失败不执行 offload；offload 失败不影响升级结果。
func doPluginUpgrade(ctx context.Context, parentLog *slog.Logger, instanceID, offloadURL, offloadUin string) {
	log := parentLog.With("instance_id", instanceID)

	// 第一步：升级插件
	log.Info("[PluginUpgrade] 开始升级插件")
	if err := ensureMemoryPluginFn(ctx, instanceID); err != nil {
		log.Warn("[PluginUpgrade] 插件升级失败，跳过 offload", "error", err)
		return
	}
	log.Info("[PluginUpgrade] 插件升级成功")

	// 升级成功后回写版本
	versionOutput, err := runInlineScriptFn(ctx, instanceID,
		`jq -r '.plugins.installs["memory-tencentdb"].version // empty' ~/.openclaw/openclaw.json`, uint64(10))
	if err == nil && versionOutput != "" {
		if dbErr := model.DB(ctx).Model(&model.MemoryTDAIPlugin{}).
			Where("instance_id = ?", instanceID).
			Update("memory_plugin_version", strings.TrimSpace(versionOutput)).Error; dbErr != nil {
			log.Warn("[PluginUpgrade] 版本回写 DB 失败", "error", dbErr)
		} else {
			log.Info("[PluginUpgrade] 版本已回写", "version", strings.TrimSpace(versionOutput))
		}
	}

	// 第二步：开启 Offload
	if offloadURL != "" {
		pluginRoot, resolveErr := ResolveMemoryPluginRoot(ctx, instanceID)
		if resolveErr != nil {
			log.Warn("[PluginUpgrade] 插件路径探测失败，跳过 offload", "error", resolveErr)
		} else {
			offloadScript := fmt.Sprintf(`
OFFLOAD_SCRIPT="%s/scripts/setup-offload.sh"
if [ -f "$OFFLOAD_SCRIPT" ]; then
    bash "$OFFLOAD_SCRIPT" --enable --user-id "%s" --backend-url "%s"
else
    echo "ERROR: setup-offload.sh not found at $OFFLOAD_SCRIPT"
    exit 1
fi
`, pluginRoot, offloadUin, offloadURL)

			log.Info("[PluginUpgrade] 开始开启 offload",
				"backend_url", offloadURL, "user_id", offloadUin)

			_, err = runInlineScriptFn(ctx, instanceID, offloadScript, uint64(60))
			if err != nil {
				log.Warn("[PluginUpgrade] offload 开启失败（非阻断）", "error", err)
			} else {
				// offload 成功后回写 DB
				offloadTrue := true
				if dbErr := model.DB(ctx).Model(&model.MemoryTDAIPlugin{}).
					Where("instance_id = ?", instanceID).
					Update("offload_enabled", &offloadTrue).Error; dbErr != nil {
					log.Warn("[PluginUpgrade] offload 回写 DB 失败", "error", dbErr)
				} else {
					log.Info("[PluginUpgrade] offload 开启成功并已回写")
				}
			}
		}
	} else {
		log.Warn("[PluginUpgrade] offload_backend_url 为空，跳过 offload 开启")
	}

	// 第三步：重启 openclaw 使配置生效（不管 offload 是否成功，只要插件升级成功就重启）
	// 与 memory_tdai_switch_pro.sh Step 7 保持一致：通过 systemctl 重启 gateway 服务
	restartScript := `
if systemctl --user is-active openclaw-gateway >/dev/null 2>&1; then
    systemctl --user restart openclaw-gateway 2>&1 || echo "WARN: gateway restart failed"
    echo "openclaw-gateway restarted"
else
    echo "WARN: openclaw-gateway not available, skip restart"
fi
`
	_, err = runInlineScriptFn(ctx, instanceID, restartScript, uint64(30))
	if err != nil {
		log.Warn("[PluginUpgrade] openclaw 重启失败（非阻断）", "error", err)
	} else {
		log.Info("[PluginUpgrade] openclaw 已重启")
	}
}

// versionLessThan 简单 semver 比较：a < b 返回 true。
func versionLessThan(a, b string) bool {
	a = strings.TrimPrefix(a, "v")
	b = strings.TrimPrefix(b, "v")

	partsA := strings.Split(a, ".")
	partsB := strings.Split(b, ".")

	for i := 0; i < 3; i++ {
		var na, nb int
		if i < len(partsA) {
			fmt.Sscanf(strings.Split(partsA[i], "-")[0], "%d", &na)
		}
		if i < len(partsB) {
			fmt.Sscanf(strings.Split(partsB[i], "-")[0], "%d", &nb)
		}
		if na < nb {
			return true
		}
		if na > nb {
			return false
		}
	}
	return false
}
