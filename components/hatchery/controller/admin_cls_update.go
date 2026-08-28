package controller

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// CLSPluginVersionV1 是旧版 CLS 插件版本（无 trace 配置）。
const CLSPluginVersionV1 = "1.0"

// CLSPluginVersionUpdating 是 CLS 插件正在升级中的标识位。
// 升级开始时写入此值，成功后写为 2.0，失败后回滚为 1.0。
// 查询目标实例时会过滤掉此状态，防止并发重复触发。
const CLSPluginVersionUpdating = "updating"

// CLSPluginVersionV2 是新版 CLS 插件版本（含 trace 配置）。
const CLSPluginVersionV2 = "2.0"

// clsUpdateRequest 是 /admin/cls/update 接口的请求体。
type clsUpdateRequest struct {
	ScopeType string `json:"scope_type"` // 采集范围模式: "all"=全量, "group"=分组
	GroupIDs  []uint `json:"group_ids"`  // 分组 ID 列表，scope_type="group" 时生效
}

// HandleAdminUpdateCLSPlugin 对已安装 CLS Agent 的实例执行插件升级检查。
//
// 无论 scope_type 为 "all" 还是 "group"，均升级当前用户下全部已安装 CLS Agent 的机器。
//
// 流程：
//  1. 校验 CLS 服务已开通
//  2. 查询所有已安装 CLS Agent 且版本非 2.0 的实例
//  3. 对每个目标实例：
//     - cls_plugin_version=2.0 → 跳过
//     - cls_plugin_version=1.0 → 下发 cls_check_trace.sh 检查 openclaw.json
//     - configured=true（enabled=true 且 traceTopicId 非空）→ 仅更新版本为 2.0
//     - configured=false → 下发 cls_plugin_reinstall.sh，仅通过 npx 卸载并重新安装
//     clawpro-diagnostics-metrics-cls-onboard-cli 插件（不动 loglistener 本体），
//     最后更新版本为 2.0
//
// 请求体（scope_type 和 group_ids 字段保留兼容，但不影响升级范围）：
//
//	{ "scope_type": "all" | "group" }
//
// 响应：
//
//	{
//	  "ok": true,
//	  "total": 10,
//	  "skipped": 3,
//	  "upgraded": 5,
//	  "reinstalled": 2,
//	  "failed": 0
//	}
//
// clsQueryRunner 是查询待升级实例的函数类型，用于在测试中注入 mock 实现。
type clsQueryRunner func(ctx context.Context, req clsUpdateRequest) ([]model.Instance, error)

// clsClawRunner 是获取 CLS 服务信息的函数类型，用于在测试中注入 mock 实现。
type clsClawRunner func(ctx context.Context) (*CLSClawServiceResult, error)

// defaultCLSQueryRunner 是生产环境使用的默认查询器。
func defaultCLSQueryRunner(ctx context.Context, req clsUpdateRequest) ([]model.Instance, error) {
	return queryCLSUpdateTargetInstances(ctx, req)
}

// defaultCLSClawRunner 是生产环境使用的默认 CLS 服务信息获取器。
func defaultCLSClawRunner(ctx context.Context) (*CLSClawServiceResult, error) {
	client, err := newCLSCommonClient(ctx)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgCLSCreateClientFail)
	}
	return openClawService(client)
}

func HandleAdminUpdateCLSPlugin(w http.ResponseWriter, r *http.Request) {
	handleAdminUpdateCLSPluginWithRunners(w, r, defaultCLSQueryRunner, defaultCLSClawRunner)
}

// handleAdminUpdateCLSPluginWithRunners 是 HandleAdminUpdateCLSPlugin 的可测试版本，
// 支持注入自定义查询器和 CLS 服务信息获取器。
func handleAdminUpdateCLSPluginWithRunners(w http.ResponseWriter, r *http.Request, queryRunner clsQueryRunner, clawRunner clsClawRunner) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	config := model.GetSiteConfig(r.Context())
	if config.CLSEnabled != 1 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgCLSServiceNotEnabled))
		return
	}

	var req clsUpdateRequest
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgBadRequest))
		return
	}

	// 无论 scope_type 为何值，均升级全部机器，忽略 group_ids
	req.GroupIDs = nil

	// 查询目标实例（已安装 CLS Agent 的实例）
	instances, err := queryRunner(r.Context(), req)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryInstanceFailed))
		return
	}

	if len(instances) == 0 {
		jsonOK(w, map[string]interface{}{
			"ok":          true,
			"total":       0,
			"skipped":     0,
			"upgraded":    0,
			"reinstalled": 0,
			"failed":      0,
			"message":     "没有需要更新的实例",
		})
		return
	}

	// 获取 CLS 服务信息（Trace 主题 ID）
	clawResult, err := clawRunner(r.Context())
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgFailedToRetrieveCLSServiceInfo))
		return
	}

	slog.Info("[CLS Update] 开始批量升级 CLS 插件",
		"total", len(instances),
		"traceTopicId", clawResult.TraceTopicId,
	)

	// 使用 detached context 执行升级，避免客户端提前断开连接导致请求 ctx 取消，
	// 进而中断正在进行的 TAT 脚本执行或数据库写入，使实例版本永久卡在 "updating" 状态。
	upgradeCtx := hcommon.DetachContext(r.Context())
	stats := runCLSPluginUpdate(upgradeCtx, instances, clawResult)

	jsonOK(w, map[string]interface{}{
		"ok":          true,
		"total":       stats.total,
		"skipped":     stats.skipped,
		"upgraded":    stats.upgraded,
		"reinstalled": stats.reinstalled,
		"failed":      stats.failed,
	})
}

// clsUpdateStats 记录升级统计数据。
type clsUpdateStats struct {
	total       int
	skipped     int64
	upgraded    int64
	reinstalled int64
	failed      int64
}

// clsUpdatingTimeout 是 updating 状态的超时时间。
// 超过此时间仍处于 updating 的实例视为上次执行异常，重新纳入本次升级范围。
const clsUpdatingTimeout = 10 * time.Minute

// clsQueryBatchSize 是查询/更新实例时每批次处理的最大条数，避免单次 SQL 影响数据库性能。
const clsQueryBatchSize = 100

// queryCLSUpdateTargetInstances 查询所有已安装 CLS Agent 且版本非 2.0 的实例。
// 过滤掉 cls_plugin_version=2.0（已升级）的实例。
// 对于 cls_plugin_version='updating' 的实例，仅过滤 cls_agent_status_at 在超时时间内的，
// 超时的 updating 实例视为上次执行异常，重新纳入本次升级范围（先回滚为 1.0 再处理）。
// 升级范围始终为全部机器，不受 scope_type/group_ids 影响。
// 查询和回滚均采用分批次操作（每批 clsQueryBatchSize 条），避免单次 SQL 影响数据库性能。
func queryCLSUpdateTargetInstances(ctx context.Context, _ clsUpdateRequest) ([]model.Instance, error) {
	// 超时阈值：cls_agent_status_at 早于此时间的 updating 实例视为超时
	timeoutThreshold := time.Now().Add(-clsUpdatingTimeout)

	// 分批回滚超时的 updating 实例为 1.0，使其重新进入待处理队列。
	// 仅回滚 cls_agent_status_at 有值且已超时的实例，
	// cls_agent_status_at 为 NULL 的实例说明尚未记录开始时间，不视为超时。
	if err := batchRollbackTimedOutUpdating(ctx, timeoutThreshold); err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgCLSUpdateRollbackTimeoutFailed)
	}

	// 分批查询待升级实例，汇总后返回
	return batchQueryCLSTargetInstances(ctx)
}

// batchRollbackTimedOutUpdating 分批将超时的 updating 实例回滚为 1.0。
// 每批最多处理 clsQueryBatchSize 条，直到没有更多超时实例为止。
func batchRollbackTimedOutUpdating(ctx context.Context, timeoutThreshold time.Time) error {
	for {
		// 先查出一批超时实例的 instance_id，再按 ID 批量更新，避免全表扫描加锁
		var ids []string
		if err := model.DB(ctx).Model(&model.Instance{}).
			Select("instance_id").
			Where("cls_plugin_version = ? AND cls_agent_status_at IS NOT NULL AND cls_agent_status_at < ?",
				CLSPluginVersionUpdating, timeoutThreshold).
			Limit(clsQueryBatchSize).
			Pluck("instance_id", &ids).Error; err != nil {
			return err
		}
		if len(ids) == 0 {
			break
		}
		if err := model.DB(ctx).Model(&model.Instance{}).
			Where("instance_id IN ?", ids).
			Updates(map[string]interface{}{
				"cls_plugin_version":  CLSPluginVersionV1,
				"cls_agent_status_at": nil,
			}).Error; err != nil {
			return err
		}
	}
	return nil
}

// batchQueryCLSTargetInstances 分批查询所有待升级实例（版本非 2.0 且非 updating）。
// 使用 LIMIT + 游标（instance_id > lastID）分页，避免 OFFSET 随页数增大而性能下降。
func batchQueryCLSTargetInstances(ctx context.Context) ([]model.Instance, error) {
	var all []model.Instance
	lastID := uint(0)
	for {
		var batch []model.Instance
		if err := model.DB(ctx).Model(&model.Instance{}).
			Where("cls_agent_status = ? AND instance_id IS NOT NULL AND instance_id != '' AND is_doctor_node = ? AND cls_plugin_version NOT IN ? AND last_cvm_state = ? AND id > ?",
				model.CLSAgentInstalled, false, []string{CLSPluginVersionV2, CLSPluginVersionUpdating}, "RUNNING", lastID).
			Order("id ASC").
			Limit(clsQueryBatchSize).
			Find(&batch).Error; err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}
		all = append(all, batch...)
		lastID = batch[len(batch)-1].ID
		if len(batch) < clsQueryBatchSize {
			break
		}
	}
	return all, nil
}

// traceCheckResult 是 cls_check_trace.sh 脚本的输出结果。
type traceCheckResult struct {
	TraceEnabled bool   `json:"trace_enabled"`
	TraceTopicID string `json:"trace_topic_id"`
	Configured   bool   `json:"configured"`
	Reason       string `json:"reason"`
}

// clsScriptRunner 是执行 TAT 脚本的函数类型，用于在测试中注入 mock 实现。
type clsScriptRunner func(ctx context.Context, instanceID, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error)

// defaultCLSScriptRunner 是生产环境使用的默认脚本执行器，直接调用 RunScript。
func defaultCLSScriptRunner(ctx context.Context, instanceID, scriptName string, timeout uint64, runtimeUser string, onOutput func(chunk string), params map[string]string) (string, error) {
	return RunScript(ctx, instanceID, scriptName, timeout, runtimeUser, onOutput, params)
}

// clsFilterRunner 是过滤 CVM 运行状态的函数类型，用于在测试中注入 mock 实现。
type clsFilterRunner func(ctx context.Context, instanceIDs []string) ([]string, error)

// defaultCLSFilterRunner 是生产环境使用的默认过滤器，通过 CVM API 查询运行中的实例。
func defaultCLSFilterRunner(ctx context.Context, instanceIDs []string) ([]string, error) {
	cvmClient, err := GetCVMClient(ctx)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgCreateCVMClientFailed)
	}
	return FilterInstancesByState(cvmClient, instanceIDs, "RUNNING")
}

// runCLSPluginUpdate 并发对实例执行升级检查，最大并发数 5。
// 执行前先通过 CVM API 过滤非运行中的实例，非 RUNNING 实例直接跳过。
func runCLSPluginUpdate(ctx context.Context, instances []model.Instance, clawResult *CLSClawServiceResult) *clsUpdateStats {
	return runCLSPluginUpdateWithRunner(ctx, instances, clawResult, defaultCLSScriptRunner, defaultCLSFilterRunner)
}

// runCLSPluginUpdateWithRunner 是 runCLSPluginUpdate 的可测试版本，支持注入自定义脚本执行器和过滤器。
func runCLSPluginUpdateWithRunner(ctx context.Context, instances []model.Instance, clawResult *CLSClawServiceResult, runner clsScriptRunner, filter clsFilterRunner) *clsUpdateStats {
	stats := &clsUpdateStats{total: len(instances)}

	// 批量检查 CVM 运行状态，过滤非 RUNNING 实例
	instanceIDs := make([]string, 0, len(instances))
	for _, inst := range instances {
		instanceIDs = append(instanceIDs, inst.InstanceId)
	}

	runningIDs, err := filter(ctx, instanceIDs)
	if err != nil {
		slog.Error("[CLS Update] 查询 CVM 运行状态失败，跳过本次升级", "error", err)
		atomic.AddInt64(&stats.failed, int64(len(instances)))
		return stats
	}

	runningSet := make(map[string]struct{}, len(runningIDs))
	for _, id := range runningIDs {
		runningSet[id] = struct{}{}
	}

	// 过滤出运行中的实例，非运行中的直接跳过
	runningInstances := make([]model.Instance, 0, len(runningIDs))
	for _, inst := range instances {
		if _, ok := runningSet[inst.InstanceId]; ok {
			runningInstances = append(runningInstances, inst)
		} else {
			slog.Info("[CLS Update] 实例非运行中，跳过",
				"instance_id", inst.InstanceId,
				"name", inst.Name,
			)
			atomic.AddInt64(&stats.skipped, 1)
		}
	}

	if len(runningInstances) == 0 {
		slog.Info("[CLS Update] 无运行中的实例需要升级")
		return stats
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, 5)

	for _, inst := range runningInstances {
		wg.Add(1)
		go func(inst model.Instance) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			runSingleCLSPluginUpdate(ctx, inst, clawResult, stats, runner, filter)
		}(inst)
	}
	wg.Wait()

	slog.Info("[CLS Update] 批量升级完成",
		"total", stats.total,
		"skipped", stats.skipped,
		"upgraded", stats.upgraded,
		"reinstalled", stats.reinstalled,
		"failed", stats.failed,
	)
	return stats
}

// runSingleCLSPluginUpdate 对单个实例执行升级检查逻辑。
func runSingleCLSPluginUpdate(ctx context.Context, inst model.Instance, clawResult *CLSClawServiceResult, stats *clsUpdateStats, runner clsScriptRunner, filter clsFilterRunner) {
	log := slog.With("instance_id", inst.InstanceId, "name", inst.Name)

	// 已是 2.0 版本，跳过（理论上查询已过滤，此处作为双重保险）
	if inst.CLSPluginVersion == CLSPluginVersionV2 {
		log.Info("[CLS Update] 插件版本已是 2.0，跳过")
		atomic.AddInt64(&stats.skipped, 1)
		return
	}

	// CAS：将版本从 1.0（或空字符串，新安装实例未写入版本）改为 updating，
	// 同时记录开始时间用于超时回退。
	// 使用 WHERE cls_plugin_version IN ('1.0', '') 作为条件，确保原子性。
	now := time.Now()
	result := model.DB(ctx).Model(&model.Instance{}).
		Where("instance_id = ? AND cls_plugin_version IN ?", inst.InstanceId, []string{CLSPluginVersionV1, ""}).
		Updates(map[string]interface{}{
			"cls_plugin_version":  CLSPluginVersionUpdating,
			"cls_agent_status_at": now,
		})
	if result.Error != nil {
		log.Error("[CLS Update] 标记 updating 状态失败", "error", result.Error)
		atomic.AddInt64(&stats.failed, 1)
		return
	}
	if result.RowsAffected == 0 {
		// 已被其他并发请求抢先处理，跳过
		log.Info("[CLS Update] 实例已被其他请求处理，跳过")
		atomic.AddInt64(&stats.skipped, 1)
		return
	}

	// CAS 成功后再次确认实例仍为 RUNNING 状态。
	// 从查询到 CAS 之间存在时间窗口，实例可能已停机或重启，
	// 提前检测可避免无效的脚本下发，减少因脚本失败触发 rollback 的概率。
	runningIDs, filterErr := filter(ctx, []string{inst.InstanceId})
	if filterErr != nil {
		log.Warn("[CLS Update] 二次确认实例状态失败，继续尝试下发脚本", "error", filterErr)
	} else if len(runningIDs) == 0 {
		log.Info("[CLS Update] 实例已不在运行中（CAS 后状态变化），回滚并跳过")
		if rbErr := rollbackCLSPluginVersion(ctx, inst.InstanceId); rbErr != nil {
			log.Error("[CLS Update] 回滚插件版本失败", "error", rbErr)
		}
		atomic.AddInt64(&stats.skipped, 1)
		return
	}

	// 下发检查脚本，查看 openclaw.json 中 trace 配置
	output, err := runner(ctx, inst.InstanceId, "cls_check_trace.sh", 10, inst.RuntimeUser, nil, nil)
	if err != nil {
		log.Error("[CLS Update] 下发 trace 检查脚本失败", "error", err)
		if rbErr := rollbackCLSPluginVersion(ctx, inst.InstanceId); rbErr != nil {
			log.Error("[CLS Update] 回滚插件版本失败", "error", rbErr)
		}
		atomic.AddInt64(&stats.failed, 1)
		return
	}

	// 解析脚本输出（取最后一行非空 JSON）
	checkResult, err := parseTraceCheckOutput(output)
	if err != nil {
		log.Error("[CLS Update] 解析 trace 检查结果失败", "output", output, "error", err)
		if rbErr := rollbackCLSPluginVersion(ctx, inst.InstanceId); rbErr != nil {
			log.Error("[CLS Update] 回滚插件版本失败", "error", rbErr)
		}
		atomic.AddInt64(&stats.failed, 1)
		return
	}

	log.Info("[CLS Update] trace 检查结果",
		"configured", checkResult.Configured,
		"trace_enabled", checkResult.TraceEnabled,
		"trace_topic_id", checkResult.TraceTopicID,
	)

	if checkResult.Configured {
		// trace 已配置（enabled=true 且 traceTopicId 非空），仅更新版本为 2.0
		if err := updateCLSPluginVersion(ctx, inst.InstanceId, CLSPluginVersionV2); err != nil {
			log.Error("[CLS Update] 更新插件版本失败", "error", err)
			if rbErr := rollbackCLSPluginVersion(ctx, inst.InstanceId); rbErr != nil {
				log.Error("[CLS Update] 回滚插件版本失败", "error", rbErr)
			}
			atomic.AddInt64(&stats.failed, 1)
			return
		}
		log.Info("[CLS Update] trace 已配置，仅更新版本为 2.0")
		atomic.AddInt64(&stats.upgraded, 1)
		return
	}

	// trace 未配置：仅通过 npx 卸载并重新安装插件（不动 loglistener 本体）
	log.Info("[CLS Update] trace 未配置，执行 npx 插件卸载后重新安装")

	params := map[string]string{
		"region":          CVMRegion,
		"role_name":       AgentCamRoleName,
		"metric_topic_id": clawResult.MetricTopicId,
		"trace_topic_id":  clawResult.TraceTopicId,
	}
	if _, err := runner(ctx, inst.InstanceId, "cls_plugin_reinstall.sh", 120, inst.RuntimeUser, nil, params); err != nil {
		log.Error("[CLS Update] npx 插件重新安装失败", "error", err)
		if rbErr := rollbackCLSPluginVersion(ctx, inst.InstanceId); rbErr != nil {
			log.Error("[CLS Update] 回滚插件版本失败", "error", rbErr)
		}
		atomic.AddInt64(&stats.failed, 1)
		return
	}

	// 安装成功，更新版本为 2.0
	if err := updateCLSPluginVersion(ctx, inst.InstanceId, CLSPluginVersionV2); err != nil {
		log.Error("[CLS Update] 更新插件版本失败", "error", err)
		if rbErr := rollbackCLSPluginVersion(ctx, inst.InstanceId); rbErr != nil {
			log.Error("[CLS Update] 回滚插件版本失败", "error", rbErr)
		}
		atomic.AddInt64(&stats.failed, 1)
		return
	}

	log.Info("[CLS Update] 重新安装成功，版本更新为 2.0")
	atomic.AddInt64(&stats.reinstalled, 1)
}

// parseTraceCheckOutput 从脚本输出中解析 traceCheckResult。
// 取最后一行非空内容作为 JSON 解析。
func parseTraceCheckOutput(output string) (*traceCheckResult, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var lastLine string
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			lastLine = line
			break
		}
	}
	if lastLine == "" {
		return nil, hcommon.I18nError(i18n.MsgCLSUpdateScriptOutputEmpty)
	}

	var result traceCheckResult
	if err := json.Unmarshal([]byte(lastLine), &result); err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgCLSUpdateJSONParseFailed)
	}
	return &result, nil
}

// updateCLSPluginVersion 更新指定实例的 CLS 插件版本，同时清空 cls_agent_status_at。
// 使用 common.DetachContext 派生独立 ctx，从请求 ctx 中复制 TenantSnapshot（保证租户隔离），
// 同时脱离请求 ctx 的取消链，确保在请求 ctx 已取消（如客户端超时断开）
// 的情况下仍能正确写入数据库，避免版本永久卡在 "updating" 状态。
func updateCLSPluginVersion(ctx context.Context, instanceID string, version string) error {
	return model.DB(hcommon.DetachContext(ctx)).Model(&model.Instance{}).
		Where("instance_id = ?", instanceID).
		Updates(map[string]interface{}{
			"cls_plugin_version":  version,
			"cls_agent_status_at": nil,
		}).Error
}

// rollbackCLSPluginVersion 将实例的 CLS 插件版本回滚为 1.0，并清空 cls_agent_status_at，
// 允许下次重试。
// 使用 common.DetachContext 派生独立 ctx，从请求 ctx 中复制 TenantSnapshot（保证租户隔离），
// 同时脱离请求 ctx 的取消链，确保在请求 ctx 已取消（如客户端超时断开）
// 的情况下仍能正确回滚，避免版本永久卡在 "updating" 状态。
func rollbackCLSPluginVersion(ctx context.Context, instanceID string) error {
	return model.DB(hcommon.DetachContext(ctx)).Model(&model.Instance{}).
		Where("instance_id = ?", instanceID).
		Updates(map[string]interface{}{
			"cls_plugin_version":  CLSPluginVersionV1,
			"cls_agent_status_at": nil,
		}).Error
}

// clsPluginVersionItem 是单个实例的 CLS 插件版本信息。
type clsPluginVersionItem struct {
	InstanceID       string `json:"instance_id"`
	Name             string `json:"name"`
	CLSPluginVersion string `json:"cls_plugin_version"`
}

// HandleAdminGetCLSUpdateStats 查询 CLS 插件版本分布统计及各实例版本明细。
//
// 响应：
//
//	{
//	  "ok": true,
//	  "v1_count": 5,
//	  "v2_count": 10,
//	  "instances": [
//	    {"instance_id": "ins-xxx", "name": "实例名", "cls_plugin_version": "1.0"},
//	    ...
//	  ]
//	}
func HandleAdminGetCLSUpdateStats(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	var instances []model.Instance
	if err := model.DB(r.Context()).Model(&model.Instance{}).
		Select("instance_id, name, cls_plugin_version").
		Where("cls_agent_status = ? AND instance_id != '' AND is_doctor_node = ? AND last_cvm_state = ?",
			model.CLSAgentInstalled, false, "RUNNING").
		Find(&instances).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgFailedToQueryInstanceVersion))
		return
	}

	var v1Count, v2Count int64
	items := make([]clsPluginVersionItem, 0, len(instances))
	for _, inst := range instances {
		v := inst.CLSPluginVersion
		if v == "" {
			v = CLSPluginVersionV1
		}
		switch v {
		case CLSPluginVersionV2:
			v2Count++
		default:
			// 1.0 和 updating 都归入 v1_count（updating 表示正在升级中）
			v1Count++
		}
		items = append(items, clsPluginVersionItem{
			InstanceID:       inst.InstanceId,
			Name:             inst.Name,
			CLSPluginVersion: v,
		})
	}

	jsonOK(w, map[string]interface{}{
		"ok":        true,
		"v1_count":  v1Count,
		"v2_count":  v2Count,
		"instances": items,
	})
}
