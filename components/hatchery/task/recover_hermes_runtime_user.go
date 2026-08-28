package task

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	hcommon "hatchery/common"
	"hatchery/controller"
	"hatchery/model"
)

// recoverResolveScriptFn / recoverRunScriptFn 是 controller.ResolveScript / RunScript 的可替换包装，
// 方便单元测试 mock TAT 调用。
var (
	recoverResolveScriptFn = controller.ResolveScript
	recoverRunScriptFn     = controller.RunScript
)

const (
	// recoverHermesRuntimeUserConcurrency 并发探测上限
	recoverHermesRuntimeUserConcurrency = 5
)

// recoverResult 单实例恢复结果
type recoverResult int

const (
	recoverSkipped recoverResult = iota // 数据一致，无需更新
	recoverFixed                        // 数据不一致，已修复
	recoverFailed                       // 探测或更新失败
)

func init() {
	RegisterTask(TaskDef{
		Name:         "recover-hermes-runtime-user",
		Interval:     0, // 一次性：仅在服务启动后执行一次，修复存量脏数据
		RunFunc:      runRecoverHermesRuntimeUser,
		NeedDistLock: true,
		PerTenant:    true,
		InitialDelay: 2 * time.Minute,
	})
}

// runRecoverHermesRuntimeUser 对所有 running 且 agent_ready=1 的 hermes 实例重新探测 runtime_user，
// 修复因 .hermes 目录被脚本日志初始化提前创建导致的脏数据（误判为已安装 hermes）。
func runRecoverHermesRuntimeUser(ctx context.Context) {
	var instances []model.Instance
	if err := model.DB(ctx).
		Where("agent_ready = 1 AND instance_id != '' AND agent_type = ? AND last_stable_state = 'RUNNING'", model.AgentTypeHermes).
		Select("id, instance_id, agent_type, runtime_user, runtime_home").
		Find(&instances).Error; err != nil {
		slog.Warn("[RecoverHermesRuntimeUser] 查询实例列表失败", "error", err)
		return
	}

	if len(instances) == 0 {
		slog.Info("[RecoverHermesRuntimeUser] 无符合条件的 hermes 实例，跳过")
		return
	}

	slog.Info("[RecoverHermesRuntimeUser] 开始本轮探测", "total", len(instances))

	sem := make(chan struct{}, recoverHermesRuntimeUserConcurrency)
	var wg sync.WaitGroup
	var fixedCount, failedCount int
	var mu sync.Mutex

	for _, inst := range instances {
		select {
		case <-ctx.Done():
			slog.Info("[RecoverHermesRuntimeUser] 收到停止信号，提前退出")
			wg.Wait()
			return
		default:
		}

		wg.Add(1)
		sem <- struct{}{}
		go func(i model.Instance) {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					slog.Error("[RecoverHermesRuntimeUser] panic recovered", "instance_id", i.InstanceId, "error", r)
				}
			}()

			result := recoverOneInstance(ctx, i)
			mu.Lock()
			switch result {
			case recoverFixed:
				fixedCount++
			case recoverFailed:
				failedCount++
			}
			mu.Unlock()
		}(inst)
	}
	wg.Wait()

	slog.Info("[RecoverHermesRuntimeUser] 本轮探测完成",
		"total", len(instances), "fixed", fixedCount, "failed", failedCount,
		"skipped", len(instances)-fixedCount-failedCount)
}

// recoverOneInstance 对单个实例执行探测并对比，返回恢复结果。
func recoverOneInstance(ctx context.Context, inst model.Instance) recoverResult {
	scriptName, rerr := recoverResolveScriptFn(ctx, "detect_install", inst.AgentType)
	if rerr != nil {
		slog.Warn("[RecoverHermesRuntimeUser] 解析探测脚本失败", "instance_id", inst.InstanceId, "error", rerr)
		return recoverFailed
	}

	output, err := recoverRunScriptFn(hcommon.DetachContext(ctx), inst.InstanceId, scriptName, 30, "", nil, nil)
	if err != nil {
		slog.Warn("[RecoverHermesRuntimeUser] 探测脚本执行失败", "instance_id", inst.InstanceId, "error", err)
		return recoverFailed
	}

	var result struct {
		RuntimeUser string `json:"runtime_user"`
		RuntimeHome string `json:"runtime_home"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		slog.Warn("[RecoverHermesRuntimeUser] 解析探测结果失败", "instance_id", inst.InstanceId, "output", output, "error", err)
		return recoverFailed
	}

	if result.RuntimeUser == "" || result.RuntimeUser == "unknown" {
		slog.Warn("[RecoverHermesRuntimeUser] 探测结果无效，跳过（不覆盖已有数据）", "instance_id", inst.InstanceId, "db_user", inst.RuntimeUser)
		return recoverFailed
	}

	// 保护：探测成功但 DB 已有非空值时，仅当探测结果确实不同才更新；
	// 若探测返回的是空值或 unknown，上面已拦截，不会走到这里覆盖。
	if result.RuntimeUser == inst.RuntimeUser && result.RuntimeHome == inst.RuntimeHome {
		return recoverSkipped
	}

	slog.Info("[RecoverHermesRuntimeUser] 发现不一致，更新数据",
		"instance_id", inst.InstanceId,
		"old_user", inst.RuntimeUser, "new_user", result.RuntimeUser,
		"old_home", inst.RuntimeHome, "new_home", result.RuntimeHome)

	if err := model.DB(ctx).Model(&model.Instance{}).
		Where("id = ?", inst.ID).
		Updates(map[string]interface{}{
			"runtime_user": result.RuntimeUser,
			"runtime_home": result.RuntimeHome,
		}).Error; err != nil {
		slog.Warn("[RecoverHermesRuntimeUser] 更新数据库失败", "instance_id", inst.InstanceId, "error", err)
		return recoverFailed
	}

	return recoverFixed
}
