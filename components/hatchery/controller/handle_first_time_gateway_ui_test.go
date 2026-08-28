package controller

import (
	"context"
	"strings"
	"testing"

	hcommon "hatchery/common"
	"hatchery/model"
)

// 这些测试覆盖 handleFirstTimeGatewayUIEnable 当前实现：
//   1. 未配置安全组（SecurityGroupId=""）→ 报错
//   2. 未配置安全组（SecurityGroupId=""）+ 有 RuleSet → 仍然报错（安全组检查优先）
//   3. GatewayUIPort=0 → 先分配新端口（即使后续 API 调用可能失败，端口字段已更新）
//   4. GatewayUIPort 已设置 → 沿用旧端口

// TestHandleFirstTimeGatewayUIEnable_NotReady_NoRuleSet 覆盖：SecurityGroupId="" → 报错
func TestHandleFirstTimeGatewayUIEnable_NotReady_NoRuleSet(t *testing.T) {
	_ = setupSGPoolTestDB(t)

	ctx := context.Background()
	cfg := &model.SiteConfig{}
	err := handleFirstTimeGatewayUIEnable(ctx, cfg)
	if err == nil {
		t.Fatal("expected error when SecurityGroupId not configured")
	}
	if !strings.Contains(hcommon.ErrorMessageWithCtx(ctx, err), "安全组") {
		t.Errorf("error should mention 安全组, got: %v", err)
	}
	if cfg.GatewayUISGMigrateDone {
		t.Error("GatewayUISGMigrateDone should not be set when not ready")
	}
}

// TestHandleFirstTimeGatewayUIEnable_NotReady_NoActiveSG 覆盖：SecurityGroupId="" → 报错
func TestHandleFirstTimeGatewayUIEnable_NotReady_NoActiveSG(t *testing.T) {
	db := setupSGPoolTestDB(t)
	// 插入 RuleSet，但安全组未配置 → 仍然报错
	db.Create(&model.RuleSet{Name: model.DefaultRuleSetName, Rules: "[]", Version: 1, IsDefault: true})

	ctx := context.Background()
	cfg := &model.SiteConfig{}
	err := handleFirstTimeGatewayUIEnable(ctx, cfg)
	if err == nil {
		t.Fatal("expected error when SecurityGroupId not configured")
	}
	if !strings.Contains(hcommon.ErrorMessageWithCtx(ctx, err), "安全组") {
		t.Errorf("error should mention 安全组, got: %v", err)
	}
}

// TestHandleFirstTimeGatewayUIEnable_Success_NewPort 覆盖：GatewayUIPort=0 → 分配新端口
// 由于 listInstanceIds/addGatewayUISecurityGroupRule 需要真实 CVM，
// 本测试仅验证在 SecurityGroupId="" 时函数不会 panic，并正确返回错误。
// 端口分配逻辑在 SecurityGroupId 检查前，可通过验证 GatewayUIPort 已被赋值来覆盖。
func TestHandleFirstTimeGatewayUIEnable_Success_NewPort(t *testing.T) {
	_ = setupSGPoolTestDB(t)

	cfg := &model.SiteConfig{GatewayUIPort: 0}
	// SecurityGroupId="" → 会报错，但端口应已分配（因为端口分配在安全组检查之前）
	err := handleFirstTimeGatewayUIEnable(context.Background(), cfg)
	if err == nil {
		// 如果意外成功（未来生产代码变更），只校验端口字段
		if cfg.GatewayUIPort == 0 {
			t.Error("GatewayUIPort should be assigned when port was 0")
		}
		return
	}
	// SecurityGroupId="" 导致的预期错误；但端口应已被分配
	if cfg.GatewayUIPort == 0 {
		t.Error("GatewayUIPort should be assigned before SecurityGroupId check")
	}
	if cfg.GatewayUISGMigrateDone {
		t.Error("GatewayUISGMigrateDone should not be set on error")
	}
}

// TestHandleFirstTimeGatewayUIEnable_Success_ExistingPort 覆盖：sticky 沿用旧端口
func TestHandleFirstTimeGatewayUIEnable_Success_ExistingPort(t *testing.T) {
	_ = setupSGPoolTestDB(t)

	const stickyPort = 23456
	cfg := &model.SiteConfig{GatewayUIPort: stickyPort}
	// SecurityGroupId="" → 报错，但端口应保持不变
	_ = handleFirstTimeGatewayUIEnable(context.Background(), cfg)

	if cfg.GatewayUIPort != stickyPort {
		t.Errorf("existing port should be preserved, want %d got %d", stickyPort, cfg.GatewayUIPort)
	}
}
