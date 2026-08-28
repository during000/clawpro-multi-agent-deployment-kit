package controller

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"sort"
	"time"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tat "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tat/v20201028"

	hcommon "hatchery/common"
	"hatchery/i18n"
)

// ============================================================================
// 常量
// ============================================================================
//
// 详见腾讯云 TAT 文档：
// - RunCommand：https://cloud.tencent.com/document/api/1340/52676
//   单批 InstanceIds 上限：200
// - DescribeInvocationTasks：https://cloud.tencent.com/document/api/1340/52680
//   单批 InvocationTaskIds 上限：100
// - Parameters JSON 字符串长度上限：1024 字节（保留缓冲）

const (
	// TATRunCommandBatchMax 单次 TAT RunCommand 最多支持的 instance 数量。
	// 与官方文档对齐（200）；后续随官方调整。
	TATRunCommandBatchMax = 200

	// TATDescribeInvocationTasksBatchMax 单次 DescribeInvocationTasks 最多查询的 InvocationTaskId 数量。
	// 同时也是单页 Limit 的硬上限（按 invocation-id Filter 查询时分页用 Offset）。
	TATDescribeInvocationTasksBatchMax = 100

	// tatBatchRunCommandTaskBindingRetryMax RunCommand 返回 InvocationId 后，立刻去拉
	// 对应的 InvocationTaskId 可能有秒级延迟；轮询次数 + 间隔。
	tatBatchRunCommandTaskBindingRetryMax = 5
	tatBatchRunCommandTaskBindingInterval = 500 * time.Millisecond
)

// ============================================================================
// 类型
// ============================================================================

// InvocationTaskBinding 把 TAT 一次 RunCommand 内每台 instance 与其 InvocationTaskId 一一对齐。
type InvocationTaskBinding struct {
	InstanceID       string // CVM ins-xxx
	InvocationTaskID string // TAT invt-xxx
}

// InvocationTaskDetail 表示批量查询单条 InvocationTask 的细节。
//
// Output 已 base64 解码为 raw 字符串；超长由调用方按业务需要自行截断。
// Finished 表示该 task 是否到达 TAT 终态（SUCCESS / FAILED / TIMEOUT 等）。
// ErrorInfo 来自 TAT InvocationTask 顶层字段，DELIVER_FAILED / START_FAILED 等
// 启动阶段失败时承载具体原因（如 "user xxx does not exist"），SUCCESS / FAILED
// 等正常执行后到达终态的场景下通常为空。
type InvocationTaskDetail struct {
	InvocationTaskID string
	InstanceID       string
	TaskStatus       string // TAT 原始状态字符串：PENDING / DELIVERING / RUNNING / SUCCESS / FAILED / TIMEOUT / DELIVER_FAILED / START_FAILED ...
	Stdout           string // base64 已解码
	Stderr           string // 当前 TAT API 不分离 stderr，保留字段以备将来；当前永远为空
	ErrorInfo        string // 启动失败时 TAT 返回的错误描述（DELIVER_FAILED / START_FAILED 时非空）
	ExitCode         *int64 // 终态后有值
	StartTime        *time.Time
	EndTime          *time.Time
	Finished         bool
	NotFound         bool // 当前批次未在 TAT 端找到该 task（已过期 / 越权）
}

// ============================================================================
// 错误
// ============================================================================

var (
	// ErrTATBatchTooMany 调用方传入的 instance / invocationTaskId 数量超过 TAT 单批上限。
	// RunInlineCommandBatchAsync 的调用方应负责按 TATRunCommandBatchMax 拆批；
	// DescribeInvocationTasksBatch 内部已经自动拆批，不会触发此错误。
	ErrTATBatchTooMany = hcommon.I18nError(i18n.MsgTATBatchTooMany)
)

// ============================================================================
// 接口（用于测试 mock）
// ============================================================================

// tatBatchClient 抽象 TAT SDK 我们用到的两个方法，便于 mock。
type tatBatchClient interface {
	RunCommand(req *tat.RunCommandRequest) (*tat.RunCommandResponse, error)
	DescribeInvocationTasks(req *tat.DescribeInvocationTasksRequest) (*tat.DescribeInvocationTasksResponse, error)
}

// tatBatchClientFactory 默认走 NewTATClient；测试可替换。
var tatBatchClientFactory = func(ctx context.Context) (tatBatchClient, error) {
	return NewTATClient(ctx)
}

// ============================================================================
// RunInlineCommandBatchAsync — 批量异步下发 inline 脚本
// ============================================================================

// RunInlineCommandBatchAsync 通过腾讯云 TAT 一次性把 inline 脚本下发到一批 instance，
// 不轮询结果，立即返回 (invocationId, bindings)。
//
// 重要约定（与 RunInlineScript / RunScript 行为有意分叉，详见
// openspec/changes/agent-command-execution/design.md §8.1–§8.2）：
//
//  1. 入参 scriptContent 为 raw 文本（用户在命令模板编辑器写的什么就是什么）。
//     函数内部完成 base64 编码，**不向调用方传递已编码字符串的责任**。
//  2. **不注入 tatRuntimePrelude**：命令模板是面向运维管理员的产品功能，
//     "用户写啥就跑啥"；详情页展示的内容 == TAT 实际执行的内容（仅 base64 wire 差异）。
//
// instanceIds 上限 = TATRunCommandBatchMax（200 台）；超过返回 ErrTATBatchTooMany。
// 由调用方负责按上限拆批，每批产生独立的 invocation 记录。
//
// params 对应 TAT RunCommand.Parameters：JSON {name: value} 序列化后下发，
// TAT 服务端做 {{name}} 替换，本函数不在 scriptContent 上做字符串替换。
func RunInlineCommandBatchAsync(
	ctx context.Context,
	instanceIds []string,
	scriptContent string,
	timeout uint64,
	runUser string,
	workdir string,
	params map[string]string,
) (string, []InvocationTaskBinding, error) {
	if len(instanceIds) == 0 {
		return "", nil, hcommon.I18nError(i18n.MsgTATInstanceIdsEmpty)
	}
	if len(instanceIds) > TATRunCommandBatchMax {
		return "", nil, hcommon.I18nRichError(ErrTATBatchTooMany, i18n.MsgTATBatchTooManyDetail, len(instanceIds), TATRunCommandBatchMax)
	}

	client, err := tatBatchClientFactory(ctx)
	if err != nil {
		return "", nil, hcommon.I18nError(i18n.MsgTATCommandDispatchFailed).WithI18nDetail(i18n.MsgTATClientCreateFailed, err)
	}

	req := tat.NewRunCommandRequest()
	// ⚠️ 决策 §8.1: scriptContent 为 raw，函数内 encode；不注入 prelude（决策 §8.2）
	req.Content = common.StringPtr(base64.StdEncoding.EncodeToString([]byte(scriptContent)))
	req.InstanceIds = common.StringPtrs(instanceIds)
	req.CommandName = common.StringPtr("agent_command_dispatch")
	req.CommandType = common.StringPtr("SHELL")
	if effective := runUserOrDefault(runUser); effective != "" {
		req.Username = common.StringPtr(effective)
	}
	if effective := workdirOrDefault(workdir); effective != "" {
		req.WorkingDirectory = common.StringPtr(effective)
	}
	req.Timeout = common.Uint64Ptr(timeout)
	req.SaveCommand = common.BoolPtr(false)

	if len(params) > 0 {
		req.EnableParameter = common.BoolPtr(true)
		paramsJSON, jsonErr := json.Marshal(params)
		if jsonErr != nil {
			return "", nil, hcommon.I18nError(i18n.MsgTATCommandDispatchFailed).WithI18nDetail(i18n.MsgTATSerializeParamsFailed, jsonErr)
		}
		req.Parameters = common.StringPtr(string(paramsJSON))
	}

	slog.Info("[TAT] RunInlineCommandBatchAsync",
		"instances", len(instanceIds), "timeout", timeout,
		"run_user", runUser, "workdir", workdir, "params", len(params))
	resp, err := client.RunCommand(req)
	if err != nil {
		slog.Error("[TAT] RunCommand 失败", "instances", len(instanceIds), "error", err)
		return "", nil, hcommon.I18nRichError(err, i18n.MsgTATCommandDispatchFailed)
	}
	if resp.Response == nil || resp.Response.InvocationId == nil ||
		*resp.Response.InvocationId == "" {
		return "", nil, hcommon.I18nError(i18n.MsgTATCommandDispatchFailed).WithI18nDetail(i18n.MsgTATNoInvocationIdReturned)
	}
	invocationID := *resp.Response.InvocationId

	bindings, rerr := fetchInvocationTaskBindings(client, invocationID, len(instanceIds))
	if rerr != nil {
		// RunCommand 已成功；获取 binding 失败是后续轮询能弥补的次要问题，
		// 仍把 invocationId 返回给上层，bindings 为空。上层可走 DescribeInvocationTasksBatch 重新拉。
		slog.Warn("[TAT] RunCommand 返回 InvocationId，但拉取 InvocationTaskBinding 失败",
			"invocation_id", invocationID, "error", hcommon.ErrorMessageWithCtx(ctx, rerr))
		return invocationID, nil, nil
	}
	return invocationID, bindings, nil
}

// runUserOrDefault 把空字符串归一化成 root；其它原样返回。
// 如果上层希望 TAT 用其默认（系统决定），传 ""，本函数返回 ""，调用方据此跳过设置 Username。
func runUserOrDefault(s string) string {
	if s == "" {
		return "root"
	}
	return s
}

// workdirOrDefault 同上语义，默认 /root。
func workdirOrDefault(s string) string {
	if s == "" {
		return "/root"
	}
	return s
}

// fetchInvocationTaskBindings 通过 DescribeInvocationTasks 按 invocation-id 反查每个 instance 对应的 InvocationTaskId。
//
// TAT 在 RunCommand 同步返回 InvocationId 后，InvocationTask 行可能有亚秒级延迟才可见，故带短轮询。
//
// expectedCount 是调用方预期的 binding 数（即 RunCommand 时传入的 instanceIds 数量）。
//   - >0：每轮按 Offset 分页拉直到累计 >= expectedCount 或返回页空（可能服务端尚未生成完）；
//     不足时进入下一轮短重试，最终仍不足则返回部分结果（含错误，由上层降级路径处理）。
//   - =0：兼容老调用，仅按单页 Limit=TATDescribeInvocationTasksBatchMax 拉取。
//
// TAT DescribeInvocationTasks 单页 Limit 硬上限为 TATDescribeInvocationTasksBatchMax (100)，
// 200 台 dispatch 必须分页。
func fetchInvocationTaskBindings(client tatBatchClient, invocationID string, expectedCount int) ([]InvocationTaskBinding, error) {
	pageLimit := uint64(TATDescribeInvocationTasksBatchMax)

	collectPage := func(offset uint64) ([]InvocationTaskBinding, int, error) {
		descReq := tat.NewDescribeInvocationTasksRequest()
		descReq.Filters = []*tat.Filter{
			{
				Name:   common.StringPtr("invocation-id"),
				Values: common.StringPtrs([]string{invocationID}),
			},
		}
		descReq.HideOutput = common.BoolPtr(true) // 此处只要 binding，不需要 output
		descReq.Limit = common.Uint64Ptr(pageLimit)
		descReq.Offset = common.Uint64Ptr(offset)

		resp, err := client.DescribeInvocationTasks(descReq)
		if err != nil {
			return nil, 0, hcommon.I18nRichError(err, i18n.MsgTATDescribeBindingFailed)
		}
		if resp.Response == nil {
			return nil, 0, nil
		}
		page := make([]InvocationTaskBinding, 0, len(resp.Response.InvocationTaskSet))
		for _, t := range resp.Response.InvocationTaskSet {
			if t.InvocationTaskId == nil || t.InstanceId == nil {
				continue
			}
			page = append(page, InvocationTaskBinding{
				InstanceID:       *t.InstanceId,
				InvocationTaskID: *t.InvocationTaskId,
			})
		}
		return page, len(resp.Response.InvocationTaskSet), nil
	}

	for attempt := 0; attempt < tatBatchRunCommandTaskBindingRetryMax; attempt++ {
		if attempt > 0 {
			time.Sleep(tatBatchRunCommandTaskBindingInterval)
		}

		var (
			out    []InvocationTaskBinding
			offset uint64
		)
		for {
			page, raw, err := collectPage(offset)
			if err != nil {
				return nil, err
			}
			out = append(out, page...)
			// 服务端单页返回少于 Limit，或 expected 已凑齐 → 终止本轮分页
			if raw < int(pageLimit) {
				break
			}
			if expectedCount > 0 && len(out) >= expectedCount {
				break
			}
			offset += uint64(raw)
			// 防御：避免 expected=0 又被疯狂分页
			if expectedCount == 0 {
				break
			}
		}

		if len(out) == 0 {
			continue // 进入下一次短重试
		}
		if expectedCount > 0 && len(out) < expectedCount {
			// 服务端尚未生成完所有 task 行，重试
			continue
		}

		// 稳定排序便于上层按 instance 对齐
		sort.Slice(out, func(i, j int) bool { return out[i].InstanceID < out[j].InstanceID })
		return out, nil
	}
	return nil, hcommon.I18nError(i18n.MsgTATInvocationTaskNotVisible,
		tatBatchRunCommandTaskBindingRetryMax, expectedCount)
}

// ============================================================================
// DescribeInvocationTasksBatch — 批量查询执行任务
// ============================================================================

// DescribeInvocationTasksBatch 按一批 InvocationTaskId 拉取每条 task 的状态、退出码、输出。
//
// 自动按 TATDescribeInvocationTasksBatchMax 拆批查询，结果合并。
// 单条 task 在 TAT 侧已过期 / 越权时，结果 map 中**不包含**该 key（由调用方判定 output_expired）。
//
// 返回值 map 的 key 是 InvocationTaskId。
func DescribeInvocationTasksBatch(
	ctx context.Context,
	invocationTaskIds []string,
) (map[string]InvocationTaskDetail, error) {
	if len(invocationTaskIds) == 0 {
		return map[string]InvocationTaskDetail{}, nil
	}
	client, err := tatBatchClientFactory(ctx)
	if err != nil {
		return nil, hcommon.I18nError(i18n.MsgTATQueryResultFailed).WithI18nDetail(i18n.MsgTATClientCreateFailed, err)
	}
	out := make(map[string]InvocationTaskDetail, len(invocationTaskIds))

	// 去重
	seen := make(map[string]struct{}, len(invocationTaskIds))
	uniqIDs := invocationTaskIds[:0:0]
	for _, id := range invocationTaskIds {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniqIDs = append(uniqIDs, id)
	}
	if len(uniqIDs) == 0 {
		return out, nil
	}

	for start := 0; start < len(uniqIDs); start += TATDescribeInvocationTasksBatchMax {
		end := start + TATDescribeInvocationTasksBatchMax
		if end > len(uniqIDs) {
			end = len(uniqIDs)
		}
		batch := uniqIDs[start:end]
		batchResults, err := describeInvocationTasksBatchOnce(client, batch)
		if err != nil {
			return nil, err
		}
		for k, v := range batchResults {
			out[k] = v
		}
	}
	return out, nil
}

// describeInvocationTasksBatchOnce 单次 TAT API 调用的实现，内部使用。
func describeInvocationTasksBatchOnce(client tatBatchClient, ids []string) (map[string]InvocationTaskDetail, error) {
	req := tat.NewDescribeInvocationTasksRequest()
	req.InvocationTaskIds = common.StringPtrs(ids)
	req.HideOutput = common.BoolPtr(false)
	req.Limit = common.Uint64Ptr(uint64(len(ids)))

	resp, err := client.DescribeInvocationTasks(req)
	if err != nil {
		slog.Error("[TAT] DescribeInvocationTasksBatch 失败", "batch_size", len(ids), "error", err)
		return nil, hcommon.I18nRichError(err, i18n.MsgTATQueryResultFailed).WithDetail(err.Error())
	}

	result := make(map[string]InvocationTaskDetail, len(ids))
	if resp.Response == nil {
		return result, nil
	}
	for _, t := range resp.Response.InvocationTaskSet {
		if t.InvocationTaskId == nil {
			continue
		}
		d := InvocationTaskDetail{InvocationTaskID: *t.InvocationTaskId}
		if t.InstanceId != nil {
			d.InstanceID = *t.InstanceId
		}
		if t.TaskStatus != nil {
			d.TaskStatus = *t.TaskStatus
			d.Finished = isTATTerminalStatus(*t.TaskStatus)
		}
		if t.StartTime != nil && *t.StartTime != "" {
			if ts, perr := parseTATTime(*t.StartTime); perr == nil {
				d.StartTime = &ts
			}
		}
		if t.EndTime != nil && *t.EndTime != "" {
			if ts, perr := parseTATTime(*t.EndTime); perr == nil {
				d.EndTime = &ts
			}
		}
		if t.ErrorInfo != nil {
			d.ErrorInfo = *t.ErrorInfo
		}
		if t.TaskResult != nil {
			if t.TaskResult.Output != nil {
				if decoded, decErr := base64.StdEncoding.DecodeString(*t.TaskResult.Output); decErr == nil {
					d.Stdout = string(decoded)
				} else {
					// 解码失败时退化保留原 base64，避免完全丢失输出
					d.Stdout = *t.TaskResult.Output
				}
			}
			if t.TaskResult.ExitCode != nil {
				ec := *t.TaskResult.ExitCode
				d.ExitCode = &ec
			}
		}
		result[*t.InvocationTaskId] = d
	}
	return result, nil
}

// isTATTerminalStatus 判断 TAT 任务状态字符串是否为终态。
// 参考 TAT 文档枚举：SUCCESS / FAILED / TIMEOUT / DELIVER_FAILED / START_FAILED / CANCELED ...
func isTATTerminalStatus(s string) bool {
	switch s {
	case "SUCCESS", "FAILED", "TIMEOUT",
		"DELIVER_FAILED", "START_FAILED",
		"CANCELED", "TERMINATED":
		return true
	}
	return false
}

// parseTATTime 解析 TAT 返回的 RFC3339 / "2006-01-02 15:04:05" 时间字符串。
func parseTATTime(s string) (time.Time, error) {
	for _, layout := range []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
	} {
		if ts, err := time.Parse(layout, s); err == nil {
			return ts, nil
		}
	}
	return time.Time{}, hcommon.I18nError(i18n.MsgTATUnrecognizedTimeFormat, s)
}

// MapTATTaskStatusToAgentTaskStatus 把 TAT 任务状态字符串映射为 model.AgentCommandTaskStatus。
//
// 映射表：
//
//	SUCCESS                                         → success
//	FAILED                                          → failed
//	TIMEOUT                                         → timeout
//	DELIVER_FAILED / START_FAILED / TERMINATED      → unreachable
//	CANCELED                                        → failed   （本期不做 cancel，理论不应出现，兜底归类失败）
//	PENDING                                         → pending
//	DELIVERING / RUNNING                            → in_progress
//	其它（未知）                                       → in_progress（保守，等下一轮轮询）
func MapTATTaskStatusToAgentTaskStatus(s string) string {
	switch s {
	case "SUCCESS":
		return "success"
	case "FAILED":
		return "failed"
	case "TIMEOUT":
		return "timeout"
	case "DELIVER_FAILED", "START_FAILED", "TERMINATED":
		return "unreachable"
	case "CANCELED":
		return "failed"
	case "PENDING":
		return "pending"
	case "DELIVERING", "RUNNING":
		return "in_progress"
	}
	return "in_progress"
}
