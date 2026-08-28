package task

import (
	"context"
	"log/slog"
	"time"

	hcommon "hatchery/common"
	"hatchery/controller"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	"gorm.io/gorm"
)

// DrainWorker：把绑在 FROZEN SG 上的实例搬到 ACTIVE 池。
//
// 触发源：
//   - SG RuleSet 初始化把老 base 标 FROZEN → DrainWorker 按云端真实绑定扫描 → 换绑到新 ACTIVE SG
//   - 未来运维工具手动 FROZEN 某个 SG 同样走此路径
//
// 核心循环（每 tick=5s）：
//  1. 查 FROZEN SG ID 列表（本期：本租户下 status=FROZEN 的全部）
//  2. 取分布式锁 sg-drain-<identifier>（跨 Pod 单写）
//  3. 云 API DescribeInstances(filter: security-group-id∈FROZEN) 扫一批
//  4. 逐实例换绑：ModifyInstancesAttribute(SecurityGroups=新 ACTIVE + 保留非 FROZEN 的其他 SG)
//  5. 成功 → 回填 instances.security_group_id + 原子 cvm_count++/--；失败 → sg_drain_state.fail_count++
//  6. fail_count ≥ 10 → 标 stuck_at，不再重试（运维介入）
//
// 关键设计：见 openspec/changes/sg-ruleset-projection/design.md D4。

const (
	drainTickInterval = 5 * time.Second
	drainBatchSize    = 100                       // 单 tick 最多换 100 实例（云 API 分页）
	drainLockTimeout  = 0                         // TryLock 不等待
	drainMaxFails     = model.DrainStuckThreshold // 10
)

// 函数指针：测试可以替换以下变量，将云 API 调用 / SG 选择 / 异步审计
// 替换为可控实现，避免真实云 API 依赖与 goroutine 生命周期风险。
var (
	// drainListFn：云 API 扫描指定 SG 下绑定的实例，测试可 stub。
	drainListFn = listInstancesBoundToSGs
	// drainModifyFn：云 API 换绑，测试可 stub。
	drainModifyFn = modifyInstanceSGs
	// drainGetDefaultRuleSetFn：缓存化的 GetDefaultRuleSet，测试可 stub。
	drainGetDefaultRuleSetFn = controller.GetDefaultRuleSet
	// drainSelectSGFn：池内选 SG，测试可 stub。
	drainSelectSGFn = controller.SelectSGForNewInstance
	// drainLogAuditFn：异步审计日志封装，测试用同步 no-op 避免 goroutine
	// 在 cleanup 后访问已销毁的 model.DB。
	// drainLogAuditFn：同步审计日志封装。
	// 不使用 go 异步，避免应用关闭时 goroutine 访问已关闭的 model.DB。
	drainLogAuditFn = func(ctx context.Context, startedAt time.Time, userID uint, username, action, resource, resourceID, status string) {
		model.LogAudit(ctx, startedAt, userID, username, action, resource, resourceID, status)
	}
)

func init() {
	RegisterTask(TaskDef{
		Name:         "drain-worker",
		Interval:     drainTickInterval,
		RunFunc:      drainTick,
		NeedDistLock: true,
		PerTenant:    true,
	})
}

// drainTick 单轮 tick。失败不向外扩散。
func drainTick(ctx context.Context) {
	// panic recovery 和分布式锁由 scheduler.executeTask 统一处理（NeedDistLock: true）

	frozenList, err := model.ListFrozenSGs(ctx)
	if err != nil {
		slog.Warn("[DrainWorker] list frozen failed", "err", err)
		return
	}
	if len(frozenList) == 0 {
		return
	}
	frozenIDs := make([]string, 0, len(frozenList))
	frozenSet := make(map[string]bool, len(frozenList))
	for _, sg := range frozenList {
		frozenIDs = append(frozenIDs, sg.SGID)
		frozenSet[sg.SGID] = true
	}

	// identifier 用于下游 drainOneInstance → drainSelectSGFn → AutoScaleSG 里
	// 构建云端 SG 名（clawpro-sg-{ident}-*）。不能用 ConfiguredIdentifier 空值，
	// 云端命名需要一个非空前缀，空时降级为 "default"。
	identifier := model.CurrentIdentifier(ctx)
	if identifier == "" {
		identifier = "default"
	}

	// 云 API 扫描 FROZEN 上绑着的实例
	instances, err := drainListFn(ctx, frozenIDs, drainBatchSize)
	if err != nil {
		slog.Warn("[DrainWorker] describe instances failed", "err", err)
		return
	}
	if len(instances) == 0 {
		return
	}

	slog.Info("[DrainWorker] found instances to drain",
		"frozen_sgs", len(frozenIDs), "instance_count", len(instances))

	for _, inst := range instances {
		if ctx.Err() != nil {
			return
		}
		drainOneInstance(ctx, identifier, inst, frozenSet)
	}
}

// drainOneInstance 单实例换绑。
func drainOneInstance(ctx context.Context, identifier string, inst instanceToDrain, frozenSet map[string]bool) {
	// 1. drain_stuck 跳过
	state, err := model.GetDrainState(ctx, inst.ID)
	if err == nil && state != nil && state.StuckAt != nil {
		slog.Debug("[DrainWorker] skip stuck instance", "instance", inst.ID)
		return
	}

	// 2. 选目标 ACTIVE SG（本期统一走 default RuleSet）
	rs, err := drainGetDefaultRuleSetFn(ctx)
	if err != nil {
		slog.Warn("[DrainWorker] get default rule set failed", "err", err)
		return
	}
	targetSG, _, err := drainSelectSGFn(ctx, identifier, rs.ID)
	if err != nil {
		slog.Warn("[DrainWorker] select target SG failed", "instance", inst.ID, "err", err)
		return
	}

	// 3. 构造新 SG 列表：FROZEN 替换为 targetSG，保留其他
	newSGs := []string{targetSG}
	sourceFrozenSGs := []string{}
	for _, sg := range inst.SecurityGroupIDs {
		if frozenSet[sg] {
			sourceFrozenSGs = append(sourceFrozenSGs, sg)
		} else if sg != targetSG {
			newSGs = append(newSGs, sg)
		}
	}
	newSGs = dedupStrings(newSGs)

	if len(sourceFrozenSGs) == 0 {
		// 云端扫描到了但实际已经不绑 FROZEN（缓存不一致或别的 Pod 刚换完）
		slog.Debug("[DrainWorker] instance already off FROZEN", "instance", inst.ID)
		return
	}

	// 4. 云 API 换绑
	if err := drainModifyFn(ctx, inst.ID, newSGs); err != nil {
		msg := err.Error()
		if len(msg) > 500 {
			msg = msg[:500]
		}
		failCount, incErr := model.IncrDrainFail(ctx, inst.ID, identifier, msg)
		if incErr != nil {
			slog.Warn("[DrainWorker] incr drain fail failed", "instance", inst.ID, "err", incErr)
		}
		slog.Warn("[DrainWorker] drain failed", "instance", inst.ID, "fail_count", failCount, "err", err)
		if failCount >= drainMaxFails {
			if mErr := model.MarkDrainStuck(ctx, inst.ID); mErr != nil {
				slog.Error("[DrainWorker] mark stuck failed", "instance", inst.ID, "err", mErr)
			}
			drainLogAuditFn(ctx, time.Now(), 0, "system", "drain_stuck", "instance", inst.ID, "stuck")
			slog.Error("[DrainWorker] instance stuck, needs manual intervention",
				"instance", inst.ID, "last_error", msg)
		}
		return
	}

	// 5. 成功：回填 + cvm_count 调整（事务保证一致性）
	if txErr := model.DB(ctx).Transaction(func(tx *gorm.DB) error {
		// 更新实例主安全组
		if err := tx.Model(&model.Instance{}).
			Where("instance_id = ?", inst.ID).
			UpdateColumn("security_group_id", targetSG).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgDrainUpdateInstanceSG)
		}
		// 目标 SG 的 cvm_count +1
		if err := tx.Model(&model.ManagedSGPool{}).
			Where("sg_id = ?", targetSG).
			UpdateColumn("cvm_count", gorm.Expr("cvm_count + 1")).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgDrainIncrementCVMCount)
		}
		// 来源 FROZEN SG 的 cvm_count -1
		for _, fz := range sourceFrozenSGs {
			if err := tx.Model(&model.ManagedSGPool{}).
				Where("sg_id = ? AND cvm_count > 0", fz).
				UpdateColumn("cvm_count", gorm.Expr("cvm_count - 1")).Error; err != nil {
				return hcommon.I18nRichError(err, i18n.MsgDrainDecrementCVMCount, fz)
			}
		}
		// 清理 drain state
		if err := tx.Where("instance_id = ?", inst.ID).Delete(&model.SGDrainState{}).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgDrainClearState)
		}
		return nil
	}); txErr != nil {
		slog.Error("[DrainWorker] post-drain transaction failed, attempting cloud rollback",
			"instance", inst.ID, "target_sg", targetSG, "err", txErr)

		// 云 API 已成功但 DB 事务失败 → 尝试回滚云端 SG 绑定（恢复为原始列表）
		rollbackErr := drainModifyFn(ctx, inst.ID, inst.SecurityGroupIDs)
		if rollbackErr != nil {
			// 回滚也失败 → 云端和 DB 都不一致，标记需要手动干预
			slog.Error("[DrainWorker] cloud rollback FAILED, marking instance for manual intervention",
				"instance", inst.ID, "target_sg", targetSG,
				"tx_err", txErr, "rollback_err", rollbackErr)
			if mErr := model.MarkDrainStuck(ctx, inst.ID); mErr != nil {
				slog.Error("[DrainWorker] mark stuck after rollback failure also failed",
					"instance", inst.ID, "err", mErr)
			}
			drainLogAuditFn(ctx, time.Now(), 0, "system", "drain_rollback_failed",
				"instance", inst.ID, "needs_manual_check")
		} else {
			// 回滚成功 → 云端恢复原状，DB 未变，下次 tick 可重试
			slog.Warn("[DrainWorker] cloud rollback succeeded, will retry next tick",
				"instance", inst.ID, "target_sg", targetSG)
			drainLogAuditFn(ctx, time.Now(), 0, "system", "drain_tx_failed_rollback_ok",
				"instance", inst.ID, "info")
		}
		return
	}
	drainLogAuditFn(ctx, time.Now(), 0, "system", "drain_instance", "instance", inst.ID, "success")
	slog.Info("[DrainWorker] drain success",
		"instance", inst.ID, "target_sg", targetSG, "replaced_frozen", sourceFrozenSGs)
}

// --- 云 API 辅助 ---

type instanceToDrain struct {
	ID               string
	SecurityGroupIDs []string
}

// listInstancesBoundToSGs 调 DescribeInstances 过滤 security-group-id 返回匹配实例。
// Tencent Cloud CVM 的 Filter 值支持数组形式，Name="security-group-id", Values=[frozen SG list]。
//
// 安全护栏：返回前按 hatchery instances 表过滤，只保留 hatchery 自己管理的 CVM。
// 云端可能扫到非 ClawPro 实例（用户把同一 SG 也用在别的机器上时），不能动它们。
func listInstancesBoundToSGs(ctx context.Context, sgIDs []string, limit int) ([]instanceToDrain, error) {
	if len(sgIDs) == 0 {
		return nil, nil
	}
	client, err := controller.NewCVMClient(ctx)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgCreateCVMClientFailed)
	}
	req := cvm.NewDescribeInstancesRequest()
	req.Filters = []*cvm.Filter{
		{
			Name:   common.StringPtr("security-group-id"),
			Values: common.StringPtrs(sgIDs),
		},
	}
	req.Limit = common.Int64Ptr(int64(limit))

	resp, apiErr := client.DescribeInstances(req)
	if apiErr != nil {
		return nil, hcommon.I18nRichError(apiErr, i18n.MsgOperationFailed)
	}
	if resp.Response == nil {
		return nil, nil
	}
	return filterManagedInstances(ctx, resp.Response.InstanceSet)
}

// filterManagedInstances 把云端 DescribeInstances 返回的实例列表过滤到
// 只剩 hatchery 自己管理的（在 instances 表里有记录），再转换成
// instanceToDrain 结构。
//
// 拆出来主要为了独立单测：覆盖 DB 过滤 + 转换逻辑，不需要 stub 云 API。
func filterManagedInstances(ctx context.Context, cloudInstances []*cvm.Instance) ([]instanceToDrain, error) {
	// 收集云端返回的 instance_id，一次查 DB 获取本租户下属于 hatchery 的 ID 集合。
	cloudIDs := make([]string, 0, len(cloudInstances))
	for _, inst := range cloudInstances {
		if inst != nil && inst.InstanceId != nil {
			cloudIDs = append(cloudIDs, *inst.InstanceId)
		}
	}
	managed := make(map[string]bool, len(cloudIDs))
	if len(cloudIDs) > 0 {
		var ids []string
		// identifier 过滤由 GORM Query callback 自动注入，多租户隔离。
		if err := model.DB(ctx).Model(&model.Instance{}).
			Where("instance_id IN ?", cloudIDs).
			Pluck("instance_id", &ids).Error; err != nil {
			return nil, hcommon.I18nRichError(err, i18n.MsgDrainFilterManagedInstances)
		}
		for _, id := range ids {
			managed[id] = true
		}
	}

	out := make([]instanceToDrain, 0, len(managed))
	for _, inst := range cloudInstances {
		if inst == nil || inst.InstanceId == nil || !managed[*inst.InstanceId] {
			continue
		}
		row := instanceToDrain{ID: *inst.InstanceId}
		for _, sg := range inst.SecurityGroupIds {
			if sg != nil {
				row.SecurityGroupIDs = append(row.SecurityGroupIDs, *sg)
			}
		}
		out = append(out, row)
	}
	return out, nil
}

// modifyInstanceSGs 调 ModifyInstancesAttribute 把实例的 SG 列表改为 newSGs。
func modifyInstanceSGs(ctx context.Context, instanceID string, newSGs []string) error {
	const maxRetries = 3
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// 指数退避：500ms, 1s
			time.Sleep(time.Duration(1<<(attempt-1)) * 500 * time.Millisecond)
		}
		client, err := controller.NewCVMClient(ctx)
		if err != nil {
			return err // 客户端创建失败不重试
		}
		req := cvm.NewModifyInstancesAttributeRequest()
		req.InstanceIds = common.StringPtrs([]string{instanceID})
		req.SecurityGroups = common.StringPtrs(newSGs)
		_, modErr := client.ModifyInstancesAttribute(req)
		if modErr == nil {
			return nil
		}
		lastErr = modErr
		slog.Debug("[DrainWorker] modifyInstanceSGs retrying",
			"instance", instanceID, "attempt", attempt+1, "err", modErr)
	}
	return lastErr
}

func dedupStrings(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" {
			continue
		}
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}
