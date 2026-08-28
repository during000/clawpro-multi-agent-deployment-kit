package controller

import (
	"context"
	hcommon "hatchery/common"
	"hatchery/i18n"
	"net/http"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// IsOverseasFromCtx 从 context 中获取租户级默认语言，判断是否为海外版
// 优先使用 IdentifierMiddleware 注入的租户 DefaultLang；若无则回退到进程级参数
func IsOverseasFromCtx(ctx context.Context) bool {
	lang := hcommon.DefaultLangFromCtx(ctx)
	if lang == "en" {
		return true
	}
	return i18n.IsOverseas()
}

// detectLang 按优先级检测语言：
//  1. URL 查询参数 ?lang=en
//  2. Accept-Language 请求头
func detectLang(r *http.Request) language.Tag {
	// 获取租户级 matcher（优先租户 DefaultLang，否则全局）
	// matcher 决定了在以下逻辑无法检测出语言时的默认语言
	m := i18n.BuildMatcher(hcommon.DefaultLangFromCtx(r.Context()))

	if q := r.URL.Query().Get("lang"); q != "" {
		tag, _ := language.Parse(q)
		matched, _, _ := m.Match(tag)
		return matched
	}
	accept := r.Header.Get("Accept-Language")
	tag, _ := language.MatchStrings(m, accept)
	return tag
}

// Middleware 解析请求语言偏好，将 *message.Printer 注入 context。
// 优先使用租户级 DefaultLang（从上下文读取），
// 若无则回退到全局 supportedLangs（由 --lang 参数设置）。
func I18nMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lang := detectLang(r)
		printer := message.NewPrinter(lang)
		ctx := context.WithValue(r.Context(), i18n.CtxKey{}, printer)
		// 将原始 Accept-Language 头存入 context，供后台函数转发给下游服务
		ctx = i18n.SetAcceptLanguage(ctx, r.Header.Get("Accept-Language"))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
