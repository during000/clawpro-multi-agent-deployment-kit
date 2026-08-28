package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/controller"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tcerr "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
	"gorm.io/gorm"
)

// SG Guardian（sg-ruleset-projection 方案）
//
// 每 5 分钟运行一轮，单实例运行（分布式锁），启动时 0~90s 随机错峰避免多 Pod 扎堆。
// 职责（见 design.md D8）：
//  1. cvm_count 纠偏：COUNT(instances) + DescribeSecurityGroupAssociationStatistics，取较大值
//  2. 漂移自愈：ACTIVE AND (rule_version < RuleSet.Version OR drift_at IS NOT NULL) → 重新下发规则
//  3. 云端失踪检测：以 DB `managed_sg_pool` 为真相源，按 sg_id 精确 Describe 云端，
//     未返回的即失踪；ACTIVE 失踪 → 自愈重建，FROZEN/DRAINING 失踪 → 标记 RETIRED。
//     不再按名字前缀反向扫全账号——同账号多 hatchery 部署会 identifier 撞车互串。
//  5. drain_stuck 告警：sg_drain_state.stuck_at IS NOT NULL → 告警
//
// 本期所有"告警"通过 audit log + slog.Error 落地；未上通知/邮件渠道。

const (
	// guardianInterval 反向校正周期，对齐 PRD 4.5 的 5 分钟还原 SLA。
	// 单轮耗时和云 API 调用：
	//   - 本租户级：ListActive/ListFrozen + 每个 SG 一次 DescribeAssoc + 漂移修复 fan-out
	//   - 多 Pod 场景下由 TryLock("sg-guardian-<identifier>") 保证同一时刻只有一个 Pod 真正工作
	// 调高频带来的 API 压力主要来自健康 ACTIVE SG 的 DescribeAssoc 巡检；一个租户 20 个 SG 上限
	// × 每 5min 一次 × 24h = 5760 次/天，远低于单租户配额。
	guardianInterval = 5 * time.Minute
	// guardianOffset 启动后首次 tick 的随机延迟上限；避免多 Pod 同时冷启动扎堆打云 API。
	// 运行后按 ticker 自然错开，不再有额外抖动。
	guardianOffset  = 90 * time.Second // 运行后按 ticker 自然错开，不再有额外抖动。
	driftAlertAfter = 5 * time.Hour    // drift_at 距今 > 5h 触发告警
)

// 云 API 重试参数（可配置 / 测试可覆盖）。
var (
	cloudRetryMaxAttempts = 3                      // 最大尝试次数
	cloudRetryBaseBackoff = 200 * time.Millisecond // 首次退避基数，后续指数增长
)

// 函数指针：测试可以替换以下变量，将云 API 调用/异步审计等替换为可控实现。
// 生产运行时不要修改。
var (
	// describeAssocStatsFn：封装 describeAssocStats。
	describeAssocStatsFn = describeAssocStats
	// createCloudSGWithRetryFn：封装 createCloudSGWithRetry。
	createCloudSGWithRetryFn = createCloudSGWithRetry
	// applyRulesToCloudSGWithRetryFn：封装 applyRulesToCloudSGWithRetry。
	applyRulesToCloudSGWithRetryFn = applyRulesToCloudSGWithRetry
	// guardianApplyRulesFn / guardianTryDeleteFn / guardianGetDefaultRuleSetFn：
	// controller 包的直接云 API / 缓存代理，测试可 stub。
	guardianApplyRulesFn        = controller.ApplyRulesToCloudSG
	guardianTryDeleteFn         = controller.TryDeleteCloudSG
	guardianGetDefaultRuleSetFn = controller.GetDefaultRuleSet
	// guardianDescribeCloudRulesFn 拉云端某 SG 的规则并转成本地 Rule 列表，测试可 stub。
	// 出错时返回 (nil, err)；SG 在云端不存在统一返回 errSGGone（由调用方跳过处理）。
	guardianDescribeCloudRulesFn = describeCloudRulesForSG
	// sgGuardianLogAuditFn：异步审计日志封装；测试用同步 no-op 避免 goroutine
	// 在 cleanup 后访问已销毁的 model.DB。
	// 同步记录审计日志，避免应用关闭时异步 goroutine 访问已关闭的 DB。
	// Guardian 本身运行在后台 goroutine，同步写不会阻塞主流程。
	sgGuardianLogAuditFn = func(ctx context.Context, startedAt time.Time, userID uint, username, action, resource, resourceID, status string) {
		model.LogAudit(ctx, startedAt, userID, username, action, resource, resourceID, status)
	}
	// migrateInstanceSGsFn 封装 modifyInstanceSGs；guardianMigrateInstances /
	// guardianRescueEmptySGInstances 通过此变量调用，让测试可 stub 云 API。
	// 与 drain_worker.go 的 drainModifyFn 同源但独立（避免两个 worker 的测试互相
	// 覆盖 stub）。签名保持一致：func(instanceID string, newSGs []string) error。
	migrateInstanceSGsFn = modifyInstanceSGs
)

func init() {
	RegisterTask(TaskDef{
		Name:         "sg-guardian",
		Interval:     guardianInterval,
		RunFunc:      guardianTick,
		NeedDistLock: true,
		PerTenant:    true,
		InitialDelay: RandomDuration(guardianOffset),
	})
}

func guardianTick(ctx context.Context) {
	// panic recovery 和分布式锁由 scheduler.executeTask 统一处理（NeedDistLock: true）

	started := time.Now()
	slog.Info("[SGGuardian] tick start")

	guardianReconcileCVMCount(ctx)
	guardianSyncSGNames(ctx)
	guardianEnsureOfficeIngressRules(ctx)
	guardianDetectCloudRuleDrift(ctx)
	guardianHealDrift(ctx)
	guardianDetectOrphans(ctx)
	guardianDrainOrphanInstances(ctx)
	slog.Info("[SGGuardian] tick done", "elapsed_ms", time.Since(started).Milliseconds())
}

func guardianEnsureOfficeIngressRules(ctx context.Context) {
	if err := controller.RefreshOfficeIngressRulesForTenant(ctx); err != nil {
		slog.Warn("[SGGuardian] refresh office ingress rules failed", "err", err)
	}
}

// 1. cvm_count 纠偏
func guardianReconcileCVMCount(ctx context.Context) {
	var pool []model.ManagedSGPool
	if err := model.DB(ctx).Find(&pool).Error; err != nil {
		slog.Warn("[SGGuardian] list pool for cvm_count reconcile failed", "err", err)
		return
	}
	if len(pool) == 0 {
		return
	}

	// 云端真实绑定数
	cloudCounts, err := describeAssocStatsFn(ctx, poolSGIDs(pool))
	if err != nil {
		slog.Warn("[SGGuardian] describe association stats failed", "err", err)
		cloudCounts = map[string]int{}
	}

	for _, p := range pool {
		if ctx.Err() != nil {
			return
		}
		var dbCnt int64
		if err := model.DB(ctx).Model(&model.Instance{}).
			Where("security_group_id = ?", p.SGID).
			Count(&dbCnt).Error; err != nil {
			slog.Warn("[SGGuardian] count instances failed", "sg", p.SGID, "err", err)
			continue
		}
		cloudCnt := cloudCounts[p.SGID]
		want := int(dbCnt)
		if cloudCnt > want {
			want = cloudCnt
		}
		if want != p.CVMCount {
			if err := model.UpdateSGCVMCount(ctx, p.SGID, want); err != nil {
				slog.Warn("[SGGuardian] update cvm_count failed", "sg", p.SGID, "err", err)
				continue
			}
			slog.Info("[SGGuardian] cvm_count reconciled",
				"sg", p.SGID, "from", p.CVMCount, "db_cnt", dbCnt, "cloud_cnt", cloudCnt, "to", want)
		}
	}
}

// guardianSyncSGNames 每轮同步一次云端 SG 名称到 DB（ManagedSGPool.sg_name）。
//
// 为什么需要：
//   - SG 名字可能被管理员手动改（云控制台 / 其他自动化），ClawPro UI 应该展示云端当前实际名字
//   - SG RuleSet 初始化导入的 FROZEN 老 base 只有 sg_id 没有名字，这里补齐
//
// 实现：一次 DescribeSecurityGroupsFn 取本租户池内所有 sg_id 的名字；差异则 UPDATE。
// 云端缺的 sg_id（已被删除）不在此处理，留给 guardianDetectOrphans。
func guardianSyncSGNames(ctx context.Context) {
	var pool []model.ManagedSGPool
	if err := model.DB(ctx).
		Where("status IN ?", []string{model.SGStatusActive, model.SGStatusFrozen, model.SGStatusDraining}).
		Find(&pool).Error; err != nil {
		slog.Warn("[SGGuardian] list pool for sg_name sync failed", "err", err)
		return
	}
	if len(pool) == 0 {
		return
	}

	names, err := describeSGNamesFn(ctx, poolSGIDs(pool))
	if err != nil {
		slog.Warn("[SGGuardian] describe SG names failed", "err", err)
		return
	}
	for _, p := range pool {
		if ctx.Err() != nil {
			return
		}
		cloudName, ok := names[p.SGID]
		if !ok || cloudName == "" {
			continue
		}
		if cloudName == p.SGName {
			continue
		}
		if err := model.UpdateSGName(ctx, p.SGID, cloudName); err != nil {
			slog.Warn("[SGGuardian] update sg_name failed", "sg", p.SGID, "err", err)
			continue
		}
		slog.Info("[SGGuardian] sg_name synced",
			"sg", p.SGID, "from", p.SGName, "to", cloudName)
	}
}

// describeSGNamesFn 允许测试注入。生产使用 describeSGNames 默认实现。
var describeSGNamesFn = describeSGNames

// describeSGNames 查云端 SG 名称（按 sg_id 精确过滤；腾讯云单次 Limit=100）。
//
// 容错：腾讯云 DescribeSecurityGroups 对"批次内任一 sg_id 不存在"会整批返回
// ResourceNotFound 错误（例：`指定资源 ['sg-xxx'] 未找到`），导致整批查询失败。
// 这会让 guardianSyncSGNames / guardianDetectOrphans 在任一 SG 被云端删除后
// 整体短路，自愈永远跑不到。这里在批级 ResourceNotFound 时降级为逐个 Describe，
// 把仍存活的 sg_id 收回来；不存在的不会进 map，下游 detectOrphans 据此识别失踪
// 并触发 guardianHealMissingSG。
func describeSGNames(ctx context.Context, sgIDs []string) (map[string]string, error) {
	if len(sgIDs) == 0 {
		return nil, nil
	}
	client, err := guardianNewVpcClientFn(ctx)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgCreateVPCClientFailed)
	}
	out := make(map[string]string, len(sgIDs))
	const batchSize = 100
	for start := 0; start < len(sgIDs); start += batchSize {
		end := start + batchSize
		if end > len(sgIDs) {
			end = len(sgIDs)
		}
		batch := sgIDs[start:end]
		if err := describeSGNamesBatch(client, batch, out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// describeSGNamesBatch 对一批 sg_id 调云 API，结果写入 out。
// 整批 ResourceNotFound 时降级为逐个查询。
func describeSGNamesBatch(client controller.SGVpcClient, batch []string, out map[string]string) error {
	req := vpc.NewDescribeSecurityGroupsRequest()
	req.SecurityGroupIds = common.StringPtrs(batch)
	req.Limit = common.StringPtr("100")
	resp, rErr := client.DescribeSecurityGroups(req)
	if rErr != nil {
		if isSGNotFoundErr(rErr) {
			// 降级单查：批里有失踪 SG，不能让整批失败拖垮自愈
			slog.Warn("[SGGuardian] batch describe hit ResourceNotFound, falling back to per-id",
				"batch_size", len(batch), "err", rErr)
			describeSGNamesPerID(client, batch, out)
			return nil
		}
		return rErr
	}
	if resp.Response == nil {
		return nil
	}
	for _, sg := range resp.Response.SecurityGroupSet {
		if sg.SecurityGroupId == nil || sg.SecurityGroupName == nil {
			continue
		}
		out[*sg.SecurityGroupId] = *sg.SecurityGroupName
	}
	return nil
}

// describeSGNamesPerID 逐个 Describe，单个 ResourceNotFound 跳过（被识别为失踪），
// 其他错误只 warn 不中断（下一轮 guardian 会重试）。
func describeSGNamesPerID(client controller.SGVpcClient, sgIDs []string, out map[string]string) {
	for _, id := range sgIDs {
		req := vpc.NewDescribeSecurityGroupsRequest()
		req.SecurityGroupIds = common.StringPtrs([]string{id})
		req.Limit = common.StringPtr("1")
		resp, err := client.DescribeSecurityGroups(req)
		if err != nil {
			if isSGNotFoundErr(err) {
				// 不进 map = 上游 detectOrphans 视为失踪
				continue
			}
			slog.Warn("[SGGuardian] per-id describe failed, skip", "sg", id, "err", err)
			continue
		}
		if resp.Response == nil {
			continue
		}
		for _, sg := range resp.Response.SecurityGroupSet {
			if sg.SecurityGroupId == nil || sg.SecurityGroupName == nil {
				continue
			}
			out[*sg.SecurityGroupId] = *sg.SecurityGroupName
		}
	}
}

// isSGNotFoundErr 与 controller.isSGGoneError 同义，独立实现避免跨包耦合。
func isSGNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	var tce *tcerr.TencentCloudSDKError
	if errors.As(err, &tce) {
		code := tce.GetCode()
		if code == "ResourceNotFound" ||
			strings.HasPrefix(code, "ResourceNotFound.") ||
			code == "InvalidParameterValue" ||
			code == "InvalidSecurityGroupId.NotFound" {
			return true
		}
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "不存在") || strings.Contains(msg, "未找到")
}

// errSGGone 表示 SG 在云端已被删除（ResourceNotFound）。
// 由 guardianDescribeCloudRulesFn 内部 wrap 后返回，调用方据此跳过处理
// （由 guardianDetectOrphans 接管失踪场景）。
var errSGGone = hcommon.I18nError(i18n.MsgSGGoneInCloud)

// confirmSGMissingFn 是 confirmSGMissingInCloud 的可注入封装，便于单测替换。
var confirmSGMissingFn = confirmSGMissingInCloud

// confirmSGMissingInCloud 对单个 SG 调云 API 二次确认是否真的失踪。
//
// 用途：guardianDetectOrphans 在标 RETIRE 之前必须用本函数二次确认，
//
//	避免 describeSGNames 内部 transient 错误（超时/限流）让 SG"不在 cloudSet"
//	被误判为失踪。
//
// 返回值语义（严格区分三态，调用方据此决策）：
//   - (true, nil)   云端 100% 确认失踪（明确收到 ResourceNotFound）→ 可以 RETIRE
//   - (false, nil)  云端确认还在（Describe 成功且返回了这条 SG）→ 绝对不能 RETIRE
//   - (false, err)  transient/未知错误（超时/限流/网络/凭证）→ 绝对不能 RETIRE，
//     调用方应跳过本轮等下一轮重试
func confirmSGMissingInCloud(ctx context.Context, sgID string) (missing bool, err error) {
	client, cErr := guardianNewVpcClientFn(ctx)
	if cErr != nil {
		return false, hcommon.I18nRichError(cErr, i18n.MsgCreateVPCClientFailed)
	}
	req := vpc.NewDescribeSecurityGroupsRequest()
	req.SecurityGroupIds = common.StringPtrs([]string{sgID})
	req.Limit = common.StringPtr("1")
	resp, dErr := client.DescribeSecurityGroups(req)
	if dErr != nil {
		if isSGNotFoundErr(dErr) {
			return true, nil
		}
		return false, dErr
	}
	if resp.Response == nil {
		// 没报错但响应空 —— 视为 transient，让下一轮重试，绝不视为失踪
		return false, hcommon.I18nError(i18n.MsgSGDescribeNilResponse)
	}
	for _, sg := range resp.Response.SecurityGroupSet {
		if sg.SecurityGroupId != nil && *sg.SecurityGroupId == sgID {
			return false, nil
		}
	}
	// 接口成功返回但目标 sgID 不在结果集里 → 视为失踪
	// （腾讯云某些场景下传不存在的 ID 不报错而是返回空集）
	return true, nil
}

// 1.5 云端规则漂移检测（反向校验：云端规则 ≠ DB RuleSet → MarkSGDrift）
//
// 设计目标：用户在腾讯云控制台/API 直接修改 SG 规则后，本机制在下一轮 guardianTick
// 自动检测到差异，把 sg 标记为 drift，由紧随其后的 guardianHealDrift 把
// DB 里的 RuleSet.Rules 重新下发覆盖云端 —— 方向永远是 DB → 云端。
//
// 范围：所有 status='ACTIVE' 的 SG。不按 rule_version 过滤，因为：
//   - rule_version < RuleSet.Version 的场景 guardianHealDrift 本来就会处理，
//     这里再标 drift 是无害的（最终动作一样：DB 覆盖云端）；
//   - rule_version == RuleSet.Version 但云端被人改了 —— 正是本函数要解决的核心问题。
//
// 不查 drift_at IS NULL：drift_at 已经被标的本来就会被 healDrift 处理，再标一次也无害。
// 简单粗暴扫全部 ACTIVE，逻辑统一更好维护。
//
// 依赖：guardianDescribeCloudRulesFn（拉云端规则）+ controller.PolicySetToRules
// 已有的 Rule.Fingerprint（五元组规范化指纹），二者的 Rule 结构一致，差集可直接对比。
func guardianDetectCloudRuleDrift(ctx context.Context) {
	rs, err := guardianGetDefaultRuleSetFn(ctx)
	if err != nil {
		slog.Warn("[SGGuardian] detect cloud rule drift: get default rule set failed", "err", err)
		return
	}
	// 解析期望规则（DB 真相源）
	expected, err := parseRuleSetRules(rs.Rules)
	if err != nil {
		slog.Warn("[SGGuardian] detect cloud rule drift: parse rules failed", "err", err)
		return
	}
	expectedFps := fingerprintSet(expected)

	var pool []model.ManagedSGPool
	if err := model.DB(ctx).Where("status = ?", model.SGStatusActive).Find(&pool).Error; err != nil {
		slog.Warn("[SGGuardian] detect cloud rule drift: list active failed", "err", err)
		return
	}
	if len(pool) == 0 {
		return
	}

	for _, sg := range pool {
		if ctx.Err() != nil {
			return
		}
		// 已经标过 drift 的跳过 —— guardianHealDrift 马上就会处理它，
		// 没必要再调一次云 API。
		if sg.DriftAt != nil {
			continue
		}
		cloudRules, err := guardianDescribeCloudRulesFn(ctx, sg.SGID)
		if err != nil {
			if errors.Is(err, errSGGone) {
				// SG 已被删除：不在本函数范围内，由 guardianDetectOrphans 处理
				continue
			}
			slog.Warn("[SGGuardian] describe cloud rules failed", "sg", sg.SGID, "err", err)
			continue
		}
		cloudFps := fingerprintSet(cloudRules)
		if fingerprintSetsEqual(expectedFps, cloudFps) {
			continue
		}
		// 不一致 → 标 drift，让 guardianHealDrift 用 DB 规则强制覆盖云端
		slog.Warn("[SGGuardian] cloud rule drift detected; marking for heal",
			"sg", sg.SGID,
			"expected_count", len(expectedFps),
			"cloud_count", len(cloudFps),
			"missing_in_cloud", diffFps(expectedFps, cloudFps),
			"extra_in_cloud", diffFps(cloudFps, expectedFps),
		)
		if err := model.MarkSGDrift(ctx, sg.SGID); err != nil {
			slog.Warn("[SGGuardian] mark drift failed", "sg", sg.SGID, "err", err)
			continue
		}
		sgGuardianLogAuditFn(ctx, time.Now(), 0, "system", "detect_cloud_rule_drift", "sg", sg.SGID, "drift")
	}
}

// describeCloudRulesForSG 拉云端某 SG 的规则并归一化为 []controller.Rule。
// SG 在云端不存在时返回 errSGGone（包装后保留原因）。
func describeCloudRulesForSG(ctx context.Context, sgID string) ([]controller.Rule, error) {
	client, err := guardianNewVpcClientFn(ctx)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgCreateVPCClientFailed)
	}
	req := vpc.NewDescribeSecurityGroupPoliciesRequest()
	req.SecurityGroupId = common.StringPtr(sgID)
	resp, err := client.DescribeSecurityGroupPolicies(req)
	if err != nil {
		if isSGNotFoundErr(err) {
			return nil, hcommon.I18nRichError(errSGGone, i18n.MsgSGCloudRuleDescribeFailed)
		}
		return nil, err
	}
	if resp == nil || resp.Response == nil || resp.Response.SecurityGroupPolicySet == nil {
		// 空规则集合（合法）：返回空 slice 让上层做差集比较
		return []controller.Rule{}, nil
	}
	return controller.PolicySetToRules(resp.Response.SecurityGroupPolicySet), nil
}

// parseRuleSetRules 把 RuleSet.Rules（JSON 字符串）解析成 []controller.Rule。
// 空字符串 / "[]" / "null" 视为零规则。
func parseRuleSetRules(raw string) ([]controller.Rule, error) {
	s := strings.TrimSpace(raw)
	if s == "" || s == "null" {
		return nil, nil
	}
	var out []controller.Rule
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgSGPoolParseRulesJSONFailed)
	}
	return out, nil
}

// fingerprintSet 计算规则集合的指纹集合（用于差集比较）。
// 注意：腾讯云的 SG 规则按顺序生效，但本函数只关心"集合等价"，因为：
//   - guardianHealDrift 会用 ApplyRulesToCloudSG 全量覆盖（顺序由 DB 决定），
//     即使顺序不同也会被纠正；
//   - 仅做"是否有差异"的判定，避免把"DB 顺序 vs 云端腾讯云返回的顺序"
//     这种工程细节当成漂移误报。
func fingerprintSet(rules []controller.Rule) map[string]struct{} {
	out := make(map[string]struct{}, len(rules))
	for _, r := range rules {
		out[r.Fingerprint()] = struct{}{}
	}
	return out
}

func fingerprintSetsEqual(a, b map[string]struct{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			return false
		}
	}
	return true
}

// diffFps 返回 a 中不存在于 b 的指纹（最多 5 条用于日志，防爆量）。
func diffFps(a, b map[string]struct{}) []string {
	const maxLog = 5
	out := make([]string, 0, maxLog)
	for k := range a {
		if _, ok := b[k]; ok {
			continue
		}
		out = append(out, k)
		if len(out) >= maxLog {
			break
		}
	}
	return out
}

// 2. 漂移自愈
func guardianHealDrift(ctx context.Context) {
	rs, err := guardianGetDefaultRuleSetFn(ctx)
	if err != nil {
		slog.Warn("[SGGuardian] get default rule set failed", "err", err)
		return
	}

	var needs []model.ManagedSGPool
	if err := model.DB(ctx).Where(
		"status = ? AND (rule_version < ? OR drift_at IS NOT NULL)",
		model.SGStatusActive, rs.Version,
	).Find(&needs).Error; err != nil {
		slog.Warn("[SGGuardian] list drift candidates failed", "err", err)
		return
	}
	if len(needs) == 0 {
		return
	}

	slog.Info("[SGGuardian] healing drift", "count", len(needs), "rs_version", rs.Version)

	for _, sg := range needs {
		if ctx.Err() != nil {
			return
		}
		// drift 超过阈值告警
		if sg.DriftAt != nil && time.Since(*sg.DriftAt) > driftAlertAfter {
			slog.Error("[SGGuardian] drift persisted beyond threshold",
				"sg", sg.SGID, "drift_since", sg.DriftAt)
			sgGuardianLogAuditFn(ctx, time.Now(), 0, "system", "alert_drift_persist", "sg", sg.SGID, "alert")
		}

		if err := guardianApplyRulesFn(ctx, sg.SGID, rs.Rules); err != nil {
			slog.Warn("[SGGuardian] heal drift failed", "sg", sg.SGID, "err", err)
			if mErr := model.MarkSGDrift(ctx, sg.SGID); mErr != nil {
				slog.Warn("[SGGuardian] mark drift failed", "sg", sg.SGID, "err", mErr)
			}
			continue
		}
		if err := model.UpdateSGRuleVersion(ctx, sg.SGID, rs.Version); err != nil {
			slog.Warn("[SGGuardian] update rule_version failed", "sg", sg.SGID, "err", err)
			continue
		}
		sgGuardianLogAuditFn(ctx, time.Now(), 0, "system", "heal_drift_sg", "sg", sg.SGID, "success")
		slog.Info("[SGGuardian] drift healed", "sg", sg.SGID, "version", rs.Version)
	}
}

// guardianDetectOrphans 检测 DB pool 中登记的 SG 在云端是否失踪，必要时触发自愈。
//
// 真相源：DB `managed_sg_pool` 表（identifier 过滤由 GORM Query 回调自动注入）。
// 云端查询：按 sg_id 精确 Describe（分页，每批 100）。
//
// 为什么不再按名字前缀扫全账号？
//
//	同一腾讯云账号下可能部署多个 hatchery（各自一份 DB）且 identifier 可能相同，
//	名字前缀 `clawpro-sg-{ident}-*` 会与别家 hatchery 重叠。过去的"反向扫描 +
//	前缀过滤"会把别家 hatchery 建的 SG 误判为自家孤儿，从而触发错误的告警/处理。
//	现在完全以"我们自己写进 pool 的 sg_id"为准，云端命名仅作人肉运维辨识。
//
// 副作用：
//  1. "孤儿告警"（云端有 pool 没有）被移除：新模型下该概念不成立，因为我们不
//     再去"反向发现"别处建的 SG。DB 写一半挂掉导致的真孤儿会在下次分配时
//     通过名字冲突/配额异常暴露，走另外的告警通道。
//  2. "失踪自愈"（pool 有云端没有）保留：按 sg_id 精查结果为空即判定失踪。
func guardianDetectOrphans(ctx context.Context) {
	var pool []model.ManagedSGPool
	if err := model.DB(ctx).Find(&pool).Error; err != nil {
		slog.Warn("[SGGuardian] list pool for orphan detect failed", "err", err)
		return
	}
	if len(pool) == 0 {
		return
	}

	// 对非 RETIRED 的 pool 成员批量 describe，构建云端存活集合
	liveIDs := make([]string, 0, len(pool))
	for _, p := range pool {
		if p.Status == model.SGStatusRetired {
			continue
		}
		liveIDs = append(liveIDs, p.SGID)
	}
	cloudSet := make(map[string]bool, len(liveIDs))
	if len(liveIDs) > 0 {
		names, err := describeSGNamesFn(ctx, liveIDs)
		if err != nil {
			slog.Warn("[SGGuardian] describe pool SGs failed", "err", err)
			return
		}
		for sgID := range names {
			cloudSet[sgID] = true
		}
	}

	// 获取 RuleSet（自愈需要）
	rs, err := guardianGetDefaultRuleSetFn(ctx)
	if err != nil {
		slog.Warn("[SGGuardian] get rule set for self-healing failed", "err", err)
		// 无法自愈，但仍告警
		for _, p := range pool {
			if !cloudSet[p.SGID] && isManagedPoolSG(p) && p.Status != model.SGStatusRetired {
				slog.Error("[SGGuardian] DB SG missing in cloud (cannot heal: no rule_set)", "sg", p.SGID, "status", p.Status)
				sgGuardianLogAuditFn(ctx, time.Now(), 0, "system", "alert_missing_cloud_sg", "sg", p.SGID, "alert")
			}
		}
		return
	}

	// DB 有、云端没有 → 失踪 → 按 Status 分支处理
	//
	// ⚠️ 安全护栏：标 RETIRE 之前必须对单个 SG 做独立的二次 Describe，
	//    明确得到 ResourceNotFound 才视为"云端确认失踪"。
	//    避免 describeSGNames 内部 transient 错误（超时/限流）让它"不在 cloudSet"
	//    被误判为失踪 → 误标 RETIRED。
	//    transient 错误的 SG 跳过本轮，等下一轮 Guardian 重试。
	for _, p := range pool {
		if ctx.Err() != nil {
			return
		}
		if cloudSet[p.SGID] {
			continue
		}
		// 仅对 clawpro 托管的 SG 处理（有 RuleSetID 的记录）
		if !isManagedPoolSG(p) {
			continue
		}

		// RETIRED 已是终态，跳过（无需二次确认）
		if p.Status == model.SGStatusRetired {
			continue
		}

		// 二次确认：单 SG 独立 Describe，区分"真失踪"与"transient 错误"
		confirmed, confirmErr := confirmSGMissingFn(ctx, p.SGID)
		if confirmErr != nil {
			// transient 或未知错误 —— 本轮跳过，下一轮 Guardian 兜底
			slog.Warn("[SGGuardian] confirm SG missing failed, skip this round (will retry next tick)",
				"sg", p.SGID, "status", p.Status, "err", confirmErr)
			continue
		}
		if !confirmed {
			// 二次 Describe 反而成功了 —— cloudSet 那次是误判（transient/限流），云端还在
			slog.Info("[SGGuardian] SG actually exists in cloud (first describe was transient), skip RETIRE",
				"sg", p.SGID, "status", p.Status)
			continue
		}

		// 至此：云端 100% 确认失踪
		switch p.Status {
		case model.SGStatusActive:
			// ACTIVE SG 消失 → 必须自愈重建（承载新实例分配职责）
			slog.Error("[SGGuardian] ACTIVE SG missing in cloud, triggering self-healing",
				"sg", p.SGID)
			guardianHealMissingSG(ctx, p, rs)

		case model.SGStatusFrozen:
			// FROZEN SG 消失 → 无需重建（已停止接客，drain 无需继续）
			// 包括：初始化导入的用户老 base、曾 ACTIVE 后被冻结的 clawpro SG
			// 直接标记 RETIRED，残留实例下次分配时自然迁入 ACTIVE SG
			slog.Warn("[SGGuardian] FROZEN SG missing in cloud, marking RETIRED",
				"sg", p.SGID)
			if err := model.DB(ctx).Model(&model.ManagedSGPool{}).
				Where("sg_id = ?", p.SGID).
				Updates(map[string]interface{}{
					"status":    model.SGStatusRetired,
					"cvm_count": 0,
				}).Error; err != nil {
				slog.Warn("[SGGuardian] mark FROZEN as RETIRED failed", "sg", p.SGID, "err", err)
			}
			sgGuardianLogAuditFn(ctx, time.Now(), 0, "system", "auto_retire_frozen_sg", "sg", p.SGID, "info")

		case model.SGStatusDraining:
			// DRAINING 失踪属正常退役，标记 RETIRED 清理
			slog.Info("[SGGuardian] DRAINING SG missing in cloud, marking RETIRED", "sg", p.SGID)
			if err := model.DB(ctx).Model(&model.ManagedSGPool{}).
				Where("sg_id = ?", p.SGID).
				Updates(map[string]interface{}{
					"status":    model.SGStatusRetired,
					"cvm_count": 0,
				}).Error; err != nil {
				slog.Warn("[SGGuardian] mark DRAINING as RETIRED failed", "sg", p.SGID, "err", err)
			}

		default:
			slog.Error("[SGGuardian] DB SG missing in cloud (unknown status)", "sg", p.SGID, "status", p.Status)
			sgGuardianLogAuditFn(ctx, time.Now(), 0, "system", "alert_missing_cloud_sg", "sg", p.SGID, "alert")
		}
	}

	// 自愈后：检查是否有 RuleSet 下无 ACTIVE SG 的情况
	guardianCheckZeroActiveSG(ctx, rs.ID)
}

// guardianHealMissingSG 完整自愈：优先复用同 RuleSet 下未满的 ACTIVE SG；
// 全满时才重建 → 下发规则 → 迁移实例 → 退役旧记录。失败时降级为仅 RETIRED（止血）。
//
// 退役（步骤 5）改造：
//   - 必须等迁移真正把老 SG 上的实例清空才标 RETIRED；否则降级为 DRAINING，
//     把残留留给 DrainWorker / 下一轮 Guardian 兜底。这样不会再出现
//     "RETIRED + cvm_count=0 + drained_at 空 + 实例还挂着" 的假退役状态。
func guardianHealMissingSG(ctx context.Context, oldPool model.ManagedSGPool, rs *model.RuleSet) {
	started := time.Now()
	identifier := model.CurrentIdentifier(ctx)

	// 0. 优先复用：同 RuleSet 下还有 ACTIVE 且 cvm_count < 阈值 的 SG，直接挪过去，
	//    避免无脑扩容把池子撑满。撞 buffer 区也复用——总比建新 SG 健康（很多失踪场景
	//    本身就是云端配额波动 / 误删，建新 SG 反而更易失败）。
	var newSGID string
	var reused bool
	if reuseSG, qErr := guardianPickReusableSG(ctx, rs.ID, oldPool.SGID); qErr != nil {
		// DB 查询失败：不阻塞自愈（下面会降级到"建新 SG"路径），但必须打 warn
		// 让监控能捕捉到 —— 否则 DB 异常会被静默吞掉，表象是"复用通道总不命中"。
		slog.Warn("[SGGuardian] heal: pick reusable SG query failed; falling back to create new",
			"old_sg", oldPool.SGID, "rule_set_id", rs.ID, "err", qErr)
	} else if reuseSG != nil {
		newSGID = reuseSG.SGID
		reused = true
		slog.Info("[SGGuardian] heal: reusing existing ACTIVE SG instead of creating new one",
			"old_sg", oldPool.SGID, "reuse_sg", newSGID, "reuse_cvm_count", reuseSG.CVMCount)
	}

	if !reused {
		// 1. 创建新云端 SG（带重试）
		ordinal, err := model.NextSGOrdinalForRuleSet(ctx, rs.ID)
		if err != nil {
			slog.Error("[SGGuardian] heal: compute ordinal failed", "sg", oldPool.SGID, "err", err)
			guardianHealFallback(ctx, oldPool, "compute_ordinal_failed")
			return
		}
		newSGName := controller.BuildManagedSGName(identifier, rs.Name, ordinal)
		newSGDesc := controller.BuildManagedSGDescription(identifier, rs.Name, ordinal)

		createdSGID, err := createCloudSGWithRetryFn(ctx, newSGName, newSGDesc)
		if err != nil {
			slog.Error("[SGGuardian] heal: create cloud SG failed", "sg", oldPool.SGID, "err", err)
			guardianHealFallback(ctx, oldPool, err.Error())
			return
		}

		// 2. 下发规则（带重试）
		if err := applyRulesToCloudSGWithRetryFn(ctx, createdSGID, rs.Rules); err != nil {
			slog.Error("[SGGuardian] heal: apply rules failed", "new_sg", createdSGID, "err", err)
			guardianTryDeleteFn(ctx, createdSGID) // 回收
			guardianHealFallback(ctx, oldPool, err.Error())
			return
		}

		// 3. 新 SG 入池
		//    Identifier 由 GORM 全局 callback（set_identifier）在 Create 前自动填充，此处不显式赋值。
		newPoolRow := model.ManagedSGPool{
			SGID:        createdSGID,
			SGName:      newSGName, // 自愈重建的 SG 名与 createCloudSGWithRetryFn 使用的一致
			RuleSetID:   rs.ID,
			RuleVersion: rs.Version,
			Status:      model.SGStatusActive,
			CVMCount:    0,
		}
		if err := model.DB(ctx).Create(&newPoolRow).Error; err != nil {
			slog.Error("[SGGuardian] heal: insert new SG to pool failed", "new_sg", createdSGID, "err", err)
			guardianTryDeleteFn(ctx, createdSGID)
			guardianHealFallback(ctx, oldPool, err.Error())
			return
		}
		newSGID = createdSGID
	}

	// 4. 迁移实例（包含 sg=oldSGID 的常规孤儿，以及 sg='' 的早期 placeholder 残留）
	migratedCount, failedCount := guardianMigrateInstances(ctx, oldPool.SGID, newSGID)

	// 4.5 sg='' 孤儿救援：早期 placeholder 在 SG 还没建好时落库导致 security_group_id
	//     为空，正常 migrate / drain 都不会捞到它们（被各自的 != '' 过滤掉）。每次自愈
	//     顺手把它们挂到 newSGID 上，让后续 reconcileCVMCount 能把它们计入。
	emptyMigrated, emptyFailed := guardianRescueEmptySGInstances(ctx, newSGID)
	migratedCount += emptyMigrated
	failedCount += emptyFailed

	// 5. 退役旧记录：必须确认实例真正清空。残留场景（云 API 还没对账成功 / DB 只更新部分）
	//    → 标 DRAINING 让 DrainWorker 接管，下一轮 Guardian 再判断是否能 RETIRE。
	//    这样不会再出现"老 SG 状态 RETIRED + drained_at 空 + 还有实例挂着"的假退役。
	remaining, cntErr := guardianCountInstancesBySG(ctx, oldPool.SGID)
	if cntErr != nil {
		slog.Warn("[SGGuardian] heal: count remaining instances failed, fall back to DRAINING",
			"old_sg", oldPool.SGID, "err", cntErr)
		remaining = -1 // 让下面的判断走 DRAINING 分支
	}

	switch {
	case failedCount == 0 && remaining == 0:
		// 全部迁完，正常退役
		now := time.Now()
		if err := model.DB(ctx).Model(&model.ManagedSGPool{}).
			Where("sg_id = ?", oldPool.SGID).
			Updates(map[string]interface{}{
				"status":     model.SGStatusRetired,
				"cvm_count":  0,
				"drained_at": &now,
			}).Error; err != nil {
			slog.Error("[SGGuardian] heal: mark old SG RETIRED failed", "sg", oldPool.SGID, "err", err)
		}
	default:
		// 还有残留，转 DRAINING + 把 cvm_count 同步成实际剩余数（remaining<0 时保持原值）
		updates := map[string]interface{}{
			"status": model.SGStatusDraining,
		}
		if remaining >= 0 {
			updates["cvm_count"] = remaining
		}
		if err := model.DB(ctx).Model(&model.ManagedSGPool{}).
			Where("sg_id = ?", oldPool.SGID).
			Updates(updates).Error; err != nil {
			slog.Error("[SGGuardian] heal: mark old SG DRAINING failed", "sg", oldPool.SGID, "err", err)
		}
		slog.Warn("[SGGuardian] heal: old SG has residual instances, kept DRAINING for next round",
			"old_sg", oldPool.SGID, "remaining", remaining, "failed", failedCount)
	}

	// 6. 审计日志
	auditStatus := "success"
	if failedCount > 0 || remaining > 0 {
		auditStatus = "partial"
	}
	sgGuardianLogAuditFn(ctx, started, 0, "system", "auto_heal_missing_sg", "sg", oldPool.SGID, auditStatus)
	slog.Info("[SGGuardian] self-healing complete",
		"old_sg", oldPool.SGID, "new_sg", newSGID, "reused", reused,
		"migrated", migratedCount, "failed", failedCount, "remaining", remaining,
		"elapsed_ms", time.Since(started).Milliseconds())
}

// guardianPickReusableSG 在同 RuleSet 内挑一个可复用的 ACTIVE SG，排除 oldPool 自己
// 和 drift 的成员。优先 cvm_count 最小者；硬上限内都允许（不在乎是否在 buffer 区——
// 自愈语境下"有地方放"比"严格遵守阈值"更重要）。查不到返回 (nil, nil)。
func guardianPickReusableSG(ctx context.Context, ruleSetID uint, excludeSGID string) (*model.ManagedSGPool, error) {
	var sg model.ManagedSGPool
	err := model.DB(ctx).Where(
		"rule_set_id = ? AND status = ? AND drift_at IS NULL AND sg_id != ? AND cvm_count < ?",
		ruleSetID, model.SGStatusActive, excludeSGID, model.SGPoolHardLimit).
		Order("cvm_count ASC, created_at ASC").
		Limit(1).
		First(&sg).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &sg, nil
}

// guardianCountInstancesBySG 统计某 SG 上仍挂着的未删除实例数（identifier 由 GORM
// callback 自动注入）。用于退役前残留判断，不受 instance_id 是否为空影响。
func guardianCountInstancesBySG(ctx context.Context, sgID string) (int, error) {
	var n int64
	if err := model.DB(ctx).Model(&model.Instance{}).
		Where("security_group_id = ? AND deleted_at IS NULL", sgID).
		Count(&n).Error; err != nil {
		return 0, err
	}
	return int(n), nil
}

// guardianHealFallback 降级路径：仅标记 RETIRED + 告警 + 审计。
func guardianHealFallback(ctx context.Context, pool model.ManagedSGPool, reason string) {
	if err := model.DB(ctx).Model(&model.ManagedSGPool{}).
		Where("sg_id = ?", pool.SGID).
		Updates(map[string]interface{}{
			"status":    model.SGStatusRetired,
			"cvm_count": 0,
		}).Error; err != nil {
		slog.Error("[SGGuardian] fallback: mark RETIRED failed", "sg", pool.SGID, "err", err)
	}
	slog.Error("[SGGuardian] CRITICAL: self-healing failed, degraded to RETIRED-only",
		"sg", pool.SGID, "reason", reason)
	sgGuardianLogAuditFn(ctx, time.Now(), 0, "system", "auto_retire_missing_sg", "sg", pool.SGID, "degraded")
}

// guardianCheckZeroActiveSG 自愈后检查 RuleSet 下是否还有 ACTIVE SG。
func guardianCheckZeroActiveSG(ctx context.Context, ruleSetID uint) {
	count, err := model.CountActiveSGsInRuleSet(ctx, ruleSetID)
	if err != nil {
		slog.Warn("[SGGuardian] count active SGs failed", "rule_set_id", ruleSetID, "err", err)
		return
	}
	if count == 0 {
		slog.Error("[SGGuardian] CRITICAL: no ACTIVE SG left for rule_set", "rule_set_id", ruleSetID)
		sgGuardianLogAuditFn(ctx, time.Now(), 0, "system", "alert_no_active_sg",
			"rule_set", fmt.Sprintf("%d", ruleSetID), "critical")
	}
}

// 5. 孤儿实例迁移：DB 中 security_group_id 不属于任何 ACTIVE pool 的未删除实例，
// 如果默认 RuleSet 有可用的 ACTIVE SG，主动迁移到 ACTIVE pool。
// 场景：老 base SG 被误标 RETIRED 后 DrainWorker 不处理，或实例关联的 SG
// 从未进入 FROZEN 状态（如首次初始化时配额不足没创建新 SG，后续补配额后才有 ACTIVE pool）。
func guardianDrainOrphanInstances(ctx context.Context) {
	// 1. 获取默认 RuleSet，确认有可用 ACTIVE pool
	rs, err := guardianGetDefaultRuleSetFn(ctx)
	if err != nil {
		return // 无默认 RuleSet，跳过
	}
	activeSGs, err := model.ListActiveSGsByRuleSet(ctx, rs.ID)
	if err != nil || len(activeSGs) == 0 {
		return // 无可用 ACTIVE SG
	}

	// 2. 查 DB 中 security_group_id 非空、不属于 ACTIVE pool、未软删除的实例
	//    identifier 过滤由 model.DB 的 GORM Query 回调自动注入（Instance 有 Identifier 字段）。
	//    identifier 变量本身仍需保留：下游 drainSelectSGFn → AutoScaleSG 需要它拼云端 SG 名。
	identifier := model.CurrentIdentifier(ctx)
	var orphans []model.Instance
	activeSGIDs := make([]string, 0, len(activeSGs))
	for _, sg := range activeSGs {
		activeSGIDs = append(activeSGIDs, sg.SGID)
	}

	// security_group_id NOT IN ? 在 MySQL/SQLite 里对 NULL/'' 的语义不一致，且 != ''
	// 会过滤掉早期 placeholder 时序竞争留下的 sg='' 孤儿。这里显式带上 OR sg='' 把
	// 它们也捞进来，统一交给下面的迁移分支处理（无 instance_id 直接改 DB；有
	// instance_id 走 modify SG）。
	query := model.DB(ctx).Where(
		"deleted_at IS NULL AND (security_group_id = '' OR security_group_id NOT IN ?)",
		activeSGIDs)
	if err := query.Limit(100).Find(&orphans).Error; err != nil {
		slog.Warn("[SGGuardian] query orphan instances failed", "err", err)
		return
	}
	if len(orphans) == 0 {
		return
	}

	slog.Info("[SGGuardian] found orphan instances to migrate", "count", len(orphans))

	// 3. 逐个迁移
	for _, inst := range orphans {
		if ctx.Err() != nil {
			return
		}
		if inst.InstanceId == "" {
			// 无实例 ID 的记录，直接更新 DB 指向第一个 ACTIVE SG
			if len(activeSGIDs) == 0 {
				continue
			}
			targetSG := activeSGIDs[0]
			if err := model.DB(ctx).Model(&model.Instance{}).
				Where("id = ?", inst.ID).
				UpdateColumn("security_group_id", targetSG).Error; err != nil {
				slog.Warn("[SGGuardian] orphan drain: update no-instance-id record failed",
					"id", inst.ID, "target_sg", targetSG, "err", err)
				continue
			}
			if err := model.IncrementSGCVMCount(ctx, targetSG); err != nil {
				slog.Warn("[SGGuardian] orphan no-instance-id: incr cvm_count failed", "sg", targetSG, "err", err)
			}
			slog.Info("[SGGuardian] orphan no-instance-id record migrated",
				"id", inst.ID, "old_sg", inst.SecurityGroupId, "target_sg", targetSG)
			continue
		}

		// 选目标 ACTIVE SG
		targetSG, _, selErr := drainSelectSGFn(ctx, identifier, rs.ID)
		if selErr != nil {
			slog.Warn("[SGGuardian] orphan drain: select target SG failed", "instance", inst.InstanceId, "err", selErr)
			continue
		}

		// 云 API 换绑：设为 [targetSG]（去掉老的非 ACTIVE SG）
		if err := drainModifyFn(ctx, inst.InstanceId, []string{targetSG}); err != nil {
			slog.Warn("[SGGuardian] orphan drain: modify instance SGs failed",
				"instance", inst.InstanceId, "old_sg", inst.SecurityGroupId, "target_sg", targetSG, "err", err)
			continue // 失败跳过，下一轮 Guardian 重试
		}

		// 云 API 成功 → 更新 DB（带重试，避免云端已换绑但 DB 不一致）
		var dbErr error
		for retry := 0; retry < cloudRetryMaxAttempts; retry++ {
			dbErr = model.DB(ctx).Model(&model.Instance{}).
				Where("instance_id = ?", inst.InstanceId).
				UpdateColumn("security_group_id", targetSG).Error
			if dbErr == nil {
				break
			}
			time.Sleep(cloudRetryBaseBackoff << retry)
		}
		if dbErr != nil {
			// 云 API 已成功但 DB 持久化失败 → ERROR 级别，需人工关注
			// 下一轮 Guardian 会再次尝试（云 API 换绑是幂等的）
			slog.Error("[SGGuardian] orphan drain: update instance sg in DB failed after retries",
				"instance", inst.InstanceId, "target_sg", targetSG, "err", dbErr)
			continue
		}
		if err := model.IncrementSGCVMCount(ctx, targetSG); err != nil {
			slog.Warn("[SGGuardian] orphan: incr cvm_count failed", "sg", targetSG, "err", err)
		}
		slog.Info("[SGGuardian] orphan instance migrated",
			"instance", inst.InstanceId, "old_sg", inst.SecurityGroupId, "target_sg", targetSG)
	}
}

// 6. drain_stuck 告警

// isInstanceNotFoundErr 判断是否为腾讯云"实例不存在"错误。
// 触发场景：实例已被释放/销毁，但 hatchery DB 中的 instances 行未清理。
// 修复策略：把这条 instance 的 security_group_id 从死 SG 摘掉，避免迁移循环卡死。
func isInstanceNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	var tce *tcerr.TencentCloudSDKError
	if errors.As(err, &tce) {
		code := tce.GetCode()
		if code == "InvalidInstanceId.NotFound" ||
			code == "InvalidInstance.NotFound" ||
			strings.HasPrefix(code, "InvalidInstanceId.") {
			return true
		}
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "invalidinstanceid")
}

// guardianMigrateInstances 分批迁移老 SG 上的实例到新 SG。
// 返回 (成功数, 失败数)。
//
// 死循环保护：
//  1. 单批"零进度"（DB 行无任何变化）→ 退出，避免无限循环
//  2. 实例云端不存在（InvalidInstanceId.NotFound）→ 把 instance.security_group_id
//     置为 newSGID（视同迁移完成），让查询下一轮跳过它
//  3. 总失败上限 guardianMigrateMaxFailures → 直接返回，由上层 fallback
const guardianMigrateMaxFailures = 500

func guardianMigrateInstances(ctx context.Context, oldSGID, newSGID string) (migrated, failed int) {
	const batchSize = 100

	for {
		if ctx.Err() != nil {
			return
		}

		// 查 DB 中绑在老 SG 上的实例（分批）
		// 范围对齐 controller.listInstanceIds：未软删（gorm 自动）+ instance_id 非空。
		// 不过滤 last_cvm_state：STOPPED 实例的 SG 绑定关系仍然有效，必须迁；
		// "实例不存在"由下方云 API 错误码 InvalidInstanceId.NotFound 兜底，比 DB 字段更可靠。
		var instances []model.Instance
		if err := model.DB(ctx).Where("security_group_id = ? AND instance_id != ''", oldSGID).
			Limit(batchSize).
			Find(&instances).Error; err != nil {
			slog.Warn("[SGGuardian] heal: query instances failed", "old_sg", oldSGID, "err", err)
			return
		}
		if len(instances) == 0 {
			return // 全部迁移完毕
		}

		// 本批进度计数器：用于检测死循环（本批 0 行被改动 → 下轮查同一批）
		batchProgress := 0

		for _, inst := range instances {
			if ctx.Err() != nil {
				return
			}

			// 云 API 换绑：把实例的安全组从 old 换到 new
			err := migrateInstanceSGsFn(ctx, inst.InstanceId, []string{newSGID})

			// 实例已不存在（云端被释放，DB 未清理）→ 视为已迁移，把 sg_id 摘到 newSGID
			// 让 WHERE security_group_id=oldSGID 查询下一轮跳过它，避免死循环
			if err != nil && isInstanceNotFoundErr(err) {
				slog.Warn("[SGGuardian] heal: instance gone in cloud, detaching from old SG",
					"instance", inst.InstanceId, "old_sg", oldSGID, "err", err)
				if uErr := model.DB(ctx).Model(&model.Instance{}).
					Where("instance_id = ?", inst.InstanceId).
					UpdateColumn("security_group_id", newSGID).Error; uErr != nil {
					slog.Warn("[SGGuardian] heal: detach gone instance from old SG failed",
						"instance", inst.InstanceId, "err", uErr)
					failed++
					if failed >= guardianMigrateMaxFailures {
						slog.Error("[SGGuardian] heal: migrate failures exceeded limit, abort",
							"old_sg", oldSGID, "failed", failed)
						return
					}
					continue
				}
				batchProgress++
				migrated++
				continue
			}

			if err != nil {
				slog.Warn("[SGGuardian] heal: migrate instance failed",
					"instance", inst.InstanceId, "old_sg", oldSGID, "new_sg", newSGID, "err", err)
				failed++
				if failed >= guardianMigrateMaxFailures {
					slog.Error("[SGGuardian] heal: migrate failures exceeded limit, abort",
						"old_sg", oldSGID, "failed", failed)
					return
				}
				continue // 失败的跳过，下一轮 Guardian 会重试
			}

			// 云 API 成功 → 更新 DB
			if err := model.DB(ctx).Model(&model.Instance{}).
				Where("instance_id = ?", inst.InstanceId).
				UpdateColumn("security_group_id", newSGID).Error; err != nil {
				slog.Warn("[SGGuardian] heal: update instance sg_id in DB failed",
					"instance", inst.InstanceId, "err", err)
				// 云端已换绑成功但 DB 更新失败 → 下轮 Guardian cvm_count 纠偏会修正
				failed++
				if failed >= guardianMigrateMaxFailures {
					slog.Error("[SGGuardian] heal: migrate failures exceeded limit, abort",
						"old_sg", oldSGID, "failed", failed)
					return
				}
				continue
			}
			batchProgress++
			migrated++
		}

		// 死循环兜底：本批一行 DB 都没改动 → 下一轮查询会拿到完全相同的数据，必然死循环
		if batchProgress == 0 {
			slog.Error("[SGGuardian] heal: migrate batch made zero progress, abort to avoid deadloop",
				"old_sg", oldSGID, "batch_size", len(instances), "failed_total", failed)
			return
		}
	}
}

// guardianRescueEmptySGInstances 救援 security_group_id=” 的早期 placeholder 实例：
// 把它们换绑到 newSGID。这类实例是早期初始化时序竞争的产物（实例创建早于第一个
// ACTIVE SG 入池，selectedSG 为空字符串落库），常规 migrate（按 oldSGID 过滤）和
// drainOrphan（要求 sg != ”）都漏掉了。
//
// 行为：
//   - instance_id 为空 → 仅更新 DB（无云端可换绑）
//   - instance_id 非空 → 调 modifyInstanceSGs 改云端绑定，成功后更新 DB
//   - 云端 InvalidInstanceId.NotFound → 视为已迁移（云端已不存在），仅清理 DB
//
// 共用 guardianMigrate 的失败上限保护，避免大量 NotFound 类错误把整轮自愈拖死。
func guardianRescueEmptySGInstances(ctx context.Context, newSGID string) (migrated, failed int) {
	const batchSize = 100

	for {
		if ctx.Err() != nil {
			return
		}

		var instances []model.Instance
		if err := model.DB(ctx).Where("security_group_id = ? AND deleted_at IS NULL", "").
			Limit(batchSize).
			Find(&instances).Error; err != nil {
			slog.Warn("[SGGuardian] rescue empty-sg: query failed", "err", err)
			return
		}
		if len(instances) == 0 {
			return
		}

		batchProgress := 0
		for _, inst := range instances {
			if ctx.Err() != nil {
				return
			}

			// 无 instance_id：直接补 DB 字段
			if inst.InstanceId == "" {
				if err := model.DB(ctx).Model(&model.Instance{}).
					Where("id = ?", inst.ID).
					UpdateColumn("security_group_id", newSGID).Error; err != nil {
					slog.Warn("[SGGuardian] rescue empty-sg: update no-instance-id row failed",
						"id", inst.ID, "err", err)
					failed++
					if failed >= guardianMigrateMaxFailures {
						return
					}
					continue
				}
				if err := model.IncrementSGCVMCount(ctx, newSGID); err != nil {
					slog.Warn("[SGGuardian] rescue: incr cvm_count failed", "sg", newSGID, "err", err)
				}
				batchProgress++
				migrated++
				slog.Info("[SGGuardian] rescue empty-sg: no-instance-id row attached",
					"id", inst.ID, "new_sg", newSGID)
				continue
			}

			// 有 instance_id：先动云端再写 DB
			err := migrateInstanceSGsFn(ctx, inst.InstanceId, []string{newSGID})
			if err != nil && isInstanceNotFoundErr(err) {
				slog.Warn("[SGGuardian] rescue empty-sg: instance gone in cloud, just attach in DB",
					"instance", inst.InstanceId, "err", err)
				if uErr := model.DB(ctx).Model(&model.Instance{}).
					Where("instance_id = ?", inst.InstanceId).
					UpdateColumn("security_group_id", newSGID).Error; uErr != nil {
					slog.Warn("[SGGuardian] rescue empty-sg: update gone instance failed",
						"instance", inst.InstanceId, "err", uErr)
					failed++
					if failed >= guardianMigrateMaxFailures {
						return
					}
					continue
				}
				batchProgress++
				migrated++
				continue
			}
			if err != nil {
				slog.Warn("[SGGuardian] rescue empty-sg: modify instance SGs failed",
					"instance", inst.InstanceId, "new_sg", newSGID, "err", err)
				failed++
				if failed >= guardianMigrateMaxFailures {
					return
				}
				continue
			}

			if err := model.DB(ctx).Model(&model.Instance{}).
				Where("instance_id = ?", inst.InstanceId).
				UpdateColumn("security_group_id", newSGID).Error; err != nil {
				slog.Warn("[SGGuardian] rescue empty-sg: update DB failed",
					"instance", inst.InstanceId, "err", err)
				failed++
				if failed >= guardianMigrateMaxFailures {
					return
				}
				continue
			}
			if err := model.IncrementSGCVMCount(ctx, newSGID); err != nil {
				slog.Warn("[SGGuardian] rescue: incr cvm_count failed", "sg", newSGID, "err", err)
			}
			batchProgress++
			migrated++
			slog.Info("[SGGuardian] rescue empty-sg: instance migrated",
				"instance", inst.InstanceId, "new_sg", newSGID)
		}

		if batchProgress == 0 {
			slog.Error("[SGGuardian] rescue empty-sg: batch made zero progress, abort to avoid deadloop",
				"batch_size", len(instances), "failed_total", failed)
			return
		}
	}
}

// createCloudSGWithRetry 创建云 SG 带重试。
func createCloudSGWithRetry(ctx context.Context, name, desc string) (string, error) {
	for attempt := 0; attempt < cloudRetryMaxAttempts; attempt++ {
		sgID, err := controller.CreateCloudSG(ctx, name, desc)
		if err == nil {
			return sgID, nil
		}
		if attempt == cloudRetryMaxAttempts-1 {
			return "", err
		}
		backoff := cloudRetryBaseBackoff << attempt
		jitter := time.Duration(rand.Int63n(int64(100 * time.Millisecond)))
		time.Sleep(backoff + jitter)
	}
	return "", hcommon.I18nError(i18n.MsgSGCreateCloudRetryExhausted)
}

// applyRulesToCloudSGWithRetry 下发规则带重试。
func applyRulesToCloudSGWithRetry(ctx context.Context, sgID, rulesJSON string) error {
	for attempt := 0; attempt < cloudRetryMaxAttempts; attempt++ {
		err := controller.ApplyRulesToCloudSG(ctx, sgID, rulesJSON)
		if err == nil {
			return nil
		}
		if attempt == cloudRetryMaxAttempts-1 {
			return err
		}
		backoff := cloudRetryBaseBackoff << attempt
		jitter := time.Duration(rand.Int63n(int64(100 * time.Millisecond)))
		time.Sleep(backoff + jitter)
	}
	return hcommon.I18nError(i18n.MsgSGApplyRulesRetryExhausted)
}

// --- 云 API 辅助 ---

// guardianNewVpcClientFn 新建 VPC client 的工厂，测试可替换。
// 类型为 func() (controller.SGVpcClient, error)，返回能力比 *vpc.Client 窄但覆盖
// guardianTick 所需的 Describe/Create/Delete/Modify 方法。
var guardianNewVpcClientFn = func(ctx context.Context) (controller.SGVpcClient, error) {
	return controller.NewVpcClient(ctx)
}

// describeAssocStats 拿每个 SG 的 CVM/ENI/CDB 关联实例总数，结果填入返回 map（key=sg_id）。
//
// 设计取舍：逐个查询而非批量。
//   - 腾讯云批量 API 在批里任何一个 SG 不存在时整批返回 ResourceNotFound，
//     一颗老鼠屎打翻整批，导致 cvm_count 纠偏全废。
//   - Guardian 5 分钟才跑一次，单租户 SG 数量个位数（即使 100 个也只是 100 次串行调用），
//     性能与限流都可忽略，不值得为"看似优化"付出 fallback 代码复杂度。
//
// 错误语义（重要！RETIRE 的判定依赖这里）：
//   - 单个 SG 明确返回 ResourceNotFound（云端已不存在）→ 不进 map，调用方据此判定失踪
//   - 单个 SG 返回 transient 错误（超时/限流/网络）→ 不进 map，但 *只 warn 不视为失踪*，
//     由下一轮 Guardian 重试。下游 RETIRE 逻辑必须以"独立的 Describe 失踪信号"为准，
//     而非"map 里没有这个 sg_id"——transient 也会让 sg_id 不在 map 里。
func describeAssocStats(ctx context.Context, sgIDs []string) (map[string]int, error) {
	if len(sgIDs) == 0 {
		return nil, nil
	}
	client, err := guardianNewVpcClientFn(ctx)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgCreateVPCClientFailed)
	}
	out := make(map[string]int, len(sgIDs))
	for _, id := range sgIDs {
		req := vpc.NewDescribeSecurityGroupAssociationStatisticsRequest()
		req.SecurityGroupIds = common.StringPtrs([]string{id})
		resp, err := client.DescribeSecurityGroupAssociationStatistics(req)
		if err != nil {
			if isSGNotFoundErr(err) {
				// 云端确认失踪 —— 不进 map。下游 cvm_count 纠偏视为 cloud_count=0；
				// RETIRE 决策走独立的 guardianDetectOrphans + 单 SG Describe 二次确认，
				// 不会仅凭这里的"map 里没有"就标 RETIRED。
				continue
			}
			// transient 错误（超时/限流/网络）：跳过本轮，下一轮重试。
			// 这条 SG 后续的 cvm_count 不会被更新，但 *绝不会* 被误判为失踪。
			slog.Warn("[SGGuardian] assoc-stats failed (transient or unknown), skip", "sg", id, "err", err)
			continue
		}
		if resp.Response == nil {
			continue
		}
		for _, s := range resp.Response.SecurityGroupAssociationStatisticsSet {
			if s.SecurityGroupId == nil {
				continue
			}
			cnt := 0
			if s.CVM != nil {
				cnt += int(*s.CVM)
			}
			if s.ENI != nil {
				cnt += int(*s.ENI)
			}
			if s.CDB != nil {
				cnt += int(*s.CDB)
			}
			out[*s.SecurityGroupId] = cnt
		}
	}
	return out, nil
}

// listCloudManagedSGs 已移除：不再通过名字前缀扫描全账号 SG。
// 同一腾讯云账号下可能部署多个 hatchery 且 identifier 可能相同，名字前缀会与
// 别家重叠导致误判。现以 DB `managed_sg_pool` 为唯一真相源，通过 describeSGNames
// 按 sg_id 精确查询云端存活情况。保留 buildManagedSGName 仅作运维辨识用途。

// 辅助

func poolSGIDs(pool []model.ManagedSGPool) []string {
	out := make([]string, 0, len(pool))
	for _, p := range pool {
		out = append(out, p.SGID)
	}
	return out
}

// isClawproSG 已移除：不再按名字前缀判定 SG 归属。是否属于本 hatchery 以
// DB `managed_sg_pool` 里是否存在对应 sg_id 为准。

// isManagedPoolSG 判断 DB pool 记录是否属于 clawpro 托管。
// 以 RuleSetID 非零为准：pool 里写进来时就有 RuleSet 关联，存量无 RuleSetID
// 的老记录不参与自愈/告警。
func isManagedPoolSG(p model.ManagedSGPool) bool {
	// 有 RuleSetID 的一定是 clawpro 新建的
	if p.RuleSetID > 0 {
		return true
	}
	// 兼容无 RuleSetID 的存量旧数据（不参与自愈）
	return false
}

func truncateID(id string, max int) string {
	if len(id) <= max {
		return id
	}
	return id[:max]
}
