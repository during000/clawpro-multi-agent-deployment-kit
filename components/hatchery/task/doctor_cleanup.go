package task

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"hatchery/common"
	"hatchery/controller"
	"hatchery/i18n"
	"hatchery/model"

	tccommon "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	"gorm.io/gorm"
)

// 启动龙虾医生清理定时任务，每 5 分钟扫描一次：
// - 清理超时的 active 会话（通过 session 文件 mtime 判断）
// - 刷新即将过期的 STS 临时密钥
func init() {
	RegisterTask(TaskDef{
		Name:         "doctor-cleanup",
		Interval:     5 * time.Minute,
		RunFunc:      cleanupDoctorSessions,
		NeedDistLock: false, // 内部自行 TryLock
		PerTenant:    true,
	})
}

func cleanupDoctorSessions(ctx context.Context) {
	// 分布式锁：多实例部署时只有一个节点执行
	lock, err := model.TryLock(ctx, "doctor:cleanup")
	if err != nil {
		slog.Info("[DoctorCleanup] 获取分布式锁失败，跳过本轮",
			"error", err)
		return
	}
	defer lock.Release()

	const timeout = 12 * time.Hour
	log := controller.Logger(ctx)

	log.Info("[DoctorCleanup] 开始扫描")

	// 0. 刷新即将过期的 STS 临时密钥
	controller.RefreshDoctorSTSFn(ctx)

	// 1. active → 通过远端 session 文件 mtime 判断超时
	// 查询活跃会话
	var activeSessions []model.DoctorSession
	if dbErr := model.DB(ctx).Where("status = ?",
		model.DoctorStatusActive).
		Find(&activeSessions).Error; dbErr != nil {
		log.Error("[DoctorCleanup] 查询 active 会话失败",
			"error", dbErr)
		return
	}

	if len(activeSessions) > 0 {
		log.Info("[DoctorCleanup] 活跃会话数",
			"count", len(activeSessions))
	}

	for _, s := range activeSessions {
		// 结构性缺陷：DoctorInstanceID 为空说明会话永远不会变 active，
		// 立即结束避免脏数据卡住
		if s.DoctorInstanceID == nil {
			log.Info("[DoctorCleanup] 会话缺少 DoctorInstanceID，立即结束",
				"session_id", s.ID)
			endDoctorSession(ctx, &s)
			continue
		}

		var doctorInst model.Instance
		instErr := model.DB(ctx).First(
			&doctorInst, *s.DoctorInstanceID).Error
		if instErr != nil {
			if errors.Is(instErr, gorm.ErrRecordNotFound) {
				log.Info("[DoctorCleanup] 会话关联的 Instance 记录不存在，立即结束",
					"session_id", s.ID,
					"doctor_instance_id", *s.DoctorInstanceID)
				endDoctorSession(ctx, &s)
			} else {
				log.Error("[DoctorCleanup] 查询 DoctorInstance 失败，跳过等待重试",
					"session_id", s.ID,
					"error", instErr)
			}
			continue
		}
		if doctorInst.InstanceId == "" {
			log.Info("[DoctorCleanup] 会话关联的 CVM InstanceId 为空，立即结束",
				"session_id", s.ID)
			endDoctorSession(ctx, &s)
			continue
		}

		sessionLog := log.With("session_id", s.ID,
			"cvm_instance_id", doctorInst.InstanceId)

		// 目标实例已被删除：龙虾医生失去诊断目标，立即结束会话以释放资源
		var target model.Instance
		targetErr := model.DB(ctx).Unscoped().
			First(&target, s.TargetInstanceID).Error
		if targetErr != nil {
			if errors.Is(targetErr, gorm.ErrRecordNotFound) {
				sessionLog.Info(
					"[DoctorCleanup] 目标实例记录不存在，立即结束会话",
					"target_instance_id", s.TargetInstanceID)
				endDoctorSession(ctx, &s)
				continue
			}
			sessionLog.Error("[DoctorCleanup] 查询目标实例失败，跳过等待重试",
				"error", targetErr)
			continue
		}
		if !target.DeletedAt.Time.IsZero() {
			sessionLog.Info(
				"[DoctorCleanup] 目标实例已删除，立即结束会话",
				"target_instance_id", s.TargetInstanceID)
			endDoctorSession(ctx, &s)
			continue
		}

		sessionLog.Info("[DoctorCleanup] 检查会话 mtime")

		result := controller.GetDoctorSessionMtimeFn(
			ctx, doctorInst.InstanceId)

		// 获取失败：启动后台验证协程持续探测，
		// 只有持续失败超过 12 小时才兜底结束，避免 TAT 偶发故障导致死会话
		if result.Err != nil {
			sessionLog.Error(
				"[DoctorCleanup] 无法获取 session mtime，启动验证协程",
				"error", result.Err)
			startProbeCheckerFn(ctx, s, doctorInst.InstanceId, log)
			continue
		}

		// 无文件：用户尚未开始对话，判断实例激活后是否超过 12 小时
		if result.NoFiles {
			activatedAt := s.CreatedAt
			if s.ActivatedAt != nil && !s.ActivatedAt.IsZero() {
				activatedAt = *s.ActivatedAt
			}
			if time.Since(activatedAt) > timeout {
				sessionLog.Info(
					"[DoctorCleanup] 用户未开始对话且超时，自动结束",
					"activated_at", activatedAt.Format(time.RFC3339))
				endDoctorSession(ctx, &s)
			} else {
				sessionLog.Info(
					"[DoctorCleanup] 无对话文件但未超时，跳过")
			}
			continue
		}

		// 有文件：根据 mtime 判断是否超时
		sessionLog.Info("[DoctorCleanup] session mtime",
			"mtime", result.Mtime.Format(time.RFC3339),
			"idle", time.Since(result.Mtime).String())

		if time.Since(result.Mtime) > timeout {
			sessionLog.Info(
				"[DoctorCleanup] 活跃会话超时，自动结束",
				"last_mtime", result.Mtime.Format(time.RFC3339))
			endDoctorSession(ctx, &s)
		}
	}

	log.Info("[DoctorCleanup] 扫描完成")

	// 2. 清理已结束但 instance 记录未删除的残留节点
	cleanupOrphanedDoctorNodes(ctx, log)

	// 3. 清理目标实例已删除但快照未清理的残留快照
	cleanupOrphanedSnapshots(ctx, log)
}

// cleanupOrphanedDoctorNodes 扫描 failed/ended 状态的 session，
// 如果关联的 instance 记录仍存在（deleted_at IS NULL），则销毁 CVM 并删除记录。
func cleanupOrphanedDoctorNodes(ctx context.Context, log *slog.Logger) {
	var sessions []model.DoctorSession
	if err := model.DB(ctx).Where(
		"status IN (?, ?) AND doctor_instance_id IS NOT NULL",
		model.DoctorStatusFailed, model.DoctorStatusEnded,
	).Find(&sessions).Error; err != nil {
		log.Error("[DoctorCleanup] 查询残留节点 session 失败",
			"error", err)
		return
	}

	for _, s := range sessions {
		var inst model.Instance
		err := model.DB(ctx).First(
			&inst, *s.DoctorInstanceID).Error
		if err != nil {
			// Instance 记录已删除或不存在，直接软删除 session 避免重复扫描
			log.Info("[DoctorCleanup] 实例已不存在，清理 session 记录",
				"session_id", s.ID,
				"doctor_instance_id", *s.DoctorInstanceID)
			if delErr := model.DB(ctx).Delete(&s).Error; delErr != nil {
				log.Error("[DoctorCleanup] 软删 session 失败",
					"session_id", s.ID,
					"error", delErr)
			}
			continue
		}

		log.Info("[DoctorCleanup] 发现残留龙虾医生节点，清理",
			"session_id", s.ID,
			"instance_db_id", inst.ID,
			"cvm_instance_id", inst.InstanceId,
			"session_status", s.Status)
		destroyDoctorNode(ctx, inst.ID, log)
	}
}

// startProbeCheckerFn 允许测试替换 startProbeChecker，避免后台协程泄漏。
var startProbeCheckerFn = startProbeChecker

// startProbeChecker 对 mtime 探测失败的 session 启动后台验证协程。
// 使用分布式锁（doctor:probe:{sessionID}）保证同一 session 只有一个协程在跑。
// 协程每 interval 重试 mtime 探测，最多 maxAttempts 次：
//   - 探测成功 → 退出
//   - maxAttempts 次全部失败且 CreatedAt 超过 12 小时 → 兜底结束会话
//   - maxAttempts 次全部失败但 CreatedAt 未超 12 小时 → 退出，下轮定时任务会重新启动
func startProbeChecker(
	ctx context.Context,
	session model.DoctorSession,
	cvmInstanceID string,
	log *slog.Logger,
) {
	probeCtx, cancel := context.WithTimeout(
		common.DetachContext(ctx), 55*time.Minute)

	lockKey := fmt.Sprintf("doctor:probe:%d", session.ID)
	lock, err := model.TryLock(probeCtx, lockKey)
	if err != nil {
		log.Info("[DoctorCleanup] 验证协程已在运行，跳过",
			"session_id", session.ID,
			"lock_key", lockKey)
		cancel()
		return
	}

	log.Info("[DoctorCleanup] 启动验证协程",
		"session_id", session.ID,
		"cvm_instance_id", cvmInstanceID)

	go func() {
		defer cancel()
		defer lock.Release()
		runProbeChecker(probeCtx, session.ID, cvmInstanceID, log,
			5*time.Minute, 10, 12*time.Hour)
	}()
}

// runProbeChecker 执行验证协程的核心探测循环。
// 抽取为独立函数便于测试，由 startProbeChecker 在后台协程中调用。
func runProbeChecker(
	ctx context.Context,
	sessionID uint,
	cvmInstanceID string,
	log *slog.Logger,
	interval time.Duration,
	maxAttempts int,
	probeTimeout time.Duration,
) {
	sessionLog := log.With("session_id", sessionID,
		"cvm_instance_id", cvmInstanceID,
		"probe", "checker")

	probeFails := 0
	dbFails := 0

	for probeFails < maxAttempts && dbFails < maxAttempts {
		var s model.DoctorSession
		if dbErr := model.DB(ctx).First(&s, sessionID).Error; dbErr != nil {
			if errors.Is(dbErr, gorm.ErrRecordNotFound) {
				sessionLog.Info("[DoctorCleanup] 验证协程：session 已不存在，退出")
				return
			}
			dbFails++
			sessionLog.Error("[DoctorCleanup] 验证协程：查询 session 失败",
				"error", dbErr,
				"db_fails", dbFails,
				"max_attempts", maxAttempts)
			if dbFails >= maxAttempts {
				sessionLog.Info("[DoctorCleanup] 验证协程：DB 持续不可用，退出")
				return
			}
			time.Sleep(interval)
			continue
		}
		dbFails = 0

		if s.Status != model.DoctorStatusActive {
			sessionLog.Info("[DoctorCleanup] 验证协程：session 已非 active，退出",
				"status", s.Status)
			return
		}

		result := controller.GetDoctorSessionMtimeFn(ctx, cvmInstanceID)
		if result.Err == nil {
			sessionLog.Info("[DoctorCleanup] 验证协程：探测恢复成功，退出")
			return
		}

		probeFails++
		sessionLog.Info("[DoctorCleanup] 验证协程：探测失败",
			"error", result.Err,
			"probe_fails", probeFails,
			"max_attempts", maxAttempts)

		if probeFails >= maxAttempts {
			if time.Since(s.CreatedAt) > probeTimeout {
				sessionLog.Info(
					"[DoctorCleanup] 验证协程：连续探测失败且超过兜底超时，结束会话",
					"probe_fails", probeFails,
					"created_at", s.CreatedAt.Format(time.RFC3339))
				endDoctorSession(ctx, &s)
			} else {
				sessionLog.Info(
					"[DoctorCleanup] 验证协程：连续探测失败但未超兜底超时，退出等待下轮",
					"probe_fails", probeFails,
					"created_at", s.CreatedAt.Format(time.RFC3339))
			}
			return
		}

		select {
		case <-ctx.Done():
			sessionLog.Info("[DoctorCleanup] 验证协程：context 取消，退出")
			return
		case <-time.After(interval):
		}
	}
}

// doctorServiceUsername 是龙虾医生后台服务作为审计主体的用户名，
// 用于区分"服务自动触发"与"真人发起"的审计记录。
func doctorServiceUsername(ctx context.Context) string {
	return i18n.T(ctx, i18n.MsgDoctorServiceUsername)
}

// endDoctorSession 将超时的活跃会话标记为 ending，
// 由 doctor_ending 定时任务统一处理后续清理。
func endDoctorSession(ctx context.Context, session *model.DoctorSession) {
	startedAt := time.Now()
	err := model.DB(ctx).Model(session).
		Update("status", model.DoctorStatusEnding).Error
	status := "success"
	if err != nil {
		status = "failed"
	}
	model.LogAudit(ctx, startedAt, session.UserID, doctorServiceUsername(ctx),
		"doctor_session_end_timeout", "doctor_session",
		fmt.Sprintf("%d", session.ID), status)
}

// destroyDoctorNode 销毁龙虾医生 CVM 并删除实例记录。
// CVM 销毁失败时不删除实例记录，等待下次清理重试。
func destroyDoctorNode(
	ctx context.Context,
	instanceDBID uint, log *slog.Logger,
) {
	var inst model.Instance
	if model.DB(ctx).First(
		&inst, instanceDBID).Error != nil {
		return
	}

	// 更新实例状态为 delete/processing
	now := time.Now()
	if err := model.DB(ctx).Model(&model.Instance{}).Where("id = ?", inst.ID).Updates(map[string]interface{}{
		"current_operation":            "delete",
		"current_operation_state":      "processing",
		"current_operation_updated_at": &now,
		"last_cvm_state":               "TERMINATING",
	}).Error; err != nil {
		log.Error("[DoctorCleanup] 标记实例为 delete/processing 失败，继续尝试销毁 CVM",
			"instance_db_id", inst.ID,
			"error", err)
	}

	if inst.InstanceId != "" {
		client, err := controller.GetCVMClient(ctx)
		if err != nil {
			log.Error(
				"[DoctorCleanup] 创建 CVM 客户端失败",
				"error", err)
			return // CVM 客户端创建失败，不删除记录，等待下次重试
		}

		req := cvm.NewTerminateInstancesRequest()
		req.InstanceIds = tccommon.StringPtrs(
			[]string{inst.InstanceId})
		resp, err := client.TerminateInstances(req)
		if err != nil {
			log.Error(
				"[DoctorCleanup] 销毁 CVM 失败，保留实例记录等待重试",
				"instance_id", inst.InstanceId,
				"error", err)
			return // 销毁失败，不删除实例记录
		}

		log.Info(
			"[DoctorCleanup] CVM 已销毁",
			"instance_id", inst.InstanceId,
			"request_id", resp.Response.RequestId)
	}

	// CVM 销毁成功或无 CVM 实例，删除数据库记录
	startedAt := time.Now()
	delErr := model.DB(ctx).Delete(
		&model.Instance{}, instanceDBID).Error
	status := "success"
	if delErr != nil {
		status = "failed"
	}
	model.LogAudit(ctx, startedAt, inst.UserID, doctorServiceUsername(ctx),
		"doctor_node_destroy", "instance",
		fmt.Sprintf("%d", instanceDBID), status)
}

// cleanupOrphanedSnapshots 扫描有快照但未删除的 session，
// 如果目标实例已被删除（软删除），则从 SMH 删除快照并标记 snapshot_deleted=true。
func cleanupOrphanedSnapshots(ctx context.Context, log *slog.Logger) {
	var sessions []model.DoctorSession
	if err := model.DB(ctx).Where(
		"has_snapshot = ? AND snapshot_deleted = ? AND snapshot_file_key != ''",
		true, false,
	).Find(&sessions).Error; err != nil {
		log.Error("[DoctorCleanup] 查询残留快照 session 失败",
			"error", err)
		return
	}

	for _, s := range sessions {
		// 检查目标实例是否已被删除（Unscoped 查询包含软删除记录）
		var target model.Instance
		err := model.DB(ctx).Unscoped().
			First(&target, s.TargetInstanceID).Error
		if err != nil {
			// 记录不存在（硬删或从未存在），也应该清理快照
		} else if target.DeletedAt.Time.IsZero() {
			// 目标实例未被删除，跳过
			continue
		}

		log.Info("[DoctorCleanup] 目标实例已删除，清理残留快照",
			"session_id", s.ID,
			"target_instance_id", s.TargetInstanceID,
			"snapshot_file_key", s.SnapshotFileKey)

		if delErr := controller.DeleteSMHCommonFileFn(ctx, s.SnapshotFileKey); delErr != nil {
			log.Error("[DoctorCleanup] 删除残留快照失败",
				"session_id", s.ID,
				"file_key", s.SnapshotFileKey,
				"error", delErr)
			continue
		}

		if err := model.DB(ctx).Model(&s).Update("snapshot_deleted", true).Error; err != nil {
			log.Error("[DoctorCleanup] 标记快照已清理失败",
				"session_id", s.ID,
				"file_key", s.SnapshotFileKey,
				"error", err)
			continue
		}
		log.Info("[DoctorCleanup] 残留快照已清理",
			"session_id", s.ID,
			"file_key", s.SnapshotFileKey)
	}

	// 清理目标实例已删除的 session 对话备份文件
	cleanupOrphanedSessionFiles(ctx, log)
}

// cleanupOrphanedSessionFiles 清理目标实例已删除的 session 对话备份文件。
func cleanupOrphanedSessionFiles(ctx context.Context, log *slog.Logger) {
	var sessions []model.DoctorSession
	if err := model.DB(ctx).Where(
		"status IN (?, ?) AND sessions_deleted = ?",
		model.DoctorStatusEnded, model.DoctorStatusFailed, false,
	).Find(&sessions).Error; err != nil {
		log.Error("[DoctorCleanup] 查询 session 备份清理列表失败",
			"error", err)
		return
	}

	for _, s := range sessions {
		var target model.Instance
		err := model.DB(ctx).Unscoped().
			First(&target, s.TargetInstanceID).Error
		if err != nil {
			// 记录不存在，应清理
		} else if target.DeletedAt.Time.IsZero() {
			// 目标实例未删除，跳过
			continue
		}

		fileKey := controller.DoctorSessionsSMHKey(s.UserID, s.TargetInstanceID)
		log.Info("[DoctorCleanup] 目标实例已删除，清理 session 对话备份",
			"session_id", s.ID,
			"target_instance_id", s.TargetInstanceID,
			"file_key", fileKey)

		if delErr := controller.DeleteSMHCommonFileFn(ctx, fileKey); delErr != nil {
			log.Error("[DoctorCleanup] 删除 session 备份失败",
				"session_id", s.ID,
				"file_key", fileKey,
				"error", delErr)
			continue
		}

		if err := model.DB(ctx).Model(&s).Update("sessions_deleted", true).Error; err != nil {
			log.Error("[DoctorCleanup] 标记 session 备份已清理失败",
				"session_id", s.ID,
				"file_key", fileKey,
				"error", err)
			continue
		}
		log.Info("[DoctorCleanup] session 备份已清理",
			"session_id", s.ID,
			"file_key", fileKey)
	}
}
