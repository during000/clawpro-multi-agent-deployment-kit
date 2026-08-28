package controller

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// testChunk 测试中使用的预分割数据块
type testChunk struct {
	data []byte
	idx  int
}

// parseDDChunkIdx 从 mock 脚本中解析出 chunk 索引。
// 脚本形如：`dd if=... bs=N count=N skip=OFFSET iflag=skip_bytes,count_bytes status=none 2>/dev/null | base64 -w0`
// 返回 chunk 索引 (offset/chunkSize)。
func parseDDChunkIdx(script string, chunkSize int) (int, bool) {
	i := strings.Index(script, "skip=")
	if i < 0 {
		return 0, false
	}
	rest := script[i+len("skip="):]
	var offset int
	if _, err := fmt.Sscanf(rest, "%d", &offset); err != nil {
		return 0, false
	}
	return offset / chunkSize, true
}

// TestReadSkillsViaChunks_SingleChunk 测试单个分块场景
func TestReadSkillsViaChunks_SingleChunk(t *testing.T) {
	original := runInlineScript

	payload := `[{"name":"test-skill","description":"A test skill","eligible":true}]`
	var gzBuf bytes.Buffer
	gw := gzip.NewWriter(&gzBuf)
	gw.Write([]byte(payload))
	gw.Close()
	gzipData := gzBuf.Bytes()

	callCount := int32(0)
	cleanupDone := make(chan struct{})
	runInlineScript = func(ctx context.Context, instanceId string, script string, timeout uint64) (string, error) {
		if strings.Contains(script, "rm -f") {
			close(cleanupDone)
			return "", nil
		}
		atomic.AddInt32(&callCount, 1)
		return base64.StdEncoding.EncodeToString(gzipData), nil
	}

	result, err := readSkillsViaChunks(context.Background(), "ins-test", "/tmp/test.gz", len(gzipData), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != payload {
		t.Errorf("expected %q, got %q", payload, result)
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("expected 1 TAT call, got %d", callCount)
	}

	// 等待后台清理 goroutine 完成后再恢复 mock，避免竞态
	select {
	case <-cleanupDone:
	case <-time.After(5 * time.Second):
	}
	runInlineScript = original
}

// TestReadSkillsViaChunks_MultipleChunks 测试多分块并发读取和组装
func TestReadSkillsViaChunks_MultipleChunks(t *testing.T) {
	original := runInlineScript
	defer func() { runInlineScript = original }()

	const chunkSize = 16000
	const numSkills = 200

	// 生成不可压缩的 JSON
	skillsJSON := "["
	for i := 0; i < numSkills; i++ {
		if i > 0 {
			skillsJSON += ","
		}
		// 只使用 JSON-safe 字符：字母数字 + 空格 + 标点(不含 " \ )
		safeChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 .,!?-_:;"
		desc := make([]byte, 200)
		for j := range desc {
			desc[j] = safeChars[rand.Intn(len(safeChars))]
		}
		skillsJSON += fmt.Sprintf(`{"name":"pad-%04d","description":"%s","eligible":true}`, i, string(desc))
	}
	skillsJSON += "]"

	var gzBuf bytes.Buffer
	gw := gzip.NewWriter(&gzBuf)
	gw.Write([]byte(skillsJSON))
	gw.Close()
	gzipData := gzBuf.Bytes()
	numChunks := (len(gzipData) + chunkSize - 1) / chunkSize

	t.Logf("gzip=%d bytes, chunks=%d", len(gzipData), numChunks)
	if numChunks < 2 {
		t.Skip("need at least 2 chunks for multi-chunk test")
	}

	var mu sync.Mutex
	callOrder := []int{}
	receivedChunks := map[int]bool{}

	// 预先分割 gzip 数据为 chunks
	chunksPre := make([]testChunk, numChunks)
	for i := 0; i < numChunks; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(gzipData) {
			end = len(gzipData)
		}
		chunksPre[i] = testChunk{data: gzipData[start:end], idx: i}
	}

	runInlineScript = func(ctx context.Context, instanceId string, script string, timeout uint64) (string, error) {
		if strings.Contains(script, "rm -f") {
			return "", nil
		}
		chunkIdx, ok := parseDDChunkIdx(script, chunkSize)
		if !ok || chunkIdx < 0 || chunkIdx >= numChunks {
			return "", fmt.Errorf("invalid script: %s", script)
		}
		mu.Lock()
		receivedChunks[chunkIdx] = true
		callOrder = append(callOrder, chunkIdx)
		mu.Unlock()
		return base64.StdEncoding.EncodeToString(chunksPre[chunkIdx].data), nil
	}

	result, err := readSkillsViaChunks(context.Background(), "ins-test", "/tmp/test.gz", len(gzipData), numChunks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != skillsJSON {
		t.Errorf("decoded JSON mismatch. want %d bytes, got %d bytes", len(skillsJSON), len(result))
	}

	mu.Lock()
	for i := 0; i < numChunks; i++ {
		if !receivedChunks[i] {
			t.Errorf("chunk %d was never requested", i)
		}
	}
	mu.Unlock()
	t.Logf("all %d chunks assembled (call order: %v)", numChunks, callOrder)
}

// TestReadSkillsViaChunks_ErrorCleanup 测试某块失败时触发清理
func TestReadSkillsViaChunks_ErrorCleanup(t *testing.T) {
	original := runInlineScript

	cleanedUp := int32(0)
	// 期望被清理调用 2 次：同步清理 + 异步清理 goroutine
	cleanupDone := make(chan struct{}, 2)
	runInlineScript = func(ctx context.Context, instanceId string, script string, timeout uint64) (string, error) {
		if strings.Contains(script, "rm -f") {
			atomic.AddInt32(&cleanedUp, 1)
			cleanupDone <- struct{}{}
			return "", nil
		}
		// 第一块（skip=0）失败
		if strings.Contains(script, "skip=0 ") {
			return "", fmt.Errorf("TAT timeout")
		}
		return "dummy", nil
	}

	_, err := readSkillsViaChunks(context.Background(), "ins-test", "/tmp/test.gz", 30000, 3)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if atomic.LoadInt32(&cleanedUp) == 0 {
		t.Error("cleanup not called on error")
	}
	t.Logf("error (expected): %v, cleanup called: %d times", err, atomic.LoadInt32(&cleanedUp))

	// 等待后台异步清理 goroutine 也完成（同步清理已在主流程内完成）
	// 一共会调用 2 次 rm -f：一次同步、一次异步 goroutine
	deadline := time.After(5 * time.Second)
	for atomic.LoadInt32(&cleanedUp) < 2 {
		select {
		case <-cleanupDone:
		case <-deadline:
			// 超时时也退出循环，避免测试挂死
			goto restore
		}
	}
restore:
	runInlineScript = original
}

// TestReadSkillsViaChunks_Concurrency 验证 5 路并发限制
func TestReadSkillsViaChunks_Concurrency(t *testing.T) {
	original := runInlineScript
	defer func() { runInlineScript = original }()

	numChunks := 10
	const chunkSize = 16000

	// 生成 gzip 测试数据（足够大以分成 10 个 16KB 块）
	var gzBuf bytes.Buffer
	gw := gzip.NewWriter(&gzBuf)
	// 写 ~160KB 不可压缩数据 → gzip ~160KB
	for i := 0; i < 800; i++ {
		// 只使用 JSON-safe 字符：字母数字 + 空格 + 标点(不含 " \ )
		safeChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 .,!?-_:;"
		desc := make([]byte, 200)
		for j := range desc {
			desc[j] = safeChars[rand.Intn(len(safeChars))]
		}
		gw.Write(desc)
	}
	gw.Close()
	gzipData := gzBuf.Bytes()
	// 如果 gzip 数据不足 10 块，减少块数
	actualChunks := (len(gzipData) + chunkSize - 1) / chunkSize
	if actualChunks < numChunks {
		t.Logf("gzip only %d bytes (%d chunks), testing with %d chunks", len(gzipData), actualChunks, actualChunks)
		numChunks = actualChunks
	}

	var maxConcurrent, currentConcurrent int32
	// 预分割 chunks，按调用顺序返回
	chunksPre := make([]testChunk, numChunks)
	for i := 0; i < numChunks; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(gzipData) {
			end = len(gzipData)
		}
		chunksPre[i] = testChunk{data: gzipData[start:end]}
	}

	runInlineScript = func(ctx context.Context, instanceId string, script string, timeout uint64) (string, error) {
		if strings.Contains(script, "rm -f") {
			return "", nil
		}
		cur := atomic.AddInt32(&currentConcurrent, 1)
		for {
			m := atomic.LoadInt32(&maxConcurrent)
			if cur <= m {
				break
			}
			if atomic.CompareAndSwapInt32(&maxConcurrent, m, cur) {
				break
			}
		}
		defer atomic.AddInt32(&currentConcurrent, -1)

		chunkIdx, ok := parseDDChunkIdx(script, chunkSize)
		if !ok || chunkIdx < 0 || chunkIdx >= numChunks {
			return "", fmt.Errorf("invalid script: %s", script)
		}
		return base64.StdEncoding.EncodeToString(chunksPre[chunkIdx].data), nil
	}

	_, err := readSkillsViaChunks(context.Background(), "ins-test", "/tmp/test.gz", len(gzipData), numChunks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mc := atomic.LoadInt32(&maxConcurrent); mc > 5 {
		t.Errorf("max concurrency %d exceeds limit of 5", mc)
	}
	t.Logf("max concurrency observed: %d (limit: 5)", atomic.LoadInt32(&maxConcurrent))
}

// TestListInstanceSkills_ChunkedMode_500Skills 极端场景端到端测试：
// 500 个 skill → gzip 76KB → 5 块分块传输 → 合并解压 → 验证 JSON 完整性
func TestListInstanceSkills_ChunkedMode_500Skills(t *testing.T) {
	origRunner := listSkillsScriptRunner
	origInline := runInlineScript
	defer func() {
		listSkillsScriptRunner = origRunner
		runInlineScript = origInline
	}()

	// ── Step 1: 生成 500 个 skill 的 JSON 并 gzip ──
	skillsJSON := "["
	for i := 0; i < 500; i++ {
		if i > 0 {
			skillsJSON += ","
		}
		// 只使用 JSON-safe 字符：字母数字 + 空格 + 标点(不含 " \ )
		safeChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 .,!?-_:;"
		desc := make([]byte, 200)
		for j := range desc {
			desc[j] = safeChars[rand.Intn(len(safeChars))]
		}
		skillsJSON += fmt.Sprintf(`{"name":"extreme-%04d","description":"%s","eligible":true}`, i, string(desc))
	}
	skillsJSON += "]"

	var gzBuf bytes.Buffer
	gw := gzip.NewWriter(&gzBuf)
	gw.Write([]byte(skillsJSON))
	gw.Close()
	gzipData := gzBuf.Bytes()

	const chunkSize = 16000
	numChunks := (len(gzipData) + chunkSize - 1) / chunkSize

	t.Logf("500 skills: JSON=%d bytes, gzip=%d bytes, chunks=%d",
		len(skillsJSON), len(gzipData), numChunks)

	if numChunks < 2 {
		t.Fatal("expected at least 2 chunks for 500 skills")
	}

	// ── Step 2: Mock list_skills.sh 返回分块元数据 ──
	fileMeta := fmt.Sprintf(`{"mode":"file","path":"/tmp/skills-500.gz","size":%d,"chunks":%d}`,
		len(gzipData), numChunks)

	listSkillsScriptRunner = func(ctx context.Context, instanceId, scriptName string,
		timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		return fileMeta, nil
	}

	// ── Step 3: Mock tail+head+base64 逐块返回 ──
	var mu sync.Mutex
	receivedChunks := map[int]bool{}

	// 预分割 chunks
	chunksPre := make([]testChunk, numChunks)
	for i := 0; i < numChunks; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(gzipData) {
			end = len(gzipData)
		}
		chunksPre[i] = testChunk{data: gzipData[start:end], idx: i}
	}

	runInlineScript = func(ctx context.Context, instanceId string, script string, timeout uint64) (string, error) {
		if strings.Contains(script, "rm -f") {
			return "", nil
		}
		chunkIdx, ok := parseDDChunkIdx(script, chunkSize)
		if !ok || chunkIdx < 0 || chunkIdx >= numChunks {
			return "", fmt.Errorf("invalid script: %s", script)
		}
		mu.Lock()
		receivedChunks[chunkIdx] = true
		mu.Unlock()
		return base64.StdEncoding.EncodeToString(chunksPre[chunkIdx].data), nil
	}

	// ── Step 4: 执行 listInstanceSkills ──
	result, err := listInstanceSkills(context.Background(), "ins-extreme-test", "openclaw", "openclaw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// ── Step 5: 验证结果 ──
	if result != skillsJSON {
		t.Errorf("JSON mismatch. want %d bytes, got %d bytes", len(skillsJSON), len(result))
	}

	// 验证所有块都被请求
	mu.Lock()
	for i := 0; i < numChunks; i++ {
		if !receivedChunks[i] {
			t.Errorf("chunk %d was never requested", i)
		}
	}
	mu.Unlock()

	// 验证 JSON 可解析并且有 500 个 skill
	var parsed []map[string]interface{}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}
	if len(parsed) != 500 {
		t.Errorf("expected 500 skills, got %d", len(parsed))
	}

	t.Logf("✓ 500 skills: %d chunks assembled, decompressed, parsed successfully", numChunks)
}

// TestListInstanceSkills_Base64Mode 常规路径：base64+gzip 单次传输
func TestListInstanceSkills_Base64Mode(t *testing.T) {
	origRunner := listSkillsScriptRunner
	defer func() { listSkillsScriptRunner = origRunner }()

	payload := `[{"name":"skill-a","description":"test","eligible":true}]`
	var gzBuf bytes.Buffer
	gw := gzip.NewWriter(&gzBuf)
	gw.Write([]byte(payload))
	gw.Close()

	listSkillsScriptRunner = func(ctx context.Context, instanceId, scriptName string,
		timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		return base64.StdEncoding.EncodeToString(gzBuf.Bytes()), nil
	}

	result, err := listInstanceSkills(context.Background(), "ins-test", "openclaw", "openclaw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != payload {
		t.Errorf("expected %q, got %q", payload, result)
	}
	t.Log("✓ Base64+gzip single transfer mode works correctly")
}

// TestListInstanceSkills_FallbackMode 旧版脚本原始 JSON 透传
func TestListInstanceSkills_FallbackMode(t *testing.T) {
	origRunner := listSkillsScriptRunner
	defer func() { listSkillsScriptRunner = origRunner }()

	rawJSON := `[{"name":"old-skill","version":"1.0"}]`

	listSkillsScriptRunner = func(ctx context.Context, instanceId, scriptName string,
		timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		return rawJSON, nil
	}

	result, err := listInstanceSkills(context.Background(), "ins-test", "openclaw", "openclaw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != rawJSON {
		t.Errorf("expected %q, got %q", rawJSON, result)
	}
	t.Log("✓ Fallback raw JSON passthrough works correctly")
}

// ── 新增覆盖率补充测试 ───────────────────────────────────────────

// TestIsTATRetryableError_NilError 测试 nil error
func TestIsTATRetryableError_NilError(t *testing.T) {
	if isTATRetryableError(nil) {
		t.Error("nil error should not be retryable")
	}
}

// TestIsTATRetryableError_AllTypes 覆盖所有新增错误类型
func TestIsTATRetryableError_AllTypes(t *testing.T) {
	tests := []struct {
		msg  string
		want bool
	}{
		{"no route to host", true},
		{"network is unreachable", true},
		{"no such host: tat.tencentcloudapi.com", true},
		{"temporary failure in name resolution", true},
		{"timeout", true},
		{"EOF", true},
		{"i/o timeout", true},
		{"write: broken pipe", true},
		{"TLS handshake timeout", true},
		{"connection refused", true},
		{"connection reset by peer", true},
		{"something else that is not retryable", false},
	}
	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			err := fmt.Errorf("%s", tt.msg)
			if got := isTATRetryableError(err); got != tt.want {
				t.Errorf("isTATRetryableError(%q) = %v, want %v", tt.msg, got, tt.want)
			}
		})
	}
}

// TestReadSkillsViaChunks_InvalidParams 测试无效参数（totalSize<=0 或 chunks<=0）
func TestReadSkillsViaChunks_InvalidParams(t *testing.T) {
	tests := []struct {
		name      string
		totalSize int
		chunks    int
	}{
		{"zero size", 0, 1},
		{"negative size", -1, 1},
		{"zero chunks", 100, 0},
		{"negative chunks", 100, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := readSkillsViaChunks(context.Background(), "ins-test", "/tmp/test.gz", tt.totalSize, tt.chunks)
			if err == nil {
				t.Errorf("expected error for invalid params (size=%d, chunks=%d)", tt.totalSize, tt.chunks)
			}
		})
	}
}

// TestReadSkillsViaChunks_DecodeError 测试 base64 解码失败
func TestReadSkillsViaChunks_DecodeError(t *testing.T) {
	original := runInlineScript
	defer func() { runInlineScript = original }()

	runInlineScript = func(ctx context.Context, instanceId string, script string, timeout uint64) (string, error) {
		if strings.Contains(script, "rm -f") {
			return "", nil
		}
		return "!!!not-valid-base64!!!", nil
	}
	_, err := readSkillsViaChunks(context.Background(), "ins-test", "/tmp/test.gz", 100, 1)
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}
}

// TestReadSkillsViaChunks_LengthMismatchThenSuccess 分块长度不匹配 → 重试 → 成功
func TestReadSkillsViaChunks_LengthMismatchThenSuccess(t *testing.T) {
	original := runInlineScript
	defer func() { runInlineScript = original }()

	payload := `[{"name":"test-skill"}]`
	var gzBuf bytes.Buffer
	gw := gzip.NewWriter(&gzBuf)
	gw.Write([]byte(payload))
	gw.Close()
	gzipData := gzBuf.Bytes()

	callCount := int32(0)
	runInlineScript = func(ctx context.Context, instanceId string, script string, timeout uint64) (string, error) {
		if strings.Contains(script, "rm -f") {
			return "", nil
		}
		count := atomic.AddInt32(&callCount, 1)
		if count == 1 {
			return base64.StdEncoding.EncodeToString(gzipData[:len(gzipData)-5]), nil
		}
		return base64.StdEncoding.EncodeToString(gzipData), nil
	}

	result, err := readSkillsViaChunks(context.Background(), "ins-test", "/tmp/test.gz", len(gzipData), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != payload {
		t.Errorf("expected %q, got %q", payload, result)
	}
	if atomic.LoadInt32(&callCount) != 2 {
		t.Errorf("expected 2 calls (1 mismatch + 1 success), got %d", callCount)
	}
	t.Log("length mismatch retry success")
}

// TestReadSkillsViaChunks_NilData 测试 chunk 返回空导致解码后 len=0
func TestReadSkillsViaChunks_NilData(t *testing.T) {
	original := runInlineScript
	defer func() { runInlineScript = original }()

	runInlineScript = func(ctx context.Context, instanceId string, script string, timeout uint64) (string, error) {
		if strings.Contains(script, "rm -f") {
			return "", nil
		}
		if strings.Contains(script, "skip=0") {
			return "", nil
		}
		return base64.StdEncoding.EncodeToString(make([]byte, 100)), nil
	}
	_, err := readSkillsViaChunks(context.Background(), "ins-test", "/tmp/test.gz", 200, 2)
	if err == nil {
		t.Fatal("expected error for zero-length chunk data, got nil")
	}
}

// TestReadSkillsViaChunks_AssembledSizeMismatch 测试合并后总长度与预期不符。
// 用 2 个 chunk，每个 chunk 单独通过长度校验（各 16000 字节），
// 但 totalSize 声明为 50000，合并后 32000 ≠ 50000 → 触发 assembled 检查。
func TestReadSkillsViaChunks_AssembledSizeMismatch(t *testing.T) {
	original := runInlineScript
	defer func() { runInlineScript = original }()

	runInlineScript = func(ctx context.Context, instanceId string, script string, timeout uint64) (string, error) {
		if strings.Contains(script, "rm -f") {
			return "", nil
		}
		chunkIdx, ok := parseDDChunkIdx(script, skillsChunkSize)
		if !ok {
			return "", fmt.Errorf("invalid script: %s", script)
		}
		_ = chunkIdx
		return base64.StdEncoding.EncodeToString(make([]byte, 16000)), nil
	}
	_, err := readSkillsViaChunks(context.Background(), "ins-test", "/tmp/test.gz", 50000, 2)
	if err == nil {
		t.Fatal("expected assembled size mismatch error, got nil")
	}
	// I18N 层会将原始 cause 包装为国际化文本，Error() 返回的是翻译后的消息。
	t.Logf("assembled size mismatch correctly detected")
}

// TestReadSkillsViaChunks_GzipDecompressError 测试 gzip 解压失败（非 gzip 数据）
func TestReadSkillsViaChunks_GzipDecompressError(t *testing.T) {
	original := runInlineScript
	defer func() { runInlineScript = original }()

	invalidData := make([]byte, 100)
	runInlineScript = func(ctx context.Context, instanceId string, script string, timeout uint64) (string, error) {
		if strings.Contains(script, "rm -f") {
			return "", nil
		}
		return base64.StdEncoding.EncodeToString(invalidData), nil
	}
	_, err := readSkillsViaChunks(context.Background(), "ins-test", "/tmp/test.gz", 100, 1)
	if err == nil {
		t.Fatal("expected gzip decompress error, got nil")
	}
}

// TestListInstanceSkills_RunScriptError 测试 RunScript 失败场景
func TestListInstanceSkills_RunScriptError(t *testing.T) {
	origRunner := listSkillsScriptRunner
	defer func() { listSkillsScriptRunner = origRunner }()

	listSkillsScriptRunner = func(ctx context.Context, instanceId, scriptName string,
		timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		return "partial output", fmt.Errorf("TAT execution failed")
	}

	output, err := listInstanceSkills(context.Background(), "ins-test", "openclaw", "openclaw")
	if err == nil {
		t.Fatal("expected error from RunScript, got nil")
	}
	if output != "partial output" {
		t.Errorf("expected partial output, got %q", output)
	}
	t.Logf("RunScript error handled correctly: %v", err)
}

// TestListInstanceSkills_EdgeJSON 测试非 fileMeta 非 base64 的 JSON 走 fallback
func TestListInstanceSkills_EdgeJSON(t *testing.T) {
	origRunner := listSkillsScriptRunner
	defer func() { listSkillsScriptRunner = origRunner }()

	rawJSON := `[{"name":"old-skill","mode":"file"}]`

	listSkillsScriptRunner = func(ctx context.Context, instanceId, scriptName string,
		timeout uint64, runtimeUser string, onOutput func(string), params map[string]string) (string, error) {
		return rawJSON, nil
	}

	result, err := listInstanceSkills(context.Background(), "ins-test", "openclaw", "openclaw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != rawJSON {
		t.Errorf("expected fallback to raw JSON, got %q", result)
	}
	t.Log("Edge JSON fallback passthrough works")
}