package controller

import (
	"context"
	hcommon "hatchery/common"
	"hatchery/i18n"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
)

// EgressBlockedMessage 是命中"出站被安全组拒绝"时替换后的面向用户文案。
// 保留此常量以兼容外部引用（openclaw_channel.go 等），值与 i18n.MsgEgressBlocked 一致。
var EgressBlockedMessage = i18n.MsgEgressBlocked

// diagnoseTimeout 限定一次诊断总耗时，避免同步阻塞 HTTP 响应过久。
const diagnoseTimeout = 5 * time.Second

// cvmInstanceDescriber 抽象 DescribeInstances 供单测 mock。
type cvmInstanceDescriber interface {
	DescribeInstances(req *cvm.DescribeInstancesRequest) (*cvm.DescribeInstancesResponse, error)
}

// networkErrorKeywords / isNetworkError 已移除：
// RunScript 返回的 RichError.Error() 只含 Message（如"命令执行失败"），
// 真正的 curl/TAT 失败文案在 Detail 里，关键词匹配命中率为 0。
// 调用方应通过"场景白名单"（如指定 channel / handler）控制诊断触发，
// 而不是靠错误字符串判断。

// isEgressRulesEffectivelyEmpty 粗粒度判定一批 egress 规则是否"实际等同于拒绝所有出站"。
// 只要存在任意一条 ACCEPT 规则即视为非空；否则（空切片 / 全部 DROP|REJECT）视为被拒。
func isEgressRulesEffectivelyEmpty(egress []*vpc.SecurityGroupPolicy) bool {
	if len(egress) == 0 {
		return true
	}
	for _, p := range egress {
		if p == nil || p.Action == nil {
			continue
		}
		if strings.EqualFold(*p.Action, "ACCEPT") {
			return false
		}
	}
	return true
}

// DiagnoseInstanceEgress 查询实例绑定的所有安全组的出站规则，粗粒度判定出站是否被拒。
// 返回 blocked=true 表示"几乎可以肯定出站被安全组拒绝"。带 diagnoseTimeout 超时保护。
//
// 声明为变量而非函数，便于单测替换为 mock 实现（HandleAutoChannel 前置诊断
// 与 maybeWrapEgressBlocked 都通过此入口，注入点统一）。
var DiagnoseInstanceEgress = func(ctx context.Context, instanceID string) (blocked bool, err error) {
	if instanceID == "" {
		return false, hcommon.I18nError(i18n.MsgEgressInstanceIDEmpty)
	}

	cvmClient, cvmErr := NewCVMClient(ctx)
	if cvmErr != nil {
		return false, hcommon.I18nRichError(cvmErr, i18n.MsgCreateCVMClientFailed)
	}
	vpcClient, vpcErr := newVpcClient(ctx)
	if vpcErr != nil {
		return false, hcommon.I18nRichError(vpcErr, i18n.MsgCreateVPCClientFailed)
	}

	diagCtx, cancel := context.WithTimeout(ctx, diagnoseTimeout)
	defer cancel()
	return diagnoseInstanceEgressWith(diagCtx, cvmClient, vpcClient, instanceID)
}

// diagnoseInstanceEgressWith 是 DiagnoseInstanceEgress 的可注入版本，便于单测。
func diagnoseInstanceEgressWith(ctx context.Context, cvmCli cvmInstanceDescriber, vpcCli sgPolicyQuerier, instanceID string) (bool, error) {
	sgIds, err := fetchInstanceSecurityGroupIds(cvmCli, instanceID)
	if err != nil {
		return false, err
	}
	if len(sgIds) == 0 {
		// 未绑定任何安全组：走腾讯云默认策略（默认全通），不判定为被拒。
		return false, nil
	}

	allEgress, err := fetchEgressPoliciesParallel(ctx, vpcCli, sgIds)
	if err != nil {
		return false, err
	}
	return isEgressRulesEffectivelyEmpty(allEgress), nil
}

// fetchInstanceSecurityGroupIds 通过 DescribeInstances 获取实例绑定的安全组 ID 列表。
func fetchInstanceSecurityGroupIds(cvmCli cvmInstanceDescriber, instanceID string) ([]string, error) {
	req := cvm.NewDescribeInstancesRequest()
	req.InstanceIds = common.StringPtrs([]string{instanceID})
	resp, err := cvmCli.DescribeInstances(req)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgQueryInstanceFailed)
	}
	if resp == nil || resp.Response == nil || len(resp.Response.InstanceSet) == 0 {
		return nil, nil
	}
	inst := resp.Response.InstanceSet[0]
	sgIds := make([]string, 0, len(inst.SecurityGroupIds))
	for _, sg := range inst.SecurityGroupIds {
		if sg != nil {
			sgIds = append(sgIds, *sg)
		}
	}
	return sgIds, nil
}

// fetchEgressPoliciesParallel 并行查询多个安全组的 egress 规则，合并后返回。
// 任一 SG 查询失败即返回错误，上层放弃诊断（宁漏诊不误诊）。
// ctx 到期会通过 select 提前中止等待，避免阻塞调用方；goroutine 继续跑完不 leak（腾讯云 SDK 自带 HTTP 超时）。
func fetchEgressPoliciesParallel(ctx context.Context, vpcCli sgPolicyQuerier, sgIds []string) ([]*vpc.SecurityGroupPolicy, error) {
	type result struct {
		egress []*vpc.SecurityGroupPolicy
		err    error
	}

	var wg sync.WaitGroup
	results := make([]result, len(sgIds))
	for i, sgID := range sgIds {
		wg.Add(1)
		go func(idx int, id string) {
			defer wg.Done()
			req := vpc.NewDescribeSecurityGroupPoliciesRequest()
			req.SecurityGroupId = common.StringPtr(id)
			resp, err := vpcCli.DescribeSecurityGroupPolicies(req)
			if err != nil {
				results[idx] = result{err: hcommon.I18nRichError(err, i18n.MsgEgressQuerySGRulesFailed, id)}
				return
			}
			if resp == nil || resp.Response == nil || resp.Response.SecurityGroupPolicySet == nil {
				return
			}
			results[idx] = result{egress: resp.Response.SecurityGroupPolicySet.Egress}
		}(i, sgID)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		return nil, hcommon.I18nRichError(ctx.Err(), i18n.MsgEgressDiagnoseTimeout)
	}

	var all []*vpc.SecurityGroupPolicy
	for _, r := range results {
		if r.err != nil {
			return nil, r.err
		}
		all = append(all, r.egress...)
	}
	return all, nil
}

// maybeWrapEgressBlocked 调用安全组诊断，命中 egress 规则为空（实际全拒绝）则返回
// 一个面向用户的 RichError（保留原错误细节到 Detail），否则原样返回 origErr。
// 调用方负责通过"场景白名单"（如特定 channel / handler）控制何时调用本函数。
func maybeWrapEgressBlocked(ctx context.Context, instanceID string, origErr error) error {
	if origErr == nil {
		return nil
	}
	blocked, diagErr := DiagnoseInstanceEgress(ctx, instanceID)
	if diagErr != nil {
		slog.Warn("egress diagnostic skipped", "instance_id", instanceID, "diag_err", diagErr)
		return origErr
	}
	if !blocked {
		return origErr
	}
	return hcommon.I18nRichError(origErr, i18n.MsgEgressBlocked)
}
