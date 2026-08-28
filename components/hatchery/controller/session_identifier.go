package controller

import (
	"context"

	"github.com/gorilla/sessions"

	"hatchery/common"
)

// 多租户 — Session identifier 注入与校验。
// 阶段一：登录时写入 identifier 到 session。
// 阶段二：universe 模式下严格校验 identifier 一致性。

const sessionKeyIdentifier = "identifier"

// setSessionIdentifier 将当前 ctx 的租户 identifier 写入 session。
// 登录成功时(HandleLogin / OneID 回调)由调用方在 session.Save 之前调用。
func setSessionIdentifier(session *sessions.Session, ctx context.Context) {
	if session == nil {
		return
	}
	id := currentIdentifierFromCtx(ctx)
	if id != "" {
		session.Values[sessionKeyIdentifier] = id
	}
}

// validateSessionIdentifier 校验 session 中的 identifier 是否与当前请求的 ctx 一致。
//   - Universe 模式下：session 必须含 identifier 且与 ctx 中的一致，否则视为未登录
//   - 非 universe 模式：兼容放行（阶段一旧 cookie 无 identifier 也算通过）
func validateSessionIdentifier(session *sessions.Session, ctx context.Context) bool {
	if session == nil {
		return false
	}

	snap, ok := common.GetTenantSnapshot(ctx)
	if !ok || snap.Identifier == "" {
		// 无 TenantSnapshot（SQLite 或 agnostic path）→ 放行
		return true
	}

	sessionId, _ := session.Values[sessionKeyIdentifier].(string)

	if common.IsUniverseMode() {
		// Universe 模式：严格校验
		if sessionId == "" {
			return false // 旧 cookie 无 identifier → 不兼容放行
		}
		return sessionId == snap.Identifier
	}

	// 非 universe 模式：兼容放行旧 cookie（无 identifier 视为通过）
	if sessionId == "" {
		return true
	}
	return sessionId == snap.Identifier
}

// currentIdentifierFromCtx 从 ctx.TenantSnapshot 读 identifier。
func currentIdentifierFromCtx(ctx context.Context) string {
	if snap, ok := common.GetTenantSnapshot(ctx); ok {
		return snap.Identifier
	}
	return ""
}
