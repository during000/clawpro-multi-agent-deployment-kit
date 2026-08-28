package controller

import (
	"context"
	"hatchery/common"
	"log/slog"
	"net/http"
	"strconv"

	"hatchery/controller/usergroup"
	"hatchery/i18n"
	"hatchery/model"
)

// ==================== 响应结构体 ====================

// ConfigStepItem 单条配置步骤状态
type ConfigStepItem struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Done  bool   `json:"done"`
}

// ==================== 配置步骤判断 ====================

// hasEnterpriseUsers 检查是否已导入企业用户
// 初始管理员始终存在，导入了用户 = 总数超过 1
func hasEnterpriseUsers(ctx context.Context) bool {
	var count int64
	result := model.DB(ctx).Model(&model.User{}).Count(&count)
	if result.Error != nil {
		slog.Error("查询用户数量失败", "error", result.Error)
		return false
	}
	return count > 1
}

// hasEnabledModel 检查是否存在已启用且全部用户可见的 AI 模型
// 排除内置占位模型（provider=hatchery, model_id=custom）
func hasEnabledModel(ctx context.Context) bool {
	return usergroup.HasGlobalModel(ctx)
}

// hasEnabledChannel 检查是否存在已启用且全部用户可见的通道
func hasEnabledChannel(ctx context.Context) bool {
	return usergroup.HasGlobalChannel(ctx)
}

// hasEnabledImage 检查是否存在已启用的镜像（全局可见）
func hasEnabledImage(ctx context.Context) bool {
	return usergroup.HasGlobalImage(ctx)
}

// hasConfiguredSecurityGroup 判断"配置安全组"步骤是否已完成。
// 兼容两种来源（任一满足即视为已配置）：
//  1. 老路径直配：SiteConfig.SecurityGroupId 非空（存量数据 / 老版"创建并自动绑定"、"绑定已有 SG"）
//  2. 新版 RuleSet + SG 池：managed_sg_pool 中存在至少一行 status=ACTIVE 的 SG
//     （新流程不再回写 SiteConfig.SecurityGroupId，只在 managed_sg_pool 落行）
func hasConfiguredSecurityGroup(ctx context.Context, config *model.SiteConfig) bool {
	if config != nil && config.SecurityGroupId != "" {
		return true
	}
	var count int64
	if err := model.DB(ctx).Model(&model.ManagedSGPool{}).
		Where("status = ?", model.SGStatusActive).
		Count(&count).Error; err != nil {
		slog.Error("查询 ACTIVE 安全组池数量失败", "error", err)
		return false
	}
	return count > 0
}

// buildConfigSteps 实时查 DB 判断配置完成状态
// 标准模式返回 6 步，OneID 模式（TenantID 非空）返回 7 步
func buildConfigSteps(ctx context.Context) []ConfigStepItem {
	config := model.GetSiteConfig(ctx)

	steps := []ConfigStepItem{
		{
			Key:   "brand",
			Label: i18n.T(ctx, i18n.MsgConfigStepBrand),
			Done:  config.Name != "",
		},
		{
			Key:   "default_quota",
			Label: i18n.T(ctx, i18n.MsgConfigStepDefaultQuota),
			Done:  config.DefaultInstanceQuota >= 1 || config.DefaultTokenQuotaDay > 0,
		},
		{
			Key:   "users",
			Label: i18n.T(ctx, i18n.MsgConfigStepUsers),
			Done:  hasEnterpriseUsers(ctx),
		},
	}

	// OneID 模式：插入"设置用户登录方式"步骤
	if common.TenantIDFromCtx(ctx) != "" {
		steps = append(steps, ConfigStepItem{
			Key:   "sso_login",
			Label: i18n.T(ctx, i18n.MsgConfigStepSSOLogin),
			Done:  len(config.GetSSOIMTypes()) > 0,
		})
	}

	steps = append(steps,
		ConfigStepItem{
			Key:   "model",
			Label: i18n.T(ctx, i18n.MsgConfigStepModel),
			Done:  hasEnabledModel(ctx),
		},
		ConfigStepItem{
			Key:   "channel",
			Label: i18n.T(ctx, i18n.MsgConfigStepChannel),
			Done:  hasEnabledChannel(ctx),
		},
		ConfigStepItem{
			Key:   "vpc",
			Label: i18n.T(ctx, i18n.MsgConfigStepVPC),
			Done:  usergroup.HasGlobalNetwork(ctx),
		},
		ConfigStepItem{
			Key:   "security_group",
			Label: i18n.T(ctx, i18n.MsgConfigStepSecurityGroup),
			Done:  usergroup.HasConfiguredSecurityGroup(ctx),
		},
		ConfigStepItem{
			Key:   "image",
			Label: i18n.T(ctx, i18n.MsgConfigStepImage),
			Done:  hasEnabledImage(ctx),
		},
	)

	return steps
}

// ==================== 公开 Handler ====================

// HandleAdminNotices 返回管控端通知栏所需的全部数据
// GET /admin/notices?limit=7&offset=0
// limit/offset 仅作用于 product_news，不传则返回全部（最多 20 条）
func HandleAdminNotices(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	// 解析产品动态分页参数
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	steps := buildConfigSteps(r.Context())
	alerts := buildQuotaAlerts(r.Context())
	news := buildProductNews(r.Context(), limit, offset)

	jsonOK(w, map[string]any{
		"config_steps": steps,
		"quota_alerts": alerts,
		"product_news": news,
	})
}
