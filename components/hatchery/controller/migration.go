package controller

// migration.go — 独立账号迁移到统一账号模式的旁路工具。
//
// 设计形态：迁移 worker 按 API 触发，主动扫描本地 DB 同步到 OneID。
// 现有所有 handler 不感知迁移状态，照常只写本地 DB。
//
// 内存状态：sync.Map[identifier]*JobState
//   - 进程重启后内存清空，重新 POST /admin/migrate 注入参数即可继续（幂等）
//   - Phase3At 归零 → 下次 run 全表重扫密码（幂等）
//   - UserMirror 归空 → 下次 run 首轮全量推 role/dept（幂等）
//
// API（3 个）:
//   POST /admin/migrate          注入 OneID 参数（初始化 job，可覆盖重注入）
//   GET  /admin/migrate          查状态快照 + 失败列表
//   POST /admin/migrate/run      执行同步（phase1+2+3，幂等可重复）
//   POST /admin/migrate/finalize 最后一次 run + 写 site_configs + 切统一模式

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// ── 内存状态 ─────────────────────────────────────────────────────────────────

// OneIDMigrateConfig 迁移所需的 OneID 参数（存于内存，不落 DB）。
type OneIDMigrateConfig struct {
	AccountID      string
	AppID          string
	ClientID       string
	ClientSecret   string
	TokenEndpoint  string
	GatewaySecret  string // Gateway X-Internal-Token HMAC 密钥，Phase 3 密码同步需要
	AdminUnionID   string // OneID 超管 union_id，迁移时直接绑定，跳过创建
	AdminLoginName string // OneID 超管登录名，绑定 oneid_login_name
}

// UserSnapshot 记录上次已知的用户 role 和所属 group，用于 run 时 mirror diff。
type UserSnapshot struct {
	Role     string
	GroupIDs []uint
}

// MigrateFailureRecord 一条失败记录。
type MigrateFailureRecord struct {
	Phase      int       `json:"phase"`
	TargetID   uint      `json:"target_id"`
	TargetName string    `json:"target_name"`
	Error      string    `json:"error"`
	At         time.Time `json:"at"`
}

// JobState 单个租户的迁移内存状态。
type JobState struct {
	Config     OneIDMigrateConfig
	Phase3At   time.Time             // 上次密码同步水位，updated_at > 此值的用户需重同步
	UserMirror map[uint]UserSnapshot // uid → {role, group_ids}，用于 mirror diff
	failures   []MigrateFailureRecord
	mu         sync.Mutex
}

const maxFailures = 1000

// addFailure 向 ring buffer 追加失败记录，超出上限覆盖最早的。
func (j *JobState) addFailure(r MigrateFailureRecord) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if len(j.failures) >= maxFailures {
		j.failures = j.failures[1:]
	}
	j.failures = append(j.failures, r)
}

// Failures 返回当前失败列表的副本。
func (j *JobState) Failures() []MigrateFailureRecord {
	j.mu.Lock()
	defer j.mu.Unlock()
	out := make([]MigrateFailureRecord, len(j.failures))
	copy(out, j.failures)
	return out
}

// jobs 全局迁移状态，identifier → *JobState。
var jobs sync.Map

// IsMigrating 判断指定租户是否有 active job。
func IsMigrating(identifier string) bool {
	_, ok := jobs.Load(identifier)
	return ok
}

// GetJobState 获取当前租户的 job（从请求 ctx 中提取 identifier）。
func GetJobState(r *http.Request) (*JobState, bool) {
	snap, ok := hcommon.GetTenantSnapshot(r.Context())
	if !ok {
		return nil, false
	}
	v, ok := jobs.Load(snap.Identifier)
	if !ok {
		return nil, false
	}
	return v.(*JobState), true
}

// ── Handlers ─────────────────────────────────────────────────────────────────

// migrateError 向迁移 API 返回标准 JSON 错误响应（复用 writeError 规范格式）。
func migrateError(w http.ResponseWriter, r *http.Request, status int, msg string) {
	var richErr *hcommon.RichError
	switch status {
	case http.StatusBadRequest:
		richErr = hcommon.I18nError(i18n.MsgBadRequest).WithDetail(msg)
	case http.StatusNotFound:
		richErr = hcommon.I18nError(i18n.MsgNotFound).WithDetail(msg)
	case http.StatusConflict:
		richErr = hcommon.I18nError(i18n.MsgInternalError).WithDetail(msg)
	case http.StatusMethodNotAllowed:
		richErr = hcommon.I18nError(i18n.MsgMethodNotAllowed).WithDetail(msg)
	default:
		richErr = hcommon.I18nError(i18n.MsgInternalError).WithDetail(msg)
	}
	writeError(w, r, status, richErr)
}

// HandleMigrateInit POST /admin/migrate
// 注入 OneID 参数，初始化（或覆盖）内存 job。
// 进程重启后重新调用此接口即可恢复迁移，无需其他操作（幂等）。
func HandleMigrateInit(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		migrateError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		AccountID      string `json:"oneid_account_id"`
		AppID          string `json:"oneid_app_id"`
		ClientID       string `json:"oneid_client_id"`
		ClientSecret   string `json:"oneid_client_secret"`
		TokenEndpoint  string `json:"oneid_token_endpoint"`
		GatewaySecret  string `json:"gateway_internal_secret"`
		AdminUnionID   string `json:"admin_union_id"`
		AdminLoginName string `json:"admin_login_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		migrateError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}
	if hcommon.IsUnifiedAccountMode(r.Context()) {
		migrateError(w, r, http.StatusConflict, "当前租户已是统一账号模式")
		return
	}
	if req.AccountID == "" || req.AppID == "" || req.ClientID == "" ||
		req.ClientSecret == "" || req.TokenEndpoint == "" {
		migrateError(w, r, http.StatusBadRequest, "所有 oneid 参数均为必填")
		return
	}
	if GatewayURL == "" {
		migrateError(w, r, http.StatusBadRequest, "GATEWAY_URL 未配置，密码同步将无法执行")
		return
	}
	if req.GatewaySecret == "" {
		migrateError(w, r, http.StatusBadRequest, "gateway_internal_secret 为必填，密码同步需要 HMAC 签名")
		return
	}
	if req.AdminUnionID == "" {
		migrateError(w, r, http.StatusBadRequest, "admin_union_id 为必填，超管 OneID union_id 不能为空")
		return
	}
	if req.AdminLoginName == "" {
		migrateError(w, r, http.StatusBadRequest, "admin_login_name 为必填，超管 OneID 登录名不能为空")
		return
	}

	snap, _ := hcommon.GetTenantSnapshot(r.Context())
	job := &JobState{
		Config: OneIDMigrateConfig{
			AccountID:      req.AccountID,
			AppID:          req.AppID,
			ClientID:       req.ClientID,
			ClientSecret:   req.ClientSecret,
			TokenEndpoint:  req.TokenEndpoint,
			GatewaySecret:  req.GatewaySecret,
			AdminUnionID:   req.AdminUnionID,
			AdminLoginName: req.AdminLoginName,
		},
		UserMirror: make(map[uint]UserSnapshot),
	}
	jobs.Store(snap.Identifier, job)
	jsonOK(w, map[string]interface{}{"ok": true})
}

// HandleMigrateStatus GET /admin/migrate
// 查看当前 job 快照和失败列表。
func HandleMigrateStatus(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	job, ok := GetJobState(r)
	if !ok {
		migrateError(w, r, http.StatusNotFound, "no active migration job")
		return
	}
	job.mu.Lock()
	mirrorCount := len(job.UserMirror)
	phase3At := job.Phase3At
	clientID := job.Config.ClientID
	accountID := job.Config.AccountID
	appID := job.Config.AppID
	job.mu.Unlock()

	failures := job.Failures()
	jsonOK(w, map[string]interface{}{
		"ok": true,
		"job": map[string]interface{}{
			"oneid_account_id":  accountID,
			"oneid_app_id":      appID,
			"oneid_client_id":   clientID,
			"phase3_at":         phase3At,
			"mirror_user_count": mirrorCount,
			"failures_count":    len(failures),
		},
		"failures": failures,
	})
}

// HandleMigrateRun POST /admin/migrate/run
// 执行一轮完整同步：分组 → 用户 → 密码 → mirror diff → 软删。
// 幂等，可重复调用；source_ref/one_id_sub 已填的自动跳过。
func HandleMigrateRun(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		migrateError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	job, ok := GetJobState(r)
	if !ok {
		migrateError(w, r, http.StatusNotFound, "no active migration job")
		return
	}
	if hcommon.IsUnifiedAccountMode(r.Context()) {
		migrateError(w, r, http.StatusConflict, "已是统一模式，无需迁移")
		return
	}

	ctx := migrateCtx(r, job)
	result := runMigrateReconcile(ctx, job)
	jsonOK(w, result)
}

// HandleMigrateFinalize POST /admin/migrate/finalize
// 最后一次完整同步 + 将 job.Config 写入 site_configs + 立刻切换到统一模式。
// 无需重启服务，写入后当前进程的后续请求立刻走统一模式逻辑。
// ?force=true 跳过残留检查（有未同步数据时仍强制切换）。
func HandleMigrateFinalize(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		migrateError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	job, ok := GetJobState(r)
	if !ok {
		migrateError(w, r, http.StatusNotFound, "no active migration job")
		return
	}
	if hcommon.IsUnifiedAccountMode(r.Context()) {
		migrateError(w, r, http.StatusConflict, "当前租户已是统一账号模式")
		return
	}

	ctx := r.Context()

	// 最后一轮同步
	mCtx := migrateCtx(r, job)
	runMigrateReconcile(mCtx, job)

	// 软检查：收集 warnings
	var warnings []string
	var unsyncedGroups int64
	model.DB(ctx).Model(&model.UserGroup{}).
		Where("(source_ref = '' OR source_ref IS NULL) AND source = ?", model.GroupSourceManual).
		Count(&unsyncedGroups)
	if unsyncedGroups > 0 {
		warnings = append(warnings, fmt.Sprintf("%d 个 user_groups 尚未同步 source_ref", unsyncedGroups))
	}

	var unsyncedUsers int64
	model.DB(ctx).Model(&model.User{}).
		Where("one_id_sub IS NULL OR one_id_sub = ''").
		Count(&unsyncedUsers)
	if unsyncedUsers > 0 {
		warnings = append(warnings, fmt.Sprintf("%d 个 users 尚未同步 one_id_sub", unsyncedUsers))
	}

	failures := job.Failures()
	if len(failures) > 0 {
		warnings = append(warnings, fmt.Sprintf("%d 条 failures in memory", len(failures)))
	}

	// 非 force 模式下有硬阻断条件：用户未全部同步（分组允许残留）
	force := r.URL.Query().Get("force") == "true"
	if !force && unsyncedUsers > 0 {
		migrateError(w, r, http.StatusConflict, fmt.Sprintf(
			"仍有 %d 个用户未同步 one_id_sub，切换后这些用户将无法登录；确认后加 ?force=true 强制执行",
			unsyncedUsers,
		))
		return
	}

	snap, _ := hcommon.GetTenantSnapshot(ctx)

	// 写入 site_configs
	job.mu.Lock()
	cfg := job.Config
	job.mu.Unlock()

	updates := map[string]interface{}{
		"one_id_account_id":     cfg.AccountID,
		"one_id_app_id":         cfg.AppID,
		"one_id_client_id":      cfg.ClientID,
		"one_id_client_secret":  cfg.ClientSecret,
		"one_id_token_endpoint": cfg.TokenEndpoint,
	}
	// internal_secret 为 per-tenant 派生密钥（tke-tools 用 master+account_id 派生后下发），
	// 迁移期间在 job.Config.GatewaySecret 中。finalize 持久化到 site_configs，
	// 确保切换到统一模式后从 DB 加载的 snapshot 仍持正确 InternalSecret 用于 gateway 验签；
	// 否则后续内部接口（add-role/reset-password 等）会因签名缺失 secret 而 401。
	if cfg.GatewaySecret != "" {
		updates["internal_secret"] = cfg.GatewaySecret
	}
	if err := model.DB(ctx).Model(&model.SiteConfig{}).
		Where("identifier = ?", snap.Identifier).
		Updates(updates).Error; err != nil {
		migrateError(w, r, http.StatusInternalServerError, fmt.Sprintf("写入 site_configs 失败: %v", err))
		return
	}

	// 清内存 job（已切换，不再需要）
	jobs.Delete(snap.Identifier)

	jsonOK(w, map[string]interface{}{
		"ok":       true,
		"warnings": warnings,
	})
}

// HandleMigrateReloadSite POST /admin/migrate/reload-site
// 多租户模式：清除当前租户的域名缓存，下次请求时自动从 DB 重新加载。
// 单租户模式：FixedSnapshot 在进程启动时加载，需重启才能刷新，返回提示。
// migrateCtx 构造注入了 job.Config 的 ctx，使现有 OneID 函数能正常工作。
// 迁移模式下 site_configs 还没写入，TenantSnapshot 里没有 OneID 字段，
// 通过此函数将内存里的 job.Config 覆盖进 snapshot。
func migrateCtx(r *http.Request, job *JobState) context.Context {
	snap, _ := hcommon.GetTenantSnapshot(r.Context())
	job.mu.Lock()
	cfg := job.Config
	job.mu.Unlock()

	snap.OneIDAccountID = cfg.AccountID
	snap.OneIDAppID = cfg.AppID
	snap.OneIDClientID = cfg.ClientID
	snap.OneIDClientSecret = cfg.ClientSecret
	snap.OneIDTokenEndpoint = cfg.TokenEndpoint
	if cfg.GatewaySecret != "" {
		snap.InternalSecret = cfg.GatewaySecret
	}
	return hcommon.InjectTenant(r.Context(), snap)
}
