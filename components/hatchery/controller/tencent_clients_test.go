package controller

import (
	"context"
	"testing"

	"hatchery/common"
)

// 这些测试不需要真实 SiteConfig 数据；getCredential 调用 GetSiteConfig()
// 会返回一个空记录，凭证字段为空时应触发 "凭据未配置" 错误分支。

func TestGetCVMClient_NoCredential(t *testing.T) {
	setupMemoryProDB(t)
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "x"})
	_, err := GetCVMClient(ctx)
	if err == nil {
		t.Fatal("expected credential error in empty config, got nil")
	}
}

func TestGetVPCClient_NoCredential(t *testing.T) {
	setupMemoryProDB(t)
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "x"})
	_, err := GetVPCClient(ctx)
	if err == nil {
		t.Fatal("expected credential error in empty config, got nil")
	}
}

func TestGetSTSClient_NoCredential(t *testing.T) {
	setupMemoryProDB(t)
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "x"})
	_, err := GetSTSClient(ctx)
	if err == nil {
		t.Fatal("expected credential error in empty config, got nil")
	}
}

func TestGetCLSClient_NoCredential(t *testing.T) {
	setupMemoryProDB(t)
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "x"})
	_, err := GetCLSClient(ctx)
	if err == nil {
		t.Fatal("expected credential error in empty config, got nil")
	}
}

func TestGetTagClient_NoCredential(t *testing.T) {
	setupMemoryProDB(t)
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "x"})
	_, err := GetTagClient(ctx)
	if err == nil {
		t.Fatal("expected credential error in empty config, got nil")
	}
}
