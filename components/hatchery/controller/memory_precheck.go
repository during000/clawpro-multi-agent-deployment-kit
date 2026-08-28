package controller

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// MemoryPrecheckResult 单个实例的网络预检结果。
type MemoryPrecheckResult struct {
	InstanceID    string
	Reachable     bool
	Skipped       bool   // true 表示因条件不满足跳过了预检（视为通过）
	SkipReason    string // 跳过原因（内部日志用）
	Reason        string // network_unreachable / precheck_timeout / precheck_skipped
	Message       string // 面向用户的错误文案
	VDBInstanceID string // VDB 实例 ID（如 vdb-p6cytw9z）
}

// precheckRunScriptFn 可替换的 RunScript 函数，方便单测 mock。
var precheckRunScriptFn = func(ctx context.Context, instanceID, scriptName string, timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
	output, rerr := agentScriptRunner(ctx, instanceID, scriptName, timeout, runtimeUser, onOutput, params)
	if rerr != nil {
		return output, rerr
	}
	return output, nil
}

// PrecheckBatchForProSwitch 并行预检一组实例的 CVM→VDB 网络连通性。
// 由调用方在 target_plan=pro 时触发。
// 返回 map[instanceID]PrecheckResult。不在 map 中的实例 = 预检通过。
func PrecheckBatchForProSwitch(ctx context.Context, instanceIDs []string) map[string]MemoryPrecheckResult {
	log := Logger(ctx)
	results := make(map[string]MemoryPrecheckResult, len(instanceIDs))
	if len(instanceIDs) == 0 {
		return results
	}

	// 1. 获取 VDB 池级信息（endpoint + test 凭证 + vdb_instance_id）
	vdbEndpoint, vdbAccount, vdbPassword, vdbInstanceID, err := getVDBPoolPrecheckTarget(ctx)
	if err != nil || vdbEndpoint == "" {
		log.Warn("[MemoryPrecheck] 无法获取 VDB 池级 endpoint，跳过所有预检",
			"error", err, "instance_count", len(instanceIDs))
		for _, id := range instanceIDs {
			results[id] = MemoryPrecheckResult{
				InstanceID: id,
				Reachable:  true,
				Skipped:    true,
				SkipReason: "vdb_pool_endpoint_unavailable",
			}
		}
		return results
	}

	log.Info("[MemoryPrecheck] 开始批量预检",
		"instance_count", len(instanceIDs),
		"vdb_endpoint", vdbEndpoint,
		"vdb_instance_id", vdbInstanceID,
	)

	// 2. 并发探测（限制并发数，防 TAT 被打爆）
	const maxConcurrency = 10
	sem := make(chan struct{}, maxConcurrency)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, instID := range instanceIDs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					log.Error("[MemoryPrecheck] probeSingleInstance panic recovered",
						"instance_id", id, "panic", r)
					mu.Lock()
					results[id] = MemoryPrecheckResult{
						InstanceID: id,
						Reachable:  true,
						Skipped:    true,
						SkipReason: fmt.Sprintf("panic: %v", r),
					}
					mu.Unlock()
				}
			}()
			sem <- struct{}{}
			defer func() { <-sem }()

			pr := probeSingleInstance(ctx, id, vdbEndpoint, vdbAccount, vdbPassword, vdbInstanceID)
			mu.Lock()
			results[id] = pr
			mu.Unlock()
		}(instID)
	}
	wg.Wait()

	return results
}

// probeSingleInstance 在指定 CVM 上探测 VDB 连通性。
func probeSingleInstance(ctx context.Context, instanceID, vdbEndpoint, vdbAccount, vdbPassword, vdbInstanceID string) MemoryPrecheckResult {
	log := Logger(ctx)
	runtimeUser := LookupRuntimeUser(ctx, instanceID)

	output, err := precheckRunScriptFn(ctx, instanceID, "precheck_vdb_connectivity.sh", 30, runtimeUser, nil,
		map[string]string{
			"vdb_endpoint": vdbEndpoint,
			"vdb_username": vdbAccount,
			"vdb_api_key":  vdbPassword,
			"vdb_database": "",
			"timeout_sec":  "5",
		})

	// 合并 stdout：TAT 成功 → output 有值；TAT 失败（脚本退出码非 0）→ stdout 在 RichError.Detail
	stdout := output
	if stdout == "" {
		var re *hcommon.RichError
		if errors.As(err, &re) {
			stdout = hcommon.ErrorDetailWithCtx(ctx, re)
		}
	}

	// 判定结果
	if strings.Contains(stdout, `"reachable":true`) {
		log.Info("[MemoryPrecheck] CVM 到 VDB 网络连通", "instance_id", instanceID)
		return MemoryPrecheckResult{
			InstanceID:    instanceID,
			Reachable:     true,
			VDBInstanceID: vdbInstanceID,
		}
	}

	if strings.Contains(stdout, `"reachable":false`) {
		// 工具不可用 → 保守视为通过
		if strings.Contains(stdout, `"probe":"none"`) || strings.Contains(stdout, `"no_probe_tool"`) {
			log.Warn("[MemoryPrecheck] CVM 无探测工具，保守视为通过", "instance_id", instanceID)
			return MemoryPrecheckResult{
				InstanceID: instanceID,
				Reachable:  true,
				Skipped:    true,
				SkipReason: "no_probe_tool",
			}
		}
		// 明确不通
		log.Warn("[MemoryPrecheck] CVM 到 VDB 网络不通",
			"instance_id", instanceID, "vdb_endpoint", vdbEndpoint, "output", stdout)
		return MemoryPrecheckResult{
			InstanceID:    instanceID,
			Reachable:     false,
			Reason:        "network_unreachable",
			VDBInstanceID: vdbInstanceID,
			Message: fmt.Sprintf(
				"Agent所在CVM (%s) 到 记忆空间所在VDB (%s, %s) 网络不通，无法切换到 Pro 版。请检查 CVM 与 VDB 所在 VPC 是否连通，以及 CVM 与 VDB 的安全组规则是否放通。",
				instanceID, vdbInstanceID, vdbEndpoint),
		}
	}

	// TAT 调用异常 / 脚本输出无法解析 → 视为超时/异常，保守跳过
	if err != nil {
		log.Warn("[MemoryPrecheck] 预检异常，保守视为通过",
			"instance_id", instanceID, "error", err, "output", stdout)
		return MemoryPrecheckResult{
			InstanceID: instanceID,
			Reachable:  true,
			Skipped:    true,
			SkipReason: fmt.Sprintf("probe_error: %v", err),
		}
	}

	// 未知情况，保守通过
	return MemoryPrecheckResult{
		InstanceID: instanceID,
		Reachable:  true,
		Skipped:    true,
		SkipReason: "unknown_probe_result",
	}
}

// getVDBPoolPrecheckTarget 从 TDAI agent memory API 获取 VDB 池级的 probe 信息。
// 返回：endpoint（如 http://10.0.0.18:80）、account、password、vdbInstanceId、error
// 策略：优先 SDK（池级 VDBVip/TestAccount）→ fallback 到 DB 已有 PRO 实例的凭证。
func getVDBPoolPrecheckTarget(ctx context.Context) (endpoint, account, password, vdbInstanceID string, err error) {
	log := Logger(ctx)

	client, sdkErr := NewMemorySDKClient(ctx)
	if sdkErr != nil {
		log.Warn("[MemoryPrecheck] SDK client 创建失败，走 fallback", "error", sdkErr)
	} else {
		resp, apiErr := client.DescribeMemoryProInstances(ctx, nil)
		if apiErr != nil {
			log.Warn("[MemoryPrecheck] DescribeMemoryProInstances 调用失败，走 fallback", "error", apiErr)
		} else if resp.TotalCount == 0 || len(resp.Items) == 0 {
			log.Warn("[MemoryPrecheck] DescribeMemoryProInstances 返回 0 个实例，走 fallback")
		} else {
			// 打印第一个实例的关键字段，便于排查
			first := resp.Items[0]
			log.Info("[MemoryPrecheck] DescribeMemoryProInstances 返回",
				"total_count", resp.TotalCount,
				"first_memory_pro_id", first.MemoryProId,
				"first_vdb_instance_id", first.VDBInstanceId,
				"first_vdb_vip", first.VDBVip,
				"first_vdb_port", first.VDBPort,
				"first_test_account", first.TestAccount,
				"first_test_password_len", len(first.TestPassword),
				"first_status", first.Status,
				"first_vdb_status", first.VDBStatus,
			)
			for i := range resp.Items {
				item := &resp.Items[i]
				if !item.IsRunning() {
					log.Info("[MemoryPrecheck] 跳过非 running 实例",
						"memory_pro_id", item.MemoryProId, "status", item.Status, "vdb_status", item.VDBStatus)
					continue
				}
				ep := item.VDBEndpoint()
				if ep == "" {
					log.Warn("[MemoryPrecheck] 实例 VDBVip/VDBPort 为空，跳过",
						"memory_pro_id", item.MemoryProId, "vdb_vip", item.VDBVip, "vdb_port", item.VDBPort)
					continue
				}
				if item.TestAccount == "" || item.TestPassword == "" {
					log.Warn("[MemoryPrecheck] 实例 TestAccount/TestPassword 为空，跳过",
						"memory_pro_id", item.MemoryProId, "test_account", item.TestAccount)
					continue
				}
				log.Info("[MemoryPrecheck] 使用 SDK 一级路径",
					"endpoint", ep, "vdb_instance_id", item.VDBInstanceId, "test_account", item.TestAccount)
				return ep, item.TestAccount, item.TestPassword, item.VDBInstanceId, nil
			}
			log.Warn("[MemoryPrecheck] SDK 返回的所有实例均不满足条件，走 fallback")
		}
	}

	// SDK 不可用 或 返回的实例不满足条件 → fallback 到 DB
	log.Info("[MemoryPrecheck] 进入 fallback 路径（从已有 PRO 实例借用凭证）")
	return getVDBPoolPrecheckTargetFallback(ctx)
}

// getVDBPoolPrecheckTargetFallback 从已有 PRO 实例的 plugin 记录中借用 endpoint/凭证。
func getVDBPoolPrecheckTargetFallback(ctx context.Context) (endpoint, account, password, vdbInstanceID string, err error) {
	var plugin model.MemoryTDAIPlugin
	if err := model.DB(ctx).Where("current_plan = ? AND endpoint != '' AND api_key_secret_ref != ''",
		model.MemoryPlanPro).First(&plugin).Error; err != nil {
		return "", "", "", "", hcommon.I18nRichError(err, i18n.MsgMemoryPrecheckNoFallbackCred)
	}

	// 获取 VDB 实例 ID（vdb-xxx）：优先从 SDK 拿（即使 TestAccount 为空，VDBInstanceId 通常有值）
	vdbID := ""
	client, sdkErr := NewMemorySDKClient(ctx)
	if sdkErr == nil {
		resp, apiErr := client.DescribeMemoryProInstances(ctx, nil)
		if apiErr == nil && len(resp.Items) > 0 {
			vdbID = resp.Items[0].VDBInstanceId
		}
	}
	// 兜底：SDK 也拿不到时用 pool_id（space-xxx，不理想但总比空好）
	if vdbID == "" {
		vdbID = plugin.PoolID
		if vdbID == "" {
			vdbID = "unknown"
		}
	}

	return plugin.Endpoint, plugin.VdbUsername, plugin.ApiKeySecretRef, vdbID, nil
}
