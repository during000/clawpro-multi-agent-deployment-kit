package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	cls "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cls/v20201016"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	sdkerrors "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	tchttp "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/http"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	sts "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sts/v20180813"
)

const (
	clsService = "cls"
	clsVersion = "2020-10-16"
)

const AgentCamRoleName = "CVM_QCSLinkedRoleInClawProAgent"

// newCLSClient creates a CLS client from SiteConfig credentials
// 等 ctx 收紧完成后，调用点应直接 GetCLSClient(ctx) 并删除此 helper。
func newCLSClient(ctx context.Context) (*cls.Client, error) {
	return GetCLSClient(ctx)
}

// newCLSCommonClient 创建一个用于调用 CLS 内部接口的 CommonClient。
// GetClsService / OpenClsService 等产品内部接口没有 SDK 封装，需要通过 CommonClient 调用。
func newCLSCommonClient(ctx context.Context) (*common.Client, error) {
	credential, err := getCredential(ctx)
	if err != nil {
		return nil, err
	}
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = clsService + ".tencentcloudapi.com"
	cpf.HttpProfile.ReqMethod = "POST"
	return common.NewCommonClient(credential, CVMRegion, cpf), nil
}

// ---------- Check ClawPro Agent Role ----------

// CheckClawProAgentRoleBound 检查当前账号是否已绑定 CVM_QCSLinkedRoleInClawProAgent 服务角色。
//
// 通过 STS AssumeRole 尝试扮演该角色来判断：
//   - 成功且凭证有效 → 返回 (true, nil)
//   - AssumeRole 失败 → 返回 (false, nil)（角色不存在属于正常业务结果，不视为错误）
//   - 凭证未配置或客户端创建失败等 → 返回 (false, error)
//
// 此函数供 HandleCheckClawProAgentRole（HTTP handler）和 task 包共同调用。
func CheckClawProAgentRoleBound(ctx context.Context) (bool, error) {
	config := model.GetSiteConfig(ctx)
	if config.AgentCamRoleSecretId == "" || config.AgentCamRoleSecretKey == "" {
		return false, hcommon.I18nError(i18n.MsgAgentCamRoleSecretNotConfigured)
	}

	slog.Info("[CheckRole] 使用 AgentCam 凭证", "AgentCamRoleSecretId", config.AgentCamRoleSecretId)

	credential := common.NewCredential(config.AgentCamRoleSecretId, config.AgentCamRoleSecretKey)
	cpf := profile.NewClientProfile()
	client, err := sts.NewClient(credential, CVMRegion, cpf)
	if err != nil {
		return false, hcommon.I18nRichError(err, i18n.MsgSTSCreateClientFailed)
	}

	req := sts.NewAssumeRoleRequest()
	roleArn := fmt.Sprintf("qcs::cam::uin/%s:role/tencentcloudServiceRoleName/%s", hcommon.CVMUinFromCtx(ctx), AgentCamRoleName)
	req.RoleArn = &roleArn
	sessionName := "checkRole"
	req.RoleSessionName = &sessionName
	var duration uint64 = 7200
	req.DurationSeconds = &duration

	slog.Info("[CheckRole] AssumeRole 请求", "roleArn", roleArn)

	resp, err := client.AssumeRole(req)
	if err != nil {
		slog.Info("[CheckRole] AssumeRole 失败，角色不存在", "error", err)
		return false, nil
	}

	reqId := ""
	if resp.Response != nil && resp.Response.RequestId != nil {
		reqId = *resp.Response.RequestId
	}
	slog.Info("[CheckRole] AssumeRole 响应", "requestId", reqId)

	cred := resp.Response.Credentials
	if cred == nil || cred.TmpSecretId == nil || cred.TmpSecretKey == nil || cred.Token == nil {
		slog.Info("[CheckRole] AssumeRole 返回凭证为空，角色可能异常", "requestId", reqId)
		return false, nil
	}

	slog.Info("[CheckRole] AssumeRole 成功，角色存在", "roleArn", roleArn, "requestId", reqId)
	return true, nil
}

// HandleCheckClawProAgentRole 通过 AssumeRole 检查当前账号是否存在 CVM_QCSLinkedRoleInClawProAgent 服务角色。
//
// 逻辑：尝试调用 STS AssumeRole 扮演该角色，
//   - 成功 → 角色存在，返回 { "has_role": true }
//   - 失败 → 角色不存在，返回 { "has_role": false }
func HandleCheckClawProAgentRole(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	hasRole, err := CheckClawProAgentRoleBound(r.Context())
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	if hasRole {
		jsonOK(w, map[string]interface{}{
			"has_role": true,
			"message":  i18n.T(r.Context(), i18n.MsgCLSRoleExists, AgentCamRoleName),
		})
	} else {
		jsonOK(w, map[string]interface{}{
			"has_role": false,
			"message":  i18n.T(r.Context(), i18n.MsgCLSRoleNotExistOrCredErr, AgentCamRoleName),
		})
	}
}

// clsServiceStatus 表示 CLS 服务查询结果。
type clsServiceStatus struct {
	Opened    bool   // 是否已开通
	Status    int    // 原始 Status 值
	RequestId string // 请求 ID
}

// getClsServiceStatus 查询当前账号的 CLS 服务开通状态。
//
// 返回值：
//   - Status == 0 → Opened = true（已开通）
//   - Status == 1 → Opened = false（未开通）
func getClsServiceStatus(client *common.Client) (*clsServiceStatus, error) {
	getReq := tchttp.NewCommonRequest(clsService, clsVersion, "GetClsService")
	if err := getReq.SetActionParameters("{}"); err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgSetClsServiceParamsFailed)
	}

	slog.Info("[CLS] GetClsService 请求", "action", "GetClsService", "params", "{}")

	getResp := tchttp.NewCommonResponse()
	if err := client.Send(getReq, getResp); err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgQueryClsServiceCallFailed)
	}

	slog.Info("[CLS] GetClsService 响应", "body", string(getResp.GetBody()))

	var result struct {
		Response struct {
			Status    *int   `json:"Status"`
			RequestId string `json:"RequestId"`
			Error     *struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error"`
		} `json:"Response"`
	}
	if err := json.Unmarshal(getResp.GetBody(), &result); err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgParseClsServiceRespFailed)
	}

	slog.Info("[CLS] GetClsService requestId", "requestId", result.Response.RequestId)

	if result.Response.Error != nil {
		return nil, hcommon.I18nError(i18n.MsgClsServiceStatusError,
			result.Response.RequestId, result.Response.Error.Code, result.Response.Error.Message)
	}

	if result.Response.Status == nil {
		return nil, hcommon.I18nError(i18n.MsgClsServiceStatusEmpty)
	}

	return &clsServiceStatus{
		Opened:    *result.Response.Status == 0,
		Status:    *result.Response.Status,
		RequestId: result.Response.RequestId,
	}, nil
}

// openClsService 调用 OpenClsService 开通 CLS 日志服务。
func openClsService(client *common.Client) (requestId string, err error) {
	openReq := tchttp.NewCommonRequest(clsService, clsVersion, "OpenClsService")
	if err := openReq.SetActionParameters("{}"); err != nil {
		return "", hcommon.I18nRichError(err, i18n.MsgCLSOpenServiceFailed)
	}

	slog.Info("[CLS] OpenClsService 请求", "action", "OpenClsService", "params", "{}")

	openResp := tchttp.NewCommonResponse()
	if err := client.Send(openReq, openResp); err != nil {
		return "", hcommon.I18nRichError(err, i18n.MsgCLSOpenServiceFailed)
	}

	slog.Info("[CLS] OpenClsService 响应", "body", string(openResp.GetBody()))

	var result struct {
		Response struct {
			RequestId string `json:"RequestId"`
			Error     *struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error"`
		} `json:"Response"`
	}
	if err := json.Unmarshal(openResp.GetBody(), &result); err != nil {
		return "", hcommon.I18nRichError(err, i18n.MsgCLSOpenServiceFailed)
	}

	slog.Info("[CLS] OpenClsService requestId", "requestId", result.Response.RequestId)

	if result.Response.Error != nil {
		return "", hcommon.I18nError(i18n.MsgClsServiceStatusError,
			result.Response.RequestId, result.Response.Error.Code, result.Response.Error.Message)
	}

	return result.Response.RequestId, nil
}

// CLSClawServiceResult 表示 OpenClawService 接口返回的关键字段。
type CLSClawServiceResult struct {
	MetricTopicId string // 指标主题 ID
	TopicId       string // 日志主题 ID
	TraceTopicId  string // Trace 主题 ID（仅开启 trace 时返回）
	RequestId     string
}

// openClawService 调用 CLS OpenClawService 接口，Tag 为 ClawPro，
// 返回 MetricTopicId（指标主题）、TopicId（日志主题）以及 TraceTopicId（Trace 主题）。
// EnableTrace 始终为 true，接口将同时创建 Trace 主题。
func openClawService(client *common.Client) (*CLSClawServiceResult, error) {
	type openClawServiceParams struct {
		Tag         string `json:"Tag"`
		EnableTrace bool   `json:"EnableTrace"`
	}
	p := openClawServiceParams{Tag: "ClawPro", EnableTrace: true}
	paramsBytes, err := json.Marshal(p)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgSerializeOpenClawParamsFailed)
	}
	params := string(paramsBytes)

	req := tchttp.NewCommonRequest(clsService, clsVersion, "OpenClawService")
	if err := req.SetActionParameters(params); err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgSetOpenClawParamsFailed)
	}

	slog.Info("[CLS] OpenClawService 请求", "action", "OpenClawService", "params", params)

	resp := tchttp.NewCommonResponse()
	if err := client.Send(req, resp); err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgCallOpenClawServiceFailed)
	}

	slog.Info("[CLS] OpenClawService 响应", "body", string(resp.GetBody()))

	var result struct {
		Response struct {
			MetricTopicId string `json:"MetricTopicId"`
			TopicId       string `json:"TopicId"`
			TraceTopicId  string `json:"TraceTopicId"`
			RequestId     string `json:"RequestId"`
			Error         *struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error"`
		} `json:"Response"`
	}
	if err := json.Unmarshal(resp.GetBody(), &result); err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgParseOpenClawRespFailed)
	}

	slog.Info("[CLS] OpenClawService requestId", "requestId", result.Response.RequestId)

	if result.Response.Error != nil {
		return nil, hcommon.I18nError(i18n.MsgOpenClawServiceBizError,
			result.Response.RequestId, result.Response.Error.Code, result.Response.Error.Message)
	}

	return &CLSClawServiceResult{
		MetricTopicId: result.Response.MetricTopicId,
		TopicId:       result.Response.TopicId,
		TraceTopicId:  result.Response.TraceTopicId,
		RequestId:     result.Response.RequestId,
	}, nil
}

// openClsServiceRequest 开启 CLS 服务的请求体（支持按分组开启）。
type openClsServiceRequest struct {
	ScopeType string `json:"scope_type"` // 可选：采集范围模式 "all"=全量, "group"=分组（默认根据 group_ids 推断）
	GroupIDs  []uint `json:"group_ids"`  // 可选：为空=全量开启，非空=按分组开启
}

// HandleAdminOpenClsService 确认当前账号是否已开通 CLS 日志服务，若未开通则自动开通。
//
// 支持可选 JSON body `{"group_ids": [1,2,3]}`。
// group_ids 非空时，将分组 ID 写入 CLS 采集范围（collectScope），仅对这些分组下的实例开启 CLS。
// group_ids 为空或未传时，保持全量开启语义。
//
// 流程：
//  1. 调用 getClsServiceStatus 查询 CLS 服务开通状态
//  2. 如果未开通，调用 openClsService 开通服务
//  3. 调用 OpenClawService 获取 Topic 信息
//  4. 保存 collectScope + 标记目标实例为待安装
func HandleAdminOpenClsService(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	// 解析可选的 group_ids 请求体
	var req openClsServiceRequest
	if r.Body != nil && r.ContentLength > 0 {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB 请求体大小限制
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// body 解析失败时不报错，按全量开启处理
			slog.Warn("[CLS] 解析请求体失败，按全量开启处理", "error", err)
			req.GroupIDs = nil
		}
	}

	// scope_type 未传时根据 group_ids 自动推断，兼容旧的空 body 调用
	if req.ScopeType == "" {
		if len(req.GroupIDs) > 0 {
			req.ScopeType = "group"
		} else {
			req.ScopeType = "all"
		}
	}
	// 校验 scope_type
	if req.ScopeType != "all" && req.ScopeType != "group" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidCLSScopeType))
		return
	}
	// 全量模式下忽略 group_ids
	if req.ScopeType == "all" {
		req.GroupIDs = nil
	}

	// 校验分组数量上限
	if len(req.GroupIDs) > maxScopeGroupIDs {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgCLSScopeGroupCountExceed, maxScopeGroupIDs))
		return
	}

	// 校验 group_ids
	if len(req.GroupIDs) > 0 {
		if err := validateGroupIDs(r.Context(), req.GroupIDs); err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
			return
		}
	}

	client, err := newCLSCommonClient(r.Context())
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgCLSCreateCommonClientFail))
		return
	}

	// ---------- 1. 查询 CLS 服务开通状态 ----------
	status, err := getClsServiceStatus(client)
	if err != nil {
		slog.Error("[CLS] 查询服务状态失败", "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	if !status.Opened {
		// ---------- 2. 未开通，调用 openClsService 开通服务 ----------
		slog.Info("[CLS] CLS 服务未开通，开始开通", "status", status.Status)

		_, err := openClsService(client)
		if err != nil {
			slog.Error("[CLS] 开通服务失败", "error", err)
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgCLSOpenServiceFailed))
			return
		}
		slog.Info("[CLS] CLS 服务开通成功")
	} else {
		slog.Info("[CLS] CLS 服务已开通", "requestId", status.RequestId)
	}

	// ---------- 3. 调用 OpenClawService 获取 Topic 信息 ----------
	clawResult, err := openClawService(client)
	if err != nil {
		slog.Error("[CLS] 调用 OpenClawService 失败", "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	slog.Info("[CLS] OpenClawService 成功",
		"metricTopicId", clawResult.MetricTopicId,
		"topicId", clawResult.TopicId,
		"traceTopicId", clawResult.TraceTopicId,
		"requestId", clawResult.RequestId,
	)

	// 更新本地 CLSEnabled 状态 & ScopeMode
	if err := model.UpdateSiteConfig(r.Context(), map[string]interface{}{
		"cls_enabled":    1,
		"cls_scope_mode": req.ScopeType,
	}); err != nil {
		// 只打印错误
		slog.Error("[CLS] 更新本地 CLSEnabled 状态失败", "error", err)
	}

	// ---------- 4. 保存 collectScope ----------
	var targetInstanceCount int
	if len(req.GroupIDs) > 0 {
		// 按分组开启：保存 scope + 标记目标实例
		if err := model.AddCLSCollectScopeGroups(r.Context(), req.GroupIDs); err != nil {
			slog.Error("[CLS] 保存 CLS 采集范围失败", "error", err)
			// 不阻断主流程
		} else {
			slog.Info("[CLS] 已保存 CLS 采集范围", "group_ids", req.GroupIDs)
		}

		// 查询目标实例并标记为待安装
		instanceIDs, err := expandAndGetCVMIDs(r.Context(), req.GroupIDs)
		if err != nil {
			slog.Error("[CLS] 查询目标实例失败", "error", err)
		} else if len(instanceIDs) > 0 {
			targetInstanceCount = len(instanceIDs)
			if err := markInstancesCLSStatusSafe(r.Context(), instanceIDs, model.CLSAgentNotInstalled); err != nil {
				slog.Error("[CLS] 标记目标实例待安装失败", "error", err)
			}
			slog.Info("[CLS] 已标记目标实例为待安装", "group_ids", req.GroupIDs, "instance_count", len(instanceIDs))
		}
	} else {
		// 全量开启模式：清除已有的分组范围，确保从分组模式正确切换为全量模式。
		// 若不清除，之前残留的 scope 记录会导致定时任务仍按分组模式运行，
		// scope 外的实例无法被安装，且已安装的 scope 外实例会被错误卸载。
		if err := model.ClearCLSCollectScope(r.Context()); err != nil {
			slog.Error("[CLS] 清空采集范围失败", "error", err)
		} else {
			slog.Info("[CLS] 全量开启模式，已清除分组范围")
		}
	}

	resp := map[string]interface{}{
		"opened":          true,
		"message":         i18n.T(r.Context(), i18n.MsgCLSServiceOpened),
		"request_id":      clawResult.RequestId,
		"topic_id":        clawResult.TopicId,
		"metric_topic_id": clawResult.MetricTopicId,
		"trace_topic_id":  clawResult.TraceTopicId,
		"enable_trace":    true,
	}
	if len(req.GroupIDs) > 0 {
		resp["group_ids"] = req.GroupIDs
		resp["target_instance_count"] = targetInstanceCount
		resp["scope_type"] = "group"
	} else {
		resp["scope_type"] = "all"
	}
	jsonOK(w, resp)
}

// ---------- Close CLS Service ----------

// isCLSTopicNotExist 判断 CLS SDK 错误是否为主题不存在（ResourceNotFound.TopicNotExist）。
// 删除主题时若主题已不存在，视为幂等成功，无需报错。
func isCLSTopicNotExist(err error) bool {
	if sdkErr, ok := err.(*sdkerrors.TencentCloudSDKError); ok {
		return sdkErr.GetCode() == "ResourceNotFound.TopicNotExist"
	}
	return false
}

// clawproLogTopicName 是 CLS ClawPro 日志主题的 TopicName（BizType=0）。
// 注意：CLS TopicName 字段使用下划线，TopicId 使用中划线，此处按 TopicName 过滤。
const clawproLogTopicName = "clawpro_log_topic"

// clawproTraceTopicName 是 CLS ClawPro Trace 主题的 TopicName（BizType=0）。
// 注意：CLS TopicName 字段使用下划线，TopicId 使用中划线，此处按 TopicName 过滤。
const clawproTraceTopicName = "clawpro_trace_topic"

// clawproMetricTopicName 是 CLS ClawPro 指标主题的 TopicName（BizType=1）。
// 注意：CLS TopicName 字段使用下划线，TopicId 使用中划线，此处按 TopicName 过滤。
const clawproMetricTopicName = "clawpro_metric_topic"

// describeClawproLogTopics 调用 DescribeTopics 接口，
// 通过 topicName 精确匹配 + BizType=0 查询日志主题列表。
func describeClawproLogTopics(client *cls.Client) (*cls.DescribeTopicsResponse, error) {
	req := cls.NewDescribeTopicsRequest()
	req.Filters = []*cls.Filter{
		{
			Key:    common.StringPtr("topicName"),
			Values: common.StringPtrs([]string{clawproLogTopicName}),
		},
	}
	// PreciseSearch = 1 表示 topicName 精确匹配
	req.PreciseSearch = common.Uint64Ptr(1)
	req.BizType = common.Uint64Ptr(0)

	slog.Debug("[CLS] DescribeTopics(log) 请求", "action", "DescribeTopics", "params", req.ToJsonString())

	resp, err := client.DescribeTopics(req)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgCLSQueryLogTopicFail)
	}

	reqId := ""
	if resp.Response != nil && resp.Response.RequestId != nil {
		reqId = *resp.Response.RequestId
	}
	slog.Debug("[CLS] DescribeTopics(log) 响应", "body", resp.ToJsonString(), "requestId", reqId)

	return resp, nil
}

// describeClawproTraceTopics 调用 DescribeTopics 接口，
// 通过 topicName 精确匹配 + BizType=0 查询 Trace 主题列表。
// Trace 主题与日志主题类型相同（BizType=0），通过名称区分。
func describeClawproTraceTopics(client *cls.Client) (*cls.DescribeTopicsResponse, error) {
	req := cls.NewDescribeTopicsRequest()
	req.Filters = []*cls.Filter{
		{
			Key:    common.StringPtr("topicName"),
			Values: common.StringPtrs([]string{clawproTraceTopicName}),
		},
	}
	// PreciseSearch = 1 表示 topicName 精确匹配
	req.PreciseSearch = common.Uint64Ptr(1)
	req.BizType = common.Uint64Ptr(0)

	slog.Debug("[CLS] DescribeTopics(trace) 请求", "action", "DescribeTopics", "params", req.ToJsonString())

	resp, err := client.DescribeTopics(req)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgCLSQueryTraceTopicFail)
	}

	reqId := ""
	if resp.Response != nil && resp.Response.RequestId != nil {
		reqId = *resp.Response.RequestId
	}
	slog.Debug("[CLS] DescribeTopics(trace) 响应", "body", resp.ToJsonString(), "requestId", reqId)

	return resp, nil
}

// describeClawproMetricTopics 调用 DescribeTopics 接口，
// 通过 topicName 精确匹配 + BizType=1 查询指标主题列表。
func describeClawproMetricTopics(client *cls.Client) (*cls.DescribeTopicsResponse, error) {
	req := cls.NewDescribeTopicsRequest()
	req.Filters = []*cls.Filter{
		{
			Key:    common.StringPtr("topicName"),
			Values: common.StringPtrs([]string{clawproMetricTopicName}),
		},
	}
	// PreciseSearch = 1 表示 topicName 精确匹配
	req.PreciseSearch = common.Uint64Ptr(1)
	req.BizType = common.Uint64Ptr(1)

	slog.Debug("[CLS] DescribeTopics(metric) 请求", "action", "DescribeTopics", "params", req.ToJsonString())

	resp, err := client.DescribeTopics(req)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgCLSQueryMetricTopicFail)
	}

	reqId := ""
	if resp.Response != nil && resp.Response.RequestId != nil {
		reqId = *resp.Response.RequestId
	}
	slog.Debug("[CLS] DescribeTopics(metric) 响应", "body", resp.ToJsonString(), "requestId", reqId)

	return resp, nil
}

// closeClsServiceRequest 关闭 CLS 服务的请求参数。
type closeClsServiceRequest struct {
	// DeleteResources 是否删除 CLS 日志/指标主题资源。
	// true（默认）：删除 Topic + 卸载 Agent + 清空 scope
	// false：仅卸载 Agent + 清空 scope，保留 Topic 资源（日志数据不丢失）
	DeleteResources *bool `json:"delete_resources"`
}

// shouldDeleteResources 返回是否需要删除资源，默认为 true（向后兼容）。
func (req *closeClsServiceRequest) shouldDeleteResources() bool {
	if req.DeleteResources == nil {
		return true
	}
	return *req.DeleteResources
}

// HandleAdminCloseClsService 关闭 CLS 日志服务。
//
// 请求参数（可选）：
//   - delete_resources: bool，是否删除 CLS Topic 资源（默认 true）
//   - true: 删除日志/指标主题 + 卸载 Agent + 清空 scope（不可恢复）
//   - false: 仅卸载 Agent + 清空 scope，保留 Topic 资源
//
// 流程：
//  1. 分别查询日志主题（BizType=0）、指标主题（BizType=1）和 Trace 主题（BizType=0，按名称精确匹配）
//  2. 如果三次查询都没有查到主题，说明服务已关闭，直接返回
//  3. 如果 delete_resources=true，调用 DeleteTopic 删除对应主题
//  4. 更新 cls_enabled=0，清空 scope
func HandleAdminCloseClsService(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	// 解析可选的请求参数
	var req closeClsServiceRequest
	if r.Body != nil && r.ContentLength > 0 {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// 解析失败按默认值处理（删除资源）
			slog.Warn("[CLS] 解析关闭请求体失败，按默认模式处理", "error", err)
		}
	}

	deleteResources := req.shouldDeleteResources()
	slog.Info("[CLS] 关闭 CLS 服务", "delete_resources", deleteResources)

	// ---------- 1. 使用 DescribeTopics 查询 ClawPro 相关主题 ----------
	clsClient, err := newCLSClient(r.Context())
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgCLSCreateClientFail))
		return
	}

	// 查询日志主题（topicName 前缀 clawpro-topic + BizType=0）
	logTopicsResp, err := describeClawproLogTopics(clsClient)
	if err != nil {
		slog.Error("[CLS] 查询日志主题失败", "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	var logTopics []*cls.TopicInfo
	var logCount int64
	if logTopicsResp.Response != nil {
		logTopics = logTopicsResp.Response.Topics
		if logTopicsResp.Response.TotalCount != nil {
			logCount = *logTopicsResp.Response.TotalCount
		}
	}

	// 查询指标主题（topicName 前缀 clawpro-metric-topic + BizType=1）
	metricTopicsResp, err := describeClawproMetricTopics(clsClient)
	if err != nil {
		slog.Error("[CLS] 查询指标主题失败", "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	var metricTopics []*cls.TopicInfo
	var metricCount int64
	if metricTopicsResp.Response != nil {
		metricTopics = metricTopicsResp.Response.Topics
		if metricTopicsResp.Response.TotalCount != nil {
			metricCount = *metricTopicsResp.Response.TotalCount
		}
	}

	// 查询 Trace 主题（topicName 前缀 clawpro-trace-topic + BizType=0）
	traceTopicsResp, err := describeClawproTraceTopics(clsClient)
	if err != nil {
		slog.Error("[CLS] 查询 Trace 主题失败", "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	var traceTopics []*cls.TopicInfo
	var traceCount int64
	if traceTopicsResp.Response != nil {
		traceTopics = traceTopicsResp.Response.Topics
		if traceTopicsResp.Response.TotalCount != nil {
			traceCount = *traceTopicsResp.Response.TotalCount
		}
	}

	// 三次查询都无数据，说明服务已关闭
	if logCount == 0 && metricCount == 0 && traceCount == 0 {
		slog.Info("[CLS] DescribeTopics 未查到任何 ClawPro 主题，服务可能已关闭",
			"logTopicCount", logCount,
			"metricTopicCount", metricCount,
			"traceTopicCount", traceCount,
		)
		if err := model.UpdateSiteConfig(r.Context(), map[string]interface{}{
			"cls_enabled": 0,
		}); err != nil {
			slog.Error("[CLS] 更新本地 CLSEnabled 状态失败", "error", err)
		}
		jsonOK(w, map[string]interface{}{
			"message": i18n.T(r.Context(), i18n.MsgCLSServiceNotEnabledNoOff),
		})
		return
	}

	// 提取 TopicId
	var topicId string
	if len(logTopics) > 0 && logTopics[0].TopicId != nil {
		topicId = *logTopics[0].TopicId
	}
	var metricTopicId string
	if len(metricTopics) > 0 && metricTopics[0].TopicId != nil {
		metricTopicId = *metricTopics[0].TopicId
	}
	var traceTopicId string
	if len(traceTopics) > 0 && traceTopics[0].TopicId != nil {
		traceTopicId = *traceTopics[0].TopicId
	}

	slog.Info("[CLS] DescribeTopics 查询到 ClawPro 主题",
		"topicId", topicId,
		"metricTopicId", metricTopicId,
		"traceTopicId", traceTopicId,
		"logTopicCount", logCount,
		"metricTopicCount", metricCount,
		"traceTopicCount", traceCount,
	)

	// ---------- 2. 根据 delete_resources 决定是否删除主题 ----------
	var deletedTopics []string

	if deleteResources {
		if topicId != "" {
			delReq := cls.NewDeleteTopicRequest()
			delReq.TopicId = common.StringPtr(topicId)
			slog.Info("[CLS] 删除日志主题", "topicId", topicId)
			delResp, err := clsClient.DeleteTopic(delReq)
			if err != nil {
				if isCLSTopicNotExist(err) {
					slog.Warn("[CLS] 日志主题已不存在，跳过删除", "topicId", topicId)
				} else {
					slog.Error("[CLS] 删除日志主题失败", "topicId", topicId, "error", err)
					writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgCLSDeleteLogTopicFail, topicId))
					return
				}
			} else {
				delReqId := ""
				if delResp.Response != nil && delResp.Response.RequestId != nil {
					delReqId = *delResp.Response.RequestId
				}
				slog.Info("[CLS] 删除日志主题成功", "topicId", topicId, "requestId", delReqId)
			}
			deletedTopics = append(deletedTopics, topicId)
		}

		if metricTopicId != "" {
			delReq := cls.NewDeleteTopicRequest()
			delReq.TopicId = common.StringPtr(metricTopicId)
			slog.Info("[CLS] 删除指标主题", "metricTopicId", metricTopicId)
			delResp, err := clsClient.DeleteTopic(delReq)
			if err != nil {
				if isCLSTopicNotExist(err) {
					slog.Warn("[CLS] 指标主题已不存在，跳过删除", "metricTopicId", metricTopicId)
				} else {
					slog.Error("[CLS] 删除指标主题失败", "metricTopicId", metricTopicId, "error", err)
					writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgCLSDeleteMetricTopicFail, metricTopicId))
					return
				}
			} else {
				delReqId := ""
				if delResp.Response != nil && delResp.Response.RequestId != nil {
					delReqId = *delResp.Response.RequestId
				}
				slog.Info("[CLS] 删除指标主题成功", "metricTopicId", metricTopicId, "requestId", delReqId)
			}
			deletedTopics = append(deletedTopics, metricTopicId)
		}

		if traceTopicId != "" {
			delReq := cls.NewDeleteTopicRequest()
			delReq.TopicId = common.StringPtr(traceTopicId)
			slog.Info("[CLS] 删除 Trace 主题", "traceTopicId", traceTopicId)
			delResp, err := clsClient.DeleteTopic(delReq)
			if err != nil {
				if isCLSTopicNotExist(err) {
					slog.Warn("[CLS] Trace 主题已不存在，跳过删除", "traceTopicId", traceTopicId)
				} else {
					slog.Error("[CLS] 删除 Trace 主题失败", "traceTopicId", traceTopicId, "error", err)
					writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgCLSDeleteTraceTopicFail, traceTopicId))
					return
				}
			} else {
				delReqId := ""
				if delResp.Response != nil && delResp.Response.RequestId != nil {
					delReqId = *delResp.Response.RequestId
				}
				slog.Info("[CLS] 删除 Trace 主题成功", "traceTopicId", traceTopicId, "requestId", delReqId)
			}
			deletedTopics = append(deletedTopics, traceTopicId)
		}
	} else {
		slog.Info("[CLS] 保留 Topic 资源，仅关闭服务并卸载 Agent",
			"topicId", topicId,
			"metricTopicId", metricTopicId,
			"traceTopicId", traceTopicId,
		)
	}

	// 更新本地 CLSEnabled 状态为未开通
	if err := model.UpdateSiteConfig(r.Context(), map[string]interface{}{
		"cls_enabled": 0,
	}); err != nil {
		slog.Error("[CLS] 更新本地 CLSEnabled 状态失败", "error", err)
	}

	// 清空 CLS 采集范围（collectScope）
	if err := model.ClearCLSCollectScope(r.Context()); err != nil {
		slog.Error("[CLS] 清空 CLS 采集范围失败", "error", err)
	} else {
		slog.Info("[CLS] 已清空 CLS 采集范围")
	}

	// 将所有待安装实例（status=0）标记为跳过（status=4），避免遗留垃圾状态
	if result := model.DB(r.Context()).Model(&model.Instance{}).
		Where("cls_agent_status = ? AND instance_id != ''", model.CLSAgentNotInstalled).
		Updates(map[string]interface{}{
			"cls_agent_status":    model.CLSAgentSkipped,
			"cls_agent_status_at": nil,
		}); result.Error != nil {
		slog.Error("[CLS] 关闭时标记待安装实例为跳过失败", "error", result.Error)
	} else if result.RowsAffected > 0 {
		slog.Info("[CLS] 已将待安装实例标记为跳过", "count", result.RowsAffected)
	}

	resp := map[string]interface{}{
		"delete_resources": deleteResources,
	}
	if deleteResources {
		resp["message"] = fmt.Sprintf("已关闭 CLS 服务并删除 %d 个主题", len(deletedTopics))
		resp["deleted_topics"] = deletedTopics
	} else {
		resp["message"] = "已关闭 CLS 服务（Agent 将由后台任务卸载），Topic 资源已保留"
		resp["preserved_topics"] = []string{topicId, metricTopicId, traceTopicId}
	}
	jsonOK(w, resp)
}

// ---------- Describe ClawPro Topics Status ----------

// HandleAdminClsStatus 查询 ClawPro 相关主题的详细状态信息。
// 参数：biz_type（0=日志主题，1=指标主题），通过 query string 传入。
func HandleAdminClsStatus(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	// 解析 biz_type 参数，默认为 0
	bizTypeStr := r.URL.Query().Get("biz_type")
	var bizType uint64
	if bizTypeStr == "1" {
		bizType = 1
	}

	client, err := newCLSClient(r.Context())
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgCLSCreateClientFail))
		return
	}

	topics, totalCount, err := describeClawProTopics(client, bizType)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 将 TopicInfo 转换为前端友好的 map 列表
	var topicList []map[string]interface{}
	for _, t := range topics {
		item := map[string]interface{}{}
		if t.TopicId != nil {
			item["TopicId"] = *t.TopicId
		}
		if t.TopicName != nil {
			item["TopicName"] = *t.TopicName
		}
		if t.LogsetId != nil {
			item["LogsetId"] = *t.LogsetId
		}
		if t.BizType != nil {
			item["BizType"] = *t.BizType
		}
		if t.PartitionCount != nil {
			item["PartitionCount"] = *t.PartitionCount
		}
		if t.Index != nil {
			item["Index"] = *t.Index
		}
		if t.Status != nil {
			item["Status"] = *t.Status
		}
		if t.AssumerUin != nil {
			item["AssumerUin"] = *t.AssumerUin
		}
		if t.AssumerName != nil {
			item["AssumerName"] = *t.AssumerName
		}
		if t.CreateTime != nil {
			item["CreateTime"] = *t.CreateTime
		}
		if t.RoleName != nil {
			item["RoleName"] = *t.RoleName
		}
		if t.AutoSplit != nil {
			item["AutoSplit"] = *t.AutoSplit
		}
		if t.MaxSplitPartitions != nil {
			item["MaxSplitPartitions"] = *t.MaxSplitPartitions
		}
		if t.StorageType != nil {
			item["StorageType"] = *t.StorageType
		}
		if t.Period != nil {
			item["Period"] = *t.Period
		}
		if t.SubAssumerName != nil {
			item["SubAssumerName"] = *t.SubAssumerName
		}
		if t.Describes != nil {
			item["Describes"] = *t.Describes
		}
		if t.HotPeriod != nil {
			item["HotPeriod"] = *t.HotPeriod
		}
		if t.IsWebTracking != nil {
			item["IsWebTracking"] = *t.IsWebTracking
		}
		if t.Tags != nil {
			var tagList []map[string]string
			for _, tag := range t.Tags {
				tagItem := map[string]string{}
				if tag.Key != nil {
					tagItem["Key"] = *tag.Key
				}
				if tag.Value != nil {
					tagItem["Value"] = *tag.Value
				}
				tagList = append(tagList, tagItem)
			}
			item["Tags"] = tagList
		}
		topicList = append(topicList, item)
	}

	jsonOK(w, map[string]interface{}{
		"biz_type":    bizType,
		"total_count": totalCount,
		"topics":      topicList,
	})
}

// ---------- Exported helpers for task package ----------

// describeClawProTopics 调用 DescribeTopics 接口，通过 assumerName=ClawPro 和指定 BizType 查询主题。
// bizType: 0=日志主题，1=指标主题。
// 返回匹配到的主题列表和 TotalCount。
func describeClawProTopics(client *cls.Client, bizType uint64) ([]*cls.TopicInfo, int64, error) {
	req := cls.NewDescribeTopicsRequest()
	req.Filters = []*cls.Filter{
		{
			Key:    common.StringPtr("assumerName"),
			Values: common.StringPtrs([]string{"ClawPro"}),
		},
	}
	req.BizType = common.Uint64Ptr(bizType)

	slog.Info("[CheckCLSClawService] DescribeTopics 请求", "bizType", bizType, "params", req.ToJsonString())

	resp, err := client.DescribeTopics(req)
	if err != nil {
		return nil, 0, hcommon.I18nRichError(err, i18n.MsgCLSQueryTopicFail, bizType)
	}

	reqId := ""
	if resp.Response != nil && resp.Response.RequestId != nil {
		reqId = *resp.Response.RequestId
	}

	var totalCount int64
	if resp.Response != nil && resp.Response.TotalCount != nil {
		totalCount = *resp.Response.TotalCount
	}

	slog.Info("[CheckCLSClawService] DescribeTopics 响应",
		"bizType", bizType,
		"totalCount", totalCount,
		"requestId", reqId,
		// "body", resp.ToJsonString(),
	)

	var topics []*cls.TopicInfo
	if resp.Response != nil {
		topics = resp.Response.Topics
	}
	return topics, totalCount, nil
}

// ---------- Modify Instances CamRole ----------

// modifyInstancesCamRoleRequest 表示 ModifyInstancesCamRole 接口的请求参数。
type modifyInstancesCamRoleRequest struct {
	InstanceIds []string `json:"instance_ids"`
}

// modifyInstancesCamRoleBatch 调用 CVM ModifyInstancesAttribute 为一批实例绑定 CamRole。
// 这是 HandleModifyInstancesCamRole 和 ModifyInstancesCamRole 的公共核心逻辑。
//
// 使用 CVM SDK 原生 Client 调用（ModifyInstancesAttribute 是 CVM 已封装的标准接口）。
//
// 参数：
//   - client: CVM SDK Client
//   - instanceIds: 本批次要处理的实例 ID 列表（调用方负责分批，单次最多 100 个）
//   - batchLabel: 用于日志标识的批次描述（如 "all" 或 "[1-100]/200"）
//
// 返回值：
//   - requestId: API 返回的 RequestId
//   - err: 错误信息
func modifyInstancesCamRoleBatch(client *cvm.Client, instanceIds []string, batchLabel string) (requestId string, err error) {
	request := cvm.NewModifyInstancesAttributeRequest()
	request.InstanceIds = common.StringPtrs(instanceIds)
	request.CamRoleName = common.StringPtr(AgentCamRoleName)
	request.CamRoleType = common.StringPtr("service_linked")

	slog.Info("[ModifyInstancesCamRole] 发送请求",
		"action", "ModifyInstancesAttribute",
		"batch", batchLabel,
		"instanceIds", instanceIds,
		"camRoleName", AgentCamRoleName,
		"camRoleType", "service_linked",
	)

	response, err := client.ModifyInstancesAttribute(request)
	if err != nil {
		return "", hcommon.I18nRichError(err, i18n.MsgCLSCallModifyAttrFail)
	}

	reqId := ""
	if response.Response != nil && response.Response.RequestId != nil {
		reqId = *response.Response.RequestId
	}

	slog.Info("[ModifyInstancesCamRole] 批次成功",
		"batch", batchLabel,
		"requestId", reqId,
	)
	return reqId, nil
}

// HandleModifyInstancesCamRole 批量为 CVM 实例绑定 ClawPro Agent 服务角色。
//
// 包装 CVM 云 API ModifyInstancesAttribute，固定设置：
//   - CamRoleName = CVM_QCSLinkedRoleInClawProAgent
//   - CamRoleType = service_linked
//
// 请求参数（JSON Body）：
//
//	{ "instance_ids": ["ins-xxx", "ins-yyy"] }
func HandleModifyInstancesCamRole(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	var req modifyInstancesCamRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgBadRequest))
		return
	}
	if len(req.InstanceIds) == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInstanceIdsCannotBeEmpty))
		return
	}

	// 本地 agent 实例的 instance_id 不是 CVM 格式，不能调 ModifyInstancesAttribute。
	var localCount int64
	if err := model.DB(r.Context()).Model(&model.Instance{}).
		Where("instance_id IN ? AND source = ?", req.InstanceIds, model.InstanceSourceLocal).
		Count(&localCount).Error; err != nil {
		slog.Error("[ModifyInstancesCamRole] 检查本地实例失败", "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryInstanceFailed))
		return
	}
	if localCount > 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgLocalInstanceUnsupportedOp))
		return
	}

	slog.Info("[ModifyInstancesCamRole] 请求", "instanceIds", req.InstanceIds)

	client, err := NewCVMClient(r.Context())
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	reqId, err := modifyInstancesCamRoleBatch(client, req.InstanceIds, "all")
	if err != nil {
		slog.Error("[ModifyInstancesCamRole] API 返回错误", "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgCLSCallModifyAttrFail))
		return
	}

	slog.Info("[ModifyInstancesCamRole] 成功",
		"instanceIds", req.InstanceIds,
		"camRoleName", AgentCamRoleName,
		"requestId", reqId,
	)

	jsonOK(w, map[string]interface{}{
		"message":       i18n.T(r.Context(), i18n.MsgCLSBindRoleSuccess, len(req.InstanceIds), AgentCamRoleName),
		"request_id":    reqId,
		"instance_ids":  req.InstanceIds,
		"cam_role_name": AgentCamRoleName,
	})
}

// ModifyInstancesCamRole 批量为 CVM 实例绑定 ClawPro Agent 服务角色（供 task 包调用）。
// 与 HandleModifyInstancesCamRole 共用同一核心逻辑（modifyInstancesCamRoleBatch），固定设置：
//   - CamRoleName = CVM_QCSLinkedRoleInClawProAgent
//   - CamRoleType = service_linked
//
// CVM ModifyInstancesAttribute 单次最多支持 100 个实例，此函数自动分批调用。
func ModifyInstancesCamRole(ctx context.Context, instanceIds []string) error {
	if len(instanceIds) == 0 {
		return nil
	}

	client, err := NewCVMClient(ctx)
	if err != nil {
		return err
	}

	// ModifyInstancesAttribute 单次最多处理 100 个实例
	const batchSize = 100
	for i := 0; i < len(instanceIds); i += batchSize {
		end := i + batchSize
		if end > len(instanceIds) {
			end = len(instanceIds)
		}
		batch := instanceIds[i:end]
		batchLabel := fmt.Sprintf("[%d-%d]/%d", i+1, end, len(instanceIds))

		if _, err := modifyInstancesCamRoleBatch(client, batch, batchLabel); err != nil {
			return err
		}
	}

	return nil
}

// CheckCLSClawServiceOpened 调用 DescribeTopics 接口检查 CLS ClawPro 服务是否仍然开通。
//
// 判断逻辑：
//   - 分别以 BizType=0（日志主题）和 BizType=1（指标主题）+ Filter assumerName=ClawPro 查询
//   - 若两次查询都能查到 1 条数据，说明 CLS 服务已开通
//   - 只要有一个查不到数据，就认为 CLS 服务已关闭
//   - 如果两次查询都有数据但数量不一致，打印 warn 日志
//   - 始终查询 BizType=2（Trace 主题）并提取 TraceTopicId
//
// 返回值：
//   - result: 包含 TopicId（日志主题）、MetricTopicId（指标主题）和 TraceTopicId（Trace 主题）；服务未开通时为 nil
//   - err: 查询过程中的错误（创建客户端失败 / 接口调用失败等）
func CheckCLSClawServiceOpened(ctx context.Context) (result *CLSClawServiceResult, err error) {
	slog.Info("[CheckCLSClawService] 开始检查 CLS ClawPro 服务状态")

	client, err := newCLSClient(ctx)
	if err != nil {
		slog.Error("[CheckCLSClawService] 创建 CLS Client 失败", "error", err)
		return nil, err
	}

	// 查询日志主题（topicName 前缀 clawpro-topic + BizType=0）
	logTopicsResp, rerr := describeClawproLogTopics(client)
	if rerr != nil {
		slog.Error("[CheckCLSClawService] 查询日志主题失败", "error", rerr)
		return nil, rerr
	}
	var logTopics []*cls.TopicInfo
	var logCount int64
	if logTopicsResp.Response != nil {
		logTopics = logTopicsResp.Response.Topics
		if logTopicsResp.Response.TotalCount != nil {
			logCount = *logTopicsResp.Response.TotalCount
		}
	}

	// 查询指标主题（topicName 前缀 clawpro-metric-topic + BizType=1）
	metricTopicsResp, rerr := describeClawproMetricTopics(client)
	if rerr != nil {
		slog.Error("[CheckCLSClawService] 查询指标主题失败", "error", rerr)
		return nil, rerr
	}
	var metricTopics []*cls.TopicInfo
	var metricCount int64
	if metricTopicsResp.Response != nil {
		metricTopics = metricTopicsResp.Response.Topics
		if metricTopicsResp.Response.TotalCount != nil {
			metricCount = *metricTopicsResp.Response.TotalCount
		}
	}

	// 两次查询都有数据但数量不一致，打印 warn 日志
	if logCount != metricCount {
		slog.Warn("[CheckCLSClawService] 日志主题与指标主题数量不一致",
			"logTopicCount", logCount,
			"metricTopicCount", metricCount,
		)
	}

	// 只要有一个查不到数据，就认为 CLS 服务已关闭
	if logCount == 0 || metricCount == 0 {
		slog.Info("[CheckCLSClawService] CLS ClawPro 服务未开通或已关闭",
			"logTopicCount", logCount,
			"metricTopicCount", metricCount,
		)
		return nil, nil
	}

	// 提取 TopicId
	var topicId string
	if len(logTopics) > 0 && logTopics[0].TopicId != nil {
		topicId = *logTopics[0].TopicId
	}
	var metricTopicId string
	if len(metricTopics) > 0 && metricTopics[0].TopicId != nil {
		metricTopicId = *metricTopics[0].TopicId
	}

	// 始终查询 Trace 主题（与日志主题同类型 BizType=0，通过 topicName 精确匹配）
	// Trace 默认开启，未获取到视为服务未开通
	traceTopicsResp, traceErr := describeClawproTraceTopics(client)
	if traceErr != nil {
		slog.Error("[CheckCLSClawService] 查询 Trace 主题失败", "error", traceErr)
		return nil, traceErr
	}
	var traceTopics []*cls.TopicInfo
	var traceCount int64
	if traceTopicsResp.Response != nil {
		traceTopics = traceTopicsResp.Response.Topics
		if traceTopicsResp.Response.TotalCount != nil {
			traceCount = *traceTopicsResp.Response.TotalCount
		}
	}
	if traceCount == 0 || len(traceTopics) == 0 || traceTopics[0].TopicId == nil {
		slog.Warn("[CheckCLSClawService] Trace 主题不存在，主题信息不完整，跳过本轮",
			"traceTopicCount", traceCount,
		)
		return &CLSClawServiceResult{
			TopicId:       topicId,
			MetricTopicId: metricTopicId,
			TraceTopicId:  "",
		}, nil
	}
	traceTopicId := *traceTopics[0].TopicId

	slog.Info("[CheckCLSClawService] CLS ClawPro 服务已开通",
		"topicId", topicId,
		"metricTopicId", metricTopicId,
		"traceTopicId", traceTopicId,
		"logTopicCount", logCount,
		"metricTopicCount", metricCount,
	)
	return &CLSClawServiceResult{
		TopicId:       topicId,
		MetricTopicId: metricTopicId,
		TraceTopicId:  traceTopicId,
	}, nil
}
