package task

import (
	"context"
	"log/slog"
	"time"

	"hatchery/controller"
	"hatchery/model"
)

// 启动龙虾医生会话激活定时任务，每 1 分钟扫描一次：
// - 推进 creating 状态的会话（委托 controller.ActivateDoctorSession 处理全部激活逻辑）
// - 超时的 creating 会话标记为 failed
func init() {
	RegisterTask(TaskDef{
		Name:         "doctor-activate",
		Interval:     1 * time.Minute,
		RunFunc:      processDoctorActivate,
		NeedDistLock: false, // 内部自行 TryLock
		PerTenant:    true,
	})
}

func processDoctorActivate(ctx context.Context) {
	lock, err := model.TryLock(ctx, "doctor:activate")
	if err != nil {
		slog.Info("[DoctorActivate] 获取分布式锁失败，跳过本轮",
			"error", err)
		return
	}
	defer lock.Release()

	log := controller.Logger(ctx)

	log.Info("[DoctorActivate] 开始扫描 creating 会话")

	var creatingSessions []model.DoctorSession
	if dbErr := model.DB(ctx).Where("status = ?",
		model.DoctorStatusCreating).
		Find(&creatingSessions).Error; dbErr != nil {
		log.Error("[DoctorActivate] 查询 creating 会话失败",
			"error", dbErr)
		return
	}

	if len(creatingSessions) == 0 {
		log.Info("[DoctorActivate] 无 creating 会话")
		return
	}

	log.Info("[DoctorActivate] creating 会话数",
		"count", len(creatingSessions))

	for _, s := range creatingSessions {
		sessionLog := log.With("session_id", s.ID)

		// 检查是否超时（10 分钟未 active 则 fail）
		if time.Since(s.CreatedAt) > 10*time.Minute {
			sessionLog.Info(
				"[DoctorActivate] 会话创建超时，标记 failed")
			model.DB(ctx).Model(&s).
				Update("status", model.DoctorStatusFailed)
			continue
		}

		// 委托 controller 层处理完整激活逻辑
		// （CVM 状态查询、TAT Agent 检查、组件安装、Gateway 就绪等）
		if !controller.ActivateDoctorSession(ctx, &s) {
			sessionLog.Info(
				"[DoctorActivate] 本轮未完成激活，等待下次轮询")
		}
	}

	log.Info("[DoctorActivate] 扫描完成")
}
