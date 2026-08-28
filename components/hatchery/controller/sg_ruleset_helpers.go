package controller

// sg_ruleset_helpers.go —— 端口放通 / 规则查询的公共 helper。
//
// 本文件是 OpenSpec change migrate-port-open-to-ruleset 的产物。
// 设计目标：把"端口放通类业务"从直接读 SiteConfig.SecurityGroupId 的老模型
// 迁移到 RuleSet + ManagedSGPool 新模型的唯一入口。
//
// ⚠️ 新增 controller 代码 MUST NOT：
//   - 读 SiteConfig.SecurityGroupId 作为"当前 SG"做业务判断（展示用途除外，且单独立项废弃）
//   - 直接调用 vpcClient.CreateSecurityGroupPolicies / DescribeSecurityGroupPolicies 操作
//     端口放通；必须通过本文件提供的 helper 间接完成
//
// 本文件提供的 helper（详见 openspec/changes/migrate-port-open-to-ruleset/）：
//   - listActiveSGIDsByRuleSet(ruleSetID)     按 RuleSet 枚举 ACTIVE SG
//   - listAllActiveSGIDs()                    跨全部 RuleSet 的 ACTIVE SG 并集（配额统计）
//   - resolveRuleSetIDForInstance(inst)       实例 → 所属 RuleSet 反查
//   - checkPortRuleOnInstanceSG(inst,...)     drift-aware 的实例视角端口查询
//   - ensureRuleInRuleSet(ctx, rsID, rule)    单 RuleSet 追加规则 + 扇出（复用 UpdateRuleSetRulesInternal）
//   - ensureRuleOnInstanceRuleSet(ctx, inst, rule)  实例视角便捷包装
//   - ensureRuleInAllRuleSets(ctx, rule)      ClawPro 系统必需规则写入所有 RuleSet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
	"gorm.io/gorm"
)

// ============================================================================
// 1. 枚举类 helper（无副作用，纯 DB 查询）
// ============================================================================

// listActiveSGIDsByRuleSet 返回指定 RuleSet 下 status=ACTIVE 的 SGID 列表。
// 顺序按 id ASC 稳定；空结果返回 []string{} 非 nil。
// Identifier callback 会自动注入多租户条件。
func listActiveSGIDsByRuleSet(ctx context.Context, ruleSetID uint) ([]string, error) {
	var sgIDs []string
	err := model.DB(ctx).Model(&model.ManagedSGPool{}).
		Where("rule_set_id = ? AND status = ?", ruleSetID, model.SGStatusActive).
		Order("id ASC").
		Pluck("sg_id", &sgIDs).Error
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgSGRulesetListActiveByIDFailed, ruleSetID)
	}
	if sgIDs == nil {
		sgIDs = []string{}
	}
	return sgIDs, nil
}

// listAllActiveSGIDs 返回本 identifier 下所有 RuleSet 的 ACTIVE SG 并集。
// 用于配额统计等租户级场景；不是按 RuleSet 隔离的写入路径。
func listAllActiveSGIDs(ctx context.Context) ([]string, error) {
	var sgIDs []string
	err := model.DB(ctx).Model(&model.ManagedSGPool{}).
		Where("status = ?", model.SGStatusActive).
		Order("id ASC").
		Pluck("sg_id", &sgIDs).Error
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgSGRulesetListAllActiveFailed)
	}
	if sgIDs == nil {
		sgIDs = []string{}
	}
	return sgIDs, nil
}

// resolveRuleSetIDForInstance 通过 inst.SecurityGroupID 反查 managed_sg_pool.rule_set_id。
// 孤儿实例（SG 不在任何 pool 里）返回明确错误，由上层 controller 转成用户友好提示。
func resolveRuleSetIDForInstance(ctx context.Context, inst *model.Instance) (uint, error) {
	if inst == nil {
		return 0, hcommon.I18nError(i18n.MsgSGRulesetInstanceNil)
	}
	sgID := strings.TrimSpace(inst.SecurityGroupId)
	if sgID == "" {
		return 0, hcommon.I18nError(i18n.MsgSGRulesetInstanceNoSG, inst.InstanceId)
	}
	var row model.ManagedSGPool
	err := model.DB(ctx).Select("rule_set_id").Where("sg_id = ?", sgID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, hcommon.I18nError(i18n.MsgSGRulesetSGNotInPool, sgID)
	}
	if err != nil {
		return 0, hcommon.I18nRichError(err, i18n.MsgSGRulesetLookupPoolBySG, sgID)
	}
	if row.RuleSetID == 0 {
		return 0, hcommon.I18nError(i18n.MsgSGRulesetLegacyRuleSetID, sgID)
	}
	return row.RuleSetID, nil
}

// ============================================================================
// 2. 查询类 helper —— drift-aware 实例视角端口放通查询
// ============================================================================

// checkPortRuleOnInstanceSG 判断实例绑的 SG 是否已放通 (port, proto)。
//
// 流程（design.md D7）：
//  1. 通过 inst.SecurityGroupID 查 managed_sg_pool.rule_version + rule_set_id
//  2. 查对应 RuleSet 的 version
//  3. rule_version == version   → 同步态：走 DB 快路径读 Rules
//     rule_version <  version   → 漂移态：调 DescribeSecurityGroupPolicies 读云端实际规则
//  4. drifting 返回值让 UI 可选择性提示"规则同步中"
//
// proto 支持 "TCP" / "UDP" / "ICMP"；规则里的 "ALL" 视为覆盖。
// 规则匹配语义：按 Ingress 顺序遍历，首条匹配的 ACCEPT 返回 allowed=true；
// 首条匹配的 DROP 返回 allowed=false；遍历完无匹配返回 allowed=false。
// 与 checkSecurityGroupIngressForPort 的优先级语义一致。
//
// opts 可选参数（variadic，向后兼容）：
//   - anyCIDR=true：同步态不检查 CidrBlock（适用于 VNC 等白名单 IP 场景）；
//   - sourceIP 非空：按来源 IP 匹配 CidrBlock。
type portRuleCheckOptions struct {
	anyCIDR  bool
	sourceIP string
}

func checkPortRuleOnInstanceSG(ctx context.Context, inst *model.Instance, port int, proto string, opts ...portRuleCheckOptions) (allowed, drifting bool, err error) {
	var opt portRuleCheckOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	sourceIP := strings.TrimSpace(opt.sourceIP)
	if inst == nil {
		return false, false, hcommon.I18nError(i18n.MsgSGRulesetInstanceNil)
	}
	sgID := strings.TrimSpace(inst.SecurityGroupId)
	if sgID == "" {
		return false, false, hcommon.I18nError(i18n.MsgSGRulesetInstanceNoSG, inst.InstanceId)
	}

	// 0. 全新租户保护：本 identifier 无任何 RuleSet 时返回 sentinel 错误
	var rsCount int64
	if err := model.DB(ctx).Model(&model.RuleSet{}).Count(&rsCount).Error; err != nil {
		return false, false, hcommon.I18nRichError(err, i18n.MsgSGRulesetCountRuleSetsFailed)
	}
	if rsCount == 0 {
		return false, false, ErrSGBootstrapNotDone
	}

	// 1. 查 SG 所属的 RuleSet 和当前 rule_version
	var sgRow model.ManagedSGPool
	err = model.DB(ctx).Where("sg_id = ?", sgID).First(&sgRow).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, false, hcommon.I18nError(i18n.MsgSGRulesetSGNotInPool, sgID)
	}
	if err != nil {
		return false, false, hcommon.I18nRichError(err, i18n.MsgSGRulesetLookupPoolFailed)
	}
	if sgRow.RuleSetID == 0 {
		return false, false, hcommon.I18nError(i18n.MsgSGRulesetLegacyRuleSetID, sgID)
	}

	// 2. 查 RuleSet 的 version + rules
	var rs model.RuleSet
	err = model.DB(ctx).First(&rs, sgRow.RuleSetID).Error
	if err != nil {
		return false, false, hcommon.I18nRichError(err, i18n.MsgSGRulesetLookupRuleSetFailed, sgRow.RuleSetID)
	}

	// 3. 判 drift 决定走 DB 还是云 API
	if sgRow.RuleVersion == rs.Version && sgRow.DriftAt == nil {
		// 同步态：走 DB 快路径
		var rules []Rule
		if rs.Rules != "" {
			if err := json.Unmarshal([]byte(rs.Rules), &rules); err != nil {
				return false, false, hcommon.I18nRichError(err, i18n.MsgSGPoolParseRulesJSONFailed)
			}
		}
		if sourceIP != "" {
			return portCoveredByRulesForSource(rules, port, proto, sourceIP), false, nil
		}
		if opt.anyCIDR {
			// VNC 等白名单 IP 场景：不限制 CidrBlock
			return portCoveredByRulesAnyCIDR(rules, port, proto), false, nil
		}
		return portCoveredByRules(rules, port, proto), false, nil
	}

	// 4. 漂移态：调云 API 读实际规则
	Logger(ctx).Info("[SGHelper] drift detected, querying cloud API",
		"sg_id", sgID, "sg_rule_version", sgRow.RuleVersion, "ruleset_version", rs.Version)
	allowedByCloud, cloudErr := checkSecurityGroupIngressForPort(ctx, sgID, port, portRuleCheckOptions{sourceIP: sourceIP})
	if cloudErr != nil {
		return false, true, cloudErr
	}
	return allowedByCloud, true, nil
}

// ErrSGBootstrapNotDone 全新租户尚未完成 SG 初始化的 sentinel。
// 调用方用 errors.Is(err, ErrSGBootstrapNotDone) 识别后，转成对用户友好的提示。
var ErrSGBootstrapNotDone = hcommon.I18nError(i18n.MsgSGBootstrapNotDone)

// portCoveredByRules 判断一组 Rule 是否放通 (port, proto) 的入站访问。
//
// 语义（design.md D4）：
//   - 按顺序遍历 INGRESS 规则；第一条匹配 (port, proto) 的规则：
//   - Action=ACCEPT 返回 true
//   - Action=DROP/REJECT 返回 false
//   - 遍历完无匹配返回 false
//
// 匹配条件：
//   - Direction 必须是 INGRESS
//   - Protocol 匹配 proto 或等于 ALL
//   - Port 等于 ALL、或数字等于 port、或范围段包含 port
//   - CidrBlock 为 0.0.0.0/0（全公网可达才算"已放通"；更严格的场景由调用方扩展）
func portCoveredByRules(rules []Rule, port int, proto string) bool {
	for _, r := range rules {
		if !ruleMatchesPortProto(r, port, proto) {
			continue
		}
		// CidrBlock 只接受 0.0.0.0/0（全公网可达）才视为"对用户放通"
		if strings.TrimSpace(r.CidrBlock) != "0.0.0.0/0" {
			continue
		}
		// 命中：按 Action 返回
		switch strings.ToUpper(strings.TrimSpace(r.Action)) {
		case "ACCEPT":
			return true
		default:
			return false // DROP / REJECT / 未知
		}
	}
	return false
}

// portCoveredByRulesForSource 判断一组 Rule 是否允许 sourceIP 访问 (port, proto)。
//
// 与 portCoveredByRules 的区别：按真实来源 IP 匹配 CidrBlock；用于 Gateway 面板
// 访问预检，白名单 ACCEPT 在兜底 DROP 前命中时应视为已放通。
func portCoveredByRulesForSource(rules []Rule, port int, proto, sourceIP string) bool {
	for _, r := range rules {
		if !ruleMatchesPortProto(r, port, proto) {
			continue
		}
		if !cidrMatchesSource(r.CidrBlock, sourceIP) {
			continue
		}
		switch strings.ToUpper(strings.TrimSpace(r.Action)) {
		case "ACCEPT":
			return true
		default:
			return false
		}
	}
	return false
}

func cidrMatchesSource(cidrBlock, sourceIP string) bool {
	cidrBlock = strings.TrimSpace(cidrBlock)
	if cidrBlock == "" {
		return false
	}
	ip := net.ParseIP(strings.TrimSpace(sourceIP))
	if ip == nil {
		return false
	}
	if !strings.Contains(cidrBlock, "/") {
		return ip.Equal(net.ParseIP(cidrBlock))
	}
	_, network, err := net.ParseCIDR(cidrBlock)
	if err != nil {
		return false
	}
	return network.Contains(ip)
}

// portCoveredByRulesAnyCIDR 判断一组 Rule 是否放通 (port, proto) 的入站访问（不限制 CidrBlock）。
//
// 与 portCoveredByRules 的区别：不检查 CidrBlock，只要端口+协议+方向匹配且 Action=ACCEPT 即视为放通。
// 语义与漂移态 checkSecurityGroupIngressForPort 一致（漂移态也不检查 CidrBlock）。
func portCoveredByRulesAnyCIDR(rules []Rule, port int, proto string) bool {
	for _, r := range rules {
		if !ruleMatchesPortProto(r, port, proto) {
			continue
		}
		// 不检查 CidrBlock —— 任何来源 IP 的 ACCEPT 规则都视为"已放通"
		switch strings.ToUpper(strings.TrimSpace(r.Action)) {
		case "ACCEPT":
			return true
		default:
			return false // DROP / REJECT / 未知
		}
	}
	return false
}

func ruleMatchesPortProto(r Rule, port int, proto string) bool {
	if normalizeDirection(r.Direction) != "INGRESS" {
		return false
	}
	ruleProto := strings.ToUpper(strings.TrimSpace(r.Protocol))
	targetProto := strings.ToUpper(strings.TrimSpace(proto))
	if ruleProto != "ALL" && ruleProto != targetProto {
		return false
	}
	return portInRuleRange(r.Port, port)
}

// portInRuleRange 判断端口号 port 是否落在规则 Port 描述里。
// 规则 Port 支持 "ALL" / 数字 / "lo-hi" 范围 / 数字逗号列表。
func portInRuleRange(rulePort string, port int) bool {
	p := strings.TrimSpace(rulePort)
	if p == "" || strings.EqualFold(p, "ALL") {
		return true
	}
	// 复用 common.PortMatchesRule
	return hcommon.PortMatchesRule(p, port)
}

// ============================================================================
// 3. 追加类 helper —— 通过 UpdateRuleSetRulesInternal 完成 2PC fan-out
// ============================================================================

// ensureRuleInRuleSet 幂等地在指定 RuleSet 里追加一条规则。
//
// 幂等语义：若现有 Rules 已"能放通"该规则（同 port/proto 的更宽规则已存在），直接返回 nil
// 不 bump version，不调云 API。否则追加新规则并调 UpdateRuleSetRulesInternal 完成 2PC fan-out
// （并发下发到该 RuleSet 的所有 ACTIVE SG，失败自动回滚已成功 SG 的云端规则）。
//
// 失败处理完全复用 UpdateRuleSetRulesInternal：云端回滚成功则 DB 维持旧 version；只有
// 云端回滚也失败的 SG 才 MarkSGDrift 交 Guardian。
//
// 成功追加时写一条 action=rule_set_append_port_rule 的审计（补充外层 handler 的审计粒度）。
func ensureRuleInRuleSet(ctx context.Context, ruleSetID uint, newRule Rule) error {
	log := Logger(ctx)
	startedAt := time.Now()

	// 1. 读 RuleSet
	var rs model.RuleSet
	if err := model.DB(ctx).First(&rs, ruleSetID).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSGRulesetLoadRuleSetFailed, ruleSetID)
	}

	// 2. 解析现有 Rules（可能为空）
	var existing []Rule
	if strings.TrimSpace(rs.Rules) != "" {
		if err := json.Unmarshal([]byte(rs.Rules), &existing); err != nil {
			return hcommon.I18nRichError(err, i18n.MsgSGPoolParseRulesJSONFailed)
		}
	}

	// 3. 判重 —— 已能放通则幂等返回
	//    newRule 必须是 INGRESS ACCEPT 才能走 portCoveredByRules 路径；
	//    其他场景（EGRESS / DROP）不在本函数 scope 内，按指纹严格去重
	if isIngressAcceptRule(newRule) {
		port, portOK := singlePortFromRule(newRule.Port)
		if portOK && portCoveredByRules(existing, port, newRule.Protocol) {
			log.Info("[SGHelper] rule already covered, skip fan-out",
				"rule_set_id", ruleSetID, "port", port, "proto", newRule.Protocol)
			return nil
		}
	}
	// fallback 指纹去重（非 port/proto 语义的规则）
	fp := newRule.Fingerprint()
	for _, e := range existing {
		if e.Fingerprint() == fp {
			log.Info("[SGHelper] rule fingerprint already exists, skip fan-out",
				"rule_set_id", ruleSetID, "fp", fp)
			return nil
		}
	}

	// 4. 分离出"用户规则"和"必需规则"——只把用户规则传给 UpdateRuleSetRulesInternal，
	//    内部会再次 MergeRequiredRules；这样新加的 newRule 作为用户规则加入。
	userRules := make([]Rule, 0, len(existing)+1)
	for _, e := range existing {
		if e.IsRequired {
			continue // 必需规则由 MergeRequiredRules 统一补回
		}
		userRules = append(userRules, e)
	}
	userRules = append(userRules, newRule)

	// 5. 调用既有 2PC helper 完成 fan-out
	newVersion, synced, driftErrs, err := UpdateRuleSetRulesInternal(ctx, rs.Name, userRules, true /* autoFixRules: 系统反推必需规则链路 */)
	if err != nil {
		// 失败也记一条 status=failed 的审计便于追溯
		model.LogAudit(ctx, startedAt, 0, "rule-fanout-engine", "rule_set_append_port_rule", "rule_set",
			fmt.Sprintf("%d", ruleSetID), "failed")
		return hcommon.I18nRichError(err, i18n.MsgSGRulesetFanOutFailed, ruleSetID)
	}
	if len(driftErrs) > 0 {
		// 理论上 err==nil 时 driftErrs 也应为空；防御性检查
		model.LogAudit(ctx, startedAt, 0, "rule-fanout-engine", "rule_set_append_port_rule", "rule_set",
			fmt.Sprintf("%d", ruleSetID), "failed")
		return hcommon.I18nError(i18n.MsgSGRulesetFanOutPartialDrift, ruleSetID, len(driftErrs))
	}
	// 成功审计
	model.LogAudit(ctx, startedAt, 0, "rule-fanout-engine", "rule_set_append_port_rule", "rule_set",
		fmt.Sprintf("%d", ruleSetID), "success")
	log.Info("[SGHelper] rule appended and fanned out",
		"rule_set_id", ruleSetID, "new_version", newVersion, "synced_sgs", synced,
		"port", newRule.Port, "proto", newRule.Protocol)
	return nil
}

// ensureRuleOnInstanceRuleSet 便捷包装：通过实例反查所属 RuleSet，再调 ensureRuleInRuleSet。
func ensureRuleOnInstanceRuleSet(ctx context.Context, inst *model.Instance, rule Rule) error {
	rsID, err := resolveRuleSetIDForInstance(ctx, inst)
	if err != nil {
		return err
	}
	return ensureRuleInRuleSet(ctx, rsID, rule)
}

// ensureRuleInAllRuleSets 把一条 ClawPro 系统必需规则写入本 identifier 的所有 RuleSet。
//
// 原子性：不保证跨 RuleSet 原子 —— 任一 RuleSet 失败即返回 error，前面已成功的 RuleSet
// 保留新规则。调用方幂等重试即可收敛（见 design.md D2b）。
func ensureRuleInAllRuleSets(ctx context.Context, rule Rule) error {
	log := Logger(ctx)
	var rsIDs []uint
	if err := model.DB(ctx).Model(&model.RuleSet{}).Order("id ASC").Pluck("id", &rsIDs).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSGRulesetListRuleSetsFailed)
	}
	if len(rsIDs) == 0 {
		return ErrSGBootstrapNotDone
	}
	for _, rsID := range rsIDs {
		if err := ensureRuleInRuleSet(ctx, rsID, rule); err != nil {
			log.Warn("[SGHelper] ensureRuleInAllRuleSets failed on ruleset (partial progress retained)",
				"rule_set_id", rsID, "err", err)
			return hcommon.I18nRichError(err, i18n.MsgSGRulesetEnsureRuleFailed, rsID)
		}
	}
	log.Info("[SGHelper] system rule ensured in all rulesets",
		"ruleset_count", len(rsIDs), "port", rule.Port, "proto", rule.Protocol)
	return nil
}

// ============================================================================
// 4. 工具函数
// ============================================================================

// RefreshAllRuleSetsForRequiredRules 对本 identifier 每个 RuleSet 触发一次规则重投影。
// 将 DB 中完整的 existing 规则（含 is_required=true）直接传给 UpdateRuleSetRulesInternal，
// 由 MergeRequiredRules 按 fingerprint 去重：已有规则保持不变，新增/移除的
// recommended 规则（gateway/VNC）按 SiteConfig 当前条件自然增删。
//
// ⚠️ 本路径永远不会注入 builtin（allow_internet/allow_ssh）；builtin 仅在
//
//	首次新建 RuleSet（CreateInitialRuleSetAndSG）入口注入。
//
// 内部会：
//   - 合入启用的 recommended 规则（按 fingerprint 去重，required 覆盖）
//   - 扇出整包规则到该 RuleSet 的 ACTIVE 池全部 SG
//   - 云端 SG 被整包覆盖，老的不符合当前 recommended 规则集的条目被清除
//
// 场景：admin 打开云端浏览器 / Gateway UI 开关时，需让 allow_vnc_whitelist、Gateway UI 端口
// 等"条件必需规则"立即生效到所有 ACTIVE SG；启动迁移也用同一路径。
//
// 错误处理策略（best-effort）：单个 RuleSet 刷新失败时记录日志并继续处理剩余 RuleSet，
// 避免因一个 RuleSet 失败而阻断其他 RuleSet 的刷新导致状态不一致。
// 全部遍历完成后，若有失败则汇总错误返回；调用方可幂等重试收敛。
func RefreshAllRuleSetsForRequiredRules(ctx context.Context) error {
	log := Logger(ctx)
	var rsIDs []uint
	if err := model.DB(ctx).Model(&model.RuleSet{}).Order("id ASC").Pluck("id", &rsIDs).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSGRulesetListRuleSetsFailed)
	}
	if len(rsIDs) == 0 {
		return ErrSGBootstrapNotDone
	}

	var failedCount int
	var lastErr error
	for _, rsID := range rsIDs {
		var rs model.RuleSet
		if err := model.DB(ctx).First(&rs, rsID).Error; err != nil {
			log.Warn("[SGHelper] load rule_set failed, skipping", "rule_set_id", rsID, "err", err)
			failedCount++
			lastErr = err
			continue
		}
		var existing []Rule
		if strings.TrimSpace(rs.Rules) != "" {
			if err := json.Unmarshal([]byte(rs.Rules), &existing); err != nil {
				log.Warn("[SGHelper] parse rule_set rules failed, skipping", "rule_set_id", rsID, "err", err)
				failedCount++
				lastErr = err
				continue
			}
		}
		// 直接传完整 existing（含 is_required=true）给 UpdateRuleSetRulesInternal，
		// MergeRequiredRules 按 fingerprint 去重：已有 required 规则保持不变，
		// 新增/移除的 recommended 规则按 SiteConfig 条件自然增删。
		_, synced, driftErrs, err := UpdateRuleSetRulesInternal(ctx, rs.Name, existing, true /* autoFixRules: SiteConfig 变更重投影 */)
		if err != nil {
			log.Warn("[SGHelper] refresh ruleset failed, continuing with remaining",
				"rule_set_id", rsID, "err", err)
			failedCount++
			lastErr = err
			continue
		}
		if len(driftErrs) > 0 {
			log.Warn("[SGHelper] refresh ruleset partially drifted, continuing",
				"rule_set_id", rsID, "drifted_sgs", len(driftErrs))
			failedCount++
			lastErr = hcommon.I18nError(i18n.MsgSGRulesetRefreshSGDrifted, rsID, len(driftErrs))
			continue
		}
		log.Info("[SGHelper] refreshed ruleset required rules", "rule_set_id", rsID, "synced_sgs", synced)
	}

	if failedCount > 0 {
		return hcommon.I18nRichError(lastErr, i18n.MsgSGRulesetRefreshFailed, failedCount, len(rsIDs))
	}
	log.Info("[SGHelper] refreshed required rules in all rulesets", "ruleset_count", len(rsIDs))
	return nil
}

// ============================================================================
// 4. 工具函数
// ============================================================================

// HasSGPoolReady 检查当前租户是否存在至少一个 RuleSet 及至少一个 ACTIVE 状态的 SG。
// 用于前置拦截——在 SG 体系未就绪时（如全新部署 Bootstrap 未完成）避免误执行规则下发。
// 返回 (ready, ruleSetCount, activeSGCount, error)。
func HasSGPoolReady(ctx ...context.Context) (bool, int64, int64, error) {
	var _ctx context.Context
	if len(ctx) > 0 {
		_ctx = ctx[0]
	} else {
		_ctx = context.Background()
	}
	var rsCount, activeSGCount int64
	if err := model.DB(_ctx).Model(&model.RuleSet{}).Count(&rsCount).Error; err != nil {
		return false, 0, 0, hcommon.I18nRichError(err, i18n.MsgSGRulesetCountRuleSetsFailed)
	}
	if rsCount == 0 {
		return false, 0, 0, nil
	}
	if err := model.DB(_ctx).Model(&model.ManagedSGPool{}).
		Where("status = ?", model.SGStatusActive).Count(&activeSGCount).Error; err != nil {
		return false, rsCount, 0, hcommon.I18nRichError(err, i18n.MsgSGRulesetCountActiveSGFailed)
	}
	return activeSGCount > 0, rsCount, activeSGCount, nil
}

// isIngressAcceptRule 判断一条规则是否是"INGRESS ACCEPT"型的端口放通规则。
// 只有这类规则才能用 portCoveredByRules 的"覆盖"语义判重。
func isIngressAcceptRule(r Rule) bool {
	return normalizeDirection(r.Direction) == "INGRESS" &&
		strings.ToUpper(strings.TrimSpace(r.Action)) == "ACCEPT"
}

// singlePortFromRule 尝试从 Rule.Port 解析出单个端口号。
// 支持 "7540" / "7540-7540"；范围段（非单点）返回 ok=false，让 fallback 走指纹去重。
func singlePortFromRule(rulePort string) (int, bool) {
	p := strings.TrimSpace(rulePort)
	if p == "" || strings.EqualFold(p, "ALL") {
		return 0, false
	}
	if idx := strings.Index(p, "-"); idx > 0 {
		lo := strings.TrimSpace(p[:idx])
		hi := strings.TrimSpace(p[idx+1:])
		if lo != hi {
			return 0, false // 范围段不单化
		}
		p = lo
	}
	n, err := strconv.Atoi(p)
	if err != nil {
		return 0, false
	}
	return n, true
}

// 防止 import 漂移（漂移检查用，本文件引用 vpc 和 common 只在其他函数里）
var _ = common.StringPtr
var _ = vpc.NewDescribeSecurityGroupPoliciesRequest
