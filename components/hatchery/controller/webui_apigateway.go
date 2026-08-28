// Package controller 中的 webui_apigateway.go 实现 WebUI 域名化接入云 API 网关的
// 软功能支持。设计第一原则：主流程不受影响。
//
// 软功能三件套：
//  1. 所有旁路调用入口 defer recover()，panic 不冒泡
//  2. 云 API 调用一律 context.WithTimeout(apiGatewayTimeout)
//  3. 旁路 error 只进 slog.Warn + 审计日志，不 return 到主流程
//
// 关联 OpenSpec change：webui-apigateway
package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tchttp "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/http"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

const (
	apiGatewayService = "apis"
	apiGatewayVersion = "2024-08-01"
	apiGatewayTimeout = 5 * time.Second
)

// CreateSignOnAgentServiceParams 对应云 API 2024-08-01 / CreateSignOnAgentService 入参。
// 参考文档：docs/apis-2024-08-01/服务管理相关接口/创建Agent认证服务.md
type CreateSignOnAgentServiceParams struct {
	InstanceID   string   `json:"InstanceID"`             // 网关实例 ID（来自 site_config.gateway_instance_id）
	AgentID      string   `json:"AgentID"`                // 业务侧唯一键：CVM 实例 ID
	IP           string   `json:"IP"`                     // 回源 IP
	Port         int      `json:"Port"`                   // 回源端口
	AgentToken   string   `json:"AgentToken,omitempty"`   // 实例自身 token（如 OpenClaw authToken）
	Path         string   `json:"Path,omitempty"`         // 回源路径
	BaseDomain   string   `json:"BaseDomain,omitempty"`   // 前端域名后缀
	AllowedUsers []string `json:"AllowedUsers,omitempty"` // OneID 白名单
}

// DeleteSignOnAgentServiceParams 对应云 API 2024-08-01 / DeleteSignOnAgentService 入参。
// 参考文档：docs/apis-2024-08-01/服务管理相关接口/删除Agent认证服务.md
type DeleteSignOnAgentServiceParams struct {
	InstanceID string `json:"InstanceID"` // 网关实例 ID
	AgentID    string `json:"AgentID"`    // 业务侧唯一键：CVM 实例 ID
}

// apiGatewayClient 是云 API 网关调用的抽象，方便测试 mock。
type apiGatewayClient interface {
	CreateSignOnAgentService(ctx context.Context, p CreateSignOnAgentServiceParams) error
	DeleteSignOnAgentService(ctx context.Context, p DeleteSignOnAgentServiceParams) error
}

// defaultAPIGatewayClient 是默认实现，运行时被 MaybeOverrideWithGateway 使用。
// 测试通过 setAPIGatewayClientForTest 替换。
var defaultAPIGatewayClient apiGatewayClient = &commonAPIGatewayClient{}

// setAPIGatewayClientForTest 替换全局客户端，返回 restore 函数。仅用于测试。
func setAPIGatewayClientForTest(c apiGatewayClient) (restore func()) {
	prev := defaultAPIGatewayClient
	defaultAPIGatewayClient = c
	return func() { defaultAPIGatewayClient = prev }
}

// resolveAPIGatewayEndpoint 依据 region 动态拼接 endpoint。
// 形如 apis.ap-guangzhou.tencentcloudapi.com。
func resolveAPIGatewayEndpoint(region string) string {
	if region == "" {
		region = "ap-guangzhou"
	}
	return "apis." + region + ".tencentcloudapi.com"
}

// maskToken 保留前 4 个字符，其余用 * 替代，用于日志脱敏（如 AgentToken / sk- Token）。
func maskToken(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 4 {
		return strings.Repeat("*", len(s))
	}
	return s[:4] + strings.Repeat("*", len(s)-4)
}

// commonAPIGatewayClient 是 apiGatewayClient 的默认实现。
type commonAPIGatewayClient struct{}

func (c *commonAPIGatewayClient) CreateSignOnAgentService(ctx context.Context, p CreateSignOnAgentServiceParams) error {
	return c.invoke(ctx, "CreateSignOnAgentService", p)
}

func (c *commonAPIGatewayClient) DeleteSignOnAgentService(ctx context.Context, p DeleteSignOnAgentServiceParams) error {
	return c.invoke(ctx, "DeleteSignOnAgentService", p)
}

// invoke 是两个 Action 的共用调用路径，走 CommonClient。
func (c *commonAPIGatewayClient) invoke(ctx context.Context, action string, params any) error {
	credential, err := getCredential(ctx)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgAPIGatewayGetCredFailed)
	}
	region := CVMRegion
	if region == "" {
		region = "ap-guangzhou"
	}
	endpoint := resolveAPIGatewayEndpoint(region)

	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = endpoint
	cpf.HttpProfile.Scheme = "https"
	cpf.HttpProfile.ReqMethod = "POST"
	cpf.HttpProfile.ReqTimeout = int(apiGatewayTimeout.Seconds())

	client := common.NewCommonClient(credential, region, cpf)
	request := tchttp.NewCommonRequest(apiGatewayService, apiGatewayVersion, action)

	actionJSON, marshalErr := json.Marshal(params)
	if marshalErr != nil {
		return hcommon.I18nRichError(marshalErr, i18n.MsgAPIGatewayMarshalParamsFailed)
	}
	if setErr := request.SetActionParameters(string(actionJSON)); setErr != nil {
		return hcommon.I18nRichError(setErr, i18n.MsgAPIGatewaySetParamsFailed)
	}

	// 传递 ctx 的超时语义：CommonClient 本身无 ctx 入参，这里用 channel + select 收敛。
	// 注：此 goroutine 仅调用已构建好的 client.Send，不访问 DB/TenantSnapshot，无需 DetachContext。
	done := make(chan error, 1)
	response := tchttp.NewCommonResponse()
	go func() {
		done <- client.Send(request, response)
	}()

	select {
	case <-ctx.Done():
		return hcommon.I18nRichError(ctx.Err(), i18n.MsgAPIGatewayCanceled, action)
	case sendErr := <-done:
		if sendErr != nil {
			return hcommon.I18nRichError(sendErr, i18n.MsgAPIGatewaySendFailed, action)
		}
	}

	respBody := response.GetBody()
	return parseAPIGatewayResponse(respBody, action)
}

// parseAPIGatewayResponse 解析云 API 响应，把业务错误码包装为 Go error。
// 独立出来方便单测覆盖，SDK 胶水层本身测试价值低。
func parseAPIGatewayResponse(respBody []byte, action string) error {
	var result struct {
		Response struct {
			Data struct {
				ID string `json:"ID"`
			} `json:"Data"`
			Error *struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error"`
			RequestId string `json:"RequestId"`
		} `json:"Response"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return hcommon.I18nRichError(err, i18n.MsgAPIGatewayParseRespFailed, action, truncate(string(respBody), 512))
	}
	if result.Response.Error != nil {
		return hcommon.I18nError(i18n.MsgAPIGatewayBizError,
			action, result.Response.Error.Code, result.Response.Error.Message, result.Response.RequestId)
	}
	slog.Info("[apigateway] call success",
		"action", action,
		"result_id", result.Response.Data.ID,
		"request_id", result.Response.RequestId)
	return nil
}

// truncate 截断字符串到 n 个字符，避免日志爆量。
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(truncated)"
}

// ---------------------------------------------------------------------------
// 软功能包装层：主流程永不受影响
// ---------------------------------------------------------------------------

// MaybeOverrideWithGateway 在命中新模式时尝试调用 CreateSignOnAgentService
// 并把返回的 URL 替换为域名；任何失败都静默降级为 primary。
//
// 参数：
//   - r: 原 HTTP 请求，用于取 ctx 和旁路审计上下文
//   - siteCfg: 站点配置
//   - user: 当前登录用户（用于 OneIDSub + 审计字段）
//   - instance: 目标实例
//   - ip: 回源 IP
//   - port: 回源端口
//   - agentToken: 实例自身 token（OpenClaw authToken 等；其它 AgentType 可为空）
//   - path: 回源路径（OpenClaw = basePath，去掉末尾 /；其它 = "/"）
//   - primary: 主流程已算出的 URL，失败/关闭时原样返回
//
// 软功能保证：入口 defer recover + context.WithTimeout(5s) + 错误只落日志/审计。
func MaybeOverrideWithGateway(
	r *http.Request,
	siteCfg model.SiteConfig,
	user *model.User,
	instance *model.Instance,
	ip string,
	port int,
	agentToken string,
	path string,
	primary string,
) (finalURL string) {
	finalURL = primary
	defer func() {
		if rec := recover(); rec != nil {
			slog.Warn("[apigateway] MaybeOverrideWithGateway panicked, fallback to primary",
				"recover", rec, "instance_id", instance.InstanceId)
			finalURL = primary
		}
	}()

	cfg, ok := siteCfg.GetAPIGatewayConfig()
	if !ok {
		return primary // JSON 坏了当关闭
	}
	// 仅 OpenClaw 走云 API 网关；Lightclaw/Hermes 的 WebUI 契约不匹配，直接返回主流程 URL
	if instance == nil || !model.AgentTypeSupportsAPIGateway(r.Context(), instance.AgentType) {
		return primary
	}
	oneIDSub := ""
	if user != nil && user.OneIDSub != nil {
		oneIDSub = *user.OneIDSub
	}
	if !cfg.ShouldActivate(oneIDSub) {
		return primary
	}

	ctx, cancel := context.WithTimeout(r.Context(), apiGatewayTimeout)
	defer cancel()

	params := CreateSignOnAgentServiceParams{
		InstanceID:   cfg.GatewayInstanceID,
		AgentID:      instance.InstanceId,
		IP:           ip,
		Port:         port,
		AgentToken:   agentToken,
		Path:         path,
		BaseDomain:   cfg.BaseDomain,
		AllowedUsers: []string{oneIDSub},
	}

	startedAt := time.Now()

	// 幂等策略：Create 之前先 Delete。网关侧 CreateSignOnAgentService 不支持 path 更新，
	// 而 OpenClaw 重装/升级后 basePath 会变，这里先清理旧记录再创建新记录以保证 URL 可用。
	// Delete 失败不阻塞（首次创建时服务本就不存在，必然报"Service记录不存在"）。
	delParams := DeleteSignOnAgentServiceParams{
		InstanceID: cfg.GatewayInstanceID,
		AgentID:    instance.InstanceId,
	}
	if delErr := defaultAPIGatewayClient.DeleteSignOnAgentService(ctx, delParams); delErr != nil {
		slog.Debug("[apigateway] pre-Delete skipped (likely first-time or already absent)",
			"instance_id", instance.InstanceId, "err", delErr)
	}

	if err := defaultAPIGatewayClient.CreateSignOnAgentService(ctx, params); err != nil {
		slog.Warn("[apigateway] CreateSignOnAgentService failed, fallback to primary URL",
			"instance_id", instance.InstanceId, "err", err, "token_mask", maskToken(agentToken))
		auditAPIGateway(startedAt, r, user, "api_gateway_create_failed", instance.InstanceId, "failed")
		return primary
	}
	auditAPIGateway(startedAt, r, user, "api_gateway_create", instance.InstanceId, "success")

	// URL path：OpenClaw 的 basePath（形如 /7gk5p7）直接拼上；其他 agent 情况 "/" 忽略避免末尾多余斜杠
	urlPath := path
	if urlPath == "/" {
		urlPath = ""
	}
	return fmt.Sprintf("%s://%s.%s%s", cfg.SchemeOrDefault(), instance.InstanceId, cfg.BaseDomain, urlPath)
}

// auditAPIGateway 旁路审计日志写入。自身也要 recover 以防数据库异常冒泡。
// 故意不使用独立 goroutine：一是保证 recover 能生效；二是 LogAudit 本身很轻，
// 不会明显拖慢主流程。即便 DB 未初始化（比如测试环境）也会被这里兜住。
func auditAPIGateway(startedAt time.Time, r *http.Request, user *model.User, action, resourceID, status string) {
	defer func() {
		if rec := recover(); rec != nil {
			slog.Warn("[apigateway] audit write panicked, ignored", "recover", rec)
		}
	}()
	if model.DB(r.Context()) == nil {
		return
	}
	var uid uint
	var uname string
	if user != nil {
		uid = user.ID
		uname = user.Username
	}
	model.LogAudit(r.Context(), startedAt, uid, uname, action, "instance", resourceID, status)
}
