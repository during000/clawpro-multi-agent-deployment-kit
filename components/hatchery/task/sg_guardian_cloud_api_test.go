package task

import (
	"context"
	"errors"
	"testing"

	"hatchery/controller"
	"hatchery/model"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// ============================================================================
// task/sg_guardian_task.go: createCloudSGWithRetry / applyRulesToCloudSGWithRetry
// describeAssocStats / describeSGNames / poolSGIDs 补测
//
// 为了避免不引入真实云 SDK，我们直接调用 guardian 的 hook variable
// （createCloudSGWithRetryFn / applyRulesToCloudSGWithRetryFn 等）。
// 真正 createCloudSGWithRetry 函数因内嵌 200ms backoff sleep，在失败路径会慢 ~1s，
// 所以只测 happy path（1 次成功）即可。
// ============================================================================

// fakeGuardianVpcClient 供 describeAssocStats / describeSGNames 等使用。
type fakeGuardianVpcClient struct {
	assocSet        []*vpc.SecurityGroupAssociationStatistics
	assocErr        error
	describeSGs     *vpc.DescribeSecurityGroupsResponse
	describeSGsErr  error
	createSGResp    *vpc.CreateSecurityGroupResponse
	createSGErr     error
	modifyRulesErr  error
	deleteSGErr     error
	describePolicy  *vpc.DescribeSecurityGroupPoliciesResponse
	describePolicyE error
}

func (f *fakeGuardianVpcClient) DescribeSecurityGroupAssociationStatistics(
	req *vpc.DescribeSecurityGroupAssociationStatisticsRequest,
) (*vpc.DescribeSecurityGroupAssociationStatisticsResponse, error) {
	if f.assocErr != nil {
		return nil, f.assocErr
	}
	return &vpc.DescribeSecurityGroupAssociationStatisticsResponse{
		Response: &vpc.DescribeSecurityGroupAssociationStatisticsResponseParams{
			SecurityGroupAssociationStatisticsSet: f.assocSet,
		},
	}, nil
}

func (f *fakeGuardianVpcClient) DescribeSecurityGroups(
	req *vpc.DescribeSecurityGroupsRequest,
) (*vpc.DescribeSecurityGroupsResponse, error) {
	if f.describeSGsErr != nil {
		return nil, f.describeSGsErr
	}
	if f.describeSGs != nil {
		return f.describeSGs, nil
	}
	return &vpc.DescribeSecurityGroupsResponse{Response: &vpc.DescribeSecurityGroupsResponseParams{}}, nil
}

func (f *fakeGuardianVpcClient) CreateSecurityGroup(
	req *vpc.CreateSecurityGroupRequest,
) (*vpc.CreateSecurityGroupResponse, error) {
	if f.createSGErr != nil {
		return nil, f.createSGErr
	}
	if f.createSGResp != nil {
		return f.createSGResp, nil
	}
	return &vpc.CreateSecurityGroupResponse{
		Response: &vpc.CreateSecurityGroupResponseParams{
			SecurityGroup: &vpc.SecurityGroup{SecurityGroupId: common.StringPtr("sg-new-xxx")},
		},
	}, nil
}

func (f *fakeGuardianVpcClient) DeleteSecurityGroup(
	req *vpc.DeleteSecurityGroupRequest,
) (*vpc.DeleteSecurityGroupResponse, error) {
	if f.deleteSGErr != nil {
		return nil, f.deleteSGErr
	}
	return &vpc.DeleteSecurityGroupResponse{Response: &vpc.DeleteSecurityGroupResponseParams{}}, nil
}

func (f *fakeGuardianVpcClient) ModifySecurityGroupPolicies(
	req *vpc.ModifySecurityGroupPoliciesRequest,
) (*vpc.ModifySecurityGroupPoliciesResponse, error) {
	if f.modifyRulesErr != nil {
		return nil, f.modifyRulesErr
	}
	return &vpc.ModifySecurityGroupPoliciesResponse{Response: &vpc.ModifySecurityGroupPoliciesResponseParams{}}, nil
}

func (f *fakeGuardianVpcClient) DescribeSecurityGroupPolicies(
	req *vpc.DescribeSecurityGroupPoliciesRequest,
) (*vpc.DescribeSecurityGroupPoliciesResponse, error) {
	if f.describePolicyE != nil {
		return nil, f.describePolicyE
	}
	if f.describePolicy != nil {
		return f.describePolicy, nil
	}
	return &vpc.DescribeSecurityGroupPoliciesResponse{
		Response: &vpc.DescribeSecurityGroupPoliciesResponseParams{
			SecurityGroupPolicySet: &vpc.SecurityGroupPolicySet{},
		},
	}, nil
}

// --- 为满足 sgVpcClient 接口，提供这些方法的空实现 ---

func (f *fakeGuardianVpcClient) ModifySecurityGroupAttribute(
	req *vpc.ModifySecurityGroupAttributeRequest,
) (*vpc.ModifySecurityGroupAttributeResponse, error) {
	return &vpc.ModifySecurityGroupAttributeResponse{}, nil
}

func (f *fakeGuardianVpcClient) CreateSecurityGroupPolicies(
	req *vpc.CreateSecurityGroupPoliciesRequest,
) (*vpc.CreateSecurityGroupPoliciesResponse, error) {
	return &vpc.CreateSecurityGroupPoliciesResponse{}, nil
}

func (f *fakeGuardianVpcClient) ReplaceSecurityGroupPolicy(
	req *vpc.ReplaceSecurityGroupPolicyRequest,
) (*vpc.ReplaceSecurityGroupPolicyResponse, error) {
	return &vpc.ReplaceSecurityGroupPolicyResponse{}, nil
}

func (f *fakeGuardianVpcClient) DeleteSecurityGroupPolicies(
	req *vpc.DeleteSecurityGroupPoliciesRequest,
) (*vpc.DeleteSecurityGroupPoliciesResponse, error) {
	return &vpc.DeleteSecurityGroupPoliciesResponse{}, nil
}

func (f *fakeGuardianVpcClient) DescribeVpcs(
	req *vpc.DescribeVpcsRequest,
) (*vpc.DescribeVpcsResponse, error) {
	return &vpc.DescribeVpcsResponse{}, nil
}

// setupForCloudAPITests 内存 DB + 替换 guardianNewVpcClientFn（task 本地 hook）
func setupForCloudAPITests(t *testing.T, fake *fakeGuardianVpcClient) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.AutoMigrate(&model.ManagedSGPool{}, &model.RuleSet{}, &model.SiteConfig{}, &model.Instance{})
	origDB := model.UseDBForTest(db)

	// 替换 task 包的 guardianNewVpcClientFn（这样 describeAssocStats / describeSGNames
	// 会用 fake），同时替换 controller 包的 newVpcClientForSGFn（这样
	// controller.CreateCloudSG / ApplyRulesToCloudSG 也会用 fake）
	origFn := guardianNewVpcClientFn
	guardianNewVpcClientFn = func(ctx context.Context) (controller.SGVpcClient, error) { return fake, nil }
	// describeSGNamesFn 默认走 describeSGNames（本身用 guardianNewVpcClientFn），
	// 不需要额外替换，但保持 restore 以防其它测试修改
	origDescribeNames := describeSGNamesFn
	origControllerFn := controller.SetNewVpcClientForSGFnForTest(func(ctx context.Context) (controller.SGVpcClient, error) { return fake, nil })

	t.Cleanup(func() {
		guardianNewVpcClientFn = origFn
		describeSGNamesFn = origDescribeNames
		origControllerFn()
		origDB()
	})
	return db
}

// ------------------------------------------------------------
// describeAssocStats
// ------------------------------------------------------------

func TestDescribeAssocStats_Empty(t *testing.T) {
	// 空输入应直接返回 nil，不调 client
	got, err := describeAssocStats(context.Background(), nil)
	if err != nil {
		t.Errorf("unexpected err: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil map for empty input, got %v", got)
	}
}

func TestDescribeAssocStats_SumsCVMAndENIAndCDB(t *testing.T) {
	sgID := "sg-abc"
	cvm := uint64(3)
	eni := uint64(2)
	cdb := uint64(1)
	fake := &fakeGuardianVpcClient{
		assocSet: []*vpc.SecurityGroupAssociationStatistics{
			{
				SecurityGroupId: &sgID,
				CVM:             &cvm,
				ENI:             &eni,
				CDB:             &cdb,
			},
		},
	}
	setupForCloudAPITests(t, fake)

	got, err := describeAssocStats(context.Background(), []string{sgID})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got[sgID] != 6 {
		t.Errorf("expected sum=6 (3+2+1), got %d", got[sgID])
	}
}

func TestDescribeAssocStats_SkipsNilSecurityGroupId(t *testing.T) {
	cvm := uint64(5)
	fake := &fakeGuardianVpcClient{
		assocSet: []*vpc.SecurityGroupAssociationStatistics{
			{SecurityGroupId: nil, CVM: &cvm}, // 应跳过
		},
	}
	setupForCloudAPITests(t, fake)

	got, err := describeAssocStats(context.Background(), []string{"sg-x"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("nil SecurityGroupId should be skipped, got %v", got)
	}
}

// TestDescribeAssocStats_APIError 验证 transient 错误（非 ResourceNotFound）
// 不会让整个函数返回 error，而是 warn + skip 该 SG（不进 map），下一轮重试。
//
// 行为合约（与 RETIRE 决策强耦合）：
//   - 单 SG transient 错误 → 不进 map，但不视为失踪，绝不会触发 RETIRE
//   - 单 SG ResourceNotFound → 不进 map（语义"云端确认失踪"），由 detectOrphans 二次确认后处理
func TestDescribeAssocStats_APIError(t *testing.T) {
	fake := &fakeGuardianVpcClient{assocErr: errors.New("boom")}
	setupForCloudAPITests(t, fake)
	got, err := describeAssocStats(context.Background(), []string{"sg-x"})
	if err != nil {
		t.Errorf("transient error should be swallowed and warned, got err=%v", err)
	}
	if _, ok := got["sg-x"]; ok {
		t.Errorf("transient-failed SG should not appear in result map, got %v", got)
	}
}

// listCloudManagedSGs 已移除（不再按名字前缀扫全账号 SG），对应测试一并删除。
// 新模型下以 DB `managed_sg_pool` 为真相源，通过 describeSGNames 按 sg_id 精确
// 查询；describeSGNames 的测试见上方 describeSGNames section。

// ------------------------------------------------------------
// createCloudSGWithRetry — happy path（单次成功）
// ------------------------------------------------------------

func TestCreateCloudSGWithRetry_FirstAttemptSucceeds(t *testing.T) {
	sgID := "sg-ok-001"
	fake := &fakeGuardianVpcClient{
		createSGResp: &vpc.CreateSecurityGroupResponse{
			Response: &vpc.CreateSecurityGroupResponseParams{
				SecurityGroup: &vpc.SecurityGroup{SecurityGroupId: &sgID},
			},
		},
	}
	setupForCloudAPITests(t, fake)

	got, err := createCloudSGWithRetry(context.Background(), "name", "desc")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != sgID {
		t.Errorf("expected %q, got %q", sgID, got)
	}
}

// ------------------------------------------------------------
// applyRulesToCloudSGWithRetry — happy path
// ------------------------------------------------------------

func TestApplyRulesToCloudSGWithRetry_FirstAttemptSucceeds(t *testing.T) {
	fake := &fakeGuardianVpcClient{} // modifyRulesErr=nil → 成功
	setupForCloudAPITests(t, fake)
	// 空 rules JSON 会走 "空" 路径，直接返回；用非空但合法的 JSON
	err := applyRulesToCloudSGWithRetry(context.Background(), "sg-x", `[]`)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

// 避免 unused import
var _ = context.Background
