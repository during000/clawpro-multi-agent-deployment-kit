package controller

import (
	"log/slog"
	"net/http"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/skillhubclient"
)

// 本文件包含所有技能相关 handler 的 SkillHub 实现。
// 与 admin_skills.go 中的本地 DB handler 一一对应，
// 通过 WithSkillHubProxy 装饰器在 main.go 中按灰度切换。
//
// 扩展新接口模式：
//   1. 在 skillhubclient/client.go 加 API 调用方法
//   2. 在 skillhubclient/adapter.go 加格式转换函数
//   3. 在本文件加 XxxViaSkillHub handler
//   4. 在 main.go 路由注册用 WithSkillHubProxy(local, skillhub)

// skillHubClientOrFail 获取 SkillHub 客户端，失败时写 502 错误并返回 nil。
// 所有 ViaSkillHub handler 统一调用此函数，避免重复的 error handling 模板。
func skillHubClientOrFail(w http.ResponseWriter, r *http.Request) *skillhubclient.Client {
	client, err := getSkillHubClient(r)
	if err != nil {
		slog.ErrorContext(r.Context(), "[SkillHub] 创建客户端失败，不降级", "error", err)
		writeError(w, r, http.StatusBadGateway, hcommon.I18nRichError(err, i18n.MsgInternalError))
		return nil
	}
	return client
}

// skillHubAPIFail SkillHub API 调用失败的统一错误处理。
func skillHubAPIFail(w http.ResponseWriter, r *http.Request, op string, err error) {
	slog.ErrorContext(r.Context(), "[SkillHub] API 调用失败", "op", op, "error", err)
	writeError(w, r, http.StatusBadGateway, hcommon.I18nRichError(err, i18n.MsgInternalError))
}

// ── 技能列表 ──

// HandleAdminSkillsViaSkillHub 技能列表（SkillHub 实现）。
func HandleAdminSkillsViaSkillHub(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	client := skillHubClientOrFail(w, r)
	if client == nil {
		return
	}

	page, pageSize := parsePagination(r)
	keyword := r.URL.Query().Get("keyword")

	resp, err := client.ListSkills(r.Context(), page, pageSize, keyword)
	if err != nil {
		skillHubAPIFail(w, r, "ListSkills", err)
		return
	}

	hatcherySkills := skillhubclient.ConvertSkillHubListToHatchery(resp)
	jsonOK(w, map[string]interface{}{
		"ok":        true,
		"skills":    hatcherySkills,
		"total":     resp.Total,
		"page":      page,
		"page_size": pageSize,
	})
}

// ── Phase 3 扩展示例（暂不实现，展示模式）──
//
// func HandleCreateSkillViaSkillHub(w, r) {
//     if !requireAdmin(w, r) { return }
//     jsonAPI(w)
//     client := skillHubClientOrFail(w, r)
//     if client == nil { return }
//     resp, err := client.CreateSkill(r.Context(), req)
//     if err != nil { skillHubAPIFail(w, r, "CreateSkill", err); return }
//     jsonOK(w, skillhubclient.ConvertSkillHubDetailToHatchery(resp))
// }
