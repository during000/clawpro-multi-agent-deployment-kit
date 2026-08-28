package controller

import (
	"context"

	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
)

// ============================================================================
// controller 包对外暴露的测试 hook（仅供其它测试包使用，例如 task/）。
//
// 保留在 _test 结尾之外的单独文件里，因为 task 包的测试需要在编译期就看到这些符号。
// 若将来 go 支持 cross-package test helper 的更好机制，可重构。
// ============================================================================

// SGVpcClient 是 sgVpcClient 的导出别名，供 task 包测试使用。
type SGVpcClient = sgVpcClient

// SetNewVpcClientForSGFnForTest 替换 newVpcClientForSGFn，并返回 restore 闭包。
// 仅用于 task 包测试从外部注入 fake VPC client；生产代码不得调用。
func SetNewVpcClientForSGFnForTest(f func(ctx context.Context) (SGVpcClient, error)) func() {
	orig := newVpcClientForSGFn
	newVpcClientForSGFn = f
	return func() { newVpcClientForSGFn = orig }
}

// PolicySetToRules 是 policySetToRules 的导出别名，供 task 包（SG Guardian）复用
// 云端规则 → 本地 Rule 模型的转换逻辑。生产代码也使用本符号（不限于测试），命名沿用
// "Test 文件外的可见 helper"约定即可。后续若 controller 内重构了实现，记得同步更新。
func PolicySetToRules(set *vpc.SecurityGroupPolicySet) []Rule {
	return policySetToRules(set)
}

// RuleFingerprint 暴露 Rule.Fingerprint（包外通过 Rule 直接调用即可，但 Rule 是导出类型，
// 这里仅作为 PolicySetToRules 的伴生说明 —— task 包可直接 r.Fingerprint() 调用）。
// 保留空 var 防 import 漂移。
var _ = (Rule{}).Fingerprint

// 避免 unused import
var _ = vpc.NewDescribeSecurityGroupsRequest
