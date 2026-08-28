package task

import (
	"context"
	"log/slog"
	"time"

	"hatchery/controller"
	"hatchery/model"
)

// 启动龙虾医生结束诊断定时任务，每 1 分钟扫描一次：
// - 处理 ending 状态的会话（回滚 + 清理资源 + 标记 ended）
func init() {
	RegisterTask(TaskDef{
		Name:         "doctor-ending",
		Interval:     1 * time.Minute,
		RunFunc:      processDoctorEnding,
		NeedDistLock: false, // 内部自行 TryLock
		PerTenant:    true,
	})
}

func processDoctorEnding(ctx context.Context) {
	lock, err := model.TryLock(ctx, "doctor:ending")
	if err != nil {
		slog.Info("[DoctorEnding] 获取分布式锁失败，跳过本轮",
			"error", err)
		return
	}
	defer lock.Release()

	log := controller.Logger(ctx)

	log.Info("[DoctorEnding] 开始扫描 ending 会话")

	var endingSessions []model.DoctorSession
	if dbErr := model.DB(ctx).Where("status = ?",
		model.DoctorStatusEnding).
		Find(&endingSessions).Error; dbErr != nil {
		log.Error("[DoctorEnding] 查询 ending 会话失败",
			"error", dbErr)
		return
	}

	if len(endingSessions) == 0 {
		log.Info("[DoctorEnding] 无 ending 会话")
		return
	}

	log.Info("[DoctorEnding] ending 会话数",
		"count", len(endingSessions))

	for _, s := range endingSessions {
		sessionLog := log.With("session_id", s.ID)

		// 回滚
		if s.RollbackRequested && s.HasSnapshot &&
			s.SnapshotFileKey != "" {
			sessionLog.Info("[DoctorEnding] 执行回滚")
			var targetInst model.Instance
			if dbErr := model.DB(ctx).First(&targetInst,
				s.TargetInstanceID).Error; dbErr != nil {
				sessionLog.Error(
					"[DoctorEnding] 查询目标实例失败",
					"error", dbErr)
			} else {
				smhURL, urlErr :=
					controller.BuildCommonSMHDownloadURL(
						ctx, s.SnapshotFileKey, true)
				if urlErr != nil {
					sessionLog.Error(
						"[DoctorEnding] 生成 SMH 下载 URL 失败",
						"error", urlErr)
				} else {
					runtimeUser := targetInst.RuntimeUser
					if runtimeUser == "" {
						runtimeUser = "root"
					}
					params := map[string]string{
						"url":          smhURL,
						"runtime_user": runtimeUser,
					}
					_, restoreErr := controller.RunScript(
						ctx,
						targetInst.InstanceId,
						"restore_post_reinstall.sh",
						600,
						targetInst.RuntimeUser,
						nil, params,
					)
					if restoreErr != nil {
						sessionLog.Error(
							"[DoctorEnding] 回滚失败",
							"error", restoreErr)
					} else {
						sessionLog.Info(
							"[DoctorEnding] 回滚成功")
					}
				}
			}
		}

		// 清理
		controller.CleanupDoctorSessionFn(ctx, &s)
	}

	log.Info("[DoctorEnding] 扫描完成")
}
