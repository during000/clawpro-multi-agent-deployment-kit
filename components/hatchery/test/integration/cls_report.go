package main

import (
	"fmt"
	"log"
	"strconv"
	"time"
	"unicode/utf8"

	cls "github.com/tencentcloud/tencentcloud-cls-sdk-go"
)

// logRecord 是单条待上报到 CLS 的测试结果。
type logRecord struct {
	RunID      string
	Script     string // 用例脚本路径（完整路径）
	Status     string // "pass" | "fail"
	DurationMs int64
	Error      string
	Output     string // 用例执行输出（stdout/stderr 摘要）
	Timestamp  time.Time
}

// LogSender 上报抽象，便于单测用 mock 替换。
type LogSender interface {
	Send(r logRecord) error
	SendPipeline(r pipelineRecord) error
	Close() error
}

// pipelineRecord 是单次流水线运行的汇总记录（一条 pipeline 级日志）。
// 字段对应报告中 Summary 的统计项，status 判定逻辑与原报告一致：failed>0 则 fail。
type pipelineRecord struct {
	RunID      string
	Status     string  // "success" | "fail"
	Total      int     // 用例总数
	Passed     int     // 通过数
	Failed     int     // 失败数
	PassRate   float64 // 通过率（0~100）
	DurationMs int64   // 整次运行耗时（毫秒）
	Timestamp  time.Time
}

// clsSender 基于腾讯云 CLS Producer SDK 的实现。
type clsSender struct {
	client  *cls.AsyncProducerClient
	topicID string
	runID   string
}

// silentCallback 仅在发送失败时打印告警，不影响主流程（best-effort）。
type silentCallback struct{}

func (silentCallback) Success(*cls.Result)     {}
func (silentCallback) Fail(result *cls.Result) { log.Printf("CLS report: send failed: %v", result) }

// newCLSSender 创建 CLS 上报器。topicID 为空时返回 (nil, nil)，调用方需判空跳过。
// clsSecretID/clsSecretKey 是 CLS 专用凭证，与应用自身的 --ak/--sk 解耦。
func newCLSSender(clsTopicID, clsRegion, clsSecretID, clsSecretKey, runID string) (LogSender, error) {
	if clsTopicID == "" {
		return nil, nil
	}
	config := cls.GetDefaultAsyncProducerClientConfig()
	config.AccessKeyID = clsSecretID
	config.AccessKeySecret = clsSecretKey
	config.SetEndpointByRegionAndNetworkType(cls.Region(clsRegion), cls.Extranet)
	client, err := cls.NewAsyncProducerClient(config)
	if err != nil {
		return nil, fmt.Errorf("create CLS producer: %w", err)
	}
	client.Start()
	return &clsSender{client: client, topicID: clsTopicID, runID: runID}, nil
}

// CLS 单条日志所有 content value 总大小上限为 1MB，超出会被拒绝。
const clsMaxTotalBytes = 1 * 1024 * 1024

// truncateForCLS 将字符串截断到不超过 maxBytes 字节，并在截断时追加提示。
// 按 rune 截断避免切断多字节字符。
func truncateForCLS(s string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if len(s) <= maxBytes {
		return s
	}
	const suffix = "\n...[truncated by CLS 1MB limit]"
	cutoff := maxBytes - len(suffix)
	if cutoff <= 0 {
		return s[:maxBytes]
	}
	for cutoff > 0 && !utf8.RuneStart(s[cutoff]) {
		cutoff--
	}
	return s[:cutoff] + suffix
}

func (s *clsSender) Send(r logRecord) error {
	// 用 run 级元数据补全每条记录
	r.RunID = s.runID
	if r.Timestamp.IsZero() {
		r.Timestamp = time.Now()
	}
	t := r.Timestamp.Unix()

	// 固定字段（始终很小）
	runID := r.RunID
	script := r.Script
	status := r.Status
	durationMs := strconv.FormatInt(r.DurationMs, 10)
	timestamp := r.Timestamp.Format(time.RFC3339)
	output := r.Output
	errMsg := r.Error

	// 计算 overhead：固定字段值 + 所有 key 名称 + CLS 结构缓冲
	fixedKeys := len("run_id") + len("script") + len("status") +
		len("duration_ms") + len("error") + len("output") + len("timestamp")
	overhead := len(runID) + len(script) + len(status) +
		len(durationMs) + len(timestamp) + fixedKeys + 256

	// 总大小超限时：优先截断 output，output 清空后仍超限再截断 error
	if overhead+len(output)+len(errMsg) > clsMaxTotalBytes {
		available := clsMaxTotalBytes - overhead
		if len(errMsg) < available {
			// error 能放下，剩余空间给 output
			output = truncateForCLS(output, available-len(errMsg))
		} else {
			// error 单独就超限，清空 output，截断 error
			output = ""
			errMsg = truncateForCLS(errMsg, available)
		}
	}

	logEntry := &cls.Log{
		Time: &t,
		Contents: []*cls.Log_Content{
			{Key: ptr("run_id"), Value: ptr(runID)},
			{Key: ptr("script"), Value: ptr(script)},
			{Key: ptr("status"), Value: ptr(status)},
			{Key: ptr("duration_ms"), Value: ptr(durationMs)},
			{Key: ptr("error"), Value: ptr(errMsg)},
			{Key: ptr("output"), Value: ptr(output)},
			{Key: ptr("timestamp"), Value: ptr(timestamp)},
		},
	}
	return s.client.SendLog(s.topicID, logEntry, silentCallback{})
}

func (s *clsSender) Close() error {
	return s.client.Close(3000)
}

// SendPipeline 上报一条流水线级汇总记录。run_id 由 sender 自动补全。
func (s *clsSender) SendPipeline(r pipelineRecord) error {
	r.RunID = s.runID
	if r.Timestamp.IsZero() {
		r.Timestamp = time.Now()
	}
	t := r.Timestamp.Unix()
	logEntry := &cls.Log{
		Time: &t,
		Contents: []*cls.Log_Content{
			{Key: ptr("run_id"), Value: ptr(r.RunID)},
			{Key: ptr("stage"), Value: ptr("pipeline")},
			{Key: ptr("status"), Value: ptr(r.Status)},
			{Key: ptr("total"), Value: ptr(strconv.Itoa(r.Total))},
			{Key: ptr("passed"), Value: ptr(strconv.Itoa(r.Passed))},
			{Key: ptr("failed"), Value: ptr(strconv.Itoa(r.Failed))},
			{Key: ptr("pass_rate"), Value: ptr(strconv.FormatFloat(r.PassRate, 'f', 1, 64))},
			{Key: ptr("duration_ms"), Value: ptr(strconv.FormatInt(r.DurationMs, 10))},
			{Key: ptr("timestamp"), Value: ptr(r.Timestamp.Format(time.RFC3339))},
		},
	}
	return s.client.SendLog(s.topicID, logEntry, silentCallback{})
}

func ptr(s string) *string { return &s }

// reportCLS 将用例级结果与流水线级汇总一并上报到 CLS；sender 为 nil 时安全跳过。
// 用例级每条结果直接以其完整 script 路径记录，不再依赖 scriptsDir 计算相对路径。
func reportCLS(sender LogSender, results []scriptResult, totalDuration time.Duration) {
	if sender == nil {
		return
	}
	for _, r := range results {
		rec := logRecord{
			Script:     r.script,
			Status:     "pass",
			DurationMs: r.duration.Milliseconds(),
			Output:     r.output,
			Timestamp:  time.Now(),
		}
		if r.err != nil {
			rec.Status = "fail"
			rec.Error = r.err.Error()
		}
		if sendErr := sender.Send(rec); sendErr != nil {
			log.Printf("CLS report: send %q failed: %v", r.script, sendErr)
		}
	}
	if sendErr := sender.SendPipeline(summarize(results, totalDuration)); sendErr != nil {
		log.Printf("CLS report: send pipeline record failed: %v", sendErr)
	}
}

// closeSender 安全关闭上报器（nil 安全）。
func closeSender(sender LogSender) {
	if sender == nil {
		return
	}
	if err := sender.Close(); err != nil {
		log.Printf("CLS report: close: %v", err)
	}
}

// summarize 由用例结果计算汇总，判定逻辑与原报告一致：failed>0 则整体为 fail。
func summarize(results []scriptResult, totalDuration time.Duration) pipelineRecord {
	total := len(results)
	passed, failed := 0, 0
	for _, r := range results {
		if r.err != nil {
			failed++
		} else {
			passed++
		}
	}
	passRate := 0.0
	if total > 0 {
		passRate = float64(passed) / float64(total) * 100
	}
	status := "success"
	if failed > 0 {
		status = "fail"
	}
	return pipelineRecord{
		Status:     status,
		Total:      total,
		Passed:     passed,
		Failed:     failed,
		PassRate:   passRate,
		DurationMs: totalDuration.Milliseconds(),
		Timestamp:  time.Now(),
	}
}
