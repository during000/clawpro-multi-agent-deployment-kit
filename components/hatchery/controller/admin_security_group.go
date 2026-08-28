package controller

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
	"gorm.io/gorm"
)

// ClawproRequiredSGRulesJSON 由外部注入的 ClawPro 所需安全组规则 JSON 数据。
// release 模式下通过 go:embed 嵌入 config/clawpro_required_sg_rules.json，
// dev 模式下从磁盘读取 config/clawpro_required_sg_rules.json。
var ClawproRequiredSGRulesJSON []byte

// requiredSGRule 描述一条 ClawPro 所需的安全组规则（用于规则检查）
type requiredSGRule struct {
	Direction   string `json:"direction"`   // "ingress" 或 "egress"
	Protocol    string `json:"protocol"`    // 协议，如 "TCP"、"UDP"、"ALL"
	Port        string `json:"port"`        // 端口或端口范围，如 "22"、"ALL"
	CidrBlock   string `json:"cidr_block"`  // IPv4 CIDR，如 "0.0.0.0/0"
	Ipv6Cidr    string `json:"ipv6_cidr"`   // IPv6 CIDR，如 "::/0"（可选）
	Action      string `json:"action"`      // "ACCEPT" 或 "DROP"
	Description string `json:"description"` // 规则说明
}

// sgRuleGroup 描述一组同名的安全组规则
type sgRuleGroup struct {
	Key            string           `json:"key"`                       // 规则组标识，如 "allow_ssh"、"allow_internet"
	Name           string           `json:"name"`                      // 规则组中文名称，如 "允许LinuxSSH登录"
	DefaultChecked bool             `json:"default_checked,omitempty"` // 是否默认选中
	Condition      string           `json:"condition,omitempty"`       // 条件标识，如 "gateway_ui_enable"，为空表示无条件
	Rules          []requiredSGRule `json:"rules"`                     // 该组包含的规则列表
}

// sgRuleCategory 描述一个规则分类（如内置规则、推荐规则）
type sgRuleCategory struct {
	Type       string        `json:"type"`        // 分类类型标识，如 "builtin"、"recommended"
	Label      string        `json:"label"`       // 分类中文名称，如 "内置规则"、"ClawPro 推荐规则"
	RuleGroups []sgRuleGroup `json:"rule_groups"` // 该分类下按名称分组的规则列表
}

// sgRuleSet 描述完整的安全组规则集合，包含多个分类
type sgRuleSet struct {
	Categories []sgRuleCategory `json:"categories"` // 规则分类列表
}

// createSGRequest 创建安全组请求体
//
// QuickRules 为规则组标识（key）列表，支持以下标识：
//   - "restrict_vpc_access"：内置特殊规则，入站拒绝 VPC CIDR（CIDR 由后端从 siteconfig.VpcId 自动查询）
//   - 其他标识：从 config/clawpro_required_sg_rules.json 的所有分类（builtin/recommended）中按 key 匹配，
//     如 "allow_ssh"、"allow_internet"、"allow_http"、"allow_https" 等。
//     新增规则只需在 JSON 配置中添加 rule_group，无需修改后端代码。
type createSGRequest struct {
	GroupName        string   `json:"GroupName"`
	GroupDescription string   `json:"GroupDescription"`
	QuickRules       []string `json:"quick_rules"`
}

// sgPolicyQuerier 抽象安全组策略查询能力，便于测试时替换（*vpc.Client 天然实现该接口）。
type sgPolicyQuerier interface {
	DescribeSecurityGroupPolicies(request *vpc.DescribeSecurityGroupPoliciesRequest) (response *vpc.DescribeSecurityGroupPoliciesResponse, err error)
}

// sgPolicyWriter 抽象安全组策略写入能力，便于测试时替换。
type sgPolicyWriter interface {
	CreateSecurityGroupPolicies(request *vpc.CreateSecurityGroupPoliciesRequest) (response *vpc.CreateSecurityGroupPoliciesResponse, err error)
}

// cvmSgBinder 抽象 CVM 实例的安全组绑定/解绑能力，便于测试时替换。
type cvmSgBinder interface {
	AssociateSecurityGroups(request *cvm.AssociateSecurityGroupsRequest) (response *cvm.AssociateSecurityGroupsResponse, err error)
	DisassociateSecurityGroups(request *cvm.DisassociateSecurityGroupsRequest) (response *cvm.DisassociateSecurityGroupsResponse, err error)
}

// sgVpcClient 聚合了安全组相关 handler 所需的 VPC 客户端能力。
// *vpc.Client 天然实现该接口，测试时可注入 fake 实例以覆盖 handler 主流程。
type sgVpcClient interface {
	DescribeSecurityGroups(request *vpc.DescribeSecurityGroupsRequest) (response *vpc.DescribeSecurityGroupsResponse, err error)
	CreateSecurityGroup(request *vpc.CreateSecurityGroupRequest) (response *vpc.CreateSecurityGroupResponse, err error)
	DeleteSecurityGroup(request *vpc.DeleteSecurityGroupRequest) (response *vpc.DeleteSecurityGroupResponse, err error)
	ModifySecurityGroupAttribute(request *vpc.ModifySecurityGroupAttributeRequest) (response *vpc.ModifySecurityGroupAttributeResponse, err error)
	DescribeSecurityGroupPolicies(request *vpc.DescribeSecurityGroupPoliciesRequest) (response *vpc.DescribeSecurityGroupPoliciesResponse, err error)
	CreateSecurityGroupPolicies(request *vpc.CreateSecurityGroupPoliciesRequest) (response *vpc.CreateSecurityGroupPoliciesResponse, err error)
	ModifySecurityGroupPolicies(request *vpc.ModifySecurityGroupPoliciesRequest) (response *vpc.ModifySecurityGroupPoliciesResponse, err error)
	ReplaceSecurityGroupPolicy(request *vpc.ReplaceSecurityGroupPolicyRequest) (response *vpc.ReplaceSecurityGroupPolicyResponse, err error)
	DeleteSecurityGroupPolicies(request *vpc.DeleteSecurityGroupPoliciesRequest) (response *vpc.DeleteSecurityGroupPoliciesResponse, err error)
	DescribeVpcs(request *vpc.DescribeVpcsRequest) (response *vpc.DescribeVpcsResponse, err error)
	DescribeSecurityGroupAssociationStatistics(request *vpc.DescribeSecurityGroupAssociationStatisticsRequest) (response *vpc.DescribeSecurityGroupAssociationStatisticsResponse, err error)
}

// 以下函数变量在测试中可被替换，用于隔离外部依赖。
var (
	describeInstancesSecurityGroupsFn = describeInstancesSecurityGroups
	newCVMClientFn                    = func(ctx context.Context) (cvmSgBinder, error) {
		c, err := NewCVMClient(ctx)
		if err != nil {
			return nil, err
		}
		return c, nil
	}
	// newVpcClientForSGFn 用于创建安全组相关 handler 使用的 VPC 客户端，测试中可替换。
	// 默认实现包装 newVpcClient()，返回 *vpc.Client（天然实现 sgVpcClient 接口）。
	newVpcClientForSGFn = func(ctx context.Context) (sgVpcClient, error) {
		c, err := newVpcClient(ctx)
		if err != nil {
			return nil, err
		}
		return c, nil
	}
	// listInstanceIdsFn 保留供测试 hook 使用（历史旧方案遗留，本期无生产调用）。
	listInstanceIdsFn = listInstanceIds
	// addMissingSGRulesRetryInterval 控制 addMissingSGRules 失败后的重试间隔，
	// 测试中可将其置零以避免 sleep。
	addMissingSGRulesRetryInterval = 2 * time.Second
)

func HandleGetSecurityGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := Logger(ctx)
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}
	config := model.GetSiteConfig(r.Context())
	if config.SecurityGroupId == "" {
		return
	}

	vpcClient, err := newVpcClientForSGFn(ctx)
	if err != nil {
		log.Error("查询安全组：创建 VPC 客户端失败", "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgCreateVPCClientFailed))
		return
	}

	req := vpc.NewDescribeSecurityGroupsRequest()
	req.SecurityGroupIds = common.StringPtrs([]string{config.SecurityGroupId})
	resp, err := vpcClient.DescribeSecurityGroups(req)
	if err != nil {
		log.Error("查询安全组失败", "security_group_id", config.SecurityGroupId, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSGQuerySGFailed))
		return
	}

	log.Info("查询安全组成功", "security_group_id", config.SecurityGroupId)
	jsonOK(w, resp)
}

func HandleCreateSecurityGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := Logger(ctx)
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error("创建安全组：读取请求体失败", "error", err)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgReadRequestBodyFailedWithError))
		return
	}

	// 解析请求体，支持 quick_rules 扩展字段
	var reqBody createSGRequest
	if err := json.Unmarshal(body, &reqBody); err != nil {
		log.Error("创建安全组：请求参数格式错误", "error", err)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgBadRequestWithError))
		return
	}
	if reqBody.GroupName == "" {
		log.Warn("创建安全组：安全组名称为空")
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSGNameRequired))
		return
	}

	log.Info("开始创建安全组", "group_name", reqBody.GroupName, "quick_rules", reqBody.QuickRules)
	config := model.GetSiteConfig(r.Context())
	vpcClient, err := newVpcClientForSGFn(ctx)
	if err != nil {
		log.Error("创建安全组：创建 VPC 客户端失败", "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgCreateVPCClientFailed))
		return
	}

	req := vpc.NewCreateSecurityGroupRequest()
	req.GroupName = common.StringPtr(reqBody.GroupName)
	req.GroupDescription = common.StringPtr(reqBody.GroupDescription)

	resp, err := vpcClient.CreateSecurityGroup(req)
	if err != nil {
		log.Error("创建安全组失败", "group_name", reqBody.GroupName, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSGCreateSGFailed))
		return
	}
	if resp.Response == nil || resp.Response.SecurityGroup == nil || resp.Response.SecurityGroup.SecurityGroupId == nil {
		log.Error("创建安全组返回数据异常", "group_name", reqBody.GroupName)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgCreateSecurityGroupDataError))
		return
	}

	sgId := *resp.Response.SecurityGroup.SecurityGroupId
	oldSgId := config.SecurityGroupId
	model.DB(r.Context()).Model(&config).Update("security_group_id", sgId)
	log.Info("安全组创建成功", "security_group_id", sgId, "group_name", reqBody.GroupName, "old_sg_id", oldSgId)

	// 处理快速规则：遍历规则组名称列表，按名称匹配生成安全组策略
	if len(reqBody.QuickRules) > 0 {
		log.Info("开始处理快速规则", "security_group_id", sgId, "quick_rules", reqBody.QuickRules)
		var ingressPolicies []*vpc.SecurityGroupPolicy
		var egressPolicies []*vpc.SecurityGroupPolicy

		// 加载配置文件中的规则组，构建 key -> rules 索引（遍历所有分类）
		ruleSet := clawproRequiredRuleSet()
		ruleGroupMap := make(map[string][]requiredSGRule)
		for _, category := range ruleSet.Categories {
			for _, group := range category.RuleGroups {
				ruleGroupMap[group.Key] = group.Rules
			}
		}

		for _, ruleName := range reqBody.QuickRules {
			// 内置特殊规则：restrict_vpc_access 需要动态查询 VPC CIDR
			if ruleName == "restrict_vpc_access" {
				vpcCidr := ""
				if config.VpcId != "" {
					descVpcReq := vpc.NewDescribeVpcsRequest()
					descVpcReq.VpcIds = common.StringPtrs([]string{config.VpcId})
					if descVpcResp, err := vpcClient.DescribeVpcs(descVpcReq); err != nil {
						log.Warn("查询 VPC CIDR 失败，跳过限制互访规则", "vpc_id", config.VpcId, "error", err)
					} else if descVpcResp.Response != nil && len(descVpcResp.Response.VpcSet) > 0 && descVpcResp.Response.VpcSet[0].CidrBlock != nil {
						vpcCidr = *descVpcResp.Response.VpcSet[0].CidrBlock
					}
				}
				if vpcCidr != "" {
					ingressPolicies = append(ingressPolicies, &vpc.SecurityGroupPolicy{
						Protocol:          common.StringPtr("ALL"),
						Port:              common.StringPtr("ALL"),
						CidrBlock:         common.StringPtr(vpcCidr),
						Action:            common.StringPtr("DROP"),
						PolicyDescription: common.StringPtr("限制 VPC 下 OpenClaw 云服务器互访"),
					})
				} else {
					log.Warn("未配置全局 VPC 或 VPC CIDR 为空，跳过限制互访规则")
				}
				continue
			}

			// 通用规则：从配置文件的 rule_groups 中按 name 匹配
			rules, ok := ruleGroupMap[ruleName]
			if !ok {
				log.Warn("未知的快速规则名称，跳过", "rule_name", ruleName)
				continue
			}
			for _, rule := range rules {
				policy := &vpc.SecurityGroupPolicy{
					Protocol:          common.StringPtr(rule.Protocol),
					Port:              common.StringPtr(rule.Port),
					Action:            common.StringPtr(rule.Action),
					PolicyDescription: common.StringPtr(rule.Description),
				}
				if rule.CidrBlock != "" {
					policy.CidrBlock = common.StringPtr(rule.CidrBlock)
				}
				if rule.Ipv6Cidr != "" {
					policy.Ipv6CidrBlock = common.StringPtr(rule.Ipv6Cidr)
				}
				if rule.Direction == "ingress" {
					ingressPolicies = append(ingressPolicies, policy)
				} else {
					egressPolicies = append(egressPolicies, policy)
				}
			}
		}

		// 批量创建规则（腾讯云不支持同时传入 Ingress 和 Egress，需分开调用）
		const maxRetry = 3
		var quickRuleErr error

		if len(ingressPolicies) > 0 {
			ingressReq := vpc.NewCreateSecurityGroupPoliciesRequest()
			ingressReq.SecurityGroupId = common.StringPtr(sgId)
			ingressReq.SecurityGroupPolicySet = &vpc.SecurityGroupPolicySet{
				Ingress: ingressPolicies,
			}
			var policyErr error
			for i := range maxRetry {
				if _, policyErr = vpcClient.CreateSecurityGroupPolicies(ingressReq); policyErr == nil {
					break
				}
				log.Warn("创建安全组入站快速规则失败，准备重试", "security_group_id", sgId, "attempt", i+1, "error", policyErr)
				if i < maxRetry-1 {
					time.Sleep(2 * time.Second)
				}
			}
			if policyErr != nil {
				log.Error("创建安全组入站快速规则失败，已达最大重试次数", "security_group_id", sgId, "error", policyErr)
				quickRuleErr = policyErr
			} else {
				log.Info("安全组入站快速规则创建完成", "security_group_id", sgId, "ingress_count", len(ingressPolicies))
			}
		}

		if len(egressPolicies) > 0 {
			egressReq := vpc.NewCreateSecurityGroupPoliciesRequest()
			egressReq.SecurityGroupId = common.StringPtr(sgId)
			egressReq.SecurityGroupPolicySet = &vpc.SecurityGroupPolicySet{
				Egress: egressPolicies,
			}
			var policyErr error
			for i := range maxRetry {
				if _, policyErr = vpcClient.CreateSecurityGroupPolicies(egressReq); policyErr == nil {
					break
				}
				log.Warn("创建安全组出站快速规则失败，准备重试", "security_group_id", sgId, "attempt", i+1, "error", policyErr)
				if i < maxRetry-1 {
					time.Sleep(2 * time.Second)
				}
			}
			if policyErr != nil {
				log.Error("创建安全组出站快速规则失败，已达最大重试次数", "security_group_id", sgId, "error", policyErr)
				quickRuleErr = policyErr
			} else {
				log.Info("安全组出站快速规则创建完成", "security_group_id", sgId, "egress_count", len(egressPolicies))
			}
		}

		if quickRuleErr != nil {
			log.Error("安全组快速规则部分或全部创建失败", "security_group_id", sgId, "error", quickRuleErr)
			// 不中断流程，安全组已创建成功
		} else if len(ingressPolicies) > 0 || len(egressPolicies) > 0 {
			log.Info("安全组快速规则全部创建完成", "security_group_id", sgId,
				"ingress_count", len(ingressPolicies), "egress_count", len(egressPolicies))
		}
	}

	// sg-ruleset-projection 方案下 HandleCreateSecurityGroup 降级为"云端创建 SG 辅助工具"——
	// 旧方案里紧跟着的"异步换绑所有实例到新 SG"逻辑已删除，因为 SG 选择由 SelectSGForNewInstance 统一管理。
	// 如果管理员想让某个新创建的 SG 替代老 base，应该通过 import-from-sg 导入规则（而非换绑实例）。
	jsonOK(w, resp)
}

// HandleBindSecurityGroup 绑定已有安全组（切换安全组）
// 将指定安全组 ID 保存为当前站点配置的安全组，并将所有实例的安全组切换到新安全组。
// 当 auto_fix_rules 为 true 时，自动检查并补齐 ClawPro 所需的缺失安全组规则。
func HandleBindSecurityGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := Logger(ctx)
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	var reqBody struct {
		SecurityGroupId string `json:"security_group_id"`
		AutoFixRules    bool   `json:"auto_fix_rules"` // 是否自动补齐缺失的安全组规则
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		log.Error("绑定安全组：请求参数格式错误", "error", err)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgBadRequestWithError, err))
		return
	}
	if reqBody.SecurityGroupId == "" {
		log.Warn("绑定安全组：security_group_id 为空")
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSGSecurityGroupIDRequired))
		return
	}

	// 拒绝绑定与当前已配置相同的安全组（绑定自身）
	currentConfig := model.GetSiteConfig(r.Context())
	if currentConfig.SecurityGroupId != "" && currentConfig.SecurityGroupId == reqBody.SecurityGroupId {
		log.Warn("绑定安全组：不能绑定当前已使用的安全组", "security_group_id", reqBody.SecurityGroupId)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSGAlreadyBound))
		return
	}

	log.Info("开始绑定安全组", "security_group_id", reqBody.SecurityGroupId, "auto_fix_rules", reqBody.AutoFixRules)

	// 验证安全组存在
	vpcClient, err := newVpcClientForSGFn(ctx)
	if err != nil {
		log.Error("绑定安全组：创建 VPC 客户端失败", "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgCreateVPCClientFailed))
		return
	}
	descReq := vpc.NewDescribeSecurityGroupsRequest()
	descReq.SecurityGroupIds = common.StringPtrs([]string{reqBody.SecurityGroupId})
	descResp, err := vpcClient.DescribeSecurityGroups(descReq)
	if err != nil {
		log.Error("绑定安全组：验证安全组失败", "security_group_id", reqBody.SecurityGroupId, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgVerifySecurityGroupFailed))
		return
	}
	if descResp.Response == nil || len(descResp.Response.SecurityGroupSet) == 0 {
		log.Warn("绑定安全组：安全组不存在", "security_group_id", reqBody.SecurityGroupId)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSGNotExist))
		return
	}
	log.Info("绑定安全组：安全组验证通过", "security_group_id", reqBody.SecurityGroupId)

	// 当 auto_fix_rules 为 true 时，检查并自动补齐缺失规则
	var fixedCount int
	if reqBody.AutoFixRules {
		log.Info("开始检查安全组规则缺失情况", "security_group_id", reqBody.SecurityGroupId)
		missingRules, err := checkMissingSGRules(ctx, vpcClient, reqBody.SecurityGroupId)
		if err != nil {
			log.Error("绑定安全组：检查安全组规则失败", "security_group_id", reqBody.SecurityGroupId, "error", err)
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSGCheckRulesFailed))
			return
		}
		log.Info("安全组规则检查完成", "security_group_id", reqBody.SecurityGroupId, "missing_count", len(missingRules))
		if len(missingRules) > 0 {
			log.Info("开始自动补齐缺失安全组规则", "security_group_id", reqBody.SecurityGroupId, "missing_count", len(missingRules))
			fixedCount, err = addMissingSGRules(vpcClient, reqBody.SecurityGroupId, missingRules)
			if err != nil {
				log.Error("绑定安全组：自动补齐安全组规则失败", "security_group_id", reqBody.SecurityGroupId, "error", err)
				writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSGAutoFixRulesFailed))
				return
			}
			log.Info("自动补齐缺失安全组规则完成", "security_group_id", reqBody.SecurityGroupId, "fixed_count", fixedCount)
		}
	}

	config := model.GetSiteConfig(r.Context())
	oldSgId := config.SecurityGroupId

	// 更新配置
	model.DB(r.Context()).Model(&config).Update("security_group_id", reqBody.SecurityGroupId)
	log.Info("安全组已切换", "old_sg_id", oldSgId, "new_sg_id", reqBody.SecurityGroupId)

	// 异步将所有实例的安全组替换为仅包含新安全组（解绑所有旧安全组）
	go func(ctx context.Context) {
		instanceIds, err := listInstanceIdsFn(ctx)
		if err != nil {
			slog.Error("切换安全组：查询实例列表失败", "error", err)
			return
		}
		if len(instanceIds) == 0 {
			return
		}
		if err := rebindAllInstancesToSingleSG(ctx, instanceIds, reqBody.SecurityGroupId); err != nil {
			slog.Error("切换安全组：换绑实例安全组失败", "error", err)
		}
	}(hcommon.DetachContext(r.Context()))

	result := map[string]interface{}{
		"ok":                true,
		"security_group_id": reqBody.SecurityGroupId,
	}
	if reqBody.AutoFixRules {
		result["fixed_rules_count"] = fixedCount
	}
	jsonOK(w, result)
}

func HandleUpdateSecurityGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := Logger(ctx)
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error("修改安全组：读取请求体失败", "error", err)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgReadRequestBodyFailedWithError))
		return
	}

	config := model.GetSiteConfig(r.Context())
	if config.SecurityGroupId == "" {
		log.Warn("修改安全组：未配置安全组")
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSGNotConfiguredCreateFirst))
		return
	}

	vpcClient, err := newVpcClientForSGFn(ctx)
	if err != nil {
		log.Error("修改安全组：创建 VPC 客户端失败", "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgCreateVPCClientFailed))
		return
	}

	req := vpc.NewModifySecurityGroupAttributeRequest()
	if err := req.FromJsonString(string(body)); err != nil {
		log.Error("修改安全组：请求参数格式错误", "error", err)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgBadRequestWithError))
		return
	}
	// 强制使用当前配置的安全组 ID
	req.SecurityGroupId = common.StringPtr(config.SecurityGroupId)

	resp, err := vpcClient.ModifySecurityGroupAttribute(req)
	if err != nil {
		log.Error("修改安全组失败", "security_group_id", config.SecurityGroupId, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSGModifySGFailed))
		return
	}

	log.Info("修改安全组成功", "security_group_id", config.SecurityGroupId)
	jsonOK(w, resp)
}

// HandleDescribeCloudSGPolicies GET /admin/config/security-group/cloud-policies?security_group_id=sg-xxx
//
// 用途：管理员在"从其他安全组导入规则"弹窗里选中一个云端 SG 后，预览它的规则。
// 语义边界（不同于已废除的 HandleDescribeCloudSecurityGroupPolicies）：
//   - 仅供"导入前预览"的纯云端透传，不作为规则真相源（真相源在 DB `rule_sets`）。
//   - 不写任何状态，不影响 managed_sg_pool / RuleSet。
//   - security_group_id 来自 URL query，不依赖 SiteConfig。
//
// 响应：透传腾讯云 DescribeSecurityGroupPolicies（`{ SecurityGroupPolicySet: { Ingress: [...], Egress: [...] } }`）。
func HandleDescribeCloudSGPolicies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := Logger(ctx)
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	sgID := strings.TrimSpace(r.URL.Query().Get("security_group_id"))
	if sgID == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSGMissingSecurityGroupIDParam))
		return
	}

	vpcClient, err := newVpcClientForSGFn(ctx)
	if err != nil {
		log.Error("预览云端安全组规则：创建 VPC 客户端失败", "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgCreateVPCClientFailed))
		return
	}

	req := vpc.NewDescribeSecurityGroupPoliciesRequest()
	req.SecurityGroupId = common.StringPtr(sgID)
	resp, err := vpcClient.DescribeSecurityGroupPolicies(req)
	if err != nil {
		log.Error("预览云端安全组规则失败", "security_group_id", sgID, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQuerySGRulesFailed))
		return
	}

	jsonOK(w, resp.Response)
}

func HandleDescribeSecurityGroupPolicies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := Logger(ctx)
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}
	config := model.GetSiteConfig(r.Context())
	if config.SecurityGroupId == "" {
		log.Warn("查询安全组规则：未配置安全组")
		return
	}

	vpcClient, err := newVpcClientForSGFn(ctx)
	if err != nil {
		log.Error("查询安全组：创建 VPC 客户端失败", "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgCreateVPCClientFailed))
		return
	}

	req := vpc.NewDescribeSecurityGroupPoliciesRequest()
	req.SecurityGroupId = common.StringPtr(config.SecurityGroupId)
	resp, err := vpcClient.DescribeSecurityGroupPolicies(req)
	if err != nil {
		log.Error("查询安全组规则失败", "security_group_id", config.SecurityGroupId, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQuerySGRulesFailed))
		return
	}

	log.Info("查询安全组规则成功", "security_group_id", config.SecurityGroupId)
	jsonOK(w, resp)
}

// HandleCreateSecurityGroupPolicies Deprecated: 已于 sg-ruleset-projection 方案合并为 HandleUpdateRuleSetRules。
// 保留此 handler 兼容旧版前端，新代码请使用 POST /admin/config/security-group/ruleset/rules。
func HandleCreateSecurityGroupPolicies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := Logger(ctx)
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error("创建安全组规则：读取请求体失败", "error", err)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgReadRequestBodyFailedWithError))
		return
	}

	config := model.GetSiteConfig(r.Context())
	if config.SecurityGroupId == "" {
		log.Warn("创建安全组规则：未配置安全组")
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSGNotConfiguredCreateFirst))
		return
	}

	vpcClient, err := newVpcClientForSGFn(ctx)
	if err != nil {
		log.Error("创建安全组规则：创建 VPC 客户端失败", "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgCreateVPCClientFailed))
		return
	}

	req := vpc.NewCreateSecurityGroupPoliciesRequest()
	if err := req.FromJsonString(string(body)); err != nil {
		log.Error("创建安全组规则：请求参数格式错误", "error", err)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgBadRequestWithError))
		return
	}
	if req.SecurityGroupPolicySet == nil {
		log.Warn("创建安全组规则：缺少必填参数 policies")
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSGPoliciesRequired))
		return
	}
	// 强制使用当前配置的安全组 ID
	req.SecurityGroupId = common.StringPtr(config.SecurityGroupId)

	resp, err := vpcClient.CreateSecurityGroupPolicies(req)
	if err != nil {
		log.Error("创建安全组规则失败", "security_group_id", config.SecurityGroupId, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSGCreateSGRulesFailed))
		return
	}

	log.Info("创建安全组规则成功", "security_group_id", config.SecurityGroupId)
	jsonOK(w, resp)
}

// HandleReplaceSecurityGroupPolicy Deprecated: 已于 sg-ruleset-projection 方案合并为 HandleUpdateRuleSetRules。
// 保留此 handler 兼容旧版前端，新代码请使用 POST /admin/config/security-group/ruleset/rules。
func HandleReplaceSecurityGroupPolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := Logger(ctx)
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error("替换安全组规则：读取请求体失败", "error", err)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgReadRequestBodyFailedWithError))
		return
	}

	config := model.GetSiteConfig(r.Context())
	if config.SecurityGroupId == "" {
		log.Warn("替换安全组规则：未配置安全组")
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSGNotConfiguredCreateFirst))
		return
	}

	vpcClient, err := newVpcClientForSGFn(ctx)
	if err != nil {
		log.Error("替换安全组规则：创建 VPC 客户端失败", "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgCreateVPCClientFailed))
		return
	}

	req := vpc.NewReplaceSecurityGroupPolicyRequest()
	if err := req.FromJsonString(string(body)); err != nil {
		log.Error("替换安全组规则：请求参数格式错误", "error", err)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgBadRequestWithError))
		return
	}
	if req.SecurityGroupPolicySet == nil {
		log.Warn("替换安全组规则：缺少必填参数 policies")
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSGPoliciesRequired))
		return
	}
	// 强制使用当前配置的安全组 ID
	req.SecurityGroupId = common.StringPtr(config.SecurityGroupId)

	resp, err := vpcClient.ReplaceSecurityGroupPolicy(req)
	if err != nil {
		log.Error("替换安全组规则失败", "security_group_id", config.SecurityGroupId, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSGReplaceSGRulesFailed))
		return
	}

	log.Info("替换安全组规则成功", "security_group_id", config.SecurityGroupId)
	jsonOK(w, resp)
}

// HandleDeleteSecurityGroupPolicies Deprecated: 已于 sg-ruleset-projection 方案合并为 HandleUpdateRuleSetRules。
// 保留此 handler 兼容旧版前端，新代码请使用 POST /admin/config/security-group/ruleset/rules。
func HandleDeleteSecurityGroupPolicies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := Logger(ctx)
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error("删除安全组规则：读取请求体失败", "error", err)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgReadRequestBodyFailedWithError))
		return
	}

	config := model.GetSiteConfig(r.Context())
	if config.SecurityGroupId == "" {
		log.Warn("删除安全组规则：未配置安全组")
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSGNotConfiguredCreateFirst))
		return
	}

	vpcClient, err := newVpcClientForSGFn(ctx)
	if err != nil {
		log.Error("删除安全组规则：创建 VPC 客户端失败", "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgCreateVPCClientFailed))
		return
	}

	req := vpc.NewDeleteSecurityGroupPoliciesRequest()
	if err := req.FromJsonString(string(body)); err != nil {
		log.Error("删除安全组规则：请求参数格式错误", "error", err)
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgBadRequestWithError))
		return
	}
	if req.SecurityGroupPolicySet == nil {
		log.Warn("删除安全组规则：缺少必填参数 policies")
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSGPoliciesRequired))
		return
	}
	// 强制使用当前配置的安全组 ID
	req.SecurityGroupId = common.StringPtr(config.SecurityGroupId)

	resp, err := vpcClient.DeleteSecurityGroupPolicies(req)
	if err != nil {
		log.Error("删除安全组规则失败", "security_group_id", config.SecurityGroupId, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSGDeleteSGRulesFailed))
		return
	}

	log.Info("删除安全组规则成功", "security_group_id", config.SecurityGroupId)
	jsonOK(w, resp)
}

// HandleDescribeCloudSecurityGroupPolicies Deprecated: 请使用 HandleDescribeCloudSGPolicies。
// 保留此 handler 兼容旧版前端路由 /admin/config/security-group/cloud-policies。
var HandleDescribeCloudSecurityGroupPolicies = HandleDescribeCloudSGPolicies

// HandleListCloudSecurityGroups 列出当前账号/地域下的安全组（支持分页和过滤）
// 查询参数对齐腾讯云 DescribeSecurityGroups 接口：
//   - offset: 偏移量，默认 0
//   - limit: 每页数量，默认 20，最大 100
//   - keyword: 模糊搜索关键字（按安全组名称/ID 搜索）
//   - security_group_id: 精确匹配安全组 ID（多个用逗号分隔）
func HandleListCloudSecurityGroups(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := Logger(ctx)
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	vpcClient, err := newVpcClientForSGFn(ctx)
	if err != nil {
		log.Error("查询云安全组列表：创建 VPC 客户端失败", "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgCreateVPCClientFailed))
		return
	}

	// 解析分页参数
	query := r.URL.Query()
	offsetStr := query.Get("offset")
	limitStr := query.Get("limit")
	keyword := query.Get("keyword")
	sgIdParam := query.Get("security_group_id")

	// 默认值
	if offsetStr == "" {
		offsetStr = "0"
	}
	if limitStr == "" {
		limitStr = "20"
	}

	req := vpc.NewDescribeSecurityGroupsRequest()
	req.Offset = common.StringPtr(offsetStr)
	req.Limit = common.StringPtr(limitStr)

	// 支持按安全组 ID 精确查询（多个用逗号分隔）
	if sgIdParam != "" {
		var sgIds []string
		for _, id := range strings.Split(sgIdParam, ",") {
			id = strings.TrimSpace(id)
			if id != "" {
				// 入参格式校验：安全组 ID 必须为 sg- 前缀，拒绝旧格式（如 GZ-xxx）避免腾讯云 API 报错
				if !strings.HasPrefix(id, "sg-") {
					writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSGIDFormatInvalid, id))
					return
				}
				sgIds = append(sgIds, id)
			}
		}
		if len(sgIds) > 0 {
			req.SecurityGroupIds = common.StringPtrs(sgIds)
		}
	}

	// 支持关键字模糊搜索（按安全组名称过滤）
	if keyword != "" {
		req.Filters = []*vpc.Filter{
			{
				Name:   common.StringPtr("security-group-name"),
				Values: common.StringPtrs([]string{keyword}),
			},
		}
	}

	resp, err := vpcClient.DescribeSecurityGroups(req)
	if err != nil {
		log.Error("查询云安全组列表失败", "offset", offsetStr, "limit", limitStr, "keyword", keyword, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSGQuerySGListFailed))
		return
	}

	// 过滤掉 ClawPro 托管分片：它们只能被自动维护，不允许作为 base 手动选择。
	// sg-ruleset-projection 方案下：过滤掉 clawpro 托管的 ACTIVE SG，
	// 避免管理员在"从其他安全组导入规则"弹窗里误选 clawpro 自建的 SG（会造成循环依赖）。
	// FROZEN/RETIRED 是用户的老 base SG，保留在候选列表——管理员创建新 RuleSet 时可能想从它导入规则。
	managedIDSet := map[string]struct{}{}
	{
		var pool []model.ManagedSGPool
		if err := model.DB(r.Context()).Select("sg_id").Where("status = ?", model.SGStatusActive).Find(&pool).Error; err != nil {
			// 查询失败仅记日志，不阻断主流程（过滤是锦上添花）
			log.Warn("查询托管 SG 列表失败，不执行过滤", "error", err)
		} else {
			for _, s := range pool {
				managedIDSet[s.SGID] = struct{}{}
			}
		}
	}

	type sgItem struct {
		SecurityGroupId   string `json:"security_group_id"`
		SecurityGroupName string `json:"security_group_name"`
		SecurityGroupDesc string `json:"security_group_desc"`
		IsDefault         bool   `json:"is_default"`
	}
	var sgList []sgItem
	var totalCount uint64
	filteredOut := 0

	if resp.Response != nil {
		if resp.Response.TotalCount != nil {
			totalCount = *resp.Response.TotalCount
		}
		for _, sg := range resp.Response.SecurityGroupSet {
			if sg.SecurityGroupId != nil {
				// 过滤旧格式安全组 ID（如 GZ-xxx），这些 ID 无法用于精确查询且对用户无实际用途
				if !strings.HasPrefix(*sg.SecurityGroupId, "sg-") {
					filteredOut++
					continue
				}
				if _, skip := managedIDSet[*sg.SecurityGroupId]; skip {
					filteredOut++
					continue
				}
			}
			// 防御层过滤：名字以 clawpro-sg- 开头的 SG 即使不在 managed_sg_pool（DB 事务失败留下的孤儿），
			// 也不允许作为"从其他安全组导入规则"的模板——导入会造成规则循环。
			// PRD 3.2 规则 5：导入模板候选不展示以 clawpro-sg- 前缀命名的安全组。
			if sg.SecurityGroupName != nil && strings.HasPrefix(*sg.SecurityGroupName, "clawpro-sg-") {
				filteredOut++
				continue
			}
			item := sgItem{}
			if sg.SecurityGroupId != nil {
				item.SecurityGroupId = *sg.SecurityGroupId
			}
			if sg.SecurityGroupName != nil {
				item.SecurityGroupName = *sg.SecurityGroupName
			}
			if sg.SecurityGroupDesc != nil {
				item.SecurityGroupDesc = *sg.SecurityGroupDesc
			}
			if sg.IsDefault != nil {
				item.IsDefault = *sg.IsDefault
			}
			sgList = append(sgList, item)
		}
	}

	if sgList == nil {
		sgList = []sgItem{}
	}

	// total_count 按腾讯云原值返回但要扣掉过滤的 shard，否则前端分页会出现空页
	adjustedTotal := totalCount
	if uint64(filteredOut) <= adjustedTotal {
		adjustedTotal -= uint64(filteredOut)
	}

	log.Info("查询云安全组列表成功",
		"total_count", totalCount, "adjusted_total", adjustedTotal,
		"returned", len(sgList), "filtered_shards", filteredOut)
	jsonOK(w, map[string]interface{}{
		"security_groups": sgList,
		"total_count":     adjustedTotal,
	})
}

// HandleDescribeCloudSecurityGroupPolicies 已废弃 —— 规则真相源在 RuleSet（DB）而非云端。
// 前端"从其他安全组导入规则"请使用 POST /admin/config/security-group/ruleset/import-from-sg；
// 查看当前生效规则请使用 GET /admin/config/security-group/ruleset。

// agentProxyRouteKindEnabled 判断当前租户是否存在已启用的指定类型 agent proxy route。
// 该状态用于 agent_proxy_* 条件规则：只有存在可用代理入口时，才需要把对应端口规则合入 RuleSet。
func agentProxyRouteKindEnabled(ctx context.Context, kind string) bool {
	var count int64
	if err := model.DB(ctx).Model(&model.AgentProxyRoute{}).
		Where("kind = ? AND enabled = ?", kind, true).
		Count(&count).Error; err != nil {
		slog.Warn("agentProxyRouteKindEnabled: 查询 proxy route 失败", "kind", kind, "error", err)
		return false
	}
	return count > 0
}

// siteConfigRequiresRecommendedRules 判断当前 SiteConfig / 运行时状态是否启用了任何"推荐规则"相关的功能开关。
//
// 背景：config/clawpro_required_sg_rules.json 里的规则分两类：
//   - builtin（allow_internet / allow_ssh）：无条件必需规则，任何场景都要
//   - recommended：条件规则，由 SiteConfig 或运行时状态对应开关决定是否启用
//
// 本函数仅判断"是否有任何 recommended 规则组被启用"。用途：
// 在 import / update 路径判定"当前环境是否有必需规则要求"——非空时必须 merge（无视 auto_fix_rules 开关）。
//
// ⚠️ 维护提醒：新增 recommended 规则组（JSON 里加 condition）时，必须同步在这里加判断分支。
// 否则会出现"功能已启用但 import 不 merge 必需规则"的语义漏洞。
func siteConfigRequiresRecommendedRules(ctx context.Context) bool {
	cfg := model.GetSiteConfig(ctx)
	// gateway_ui_enable：需要 Enable=true、Port>0、且 addr_type 不是 private 才算真正启用。
	// addr_type=private 时用户走 VPC 内网通道访问 Gateway UI，不需要在 SG 上对公网放通端口；
	// 应当与 GatewayUIEnable=false 等价处理，避免必需规则被强制注入。
	if cfg.GatewayUIEnable && cfg.GatewayUIPort > 0 && cfg.GatewayUIAddrType != "private" {
		return true
	}
	// browser_vnc_enable
	if cfg.BrowserVNCEnable {
		return true
	}
	// agent_proxy_teams_enable
	if agentProxyRouteKindEnabled(ctx, model.AgentProxyRouteKindTeams) {
		return true
	}
	return false
}

// clawproRequiredRuleSet 从配置文件加载并返回 ClawPro 所需的安全组规则集合。
// 规则定义在 config/clawpro_required_sg_rules.json 中。
func clawproRequiredRuleSet() sgRuleSet {
	data := ClawproRequiredSGRulesJSON
	if len(data) == 0 {
		// 兜底：尝试从磁盘读取
		var err error
		data, err = os.ReadFile("config/clawpro_required_sg_rules.json")
		if err != nil {
			slog.Warn("读取 ClawPro 安全组规则配置文件失败", "error", err)
			return sgRuleSet{}
		}
	}
	var ruleSet sgRuleSet
	if err := json.Unmarshal(data, &ruleSet); err != nil {
		slog.Warn("解析 ClawPro 安全组规则配置 JSON 失败", "error", err)
		return sgRuleSet{}
	}
	return ruleSet
}

// resolveConditionalRules 根据 SiteConfig 的运行时状态处理配置文件中带条件的规则组：
// 1. 评估每个规则组的 condition 字段，移除条件不满足的规则组
// 2. 将规则中的动态占位符替换为实际值（如 {{GATEWAY_UI_PORT}}）
// 目前支持的条件：
//   - "gateway_ui_enable"：当 SiteConfig.GatewayUIEnable=true 且 GatewayUIPort>0 且
//     GatewayUIAddrType!="private" 时保留。私网模式下用户走 VPC 内网通道访问 Gateway UI，
//     不需要在 SG 上对公网放通端口，等价于该条件未启用。
//   - "browser_vnc_enable"：当 SiteConfig.BrowserVNCEnable 为 true 时保留
//   - "agent_proxy_teams_enable"：当前租户存在 enabled 的 Teams proxy route 时保留
func resolveConditionalRules(ctx context.Context, ruleSet *sgRuleSet) {
	config := model.GetSiteConfig(ctx)
	// 私网模式：addr_type=private 时整组剔除 allow_gateway_ui，等价于 enable=false
	gatewayUIEnabled := config.GatewayUIEnable &&
		config.GatewayUIPort > 0 &&
		config.GatewayUIAddrType != "private"
	var portStr string
	if gatewayUIEnabled {
		portStr = strconv.Itoa(config.GatewayUIPort)
	}
	teamsProxyEnabled := agentProxyRouteKindEnabled(ctx, model.AgentProxyRouteKindTeams)
	lineProxyEnabled := agentProxyRouteKindEnabled(ctx, model.AgentProxyRouteKindLine)

	for i := range ruleSet.Categories {
		var kept []sgRuleGroup
		for _, group := range ruleSet.Categories[i].RuleGroups {
			// 评估条件：如果 condition 非空且条件不满足，则跳过该规则组
			switch group.Condition {
			case "gateway_ui_enable":
				if !gatewayUIEnabled {
					continue
				}
				// 替换端口占位符
				for k := range group.Rules {
					if group.Rules[k].Port == "{{GATEWAY_UI_PORT}}" {
						group.Rules[k].Port = portStr
					}
				}
			case "browser_vnc_enable":
				if !config.BrowserVNCEnable {
					continue
				}
			case "agent_proxy_teams_enable":
				if !teamsProxyEnabled {
					continue
				}
			case "agent_proxy_line_enable":
				if !lineProxyEnabled {
					continue
				}
			case "":
				// 无条件，直接保留
			default:
				// 未知条件，跳过
				continue
			}
			group.Condition = "" // 清除条件标识，不暴露给前端
			kept = append(kept, group)
		}
		ruleSet.Categories[i].RuleGroups = kept
	}
}

// resolveVpcCidr 从 SiteConfig 的 VpcId 查询真实的 VPC CIDR 地址段。
// 如果未配置 VpcId 或查询失败，返回空字符串。
func resolveVpcCidr(ctx context.Context) string {
	var config model.SiteConfig
	if err := model.DB(ctx).First(&config).Error; err != nil {
		slog.Warn("resolveVpcCidr: 读取 SiteConfig 失败", "error", err)
		return ""
	}
	if config.VpcId == "" {
		return ""
	}
	vpcClient, err := newVpcClientForSGFn(ctx)
	if err != nil {
		slog.Warn("resolveVpcCidr: 创建 VPC 客户端失败", "error", err)
		return ""
	}
	descReq := vpc.NewDescribeVpcsRequest()
	descReq.VpcIds = common.StringPtrs([]string{config.VpcId})
	descResp, err := vpcClient.DescribeVpcs(descReq)
	if err != nil {
		slog.Warn("resolveVpcCidr: 查询 VPC CIDR 失败", "vpc_id", config.VpcId, "error", err)
		return ""
	}
	if descResp.Response != nil && len(descResp.Response.VpcSet) > 0 && descResp.Response.VpcSet[0].CidrBlock != nil {
		return *descResp.Response.VpcSet[0].CidrBlock
	}
	return ""
}

// replaceVpcCidrPlaceholder 遍历 ruleSet 中所有规则，将 {{VPC_CIDR}} 占位符替换为真实的 VPC CIDR。
// 如果 vpcCidr 为空，则保留占位符不替换。
func replaceVpcCidrPlaceholder(ruleSet *sgRuleSet, vpcCidr string) {
	if vpcCidr == "" {
		return
	}
	for i := range ruleSet.Categories {
		for j := range ruleSet.Categories[i].RuleGroups {
			for k := range ruleSet.Categories[i].RuleGroups[j].Rules {
				rule := &ruleSet.Categories[i].RuleGroups[j].Rules[k]
				if rule.CidrBlock == "{{VPC_CIDR}}" {
					rule.CidrBlock = vpcCidr
				}
			}
		}
	}
}

// sgPolicyMatchesRule 判断一条云上安全组策略是否满足指定的所需规则。
// 注意：腾讯云 API 返回的 Protocol/Action 字段大小写不固定（例如用户在控制台创建的规则
// Protocol 可能返回小写 "tcp"，而 API 创建的返回大写 "TCP"），因此使用大小写不敏感比较。
func sgPolicyMatchesRule(policy *vpc.SecurityGroupPolicy, rule requiredSGRule) bool {
	if policy == nil {
		return false
	}
	// 协议匹配（大小写不敏感）
	if policy.Protocol != nil && !strings.EqualFold(*policy.Protocol, rule.Protocol) {
		return false
	}
	// 端口匹配
	if policy.Port != nil && *policy.Port != rule.Port {
		return false
	}
	// 动作匹配（大小写不敏感）
	if policy.Action != nil && !strings.EqualFold(*policy.Action, rule.Action) {
		return false
	}
	// CIDR 匹配（IPv4 或 IPv6）
	if rule.CidrBlock != "" {
		if policy.CidrBlock == nil || *policy.CidrBlock != rule.CidrBlock {
			return false
		}
	}
	if rule.Ipv6Cidr != "" {
		if policy.Ipv6CidrBlock == nil || *policy.Ipv6CidrBlock != rule.Ipv6Cidr {
			return false
		}
	}
	return true
}

// checkMissingSGRules 检查指定安全组中缺失的 ClawPro 所需规则。
// 仅检查 recommended 分类的规则，不检查 builtin 规则。
// 返回缺失的规则列表；如果全部满足则返回空切片。
func checkMissingSGRules(ctx context.Context, vpcClient sgPolicyQuerier, sgId string) ([]requiredSGRule, error) {
	slog.Info("开始检查安全组缺失规则", "security_group_id", sgId)
	req := vpc.NewDescribeSecurityGroupPoliciesRequest()
	req.SecurityGroupId = common.StringPtr(sgId)
	resp, err := vpcClient.DescribeSecurityGroupPolicies(req)
	if err != nil {
		slog.Error("检查安全组缺失规则：查询安全组规则失败", "security_group_id", sgId, "error", err)
		return nil, hcommon.I18nRichError(err, i18n.MsgQuerySGRulesFailed)
	}

	// 将 {{VPC_CIDR}} 占位符替换为真实的 VPC CIDR 后再进行规则匹配
	ruleSet := clawproRequiredRuleSet()
	if vpcCidr := resolveVpcCidr(ctx); vpcCidr != "" {
		replaceVpcCidrPlaceholder(&ruleSet, vpcCidr)
	}
	// 根据运行时条件过滤规则组并替换端口占位符
	resolveConditionalRules(ctx, &ruleSet)
	var required []requiredSGRule
	for _, category := range ruleSet.Categories {
		// 只检查 recommended 分类的规则，跳过 builtin 等其他分类
		if category.Type != "recommended" {
			continue
		}
		for _, group := range category.RuleGroups {
			required = append(required, group.Rules...)
		}
	}
	slog.Info("加载所需规则完成", "security_group_id", sgId, "required_count", len(required))
	var missingRules []requiredSGRule

	for _, rule := range required {
		matched := false
		if resp.Response != nil && resp.Response.SecurityGroupPolicySet != nil {
			policySet := resp.Response.SecurityGroupPolicySet
			var policies []*vpc.SecurityGroupPolicy
			if rule.Direction == "ingress" {
				policies = policySet.Ingress
			} else {
				policies = policySet.Egress
			}
			for _, p := range policies {
				if sgPolicyMatchesRule(p, rule) {
					matched = true
					break
				}
			}
		}
		if !matched {
			missingRules = append(missingRules, rule)
		}
	}

	if missingRules == nil {
		missingRules = []requiredSGRule{}
	}
	slog.Info("安全组缺失规则检查完成", "security_group_id", sgId, "missing_count", len(missingRules))
	return missingRules, nil
}

// checkMissingRecommendedSGRules 仅检查 recommended 分类中缺失的安全组规则。
// 与 checkMissingSGRules 不同，该函数只关注推荐规则，不检查 builtin 规则。
func checkMissingRecommendedSGRules(ctx context.Context, vpcClient sgPolicyQuerier, sgId string) ([]requiredSGRule, error) {
	slog.Info("开始检查安全组推荐规则", "security_group_id", sgId)
	req := vpc.NewDescribeSecurityGroupPoliciesRequest()
	req.SecurityGroupId = common.StringPtr(sgId)
	resp, err := vpcClient.DescribeSecurityGroupPolicies(req)
	if err != nil {
		slog.Error("检查安全组推荐规则：查询安全组规则失败", "security_group_id", sgId, "error", err)
		return nil, hcommon.I18nRichError(err, i18n.MsgQuerySGRulesFailed)
	}

	// 只提取 recommended 分类的规则
	ruleSet := clawproRequiredRuleSet()
	if vpcCidr := resolveVpcCidr(ctx); vpcCidr != "" {
		replaceVpcCidrPlaceholder(&ruleSet, vpcCidr)
	}
	resolveConditionalRules(ctx, &ruleSet)
	var required []requiredSGRule
	for _, category := range ruleSet.Categories {
		if category.Type != "recommended" {
			continue
		}
		for _, group := range category.RuleGroups {
			required = append(required, group.Rules...)
		}
	}
	slog.Info("加载推荐规则完成", "security_group_id", sgId, "required_count", len(required))

	var missingRules []requiredSGRule
	for _, rule := range required {
		matched := false
		if resp.Response != nil && resp.Response.SecurityGroupPolicySet != nil {
			policySet := resp.Response.SecurityGroupPolicySet
			var policies []*vpc.SecurityGroupPolicy
			if rule.Direction == "ingress" {
				policies = policySet.Ingress
			} else {
				policies = policySet.Egress
			}
			for _, p := range policies {
				if sgPolicyMatchesRule(p, rule) {
					matched = true
					break
				}
			}
		}
		if !matched {
			missingRules = append(missingRules, rule)
		}
	}

	if missingRules == nil {
		missingRules = []requiredSGRule{}
	}
	slog.Info("安全组推荐规则检查完成", "security_group_id", sgId, "missing_count", len(missingRules))
	return missingRules, nil
}

// addMissingSGRules 将缺失的规则批量添加到指定安全组。
// 返回实际添加的规则数量和可能的错误。
func addMissingSGRules(vpcClient sgPolicyWriter, sgId string, missingRules []requiredSGRule) (int, error) {
	if len(missingRules) == 0 {
		slog.Info("无需补齐安全组规则", "security_group_id", sgId)
		return 0, nil
	}

	slog.Info("开始补齐缺失安全组规则", "security_group_id", sgId, "missing_count", len(missingRules))
	var ingressPolicies []*vpc.SecurityGroupPolicy
	var egressPolicies []*vpc.SecurityGroupPolicy

	for _, rule := range missingRules {
		policy := &vpc.SecurityGroupPolicy{
			Protocol:          common.StringPtr(rule.Protocol),
			Port:              common.StringPtr(rule.Port),
			Action:            common.StringPtr(rule.Action),
			PolicyDescription: common.StringPtr(rule.Description),
		}
		if rule.CidrBlock != "" {
			policy.CidrBlock = common.StringPtr(rule.CidrBlock)
		}
		if rule.Ipv6Cidr != "" {
			policy.Ipv6CidrBlock = common.StringPtr(rule.Ipv6Cidr)
		}
		if rule.Direction == "ingress" {
			ingressPolicies = append(ingressPolicies, policy)
		} else {
			egressPolicies = append(egressPolicies, policy)
		}
	}

	// 腾讯云不支持同时传入 Ingress 和 Egress，需分开调用
	const maxRetry = 3

	if len(ingressPolicies) > 0 {
		ingressReq := vpc.NewCreateSecurityGroupPoliciesRequest()
		ingressReq.SecurityGroupId = common.StringPtr(sgId)
		ingressReq.SecurityGroupPolicySet = &vpc.SecurityGroupPolicySet{
			Ingress: ingressPolicies,
		}
		var policyErr error
		for i := range maxRetry {
			if _, policyErr = vpcClient.CreateSecurityGroupPolicies(ingressReq); policyErr == nil {
				break
			}
			slog.Warn("添加缺失入站安全组规则失败，准备重试", "security_group_id", sgId, "attempt", i+1, "error", policyErr)
			if i < maxRetry-1 {
				time.Sleep(addMissingSGRulesRetryInterval)
			}
		}
		if policyErr != nil {
			slog.Error("添加缺失入站安全组规则失败，已达最大重试次数", "security_group_id", sgId, "error", policyErr)
			return 0, hcommon.I18nRichError(policyErr, i18n.MsgAddIngressSGRulesFailed, maxRetry)
		}
		slog.Info("补齐缺失入站安全组规则完成", "security_group_id", sgId, "ingress_count", len(ingressPolicies))
	}

	if len(egressPolicies) > 0 {
		egressReq := vpc.NewCreateSecurityGroupPoliciesRequest()
		egressReq.SecurityGroupId = common.StringPtr(sgId)
		egressReq.SecurityGroupPolicySet = &vpc.SecurityGroupPolicySet{
			Egress: egressPolicies,
		}
		var policyErr error
		for i := range maxRetry {
			if _, policyErr = vpcClient.CreateSecurityGroupPolicies(egressReq); policyErr == nil {
				break
			}
			slog.Warn("添加缺失出站安全组规则失败，准备重试", "security_group_id", sgId, "attempt", i+1, "error", policyErr)
			if i < maxRetry-1 {
				time.Sleep(addMissingSGRulesRetryInterval)
			}
		}
		if policyErr != nil {
			slog.Error("添加缺失出站安全组规则失败，已达最大重试次数", "security_group_id", sgId, "error", policyErr)
			return 0, hcommon.I18nRichError(policyErr, i18n.MsgAddEgressSGRulesFailed, maxRetry)
		}
		slog.Info("补齐缺失出站安全组规则完成", "security_group_id", sgId, "egress_count", len(egressPolicies))
	}

	slog.Info("补齐缺失安全组规则全部完成", "security_group_id", sgId,
		"ingress_count", len(ingressPolicies), "egress_count", len(egressPolicies))
	return len(missingRules), nil
}

// HandleGetRequiredSGRules 查询内部配置的 ClawPro 所需安全组规则列表。
// GET /admin/config/security-group/required-rules?type=builtin
// 查询参数：
//   - type: 规则分类类型，可选值：builtin（默认）、recommended、all（返回所有分类）
func HandleGetRequiredSGRules(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := Logger(ctx)
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}
	ruleSet := clawproRequiredRuleSet()

	// 将 {{VPC_CIDR}} 占位符替换为真实的 VPC CIDR
	if vpcCidr := resolveVpcCidr(r.Context()); vpcCidr != "" {
		replaceVpcCidrPlaceholder(&ruleSet, vpcCidr)
	}
	// 根据运行时条件过滤规则组并替换端口占位符
	resolveConditionalRules(ctx, &ruleSet)

	// 根据 type 参数过滤分类
	typeParam := r.URL.Query().Get("type")
	if typeParam == "" {
		typeParam = "builtin"
	}

	if typeParam != "all" {
		var filtered []sgRuleCategory
		for _, cat := range ruleSet.Categories {
			if cat.Type == typeParam {
				filtered = append(filtered, cat)
			}
		}
		ruleSet.Categories = filtered
	}

	log.Info("查询所需安全组规则", "type", typeParam, "categories_count", len(ruleSet.Categories))
	jsonOK(w, ruleSet)
}

// HandleCheckSecurityGroupRules 查询指定云端安全组的实际规则，对比 ClawPro 推荐规则，返回缺失列表。
// GET /admin/config/security-group/check-rules?security_group_id=sg-xxx
func HandleCheckSecurityGroupRules(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := Logger(ctx)
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	sgID := r.URL.Query().Get("security_group_id")
	if sgID == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSGMissingSecurityGroupIDParam))
		return
	}

	vpcClient, err := newVpcClientForSGFn(ctx)
	if err != nil {
		log.Error("检查规则：创建 VPC 客户端失败", "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgCreateVPCClientFailed))
		return
	}

	missingRules, err := checkMissingSGRules(ctx, vpcClient, sgID)
	if err != nil {
		log.Error("检查规则：查询云端安全组规则失败", "security_group_id", sgID, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSGCheckRulesFailed))
		return
	}

	log.Info("云端安全组规则校验完成", "security_group_id", sgID, "missing_count", len(missingRules))
	jsonOK(w, map[string]interface{}{
		"missing_rules": missingRules,
	})
}

// rebindAllInstancesToSingleSG 将所有指定实例的安全组替换为仅包含 targetSgId，
// 解绑实例之前绑定的所有其他安全组。
// 流程：先查询每个实例当前绑定的安全组 → 绑定新安全组 → 逐个解绑旧安全组，
// 确保实例在任何时刻都至少有一个安全组保护。
func rebindAllInstancesToSingleSG(ctx context.Context, instanceIds []string, targetSgId string) error {
	if len(instanceIds) == 0 {
		return nil
	}

	// 第一步：批量查询所有实例当前绑定的安全组
	instanceSgMap, err := describeInstancesSecurityGroupsFn(ctx, instanceIds)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgQueryInstanceSGListFailed)
	}

	cvmClient, err := newCVMClientFn(ctx)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgCreateCVMClientFailed)
	}

	var lastErr error
	successCount := 0

	for _, instanceId := range instanceIds {
		currentSgs := instanceSgMap[instanceId]

		// 检查是否已经只绑定了目标安全组，无需操作
		if len(currentSgs) == 1 && currentSgs[0] == targetSgId {
			successCount++
			continue
		}

		// 检查目标安全组是否已绑定
		alreadyBound := false
		for _, sg := range currentSgs {
			if sg == targetSgId {
				alreadyBound = true
				break
			}
		}

		// 第二步：先绑定新安全组，确保实例始终有安全组保护
		if !alreadyBound {
			assocReq := cvm.NewAssociateSecurityGroupsRequest()
			assocReq.SecurityGroupIds = common.StringPtrs([]string{targetSgId})
			assocReq.InstanceIds = common.StringPtrs([]string{instanceId})
			if _, err := cvmClient.AssociateSecurityGroups(assocReq); err != nil {
				slog.Error("绑定新安全组失败，跳过该实例",
					"instance_id", instanceId, "target_sg", targetSgId, "error", err)
				lastErr = err
				continue
			}
		}

		// 第三步：逐个解绑所有旧安全组（排除目标安全组）
		for _, oldSg := range currentSgs {
			if oldSg == targetSgId {
				continue
			}
			disReq := cvm.NewDisassociateSecurityGroupsRequest()
			disReq.SecurityGroupIds = common.StringPtrs([]string{oldSg})
			disReq.InstanceIds = common.StringPtrs([]string{instanceId})
			if _, err := cvmClient.DisassociateSecurityGroups(disReq); err != nil {
				slog.Error("解绑旧安全组失败（新安全组已绑定，实例仍受保护）",
					"instance_id", instanceId, "old_sg", oldSg, "error", err)
				lastErr = err
			} else {
				slog.Info("已解绑旧安全组",
					"instance_id", instanceId, "old_sg", oldSg, "target_sg", targetSgId)
			}
		}
		successCount++
	}

	slog.Info("实例安全组换绑完成", "total", len(instanceIds), "success", successCount, "target_sg", targetSgId)
	return lastErr
}

// HandleReorderRuleSetRules POST /admin/config/security-group/ruleset/rules/reorder
//
// 用途：管理员在导入/创建规则后，按需调整出入站规则的匹配顺序（自上而下匹配，越靠前优先级越高）。
//
// 设计要点：
//  1. 不修改任何规则的内容（Direction/Protocol/Port/CidrBlock/Action/Description/IsRequired），
//     仅按 ordered_fingerprints 重排 RuleSet.Rules 数组顺序。
//  2. 必需规则可以参与排序（管理员可决定其位置），但不允许通过本接口删除——
//     未在 ordered_fingerprints 中列出的规则会按原相对顺序追加在末尾，因此"漏列"等价于"放最后"，不会丢规则。
//  3. 复用 UpdateRuleSetRulesInternal 的 2PC fan-out + 失败回滚 + version++ 语义，
//     不走任何独立的云端调用路径——这意味着 reorder 失败的回滚行为与普通保存完全一致。
//  4. autoFixRules=false：reorder 是"管理员明确控制顺序"的语义，不能在内部再被 MergeRequiredRules
//     合入新的 recommended 规则（否则可能引入用户根本不知道的新规则导致预期外效果）；
//     即便如此，如果 SiteConfig 已启用 recommended，UpdateRuleSetRulesInternal 内部仍会兜底 merge——
//     此时"未列出的必需规则"会被 MergeRequiredRules 按原序追加到末尾，符合本接口"漏列即放末尾"的语义。
//
// 请求体：
//
//	{
//	  "name": "default",                              // 可选，缺省走默认 RuleSet
//	  "ordered_fingerprints": ["fp1", "fp2", ...]     // 新顺序；fp 由 Rule.Fingerprint() 计算
//	}
//
// 响应：与 HandleUpdateRuleSetRules 一致（version / synced / drifted / drift_errors）。
//
// 错误：
//   - 400：name 不合法 / ordered_fingerprints 为空 / 出现未知 fingerprint / 重复 fingerprint
//   - 404：指定 name 的 RuleSet 不存在
//   - 409：fan-out 失败（已自动回滚，DB 不变）
func HandleReorderRuleSetRules(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}
	ctx := r.Context()
	log := Logger(ctx)

	var req struct {
		Name                string   `json:"name,omitempty"`
		OrderedFingerprints []string `json:"ordered_fingerprints"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgBadRequestWithError))
		return
	}
	name := strings.TrimSpace(req.Name)
	if name != "" && !isValidRuleSetName(name) {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgRuleSetNameInvalid, req.Name))
		return
	}
	if len(req.OrderedFingerprints) == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgReorderFingerprintsRequired))
		return
	}
	// 入参 fingerprint 内部去重检查（同一个 fp 出现两次属于明显误用）
	dupSet := make(map[string]struct{}, len(req.OrderedFingerprints))
	for _, fp := range req.OrderedFingerprints {
		fp = strings.TrimSpace(fp)
		if fp == "" {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgReorderFingerprintsEmptyStr))
			return
		}
		if _, dup := dupSet[fp]; dup {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgReorderDuplicateFingerprint, fp))
			return
		}
		dupSet[fp] = struct{}{}
	}

	// 1. 读取现有 RuleSet（容忍 name=""）
	var rs *model.RuleSet
	var err error
	if name == "" {
		rs, err = model.GetDefaultRuleSet(ctx)
	} else {
		rs, err = model.GetRuleSetByName(ctx, name)
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgRuleSetNotFound, req.Name))
			return
		}
		log.Error("[ruleset-reorder] get rule_set failed", "name", req.Name, "err", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgRuleSetQueryFailed))
		return
	}

	// 2. 反序列化现有 Rules
	var existing []Rule
	if rs.Rules != "" && rs.Rules != "[]" {
		if err := json.Unmarshal([]byte(rs.Rules), &existing); err != nil {
			log.Error("[ruleset-reorder] unmarshal existing rules failed", "rule_set_id", rs.ID, "err", err)
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgRuleSetParseRulesFailed))
			return
		}
	}
	if len(existing) == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgRuleSetEmptyNoReorder))
		return
	}

	// 3. 建立 fp → Rule 索引（同一 fp 出现两次属于历史数据异常，仅取首次）
	byFp := make(map[string]Rule, len(existing))
	for _, rule := range existing {
		fp := rule.Fingerprint()
		if _, dup := byFp[fp]; dup {
			continue
		}
		byFp[fp] = rule
	}

	// 4. 校验入参 fp 都能在现有规则中找到
	for _, fp := range req.OrderedFingerprints {
		if _, ok := byFp[strings.TrimSpace(fp)]; !ok {
			writeError(w, r, http.StatusBadRequest,
				hcommon.I18nError(i18n.MsgReorderFingerprintNotFound, fp))
			return
		}
	}

	// 5. 构造新顺序：先按 ordered_fingerprints 排，未列出的按原相对顺序追加末尾
	newRules := make([]Rule, 0, len(existing))
	picked := make(map[string]struct{}, len(req.OrderedFingerprints))
	for _, fp := range req.OrderedFingerprints {
		fp = strings.TrimSpace(fp)
		newRules = append(newRules, byFp[fp])
		picked[fp] = struct{}{}
	}
	for _, rule := range existing {
		fp := rule.Fingerprint()
		if _, done := picked[fp]; done {
			continue
		}
		picked[fp] = struct{}{}
		newRules = append(newRules, rule)
	}

	// 6. 复用 UpdateRuleSetRulesInternal 走 2PC fan-out（autoFixRules=false：不再额外注入 recommended）
	//    内部 MergeRequiredRules 的"保留用户原序"行为保证我们刚刚排好的顺序不被打乱；
	//    SiteConfig 兜底注入的 recommended 规则会按 MergeRequiredRules 语义追加在末尾。
	log.Info("[ruleset-reorder] start",
		"rule_set_id", rs.ID, "name", rs.Name,
		"old_version", rs.Version,
		"existing_count", len(existing),
		"input_fp_count", len(req.OrderedFingerprints),
		"new_rules_count", len(newRules))

	version, synced, driftErrs, rerr := UpdateRuleSetRulesInternal(ctx, rs.Name, newRules, false /* autoFixRules */)
	if rerr != nil {
		log.Error("[ruleset-reorder] update failed", "err", rerr, "drift_errors", len(driftErrs))
		if driftErrs == nil {
			driftErrs = []DriftError{}
		}
		writeError(w, r, http.StatusConflict, hcommon.I18nRichError(rerr, i18n.MsgReorderFailed).
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
	log.Info("[ruleset-reorder] success", "rule_set_id", rs.ID, "new_version", version, "synced", synced)
	jsonOK(w, map[string]any{
		"version":      version,
		"synced":       synced,
		"drifted":      len(driftErrs),
		"drift_errors": driftErrs,
	})
}
