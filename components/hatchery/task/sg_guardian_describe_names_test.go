package task

import (
	"context"
	"errors"
	"testing"

	"hatchery/controller"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tcerr "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
)

// describeFn-driven fake: 每次 DescribeSecurityGroups 都走自定义函数，
// 便于按入参 sg_id 返回不同响应/错误。其它接口空实现（满足 SGVpcClient）。
type describeFnVpcClient struct {
	fakeGuardianVpcClient
	fn func(req *vpc.DescribeSecurityGroupsRequest) (*vpc.DescribeSecurityGroupsResponse, error)
}

func (c *describeFnVpcClient) DescribeSecurityGroups(
	req *vpc.DescribeSecurityGroupsRequest,
) (*vpc.DescribeSecurityGroupsResponse, error) {
	return c.fn(req)
}

// makeSGResp 构造单条 SG 响应。
func makeSGResp(items ...[2]string) *vpc.DescribeSecurityGroupsResponse {
	set := make([]*vpc.SecurityGroup, 0, len(items))
	for _, it := range items {
		id, name := it[0], it[1]
		set = append(set, &vpc.SecurityGroup{
			SecurityGroupId:   common.StringPtr(id),
			SecurityGroupName: common.StringPtr(name),
		})
	}
	return &vpc.DescribeSecurityGroupsResponse{
		Response: &vpc.DescribeSecurityGroupsResponseParams{SecurityGroupSet: set},
	}
}

// withGuardianClient 注入 fake client，restore 在 t.Cleanup。
func withGuardianClient(t *testing.T, c controller.SGVpcClient) {
	t.Helper()
	orig := guardianNewVpcClientFn
	guardianNewVpcClientFn = func(ctx context.Context) (controller.SGVpcClient, error) { return c, nil }
	t.Cleanup(func() { guardianNewVpcClientFn = orig })
}

// ----------------------------------------------------------------------------
// describeSGNames：空输入 / 正常批查 / 单批 ResourceNotFound 降级 / 非 NotFound 错
// ----------------------------------------------------------------------------

func TestDescribeSGNames_EmptyInput(t *testing.T) {
	got, err := describeSGNames(context.Background(), nil)
	if err != nil || got != nil {
		t.Errorf("expected (nil, nil), got (%v, %v)", got, err)
	}
}

func TestDescribeSGNames_BatchHappyPath(t *testing.T) {
	fake := &describeFnVpcClient{
		fn: func(req *vpc.DescribeSecurityGroupsRequest) (*vpc.DescribeSecurityGroupsResponse, error) {
			return makeSGResp([2]string{"sg-a", "name-a"}, [2]string{"sg-b", "name-b"}), nil
		},
	}
	withGuardianClient(t, fake)

	got, err := describeSGNames(context.Background(), []string{"sg-a", "sg-b"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got["sg-a"] != "name-a" || got["sg-b"] != "name-b" {
		t.Errorf("unexpected map: %v", got)
	}
}

func TestDescribeSGNames_BatchNotFound_FallsBackPerID(t *testing.T) {
	// 批查整批 ResourceNotFound；单查时 sg-live 存在，sg-dead 仍 NotFound。
	calls := 0
	fake := &describeFnVpcClient{
		fn: func(req *vpc.DescribeSecurityGroupsRequest) (*vpc.DescribeSecurityGroupsResponse, error) {
			calls++
			ids := req.SecurityGroupIds
			// 第一次：批查（含两个 id），返回 ResourceNotFound
			if len(ids) == 2 {
				return nil, &tcerr.TencentCloudSDKError{
					Code:    "ResourceNotFound",
					Message: "指定资源 ['sg-dead'] 未找到",
				}
			}
			// 单查
			id := *ids[0]
			if id == "sg-dead" {
				return nil, &tcerr.TencentCloudSDKError{Code: "ResourceNotFound", Message: "not found"}
			}
			return makeSGResp([2]string{id, "name-" + id}), nil
		},
	}
	withGuardianClient(t, fake)

	got, err := describeSGNames(context.Background(), []string{"sg-live", "sg-dead"})
	if err != nil {
		t.Fatalf("expected nil err on batch fallback, got %v", err)
	}
	if got["sg-live"] != "name-sg-live" {
		t.Errorf("expected sg-live present, got %v", got)
	}
	if _, ok := got["sg-dead"]; ok {
		t.Errorf("sg-dead should NOT be in map (treated as missing), got %v", got)
	}
	if calls != 3 { // 1 批 + 2 单
		t.Errorf("expected 3 client calls (1 batch + 2 per-id), got %d", calls)
	}
}

func TestDescribeSGNames_BatchHardError_Propagates(t *testing.T) {
	fake := &describeFnVpcClient{
		fn: func(req *vpc.DescribeSecurityGroupsRequest) (*vpc.DescribeSecurityGroupsResponse, error) {
			return nil, errors.New("network broken")
		},
	}
	withGuardianClient(t, fake)

	_, err := describeSGNames(context.Background(), []string{"sg-a"})
	if err == nil {
		t.Error("expected error to propagate for non-NotFound failures")
	}
}

func TestDescribeSGNames_NewClientError(t *testing.T) {
	orig := guardianNewVpcClientFn
	guardianNewVpcClientFn = func(ctx context.Context) (controller.SGVpcClient, error) { return nil, errors.New("no creds") }
	t.Cleanup(func() { guardianNewVpcClientFn = orig })

	_, err := describeSGNames(context.Background(), []string{"sg-a"})
	if err == nil {
		t.Error("expected error when vpc client init fails")
	}
}

func TestDescribeSGNames_NilResponseSkipped(t *testing.T) {
	fake := &describeFnVpcClient{
		fn: func(req *vpc.DescribeSecurityGroupsRequest) (*vpc.DescribeSecurityGroupsResponse, error) {
			return &vpc.DescribeSecurityGroupsResponse{Response: nil}, nil
		},
	}
	withGuardianClient(t, fake)

	got, err := describeSGNames(context.Background(), []string{"sg-a"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}

func TestDescribeSGNames_SkipsNilFields(t *testing.T) {
	fake := &describeFnVpcClient{
		fn: func(req *vpc.DescribeSecurityGroupsRequest) (*vpc.DescribeSecurityGroupsResponse, error) {
			return &vpc.DescribeSecurityGroupsResponse{
				Response: &vpc.DescribeSecurityGroupsResponseParams{
					SecurityGroupSet: []*vpc.SecurityGroup{
						{SecurityGroupId: nil, SecurityGroupName: common.StringPtr("x")},    // skip
						{SecurityGroupId: common.StringPtr("sg-a"), SecurityGroupName: nil}, // skip
						{SecurityGroupId: common.StringPtr("sg-b"), SecurityGroupName: common.StringPtr("name-b")},
					},
				},
			}, nil
		},
	}
	withGuardianClient(t, fake)

	got, err := describeSGNames(context.Background(), []string{"sg-a", "sg-b"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got) != 1 || got["sg-b"] != "name-b" {
		t.Errorf("expected only sg-b, got %v", got)
	}
}

// ----------------------------------------------------------------------------
// describeSGNamesPerID：单个非 NotFound 错误只 warn 不中断；nil response 跳过；nil 字段跳过
// ----------------------------------------------------------------------------

func TestDescribeSGNamesPerID_TransientErrorSkippedNotAborted(t *testing.T) {
	calls := 0
	fake := &describeFnVpcClient{
		fn: func(req *vpc.DescribeSecurityGroupsRequest) (*vpc.DescribeSecurityGroupsResponse, error) {
			calls++
			id := *req.SecurityGroupIds[0]
			if id == "sg-flaky" {
				return nil, errors.New("transient timeout")
			}
			if id == "sg-nilresp" {
				return &vpc.DescribeSecurityGroupsResponse{Response: nil}, nil
			}
			if id == "sg-nilfield" {
				return &vpc.DescribeSecurityGroupsResponse{
					Response: &vpc.DescribeSecurityGroupsResponseParams{
						SecurityGroupSet: []*vpc.SecurityGroup{{SecurityGroupId: nil, SecurityGroupName: nil}},
					},
				}, nil
			}
			return makeSGResp([2]string{id, "name-" + id}), nil
		},
	}

	out := map[string]string{}
	describeSGNamesPerID(fake, []string{"sg-flaky", "sg-nilresp", "sg-nilfield", "sg-ok"}, out)

	if calls != 4 {
		t.Errorf("expected 4 per-id calls regardless of failures, got %d", calls)
	}
	if out["sg-ok"] != "name-sg-ok" {
		t.Errorf("expected sg-ok recovered, got %v", out)
	}
	if len(out) != 1 {
		t.Errorf("only sg-ok should be present, got %v", out)
	}
}

// ----------------------------------------------------------------------------
// isSGNotFoundErr：各种错误码 / 文本场景
// ----------------------------------------------------------------------------

func TestIsSGNotFoundErr(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"sdk ResourceNotFound", &tcerr.TencentCloudSDKError{Code: "ResourceNotFound"}, true},
		{"sdk ResourceNotFound.Sub", &tcerr.TencentCloudSDKError{Code: "ResourceNotFound.SecurityGroup"}, true},
		{"sdk InvalidParameterValue", &tcerr.TencentCloudSDKError{Code: "InvalidParameterValue"}, true},
		{"sdk InvalidSecurityGroupId.NotFound", &tcerr.TencentCloudSDKError{Code: "InvalidSecurityGroupId.NotFound"}, true},
		{"sdk other code", &tcerr.TencentCloudSDKError{Code: "InternalError"}, false},
		{"plain not found", errors.New("resource Not Found"), true},
		{"plain 不存在", errors.New("某 SG 不存在"), true},
		{"plain 未找到", errors.New("未找到该资源"), true},
		{"plain unrelated", errors.New("network error"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isSGNotFoundErr(tc.err); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
