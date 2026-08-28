package controller

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	client "cnb.cool/tencent/cloud/smh/smh-go-sdk"
	"cnb.cool/tencent/cloud/smh/smh-go-sdk/transfer"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// smhRequestId 从 SMH HTTP 响应中提取 X-Request-Id。
func smhRequestId(resp *http.Response) string {
	if resp == nil {
		return ""
	}
	return resp.Header.Get("X-Request-Id")
}

// StorageClient 定义文件存储操作接口。
type StorageClient interface {
	Upload(key string, data []byte, contentType string) error
	Delete(key string, permanent bool) error
	DeletePrefix(prefix string, permanent bool) error
	List(prefix string) ([]string, error)
}

// ── SMH Token 管理（DB 持久化，无进程内缓存）─────────────────────────
//
// 设计原则：
//   - Token 持久化在 smh_spaces 表（admin_token/read_token），参考 STS 凭证模式
//   - smh-token-refresh 定时任务负责每 12h 刷新并写 DB
//   - controller 每次需要 token 时直接读 DB；若过期或为空则按需自愈
//   - 不做进程内缓存，避免多副本/多租户/task-controller 拆分部署场景下的一致性问题

// InitSMHTokenRefresher 为指定租户刷新所有 SMH Space 的 Token 并持久化到 DB。
// 由 task/smh_token_refresh.go 定时调用（PerTenant=true，每 12h 执行一次）。
func InitSMHTokenRefresher(ctx context.Context, smhConfig model.SMHConfig) {
	apiClient := newSMHAPIClient(smhConfig.Endpoint)

	for _, spaceTag := range []string{"common", "skillhub"} {
		spaceID := smhConfig.CommonSpace
		if spaceTag == "skillhub" {
			spaceID = smhConfig.SkillhubSpace
		}
		if spaceID == "" {
			continue
		}
		// 刷新 admin token 并写 DB
		if token, err := createOrRenewSpaceToken(ctx, apiClient, smhConfig, spaceTag, true); err != nil {
			slog.Error("SMH 刷新 admin Token 失败", "space_tag", spaceTag, "error", err)
		} else if err := model.UpdateSMHSpaceToken(ctx, spaceTag, true, token, time.Now().Add(24*time.Hour).Unix()); err != nil {
			slog.Error("SMH 持久化 admin Token 失败", "space_tag", spaceTag, "error", err)
		}
		// 刷新 read token 并写 DB
		if token, err := createOrRenewSpaceToken(ctx, apiClient, smhConfig, spaceTag, false); err != nil {
			slog.Error("SMH 刷新 read Token 失败", "space_tag", spaceTag, "error", err)
		} else if err := model.UpdateSMHSpaceToken(ctx, spaceTag, false, token, time.Now().Add(24*time.Hour).Unix()); err != nil {
			slog.Error("SMH 持久化 read Token 失败", "space_tag", spaceTag, "error", err)
		}
	}

	slog.Info("SMH Token 刷新完成",
		"common_space", smhConfig.CommonSpace,
		"skillhub_space", smhConfig.SkillhubSpace)
}

// newSMHAPIClient 创建 SMH SDK Client（无状态，每次新建）。
func newSMHAPIClient(endpoint string) *client.APIClient {
	cfg := client.NewConfiguration()
	cfg.Servers = client.ServerConfigurations{
		{URL: endpoint, Description: "SMH API Server"},
	}
	cfg.HTTPClient = &http.Client{
		Transport: &smhInternalDomainTransport{base: http.DefaultTransport},
	}
	return client.NewAPIClient(cfg)
}

// newSMHClientConfig 创建 SMH SDK Client + Configuration（Upload 等操作需要 cfg）。
func newSMHClientConfig(ctx context.Context) (*client.APIClient, *client.Configuration, error) {
	smhConfig := model.GetSMHConfig(ctx)
	if !smhConfig.IsConfigured() {
		return nil, nil, hcommon.I18nError(i18n.MsgSmhClientNotInitialized)
	}
	cfg := client.NewConfiguration()
	cfg.Servers = client.ServerConfigurations{
		{URL: smhConfig.Endpoint, Description: "SMH API Server"},
	}
	cfg.HTTPClient = &http.Client{
		Transport: &smhInternalDomainTransport{base: http.DefaultTransport},
	}
	return client.NewAPIClient(cfg), cfg, nil
}

// createOrRenewSpaceToken 为指定 Space 创建新 Token（尝试续期已有 token，失败则重新创建）。
// admin=true 创建 space_admin token，false 创建只读 token。
func createOrRenewSpaceToken(ctx context.Context, apiClient *client.APIClient, smhConfig model.SMHConfig, spaceTag string, admin bool) (string, error) {
	spaceID := smhConfig.CommonSpace
	if spaceTag == "skillhub" {
		spaceID = smhConfig.SkillhubSpace
	}

	// 读取 DB 中现有 token，尝试续期
	space, found := model.GetSMHSpaceRecord(ctx, spaceTag)
	if found {
		currentToken := space.AdminToken
		expiredAt := space.AdminTokenExpiredAt
		if !admin {
			currentToken = space.ReadToken
			expiredAt = space.ReadTokenExpiredAt
		}
		if currentToken != "" && time.Now().Unix() < expiredAt {
			resp, httpResp, err := apiClient.TokenAPI.RenewToken(ctx, smhConfig.LibraryId, currentToken).Execute()
			if httpResp != nil && httpResp.Body != nil {
				httpResp.Body.Close()
			}
			if err == nil && resp != nil && resp.AccessToken != nil {
				slog.Info("SMH Token 续期成功", "space_tag", spaceTag, "admin", admin, "request_id", smhRequestId(httpResp))
				return *resp.AccessToken, nil
			}
			slog.Warn("SMH Token 续期失败，重新创建", "space_tag", spaceTag, "admin", admin, "error", err)
		}
	}

	// 创建新 Token
	req := apiClient.TokenAPI.CreateToken(ctx).
		LibraryId(smhConfig.LibraryId).
		LibrarySecret(smhConfig.LibrarySecret).
		SpaceId(spaceID).
		Period(86400)
	if admin {
		req = req.Grant("space_admin")
	}
	tokenResp, httpResp, err := req.Execute()
	if httpResp != nil && httpResp.Body != nil {
		httpResp.Body.Close()
	}
	if err != nil {
		return "", hcommon.I18nRichError(err, i18n.MsgSmhCreateTokenFailed, spaceID, smhRequestId(httpResp))
	}
	if tokenResp == nil || tokenResp.AccessToken == nil {
		return "", hcommon.I18nError(i18n.MsgSmhCreateTokenEmpty, spaceID, smhRequestId(httpResp))
	}
	slog.Info("SMH Token 创建成功", "space_tag", spaceTag, "admin", admin, "request_id", smhRequestId(httpResp))
	return *tokenResp.AccessToken, nil
}

// getSpaceToken 从 DB 读取指定 Space 的 Token，过期或为空时按需自愈（CreateToken + 写回 DB）。
// spaceTag: "common" 或 "skillhub"；admin: true=读写 token, false=只读 token。
func getSpaceToken(ctx context.Context, spaceTag string, admin bool) (string, error) {
	space, found := model.GetSMHSpaceRecord(ctx, spaceTag)
	if !found {
		return "", hcommon.I18nError(i18n.MsgSmhSpaceTokenNotInitialized)
	}
	token := space.AdminToken
	expiredAt := space.AdminTokenExpiredAt
	if !admin {
		token = space.ReadToken
		expiredAt = space.ReadTokenExpiredAt
	}
	// token 有效则直接返回，提前 5 分钟视为过期以留出刷新余量
	if token != "" && time.Now().Unix() < expiredAt-300 {
		return token, nil
	}
	// 按需自愈
	slog.Info("SMH Token 为空或已过期，按需刷新", "space_tag", spaceTag, "admin", admin)
	smhConfig := model.GetSMHConfig(ctx)
	if !smhConfig.IsConfigured() {
		return "", hcommon.I18nError(i18n.MsgSmhNotConfigured)
	}
	apiClient := newSMHAPIClient(smhConfig.Endpoint)
	newToken, err := createOrRenewSpaceToken(ctx, apiClient, smhConfig, spaceTag, admin)
	if err != nil {
		return "", hcommon.I18nRichError(err, i18n.MsgSmhTokenRefreshFailed, space.SpaceId)
	}
	if dbErr := model.UpdateSMHSpaceToken(ctx, spaceTag, admin, newToken, time.Now().Add(24*time.Hour).Unix()); dbErr != nil {
		slog.Warn("SMH Token 自愈写回 DB 失败", "space_tag", spaceTag, "admin", admin, "error", dbErr)
	}
	return newToken, nil
}

// GetCommonSpaceToken 获取 common 空间的 admin Token（读写，供内部使用）。
func GetCommonSpaceToken(ctx context.Context) (string, error) {
	return getSpaceToken(ctx, "common", true)
}

// GetCommonSpaceReadToken 获取 common 空间的只读 Token。
func GetCommonSpaceReadToken(ctx context.Context) (string, error) {
	return getSpaceToken(ctx, "common", false)
}

// GetSkillhubSpaceReadToken 获取 skillhub 空间的只读 Token。
func GetSkillhubSpaceReadToken(ctx context.Context) (string, error) {
	return getSpaceToken(ctx, "skillhub", false)
}

// ── SMH StorageClient 实现 ─────────────────────────────────────────

// dirOnce 封装目录创建的一次性执行结果。
type dirOnce struct {
	once sync.Once
	err  error
}

type smhClient struct {
	ctx         context.Context
	libraryID   string
	spaceID     string
	spaceTag    string   // "common" 或 "skillhub"，用于从 DB 读取 token
	createdDirs sync.Map // map[string]*dirOnce，按目录粒度确保只创建一次
}

// isRateLimitError 检测 SMH SDK 返回的 429 限流错误。
// SMH 单 space QPS 100/s，多端并发可能触发限流。
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "429")
}

// transferUpload 是 transfer.UploadFileFromReader 的函数变量，方便测试 mock。
var transferUpload = transfer.UploadFileFromReader

func (s *smhClient) Upload(key string, data []byte, contentType string) error {
	token, err := getSpaceToken(s.ctx, s.spaceTag, true)
	if err != nil {
		return err
	}

	apiClient, cfg, err := newSMHClientConfig(s.ctx)
	if err != nil {
		return err
	}

	ctx := s.ctx
	filePath := "/" + key

	// 递归创建所有祖先目录（SMH 不会自动创建目录）
	if err := s.ensureParentDirs(ctx, apiClient, token, filePath); err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSmhCreateParentDirFailed)
	}

	// 指数退避参数
	const (
		baseDelay  = 1 * time.Second
		maxDelay   = 30 * time.Second
		maxRetries = 5
	)

	file := transfer.ReaderFileOptions{
		Name: key[strings.LastIndex(key, "/")+1:],
		Size: int64(len(data)),
	}
	opts := &transfer.UploadOptions{
		LibraryID:   s.libraryID,
		SpaceID:     s.spaceID,
		FilePath:    filePath,
		AccessToken: token,
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// 指数退避：1s → 2s → 4s → 8s → 16s（封顶 30s）
			delay := baseDelay * (1 << (attempt - 1))
			if delay > maxDelay {
				delay = maxDelay
			}
			// ±25% 随机抖动，避免惊群
			jitter := time.Duration(float64(delay) * 0.25 * (rand.Float64()*2 - 1))
			delay += jitter
			slog.Warn("SMH 上传命中限流，退避重试",
				"key", key,
				"attempt", attempt,
				"max_retries", maxRetries,
				"delay_ms", delay/time.Millisecond,
			)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		// 每次重试需要新的 Reader（Reader 为一次性消耗品）
		file.Reader = bytes.NewReader(data)
		_, lastErr = transferUpload(ctx, file, opts, cfg)
		if lastErr == nil {
			return nil
		}
		if !isRateLimitError(lastErr) {
			return lastErr
		}
	}
	slog.Error("SMH 上传重试耗尽",
		"key", key,
		"max_retries", maxRetries,
		"error", lastErr,
	)
	return lastErr
}

// ensureParentDirs 确保文件的父目录存在。
// SMH CreateDirectory 支持递归创建（类似 mkdir -p），传入 /a/b/c 会自动创建整个目录链。
// 因此只需对直接父目录调用一次，配合 createdDirs 缓存避免重复请求。
func (s *smhClient) ensureParentDirs(ctx context.Context, apiClient *client.APIClient, token string, filePath string) error {
	dir := path.Dir(filePath)
	if dir == "/" || dir == "." {
		return nil
	}

	// 使用 sync.Once 确保同一目录只创建一次，不同目录之间互不阻塞
	val, _ := s.createdDirs.LoadOrStore(dir, &dirOnce{})
	d := val.(*dirOnce)
	d.once.Do(func() {
		_, httpResp, err := apiClient.DirectoryAPI.CreateDirectory(ctx, s.libraryID, s.spaceID, dir).
			AccessToken(token).
			ConflictResolutionStrategy("ask").
			Execute()
		if httpResp != nil && httpResp.Body != nil {
			httpResp.Body.Close()
		}
		if err != nil && !strings.Contains(err.Error(), "409") {
			d.err = hcommon.I18nRichError(err, i18n.MsgSmhCreateDirFailed, dir, smhRequestId(httpResp))
			// 创建失败时移除缓存，允许后续重试
			s.createdDirs.Delete(dir)
			return
		}
		// 成功或 409 冲突时，同时标记所有祖先目录为已创建
		for ancestor := path.Dir(dir); ancestor != "/" && ancestor != "."; ancestor = path.Dir(ancestor) {
			s.createdDirs.LoadOrStore(ancestor, &dirOnce{})
		}
	})
	return d.err
}

// Delete deletes a file. When permanent is true, the file is permanently deleted;
// when false, it is moved to the recycle bin.
func (s *smhClient) Delete(key string, permanent bool) error {
	token, err := getSpaceToken(s.ctx, s.spaceTag, true)
	if err != nil {
		return err
	}

	apiClient, _, err := newSMHClientConfig(s.ctx)
	if err != nil {
		return err
	}

	ctx := s.ctx
	req := apiClient.FileAPI.DeleteFile(ctx, s.libraryID, s.spaceID, "/"+key).
		AccessToken(token)
	if permanent {
		req = req.Permanent(1)
	}
	_, httpResp, err := req.Execute()
	if httpResp != nil && httpResp.Body != nil {
		httpResp.Body.Close()
	}
	if err != nil {
		slog.Warn("SMH 删除文件失败", "key", key, "permanent", permanent, "request_id", smhRequestId(httpResp), "error", err)
	}
	return err
}

// DeletePrefix deletes a directory by prefix. When permanent is true, the directory
// is permanently deleted; when false, it is moved to the recycle bin.
func (s *smhClient) DeletePrefix(prefix string, permanent bool) error {
	token, err := getSpaceToken(s.ctx, s.spaceTag, true)
	if err != nil {
		return err
	}

	apiClient, _, err := newSMHClientConfig(s.ctx)
	if err != nil {
		return err
	}

	ctx := s.ctx
	req := apiClient.DirectoryAPI.DeleteDirectory(ctx, s.libraryID, s.spaceID, "/"+strings.TrimSuffix(prefix, "/")).
		AccessToken(token)
	if permanent {
		req = req.Permanent(1)
	}
	_, httpResp, err := req.Execute()
	if httpResp != nil && httpResp.Body != nil {
		httpResp.Body.Close()
	}
	if err != nil {
		slog.Warn("SMH 删除目录失败", "prefix", prefix, "permanent", permanent, "request_id", smhRequestId(httpResp), "error", err)
	}

	return err
}

func (s *smhClient) List(prefix string) ([]string, error) {
	token, err := getSpaceToken(s.ctx, s.spaceTag, true)
	if err != nil {
		return nil, err
	}

	apiClient, _, err := newSMHClientConfig(s.ctx)
	if err != nil {
		return nil, err
	}

	ctx := s.ctx
	// SMH ListDirectory 为分页接口，ByMarker 为必填参数。
	// 循环翻页拉取全量目录项。
	var files []string
	marker := ""
	const pageLimit int32 = 200
	dirPath := "/" + strings.TrimSuffix(prefix, "/")
	for {
		resp, httpResp, err := apiClient.DirectoryAPI.ListDirectory(ctx, s.libraryID, s.spaceID, dirPath).
			AccessToken(token).
			ByMarker(1).
			Marker(marker).
			Limit(pageLimit).
			Execute()
		if httpResp != nil && httpResp.Body != nil {
			httpResp.Body.Close()
		}
		if err != nil {
			slog.Warn("SMH 列举目录失败", "prefix", prefix, "request_id", smhRequestId(httpResp), "error", err)
			return nil, err
		}
		if resp != nil {
			for _, item := range resp.GetContents() {
				files = append(files, item.GetName())
			}
			next := resp.GetNextMarker()
			if next == "" || next == marker {
				break
			}
			marker = next
			continue
		}
		break
	}
	return files, nil
}

// ── 公共入口函数 ───────────────────────────────────────────────────

// getStorageClient 返回 skillhub 空间的存储客户端。
// 每次调用创建新实例，createdDirs 缓存仅在单次上传生命周期内有效，避免缓存中毒。
func getStorageClient(ctx context.Context) (StorageClient, error) {
	smhConfig := model.GetSMHConfig(ctx)
	if !smhConfig.IsConfigured() {
		return nil, hcommon.I18nError(i18n.MsgSmhNotConfiguredCheckLog)
	}
	if smhConfig.SkillhubSpace == "" {
		return nil, hcommon.I18nError(i18n.MsgSmhSkillhubSpaceNotConfiguredCheckLog)
	}
	return &smhClient{
		ctx:       ctx,
		libraryID: smhConfig.LibraryId,
		spaceID:   smhConfig.SkillhubSpace,
		spaceTag:  "skillhub",
	}, nil
}

// uploadToCOS 上传文件到存储（保留函数名以兼容现有调用方）
func uploadToCOS(ctx context.Context, key string, data []byte, contentType string) error {
	c, err := getStorageClient(ctx)
	if err != nil {
		return err
	}
	return c.Upload(key, data, contentType)
}

// deleteCOSPrefix 按前缀批量删除存储对象（保留函数名以兼容现有调用方）
func deleteCOSPrefix(ctx context.Context, prefix string) error {
	c, err := getStorageClient(ctx)
	if err != nil {
		return err
	}
	return c.DeletePrefix(prefix, true)
}

// buildSMHDownloadURL 生成 SMH 文件下载 URL，供 TAT 脚本 curl 使用。
// URL 格式：{endpoint}/api/v1/file/{libraryId}/{spaceId}/{filePath}?access_token=xxx
// 注意：SMH 下载接口返回 302 重定向，curl 需要 -L 跟随。
// 使用只读 Token，遵循最小权限原则。
// internalDomain 为 true 时使用内网域名（适用于 CVM 等内网环境），为 false 时使用公网域名。
func buildSMHDownloadURL(ctx context.Context, fileKey string, internalDomain bool) (string, error) {
	smhConfig := model.GetSMHConfig(ctx)
	if !smhConfig.IsConfigured() {
		return "", hcommon.I18nError(i18n.MsgSmhNotConfigured)
	}
	if smhConfig.SkillhubSpace == "" {
		return "", hcommon.I18nError(i18n.MsgSmhSkillhubSpaceNotConfiguredCheckLog)
	}
	token, err := GetSkillhubSpaceReadToken(ctx)
	if err != nil {
		return "", err
	}
	encodedFileKey := encodeFileKeyPath(fileKey)
	downloadURL := fmt.Sprintf("%s/api/v1/file/%s/%s/%s?access_token=%s",
		strings.TrimSuffix(smhConfig.Endpoint, "/"),
		smhConfig.LibraryId,
		smhConfig.SkillhubSpace,
		encodedFileKey,
		token,
	)
	if internalDomain {
		downloadURL += "&internal_domain=1"
	}
	return downloadURL, nil
}

// ── Common Space 存储客户端 ────────────────────────────────────────

// GetCommonStorageClient 返回 common 空间的存储客户端。
// 每次调用创建新实例，createdDirs 缓存仅在单次上传生命周期内有效，避免缓存中毒。
func GetCommonStorageClient(ctx context.Context) (StorageClient, error) {
	smhConfig := model.GetSMHConfig(ctx)
	if !smhConfig.IsConfigured() {
		return nil, hcommon.I18nError(i18n.MsgSmhNotConfigured)
	}
	if smhConfig.CommonSpace == "" {
		return nil, hcommon.I18nError(i18n.MsgSmhCommonSpaceNotConfigured)
	}
	return &smhClient{
		ctx:       ctx,
		libraryID: smhConfig.LibraryId,
		spaceID:   smhConfig.CommonSpace,
		spaceTag:  "common",
	}, nil
}

// BuildCommonSMHDownloadURL 生成 common space 的文件下载 URL。
// 用于 installSkillsAsync 中为 CVM 生成技能包 zip 的内网下载链接。
func BuildCommonSMHDownloadURL(ctx context.Context, fileKey string, internalDomain bool) (string, error) {
	smhConfig := model.GetSMHConfig(ctx)
	if !smhConfig.IsConfigured() {
		return "", hcommon.I18nError(i18n.MsgSmhNotConfigured)
	}
	if smhConfig.CommonSpace == "" {
		return "", hcommon.I18nError(i18n.MsgSmhCommonSpaceNotConfigured)
	}
	token, err := GetCommonSpaceReadToken(ctx)
	if err != nil {
		return "", err
	}
	// 对 fileKey 的每段路径做 URL 编码（路径中可能含中文，如"通用技能包"）
	encodedFileKey := encodeFileKeyPath(fileKey)
	downloadURL := fmt.Sprintf("%s/api/v1/file/%s/%s/%s?access_token=%s",
		strings.TrimSuffix(smhConfig.Endpoint, "/"),
		smhConfig.LibraryId,
		smhConfig.CommonSpace,
		encodedFileKey,
		token,
	)
	if internalDomain {
		downloadURL += "&internal_domain=1"
	}
	return downloadURL, nil
}

// getSMHCommonSpaceAvailable 查询 common 空间的可用容量（字节）。
// 返回 -1 表示无配额限制（无限空间）。
func getSMHCommonSpaceAvailable(ctx context.Context) (int64, error) {
	smhConfig := model.GetSMHConfig(ctx)
	if !smhConfig.IsConfigured() {
		return 0, hcommon.I18nError(i18n.MsgSmhNotConfigured)
	}
	if smhConfig.CommonSpace == "" {
		return 0, hcommon.I18nError(i18n.MsgSmhCommonSpaceNotConfigured)
	}

	apiClient, _, err := newSMHClientConfig(ctx)
	if err != nil {
		return 0, hcommon.I18nError(i18n.MsgSmhClientNotInitialized)
	}

	accessToken, err := getSpaceToken(ctx, "common", true)
	if err != nil {
		return 0, err
	}

	resp, httpResp, err := apiClient.UsageAPI.GetUsage(ctx, smhConfig.LibraryId, smhConfig.CommonSpace).
		AccessToken(accessToken).
		Execute()
	if httpResp != nil && httpResp.Body != nil {
		httpResp.Body.Close()
	}
	if err != nil {
		return 0, hcommon.I18nRichError(err, i18n.MsgSmhQueryUsageFailed, smhRequestId(httpResp))
	}
	if resp == nil || len(*resp) == 0 {
		return 0, hcommon.I18nError(i18n.MsgSmhUsageQueryEmpty)
	}

	usage := (*resp)[0]
	// AvailableSpace 为 null 表示无配额限制
	if usage.AvailableSpace.IsSet() && usage.AvailableSpace.Get() != nil {
		avail, err := strconv.ParseInt(*usage.AvailableSpace.Get(), 10, 64)
		if err != nil {
			return 0, hcommon.I18nRichError(err, i18n.MsgSmhParseAvailableSpaceFailed)
		}
		return avail, nil
	}
	// 无配额限制，返回 -1
	return -1, nil
}

// smhMultipartPartSize 分块上传每块大小：50MB。
const smhMultipartPartSize = 50 * 1024 * 1024 // 50MB

// SMHUploadCredential 包含 CVM 侧分块上传备份包所需的凭证信息。
type SMHUploadCredential struct {
	PartURLTemplate string            // 分块 PUT URL 模板，{partNumber} 为占位符，例如 https://{domain}{path}?partNumber={partNumber}&uploadId={uploadId}
	PartHeaders     map[string]string // SMH 返回的分块 PUT 请求所需额外 Header（如鉴权 Header）
	ConfirmKey      string            // 上传完成后调用 ConfirmUpload 所需的 key
	FileKey         string            // SMH 文件 key，用于生成下载 URL
	PartSize        int64             // 每个分块的字节数（最后一块可能更小）
	TotalParts      int               // 总分块数
	Expiration      *time.Time        // SMH 返回的上传凭证有效期，超过后 PART_URL_TEMPLATE / ConfirmKey 均失效；秒传时为 nil

	// UsedInternalDomain 标记本次凭证是否走的内网域名（internal_domain=1）。
	// Renew 时需要据此保持一致：内网凭证续期仍走内网，外网凭证续期仍走外网，
	// 避免续期后 Domain 突变导致正在进行的分块上传方向被打断。
	UsedInternalDomain bool

	// 秒传时以下字段均为空/零值，PartURLTemplate 为空表示秒传成功
}

// PrepareSMHCommonUpload 调用 SMH MultipartUploadFile API，初始化分块上传，获取 CVM 侧上传备份包所需的凭证。
// archivePath：CVM 上的压缩包路径（仅用于提取文件名构造 fileKey）；archiveSize：字节数。
// 返回 SMHUploadCredential，CVM 侧按 PartSize 分块，依次 PUT 各分块 URL，完成后调用 ConfirmSMHCommonUpload。
//
// 内网/外网选择：若 ctx 通过 WithSMHInternalDomain(ctx, true) 标记了使用内网，则本次 API 调用
// 会在 query 上追加 internal_domain=1，SMH 返回的 Domain 为内网 COS 接入域。调用方拿到 cred 后
// 应该做一次 hatchery 侧的 TCP 连通性探测；探测失败则改用 WithSMHInternalDomain(ctx, false)
// 重新调用本函数获取外网凭证作为降级。
func PrepareSMHCommonUpload(ctx context.Context, instanceId string, archivePath string, archiveSize int64) (*SMHUploadCredential, error) {
	log := Logger(ctx)
	smhConfig := model.GetSMHConfig(ctx)
	// 只检查上传备份所需的字段（不要求 SkillhubSpace 也配置）
	if smhConfig.Endpoint == "" || smhConfig.LibraryId == "" || smhConfig.LibrarySecret == "" {
		return nil, hcommon.I18nError(i18n.MsgSmhNotConfiguredCannotUpload)
	}
	if smhConfig.CommonSpace == "" {
		return nil, hcommon.I18nError(i18n.MsgSmhCommonSpaceNotConfigured)
	}

	// 检查 common 空间剩余容量
	available, err := getSMHCommonSpaceAvailable(ctx)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgSmhCheckCapacityFailed)
	}
	if archiveSize <= 0 {
		return nil, hcommon.I18nError(i18n.MsgSmhInvalidArchiveSize)
	}
	if available >= 0 && archiveSize > available {
		return nil, hcommon.I18nError(i18n.MsgSmhInsufficientSpace, archiveSize, available)
	}
	if available >= 0 {
		log.Info("SMH common 空间容量检查通过", "archiveSize", archiveSize, "available", available)
	} else {
		log.Info("SMH common 空间无配额限制，跳过容量检查", "archiveSize", archiveSize)
	}

	// 构造 SMH 文件 key：backups/{instanceId}/{filename}
	filename := archivePath[strings.LastIndex(archivePath, "/")+1:]
	fileKey := fmt.Sprintf("backups/%s/%s", instanceId, filename)

	apiClient, _, err := newSMHClientConfig(ctx)
	if err != nil {
		return nil, hcommon.I18nError(i18n.MsgSmhClientNotInitialized)
	}
	accessToken, err := getSpaceToken(ctx, "common", true)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgSmhGetTokenFailed)
	}

	// 计算分块数
	partSize := int64(smhMultipartPartSize)
	totalParts := int((archiveSize + partSize - 1) / partSize)

	smhFilePath := "/" + fileKey

	// 确保父目录存在（SMH 不会自动创建目录，409 冲突表示已存在，忽略）
	parentDir := path.Dir(smhFilePath)
	if parentDir != "/" && parentDir != "." {
		log.Info("SMH 创建父目录", "dir", parentDir)
		_, httpRespDir, errDir := apiClient.DirectoryAPI.
			CreateDirectory(ctx, smhConfig.LibraryId, smhConfig.CommonSpace, parentDir).
			AccessToken(accessToken).
			ConflictResolutionStrategy("ask").
			Execute()
		if httpRespDir != nil && httpRespDir.Body != nil {
			httpRespDir.Body.Close()
		}
		if errDir != nil && !strings.Contains(errDir.Error(), "409") {
			return nil, hcommon.I18nRichError(errDir, i18n.MsgSmhCreateDirFailed, parentDir, smhRequestId(httpRespDir))
		}
	}

	log.Info("SMH MultipartUploadFile 请求参数",
		"endpoint", smhConfig.Endpoint,
		"libraryId", smhConfig.LibraryId,
		"spaceId", smhConfig.CommonSpace,
		"filePath", smhFilePath,
		"archiveSize", archiveSize,
		"partSize", partSize,
		"totalParts", totalParts,
	)
	resp201, resp200, httpResp, err := apiClient.FileAPI.
		MultipartUploadFile(ctx, smhConfig.LibraryId, smhConfig.CommonSpace, smhFilePath).
		Multipart(1).
		AccessToken(accessToken).
		Filesize(archiveSize).
		Execute()
	if httpResp != nil && httpResp.Body != nil {
		httpResp.Body.Close()
	}
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgSmhMultipartUploadFailed, smhRequestId(httpResp))
	}

	// 200 表示秒传成功（相同内容已存在），无需再上传
	if resp200 != nil {
		log.Info("SMH 秒传成功（内容已存在）", "fileKey", fileKey)
		return &SMHUploadCredential{FileKey: fileKey}, nil
	}

	if resp201 == nil || resp201.Domain == nil || resp201.Path == nil || resp201.ConfirmKey == nil || resp201.UploadId == nil {
		return nil, hcommon.I18nError(i18n.MsgSmhMultipartUploadIncomplete)
	}

	// 分块 PUT URL 模板：{partNumber} 由 CVM 侧脚本替换为实际分块序号（从 1 开始）
	// 鉴权信息通过 resp201.Headers 传递，不放在 URL 中
	partURLTemplate := fmt.Sprintf("https://%s%s?partNumber={partNumber}&uploadId=%s",
		*resp201.Domain, *resp201.Path, *resp201.UploadId)

	// 提取 SMH 返回的分块 PUT 所需额外 Header（如鉴权 Header）
	var partHeaders map[string]string
	if resp201.Headers != nil {
		partHeaders = *resp201.Headers
	}

	expiration := resp201.Expiration

	log.Info("SMH 分块上传凭证已获取",
		"fileKey", fileKey,
		"uploadId", *resp201.UploadId,
		"totalParts", totalParts,
		"partSize", partSize,
		"internal_domain", smhInternalDomainEnabled(ctx),
		"domain", *resp201.Domain,
	)
	return &SMHUploadCredential{
		PartURLTemplate:    partURLTemplate,
		PartHeaders:        partHeaders,
		ConfirmKey:         *resp201.ConfirmKey,
		FileKey:            fileKey,
		PartSize:           partSize,
		TotalParts:         totalParts,
		Expiration:         expiration,
		UsedInternalDomain: smhInternalDomainEnabled(ctx),
	}, nil
}

// RenewSMHCommonUpload 续期分块上传凭证，原地刷新 cred 中的 PartURLTemplate / PartHeaders / Expiration。
//
// 适用场景：备份包很大、分块数较多导致上传耗时长，初次 MultipartUploadFile 返回的
// PartURLTemplate / PartHeaders 在 cred.Expiration 之后会失效。每隔一段时间或检测到剩余
// 有效期不足时调用本函数即可继续使用 cred.ConfirmKey 上传剩余分块。
//
// 注意：续期成功后 PartHeaders 中的鉴权 Header 会变化，调用方必须使用新值重新构造 PUT 请求。
func RenewSMHCommonUpload(ctx context.Context, cred *SMHUploadCredential) error {
	log := Logger(ctx)
	if cred == nil {
		return hcommon.I18nError(i18n.MsgSmhCredCannotBeEmpty)
	}
	if cred.ConfirmKey == "" {
		return hcommon.I18nError(i18n.MsgSmhConfirmKeyEmptyCannotRenew)
	}

	smhConfig := model.GetSMHConfig(ctx)
	if !smhConfig.IsConfigured() {
		return hcommon.I18nError(i18n.MsgSmhNotConfigured)
	}
	if smhConfig.CommonSpace == "" {
		return hcommon.I18nError(i18n.MsgSmhCommonSpaceNotConfigured)
	}

	apiClient, _, err := newSMHClientConfig(ctx)
	if err != nil {
		return hcommon.I18nError(i18n.MsgSmhClientNotInitialized)
	}
	accessToken, err := getSpaceToken(ctx, "common", true)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSmhGetTokenFailed)
	}

	// 续期时保持与初次 Prepare 一致的内外网选择，避免续期后 Domain 突变导致正在进行的分块上传被打断。
	// 显式双向覆盖：无论上游 ctx 是否带有 smhInternalDomainCtxKey，都以 cred.UsedInternalDomain 为准，
	// 防止外网凭证被上游 ctx 误打成内网（反之亦然）。
	ctx = WithSMHInternalDomain(ctx, cred.UsedInternalDomain)

	resp, httpResp, err := apiClient.FileAPI.
		RenewMultipartUpload(ctx, smhConfig.LibraryId, smhConfig.CommonSpace, cred.ConfirmKey).
		Renew(1).
		AccessToken(accessToken).
		Execute()
	if httpResp != nil && httpResp.Body != nil {
		httpResp.Body.Close()
	}
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSmhRenewMultipartUploadFailed, smhRequestId(httpResp))
	}
	if resp == nil || resp.Domain == nil || resp.Path == nil || resp.UploadId == nil {
		return hcommon.I18nError(i18n.MsgSmhRenewMultipartUploadIncomplete)
	}

	// 重新拼装 PartURLTemplate（与 PrepareSMHCommonUpload 保持一致）
	newPartURLTemplate := fmt.Sprintf("https://%s%s?partNumber={partNumber}&uploadId=%s",
		*resp.Domain, *resp.Path, *resp.UploadId)

	var newHeaders map[string]string
	if resp.Headers != nil {
		newHeaders = *resp.Headers
	}

	expiration := resp.Expiration
	if expiration == nil {
		return hcommon.I18nError(i18n.MsgSmhRenewMultipartUploadNoExpiration)
	}

	cred.PartURLTemplate = newPartURLTemplate
	cred.PartHeaders = newHeaders
	cred.Expiration = expiration

	log.Info("SMH 分块上传凭证已续期",
		"confirmKey", cred.ConfirmKey,
		"uploadId", *resp.UploadId,
		"expiration", cred.Expiration,
		"internal_domain", cred.UsedInternalDomain,
		"domain", *resp.Domain,
		"request_id", smhRequestId(httpResp),
	)
	return nil
}

// GetSMHCommonUploadParts 查询指定 confirmKey 对应的分块上传任务中已完成的分块列表。
// 返回已上传的分块编号集合（partNumber 从 1 开始），用于断点续传时跳过已完成的分块。
// 若查询失败（如 confirmKey 已过期或不存在），返回空集合而非 error，调用方降级为全量上传。
func GetSMHCommonUploadParts(ctx context.Context, confirmKey string) (map[int]bool, error) {
	log := Logger(ctx)
	if confirmKey == "" {
		return nil, hcommon.I18nError(i18n.MsgSmhConfirmKeyCannotBeEmpty)
	}

	smhConfig := model.GetSMHConfig(ctx)
	if !smhConfig.IsConfigured() {
		return nil, hcommon.I18nError(i18n.MsgSmhNotConfigured)
	}
	if smhConfig.CommonSpace == "" {
		return nil, hcommon.I18nError(i18n.MsgSmhCommonSpaceNotConfigured)
	}

	apiClient, _, err := newSMHClientConfig(ctx)
	if err != nil {
		return nil, hcommon.I18nError(i18n.MsgSmhClientNotInitialized)
	}
	accessToken, err := getSpaceToken(ctx, "common", true)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgSmhGetTokenFailed)
	}

	resp, httpResp, err := apiClient.FileAPI.
		GetFileUpload(ctx, smhConfig.LibraryId, smhConfig.CommonSpace, confirmKey).
		Upload(1).
		AccessToken(accessToken).
		Execute()
	if httpResp != nil && httpResp.Body != nil {
		httpResp.Body.Close()
	}
	if err != nil {
		// 查询失败（任务不存在/已过期）降级为全量上传，不阻断流程
		log.Warn("SMH 查询已上传分块失败，降级为全量上传",
			"confirmKey", confirmKey,
			"request_id", smhRequestId(httpResp),
			"error", err,
		)
		return map[int]bool{}, nil
	}

	uploaded := make(map[int]bool)
	if resp != nil {
		for _, part := range resp.Parts {
			if part.PartNumber != nil {
				uploaded[int(*part.PartNumber)] = true
			}
		}
	}
	log.Info("SMH 已上传分块查询完成",
		"confirmKey", confirmKey,
		"uploadedParts", len(uploaded),
	)
	return uploaded, nil
}

// ConfirmSMHCommonUpload 在 CVM 侧上传完成后，调用 SMH CompleteFileUpload API 确认上传。
func ConfirmSMHCommonUpload(ctx context.Context, confirmKey string) error {
	smhConfig := model.GetSMHConfig(ctx)
	if !smhConfig.IsConfigured() {
		return hcommon.I18nError(i18n.MsgSmhNotConfigured)
	}

	apiClient, _, err := newSMHClientConfig(ctx)
	if err != nil {
		return hcommon.I18nError(i18n.MsgSmhClientNotInitialized)
	}
	accessToken, err := getSpaceToken(ctx, "common", true)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSmhGetTokenFailed)
	}

	_, httpResp, err := apiClient.FileAPI.
		CompleteFileUpload(ctx, smhConfig.LibraryId, smhConfig.CommonSpace, confirmKey).
		Confirm(1).
		AccessToken(accessToken).
		Execute()
	if httpResp != nil && httpResp.Body != nil {
		httpResp.Body.Close()
	}
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSmhConfirmUploadFailed, smhRequestId(httpResp))
	}
	slog.Info("SMH 上传确认成功", "confirmKey", confirmKey, "request_id", smhRequestId(httpResp))
	return nil
}

// DeleteSMHCommonFile 从 common 空间永久删除指定文件（使用 admin token）。
// fileKey 为文件在 common 空间中的路径，例如 "backups/{instanceId}/openclaw-state-xxx.tgz"。
func DeleteSMHCommonFile(ctx context.Context, fileKey string) error {
	smhConfig := model.GetSMHConfig(ctx)
	if !smhConfig.IsConfigured() {
		return hcommon.I18nError(i18n.MsgSmhNotConfigured)
	}
	if smhConfig.CommonSpace == "" {
		return hcommon.I18nError(i18n.MsgSmhCommonSpaceNotConfigured)
	}

	apiClient, _, err := newSMHClientConfig(ctx)
	if err != nil {
		return hcommon.I18nError(i18n.MsgSmhClientNotInitialized)
	}
	accessToken, err := getSpaceToken(ctx, "common", true)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSmhGetTokenFailed)
	}

	_, httpResp, err := apiClient.FileAPI.
		DeleteFile(ctx, smhConfig.LibraryId, smhConfig.CommonSpace, "/"+strings.TrimPrefix(fileKey, "/")).
		AccessToken(accessToken).
		Permanent(1).
		Execute()
	if httpResp != nil && httpResp.Body != nil {
		httpResp.Body.Close()
	}
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSmhDeleteFileFailed, fileKey, smhRequestId(httpResp))
	}
	slog.Info("SMH 备份文件已删除", "fileKey", fileKey, "request_id", smhRequestId(httpResp))
	return nil
}

// DeleteSMHCommonDirectory 从 common 空间永久删除整个目录（使用 admin token）。
// dirKey 为目录在 common 空间中的路径，例如 "backups/{instanceId}"。
// 删除为递归删除，会连同该目录下的所有文件一起清理。
// 若目录不存在（404），视为已删除，返回 nil。
func DeleteSMHCommonDirectory(ctx context.Context, dirKey string) error {
	smhConfig := model.GetSMHConfig(ctx)
	if !smhConfig.IsConfigured() {
		return hcommon.I18nError(i18n.MsgSmhNotConfigured)
	}
	if smhConfig.CommonSpace == "" {
		return hcommon.I18nError(i18n.MsgSmhCommonSpaceNotConfigured)
	}

	apiClient, _, err := newSMHClientConfig(ctx)
	if err != nil {
		return hcommon.I18nError(i18n.MsgSmhClientNotInitialized)
	}
	accessToken, err := getSpaceToken(ctx, "common", true)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSmhGetTokenFailed)
	}

	// 规整路径：去掉前后多余的 '/'
	cleanKey := strings.Trim(dirKey, "/")
	if cleanKey == "" {
		return hcommon.I18nError(i18n.MsgSmhDirPathCannotBeEmpty)
	}

	_, httpResp, err := apiClient.DirectoryAPI.
		DeleteDirectory(ctx, smhConfig.LibraryId, smhConfig.CommonSpace, "/"+cleanKey).
		AccessToken(accessToken).
		Permanent(1).
		Execute()
	if httpResp != nil && httpResp.Body != nil {
		httpResp.Body.Close()
	}
	if err != nil {
		// 目录不存在视为已删除
		if httpResp != nil && httpResp.StatusCode == 404 {
			slog.Info("SMH common 目录不存在，视为已删除", "dirKey", cleanKey, "request_id", smhRequestId(httpResp))
			return nil
		}
		return hcommon.I18nRichError(err, i18n.MsgSmhDeleteDirFailed, cleanKey, smhRequestId(httpResp))
	}
	slog.Info("SMH common 目录已删除", "dirKey", cleanKey, "request_id", smhRequestId(httpResp))
	return nil
}

// encodeFileKeyPath 对文件路径的每段做 URL 编码，保留 '/' 不编码。
// 例如 "skill-bundles/通用技能包/foo.zip" → "skill-bundles/%E9%80%9A%E7%94%A8%E6%8A%80%E8%83%BD%E5%8C%85/foo.zip"
func encodeFileKeyPath(fileKey string) string {
	segments := strings.Split(fileKey, "/")
	for i, seg := range segments {
		segments[i] = url.PathEscape(seg)
	}
	return strings.Join(segments, "/")
}

// PrepareMigrationUpload 为 agent 迁移初始化 SMH common space 分块上传。
// fileKey 由调用方指定（格式：migrations/{cvmInstanceId}/agent-export.tgz）。
// 不做容量预检（迁移文件大小未知），直接初始化分块上传。
// estimatedSize 用于计算分块数，调用方应传入合理估算值（如 500MB）。
func PrepareMigrationUpload(ctx context.Context, fileKey string, estimatedSize int64) (*SMHUploadCredential, error) {
	smhConfig := model.GetSMHConfig(ctx)
	if smhConfig.Endpoint == "" || smhConfig.LibraryId == "" || smhConfig.LibrarySecret == "" {
		return nil, hcommon.I18nError(i18n.MsgSmhNotConfigured)
	}
	if smhConfig.CommonSpace == "" {
		return nil, hcommon.I18nError(i18n.MsgSmhCommonSpaceNotConfigured)
	}

	apiClient, _, err := newSMHClientConfig(ctx)
	if err != nil {
		return nil, hcommon.I18nError(i18n.MsgSmhClientNotInitialized)
	}
	accessToken, err := getSpaceToken(ctx, "common", true)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgSmhGetTokenFailed)
	}

	partSize := int64(smhMultipartPartSize)
	totalParts := int((estimatedSize + partSize - 1) / partSize)
	if totalParts < 1 {
		totalParts = 1
	}

	smhFilePath := "/" + fileKey

	parentDir := path.Dir(smhFilePath)
	if parentDir != "/" && parentDir != "." {
		_, httpRespDir, errDir := apiClient.DirectoryAPI.
			CreateDirectory(ctx, smhConfig.LibraryId, smhConfig.CommonSpace, parentDir).
			AccessToken(accessToken).
			ConflictResolutionStrategy("ask").
			Execute()
		if httpRespDir != nil && httpRespDir.Body != nil {
			httpRespDir.Body.Close()
		}
		if errDir != nil && !strings.Contains(errDir.Error(), "409") {
			return nil, hcommon.I18nRichError(errDir, i18n.MsgSmhCreateDirFailed, parentDir, smhRequestId(httpRespDir))
		}
	}

	resp201, resp200, httpResp, err := apiClient.FileAPI.
		MultipartUploadFile(ctx, smhConfig.LibraryId, smhConfig.CommonSpace, smhFilePath).
		Multipart(1).
		ConflictResolutionStrategy("overwrite").
		AccessToken(accessToken).
		Execute()
	if httpResp != nil && httpResp.Body != nil {
		httpResp.Body.Close()
	}
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgSmhMultipartUploadFailed, smhRequestId(httpResp))
	}

	if resp200 != nil {
		slog.Info("SMH 秒传成功（内容已存在）", "fileKey", fileKey)
		return &SMHUploadCredential{FileKey: fileKey}, nil
	}

	if resp201 == nil || resp201.Domain == nil || resp201.Path == nil || resp201.ConfirmKey == nil || resp201.UploadId == nil {
		return nil, hcommon.I18nError(i18n.MsgSmhMultipartUploadIncomplete)
	}

	partURLTemplate := fmt.Sprintf("https://%s%s?partNumber={partNumber}&uploadId=%s",
		*resp201.Domain, *resp201.Path, *resp201.UploadId)

	var partHeaders map[string]string
	if resp201.Headers != nil {
		partHeaders = *resp201.Headers
	}

	slog.Info("SMH migration 分块上传凭证已获取", "fileKey", fileKey, "uploadId", *resp201.UploadId, "totalParts", totalParts)
	return &SMHUploadCredential{
		PartURLTemplate: partURLTemplate,
		PartHeaders:     partHeaders,
		ConfirmKey:      *resp201.ConfirmKey,
		FileKey:         fileKey,
		PartSize:        partSize,
		TotalParts:      totalParts,
		Expiration:      resp201.Expiration,
	}, nil
}

// CheckSMHCommonFileExists 检查 common space 中指定 fileKey 的文件是否存在。
// 返回 (exists, fileSize, error)，fileSize 为字节数（文件不存在时为 0）。
func CheckSMHCommonFileExists(ctx context.Context, fileKey string) (bool, int64, error) {
	smhConfig := model.GetSMHConfig(ctx)
	if !smhConfig.IsConfigured() {
		return false, 0, hcommon.I18nError(i18n.MsgSmhNotConfigured)
	}
	if smhConfig.CommonSpace == "" {
		return false, 0, hcommon.I18nError(i18n.MsgSmhCommonSpaceNotConfigured)
	}

	apiClient, _, err := newSMHClientConfig(ctx)
	if err != nil {
		return false, 0, hcommon.I18nError(i18n.MsgSmhClientNotInitialized)
	}
	accessToken, err := getSpaceToken(ctx, "common", true)
	if err != nil {
		return false, 0, hcommon.I18nRichError(err, i18n.MsgSmhGetTokenFailed)
	}

	resp, httpResp, err := apiClient.FileAPI.
		InfoFile(ctx, smhConfig.LibraryId, smhConfig.CommonSpace, "/"+strings.TrimPrefix(fileKey, "/")).
		Info(1).
		AccessToken(accessToken).
		Execute()
	if httpResp != nil && httpResp.Body != nil {
		httpResp.Body.Close()
	}
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			return false, 0, nil
		}
		return false, 0, hcommon.I18nRichError(err, i18n.MsgSmhInfoFileFailed, fileKey, smhRequestId(httpResp))
	}
	if resp == nil {
		return false, 0, nil
	}

	var fileSize int64
	if sizeStr := resp.GetSize(); sizeStr != "" {
		fmt.Sscanf(sizeStr, "%d", &fileSize)
	}
	return true, fileSize, nil
}

// upgradeBackupPrefix 返回运行时对应的升级备份文件名前缀。
// 空类型与未知类型保留 OpenClaw 兼容语义；升级入口已通过能力校验拦截不支持的类型。
func upgradeBackupPrefix(ctx context.Context, agentType string) string {
	if model.GetAgentRuntimeType(ctx, agentType) == model.AgentTypeHermes {
		return "hermes-state-"
	}
	return "openclaw-state-"
}

// FindLatestSMHCommonBackup 在 common space 的 backups/{instanceId}/ 目录下查找指定运行时的最新升级备份。
// agentType 为空字符串时按 OpenClaw 兼容语义使用默认前缀 "openclaw-state-"；
// 传入具体 agent type（如 "hermes"）时按运行时筛选文件名前缀。
// 返回 (fileKey, found, error)，其中 found=true 表示找到至少一个匹配备份。
// 备份文件名包含 YYYYMMDD_HHMMSS，字典序最大的即时间最新；目录不存在视为无备份。
func FindLatestSMHCommonBackup(ctx context.Context, instanceId string, agentType string) (string, bool, error) {
	smhConfig := model.GetSMHConfig(ctx)
	if !smhConfig.IsConfigured() {
		return "", false, hcommon.I18nError(i18n.MsgSmhNotConfigured)
	}
	if smhConfig.CommonSpace == "" {
		return "", false, hcommon.I18nError(i18n.MsgSmhCommonSpaceNotConfigured)
	}
	if instanceId == "" {
		return "", false, hcommon.I18nError(i18n.MsgSmhInstanceIdCannotBeEmpty)
	}

	apiClient, _, err := newSMHClientConfig(ctx)
	if err != nil {
		return "", false, hcommon.I18nError(i18n.MsgSmhClientNotInitialized)
	}
	accessToken, err := getSpaceToken(ctx, "common", true)
	if err != nil {
		return "", false, hcommon.I18nRichError(err, i18n.MsgSmhGetTokenFailed)
	}

	dirPath := fmt.Sprintf("/backups/%s", instanceId)

	// SMH ListDirectory 为分页接口，ByMarker 为必填参数（ByMarker=1 表示使用字符串 marker 分页，
	// Marker="" 表示从头开始）。循环翻页拉取全量，避免备份数量超过单页 Limit 时漏掉最新备份。
	backupPrefix := "openclaw-state-"
	if agentType != "" {
		backupPrefix = upgradeBackupPrefix(ctx, agentType)
	}

	var names []string
	marker := ""
	const pageLimit int32 = 200
	for {
		req := apiClient.DirectoryAPI.
			ListDirectory(ctx, smhConfig.LibraryId, smhConfig.CommonSpace, dirPath).
			AccessToken(accessToken).
			ByMarker(1).
			Marker(marker).
			Limit(pageLimit)
		resp, httpResp, err := req.Execute()
		if httpResp != nil && httpResp.Body != nil {
			httpResp.Body.Close()
		}
		if err != nil {
			// 目录不存在：按"无备份"处理
			if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
				return "", false, nil
			}
			return "", false, hcommon.I18nRichError(err, i18n.MsgSmhListDirFailed, dirPath, smhRequestId(httpResp))
		}
		if resp == nil {
			break
		}
		for _, item := range resp.GetContents() {
			name := item.GetName()
			// 只接受当前运行时的升级备份，避免 Hermes/OpenClaw 在同一目录下互相误用。
			if strings.HasPrefix(name, backupPrefix) && strings.HasSuffix(name, ".tgz") {
				names = append(names, name)
			}
		}
		next := resp.GetNextMarker()
		if next == "" || next == marker {
			break
		}
		marker = next
	}
	if len(names) == 0 {
		return "", false, nil
	}
	// 字典序最大的即时间最新（文件名含 YYYYMMDD_HHMMSS）
	sort.Strings(names)
	latest := names[len(names)-1]
	fileKey := fmt.Sprintf("backups/%s/%s", instanceId, latest)
	slog.Info("找到最新的 SMH 升级备份", "instanceId", instanceId, "fileKey", fileKey, "totalCandidates", len(names))
	return fileKey, true, nil
}
