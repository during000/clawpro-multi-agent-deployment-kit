package main

import (
	"bytes"
	"context"
	"flag"
	"io"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
	"time"
	_ "time/tzdata"

	"hatchery/common"
	"hatchery/controller"
	"hatchery/i18n"
	"hatchery/model"
	"hatchery/task"

	"github.com/dchest/captcha"
	"github.com/gorilla/sessions"
)

func deprecatedFlags() map[string]interface{} {
	flags := make(map[string]interface{})

	_ = flag.Bool("disable-ui", true, "deprecated: disable HTML UI rendering (API only)")
	_ = flag.Int("user-limit", 1000, "deprecated: user limit is no longer enforced")

	flags["disable-ui"] = true
	flags["user-limit"] = 1000

	return flags
}

func applyFlags(lang string) {
	i18n.SetDefaultLang(lang)
}

func main() {
	addr := flag.String("addr", ":8080", "listen address (e.g. :8080, 127.0.0.1:8080, [::]:8080)")
	secretKey := flag.String("secret", "hatchery-secret-key-change-me", "cookie secret key for session encryption")
	dbPath := flag.String("db", "hatchery.db", "path to SQLite database file or MySQL DSN")
	dbType := flag.String("db-type", "sqlite", "database type: sqlite or mysql")
	initUser := flag.String("init-user", "", "initial admin username (required on first run)")
	initPass := flag.String("init-pass", "", "initial admin password (required on first run)")
	adminToken := flag.String("admin-token", "", "Bearer token for admin API authentication (optional)")
	logJSON := flag.Bool("log-json", false, "output logs in JSON format")
	cvmUin := flag.String("uin", "", "CVM UIN for Tencent Cloud")
	identifier := flag.String("identifier", "", "multi-tenant identifier for data isolation")
	domain := flag.String("domain", "", "base URL of this service, e.g. https://hatchery.example.com")
	cvmRegion := flag.String("region", "", "CVM region, required (e.g. ap-guangzhou, ap-beijing)")
	emailAPIURL := flag.String("email-api-url", "", "email API URL, e.g. http://cvm.test.tencentcloudapi.com")
	internalSecret := flag.String("internal-secret", "", "HMAC shared secret for Gateway internal-login tokens")
	dbMigrate := flag.String("db-migrate", "", "source SQLite DB path for one-time data migration to MySQL (requires --db-type=mysql and --identifier)")
	universe := flag.Bool("universe", false, "enable universe multi-tenant mode (requires --db-type=mysql, mutually exclusive with --identifier)")
	lang := flag.String("lang", "zh", "default language: zh or en. Note: This option is also used to determine whether it is the overseas version. If the default language is not Chinese, it is the overseas version")
	ssrf := flag.Bool("ssrf", true, "enable ssrf protection")

	deprecatedFlagsMap := deprecatedFlags()

	flag.Parse()

	applyFlags(*lang)

	// 优先初始化 slog，确保后续所有日志（包括参数校验失败的错误日志）都使用统一格式和 hostname 字段。
	hostname, _ := os.Hostname()
	logLevel := &slog.LevelVar{} // 默认 INFO，支持运行时动态调整
	if *logJSON {
		handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}).
			WithAttrs([]slog.Attr{slog.String("hostname", hostname)})
		slog.SetDefault(slog.New(handler))
	} else {
		handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}).
			WithAttrs([]slog.Attr{slog.String("hostname", hostname)})
		slog.SetDefault(slog.New(handler))
	}

	if *cvmRegion == "" {
		slog.Error("--region is required")
		os.Exit(1)
	}
	if *domain == "" && !*universe {
		slog.Error("--domain is required")
		os.Exit(1)
	}
	if *dbType == "mysql" && *identifier == "" && !*universe {
		slog.Error("--identifier or --universe is required when using MySQL backend")
		os.Exit(1)
	}
	if *universe && *dbType != "mysql" {
		slog.Error("--universe requires --db-type=mysql")
		os.Exit(1)
	}
	if *universe && *identifier != "" {
		slog.Error("--universe and --identifier are mutually exclusive")
		os.Exit(1)
	}
	if *dbMigrate != "" && *dbType != "mysql" {
		slog.Error("--db-migrate requires --db-type=mysql")
		os.Exit(1)
	}
	if ri, ok := controller.Regions[*cvmRegion]; !ok || len(ri.Zones) == 0 {
		slog.Error("unknown region", "region", *cvmRegion, "available", func() []string {
			keys := make([]string, 0, len(controller.Regions))
			for k, v := range controller.Regions {
				if len(v.Zones) > 0 {
					keys = append(keys, k)
				}
			}
			return keys
		}())
		os.Exit(1)
	}

	controller.AdminToken = *adminToken
	if controller.AdminToken == "" {
		controller.AdminToken = os.Getenv("ADMIN_TOKEN")
	}

	if *secretKey == "hatchery-secret-key-change-me" && controller.AdminToken != "" {
		*secretKey = controller.AdminToken
	}

	if *initUser == "" {
		*initUser = os.Getenv("INIT_USER")
	}
	if *initPass == "" {
		*initPass = os.Getenv("INIT_PASS")
	}

	model.DefaultRolesJSON = loadDefaultRoles()
	controller.ClawproRequiredSGRulesJSON = loadClawproRequiredSGRules()
	model.PluginVersionMapJSON = loadPluginVersionMap()
	model.InitDB(*dbPath, *dbType, *identifier, *initUser, *initPass, *dbMigrate, *universe)

	securityPolicies := make([]string, 0)
	if *ssrf {
		securityPolicies = append(securityPolicies, "SSRF")
	}

	// —— 多租户：回填租户级字段到 site_configs，并构造 FixedSnapshot ——
	// universe 模式下跳过（没有固定的单租户 identifier，由 Host 动态路由）
	if !*universe {
		cmdlineConfig := CmdlineConfig{
			Identifier:       *identifier,
			Uin:              *cvmUin,
			Domain:           *domain,
			InternalSecret:   *internalSecret,
			Lang:             *lang,
			SecurityPolicies: securityPolicies,
		}
		// 必须紧跟 InitDB 之后执行，后续所有 DB 操作（含 MigrateMemoryPlanFromStatus）
		// 依赖 FixedSnapshot 通过 model.DB(ctx) 正确注入 identifier。
		backfillSiteConfig(cmdlineConfig)
		buildFixedSnapshot(cmdlineConfig)
	}

	controller.RegisterDBLogger(model.DB(context.Background()))
	// 业务时区按 region 注入（如 Asia/Shanghai）：配额日期汇总与定时任务表达式统一来源，
	// 与容器 TZ 解耦，避免线上未配 TZ 时回退 UTC。
	common.SetBusinessTimezone(controller.Regions[*cvmRegion].Timezone)

	// LoadScript 必须在 HTTP 服务启动前赋值（handler 中会调用）
	controller.LoadScript = loadScript

	controller.CVMRegion = *cvmRegion
	if *emailAPIURL != "" {
		controller.EmailAPIURL = *emailAPIURL
	} else if *cvmRegion != "" {
		controller.EmailAPIURL = "https://cvm." + *cvmRegion + ".tencentcloudapi.com"
	}
	controller.DisableUI = deprecatedFlagsMap["disable-ui"].(bool)
	controller.GatewayURL = os.Getenv("GATEWAY_URL")

	controller.Store = sessions.NewCookieStore([]byte(*secretKey))
	controller.Store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 天
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}

	http.Handle("/captcha/", captcha.Server(captcha.StdWidth, captcha.StdHeight))
	http.HandleFunc("/", controller.HandleIndex)
	http.HandleFunc("/login", controller.HandleLogin)
	http.HandleFunc("/auth/passwordless-login", controller.WithAudit(controller.HandlePasswordlessLogin))
	http.HandleFunc("/admin/passwordless-login-link", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminCreatePasswordlessLoginLink)))
	http.HandleFunc("/logout", controller.HandleLogout)
	http.HandleFunc("/change-password", controller.WithAudit(controller.HandleChangePassword))
	http.HandleFunc("/openclaw/list", controller.WithOpenAPI(controller.HandleInstanceList))
	http.HandleFunc("/openclaw/current-image", controller.WithOpenAPI(controller.HandleCurrentImage))
	http.HandleFunc("/openclaw/delete", controller.WithAudit(controller.WithOpenAPI(controller.HandleDeleteInstance)))
	http.HandleFunc("/openclaw/start", controller.WithAudit(controller.WithOpenAPI(controller.HandleStartInstance)))
	http.HandleFunc("/openclaw/stop", controller.WithAudit(controller.WithOpenAPI(controller.HandleStopInstance)))
	http.HandleFunc("/openclaw/reboot", controller.WithAudit(controller.WithOpenAPI(controller.HandleRebootInstance)))
	http.HandleFunc("/openclaw/restart-gateway", controller.WithAudit(controller.WithOpenAPI(controller.HandleRestartGatewayInstance)))
	http.HandleFunc("/openclaw/reset", controller.WithAudit(controller.WithOpenAPI(controller.HandleResetInstance)))
	http.HandleFunc("/openclaw/create", controller.WithAudit(controller.WithOpenAPI(controller.HandleCreateInstance)))
	http.HandleFunc("/openclaw/status", controller.WithOpenAPI(controller.HandleInstanceStatus))
	http.HandleFunc("/openclaw/retry", controller.WithAudit(controller.WithOpenAPI(controller.HandleRetryInstance)))
	http.HandleFunc("/openclaw/rename", controller.WithAudit(controller.WithOpenAPI(controller.HandleRenameInstance)))
	http.HandleFunc("/openclaw/notifications", controller.WithOpenAPI(controller.HandleGetNotifications))
	http.HandleFunc("/openclaw/notifications/read", controller.WithAudit(controller.WithOpenAPI(controller.HandleReadNotification)))
	http.HandleFunc("/openclaw/notifications/count", controller.WithOpenAPI(controller.HandleGetUnreadCount))
	http.HandleFunc("/openclaw/notifications/delete", controller.WithAudit(controller.WithOpenAPI(controller.HandleDeleteNotification)))
	http.HandleFunc("/openclaw/service-status", controller.WithOpenAPI(controller.HandleServiceStatus))
	http.HandleFunc("/openclaw/check-openclaw-port", controller.WithOpenAPI(controller.HandleCheckAgentReady))
	http.HandleFunc("/openclaw/approve", controller.WithAudit(controller.WithOpenAPI(controller.HandleApprove)))
	handleSkillsList, handleUserUpdateSkill, handleUserUninstallSkill := controller.NewUserSkillHandlers()
	http.HandleFunc("/openclaw/skills", controller.WithOpenAPI(handleSkillsList))
	http.HandleFunc("/openclaw/add-skill", controller.WithAudit(controller.WithOpenAPI(controller.HandleAddSkill)))
	http.HandleFunc("/openclaw/update-skill", controller.WithAudit(controller.WithOpenAPI(handleUserUpdateSkill)))
	http.HandleFunc("/openclaw/uninstall-skill", controller.WithAudit(controller.WithOpenAPI(handleUserUninstallSkill)))
	http.HandleFunc("/openclaw/upgrade", controller.WithAudit(controller.WithOpenAPI(controller.HandleUpgrade)))
	http.HandleFunc("/openclaw/upgrade/retry", controller.WithAudit(controller.WithOpenAPI(controller.HandleUpgradeRetry)))
	http.HandleFunc("/openclaw/migration/export", controller.WithAudit(controller.WithOpenAPI(controller.HandleMigrationExport)))
	http.HandleFunc("/openclaw/migration/status", controller.WithOpenAPI(controller.HandleMigrationStatus))
	http.HandleFunc("/openclaw/migration/progress", controller.WithOpenAPI(controller.HandleMigrationProgress))
	http.HandleFunc("/openclaw/migration/import", controller.WithAudit(controller.WithOpenAPI(controller.HandleMigrationImport)))
	http.HandleFunc("/openclaw/add-plugin", controller.WithAudit(controller.WithOpenAPI(controller.HandleAddPlugin)))
	http.HandleFunc("/openclaw/install-skills", controller.WithOpenAPI(controller.HandleInstallSkills))
	http.HandleFunc("/openclaw/local/pending-skills", controller.WithOpenAPI(controller.HandleLocalPendingSkillsRouter))

	http.HandleFunc("/openclaw/retry-failed-skills", controller.WithAudit(controller.WithOpenAPI(controller.HandleRetryFailedSkills)))
	http.HandleFunc("/openclaw/cancel-failed-skills", controller.WithAudit(controller.WithOpenAPI(controller.HandleCancelFailedSkills)))

	// 本地 agent reporter 接口（clawpro 一期/二期）
	http.HandleFunc("/local-agent/report", controller.WithAudit(controller.WithOpenAPI(controller.HandleLocalAgentReport)))
	http.HandleFunc("/local-agent/sync", controller.WithAudit(controller.WithOpenAPI(controller.HandleLocalAgentSync)))
	http.HandleFunc("/local-agent/commands/ack", controller.WithAudit(controller.WithOpenAPI(controller.HandleLocalAgentAck)))
	http.HandleFunc("/local-agent/wake-ticket", controller.WithAudit(controller.WithOpenAPI(controller.HandleLocalAgentWakeTicket)))
	http.HandleFunc("/local-agent/wake", controller.WithOpenAPI(controller.HandleLocalAgentWake))
	http.HandleFunc("/local-agent/availability", controller.WithOpenAPI(controller.HandleLocalAgentAvailability))
	http.HandleFunc("/local-agent/get-config", controller.WithOpenAPI(controller.HandleLocalAgentGetConfig))
	// ClawPro → TeamAI/Edge Runtime 本地 Agent 任务
	http.HandleFunc("/agent-tasks", controller.WithOpenAPI(controller.HandleAgentTasks))
	http.HandleFunc("/agent-tasks/create", controller.WithAudit(controller.WithOpenAPI(controller.HandleAgentTaskCreate)))
	// 本地 agent 三期：移除本地 Agent（用户端 / 管控端）
	http.HandleFunc("/local-agent/remove", controller.WithAudit(controller.WithOpenAPI(controller.HandleLocalAgentRemove)))
	http.HandleFunc("/admin/local-agent/remove", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminLocalAgentRemove)))
	http.HandleFunc("/projects/mine", controller.WithOpenAPI(controller.HandleProjectsMine))
	// 本地 agent 二期：前端切换用户级分组
	http.HandleFunc("/openclaw/local/user-group", controller.WithAudit(controller.WithOpenAPI(controller.HandleSwitchUserLevelGroup)))
	http.HandleFunc("/admin/feature-allowlist/check", controller.WithOpenAPI(controller.HandleAdminFeatureAllowlistCheck))
	http.HandleFunc("/admin/local-agent-types", controller.WithOpenAPI(controller.HandleAdminLocalAgentTypes))
	http.HandleFunc("/openclaw/channels", controller.WithOpenAPI(controller.HandleChannelsList))
	http.HandleFunc("/openclaw/proxy/prepare", controller.WithAudit(controller.WithOpenAPI(controller.HandleProxyPrepare)))
	http.HandleFunc("/openclaw/set-channel", controller.WithAudit(controller.WithOpenAPI(controller.HandleSetChannel)))
	http.HandleFunc("/openclaw/del-channel", controller.WithAudit(controller.WithOpenAPI(controller.HandleDelChannel)))
	http.HandleFunc("/openclaw/auto-channel", controller.WithAudit(controller.WithOpenAPI(controller.HandleAutoChannel)))
	http.HandleFunc("/openclaw/models", controller.WithOpenAPI(controller.HandleModelsList))
	http.HandleFunc("/openclaw/models/connectivity", controller.WithOpenAPI(controller.HandleModelConnectivity))
	http.HandleFunc("/openclaw/set-model", controller.WithAudit(controller.WithOpenAPI(controller.HandleSetModel)))
	http.HandleFunc("/openclaw/config-overview", controller.WithOpenAPI(controller.HandleOpenClawConfigOverview))
	// 多模型 Fallback 接口（v2.0）
	http.HandleFunc("/openclaw/add-model", controller.WithAudit(controller.WithOpenAPI(controller.HandleAddModel)))
	http.HandleFunc("/openclaw/switch-primary-model", controller.WithAudit(controller.WithOpenAPI(controller.HandleSwitchPrimaryModel)))
	http.HandleFunc("/openclaw/del-model", controller.WithAudit(controller.WithOpenAPI(controller.HandleDelModel)))
	http.HandleFunc("/openclaw/instance-models", controller.WithOpenAPI(controller.HandleInstanceModels))
	http.HandleFunc("/openclaw/version", controller.WithOpenAPI(controller.HandleVersionInfo))
	http.HandleFunc("/openclaw/agent-count", controller.WithOpenAPI(controller.HandleAgentCount))

	// 用户端 MCP 管理
	http.HandleFunc("/openclaw/mcp/available", controller.WithOpenAPI(controller.HandleUserMcpAvailable))
	http.HandleFunc("/openclaw/mcp/add", controller.WithAudit(controller.WithOpenAPI(controller.HandleUserMcpAdd)))
	http.HandleFunc("/openclaw/mcp/list", controller.WithOpenAPI(controller.HandleUserMcpList))
	http.HandleFunc("/openclaw/mcp/refresh-status", controller.WithOpenAPI(controller.HandleUserMcpRefreshStatus))
	http.HandleFunc("/openclaw/mcp/update-config", controller.WithAudit(controller.WithOpenAPI(controller.HandleUserMcpUpdateConfig)))
	http.HandleFunc("/openclaw/mcp/delete", controller.WithAudit(controller.WithOpenAPI(controller.HandleUserMcpDelete)))
	http.HandleFunc("/openclaw/mcp/toggle", controller.WithAudit(controller.WithOpenAPI(controller.HandleUserMcpToggle)))
	http.HandleFunc("/openclaw/zones", controller.WithOpenAPI(controller.HandleDescribeZones))
	http.HandleFunc("/openclaw/agent-types", controller.WithOpenAPI(controller.HandleUserAgentTypes))
	http.HandleFunc("/openclaw/images/update-notices", controller.WithOpenAPI(controller.HandleUserImageUpdateNotices))
	http.HandleFunc("/openclaw/denied-actions", controller.WithOpenAPI(controller.HandleInstanceDeniedActions))
	http.HandleFunc("/openclaw/terminal-url", controller.WithAudit(controller.WithOpenAPI(controller.HandleInstanceTerminal)))
	http.HandleFunc("/openclaw/set-gateway-ui", controller.WithAudit(controller.WithOpenAPI(controller.HandleSetGatewayUi)))
	http.HandleFunc("/openclaw/check-gateway-access", controller.WithOpenAPI(controller.HandleCheckGatewayAccess))
	http.HandleFunc("/openclaw/ws-url", controller.WithAudit(controller.WithOpenAPI(controller.HandleGetWSUrl)))
	// Agent-Bridge 回调接口（通过用户 API Token 进行 Bearer Token 鉴权，复用 WithOpenAPI + requireLogin）
	http.HandleFunc("/agent-bridge/sts", controller.WithOpenAPI(controller.HandleAgentBridgeSTS))
	http.HandleFunc("/agent-bridge/auth", controller.WithOpenAPI(controller.HandleAgentBridgeAuth))
	http.HandleFunc("/agent-bridge/instances", controller.WithOpenAPI(controller.HandleAgentBridgeInstances))
	http.HandleFunc("/agent-bridge/audit", controller.WithOpenAPI(controller.HandleAgentBridgeAudit))

	// 云端浏览器（Browser VNC）
	http.HandleFunc("/openclaw/browser-vnc-access", controller.WithOpenAPI(controller.HandleBrowserVNCAccess))
	http.HandleFunc("/openclaw/browser-vnc-check", controller.WithOpenAPI(controller.HandleBrowserVNCCheck))
	http.HandleFunc("/openclaw/browser-vnc-install", controller.WithAudit(controller.WithOpenAPI(controller.HandleBrowserVNCInstall)))
	http.HandleFunc("/openclaw/browser-status", controller.WithOpenAPI(controller.HandleBrowserStatus))
	http.HandleFunc("/openclaw/browser-takeover", controller.WithAudit(controller.WithOpenAPI(controller.HandleBrowserTakeover)))
	http.HandleFunc("/openclaw/vnc-ws-proxy", controller.HandleBrowserVNCProxy)
	http.HandleFunc("/openclaw/set-env", controller.WithAudit(controller.WithOpenAPI(controller.HandleSetEnv)))
	http.HandleFunc("/openclaw/env", controller.WithOpenAPI(controller.HandleGetEnv))
	http.HandleFunc("/openclaw/smh-status", controller.WithOpenAPI(controller.HandleOpenClawSMHStatus))
	http.HandleFunc("/openclaw/smh-token", controller.WithOpenAPI(controller.HandleOpenClawSMHToken))
	http.HandleFunc("/openclaw/memory-tdai-status", controller.WithOpenAPI(controller.HandleOpenClawMemoryTDAIStatus))
	// LightClaw 对接接口
	http.HandleFunc("/openclaw/lightclaw/token", controller.HandleLightClawToken)
	http.HandleFunc("/openclaw/lightclaw/auth", controller.HandleLightClawAuth)
	http.HandleFunc("/openclaw/lightclaw/describe-invocations", controller.HandleLightClawDescribeInvocations)
	http.HandleFunc("/openclaw/lightclaw/describe-invocation-tasks", controller.HandleLightClawDescribeInvocationTasks)
	http.HandleFunc("/openclaw/lightclaw/run-command", controller.WithAudit(controller.HandleLightClawRunCommand))

	// 龙虾医院接口
	http.HandleFunc("/openclaw/doctor/quick-fix", controller.WithAudit(controller.WithOpenAPI(controller.HandleDoctorQuickFix)))
	http.HandleFunc("/openclaw/doctor/quick-fix/status", controller.WithOpenAPI(controller.HandleDoctorQuickFixStatus))
	http.HandleFunc("/openclaw/doctor/feature", controller.WithOpenAPI(controller.HandleDoctorFeature))
	http.HandleFunc("/openclaw/doctor/authorize", controller.WithAudit(controller.WithOpenAPI(controller.HandleDoctorAuthorize)))
	http.HandleFunc("/openclaw/doctor/start", controller.WithAudit(controller.WithOpenAPI(controller.HandleDoctorStart)))
	http.HandleFunc("/openclaw/doctor/status", controller.WithOpenAPI(controller.HandleDoctorStatus))
	http.HandleFunc("/openclaw/doctor/end", controller.WithAudit(controller.WithOpenAPI(controller.HandleDoctorEnd)))

	// ── 技能广场（Skill Store） ──
	http.HandleFunc("/openclaw/skillstore", controller.WithOpenAPI(controller.HandleSkillStore))
	http.HandleFunc("/openclaw/skillstore/detail", controller.WithOpenAPI(controller.HandleSkillStoreDetail))
	http.HandleFunc("/openclaw/skillstore/categories", controller.WithOpenAPI(controller.HandleSkillStoreCategories))
	http.HandleFunc("/openclaw/skillstore/instances", controller.WithOpenAPI(controller.HandleSkillStoreInstances))
	http.HandleFunc("/openclaw/skillstore/tasks", controller.WithOpenAPI(controller.HandleSkillStoreTasks))
	http.HandleFunc("/openclaw/skillstore/distribute", controller.WithAudit(controller.WithOpenAPI(controller.HandleSkillStoreDistribute)))
	http.HandleFunc("/openclaw/skillstore/uninstall", controller.WithAudit(controller.WithOpenAPI(controller.HandleSkillStoreUninstall)))
	http.HandleFunc("/openclaw/skillstore/download", controller.WithAudit(controller.WithOpenAPI(controller.HandleSkillStoreDownload)))

	// ── 技能共建审核（员工端）──
	http.HandleFunc("/openclaw/skills/contribute", controller.WithAudit(controller.WithOpenAPI(controller.HandleContributeSkill)))
	http.HandleFunc("/openclaw/skills/takedown", controller.WithAudit(controller.WithOpenAPI(controller.HandleTakedownSkill)))
	http.HandleFunc("/openclaw/skills/contributions", controller.WithOpenAPI(controller.HandleMyContributions))
	http.HandleFunc("/openclaw/skills/contributions/detail", controller.WithOpenAPI(controller.HandleMyContributionDetail))

	// ── 技能共建审核（管理员端）──
	http.HandleFunc("/admin/contributions", controller.WithOpenAPI(controller.HandleAdminContributions))
	http.HandleFunc("/admin/contributions/detail", controller.WithOpenAPI(controller.HandleAdminContributionDetail))
	http.HandleFunc("/admin/contributions/approve", controller.WithAudit(controller.WithOpenAPI(controller.HandleApproveContribution)))
	http.HandleFunc("/admin/contributions/reject", controller.WithAudit(controller.WithOpenAPI(controller.HandleRejectContribution)))
	http.HandleFunc("/openclaw/skills/contributions/withdraw", controller.WithAudit(controller.WithOpenAPI(controller.HandleWithdrawContribution)))

	// ── 管理员技能下架/上架（直接生效，不走审核）──
	http.HandleFunc("/admin/skills/offline", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminSkillOffline)))
	http.HandleFunc("/admin/skills/online", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminSkillOnline)))

	http.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/config", http.StatusFound)
	})
	http.HandleFunc("/admin/users", controller.WithOpenAPI(controller.HandleAdmin))
	http.HandleFunc("/admin/departments", controller.WithOpenAPI(controller.HandleDepartments))
	http.HandleFunc("/admin/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			controller.WithAudit(controller.WithOpenAPI(controller.HandleUpdateConfig))(w, r)
		} else {
			controller.WithOpenAPI(controller.HandleAdminConfig)(w, r)
		}
	})
	http.HandleFunc("/admin/config/cvm", controller.WithAudit(controller.WithOpenAPI(controller.HandleUpdateCVMConfig)))
	http.HandleFunc("/admin/config/template", controller.WithAudit(controller.WithOpenAPI(controller.HandleUpdateTemplate)))
	http.HandleFunc("/admin/resource-policies", controller.WithOpenAPI(controller.HandleListResourcePolicies))
	http.HandleFunc("/admin/resource-policies/create", controller.WithAudit(controller.WithOpenAPI(controller.HandleCreateResourcePolicy)))
	http.HandleFunc("/admin/resource-policies/update", controller.WithAudit(controller.WithOpenAPI(controller.HandleUpdateResourcePolicy)))
	http.HandleFunc("/admin/resource-policies/delete", controller.WithAudit(controller.WithOpenAPI(controller.HandleDeleteResourcePolicy)))
	http.HandleFunc("/admin/resource-policies/options/instance-types", controller.WithOpenAPI(controller.HandleResourceOptionsInstanceTypes))
	http.HandleFunc("/admin/resource-policies/options/system-disks", controller.WithOpenAPI(controller.HandleResourceOptionsSystemDisks))
	http.HandleFunc("/admin/vpc/cloud", controller.WithOpenAPI(controller.HandleListCloudVpcs))
	http.HandleFunc("/admin/subnet/cloud", controller.WithOpenAPI(controller.HandleListCloudSubnets))
	http.HandleFunc("/admin/notices", controller.WithOpenAPI(controller.HandleAdminNotices))

	// Tag 标签管理
	http.HandleFunc("/api/tags/keys", controller.WithOpenAPI(controller.HandleGetTagKeys))
	http.HandleFunc("/api/tags/values", controller.WithOpenAPI(controller.HandleGetTagValues))
	http.HandleFunc("/admin/tags", controller.WithOpenAPI(controller.HandleAdminTags))
	http.HandleFunc("/admin/tags/create", controller.WithAudit(controller.WithOpenAPI(controller.HandleCreateTag)))
	http.HandleFunc("/admin/tags/update", controller.WithAudit(controller.WithOpenAPI(controller.HandleUpdateTag)))
	http.HandleFunc("/admin/tags/replace-all", controller.WithAudit(controller.WithOpenAPI(controller.HandleReplaceAllTags)))
	http.HandleFunc("/admin/tags/delete", controller.WithAudit(controller.WithOpenAPI(controller.HandleDeleteTag)))

	http.HandleFunc("/admin/config/smh", controller.WithAudit(controller.HandleUpdateSMHConfig))
	http.HandleFunc("/admin/smh/config", controller.HandleAdminSMHConfigAPI)
	http.HandleFunc("/admin/smh/personal-spaces", controller.WithOpenAPI(controller.HandleAdminPersonalSpaces))
	http.HandleFunc("/admin/smh/personal-spaces/token", controller.WithOpenAPI(controller.HandleAdminPersonalSpaceToken))
	http.HandleFunc("/admin/smh/instances", controller.WithOpenAPI(controller.HandleAdminSMHInstances))
	http.HandleFunc("/admin/smh/instance-space", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminInstancePersonalSpace)))
	http.HandleFunc("/admin/smh/personal-space-auto-provision", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminSMHAutoProvision)))
	http.HandleFunc("/admin/smh/stat", controller.WithOpenAPI(controller.HandleAdminSMHStat))
	http.HandleFunc("/admin/config/security-group", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			controller.WithOpenAPI(controller.HandleGetSecurityGroup)(w, r)
		case http.MethodPost:
			controller.WithAudit(controller.WithOpenAPI(controller.HandleCreateSecurityGroup))(w, r)
		case http.MethodPut:
			controller.WithAudit(controller.WithOpenAPI(controller.HandleUpdateSecurityGroup))(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/admin/config/security-group/list", controller.WithOpenAPI(controller.HandleListCloudSecurityGroups))
	http.HandleFunc("/admin/config/security-group/required-rules", controller.WithOpenAPI(controller.HandleGetRequiredSGRules))
	http.HandleFunc("/admin/config/security-group/check-rules", controller.WithOpenAPI(controller.HandleCheckSecurityGroupRules))
	http.HandleFunc("/admin/config/security-group/bind", controller.WithAudit(controller.WithOpenAPI(controller.HandleBindSecurityGroup)))
	http.HandleFunc("/admin/config/security-group/ruleset", controller.WithOpenAPI(controller.HandleGetRuleSet))
	http.HandleFunc("/admin/config/security-group/rulesets", controller.WithAudit(controller.WithOpenAPI(controller.HandleCreateRuleSet)))
	http.HandleFunc("/admin/config/security-group/ruleset/rules", controller.WithAudit(controller.WithOpenAPI(controller.HandleUpdateRuleSetRules)))
	http.HandleFunc("/admin/config/security-group/ruleset/import-from-sg", controller.WithAudit(controller.WithOpenAPI(controller.HandleImportRulesFromSG)))
	http.HandleFunc("/admin/config/security-group/ruleset/rules/reorder", controller.WithAudit(controller.WithOpenAPI(controller.HandleReorderRuleSetRules)))
	// Deprecated: POST/PUT/DELETE 已合并为 /ruleset/rules，保留兼容旧版前端
	http.HandleFunc("/admin/config/security-group/policies", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			controller.WithOpenAPI(controller.HandleDescribeSecurityGroupPolicies)(w, r)
		case http.MethodPost:
			controller.WithAudit(controller.WithOpenAPI(controller.HandleCreateSecurityGroupPolicies))(w, r)
		case http.MethodPut:
			controller.WithAudit(controller.WithOpenAPI(controller.HandleReplaceSecurityGroupPolicy))(w, r)
		case http.MethodDelete:
			controller.WithAudit(controller.WithOpenAPI(controller.HandleDeleteSecurityGroupPolicies))(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	// GET 预览任意云端 SG 的规则，供"从其他安全组导入规则"弹窗使用（透传腾讯云 DescribeSecurityGroupPolicies）
	http.HandleFunc("/admin/config/security-group/cloud-policies", controller.WithOpenAPI(controller.HandleDescribeCloudSecurityGroupPolicies))
	http.HandleFunc("/admin/create", controller.WithAudit(controller.WithOpenAPI(controller.HandleCreateUser)))
	http.HandleFunc("/admin/batch-create", controller.WithAudit(controller.WithOpenAPI(controller.HandleBatchCreateUser)))
	http.HandleFunc("/admin/delete", controller.WithAudit(controller.WithOpenAPI(controller.HandleDeleteUser)))
	http.HandleFunc("/admin/hard-delete", controller.WithAudit(controller.WithOpenAPI(controller.HandleHardDeleteUser)))
	http.HandleFunc("/admin/restore", controller.WithAudit(controller.WithOpenAPI(controller.HandleRestoreUser)))
	http.HandleFunc("/admin/user-limit", controller.WithOpenAPI(controller.HandleUserLimit))
	http.HandleFunc("/admin/user-vpc", controller.WithOpenAPI(controller.HandleUserVPC))
	http.HandleFunc("/admin/reset-password", controller.WithAudit(controller.WithOpenAPI(controller.HandleResetPassword)))
	http.HandleFunc("/admin/update-user", controller.WithAudit(controller.WithOpenAPI(controller.HandleUpdateUser)))
	http.HandleFunc("/admin/export-tokens", controller.WithAudit(controller.WithOpenAPI(controller.HandleExportTokens)))
	http.HandleFunc("/admin/user-token", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminUserToken)))
	http.HandleFunc("/admin/token/disable", controller.WithAudit(controller.WithOpenAPI(controller.HandleDisableToken)))
	http.HandleFunc("/admin/token/enable", controller.WithAudit(controller.WithOpenAPI(controller.HandleEnableToken)))
	http.HandleFunc("/admin/channels", controller.WithOpenAPI(controller.HandleAdminChannels))
	http.HandleFunc("/admin/channels/toggle", controller.WithAudit(controller.WithOpenAPI(controller.HandleToggleChannel)))
	http.HandleFunc("/admin/channels/add", controller.WithAudit(controller.WithOpenAPI(controller.HandleAddChannel)))
	http.HandleFunc("/admin/channels/delete", controller.WithAudit(controller.WithOpenAPI(controller.HandleDeleteChannel)))
	http.HandleFunc("/admin/channels/visibility", controller.WithAudit(controller.WithOpenAPI(controller.HandleChannelVisibility)))
	// Model management (AIModel table, replaces old LLMProvider routes)
	http.HandleFunc("/admin/models", controller.WithOpenAPI(controller.HandleAdminModels))
	http.HandleFunc("/admin/models/create", controller.WithAudit(controller.WithOpenAPI(controller.HandleCreateModel)))
	http.HandleFunc("/admin/models/update", controller.WithAudit(controller.WithOpenAPI(controller.HandleUpdateModel)))
	http.HandleFunc("/admin/models/delete", controller.WithAudit(controller.WithOpenAPI(controller.HandleDeleteModel)))
	http.HandleFunc("/admin/models/toggle", controller.WithAudit(controller.WithOpenAPI(controller.HandleToggleModel)))
	http.HandleFunc("/admin/models/toggle-enabled", controller.WithAudit(controller.WithOpenAPI(controller.HandleToggleModelEnabled)))
	http.HandleFunc("/admin/models/toggle-default", controller.WithAudit(controller.WithOpenAPI(controller.HandleToggleDefault)))
	http.HandleFunc("/admin/models/visibility", controller.WithAudit(controller.WithOpenAPI(controller.HandleUpdateModelVisibility)))
	http.HandleFunc("/admin/models/connectivity", controller.WithOpenAPI(controller.HandleAdminModelConnectivity))
	// Image management
	http.HandleFunc("/admin/images", controller.WithOpenAPI(controller.HandleAdminImages))
	http.HandleFunc("/admin/images/cloud", controller.WithOpenAPI(controller.HandleListCloudImages))
	http.HandleFunc("/admin/images/import", controller.WithAudit(controller.WithOpenAPI(controller.HandleImportImage)))
	http.HandleFunc("/admin/images/delete", controller.WithAudit(controller.WithOpenAPI(controller.HandleDeleteImage)))
	http.HandleFunc("/admin/images/enable", controller.WithAudit(controller.WithOpenAPI(controller.HandleEnableImage)))
	http.HandleFunc("/admin/images/update", controller.WithAudit(controller.WithOpenAPI(controller.HandleUpdateImage)))
	http.HandleFunc("/admin/images/history/publish", controller.WithAudit(controller.HandlePublishImageUpdate))
	http.HandleFunc("/admin/images/history/update", controller.WithAudit(controller.HandleUpdateImageHistory))
	http.HandleFunc("/admin/images/history/delete", controller.WithAudit(controller.HandleDeleteImageHistory))
	http.HandleFunc("/admin/images/history/restore", controller.WithAudit(controller.HandleRestoreImageHistory))
	http.HandleFunc("/admin/images/update-notice", controller.WithAudit(controller.WithOpenAPI(controller.HandleUpdateImageNotice)))
	http.HandleFunc("/admin/images/history", controller.WithOpenAPI(controller.HandleImageUpdateHistory))
	http.HandleFunc("/admin/images/type-visibility", controller.WithAudit(controller.WithOpenAPI(controller.HandleImageTypeVisibility)))
	// Agent type management
	http.HandleFunc("/admin/agent-types", controller.WithOpenAPI(controller.HandleAdminAgentTypes))
	http.HandleFunc("/admin/agent-types/enabled", controller.WithAudit(controller.WithOpenAPI(controller.HandleUpdateAgentTypeEnabled)))
	http.HandleFunc("/admin/agent-types/create", controller.WithAudit(controller.WithOpenAPI(controller.HandleCreateCustomAgentType)))
	http.HandleFunc("/admin/agent-types/delete", controller.WithAudit(controller.WithOpenAPI(controller.HandleDeleteCustomAgentType)))
	http.HandleFunc("/admin/images/set-default-type", controller.WithAudit(controller.WithOpenAPI(controller.HandleSetDefaultAgentType)))
	// Agent 命令执行（feature/agent_command_execution）
	http.HandleFunc("/admin/agent-commands", controller.WithOpenAPI(controller.HandleListAgentCommands))
	http.HandleFunc("/admin/agent-commands/create", controller.WithAudit(controller.WithOpenAPI(controller.HandleCreateAgentCommand)))
	http.HandleFunc("/admin/agent-commands/update", controller.WithAudit(controller.WithOpenAPI(controller.HandleUpdateAgentCommand)))
	http.HandleFunc("/admin/agent-commands/delete", controller.WithAudit(controller.WithOpenAPI(controller.HandleDeleteAgentCommand)))
	http.HandleFunc("/admin/agent-commands/dispatch", controller.WithAudit(controller.WithOpenAPI(controller.HandleDispatchAgentCommand)))
	http.HandleFunc("/admin/agent-commands/tasks", controller.WithOpenAPI(controller.HandleListAgentCommandTasks))
	http.HandleFunc("/admin/agent-commands/tasks/detail", controller.WithOpenAPI(controller.HandleAgentCommandTaskDetail))
	http.HandleFunc("/admin/agent-commands/agent-status", controller.WithOpenAPI(controller.HandleAgentCommandAgentStatus))
	// Agent 命令定时任务
	http.HandleFunc("/admin/agent-commands/schedules", controller.WithOpenAPI(controller.HandleListAgentCommandSchedules))
	http.HandleFunc("/admin/agent-commands/schedules/create", controller.WithAudit(controller.WithOpenAPI(controller.HandleCreateAgentCommandSchedule)))
	http.HandleFunc("/admin/agent-commands/schedules/delete", controller.WithAudit(controller.WithOpenAPI(controller.HandleDeleteAgentCommandSchedule)))
	http.HandleFunc("/admin/agent-commands/schedules/toggle", controller.WithAudit(controller.WithOpenAPI(controller.HandleToggleAgentCommandSchedule)))
	http.HandleFunc("/admin/agent-commands/schedules/records", controller.WithOpenAPI(controller.HandleListAgentCommandScheduleRecords))
	// Instance monitoring
	http.HandleFunc("/admin/instances", controller.WithOpenAPI(controller.HandleAdminInstances))
	http.HandleFunc("/admin/instances/adjust-config/validate", controller.WithOpenAPI(controller.HandleAdminInstanceAdjustmentValidate))
	http.HandleFunc("/admin/instances/adjust-config", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminInstanceAdjustment)))
	http.HandleFunc("/admin/instances/group-check", controller.WithOpenAPI(controller.HandleAdminInstancesGroupCheck))
	http.HandleFunc("/admin/instances/create", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminCreateInstance)))
	http.HandleFunc("/admin/instances/by-user-group", controller.WithOpenAPI(controller.HandleAdminInstancesByUserGroup))
	http.HandleFunc("/admin/instances/delete", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminDeleteInstance)))
	http.HandleFunc("/admin/instances/start", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminStartInstance)))
	http.HandleFunc("/admin/instances/stop", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminStopInstance)))
	http.HandleFunc("/admin/instances/reboot", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminRebootInstance)))
	http.HandleFunc("/admin/instances/restart-gateway", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminRestartGateway)))
	http.HandleFunc("/admin/instances/reset", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminResetInstance)))
	http.HandleFunc("/admin/instances/terminal-url", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminInstanceTerminal)))
	http.HandleFunc("/admin/instances/denied-actions", controller.WithOpenAPI(controller.HandleAdminInstanceDeniedActions))
	http.HandleFunc("/admin/instances/status", controller.WithOpenAPI(controller.HandleAdminInstanceStatus))
	http.HandleFunc("/admin/instances/channels", controller.WithOpenAPI(controller.HandleAdminInstanceChannels))
	http.HandleFunc("/admin/instances/skills", controller.WithOpenAPI(controller.HandleAdminInstanceSkills))
	http.HandleFunc("/admin/instances/rules", controller.WithOpenAPI(controller.HandleAdminInstanceRules))
	http.HandleFunc("/admin/instances/models", controller.WithOpenAPI(controller.HandleAdminInstanceModels))
	http.HandleFunc("/admin/instances/available-models", controller.WithOpenAPI(controller.HandleAdminAvailableModels))
	http.HandleFunc("/admin/instances/available-channels", controller.WithOpenAPI(controller.HandleAdminAvailableChannels))
	http.HandleFunc("/admin/instances/set-model", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminSetModel)))
	http.HandleFunc("/admin/instances/batch-set-model", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminBatchSetModel)))
	http.HandleFunc("/admin/instances/add-model", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminAddModel)))
	http.HandleFunc("/admin/instances/switch-primary-model", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminSwitchPrimaryModel)))
	http.HandleFunc("/admin/instances/del-model", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminDelModel)))
	http.HandleFunc("/admin/instances/proxy/prepare", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminProxyPrepare)))
	http.HandleFunc("/admin/instances/set-channel", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminSetChannel)))
	http.HandleFunc("/admin/instances/del-channel", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminDelChannel)))
	http.HandleFunc("/admin/instances/refresh-version", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminRefreshInstanceVersion)))
	http.HandleFunc("/admin/instances/batch-upgrade", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminBatchUpgrade)))
	http.HandleFunc("/admin/instances/detect-install", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminDetectInstall)))
	// CLS management
	http.HandleFunc("/admin/check-role", controller.HandleCheckClawProAgentRole)
	http.HandleFunc("/admin/cls/open", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminOpenClsService)))
	http.HandleFunc("/admin/cls/close", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminCloseClsService)))
	http.HandleFunc("/admin/cls/status", controller.WithOpenAPI(controller.HandleAdminClsStatus))
	http.HandleFunc("/admin/cls/scope", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			controller.WithOpenAPI(controller.HandleAdminGetCLSScope)(w, r)
		case http.MethodPost:
			controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminUpdateCLSScope))(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/admin/cls/update", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			controller.WithOpenAPI(controller.HandleAdminGetCLSUpdateStats)(w, r)
		case http.MethodPost:
			controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminUpdateCLSPlugin))(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/admin/instances/cam-role", controller.WithAudit(controller.WithOpenAPI(controller.HandleModifyInstancesCamRole)))
	// Memory TDAI management (legacy)
	http.HandleFunc("/admin/memory-tdai/config", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			controller.HandleAdminMemoryTDAIConfig(w, r)
		case http.MethodPut:
			controller.WithAudit(controller.HandleAdminUpdateMemoryTDAIConfig)(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	// Memory Pro management (管控端记忆管理)
	http.HandleFunc("/admin/memory/overview", controller.WithOpenAPI(controller.HandleAdminMemoryOverview))
	http.HandleFunc("/admin/memory/pro/activate", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminMemoryProActivate)))
	http.HandleFunc("/admin/memory/pro/release", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminMemoryProRelease)))
	http.HandleFunc("/admin/memory/plan/switch", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminMemoryPlanSwitch)))
	http.HandleFunc("/admin/memory/instances", controller.WithOpenAPI(controller.HandleAdminMemoryInstances))
	http.HandleFunc("/admin/memory/plugin-upgrade/candidates", controller.WithOpenAPI(controller.HandleAdminMemoryPluginUpgradeCandidates))
	http.HandleFunc("/admin/memory/plugin-upgrade/execute", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminMemoryPluginUpgradeExecute)))
	http.HandleFunc("/admin/memory/default-plan", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			controller.WithOpenAPI(controller.HandleAdminMemoryDefaultPlan)(w, r)
		case http.MethodPut:
			controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminMemoryDefaultPlan))(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/admin/memory/group-policies", controller.WithOpenAPI(controller.HandleAdminMemoryGroupPolicies))
	http.HandleFunc("/admin/memory/group-policy", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminMemoryGroupPolicy)))
	http.HandleFunc("/admin/memory/group-policy/delete", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminMemoryGroupPolicyDelete)))
	// Usage dashboard
	http.HandleFunc("/admin/usage/data", controller.WithOpenAPI(controller.HandleAdminUsageData))
	http.HandleFunc("/admin/usage/logs", controller.WithOpenAPI(controller.HandleAdminUsageLogs))
	// Audit log
	http.HandleFunc("/admin/audit", controller.WithOpenAPI(controller.HandleAdminAudit))
	// Cloud API proxy (通用腾讯云 API 透传)
	http.HandleFunc("/admin/cloud", controller.HandleCloudProxyActions)
	http.HandleFunc("/admin/cloud/query/", controller.HandleCloudProxyQuery)
	http.HandleFunc("/admin/cloud/mutate/", controller.WithCloudProxyAudit(controller.HandleCloudProxyMutate))
	// Skill category management
	http.HandleFunc("/admin/skill-categories", controller.WithOpenAPI(controller.HandleAdminSkillCategories))
	http.HandleFunc("/admin/skill-categories/create", controller.WithAudit(controller.WithOpenAPI(controller.HandleCreateSkillCategory)))
	http.HandleFunc("/admin/skill-categories/update", controller.WithAudit(controller.WithOpenAPI(controller.HandleUpdateSkillCategory)))
	http.HandleFunc("/admin/skill-categories/delete", controller.WithAudit(controller.WithOpenAPI(controller.HandleDeleteSkillCategory)))
	// Skill management — 灰度代理：SkillHub 命中走 SkillHub API，否则走本地 DB
	http.HandleFunc("/admin/skills",
		controller.WithSkillHubProxy(
			controller.WithOpenAPI(controller.HandleAdminSkills),
			controller.WithOpenAPI(controller.HandleAdminSkillsViaSkillHub),
		))
	http.HandleFunc("/admin/skills/create", controller.WithAudit(controller.WithOpenAPI(controller.HandleCreateSkill)))
	http.HandleFunc("/admin/skills/update", controller.WithAudit(controller.WithOpenAPI(controller.HandleUpdateSkill)))
	http.HandleFunc("/admin/skills/delete", controller.WithAudit(controller.WithOpenAPI(controller.HandleDeleteSkill)))
	http.HandleFunc("/admin/skills/references", controller.WithOpenAPI(controller.HandleSkillReferences))
	http.HandleFunc("/admin/skills/detail", controller.WithOpenAPI(controller.HandleAdminSkillDetail))
	http.HandleFunc("/admin/skills/files", controller.WithOpenAPI(controller.HandleAdminSkillFiles))
	http.HandleFunc("/admin/skills/tasks", controller.WithOpenAPI(controller.HandleAdminSkillTasks))
	http.HandleFunc("/admin/skills/instances", controller.WithOpenAPI(controller.HandleAdminSkillInstances))
	http.HandleFunc("/admin/skills/distribute", controller.WithAudit(controller.WithOpenAPI(controller.HandleDistributeSkill)))
	http.HandleFunc("/admin/skills/uninstall", controller.WithAudit(controller.WithOpenAPI(controller.HandleUninstallSkill)))
	http.HandleFunc("/admin/skills/download", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminSkillDownload)))
	// SkillHub status
	http.HandleFunc("/admin/skillhub-status", controller.WithOpenAPI(controller.HandleSkillHubStatus))
	// Enterprise rule library (本地 agent 二期) - CRUD
	http.HandleFunc("/admin/rules", controller.WithOpenAPI(controller.HandleAdminRules))
	http.HandleFunc("/admin/rules/detail", controller.WithOpenAPI(controller.HandleAdminRuleDetail))
	http.HandleFunc("/admin/rules/create", controller.WithAudit(controller.WithOpenAPI(controller.HandleCreateRule)))
	http.HandleFunc("/admin/rules/delete", controller.WithAudit(controller.WithOpenAPI(controller.HandleDeleteRule)))
	http.HandleFunc("/admin/rules/files", controller.WithOpenAPI(controller.HandleAdminRuleFiles))
	http.HandleFunc("/admin/rules/update", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminRuleUpdate)))
	http.HandleFunc("/admin/rules/tasks", controller.WithOpenAPI(controller.HandleAdminRuleTasks))
	http.HandleFunc("/admin/rules/instances", controller.WithOpenAPI(controller.HandleAdminRuleInstances))
	http.HandleFunc("/admin/rules/distribute", controller.WithAudit(controller.WithOpenAPI(controller.HandleDistributeRule)))
	http.HandleFunc("/admin/rules/uninstall", controller.WithAudit(controller.WithOpenAPI(controller.HandleUninstallRule)))
	// Skill security scan management
	http.HandleFunc("/admin/skills/scan-trigger", controller.WithAudit(controller.WithOpenAPI(controller.HandleSkillScanTrigger)))
	http.HandleFunc("/admin/skills/scan-config", controller.WithAudit(controller.WithOpenAPI(controller.HandleSkillScanConfigRouter)))
	http.HandleFunc("/admin/smhinfo", controller.HandleAdminSMHToken)
	// Skill bundle management
	http.HandleFunc("/admin/skill-bundles/create", controller.WithAudit(controller.WithOpenAPI(controller.HandleCreateSkillBundle)))
	http.HandleFunc("/admin/skill-bundles", controller.WithOpenAPI(controller.HandleAdminSkillBundles))
	http.HandleFunc("/admin/skill-bundles/delete", controller.WithAudit(controller.WithOpenAPI(controller.HandleDeleteSkillBundle)))
	http.HandleFunc("/admin/skill-bundles/toggle", controller.WithAudit(controller.WithOpenAPI(controller.HandleToggleSkillBundle)))
	http.HandleFunc("/admin/skill-bundles/detail", controller.WithOpenAPI(controller.HandleSkillBundleDetail))
	http.HandleFunc("/admin/skill-bundles/update-skills", controller.WithAudit(controller.WithOpenAPI(controller.HandleUpdateSkillBundleSkills)))
	http.HandleFunc("/admin/skill-bundles/batch-add-skills", controller.WithAudit(controller.WithOpenAPI(controller.HandleBatchAddSkillBundleSkills)))
	http.HandleFunc("/admin/skill-bundles/update-visibility", controller.WithAudit(controller.WithOpenAPI(controller.HandleUpdateSkillBundleVisibility)))
	http.HandleFunc("/admin/skills/favorite", controller.WithAudit(controller.WithOpenAPI(controller.HandleFavoriteSkill)))
	http.HandleFunc("/admin/skills/unfavorite", controller.WithAudit(controller.WithOpenAPI(controller.HandleUnfavoriteSkill)))
	http.HandleFunc("/admin/skills/favorited", controller.WithOpenAPI(controller.HandleAdminFavoritedSkills))
	http.HandleFunc("/admin/skillsets/favorite", controller.WithAudit(controller.WithOpenAPI(controller.HandleFavoriteSkillSet)))
	http.HandleFunc("/admin/skillsets/unfavorite", controller.WithAudit(controller.WithOpenAPI(controller.HandleUnfavoriteSkillSet)))
	http.HandleFunc("/admin/skillsets/favorited", controller.WithOpenAPI(controller.HandleAdminFavoritedSkillSets))
	// Plugin category management
	http.HandleFunc("/admin/plugin-categories", controller.WithOpenAPI(controller.HandleAdminPluginCategories))
	http.HandleFunc("/admin/plugin-categories/create", controller.WithAudit(controller.WithOpenAPI(controller.HandleCreatePluginCategory)))
	http.HandleFunc("/admin/plugin-categories/update", controller.WithAudit(controller.WithOpenAPI(controller.HandleUpdatePluginCategory)))
	http.HandleFunc("/admin/plugin-categories/delete", controller.WithAudit(controller.WithOpenAPI(controller.HandleDeletePluginCategory)))
	// Plugin management
	http.HandleFunc("/admin/plugins", controller.WithOpenAPI(controller.HandleAdminPlugins))
	http.HandleFunc("/admin/plugins/create", controller.WithAudit(controller.WithOpenAPI(controller.HandleCreatePlugin)))
	http.HandleFunc("/admin/plugins/update", controller.WithAudit(controller.WithOpenAPI(controller.HandleUpdatePlugin)))
	http.HandleFunc("/admin/plugins/delete", controller.WithAudit(controller.WithOpenAPI(controller.HandleDeletePlugin)))
	http.HandleFunc("/admin/plugins/detail", controller.WithOpenAPI(controller.HandleAdminPluginDetail))
	http.HandleFunc("/admin/plugins/files", controller.WithOpenAPI(controller.HandleAdminPluginFiles))
	http.HandleFunc("/admin/plugins/tasks", controller.WithOpenAPI(controller.HandleAdminPluginTasks))
	http.HandleFunc("/admin/plugins/instances", controller.WithOpenAPI(controller.HandleAdminPluginInstances))
	http.HandleFunc("/admin/plugins/distribute", controller.WithAudit(controller.WithOpenAPI(controller.HandleDistributePlugin)))
	http.HandleFunc("/admin/plugins/uninstall", controller.WithAudit(controller.WithOpenAPI(controller.HandleUninstallPlugin)))
	http.HandleFunc("/admin/plugins/favorite", controller.WithAudit(controller.WithOpenAPI(controller.HandleFavoritePlugin)))
	http.HandleFunc("/admin/plugins/unfavorite", controller.WithAudit(controller.WithOpenAPI(controller.HandleUnfavoritePlugin)))
	http.HandleFunc("/admin/plugins/favorited", controller.WithOpenAPI(controller.HandleAdminFavoritedPlugins))
	// Plugin bundle management
	http.HandleFunc("/admin/plugin-bundles", controller.WithOpenAPI(controller.HandleAdminPluginBundles))
	http.HandleFunc("/admin/plugin-bundles/create", controller.WithAudit(controller.WithOpenAPI(controller.HandleCreatePluginBundle)))
	http.HandleFunc("/admin/plugin-bundles/delete", controller.WithAudit(controller.WithOpenAPI(controller.HandleDeletePluginBundle)))
	http.HandleFunc("/admin/plugin-bundles/toggle", controller.WithAudit(controller.WithOpenAPI(controller.HandleTogglePluginBundle)))
	http.HandleFunc("/admin/plugin-bundles/detail", controller.WithOpenAPI(controller.HandlePluginBundleDetail))
	http.HandleFunc("/admin/plugin-bundles/update-plugins", controller.WithAudit(controller.WithOpenAPI(controller.HandleUpdatePluginBundlePlugins)))
	// MCP 企业库
	http.HandleFunc("/admin/mcp", controller.WithOpenAPI(controller.HandleAdminMcpList))
	http.HandleFunc("/admin/mcp/create", controller.WithAudit(controller.WithOpenAPI(controller.HandleCreateMcp)))
	http.HandleFunc("/admin/mcp/update", controller.WithAudit(controller.WithOpenAPI(controller.HandleCreateMcpVersion)))
	http.HandleFunc("/admin/mcp/meta", controller.WithAudit(controller.WithOpenAPI(controller.HandleUpdateMcpMeta)))
	http.HandleFunc("/admin/mcp/detail", controller.WithOpenAPI(controller.HandleAdminMcpDetail))
	http.HandleFunc("/admin/mcp/delete", controller.WithAudit(controller.WithOpenAPI(controller.HandleDeleteMcp)))
	http.HandleFunc("/admin/mcp/versions", controller.WithOpenAPI(controller.HandleAdminMcpVersionsRouter))
	http.HandleFunc("/admin/mcp/distribute", controller.WithAudit(controller.WithOpenAPI(controller.HandleDistributeMcp)))
	http.HandleFunc("/admin/mcp/visibility", controller.WithAudit(controller.WithOpenAPI(controller.HandleMCPVisibility)))
	http.HandleFunc("/admin/mcp/tasks", controller.WithOpenAPI(controller.HandleAdminMcpTasks))
	http.HandleFunc("/admin/mcp/instances", controller.WithOpenAPI(controller.HandleAdminMcpInstances))
	// User group management (用户组管理)
	http.HandleFunc("/admin/user-groups", controller.WithOpenAPI(controller.HandleAdminListUserGroups))
	http.HandleFunc("/admin/user-groups/create", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminCreateUserGroup)))
	http.HandleFunc("/admin/user-groups/update", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminUpdateUserGroup)))
	http.HandleFunc("/admin/user-groups/delete", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminDeleteUserGroup)))
	http.HandleFunc("/admin/user-groups/delete-impact", controller.WithOpenAPI(controller.HandleAdminGetGroupDeleteImpact))
	http.HandleFunc("/admin/user-groups/tree", controller.WithOpenAPI(controller.HandleAdminGetGroupTree))

	// 分组 VPC 配置
	http.HandleFunc("/admin/group-vpc-configs", controller.WithOpenAPI(controller.HandleListGroupVpcConfigs))
	http.HandleFunc("/admin/group-vpc-configs/create", controller.WithAudit(controller.WithOpenAPI(controller.HandleCreateGroupVpcConfig)))
	http.HandleFunc("/admin/group-vpc-configs/update", controller.WithAudit(controller.WithOpenAPI(controller.HandleUpdateGroupVpcConfig)))
	http.HandleFunc("/admin/group-vpc-configs/delete", controller.WithAudit(controller.WithOpenAPI(controller.HandleDeleteGroupVpcConfig)))
	http.HandleFunc("/admin/user-groups/members", controller.WithOpenAPI(controller.HandleAdminGetGroupMembers))
	http.HandleFunc("/admin/user-groups/members/set", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminSetGroupMembers)))
	http.HandleFunc("/admin/user-groups/members/add", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminAddGroupMembers)))
	http.HandleFunc("/admin/user-groups/members/remove", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminRemoveGroupMembers)))
	http.HandleFunc("/admin/user-groups/config-overview", controller.WithOpenAPI(controller.HandleAdminGetGroupConfigOverview))
	http.HandleFunc("/admin/user-groups/groups-by-users", controller.WithOpenAPI(controller.HandleAdminGetGroupsByUsers))
	http.HandleFunc("/admin/user-groups/instances", controller.WithOpenAPI(controller.HandleAdminUserGroupInstances))
	http.HandleFunc("/admin/user-groups/associated-models", controller.WithOpenAPI(controller.HandleGetGroupAssociatedModels))
	// 项目管理与统一资产管理（本期仅本地 Workspace 项目关系）
	http.HandleFunc("/admin/projects", controller.WithOpenAPI(controller.HandleAdminProjects))
	http.HandleFunc("/admin/projects/create", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminProjectCreate)))
	http.HandleFunc("/admin/projects/update", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminProjectUpdate)))
	http.HandleFunc("/admin/projects/delete-impact", controller.WithOpenAPI(controller.HandleAdminProjectDeleteImpact))
	http.HandleFunc("/admin/projects/delete", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminProjectDelete)))
	http.HandleFunc("/admin/projects/members", controller.WithOpenAPI(controller.HandleAdminProjectMembers))
	http.HandleFunc("/admin/projects/members/set", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminProjectMembersSet)))
	http.HandleFunc("/admin/projects/members/add", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminProjectMembersAdd)))
	http.HandleFunc("/admin/projects/members/remove", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminProjectMembersRemove)))
	http.HandleFunc("/admin/projects/projects-by-users", controller.WithOpenAPI(controller.HandleAdminProjectsByUsers))
	http.HandleFunc("/admin/projects/config-overview", controller.WithOpenAPI(controller.HandleAdminProjectConfigOverview))
	http.HandleFunc("/admin/projects/instances", controller.WithOpenAPI(controller.HandleAdminProjectInstances))
	http.HandleFunc("/admin/assets/detail", controller.WithOpenAPI(controller.HandleAdminAssetDetail))
	http.HandleFunc("/admin/assets/candidates", controller.WithOpenAPI(controller.HandleAdminAssetCandidates))
	http.HandleFunc("/admin/assets/save", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminAssetSave)))
	http.HandleFunc("/admin/assets/versions", controller.WithOpenAPI(controller.HandleAdminAssetVersions))
	http.HandleFunc("/admin/users/multi-group-stats", controller.WithOpenAPI(controller.HandleAdminMultiGroupStats))
	// 分组配置绑定
	http.HandleFunc("/admin/group-config/groups", controller.WithOpenAPI(controller.HandleGroupConfigGroups))
	http.HandleFunc("/admin/group-config/policy", controller.WithAudit(controller.WithOpenAPI(controller.HandleSetGroupPolicy)))
	http.HandleFunc("/admin/group-config/policy/delete", controller.WithAudit(controller.WithOpenAPI(controller.HandleDeleteGroupPolicy)))

	// 存量实例分组归属处理（stale-instances v1.0）— 管理端
	http.HandleFunc("/admin/stale-instances/config-diff", controller.WithOpenAPI(controller.HandleAdminStaleInstancesConfigDiff))
	http.HandleFunc("/admin/stale-instances/apply", controller.WithAudit(controller.WithOpenAPI(controller.HandleAdminStaleInstancesApply)))
	http.HandleFunc("/admin/stale-instances/records", controller.WithOpenAPI(controller.HandleAdminStaleInstancesRecords))
	http.HandleFunc("/admin/stale-instances/action-options", controller.WithOpenAPI(controller.HandleAdminStaleInstancesActionOptions))
	// 存量实例分组归属处理（stale-instances v1.0）— 用户端
	http.HandleFunc("/openclaw/stale-instances/rebind", controller.WithAudit(controller.WithOpenAPI(controller.HandleUserStaleInstancesRebind)))
	http.HandleFunc("/openclaw/stale-instances/initiate", controller.WithAudit(controller.WithOpenAPI(controller.HandleUserStaleInstancesHandoverInitiate)))
	http.HandleFunc("/openclaw/stale-instances/cancel", controller.WithAudit(controller.WithOpenAPI(controller.HandleUserStaleInstancesHandoverCancel)))
	http.HandleFunc("/openclaw/stale-instances/accept", controller.WithAudit(controller.WithOpenAPI(controller.HandleUserStaleInstancesHandoverAccept)))
	http.HandleFunc("/openclaw/stale-instances/reject", controller.WithAudit(controller.WithOpenAPI(controller.HandleUserStaleInstancesHandoverReject)))
	// User group query for regular users (普通用户查询自己所在的用户组)
	http.HandleFunc("/user-groups/mine", controller.WithOpenAPI(controller.HandleGetMyUserGroups))
	// Role management (角色设定)
	http.HandleFunc("/admin/roles", controller.WithOpenAPI(controller.HandleAdminRoles))
	http.HandleFunc("/admin/roles/create", controller.WithAudit(controller.WithOpenAPI(controller.HandleCreateRole)))
	http.HandleFunc("/admin/roles/update", controller.WithAudit(controller.WithOpenAPI(controller.HandleUpdateRole)))
	http.HandleFunc("/admin/roles/delete", controller.WithAudit(controller.WithOpenAPI(controller.HandleDeleteRole)))
	http.HandleFunc("/admin/roles/toggle-visible", controller.WithAudit(controller.WithOpenAPI(controller.HandleToggleRoleVisible)))
	http.HandleFunc("/admin/roles/reorder", controller.WithAudit(controller.WithOpenAPI(controller.HandleReorderRoles)))
	http.HandleFunc("/admin/roles/detail", controller.WithOpenAPI(controller.HandleRoleDetail))
	http.HandleFunc("/admin/roles/distribute", controller.WithAudit(controller.WithOpenAPI(controller.HandleDistributeRole)))
	http.HandleFunc("/admin/roles/instances", controller.WithOpenAPI(controller.HandleAdminRoleInstances))
	http.HandleFunc("/admin/roles/records", controller.WithOpenAPI(controller.HandleAdminRoleRecords))
	// Employee-facing role APIs
	http.HandleFunc("/openclaw/roles", controller.WithOpenAPI(controller.HandleOpenClawRoles))
	http.HandleFunc("/openclaw/remove-role", controller.WithAudit(controller.WithOpenAPI(controller.HandleRemoveInstanceRole)))
	http.HandleFunc("/openclaw/switch-role", controller.WithAudit(controller.WithOpenAPI(controller.HandleSwitchRole)))

	// Memory plan management (记忆计划切换)
	// http.HandleFunc("/openclaw/memory/plan/switch", controller.WithAudit(controller.WithOpenAPI(controller.HandleMemoryPlanSwitch)))
	http.HandleFunc("/openclaw/memory/config", controller.WithOpenAPI(controller.HandleMemoryConfig))
	http.HandleFunc("/openclaw/memory/task", controller.WithOpenAPI(controller.HandleMemoryTaskDetail))
	http.HandleFunc("/openclaw/memory/library/detail", controller.WithOpenAPI(controller.HandleMemoryLibraryDetail))

	// ====== 多租户管理 API（AdminToken 鉴权，两种模式均可用）======
	http.HandleFunc("/tenants/init", controller.WithAudit(controller.HandleInitTenant))
	http.HandleFunc("/tenants/domains", controller.WithAudit(controller.HandleTenantDomains))
	http.HandleFunc("/tenants/", controller.HandleListTenantDomains)

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	http.HandleFunc("/site", controller.WithOpenAPI(controller.HandleSite))
	http.HandleFunc("/favicon.ico", controller.HandleLogo)
	http.HandleFunc("/logo", controller.HandleLogo)

	// User API Token self-management
	http.HandleFunc("/api-token", controller.HandleGetAPIToken)
	http.HandleFunc("/api-token/create", controller.WithAudit(controller.HandleCreateAPIToken))
	http.HandleFunc("/api-token/reset", controller.WithAudit(controller.HandleResetAPIToken))
	http.HandleFunc("/api-token/revoke", controller.WithAudit(controller.HandleRevokeAPIToken))

	// OneID SSO (Gateway 模式)
	http.HandleFunc("/auth/oneid", controller.HandleOneIDLogin)
	http.HandleFunc("/auth/sso-providers", controller.HandleSsoProviders)
	http.HandleFunc("/auth/internal-login", controller.WithAudit(controller.HandleInternalLogin))
	http.HandleFunc("/auth/oneid-code", controller.HandleOneIDCode)
	http.HandleFunc("/auth/oneid-register", controller.WithAudit(controller.HandleOneIDRegister))
	http.HandleFunc("/auth/oneid/jump", controller.HandleOneIDJump)
	http.HandleFunc("/spi/logout", controller.WithAudit(controller.HandleOneIDLogout))
	http.HandleFunc("/spi/event", controller.WithAudit(controller.HandleOneIDEvent))
	http.HandleFunc("/admin/oneid-sync-enterprise", controller.WithAudit(controller.HandleSyncEnterprise))
	http.HandleFunc("/admin/oneid-sync-users", controller.WithAudit(controller.HandleSyncOneIDUsers))
	http.HandleFunc("/admin/oneid-sync-users/status", controller.HandleSyncOneIDUsersStatus)

	// 独立账号 → 统一账号迁移工具（旁路，不改动现有 handler）
	http.HandleFunc("/admin/migrate", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			controller.HandleMigrateInit(w, r)
		case http.MethodGet:
			controller.HandleMigrateStatus(w, r)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/admin/migrate/run", controller.HandleMigrateRun)
	http.HandleFunc("/admin/migrate/finalize", controller.HandleMigrateFinalize)

	// OneID 登录透明代理（统一账号模式）
	http.HandleFunc("/oneid/encrypt_setting", controller.HandleOneIDAuthnProxy)
	http.HandleFunc("/oneid/login-name", controller.HandleOneIDLoginName)
	http.HandleFunc("/oneid/enterprise", controller.WithAudit(controller.HandleOneIDAuthnLogin))
	http.HandleFunc("/oneid/password-reset", controller.WithAudit(controller.HandleOneIDPasswordReset))
	http.HandleFunc("/oneid/password-verify", controller.HandleOneIDAuthnProxy)
	http.HandleFunc("/oneid/password-change", controller.WithAudit(controller.HandleOneIDPasswordReset))

	// User quota page
	http.HandleFunc("/quota/data", controller.WithOpenAPI(controller.HandleQuotaData))
	http.HandleFunc("/quota/logs", controller.WithOpenAPI(controller.HandleQuotaLogs))

	// Agent reverse proxy (tenant from Host, route secret from URL)
	http.HandleFunc("/proxy/", controller.HandleAgentProxy)

	// LLM Proxy (Bearer token auth, no session)
	http.HandleFunc("/v1/chat/completions", controller.HandleLLMProxy)
	http.HandleFunc("/v1/models", controller.HandleLLMModels)

	controller.InitUsageLogger()

	// 启动统一后台任务调度器（所有任务通过 init() + RegisterTask 注册）
	task.StartScheduler(0) // 0 = 使用默认 worker 数量

	logHandler := logMiddleware(forceJSONMiddleware(sanitizeOpenAPIHeader(http.DefaultServeMux)))
	logHandler = controller.I18nMiddleware(logHandler)
	// identifierMiddleware 须在 logMiddleware 之前生效：
	//   1. 确保 HTTP handler 的 r.Context() 中始终有 TenantSnapshot，GORM 回调据此注入 identifier
	//   2. 非 universe 模式：注入 FixedSnapshot
	//   3. Universe 模式：从 Host 动态查 tenant_domains 路由
	// identifierMiddleware 必须在 I18nMiddleware 之前生效
	// 以确保 I18nMiddleware 可以从 TenantSnapshot 中正确获取到用户语言设置
	logHandler = controller.IdentifierMiddleware(logHandler)

	srv := &http.Server{
		Addr:    *addr,
		Handler: pprofAuthMiddleware(logHandler),
	}

	// Graceful shutdown on SIGINT / SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-quit
		slog.Info("Received signal, shutting down...", "signal", sig)

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("HTTP server shutdown error", "error", err)
		}
		// 停止后台任务调度器
		task.StopScheduler()
	}()

	slog.Info("Server started", "addr", *addr)

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}

	// Server has stopped accepting new connections; clean up resources.
	slog.Info("Server stopped, cleaning up...")
	controller.FlushUsageLogs()
	model.CloseDB()
	slog.Info("Shutdown complete")
}

// pprofAuthMiddleware 包装 pprof 路由的认证检查。
// pprof 调试接口需要 admin token 认证，用于排查 goroutine/内存/CPU 问题。
// 使用方式: curl -H "Authorization: Bearer <admin-token>" http://localhost:8080/debug/pprof/goroutine?debug=1
// Go 1.22+ 的 net/http/pprof init() 已自动注册路由到 DefaultServeMux（使用 method pattern），
// 此处通过顶层 handler 包装 auth 检查，而非重复注册路由（会与 init() 的注册冲突）。
func pprofAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/debug/pprof") {
			if controller.AdminToken == "" {
				http.Error(w, "pprof disabled (no admin-token configured)", http.StatusForbidden)
				return
			}
			auth := r.Header.Get("Authorization")
			if auth != "Bearer "+controller.AdminToken {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// 0. 跳过高频/无意义路径，避免日志噪音
		switch {
		case r.URL.Path == "/health":
			next.ServeHTTP(w, r)
			return
		case r.URL.Path == "/favicon.ico" || r.URL.Path == "/logo":
			next.ServeHTTP(w, r)
			return
		case strings.HasPrefix(r.URL.Path, "/static/"):
			next.ServeHTTP(w, r)
			return
		case strings.HasPrefix(r.URL.Path, "/captcha/"):
			next.ServeHTTP(w, r)
			return
		}

		// 1. 注入 request_id / trace_id / interface / uin 到 context
		r = controller.NewRequestContext(r)

		// 2. 尝试解析用户，注入 subuin（非强制，失败不影响请求）
		if user, err := controller.RequestUser(r); user != nil && err == nil {
			r = controller.InjectSubUin(r, user.ID)
		}

		ctx := r.Context()

		// 3. 读取请求 body（需要回填，供后续 handler 继续读取）
		// 限制日志记录的 body 大小为 64KB，防止文件上传等大 body 请求导致内存压力
		const maxLogBody = 64 * 1024
		var reqBody []byte
		if r.Body != nil && r.Body != http.NoBody {
			reqBody, _ = io.ReadAll(io.LimitReader(r.Body, maxLogBody))
			// 将已读取的部分和剩余部分拼接回去，供后续 handler 继续读取
			r.Body = io.NopCloser(io.MultiReader(bytes.NewReader(reqBody), r.Body))
		}

		// 4. 记录 Rcv request
		controller.LogRcvRequest(ctx, r, reqBody)

		// 5. 包装 ResponseWriter 捕获响应
		rec := controller.NewResponseCapture(w)

		// 6. panic recover → Uncaught exception
		defer func() {
			if rv := recover(); rv != nil {
				stack := string(debug.Stack())
				msg := ""
				switch v := rv.(type) {
				case error:
					msg = v.Error()
				default:
					msg = "unknown panic"
				}
				controller.LogUncaughtException(ctx, r, "panic", 500, msg, stack)
				http.Error(rec, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(rec, r)

		cost := time.Since(start)
		// 流式响应（SSE）的 body 是原始 SSE 帧，记录无意义；替换为 [streaming] 占位符。
		// 实际的 token 用量等信息由 HandleLLMProxy 在流式完成后通过 LogLLMStream 单独记录。
		respBody := rec.Body
		if rec.IsStreaming {
			respBody = []byte("[streaming]")
		}
		controller.LogSendResponse(ctx, r, rec.StatusCode, rec.Header(), respBody, cost)
	})
}

func forceJSONMiddleware(next http.Handler) http.Handler {
	if !controller.DisableUI {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Set("Accept", "application/json")
		next.ServeHTTP(w, r)
	})
}

// sanitizeOpenAPIHeader 清除外部请求中可能伪造的 X-Hatchery-OpenAPI header。
// 该 header 仅由内部 WithOpenAPI 装饰器注入，外部不可信。
func sanitizeOpenAPIHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Del("X-Hatchery-OpenAPI")
		next.ServeHTTP(w, r)
	})
}
