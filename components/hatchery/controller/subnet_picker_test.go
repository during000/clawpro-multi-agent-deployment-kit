package controller

import (
	"errors"
	"strings"
	"testing"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
)

// fakeSubnetDescriber 是 subnetDescriber 的测试替身。
type fakeSubnetDescriber struct {
	resp *vpc.DescribeSubnetsResponse
	err  error
	// 记录被请求的子网列表，便于断言
	requestedSubnetIds []string
}

func (f *fakeSubnetDescriber) DescribeSubnets(req *vpc.DescribeSubnetsRequest) (*vpc.DescribeSubnetsResponse, error) {
	if req != nil {
		for _, p := range req.SubnetIds {
			if p != nil {
				f.requestedSubnetIds = append(f.requestedSubnetIds, *p)
			}
		}
	}
	return f.resp, f.err
}

// newSubnetsResp 辅助构造 DescribeSubnets 响应。
func newSubnetsResp(subnets map[string]uint64) *vpc.DescribeSubnetsResponse {
	resp := vpc.NewDescribeSubnetsResponse()
	set := make([]*vpc.Subnet, 0, len(subnets))
	for id, avail := range subnets {
		id := id
		avail := avail
		set = append(set, &vpc.Subnet{
			SubnetId:                common.StringPtr(id),
			AvailableIpAddressCount: common.Uint64Ptr(avail),
		})
	}
	resp.Response = &vpc.DescribeSubnetsResponseParams{SubnetSet: set}
	return resp
}

func TestPickSubnetByAvailableIP_EmptyCandidates(t *testing.T) {
	fake := &fakeSubnetDescriber{}
	_, err := pickSubnetByAvailableIP(fake, nil)
	if err == nil {
		t.Fatal("want error for empty candidates")
	}
	if !strings.Contains(err.Error(), "候选子网列表为空") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPickSubnetByAvailableIP_DescribeError(t *testing.T) {
	fake := &fakeSubnetDescriber{
		err: errors.New("api quota exceeded"),
	}
	_, err := pickSubnetByAvailableIP(fake, []string{"subnet-a"})
	if err == nil {
		t.Fatal("want error when describe fails")
	}
	if !strings.Contains(err.Error(), "查询子网可用 IP 数失败") {
		t.Errorf("unexpected error wrapping: %v", err)
	}
}

func TestPickSubnetByAvailableIP_SingleCandidate(t *testing.T) {
	fake := &fakeSubnetDescriber{
		resp: newSubnetsResp(map[string]uint64{"subnet-a": 100}),
	}
	got, err := pickSubnetByAvailableIP(fake, []string{"subnet-a"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "subnet-a" {
		t.Errorf("got %q, want subnet-a", got)
	}
	if len(fake.requestedSubnetIds) != 1 || fake.requestedSubnetIds[0] != "subnet-a" {
		t.Errorf("requested subnet mismatch: %v", fake.requestedSubnetIds)
	}
}

func TestPickSubnetByAvailableIP_AllFull(t *testing.T) {
	fake := &fakeSubnetDescriber{
		resp: newSubnetsResp(map[string]uint64{
			"subnet-a": 0,
			"subnet-b": 0,
		}),
	}
	_, err := pickSubnetByAvailableIP(fake, []string{"subnet-a", "subnet-b"})
	if err == nil {
		t.Fatal("want error when all subnets are full")
	}
	if !strings.Contains(err.Error(), "IP 已满") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPickSubnetByAvailableIP_SkipsFullSubnet(t *testing.T) {
	fake := &fakeSubnetDescriber{
		resp: newSubnetsResp(map[string]uint64{
			"subnet-full":   0,
			"subnet-plenty": 100,
		}),
	}
	// 跑 20 次确认永远不会返回满的那个
	for i := 0; i < 20; i++ {
		got, err := pickSubnetByAvailableIP(fake, []string{"subnet-full", "subnet-plenty"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "subnet-plenty" {
			t.Errorf("iteration %d: got %q, want subnet-plenty (subnet-full should be skipped)", i, got)
		}
	}
}

func TestPickSubnetByAvailableIP_WeightedPickingFavorsLarger(t *testing.T) {
	// 1 vs 999 的极端比例：100 次采样里 subnet-large 应该远超半数
	fake := &fakeSubnetDescriber{
		resp: newSubnetsResp(map[string]uint64{
			"subnet-small": 1,
			"subnet-large": 999,
		}),
	}
	countLarge := 0
	for i := 0; i < 100; i++ {
		got, err := pickSubnetByAvailableIP(fake, []string{"subnet-small", "subnet-large"})
		if err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", i, err)
		}
		if got == "subnet-large" {
			countLarge++
		}
	}
	// 1 vs 999 的加权下，100 次中 subnet-large 应占 >= 90%
	if countLarge < 90 {
		t.Errorf("expected subnet-large to dominate (>=90/100), got %d/100", countLarge)
	}
}

func TestPickSubnetByAvailableIP_SkipsNilFields(t *testing.T) {
	// 响应里混入了 SubnetId==nil 或 AvailableIpAddressCount==nil 的条目，
	// 应当被跳过（防御性处理 SDK 返回异常）。
	resp := vpc.NewDescribeSubnetsResponse()
	resp.Response = &vpc.DescribeSubnetsResponseParams{
		SubnetSet: []*vpc.Subnet{
			{SubnetId: nil, AvailableIpAddressCount: common.Uint64Ptr(100)},
			{SubnetId: common.StringPtr("subnet-x"), AvailableIpAddressCount: nil},
			{SubnetId: common.StringPtr("subnet-valid"), AvailableIpAddressCount: common.Uint64Ptr(50)},
		},
	}
	fake := &fakeSubnetDescriber{resp: resp}
	got, err := pickSubnetByAvailableIP(fake, []string{"subnet-x", "subnet-valid"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "subnet-valid" {
		t.Errorf("got %q, want subnet-valid (others should be filtered)", got)
	}
}

func TestPickSubnetByAvailableIP_NilResponse(t *testing.T) {
	// resp.Response == nil 的边界情况
	resp := vpc.NewDescribeSubnetsResponse()
	resp.Response = nil
	fake := &fakeSubnetDescriber{resp: resp}
	_, err := pickSubnetByAvailableIP(fake, []string{"subnet-a"})
	if err == nil {
		t.Fatal("want error when response body is nil / no valid subnets")
	}
}

// ──────── validateSubnetMapOnCloud ────────

// newSubnetsRespWithVpcAndZone 构造带 VpcId 和 Zone 的 DescribeSubnets 响应。
func newSubnetsRespWithVpcAndZone(subnets []struct {
	Id, Vpc, Zone string
}) *vpc.DescribeSubnetsResponse {
	resp := vpc.NewDescribeSubnetsResponse()
	set := make([]*vpc.Subnet, 0, len(subnets))
	for _, s := range subnets {
		s := s
		set = append(set, &vpc.Subnet{
			SubnetId: common.StringPtr(s.Id),
			VpcId:    common.StringPtr(s.Vpc),
			Zone:     common.StringPtr(s.Zone),
		})
	}
	resp.Response = &vpc.DescribeSubnetsResponseParams{SubnetSet: set}
	return resp
}

func TestValidateSubnetMapOnCloud_EmptyMap(t *testing.T) {
	fake := &fakeSubnetDescriber{}
	err := validateSubnetMapOnCloud(fake, "vpc-x", map[string][]string{})
	if err == nil {
		t.Fatal("want error for empty map")
	}
	if !strings.Contains(err.Error(), "子网不能为空") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateSubnetMapOnCloud_OnlyEmptyStrings(t *testing.T) {
	// 所有 subnet id 都是空串 → 视同空
	fake := &fakeSubnetDescriber{}
	err := validateSubnetMapOnCloud(fake, "vpc-x", map[string][]string{
		"ap-gz-6": {""},
	})
	if err == nil {
		t.Fatal("want error when subnet ids are all empty")
	}
}

func TestValidateSubnetMapOnCloud_DescribeError(t *testing.T) {
	fake := &fakeSubnetDescriber{
		err: errors.New("quota exceeded"),
	}
	err := validateSubnetMapOnCloud(fake, "vpc-x", map[string][]string{
		"ap-gz-6": {"subnet-a"},
	})
	if err == nil {
		t.Fatal("want error when describe fails")
	}
	// 云 API 故障应该是 *subnetValidateCloudError
	var cloudErr *subnetValidateCloudError
	if !errors.As(err, &cloudErr) {
		t.Errorf("expected *subnetValidateCloudError, got %T", err)
	}
}

func TestValidateSubnetMapOnCloud_VpcMismatch(t *testing.T) {
	// 云端返回的子网属于别的 VPC → 报"vpc和子网不匹配"
	fake := &fakeSubnetDescriber{
		resp: newSubnetsRespWithVpcAndZone([]struct{ Id, Vpc, Zone string }{
			{"subnet-a", "vpc-other", "ap-gz-6"},
		}),
	}
	err := validateSubnetMapOnCloud(fake, "vpc-x", map[string][]string{
		"ap-gz-6": {"subnet-a"},
	})
	if err == nil {
		t.Fatal("want error when subnet in different VPC")
	}
	if !strings.Contains(err.Error(), "vpc和子网不匹配") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateSubnetMapOnCloud_SubnetNotFound(t *testing.T) {
	// 云端没返回 subnet-b → 报不存在
	fake := &fakeSubnetDescriber{
		resp: newSubnetsRespWithVpcAndZone([]struct{ Id, Vpc, Zone string }{
			{"subnet-a", "vpc-x", "ap-gz-6"},
		}),
	}
	err := validateSubnetMapOnCloud(fake, "vpc-x", map[string][]string{
		"ap-gz-6": {"subnet-a", "subnet-b"},
	})
	if err == nil {
		t.Fatal("want error for missing subnet")
	}
	if !strings.Contains(err.Error(), "subnet-b") {
		t.Errorf("error should mention missing subnet: %v", err)
	}
}

func TestValidateSubnetMapOnCloud_ZoneMismatch(t *testing.T) {
	// 云端返回的子网 zone 与 subnetMap key 不一致
	fake := &fakeSubnetDescriber{
		resp: newSubnetsRespWithVpcAndZone([]struct{ Id, Vpc, Zone string }{
			{"subnet-a", "vpc-x", "ap-gz-7"}, // 实际在 gz-7
		}),
	}
	err := validateSubnetMapOnCloud(fake, "vpc-x", map[string][]string{
		"ap-gz-6": {"subnet-a"}, // 却配在 gz-6
	})
	if err == nil {
		t.Fatal("want error when zone mismatches")
	}
	if !strings.Contains(err.Error(), "可用区") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateSubnetMapOnCloud_AllValid(t *testing.T) {
	fake := &fakeSubnetDescriber{
		resp: newSubnetsRespWithVpcAndZone([]struct{ Id, Vpc, Zone string }{
			{"subnet-a", "vpc-x", "ap-gz-6"},
			{"subnet-b", "vpc-x", "ap-gz-6"},
			{"subnet-c", "vpc-x", "ap-gz-7"},
		}),
	}
	err := validateSubnetMapOnCloud(fake, "vpc-x", map[string][]string{
		"ap-gz-6": {"subnet-a", "subnet-b"},
		"ap-gz-7": {"subnet-c"},
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateSubnetMapOnCloud_EmptyZoneSlice(t *testing.T) {
	// 某 zone 的 slice 为空 → 报错
	fake := &fakeSubnetDescriber{
		resp: newSubnetsRespWithVpcAndZone([]struct{ Id, Vpc, Zone string }{
			{"subnet-a", "vpc-x", "ap-gz-6"},
		}),
	}
	err := validateSubnetMapOnCloud(fake, "vpc-x", map[string][]string{
		"ap-gz-6": {"subnet-a"},
		"ap-gz-7": {}, // 空 slice
	})
	if err == nil {
		t.Fatal("want error when a zone has empty subnet slice")
	}
	if !strings.Contains(err.Error(), "ap-gz-7") {
		t.Errorf("error should mention empty zone: %v", err)
	}
}
