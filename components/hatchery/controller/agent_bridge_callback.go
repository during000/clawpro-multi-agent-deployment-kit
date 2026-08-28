package controller

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	tencentcommon "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	"gorm.io/gorm"
)

// ========== Agent-Bridge 回调接口 ==========
//
// Agent-Bridge 使用 HTTPAdapter 模式时，会回调 Hatchery 获取凭证、鉴权和实例列表。
// 这些接口通过 WithOpenAPI + requireLogin 进行用户 API Token 鉴权。
// Agent-Bridge 透传前端用户的 Authorization: Bearer hk-xxx Header，
// Hatchery 通过 getUserFromToken 解析用户身份，确保 user_id 不可伪造。
//
// 回调路径由 Agent-Bridge 配置中的 callback_base_url 决定：
//   callback_base_url: "http://hatchery:8080/agent-bridge"
//   → POST /agent-bridge/sts
//   → POST /agent-bridge/auth
//   → POST /agent-bridge/instances

// resolveAgentBridgeIdentity 解析 agent-bridge 回调请求的身份。
//
// 鉴权优先级：
//  1. Bearer 前缀为 "sk-"：查 instances.proxy_token，返回该实例对应的 user +
//     boundInstance（调用方应将本次请求锁定到这台实例）
//  2. 其他（hk- API Token / AdminToken / Session）：走原 requireLogin
//
// 返回值：
//   - user:          认证通过的用户（非 nil 时表示鉴权通过）
//   - boundInstance: 仅 sk- 模式非 nil；调用方据此把本次请求锁定到这台实例
//   - ok:            false 表示鉴权失败，已写入响应，调用方直接 return
func resolveAgentBridgeIdentity(w http.ResponseWriter, r *http.Request) (*model.User, *model.Instance, bool) {
	auth := r.Header.Get("Authorization")
	token := strings.TrimPrefix(auth, "Bearer ")

	// sk- 前缀走 ProxyToken 路径（hk- / Admin / Session 都走 else 分支）
	if strings.HasPrefix(token, "sk-") {
		var inst model.Instance
		if err := model.DB(r.Context()).Where("proxy_token = ?", token).First(&inst).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				writeError(w, r, http.StatusUnauthorized, hcommon.I18nError(i18n.MsgABInvalidProxyToken))
			} else {
				slog.Error("[AgentBridge] proxy token lookup failed", "error", err)
				writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgInternalError))
			}
			return nil, nil, false
		}
		var user model.User
		// Unscoped 以便检查软删除（封禁）状态
		if err := model.DB(r.Context()).Unscoped().First(&user, inst.UserID).Error; err != nil {
			writeError(w, r, http.StatusUnauthorized, hcommon.I18nError(i18n.MsgABOrphanProxyToken))
			return nil, nil, false
		}
		if user.DeletedAt.Valid {
			writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgABUserBanned))
			return nil, nil, false
		}
		return &user, &inst, true
	}

	// 其他凭证类型走原 requireLogin（hk- / Admin / Session 都在内部分支处理）
	if user := requireLogin(w, r); user != nil {
		return user, nil, true
	}
	return nil, nil, false // requireLogin 已写入错误响应
}

// HandleAgentBridgeSTS 处理 Agent-Bridge 的 STS 凭证回调。
//
// Agent-Bridge 的 STSFetch 中间件在 Redis 缓存未命中时，
// 通过 HTTPAdapter.GetSTS() 回调此接口获取腾讯云操作凭证。
//
// 鉴权：通过 WithOpenAPI + requireLogin，从 Bearer Token 解析用户身份。
//
// POST /agent-bridge/sts
//
// 请求头:
//
//	Authorization: Bearer hk-xxx
//
// 请求体:
//
//	{"platform_id": "hatchery", "user_id": "123"}
//
// 响应体:
//
//	{"tmp_secret_id": "...", "tmp_secret_key": "...", "token": "...", "expired_time": 1234567890}
//
// 凭证模式:
//   - 有 UIN 配置时: 返回 STS 临时密钥（token 非空，expired_time 为 Unix 时间戳）
//   - 无 UIN 配置时: 返回长期 AK/SK（token 为空，expired_time 为 0）
func HandleAgentBridgeSTS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, hcommon.I18nError(i18n.MsgMethodNotAllowed))
		return
	}

	// 鉴权（hk- / Admin / Session / sk- ProxyToken 全部支持）
	user, boundInst, ok := resolveAgentBridgeIdentity(w, r)
	if !ok {
		return
	}

	var req struct {
		PlatformID string `json:"platform_id"`
		UserID     string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}

	if boundInst != nil {
		slog.Info("[AgentBridge] STS 回调 (sk- 模式)",
			"platform_id", req.PlatformID, "user_id", user.ID,
			"instance_id", boundInst.InstanceId)
	} else {
		slog.Info("[AgentBridge] STS 回调", "platform_id", req.PlatformID, "user_id", user.ID)
	}

	config := model.GetSiteConfig(r.Context())
	if config.CVMSecretId == "" || config.CVMSecretKey == "" {
		slog.Error("[AgentBridge] STS 回调失败: 凭据未配置")
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgABCredsNotConfigured))
		return
	}

	// 判断是否使用 STS 模式（有 UIN 配置时走 AssumeRole 临时凭证）
	uin := hcommon.CVMUinFromCtx(r.Context())
	if uin != "" {
		// STS 模式：检查临时密钥是否有效（提前 5 分钟刷新）
		if config.STSTmpSecretId == "" || config.STSToken == "" ||
			time.Now().Unix() >= config.STSExpiredAt-300 {
			if err := RefreshSTSCredentials(r.Context()); err != nil {
				slog.Error("[AgentBridge] STS 刷新失败", "error", err)
				writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgABSTSRefreshFailed))
				return
			}
			config = model.GetSiteConfig(r.Context())
		}
		jsonOK(w, map[string]interface{}{
			"tmp_secret_id":  config.STSTmpSecretId,
			"tmp_secret_key": config.STSTmpSecretKey,
			"token":          config.STSToken,
			"expired_time":   config.STSExpiredAt,
		})
	} else {
		// 长期密钥模式：直接返回 AK/SK，token 为空
		jsonOK(w, map[string]interface{}{
			"tmp_secret_id":  config.CVMSecretId,
			"tmp_secret_key": config.CVMSecretKey,
			"token":          "",
			"expired_time":   0,
		})
	}
}

// HandleAgentBridgeAuth 处理 Agent-Bridge 的鉴权回调。
//
// Agent-Bridge 的 Authorize 中间件在每个需要鉴权的请求中，
// 通过 HTTPAdapter.CheckAuth() 回调此接口校验用户权限。
//
// 鉴权：通过 WithOpenAPI + requireLogin，从 Bearer Token 解析用户身份。
// 用户身份从 Token 中获取，不再依赖 Body 中的 user_id（防伪造）。
//
// POST /agent-bridge/auth
//
// 请求头:
//
//	Authorization: Bearer hk-xxx
//
// 请求体:
//
//	{"platform_id": "hatchery", "user_id": "123", "action": "desktop:install", "resource": "hatchery"}
//
// 响应体:
//
//	{"allowed": true, "reason": ""}
//
// 校验逻辑:
//  1. 通过 Bearer Token 校验用户身份（requireLogin）
//  2. 校验功能开关（desktop:* 操作需要 BrowserVNCEnable 开启）
func HandleAgentBridgeAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, hcommon.I18nError(i18n.MsgMethodNotAllowed))
		return
	}

	// 鉴权（hk- / Admin / Session / sk- ProxyToken 全部支持）
	user, boundInst, ok := resolveAgentBridgeIdentity(w, r)
	if !ok {
		return
	}

	var req struct {
		PlatformID string `json:"platform_id"`
		UserID     string `json:"user_id"`
		Action     string `json:"action"`
		Resource   string `json:"resource"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}

	if boundInst != nil {
		slog.Info("[AgentBridge] Auth 回调 (sk- 模式)",
			"platform_id", req.PlatformID,
			"user_id", user.ID,
			"action", req.Action,
			"resource", req.Resource,
			"bound_instance_id", boundInst.InstanceId)

		// Phase 1（变体 A，保守模式）：sk- 模式下，如果 resource 字段携带 instance:ins-xxx
		// 且与绑定实例不一致，仅打 warn 日志，不拒绝。
		// 等 agent-bridge 上游 Authorize 中间件的 resource 字段约定明确后，再切换为严格模式。
		if resInst := extractInstanceIDFromResource(req.Resource); resInst != "" && resInst != boundInst.InstanceId {
			slog.Warn("[AgentBridge] sk- 模式下 resource 与绑定实例不一致 (仅告警，未拒绝)",
				"bound_instance_id", boundInst.InstanceId,
				"resource_instance_id", resInst,
				"action", req.Action)
		}
	} else {
		slog.Info("[AgentBridge] Auth 回调",
			"platform_id", req.PlatformID,
			"user_id", user.ID,
			"action", req.Action)
	}

	// 用户身份已通过 Token 校验（requireLogin 已检查封禁状态），
	// 此处只需检查功能开关

	// 功能开关检查：desktop:* 相关操作需要 BrowserVNCEnable 开启
	if strings.HasPrefix(req.Action, "desktop:") {
		config := model.GetSiteConfig(r.Context())
		if !config.BrowserVNCEnable {
			jsonOK(w, map[string]interface{}{
				"allowed": false,
				"reason":  "browser vnc feature is disabled",
			})
			return
		}
	}

	// 放行
	jsonOK(w, map[string]interface{}{
		"allowed": true,
		"reason":  "",
	})
}

// HandleAgentBridgeInstances 处理 Agent-Bridge 的实例列表回调。
//
// Agent-Bridge 的 VNCURLService.Resolve() 在缓存未命中时，
// 通过 HTTPAdapter.ListInstances() 回调此接口获取实例公网 IP。
//
// 鉴权：通过 WithOpenAPI + requireLogin，从 Bearer Token 解析用户身份。
// 用户身份从 Token 中获取，确保只能查询自己的实例。
//
// POST /agent-bridge/instances
//
// 请求头:
//
//	Authorization: Bearer hk-xxx
//
// 请求体:
//
//	{"platform_id": "hatchery", "user_id": "123", "region": "ap-guangzhou"}
//
// 响应体:
//
//	{"instances": [{"instance_id": "ins-xxx", "instance_type": "CVM", "region": "ap-guangzhou",
//	  "public_ip": "43.139.x.x", "private_ip": "10.0.1.x", "name": "my-instance", "status": "RUNNING"}]}
//
// 实现逻辑:
//  1. 通过 Bearer Token 获取用户身份
//  2. 从 Hatchery DB 查询该用户的 Instance 记录
//  3. 批量调用 CVM DescribeInstances 获取实时公网 IP 和状态
//  4. 按 region 过滤后返回
func HandleAgentBridgeInstances(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, hcommon.I18nError(i18n.MsgMethodNotAllowed))
		return
	}

	// 鉴权（hk- / Admin / Session / sk- ProxyToken 全部支持）
	user, boundInst, ok := resolveAgentBridgeIdentity(w, r)
	if !ok {
		return
	}

	var req struct {
		PlatformID string `json:"platform_id"`
		UserID     string `json:"user_id"`
		Region     string `json:"region"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}

	if boundInst != nil {
		slog.Info("[AgentBridge] Instances 回调 (sk- 模式)",
			"platform_id", req.PlatformID,
			"user_id", user.ID,
			"region", req.Region,
			"bound_instance_id", boundInst.InstanceId)
	} else {
		slog.Info("[AgentBridge] Instances 回调",
			"platform_id", req.PlatformID,
			"user_id", user.ID,
			"region", req.Region)
	}

	// 使用 Token 解析的 user.ID 查询实例（防止 user_id 伪造）
	// sk- 模式下额外锁定到 boundInst.ID，确保只返回 ProxyToken 绑定的那一台
	var instances []model.Instance
	q := model.DB(r.Context()).Where("user_id = ? AND instance_id != ''", user.ID)
	if boundInst != nil {
		q = q.Where("id = ?", boundInst.ID)
	}
	if err := q.Find(&instances).Error; err != nil {
		slog.Error("[AgentBridge] 查询用户实例失败", "user_id", user.ID, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgABQueryInstancesFailed))
		return
	}

	if len(instances) == 0 {
		jsonOK(w, map[string]interface{}{"instances": []interface{}{}})
		return
	}

	// 2. 批量查询 CVM 实例的实时信息（公网 IP、内网 IP、状态）
	instanceIDs := make([]string, 0, len(instances))
	for _, inst := range instances {
		instanceIDs = append(instanceIDs, inst.InstanceId)
	}
	cvmInfoMap := agentBridgeBatchDescribeCVM(r.Context(), instanceIDs)

	// 3. 组装响应（符合 Agent-Bridge domain.Instance 结构）
	type instanceResp struct {
		InstanceID   string `json:"instance_id"`
		InstanceType string `json:"instance_type"`
		Region       string `json:"region"`
		PublicIP     string `json:"public_ip"`
		PrivateIP    string `json:"private_ip"`
		Name         string `json:"name"`
		Status       string `json:"status"`
	}

	result := make([]instanceResp, 0, len(instances))
	for _, inst := range instances {
		resp := instanceResp{
			InstanceID:   inst.InstanceId,
			InstanceType: "CVM",
			Region:       CVMRegion, // Hatchery 是单 Region 模式
			Name:         inst.Name,
			Status:       "UNKNOWN",
		}

		// 用 CVM API 返回的实时信息覆盖
		if info, ok := cvmInfoMap[inst.InstanceId]; ok {
			resp.PublicIP = info.publicIP
			resp.PrivateIP = info.privateIP
			resp.Status = info.state
		}

		// region 过滤
		if req.Region != "" && resp.Region != req.Region {
			continue
		}

		result = append(result, resp)
	}

	jsonOK(w, map[string]interface{}{"instances": result})
}

// ========== 内部辅助函数 ==========

// extractInstanceIDFromResource 从 agent-bridge Auth 回调的 resource 字段提取 instance_id。
// 当前约定的格式（可能扩展）：
//   - "instance:ins-xxx"  → 返回 "ins-xxx"
//   - 空串或其他形态     → 返回 ""（表示不约束）
func extractInstanceIDFromResource(resource string) string {
	const prefix = "instance:"
	if strings.HasPrefix(resource, prefix) {
		return strings.TrimPrefix(resource, prefix)
	}
	return ""
}

// agentBridgeCVMInfo 存储单个 CVM 实例的实时信息
type agentBridgeCVMInfo struct {
	state     string
	publicIP  string
	privateIP string
}

// HandleAgentBridgeAudit 处理 Agent-Bridge 的 TAT 执行审计回调。
//
// Agent-Bridge 在通过 TAT 执行脚本后，异步回调此接口将审计信息写入 Hatchery 的 audit_logs 表，
// 使管理员可以在统一的审计日志页面查看所有 TAT 操作（包括 Agent-Bridge 发起的）。
//
// 鉴权：与其他 agent-bridge 回调接口一致，通过 WithOpenAPI + resolveAgentBridgeIdentity 进行
// Bearer Token 鉴权（支持 hk- / sk- / Admin / Session）。
// 用户身份从 Token 中获取，确保 user_id 不可伪造。
//
// POST /agent-bridge/audit
//
// 请求头:
//
//	Authorization: Bearer hk-xxx
//
// 请求体:
//
//	{
//	  "platform_id": "hatchery",
//	  "user_id": "123",
//	  "action": "desktop_install",
//	  "resource": "instance",
//	  "resource_id": "ins-xxx",
//	  "invocation_id": "inv-xxx",
//	  "script_name": "install.sh",
//	  "status": "success",
//	  "trace_id": "trace-xxx",
//	  "started_at": 1716192000
//	}
//
// 响应体:
//
//	{"ok": true}
//
// 字段说明:
//   - action:        操作类型，建议使用 "agent_bridge_" 前缀（如 agent_bridge_desktop_install）
//   - resource:      资源类型（如 instance）
//   - resource_id:   资源标识（通常为 CVM 实例 ID）
//   - invocation_id: TAT invocation ID，用于关联 TAT 执行记录
//   - script_name:   执行的脚本名称（可选，用于详细追踪）
//   - status:        执行结果（success / failed / timeout / dispatched）
//   - trace_id:      调用链追踪 ID（可选）
//   - started_at:    操作开始时间（Unix 时间戳，秒；可选，为 0 时使用当前时间）
func HandleAgentBridgeAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, hcommon.I18nError(i18n.MsgMethodNotAllowed))
		return
	}

	// 请求体大小限制（审计请求体不应超过 4KB，防止 OOM 攻击）
	r.Body = http.MaxBytesReader(w, r.Body, 4096)

	// 鉴权（hk- / Admin / Session / sk- ProxyToken 全部支持）
	user, boundInst, ok := resolveAgentBridgeIdentity(w, r)
	if !ok {
		return
	}

	var req struct {
		PlatformID   string `json:"platform_id"`
		UserID       string `json:"user_id"`
		Action       string `json:"action"`
		Resource     string `json:"resource"`
		ResourceID   string `json:"resource_id"`
		InvocationID string `json:"invocation_id"`
		ScriptName   string `json:"script_name"`
		Status       string `json:"status"`
		TraceID      string `json:"trace_id"`
		StartedAt    int64  `json:"started_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}

	// 校验必填字段
	if req.Action == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "action"))
		return
	}
	if req.Status == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "status"))
		return
	}

	// 校验 action 前缀：强制要求 "agent_bridge_" 前缀，防止伪造其他类型的审计记录
	if !strings.HasPrefix(req.Action, "agent_bridge_") {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalidWithDetail, "action", "must start with 'agent_bridge_' prefix"))
		return
	}

	// 校验 status 值合法性
	switch req.Status {
	case "success", "failed", "timeout", "dispatched":
		// 合法值
	default:
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamInvalidWithDetail, "status", "must be one of: success, failed, timeout, dispatched"))
		return
	}

	// sk- 模式下，resource_id 必须与绑定实例一致（安全校验）
	if boundInst != nil && req.ResourceID != "" && req.ResourceID != boundInst.InstanceId {
		slog.Warn("[AgentBridge] Audit 回调: sk- 模式下 resource_id 与绑定实例不一致",
			"bound_instance_id", boundInst.InstanceId,
			"resource_id", req.ResourceID,
			"action", req.Action)
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgABResourceIdMismatch))
		return
	}

	// 确定操作开始时间（限制在合理范围内：不早于 24 小时前，不晚于当前时间 + 5 分钟）
	startedAt := time.Now()
	if req.StartedAt > 0 {
		t := time.Unix(req.StartedAt, 0)
		if time.Since(t) <= 24*time.Hour && !t.After(time.Now().Add(5*time.Minute)) {
			startedAt = t
		}
		// 超出合理范围时使用当前时间（静默修正，不拒绝请求）
	}

	// 确定 resource_id：优先使用请求中的，sk- 模式下可从绑定实例补充
	resourceID := req.ResourceID
	if resourceID == "" && boundInst != nil {
		resourceID = boundInst.InstanceId
	}

	slog.Info("[AgentBridge] Audit 回调",
		"platform_id", req.PlatformID,
		"user_id", user.ID,
		"username", user.Username,
		"action", req.Action,
		"resource", req.Resource,
		"resource_id", resourceID,
		"invocation_id", req.InvocationID,
		"script_name", req.ScriptName,
		"status", req.Status,
		"trace_id", req.TraceID,
		"bound_instance", boundInst != nil)

	// 异步写入审计日志（与 WithAudit 中间件行为一致）
	// invocation_id、script_name、trace_id 通过上方 slog.Info 记录在结构化日志中，
	// 不额外持久化到数据库，保持 audit_logs 表结构不变。
	go model.LogAudit(
		hcommon.DetachContext(r.Context()),
		startedAt,
		user.ID,
		user.Username,
		req.Action,
		req.Resource,
		resourceID,
		req.Status,
	)

	jsonOK(w, map[string]interface{}{"ok": true})
}

// agentBridgeBatchDescribeCVM 批量查询 CVM 实例信息（公网 IP、内网 IP、状态）。
// CVM DescribeInstances 单次最多查询 100 个实例。
// 查询失败时返回空 map（不阻断主流程），由调用方降级处理。
func agentBridgeBatchDescribeCVM(ctx context.Context, instanceIDs []string) map[string]*agentBridgeCVMInfo {
	result := make(map[string]*agentBridgeCVMInfo, len(instanceIDs))

	client, err := GetCVMClient(ctx)
	if err != nil {
		slog.Warn("[AgentBridge] 创建 CVM 客户端失败", "error", err)
		return result
	}

	// 分批查询，每批最多 100 个
	const batchSize = 100
	for i := 0; i < len(instanceIDs); i += batchSize {
		end := i + batchSize
		if end > len(instanceIDs) {
			end = len(instanceIDs)
		}
		batch := instanceIDs[i:end]

		request := cvm.NewDescribeInstancesRequest()
		request.InstanceIds = tencentcommon.StringPtrs(batch)
		response, err := client.DescribeInstances(request)
		if err != nil {
			slog.Warn("[AgentBridge] DescribeInstances 失败", "error", err, "batch_size", len(batch))
			continue
		}

		if response.Response == nil {
			continue
		}

		for _, inst := range response.Response.InstanceSet {
			info := &agentBridgeCVMInfo{}
			if inst.InstanceState != nil {
				info.state = *inst.InstanceState
			}
			if len(inst.PublicIpAddresses) > 0 && inst.PublicIpAddresses[0] != nil {
				info.publicIP = *inst.PublicIpAddresses[0]
			}
			if len(inst.PrivateIpAddresses) > 0 && inst.PrivateIpAddresses[0] != nil {
				info.privateIP = *inst.PrivateIpAddresses[0]
			}
			if inst.InstanceId != nil {
				result[*inst.InstanceId] = info
			}
		}
	}

	return result
}
