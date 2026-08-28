package common

import (
	"strconv"
	"strings"
)

// PortMatchesRule 判断指定端口是否匹配安全组规则中的端口字段。
// 支持格式：单端口 "8080"、端口范围 "8000-9000"、多端口 "80,443,8080"、全部 "ALL"
func PortMatchesRule(rulePort string, targetPort int) bool {
	rulePort = strings.TrimSpace(rulePort)
	if strings.EqualFold(rulePort, "ALL") {
		return true
	}
	for _, seg := range strings.Split(rulePort, ",") {
		seg = strings.TrimSpace(seg)
		if parts := strings.SplitN(seg, "-", 2); len(parts) == 2 {
			low, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
			high, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err1 == nil && err2 == nil && targetPort >= low && targetPort <= high {
				return true
			}
		} else {
			if p, err := strconv.Atoi(seg); err == nil && p == targetPort {
				return true
			}
		}
	}
	return false
}
