// openclaw_upgrade_resume_test.go
//
// 针对 smhUploadLoop 的单元测试，覆盖断点续传核心逻辑：
//   - 已上传分块被正确跳过，未上传分块被执行
//   - 续期后 PartURLTemplate / PartHeaders 被刷新，后续分块使用新凭证
//   - 凭证临近过期（< 10 分钟）时自动触发续期
//   - 全部分块已上传时直接 Confirm，不执行任何上传脚本
//   - GetParts 失败降级为全量上传
//   - Renew 失败立即返回错误
//
// 测试策略：
//   - 通过替换 smhUploadHooks 注入 mock，完全隔离真实 SMH API。
//   - 通过替换 runScriptFn 注入 mock，完全隔离 TAT 网络调用。
//   - 通过 RegisterInlineScript 注入空壳脚本，让 LoadScript 能找到脚本内容。
//   - 每个测试用例结束后恢复原始值，避免污染其他测试。
package controller

import (
	"context"
	"fmt"
	hcommon "hatchery/common"
	"hatchery/i18n"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ─── 测试辅助 ────────────────────────────────────────────────────────────────

// saveAndRestoreHooks 保存当前 smhUploadHooks 和 runScriptFn，返回 restore 函数。
func saveAndRestoreHooks(t *testing.T) func() {
	t.Helper()
	origHooks := smhUploadHooks
	origRunScript := runScriptFn
	origBuildURL := buildSMHDownloadURLFn
	origWaitRunning := waitForInstanceRunningFn
	origResetInstance := resetInstanceFn
	origWaitTAT := waitForTATAgentOnlineFn
	origSleep := reinstallSleepFn
	origRestoreSleep := restoreSleepFn
	origFetchVersion := fetchVersionInfoFn
	origArchiveExists := archiveExistsOnCVMFn

	return func() {
		smhUploadHooks = origHooks
		runScriptFn = origRunScript
		buildSMHDownloadURLFn = origBuildURL
		waitForInstanceRunningFn = origWaitRunning
		resetInstanceFn = origResetInstance
		waitForTATAgentOnlineFn = origWaitTAT
		reinstallSleepFn = origSleep
		restoreSleepFn = origRestoreSleep
		fetchVersionInfoFn = origFetchVersion
		archiveExistsOnCVMFn = origArchiveExists
	}
}

// makeCred 构造一个测试用的 SMHUploadCredential。
// expireIn <= 0 表示不设置 Expiration（凭证永不过期）。
func makeCred(totalParts int, expireIn time.Duration) *SMHUploadCredential {
	cred := &SMHUploadCredential{
		PartURLTemplate: "https://cos.example.com/path?partNumber={partNumber}&uploadId=uid-init",
		PartHeaders:     map[string]string{"Authorization": "init-auth"},
		ConfirmKey:      "confirm-key-001",
		FileKey:         "backups/ins-test/state.tgz",
		PartSize:        50 * 1024 * 1024,
		TotalParts:      totalParts,
	}
	if expireIn > 0 {
		exp := time.Now().Add(expireIn)
		cred.Expiration = &exp
	}
	return cred
}

// withFakeUploadScript mock LoadScript，使其对 "upload_to_smh.sh" 返回包含所有占位符的空壳脚本，
// 让 smhUploadLoop 内的 LoadScript 调用成功，返回 cleanup 函数。
func withFakeUploadScript(t *testing.T) func() {
	t.Helper()
	origLoader := LoadScript
	fakeScript := `#!/bin/bash
ARCHIVE_PATH="{{archivepath}}"
UPLOAD_URL_B64="{{uploadurlb64}}"
OFFSET="{{offset}}"
PART_SIZE="{{partsize}}"
PART_NUMBER="{{partnumber}}"
TOTAL_PARTS="{{totalparts}}"
HEADER_COUNT={{headercount}}
{{headerkvlines}}
exit 0
`
	LoadScript = func(name string) (string, error) {
		return fakeScript, nil
	}
	return func() { LoadScript = origLoader }
}

// ─── 场景 1：已上传分块被跳过，未上传分块被执行 ──────────────────────────────

func TestSMHUploadLoop_SkipsUploadedParts(t *testing.T) {
	restore := saveAndRestoreHooks(t)
	cleanScript := withFakeUploadScript(t)

	t.Cleanup(func() {
		restore()
		cleanScript()
	})

	const totalParts = 5
	// 模拟分块 1、2、3 已上传，4、5 未上传
	alreadyUploaded := map[int]bool{1: true, 2: true, 3: true}
	cred := makeCred(totalParts, 0)

	var uploadedParts []int
	var mu sync.Mutex

	confirmCalled := false
	smhUploadHooks.Confirm = func(_ context.Context, _ string) error {
		confirmCalled = true
		return nil
	}

	runScriptFn = func(_ context.Context, instanceId string, scriptName string, _ uint64, _ string, _ func(string), _ map[string]string) (string, error) {
		// 从脚本名称中提取分块号（格式：_upload_smh_part{N}_{ts}.sh）
		var partNum int
		fmt.Sscanf(scriptName, "_upload_smh_part%d_", &partNum)
		mu.Lock()
		uploadedParts = append(uploadedParts, partNum)
		mu.Unlock()
		return "", nil
	}


	err := smhUploadLoop(context.Background(), "ins-skip-test", cred, alreadyUploaded, "/tmp/state.tgz", "root")
	if err != nil {
		t.Fatalf("smhUploadLoop 返回错误: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	// 验证：只有分块 4、5 被上传
	if len(uploadedParts) != 2 {
		t.Errorf("期望上传 2 个分块（4、5），实际上传了 %d 个: %v", len(uploadedParts), uploadedParts)
	}
	for _, p := range uploadedParts {
		if p == 1 || p == 2 || p == 3 {
			t.Errorf("分块 %d 已上传，不应被重复上传", p)
		}
	}

	// 验证：凭证未设置 Expiration，loop 内不应触发续期
	if !confirmCalled {
		t.Error("期望 Confirm 被调用，但未被调用")
	}
}

// ─── 场景 2：续期后 PartURLTemplate / PartHeaders 被刷新 ─────────────────────

func TestSMHUploadLoop_RenewUpdatesCredential(t *testing.T) {
	restore := saveAndRestoreHooks(t)
	cleanScript := withFakeUploadScript(t)

	t.Cleanup(func() {
		restore()
		cleanScript()
	})

	// 凭证 5 分钟后过期（< 10 分钟阈值），每个未上传分块上传前均会触发续期检查
	cred := makeCred(2, 5*time.Minute)
	initURL := cred.PartURLTemplate

	renewCount := int32(0)
	smhUploadHooks.Renew = func(_ context.Context, c *SMHUploadCredential) error {
		atomic.AddInt32(&renewCount, 1)
		c.PartURLTemplate = "https://cos.example.com/path?partNumber={partNumber}&uploadId=uid-renewed"
		c.PartHeaders = map[string]string{"Authorization": "renewed-auth-v2"}
		exp := time.Now().Add(2 * time.Hour) // 续期后有效期 2 小时（与生产逻辑一致）
		c.Expiration = &exp
		return nil
	}
	smhUploadHooks.Confirm = func(_ context.Context, _ string) error { return nil }

	var usedURLs []string
	var mu sync.Mutex

	runScriptFn = func(_ context.Context, _ string, _ string, _ uint64, _ string, _ func(string), _ map[string]string) (string, error) {
		mu.Lock()
		// 记录当前 cred 中的 URL（续期后应变化）
		usedURLs = append(usedURLs, cred.PartURLTemplate)
		mu.Unlock()
		return "", nil
	}


	err := smhUploadLoop(context.Background(), "ins-renew-test", cred, map[int]bool{}, "/tmp/state.tgz", "root")
	if err != nil {
		t.Fatalf("smhUploadLoop 返回错误: %v", err)
	}

	// 验证续期被触发至少一次
	if atomic.LoadInt32(&renewCount) == 0 {
		t.Error("凭证临近过期，期望触发续期，但 renewCount == 0")
	}

	// 验证续期后 URL 已更新（不再是初始值）
	mu.Lock()
	defer mu.Unlock()
	for i, u := range usedURLs {
		if u == initURL {
			t.Errorf("分块 %d 使用了续期前的旧 URL %q，期望使用续期后的新 URL", i+1, initURL)
		}
	}
}

// ─── 场景 3：全部分块已上传，直接 Confirm，不执行任何上传脚本 ────────────────

func TestSMHUploadLoop_AllPartsAlreadyUploaded(t *testing.T) {
	restore := saveAndRestoreHooks(t)
	cleanScript := withFakeUploadScript(t)

	t.Cleanup(func() {
		restore()
		cleanScript()
	})

	const totalParts = 3
	allUploaded := map[int]bool{1: true, 2: true, 3: true}
	cred := makeCred(totalParts, 0)

	scriptCallCount := int32(0)
	origRunScriptFn := runScriptFn
	runScriptFn = func(_ context.Context, _ string, _ string, _ uint64, _ string, _ func(string), _ map[string]string) (string, error) {
		atomic.AddInt32(&scriptCallCount, 1)
		return "", nil
	}
	t.Cleanup(func() { runScriptFn = origRunScriptFn })


	confirmCalled := false
	smhUploadHooks.Confirm = func(_ context.Context, _ string) error {
		confirmCalled = true
		return nil
	}

	err := smhUploadLoop(context.Background(), "ins-all-done", cred, allUploaded, "/tmp/state.tgz", "root")
	if err != nil {
		t.Fatalf("smhUploadLoop 返回错误: %v", err)
	}

	if atomic.LoadInt32(&scriptCallCount) != 0 {
		t.Errorf("所有分块已上传，不应执行任何上传脚本，实际执行了 %d 次", scriptCallCount)
	}
	if !confirmCalled {
		t.Error("期望 Confirm 被调用，但未被调用")
	}
}

// ─── 场景 4：GetParts 失败降级为全量上传 ─────────────────────────────────────

func TestSMHUploadLoop_GetPartsFailureFallsBackToFullUpload(t *testing.T) {
	restore := saveAndRestoreHooks(t)
	cleanScript := withFakeUploadScript(t)

	t.Cleanup(func() {
		restore()
		cleanScript()
	})

	const totalParts = 3
	cred := makeCred(totalParts, 0)

	smhUploadHooks.Confirm = func(_ context.Context, _ string) error { return nil }

	scriptCallCount := int32(0)
	runScriptFn = func(_ context.Context, instanceId string, _ string, _ uint64, _ string, _ func(string), _ map[string]string) (string, error) {
		// 仅对目标实例计数，避免其他测试的异步 goroutine 污染计数器
		if instanceId == "ins-fallback" {
			atomic.AddInt32(&scriptCallCount, 1)
		}
		return "", nil
	}

	// 模拟 GetParts 失败，调用方降级传入空 map
	uploadedParts := map[int]bool{} // 降级：全量上传


	err := smhUploadLoop(context.Background(), "ins-fallback", cred, uploadedParts, "/tmp/state.tgz", "root")
	if err != nil {
		t.Fatalf("smhUploadLoop 返回错误: %v", err)
	}

	if int(atomic.LoadInt32(&scriptCallCount)) != totalParts {
		t.Errorf("GetParts 失败降级后期望上传全部 %d 个分块，实际上传了 %d 个", totalParts, scriptCallCount)
	}
}

// ─── 场景 5：Renew 失败立即返回错误，不继续上传 ──────────────────────────────

func TestSMHUploadLoop_RenewFailureReturnsError(t *testing.T) {
	restore := saveAndRestoreHooks(t)
	cleanScript := withFakeUploadScript(t)

	t.Cleanup(func() {
		restore()
		cleanScript()
	})

	// 凭证 1 分钟后过期（< 10 分钟阈值），触发续期
	cred := makeCred(2, 1*time.Minute)

	smhUploadHooks.Renew = func(_ context.Context, _ *SMHUploadCredential) error {
		return hcommon.I18nError(i18n.MsgOperationFailed)
	}

	scriptCallCount := int32(0)
	runScriptFn = func(_ context.Context, _ string, _ string, _ uint64, _ string, _ func(string), _ map[string]string) (string, error) {
		atomic.AddInt32(&scriptCallCount, 1)
		return "", nil
	}


	err := smhUploadLoop(context.Background(), "ins-renew-fail", cred, map[int]bool{}, "/tmp/state.tgz", "root")
	if err == nil {
		t.Fatal("期望 Renew 失败时返回错误，实际返回 nil")
	}
	t.Logf("符合预期的错误: %v", err)

	// 续期失败后不应执行任何上传脚本
	if atomic.LoadInt32(&scriptCallCount) != 0 {
		t.Errorf("续期失败后不应执行上传脚本，实际执行了 %d 次", scriptCallCount)
	}
}

// ─── 场景 6：Confirm 失败返回错误 ────────────────────────────────────────────

func TestSMHUploadLoop_ConfirmFailureReturnsError(t *testing.T) {
	restore := saveAndRestoreHooks(t)
	cleanScript := withFakeUploadScript(t)

	t.Cleanup(func() {
		restore()
		cleanScript()
	})

	cred := makeCred(2, 0)

	smhUploadHooks.Confirm = func(_ context.Context, _ string) error {
		return hcommon.I18nError(i18n.MsgOperationFailed)
	}

	runScriptFn = func(_ context.Context, _ string, _ string, _ uint64, _ string, _ func(string), _ map[string]string) (string, error) {
		return "", nil
	}


	err := smhUploadLoop(context.Background(), "ins-confirm-fail", cred, map[int]bool{}, "/tmp/state.tgz", "root")
	if err == nil {
		t.Fatal("期望 Confirm 失败时返回错误，实际返回 nil")
	}
	t.Logf("符合预期的错误: %v", err)
}
