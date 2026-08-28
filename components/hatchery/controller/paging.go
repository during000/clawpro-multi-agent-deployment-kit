package controller

import "strconv"

// normalizePagingParams 把分页参数（字符串形式）规整成合法范围：
//   - offset: 非负整数；空串/非法值 → "0"
//   - limit:  [1, 100] 的整数；空串/非法/<1 → "20"；>100 → "100"
//
// 返回 (normalized_offset, normalized_limit) 字符串。
// 抽出为独立函数是为了便于单测验证边界行为。
func normalizePagingParams(offsetStr, limitStr string) (string, string) {
	if offsetStr == "" {
		offsetStr = "0"
	}
	if limitStr == "" {
		limitStr = "20"
	}

	// offset：非法或负数 → 0
	offsetVal, err := strconv.Atoi(offsetStr)
	if err != nil || offsetVal < 0 {
		offsetStr = "0"
	}

	// limit：非法或 <1 → 20；>100 → 100
	limitVal, err := strconv.Atoi(limitStr)
	if err != nil || limitVal < 1 {
		limitStr = "20"
	} else if limitVal > 100 {
		limitStr = "100"
	}

	return offsetStr, limitStr
}
