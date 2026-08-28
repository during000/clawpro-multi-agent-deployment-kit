package controller

import (
	"context"
	"errors"
	hcommon "hatchery/common"
	"hatchery/i18n"
	"testing"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
)

// ========== isEgressRulesEffectivelyEmpty ==========

func policy(action, cidr, protocol, port string) *vpc.SecurityGroupPolicy {
	return &vpc.SecurityGroupPolicy{
		Action:    common.StringPtr(action),
		CidrBlock: common.StringPtr(cidr),
		Protocol:  common.StringPtr(protocol),
		Port:      common.StringPtr(port),
	}
}

func TestIsEgressRulesEffectivelyEmpty_Empty(t *testing.T) {
	if !isEgressRulesEffectivelyEmpty(nil) {
		t.Error("nil slice should be empty")
	}
	if !isEgressRulesEffectivelyEmpty([]*vpc.SecurityGroupPolicy{}) {
		t.Error("empty slice should be empty")
	}
}

func TestIsEgressRulesEffectivelyEmpty_AnyAcceptReturnsFalse(t *testing.T) {
	rules := []*vpc.SecurityGroupPolicy{
		policy("ACCEPT", "0.0.0.0/0", "TCP", "443"),
	}
	if isEgressRulesEffectivelyEmpty(rules) {
		t.Error("single ACCEPT rule should not be treated as empty")
	}

	rules2 := []*vpc.SecurityGroupPolicy{
		policy("DROP", "10.0.0.0/8", "ALL", "ALL"),
		policy("ACCEPT", "0.0.0.0/0", "TCP", "443"),
	}
	if isEgressRulesEffectivelyEmpty(rules2) {
		t.Error("mix with ACCEPT should not be empty")
	}
}

func TestIsEgressRulesEffectivelyEmpty_AllDropReturnsTrue(t *testing.T) {
	rules := []*vpc.SecurityGroupPolicy{
		policy("DROP", "0.0.0.0/0", "ALL", "ALL"),
	}
	if !isEgressRulesEffectivelyEmpty(rules) {
		t.Error("all-DROP rules should be treated as empty")
	}

	rules2 := []*vpc.SecurityGroupPolicy{
		policy("DROP", "10.0.0.0/8", "TCP", "22"),
		policy("REJECT", "0.0.0.0/0", "ALL", "ALL"),
	}
	if !isEgressRulesEffectivelyEmpty(rules2) {
		t.Error("DROP+REJECT should be treated as empty")
	}
}

func TestIsEgressRulesEffectivelyEmpty_NilEntries(t *testing.T) {
	rules := []*vpc.SecurityGroupPolicy{
		nil,
		{Action: nil}, // Action 为 nil 时跳过
		policy("ACCEPT", "0.0.0.0/0", "TCP", "443"),
	}
	if isEgressRulesEffectivelyEmpty(rules) {
		t.Error("should not treat ACCEPT as empty even with nil entries")
	}

	// 全部 nil Action 视为空（因为找不到任何 ACCEPT）
	allNil := []*vpc.SecurityGroupPolicy{nil, {Action: nil}}
	if !isEgressRulesEffectivelyEmpty(allNil) {
		t.Error("all-nil entries should be treated as empty")
	}
}

func TestIsEgressRulesEffectivelyEmpty_CaseInsensitiveAction(t *testing.T) {
	rules := []*vpc.SecurityGroupPolicy{
		policy("accept", "0.0.0.0/0", "TCP", "443"),
	}
	if isEgressRulesEffectivelyEmpty(rules) {
		t.Error("lowercase 'accept' should still count as ACCEPT")
	}
}

// ========== diagnoseInstanceEgressWith ==========

type fakeCvmClient struct {
	resp *cvm.DescribeInstancesResponse
	err  error
}

func (f *fakeCvmClient) DescribeInstances(req *cvm.DescribeInstancesRequest) (*cvm.DescribeInstancesResponse, error) {
	return f.resp, f.err
}

type fakeVpcClient struct {
	// 按 SG ID 返回不同的 egress 规则
	egressBySGID map[string][]*vpc.SecurityGroupPolicy
	err          error
}

func (f *fakeVpcClient) DescribeSecurityGroupPolicies(req *vpc.DescribeSecurityGroupPoliciesRequest) (*vpc.DescribeSecurityGroupPoliciesResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	id := ""
	if req.SecurityGroupId != nil {
		id = *req.SecurityGroupId
	}
	resp := &vpc.DescribeSecurityGroupPoliciesResponse{
		Response: &vpc.DescribeSecurityGroupPoliciesResponseParams{
			SecurityGroupPolicySet: &vpc.SecurityGroupPolicySet{
				Egress: f.egressBySGID[id],
			},
		},
	}
	return resp, nil
}

func makeCvmResp(sgIds ...string) *cvm.DescribeInstancesResponse {
	ptrs := make([]*string, 0, len(sgIds))
	for _, id := range sgIds {
		ptrs = append(ptrs, common.StringPtr(id))
	}
	return &cvm.DescribeInstancesResponse{
		Response: &cvm.DescribeInstancesResponseParams{
			InstanceSet: []*cvm.Instance{
				{SecurityGroupIds: ptrs},
			},
		},
	}
}

func TestDiagnoseInstanceEgress_NilResponse(t *testing.T) {
	cvmCli := &fakeCvmClient{}
	vpcCli := &fakeVpcClient{}
	blocked, err := diagnoseInstanceEgressWith(context.Background(), cvmCli, vpcCli, "ins-xxx")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if blocked {
		t.Error("nil resp should not be blocked")
	}
}

func TestDiagnoseInstanceEgress_NoSecurityGroups(t *testing.T) {
	cvmCli := &fakeCvmClient{resp: makeCvmResp()}
	vpcCli := &fakeVpcClient{}
	blocked, err := diagnoseInstanceEgressWith(context.Background(), cvmCli, vpcCli, "ins-xxx")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if blocked {
		t.Error("instance without SG should not be blocked (default allow)")
	}
}

func TestDiagnoseInstanceEgress_BlockedWhenEmptyEgress(t *testing.T) {
	cvmCli := &fakeCvmClient{resp: makeCvmResp("sg-1")}
	vpcCli := &fakeVpcClient{
		egressBySGID: map[string][]*vpc.SecurityGroupPolicy{
			"sg-1": nil, // 空 egress
		},
	}
	blocked, err := diagnoseInstanceEgressWith(context.Background(), cvmCli, vpcCli, "ins-xxx")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !blocked {
		t.Error("empty egress should be blocked=true")
	}
}

func TestDiagnoseInstanceEgress_NotBlockedWhenAccept(t *testing.T) {
	cvmCli := &fakeCvmClient{resp: makeCvmResp("sg-1")}
	vpcCli := &fakeVpcClient{
		egressBySGID: map[string][]*vpc.SecurityGroupPolicy{
			"sg-1": {policy("ACCEPT", "0.0.0.0/0", "ALL", "ALL")},
		},
	}
	blocked, err := diagnoseInstanceEgressWith(context.Background(), cvmCli, vpcCli, "ins-xxx")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if blocked {
		t.Error("ACCEPT rule should not be blocked")
	}
}

func TestDiagnoseInstanceEgress_BlockedWhenMultiSGAllEmpty(t *testing.T) {
	cvmCli := &fakeCvmClient{resp: makeCvmResp("sg-1", "sg-2")}
	vpcCli := &fakeVpcClient{
		egressBySGID: map[string][]*vpc.SecurityGroupPolicy{
			"sg-1": nil,
			"sg-2": {policy("DROP", "0.0.0.0/0", "ALL", "ALL")},
		},
	}
	blocked, err := diagnoseInstanceEgressWith(context.Background(), cvmCli, vpcCli, "ins-xxx")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !blocked {
		t.Error("all SGs empty/drop → blocked=true")
	}
}

func TestDiagnoseInstanceEgress_NotBlockedIfAnySGHasAccept(t *testing.T) {
	cvmCli := &fakeCvmClient{resp: makeCvmResp("sg-1", "sg-2")}
	vpcCli := &fakeVpcClient{
		egressBySGID: map[string][]*vpc.SecurityGroupPolicy{
			"sg-1": nil,
			"sg-2": {policy("ACCEPT", "0.0.0.0/0", "TCP", "443")},
		},
	}
	blocked, err := diagnoseInstanceEgressWith(context.Background(), cvmCli, vpcCli, "ins-xxx")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if blocked {
		t.Error("any ACCEPT in any SG → not blocked")
	}
}

func TestDiagnoseInstanceEgress_CvmSDKErrorReturnsErrNotBlocked(t *testing.T) {
	cvmCli := &fakeCvmClient{err: errors.New("sdk boom")}
	vpcCli := &fakeVpcClient{}
	blocked, err := diagnoseInstanceEgressWith(context.Background(), cvmCli, vpcCli, "ins-xxx")
	if err == nil {
		t.Error("expected err on CVM SDK failure")
	}
	if blocked {
		t.Error("on SDK error, blocked must be false (不能给出错误诊断)")
	}
}

func TestDiagnoseInstanceEgress_VpcSDKErrorReturnsErrNotBlocked(t *testing.T) {
	cvmCli := &fakeCvmClient{resp: makeCvmResp("sg-1")}
	vpcCli := &fakeVpcClient{err: errors.New("vpc boom")}
	blocked, err := diagnoseInstanceEgressWith(context.Background(), cvmCli, vpcCli, "ins-xxx")
	if err == nil {
		t.Error("expected err on VPC SDK failure")
	}
	if blocked {
		t.Error("on SDK error, blocked must be false")
	}
}

func TestDiagnoseInstanceEgress_EmptyUUID(t *testing.T) {
	blocked, err := DiagnoseInstanceEgress(context.Background(), "")
	if err == nil {
		t.Error("empty uuid should return error")
	}
	if blocked {
		t.Error("empty uuid must return blocked=false")
	}
}

// ========== maybeWrapEgressBlocked ==========

func TestMaybeWrapEgressBlocked_NilErr(t *testing.T) {
	if got := maybeWrapEgressBlocked(context.Background(), "ins-x", nil); got != nil {
		t.Errorf("nil err should passthrough, got %v", got)
	}
}

// withDiagnoseInstanceEgress 临时替换包级 DiagnoseInstanceEgress 为 mock，
// 返回 cleanup 还原原实现。测试隔离用。
func withDiagnoseInstanceEgress(t *testing.T, fn func(ctx context.Context, instanceID string) (bool, error)) {
	t.Helper()
	orig := DiagnoseInstanceEgress
	DiagnoseInstanceEgress = fn
	t.Cleanup(func() { DiagnoseInstanceEgress = orig })
}

// TestMaybeWrapEgressBlocked_BlockedReplacesMessage 命中 blocked=true 时应替换面向
// 用户文案为 egress blocked 消息，并把原错误细节保留到 Detail。
func TestMaybeWrapEgressBlocked_BlockedReplacesMessage(t *testing.T) {
	withDiagnoseInstanceEgress(t, func(ctx context.Context, instanceID string) (bool, error) {
		return true, nil
	})

	orig := hcommon.I18nError(i18n.MsgTATExecuteCommandFailed)
	err := maybeWrapEgressBlocked(context.Background(), "ins-blocked", orig)
	wanted := hcommon.I18nError(i18n.MsgEgressBlocked)
	if !errors.Is(wanted, err) {
		t.Errorf("Message 应被替换为 egress blocked 消息，实际=%q", hcommon.ErrorMessageWithCtx(context.Background(), err))
	}
	var re *hcommon.RichError
	if !errors.As(err, &re) {
		t.Errorf("应该返回 RichError")
	}
	if hcommon.ErrorDetailWithCtx(context.Background(), re) != hcommon.ErrorDetailWithCtx(context.Background(), orig) {
		t.Errorf("Detail 应保留原 Detail=%q，实际=%q", hcommon.ErrorDetailWithCtx(context.Background(), orig), hcommon.ErrorDetailWithCtx(context.Background(), re))
	}
}

// TestMaybeWrapEgressBlocked_NotBlockedPassthrough blocked=false 时原错误透传，
// 不做替换，避免对非出站问题误报。
func TestMaybeWrapEgressBlocked_NotBlockedPassthrough(t *testing.T) {
	withDiagnoseInstanceEgress(t, func(ctx context.Context, instanceID string) (bool, error) {
		return false, nil
	})

	orig := hcommon.I18nError(i18n.MsgTATScriptWithError, "脚本语法错误", "")
	got := maybeWrapEgressBlocked(context.Background(), "ins-x", orig)
	if got != orig {
		t.Errorf("blocked=false 应透传原错误，实际=%v", got)
	}
}

// TestMaybeWrapEgressBlocked_DiagErrorPassthrough 诊断失败（云 API 不可用 /
// 凭证问题）时原错误透传，不干扰业务失败路径。
func TestMaybeWrapEgressBlocked_DiagErrorPassthrough(t *testing.T) {
	withDiagnoseInstanceEgress(t, func(ctx context.Context, instanceID string) (bool, error) {
		return false, hcommon.I18nError(i18n.MsgQueryInstanceFailed)
	})

	orig := hcommon.I18nError(i18n.MsgTATScriptWithError, "脚本业务错误", "")
	got := maybeWrapEgressBlocked(context.Background(), "ins-x", orig)
	if got != orig {
		t.Errorf("诊断失败应透传原错误，实际=%v", got)
	}
}

// TestMaybeWrapEgressBlocked_PreservesRichErrorFields 原错误是 *RichError 时，
// RequestId / BizRequestId / InstanceId 等上下文字段应跟随迁移到新的 RichError。
func TestMaybeWrapEgressBlocked_PreservesRichErrorFields(t *testing.T) {
	withDiagnoseInstanceEgress(t, func(ctx context.Context, instanceID string) (bool, error) {
		return true, nil
	})

	orig := &hcommon.RichError{
		RequestId:    "req-abc",
		BizRequestId: "biz-123",
		InstanceId:   "ins-orig",
	}
	wrapped := maybeWrapEgressBlocked(context.Background(), "ins-orig", orig)

	var re *hcommon.RichError
	if !errors.As(wrapped, &re) {
		t.Fatalf("wrapped 应是 *RichError，实际=%T", wrapped)
	}
	if re.RequestId != orig.RequestId {
		t.Errorf("RequestId 应保留，实际=%q", re.RequestId)
	}
	if re.BizRequestId != orig.BizRequestId {
		t.Errorf("BizRequestId 应保留，实际=%q", re.BizRequestId)
	}
	if re.InstanceId != orig.InstanceId {
		t.Errorf("InstanceId 应保留，实际=%q", re.InstanceId)
	}
}
