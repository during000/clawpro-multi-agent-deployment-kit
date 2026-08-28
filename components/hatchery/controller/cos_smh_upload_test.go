// cos_smh_upload_test.go
//
// 针对 PrepareSMHCommonUpload / RenewSMHCommonUpload / GetSMHCommonUploadParts
// 的单元测试，使用 httptest.Server 模拟 SMH API，完全隔离真实网络。
package controller

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"
)

// ============================================================================
// PrepareSMHCommonUpload 前置校验分支
// ============================================================================

// TestPrepareSMHCommonUpload_NotConfigured 覆盖 "SMH 未配置" 分支。
func TestPrepareSMHCommonUpload_NotConfigured(t *testing.T) {
	cleanup := setupCosCommonTestDB(t)
	defer cleanup()
	// 未 seed 任何 SMH 配置 → IsConfigured() 为 false
	_, err := PrepareSMHCommonUpload(context.Background(), "ins-1", "/tmp/state.tgz", 1024)
	if err == nil {
		t.Error("未配置时应返回 error")
	}
}

// TestPrepareSMHCommonUpload_CommonSpaceEmpty 覆盖 "SMH common 空间未配置" 分支。
func TestPrepareSMHCommonUpload_CommonSpaceEmpty(t *testing.T) {
	cleanup := setupCosCommonTestDB(t)
	defer cleanup()
	// 只写 SiteConfig，不写 SMHSpace（CommonSpace 为空）
	seedSMHNoCommonSpace(t)
	_, err := PrepareSMHCommonUpload(context.Background(), "ins-1", "/tmp/state.tgz", 1024)
	if err == nil {
		t.Error("common 空间未配置时应返回 error")
	}
}

// TestPrepareSMHCommonUpload_APIClientNil 覆盖 "SMH 客户端未初始化" 分支。
func TestPrepareSMHCommonUpload_APIClientNil(t *testing.T) {
	cleanup := setupCosCommonTestDB(t)
	defer cleanup()
	seedSMHFullyConfigured(t)
	// DB 中无 token，SMH 操作会触发自愈（但因无真实 SMH 服务而失败）
	_, err := PrepareSMHCommonUpload(context.Background(), "ins-1", "/tmp/state.tgz", 1024)
	if err == nil {
		t.Error("客户端未初始化时应返回 error")
	}
}

func TestPrepareSMHCommonUpload_InvalidArchiveSize(t *testing.T) {
	// 用 httptest.Server 返回 usage 响应（available=-1 表示无限制），
	// 让函数通过容量检查，再触发 archiveSize 校验。
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/usage/") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// capacity 不设置（null）表示无限制，availableSpace 同理（camelCase）
			w.Write([]byte(`[{"spaceId":"sp-common","size":"0"}]`))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	})
	_, cleanup := setupSMHHTTPTestEnv(t, h)
	t.Cleanup(cleanup)

	ctx := context.Background()
	_, err := PrepareSMHCommonUpload(ctx, "ins-1", "/tmp/state.tgz", 0)
	if err == nil {
		t.Error("archiveSize=0 时应返回 error")
	}
	wanted := hcommon.I18nError(i18n.MsgSmhInvalidArchiveSize).ErrorMessage(ctx)
	if !strings.Contains(hcommon.ErrorMessageWithCtx(ctx, err), wanted) {
		t.Errorf("错误信息应包含 %s，实际=%v", wanted, err)
	}
}

// TestPrepareSMHCommonUpload_InsufficientSpace 覆盖 "容量不足" 分支。
func TestPrepareSMHCommonUpload_InsufficientSpace(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/usage/") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// availableSpace=100 字节（camelCase），远小于 archiveSize=500
			w.Write([]byte(`[{"spaceId":"sp-common","size":"900","capacity":"1000","availableSpace":"100"}]`))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	})
	_, cleanup := setupSMHHTTPTestEnv(t, h)
	defer cleanup()

	_, err := PrepareSMHCommonUpload(context.Background(), "ins-1", "/tmp/state.tgz", 500)
	if err == nil {
		t.Error("容量不足时应返回 error")
	}
	if !strings.Contains(err.Error(), "容量不足") {
		t.Errorf("错误信息应包含 '容量不足'，实际=%v", err)
	}
}

// TestPrepareSMHCommonUpload_MultipartUploadFails 覆盖 "MultipartUploadFile API 失败" 分支。
func TestPrepareSMHCommonUpload_MultipartUploadFails(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/usage/") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[{"space_id":"sp-common","size":"0","capacity":"","available_space":""}]`))
			return
		}
		if strings.Contains(r.URL.Path, "/directory/") {
			// 创建目录成功（201 + path 字段，Content-Type 必须为 json）
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"path":["backups","ins-1"]}`))
			return
		}
		// MultipartUploadFile 返回 500
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"code":"InternalError","message":"server error"}`))
	})
	_, cleanup := setupSMHHTTPTestEnv(t, h)
	defer cleanup()

	_, err := PrepareSMHCommonUpload(context.Background(), "ins-1", "/tmp/state.tgz", 1024)
	if err == nil {
		t.Error("MultipartUploadFile 失败时应返回 error")
	}
}

// TestPrepareSMHCommonUpload_InstantUpload200 覆盖 "秒传成功（200）" 分支。
func TestPrepareSMHCommonUpload_InstantUpload200(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/usage/") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[{"spaceId":"sp-common","size":"0"}]`))
			return
		}
		if strings.Contains(r.URL.Path, "/directory/") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"path":["backups","ins-1"]}`))
			return
		}
		// MultipartUploadFile 返回 200 → 秒传成功
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"path":["/backups/ins-1/state.tgz"]}`)) // 200 响应 path 为数组
	})
	_, cleanup := setupSMHHTTPTestEnv(t, h)
	defer cleanup()

	cred, err := PrepareSMHCommonUpload(context.Background(), "ins-1", "/tmp/state.tgz", 1024)
	if err != nil {
		t.Fatalf("秒传成功路径不应返回 error，实际=%v", err)
	}
	if cred == nil {
		t.Fatal("cred 不应为 nil")
	}
	// 秒传时 PartURLTemplate 为空，ConfirmKey 为空
	if cred.PartURLTemplate != "" {
		t.Errorf("秒传时 PartURLTemplate 应为空，实际=%q", cred.PartURLTemplate)
	}
	if cred.FileKey == "" {
		t.Error("秒传时 FileKey 不应为空")
	}
}

// TestPrepareSMHCommonUpload_MultipartUpload201 覆盖 "分块上传凭证获取成功（201）" 主流程。
func TestPrepareSMHCommonUpload_MultipartUpload201(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/usage/") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[{"spaceId":"sp-common","size":"0"}]`))
			return
		}
		if strings.Contains(r.URL.Path, "/directory/") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"path":["backups","ins-1"]}`))
			return
		}
		// MultipartUploadFile 返回 201 → 分块上传（字段名 camelCase）
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{
			"domain": "cos.example.com",
			"path": "/backups/ins-1/state.tgz",
			"uploadId": "upload-id-001",
			"confirmKey": "confirm-key-001",
			"headers": {"Authorization": "q-sign-algorithm=sha1&q-sign-time=1234567890;1234575890"},
			"expiration": "2099-01-01T00:00:00Z"
		}`))
	})
	_, cleanup := setupSMHHTTPTestEnv(t, h)
	defer cleanup()

	cred, err := PrepareSMHCommonUpload(context.Background(), "ins-1", "/tmp/state.tgz", 1024)
	if err != nil {
		t.Fatalf("分块上传凭证获取不应失败，实际=%v", err)
	}
	if cred == nil {
		t.Fatal("cred 不应为 nil")
	}
	if cred.ConfirmKey != "confirm-key-001" {
		t.Errorf("ConfirmKey 期望 'confirm-key-001'，实际=%q", cred.ConfirmKey)
	}
	if !strings.Contains(cred.PartURLTemplate, "upload-id-001") {
		t.Errorf("PartURLTemplate 应包含 uploadId，实际=%q", cred.PartURLTemplate)
	}
	if !strings.Contains(cred.PartURLTemplate, "{partNumber}") {
		t.Errorf("PartURLTemplate 应包含 {partNumber} 占位符，实际=%q", cred.PartURLTemplate)
	}
	if cred.TotalParts <= 0 {
		t.Errorf("TotalParts 应 > 0，实际=%d", cred.TotalParts)
	}
	if cred.Expiration == nil {
		t.Error("Expiration 不应为 nil")
	}
}

// TestPrepareSMHCommonUpload_201IncompleteResponse 覆盖 "201 返回数据不完整" 分支。
func TestPrepareSMHCommonUpload_201IncompleteResponse(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/usage/") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[{"spaceId":"sp-common","size":"0"}]`))
			return
		}
		if strings.Contains(r.URL.Path, "/directory/") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"path":["backups","ins-1"]}`))
			return
		}
		// 返回 201 但缺少必要字段（domain/path/confirm_key/upload_id 均缺失）
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{}`))
	})
	_, cleanup := setupSMHHTTPTestEnv(t, h)
	defer cleanup()

	_, err := PrepareSMHCommonUpload(context.Background(), "ins-1", "/tmp/state.tgz", 1024)
	if err == nil {
		t.Error("201 返回数据不完整时应返回 error")
	}
}

func TestPrepareMigrationUpload_UsesOverwriteConflictStrategy(t *testing.T) {
	sawUploadRequest := false
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/directory/") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"path":["migrations","ins-test"]}`))
			return
		}

		sawUploadRequest = true
		if got := r.URL.Query().Get("conflict_resolution_strategy"); got != "overwrite" {
			t.Errorf("conflict_resolution_strategy should be overwrite, got %q", got)
		}
		if got := r.URL.Query().Get("multipart"); got != "1" {
			t.Errorf("multipart should be 1, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{
			"domain": "cos.example.com",
			"path": "/migrations/ins-test/agent-export.tgz",
			"uploadId": "migration-upload-id",
			"confirmKey": "migration-confirm-key",
			"headers": {"Authorization": "q-sign-algorithm=sha1&q-sign-time=1234567890;1234575890"},
			"expiration": "2099-01-01T00:00:00Z"
		}`))
	})
	_, cleanup := setupSMHHTTPTestEnv(t, h)
	defer cleanup()

	cred, err := PrepareMigrationUpload(context.Background(), "migrations/ins-test/agent-export.tgz", 1024)
	if err != nil {
		t.Fatalf("migration upload credential should be prepared, got %v", err)
	}
	if !sawUploadRequest {
		t.Fatal("expected multipart upload request")
	}
	if cred.ConfirmKey != "migration-confirm-key" {
		t.Errorf("ConfirmKey mismatch: %q", cred.ConfirmKey)
	}
}

// TestPrepareSMHCommonUpload_CapacityUnlimited 覆盖 "available=-1（无限制）跳过容量检查" 分支。
func TestPrepareSMHCommonUpload_CapacityUnlimited(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/usage/") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// availableSpace 不设置 → NullableString.IsSet()=false → available=-1 → 无限制
			w.Write([]byte(`[{"spaceId":"sp-common","size":"0"}]`))
			return
		}
		if strings.Contains(r.URL.Path, "/directory/") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"path":["backups","ins-1"]}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{
			"domain": "cos.example.com",
			"path": "/backups/ins-1/state.tgz",
			"uploadId": "uid-unlimited",
			"confirmKey": "ck-unlimited",
			"expiration": "2099-01-01T00:00:00Z"
		}`))
	})
	_, cleanup := setupSMHHTTPTestEnv(t, h)
	defer cleanup()

	cred, err := PrepareSMHCommonUpload(context.Background(), "ins-1", "/tmp/state.tgz", 100*1024*1024)
	if err != nil {
		t.Fatalf("无限制容量时不应失败，实际=%v", err)
	}
	if cred.ConfirmKey != "ck-unlimited" {
		t.Errorf("ConfirmKey 期望 'ck-unlimited'，实际=%q", cred.ConfirmKey)
	}
}

// ============================================================================
// RenewSMHCommonUpload 前置校验分支
// ============================================================================

// TestRenewSMHCommonUpload_CredNil 覆盖 "cred 为 nil" 分支。
func TestRenewSMHCommonUpload_CredNil(t *testing.T) {
	err := RenewSMHCommonUpload(context.Background(), nil)
	if err == nil {
		t.Error("cred=nil 时应返回 error")
	}
}

// TestRenewSMHCommonUpload_ConfirmKeyEmpty 覆盖 "ConfirmKey 为空" 分支。
func TestRenewSMHCommonUpload_ConfirmKeyEmpty(t *testing.T) {
	err := RenewSMHCommonUpload(context.Background(), &SMHUploadCredential{ConfirmKey: ""})
	if err == nil {
		t.Error("ConfirmKey 为空时应返回 error")
	}
}

// TestRenewSMHCommonUpload_NotConfigured 覆盖 "SMH 未配置" 分支。
func TestRenewSMHCommonUpload_NotConfigured(t *testing.T) {
	cleanup := setupCosCommonTestDB(t)
	defer cleanup()
	err := RenewSMHCommonUpload(context.Background(), &SMHUploadCredential{ConfirmKey: "ck-001"})
	if err == nil {
		t.Error("未配置时应返回 error")
	}
}

// TestRenewSMHCommonUpload_CommonSpaceEmpty 覆盖 "SMH common 空间未配置" 分支。
func TestRenewSMHCommonUpload_CommonSpaceEmpty(t *testing.T) {
	cleanup := setupCosCommonTestDB(t)
	defer cleanup()
	seedSMHNoCommonSpace(t)
	err := RenewSMHCommonUpload(context.Background(), &SMHUploadCredential{ConfirmKey: "ck-001"})
	if err == nil {
		t.Error("common 空间未配置时应返回 error")
	}
}

// TestRenewSMHCommonUpload_APIClientNil 覆盖 "SMH 客户端未初始化" 分支。
func TestRenewSMHCommonUpload_APIClientNil(t *testing.T) {
	cleanup := setupCosCommonTestDB(t)
	defer cleanup()
	seedSMHFullyConfigured(t)
	err := RenewSMHCommonUpload(context.Background(), &SMHUploadCredential{ConfirmKey: "ck-001"})
	if err == nil {
		t.Error("客户端未初始化时应返回 error")
	}
}

// TestRenewSMHCommonUpload_APIFails 覆盖 "RenewMultipartUpload API 失败" 分支。
func TestRenewSMHCommonUpload_APIFails(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"code":"InternalError","message":"renew failed"}`))
	})
	_, cleanup := setupSMHHTTPTestEnv(t, h)
	defer cleanup()

	err := RenewSMHCommonUpload(context.Background(), &SMHUploadCredential{ConfirmKey: "ck-001"})
	if err == nil {
		t.Error("API 失败时应返回 error")
	}
}

// TestRenewSMHCommonUpload_IncompleteResponse 覆盖 "返回数据不完整" 分支。
func TestRenewSMHCommonUpload_IncompleteResponse(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// domain/path/upload_id 均缺失
		w.Write([]byte(`{}`))
	})
	_, cleanup := setupSMHHTTPTestEnv(t, h)
	defer cleanup()

	err := RenewSMHCommonUpload(context.Background(), &SMHUploadCredential{ConfirmKey: "ck-001"})
	if err == nil {
		t.Error("返回数据不完整时应返回 error")
	}
}

// TestRenewSMHCommonUpload_ExpirationNil 覆盖 "expiration 为 nil" 分支。
func TestRenewSMHCommonUpload_ExpirationNil(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// 有 domain/path/upload_id，但无 expiration
		w.Write([]byte(`{
			"domain": "cos.example.com",
			"path": "/backups/ins-1/state.tgz",
			"uploadId": "uid-renew-001"
		}`))
	})
	_, cleanup := setupSMHHTTPTestEnv(t, h)
	defer cleanup()

	err := RenewSMHCommonUpload(context.Background(), &SMHUploadCredential{ConfirmKey: "ck-001"})
	if err == nil {
		t.Error("expiration 为 nil 时应返回 error")
	}
}

// TestRenewSMHCommonUpload_Success 覆盖 "续期成功" 主流程，验证 cred 字段被正确更新。
func TestRenewSMHCommonUpload_Success(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"domain": "cos.example.com",
			"path": "/backups/ins-1/state.tgz",
			"uploadId": "uid-renewed-001",
			"headers": {"Authorization": "renewed-auth"},
			"expiration": "2099-06-01T00:00:00Z"
		}`))
	})
	_, cleanup := setupSMHHTTPTestEnv(t, h)
	defer cleanup()

	cred := &SMHUploadCredential{
		ConfirmKey:      "ck-001",
		PartURLTemplate: "https://old.example.com?partNumber={partNumber}&uploadId=old",
		PartHeaders:     map[string]string{"Authorization": "old-auth"},
	}
	oldURL := cred.PartURLTemplate

	if err := RenewSMHCommonUpload(context.Background(), cred); err != nil {
		t.Fatalf("续期成功路径不应返回 error，实际=%v", err)
	}

	// 验证 PartURLTemplate 已更新
	if cred.PartURLTemplate == oldURL {
		t.Error("续期后 PartURLTemplate 应已更新，但仍为旧值")
	}
	if !strings.Contains(cred.PartURLTemplate, "uid-renewed-001") {
		t.Errorf("续期后 PartURLTemplate 应包含新 uploadId，实际=%q", cred.PartURLTemplate)
	}
	// 验证 PartHeaders 已更新
	if cred.PartHeaders["Authorization"] != "renewed-auth" {
		t.Errorf("续期后 Authorization 期望 'renewed-auth'，实际=%q", cred.PartHeaders["Authorization"])
	}
	// 验证 Expiration 已更新
	if cred.Expiration == nil {
		t.Error("续期后 Expiration 不应为 nil")
	} else if cred.Expiration.Before(time.Now()) {
		t.Errorf("续期后 Expiration 应在未来，实际=%v", cred.Expiration)
	}
}

// ============================================================================
// GetSMHCommonUploadParts 前置校验分支
// ============================================================================

// TestGetSMHCommonUploadParts_ConfirmKeyEmpty 覆盖 "confirmKey 为空" 分支。
func TestGetSMHCommonUploadParts_ConfirmKeyEmpty(t *testing.T) {
	_, err := GetSMHCommonUploadParts(context.Background(), "")
	if err == nil {
		t.Error("confirmKey 为空时应返回 error")
	}
}

// TestGetSMHCommonUploadParts_NotConfigured 覆盖 "SMH 未配置" 分支。
func TestGetSMHCommonUploadParts_NotConfigured(t *testing.T) {
	cleanup := setupCosCommonTestDB(t)
	defer cleanup()
	_, err := GetSMHCommonUploadParts(context.Background(), "ck-001")
	if err == nil {
		t.Error("未配置时应返回 error")
	}
}

// TestGetSMHCommonUploadParts_CommonSpaceEmpty 覆盖 "SMH common 空间未配置" 分支。
func TestGetSMHCommonUploadParts_CommonSpaceEmpty(t *testing.T) {
	cleanup := setupCosCommonTestDB(t)
	defer cleanup()
	seedSMHNoCommonSpace(t)
	_, err := GetSMHCommonUploadParts(context.Background(), "ck-001")
	if err == nil {
		t.Error("common 空间未配置时应返回 error")
	}
}

// TestGetSMHCommonUploadParts_APIClientNil 覆盖 "SMH 客户端未初始化" 分支。
func TestGetSMHCommonUploadParts_APIClientNil(t *testing.T) {
	cleanup := setupCosCommonTestDB(t)
	defer cleanup()
	seedSMHFullyConfigured(t)
	_, err := GetSMHCommonUploadParts(context.Background(), "ck-001")
	if err == nil {
		t.Error("客户端未初始化时应返回 error")
	}
}

// TestGetSMHCommonUploadParts_APIFails_FallbackEmpty 覆盖 "API 失败降级为空集合" 分支。
func TestGetSMHCommonUploadParts_APIFails_FallbackEmpty(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"code":"NotFound","message":"upload task not found"}`))
	})
	_, cleanup := setupSMHHTTPTestEnv(t, h)
	defer cleanup()

	parts, err := GetSMHCommonUploadParts(context.Background(), "ck-not-exist")
	// API 失败时降级返回空集合，err 应为 nil
	if err != nil {
		t.Errorf("API 失败时应降级返回空集合（err=nil），实际 err=%v", err)
	}
	if len(parts) != 0 {
		t.Errorf("降级时应返回空集合，实际=%v", parts)
	}
}

// TestGetSMHCommonUploadParts_Success 覆盖 "查询成功，返回已上传分块" 主流程。
func TestGetSMHCommonUploadParts_Success(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// 已上传分块 1、2、3（PartNumber 大写，与 SDK 结构体一致）
		w.Write([]byte(`{
			"parts": [
				{"PartNumber": 1, "Size": 52428800},
				{"PartNumber": 2, "Size": 52428800},
				{"PartNumber": 3, "Size": 10240}
			]
		}`))
	})
	_, cleanup := setupSMHHTTPTestEnv(t, h)
	defer cleanup()

	parts, err := GetSMHCommonUploadParts(context.Background(), "ck-001")
	if err != nil {
		t.Fatalf("查询成功路径不应返回 error，实际=%v", err)
	}
	if len(parts) != 3 {
		t.Errorf("期望 3 个已上传分块，实际=%d", len(parts))
	}
	for _, n := range []int{1, 2, 3} {
		if !parts[n] {
			t.Errorf("分块 %d 应在已上传集合中", n)
		}
	}
}

// TestGetSMHCommonUploadParts_EmptyParts 覆盖 "查询成功但无已上传分块" 分支。
func TestGetSMHCommonUploadParts_EmptyParts(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"parts": []}`))
	})
	_, cleanup := setupSMHHTTPTestEnv(t, h)
	defer cleanup()

	parts, err := GetSMHCommonUploadParts(context.Background(), "ck-empty")
	if err != nil {
		t.Fatalf("空分块列表不应返回 error，实际=%v", err)
	}
	if len(parts) != 0 {
		t.Errorf("空分块列表应返回空集合，实际=%v", parts)
	}
}

// ============================================================================
// 辅助函数
// ============================================================================

// seedSMHNoCommonSpace 写入 SiteConfig（Endpoint/LibraryId/LibrarySecret 均有值），
// 但不写 SMHSpace，使 GetSMHConfig().CommonSpace 为空。
func seedSMHNoCommonSpace(t *testing.T) {
	t.Helper()
	if err := model.DB(context.Background()).Create(&model.SiteConfig{
		SMHEnabled:       1,
		SMHEndpoint:      "https://smh.example.com",
		SMHLibraryId:     "lib-test",
		SMHLibrarySecret: "secret-test",
	}).Error; err != nil {
		t.Fatalf("写入 SiteConfig 失败: %v", err)
	}
	// 故意不写 SMHSpace，使 CommonSpace 为空
}
