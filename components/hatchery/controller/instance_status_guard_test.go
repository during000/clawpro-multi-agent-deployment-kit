package controller

import (
	"context"

	"hatchery/model"
)

// mockRunningStatusResolver 是测试用的 instanceStatusResolver 实现，
// 总是返回 running 状态，使 guard 放行。
type mockRunningStatusResolver struct{}

func (mockRunningStatusResolver) ResolveStatus(_ context.Context, instance *model.Instance) (InstanceStatusResponse, error) {
	return InstanceStatusResponse{Status: model.StatusRunning, Label: "运行中"}, nil
}

// testCVMFetcher 是全局测试 mock，所有双层化 handler 测试使用此实例。
var testCVMFetcher instanceStatusResolver = mockRunningStatusResolver{}

// mockStatusResolverWithStatus 返回指定状态，用于测试 guard 拒绝场景。
type mockStatusResolverWithStatus struct {
	status string
	label  string
}

func (m *mockStatusResolverWithStatus) ResolveStatus(_ context.Context, instance *model.Instance) (InstanceStatusResponse, error) {
	return InstanceStatusResponse{Status: m.status, Label: m.label}, nil
}
