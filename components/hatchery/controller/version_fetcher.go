package controller

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// versionInfoResult get_version_info.sh 脚本输出的 JSON 结构
type versionInfoResult struct {
	AgentVersion string            `json:"agent_version"`
	AgentType    string            `json:"agent_type"`
	Plugins      map[string]string `json:"plugins"`
}

// runScriptFn 是 RunScript 的可替换包装，方便单元测试 mock
var runScriptFn = RunScript

// versionFetchInFlight 防止同一实例被并发拉取（定时任务 + 手动刷新可能同时触发）
var versionFetchInFlight sync.Map // key: uint (instance DB ID)

// FetchAndSaveVersionInfoSync 同步拉取版本信息，返回错误（供重试接口调用）。
// 内部使用 30s 超时的 TAT 脚本执行。
func FetchAndSaveVersionInfoSync(ctx context.Context, inst model.Instance) error {
	return doFetchAndSaveVersionInfo(ctx, inst)
}

func parseDetectedAgentTypeToken(output string) (string, error) {
	lines := strings.Split(output, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		token := strings.ToLower(strings.TrimSpace(lines[i]))
		if token == "" {
			continue
		}
		switch token {
		case model.AgentTypeOpenClaw:
			return model.AgentTypeOpenClaw, nil
		case model.AgentTypeHermes:
			return model.AgentTypeHermes, nil
		case model.AgentTypeLightclawACE, "lightclaw-ace", "ace":
			return model.AgentTypeLightclawACE, nil
		case "unknown":
			return "unknown", nil
		default:
			return "", hcommon.I18nError(i18n.MsgVersionFetcherInvalidDetectToken, token)
		}
	}
	return "", hcommon.I18nError(i18n.MsgVersionFetcherEmptyDetectOutput)
}

// detectAndRepairAgentType 在 get_version_info 前做一次类型探测。
// 若探测值与 DB 不一致，则立即修正 instances.agent_type（用于存量脏数据自愈）。
func detectAndRepairAgentType(ctx context.Context, inst *model.Instance) string {
	effective := strings.TrimSpace(inst.AgentType)
	if effective == "" {
		effective = model.AgentTypeOpenClaw
	}
	if model.IsCustomAgentType(ctx, effective) {
		return effective
	}

	output, err := runScriptFn(ctx, inst.InstanceId, "detect_agent_type.sh", 30, "", nil, nil)
	if err != nil {
		slog.Warn("[VersionFetcher] detect_agent_type 执行失败，沿用 DB agent_type",
			"id", inst.ID, "instance_id", inst.InstanceId, "db_agent_type", effective, "error", err)
		return effective
	}

	detected, rerr := parseDetectedAgentTypeToken(output)
	if rerr != nil {
		slog.Warn("[VersionFetcher] detect_agent_type 输出非法，沿用 DB agent_type",
			"id", inst.ID, "instance_id", inst.InstanceId, "db_agent_type", effective, "error", rerr, "output", output)
		return effective
	}
	if detected == "unknown" || detected == "" {
		slog.Warn("[VersionFetcher] detect_agent_type 未识别到类型，沿用 DB agent_type",
			"id", inst.ID, "instance_id", inst.InstanceId, "db_agent_type", effective)
		return effective
	}
	if detected == effective {
		return effective
	}

	if err := model.DB(ctx).Model(&model.Instance{}).Where("id = ?", inst.ID).Update("agent_type", detected).Error; err != nil {
		slog.Warn("[VersionFetcher] agent_type 脏数据修正写库失败，沿用 DB agent_type",
			"id", inst.ID, "instance_id", inst.InstanceId, "db_agent_type", effective, "detected_agent_type", detected, "error", err)
		return effective
	}

	slog.Info("[VersionFetcher] 纠正实例 agent_type",
		"id", inst.ID, "instance_id", inst.InstanceId, "from", effective, "to", detected)
	inst.AgentType = detected
	return detected
}

// doFetchAndSaveVersionInfo 核心实现：TAT 执行脚本 → 解析 → 写 DB。
// 内置去重保护：同一实例不会被并发拉取。
func doFetchAndSaveVersionInfo(ctx context.Context, inst model.Instance) error {
	if inst.InstanceId == "" {
		return nil
	}
	if model.GetAgentRuntimeType(ctx, inst.AgentType) == "" {
		slog.Info("[VersionFetcher] 无兼容运行时类型，跳过版本同步", "id", inst.ID, "instance_id", inst.InstanceId, "agent_type", inst.AgentType)
		return nil
	}

	// 防止同一实例被并发拉取（定时任务 + 手动刷新可能同时触发）
	if _, loaded := versionFetchInFlight.LoadOrStore(inst.ID, true); loaded {
		slog.Debug("[VersionFetcher] 实例正在拉取中，跳过", "id", inst.ID)
		return nil
	}
	defer versionFetchInFlight.Delete(inst.ID)

	slog.Info("[VersionFetcher] 开始拉取版本信息", "id", inst.ID, "instance_id", inst.InstanceId, "agent_type", inst.AgentType)

	// 临时自愈：每次拉取版本前先探测真实类型并修正 DB 脏数据。
	effectiveAgentType := detectAndRepairAgentType(ctx, &inst)

	// final：按 agent_type 分派脚本，避免所有类型都跑 openclaw 专属脚本
	// 拿到空版本 / 错 agent_type 等问题。
	// 若该 agent_type 未配置脚本 → fail-closed 跳过版本同步（不视为硬错误，避免
	// 定时任务全量失败），由后续新增脚本的 merge 回填。
	scriptName, resolveErr := ResolveScript(ctx, "get_version_info", effectiveAgentType)
	if resolveErr != nil {
		slog.Warn("[VersionFetcher] 该 agent_type 未配置版本脚本，跳过同步",
			"id", inst.ID, "agent_type", effectiveAgentType, "error", resolveErr)
		return nil
	}

	output, err := runScriptFn(ctx, inst.InstanceId, scriptName, 30, inst.RuntimeUser, nil, nil)
	if err != nil {
		slog.Warn("[VersionFetcher] 执行脚本失败", "id", inst.ID, "script", scriptName, "error", err)
		return err
	}

	// 从输出中提取 JSON 行（取最后一个以 { 开头的行，因为脚本 JSON 输出在最后）
	jsonLine := ""
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "{") {
			jsonLine = line
		}
	}
	if jsonLine == "" {
		slog.Warn("[VersionFetcher] 脚本输出中未找到 JSON", "id", inst.ID, "output", output)
		return nil
	}

	var result versionInfoResult
	if err := json.Unmarshal([]byte(jsonLine), &result); err != nil {
		slog.Warn("[VersionFetcher] 解析版本 JSON 失败", "id", inst.ID, "error", err, "json", jsonLine)
		return hcommon.I18nRichError(err, i18n.MsgVersionFetcherParseJSONFailed)
	}

	// 序列化插件版本为 JSON 字符串存入 DB
	pluginsJSON := "{}"
	if len(result.Plugins) > 0 {
		if b, err := json.Marshal(result.Plugins); err == nil {
			pluginsJSON = string(b)
		}
	}

	now := time.Now()
	// 【修复】不再写回 agent_type：
	//   1. agent_type 是实例创建时确定的不变量，只应由 CreateInstance 写入；
	//   2. get_version_info.sh 目前硬编码返回 "openclaw"，会把 hermes/ace 实例错误覆盖成 openclaw；
	//   3. 如果脚本返回空 agent_type，map 里的 "" 也会覆盖掉 DB 里的正确值 —— 同样是 bug；
	// 本函数只负责同步 agent_version 和插件版本信息。
	//
	// 对 agent_type 的一致性校验：若脚本返回的类型与 DB 已记录的类型不一致，
	// 打 warning 但不覆盖，提示运维排查。
	if result.AgentType != "" && result.AgentType != effectiveAgentType {
		slog.Warn("[VersionFetcher] 脚本返回的 agent_type 与实例记录不一致，忽略（以 DB 记录为准）",
			"id", inst.ID,
			"instance_id", inst.InstanceId,
			"db_agent_type", effectiveAgentType,
			"script_agent_type", result.AgentType)
	}
	updates := map[string]interface{}{
		"agent_version":        result.AgentVersion,
		"plugin_versions_json": pluginsJSON,
		"version_fetched_at":   &now,
	}

	if err := model.DB(ctx).Model(&inst).Updates(updates).Error; err != nil {
		slog.Error("[VersionFetcher] 写入版本信息失败", "id", inst.ID, "error", err)
		return hcommon.I18nRichError(err, i18n.MsgVersionFetcherWriteDBFailed)
	}

	slog.Info("[VersionFetcher] 版本信息已更新",
		"id", inst.ID,
		"agent_version", result.AgentVersion,
		"plugin_count", len(result.Plugins),
	)
	return nil
}
