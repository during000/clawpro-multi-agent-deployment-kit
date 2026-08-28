package common

import "time"

// businessLocation 是应用级业务时区（按 region 注入，如 Asia/Shanghai）。
//
// 单一来源：配额日期汇总（model/llm.go）与定时任务表达式（model/agent_command_schedule.go）
// 统一从这里取，避免各自维护、来源不一致。
//

var businessLocation = time.Local

// BusinessLocation 返回当前业务时区。
func BusinessLocation() *time.Location { return businessLocation }

// SetBusinessTimezone 用 IANA 时区名（如 Asia/Shanghai）设置业务时区。
// 名称为空或非法（zoneinfo 缺失）时保持原值不变。
func SetBusinessTimezone(timezoneName string) {
	if timezoneName == "" {
		return
	}
	if loc, err := time.LoadLocation(timezoneName); err == nil {
		businessLocation = loc
	}
}

// SetBusinessLocation 直接以 *time.Location 设置业务时区，返回 restore 函数用于 defer 恢复。
// 主要用于单测（如需固定偏移 time.FixedZone）；loc 为 nil 时不改变当前值。
func SetBusinessLocation(loc *time.Location) (restore func()) {
	prev := businessLocation
	if loc != nil {
		businessLocation = loc
	}
	return func() { businessLocation = prev }
}
