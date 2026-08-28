package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	hcommon "hatchery/common"
	"hatchery/controller/usergroup"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
)

func HandleSetGatewayUi(w http.ResponseWriter, r *http.Request) {
	handleSetGatewayUi(w, r, defaultStatusResolver)
}

func handleSetGatewayUi(w http.ResponseWriter, r *http.Request, resolver instanceStatusResolver) {
	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		return
	}

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 校验 Gateway UI 是否开启及端口是否已分配（按 agent 绑定的分组策略）
	siteConfig := model.GetSiteConfig(r.Context())
	if !usergroup.ResolvePolicyBoolForGroup(r.Context(), usergroup.PolicyKeyGatewayUI, instance.GroupID, siteConfig.GatewayUIEnable) {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgGatewayUINotEnabled))
		return
	}
	if siteConfig.GatewayUIPort == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgGatewayUIPortNotConfigured))
		return
	}

	// 本地实例：Gateway UI 域名化基于 CVM 公网环境，本地 agent 不支持。
	if rejectLocalOrWrite(w, r, instance) {
		return
	}
	// 状态准入：仅 running 状态允许配置 Gateway UI
	if _, err := requireInstanceRunning(r.Context(), instance, resolver); err != nil {
		writeAgentGuardError(w, r, err)
		return
	}

	client, err := NewCVMClient(r.Context())
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgCreateCVMClientFailed))
		return
	}

	request := cvm.NewDescribeInstancesRequest()
	request.InstanceIds = common.StringPtrs([]string{instance.InstanceId})

	var response *cvm.DescribeInstancesResponse
	callErr := RetryCloudCall(r.Context(), func() error {
		var err error
		response, err = client.DescribeInstances(request)
		return err
	})
	if callErr != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(callErr, i18n.MsgQueryInstanceFailed))
		return
	}

	if response.Response == nil || len(response.Response.InstanceSet) == 0 {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgCVMInstanceNotFound))
		return
	}

	inst := response.Response.InstanceSet[0]
	networkType := strings.ToLower(r.FormValue("network_type")) // "public" or "private"，默认 public
	if networkType == "" {
		networkType = "public"
	}
	var gatewayIp string
	if networkType == "private" {
		// 强制使用私网 IP
		if len(inst.PrivateIpAddresses) > 0 && *inst.PrivateIpAddresses[0] != "" {
			gatewayIp = *inst.PrivateIpAddresses[0]
		} else {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgInstanceNoPrivateIP))
			return
		}
	} else {
		// 默认公网优先，无公网则降级到私网
		if len(inst.PublicIpAddresses) > 0 && *inst.PublicIpAddresses[0] != "" {
			gatewayIp = *inst.PublicIpAddresses[0]
		} else if len(inst.PrivateIpAddresses) > 0 && *inst.PrivateIpAddresses[0] != "" {
			gatewayIp = *inst.PrivateIpAddresses[0]
		} else {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgInstanceNoUsableIP))
			return
		}
	}

	params := map[string]string{
		"gateway_ip":      gatewayIp,
		"gateway_ui_port": strconv.Itoa(siteConfig.GatewayUIPort),
	}

	// 根据实例运行时类型分发到不同的脚本（兼容自定义类型按 compatible_with 解析）
	switch model.GetAgentRuntimeType(r.Context(), instance.AgentType) {
	case model.AgentTypeLightclawACE:
		setLightclawUI(w, r, instance, gatewayIp, params)
	case model.AgentTypeHermes:
		setHermesUI(w, r, instance, gatewayIp, params)
	case model.AgentTypeOpenClaw:
		setOpenClawGatewayUI(w, r, user, siteConfig, instance, gatewayIp, params)
	default:
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgAgentTypeNotSupportWebUI, model.GetAgentTypeDisplayName(r.Context(), instance.AgentType)))
		return
	}
}

// setOpenClawGatewayUI 处理 OpenClaw 类型实例的 Gateway UI 配置。
func setOpenClawGatewayUI(w http.ResponseWriter, r *http.Request, user *model.User, siteConfig model.SiteConfig, instance *model.Instance, gatewayIp string, params map[string]string) {
	output, err := RunScript(r.Context(), instance.InstanceId, "set_gateway_ui.sh", 120, instance.RuntimeUser, nil, params)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	var scriptResult struct {
		Port      int    `json:"port"`
		BasePath  string `json:"basePath"`
		AuthToken string `json:"authToken"`
	}
	if err := json.Unmarshal([]byte(output), &scriptResult); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgParseScriptOutputFailed))
		return
	}

	if scriptResult.Port == 0 || scriptResult.AuthToken == "" {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgScriptReturnIncomplete))
		return
	}

	basePath := strings.TrimRight(scriptResult.BasePath, "/")
	primaryURL := fmt.Sprintf("http://%s:%d%s?token=%s", gatewayIp, scriptResult.Port, basePath, scriptResult.AuthToken)

	// 软功能：命中新模式时尝试用域名 URL 覆盖 primary；失败/关闭一律原样返回
	finalURL := MaybeOverrideWithGateway(r, siteConfig, user, instance,
		gatewayIp, scriptResult.Port, scriptResult.AuthToken, basePath, primaryURL)

	jsonOK(w, map[string]string{
		"gatewayUI": finalURL,
		"token":     scriptResult.AuthToken,
	})
}

// setLightclawUI 处理 LightclawACE 类型实例的 WebUI 配置。
// 通过 TAT 执行 set_lightclaw_ui.sh 重启 lightclaw 并绑定到指定端口，
// 从 lightclaw-login.txt 读取 password，返回统一的 {gatewayUI, token} 格式。
//
// 注意：LightclawACE 不接入云 API 网关（AgentTypeSupportsAPIGateway=false），
// 其 WebUI 走实例公网端口 + Basic Auth（password），与网关 AgentToken 透传契约不匹配。
func setLightclawUI(w http.ResponseWriter, r *http.Request, instance *model.Instance, gatewayIp string, params map[string]string) {
	output, err := RunScript(r.Context(), instance.InstanceId, "set_lightclaw_ui.sh", 120, instance.RuntimeUser, nil, params)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	var scriptResult struct {
		Port     int    `json:"port"`
		Password string `json:"password"`
	}
	if err := json.Unmarshal([]byte(output), &scriptResult); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgParseScriptOutputFailed))
		return
	}

	if scriptResult.Port == 0 || scriptResult.Password == "" {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgScriptReturnIncomplete))
		return
	}

	gatewayUI := fmt.Sprintf("http://%s:%d", gatewayIp, scriptResult.Port)

	jsonOK(w, map[string]string{
		"gatewayUI": gatewayUI,
		"token":     scriptResult.Password,
	})
}

// setHermesUI 通过 TAT 执行 set_hermes_ui.sh 启动 hermes dashboard。
// Hermes 不接入云 API 网关，WebUI 走实例公网端口。
// 不做后端 HTTP 探测（Hatchery 与 CVM 可能不在同一网络）。
// TAT timeout 210s = 脚本 MAX_WAIT(180s) + 余量(30s)，对齐脚本内的自适应等待策略。
func setHermesUI(w http.ResponseWriter, r *http.Request, instance *model.Instance, gatewayIp string, params map[string]string) {
	_, err := RunScript(r.Context(), instance.InstanceId, "set_hermes_ui.sh", 210, instance.RuntimeUser, nil, params)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	port := params["gateway_ui_port"]
	targetURL := fmt.Sprintf("http://%s:%s/", gatewayIp, port)

	jsonOK(w, map[string]string{
		"gatewayUI": targetURL,
		"token":     "",
	})
}

// HandleCheckGatewayAccess 独立接口：检查实例绑定的安全组入站规则，确认面板端口是否可访问。
// 通过 DescribeInstances 获取实例实际绑定的安全组列表，逐个检查入站规则。
func HandleCheckGatewayAccess(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		return
	}

	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	// 获取用户实例（提前获取，用于分组策略解析）
	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 校验 Gateway UI 是否开启（按 agent 绑定的分组策略）
	siteConfig := model.GetSiteConfig(r.Context())
	if !usergroup.ResolvePolicyBoolForGroup(r.Context(), usergroup.PolicyKeyGatewayUI, instance.GroupID, siteConfig.GatewayUIEnable) {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgGatewayUINotEnabled))
		return
	}
	if siteConfig.GatewayUIPort == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgGatewayUIPortNotConfigured))
		return
	}

	if instance.InstanceId == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInstanceNoCVM))
		return
	}

	// 校验实例运行时类型是否支持 WebUI（兼容自定义类型按 compatible_with 解析）
	switch model.GetAgentRuntimeType(r.Context(), instance.AgentType) {
	case model.AgentTypeOpenClaw, model.AgentTypeLightclawACE, model.AgentTypeHermes:
		// 支持
	default:
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgAgentTypeNotSupportWebUI, model.GetAgentTypeDisplayName(r.Context(), instance.AgentType)))
		return
	}

	// ── 新模型：端口放通查询通过 RuleSet helper 间接完成 ──────────────────
	// 之前的实现校验 siteConfig.SecurityGroupId 是否绑在实例上，迁移到 RuleSet +
	// ManagedSGPool 多 SG 模型后该字段已指向 FROZEN 的老 base SG，校验必然失败
	// 并报错 "当前配置的安全组 sg-xxx 未绑定在实例上"。
	// 现改为按实例绑的 SG 反查所属 RuleSet，drift-aware 判断端口是否放通。
	//
	// securityGroupIds 字段保留（公共 API 稳定契约要求不能移除），值为实例当前绑的 SG。
	sgIDsOut := []string{}
	if sgid := strings.TrimSpace(instance.SecurityGroupId); sgid != "" {
		sgIDsOut = append(sgIDsOut, sgid)
	}
	if len(sgIDsOut) == 0 {
		jsonOK(w, map[string]interface{}{
			"accessible":       false,
			"port":             siteConfig.GatewayUIPort,
			"securityGroupIds": sgIDsOut,
			"drifting":         false,
			"message":          i18n.T(r.Context(), i18n.MsgInstanceNoSecurityGroupBind),
		})
		return
	}

	allowed, drifting, checkErr := checkPortRuleOnInstanceSG(r.Context(), instance, siteConfig.GatewayUIPort, "TCP", portRuleCheckOptions{sourceIP: ExtractClientIP(r)})
	if checkErr != nil {
		if errors.Is(checkErr, ErrSGBootstrapNotDone) {
			// 全新租户：把 500 降级成 200 + 友好 message，避免开通体验差
			jsonOK(w, map[string]interface{}{
				"accessible":       false,
				"port":             siteConfig.GatewayUIPort,
				"securityGroupIds": sgIDsOut,
				"drifting":         false,
				"message":          checkErr.Error(),
			})
			return
		}
		slog.Warn("检查端口放通失败", "instance_id", instance.InstanceId, "sg_id", instance.SecurityGroupId, "err", checkErr)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(checkErr, i18n.MsgCheckSGIngressFailed))
		return
	}

	if allowed {
		msg := i18n.T(r.Context(), i18n.MsgPanelPortAccessible)
		if drifting {
			msg = i18n.T(r.Context(), i18n.MsgPanelPortAccessibleDrifting)
		}
		jsonOK(w, map[string]interface{}{
			"accessible":       true,
			"port":             siteConfig.GatewayUIPort,
			"securityGroupIds": sgIDsOut,
			"drifting":         drifting,
			"message":          msg,
		})
		return
	}

	msg := i18n.T(r.Context(), i18n.MsgPanelPortNotOpen, siteConfig.GatewayUIPort)
	if drifting {
		msg = i18n.T(r.Context(), i18n.MsgPanelPortNotOpenDrifting, siteConfig.GatewayUIPort)
	}
	jsonOK(w, map[string]interface{}{
		"accessible":       false,
		"port":             siteConfig.GatewayUIPort,
		"securityGroupIds": sgIDsOut,
		"drifting":         drifting,
		"message":          msg,
	})
}

// checkSecurityGroupIngressForPort 检查安全组入站规则中是否允许指定端口的访问。
// 严格按照腾讯云安全组规则的优先级顺序（从上到下）匹配，遇到第一条匹配目标端口的规则时，
// 根据其 Action 决定返回结果，不再继续往下匹配。
//
// 返回值:
//   - true:  第一条匹配到目标端口的规则为 ACCEPT
//   - false: 入站规则为空（全关闭）、第一条匹配到目标端口的规则为 DROP/REJECT、或无任何规则匹配目标端口
func checkSecurityGroupIngressForPort(ctx context.Context, securityGroupId string, port int, opts ...portRuleCheckOptions) (bool, error) {
	var opt portRuleCheckOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	sourceIP := strings.TrimSpace(opt.sourceIP)
	vpcClient, err := newVpcClient(ctx)
	if err != nil {
		return false, hcommon.I18nRichError(err, i18n.MsgCreateVPCClientFailed)
	}

	req := vpc.NewDescribeSecurityGroupPoliciesRequest()
	req.SecurityGroupId = common.StringPtr(securityGroupId)
	resp, err := vpcClient.DescribeSecurityGroupPolicies(req)
	if err != nil {
		return false, hcommon.I18nRichError(err, i18n.MsgQuerySGRulesFailed)
	}

	if resp.Response == nil || resp.Response.SecurityGroupPolicySet == nil {
		return false, nil // 无规则集，视为全关闭
	}

	ingress := resp.Response.SecurityGroupPolicySet.Ingress
	if len(ingress) == 0 {
		return false, nil // 无入站规则，视为全关闭
	}

	// 严格按规则顺序匹配，腾讯云安全组规则从上到下依次匹配，命中即停止
	for _, policy := range ingress {
		if policy.Action == nil {
			continue
		}
		matched := policyMatchesPortProto(policy, port, "TCP")

		if matched && strings.TrimSpace(sourceIP) != "" && !policyMatchesSource(policy, sourceIP) {
			continue
		}

		// 如果当前规则匹配了目标端口，根据 Action 决定结果
		if matched {
			action := strings.ToUpper(*policy.Action)
			if action == "ACCEPT" {
				return true, nil
			}
			// DROP 或其他拒绝动作，直接返回 false（优先级更高的规则已拒绝）
			return false, nil
		}
	}

	// 遍历完所有入站规则，无任何规则匹配目标端口
	return false, nil
}

func policyMatchesPortProto(policy *vpc.SecurityGroupPolicy, port int, proto string) bool {
	if policy == nil {
		return false
	}
	protocol := ""
	if policy.Protocol != nil {
		protocol = strings.ToUpper(*policy.Protocol)
	}
	policyPort := ""
	if policy.Port != nil {
		policyPort = strings.ToUpper(*policy.Port)
	}
	if protocol == "ALL" && policyPort == "ALL" {
		return true
	}
	targetProto := strings.ToUpper(strings.TrimSpace(proto))
	if protocol != "ALL" && protocol != targetProto {
		return false
	}
	return hcommon.PortMatchesRule(policyPort, port)
}

func policyMatchesSource(policy *vpc.SecurityGroupPolicy, sourceIP string) bool {
	if policy == nil {
		return false
	}
	if policy.CidrBlock != nil && cidrMatchesSource(*policy.CidrBlock, sourceIP) {
		return true
	}
	if policy.Ipv6CidrBlock != nil && cidrMatchesSource(*policy.Ipv6CidrBlock, sourceIP) {
		return true
	}
	return false
}
