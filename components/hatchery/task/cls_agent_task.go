package task

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	hcommon "hatchery/common"
	"hatchery/controller"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	"gorm.io/gorm"
)

// CLSAgentStaleTimeout 中间状态超时阈值：超过此时间仍处于"安装中/卸载中"的实例
// 会被自动回退到初始状态，防止服务重启后状态卡死。
const CLSAgentStaleTimeout = 10 * time.Minute

// processStartTime 记录当前进程启动时间。
// 用于服务重启，加速回退状态，无需等待 CLSAgentStaleTimeout
var processStartTime = time.Now()

// CLSAgentNotRunningCooldown 非运行实例冷却时间：实例不在运行中时，
// 仅更新 cls_agent_status_at 作为"最近已检查"标记，在冷却期内不再重复捞取，
// 超过冷却期后会自动重新被调度检查。
const CLSAgentNotRunningCooldown = 5 * time.Minute

// defaultBatchLimit 每轮轮询最多处理的实例数
const defaultBatchLimit = 50

// getCLSInterval 获取轮询间隔（秒），通过环境变量 CLS_TASK_INTERVAL 配置，默认 60。
func getCLSInterval() int {
	if v := os.Getenv("CLS_TASK_INTERVAL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
		slog.Warn("[CLS Agent] 环境变量 CLS_TASK_INTERVAL 值无效，使用默认值", "value", v, "default", 60)
	}
	return 60
}

// getCLSBatchLimit 获取每轮处理的批次大小
func getCLSBatchLimit() int {
	if v := os.Getenv("CLS_BATCH_LIMIT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
		slog.Warn("[CLS Agent] 环境变量 CLS_BATCH_LIMIT 值无效，使用默认值", "value", v, "default", defaultBatchLimit)
	}
	return defaultBatchLimit
}

// clsAgentPollCount 记录 CLS Agent 轮询次数
var clsAgentPollCount tenantInt64

// describeInstancesWithoutAgentRole 批量查询 CVM 实例，返回四组结果：
//   - unboundIds:  **运行中**且**未绑定** AgentCamRoleName 角色的实例 ID 列表
//   - runningIds:  所有**运行中**的实例 ID 列表
//   - skipIds:     需要跳过的实例 ID 列表（已释放 RELEASED + CVM API 查不到的实例）
//   - pendingIds:  正在创建中（PENDING）的实例 ID 列表，不应进入冷却队列
//
// 只查询运行中（RUNNING）的机器；已关机、已停止等非运行状态的实例会被跳过。
// 已释放（RELEASED）和 CVM API 完全查不到（已彻底删除/过期/不存在）的实例
// 会合并收集到 skipIds，供调用方统一标记为跳过，不再尝试安装/卸载。
// 创建中（PENDING）的实例会单独收集到 pendingIds，调用方应忽略这些实例，
// 既不进入冷却队列也不标记跳过，下一轮轮询时自然被重新捞取。
// CVM DescribeInstances 单次最多查 100 个实例，此函数自动分批查询。
func describeInstancesWithoutAgentRole(ctx context.Context, instanceIds []string) (unboundIds []string, runningIds []string, skipIds []string, pendingIds []string, err error) {
	if len(instanceIds) == 0 {
		return nil, nil, nil, nil, nil
	}

	client, rerr := controller.NewCVMClient(ctx)
	if rerr != nil {
		return nil, nil, nil, nil, rerr
	}

	// 记录 API 实际返回的所有实例 ID，用于最终比对找出"消失"的实例
	returnedSet := make(map[string]struct{})

	// DescribeInstances 单次最多查 100 个
	const batchSize = 100
	for i := 0; i < len(instanceIds); i += batchSize {
		end := i + batchSize
		if end > len(instanceIds) {
			end = len(instanceIds)
		}
		batch := instanceIds[i:end]

		req := cvm.NewDescribeInstancesRequest()
		req.InstanceIds = common.StringPtrs(batch)
		req.Limit = common.Int64Ptr(int64(len(batch))) // 显式设置 Limit，CVM API 默认 Limit=20 会导致截断

		resp, err := client.DescribeInstances(req)
		if err != nil {
			return nil, nil, nil, nil, hcommon.I18nRichError(err, i18n.MsgDescribeInstancesFailed)
		}

		if resp.Response != nil {
			for _, inst := range resp.Response.InstanceSet {
				if inst.InstanceId == nil {
					continue
				}

				returnedSet[*inst.InstanceId] = struct{}{}

				// 已释放的实例归入 skipIds，不再尝试安装/卸载
				if inst.InstanceState != nil && *inst.InstanceState == "RELEASED" {
					skipIds = append(skipIds, *inst.InstanceId)
					continue
				}

				// 创建中的实例归入 pendingIds，不进入冷却队列，下一轮自然重新捞取
				if inst.InstanceState != nil && *inst.InstanceState == "PENDING" {
					pendingIds = append(pendingIds, *inst.InstanceId)
					continue
				}

				// 只处理运行中的实例，跳过已关机/已停止等非运行状态
				if inst.InstanceState == nil || *inst.InstanceState != "RUNNING" {
					continue
				}
				runningIds = append(runningIds, *inst.InstanceId)
				// 未绑定角色 或 角色名不匹配 → 需要绑定
				if inst.CamRoleName == nil || *inst.CamRoleName != controller.AgentCamRoleName {
					unboundIds = append(unboundIds, *inst.InstanceId)
				}
			}
		}
	}

	// CVM API 完全查不到的实例（已彻底删除/过期/不存在）也归入 skipIds
	for _, id := range instanceIds {
		if _, found := returnedSet[id]; !found {
			skipIds = append(skipIds, id)
		}
	}

	return unboundIds, runningIds, skipIds, pendingIds, nil
}

func init() {
	// 通过环境变量 ENABLE_CLS_TASK 控制是否启动 CLS Agent 任务，默认启用。
	// 设置为 "false" / "0" / "off" 可关闭。
	if v := os.Getenv("ENABLE_CLS_TASK"); v != "" {
		switch strings.ToLower(v) {
		case "false", "0", "off":
			return
		}
	}

	RegisterTask(TaskDef{
		Name:         "cls-agent",
		Interval:     time.Duration(getCLSInterval()) * time.Second,
		RunFunc:      safeCLSAgentTask,
		NeedDistLock: false, // 内部自行 TryLock
		PerTenant:    true,
		InitialDelay: 30 * time.Second,
	})
}

// safeCLSAgentTask 包装 runCLSAgentTask，记录轮询计数并执行。
// panic recovery 已由调度器（executeTask）统一处理。
func safeCLSAgentTask(ctx context.Context) {
	identifier := model.CurrentIdentifier(ctx)
	n := clsAgentPollCount.Add(identifier, 1)
	log := controller.Logger(ctx)
	log.Info("[CLS Agent] 轮询执行中...", "round", n)
	runCLSAgentTask(ctx)
}

// runCLSAgentTask 检查 CLS 服务状态，根据结果执行安装或卸载。
// MySQL 多实例部署时通过分布式锁保证同一时刻只有一个节点执行，
// SQLite 模式下锁为空操作，行为不变。
func runCLSAgentTask(ctx context.Context) {
	// ---- 分布式锁：保证多实例部署时只有一个节点执行 ----

	lock, err := model.TryLock(ctx, "cls-agent-task")
	if err != nil {
		slog.Info("[CLS Agent] 未获取到分布式锁，其他节点正在执行，跳过本轮", "error", err)
		return
	}
	defer lock.Release()

	// ---- 前置检查：服务角色是否已绑定 ----
	// 通过 STS AssumeRole 检查 CVM_QCSLinkedRoleInClawProAgent 角色是否存在，
	// 若角色未绑定则跳过本轮，避免后续安装/绑定操作因权限不足而失败。
	hasRole, rerr := controller.CheckClawProAgentRoleBound(ctx)
	if rerr != nil {
		slog.Warn("[CLS Agent] 检查服务角色绑定状态失败，跳过本轮", "error", rerr)
		return
	}
	if !hasRole {
		slog.Warn("[CLS Agent] 服务角色 " + controller.AgentCamRoleName + " 未绑定，跳过本轮")
		return
	}
	slog.Info("[CLS Agent] 服务角色 " + controller.AgentCamRoleName + " 已绑定，继续执行")

	// 每轮开始前先重置超时的中间状态，防止服务重启导致状态卡死
	resetStaleCLSAgentStatus(ctx)

	// 通过 OpenClawService 接口检查 CLS 服务是否开通
	result, err := controller.CheckCLSClawServiceOpened(ctx)
	if err != nil {
		slog.Error("[CLS Agent] 查询 CLS 服务状态失败，跳过本轮", "error", err)
		return
	}

	if result != nil && result.MetricTopicId != "" && result.TopicId != "" && result.TraceTopicId != "" {
		// CLS 服务已开通且主题信息完整 → 执行安装逻辑
		runCLSAgentInstall(ctx, result)

		// scope 非空时，额外将已安装但不在 scope 范围内的实例标记为待卸载
		runCLSAgentScopeUninstall(ctx)
	} else if result == nil {
		// CLS 服务未开通 → 检查本地是否仍有已安装实例，有则自动卸载
		// 同步本地配置状态
		config := model.GetSiteConfig(ctx)
		if config.CLSEnabled == 1 {
			if err := model.UpdateSiteConfig(ctx, map[string]interface{}{"cls_enabled": 0}); err != nil {
				slog.Error("[CLS Agent] 更新本地 CLSEnabled 状态失败", "error", err)
			}
		}

		// 判断是否需要卸载：存在已安装实例 或 环境变量强制开启
		var installedCount int64
		model.DB(ctx).Model(&model.Instance{}).
			Where("cls_agent_status = ? AND instance_id != '' AND is_doctor_node = ? AND source = ?",
				model.CLSAgentInstalled, false, model.InstanceSourceCVM).
			Count(&installedCount)
		if installedCount > 0 || isCLSUninstallEnabled() {
			slog.Info("[CLS Agent] CLS 服务未开通，存在已安装实例，执行卸载", "installed_count", installedCount, "env_enabled", isCLSUninstallEnabled())
			runCLSAgentUninstall(ctx)
		}
	} else {
		slog.Info("[CLS Agent] CLS 主题信息不完整，跳过本轮", "metricTopicId", result.MetricTopicId, "topicId", result.TopicId, "traceTopicId", result.TraceTopicId)
	}
}

// runCLSAgentInstall 对未安装 CLS Agent 的实例分批执行安装。
// 每轮最多处理 batchLimit 条实例（由 CLS_BATCH_LIMIT 环境变量或 defaultBatchLimit 控制），
// 剩余实例将在后续轮次中继续处理。
//
// 感知 CLS collectScope 采集范围：
//   - scope 为空：全量扫描（向后兼容）
//   - scope 非空：仅处理 scope 内分组对应的实例
//
// 流程：
//  1. 分批查询 cls_agent_status=0 的实例（LIMIT batchLimit），按 scope 过滤
//  2. 调用 CVM DescribeInstances 筛选出未绑定 AgentCamRoleName 的实例
//  3. 对未绑定角色的实例调用 ModifyInstancesCamRole 异步绑定角色
//  4. 对所有实例批量执行安装脚本
func runCLSAgentInstall(ctx context.Context, result *controller.CLSClawServiceResult) {
	// 同步更新本地 CLSEnabled 状态
	if err := model.UpdateSiteConfig(ctx, map[string]interface{}{"cls_enabled": 1}); err != nil {
		slog.Error("[CLS Agent] 更新本地 CLSEnabled 状态失败", "error", err)
		return
	}

	batchLimit := getCLSBatchLimit()

	// 冷却截止时间：cls_agent_status_at 早于此时间的实例才会被捞取
	cooldownCutoff := time.Now().Add(-CLSAgentNotRunningCooldown)

	// 根据 CLSScopeMode 判断是否为分组模式
	config := model.GetSiteConfig(ctx)
	isGroupMode := config.CLSScopeMode == "group"

	// 查询 collectScope 内的实例 ID（用于过滤）
	scopeInstanceIDs, hasScopeFilter, err := model.GetCLSCollectScopeCVMInstanceIDs(ctx)
	if err != nil {
		slog.Error("[CLS Agent] 查询 CLS 采集范围实例失败", "error", err)
		return
	}

	// 分组模式下，即使 bindings 为空也视为 hasScopeFilter=true
	if isGroupMode {
		hasScopeFilter = true
	}

	// hasScopeFilter=true 且 len==0 表示分组模式下无匹配实例，跳过安装
	if hasScopeFilter && len(scopeInstanceIDs) == 0 {
		slog.Info("[CLS Agent] CLS 采集范围为分组模式但无对应实例，跳过本轮")
		return
	}

	if hasScopeFilter {
		slog.Info("[CLS Agent] CLS 采集范围已配置，仅处理范围内实例", "scope_instance_count", len(scopeInstanceIDs))
	}

	// 构建查询条件
	baseQuery := model.DB(ctx).Model(&model.Instance{}).
		Where("cls_agent_status = ? AND instance_id != '' AND is_doctor_node = ? AND source = ? AND (cls_agent_status_at IS NULL OR cls_agent_status_at < ?)",
			model.CLSAgentNotInstalled, false, model.InstanceSourceCVM, cooldownCutoff).
		// 跳过正在执行操作（重装/创建/升级等 processing 中）的实例：
		// 重装/升级期间会先把 cls_agent_status 置 0，但旧系统盘此刻可能仍在运行旧 loglistener，
		// 安装脚本预检 systemctl is-active loglistener 会命中并 exit 0，导致 cls_agent_status
		// 被错误标记为"已安装"，而随后机器被抹盘真正失去了 Agent，从此不再触发安装。
		// 等操作收敛（current_operation 被 agent_checker 清空、机器就绪）后再安装。
		Where("(current_operation_state IS NULL OR current_operation_state != ?)", model.OpStateProcessing)

	if hasScopeFilter {
		baseQuery = baseQuery.Where("instance_id IN ?", scopeInstanceIDs)
	}

	// 先查总待安装数用于日志展示
	var totalPending int64
	if err := baseQuery.Session(&gorm.Session{}).Count(&totalPending).Error; err != nil {
		slog.Error("[CLS Agent] 查询待安装实例总数失败", "error", err)
		return
	}

	if totalPending == 0 {
		slog.Info("[CLS Agent] 没有待安装的实例，跳过本轮")
		return
	}

	var instances []model.Instance
	if err := baseQuery.Session(&gorm.Session{}).Limit(batchLimit).Find(&instances).Error; err != nil {
		slog.Error("[CLS Agent] 查询待安装实例失败", "error", err)
		return
	}

	slog.Info("[CLS Agent] 本轮分批处理待安装实例",
		"totalPending", totalPending,
		"batchLimit", batchLimit,
		"batchSize", len(instances),
		"remaining", totalPending-int64(len(instances)),
	)

	// 收集所有实例 ID
	instanceIds := make([]string, 0, len(instances))
	for _, inst := range instances {
		instanceIds = append(instanceIds, inst.InstanceId)
	}

	// ---- Step 1: 查询未绑定 AgentCamRoleName 的实例（同时获取运行中实例列表、需跳过的实例列表和创建中的实例列表） ----
	unboundIds, runningIds, skipIds, pendingIds, err := describeInstancesWithoutAgentRole(ctx, instanceIds)
	if err != nil {
		slog.Error("[CLS Agent] 查询实例 CamRoleName 失败，跳过本轮", "error", err)
		return
	}

	// 将已释放/查不到的实例统一标记为"已跳过"，后续不再调度
	if len(skipIds) > 0 {
		if err := model.DB(ctx).Model(&model.Instance{}).
			Where("instance_id IN ?", skipIds).
			Updates(map[string]interface{}{
				"cls_agent_status":    model.CLSAgentSkipped,
				"cls_agent_status_at": time.Now(),
			}).Error; err != nil {
			slog.Error("[CLS Agent] 标记不可操作实例为跳过状态失败", "error", err)
		} else {
			slog.Info("[CLS Agent] 已标记不可操作实例为跳过状态（已释放/已删除），不再尝试安装",
				"skipCount", len(skipIds),
				"skipIds", strings.Join(skipIds, ", "),
			)
		}
	}

	// 创建中（PENDING）的实例仅记录日志，不进入冷却队列也不标记跳过，下一轮自然重新捞取
	if len(pendingIds) > 0 {
		slog.Info("[CLS Agent] 发现创建中（PENDING）的实例，跳过本轮处理，下轮将重新检查",
			"pendingCount", len(pendingIds),
			"pendingIds", strings.Join(pendingIds, ", "),
		)
	}

	// 构建运行中实例 ID 集合、已跳过实例 ID 集合和创建中实例 ID 集合，用于后续过滤
	runningSet := make(map[string]struct{}, len(runningIds))
	for _, id := range runningIds {
		runningSet[id] = struct{}{}
	}
	skipSet := make(map[string]struct{}, len(skipIds))
	for _, id := range skipIds {
		skipSet[id] = struct{}{}
	}
	pendingSet := make(map[string]struct{}, len(pendingIds))
	for _, id := range pendingIds {
		pendingSet[id] = struct{}{}
	}

	// 只保留运行中的实例；已跳过的实例已标记 CLSAgentSkipped，不进入冷却；
	// 创建中（PENDING）的实例直接忽略，不进入冷却队列，下一轮自然重新捞取；
	// 其余非运行状态的实例仅更新检查时间戳进入冷却
	var runningInstances []model.Instance
	var notRunningIDs []string
	for _, inst := range instances {
		if _, ok := runningSet[inst.InstanceId]; ok {
			runningInstances = append(runningInstances, inst)
		} else if _, skipped := skipSet[inst.InstanceId]; skipped {
			// 已跳过的实例，已在上面标记 CLSAgentSkipped
			continue
		} else if _, pending := pendingSet[inst.InstanceId]; pending {
			// 创建中的实例，不做任何处理，下一轮自然重新捞取
			continue
		} else {
			// 其他非运行实例进入冷却
			notRunningIDs = append(notRunningIDs, inst.InstanceId)
		}
	}

	// 对非运行实例更新 cls_agent_status_at，使其在冷却期内不再被捞取，
	// 冷却期结束后自动重新被调度检查（无需额外 reset 逻辑）
	if len(notRunningIDs) > 0 {
		if err := model.DB(ctx).Model(&model.Instance{}).
			Where("instance_id IN ?", notRunningIDs).
			Update("cls_agent_status_at", time.Now()).Error; err != nil {
			slog.Error("[CLS Agent] 更新非运行实例检查时间戳失败", "error", err)
		} else {
			slog.Info("[CLS Agent] 已标记非运行实例进入冷却期",
				"cooldown", CLSAgentNotRunningCooldown,
				"notRunningCount", len(notRunningIDs),
				"notRunningIds", strings.Join(notRunningIDs, ", "),
			)
		}
	}

	if len(runningInstances) == 0 {
		slog.Info("[CLS Agent] 本批无运行中的实例，已进入冷却或标记跳过或创建中，下轮将调度后续实例",
			"batchSize", len(instances),
			"skippedCount", len(skipIds),
			"pendingCount", len(pendingIds),
			"notRunningCount", len(notRunningIDs),
		)
		return
	}

	if len(runningInstances) < len(instances) {
		slog.Info("[CLS Agent] 部分实例不在运行中（已进入冷却或标记跳过或创建中），仅对运行中实例执行安装",
			"batchSize", len(instances),
			"runningCount", len(runningInstances),
			"skippedCount", len(skipIds),
			"pendingCount", len(pendingIds),
			"notRunningCount", len(notRunningIDs),
		)
	}

	// ---- Step 2: 对未绑定角色的实例异步绑定角色 ----
	if len(unboundIds) > 0 {
		slog.Info("[CLS Agent] 发现未绑定 AgentCamRoleName 的实例，开始异步绑定角色",
			"unboundCount", len(unboundIds),
			"runningCount", len(runningInstances),
			"unboundIds", strings.Join(unboundIds, ", "),
		)
		if err := controller.ModifyInstancesCamRole(ctx, unboundIds); err != nil {
			slog.Error("[CLS Agent] 批量绑定 CamRole 失败，跳过本轮", "error", err)
			return
		}
		slog.Info("[CLS Agent] 批量绑定 CamRole 成功", "count", len(unboundIds))
	} else {
		slog.Info("[CLS Agent] 所有运行中实例均已绑定 AgentCamRoleName，无需绑定角色")
	}

	// ---- Step 3: 对运行中的实例批量执行安装脚本 ----
	slog.Info("[CLS Agent] CLS 服务已开通，开始批量安装 CLS Agent",
		"count", len(runningInstances),
		"boundInThisRound", len(unboundIds),
		"metricTopicId", result.MetricTopicId,
		"topicId", result.TopicId,
	)

	// 批量将待安装实例标记为"安装中"，防止下一轮调度重复拉起安装
	runningInstIDs := make([]string, 0, len(runningInstances))
	for _, inst := range runningInstances {
		runningInstIDs = append(runningInstIDs, inst.InstanceId)
	}
	// 注：批量标记"安装中"失败不阻塞后续安装流程，仅记录日志
	now := time.Now()
	if err := model.DB(ctx).Model(&model.Instance{}).
		Where("instance_id IN ?", runningInstIDs).
		Updates(map[string]interface{}{
			"cls_agent_status":    model.CLSAgentInstalling,
			"cls_agent_status_at": now,
		}).Error; err != nil {
		slog.Error("[CLS Agent] 批量标记安装中状态失败", "error", err)
	}

	params := map[string]string{
		"region":          controller.CVMRegion,
		"role_name":       controller.AgentCamRoleName,
		"metric_topic_id": result.MetricTopicId,
		"trace_topic_id":  result.TraceTopicId,
	}

	batchRunScript(ctx, runningInstances, "cls_agent_installer.sh", params, model.CLSAgentInstalled, model.CLSAgentNotInstalled, "安装")
}

// runCLSAgentUninstall 对已安装 CLS Agent 的实例分批执行卸载。
// 每轮最多处理 batchLimit 条实例，剩余实例将在后续轮次中继续处理。
//
// 感知 CLS collectScope 采集范围：当 scope 非空时，额外将已安装但不在 scope 范围内的实例也纳入卸载。
func runCLSAgentUninstall(ctx context.Context) {
	// 先检查本地配置：如果 CLSEnabled == 1，说明可能是刚关闭，先更新配置
	config := model.GetSiteConfig(ctx)
	if config.CLSEnabled == 1 {
		// 远程接口已确认服务未开通，同步更新本地配置
		if err := model.UpdateSiteConfig(ctx, map[string]interface{}{"cls_enabled": 0, "cls_scope_mode": "all"}); err != nil {
			slog.Error("[CLS Agent] 更新本地 CLSEnabled 状态失败", "error", err)
			return
		}
		slog.Info("[CLS Agent] CLS 服务已关闭，已更新本地配置")

		// 同时清空采集范围
		if err := model.ClearCLSCollectScope(ctx); err != nil {
			slog.Error("[CLS Agent] 清空 CLS 采集范围失败", "error", err)
		}
	}

	batchLimit := getCLSBatchLimit()

	// 冷却截止时间：cls_agent_status_at 早于此时间的实例才会被捞取
	cooldownCutoff := time.Now().Add(-CLSAgentNotRunningCooldown)

	// 先查总待卸载数用于日志展示（排除冷却期内的实例）
	var totalPending int64
	if err := model.DB(ctx).Model(&model.Instance{}).
		Where("cls_agent_status = ? AND instance_id != '' AND is_doctor_node = ? AND source = ? AND (cls_agent_status_at IS NULL OR cls_agent_status_at < ?)",
			model.CLSAgentInstalled, false, model.InstanceSourceCVM, cooldownCutoff).
		Count(&totalPending).Error; err != nil {
		slog.Error("[CLS Agent] 查询待卸载实例总数失败", "error", err)
		return
	}

	if totalPending == 0 {
		slog.Info("[CLS Agent] 没有待卸载的实例，跳过本轮")
		return
	}

	var instances []model.Instance
	if err := model.DB(ctx).
		Where("cls_agent_status = ? AND instance_id != '' AND is_doctor_node = ? AND source = ? AND (cls_agent_status_at IS NULL OR cls_agent_status_at < ?)",
			model.CLSAgentInstalled, false, model.InstanceSourceCVM, cooldownCutoff).
		Limit(batchLimit).Find(&instances).Error; err != nil {
		slog.Error("[CLS Agent] 查询待卸载实例失败", "error", err)
		return
	}

	slog.Info("[CLS Agent] 本轮分批处理待卸载实例",
		"totalPending", totalPending,
		"batchLimit", batchLimit,
		"batchSize", len(instances),
		"remaining", totalPending-int64(len(instances)),
	)

	// 收集所有实例 ID，通过 CVM API 查询运行状态
	instanceIds := make([]string, 0, len(instances))
	for _, inst := range instances {
		instanceIds = append(instanceIds, inst.InstanceId)
	}

	_, runningIds, skipIds, pendingIds, err := describeInstancesWithoutAgentRole(ctx, instanceIds)
	if err != nil {
		slog.Error("[CLS Agent] 查询实例运行状态失败，跳过本轮", "error", err)
		return
	}

	// 将已释放/查不到的实例统一标记为"已跳过"，后续不再调度
	if len(skipIds) > 0 {
		if err := model.DB(ctx).Model(&model.Instance{}).
			Where("instance_id IN ?", skipIds).
			Updates(map[string]interface{}{
				"cls_agent_status":    model.CLSAgentSkipped,
				"cls_agent_status_at": time.Now(),
			}).Error; err != nil {
			slog.Error("[CLS Agent] 标记不可操作实例为跳过状态失败", "error", err)
		} else {
			slog.Info("[CLS Agent] 已标记不可操作实例为跳过状态（已释放/已删除），不再尝试卸载",
				"skipCount", len(skipIds),
				"skipIds", strings.Join(skipIds, ", "),
			)
		}
	}

	// 创建中（PENDING）的实例仅记录日志，不进入冷却队列也不标记跳过，下一轮自然重新捞取
	if len(pendingIds) > 0 {
		slog.Info("[CLS Agent] 发现创建中（PENDING）的实例，跳过本轮处理，下轮将重新检查",
			"pendingCount", len(pendingIds),
			"pendingIds", strings.Join(pendingIds, ", "),
		)
	}

	// 构建运行中实例 ID 集合、已跳过实例 ID 集合和创建中实例 ID 集合，用于过滤
	runningSet := make(map[string]struct{}, len(runningIds))
	for _, id := range runningIds {
		runningSet[id] = struct{}{}
	}
	skipSet := make(map[string]struct{}, len(skipIds))
	for _, id := range skipIds {
		skipSet[id] = struct{}{}
	}
	pendingSet := make(map[string]struct{}, len(pendingIds))
	for _, id := range pendingIds {
		pendingSet[id] = struct{}{}
	}

	// 只保留运行中的实例；已跳过的实例已标记 CLSAgentSkipped，不进入冷却；
	// 创建中（PENDING）的实例直接忽略，不进入冷却队列，下一轮自然重新捞取；
	// 其余非运行状态的实例仅更新检查时间戳进入冷却
	var runningInstances []model.Instance
	var notRunningIDs []string
	for _, inst := range instances {
		if _, ok := runningSet[inst.InstanceId]; ok {
			runningInstances = append(runningInstances, inst)
		} else if _, skipped := skipSet[inst.InstanceId]; skipped {
			// 已跳过的实例，已在上面标记 CLSAgentSkipped
			continue
		} else if _, pending := pendingSet[inst.InstanceId]; pending {
			// 创建中的实例，不做任何处理，下一轮自然重新捞取
			continue
		} else {
			// 其他非运行实例进入冷却
			notRunningIDs = append(notRunningIDs, inst.InstanceId)
		}
	}

	// 对非运行实例更新 cls_agent_status_at，使其在冷却期内不再被捞取
	if len(notRunningIDs) > 0 {
		if err := model.DB(ctx).Model(&model.Instance{}).
			Where("instance_id IN ?", notRunningIDs).
			Update("cls_agent_status_at", time.Now()).Error; err != nil {
			slog.Error("[CLS Agent] 更新非运行实例检查时间戳失败", "error", err)
		} else {
			slog.Info("[CLS Agent] 已标记非运行实例进入冷却期",
				"cooldown", CLSAgentNotRunningCooldown,
				"notRunningCount", len(notRunningIDs),
				"notRunningIds", strings.Join(notRunningIDs, ", "),
			)
		}
	}

	if len(runningInstances) == 0 {
		slog.Info("[CLS Agent] 本批无运行中的实例，已进入冷却或标记跳过或创建中，下轮将调度后续实例",
			"batchSize", len(instances),
			"skippedCount", len(skipIds),
			"pendingCount", len(pendingIds),
			"notRunningCount", len(notRunningIDs),
		)
		return
	}

	if len(runningInstances) < len(instances) {
		slog.Info("[CLS Agent] 部分实例不在运行中（已进入冷却或标记跳过或创建中），仅对运行中实例执行卸载",
			"batchSize", len(instances),
			"runningCount", len(runningInstances),
			"skippedCount", len(skipIds),
			"pendingCount", len(pendingIds),
			"notRunningCount", len(notRunningIDs),
		)
	}

	slog.Info("[CLS Agent] CLS 服务未开通，开始卸载 Agent", "count", len(runningInstances))

	// 批量将待卸载实例标记为"卸载中"，防止下一轮调度重复拉起卸载
	runningInstIDs := make([]string, 0, len(runningInstances))
	for _, inst := range runningInstances {
		runningInstIDs = append(runningInstIDs, inst.InstanceId)
	}
	// 注：批量标记"卸载中"失败不阻塞后续卸载流程，仅记录日志
	now := time.Now()
	if err := model.DB(ctx).Model(&model.Instance{}).
		Where("instance_id IN ?", runningInstIDs).
		Updates(map[string]interface{}{
			"cls_agent_status":    model.CLSAgentUninstalling,
			"cls_agent_status_at": now,
		}).Error; err != nil {
		slog.Error("[CLS Agent] 批量标记卸载中状态失败", "error", err)
	}

	batchRunScript(ctx, runningInstances, "cls_agent_uninstaller.sh", nil, model.CLSAgentNotInstalled, model.CLSAgentInstalled, "卸载")

	slog.Info("[CLS Agent] 本轮卸载任务完成")
}

// batchRunScript 批量对实例并发执行脚本，最大并发数 5。
// targetStatus 为脚本执行成功后 cls_agent_status 的目标值。
// failbackStatus 为脚本执行失败后 cls_agent_status 的回退值。
func batchRunScript(ctx context.Context, instances []model.Instance, script string, params map[string]string, targetStatus int, failbackStatus int, action string) {
	total := len(instances)
	var completed atomic.Int64 // 已完成计数（含成功和失败）
	var succeeded atomic.Int64 // 成功计数
	var failed atomic.Int64    // 失败计数

	slog.Info("[CLS Agent] 批量"+action+"任务开始",
		"total", total,
		"maxConcurrency", 5,
	)

	var wg sync.WaitGroup
	sem := make(chan struct{}, 5) // 最大并发数 5

	for i, inst := range instances {
		wg.Add(1)
		go func(idx int, inst model.Instance) {
			defer wg.Done()
			sem <- struct{}{}        // 获取信号量
			defer func() { <-sem }() // 释放信号量

			slog.Info("[CLS Agent] 开始"+action,
				"progress", fmt.Sprintf("[%d/%d]", idx+1, total),
				"instance_id", inst.InstanceId,
				"name", inst.Name,
			)

			_, err := controller.RunScript(ctx, inst.InstanceId, script, 300, inst.RuntimeUser, nil, params)
			done := completed.Add(1)
			if err != nil {
				failCount := failed.Add(1)
				slog.Error("[CLS Agent] "+action+"失败，回退状态",
					"progress", fmt.Sprintf("[%d/%d]", done, total),
					"instance_id", inst.InstanceId,
					"error", err,
					"stats", fmt.Sprintf("成功:%d 失败:%d 剩余:%d", succeeded.Load(), failCount, int64(total)-done),
				)

				// 安装/卸载失败回退 cls_agent_status_at 重置为 nil，下轮将重新检查，避免丢入冷却
				if dbErr := model.DB(ctx).Model(&inst).Updates(map[string]interface{}{
					"cls_agent_status":    failbackStatus,
					"cls_agent_status_at": nil,
				}).Error; dbErr != nil {
					slog.Error("[CLS Agent] 回退 cls_agent_status 失败", "instance_id", inst.InstanceId, "error", dbErr)
				}
				return
			}

			okCount := succeeded.Add(1)
			updates := map[string]interface{}{"cls_agent_status": targetStatus}
			// 新安装成功（targetStatus=CLSAgentInstalled）时，同步标记插件版本为 2.0
			if targetStatus == model.CLSAgentInstalled {
				updates["cls_plugin_version"] = controller.CLSPluginVersionV2
			}
			if dbErr := model.DB(ctx).Model(&inst).Updates(updates).Error; dbErr != nil {
				slog.Error("[CLS Agent] 更新 cls_agent_status 失败", "instance_id", inst.InstanceId, "error", dbErr)
			}
			slog.Info("[CLS Agent] "+action+"成功",
				"progress", fmt.Sprintf("[%d/%d]", done, total),
				"instance_id", inst.InstanceId,
				"stats", fmt.Sprintf("成功:%d 失败:%d 剩余:%d", okCount, failed.Load(), int64(total)-done),
			)
		}(i, inst)
	}
	wg.Wait()

	slog.Info("[CLS Agent] 批量"+action+"任务完成",
		"total", total,
		"succeeded", succeeded.Load(),
		"failed", failed.Load(),
	)
}

// isCLSUninstallEnabled 检查环境变量 ENABLE_CLS_UNINSTALL 是否开启。
// 默认关闭（false），设置为 "true" / "1" / "on" 可开启自动卸载。
func isCLSUninstallEnabled() bool {
	v := os.Getenv("ENABLE_CLS_UNINSTALL")
	if v == "" {
		return false
	}
	switch strings.ToLower(v) {
	case "true", "1", "on":
		return true
	default:
		return false
	}
}

// resetStaleCLSAgentStatus 将超时停留在"安装中/卸载中"中间状态的实例回退到初始状态。
// 场景：服务在批量标记后、脚本执行前重启，导致实例永久卡在中间状态。
//   - cls_agent_status=2 (安装中) 且超时 → 回退为 0 (未安装)
//   - cls_agent_status=3 (卸载中) 且超时 → 回退为 1 (已安装)
//   - cls_agent_status_at 为 NULL 的中间状态视为历史遗留，直接回退
//
// cutoff 取 min(processStartTime, now-CLSAgentStaleTimeout)：
//   - 进程刚启动（运行 < 10 分钟）：now-CLSAgentStaleTimeout 更早，取 now-CLSAgentStaleTimeout，
//     避免误回退未超时的中间状态实例；
//   - 进程运行 > 10 分钟：processStartTime 更早，取 processStartTime，
//     能立即感知上一进程残留的中间状态，无需等待 10 分钟。
//
// 注：非运行实例的冷却机制通过 SQL 查询条件（cls_agent_status_at）自然实现，
// 无需在此函数中额外处理。
func resetStaleCLSAgentStatus(ctx context.Context) {
	cutoff := time.Now().Add(-CLSAgentStaleTimeout)
	if processStartTime.Before(cutoff) {
		cutoff = processStartTime
	}

	slog.Info("[CLS Agent] 开始重置超时的中间状态实例",
		"timeout", CLSAgentStaleTimeout,
		"processStartTime", processStartTime.Format(time.RFC3339),
		"effectiveCutoff", cutoff.Format(time.RFC3339),
	)

	// 回退超时的"安装中" → "未安装"
	var staleInstalling []model.Instance
	model.DB(ctx).Where("cls_agent_status = ? AND instance_id != '' AND is_doctor_node = ? AND source = ? AND (cls_agent_status_at IS NULL OR cls_agent_status_at < ?)", model.CLSAgentInstalling, false, model.InstanceSourceCVM, cutoff).
		Find(&staleInstalling)

	if len(staleInstalling) > 0 {
		staleInstallingIDs := make([]string, 0, len(staleInstalling))
		for _, inst := range staleInstalling {
			staleInstallingIDs = append(staleInstallingIDs, inst.InstanceId)
		}

		result := model.DB(ctx).Model(&model.Instance{}).
			Where("instance_id IN ?", staleInstallingIDs).
			Updates(map[string]interface{}{
				"cls_agent_status":    model.CLSAgentNotInstalled,
				"cls_agent_status_at": nil,
			})
		if result.Error != nil {
			slog.Error("[CLS Agent] 重置超时安装中状态失败", "error", result.Error)
		} else {
			slog.Warn("[CLS Agent] 已重置超时的安装中实例",
				"count", result.RowsAffected,
				"timeout", CLSAgentStaleTimeout,
				"instances", strings.Join(staleInstallingIDs, ", "),
			)
		}
	}

	// 回退超时的"卸载中" → "已安装"
	var staleUninstalling []model.Instance
	model.DB(ctx).Where("cls_agent_status = ? AND instance_id != '' AND is_doctor_node = ? AND source = ? AND (cls_agent_status_at IS NULL OR cls_agent_status_at < ?)", model.CLSAgentUninstalling, false, model.InstanceSourceCVM, cutoff).
		Find(&staleUninstalling)

	if len(staleUninstalling) > 0 {
		staleUninstallingIDs := make([]string, 0, len(staleUninstalling))
		for _, inst := range staleUninstalling {
			staleUninstallingIDs = append(staleUninstallingIDs, inst.InstanceId)
		}

		result := model.DB(ctx).Model(&model.Instance{}).
			Where("instance_id IN ?", staleUninstallingIDs).
			Updates(map[string]interface{}{
				"cls_agent_status":    model.CLSAgentInstalled,
				"cls_agent_status_at": nil,
			})
		if result.Error != nil {
			slog.Error("[CLS Agent] 重置超时卸载中状态失败", "error", result.Error)
		} else {
			slog.Warn("[CLS Agent] 已重置超时的卸载中实例",
				"count", result.RowsAffected,
				"timeout", CLSAgentStaleTimeout,
				"instances", strings.Join(staleUninstallingIDs, ", "),
			)
		}
	}

	// 回退超时的 cls_plugin_version='updating' → '1.0'
	// 防止 update 接口执行中途服务重启导致实例永久卡在 updating 状态。
	var staleUpdating []model.Instance
	model.DB(ctx).Where("cls_plugin_version = ? AND instance_id != '' AND is_doctor_node = ? AND source = ? AND (cls_agent_status_at IS NULL OR cls_agent_status_at < ?)",
		controller.CLSPluginVersionUpdating, false, model.InstanceSourceCVM, cutoff).
		Find(&staleUpdating)

	if len(staleUpdating) > 0 {
		staleUpdatingIDs := make([]string, 0, len(staleUpdating))
		for _, inst := range staleUpdating {
			staleUpdatingIDs = append(staleUpdatingIDs, inst.InstanceId)
		}

		result := model.DB(ctx).Model(&model.Instance{}).
			Where("instance_id IN ?", staleUpdatingIDs).
			Updates(map[string]interface{}{
				"cls_plugin_version":  controller.CLSPluginVersionV1,
				"cls_agent_status_at": nil,
			})
		if result.Error != nil {
			slog.Error("[CLS Agent] 重置超时 updating 插件版本状态失败", "error", result.Error)
		} else {
			slog.Warn("[CLS Agent] 已重置超时的 updating 插件版本实例",
				"count", result.RowsAffected,
				"timeout", CLSAgentStaleTimeout,
				"instances", strings.Join(staleUpdatingIDs, ", "),
			)
		}
	}
}

// runCLSAgentScopeUninstall 当 CLS 开启且 collectScope 非空时，
// 将已安装 CLS Agent 但不在 scope 范围内的实例标记为待卸载（cls_agent_status=1 → 由 runCLSAgentUninstall 拾取）。
// scope 为空时不执行任何操作（全量模式）。
func runCLSAgentScopeUninstall(ctx context.Context) {
	// 根据 CLSScopeMode 判断是否为分组模式
	config := model.GetSiteConfig(ctx)
	isGroupMode := config.CLSScopeMode == "group"

	scopeInstanceIDs, scopeSet, err := model.GetCLSCollectScopeCVMInstanceIDs(ctx)
	if err != nil {
		slog.Error("[CLS Agent] 查询 CLS 采集范围实例失败", "error", err)
		return
	}

	// 分组模式下，即使 bindings 为空也视为 scopeSet=true
	if isGroupMode {
		scopeSet = true
	}

	// 非分组模式（全量模式），不需要卸载
	if !scopeSet {
		return
	}

	batchLimit := getCLSBatchLimit()

	// 冷却截止时间：cls_agent_status_at 早于此时间的实例才会被捞取
	cooldownCutoff := time.Now().Add(-CLSAgentNotRunningCooldown)

	// 查询 scope 外的已安装实例：
	// - scopeInstanceIDs 非空时，用 NOT IN 过滤
	// - scopeInstanceIDs 为空时（scope 已配置但分组下无实例），所有已安装实例都不在 scope 内
	var outOfScopeInstances []model.Instance
	query := model.DB(ctx).
		Where("cls_agent_status = ? AND instance_id != '' AND is_doctor_node = ? AND source = ? AND (cls_agent_status_at IS NULL OR cls_agent_status_at < ?)",
			model.CLSAgentInstalled, false, model.InstanceSourceCVM, cooldownCutoff)
	if len(scopeInstanceIDs) > 0 {
		query = query.Where("instance_id NOT IN ?", scopeInstanceIDs)
	}
	if err := query.
		Order("cls_agent_status_at ASC"). // 按状态更新时间升序，确保每轮都能处理最早的实例
		Limit(batchLimit).
		Find(&outOfScopeInstances).Error; err != nil {
		slog.Error("[CLS Agent] 查询 scope 外已安装实例失败", "error", err)
		return
	}

	if len(outOfScopeInstances) == 0 {
		return
	}

	slog.Info("[CLS Agent] 发现已安装但不在 CLS 采集范围内的实例，检查 CVM 运行状态", "count", len(outOfScopeInstances))

	// 收集实例 ID，先通过 CVM API 检查运行状态（与 runCLSAgentUninstall 对齐）
	outIDs := make([]string, 0, len(outOfScopeInstances))
	for _, inst := range outOfScopeInstances {
		outIDs = append(outIDs, inst.InstanceId)
	}

	_, runningIds, skipIds, pendingIds, err := describeInstancesWithoutAgentRole(ctx, outIDs)
	if err != nil {
		slog.Error("[CLS Agent] 查询 scope 外实例运行状态失败，跳过本轮", "error", err)
		return
	}

	// 将已释放/查不到的实例标记为"已跳过"
	if len(skipIds) > 0 {
		if err := model.DB(ctx).Model(&model.Instance{}).
			Where("instance_id IN ?", skipIds).
			Updates(map[string]interface{}{
				"cls_agent_status":    model.CLSAgentSkipped,
				"cls_agent_status_at": time.Now(),
			}).Error; err != nil {
			slog.Error("[CLS Agent] 标记 scope 外不可操作实例为跳过状态失败", "error", err)
		} else {
			slog.Info("[CLS Agent] 已标记 scope 外不可操作实例为跳过状态", "skipCount", len(skipIds), "skipIds", strings.Join(skipIds, ", "))
		}
	}

	if len(pendingIds) > 0 {
		slog.Info("[CLS Agent] scope 外发现创建中（PENDING）的实例，跳过处理", "pendingCount", len(pendingIds))
	}

	// 只对运行中的实例执行卸载
	runningSet := make(map[string]struct{}, len(runningIds))
	for _, id := range runningIds {
		runningSet[id] = struct{}{}
	}
	skipSetLocal := make(map[string]struct{}, len(skipIds))
	for _, id := range skipIds {
		skipSetLocal[id] = struct{}{}
	}
	pendingSetLocal := make(map[string]struct{}, len(pendingIds))
	for _, id := range pendingIds {
		pendingSetLocal[id] = struct{}{}
	}

	var runningOutOfScope []model.Instance
	var notRunningIDs []string
	for _, inst := range outOfScopeInstances {
		if _, ok := runningSet[inst.InstanceId]; ok {
			runningOutOfScope = append(runningOutOfScope, inst)
		} else if _, ok := skipSetLocal[inst.InstanceId]; ok {
			continue
		} else if _, ok := pendingSetLocal[inst.InstanceId]; ok {
			continue
		} else {
			notRunningIDs = append(notRunningIDs, inst.InstanceId)
		}
	}

	// 非运行实例进入冷却
	if len(notRunningIDs) > 0 {
		if err := model.DB(ctx).Model(&model.Instance{}).
			Where("instance_id IN ?", notRunningIDs).
			Update("cls_agent_status_at", time.Now()).Error; err != nil {
			slog.Error("[CLS Agent] 更新 scope 外非运行实例检查时间戳失败", "error", err)
		}
	}

	if len(runningOutOfScope) == 0 {
		slog.Info("[CLS Agent] scope 外无运行中的已安装实例需要卸载")
		return
	}

	// 标记为"卸载中"
	runningOutIDs := make([]string, 0, len(runningOutOfScope))
	for _, inst := range runningOutOfScope {
		runningOutIDs = append(runningOutIDs, inst.InstanceId)
	}
	now := time.Now()
	if err := model.DB(ctx).Model(&model.Instance{}).
		Where("instance_id IN ?", runningOutIDs).
		Updates(map[string]interface{}{
			"cls_agent_status":    model.CLSAgentUninstalling,
			"cls_agent_status_at": now,
		}).Error; err != nil {
		slog.Error("[CLS Agent] 标记 scope 外实例为卸载中失败", "error", err)
		return
	}

	slog.Info("[CLS Agent] 开始卸载 scope 外实例的 CLS Agent", "count", len(runningOutOfScope))
	batchRunScript(ctx, runningOutOfScope, "cls_agent_uninstaller.sh", nil, model.CLSAgentNotInstalled, model.CLSAgentInstalled, "scope外卸载")
}
