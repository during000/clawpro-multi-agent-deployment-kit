package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	sdkerrors "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
)

func HandleAdminConfig(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	config := model.GetSiteConfig(r.Context())
	ri := Regions[CVMRegion]

	defaultLang := hcommon.DefaultLangFromCtx(r.Context())

	var cvmRegionName string
	if defaultLang == "zh" {
		cvmRegionName = ri.Name
	} else {
		cvmRegionName = ri.NameEn
	}

	defaultRules := config.ResolvedDefaultTokenQuotaRules()
	globalRules := config.ResolvedGlobalTokenQuotaRules()
	globalQuotaDay, globalQuotaPeriod := model.EffectiveGlobalTokenQuotaLegacyFields(
		config.GlobalTokenQuotaDay,
		config.GlobalTokenQuotaPeriod,
		config.GlobalTokenQuotaRules,
	)

	defaultTags, err := model.GetGlobalTagItemsForConfig(r.Context(), config.DefaultTags)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgQueryDefaultTagsFailed).WithDetail(err.Error()))
		return
	}

	effectiveResourceConfig, err := effectiveDefaultResourceConfig(r.Context(), config.CVMTemplate)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgParseCVMTemplateFailed))
		return
	}
	instanceChargeType := effectiveResourceConfig.InstanceChargeType

	result := map[string]interface{}{
		"name":                 config.Name,
		"has_logo":             len(config.Logo) > 0,
		"cvm_region":           cvmRegionName,
		"cvm_region_id":        CVMRegion,
		"available_zones":      ri.Zones,
		"cvm_secret_id":        config.CVMSecretId,
		"cvm_template":         config.CVMTemplate,
		"instance_charge_type": instanceChargeType,
		"security_group_id":    config.SecurityGroupId,
		"skillhub":             config.SkillHub,
		"cvm_uin":              hcommon.CVMUinFromCtx(r.Context()),
		"domain":               hcommon.DomainFromCtx(r.Context()),
		// 历史 API 字段名保留 day；实际周期见 global_token_quota_period。
		"global_token_quota_day":      globalQuotaDay,
		"global_token_quota_period":   globalQuotaPeriod,
		"global_token_quota_rules":    globalRules,
		"public_image_id":             config.PublicImageId,
		"vpc_id":                      config.VpcId,
		"subnet_ids":                  config.SubnetIds,
		"terminal_enabled":            config.TerminalEnabled,
		"chat_view_enabled":           config.ChatViewEnabled,
		"gateway_ui_enable":           config.GatewayUIEnable,
		"gateway_ui_port":             config.GatewayUIPort,
		"gateway_ui_addr_type":        config.GatewayUIAddrType,
		"browser_vnc_enable":          config.BrowserVNCEnable,
		"user_data_enabled":           config.UserDataEnabled,
		"user_config_model_enabled":   config.UserConfigModelEnabled,
		"user_config_channel_enabled": config.UserConfigChannelEnabled,
		"model_quota_enabled":         config.ModelQuotaEnabled,
		"agent_cam_role_secret_id":    config.AgentCamRoleSecretId,
		"has_oneid":                   hcommon.TenantIDFromCtx(r.Context()) != "",
		"is_unified_account":          hcommon.IsUnifiedAccountMode(r.Context()),
		"sso_im_types":                config.GetSSOIMTypes(),
		"sso_im_type_options":         model.SSOIMTypeOptions,
		"default_instance_quota":      config.DefaultInstanceQuota,
		"default_token_quota_day":     model.EffectiveTokenQuotaDay(config.DefaultTokenQuotaDay, config.DefaultTokenQuotaRules),
		"default_token_quota_rules":   defaultRules,
		"default_tags":                model.MarshalTagItems(defaultTags),
		"api_gateway_config":          config.APIGatewayConfig,
		"doctor_enabled":              config.DoctorEnabled,
		"local_agent_enabled":         config.LocalAgentEnabled,
		"default_lang":                config.DefaultLang,
	}

	// 资源子模块返回默认 ResourcePolicy 覆盖 CVMTemplate 后的实际生效值。
	// 未传 template_path 时仍默认返回 internet_accessible，保持旧前端兼容。
	paths := r.URL.Query()["template_path"]
	if len(paths) == 0 {
		// 向后兼容：未传 template_path 时默认返回 internet_accessible
		paths = []string{"internet_accessible"}
	}
	for _, p := range paths {
		section, err := resourceConfigSection(effectiveResourceConfig, p)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
			return
		}
		if section != nil {
			result[p] = section
		}
	}

	jsonOK(w, map[string]interface{}{"config": result})
}

func HandleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	const maxLogoSize = 512 << 10 // 512KB
	r.ParseMultipartForm(maxLogoSize + 4096)

	config := model.GetSiteConfig(r.Context())
	// 只记录本次请求实际修改的字段，避免 /admin/config 与 /admin/config/cvm 并发更新同一行时互相覆盖。
	updateFields := make([]string, 0, 32)

	name := r.FormValue("name")
	if name != "" {
		config.Name = name
		updateFields = append(updateFields, "Name")
	}

	if v := r.FormValue("global_token_quota_day"); v != "" {
		q, err := strconv.Atoi(v)
		if err != nil || q < -1 {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgGlobalQuotaInvalid))
			return
		}
		// GlobalTokenQuotaDay 是历史字段名；实际可按日或按月统计，周期由 GlobalTokenQuotaPeriod 决定。
		period := config.NormalizedGlobalTokenQuotaPeriod()
		if pv := r.FormValue("global_token_quota_period"); model.IsValidGlobalTokenQuotaPeriod(pv) {
			period = model.NormalizeGlobalTokenQuotaPeriod(pv)
		}
		config.GlobalTokenQuotaDay = -1
		config.GlobalTokenQuotaRules = model.UpsertGlobalPeriodRule(config.GlobalTokenQuotaRules, period, q)
		updateFields = append(updateFields, "GlobalTokenQuotaDay", "GlobalTokenQuotaRules")
	}
	if v := r.FormValue("global_token_quota_period"); v != "" {
		if !model.IsValidGlobalTokenQuotaPeriod(v) {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgGlobalQuotaPeriodInvalid))
			return
		}
		config.GlobalTokenQuotaPeriod = model.NormalizeGlobalTokenQuotaPeriod(v)
		updateFields = append(updateFields, "GlobalTokenQuotaPeriod")
	}
	if v := r.FormValue("global_token_quota_rules"); v != "" {
		normalized, err := model.NormalizeTokenQuotaRules(v)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
			return
		}
		config.GlobalTokenQuotaDay = -1
		config.GlobalTokenQuotaRules = normalized
		updateFields = append(updateFields, "GlobalTokenQuotaDay", "GlobalTokenQuotaRules")
	}

	if v := r.FormValue("default_instance_quota"); v != "" {
		q, err := strconv.Atoi(v)
		if err != nil || q < 0 || q > 999 {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgDefaultInstanceQuotaInvalid))
			return
		}
		config.DefaultInstanceQuota = q
		updateFields = append(updateFields, "DefaultInstanceQuota")
	}

	if v := r.FormValue("default_token_quota_day"); v != "" {
		q, err := strconv.Atoi(v)
		if err != nil || q < -1 {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgDefaultTokenQuotaInvalid))
			return
		}
		config.DefaultTokenQuotaDay = -1
		config.DefaultTokenQuotaRules = model.UpsertDayRule(config.DefaultTokenQuotaRules, q)
		updateFields = append(updateFields, "DefaultTokenQuotaDay", "DefaultTokenQuotaRules")
	}

	if v := r.FormValue("default_token_quota_rules"); v != "" {
		normalized, err := model.NormalizeTokenQuotaRules(v)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
			return
		}
		config.DefaultTokenQuotaDay = -1
		config.DefaultTokenQuotaRules = normalized
		updateFields = append(updateFields, "DefaultTokenQuotaDay", "DefaultTokenQuotaRules")
	}

	if v := r.FormValue("terminal_enabled"); v != "" {
		config.TerminalEnabled = v == "true"
		updateFields = append(updateFields, "TerminalEnabled")
	}

	if v := r.FormValue("chat_view_enabled"); v != "" {
		config.ChatViewEnabled = v == "true"
		updateFields = append(updateFields, "ChatViewEnabled")
	}

	if v := r.FormValue("user_config_model_enabled"); v != "" {
		config.UserConfigModelEnabled = v == "true"
		updateFields = append(updateFields, "UserConfigModelEnabled")
	}
	if v := r.FormValue("user_config_channel_enabled"); v != "" {
		config.UserConfigChannelEnabled = v == "true"
		updateFields = append(updateFields, "UserConfigChannelEnabled")
	}
	if v := r.FormValue("model_quota_enabled"); v != "" {
		config.ModelQuotaEnabled = v == "true"
		updateFields = append(updateFields, "ModelQuotaEnabled")
	}
	if v := r.FormValue("user_data_enabled"); v != "" {
		config.UserDataEnabled = v == "true"
		updateFields = append(updateFields, "UserDataEnabled")
	}

	resourceConfigChanged := false
	if r.Form.Has("instance_charge_type") {
		updatedTemplate, err := applyInstanceChargeTypeToCVMTemplate(config.CVMTemplate, r.FormValue("instance_charge_type"))
		if err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
			return
		}
		config.CVMTemplate = updatedTemplate
		resourceConfigChanged = true
	}

	// 默认标签配置（JSON 数组）
	if r.Form.Has("default_tags") {
		raw := r.FormValue("default_tags")
		var tags []model.TagItem
		if raw == "" || raw == "[]" {
			tags = []model.TagItem{}
		} else {
			if err := json.Unmarshal([]byte(raw), &tags); err != nil {
				writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgDefaultTagsFormatInvalid))
				return
			}
		}
		if err := model.ReplaceGlobalTags(r.Context(), tags); err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgSaveDefaultTagsFailed).WithDetail(err.Error()))
			return
		}
		config.DefaultTags = "[]"
		updateFields = append(updateFields, "DefaultTags")
	}

	// 龙虾医生配置
	if v := r.FormValue("doctor_enabled"); v != "" {
		config.DoctorEnabled = v == "true"
		updateFields = append(updateFields, "DoctorEnabled")
	}

	// 本地 Agent 全局预设：与 doctor_enabled 同语义（v=="" 不变，v=="true" 开启，其它值关闭）
	if v := r.FormValue("local_agent_enabled"); v != "" {
		config.LocalAgentEnabled = v == "true"
		updateFields = append(updateFields, "LocalAgentEnabled")
	}

	// API 网关接入配置（软功能：WebUI 域名化访问）
	// JSON 结构：{"enable": bool, "gateway_instance_id": "ins-xxx", "base_domain": "xxx.com"}
	if r.Form.Has("api_gateway_config") {
		raw := strings.TrimSpace(r.FormValue("api_gateway_config"))
		if raw == "" {
			raw = "{}"
		}
		var probe model.APIGatewayConfig
		if err := json.Unmarshal([]byte(raw), &probe); err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgAPIGatewayConfigFormatInvalid))
			return
		}
		if probe.Enable && (probe.GatewayInstanceID == "" || probe.BaseDomain == "") {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgAPIGatewayFieldsRequired))
			return
		}
		config.APIGatewayConfig = raw
		updateFields = append(updateFields, "APIGatewayConfig")
	}

	if r.Form.Has("sso_im_types") {
		// 接收 JSON 数组字符串，如 ["wecom","feishu"]
		raw := r.FormValue("sso_im_types")
		var types []string
		if raw == "" || raw == "[]" {
			types = []string{}
		} else if err := json.Unmarshal([]byte(raw), &types); err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSSOIMTypesFormatInvalid))
			return
		}
		// 校验每个值：白名单来自 model.SSOIMTypeOptions，避免硬编码遗漏
		validIMTypes := make(map[string]bool, len(model.SSOIMTypeOptions))
		for _, opt := range model.SSOIMTypeOptions {
			validIMTypes[opt["value"]] = true
		}
		for _, t := range types {
			if !validIMTypes[t] {
				writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSSOIMTypeUnsupported, t))
				return
			}
		}
		config.SetSSOIMTypes(types)
		updateFields = append(updateFields, "SSOIMType")
	}

	// 开启 openClaw gateway ui 安全组规则判断
	wasEnabled := config.GatewayUIEnable
	wasAddrType := config.GatewayUIAddrType
	if v := r.FormValue("gateway_ui_enable"); v != "" {
		config.GatewayUIEnable = v == "true"
		updateFields = append(updateFields, "GatewayUIEnable")
	}
	if v := r.FormValue("gateway_ui_addr_type"); v != "" {
		if v != "private" && v != "public" {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgGatewayUIAddrTypeInvalid))
			return
		}
		config.GatewayUIAddrType = v
		updateFields = append(updateFields, "GatewayUIAddrType")
	}
	if config.GatewayUIEnable && !config.GatewayUISGMigrateDone {
		if err := handleFirstTimeGatewayUIEnable(r.Context(), &config); err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
			return
		}
	}
	// 触发 refresh 的两种场景（迁移已完成、且当前 Gateway UI 处于开启态）：
	//   1) 关→开（wasEnabled=false → 现在 true）：合入 allow_gateway_ui 必需规则
	//   2) addr_type 变更（public ↔ private）：private 模式下应剔除 allow_gateway_ui 规则组，
	//      避免在 SG 上自动注入 0.0.0.0/0 公网放通；public 模式下重新合入。
	// 仅 enable=true 才触发 —— enable=false 时 condition 评估已经天然不会注入规则。
	addrTypeChanged := wasAddrType != config.GatewayUIAddrType
	if config.GatewayUIEnable && config.GatewayUISGMigrateDone && (!wasEnabled || addrTypeChanged) {
		// refresh 会重新从 DB 读取 SiteConfig；这些字段必须先落库：
		//   - GatewayUIEnable / GatewayUIAddrType / GatewayUIPort：决定 allow_gateway_ui 是否合入/剔除，并替换端口占位符。
		//   - GatewayUISGMigrateDone：refresh 不读取；这里只同步迁移状态，避免后续请求误判为首次启用。
		if err := model.SaveSelectedFields(r.Context(), &config,
			"GatewayUIEnable",
			"GatewayUIAddrType",
			"GatewayUIPort",
			"GatewayUISGMigrateDone",
		); err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSaveGatewayUIFailed))
			return
		}
		// 新模型：通过 RefreshAllRuleSetsForRequiredRules 让 Gateway UI 端口必需规则
		// （{{GATEWAY_UI_PORT}} 由 resolveConditionalRules 替换）合入所有 RuleSet 并扇出到
		// 各自 ACTIVE 池。迁移后 config.SecurityGroupId 已是 FROZEN 的老 base，不能作为下发对象。
		// addr_type 切到 private 时，整包下发会自动从云端 SG 清除老的 0.0.0.0/0:port 规则。
		//
		// 使用 hcommon.DetachContext(r.Context()) 让 refresh 与 HTTP 请求生命周期脱钩——
		// 响应已返回后 r.Context() 会被取消，否则 fan-out 会中途中断造成云端"部分成功"的回滚黑洞；
		// 同时保留 TenantSnapshot / trace_id / request_id 等链路字段，确保多租户路由正确 + 日志可追踪。
		bgCtx := hcommon.DetachContext(r.Context())
		if err := RefreshAllRuleSetsForRequiredRules(bgCtx); err != nil {
			// 路线 3：refresh 失败不回滚开关、不改响应码（保持接口契约不变），
			// 仅升级日志严重度到 Error 并写 audit log，让运维能在后台感知到
			// "DB 开关已开但云端规则未下发" 的不一致状态，主动重试或人工介入。
			slog.Error("[AdminConfig] CRITICAL: gateway_ui 配置已落库但安全组规则下发失败，存在不一致风险",
				"error", err, "user_id", getUserIDFromRequest(r),
				"was_enabled", wasEnabled, "now_enabled", config.GatewayUIEnable,
				"was_addr_type", wasAddrType, "now_addr_type", config.GatewayUIAddrType)
			go model.LogAudit(bgCtx, time.Now(), getUserIDFromRequest(r), "admin",
				"toggle_gateway_ui_refresh_failed", "site_config", "gateway_ui_enable",
				fmt.Sprintf("failed: %s", err.Error()))
		}
	}

	// 云端浏览器开关
	var browserVNCPortWarning string
	if v := r.FormValue("browser_vnc_enable"); v != "" {
		config.BrowserVNCEnable = v == "true"
		updateFields = append(updateFields, "BrowserVNCEnable")
		if config.BrowserVNCEnable {
			// 前置校验：SG 池就绪（至少有 RuleSet + ACTIVE SG）
			ready, _, _, err := HasSGPoolReady(r.Context())
			if err != nil {
				writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQuerySGReadyFailed))
				return
			}
			if !ready {
				writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgPleaseInitSGFirst))
				return
			}
			// refresh 会重新从 DB 读取 SiteConfig；Browser VNC 规则只依赖 BrowserVNCEnable。
			if err := model.SaveSelectedFields(r.Context(), &config, "BrowserVNCEnable"); err != nil {
				writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSaveBrowserVNCFailed))
				return
			}
			// 确保 allow_vnc_whitelist 必需规则（由 browser_vnc_enable 开关控制）合入所有 RuleSet
			// 并扇出到各 ACTIVE SG。整包下发时云端 SG 会自动被规范化（老的 0.0.0.0/0 被清除）。
			//
			// 使用 hcommon.DetachContext(r.Context()) 让 refresh 与 HTTP 请求生命周期脱钩，
			// 避免响应已返回后 ctx 取消导致 fan-out 中途中断；
			// 同时保留 TenantSnapshot / trace_id 等链路字段，多租户路由正确 + 日志可追踪。
			bgCtx := hcommon.DetachContext(r.Context())
			if err := RefreshAllRuleSetsForRequiredRules(bgCtx); err != nil {
				// 路线 3：refresh 失败不回滚开关、不改响应码（保持接口契约不变），
				// 仅升级日志严重度到 Error 并写 audit log，让运维能在后台感知到
				// "DB 开关已开但云端规则未下发" 的不一致状态，主动重试或人工介入。
				slog.Error("[BrowserVNC] CRITICAL: browser_vnc_enable=true 已落库但安全组规则下发失败，存在不一致风险",
					"error", err, "user_id", getUserIDFromRequest(r))
				go model.LogAudit(bgCtx, time.Now(), getUserIDFromRequest(r), "admin",
					"toggle_browser_vnc_refresh_failed", "site_config", "browser_vnc_enable",
					fmt.Sprintf("failed: %s", err.Error()))
				browserVNCPortWarning = i18n.T(r.Context(), i18n.MsgBrowserVNCPortOpenFailed, err.Error())
			}
		}
	}

	file, header, err := r.FormFile("logo")
	if err == nil {
		defer file.Close()

		mime := header.Header.Get("Content-Type")
		if mime != "image/png" && mime != "image/jpeg" && mime != "image/svg+xml" {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgLogoTypeUnsupported))
			return
		}
		if header.Size > maxLogoSize {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgLogoTooLarge))
			return
		}

		data, err := io.ReadAll(file)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgReadFileFailed))
			return
		}
		config.Logo = data
		config.LogoMIME = mime
		updateFields = append(updateFields, "Logo", "LogoMIME")
	}

	if resourceConfigChanged {
		if err := saveLegacyResourceConfig(r.Context(), &config, map[string]any{"instance_charge_type": true}, false); err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
			return
		}
	}
	if err := model.SaveSelectedFields(r.Context(), &config, updateFields...); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgDatabaseOperationFailed))
		return
	}

	resp := map[string]interface{}{"ok": true}
	if config.GatewayUIEnable {
		resp["gateway_ui_port"] = config.GatewayUIPort
	}
	if browserVNCPortWarning != "" {
		resp["warning"] = browserVNCPortWarning
	}
	jsonOK(w, resp)
}

func HandleSite(w http.ResponseWriter, r *http.Request) {
	config := model.GetSiteConfig(r.Context())

	// OneID 模式（旧模式或统一账号模式）下，如果 oneid_domain 为空则主动拉取一次写入 DB
	oneIDDomain := config.OneIDDomain
	if oneIDDomain == "" && hcommon.TenantIDFromCtx(r.Context()) != "" {
		oneIDDomain = fetchOneIDDomain(r.Context())
	}

	data := map[string]interface{}{
		"name":                config.Name,
		"has_logo":            len(config.Logo) > 0,
		"has_oneid":           hcommon.TenantIDFromCtx(r.Context()) != "",
		"is_unified_account":  hcommon.IsUnifiedAccountMode(r.Context()),
		"is_universe":         hcommon.IsUniverseMode(),
		"oneid_domain":        oneIDDomain,
		"chat_view_enabled":   config.ChatViewEnabled,
		"sso_im_types":        config.GetSSOIMTypes(),
		"sso_im_type_options": model.SSOIMTypeOptions,
		"is_overseas":         IsOverseasFromCtx(r.Context()),
	}
	// 暴露 OneID 企业 account_id 给前端登录组件（如嵌入式 select_account）
	if hcommon.TenantIDFromCtx(r.Context()) != "" {
		data["oneid_account_id"] = hcommon.TenantIDFromCtx(r.Context())
	}
	if user, err := getLoginUser(r); user != nil && err == nil {
		data["skillhub"] = config.SkillHub
		// 功能开关返回全局默认值，前端根据 agent 绑定的 group_id 查配置应用
		data["chat_view_enabled"] = config.ChatViewEnabled
		data["terminal_enabled"] = config.TerminalEnabled
		data["gateway_ui_enable"] = config.GatewayUIEnable
		data["gateway_ui_addr_type"] = config.GatewayUIAddrType
		data["browser_vnc_enable"] = config.BrowserVNCEnable
		data["user_config_model_enabled"] = config.UserConfigModelEnabled
		data["user_config_channel_enabled"] = config.UserConfigChannelEnabled
		data["model_quota_enabled"] = config.ModelQuotaEnabled
		data["cvm_region_id"] = CVMRegion
	}
	jsonOK(w, data)
}

// defaultLogoSVG 是未上传 Logo 时返回的默认图标（机器人头像）。
const defaultLogoSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64" width="64" height="64">
  <rect width="64" height="64" rx="12" fill="#0052d9"/>
  <rect x="18" y="20" width="28" height="22" rx="6" fill="white"/>
  <circle cx="24" cy="30" r="3" fill="#0052d9"/>
  <circle cx="40" cy="30" r="3" fill="#0052d9"/>
  <rect x="24" y="36" width="16" height="2.5" rx="1.25" fill="#0052d9"/>
  <rect x="30" y="12" width="4" height="8" rx="2" fill="white"/>
  <circle cx="32" cy="11" r="3" fill="white"/>
  <rect x="10" y="26" width="5" height="10" rx="2.5" fill="white"/>
  <rect x="49" y="26" width="5" height="10" rx="2.5" fill="white"/>
</svg>`

func HandleLogo(w http.ResponseWriter, r *http.Request) {
	config := model.GetSiteConfig(r.Context())
	if len(config.Logo) == 0 {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Cache-Control", "public, max-age=60")
		w.Write([]byte(defaultLogoSVG))
		return
	}
	w.Header().Set("Content-Type", config.LogoMIME)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write(config.Logo)
}

func HandleUpdateCVMConfig(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	r.ParseForm()
	config := model.GetSiteConfig(r.Context())
	// CVM 页与基础配置共用 SiteConfig 行；只记录本次请求实际修改的字段，避免并发覆盖。
	updateFields := make([]string, 0, 32)

	// API Token 调用时，不允许修改敏感字段（密钥、模板、镜像）
	// 但 AdminToken（启动参数设置的超级管理令牌）不受此限制
	sensitiveBlocked := hasBearerToken(r) && !isAdminTokenRequest(r)

	if r.Form.Has("cvm_secret_id") && !sensitiveBlocked {
		config.CVMSecretId = r.FormValue("cvm_secret_id")
		updateFields = append(updateFields, "CVMSecretId")
	}
	if sk := r.FormValue("cvm_secret_key"); sk != "" && !sensitiveBlocked {
		config.CVMSecretKey = sk
		updateFields = append(updateFields, "CVMSecretKey")
	}
	cvmTemplateChanged := false
	if r.Form.Has("cvm_template") && !sensitiveBlocked {
		tpl := r.FormValue("cvm_template")
		if tpl != "" {
			if !json.Valid([]byte(tpl)) {
				writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgCVMTemplateMustBeJSON))
				return
			}
			// 新增：InternetAccessible 业务规则校验
			overview, err := model.ParseCVMTemplateOverview(tpl)
			if err != nil {
				writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
				return
			}
			if overview != nil && overview.InternetAccessible != nil {
				if err := model.ValidateInternetAccessible(overview.InternetAccessible, overview.InstanceChargeType); err != nil {
					writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
					return
				}
			}
		}
		config.CVMTemplate = tpl
		cvmTemplateChanged = true
	}
	if r.Form.Has("skillhub") {
		config.SkillHub = r.FormValue("skillhub")
		updateFields = append(updateFields, "SkillHub")
	}
	if r.Form.Has("public_image_id") && !sensitiveBlocked {
		config.PublicImageId = r.FormValue("public_image_id")
		updateFields = append(updateFields, "PublicImageId")
	}
	if r.Form.Has("vpc_id") {
		config.VpcId = r.FormValue("vpc_id")
		updateFields = append(updateFields, "VpcId")
		if config.VpcId == "" {
			config.SubnetIds = ""
			updateFields = append(updateFields, "SubnetIds")
		}
	}
	if r.Form.Has("subnet_ids") && config.VpcId != "" {
		config.SubnetIds = r.FormValue("subnet_ids")
		updateFields = append(updateFields, "SubnetIds")
	}
	if r.Form.Has("agent_cam_role_secret_id") {
		config.AgentCamRoleSecretId = r.FormValue("agent_cam_role_secret_id")
		updateFields = append(updateFields, "AgentCamRoleSecretId")
	}
	if sk := r.FormValue("agent_cam_role_secret_key"); sk != "" {
		config.AgentCamRoleSecretKey = sk
		updateFields = append(updateFields, "AgentCamRoleSecretKey")
	}

	// Validate VPC and subnet configuration
	if config.VpcId == "" && config.SubnetIds != "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgVPCWithoutGlobalID))
		return
	}
	if config.VpcId != "" {
		subnetMap := config.GetSubnetMap()
		if len(subnetMap) == 0 {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgSubnetCannotBeEmpty))
			return
		}

		// Verify zones are in the allowed RegionZones list
		allowedZones := make(map[string]bool)
		for _, z := range Regions[CVMRegion].Zones {
			allowedZones[z] = true
		}
		for zone := range subnetMap {
			if !allowedZones[zone] {
				writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgZoneNotInRegion, zone))
				return
			}
		}

		vpcClient, err := newVpcClient(r.Context())
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgCreateVPCClientFailed))
			return
		}

		if err := validateSubnetMapOnCloud(vpcClient, config.VpcId, subnetMap); err != nil {
			// 根据错误类型选择合适的 HTTP 状态码：腾讯云 API 故障走 500，其他走 400
			var cloudErr *subnetValidateCloudError
			if errors.As(err, &cloudErr) {
				writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
			} else {
				writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
			}
			return
		}
	}

	if cvmTemplateChanged {
		if err := saveLegacyResourceConfig(r.Context(), &config, nil, true); err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
			return
		}
	}
	// vpc_id 清空时需要同时落库 SubnetIds；sensitiveBlocked 跳过的敏感字段不进入更新列表。
	if err := model.SaveSelectedFields(r.Context(), &config, updateFields...); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgDatabaseOperationFailed))
		return
	}
	jsonOK(w, map[string]interface{}{"ok": true})
}

func HandleListCloudVpcs(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	// 分页参数：offset（偏移量，默认 "0"）、limit（每页条数，默认 "20"，最大 "100"）
	offsetStr := r.URL.Query().Get("offset")
	limitStr := r.URL.Query().Get("limit")
	offsetStr, limitStr = normalizePagingParams(offsetStr, limitStr)

	vpcClient, err := newVpcClient(r.Context())
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgCreateVPCClientFailed))
		return
	}

	req := vpc.NewDescribeVpcsRequest()
	req.Offset = common.StringPtr(offsetStr)
	req.Limit = common.StringPtr(limitStr)

	var filters []*vpc.Filter

	// 快捷参数：vpc_name -> Filter "vpc-name"
	if vpcName := r.URL.Query().Get("vpc_name"); vpcName != "" {
		filters = append(filters, &vpc.Filter{
			Name:   common.StringPtr("vpc-name"),
			Values: common.StringPtrs([]string{vpcName}),
		})
	}
	// 快捷参数：vpc_id -> Filter "vpc-id"
	if vpcId := r.URL.Query().Get("vpc_id"); vpcId != "" {
		filters = append(filters, &vpc.Filter{
			Name:   common.StringPtr("vpc-id"),
			Values: common.StringPtrs([]string{vpcId}),
		})
	}

	if len(filters) > 0 {
		req.Filters = filters
	}

	// reqJSON, _ := json.Marshal(req)
	// slog.Info("DescribeVpcs", "req", string(reqJSON))

	resp, err := vpcClient.DescribeVpcs(req)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQueryVPCListFailed))
		return
	}

	type vpcItem struct {
		VpcId     string `json:"vpc_id"`
		Name      string `json:"name"`
		CidrBlock string `json:"cidr_block"`
	}
	var vpcs []vpcItem
	var totalCount uint64
	if resp.Response != nil {
		if resp.Response.TotalCount != nil {
			totalCount = *resp.Response.TotalCount
		}
		for _, v := range resp.Response.VpcSet {
			item := vpcItem{}
			if v.VpcId != nil {
				item.VpcId = *v.VpcId
			}
			if v.VpcName != nil {
				item.Name = *v.VpcName
			}
			if v.CidrBlock != nil {
				item.CidrBlock = *v.CidrBlock
			}
			vpcs = append(vpcs, item)
		}
	}
	if vpcs == nil {
		vpcs = []vpcItem{}
	}

	jsonOK(w, map[string]interface{}{
		"vpcs":        vpcs,
		"total_count": totalCount,
	})
}

func HandleListCloudSubnets(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}

	vpcId := r.URL.Query().Get("vpc_id")
	zone := r.URL.Query().Get("zone")
	if vpcId == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgVpcIDParamRequired))
		return
	}
	if zone == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgZoneParamRequired))
		return
	}

	vpcClient, err := newVpcClient(r.Context())
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgCreateVPCClientFailed))
		return
	}

	req := vpc.NewDescribeSubnetsRequest()
	req.Filters = []*vpc.Filter{
		{
			Name:   common.StringPtr("vpc-id"),
			Values: common.StringPtrs([]string{vpcId}),
		},
		{
			Name:   common.StringPtr("zone"),
			Values: common.StringPtrs([]string{zone}),
		},
	}

	resp, err := vpcClient.DescribeSubnets(req)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgQuerySubnetListFailed))
		return
	}

	type subnetItem struct {
		SubnetId         string `json:"subnet_id"`
		Name             string `json:"name"`
		CidrBlock        string `json:"cidr_block"`
		AvailableIPCount uint64 `json:"available_ip_count"`
		TotalIPCount     uint64 `json:"total_ip_count"`
	}
	var subnets []subnetItem
	if resp.Response != nil {
		for _, s := range resp.Response.SubnetSet {
			item := subnetItem{}
			if s.SubnetId != nil {
				item.SubnetId = *s.SubnetId
			}
			if s.SubnetName != nil {
				item.Name = *s.SubnetName
			}
			if s.CidrBlock != nil {
				item.CidrBlock = *s.CidrBlock
			}
			if s.AvailableIpAddressCount != nil {
				item.AvailableIPCount = *s.AvailableIpAddressCount
			}
			if s.TotalIpAddressCount != nil {
				item.TotalIPCount = *s.TotalIpAddressCount
			}
			subnets = append(subnets, item)
		}
	}
	if subnets == nil {
		subnets = []subnetItem{}
	}

	jsonOK(w, map[string]interface{}{"subnets": subnets})
}

func newVpcClient(ctx context.Context) (*vpc.Client, error) {
	credential, err := getCredential(ctx)
	if err != nil {
		return nil, err
	}
	cpf := profile.NewClientProfile()
	return vpc.NewClient(credential, CVMRegion, cpf)
}

// checkSecurityGroupExists 检查指定安全组在云端是否真实存在。
// 使用包级变量以便单元测试中可替换。
var checkSecurityGroupExists = checkSecurityGroupExistsImpl

func checkSecurityGroupExistsImpl(ctx context.Context, securityGroupId string) (bool, error) {
	vpcClient, err := newVpcClient(ctx)
	if err != nil {
		return false, hcommon.I18nError(i18n.MsgCreateVPCClientFailed).WithDetail(err.Error())
	}
	return doCheckSecurityGroupExists(vpcClient, securityGroupId)
}

// doCheckSecurityGroupExists 执行安全组存在性检查的核心逻辑（与客户端创建解耦，便于测试）。
func doCheckSecurityGroupExists(vpcClient *vpc.Client, securityGroupId string) (bool, error) {
	req := vpc.NewDescribeSecurityGroupsRequest()
	req.SecurityGroupIds = common.StringPtrs([]string{securityGroupId})
	resp, err := vpcClient.DescribeSecurityGroups(req)
	if err != nil {
		if sdkErr, ok := err.(*sdkerrors.TencentCloudSDKError); ok && sdkErr.GetCode() == "ResourceNotFound" {
			slog.Warn("[checkSecurityGroupExists] 安全组不存在", "security_group_id", securityGroupId)
			return false, nil
		}
		return false, hcommon.I18nError(i18n.MsgVerifySecurityGroupFailed).WithDetail(err.Error())
	}

	if resp.Response == nil || len(resp.Response.SecurityGroupSet) == 0 {
		slog.Warn("[checkSecurityGroupExists] 安全组不存在", "security_group_id", securityGroupId)
		return false, nil
	}
	return true, nil
}

// checkGatewayUIPortRuleExists 检查安全组入站规则中是否存在指定端口的 TCP 放行规则。
// 支持匹配单端口、多端口（逗号分隔）、连续端口范围（如 8000-9000）以及 ALL。
func checkGatewayUIPortRuleExists(ctx context.Context, securityGroupId string, port int) (bool, error) {
	vpcClient, err := newVpcClient(ctx)
	if err != nil {
		return false, hcommon.I18nError(i18n.MsgCreateVPCClientFailed).WithDetail(err.Error())
	}

	req := vpc.NewDescribeSecurityGroupPoliciesRequest()
	req.SecurityGroupId = common.StringPtr(securityGroupId)
	resp, err := vpcClient.DescribeSecurityGroupPolicies(req)
	if err != nil {
		return false, hcommon.I18nError(i18n.MsgDescribeSGPoliciesFailed).WithDetail(err.Error())
	}

	if resp.Response == nil || resp.Response.SecurityGroupPolicySet == nil {
		return false, nil
	}

	for _, policy := range resp.Response.SecurityGroupPolicySet.Ingress {
		if policy.Action == nil || !strings.EqualFold(*policy.Action, "ACCEPT") {
			continue
		}
		// Protocol 为 ALL 或 TCP 均可覆盖 TCP 端口
		if policy.Protocol == nil {
			continue
		}
		proto := strings.ToUpper(*policy.Protocol)
		if proto != "TCP" && proto != "ALL" {
			continue
		}
		if policy.Port != nil && hcommon.PortMatchesRule(*policy.Port, port) {
			return true, nil
		}
	}
	return false, nil
}

// handleFirstTimeGatewayUIEnable 首次开启 Gateway UI 时分配端口并触发必需规则扇出。
//
// RuleSet 新模型下本函数职责已大幅简化：
//   - 分配 GatewayUIPort（resolveConditionalRules 依赖 GatewayUIPort > 0 才保留 allow_gateway_ui 规则组）
//   - 校验 SG 池就绪（否则无处下发规则）
//   - 标记 GatewayUISGMigrateDone=true 并落库
//   - 触发 RefreshAllRuleSetsForRequiredRules 将 allow_gateway_ui 端口规则扇出到所有 ACTIVE SG
//
// legacy 单 SG 迁移路径（migrateInstanceSecurityGroups 等）已废弃：
// 启动期 InitSGRuleSet 会把存量老 base SG 标 FROZEN 并建首个 ACTIVE 池，此后所有
// 租户都处在 RuleSet 模型下，不需要再走"把实例换绑到 siteConfig.SecurityGroupId"的逻辑。
func handleFirstTimeGatewayUIEnable(ctx context.Context, config *model.SiteConfig) error {
	// 1. 分配随机端口（已有则沿用）
	newPort := config.GatewayUIPort == 0
	if newPort {
		config.GatewayUIPort = model.GenerateGatewayUIPort()
	}

	// 2. 校验 SG 池就绪（RuleSet + 至少一个 ACTIVE SG）
	ready, rsCount, activeSGCount, err := HasSGPoolReady(ctx)
	if err != nil {
		return hcommon.I18nError(i18n.MsgQuerySGReadyFailed).WithDetail(err.Error())
	}
	if !ready {
		return hcommon.I18nError(i18n.MsgSGBootstrapNotDoneForGWUI, rsCount, activeSGCount)
	}

	// 3. 随后的 refresh 会重新从 DB 读取 SiteConfig；这些字段必须先落库：
	//    - GatewayUIEnable / GatewayUIAddrType / GatewayUIPort：决定 allow_gateway_ui 是否生效并替换端口占位符。
	//    - GatewayUISGMigrateDone：refresh 不读取；handler 依赖它跳过首次启用流程。
	config.GatewayUISGMigrateDone = true
	if err := model.SaveSelectedFields(ctx, config,
		"GatewayUIEnable",
		"GatewayUIAddrType",
		"GatewayUIPort",
		"GatewayUISGMigrateDone",
	); err != nil {
		return hcommon.I18nError(i18n.MsgSaveGatewayUIConfigFailed).WithDetail(err.Error())
	}

	slog.Info("[GatewayUIEnable] RuleSet 模式就绪，分配端口并触发规则扇出",
		"rule_set_count", rsCount, "active_sg_count", activeSGCount,
		"gateway_ui_port", config.GatewayUIPort, "new_port", newPort)

	// 4. 触发必需规则扇出：allow_gateway_ui 端口规则合入所有 RuleSet 并下发到 ACTIVE SG。
	//    失败仅告警、不阻断开关打开——管理员可通过「安全组管理」页立即重试刷新。
	if err := RefreshAllRuleSetsForRequiredRules(ctx); err != nil {
		slog.Warn("[GatewayUIEnable] 刷新 Gateway UI 端口规则失败，开关已打开但端口可能未放通",
			"port", config.GatewayUIPort, "error", err)
	}
	return nil
}

// ensureGatewayUIPortRule 开启时检查安全组端口规则是否被删除，如果被删除则重新添加。
func ensureGatewayUIPortRule(ctx context.Context, securityGroupId string, port int) {
	if securityGroupId == "" || port == 0 {
		return
	}
	portExists, err := checkGatewayUIPortRuleExists(ctx, securityGroupId, port)
	if err != nil {
		slog.Error("检查 Gateway UI 端口规则失败", "error", err)
		// 查询失败不阻断，继续保存配置
		return
	}
	if !portExists {
		if err := addGatewayUISecurityGroupRule(ctx, securityGroupId, port); err != nil {
			slog.Error("重新创建 Gateway UI 安全组规则失败", "port", port, "error", err)
		} else {
			slog.Info("Gateway UI 端口规则已被删除，已重新添加", "port", port)
		}
	}
}

// addGatewayUISecurityGroupRule 为 Gateway UI 端口创建安全组入站规则。
func addGatewayUISecurityGroupRule(ctx context.Context, securityGroupId string, port int) error {
	vpcClient, err := newVpcClient(ctx)
	if err != nil {
		return hcommon.I18nError(i18n.MsgCreateVPCClientFailed).WithDetail(err.Error())
	}

	portStr := strconv.Itoa(port)
	req := vpc.NewCreateSecurityGroupPoliciesRequest()
	req.SecurityGroupId = common.StringPtr(securityGroupId)
	req.SecurityGroupPolicySet = &vpc.SecurityGroupPolicySet{
		Ingress: []*vpc.SecurityGroupPolicy{
			{
				PolicyIndex:       common.Int64Ptr(0),
				CidrBlock:         common.StringPtr("0.0.0.0/0"),
				Protocol:          common.StringPtr("tcp"),
				Port:              common.StringPtr(portStr),
				Action:            common.StringPtr("ACCEPT"),
				PolicyDescription: common.StringPtr("用于OpenClaw面板"),
			},
		},
	}

	_, err = vpcClient.CreateSecurityGroupPolicies(req)
	if err != nil {
		return hcommon.I18nError(i18n.MsgCreateSGPoliciesFailed).WithDetail(err.Error())
	}
	slog.Info("Gateway UI 安全组规则创建成功", "security_group_id", securityGroupId, "port", port)
	return nil
}

// snakeToCamel 将 snake_case 字符串转换为大驼峰（PascalCase）
// 规则：按 "_" 分割，每个 word 首字母大写拼接
// 例如：internet_accessible → InternetAccessible, public_ip_assigned → PublicIpAssigned
func snakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		runes := []rune(p)
		runes[0] = unicode.ToUpper(runes[0])
		b.WriteString(string(runes))
	}
	return b.String()
}

// camelToSnake 将大驼峰（PascalCase）字符串转换为 snake_case
// 规则：在大写字母前插入 "_"，然后全部转为小写
// 例如：InternetAccessible → internet_accessible, PublicIpAssigned → public_ip_assigned
func camelToSnake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 {
			b.WriteByte('_')
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

// allowedTemplateKeys 白名单：可通过 /admin/config/template 修改的 top-level key（snake_case）
var allowedTemplateKeys = map[string]bool{
	"internet_accessible":     true,
	"system_disk":             true,
	"instance_type":           true,
	"instance_charge_type":    true,
	"instance_charge_prepaid": true,
}

// convertSnakeToCamelKeys 将请求中的 snake_case key 递归转换为 cvm_template 的大驼峰格式
func convertSnakeToCamelKeys(patch map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(patch))
	for key, val := range patch {
		camelKey := snakeToCamel(key)
		// 如果值是 map（子对象），递归转换子字段
		if subMap, ok := val.(map[string]interface{}); ok {
			val = convertSubFields(subMap)
		}
		result[camelKey] = val
	}
	return result
}

// convertSubFields 将子对象的 snake_case 字段名转换为大驼峰
func convertSubFields(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		result[snakeToCamel(k)] = v
	}
	return result
}

// HandleUpdateTemplate 通用模板修改接口
// POST /admin/config/template
// 请求体：JSON patch 对象，key 为 snake_case（与响应格式一致）
// 后端自动转换为 cvm_template 内部的大驼峰格式进行合并和存储
func HandleUpdateTemplate(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if !requireAdmin(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, hcommon.I18nError(i18n.MsgOnlyPostMethod))
		return
	}

	// 1. 解析请求体为 map（snake_case key）
	var patch map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgInvalidJSON))
		return
	}
	if len(patch) == 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgRequestBodyCannotBeEmpty))
		return
	}

	// 2. 白名单过滤（基于 snake_case key）
	var rejected []string
	for key := range patch {
		if !allowedTemplateKeys[key] {
			rejected = append(rejected, key)
		}
	}
	if len(rejected) > 0 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgFieldsNotAllowed, strings.Join(rejected, ", ")))
		return
	}

	// 3. 转换为大驼峰格式（与 cvm_template 内部格式一致）
	camelPatch := convertSnakeToCamelKeys(patch)

	// 4. 读取当前 template
	config := model.GetSiteConfig(r.Context())
	var tplMap map[string]interface{}
	tplSrc := config.CVMTemplate
	if tplSrc == "" {
		tplSrc = model.DefaultCVMTemplate
	}
	if err := json.Unmarshal([]byte(tplSrc), &tplMap); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgCurrentCVMTemplateInvalid))
		return
	}

	// 5. 对象级合并：有值替换、null 删除、未传保持
	for key, val := range camelPatch {
		if val == nil {
			delete(tplMap, key) // null → 删除
		} else {
			tplMap[key] = val // 非 null → 替换
		}
	}

	// 6. 分模块校验（基于合并后的完整 template，使用 snake_case 原始 key 判断）
	if _, ok := patch["internet_accessible"]; ok {
		if err := validateInternetAccessibleInTemplate(tplMap); err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
			return
		}
	}
	if _, ok := patch["instance_type"]; ok {
		if v, ok := tplMap["InstanceType"].(string); ok {
			if err := validateInstanceTypeWithCloud(r.Context(), v); err != nil {
				writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
				return
			}
		}
	}
	if _, ok := patch["system_disk"]; ok {
		if err := validateSystemDiskInTemplate(tplMap); err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
			return
		}
	}

	// 7. 序列化并保存
	updatedTpl, err := json.Marshal(tplMap)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgSerializeTemplateFailed))
		return
	}
	config.CVMTemplate = string(updatedTpl)
	if err := saveLegacyResourceConfig(r.Context(), &config, patch, false); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
		return
	}

	// 8. 构造响应
	resp := map[string]interface{}{
		"ok":           true,
		"cvm_template": string(updatedTpl),
		"message":      i18n.T(r.Context(), i18n.MsgTemplateConfigSaved),
	}
	if overview, err := model.ParseCVMTemplateOverview(string(updatedTpl)); err == nil && overview != nil && overview.InternetAccessible != nil {
		resp["internet_accessible"] = overview.InternetAccessible.ToResp()
	}
	jsonOK(w, resp)
}

// validateInternetAccessibleInTemplate 从合并后的 tplMap 中提取并校验 InternetAccessible
func validateInternetAccessibleInTemplate(tplMap map[string]interface{}) error {
	iaRaw, ok := tplMap["InternetAccessible"]
	if !ok {
		return nil
	}
	// 重新序列化再反序列化为结构体
	iaBytes, err := json.Marshal(iaRaw)
	if err != nil {
		return hcommon.I18nError(i18n.MsgInternetAccessibleFormatError).WithDetail(err.Error())
	}
	var ia model.InternetAccessible
	if err := json.Unmarshal(iaBytes, &ia); err != nil {
		return hcommon.I18nError(i18n.MsgInternetAccessibleParseError).WithDetail(err.Error())
	}

	model.NormalizeInternetAccessible(&ia)

	instanceChargeType := ""
	if v, ok := tplMap["InstanceChargeType"].(string); ok {
		instanceChargeType = v
	}
	if err := model.ValidateInternetAccessible(&ia, instanceChargeType); err != nil {
		return err
	}

	// 将 Normalize 后的结果写回 tplMap（不分配IP时不写入空字符串 InternetChargeType）
	iaMap := map[string]interface{}{
		"PublicIpAssigned":        ia.PublicIpAssigned,
		"InternetMaxBandwidthOut": ia.InternetMaxBandwidthOut,
	}
	if ia.PublicIpAssigned {
		iaMap["InternetChargeType"] = ia.InternetChargeType
	}
	tplMap["InternetAccessible"] = iaMap
	return nil
}

// validateSystemDiskInTemplate 从合并后的 tplMap 中提取并校验 SystemDisk
func validateSystemDiskInTemplate(tplMap map[string]interface{}) error {
	sdRaw, ok := tplMap["SystemDisk"]
	if !ok {
		return nil
	}
	sdMap, ok := sdRaw.(map[string]interface{})
	if !ok {
		return hcommon.I18nError(i18n.MsgSystemDiskFormatError)
	}

	// 校验 DiskType
	if diskTypeRaw, ok := sdMap["DiskType"]; ok {
		diskType, ok := diskTypeRaw.(string)
		if !ok {
			return hcommon.I18nError(i18n.MsgSystemDiskDiskTypeMustBeStr)
		}
		if err := model.ValidateDiskType(diskType); err != nil {
			return err
		}
	}

	// 校验 DiskSize（CVM API 定义为 *int64，必须为整数）
	if diskSizeRaw, ok := sdMap["DiskSize"]; ok {
		v, ok := diskSizeRaw.(float64)
		if !ok {
			return hcommon.I18nError(i18n.MsgSystemDiskDiskSizeMustBeNum)
		}
		if v != float64(int(v)) {
			return hcommon.I18nError(i18n.MsgSystemDiskDiskSizeMustBeInt, diskSizeRaw)
		}
		if err := model.ValidateSystemDisk(int(v)); err != nil {
			return err
		}
	}
	return nil
}

// validateInstanceTypeWithCloud 校验实例规格：
// 1. 调用云 API 获取当前 Region 所有可用机型
// 2. 用白名单对可用机型进行二次过滤，得到实际可选机型
// 3. 校验用户传入的机型是否在可选列表中
func validateInstanceTypeWithCloud(ctx context.Context, instanceType string) error {
	cvmClient, rerr := NewCVMClient(ctx)
	if rerr != nil {
		return rerr
	}

	resp, err := cvmClient.DescribeInstanceTypeConfigs(cvm.NewDescribeInstanceTypeConfigsRequest())
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgQueryCloudInstanceTypesFailed)
	}

	// 收集云端可用机型
	cloudTypes := make(map[string]bool)
	if resp.Response != nil {
		for _, item := range resp.Response.InstanceTypeConfigSet {
			if item.InstanceType != nil {
				cloudTypes[*item.InstanceType] = true
			}
		}
	}

	// 白名单与云端可用机型取交集
	var available []string
	for _, t := range model.AllowedInstanceTypes {
		if cloudTypes[t] {
			available = append(available, t)
		}
	}

	// 校验用户传入的机型是否在交集中
	if slices.Contains(available, instanceType) {
		return nil
	}

	if len(available) == 0 {
		return hcommon.I18nError(i18n.MsgInstanceTypeNotAvailableAll, instanceType, CVMRegion)
	}
	return hcommon.I18nError(i18n.MsgInstanceTypeNotAvailable, instanceType, strings.Join(available, ", "))
}

// extractTemplateSection 从 cvm_template JSON 中按 snake_case key 提取指定子模块，
// 并将内部大驼峰字段转换为 snake_case 返回。
// templatePath 为前端传入的 snake_case key，如 "internet_accessible"
func extractTemplateSection(cvmTemplate string, templatePath string) (interface{}, error) {
	// 校验 templatePath 是否在白名单中
	if !allowedTemplateKeys[templatePath] {
		return nil, hcommon.I18nError(i18n.MsgUnsupportedTemplatePath, templatePath)
	}
	camelKey := snakeToCamel(templatePath)

	if cvmTemplate == "" {
		cvmTemplate = model.DefaultCVMTemplate
	}

	var tplMap map[string]interface{}
	if err := json.Unmarshal([]byte(cvmTemplate), &tplMap); err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgCVMTemplateFormatError)
	}

	raw, exists := tplMap[camelKey]
	if !exists {
		return nil, nil
	}

	// 如果值是 map（子对象），将大驼峰字段名转换为 snake_case
	if subMap, ok := raw.(map[string]interface{}); ok {
		result := make(map[string]interface{}, len(subMap))
		for k, v := range subMap {
			result[camelToSnake(k)] = v
		}
		return result, nil
	}

	// 标量值（如 instance_type: "S5.MEDIUM4"）直接返回
	return raw, nil
}

// ========== Gateway UI 安全组迁移辅助函数 ==========

// createSystemSecurityGroup 创建 clawpro-default 安全组并添加出站全放通规则，写入 DB，返回新安全组 ID。
func createSystemSecurityGroup(ctx context.Context) (string, error) {
	vpcClient, err := newVpcClient(ctx)
	if err != nil {
		return "", hcommon.I18nError(i18n.MsgCreateVPCClientFailed).WithDetail(err.Error())
	}

	req := vpc.NewCreateSecurityGroupRequest()
	req.GroupName = common.StringPtr("clawpro-default")
	req.GroupDescription = common.StringPtr("clawpro默认安全组")

	resp, err := vpcClient.CreateSecurityGroup(req)
	if err != nil {
		return "", hcommon.I18nError(i18n.MsgCreateSecurityGroupCallFailed).WithDetail(err.Error())
	}
	if resp.Response == nil || resp.Response.SecurityGroup == nil || resp.Response.SecurityGroup.SecurityGroupId == nil {
		return "", hcommon.I18nError(i18n.MsgCreateSecurityGroupDataError)
	}

	sgId := *resp.Response.SecurityGroup.SecurityGroupId

	// 为新创建的安全组添加出站全放通规则
	egressReq := vpc.NewCreateSecurityGroupPoliciesRequest()
	egressReq.SecurityGroupId = common.StringPtr(sgId)
	egressReq.SecurityGroupPolicySet = &vpc.SecurityGroupPolicySet{
		Egress: []*vpc.SecurityGroupPolicy{
			{
				Protocol:          common.StringPtr("ALL"),
				Port:              common.StringPtr("ALL"),
				CidrBlock:         common.StringPtr("0.0.0.0/0"),
				Action:            common.StringPtr("ACCEPT"),
				PolicyDescription: common.StringPtr("全放通出站规则"),
			},
		},
	}
	if _, err := vpcClient.CreateSecurityGroupPolicies(egressReq); err != nil {
		slog.Error("为系统安全组添加出站规则失败", "security_group_id", sgId, "error", err)
		// 不中断流程，安全组已创建成功，出站规则可后续手动补充
	}

	config := model.GetSiteConfig(ctx)
	model.DB(ctx).Model(&config).Update("security_group_id", sgId)

	slog.Info("系统安全组创建成功", "security_group_id", sgId)
	return sgId, nil
}

// cloneSecurityGroup 从 srcSgId 克隆出新的 clawpro-default 安全组（含全部规则），写入 DB，返回新安全组 ID。
func cloneSecurityGroup(ctx context.Context, srcSgId string) (string, error) {
	vpcClient, err := newVpcClient(ctx)
	if err != nil {
		return "", hcommon.I18nError(i18n.MsgCreateVPCClientFailed).WithDetail(err.Error())
	}

	req := vpc.NewCloneSecurityGroupRequest()
	req.SecurityGroupId = common.StringPtr(srcSgId)
	req.GroupName = common.StringPtr("clawpro-default")
	req.GroupDescription = common.StringPtr("clawpro默认安全组")

	resp, err := vpcClient.CloneSecurityGroup(req)
	if err != nil {
		return "", hcommon.I18nError(i18n.MsgCloneSecurityGroupCallFailed).WithDetail(err.Error())
	}
	if resp.Response == nil || resp.Response.SecurityGroup == nil || resp.Response.SecurityGroup.SecurityGroupId == nil {
		return "", hcommon.I18nError(i18n.MsgCloneSecurityGroupDataError)
	}

	sgId := *resp.Response.SecurityGroup.SecurityGroupId
	config := model.GetSiteConfig(ctx)
	model.DB(ctx).Model(&config).Update("security_group_id", sgId)

	slog.Info("系统安全组克隆成功", "src", srcSgId, "security_group_id", sgId)
	return sgId, nil
}

// fetchAllDBInstances 从 DB 取所有有效实例的 InstanceId 列表（未删除且 instance_id 非空）。
func listInstanceIds(ctx context.Context) ([]string, error) {
	var instances []model.Instance
	if err := model.DB(ctx).Where("instance_id != ''").Find(&instances).Error; err != nil {
		return nil, hcommon.I18nError(i18n.MsgQueryInstanceListFailed).WithDetail(err.Error())
	}
	ids := make([]string, 0, len(instances))
	for _, inst := range instances {
		if inst.InstanceId != "" {
			ids = append(ids, inst.InstanceId)
		}
	}
	return ids, nil
}

// describeInstancesSecurityGroups 分批查询 CVM 实例当前安全组列表。
// 返回 map[instanceId][]sgId。
func describeInstancesSecurityGroups(ctx context.Context, instanceIds []string) (map[string][]string, error) {
	if len(instanceIds) == 0 {
		return map[string][]string{}, nil
	}

	cvmClient, err := NewCVMClient(ctx)
	if err != nil {
		return nil, hcommon.I18nError(i18n.MsgCreateCVMClientFailedFmt, err)
	}

	result := make(map[string][]string)
	const batchSize = 100

	for i := 0; i < len(instanceIds); i += batchSize {
		end := i + batchSize
		if end > len(instanceIds) {
			end = len(instanceIds)
		}
		batch := instanceIds[i:end]

		req := cvm.NewDescribeInstancesRequest()
		req.InstanceIds = common.StringPtrs(batch)
		req.Limit = common.Int64Ptr(int64(len(batch)))

		resp, err := cvmClient.DescribeInstances(req)
		if err != nil {
			return nil, hcommon.I18nError(i18n.MsgDescribeInstancesFailed).WithDetail(err.Error())
		}
		if resp.Response == nil {
			continue
		}
		for _, inst := range resp.Response.InstanceSet {
			if inst.InstanceId == nil {
				continue
			}
			sgIds := make([]string, 0, len(inst.SecurityGroupIds))
			for _, sg := range inst.SecurityGroupIds {
				if sg != nil {
					sgIds = append(sgIds, *sg)
				}
			}
			result[*inst.InstanceId] = sgIds
		}
	}

	return result, nil
}

// collectAllSgIds 收集 instanceSgMap 中所有出现的安全组 ID（去重）。
func collectAllSgIds(instanceSgMap map[string][]string) []string {
	seen := make(map[string]struct{})
	var result []string
	for _, sgList := range instanceSgMap {
		for _, sg := range sgList {
			if _, ok := seen[sg]; !ok {
				seen[sg] = struct{}{}
				result = append(result, sg)
			}
		}
	}
	return result
}

// findDefaultSgIdFromList 在一批安全组 ID 中查询，找出 IsDefault==true 的那个。
// 若找不到返回 ("", nil)。DescribeSecurityGroups 单次最多 100 个，超出时分批查询。
func findDefaultSgIdFromList(ctx context.Context, sgIds []string) (string, error) {
	if len(sgIds) == 0 {
		return "", nil
	}

	vpcClient, err := newVpcClient(ctx)
	if err != nil {
		return "", hcommon.I18nError(i18n.MsgCreateVPCClientFailed).WithDetail(err.Error())
	}

	const batchSize = 100
	for i := 0; i < len(sgIds); i += batchSize {
		end := min(i+batchSize, len(sgIds))
		batch := sgIds[i:end]

		req := vpc.NewDescribeSecurityGroupsRequest()
		req.SecurityGroupIds = common.StringPtrs(batch)

		resp, err := vpcClient.DescribeSecurityGroups(req)
		if err != nil {
			return "", hcommon.I18nError(i18n.MsgDescribeSecurityGroupsFailed).WithDetail(err.Error())
		}
		if resp.Response == nil {
			continue
		}

		for _, sg := range resp.Response.SecurityGroupSet {
			if sg.IsDefault != nil && *sg.IsDefault && sg.SecurityGroupId != nil {
				return *sg.SecurityGroupId, nil
			}
		}
	}
	return "", nil
}

// replaceInSlice 将 slice 中的 oldVal 替换为 newVal。
func replaceInSlice(slice []string, oldVal, newVal string) []string {
	result := make([]string, len(slice))
	for i, s := range slice {
		if s == oldVal {
			result[i] = newVal
		} else {
			result[i] = s
		}
	}
	return result
}

// migrateInstanceSecurityGroups 对所有实例执行安全组替换/追加逻辑，同步执行。
// defaultSgId 为空字符串时只追加，不替换。
func migrateInstanceSecurityGroups(
	ctx context.Context,
	instanceSgMap map[string][]string,
	systemSgId string,
	defaultSgId string,
) error {
	if len(instanceSgMap) == 0 {
		return nil
	}

	type migrateItem struct {
		instanceId string
		newSgList  []string
	}
	var items []migrateItem

	skipped := 0
	replaced := 0
	appended := 0

	for instanceId, sgList := range instanceSgMap {
		if slices.Contains(sgList, systemSgId) {
			skipped++
			continue
		}
		var newList []string
		if defaultSgId != "" && slices.Contains(sgList, defaultSgId) {
			newList = replaceInSlice(sgList, defaultSgId, systemSgId)
			replaced++
		} else {
			newList = append(append([]string{}, sgList...), systemSgId)
			appended++
		}
		items = append(items, migrateItem{instanceId: instanceId, newSgList: newList})
	}

	slog.Info("开始迁移实例安全组", "total_instances", len(instanceSgMap),
		"to_modify", len(items), "skipped", skipped)

	if len(items) == 0 {
		slog.Info("实例安全组迁移完成", "replaced", replaced, "appended", appended, "skipped", skipped)
		return nil
	}

	cvmClient, err := NewCVMClient(ctx)
	if err != nil {
		return hcommon.I18nError(i18n.MsgCreateCVMClientFailedFmt, err)
	}

	var lastErr error
	successCount := 0

	// 按目标安全组列表分组，同一目标列表的实例合并为一批（最多 100 个）
	groups := make(map[string][]migrateItem)
	for _, item := range items {
		key := strings.Join(item.newSgList, ",")
		groups[key] = append(groups[key], item)
	}

	const batchSize = 100
	for _, groupItems := range groups {
		for i := 0; i < len(groupItems); i += batchSize {
			end := i + batchSize
			if end > len(groupItems) {
				end = len(groupItems)
			}
			batch := groupItems[i:end]

			instanceIds := make([]string, len(batch))
			for j, item := range batch {
				instanceIds[j] = item.instanceId
			}
			targetSgList := batch[0].newSgList

			req := cvm.NewModifyInstancesAttributeRequest()
			req.InstanceIds = common.StringPtrs(instanceIds)
			req.SecurityGroups = common.StringPtrs(targetSgList)

			_, err := cvmClient.ModifyInstancesAttribute(req)
			if err != nil {
				slog.Error("实例安全组迁移失败", "instance_ids", instanceIds, "error", err)
				lastErr = err
			} else {
				successCount += len(batch)
			}
		}
	}

	slog.Info("实例安全组迁移完成", "replaced", replaced, "appended", appended, "skipped", skipped, "success", successCount)
	return lastErr
}

// ==================== 导出函数（供 task 包中的临时修复脚本调用） start ====================
// 注意：以下导出函数仅供 task/fix_default_sg_rebind.go 使用，
// 待该临时修复脚本删除后，这些导出函数也应一并删除。

// ListInstanceIds 查询数据库中所有有效实例的 CVM InstanceId 列表。
func ListInstanceIds(ctx context.Context) ([]string, error) {
	return listInstanceIds(ctx)
}

// DescribeInstancesSecurityGroups 批量查询 CVM 实例当前绑定的安全组列表。
func DescribeInstancesSecurityGroups(ctx context.Context, instanceIds []string) (map[string][]string, error) {
	return describeInstancesSecurityGroups(ctx, instanceIds)
}

// CollectAllSgIds 从实例安全组映射中收集所有不重复的安全组 ID。
func CollectAllSgIds(instanceSgMap map[string][]string) []string {
	return collectAllSgIds(instanceSgMap)
}

// FindDefaultSgIdFromList 在一批安全组 ID 中查询，找出 IsDefault==true 的那个。
func FindDefaultSgIdFromList(ctx context.Context, sgIds []string) (string, error) {
	return findDefaultSgIdFromList(ctx, sgIds)
}

// NewVpcClient 创建腾讯云 VPC 客户端。
func NewVpcClient(ctx context.Context) (*vpc.Client, error) {
	return newVpcClient(ctx)
}

// ==================== 导出函数（供 task 包中的临时修复脚本调用）end ====================
