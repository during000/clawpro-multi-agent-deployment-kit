package controller

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	tcommon "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tchttp "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/http"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

const (
	emailTypeWelcome       = 2 // 欢迎（新用户密码通知）
	emailTypeResetPassword = 3 // 密码重置
)

func emailLoginURL(ctx context.Context, config model.SiteConfig) string {
	loginURL := strings.TrimSpace(hcommon.DomainFromCtx(ctx))
	if loginURL == "" {
		loginURL = strings.TrimSpace(config.Domain)
	}
	return loginURL
}

func emailLoginURLForRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	host := extractHost(r)
	if host == "" {
		return ""
	}
	return (&url.URL{Scheme: "https", Host: host}).String()
}

// sendEmail 发送邮件，通过腾讯云 SDK CommonClient 调用 CreateSendOpenClawEmail API。
// apiURL 示例：http://cvm.test.tencentcloudapi.com，自动解析 scheme、endpoint 和 service。
func sendEmail(ctx context.Context, to string, emailType int, region string, apiURL string, params map[string]any) error {
	config := model.GetSiteConfig(ctx)
	if apiURL == "" {
		return hcommon.I18nError(i18n.MsgEmailAPIURLNotConfigured)
	}

	u, err := url.Parse(apiURL)
	if err != nil || u.Host == "" {
		return hcommon.I18nError(i18n.MsgEmailInvalidAPIURL, apiURL, err)
	}
	scheme := u.Scheme
	endpoint := u.Host
	service := strings.SplitN(u.Host, ".", 2)[0]

	if params == nil {
		params = map[string]any{}
	}
	params["sender_name"] = config.Name
	loginURL, hasLoginURL := params["login_url"].(string)
	if !hasLoginURL || strings.TrimSpace(loginURL) == "" {
		if loginURL := emailLoginURL(ctx, config); loginURL != "" {
			params["login_url"] = loginURL
		}
	}
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgEmailMarshalParamsFailed)
	}

	slog.Info("[EMAIL] 发送邮件", "to", to, "type", emailType, "params", string(paramsJSON))

	credential, rerr := getCredential(ctx)
	if rerr != nil {
		return hcommon.I18nRichError(rerr, i18n.MsgEmailGetCredFailed)
	}
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = endpoint
	cpf.HttpProfile.Scheme = scheme
	cpf.HttpProfile.ReqMethod = "POST"
	client := tcommon.NewCommonClient(credential, region, cpf)

	request := tchttp.NewCommonRequest(service, "2017-03-12", "CreateSendOpenClawEmail")
	actionParams := map[string]any{
		"AccessToken":        AdminToken,
		"DestinationAddress": to,
		"Type":               emailType,
		"Parameters":         string(paramsJSON),
	}
	actionJSON, err := json.Marshal(actionParams)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgEmailMarshalActionFailed)
	}
	if err := request.SetActionParameters(string(actionJSON)); err != nil {
		return hcommon.I18nRichError(err, i18n.MsgSetActionParamsFailed)
	}

	response := tchttp.NewCommonResponse()
	if err := client.Send(request, response); err != nil {
		slog.Error("[EMAIL] 发送失败", "error", err)
		return hcommon.I18nRichError(err, i18n.MsgEmailSendFailed)
	}

	respBody := response.GetBody()
	var result struct {
		Response struct {
			Error *struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error"`
		} `json:"Response"`
	}
	if json.Unmarshal(respBody, &result) == nil && result.Response.Error != nil {
		slog.Error("[EMAIL] API 错误", "code", result.Response.Error.Code, "message", result.Response.Error.Message)
		return hcommon.I18nError(i18n.MsgEmailAPIError, result.Response.Error.Code, result.Response.Error.Message)
	}

	slog.Info("[EMAIL] 发送成功", "to", to, "type", emailType, "resp", string(respBody))
	return nil
}
