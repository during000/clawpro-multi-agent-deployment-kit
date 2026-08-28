package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// ============================================================================
// 请求体 / 响应体类型
// ============================================================================

// agentCommandCreateReq 创建命令模板的请求体。
//
// 注意：slug 不在请求体内（决策 Q7：系统生成）；type 默认 SHELL（决策 Q8）。
type agentCommandCreateReq struct {
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	Type        string                    `json:"type"`
	Content     string                    `json:"content"`
	TimeoutSec  uint                      `json:"timeout_sec"`
	RunUser     string                    `json:"run_user"`
	Workdir     string                    `json:"workdir"`
	Params      []model.AgentCommandParam `json:"params"`
}

// agentCommandUpdateReq 编辑命令请求体（在 create 基础上多了 id）。
type agentCommandUpdateReq struct {
	ID uint `json:"id"`
	agentCommandCreateReq
}

// agentCommandDeleteReq 删除命令请求体。
type agentCommandDeleteReq struct {
	ID uint `json:"id"`
}

// agentCommandView 命令模板对前端的展示视图。
type agentCommandView struct {
	ID                uint                      `json:"id"`
	Slug              string                    `json:"slug"`
	Name              string                    `json:"name"`
	Description       string                    `json:"description"`
	Type              string                    `json:"type"`
	Content           string                    `json:"content"`
	TimeoutSec        uint                      `json:"timeout_sec"`
	RunUser           string                    `json:"run_user"`
	Workdir           string                    `json:"workdir"`
	Params            []model.AgentCommandParam `json:"params"`
	CreatedByUserID   uint                      `json:"created_by_user_id"`
	CreatedByUsername string                    `json:"created_by_username"`
	CreatedByEmail    string                    `json:"created_by_email"`
	CanEdit           bool                      `json:"can_edit"`
	LastExecutedAt    *time.Time                `json:"last_executed_at"`
	ExecutedCount     uint                      `json:"executed_count"`
	CreatedAt         time.Time                 `json:"created_at"`
	UpdatedAt         time.Time                 `json:"updated_at"`
}

// agentCommandListResp 命令模板列表响应。
type agentCommandListResp struct {
	Commands []agentCommandView `json:"commands"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
}

// blockingTaskInfo 删除拒绝时返回的阻塞任务信息。
type blockingTaskInfo struct {
	DispatchSlug string     `json:"dispatch_slug"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
}

// ============================================================================
// Handler: GET /admin/agent-commands
// ============================================================================

// HandleListAgentCommands 命令模板列表，支持搜索 + 分页 + 排序。
//
// 关键点：
//   - GORM 默认过滤 deleted_at IS NULL，软删行不会出现
//   - 多租户由 GORM identifier 回调自动过滤
//   - can_edit 字段由 handler 计算（创建者本人 OR 初始管理员）
func HandleListAgentCommands(w http.ResponseWriter, r *http.Request) {
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
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	page, pageSize := parsePagination(r)
	sortKey := r.URL.Query().Get("sort")

	tx := model.DB(ctx).Model(&model.AgentCommand{})
	if q != "" {
		like := "%" + q + "%"
		// 先按用户名预查 user_ids，让 q 能命中创建人；命中为空时退化到 3 字段。
		creatorIDs := findUserIDsByUsernameLike(ctx, q)
		if len(creatorIDs) > 0 {
			tx = tx.Where(
				"slug LIKE ? OR name LIKE ? OR content LIKE ? OR created_by_user_id IN ?",
				like, like, like, creatorIDs)
		} else {
			tx = tx.Where(
				"slug LIKE ? OR name LIKE ? OR content LIKE ?",
				like, like, like)
		}
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgAgentCmdCountFailed))
		return
	}

	switch sortKey {
	case "name_asc":
		tx = tx.Order("name asc")
	default: // updated_at_desc
		tx = tx.Order("updated_at desc")
	}

	var rows []model.AgentCommand
	if err := tx.Limit(pageSize).Offset((page - 1) * pageSize).Find(&rows).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgAgentCmdQueryListFailed))
		return
	}

	// 批量取创建者用户名
	creatorIDs := make([]uint, 0, len(rows))
	cmdIDs := make([]uint, 0, len(rows))
	for _, c := range rows {
		creatorIDs = append(creatorIDs, c.CreatedByUserID)
		cmdIDs = append(cmdIDs, c.ID)
	}
	creators := batchUsersByIDs(ctx, creatorIDs)
	stats, _ := model.BatchCommandExecutionStats(ctx, cmdIDs)

	isInitialAdmin := user.IsInitialAdmin(ctx)
	commands := make([]agentCommandView, 0, len(rows))
	for _, c := range rows {
		v := newAgentCommandView(&c, creators[c.CreatedByUserID], stats[c.ID])
		v.CanEdit = canEditAgentCommand(user, &c, isInitialAdmin)
		commands = append(commands, v)
	}
	jsonOK(w, agentCommandListResp{
		Commands: commands, Total: total, Page: page, PageSize: pageSize,
	})
}

// ============================================================================
// Handler: POST /admin/agent-commands/create
// ============================================================================

// HandleCreateAgentCommand 创建命令模板。
func HandleCreateAgentCommand(w http.ResponseWriter, r *http.Request) {
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

	var req agentCommandCreateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgInvalidJSON))
		return
	}
	if offender, err := validateAndNormalizeAgentCommandReq(&req); err != nil {
		writeAgentCommandValidationError(w, r, err, offender)
		return
	}

	ctx := r.Context()

	// 同租户下命令名称不可重复（仅校验未软删行；GORM 默认 query 自动过滤 deleted_at IS NULL）。
	// MySQL 部署上还会被 generated column `name_active` + UNIQUE 索引兜底（详见 init.sql §agent_commands）。
	if err := checkAgentCommandNameUnique(ctx, req.Name, 0); err != nil {
		writeError(w, r, http.StatusConflict, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 配额校验（决策 R-NEW-4）
	count, err := model.CountActiveAgentCommands(ctx)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if count >= model.MaxAgentCommandsPerTenant {
		writeError(w, r, http.StatusConflict,
			hcommon.I18nError(i18n.MsgAgentCmdQuotaExceeded, model.MaxAgentCommandsPerTenant).WithPrefix("quota_exceeded"))
		return
	}

	cmd := &model.AgentCommand{
		Name:            req.Name,
		Description:     req.Description,
		Type:            req.Type,
		Content:         req.Content, // ⚠️ 决策 §8.1：raw 文本入库
		TimeoutSec:      req.TimeoutSec,
		RunUser:         req.RunUser,
		Workdir:         req.Workdir,
		VisibilityType:  model.AgentCommandVisibilityTenant,
		CreatedByUserID: user.ID,
	}
	if err := cmd.SetParams(req.Params); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if err := model.CreateAgentCommandWithSlugRetry(ctx, cmd, 8); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 返回完整视图
	creator := batchUsersByIDs(ctx, []uint{cmd.CreatedByUserID})[cmd.CreatedByUserID]
	view := newAgentCommandView(cmd, creator, model.CommandExecutionStat{})
	view.CanEdit = true
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, view)
}

// ============================================================================
// Handler: POST /admin/agent-commands/update
// ============================================================================

// HandleUpdateAgentCommand 编辑命令模板，仅创建者或初始管理员（super_admin）有权限。
func HandleUpdateAgentCommand(w http.ResponseWriter, r *http.Request) {
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

	var req agentCommandUpdateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgInvalidJSON))
		return
	}
	if req.ID == 0 {
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgAgentCmdIDRequired).WithPrefix("id_required"))
		return
	}
	if offender, err := validateAndNormalizeAgentCommandReq(&req.agentCommandCreateReq); err != nil {
		writeAgentCommandValidationError(w, r, err, offender)
		return
	}

	ctx := r.Context()

	cmd, err := model.FindAgentCommandByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, model.ErrAgentCommandNotFound) {
			writeError(w, r, http.StatusNotFound,
				hcommon.I18nError(i18n.MsgCommandNotFound).WithPrefix("command_not_found"))
			return
		}
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	// 权限检查：本人 OR 初始管理员（决策 Q1）
	if !canEditAgentCommand(user, cmd, user.IsInitialAdmin(ctx)) {
		writeError(w, r, http.StatusForbidden,
			hcommon.I18nError(i18n.MsgAgentCmdEditDenied).WithPrefix("permission_denied"))
		return
	}

	// 改名时也要查重；自身行 id 排除避免误判（"改成自己原来的名"算不变更）。
	if req.Name != cmd.Name {
		if err := checkAgentCommandNameUnique(ctx, req.Name, cmd.ID); err != nil {
			writeError(w, r, http.StatusConflict, hcommon.EnsureRichErrorOrPanic(err))
			return
		}
	}

	// type 不可修改（保留原值），slug 不可修改（保留原值）
	cmd.Name = req.Name
	cmd.Description = req.Description
	cmd.Content = req.Content
	cmd.TimeoutSec = req.TimeoutSec
	cmd.RunUser = req.RunUser
	cmd.Workdir = req.Workdir
	if err := cmd.SetParams(req.Params); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if err := model.DB(ctx).Save(cmd).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgAgentCmdUpdateFailed))
		return
	}

	creator := batchUsersByIDs(ctx, []uint{cmd.CreatedByUserID})[cmd.CreatedByUserID]
	stats, _ := model.BatchCommandExecutionStats(ctx, []uint{cmd.ID})
	view := newAgentCommandView(cmd, creator, stats[cmd.ID])
	view.CanEdit = true
	jsonOK(w, view)
}

// ============================================================================
// Handler: POST /admin/agent-commands/delete
// ============================================================================

// HandleDeleteAgentCommand 软删命令模板，存在 in_progress 引用时拒绝。
func HandleDeleteAgentCommand(w http.ResponseWriter, r *http.Request) {
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

	var req agentCommandDeleteReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgInvalidJSON))
		return
	}
	if req.ID == 0 {
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgAgentCmdIDRequired).WithPrefix("id_required"))
		return
	}

	ctx := r.Context()
	cmd, err := model.FindAgentCommandByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, model.ErrAgentCommandNotFound) {
			writeError(w, r, http.StatusNotFound,
				hcommon.I18nError(i18n.MsgCommandNotFound).WithPrefix("command_not_found"))
			return
		}
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if !canEditAgentCommand(user, cmd, user.IsInitialAdmin(ctx)) {
		writeError(w, r, http.StatusForbidden,
			hcommon.I18nError(i18n.MsgAgentCmdDeleteDenied).WithPrefix("permission_denied"))
		return
	}

	// 删除前置校验（决策 Q2）
	hasInProgress, slugs, err := model.HasInProgressDispatches(ctx, cmd.ID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if hasInProgress {
		blockers := make([]blockingTaskInfo, 0, len(slugs))
		for _, s := range slugs {
			blockers = append(blockers, blockingTaskInfo{DispatchSlug: s})
		}
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgAgentCmdInUseDetail).
			WithPrefix("command_in_use").
			WithCustomData(map[string]any{
				"code":           http.StatusConflict,
				"message":        i18n.T(r.Context(), i18n.MsgAgentCmdInUseDetail),
				"blocking_tasks": blockers,
			}))
		return
	}

	if err := model.DB(ctx).Delete(cmd).Error; err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgAgentCmdDeleteFailed))
		return
	}
	now := time.Now()
	jsonOK(w, map[string]any{
		"ok":         true,
		"id":         cmd.ID,
		"deleted_at": now.Format(time.RFC3339),
	})
}

// ============================================================================
// 辅助函数
// ============================================================================

// canEditAgentCommand 判断给定用户是否可以编辑/删除某命令。
// 决策 Q1：创建者本人 或 初始管理员（super_admin）。
func canEditAgentCommand(user *model.User, cmd *model.AgentCommand, isInitialAdmin bool) bool {
	if user == nil || cmd == nil {
		return false
	}
	if cmd.CreatedByUserID == user.ID {
		return true
	}
	return isInitialAdmin
}

// findUserIDsByUsernameLike 按 username 模糊查 user_id 列表，用于命令列表 q 命中创建人用户名场景。
//
// 含软删用户（Unscoped），保证历史命令的创建人即使账号已删也能搜出来。
// 失败或空命中返回 nil。上限 200 条避免极端情况下 IN 列表过长。
func findUserIDsByUsernameLike(ctx context.Context, q string) []uint {
	if q == "" {
		return nil
	}
	var rows []struct {
		ID uint
	}
	err := model.DB(ctx).Model(&model.User{}).Unscoped().
		Select("id").
		Where("username LIKE ?", "%"+q+"%").
		Limit(200).
		Find(&rows).Error
	if err != nil || len(rows) == 0 {
		return nil
	}
	ids := make([]uint, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	return ids
}

// batchUsersByIDs 按 ids 批量取 User 行（含软删，便于展示历史命令的创建者）。
// 返回 map[id] -> User 副本。失败时返回空 map（不阻塞调用方）。
func batchUsersByIDs(ctx context.Context, ids []uint) map[uint]model.User {
	out := make(map[uint]model.User)
	if len(ids) == 0 {
		return out
	}
	dedup := make(map[uint]struct{}, len(ids))
	uniq := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := dedup[id]; ok {
			continue
		}
		dedup[id] = struct{}{}
		uniq = append(uniq, id)
	}
	if len(uniq) == 0 {
		return out
	}
	var rows []model.User
	if err := model.DB(ctx).Unscoped().Where("id IN ?", uniq).Find(&rows).Error; err != nil {
		return out
	}
	for _, u := range rows {
		out[u.ID] = u
	}
	return out
}

// newAgentCommandView 构建展示视图。
func newAgentCommandView(c *model.AgentCommand, creator model.User, stat model.CommandExecutionStat) agentCommandView {
	v := agentCommandView{
		ID:              c.ID,
		Slug:            c.Slug,
		Name:            c.Name,
		Description:     c.Description,
		Type:            c.Type,
		Content:         c.Content,
		TimeoutSec:      c.TimeoutSec,
		RunUser:         c.RunUser,
		Workdir:         c.Workdir,
		Params:          c.Params(),
		CreatedByUserID: c.CreatedByUserID,
		LastExecutedAt:  stat.LastExecutedAt,
		ExecutedCount:   stat.ExecutedCount,
		CreatedAt:       c.CreatedAt,
		UpdatedAt:       c.UpdatedAt,
	}
	if v.Params == nil {
		v.Params = []model.AgentCommandParam{}
	}
	if creator.ID != 0 {
		v.CreatedByUsername = creator.Username
		v.CreatedByEmail = creator.Email
	}
	return v
}

// checkAgentCommandNameUnique 校验同租户下 name 在「未软删」行内不重复。
//
// 程序层防御：GORM 默认 query 已经自动加 `WHERE deleted_at IS NULL`，且
// model.DB(ctx) 自动注入 `WHERE identifier = ?`，所以 Where 子句这里只写 name。
//
// excludeID > 0 时把自身行排除（编辑场景：name 不变也不报）。
//
// 严格 race-safe 由 MySQL 端的 generated column `name_active` + UNIQUE 兜底
// （详见 sql/init.sql §agent_commands）；此处程序层先 Count 是为了在并发非
// 极端场景给出干净的 409 错误码而不是裸 SQL 错误。
func checkAgentCommandNameUnique(ctx context.Context, name string, excludeID uint) error {
	q := model.DB(ctx).Model(&model.AgentCommand{}).Where("name = ?", name)
	if excludeID > 0 {
		q = q.Where("id != ?", excludeID)
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return hcommon.I18nRichError(err, i18n.MsgAgentCmdNameCheckFailed)
	}
	if count > 0 {
		return hcommon.I18nError(i18n.MsgAgentCmdNameAlreadyExists, name).WithPrefix("name_already_exists")
	}
	return nil
}

// validateAndNormalizeAgentCommandReq 把请求体校验后填充默认值。
//
// 返回 (offender, err)。当 err 为 ValidateAgentCommandParams 抛出的参数级错误
// （name 非法 / 重名 / default 超长 / description 超长）时，offender 是冲突
// 参数的 name，便于错误响应文案携带。其它情况 offender 为空。
func validateAndNormalizeAgentCommandReq(req *agentCommandCreateReq) (string, error) {
	if req.Type == "" {
		req.Type = model.AgentCommandTypeShell
	}
	if req.TimeoutSec == 0 {
		req.TimeoutSec = model.AgentCommandTimeoutDefault
	}
	if req.RunUser == "" {
		req.RunUser = "root"
	}
	if req.Workdir == "" {
		req.Workdir = "/root"
	}
	if err := model.ValidateAgentCommandName(req.Name); err != nil {
		return "", err
	}
	if err := model.ValidateAgentCommandDescription(req.Description); err != nil {
		return "", err
	}
	if err := model.ValidateAgentCommandType(req.Type); err != nil {
		return "", err
	}
	if err := model.ValidateAgentCommandContent(req.Content); err != nil {
		return "", err
	}
	if err := model.ValidateAgentCommandTimeout(req.TimeoutSec); err != nil {
		return "", err
	}
	if err := model.ValidateAgentCommandRunUser(req.RunUser); err != nil {
		return "", err
	}
	if err := model.ValidateAgentCommandWorkdir(req.Workdir); err != nil {
		return "", err
	}
	if offender, err := model.ValidateAgentCommandParams(req.Params); err != nil {
		return offender, err
	}
	return "", nil
}

// writeAgentCommandValidationError 把模型层抛出的校验错误映射为 HTTP 400 + 业务错误码。
func writeAgentCommandValidationError(w http.ResponseWriter, r *http.Request, err error, offender string) {
	switch {
	case errors.Is(err, model.ErrAgentCommandNameInvalid):
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgAgentCmdNameInvalidChars).WithPrefix("name_invalid_chars"))
	case errors.Is(err, model.ErrAgentCommandNameTooLong):
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgAgentCmdNameTooLong).WithPrefix("name_too_long"))
	case errors.Is(err, model.ErrAgentCommandDescTooLong):
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgAgentCmdDescriptionTooLong).WithPrefix("description_too_long"))
	case errors.Is(err, model.ErrAgentCommandContentEmpty):
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgAgentCmdContentRequired).WithPrefix("content_required"))
	case errors.Is(err, model.ErrAgentCommandContentTooLong):
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgAgentCmdContentTooLong).WithPrefix("content_too_long"))
	case errors.Is(err, model.ErrAgentCommandTimeoutOOR):
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgAgentCmdTimeoutOutOfRange).WithPrefix("timeout_out_of_range"))
	case errors.Is(err, model.ErrAgentCommandRunUserTooLong):
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgAgentCmdRunUserTooLong).WithPrefix("run_user_too_long"))
	case errors.Is(err, model.ErrAgentCommandWorkdirTooLong):
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgAgentCmdWorkdirTooLong).WithPrefix("workdir_too_long"))
	case errors.Is(err, model.ErrAgentCommandTypeInvalid):
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgAgentCmdInvalidType).WithPrefix("invalid_type"))
	case errors.Is(err, model.ErrAgentCommandParamsTooMany):
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgAgentCmdParamsTooMany).WithPrefix("params_too_many"))
	case errors.Is(err, model.ErrAgentCommandParamNameInvalid):
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgAgentCmdParamNameInvalid, offender).WithPrefix("param_name_invalid"))
	case errors.Is(err, model.ErrAgentCommandParamNameDup):
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgAgentCmdParamNameDuplicated, offender).WithPrefix("param_name_duplicated"))
	case errors.Is(err, model.ErrAgentCommandParamDefaultTooLong):
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgAgentCmdParamDefaultTooLong, offender).WithPrefix("param_default_too_long"))
	case errors.Is(err, model.ErrAgentCommandParamDescTooLong):
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgAgentCmdParamDescriptionTooLong, offender).WithPrefix("param_description_too_long"))
	default:
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
	}
}
