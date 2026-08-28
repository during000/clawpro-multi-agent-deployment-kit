package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tat "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tat/v20201028"
)

// ============================================================================
// Handler: POST /admin/agent-commands/agent-status
// ============================================================================

// agentStatusReq POST 请求体。
type agentStatusReq struct {
	InstanceIDs []string `json:"instance_ids"`
}

// agentStatusItem 单个实例的 TAT Agent 状态。
//
// 字段尽量贴合腾讯云 DescribeAutomationAgentStatus 接口原样字段，便于前端
// 直接展示；agent_status 在 TAT 没返回时补 "Unknown"。
type agentStatusItem struct {
	InstanceID        string `json:"instance_id"`
	AgentStatus       string `json:"agent_status"` // Online / Offline / Unknown
	Version           string `json:"version,omitempty"`
	LastHeartbeatTime string `json:"last_heartbeat_time,omitempty"` // TAT 原样字符串
	Environment       string `json:"environment,omitempty"`         // Linux / Windows
}

type agentStatusResp struct {
	Agents []agentStatusItem `json:"agents"`
}

// describeTATAgentStatusFn 是 TAT SDK 调用的可测试 seam；测试中替换为 mock。
//
// 默认实现直接调用 DescribeAutomationAgentStatusWithContext，把 SDK 的
// AutomationAgentInfo 列表透传出来；handler 负责字段映射 + Unknown 兜底。
var describeTATAgentStatusFn = func(ctx context.Context, instanceIDs []string) ([]*tat.AutomationAgentInfo, error) {
	client, err := NewTATClient(ctx)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgCreateTATClientFailed)
	}
	req := tat.NewDescribeAutomationAgentStatusRequest()
	req.InstanceIds = common.StringPtrs(instanceIDs)
	resp, err := client.DescribeAutomationAgentStatusWithContext(ctx, req)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgQueryTATAgentStatusFailed)
	}
	if resp == nil || resp.Response == nil {
		return nil, nil
	}
	return resp.Response.AutomationAgentSet, nil
}

// HandleAgentCommandAgentStatus POST /admin/agent-commands/agent-status
//
// 查询给定实例列表的 TAT Agent 客户端状态。原样转发腾讯云
// DescribeAutomationAgentStatus 接口字段，对没在 TAT 返回结果中出现的实例补一行
// agent_status="Unknown"，前端按 instance_ids 顺序就能直接渲染。
//
// 限流：单次最多 100 个 instance_id，与 TAT API 单批上限对齐。
func HandleAgentCommandAgentStatus(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, hcommon.I18nError(i18n.MsgMethodNotAllowed))
		return
	}
	if !requireAdmin(w, r) {
		return
	}

	var req agentStatusReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}

	// 去空 + 去重，保留第一次出现的顺序，便于前端直接按响应渲染。
	seen := make(map[string]struct{}, len(req.InstanceIDs))
	cleanIDs := make([]string, 0, len(req.InstanceIDs))
	for _, id := range req.InstanceIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		cleanIDs = append(cleanIDs, id)
	}
	if len(cleanIDs) == 0 {
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgBadRequestMissingParamWithKey, "instance_ids"))
		return
	}
	if len(cleanIDs) > model.AgentDispatchMaxTargets {
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgTooManyInstanceIDs, model.AgentDispatchMaxTargets).WithPrefix("too_many_instance_ids"))
		return
	}

	agents, err := describeTATAgentStatusFn(r.Context(), cleanIDs)
	if err != nil {
		rerr := hcommon.EnsureRichErrorOrPanic(err)
		// 502 而非 500：失败来自上游 TAT API，不是本服务内部错误。
		writeError(w, r, http.StatusBadGateway, rerr.WithPrefix("tat_describe_agent_failed"))
		return
	}

	// TAT 返回的列表按 instance_id 索引，便于按入参顺序找回。
	byID := make(map[string]*tat.AutomationAgentInfo, len(agents))
	for _, a := range agents {
		if a == nil || a.InstanceId == nil {
			continue
		}
		byID[*a.InstanceId] = a
	}

	items := make([]agentStatusItem, 0, len(cleanIDs))
	for _, id := range cleanIDs {
		a, ok := byID[id]
		if !ok || a == nil {
			items = append(items, agentStatusItem{
				InstanceID:  id,
				AgentStatus: "Unknown",
			})
			continue
		}
		item := agentStatusItem{InstanceID: id, AgentStatus: "Unknown"}
		if a.AgentStatus != nil && *a.AgentStatus != "" {
			item.AgentStatus = *a.AgentStatus
		}
		if a.Version != nil {
			item.Version = *a.Version
		}
		if a.LastHeartbeatTime != nil {
			item.LastHeartbeatTime = *a.LastHeartbeatTime
		}
		if a.Environment != nil {
			item.Environment = *a.Environment
		}
		items = append(items, item)
	}

	jsonOK(w, agentStatusResp{Agents: items})
}
