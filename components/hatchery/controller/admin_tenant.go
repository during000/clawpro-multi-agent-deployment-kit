package controller

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"gorm.io/gorm"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// ==================== 请求/响应结构体 ====================

// TenantInitAPIRequest 对应 openclaw-api 的 InitTenantRequest。
type TenantInitAPIRequest struct {
	Identifier            string   `json:"identifier"`
	Domains               []string `json:"domains"`        // 租户绑定的域名列表（纯 host），写入 tenant_domains 表
	PrimaryDomain         string   `json:"primary_domain"` // 主域名（带协议的完整 URL），写入 site_configs.domain
	Uin                   string   `json:"uin"`
	InitUser              string   `json:"init_user"`
	InitPass              string   `json:"init_pass"`
	InternalSecret        string   `json:"internal_secret"`
	OneIDAccountID        string   `json:"oneid_account_id"`
	OneIDAppID            string   `json:"oneid_app_id"`
	OneIDClientID         string   `json:"oneid_client_id"`
	OneIDClientSecret     string   `json:"oneid_client_secret"`
	OneIDTokenEndpoint    string   `json:"oneid_token_endpoint"`
	OneIDDomain           string   `json:"oneid_domain"`
	SecretId              string   `json:"secret_id"`
	SecretKey             string   `json:"secret_key"`
	AgentCamRoleSecretId  string   `json:"agent_cam_role_secret_id"`
	AgentCamRoleSecretKey string   `json:"agent_cam_role_secret_key"`
	DefaultLang           string   `json:"default_lang"`      // 租户默认语言：zh 或 en
	SecurityPolicies      []string `json:"security_policies"` // 安全策略
}

var supportedSecurityPolicies = []string{"SSRF"}

// TenantDomainRequest 域名管理请求体。
type TenantDomainRequest struct {
	Identifier string `json:"identifier"`
	Domain     string `json:"domain"`
}

// ==================== Handler: POST /tenants/init ====================

// HandleInitTenant 创建新租户（全量初始化）。
// 幂等：identifier 已存在返回 409。
func HandleInitTenant(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, hcommon.I18nError(i18n.MsgMethodNotAllowed))
		return
	}
	if !isAdminTokenRequest(r) {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgAdminRequired))
		return
	}

	var req TenantInitAPIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgBadRequest))
		return
	}
	if req.Identifier == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "identifier"))
		return
	}
	if len(req.Domains) == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgTenantDomainRequired))
		return
	}
	// 默认为中文
	switch req.DefaultLang {
	case "zh", "en":
	case "":
		req.DefaultLang = "zh"
	default:
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgDefaultLangInvalid, req.DefaultLang))
		return
	}

	// 校验安全策略
	for _, policy := range req.SecurityPolicies {
		if !slices.Contains(supportedSecurityPolicies, policy) {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSecurityPolicyInvalid, policy))
			return
		}
	}

	ctx := hcommon.WithSkipIdentifier(r.Context())

	// 分布式锁防并发
	lock, err := model.AcquireLock(ctx, "tenant:init:"+req.Identifier, 30*time.Second)
	if err != nil {
		slog.Warn("[Tenant] 获取初始化锁失败", "identifier", req.Identifier, "error", err)
		writeError(w, r, http.StatusConflict, hcommon.I18nRichError(err, i18n.MsgOperationInProgress))
		return
	}
	defer lock.Release()

	// 检查 identifier 是否已存在
	var existCount int64
	model.DBGlobal(ctx).Model(&model.SiteConfig{}).Where("identifier = ?", req.Identifier).Count(&existCount)
	if existCount > 0 {
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgTenantIdentifierExists))
		return
	}

	if err := executeTenantInit(ctx, &req); err != nil {
		slog.Error("[Tenant] 初始化失败", "identifier", req.Identifier, "error", err)
		if errors.Is(err, hcommon.I18nError(i18n.MsgTenantDomainAlreadyMapped)) {
			writeError(w, r, http.StatusConflict, hcommon.EnsureRichErrorOrPanic(err))
		} else {
			writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		}
		return
	}

	snap := buildSnapFromInitReq(&req)

	// 事务提交成功后执行幂等 Seed（复用 task scheduler 的 RunAllSeeds）
	tenantCtx := hcommon.InjectTenant(ctx, snap)
	model.RunAllSeeds(tenantCtx, NewCVMClient)

	// 预热缓存
	for _, domain := range req.Domains {
		model.WarmTenantCache(domain, snap)
	}

	slog.Info("[Tenant] 初始化成功", "identifier", req.Identifier, "domains", req.Domains)
	jsonOK(w, map[string]interface{}{"ok": true, "identifier": req.Identifier})
}

// executeTenantInit 在事务内创建租户核心资源（域名映射、站点配置、管理员）。
// Seed 数据在事务外通过 model.RunAllSeeds 幂等执行。
func executeTenantInit(ctx context.Context, req *TenantInitAPIRequest) error {
	err := model.DBGlobal(ctx).Transaction(func(tx *gorm.DB) error {
		// 写入 tenant_domains
		for _, domain := range req.Domains {
			td := model.TenantDomain{Domain: domain, Identifier: req.Identifier}
			if err := tx.Create(&td).Error; err != nil {
				if model.IsDuplicateError(err) {
					return hcommon.I18nRichError(err, i18n.MsgTenantDomainAlreadyMapped, domain)
				}
				return hcommon.I18nRichError(err, i18n.MsgTenantCreateDomainFailed, domain)
			}
		}

		// 创建 SiteConfig（身份字段 + 业务默认值）
		config := model.SiteConfig{
			Identifier:            req.Identifier,
			Uin:                   req.Uin,
			Domain:                req.PrimaryDomain,
			InternalSecret:        req.InternalSecret,
			OneIDAccountID:        req.OneIDAccountID,
			OneIDAppID:            req.OneIDAppID,
			OneIDClientID:         req.OneIDClientID,
			OneIDClientSecret:     req.OneIDClientSecret,
			OneIDTokenEndpoint:    req.OneIDTokenEndpoint,
			OneIDDomain:           req.OneIDDomain,
			CVMSecretId:           req.SecretId,
			CVMSecretKey:          req.SecretKey,
			AgentCamRoleSecretId:  req.AgentCamRoleSecretId,
			AgentCamRoleSecretKey: req.AgentCamRoleSecretKey,
			DefaultLang:           req.DefaultLang,
			SecurityPolicies:      strings.Join(req.SecurityPolicies, ","),
		}
		model.ApplySiteConfigDefaults(&config)
		if err := tx.Create(&config).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgTenantCreateSiteConfigFailed)
		}

		// 创建管理员
		if req.InitUser != "" && req.InitPass != "" {
			tenantCtx := hcommon.InjectTenant(tx.Statement.Context, hcommon.TenantSnapshot{Identifier: req.Identifier})
			tenantTx := tx.WithContext(tenantCtx)
			if err := model.CreateInitAdmin(tenantTx, req.InitUser, req.InitPass); err != nil {
				return hcommon.I18nRichError(err, i18n.MsgTenantCreateAdminFailed)
			}
		}
		return nil
	})
	if err == nil {
		return nil
	}
	var richErr *hcommon.RichError
	if errors.As(err, &richErr) {
		return richErr
	}
	return hcommon.I18nRichError(err, i18n.MsgTenantInitFailed)
}

// buildSnapFromInitReq 从初始化请求构造 TenantSnapshot（用于缓存预热）。
func buildSnapFromInitReq(req *TenantInitAPIRequest) hcommon.TenantSnapshot {
	return hcommon.TenantSnapshot{
		Identifier:            req.Identifier,
		Uin:                   req.Uin,
		Domain:                req.PrimaryDomain,
		InternalSecret:        req.InternalSecret,
		OneIDAccountID:        req.OneIDAccountID,
		OneIDAppID:            req.OneIDAppID,
		OneIDClientID:         req.OneIDClientID,
		OneIDClientSecret:     req.OneIDClientSecret,
		OneIDTokenEndpoint:    req.OneIDTokenEndpoint,
		OneIDDomain:           req.OneIDDomain,
		CVMSecretId:           req.SecretId,
		CVMSecretKey:          req.SecretKey,
		AgentCamRoleSecretId:  req.AgentCamRoleSecretId,
		AgentCamRoleSecretKey: req.AgentCamRoleSecretKey,
		DefaultLang:           req.DefaultLang,
	}
}

// ==================== Handler: POST/DELETE /tenants/domains ====================

// HandleTenantDomains 管理租户域名映射。
//   - POST: 新增域名映射（幂等：409）
//   - DELETE: 移除域名映射（幂等：不存在也 200）
func HandleTenantDomains(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)

	if !isAdminTokenRequest(r) {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgAdminRequired))
		return
	}

	switch r.Method {
	case http.MethodPost:
		handleAddTenantDomain(w, r)
	case http.MethodDelete:
		handleRemoveTenantDomain(w, r)
	default:
		writeError(w, r, http.StatusMethodNotAllowed, hcommon.I18nError(i18n.MsgMethodNotAllowed))
	}
}

func handleAddTenantDomain(w http.ResponseWriter, r *http.Request) {
	var req TenantDomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgBadRequest))
		return
	}
	if req.Identifier == "" || req.Domain == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "identifier and domain"))
		return
	}

	ctx := hcommon.WithSkipIdentifier(r.Context())

	td := model.TenantDomain{Domain: req.Domain, Identifier: req.Identifier}
	if err := model.DBGlobal(ctx).Create(&td).Error; err != nil {
		if model.IsDuplicateError(err) {
			writeError(w, r, http.StatusConflict, hcommon.I18nRichError(err, i18n.MsgTenantDomainAlreadyMapped, req.Domain))
			return
		}
		slog.Error("[Tenant] 新增域名失败", "domain", req.Domain, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalServerError))
		return
	}

	// 预热缓存
	var config model.SiteConfig
	if model.DBGlobal(ctx).Where("identifier = ?", req.Identifier).First(&config).Error == nil {
		model.WarmTenantCache(req.Domain, model.SnapFromConfig(config))
	}

	slog.Info("[Tenant] 新增域名映射", "identifier", req.Identifier, "domain", req.Domain)
	jsonOK(w, map[string]interface{}{"ok": true})
}

func handleRemoveTenantDomain(w http.ResponseWriter, r *http.Request) {
	var req TenantDomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgBadRequest))
		return
	}
	if req.Identifier == "" || req.Domain == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "identifier and domain"))
		return
	}

	ctx := hcommon.WithSkipIdentifier(r.Context())

	// 禁止删除主域名
	var config model.SiteConfig
	if model.DBGlobal(ctx).Where("identifier = ?", req.Identifier).First(&config).Error == nil {
		if model.ExtractHostFromURL(config.Domain) == req.Domain {
			writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgTenantCannotRemovePrimaryDomain))
			return
		}
	}

	model.DBGlobal(ctx).Where("identifier = ? AND domain = ?", req.Identifier, req.Domain).Delete(&model.TenantDomain{})
	model.InvalidateTenantCache(req.Domain)

	slog.Info("[Tenant] 移除域名映射", "identifier", req.Identifier, "domain", req.Domain)
	jsonOK(w, map[string]interface{}{"ok": true})
}

// ==================== Handler: GET /tenants/{identifier}/domains ====================

// HandleListTenantDomains 查询指定租户的所有域名。
func HandleListTenantDomains(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)

	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, hcommon.I18nError(i18n.MsgMethodNotAllowed))
		return
	}
	if !isAdminTokenRequest(r) {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgAdminRequired))
		return
	}

	// 从路径提取 identifier：/tenants/{identifier}/domains
	path := strings.TrimPrefix(r.URL.Path, "/tenants/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 || parts[0] == "" || parts[1] != "domains" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}
	identifier := parts[0]

	ctx := hcommon.WithSkipIdentifier(r.Context())

	var domains []model.TenantDomain
	if err := model.DBGlobal(ctx).Where("identifier = ?", identifier).Find(&domains).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgTenantQueryDomainsFailed))
		return
	}

	var config model.SiteConfig
	mainHost := ""
	if model.DBGlobal(ctx).Where("identifier = ?", identifier).First(&config).Error == nil {
		mainHost = model.ExtractHostFromURL(config.Domain)
	}

	type domainItem struct {
		Domain    string `json:"domain"`
		IsMain    bool   `json:"is_main"`
		CreatedAt string `json:"created_at,omitempty"`
	}
	result := make([]domainItem, 0, len(domains))
	for _, d := range domains {
		result = append(result, domainItem{
			Domain:    d.Domain,
			IsMain:    d.Domain == mainHost,
			CreatedAt: d.CreatedAt.Format(time.RFC3339),
		})
	}

	jsonOK(w, map[string]interface{}{
		"ok":         true,
		"identifier": identifier,
		"domains":    result,
	})
}
