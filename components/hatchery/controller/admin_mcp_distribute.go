package controller

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"sync"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ========== 端点 8: POST /admin/mcp/distribute — 发起批量下发 ==========

const mcpDistributionBatchSize = 200

type mcpDistributionTarget struct {
	InstanceID  uint   `gorm:"column:instance_id"`
	InstanceCID string `gorm:"column:cvm_instance_id"`
}

func normalizeMCPDistributionStatuses(statuses []string) ([]string, error) {
	return normalizeDistributionStatuses(
		statuses,
		[]string{"uninstalled", "installed", "outdated", "failed"},
		[]string{"installing"},
	)
}

func HandleDistributeMcp(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	var req struct {
		ServiceID string `json:"service_id"`
		Version   string `json:"version"`
		distributionSelection
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequest))
		return
	}
	if err := req.distributionSelection.validate(); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
		return
	}
	if req.SelectAll {
		if _, err := normalizeMCPDistributionStatuses(req.Statuses); err != nil {
			writeError(w, r, http.StatusBadRequest, hcommon.EnsureRichErrorOrPanic(err))
			return
		}
	}

	if req.ServiceID == "" {
		writeError(w, r, http.StatusBadRequest,
			hcommon.I18nError(i18n.MsgBadRequestParamRequired, "service_id"))
		return
	}
	if req.Version == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMcpDistributeVersionRequired))
		return
	}
	if !req.SelectAll && len(req.InstanceIDs) > 500 {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMcpDistributeMaxInstances500))
		return
	}

	// instance_ids 去重
	seen := make(map[uint]bool, len(req.InstanceIDs))
	var uniqueIDs []uint
	for _, id := range req.InstanceIDs {
		if !seen[id] {
			seen[id] = true
			uniqueIDs = append(uniqueIDs, id)
		}
	}
	req.InstanceIDs = uniqueIDs

	// 查找 MCP
	var server model.McpServer
	if err := model.DB(r.Context()).Where("service_id = ?", req.ServiceID).First(&server).Error; err != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgMcpNotFound))
		return
	}

	// 查找版本
	var version model.McpVersion
	if err := model.DB(r.Context()).Where("mcp_id = ? AND version = ?", server.ID, req.Version).First(&version).Error; err != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgMcpVersionNotFoundDetail, req.Version))
		return
	}

	// 获取分布式锁
	resource := fmt.Sprintf("mcp_distribute:%s:%s", req.ServiceID, req.Version)
	lock, err := model.AcquireLock(r.Context(), resource, 30*time.Minute)
	if err != nil {
		writeError(w, r, http.StatusConflict, hcommon.I18nError(i18n.MsgMcpDistributeLockBusy))
		return
	}

	if req.SelectAll {
		var operatorID uint
		if user, userErr := RequestUser(r); user != nil && userErr == nil {
			operatorID = user.ID
		}
		task, total, createErr := createMCPSelectAllTask(r.Context(), server, version, operatorID, req.distributionSelection)
		if createErr != nil {
			lock.Release()
			var richErr *hcommon.RichError
			if errors.As(createErr, &richErr) {
				writeError(w, r, http.StatusBadRequest, richErr)
			} else {
				slog.Error("[MCPSelectAll] 创建下发任务失败", "service_id", server.ServiceID, "version", version.Version, "error", createErr)
				writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgMcpDistributeCreateTaskFailed))
			}
			return
		}
		w.WriteHeader(http.StatusAccepted)
		jsonOK(w, map[string]interface{}{
			"task_id": task.ID,
			"total":   total,
		})
		taskCtx := i18n.WithPrinter(hcommon.DetachContext(r.Context()), r.Context())
		go func() {
			defer lock.Release()
			defer recoverMCPSelectAllTaskPanic(taskCtx, task)
			runMCPSelectAllTask(taskCtx, server, version, task)
		}()
		return
	}

	// 过滤实例：agent_type=openclaw + RUNNING + 当前 identifier
	var instances []model.Instance
	model.DB(r.Context()).Where("id IN ?", req.InstanceIDs).
		Where("instance_id != ''").
		Find(&instances)

	type warningItem struct {
		InstanceID uint   `json:"instance_id"`
		Reason     string `json:"reason"`
		Detail     string `json:"detail"`
	}

	var validInstances []model.Instance
	var warnings []warningItem

	for _, inst := range instances {
		if err := requireNoResourceAdjustment(&inst); err != nil {
			warnings = append(warnings, warningItem{
				InstanceID: inst.ID,
				Reason:     reasonOperationInProgress,
				Detail:     i18n.T(r.Context(), i18n.MsgOperationInProgress),
			})
			continue
		}
		if !model.AgentTypeSupportsPlugin(r.Context(), inst.AgentType) {
			warnings = append(warnings, warningItem{
				InstanceID: inst.ID,
				Reason:     "wrong_agent_type",
				Detail:     fmt.Sprintf("agent_type = %s", inst.AgentType),
			})
			continue
		}
		validInstances = append(validInstances, inst)
	}

	// 检查请求中不存在的实例
	foundIDs := make(map[uint]bool)
	for _, inst := range instances {
		foundIDs[inst.ID] = true
	}
	for _, id := range req.InstanceIDs {
		if !foundIDs[id] {
			warnings = append(warnings, warningItem{
				InstanceID: id,
				Reason:     "not_found",
				Detail:     "instance not found",
			})
		}
	}

	// 无有效实例时提前返回
	if len(validInstances) == 0 {
		lock.Release()
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgMcpDistributeNoValidInstance))
		return
	}

	// 获取操作者 ID
	var operatorID uint
	if user, err := RequestUser(r); user != nil && err == nil {
		operatorID = user.ID
	}

	// 预查 MCP 名称（在 goroutine 启动前）
	mcpName := server.Name

	// 事务：创建 task + records + upsert installations
	var task model.McpDistributionTask
	var records []model.McpDistributionRecord

	txErr := model.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		task = model.McpDistributionTask{
			MCPID:                server.ID,
			McpSnapshotServiceID: server.ServiceID,
			McpSnapshotName:      server.Name,
			VersionID:            version.ID,
			VersionSnapshot:      version.Version,
			OperatorID:           operatorID,
			Total:                len(validInstances),
			Status:               "running",
		}
		if err := tx.Create(&task).Error; err != nil {
			return err
		}

		for _, inst := range validInstances {
			record := model.McpDistributionRecord{
				TaskID:          task.ID,
				MCPID:           server.ID,
				InstanceID:      inst.ID,
				InstanceCID:     inst.InstanceId,
				VersionSnapshot: version.Version,
				Status:          "pending",
			}
			if err := tx.Create(&record).Error; err != nil {
				return err
			}
			records = append(records, record)

			// upsert installation 为 Installing
			installation := model.McpInstallation{
				InstanceID:    inst.ID,
				MCPID:         server.ID,
				ServiceID:     server.ServiceID,
				Name:          mcpName,
				InstallStatus: model.McpInstalling,
				LastTaskID:    task.ID,
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "identifier"}, {Name: "instance_id"}, {Name: "service_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"mcp_id", "name", "install_status", "last_task_id", "updated_at"}),
			}).Create(&installation).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if txErr != nil {
		lock.Release()
		slog.Error("创建 MCP 下发任务失败", "error", txErr)
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgMcpDistributeCreateTaskFailed))
		return
	}

	// 构造响应
	type perInstanceItem struct {
		InstanceID uint   `json:"instance_id"`
		RecordID   uint   `json:"record_id"`
		Status     string `json:"status"`
	}
	perInstance := make([]perInstanceItem, 0, len(records))
	for _, rec := range records {
		perInstance = append(perInstance, perInstanceItem{
			InstanceID: rec.InstanceID,
			RecordID:   rec.ID,
			Status:     "pending",
		})
	}

	// 返回 202
	w.WriteHeader(http.StatusAccepted)
	jsonOK(w, map[string]interface{}{
		"task_id":      task.ID,
		"total":        task.Total,
		"per_instance": perInstance,
		"warnings":     warnings,
	})

	// 异步 goroutine 执行下发
	go func(ctx context.Context) {
		defer lock.Release()

		maxConcurrency := 100
		var siteConfig model.SiteConfig
		if err := model.DB(ctx).First(&siteConfig).Error; err == nil && siteConfig.SkillDistributeConcurrency > 0 {
			maxConcurrency = siteConfig.SkillDistributeConcurrency
		}

		sem := make(chan struct{}, maxConcurrency)
		var wg sync.WaitGroup

		// base64 编码 config_json
		configB64 := base64.StdEncoding.EncodeToString([]byte(version.ConfigJSON))

		// 批量查询所有目标实例的 RuntimeUser，避免 goroutine 内逐个查询
		instIDs := make([]uint, 0, len(records))
		for _, rec := range records {
			instIDs = append(instIDs, rec.InstanceID)
		}
		instRuntimeUserMap := make(map[uint]string)
		if len(instIDs) > 0 {
			var insts []model.Instance
			model.DB(ctx).Select("id, runtime_user").Where("id IN ?", instIDs).Find(&insts)
			for _, inst := range insts {
				instRuntimeUserMap[inst.ID] = inst.RuntimeUser
			}
		}

		for _, record := range records {
			wg.Add(1)
			go func(rec model.McpDistributionRecord) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				runtimeUser := instRuntimeUserMap[rec.InstanceID]

				params := map[string]string{
					"service_id":         server.ServiceID,
					"config_json_base64": configB64,
				}

				_, runErr := RunScript(ctx, rec.InstanceCID, "mcp_set.sh", 120, runtimeUser, nil, params)

				ok := runErr == nil
				errMsg := ""
				if runErr != nil {
					errMsg = runErr.Error()
				}

				if finalizeErr := finalizeDistributionRecord(ctx, rec, server.ServiceID, mcpName, version.Version, version.ConfigJSON, ok, errMsg); finalizeErr != nil {
					slog.Error("mcp distribute finalize failed", "record_id", rec.ID, "error", finalizeErr)
				}
			}(record)
		}

		wg.Wait()
		model.DB(ctx).Model(&model.McpDistributionTask{}).Where("id = ?", task.ID).Update("status", "completed")
	}(hcommon.DetachContext(r.Context()))
}

func createMCPSelectAllTask(
	ctx context.Context,
	server model.McpServer,
	version model.McpVersion,
	operatorID uint,
	selection distributionSelection,
) (model.McpDistributionTask, int, error) {
	statuses, err := normalizeMCPDistributionStatuses(selection.Statuses)
	if err != nil {
		return model.McpDistributionTask{}, 0, err
	}
	baseQuery := model.BuildMcpInstanceQueryV2(ctx, server.ServiceID, version.Version).
		Where(model.McpInstallStatusCase(version.Version)+" IN ?", statuses).
		Where("instances.agent_type IN ?", model.GetMCPSupportedAgentTypes(ctx))
	baseQuery = model.FilterInstancesByUserGroups(ctx, baseQuery, selection.GroupIDs)
	baseQuery = applyDistributionSearch(baseQuery, selection.Search)

	var task model.McpDistributionTask
	var afterID uint
	total := 0
	for {
		var rows []mcpDistributionTarget
		if err := baseQuery.Session(&gorm.Session{}).
			Where("instances.id > ?", afterID).
			Order(clause.OrderByColumn{
				Column:  clause.Column{Table: "instances", Name: "id"},
				Reorder: true,
			}).
			Limit(mcpDistributionBatchSize).
			Scan(&rows).Error; err != nil {
			failMCPSelectAllPreparation(hcommon.DetachContext(ctx), task.ID, err)
			return model.McpDistributionTask{}, 0, err
		}
		if len(rows) == 0 {
			break
		}

		targets := make([]mcpDistributionTarget, 0, len(rows))
		seen := make(map[uint]struct{}, len(rows))
		for _, row := range rows {
			if row.InstanceID > afterID {
				afterID = row.InstanceID
			}
			if _, ok := seen[row.InstanceID]; ok {
				continue
			}
			seen[row.InstanceID] = struct{}{}
			targets = append(targets, row)
		}
		if len(targets) == 0 {
			continue
		}

		if err := model.DB(ctx).Transaction(func(tx *gorm.DB) error {
			if task.ID == 0 {
				task = model.McpDistributionTask{
					MCPID:                server.ID,
					McpSnapshotServiceID: server.ServiceID,
					McpSnapshotName:      server.Name,
					VersionID:            version.ID,
					VersionSnapshot:      version.Version,
					OperatorID:           operatorID,
					Status:               model.TaskStatusRunning,
				}
				if err := tx.Create(&task).Error; err != nil {
					return err
				}
			}

			records := make([]model.McpDistributionRecord, 0, len(targets))
			installations := make([]model.McpInstallation, 0, len(targets))
			for _, target := range targets {
				records = append(records, model.McpDistributionRecord{
					TaskID:          task.ID,
					MCPID:           server.ID,
					InstanceID:      target.InstanceID,
					InstanceCID:     target.InstanceCID,
					VersionSnapshot: version.Version,
					Status:          model.RecordStatusPending,
				})
				installations = append(installations, model.McpInstallation{
					InstanceID:    target.InstanceID,
					MCPID:         server.ID,
					ServiceID:     server.ServiceID,
					Name:          server.Name,
					InstallStatus: model.McpInstalling,
					LastTaskID:    task.ID,
				})
			}
			if err := tx.Create(&records).Error; err != nil {
				return err
			}
			return tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "identifier"}, {Name: "instance_id"}, {Name: "service_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"mcp_id", "name", "install_status", "last_task_id", "updated_at"}),
			}).Create(&installations).Error
		}); err != nil {
			failMCPSelectAllPreparation(hcommon.DetachContext(ctx), task.ID, err)
			return model.McpDistributionTask{}, 0, err
		}
		total += len(targets)
	}
	if total == 0 {
		return model.McpDistributionTask{}, 0, hcommon.I18nError(i18n.MsgMcpDistributeNoValidInstance)
	}
	task.Total = total
	if err := model.DB(ctx).Model(&task).Update("total", total).Error; err != nil {
		failMCPSelectAllPreparation(hcommon.DetachContext(ctx), task.ID, err)
		return model.McpDistributionTask{}, 0, err
	}
	return task, total, nil
}

func failMCPSelectAllPreparation(ctx context.Context, taskID uint, cause error) {
	if taskID == 0 {
		return
	}
	message := hcommon.ErrorMessageWithCtx(ctx, cause)
	var pending int64
	if err := model.DB(ctx).Model(&model.McpDistributionRecord{}).
		Where("task_id = ? AND status = ?", taskID, model.RecordStatusPending).
		Count(&pending).Error; err != nil {
		slog.Error("[MCPSelectAll] 统计准备失败记录失败", "task_id", taskID, "error", err)
	}
	if err := model.DB(ctx).Model(&model.McpDistributionRecord{}).
		Where("task_id = ? AND status = ?", taskID, model.RecordStatusPending).
		Updates(map[string]interface{}{"status": model.RecordStatusFailed, "error": message}).Error; err != nil {
		slog.Error("[MCPSelectAll] 收敛准备失败记录失败", "task_id", taskID, "error", err)
	}
	if err := model.DB(ctx).Model(&model.McpInstallation{}).
		Where("last_task_id = ?", taskID).
		Updates(map[string]interface{}{"install_status": model.McpInstallFailed, "error_message": message}).Error; err != nil {
		slog.Error("[MCPSelectAll] 收敛准备失败安装状态失败", "task_id", taskID, "error", err)
	}
	if err := model.DB(ctx).Model(&model.McpDistributionTask{}).Where("id = ?", taskID).
		Updates(map[string]interface{}{
			"total":  int(pending),
			"failed": int(pending),
			"status": model.TaskStatusCompleted,
		}).Error; err != nil {
		slog.Error("[MCPSelectAll] 收敛准备失败任务失败", "task_id", taskID, "error", err)
	}
}

func failMCPSelectAllPendingRecords(ctx context.Context, taskID uint, cause error) error {
	message := hcommon.ErrorMessageWithCtx(ctx, cause)
	return model.DB(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.McpDistributionRecord{}).
			Where("task_id = ? AND status = ?", taskID, model.RecordStatusPending).
			Updates(map[string]interface{}{
				"status": model.RecordStatusFailed,
				"error":  message,
			})
		if result.Error != nil {
			return result.Error
		}
		if err := tx.Model(&model.McpInstallation{}).
			Where("last_task_id = ? AND install_status = ?", taskID, model.McpInstalling).
			Updates(map[string]interface{}{
				"install_status": model.McpInstallFailed,
				"error_message":  message,
			}).Error; err != nil {
			return err
		}
		if result.RowsAffected == 0 {
			return nil
		}
		return tx.Model(&model.McpDistributionTask{}).
			Where("id = ?", taskID).
			UpdateColumn("failed", gorm.Expr("failed + ?", result.RowsAffected)).Error
	})
}

func recoverMCPSelectAllTaskPanic(ctx context.Context, task model.McpDistributionTask) {
	recovered := recover()
	if recovered == nil {
		return
	}
	cause := fmt.Errorf("panic: %v", recovered)
	slog.Error("[MCPSelectAll] task panic", "task_id", task.ID, "panic", recovered, "stack", string(debug.Stack()))
	if err := failMCPSelectAllPendingRecords(ctx, task.ID, cause); err != nil {
		slog.Error("[MCPSelectAll] 收敛 panic 任务失败", "task_id", task.ID, "error", err)
	}
	if err := model.DB(ctx).Model(&model.McpDistributionTask{}).
		Where("id = ?", task.ID).
		Update("status", model.TaskStatusCompleted).Error; err != nil {
		slog.Error("[MCPSelectAll] 更新 panic 任务状态失败", "task_id", task.ID, "error", err)
	}
}

func runMCPSelectAllTask(
	ctx context.Context,
	server model.McpServer,
	version model.McpVersion,
	task model.McpDistributionTask,
) {
	maxConcurrency := model.GetSiteConfig(ctx).SkillDistributeConcurrency
	if maxConcurrency <= 0 {
		maxConcurrency = 100
	}
	configB64 := base64.StdEncoding.EncodeToString([]byte(version.ConfigJSON))
	var afterID uint
	for {
		records, runtimeUsers, err := loadMCPSelectAllBatch(ctx, server, version, task.ID, afterID)
		if err != nil {
			slog.Error("[MCPSelectAll] 分批读取下发记录失败", "task_id", task.ID, "error", err)
			if convergeErr := failMCPSelectAllPendingRecords(ctx, task.ID, err); convergeErr != nil {
				slog.Error("[MCPSelectAll] 收敛未处理记录失败", "task_id", task.ID, "error", convergeErr)
			}
			break
		}
		if len(records) == 0 {
			break
		}
		afterID = records[len(records)-1].ID
		if runtimeUsers == nil {
			continue
		}
		executeMCPSelectAllBatch(ctx, server, version, records, runtimeUsers, configB64, maxConcurrency)
	}
	if err := model.DB(ctx).Model(&model.McpDistributionTask{}).
		Where("id = ?", task.ID).
		Update("status", model.TaskStatusCompleted).Error; err != nil {
		slog.Error("[MCPSelectAll] 更新任务状态失败", "task_id", task.ID, "error", err)
	}
}

func loadMCPSelectAllBatch(
	ctx context.Context,
	server model.McpServer,
	version model.McpVersion,
	taskID uint,
	afterID uint,
) ([]model.McpDistributionRecord, map[uint]string, error) {
	var records []model.McpDistributionRecord
	if err := model.DB(ctx).
		Where("task_id = ? AND id > ? AND status = ?", taskID, afterID, model.RecordStatusPending).
		Order("id ASC").
		Limit(mcpDistributionBatchSize).
		Find(&records).Error; err != nil {
		return nil, nil, err
	}
	if len(records) == 0 {
		return nil, nil, nil
	}

	instanceIDs := make([]uint, 0, len(records))
	for _, record := range records {
		instanceIDs = append(instanceIDs, record.InstanceID)
	}
	var instances []model.Instance
	if err := model.DB(ctx).
		Select("id, runtime_user").
		Where("id IN ?", instanceIDs).
		Find(&instances).Error; err != nil {
		slog.Error("[MCPSelectAll] 批量加载实例失败", "task_id", taskID, "error", err)
		for _, record := range records {
			if finalizeErr := finalizeDistributionRecord(ctx, record, server.ServiceID, server.Name, version.Version, version.ConfigJSON, false, hcommon.ErrorMessageWithCtx(ctx, err)); finalizeErr != nil {
				slog.Error("[MCPSelectAll] 收敛实例查询失败记录失败", "record_id", record.ID, "error", finalizeErr)
			}
		}
		return records, nil, nil
	}
	runtimeUsers := make(map[uint]string, len(instances))
	for _, instance := range instances {
		runtimeUsers[instance.ID] = instance.RuntimeUser
	}
	return records, runtimeUsers, nil
}

func executeMCPSelectAllBatch(
	ctx context.Context,
	server model.McpServer,
	version model.McpVersion,
	records []model.McpDistributionRecord,
	runtimeUsers map[uint]string,
	configB64 string,
	maxConcurrency int,
) {
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	defer wg.Wait()
	for _, record := range records {
		sem <- struct{}{}
		wg.Add(1)
		go func(record model.McpDistributionRecord) {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				recovered := recover()
				if recovered == nil {
					return
				}
				cause := fmt.Errorf("panic: %v", recovered)
				slog.Error("[MCPSelectAll] record panic", "record_id", record.ID, "panic", recovered, "stack", string(debug.Stack()))
				if err := finalizeDistributionRecord(
					ctx,
					record,
					server.ServiceID,
					server.Name,
					version.Version,
					version.ConfigJSON,
					false,
					hcommon.ErrorMessageWithCtx(ctx, cause),
				); err != nil {
					slog.Error("[MCPSelectAll] 收敛 panic 记录失败", "record_id", record.ID, "error", err)
				}
			}()

			runtimeUser, ok := runtimeUsers[record.InstanceID]
			if !ok {
				errMessage := i18n.T(ctx, i18n.MsgInstanceNotFound)
				if err := finalizeDistributionRecord(ctx, record, server.ServiceID, server.Name, version.Version, version.ConfigJSON, false, errMessage); err != nil {
					slog.Error("[MCPSelectAll] 收敛缺失实例记录失败", "record_id", record.ID, "error", err)
				}
				return
			}
			_, runErr := RunScript(ctx, record.InstanceCID, "mcp_set.sh", 120, runtimeUser, nil, map[string]string{
				"service_id":         server.ServiceID,
				"config_json_base64": configB64,
			})
			errMessage := ""
			if runErr != nil {
				errMessage = runErr.Error()
			}
			if err := finalizeDistributionRecord(ctx, record, server.ServiceID, server.Name, version.Version, version.ConfigJSON, runErr == nil, errMessage); err != nil {
				slog.Error("[MCPSelectAll] 收敛下发记录失败", "record_id", record.ID, "error", err)
			}
		}(record)
	}
}

// finalizeDistributionRecord 把一条 record 的 TAT 执行结果原子写入三张表。
// 必须在单个事务内完成，顺序：record.status → installation upsert → task 计数。
// mcpName: MCP 的显示名称，在异步 goroutine 启动前从 mcp_servers 表预查好
// configJSON: 下发的配置内容，写入 McpInstallation 便于用户端展示和编辑
func finalizeDistributionRecord(ctx context.Context, record model.McpDistributionRecord, serviceID, mcpName, version, configJSON string, ok bool, errMsg string) error {
	return model.DB(ctx).Transaction(func(tx *gorm.DB) error {
		newStatus := "success"
		if !ok {
			newStatus = "failed"
		}

		// 1. 乐观锁 CAS：只有 status='pending' 时才改（幂等保护）
		res := tx.Model(&model.McpDistributionRecord{}).
			Where("id = ? AND status = ?", record.ID, "pending").
			Updates(map[string]interface{}{"status": newStatus, "error": errMsg})
		if res.RowsAffected == 0 {
			return nil // 已被其他路径终态化，幂等返回
		}

		// 2. UPSERT installation
		installStatus := model.McpInstallSuccess
		installVersion := version
		if !ok {
			installStatus = model.McpInstallFailed
			installVersion = ""
		}
		inst := model.McpInstallation{
			InstanceID:    record.InstanceID,
			MCPID:         record.MCPID,
			ServiceID:     serviceID,
			Name:          mcpName,
			Version:       installVersion,
			InstallStatus: installStatus,
			LastTaskID:    record.TaskID,
			ErrorMessage:  errMsg,
			ConfigJSON:    configJSON,
			Source:        "admin",
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "identifier"}, {Name: "instance_id"}, {Name: "service_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"mcp_id", "name", "version", "install_status", "last_task_id", "error_message", "config_json", "source", "updated_at"}),
		}).Create(&inst).Error; err != nil {
			return err
		}

		// 3. 原子计数（避免读-改-写竞争）
		column := "success"
		if !ok {
			column = "failed"
		}
		if err := tx.Model(&model.McpDistributionTask{}).
			Where("id = ?", record.TaskID).
			UpdateColumn(column, gorm.Expr(column+" + 1")).Error; err != nil {
			return err
		}

		return nil
	})
}

// ========== 端点 9: GET /admin/mcp/tasks — 下发任务列表（含每个任务的 records） ==========

func HandleAdminMcpTasks(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	jsonAPI(w)

	serviceID := r.URL.Query().Get("service_id")
	if serviceID == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgBadRequestParamRequired, "service_id"))
		return
	}

	var server model.McpServer
	if err := model.DB(r.Context()).Where("service_id = ?", serviceID).First(&server).Error; err != nil {
		writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgMcpNotFound))
		return
	}

	page, pageSize := parsePagination(r)

	var total int64
	model.DB(r.Context()).Model(&model.McpDistributionTask{}).Where("mcp_id = ?", server.ID).Count(&total)

	var tasks []model.McpDistributionTask
	model.DB(r.Context()).Where("mcp_id = ?", server.ID).
		Order("id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&tasks)

	type recordResp struct {
		InstanceID    uint   `json:"instance_id"`
		CVMInstanceID string `json:"cvm_instance_id"`
		InstanceName  string `json:"instance_name"`
		Username      string `json:"username"`
		Status        string `json:"status"`
		Error         string `json:"error"`
	}
	type taskResp struct {
		ID        uint         `json:"id"`
		CreatedAt interface{}  `json:"created_at"`
		Operator  string       `json:"operator"`
		Version   string       `json:"version"`
		Total     int          `json:"total"`
		Success   int          `json:"success"`
		Failed    int          `json:"failed"`
		Pending   int          `json:"pending"`
		Status    string       `json:"status"`
		Records   []recordResp `json:"records"`
	}

	var result []taskResp
	if len(tasks) > 0 {
		// 批量查询所有 task 的 record 计数（消除 N+1）
		taskIDs := make([]uint, len(tasks))
		operatorIDs := make(map[uint]struct{})
		for i, t := range tasks {
			taskIDs[i] = t.ID
			if t.OperatorID > 0 {
				operatorIDs[t.OperatorID] = struct{}{}
			}
		}

		// 一次 SQL 聚合所有 task 的 record 计数
		type taskStatusCount struct {
			TaskID uint
			Status string
			Count  int
		}
		var allCounts []taskStatusCount
		model.DB(r.Context()).Model(&model.McpDistributionRecord{}).
			Select("task_id, status, COUNT(*) as count").
			Where("task_id IN ?", taskIDs).
			Group("task_id, status").
			Scan(&allCounts)
		type counters struct{ Success, Failed, Pending int }
		countMap := make(map[uint]*counters)
		for _, c := range allCounts {
			if countMap[c.TaskID] == nil {
				countMap[c.TaskID] = &counters{}
			}
			switch c.Status {
			case "success":
				countMap[c.TaskID].Success = c.Count
			case "failed":
				countMap[c.TaskID].Failed = c.Count
			case "pending":
				countMap[c.TaskID].Pending = c.Count
			}
		}

		// 批量查询操作人
		opIDs := make([]uint, 0, len(operatorIDs))
		for id := range operatorIDs {
			opIDs = append(opIDs, id)
		}
		userMap := make(map[uint]string)
		if len(opIDs) > 0 {
			var users []model.User
			model.DB(r.Context()).Where("id IN ?", opIDs).Find(&users)
			for _, u := range users {
				userMap[u.ID] = u.Username
			}
		}

		// 批量查询所有 record
		var allRecords []model.McpDistributionRecord
		model.DB(r.Context()).Where("task_id IN ?", taskIDs).Find(&allRecords)

		// 批量查询所有涉及的实例
		instIDSet := make(map[uint]struct{})
		for _, rec := range allRecords {
			instIDSet[rec.InstanceID] = struct{}{}
		}
		instIDs := make([]uint, 0, len(instIDSet))
		for id := range instIDSet {
			instIDs = append(instIDs, id)
		}
		instNameMap := make(map[uint]string)
		type instDetail struct {
			ID     uint
			Name   string
			UserID uint
		}
		instDetailMap := make(map[uint]instDetail)
		instUserMap := make(map[uint]string)
		if len(instIDs) > 0 {
			var insts []instDetail
			model.DB(r.Context()).Model(&model.Instance{}).Select("id, name, user_id").Where("id IN ?", instIDs).Scan(&insts)
			instUserIDs := make(map[uint]struct{})
			for _, inst := range insts {
				instNameMap[inst.ID] = inst.Name
				instDetailMap[inst.ID] = inst
				if inst.UserID > 0 {
					instUserIDs[inst.UserID] = struct{}{}
				}
			}
			// 批量查询实例关联的用户名
			instUserIDList := make([]uint, 0, len(instUserIDs))
			for id := range instUserIDs {
				instUserIDList = append(instUserIDList, id)
			}
			if len(instUserIDList) > 0 {
				var instUsers []model.User
				model.DB(r.Context()).Where("id IN ?", instUserIDList).Find(&instUsers)
				for _, u := range instUsers {
					instUserMap[u.ID] = u.Username
				}
			}
		}

		// 按 task_id 分组 records
		recordsByTask := make(map[uint][]model.McpDistributionRecord)
		for _, rec := range allRecords {
			recordsByTask[rec.TaskID] = append(recordsByTask[rec.TaskID], rec)
		}

		for _, t := range tasks {
			c := countMap[t.ID]
			tr := taskResp{
				ID:        t.ID,
				CreatedAt: t.CreatedAt,
				Version:   t.VersionSnapshot,
				Total:     t.Total,
				Status:    t.Status,
				Operator:  userMap[t.OperatorID],
			}
			if c != nil {
				tr.Success = c.Success
				tr.Failed = c.Failed
				tr.Pending = c.Pending
			}
			for _, rec := range recordsByTask[t.ID] {
				rr := recordResp{
					InstanceID:    rec.InstanceID,
					CVMInstanceID: rec.InstanceCID,
					InstanceName:  instNameMap[rec.InstanceID],
					Status:        rec.Status,
					Error:         rec.Error,
				}
				if inst, ok := instDetailMap[rec.InstanceID]; ok {
					rr.Username = instUserMap[inst.UserID]
				}
				tr.Records = append(tr.Records, rr)
			}
			if tr.Records == nil {
				tr.Records = []recordResp{}
			}
			result = append(result, tr)
		}
	}

	jsonOK(w, map[string]interface{}{
		"tasks":     result,
		"page":      page,
		"page_size": pageSize,
		"total":     total,
	})
}
