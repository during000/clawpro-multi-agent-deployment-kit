package controller

import (
	hcommon "hatchery/common"
	"hatchery/i18n"
	"math/rand"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
)

// subnetDescriber 抽象 DescribeSubnets 能力，便于测试时替换。
// *vpc.Client 天然实现该接口。
type subnetDescriber interface {
	DescribeSubnets(req *vpc.DescribeSubnetsRequest) (*vpc.DescribeSubnetsResponse, error)
}

// subnetValidateCloudError 表示 validateSubnetMapOnCloud 遇到腾讯云 API 故障，
// 调用方应返回 5xx 而非 4xx（用户请求本身是合法的，是后端不可用）。
type subnetValidateCloudError struct{ err error }

func (e *subnetValidateCloudError) Error() string { return e.err.Error() }
func (e *subnetValidateCloudError) Unwrap() error { return e.err }

// validateSubnetMapOnCloud 向腾讯云校验 subnetMap 里所有子网：
//  1. 子网属于指定 VPC
//  2. 子网可用区与 subnetMap key 声明的可用区一致
//  3. 每个可用区至少配了一个有效子网
//
// 返回 error 的类型：
//   - *subnetValidateCloudError 在 cause 链中: 云 API 故障，上层应返回 500
//   - 其它 error: 用户配置问题，上层应返回 400
func validateSubnetMapOnCloud(vpcClient subnetDescriber, vpcId string, subnetMap map[string][]string) error {
	// 去重收集所有子网 ID，批量查询一次
	seen := make(map[string]bool)
	subnetIdList := make([]string, 0)
	for _, sids := range subnetMap {
		for _, sid := range sids {
			if sid == "" || seen[sid] {
				continue
			}
			seen[sid] = true
			subnetIdList = append(subnetIdList, sid)
		}
	}
	if len(subnetIdList) == 0 {
		return hcommon.I18nError(i18n.MsgSubnetCannotBeEmpty)
	}

	descReq := vpc.NewDescribeSubnetsRequest()
	descReq.SubnetIds = common.StringPtrs(subnetIdList)
	descResp, err := vpcClient.DescribeSubnets(descReq)
	if err != nil {
		return hcommon.I18nRichError(&subnetValidateCloudError{err: err}, i18n.MsgQuerySubnetFailed)
	}

	foundSubnets := make(map[string]*vpc.Subnet)
	if descResp.Response != nil {
		for _, s := range descResp.Response.SubnetSet {
			if s.VpcId == nil || *s.VpcId != vpcId {
				return hcommon.I18nError(i18n.MsgVpcSubnetMismatch)
			}
			if s.SubnetId != nil {
				foundSubnets[*s.SubnetId] = s
			}
		}
	}

	for zone, sids := range subnetMap {
		if len(sids) == 0 {
			return hcommon.I18nError(i18n.MsgZoneNoSubnetConfigured, zone)
		}
		for _, sid := range sids {
			subnet, ok := foundSubnets[sid]
			if !ok {
				return hcommon.I18nError(i18n.MsgVpcOrSubnetNotExist, zone, sid)
			}
			if subnet.Zone != nil && *subnet.Zone != zone {
				return hcommon.I18nError(i18n.MsgSubnetZoneMismatch, sid, *subnet.Zone, zone)
			}
		}
	}
	return nil
}

// pickSubnetByAvailableIP 按 AvailableIpAddressCount 加权随机从候选子网中挑选一个。
// 会自动跳过 AvailableIpAddressCount == 0 的子网。
// candidates 为同一可用区下的候选子网 ID 列表。
// 返回被选中的 subnetId；若所有候选子网都已满（可用 IP 为 0），返回明确错误。
//
// 数据源：腾讯云 VPC DescribeSubnets，单次批量查询所有候选子网；
// 不做缓存（创建实例非 QPS 敏感路径）。
func pickSubnetByAvailableIP(vpcClient subnetDescriber, candidates []string) (string, error) {
	if len(candidates) == 0 {
		return "", hcommon.I18nError(i18n.MsgCandidateSubnetEmpty)
	}
	// 单个子网时也批量查询一次 AvailableIpAddressCount，避免撞满子网直接失败
	req := vpc.NewDescribeSubnetsRequest()
	req.SubnetIds = common.StringPtrs(candidates)
	resp, err := vpcClient.DescribeSubnets(req)
	if err != nil {
		return "", hcommon.I18nRichError(err, i18n.MsgQuerySubnetIPFailed)
	}

	type entry struct {
		id        string
		available uint64
	}
	var entries []entry
	if resp.Response != nil {
		for _, s := range resp.Response.SubnetSet {
			if s.SubnetId == nil || s.AvailableIpAddressCount == nil {
				continue
			}
			avail := *s.AvailableIpAddressCount
			if avail == 0 {
				continue
			}
			entries = append(entries, entry{id: *s.SubnetId, available: avail})
		}
	}
	if len(entries) == 0 {
		return "", hcommon.I18nError(i18n.MsgSubnetIPExhausted)
	}

	// 加权随机：按 available 求和 total，落点 [0, total) 选中对应子网
	var total uint64
	for _, e := range entries {
		total += e.available
	}
	if total == 0 {
		// 理论不会走到（上面已过滤 available==0），防御性处理
		return entries[rand.Intn(len(entries))].id, nil
	}
	// rand.Int63n 支持的范围足以覆盖现实中的 IP 池容量（远小于 2^63）。
	// 注：Go 1.20+ 起 math/rand 的全局 Source 已默认用随机种子初始化
	// （见 go1.20 release notes），因此不需要在 main 里显式 rand.Seed。
	point := uint64(rand.Int63n(int64(total)))
	var acc uint64
	for _, e := range entries {
		acc += e.available
		if point < acc {
			return e.id, nil
		}
	}
	// 浮点精度兜底
	return entries[len(entries)-1].id, nil
}
