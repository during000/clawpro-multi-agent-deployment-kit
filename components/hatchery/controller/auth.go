package controller

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dchest/captcha"
	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// RegionInfo 聚合单个地域的所有元信息
type RegionInfo struct {
	Zones       []string // 可用区列表
	Name        string   // 完整中文名，如 "华北地区（北京）"
	NameEn      string   // 完整英文名，如 "Northern China (Beijing)"
	ShortName   string   // 简称，如 "北京"
	ShortNameEn string   // 简称英文，如 "Beijing"
	ID          int      // 腾讯云控制台 rid 参数，fallback 0
	Timezone    string   // IANA 时区名，如 Asia/Shanghai
}

var (
	Store       *sessions.CookieStore
	LoadScript  func(name string) (string, error)
	AdminToken  string
	CVMRegion   string        // 从 --region 参数读取
	EmailAPIURL string        // 从 --email-api-url 参数读取，邮件 API 地址
	DisableUI   bool   = true // 是否禁用 HTML UI，true 则仅提供 API (默认不再提供 HTML UI)
	GatewayURL  string        // 从 GATEWAY_URL 环境变量读取，Gateway 模式下 SSO 登录 redirect 到此 URL
)

var (

	// Regions 统一的地域信息表，包含可用区、名称、简称和控制台 ID
	Regions = map[string]RegionInfo{
		"ap-beijing": {
			Zones:       []string{"ap-beijing-6", "ap-beijing-8"},
			Name:        "华北地区（北京）",
			NameEn:      "Northern China (Beijing)",
			ShortName:   "北京",
			ShortNameEn: "Beijing",
			ID:          8,
			Timezone:    "Asia/Shanghai",
		},
		"ap-shanghai": {
			Zones:       []string{"ap-shanghai-5", "ap-shanghai-8"},
			Name:        "华东地区（上海）",
			NameEn:      "Eastern China (Shanghai)",
			ShortName:   "上海",
			ShortNameEn: "Shanghai",
			ID:          4,
			Timezone:    "Asia/Shanghai",
		},
		"ap-guangzhou": {
			Zones:       []string{"ap-guangzhou-6", "ap-guangzhou-7"},
			Name:        "华南地区（广州）",
			NameEn:      "Southern China (Guangzhou)",
			ShortName:   "广州",
			ShortNameEn: "Guangzhou",
			ID:          1,
			Timezone:    "Asia/Shanghai",
		},
		"ap-nanjing": {
			Zones:       []string{"ap-nanjing-1", "ap-nanjing-3"},
			Name:        "华东地区（南京）",
			NameEn:      "Eastern China (Nanjing)",
			ShortName:   "南京",
			ShortNameEn: "Nanjing",
			ID:          33,
			Timezone:    "Asia/Shanghai",
		},
		"ap-chongqing": {
			ShortName:   "重庆",
			ShortNameEn: "Chongqing",
			ID:          19,
			Timezone:    "Asia/Shanghai",
		},
		"eu-frankfurt": {
			Zones:       []string{"eu-frankfurt-1", "eu-frankfurt-2"},
			Name:        "欧洲地区（法兰克福）",
			NameEn:      "Europe (Frankfurt)",
			ShortName:   "法兰克福",
			ShortNameEn: "Frankfurt",
			ID:          17,
			Timezone:    "Europe/Berlin",
		},
		"ap-bangkok": {
			Zones:       []string{"ap-bangkok-1", "ap-bangkok-2"},
			Name:        "亚太东南（曼谷）",
			NameEn:      "Southeast Asia (Bangkok)",
			ShortName:   "曼谷",
			ShortNameEn: "Bangkok",
			ID:          23,
			Timezone:    "Asia/Bangkok",
		},
		"ap-singapore": {
			Zones:       []string{"ap-singapore-1", "ap-singapore-2", "ap-singapore-3", "ap-singapore-4"},
			Name:        "亚太东南（新加坡）",
			NameEn:      "Southeast Asia (Singapore)",
			ShortName:   "新加坡",
			ShortNameEn: "Singapore",
			ID:          90,
			Timezone:    "Asia/Singapore",
		},
		"ap-jakarta": {
			Zones:       []string{"ap-jakarta-1", "ap-jakarta-2"},
			Name:        "亚太东南（雅加达）",
			NameEn:      "Southeast Asia (Jakarta)",
			ShortName:   "雅加达",
			ShortNameEn: "Jakarta",
			ID:          72,
			Timezone:    "Asia/Jakarta",
		},
		"ap-hongkong": {
			Zones:       []string{"ap-hongkong-2", "ap-hongkong-3"},
			Name:        "港澳台地区（中国香港）",
			NameEn:      "Hong Kong, Macao and Taiwan (Hong Kong)",
			ShortName:   "香港",
			ShortNameEn: "Hong Kong",
			ID:          5,
			Timezone:    "Asia/Hong_Kong",
		},
	}
)

// getUserFromToken 从 Bearer Token 中解析用户。
// 未携带 Token 时返回 (nil, nil)；携带了 Token 但无法匹配时返回 (nil, err)。
func getUserFromToken(r *http.Request) (*model.User, error) {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return nil, nil
	}
	token := strings.TrimPrefix(auth, "Bearer ")

	// 1. 后台管理 token
	if AdminToken != "" && subtle.ConstantTimeCompare([]byte(token), []byte(AdminToken)) == 1 {
		return &model.User{Username: "admin-token", Role: "admin"}, nil
	}

	// 2. 用户 API Token —— 仅在 OpenAPI 路由中生效
	if !isOpenAPIRequest(r) {
		return nil, nil // 非 OpenAPI 路由，忽略用户 Token，回退到 Session
	}

	user, err := model.GetUserByAPIToken(r.Context(), token)
	if user != nil {
		// 鉴权成功，异步更新最近调用时间
		// 多租户阶段一：goroutine 脱离 r.Context() 的 deadline，但保留 TenantSnapshot
		// 保证 UpdateAPITokenLastUsed 里的 DB 回调仍能拿到租户 identifier
		go func(ctx context.Context, uid uint) {
			if err := model.UpdateAPITokenLastUsed(ctx, uid); err != nil {
				slog.Warn("更新 API Token 最近调用时间失败", "user_id", uid, "err", err)
			}
		}(hcommon.DetachContext(r.Context()), user.ID)
		return user, nil
	}
	if err != nil {
		// 封禁或 Token 被禁用
		return nil, hcommon.I18nRichError(err, i18n.MsgInvalidAPIToken)
	}
	return nil, hcommon.I18nError(i18n.MsgInvalidAPIToken)
}

func getSession(r *http.Request) *sessions.Session {
	session, _ := Store.Get(r, "hatchery-session")
	return session
}

func getLoginFailures(r *http.Request) int {
	session := getSession(r)
	if failures, ok := session.Values["login_failures"].(int); ok {
		return failures
	}
	return 0
}

func HandleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	session := getSession(r)
	if username, ok := session.Values["username"].(string); ok && username != "" {
		// 检查 OneID session 黑名单（仅 Gateway 模式下有效）
		// 条件与登出逻辑保持一致：只有同时配置了 GatewayURL 和 TenantID 才会有 OneID 登录/登出流程
		if GatewayURL != "" && hcommon.TenantIDFromCtx(r.Context()) != "" {
			sid, _ := session.Values["oneid_sid"].(string)
			sub, _ := session.Values["oneid_sub"].(string)
			loginAtUnix, _ := session.Values["login_at"].(int64)
			loginAt := time.Unix(loginAtUnix, 0)
			if model.IsSessionRevoked(r.Context(), sid, sub, loginAt) {
				session.Values["username"] = ""
				session.Options.MaxAge = -1
				session.Save(r, w)

				jsonOK(w, map[string]interface{}{"authenticated": false})
				return

			}
		}

		role, _ := session.Values["role"].(string)
		userID, _ := session.Values["user_id"].(uint)

		// 从数据库读取最新 role，避免 session 缓存过期。
		// 同时验证用户未被删除（软删除后 First 返回 ErrRecordNotFound）。
		var dbUser model.User
		err := model.DB(r.Context()).Select("role").Where("username = ?", username).First(&dbUser).Error
		if err != nil {
			// 用户不存在或已被删除，清除 session
			session.Values["username"] = ""
			session.Options.MaxAge = -1
			session.Save(r, w)
			jsonOK(w, map[string]interface{}{"authenticated": false})
			return
		}
		if dbUser.Role != role {
			role = dbUser.Role
			session.Values["role"] = role
			session.Save(r, w)
		}

		jsonOK(w, map[string]interface{}{
			"ok":       true,
			"redirect": "/my-openclaw",
			"username": username,
			"role":     role,
			"user_id":  userID,
		})
		return

	}

	resp := map[string]interface{}{"authenticated": false}
	if getLoginFailures(r) > 0 {
		resp["captchaId"] = captcha.New()
	}
	jsonOK(w, resp)
}

func HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	renderErr := func(msg i18n.Key) {
		session := getSession(r)
		failures := getLoginFailures(r)
		failures++
		session.Values["login_failures"] = failures
		session.Save(r, w)

		writeError(w, r, http.StatusUnauthorized, hcommon.I18nError(msg).WithCustomData(
			map[string]any{
				"captchaId": captcha.New(),
			}),
		)
	}

	if getLoginFailures(r) > 0 {
		captchaID := r.FormValue("captchaId")
		captchaValue := r.FormValue("captchaValue")
		if !captcha.VerifyString(captchaID, captchaValue) {
			renderErr(i18n.MsgInvalidCaptcha)
			return
		}
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	var user model.User
	// 使用 Unscoped() 查询以包含软删除的记录
	result := model.DB(r.Context()).Unscoped().Where("username = ?", username).First(&user)
	if result.Error != nil {
		renderErr(i18n.MsgInvalidCredentials)
		return
	}

	// 检查用户是否被封禁
	if user.DeletedAt.Valid {
		renderErr(i18n.MsgAccountBanned)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		renderErr(i18n.MsgInvalidCredentials)
		return
	}

	if err := establishLocalSession(w, r, &user); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
		return
	}

	jsonOK(w, map[string]interface{}{
		"ok":       true,
		"redirect": "/",
		"role":     user.Role,
	})
}

func establishLocalSession(w http.ResponseWriter, r *http.Request, user *model.User) error {
	session := getSession(r)
	session.Values["username"] = user.Username
	session.Values["role"] = user.Role
	session.Values["login_at"] = time.Now().Unix()
	// 多租户阶段一：登录成功后写入 identifier 防跨租户 cookie 冒用
	setSessionIdentifier(session, r.Context())
	// 清除残留的 OneID session 字段，防止被 session 黑名单误判
	delete(session.Values, "oneid_sid")
	delete(session.Values, "oneid_sub")
	delete(session.Values, "oneid_amr")
	delete(session.Values, "user_id")
	delete(session.Values, "login_failures")
	return session.Save(r, w)
}

func HandleChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	session := getSession(r)
	username, _ := session.Values["username"].(string)
	if username == "" {
		writeError(w, r, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	// JSON 请求未传密码参数时，返回新的 captchaId 供 API 客户端使用
	if r.FormValue("old_password") == "" && r.FormValue("new_password") == "" {
		jsonOK(w, map[string]interface{}{"captchaId": captcha.New()})
		return
	}

	captchaID := r.FormValue("captchaId")
	captchaValue := r.FormValue("captchaValue")
	if !captcha.VerifyString(captchaID, captchaValue) {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidCaptcha))
		return
	}

	oldPassword := r.FormValue("old_password")
	newPassword := r.FormValue("new_password")
	if oldPassword == "" || newPassword == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgIncompleteForm))
		return
	}

	var user model.User
	// 使用 Unscoped() 查询以包含软删除的记录
	if model.DB(r.Context()).Unscoped().Where("username = ?", username).First(&user).Error != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgUserNotExist))
		return
	}

	// 检查用户是否被封禁
	if user.DeletedAt.Valid {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgAccountBanned))
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgOldPasswordWrong))
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgInternalError))
		return
	}

	model.DB(r.Context()).Model(&user).Update("password", string(hash))

	session.Values["username"] = ""
	session.Options.MaxAge = -1
	session.Save(r, w)

	jsonOK(w, map[string]interface{}{"ok": true})
}

func HandleLogout(w http.ResponseWriter, r *http.Request) {
	session := getSession(r)

	// 清除本地 session
	session.Values["username"] = ""
	session.Options.MaxAge = -1
	session.Save(r, w)

	// Gateway 模式：302 跳转到 Gateway /auth/logout，由 Gateway 统一处理 OneID 登出
	if GatewayURL != "" && hcommon.TenantIDFromCtx(r.Context()) != "" {
		logoutURL := GatewayURL + "/auth/logout?tenant=" + url.QueryEscape(hcommon.TenantIDFromCtx(r.Context()))

		jsonRedirect(w, logoutURL)
		return
	}

	// 未配置 Gateway，直接跳回本站首页
	jsonRedirect(w, "/")
}

// HandleGetAPIToken 查看用户当前的 API Token 状态。
//
// GET /api-token
//
// 认证: 仅 Session Cookie（需网页登录）
//
// 请求参数（query）：
//
//	reveal=true 时额外返回明文 token（仅本人查看，且会写入 audit log）。
//	为避免意外泄露，mask 字段仍始终返掩码，明文走独立的 token 字段。
//
// 响应:
//
//	无 Token：   {"exists": false}
//	有 Token：   {"exists": true, "mask": "hk-abcd****wxyz", "disabled": false, "created_at": "..."}
//	reveal=true: 上述基础上额外携带 "token": "<明文>"
func HandleGetAPIToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	jsonAPI(w)

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	reveal := strings.EqualFold(r.URL.Query().Get("reveal"), "true")

	// reveal=true 路径复用 local-agent 白名单：明文 token 是为 reporter 安装场景服务的，
	// 未开放本地 Agent 功能的租户也没有看明文的场景需求。不在白名单则退化为
	// 默认掩码路径（不报 403，以免前端需要区分「未启用」与「未登录」两种类型的错误）。
	if reveal {
		allowed, err := model.IsFeatureAllowed(r.Context(), model.FeatureAllowlistTypeLocalAgent, user.Identifier)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
			return
		}
		if !allowed {
			reveal = false
		}
	}

	if user.HasAPIToken() {
		resp := map[string]interface{}{
			"exists":     true,
			"mask":       model.MaskAPIToken(*user.APIToken),
			"disabled":   user.APITokenDisabled,
			"created_at": user.APITokenCreatedAt,
		}
		if reveal {
			resp["token"] = *user.APIToken
			// 明文获取需走 audit：将来排查“token 为什么被泄”能溯源到是哪台设备拉走的。
			// 使用 detach context 避免调用方请求中断后写 audit 被取消。
			go model.LogAudit(hcommon.DetachContext(r.Context()), time.Now(),
				user.ID, user.Username, "token_reveal", "api_token", "", "success")
		}
		jsonOK(w, resp)
		return
	}

	jsonOK(w, map[string]interface{}{
		"exists": false,
	})
}

// HandleCreateAPIToken 创建用户的 API Token。若已有 Token 则返回错误。
//
// POST /api-token/create
//
// 认证: 仅 Session Cookie（需网页登录）
//
// 响应:
//
//	{"token": "hk-...", "mask": "hk-abcd****wxyz"}
func HandleCreateAPIToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	jsonAPI(w)

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	// 被禁用时不允许创建（防止通过销毁→重建绕过禁用）
	if user.APITokenDisabled {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgAPITokenDisabledByAdmin))
		return
	}

	if user.HasAPIToken() {
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgAPITokenAlreadyExists))
		return
	}

	token, err := model.GenerateAPIToken(r.Context(), user.ID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgAPITokenGenerateFailed))
		return
	}

	jsonOK(w, map[string]interface{}{
		"token": token,
		"mask":  model.MaskAPIToken(token),
	})
}

// HandleResetAPIToken 重置用户的 API Token，旧 Token 立即失效。
//
// POST /api-token/reset
//
// 认证: 仅 Session Cookie（需网页登录）
//
// 响应:
//
//	{"token": "hk-...", "mask": "hk-abcd****wxyz"}
func HandleResetAPIToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	jsonAPI(w)

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	if user.APITokenDisabled {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgAPITokenDisabledByAdmin))
		return
	}

	token, err := model.GenerateAPIToken(r.Context(), user.ID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgAPITokenResetFailed))
		return
	}

	jsonOK(w, map[string]interface{}{
		"token": token,
		"mask":  model.MaskAPIToken(token),
	})
}

// HandleRevokeAPIToken 销毁用户的 API Token，Token 立即失效。
//
// POST /api-token/revoke
//
// 认证: 仅 Session Cookie（需网页登录）
//
// 响应:
//
//	{"ok": true}
func HandleRevokeAPIToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	jsonAPI(w)

	user := requireLogin(w, r)
	if user == nil {
		return
	}

	if user.APITokenDisabled {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgAPITokenDisabledByAdmin))
		return
	}

	if err := model.RevokeAPIToken(r.Context(), user.ID); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgAPITokenRevokeFailed))
		return
	}

	jsonOK(w, map[string]interface{}{
		"ok": true,
	})
}
