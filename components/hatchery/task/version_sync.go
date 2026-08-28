package task

import (
	"context"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"hatchery/controller"
	"hatchery/model"
)

const (
	// versionFetchInterval 版本信息过期阈值，超过此时间视为需要重新拉取
	versionFetchInterval = 24 * time.Hour
	// versionSyncConcurrency 定时任务并发拉取上限
	versionSyncConcurrency = 5
)

// versionSyncJitterMax 每个实例拉取前的最大随机抖动时间，用于打散请求避免瞬间并发
var versionSyncJitterMax = 10 * time.Minute

func init() {
	RegisterTask(TaskDef{
		Name:         "version-sync",
		Interval:     24 * time.Hour,
		RunFunc:      runVersionSync,
		NeedDistLock: true,
		PerTenant:    true,
	})
}

// runVersionSync 执行一轮版本同步：查询需要拉取的实例，并发拉取。
func runVersionSync(ctx context.Context) {
	// 查询所有 agent_ready=1 且有 instance_id 的 CVM 实例
	// source='cvm' 守卫：本地 agent 实例不走 TAT 拉取版本路径，
	// 防止未来哪里误将本地实例 agent_ready 设为 1 后误入。
	var instances []model.Instance
	if err := model.DB(ctx).
		Where("agent_ready = 1 AND instance_id != '' AND source = ?", model.InstanceSourceCVM).
		Select("id, instance_id, agent_type, runtime_user, agent_version, version_fetched_at, agent_ready").
		Find(&instances).Error; err != nil {
		slog.Warn("[VersionSync] 查询实例列表失败", "error", err)
		return
	}

	// 筛选需要拉取的实例：版本为空 或 超过 versionFetchInterval 未拉取
	// 无兼容运行时类型的 Agent 跳过（无可用脚本，如未声明 compatible_with 的自定义类型）
	var needFetch []model.Instance
	for _, inst := range instances {
		if model.GetAgentRuntimeType(ctx, inst.AgentType) == "" {
			continue
		}
		if inst.AgentVersion == "" ||
			inst.VersionFetchedAt == nil ||
			time.Since(*inst.VersionFetchedAt) > versionFetchInterval {
			needFetch = append(needFetch, inst)
		}
	}

	if len(needFetch) == 0 {
		return
	}

	slog.Info("[VersionSync] 开始本轮版本同步", "total", len(instances), "need_fetch", len(needFetch))

	sem := make(chan struct{}, versionSyncConcurrency)
	var wg sync.WaitGroup
	for _, inst := range needFetch {
		select {
		case <-ctx.Done():
			slog.Info("[VersionSync] 同步中途收到停止信号，提前退出")
			wg.Wait()
			return
		default:
		}

		wg.Add(1)
		sem <- struct{}{}
		go func(i model.Instance) {
			defer wg.Done()
			defer func() { <-sem }()
			// 随机抖动，打散对 TAT 的并发请求
			if versionSyncJitterMax > 0 {
				jitter := time.Duration(rand.Int63n(int64(versionSyncJitterMax)))
				select {
				case <-time.After(jitter):
				case <-ctx.Done():
					return
				}
			}
			if err := controller.FetchAndSaveVersionInfoSync(ctx, i); err != nil {
				slog.Warn("[VersionSync] 拉取版本失败", "id", i.ID, "instance_id", i.InstanceId, "error", err)
			}
		}(inst)
	}
	wg.Wait()
	slog.Info("[VersionSync] 本轮版本同步完成", "fetched", len(needFetch))
}
