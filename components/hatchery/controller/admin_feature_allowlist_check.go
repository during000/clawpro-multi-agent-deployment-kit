package controller

import (
	"net/http"
	"strings"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// HandleAdminFeatureAllowlistCheck 通用白名单查询接口。
//
// 鉴权：requireAdmin
// Path：GET /admin/feature-allowlist/check?type=<>
// 返回：{ "in_allowlist": bool }
//
// 一期主要服务于 type='local-agent' 的管控页诊断；参数化设计让未来新增
// feature 时（如其他跨租户灰度功能）可零接口扩展直接复用。
//
// 空表行为与业务入口一致，由 model 中各 feature type 的集中配置决定。
//
// identifier 取自当前登录的 tenant admin（与 HandleAdminLocalAgentTypes
// 一致），不再由前端任意指定：超管（AdminToken 请求）绕过白名单直接返
// in_allowlist=true，否则按其所属租户的 identifier 过白名单。
//
// 详见 iwiki §5.C.5。
func HandleAdminFeatureAllowlistCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	featureType := strings.TrimSpace(r.URL.Query().Get("type"))
	if featureType == "" {
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgBadRequestParamRequired, "type"))
		return
	}

	// 超管（AdminToken 直通）绕过白名单，始终 in_allowlist=true。
	if isAdminTokenRequest(r) {
		jsonOK(w, map[string]any{"in_allowlist": true})
		return
	}

	// 非超管：按当前 tenant admin 的 identifier 过白名单，不再接受前端任意 identifier。
	user, err := RequestUser(r)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
		return
	}
	if user == nil {
		// 异常边界：拿不到当前 tenant admin 的 user 时，保守返 false，
		// 不假装 in_allowlist=true 误导前端。
		jsonOK(w, map[string]any{"in_allowlist": false})
		return
	}

	// 复用 IsFeatureAllowed 已经吸收了「空表=全开」语义，本接口对外承诺相同。
	allowed, err := model.IsFeatureAllowed(r.Context(), featureType, user.Identifier)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError,
			hcommon.I18nRichError(err, i18n.MsgInternalError))
		return
	}
	jsonOK(w, map[string]any{
		"in_allowlist": allowed,
	})
}
