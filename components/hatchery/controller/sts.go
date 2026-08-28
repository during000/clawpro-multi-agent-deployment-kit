package controller

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	sts "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sts/v20180813"
)

// ─── 通用 AssumeRole 底层 ─────────────────────────────────────────────────────

// AssumeRoleOptions 定义 AssumeRole 调用的可选参数。
type AssumeRoleOptions struct {
	// DurationSeconds STS 临时凭证有效期（秒），默认 7200（2 小时）。
	DurationSeconds uint64
	// Policy 权限策略 JSON 字符串。为空时不限制（继承角色全部权限）。
	Policy string
	// SessionName 会话名称，默认 "hatchery"。
	SessionName string
}

// STSCredentials 保存临时密钥三元组。
type STSCredentials struct {
	SecretId  string
	SecretKey string
	Token     string
	// ExpiredAt Unix 秒级过期时间戳（来自 AssumeRole 响应）。
	ExpiredAt int64
}

// ─── 全局 STS 刷新（存 DB） ──────────────────────────────────────────────────

// RefreshSTSCredentials 调用 AssumeRole 获取临时凭证并保存到 site_configs 表。
func RefreshSTSCredentials(ctx context.Context) error {
	cred, err := AssumeRoleWithOptions(ctx, nil)
	if err != nil {
		return err
	}
	if cred.ExpiredAt == 0 {
		return hcommon.I18nError(i18n.MsgSTSExpiredTimeEmpty)
	}

	model.UpdateSiteConfig(ctx, map[string]interface{}{
		"sts_tmp_secret_id":  cred.SecretId,
		"sts_tmp_secret_key": cred.SecretKey,
		"sts_token":          cred.Token,
		"sts_expired_at":     cred.ExpiredAt,
	})

	slog.Info("STS 临时密钥刷新成功", "expired_at", cred.ExpiredAt)
	return nil
}

// ─── 凭证获取 ────────────────────────────────────────────────────────────────

// getCredential returns the appropriate credential for Tencent Cloud SDK clients.
// When CVMUin is set, it uses STS temporary credentials; otherwise, it uses the permanent ones.
func getCredential(ctx context.Context) (*common.Credential, error) {
	config := model.GetSiteConfig(ctx)
	if config.CVMSecretId == "" || config.CVMSecretKey == "" {
		return nil, hcommon.I18nError(i18n.MsgSTSCredentialNotConfig)
	}
	uin := hcommon.CVMUinFromCtx(ctx)
	if uin != "" {
		if config.STSTmpSecretId == "" || config.STSToken == "" || time.Now().Unix() >= config.STSExpiredAt-300 {
			if err := RefreshSTSCredentials(ctx); err != nil {
				return nil, hcommon.I18nRichError(err, i18n.MsgSTSRefreshFailed)
			}
			config = model.GetSiteConfig(ctx)
		}
		return common.NewTokenCredential(config.STSTmpSecretId, config.STSTmpSecretKey, config.STSToken), nil
	}
	return common.NewCredential(config.CVMSecretId, config.CVMSecretKey), nil
}

// ─── 实例级 STS（龙虾医生用） ─────────────────────────────────────────────────

// instanceIdRe 校验 CVM 实例 ID 格式（ins-xxxxxxxx）。
var instanceIdRe = regexp.MustCompile(
	`^ins-[a-zA-Z0-9]{5,20}$`)

func isValidInstanceId(id string) bool {
	return instanceIdRe.MatchString(id)
}

// RequestInstanceScopedSTS 申请限定到指定实例的 TAT 操作权限的 STS 临时密钥。
// 有效期 2 小时，仅允许对目标实例执行 TAT 相关操作。
func RequestInstanceScopedSTS(
	ctx context.Context,
	instanceId string,
) (*STSCredentials, error) {
	// 校验 instanceId 格式，防止 policy JSON 注入
	if !isValidInstanceId(instanceId) {
		return nil, hcommon.I18nError(i18n.MsgSTSInvalidInstanceId, instanceId)
	}

	policy := fmt.Sprintf(`{
	"version": "2.0",
	"statement": [
		{
			"effect": "allow",
			"action": [
				"tat:StartSession",
				"tat:DescribeAutomationAgentStatus"
			],
			"resource": [
				"qcs::cvm:%s:uin/%s:instance/%s"
			]
		}
	]
}`, CVMRegion, hcommon.CVMUinFromCtx(ctx), instanceId)

	cred, err := AssumeRoleWithOptions(ctx, &AssumeRoleOptions{
		Policy: policy,
	})
	if err != nil {
		return nil, err
	}

	slog.Info("[Doctor] 获取实例级 STS 临时凭证成功",
		"instance", instanceId)
	return cred, nil
}

// AssumeRoleWithOptions 使用永久密钥调用 STS AssumeRole，返回临时凭证。
// 这是底层通用函数，RefreshSTSCredentials 和 RequestInstanceScopedSTS 均委托此函数。
func AssumeRoleWithOptions(ctx context.Context, opts *AssumeRoleOptions) (*STSCredentials, error) {
	config := model.GetSiteConfig(ctx)
	if config.CVMSecretId == "" || config.CVMSecretKey == "" {
		return nil, hcommon.I18nError(i18n.MsgSTSSecretNotConfigured)
	}

	// 填充默认值
	if opts == nil {
		opts = &AssumeRoleOptions{}
	}
	if opts.DurationSeconds == 0 {
		opts.DurationSeconds = 7200
	}
	if opts.SessionName == "" {
		opts.SessionName = "hatchery"
	}

	credential := common.NewCredential(
		config.CVMSecretId, config.CVMSecretKey)
	cpf := profile.NewClientProfile()
	client, err := sts.NewClient(credential, CVMRegion, cpf)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgSTSCreateClientFailed)
	}

	req := sts.NewAssumeRoleRequest()
	roleArn := fmt.Sprintf(
		"qcs::cam::uin/%s:role/"+
			"tencentcloudServiceRoleName/"+
			"CVM_QCSLinkedRoleInClawPro",
		hcommon.CVMUinFromCtx(ctx))
	req.RoleArn = &roleArn
	req.RoleSessionName = &opts.SessionName
	req.DurationSeconds = &opts.DurationSeconds

	if opts.Policy != "" {
		req.Policy = &opts.Policy
	}

	resp, err := client.AssumeRole(req)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgSTSAssumeRoleFailed)
	}

	cred := resp.Response.Credentials
	if cred == nil || cred.TmpSecretId == nil ||
		cred.TmpSecretKey == nil || cred.Token == nil {
		return nil, hcommon.I18nError(i18n.MsgSTSCredentialEmpty)
	}

	var expiredAt int64
	if resp.Response.ExpiredTime != nil {
		expiredAt = int64(*resp.Response.ExpiredTime)
	}

	return &STSCredentials{
		SecretId:  *cred.TmpSecretId,
		SecretKey: *cred.TmpSecretKey,
		Token:     *cred.Token,
		ExpiredAt: expiredAt,
	}, nil
}
