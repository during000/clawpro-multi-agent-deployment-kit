package tdaimemorysdk

import (
	"fmt"

	hcommon "hatchery/common"
	"hatchery/i18n"
)

var (
	ErrEmptySecretID  = hcommon.I18nError(i18n.MsgTDAISDKSecretIDRequired)
	ErrEmptySecretKey = hcommon.I18nError(i18n.MsgTDAISDKSecretKeyRequired)
	ErrEmptyAction    = hcommon.I18nError(i18n.MsgTDAISDKActionRequired)
)

// APIError 对应腾讯云标准错误结构：Response.Error。
type APIError struct {
	Code      string
	Message   string
	RequestID string
}

func (e *APIError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.RequestID == "" {
		return fmt.Sprintf("tdai api error: code=%s, message=%s", e.Code, e.Message)
	}
	return fmt.Sprintf("tdai api error: code=%s, message=%s, request_id=%s", e.Code, e.Message, e.RequestID)
}
