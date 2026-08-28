package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// ============================================================================
// 定时任务（agent command schedule）管理
//
// 定时任务 = "何时触发一次 dispatch" 的配置。到期由后台 runner 调用 startDispatch
// 触发，执行链路完全复用既有 dispatch 体系。
//
// 端点（均 requireAdmin）：
//   GET  /admin/agent-commands/schedules            列表
//   POST /admin/agent-commands/schedules/create     创建
//   POST /admin/agent-commands/schedules/delete     软删
//   POST /admin/agent-commands/schedules/toggle     启停
// ============================================================================

// agentScheduleCreateReq 创建请求体。
//
// schedule 为函数式调度表达式（见 model.SetScheduleExpr 语法）：
//
//	once(2026-06-30 15:00) / every(d, at=02:00) / every(w, on=1, at=09:00) /
//	every(m, on=1, at=09:00) / cron(*/5 * * * *) / interval(1m, begin=2026-06-30 15:00)
//
// every 周几/几号均为数字（on: 周级 1=Mon..7=Sun，月级 1..31）；cron 为标准 5 字段
// （分 时 日 月 周，周 0-6，0=周日..6=周六）；interval 从 begin 起每隔 N（m/h/d）触发、可选 end 截止。
// 时间一律按业务时区（region 时区）解释。
type agentScheduleCreateReq struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	CommandID   uint              `json:"command_id"`
	InstanceIDs []uint            `json:"instance_ids"`
	ParamValues map[string]string `json:"param_values"`
	Schedule    string            `json:"schedule"`
}

// agentScheduleView 列表/详情视图。schedule 回显单字符串表达式。
// id 为对外资源 ID（sch-xxxx），非数据库主键。
type agentScheduleView struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	Description       string            `json:"description"`
	CommandID         uint              `json:"command_id"`
	CommandName       string            `json:"command_name"`
	InstanceIDs       []uint            `json:"instance_ids"`
	ParamValues       map[string]string `json:"param_values"`
	Schedule          string            `json:"schedule"`
	Enabled           bool              `json:"enabled"`
	IsRunning         bool              `json:"is_running"`
	Status            string            `json:"status"` // 合成状态：running/completed/paused/pending/waiting
	NextRunAt         *time.Time        `json:"next_run_at"`
	FirstRunAt        *time.Time        `json:"first_run_at"`
	LastRunAt         *time.Time        `json:"last_run_at"`
	LastDispatchSlug  string            `json:"last_dispatch_slug"`
	CreatedByUserID   uint              `json:"created_by_user_id"`
	CreatedByUsername string            `json:"created_by_username"`
	CanEdit           bool              `json:"can_edit"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

type agentScheduleListResp struct {
	Schedules []agentScheduleView `json:"schedules"`
	Total     int64               `json:"total"`
	Page      int                 `json:"page"`
	PageSize  int                 `json:"page_size"`
}

// ============================================================================
// GET /admin/agent-commands/schedules
// ============================================================================

func HandleListAgentCommandSchedules(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}
	user, err := getLoginUser(r)
	if err != nil || user == nil {
		writeError(w, r, http.StatusUnauthorized, hcommon.I18nError(i18n.MsgUnauthorized))
		return
	}

	ctx := r.Context()
	page, pageSize := parsePagination(r)
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))

	tx := model.DB(ctx).Model(&model.AgentCommandSchedule{})
	if q != "" {
		// q 同时匹配对外资源 ID（slug，形如 sch-xxxx）与名称，与命令模板列表搜索口径一致。
		like := "%" + q + "%"
		tx = tx.Where("slug LIKE ? OR name LIKE ?", like, like)
	}
	if status != "" {
		cond, args, ok := model.ScheduleStatusCondition(status)
		if !ok {
			writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgScheduleInvalidStatus).WithPrefix("invalid_status"))
			return
		}
		tx = tx.Where(cond, args...)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
		return
	}

	var rows []model.AgentCommandSchedule
	if err := tx.Order("created_at desc").
		Limit(pageSize).Offset((page - 1) * pageSize).
		Find(&rows).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
		return
	}

	// 批量取命令名 + 创建者用户名
	cmdIDs := make([]uint, 0, len(rows))
	uidList := make([]uint, 0, len(rows))
	for i := range rows {
		cmdIDs = append(cmdIDs, rows[i].CommandID)
		uidList = append(uidList, rows[i].CreatedByUserID)
	}
	cmdNames := batchCommandNamesByIDs(ctx, cmdIDs)
	users := batchUsersByIDs(ctx, uidList)
	isInitialAdmin := user.IsInitialAdmin(ctx)

	views := make([]agentScheduleView, 0, len(rows))
	for i := range rows {
		v := newAgentScheduleView(&rows[i], cmdNames[rows[i].CommandID], users[rows[i].CreatedByUserID])
		v.CanEdit = canEditSchedule(user, &rows[i], isInitialAdmin)
		views = append(views, v)
	}
	jsonOK(w, agentScheduleListResp{Schedules: views, Total: total, Page: page, PageSize: pageSize})
}

// ============================================================================
// POST /admin/agent-commands/schedules/create
// ============================================================================

func HandleCreateAgentCommandSchedule(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}
	user, err := getLoginUser(r)
	if err != nil || user == nil {
		writeError(w, r, http.StatusUnauthorized, hcommon.I18nError(i18n.MsgUnauthorized))
		return
	}

	var req agentScheduleCreateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgInvalidJSON).WithPrefix("invalid_body"))
		return
	}
	ctx := r.Context()

	// 配额
	count, err := model.CountAgentCommandSchedules(ctx)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
		return
	}
	if count >= model.MaxAgentCommandSchedulesPerTenant {
		writeError(w, r, http.StatusConflict,
			hcommon.I18nError(i18n.MsgScheduleQuotaExceeded, model.MaxAgentCommandSchedulesPerTenant).WithPrefix("quota_exceeded"))
		return
	}

	// 拒绝本地 agent 实例：本地实例不支持命令下发，避免创建后定时任务永远失败
	if len(req.InstanceIDs) > 0 {
		var localInstances []model.Instance
		if err := model.DB(ctx).Where("id IN ? AND source = ?", req.InstanceIDs, model.InstanceSourceLocal).Find(&localInstances).Error; err != nil {
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
			return
		}
		if len(localInstances) > 0 {
			localNames := make([]string, 0, len(localInstances))
			for _, ins := range localInstances {
				localNames = append(localNames, ins.Name)
			}
			writeError(w, r, http.StatusBadRequest,
				hcommon.I18nError(i18n.MsgLocalInstanceTargetUnsupported, strings.Join(localNames, ", ")).
					WithPrefix("local_instance_target_unsupported"))
			return
		}
	}

	s := &model.AgentCommandSchedule{CreatedByUserID: user.ID}
	if err := applyScheduleReq(ctx, s, &req); err != nil {
		writeScheduleValidationError(w, r, err)
		return
	}
	// 计算首个 next_run_at
	next, err := s.ComputeNextRun(time.Now())
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgScheduleSpecInvalid).WithPrefix("schedule_spec_invalid").WithDetail(err.Error()))
		return
	}
	if next == nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgScheduleExpiredCreate).WithPrefix("schedule_spec_invalid"))
		return
	}
	s.NextRunAt = next
	s.Enabled = true

	if err := model.CreateScheduleWithSlugRetry(ctx, s, 5); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
		return
	}
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, scheduleViewWithMeta(ctx, s))
}

// ============================================================================
// POST /admin/agent-commands/schedules/delete
// ============================================================================

func HandleDeleteAgentCommandSchedule(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}
	user, err := getLoginUser(r)
	if err != nil || user == nil {
		writeError(w, r, http.StatusUnauthorized, hcommon.I18nError(i18n.MsgUnauthorized))
		return
	}

	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgInvalidJSON).WithPrefix("invalid_body"))
		return
	}
	if strings.TrimSpace(req.ID) == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgScheduleIDRequired).WithPrefix("id_required"))
		return
	}
	ctx := r.Context()

	s, err := model.FindScheduleBySlug(ctx, req.ID)
	if err != nil {
		if errors.Is(err, model.ErrScheduleNotFound) {
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgScheduleNotFound).WithPrefix("schedule_not_found"))
			return
		}
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
		return
	}
	if !canEditSchedule(user, s, user.IsInitialAdmin(ctx)) {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgSchedulePermDeniedDelete).WithPrefix("permission_denied"))
		return
	}
	if err := model.DB(ctx).Delete(s).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
		return
	}
	jsonOK(w, map[string]any{"ok": true, "id": s.Slug})
}

// ============================================================================
// POST /admin/agent-commands/schedules/toggle
// ============================================================================

func HandleToggleAgentCommandSchedule(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}
	user, err := getLoginUser(r)
	if err != nil || user == nil {
		writeError(w, r, http.StatusUnauthorized, hcommon.I18nError(i18n.MsgUnauthorized))
		return
	}

	var req struct {
		ID      string `json:"id"`
		Enabled bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgInvalidJSON).WithPrefix("invalid_body"))
		return
	}
	if strings.TrimSpace(req.ID) == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgScheduleIDRequired).WithPrefix("id_required"))
		return
	}
	ctx := r.Context()

	s, err := model.FindScheduleBySlug(ctx, req.ID)
	if err != nil {
		if errors.Is(err, model.ErrScheduleNotFound) {
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgScheduleNotFound).WithPrefix("schedule_not_found"))
			return
		}
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
		return
	}
	if !canEditSchedule(user, s, user.IsInitialAdmin(ctx)) {
		writeError(w, r, http.StatusForbidden, hcommon.I18nError(i18n.MsgSchedulePermDeniedOperate).WithPrefix("permission_denied"))
		return
	}
	// 已完成的终态任务（once 已执行 / interval 已到截止）启停均无意义，直接拦截
	if s.Status() == model.ScheduleStatusCompleted {
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgScheduleCompleted).WithPrefix("schedule_completed"))
		return
	}

	updates := map[string]any{"enabled": req.Enabled, "updated_at": time.Now()}
	if req.Enabled {
		// 启用时重算 next_run_at（停用期间可能已过期）
		next, cerr := s.ComputeNextRun(time.Now())
		if cerr != nil || next == nil {
			writeError(w, r, http.StatusBadRequest,
				hcommon.I18nError(i18n.MsgScheduleExpiredToggle).WithPrefix("schedule_spec_invalid"))
			return
		}
		updates["next_run_at"] = next
	}
	if err := model.DB(ctx).Model(&model.AgentCommandSchedule{}).
		Where("id = ?", s.ID).Updates(updates).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
		return
	}
	s, _ = model.FindScheduleByID(ctx, s.ID)
	jsonOK(w, scheduleViewWithMeta(ctx, s))
}

// ============================================================================
// GET /admin/agent-commands/schedules/records
// ============================================================================

// agentScheduleRecordView 执行记录视图：record 只存 dispatch_slug，状态实时查 dispatch 拼装。
type agentScheduleRecordView struct {
	ID           uint       `json:"id"`
	DispatchSlug string     `json:"dispatch_slug"`
	Status       string     `json:"status"`
	TargetCount  uint       `json:"target_count"`
	SuccessCount uint       `json:"success_count"`
	FailedCount  uint       `json:"failed_count"`
	StartedAt    *time.Time `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

type agentScheduleRecordListResp struct {
	Records  []agentScheduleRecordView `json:"records"`
	Total    int64                     `json:"total"`
	Page     int                       `json:"page"`
	PageSize int                       `json:"page_size"`
}

// HandleListAgentCommandScheduleRecords 列出某定时任务的历史执行记录。
//
// record 仅存 dispatch_slug；本接口实时批量查 dispatch 表拼装状态/计数返回（避免 N+1）。
func HandleListAgentCommandScheduleRecords(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireAdmin(w, r) {
		return
	}

	ctx := r.Context()
	slug := strings.TrimSpace(r.URL.Query().Get("schedule_id"))
	if slug == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgScheduleIDQueryRequired).WithPrefix("schedule_id_required"))
		return
	}
	sched, err := model.FindScheduleBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, model.ErrScheduleNotFound) {
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgScheduleNotFound).WithPrefix("schedule_not_found"))
			return
		}
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
		return
	}
	page, pageSize := parsePagination(r)

	records, total, err := model.ListScheduleRecords(ctx, sched.ID, page, pageSize)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
		return
	}

	// 批量查 dispatch 拼状态（避免 N+1）
	slugs := make([]string, 0, len(records))
	for i := range records {
		if records[i].DispatchSlug != "" {
			slugs = append(slugs, records[i].DispatchSlug)
		}
	}
	dispBySlug := make(map[string]model.AgentCommandDispatch, len(slugs))
	if len(slugs) > 0 {
		var rows []model.AgentCommandDispatch
		if err := model.DB(ctx).Where("slug IN ?", slugs).Find(&rows).Error; err == nil {
			for i := range rows {
				dispBySlug[rows[i].Slug] = rows[i]
			}
		}
	}

	views := make([]agentScheduleRecordView, 0, len(records))
	for i := range records {
		rec := &records[i]
		v := agentScheduleRecordView{
			ID:           rec.ID,
			DispatchSlug: rec.DispatchSlug,
			CreatedAt:    rec.CreatedAt,
		}
		if d, ok := dispBySlug[rec.DispatchSlug]; ok {
			v.Status = d.Status
			v.TargetCount = d.TargetCount
			v.SuccessCount = d.SuccessCount
			v.FailedCount = d.FailedCount
			started := d.StartedAt
			v.StartedAt = &started
			v.FinishedAt = d.FinishedAt
		}
		views = append(views, v)
	}
	jsonOK(w, agentScheduleRecordListResp{Records: views, Total: total, Page: page, PageSize: pageSize})
}

// ============================================================================
// 辅助
// ============================================================================

// TriggerScheduleDispatch 用 schedule 配置触发一次 dispatch（复用 startDispatch），返回 dispatch slug。
//
// 供后台 runner（task 包）调用——派发逻辑（startDispatch + TAT）封装在 controller 内，
// runner 只负责编排（找到期 / 抢占 / 串行保护 / 记录），通过本函数完成实际触发。
// 定时任务允许「尽力而为」：部分目标离线时只对在线机器下发，离线机器记为失败、不阻塞整批。
func TriggerScheduleDispatch(ctx context.Context, s *model.AgentCommandSchedule) (string, error) {
	res, err := startDispatch(ctx, startDispatchInput{
		CommandID:           s.CommandID,
		InstanceIDs:         s.InstanceIDsList(),
		ParamValues:         s.ParamValuesMap(),
		TriggeredByUserID:   s.CreatedByUserID,
		AllowPartialOffline: true,
	})
	if err != nil {
		return "", err
	}
	return res.DispatchSlug, nil
}

// applyScheduleReq 把请求体写入 schedule 并校验归一化。
func applyScheduleReq(ctx context.Context, s *model.AgentCommandSchedule, req *agentScheduleCreateReq) error {
	// 命令存在性校验（当前租户）
	if _, err := model.FindAgentCommandByID(ctx, req.CommandID); err != nil {
		if errors.Is(err, model.ErrAgentCommandNotFound) {
			return fmt.Errorf("%w: 命令不存在或已被删除", model.ErrScheduleSpecInvalid)
		}
		return err
	}

	s.Name = strings.TrimSpace(req.Name)
	s.Description = strings.TrimSpace(req.Description)
	s.CommandID = req.CommandID
	if err := s.SetInstanceIDs(dedupUintSlice(req.InstanceIDs)); err != nil {
		return err
	}
	if err := s.SetParamValues(req.ParamValues); err != nil {
		return err
	}

	if err := s.SetScheduleExpr(req.Schedule); err != nil {
		return err
	}

	return s.ValidateAndNormalize()
}

// writeScheduleValidationError 把校验错误映射为 HTTP code。
func writeScheduleValidationError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, model.ErrScheduleSpecInvalid) {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgScheduleSpecInvalid).WithPrefix("schedule_spec_invalid").WithDetail(err.Error()))
		return
	}
	writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
}

// canEditSchedule 创建者本人或初始管理员可改/删。
func canEditSchedule(user *model.User, s *model.AgentCommandSchedule, isInitialAdmin bool) bool {
	if user == nil || s == nil {
		return false
	}
	if s.CreatedByUserID == user.ID {
		return true
	}
	return isInitialAdmin
}

// batchCommandNamesByIDs 批量取命令名（含软删，便于展示历史定时任务的命令名）。
func batchCommandNamesByIDs(ctx context.Context, ids []uint) map[uint]string {
	out := make(map[uint]string)
	if len(ids) == 0 {
		return out
	}
	uniq := dedupUintSlice(ids)
	if len(uniq) == 0 {
		return out
	}
	var rows []model.AgentCommand
	if err := model.DB(ctx).Unscoped().Where("id IN ?", uniq).Find(&rows).Error; err != nil {
		return out
	}
	for i := range rows {
		out[rows[i].ID] = rows[i].Name
	}
	return out
}

// scheduleViewWithMeta 单条 schedule 转视图（补命令名 + 创建者），CanEdit=true（操作者本人路径）。
func scheduleViewWithMeta(ctx context.Context, s *model.AgentCommandSchedule) agentScheduleView {
	cmdName := batchCommandNamesByIDs(ctx, []uint{s.CommandID})[s.CommandID]
	creator := batchUsersByIDs(ctx, []uint{s.CreatedByUserID})[s.CreatedByUserID]
	v := newAgentScheduleView(s, cmdName, creator)
	v.CanEdit = true
	return v
}

// newAgentScheduleView 构建展示视图。
func newAgentScheduleView(s *model.AgentCommandSchedule, cmdName string, creator model.User) agentScheduleView {
	v := agentScheduleView{
		ID:               s.Slug,
		Name:             s.Name,
		Description:      s.Description,
		CommandID:        s.CommandID,
		CommandName:      cmdName,
		InstanceIDs:      s.InstanceIDsList(),
		ParamValues:      s.ParamValuesMap(),
		Schedule:         s.ScheduleExpr,
		Enabled:          s.Enabled,
		IsRunning:        s.IsRunning,
		Status:           s.Status(),
		NextRunAt:        s.NextRunAt,
		FirstRunAt:       s.FirstRunAt,
		LastRunAt:        s.LastRunAt,
		LastDispatchSlug: s.LastDispatchSlug,
		CreatedByUserID:  s.CreatedByUserID,
		CreatedAt:        s.CreatedAt,
		UpdatedAt:        s.UpdatedAt,
	}
	if v.InstanceIDs == nil {
		v.InstanceIDs = []uint{}
	}
	if creator.ID != 0 {
		v.CreatedByUsername = creator.Username
	}
	return v
}
