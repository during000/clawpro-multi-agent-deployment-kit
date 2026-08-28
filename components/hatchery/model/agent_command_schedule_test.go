package model

import (
	"testing"
	"time"
)

// mkSchedExpr 构造一个用于 ComputeNextRun 测试的 schedule。
func mkSchedExpr(t *testing.T, expr string) AgentCommandSchedule {
	t.Helper()
	s := AgentCommandSchedule{}
	if err := s.SetScheduleExpr(expr); err != nil {
		t.Fatalf("setup expr %q err: %v", expr, err)
	}
	return s
}

func TestComputeNextRun_Once(t *testing.T) {
	loc := time.Local
	from := time.Date(2026, 6, 23, 12, 0, 0, 0, loc)

	// 未来 → 返回 run_at（精确到分钟）
	s := mkSchedExpr(t, "once(2026-06-30 15:00)")
	next, err := s.ComputeNextRun(from)
	if err != nil {
		t.Fatalf("once future err: %v", err)
	}
	want := time.Date(2026, 6, 30, 15, 0, 0, 0, loc)
	if next == nil || !next.Equal(want) {
		t.Fatalf("once future = %v, want %v", next, want)
	}

	// 已过期 → nil
	s2 := mkSchedExpr(t, "once(2026-06-01 09:00)")
	next2, err := s2.ComputeNextRun(from)
	if err != nil {
		t.Fatalf("once past err: %v", err)
	}
	if next2 != nil {
		t.Fatalf("once past = %v, want nil", next2)
	}
}

func TestComputeNextRun_RateDay(t *testing.T) {
	loc := time.Local
	from := time.Date(2026, 6, 23, 12, 0, 0, 0, loc)

	// 当天 02:00 已过（now=12:00）→ 明天 02:00
	s := mkSchedExpr(t, "every(d, at=02:00)")
	next, err := s.ComputeNextRun(from)
	if err != nil {
		t.Fatalf("rate d err: %v", err)
	}
	want := time.Date(2026, 6, 24, 2, 0, 0, 0, loc)
	if next == nil || !next.Equal(want) {
		t.Fatalf("rate d = %v, want %v", next, want)
	}

	// 当天 18:00 未到（now=12:00）→ 今天 18:00
	s2 := mkSchedExpr(t, "every(d, at=18:00)")
	next2, _ := s2.ComputeNextRun(from)
	want2 := time.Date(2026, 6, 23, 18, 0, 0, 0, loc)
	if next2 == nil || !next2.Equal(want2) {
		t.Fatalf("rate d same-day = %v, want %v", next2, want2)
	}
}

func TestComputeNextRun_RateWeek(t *testing.T) {
	loc := time.Local
	// 2026-06-23 是周二。要求每周一 09:00 → 下一个周一 2026-06-29 09:00
	from := time.Date(2026, 6, 23, 12, 0, 0, 0, loc)
	s := mkSchedExpr(t, "every(w, on=1, at=09:00)")
	next, err := s.ComputeNextRun(from)
	if err != nil {
		t.Fatalf("rate w err: %v", err)
	}
	want := time.Date(2026, 6, 29, 9, 0, 0, 0, loc)
	if next == nil || !next.Equal(want) {
		t.Fatalf("rate w Mon = %v (weekday %v), want %v", next, next.Weekday(), want)
	}

	// 周日（7）→ 2026-06-28 09:00
	s2 := mkSchedExpr(t, "every(w, on=7, at=09:00)")
	next2, _ := s2.ComputeNextRun(from)
	want2 := time.Date(2026, 6, 28, 9, 0, 0, 0, loc)
	if next2 == nil || !next2.Equal(want2) {
		t.Fatalf("rate w Sun = %v (weekday %v), want %v", next2, next2.Weekday(), want2)
	}
}

func TestComputeNextRun_RateMonth(t *testing.T) {
	loc := time.Local
	from := time.Date(2026, 6, 23, 12, 0, 0, 0, loc)

	// 每月 1 号 00:00 → 7-1 00:00
	s := mkSchedExpr(t, "every(m, on=1, at=00:00)")
	next, err := s.ComputeNextRun(from)
	if err != nil {
		t.Fatalf("rate m err: %v", err)
	}
	want := time.Date(2026, 7, 1, 0, 0, 0, 0, loc)
	if next == nil || !next.Equal(want) {
		t.Fatalf("rate m day1 = %v, want %v", next, want)
	}

	// 每月 25 号 10:00（当月 25 号未到）→ 6-25 10:00
	s2 := mkSchedExpr(t, "every(m, on=25, at=10:00)")
	next2, _ := s2.ComputeNextRun(from)
	want2 := time.Date(2026, 6, 25, 10, 0, 0, 0, loc)
	if next2 == nil || !next2.Equal(want2) {
		t.Fatalf("rate m day25 = %v, want %v", next2, want2)
	}
}

func TestComputeNextRun_RateMonth_SkipMonthsWithoutDay(t *testing.T) {
	loc := time.Local
	// 2026-01-31 之后，要求每月 31 号 → 2 月没有 31 号 → 跳过无 31 号的月份，下一个是 3-31
	from := time.Date(2026, 1, 31, 23, 0, 0, 0, loc)
	s := mkSchedExpr(t, "every(m, on=31, at=08:00)")
	next, err := s.ComputeNextRun(from)
	if err != nil {
		t.Fatalf("rate m skip err: %v", err)
	}
	want := time.Date(2026, 3, 31, 8, 0, 0, 0, loc)
	if next == nil || !next.Equal(want) {
		t.Fatalf("rate m31 skip = %v, want %v", next, want)
	}
}

func TestSetScheduleExpr_Errors(t *testing.T) {
	bad := []string{
		"",
		"weekly",                     // 无括号
		"every(02:00)",                // 缺单位（首段非 d/w/m）
		"every(dd, at=02:00)",         // 非法单位
		"every(1d, at=02:00)",         // 不再支持数字前缀
		"every(week, on=1, at=09:00)", // 单位不支持全写
		"every(w, on=Mon, at=09:00)",  // on 不支持英文
		"every(w, on=8, at=09:00)",    // 周几越界
		"every(d, at=99:99)",          // 时间非法
		"every(m, on=32, at=08:00)",   // 几号越界
		"every(w, at=09:00)",          // 周级缺 on
		"every(m, at=09:00)",          // 月级缺 on
		"every(d)",                    // 缺 at
		"once()",                     // 缺时间
		"once(not-a-time)",           // 时间非法
		"foo(d, at=02:00)",           // 未知关键字
	}
	for _, expr := range bad {
		s := AgentCommandSchedule{}
		if err := s.SetScheduleExpr(expr); err == nil {
			t.Fatalf("expr %q expected error, got nil", expr)
		}
	}
}

func TestSetScheduleExpr_Canonical(t *testing.T) {
	cases := map[string]string{
		"  every(d, at=2:00)  ":      "every(d, at=02:00)",       // trim + 补零
		"EVERY(w, on=1, at=9:5)":     "every(w, on=1, at=09:05)", // 关键字大小写
		"every(m, on=1, at=00:00)":   "every(m, on=1, at=00:00)",
		"once(2026-06-30 15:00)":    "once(2026-06-30 15:00)",
		"once(2026-06-30T15:00:00)": "once(2026-06-30 15:00)", // 秒被截断到分钟
	}
	for in, want := range cases {
		s := AgentCommandSchedule{}
		if err := s.SetScheduleExpr(in); err != nil {
			t.Fatalf("%q err: %v", in, err)
		}
		if s.ScheduleExpr != want {
			t.Fatalf("canonical(%q) = %q, want %q", in, s.ScheduleExpr, want)
		}
	}
}

func TestValidateAndNormalize(t *testing.T) {
	base := func() AgentCommandSchedule {
		s := AgentCommandSchedule{Name: "t", CommandID: 1, ScheduleExpr: "every(d, at=09:00)"}
		_ = s.SetInstanceIDs([]uint{1})
		return s
	}

	// 合法
	ok := base()
	if err := ok.ValidateAndNormalize(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 名称为空
	s := base()
	s.Name = ""
	if err := s.ValidateAndNormalize(); err == nil {
		t.Fatalf("expected error for empty name")
	}

	// 名称超长（>32）
	s = base()
	s.Name = "012345678901234567890123456789012" // 33 chars
	if err := s.ValidateAndNormalize(); err == nil {
		t.Fatalf("expected error for name too long")
	}

	// 描述超长（>64）
	s = base()
	long := make([]byte, 65)
	for i := range long {
		long[i] = 'a'
	}
	s.Description = string(long)
	if err := s.ValidateAndNormalize(); err == nil {
		t.Fatalf("expected error for description too long")
	}

	// 无目标 agent
	s = base()
	_ = s.SetInstanceIDs([]uint{})
	if err := s.ValidateAndNormalize(); err == nil {
		t.Fatalf("expected error for no instances")
	}

	// 非法表达式
	s = base()
	s.ScheduleExpr = "weekly"
	if err := s.ValidateAndNormalize(); err == nil {
		t.Fatalf("expected error for bad expr")
	}
}

func TestScheduleStatus(t *testing.T) {
	ranAt := time.Now()
	cases := []struct {
		name string
		s    AgentCommandSchedule
		want string
	}{
		{"once 完成（即使已自动停用）", AgentCommandSchedule{ScheduleExpr: "once(2026-06-30 15:00)", Enabled: false, LastRunAt: &ranAt}, ScheduleStatusCompleted},
		{"once 完成优先于执行中", AgentCommandSchedule{ScheduleExpr: "once(2026-06-30 15:00)", Enabled: false, LastRunAt: &ranAt, IsRunning: true}, ScheduleStatusCompleted},
		{"rate 执行中", AgentCommandSchedule{ScheduleExpr: "every(d, at=09:00)", Enabled: true, LastRunAt: &ranAt, IsRunning: true}, ScheduleStatusRunning},
		{"rate 暂停", AgentCommandSchedule{ScheduleExpr: "every(d, at=09:00)", Enabled: false, LastRunAt: &ranAt}, ScheduleStatusPaused},
		{"rate 未开始", AgentCommandSchedule{ScheduleExpr: "every(d, at=09:00)", Enabled: true}, ScheduleStatusPending},
		{"rate 待执行", AgentCommandSchedule{ScheduleExpr: "every(d, at=09:00)", Enabled: true, LastRunAt: &ranAt}, ScheduleStatusWaiting},
		{"once 未执行被暂停", AgentCommandSchedule{ScheduleExpr: "once(2026-06-30 15:00)", Enabled: false}, ScheduleStatusPaused},
		{"once 未执行待触发", AgentCommandSchedule{ScheduleExpr: "once(2026-06-30 15:00)", Enabled: true}, ScheduleStatusPending},
		{"interval 已过 end（next 为空）→ 完成", AgentCommandSchedule{ScheduleExpr: "interval(1m, begin=2026-06-30 15:00)", Enabled: false, LastRunAt: &ranAt}, ScheduleStatusCompleted},
		{"interval 已过 end 即使在执行也算完成", AgentCommandSchedule{ScheduleExpr: "interval(1m, begin=2026-06-30 15:00)", Enabled: false, LastRunAt: &ranAt, IsRunning: true}, ScheduleStatusCompleted},
		{"interval 执行中（next 有值）", AgentCommandSchedule{ScheduleExpr: "interval(1m, begin=2026-06-30 15:00)", Enabled: true, NextRunAt: &ranAt, LastRunAt: &ranAt, IsRunning: true}, ScheduleStatusRunning},
		{"interval 待执行（next 有值）", AgentCommandSchedule{ScheduleExpr: "interval(1m, begin=2026-06-30 15:00)", Enabled: true, NextRunAt: &ranAt, LastRunAt: &ranAt}, ScheduleStatusWaiting},
	}
	for _, c := range cases {
		if got := c.s.Status(); got != c.want {
			t.Errorf("%s: Status() = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestComputeNextRun_Cron(t *testing.T) {
	loc := time.Local
	// 2026-06-23 是周二 12:02
	from := time.Date(2026, 6, 23, 12, 2, 0, 0, loc)

	cases := []struct {
		expr string
		want time.Time
		desc string
	}{
		{"cron(*/5 * * * *)", time.Date(2026, 6, 23, 12, 5, 0, 0, loc), "每 5 分钟 → 12:05"},
		{"cron(0 9 * * 1-5)", time.Date(2026, 6, 24, 9, 0, 0, 0, loc), "工作日 9 点 → 周三 9:00"},
		{"cron(0 9 * * 1)", time.Date(2026, 6, 29, 9, 0, 0, 0, loc), "周一 9 点 → 下周一 9:00（1=周一）"},
		{"cron(0 9 * * 0)", time.Date(2026, 6, 28, 9, 0, 0, 0, loc), "周日 9 点 → 本周日 9:00（0=周日）"},
		{"cron(30 8 1 * *)", time.Date(2026, 7, 1, 8, 30, 0, 0, loc), "每月 1 号 8:30 → 7-1 8:30"},
	}
	for _, c := range cases {
		s := mkSchedExpr(t, c.expr)
		next, err := s.ComputeNextRun(from)
		if err != nil {
			t.Fatalf("%s: err %v", c.desc, err)
		}
		if next == nil || !next.Equal(c.want) {
			t.Fatalf("%s: cron %q next = %v (weekday %v), want %v", c.desc, c.expr, next, next.Weekday(), c.want)
		}
	}
}

func TestSetScheduleExpr_CronCanonical(t *testing.T) {
	cases := map[string]string{
		"cron(*/5 * * * *)":      "cron(*/5 * * * *)",
		"  cron(*/5   *  * * *)": "cron(*/5 * * * *)", // trim + 多空格归一
		"CRON(0 9 * * 1-5)":      "cron(0 9 * * 1-5)", // 关键字大小写
	}
	for in, want := range cases {
		s := AgentCommandSchedule{}
		if err := s.SetScheduleExpr(in); err != nil {
			t.Fatalf("%q err: %v", in, err)
		}
		if s.ScheduleExpr != want {
			t.Fatalf("canonical(%q) = %q, want %q", in, s.ScheduleExpr, want)
		}
	}
}

func TestSetScheduleExpr_CronErrors(t *testing.T) {
	bad := []string{
		"cron()",            // 空
		"cron(* * * *)",     // 仅 4 字段
		"cron(* * * * * *)", // 6 字段
		"cron(60 * * * *)",  // 分钟越界
		"cron(* 24 * * *)",  // 小时越界
		"cron(* * 32 * *)",  // 日越界
		"cron(* * * 13 *)",  // 月越界
		"cron(* * * * 7)",   // 周越界（标准 0-6，7 不接受）
		"cron(abc * * * *)", // 非法字符
	}
	for _, expr := range bad {
		s := AgentCommandSchedule{}
		if err := s.SetScheduleExpr(expr); err == nil {
			t.Fatalf("expr %q expected error, got nil", expr)
		}
	}
}

func TestComputeNextRun_Interval(t *testing.T) {
	loc := time.Local

	cases := []struct {
		expr string
		from time.Time
		want *time.Time
		desc string
	}{
		{
			"interval(1h, begin=2026-06-23 15:00)",
			time.Date(2026, 6, 23, 12, 0, 0, 0, loc),
			ptrTime(time.Date(2026, 6, 23, 15, 0, 0, 0, loc)),
			"begin 在未来 → 首次即 begin",
		},
		{
			"interval(1h, begin=2026-06-23 10:00)",
			time.Date(2026, 6, 23, 12, 30, 0, 0, loc),
			ptrTime(time.Date(2026, 6, 23, 13, 0, 0, 0, loc)),
			"begin 已过，每小时 → 13:00",
		},
		{
			"interval(1m, begin=2026-06-23 12:00)",
			time.Date(2026, 6, 23, 12, 0, 30, 0, loc),
			ptrTime(time.Date(2026, 6, 23, 12, 1, 0, 0, loc)),
			"每分钟 → 12:01",
		},
		{
			"interval(1d, begin=2026-06-20 09:00)",
			time.Date(2026, 6, 23, 12, 0, 0, 0, loc),
			ptrTime(time.Date(2026, 6, 24, 9, 0, 0, 0, loc)),
			"每天 → 次日 09:00",
		},
		{
			"interval(1h, begin=2026-06-23 10:00, end=2026-06-23 12:00)",
			time.Date(2026, 6, 23, 12, 30, 0, 0, loc),
			nil,
			"超过 end → 无后续触发",
		},
	}
	for _, c := range cases {
		s := mkSchedExpr(t, c.expr)
		next, err := s.ComputeNextRun(c.from)
		if err != nil {
			t.Fatalf("%s: err %v", c.desc, err)
		}
		switch {
		case c.want == nil && next != nil:
			t.Fatalf("%s: next = %v, want nil", c.desc, next)
		case c.want != nil && (next == nil || !next.Equal(*c.want)):
			t.Fatalf("%s: interval %q next = %v, want %v", c.desc, c.expr, next, *c.want)
		}
	}
}

func TestSetScheduleExpr_IntervalCanonical(t *testing.T) {
	cases := map[string]string{
		"interval(1m, begin=2026-06-30 15:00)":                       "interval(1m, begin=2026-06-30 15:00)",
		"  INTERVAL(2h , begin=2026-06-30T15:00 )  ":                 "interval(2h, begin=2026-06-30 15:00)", // trim + 大小写 + T 分隔
		"interval(1d, begin=2026-06-30 15:00, end=2026-07-30 15:00)": "interval(1d, begin=2026-06-30 15:00, end=2026-07-30 15:00)",
	}
	for in, want := range cases {
		s := AgentCommandSchedule{}
		if err := s.SetScheduleExpr(in); err != nil {
			t.Fatalf("%q err: %v", in, err)
		}
		if s.ScheduleExpr != want {
			t.Fatalf("canonical(%q) = %q, want %q", in, s.ScheduleExpr, want)
		}
	}
}

func TestSetScheduleExpr_IntervalErrors(t *testing.T) {
	bad := []string{
		"interval(1m)",                         // 缺 begin
		"interval(begin=2026-06-30 15:00)",     // 缺间隔
		"interval(0m, begin=2026-06-30 15:00)", // 间隔为 0
		"interval(1x, begin=2026-06-30 15:00)", // 非法单位
		"interval(1m, begin=not-a-time)",       // begin 非法
		"interval(1m, end=2026-06-30 15:00)",   // 只有 end 无 begin
		"interval(1m, begin=2026-06-30 15:00, end=2026-06-29 15:00)", // end 早于 begin
	}
	for _, expr := range bad {
		s := AgentCommandSchedule{}
		if err := s.SetScheduleExpr(expr); err == nil {
			t.Fatalf("expr %q expected error, got nil", expr)
		}
	}
}
