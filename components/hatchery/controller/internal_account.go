// Package controller 中的 internal_account.go 提供"当前部署是否运行在内部测试账号下"的统一判断。
//
// 设计要点：
//  1. 业务侧统一调用 IsInternalAccount(ctx)，避免散落各处的 UIN 字符串硬编码；
//  2. UIN 获取兼容两种来源——优先从 ctx 中的 TenantSnapshot 拿（多租户模式下已注入），
//     若拿不到（比如未注入快照的早期/异步路径），则降级调用腾讯云 CAM GetUserAppId 接口
//     （文档：https://cloud.tencent.com/document/api/598/70416）；
//  3. UIN 是部署级常量，CAM 调用结果使用 sync.Once 进程内缓存成功值；失败不缓存，下次自然重试。
package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tchttp "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/http"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"

	hcommon "hatchery/common"
	"hatchery/model"
)

// internalAccountUins 是被视为"内部测试账号"的腾讯云主账号 UIN 白名单。
// 命中其中任意一项，则同时启用：办公网入站安全组规则 + 内部测试隐藏参数
// （如 image_id）。后续如需新增内部账号，直接在此处追加，不要再在业务代码里
// 硬编码 UIN 字符串；业务侧统一调用 IsInternalAccount(ctx) 进行判定。
var internalAccountUins = map[string]struct{}{
	"3205597606":   {},
	"100049049642": {},
}

// isInternalAccountUin 仅做 UIN→bool 的纯映射查询，是 IsInternalAccount 的内部
// 实现细节。业务方请勿绕开 IsInternalAccount 直接调用本函数，更不要硬编码 UIN
// 字符串——所有"是否内部账号"的判定都应汇聚到 IsInternalAccount(ctx)。
func isInternalAccountUin(uin string) bool {
	_, ok := internalAccountUins[uin]
	return ok
}

// CAM GetUserAppId 接口元信息。该接口与地域无关，但 SDK 仍要求传入非空 region 占位。
const (
	camService            = "cam"
	camVersion            = "2019-01-16"
	camAction             = "GetUserAppId"
	camRegion             = "ap-guangzhou"
	camCallTimeoutSeconds = 5
)

// CAM 调用的成功结果进程内缓存：UIN 一旦确定就是部署级常量，只查一次即可。
// camUinMu 同时守护 camUinCache 和 camUinOnce 的读写，确保失败重置 once
// 与首次并发调用之间没有 data race。
var (
	camUinMu    sync.Mutex
	camUinCache string
	camUinOnce  sync.Once
)

// cloudUinFetcher 为真实 CAM 调用函数引用，定义为包级变量便于单测注入替换。
var cloudUinFetcher = fetchCloudUinViaCAM

// camAKSKProvider 返回调用 CAM 所需的永久 AKSK。
// 生产环境从 site_configs 表读，测试时可替换为固定值，避免依赖 DB。
var camAKSKProvider = func(ctx context.Context) (secretID, secretKey string) {
	cfg := model.GetSiteConfig(ctx)
	return cfg.CVMSecretId, cfg.CVMSecretKey
}

// camEndpointOverride 允许测试把 CAM 域名指向 httptest 服务器；空字符串表示走线上。
var camEndpointOverride string

// IsInternalAccount 判断当前部署是否运行在内部测试账号下。
//
// 判断顺序：
//  1. 从 ctx 的 TenantSnapshot 中读取 UIN；
//  2. 上一步拿不到时，调用 CAM GetUserAppId 接口拉取（结果进程内缓存）。
//
// 返回 (false, err) 表示判断失败（CAM 调用异常等），调用方一般按"非内部"降级处理。
func IsInternalAccount(ctx context.Context) (bool, error) {
	uin, err := ResolveCloudUin(ctx)
	if err != nil {
		return false, err
	}
	if uin == "" {
		return false, nil
	}
	return isInternalAccountUin(uin), nil
}

// ResolveCloudUin 按"ctx → CAM"的顺序解析当前部署所属的腾讯云主账号 UIN。
// 与 IsInternalAccount 拆开导出，方便其他需要拿到 UIN 本身的调用方复用。
func ResolveCloudUin(ctx context.Context) (string, error) {
	if uin := hcommon.CVMUinFromCtx(ctx); uin != "" {
		return uin, nil
	}
	if cloudUinFetcher == nil {
		return "", errors.New("cloud uin fetcher is not configured")
	}
	return cloudUinFetcher(ctx)
}

// ResetCAMUinCacheForTest 清空 CAM UIN 进程内缓存。仅供测试调用。
func ResetCAMUinCacheForTest() {
	camUinMu.Lock()
	defer camUinMu.Unlock()
	camUinCache = ""
	camUinOnce = sync.Once{}
}

// fetchCloudUinViaCAM 调用 CAM GetUserAppId 拿到主账号 UIN，成功结果进程内缓存。
// 失败时不进入缓存，调用方下次再调会重新尝试。
//
// 并发模型：
//   - 整个函数串行化在 camUinMu 之下，保证 camUinCache 与 camUinOnce 的读写一致；
//   - 命中缓存的 fast path 也走锁（开销极低，只是一次 mutex 抢占），
//     换来对 -race 完全干净的语义。
func fetchCloudUinViaCAM(ctx context.Context) (string, error) {
	camUinMu.Lock()
	defer camUinMu.Unlock()

	if camUinCache != "" {
		return camUinCache, nil
	}
	var (
		uin string
		err error
	)
	camUinOnce.Do(func() {
		uin, err = doCallCAMGetUserAppId(ctx)
		if err == nil {
			camUinCache = uin
		}
	})
	if err != nil {
		// once 已被消费但未缓存成功值，重置以允许下次重试。
		camUinOnce = sync.Once{}
		return "", err
	}
	if camUinCache != "" {
		return camUinCache, nil
	}
	return uin, nil
}

// camGetUserAppIdResp 仅取主账号 UIN 字段；官方文档明确为 String 类型。
type camGetUserAppIdResp struct {
	Response struct {
		OwnerUin  string `json:"OwnerUin"`
		Uin       string `json:"Uin"`
		RequestId string `json:"RequestId"`
	} `json:"Response"`
}

// doCallCAMGetUserAppId 真正向腾讯云 CAM 发起请求并解出主账号 UIN。
//
// 注意：CAM 必须用永久 AKSK 调用，复用 site_configs 表里的 CVMSecretId/CVMSecretKey。
// 业务错误已由 SDK 的 client.Send 包装为 error 返回，无需手工解析 Response.Error。
func doCallCAMGetUserAppId(ctx context.Context) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	secretID, secretKey := camAKSKProvider(ctx)
	if secretID == "" || secretKey == "" {
		return "", errors.New("CAM GetUserAppId 调用失败：永久 AKSK 未配置")
	}

	credential := common.NewCredential(secretID, secretKey)
	cpf := profile.NewClientProfile()
	endpoint := camService + ".tencentcloudapi.com"
	if camEndpointOverride != "" {
		endpoint = camEndpointOverride
		cpf.HttpProfile.Scheme = "HTTP"
	}
	cpf.HttpProfile.Endpoint = endpoint
	cpf.HttpProfile.ReqTimeout = camCallTimeoutSeconds
	client := common.NewCommonClient(credential, camRegion, cpf)

	req := tchttp.NewCommonRequest(camService, camVersion, camAction)
	if err := req.SetActionParameters(map[string]interface{}{}); err != nil {
		return "", fmt.Errorf("CAM GetUserAppId 设置参数失败: %w", err)
	}

	callCtx, cancel := context.WithTimeout(ctx, camCallTimeoutSeconds*time.Second)
	defer cancel()
	req.SetContext(callCtx)

	resp := tchttp.NewCommonResponse()
	if err := client.Send(req, resp); err != nil {
		return "", fmt.Errorf("CAM GetUserAppId 调用失败: %w", err)
	}

	var parsed camGetUserAppIdResp
	if err := json.Unmarshal(resp.GetBody(), &parsed); err != nil {
		return "", fmt.Errorf("CAM GetUserAppId 响应解析失败: %w", err)
	}
	if parsed.Response.OwnerUin != "" {
		return parsed.Response.OwnerUin, nil
	}
	if parsed.Response.Uin != "" {
		return parsed.Response.Uin, nil
	}
	return "", errors.New("CAM GetUserAppId 响应缺少 OwnerUin/Uin 字段")
}
