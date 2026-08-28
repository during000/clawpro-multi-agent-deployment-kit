package controller

import (
	"errors"
	"strconv"
	"strings"

	"hatchery/common"
	"hatchery/i18n"
)

// errVersionNotHigher 角色版本号未严格大于旧版本号的 sentinel 错误，
// 用于在 tx 内 i18nRichError 包装时被外层路由识别并返回 400。
var errVersionNotHigher = errors.New("role_version_not_higher")

// parseRoleVersion 把 "X.Y" 解析为 (major, minor) 整数对。
//   - 空串返回 (-1, -1)：视为最低版本（"从未下发过"）
//   - 非法格式（如 "v1.0" / "1" / "1.0.0" / "abc"）返回 (0, 0)：等价于 0.0
//
// 不在这里做边界报错，调用方根据用途决定如何处理。
func parseRoleVersion(v string) (int, int) {
	if v == "" {
		return -1, -1
	}
	parts := strings.SplitN(v, ".", 2)
	if len(parts) != 2 {
		return 0, 0
	}
	major, err1 := strconv.Atoi(parts[0])
	minor, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || major < 0 || minor < 0 {
		return 0, 0
	}
	return major, minor
}

// compareRoleVersions 比较两个角色版本号 "X.Y"。
//   - 返回负数表示 a < b
//   - 返回 0 表示 a == b
//   - 返回正数表示 a > b
//
// 使用比较操作符代替减法，避免极端值溢出。
// 空串视为最低（"从未下发过" < "0.0"），非法格式视为 0.0。
func compareRoleVersions(a, b string) int {
	aMajor, aMinor := parseRoleVersion(a)
	bMajor, bMinor := parseRoleVersion(b)
	if aMajor != bMajor {
		if aMajor > bMajor {
			return 1
		}
		return -1
	}
	if aMinor > bMinor {
		return 1
	}
	if aMinor < bMinor {
		return -1
	}
	return 0
}

// validateRoleVersionFormat 校验角色版本号格式必须为 X.Y（两段非负整数）。
// 不允许 v 前缀、不允许三段式（如 1.0.0）、不允许空。
func validateRoleVersionFormat(v string) error {
	if v == "" {
		return common.I18nError(i18n.MsgRoleVersionFormatInvalid)
	}
	parts := strings.SplitN(v, ".", 3)
	if len(parts) != 2 {
		return common.I18nError(i18n.MsgRoleVersionFormatInvalid)
	}
	for _, p := range parts {
		if p == "" {
			return common.I18nError(i18n.MsgRoleVersionFormatInvalid)
		}
		if _, err := strconv.Atoi(p); err != nil {
			return common.I18nError(i18n.MsgRoleVersionFormatInvalid)
		}
	}
	return nil
}
