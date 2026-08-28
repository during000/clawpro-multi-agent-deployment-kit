package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
)

// 可替换的函数变量，便于单元测试 mock
var (
	fetchPrivateIPFn      = fetchPrivateIP
	getOpenClawWSInfoFn   = getOpenClawWSInfo
	getHermesAPIInfoFn    = getHermesAPIInfo
	checkWSPortAccessFn   = checkWSPortAccessible
	wsUrlRunScriptFn      = RunScript
	wsUrlCheckSGIngressFn = func(ctx context.Context, sg string, port int) (bool, error) {
		return checkSecurityGroupIngressForPort(ctx, sg, port)
	}
)

// HandleGetWSUrl 返回指定实例的内网连接 URL。
// POST /openclaw/ws-url
//
// SDK 用户通过 Bearer Token 鉴权，传入 CVM 实例 ID，获取内网连接地址。
// 支持 OpenClaw（WebSocket）和 Hermes（HTTP SSE）两种 agent 类型，按类型分发不同的初始化逻辑。
func HandleGetWSUrl(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	log := Logger(r.Context())

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	// 参数校验：从 JSON body 读取 instance_id（CVM 实例 ID，ins-xxxxxxxx 格式）
	var req struct {
		InstanceID string `json:"instance_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidJSON))
		return
	}
	instanceId := strings.TrimSpace(req.InstanceID)
	if instanceId == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestMissingParamWithKey, "instance_id"))
		return
	}
	if !strings.HasPrefix(instanceId, "ins-") {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidInstanceIDFormat))
		return
	}

	// 通过 CVM instance_id 查询实例并校验归属
	var instance model.Instance
	if err := model.DB(r.Context()).Where("instance_id = ? AND user_id = ?", instanceId, user.ID).First(&instance).Error; err != nil {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgInstanceNotFoundOrNoPerm))
		return
	}
	if rejectLocalOrWrite(w, r, &instance) {
		return
	}
	w = WrapInstanceId(w, instance.InstanceId)

	// 校验实例状态（LastStableState 存储的是 CVM 大写状态如 "RUNNING"）
	if instance.LastStableState != "RUNNING" {
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgInstanceNotRunningForWS, instance.LastStableState))
		return
	}

	// 查询 CVM 获取内网 IP
	privateIP, cvmInst, err := fetchPrivateIPFn(r.Context(), instance.InstanceId)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 按 agent 类型分发
	ctx := r.Context()

	switch instance.AgentType {
	case model.AgentTypeOpenClaw, "":
		// 加载站点配置，校验 GatewayUIPort 已分配
		siteConfig := model.GetSiteConfig(ctx)
		log.Info("[WSUrl] 站点配置端口", "gateway_ui_port", siteConfig.GatewayUIPort)
		if siteConfig.GatewayUIPort == 0 {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgGatewayUIPortNotAllocated))
			return
		}

		port, token, basePath, ocErr := getOpenClawWSInfoFn(ctx, &instance, siteConfig.GatewayUIPort)
		if ocErr != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(ocErr))
			return
		}

		// 检查安全组是否放通端口的内网入站（仅日志告警，不阻断返回）
		if sgErr := checkWSPortAccessFn(ctx, cvmInst, port); sgErr != nil {
			log.Error("[WSUrl] 安全组端口放通检查异常",
				"instance_id", instance.InstanceId, "port", port, "error", sgErr)
		}

		wsURL := fmt.Sprintf("ws://%s:%d/ws?token=%s", privateIP, port, token)
		jsonOK(w, map[string]interface{}{
			"url":      wsURL,
			"protocol": "websocket",
			"token":    token,
			"basePath": basePath,
		})

	case model.AgentTypeHermes:
		port, token, hErr := getHermesAPIInfoFn(ctx, &instance)
		if hErr != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(hErr))
			return
		}

		// 检查安全组是否放通端口的内网入站（仅日志告警，不阻断返回）
		if sgErr := checkWSPortAccessFn(ctx, cvmInst, port); sgErr != nil {
			log.Error("[WSUrl] 安全组端口放通检查异常",
				"instance_id", instance.InstanceId, "port", port, "error", sgErr)
		}

		apiURL := fmt.Sprintf("http://%s:%d", privateIP, port)
		jsonOK(w, map[string]interface{}{
			"url":      apiURL,
			"protocol": "sse",
			"token":    token,
		})

	default:
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgAgentTypeNotSupportWS, model.GetAgentTypeDisplayName(r.Context(), instance.AgentType)))
		return
	}
}

// fetchPrivateIP 查询 CVM 实例的内网 IP，同时返回 CVM 实例信息（供后续安全组检查使用）。
func fetchPrivateIP(ctx context.Context, cvmInstanceId string) (string, *cvm.Instance, error) {
	client, cvmErr := GetCVMClient(ctx)
	if cvmErr != nil {
		return "", nil, hcommon.I18nRichError(cvmErr, i18n.MsgCreateCVMClientFailed)
	}

	request := cvm.NewDescribeInstancesRequest()
	request.InstanceIds = common.StringPtrs([]string{cvmInstanceId})

	response, descErr := client.DescribeInstances(request)
	if descErr != nil {
		return "", nil, hcommon.I18nRichError(descErr, i18n.MsgQueryInstanceFailed)
	}

	if response.Response == nil || len(response.Response.InstanceSet) == 0 {
		return "", nil, hcommon.I18nError(i18n.MsgCVMInstanceNotFound)
	}

	inst := response.Response.InstanceSet[0]
	if len(inst.PrivateIpAddresses) == 0 || inst.PrivateIpAddresses[0] == nil || *inst.PrivateIpAddresses[0] == "" {
		return "", nil, hcommon.I18nError(i18n.MsgInstanceNoPrivateIP)
	}

	return *inst.PrivateIpAddresses[0], inst, nil
}

// checkWSPortAccessible 检查 CVM 安全组是否放通指定端口的内网入站规则。
func checkWSPortAccessible(ctx context.Context, inst *cvm.Instance, port int) error {
	log := Logger(ctx)
	sgIds := make([]string, 0, len(inst.SecurityGroupIds))
	for _, sg := range inst.SecurityGroupIds {
		if sg != nil {
			sgIds = append(sgIds, *sg)
		}
	}
	if len(sgIds) == 0 {
		return hcommon.I18nError(i18n.MsgInstanceNoSecurityGroup, port)
	}

	for _, sgId := range sgIds {
		ok, err := wsUrlCheckSGIngressFn(ctx, sgId, port)
		if err != nil {
			log.Warn("[WSUrl] 检查安全组入站规则失败", "sg_id", sgId, "port", port, "error", err)
			continue
		}
		if ok {
			return nil
		}
	}
	return hcommon.I18nError(i18n.MsgSGPortNotOpen, port)
}

// ========== OpenClaw ==========

// getOpenClawWSInfo 通过 TAT 脚本确保 OpenClaw gateway 对内网可访问，并读取 port、authToken 和 basePath。
func getOpenClawWSInfo(ctx context.Context, instance *model.Instance, gatewayUIPort int) (port int, token string, basePath string, err error) {
	log := Logger(ctx)
	log.Info("[WSUrl] getOpenClawWSInfo 开始", "instance_id", instance.InstanceId, "gateway_ui_port_param", gatewayUIPort)

	params := map[string]string{
		"gateway_ui_port": strconv.Itoa(gatewayUIPort),
	}
	output, runErr := wsUrlRunScriptFn(ctx, instance.InstanceId, "get_ws_url.sh", 60, instance.RuntimeUser, nil, params)
	if runErr != nil {
		log.Error("[WSUrl] OpenClaw TAT 脚本执行失败", "instance_id", instance.InstanceId, "error", runErr)
		return 0, "", "", hcommon.I18nRichError(runErr, i18n.MsgGetWSConnInfoFailed)
	}

	var result struct {
		Port      int    `json:"port"`
		AuthToken string `json:"authToken"`
		BasePath  string `json:"basePath"`
		Error     string `json:"error,omitempty"`
	}
	if parseErr := json.Unmarshal([]byte(output), &result); parseErr != nil {
		log.Error("[WSUrl] 解析脚本输出失败", "instance_id", instance.InstanceId, "output", output, "error", parseErr)
		return 0, "", "", hcommon.I18nRichError(parseErr, i18n.MsgParseWSConnInfoFailed)
	}

	log.Info("[WSUrl] 脚本返回结果", "instance_id", instance.InstanceId, "script_port", result.Port, "expected_port", gatewayUIPort, "basePath", result.BasePath)

	if result.Error != "" {
		return 0, "", "", hcommon.I18nError(i18n.MsgInstanceReturnedError, result.Error)
	}
	if result.Port == 0 || result.AuthToken == "" {
		return 0, "", "", hcommon.I18nError(i18n.MsgGatewayConfigIncomplete)
	}

	return result.Port, result.AuthToken, result.BasePath, nil
}

// ========== Hermes ==========

// getHermesAPIInfo 通过 TAT 脚本确保 Hermes API Server 已启动，返回端口和 API Key。
// Hermes API Server 提供 OpenAI 兼容的 HTTP 端点（支持 SSE 流式），
// SDK 通过 POST /v1/chat/completions (stream=true) 实现双向对话。
func getHermesAPIInfo(ctx context.Context, instance *model.Instance) (port int, token string, err error) {
	log := Logger(ctx)

	output, runErr := wsUrlRunScriptFn(ctx, instance.InstanceId, "get_hermes_api.sh", 60, instance.RuntimeUser, nil, nil)
	if runErr != nil {
		log.Error("[WSUrl] Hermes API TAT 脚本执行失败", "instance_id", instance.InstanceId, "error", runErr)
		return 0, "", hcommon.I18nRichError(runErr, i18n.MsgGetWSConnInfoFailed)
	}

	var result struct {
		Port  int    `json:"port"`
		Key   string `json:"key"`
		Error string `json:"error,omitempty"`
	}
	if parseErr := json.Unmarshal([]byte(output), &result); parseErr != nil {
		log.Error("[WSUrl] 解析 Hermes API 脚本输出失败", "instance_id", instance.InstanceId, "output", output, "error", parseErr)
		return 0, "", hcommon.I18nRichError(parseErr, i18n.MsgParseWSConnInfoFailed)
	}

	if result.Error != "" {
		return 0, "", hcommon.I18nError(i18n.MsgInstanceReturnedError, result.Error)
	}
	if result.Port == 0 || result.Key == "" {
		return 0, "", hcommon.I18nError(i18n.MsgHermesConfigIncomplete)
	}

	return result.Port, result.Key, nil
}
