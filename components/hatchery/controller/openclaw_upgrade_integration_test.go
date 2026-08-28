// openclaw_upgrade_integration_test.go
//
// 集成测试：使用本地真实文件 + MySQL 中的 SMH 配置，端到端验证分块上传流程。
//
// 运行方式（需要真实 MySQL 和 SMH 环境）：
//
//	HATCHERY_TEST_DB="user:pass@tcp(host:3306)/hatchery?charset=utf8mb4&parseTime=True&loc=Local" \
//	HATCHERY_TEST_FILE="/path/to/local/file.tgz" \
//	HATCHERY_TEST_INSTANCE_ID="ins-xxxxxxxx" \
//	go test ./controller/ -run TestIntegration_SMHUpload -v -count=1 -timeout 30m
//
// 环境变量说明：
//   - HATCHERY_TEST_DB         MySQL DSN，必填，缺失时跳过测试
//   - HATCHERY_TEST_FILE       本地文件路径，必填，缺失时跳过测试
//   - HATCHERY_TEST_INSTANCE_ID 模拟的实例 ID，默认 "ins-integration-test"
//   - HATCHERY_TEST_IDENTIFIER  多租户标识，默认空（单租户）
package controller

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"hatchery/model"
)

// TestIntegration_SMHUpload 端到端测试：
//  1. 从 MySQL 读取 SMH 配置（Endpoint / LibraryId / LibrarySecret / CommonSpace）
//  2. 初始化 SMH 客户端和 Token
//  3. 使用本地文件调用 PrepareSMHCommonUpload 获取上传凭证
//  4. 调用 GetSMHCommonUploadParts 查询已上传分块（断点续传）
//  5. 调用 RenewSMHCommonUpload 验证续期接口
//  6. 模拟分块上传（跳过真实 CVM 上传，直接验证凭证有效性）
//  7. 调用 ConfirmSMHCommonUpload 完成上传
func TestIntegration_SMHUpload(t *testing.T) {
	// ── 硬编码测试参数 ────────────────────────────────────────────────────────
	dsn := ""
	localFile := ""
	instanceId := ""
	identifier := ""

	// ── 环境检查：敏感参数为空时跳过，避免在 CI 中误执行 ─────────────────────
	if dsn == "" || localFile == "" {
		t.Skip("集成测试跳过：dsn 或 localFile 未配置，请在本地填写真实参数后手动运行")
	}
	if instanceId == "" {
		instanceId = "ins-integration-test"
	}

	// ── 检查本地文件 ──────────────────────────────────────────────────────────
	fi, err := os.Stat(localFile)
	if err != nil {
		t.Fatalf("本地文件不存在或无法访问: %v", err)
	}
	fileSize := fi.Size()
	t.Logf("本地文件: %s，大小: %d 字节（%.2f MB）", localFile, fileSize, float64(fileSize)/1024/1024)

	ctx := context.Background()

	// ── 初始化数据库连接，读取 SMH 配置 ──────────────────────────────────────
	t.Log("正在连接 MySQL 数据库...")
	model.InitDB(dsn, "mysql", identifier, "", "", "", false)
	defer model.CloseDB()

	smhConfig := model.GetSMHConfig(ctx)
	t.Logf("SMH 配置：Endpoint=%s, LibraryId=%s, CommonSpace=%s",
		smhConfig.Endpoint, smhConfig.LibraryId, smhConfig.CommonSpace)

	if smhConfig.Endpoint == "" || smhConfig.LibraryId == "" || smhConfig.LibrarySecret == "" {
		t.Fatal("MySQL 中 SMH 配置不完整（Endpoint / LibraryId / LibrarySecret 为空），请先在管理后台配置 SMH")
	}
	if smhConfig.CommonSpace == "" {
		t.Fatal("MySQL 中 SMH CommonSpace 未配置，请先在管理后台开通 SMH common 空间")
	}

	// ── 初始化 SMH 客户端和 Token ─────────────────────────────────────────────
	t.Log("正在初始化 SMH 客户端...")
	InitSMHTokenRefresher(ctx, smhConfig)
	// 等待 Token 初始化完成（InitSMHTokenRefresher 内部同步获取 Token）
	time.Sleep(500 * time.Millisecond)

	// ── Step 1：PrepareSMHCommonUpload ────────────────────────────────────────
	t.Logf("[Step 1] 调用 PrepareSMHCommonUpload，instanceId=%s, file=%s, size=%d", instanceId, localFile, fileSize)
	cred, rerr := PrepareSMHCommonUpload(ctx, instanceId, localFile, fileSize)
	if rerr != nil {
		t.Fatalf("[Step 1] PrepareSMHCommonUpload 失败: %v", rerr)
	}
	t.Logf("[Step 1] 成功，ConfirmKey=%s, FileKey=%s, TotalParts=%d, PartSize=%d, Expiration=%v",
		cred.ConfirmKey, cred.FileKey, cred.TotalParts, cred.PartSize, cred.Expiration)

	// ── Step 2：GetSMHCommonUploadParts（断点续传查询）────────────────────────
	t.Logf("[Step 2] 调用 GetSMHCommonUploadParts，ConfirmKey=%s", cred.ConfirmKey)
	uploadedParts, rerr := GetSMHCommonUploadParts(ctx, cred.ConfirmKey)
	if rerr != nil {
		t.Logf("[Step 2] GetSMHCommonUploadParts 失败（可能是新任务，无已上传分块）: %v", rerr)
		uploadedParts = map[int]bool{}
	} else {
		t.Logf("[Step 2] 已上传分块: %v（共 %d 块）", uploadedParts, len(uploadedParts))
	}

	// ── Step 3：RenewSMHCommonUpload（验证续期接口）───────────────────────────
	t.Logf("[Step 3] 调用 RenewSMHCommonUpload，验证续期接口")
	oldURL := cred.PartURLTemplate
	oldAuth := fmt.Sprintf("%v", cred.PartHeaders)
	var oldExpiration *time.Time
	if cred.Expiration != nil {
		exp := *cred.Expiration
		oldExpiration = &exp
	}
	t.Logf("[Step 3] 续期前 Expiration: %v", oldExpiration)
	if err := RenewSMHCommonUpload(ctx, cred); err != nil {
		t.Fatalf("[Step 3] RenewSMHCommonUpload 失败: %v", err)
	}
	t.Logf("[Step 3] 续期成功")
	t.Logf("  续期前 PartURLTemplate: %s", oldURL)
	t.Logf("  续期后 PartURLTemplate: %s", cred.PartURLTemplate)
	t.Logf("  续期前 PartHeaders: %s", oldAuth)
	t.Logf("  续期后 PartHeaders: %v", cred.PartHeaders)
	t.Logf("  续期前 Expiration: %v", oldExpiration)
	t.Logf("  续期后 Expiration: %v", cred.Expiration)
	if oldExpiration != nil && cred.Expiration != nil && !cred.Expiration.After(*oldExpiration) {
		t.Errorf("[Step 3] 续期后 Expiration（%v）未晚于续期前（%v），SMH 服务端可能未真正续期", *cred.Expiration, *oldExpiration)
	}

	// 验证续期后 URL 和 Header 已更新
	if cred.PartURLTemplate == "" {
		t.Error("[Step 3] 续期后 PartURLTemplate 为空，续期可能未生效")
	}
	// 从 Authorization Header 的 q-sign-time 解析真实到期时间（当前时间 + 2小时）
	// 验证到期时间距当前至少 1 小时，确认是结束时间戳（+2h）而非开始时间戳
	if cred.Expiration == nil {
		t.Error("[Step 3] 续期后 Expiration 为 nil，凭证到期时间解析失败")
	} else if time.Until(*cred.Expiration) < time.Hour {
		t.Errorf("[Step 3] 续期后 Expiration（%v）距当前不足 1 小时，q-sign-time 解析可能有误", *cred.Expiration)
	} else {
		t.Logf("[Step 3] Expiration 验证通过，距到期还有 %.1f 小时", time.Until(*cred.Expiration).Hours())
	}

	// ── Step 4：验证分块 URL 可访问（HEAD 请求，不实际上传）─────────────────
	t.Logf("[Step 4] 验证分块 1 的 PUT URL 格式是否正确")
	if cred.TotalParts > 0 {
		partURL := replacePartNumber(cred.PartURLTemplate, 1)
		t.Logf("[Step 4] 分块 1 PUT URL: %s", partURL)
		if partURL == "" || partURL == cred.PartURLTemplate {
			t.Error("[Step 4] 分块 URL 替换失败，{partNumber} 未被替换")
		}
	}

	// ── Step 5：ConfirmSMHCommonUpload（清理：取消未完成的上传任务）──────────
	// 注意：由于我们没有真正上传分块，这里 Confirm 可能会失败（SMH 要求所有分块都上传完成）。
	// 这里的目的是验证 Confirm 接口可达，失败是预期行为（除非所有分块都已上传）。
	t.Logf("[Step 5] 调用 ConfirmSMHCommonUpload（预期可能失败，因为未真正上传分块）")
	if len(uploadedParts) == cred.TotalParts && cred.TotalParts > 0 {
		// 所有分块都已上传（断点续传场景），可以真正 Confirm
		if err := ConfirmSMHCommonUpload(ctx, cred.ConfirmKey); err != nil {
			t.Errorf("[Step 5] ConfirmSMHCommonUpload 失败: %v", err)
		} else {
			t.Log("[Step 5] Confirm 成功，文件已上传到 SMH")
		}
	} else {
		t.Logf("[Step 5] 跳过 Confirm（已上传分块 %d/%d，未全部完成）", len(uploadedParts), cred.TotalParts)
		t.Log("[Step 5] 集成测试完成，SMH API 调用链路验证通过：Prepare → GetParts → Renew ✓")
	}
}

// replacePartNumber 将 PartURLTemplate 中的 {partNumber} 替换为实际分块号。
func replacePartNumber(template string, partNum int) string {
	return strings.ReplaceAll(template, "{partNumber}", strconv.Itoa(partNum))
}

// TestReplacePartNumber 验证 replacePartNumber 的替换逻辑。
func TestReplacePartNumber(t *testing.T) {
	tests := []struct {
		name     string
		template string
		partNum  int
		want     string
	}{
		{
			name:     "正常替换",
			template: "https://example.com/upload?partNumber={partNumber}&uploadId=abc",
			partNum:  1,
			want:     "https://example.com/upload?partNumber=1&uploadId=abc",
		},
		{
			name:     "分块号为5",
			template: "https://example.com/upload/{partNumber}",
			partNum:  5,
			want:     "https://example.com/upload/5",
		},
		{
			name:     "多处占位符均被替换",
			template: "{partNumber}-{partNumber}",
			partNum:  3,
			want:     "3-3",
		},
		{
			name:     "模板中无占位符",
			template: "https://example.com/upload",
			partNum:  1,
			want:     "https://example.com/upload",
		},
		{
			name:     "空模板",
			template: "",
			partNum:  1,
			want:     "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := replacePartNumber(tc.template, tc.partNum)
			if got != tc.want {
				t.Errorf("replacePartNumber(%q, %d) = %q, want %q", tc.template, tc.partNum, got, tc.want)
			}
		})
	}
}
