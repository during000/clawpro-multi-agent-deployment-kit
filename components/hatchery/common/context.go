package common

import (
	"context"
	"hatchery/i18n"
	"math"
	"slices"

	"github.com/google/uuid"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// contextKey 是本包内部 context key 的私有类型，避免与其他包 key 碰撞。
type contextKey int

const (
	// ctxKeyTenantSnapshot 承载当前请求/任务的 TenantSnapshot。
	ctxKeyTenantSnapshot contextKey = iota + 1

	// ctxKeySkipIdentifier 标记本次 DB 调用跳过 identifier 过滤（用于全局表）。
	ctxKeySkipIdentifier

	// CtxKeyRequestID 承载当前请求/任务的唯一标识符。
	CtxKeyRequestID

	// CtxKeyTraceID 承载当前请求/任务的链路追踪标识符。
	CtxKeyTraceID

	// CtxKeyInterface 承载当前请求/任务的接口名称。
	CtxKeyInterface

	// CtxKeyIsTask 标记当前 context 是否为定时任务/后台 goroutine。
	CtxKeyIsTask

	// CtxKeyUin 承载当前请求/任务的腾讯云 UIN。
	CtxKeyUin

	// CtxKeySubUin 承载当前请求/任务的系统用户 ID。
	CtxKeySubUin
)

// TenantSnapshot 是一条租户的身份 + 凭证快照。
// 仅缓存"初始化后几乎不变"的字段，频繁变更的配置（如 STS 临时密钥、Session 相关）
// 不入 Snapshot，每次穿透 DB 查 SiteConfig 获取。
//
// 详见 openspec/changes/multi-tenant-universe-mode/design.md §4.2。
type TenantSnapshot struct {
	Identifier string // 租户标识（GORM 回调据此注入 WHERE identifier=?）
	Uin        string // 腾讯云 UIN
	Domain     string // 对外域名

	// ── OneID 相关 ──
	OneIDAccountID     string // OneID 租户 ID（account_id）
	OneIDAppID         string // 自建应用 ID（非空=统一账号模式）
	OneIDClientID      string // 自建应用 client_id
	OneIDClientSecret  string // 自建应用 client_secret
	OneIDTokenEndpoint string // Token 获取端点
	OneIDDomain        string // OneID 企业域名
	InternalSecret     string // Gateway 内部鉴权密钥（HMAC 签名用）

	// ── 云资源凭证 ──
	CVMSecretId           string // 永久凭证（腾讯云 SDK 用）
	CVMSecretKey          string
	AgentCamRoleSecretId  string // Agent CAM 角色凭证
	AgentCamRoleSecretKey string

	// ── 多租户语言配置 ──
	DefaultLang string // 租户默认语言："zh" 或 "en"，为空时回退到进程级 --lang 参数

	// ── 多租户安全策略配置 ──
	SecurityPolicies []string
}

// InjectTenant 将 TenantSnapshot 注入 ctx，同时清除 SkipIdentifier 标记。
// 两者互斥：注入租户身份意味着后续 DB 操作应使用该 identifier。
func InjectTenant(ctx context.Context, snap TenantSnapshot) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = context.WithValue(ctx, ctxKeySkipIdentifier, false)
	return context.WithValue(ctx, ctxKeyTenantSnapshot, snap)
}

// GetTenantSnapshot 从 ctx 中提取 TenantSnapshot。
// 第二个返回值为 false 表示 ctx 中未注入租户快照。
func GetTenantSnapshot(ctx context.Context) (TenantSnapshot, bool) {
	if ctx == nil {
		return TenantSnapshot{}, false
	}
	v := ctx.Value(ctxKeyTenantSnapshot)
	if v == nil {
		return TenantSnapshot{}, false
	}
	snap, ok := v.(TenantSnapshot)
	return snap, ok
}

// WithSkipIdentifier 派生一个标记"跳过 identifier 过滤"的 ctx，同时清除 TenantSnapshot。
// 两者互斥：跳过隔离意味着后续 DB 操作不应绑定任何租户。
// 用于访问全局表（如 tenant_domains、分布式锁辅助表）时绕过 GORM 回调的租户隔离。
func WithSkipIdentifier(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = context.WithValue(ctx, ctxKeyTenantSnapshot, nil)
	return context.WithValue(ctx, ctxKeySkipIdentifier, true)
}

// ShouldSkipIdentifier 报告是否处于"跳过 identifier 过滤"模式。
func ShouldSkipIdentifier(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	v, _ := ctx.Value(ctxKeySkipIdentifier).(bool)
	return v
}

// WithTaskTrace 为定时任务/后台 goroutine 注入日志链路字段（request_id、trace_id、interface、isTask、uin）。
func WithTaskTrace(ctx context.Context, taskName string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	// 注入日志链路字段
	ctx = context.WithValue(ctx, CtxKeyRequestID, uuid.New().String())
	ctx = context.WithValue(ctx, CtxKeyTraceID, uuid.New().String())
	ctx = context.WithValue(ctx, CtxKeyInterface, taskName)
	ctx = context.WithValue(ctx, CtxKeyIsTask, true)

	// 获取 uin：从 TenantSnapshot 读取
	uin := ""
	if snap, ok := GetTenantSnapshot(ctx); ok && snap.Uin != "" {
		uin = snap.Uin
	}
	if uin != "" {
		ctx = context.WithValue(ctx, CtxKeyUin, uin)
	}

	return ctx
}

// GetRequestID 从 ctx 中提取 request_id。
func GetRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(CtxKeyRequestID).(string); ok {
		return v
	}
	return ""
}

// GetTraceID 从 ctx 中提取 trace_id。
func GetTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(CtxKeyTraceID).(string); ok {
		return v
	}
	return ""
}

// GetInterface 从 ctx 中提取 interface（接口名称）。
func GetInterface(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(CtxKeyInterface).(string); ok {
		return v
	}
	return ""
}

// IsTask 报告当前 ctx 是否为定时任务/后台 goroutine。
func IsTask(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	v, _ := ctx.Value(CtxKeyIsTask).(bool)
	return v
}

// GetSubUin 从 ctx 中提取 sub_uin（系统用户 ID）。
func GetSubUin(ctx context.Context) uint {
	if ctx == nil {
		return 0
	}
	if v, ok := ctx.Value(CtxKeySubUin).(uint); ok {
		return v
	}
	return 0
}

// DetachContext 从父 context 派生一个新 context，完整复制所有已知 value（租户快照、链路追踪字段等），
// 但不继承取消信号。用于在异步任务中保持完整上下文，同时避免父 context 取消影响子任务。
func DetachContext(parent context.Context) context.Context {
	ctx := context.Background()
	if parent == nil {
		return ctx
	}

	// 复制租户快照
	if snap, ok := GetTenantSnapshot(parent); ok {
		ctx = InjectTenant(ctx, snap)
	}

	ctx = i18n.WithPrinter(ctx, parent)
	if _, exists := i18n.Printer(ctx); !exists {
		ctx = InjectI18nPrinter(ctx)
	}

	// 复制跳过标识符标记
	if ShouldSkipIdentifier(parent) {
		ctx = WithSkipIdentifier(ctx)
	}

	// 复制链路追踪字段（WithTaskTrace 注入的）
	if v := parent.Value(CtxKeyRequestID); v != nil {
		ctx = context.WithValue(ctx, CtxKeyRequestID, v)
	}
	if v := parent.Value(CtxKeyTraceID); v != nil {
		ctx = context.WithValue(ctx, CtxKeyTraceID, v)
	}
	if v := parent.Value(CtxKeyInterface); v != nil {
		ctx = context.WithValue(ctx, CtxKeyInterface, v)
	}
	if v := parent.Value(CtxKeyIsTask); v != nil {
		ctx = context.WithValue(ctx, CtxKeyIsTask, v)
	}
	if v := parent.Value(CtxKeyUin); v != nil {
		ctx = context.WithValue(ctx, CtxKeyUin, v)
	}
	if v := parent.Value(CtxKeySubUin); v != nil {
		ctx = context.WithValue(ctx, CtxKeySubUin, v)
	}

	return ctx
}

// CVMUinFromCtx 返回当前租户的腾讯云 UIN。
func CVMUinFromCtx(ctx context.Context) string {
	if snap, ok := GetTenantSnapshot(ctx); ok {
		return snap.Uin
	}
	return ""
}

// DomainFromCtx 返回当前租户的对外域名。
func DomainFromCtx(ctx context.Context) string {
	if snap, ok := GetTenantSnapshot(ctx); ok {
		return snap.Domain
	}
	return ""
}

// ── OneID 相关 Context Helper ────────────────────────────────────────────────

// InternalSecretFromCtx 返回当前租户的 Gateway 内部鉴权密钥。
func InternalSecretFromCtx(ctx context.Context) string {
	if snap, ok := GetTenantSnapshot(ctx); ok {
		return snap.InternalSecret
	}
	return ""
}

// TenantIDFromCtx 返回当前租户的 OneID 账号 ID（account_id）。
func TenantIDFromCtx(ctx context.Context) string {
	if snap, ok := GetTenantSnapshot(ctx); ok {
		return snap.OneIDAccountID
	}
	return ""
}

// OneIDAppIDFromCtx 返回当前租户的 OneID 自建应用 ID。
// 非空表示该租户启用了统一账号模式。
func OneIDAppIDFromCtx(ctx context.Context) string {
	if snap, ok := GetTenantSnapshot(ctx); ok {
		return snap.OneIDAppID
	}
	return ""
}

// IsUnifiedAccountMode 判断当前租户是否启用了统一账号模式。
func IsUnifiedAccountMode(ctx context.Context) bool {
	return OneIDAppIDFromCtx(ctx) != ""
}

// DefaultLangFromCtx 返回当前租户的默认语言。
// 若 ctx 中无租户快照或 DefaultLang 为空，返回空字符串（调用方自行回退到全局 --lang）。
func DefaultLangFromCtx(ctx context.Context) string {
	if snap, ok := GetTenantSnapshot(ctx); ok {
		return snap.DefaultLang
	}
	return ""
}

func InjectI18nPrinter(ctx context.Context) context.Context {
	defaultLang := DefaultLangFromCtx(ctx)
	m := i18n.BuildMatcher(defaultLang)
	tag, _ := language.MatchStrings(m, defaultLang)
	printer := message.NewPrinter(tag)
	return context.WithValue(ctx, i18n.CtxKey{}, printer)
}

// IsSSRFEnabledInSecurityPolicies 判断当前租户是否启用了 SSRF 安全策略。
func IsSSRFEnabledInSecurityPolicies(ctx context.Context) bool {
	if snap, ok := GetTenantSnapshot(ctx); ok {
		if slices.Contains(snap.SecurityPolicies, "SSRF") {
			return true
		}
	}
	return false
}

// UserLimitFromCtx 返回当前租户的用户数上限。
// 多租户阶段一：不再从数据库获取限制，始终返回无限制（math.MaxInt）。
func UserLimitFromCtx(_ context.Context) int {
	return math.MaxInt
}
