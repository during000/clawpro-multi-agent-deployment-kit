package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tcerr "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
	"gorm.io/gorm"
)

// sg-ruleset-projection 的规则组（RuleSet）管理层。
//
// Rule 是规则的标准内存表示（snake_case JSON tag，与前端对齐）；
// Rules 序列化成 RuleSet.Rules（type:text） 字段存入 DB。

// Rule 一条安全组规则。指纹由 (Direction, Protocol, Port, CidrBlock, Action) 五元组决定。
//
// ⚠️ Direction 必须含在指纹里：腾讯云 SG 的入站（Ingress）和出站（Egress）是两个独立规则列表，
// 同 proto/port/cidr/action 的 Ingress 和 Egress 是两条不同规则。
// 指纹漏 Direction 会导致合并时误去重 → 丢一个方向的规则 → 整包覆盖云端 SG 后缺策略。
type Rule struct {
	Direction         string `json:"direction"`                    // INGRESS / EGRESS
	Protocol          string `json:"protocol"`                     // TCP / UDP / ICMP / ALL
	Port              string `json:"port"`                         // "22" / "80-443" / "ALL"
	CidrBlock         string `json:"cidr_block"`                   // IPv4 / IPv6 CIDR
	Action            string `json:"action"`                       // ACCEPT / DROP
	PolicyDescription string `json:"policy_description,omitempty"` // 备注（不纳入指纹）
	IsRequired        bool   `json:"is_required,omitempty"`        // 是否为 ClawPro 必需规则（前端只读展示用）
	// Prepend 为 true 时，合并规则放到所有用户规则之前。
	Prepend bool `json:"prepend,omitempty"`
}

// Fingerprint 返回规则的规范化指纹（五元组字符串，| 分隔）。
// 规范化规则：
//   - Direction / Protocol / Action 一律大写
//   - Port 里 "22" 和 "22-22" 视为同一；空值和 "ALL" 视为同一（统一为 "ALL"）
//   - CidrBlock "10.0.0.1" 补齐 "/32"；"::1" 补齐 "/128"；IPv6 大小写归一
func (r Rule) Fingerprint() string {
	return strings.Join([]string{
		normalizeDirection(r.Direction),
		strings.ToUpper(strings.TrimSpace(r.Protocol)),
		normalizePortRange(r.Port),
		normalizeCIDR(r.CidrBlock),
		strings.ToUpper(strings.TrimSpace(r.Action)),
	}, "|")
}

func normalizeDirection(d string) string {
	u := strings.ToUpper(strings.TrimSpace(d))
	switch u {
	case "INGRESS", "IN":
		return "INGRESS"
	case "EGRESS", "OUT":
		return "EGRESS"
	}
	return u // 未知值原样返回（会和其他同样未知的规则产生唯一指纹）
}

func normalizePortRange(p string) string {
	p = strings.TrimSpace(p)
	if p == "" || strings.EqualFold(p, "ALL") {
		return "ALL"
	}
	// 去掉重复的 "22-22" → "22"
	if i := strings.Index(p, "-"); i > 0 {
		lo := strings.TrimSpace(p[:i])
		hi := strings.TrimSpace(p[i+1:])
		if lo == hi {
			return lo
		}
	}
	return p
}

func normalizeCIDR(c string) string {
	c = strings.TrimSpace(c)
	if c == "" {
		return ""
	}
	// sg-xxx / ipm-xxx / ipmg-xxx 是腾讯云资源标识而非 CIDR，原样返回（不走 IP 解析）。
	// 本分支必须在 ParseIP/ParseCIDR 之前，避免 net.ParseCIDR 把 "sg-xxx" 当作带 "/" 的 CIDR 误解析
	// （虽然当前不含 "/"，但防御性检查保证行为明确）。
	if kind := classifyRuleSource(c); kind == srcSG || kind == srcAddressTpl || kind == srcAddressGroup {
		return c
	}
	// 已含 / 则 ParseCIDR 规范化 IPv6 大小写；不含 / 补默认前缀
	if !strings.Contains(c, "/") {
		ip := net.ParseIP(c)
		if ip == nil {
			return c
		}
		if ip.To4() != nil {
			return ip.String() + "/32"
		}
		return ip.String() + "/128"
	}
	_, ipnet, err := net.ParseCIDR(c)
	if err != nil {
		return c
	}
	return ipnet.String()
}

// sourceKind 描述 Rule.CidrBlock 字段承载的「来源/目的」类型。
//
// 历史背景：最初 Rule.CidrBlock 只存 CIDR；为支持腾讯云 SG 的 SecurityGroupId / AddressTemplate
// 引用类型规则，又要避免扩 schema 破坏 DB 存量 + 前端 UI，我们把 CidrBlock 字段语义扩展为
// "规则来源标识"，按前缀区分五类。所有正向（ruleToPolicy）/反向（policyToRule）/
// 归一化（normalizeCIDR）路径共用 classifyRuleSource 做一致判定。
type sourceKind int

const (
	srcUnknown      sourceKind = iota // 空串或未识别（非法输入，调用方自行处理）
	srcIPv4CIDR                       // 1.2.3.4/32、10.0.0.0/8 等含 "." 的 CIDR（或裸 IPv4）
	srcIPv6CIDR                       // ::/0、2001:db8::/64 等含 ":" 的 CIDR（或裸 IPv6）
	srcSG                             // sg-xxxxxxxx（安全组作为来源/目的）
	srcAddressTpl                     // ipm-xxxxxxxx（IP 地址模板）
	srcAddressGroup                   // ipmg-xxxxxxxx（IP 地址模板组）
)

// classifyRuleSource 按前缀把 Rule.CidrBlock 字符串分类到五种来源之一。
//
// ⚠️ 前缀判定顺序敏感：`ipmg-` 是 `ipm-` 的前缀超集，必须先判 `ipmg-` 再判 `ipm-`，
// 否则 "ipmg-abc" 会被错误识别为 srcAddressTpl。
//
// 本函数只做语法分类，不做 ID 长度/字符集严格校验（那是 isValidRuleSource 的职责）。
// 云端下发的合法性由腾讯云 API 兜底（架构决定，见 UpdateRuleSetRulesInternal 注释）。
func classifyRuleSource(s string) sourceKind {
	s = strings.TrimSpace(s)
	if s == "" {
		return srcUnknown
	}
	switch {
	case strings.HasPrefix(s, "sg-"):
		return srcSG
	case strings.HasPrefix(s, "ipmg-"): // 必须在 "ipm-" 之前判断
		return srcAddressGroup
	case strings.HasPrefix(s, "ipm-"):
		return srcAddressTpl
	case strings.Contains(s, ":"):
		return srcIPv6CIDR
	default:
		return srcIPv4CIDR
	}
}

// isValidRuleSource 判断一个字符串是否是合法的 Rule 来源标识（五种之一）。
//
// 仅用于单测与内部断言；不在写入路径（UpdateRuleSetRulesInternal / ImportRulesFromSGInternal）
// 强制调用——仓库既定架构是"规则合法性由腾讯云 API 兜底校验，本地不再做格式预检"
// （见 ruleset_helpers.go 原始注释）。前端脏数据会在 fan-out 阶段被云端 API
// 自然拦截，回滚机制保证 DB 不被污染。
func isValidRuleSource(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	switch classifyRuleSource(s) {
	case srcSG:
		return len(s) > len("sg-")
	case srcAddressGroup:
		return len(s) > len("ipmg-")
	case srcAddressTpl:
		return len(s) > len("ipm-")
	case srcIPv4CIDR, srcIPv6CIDR:
		// 允许 "1.2.3.4/32"、"::/0" 等标准 CIDR，也允许裸 IP（normalizeCIDR 会补前缀）
		if _, _, err := net.ParseCIDR(s); err == nil {
			return true
		}
		if ip := net.ParseIP(s); ip != nil {
			return true
		}
		return false
	default:
		return false
	}
}

// MergeRequiredRules 合并用户规则和必需规则；按指纹去重，冲突时必需规则优先（含 Description 覆盖）。
//
//   - requiredRules 中 Prepend=true 的规则按 requiredRules 相对顺序插到所有用户规则之前
//   - 其后用户规则按 userRules 入参顺序保留（同一 fingerprint 重复时仅保留首次出现的位置）
//   - 当某条用户规则与非 Prepend 必需规则同 fingerprint：原位置不变，使用必需规则副本覆盖描述并标 IsRequired
//   - 用户规则中未涉及的非 Prepend 必需规则按 requiredRules 相对顺序追加在末尾
//
// 顺序保留是 reorder 接口"调整完顺序后续操作不被打乱"的前提；不再做指纹字典序排序。
// 由此带来的对腾讯云"自上而下匹配"语义的真正落地，由 applyRulesToCloudSG 中的 SortPolicys=true 保证。
func MergeRequiredRules(userRules, requiredRules []Rule) []Rule {
	var prepend, rest []Rule
	requiredByFp := make(map[string]Rule, len(requiredRules))
	for _, r := range requiredRules {
		r.IsRequired = true
		requiredByFp[r.Fingerprint()] = r
		if r.Prepend {
			prepend = append(prepend, r)
		} else {
			rest = append(rest, r)
		}
	}

	out := make([]Rule, 0, len(userRules)+len(requiredRules))
	seen := make(map[string]struct{}, len(userRules)+len(requiredRules))
	for _, r := range prepend {
		fp := r.Fingerprint()
		if _, dup := seen[fp]; dup {
			continue
		}
		seen[fp] = struct{}{}
		r.IsRequired = true
		out = append(out, r)
	}

	// 1) 先按用户规则原序输出；命中必需规则的位置改用必需规则副本（含 Description 覆盖）
	for _, r := range userRules {
		fp := r.Fingerprint()
		if _, dup := seen[fp]; dup {
			continue // 用户规则内部去重（保留首次出现位置）
		}
		seen[fp] = struct{}{}
		if req, ok := requiredByFp[fp]; ok {
			out = append(out, req)
		} else {
			r.IsRequired = false
			out = append(out, r)
		}
	}

	// 2) 用户规则未出现的必需规则按原顺序追加在末尾
	for _, r := range rest {
		fp := r.Fingerprint()
		if _, dup := seen[fp]; dup {
			continue
		}
		seen[fp] = struct{}{}
		r.IsRequired = true
		out = append(out, r)
	}
	return out
}

// LoadClawproRequiredRules 从 config/clawpro_required_sg_rules.json 加载必需规则并归一化为 []Rule。
//
// ⚠️ 注入语义（本代码库的铁律）：
//   - 仅「首次新建 RuleSet」入口（CreateInitialRuleSetAndSG，即 HandleCreateRuleSet / 导入兜底建组）
//     调用 [LoadClawproRequiredRulesAll]，注入 builtin（allow_internet/allow_ssh）+ 启用的 recommended
//   - 其他所有入口（保存规则 / 系统反推 / SiteConfig 开关切换 / 导入已存在 RuleSet）
//     调用 [LoadClawproRequiredRulesRecommendedOnly]，永远不注入 builtin，仅按开关合入 recommended
//
// 本函数保留为 [LoadClawproRequiredRulesAll] 的别名，保持向后兼容；新代码请直接使用语义明确的变体。
func LoadClawproRequiredRules(ctx context.Context) []Rule {
	return LoadClawproRequiredRulesAll(ctx)
}

// LoadClawproRequiredRulesAll 返回全部必需规则：builtin + 启用的 recommended。
// 仅「首次新建 RuleSet」入口使用。
//
// ⚠️ 必须先处理动态占位符：
//   - resolveConditionalRules：按 SiteConfig 的 gateway_ui_enable / browser_vnc_enable 过滤规则组，
//     并把 `{{GATEWAY_UI_PORT}}` 替换为真实端口。未开启功能的 recommended 规则组会被整组丢弃。
//   - replaceVpcCidrPlaceholder：把 `{{VPC_CIDR}}` 替换为 VPC CIDR（走云 API 解析 SiteConfig.VpcId）。
//
// 直接传 `{{...}}` 给腾讯云 API 会 InvalidParameterValue 报错。
func LoadClawproRequiredRulesAll(ctx context.Context) []Rule {
	return loadRequiredRulesFiltered(ctx, func(cat sgRuleCategory) bool {
		// 所有分类都纳入（builtin + recommended）
		_ = cat
		return true
	})
}

// LoadClawproRequiredRulesRecommendedOnly 仅返回启用的 recommended 规则，永不包含 builtin。
// 用于保存规则 / 系统反推 / SiteConfig 开关切换 / 导入已存在 RuleSet 等入口——
// 这些路径禁止向用户 SG 注入 allow_internet / allow_ssh 等无条件 builtin 规则。
func LoadClawproRequiredRulesRecommendedOnly(ctx context.Context) []Rule {
	return loadRequiredRulesFiltered(ctx, func(cat sgRuleCategory) bool {
		return cat.Type == "recommended"
	})
}

// loadRequiredRulesFiltered 是 LoadClawproRequiredRules* 的共享实现。
// keepCategory 决定某个分类是否纳入结果；其他逻辑（占位符替换、IPv4/IPv6 展开、保险丝）对所有变体一致。
func loadRequiredRulesFiltered(ctx context.Context, keepCategory func(sgRuleCategory) bool) []Rule {
	cfg := clawproRequiredRuleSet()
	// 1. 评估条件 / 替换端口占位符
	resolveConditionalRules(ctx, &cfg)
	// 2. 替换 VPC CIDR 占位符（vpcCidr 为空时保持原样；调用方无 VPC 也能继续，只会保留占位符——这种情况后面 flatten 时我们会跳过带占位符的 cidr）
	replaceVpcCidrPlaceholder(&cfg, resolveVpcCidr(ctx))

	var rules []Rule
	for _, cat := range cfg.Categories {
		if !keepCategory(cat) {
			continue
		}
		for _, grp := range cat.RuleGroups {
			for _, rr := range grp.Rules {
				// 保险丝：若仍有未解析的 `{{...}}` 占位符（例如没配 VPC 导致 VPC_CIDR 留白），直接丢弃该条规则
				// 而不是让云 API 拒绝请求。
				if hasPlaceholder(rr.CidrBlock) || hasPlaceholder(rr.Ipv6Cidr) || hasPlaceholder(rr.Port) || hasPlaceholder(rr.Protocol) {
					continue
				}
				if rr.CidrBlock != "" {
					rules = append(rules, requiredRuleToRule(rr, rr.CidrBlock))
				}
				if rr.Ipv6Cidr != "" {
					rules = append(rules, requiredRuleToRule(rr, rr.Ipv6Cidr))
				}
			}
		}
	}
	rules = append(rules, loadOfficeIngressRules(ctx)...)
	return rules
}

// hasPlaceholder 粗略检测 `{{...}}` 未替换占位符。
func hasPlaceholder(s string) bool {
	return strings.Contains(s, "{{") && strings.Contains(s, "}}")
}

// requiredRuleToRule 把历史 requiredSGRule 结构转为新 Rule。
// cidr 由调用方传入（IPv4 或 IPv6 分别展开）。
func requiredRuleToRule(rr requiredSGRule, cidr string) Rule {
	return Rule{
		Direction:         rr.Direction,
		Protocol:          rr.Protocol,
		Port:              rr.Port,
		CidrBlock:         cidr,
		Action:            rr.Action,
		PolicyDescription: rr.Description,
		IsRequired:        true,
	}
}

// --- RuleSet 的 fan-out / 更新逻辑 ---

// fanoutConcurrency 规则投影时的并发度上限。10 低于腾讯云 VPC 20 QPS 限制，留余量。
const fanoutConcurrency = 10

// UpdateRuleSetRulesInternal 整包提交规则变更（两阶段提交语义）：
//
//  1. 进程锁 sg-ruleset-update-<identifier> 串行化多管理员并发保存
//  2. 读指定 (identifier, name) 的 RuleSet 作"回滚备份"（rollbackJSON）
//  3. 合并必需规则（规则合法性由腾讯云 API 兜底校验，本地不再做格式预检）
//  4. 列出该 RuleSet 下所有 ACTIVE SG → 并发 fan-out "新" 规则
//  5. 任一 SG 失败：
//     - best-effort 用 rollbackJSON 覆盖已成功的 SG（回滚云端）
//     - DB 完全不改，version 不 ++
//     - 返回 driftErrs（失败原因清单）+ 错误
//     - 回滚也失败的 SG 才 MarkSGDrift（Guardian 兜底）
//  6. 全部成功：
//     - 事务 UPDATE RuleSet.rules + version++
//     - 批量 UpdateSGRuleVersion
//
// name 为空时 fallback 到 DefaultRuleSetName（"default"），向后兼容老客户端。
//
// 返回：新 version（失败时为旧 version）/ 成功同步 SG 数 / 失败 SG 详情 / 错误。
// 调用方判定成功：err == nil && len(driftErrs) == 0。
//
// autoFixRules 参数：
//   - 本函数所有路径（保存规则 / 系统反推 / SiteConfig 开关切换 / 导入已存在 RuleSet）
//     永远 *不* 注入 builtin 规则（allow_internet 全出站 / allow_ssh 22）。builtin 仅在
//     "首次新建 RuleSet"（CreateInitialRuleSetAndSG）入口合入。
//   - autoFixRules=true 或 SiteConfig 启用任一 recommended（gateway/VNC）时，
//     注入启用的 recommended 规则；两者都不满足时，userRules 原样落盘。
//   - 历史入参 auto_fix_rules 来自 HandleImportRulesFromSG / HandleUpdateRuleSetRules 请求字段；
//     本次改造后它仅影响 recommended，不再与 builtin 挂钩。
func UpdateRuleSetRulesInternal(ctx context.Context, name string, newRules []Rule, autoFixRules bool) (version, synced int, driftErrs []DriftError, rerr error) {
	log := Logger(ctx)

	// 1. 进程锁：避免两个管理员并发保存导致"部分成功部分失败"的窗口
	//    注意锁 key 不含 name：未来多 RuleSet 场景下也要求整租户串行 fan-out，
	//    避免两个 RuleSet 同时改、某个 SG 被两边都覆盖的顺序问题。
	//    锁 key 无需拼 identifier：model.AcquireLock 内部的 lockName() 会自动加
	//    "{identifier}:{resource}" 前缀做多租户隔离。
	lock, lockErr := model.AcquireLock(ctx, "sg-ruleset-update", 30*time.Second)
	if lockErr != nil {
		return 0, 0, nil, hcommon.I18nRichError(lockErr, i18n.MsgUpdateRuleSetConflict).WithPrefix("acquire update lock")
	}
	defer lock.Release()

	// 2. 读当前 RuleSet（作回滚备份 + 拿 ID + 旧 version）
	//    name 为空时走 is_default=true 查找，兼容自定义名称的默认规则组。
	var oldRS *model.RuleSet
	var err error
	if strings.TrimSpace(name) == "" {
		oldRS, err = model.GetDefaultRuleSet(ctx)
	} else {
		oldRS, err = model.GetRuleSetByName(ctx, name)
	}
	if err != nil {
		return 0, 0, nil, hcommon.I18nRichError(err, i18n.MsgFailedToReadRuleSet, name)
	}
	// 用实际 DB 中的 name 覆盖入参，确保后续 LockRuleSetForUpdate 能命中正确行
	name = oldRS.Name
	rollbackJSON := oldRS.Rules // 旧规则的 JSON（用于失败回滚）

	// 3. 合并必需规则（规则合法性由腾讯云 API 兜底校验，本地不做格式预检）
	//    本路径永远 *不* 注入 builtin（allow_internet/allow_ssh）——builtin 仅首次新建 RuleSet 注入。
	//    这里只按「SiteConfig 开关 or autoFixRules」决定是否合入 recommended（gateway/VNC）。
	//    决策：shouldMerge = SiteConfig 启用任何 recommended OR autoFixRules
	//      - 满足任一 → 注入启用的 recommended 规则
	//      - 都不满足 → userRules 原样落盘
	//    空规则集合（merged=[]）会进入 fan-out 走 applyRulesToCloudSG → clearAllRulesForSG 真清空路径
	var merged []Rule
	shouldMerge := siteConfigRequiresRecommendedRules(ctx) || autoFixRules || shouldApplyOfficeIngressRules(ctx)
	if shouldMerge {
		required := LoadClawproRequiredRulesRecommendedOnly(ctx)
		merged = MergeRequiredRules(newRules, required)
	} else {
		merged = newRules
		log.Info("[ruleset] skip merging recommended rules",
			"reason", "siteconfig has no recommended requirement and autoFixRules=false",
			"rule_count", len(newRules))
	}

	newRulesJSON, err := json.Marshal(merged)
	if err != nil {
		return oldRS.Version, 0, nil, hcommon.I18nRichError(err, i18n.MsgMarshalMergedRulesFailed)
	}

	// 4. 列出所有 ACTIVE SG
	activeSGs, err := model.ListActiveSGsForFanout(ctx, oldRS.ID)
	if err != nil {
		return oldRS.Version, 0, nil, hcommon.I18nRichError(err, i18n.MsgListActiveSGFailed)
	}

	// 🔎 排障日志：fan-out 之前打出本次将下发的规则总览。
	// 出问题时配合"fan-out single SG failed (full cloud error)"日志可快速定位是哪条规则被拒。
	log.Info("[ruleset] about to fan-out rules to active SGs",
		"rule_set_id", oldRS.ID,
		"rule_set_name", name,
		"old_version", oldRS.Version,
		"user_rules_in", len(newRules),
		"merged_rules_out", len(merged),
		"active_sg_count", len(activeSGs),
		"should_merge", shouldMerge,
		"merged_preview", previewRulesJSON(string(newRulesJSON), 1500))

	// 5. 并发 fan-out 新规则（失败不 MarkDrift，留给下面回滚阶段决定）
	successIDs, failErrs := tryFanoutRulesToSGs(ctx, activeSGs, string(newRulesJSON))

	// 6. 任一失败 → 回滚已成功的 SG + 错误返回（DB 不变）
	if len(failErrs) > 0 {
		log.Warn("[ruleset] fan-out failed, rolling back successful SGs",
			"succeeded_before_fail", len(successIDs), "failed", len(failErrs))

		if len(successIDs) > 0 {
			rbFails := rollbackSGsToOldRules(ctx, successIDs, rollbackJSON)
			if len(rbFails) > 0 {
				// 回滚失败的 SG：云端现在是新规则但 DB 还是旧规则 → drift
				// MarkSGDrift + Guardian 每小时会继续重试
				log.Error("[ruleset] rollback failed for some SGs, marking drift",
					"rollback_failed_count", len(rbFails))
				for _, rf := range rbFails {
					if merr := model.MarkSGDrift(ctx, rf.SGID); merr != nil {
						log.Error("[ruleset] mark drift failed", "sg_id", rf.SGID, "err", merr)
					}
				}
				// 把回滚失败的 SG 追加到用户可见错误里
				for _, rf := range rbFails {
					failErrs = append(failErrs, DriftError{
						SGID:  rf.SGID,
						Error: "[ROLLBACK FAILED] " + rf.Error + "（云端保留新规则，DB 规则未变；Guardian 将在下小时尝试修复）",
					})
				}
			}
		}

		return oldRS.Version, 0, failErrs,
			hcommon.I18nError(i18n.MsgRuleSetDistributeFailed, len(failErrs))
	}

	// 7. 全部成功 → 事务写 DB
	var newVersion int
	err = model.DB(ctx).Transaction(func(tx *gorm.DB) error {
		locked, err := model.LockRuleSetForUpdate(tx, name)
		if err != nil {
			return err
		}
		if err := model.IncrementRuleSetVersion(tx, locked.ID, string(newRulesJSON)); err != nil {
			return err
		}
		var fresh model.RuleSet
		if err := tx.First(&fresh, locked.ID).Error; err != nil {
			return err
		}
		newVersion = fresh.Version
		return nil
	})
	if err != nil {
		// DB 写失败的极端情况：云端已全是新规则，但 DB 还是旧 version
		// 这是非常罕见的（DB 本地写 vs 云 API 全部成功），standalone MarkDrift 所有 SG
		log.Error("[ruleset] cloud succeeded but DB write failed, marking all SGs drift", "err", err)
		for _, sgID := range successIDs {
			_ = model.MarkSGDrift(ctx, sgID)
		}
		return oldRS.Version, 0, nil, hcommon.I18nRichError(err, i18n.MsgUpdateRuleSetFailed)
	}

	// 8. 批量更新 SG 的 rule_version
	for _, sgID := range successIDs {
		if err := model.UpdateSGRuleVersion(ctx, sgID, newVersion); err != nil {
			log.Warn("[ruleset] update sg rule_version failed (cloud + DB ok)", "sg_id", sgID, "err", err)
		}
	}

	log.Info("[ruleset] fan-out complete (2pc)",
		"rule_set_id", oldRS.ID, "new_version", newVersion, "synced", len(successIDs))
	return newVersion, len(successIDs), nil, nil
}

// ImportRulesFromSGInternal 从云账号下指定 SG 读规则 → 覆盖指定 (identifier, name) 的 RuleSet → fan-out。
// 校验：sourceSGID 不能在 managed_sg_pool 里（防止从自建 SG 导入造成循环）。
// name 为空时按 is_default=true 查找默认规则组。
// 若规则组尚不存在，自动走 CreateInitialRuleSetAndSG 创建（不存在则创建语义）。
//
// autoFixRules：是否注入 ClawPro 必需规则（仅影响 recommended）
//   - false（默认，纯导入）：以源 SG 规则原样落盘，不注入 recommended；builtin 也不注入
//   - true：合入启用的 recommended 规则（gateway/VNC），builtin 仍然不注入
//
// ⚠️ builtin 规则（allow_internet/allow_ssh）仅在「新建 RuleSet 分支」（即规则组
//
//	不存在、走 CreateInitialRuleSetAndSG 兜底建组）时才会注入；已存在 RuleSet
//	走 UpdateRuleSetRulesInternal 时永远不会合入 builtin。
func ImportRulesFromSGInternal(ctx context.Context, name, sourceSGID string, autoFixRules bool) (version, synced int, driftErrs []DriftError, rerr error) {
	log := Logger(ctx)

	// 1. 校验不是 clawpro 自建
	isManaged, err := model.IsManagedSG(ctx, sourceSGID)
	if err != nil {
		return 0, 0, nil, hcommon.I18nRichError(err, i18n.MsgImportCheckManagedSGFail)
	}
	if isManaged {
		return 0, 0, nil, hcommon.I18nError(i18n.MsgImportCannotImportFromManagedSG)
	}

	// 2. 读云端 SG 规则
	client, err := newVpcClientForSGFn(ctx)
	if err != nil {
		return 0, 0, nil, hcommon.I18nRichError(err, i18n.MsgImportCreateVpcClientFail)
	}
	descReq := vpc.NewDescribeSecurityGroupPoliciesRequest()
	descReq.SecurityGroupId = common.StringPtr(sourceSGID)
	descResp, err := client.DescribeSecurityGroupPolicies(descReq)
	if err != nil {
		return 0, 0, nil, hcommon.I18nRichError(err, i18n.MsgImportDescribeSGPoliciesFail)
	}
	if descResp.Response == nil || descResp.Response.SecurityGroupPolicySet == nil {
		return 0, 0, nil, hcommon.I18nError(i18n.MsgImportSourceSGRulesEmpty)
	}

	// 3. 云端 policy → []Rule
	userRules := policySetToRules(descResp.Response.SecurityGroupPolicySet)
	log.Info("[ruleset] importing rules from source sg",
		"source_sg_id", sourceSGID, "rule_count", len(userRules))

	// 4. 判断规则组是否已存在；不存在则自动创建
	var rsExists bool
	if strings.TrimSpace(name) == "" {
		_, rsErr := model.GetDefaultRuleSet(ctx)
		rsExists = rsErr == nil
	} else {
		_, rsErr := model.GetRuleSetByName(ctx, name)
		rsExists = rsErr == nil
	}

	if !rsExists {
		// 规则组不存在：走创建流程，把 autoFixRules 透传给 CreateInitialRuleSetAndSG
		// 内部按 3 态决策：
		//   - SiteConfig 启用 recommended → 必 merge（兜底，无视开关）
		//   - SiteConfig 全关 + autoFixRules=true → merge（仅 builtin）
		//   - SiteConfig 全关 + autoFixRules=false → 不 merge（纯导入，userRules 原样）
		rsName := strings.TrimSpace(name)
		if rsName == "" {
			rsName = model.DefaultRuleSetName
		}
		log.Info("[ruleset] import: rule_set not found, creating via import",
			"name", rsName, "source_sg_id", sourceSGID, "auto_fix_rules", autoFixRules)
		rsID, newSGID, createErr := CreateInitialRuleSetAndSG(ctx, rsName, "Agent 默认安全组", userRules, "", 0, false /* forceSkipMerge */, autoFixRules)
		if createErr != nil {
			return 0, 0, nil, hcommon.I18nRichError(createErr, i18n.MsgImportAutoCreateRuleSetFailed)
		}
		log.Info("[ruleset] import: auto-created rule_set",
			"rule_set_id", rsID, "new_sg_id", newSGID)
		// CreateInitialRuleSetAndSG 内部 version=1，synced=1（刚建的 SG 就是最新的）
		return 1, 1, nil, nil
	}

	// 5. 规则组已存在：复用 UpdateRuleSetRulesInternal（按 autoFixRules 决定是否合并必需规则）
	return UpdateRuleSetRulesInternal(ctx, name, userRules, autoFixRules)
}

// policySetToRules 云端 vpc.SecurityGroupPolicySet → []Rule。
//
// ⚠️ 来源类型支持（本 change 升级）：腾讯云 SG 规则的"来源/目标"有 4 种填法
// （CidrBlock / Ipv6CidrBlock / SecurityGroupId / AddressTemplate）。升级后：
//   - CidrBlock / Ipv6CidrBlock：原样保留到 Rule.CidrBlock（IPv6 通过 normalizeCIDR 归一化）
//   - SecurityGroupId（sg-xxx）：写入 Rule.CidrBlock
//   - AddressTemplate.AddressId（ipm-xxx）/ AddressGroupId（ipmg-xxx）：写入 Rule.CidrBlock
//
// 仅当一条 SDK policy 的所有来源字段皆空（这在腾讯云是非法状态，理论不会出现）才会 skip。
// Skip 场景保留 slog.Warn 作为异常观测。
func policySetToRules(set *vpc.SecurityGroupPolicySet) []Rule {
	var out []Rule
	for _, p := range set.Ingress {
		if r, ok := policyToRuleSkippable(p, "INGRESS"); ok {
			out = append(out, r)
		}
	}
	for _, p := range set.Egress {
		if r, ok := policyToRuleSkippable(p, "EGRESS"); ok {
			out = append(out, r)
		}
	}
	return out
}

// policyToRuleSkippable 在 policyToRule 基础上加防御性跳过判断：当所有来源字段皆空
// （CidrBlock / Ipv6CidrBlock / SecurityGroupId / AddressTemplate 都为 nil 或空串）
// 时返回 ok=false。这种 policy 在腾讯云 API 层面是非法输入，理论不会出现，作为兜底。
func policyToRuleSkippable(p *vpc.SecurityGroupPolicy, direction string) (Rule, bool) {
	r := policyToRule(p, direction)
	if r.CidrBlock != "" {
		return r, true
	}
	// 所有来源字段皆空（异常场景），记 warn 便于定位问题数据
	slog.Warn("import-sg: skip rule with empty source",
		"direction", direction,
		"protocol", r.Protocol,
		"port", r.Port,
		"action", r.Action,
		"description", r.PolicyDescription,
	)
	return Rule{}, false
}

// policyToRule 云端 vpc.SecurityGroupPolicy → Rule。
//
// 来源字段优先级链（D4，本 change 升级）：
//  1. SecurityGroupId          （sg-xxx）
//  2. AddressTemplate.AddressGroupId   （ipmg-xxx，必须在 AddressId 之前判断）
//  3. AddressTemplate.AddressId        （ipm-xxx）
//  4. Ipv6CidrBlock            （IPv6 CIDR）
//  5. CidrBlock                （IPv4 CIDR，最低优先级）
//
// 四种来源在腾讯云 API 层面互斥（一条 policy 只会填一个），多字段同时非空属异常输入，
// 按优先级取第一个非空字段作为安全降级。所有类型统一写入 Rule.CidrBlock，供 Fingerprint 和前端透传使用。
func policyToRule(p *vpc.SecurityGroupPolicy, direction string) Rule {
	r := Rule{Direction: direction}
	if p.Protocol != nil {
		r.Protocol = *p.Protocol
	}
	if p.Port != nil {
		r.Port = *p.Port
	}
	// 按 D4 优先级链识别来源字段
	switch {
	case p.SecurityGroupId != nil && *p.SecurityGroupId != "":
		r.CidrBlock = *p.SecurityGroupId
	case p.AddressTemplate != nil && p.AddressTemplate.AddressGroupId != nil && *p.AddressTemplate.AddressGroupId != "":
		r.CidrBlock = *p.AddressTemplate.AddressGroupId
	case p.AddressTemplate != nil && p.AddressTemplate.AddressId != nil && *p.AddressTemplate.AddressId != "":
		r.CidrBlock = *p.AddressTemplate.AddressId
	case p.Ipv6CidrBlock != nil && *p.Ipv6CidrBlock != "":
		r.CidrBlock = *p.Ipv6CidrBlock
	case p.CidrBlock != nil && *p.CidrBlock != "":
		r.CidrBlock = *p.CidrBlock
	}
	if p.Action != nil {
		r.Action = *p.Action
	}
	if p.PolicyDescription != nil {
		r.PolicyDescription = *p.PolicyDescription
	}
	return r
}

// DriftError 单个 SG fan-out 失败的详情，透传到前端供管理员定位根因。
type DriftError struct {
	SGID  string `json:"sg_id"`
	Error string `json:"error"`
}

// tryFanoutRulesToSGs 并发把 newRulesJSON 下发到给定 SG 列表。
//
// 不同于老 fanoutRulesToSGs（已废除）：
//   - 失败的 SG 不 MarkSGDrift（由调用方决定是否标 drift，因为上层是"全成功才落库"语义）
//   - 返回 successIDs（成功的 SG ID 列表，用于失败时的回滚）+ failErrs
//   - 不 UpdateSGRuleVersion（同样留给调用方在全成功后批量更新）
//
// 并发度 fanoutConcurrency=10；单 SG 内部调 applyRulesToSGWithRetry（transient retry 3 次）。
func tryFanoutRulesToSGs(ctx context.Context, sgs []model.ManagedSGPool, newRulesJSON string) (successIDs []string, failErrs []DriftError) {
	var mu sync.Mutex
	successIDs = make([]string, 0, len(sgs))
	failErrs = make([]DriftError, 0)
	sem := make(chan struct{}, fanoutConcurrency)
	var wg sync.WaitGroup

	for i := range sgs {
		sg := &sgs[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := applyRulesToSGWithRetry(ctx, sg.SGID, newRulesJSON); err != nil {
				// 🔎 排障日志：把云端原始报错完整打出来（不走 truncateErrMsg），
				// 包含腾讯云 SDK 的 Code / Message / RequestId，便于定位规则被拒原因。
				log := Logger(ctx)
				var tce *tcerr.TencentCloudSDKError
				if errors.As(err, &tce) {
					log.Warn("[ruleset] fan-out single SG failed (TencentCloudSDKError)",
						"sg_id", sg.SGID,
						"code", tce.Code,
						"cloud_request_id", tce.RequestId,
						"message", tce.Message,
						"rules_bytes", len(newRulesJSON),
						"rules_preview", previewRulesJSON(newRulesJSON, 1500))
				} else {
					log.Warn("[ruleset] fan-out single SG failed (non-SDK error)",
						"sg_id", sg.SGID,
						"rules_bytes", len(newRulesJSON),
						"rules_preview", previewRulesJSON(newRulesJSON, 1500),
						"err_full", err.Error())
				}
				mu.Lock()
				failErrs = append(failErrs, DriftError{SGID: sg.SGID, Error: truncateErrMsg(err.Error())})
				mu.Unlock()
				return
			}
			mu.Lock()
			successIDs = append(successIDs, sg.SGID)
			mu.Unlock()
		}()
	}
	wg.Wait()
	return successIDs, failErrs
}

// rollbackSGsToOldRules best-effort 把已成功下发的 SG 回滚到 rollbackJSON（旧规则）。
// 返回值：回滚失败的 SG 列表（云端保留新规则，DB 已被上层决定不改）。
//
// 这里不重试 transient：回滚是"止损"动作，快速失败比拖延更重要。Guardian 每小时会继续修复。
func rollbackSGsToOldRules(ctx context.Context, sgIDs []string, rollbackJSON string) (failures []DriftError) {
	log := Logger(ctx)
	var mu sync.Mutex
	failures = make([]DriftError, 0)
	sem := make(chan struct{}, fanoutConcurrency)
	var wg sync.WaitGroup

	for _, id := range sgIDs {
		sgID := id
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := applyRulesToCloudSG(ctx, sgID, rollbackJSON); err != nil {
				log.Warn("[ruleset] rollback sg to old rules failed", "sg_id", sgID, "err", err)
				mu.Lock()
				failures = append(failures, DriftError{SGID: sgID, Error: truncateErrMsg(err.Error())})
				mu.Unlock()
				return
			}
			log.Info("[ruleset] rolled back sg to old rules", "sg_id", sgID)
		}()
	}
	wg.Wait()
	return failures
}

// validateCIDR 校验 CIDR/IP 格式；不合法返回 error。
//
// 合法格式：
//   - IPv4 CIDR：`x.x.x.x/N`（N∈[0,32]），如 `0.0.0.0/0`、`10.0.0.0/24`
//   - IPv6 CIDR：`x:x::x/N`（N∈[0,128]）
//   - 单 IP（会被上层规范化为 /32 或 /128）：`192.168.1.1`、`::1`
//
// 不合法：
//   - 裸 `0.0.0.0`（用户意图通常是 /0，不是 /32；要求显式写 `/0` 避免歧义）
//   - 裸 `::`（同上）
//   - 无意义的表达：`10.0.0.1/33`、`xxx.yyy`
func validateCIDR(s string) error {
	// 特殊处理裸 0.0.0.0 / ::：强制要求用户显式写 /0 或 /32
	if s == "0.0.0.0" || s == "::" {
		return hcommon.I18nError(i18n.MsgRulesetCIDRFormatHint, s, s, s)
	}

	if !strings.Contains(s, "/") {
		// 单 IP：net.ParseIP 能解析即可，上层 normalizeCIDR 会补 /32 或 /128
		if net.ParseIP(s) == nil {
			return hcommon.I18nError(i18n.MsgRulesetInvalidIPOrCIDR)
		}
		return nil
	}
	if _, _, err := net.ParseCIDR(s); err != nil {
		return hcommon.I18nError(i18n.MsgRulesetInvalidCIDR, err)
	}
	return nil
}

// truncateErrMsg 防止单条错误把响应撑爆。腾讯云错误一般在 500 字节内。
func truncateErrMsg(s string) string {
	const max = 500
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}

// previewRulesJSON 截取 rulesJSON 头部用于排障日志，避免把超大规则集刷爆日志。
// 仅用于日志展示——真实下发用完整 JSON。
func previewRulesJSON(s string, max int) string {
	if max <= 0 {
		max = 500
	}
	if len(s) > max {
		return s[:max] + "...(truncated)"
	}
	return s
}

// applyRulesToSGWithRetry 对单 SG 调 ModifySecurityGroupPolicies，transient 错误指数退避重试 3 次。
// Permanent 错误（InvalidParameter / SecurityGroupNotFound / 配额）立即返回不重试。
// 退避与重试判定统一复用 RetryCloudCall（详见 cloud_retry.go）。
func applyRulesToSGWithRetry(ctx context.Context, sgID, rulesJSON string) error {
	return RetryCloudCall(ctx, func() error {
		return applyRulesToCloudSG(ctx, sgID, rulesJSON)
	})
}

// --- HTTP handlers（路由在 main.go 注册） ---

// ruleSetResponse `GET /admin/config/security-group/ruleset` 的响应体。
//
// initialized=false 表示当前租户尚未初始化 RuleSet（新租户从未配过云安全组）；
// 前端据此显示"未初始化"引导卡片，调用 POST .../rulesets 完成创建。
type ruleSetResponse struct {
	Initialized bool               `json:"initialized"`
	ID          uint               `json:"id,omitempty"`
	Name        string             `json:"name,omitempty"`
	Description string             `json:"description,omitempty"` // 管理员自定义备注，UI 展示用
	Version     int                `json:"version,omitempty"`
	Rules       []ruleResponseItem `json:"rules,omitempty"`
	ProjectedTo []sgRef            `json:"projected_to,omitempty"` // 只含 ACTIVE 成员

	// 预留字段：当前 RuleSet 作用到的用户组 ID 列表（本期恒为空数组）
	UserGroupIDs []string `json:"user_group_ids,omitempty"`
	// 预留字段：是否为默认规则组（本期单一 RuleSet 恒为 true）
	IsDefault bool `json:"is_default,omitempty"`
}

// ruleResponseItem 是 GET ruleset 响应里每条 rule 的视图类型：
// 在 Rule 的全部 JSON 字段基础上额外注入 `fingerprint` 字段，前端可直接复用，
// 不必再在 JS 侧重新实现 normalizeDirection/Port/CIDR 等归一化逻辑。
//
// ⚠️ 仅用于响应序列化，**绝不能用作 DB 持久化或入参反序列化**：
//   - DB 持久化路径（RuleSet.Rules text 字段）继续走 []Rule，避免把派生字段写进存储
//   - 入参 []Rule 的 JSON 反序列化里，多余的 fingerprint 字段会被自动忽略，无副作用
type ruleResponseItem struct {
	Rule
	Fingerprint string `json:"fingerprint"`
}

type sgRef struct {
	SGID     string `json:"sg_id"`
	SGName   string `json:"sg_name,omitempty"` // 云端分片名；由 Guardian 每 5 分钟从云 API 同步到 DB
	CVMCount int    `json:"cvm_count"`
}

// HandleGetRuleSet GET /admin/config/security-group/ruleset
// 已初始化：返回 {initialized:true, rules, version, projected_to}；
// 未初始化（全新租户从未触发 initialize）：返回 200 {initialized:false}。前端据此展示引导。
func HandleGetRuleSet(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}
	log := Logger(r.Context())
	rs, err := GetDefaultRuleSet(r.Context())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 未初始化：正常响应让前端进入空状态，而不是报错
			jsonOK(w, ruleSetResponse{Initialized: false})
			return
		}
		log.Error("[ruleset] get default rule_set failed", "err", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
		return
	}
	writeRuleSetResponse(w, r, rs)
}

// HandleCreateRuleSet POST /admin/config/security-group/rulesets
//
// 创建一个规则组（RuleSet）。本期每租户固定一个 name="default" 的 RuleSet，
// 未来会支持多 RuleSet（按用户组路由），届时此接口的 name 参数和列表语义直接兼容。
//
// 本期同时会为新建的 RuleSet 建第一个 ACTIVE SG（规则投影容器）；未来多 RuleSet
// 场景下可能拆出单独的 "provision" 操作（某个 RuleSet 首次被用户组绑定时才建 SG）。
//
// 请求体：
//
//	{
//	  "name": "default",             // 可选，本期固定 default；传其它值会被校验为不支持
//	  "rules": [...],                // 可选，管理员在弹窗里配置的自定义规则（必需规则由后端自动合并）
//	  "import_from_sg_id": "sg-xxx"  // 可选，若提供则从此 SG 读规则作基底（忽略 rules 参数）
//	}
//
// 行为：
//  1. 幂等：若同 name 的 RuleSet 已存在，直接返回当前详情
//  2. 否则走 CreateInitialRuleSetAndSG（不标 FROZEN，因为没有老 base）
//  3. 返回 {initialized:true, version:1, rules, projected_to}
func HandleCreateRuleSet(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}
	ctx := r.Context()
	log := Logger(ctx)

	// 1. 快速幂等检查
	if rs, err := GetDefaultRuleSet(r.Context()); err == nil {
		log.Info("[ruleset-create] already exists, returning existing state", "rule_set_id", rs.ID)
		writeRuleSetResponse(w, r, rs)
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Error("[ruleset-create] pre-check failed", "err", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
		return
	}

	// 2. 解析请求体
	var req struct {
		Name           string `json:"name,omitempty"`
		Description    string `json:"description,omitempty"` // 管理员自定义备注，UI 展示用，不影响业务逻辑
		Rules          []Rule `json:"rules,omitempty"`
		ImportFromSGID string `json:"import_from_sg_id,omitempty"`
		AutoFixRules   bool   `json:"auto_fix_rules,omitempty"`
	}
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgInvalidJSON))
			return
		}
	}

	// 本期每租户仅一行 RuleSet。若传了 name 且不是 default 之外的合法名，放行；
	// 若传了空，后端补默认。校验规则：长度 1-64，允许字母/数字/短横/下划线。
	rsName := strings.TrimSpace(req.Name)
	if rsName == "" {
		rsName = model.DefaultRuleSetName
	}
	if !isValidRuleSetName(rsName) {
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgRuleSetNameInvalid, rsName))
		return
	}
	// 预留说明：本期每租户仅允许一行 RuleSet；未来支持多 RuleSet 时此处会放开
	// "同名已存在"的幂等检查扩展到 GetRuleSetByName(rsName)。

	// 3. 分布式锁串行化并发请求
	//    与 EnsureDefaultRuleSet 共用同一把锁，避免启动期自愈和 HTTP 创建 race。
	//    锁 key 无需拼 identifier：model.AcquireLock 内部的 lockName() 会自动加
	//    "{identifier}:{resource}" 前缀做多租户隔离。
	lock, err := model.AcquireLock(ctx, "sg-bootstrap", 60*time.Second)
	if err != nil {
		writeError(w, r, http.StatusConflict, hcommon.I18nRichError(err, i18n.MsgRuleSetCreateConflict))
		return
	}
	defer lock.Release()

	// 4. double-check
	if rs, err := GetDefaultRuleSet(r.Context()); err == nil {
		writeRuleSetResponse(w, r, rs)
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
		return
	}

	// 5. 决定初始 rules 来源
	var userRules []Rule
	if strings.TrimSpace(req.ImportFromSGID) != "" {
		sgID := strings.TrimSpace(req.ImportFromSGID)
		// 校验不是 clawpro 自建（避免从自家池子导入造成循环）
		isManaged, mErr := model.IsManagedSG(ctx, sgID)
		if mErr != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(mErr, i18n.MsgRuleSetCheckManagedSGFail))
			return
		}
		if isManaged {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgRuleSetCannotImportFromCP))
			return
		}
		rules, dErr := DescribeSGPoliciesWithRetry(ctx, sgID)
		if dErr != nil {
			writeError(w, r, http.StatusBadGateway, hcommon.I18nRichError(dErr, i18n.MsgRuleSetReadSourceSGFail))
			return
		}
		userRules = rules
		log.Info("[ruleset-create] imported rules from external SG", "source_sg_id", sgID, "rule_count", len(rules))
	} else {
		userRules = req.Rules
	}

	// 6. 建规则组 + 首个 ACTIVE SG（无 FROZEN 老 base）
	//    规则合法性由腾讯云 API 兜底校验（cidr 模板 ipm-/sg-、参数模板等本地难枚举）
	//    前端主动建组：forceSkipMerge=false，autoFixRules 由请求 auto_fix_rules 字段决定
	//    （SiteConfig 启用 recommended 时仍然兜底注入，无视开关；详见 CreateInitialRuleSetAndSG 注释）
	rsID, newSGID, err := CreateInitialRuleSetAndSG(ctx, rsName, strings.TrimSpace(req.Description), userRules, "", 0, false /* forceSkipMerge */, req.AutoFixRules)
	if err != nil {
		log.Error("[ruleset-create] create failed", "err", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
		return
	}

	// 8. 审计 + 响应
	model.LogAudit(ctx, time.Now(), getUserIDFromRequest(r), "admin", "create_ruleset",
		"rule_set", fmt.Sprintf("%d", rsID), "success")
	log.Info("[ruleset-create] manual create success",
		"rule_set_id", rsID, "new_sg_id", newSGID, "source", req.ImportFromSGID)

	// 重新读 RuleSet 返回给前端
	rs, err := GetDefaultRuleSet(r.Context())
	if err != nil {
		// 理论不应该发生
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
		return
	}
	writeRuleSetResponse(w, r, rs)
}

// isValidRuleSetName RuleSet 名称校验。对齐 PRD 规则 7：
//   - 长度 3-32 字符
//   - 必须以字母开头
//   - 仅允许字母、数字、短横线（不允许下划线 / 点 / 其他符号）
//
// 前端在 name 输入框失焦时复用同一份规则做本地校验，失败在输入框下方红字提示。
func isValidRuleSetName(name string) bool {
	n := len(name)
	if n < 3 || n > 32 {
		return false
	}
	for i, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case (r >= '0' && r <= '9') && i != 0: // 首字符不能是数字
		case r == '-' && i != 0: // 首字符不能是短横线
		default:
			return false
		}
	}
	return true
}

// writeRuleSetResponse 和 HandleGetRuleSet 共享的响应构造。
//
// 这里把 DB 反序列化得到的 []Rule 转换成 []ruleResponseItem，附加 `fingerprint`
// 派生字段：前端拖拽排序后可直接 .map(r => r.fingerprint) 拿到 ordered_fingerprints
// 提交给 POST /admin/config/security-group/ruleset/rules/reorder，无需在前端重写归一化算法。
func writeRuleSetResponse(w http.ResponseWriter, r *http.Request, rs *model.RuleSet) {
	ctx := r.Context()
	var rules []Rule
	if rs.Rules != "" && rs.Rules != "[]" {
		_ = json.Unmarshal([]byte(rs.Rules), &rules)
	}
	// 注入 fingerprint：派生字段，仅出现在响应里，不写回 DB
	responseRules := make([]ruleResponseItem, 0, len(rules))
	for _, rule := range rules {
		responseRules = append(responseRules, ruleResponseItem{
			Rule:        rule,
			Fingerprint: rule.Fingerprint(),
		})
	}
	// 只投影 ACTIVE 成员；sg_name 是由 Guardian 每 5 分钟从云 API 同步的值，
	// 不用代码侧按序号拼（避免脏数据：云端被改名后 DB 会在一个周期内对齐）。
	activeSGs, _ := model.ListActiveSGsForFanout(ctx, rs.ID)
	projected := make([]sgRef, 0, len(activeSGs))
	for _, sg := range activeSGs {
		projected = append(projected, sgRef{
			SGID:     sg.SGID,
			SGName:   sg.SGName,
			CVMCount: sg.CVMCount,
		})
	}
	// 反序列化预留字段 user_group_ids：DB 存的是 JSON 字符串，空串 / "[]" / 非法 JSON 都视为空切片
	userGroupIDs := []string{}
	if s := strings.TrimSpace(rs.UserGroupIDs); s != "" && s != "[]" {
		_ = json.Unmarshal([]byte(s), &userGroupIDs)
	}
	jsonOK(w, ruleSetResponse{
		Initialized:  true,
		ID:           rs.ID,
		Name:         rs.Name,
		Description:  rs.Description,
		Version:      rs.Version,
		Rules:        responseRules,
		ProjectedTo:  projected,
		UserGroupIDs: userGroupIDs,
		IsDefault:    rs.IsDefault,
	})
}

// getUserIDFromRequest 从请求上下文拿当前管理员 ID，拿不到返回 0（audit fallback）。
func getUserIDFromRequest(r *http.Request) uint {
	if u, err := getUserFromToken(r); err == nil && u != nil {
		return u.ID
	}
	if u, err := getLoginUser(r); err == nil && u != nil {
		return u.ID
	}
	return 0
}

// HandleUpdateRuleSetRules POST /admin/config/security-group/ruleset/rules
// 整包提交规则：按 auto_fix_rules + SiteConfig 决定是否合并必需规则 → UPDATE rule_sets + version++ → 并发 fan-out。
//
// 请求体 name 字段可选，指定要更新哪个规则组；缺省时走 DefaultRuleSetName（"default"）。
// 未来支持多 RuleSet 时前端需要明确传 name；本期只有 default 一个，不传也能工作。
//
// auto_fix_rules（默认 false）：
//   - false：以 rules 原样落盘，不主动注入 ClawPro 必需规则（含 builtin SSH/出站）
//   - true：按 SiteConfig + builtin 规则 merge（系统注入兜底）
//
// "SiteConfig 兜底"覆盖语义：当 SiteConfig 启用任何 recommended（gateway/VNC）时，
// 即使 auto_fix_rules=false，必需规则仍会被 merge（避免管理员误删后业务功能不可用）。
func HandleUpdateRuleSetRules(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}
	log := Logger(r.Context())
	var req struct {
		Name         string `json:"name,omitempty"`
		Rules        []Rule `json:"rules"`
		AutoFixRules bool   `json:"auto_fix_rules,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgInvalidJSON))
		return
	}
	// 校验 name（空 → 后端 fallback default；非空必须通过正则）
	if strings.TrimSpace(req.Name) != "" && !isValidRuleSetName(strings.TrimSpace(req.Name)) {
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgRuleSetNameInvalid, req.Name))
		return
	}
	version, synced, driftErrs, err := UpdateRuleSetRulesInternal(r.Context(), strings.TrimSpace(req.Name), req.Rules, req.AutoFixRules)
	if err != nil {
		log.Error("[ruleset] update rules failed", "err", err, "drift_errors", len(driftErrs))
		if driftErrs == nil {
			driftErrs = []DriftError{}
		}
		// 把 driftErrs 里的腾讯云具体报错拼进 error 字段，前端只读 error 也能看到详情
		var errBuf strings.Builder
		for i, de := range driftErrs {
			if de.Error != "" {
				fmt.Fprintf(&errBuf, "\n[%d] %s", i+1, de.Error)
			}
		}
		rerr := hcommon.EnsureRichErrorOrPanic(err)
		writeError(w, r, http.StatusConflict, rerr.WithDetail(errBuf.String()).
			WithCustomData(map[string]any{
				"version":      version,
				"synced":       synced,
				"drifted":      len(driftErrs),
				"drift_errors": driftErrs,
			}))
		return
	}
	if driftErrs == nil {
		driftErrs = []DriftError{}
	}
	jsonOK(w, map[string]any{
		"version":      version,
		"synced":       synced,
		"drifted":      len(driftErrs),
		"drift_errors": driftErrs,
	})
}

// HandleImportRulesFromSG POST /admin/config/security-group/ruleset/import-from-sg
// 从云账号下任一 SG 读规则，覆盖指定 name 的 RuleSet。name 缺省走 default。
// 请求体：{"name": "default", "source_sg_id": "sg-xxx", "auto_fix_rules": false}。
// 源 SG 不能是 clawpro 自建。
//
// auto_fix_rules（默认 false）：
//   - false：纯导入，以源 SG 规则原样落盘，不注入任何 ClawPro 必需规则
//   - true：导入 + 按 SiteConfig 合并 ClawPro 必需规则（"未来要用的规则一次性备好"）
func HandleImportRulesFromSG(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}
	log := Logger(r.Context())
	var req struct {
		Name         string `json:"name,omitempty"`
		SourceSGID   string `json:"source_sg_id"`
		AutoFixRules bool   `json:"auto_fix_rules,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgInvalidJSON))
		return
	}
	if strings.TrimSpace(req.SourceSGID) == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgRuleSetSourceSGIDRequired))
		return
	}
	if strings.TrimSpace(req.Name) != "" && !isValidRuleSetName(strings.TrimSpace(req.Name)) {
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgRuleSetNameInvalid, req.Name))
		return
	}
	version, synced, driftErrs, err := ImportRulesFromSGInternal(r.Context(), strings.TrimSpace(req.Name), req.SourceSGID, req.AutoFixRules)
	if err != nil {
		log.Error("[ruleset] import rules failed", "source_sg_id", req.SourceSGID, "auto_fix_rules", req.AutoFixRules, "err", err, "drift_errors", len(driftErrs))
		if driftErrs == nil {
			driftErrs = []DriftError{}
		}
		var errBuf strings.Builder
		for i, de := range driftErrs {
			if de.Error != "" {
				fmt.Fprintf(&errBuf, "\n[%d] %s", i+1, de.Error)
			}
		}
		rerr := hcommon.EnsureRichErrorOrPanic(err)
		writeError(w, r, http.StatusConflict, rerr.WithDetail(errBuf.String()).WithCustomData(map[string]any{
			"version":       version,
			"synced":        synced,
			"drifted":       len(driftErrs),
			"drift_errors":  driftErrs,
			"imported_from": req.SourceSGID,
		}))
		return
	}
	if driftErrs == nil {
		driftErrs = []DriftError{}
	}
	jsonOK(w, map[string]any{
		"version":       version,
		"synced":        synced,
		"drifted":       len(driftErrs),
		"drift_errors":  driftErrs,
		"imported_from": req.SourceSGID,
	})
}

// --- 本文件内部：不影响外部的辅助 ---

// 消掉 Import 意外的 unused（rand 在 sg_pool.go 也被 import）
var _ = slog.Default

// 让 import "net" 被明确使用在 normalizeCIDR 之外（防止未来某次重构导致 import 漂移）
var _ = net.ParseIP
