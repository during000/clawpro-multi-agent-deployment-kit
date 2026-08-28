package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net/netip"
	"strings"
	"sync"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
	"gorm.io/gorm"
)

// sg-ruleset-projection 方案的"SG 池"核心逻辑。
//
// 与旧 sg-managed-sharding 方案的差异：
//   * 无 base/shard 二元概念；所有 SG 平等（都是 ManagedSGPool 行）。
//   * 选 SG 按 rule_set_id 分区（未来多 RuleSet 场景的能力预留）。
//   * 阈值内正常选；阈值 ~ 硬上限 buffer 区 fallback（避免撞子池上限时创建实例失败）。

// ErrNoBaseConfigured 创建实例时 SiteConfig.SecurityGroupId 为空且 SG 初始化未完成时返回。
// 理论上初始化后总有 ACTIVE SG，此错误实际不会发生——保留为安全兜底。
var ErrNoBaseConfigured = hcommon.I18nError(i18n.MsgSGPoolBaseNotConfigured)

// ErrPoolAtHardLimit 所有 ACTIVE SG 都达到硬上限 2000（真正的池耗尽）时返回。
// 管理员需决策：加 MaxSGPerRuleSet 上限 / 拆租户 / 迁走部分实例。
var ErrPoolAtHardLimit = hcommon.I18nError(i18n.MsgSGPoolPoolExhausted)

// defaultRuleSetCache 进程内缓存 identifier -> RuleSet.ID，避免每次创建实例都查 DB。
// SG 初始化完成时 ID 确定后就不再变；多 Pod 场景各自缓存，互不影响。
var (
	defaultRuleSetCache   sync.Map // map[string]uint
	defaultRuleSetCacheMu sync.Mutex
)

// sgPoolLogAuditFn 封装 AutoScaleSG 的异步审计日志写入，允许测试替换为同步 no-op
// 避免 goroutine 在测试 cleanup 后访问已销毁的 model.DB 造成 nil 解引用 panic。
var sgPoolLogAuditFn = func(ctx context.Context, startedAt time.Time, userID uint, username, action, resource, resourceID, status string) {
	go model.LogAudit(hcommon.DetachContext(ctx), startedAt, userID, username, action, resource, resourceID, status)
}

// GetDefaultRuleSet 查询（并缓存）当前租户的 default RuleSet。
// 首次调用查 DB，后续命中 sync.Map 直接返回。
func GetDefaultRuleSet(ctx context.Context) (*model.RuleSet, error) {
	ident := model.CurrentIdentifier(ctx)
	// 缓存命中：ID 不会变，只缓存 ID 不缓存全部字段（Rules 随时间变化）
	if cachedID, ok := defaultRuleSetCache.Load(ident); ok {
		id, valid := cachedID.(uint)
		if !valid {
			// 缓存数据类型异常，清除后重查
			defaultRuleSetCache.Delete(ident)
		} else {
			var rs model.RuleSet
			if err := model.DB(ctx).First(&rs, id).Error; err == nil {
				return &rs, nil
			}
			// 缓存失效（DB 里行被删？）——清缓存重查
			defaultRuleSetCache.Delete(ident)
		}
	}

	// 加锁防止多个 goroutine 同时查 DB（thundering herd）
	defaultRuleSetCacheMu.Lock()
	defer defaultRuleSetCacheMu.Unlock()

	// double-check：拿到锁后再看一次缓存，可能已被其他 goroutine 填充
	if cachedID, ok := defaultRuleSetCache.Load(ident); ok {
		var rs model.RuleSet
		if err := model.DB(ctx).First(&rs, cachedID.(uint)).Error; err == nil {
			return &rs, nil
		}
		defaultRuleSetCache.Delete(ident)
	}

	rs, err := model.GetDefaultRuleSet(ctx)
	if err != nil {
		return nil, err
	}
	defaultRuleSetCache.Store(ident, rs.ID)
	return rs, nil
}

// InvalidateDefaultRuleSetCache 清除指定 identifier 的缓存（SG 初始化完成后调用）。
func InvalidateDefaultRuleSetCache(identifier string) {
	defaultRuleSetCache.Delete(identifier)
}

// SelectSGForNewInstance 为新实例选一个 SG。
//
// 签名带 ruleSetID（本期固定传 GetDefaultRuleSet().ID；未来 per-user_group 场景由调用方推导）。
// 返回值：
//   - sgID: 选中的 SG ID
//   - usedBuffer: 是否走到了 buffer fallback（即 cvm_count ∈ [阈值, 硬上限) 的 SG）。
//     仅用于调用方写 log/audit，不影响业务行为。
//   - err: ErrNoBaseConfigured / ErrPoolAtHardLimit / 其他 DB 错误
//
// 流程（见 design.md D6）：
//  1. 路径 1（常规）：cvm_count < 阈值 的 SG 里按 cvm_count ASC 选第一个
//  2. 路径 2（扩容）：路径 1 无结果 → AutoScaleSG 新建一个 SG
//  3. 路径 3（buffer）：AutoScaleSG 撞子池上限 → cvm_count < 硬上限 的 SG 里选最小
//  4. 路径 4（耗尽）：buffer 也空 → ErrPoolAtHardLimit
func SelectSGForNewInstance(ctx context.Context, identifier string, ruleSetID uint) (sgID string, usedBuffer bool, err error) {
	log := Logger(ctx)
	threshold := effectiveSGPoolThreshold(ctx)

	// 路径 1：常规选
	sg, err := selectActiveSGBelowCount(ctx, ruleSetID, threshold)
	if err != nil {
		return "", false, hcommon.I18nRichError(err, i18n.MsgSGPoolSelectActiveFailed)
	}
	if sg != nil {
		return sg.SGID, false, nil
	}

	// 路径 2：扩容
	rs, err := GetDefaultRuleSet(ctx)
	if err != nil {
		return "", false, ErrNoBaseConfigured.WithCause(err)
	}
	newSG, scaleErr := AutoScaleSG(ctx, identifier, rs)
	if scaleErr == nil {
		return newSG.SGID, false, nil
	}
	// 撞上限才触发 fallback；云 API 错误直接返回
	if !errors.Is(scaleErr, ErrPoolAtMaxSize) {
		return "", false, hcommon.I18nRichError(scaleErr, i18n.MsgSGPoolAutoScaleFailed)
	}

	// 路径 3：buffer fallback（cvm_count ∈ [阈值, 硬上限)）
	fallbackSG, err := selectActiveSGBelowCount(ctx, ruleSetID, model.SGPoolHardLimit)
	if err != nil {
		return "", false, hcommon.I18nRichError(err, i18n.MsgSGPoolSelectFallbackFailed)
	}
	if fallbackSG != nil {
		log.Warn("[sg-pool] pool at max size, using buffer zone",
			"rule_set_id", ruleSetID,
			"sg_id", fallbackSG.SGID,
			"cvm_count", fallbackSG.CVMCount)
		return fallbackSG.SGID, true, nil
	}

	// 路径 4：真正耗尽
	return "", false, ErrPoolAtHardLimit
}

// selectActiveSGBelowCount 返回 cvm_count < maxCount 的 ACTIVE SG 中 cvm_count 最小者；
// 查不到返回 nil, nil。
func selectActiveSGBelowCount(ctx context.Context, ruleSetID uint, maxCount int) (*model.ManagedSGPool, error) {
	var sg model.ManagedSGPool
	err := model.DB(ctx).Where("rule_set_id = ? AND status = ? AND cvm_count < ? AND drift_at IS NULL",
		ruleSetID, model.SGStatusActive, maxCount).
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

// ErrPoolAtMaxSize AutoScaleSG 因撞 MaxSGPerRuleSet 返回。
var ErrPoolAtMaxSize = errors.New("sg pool at max size")

// AutoScaleSG 新建一个 ACTIVE SG 加到池里。
//
// 流程：
//  1. 拿 update 锁 sg-ruleset-update（和 UpdateRuleSetRulesInternal 互斥；identifier 由 lockName 自动加前缀）
//     → 避免"管理员正在改规则时扩容"读到旧 RuleSet 导致新 SG 规则落后
//  2. 分布式锁 sg-scale-<rule_set_id>（按 RuleSet 独立锁，跨 RuleSet 并发扩容不阻塞）
//  3. double-check 当前子池规模，撞 MaxSGPerRuleSet → ErrPoolAtMaxSize
//  4. 重新读最新 RuleSet（拿到 update 锁期间可能已被管理员 commit 了新 version）
//  5. 云 API CreateSecurityGroup + CreateSecurityGroupPolicies（应用最新 RuleSet.Rules）
//  6. INSERT managed_sg_pool（ACTIVE, cvm_count=0, rule_version=RuleSet.Version）
//  7. 任一步失败：tryDeleteCloudSG 回收云端资源后返回错误
func AutoScaleSG(ctx context.Context, identifier string, rs *model.RuleSet) (*model.ManagedSGPool, error) {
	log := Logger(ctx)

	// 1. 先拿 update 锁：避免和管理员保存规则并发
	//    场景：管理员保存中 → fan-out 到 sg1/sg2 成功但未 commit DB → 此时扩容建 sg3 读到旧 rules → 管理员 commit → sg3 落后
	//    加锁后：扩容要么发生在 update 之前（用旧规则）要么之后（用新规则），不会读到"云端已新但 DB 还旧"的中间态
	//    锁 key 须与 ruleset_helpers.go 管理员侧保持一致（都用 "sg-ruleset-update"）；
	//    多租户隔离由 model.AcquireLock 内部的 lockName() 自动加 "{identifier}:" 前缀。
	updLock, err := model.AcquireLock(ctx, "sg-ruleset-update", 30*time.Second)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgSGPoolAcquireUpdateLockFailed)
	}
	defer updLock.Release()

	// 2. 再拿扩容锁（等 5s：高并发时让后续请求等前一个扩容完成后 double-check 复用，而非直接失败）
	//    锁 key 无需拼 identifier：model.AcquireLock 内部的 lockName() 自动加前缀。
	lockKey := fmt.Sprintf("sg-scale-%d", rs.ID)
	lock, err := model.AcquireLock(ctx, lockKey, 5*time.Second)
	if err != nil {
		// 锁没拿到——等待超时（真并发扩容耗时较长）或 DB 错误
		return nil, hcommon.I18nRichError(err, i18n.MsgSGPoolAcquireScaleLockFailed, lockKey)
	}
	defer lock.Release()

	// 3. double-check 池规模
	count, err := model.CountActiveSGsInRuleSet(ctx, rs.ID)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgSGPoolCountPoolFailed)
	}
	if count >= model.MaxSGPerRuleSet {
		log.Warn("[sg-pool] rule_set sub-pool at max size", "rule_set_id", rs.ID, "count", count)
		return nil, ErrPoolAtMaxSize
	}

	// 4. 重新读最新 RuleSet（拿到 update 锁期间可能已被 commit 新 version）
	freshRS, err := model.GetDefaultRuleSet(ctx)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgSGPoolReloadRuleSetFailed)
	}

	// 5. 计算序号 + 云 API 建 SG（PRD 4.2：clawpro-sg-{ident}-{name}-{NN}）
	ordinal, err := model.NextSGOrdinalForRuleSet(ctx, freshRS.ID)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgSGPoolComputeOrdinalFailed)
	}
	sgName := buildManagedSGName(identifier, freshRS.Name, ordinal)
	sgDesc := buildManagedSGDescription(identifier, freshRS.Name, ordinal)
	sgID, err := createCloudSG(ctx, sgName, sgDesc)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgSGPoolCreateCloudSGFailed)
	}
	// 应用规则（用最新的 freshRS.Rules）
	if err := applyRulesToCloudSG(ctx, sgID, freshRS.Rules); err != nil {
		tryDeleteCloudSG(ctx, sgID)
		return nil, hcommon.I18nRichError(err, i18n.MsgSGPoolApplyRulesFailed, sgID)
	}

	// 6. INSERT DB（rule_version 记为 freshRS.Version，和当前 RuleSet 版本一致）
	//    Identifier 由 GORM 全局 callback（set_identifier）在 Create 前自动填充，此处不显式赋值。
	row := model.ManagedSGPool{
		SGID:        sgID,
		SGName:      sgName,
		RuleSetID:   freshRS.ID,
		RuleVersion: freshRS.Version,
		Status:      model.SGStatusActive,
		CVMCount:    0,
	}
	if err := model.DB(ctx).Create(&row).Error; err != nil {
		tryDeleteCloudSG(ctx, sgID)
		return nil, hcommon.I18nRichError(err, i18n.MsgSGPoolInsertDBFailed)
	}

	log.Info("[sg-pool] auto-scaled new SG",
		"sg_id", sgID, "rule_set_id", rs.ID, "rule_version", rs.Version)
	// 审计日志（非 HTTP 上下文，直接调 model.LogAudit）
	sgPoolLogAuditFn(ctx, time.Now(), 0, "system", "auto_scale_sg", "sg", sgID, "success")
	return &row, nil
}

// MarkInstanceUnbound 实例销毁时调用：DecrementSGCVMCount。
// 兼容旧签名：忽略 identifier 参数（现在按 sg_id 直接定位，多租户隔离由 identifier callback 覆盖）。
func MarkInstanceUnbound(ctx context.Context, identifier, sgID string) {
	if sgID == "" {
		return
	}
	if err := model.DecrementSGCVMCount(ctx, sgID); err != nil {
		slog.Warn("[sg-pool] decrement cvm_count failed", "sg_id", sgID, "err", err)
	}
}

// MarkInstanceBound 实例创建成功后调用：IncrementSGCVMCount。
func MarkInstanceBound(ctx context.Context, sgID string) {
	if sgID == "" {
		return
	}
	if err := model.IncrementSGCVMCount(ctx, sgID); err != nil {
		slog.Warn("[sg-pool] increment cvm_count failed", "sg_id", sgID, "err", err)
	}
}

// effectiveSGPoolThreshold 读 SiteConfig 的阈值；零值或负值兜底 1800。
func effectiveSGPoolThreshold(ctx context.Context) int {
	cfg := model.GetSiteConfig(ctx)
	return cfg.EffectiveSGPoolAutoScaleThreshold()
}

// truncateIdentifier 超过 max 字符的 identifier 截断前 max 字符。
// SG 名总长 ≤ 60（腾讯云限制）：clawpro-sg- (11) + ident (≤20) + - (1) + name (≤20) + -NN (3) = 55 字符。
func truncateIdentifier(id string, max int) string {
	if len(id) <= max {
		return id
	}
	return id[:max]
}

// buildManagedSGName 生成云端分片名（PRD 4.2 / 4.7）：
//
//	clawpro-sg-{identifier}-{rule_set_name}-{ordinal:02d}
//
// identifier / name 长度分别截到 20 字符，ordinal 两位零填充。
// 序号超过 99 也照样输出（例如 "100"），但本期 MaxSGPerRuleSet=20 撞不到。
func buildManagedSGName(identifier, ruleSetName string, ordinal int) string {
	ident := truncateIdentifier(identifier, 20)
	nm := truncateIdentifier(ruleSetName, 20)
	return fmt.Sprintf("clawpro-sg-%s-%s-%02d", ident, nm, ordinal)
}

// buildManagedSGDescription 生成云端分片的描述（PRD 4.2）：
//
//	【请勿删除】ClawPro 自动管理的安全组（{identifier}-{rule_set_name} #{ordinal}），手动改动无效，误删会造成业务短暂失联
func buildManagedSGDescription(identifier, ruleSetName string, ordinal int) string {
	return fmt.Sprintf(
		"【请勿删除】ClawPro 自动管理的安全组（%s-%s #%d），手动改动无效，误删会造成业务短暂失联",
		identifier, ruleSetName, ordinal,
	)
}

// shortRand 返回 n 个字符的随机小写字母+数字。
func shortRand(n int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// createCloudSG 调云 API 新建一个空 SG，返回 sg_id。
func createCloudSG(ctx context.Context, name, desc string) (string, error) {
	client, err := newVpcClientForSGFn(ctx)
	if err != nil {
		return "", err
	}
	req := vpc.NewCreateSecurityGroupRequest()
	req.GroupName = common.StringPtr(name)
	req.GroupDescription = common.StringPtr(desc)
	resp, err := client.CreateSecurityGroup(req)
	if err != nil {
		return "", err
	}
	if resp.Response == nil || resp.Response.SecurityGroup == nil ||
		resp.Response.SecurityGroup.SecurityGroupId == nil {
		return "", hcommon.I18nError(i18n.MsgSGPoolCreateSGEmptyResp)
	}
	return *resp.Response.SecurityGroup.SecurityGroupId, nil
}

// applyRulesToCloudSG 把 RuleSet.Rules JSON 应用到指定 SG（整包覆盖语义）。
//
// 分两条路径：
//   - merged 非空：调 ModifySecurityGroupPolicies 整包覆盖
//   - merged 为空（rulesJSON="" 或 "[]"）：腾讯云 ModifySecurityGroupPolicies 不接受空 PolicySet，
//     改走 clearAllRulesForSG 路径——先 Describe 拿到当前所有规则，再分别 DeleteSecurityGroupPolicies
//     删除 ingress / egress（腾讯云 API 限制：一个请求只能删除单方向）
func applyRulesToCloudSG(ctx context.Context, sgID, rulesJSON string) error {
	client, err := newVpcClientForSGFn(ctx)
	if err != nil {
		return err
	}
	policySet, err := rulesJSONToPolicySet(rulesJSON)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSGPoolParseRulesJSONFailed)
	}
	// 空规则集合走真清空路径（避免腾讯云 InvalidParameterValue.Empty）
	if len(policySet.Ingress) == 0 && len(policySet.Egress) == 0 {
		return clearAllRulesForSG(client, sgID)
	}
	modReq := vpc.NewModifySecurityGroupPoliciesRequest()
	modReq.SecurityGroupId = common.StringPtr(sgID)
	modReq.SecurityGroupPolicySet = policySet
	// SortPolicys=true：严格按入参 Policies 数组顺序重置规则，自上而下匹配。
	// 这是 RuleSet.Rules 数组顺序在云端真实生效的前提；reorder 接口和 MergeRequiredRules
	// 的"保留原序"语义共同保证 DB 顺序 = 云端实际匹配顺序。
	modReq.SortPolicys = common.BoolPtr(true)
	_, err = client.ModifySecurityGroupPolicies(modReq)
	return err
}

// clearAllRulesForSG 清空指定 SG 的所有规则（ingress + egress）。
//
// 实现：
//  1. DescribeSecurityGroupPolicies 拿到当前所有规则的 PolicyIndex
//  2. 按方向分两次调 DeleteSecurityGroupPolicies（腾讯云 API 限制：一个请求只能删除单方向）
//  3. ⚠️ Delete 请求只传 PolicyIndex，不传其他字段——否则 Describe 返回的 Policy 里
//     可能含空 AddressTemplate 等无效字段，Delete API 会报 InvalidParameterValue.Malformed
//
// 幂等：如云端本来就空 → Describe 返回空 → 跳过 Delete 直接 return nil。
func clearAllRulesForSG(client sgVpcClient, sgID string) error {
	descReq := vpc.NewDescribeSecurityGroupPoliciesRequest()
	descReq.SecurityGroupId = common.StringPtr(sgID)
	descResp, err := client.DescribeSecurityGroupPolicies(descReq)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSGPoolDescribeForClearFailed)
	}
	if descResp.Response == nil || descResp.Response.SecurityGroupPolicySet == nil {
		return nil
	}
	policySet := descResp.Response.SecurityGroupPolicySet

	// 删除 ingress（仅传 PolicyIndex 避免 AddressTemplate 等字段污染）
	if len(policySet.Ingress) > 0 {
		ingressByIndex := make([]*vpc.SecurityGroupPolicy, 0, len(policySet.Ingress))
		for _, p := range policySet.Ingress {
			if p == nil || p.PolicyIndex == nil {
				continue
			}
			ingressByIndex = append(ingressByIndex, &vpc.SecurityGroupPolicy{PolicyIndex: p.PolicyIndex})
		}
		if len(ingressByIndex) > 0 {
			delReq := vpc.NewDeleteSecurityGroupPoliciesRequest()
			delReq.SecurityGroupId = common.StringPtr(sgID)
			delReq.SecurityGroupPolicySet = &vpc.SecurityGroupPolicySet{
				Ingress: ingressByIndex,
			}
			if _, err := client.DeleteSecurityGroupPolicies(delReq); err != nil {
				return hcommon.I18nRichError(err, i18n.MsgSGPoolDeleteIngressFailed)
			}
		}
	}
	// 删除 egress（仅传 PolicyIndex）
	if len(policySet.Egress) > 0 {
		egressByIndex := make([]*vpc.SecurityGroupPolicy, 0, len(policySet.Egress))
		for _, p := range policySet.Egress {
			if p == nil || p.PolicyIndex == nil {
				continue
			}
			egressByIndex = append(egressByIndex, &vpc.SecurityGroupPolicy{PolicyIndex: p.PolicyIndex})
		}
		if len(egressByIndex) > 0 {
			delReq := vpc.NewDeleteSecurityGroupPoliciesRequest()
			delReq.SecurityGroupId = common.StringPtr(sgID)
			delReq.SecurityGroupPolicySet = &vpc.SecurityGroupPolicySet{
				Egress: egressByIndex,
			}
			if _, err := client.DeleteSecurityGroupPolicies(delReq); err != nil {
				return hcommon.I18nRichError(err, i18n.MsgSGPoolDeleteEgressFailed)
			}
		}
	}
	return nil
}

// rulesJSONToPolicySet 把 RuleSet.Rules 里存的 []Rule JSON 反序列化成 vpc SecurityGroupPolicySet。
// 区分 Direction=INGRESS → Ingress 数组；Direction=EGRESS → Egress 数组；大小写不敏感。
// 空/无效 JSON 视为空 policy set（等于清空规则）。
func rulesJSONToPolicySet(rulesJSON string) (*vpc.SecurityGroupPolicySet, error) {
	set := &vpc.SecurityGroupPolicySet{
		Ingress: []*vpc.SecurityGroupPolicy{},
		Egress:  []*vpc.SecurityGroupPolicy{},
	}
	if strings.TrimSpace(rulesJSON) == "" || rulesJSON == "[]" {
		return set, nil
	}
	var rules []Rule
	if err := json.Unmarshal([]byte(rulesJSON), &rules); err != nil {
		return nil, err
	}
	for i := range rules {
		r := &rules[i]
		policy := ruleToPolicy(r)
		switch strings.ToUpper(r.Direction) {
		case "INGRESS":
			set.Ingress = append(set.Ingress, policy)
		case "EGRESS":
			set.Egress = append(set.Egress, policy)
		default:
			// 未知 direction 视为 INGRESS（保守），但记录警告以暴露配置错误
			slog.Warn("[sg-pool] unknown rule direction, defaulting to INGRESS",
				"direction", r.Direction, "rule_index", i, "cidr", r.CidrBlock)
			set.Ingress = append(set.Ingress, policy)
		}
	}
	return set, nil
}

// ruleToPolicy Rule JSON → vpc.SecurityGroupPolicy 的字段映射（不包含 Direction，调用方按 Direction 分桶）。
//
// ⚠️ 来源字段路由（本 change 升级，五选一互斥）：
// 按 Rule.CidrBlock 字符串的前缀分发到腾讯云 SDK 对应字段——这四类字段在 API 层面互斥：
//
//	前缀 `sg-`    → p.SecurityGroupId          （安全组作为来源/目的）
//	前缀 `ipmg-`  → p.AddressTemplate.AddressGroupId  （IP 地址模板组，必须先于 ipm- 判断）
//	前缀 `ipm-`   → p.AddressTemplate.AddressId       （IP 地址模板）
//	含 `:`        → p.Ipv6CidrBlock             （IPv6 CIDR）
//	其他          → p.CidrBlock                 （IPv4 CIDR）
//
// 这样做的好处：DB 存量 JSON schema 零扩展、前端 UI 零改动；对用户而言 "source" 字段可直接填
// sg-xxx / ipm-xxx / ipmg-xxx 字符串，后端识别前缀自动路由。
//
// ⚠️ 历史 IPv6 说明：腾讯云 VPC SDK 的 CidrBlock / Ipv6CidrBlock 是两个字段；IPv6 CIDR
// （含冒号）必须塞到 Ipv6CidrBlock，否则 `.SecurityGroupPolicySet.Egress.N.CidrBlock` 会报
// InvalidParameterValue。本函数的 classifyRuleSource 分支保证了这一点。
//
// ⚠️ Protocol / Port 的 "ALL" 归一化：
//
//	本地 RuleSet schema（validateRules）允许把 "ALL" 作为"所有协议/所有端口"的业务层简写，
//	存储与 UI 展示都以此为准。但腾讯云 ReplaceSecurityGroupPolicies API 对 Protocol=="ALL"
//	直接返回 InvalidParameterValue —— 云端要求用"字段不传（nil）"来表达"全部"语义。
//	这里在适配层统一把 "ALL" 翻成 nil，避免调云失败。Port 同理。
func ruleToPolicy(r *Rule) *vpc.SecurityGroupPolicy {
	p := &vpc.SecurityGroupPolicy{}
	// ⚠️ 腾讯云 ModifySecurityGroupPolicies API 对 Protocol="ALL" 返回 InvalidParameterValue，
	// 要求"省略 Protocol 字段"才能表达"全部协议"。Port 同理。
	// （注意：CreateSecurityGroupPolicies 行为不同，接受 "ALL" 字符串。本函数仅服务于 Modify 路径。）
	if r.Protocol != "" && !strings.EqualFold(r.Protocol, "ALL") {
		p.Protocol = common.StringPtr(r.Protocol)
	}
	if r.Port != "" && !strings.EqualFold(r.Port, "ALL") {
		p.Port = common.StringPtr(r.Port)
	}
	// 来源字段按前缀分发（D2，顺序敏感：ipmg- 必须先于 ipm-；classifyRuleSource 已保证）
	if r.CidrBlock != "" {
		src := r.CidrBlock
		switch classifyRuleSource(src) {
		case srcSG:
			p.SecurityGroupId = common.StringPtr(src)
		case srcAddressGroup:
			p.AddressTemplate = &vpc.AddressTemplateSpecification{
				AddressGroupId: common.StringPtr(src),
			}
		case srcAddressTpl:
			p.AddressTemplate = &vpc.AddressTemplateSpecification{
				AddressId: common.StringPtr(src),
			}
		case srcIPv6CIDR:
			p.Ipv6CidrBlock = common.StringPtr(src)
		default: // srcIPv4CIDR / srcUnknown：按 IPv4 CIDR 处理，腾讯云兜底校验
			p.CidrBlock = common.StringPtr(src)
		}
	}
	if r.Action != "" {
		p.Action = common.StringPtr(r.Action)
	}
	if r.PolicyDescription != "" {
		p.PolicyDescription = common.StringPtr(r.PolicyDescription)
	}
	return p
}

// isIPv6CIDR 判断给定的 CIDR 字符串是否是 IPv6 地址。
// 支持带前缀（如 "::1/128"）和不带前缀（如 "::1"）两种格式。
func isIPv6CIDR(cidr string) bool {
	// 先尝试解析为 CIDR（如 "2001:db8::/32"）
	if prefix, err := netip.ParsePrefix(cidr); err == nil {
		return prefix.Addr().Is6()
	}
	// 再尝试解析为纯 IP（如 "::1"）
	if addr, err := netip.ParseAddr(cidr); err == nil {
		return addr.Is6()
	}
	// 解析失败：无法识别的格式不应被视为 IPv6
	return false
}

// tryDeleteCloudSG AutoScaleSG/SG初始化 失败时的回滚：删除云端新建但未入 DB 的 SG。
// 本函数是 best-effort：删失败也不返回错误（Guardian 反向扫描会告警）。
func tryDeleteCloudSG(ctx context.Context, sgID string) {
	client, err := newVpcClientForSGFn(ctx)
	if err != nil {
		slog.Warn("[sg-pool] rollback: new vpc client failed", "sg_id", sgID, "err", err)
		return
	}
	req := vpc.NewDeleteSecurityGroupRequest()
	req.SecurityGroupId = common.StringPtr(sgID)
	if _, err := client.DeleteSecurityGroup(req); err != nil {
		slog.Warn("[sg-pool] rollback delete cloud sg failed", "sg_id", sgID, "err", err)
	}
}

// IsClawproManagedSGName 判断 SG 名称前缀是否是 clawpro 自建（用于 list-cloud 过滤兜底）。
func IsClawproManagedSGName(name string) bool {
	return strings.HasPrefix(name, "clawpro-sg-")
}

// --- 导出函数供 task 包调用 ---

// CreateCloudSG 创建一个空云端 SG，返回 sg_id。
func CreateCloudSG(ctx context.Context, name, desc string) (string, error) {
	return createCloudSG(ctx, name, desc)
}

// ApplyRulesToCloudSG 将 rules JSON 整包覆盖到指定 SG。
func ApplyRulesToCloudSG(ctx context.Context, sgID, rulesJSON string) error {
	return applyRulesToCloudSG(ctx, sgID, rulesJSON)
}

// TryDeleteCloudSG best-effort 删除云端 SG（回滚用）。
func TryDeleteCloudSG(ctx context.Context, sgID string) {
	tryDeleteCloudSG(ctx, sgID)
}

// BuildManagedSGName 生成云端分片名（供 Guardian 自愈重建使用）。
func BuildManagedSGName(identifier, ruleSetName string, ordinal int) string {
	return buildManagedSGName(identifier, ruleSetName, ordinal)
}

// BuildManagedSGDescription 生成云端分片描述。
func BuildManagedSGDescription(identifier, ruleSetName string, ordinal int) string {
	return buildManagedSGDescription(identifier, ruleSetName, ordinal)
}
