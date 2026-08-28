package controller

import (
	hcommon "hatchery/common"
	"hatchery/i18n"
)

// ErrMethodNotAllowed 请求方法不允许（405）。
// 所有 handler 应使用此常量替代硬编码的 "Method not allowed"。
var ErrMethodNotAllowed = hcommon.I18nError(i18n.MsgMethodNotAllowed)

// ErrUnauthorized 未登录（401）。
var ErrUnauthorized = hcommon.I18nError(i18n.MsgUnauthorized)

// ErrForbidden 无权限访问（403）。
var ErrForbidden = hcommon.I18nError(i18n.MsgForbidden)

// ErrAdminRequired 需要管理员权限（403）。
var ErrAdminRequired = hcommon.I18nError(i18n.MsgAdminRequired)
