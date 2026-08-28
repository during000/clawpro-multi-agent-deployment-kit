package controller

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	sdkcommon "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	sdkerrors "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	tat "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tat/v20201028"
)

// LightClaw 相关常量
const (
	lightClawProductCode = "clawpro" // 产品标识，用于 token 签名和 auth 鉴权校验
)

type lightClawHandlerDependencies interface {
	RunScript(ctx context.Context, instanceId string, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error)
}

type defaultLightClawHandlerDependencies struct{}

func (defaultLightClawHandlerDependencies) RunScript(ctx context.Context, instanceId string, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
	output, rerr := agentScriptRunner(ctx, instanceId, scriptName, timeout, runtimeUser, onOutput, params)
	if rerr != nil {
		return output, rerr
	}
	return output, nil
}

type lightClawHandler struct {
	deps lightClawHandlerDependencies
}

func newLightClawHandler(deps lightClawHandlerDependencies) *lightClawHandler {
	return &lightClawHandler{deps: deps}
}

// lightClawResponse 是 LightClaw TAT 接口的统一响应结构体。
// Code 成功时为 0（int），失败时为云 API Error.Code（string），使用 interface{} 兼容两种类型。
type lightClawResponse struct {
	Code      interface{} `json:"code"`
	Data      interface{} `json:"data"`
	Message   string      `json:"message"`
	Error     *string     `json:"error"`
	Timestamp string      `json:"timestamp"`
	Tid       string      `json:"tid"`
}

// newTid 使用 crypto/rand 生成 32 位 hex 追踪 ID。
func newTid() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// lightClawOK 包装成功响应并写入 ResponseWriter。
func lightClawOK(w http.ResponseWriter, data interface{}) {
	resp := lightClawResponse{
		Code:      0,
		Data:      data,
		Message:   "OK",
		Error:     nil,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Tid:       newTid(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// lightClawError 包装错误响应并写入 ResponseWriter。
// errCode 为云 API ErrorCode（如 "InvalidParameterValue"），将直接作为响应 code 字段，供前端组件识别。
// message 为错误描述，rawResponse 为云 API 原始响应（可为 nil）。
func lightClawError(w http.ResponseWriter, errCode, message string, rawResponse interface{}) {
	resp := lightClawResponse{
		Code:      errCode,
		Data:      rawResponse,
		Message:   message,
		Error:     &errCode,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Tid:       newTid(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// generateSign 生成 LightClaw HMAC-SHA256 签名。
// params 中包含 businessCode、callbackUrl、proxyToken、timestamp 四项，
// sid 和 skey 内置于函数中，不对外暴露。
func generateSign(params map[string]string) string {
	const (
		sid  = "clawpro_2026"
		skey = "clawpro_ai_7s6kN2p5"
	)
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+params[k])
	}
	signStr := strings.Join(parts, "&") + "&secretId=" + sid + "&secretKey=" + skey
	mac := hmac.New(sha256.New, []byte(skey))
	mac.Write([]byte(signStr))
	return hex.EncodeToString(mac.Sum(nil))
}

// HandleLightClawToken 获取实例 ProxyToken 及签名信息，供前端传递给 LightClaw 组件。
// GET /openclaw/lightclaw/token?id=<实例数据库ID>
func HandleLightClawToken(w http.ResponseWriter, r *http.Request) {
	newLightClawHandler(defaultLightClawHandlerDependencies{}).handleToken(w, r)
}

func (h *lightClawHandler) handleToken(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	// 校验服务域名
	if hcommon.DomainFromCtx(r.Context()) == "" {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgServiceDomainNotConfigured))
		return
	}

	// 查询实例，校验归属
	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, http.StatusNotFound, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	idStr := r.URL.Query().Get("id")

	// 校验 ProxyToken
	if instance.ProxyToken == nil || *instance.ProxyToken == "" {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgInstanceProxyTokenNotConfigured))
		return
	}
	proxyToken := *instance.ProxyToken

	// 构造 callbackUrl 和签名
	callbackUrl := hcommon.DomainFromCtx(r.Context()) + "/api/openclaw/lightclaw/auth"
	timestamp := time.Now().Unix()
	timestampStr := strconv.FormatInt(timestamp, 10)

	params := map[string]string{
		"businessCode": lightClawProductCode,
		"timestamp":    timestampStr,
		"callbackUrl":  callbackUrl,
		"proxyToken":   proxyToken,
	}
	signature := generateSign(params)

	if model.AgentTypeSupportsApprove(r.Context(), instance.AgentType) {
		// final：用户首次在新建实例上点 LightClaw token 时，openclaw-gateway 可能尚未就绪
		// （TAT online ≠ gateway online），直接下发 approve_device.sh 会因 paired.json
		// 未落盘而 step 1 报错。这里前置 waitForOpenclawReady（最多 5 分钟，比新建/升级
		// 路径短，因为这是用户主动触发的同步接口，需要尽量快返回），与 approveDeviceAsync /
		// approveDeviceAfterUpgrade 保持一致语义。
		if err := waitForOpenclawReady(r.Context(), instance.InstanceId, instance.AgentType, 5*time.Minute); err != nil {
			slog.Warn("[LightClaw] 等待 openclaw 就绪超时", "instance_id", instance.InstanceId, "error", err)
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgLightClawInstanceServiceNotReady))
			return
		}
		if _, err := h.deps.RunScript(r.Context(), instance.InstanceId, "approve_device.sh", 300, instance.RuntimeUser, nil, nil); err != nil {
			slog.Warn("[LightClaw] approve device 失败", "instance_id", instance.InstanceId, "error", err)
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgLightClawApproveDeviceFailed))
			return
		}
	}

	// 手动记录审计日志（WithAudit 跳过 GET 请求）
	go model.LogAudit(hcommon.DetachContext(r.Context()), time.Now(), user.ID, user.Username, "lightclaw_token_get", "instance", idStr, "success")

	jsonOK(w, map[string]interface{}{
		"proxyToken":   proxyToken,
		"callbackUrl":  callbackUrl,
		"businessCode": lightClawProductCode,
		"timestamp":    timestamp,
		"sign":         signature,
	})
}

// HandleLightClawAuth 鉴权接口，供 LightClaw 后端以 JSON body 调用，返回用户信息。
// POST /openclaw/lightclaw/auth
func HandleLightClawAuth(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	// 解析请求体
	var body struct {
		Product     string `json:"product"`
		AccessToken string `json:"accessToken"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}

	// 校验 product
	if body.Product != lightClawProductCode {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidProduct))
		return
	}

	// 校验 accessToken
	token := body.AccessToken
	if token == "" {
		writeError(w, r, http.StatusUnauthorized, hcommon.I18nError(i18n.MsgInvalidAccessToken))
		return
	}

	// 查询实例（proxy_token 字段）
	var instance model.Instance
	if model.DB(r.Context()).Where("proxy_token = ?", token).First(&instance).Error != nil {
		// Token 无效与实例不存在合并，避免枚举攻击
		writeError(w, r, http.StatusUnauthorized, hcommon.I18nError(i18n.MsgInvalidAccessToken))
		return
	}

	// 查询关联用户
	var user model.User
	if model.DB(r.Context()).First(&user, instance.UserID).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgUserDataAbnormal))
		return
	}

	jsonOK(w, map[string]interface{}{
		"code": 0,
		"data": map[string]interface{}{
			"user_id":     lightClawUserID(r.Context(), user.ID),
			"username":    user.Username,
			"id":          instance.ID,
			"instance_id": instance.InstanceId,
		},
		"message": "OK",
	})
}

// HandleLightClawDescribeInvocations 封装 TAT DescribeInvocations 接口，校验返回结果 InstanceId 归属。
// POST /openclaw/lightclaw/describe-invocations?id=<实例数据库ID>
func HandleLightClawDescribeInvocations(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, http.StatusNotFound, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 解析请求体
	var body struct {
		InvocationIds []string `json:"InvocationIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err.Error() != "EOF" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}

	// InvocationIds 不允许为空
	if len(body.InvocationIds) == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvocationIdsRequired))
		return
	}

	// 创建 TAT 客户端
	client, err := NewTATClient(r.Context())
	if err != nil {
		lightClawError(w, "InternalError", "InternalError: 创建 TAT 客户端失败", nil)
		return
	}

	// 构造请求
	req := tat.NewDescribeInvocationsRequest()
	req.InvocationIds = sdkcommon.StringPtrs(body.InvocationIds)

	resp, err := client.DescribeInvocations(req)
	if err != nil {
		var sdkErr *sdkerrors.TencentCloudSDKError
		if ok := isSDKError(err, &sdkErr); ok {
			lightClawError(w, sdkErr.GetCode(), sdkErr.GetCode()+": "+sdkErr.GetMessage(), nil)
		} else {
			lightClawError(w, "InternalError", "InternalError: "+err.Error(), nil)
		}
		return
	}

	// 校验返回结果中的 InstanceId 是否与当前实例匹配
	targetId := instance.InstanceId
	for _, inv := range resp.Response.InvocationSet {
		for _, task := range inv.InvocationTaskBasicInfoSet {
			if task.InstanceId == nil || *task.InstanceId != targetId {
				lightClawError(w, "InvalidInstance", "InstanceId 不匹配", nil)
				return
			}
		}
	}

	lightClawOK(w, map[string]interface{}{"Response": resp.Response})
}

// HandleLightClawDescribeInvocationTasks 封装 TAT DescribeInvocationTasks 接口，校验返回结果 InstanceId 归属。
// POST /openclaw/lightclaw/describe-invocation-tasks?id=<实例数据库ID>
func HandleLightClawDescribeInvocationTasks(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, http.StatusNotFound, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 解析请求体
	var body struct {
		InvocationTaskIds []string `json:"InvocationTaskIds"`
		HideOutput        bool     `json:"HideOutput"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err.Error() != "EOF" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}

	// InvocationTaskIds 不允许为空
	if len(body.InvocationTaskIds) == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvocationTaskIdsRequired))
		return
	}

	// 创建 TAT 客户端
	client, err := NewTATClient(r.Context())
	if err != nil {
		lightClawError(w, "InternalError", "InternalError: 创建 TAT 客户端失败", nil)
		return
	}

	// 构造请求
	req := tat.NewDescribeInvocationTasksRequest()
	req.InvocationTaskIds = sdkcommon.StringPtrs(body.InvocationTaskIds)
	req.HideOutput = sdkcommon.BoolPtr(body.HideOutput)

	resp, err := client.DescribeInvocationTasks(req)
	if err != nil {
		var sdkErr *sdkerrors.TencentCloudSDKError
		if ok := isSDKError(err, &sdkErr); ok {
			lightClawError(w, sdkErr.GetCode(), sdkErr.GetCode()+": "+sdkErr.GetMessage(), nil)
		} else {
			lightClawError(w, "InternalError", "InternalError: "+err.Error(), nil)
		}
		return
	}

	// 校验返回结果中的 InstanceId 是否与当前实例匹配
	targetId := instance.InstanceId
	for _, task := range resp.Response.InvocationTaskSet {
		if task.InstanceId == nil || *task.InstanceId != targetId {
			lightClawError(w, "InvalidInstance", "InstanceId 不匹配", nil)
			return
		}
	}

	lightClawOK(w, map[string]interface{}{"Response": resp.Response})
}

// HandleLightClawRunCommand 封装 TAT RunCommand 接口，校验 InstanceIds 权限后下发命令。
// POST /openclaw/lightclaw/run-command?id=<实例数据库ID>
func HandleLightClawRunCommand(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, http.StatusNotFound, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 解析请求体
	var body struct {
		Content     string   `json:"Content"`
		InstanceIds []string `json:"InstanceIds"`
		CommandType string   `json:"CommandType"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}

	// 校验必填参数
	if body.Content == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgContentRequired))
		return
	}

	// 校验 InstanceIds：长度必须为 1 且等于该实例的 CVM InstanceId
	if len(body.InstanceIds) != 1 || body.InstanceIds[0] != instance.InstanceId {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgInstanceIdsMismatch))
		return
	}

	// 创建 TAT 客户端
	client, tatErr := NewTATClient(r.Context())
	if tatErr != nil {
		lightClawError(w, "InternalError", "InternalError: 创建 TAT 客户端失败", nil)
		return
	}

	// 校验 TAT Agent 是否在线
	if agentErr := checkAgentOnline(client, instance.InstanceId); agentErr != nil {
		lightClawError(w, "InvalidInstance.NotRunning", agentErr.Error(), nil)
		return
	}

	// 构造请求
	req := tat.NewRunCommandRequest()
	// 以实例运行用户身份执行远程命令（非 root）
	runUser, workdir := getDefaultTATRunIdentity(instance.RuntimeUser)
	req.Content = sdkcommon.StringPtr(body.Content)
	req.InstanceIds = sdkcommon.StringPtrs(body.InstanceIds)
	req.SaveCommand = sdkcommon.BoolPtr(false)
	req.Timeout = sdkcommon.Uint64Ptr(600) // 超时时间 600 秒
	req.Username = sdkcommon.StringPtr(runUser)
	req.WorkingDirectory = sdkcommon.StringPtr(workdir)
	if body.CommandType != "" {
		req.CommandType = sdkcommon.StringPtr(body.CommandType)
	} else {
		req.CommandType = sdkcommon.StringPtr("SHELL")
	}

	resp, err := client.RunCommand(req)
	if err != nil {
		var sdkErr *sdkerrors.TencentCloudSDKError
		if ok := isSDKError(err, &sdkErr); ok {
			lightClawError(w, sdkErr.GetCode(), sdkErr.GetCode()+": "+sdkErr.GetMessage(), nil)
		} else {
			lightClawError(w, "InternalError", "InternalError: "+err.Error(), nil)
		}
		return
	}

	lightClawOK(w, map[string]interface{}{"Response": resp.Response})
}

// lightClawUserID 根据 Domain 提取域名前缀，拼接用户 ID，生成 LightClaw 场景下的唯一 user_id。
// 例如 Domain="https://x8swfkbg.tcaisite.com"，userID=3 → "x8swfkbg-3"。
// 使用 domain 前缀而非 CVMUin，保证同一 uin 下多套 hatchery 部署时 user_id 全局唯一。
func lightClawUserID(ctx context.Context, userID uint) string {
	prefix := hcommon.DomainFromCtx(ctx)
	prefix = strings.TrimPrefix(prefix, "https://")
	prefix = strings.TrimPrefix(prefix, "http://")
	if idx := strings.Index(prefix, "."); idx > 0 {
		prefix = prefix[:idx]
	}
	return fmt.Sprintf("%s-%d", prefix, userID)
}

// isSDKError 判断 err 是否为腾讯云 SDK 错误，若是则通过 out 指针返回强类型错误对象。
func isSDKError(err error, out **sdkerrors.TencentCloudSDKError) bool {
	if sdkErr, ok := err.(*sdkerrors.TencentCloudSDKError); ok {
		*out = sdkErr
		return true
	}
	return false
}
