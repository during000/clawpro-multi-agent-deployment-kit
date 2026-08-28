package main

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// mockSender 是内存版 LogSender，用于单测上报逻辑。
type mockSender struct {
	mu           sync.Mutex
	sent         []logRecord
	pipelineSent []pipelineRecord
	closeErr     error
	closed       bool
}

func (m *mockSender) Send(r logRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, r)
	return nil
}

func (m *mockSender) SendPipeline(r pipelineRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pipelineSent = append(m.pipelineSent, r)
	return nil
}

func (m *mockSender) Close() error {
	m.closed = true
	return m.closeErr
}

func (m *mockSender) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sent)
}

func (m *mockSender) pipelineCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.pipelineSent)
}

func TestNewCLSSenderEmptyTopic(t *testing.T) {
	sender, err := newCLSSender("", "ap-guangzhou", "id", "key", "suffix")
	if sender != nil || err != nil {
		t.Fatalf("empty topic should return (nil, nil), got (%v, %v)", sender, err)
	}
}

func TestReportResultsMapping(t *testing.T) {
	mock := &mockSender{}
	results := []scriptResult{
		{script: "/x/test_pass.py", err: nil, duration: 1500 * time.Millisecond},
		{script: "/x/test_fail.py", err: errString("boom"), duration: 2500 * time.Millisecond},
	}
	reportCLS(mock, results, 4000*time.Millisecond)

	if mock.count() != 2 {
		t.Fatalf("expected 2 case records, got %d", mock.count())
	}
	pass := mock.sent[0]
	if pass.Script != "/x/test_pass.py" || pass.Status != "pass" || pass.Error != "" || pass.DurationMs != 1500 {
		t.Errorf("pass record mismatch: %+v", pass)
	}
	fail := mock.sent[1]
	if fail.Script != "/x/test_fail.py" || fail.Status != "fail" || fail.Error != "boom" || fail.DurationMs != 2500 {
		t.Errorf("fail record mismatch: %+v", fail)
	}
	// 流水线级汇总：2 条、1 失败、整体 fail
	if mock.pipelineCount() != 1 {
		t.Fatalf("expected 1 pipeline record, got %d", mock.pipelineCount())
	}
	p := mock.pipelineSent[0]
	if p.Status != "fail" || p.Total != 2 || p.Passed != 1 || p.Failed != 1 || p.DurationMs != 4000 {
		t.Errorf("pipeline record mismatch: %+v", p)
	}
}

func TestReportResultsNilSender(t *testing.T) {
	// 不应 panic
	reportCLS(nil, []scriptResult{{script: "/x/a.py"}}, time.Second)
	closeSender(nil)
}

// TestWriteReportHTMLOnly 验证 writeReport 只生成 HTML 报告，
// 不涉及任何 CLS 上报（CLS 上报由 main 的 defer 独立调用）。
func TestWriteReportHTMLOnly(t *testing.T) {
	dir := t.TempDir()
	reportDir = dir
	defer func() { reportDir = "" }()

	mock := &mockSender{}
	results := []scriptResult{
		{script: "setup", err: errString("k8s connect failed"), duration: 100 * time.Millisecond, output: "diag"},
	}
	writeReport(results, 100*time.Millisecond)

	// HTML 报告已生成
	if _, err := os.Stat(filepath.Join(dir, "index.html")); err != nil {
		t.Errorf("report html not generated: %v", err)
	}
	// writeReport 不应触发任何 CLS 上报
	if mock.count() != 0 {
		t.Errorf("writeReport should not send CLS case records, got %d", mock.count())
	}
	if mock.pipelineCount() != 0 {
		t.Errorf("writeReport should not send CLS pipeline records, got %d", mock.pipelineCount())
	}
}

// TestReportPipeline 验证流水线级汇总上报（setup 失败场景：1 条、全失败）。
func TestReportPipeline(t *testing.T) {
	mock := &mockSender{}
	results := []scriptResult{
		{script: "setup", err: errString("k8s connect failed"), duration: 100 * time.Millisecond},
	}
	reportCLS(mock, results, 100*time.Millisecond)

	if mock.count() != 1 {
		t.Fatalf("expected 1 case record, got %d", mock.count())
	}
	c := mock.sent[0]
	if c.Script != "setup" || c.Status != "fail" || c.Error != "k8s connect failed" {
		t.Errorf("case record mismatch: %+v", c)
	}
	if mock.pipelineCount() != 1 {
		t.Fatalf("expected 1 pipeline record, got %d", mock.pipelineCount())
	}
	p := mock.pipelineSent[0]
	if p.Status != "fail" || p.Total != 1 || p.Failed != 1 || p.Passed != 0 {
		t.Errorf("pipeline record mismatch: %+v", p)
	}
}

func TestCloseSender(t *testing.T) {
	mock := &mockSender{}
	closeSender(mock)
	if !mock.closed {
		t.Error("Close should have been called")
	}
}

// errString 是最小错误实现，便于在测试中构造带消息的错误。
type errString string

func (e errString) Error() string { return string(e) }
