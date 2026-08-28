package task

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/controller"
	"hatchery/i18n"
	"hatchery/model"

	sdk "hatchery/internal/tdaimemorysdk"
)

// 可替换函数变量，方便单元测试 mock 外部调用
var newAgentMemoryClientFn = func(ctx context.Context) (*sdk.Client, error) { return controller.NewMemorySDKClient(ctx) }

// handleSwitchToPro 处理 OFF/FREE -> PRO 切换。
// 步骤：validate → wait_instance_ready → allocate_database → persist_binding → migrate_to_pro → update_plan_status
func handleSwitchToPro(job *model.TdaiJob) error {
	ctx := jobCtx(job)
	log := jobLogger(job)
	// Step 1: validate
	updateStep(job, "validate_request", 5)
	if err := checkInstanceSupportsMemoryTask(ctx, job.InstanceID); err != nil {
		return err
	}
	plugin, err := getPluginByInstanceID(ctx, job.InstanceID)
	if err != nil {
		return err
	}
	previousPlan := string(plugin.CurrentPlan) // 记录切换前的计划（OFF / FREE / PRO）
	log.Info("[SwitchToPro] validate 完成",
		"current_plan", plugin.CurrentPlan,
		"desired_plan", plugin.DesiredPlan,
		"switch_status", plugin.SwitchStatus,
		"pool_id", plugin.PoolID,
	)
	// 重装场景：实例已在 PRO 且有完整绑定信息 → 跳过分配，直接重新下发配置
	isRedelivery := plugin.CurrentPlan == model.MemoryPlanPro && plugin.PoolID != "" &&
		plugin.Endpoint != "" && plugin.Endpoint != "http://:0"

	if plugin.CurrentPlan == model.MemoryPlanPro && !isRedelivery {
		// 已是 PRO 且绑定信息不完整 → 异常状态，当 Non-retryable 处理
		log.Warn("[SwitchToPro] 实例已在 PRO 但绑定信息不完整",
			"pool_id", plugin.PoolID, "endpoint", plugin.Endpoint)
		return NewNonRetryableError(i18n.T(ctx, i18n.MsgTDAIProBindingIncomplete))
	}

	// Step 2: 等待实例就绪（CVM RUNNING）
	updateStep(job, "wait_instance_ready", 10)
	if err := checkInstanceRunning(ctx, job.InstanceID); err != nil {
		return err
	}
	log.Info("[SwitchToPro] 实例已就绪")

	// Step 2.5: Hermes 实例先执行 install 脚本（安装插件），再 ensure 校验
	// 兼容 hermes 的自定义类型走相同路径。
	agentType := controller.LookupAgentType(ctx, job.InstanceID)
	runtimeUser := controller.LookupRuntimeUser(ctx, job.InstanceID)
	if model.GetAgentRuntimeType(ctx, agentType) == model.AgentTypeHermes {
		updateStep(job, "install_plugin", 15)
		if _, err := taskRunScriptFn(ctx, job.InstanceID, "install_hermes_tdai_gateway.sh",
			600, runtimeUser, nil, nil); err != nil {
			log.Warn("[SwitchToPro] Hermes install 脚本执行失败",
				"error", err)
			return hcommon.I18nRichError(err, i18n.MsgTDAIHermesInstallFailed)
		}
	}

	// Step 2.6: ensure plugin installed
	updateStep(job, "ensure_plugin", 18)
	ensureStart := time.Now()
	log.Info("[SwitchToPro] 调用 EnsureMemoryPlugin 开始")
	if err := controller.EnsureMemoryPlugin(ctx, job.InstanceID); err != nil {
		log.Warn("[SwitchToPro] 插件就绪检查失败",
			"cost", time.Since(ensureStart).String(),
			"error", err)
		return hcommon.I18nRichError(err, i18n.MsgTDAIPluginReadyCheckFailed)
	}
	log.Info("[SwitchToPro] EnsureMemoryPlugin 完成", "cost", time.Since(ensureStart).String())

	// Step 3: allocate database（创建记忆库）
	updateStep(job, "allocate_database", 25)
	allocStart := time.Now()
	log.Info("[SwitchToPro] 开始分配记忆库")
	client, err := newAgentMemoryClientFn(ctx)
	if err != nil {
		log.Error("[SwitchToPro] 初始化 Agent Memory SDK 失败",
			"cost", time.Since(allocStart).String(),
			"error", err)
		return hcommon.I18nRichError(err, i18n.MsgTDAISDKInitFailed)
	}

	var vdbEndpoint, database, apiKey, spaceId, vdbUsername, embeddingModel string

	// 幂等检查：若 plugin 已有完整绑定信息，直接复用 DB 中的值
	// DescribeMemSpaces 查询接口不返回 Vip/Port/ApiKey 等连接信息，不能用它覆盖
	if plugin.PoolID != "" && plugin.Endpoint != "" && plugin.Endpoint != "http://:0" {
		// 确认远端记忆库仍存在
		descStart := time.Now()
		log.Info("[SwitchToPro] 调用 DescribeMemSpaces 校验远端记忆库存在性",
			"pool_id", plugin.PoolID)
		descSpaces, err := client.DescribeMemSpaces(ctx, &sdk.DescribeMemSpacesRequest{
			SpaceIds: []string{plugin.PoolID},
		})
		if err != nil {
			log.Warn("[SwitchToPro] DescribeMemSpaces 调用失败，将走重新创建分支",
				"pool_id", plugin.PoolID,
				"cost", time.Since(descStart).String(),
				"error", err)
		} else if descSpaces.TotalCount > 0 {
			spaceId = plugin.PoolID
			vdbEndpoint = plugin.Endpoint
			database = plugin.DatabaseName
			apiKey = plugin.ApiKeySecretRef
			vdbUsername = plugin.VdbUsername
			embeddingModel = plugin.EmbeddingModel
			log.Info("[SwitchToPro] 记忆库已存在，复用 DB 绑定信息",
				"space_id", spaceId,
				"database", database,
				"endpoint", vdbEndpoint,
				"cost", time.Since(descStart).String(),
			)
		} else {
			log.Warn("[SwitchToPro] DB 有绑定但远端记忆库不存在，将重新创建",
				"pool_id", plugin.PoolID,
				"cost", time.Since(descStart).String())
		}
	}

	// 若无已有记忆库，创建新的
	if spaceId == "" {
		descInstStart := time.Now()
		log.Info("[SwitchToPro] 调用 DescribeMemoryProInstances 查询可用 Memory Pro 实例")
		proInstances, err := client.DescribeMemoryProInstances(ctx, &sdk.DescribeMemoryProInstancesRequest{})
		if err != nil {
			log.Error("[SwitchToPro] 查询 Memory Pro 实例失败",
				"cost", time.Since(descInstStart).String(),
				"error", err)
			return hcommon.I18nRichError(err, i18n.MsgTDAIQueryMemoryProFailed)
		}
		log.Info("[SwitchToPro] DescribeMemoryProInstances 完成",
			"total_count", proInstances.TotalCount,
			"items", len(proInstances.Items),
			"cost", time.Since(descInstStart).String(),
		)
		if proInstances.TotalCount == 0 || len(proInstances.Items) == 0 {
			log.Warn("[SwitchToPro] 未开通 Memory Pro 服务")
			return NewNonRetryableError(i18n.T(ctx, i18n.MsgTDAIMemoryProNotEnabled))
		}
		// 取第一个 online 状态的实例
		var proInstance *sdk.MemoryProInstanceInfo
		for i := range proInstances.Items {
			if proInstances.Items[i].Status == "online" {
				proInstance = &proInstances.Items[i]
				break
			}
		}
		if proInstance == nil {
			log.Warn("[SwitchToPro] 无可用的 online 状态 Memory Pro 实例",
				"total", proInstances.TotalCount)
			return hcommon.I18nError(i18n.MsgTDAINoOnlineMemoryPro, proInstances.TotalCount)
		}

		createStart := time.Now()
		log.Info("[SwitchToPro] 调用 CreateMemSpace 创建记忆库",
			"memory_pro_id", proInstance.MemoryProId)
		createResp, err := client.CreateMemSpace(ctx, &sdk.CreateMemSpaceRequest{
			MemoryProId: proInstance.MemoryProId,
		})
		if err != nil {
			log.Error("[SwitchToPro] 创建记忆库失败",
				"memory_pro_id", proInstance.MemoryProId,
				"cost", time.Since(createStart).String(),
				"error", err)
			return hcommon.I18nRichError(err, i18n.MsgTDAICreateMemSpaceFailed)
		}
		vdbEndpoint = fmt.Sprintf("http://%s:%d", createResp.Vip, createResp.Port)
		database = createResp.DatabaseName
		apiKey = createResp.ApiKey
		vdbUsername = createResp.Account
		spaceId = createResp.SpaceId
		embeddingModel = createResp.EmbeddingModel
		log.Info("[SwitchToPro] 记忆库创建成功",
			"memory_pro_id", proInstance.MemoryProId,
			"space_id", spaceId,
			"database", database,
			"endpoint", vdbEndpoint,
			"cost", time.Since(createStart).String(),
		)
	}
	log.Info("[SwitchToPro] 记忆库分配完成",
		"space_id", spaceId,
		"cost", time.Since(allocStart).String(),
	)

	// Step 4: persist binding（落库绑定信息）
	updateStep(job, "persist_binding", 45)
	log.Info("[SwitchToPro] 落库绑定信息",
		"space_id", spaceId,
		"database", database,
		"endpoint", vdbEndpoint,
		"username", vdbUsername,
		"embedding_model", embeddingModel,
	)
	if err := model.DB(ctx).Model(plugin).Updates(map[string]any{
		"pool_id":            spaceId,
		"database_name":      database,
		"endpoint":           vdbEndpoint,
		"api_key_secret_ref": apiKey,
		"vdb_username":       vdbUsername,
		"embedding_model":    embeddingModel,
	}).Error; err != nil {
		log.Error("[SwitchToPro] 落库绑定信息失败", "error", err)
		return hcommon.I18nRichError(err, i18n.MsgTDAIPersistBindingFailed)
	}

	// Step 4.5: VDB 连通性预检
	// 在 CVM 上探测能否访问 VDB endpoint，避免后续 switch_pro 脚本耗尽 10min 超时才失败、
	// 并被任务框架反复重试。网络不通 → NonRetryable 直接终态 + 自动 rollback mem space。
	updateStep(job, "vdb_connectivity_precheck", 50)
	if err := precheckVDBConnectivity(ctx, job.InstanceID, vdbEndpoint, vdbUsername, apiKey); err != nil {
		log.Warn("[SwitchToPro] VDB 连通性预检失败", "endpoint", vdbEndpoint, "error", err)
		return err
	}
	log.Info("[SwitchToPro] VDB 连通性预检通过", "endpoint", vdbEndpoint)

	// Step 5: switch_pro（升级插件 + 调用 migrate.ts：配置下发 + 按需数据迁移 + 重启）
	updateStep(job, "switch_pro", 60)
	switchStart := time.Now()
	log.Info("[SwitchToPro] 调用 runSwitchPro 开始（执行 memory_tdai_switch_pro.sh）",
		"previous_plan", previousPlan,
		"endpoint", vdbEndpoint,
		"database", database,
	)
	if err := runSwitchPro(job.ID, job.InstanceID, previousPlan, vdbEndpoint, database, apiKey, vdbUsername, embeddingModel); err != nil {
		log.Error("[SwitchToPro] switch_pro 脚本执行失败",
			"cost", time.Since(switchStart).String(),
			"previous_plan", previousPlan, "error", err)
		return hcommon.I18nRichError(err, i18n.MsgTDAISwitchProFailed)
	}
	log.Info("[SwitchToPro] runSwitchPro 完成", "cost", time.Since(switchStart).String())

	// Step 6: update plan status
	updateStep(job, "update_plan_status", 90)
	now := time.Now()
	if err := model.DB(ctx).Model(plugin).Updates(map[string]any{
		"current_plan":     model.MemoryPlanPro,
		"desired_plan":     model.MemoryPlanPro,
		"switch_status":    model.MemorySwitchStatusNone,
		"last_task_id":     job.ID,
		"last_switched_at": &now,
	}).Error; err != nil {
		log.Error("[SwitchToPro] 更新 plan 状态失败", "error", err)
		return hcommon.I18nRichError(err, i18n.MsgTDAIUpdatePlanFailed)
	}
	log.Info("[SwitchToPro] 计划已更新为 PRO",
		"space_id", spaceId, "previous_plan", previousPlan)

	updateStep(job, "mark_succeeded", 100)
	return nil
}

// runSwitchPro 调用 memory_tdai_switch_pro.sh 完成 Pro 切换。
// previousPlan 传入切换前的计划（OFF/FREE），脚本据此决定是否执行数据迁移。
func runSwitchPro(jobID uint, instanceID, previousPlan, endpoint, database, apiKey, username, embeddingModel string) error {
	ctx := lookupJobCtx(jobID)
	log := controller.Logger(ctx).With("instance_id", instanceID, "job_id", jobID)

	// embeddingModel 为空时使用默认值（兼容历史数据尚无 embedding_model 的场景）
	if embeddingModel == "" {
		embeddingModel = "qwen3-embedding-0.6b"
		log.Warn("[SwitchToPro] embeddingModel 为空，使用默认值", "default", embeddingModel)
	}

	offloadURL := controller.GetOffloadBackendURL()
	offloadUin := hcommon.CVMUinFromCtx(ctx)

	// 探测插件根目录（仅 OpenClaw 需要，Hermes 在脚本内走固定路径分支）
	agentType := controller.LookupAgentType(ctx, instanceID)
	pluginRoot := ""
	if agentType != model.AgentTypeHermes {
		var resolveErr error
		pluginRoot, resolveErr = controller.ResolveMemoryPluginRoot(ctx, instanceID)
		if resolveErr != nil {
			log.Warn("[SwitchToPro] 插件路径探测失败", "error", resolveErr)
			return hcommon.I18nRichError(resolveErr, i18n.MsgMemoryPluginPathFailed)
		}
	}

	log.Info("[SwitchToPro] runSwitchPro 透传脚本参数",
		"previous_plan", previousPlan,
		"endpoint", endpoint,
		"database", database,
		"username", username,
		"embedding_model", embeddingModel,
		"offload_backend_url", offloadURL,
		"offload_user_id", offloadUin,
		"plugin_root", pluginRoot,
	)

	_, err := taskRunScriptFn(ctx, instanceID, "memory_tdai_switch_pro.sh", 600, controller.LookupRuntimeUser(ctx, instanceID), nil,
		map[string]string{
			"plugin":              model.DefaultMemoryTDAIPluginName,
			"previous_plan":       previousPlan,
			"vdb_endpoint":        endpoint,
			"vdb_database":        database,
			"vdb_api_key":         apiKey,
			"vdb_username":        username,
			"embedding_model":     embeddingModel,
			"job_id":              fmt.Sprintf("%d", jobID),
			"agent_type":          agentType,
			"offload_backend_url": offloadURL,
			"offload_user_id":     offloadUin,
			"plugin_root":         pluginRoot,
		})
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgTDAIMigrateScriptFailed)
	}

	log.Info("[SwitchToPro] migrate_to_pro 脚本执行完成")
	return nil
}

// vdbReachability 描述 VDB 连通性预检的三态结果。
type vdbReachability int

const (
	vdbReachabilityReachable   vdbReachability = iota // 网络通
	vdbReachabilityUnreachable                        // 真不通（probe 工具明确判定）
	vdbReachabilityUnknown                            // 工具不可用 / 输出无法解析 / TAT 调用失败
)

// probeVDBReachable 调用 precheck_vdb_connectivity.sh 探测 VDB 连通性，返回三态结果。
// 这是底层 helper，由不同业务场景的判定函数复用：
//   - 切 Pro 前：unreachable → NonRetryable 拒绝切换
//   - 切 OFF 前：unreachable → 跳过 export 继续执行
//
// 第二个返回值 detail 是脚本原始输出，便于上层日志/错误信息透传。
// 第三个返回值 err 在以下场景非 nil：
//   - TAT 调用失败（agent 离线、超时等）
//   - 脚本输出无任何合法 JSON 摘要（脚本异常 / 超时被 kill 等）
//
// "工具不可用"是脚本明确判定的合法结果，err 为 nil（state=Unknown）。
//
// 重要：TAT 在脚本退出码非 0 时，会把 stdout 装进 *RichError.Detail 而 output 返回 ""。
// 故必须从 RichError.Detail 二次提取，否则 reachable=false 的 JSON 永远拿不到。
func probeVDBReachable(ctx context.Context, instanceID, vdbEndpoint, vdbUsername, vdbApiKey string) (vdbReachability, string, error) {
	if vdbEndpoint == "" {
		return vdbReachabilityUnknown, "", hcommon.I18nError(i18n.MsgTDAIVDBInfoIncomplete)
	}

	runtimeUser := controller.LookupRuntimeUser(ctx, instanceID)
	output, err := taskRunScriptFn(ctx, instanceID, "precheck_vdb_connectivity.sh", 30, runtimeUser, nil,
		map[string]string{
			"vdb_endpoint": vdbEndpoint,
			"vdb_username": vdbUsername,
			"vdb_api_key":  vdbApiKey,
			"vdb_database": "",
			"timeout_sec":  "5",
		})

	// 合并 stdout：TAT 成功 → output 有值；TAT 失败（脚本退出码非 0）→ stdout 在 RichError.Detail
	stdout := output
	if stdout == "" {
		stdout = hcommon.ErrorDetailWithCtx(ctx, err)
	}

	// 用合并后的 stdout 判定结果，无论 err 是否为 nil 都先看脚本输出
	if strings.Contains(stdout, `"reachable":true`) {
		return vdbReachabilityReachable, stdout, nil
	}
	if strings.Contains(stdout, `"reachable":false`) {
		// 工具不可用 → 不视为"真不通"，但是合法判定（err 置 nil）
		if strings.Contains(stdout, `"probe":"none"`) || strings.Contains(stdout, `"no_probe_tool"`) {
			return vdbReachabilityUnknown, stdout, nil
		}
		return vdbReachabilityUnreachable, stdout, nil
	}

	// stdout 无合法 JSON：TAT 调用失败 / 脚本异常 / 输出被截断 等
	// 把 err 透传给上层，让上层按异常处理（切 Pro 走重试，切 OFF 仍保守按通处理）
	if err == nil {
		err = hcommon.I18nError(i18n.MsgTATScriptOutputAbnormal, strings.TrimSpace(stdout))
	}
	return vdbReachabilityUnknown, stdout, err
}

// precheckVDBConnectivity 切 Pro 前用：根据连通性结果决定是否允许切换。
//
//	网络通       → nil（继续 switch_pro）
//	网络明确不通 → NonRetryableError（dispatcher 直接终态 + 自动 rollback mem space）
//	工具不可用   → nil（保守通过，让 switch_pro 自己去试）
//	其他异常     → 普通 error（可重试）
//
// 网络不通时返回 NonRetryableError 的考量：
//
//	网络问题不会因为 retry 自愈，退避重试除了浪费时间和日志噪音外没有任何价值；
//	直接终态失败让用户/运维快速看到错误并去找网络组排查，是更优的体验。
func precheckVDBConnectivity(ctx context.Context, instanceID, vdbEndpoint, vdbUsername, vdbApiKey string) error {
	if vdbEndpoint == "" {
		return NewNonRetryableError(i18n.T(ctx, i18n.MsgTDAIVDBInfoIncompleteNonRetry))
	}

	state, output, err := probeVDBReachable(ctx, instanceID, vdbEndpoint, vdbUsername, vdbApiKey)
	switch state {
	case vdbReachabilityReachable:
		return nil
	case vdbReachabilityUnreachable:
		return NewNonRetryableError(i18n.T(ctx, i18n.MsgTDAIVDBNetworkUnreachable,
			vdbEndpoint, strings.TrimSpace(output)))
	case vdbReachabilityUnknown:
		if err != nil {
			return hcommon.I18nRichError(err, i18n.MsgTDAIVDBPrecheckScriptFailed)
		}
		// 脚本明确说工具不可用 → 保守通过
		slog.Warn("[VDBConnPrecheck] 无法判定连通性，保守通过",
			"instance_id", instanceID, "endpoint", vdbEndpoint, "output", output)
		return nil
	}
	return nil
}

// shouldSkipVDBExportOnDisable 切 OFF 前用：根据连通性结果决定 disable 时是否跳过 VDB 数据 export。
//
//	网络通       → false（正常 export）
//	网络明确不通 → true（跳过 export，避免必然超时阻断 OFF 流程）
//	工具不可用   → false（保守，让 export 自己去试）
//	TAT 异常     → false（保守，且单独 log 出来便于排查）
//
// 与 precheckVDBConnectivity 的差异：本函数永不阻断 OFF 流程（OFF 是"止损"动作，
// 不应因为预检环节本身不稳定而拒绝执行）；网络问题只影响 export 取舍，不影响清理。
func shouldSkipVDBExportOnDisable(ctx context.Context, instanceID, vdbEndpoint, vdbUsername, vdbApiKey string) bool {
	if vdbEndpoint == "" || vdbUsername == "" || vdbApiKey == "" {
		// 上游没传完整鉴权 → 让 disable 脚本自己决定（通常意味着不是 Pro 模式 / 历史脏数据）
		return false
	}

	state, output, err := probeVDBReachable(ctx, instanceID, vdbEndpoint, vdbUsername, vdbApiKey)
	switch state {
	case vdbReachabilityUnreachable:
		slog.Warn("[VDBConnPrecheck] CVM 与 VDB 网络不通，disable 时将跳过 VDB 数据 export",
			"instance_id", instanceID, "endpoint", vdbEndpoint, "output", strings.TrimSpace(output))
		return true
	case vdbReachabilityReachable:
		return false
	case vdbReachabilityUnknown:
		if err != nil {
			slog.Warn("[VDBConnPrecheck] 预检脚本调用异常，保守按 reachable 处理（仍尝试 export）",
				"instance_id", instanceID, "endpoint", vdbEndpoint, "error", err)
		} else {
			slog.Info("[VDBConnPrecheck] 探测工具不可用，保守按 reachable 处理（仍尝试 export）",
				"instance_id", instanceID, "endpoint", vdbEndpoint, "output", strings.TrimSpace(output))
		}
		return false
	}
	return false
}

func handleDeleteMemSpace(ctx context.Context, plugin *model.MemoryTDAIPlugin) error {
	log := slog.Default().With("instance_id", plugin.InstanceID, "pool_id", plugin.PoolID)
	if plugin.PoolID == "" {
		log.Info("[SwitchToOff] 无 Pro 绑定信息，跳过释放记忆库")
		return nil
	}

	client, err := newAgentMemoryClientFn(ctx)
	if err != nil {
		log.Error("[SwitchToOff] 初始化 Agent Memory SDK 失败",
			"instance_id", plugin.InstanceID, "pool_id", plugin.PoolID, "error", err)
		return hcommon.I18nRichError(err, i18n.MsgTDAISDKInitFailed)
	}

	_, err = client.DeleteMemSpace(ctx, &sdk.DeleteMemSpaceRequest{
		SpaceId: plugin.PoolID,
	})
	if err != nil {
		log.Error("[SwitchToOff] 释放记忆库失败",
			"instance_id", plugin.InstanceID, "pool_id", plugin.PoolID, "error", err)
		return hcommon.I18nRichError(err, i18n.MsgTDAIDeleteMemSpaceFailed)
	}

	log.Info("[SwitchToOff] 记忆库已释放",
		"instance_id", plugin.InstanceID,
		"pool_id", plugin.PoolID,
		"database", plugin.DatabaseName,
	)
	return nil
}

// rollbackProMemSpace 在 SWITCH_TO_PRO 最终失败时，释放已分配的远端记忆库并清理本地绑定信息，
// 避免 pro_capacity.used 泄漏。
// 仅在 current_plan 仍不是 PRO（即切换没成功）且 pool_id 非空时才回滚。
func rollbackProMemSpace(ctx context.Context, instanceID string) {
	log := slog.Default().With("instance_id", instanceID)
	plugin := model.GetMemoryTDAIPlugin(ctx, instanceID)
	if plugin == nil || plugin.PoolID == "" {
		return
	}
	// 若 current_plan 已变为 PRO，说明切换实际成功了（不应回滚）
	if plugin.CurrentPlan == model.MemoryPlanPro {
		return
	}

	log.Info("[RollbackProMemSpace] 切换 PRO 失败，开始回滚远端记忆库",
		"pool_id", plugin.PoolID)

	client, err := newAgentMemoryClientFn(ctx)
	if err != nil {
		log.Error("[RollbackProMemSpace] 初始化 SDK 失败，远端记忆库未释放，需人工清理",
			"pool_id", plugin.PoolID, "error", err)
		return
	}

	_, err = client.DeleteMemSpace(ctx, &sdk.DeleteMemSpaceRequest{
		SpaceId: plugin.PoolID,
	})
	if err != nil {
		log.Error("[RollbackProMemSpace] 释放远端记忆库失败，需人工清理",
			"pool_id", plugin.PoolID, "error", err)
		return
	}

	// 清理本地绑定信息
	model.DB(ctx).Model(plugin).Updates(map[string]any{
		"pool_id":            "",
		"database_name":      "",
		"endpoint":           "",
		"api_key_secret_ref": "",
		"vdb_username":       "",
		"embedding_model":    "",
	})

	log.Info("[RollbackProMemSpace] 远端记忆库已释放，本地绑定已清理",
		"pool_id", plugin.PoolID)
}
