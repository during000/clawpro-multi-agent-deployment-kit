package controller

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

const (
	migrationEstimatedSize = 500 * 1024 * 1024 // 500MB
)

// instanceMigrationDependencies 是迁移 handler 的外部依赖接口，测试时可替换。
type instanceMigrationDependencies interface {
	instanceStatusResolver
	PrepareMigrationUpload(ctx context.Context, fileKey string, estimatedSize int64) (*SMHUploadCredential, error)
	CheckSMHCommonFileExists(ctx context.Context, fileKey string) (bool, int64, error)
	BuildMigrationScript(ctx context.Context, m *model.AgentMigration, cred *SMHUploadCredential, agentType string) (string, error)
	GetCommonSpaceToken(ctx context.Context) (string, error)
}

// defaultInstanceMigrationDependencies 是生产环境使用的真实实现。
type defaultInstanceMigrationDependencies struct {
	defaultInstanceStatusResolver
}

func (defaultInstanceMigrationDependencies) PrepareMigrationUpload(ctx context.Context, fileKey string, estimatedSize int64) (*SMHUploadCredential, error) {
	return PrepareMigrationUpload(ctx, fileKey, estimatedSize)
}
func (defaultInstanceMigrationDependencies) CheckSMHCommonFileExists(ctx context.Context, fileKey string) (bool, int64, error) {
	return CheckSMHCommonFileExists(ctx, fileKey)
}
func (defaultInstanceMigrationDependencies) BuildMigrationScript(ctx context.Context, m *model.AgentMigration, cred *SMHUploadCredential, agentType string) (string, error) {
	return buildMigrationScript(ctx, m, cred, agentType, func() (string, error) { return GetCommonSpaceToken(ctx) })
}
func (defaultInstanceMigrationDependencies) GetCommonSpaceToken(ctx context.Context) (string, error) {
	return GetCommonSpaceToken(ctx)
}

var defaultMigrationDeps instanceMigrationDependencies = defaultInstanceMigrationDependencies{}

// HandleMigrationExport 为目标实例生成迁移导出脚本。
// POST /openclaw/migration/export
func HandleMigrationExport(w http.ResponseWriter, r *http.Request) {
	handleMigrationExport(w, r, defaultMigrationDeps)
}

func handleMigrationExport(w http.ResponseWriter, r *http.Request, deps instanceMigrationDependencies) {
	ctx := r.Context()
	log := Logger(ctx)
	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireSMHEnabled(w, r) {
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if rejectLocalOrWrite(w, r, instance) {
		return
	}

	log.Info("[MigrationExport] 目标实例", "instanceId", instance.InstanceId)

	fileKey := fmt.Sprintf("migrations/%s/agent-export.tgz", instance.InstanceId)

	cred, err := deps.PrepareMigrationUpload(r.Context(), fileKey, migrationEstimatedSize)
	if err != nil {
		log.Error("[MigrationExport] 初始化 SMH 上传失败", "instanceId", instance.InstanceId, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgMigrationInitUploadFailed))
		return
	}

	// 查找或创建迁移记录（每个实例只保留一条 pending_upload 记录）
	var migration model.AgentMigration
	model.DB(r.Context()).Where("instance_id = ? AND status = ?", instance.ID, model.MigrationStatusPendingUpload).
		Order("id DESC").First(&migration)

	if migration.ID == 0 {
		migration = model.AgentMigration{
			InstanceID:    instance.ID,
			CVMInstanceID: instance.InstanceId,
			FileKey:       fileKey,
			Status:        model.MigrationStatusPendingUpload,
		}
		if err := model.DB(r.Context()).Create(&migration).Error; err != nil {
			log.Error("[MigrationExport] 写入迁移记录失败", "error", err)
			writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgMigrationCreateRecordFailed))
			return
		}
	}

	// 凭证不存 DB，每次实时生成脚本
	script, err := deps.BuildMigrationScript(r.Context(), &migration, cred, instance.AgentType)
	if err != nil {
		log.Error("[MigrationExport] 生成脚本失败", "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgMigrationGenScriptFailed))
		return
	}

	log.Info("[MigrationExport] 迁移脚本已生成", "migrationId", migration.ID, "fileKey", fileKey)
	resp := map[string]interface{}{
		"migration_id": migration.ID,
		"script":       script,
		"file_key":     fileKey,
	}
	if cred.Expiration != nil {
		resp["expire_at"] = cred.Expiration.Format(time.RFC3339)
	}
	jsonOK(w, resp)
}

// HandleMigrationStatus 查询目标实例的迁移状态（SMH 文件是否就绪）。
// GET /openclaw/migration/status?id=xxx
func HandleMigrationStatus(w http.ResponseWriter, r *http.Request) {
	handleMigrationStatus(w, r, defaultMigrationDeps)
}

func handleMigrationStatus(w http.ResponseWriter, r *http.Request, deps instanceMigrationDependencies) {
	ctx := r.Context()
	log := Logger(ctx)
	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	if !requireSMHEnabled(w, r) {
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	var migration model.AgentMigration
	if err := model.DB(r.Context()).Where("instance_id = ?", instance.ID).
		Where("status = ?", model.MigrationStatusPendingUpload).
		Order("id DESC").First(&migration).Error; err != nil {
		jsonOK(w, map[string]interface{}{
			"has_migration": false,
		})
		return
	}

	exists, fileSize, err := deps.CheckSMHCommonFileExists(r.Context(), migration.FileKey)
	if err != nil {
		log.Error("[MigrationStatus] 检查 SMH 文件失败", "fileKey", migration.FileKey, "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgMigrationCheckFileFailed))
		return
	}

	jsonOK(w, map[string]interface{}{
		"has_migration": true,
		"migration_id":  migration.ID,
		"file_key":      migration.FileKey,
		"file_ready":    exists,
		"file_size":     fileSize,
		"can_import":    exists,
	})
}

// HandleMigrationProgress 查询迁移 import 进展（纯 DB 查询，无 SMH 调用）。
// GET /openclaw/migration/progress?id=xxx
func HandleMigrationProgress(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}

	var migration model.AgentMigration
	if err := model.DB(r.Context()).Where("instance_id = ?", instance.ID).
		Order("id DESC").First(&migration).Error; err != nil {
		jsonOK(w, map[string]interface{}{
			"has_migration": false,
		})
		return
	}

	resp := map[string]interface{}{
		"has_migration": true,
		"migration_id":  migration.ID,
		"status":        migration.Status,
		"steps":         model.ParseMigrationSteps(&migration),
	}
	if migration.FailReason != "" {
		resp["fail_reason"] = migration.FailReason
	}
	jsonOK(w, resp)
}

// HandleMigrationImport 触发目标实例从 SMH 下载并恢复 agent 数据。
// POST /openclaw/migration/import
func HandleMigrationImport(w http.ResponseWriter, r *http.Request) {
	handleMigrationImport(w, r, defaultMigrationDeps)
}

func handleMigrationImport(w http.ResponseWriter, r *http.Request, deps instanceMigrationDependencies) {
	ctx := r.Context()
	log := Logger(ctx)
	jsonAPI(w)
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if !requireSMHEnabled(w, r) {
		return
	}

	instance, err := getInstanceByID(&w, r, user)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	log.Info("[MigrationImport] 目标实例", "instanceId", instance.InstanceId)

	// 查找有效的迁移记录
	var migration model.AgentMigration
	if err := model.DB(r.Context()).Where("instance_id = ? AND status = ?",
		instance.ID, model.MigrationStatusPendingUpload).
		Order("id DESC").First(&migration).Error; err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMigrationNoExportRecord))
		return
	}

	// 确认 SMH 文件已就绪
	exists, _, err := deps.CheckSMHCommonFileExists(r.Context(), migration.FileKey)
	if err != nil {
		log.Error("[MigrationImport] 检查 SMH 文件失败", "error", err)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgMigrationCheckFileFailed))
		return
	}
	if !exists {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMigrationFileNotUploaded))
		return
	}

	// 本地实例：不支持迁移（无 CVM 实例需要迁移）。
	if rejectLocalOrWrite(w, r, instance) {
		return
	}
	// 实例必须 running（TAT 需要）—— 与 /openclaw/status 一致口径
	if _, err := requireInstanceRunning(r.Context(), instance, deps); err != nil {
		writeAgentGuardError(w, r, err)
		return
	}
	// 防重入：已有操作进行中时拒绝
	if instance.CurrentOperation != "" && instance.CurrentOperationState == model.OpStateProcessing {
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgUpgradeOperationInProgress, instance.CurrentOperation))
		return
	}

	if err := setOperation(model.DB(r.Context()), instance, model.OpMigrate); err != nil {
		log.Error("[MigrationImport] 设置操作锁失败", "error", err)
		writeError(w, r, http.StatusConflict, hcommon.I18nRichError(err, i18n.MsgMigrationSetOpLockFailed))
		return
	}

	model.DB(r.Context()).Model(&migration).Update("status", model.MigrationStatusImporting)
	model.InitMigrationSteps(r.Context(), model.DB(r.Context()), &migration, instance.AgentType)

	go func(ctx context.Context) {
		if err := performMigrationImport(ctx, instance, &migration); err != nil {
			log.Error("[MigrationImport] 异步迁移失败", "instanceId", instance.InstanceId, "error", err)
		}
	}(hcommon.DetachContext(ctx))
	clearAdjustmentFailure(r.Context(), instance.ID)

	log.Info("[MigrationImport] 迁移任务已提交", "instanceId", instance.InstanceId, "migrationId", migration.ID)
	jsonOK(w, i18n.T(r.Context(), i18n.MsgMigrationStarted))
}

// performMigrationImportDeps 是 performMigrationImport 的外部依赖接口，测试时可替换。
type performMigrationImportDeps struct {
	BuildSMHDownloadURL func(ctx context.Context, fileKey string) (string, error)
	RunRestoreScript    func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string)) error
	WaitForReady        func(ctx context.Context, instanceId, agentType string, timeout time.Duration) error
	SyncModels          func(ctx context.Context, instance *model.Instance, log *slog.Logger) bool
	SyncSMHSpace        func(ctx context.Context, space *model.SMHPersonalSpace) error
	DeleteMigrationFile func(ctx context.Context, fileKey string) error
}

var defaultPerformMigrationImportDeps = performMigrationImportDeps{
	BuildSMHDownloadURL: func(ctx context.Context, fileKey string) (string, error) {
		return BuildCommonSMHDownloadURL(ctx, fileKey, true)
	},
	RunRestoreScript: func(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string, onOutput func(string)) error {
		_, err := agentScriptRunner(ctx, instanceId, scriptName, timeout, runtimeUser, onOutput, nil)
		if err != nil {
			return err
		}
		return nil
	},
	WaitForReady: func(ctx context.Context, instanceId, agentType string, timeout time.Duration) error {
		return waitForOpenclawReady(ctx, instanceId, agentType, timeout)
	},
	SyncModels: func(ctx context.Context, instance *model.Instance, log *slog.Logger) bool {
		return syncMigrationModels(ctx, instance, log)
	},
	SyncSMHSpace: func(ctx context.Context, space *model.SMHPersonalSpace) error {
		return TriggerSyncPersonalSpaceEnv(ctx, space, true)
	},
	DeleteMigrationFile: func(ctx context.Context, fileKey string) error {
		return DeleteSMHCommonFile(ctx, fileKey)
	},
}

// performMigrationImport 异步执行迁移恢复流程。
func performMigrationImport(ctx context.Context, instance *model.Instance, migration *model.AgentMigration) (importErr error) {
	return performMigrationImportWithDeps(ctx, instance, migration, defaultPerformMigrationImportDeps)
}

func performMigrationImportWithDeps(ctx context.Context, instance *model.Instance, migration *model.AgentMigration, deps performMigrationImportDeps) (importErr error) {
	log := Logger(ctx)

	succeeded := false
	defer func() {
		state := model.OpStateFailed
		if succeeded {
			state = model.OpStateSuccess
		}
		if err := clearOperation(model.DB(ctx), instance, state); err != nil {
			log.Error("[MigrationImport] 清除操作锁失败", "error", err)
		}
		if !succeeded {
			errMsg := ""
			if importErr != nil {
				errMsg = importErr.Error()
			}
			model.DB(ctx).Model(migration).Updates(map[string]interface{}{
				"status":      model.MigrationStatusFailed,
				"fail_reason": errMsg,
			})
		} else {
			model.DB(ctx).Model(migration).Update("status", model.MigrationStatusDone)
		}
	}()

	smhURL, err := deps.BuildSMHDownloadURL(ctx, migration.FileKey)
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgUpgradeBuildSMHDownloadURLFailed)
	}
	log.Info("[MigrationImport] SMH 下载 URL 已生成", "instanceId", instance.InstanceId)

	// TAT SaveCommand=false 时 Parameters 替换不生效，在 Go 侧替换占位符
	restoreScript, err := LoadScript("restore_from_migration.sh")
	if err != nil {
		return hcommon.I18nRichError(err, i18n.MsgMigrationLoadRestoreScriptFailed)
	}
	agentType := model.NormalizeAgentType(instance.AgentType)
	restoreScript = strings.ReplaceAll(restoreScript, `RUNTIME_USER="{{runtime_user}}"`, fmt.Sprintf(`RUNTIME_USER=%s`, shellQuote(instance.RuntimeUser)))
	restoreScript = strings.ReplaceAll(restoreScript, `ARCHIVE_URL="{{url}}"`, fmt.Sprintf(`ARCHIVE_URL=%s`, shellQuote(smhURL)))
	restoreScript = strings.ReplaceAll(restoreScript, `AGENT_TYPE="{{agent_type}}"`, fmt.Sprintf(`AGENT_TYPE=%s`, shellQuote(agentType)))
	// 注入保留路径列表（空格分隔），restore 脚本解压后从备份恢复这些平台目录
	preservedPathsStr := strings.Join(agentMigrationPreservedPaths(ctx, agentType), " ")
	restoreScript = strings.ReplaceAll(restoreScript, `PRESERVED_PATHS="{{preserved_paths}}"`, fmt.Sprintf(`PRESERVED_PATHS=%s`, shellQuote(preservedPathsStr)))

	tmpName := fmt.Sprintf("_restore_migration_%d.sh", time.Now().UnixNano())
	RegisterInlineScript(tmpName, restoreScript)
	defer UnregisterInlineScript(tmpName)

	const restoreMaxRetry = 3
	var restoreErr error

	// onOutput 解析脚本输出的 PROGRESS:xxx 行，实时更新迁移阶段。
	// 收到新阶段时，先将所有仍处于 running 的脚本阶段标为 success，再将新阶段标为 running。
	onOutput := func(chunk string) {
		for _, line := range strings.Split(chunk, "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "PROGRESS:") {
				continue
			}
			step := strings.TrimPrefix(line, "PROGRESS:")
			for _, s := range model.ParseMigrationSteps(migration) {
				if s.Status == model.MigrationStepStatusRunning {
					model.UpdateMigrationStep(model.DB(ctx), migration, s.Step, model.MigrationStepStatusSuccess, nil)
				}
			}
			model.UpdateMigrationStep(model.DB(ctx), migration, step, model.MigrationStepStatusRunning, nil)
		}
	}

	for attempt := 1; attempt <= restoreMaxRetry; attempt++ {
		if attempt > 1 {
			log.Warn("[MigrationImport] 恢复命令下发失败，等待后重试",
				"instanceId", instance.InstanceId, "attempt", attempt, "error", restoreErr)
			time.Sleep(30 * time.Second)
		}
		restoreErr = deps.RunRestoreScript(ctx, instance.InstanceId, tmpName, 600, instance.RuntimeUser, onOutput)
		if restoreErr == nil {
			break
		}
		richErr, ok := restoreErr.(*hcommon.RichError)
		if !ok || (!errors.Is(richErr, ErrTATCommandDispatchFailed) && !errors.Is(richErr, ErrTATCommandStartFailed)) {
			break
		}
	}
	if restoreErr != nil {
		// 将当前 running 中的脚本阶段标为 failed
		for _, step := range []string{model.MigrationStepDownloading, model.MigrationStepBackingUp, model.MigrationStepExtracting} {
			for _, s := range model.ParseMigrationSteps(migration) {
				if s.Step == step && s.Status == model.MigrationStepStatusRunning {
					model.UpdateMigrationStep(model.DB(ctx), migration, step, model.MigrationStepStatusFailed, nil)
				}
			}
		}
		return hcommon.I18nRichError(restoreErr, i18n.MsgMigrationRestoreAgentFailed)
	}
	log.Info("[MigrationImport] agent 数据恢复完成", "instanceId", instance.InstanceId)

	// ── 等待 agent 重启就绪 ───────────────────────────────────────────────
	log.Info("[MigrationImport] 等待 agent 重启就绪", "instanceId", instance.InstanceId)
	if err := deps.WaitForReady(ctx, instance.InstanceId, instance.AgentType, 10*time.Minute); err != nil {
		model.UpdateMigrationStep(model.DB(ctx), migration, model.MigrationStepRestarting, model.MigrationStepStatusFailed, nil)
		return hcommon.I18nRichError(err, i18n.MsgMigrationWaitAgentReadyTimeout)
	}
	model.UpdateMigrationStep(model.DB(ctx), migration, model.MigrationStepRestarting, model.MigrationStepStatusSuccess, nil)
	log.Info("[MigrationImport] agent 已就绪，开始后处理", "instanceId", instance.InstanceId)

	// ── 后处理：修复迁移后的 DB 不一致 ──────────────────────────────────

	// 1. 从源实例 agent 配置提取可迁移模型，写入目标实例 instance_models
	model.UpdateMigrationStep(model.DB(ctx), migration, model.MigrationStepSyncingModels, model.MigrationStepStatusRunning, nil)
	isPrimaryValid := deps.SyncModels(ctx, instance, log)
	model.UpdateMigrationStep(model.DB(ctx), migration, model.MigrationStepSyncingModels, model.MigrationStepStatusSuccess,
		map[string]interface{}{"is_primary_model_valid": isPrimaryValid})

	// 2. SMH 个人空间：直接 sync（覆盖 agent 目录里源实例的 space_id/token）
	if model.AgentTypeSupportsSMH(ctx, instance.AgentType) {
		model.UpdateMigrationStep(model.DB(ctx), migration, model.MigrationStepSyncingSMH, model.MigrationStepStatusRunning, nil)
		var space model.SMHPersonalSpace
		if model.DB(ctx).Where("instance_id = ? AND to_be_deleted_at IS NULL", instance.ID).First(&space).Error == nil {
			if err := deps.SyncSMHSpace(ctx, &space); err != nil {
				log.Warn("[MigrationImport] SMH 个人空间 sync 失败（不影响结果）", "instanceId", instance.InstanceId, "error", err)
				model.UpdateMigrationStep(model.DB(ctx), migration, model.MigrationStepSyncingSMH, model.MigrationStepStatusFailed, nil)
			} else {
				log.Info("[MigrationImport] SMH 个人空间已同步", "instanceId", instance.InstanceId)
				model.UpdateMigrationStep(model.DB(ctx), migration, model.MigrationStepSyncingSMH, model.MigrationStepStatusSuccess, nil)
			}
		} else {
			model.UpdateMigrationStep(model.DB(ctx), migration, model.MigrationStepSyncingSMH, model.MigrationStepStatusSuccess, nil)
		}
	}

	// 3. 版本信息重置（缓存，等 AgentChecker 重新探测）
	model.DB(ctx).WithContext(ctx).Model(instance).Updates(map[string]interface{}{
		"agent_ready":          0,
		"agent_version":        "",
		"plugin_versions_json": "",
		"version_fetched_at":   nil,
	})

	// 4. Memory 状态重置（仅支持 Memory 的类型）
	if model.AgentTypeSupportsMemory(ctx, instance.AgentType) {
		resetMemoryPluginForReinstall(ctx, instance.InstanceId)
	}

	// 5. 技能/插件/MCP 安装记录清空（磁盘上已有文件，不触发重新安装任务）
	model.DB(ctx).Where("instance_id = ?", instance.ID).Delete(&model.SkillInstallation{})
	model.DB(ctx).Where("instance_id = ?", instance.ID).Delete(&model.PluginInstallation{})
	model.DB(ctx).Where("instance_id = ?", instance.ID).Delete(&model.McpInstallation{})

	log.Info("[MigrationImport] 后处理完成", "instanceId", instance.InstanceId)

	// 恢复成功后删除 SMH 上的迁移文件
	if err := deps.DeleteMigrationFile(ctx, migration.FileKey); err != nil {
		log.Warn("[MigrationImport] 删除 SMH 迁移文件失败（不影响结果）", "fileKey", migration.FileKey, "error", err)
	} else {
		log.Info("[MigrationImport] SMH 迁移文件已删除", "fileKey", migration.FileKey)
	}

	succeeded = true
	return nil
}

// migrationModelEntry 是 extract_migration_models.sh 输出的单个模型条目。
type migrationModelEntry struct {
	Role       string   `json:"role"`
	ModelID    string   `json:"model_id"`
	ModelName  string   `json:"model_name"`
	BaseURL    string   `json:"base_url"`
	APIKey     string   `json:"api_key"`
	APIMode    string   `json:"api_mode"`
	ContextLen int      `json:"context_len"`
	InputTypes []string `json:"input_types"`
}

type migrationModelsOutput struct {
	AgentType string                `json:"agent_type"`
	Models    []migrationModelEntry `json:"models"`
}

// migrationModelsDependencies 是 syncMigrationModels 的外部依赖接口，测试时可替换。
type migrationModelsDependencies interface {
	RunScript(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string) (string, error)
}

type defaultMigrationModelsDependencies struct{}

func (defaultMigrationModelsDependencies) RunScript(ctx context.Context, instanceId, scriptName string, timeout uint64, runtimeUser string) (string, error) {
	output, rerr := agentScriptRunner(ctx, instanceId, scriptName, timeout, runtimeUser, nil, nil)
	if rerr != nil {
		return output, rerr
	}
	return output, nil
}

// syncMigrationModels 执行 extract_migration_models 脚本，解析输出，
// 清空目标实例的模型记录，并将源实例的可迁移模型写入 instance_models。
// 返回 isPrimaryValid：是否存在 role=primary 的有效模型（用户可直接使用）。
func syncMigrationModels(ctx context.Context, instance *model.Instance, log *slog.Logger) (isPrimaryValid bool) {
	return syncMigrationModelsWithDeps(ctx, instance, log, defaultMigrationModelsDependencies{})
}

func syncMigrationModelsWithDeps(ctx context.Context, instance *model.Instance, log *slog.Logger, deps migrationModelsDependencies) (isPrimaryValid bool) {
	agentType := LookupAgentType(ctx, instance.InstanceId)
	scriptName, rerr := ResolveScript(ctx, "extract_migration_models", agentType)
	if rerr != nil {
		log.Warn("[MigrationModels] 不支持该 agent 类型，跳过模型迁移", "agentType", agentType, "error", rerr)
		return false
	}

	output, err := deps.RunScript(ctx, instance.InstanceId, scriptName, 30, instance.RuntimeUser)
	if err != nil {
		log.Warn("[MigrationModels] 提取模型配置失败（不影响结果）", "instanceId", instance.InstanceId, "error", err)
		return false
	}

	// 从输出里找最后一行 JSON
	var jsonLine string
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "{") {
			jsonLine = line
		}
	}
	if jsonLine == "" {
		log.Warn("[MigrationModels] 脚本未输出有效 JSON", "instanceId", instance.InstanceId)
		return false
	}

	var result migrationModelsOutput
	if err := json.Unmarshal([]byte(jsonLine), &result); err != nil {
		log.Warn("[MigrationModels] 解析模型 JSON 失败", "instanceId", instance.InstanceId, "error", err)
		return false
	}
	if len(result.Models) == 0 {
		log.Info("[MigrationModels] 无可迁移模型", "instanceId", instance.InstanceId)
		return false
	}

	// openclaw 使用 instance_models 表（多模型 primary/fallback）
	// hermes/ace 使用旧的 instances.custom_model_config 字段（单模型）
	// 兼容 hermes/ace 的自定义类型走相同的旧字段路径。
	runtimeType := model.GetAgentRuntimeType(ctx, instance.AgentType)
	if runtimeType == model.AgentTypeHermes || runtimeType == model.AgentTypeLightclawACE {
		// 找 primary 模型（只写第一个）
		var primary *migrationModelEntry
		for i := range result.Models {
			if result.Models[i].Role == model.ModelRolePrimary {
				primary = &result.Models[i]
				break
			}
		}
		if primary == nil && len(result.Models) > 0 {
			primary = &result.Models[0]
		}
		if primary == nil {
			return false
		}
		cfg := customModelConfig{
			Provider:   "custom",
			ModelID:    primary.ModelID,
			ModelName:  primary.ModelName,
			APIKey:     primary.APIKey,
			URL:        primary.BaseURL,
			ModelType:  primary.APIMode,
			InputTypes: primary.InputTypes,
			ContextLen: primary.ContextLen,
		}
		cfgJSON, err := json.Marshal(cfg)
		if err != nil {
			log.Warn("[MigrationModels] 序列化模型配置失败", "model_id", primary.ModelID, "error", err)
			return false
		}
		model.DB(ctx).Model(instance).Updates(map[string]interface{}{
			"ai_model_id":         0,
			"custom_model_config": string(cfgJSON),
		})
		log.Info("[MigrationModels] 模型迁移完成（旧字段）", "instanceId", instance.InstanceId, "model_id", primary.ModelID)
		return true
	}

	// openclaw：清空 instance_models 并写入多模型记录
	model.DB(ctx).Unscoped().Where("instance_id = ?", instance.ID).Delete(&model.InstanceModel{})
	model.DB(ctx).Model(instance).Updates(map[string]interface{}{
		"ai_model_id":         0,
		"custom_model_config": "",
	})

	// 写入源实例的模型配置，同时检查是否有 primary 模型成功写入
	for i, m := range result.Models {
		cfg := customModelConfig{
			Provider:   "custom",
			ModelID:    m.ModelID,
			ModelName:  m.ModelName,
			APIKey:     m.APIKey,
			URL:        m.BaseURL,
			ModelType:  m.APIMode,
			InputTypes: m.InputTypes,
			ContextLen: m.ContextLen,
		}
		cfgJSON, err := json.Marshal(cfg)
		if err != nil {
			log.Warn("[MigrationModels] 序列化模型配置失败，跳过", "model_id", m.ModelID, "error", err)
			continue
		}
		im := model.InstanceModel{
			InstanceID:        instance.ID,
			AIModelID:         0,
			CustomModelID:     m.ModelID,
			Role:              m.Role,
			SortOrder:         i,
			CustomModelConfig: string(cfgJSON),
		}
		if err := model.DB(ctx).Create(&im).Error; err != nil {
			log.Warn("[MigrationModels] 写入模型记录失败", "model_id", m.ModelID, "error", err)
			continue
		}
		if m.Role == model.ModelRolePrimary {
			isPrimaryValid = true
		}
	}

	log.Info("[MigrationModels] 模型迁移完成", "instanceId", instance.InstanceId, "count", len(result.Models), "isPrimaryValid", isPrimaryValid)
	return isPrimaryValid
}

// agentMigrationDirName 返回指定 agent type 对应的 home 子目录名（如 ".hermes"）。
// 对自定义类型按 compatible_with 解析为兼容内置类型的目录布局。
func agentMigrationDirName(ctx context.Context, agentType string) string {
	switch model.GetAgentRuntimeType(ctx, agentType) {
	case model.AgentTypeHermes:
		return ".hermes"
	case model.AgentTypeLightclawACE:
		return ".lightclaw"
	default:
		return ".openclaw"
	}
}

// agentMigrationExcludedPaths 返回打包时应排除的路径列表（导出侧）。
// 包括：平台目录、运行时文件、缓存目录。
// 对自定义类型按 compatible_with 解析为兼容内置类型的路径布局。
func agentMigrationExcludedPaths(ctx context.Context, agentType string) []string {
	switch model.GetAgentRuntimeType(ctx, agentType) {
	case model.AgentTypeHermes:
		return []string{"hermes-agent", "audio_cache", "image_cache", "cache", "logs", "sandboxes", "gateway.pid", "config.yaml.lock", "cron/.tick.lock"}
	case model.AgentTypeLightclawACE:
		return []string{"venv", "logs"}
	default: // openclaw
		return []string{"browser-existing-session", "logs", "update-check.json"}
	}
}

// agentMigrationPreservedPaths 返回导入时需从备份恢复的路径列表（导入侧）。
// 仅包含平台相关目录（打包时被排除，目标机器安装时自带，解压后需恢复本机版本）。
// 不包含运行时文件（gateway.pid 等）和缓存目录（目标机器会重新生成）。
// 对自定义类型按 compatible_with 解析为兼容内置类型的路径布局。
func agentMigrationPreservedPaths(ctx context.Context, agentType string) []string {
	switch model.GetAgentRuntimeType(ctx, agentType) {
	case model.AgentTypeHermes:
		return []string{"hermes-agent"}
	case model.AgentTypeLightclawACE:
		return []string{"venv"}
	default: // openclaw
		return []string{}
	}
}

// buildMigrationScript 根据迁移记录和上传凭证生成可直接执行的 shell 脚本字符串。
// agentType 用于确定源实例上的 agent 目录（目标实例与源实例类型一致）。
func buildMigrationScript(ctx context.Context, m *model.AgentMigration, cred *SMHUploadCredential, agentType string, getToken func() (string, error)) (string, error) {
	smhConfig := model.GetSMHConfig(ctx)
	if !smhConfig.IsConfigured() {
		return "", hcommon.I18nError(i18n.MsgSmhNotConfigured)
	}

	accessToken, err := getToken()
	if err != nil {
		return "", hcommon.I18nRichError(err, i18n.MsgSmhGetTokenFailed)
	}

	headersB64 := base64.StdEncoding.EncodeToString(func() []byte {
		b, _ := json.Marshal(cred.PartHeaders)
		return b
	}())

	// 根据 agent type 确定源实例上的 agent 目录和排除列表
	agentDirName := agentMigrationDirName(ctx, agentType)
	excludedPaths := agentMigrationExcludedPaths(ctx, agentType)
	excludedPathsJSON, err := json.Marshal(excludedPaths)
	if err != nil {
		return "", fmt.Errorf("marshal migration excluded paths: %w", err)
	}
	excludedPathsB64 := base64.StdEncoding.EncodeToString(excludedPathsJSON)
	allowAgentRootChangeWarning := "0"
	if model.GetAgentRuntimeType(ctx, agentType) == model.AgentTypeHermes {
		allowAgentRootChangeWarning = "1"
	}

	// 将排除路径列表序列化为 shell 数组赋值
	excludeArgs := ""
	for _, p := range excludedPaths {
		excludeArgs += " --exclude=" + shellQuote(p)
	}

	envVars := strings.Join([]string{
		`PART_URL_TEMPLATE=` + shellQuote(cred.PartURLTemplate),
		`PART_HEADERS_B64=` + shellQuote(headersB64),
		`CONFIRM_KEY=` + shellQuote(cred.ConfirmKey),
		`ACCESS_TOKEN=` + shellQuote(accessToken),
		`LIBRARY_ID=` + shellQuote(smhConfig.LibraryId),
		`SPACE_ID=` + shellQuote(smhConfig.CommonSpace),
		`SMH_ENDPOINT=` + shellQuote(strings.TrimSuffix(smhConfig.Endpoint, "/")),
		`FILE_KEY=` + shellQuote(m.FileKey),
		`AGENT_DIR="$HOME/` + agentDirName + `"`,
		`EXCLUDE_ARGS=` + shellQuote(excludeArgs),
		`EXCLUDED_PATHS_B64=` + shellQuote(excludedPathsB64),
		`ALLOW_AGENT_ROOT_CHANGE_WARNING=` + shellQuote(allowAgentRootChangeWarning),
	}, " \\\n")

	exportScript, err := LoadScript("export_migration.sh")
	if err != nil {
		return "", hcommon.I18nRichError(err, i18n.MsgMigrationLoadExportScriptFailed)
	}
	if !strings.HasSuffix(exportScript, "\n") {
		exportScript += "\n"
	}
	return envVars + " \\\nbash <<'BASH'\n" + exportScript + "BASH", nil
}

// shellQuote 对字符串做单引号转义，安全嵌入 shell 赋值语句。
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
