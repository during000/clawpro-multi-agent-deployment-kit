package controller

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	hcommon "hatchery/common"
	"hatchery/i18n"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tchttp "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/http"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

// cloudProxyService 定义一个腾讯云产品的代理配置
type cloudProxyService struct {
	Service string // 产品标识，如 "cvm"、"vpc"、"tat"
	Version string // API 版本，如 "2017-03-12"
	// Endpoint 可选，为空则自动使用 <service>.<region>.tencentcloudapi.com
	Endpoint string
	// ReadActions 为允许透传的只读 Action 白名单（Describe/Inquiry 等查询类接口）
	ReadActions map[string]bool
	// WriteActions 为允许透传的写操作 Action 白名单（Create/Delete/Start/Stop 等变更类接口）
	WriteActions map[string]bool
}

// allActions 返回该 service 所有允许的 Actions（读+写）
func (s *cloudProxyService) allActions() map[string]bool {
	all := make(map[string]bool, len(s.ReadActions)+len(s.WriteActions))
	for a := range s.ReadActions {
		all[a] = true
	}
	for a := range s.WriteActions {
		all[a] = true
	}
	return all
}

// cloudProxyRegistry 注册所有允许透传的云产品及其 Actions
// ReadActions: 查询类接口，无需审计
// WriteActions: 变更类接口，需要审计
// 只需在此处添加新的产品和接口即可，无需修改其他代码
var cloudProxyRegistry = map[string]*cloudProxyService{
	"cvm": {
		Service: "cvm",
		Version: "2017-03-12",
		ReadActions: map[string]bool{
			"DescribeInstances": true,
		},
		WriteActions: map[string]bool{
			// 安全组关联
			"AssociateSecurityGroups":    true,
			"DisassociateSecurityGroups": true,
		},
	},
	"vpc": {
		Service: "vpc",
		Version: "2017-03-12",
		ReadActions: map[string]bool{
			"DescribeSecurityGroups":        true,
			"DescribeSecurityGroupPolicies": true,
		},
		WriteActions: map[string]bool{
			"DeleteSecurityGroupPolicies":     true,
			"DeleteSecurityGroup":             true,
			"CreateSecurityGroupWithPolicies": true,
			"CreateSecurityGroupPolicies":     true,
		},
	},
	"cls": {
		Service: "cls",
		Version: "2020-10-16",
		ReadActions: map[string]bool{
			"DescribeLogsets":        true,
			"DescribeTopics":         true,
			"SearchLog":              true,
			"QueryMetric":            true,
			"QueryRangeMetric":       true,
			"GetClsService":          true,
			"DescribeRainbowConfigs": true, // 新增白名单接口
			"DescribeTemplates":      true, // 2026-03-26 新增
		},
		WriteActions: map[string]bool{
			"OpenClsService":  true,
			"OpenClawService": true,
			"DeleteTopic":     true,
		},
	},
	"billing": {
		Service: "billing",
		Version: "2018-07-09",
		ReadActions: map[string]bool{
			"DescribeMeasureResources": true, // 查询用户购买的套餐包详情列表 (https://iwiki.woa.com/p/1151041572)
		},
		WriteActions: map[string]bool{
			"CreateOrdersAndPay": true, // 下单并支付V2 (https://tcloud4api.woa.com/document/product/555/45472?!preview&!document=1)
		},
	},
	"csip": {
		Service: "csip",
		Version: "2022-11-21",
		ReadActions: map[string]bool{
			"DescribeAIAgentAssetList":        true,
			"DescribeABTestConfig":            true,
			"DescribeOrganizationInfo":        true,
			"DescribePayInfo":                 true,
			"DescribeUserAccountInfo":         true,
			"DescribeUserOperationPermission": true,
			"GetLocalStorageItem":             true,
			"DescribeAgentlessVulAssetDetail": true,
			"DescribeExposeRules":             true,
			"DescribeExposeAssetCategory":     true,
			"DescribeExposePath":              true,
			"DescribeCVMAssets":               true,
			"DescribeAssetProcessList":        true,
			"DescribeVulRiskList":             true,
			"DescribeHighBaseLineRiskList":    true,
			"DescribeExposures":               true,
			// 2026-03-24 新增
			"DescribeAgentlessVulRiskList":     true,
			"DescribeAgentlessVulAssetList":    true,
			"DescribeKeySandboxCredentialList": true,
			"DescribeKeySandboxCredential":     true,
			"DescribeExportJobDownloadURL":     true,
			// 2026-03-30 新增
			"DescribeAIAgentSkillList": true,
			// 2026-05 SkillScan 试用管理
			"DescribeSkillScanResult":  true,
			"DescribeTrialStatus":      true,
			"DescribeSkillScanPayInfo": true,
			// 2026-06-10 新增
			"DescribeCSCPayInfo":                    true,
			"DescribeAIAgentAssetInContainerList":   true,
			"DescribeSandboxACLAlertList":           true,
			"DescribeSandboxDLPAlertList":           true,
			"DescribeSandboxACLRuleList":            true,
			"DescribeSandboxLLMAuditAlertList":      true,
			"DescribeSandboxLLMAuditRuleList":       true,
			"DescribeSandboxDLPRuleList":            true,
			"DescribeAIAgentCredentialList":         true,
			"DescribeSandboxLLMAuditSystemRuleList": true,
			"DescribeSandboxFileRuleList":           true,
			"DescribeSandboxDLPSystemRuleList":      true,
			"DescribeLogCategory":                   true,
			"DescribeCLSLogListV3":                  true,
		},
		WriteActions: map[string]bool{
			"SetLocalStorageItem": true,
			// 2026-03-24 新增
			"CreateScanTask":             true,
			"ApplyTrial":                 true,
			"CreateKeySandboxCredential": true,
			"ModifyKeySandboxCredential": true,
			"InstallKeySandboxSkill":     true,
			"UninstallKeySandboxSkill":   true,
			"DeleteKeySandboxCredential": true,
			"CreateExposuresExportJob":   true,
			// 2026-05 SkillScan 试用管理
			"CreateSkillScan":   true,
			"ModifyTrialStatus": true,
			// 2026-06-10 新增
			"ModifySandboxAlertStatus":        true,
			"DeleteSandboxACLRule":            true,
			"ModifySandboxACLRuleStatus":      true,
			"CreateSandboxACLRule":            true,
			"ModifySandboxACLRule":            true,
			"DeleteSandboxLLMAuditRule":       true,
			"ModifySandboxLLMAuditRuleStatus": true,
			"CreateSandboxLLMAuditRule":       true,
			"ModifySandboxLLMAuditRule":       true,
			"DeleteSandboxFileRule":           true,
			"ModifySandboxFileRuleStatus":     true,
			"CreateSandboxFileRule":           true,
			"ModifySandboxFileRule":           true,
			"DeleteSandboxDLPRule":            true,
			"ModifySandboxDLPRuleStatus":      true,
			"CreateSandboxDLPRule":            true,
			"ModifySandboxDLPRule":            true,
			"InstallSandboxPlugin":            true,
		},
	},
	"cwp": {
		Service: "cwp",
		Version: "2018-02-28",
		ReadActions: map[string]bool{
			"DescribeLicenseWhiteConfig":  true,
			"DescribeOrderList":           true,
			"DescribeLogStorageConfig":    true,
			"DescribeLicenseBindSchedule": true,
			"DescribeBashEventsNew":       true,
			"DescribeRiskDnsEventList":    true,
			"DescribeVersionStatistics":   true,
			"DescribeBashEventsInfoNew":   true,
			"DescribeMachines":            true,
			"DescribeRiskDnsEventInfo":    true,
			"DescribeBashPolicies":        true,
			"DescribeMalWareList":         true,
			"DescribeRiskDnsPolicyList":   true,
			"DescribeSkillInfo":           true,
			"DescribeImportMachineInfo":   true,
			"DescribeRiskBatchStatus":     true,
			"DescribeTags":                true,
			"DescribeMachineRegionList":   true,
			"DescribeLicenseGeneral":      true,
			"GetLocalStorageItem":         true,
			"DescribeMachineGeneral":      true,
			"DescribeLogHistogram":        true,
			"DescribeLogStorageStatistic": true,
			"SearchLog":                   true,
			"DescribeMachineInfo":         true,
			"DescribeMalwareInfo":         true,
			// 2026-03-24 新增
			"DescribeVulList":                 true,
			"DescribeVulInfoCvss":             true,
			"DescribeVulIgnoreRule":           true,
			"DescribeVulEffectHostList":       true,
			"DescribeLicenseList":             true,
			"DescribeGrayPolicy":              true,
			"DescribeUsersConfig":             true,
			"DescribeMachinesSimple":          true,
			"DescribeLicenseBindList":         true,
			"DescribeHostInfo":                true,
			"DescribeBaselineItemDetectList":  true,
			"DescribeBaselineRuleDetectList":  true,
			"DescribeBaselineHostDetectList":  true,
			"DescribeBaselineItemList":        true,
			"DescribeIgnoreHostAndItemConfig": true,
			"DescribeBaselineDownloadList":    true,
			"DescribeBaselineRuleIgnoreList":  true,
			// 2026-05-19新增
			"DescribeAIAgentAutoOpenConfig": true,
			// 2026-06-10 新增
			"DescribeMachineRegions": true,
		},
		WriteActions: map[string]bool{
			"CreateWhiteListOrder":               true,
			"ModifyLicenseBinds":                 true,
			"ModifyLogStorageConfig":             true,
			"ScanAsset":                          true,
			"SyncAssetScan":                      true,
			"ModifyRiskEventsStatus":             true,
			"SetLocalStorageItem":                true,
			"RemoveLocalStorageItem":             true,
			"ModifyBashPolicyStatus":             true,
			"DeleteBashPolicies":                 true,
			"ModifyBashPolicy":                   true,
			"CheckBashPolicyParams":              true,
			"ModifyRiskDnsPolicy":                true,
			"ModifyRiskDnsPolicyStatus":          true,
			"DeleteRiskDnsPolicy":                true,
			"ModifyReverseShellRulesAggregation": true,
			// 2026-03-24 新增
			"ScanVulAgain":                 true,
			"ExportVulList":                true,
			"ExportTasks":                  true,
			"ExportVulEffectHostList":      true,
			"AddVulIgnoreRule":             true,
			"CancelVulIgnoreRule":          true,
			"ExportRiskDnsPolicyList":      true,
			"ExportBashPolicies":           true,
			"StartBaselineDetect":          true,
			"SyncBaselineDetectSummary":    true,
			"ModifyBaselineRuleIgnore":     true,
			"ExportBaselineItemList":       true,
			"ExportBaselineItemDetectList": true,
			// 2026-05-19新增
			"ModifyAIAgentAutoOpenConfig": true,
			// 2026-07-08新增
			"ModifyOrder": true,
		},
	},
	"vdb": {
		Service:     "vdb",
		Version:     "2023-06-16",
		ReadActions: map[string]bool{},
		WriteActions: map[string]bool{
			"CreateInstance":   true,
			"ScaleOutInstance": true,
			"ScaleUpInstance":  true,
		},
	},
	"smh": {
		Service: "smh",
		Version: "2021-07-12",
		ReadActions: map[string]bool{
			"DescribeLibrarySecret": true,
			"DescribeLibraries":     true,
		},
		WriteActions: map[string]bool{
			"CreateLibrary": true,
			"ModifyLibrary": true,
			"DeleteLibrary": true,
		},
	},
}

// HandleCloudProxyQuery 处理只读类腾讯云 API 透传（Describe/Inquiry 等查询接口）。
//
// 路由: POST /admin/cloud/query/{service}
//
// 请求格式与腾讯云 API 完全一致，通过 HTTP Header 传递 Action：
//
//	Header:
//	  X-TC-Action: DescribeInstances   (必须)
//	  X-TC-Version: 2017-03-12          (可选，默认使用注册的版本)
//	Body:
//	  腾讯云 API 的 JSON 请求体（与官方文档完全一致）
//
// 响应格式与腾讯云 API 完全一致，原样透传。
func HandleCloudProxyQuery(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	// 从 URL 路径提取 service 名称: /admin/cloud/query/{service}
	path := strings.TrimPrefix(r.URL.Path, "/admin/cloud/query/")
	service := strings.TrimRight(path, "/")
	if service == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgCloudProxyMissingServiceQuery))
		return
	}

	svcConfig, ok := cloudProxyRegistry[service]
	if !ok {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgCloudProxyUnsupportedService, service, availableServices()))
		return
	}

	action := getAction(r)
	if action == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgCloudProxyMissingAction))
		return
	}

	// 检查 Action 是否在只读白名单中
	if !svcConfig.ReadActions[action] {
		allowed := actionList(svcConfig.ReadActions)
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgCloudProxyActionNotInReadWhitelist,
			action, service, strings.Join(allowed, ", ")))
		return
	}

	doCloudProxy(w, r, svcConfig, service, action, "read")
}

// HandleCloudProxyMutate 处理变更类腾讯云 API 透传（Create/Delete/Start/Stop/Run 等变更接口）。
// 该 handler 应配合 WithCloudProxyAudit 使用，以记录操作审计日志。
//
// 路由: POST /admin/cloud/mutate/{service}
//
// 请求格式与查询接口完全一致。
func HandleCloudProxyMutate(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	// 从 URL 路径提取 service 名称: /admin/cloud/mutate/{service}
	path := strings.TrimPrefix(r.URL.Path, "/admin/cloud/mutate/")
	service := strings.TrimRight(path, "/")
	if service == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgCloudProxyMissingServiceMutate))
		return
	}

	svcConfig, ok := cloudProxyRegistry[service]
	if !ok {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgCloudProxyUnsupportedService, service, availableServices()))
		return
	}

	action := getAction(r)
	if action == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgCloudProxyMissingAction))
		return
	}

	// 检查 Action 是否在写操作白名单中
	if !svcConfig.WriteActions[action] {
		allowed := actionList(svcConfig.WriteActions)
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgCloudProxyActionNotInWriteWhitelist,
			action, service, strings.Join(allowed, ", ")))
		return
	}

	doCloudProxy(w, r, svcConfig, service, action, "write")
}

// getAction 从请求中提取 Action（Header 优先，Query 参数兜底）
func getAction(r *http.Request) string {
	action := r.Header.Get("X-TC-Action")
	if action == "" {
		action = r.URL.Query().Get("Action")
	}
	return action
}

// availableServices 返回所有已注册 service 名称的逗号分隔字符串
func availableServices() string {
	available := make([]string, 0, len(cloudProxyRegistry))
	for k := range cloudProxyRegistry {
		available = append(available, k)
	}
	return strings.Join(available, ", ")
}

// actionList 将 Action 白名单 map 转为字符串切片
func actionList(actions map[string]bool) []string {
	list := make([]string, 0, len(actions))
	for a := range actions {
		list = append(list, a)
	}
	return list
}

// doCloudProxy 是读写接口共用的核心转发逻辑
func doCloudProxy(w http.ResponseWriter, r *http.Request, svcConfig *cloudProxyService, service, action, opType string) {
	jsonAPI(w)

	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgReadRequestBody))
		return
	}
	defer r.Body.Close()

	// 如果 body 为空，使用空 JSON 对象
	if len(body) == 0 {
		body = []byte("{}")
	}

	// 验证请求体是合法 JSON
	if !json.Valid(body) {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidJSON))
		return
	}

	region := CVMRegion

	// 获取 Version：Header > 注册默认
	version := r.Header.Get("X-TC-Version")
	if version == "" {
		version = svcConfig.Version
	}

	slog.Info("[CloudProxy] 转发请求",
		"type", opType,
		"service", service,
		"action", action,
		"region", region,
		"version", version,
		"body_size", len(body),
	)

	// 获取凭证（支持 STS 角色扮演）
	credential, err := getCredential(r.Context())
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 构建 CommonClient
	cpf := profile.NewClientProfile()
	if svcConfig.Endpoint != "" {
		cpf.HttpProfile.Endpoint = svcConfig.Endpoint
	} else {
		cpf.HttpProfile.Endpoint = service + ".tencentcloudapi.com"
	}
	cpf.HttpProfile.ReqMethod = "POST"

	client := common.NewCommonClient(credential, region, cpf)

	// 构造请求
	request := tchttp.NewCommonRequest(service, version, action)
	if err := request.SetActionParameters(string(body)); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgSetRequestParamsFailed))
		return
	}

	// 发起调用
	response := tchttp.NewCommonResponse()
	if err := client.Send(request, response); err != nil {
		slog.Error("[CloudProxy] 调用失败",
			"type", opType,
			"service", service,
			"action", action,
			"error", err,
		)
		// 尝试将 SDK 错误转成腾讯云标准错误格式返回
		w.WriteHeader(http.StatusOK) // 腾讯云 API 错误也返回 200
		json.NewEncoder(w).Encode(map[string]interface{}{
			"Response": map[string]interface{}{
				"Error": map[string]string{
					"Code":    "InternalError",
					"Message": err.Error(),
				},
			},
		})
		return
	}

	respBody := response.GetBody()

	slog.Info("[CloudProxy] 调用成功",
		"type", opType,
		"service", service,
		"action", action,
		"resp_size", len(respBody),
	)

	// 原样透传响应
	w.Write(respBody)
}

// HandleCloudProxyActions 返回所有已注册的 service 及其可用 Actions（区分读写），方便前端/调试使用。
//
// 路由: GET /admin/cloud
func HandleCloudProxyActions(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	result := make(map[string]interface{})
	for svc, cfg := range cloudProxyRegistry {
		readActions := actionList(cfg.ReadActions)
		writeActions := actionList(cfg.WriteActions)
		result[svc] = map[string]interface{}{
			"service":       cfg.Service,
			"version":       cfg.Version,
			"read_actions":  readActions,
			"write_actions": writeActions,
		}
	}
	jsonOK(w, result)
}
