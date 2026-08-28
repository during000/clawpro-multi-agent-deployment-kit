package controller

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tchttp "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/http"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"

	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
)

// ==================== CheckAccountBalance 接口常量 ====================

const (
	// billing API 请求配置
	billingEndpoint = "billing.tencentcloudapi.com"
	billingScheme   = "https"
	billingService  = "billing"
	billingVersion  = "2018-07-09"
	billingAction   = "CheckAccountBalance"
	billingRegion   = "ap-guangzhou" // CheckAccountBalance 仅支持广州
)

// ==================== 常量 ====================

const (
	quotaCacheTTL = 5 * time.Minute

	// 告警级别
	alertLevelInfo     = "info"
	alertLevelCritical = "critical"

	// 告警 ID
	alertIDVPC            = "vpc"
	alertIDSubnet         = "subnet"
	alertIDSecurityGroup  = "security_group"
	alertIDAccountArrears = "account_arrears"

	// 安全组关联实例数默认上限（接口查询失败时的 fallback）
	securityGroupCVMLimitDefault uint64 = 2000

	// VPC 工单链接（产品分类=私有网络，问题分类=配额申请）
	vpcWorkorderHref = "https://console.cloud.tencent.com/workorder/category?level1_id=6&source=14&level2_id=168&data_title=%E7%A7%81%E6%9C%89%E7%BD%91%E7%BB%9C&level3_id=184&radio_title=%E9%85%8D%E9%A2%9D%E7%94%B3%E8%AF%B7&queue=96&scene_code=34515"

	// 安全组工单链接（产品分类=私有网络，问题分类=安全组）
	sgWorkorderHref = "https://console.cloud.tencent.com/workorder/category?level1_id=6&source=14&level2_id=168&data_title=%E7%A7%81%E6%9C%89%E7%BD%91%E7%BB%9C&level3_id=183&radio_title=%E5%AE%89%E5%85%A8%E7%BB%84&queue=96&scene_code=17127"

	// 充值链接
	rechargeHref = "https://console.cloud.tencent.com/expense/recharge"
)

// ==================== 响应结构体 ====================

// QuotaAlertAction 配额告警跳转操作
type QuotaAlertAction struct {
	Label    string `json:"label"`
	Href     string `json:"href"`
	External bool   `json:"external"`
}

// QuotaAlertItem 单条配额告警
type QuotaAlertItem struct {
	ID      string            `json:"id"`
	Level   string            `json:"level"` // "info" | "critical"
	Message string            `json:"message"`
	Action  *QuotaAlertAction `json:"action"`
	Detail  interface{}       `json:"detail"`
}

// ==================== 详情结构体 ====================

// vpcQuotaDetail VPC 配额告警详情
type vpcQuotaDetail struct {
	Region       string  `json:"region"`
	Total        uint64  `json:"total"`
	Used         uint64  `json:"used"`
	Remaining    uint64  `json:"remaining"`
	UsagePercent float64 `json:"usage_percent"`
}

// subnetIPDetail 子网可用 IP 告警详情
type subnetIPDetail struct {
	Subnets []subnetIPItem `json:"subnets"`
}

type subnetIPItem struct {
	SubnetID         string `json:"subnet_id"`
	Zone             string `json:"zone"`
	AvailableIPCount uint64 `json:"available_ip_count"`
	TotalIPCount     uint64 `json:"total_ip_count"`
}

// sgQuotaDetail 安全组实例数告警详情
type sgQuotaDetail struct {
	SecurityGroupID string `json:"security_group_id"`
	CVMCount        uint64 `json:"cvm_count"`
	Limit           uint64 `json:"limit"`
}

// accountArrearsDetail 账号欠费告警详情
type accountArrearsDetail struct {
	UIN          string `json:"uin"`
	IsOwed       bool   `json:"is_owed"`        // 是否欠费
	IsOverCredit bool   `json:"is_over_credit"` // 是否超额
}

// checkAccountBalanceResult CheckAccountBalance 接口返回结构
type checkAccountBalanceResult struct {
	IsOwed       bool
	IsOverCredit bool
	Uin          int64
}

// ==================== 配额缓存 ====================

// quotaAlertCache 配额告警内存缓存，避免频繁调用云 API
type quotaAlertCache struct {
	mu      sync.RWMutex
	alerts  []QuotaAlertItem
	expires time.Time
}

// get 获取缓存，过期返回 nil
func (c *quotaAlertCache) get() []QuotaAlertItem {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.expires.IsZero() || time.Now().After(c.expires) {
		return nil
	}
	result := make([]QuotaAlertItem, len(c.alerts))
	copy(result, c.alerts)
	return result
}

// set 写入缓存并设置过期时间
func (c *quotaAlertCache) set(alerts []QuotaAlertItem) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.alerts = make([]QuotaAlertItem, len(alerts))
	copy(c.alerts, alerts)
	c.expires = time.Now().Add(quotaCacheTTL)
}

// 全局缓存实例
var globalQuotaCache = &quotaAlertCache{}

// ==================== 纯函数（可测试） ====================

// calcVPCQuotaLevel 根据 VPC 用量判断告警级别
func calcVPCQuotaLevel(total, used uint64) string {
	if total == 0 || used >= total {
		return alertLevelCritical
	}
	return alertLevelInfo
}

// calcSubnetIPLevel 根据子网可用 IP 判断告警级别
// 所有子网可用 IP 均为 0 时告警
func calcSubnetIPLevel(subnets []subnetIPItem) string {
	if len(subnets) == 0 {
		return alertLevelInfo
	}
	for _, s := range subnets {
		if s.AvailableIPCount > 0 {
			return alertLevelInfo
		}
	}
	return alertLevelCritical
}

// calcSGQuotaLevel 根据安全组关联 CVM 数判断告警级别
func calcSGQuotaLevel(cvmCount, limit uint64) string {
	if cvmCount >= limit {
		return alertLevelCritical
	}
	return alertLevelInfo
}

// calcAccountArrearsLevel 根据是否超额判断告警级别
// 仅关注 IsOverCredit 字段，IsOwed 不再作为告警触发条件
func calcAccountArrearsLevel(isOverCredit bool) string {
	if isOverCredit {
		return alertLevelCritical
	}
	return alertLevelInfo
}

// parseCheckAccountBalanceResp 解析 CheckAccountBalance API 响应体
// 响应示例：{"Response":{"IsOwed":false,"IsOverCredit":false,"Uin":3205597606,"RequestId":"..."}}
// 出错时返回 API Error 信息
func parseCheckAccountBalanceResp(body []byte) (checkAccountBalanceResult, error) {
	var parsed struct {
		Response struct {
			IsOwed       bool   `json:"IsOwed"`
			IsOverCredit bool   `json:"IsOverCredit"`
			Uin          int64  `json:"Uin"`
			RequestId    string `json:"RequestId"`
			Error        *struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error"`
		} `json:"Response"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return checkAccountBalanceResult{}, hcommon.I18nRichError(err, i18n.MsgUnmarshalBillingRespFailed)
	}
	if parsed.Response.Error != nil {
		return checkAccountBalanceResult{}, hcommon.I18nError(i18n.MsgBillingAPIError,
			parsed.Response.Error.Code, parsed.Response.Error.Message)
	}
	return checkAccountBalanceResult{
		IsOwed:       parsed.Response.IsOwed,
		IsOverCredit: parsed.Response.IsOverCredit,
		Uin:          parsed.Response.Uin,
	}, nil
}

// ==================== 配额检查函数 ====================

// checkVPCQuota 检查 VPC 配额
// 仅自动分配模式（config.VpcId 为空）时检查
func checkVPCQuota(ctx context.Context, config model.SiteConfig, vpcClient *vpc.Client) *QuotaAlertItem {
	// 管理员指定了 VPC，不需要检查
	if config.VpcId != "" {
		return nil
	}

	// 查询 VPC 配额上限
	limReq := vpc.NewDescribeVpcLimitsRequest()
	limReq.LimitTypes = common.StringPtrs([]string{"appid-max-vpcs"})
	limResp, err := vpcClient.DescribeVpcLimits(limReq)
	if err != nil {
		slog.Error("[配额告警] 查询 VPC 配额上限失败", "error", err)
		return nil
	}
	var total uint64
	for _, lim := range limResp.Response.VpcLimitSet {
		if lim.LimitType != nil && *lim.LimitType == "appid-max-vpcs" && lim.LimitValue != nil {
			total = *lim.LimitValue
			break
		}
	}

	// 查询已使用 VPC 数量
	vpcReq := vpc.NewDescribeVpcsRequest()
	vpcReq.Limit = common.StringPtr("1")
	vpcResp, err := vpcClient.DescribeVpcs(vpcReq)
	if err != nil {
		slog.Error("[配额告警] 查询 VPC 数量失败", "error", err)
		return nil
	}
	var used uint64
	if vpcResp.Response.TotalCount != nil {
		used = *vpcResp.Response.TotalCount
	}

	var remaining uint64
	if total > used {
		remaining = total - used
	}
	var usagePercent float64
	if total > 0 {
		usagePercent = float64(used) / float64(total) * 100
	}

	level := calcVPCQuotaLevel(total, used)
	regionName := Regions[CVMRegion].ShortName

	slog.Info("[配额告警] VPC 配额查询完成",
		"region", CVMRegion, "total", total, "used", used, "level", level)

	var message string
	if level == alertLevelCritical {
		message = i18n.T(ctx, i18n.MsgQuotaAlertVPCExceeded, regionName)
	}

	return &QuotaAlertItem{
		ID:      alertIDVPC,
		Level:   level,
		Message: message,
		Action: &QuotaAlertAction{
			Label:    i18n.T(ctx, i18n.MsgQuotaAlertActionGoToHandle),
			Href:     vpcWorkorderHref,
			External: true,
		},
		Detail: vpcQuotaDetail{
			Region:       CVMRegion,
			Total:        total,
			Used:         used,
			Remaining:    remaining,
			UsagePercent: usagePercent,
		},
	}
}

// checkSubnetIPQuota 检查子网可用 IP
func checkSubnetIPQuota(ctx context.Context, config model.SiteConfig, vpcClient *vpc.Client) *QuotaAlertItem {
	// 确定当前使用的子网列表
	var subnetIDs []string
	if config.VpcId != "" {
		// 手动指定模式
		for _, sids := range config.GetSubnetMap() {
			subnetIDs = append(subnetIDs, sids...)
		}
	} else {
		// 自动分配模式
		for _, sids := range config.GetDefaultSubnetMap() {
			subnetIDs = append(subnetIDs, sids...)
		}
	}

	// 去重
	seen := make(map[string]bool)
	var uniqueIDs []string
	for _, id := range subnetIDs {
		if id != "" && !seen[id] {
			seen[id] = true
			uniqueIDs = append(uniqueIDs, id)
		}
	}

	// 无子网配置，不告警
	if len(uniqueIDs) == 0 {
		return nil
	}

	req := vpc.NewDescribeSubnetsRequest()
	req.SubnetIds = common.StringPtrs(uniqueIDs)
	resp, err := vpcClient.DescribeSubnets(req)
	if err != nil {
		slog.Error("[配额告警] 查询子网失败", "error", err)
		return nil
	}

	var items []subnetIPItem
	for _, s := range resp.Response.SubnetSet {
		if s.SubnetId == nil {
			continue
		}
		item := subnetIPItem{SubnetID: *s.SubnetId}
		if s.Zone != nil {
			item.Zone = *s.Zone
		}
		if s.AvailableIpAddressCount != nil {
			item.AvailableIPCount = *s.AvailableIpAddressCount
		}
		if s.TotalIpAddressCount != nil {
			item.TotalIPCount = *s.TotalIpAddressCount
		}
		items = append(items, item)
	}

	level := calcSubnetIPLevel(items)

	var totalAvailableIP uint64
	for _, item := range items {
		totalAvailableIP += item.AvailableIPCount
	}
	slog.Info("[配额告警] 子网可用 IP 查询完成",
		"checked_subnets", len(items), "total_available_ip", totalAvailableIP, "level", level)

	var message string
	if level == alertLevelCritical {
		message = i18n.T(ctx, i18n.MsgQuotaAlertSubnetIPExhausted)
	}

	return &QuotaAlertItem{
		ID:      alertIDSubnet,
		Level:   level,
		Message: message,
		Action: &QuotaAlertAction{
			Label:    i18n.T(ctx, i18n.MsgQuotaAlertActionGoToHandle),
			Href:     "/admin/security-group",
			External: false,
		},
		Detail: subnetIPDetail{Subnets: items},
	}
}

// checkSecurityGroupQuota 检查所有 ACTIVE SG 关联实例数的最高水位
//
// 新模型：迁移到 RuleSet + ManagedSGPool 后，tenant 下有多个 ACTIVE SG。老实现只看
// config.SecurityGroupId（已是 FROZEN），统计偏低导致管理员误判还有余量。现遍历所有
// ACTIVE SG 取 CVMCount 最高者产生告警，detail 带该 SG 的 ID 便于定位。
func checkSecurityGroupQuota(ctx context.Context, config model.SiteConfig, vpcClient *vpc.Client) *QuotaAlertItem {
	_ = config // 保留签名；配置不再决定哪个 SG 参与统计
	sgIDs, err := listAllActiveSGIDs(ctx)
	if err != nil {
		slog.Warn("[配额告警] 枚举 ACTIVE SG 失败", "error", err)
		return nil
	}
	if len(sgIDs) == 0 {
		return nil
	}

	// 查询安全组关联实例数上限（用户可能自行提升过）
	limit := securityGroupCVMLimitDefault
	limReq := vpc.NewDescribeSecurityGroupLimitsRequest()
	limResp, err := vpcClient.DescribeSecurityGroupLimits(limReq)
	if err != nil {
		slog.Warn("[配额告警] 查询安全组配额上限失败，使用默认值", "error", err, "default_limit", limit)
	} else if limResp.Response != nil && limResp.Response.SecurityGroupLimitSet != nil &&
		limResp.Response.SecurityGroupLimitSet.SecurityGroupInstanceLimit != nil {
		limit = *limResp.Response.SecurityGroupLimitSet.SecurityGroupInstanceLimit
	}

	// 批量查询所有 ACTIVE SG 的关联统计
	req := vpc.NewDescribeSecurityGroupAssociationStatisticsRequest()
	req.SecurityGroupIds = common.StringPtrs(sgIDs)
	resp, err := vpcClient.DescribeSecurityGroupAssociationStatistics(req)
	if err != nil {
		slog.Error("[配额告警] 查询安全组关联统计失败", "error", err, "sg_count", len(sgIDs))
		return nil
	}

	// 取 CVM 数最高的 SG 作为告警代表
	var peakSGID string
	var peakCVMCount uint64
	if resp.Response != nil {
		for _, stat := range resp.Response.SecurityGroupAssociationStatisticsSet {
			if stat == nil || stat.CVM == nil || stat.SecurityGroupId == nil {
				continue
			}
			if *stat.CVM > peakCVMCount {
				peakCVMCount = *stat.CVM
				peakSGID = *stat.SecurityGroupId
			}
		}
	}

	level := calcSGQuotaLevel(peakCVMCount, limit)

	slog.Info("[配额告警] 所有 ACTIVE SG 关联实例数查询完成",
		"peak_security_group_id", peakSGID,
		"peak_cvm_count", peakCVMCount, "limit", limit, "level", level,
		"sg_count", len(sgIDs))

	var message string
	if level == alertLevelCritical {
		message = i18n.T(ctx, i18n.MsgQuotaAlertSGInstanceExceeded,
			peakSGID)
	}

	return &QuotaAlertItem{
		ID:      alertIDSecurityGroup,
		Level:   level,
		Message: message,
		Action: &QuotaAlertAction{
			Label:    i18n.T(ctx, i18n.MsgQuotaAlertActionGoToHandle),
			Href:     sgWorkorderHref,
			External: true,
		},
		Detail: sgQuotaDetail{
			SecurityGroupID: peakSGID,
			CVMCount:        peakCVMCount,
			Limit:           limit,
		},
	}
}

// accountBalanceFetcher 可替换为 mock 实现以便单测
// 返回原始响应体，由 parseCheckAccountBalanceResp 解析
var accountBalanceFetcher = callCheckAccountBalance

// callCheckAccountBalance 通过 CommonClient 调用 billing.CheckAccountBalance
func callCheckAccountBalance(ctx context.Context) ([]byte, error) {
	credential, err := getCredential(ctx)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgAPIGatewayGetCredFailed)
	}

	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = billingEndpoint
	cpf.HttpProfile.Scheme = billingScheme
	cpf.HttpProfile.ReqMethod = "POST"
	client := common.NewCommonClient(credential, billingRegion, cpf)

	request := tchttp.NewCommonRequest(billingService, billingVersion, billingAction)
	if err := request.SetActionParameters("{}"); err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgSetActionParamsFailed)
	}
	response := tchttp.NewCommonResponse()
	if err := client.Send(request, response); err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgSendCheckBalanceFailed)
	}
	return response.GetBody(), nil
}

// checkAccountArrears 检查腾讯云账号是否欠费/超额
// 使用 CommonClient + billing.CheckAccountBalance 接口
func checkAccountArrears(ctx context.Context) *QuotaAlertItem {
	body, err := accountBalanceFetcher(ctx)
	if err != nil {
		slog.Error("[配额告警] 调用 CheckAccountBalance 失败", "error", err)
		return nil
	}

	result, err := parseCheckAccountBalanceResp(body)
	if err != nil {
		slog.Error("[配额告警] 解析 CheckAccountBalance 响应失败", "error", err, "body", string(body))
		return nil
	}

	level := calcAccountArrearsLevel(result.IsOverCredit)

	uin := hcommon.CVMUinFromCtx(ctx)
	slog.Info("[配额告警] 账户欠费状态查询完成",
		"uin", uin, "is_owed", result.IsOwed, "is_over_credit", result.IsOverCredit, "level", level)

	var message string
	if level == alertLevelCritical {
		message = i18n.T(ctx, i18n.MsgQuotaAlertAccountOverdue,
			uin)
	}

	return &QuotaAlertItem{
		ID:      alertIDAccountArrears,
		Level:   level,
		Message: message,
		Action: &QuotaAlertAction{
			Label:    i18n.T(ctx, i18n.MsgQuotaAlertActionGoToHandle),
			Href:     rechargeHref,
			External: true,
		},
		Detail: accountArrearsDetail{
			UIN:          uin,
			IsOwed:       result.IsOwed,
			IsOverCredit: result.IsOverCredit,
		},
	}
}

// ==================== 编排函数 ====================

// buildQuotaAlerts 构建配额告警列表，带缓存
func buildQuotaAlerts(ctx context.Context) []QuotaAlertItem {
	// 命中缓存直接返回
	if cached := globalQuotaCache.get(); cached != nil {
		return cached
	}

	// 未配置云 API 密钥则跳过
	config := model.GetSiteConfig(ctx)
	if config.CVMSecretId == "" {
		return []QuotaAlertItem{}
	}

	// 创建共享 VPC 客户端（VPC/子网/安全组检查共用）
	vpcClient, err := newVpcClient(ctx)
	if err != nil {
		slog.Error("[配额告警] 创建 VPC 客户端失败", "error", err)
		return []QuotaAlertItem{}
	}

	// 并行执行 4 个配额检查
	type result struct {
		index int
		item  *QuotaAlertItem
	}
	ch := make(chan result, 4)
	var wg sync.WaitGroup

	checks := []func(){
		func() { ch <- result{0, checkVPCQuota(ctx, config, vpcClient)} },
		func() { ch <- result{1, checkSubnetIPQuota(ctx, config, vpcClient)} },
		func() { ch <- result{2, checkSecurityGroupQuota(ctx, config, vpcClient)} },
		func() { ch <- result{3, checkAccountArrears(ctx)} },
	}
	wg.Add(len(checks))
	for _, fn := range checks {
		go func(f func()) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					slog.Error("[配额告警] check 函数 panic", "recover", r)
				}
			}()
			f()
		}(fn)
	}
	wg.Wait()
	close(ch)

	// 按固定顺序收集结果
	results := make([]*QuotaAlertItem, 4)
	for r := range ch {
		results[r.index] = r.item
	}

	var alerts []QuotaAlertItem
	for _, item := range results {
		if item != nil {
			alerts = append(alerts, *item)
		}
	}

	if alerts == nil {
		alerts = []QuotaAlertItem{}
	}

	globalQuotaCache.set(alerts)
	return alerts
}
