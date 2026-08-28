package controller

import (
	"net/http"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// LocalAgentTypeResponse /admin/local-agent-types 单个本地 agent 类型条目。
//
// 与 /admin/agent-types 的 AgentTypeResponse 故意分开：本地 agent 没有镜像、
// 不参与 hatchery 创实例流程、所有 supports_* 能力位都不适用，硬塞过去会让
// 前端看到一堆灰按钮误导用户。详见 iwiki §5.C.x。
type LocalAgentTypeResponse struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}
// HandleAdminLocalAgentTypes 列举本期支持的本地 agent 类型。
//
// Path: GET /admin/local-agent-types
// 鉴权: requireAdmin
// 入参: 无
// 返回: { "local_agent_types": [{ code, name, description }, ...] }
//
// 一期 codebuddy/workbuddy 写死，与 reporter 接口的 agent_type 校验同源
// （都派生自 localAgentTypes 数组）。二期扩类型时只需改 localAgentTypes，
// 本接口与 reporter 校验同时生效。
//
// 不带 instance_count 等聚合字段——前端要数字调 /admin/instances?source=local
// 自己 group by；这个接口的语义是"我支持哪些 type"，不是"我有哪些实例"。
//
// feature_allowlist 守卫 (type='local-agent')：
//   - 空表 → 全开（IsFeatureAllowed 语义）
//   - 有记录但当前 tenant 未命中 → **返回空数组**（不返 403）
//     语义与 auth.go 的 reveal 路径一致：静默降级，让前端用同一个 200 响应
//     判断「本地 agent 功能是否可见」，不用区分「未启用」与「无权限」。
//   - AdminToken（启动参数配置的超管 token）：绕过白名单，始终返回全量。
//     否则线上超管一开白名单就全瞎，运维体验差。判定复用 isAdminTokenRequest
//     （直接对比 header token），比看 user.Username 更 robust。
func HandleAdminLocalAgentTypes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	// AdminToken 直通全量；否则按 tenant identifier 过白名单。
	if !isAdminTokenRequest(r) {
		// requireAdmin 已经放过，这里 RequestUser 拿当前 tenant admin 的 user。
		// 拿不到（异常边界）走「未命中」分支返空数组，与其它分支保持 200 契约。
		user, _ := RequestUser(r)
		if user == nil {
			jsonOK(w, map[string]any{"local_agent_types": []LocalAgentTypeResponse{}})
			return
		}
		allowed, err := model.IsFeatureAllowed(r.Context(), model.FeatureAllowlistTypeLocalAgent, user.Identifier)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError,
				hcommon.I18nRichError(err, i18n.MsgInternalError))
			return
		}
		if !allowed {
			// 静默降级：返 200 + 空数组，让前端「看不到本地 agent 类型」= 「没启用」。
			jsonOK(w, map[string]any{"local_agent_types": []LocalAgentTypeResponse{}})
			return
		}
	}

	resp := make([]LocalAgentTypeResponse, 0, len(localAgentTypes))
	for _, t := range localAgentTypes {
		resp = append(resp, LocalAgentTypeResponse{
			Code:        t.Code,
			Name:        t.Name,
			Description: t.Description,
		})
	}
	jsonOK(w, map[string]any{
		"local_agent_types": resp,
	})
}
