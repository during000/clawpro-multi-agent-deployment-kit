package controller

import (
	"context"
	"errors"
	hcommon "hatchery/common"
	"strings"
	"testing"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
)

// fakeVpcValidator 是 vpcValidatorClient 的测试替身。
type fakeVpcValidator struct {
	vpcResp *vpc.DescribeVpcsResponse
	vpcErr  error

	subnetResp *vpc.DescribeSubnetsResponse
	subnetErr  error
}

func (f *fakeVpcValidator) DescribeVpcs(req *vpc.DescribeVpcsRequest) (*vpc.DescribeVpcsResponse, error) {
	return f.vpcResp, f.vpcErr
}

func (f *fakeVpcValidator) DescribeSubnets(req *vpc.DescribeSubnetsRequest) (*vpc.DescribeSubnetsResponse, error) {
	return f.subnetResp, f.subnetErr
}

// newVpcResp 构造 DescribeVpcs 响应，vpcIds 非空 → 返回对应 VpcSet；空 → Response 里 VpcSet 空。
func newVpcResp(vpcIds ...string) *vpc.DescribeVpcsResponse {
	resp := vpc.NewDescribeVpcsResponse()
	set := make([]*vpc.Vpc, 0, len(vpcIds))
	for _, id := range vpcIds {
		id := id
		set = append(set, &vpc.Vpc{VpcId: common.StringPtr(id)})
	}
	resp.Response = &vpc.DescribeVpcsResponseParams{VpcSet: set}
	return resp
}

// newSubnetsRespSimple 构造 DescribeSubnets 响应，仅关心 SubnetId 存在性。
func newSubnetsRespSimple(subnetIds ...string) *vpc.DescribeSubnetsResponse {
	resp := vpc.NewDescribeSubnetsResponse()
	set := make([]*vpc.Subnet, 0, len(subnetIds))
	for _, id := range subnetIds {
		id := id
		set = append(set, &vpc.Subnet{SubnetId: common.StringPtr(id)})
	}
	resp.Response = &vpc.DescribeSubnetsResponseParams{SubnetSet: set}
	return resp
}

func TestValidateGlobalVpcAndSubnets_VpcNotExist(t *testing.T) {
	fake := &fakeVpcValidator{
		vpcResp: newVpcResp(), // 空 VpcSet → vpc 不存在
	}
	err := validateGlobalVpcAndSubnetsCore(fake, "vpc-missing",
		map[string][]string{"ap-gz-6": {"subnet-a"}})
	if err == nil {
		t.Fatal("want error when VPC missing")
	}
	if !strings.Contains(hcommon.ErrorMessageWithCtx(context.Background(), err), "不存在") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateGlobalVpcAndSubnets_DescribeVpcError(t *testing.T) {
	fake := &fakeVpcValidator{
		vpcErr: errors.New("auth failed"),
	}
	err := validateGlobalVpcAndSubnetsCore(fake, "vpc-x", map[string][]string{"z": {"s"}})
	if err == nil {
		t.Fatal("want error when DescribeVpcs fails")
	}
	if !strings.Contains(hcommon.ErrorMessageWithCtx(context.Background(), err), "查询全局 VPC 失败") {
		t.Errorf("unexpected error wrapping: %v", err)
	}
}

func TestValidateGlobalVpcAndSubnets_AllSubnetsExist(t *testing.T) {
	fake := &fakeVpcValidator{
		vpcResp:    newVpcResp("vpc-x"),
		subnetResp: newSubnetsRespSimple("subnet-a", "subnet-b", "subnet-c"),
	}
	err := validateGlobalVpcAndSubnetsCore(fake, "vpc-x",
		map[string][]string{
			"ap-gz-6": {"subnet-a", "subnet-b"},
			"ap-gz-7": {"subnet-c"},
		})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateGlobalVpcAndSubnets_MissingSubnet(t *testing.T) {
	fake := &fakeVpcValidator{
		vpcResp:    newVpcResp("vpc-x"),
		subnetResp: newSubnetsRespSimple("subnet-a"), // subnet-b 不在云端
	}
	err := validateGlobalVpcAndSubnetsCore(fake, "vpc-x",
		map[string][]string{"ap-gz-6": {"subnet-a", "subnet-b"}})
	if err == nil {
		t.Fatal("want error when some subnet missing")
	}
	if !strings.Contains(hcommon.ErrorMessageWithCtx(context.Background(), err), "subnet-b") {
		t.Errorf("error should mention missing subnet: %v", err)
	}
}

func TestValidateGlobalVpcAndSubnets_EmptySubnetMap(t *testing.T) {
	// VPC 存在、subnetMap 为空（或所有 zone 的 slice 都空）→ 不查询 DescribeSubnets，直接通过
	fake := &fakeVpcValidator{
		vpcResp: newVpcResp("vpc-x"),
		// 不设 subnetResp，若被调用会返回 nil panic
	}
	err := validateGlobalVpcAndSubnetsCore(fake, "vpc-x", map[string][]string{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// 所有 slice 空
	err = validateGlobalVpcAndSubnetsCore(fake, "vpc-x",
		map[string][]string{"ap-gz-6": {}})
	if err != nil {
		t.Errorf("unexpected error (all empty slices): %v", err)
	}
}

func TestValidateGlobalVpcAndSubnets_DedupesSubnetIds(t *testing.T) {
	// 不同 zone 里重复的 subnet id（配置错误），应去重后只查询一次
	fake := &fakeVpcValidator{
		vpcResp:    newVpcResp("vpc-x"),
		subnetResp: newSubnetsRespSimple("subnet-a"),
	}
	err := validateGlobalVpcAndSubnetsCore(fake, "vpc-x",
		map[string][]string{
			"ap-gz-6": {"subnet-a", "subnet-a"},
			"ap-gz-7": {"subnet-a"},
		})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateGlobalVpcAndSubnets_DescribeSubnetsError(t *testing.T) {
	fake := &fakeVpcValidator{
		vpcResp:   newVpcResp("vpc-x"),
		subnetErr: errors.New("quota exceeded"),
	}
	err := validateGlobalVpcAndSubnetsCore(fake, "vpc-x",
		map[string][]string{"z": {"s"}})
	if err == nil {
		t.Fatal("want error when DescribeSubnets fails")
	}
	if !strings.Contains(hcommon.ErrorMessageWithCtx(context.Background(), err), "查询全局子网失败") {
		t.Errorf("unexpected error: %v", err)
	}
}
