package controller

import (
	"context"
	"fmt"
	"strings"

	hcommon "hatchery/common"
	"hatchery/i18n"
)

const (
	memoryPluginID = "memory-tencentdb"
	memoryNpmPkg   = "@tencentdb-agent-memory/memory-tencentdb"

	resolvePluginRootScript = "resolve_memory_plugin_root.sh"
)

// resolveMemoryPluginRootFn 是可替换的函数变量，方便单元测试 mock。
var resolveMemoryPluginRootFn = resolveMemoryPluginRootImpl

// runScriptForPathFn 是 RunScript 的包装，方便单元测试 mock TAT 调用。
var runScriptForPathFn = RunScript

// ResolveMemoryPluginRoot 通过 TAT 在 CVM 上探测记忆插件的实际安装路径。
//
// 探测优先级（新路径优先）：
//
//  1. ~/.openclaw/npm/projects/tencentdb-agent-memory-memory-tencentdb-<hash>/node_modules/@tencentdb-agent-memory/memory-tencentdb  (OpenClaw 5.28+)
//  2. ~/.openclaw/npm/node_modules/@tencentdb-agent-memory/memory-tencentdb  (OpenClaw 5.2 ~ 5.7)
//  3. ~/.openclaw/extensions/memory-tencentdb                                (OpenClaw ≤ 5.1)
//
// 兜底策略：
//
//	TAT 探测失败（网络抖动、agent 短暂离线等）时，fallback 到旧路径，
//	避免因瞬时网络问题阻断业务流程。即使 fallback 路径不存在，后续脚本
//	执行时也会因 cd 失败或文件不存在而报明确错误，不会静默出错。
func ResolveMemoryPluginRoot(ctx context.Context, instanceID string) (string, error) {
	return resolveMemoryPluginRootFn(ctx, instanceID)
}

func resolveMemoryPluginRootImpl(ctx context.Context, instanceID string) (string, error) {
	log := Logger(ctx)

	output, err := runScriptForPathFn(ctx, instanceID, resolvePluginRootScript, 60, "", nil, map[string]string{
		"memory_plugin_id": memoryPluginID,
		"memory_npm_pkg":   memoryNpmPkg,
	})
	if err != nil {
		// 兜底：探测失败时 fallback 到旧路径，避免因网络抖动阻断业务
		fallback := fallbackMemoryPluginRoot(ctx, instanceID)
		log.Warn("[ResolveMemoryPluginRoot] TAT 探测失败，fallback 到旧路径",
			"instance_id", instanceID, "fallback", fallback, "error", err)
		return fallback, nil
	}
	root := strings.TrimSpace(output)
	if root == "" {
		return "", hcommon.I18nError(i18n.MsgMemoryPluginNotFoundOnCVM)
	}
	log.Info("[ResolveMemoryPluginRoot] 探测成功",
		"instance_id", instanceID, "plugin_root", root)
	return root, nil
}

// fallbackMemoryPluginRoot 探测失败时的兜底路径：使用 $HOME 形式避免 WAF 拦截。
// 存量机器绝大多数还在旧路径，fallback 大概率是对的。
func fallbackMemoryPluginRoot(ctx context.Context, instanceID string) string {
	return fmt.Sprintf("$HOME/.openclaw/extensions/%s", memoryPluginID)
}
