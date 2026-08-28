package i18n

// SSO IM
var (
	MsgSSOIMNamePrivateWeCom = Key{string: "私有化企微"}
	MsgSSOIMNameMSEntraID    = Key{string: "微软 Entra ID"}
)

var (
	MsgGeneralAssistant = Key{string: "通用助手"}
)

// 认证相关
var (
	MsgInvalidCaptcha                      = Key{string: "验证码错误"}
	MsgInvalidCredentials                  = Key{string: "用户名或密码错误"}
	MsgAccountBanned                       = Key{string: "账户已被封禁"}
	MsgIncompleteForm                      = Key{string: "请填写完整"}
	MsgUserNotFound                        = Key{string: "用户 %s 不存在"}
	MsgOldPasswordWrong                    = Key{string: "原密码错误"}
	MsgUnauthorized                        = Key{string: "未登录"}
	MsgForbidden                           = Key{string: "无权限访问"}
	MsgAdminRequired                       = Key{string: "需要管理员权限"}
	MsgPasswordlessLoginFeatureUnavailable = Key{string: "免登录功能未对当前租户开放"}
	MsgPasswordlessLoginTokenInvalid       = Key{string: "免登录凭证无效或已过期"}
)

// 通用
var (
	MsgMethodNotAllowed                   = Key{string: "请求方法不允许"}
	MsgInvalidID                          = Key{string: "无效的 id"}
	MsgInvalidJSON                        = Key{string: "无效的 JSON 格式"}
	MsgContentTypeApplicationJSONRequired = Key{string: "Content-Type 必须为 application/json"}
	MsgBadRequest                         = Key{string: "请求体格式错误"}
	MsgBadRequestWithError                = Key{string: "请求体格式错误: %v"}
	MsgReadRequestBodyFailedWithError     = Key{string: "读取请求体失败: %v"}
	MsgReadRequestBody                    = Key{string: "读取请求体失败"}
	MsgRequestBodyCannotBeEmpty           = Key{string: "请求体不能为空"}
	MsgRequestBodyTooLargeWithError       = Key{string: "请求体过大或格式错误: %v"}
	MsgBadRequestMissingParamWithKey      = Key{string: "缺少参数: %s"}
	MsgBadRequestParamRequired            = Key{string: "参数 %s 不能为空"}
	MsgBadRequestParamInvalid             = Key{string: "参数 %s 无效"}
	MsgBadRequestParamInvalidWithDetail   = Key{string: "参数 %s 无效: %s"}
	MsgBadRequestParamFormatError         = Key{string: "参数 %s 格式错误"}
	MsgDistributionTargetModeConflict     = Key{string: "instance_ids 与 select_all=true 必须二选一"}
	MsgDistributionSelectAllRequired      = Key{string: "statuses 和 group_ids 仅可在 select_all=true 时使用"}
	MsgDistributionTransitionalStatus     = Key{string: "不能按过渡状态 %s 全选下发"}
	MsgDistributionStatusInvalid          = Key{string: "不支持的安装状态: %s"}
	MsgInvalidPhoneFormat                 = Key{string: "手机号格式无效：7-15 位数字（1-9 开头），不含 + 号，如 85266803489"}
	MsgParamMustBeBool                    = Key{string: "参数 %s 必须为 bool 类型"}
	MsgNotFound                           = Key{string: "资源不存在"}
	MsgOperationFailed                    = Key{string: "操作失败"}
	MsgFailedToMarshalJSON                = Key{string: "序列化 JSON 失败"}
	MsgUnknownError                       = Key{string: "未知错误"}
)

var (
	MsgNetworkUnreachable     = Key{string: "网络不通或上游地址无法访问"}
	MsgUpstreamDenied         = Key{string: "上游服务拒绝访问"}
	MsgUpstreamError          = Key{string: "上游服务异常"}
	MsgConnectivityCheckError = Key{string: "连通性检测失败"}
)

// OneID / SSO / Gateway 相关 (auth_oneid.go, auth_oneid_jump.go)
var (
	MsgGatewayNotConfigured      = Key{string: "Gateway 未配置，请联系管理员"}
	MsgOneIDTenantNotConfigured  = Key{string: "OneID 租户未配置（缺少 ONEID_ACCOUNT_ID 环境变量）"}
	MsgUserNotLinkedToOneID      = Key{string: "用户未关联 OneID"}
	MsgInternalError             = Key{string: "内部错误"}
	MsgFetchSSOLinkFailed        = Key{string: "获取免登链接失败: HTTP %d"}
	MsgOneIdRequireSessionToken  = Key{string: "OneID 登录响应缺少 session_token"}
	MsgOneIdObtainUserInfoFailed = Key{string: "获取 OneID 用户信息失败"}
	MsgOneIdUnionIDEmpty         = Key{string: "OneID 返回的用户标识为空"}
	MsgOneIdUserNotFound         = Key{string: "用户在本地系统中不存在，请联系管理员同步用户"}
)

// 实例 / 资源相关 (跨多个 controller 高频复用)
var (
	MsgInstanceNotFound         = Key{string: "实例不存在"}
	MsgInstanceNoCVM            = Key{string: "该实例无关联的 CVM"}
	MsgInstanceNotFoundOrNoPerm = Key{string: "实例不存在或无权访问"}
	MsgQueryInstanceFailed      = Key{string: "查询实例失败"}
)

// 业务相关
var (
	MsgQuotaExceeded = Key{string: "已使用 %d 次，配额上限 %d 次"}
)

// API Token 相关 (auth.go)
var (
	MsgInvalidAPIToken         = Key{string: "无效的 API Token"}
	MsgUserNotExist            = Key{string: "用户不存在"}
	MsgAPITokenDisabledByAdmin = Key{string: "您的 Token 已被管理员禁用，如需恢复请联系企业管理员"}
	MsgAPITokenAlreadyExists   = Key{string: "Token 已存在，如需更换请使用重置功能"}
	MsgAPITokenGenerateFailed  = Key{string: "生成 Token 失败"}
	MsgAPITokenResetFailed     = Key{string: "重置 Token 失败"}
	MsgAPITokenRevokeFailed    = Key{string: "销毁 Token 失败"}
)

var (
	MsgSecurityPolicyInvalid = Key{string: "安全策略 `%s` 不存在"}
)

// LLM Proxy 相关 (llm_proxy.go)
var (
	MsgInternalServerError        = Key{string: "服务器内部错误"}
	MsgMissingAPIKey              = Key{string: "缺少或无效的 API 密钥"}
	MsgInvalidAPIKey              = Key{string: "无效的 API 密钥"}
	MsgNoActiveModel              = Key{string: "该实例未配置可用模型"}
	MsgModelNotFound              = Key{string: "请求的模型未绑定到当前实例"}
	MsgGlobalQuotaExceeded        = Key{string: "全局每日令牌配额已用尽"}
	MsgGlobalMonthlyQuotaExceeded = Key{string: "全局每月令牌配额已用尽"}
	MsgModelQuotaExceeded         = Key{string: "模型每日令牌配额已用尽"}
	MsgUserQuotaExceeded          = Key{string: "用户每日令牌配额已用尽"}
	MsgLoadUserFailed             = Key{string: "加载用户信息失败"}
	MsgStreamNotSupported         = Key{string: "不支持流式响应"}
	MsgLLMBackendConnect          = Key{string: "连接 LLM 后端失败"}
)

// LLM Quota 相关 (llm_quota.go)
var (
	MsgInvalidOrderBy         = Key{string: "order_by 参数无效，仅支持 total_tokens 或 request_count"}
	MsgQueryUsageDataFailed   = Key{string: "查询用量数据失败"}
	MsgQueryRecordCountFailed = Key{string: "查询记录总数失败"}
	MsgQueryUsageLogsFailed   = Key{string: "查询使用记录失败"}
)

// 个人空间 / SMH 相关
var (
	MsgInstanceNoSMH     = Key{string: "该实例未绑定个人空间"}
	MsgGetSMHTokenFailed = Key{string: "获取访问 Token 失败"}
)

// 角色 / Soul 相关
var (
	MsgClearSoulSetAtFailed  = Key{string: "清除 soul_set_at 失败"}
	MsgSoulRemoveFailed      = Key{string: "Soul 移除失败，实例可能处于其他操作中或未运行，请重试"}
	MsgSoulSetAtUpdateFailed = Key{string: "更新 soul_set_at 失败"}
)

// 配置概览 (config-overview)
var (
	MsgInvalidKey                    = Key{string: "无效的 key: %s"}
	MsgIDsOrGroupIDsRequired         = Key{string: "ids 或 group_ids 参数必须传其一"}
	MsgInvalidIDsFormat              = Key{string: "ids 格式错误"}
	MsgParseGroupAncestorChainFailed = Key{string: "解析分组祖先链失败"}
	MsgNoInstancesForUser            = Key{string: "未找到属于当前用户的实例"}
)

// 环境变量 (env)
var (
	MsgEnvRequired               = Key{string: "env 不能为空"}
	MsgEnvCountLimit             = Key{string: "env 数量不能超过 %d"}
	MsgInvalidEnvName            = Key{string: "无效的环境变量名: %s（仅允许字母、数字、下划线，且不能以数字开头）"}
	MsgInvalidEnvValue           = Key{string: "环境变量 %s 的值必须是字符串或 null"}
	MsgMarshalParamsFailed       = Key{string: "序列化参数失败"}
	MsgAgentTypeNotSupportSetEnv = Key{string: "该 agent_type (%s) 不支持 set_env"}
	MsgAgentTypeNotSupportGetEnv = Key{string: "该 agent_type (%s) 不支持 get_env"}
	MsgSetEnvFailed              = Key{string: "设置环境变量失败"}
	MsgGetEnvFailed              = Key{string: "查询环境变量失败"}
	MsgParseEnvFailed            = Key{string: "解析环境变量失败"}
)

// WebSocket 接入地址 (ws-url)
var (
	MsgInvalidInstanceIDFormat   = Key{string: "instance_id 格式无效，应为 ins-xxxxxxxx"}
	MsgInstanceNotRunningForWS   = Key{string: "实例当前状态为 %s，仅 RUNNING 状态可获取连接地址"}
	MsgGatewayUIPortNotAllocated = Key{string: "Gateway UI 端口未分配，请先在管理后台开启 Gateway UI"}
	MsgAgentTypeNotSupportWS     = Key{string: "当前实例类型（%s）暂不支持获取连接地址"}
	MsgCreateCVMClientFailed     = Key{string: "创建 CVM 客户端失败"}
	MsgCVMInstanceNotFound       = Key{string: "未找到 CVM 实例"}
	MsgInstanceNoPrivateIP       = Key{string: "实例无可用内网 IP"}
	MsgInstanceNoUsableIP        = Key{string: "实例无可用 IP"}
	MsgInstanceNoSecurityGroup   = Key{string: "实例未绑定任何安全组，无法确认端口 %d 是否可访问"}
	MsgSGPortNotOpen             = Key{string: "安全组未放通端口 %d 的内网入站规则，请联系管理员在安全组中添加对应规则"}
	MsgGetWSConnInfoFailed       = Key{string: "获取连接信息失败"}
	MsgParseWSConnInfoFailed     = Key{string: "解析连接信息失败"}
	MsgInstanceReturnedError     = Key{string: "实例返回错误: %s"}
	MsgGatewayConfigIncomplete   = Key{string: "实例 Gateway 配置不完整（port 或 authToken 为空）"}
	MsgHermesConfigIncomplete    = Key{string: "Hermes API Server 配置不完整（port 或 key 为空）"}
)

// 实例重装 (reinstall)
var (
	MsgDoctorNodeNotAllowed          = Key{string: "龙虾医生节点不允许该操作"}
	MsgInstancePendingUserAction     = Key{string: "实例待用户处理分组归属，请先在弹窗完成迁移或移交后再开机"}
	MsgInstanceStaleGroup            = Key{string: "实例分组归属异常（stale_group），请联系管理员在管控端处理后再开机"}
	MsgAgentNotAllowed               = Key{string: "Agent 状态不允许该操作"}
	MsgAgentStatusStopped            = Key{string: "实例已关机，请先开机并等待实例恢复运行中后再操作"}
	MsgAgentStatusTransition         = Key{string: "实例当前为%s，请等待实例恢复运行中后再操作"}
	MsgAgentStatusFailed             = Key{string: "实例当前为%s，无法执行该操作"}
	MsgResetVersionFailed            = Key{string: "重置版本信息失败"}
	MsgReinstallInstanceFailed       = Key{string: "重装实例失败"}
	MsgReinstallInstanceFailedNotify = Key{string: "实例重装失败"}
	MsgReinstallNotSupported         = Key{string: "当前实例类型不支持重装"}
	MsgReinstallImageMismatch        = Key{string: "重装镜像与实例类型不匹配"}
	MsgModelNotSupported             = Key{string: "当前实例类型不支持该模型"}
	MsgChannelNotSupported           = Key{string: "当前实例类型不支持该 Channel"}
	MsgChannelNotSupportedWithDetail = Key{string: "当前实例类型 `%s` 不支持该 Channel"}
	MsgQueryImageFailed              = Key{string: "查询镜像失败"}
	MsgNoImageForType                = Key{string: "管理员尚未为 %s 类型配置生效镜像，无法重装实例"}
)

// Sentinel 错误（包级 var ErrXxx 的国际化承载，writeError 自动按请求语言翻译）
var (
	MsgOperationInProgress = Key{string: "操作进行中，请稍后再试"}
	MsgOperationConflict   = Key{string: "操作冲突，请刷新后重试"}
	MsgScanInProgress      = Key{string: "该技能已有正在进行的安全检测，请等待完成"}
	MsgFileTooLargeForScan = Key{string: "技能文件超过安全检测大小限制（7MB），无法进行安全检测"}
	MsgSGBootstrapNotDone  = Key{string: "ClawPro 安全组尚未初始化，请联系管理员完成初始化后再使用本功能"}
)

// 技能 / 插件 (skill / plugin)
var (
	MsgInvalidPluginName             = Key{string: "插件名称格式不合法"}
	MsgParseListSkillsScriptFailed   = Key{string: "解析 list_skills 脚本失败 (agent_type=%s)"}
	MsgParseAddSkillScriptFailed     = Key{string: "解析 add_skill 脚本失败"}
	MsgQuerySkillInstallationFailed  = Key{string: "查询技能安装记录失败"}
	MsgSkillNotFoundCheckName        = Key{string: "Skill 不存在，请检查 Skill 名称"}
	MsgRateLimitExceeded             = Key{string: "触发接口限频，请稍后重试"}
	MsgSkillInstallSuccess           = Key{string: "技能安装成功"}
	MsgSkillInstallDispatched        = Key{string: "已下发，等待客户端拉取后安装"}
	MsgSkillInstallFailedTitle       = Key{string: "技能安装失败"}
	MsgSkillNamedInstallFailed       = Key{string: "技能「%s」安装失败"}
	MsgWaitCVMTimeout                = Key{string: "等待 CVM 就绪超时，技能安装未完成"}
	MsgUnsupportedAgentType          = Key{string: "不支持的 agent_type: %s"}
	MsgSkillScriptExecuteFailed      = Key{string: "技能安装脚本执行失败"}
	MsgSkillScriptOutputAbnormal     = Key{string: "技能安装脚本输出异常，未找到安装结果"}
	MsgParseSkillInstallResultFailed = Key{string: "解析技能安装结果失败"}
	MsgPartialSkillsInstallFailed    = Key{string: "部分技能安装失败"}
	MsgSkillsBatchFailed             = Key{string: "%d 个技能安装失败：%s"}
	MsgSkillSMHSyncPending           = Key{string: "技能包 %s 尚未完成 SMH 同步，请稍后重试"}
	MsgPluginInstallSuccess          = Key{string: "插件安装成功"}
)

// Gateway UI / 网关配置 (gateway)
var (
	MsgGatewayUINotEnabled         = Key{string: "Gateway UI 功能未开启，请先在管理后台开启"}
	MsgGatewayUIPortNotConfigured  = Key{string: "Gateway UI 端口未分配，请先在管理后台配置"}
	MsgAgentTypeNotSupportWebUI    = Key{string: "当前实例类型（%s）暂不支持 WebUI"}
	MsgParseScriptOutputFailed     = Key{string: "解析脚本输出失败"}
	MsgGatewayRestarted            = Key{string: "gateway 已重启"}
	MsgScriptReturnIncomplete      = Key{string: "脚本返回数据不完整"}
	MsgInstanceNoSecurityGroupBind = Key{string: "实例未绑定任何安全组，无法检查入站规则，默认不可访问"}
	MsgCheckSGIngressFailed        = Key{string: "检查安全组入站规则失败"}
	MsgPanelPortAccessible         = Key{string: "面板端口可正常访问"}
	MsgPanelPortAccessibleDrifting = Key{string: "面板端口可正常访问（安全组规则同步中，显示结果来自云端实际配置）"}
	MsgPanelPortNotOpen            = Key{string: "面板端口 %d 尚未放通，请联系管理员在规则编辑页追加规则"}
	MsgPanelPortNotOpenDrifting    = Key{string: "面板端口 %d 尚未放通（安全组规则同步中，请稍后重试）"}
	MsgCreateVPCClientFailed       = Key{string: "创建 VPC 客户端失败"}
	MsgQuerySGRulesFailed          = Key{string: "查询安全组规则失败"}
)

// 通道配置 (channel)
var (
	MsgAgentTypeNotSupportChannel           = Key{string: "agent_type %s 不支持通道 %s"}
	MsgChannelOnlyForGroup                  = Key{string: "通道 %s 仅限分组用户使用"}
	MsgChannelNotVisible                    = Key{string: "通道 %s 对当前实例不可见"}
	MsgMissingConfigKeys                    = Key{string: "缺少配置参数: 未提供任何 key 参数"}
	MsgMissingConfigValues                  = Key{string: "缺少配置参数: 未提供任何 value 参数"}
	MsgKeyValueCountMismatch                = Key{string: "参数不匹配: 提供了 %d 个 key 但只有 %d 个 value"}
	MsgEmptyConfigKey                       = Key{string: "参数格式错误: 第 %d 个 key 为空"}
	MsgEmptyConfigValue                     = Key{string: "参数格式错误: key '%s' 对应的 value 为空"}
	MsgParseCustomChannelConfigFailed       = Key{string: "自定义通道配置解析失败"}
	MsgParseSetChannelScriptFailed          = Key{string: "解析 set_channel 脚本失败"}
	MsgParseDelChannelScriptFailed          = Key{string: "解析 del_channel 脚本失败"}
	MsgParseListChannelsScriptFailed        = Key{string: "解析 list_channels 脚本失败"}
	MsgParseChannelListFailed               = Key{string: "解析通道列表失败"}
	MsgChannelConfigFailed                  = Key{string: "通道配置失败"}
	MsgGenerateProxyRouteFailed             = Key{string: "生成代理路由失败"}
	MsgDisableProxyRouteFailed              = Key{string: "禁用代理路由失败"}
	MsgApplyAgentProxySGRulesFailed         = Key{string: "下发代理安全组规则失败"}
	MsgChannelNotSupportAutoConfig          = Key{string: "不支持自动配置的通道类型: %s"}
	MsgAgentTypeNotSupportChannelAutoConfig = Key{string: "agent_type %s 不支持通道 %s 的自动配置，请改用手动配置"}
	MsgAutoChannelDone                      = Key{string: "配置完成"}
	MsgLINEMissingAccessToken               = Key{string: "LINE 通道缺少 channel_access_token 参数"}
	MsgLINEMissingSecret                    = Key{string: "LINE 通道缺少 channel_secret 参数"}
)

// 云 API 代理 (cloud_common_proxy)
var (
	MsgCloudProxyMissingServiceQuery       = Key{string: "缺少 service 参数，格式: /admin/cloud/query/{service}"}
	MsgCloudProxyMissingServiceMutate      = Key{string: "缺少 service 参数，格式: /admin/cloud/mutate/{service}"}
	MsgCloudProxyUnsupportedService        = Key{string: "不支持的 service: %s, 可用: %s"}
	MsgCloudProxyMissingAction             = Key{string: "缺少 X-TC-Action Header 或 Action 参数"}
	MsgCloudProxyActionNotInReadWhitelist  = Key{string: "Action %q 不在 %s 的读接口白名单中, 允许的 Actions: %s"}
	MsgCloudProxyActionNotInWriteWhitelist = Key{string: "Action %q 不在 %s 的写接口白名单中, 允许的 Actions: %s"}
	MsgGetCloudCredentialFailed            = Key{string: "获取云 API 凭证失败"}
	MsgSetRequestParamsFailed              = Key{string: "设置请求参数失败"}
	MsgCSIPAPICallFailedWithAction         = Key{string: "CSIP API 调用失败 [%s]"}
	MsgCSIPAPIErrorWithCodeMessage         = Key{string: "CSIP API 错误 [%s]: %s"}
	MsgCSIPParamError                      = Key{string: "CSIP 设置请求参数错误"}
	MsgCSIPFailedToParseResponse           = Key{string: "解析 CreateSkillScan 响应失败"}
	MsgCSIPQueryScanRecordFailed           = Key{string: "查询已有扫描记录失败"}
	MsgCSIPFailedToCreateScanRecord        = Key{string: "创建扫描记录失败"}
	MsgCSIPQueryPendingScansFailed         = Key{string: "查询待扫描记录失败"}
	MsgCSIPUpdateTimeoutStatusFailed       = Key{string: "更新超时状态失败"}
	MsgCSIPDescribeResultCallFailed        = Key{string: "调用 DescribeSkillScanResult 失败"}
	MsgCSIPParseDescribeResultFailed       = Key{string: "解析 DescribeSkillScanResult 响应失败"}
	MsgCSIPDescribeResultError             = Key{string: "DescribeSkillScanResult 错误 [%s]: %s"}
	MsgCSIPParseScanDetailFailed           = Key{string: "解析扫描结果详情失败"}
	MsgCSIPUpdateScanSuccessFailed         = Key{string: "更新扫描成功状态失败"}
	MsgCSIPUpdateScanFailedFailed          = Key{string: "更新扫描失败状态失败"}
	MsgCSIPCreateViolationFailed           = Key{string: "创建违规记录失败 (rule=%s)"}
)

var (
	MsgFailedToReadRuleSet      = Key{string: "读取 rule_set 失败 (name=%q)"}
	MsgUpdateRuleSetConflict    = Key{string: "另一个管理员正在保存规则，请稍后重试"}
	MsgRuleSetDistributeFailed  = Key{string: "%d 个安全组规则下发失败，所有变更已回滚"}
	MsgUpdateRuleSetFailed      = Key{string: "所有云端安全组已更新但 DB 写入失败，Guardian 将在下次同步"}
	MsgListActiveSGFailed       = Key{string: "列出所有 ACTIVE SG 失败"}
	MsgMarshalMergedRulesFailed = Key{string: "序列化合并后的规则失败"}
)

// MCP 配置 (mcp)
var (
	MsgMcpMaxInstancesPerCall           = Key{string: "单次最多下发 50 个实例"}
	MsgMcpConfigJsonInvalid             = Key{string: "config_json 格式错误: %v"}
	MsgMcpConfigJsonMissingURLOrCommand = Key{string: "config_json 必须包含 url 或 command 字段"}
	MsgMcpNotFound                      = Key{string: "MCP 不存在"}
	MsgMcpNoAccess                      = Key{string: "无权访问该 MCP"}
	MsgInstanceNotRunningWithState      = Key{string: "实例未运行（状态: %s）"}
	MsgInstanceStartingAgentNotReady    = Key{string: "实例正在启动中，Agent 尚未就绪，请稍后重试"}
	MsgMcpSaveConfigFailed              = Key{string: "保存配置失败"}
	MsgQueryMcpInstallationsFailed      = Key{string: "查询 MCP 安装记录失败"}
	MsgMcpInstanceRefreshing            = Key{string: "该实例正在刷新状态中，请稍后再试"}
	MsgMcpNotInstalled                  = Key{string: "该 MCP 未安装"}
	MsgMcpConfigParseFailed             = Key{string: "配置解析失败"}
	MsgMcpConfigMarshalFailed           = Key{string: "配置序列化失败"}
	MsgMissingParamID                   = Key{string: "缺少参数 id"}
	MsgMcpPlaceholderUnfilled           = Key{string: "配置中存在未填写的占位符: %s"}
	MsgMcpMissingHostedFields           = Key{string: "请填写以下托管字段: %s"}
	MsgMcpKeyHostedNoManualEdit         = Key{string: "凭据托管的 MCP 不支持手动修改配置"}
	MsgMcpConfigDeployFailed            = Key{string: "配置下发失败: %s"}
	MsgMcpDeleteFailedWithDetail        = Key{string: "删除失败: %s"}
	MsgMcpKeyHostedNeedPlaceholder      = Key{string: "开启凭据托管时，config_json 的 headers 中至少需要一个占位符字段（如 <your-token>）"}
	MsgMcpKeyHostedRequirePlaceholder   = Key{string: "该 MCP 已开启密钥托管，config_json 中必须包含占位符"}
	MsgMcpHostedDefaultNotPlaceholder   = Key{string: "hosted_defaults 中的 %q 不是 config_json 中的占位符"}
	MsgMcpModifyFieldNotAllowed         = Key{string: "不允许修改配置字段 %s"}
	MsgMcpAddFieldNotAllowed            = Key{string: "不允许新增配置字段 %s"}
	MsgMcpModifyURLPathNotAllowed       = Key{string: "不允许修改 URL 路径"}
	MsgMcpModifyURLParamNotAllowed      = Key{string: "不允许修改 URL 参数 %s"}
	MsgMcpHeaderValueMustBeString       = Key{string: "header %s 值必须是字符串"}
	MsgMcpModifyHeaderNotAllowed        = Key{string: "不允许修改 header %s"}
	MsgMcpAddHeaderNotAllowed           = Key{string: "不允许新增 header %s"}
	MsgMcpSaveHostedKeyFailed           = Key{string: "保存托管字段 %s 失败"}
)

// 模型管理 (model)
var (
	MsgModelNotFoundOrDisabled             = Key{string: "模型不存在或已禁用"}
	MsgModelNotFoundDisabledOrInvisible    = Key{string: "模型不存在、已禁用或不可见"}
	MsgModelNoAccess                       = Key{string: "您无权使用该模型"}
	MsgModelDomainNotConfigured            = Key{string: "服务地址未配置，请通过 -domain 启动参数设置"}
	MsgModelParseSetModelScriptFailed      = Key{string: "解析 set_model 脚本失败"}
	MsgModelTATExecuteFailed               = Key{string: "TAT 执行失败"}
	MsgModelSaveBindingFailed              = Key{string: "保存模型绑定失败"}
	MsgBatchSetModelSuccess                = Key{string: "模型配置已覆盖"}
	MsgBatchSetModelDuplicateModel         = Key{string: "主模型和备选模型不能重复，备选模型之间也不能重复"}
	MsgBatchSetModelMixedAgentTypes        = Key{string: "所选 Agent 包含 openclaw 与非 openclaw 两种类型，不能一起批量配置模型"}
	MsgBatchSetModelFallbackUnsupported    = Key{string: "非 openclaw 类型仅支持配置一个模型"}
	MsgBatchSetModelCustomAgentUnsupported = Key{string: "自定义 Agent 类型不支持批量配置模型"}
	MsgModelCustomModelDisabled            = Key{string: "自定义模型功能未开启"}
	MsgModelCustomModelFieldsRequired      = Key{string: "模型ID、API Key、URL、接口类型不能为空"}
	MsgModelMarshalConfigFailed            = Key{string: "序列化配置失败"}
	MsgModelSaveCustomBindingFailed        = Key{string: "保存自定义模型绑定失败"}
	MsgModelAgentFallbackUnsupported       = Key{string: "Agent 3.28 暂不支持多模型 fallback 功能，请升级到更高版本后再使用"}
	MsgModelFallbackUnsupportedByAgentType = Key{string: "%s 类型实例仅支持单模型，不支持 primary/fallback 多模型配置"}
	MsgModelAlreadyBound                   = Key{string: "该模型已绑定到此实例"}
	MsgModelAlreadyBoundAsFallback         = Key{string: "该模型已被绑定为 fallback 模型，请先解绑后再设置为主模型"}
	MsgModelCreateBindingFailed            = Key{string: "创建模型绑定记录失败"}
	MsgModelCustomMissingModelID           = Key{string: "自定义模型缺少 model_id"}
	MsgModelCustomAlreadyBound             = Key{string: "该自定义模型已绑定到此实例"}
	MsgModelCreateCustomBindingFailed      = Key{string: "创建自定义模型绑定记录失败"}
	MsgModelCustomIDEmpty                  = Key{string: "模型ID不能为空"}
	MsgModelCustomIDInvalidChars           = Key{string: "模型ID仅允许字母、数字、'.'、'_'、'-'、':'、'/'，长度 1~128"}
	MsgModelURLInvalid                     = Key{string: "URL 格式无效"}
	MsgModelURLSchemeNotHTTP               = Key{string: "URL 必须以 http:// 或 https:// 开头"}
	MsgModelURLHostEmpty                   = Key{string: "URL 缺少有效的主机地址"}
	MsgModelTypeInvalid                    = Key{string: "接口类型无效，仅支持 openai-completions 或 anthropic-messages"}
	MsgModelInputTypeInvalid               = Key{string: "输入类型无效，仅支持 text、image"}
	MsgModelInvalidInstanceModelID         = Key{string: "缺少或无效的 instance_model_id 参数"}
	MsgModelTargetNotFoundOrNotInInstance  = Key{string: "目标模型记录不存在或不属于该实例"}
	MsgModelCannotSwitchToSelf             = Key{string: "无法切换到自身：目标模型已是当前主模型"}
	MsgModelBeginTxFailed                  = Key{string: "开启事务失败"}
	MsgModelDemoteOldPrimaryFailed         = Key{string: "降级原主模型失败"}
	MsgModelPromoteTargetFailed            = Key{string: "提升目标模型失败"}
	MsgModelUpdateInstancePrimaryFailed    = Key{string: "更新实例主模型失败"}
	MsgModelCommitTxFailed                 = Key{string: "提交事务失败"}
	MsgModelGenImageModelParamFailed       = Key{string: "生成 imageModel 参数失败"}
	MsgModelCountBindingFailed             = Key{string: "统计模型绑定数失败"}
	MsgModelDeleteBindingFailed            = Key{string: "删除模型绑定失败"}
	MsgModelQueryCandidatePrimaryFailed    = Key{string: "查询候选主模型失败"}
	MsgModelPromoteNextPrimaryFailed       = Key{string: "提升主模型失败"}
	MsgModelUpdatePrimaryIDFailed          = Key{string: "更新实例主模型 ID 失败"}
	MsgModelClearPrimaryIDFailed           = Key{string: "清空实例主模型 ID 失败"}
	MsgModelGenPrimaryFallbacksFailed      = Key{string: "生成 primary/fallbacks 参数失败"}
	MsgModelQueryInstanceModelsFailed      = Key{string: "查询实例模型列表失败"}
	MsgModelBatchQueryAIModelsFailed       = Key{string: "批量查询 ai_models 失败"}
	MsgModelMarshalFallbacksFailed         = Key{string: "序列化 imageModel.fallbacks 失败"}
	MsgModelConfigFailedTitle              = Key{string: "模型配置失败"}
	MsgModelAddFailedTitle                 = Key{string: "添加模型失败"}
	MsgModelSwitchFailedTitle              = Key{string: "切换主模型失败"}
	MsgModelDeleteSyncFailedTitle          = Key{string: "删除模型后同步配置失败"}
	MsgModelVersionParseFailed             = Key{string: "版本信息解析失败"}
	MsgModelDomainNotConfiguredAlt         = Key{string: "服务地址未配置"}
	MsgModelQueryListFailed                = Key{string: "查询模型列表失败"}
	MsgModelCleanProviderFailed            = Key{string: "清理 provider 失败"}
	MsgModelSwitchTATFailed                = Key{string: "switch_model TAT 失败"}
	MsgModelUnknownModel                   = Key{string: "未知模型"}
	MsgModelCustomHeadersSizeExceeded      = Key{string: "自定义 HTTP 头部超过最大长度限制 %d 字节"}
	MsgModelCustomHeadersInvalidFormat     = Key{string: "自定义 HTTP 头部格式无效，需为 JSON 对象字符串"}
	MsgModelCustomHeadersCountExceeded     = Key{string: "自定义 HTTP 头部最多允许 %d 个，实际 %d 个"}
	MsgModelCustomHeadersKeyEmpty          = Key{string: "自定义 HTTP 头部键名不得为空"}
	MsgModelCustomHeadersKeyInvalidChars   = Key{string: "自定义 HTTP 头部键名 '%s' 仅允许字母、数字、'-'、'_'"}
	MsgModelCustomHeadersKeyReserved       = Key{string: "自定义 HTTP 头部键名 '%s' 为保留头部，不允许设置"}
	MsgModelCustomHeadersValueNewline      = Key{string: "自定义 HTTP 头部值不得包含换行符 (键: '%s')"}
)

// 云端浏览器相关 (browser_vnc.go)
var (
	MsgVNCFeatureNotEnabled        = Key{string: "云端浏览器功能未开启"}
	MsgVNCContactAdminToEnable     = Key{string: "云端浏览器功能未开启，请联系管理员在后台开启"}
	MsgInstanceNoPublicIP          = Key{string: "实例无公网 IP"}
	MsgInstanceNoPublicIPForVNC    = Key{string: "实例无公网 IP，无法使用云端浏览器"}
	MsgVNCConnectionLimitReached   = Key{string: "该实例 VNC 连接数已达上限（%d）"}
	MsgVNCCVMConnectFailed         = Key{string: "连接 CVM VNC 服务失败"}
	MsgVNCBackendResponseAbnormal  = Key{string: "CVM VNC 服务响应异常"}
	MsgVNCBackendReturnedStatus    = Key{string: "CVM VNC 服务返回 %d"}
	MsgServerNoUpgradeSupport      = Key{string: "服务器不支持连接升级"}
	MsgWebSocketRequired           = Key{string: "需要 WebSocket 连接"}
	MsgMissingWebSocketKey         = Key{string: "缺少 Sec-WebSocket-Key 头"}
	MsgVNCInstallFailed            = Key{string: "安装 VNC 环境失败"}
	MsgParseInstallResultFailed    = Key{string: "解析安装结果失败"}
	MsgInstanceInstallingNoRepeat  = Key{string: "该实例正在安装中，请勿重复操作"}
	MsgInvalidActionParameter      = Key{string: "action 参数无效，可选值: start, stop"}
	MsgInstanceStateNotRunning     = Key{string: "实例当前状态为 %s，云端浏览器仅在实例运行中时可用"}
	MsgInstanceStateNotRunningRUN  = Key{string: "实例当前状态为 %s，云端浏览器仅在实例运行中（RUNNING）时可用"}
	MsgCheckVNCEnvFailed           = Key{string: "检查 VNC 环境失败"}
	MsgParseCheckResultFailed      = Key{string: "解析检查结果失败"}
	MsgVNCWhitelistRequired        = Key{string: "未配置 VNC 白名单规则（allow_vnc_whitelist），无法放通端口"}
	MsgVNCSGNotConfigured          = Key{string: "未配置安全组，无法自动放通 VNC 端口"}
	MsgVNCSGCheckFailed            = Key{string: "安全组规则检查失败，请联系管理员"}
	MsgVNCSGNotConfiguredCheck     = Key{string: "未配置安全组，无法检查端口放通状态"}
	MsgVNCInstanceWithoutSG        = Key{string: "实例未绑定任何安全组，无法检查端口放通状态"}
	MsgVNCSGSyncing                = Key{string: "安全组规则同步中，端口 %d 尚未放通，请稍后重试"}
	MsgVNCSGNotOpened              = Key{string: "安全组未放通端口 %d(noVNC)，请联系管理员检查安全组配置"}
	MsgVNCSGNotOpenedAlt           = Key{string: "安全组未放通端口 %d(noVNC)，请联系管理员检查云端浏览器开关是否开启"}
	MsgVNCDeleteOldSGRuleFailed    = Key{string: "删除旧 0.0.0.0/0 安全组规则失败"}
	MsgVNCAddWhitelistSGRuleFailed = Key{string: "添加白名单安全组规则失败"}
	MsgVNCCheckPortRuleFailed      = Key{string: "检查端口规则失败"}
	MsgVNCAddPortRuleDeprecated    = Key{string: "addSecurityGroupPortRule 已禁用：VNC 端口放通请使用 ensureBrowserVNCPortRule（白名单模式）"}
)

// 实例管理相关 (openclaw.go, admin_instances.go)
var (
	MsgAPITokenDisabled                            = Key{string: "API Token 已被禁用"}
	MsgSessionExpiredRelogin                       = Key{string: "登录态已失效，请重新登录"}
	MsgInstanceNameRequired                        = Key{string: "实例名称不能为空且不能超过128个字符"}
	MsgInstanceNotReadyForNameChange               = Key{string: "实例尚未就绪，无法修改名称"}
	MsgCurrentStateCannotChangeName                = Key{string: "当前实例状态为 %s，无法修改名称"}
	MsgModifyCVMNameFailed                         = Key{string: "修改 CVM 名称失败"}
	MsgUpdateLocalNameFailed                       = Key{string: "更新本地名称失败"}
	MsgRoleNotExistOrUnavailable                   = Key{string: "所选角色不存在或不可用"}
	MsgGroupNotExist                               = Key{string: "所选分组不存在"}
	MsgGroupNotSupportRole                         = Key{string: "所选分组不支持该角色"}
	MsgRoleNotSupportUngroupedUser                 = Key{string: "该角色不支持未分组用户使用"}
	MsgTypeCannotCreateContactAdmin                = Key{string: "%s 类型暂不可创建，请联系管理员"}
	MsgCVMInstanceDestroyFailed                    = Key{string: "销毁 CVM 实例失败"}
	MsgGetTerminalLoginURLFailed                   = Key{string: "获取终端登录 URL 失败"}
	MsgAgentTypeDoNotSupportModelConfigWithDetail  = Key{string: "%s 类型实例不支持模型配置"}
	MsgAgentTypeDoNotSupportPluginWithDetail       = Key{string: "%s 类型实例不支持插件功能"}
	MsgAgentTypeDoNotSupportChatbotWithDetail      = Key{string: "%s 类型实例不支持 Chatbot 功能"}
	MsgAgentTypeDoNotSupportDetailConfigWithDetail = Key{string: "%s 类型实例不支持详细配置"}
	MsgAgentTypeDoNotSupportSkillWithDetail        = Key{string: "%s 类型实例不支持技能功能"}
	MsgAgentTypeDoNotSupportReinstallWithDetail    = Key{string: "%s 类型实例不支持重装，请删除后重建"}
	MsgAgentTypeDoNotSupportUpgradeWithDetail      = Key{string: "%s 类型实例暂不支持一键升级"}
	MsgAgentTypeDoNotSupportBrowserVNCWithDetail   = Key{string: "%s 类型实例不支持云端浏览器功能"}
	MsgAgentTypeDoNotSupportApproveWithDetail      = Key{string: "%s 类型实例的授权流程由各自 Server/OAuth 处理，无需调用此接口"}
	MsgAgentTypeDoNotSupportMcpWithDetail          = Key{string: "%s 类型实例不支持 MCP 功能"}
	MsgReinstallImageTypeMismatchWithDetail        = Key{string: "启用镜像的类型（%s）与实例类型（%s）不匹配，无法重装"}
	MsgMcpVersionTooLow                            = Key{string: "实例版本过低（当前 %s），MCP 功能需要 %s 及以上版本"}
)

// 用户管理相关 (admin_users.go)
var (
	MsgInstanceQuotaMustBeInteger   = Key{string: "实例配额必须为 0~999 的整数"}
	MsgTokenQuotaMustBeValid        = Key{string: "Token 配额必须为 -1 或非负整数"}
	MsgCannotDeleteInitialAdmin     = Key{string: "不能删除初始管理员"}
	MsgShutdownFailed               = Key{string: "关机失败"}
	MsgDisableUserFailed            = Key{string: "禁用用户失败"}
	MsgUserHasInstancesExist        = Key{string: "该用户还有实例存在，请先删除其实例"}
	MsgUserVPCHasResources          = Key{string: "用户自动创建的关联 VPC（%s）下仍有资源占用，请先清理后再删除用户"}
	MsgDeleteUserFailed             = Key{string: "删除用户失败"}
	MsgCannotOperateInitialAdmin    = Key{string: "不能操作初始管理员"}
	MsgPasswordCannotBeEmpty        = Key{string: "密码不能为空"}
	MsgInitialAdminPasswordReset    = Key{string: "初始管理员密码只能通过 admin-token 重置"}
	MsgCannotModifyInitialAdminRole = Key{string: "不能修改初始管理员的角色"}
	MsgInstanceQuotaDetailed        = Key{string: "实例配额必须为 -1 或 0~999 的整数（-1 表示无限制，0 表示无配额）"}
	MsgNoFieldsToUpdate             = Key{string: "没有可更新的字段"}
	MsgUserNoAPIToken               = Key{string: "该用户没有 API Token"}
	MsgTokenAlreadyDisabled         = Key{string: "该用户 Token 已处于禁用状态"}
	MsgTokenNotDisabled             = Key{string: "该用户 Token 未被禁用"}
	MsgTokenOperationFailed         = Key{string: "%s Token 失败"}
	MsgInitialAdminNotExist         = Key{string: "初始管理员不存在"}
	MsgUserListEmptyOrTooLarge      = Key{string: "用户列表不能为空，不能超过5000"}
	MsgUserLimitExceededImport      = Key{string: "导入后将超过用户数上限（%d），当前已有 %d 个用户"}
	MsgOnlyGetMethod                = Key{string: "仅支持 GET 方法"}
	MsgQueryVpcFailed               = Key{string: "查询 VPC 失败"}
	MsgGroupMemberLimitReached      = Key{string: "目标用户组成员数量已达上限（10000 人），无法加入"}
	MsgOneIDReadonlyUserOp          = Key{string: "OneID 模式下不允许在管控端修改用户与 OneID 同步分组的关系，请到 OneID 系统操作后等待同步"}
)

// 升级相关 (openclaw_upgrade.go)
var (
	MsgUpgradeAlreadyLatest               = Key{string: "实例已是最新版本，无需升级"}
	MsgUpgradeStarted                     = Key{string: "升级已开始"}
	MsgAdminStatsLabelNeedAttention       = Key{string: "需关注"}
	MsgAdminStatsLabelInProgress          = Key{string: "处理中"}
	MsgUpgradeRetryAlreadyLatest          = Key{string: "实例版本已是最新，且无历史备份可复用，升级重试已完成"}
	MsgUpgradeRetryVersionDowngrade       = Key{string: "当前实例 OpenClaw 版本（%s）高于官方镜像版本（%s），不允许通过官方镜像降级升级"}
	MsgUpgradeVersionDowngradeCustomImage = Key{string: "当前实例 OpenClaw 版本（%s）高于目标自定义镜像版本（%s），不允许通过自定义镜像降级升级，请联系管理员检查后台启用镜像配置"}
	MsgUpgradeRetryStartedWithBackup      = Key{string: "升级重试已开始（复用 SMH 历史备份）"}
	MsgUpgradeRetryStarted                = Key{string: "升级重试已开始"}
)

// 分组配置相关 (admin_group_config.go)
var (
	MsgQueriesNotEmpty     = Key{string: "queries 不能为空数组"}
	MsgInvalidConfigType   = Key{string: "config_type 无效: %s"}
	MsgInvalidConfigKey    = Key{string: "策略名无效"}
	MsgGroupNotFound       = Key{string: "分组不存在"}
	MsgPolicyNotConfigured = Key{string: "该组未配置此策略项"}
	MsgChannelNotFound     = Key{string: "通道不存在: %s"}
)

// 升级 (upgrade)
var (
	// Handler 用户可见
	MsgUpgradeOperationInProgress    = Key{string: "实例正在进行 %s 操作，请稍后再试"}
	MsgUpgradeNoImageForType         = Key{string: "暂无生效的%s类型镜像，请联系管理员处理"}
	MsgUpgradeRetryNotInFailedState  = Key{string: "当前不是升级失败状态，无法重试（current_operation=%s, state=%s）"}
	MsgUpgradeClearFailedStateFailed = Key{string: "清除升级失败态失败"}
	MsgUpgradeRetrySetOpLockFailed   = Key{string: "设置升级重试操作锁失败"}

	// 内部 helper / async（writeError / createErrorNotification 链上）
	MsgUpgradeFixOfficialImageRuntimeUserFailed = Key{string: "校正官方镜像 runtime_user/home 失败"}
	MsgUpgradeDefaultImageEmpty                 = Key{string: "默认镜像为空，无法进行升级检查"}
	MsgUpgradeCannotGetCVMInfo                  = Key{string: "无法获取 CVM 实例 %s 的信息"}
	MsgUpgradeInstanceNotRunning                = Key{string: "实例非运行状态，无法执行升级"}
	MsgUpgradeCVMImageIDEmpty                   = Key{string: "CVM 实例 %s 的镜像 ID 为空"}
	MsgUpgradeInstanceNotRecover                = Key{string: "实例 %s 在 %v 内未恢复到 RUNNING 状态"}
	MsgUpgradeResolveCheckReadyFailed           = Key{string: "无法解析 check_ready 脚本 (agent_type=%s)"}
	MsgUpgradeWaitAgentReadyCanceled            = Key{string: "等待 agent 就绪被取消"}
	MsgUpgradeOpenclawNotReady                  = Key{string: "实例 %s 上的 openclaw 在 %v 内未就绪"}
	MsgUpgradeRenewSMHCredFailed                = Key{string: "续期 SMH 上传凭证失败"}
	MsgUpgradeBackupFailed                      = Key{string: "数据备份失败"}
	MsgUpgradeBackupArchivePathMissing          = Key{string: "无法获取备份压缩包路径，恢复中止"}
	MsgUpgradeSMHCredMissingChunkURL            = Key{string: "SMH 上传凭证缺少分块上传 URL 模板"}
	MsgUpgradeLoadUploadScriptFailed            = Key{string: "加载上传脚本失败"}
	MsgUpgradeUploadChunkFailed                 = Key{string: "上传备份包分块 %d/%d 到 SMH 失败"}
	MsgUpgradeConfirmSMHUploadFailed            = Key{string: "确认 SMH 上传失败"}
	MsgUpgradeFileKeyEmpty                      = Key{string: "reinstallAndRestore: fileKey 不能为空"}
	MsgUpgradeBuildSMHDownloadURLFailed         = Key{string: "生成 SMH 下载 URL 失败"}
	MsgUpgradeWaitReinstallTimeout              = Key{string: "等待实例重装完成超时"}
	MsgUpgradeRetryReinstallFailed              = Key{string: "第 %d 次重新重装实例失败"}
	MsgUpgradeRetryReinstallWaitTimeout         = Key{string: "第 %d 次重新重装后等待实例恢复超时"}
	MsgUpgradeAgentReadyAttemptsExhausted       = Key{string: "TAT Agent 经过 %d 次重装仍未就绪，升级中止"}
	MsgUpgradeWaitRuntimeUserTimeout            = Key{string: "等待运行用户 %s 在新系统上创建超时"}
	MsgUpgradeRedetectRuntimeUserFailed         = Key{string: "重装后探测 %s 运行用户失败：detect_install 脚本连续超时或返回无效结果"}
	MsgUpgradePersistRuntimeUserFailed          = Key{string: "重装后保存 %s 运行用户失败"}
	MsgUpgradeWaitOpenclawReadyTimeout          = Key{string: "等待 openclaw 就绪超时，数据恢复中止"}
	MsgUpgradeRestoreFailed                     = Key{string: "数据恢复失败"}
	MsgUpgradeResetAgentReadyFailed             = Key{string: "重置 agent_ready 失败"}
	MsgUpgradeCheckUpgradeStatusFailed          = Key{string: "检查升级状态失败"}
	MsgFailedToSetUpgradeOpLock                 = Key{string: "设置升级操作锁失败"}

	// 升级前存储空间预探测 (precheck_upgrade_space.sh)
	// %s 依次为：预估需要空间、当前 /tmp 可用空间（均为人类可读格式，如 "512 MB"）
	MsgUpgradePrecheckDiskInsufficient = Key{string: "存储空间不足（预估需要 %s，当前可用 %s），请清理磁盘后重试"}
	// 异步阶段二次探测失败使用的通知文案（复用 NotifyTypeInstanceUpgradeFailed 类型）
	MsgUpgradeAbortedDiskInsufficient = Key{string: "因存储空间不足（预估需要 %s，当前可用 %s），本次升级未开始，原实例仍可正常使用，请清理后重试"}
	// 通知标题（异步中止分支使用）
	MsgUpgradeAbortedTitle = Key{string: "实例升级未开始"}
	// 备份阶段本地数据库损坏且无法无损修复时的中止通知文案（复用 NotifyTypeInstanceUpgradeFailed 类型）
	MsgUpgradeAbortedDBUnrecoverable = Key{string: "检测到本地数据库损坏且无法自动修复，为避免重装导致数据永久丢失，本次升级未开始，原实例仍可正常使用，请联系管理员或技术支持排查数据库后再升级"}
	// 升级失败通知标题
	MsgUpgradeFailedTitle = Key{string: "实例升级失败"}
	// 升级成功通知标题
	MsgUpgradeSuccessTitle = Key{string: "实例升级完成"}
	// 升级成功通知正文（%s = 实例名称）
	MsgUpgradeSuccessContent = Key{string: "您的实例「%s」已升级到最新版本。"}
	// recovery 兜底删库重建空库后的降级通知标题（升级本身仍算成功）
	MsgUpgradeSuccessDBRebuiltTitle = Key{string: "实例升级完成（历史数据未能恢复）"}
	// recovery 兜底删库重建空库后的降级通知正文（%s = 实例名称）
	MsgUpgradeSuccessDBRebuiltContent = Key{string: "您的实例「%s」已升级到最新版本；但升级过程中检测到本地数据库损坏且无法修复，历史会话/数据已重置为初始状态（损坏数据已在服务器备份留存）。"}
)

// CLS 采集范围 (admin_cls_scope.go)
var (
	MsgQueryCLSScopeFailed            = Key{string: "查询 CLS 采集范围失败"}
	MsgCLSServiceNotEnabled           = Key{string: "CLS 服务未开启，请先开启 CLS 服务"}
	MsgInvalidCLSScopeType            = Key{string: "scope_type 必须为 \"}all\" 或 \"group\""}
	MsgCLSScopeGroupCountExceed       = Key{string: "分组数量不能超过 %d"}
	MsgUpdateCLSScopeModeFailed       = Key{string: "更新 CLS scope_mode 失败"}
	MsgUpdateCLSScopeFailed           = Key{string: "更新 CLS 采集范围失败"}
	MsgFailedToRetrieveCLSServiceInfo = Key{string: "获取 CLS 服务信息失败"}
	MsgCLSQueryExistingScopeFailed    = Key{string: "查询现有 CLS 采集范围失败"}
	MsgCLSDescendantExpandFailed      = Key{string: "展开分组子孙失败"}
	MsgFailedToQueryInstanceVersion   = Key{string: "查询实例版本信息失败"}
)

// Agent 命令任务 (admin_agent_command_tasks.go, admin_agent_command_agent_status.go)
var (
	MsgTooManyInstanceIDs      = Key{string: "instance_ids 数量超过上限 %d"}
	MsgCancelInvocationsFailed = Key{string: "取消任务失败"}
)

// LightClaw (lightclaw.go)
var (
	MsgServiceDomainNotConfigured       = Key{string: "服务域名未配置"}
	MsgInstanceProxyTokenNotConfigured  = Key{string: "实例 ProxyToken 未配置"}
	MsgLightClawInstanceServiceNotReady = Key{string: "获取鉴权失败：实例服务尚未就绪，请稍后重试"}
	MsgLightClawApproveDeviceFailed     = Key{string: "获取鉴权失败，请刷新页面重试"}
	MsgInvalidProduct                   = Key{string: "product 无效或缺失"}
	MsgInvalidAccessToken               = Key{string: "accessToken 无效或缺失"}
	MsgUserDataAbnormal                 = Key{string: "用户数据异常"}
	MsgInvocationIdsRequired            = Key{string: "InvocationIds 不能为空"}
	MsgInvocationTaskIdsRequired        = Key{string: "InvocationTaskIds 不能为空"}
	MsgContentRequired                  = Key{string: "Content 不能为空"}
	MsgInstanceIdsMismatch              = Key{string: "InstanceIds 与实例不匹配"}
)

// 技能安全检测 (admin_skill_security_scan.go)
var (
	MsgSkillNotExist              = Key{string: "技能不存在"}
	MsgSkillNotPublished          = Key{string: "技能未上架，暂不可下发"}
	MsgQuerySkillFailed           = Key{string: "查询技能失败"}
	MsgQueryChannelFailed         = Key{string: "查询渠道失败"}
	MsgGenerateDownloadLinkFailed = Key{string: "生成下载链接失败"}
	MsgDownloadSkillFileFailed    = Key{string: "下载技能文件失败"}
	MsgReadSkillFileFailed        = Key{string: "读取技能文件失败 (status=%d)"}
	MsgTriggerScanFailed          = Key{string: "触发安全检测失败"}
	MsgSkillScanSubmitted         = Key{string: "已提交安全检测，预计 5 分钟后完成"}
	MsgUpdateConfigFailed         = Key{string: "更新配置失败"}
)

// SMH 管理
var (
	MsgEnabledMustBe01               = Key{string: "参数 enabled 必须为 0 或 1"}
	MsgPersonalSpaceNotFound         = Key{string: "个人空间不存在"}
	MsgActionMustBeEnableDisable     = Key{string: "参数 action 必须为 enable 或 disable"}
	MsgInstanceIdsCannotBeEmpty      = Key{string: "instance_ids 不能为空"}
	MsgUpdateSiteConfigFailed        = Key{string: "更新 SiteConfig 失败"}
	MsgSMHServiceNotEnabled          = Key{string: "SMH 服务未启用，请先在管理后台开通 SMH 服务"}
	MsgSMHAgentTypeNotSupported      = Key{string: "%s 类型实例不支持网盘功能"}
	MsgSMHInstanceUserNotFound       = Key{string: "实例所属用户不存在"}
	MsgSMHCreatePersonalSpaceFailed  = Key{string: "创建个人空间失败"}
	MsgSMHRestorePersonalSpaceFailed = Key{string: "恢复个人空间失败"}
	MsgSMHSpaceStatusChanged         = Key{string: "空间状态已变更，请刷新后重试"}
	MsgSMHNoActivePersonalSpace      = Key{string: "该实例无活跃个人空间"}
	MsgSMHMarkRecycleFailed          = Key{string: "标记个人空间待回收失败"}
)

// 技能包收藏 (admin_skillset.go)
var (
	MsgFavoriteSkillSetFailed = Key{string: "收藏技能包失败"}
	MsgSkillsetNotFound       = Key{string: "技能集不存在"}
	MsgIDOrSlugRequired       = Key{string: "缺少参数 id 或 slug"}
	MsgIDAndSlugConflict      = Key{string: "id 和 slug 不能同时指定"}
	MsgInvalidIDFormat        = Key{string: "id 格式无效"}
)

// 用户组树 (admin_user_group_tree.go)
var (
	MsgUserGroupIDRequired                 = Key{string: "用户组 ID 不能为空"}
	MsgUserGroupIDFormatError              = Key{string: "用户组 ID 格式错误"}
	MsgPartialUserGroupsNotFound           = Key{string: "部分用户组不存在"}
	MsgGroupTreeSubLabelInitialSkillBundle = Key{string: "初始技能包"}
	MsgGroupTreeSubLabelRole               = Key{string: "角色"}
	MsgGroupTreeSubLabelSkillSource        = Key{string: "技能安装来源"}
	MsgGroupTreeSubLabelEnterpriseSkill    = Key{string: "企业技能"}
	MsgGroupTreeSubLabelEnterprisePlugin   = Key{string: "企业插件"}
	MsgGroupTreeSubLabelEnterpriseMCP      = Key{string: "企业MCP"}
	MsgGroupTreeSubLabelSecurityGroup      = Key{string: "安全组"}
	MsgGroupTreeSubLabelInternet           = Key{string: "公网"}
	MsgGroupTreeSubLabelInternetPublicIP   = Key{string: "公网 IP"}
	MsgGroupTreeSubLabelInternetChargeType = Key{string: "计费模式"}
	MsgGroupTreeSubLabelInternetBandwidth  = Key{string: "带宽上限"}
	MsgGroupTreeLabelInternetConfig        = Key{string: "公网配置"}
	MsgGroupTreeSubLabelVpcSubnet          = Key{string: "私有网络与子网"}
	MsgGroupTreeAutoAssign                 = Key{string: "自动分配"}
	MsgGroupTreeLabelOff                   = Key{string: "关闭"}
	MsgGroupTreeLabelOn                    = Key{string: "开启"}
	MsgGroupTreeLabelDefault               = Key{string: "默认"}
	MsgGroupTreeMemoryProEdition           = Key{string: "Pro 版"}
	MsgGroupTreeMemoryFreeEdition          = Key{string: "Free 版"}
	MsgGroupTreeAIAgentSecurityBasic       = Key{string: "基础版"}
	MsgGroupTreeAIAgentSecurityFlagship    = Key{string: "旗舰版"}
	MsgGroupTreeSubLabelUserQuota          = Key{string: "用户配额"}
	MsgGroupTreeSubLabelModelQuota         = Key{string: "模型配额"}
	MsgGroupTreeSubLabelFeatureToggle      = Key{string: "功能权限开关"}
	MsgGroupTreeMetaUnlimited              = Key{string: "无限制"}
	MsgQuotaPeriodDaily                    = Key{string: "每日"}
	MsgQuotaPeriodMonthly                  = Key{string: "每月"}
	MsgQuotaNoEndTime                      = Key{string: "无终止"}
	MsgQuotaRefreshDaily                   = Key{string: "按日刷新"}
	MsgQuotaRefreshMonthly                 = Key{string: "按月刷新"}
	MsgQuotaRefreshYearly                  = Key{string: "按年刷新"}
	MsgQuotaRefreshNone                    = Key{string: "不刷新"}
	// config-diff 分类标签（计费模式复用 MsgGroupTreeSubLabelInternetChargeType）
	MsgCategoryModel           = Key{string: "模型"}
	MsgCategoryChannel         = Key{string: "通道"}
	MsgCategorySkill           = Key{string: "技能"}
	MsgCategoryAgentTool       = Key{string: "Agent 工具"}
	MsgCategoryMemory          = Key{string: "记忆"}
	MsgCategoryDrive           = Key{string: "网盘"}
	MsgCategoryImageType       = Key{string: "镜像"}
	MsgCategoryNetwork         = Key{string: "网络"}
	MsgCategoryCLS             = Key{string: "CLS 日志服务"}
	MsgCategoryAIAgentSecurity = Key{string: "AI Agent 安全"}
	MsgCategoryPlatformPolicy  = Key{string: "平台策略"}
	// config-diff 策略标签
	MsgPolicyInstanceQuota                = Key{string: "单用户 Agent 数量上限"}
	MsgPolicyTokenQuotaDay                = Key{string: "单用户 Tokens 上限"}
	MsgPolicyTokenQuotaRules              = Key{string: "用户 Token 配额规则"}
	MsgPolicyGlobalTokenQuotaDay          = Key{string: "全局 Tokens 上限"}
	MsgPolicyGlobalTokenQuotaRules        = Key{string: "全局 Token 配额规则"}
	MsgPolicyGlobalTokenQuotaRulesDaily   = Key{string: "全局 Token 配额规则（站点周期：每日）"}
	MsgPolicyGlobalTokenQuotaRulesMonthly = Key{string: "全局 Token 配额规则（站点周期：每月）"}
	MsgPolicyUserConfigModel              = Key{string: "允许用户配置模型"}
	MsgPolicyUserConfigChannel            = Key{string: "允许用户配置通道"}
	MsgPolicyCustomModel                  = Key{string: "允许用户添加自定义模型"}
	MsgPolicyAgentTerminal                = Key{string: "允许用户进入 Agent 终端"}
	MsgPolicyGatewayUI                    = Key{string: "允许用户访问 Agent 面板"}
	MsgPolicyChatView                     = Key{string: "允许用户使用对话视图"}
	MsgPolicyBrowserVNC                   = Key{string: "允许用户访问 Agent 云桌面"}
	MsgPolicyLobsterDoctor                = Key{string: "允许用户使用龙虾医生"}
	MsgPolicyModelQuota                   = Key{string: "允许用户查看模型额度"}
	MsgPolicySMHAutoProvision             = Key{string: "创建实例时自动开启网盘"}
)

// 标签 (tag.go)
var (
	MsgCreateTagClientFailed = Key{string: "创建 Tag 客户端失败"}
	MsgQueryTagKeysFailed    = Key{string: "查询标签键失败"}
	MsgQueryTagValuesFailed  = Key{string: "查询标签值失败"}
	MsgKeyParamRequired      = Key{string: "缺少 key 参数"}
)

// Agent 命令下发 (admin_agent_command_tasks.go)
var (
	MsgDispatchSlugWithExtraParams     = Key{string: "携带 dispatch_slug 时不可同时传 command_id / instance_ids 等字段"}
	MsgDispatchSlugRequired            = Key{string: "abort=true 必须配合 dispatch_slug 使用"}
	MsgCommandRequired                 = Key{string: "请选择要下发的命令"}
	MsgTargetsRequired                 = Key{string: "请至少选择 1 台 Agent"}
	MsgTooManyTargets                  = Key{string: "单次最多下发 %d 台，请减少选择"}
	MsgCommandNotFound                 = Key{string: "命令不存在或已被删除"}
	MsgCommandFailed                   = Key{string: "命令执行失败"}
	MsgSerializeScriptFailedWithDetail = Key{string: "序列化脚本失败: %s"}
	MsgInstanceNotFoundInTenant        = Key{string: "部分 Agent 不存在或不在当前租户：%v"}
	MsgTargetOffline                   = Key{string: "部分 Agent 未运行，无法接收命令：%s"}
	MsgAllTargetsOffline               = Key{string: "所有 Agent 均未运行，无法下发"}
	MsgTestTargetRequired              = Key{string: "请从所选 Agent 中指定一台作为测试机"}
	MsgTestTargetInvalid               = Key{string: "测试机必须从所选 Agent 中选择"}
	MsgDispatchNotFound                = Key{string: "dispatch_slug 不存在或不属于当前租户"}
	MsgDispatchPermissionDenied        = Key{string: "仅 dispatch 发起人或初始管理员可继续下发"}
	MsgDispatchAbortPermissionDenied   = Key{string: "仅 dispatch 发起人或初始管理员可终止下发"}
	MsgNothingToAbort                  = Key{string: "没有可终止的待发批次"}
	MsgDispatchNotFoundDetail          = Key{string: "该下发记录不存在或不在当前租户"}
	MsgDispatchSlugRequiredDetail      = Key{string: "缺少 dispatch_slug"}
	MsgAgentStatusNotAllowed           = Key{string: "Agent 状态不允许操作"}
)

// Agent 命令定时任务 (admin_agent_command_schedules.go)
var (
	MsgScheduleInvalidStatus     = Key{string: "非法状态筛选值"}
	MsgScheduleIDRequired        = Key{string: "缺少定时任务 id"}
	MsgScheduleIDQueryRequired   = Key{string: "缺少 schedule_id"}
	MsgScheduleNotFound          = Key{string: "定时任务不存在或已被删除"}
	MsgSchedulePermDeniedDelete  = Key{string: "仅创建者或超级管理员可删除该定时任务"}
	MsgSchedulePermDeniedOperate = Key{string: "仅创建者或超级管理员可操作该定时任务"}
	MsgScheduleCompleted         = Key{string: "任务已结束（一次性已执行或周期已到截止），无法启停"}
	MsgScheduleQuotaExceeded     = Key{string: "已达每租户 %d 个定时任务上限"}
	MsgScheduleExpiredCreate     = Key{string: "触发时刻已过期，请调整时间"}
	MsgScheduleExpiredToggle     = Key{string: "触发时刻已过期，请编辑后再启用"}
	MsgScheduleSpecInvalid       = Key{string: "定时任务规则无效"}
)

// Agent 命令下发 (model/agent_command_dispatch.go)
var (
	MsgDispatchRecordNotFound        = Key{string: "下发记录不存在"}
	MsgDispatchSlugConflictRetry     = Key{string: "下发标识冲突，请重试"}
	MsgDispatchFindBySlugFailed      = Key{string: "按 slug 查询下发记录失败"}
	MsgDispatchFindByIDFailed        = Key{string: "按 ID 查询下发记录失败"}
	MsgDispatchQueryInProgressFailed = Key{string: "查询进行中的下发记录失败"}
	MsgDispatchBatchStatsFailed      = Key{string: "批量查询命令执行统计失败"}
	MsgDispatchFindUnfinishedFailed  = Key{string: "查询未完成下发记录失败"}
)

// 分组配置绑定 (model/group_config_binding.go)
var (
	MsgGroupConfigDeleteOldBindingFailed     = Key{string: "删除旧绑定失败"}
	MsgGroupConfigCreateBindingFailed        = Key{string: "创建绑定失败"}
	MsgDeletePolicyBindingFailed             = Key{string: "删除策略绑定失败"}
	MsgGroupConfigQueryPolicyBindingFailed   = Key{string: "查询策略绑定失败"}
	MsgGroupConfigUpdatePolicyBindingFailed  = Key{string: "更新策略绑定失败"}
	MsgGroupConfigCreatePolicyBindingFailed  = Key{string: "创建策略绑定失败"}
	MsgGroupConfigBatchQueryVisibilityFailed = Key{string: "批量查询 %s 可见性关联失败"}
)

// 模型可见性 (model/model_visibility.go)
var (
	MsgVisibilityQueryModelAssocFailed      = Key{string: "查询模型可见性关联失败"}
	MsgVisibilityBatchQueryModelAssocFailed = Key{string: "批量查询模型可见性关联失败"}
	MsgVisibilityCleanupByGroupFailed       = Key{string: "清理分组模型可见性关联失败"}
	MsgVisibilityCleanupByModelFailed       = Key{string: "清理模型可见性关联失败"}
	MsgVisibilityCheckGroupUsedFailed       = Key{string: "检查用户组是否被模型可见性引用失败"}
	MsgVisibilityQueryModelsByGroupFailed   = Key{string: "查询用户组关联的模型列表失败"}
)

// 绑定 CRUD (controller/usergroup/binding_crud.go)
var (
	MsgBindingUnsupportedConfigType = Key{string: "不支持的 config_type: %s"}
	MsgBindingConfigTypeNotAdditive = Key{string: "config_type %s 不是加法型，不能设置 visibility"}
	MsgBindingUnsupportedPolicyKey  = Key{string: "不支持的策略配置项: %s"}
	MsgBindingQueryByResourceFailed = Key{string: "查询资源绑定失败"}
	MsgBindingInvalidGroupIDs       = Key{string: "存在不合法的用户组 ID"}
)

// 实例状态 (controller/instance_state.go)
var (
	MsgInstanceStateDestroyCVMFailed    = Key{string: "彻底销毁 CVM 实例失败"}
	MsgInstanceExternallyDestroyedTitle = Key{string: "实例已被销毁"}
	MsgInstanceExternallyDestroyed      = Key{string: "您的实例「%s」已被销毁，如有疑问请联系管理员。"}
)

// 命令调用 (model/agent_command_invocation.go)
var (
	MsgInvocationFindBySlugFailed = Key{string: "按 slug 查询命令调用记录失败"}
	MsgInvocationFindByIDFailed   = Key{string: "按 ID 查询命令调用记录失败"}
)

// 命令任务 (model/agent_command_task.go)
var (
	MsgTaskFindByInvocationIDFailed = Key{string: "按 invocation_id 查询命令任务失败"}
	MsgTaskFindBySlugFailed         = Key{string: "按 slug 查询命令任务失败"}
	MsgTaskFindUnfinishedFailed     = Key{string: "查询未完成命令任务失败"}
)

// Agent 命令下发控制 (controller/admin_agent_command_tasks.go)
var (
	MsgCmdTaskAlreadyCompleted           = Key{string: "dispatch 已是终态"}
	MsgCmdTaskNotAwaiting                = Key{string: "dispatch 未处于待确认状态"}
	MsgCmdTaskTestPhaseInProgress        = Key{string: "测试机仍在执行，请等待终态后再操作"}
	MsgCmdTaskTestPhaseFailed            = Key{string: "测试机执行失败，dispatch 已进入终态"}
	MsgCmdTaskAlreadyContinued           = Key{string: "已开始下发剩余批次，无法重复操作"}
	MsgCmdTaskPreCreateInvocationsFailed = Key{string: "预创建命令调用记录失败"}
	MsgCmdTaskCreateDispatchFailed       = Key{string: "创建下发记录失败"}
	MsgCmdTaskCreateTestInvocationFailed = Key{string: "创建测试机调用记录失败"}
	MsgCmdTaskCreateTestTaskFailed       = Key{string: "创建测试机任务失败"}
	MsgCmdTaskCreateProdInvocationFailed = Key{string: "创建生产批次调用记录 %d 失败"}
	MsgCmdTaskCreateProdTaskFailed       = Key{string: "创建生产批次任务失败"}
	MsgCmdTaskCheckDispatchSlugFailed    = Key{string: "检查下发标识唯一性失败"}
	MsgCmdTaskParamValueRequired         = Key{string: "缺少参数值：%s"}
	MsgCmdTaskParamUnknown               = Key{string: "提供了命令未声明的参数：%s"}
	MsgCmdFailedToCountActiveCmds        = Key{string: "统计当前租户活跃命令数失败"}
	MsgCmdTaskBuildSnapshotFailed        = Key{string: "构建快照失败"}
)

// 分类（插件 / 技能） (admin_plugin_categories.go, admin_skill_categories.go)
var (
	MsgQueryCategoryCountFailed       = Key{string: "查询分类总数失败"}
	MsgQueryCategoryListFailed        = Key{string: "查询分类列表失败"}
	MsgQueryCategoryPluginCountFailed = Key{string: "查询分类插件数量失败"}
	MsgQueryCategorySkillCountFailed  = Key{string: "查询分类技能数量失败"}
	MsgCategoryNameRequired           = Key{string: "分类名称不能为空"}
	MsgCategoryNameTooLong            = Key{string: "分类名称不能超过 100 个字符"}
	MsgQueryCategoryNameFailed        = Key{string: "查询分类名称失败"}
	MsgCategoryNameExists             = Key{string: "分类名称已存在"}
	MsgCreateCategoryFailed           = Key{string: "创建分类失败"}
	MsgCategoryIDMustBeNumber         = Key{string: "分类 ID 必须为数字"}
	MsgCategoryIDRequired             = Key{string: "缺少分类 ID"}
	MsgParseFormFailed                = Key{string: "解析表单失败"}
	MsgCategoryNotFound               = Key{string: "分类不存在"}
	MsgUpdateCategoryFailed           = Key{string: "更新分类失败"}
	MsgCleanupCategoryMappingFailed   = Key{string: "清理分类关联失败"}
	MsgDeleteCategoryFailed           = Key{string: "删除分类失败"}
)

var (
	MsgTATNoInvocationId            = Key{string: "未获取到执行 ID"}
	MsgTATInvocationTimeout         = Key{string: "命令执行超时"}
	MsgTATUnavailable               = Key{string: "拉取执行结果失败，请稍后重试"}
	MsgTATLoadScriptFailed          = Key{string: "加载命令失败"}
	MsgTATScriptWithError           = Key{string: "脚本 %s: %s"}
	MsgTATScriptIncludeExpandFailed = Key{string: "脚本 include 展开失败"}
	MsgTATScriptOutputAbnormal      = Key{string: "预检脚本输出无合法 JSON: %q"}
	MsgTATCommandDispatchFailed     = Key{string: "TAT 命令下发失败"}
	MsgTATClientCreateFailed        = Key{string: "创建 TAT 客户端失败: %s"}
	MsgTATQueryResultFailed         = Key{string: "查询执行结果失败"}
	MsgTATCommandStartFailed        = Key{string: "命令启动失败"}
	MsgTATWaitResultTimeout         = Key{string: "等待执行结果超时"}
	MsgTATExecuteCommandFailed      = Key{string: "执行命令失败"}
	MsgTATAgentNotReady             = Key{string: "TAT Agent 未就绪，实例 %s 不在线"}
	MsgTATSerializeParamsFailed     = Key{string: "序列化命令参数失败: %s"}
	MsgTATNoInvocationIdReturned    = Key{string: "TAT 未返回 InvocationId"}
)

// AI 通道 (admin_channels.go)
var (
	MsgChannelNotExist                = Key{string: "通道不存在"}
	MsgInvalidRequestFormat           = Key{string: "请求格式错误"}
	MsgChannelIDCannotBeEmpty         = Key{string: "Channel ID 不能为空"}
	MsgChannelIDInvalidChars          = Key{string: "Channel ID 仅允许英文字母、数字和下划线"}
	MsgChannelNameCannotBeEmpty       = Key{string: "通道名称不能为空"}
	MsgCustomChannelConfigRequired    = Key{string: "缺少自定义通道配置"}
	MsgCustomChannelConfigFormatError = Key{string: "自定义通道配置格式错误"}
	MsgIMServerConfigRequired         = Key{string: "缺少 IM 服务器配置"}
	MsgIMServerConfigMustBeJSON       = Key{string: "IM 服务器配置必须为 JSON 对象"}
	MsgCredKeyInvalidChars            = Key{string: "凭证字段 key 仅允许英文字母、数字和下划线"}
	MsgCredLabelRequired              = Key{string: "凭证字段 label 不能为空"}
	MsgCredKeyDuplicate               = Key{string: "凭证字段 key 重复"}
	MsgChannelIDExists                = Key{string: "Channel ID 已存在"}
	MsgCreateChannelFailed            = Key{string: "创建通道失败"}
	MsgPredefinedChannelCannotDelete  = Key{string: "预定义通道不允许删除"}
)

// AI 模型管理 (admin_models.go)
var (
	MsgQuotaDayInvalid                   = Key{string: "每日Token上限必须为 -1 或非负整数"}
	MsgModelRequiredFieldsCreate         = Key{string: "模型ID、API Key、URL 和接口类型不能为空"}
	MsgModelRequiredFieldsUpdate         = Key{string: "模型ID、URL 和接口类型不能为空"}
	MsgCreateModelFailed                 = Key{string: "创建失败"}
	MsgDeleteModelFailed                 = Key{string: "删除失败"}
	MsgAdminModelNotFound                = Key{string: "模型不存在"}
	MsgCleanupInstanceModelBindingFailed = Key{string: "清理实例模型绑定失败"}
	MsgRequestBodyShouldBeJSON           = Key{string: "请求体格式错误，应为 JSON"}
	MsgBuiltinModelCannotModify          = Key{string: "系统内置记录不可修改"}
	MsgBuiltinModelCannotDelete          = Key{string: "系统内置记录不可删除"}
	MsgEnableModelBeforeDefault          = Key{string: "请先开启该模型的「用户可见」后再设为默认"}
	MsgEnableEnabledBeforeDefault        = Key{string: "请先开启该模型的「启用」后再设为默认"}
	MsgInvalidVisibilityForModel         = Key{string: "visibility_type 必须为 all 或 group"}
	MsgGroupRequiredForVisibility        = Key{string: "按分组可见时必须选择至少一个分组"}
	MsgQueryGroupFailed                  = Key{string: "查询分组失败"}
	MsgGroupNotFoundList                 = Key{string: "分组不存在: %v"}
	MsgUpdateVisibilityFailed            = Key{string: "更新可见范围失败"}
	MsgMaxTokensMustBeInteger            = Key{string: "最大输出 Token 数必须为整数"}
	MsgMaxTokensMustBeNonNegative        = Key{string: "最大输出 Token 数必须为非负整数"}
	MsgModelMarshalCustomHeadersFailed   = Key{string: "序列化自定义 HTTP 头部失败"}
)

// 用户组管理 (admin_user_groups.go)
var (
	MsgQueryGroupInstanceTotalFailed = Key{string: "查询分组实例总数失败"}
	MsgGroupMoveInProgress           = Key{string: "另一个分组移动正在进行，请稍后重试"}
	MsgGroupOperationInProgress      = Key{string: "另一个分组操作正在进行，请稍后重试"}
	MsgGroupMemberLimitExceeded      = Key{string: "成员数量超过上限（10000 人）"}
	MsgGroupReadonlyMembers          = Key{string: "该分组为只读，不允许变更成员"}
	MsgUserIDInvalidFormat           = Key{string: "user_ids 中存在格式错误的 ID: %s"}
	MsgUserIDsTooMany                = Key{string: "user_ids 最多支持 100 个"}
	MsgQueryAssociatedModelsFailed   = Key{string: "查询关联模型失败"}
	MsgQueryModelDetailFailed        = Key{string: "查询模型详情失败"}
	MsgGroupDeletedButCLSCleanupFail = Key{string: "分组已删除，但 CLS 采集范围清理失败，请手动检查"}
	MsgParentIDFormatError           = Key{string: "parent_id 格式错误"}
	MsgOneIDCreateDeptFailed         = Key{string: "OneID 创建部门失败: %v"}
	MsgOneIDUpdateDeptFailed         = Key{string: "OneID 更新部门失败: %v"}
	MsgOneIDDeleteDeptFailed         = Key{string: "OneID 删除部门失败: %v"}
	MsgOneIDSyncRemoveUserDeptFailed = Key{string: "OneID 同步移除用户部门失败: %v"}
)

// 项目管理 (admin_projects.go)
var (
	MsgProjectInvalidName                = Key{string: "项目名称不能为空或超过长度限制"}
	MsgProjectNameConflict               = Key{string: "项目名称已存在"}
	MsgProjectNotFound                   = Key{string: "项目不存在"}
	MsgProjectHasDependencies            = Key{string: "项目仍有关联工具应用范围，无法删除"}
	MsgProjectConfigAgentToolDescription = Key{string: "企业技能与企业规范"}
	MsgProjectConfigEnterpriseRule       = Key{string: "企业规范"}
)

// Agent 类型 (admin_agent_types.go)
var (
	MsgQueryImageStatsFailed      = Key{string: "查询镜像统计失败"}
	MsgQueryInstanceCountFailed   = Key{string: "查询实例数量失败"}
	MsgAgentTypeCannotBeEmpty     = Key{string: "agent_type 不能为空"}
	MsgEnabledAndToggleMutex      = Key{string: "enabled 和 toggle 不能同时传"}
	MsgEnabledOrToggleRequired    = Key{string: "必须传 enabled 或 toggle"}
	MsgToggleMustBeTrue           = Key{string: "toggle 必须为 true"}
	MsgNameCannotBeEmpty          = Key{string: "name 不能为空"}
	MsgBuiltinAgentTypeCannotDel  = Key{string: "内置智能体类型不能删除"}
	MsgCustomAgentTypeNotFound    = Key{string: "自定义智能体类型不存在: %s"}
	MsgQueryUserGroupFailed       = Key{string: "查询用户分组失败"}
	MsgResolveImageVisibilityFail = Key{string: "解析镜像类型可见性失败"}
)

// 自定义智能体类型 (custom_agent_type.go)
var (
	MsgCustomAgentTypeNameRequired        = Key{string: "名称不能为空"}
	MsgCustomAgentTypeNameHasTrimSpace    = Key{string: "名称不能包含首尾空格"}
	MsgCustomAgentTypeNameTooLong         = Key{string: "名称不能超过 %d 个字符"}
	MsgCustomAgentTypeNameConflictBuiltin = Key{string: "名称不能与内置智能体类型重复: %s"}
	MsgCustomAgentTypeCompatibleInvalid   = Key{string: "兼容类型必须是内置智能体类型"}
	MsgCustomAgentTypeAlreadyExists       = Key{string: "自定义智能体类型已存在: %s"}
	MsgCustomAgentTypeIsDefaultDelete     = Key{string: "该类型是用户端首选类型，不能删除"}
	MsgCustomAgentTypeDisableBeforeDelete = Key{string: "请先禁用该智能体类型或取消启用该类型下的镜像后再删除"}
	MsgCustomAgentTypeHasInstances        = Key{string: "该类型下存在实例，不能删除"}
	MsgCustomAgentTypeCreateDBFailed      = Key{string: "创建自定义智能体类型失败"}
	MsgCustomAgentTypeQueryDBFailed       = Key{string: "查询自定义智能体类型失败"}
	MsgCustomAgentTypeDeleteDBFailed      = Key{string: "删除自定义智能体类型失败"}
)

// 站点配置 / 公网配置 (site_config.go)
var (
	MsgInternetChargeTypeRequired        = Key{string: "分配公网 IP 时必须指定带宽计费模式"}
	MsgInternetChargeTypeUnsupported     = Key{string: "不支持的带宽计费模式: %s，合法值: BANDWIDTH_PREPAID, TRAFFIC_POSTPAID_BY_HOUR, BANDWIDTH_POSTPAID_BY_HOUR, BANDWIDTH_PACKAGE"}
	MsgBandwidthPrepaidRequiresPrepaid   = Key{string: "包月带宽(BANDWIDTH_PREPAID)仅可用于预付费(PREPAID)实例，当前实例计费模式为: %s"}
	MsgPrepaidBandwidthOutOfRange        = Key{string: "包月带宽(BANDWIDTH_PREPAID)模式下，带宽上限范围为 1-20 Mbps，当前值: %d"}
	MsgBandwidthOutOfRange               = Key{string: "当前计费模式(%s)下，带宽上限范围为 1-2000 Mbps，当前值: %d"}
	MsgTrafficBandwidthOutOfRange        = Key{string: "按流量计费(TRAFFIC_POSTPAID_BY_HOUR)模式下，带宽上限范围为 1-200 Mbps，当前值: %d"}
	MsgParseCVMTemplateFailed            = Key{string: "解析 CVM 模板失败"}
	MsgDiskTypeUnsupported               = Key{string: "不支持的系统盘类型: %s，允许的类型: %s"}
	MsgWrongDiskType                     = Key{string: "磁盘类型错误"}
	MsgSystemDiskTooSmall                = Key{string: "系统盘大小不能小于 %dGB，当前值: %dGB"}
	MsgInvalidAgentType                  = Key{string: "无效的智能体类型: %s"}
	MsgAgentTypeIsDefaultCannotDisable   = Key{string: "该类型是用户端首选，不可禁用"}
	MsgSerializeDisabledAgentTypesFailed = Key{string: "序列化禁用智能体类型列表失败"}
)

// 版本信息拉取 (version_fetcher.go)
var (
	MsgVersionFetcherInvalidDetectToken = Key{string: "版本探测输出无效: %q"}
	MsgVersionFetcherEmptyDetectOutput  = Key{string: "版本探测输出为空"}
	MsgVersionFetcherParseJSONFailed    = Key{string: "解析版本信息 JSON 失败"}
	MsgVersionFetcherWriteDBFailed      = Key{string: "写入版本信息失败"}
)

// 脚本注册与解析 (script_registry.go)
var (
	MsgScriptResolveFailed             = Key{string: "脚本解析失败"}
	MsgFeatureNotSupportedForAgentType = Key{string: "功能 %s 不支持 agent_type %s"}
	MsgUnknownFeature                  = Key{string: "未知的功能: %s"}
	MsgInstanceIsNil                   = Key{string: "instance 为空"}
	MsgScriptResolveFailedWrap         = Key{string: "脚本解析失败: feature=%s, agent_type=%s"}
	MsgScriptRunFailed                 = Key{string: "脚本执行失败"}
	MsgScriptRunFailedWrap             = Key{string: "脚本执行失败: script=%s"}
	MsgIncludeDepthExceeded            = Key{string: "include 嵌套深度超限（可能存在循环引用）"}
	MsgInvalidIncludeName              = Key{string: "无效的 include 文件名: %s（必须匹配 lib_*.sh）"}
	MsgIncludeLoadFailed               = Key{string: "加载 include %s 失败"}
)

// 子网选择 (subnet_picker.go)
var (
	MsgVpcSubnetMismatch      = Key{string: "vpc和子网不匹配"}
	MsgZoneNoSubnetConfigured = Key{string: "可用区 %s 未配置任何子网"}
	MsgVpcOrSubnetNotExist    = Key{string: "VPC 或子网不存在（可用区 %s，子网 %s）"}
	MsgQuerySubnetFailed      = Key{string: "查询子网失败"}
	MsgCandidateSubnetEmpty   = Key{string: "候选子网列表为空"}
	MsgQuerySubnetIPFailed    = Key{string: "查询子网可用 IP 数失败"}
	MsgSubnetIPExhausted      = Key{string: "所选可用区子网 IP 已满，请联系管理员增加子网或扩容"}
)

// TAT 批量操作 (tat_batch.go)
var (
	MsgTATBatchTooMany             = Key{string: "TAT 批量操作超出单次调用上限"}
	MsgTATInstanceIdsEmpty         = Key{string: "instanceIds 不能为空"}
	MsgTATBatchTooManyDetail       = Key{string: "TAT 批量操作超出上限: 传入 %d 个，最大 %d 个"}
	MsgTATDescribeBindingFailed    = Key{string: "查询 TAT invocation task binding 失败"}
	MsgTATInvocationTaskNotVisible = Key{string: "TAT InvocationTask 在 %d 次重试后仍未完全可见（预期 %d 个）"}
	MsgTATFailed                   = Key{string: "TAT 操作失败"}
	MsgTATTimeout                  = Key{string: "TAT 操作超时"}
	MsgTATSyncFailedFor            = Key{string: "TAT 同步失败: %s"}
)

// STS 凭证 (sts.go)
var (
	MsgSTSExpiredTimeEmpty    = Key{string: "AssumeRole 返回的过期时间为空"}
	MsgSTSCredentialNotConfig = Key{string: "凭据未配置"}
	MsgSTSRefreshFailed       = Key{string: "STS 临时密钥刷新失败"}
	MsgSTSInvalidInstanceId   = Key{string: "无效的 instanceId 格式: %s"}
	MsgSTSSecretNotConfigured = Key{string: "CVMSecretId/CVMSecretKey 未配置，无法申请 STS 临时密钥"}
	MsgSTSCreateClientFailed  = Key{string: "创建 STS 客户端失败"}
	MsgSTSAssumeRoleFailed    = Key{string: "AssumeRole 失败"}
	MsgSTSCredentialEmpty     = Key{string: "AssumeRole 返回的凭证为空"}
	MsgSTSServiceDown         = Key{string: "STS 服务异常"}
)

// CLS 内部 helper (admin_cls.go)
var (
	MsgAgentCamRoleSecretNotConfigured = Key{string: "AgentCamRoleSecretId/AgentCamRoleSecretKey 未配置"}
	MsgSetClsServiceParamsFailed       = Key{string: "设置 GetClsService 请求参数失败"}
	MsgQueryClsServiceCallFailed       = Key{string: "查询 CLS 服务状态调用失败"}
	MsgParseClsServiceRespFailed       = Key{string: "解析 GetClsService 响应失败"}
	MsgClsServiceStatusError           = Key{string: "CLS 服务状态查询失败 (requestId=%s): %s - %s"}
	MsgClsServiceStatusEmpty           = Key{string: "GetClsService 返回数据异常: Status 为空"}
	MsgSerializeOpenClawParamsFailed   = Key{string: "序列化 OpenClawService 请求参数失败"}
	MsgSetOpenClawParamsFailed         = Key{string: "设置 OpenClawService 请求参数失败"}
	MsgCallOpenClawServiceFailed       = Key{string: "调用 OpenClawService 失败"}
	MsgParseOpenClawRespFailed         = Key{string: "解析 OpenClawService 响应失败"}
	MsgOpenClawServiceBizError         = Key{string: "OpenClawService 失败 (requestId=%s): %s - %s"}
)

// SMH 上传 (smh_upload.go)
var (
	MsgSMHUploadCredFailed = Key{string: "获取 SMH 上传凭证失败"}
	MsgSMHUploadPartFailed = Key{string: "上传分块 %d/%d 失败"}
	MsgSMHUploadFailed     = Key{string: "SMH 上传失败"}
)

// API 网关 (webui_apigateway.go)
var (
	MsgAPIGatewayGetCredFailed       = Key{string: "获取凭据失败"}
	MsgAPIGatewayMarshalParamsFailed = Key{string: "序列化 API 网关请求参数失败"}
	MsgAPIGatewaySetParamsFailed     = Key{string: "设置 API 网关请求参数失败"}
	MsgAPIGatewayCanceled            = Key{string: "API 网关 %s 请求取消"}
	MsgAPIGatewaySendFailed          = Key{string: "API 网关 %s 请求失败"}
	MsgAPIGatewayParseRespFailed     = Key{string: "API 网关 %s 响应解析失败 (body=%s)"}
	MsgAPIGatewayBizError            = Key{string: "API 网关 %s 错误: %s - %s (RequestId=%s)"}
	MsgCreateTATClientFailed         = Key{string: "创建 TAT 客户端失败"}
	MsgQueryTATAgentStatusFailed     = Key{string: "查询 TAT Agent 状态失败"}
)

// 用户组 VPC (admin_group_vpc.go)
var (
	MsgQueryVpcConfigFailed     = Key{string: "查询 VPC 配置失败"}
	MsgSubnetIdsParseFailed     = Key{string: "subnet_ids 解析失败"}
	MsgCreateVpcConfigFailed    = Key{string: "创建 VPC 配置失败"}
	MsgCreateGroupBindingFailed = Key{string: "创建分组绑定失败"}
	MsgIDCannotBeEmpty          = Key{string: "id 不能为空"}
	MsgVpcConfigNotFound        = Key{string: "VPC 配置不存在"}
	MsgUpdateVpcConfigFailed    = Key{string: "更新 VPC 配置失败"}
	MsgUpdateGroupBindingFailed = Key{string: "更新分组绑定失败"}
	MsgDeleteVpcConfigFailed    = Key{string: "删除 VPC 配置失败"}
	MsgDeleteGroupBindingFailed = Key{string: "删除分组绑定失败"}

	// VPC 校验
	MsgVpcIdCannotBeEmpty      = Key{string: "vpc_id 不能为空"}
	MsgSubnetIdsCannotBeEmpty  = Key{string: "subnet_ids 不能为空"}
	MsgGroupRequiredForVpc     = Key{string: "必须至少选择一个分组"}
	MsgStrategyNameTooLong     = Key{string: "strategy_name 不能超过 20 个字符"}
	MsgSubnetIdsInvalidJSON    = Key{string: "subnet_ids 必须是合法的 JSON"}
	MsgGroupAlreadyBoundVpc    = Key{string: "当前选择分组已配置 VPC，不支持重复配置网络，请调整。"}
	MsgQueryGroupBindingFailed = Key{string: "查询分组绑定失败"}
	MsgVerifyVpcFailed         = Key{string: "校验 VPC 失败"}
	MsgVpcNotExist             = Key{string: "指定的 VPC 不存在: %s"}
	MsgSubnetRequired          = Key{string: "至少需要配置一个子网"}
	MsgVerifySubnetFailed      = Key{string: "校验子网失败"}
	MsgSubnetNotExist          = Key{string: "子网不存在: %s"}
	MsgSubnetNotBelongVpc      = Key{string: "子网 %s 不属于 VPC %s"}
	MsgSubnetZoneMismatch      = Key{string: "子网 %s 所在可用区为 %s，与配置的可用区 %s 不匹配"}
	MsgSubnetVerifyRespEmpty   = Key{string: "校验子网失败：响应为空"}
)

// 资源策略 (admin_resource_policy.go)
var (
	MsgResourcePolicyNotFound         = Key{string: "资源策略不存在"}
	MsgResourcePolicyNameConflict     = Key{string: "资源策略名称已存在"}
	MsgResourcePolicyGroupOccupied    = Key{string: "所选分组已应用其他资源策略"}
	MsgDefaultResourcePolicyProtected = Key{string: "企业默认资源策略受保护"}
	MsgDefaultResourcePolicyName      = Key{string: "企业默认资源策略"}
)

// 技能包 (admin_skill_bundle.go)
var (
	MsgSkillBundleNameRequired           = Key{string: "技能包名称不能为空"}
	MsgSkillBundleNotFound               = Key{string: "技能包不存在"}
	MsgSkillBundleConflict               = Key{string: "同名技能包已存在"}
	MsgSkillBundleEnabledNeedsOff        = Key{string: "技能包正在生效中，需先禁用"}
	MsgDeleteSkillBundleFailed           = Key{string: "删除技能包失败"}
	MsgAllVisibilityBundleConflict       = Key{string: "已有其他应用范围为「全部用户」的技能包处于启用状态，请先禁用"}
	MsgAddSourceCannotBeEmpty            = Key{string: "add[].source 不能为空"}
	MsgEnterpriseSkillNotFound           = Key{string: "企业技能 ID=%d 不存在"}
	MsgGenEnterpriseDownloadURLFail      = Key{string: "生成企业技能下载 URL 失败"}
	MsgDownloadEnterpriseZipFail         = Key{string: "下载企业技能 zip 失败"}
	MsgReadEnterpriseZipFail             = Key{string: "读取企业技能 zip 失败 (status=%d)"}
	MsgUploadSkillZipFail                = Key{string: "上传技能 zip 到 common space 失败"}
	MsgPublicSkillNotFound               = Key{string: "公共技能 ID=%d 不存在"}
	MsgDownloadPublicZipFail             = Key{string: "下载公共技能 zip 失败"}
	MsgReadPublicZipFail                 = Key{string: "读取公共技能 zip 失败 (status=%d)"}
	MsgUnsupportedSkillSource            = Key{string: "不支持的来源类型: %s"}
	MsgGetCommonStorageClientFail        = Key{string: "获取 common space 存储客户端失败"}
	MsgFavoriteSkillFailed               = Key{string: "收藏技能失败"}
	MsgNameSlugCannotBeEmpty             = Key{string: "name 和 slug 不能为空"}
	MsgVisibilityTypeCannotBeEmpty       = Key{string: "visibility_type 不能为空"}
	MsgCreateSkillBundleFailed           = Key{string: "创建技能包失败"}
	MsgToggleSkillBundleFailed           = Key{string: "切换技能包状态失败"}
	MsgUpdateSkillBundleSkillsFailed     = Key{string: "更新技能包技能失败"}
	MsgUpdateSkillBundleVisibilityFailed = Key{string: "更新技能包可见范围失败"}
	MsgSkillVersionConflictInBundle      = Key{string: "技能 %s-%s 已存在于该技能包中"}
	MsgSBInsertToDBFailed                = Key{string: "写入技能包记录失败"}
	MsgSBCreateSMHDirFailed              = Key{string: "创建 SMH 目录失败"}
	MsgSBDisableBundleFailed             = Key{string: "禁用技能包失败"}
	MsgSBReadDefaultBundleFailed         = Key{string: "读取默认技能包失败"}

	// model/skill_bundle_visibility.go
	MsgSBVisQueryAssocFailed       = Key{string: "查询技能包可见性关联失败"}
	MsgSBVisBatchQueryAssocFailed  = Key{string: "批量查询技能包可见性关联失败"}
	MsgSBVisDeleteOldAssocFailed   = Key{string: "删除旧技能包可见性关联失败"}
	MsgSBVisCreateAssocFailed      = Key{string: "创建技能包可见性关联失败"}
	MsgSBVisUpdateVisibilityFailed = Key{string: "更新技能包 visibility_type 失败"}
)

// model/skill_bundle.go
var (
	MsgSBReadSiteConfigFailed      = Key{string: "SeedDefaultSkillBundle 读 SiteConfig 失败"}
	MsgSBSetDefaultSeededFailed    = Key{string: "设置 DefaultBundleSeeded 标记失败"}
	MsgSBCreateDefaultBundleFailed = Key{string: "创建默认技能包失败"}
	MsgSBCreateDefaultSkillFailed  = Key{string: "创建默认技能包技能失败 slug=%s"}
	MsgSBUpdateDefaultSeededFailed = Key{string: "更新 DefaultBundleSeeded 标记失败"}
)

// SG 池管理 (controller/sg_pool.go)
var (
	MsgSGPoolSelectActiveFailed      = Key{string: "查询可用 SG 失败"}
	MsgSGPoolAutoScaleFailed         = Key{string: "自动扩容 SG 失败"}
	MsgSGPoolSelectFallbackFailed    = Key{string: "查询 fallback SG 失败"}
	MsgSGPoolAcquireUpdateLockFailed = Key{string: "获取规则更新锁失败"}
	MsgSGPoolAcquireScaleLockFailed  = Key{string: "获取扩容锁失败: %s"}
	MsgSGPoolCountPoolFailed         = Key{string: "统计池规模失败"}
	MsgSGPoolReloadRuleSetFailed     = Key{string: "重新加载 RuleSet 失败"}
	MsgSGPoolComputeOrdinalFailed    = Key{string: "计算序号失败"}
	MsgSGPoolCreateCloudSGFailed     = Key{string: "创建云端 SG 失败"}
	MsgSGPoolApplyRulesFailed        = Key{string: "应用规则到 SG %s 失败"}
	MsgSGPoolInsertDBFailed          = Key{string: "写入 managed_sg_pool 失败"}
	MsgSGPoolCreateSGEmptyResp       = Key{string: "创建 SG 返回空"}
	MsgSGPoolParseRulesJSONFailed    = Key{string: "解析规则 JSON 失败"}
	MsgSGPoolDescribeForClearFailed  = Key{string: "查询 SG 规则失败"}
	MsgSGPoolDeleteIngressFailed     = Key{string: "删除入站规则失败"}
	MsgSGPoolDeleteEgressFailed      = Key{string: "删除出站规则失败"}
	MsgSGPoolBaseNotConfigured       = Key{string: "安全组未初始化"}
	MsgSGPoolPoolExhausted           = Key{string: "SG 池已耗尽"}
)

// SG Ruleset 辅助 (controller/sg_ruleset_helpers.go)
var (
	MsgSGRulesetInstanceNil          = Key{string: "实例为 nil"}
	MsgSGRulesetInstanceNoSG         = Key{string: "实例 %s 无 SecurityGroupId"}
	MsgSGRulesetSGNotInPool          = Key{string: "SG %q 不在托管池中"}
	MsgSGRulesetLookupPoolBySG       = Key{string: "按 SG %q 查询 managed_sg_pool 失败"}
	MsgSGRulesetLegacyRuleSetID      = Key{string: "managed_sg_pool 行 SG %q 的 rule_set_id=0（旧数据）"}
	MsgSGRulesetCountRuleSetsFailed  = Key{string: "统计 RuleSet 数量失败"}
	MsgSGRulesetLookupPoolFailed     = Key{string: "查询 managed_sg_pool 失败"}
	MsgSGRulesetLookupRuleSetFailed  = Key{string: "查询 RuleSet %d 失败"}
	MsgSGRulesetListActiveByIDFailed = Key{string: "按 RuleSet %d 查询 SG 列表失败"}
	MsgSGRulesetListAllActiveFailed  = Key{string: "查询所有 SG 列表失败"}
	MsgSGRulesetLoadRuleSetFailed    = Key{string: "加载 RuleSet %d 失败"}
	MsgSGRulesetFanOutFailed         = Key{string: "扇出规则到 RuleSet %d 失败"}
	MsgSGRulesetFanOutPartialDrift   = Key{string: "扇出部分偏离 RuleSet %d（%d 个错误）"}
	MsgSGRulesetListRuleSetsFailed   = Key{string: "查询 RuleSet 列表失败"}
	MsgSGRulesetEnsureRuleFailed     = Key{string: "确保 RuleSet %d 规则失败"}
	MsgSGRulesetRefreshSGDrifted     = Key{string: "RuleSet %d: %d 个 SG 偏离"}
	MsgSGRulesetRefreshFailed        = Key{string: "刷新必需规则失败: %d/%d 个 RuleSet 失败"}
	MsgSGRulesetCountActiveSGFailed  = Key{string: "统计活跃 SG 数量失败"}
)

// SG Ruleset 初始化 (controller/sg_ruleset_init.go)
var (
	MsgSGRulesetInitMarshalRules  = Key{string: "序列化合并规则失败"}
	MsgSGRulesetInitInsertRuleSet = Key{string: "写入 rule_set 失败"}
	MsgSGRulesetInitCleanupFrozen = Key{string: "清理残留 FROZEN 记录失败"}
	MsgSGRulesetInitInsertFrozen  = Key{string: "写入 FROZEN 旧基础 SG 失败"}
	MsgSGRulesetInitInsertActive  = Key{string: "写入 ACTIVE 新 SG 失败"}
	MsgSGRulesetInitDBTxFailed    = Key{string: "DB 事务失败"}
)

// model/openclaw_role.go
var (
	MsgRoleSeedReadSiteConfig    = Key{string: "SeedDefaultRoles 读 SiteConfig 失败"}
	MsgRoleSeedCountRoles        = Key{string: "SeedDefaultRoles 计数角色失败"}
	MsgRoleSeedSetSeededIfExists = Key{string: "补设 DefaultRolesSeeded 标记失败"}
	MsgRoleSeedCreateRole        = Key{string: "创建预置角色失败 name=%s"}
	MsgRoleSeedCreateSkill       = Key{string: "创建预置角色技能失败 role=%s skill=%s"}
	MsgRoleSeedUpdateSeededFlag  = Key{string: "更新 DefaultRolesSeeded 标记失败"}
)

// model/plugin_bundle.go
var (
	MsgPluginSeedReadSiteConfig    = Key{string: "SeedDefaultPluginBundle 读 SiteConfig 失败"}
	MsgPluginSeedSetSeededIfExists = Key{string: "补设 DefaultPluginBundleSeeded 标记失败"}
	MsgPluginSeedCreateBundle      = Key{string: "创建默认插件包失败"}
	MsgPluginSeedUpdateSeededFlag  = Key{string: "更新 DefaultPluginBundleSeeded 标记失败"}
)

// OneID 部门组落地 (controller/usergroup/oneid_dept_landing.go)
var (
	MsgOneIDDeptLandingReadDepts   = Key{string: "LandOneIDDepartmentsToGroups: 读取 oneid_departments 失败"}
	MsgOneIDDeptLandingReadLocal   = Key{string: "LandOneIDDepartmentsToGroups: 读取本地 oneid_dept 组失败"}
	MsgOneIDDeptSyncReadProfiles   = Key{string: "SyncOneIDMemberships: 读 profiles 失败"}
	MsgOneIDDeptSyncReadUsers      = Key{string: "SyncOneIDMemberships: 读 users 失败"}
	MsgOneIDDeptSyncReadDeptGroups = Key{string: "SyncOneIDMemberships: 读 oneid_dept 组失败"}
	MsgOneIDDeptSyncReadMembers    = Key{string: "SyncOneIDMemberships: 读当前 oneid_dept 成员失败"}
	MsgOneIDDeptDeleteStaleMember  = Key{string: "删除失效 oneid_dept 成员失败"}
	MsgOneIDDeptInsertMember       = Key{string: "插入 oneid_dept 成员失败 (gid=%d,uid=%d)"}
	MsgOneIDDeptUpdateIsMain       = Key{string: "更新 oneid_dept 成员 is_main 失败 (id=%d)"}
	MsgOneIDDeptQueryMultiMain     = Key{string: "查询多主部门用户失败"}
	MsgOneIDDeptClearExtraMain     = Key{string: "清多余主部门失败 (keep=%v)"}
	MsgOneIDDeptLandingSkipped     = Key{string: "OneID 部门落地未执行"}
)

// controller/instance_operation.go
var (
	MsgInstanceCannotDeleteLoading  = Key{string: "实例加载中，无法删除，请等待操作完成后再试"}
	MsgInstanceCannotDeleteDisabled = Key{string: "实例已停用，无法删除，请联系管理员处理"}
	MsgInstanceCannotDeleteCreating = Key{string: "实例创建中，无法删除"}
)

// Provider 通用错误 (controller/provider/openai.go, anthropic.go)
var (
	MsgProviderCreateRequest   = Key{string: "创建请求失败"}
	MsgProviderUpstreamRequest = Key{string: "上游请求失败"}
	MsgProviderReadResponse    = Key{string: "读取响应失败"}
	MsgProviderParseRequest    = Key{string: "解析请求失败"}
	MsgProviderMarshalRequest  = Key{string: "序列化请求失败"}
	MsgProviderParseOpenAIReq  = Key{string: "解析 OpenAI 请求失败"}
)

// controller/ruleset_helpers.go
var (
	MsgRulesetCIDRFormatHint  = Key{string: "请明确写为 %s/0（表示所有地址）或 %s/32 / %s/128（表示单个地址）"}
	MsgRulesetInvalidIPOrCIDR = Key{string: "不是合法的 IP 或 CIDR"}
	MsgRulesetInvalidCIDR     = Key{string: "不是合法的 CIDR（%v）"}
)

// model 包种子数据和查询错误
var (
	MsgInstanceQueryStatsFailed   = Key{string: "查询实例统计失败"}
	MsgInstanceCleanupPlaceholder = Key{string: "清理残留占位记录失败"}
	MsgMcpQueryVersionListFailed  = Key{string: "查询版本列表失败"}
	MsgMigrateBulkInsert          = Key{string: "批量写入 batch %d 失败"}
	MsgMigrateOpenSQLite          = Key{string: "打开 sqlite %s 失败"}
	MsgPluginCatSeedQuery         = Key{string: "SeedPluginCategories 查询已有分类失败"}
	MsgPluginCatSeedInsert        = Key{string: "SeedPluginCategories 批量写入失败"}
	MsgSiteConfigListTenants      = Key{string: "查询租户列表失败"}
	MsgSiteConfigMarshalSubnet    = Key{string: "序列化 SubnetMap 失败"}
	MsgSiteConfigMarshalDefSubnet = Key{string: "序列化 DefaultSubnetMap 失败"}
	MsgTagKeyEmpty                = Key{string: "tag key 不能为空"}
	MsgTagGroupRequired           = Key{string: "按分组应用时必须选择至少一个分组"}
	MsgSkillCatSeedQuery          = Key{string: "SeedCategories 查询已有分类失败"}
	MsgSkillCatSeedInsert         = Key{string: "SeedCategories 批量写入失败"}
)

// 记忆计划 (memory_plan.go)
var (
	MsgInstanceIDRequired           = Key{string: "instance_id 为必填字段"}
	MsgInstanceCVMNotFound          = Key{string: "实例 %s 未找到"}
	MsgInstanceDBNotFound           = Key{string: "实例 id=%d 未找到"}
	MsgNoAccessToInstance           = Key{string: "无权访问该实例"}
	MsgOnlyPostMethod               = Key{string: "仅支持 POST 方法"}
	MsgInvalidTargetPlan            = Key{string: "target_plan 必须是 off/free/pro，得到 %q"}
	MsgMemorySwitchInProgress       = Key{string: "实例 %s 有进行中的切换操作（%s），请等待完成后再试"}
	MsgProToFreeNotSupported        = Key{string: "当前为 Pro 模式，不支持直接切换到 Free，请先切到 OFF"}
	MsgSubmitJobFailed              = Key{string: "提交任务失败"}
	MsgInstanceNoMemoryConfig       = Key{string: "实例 %s 未找到记忆配置"}
	MsgTaskIDRequired               = Key{string: "task_id 参数为必填"}
	MsgTaskIDMustBeNumber           = Key{string: "task_id 必须为合法的数字"}
	MsgTaskNotFound                 = Key{string: "任务 %d 不存在"}
	MsgInstanceForTaskNotFound      = Key{string: "任务关联的实例不存在"}
	MsgNoAccessToTask               = Key{string: "无权访问该任务"}
	MsgInstanceIDParamRequired      = Key{string: "instance_id 参数为必填"}
	MsgTypeParamRequired            = Key{string: "type 参数为必填（persona/scene/memory/conversation）"}
	MsgInvalidMemoryType            = Key{string: "type 必须是 persona/scene/memory/conversation，得到 %q"}
	MsgInvalidMemorySubType         = Key{string: "sub_type 必须是 persona/episodic/instruction，得到 %q"}
	MsgMemoryServiceNotEnabled      = Key{string: "实例未开通记忆服务（当前为 OFF）"}
	MsgProMemoryNotAllocated        = Key{string: "Pro 记忆库未分配"}
	MsgProMemReleaseFailedWaitRetry = Key{string: "远端 Pro 记忆库释放失败，已保留实例记录等待自动补偿，请稍后重试"}
	MsgInitMemorySDKFailed          = Key{string: "初始化 SDK 失败"}
	MsgQueryMemoryRecordFailed      = Key{string: "查询记忆库记录失败"}
	MsgMemoryPluginNotReady         = Key{string: "记忆插件未就绪，请安装最新版记忆插件"}
	MsgMemoryPluginPathFailed       = Key{string: "插件路径探测失败"}
	MsgReadLocalMemoryFailed        = Key{string: "读取本地记忆数据失败"}
	MsgLocalMemoryDataTooLarge      = Key{string: "本地记忆数据量过大（超过 TAT 24KB 输出限制），请缩小查询范围或使用分页"}
	MsgParseLocalMemoryFailed       = Key{string: "解析本地记忆数据失败"}
	MsgReadSceneListFailed          = Key{string: "读取场景记忆列表失败"}
	MsgParseSceneListFailed         = Key{string: "解析场景记忆列表失败"}
	MsgMemoryPlanReadFailed         = Key{string: "读取失败"}
	MsgMemoryPlanContentTooLarge    = Key{string: "内容过大，暂不支持在线查看"}
	MsgMemoryPlanParseFailed        = Key{string: "解析失败"}
)

// 资源配置 - 镜像系统盘校验 (resource_config.go)
var (
	MsgImageSystemDiskTooSmall = Key{string: "当前资源配置中限制系统盘容量为 %dGiB，所选镜像容量要求为 %dGiB，请联系管理员调整。"}
)

// 站点配置 (admin_config.go)
var (
	MsgTemplateConfigSaved              = Key{string: "模板配置保存成功"}
	MsgGlobalQuotaInvalid               = Key{string: "全局配额必须为 -1 或非负整数"}
	MsgGlobalQuotaPeriodInvalid         = Key{string: "全局配额周期必须为 day 或 month"}
	MsgDefaultInstanceQuotaInvalid      = Key{string: "默认实例配额必须为 0~999 的整数（0 表示无配额）"}
	MsgDefaultTokenQuotaInvalid         = Key{string: "默认 Token 配额必须为 -1 或非负整数"}
	MsgDefaultLangInvalid               = Key{string: "default_lang 必须为 zh 或 en，收到: %s"}
	MsgDefaultTagsFormatInvalid         = Key{string: "default_tags 格式错误"}
	MsgMigrateDefaultTagsFailed         = Key{string: "迁移默认标签失败"}
	MsgTagNotFound                      = Key{string: "标签不存在"}
	MsgAPIGatewayConfigFormatInvalid    = Key{string: "api_gateway_config JSON 格式错误"}
	MsgAPIGatewayFieldsRequired         = Key{string: "启用 API 网关时 gateway_instance_id 和 base_domain 必填"}
	MsgSSOIMTypesFormatInvalid          = Key{string: "sso_im_types 格式错误，需为 JSON 数组"}
	MsgSSOIMTypeUnsupported             = Key{string: "sso_im_types 包含不支持的值: %s"}
	MsgGatewayUIAddrTypeInvalid         = Key{string: "gateway_ui_addr_type 仅支持 \"}private\" 或 \"public\""}
	MsgSaveGatewayUIFailed              = Key{string: "保存 Gateway UI 开关失败"}
	MsgQuerySGReadyFailed               = Key{string: "查询安全组就绪状态失败"}
	MsgPleaseInitSGFirst                = Key{string: "请先完成 ClawPro 安全组初始化并确保有 ACTIVE 安全组后再开启该功能"}
	MsgSaveBrowserVNCFailed             = Key{string: "保存云端浏览器开关失败"}
	MsgLogoTypeUnsupported              = Key{string: "仅支持 PNG、JPEG、SVG 格式的图片"}
	MsgLogoTooLarge                     = Key{string: "Logo 文件不能超过 512KB"}
	MsgReadFileFailed                   = Key{string: "读取文件失败"}
	MsgCVMTemplateMustBeJSON            = Key{string: "CVM 模板必须是合法的 JSON"}
	MsgInstanceChargeTypeUnsupported    = Key{string: "instance_charge_type 仅支持 PREPAID 或 POSTPAID_BY_HOUR"}
	MsgSerializeCVMTemplateFailed       = Key{string: "序列化 CVM 模板失败"}
	MsgVPCWithoutGlobalID               = Key{string: "配置了子网但未配置全局 VPC ID"}
	MsgSubnetCannotBeEmpty              = Key{string: "子网不能为空"}
	MsgZoneNotInRegion                  = Key{string: "可用区 %s 不在当前 Region 的可用区列表中"}
	MsgVpcIDParamRequired               = Key{string: "vpc_id 参数不能为空"}
	MsgZoneParamRequired                = Key{string: "zone 参数不能为空"}
	MsgQueryVPCListFailed               = Key{string: "查询 VPC 列表失败"}
	MsgQuerySubnetListFailed            = Key{string: "查询子网列表失败"}
	MsgFieldsNotAllowed                 = Key{string: "不允许修改的字段: %s"}
	MsgCurrentCVMTemplateInvalid        = Key{string: "当前 CVM 模板格式错误"}
	MsgSerializeTemplateFailed          = Key{string: "序列化模板失败"}
	MsgSaveDefaultTagsFailed            = Key{string: "保存默认标签失败"}
	MsgBrowserVNCPortOpenFailed         = Key{string: "云端浏览器功能已开启，但安全组端口放通失败: %s"}
	MsgVerifySecurityGroupFailed        = Key{string: "验证安全组失败"}
	MsgDescribeSGPoliciesFailed         = Key{string: "调用 DescribeSecurityGroupPolicies 失败"}
	MsgSGBootstrapNotDoneForGWUI        = Key{string: "ClawPro 安全组尚未初始化（rule_set=%d, active_sg=%d），请等待安全组初始化完成后再开启 Gateway UI"}
	MsgSaveGatewayUIConfigFailed        = Key{string: "保存 Gateway UI 配置失败"}
	MsgCreateSGPoliciesFailed           = Key{string: "调用 CreateSecurityGroupPolicies 失败"}
	MsgInternetAccessibleFormatError    = Key{string: "InternetAccessible 格式错误"}
	MsgInternetAccessibleParseError     = Key{string: "InternetAccessible 解析失败"}
	MsgSystemDiskFormatError            = Key{string: "SystemDisk 格式错误，应为对象"}
	MsgSystemDiskDiskTypeMustBeStr      = Key{string: "SystemDisk.DiskType 必须为字符串"}
	MsgSystemDiskDiskSizeMustBeNum      = Key{string: "SystemDisk.DiskSize 必须为数字"}
	MsgSystemDiskDiskSizeMustBeInt      = Key{string: "SystemDisk.DiskSize 必须为整数，当前值: %v"}
	MsgQueryCloudInstanceTypesCVMFailed = Key{string: "查询云端可用机型失败: 创建 CVM 客户端失败"}
	MsgQueryCloudInstanceTypesFailed    = Key{string: "查询云端可用机型失败"}
	MsgInstanceTypeNotAvailableAll      = Key{string: "实例规格 %s 不可用，当前 Region（%s）下白名单机型均不可用"}
	MsgInstanceTypeNotAvailable         = Key{string: "实例规格 %s 不可用，当前可选机型: %s"}
	MsgUnsupportedTemplatePath          = Key{string: "不支持的 template_path: %s"}
	MsgCVMTemplateFormatError           = Key{string: "CVM 模板格式错误"}
	MsgCreateSecurityGroupCallFailed    = Key{string: "调用 CreateSecurityGroup 失败"}
	MsgCloneSecurityGroupCallFailed     = Key{string: "调用 CloneSecurityGroup 失败"}
	MsgCloneSecurityGroupDataError      = Key{string: "克隆安全组返回数据异常"}
	MsgQueryInstanceListFailed          = Key{string: "查询实例列表失败"}
	MsgDescribeInstancesFailed          = Key{string: "调用 DescribeInstances 失败"}
	MsgDescribeSecurityGroupsFailed     = Key{string: "调用 DescribeSecurityGroups 失败"}
)

// 角色管理 (admin_roles.go)
var (
	MsgRoleNameCannotBeEmpty        = Key{string: "角色名称不能为空"}
	MsgRoleNameTooLong              = Key{string: "角色名称不能超过 %d 个字"}
	MsgSameRoleExists               = Key{string: "同名角色已存在"}
	MsgRoleNotFound                 = Key{string: "角色不存在"}
	MsgDeleteRoleFailed             = Key{string: "删除角色失败"}
	MsgToggleVisibilityFailed       = Key{string: "切换可见性失败"}
	MsgKeyRoleCreateFailed          = Key{string: "创建角色失败"}
	MsgKeyRoleUpdateFailed          = Key{string: "更新角色失败"}
	MsgKeyRoleSkillCreateFailed     = Key{string: "创建角色技能失败 slug=%s"}
	MsgKeyRolePluginCreateFailed    = Key{string: "创建角色插件失败 slug=%s"}
	MsgKeyRoleDeleteOldSkillFailed  = Key{string: "删除旧技能失败"}
	MsgKeyRoleDeleteOldPluginFailed = Key{string: "删除旧插件失败"}
	MsgKeyRoleReorderFailed         = Key{string: "更新排序失败 id=%d"}
	MsgIDsCannotBeEmpty             = Key{string: "ids 不能为空"}
	MsgInstanceNotLinkedToRole      = Key{string: "该实例未关联角色"}
	MsgInstanceNotSupportRole       = Key{string: "该实例类型不支持角色配置"}
	MsgRemoveRoleFailed             = Key{string: "移除角色失败"}
)

// 角色版本化 / 切换 / 分发 (admin_roles_distribute.go, openclaw_role_apply.go, admin_roles_instances.go)
var (
	MsgRoleVersionFormatInvalid     = Key{string: "版本号格式必须为 X.Y"}
	MsgRoleVersionMustBeHigher      = Key{string: "新版本号需高于上个版本号 %s"}
	MsgRoleSwitchInstanceIDInvalid  = Key{string: "instance_id 参数无效"}
	MsgRoleSwitchRoleIDInvalid      = Key{string: "role_id 参数无效"}
	MsgRoleDistributeMaxInstances   = Key{string: "批量推送实例数量超过上限 %d"}
	MsgRoleDistributeRoleIDRequired = Key{string: "role_id 参数必填"}
	MsgRoleDistributeIDsEmpty       = Key{string: "请选择需要推送的实例"}
	MsgRoleApplyDBUpdateFailed      = Key{string: "更新实例角色字段失败"}
	MsgRoleApplyLoadRoleFailed      = Key{string: "加载角色信息失败"}
	MsgRoleSyncStatusInvalid        = Key{string: "role_sync_status 参数无效"}
	MsgRoleListBuildGroupPathFailed = Key{string: "构建分组路径失败"}
)

// 角色可见性 (model/role_visibility.go)
var (
	MsgRoleVisQueryAssocFailed       = Key{string: "查询角色可见性关联失败"}
	MsgRoleVisBatchQueryAssocFailed  = Key{string: "批量查询角色可见性关联失败"}
	MsgRoleVisDeleteOldAssocFailed   = Key{string: "删除旧角色可见性关联失败"}
	MsgRoleVisCreateAssocFailed      = Key{string: "创建角色可见性关联失败"}
	MsgRoleVisUpdateVisibilityFailed = Key{string: "更新角色 visibility_type 失败"}
)

// 镜像管理 (admin_images.go)
var (
	MsgImageIDCannotBeEmpty        = Key{string: "镜像 ID 不能为空"}
	MsgAgentVersionCannotBeEmpty   = Key{string: "agent_version 不能为空"}
	MsgAgentVersionFormatOpenClaw  = Key{string: "OpenClaw 版本格式应为 YYYY.M.D，如 2026.3.28"}
	MsgAgentVersionFormatSemver    = Key{string: "%s 版本格式应为 X.Y.Z，如 0.9.0"}
	MsgAgentVersionFormatInvalid   = Key{string: "无效的版本号格式: %s"}
	MsgImageIDExists               = Key{string: "镜像 ID 已存在: %s"}
	MsgImageNotFound               = Key{string: "镜像不存在"}
	MsgImageNotFoundByID           = Key{string: "未找到镜像: %s"}
	MsgImageSizeTooLarge           = Key{string: "镜像大小不能超过 50GB"}
	MsgSaveImageFailed             = Key{string: "保存镜像失败"}
	MsgEnabledImageCannotDelete    = Key{string: "启用状态的镜像不能删除，请先禁用"}
	MsgDefaultTypeCannotDisable    = Key{string: "该类型为用户端首选，不可取消启用"}
	MsgPublicImageCannotEdit       = Key{string: "公共镜像不支持编辑，其类型和版本由系统维护"}
	MsgEnabledImageTypeCantChange  = Key{string: "启用中的镜像不能修改 Agent 类型，请先禁用后再编辑"}
	MsgNoFieldToUpdateImage        = Key{string: "没有需要更新的字段"}
	MsgUpdateFailed                = Key{string: "更新失败"}
	MsgDefaultAgentTypeNoEnabled   = Key{string: "该类型（%s）没有已启用镜像，无法设为首选"}
	MsgSetDefaultFailed            = Key{string: "设置失败"}
	MsgSetAgentVersionBeforeEnable = Key{string: "请先设置 Agent 版本后再启用"}
)

// MCP 管理
var (
	MsgMcpServiceConfigRequired      = Key{string: "请填写服务配置"}
	MsgMcpServiceIDExists            = Key{string: "该服务ID 已存在，请使用其他标识"}
	MsgInvalidCIDRFormat             = Key{string: "ip_whitelist 格式错误: %q 不是有效的 CIDR"}
	MsgInvalidIPAddress              = Key{string: "ip_whitelist 格式错误: %q 不是有效的 IP 地址"}
	MsgMcpVersionNotFoundDetail      = Key{string: "版本 %s 不存在"}
	MsgMcpDeleteSuccess              = Key{string: "删除成功"}
	MsgMcpDistributionRunning        = Key{string: "有下发任务正在进行中，请等待完成后再删除"}
	MsgMcpInstanceInstalling         = Key{string: "有实例正在安装中，请等待完成后再删除"}
	MsgMcpVersionFormatInvalid       = Key{string: "版本号格式不合法，需要 x.y.z 格式"}
	MsgMcpServiceIDRequired          = Key{string: "请输入服务ID"}
	MsgMcpServiceIDInvalidChars      = Key{string: "服务ID 仅支持英文、数字、中划线、下划线"}
	MsgMcpServiceIDTooLong           = Key{string: "服务ID 长度不能超过 48 个字符"}
	MsgMcpTransportTypeRequired      = Key{string: "请选择连接方式"}
	MsgMcpConfigJsonTooLarge         = Key{string: "服务配置 JSON 大小不能超过 16 KB"}
	MsgMcpConfigJsonParseError       = Key{string: "JSON 格式错误，请检查: %v"}
	MsgMcpConfigAtLeastOneServer     = Key{string: "请至少配置一个服务器"}
	MsgMcpTransportTypeMismatch      = Key{string: "transportType 与连接方式不一致（期望 \"}%s\"，实际 \"%s\"）"}
	MsgMcpURLRequired                = Key{string: "URL 不能为空"}
	MsgMcpURLMustStartWithHTTP       = Key{string: "URL 必须以 http 或 https 开头"}
	MsgMcpCommandRequired            = Key{string: "请输入可执行命令"}
	MsgMcpCreateVersionFailed        = Key{string: "创建版本失败"}
	MsgMcpVersionAlreadyExists       = Key{string: "版本 %s 已存在"}
	MsgMcpVersionMustBeGreaterMax    = Key{string: "版本号必须大于当前最高版本 %s"}
	MsgMcpDistributeVersionRequired  = Key{string: "version 不能为空"}
	MsgMcpDistributeMaxInstances500  = Key{string: "instance_ids 数量不能超过 500"}
	MsgMcpDistributeLockBusy         = Key{string: "有下发任务正在进行中，请稍后重试"}
	MsgMcpDistributeNoValidInstance  = Key{string: "没有符合条件的实例可以下发"}
	MsgMcpDistributeCreateTaskFailed = Key{string: "创建下发任务失败"}
)

// MCP 连通性探测 (controller/mcp_probe.go)
var (
	MsgMcpProbeInitializeFailed       = Key{string: "initialize 失败"}
	MsgMcpProbeInitializeError        = Key{string: "initialize 返回错误: %v"}
	MsgMcpProbeInitializeInvalidResp  = Key{string: "initialize 响应格式异常"}
	MsgMcpProbeCreateSSERequestFailed = Key{string: "创建 SSE 请求失败"}
	MsgMcpProbeSSEConnectFailed       = Key{string: "SSE 连接失败"}
	MsgMcpProbeSSEHTTPError           = Key{string: "SSE 连接返回 HTTP %d"}
	MsgMcpProbeSSEEndpointTimeout     = Key{string: "等待 endpoint 事件超时"}
	MsgMcpProbeSSENoEndpoint          = Key{string: "SSE 流关闭，未收到 endpoint 事件"}
	MsgMcpProbeSSEReadError           = Key{string: "SSE 读取错误"}
	MsgMcpProbeSSEEndpointInvalidURL  = Key{string: "SSE endpoint URL 格式错误"}
	MsgMcpProbeSSEEndpointPrivate     = Key{string: "SSE endpoint 指向内网地址，拒绝访问"}
	MsgMcpProbeJSONParseFailed        = Key{string: "响应 JSON 解析失败"}
	MsgMcpProbeHTTPError              = Key{string: "HTTP 状态码为 %d"}
	MsgMcpProbeSSEClosed              = Key{string: "SSE 流已关闭"}
	MsgMcpProbeSSEResponseReadFailed  = Key{string: "SSE 响应读取失败"}
	MsgMcpProbeSSENoData              = Key{string: "SSE 响应中未找到 JSON-RPC 数据"}
)

// 插件包管理 (admin_plugin_bundle.go)
var (
	MsgPluginBundleNameRequired      = Key{string: "插件包名称不能为空"}
	MsgPluginBundleNameTooLong       = Key{string: "插件包名称不能超过 100 个字符"}
	MsgPluginBundleConflict          = Key{string: "同名插件包已存在"}
	MsgPluginBundleNotFound          = Key{string: "插件包不存在"}
	MsgPluginBundleEnabledNeedsOff   = Key{string: "插件包正在生效中，需先禁用"}
	MsgCreatePluginBundleDBFailed    = Key{string: "创建插件包失败"}
	MsgPluginVersionConflictInBundle = Key{string: "插件 %s-%s 已存在于该插件包中"}
	MsgSmhResolveScriptFailed        = Key{string: "解析 %s 脚本失败 (agentType = %s)"}
	MsgDeletePluginBundleFailed      = Key{string: "删除插件包失败"}
	MsgOtherPluginBundleEnabled      = Key{string: "已有其他插件包处于启用状态，请先禁用"}
	MsgEnterprisePluginNotFound      = Key{string: "企业插件 ID=%d 不存在"}
	MsgGenEnterprisePluginURLFail    = Key{string: "生成企业插件下载 URL 失败"}
	MsgDownloadEnterprisePluginFail  = Key{string: "下载企业插件 zip 失败"}
	MsgDownloadEnterprisePluginCode  = Key{string: "下载企业插件 zip 失败 (status=%d)"}
	MsgReadEnterprisePluginFail      = Key{string: "读取企业插件 zip 失败"}
	MsgEnterprisePluginTooLarge      = Key{string: "企业插件 zip 文件超过 200MB 限制"}
	MsgUploadPluginZipFail           = Key{string: "上传插件 zip 到 common space 失败"}
)

// 龙虾医生 (doctor.go) — writeError 现场文案
var (
	MsgDoctorMissingInvocationID      = Key{string: "缺少参数 invocation_id"}
	MsgDoctorQueryAuthRecordFailed    = Key{string: "查询授权记录失败"}
	MsgDoctorCreateAuthRecordFail     = Key{string: "创建授权记录失败"}
	MsgDoctorAlreadyAuthorized        = Key{string: "已授权"}
	MsgDoctorAuthorizeSuccess         = Key{string: "授权成功"}
	MsgDoctorOnlyOpenClaw             = Key{string: "龙虾医生暂仅支持 OpenClaw 类型实例"}
	MsgDoctorFeatureDisabled          = Key{string: "龙虾医生功能未开启"}
	MsgDoctorNotAuthorized            = Key{string: "请先授权龙虾医生使用该实例"}
	MsgDoctorSGNotConfigured          = Key{string: "安全组未配置"}
	MsgDoctorActiveSessionExists      = Key{string: "已有进行中的诊断会话"}
	MsgDoctorGenProxyTokenFailed      = Key{string: "生成 ProxyToken 失败"}
	MsgDoctorCreateInstanceFailed     = Key{string: "创建实例记录失败"}
	MsgDoctorRequestSTSFailed         = Key{string: "申请 STS 临时密钥失败"}
	MsgDoctorGetEnabledImageFailed    = Key{string: "获取启用镜像失败"}
	MsgDoctorCVMCreateFailed          = Key{string: "CVM 创建失败"}
	MsgDoctorCVMReturnEmptyID         = Key{string: "CVM 返回空 InstanceId"}
	MsgDoctorSessionNotFound          = Key{string: "会话不存在"}
	MsgDoctorSessionAlreadyEnded      = Key{string: "会话已结束"}
	MsgDoctorCreateSessionFailed      = Key{string: "创建诊断会话失败"}
	MsgCVMTemplateConfigError         = Key{string: "CVM 模板配置错误"}
	MsgDoctorCVMTemplateNotConfigured = Key{string: "CVM 模板未配置 InstanceType，请联系管理员"}
	MsgDoctorSubnetZoneNotFound       = Key{string: "无法确定子网 %s 所在可用区，请联系管理员检查 VPC 配置"}
	MsgDoctorInstallCLIFailed         = Key{string: "安装 doctor-cli 失败"}
	MsgDoctorInstallSkillFailed       = Key{string: "安装龙虾医生 Skill 失败"}
	MsgDoctorSnapshotTargetNotFound   = Key{string: "目标实例不存在"}
	MsgDoctorSnapshotBackupFailed     = Key{string: "快照备份失败"}
	MsgDoctorSnapshotArchivePathEmpty = Key{string: "无法获取备份压缩包路径"}
	MsgDoctorSnapshotUploadFailed     = Key{string: "快照上传失败"}
	MsgDoctorParseMtimeFailed         = Key{string: "解析 mtime 失败: %s"}
)

// 技能共建审核 (contribution_skill.go)
var (
	MsgSkillContributeSuccess       = Key{string: "技能提交成功，等待管理员审核"}
	MsgSkillContributePendingExists = Key{string: "该技能已有进行中的申请，请等待审核完成"}
	MsgSkillTakedownSuccess         = Key{string: "下架申请已提交，等待管理员审核"}
	MsgSkillTakedownNotOwner        = Key{string: "只能下架自己上传的技能"}
	MsgSkillTakedownReasonRequired  = Key{string: "下架理由不能为空"}
	MsgReviewRequestNotFound        = Key{string: "审核申请不存在"}
	MsgReviewRequestNotPending      = Key{string: "该申请已审核，无法重复操作"}
	MsgReviewRejectCommentRequired  = Key{string: "拒绝理由不能为空"}
	MsgReviewApproveSuccess         = Key{string: "审核通过"}
	MsgReviewRejectSuccess          = Key{string: "已拒绝"}
	MsgReviewSkillNotExist          = Key{string: "关联的技能不存在"}
	MsgReviewNotOwner               = Key{string: "无权查看此申请"}
)

// 技能共建审核通知
var (
	NotifTitleSkillReviewApproved   = Key{string: "技能审核通过"}
	NotifTitleSkillReviewRejected   = Key{string: "技能审核未通过"}
	NotifTitleSkillTakedownApproved = Key{string: "技能下架通过"}
	NotifTitleSkillTakedownRejected = Key{string: "技能下架未通过"}
	NotifTitleNewReviewRequest      = Key{string: "新的技能审核申请"}
)

var (
	NotifMsgSkillReviewApproved   = Key{string: "您提交的技能「%s」已通过审核，已上架到企业技能库"}
	NotifMsgSkillReviewRejected   = Key{string: "您提交的技能「%s」未通过审核，原因：%s"}
	NotifMsgSkillTakedownApproved = Key{string: "您申请下架的技能「%s」已通过审核，已下架"}
	NotifMsgSkillTakedownRejected = Key{string: "您申请下架的技能「%s」未通过审核，原因：%s"}
	NotifMsgNewReviewRequest      = Key{string: "员工 %s 提交了技能「%s」的%s申请，请前往审核"}
)

// 技能广场 (openclaw_skillstore.go)
var (
	MsgSkillStoreSlugRequired        = Key{string: "缺少 slug 参数"}
	MsgSkillStoreVisibilityCheckFail = Key{string: "可见性检查失败"}
	MsgSkillStoreNotOwnInstance      = Key{string: "包含非本人实例"}
	MsgSkillStoreVersionLocked       = Key{string: "该技能版本正在被其他操作处理，请稍后重试"}
	MsgSkillStoreSkillLocked         = Key{string: "该技能正在被其他操作处理，请稍后重试"}
	MsgSkillStoreQueryInstanceFail   = Key{string: "查询实例信息失败"}
	MsgSkillStoreNoValidInstall      = Key{string: "没有符合条件的实例，所选实例类型不支持技能安装"}
	MsgSkillStoreNoValidUninstall    = Key{string: "没有符合条件的实例，所选实例类型不支持技能"}
	MsgSkillStoreQueryInstancesFail  = Key{string: "查询实例列表失败: %v"}
	MsgSkillStoreCreateRecordFail    = Key{string: "创建下发记录失败: %v"}
	MsgSkillStoreCreateUninstallTask = Key{string: "创建卸载任务失败: %v"}
	MsgSkillStoreCreateUninstallRec  = Key{string: "创建卸载记录失败: %v"}
	MsgSkillStoreGenDownloadURLFail  = Key{string: "生成下载链接失败: %v"}
)

// CLS 日志服务 (admin_cls.go)
var (
	MsgCLSCheckRoleFailed        = Key{string: "检查服务角色失败"}
	MsgCLSCreateCommonClientFail = Key{string: "创建 CLS CommonClient 失败"}
	MsgCLSQueryServiceStatusFail = Key{string: "查询 CLS 服务状态失败"}
	MsgCLSOpenServiceFailed      = Key{string: "开通 CLS 服务失败"}
	MsgCLSServiceOpened          = Key{string: "CLS 日志服务已开通"}
	MsgCLSCreateClientFail       = Key{string: "创建 CLS Client 失败"}
	MsgCLSQueryLogTopicFail      = Key{string: "查询 CLS 日志主题失败"}
	MsgCLSQueryMetricTopicFail   = Key{string: "查询 CLS 指标主题失败"}
	MsgCLSDeleteLogTopicFail     = Key{string: "删除日志主题 %s 失败"}
	MsgCLSDeleteMetricTopicFail  = Key{string: "删除指标主题 %s 失败"}
	MsgCLSQueryTraceTopicFail    = Key{string: "查询 CLS Trace 主题失败"}
	MsgCLSDeleteTraceTopicFail   = Key{string: "删除 Trace 主题 %s 失败"}
	MsgCLSQueryTopicFail         = Key{string: "查询 CLS 主题失败"}
	MsgCLSCallModifyAttrFail     = Key{string: "调用 ModifyInstancesAttribute 失败"}
	MsgCLSRoleExists             = Key{string: "角色 %s 存在"}
	MsgCLSRoleNotExistOrCredErr  = Key{string: "角色 %s 不存在或凭证异常"}
	MsgCLSServiceNotEnabledNoOff = Key{string: "CLS ClawPro 服务未开通，无需关闭"}
	MsgCLSBindRoleSuccess        = Key{string: "已为 %d 台实例绑定角色 %s"}
)

// CLS 插件升级 (controller/admin_cls_update.go)
var (
	MsgCLSUpdateRollbackTimeoutFailed = Key{string: "回滚超时 updating 实例失败"}
	MsgCLSUpdateScriptOutputEmpty     = Key{string: "脚本输出为空"}
	MsgCLSUpdateJSONParseFailed       = Key{string: "解析 JSON 失败"}
)

// 通知中心 (notification.go)
var (
	MsgNotificationIsReadInvalid   = Key{string: "is_read 参数值无效，仅支持 true 或 false"}
	MsgNotificationCategoryInvalid = Key{string: "category 参数值无效，仅支持 success/error/notice"}
	MsgNotificationQueryFailed     = Key{string: "查询通知失败"}
	MsgNotificationMarkAllFailed   = Key{string: "标记全部已读失败"}
	MsgNotificationMarkReadFailed  = Key{string: "标记已读失败"}
	MsgNotificationQueryUnread     = Key{string: "查询未读数量失败"}
	MsgNotificationDeleteMax100    = Key{string: "单次最多删除 100 条通知"}
	MsgNotificationDeleteFailed    = Key{string: "删除通知失败"}
)

// 插件管理 (admin_plugins.go)
var (
	MsgPluginSlugNameVerRequired   = Key{string: "slug、name、version 为必填字段"}
	MsgPluginInvalidSlug           = Key{string: "slug 格式不合法，只允许小写字母、数字和连字符，长度 3-50，不能以连字符开头或结尾"}
	MsgPluginVersionExist          = Key{string: "该插件版本已存在（slug=%s, version=%s），请修改后重试"}
	MsgPluginFileFieldMissing      = Key{string: "缺少 file 字段"}
	MsgPluginFileSizeTooLarge      = Key{string: "文件大小超过 200MB 限制"}
	MsgPluginFileListTooLarge      = Key{string: "插件包内文件数量过多，文件列表超过数据库字段限制（65535字节），请减少文件数量或缩短文件路径"}
	MsgPluginReadUploadFail        = Key{string: "读取上传文件失败: %v"}
	MsgPluginCreateRecordFail      = Key{string: "创建插件记录失败: %v"}
	MsgPluginSMHUnavailable        = Key{string: "SMH 存储服务不可用: %v"}
	MsgPluginUploadZipFail         = Key{string: "上传插件 zip 包到 SMH 失败: %v"}
	MsgPluginParseFormFail         = Key{string: "解析表单失败: %v"}
	MsgPluginSlugVersionRequired   = Key{string: "slug 和 version 为必填字段"}
	MsgPluginNotFound              = Key{string: "插件不存在"}
	MsgPluginVersionNotFoundDetail = Key{string: "版本不存在（slug=%s, version=%s）"}
	MsgPluginDeleteFailed          = Key{string: "删除插件失败: %v"}
	MsgPluginRequestFormatErr      = Key{string: "请求格式错误: %v"}
	MsgPluginMaxInstances500       = Key{string: "单次下发实例数不能超过 500"}
	MsgPluginVersionLocked         = Key{string: "该插件版本正在被其他操作处理，请稍后重试"}
	MsgPluginQueryInstanceInfo     = Key{string: "查询实例信息失败: %v"}
	MsgPluginNoValidInstall        = Key{string: "没有符合条件的实例，所选实例类型不支持插件安装"}
	MsgPluginCreateTaskFail        = Key{string: "创建下发任务失败: %v"}
	MsgPluginAlreadyFavorited      = Key{string: "该插件已收藏"}
	MsgPluginFavoriteFail          = Key{string: "收藏插件失败"}
	MsgPluginVersionInUse          = Key{string: "该版本有进行中的下发任务，无法删除"}

	// 插件 zip 校验
	MsgPluginZipNoManifestOrBundle  = Key{string: "zip 中未找到 openclaw.plugin.json 或 Bundle 目录（.codex-plugin/.claude-plugin/.cursor-plugin），请检查插件包格式"}
	MsgPluginZipReadManifestFail    = Key{string: "读取 openclaw.plugin.json 失败"}
	MsgPluginZipManifestTooLarge    = Key{string: "openclaw.plugin.json 文件过大（超过 1MB），请检查插件包"}
	MsgPluginZipParseManifestFail   = Key{string: "解析 openclaw.plugin.json 失败"}
	MsgPluginZipManifestMissingID   = Key{string: "openclaw.plugin.json 中缺少 id 字段"}
	MsgPluginZipManifestInvalidKind = Key{string: "openclaw.plugin.json 中 kind 字段不合法，仅支持 memory / context-engine / 空"}
	MsgPluginZipBombDetected        = Key{string: "zip 实际解压大小超过 200MB 限制（疑似 ZIP 炸弹）"}

	// 插件可见性
	MsgPluginVisibilityGroupIDsRequired = Key{string: "visibility_type 为 group 时 group_ids 不能为空"}
	MsgPluginSetVisibilityFail          = Key{string: "设置插件可见范围失败"}
	MsgPluginVisQueryGroupIDsFailed     = Key{string: "批量查询插件可见性关联失败"}
	MsgPluginVisDeleteOldFailed         = Key{string: "删除旧插件可见性关联失败"}
	MsgPluginVisCreateFailed            = Key{string: "创建插件可见性关联失败"}
	MsgPluginVisUpdateTypeFailed        = Key{string: "更新插件 visibility_type 失败"}
	MsgPluginVisCopyTypeFailed          = Key{string: "复制 visibility_type 失败"}
	MsgPluginVisQueryOldFailed          = Key{string: "查询旧版本可见性关联失败"}
	MsgPluginVisCopyGroupsFailed        = Key{string: "复制插件可见性关联失败"}

	// 插件版本号解析
	MsgPluginVersionFormatInvalid = Key{string: "版本号格式不合法，需要 x.y.z 格式: %s"}
	MsgPluginVersionMajorInvalid  = Key{string: "版本号 major 不合法: %s"}
	MsgPluginVersionMinorInvalid  = Key{string: "版本号 minor 不合法: %s"}
	MsgPluginVersionPatchInvalid  = Key{string: "版本号 patch 不合法: %s"}
	MsgPluginVersionExceeds999    = Key{string: "版本号各段不能超过 999: %s"}
	MsgPluginVersionNegative      = Key{string: "版本号各段不能为负数: %s"}
	MsgPluginVersionParseFailed   = Key{string: "版本号解析失败（需要数字）: %s"}

	// 插件版本
	MsgPluginVersionNotExist = Key{string: "插件版本不存在"}

	// 插件类型参数
	MsgPluginInvalidTypeParam = Key{string: "type 参数无效，只接受 distribute 或 uninstall"}

	// 插件卸载
	MsgPluginBuildQueryFail               = Key{string: "构造查询失败"}
	MsgPluginCreateUninstallTaskFail      = Key{string: "创建卸载任务失败"}
	MsgPluginCreateUninstallRecordFail    = Key{string: "创建卸载记录失败"}
	MsgPluginAgentTypeNotSupportUninstall = Key{string: "agent_type %s 不支持插件卸载"}
)

// 镜像更新历史 (admin_image_updates.go)
var (
	MsgImgUpdPublishedAtInvalid    = Key{string: "published_at 格式应为 RFC3339 或 YYYY-MM-DD"}
	MsgImgUpdAdminTokenOnlyPublish = Key{string: "仅 admin-token 可以发布镜像更新动态"}
	MsgImgUpdOnlyOfficialPublish   = Key{string: "仅官方镜像支持发布更新通知"}
	MsgImgUpdQueryHistoryFail      = Key{string: "查询镜像更新历史失败"}
	MsgImgUpdSaveHistoryFail       = Key{string: "保存镜像更新历史失败"}
	MsgImgUpdSyncVersionFail       = Key{string: "同步镜像版本失败"}
	MsgImgUpdOnlyOfficialSync      = Key{string: "仅官方镜像支持同步版本"}
	MsgImgUpdAdminTokenOnlyEdit    = Key{string: "仅 admin-token 可以修改镜像更新历史"}
	MsgImgUpdVersionOrTimeRequired = Key{string: "agent_version 和 published_at 至少填写一个"}
	MsgImgUpdHistoryNotFound       = Key{string: "镜像更新历史不存在"}
	MsgImgUpdOnlyOfficialEdit      = Key{string: "仅官方镜像支持修改更新历史"}
	MsgImgUpdEditHistoryFail       = Key{string: "修改镜像更新历史失败"}
	MsgImgUpdAdminTokenOnlyDelete  = Key{string: "仅 admin-token 可以删除镜像更新历史"}
	MsgImgUpdIDOrImageIDRequired   = Key{string: "id 或 image_id 不能为空"}
	MsgImgUpdOnlyOfficialDelete    = Key{string: "仅官方镜像支持删除更新历史"}
	MsgImgUpdDeleteHistoryFail     = Key{string: "删除镜像更新历史失败"}
	MsgImgUpdAdminTokenOnlyEnable  = Key{string: "仅 admin-token 可以启用镜像更新历史"}
	MsgImgUpdOnlyOfficialEnable    = Key{string: "仅官方镜像支持启用更新历史"}
	MsgImgUpdEnableHistoryFail     = Key{string: "启用镜像更新历史失败"}
	MsgImgUpdOnlyOfficialNotice    = Key{string: "仅官方镜像支持更新通知开关"}
	MsgImgUpdNoNoticeForImage      = Key{string: "该镜像尚无更新动态"}
	MsgImgUpdToggleNoticeFail      = Key{string: "更新通知开关失败"}
	MsgImgUpdQueryEnabledImage     = Key{string: "查询当前租户启用镜像失败"}
	MsgImgUpdQueryTenantImage      = Key{string: "查询当前租户镜像失败"}
	MsgImgUpdQueryLatestHistory    = Key{string: "查询镜像最新更新历史失败"}
	MsgImgUpdQueryNoticeStatus     = Key{string: "查询镜像更新通知状态失败"}
	MsgImgUpdQueryEnabledStatus    = Key{string: "查询镜像启用状态失败"}
	MsgImgUpdQueryOutdatedCount    = Key{string: "查询旧版本运行实例数量失败"}
	MsgImgUpdQueryNotice           = Key{string: "查询镜像更新通知失败"}
	MsgImgUpdLogPublishHistoryFail = Key{string: "发布镜像更新历史失败"}
	MsgImgUpdLogSyncVersionsFail   = Key{string: "同步所有租户官方镜像版本失败"}
)

// Memory Pro 管理 (admin_memory_pro.go)
var (
	MsgMemoryLimitMustGTZero         = Key{string: "memory_limit 必须大于 0"}
	MsgMemoryProAlreadyActivated     = Key{string: "Pro 服务已开通，请勿重复创建"}
	MsgMemoryGetNetworkParamsFail    = Key{string: "获取网络参数失败"}
	MsgMemoryCreateVDBFail           = Key{string: "创建 VDB 实例失败"}
	MsgMemoryQueryProInstanceFail    = Key{string: "查询 Pro 实例失败"}
	MsgMemoryProInstanceNotFound     = Key{string: "未找到 Pro 服务实例"}
	MsgMemoryProInUseRefuseRelease   = Key{string: "仍有 %d 个实例在使用 Pro 记忆库（%s%s），请先将这些实例切换到 OFF 后再关闭 Pro 服务"}
	MsgMemoryReleaseProFailed        = Key{string: "以下 Pro 实例释放失败，请查看日志并重试：%s"}
	MsgMemoryUpdateDefaultPlanFail   = Key{string: "更新默认计划失败"}
	MsgMemoryOnlyGetPutMethod        = Key{string: "仅支持 GET/PUT"}
	MsgMemoryProAgentTypeNoMemoryFmt = Key{string: "%s 类型实例不支持记忆功能"}
	MsgMemoryProMoreSuffixFmt        = Key{string: " 等共 %d 个"}
	MsgMemoryProAllRejectedTip       = Key{string: "以下 Agent 所在 CVM (%s) 到 %s 网络不通，无法切换到 Pro 版。请检查 CVM 与 VDB 所在 VPC 是否连通，以及 CVM 与 VDB 的安全组规则是否放通。"}
)

// Agent 命令模板 (admin_agent_commands.go)
var (
	MsgAgentCmdIDRequired              = Key{string: "缺少命令 id"}
	MsgAgentCmdEditDenied              = Key{string: "仅创建者或超级管理员可编辑该命令"}
	MsgAgentCmdDeleteDenied            = Key{string: "仅创建者或超级管理员可删除该命令"}
	MsgAgentCmdQuotaExceeded           = Key{string: "已达每租户 %d 个命令上限"}
	MsgAgentCmdInUseDetail             = Key{string: "命令仍有进行中的执行任务，无法删除"}
	MsgAgentCmdNameInvalidChars        = Key{string: "命令名称含非法字符"}
	MsgAgentCmdNameTooLong             = Key{string: "命令名称过长"}
	MsgAgentCmdDescriptionTooLong      = Key{string: "命令描述超过 512 字符上限"}
	MsgAgentCmdContentRequired         = Key{string: "命令内容必填"}
	MsgAgentCmdContentTooLong          = Key{string: "命令内容过长"}
	MsgAgentCmdTimeoutOutOfRange       = Key{string: "超时时长需在 1–86400 秒之间"}
	MsgAgentCmdRunUserTooLong          = Key{string: "执行用户名超过 64 字符上限"}
	MsgAgentCmdWorkdirTooLong          = Key{string: "工作目录超过 255 字符上限"}
	MsgAgentCmdInvalidType             = Key{string: "不支持的命令类型"}
	MsgAgentCmdParamsTooMany           = Key{string: "参数数量超过 10 个上限"}
	MsgAgentCmdParamNameInvalid        = Key{string: "参数名格式错误：%s"}
	MsgAgentCmdParamNameDuplicated     = Key{string: "参数名重复：%s"}
	MsgAgentCmdParamDefaultTooLong     = Key{string: "参数 %s 的默认值超过 128 字符上限"}
	MsgAgentCmdParamDescriptionTooLong = Key{string: "参数 %s 的描述超过 200 字符上限"}
	MsgAgentCmdMarshalParamsFailed     = Key{string: "序列化命令参数失败"}
	MsgAgentCmdFindByIDFailed          = Key{string: "查询命令失败"}
	MsgAgentCmdFindBySlugFailed        = Key{string: "按 slug 查询命令失败"}
	MsgAgentCmdCreateFailed            = Key{string: "创建命令失败"}
	MsgAgentCmdSlugCheckFailed         = Key{string: "检查命令 slug 冲突失败"}
	MsgAgentCmdSlugConflict            = Key{string: "命令 slug 冲突，请重试"}
	MsgAgentCmdCountFailed             = Key{string: "查询命令数量失败"}
	MsgAgentCmdQueryListFailed         = Key{string: "查询命令列表失败"}
	MsgAgentCmdUpdateFailed            = Key{string: "更新命令失败"}
	MsgAgentCmdDeleteFailed            = Key{string: "删除命令失败"}
	MsgAgentCmdNameCheckFailed         = Key{string: "检查命令名称冲突失败"}
	MsgAgentCmdNameAlreadyExists       = Key{string: "命令名称已存在：%s"}
)

// OneID Sync (oneid_sync handler 部分)
var (
	MsgOneIDSyncGatewayNotConfigured = Key{string: "Gateway 未配置"}
	MsgOneIDSyncCreateRequestFailed  = Key{string: "创建请求失败: %v"}
	MsgOneIDSyncRequestGatewayFailed = Key{string: "请求 Gateway 失败: %v"}
	MsgOneIDSyncGatewayReturnedError = Key{string: "Gateway 接口返回 %d: %s"}
	MsgOneIDSyncParseResponseFailed  = Key{string: "解析响应失败: %v"}
	MsgOneIDSyncOneIDReturnedError   = Key{string: "OneID 返回错误: code=%d, msg=%s"}
	MsgOneIDSyncUpdateConfigFailed   = Key{string: "更新配置失败: %v"}
	MsgOneIDSyncInProgress           = Key{string: "同步正在进行中，请稍后再试"}
	MsgOneIDSyncCompleted            = Key{string: "同步完成"}
)

// OneID 统一账号 (controller/oneid_unified.go)
var (
	MsgOneIDCredsNotConfigured      = Key{string: "统一账号模式凭证未配置"}
	MsgOneIDBuildTokenRequestFailed = Key{string: "构建 Token 请求失败"}
	MsgOneIDTokenRequestFailed      = Key{string: "Token 请求失败"}
	MsgOneIDTokenEndpointError      = Key{string: "Token 端点返回 %d: %s"}
	MsgOneIDParseTokenFailed        = Key{string: "解析 Token 响应失败: %s"}
	MsgOneIDParseRootsFailed        = Key{string: "解析根部门响应失败"}
	MsgOneIDAPIError                = Key{string: "OneID API 错误: code=%d msg=%s"}
	MsgOneIDGetAppTokenFailed       = Key{string: "获取自建应用 Token 失败"}
	MsgOneIDParseCreateUserFailed   = Key{string: "解析创建用户响应失败"}
	MsgOneIDGatewayURLNotConfigured = Key{string: "GATEWAY_URL 未配置"}
	MsgOneIDAccountIDNotAvailable   = Key{string: "OneID account_id 不可用"}
	MsgOneIDBuildRequestFailed      = Key{string: "构建请求失败"}
	MsgOneIDGatewayRequestFailed    = Key{string: "Gateway 请求失败"}
	MsgOneIDGatewayReturnedError    = Key{string: "Gateway 返回 %d: %s"}
	MsgOneIDAPIRequestFailed        = Key{string: "OneID API 请求失败"}
	MsgOneIDAPIReturned             = Key{string: "OneID API 返回 %d: %s"}
	MsgOneIDUsernameNotEmpty        = Key{string: "用户名不能为空"}
	MsgOneIDUsernameTooLong         = Key{string: "用户名长度不能超过 191 个字符"}
	MsgOneIDUsernameInvalidChars    = Key{string: "用户名仅支持大小写字母、数字和特殊字符"}
	MsgOneIDSessionTokenEmpty       = Key{string: "session_token 为空"}
	MsgOneIDGetSelfV3Failed         = Key{string: "请求 OneID get_self_v3 失败"}
	MsgOneIDGetSelfV3HTTPError      = Key{string: "OneID get_self_v3 返回 HTTP %d: %s"}
	MsgOneIDParseGetSelfV3Failed    = Key{string: "解析 OneID get_self_v3 响应失败"}
	MsgOneIDGetSelfV3BizError       = Key{string: "OneID get_self_v3 业务错误: %s %s"}
	MsgNotUnifiedAccountMode        = Key{string: "未处于统一账号模式"}
	MsgOneIDMissingUsernameParam    = Key{string: "缺少 username 参数"}
	MsgOneIDBuildProxyRequestFailed = Key{string: "构建代理请求失败"}
	MsgOneIDRequestFailed           = Key{string: "OneID 请求失败"}
)

// OneID 认证 (controller/auth_oneid.go)
var (
	MsgOneIDAuthInvalidIDTokenFormat   = Key{string: "id_token 格式无效"}
	MsgOneIDAuthBase64DecodePayload    = Key{string: "base64 解码 token payload 失败"}
	MsgOneIDAuthJSONParsePayload       = Key{string: "JSON 解析 token payload 失败"}
	MsgOneIDAuthIDTokenMissingSub      = Key{string: "id_token 缺少 sub 字段"}
	MsgOneIDAuthInvalidLogoutTokenFmt  = Key{string: "logout_token 格式无效"}
	MsgOneIDAuthInternalSecretNotSet   = Key{string: "InternalSecret 未配置"}
	MsgOneIDAuthMissingInternalToken   = Key{string: "缺少 X-Internal-Token 头"}
	MsgOneIDAuthTokenMissingSeparator  = Key{string: "token 格式无效：缺少分隔符"}
	MsgOneIDAuthInvalidTimestamp       = Key{string: "时间戳无效"}
	MsgOneIDAuthTokenExpired           = Key{string: "token 已过期"}
	MsgOneIDAuthTokenExpiredDiff       = Key{string: "token 已过期 (diff=%ds)"}
	MsgOneIDAuthInvalidSignatureEnc    = Key{string: "签名编码无效"}
	MsgOneIDAuthSignatureMismatch      = Key{string: "签名不匹配"}
	MsgOneIDAuthDecodeSignatureFailed  = Key{string: "解码签名失败"}
	MsgOneIDAuthInvalidSignature       = Key{string: "签名无效"}
	MsgOneIDAuthDecodePayloadFailed    = Key{string: "解码 payload 失败"}
	MsgOneIDAuthUnmarshalPayloadFailed = Key{string: "反序列化 payload 失败"}
)

// OneID 统一账号 - 部门同步 (controller/oneid_unified_dept.go)
var (
	MsgOneIDParseCreateDeptResponse  = Key{string: "解析创建部门响应失败"}
	MsgOneIDCreateDeptError          = Key{string: "OneID 创建部门错误: code=%d msg=%s"}
	MsgOneIDParseUpdateDeptResponse  = Key{string: "解析更新部门响应失败"}
	MsgOneIDUpdateDeptError          = Key{string: "OneID 更新部门错误: code=%d msg=%s"}
	MsgOneIDParseDeleteDeptResponse  = Key{string: "解析删除部门响应失败"}
	MsgOneIDDeleteDeptError          = Key{string: "OneID 删除部门错误: code=%d msg=%s"}
	MsgOneIDNonManualGroupNoRef      = Key{string: "非 manual 分组 %d (source=%s) 无 source_ref"}
	MsgOneIDResolveParentDeptFailed  = Key{string: "解析分组 %d 的父部门失败"}
	MsgOneIDCreateDeptForGroupFailed = Key{string: "为分组 %d (%s) 创建 OneID 部门失败"}
	MsgOneIDStoreSourceRefFailed     = Key{string: "存储分组 %d 的 source_ref 失败"}
	MsgOneIDGetRootDeptFailed        = Key{string: "获取 OneID 根部门失败"}
	MsgOneIDFindParentGroupFailed    = Key{string: "查询父分组 %d 失败"}
	MsgOneIDResolveNewParentFailed   = Key{string: "解析新父部门失败"}
	MsgOneIDQueryUsersFailed         = Key{string: "查询用户失败"}
	MsgOneIDUpdateUserDeptsFailed    = Key{string: "更新用户 %s 部门归属失败"}
)

// 技能管理 (admin_skills.go)
var (
	MsgSkillVisQueryGroupFail       = Key{string: "查询分组失败: %v"}
	MsgSkillNewVersionMustBeGreater = Key{string: "新版本号 %s 必须大于现有最高版本 %s"}
	MsgSkillVersionExist            = Key{string: "该技能版本已存在（slug=%s, version=%s），请修改后重试"}
	MsgSkillFileSizeTooLarge        = Key{string: "文件大小超过 50MB 限制"}  // Bundle 远端下载上限
	MsgSkillUploadFileSizeTooLarge  = Key{string: "文件大小超过 100MB 限制"} // Skill 本地上传上限
	MsgSkillFileListTooLarge        = Key{string: "技能包内文件数量过多，文件列表超过数据库字段限制（65535字节），请减少文件数量或缩短文件路径"}
	MsgSkillInjectMetaFail          = Key{string: "注入 _meta.json 失败: %v"}
	MsgSkillCreateRecordFail        = Key{string: "创建技能记录失败: %v"}
	MsgSkillSetVisibilityFail       = Key{string: "设置技能可见范围失败: %v"}
	MsgSkillInheritVisibilityFail   = Key{string: "继承技能可见范围失败: %v"}
	MsgSkillUploadZipFail           = Key{string: "上传技能 zip 包到 SMH 失败: %v"}
	MsgSkillUpdateFail              = Key{string: "更新技能失败: %v"}
	MsgSkillSlugRequired            = Key{string: "slug 为必填字段"}
	MsgSkillDeleteFail              = Key{string: "删除技能失败: %v"}

	// 技能 zip 校验
	MsgZipParseFail       = Key{string: "无法解析 zip 文件: %v"}
	MsgZipEmpty           = Key{string: "zip 文件为空"}
	MsgZipMultiSkillMd    = Key{string: "zip 中存在多个 SKILL.md，请确保只有一个"}
	MsgZipNoSkillMd       = Key{string: "不存在 SKILL.md 文件，请修改后重试"}
	MsgZipInvalidPath     = Key{string: "zip 包含非法路径: %s"}
	MsgZipTooLarge        = Key{string: "zip 解压后总大小超过 200MB 限制"}
	MsgZipReadEntryFail   = Key{string: "读取 zip 文件失败: %v"}
	MsgZipRepackFail      = Key{string: "重新打包 zip 失败: %v"}
	MsgZipFinishFail      = Key{string: "完成 zip 打包失败: %v"}
	MsgZipBadFileName     = Key{string: "以下文件名包含不支持的特殊字符，请重命名后重新打包：%s"}
	MsgZipFileNameNotUTF8 = Key{string: "zip 中存在非 UTF-8 编码的文件名，请将文件名统一转换为 UTF-8 后重新打包上传"}
	MsgZipNoValidFile     = Key{string: "zip 文件中没有有效文件"}

	// 技能下发/卸载内部错误
	MsgSkillFileUploadSMHFail        = Key{string: "文件 [%s] 上传 SMH 失败: %v"}
	MsgSkillDownloadURLGenFail       = Key{string: "SMH 下载 URL 生成失败: %s"}
	MsgSkillTopLevelFieldsWithSkills = Key{string: "设置 skills 时顶层技能字段必须为空"}
	MsgSkillBatchItemsCountLimit     = Key{string: "skills 数量不能超过 %d"}
	MsgSkillCheckTaskFail            = Key{string: "检查下发任务失败: %v"}
	MsgSkillCascadeDeleteBundleFail  = Key{string: "级联删除技能包技能失败: %v"}
	MsgSkillCascadeDeleteRoleFail    = Key{string: "级联删除角色技能失败: %v"}
	MsgSkillDeleteRecordFail         = Key{string: "删除技能记录失败: %v"}
)

// 企业规范库（本地 agent 二期）
var (
	MsgRuleTypeRequired            = Key{string: "type 为必填字段（prompt / rule / hook）"}
	MsgRuleTypeInvalid             = Key{string: "type 必须为 prompt 或 rule 或 hook"}
	MsgHookEventInvalid            = Key{string: "event 必须为 SessionStart / UserPromptSubmit / PreToolUse / PostToolUse / Stop 之一（当前=%s）"}
	MsgHookCmdRequired             = Key{string: "hook 类型必须提供 cmd（执行命令）"}
	MsgRuleTypeMismatch            = Key{string: "type 与首版本不一致（slug=%s, 首版 type=%s）"}
	MsgRuleNotExist                = Key{string: "规范不存在"}
	MsgRuleVersionNotFound         = Key{string: "规范版本不存在（slug=%s, version=%s）"}
	MsgRuleFileFieldMissing        = Key{string: "缺少规范 file 字段"}
	MsgRuleFileMustBeMD            = Key{string: "文件必须为 .md，且大小不超过 1 MiB"}
	MsgRuleFileContentInvalid      = Key{string: "文件内容非法：非 UTF-8 或包含 \\x00"}
	MsgRuleFileContentEmpty        = Key{string: "文件内容不能为空"}
	MsgRuleReadUploadFail          = Key{string: "读取规范上传文件失败: %v"}
	MsgRuleUploadSMHFail           = Key{string: "上传 SMH 失败: %v"}
	MsgRuleCreateRecordFail        = Key{string: "创建规范记录失败: %v"}
	MsgRuleVersionExist            = Key{string: "该规范版本已存在（slug=%s, version=%s），请修改后重试"}
	MsgRuleNewVersionMustBeGreater = Key{string: "新规范版本号 %s 必须大于现有最高版本 %s"}
	MsgRuleHasRunningTask          = Key{string: "规范有正在运行的下发任务，请等待任务结束后重试"}
	MsgRuleCheckTaskFail           = Key{string: "检查规范下发任务失败: %v"}
	MsgRuleDeleteRecordFail        = Key{string: "删除规范记录失败: %v"}
	MsgRuleDeleteFail              = Key{string: "删除规范失败: %v"}
	MsgRuleSlugOrBatchIDRequired   = Key{string: "slug 或 batch_id 至少填一个"}
)

// 技能可见性 (model/skill_visibility.go)
var (
	MsgSkillVisQueryAssocFailed       = Key{string: "查询技能可见性关联失败"}
	MsgSkillVisBatchQueryAssocFailed  = Key{string: "批量查询技能可见性关联失败"}
	MsgSkillVisDeleteOldAssocFailed   = Key{string: "删除旧技能可见性关联失败"}
	MsgSkillVisCreateAssocFailed      = Key{string: "创建技能可见性关联失败"}
	MsgSkillVisUpdateVisibilityFailed = Key{string: "更新技能 visibility_type 失败"}
	MsgSkillVisCopyAssocFailed        = Key{string: "复制技能可见性关联失败"}
)

// 技能版本解析 (model/skill.go)
var (
	MsgSkillVersionMajorInvalid         = Key{string: "版本号 major 不合法"}
	MsgSkillVersionMinorInvalid         = Key{string: "版本号 minor 不合法"}
	MsgSkillVersionPatchInvalid         = Key{string: "版本号 patch 不合法"}
	MsgSkillQueryOldVersionFailed       = Key{string: "查询旧版本失败"}
	MsgSkillUpdateDistributeCountFailed = Key{string: "更新 distribute_count 失败"}
)

// Token配额规则验证相关
var (
	MsgInvalidTokenQuotaRules  = Key{string: "无效的token配额规则格式"}
	MsgDatabaseOperationFailed = Key{string: "数据库操作失败"}
	MsgQuotaRulesCannotBeEmpty = Key{string: "配额规则不能为空"}
	MsgInvalidQuotaRulesJSON   = Key{string: "无效的配额规则 JSON: %v"}
	MsgQuotaModeDuplicate      = Key{string: "周期类型不允许重复"}
	MsgQuotaModeInvalid        = Key{string: "周期类型不合法"}
	MsgQuotaRefreshModeInvalid = Key{string: "周期刷新模式不合法"}
	MsgQuotaEndBeforeStart     = Key{string: "结束时间必须晚于开始时间"}
)

// 租户解析 (identifier_middleware.go)
var (
	MsgUnknownTenant = Key{string: "未知租户"}
)

// 内存 TDAI 管理 (admin_memory_tdai.go)
var (
	MsgOnlyPutMethodSupported     = Key{string: "仅支持 PUT 方法"}
	MsgMemoryTDAIEnableRequired   = Key{string: "memory_tdai_enable 为必填字段"}
	MsgQueryMemoryTDAIStatsFailed = Key{string: "查询 memory_tdai 状态统计失败"}
	MsgUpdateMemoryTDAIFailed     = Key{string: "更新 memory_tdai_enable 失败"}
	MsgFailedToCreateTask         = Key{string: "创建任务失败"}
)

// TDAI Pro 切换任务 handler (task/tdai_handler_pro.go)
var (
	MsgTDAIHermesInstallFailed     = Key{string: "安装 Hermes 插件失败"}
	MsgTDAIPluginReadyCheckFailed  = Key{string: "插件就绪检查失败"}
	MsgTDAISDKInitFailed           = Key{string: "初始化 Agent Memory SDK 失败"}
	MsgTDAIQueryMemoryProFailed    = Key{string: "查询 Memory Pro 实例失败"}
	MsgTDAINoOnlineMemoryPro       = Key{string: "无可用的 online 状态 Memory Pro 实例（总数=%d）"}
	MsgTDAICreateMemSpaceFailed    = Key{string: "创建记忆库失败"}
	MsgTDAIPersistBindingFailed    = Key{string: "落库绑定信息失败"}
	MsgTDAISwitchProFailed         = Key{string: "switch_pro 失败"}
	MsgTDAIUpdatePlanFailed        = Key{string: "更新 plan 状态失败"}
	MsgTDAIMigrateScriptFailed     = Key{string: "migrate 脚本执行失败"}
	MsgTDAIVDBInfoIncomplete       = Key{string: "VDB endpoint/username/apiKey 不完整"}
	MsgTDAIVDBPrecheckScriptFailed = Key{string: "VDB 连通性预检脚本调用失败"}
	MsgTDAIDeleteMemSpaceFailed    = Key{string: "释放记忆库失败"}

	// task/tdai_handler_switch.go
	MsgTDAIEnableFreePluginFailed = Key{string: "启用 Free 插件失败"}
	MsgTDAIDisablePluginFailed    = Key{string: "禁用插件失败"}
	MsgTDAIReleaseProMemFailed    = Key{string: "释放 Pro 记忆库失败（本地绑定信息已保留）"}
	MsgTDAIQueryPluginRowFailed   = Key{string: "查询实例 %s plugin 行失败"}
	MsgTDAIQueryInstanceFailed    = Key{string: "查询实例 %s 失败，等待重试"}
	MsgTDAIInstanceNotFound       = Key{string: "实例 %s 未找到，等待重试"}
	MsgTDAIInstanceNotReady       = Key{string: "实例 %s 尚未就绪（当前状态: %s），等待重试"}
)

// 内存插件升级 (admin_memory_plugin_upgrade.go)
var (
	MsgOnlyPostMethodSupported        = Key{string: "仅支持 POST"}
	MsgPluginUpgradeQueryFail         = Key{string: "查询失败"}
	MsgMemPluginInstanceNotPro        = Key{string: "实例当前非 Pro 版"}
	MsgMemPluginSwitchInProgress      = Key{string: "有进行中的切换操作（%s）"}
	MsgMemPluginInstanceRecordMissing = Key{string: "实例记录不存在"}
	MsgMemPluginAgentTypeUnsupported  = Key{string: "该类型实例不支持插件升级"}
)

// OneID 回调 (auth_oneid.go)
var (
	MsgOneIDMissingCode         = Key{string: "缺少 code 参数"}
	MsgOneIDUnauthorized        = Key{string: "未授权"}
	MsgOneIDMissingLogoutToken  = Key{string: "缺少 logout_token 参数"}
	MsgOneIDInvalidLogoutToken  = Key{string: "logout_token 无效"}
	MsgInternalLoginNotConfig   = Key{string: "内部登录未配置"}
	MsgInternalLoginMissToken   = Key{string: "缺少 token 参数"}
	MsgInternalLoginTokenInval  = Key{string: "token 无效或已过期"}
	MsgInternalLoginMissSub     = Key{string: "token 无效：缺少 sub 字段"}
	MsgInternalLoginCreateUser  = Key{string: "创建用户失败"}
	MsgInternalLoginRestoreUser = Key{string: "恢复用户失败"}
)

// Agent Bridge 回调 (agent_bridge_callback.go)
var (
	MsgABInvalidProxyToken    = Key{string: "无效的 proxy token"}
	MsgABOrphanProxyToken     = Key{string: "Proxy token 关联的用户不存在"}
	MsgABUserBanned           = Key{string: "用户已被封禁"}
	MsgABCredsNotConfigured   = Key{string: "云凭证未配置"}
	MsgABSTSRefreshFailed     = Key{string: "STS 刷新失败"}
	MsgABQueryInstancesFailed = Key{string: "查询用户实例失败"}
	MsgABResourceIdMismatch   = Key{string: "resource_id 与绑定实例不匹配"}
)

// 规则集（ruleset_helpers.go）
var (
	MsgRuleSetCreateConflict           = Key{string: "另一个请求正在创建规则组，请稍后重试"}
	MsgRuleSetCheckManagedSGFail       = Key{string: "检查托管安全组失败"}
	MsgRuleSetCannotImportFromCP       = Key{string: "不允许从 ClawPro 自建安全组导入"}
	MsgRuleSetReadSourceSGFail         = Key{string: "读取源安全组规则失败"}
	MsgRuleSetSourceSGIDRequired       = Key{string: "source_sg_id 不能为空"}
	MsgRuleSetNameInvalid              = Key{string: "规则组名称不合法：需 3~32 个字符，必须以字母开头，仅允许字母、数字、短横线（当前=%q）"}
	MsgImportCheckManagedSGFail        = Key{string: "导入时检查托管安全组失败"}
	MsgImportCannotImportFromManagedSG = Key{string: "不允许从 ClawPro 自建安全组导入（会造成循环依赖）"}
	MsgImportCreateVpcClientFail       = Key{string: "导入时创建 VPC 客户端失败"}
	MsgImportDescribeSGPoliciesFail    = Key{string: "导入时查询源安全组规则失败"}
	MsgImportSourceSGRulesEmpty        = Key{string: "源安全组规则为空"}
	MsgImportAutoCreateRuleSetFailed   = Key{string: "导入时自动创建规则组失败"}
)

// 实例迁移 (openclaw_migration.go)
var (
	MsgMigrationInitUploadFailed        = Key{string: "初始化上传失败"}
	MsgMigrationCreateRecordFailed      = Key{string: "创建迁移记录失败"}
	MsgMigrationGenScriptFailed         = Key{string: "生成迁移脚本失败"}
	MsgMigrationCheckFileFailed         = Key{string: "检查文件状态失败"}
	MsgMigrationNoExportRecord          = Key{string: "未找到有效的迁移导出记录，请先运行导出脚本"}
	MsgMigrationFileNotUploaded         = Key{string: "迁移文件尚未上传，请先在源实例终端运行导出脚本"}
	MsgMigrationSetOpLockFailed         = Key{string: "设置迁移操作锁失败"}
	MsgMigrationStarted                 = Key{string: "迁移已开始"}
	MsgMigrationLoadRestoreScriptFailed = Key{string: "加载恢复脚本失败"}
	MsgMigrationLoadExportScriptFailed  = Key{string: "加载迁移导出脚本失败"}
	MsgMigrationRestoreAgentFailed      = Key{string: "恢复 agent 数据失败"}
	MsgMigrationWaitAgentReadyTimeout   = Key{string: "等待 agent 就绪超时"}
)

// 实例（openclaw.go） — 创建/查询场景
var (
	MsgCreateUserVPCFailed             = Key{string: "创建用户 VPC 失败"}
	MsgCreateUserSubnetFailed          = Key{string: "创建用户子网失败"}
	MsgUserDataDisabled                = Key{string: "UserData 功能未开启，请联系管理员在后台开启"}
	MsgGenerateProxyTokenFailed        = Key{string: "生成代理 Token 失败"}
	MsgNoPermCreateInstanceType        = Key{string: "您无权创建 %s 类型的实例，请联系管理员"}
	MsgNoImageForTypeContactAdmin      = Key{string: "管理员尚未为 %s 类型配置生效镜像，请联系管理员"}
	MsgResolveVisibleImageTypesFailed  = Key{string: "解析当前用户可见的镜像类型失败"}
	MsgQueryRestrictedImageTypesFailed = Key{string: "查询受限镜像类型列表失败"}
	MsgQueryInstanceStateFailed        = Key{string: "查询实例状态失败"}
	MsgInstanceQuotaReached            = Key{string: "当前可创建的实例数量已达上限，请联系管理员调整。"}
	MsgInstanceGroupQuotaReached       = Key{string: "当前分组可创建的实例数量已达上限，请联系管理员调整。"}
	MsgCVMConfigIncomplete             = Key{string: "CVM 配置不完整，请联系管理员"}
	MsgCVMRegionNotConfigured          = Key{string: "未配置 CVM Region，请联系管理员"}
	MsgPickSubnetByIPFailed            = Key{string: "按可用 IP 挑选子网失败"}
	MsgVPCConfiguredWithoutSubnet      = Key{string: "已配置 VPC 但未配置任何子网，请联系管理员"}
	MsgCreateDefaultVPCFailed          = Key{string: "创建默认 VPC/子网失败"}
	MsgDefaultVPCNoUsableSubnet        = Key{string: "默认 VPC 无可用子网，请联系管理员"}
	MsgClawProSGNotConfigured          = Key{string: "当前 ClawPro 未配置安全组，请联系企业管理员配置安全组后重试。"}
	MsgSGCapacityFull                  = Key{string: "安全组容量已满，请联系管理员"}
	MsgSGAllocationFailed              = Key{string: "安全组分配失败，请稍后重试"}
	MsgRebootInstanceFailed            = Key{string: "重启实例失败"}
	MsgMissingCodeParam                = Key{string: "缺少参数 code"}
	MsgAgentTypeNotSupportCheck        = Key{string: "该 agent_type (%s) 不支持 check_service"}
	MsgTerminalFeatureDisabled         = Key{string: "终端功能未开启，请联系管理员"}
	MsgQueryCVMInstanceFailed          = Key{string: "查询 CVM 实例失败"}
	MsgQueryZonesFailed                = Key{string: "查询可用区失败"}
	MsgNoEnabledImage                  = Key{string: "未启用任何镜像，无法重装实例"}
	MsgImageAgentTypeMismatch          = Key{string: "指定镜像的 agent_type 与请求不匹配"}
)

var (
	MsgCVMAPITimeout = Key{string: "CVM API 调用超时"}
)

var (
	MsgUnexpectedScript = Key{string: "脚本 %s 不匹配"}
)

// admin_instances.go 补充键
var (
	MsgInstanceNoAssociatedCVM         = Key{string: "无关联 CVM"}
	MsgPowerOnFailed                   = Key{string: "开机失败"}
	MsgPowerOnStarted                  = Key{string: "开机已下发"}
	MsgShutdownStarted                 = Key{string: "关机已下发"}
	MsgRebootFailed                    = Key{string: "重启失败"}
	MsgReinstallFailedShort            = Key{string: "重装失败"}
	MsgCVMInstanceNotExist             = Key{string: "CVM 实例 %s 不存在"}
	MsgMaxDetectInstances50            = Key{string: "单次最多检测 50 个实例"}
	MsgMissingIDInstanceIDIDsParam     = Key{string: "缺少参数 id、instance_id、ids 或 instance_ids"}
	MsgInstanceAgentNotReadyForVersion = Key{string: "实例 Agent 未就绪，无法拉取版本信息"}
	MsgPullVersionInfoFailed           = Key{string: "版本信息拉取失败"}
	MsgReadUpdatedInstanceFailed       = Key{string: "读取更新后的实例失败"}
	MsgBatchUpgradeMax20               = Key{string: "单次批量升级最多支持 20 个实例"}
	MsgMissingIdsOrInstanceIdsParam    = Key{string: "缺少参数 ids 或 instance_ids"}
	MsgNoValidInstanceFound            = Key{string: "未找到任何有效实例"}
	MsgQueryEnabledImageFailed         = Key{string: "查询启用镜像失败"}
	MsgNoDefaultImage                  = Key{string: "无法获取默认镜像，请先在后台开启一个镜像"}
	MsgInstanceNoCVMCannotUpgrade      = Key{string: "实例 ID=%d 无关联的 CVM，无法升级"}
	MsgNonOfficialInstancesCannotBatch = Key{string: "存在无法获取的实例信息，无法批量升级：%s"}
	MsgUserGroupIDsOrGroupIDsRequired  = Key{string: "user_group_ids 和 group_ids 至少传一个"}
	MsgQueryGroupSubtreeFailed         = Key{string: "查询分组子树失败"}
	MsgQueryGroupMembersFailed         = Key{string: "查询分组成员失败"}
	MsgPlaceholderInstanceCleaned      = Key{string: "占位实例已清理"}
	MsgCreateCVMClientFailedFmt        = Key{string: "创建 CVM 客户端失败: %v"}
	MsgDestroyCVMInstanceFailedFmt     = Key{string: "销毁 CVM 实例失败: %v"}
	MsgDestroyDispatchedAsyncCleanup   = Key{string: "已下发销毁，异步清理进行中"}
	MsgCVMNotExistLocalCleaned         = Key{string: "CVM 已不存在，本地记录已清理"}
	MsgNoEnabledImageForType           = Key{string: "暂无生效的%s类型镜像，请先在后台为该类型启用镜像"}
	MsgInstanceCannotGetCVMInfoFmt     = Key{string: "%s(ID=%d, 无法获取CVM信息)"}
	MsgInstanceOperationInProgress     = Key{string: "实例正在进行 %s 操作，跳过"}
	MsgCheckUpgradeStatusFailedFmt     = Key{string: "检查升级状态失败: %v"}
	MsgSetUpgradeOpLockFailedFmt       = Key{string: "设置升级操作锁失败: %v"}
	MsgQueryInstanceDeniedOpsFailed    = Key{string: "查询实例禁用操作失败"}
	MsgUpgradeInvalidModelProvider     = Key{string: "实例配置文件 openclaw.json 中 models.providers 存在非法 key（不能包含 %s）：%s，请先修复配置后再升级"}
	// 查询过滤参数
	MsgTagValuesCountExceed = Key{string: "tag_values 数量超过上限 %d"}
	MsgTagKeysCountExceed   = Key{string: "tag_keys 数量超过上限 %d"}
	MsgIDsCountExceed       = Key{string: "ids 数量超过上限 %d"}
	// 内部查询 / 解析
	MsgQueryLightInstancesFailed = Key{string: "查询轻量实例列表失败"}
	MsgQueryInstancesByIDsFailed = Key{string: "通过 instance_ids 查询实例失败"}
	// 批量删除参数校验
	MsgIDsEmptyList              = Key{string: "ids 不能为空列表"}
	MsgIDsContainZeroOrDuplicate = Key{string: "ids 不能全部为 0 或重复"}
	MsgInstanceIdsEmptyList      = Key{string: "instance_ids 不能为空列表"}
	MsgInstanceIdsNotFound       = Key{string: "instance_ids 对应的实例不存在"}
	MsgMissingIDOrInstanceID     = Key{string: "缺少参数 id 或 instance_id"}
	// 云 API 内部调用
	MsgSerializeRequestParamsFailed   = Key{string: "序列化请求参数失败"}
	MsgParseResponseFailed            = Key{string: "解析响应失败"}
	MsgCallGenerateAuthLoginURLFailed = Key{string: "调用 GenerateAuthLoginUrl 失败"}
	MsgLoginURLEmpty                  = Key{string: "返回的 LoginUrl 为空"}
	MsgAPIErrorFormat                 = Key{string: "API 错误 [%s]: %s"}
)

// CVM 规格升配与系统盘扩容。
var (
	MsgAdjustmentInvalidEnvelope             = Key{string: "资源调整请求格式错误"}
	MsgAdjustmentIDsExclusive                = Key{string: "ids 与 instance_ids 必须且只能提供一种"}
	MsgAdjustmentTargetCount                 = Key{string: "资源调整目标数量必须为 1 到 100"}
	MsgAdjustmentInvalidType                 = Key{string: "adjustment_type 必须为 instance_type 或 system_disk"}
	MsgAdjustmentMissingInstanceType         = Key{string: "规格调整必须提供 target_instance_type"}
	MsgAdjustmentMissingDiskSize             = Key{string: "系统盘扩容必须提供 target_system_disk_size"}
	MsgAdjustmentInvalidResizeMode           = Key{string: "resize_mode 必须为 online 或 offline"}
	MsgAdjustmentCloudUnavailable            = Key{string: "云资源检查暂时不可用，请稍后重试"}
	MsgAdjustmentReasonInternalError         = Key{string: "内部异常"}
	MsgAdjustmentReasonCloudRequired         = Key{string: "仅云端实例支持资源调整"}
	MsgAdjustmentReasonDoctorNode            = Key{string: "龙虾医生节点不允许调整资源"}
	MsgAdjustmentReasonOperationInProgress   = Key{string: "实例已有操作正在进行"}
	MsgAdjustmentReasonStatus                = Key{string: "当前实例状态不支持资源调整"}
	MsgAdjustmentReasonCVMNotFound           = Key{string: "腾讯云实例不存在"}
	MsgAdjustmentReasonCVMRestricted         = Key{string: "腾讯云实例已被限制或隔离"}
	MsgAdjustmentReasonCVMOperation          = Key{string: "腾讯云实例正在执行其他操作"}
	MsgAdjustmentReasonCVMQuery              = Key{string: "查询腾讯云实例信息失败"}
	MsgAdjustmentReasonStopCharging          = Key{string: "关机停止计费实例暂不支持资源调整"}
	MsgAdjustmentReasonInvalidTarget         = Key{string: "调整目标无效"}
	MsgAdjustmentReasonUnsupportedType       = Key{string: "目标实例规格不受支持"}
	MsgAdjustmentReasonNotUpgrade            = Key{string: "目标规格必须高于当前规格且属于同一 AI2 升配序列"}
	MsgAdjustmentReasonInstanceTypeUnchanged = Key{string: "已是目标规格"}
	MsgAdjustmentReasonInstanceTypeDowngrade = Key{string: "不支持降配"}
	MsgAdjustmentReasonCloudDisk             = Key{string: "实例包含不受支持的本地盘或非云盘"}
	MsgAdjustmentReasonDiskType              = Key{string: "当前系统盘类型不支持该调整"}
	MsgAdjustmentReasonTypeUnavailable       = Key{string: "目标规格在当前可用区或计费模式下不可售"}
	MsgAdjustmentReasonDiskQuota             = Key{string: "当前系统盘配置没有可用扩容配额"}
	MsgAdjustmentReasonChargeType            = Key{string: "当前实例或磁盘计费类型不受支持"}
	MsgAdjustmentReasonDiskNotReady          = Key{string: "系统盘未挂载、归属异常或正在执行其他操作"}
	MsgAdjustmentReasonCloudDiskUnavailable  = Key{string: "腾讯云云硬盘资源暂不可用"}
	MsgAdjustmentReasonNetwork               = Key{string: "当前网络配置与目标规格不兼容"}
	MsgAdjustmentReasonResourceLimit         = Key{string: "实例关联的弹性网卡、公网 IP 或带宽配置超过目标规格限制"}
	MsgAdjustmentReasonResourceQuota         = Key{string: "账号资源配额不足，暂时无法完成本次调整"}
	MsgAdjustmentReasonImage                 = Key{string: "当前镜像不支持目标规格"}
	MsgAdjustmentReasonFeature               = Key{string: "当前实例特性不支持此资源调整"}
	MsgAdjustmentReasonPromotion             = Key{string: "促销或活动实例不支持此资源调整"}
	MsgAdjustmentReasonDiskSize              = Key{string: "目标系统盘容量不符合扩容范围或步长"}
	MsgAdjustmentReasonDiskMustGrow          = Key{string: "目标容量需大于当前系统盘容量"}
	MsgAdjustmentReasonDiskShrink            = Key{string: "不支持缩容"}
	MsgAdjustmentReasonOnlineResize          = Key{string: "当前系统盘不支持在线扩容"}
	MsgAdjustmentReasonBalance               = Key{string: "腾讯云账户余额不足"}
	MsgAdjustmentReasonUnpaidOrder           = Key{string: "腾讯云账户存在未支付订单"}
	MsgAdjustmentReasonSoldOut               = Key{string: "目标云资源已售罄"}
	MsgAdjustmentReasonCloudFailed           = Key{string: "腾讯云未能执行资源调整"}
	MsgAdjustmentReasonTimeout               = Key{string: "资源调整执行超时"}
	MsgAdjustmentReasonRestoreFailed         = Key{string: "资源调整后恢复实例原状态失败"}
)

// 产品动态 (controller/product_news.go)
var (
	MsgProductNewsAPICallFailed = Key{string: "调用 DescribeClawProProductNews 失败"}
	MsgProductNewsAPIError      = Key{string: "云 API 返回错误: %s - %s"}
)

// SSRF 安全校验 (controller/provider/provider.go)
var (
	MsgSSRFURLParseError      = Key{string: "解析 URL 错误: %v"}
	MsgSSRFSchemeNotAllowed   = Key{string: "scheme %q 不允许访问"}
	MsgSSRFHostEmpty          = Key{string: "Host 为空"}
	MsgSSRFInternalAddress    = Key{string: "地址 %s 为内部地址"}
	MsgSSRFNoMappedIP         = Key{string: "Host %q 无映射 IP 地址"}
	MsgSSRFResolvedToInternal = Key{string: "Host %q 解析到内部地址 %s"}
	MsgSSRFCannotResolve      = Key{string: "%q 无法解析为可访问的 IP 地址"}
)

// admin_security_group.go 补充键
var (
	MsgSGQuerySGFailed               = Key{string: "查询安全组失败"}
	MsgSGNameRequired                = Key{string: "安全组名称不能为空"}
	MsgSGCreateSGFailed              = Key{string: "创建安全组失败"}
	MsgSGSecurityGroupIDRequired     = Key{string: "security_group_id 不能为空"}
	MsgSGAlreadyBound                = Key{string: "该安全组已是当前使用的安全组，无需重复绑定"}
	MsgSGNotExist                    = Key{string: "安全组不存在"}
	MsgSGCheckRulesFailed            = Key{string: "检查安全组规则失败"}
	MsgSGAutoFixRulesFailed          = Key{string: "自动补齐安全组规则失败"}
	MsgSGNotConfiguredCreateFirst    = Key{string: "未配置安全组，请先创建或绑定安全组"}
	MsgSGModifySGFailed              = Key{string: "修改安全组失败"}
	MsgSGMissingSecurityGroupIDParam = Key{string: "缺少参数 security_group_id"}
	MsgSGPoliciesRequired            = Key{string: "缺少必填参数 policies"}
	MsgSGCreateSGRulesFailed         = Key{string: "创建安全组规则失败"}
	MsgSGReplaceSGRulesFailed        = Key{string: "替换安全组规则失败"}
	MsgSGDeleteSGRulesFailed         = Key{string: "删除安全组规则失败"}
	MsgSGIDFormatInvalid             = Key{string: "安全组 ID 格式错误: %s，应为 sg-xxxxxxxx 格式"}
	MsgSGQuerySGListFailed           = Key{string: "查询安全组列表失败"}
	MsgDoctorNodeNotAllowedRemoveIDs = Key{string: "龙虾医生节点不允许该操作：请从列表中移除实例 ID %v"}
	MsgUserGroupPairsExceedLimit     = Key{string: "展开后的 (user_id, group_id) 对数量超过上限 %d（实际 %d），请缩小查询范围"}

	// 内部 helper 错误
	MsgAddIngressSGRulesFailed   = Key{string: "添加缺失入站安全组规则失败（已重试 %d 次）"}
	MsgAddEgressSGRulesFailed    = Key{string: "添加缺失出站安全组规则失败（已重试 %d 次）"}
	MsgQueryInstanceSGListFailed = Key{string: "查询实例安全组列表失败"}

	// HandleReorderRuleSetRules 相关
	MsgReorderFingerprintsRequired = Key{string: "ordered_fingerprints 不能为空"}
	MsgReorderFingerprintsEmptyStr = Key{string: "ordered_fingerprints 含空字符串"}
	MsgReorderDuplicateFingerprint = Key{string: "ordered_fingerprints 含重复 fingerprint: %s"}
	MsgRuleSetNotFound             = Key{string: "规则组不存在 (name=%q)"}
	MsgRuleSetQueryFailed          = Key{string: "查询规则组失败"}
	MsgRuleSetParseRulesFailed     = Key{string: "解析现有规则失败"}
	MsgRuleSetEmptyNoReorder       = Key{string: "规则组当前无任何规则，无需排序"}
	MsgReorderFingerprintNotFound  = Key{string: "fingerprint 不存在于规则组中: %s"}
	MsgReorderFailed               = Key{string: "安全组规则重排序失败，所有变更已回滚"}
)

// openclaw.go 补充键
var (
	MsgMultiAgentParseResultFailed       = Key{string: "multi-agent 查询结果解析失败"}
	MsgMultiAgentQueryFailed             = Key{string: "multi-agent 查询失败"}
	MsgQueryGlobalVPCFailed              = Key{string: "查询全局 VPC 失败"}
	MsgVPCNotExistContactAdmin           = Key{string: "私有网络 VPC [%s] 不存在，请联系管理员更新网络配置"}
	MsgQueryGlobalSubnetFailed           = Key{string: "查询全局子网失败"}
	MsgSubnetNotExistContactAdmin        = Key{string: "私有网络 VPC [%s] 下可用区子网 [%s] 不存在，请联系管理员更新子网配置"}
	MsgAcquireDistributedLockFailed      = Key{string: "获取分布式锁失败"}
	MsgRegionNotConfiguredOrNoZone       = Key{string: "未配置 region 或 region %q 无可用区"}
	MsgVerifyDefaultVPCQueryFailed       = Key{string: "验证默认 VPC 是否存在时查询失败"}
	MsgQueryDefaultVPCFailed             = Key{string: "查询默认 VPC 失败"}
	MsgCreateDefaultVPCDataError         = Key{string: "创建默认 VPC 返回数据异常"}
	MsgQueryVPCSubnetsFailed             = Key{string: "查询 VPC 已有子网失败"}
	MsgCreateSubnetFailed                = Key{string: "创建子网 %s (zone=%s, cidr=%s) 失败"}
	MsgCreateSubnetDataError             = Key{string: "创建子网 %s 返回数据异常"}
	MsgSerializeSubnetMapFailed          = Key{string: "序列化子网映射失败"}
	MsgPersistVPCConfigFailed            = Key{string: "持久化 VPC/子网配置失败"}
	MsgLockUserRecordFailed              = Key{string: "锁定用户记录失败"}
	MsgCreatePlaceholderRecordFailed     = Key{string: "创建占位记录失败"}
	MsgQueryDefaultTagsFailed            = Key{string: "查询默认标签失败"}
	MsgLoadScriptFailed                  = Key{string: "加载 %s 失败"}
	MsgParseTemplateFailed               = Key{string: "解析 %s 模板失败"}
	MsgRenderTemplateFailed              = Key{string: "渲染 %s 模板失败"}
	MsgMergeUserDataFailed               = Key{string: "合并系统 UserData 与用户 multipart 失败"}
	MsgUserDataExceedSize                = Key{string: "user_data 不能超过 %dKB（base64 字符串长度）"}
	MsgUserDataInvalidBase64             = Key{string: "user_data 必须是合法的 base64 字符串"}
	MsgUserDataMultipartBoundaryNotFound = Key{string: "user_data 声明了 multipart 但内容中找不到 boundary %q，请检查格式是否符合 MIME 规范"}
	MsgUserDataExceedCVMSize             = Key{string: "UserData base64 编码后不能超过 %dKB（CVM 限制）"}
	MsgParseContentTypeFailed            = Key{string: "无法解析 Content-Type"}
	MsgMultipartMissingBoundary          = Key{string: "multipart 缺少 boundary 参数"}
	MsgCreateSecurityGroupDataError      = Key{string: "创建安全组返回数据异常"}
	MsgPlatformCapacityLimited           = Key{string: "当前平台扩容能力受限，请联系管理员进行提交工单处理。"}
	MsgPlatformCapacityLimitedAdmin      = Key{string: "当前平台扩容能力受限，请提交工单处理。"}
	MsgDeleteCapacityLimited             = Key{string: "当前删除能力受限，请联系管理员进行提交工单处理。"}
	MsgDeleteCapacityLimitedAdmin        = Key{string: "当前删除能力受限，可能由实例相关配额不足导致，请提交工单处理。"}
	MsgDeleteAbortProMemFailed           = Key{string: "CVM 已不存在，但远端 Pro 记忆库释放失败，已中止删除，请稍后重试"}
	MsgDeleteRetainProMemFailed          = Key{string: "CVM 已不存在，但远端 Pro 记忆库释放失败，已保留实例记录等待自动补偿，请稍后重试"}
	MsgCreateCVMInstanceFailed           = Key{string: "创建实例失败"}
	MsgTerminateCVMInstanceFailed        = Key{string: "退还 CVM 实例失败"}
	MsgCreateCapacityLimited             = Key{string: "当前创建能力受限，请联系管理员进行提交工单处理。"}
	MsgCreateCapacityLimitedAdmin        = Key{string: "当前创建能力受限，可能由实例相关配额不足导致，请提交工单处理。"}
)

// admin_notices.go (config steps)
var (
	MsgConfigStepBrand         = Key{string: "设置平台名称与品牌"}
	MsgConfigStepDefaultQuota  = Key{string: "配置用户默认配额"}
	MsgConfigStepUsers         = Key{string: "导入企业用户"}
	MsgConfigStepSSOLogin      = Key{string: "设置用户登录方式"}
	MsgConfigStepModel         = Key{string: "配置至少一个模型"}
	MsgConfigStepChannel       = Key{string: "配置至少一个通道"}
	MsgConfigStepVPC           = Key{string: "配置私有网络"}
	MsgConfigStepSecurityGroup = Key{string: "配置安全组"}
	MsgConfigStepImage         = Key{string: "配置至少一个镜像"}
)

// instance_state.go (fallback status)
var (
	MsgInstanceStatusMaintaining        = Key{string: "维护中"}
	MsgInstanceStatusMaintainingTooltip = Key{string: "状态异常，请联系管理员"}
)

// cloud_quota.go (alert messages and labels)
var (
	MsgQuotaAlertVPCExceeded        = Key{string: "您在%s地域的私有网络可创建数量已达上限，用户端将无法创建 Agent，请提交腾讯云工单申请提升配额，"}
	MsgQuotaAlertSubnetIPExhausted  = Key{string: "当前子网的可用IP已耗尽，用户端将无法创建 Agent，请在网络管理更换有可用IP的子网，"}
	MsgQuotaAlertSGInstanceExceeded = Key{string: "您的安全组 %s 关联云服务器实例数已达上限，用户端将无法创建 Agent，请提交腾讯云工单申请提升配额，"}
	MsgQuotaAlertAccountOverdue     = Key{string: "您的腾讯云账号 %s 已欠费，将影响用户端 Agent 的正常创建，请尽快充值，"}
	MsgQuotaAlertActionGoToHandle   = Key{string: "前往处理"}
)

// usergroup/admin_query.go
var (
	MsgGroupDeleteNonBlockingMembersNote = Key{string: "成员不阻塞删除；只属于此组的用户将变为游离用户。"}
	MsgGroupDeleteHintNoBlockers         = Key{string: "此组当前无阻塞项，可直接删除。"}
	MsgGroupDeleteHintHasInstances       = Key{string: "该分组下存在直属创建的 Agent，请先迁移或销毁这些 Agent 再重试。"}
	MsgGroupDeleteHintHasOtherBlockers   = Key{string: "请先解除资源绑定 / 删除子分组，再重试。"}
)

// 记忆分组策略 (admin_memory_group_policy.go)
var (
	MsgGroupPolicyPriorityMustBe1Or2        = Key{string: "priority 必须是 1 或 2"}
	MsgGroupPolicyPlanMustBeOffFreePro      = Key{string: "plan 必须是 off/free/pro"}
	MsgGroupPolicyPlanSameAsDefault         = Key{string: "分组策略的 plan 不能与预设策略相同（当前预设=%s）"}
	MsgGroupPolicyPriorityAlreadyExists     = Key{string: "priority=%d 已存在分组策略，请使用 PUT 修改"}
	MsgGroupPolicyGroupIDsCannotBeEmpty     = Key{string: "group_ids 不能为空"}
	MsgGroupPolicyPartialGroupIDNotExist    = Key{string: "部分 group_id 不存在"}
	MsgGroupPolicyPartialGroupOccupied      = Key{string: "部分分组已被其他策略占用"}
	MsgGroupPolicyPriorityNotExist          = Key{string: "priority=%d 不存在分组策略，请先 POST 创建"}
	MsgGroupPolicyPlanUsedByOther           = Key{string: "plan=%s 已被策略 %d 使用"}
	MsgGroupPolicyUpdateFailedMaybeOccupied = Key{string: "操作失败（可能部分分组已被其他策略占用）"}
)

// 用户组 model 哨兵错误 (model/user_group.go)
var (
	MsgUgLimitExceeded                       = Key{string: "已达到用户组数量上限（1000 个）"}
	MsgUgNotFound                            = Key{string: "用户组不存在"}
	MsgUgMemberLimitReached                  = Key{string: "目标用户组成员数量已达上限（10000 人）"}
	MsgUgAddMemberWouldExceed                = Key{string: "添加后成员总数将超过上限（10000 人）"}
	MsgUgInvalidUserID                       = Key{string: "存在不合法的用户 ID"}
	MsgUgInvalidName                         = Key{string: "分组名非法（空、含 '/'、或超过长度限制）"}
	MsgUgNameConflict                        = Key{string: "同父分组下已存在同名分组"}
	MsgUgMaxDepthExceeded                    = Key{string: "分组层级超过上限（10 层）"}
	MsgUgFullPathTooLong                     = Key{string: "分组全路径长度超过 512"}
	MsgUgParentCycleDetected                 = Key{string: "换父形成循环：新父组是自身或自身后代"}
	MsgUgManualCannotUnderOneIDDept          = Key{string: "manual 分组不允许挂到 oneid_dept 分组下"}
	MsgUgOneIDDeptReadonly                   = Key{string: "oneid_dept 分组为只读，不可修改"}
	MsgUgHasDependencies                     = Key{string: "分组存在依赖（资源绑定 / manual 子组 / 直属 Agent），无法删除"}
	MsgUgToBeDeletedReadonly                 = Key{string: "分组处于待删除状态，不可操作"}
	MsgUgNotSelectable                       = Key{string: "分组不可选择（不在用户直属且未失效）"}
	MsgRebindTargetGroupNotInUserGroups      = Key{string: "目标分组 %d 不在你当前所在的分组列表 %v 中，无法自迁"}
	MsgRebindTargetGroupZeroButUserHasGroups = Key{string: "你已加入分组 %v，target_group_id 必须是其中之一（不能为 0）"}
	MsgRebindNotPendingUserAction            = Key{string: "该实例当前不需要你处理分组归属（未标记 pending_user_action），无法自迁"}
	MsgRebindNotAllowedByAdmin               = Key{string: "管理员未允许你自行迁移该实例的分组（缺少 allow_migrate 标），请联系管理员或改用移交"}
	// 用户端同组移交（POST /openclaw/stale-instances/initiate）业务规则错误
	MsgHandoverInstanceUngrouped        = Key{string: "该实例当前未分组，无法发起同组移交（同组移交要求实例已归属某个分组）"}
	MsgHandoverNotAllowedByAdmin        = Key{string: "管理员未允许你对该实例发起同组移交（缺少 allow_same_group_handover 标），请联系管理员或改用自迁"}
	MsgHandoverTargetUserNotFound       = Key{string: "目标用户 %q 不存在"}
	MsgHandoverTargetIsSelf             = Key{string: "target_username 不能是你自己"}
	MsgHandoverTargetNotInInstanceGroup = Key{string: "目标用户 %q 不在实例当前所属分组内，无法接收同组移交"}
	MsgHandoverNoActiveHandover         = Key{string: "该实例当前不需要你处理移交"}
	MsgHandoverCancelNotOwner           = Key{string: "只有实例的原持有人 %q 才能取消移交（你是 %q）"}
	MsgHandoverAcceptNotTarget          = Key{string: "只有移交的目标用户 %q 才能接受此移交（你是 %q）"}
	MsgHandoverRejectNotTarget          = Key{string: "只有移交的目标用户 %q 才能拒绝此移交（你是 %q）"}
	MsgUgParentNotFound                 = Key{string: "父分组不存在"}
	MsgUgRecomputeFullPathFailed        = Key{string: "重算全路径: 更新分组 %d 失败"}
	// DB 操作包装
	MsgUgDBQueryFailed  = Key{string: "查询用户组失败"}
	MsgUgDBCreateFailed = Key{string: "创建用户组失败"}
	MsgUgDBUpdateFailed = Key{string: "更新用户组失败"}
	MsgUgDBDeleteFailed = Key{string: "删除用户组失败"}
)

// 出站诊断 (controller/egress_diagnostic.go)
var (
	MsgEgressInstanceIDEmpty    = Key{string: "instanceID 为空"}
	MsgEgressQuerySGRulesFailed = Key{string: "查询安全组 %s 规则失败"}
	MsgEgressDiagnoseTimeout    = Key{string: "诊断超时"}
	MsgEgressBlocked            = Key{string: "出站流量被拒绝，请联系管理员修改安全组出站规则"}
)

// 用户管理 (admin_users.go)
var (
	// 创建/校验
	MsgPasswordEncryptFailed        = Key{string: "密码加密失败"}
	MsgUserLimitReached             = Key{string: "已达到用户数上限（%d）"}
	MsgUsernameExists               = Key{string: "创建失败：用户名已存在"}
	MsgCreateUserGenTokenFailed     = Key{string: "创建失败：生成 API Token 失败"}
	MsgCreateUserDBError            = Key{string: "创建失败：数据库错误"}
	MsgCreateUserReadFailed         = Key{string: "创建失败：读取新用户记录失败"}
	MsgUserGroupMembershipSetFailed = Key{string: "用户组归属设置失败"}
	MsgCreateUserGroupFailed        = Key{string: "用户组创建失败"}
	MsgCreateUserGroupMemberFailed  = Key{string: "创建用户组成员失败"}
	MsgDeleteUserGroupMemberFailed  = Key{string: "删除用户组成员失败"}
	MsgSelectUserGroupMemberFailed  = Key{string: "查找用户组成员失败"}
	MsgUserGroupMemberAlreadyExists = Key{string: "用户组成员已存在"}
	// 批量导入
	MsgGroupIDsFormatError       = Key{string: "group_ids 格式错误：应为分号分隔的用户组路径字符串（如 \"}根组/研发一组;根组/研发二组\"）"}
	MsgBatchUsernameMustBeString = Key{string: "用户名必须为字符串"}
	MsgBatchPasswordMustBeString = Key{string: "密码必须为字符串"}
	MsgBatchRoleMustBeString     = Key{string: "角色必须为字符串"}
	MsgBatchEmailMustBeString    = Key{string: "邮箱必须为字符串"}
	MsgBatchFieldFormatError     = Key{string: "用户字段格式错误"}
	MsgBatchRequestBodyJSONError = Key{string: "请求体 JSON 格式错误：%v"}
	MsgBatchQueryGroupFailed     = Key{string: "查询用户组失败：%v"}
	MsgBatchGroupPathNotFound    = Key{string: "以下用户组路径不存在：%v"}
	// OneID 统一账号
	MsgOneIDCreateUserFailed     = Key{string: "OneID 创建用户失败: %v"}
	MsgOneIDAddRoleFailed        = Key{string: "OneID 添加角色失败: %v"}
	MsgOneIDDisableUserFailed    = Key{string: "OneID 停用用户失败: %v"}
	MsgOneIDDeleteUserFailed     = Key{string: "OneID 删除用户失败: %v"}
	MsgOneIDEnableUserFailed     = Key{string: "OneID 启用用户失败: %v"}
	MsgOneIDResetPasswordFailed  = Key{string: "OneID 重置密码失败: %v"}
	MsgOneIDUpdateUserFailed     = Key{string: "OneID 更新用户失败: %v"}
	MsgOneIDRemoveRoleFailed     = Key{string: "OneID 移除角色失败: %v"}
	MsgOneIDSyncDepartmentFailed = Key{string: "OneID 同步部门失败: %v"}
	MsgOneIDSyncUserDeptFailed   = Key{string: "OneID 同步用户部门失败: %v"}
	// 内部 helper (toAdminJSON)
	MsgQueryUserGroupInfoFailed          = Key{string: "查询用户组信息失败"}
	MsgQueryMemberRelationFailed         = Key{string: "查询成员关系失败"}
	MsgQueryUserGroupInstanceCountFailed = Key{string: "查询用户分组实例数量失败"}
	// VPC 资源查询
	MsgQueryVPCResourceFailed = Key{string: "查询 VPC 资源失败"}
	MsgQueryENIFailed         = Key{string: "查询 ENI 失败"}
	MsgDeleteVPCFailed        = Key{string: "删除 VPC 失败"}
	// Token 启用/禁用动作词
	MsgTokenEnableAction  = Key{string: "启用"}
	MsgTokenDisableAction = Key{string: "禁用"}
	// SMH 存储客户端 (cos.go)
	MsgSmhClientNotInitialized               = Key{string: "SMH 客户端未初始化"}
	MsgSmhSpaceTokenNotInitialized           = Key{string: "SMH Space Token 未初始化"}
	MsgSmhNotConfiguredCheckLog              = Key{string: "SMH 未配置，请检查 hatchery 启动初始化 SMH 日志"}
	MsgSmhSkillhubSpaceNotConfiguredCheckLog = Key{string: "SMH 技能空间未配置，请检查 hatchery 启动初始化 SMH 日志"}
	MsgSmhSkillhubTokenNotInitialized        = Key{string: "SMH skillhub Token 未初始化"}
	MsgSmhNotConfigured                      = Key{string: "SMH 未配置"}
	MsgSmhCommonSpaceNotConfigured           = Key{string: "SMH common 空间未配置"}
	MsgSmhCommonTokenNotInitialized          = Key{string: "SMH common Token 未初始化"}
	MsgSmhUsageQueryEmpty                    = Key{string: "SMH 使用量查询返回空"}
	MsgSmhNotConfiguredCannotUpload          = Key{string: "SMH 未配置，无法上传备份"}
	MsgSmhMultipartUploadIncomplete          = Key{string: "SMH MultipartUploadFile 返回数据不完整"}
	MsgSmhCredCannotBeEmpty                  = Key{string: "cred 不能为空"}
	MsgSmhConfirmKeyEmptyCannotRenew         = Key{string: "ConfirmKey 为空，无法续期"}
	MsgSmhRenewMultipartUploadIncomplete     = Key{string: "SMH RenewMultipartUpload 返回数据不完整"}
	MsgSmhRenewMultipartUploadNoExpiration   = Key{string: "SMH RenewMultipartUpload 未能获取凭证到期时间"}
	MsgSmhConfirmKeyCannotBeEmpty            = Key{string: "confirmKey 不能为空"}
	MsgSmhDirPathCannotBeEmpty               = Key{string: "SMH 目录路径不能为空"}
	MsgSmhInstanceIdCannotBeEmpty            = Key{string: "instanceId 不能为空"}
	MsgSmhCreateTokenFailed                  = Key{string: "SMH CreateToken 失败 (space=%s request_id=%s)"}
	MsgSmhRemoveEnvFailed                    = Key{string: "卸载 SMH 环境失败 (cvm=%s agent_type=%s)"}
	MsgSmhCreateTokenEmpty                   = Key{string: "SMH CreateToken 返回空 (space=%s request_id=%s)"}
	MsgSmhTokenRefreshFailed                 = Key{string: "SMH Token 重新获取失败 (space=%s)"}
	MsgSmhTokenNotReady                      = Key{string: "SMH Token 仍未就绪 (space=%s)"}
	MsgSmhCreateParentDirFailed              = Key{string: "SMH 创建父目录失败"}
	MsgSmhCreateDirFailed                    = Key{string: "SMH 创建目录失败 (dir=%s request_id=%s)"}
	MsgSmhQueryUsageFailed                   = Key{string: "查询 SMH 空间使用量失败 (request_id=%s)"}
	MsgSmhParseAvailableSpaceFailed          = Key{string: "解析 SMH 可用空间失败"}
	MsgSmhCheckCapacityFailed                = Key{string: "检查 SMH 空间容量失败"}
	MsgSmhInvalidArchiveSize                 = Key{string: "备份包大小无效，无法上传"}
	MsgSmhInsufficientSpace                  = Key{string: "SMH 空间剩余容量不足 (需要 %d 字节，剩余 %d 字节)"}
	MsgSmhGetTokenFailed                     = Key{string: "获取 SMH Token 失败"}
	MsgSmhMultipartUploadFailed              = Key{string: "SMH MultipartUploadFile 失败 (request_id=%s)"}
	MsgSmhRenewMultipartUploadFailed         = Key{string: "SMH RenewMultipartUpload 失败 (request_id=%s)"}
	MsgSmhConfirmUploadFailed                = Key{string: "SMH ConfirmUpload 失败 (request_id=%s)"}
	MsgSmhDeleteFileFailed                   = Key{string: "SMH 删除文件失败 (fileKey=%s request_id=%s)"}
	MsgSmhDeleteDirFailed                    = Key{string: "SMH 删除目录失败 (dirKey=%s request_id=%s)"}
	MsgSmhInfoFileFailed                     = Key{string: "SMH InfoFile 失败 (fileKey=%s request_id=%s)"}
	MsgSmhListDirFailed                      = Key{string: "SMH ListDirectory 失败 (dir=%s request_id=%s)"}

	// SMH 开通与运维 (smh.go)
	MsgSmhCreateTokenFailedWithSpace   = Key{string: "SMH CreateToken 失败 (space=%s, request_id=%s): %v"}
	MsgSmhCreateTokenEmptyWithSpace    = Key{string: "SMH CreateToken 返回空 (space=%s, request_id=%s)"}
	MsgSmhDescribeLibrariesFailed      = Key{string: "DescribeLibraries 失败: %v"}
	MsgSmhDescribeLibrariesNotFound    = Key{string: "DescribeLibraries: 库 %s 未找到"}
	MsgSmhLibraryNotProvisioned        = Key{string: "SMH 库尚未开通"}
	MsgSmhCreateAPIClientFailed        = Key{string: "创建 SMH 云 API 客户端失败: %v"}
	MsgSmhDescribeLibraryFailed        = Key{string: "describeLibrary 失败: %v"}
	MsgSmhModifyLibrarySearchFailed    = Key{string: "修改库以启用搜索失败: %v"}
	MsgSmhCreateLibraryAPIError        = Key{string: "创建库失败 (requestId=%s): %s - %s"}
	MsgSmhCreateSpaceSDKFailed         = Key{string: "smh create space (libraryId=%s, tag=%s, requestId=%s): %v"}
	MsgSmhCreateSpaceMissingSpaceId    = Key{string: "smh create space: response missing spaceId (libraryId=%s, tag=%s, requestId=%s)"}
	MsgSmhSetUpdateLibParamsFailed     = Key{string: "设置 UpdateLibraryInternal 参数失败: %v"}
	MsgSmhUpdateLibFailed              = Key{string: "UpdateLibraryInternal 失败: %v"}
	MsgSmhParseUpdateLibRespFailed     = Key{string: "解析 UpdateLibraryInternal 响应失败: %v"}
	MsgSmhUpdateLibAPIError            = Key{string: "更新库失败 (requestId=%s): %s - %s"}
	MsgSmhSetUpdateSpaceParamsFailed   = Key{string: "设置 UpdateSpaceInternal 参数失败: %v"}
	MsgSmhUpdateSpaceFailed            = Key{string: "UpdateSpaceInternal 失败: %v"}
	MsgSmhParseUpdateSpaceRespFailed   = Key{string: "解析 UpdateSpaceInternal 响应失败: %v"}
	MsgSmhUpdateSpaceAPIError          = Key{string: "更新空间失败 (requestId=%s): %s - %s"}
	MsgSmhCreateQuotaSDKFailed         = Key{string: "smh create quota (libraryId=%s, spaceId=%s, requestId=%s): %v"}
	MsgSmhDeleteSpaceSDKFailed         = Key{string: "smh delete space (libraryId=%s, spaceId=%s, requestId=%s): %v"}
	MsgSmhPersonalSpaceCreateFailed    = Key{string: "创建 SMH 个人空间失败: %v"}
	MsgSmhSetFreeQuotaFailed           = Key{string: "设置免费配额失败: %v"}
	MsgSmhSetQuotaLimitFailed          = Key{string: "设置配额上限失败: %v"}
	MsgSmhGetUsageSDKFailed            = Key{string: "smh GetUsage 失败 (requestId=%s): %v"}
	MsgSmhGetUsageEmpty                = Key{string: "smh GetUsage 返回空 (requestId=%s)"}
	MsgSmhGetLibUsageSDKFailed         = Key{string: "smh GetLibraryUsage 失败 (requestId=%s): %v"}
	MsgSmhGetLibUsageEmpty             = Key{string: "smh GetLibraryUsage 返回空 (requestId=%s)"}
	MsgSmhAcquireDistLockFailed        = Key{string: "获取分布式锁失败: %v"}
	MsgSmhCheckPersonalSpaceFailed     = Key{string: "检查个人空间是否存在失败: %v"}
	MsgSmhSavePersonalSpaceFailed      = Key{string: "写入个人空间记录失败: %v"}
	MsgSmhGetTokenFailedSpace          = Key{string: "获取 token 失败 (space=%s): %v"}
	MsgSmhPersonalTokenFailed          = Key{string: "获取 token 失败: %v"}
	MsgSmhResolveInitEnvScriptFailed   = Key{string: "解析 init_smh_env 脚本失败 (agent_type=%s): %v"}
	MsgSmhInitEnvFailed                = Key{string: "初始化 SMH 环境失败 (cvm=%s, agent_type=%s): %v"}
	MsgSmhUpdateEnvStatusFailed        = Key{string: "更新 SMH 环境状态失败: %v"}
	MsgSmhResolveRemoveEnvScriptFailed = Key{string: "解析 remove_smh_env 脚本失败 (agent_type=%s): %v"}
	MsgSmhUninstallEnvFailed           = Key{string: "卸载 SMH 环境失败 (cvm=%s, agent_type=%s): %v"}
	MsgSmhResolveSetTokenScriptFailed  = Key{string: "解析 set_smh_token 脚本失败 (agent_type=%s): %v"}
	MsgSmhInjectEnvVarFailed           = Key{string: "注入 SMH 环境变量失败: %v"}
)

// 记忆 TDAI 模型 (model/memory_tdai.go)
var (
	MsgMemoryTDAIVersionsMustBeJSONArray    = Key{string: "memory_tdai_supported_versions 必须为 JSON 字符串数组"}
	MsgMemoryTDAIVersionsCannotContainEmpty = Key{string: "memory_tdai_supported_versions 不能包含空字符串"}
	MsgMemoryTDAIVersionsNormalizeFailed    = Key{string: "memory_tdai_supported_versions 规范化失败"}
)

// Drain Worker (task/drain_worker.go)
var (
	MsgDrainUpdateInstanceSG       = Key{string: "更新实例安全组失败"}
	MsgDrainIncrementCVMCount      = Key{string: "增加目标安全组CVM计数失败"}
	MsgDrainDecrementCVMCount      = Key{string: "减少冻结安全组CVM计数失败: %s"}
	MsgDrainClearState             = Key{string: "清理Drain状态失败"}
	MsgDrainFilterManagedInstances = Key{string: "过滤托管实例失败"}
)

// SG Guardian (task/sg_guardian_task.go)
var (
	MsgSGDescribeNilResponse       = Key{string: "查询安全组返回空响应"}
	MsgSGCreateCloudRetryExhausted = Key{string: "创建云安全组重试已耗尽"}
	MsgSGApplyRulesRetryExhausted  = Key{string: "下发规则重试已耗尽"}
)

// SG RuleSet Init Task (task/sg_ruleset_init_task.go)
var (
	MsgSGRulesetInitPreCheck        = Key{string: "预检查规则集失败"}
	MsgSGRulesetInitAcquireLock     = Key{string: "获取SG初始化锁失败"}
	MsgSGRulesetInitPostLockCheck   = Key{string: "锁后检查规则集失败"}
	MsgSGRulesetInitDescribeOldBase = Key{string: "查询旧基准安全组 %s 失败"}
	MsgSGRulesetInitFailed          = Key{string: "SG规则集初始化失败"}

	// SG RuleSet Init Task 通知文案（task/sg_ruleset_init_task.go）
	MsgSGMigrationNoticeTitle   = Key{string: "ClawPro 安全组独立化升级已完成"}
	MsgSGMigrationNoticeMessage = Key{string: "ClawPro 安全组独立化升级已完成，原规则与绑定 Agent 已迁移至 ClawPro-Default"}
	MsgSGInitFailureTitle       = Key{string: "安全组初始化失败，请检查云配额"}
	MsgSGInitFailureMessage     = Key{string: "ClawPro 安全组初始化失败：%s。请前往腾讯云控制台检查安全组配额并释放或提升配额，系统将每 30 秒自动重试。"}
	MsgSGInitRecoveryTitle      = Key{string: "安全组初始化已恢复"}
	MsgSGInitRecoveryMessage    = Key{string: "ClawPro 安全组初始化已自动重试成功，安全组功能恢复正常。"}
)

// SMH 自动开通 (task/smh_auto_provision.go)
var (
	MsgSMHProvisionPanic = Key{string: "SMH开通发生异常: %v"}
)

// TDAI Dispatcher (task/tdai_dispatcher.go)
var (
	MsgTDAIUnknownJobType = Key{string: "未知任务类型: %s"}
)

// 租户管理 (admin_tenant.go)
var (
	MsgTenantIdentifierExists          = Key{string: "租户标识已存在"}
	MsgTenantDomainAlreadyMapped       = Key{string: "域名 %s 已被映射"}
	MsgTenantCannotRemovePrimaryDomain = Key{string: "不能删除主域名"}
	MsgTenantInitFailed                = Key{string: "租户初始化失败"}
	MsgTenantCreateDomainFailed        = Key{string: "创建租户域名失败"}
	MsgTenantCreateSiteConfigFailed    = Key{string: "创建站点配置失败"}
	MsgTenantCreateAdminFailed         = Key{string: "创建管理员失败"}
	MsgTenantDomainRequired            = Key{string: "至少需要一个域名"}
	MsgTenantQueryDomainsFailed        = Key{string: "查询域名列表失败"}
)

// Access Log (controller/access_log.go)
var (
	MsgHijackNotSupported = Key{string: "底层 ResponseWriter 不支持 Hijack"}
)

// 智能体类型 (controller/admin_agent_types.go)
var ()

// CLS 采集范围 (controller/admin_cls_scope.go)
var (
	MsgCheckCLSScopeForInstanceFailed = Key{string: "检查实例 CLS 采集范围失败"}
	MsgMarkInstanceCLSPendingFailed   = Key{string: "标记实例 CLS 待安装失败"}
)

// 内存插件升级 (controller/admin_memory_plugin_upgrade.go)
var (
	MsgPluginUpgradePanic           = Key{string: "插件升级发生异常: %v"}
	MsgPluginUpgradeParseJSONFailed = Key{string: "解析版本 JSON 失败"}
)

// 内存 Pro 管理 (controller/admin_memory_pro.go)
var (
	MsgVPCNotConfigured    = Key{string: "未配置 VPC，请先在管控端配置网络"}
	MsgSubnetNotConfigured = Key{string: "未配置子网，请先在管控端配置网络"}
)

// 云配额告警 (controller/cloud_quota.go)
var (
	MsgUnmarshalBillingRespFailed = Key{string: "解析 CheckAccountBalance 响应失败"}
	MsgBillingAPIError            = Key{string: "计费 API 错误: %s - %s"}
	MsgSetActionParamsFailed      = Key{string: "设置接口请求参数失败"}
	MsgSendCheckBalanceFailed     = Key{string: "调用 CheckAccountBalance 失败"}
)

// OneID 同步 (controller/oneid_sync.go)
var (
	MsgOneIDGwBuildRequestFailed   = Key{string: "构建 Gateway 请求失败"}
	MsgOneIDGwReturnedError        = Key{string: "Gateway 返回错误: %d %s"}
	MsgOneIDGwDataScopeFailed      = Key{string: "app_data_scope 失败"}
	MsgOneIDGwParseDataScopeFailed = Key{string: "解析 app_data_scope 响应失败"}
	MsgOneIDGwDataScopeCodeError   = Key{string: "app_data_scope 返回错误码 %d: %s"}
)

// 插件路径探测 (controller/plugin_path.go)
var (
	MsgMemoryPluginNotFoundOnCVM = Key{string: "CVM 上未找到记忆插件（三个路径均不存在）"}
)

// 邮件发送 (controller/email.go)
var (
	MsgEmailAPIURLNotConfigured = Key{string: "未配置 --email-api-url，无法发送邮件"}
	MsgEmailInvalidAPIURL       = Key{string: "无效的邮件 API URL %q: %v"}
	MsgEmailMarshalParamsFailed = Key{string: "序列化邮件参数失败"}
	MsgEmailGetCredFailed       = Key{string: "获取凭据失败，无法发送邮件"}
	MsgEmailMarshalActionFailed = Key{string: "构建邮件接口参数失败"}
	MsgEmailSendFailed          = Key{string: "发送邮件失败"}
	MsgEmailAPIError            = Key{string: "邮件 API 错误: %s - %s"}
)

// 内存预检 (controller/memory_precheck.go)
var (
	MsgMemoryPrecheckNoFallbackCred = Key{string: "无可用的 PRO 实例凭证做 fallback 预检"}
)

// SMH 开通 (controller/smh.go)
var (
	MsgSMHProvisionLockFailed       = Key{string: "获取 SMH 开通锁失败"}
	MsgCreateSMHClientFailed        = Key{string: "创建 SMH 云 API 客户端失败"}
	MsgSMHLibraryEmptyAccessDomain  = Key{string: "媒体库 %s 的 AccessDomain 为空"}
	MsgSMHDescribeSecretFailed      = Key{string: "查询媒体库密钥失败"}
	MsgSMHSecretResponseEmpty       = Key{string: "查询媒体库密钥返回空"}
	MsgSMHCreateSpaceFailed         = Key{string: "创建 SMH 空间 %s 失败"}
	MsgSMHUpdateSpaceInternalFailed = Key{string: "更新 SMH 空间内部配置 %s 失败"}
	MsgSMHCreateSpaceQuotaFailed    = Key{string: "创建 SMH 空间配额 %s 失败"}
	MsgSMHUpsertSpaceFailed         = Key{string: "持久化 SMH 空间 %s 失败"}
	MsgSMHSetCreateLibParamsFailed  = Key{string: "设置 CreateLibrary 参数失败"}
	MsgSMHCreateLibraryFailed       = Key{string: "创建媒体库失败"}
	MsgSMHParseCreateLibRespFailed  = Key{string: "解析 CreateLibrary 响应失败"}
	MsgSMHCreateLibRespMissing      = Key{string: "CreateLibrary 返回缺少 LibraryId 或 AccessDomain"}
	MsgSMHUpdateLibInternalFailed   = Key{string: "更新媒体库内部配置失败"}
	MsgSMHPersistLibraryFailed      = Key{string: "持久化媒体库到 SiteConfig 失败"}
)

// TDAI 任务 (model/tdai_job.go)
var (
	MsgTDAIJobRetryFailed  = Key{string: "任务 %d 不存在或状态不是 FAILED"}
	MsgTDAIJobCancelFailed = Key{string: "任务 %d 不存在或状态不是 PENDING"}
)

// 分布式锁 (model/distlock.go)
var (
	MsgDistLockGetSQLDBFailed    = Key{string: "获取 sql.DB 失败"}
	MsgDistLockAcquireConnFailed = Key{string: "获取数据库连接失败"}
	MsgDistLockNameTooLong       = Key{string: "锁名称 %q 超过 64 字符 (%d)"}
	MsgDistLockGETLockFailed     = Key{string: "GET_LOCK(%q) 失败"}
	MsgDistLockGETLockNullErr    = Key{string: "GET_LOCK(%q) 内部错误 (NULL)"}
	MsgDistLockGETLockTimeout    = Key{string: "GET_LOCK(%q) 超时 (%v)"}
	MsgDistLockISFreeLockFailed  = Key{string: "IS_FREE_LOCK(%q) 失败"}
	MsgDistLockISFreeLockNullErr = Key{string: "IS_FREE_LOCK(%q) 内部错误 (NULL)"}
)

// AI 模型 (model/ai_model.go)
var (
	MsgAICountBuiltinModelFailed = Key{string: "统计内置模型数量失败"}
	MsgAISeedBuiltinModelFailed  = Key{string: "写入内置模型占位记录失败"}
)

// 策略解析 (controller/usergroup/resolve.go)
var (
	MsgResolvePolicyValueFailed   = Key{string: "解析策略值失败 key=%s"}
	MsgResolvePolicyBindingFailed = Key{string: "查询策略绑定失败 key=%s"}
	MsgResolveInvalidUint         = Key{string: "无效的数字: %s"}
)

// TDAI Memory SDK (internal/tdaimemorysdk/client.go)
var (
	MsgTDAISDKClientNotInit        = Key{string: "SDK 客户端未初始化"}
	MsgTDAISDKConvertToMapFailed   = Key{string: "转换请求为 map 失败"}
	MsgTDAISDKSendRequestFailed    = Key{string: "发送请求失败"}
	MsgTDAISDKEmptyResponseBody    = Key{string: "响应体为空"}
	MsgTDAISDKDecodeEnvelopeFailed = Key{string: "解析响应信封失败"}
	MsgTDAISDKMissingResponseField = Key{string: "响应缺少 Response 字段"}
	MsgTDAISDKDecodeMetaFailed     = Key{string: "解析响应元信息失败"}
	MsgTDAISDKDecodePayloadFailed  = Key{string: "解析响应内容失败"}
)

// AI 通道 (model/ai_channel.go)
var (
	MsgSeedChannelFailed = Key{string: "初始化通道 %s 失败"}
)

// 规则组 (model/rule_set.go)
var (
	MsgLockRuleSetFailed = Key{string: "锁定规则组 %s 失败"}
)

// 租户域名 (model/tenant_domain.go)
var (
	MsgUnknownDomain        = Key{string: "未知域名: %s"}
	MsgTenantConfigNotFound = Key{string: "租户配置不存在: %s"}
)

// TAT (controller/tat.go, controller/tat_batch.go)
var (
	MsgTATUnrecognizedTimeFormat = Key{string: "无法识别的 TAT 时间格式: %q"}
)

// TDAI Memory SDK sentinels (internal/tdaimemorysdk/errors.go)
var (
	MsgTDAISDKSecretIDRequired  = Key{string: "secret_id 不能为空"}
	MsgTDAISDKSecretKeyRequired = Key{string: "secret_key 不能为空"}
	MsgTDAISDKActionRequired    = Key{string: "action 不能为空"}
)

// SG Guardian sentinel (task/sg_guardian_task.go)
var (
	MsgSGGoneInCloud             = Key{string: "SG 在云端已删除"}
	MsgSGCloudRuleDescribeFailed = Key{string: "查询云端 SG 规则失败"}
)

// 本地 agent (clawpro 一期，controller/local_agent.go + 其它路径)
var (
	MsgLocalAgentInvalidAgentType       = Key{string: "agent_type 无效，仅支持 workbuddy / codebuddy"}
	MsgLocalAgentInvalidLocalAgentID    = Key{string: "local_agent_id 必须为 16 位 hex"}
	MsgLocalAgentNotAllowed             = Key{string: "本地 Agent 功能未开放"}
	MsgLocalGetConfigCredentialNotReady = Key{string: "CLS 凭据未配置，请联系管理员"}
	MsgLocalAgentRuleNotImplemented     = Key{string: "规范（rule）分发功能本期未实现"}
	MsgLocalOnlyEndpoint                = Key{string: "该接口仅支持本地实例"}
	MsgLocalInstanceUnsupportedOp       = Key{string: "本地实例不支持此操作"}
	MsgLocalInstanceNotConnected        = Key{string: "本地实例未接入"}
	MsgLocalPendingSkillAlreadySuc      = Key{string: "已成功安装的技能不允许从待处理列表删除"}
	MsgLocalInstanceTargetUnsupported   = Key{string: "部分目标为本地 Agent 实例，不支持下发命令：%s"}
)

var (
	MsgFailedToGetDeleteImpact = Key{string: "查询删除影响报告失败"}

	// ---- 企业规范库（admin/instances/rules）----
	MsgQueryRuleInstallationFailed  = Key{string: "查询规范安装记录失败"}
	MsgQueryRuleFailed              = Key{string: "查询规范失败"}
	MsgRuleNotLocalInstance         = Key{string: "该实例不是本地实例，不支持规范管理"}
	MsgRuleRecordNotFound           = Key{string: "规范下发记录不存在"}
	MsgRuleRecordNotPendingOrFailed = Key{string: "只能删除待处理或失败的规范下发记录"}
	MsgRuleUpdateFail               = Key{string: "更新规范失败: %v"}
	MsgRuleSlugVersionRequired      = Key{string: "slug 和 version 为必填参数"}
	MsgRuleTasksQueryFail           = Key{string: "查询规范下发任务失败"}
	MsgRuleInstancesQueryFail       = Key{string: "查询规范安装实例失败"}
	MsgRuleDistributeFailed         = Key{string: "下发规范失败: %v"}
	MsgRuleNoValidInstance          = Key{string: "没有符合条件的本地实例"}
	MsgRuleUninstallFailed          = Key{string: "卸载规范失败: %v"}
	MsgRuleDistributeOK             = Key{string: "已下发，请等待客户端拉取"}
	MsgRuleUninstallOK              = Key{string: "已卸载，请等待客户端拉取"}
	MsgSkillNewVersionDistributed   = Key{string: "已下发新的版本"}
)

// stale-instances/config-diff 展示文案（i18n 化，避免代码中直接返回中文）
var (
	// 已分配/未分配 — 用于公网 IP 是否分配的显示
	MsgLabelAllocated    = Key{string: "已分配"}
	MsgLabelNotAllocated = Key{string: "未分配"}

	// 实例计费模式展示（对应 CVM InstanceChargeType）
	MsgInstanceChargeTypePrepaidLabel        = Key{string: "包年包月"}
	MsgInstanceChargeTypePostpaidByHourLabel = Key{string: "按量计费"}

	// 公网带宽计费模式展示（对应 CVM InternetAccessible.InternetChargeType）
	MsgInternetChargeBandwidthPostpaidLabel = Key{string: "按带宽计费"}
	MsgInternetChargeTrafficPostpaidLabel   = Key{string: "按流量计费"}
	MsgInternetChargeBandwidthPackageLabel  = Key{string: "带宽包"}
	MsgInternetChargeBandwidthPrepaidLabel  = Key{string: "包年包月带宽"}
)

// stale-instances 通知标题（title）
var (
	NotifTitleAgentMigrated             = Key{string: "Agent 已迁移至新组织"}
	NotifTitleAgentHandoverByAdmin      = Key{string: "Agent 已被管理员移交"}
	NotifTitleAgentReceivedFromAdmin    = Key{string: "管理员为您移入了一个 Agent"}
	NotifTitleAgentPendingUserAction    = Key{string: "您有一个 Agent 需要处理"}
	NotifTitleAgentArchivedByAdmin      = Key{string: "您的 Agent 已被管理员归档关机"}
	NotifTitleAgentOrgConfigUpdated     = Key{string: "您的 Agent 组织配置已更新"}
	NotifTitleHandoverInitiated         = Key{string: "已发起 Agent 移交"}
	NotifTitleHandoverReceived          = Key{string: "您收到一个 Agent 移交请求"}
	NotifTitleHandoverCancelled         = Key{string: "已取消 Agent 移交"}
	NotifTitleHandoverCancelledByOther  = Key{string: "对方取消了 Agent 移交请求"}
	NotifTitleHandoverAccepted          = Key{string: "Agent 移交成功"}
	NotifTitleHandoverAcceptedReceived  = Key{string: "已接收 Agent"}
	NotifTitleHandoverRejected          = Key{string: "Agent 移交被拒绝"}
	NotifTitleHandoverRejectedConfirmed = Key{string: "已拒绝 Agent 移交请求"}
)

// stale-instances 通知正文（message），支持 %s 格式化占位符
// 文案严格对齐产品文档 A/B/C 三类通知规范
var (
	// A1: migrate — 管理端场景 A/B/C（3 args: 原组织全路径, 实例名, 新组织全路径）
	NotifMsgMigratedByAdmin = Key{string: "由于您已不在原组织「%s」，管理员已将您在原组织下创建的 Agent「%s」迁移至「%s」，平台策略会立即应用新组织配置（包括您可创建的 Agent 数量上限、您的单用户 Tokens 上限、功能权限等），其他已配置项保留不变。"}
	// migrate — 管理端场景 D（1 arg: 实例名）
	NotifMsgMigratedScenarioD = Key{string: "由于您原组织的上级组织发生变更，平台策略已按新组织配置重新生效，您的 Agent「%s」的已配置项保留不变。"}
	// B2: migrate — 用户端自迁完成（2 args: 实例名, 新组织全路径）
	NotifMsgMigratedByUser = Key{string: "您的 Agent「%s」已迁移至「%s」，平台策略已应用新组织配置（包括您可创建的 Agent 数量上限、您的单用户 Tokens 上限、功能权限等），其他已配置项保留不变，已为您自动开机。"}

	// A2: handover — 管理端，通知原 owner（3 args: 原组织全路径, 实例名, 对方账号）
	NotifMsgHandoverByAdminToOwner = Key{string: "由于您已不在原组织「%s」，管理员已将您在原组织下创建的 Agent「%s」移交给 %s，该 Agent 不再归属于您。"}
	// handover — 管理端，通知目标用户（产品文档未覆盖，1 arg: 实例名）
	NotifMsgHandoverByAdminToTarget = Key{string: "管理员已将 Agent「%s」移交给您，该 Agent 现已归属于您，平台策略将按您当前所在组织配置立即生效。"}

	// A3: pending_user — 允许迁移 + 允许移交（2 args: 原组织全路径, 实例名）
	NotifMsgPendingUserBoth = Key{string: "由于您已不在原组织「%s」，您在原组织下创建的 Agent「%s」需要您选择迁移至新组织或移交给原组织其他用户，请前往「我的 Agent」处理。"}
	// A3 变体: pending_user — 仅允许迁移
	NotifMsgPendingUserMigrateOnly = Key{string: "由于您已不在原组织「%s」，您在原组织下创建的 Agent「%s」需要您选择迁移至新组织，请前往「我的 Agent」处理。"}
	// A3 变体: pending_user — 仅允许移交
	NotifMsgPendingUserHandoverOnly = Key{string: "由于您已不在原组织「%s」，您在原组织下创建的 Agent「%s」需要您选择移交给原组织其他用户，请前往「我的 Agent」处理。"}

	// A4: archive_stop（2 args: 原组织全路径, 实例名）
	NotifMsgArchivedByAdmin = Key{string: "由于您已不在原组织「%s」，管理员已将您在原组织下创建的 Agent「%s」保留和关机，如需恢复开机使用请联系管理员。"}

	// A5: OneID 同步 / 父分组变更 — 按用户聚合（2 args: 新组织全路径, 实例名列表）
	NotifMsgOrgConfigUpdated = Key{string: "由于您原组织的上级组织发生变更，变为新组织「%s」，您在原组织下创建的 Agent「%s」已自动迁移至新组织。"}

	// C1-1: handover initiate — 通知发起方（2 args: 对方账号, 实例名）
	NotifMsgHandoverInitiatedToInitiator = Key{string: "您已向 %s 发起 Agent「%s」的移交，待对方确认接收，移交期间 Agent 保持关机状态。"}
	// C2-1: handover initiate — 通知接收方（2 args: 对方账号, 实例名）
	NotifMsgHandoverInitiatedToTarget = Key{string: "%s 向您移交了 Agent「%s」，请前往「我的 Agent」选择确认接收或拒绝。"}

	// C1-5: handover cancel — 通知发起方（2 args: 实例名, 对方账号）
	NotifMsgHandoverCancelledToInitiator = Key{string: "您已取消 Agent「%s」对 %s 的移交，请您前往「我的 Agent」继续处理。"}
	// C2-5: handover cancel — 通知接收方（2 args: 对方账号, 实例名）
	NotifMsgHandoverCancelledToTarget = Key{string: "%s 已取消向您移交 Agent「%s」。"}

	// C1-2: handover accept — 通知原 owner（2 args: 对方账号, 实例名）
	NotifMsgHandoverAcceptedToOwner = Key{string: "%s 已确认接收 Agent「%s」，移交成功，该 Agent 已转移到对方的 Agent 列表。"}
	// C2-2: handover accept — 通知接收方（2 args: 对方账号, 实例名）
	NotifMsgHandoverAcceptedToReceiver = Key{string: "您已确认接收来自 %s 的 Agent「%s」，移交成功，已配置项保留不变，已为您自动开机。"}

	// C1-4: handover reject — 通知原 owner（2 args: 对方账号, 实例名）
	NotifMsgHandoverRejectedToOwner = Key{string: "%s 已拒绝接收 Agent「%s」，移交失败，请您前往「我的 Agent」继续处理。"}
	// C2-4: handover reject — 通知拒绝方（2 args: 对方账号, 实例名）
	NotifMsgHandoverRejectedToRejecter = Key{string: "您已拒绝接收来自 %s 的 Agent「%s」。"}
)

// stale-instances apply 错误码对应的 i18n 文案（完整句子）
// 内部仍用 snake_case 错误码传递，在 recordResult 前通过 translateApplyError 翻译。
var (
	MsgStaleErrUnsupportedAction                 = Key{string: "不支持的操作类型"}
	MsgStaleErrLoadUserGroupsFailed              = Key{string: "查询用户分组信息失败"}
	MsgStaleErrActionNotAllowedInScenario        = Key{string: "当前场景下不允许执行此操作"}
	MsgStaleErrTargetGroupNotFound               = Key{string: "目标分组不存在"}
	MsgStaleErrTargetUserNotFound                = Key{string: "目标用户不存在"}
	MsgStaleErrTargetGroupIDRequired             = Key{string: "迁移操作必须指定目标分组"}
	MsgStaleErrTargetGroupNotInUserGroups        = Key{string: "目标分组不在用户当前所属分组中"}
	MsgStaleErrTargetGroupIDMustZeroForUngrouped = Key{string: "用户已无任何分组，目标分组必须为 0（回退到未分组）"}
	MsgStaleErrTargetUserIDRequired              = Key{string: "移交操作必须指定目标用户"}
	MsgStaleErrTargetUserSameAsOwner             = Key{string: "目标用户与实例当前所有者相同，无法移交给本人"}
	MsgStaleErrLoadTargetUserGroupsFailed        = Key{string: "查询目标用户分组信息失败"}
	MsgStaleErrTargetUserNoGroupsMustZero        = Key{string: "目标用户无任何分组，目标分组 ID 必须为 0（回退到未分组）"}
	MsgStaleErrTargetGroupNotInTargetUserGroups  = Key{string: "指定的目标分组不在目标用户的分组列表中"}
	MsgStaleErrTargetGroupRequiredForMultiGroup  = Key{string: "目标用户属于多个分组，必须显式指定目标分组"}
	MsgStaleErrTargetUserNotInSameGroup          = Key{string: "目标用户不在当前实例所属的分组中，无法进行同分组移交"}
	MsgStaleErrAtLeastOneSubOptionRequired       = Key{string: "至少需要选择一个子选项（允许迁移或允许移交）"}
)

// 管控端创建 Agent (controller/admin_instance_create.go)
var (
	MsgAdminCreateGroupInvalid           = Key{string: "所选分组不属于目标用户或已失效"}
	MsgAdminCreateModelDuplicate         = Key{string: "初始模型不能重复"}
	MsgAdminCreateModelUnavailable       = Key{string: "模型 %d 不存在、未启用或对目标分组不可见"}
	MsgAdminCreateChannelDuplicate       = Key{string: "初始通道不能重复"}
	MsgAdminCreateChannelConfigInvalid   = Key{string: "通道配置键和值不能为空"}
	MsgAdminCreateSkillSourceUnsupported = Key{string: "初始技能 source 仅支持 public 或 enterprise"}
	MsgAdminCreateSkillUnavailable       = Key{string: "技能 %s 不存在"}
	MsgAdminCreateSkillNotVisible        = Key{string: "技能 %s 对目标分组不可见"}
)

// 定时任务通知消息
var (
	// Agent 创建成功通知
	MsgInstanceCreateSuccessTitle   = Key{string: "实例创建成功"}
	MsgInstanceCreateSuccessMessage = Key{string: "您的实例「%s」已创建成功，可以开始使用了。"}

	// Agent 重装完成通知
	MsgInstanceReinstallSuccessTitle   = Key{string: "实例重装完成"}
	MsgInstanceReinstallSuccessMessage = Key{string: "您的实例「%s」已重新安装完成。"}

	// 服务重启中断（recover_tasks.go）
	MsgTaskInterruptedByRestart = Key{string: "服务重启中断"}

	// 定时任务调度（agent_command_schedule.go）
	MsgScheduleSkippedPreviousRunning = Key{string: "上一轮执行未完成，本次已跳过"}

	// 龙虾医生服务（doctor_cleanup.go）
	MsgDoctorServiceUsername = Key{string: "龙虾医生服务"}

	// TDAI 任务引擎不可重试错误（tdai_handler_pro.go / tdai_handler_switch.go）
	MsgTDAIProBindingIncomplete      = Key{string: "实例已在 PRO 但绑定信息不完整，请联系管理员排查"}
	MsgTDAIMemoryProNotEnabled       = Key{string: "未开通 Memory Pro 服务，请先在管控端开通"}
	MsgTDAIVDBInfoIncompleteNonRetry = Key{string: "VDB endpoint/username/apiKey 不完整，无法预检连通性"}
	MsgTDAIVDBNetworkUnreachable     = Key{string: "CVM 与 VDB 网络不通，无法切换到 Pro: endpoint=%s, probe_output=%s"}
	MsgTDAIAgentTypeNotSupportMemory = Key{string: "实例 %s 的类型 %q 不支持记忆功能"}
)
