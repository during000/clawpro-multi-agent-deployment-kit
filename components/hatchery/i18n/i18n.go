package i18n

import (
	"context"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type Key struct {
	string
}

func (key *Key) String() string {
	return key.string
}

// 直接从字符串构建 I18n Key，尽量不要用这个构造函数
// 除非万不得已
func NewKey(s string) Key {
	return Key{s}
}

type KeyAndArgs struct {
	Key  Key
	Args []any
}

// 支持的语言，index 0 为默认语言
var supportedLangs = []language.Tag{
	language.Chinese,
	language.English,
}

var matcher = language.NewMatcher(supportedLangs)

// SetDefault 如果输入为 `en`，则设置为英文；如果输入为 `zh`，或其他不合法参数，则保持默认中文
func SetDefaultLang(lang string) {
	if lang == "en" {
		supportedLangs = []language.Tag{
			language.English,
			language.Chinese,
		}
	} else {
		supportedLangs = []language.Tag{
			language.Chinese,
			language.English,
		}
	}
	matcher = language.NewMatcher(supportedLangs)
}

// BuildMatcher 根据给定的默认语言构建语言匹配器。
// 返回的 matcher 首语言为 defaultLang，用于租户级语言检测。
func BuildMatcher(defaultLang string) language.Matcher {
	if defaultLang == "" {
		return matcher
	}

	if defaultLang == "en" {
		return language.NewMatcher([]language.Tag{
			language.English,
			language.Chinese,
		})
	}

	return language.NewMatcher([]language.Tag{
		language.Chinese,
		language.English,
	})
}

// IsOverseas 判断全局默认语言是否为海外版（非中文）。
// 该函数仅用于非 universe 模式或兜底场景；
// universe 模式应使用 IsOverseasFromCtx()。
func IsOverseas() bool {
	return supportedLangs[0] != language.Chinese
}

type CtxKey struct{}

// ctxKeyAcceptLanguage 承载原始请求的 Accept-Language 头值，
// 供后台函数（无 *http.Request）转发给下游服务使用。
type ctxKeyAcceptLanguage struct{}

// SetAcceptLanguage 将原始请求的 Accept-Language 头值存入 context。
func SetAcceptLanguage(ctx context.Context, al string) context.Context {
	return context.WithValue(ctx, ctxKeyAcceptLanguage{}, al)
}

// AcceptLanguageFromCtx 从 context 中取出原始请求的 Accept-Language 头值。
// 若未设置则返回空字符串。
func AcceptLanguageFromCtx(ctx context.Context) string {
	if al, ok := ctx.Value(ctxKeyAcceptLanguage{}).(string); ok {
		return al
	}
	return ""
}

// Printer 从 context 中获取当前请求的 Printer。
func Printer(ctx context.Context) (*message.Printer, bool) {
	if p, ok := ctx.Value(CtxKey{}).(*message.Printer); ok {
		return p, true
	}
	return message.NewPrinter(language.Chinese), false
}

// T 翻译并格式化消息，用法与 fmt.Sprintf 一致。
func T(ctx context.Context, key Key, args ...interface{}) string {
	p, _ := Printer(ctx)
	return p.Sprintf(key.string, args...)
}

// WithPrinter 把 src 中的 *message.Printer 复制到 parent 上，返回派生 ctx。
//
// 适用场景：异步任务（go ...）需要保留请求级语言偏好，但又要解耦请求生命周期
// （例如 controller 已用 common.DetachContext 取得不会被 r.Context() 取消的 ctx，
// 但 DetachContext 只复制部分 value，并不一定带上 i18n 的 Printer）。
//
// 若 src 没有 Printer，直接返回 parent，T() 会 fallback 到中文。
func WithPrinter(parent, src context.Context) context.Context {
	if p, ok := src.Value(CtxKey{}).(*message.Printer); ok {
		return context.WithValue(parent, CtxKey{}, p)
	}
	return parent
}
