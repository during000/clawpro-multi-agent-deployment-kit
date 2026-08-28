package task

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/controller"
	"hatchery/i18n"
	"hatchery/model"
)

// smhProvisionDeps 定义 SMH 自动开通任务的外部依赖，方便单测通过接口注入 mock。
type smhProvisionDeps interface {
	GetSiteConfig(ctx context.Context) model.SiteConfig
	UpdateSiteConfig(ctx context.Context, updates interface{}) error
	ProvisionSMH(ctx context.Context) error
	EnsureLibrarySearchEnabled(ctx context.Context) error
	StartDefaultBundleSMHSync(ctx context.Context)
}

// defaultSMHProvisionDeps 是生产环境使用的默认实现，委托给 model / controller 包。
type defaultSMHProvisionDeps struct{}

func (d *defaultSMHProvisionDeps) GetSiteConfig(ctx context.Context) model.SiteConfig {
	return model.GetSiteConfig(ctx)
}
func (d *defaultSMHProvisionDeps) UpdateSiteConfig(ctx context.Context, updates interface{}) error {
	return model.UpdateSiteConfig(ctx, updates)
}
func (d *defaultSMHProvisionDeps) ProvisionSMH(ctx context.Context) error {
	return controller.ProvisionSMH(ctx)
}
func (d *defaultSMHProvisionDeps) EnsureLibrarySearchEnabled(ctx context.Context) error {
	return controller.EnsureLibrarySearchEnabled(ctx)
}
func (d *defaultSMHProvisionDeps) StartDefaultBundleSMHSync(ctx context.Context) {
	runDefaultBundleSMHSync(ctx)
}

// smhAutoProvisionTask 持有依赖，执行 SMH 自动开通逻辑。
type smhAutoProvisionTask struct {
	deps smhProvisionDeps
}

// newSMHAutoProvisionTask 创建任务实例；deps 为 nil 时使用默认实现。
func newSMHAutoProvisionTask(deps smhProvisionDeps) *smhAutoProvisionTask {
	if deps == nil {
		deps = &defaultSMHProvisionDeps{}
	}
	return &smhAutoProvisionTask{deps: deps}
}

// starts a background goroutine that automatically provisions SMH service.
//
// Behavior:
//   - Waits 10s after startup (buffer for STS credential refresh)
//   - Checks SiteConfig.SMHEnabled; if already provisioned, exits immediately
//   - If not provisioned, calls controller.ProvisionSMH()
//   - On failure, waits 300s (configurable via SMH_PROVISION_INTERVAL) and retries
//   - On success, goroutine exits (one-shot task)
//
// Environment variable control:
//   - DISABLE_SMH_AUTO_PROVISION: default "false" (enabled); set to "true" / "1" / "on" to disable
//   - SMH_PROVISION_INTERVAL: retry interval in seconds (default: 300)
func init() {
	// Check environment variable switch — default is enabled (false)
	v := os.Getenv("DISABLE_SMH_AUTO_PROVISION")
	if v == "" {
		v = "false" // default: enabled
	}
	switch strings.ToLower(v) {
	case "true", "1", "on":
		slog.Info("[SMH Provision] Disabled via DISABLE_SMH_AUTO_PROVISION, skipping")
		return
	default:
		// enabled, continue
	}

	// Parse retry interval
	intervalSec := 300
	if v := os.Getenv("SMH_PROVISION_INTERVAL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			intervalSec = n
		} else {
			slog.Warn("[SMH Provision] Invalid SMH_PROVISION_INTERVAL, using default", "value", v, "default", 300)
		}
	}

	RegisterTask(TaskDef{
		Name:         "smh-auto-provision",
		Interval:     time.Duration(intervalSec) * time.Second,
		RunFunc:      runSMHAutoProvisionEntry,
		NeedDistLock: true,
		PerTenant:    true,
		InitialDelay: 10 * time.Second,
	})
}

// runSMHAutoProvisionEntry 是 scheduler 调用的入口。
// 内部幂等：已完成开通则直接 return，未完成则尝试一次。
func runSMHAutoProvisionEntry(ctx context.Context) {
	task := newSMHAutoProvisionTask(nil)
	if err := task.safeSMHProvision(ctx); err == nil {
		slog.Info("[SMH Provision] SMH service is provisioned, triggering default bundle SMH sync")
		task.deps.StartDefaultBundleSMHSync(ctx)
	}
}

// safeSMHProvision wraps the provision logic with panic recovery to prevent goroutine crash.
// Returns nil only when both provisioning and EnableSearch check succeed; otherwise returns an error to trigger retry.
func (t *smhAutoProvisionTask) safeSMHProvision(ctx context.Context) (retErr error) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("[SMH Provision] Panic recovered, will retry", "panic", r)
			retErr = hcommon.I18nError(i18n.MsgSMHProvisionPanic, r)
		}
	}()

	config := t.deps.GetSiteConfig(ctx)
	if config.SMHEnabled == 1 {
		slog.Info("[SMH Provision] SMH service already provisioned, checking EnableSearch")
		if err := t.deps.EnsureLibrarySearchEnabled(ctx); err != nil {
			slog.Error("[SMH Provision] EnsureLibrarySearchEnabled failed, will retry", "error", err)
			return err
		}
		return nil
	}

	slog.Info("[SMH Provision] Starting SMH provisioning flow")

	if err := t.deps.ProvisionSMH(ctx); err != nil {
		slog.Error("[SMH Provision] Provisioning failed, will retry", "error", err)
		// 持久化错误信息，供前端查询展示
		_ = t.deps.UpdateSiteConfig(ctx, map[string]interface{}{"smh_provision_error": smhProvisionErrorMessage(err)})
		return err
	}

	slog.Info("[SMH Provision] SMH service provisioned successfully")

	// Ensure EnableSearch is turned on for the newly provisioned library
	if err := t.deps.EnsureLibrarySearchEnabled(ctx); err != nil {
		slog.Error("[SMH Provision] EnsureLibrarySearchEnabled failed after provisioning, will retry", "error", err)
		return err
	}

	return nil
}

// smhProvisionErrorMessage 将 SMH 开通错误转换为英文错误码。
// 已知错误映射为固定错误码，未知错误统一返回 INTERNAL_ERROR；
// 详细错误信息仅在日志中打印，不暴露给前端。
//
// 识别策略（按优先级从高到低）：
//  1. 先通过 errors.As 提取 controller.SMHProvisionError 中的错误码 —— 覆盖所有 hatchery 内部错误
//  2. 再对云 API 返回的原始错误信息做字符串匹配 —— 仅匹配外部不可控的错误关键字
//  3. 兜底返回 INTERNAL_ERROR
//
// 错误码清单：
//   - INSUFFICIENT_BALANCE   — 账户余额不足
//   - STS_ROLE_NOT_FOUND     — STS 角色不存在，凭证获取失败
//   - PROVISION_IN_PROGRESS  — 另一个实例正在执行开通
//   - CREATE_LIBRARY_FAILED  — 创建媒体库失败
//   - UPDATE_LIBRARY_FAILED  — 更新媒体库配置失败
//   - DESCRIBE_SECRET_FAILED — 获取媒体库密钥失败
//   - CREATE_SPACE_FAILED    — 创建空间失败
//   - INTERNAL_ERROR         — 内部错误（含网络错误、未知错误等）
func smhProvisionErrorMessage(err error) string {
	// ---- 第一优先级：从自定义错误类型中提取错误码 ----
	var provErr *controller.SMHProvisionError
	if errors.As(err, &provErr) {
		// 即使已有错误码，仍需检查云 API 错误信息中是否包含更具体的原因
		// 例如 CreateLibrary 因余额不足失败时，错误码为 CREATE_LIBRARY_FAILED，
		// 但云 API 错误信息中包含 BalanceLess，应优先返回 INSUFFICIENT_BALANCE
		if code := matchCloudAPIError(err.Error()); code != "" {
			return code
		}
		return provErr.Code
	}

	// ---- 第二优先级：字符串匹配云 API 返回的外部错误 ----
	if code := matchCloudAPIError(err.Error()); code != "" {
		return code
	}

	// ---- 兜底 ----
	return "INTERNAL_ERROR"
}

// matchCloudAPIError 对云 API / 外部错误信息做字符串匹配，返回对应错误码。
// 仅匹配我们无法控制的外部错误关键字（腾讯云 SDK 错误码、网络错误等）。
// 无法识别时返回空字符串。
func matchCloudAPIError(msg string) string {
	switch {
	// 余额不足（腾讯云计费系统返回）
	case strings.Contains(msg, "BalanceLess") || strings.Contains(msg, "INSUFFICIENT_BALANCE") || strings.Contains(msg, "balance less"):
		return "INSUFFICIENT_BALANCE"
	// STS 角色不存在（CAM 返回）
	case strings.Contains(msg, "role not exist") || strings.Contains(msg, "GetRoleError"):
		return "STS_ROLE_NOT_FOUND"
	// 网络错误 → 统一归为 INTERNAL_ERROR
	case strings.Contains(msg, "timeout") || strings.Contains(msg, "connection refused") || strings.Contains(msg, "no such host"):
		return "INTERNAL_ERROR"
	}
	return ""
}
