package controller

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SMH 内网域名切换机制
//
// 背景：
//   - SMH OpenAPI 提供 internal_domain=0/1 查询参数控制返回的 COS 上传/下载域名走内网还是外网。
//   - 当前依赖的 smh-go-sdk 在 MultipartUploadFile / RenewMultipartUpload builder 上未暴露
//     InternalDomain() 方法，无法在调用链上直接传递。
//   - 因此通过 ctx 标记 + 自定义 RoundTripper 的方式，在 HTTP 层面对相关接口的 URL query
//     注入 internal_domain=1，等价于"用上了 SDK 暂未暴露的能力"。
//
// 使用方式：
//
//	ctx := WithSMHInternalDomain(r.Context(), true)
//	cred, err := PrepareSMHCommonUpload(ctx, ...)
//	// cred.UsedInternalDomain == true，Renew 时会自动保持一致
//
//	// CVM 角度可达性不确定时，可用 ProbeSMHInternalReachable 做探测并降级：
//	if !ProbeSMHInternalReachable(ctx, cred.PartURLTemplate) {
//	    cred, err = PrepareSMHCommonUpload(WithSMHInternalDomain(ctx, false), ...)
//	}

// smhInternalDomainCtxKey 是用于在 context 上携带"使用内网域名"标记的私有 key。
type smhInternalDomainCtxKey struct{}

// WithSMHInternalDomain 返回一个标记了"是否使用 SMH 内网域名"的派生 context。
// 当 on==true 时，后续通过该 ctx 调用的 SMH 上传/续期 API 将在 query 上追加
// internal_domain=1；当 on==false 时移除任何已有的标记（即走外网）。
func WithSMHInternalDomain(ctx context.Context, on bool) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, smhInternalDomainCtxKey{}, on)
}

// smhInternalDomainEnabled 判断给定 ctx 是否携带"使用内网域名"标记。
func smhInternalDomainEnabled(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	v, _ := ctx.Value(smhInternalDomainCtxKey{}).(bool)
	return v
}

// smhInternalDomainTransport 是注入到 SMH SDK HTTPClient 上的 RoundTripper。
// 仅对会返回上传域名的 API 路径（MultipartUploadFile / RenewMultipartUpload）
// 且 ctx 标记了"使用内网域名"时，自动在 URL query 上追加 internal_domain=1。
// 其它请求保持原样直通，避免对鉴权、目录管理等无关接口造成副作用。
type smhInternalDomainTransport struct {
	base http.RoundTripper
}

// smhInternalDomainPaths 是需要识别并注入 internal_domain 参数的 API 路径关键字。
// 之所以用 Contains 匹配而不是精确路径：SMH 不同 SDK 版本可能路径前缀略有差异，
// 但这些关键字（含 LibraryId/SpaceId/FilePath 之外的固定标识）足以稳定区分。
//
// 注意：SMH SDK 中 RenewMultipartUpload 也走 /api/v1/file/{lib}/{space}/{confirmKey}?renew=1
// 与 MultipartUploadFile 共享 /api/v1/file/ 前缀，因此一个关键字即可覆盖两类请求；
// 区分两者通过下方 query 上的 multipart / renew flag。
var smhInternalDomainPaths = []string{
	// 开始分块上传：POST /api/v1/file/{lib}/{space}/{filePath}?multipart=1
	// 续期分块上传：PUT  /api/v1/file/{lib}/{space}/{confirmKey}?renew=1
	"/api/v1/file/",
}

// shouldInjectInternalDomain 判断当前请求是否需要追加 internal_domain=1。
// 满足两个条件：
//  1. ctx 带 smhInternalDomainCtxKey=true；
//  2. URL 路径属于上传/续期相关 API（避免误伤其它接口）；
//  3. 已有 query 上的 multipart=1 或 renew=1（进一步收敛到本次方案关心的两个动作）。
func shouldInjectInternalDomain(req *http.Request) bool {
	if req == nil || req.URL == nil {
		return false
	}
	if !smhInternalDomainEnabled(req.Context()) {
		return false
	}
	hit := false
	for _, p := range smhInternalDomainPaths {
		if strings.Contains(req.URL.Path, p) {
			hit = true
			break
		}
	}
	if !hit {
		return false
	}
	q := req.URL.Query()
	if _, ok := q["multipart"]; ok {
		return true
	}
	if _, ok := q["renew"]; ok {
		return true
	}
	return false
}

// RoundTrip 实现 http.RoundTripper：必要时在 query 上注入 internal_domain=1。
// 该方法不会修改入参 req，而是基于 req 浅拷贝出新 request 再修改 URL，
// 符合 http.RoundTripper 契约。
func (t *smhInternalDomainTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	if !shouldInjectInternalDomain(req) {
		return base.RoundTrip(req)
	}
	// 拷贝 URL，避免直接修改入参
	newURL := *req.URL
	q := newURL.Query()
	// 幂等：若已存在则覆盖为 1
	q.Set("internal_domain", "1")
	newURL.RawQuery = q.Encode()

	newReq := req.Clone(req.Context())
	newReq.URL = &newURL

	// DEBUG 日志：记录注入行为，便于在抓不到包但能拿到日志的环境中确认
	// SDK 请求确实带上了 internal_domain=1。仅在 DEBUG 级别输出，不会刷屏。
	// 注意：从 req.Context() 直接取 logger 可能拿不到调用方注入的 logger（SDK 内部会重新构造 request），
	// 因此使用全局 slog 以保证至少能落到主日志流。
	slog.Debug("SMH HTTP 请求注入 internal_domain=1",
		"method", newReq.Method,
		"path", newReq.URL.Path,
		"host", newReq.URL.Host,
	)
	return base.RoundTrip(newReq)
}

// smhInternalProbeTimeout 是 hatchery 侧探测 SMH 内网 Host 可达性的超时时间。
// 取值较短：探测仅作为内网→外网降级的判定依据，不应阻塞主升级流程。
const smhInternalProbeTimeout = 2 * time.Second

// smhInternalProbeDial 暴露 dial 函数指针以便单测注入。
var smhInternalProbeDial = (&net.Dialer{Timeout: smhInternalProbeTimeout}).DialContext

// ProbeSMHInternalReachable 探测给定 URL 的 Host:Port 是否可达，用于判断当前 hatchery
// 所在 VPC 是否能够通过内网域名访问 SMH。返回 true 表示可达，false 表示需要降级走外网。
//
// 注意：本探测是从 hatchery 视角进行的，与最终发起上传的 CVM 视角可能存在差异；
// 但二者一般部署在同 VPC 内（管控面与数据面同地域），探测结果具有较强参考价值。
// 若 hatchery 与 CVM 不在同 VPC，应通过显式配置（如 SMH_PREFER_INTERNAL_UPLOAD）覆盖。
func ProbeSMHInternalReachable(ctx context.Context, partURLTemplate string) bool {
	if partURLTemplate == "" {
		return false
	}
	u, err := url.Parse(partURLTemplate)
	if err != nil || u.Host == "" {
		return false
	}
	host := u.Host
	// url.Host 不一定带端口，按 scheme 推断默认端口
	if !strings.Contains(host, ":") {
		switch strings.ToLower(u.Scheme) {
		case "http":
			host += ":80"
		default:
			host += ":443"
		}
	}
	probeCtx, cancel := context.WithTimeout(ctx, smhInternalProbeTimeout)
	defer cancel()
	conn, err := smhInternalProbeDial(probeCtx, "tcp", host)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
