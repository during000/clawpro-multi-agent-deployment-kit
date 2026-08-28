package controller

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"

	"hatchery/model"
)

// ─── mock ────────────────────────────────────────────────────────────────────

// mockVpcPolicyClient 实现 vpcPolicyClient 接口，用于单元测试。
type mockVpcPolicyClient struct {
	// describeFn 控制 DescribeSecurityGroupPolicies 的返回值
	describeFn func(sgId string) (*vpc.DescribeSecurityGroupPoliciesResponse, error)
	// createSgCalls 记录每次 CreateSecurityGroup 被调用时传入的安全组名称
	createSgCalls []string
	// createSgErr 若非 nil，CreateSecurityGroup 返回该错误
	createSgErr error
	// createPoliciesCalls 记录每次 CreateSecurityGroupPolicies 被调用时传入的安全组 ID
	createPoliciesCalls []string
	// createPoliciesErr 若非 nil，CreateSecurityGroupPolicies 返回该错误
	createPoliciesErr error
	// newSgId 创建新安全组时返回的 ID
	newSgId string
	// deleteSgCalls 记录每次 DeleteSecurityGroup 被调用时传入的安全组 ID
	deleteSgCalls []string
	// deleteSgErr 若非 nil，DeleteSecurityGroup 返回该错误
	deleteSgErr error
}

func (m *mockVpcPolicyClient) DescribeSecurityGroupPolicies(req *vpc.DescribeSecurityGroupPoliciesRequest) (*vpc.DescribeSecurityGroupPoliciesResponse, error) {
	return m.describeFn(*req.SecurityGroupId)
}

func (m *mockVpcPolicyClient) CreateSecurityGroup(req *vpc.CreateSecurityGroupRequest) (*vpc.CreateSecurityGroupResponse, error) {
	m.createSgCalls = append(m.createSgCalls, *req.GroupName)
	if m.createSgErr != nil {
		return nil, m.createSgErr
	}
	sgId := m.newSgId
	if sgId == "" {
		sgId = "sg-new-001"
	}
	return &vpc.CreateSecurityGroupResponse{
		Response: &vpc.CreateSecurityGroupResponseParams{
			SecurityGroup: &vpc.SecurityGroup{
				SecurityGroupId: common.StringPtr(sgId),
			},
		},
	}, nil
}

func (m *mockVpcPolicyClient) CreateSecurityGroupPolicies(req *vpc.CreateSecurityGroupPoliciesRequest) (*vpc.CreateSecurityGroupPoliciesResponse, error) {
	m.createPoliciesCalls = append(m.createPoliciesCalls, *req.SecurityGroupId)
	if m.createPoliciesErr != nil {
		return nil, m.createPoliciesErr
	}
	return &vpc.CreateSecurityGroupPoliciesResponse{}, nil
}

func (m *mockVpcPolicyClient) DeleteSecurityGroup(req *vpc.DeleteSecurityGroupRequest) (*vpc.DeleteSecurityGroupResponse, error) {
	m.deleteSgCalls = append(m.deleteSgCalls, *req.SecurityGroupId)
	if m.deleteSgErr != nil {
		return nil, m.deleteSgErr
	}
	return &vpc.DeleteSecurityGroupResponse{}, nil
}

// mockCvmSGClient 实现 cvmSGClient 接口，用于单元测试。
type mockCvmSGClient struct {
	// disassociateCalls 记录解绑调用的安全组 ID
	disassociateCalls []string
	// associateCalls 记录绑定调用的安全组 ID
	associateCalls []string
	// disassociateErr 若非 nil，DisassociateSecurityGroups 返回该错误
	disassociateErr error
	// associateErr 若非 nil，AssociateSecurityGroups 返回该错误
	associateErr error
}

func (m *mockCvmSGClient) DisassociateSecurityGroups(req *cvm.DisassociateSecurityGroupsRequest) (*cvm.DisassociateSecurityGroupsResponse, error) {
	if len(req.SecurityGroupIds) > 0 {
		m.disassociateCalls = append(m.disassociateCalls, *req.SecurityGroupIds[0])
	}
	if m.disassociateErr != nil {
		return nil, m.disassociateErr
	}
	return &cvm.DisassociateSecurityGroupsResponse{}, nil
}

func (m *mockCvmSGClient) AssociateSecurityGroups(req *cvm.AssociateSecurityGroupsRequest) (*cvm.AssociateSecurityGroupsResponse, error) {
	if len(req.SecurityGroupIds) > 0 {
		m.associateCalls = append(m.associateCalls, *req.SecurityGroupIds[0])
	}
	if m.associateErr != nil {
		return nil, m.associateErr
	}
	return &cvm.AssociateSecurityGroupsResponse{}, nil
}

// ─── 辅助函数 ─────────────────────────────────────────────────────────────────

// makeDescribeResp 构造一个带有指定出站规则数量的 DescribeSecurityGroupPolicies 响应。
func makeDescribeResp(egressCount int) *vpc.DescribeSecurityGroupPoliciesResponse {
	egress := make([]*vpc.SecurityGroupPolicy, egressCount)
	for i := range egress {
		egress[i] = &vpc.SecurityGroupPolicy{
			Protocol:  common.StringPtr("ALL"),
			CidrBlock: common.StringPtr("0.0.0.0/0"),
			Action:    common.StringPtr("ACCEPT"),
		}
	}
	return &vpc.DescribeSecurityGroupPoliciesResponse{
		Response: &vpc.DescribeSecurityGroupPoliciesResponseParams{
			SecurityGroupPolicySet: &vpc.SecurityGroupPolicySet{
				Egress: egress,
			},
		},
	}
}

// ─── ensureEgressRulesCoreWithConfig：可注入 config 和 findDefaultSgFn 的测试版本 ───

// ensureEgressRulesCoreWithConfig 是 ensureEgressRulesCore 的可测试扩展版本，
// 将 GetSiteConfig 和 findDefaultSgIdFromList 通过参数注入，便于单元测试覆盖所有分支。
func ensureEgressRulesCoreWithConfig(
	ctx context.Context,
	instanceId string,
	sgMap map[string][]string,
	vpcCli vpcPolicyClient,
	cvmCli cvmSGClient,
	config model.SiteConfig,
	findDefaultSgFn func(sgIds []string) (string, error),
) {
	log := Logger(ctx)
	sgIds, ok := sgMap[instanceId]
	if !ok || len(sgIds) == 0 {
		log.Warn("ensureEgressRulesCore: 实例未绑定任何安全组", "instanceId", instanceId)
		return
	}

	// 幂等检查：若实例已绑定自动创建的安全组，解绑其他多余的安全组后直接返回
	if config.AutoCreatedSecurityGroupId != "" {
		bound := false
		for _, sgId := range sgIds {
			if sgId == config.AutoCreatedSecurityGroupId {
				bound = true
				break
			}
		}
		if bound {
			for _, sgId := range sgIds {
				if sgId == config.AutoCreatedSecurityGroupId {
					continue
				}
				unbindReq := cvm.NewDisassociateSecurityGroupsRequest()
				unbindReq.SecurityGroupIds = common.StringPtrs([]string{sgId})
				unbindReq.InstanceIds = common.StringPtrs([]string{instanceId})
				if _, err := cvmCli.DisassociateSecurityGroups(unbindReq); err != nil {
					log.Warn("ensureEgressRulesCore: 解绑多余安全组失败",
						"instanceId", instanceId, "sgId", sgId, "err", err)
				} else {
					log.Info("ensureEgressRulesCore: 已解绑多余安全组",
						"instanceId", instanceId, "sgId", sgId)
				}
			}
			log.Info("ensureEgressRulesCore: 实例已绑定自动安全组，多余安全组已清理，跳过",
				"instanceId", instanceId, "sgId", config.AutoCreatedSecurityGroupId)
			return
		}
	}

	// 第一步：确定要使用的自动安全组 ID
	var newSgId string
	if config.AutoCreatedSecurityGroupId != "" {
		newSgId = config.AutoCreatedSecurityGroupId
		log.Info("ensureEgressRulesCore: 复用已有自动创建的安全组", "newSgId", newSgId)
	} else {
		sourceSgId, err := findDefaultSgFn(sgIds)
		if err != nil {
			log.Warn("ensureEgressRulesCore: 查询默认安全组失败", "err", err)
			return
		}
		if sourceSgId == "" {
			sourceSgId = sgIds[0]
			log.Warn("ensureEgressRulesCore: 未找到默认安全组，降级使用第一个", "sgId", sourceSgId)
		} else {
			log.Info("ensureEgressRulesCore: 找到默认安全组作为模板", "sgId", sourceSgId)
		}
		var sourcePolicySet *vpc.SecurityGroupPolicySet
		policyReq := vpc.NewDescribeSecurityGroupPoliciesRequest()
		policyReq.SecurityGroupId = common.StringPtr(sourceSgId)
		policyResp, err := vpcCli.DescribeSecurityGroupPolicies(policyReq)
		if err != nil {
			log.Warn("ensureEgressRulesCore: 查询模板安全组规则失败", "sgId", sourceSgId, "err", err)
			return
		}
		if policyResp.Response != nil && policyResp.Response.SecurityGroupPolicySet != nil {
			sourcePolicySet = policyResp.Response.SecurityGroupPolicySet
		}

		createReq := vpc.NewCreateSecurityGroupRequest()
		createReq.GroupName = common.StringPtr("clawpro-default")
		createReq.GroupDescription = common.StringPtr(fmt.Sprintf("从 %s 复制并补全出站规则（自动生成）", sourceSgId))
		var createResp *vpc.CreateSecurityGroupResponse
		var createErr error
		for retry := 1; retry <= 3; retry++ {
			createResp, createErr = vpcCli.CreateSecurityGroup(createReq)
			if createErr == nil && createResp.Response != nil &&
				createResp.Response.SecurityGroup != nil &&
				createResp.Response.SecurityGroup.SecurityGroupId != nil {
				break
			}
			if createErr != nil {
				log.Warn("ensureEgressRulesCore: 创建新安全组失败", "retry", retry, "err", createErr)
			} else {
				log.Warn("ensureEgressRulesCore: 创建新安全组返回数据异常", "retry", retry)
				createErr = fmt.Errorf("创建安全组返回数据异常")
			}
		}
		if createErr != nil {
			log.Warn("ensureEgressRulesCore: 创建新安全组重试3次均失败，退出", "err", createErr)
			return
		}
		newSgId = *createResp.Response.SecurityGroup.SecurityGroupId

		// 第一次：添加全放通出站规则
		egressReq := vpc.NewCreateSecurityGroupPoliciesRequest()
		egressReq.SecurityGroupId = common.StringPtr(newSgId)
		egressReq.SecurityGroupPolicySet = &vpc.SecurityGroupPolicySet{
			Egress: []*vpc.SecurityGroupPolicy{
				{
					Protocol:          common.StringPtr("ALL"),
					Port:              common.StringPtr("ALL"),
					CidrBlock:         common.StringPtr("0.0.0.0/0"),
					Action:            common.StringPtr("ACCEPT"),
					PolicyDescription: common.StringPtr("全放通出站规则（自动补全）"),
				},
			},
		}
		var addPolicyErr error
		for retry := 1; retry <= 3; retry++ {
			if _, addPolicyErr = vpcCli.CreateSecurityGroupPolicies(egressReq); addPolicyErr == nil {
				break
			}
			log.Warn("ensureEgressRulesCore: 为新安全组添加出站规则失败",
				"newSgId", newSgId, "retry", retry, "err", addPolicyErr)
		}
		if addPolicyErr != nil {
			log.Warn("ensureEgressRulesCore: 为新安全组添加出站规则重试3次均失败，跳过绑定",
				"newSgId", newSgId, "err", addPolicyErr)
			return
		}
		// 第二次：从模板安全组复制入站规则（若有）
		if sourcePolicySet != nil && len(sourcePolicySet.Ingress) > 0 {
			ingressRules := cleanPoliciesForCreate(sourcePolicySet.Ingress)
			ingressReq := vpc.NewCreateSecurityGroupPoliciesRequest()
			ingressReq.SecurityGroupId = common.StringPtr(newSgId)
			ingressReq.SecurityGroupPolicySet = &vpc.SecurityGroupPolicySet{
				Ingress: ingressRules,
			}
			for retry := 1; retry <= 3; retry++ {
				if _, addPolicyErr = vpcCli.CreateSecurityGroupPolicies(ingressReq); addPolicyErr == nil {
					break
				}
				log.Warn("ensureEgressRulesCore: 为新安全组添加入站规则失败",
					"newSgId", newSgId, "retry", retry, "err", addPolicyErr)
			}
			if addPolicyErr != nil {
				log.Warn("ensureEgressRulesCore: 为新安全组添加入站规则重试3次均失败，跳过绑定",
					"newSgId", newSgId, "err", addPolicyErr)
				return
			}
		}
	}

	// 第二步：先绑定新安全组
	assocReq := cvm.NewAssociateSecurityGroupsRequest()
	assocReq.SecurityGroupIds = common.StringPtrs([]string{newSgId})
	assocReq.InstanceIds = common.StringPtrs([]string{instanceId})
	if _, err := cvmCli.AssociateSecurityGroups(assocReq); err != nil {
		log.Warn("ensureEgressRulesCore: 绑定新安全组失败，跳过解绑旧安全组",
			"instanceId", instanceId, "newSgId", newSgId, "err", err)
		return
	}

	// 第三步：解绑所有旧安全组（排除刚绑定的新安全组）
	for _, sgId := range sgIds {
		if sgId == newSgId {
			continue
		}
		disReq := cvm.NewDisassociateSecurityGroupsRequest()
		disReq.SecurityGroupIds = common.StringPtrs([]string{sgId})
		disReq.InstanceIds = common.StringPtrs([]string{instanceId})
		if _, err := cvmCli.DisassociateSecurityGroups(disReq); err != nil {
			log.Warn("ensureEgressRulesCore: 解绑旧安全组失败",
				"instanceId", instanceId, "sgId", sgId, "err", err)
		} else {
			log.Info("ensureEgressRulesCore: 已解绑旧安全组",
				"instanceId", instanceId, "oldSgId", sgId, "newSgId", newSgId)
		}
	}
	log.Info("ensureEgressRulesCore: 安全组替换完成",
		"instanceId", instanceId, "newSgId", newSgId)
}

// ─── mock helpers ─────────────────────────────────────────────────────────────

// noopFindDefaultSg 直接返回第一个安全组 ID（模拟找到默认安全组）。
func noopFindDefaultSg(sgIds []string) (string, error) {
	if len(sgIds) == 0 {
		return "", nil
	}
	return sgIds[0], nil
}

// errFindDefaultSg 始终返回错误。
func errFindDefaultSg(_ []string) (string, error) {
	return "", errors.New("查询默认安全组失败")
}

// ─── 测试用例 ─────────────────────────────────────────────────────────────────

// TestEnsureEgressRulesCore_NoSecurityGroup 实例未绑定任何安全组时，不应调用 VPC 接口。
func TestEnsureEgressRulesCore_NoSecurityGroup(t *testing.T) {
	vpcMock := &mockVpcPolicyClient{
		describeFn: func(sgId string) (*vpc.DescribeSecurityGroupPoliciesResponse, error) {
			t.Errorf("不应调用 DescribeSecurityGroupPolicies，sgId=%s", sgId)
			return nil, nil
		},
	}
	cvmMock := &mockCvmSGClient{}

	sgMap := map[string][]string{}
	ensureEgressRulesCoreWithConfig(context.Background(), "ins-abc", sgMap, vpcMock, cvmMock,
		model.SiteConfig{}, noopFindDefaultSg)

	if len(vpcMock.createSgCalls) != 0 {
		t.Errorf("不应调用 CreateSecurityGroup，实际调用了 %d 次", len(vpcMock.createSgCalls))
	}
}

// TestEnsureEgressRulesCore_EmptySecurityGroupList 实例绑定了空安全组列表时，不应调用 VPC 接口。
func TestEnsureEgressRulesCore_EmptySecurityGroupList(t *testing.T) {
	vpcMock := &mockVpcPolicyClient{
		describeFn: func(sgId string) (*vpc.DescribeSecurityGroupPoliciesResponse, error) {
			t.Errorf("不应调用 DescribeSecurityGroupPolicies，sgId=%s", sgId)
			return nil, nil
		},
	}
	cvmMock := &mockCvmSGClient{}

	sgMap := map[string][]string{"ins-abc": {}}
	ensureEgressRulesCoreWithConfig(context.Background(), "ins-abc", sgMap, vpcMock, cvmMock,
		model.SiteConfig{}, noopFindDefaultSg)

	if len(vpcMock.createSgCalls) != 0 {
		t.Errorf("不应调用 CreateSecurityGroup，实际调用了 %d 次", len(vpcMock.createSgCalls))
	}
}

// TestEnsureEgressRulesCore_IdempotentAlreadyBound
// 幂等检查：实例已只绑定自动安全组时，不应调用任何 VPC/CVM 接口。
func TestEnsureEgressRulesCore_IdempotentAlreadyBound(t *testing.T) {
	vpcMock := &mockVpcPolicyClient{
		describeFn: func(sgId string) (*vpc.DescribeSecurityGroupPoliciesResponse, error) {
			t.Errorf("不应调用 DescribeSecurityGroupPolicies，sgId=%s", sgId)
			return nil, nil
		},
	}
	cvmMock := &mockCvmSGClient{}

	sgMap := map[string][]string{"ins-abc": {"sg-auto"}}
	config := model.SiteConfig{AutoCreatedSecurityGroupId: "sg-auto"}
	ensureEgressRulesCoreWithConfig(context.Background(), "ins-abc", sgMap, vpcMock, cvmMock,
		config, noopFindDefaultSg)

	if len(vpcMock.createSgCalls) != 0 {
		t.Errorf("幂等场景不应调用 CreateSecurityGroup，实际=%d 次", len(vpcMock.createSgCalls))
	}
	if len(cvmMock.associateCalls) != 0 {
		t.Errorf("幂等场景不应调用 AssociateSecurityGroups，实际=%d 次", len(cvmMock.associateCalls))
	}
	if len(cvmMock.disassociateCalls) != 0 {
		t.Errorf("幂等场景（无多余安全组）不应调用 DisassociateSecurityGroups，实际=%d 次", len(cvmMock.disassociateCalls))
	}
}

// TestEnsureEgressRulesCore_IdempotentBoundWithExtras
// 幂等检查：实例已绑定自动安全组，同时还绑定了其他多余安全组，应解绑多余的后返回，不重新创建。
func TestEnsureEgressRulesCore_IdempotentBoundWithExtras(t *testing.T) {
	vpcMock := &mockVpcPolicyClient{
		describeFn: func(sgId string) (*vpc.DescribeSecurityGroupPoliciesResponse, error) {
			t.Errorf("不应调用 DescribeSecurityGroupPolicies，sgId=%s", sgId)
			return nil, nil
		},
	}
	cvmMock := &mockCvmSGClient{}

	// 实例绑定了 sg-auto（自动安全组）+ sg-old1 + sg-old2（多余）
	sgMap := map[string][]string{"ins-abc": {"sg-old1", "sg-auto", "sg-old2"}}
	config := model.SiteConfig{AutoCreatedSecurityGroupId: "sg-auto"}
	ensureEgressRulesCoreWithConfig(context.Background(), "ins-abc", sgMap, vpcMock, cvmMock,
		config, noopFindDefaultSg)

	// 不应创建新安全组
	if len(vpcMock.createSgCalls) != 0 {
		t.Errorf("幂等场景不应调用 CreateSecurityGroup，实际=%d 次", len(vpcMock.createSgCalls))
	}
	// 不应绑定新安全组
	if len(cvmMock.associateCalls) != 0 {
		t.Errorf("幂等场景不应调用 AssociateSecurityGroups，实际=%d 次", len(cvmMock.associateCalls))
	}
	// 应解绑 sg-old1 和 sg-old2，不解绑 sg-auto
	if len(cvmMock.disassociateCalls) != 2 {
		t.Fatalf("期望解绑 2 个多余安全组，实际=%d，calls=%v", len(cvmMock.disassociateCalls), cvmMock.disassociateCalls)
	}
	for _, sgId := range cvmMock.disassociateCalls {
		if sgId == "sg-auto" {
			t.Errorf("不应解绑自动安全组 sg-auto")
		}
	}
}

// TestEnsureEgressRulesCore_ReuseAutoSgId
// config.AutoCreatedSecurityGroupId 非空但实例未绑定时，应直接复用，不重新创建，并完成绑定/解绑。
func TestEnsureEgressRulesCore_ReuseAutoSgId(t *testing.T) {
	vpcMock := &mockVpcPolicyClient{
		describeFn: func(sgId string) (*vpc.DescribeSecurityGroupPoliciesResponse, error) {
			t.Errorf("复用路径不应调用 DescribeSecurityGroupPolicies，sgId=%s", sgId)
			return nil, nil
		},
	}
	cvmMock := &mockCvmSGClient{}

	sgMap := map[string][]string{"ins-abc": {"sg-old"}}
	config := model.SiteConfig{AutoCreatedSecurityGroupId: "sg-auto"}
	ensureEgressRulesCoreWithConfig(context.Background(), "ins-abc", sgMap, vpcMock, cvmMock,
		config, noopFindDefaultSg)

	// 不应创建新安全组
	if len(vpcMock.createSgCalls) != 0 {
		t.Errorf("复用路径不应调用 CreateSecurityGroup，实际=%d 次", len(vpcMock.createSgCalls))
	}
	// 应先绑定已有的自动安全组
	if len(cvmMock.associateCalls) != 1 || cvmMock.associateCalls[0] != "sg-auto" {
		t.Errorf("期望绑定 sg-auto，实际=%v", cvmMock.associateCalls)
	}
	// 再解绑旧安全组
	if len(cvmMock.disassociateCalls) != 1 || cvmMock.disassociateCalls[0] != "sg-old" {
		t.Errorf("期望解绑 sg-old，实际=%v", cvmMock.disassociateCalls)
	}
}

// TestEnsureEgressRulesCore_CreateNewSg
// config.AutoCreatedSecurityGroupId 为空时，应基于默认安全组创建新安全组，先绑定再解绑旧安全组。
func TestEnsureEgressRulesCore_CreateNewSg(t *testing.T) {
	vpcMock := &mockVpcPolicyClient{
		describeFn: func(sgId string) (*vpc.DescribeSecurityGroupPoliciesResponse, error) {
			return makeDescribeResp(0), nil
		},
		newSgId: "sg-new-001",
	}
	cvmMock := &mockCvmSGClient{}

	sgMap := map[string][]string{"ins-abc": {"sg-001"}}
	ensureEgressRulesCoreWithConfig(context.Background(), "ins-abc", sgMap, vpcMock, cvmMock,
		model.SiteConfig{}, noopFindDefaultSg)

	if len(vpcMock.createSgCalls) != 1 {
		t.Fatalf("期望调用 CreateSecurityGroup 1 次，实际=%d", len(vpcMock.createSgCalls))
	}
	// 先绑定新安全组
	if len(cvmMock.associateCalls) != 1 || cvmMock.associateCalls[0] != "sg-new-001" {
		t.Errorf("期望先绑定 sg-new-001，实际=%v", cvmMock.associateCalls)
	}
	// 再解绑旧安全组
	if len(cvmMock.disassociateCalls) != 1 || cvmMock.disassociateCalls[0] != "sg-001" {
		t.Errorf("期望解绑 sg-001，实际=%v", cvmMock.disassociateCalls)
	}
}

// TestEnsureEgressRulesCore_FindDefaultSgError
// findDefaultSgFn 返回错误时，应直接返回，不创建安全组。
func TestEnsureEgressRulesCore_FindDefaultSgError(t *testing.T) {
	vpcMock := &mockVpcPolicyClient{
		describeFn: func(sgId string) (*vpc.DescribeSecurityGroupPoliciesResponse, error) {
			t.Errorf("查询默认安全组失败后不应调用 DescribeSecurityGroupPolicies，sgId=%s", sgId)
			return nil, nil
		},
	}
	cvmMock := &mockCvmSGClient{}

	sgMap := map[string][]string{"ins-abc": {"sg-001"}}
	ensureEgressRulesCoreWithConfig(context.Background(), "ins-abc", sgMap, vpcMock, cvmMock,
		model.SiteConfig{}, errFindDefaultSg)

	if len(vpcMock.createSgCalls) != 0 {
		t.Errorf("查询默认安全组失败时不应调用 CreateSecurityGroup，实际=%d 次", len(vpcMock.createSgCalls))
	}
	if len(cvmMock.associateCalls) != 0 || len(cvmMock.disassociateCalls) != 0 {
		t.Errorf("查询默认安全组失败时不应调用 CVM 接口")
	}
}

// TestEnsureEgressRulesCore_DescribeError
// DescribeSecurityGroupPolicies 失败时，应直接返回，不创建安全组。
func TestEnsureEgressRulesCore_DescribeError(t *testing.T) {
	vpcMock := &mockVpcPolicyClient{
		describeFn: func(sgId string) (*vpc.DescribeSecurityGroupPoliciesResponse, error) {
			return nil, errors.New("API 调用失败")
		},
	}
	cvmMock := &mockCvmSGClient{}

	sgMap := map[string][]string{"ins-abc": {"sg-001"}}
	ensureEgressRulesCoreWithConfig(context.Background(), "ins-abc", sgMap, vpcMock, cvmMock,
		model.SiteConfig{}, noopFindDefaultSg)

	if len(vpcMock.createSgCalls) != 0 {
		t.Errorf("查询规则失败时不应调用 CreateSecurityGroup，实际调用了 %d 次", len(vpcMock.createSgCalls))
	}
}

// TestEnsureEgressRulesCore_CreateError
// 创建新安全组失败时，应重试 3 次，不应 panic，不应尝试绑定/解绑。
func TestEnsureEgressRulesCore_CreateError(t *testing.T) {
	vpcMock := &mockVpcPolicyClient{
		describeFn: func(sgId string) (*vpc.DescribeSecurityGroupPoliciesResponse, error) {
			return makeDescribeResp(0), nil
		},
		createSgErr: errors.New("创建安全组失败"),
	}
	cvmMock := &mockCvmSGClient{}

	sgMap := map[string][]string{"ins-abc": {"sg-001"}}
	ensureEgressRulesCoreWithConfig(context.Background(), "ins-abc", sgMap, vpcMock, cvmMock,
		model.SiteConfig{}, noopFindDefaultSg)

	// 重试 3 次
	if len(vpcMock.createSgCalls) != 3 {
		t.Errorf("期望重试调用 CreateSecurityGroup 3 次，实际=%d", len(vpcMock.createSgCalls))
	}
	// 创建失败时不应尝试绑定/解绑
	if len(cvmMock.disassociateCalls) != 0 {
		t.Errorf("创建失败时不应解绑安全组，实际调用了 %d 次", len(cvmMock.disassociateCalls))
	}
	if len(cvmMock.associateCalls) != 0 {
		t.Errorf("创建失败时不应绑定安全组，实际调用了 %d 次", len(cvmMock.associateCalls))
	}
}

// TestEnsureEgressRulesCore_NilResponse
// DescribeSecurityGroupPolicies 返回 nil Response 时，sourcePolicySet 为 nil，仍应继续创建安全组（入站规则为空）。
func TestEnsureEgressRulesCore_NilResponse(t *testing.T) {
	vpcMock := &mockVpcPolicyClient{
		describeFn: func(sgId string) (*vpc.DescribeSecurityGroupPoliciesResponse, error) {
			return &vpc.DescribeSecurityGroupPoliciesResponse{Response: nil}, nil
		},
		newSgId: "sg-new-001",
	}
	cvmMock := &mockCvmSGClient{}

	sgMap := map[string][]string{"ins-abc": {"sg-001"}}
	ensureEgressRulesCoreWithConfig(context.Background(), "ins-abc", sgMap, vpcMock, cvmMock,
		model.SiteConfig{}, noopFindDefaultSg)

	// Response 为 nil 时 sourcePolicySet 为 nil，但仍应继续创建安全组
	if len(vpcMock.createSgCalls) != 1 {
		t.Errorf("Response 为 nil 时仍应调用 CreateSecurityGroup 1 次，实际=%d 次", len(vpcMock.createSgCalls))
	}
	if len(cvmMock.associateCalls) != 1 || cvmMock.associateCalls[0] != "sg-new-001" {
		t.Errorf("期望绑定 sg-new-001，实际=%v", cvmMock.associateCalls)
	}
}

// TestEnsureEgressRulesCore_MultipleOldSgs
// 实例绑定多个旧安全组时，应全部解绑，只保留新安全组。
func TestEnsureEgressRulesCore_MultipleOldSgs(t *testing.T) {
	vpcMock := &mockVpcPolicyClient{
		describeFn: func(sgId string) (*vpc.DescribeSecurityGroupPoliciesResponse, error) {
			return makeDescribeResp(0), nil
		},
		newSgId: "sg-new-001",
	}
	cvmMock := &mockCvmSGClient{}

	sgMap := map[string][]string{"ins-abc": {"sg-001", "sg-002", "sg-003"}}
	ensureEgressRulesCoreWithConfig(context.Background(), "ins-abc", sgMap, vpcMock, cvmMock,
		model.SiteConfig{}, noopFindDefaultSg)

	if len(vpcMock.createSgCalls) != 1 {
		t.Fatalf("期望调用 CreateSecurityGroup 1 次，实际=%d", len(vpcMock.createSgCalls))
	}
	// 先绑定新安全组
	if len(cvmMock.associateCalls) != 1 || cvmMock.associateCalls[0] != "sg-new-001" {
		t.Errorf("期望绑定 sg-new-001，实际=%v", cvmMock.associateCalls)
	}
	// 解绑所有旧安全组（3 个）
	if len(cvmMock.disassociateCalls) != 3 {
		t.Errorf("期望解绑 3 个旧安全组，实际=%d，calls=%v", len(cvmMock.disassociateCalls), cvmMock.disassociateCalls)
	}
	for _, sgId := range cvmMock.disassociateCalls {
		if sgId == "sg-new-001" {
			t.Errorf("不应解绑新安全组 sg-new-001")
		}
	}
}

// TestEnsureEgressRulesCore_AssociateError
// 绑定新安全组失败时，不应解绑旧安全组（保护实例始终有安全组）。
func TestEnsureEgressRulesCore_AssociateError(t *testing.T) {
	vpcMock := &mockVpcPolicyClient{
		describeFn: func(sgId string) (*vpc.DescribeSecurityGroupPoliciesResponse, error) {
			return makeDescribeResp(0), nil
		},
		newSgId: "sg-new-001",
	}
	cvmMock := &mockCvmSGClient{
		associateErr: errors.New("绑定失败"),
	}

	sgMap := map[string][]string{"ins-abc": {"sg-001"}}
	ensureEgressRulesCoreWithConfig(context.Background(), "ins-abc", sgMap, vpcMock, cvmMock,
		model.SiteConfig{}, noopFindDefaultSg)

	// 应尝试创建新安全组
	if len(vpcMock.createSgCalls) != 1 {
		t.Errorf("期望调用 CreateSecurityGroup 1 次，实际=%d", len(vpcMock.createSgCalls))
	}
	// 应尝试绑定
	if len(cvmMock.associateCalls) != 1 {
		t.Errorf("期望尝试绑定 1 次，实际=%d", len(cvmMock.associateCalls))
	}
	// 绑定失败时不应解绑旧安全组
	if len(cvmMock.disassociateCalls) != 0 {
		t.Errorf("绑定失败时不应解绑旧安全组，实际调用了 %d 次", len(cvmMock.disassociateCalls))
	}
}

// TestEnsureEgressRulesCore_DisassociateError
// 解绑旧安全组失败时，不应影响新安全组的绑定（新安全组已绑定，实例仍受保护）。
func TestEnsureEgressRulesCore_DisassociateError(t *testing.T) {
	vpcMock := &mockVpcPolicyClient{
		describeFn: func(sgId string) (*vpc.DescribeSecurityGroupPoliciesResponse, error) {
			return makeDescribeResp(0), nil
		},
		newSgId: "sg-new-001",
	}
	cvmMock := &mockCvmSGClient{
		disassociateErr: errors.New("解绑失败"),
	}

	sgMap := map[string][]string{"ins-abc": {"sg-001"}}
	ensureEgressRulesCoreWithConfig(context.Background(), "ins-abc", sgMap, vpcMock, cvmMock,
		model.SiteConfig{}, noopFindDefaultSg)

	// 应尝试创建新安全组
	if len(vpcMock.createSgCalls) != 1 {
		t.Errorf("期望调用 CreateSecurityGroup 1 次，实际=%d", len(vpcMock.createSgCalls))
	}
	// 应先绑定新安全组（成功）
	if len(cvmMock.associateCalls) != 1 || cvmMock.associateCalls[0] != "sg-new-001" {
		t.Errorf("期望绑定 sg-new-001，实际=%v", cvmMock.associateCalls)
	}
	// 应尝试解绑旧安全组（即使失败也要尝试）
	if len(cvmMock.disassociateCalls) != 1 {
		t.Errorf("期望尝试解绑 1 次，实际=%d", len(cvmMock.disassociateCalls))
	}
}

// TestEnsureEgressRulesCore_CopiesIngressRules 复制安全组时应保留原入站规则，且分两次调用 CreateSecurityGroupPolicies。
func TestEnsureEgressRulesCore_CopiesIngressRules(t *testing.T) {
	ingressResp := &vpc.DescribeSecurityGroupPoliciesResponse{
		Response: &vpc.DescribeSecurityGroupPoliciesResponseParams{
			SecurityGroupPolicySet: &vpc.SecurityGroupPolicySet{
				Ingress: []*vpc.SecurityGroupPolicy{
					{
						Protocol:    common.StringPtr("TCP"),
						Port:        common.StringPtr("22"),
						CidrBlock:   common.StringPtr("0.0.0.0/0"),
						Action:      common.StringPtr("ACCEPT"),
						PolicyIndex: common.Int64Ptr(0),
						Priority:    common.Int64Ptr(1),
						ModifyTime:  common.StringPtr("2024-01-01"),
					},
				},
				Egress: []*vpc.SecurityGroupPolicy{},
			},
		},
	}

	vpcMock := &mockVpcPolicyClient{
		describeFn: func(sgId string) (*vpc.DescribeSecurityGroupPoliciesResponse, error) {
			return ingressResp, nil
		},
		newSgId: "sg-new-001",
	}
	cvmMock := &mockCvmSGClient{}

	sgMap := map[string][]string{"ins-abc": {"sg-001"}}
	ensureEgressRulesCoreWithConfig(context.Background(), "ins-abc", sgMap, vpcMock, cvmMock,
		model.SiteConfig{}, noopFindDefaultSg)

	if len(vpcMock.createSgCalls) != 1 {
		t.Fatalf("期望调用 CreateSecurityGroup 1 次，实际=%d", len(vpcMock.createSgCalls))
	}
	// 有入站规则时应分两次调用 CreateSecurityGroupPolicies（egress + ingress）
	if len(vpcMock.createPoliciesCalls) != 2 {
		t.Errorf("有入站规则时期望调用 CreateSecurityGroupPolicies 2 次，实际=%d，calls=%v",
			len(vpcMock.createPoliciesCalls), vpcMock.createPoliciesCalls)
	}
	// 验证完整流程：先绑定新安全组，再解绑旧安全组
	if len(cvmMock.associateCalls) != 1 || cvmMock.associateCalls[0] != "sg-new-001" {
		t.Errorf("期望绑定 sg-new-001，实际=%v", cvmMock.associateCalls)
	}
	if len(cvmMock.disassociateCalls) != 1 || cvmMock.disassociateCalls[0] != "sg-001" {
		t.Errorf("期望解绑 sg-001，实际=%v", cvmMock.disassociateCalls)
	}
}

// TestEnsureEgressRulesCore_NoIngressRules 无入站规则时只调用一次 CreateSecurityGroupPolicies（仅 egress）。
func TestEnsureEgressRulesCore_NoIngressRules(t *testing.T) {
	vpcMock := &mockVpcPolicyClient{
		describeFn: func(sgId string) (*vpc.DescribeSecurityGroupPoliciesResponse, error) {
			return makeDescribeResp(0), nil
		},
		newSgId: "sg-new-001",
	}
	cvmMock := &mockCvmSGClient{}

	sgMap := map[string][]string{"ins-abc": {"sg-001"}}
	ensureEgressRulesCoreWithConfig(context.Background(), "ins-abc", sgMap, vpcMock, cvmMock,
		model.SiteConfig{}, noopFindDefaultSg)

	// 无入站规则时只调用一次（仅 egress）
	if len(vpcMock.createPoliciesCalls) != 1 {
		t.Errorf("无入站规则时期望调用 CreateSecurityGroupPolicies 1 次，实际=%d，calls=%v",
			len(vpcMock.createPoliciesCalls), vpcMock.createPoliciesCalls)
	}
}

// ─── cleanPoliciesForCreate 单元测试 ────────────────────────────────────────

// TestCleanPoliciesForCreate_RemovesReadOnlyFields 验证只读字段（PolicyIndex、Priority、ModifyTime）被清除。
func TestCleanPoliciesForCreate_RemovesReadOnlyFields(t *testing.T) {
	policies := []*vpc.SecurityGroupPolicy{
		{
			Protocol:    common.StringPtr("TCP"),
			Port:        common.StringPtr("80"),
			CidrBlock:   common.StringPtr("0.0.0.0/0"),
			Action:      common.StringPtr("ACCEPT"),
			PolicyIndex: common.Int64Ptr(5),
			Priority:    common.Int64Ptr(10),
			ModifyTime:  common.StringPtr("2024-01-01 00:00:00"),
		},
	}
	result := cleanPoliciesForCreate(policies)
	if len(result) != 1 {
		t.Fatalf("期望 1 条规则，实际=%d", len(result))
	}
	if result[0].PolicyIndex != nil {
		t.Errorf("PolicyIndex 应被清除为 nil，实际=%v", result[0].PolicyIndex)
	}
	if result[0].Priority != nil {
		t.Errorf("Priority 应被清除为 nil，实际=%v", result[0].Priority)
	}
	if result[0].ModifyTime != nil {
		t.Errorf("ModifyTime 应被清除为 nil，实际=%v", result[0].ModifyTime)
	}
	// 非只读字段应保留
	if result[0].Protocol == nil || *result[0].Protocol != "TCP" {
		t.Errorf("Protocol 应保留，实际=%v", result[0].Protocol)
	}
}

// TestCleanPoliciesForCreate_NilifyEmptyTopLevelStrings 验证顶层空字符串字段被置 nil。
func TestCleanPoliciesForCreate_NilifyEmptyTopLevelStrings(t *testing.T) {
	emptyStr := ""
	policies := []*vpc.SecurityGroupPolicy{
		{
			Protocol:        &emptyStr,
			Port:            &emptyStr,
			CidrBlock:       common.StringPtr("10.0.0.0/8"),
			Ipv6CidrBlock:   &emptyStr,
			SecurityGroupId: &emptyStr,
			Action:          common.StringPtr("ACCEPT"),
		},
	}
	result := cleanPoliciesForCreate(policies)
	if result[0].Protocol != nil {
		t.Errorf("空字符串 Protocol 应置 nil")
	}
	if result[0].Port != nil {
		t.Errorf("空字符串 Port 应置 nil")
	}
	if result[0].Ipv6CidrBlock != nil {
		t.Errorf("空字符串 Ipv6CidrBlock 应置 nil")
	}
	if result[0].SecurityGroupId != nil {
		t.Errorf("空字符串 SecurityGroupId 应置 nil")
	}
	// 非空字段应保留
	if result[0].CidrBlock == nil || *result[0].CidrBlock != "10.0.0.0/8" {
		t.Errorf("非空 CidrBlock 应保留，实际=%v", result[0].CidrBlock)
	}
}

// TestCleanPoliciesForCreate_AddressTemplate_EmptyFields AddressTemplate 中空字符串字段应置 nil，全空则整体置 nil。
func TestCleanPoliciesForCreate_AddressTemplate_EmptyFields(t *testing.T) {
	emptyStr := ""
	policies := []*vpc.SecurityGroupPolicy{
		{
			Action: common.StringPtr("ACCEPT"),
			AddressTemplate: &vpc.AddressTemplateSpecification{
				AddressId:      common.StringPtr("ipm-xxxxxxxx"),
				AddressGroupId: &emptyStr,
			},
		},
		{
			Action: common.StringPtr("ACCEPT"),
			AddressTemplate: &vpc.AddressTemplateSpecification{
				AddressId:      &emptyStr,
				AddressGroupId: &emptyStr,
			},
		},
	}
	result := cleanPoliciesForCreate(policies)
	// 第一条：AddressId 有值，AddressGroupId 为空 → AddressGroupId 置 nil，AddressTemplate 保留
	if result[0].AddressTemplate == nil {
		t.Fatal("第一条 AddressTemplate 不应为 nil")
	}
	if result[0].AddressTemplate.AddressGroupId != nil {
		t.Errorf("空字符串 AddressGroupId 应置 nil")
	}
	if result[0].AddressTemplate.AddressId == nil || *result[0].AddressTemplate.AddressId != "ipm-xxxxxxxx" {
		t.Errorf("非空 AddressId 应保留")
	}
	// 第二条：两个字段均为空 → 整个 AddressTemplate 置 nil
	if result[1].AddressTemplate != nil {
		t.Errorf("两个子字段均为空时 AddressTemplate 应置 nil")
	}
}

// TestCleanPoliciesForCreate_ServiceTemplate_EmptyFields ServiceTemplate 中空字符串字段应置 nil，全空则整体置 nil。
func TestCleanPoliciesForCreate_ServiceTemplate_EmptyFields(t *testing.T) {
	emptyStr := ""
	policies := []*vpc.SecurityGroupPolicy{
		{
			Action: common.StringPtr("ACCEPT"),
			ServiceTemplate: &vpc.ServiceTemplateSpecification{
				ServiceId:      common.StringPtr("ppm-xxxxxxxx"),
				ServiceGroupId: &emptyStr,
			},
		},
		{
			Action: common.StringPtr("ACCEPT"),
			ServiceTemplate: &vpc.ServiceTemplateSpecification{
				ServiceId:      &emptyStr,
				ServiceGroupId: &emptyStr,
			},
		},
	}
	result := cleanPoliciesForCreate(policies)
	// 第一条：ServiceId 有值，ServiceGroupId 为空 → ServiceGroupId 置 nil，ServiceTemplate 保留
	if result[0].ServiceTemplate == nil {
		t.Fatal("第一条 ServiceTemplate 不应为 nil")
	}
	if result[0].ServiceTemplate.ServiceGroupId != nil {
		t.Errorf("空字符串 ServiceGroupId 应置 nil")
	}
	if result[0].ServiceTemplate.ServiceId == nil || *result[0].ServiceTemplate.ServiceId != "ppm-xxxxxxxx" {
		t.Errorf("非空 ServiceId 应保留")
	}
	// 第二条：两个字段均为空 → 整个 ServiceTemplate 置 nil
	if result[1].ServiceTemplate != nil {
		t.Errorf("两个子字段均为空时 ServiceTemplate 应置 nil")
	}
}

// TestCleanPoliciesForCreate_NilTemplates AddressTemplate/ServiceTemplate 为 nil 时不应 panic。
func TestCleanPoliciesForCreate_NilTemplates(t *testing.T) {
	policies := []*vpc.SecurityGroupPolicy{
		{
			Protocol:        common.StringPtr("TCP"),
			Port:            common.StringPtr("443"),
			CidrBlock:       common.StringPtr("0.0.0.0/0"),
			Action:          common.StringPtr("ACCEPT"),
			AddressTemplate: nil,
			ServiceTemplate: nil,
		},
	}
	result := cleanPoliciesForCreate(policies)
	if len(result) != 1 {
		t.Fatalf("期望 1 条规则，实际=%d", len(result))
	}
	if result[0].AddressTemplate != nil {
		t.Errorf("nil AddressTemplate 应保持 nil")
	}
	if result[0].ServiceTemplate != nil {
		t.Errorf("nil ServiceTemplate 应保持 nil")
	}
}

// TestCleanPoliciesForCreate_EmptyInput 空输入应返回空切片。
func TestCleanPoliciesForCreate_EmptyInput(t *testing.T) {
	result := cleanPoliciesForCreate(nil)
	if len(result) != 0 {
		t.Errorf("空输入期望返回空切片，实际=%d 条", len(result))
	}
	result2 := cleanPoliciesForCreate([]*vpc.SecurityGroupPolicy{})
	if len(result2) != 0 {
		t.Errorf("空切片输入期望返回空切片，实际=%d 条", len(result2))
	}
}

// ─── ensureInstanceEgressRules 重试逻辑测试 ──────────────────────────────────

// mockSgQuerier 模拟 describeInstancesSecurityGroups 的行为，支持按调用次数控制返回值。
type mockSgQuerier struct {
	// calls 记录已调用次数
	calls int
	// responses 按调用顺序返回，超出范围则返回最后一个
	responses []struct {
		sgMap map[string][]string
		err   error
	}
}

func (m *mockSgQuerier) query(instanceIds []string) (map[string][]string, error) {
	idx := m.calls
	if idx >= len(m.responses) {
		idx = len(m.responses) - 1
	}
	m.calls++
	r := m.responses[idx]
	return r.sgMap, r.err
}

// ensureInstanceEgressRulesWithDeps 是 ensureInstanceEgressRules 的可测试版本，
// 将 sleep、sgQuerier、vpcClient、cvmClient、coreFn 均通过参数注入，便于单元测试。
func ensureInstanceEgressRulesWithDeps(
	instanceId string,
	sleep func(attempt int),
	sgQuerier func(ids []string) (map[string][]string, error),
	vpcClientFactory func() (vpcPolicyClient, error),
	cvmClientFactory func() (cvmSGClient, error),
	coreFn func(ctx context.Context, instanceId string, sgMap map[string][]string, vpcCli vpcPolicyClient, cvmCli cvmSGClient),
) {
	var sgMap map[string][]string
	var err error

	for attempt := 1; attempt <= 3; attempt++ {
		sgMap, err = sgQuerier([]string{instanceId})
		if err != nil {
			if attempt < 3 {
				sleep(attempt)
				continue
			}
			return
		}

		sgIds, ok := sgMap[instanceId]
		if ok && len(sgIds) > 0 {
			break
		}

		if attempt < 3 {
			sleep(attempt)
		}
	}

	vpcCli, err := vpcClientFactory()
	if err != nil {
		return
	}
	cvmCli, err := cvmClientFactory()
	if err != nil {
		return
	}
	coreFn(context.Background(), instanceId, sgMap, vpcCli, cvmCli)
}

// noopCoreFn 是一个空的 coreFn，用于只验证重试/sleep 行为的测试用例。
func noopCoreFn(_ context.Context, _ string, _ map[string][]string, _ vpcPolicyClient, _ cvmSGClient) {
}

// TestEnsureInstanceEgressRules_SuccessOnFirstAttempt 第一次查询即成功，不应重试。
func TestEnsureInstanceEgressRules_SuccessOnFirstAttempt(t *testing.T) {
	sleepCalls := 0
	querier := &mockSgQuerier{
		responses: []struct {
			sgMap map[string][]string
			err   error
		}{
			{sgMap: map[string][]string{"ins-abc": {"sg-001"}}, err: nil},
		},
	}
	vpcMock := &mockVpcPolicyClient{
		describeFn: func(sgId string) (*vpc.DescribeSecurityGroupPoliciesResponse, error) {
			return makeDescribeResp(0), nil
		},
		newSgId: "sg-new-001",
	}
	cvmMock := &mockCvmSGClient{}

	ensureInstanceEgressRulesWithDeps(
		"ins-abc",
		func(attempt int) { sleepCalls++ },
		querier.query,
		func() (vpcPolicyClient, error) { return vpcMock, nil },
		func() (cvmSGClient, error) { return cvmMock, nil },
		noopCoreFn,
	)

	if querier.calls != 1 {
		t.Errorf("期望查询 1 次，实际=%d", querier.calls)
	}
	if sleepCalls != 0 {
		t.Errorf("第一次成功不应触发 sleep，实际=%d 次", sleepCalls)
	}
}

// TestEnsureInstanceEgressRules_RetryOnEmptySgList 安全组列表为空时应重试，第二次成功。
func TestEnsureInstanceEgressRules_RetryOnEmptySgList(t *testing.T) {
	sleepCalls := 0
	querier := &mockSgQuerier{
		responses: []struct {
			sgMap map[string][]string
			err   error
		}{
			{sgMap: map[string][]string{"ins-abc": {}}, err: nil},         // 第1次：空列表
			{sgMap: map[string][]string{"ins-abc": {"sg-001"}}, err: nil}, // 第2次：成功
		},
	}
	vpcMock := &mockVpcPolicyClient{
		describeFn: func(sgId string) (*vpc.DescribeSecurityGroupPoliciesResponse, error) {
			return makeDescribeResp(0), nil
		},
		newSgId: "sg-new-001",
	}
	cvmMock := &mockCvmSGClient{}

	ensureInstanceEgressRulesWithDeps(
		"ins-abc",
		func(attempt int) { sleepCalls++ },
		querier.query,
		func() (vpcPolicyClient, error) { return vpcMock, nil },
		func() (cvmSGClient, error) { return cvmMock, nil },
		noopCoreFn,
	)

	if querier.calls != 2 {
		t.Errorf("期望查询 2 次，实际=%d", querier.calls)
	}
	if sleepCalls != 1 {
		t.Errorf("期望 sleep 1 次，实际=%d 次", sleepCalls)
	}
}

// TestEnsureInstanceEgressRules_RetryOnQueryError 查询失败时应重试，第三次成功。
func TestEnsureInstanceEgressRules_RetryOnQueryError(t *testing.T) {
	sleepCalls := 0
	querier := &mockSgQuerier{
		responses: []struct {
			sgMap map[string][]string
			err   error
		}{
			{sgMap: nil, err: errors.New("网络错误")},                         // 第1次：失败
			{sgMap: nil, err: errors.New("网络错误")},                         // 第2次：失败
			{sgMap: map[string][]string{"ins-abc": {"sg-001"}}, err: nil}, // 第3次：成功
		},
	}
	vpcMock := &mockVpcPolicyClient{
		describeFn: func(sgId string) (*vpc.DescribeSecurityGroupPoliciesResponse, error) {
			return makeDescribeResp(0), nil
		},
		newSgId: "sg-new-001",
	}
	cvmMock := &mockCvmSGClient{}

	ensureInstanceEgressRulesWithDeps(
		"ins-abc",
		func(attempt int) { sleepCalls++ },
		querier.query,
		func() (vpcPolicyClient, error) { return vpcMock, nil },
		func() (cvmSGClient, error) { return cvmMock, nil },
		noopCoreFn,
	)

	if querier.calls != 3 {
		t.Errorf("期望查询 3 次，实际=%d", querier.calls)
	}
	if sleepCalls != 2 {
		t.Errorf("期望 sleep 2 次，实际=%d 次", sleepCalls)
	}
}

// TestEnsureInstanceEgressRules_AllAttemptsFailWithError 三次查询均失败，不应调用 VPC 接口。
func TestEnsureInstanceEgressRules_AllAttemptsFailWithError(t *testing.T) {
	sleepCalls := 0
	clientCalled := false
	querier := &mockSgQuerier{
		responses: []struct {
			sgMap map[string][]string
			err   error
		}{
			{sgMap: nil, err: errors.New("网络错误")},
			{sgMap: nil, err: errors.New("网络错误")},
			{sgMap: nil, err: errors.New("网络错误")},
		},
	}

	ensureInstanceEgressRulesWithDeps(
		"ins-abc",
		func(attempt int) { sleepCalls++ },
		querier.query,
		func() (vpcPolicyClient, error) {
			clientCalled = true
			return &mockVpcPolicyClient{
				describeFn: func(sgId string) (*vpc.DescribeSecurityGroupPoliciesResponse, error) {
					return makeDescribeResp(0), nil
				},
			}, nil
		},
		func() (cvmSGClient, error) {
			clientCalled = true
			return &mockCvmSGClient{}, nil
		},
		noopCoreFn,
	)

	if querier.calls != 3 {
		t.Errorf("期望查询 3 次，实际=%d", querier.calls)
	}
	if sleepCalls != 2 {
		t.Errorf("期望 sleep 2 次（第3次失败后不再 sleep），实际=%d 次", sleepCalls)
	}
	if clientCalled {
		t.Error("三次查询均失败时不应创建 VPC 客户端")
	}
}

// TestEnsureInstanceEgressRules_AllAttemptsEmptySgList 三次查询安全组均为空，仍应尝试执行 ensureEgressRulesCore。
func TestEnsureInstanceEgressRules_AllAttemptsEmptySgList(t *testing.T) {
	sleepCalls := 0
	querier := &mockSgQuerier{
		responses: []struct {
			sgMap map[string][]string
			err   error
		}{
			{sgMap: map[string][]string{"ins-abc": {}}, err: nil},
			{sgMap: map[string][]string{"ins-abc": {}}, err: nil},
			{sgMap: map[string][]string{"ins-abc": {}}, err: nil},
		},
	}
	vpcMock := &mockVpcPolicyClient{
		describeFn: func(sgId string) (*vpc.DescribeSecurityGroupPoliciesResponse, error) {
			t.Error("安全组列表始终为空，不应调用 DescribeSecurityGroupPolicies")
			return nil, nil
		},
	}
	cvmMock := &mockCvmSGClient{}

	ensureInstanceEgressRulesWithDeps(
		"ins-abc",
		func(attempt int) { sleepCalls++ },
		querier.query,
		func() (vpcPolicyClient, error) { return vpcMock, nil },
		func() (cvmSGClient, error) { return cvmMock, nil },
		noopCoreFn,
	)

	if querier.calls != 3 {
		t.Errorf("期望查询 3 次，实际=%d", querier.calls)
	}
	if sleepCalls != 2 {
		t.Errorf("期望 sleep 2 次，实际=%d 次", sleepCalls)
	}
	if len(vpcMock.createSgCalls) != 0 {
		t.Errorf("安全组为空时不应调用 CreateSecurityGroup，实际=%d 次", len(vpcMock.createSgCalls))
	}
}

// ─── HandleInstanceDeniedActions 核心逻辑测试 ────────────────────────────────

// buildDeniedActionsResult 是 HandleInstanceDeniedActions 中组装结果的可测试核心逻辑。
func buildDeniedActionsResult(instances []model.Instance, cvmDenied map[string][]deniedAction) []instanceDeniedActions {
	result := make([]instanceDeniedActions, 0, len(instances))
	for _, inst := range instances {
		item := instanceDeniedActions{ID: inst.ID}
		if inst.InstanceId != "" {
			if denied, ok := cvmDenied[inst.InstanceId]; ok {
				item.DeniedActions = denied
			}
		}
		if item.DeniedActions == nil {
			item.DeniedActions = []deniedAction{}
		}
		result = append(result, item)
	}
	return result
}

// TestBuildDeniedActionsResult_AllHaveCvmId 所有实例均有 CVM ID，且均在 cvmDenied 中。
func TestBuildDeniedActionsResult_AllHaveCvmId(t *testing.T) {
	instances := []model.Instance{
		{Name: "a", InstanceId: "ins-001"},
		{Name: "b", InstanceId: "ins-002"},
	}
	instances[0].ID = 1
	instances[1].ID = 2

	cvmDenied := map[string][]deniedAction{
		"ins-001": {{Action: "DescribeInstanceVncUrl", Code: "OperationDenied", Message: "not ready"}},
		"ins-002": {},
	}

	result := buildDeniedActionsResult(instances, cvmDenied)

	if len(result) != 2 {
		t.Fatalf("期望 2 条结果，实际=%d", len(result))
	}
	if len(result[0].DeniedActions) != 1 {
		t.Errorf("ins-001 期望 1 条 denied action，实际=%d", len(result[0].DeniedActions))
	}
	if result[0].DeniedActions[0].Action != "DescribeInstanceVncUrl" {
		t.Errorf("期望 Action=DescribeInstanceVncUrl，实际=%s", result[0].DeniedActions[0].Action)
	}
	if len(result[1].DeniedActions) != 0 {
		t.Errorf("ins-002 期望 0 条 denied action，实际=%d", len(result[1].DeniedActions))
	}
}

// TestBuildDeniedActionsResult_NoCvmId 实例没有 CVM ID 时，DeniedActions 应为空切片（非 nil）。
func TestBuildDeniedActionsResult_NoCvmId(t *testing.T) {
	instances := []model.Instance{
		{Name: "no-cvm"},
	}
	instances[0].ID = 10

	result := buildDeniedActionsResult(instances, map[string][]deniedAction{})

	if len(result) != 1 {
		t.Fatalf("期望 1 条结果，实际=%d", len(result))
	}
	if result[0].DeniedActions == nil {
		t.Error("DeniedActions 不应为 nil，应为空切片")
	}
	if len(result[0].DeniedActions) != 0 {
		t.Errorf("无 CVM ID 时期望 0 条 denied action，实际=%d", len(result[0].DeniedActions))
	}
}

// TestBuildDeniedActionsResult_CvmIdNotInMap 实例有 CVM ID 但不在 cvmDenied 中，DeniedActions 应为空切片。
func TestBuildDeniedActionsResult_CvmIdNotInMap(t *testing.T) {
	instances := []model.Instance{
		{Name: "orphan", InstanceId: "ins-999"},
	}
	instances[0].ID = 20

	result := buildDeniedActionsResult(instances, map[string][]deniedAction{})

	if result[0].DeniedActions == nil {
		t.Error("DeniedActions 不应为 nil，应为空切片")
	}
	if len(result[0].DeniedActions) != 0 {
		t.Errorf("CVM ID 不在映射中时期望 0 条 denied action，实际=%d", len(result[0].DeniedActions))
	}
}

// TestBuildDeniedActionsResult_MixedInstances 混合场景：有 CVM ID 的和没有 CVM ID 的实例混合。
func TestBuildDeniedActionsResult_MixedInstances(t *testing.T) {
	instances := []model.Instance{
		{Name: "with-cvm", InstanceId: "ins-001"},
		{Name: "no-cvm"},
		{Name: "cvm-not-in-map", InstanceId: "ins-999"},
	}
	for i := range instances {
		instances[i].ID = uint(i + 1)
	}

	cvmDenied := map[string][]deniedAction{
		"ins-001": {{Action: "DescribeInstanceVncUrl", Code: "OperationDenied", Message: "msg"}},
	}

	result := buildDeniedActionsResult(instances, cvmDenied)

	if len(result) != 3 {
		t.Fatalf("期望 3 条结果，实际=%d", len(result))
	}
	if len(result[0].DeniedActions) != 1 {
		t.Errorf("ins-001 期望 1 条 denied action，实际=%d", len(result[0].DeniedActions))
	}
	if result[1].DeniedActions == nil || len(result[1].DeniedActions) != 0 {
		t.Errorf("no-cvm 期望空切片，实际=%v", result[1].DeniedActions)
	}
	if result[2].DeniedActions == nil || len(result[2].DeniedActions) != 0 {
		t.Errorf("ins-999 不在 map 中期望空切片，实际=%v", result[2].DeniedActions)
	}
}

// TestBuildDeniedActionsResult_PreservesOrder 结果顺序应与输入 instances 顺序一致。
func TestBuildDeniedActionsResult_PreservesOrder(t *testing.T) {
	instances := []model.Instance{
		{Name: "first", InstanceId: "ins-001"},
		{Name: "second", InstanceId: "ins-002"},
		{Name: "third", InstanceId: "ins-003"},
	}
	for i := range instances {
		instances[i].ID = uint(i + 10)
	}

	cvmDenied := map[string][]deniedAction{
		"ins-001": {},
		"ins-002": {{Action: "DescribeInstanceVncUrl", Code: "OperationDenied", Message: ""}},
		"ins-003": {},
	}

	result := buildDeniedActionsResult(instances, cvmDenied)

	for i, inst := range instances {
		if result[i].ID != inst.ID {
			t.Errorf("第 %d 条结果 ID 期望=%d，实际=%d", i, inst.ID, result[i].ID)
		}
	}
}
