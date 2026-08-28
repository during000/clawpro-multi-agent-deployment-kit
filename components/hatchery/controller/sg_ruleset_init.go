package controller

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tcerr "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
	"gorm.io/gorm"
)

// SG RuleSet 初始化职责（sg-ruleset-projection 方案，启动时一次性执行）：
//   1. 如 RuleSet(default) 已存在 → 幂等直接返回（MySQL 多 Pod 场景用 GET_LOCK 兜 double-check）
//   2. 读 SiteConfig.SecurityGroupId（老 base）→ DescribeSecurityGroupPolicies → rules JSON
//      - 若老 base 为空（全新部署）→ rules="[]"
//   3. 合并必需规则（保证新实例一上来就有 SSH / ClawPro 通信等规则）
//   4. 云 API：创建新 SG + 应用 rules（transient 失败 retry 3 次）
//   5. DB 事务：
//      - INSERT rule_sets(default, rules, version=1)
//      - IF 老 base 非空：INSERT managed_sg_pool(老 SG, status=FROZEN, cvm_count=已绑定实例数)
//      - INSERT managed_sg_pool(新 SG, status=ACTIVE, cvm_count=0)
//   6. 写审计 sg_ruleset_init
//
// 失败时云端新建但 DB 事务失败 → tryDeleteCloudSG 回收；Guardian 反向扫描兜底孤儿。
//
// sgInitCompleted 供 /health 查询；初始化完成前返回 503，便于 K8s 探针辨识。

// CreateInitialRuleSetAndSG 通用的"建规则组 + 首个 ACTIVE SG"流程。
// InitSGRuleSet 和 HandleCreateRuleSet 共用这个底座。
//
// 参数：
//
//   - name: RuleSet 名称；空串默认为 "default"（初始化场景总是 default）
//
//   - userRules: 用户侧已有规则（初始化场景=老 base 读出来的规则；Create 场景=管理员在弹窗里填的规则或空）
//
//   - oldBaseID: 如果要顺便把一个老 SG 标 FROZEN，传它的 ID；否则空串
//
//   - oldBaseCVMCount: 老 SG 上绑的实例数（FROZEN 行的 cvm_count 初值）
//
//   - forceSkipMerge: 是否硬保护"严格保留 userRules 原貌"，无视 SiteConfig、无视 autoFixRules。
//     ⚠️ 仅启动期存量租户自愈分支 2（InitSGRuleSet 读到老 base）允许传 true，
//     其他所有调用点（HandleCreateRuleSet / ImportRulesFromSGInternal / 测试）MUST 传 false。
//     目的：避免升级时替客户在老 base 上偷偷放通他从未授权的 ClawPro 必需端口。
//
//   - autoFixRules: 当 forceSkipMerge=false 时生效，是否主动注入必需规则
//     合并决策：shouldMerge = siteConfigRequiresRecommendedRules(ctx) || autoFixRules
//
//   - 满足任一 → merge（注入 builtin: allow_internet/allow_ssh + 启用的 recommended）
//
//   - 都不满足 → 不 merge，userRules 原样
//
//     ⚠️ 本函数是全代码库唯一会注入 builtin 规则的入口。
//     其他所有写入路径（UpdateRuleSetRulesInternal / ensureRuleInRuleSet /
//     RefreshAllRuleSetsForRequiredRules 等）只会注入 recommended，永不注入 builtin。
//
// 返回：rsID / newSGID / err
// 失败已自动 tryDeleteCloudSG 清理云端新建的 SG。
func CreateInitialRuleSetAndSG(ctx context.Context, name, description string, userRules []Rule, oldBaseID string, oldBaseCVMCount int, forceSkipMerge bool, autoFixRules bool) (uint, string, error) {
	log := Logger(ctx)
	identifier := model.CurrentIdentifier(ctx)
	started := time.Now()

	if strings.TrimSpace(name) == "" {
		name = model.DefaultRuleSetName
	}

	// 合并决策：forceSkipMerge 优先级最高（启动期自愈硬保护）；否则按 SiteConfig + autoFixRules 3 态判
	var merged []Rule
	switch {
	case forceSkipMerge:
		merged = userRules
		log.Info("[ruleset-init] forceSkipMerge=true, preserving userRules as-is",
			"rule_count", len(userRules), "reason", "preserve legacy base SG rules")
	case siteConfigRequiresRecommendedRules(ctx) || autoFixRules || shouldApplyOfficeIngressRules(ctx):
		// 全代码库唯一注入 builtin 的位置：首次建组时合入 builtin + 启用的 recommended
		merged = MergeRequiredRules(userRules, LoadClawproRequiredRulesAll(ctx))
	default:
		merged = userRules
		log.Info("[ruleset-init] skip merging required rules",
			"reason", "siteconfig has no recommended requirement and autoFixRules=false",
			"rule_count", len(userRules))
	}
	mergedJSON, err := json.Marshal(merged)
	if err != nil {
		return 0, "", hcommon.I18nRichError(err, i18n.MsgSGRulesetInitMarshalRules)
	}

	// 云 API：建新 SG + 应用规则（PRD 4.2：clawpro-sg-{ident}-{name}-{NN}）
	//   - 首次建 RuleSet 阶段 rule_set 还没插 DB，ordinal 恒为 01
	//   - 上限 AutoScaleSG 走 NextSGOrdinalForRuleSet 重新算
	newSGName := buildManagedSGName(identifier, name, 1)
	newSGDesc := buildManagedSGDescription(identifier, name, 1)
	newSGID, err := createCloudSGWithRetry(ctx, newSGName, newSGDesc)
	if err != nil {
		return 0, "", hcommon.I18nRichError(err, i18n.MsgSGPoolCreateCloudSGFailed)
	}
	if err := applyRulesToCloudSGWithRetry(ctx, newSGID, string(mergedJSON)); err != nil {
		tryDeleteCloudSG(ctx, newSGID)
		return 0, "", hcommon.I18nRichError(err, i18n.MsgSGPoolApplyRulesFailed, newSGID)
	}
	log.Info("[ruleset-init] cloud SG created and rules applied",
		"new_sg_id", newSGID, "rule_count", len(merged), "old_base", oldBaseID, "name", name)

	// DB 事务：RuleSet + 最多 2 行 ManagedSGPool
	//   三条 Create 的 Identifier 都由 GORM 全局 callback（set_identifier）在 Create 前自动填充，此处不显式赋值。
	var rsID uint
	err = model.DB(ctx).Transaction(func(tx *gorm.DB) error {
		rs := model.RuleSet{
			Name:         name,
			Description:  description,
			Rules:        string(mergedJSON),
			Version:      1,
			UserGroupIDs: "[]", // 预留字段：本期恒空
			IsDefault:    true, // 预留字段：本期单一 RuleSet，恒为默认
		}
		if err := tx.Create(&rs).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgSGRulesetInitInsertRuleSet)
		}
		rsID = rs.ID

		if oldBaseID != "" {
			// 清理可能残留的旧记录（半清理状态恢复：rule_set 已删但 managed_sg_pool 未清）
			if err := tx.Where("sg_id = ?", oldBaseID).Delete(&model.ManagedSGPool{}).Error; err != nil {
				return hcommon.I18nRichError(err, i18n.MsgSGRulesetInitCleanupFrozen)
			}
			frozen := model.ManagedSGPool{
				SGID: oldBaseID,
				// 老 base SGName 留空，由 Guardian 下一轮从云 API 同步填入
				RuleSetID:   rs.ID,
				RuleVersion: 0, // 0 表示导入的遗留 SG，不计入 ordinal 编号
				Status:      model.SGStatusFrozen,
				CVMCount:    oldBaseCVMCount,
			}
			if err := tx.Create(&frozen).Error; err != nil {
				return hcommon.I18nRichError(err, i18n.MsgSGRulesetInitInsertFrozen)
			}
		}

		active := model.ManagedSGPool{
			SGID:        newSGID,
			SGName:      newSGName, // 刚新建的 SG，名字 = buildManagedSGName 的结果
			RuleSetID:   rs.ID,
			RuleVersion: 1,
			Status:      model.SGStatusActive,
			CVMCount:    0,
		}
		if err := tx.Create(&active).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgSGRulesetInitInsertActive)
		}
		return nil
	})
	if err != nil {
		tryDeleteCloudSG(ctx, newSGID)
		return 0, "", hcommon.I18nRichError(err, i18n.MsgSGRulesetInitDBTxFailed)
	}

	InvalidateDefaultRuleSetCache(identifier)
	log.Info("[ruleset-init] complete",
		"rule_set_id", rsID, "new_sg_id", newSGID, "old_base", oldBaseID, "name", name,
		"rule_count", len(merged), "elapsed_ms", time.Since(started).Milliseconds())
	return rsID, newSGID, nil
}

// DescribeSGPoliciesWithRetry 读云端 SG 规则，transient retry 3 次（复用 RetryCloudCall）。
func DescribeSGPoliciesWithRetry(ctx context.Context, sgID string) ([]Rule, error) {
	var rules []Rule
	err := RetryCloudCall(ctx, func() error {
		var e error
		rules, e = describeSGPoliciesOnce(ctx, sgID)
		return e
	})
	return rules, err
}

func describeSGPoliciesOnce(ctx context.Context, sgID string) ([]Rule, error) {
	client, err := newVpcClientForSGFn(ctx)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgCreateVPCClientFailed)
	}
	req := vpc.NewDescribeSecurityGroupPoliciesRequest()
	req.SecurityGroupId = common.StringPtr(sgID)
	resp, err := client.DescribeSecurityGroupPolicies(req)
	if err != nil {
		return nil, err
	}
	if resp.Response == nil || resp.Response.SecurityGroupPolicySet == nil {
		return nil, nil
	}
	return policySetToRules(resp.Response.SecurityGroupPolicySet), nil
}

// createCloudSGWithRetry 建 SG 带 transient retry（复用 RetryCloudCall）。
func createCloudSGWithRetry(ctx context.Context, name, desc string) (string, error) {
	var sgID string
	err := RetryCloudCall(ctx, func() error {
		var e error
		sgID, e = createCloudSG(ctx, name, desc)
		return e
	})
	return sgID, err
}

// applyRulesToCloudSGWithRetry 应用规则带 transient retry（复用 RetryCloudCall）。
func applyRulesToCloudSGWithRetry(ctx context.Context, sgID, rulesJSON string) error {
	return RetryCloudCall(ctx, func() error {
		return applyRulesToCloudSG(ctx, sgID, rulesJSON)
	})
}

// IsSGGoneError 判断云 API 错误是否表示安全组已在云端不存在。
// 腾讯云 DescribeSecurityGroupPolicies 对不存在的 SG 可能返回：
//   - ResourceNotFound
//   - InvalidParameterValue（"安全组不存在"）
//   - InvalidSecurityGroupId.NotFound
func IsSGGoneError(err error) bool {
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
	// 兜底：错误信息里包含"不存在"或"not found"
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "not found") || strings.Contains(msg, "不存在") {
		return true
	}
	return false
}

// 明确使用 slog 防 import 漂移
var _ = slog.Default
