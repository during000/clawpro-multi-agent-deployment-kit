package controller

import (
	"net"
	"net/http"
	"strings"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// isTenantAgnosticPath 判断 path 是否允许跳过租户注入。
// 这些路径的 handler 自行处理租户上下文（或不需要租户上下文）。
func isTenantAgnosticPath(path string) bool {
	if path == "/health" {
		return true
	}
	// 多租户管理 API：AdminToken 鉴权，handler 内自行构造租户 ctx
	if strings.HasPrefix(path, "/tenants") {
		return true
	}
	return false
}

// extractHost 从请求中提取主机名（去端口号），统一转小写。
func extractHost(r *http.Request) string {
	host := r.Host
	// net.SplitHostPort 处理 host:port 和 [ipv6]:port
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return strings.ToLower(host)
}

// IdentifierMiddleware 将租户身份注入请求 ctx。
//   - 租户无关路径（/health、/tenants/*）：直接放行，不注入 TenantSnapshot
//   - 非 universe 模式（FixedSnapshot != nil）：注入 FixedSnapshot
//   - Universe 模式（FixedSnapshot == nil）：从 Host 动态解析租户
func IdentifierMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 租户无关路径：注入 SkipIdentifier 标记，确保后续 GORM 回调不 panic
		if isTenantAgnosticPath(r.URL.Path) {
			ctx := hcommon.WithSkipIdentifier(r.Context())
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// 非 universe 模式：使用启动时构造的固定快照
		if hcommon.FixedSnapshot != nil {
			ctx := hcommon.InjectTenant(r.Context(), *hcommon.FixedSnapshot)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Universe 模式：从 Host 动态解析租户
		host := extractHost(r)
		snap, err := model.ResolveTenant(r.Context(), host)
		if err != nil {
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgUnknownTenant))
			return
		}
		ctx := hcommon.InjectTenant(r.Context(), snap)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
