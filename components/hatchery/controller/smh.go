package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	smhsdk "cnb.cool/tencent/cloud/smh/smh-go-sdk"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tchttp "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/http"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	tencentcloudsmh "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/smh/v20210712"
	"gorm.io/gorm"
)

// SMHProvisionError 是 SMH 开通流程中的自定义错误类型，携带英文错误码。
// task 包通过 errors.As 提取错误码，避免对 hatchery 内部错误做脆弱的字符串匹配。
type SMHProvisionError struct {
	Code string // 英文错误码，如 "CREATE_LIBRARY_FAILED"
	Err  error  // 原始错误
}

func (e *SMHProvisionError) Error() string { return e.Err.Error() }
func (e *SMHProvisionError) Unwrap() error { return e.Err }

// newProvisionError 创建一个带错误码的 SMHProvisionError。
func newProvisionError(code string, err error) *SMHProvisionError {
	return &SMHProvisionError{Code: code, Err: err}
}

const (
	smhService = "smh"
	smhVersion = "2021-07-12"

	// clusterTag is the cluster/bucket tag used when creating SMH libraries.
	clusterTag = "clawpro"

	// 空间配额常量
	commonSpaceQuotaBytes   int64 = 50 * 1024 * 1024 * 1024 // common 空间配额（字节）
	skillhubSpaceQuotaBytes int64 = 50 * 1024 * 1024 * 1024 // skillhub 空间配额（字节）
	personalSpaceQuotaBytes int64 = 50 * 1024 * 1024 * 1024 // 个人空间配额（字节）

	// spaceFreeQuota 空间免费配额（UpdateSpaceInternal 需要传入的是字符串）
	// - 公共空间（skillhub、common）：50GB 永久免费
	// - 个人空间：50GB 3 个月免费
	spaceFreeQuota = "50GiB"

	// personalSpaceTokenTTL 访问 Token 有效期 24 小时
	personalSpaceTokenTTL = 24 * time.Hour
	// personalSpaceTokenRefreshBefore 剩余 18 小时或以下时刷新
	personalSpaceTokenRefreshBefore = 18 * time.Hour

	// recycleBinRetention 回收站保留期：15 天
	recycleBinRetention = 15 * 24 * time.Hour

	// CurrentSMHProvisionRev 声明 hatchery 当前期望的 SMH 个人空间环境版本号。
	// 增大该值会自动为全网实例升级smh skill，这个值无法用于回滚，降低该值不会触发任何重装行为。
	// 语义：SMHPersonalSpace.EnvProvisionRev < CurrentSMHProvisionRev 的实例会被
	// task/personal_space.go 的 syncEnvs() 自动视为需要重装 init_smh_env.sh。
	CurrentSMHProvisionRev = 3
)

// personalSpaceTokenCache 个人空间 access token 缓存，key 为 spaceId
var personalSpaceTokenCache sync.Map

var (
	envSyncInflight     sync.Map // key: space.InstanceId — SyncPersonalSpaceEnv
	injectTokenInflight sync.Map // key: space.InstanceId — refreshPersonalSpaceToken
)

type cachedSpaceToken struct {
	mu          sync.Mutex
	accessToken string
	expiresAt   time.Time
}

// ensurePersonalSpaceToken 确保指定 spaceId 的 token 有效，需要时刷新。
// refreshed 为 true 表示本次调用申请了新 token，false 表示命中缓存。
func ensurePersonalSpaceToken(ctx context.Context, spaceId string) (string, time.Time, bool, error) {
	val, _ := personalSpaceTokenCache.LoadOrStore(spaceId, &cachedSpaceToken{})
	cached := val.(*cachedSpaceToken)

	cached.mu.Lock()
	defer cached.mu.Unlock()

	if cached.accessToken != "" && time.Now().Before(cached.expiresAt.Add(-personalSpaceTokenRefreshBefore)) {
		return cached.accessToken, cached.expiresAt, false, nil
	}

	newToken, err := fetchPersonalSpaceToken(ctx, spaceId)
	if err != nil {
		return "", time.Time{}, false, err
	}

	cached.accessToken = newToken
	cached.expiresAt = time.Now().Add(personalSpaceTokenTTL)

	return cached.accessToken, cached.expiresAt, true, nil
}

// fetchPersonalSpaceToken 调用 SMH API 创建一个新的访问 token。
func fetchPersonalSpaceToken(ctx context.Context, spaceId string) (string, error) {
	smhConfig := model.GetSMHConfig(ctx)
	if !smhConfig.IsConfigured() {
		return "", hcommon.I18nError(i18n.MsgSmhNotConfigured)
	}

	apiClient := newSMHAPIClient(smhConfig.Endpoint)
	tokenResp, httpResp, tokenErr := apiClient.TokenAPI.CreateToken(ctx).
		LibraryId(smhConfig.LibraryId).
		LibrarySecret(smhConfig.LibrarySecret).
		SpaceId(spaceId).
		Period(int32(personalSpaceTokenTTL.Seconds())).
		Grant("space_admin").
		Execute()
	if httpResp != nil && httpResp.Body != nil {
		httpResp.Body.Close()
	}
	if tokenErr != nil {
		return "", hcommon.I18nRichError(tokenErr, i18n.MsgSmhCreateTokenFailedWithSpace, spaceId, smhRequestId(httpResp), tokenErr)
	}
	if tokenResp == nil || tokenResp.AccessToken == nil {
		return "", hcommon.I18nError(i18n.MsgSmhCreateTokenEmptyWithSpace, spaceId, smhRequestId(httpResp))
	}

	return *tokenResp.AccessToken, nil
}

// InvalidatePersonalSpaceTokenCache 从缓存中移除指定 spaceId 的 token。
func InvalidatePersonalSpaceTokenCache(spaceId string) {
	personalSpaceTokenCache.Delete(spaceId)
}

// requireSMHEnabled 检查 SMH 服务是否已启用，未启用时返回错误响应并返回 false。
func requireSMHEnabled(w http.ResponseWriter, r *http.Request) bool {
	config := model.GetSiteConfig(r.Context())
	if config.SMHEnabled != 1 {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgSMHServiceNotEnabled))
		return false
	}
	return true
}

// ProvisionSMH executes the full SMH service provisioning flow (called by task package).
//
// Flow:
//  1. Check SiteConfig — if smh_library_id exists, skip library creation
//  2. provisionSMHLibrary — CreateLibrary + UpdateLibraryInternal + persist to SiteConfig (atomic)
//  3. describeLibrarySecret — retrieve library secret via DescribeLibrarySecret API
//  4. provisionSMHSpaces — create spaces (skillhub, common), set free quota, persist to DB
//  5. Update SiteConfig — persist full configuration (secret, spaces, enabled flag)
//
// Idempotency:
//   - Step 1 checks SiteConfig; if library exists, step 2 is skipped
//   - Step 4 checks smh_spaces table; existing Spaces are skipped
//
// Rollback:
//   - provisionSMHLibrary rolls back on its own failure
//   - provisionSMHSpaces rolls back individual spaces on failure
//
// Returns nil on success.
func ProvisionSMH(ctx context.Context) error {
	// ---------- 0. 分布式锁：防止多实例并发开通 ----------
	// 使用 TryLock 立即尝试获取锁，不阻塞等待。
	// 获取失败说明另一个实例正在执行开通，直接返回错误，
	// 由外层重试循环在下一轮重新尝试（届时要么锁已释放，要么已开通成功）。
	lock, err := model.TryLock(ctx, "smh:provision")
	if err != nil {
		return newProvisionError("PROVISION_IN_PROGRESS", hcommon.I18nRichError(err, i18n.MsgSMHProvisionLockFailed))
	}
	defer lock.Release()

	// ---------- 1. Create cloud API client ----------
	smhClient, err := newSMHClient(ctx)
	if err != nil {
		return newProvisionError("INTERNAL_ERROR", hcommon.I18nRichError(err, i18n.MsgCreateSMHClientFailed))
	}
	cloudClient := &smhClient.Client

	// ---------- 2. Check for existing ClawPro library ----------
	hostname, err := os.Hostname()
	if err != nil {
		slog.Warn("[SMH Provision] Failed to get hostname, using default", "error", err)
		hostname = "unknown"
	}
	libraryName := "ClawPro-" + hostname

	siteConfig := model.GetSiteConfig(ctx)

	var libraryId, endpoint string

	if siteConfig.SMHLibraryId != "" {
		libraryId = siteConfig.SMHLibraryId
		endpoint = siteConfig.SMHEndpoint
		slog.Info("[SMH Provision] Found existing ClawPro library in SiteConfig", "libraryId", libraryId)
	} else {
		libraryId, endpoint, err = provisionSMHLibrary(ctx, smhClient, cloudClient, libraryName)
		if err != nil {
			return err
		}
	}

	if endpoint == "" {
		return newProvisionError("CREATE_LIBRARY_FAILED", hcommon.I18nError(i18n.MsgSMHLibraryEmptyAccessDomain, libraryId))
	}

	// ---------- 3. Get library secret ----------
	librarySecret := siteConfig.SMHLibrarySecret
	if librarySecret == "" {
		librarySecret, err = describeLibrarySecret(smhClient, libraryId)
		if err != nil {
			return err
		}
		// Persist librarySecret immediately so retries won't re-fetch it
		if err := model.UpdateSiteConfig(ctx, map[string]interface{}{"smh_library_secret": librarySecret}); err != nil {
			return newProvisionError("INTERNAL_ERROR", hcommon.I18nRichError(err, i18n.MsgUpdateSiteConfigFailed))
		}
	}

	// ---------- 4. Create spaces ----------
	_, err = provisionSMHSpaces(ctx, cloudClient, libraryId, endpoint, librarySecret)
	if err != nil {
		return err
	}

	// ---------- 5. Update SiteConfig ----------
	updates := map[string]interface{}{
		"smh_enabled":         1,
		"smh_library_id":      libraryId,
		"smh_library_secret":  librarySecret,
		"smh_endpoint":        endpoint,
		"smh_provision_error": "", // 开通成功，清空错误信息
	}

	if err := model.UpdateSiteConfig(ctx, updates); err != nil {
		return newProvisionError("INTERNAL_ERROR", hcommon.I18nRichError(err, i18n.MsgUpdateSiteConfigFailed))
	}

	slog.Info("[SMH Provision] SMH service has been provisioned successfully",
		"libraryId", libraryId,
		"endpoint", endpoint,
	)
	return nil
}

// ---------- Helper Functions ----------

// spaceEntry holds a space tag and its corresponding space ID.
type spaceEntry struct {
	tag string
	id  string
}

// describeLibrary retrieves the library configuration via the DescribeLibraries SDK API.
// It queries a single libraryId and returns the corresponding Library detail.
func describeLibrary(smhClient *tencentcloudsmh.Client, libraryId string) (*tencentcloudsmh.Library, error) {
	req := tencentcloudsmh.NewDescribeLibrariesRequest()
	req.LibraryIds = []*string{&libraryId}

	resp, err := smhClient.DescribeLibraries(req)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgSmhDescribeLibrariesFailed, err)
	}

	slog.Info("[SMH] DescribeLibraries response", "body", resp.ToJsonString())

	if resp.Response == nil || len(resp.Response.List) == 0 {
		return nil, hcommon.I18nError(i18n.MsgSmhDescribeLibrariesNotFound, libraryId)
	}

	return resp.Response.List[0], nil
}

// EnsureLibrarySearchEnabled checks whether the library has EnableSearch turned on.
// If not, it calls ModifyLibrary to enable it. This is safe to call repeatedly (idempotent).
func EnsureLibrarySearchEnabled(ctx context.Context) error {
	siteConfig := model.GetSiteConfig(ctx)
	if siteConfig.SMHLibraryId == "" {
		return hcommon.I18nError(i18n.MsgSmhLibraryNotProvisioned)
	}

	smhClient, err := newSMHClient(ctx)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSmhCreateAPIClientFailed, err)
	}

	lib, rerr := describeLibrary(smhClient, siteConfig.SMHLibraryId)
	if rerr != nil {
		return hcommon.I18nRichError(rerr, i18n.MsgSmhDescribeLibraryFailed, rerr)
	}

	// Check if EnableSearch is already on
	if lib.LibraryExtension != nil && lib.LibraryExtension.EnableSearch != nil && *lib.LibraryExtension.EnableSearch {
		slog.Info("[SMH] EnableSearch is already enabled", "libraryId", siteConfig.SMHLibraryId)
		return nil
	}

	// EnableSearch is not on, call ModifyLibrary to enable it
	slog.Info("[SMH] EnableSearch is not enabled, enabling now", "libraryId", siteConfig.SMHLibraryId)

	modReq := tencentcloudsmh.NewModifyLibraryRequest()
	modReq.LibraryId = &siteConfig.SMHLibraryId
	enableSearch := true
	modReq.LibraryExtension = &tencentcloudsmh.LibraryExtension{
		EnableSearch: &enableSearch,
	}

	_, err = smhClient.ModifyLibrary(modReq)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSmhModifyLibrarySearchFailed, err)
	}

	slog.Info("[SMH] EnableSearch has been enabled successfully", "libraryId", siteConfig.SMHLibraryId)
	return nil
}

// describeLibrarySecret retrieves the library secret for the given libraryId.
// Returns the secret string on success.
func describeLibrarySecret(smhClient *tencentcloudsmh.Client, libraryId string) (string, error) {
	secretReq := tencentcloudsmh.NewDescribeLibrarySecretRequest()
	secretReq.LibraryId = &libraryId
	secretResp, err := smhClient.DescribeLibrarySecret(secretReq)
	if err != nil {
		return "", newProvisionError("DESCRIBE_SECRET_FAILED", hcommon.I18nRichError(err, i18n.MsgSMHDescribeSecretFailed))
	}

	if secretResp.Response == nil || secretResp.Response.LibrarySecret == nil || *secretResp.Response.LibrarySecret == "" {
		return "", newProvisionError("DESCRIBE_SECRET_FAILED", hcommon.I18nError(i18n.MsgSMHSecretResponseEmpty))
	}

	slog.Info("[SMH Provision] Library secret retrieved", "libraryId", libraryId)
	return *secretResp.Response.LibrarySecret, nil
}

// provisionSMHSpaces creates the required spaces (skillhub, common) in the given library.
// For each new space it also sets the free quota and persists
// the space to the smh_spaces table. Existing spaces (already in DB) are skipped.
//
// On failure, newly created spaces are rolled back (deleted).
func provisionSMHSpaces(ctx context.Context, cloudClient *common.Client, libraryId, endpoint, librarySecret string) ([]spaceEntry, error) {
	var createdSpaces []spaceEntry

	for _, tag := range []string{"skillhub", "common"} {
		// Check if space already exists in DB
		if existingId := model.GetSMHSpace(ctx, tag); existingId != "" {
			slog.Info("[SMH Provision] Space already exists, skipping", "spaceTag", tag, "spaceId", existingId)
			createdSpaces = append(createdSpaces, spaceEntry{tag: tag, id: existingId})
			continue
		}

		// 根据空间类型选择配额
		var quotaBytes int64
		switch tag {
		case "skillhub":
			quotaBytes = skillhubSpaceQuotaBytes
		default:
			quotaBytes = commonSpaceQuotaBytes
		}
		quotaStr := fmt.Sprintf("%d", quotaBytes)

		spaceId, err := createSMHSpace(ctx, endpoint, libraryId, librarySecret, tag, "admin")
		if err != nil {
			slog.Error("[SMH Provision] CreateSpace failed", "spaceTag", tag, "error", err)
			return nil, newProvisionError("CREATE_SPACE_FAILED", hcommon.I18nRichError(err, i18n.MsgSMHCreateSpaceFailed, tag))
		}

		slog.Info("[SMH Provision] Space created", "spaceTag", tag, "spaceId", spaceId)

		// Set free quota: unlimited duration (cloud API)
		// TODO(2026-04-30): 增加每月 20GB 的下行流量免费额度
		if err := updateSpaceInternal(cloudClient, updateSpaceInternalParams{
			LibraryId:           libraryId,
			SpaceId:             spaceId,
			FreeQuota:           spaceFreeQuota,
			IsUnlimitedDuration: true,
		}); err != nil {
			slog.Error("[SMH Provision] UpdateSpaceInternal failed, rolling back space", "spaceTag", tag, "spaceId", spaceId, "error", err)
			rollbackDeleteSpace(ctx, endpoint, libraryId, librarySecret, spaceId)
			return nil, newProvisionError("INTERNAL_ERROR", hcommon.I18nRichError(err, i18n.MsgSMHUpdateSpaceInternalFailed, tag))
		}
		slog.Info("[SMH Provision] Space free quota set", "spaceTag", tag, "spaceId", spaceId, "freeQuota", spaceFreeQuota)

		// Set space quota limit via smh-go-sdk
		if err := createSpaceQuota(ctx, endpoint, libraryId, librarySecret, spaceId, quotaStr); err != nil {
			slog.Error("[SMH Provision] CreateSpaceQuota failed, rolling back space", "spaceTag", tag, "spaceId", spaceId, "error", err)
			rollbackDeleteSpace(ctx, endpoint, libraryId, librarySecret, spaceId)
			return nil, newProvisionError("INTERNAL_ERROR", hcommon.I18nRichError(err, i18n.MsgSMHCreateSpaceQuotaFailed, tag))
		}
		slog.Info("[SMH Provision] Space quota limit set", "spaceTag", tag, "spaceId", spaceId, "quota", quotaStr)

		// Persist to smh_spaces table
		if err := model.UpsertSMHSpace(ctx, tag, spaceId, libraryId); err != nil {
			slog.Error("[SMH Provision] UpsertSMHSpace failed, rolling back space", "spaceTag", tag, "spaceId", spaceId, "error", err)
			rollbackDeleteSpace(ctx, endpoint, libraryId, librarySecret, spaceId)
			return nil, newProvisionError("INTERNAL_ERROR", hcommon.I18nRichError(err, i18n.MsgSMHUpsertSpaceFailed, tag))
		}

		createdSpaces = append(createdSpaces, spaceEntry{tag: tag, id: spaceId})
	}

	return createdSpaces, nil
}

// provisionSMHLibrary creates a new SMH library, updates its cluster tag, and persists
// the libraryId/endpoint to SiteConfig. On any failure after CreateLibrary, the library
// is rolled back (deleted). Returns (libraryId, endpoint, error).
func provisionSMHLibrary(ctx context.Context, smhClient *tencentcloudsmh.Client, cloudClient *common.Client, libraryName string) (string, string, error) {
	// 1. CreateLibrary
	createLibraryParams := map[string]interface{}{
		"Name":      libraryName,
		"BucketTag": clusterTag,
		"LibraryExtension": map[string]interface{}{
			"IsFileLibrary": true,
			"IsMultiSpace":  true,
			"UseRecycleBin": true,
			"EnableSearch":  true,
		},
	}

	// 海外地域不设置 ClusterTag, TODO: 临时硬编码海外地域列表，后续需要修改
	overseasRegions := []string{
		"ap-hongkong", "ap-singapore", "ap-jakarta", "ap-seoul", "ap-bangkok",
		"ap-tokyo", "me-saudi-arabia", "na-siliconvalley", "na-ashburn",
		"sa-saopaulo", "eu-frankfurt",
	}
	isOverseas := false
	for _, r := range overseasRegions {
		if strings.EqualFold(CVMRegion, r) {
			isOverseas = true
			break
		}
	}
	if !isOverseas {
		createLibraryParams["ClusterTag"] = clusterTag
	}

	createParams, _ := json.Marshal(createLibraryParams)
	createReq := tchttp.NewCommonRequest(smhService, smhVersion, "CreateLibrary")
	if err := createReq.SetActionParameters(string(createParams)); err != nil {
		return "", "", newProvisionError("CREATE_LIBRARY_FAILED", hcommon.I18nRichError(err, i18n.MsgSMHSetCreateLibParamsFailed))
	}
	createResp := tchttp.NewCommonResponse()
	if err := cloudClient.Send(createReq, createResp); err != nil {
		return "", "", newProvisionError("CREATE_LIBRARY_FAILED", hcommon.I18nRichError(err, i18n.MsgSMHCreateLibraryFailed))
	}

	slog.Info("[SMH Provision] CreateLibrary response", "body", string(createResp.GetBody()))

	var createResult struct {
		Response struct {
			LibraryId    string `json:"LibraryId"`
			AccessDomain string `json:"AccessDomain"`
			RequestId    string `json:"RequestId"`
			Error        *struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error"`
		} `json:"Response"`
	}
	if err := json.Unmarshal(createResp.GetBody(), &createResult); err != nil {
		return "", "", newProvisionError("CREATE_LIBRARY_FAILED", hcommon.I18nRichError(err, i18n.MsgSMHParseCreateLibRespFailed))
	}
	if createResult.Response.Error != nil {
		// 云 API 业务错误：保留原始错误信息，由 task 层做字符串匹配识别具体原因（如余额不足）
		return "", "", hcommon.I18nError(i18n.MsgSmhCreateLibraryAPIError,
			createResult.Response.RequestId, createResult.Response.Error.Code, createResult.Response.Error.Message)
	}

	if createResult.Response.LibraryId == "" || createResult.Response.AccessDomain == "" {
		return "", "", newProvisionError("CREATE_LIBRARY_FAILED", hcommon.I18nError(i18n.MsgSMHCreateLibRespMissing))
	}

	libraryId := createResult.Response.LibraryId
	endpoint := ensureHTTPS(createResult.Response.AccessDomain)
	slog.Info("[SMH Provision] Library created", "libraryId", libraryId, "endpoint", endpoint)

	// 2. UpdateLibraryInternal — set OmitStorageMeasure
	newEndpoint, err := updateLibraryInternal(cloudClient, libraryId)
	if err != nil {
		slog.Error("[SMH Provision] UpdateLibraryInternal failed, rolling back", "libraryId", libraryId, "error", err)
		rollbackDeleteLibrary(smhClient, libraryId)
		return "", "", newProvisionError("UPDATE_LIBRARY_FAILED", hcommon.I18nRichError(err, i18n.MsgSMHUpdateLibInternalFailed))
	}
	if newEndpoint != "" {
		endpoint = ensureHTTPS(newEndpoint)
	}
	slog.Info("[SMH Provision] Library cluster tag updated", "libraryId", libraryId, "endpoint", endpoint)

	// 3. Persist libraryId early to avoid duplicate creation on retry
	if err := model.UpdateSiteConfig(ctx, map[string]interface{}{
		"smh_library_id": libraryId,
		"smh_endpoint":   endpoint,
	}); err != nil {
		slog.Error("[SMH Provision] Failed to persist libraryId to SiteConfig, rolling back", "libraryId", libraryId, "error", err)
		rollbackDeleteLibrary(smhClient, libraryId)
		return "", "", newProvisionError("INTERNAL_ERROR", hcommon.I18nRichError(err, i18n.MsgSMHPersistLibraryFailed))
	}
	slog.Info("[SMH Provision] Library persisted to SiteConfig", "libraryId", libraryId)

	return libraryId, endpoint, nil
}

// newSMHClient creates a Tencent Cloud SMH SDK client using STS credentials.
func newSMHClient(ctx context.Context) (*tencentcloudsmh.Client, error) {
	credential, err := getCredential(ctx)
	if err != nil {
		return nil, err
	}
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = smhService + ".tencentcloudapi.com"
	cpf.HttpProfile.ReqMethod = "POST"
	return tencentcloudsmh.NewClient(credential, CVMRegion, cpf)
}

// createSMHSpace creates a space in an SMH library using the business SDK (smh-go-sdk).
// Uses LibrarySecret directly for authentication (no token exchange needed).
func createSMHSpace(ctx context.Context, endpoint, libraryId, librarySecret, spaceTag, smhOwner string) (string, error) {
	cfg := smhsdk.NewConfiguration()
	cfg.Servers = smhsdk.ServerConfigurations{
		{URL: endpoint, Description: "SMH API Server"},
	}
	apiClient := smhsdk.NewAPIClient(cfg)

	createReq := *smhsdk.NewCreateSpaceRequest()
	createReq.SetSpaceTag(spaceTag)

	spaceResp, httpResp, err := apiClient.SpaceAPI.CreateSpace(ctx, libraryId).
		LibrarySecret(librarySecret).
		UserId(smhOwner).
		CreateSpaceRequest(createReq).
		Execute()
	if err != nil {
		return "", hcommon.I18nRichError(err, i18n.MsgSmhCreateSpaceSDKFailed, libraryId, spaceTag, smhRequestId(httpResp), err)
	}
	if spaceResp == nil || spaceResp.SpaceId == nil {
		return "", hcommon.I18nError(i18n.MsgSmhCreateSpaceMissingSpaceId, libraryId, spaceTag, smhRequestId(httpResp))
	}

	return *spaceResp.SpaceId, nil
}

// updateLibraryInternal calls the UpdateLibraryInternal cloud API to update a library's
// internal configuration. Returns the new AccessDomain on success.
func updateLibraryInternal(client *common.Client, libraryId string) (string, error) {
	params, _ := json.Marshal(map[string]interface{}{
		"LibraryId":                  libraryId,
		"OmitStorageMeasure":         true,
		"OmitStorageMeasureDuration": "6m",
	})
	req := tchttp.NewCommonRequest(smhService, smhVersion, "UpdateLibraryInternal")
	if err := req.SetActionParameters(string(params)); err != nil {
		return "", hcommon.I18nRichError(err, i18n.MsgSmhSetUpdateLibParamsFailed, err)
	}
	resp := tchttp.NewCommonResponse()
	if err := client.Send(req, resp); err != nil {
		return "", hcommon.I18nRichError(err, i18n.MsgSmhUpdateLibFailed, err)
	}

	slog.Info("[SMH] UpdateLibraryInternal response", "body", string(resp.GetBody()))

	var result struct {
		Response struct {
			AccessDomain string `json:"AccessDomain"`
			RequestId    string `json:"RequestId"`
			Error        *struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error"`
		} `json:"Response"`
	}
	if err := json.Unmarshal(resp.GetBody(), &result); err != nil {
		return "", hcommon.I18nRichError(err, i18n.MsgSmhParseUpdateLibRespFailed, err)
	}
	if result.Response.Error != nil {
		return "", hcommon.I18nError(i18n.MsgSmhUpdateLibAPIError,
			result.Response.RequestId, result.Response.Error.Code, result.Response.Error.Message)
	}

	return result.Response.AccessDomain, nil
}

// ensureHTTPS prepends "https://" to the domain if it does not already have a scheme.
func ensureHTTPS(domain string) string {
	if domain == "" {
		return domain
	}
	if !strings.HasPrefix(domain, "https://") && !strings.HasPrefix(domain, "http://") {
		return "https://" + domain
	}
	return domain
}

// updateSpaceInternalParams holds the parameters for UpdateSpaceInternal cloud API.
type updateSpaceInternalParams struct {
	LibraryId            string
	SpaceId              string
	FreeQuota            string
	IsUnlimitedFreeQuota bool
	IsUnlimitedDuration  bool
	Duration             string
}

// updateSpaceInternal calls the UpdateSpaceInternal cloud API to update a space's
// internal configuration (e.g. free quota and duration).
func updateSpaceInternal(client *common.Client, p updateSpaceInternalParams) error {
	m := map[string]interface{}{
		"LibraryId":            p.LibraryId,
		"SpaceId":              p.SpaceId,
		"FreeQuota":            p.FreeQuota,
		"IsUnlimitedFreeQuota": p.IsUnlimitedFreeQuota,
		"IsUnlimitedDuration":  p.IsUnlimitedDuration,
	}
	if p.Duration != "" {
		m["Duration"] = p.Duration
	}
	params, _ := json.Marshal(m)
	req := tchttp.NewCommonRequest(smhService, smhVersion, "UpdateSpaceInternal")
	if err := req.SetActionParameters(string(params)); err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSmhSetUpdateSpaceParamsFailed, err)
	}
	resp := tchttp.NewCommonResponse()
	if err := client.Send(req, resp); err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSmhUpdateSpaceFailed, err)
	}

	slog.Info("[SMH] UpdateSpaceInternal response", "body", string(resp.GetBody()))

	var result struct {
		Response struct {
			RequestId string `json:"RequestId"`
			Error     *struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error"`
		} `json:"Response"`
	}
	if err := json.Unmarshal(resp.GetBody(), &result); err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSmhParseUpdateSpaceRespFailed, err)
	}
	if result.Response.Error != nil {
		return hcommon.I18nError(i18n.MsgSmhUpdateSpaceAPIError,
			result.Response.RequestId, result.Response.Error.Code, result.Response.Error.Message)
	}

	return nil
}

// createSpaceQuota creates a quota for a space using smh-go-sdk.
// capacity is in bytes (e.g. commonSpaceQuotaBytes).
func createSpaceQuota(ctx context.Context, endpoint, libraryId, librarySecret, spaceId, capacity string) error {
	cfg := smhsdk.NewConfiguration()
	cfg.Servers = smhsdk.ServerConfigurations{
		{URL: endpoint, Description: "SMH API Server"},
	}
	apiClient := smhsdk.NewAPIClient(cfg)

	quotaReq := *smhsdk.NewCreateQuotaRequest(false, 0)
	quotaReq.SetCapacity(capacity)
	quotaReq.SetSpaces([]string{spaceId})

	_, httpResp, err := apiClient.QuotaAPI.CreateQuota(ctx, libraryId).
		LibrarySecret(librarySecret).
		CreateQuotaRequest(quotaReq).
		Execute()
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSmhCreateQuotaSDKFailed, libraryId, spaceId, smhRequestId(httpResp), err)
	}

	return nil
}

// rollbackDeleteSpace attempts to delete a space as part of error rollback using smh-go-sdk.
// Errors are logged but not propagated.
func rollbackDeleteSpace(ctx context.Context, endpoint, libraryId, librarySecret, spaceId string) {
	if err := DeleteSMHSpace(ctx, endpoint, libraryId, librarySecret, spaceId); err != nil {
		slog.Error("[SMH] Rollback delete space failed", "libraryId", libraryId, "spaceId", spaceId, "error", err)
	} else {
		slog.Info("[SMH] Rollback delete space succeeded", "libraryId", libraryId, "spaceId", spaceId)
	}
}

// DeleteSMHSpace deletes a space using smh-go-sdk. Returns error on failure.
func DeleteSMHSpace(ctx context.Context, endpoint, libraryId, librarySecret, spaceId string) error {
	cfg := smhsdk.NewConfiguration()
	cfg.Servers = smhsdk.ServerConfigurations{
		{URL: endpoint, Description: "SMH API Server"},
	}
	apiClient := smhsdk.NewAPIClient(cfg)

	var force int32 = 1
	httpResp, err := apiClient.SpaceAPI.DeleteSpace(ctx, libraryId, spaceId).
		LibrarySecret(librarySecret).
		Force(force).
		Execute()
	if httpResp != nil && httpResp.Body != nil {
		httpResp.Body.Close()
	}
	if err != nil {
		// 空间本来就不存在（404 SpaceNotFound），视为删除成功
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			var apiErr smhsdk.GenericOpenAPIError
			if errors.As(err, &apiErr) {
				var errBody struct {
					Code string `json:"code"`
				}
				if json.Unmarshal(apiErr.Body(), &errBody) == nil && errBody.Code == "SpaceNotFound" {
					slog.Info("[SMH] Space not found, treat as already deleted",
						"libraryId", libraryId, "spaceId", spaceId,
						"requestId", smhRequestId(httpResp))
					return nil
				}
			}
		}
		return hcommon.I18nRichError(err, i18n.MsgSmhDeleteSpaceSDKFailed, libraryId, spaceId, smhRequestId(httpResp), err)
	}
	return nil
}

// rollbackDeleteLibrary attempts to delete a library as part of error rollback.
// Errors are logged but not propagated.
func rollbackDeleteLibrary(client *tencentcloudsmh.Client, libraryId string) {
	req := tencentcloudsmh.NewDeleteLibraryRequest()
	req.LibraryId = &libraryId
	if _, err := client.DeleteLibrary(req); err != nil {
		slog.Error("[SMH] Rollback delete library failed", "libraryId", libraryId, "error", err)
	} else {
		slog.Info("[SMH] Rollback delete library succeeded", "libraryId", libraryId)
	}
}

// ==========================================================================
// SMH API 辅助函数
// ==========================================================================

// personalSpaceProvisionResult 封装个人空间创建结果。
type personalSpaceProvisionResult struct {
	spaceId            string    // 新创建的空间 ID
	freeQuota          int64     // 免费配额（字节）
	capacity           int64     // 空间总配额（字节）
	freeQuotaExpiresAt time.Time // 免费配额过期时间
}

// provisionPersonalSMHSpace 为单个实例创建个人空间并设置配额。
// 封装 CreateSpace + UpdateSpaceInternal + CreateSpaceQuota 三步操作。
// 用于：ProvisionPersonalSpace（新建实例自动绑定、存量实例批量创建）。
//
// 参数：
//   - endpoint:      SMH 访问域名
//   - libraryId:     媒体库 ID
//   - librarySecret: 媒体库密钥
//   - smhOwner:      空间用户标识，格式建议为 "claw-{instanceDbId}"
//   - capacity:      空间配额（字节），如 personalSpaceQuotaBytes
//
// 返回：
//   - result: 创建结果（spaceId / freeQuota / capacity / freeQuotaExpiresAt）
//   - error:  错误信息，失败时已自动回滚（删除已创建的 Space）
func provisionPersonalSMHSpace(ctx context.Context, endpoint, libraryId, librarySecret, smhOwner string, capacity int64) (result personalSpaceProvisionResult, rerr error) {
	capacityStr := fmt.Sprintf("%d", capacity)

	// 1. CreateSpace
	spaceId, rerr := createSMHSpace(ctx, endpoint, libraryId, librarySecret, "", smhOwner)
	if rerr != nil {
		return personalSpaceProvisionResult{}, rerr
	}

	// 2. UpdateSpaceInternal — 设置免费配额
	// TODO(2026-04-30): 个人空间每月限 20GB 下行流量
	cloudClient, err := newSMHClient(ctx)
	if err != nil {
		slog.Error("[SMH] 创建云 API 客户端失败，回滚空间", "space_id", spaceId, "error", err)
		rollbackDeleteSpace(ctx, endpoint, libraryId, librarySecret, spaceId)
		return personalSpaceProvisionResult{}, hcommon.I18nRichError(err, i18n.MsgSmhCreateAPIClientFailed, err)
	}

	now := time.Now()
	if err := updateSpaceInternal(&cloudClient.Client, updateSpaceInternalParams{
		LibraryId: libraryId,
		SpaceId:   spaceId,
		FreeQuota: spaceFreeQuota,
		Duration:  "3m",
	}); err != nil {
		slog.Error("[SMH] 设置免费配额失败，回滚空间", "space_id", spaceId, "error", err)
		rollbackDeleteSpace(ctx, endpoint, libraryId, librarySecret, spaceId)
		return personalSpaceProvisionResult{}, hcommon.I18nRichError(err, i18n.MsgSmhSetFreeQuotaFailed, err)
	}

	// 3. CreateSpaceQuota — 设置配额上限
	if err := createSpaceQuota(ctx, endpoint, libraryId, librarySecret, spaceId, capacityStr); err != nil {
		slog.Error("[SMH] 设置配额上限失败，回滚空间", "space_id", spaceId, "error", err)
		rollbackDeleteSpace(ctx, endpoint, libraryId, librarySecret, spaceId)
		return personalSpaceProvisionResult{}, hcommon.I18nRichError(err, i18n.MsgSmhSetQuotaLimitFailed, err)
	}

	return personalSpaceProvisionResult{
		spaceId:            spaceId,
		freeQuota:          capacity,
		capacity:           capacity,
		freeQuotaExpiresAt: now.AddDate(0, 3, 0),
	}, nil
}

// spaceUsageInfo 封装 Space 容量信息。
type spaceUsageInfo struct {
	capacity       int64 // 空间总配额（字节）
	availableSpace int64 // 空间剩余可用量（字节）
	size           int64 // 空间已使用量（字节）
}

// describeSpaceUsage 批量查询 Space 的容量信息。
func describeSpaceUsage(ctx context.Context, endpoint, libraryId, librarySecret string, spaceIds []string) (infoMap map[string]spaceUsageInfo, rerr error) {
	cfg := smhsdk.NewConfiguration()
	cfg.Servers = smhsdk.ServerConfigurations{
		{URL: endpoint, Description: "SMH API Server"},
	}
	apiClient := smhsdk.NewAPIClient(cfg)

	joined := strings.Join(spaceIds, ",")
	resp, httpResp, err := apiClient.UsageAPI.GetUsage(ctx, libraryId, joined).
		LibrarySecret(librarySecret).
		Execute()
	if httpResp != nil && httpResp.Body != nil {
		httpResp.Body.Close()
	}
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgSmhGetUsageSDKFailed, smhRequestId(httpResp), err)
	}
	if resp == nil {
		return nil, hcommon.I18nError(i18n.MsgSmhGetUsageEmpty, smhRequestId(httpResp))
	}

	infoMap = make(map[string]spaceUsageInfo, len(*resp))
	for _, item := range *resp {
		info := spaceUsageInfo{}
		if s := item.GetSize(); s != "" {
			fmt.Sscanf(s, "%d", &info.size)
		}
		if c := item.GetCapacity(); c != "" {
			fmt.Sscanf(c, "%d", &info.capacity)
		}
		if a := item.GetAvailableSpace(); a != "" {
			fmt.Sscanf(a, "%d", &info.availableSpace)
		}
		infoMap[item.GetSpaceId()] = info
	}
	return infoMap, nil
}

// describeLibraryUsage 查询整个 Library 的总已用存储（字节）。
func describeLibraryUsage(ctx context.Context, endpoint, libraryId, librarySecret string) (totalFileSize int64, rerr error) {
	cfg := smhsdk.NewConfiguration()
	cfg.Servers = smhsdk.ServerConfigurations{
		{URL: endpoint, Description: "SMH API Server"},
	}
	apiClient := smhsdk.NewAPIClient(cfg)

	resp, httpResp, err := apiClient.UsageAPI.GetLibraryUsage(ctx, libraryId).
		LibrarySecret(librarySecret).
		Execute()
	if httpResp != nil && httpResp.Body != nil {
		httpResp.Body.Close()
	}
	if err != nil {
		return 0, hcommon.I18nRichError(err, i18n.MsgSmhGetLibUsageSDKFailed, smhRequestId(httpResp), err)
	}
	if resp == nil {
		return 0, hcommon.I18nError(i18n.MsgSmhGetLibUsageEmpty, smhRequestId(httpResp))
	}

	if s := resp.GetTotalFileSize(); s != "" {
		fmt.Sscanf(s, "%d", &totalFileSize)
	}
	return totalFileSize, nil
}

// RecyclePersonalSpace 乐观锁将空间标记进回收站。
// 返回 true 表示实际发生变更，false 表示记录不存在或已在回收站。
func RecyclePersonalSpace(ctx context.Context, space *model.SMHPersonalSpace) (bool, error) {
	deleteAt := time.Now().Add(recycleBinRetention)
	result := model.DB(ctx).Model(space).
		Where("to_be_deleted_at IS NULL").
		Update("to_be_deleted_at", deleteAt)
	return result.RowsAffected > 0, result.Error
}

// RestorePersonalSpace 乐观锁将空间从回收站恢复为活跃，并重置 env_initialized。
// 返回 true 表示实际发生变更，false 表示记录不存在、已是活跃状态或关联实例已删除。
// 通过子查询校验实例仍存在，防止实例删除与空间恢复之间的竞争导致悬空空间。
func RestorePersonalSpace(ctx context.Context, space *model.SMHPersonalSpace) (bool, error) {
	result := model.DB(ctx).Model(space).
		Where("to_be_deleted_at IS NOT NULL").
		Where("instance_id IN (?)",
			model.DB(ctx).Model(&model.Instance{}).Select("id").Where("id = ?", space.InstanceId),
		).
		Updates(map[string]interface{}{
			"to_be_deleted_at": nil,
			"env_initialized":  false,
		})
	return result.RowsAffected > 0, result.Error
}

// MarkPersonalSpaceToBeDeleted 将实例关联的个人空间标记为待删除。
// 实例删除时调用，无关联空间时静默跳过。
func MarkPersonalSpaceToBeDeleted(ctx context.Context, instanceDBID uint) {
	deleteAt := time.Now().Add(recycleBinRetention)
	result := model.DB(ctx).Model(&model.SMHPersonalSpace{}).
		Where("instance_id = ? AND to_be_deleted_at IS NULL", instanceDBID).
		Update("to_be_deleted_at", deleteAt)
	if result.RowsAffected > 0 {
		slog.Info("[SMH] 个人空间已标记为回收站", "instance_id", instanceDBID, "to_be_deleted_at", deleteAt)
	}
}

// MarkPersonalSpacesToBeDeletedByUser 批量将用户名下所有实例的个人空间标记为待删除。
// 用户删除时调用，通过子查询一条 SQL 完成，避免 N+1 查询。
func MarkPersonalSpacesToBeDeletedByUser(ctx context.Context, userID uint) {
	deleteAt := time.Now().Add(recycleBinRetention)
	result := model.DB(ctx).Model(&model.SMHPersonalSpace{}).
		Where("instance_id IN (?) AND to_be_deleted_at IS NULL",
			model.DB(ctx).Model(&model.Instance{}).Where("user_id = ?", userID).Select("id"),
		).Update("to_be_deleted_at", deleteAt)
	if result.RowsAffected > 0 {
		slog.Info("[SMH] 用户实例关联的个人空间已批量标记回收站", "user_id", userID, "count", result.RowsAffected)
	}
}

// waitForCVMRunning 轮询等待 CVM 实例进入 RUNNING 状态。
// 每 10 秒查询一次，超时返回 false。
func waitForCVMRunning(ctx context.Context, cvmInstanceId string, timeout time.Duration) bool {
	if cvmInstanceId == "" {
		return false
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		state, err := fetchCVMState(ctx, cvmInstanceId)
		if err != nil {
			slog.Warn("[SMH] 等待 CVM 就绪：查询状态失败", "cvm_instance_id", cvmInstanceId, "error", err)
			time.Sleep(10 * time.Second)
			continue
		}
		if state == "RUNNING" {
			return true
		}
		time.Sleep(10 * time.Second)
	}
	return false
}

// waitForTATAgentOnline 轮询等待 TAT Agent 变为 Online 状态。
// 返回 true 表示就绪，false 表示超时。
func waitForTATAgentOnline(ctx context.Context, cvmInstanceId string, timeout time.Duration) bool {
	if cvmInstanceId == "" {
		return false
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		client, err := NewTATClient(ctx)
		if err != nil {
			slog.Warn("[SMH] 等待 TAT Agent 就绪：创建客户端失败", "cvm_instance_id", cvmInstanceId, "error", err)
			time.Sleep(10 * time.Second)
			continue
		}
		if err := checkAgentOnline(client, cvmInstanceId); err == nil {
			return true
		}
		slog.Info("[SMH] TAT Agent 尚未就绪，继续等待", "cvm_instance_id", cvmInstanceId)
		time.Sleep(10 * time.Second)
	}
	return false
}

// tryAcquireWithLock 两层锁：本地 sync.Map 做进程内快速去重，MySQL 分布式锁做跨实例互斥。
// 成功返回释放函数和 true；已有在执行则返回 nil 和 false。
// lockResource 为分布式锁的资源名（如 "smh:env-sync:123"）。
func tryAcquireWithLock(ctx context.Context, m *sync.Map, key interface{}, lockResource string) (release func(), ok bool) {
	// 第一层：进程内去重（SQLite 单实例模式下也生效）
	if _, loaded := m.LoadOrStore(key, struct{}{}); loaded {
		return nil, false
	}

	// 第二层：MySQL 分布式锁（SQLite 模式下为空操作）
	lock, err := model.TryLock(ctx, lockResource)
	if err != nil {
		m.Delete(key)
		slog.Warn("[SMH] 获取分布式锁失败，跳过", "resource", lockResource, "error", err)
		return nil, false
	}

	return func() {
		lock.Release()
		m.Delete(key)
	}, true
}

// syncSMHEnvWhenReadyFn 是 syncSMHEnvWhenReady 的间接引用，便于单测 mock。
var syncSMHEnvWhenReadyFn = syncSMHEnvWhenReady

// syncSMHEnvWhenReady 重置 SMH 环境状态并等待 CVM 就绪后触发完整初始化（init_smh_env.sh）。
//
// 执行步骤：
//  1. 前置查询 space 是否存在（不存在则直接返回，避免无意义等待）
//  2. 重置 env_initialized=false（确保即使后续步骤失败，syncEnvs 定时任务也能兜底）
//  3. 等待 CVM RUNNING + TAT Agent Online
//  4. 触发 TriggerSyncPersonalSpaceEnv(install=true)
//
// 注意：
//   - 本函数不检查 AgentTypeSupportsSMH，因为前置 space 查询已是天然屏障
//   - 本函数为阻塞调用，调用方根据上下文决定是否 go 启动
//   - 并发安全：TriggerSyncPersonalSpaceEnv 内置双层锁（进程内 sync.Map + MySQL 分布式锁），
//     主动触发与 syncEnvs 兜底不会重复执行
func syncSMHEnvWhenReady(ctx context.Context, instance model.Instance) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("[SMH] syncSMHEnvWhenReady panic", "instance_id", instance.ID, "cvm_instance_id", instance.InstanceId, "error", r)
		}
	}()

	// 前置检查：如果没有活跃的个人空间记录，直接返回，避免无意义等待 CVM
	var space model.SMHPersonalSpace
	err := model.DB(ctx).Where("instance_id = ? AND to_be_deleted_at IS NULL", instance.ID).First(&space).Error
	switch {
	case err == nil:
		// 找到活跃空间，继续处理
	case errors.Is(err, gorm.ErrRecordNotFound):
		slog.Info("[SMH] 未找到活跃个人空间，跳过环境同步", "instance_id", instance.ID, "cvm_instance_id", instance.InstanceId)
		return
	default:
		slog.Warn("[SMH] 查询个人空间失败", "instance_id", instance.ID, "cvm_instance_id", instance.InstanceId, "error", err)
		return
	}

	// 重置 env_initialized=false，确保即使后续等待超时，syncEnvs 定时任务也能兜底
	if err := model.DB(ctx).Model(&space).Update("env_initialized", false).Error; err != nil {
		slog.Warn("[SMH] 重置环境初始化状态失败", "instance_id", instance.ID, "cvm_instance_id", instance.InstanceId, "error", err)
		return
	}

	if !waitForCVMRunning(ctx, instance.InstanceId, 10*time.Minute) {
		slog.Warn("[SMH] 等待 CVM 就绪超时，后台服务兜底", "instance_id", instance.ID, "cvm_instance_id", instance.InstanceId)
		return
	}
	if !waitForTATAgentOnline(ctx, instance.InstanceId, 3*time.Minute) {
		slog.Warn("[SMH] 等待 TAT Agent 就绪超时，后台服务兜底", "instance_id", instance.ID, "cvm_instance_id", instance.InstanceId)
		return
	}
	if err := TriggerSyncPersonalSpaceEnv(ctx, &space, true); err != nil {
		slog.Warn("[SMH] 触发环境初始化失败，后台服务兜底", "instance_id", instance.ID, "cvm_instance_id", instance.InstanceId, "error", err)
	}
}

// TriggerSyncPersonalSpaceEnv 安装或卸载个人空间环境（skill + token），同一实例并发调用自动跳过。
// 两层锁：本地 sync.Map 做进程内快速去重，MySQL 分布式锁做跨实例互斥。
func TriggerSyncPersonalSpaceEnv(ctx context.Context, space *model.SMHPersonalSpace, install bool) error {
	lockResource := fmt.Sprintf("smh:env-sync:%d", space.InstanceId)
	release, ok := tryAcquireWithLock(ctx, &envSyncInflight, space.InstanceId, lockResource)
	if !ok {
		slog.Info("[SMH] 个人空间环境同步已在执行，跳过", "instance_id", space.InstanceId, "cvm_instance_id", space.CVMInstanceId, "install", install)
		return nil
	}
	defer release()

	if rerr := SyncPersonalSpaceEnv(ctx, space, install); rerr != nil {
		return rerr
	}
	return nil
}

// TriggerRefreshPersonalSpaceToken 刷新 token 并注入，同一实例并发调用自动跳过。
// 两层锁：本地 sync.Map 做进程内快速去重，MySQL 分布式锁做跨实例互斥。
func TriggerRefreshPersonalSpaceToken(ctx context.Context, space *model.SMHPersonalSpace) error {
	lockResource := fmt.Sprintf("smh:token-refresh:%d", space.InstanceId)
	release, ok := tryAcquireWithLock(ctx, &injectTokenInflight, space.InstanceId, lockResource)
	if !ok {
		slog.Info("[SMH] 个人空间 token 刷新已在执行，跳过", "instance_id", space.InstanceId, "cvm_instance_id", space.CVMInstanceId)
		return nil
	}
	defer release()
	return refreshPersonalSpaceToken(ctx, space)
}

// CreatePersonalSpaceForInstance 为实例创建个人空间并写入数据库，返回 spaceId。
// 使用分布式锁（MySQL）+ 进程内 double-check 防止并发重复创建。
func CreatePersonalSpaceForInstance(ctx context.Context, instance *model.Instance, user *model.User) (string, error) {
	smhConfig := model.GetSMHConfig(ctx)
	if !smhConfig.IsConfigured() {
		return "", hcommon.I18nError(i18n.MsgSmhNotConfigured)
	}

	lockResource := fmt.Sprintf("smh:personal-space-provision:%d", instance.ID)
	lock, err := model.TryLock(ctx, lockResource)
	if err != nil {
		return "", hcommon.I18nRichError(err, i18n.MsgSmhAcquireDistLockFailed, err)
	}
	defer lock.Release()

	// double-check：拿到锁后再确认空间尚未创建
	exists, err := model.HasPersonalSpace(ctx, instance.ID)
	if err != nil {
		return "", hcommon.I18nRichError(err, i18n.MsgSmhCheckPersonalSpaceFailed, err)
	}
	if exists {
		slog.Info("[SMH] 个人空间已存在，跳过创建", "cvm_instance_id", instance.InstanceId)
		var existing model.SMHPersonalSpace
		model.DB(ctx).Where("instance_id = ?", instance.ID).First(&existing)
		return existing.SpaceId, nil
	}

	smhOwner := fmt.Sprintf("claw-%s", instance.InstanceId)
	provResult, rerr := provisionPersonalSMHSpace(ctx, smhConfig.Endpoint, smhConfig.LibraryId, smhConfig.LibrarySecret, smhOwner, personalSpaceQuotaBytes)
	if rerr != nil {
		return "", rerr
	}

	space := model.SMHPersonalSpace{
		SpaceId:          provResult.spaceId,
		UserId:           user.ID,
		InstanceId:       instance.ID,
		UserName:         user.Username,
		InstanceName:     instance.Name,
		CVMInstanceId:    instance.InstanceId,
		StorageQuota:     provResult.capacity,
		FreeStorageQuota: provResult.freeQuota,
		ExpiresAt:        &provResult.freeQuotaExpiresAt,
	}
	if err := model.CreatePersonalSpace(ctx, &space); err != nil {
		return "", hcommon.I18nRichError(err, i18n.MsgSmhSavePersonalSpaceFailed, err)
	}

	slog.Info("[SMH] 个人空间已创建", "cvm_instance_id", instance.InstanceId, "space_id", provResult.spaceId, "expires_at", provResult.freeQuotaExpiresAt)
	return provResult.spaceId, nil
}

// SyncPersonalSpaceEnv 安装或卸载个人空间环境（skill + token）。
// install=true：执行 init_smh_env_*.sh（单次 TAT，同时安装 skill 并注入 token），成功后置 env_initialized=true。
// install=false：执行 remove_smh_env_*.sh（单次 TAT，卸载 skill 并清除 token），成功后置 env_initialized=false。
// final：按 agent_type 分派脚本（openclaw / hermes / ace），
// 避免原硬编码 openclaw 脚本导致 Hermes/ACE 实例执行错脚本。
// 调用方需尽量确保 CVM 处于 RUNNING 状态。
func SyncPersonalSpaceEnv(ctx context.Context, space *model.SMHPersonalSpace, install bool) error {
	agentType := LookupAgentType(ctx, space.CVMInstanceId)
	runtimeUser := ensureRuntimeUser(ctx, space.InstanceId, space.CVMInstanceId, agentType)

	if install {
		token, _, _, err := ensurePersonalSpaceToken(ctx, space.SpaceId)
		if err != nil {
			return hcommon.I18nRichError(err, i18n.MsgSmhGetTokenFailedSpace, space.SpaceId, err)
		}

		smhConfig := model.GetSMHConfig(ctx)
		if !smhConfig.IsConfigured() {
			return hcommon.I18nError(i18n.MsgSmhNotConfigured)
		}

		scriptName, rerr := ResolveScript(ctx, "init_smh_env", agentType)
		if rerr != nil {
			return hcommon.I18nRichError(rerr, i18n.MsgSmhResolveScriptFailed, "init_smh_env", agentType)
		}

		// 使用 runScriptFn（定义于 version_fetcher.go）而非直接调用 RunScript，
		// 以便在单元测试中 mock 该函数覆盖成功/失败路径。
		_, err = runScriptFn(ctx, space.CVMInstanceId, scriptName, 180, runtimeUser, nil, map[string]string{
			"agent_type":  agentType,
			"skill_name":  "tencent-agent-storage",
			"basePath":    smhConfig.Endpoint,
			"libraryId":   smhConfig.LibraryId,
			"spaceId":     space.SpaceId,
			"accessToken": token,
		})
		if err != nil {
			return hcommon.I18nRichError(err, i18n.MsgSmhInitEnvFailed, space.CVMInstanceId, agentType, err)
		}

		if err := model.DB(ctx).Model(space).Updates(map[string]interface{}{
			"env_initialized":   true,
			"env_provision_rev": CurrentSMHProvisionRev,
		}).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgSmhUpdateEnvStatusFailed, err)
		}
		space.EnvInitialized = true
		space.EnvProvisionRev = CurrentSMHProvisionRev
		slog.Info("[SMH] 个人空间环境已初始化", "cvm_instance_id", space.CVMInstanceId, "space_id", space.SpaceId, "agent_type", agentType, "env_provision_rev", CurrentSMHProvisionRev)
	} else {
		scriptName, rerr := ResolveScript(ctx, "remove_smh_env", agentType)
		if rerr != nil {
			return hcommon.I18nRichError(rerr, i18n.MsgSmhResolveScriptFailed, "remove_smh_env", agentType)
		}
		// 使用 runScriptFn 便于单测 mock（见上方 install 分支注释）。
		_, err := runScriptFn(ctx, space.CVMInstanceId, scriptName, 120, runtimeUser, nil, map[string]string{
			"agent_type": agentType,
		})
		if err != nil {
			return hcommon.I18nRichError(err, i18n.MsgSmhRemoveEnvFailed, space.CVMInstanceId, agentType)
		}

		if err := model.DB(ctx).Model(space).Update("env_initialized", false).Error; err != nil {
			return hcommon.I18nRichError(err, i18n.MsgSmhUpdateEnvStatusFailed, err)
		}
		space.EnvInitialized = false
		slog.Info("[SMH] 个人空间环境已卸载", "cvm_instance_id", space.CVMInstanceId, "space_id", space.SpaceId, "agent_type", agentType)
	}
	return nil
}

// refreshPersonalSpaceToken 获取 token 并通过 TAT 注入到实例。
// 注意：此函数只刷新 token，不安装/卸载 skill。
func refreshPersonalSpaceToken(ctx context.Context, space *model.SMHPersonalSpace) error {
	token, expiresAt, _, err := ensurePersonalSpaceToken(ctx, space.SpaceId)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSmhPersonalTokenFailed, err)
	}

	smhConfig := model.GetSMHConfig(ctx)
	if !smhConfig.IsConfigured() {
		return hcommon.I18nError(i18n.MsgSmhNotConfigured)
	}

	agentType := LookupAgentType(ctx, space.CVMInstanceId)
	runtimeUser := ensureRuntimeUser(ctx, space.InstanceId, space.CVMInstanceId, agentType)
	params := map[string]string{
		"agent_type":  agentType,
		"basePath":    smhConfig.Endpoint,
		"libraryId":   smhConfig.LibraryId,
		"spaceId":     space.SpaceId,
		"accessToken": token,
	}
	scriptName, rerr := ResolveScript(ctx, "set_smh_token", agentType)
	if rerr != nil {
		return hcommon.I18nRichError(rerr, i18n.MsgSmhResolveScriptFailed, "set_smh_token", agentType)
	}
	// 使用 runScriptFn（定义于 version_fetcher.go）而非直接调用 RunScript，
	// 以便在单元测试中 mock 该函数覆盖成功路径。
	_, err = runScriptFn(ctx, space.CVMInstanceId, scriptName, 60, runtimeUser, nil, params)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSmhInjectEnvVarFailed, err)
	}

	// 回写下发时间用于 task 层门控；失败不影响本次 TAT 已成功的事实。
	if err := model.DB(ctx).Model(space).Update("last_pushed_token_expires_at", expiresAt).Error; err != nil {
		slog.Warn("[SMH] 回写 last_pushed_token_expires_at 失败，下轮会自愈", "cvm_instance_id", space.CVMInstanceId, "space_id", space.SpaceId, "error", err)
	}

	slog.Info("[SMH] 个人空间 token 已注入", "cvm_instance_id", space.CVMInstanceId, "space_id", space.SpaceId, "agent_type", agentType, "expires_at", expiresAt)
	return nil
}
