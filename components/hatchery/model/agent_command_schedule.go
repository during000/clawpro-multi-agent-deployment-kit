package model

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	hcommon "hatchery/common"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

// ============================================================================
// AgentCommandSchedule —— 命令模板的定时任务配置
//
// 一条 schedule = "何时触发一次 dispatch" 的配置。它不直接执行命令，而是由后台
// runner（task/agent_command_schedule_runner.go）扫描到期行后调用 controller.startDispatch
// 触发一次普通 dispatch，执行链路 / 详情 / 状态聚合完全复用既有 dispatch 体系。
//
// 调度规格用单字符串表达式 schedule_expr 存储（不再拆成结构化列），运行时由
// ComputeNextRun 直接解析。语法为函数式参数化风格（参考 AWS EventBridge rate(...)）：
//   once(<time>)                     例: once(2026-06-30 15:00)（精确到分钟）
//   every(d, at=<HH:MM>)             例: every(d, at=02:00)（每天）
//   every(w, on=<1-7>, at=<HH:MM>)   例: every(w, on=1, at=09:00)（每周，1=周一..7=周日）
//   every(m, on=<1-31>, at=<HH:MM>)  例: every(m, on=1, at=09:00)（每月，无该日的月份整月跳过）
//   cron(<分 时 日 月 周>)            例: cron(*/5 * * * *)（标准 5 字段；周 0-6，0=周日..6=周六）
//   interval(<n><m|h|d>, begin=<time>[, end=<time>])  例: interval(1m, begin=2026-06-30 15:00)（从 begin 起每隔 N 触发）
//
// 注意：every 与 cron 是两套独立解析体系，周字段约定不同 —— every 用 1-7（1=周一..7=周日），
// cron 沿用标准 crontab 的 0-6（0=周日..6=周六，不接受 7）。两者互不复用解析逻辑。
//
// 关键字大小写不敏感；单位只用缩写 d/w/m，on 与几号均为数字。
// 存储与回显统一为 canonical 形态（小写关键字、HH:MM 补零）。
// next_run_at 是调度扫描依据（带 (enabled,next_run_at) 联合索引），由 ComputeNextRun 计算。
// is_running 标记当前是否有一次 dispatch 在执行：触发时置 true，由 1s 的 reconcile 任务
// 扫描 last_dispatch_slug 对应 dispatch 是否终态后订正为 false；展示态由 Status() 合成。
// ============================================================================

// schedule 类型枚举（内部值，不落库；schedule_expr 才是持久化真相源）
const (
	ScheduleTypeOnce     = "once"
	ScheduleTypeEvery    = "every"
	ScheduleTypeCron     = "cron"
	ScheduleTypeInterval = "interval"
)

// cronParser 标准 5 字段 cron 解析器（分 时 日 月 周）。
// 周字段 robfig 语义：0-6，0=周日、1=周一..6=周六（不接受 7）。
var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

// 周期单位枚举
const (
	ScheduleUnitDay   = "d"
	ScheduleUnitWeek  = "w"
	ScheduleUnitMonth = "m"
)

// MaxAgentCommandSchedulesPerTenant 单租户定时任务数量上限（含停用，不含软删）。
const MaxAgentCommandSchedulesPerTenant = 1000

// 合成展示状态（由 Status() 计算 / ScheduleStatusCondition 下推 SQL 筛选）。
const (
	ScheduleStatusRunning   = "running"   // 执行中：当前有 dispatch 在跑
	ScheduleStatusCompleted = "completed" // 已完成：once 已执行过（终态）
	ScheduleStatusPaused    = "paused"    // 已暂停：被主动停用
	ScheduleStatusPending   = "pending"   // 未开始：从未触发过
	ScheduleStatusWaiting   = "waiting"   // 待执行：执行过、等待下次触发
)

// 名称 / 描述字符上限。
const (
	AgentCommandScheduleNameMaxChars = 32
	AgentCommandScheduleDescMaxChars = 64
)

// AgentCommandScheduleSlugPrefix 定时任务对外资源 ID 前缀，与 cmd-/task- 风格对齐。
const AgentCommandScheduleSlugPrefix = "sch-"

// AgentCommandScheduleSlugRandLen slug 随机部分长度（"sch-{8 位随机}"）。
const AgentCommandScheduleSlugRandLen = 8

var (
	ErrScheduleNotFound     = errors.New("agent command schedule not found")
	ErrScheduleSpecInvalid  = errors.New("agent command schedule spec invalid")
	ErrScheduleSlugConflict = errors.New("agent command schedule slug conflict")
)

// AgentCommandSchedule 定时任务配置。软删（与 agent_commands 一致），保留审计。
type AgentCommandSchedule struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Identifier  string `gorm:"uniqueIndex:idx_sched_ident_slug,priority:1;index;default:''" json:"-"`
	Slug        string `gorm:"uniqueIndex:idx_sched_ident_slug,priority:2;type:varchar(32);not null;default:''" json:"slug"`
	Name        string `gorm:"type:varchar(32);not null;default:''" json:"name"`
	Description string `gorm:"type:varchar(64);not null;default:''" json:"description"`

	CommandID       uint   `gorm:"not null;default:0;index" json:"command_id"`
	InstanceIDsJSON string `gorm:"type:varchar(8192);not null;default:'[]'" json:"-"`
	ParamValuesJSON string `gorm:"type:varchar(4096);not null;default:'{}'" json:"-"`

	// 调度规格：单字符串表达式（时间一律按服务器本地时区解释）
	ScheduleExpr string `gorm:"type:varchar(64);not null;default:''" json:"schedule"`

	// ===== 运行态 =====
	Enabled          bool       `gorm:"not null;default:true;index:idx_sched_due,priority:1" json:"enabled"`
	NextRunAt        *time.Time `gorm:"default:null;index:idx_sched_due,priority:2" json:"next_run_at"`
	FirstRunAt       *time.Time `gorm:"default:null" json:"first_run_at"`
	LastRunAt        *time.Time `gorm:"default:null" json:"last_run_at"`
	LastDispatchSlug string     `gorm:"type:varchar(32);not null;default:''" json:"last_dispatch_slug"`
	IsRunning        bool       `gorm:"not null;default:false" json:"is_running"`
	LastError        string     `gorm:"type:varchar(512);not null;default:''" json:"last_error"`
	CreatedByUserID  uint       `gorm:"not null;default:0;index" json:"created_by_user_id"`
}

func (AgentCommandSchedule) TableName() string { return "agent_command_schedules" }

// ============================================================================
// JSON 字段辅助
// ============================================================================

// InstanceIDsList 反序列化目标实例 ID 列表。
func (s *AgentCommandSchedule) InstanceIDsList() []uint {
	if strings.TrimSpace(s.InstanceIDsJSON) == "" {
		return nil
	}
	var ids []uint
	if err := json.Unmarshal([]byte(s.InstanceIDsJSON), &ids); err != nil {
		return nil
	}
	return ids
}

// SetInstanceIDs 序列化目标实例 ID 列表。
func (s *AgentCommandSchedule) SetInstanceIDs(ids []uint) error {
	if ids == nil {
		ids = []uint{}
	}
	b, err := json.Marshal(ids)
	if err != nil {
		return fmt.Errorf("marshal instance_ids: %w", err)
	}
	s.InstanceIDsJSON = string(b)
	return nil
}

// ParamValuesMap 反序列化参数取值。
func (s *AgentCommandSchedule) ParamValuesMap() map[string]string {
	if strings.TrimSpace(s.ParamValuesJSON) == "" {
		return map[string]string{}
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(s.ParamValuesJSON), &m); err != nil || m == nil {
		return map[string]string{}
	}
	return m
}

// SetParamValues 序列化参数取值。
func (s *AgentCommandSchedule) SetParamValues(m map[string]string) error {
	if m == nil {
		m = map[string]string{}
	}
	b, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal param_values: %w", err)
	}
	s.ParamValuesJSON = string(b)
	return nil
}

// SetScheduleExpr 校验调度表达式并以 canonical 形态写入（统一大小写/空格/补零）。
//
// once 时间按服务器本地时区解释。
func (s *AgentCommandSchedule) SetScheduleExpr(expr string) error {
	sp, err := parseScheduleExpr(expr)
	if err != nil {
		return err
	}
	s.ScheduleExpr = sp.canonical()
	return nil
}

// ============================================================================
// 校验与归一化
// ============================================================================

// ValidateAndNormalize 校验整条 schedule（名称/描述/命令/目标/表达式）。
func (s *AgentCommandSchedule) ValidateAndNormalize() error {
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("%w: 名称必填", ErrScheduleSpecInvalid)
	}
	if len([]rune(s.Name)) > AgentCommandScheduleNameMaxChars {
		return fmt.Errorf("%w: 名称最长 %d 字符", ErrScheduleSpecInvalid, AgentCommandScheduleNameMaxChars)
	}
	if len([]rune(s.Description)) > AgentCommandScheduleDescMaxChars {
		return fmt.Errorf("%w: 描述最长 %d 字符", ErrScheduleSpecInvalid, AgentCommandScheduleDescMaxChars)
	}
	if s.CommandID == 0 {
		return fmt.Errorf("%w: command_id 必填", ErrScheduleSpecInvalid)
	}
	if len(s.InstanceIDsList()) == 0 {
		return fmt.Errorf("%w: 至少选择 1 台 Agent", ErrScheduleSpecInvalid)
	}
	if _, err := parseScheduleExpr(s.ScheduleExpr); err != nil {
		return err
	}
	return nil
}

// ComputeNextRun 计算严格晚于 from 的下一次触发时刻。
//
// 返回 nil 表示没有未来触发（once 已过期）。运行时解析 schedule_expr。
func (s *AgentCommandSchedule) ComputeNextRun(from time.Time) (*time.Time, error) {
	spec, err := parseScheduleExpr(s.ScheduleExpr)
	if err != nil {
		return nil, err
	}
	return spec.nextRun(from)
}

// ============================================================================
// 合成展示状态
// ============================================================================

// IsOnce 是否为一次性任务（canonical 形态下 once 表达式以 "once(" 开头）。
func (s *AgentCommandSchedule) IsOnce() bool {
	return strings.HasPrefix(s.ScheduleExpr, ScheduleTypeOnce+"(")
}

// IsInterval 是否为固定间隔任务（canonical 形态下以 "interval(" 开头）。
func (s *AgentCommandSchedule) IsInterval() bool {
	return strings.HasPrefix(s.ScheduleExpr, ScheduleTypeInterval+"(")
}

// Status 计算合成展示状态。优先级（短路）：
//
//	completed：终态（最高，需先于 paused 判断避免被误判为暂停）——
//	           once 已执行（last_run_at 非空）；或 interval 已过 end（next_run_at 被置空，不再触发）
//	running  ：当前有 dispatch 在执行（is_running）
//	paused   ：被主动停用
//	pending  ：从未触发
//	waiting  ：执行过、等待下次触发
//
// 该逻辑与 ScheduleStatusCondition 的 SQL 下推保持一致。
func (s *AgentCommandSchedule) Status() string {
	switch {
	case s.IsOnce() && s.LastRunAt != nil:
		return ScheduleStatusCompleted
	case s.IsInterval() && s.NextRunAt == nil:
		return ScheduleStatusCompleted
	case s.IsRunning:
		return ScheduleStatusRunning
	case !s.Enabled:
		return ScheduleStatusPaused
	case s.LastRunAt == nil:
		return ScheduleStatusPending
	default:
		return ScheduleStatusWaiting
	}
}

// ScheduleStatusCondition 把合成状态翻译为可下推 SQL 的 WHERE 子句（与 Status() 等价、互斥）。
// ok=false 表示 status 非法（调用方应忽略该筛选）。
//
// 终态 completedExpr：once 已执行（schedule_expr LIKE 'once(%' 且 last_run_at 非空）
// 或 interval 已过 end（schedule_expr LIKE 'interval(%' 且 next_run_at 为空）。
// 其余状态均以 "NOT completedExpr" 排除终态，严格镜像 Status() 的短路优先级。
func ScheduleStatusCondition(status string) (cond string, args []any, ok bool) {
	const onceLike = "once(%"
	const intervalLike = "interval(%"
	const completedExpr = "((schedule_expr LIKE ? AND last_run_at IS NOT NULL) OR (schedule_expr LIKE ? AND next_run_at IS NULL))"
	completedArgs := []any{onceLike, intervalLike}
	switch status {
	case ScheduleStatusCompleted:
		return completedExpr, completedArgs, true
	case ScheduleStatusRunning:
		return "is_running = ? AND NOT " + completedExpr, append([]any{true}, completedArgs...), true
	case ScheduleStatusPaused:
		return "enabled = ? AND is_running = ? AND NOT " + completedExpr, append([]any{false, false}, completedArgs...), true
	case ScheduleStatusPending:
		return "last_run_at IS NULL AND is_running = ? AND enabled = ? AND NOT " + completedExpr,
			append([]any{false, true}, completedArgs...), true
	case ScheduleStatusWaiting:
		return "last_run_at IS NOT NULL AND is_running = ? AND enabled = ? AND NOT " + completedExpr,
			append([]any{false, true}, completedArgs...), true
	}
	return "", nil, false
}

// ============================================================================
// 表达式解析（运行时，不落库）
// ============================================================================

// scheduleSpec 解析后的内部表示（不持久化）。
type scheduleSpec struct {
	typ        string
	unit       string
	hh, mm     int
	dayOfWeek  int // 1=Mon..7=Sun
	dayOfMonth int // 1..31
	runAt      *time.Time
	cronExpr   string        // typ==cron：归一化后的 5 字段表达式（单空格分隔）
	cronSched  cron.Schedule // typ==cron：解析后的调度器，用于 Next 计算

	intervalN    int           // typ==interval：间隔数字
	intervalUnit string        // typ==interval：间隔单位 m/h/d
	intervalStep time.Duration // typ==interval：间隔时长（n×unit）
	begin        *time.Time    // typ==interval：首次触发时刻（必填）
	end          *time.Time    // typ==interval：截止时刻（可选，晚于此不再触发）
}

// scheduleOnceLayouts once 时间支持的解析格式（按序尝试）。canonical 统一截断到分钟。
var scheduleOnceLayouts = []string{
	"2006-01-02 15:04",
	"2006-01-02T15:04",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05",
	time.RFC3339,
}

// errSchedExpr 构造包装 ErrScheduleSpecInvalid 的调度表达式错误。
func errSchedExpr(msg string) error {
	return fmt.Errorf("%w: %s", ErrScheduleSpecInvalid, msg)
}

// parseScheduleExpr 直接解析函数式调度表达式为内部 scheduleSpec（keyword(inner) 形态）。
//
//	once(2026-06-30 15:00)              一次性，精确到分钟
//	every(d, at=02:00)                  每天
//	every(w, on=1, at=09:00)            每周的周几（on: 1=Mon..7=Sun）
//	every(m, on=1, at=09:00)            每月的几号（on: 1..31，无该日的月份整月跳过）
//	cron(*/5 * * * *)                   标准 5 字段 cron：分 时 日 月 周（周 0-6，0=周日..6=周六）
//	interval(1m, begin=2026-06-30 15:00 [, end=...])  从 begin 起每隔 N（单位 m/h/d）触发，可选 end 截止
//
// once / interval 的时间均按业务时区（common.BusinessLocation）解释。失败返回包装 ErrScheduleSpecInvalid 的错误。
func parseScheduleExpr(expr string) (*scheduleSpec, error) {
	raw := strings.TrimSpace(expr)
	open := strings.IndexByte(raw, '(')
	if open <= 0 || !strings.HasSuffix(raw, ")") {
		return nil, errSchedExpr("格式应为 once(2006-01-02 15:04) / every(d, at=02:00) / every(w, on=1, at=09:00) / every(m, on=1, at=09:00) / cron(*/5 * * * *) / interval(1m, begin=2006-01-02 15:04)")
	}
	keyword := strings.ToLower(strings.TrimSpace(raw[:open]))
	inner := strings.TrimSpace(raw[open+1 : len(raw)-1])

	switch keyword {
	case ScheduleTypeOnce:
		t, ok := parseOnceTime(inner)
		if !ok {
			return nil, errSchedExpr(fmt.Sprintf("无法解析时间 %q（精确到分钟，如 2026-06-30 15:00）", inner))
		}
		return &scheduleSpec{typ: ScheduleTypeOnce, runAt: &t}, nil
	case ScheduleTypeEvery:
		return parseEveryExpr(inner)
	case ScheduleTypeCron:
		return parseCronExpr(inner)
	case ScheduleTypeInterval:
		return parseIntervalExpr(inner)
	}
	return nil, errSchedExpr(fmt.Sprintf("未知关键字 %q（应为 once / every / cron / interval）", keyword))
}

// parseCronExpr 解析 cron(...) 括号内的标准 5 字段表达式（分 时 日 月 周）。
// 归一化为单空格分隔后用 robfig/cron 校验；周字段 0-6（0=周日，1=周一..6=周六）。
func parseCronExpr(inner string) (*scheduleSpec, error) {
	fields := strings.Fields(inner)
	if len(fields) != 5 {
		return nil, errSchedExpr("cron 需为标准 5 字段：分 时 日 月 周，如 cron(*/5 * * * *) / cron(0 9 * * 1-5)")
	}
	norm := strings.Join(fields, " ")
	sched, err := cronParser.Parse(norm)
	if err != nil {
		return nil, errSchedExpr(fmt.Sprintf("非法 cron 表达式 %q：%v", norm, err))
	}
	return &scheduleSpec{typ: ScheduleTypeCron, cronExpr: norm, cronSched: sched}, nil
}

// parseEveryExpr 解析 every(...) 括号内参数：首段为单位 d|w|m，其余为 key=value（at / on）。
func parseEveryExpr(inner string) (*scheduleSpec, error) {
	parts := strings.Split(inner, ",")
	unit := strings.ToLower(strings.TrimSpace(parts[0]))

	// 其余 key=value（at / on）
	kv := make(map[string]string, len(parts)-1)
	for _, p := range parts[1:] {
		k, v, ok := strings.Cut(p, "=")
		if !ok {
			return nil, errSchedExpr("参数需为 key=value 形式")
		}
		kv[strings.ToLower(strings.TrimSpace(k))] = strings.TrimSpace(v)
	}

	hh, mm, herr := parseHHMM(kv["at"])
	if herr != nil {
		return nil, errSchedExpr("缺少或非法 at=HH:MM")
	}
	sp := &scheduleSpec{typ: ScheduleTypeEvery, unit: unit, hh: hh, mm: mm}

	switch unit {
	case ScheduleUnitDay:
		// 仅需 at
	case ScheduleUnitWeek:
		on, derr := strconv.Atoi(kv["on"])
		if derr != nil || on < 1 || on > 7 {
			return nil, errSchedExpr("周级需 on=1..7（1=周一..7=周日）")
		}
		sp.dayOfWeek = on
	case ScheduleUnitMonth:
		on, derr := strconv.Atoi(kv["on"])
		if derr != nil || on < 1 || on > 31 {
			return nil, errSchedExpr("月级需 on=1..31")
		}
		sp.dayOfMonth = on
	default:
		return nil, errSchedExpr("周期单位需为 d/w/m")
	}
	return sp, nil
}

// parseIntervalExpr 解析 interval(...)：首段为 <n><unit>（unit: m/h/d），其余 key=value（begin 必填、end 可选）。
// begin/end 为绝对时刻（YYYY-MM-DD HH:MM），按业务时区解释；从 begin 起每隔 N 触发，超过 end 不再触发。
func parseIntervalExpr(inner string) (*scheduleSpec, error) {
	parts := strings.Split(inner, ",")
	n, unit, dur, derr := parseIntervalEvery(strings.TrimSpace(parts[0]))
	if derr != nil {
		return nil, derr
	}

	kv := make(map[string]string, len(parts)-1)
	for _, p := range parts[1:] {
		k, v, ok := strings.Cut(p, "=")
		if !ok {
			return nil, errSchedExpr("参数需为 key=value 形式（begin=... [, end=...]）")
		}
		kv[strings.ToLower(strings.TrimSpace(k))] = strings.TrimSpace(v)
	}

	beginStr := kv["begin"]
	if beginStr == "" {
		return nil, errSchedExpr("interval 需指定 begin=YYYY-MM-DD HH:MM")
	}
	begin, ok := parseOnceTime(beginStr)
	if !ok {
		return nil, errSchedExpr(fmt.Sprintf("非法 begin 时间 %q（如 2026-06-30 15:00）", beginStr))
	}
	sp := &scheduleSpec{typ: ScheduleTypeInterval, intervalN: n, intervalUnit: unit, intervalStep: dur, begin: &begin}

	if endStr := kv["end"]; endStr != "" {
		end, ok := parseOnceTime(endStr)
		if !ok {
			return nil, errSchedExpr(fmt.Sprintf("非法 end 时间 %q（如 2026-06-30 15:00）", endStr))
		}
		if !end.After(begin) {
			return nil, errSchedExpr("end 必须晚于 begin")
		}
		sp.end = &end
	}
	return sp, nil
}

// parseIntervalEvery 解析 "<n><unit>"：unit ∈ {m,h,d}，n 为正整数；返回 n、unit 及对应 duration。
func parseIntervalEvery(s string) (int, string, time.Duration, error) {
	if len(s) < 2 {
		return 0, "", 0, errSchedExpr("间隔需为 <数字><单位>，单位 m/h/d，如 1m / 2h / 1d")
	}
	unit := strings.ToLower(s[len(s)-1:])
	n, err := strconv.Atoi(s[:len(s)-1])
	if err != nil || n <= 0 {
		return 0, "", 0, errSchedExpr("间隔数字需为正整数，如 1m / 2h / 1d")
	}
	var base time.Duration
	switch unit {
	case "m":
		base = time.Minute
	case "h":
		base = time.Hour
	case "d":
		base = 24 * time.Hour
	default:
		return 0, "", 0, errSchedExpr("间隔单位需为 m(分)/h(时)/d(天)")
	}
	return n, unit, time.Duration(n) * base, nil
}

// parseHHMM 解析 "HH:MM"（时 0-23，分 0-59）。
func parseHHMM(s string) (hh, mm int, err error) {
	parts := strings.Split(strings.TrimSpace(s), ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid time %q", s)
	}
	hh, err = strconv.Atoi(parts[0])
	if err != nil || hh < 0 || hh > 23 {
		return 0, 0, fmt.Errorf("invalid hour %q", s)
	}
	mm, err = strconv.Atoi(parts[1])
	if err != nil || mm < 0 || mm > 59 {
		return 0, 0, fmt.Errorf("invalid minute %q", s)
	}
	return hh, mm, nil
}

// canonical 生成标准化表达式（小写关键字、HH:MM 补零、once 精确到分钟、on 用数字）。
func (sp *scheduleSpec) canonical() string {
	switch sp.typ {
	case ScheduleTypeOnce:
		if sp.runAt == nil {
			return ""
		}
		return fmt.Sprintf("once(%s)", sp.runAt.In(hcommon.BusinessLocation()).Format("2006-01-02 15:04"))
	case ScheduleTypeEvery:
		at := fmt.Sprintf("%02d:%02d", sp.hh, sp.mm)
		switch sp.unit {
		case ScheduleUnitDay:
			return fmt.Sprintf("every(d, at=%s)", at)
		case ScheduleUnitWeek:
			return fmt.Sprintf("every(w, on=%d, at=%s)", sp.dayOfWeek, at)
		case ScheduleUnitMonth:
			return fmt.Sprintf("every(m, on=%d, at=%s)", sp.dayOfMonth, at)
		}
	case ScheduleTypeCron:
		return fmt.Sprintf("cron(%s)", sp.cronExpr)
	case ScheduleTypeInterval:
		if sp.begin == nil {
			return ""
		}
		s := fmt.Sprintf("interval(%d%s, begin=%s", sp.intervalN, sp.intervalUnit,
			sp.begin.In(hcommon.BusinessLocation()).Format("2006-01-02 15:04"))
		if sp.end != nil {
			s += fmt.Sprintf(", end=%s", sp.end.In(hcommon.BusinessLocation()).Format("2006-01-02 15:04"))
		}
		return s + ")"
	}
	return ""
}

// nextRun 计算严格晚于 from 的下一次触发时刻。
func (sp *scheduleSpec) nextRun(from time.Time) (*time.Time, error) {
	loc := hcommon.BusinessLocation()
	from = from.In(loc)
	switch sp.typ {
	case ScheduleTypeOnce:
		if sp.runAt == nil {
			return nil, nil
		}
		if sp.runAt.In(loc).After(from) {
			t := *sp.runAt
			return &t, nil
		}
		return nil, nil

	case ScheduleTypeEvery:
		switch sp.unit {
		case ScheduleUnitDay:
			cand := time.Date(from.Year(), from.Month(), from.Day(), sp.hh, sp.mm, 0, 0, loc)
			for !cand.After(from) {
				cand = cand.AddDate(0, 0, 1)
			}
			return ptrTime(cand), nil

		case ScheduleUnitWeek:
			// Go: Sunday=0..Saturday=6；本系统 1=Mon..7=Sun → 7 映射为 0(Sun)
			target := time.Weekday(sp.dayOfWeek % 7)
			cand := time.Date(from.Year(), from.Month(), from.Day(), sp.hh, sp.mm, 0, 0, loc)
			for i := 0; i < 14; i++ {
				if cand.After(from) && cand.Weekday() == target {
					return ptrTime(cand), nil
				}
				cand = cand.AddDate(0, 0, 1)
			}
			return nil, fmt.Errorf("compute weekly next run failed")

		case ScheduleUnitMonth:
			// 从 from 所在月开始逐月查找；若某月没有 dayOfMonth 这一天（如 2 月无 30/31 号），整月跳过到下一个月。
			// 缺某日的月份后紧跟的月份必然包含该日（不会连续两月都缺），故最多 3 次即命中：
			base := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, loc)
			for i := 0; i < 3; i++ {
				m := base.AddDate(0, i, 0)
				if daysInMonth(m.Year(), m.Month()) < sp.dayOfMonth {
					continue // 该月没有这一天 → 跳过
				}
				cand := time.Date(m.Year(), m.Month(), sp.dayOfMonth, sp.hh, sp.mm, 0, 0, loc)
				if cand.After(from) {
					return ptrTime(cand), nil
				}
			}
			return nil, fmt.Errorf("compute monthly next run failed")
		}

	case ScheduleTypeCron:
		if sp.cronSched == nil {
			return nil, fmt.Errorf("%w: cron schedule not parsed", ErrScheduleSpecInvalid)
		}
		// robfig 的 Next 返回严格晚于 from 的下一次触发，且按 from 的时区计算（此处即 hcommon.BusinessLocation()）。
		next := sp.cronSched.Next(from)
		if next.IsZero() {
			return nil, nil // 5 年内无匹配（理论上不会发生），视为无后续触发
		}
		return ptrTime(next), nil

	case ScheduleTypeInterval:
		if sp.begin == nil || sp.intervalStep <= 0 {
			return nil, fmt.Errorf("%w: interval not parsed", ErrScheduleSpecInvalid)
		}
		begin := sp.begin.In(loc)
		step := sp.intervalStep
		var next time.Time
		if begin.After(from) {
			next = begin // 尚未到首次触发
		} else {
			// 最小 k 使 begin + k*step 严格晚于 from
			k := from.Sub(begin)/step + 1
			next = begin.Add(k * step)
			for !next.After(from) { // 安全兜底（理论上一次即满足）
				next = next.Add(step)
			}
		}
		if sp.end != nil && next.After(sp.end.In(loc)) {
			return nil, nil // 超过截止，无后续触发
		}
		return ptrTime(next), nil
	}
	return nil, fmt.Errorf("%w: unsupported schedule", ErrScheduleSpecInvalid)
}

// daysInMonth 返回指定年月的天数（28~31）。
func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 1, -1).Day()
}

// parseOnceTime 按多种 layout 解析 once 时间；不带时区的按业务时区（common.BusinessLocation）解释。
func parseOnceTime(s string) (time.Time, bool) {
	loc := hcommon.BusinessLocation()
	for _, layout := range scheduleOnceLayouts {
		if layout == time.RFC3339 {
			if t, err := time.Parse(layout, s); err == nil {
				return t, true
			}
			continue
		}
		if t, err := time.ParseInLocation(layout, s, loc); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func ptrTime(t time.Time) *time.Time { return &t }

// ============================================================================
// CRUD helper
// ============================================================================

// CountAgentCommandSchedules 统计当前租户定时任务数（不含软删）。
func CountAgentCommandSchedules(ctx context.Context) (int64, error) {
	var n int64
	if err := DB(ctx).Model(&AgentCommandSchedule{}).Count(&n).Error; err != nil {
		return 0, fmt.Errorf("count agent command schedules: %w", err)
	}
	return n, nil
}

// FindScheduleByID 按主键查当前租户内的定时任务；软删行不返回。
func FindScheduleByID(ctx context.Context, id uint) (*AgentCommandSchedule, error) {
	var s AgentCommandSchedule
	err := DB(ctx).Where("id = ?", id).First(&s).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrScheduleNotFound
		}
		return nil, fmt.Errorf("find schedule by id: %w", err)
	}
	return &s, nil
}

// FindScheduleBySlug 按对外资源 ID（sch-xxxx）查当前租户内的定时任务；软删行不返回。
func FindScheduleBySlug(ctx context.Context, slug string) (*AgentCommandSchedule, error) {
	var s AgentCommandSchedule
	err := DB(ctx).Where("slug = ?", slug).First(&s).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrScheduleNotFound
		}
		return nil, fmt.Errorf("find schedule by slug: %w", err)
	}
	return &s, nil
}

// GenerateScheduleSlug 生成 "sch-{8 位随机}" 资源 ID。字符集 [a-z0-9]。
func GenerateScheduleSlug() string {
	return AgentCommandScheduleSlugPrefix + randomLowerAlnum(AgentCommandScheduleSlugRandLen)
}

// CreateScheduleWithSlugRetry 在租户内为定时任务生成不冲突的 slug 并落库。
//
// 策略：随机生成 → Unscoped 检查（含软删行，避免随机串被软删行占用）→ 不冲突即 Create → 冲突重试。
// 超过 retries 仍冲突返回 ErrScheduleSlugConflict（8 字符 36^8 空间，碰撞概率近似 0，仅兜底极端并发）。
func CreateScheduleWithSlugRetry(ctx context.Context, s *AgentCommandSchedule, retries int) error {
	if retries <= 0 {
		retries = 5
	}
	for i := 0; i < retries; i++ {
		s.Slug = GenerateScheduleSlug()
		exists, err := scheduleSlugExists(ctx, s.Slug)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		if err := DB(ctx).Create(s).Error; err != nil {
			if isUniqueConflict(err) {
				continue
			}
			return fmt.Errorf("create schedule: %w", err)
		}
		return nil
	}
	return ErrScheduleSlugConflict
}

// scheduleSlugExists 通过 Unscoped 查询同租户内 slug 是否已被任何行（含软删）占用。
func scheduleSlugExists(ctx context.Context, slug string) (bool, error) {
	var n int64
	if err := DB(ctx).Unscoped().Model(&AgentCommandSchedule{}).
		Where("slug = ?", slug).Count(&n).Error; err != nil {
		return false, fmt.Errorf("check schedule slug: %w", err)
	}
	return n > 0, nil
}

// FindDueSchedules 列出当前租户内已到期、启用且 next_run_at <= now 的定时任务。
func FindDueSchedules(ctx context.Context, now time.Time, limit int) ([]AgentCommandSchedule, error) {
	q := DB(ctx).
		Where("enabled = ? AND next_run_at IS NOT NULL AND next_run_at <= ?", true, now).
		Order("next_run_at asc")
	if limit > 0 {
		q = q.Limit(limit)
	}
	var rows []AgentCommandSchedule
	if err := q.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("find due schedules: %w", err)
	}
	return rows, nil
}

// ClaimScheduleRun 抢占式推进 next_run_at（CAS），独占本周期触发权（多实例去重）。
// 仅当库中 next_run_at 仍等于 expected 时才更新成功，返回 true。
// next 为 nil 表示无后续触发（once）；此时同时停用，避免悬空 enabled。
func ClaimScheduleRun(ctx context.Context, id uint, expected time.Time, next *time.Time) (bool, error) {
	updates := map[string]any{
		"next_run_at": next,
		"updated_at":  time.Now(),
	}
	if next == nil {
		updates["enabled"] = false
	}
	res := DB(ctx).Model(&AgentCommandSchedule{}).
		Where("id = ? AND next_run_at = ?", id, expected).
		Updates(updates)
	if res.Error != nil {
		return false, fmt.Errorf("claim schedule run: %w", res.Error)
	}
	return res.RowsAffected == 1, nil
}

// MarkScheduleRunResult 记录一次成功/失败触发后的运行结果。
// running=true 表示 dispatch 已成功发起、正在执行（供 reconcile 后续订正）。
func MarkScheduleRunResult(ctx context.Context, id uint, ranAt time.Time, dispatchSlug, lastErr string, running bool) error {
	updates := map[string]any{
		"first_run_at":       gorm.Expr("COALESCE(first_run_at, ?)", ranAt),
		"last_run_at":        ranAt,
		"last_dispatch_slug": dispatchSlug,
		"last_error":         TruncateRunes(lastErr, 500),
		"is_running":         running,
		"updated_at":         time.Now(),
	}
	return DB(ctx).Model(&AgentCommandSchedule{}).
		Where("id = ?", id).Updates(updates).Error
}

// FindRunningSchedules 列出当前租户内 is_running=true 的定时任务（供 reconcile 订正）。
func FindRunningSchedules(ctx context.Context) ([]AgentCommandSchedule, error) {
	var rows []AgentCommandSchedule
	if err := DB(ctx).Where("is_running = ?", true).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("find running schedules: %w", err)
	}
	return rows, nil
}

// ClearScheduleRunning 批量把指定 id 的 is_running 订正为 false（dispatch 已终态）。
func ClearScheduleRunning(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return DB(ctx).Model(&AgentCommandSchedule{}).
		Where("id IN ?", ids).
		Updates(map[string]any{"is_running": false, "updated_at": time.Now()}).Error
}

// MarkScheduleSkipped 记录"上一轮未完成、本次跳过"（不改 last_dispatch_slug）。
func MarkScheduleSkipped(ctx context.Context, id uint, reason string) error {
	return DB(ctx).Model(&AgentCommandSchedule{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"last_error": TruncateRunes(reason, 500),
			"updated_at": time.Now(),
		}).Error
}
