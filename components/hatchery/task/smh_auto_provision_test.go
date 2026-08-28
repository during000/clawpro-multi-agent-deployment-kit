package task

import (
	"context"
	"errors"
	"testing"

	hcommon "hatchery/common"
	"hatchery/controller"
	"hatchery/i18n"
	"hatchery/model"
)

// ---------- mock 辅助 ----------

// mockSMHProvisionDeps 实现 smhProvisionDeps 接口，用于测试。
type mockSMHProvisionDeps struct {
	getSiteConfig              func(ctx context.Context) model.SiteConfig
	updateSiteConfig           func(ctx context.Context, updates interface{}) error
	provisionSMH               func(ctx context.Context) error
	ensureLibrarySearchEnabled func() error
	startDefaultBundleSMHSync  func(ctx context.Context)
}

func (m *mockSMHProvisionDeps) GetSiteConfig(ctx context.Context) model.SiteConfig {
	return m.getSiteConfig(ctx)
}
func (m *mockSMHProvisionDeps) UpdateSiteConfig(ctx context.Context, updates interface{}) error {
	if m.updateSiteConfig != nil {
		return m.updateSiteConfig(ctx, updates)
	}
	return nil
}
func (m *mockSMHProvisionDeps) ProvisionSMH(ctx context.Context) error {
	if m.provisionSMH != nil {
		return m.provisionSMH(ctx)
	}
	return nil
}
func (m *mockSMHProvisionDeps) EnsureLibrarySearchEnabled(ctx context.Context) error {
	if m.ensureLibrarySearchEnabled != nil {
		return m.ensureLibrarySearchEnabled()
	}
	return nil
}
func (m *mockSMHProvisionDeps) StartDefaultBundleSMHSync(ctx context.Context) {
	if m.startDefaultBundleSMHSync != nil {
		m.startDefaultBundleSMHSync(ctx)
	}
}

// newTestTask 创建一个使用 mock 依赖的测试任务实例。
func newTestTask(mock *mockSMHProvisionDeps) *smhAutoProvisionTask {
	return newSMHAutoProvisionTask(mock)
}

// =====================================================================
// safeSMHProvision 测试
// =====================================================================

// ---------- Test: 已开通 + EnsureSearch 成功 → 直接返回 nil ----------

func TestSafeSMHProvision_AlreadyEnabled(t *testing.T) {
	task := newTestTask(&mockSMHProvisionDeps{
		getSiteConfig:              func(ctx context.Context) model.SiteConfig { return model.SiteConfig{SMHEnabled: 1} },
		ensureLibrarySearchEnabled: func() error { return nil },
	})

	err := task.safeSMHProvision(context.Background())
	if err != nil {
		t.Errorf("已开通场景期望 nil，实际=%v", err)
	}
}

// ---------- Test: 已开通 + EnsureSearch 失败 → 返回错误 ----------

func TestSafeSMHProvision_AlreadyEnabled_SearchFail(t *testing.T) {
	task := newTestTask(&mockSMHProvisionDeps{
		getSiteConfig:              func(ctx context.Context) model.SiteConfig { return model.SiteConfig{SMHEnabled: 1} },
		ensureLibrarySearchEnabled: func() error { return errors.New("ModifyLibrary failed") },
	})

	err := task.safeSMHProvision(context.Background())
	if err == nil {
		t.Error("EnsureSearch 失败时期望返回错误")
	}
}

// ---------- Test: 未开通 + ProvisionSMH 成功 + EnsureSearch 成功 → nil ----------

func TestSafeSMHProvision_ProvisionSuccess(t *testing.T) {
	provisionCalled := false
	searchCalled := false

	task := newTestTask(&mockSMHProvisionDeps{
		getSiteConfig: func(ctx context.Context) model.SiteConfig { return model.SiteConfig{SMHEnabled: 0} },
		provisionSMH: func(ctx context.Context) error {
			provisionCalled = true
			return nil
		},
		ensureLibrarySearchEnabled: func() error {
			searchCalled = true
			return nil
		},
	})

	err := task.safeSMHProvision(context.Background())
	if err != nil {
		t.Errorf("开通成功场景期望 nil，实际=%v", err)
	}
	if !provisionCalled {
		t.Error("期望调用 provisionSMH")
	}
	if !searchCalled {
		t.Error("期望调用 ensureLibrarySearchEnabled")
	}
}

// ---------- Test: 未开通 + ProvisionSMH 失败 → 持久化错误码到 SiteConfig ----------
// 使用真实生产日志中的错误信息

func TestSafeSMHProvision_ProvisionFail_PersistError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode string
	}{
		{
			name: "真实日志：余额不足 (BalanceLess)",
			err: errors.New(
				`CreateLibrary: [TencentCloudSDKError] Code=UnsupportedOperation.BalanceLess, ` +
					`Message=margin block balance less error[[ERR_INSUFFICIENT_BALANCE]balance less error:` +
					`call order-center error:uin:100000679786 The total balance -2774 is less than the frozen amount 0]` +
					`[seqId:q4qtal1m-zreg-ttkd-qrio-03adcef94314], RequestId=13383cd0-8fe1-4797-aed1-a465ec676d19`),
			expectedCode: "INSUFFICIENT_BALANCE",
		},
		{
			name: "真实日志：STS 角色不存在",
			err: errors.New(
				`create SMH cloud client: STS 临时密钥刷新失败: AssumeRole 失败: ` +
					`[TencentCloudSDKError] Code=InternalError.GetRoleError, Message=role not exist, ` +
					`RequestId=20d4c6fe-4c01-48a0-a4f5-de949e533def`),
			expectedCode: "STS_ROLE_NOT_FOUND",
		},
		{
			name:         "网络错误",
			err:          errors.New("dial tcp 10.0.0.1:443: i/o timeout"),
			expectedCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var persistedCode string

			task := newTestTask(&mockSMHProvisionDeps{
				getSiteConfig: func(ctx context.Context) model.SiteConfig { return model.SiteConfig{SMHEnabled: 0} },
				provisionSMH: func(ctx context.Context) error {
					return tt.err
				},
				updateSiteConfig: func(ctx context.Context, updates interface{}) error {
					m := updates.(map[string]interface{})
					persistedCode = m["smh_provision_error"].(string)
					return nil
				},
			})

			err := task.safeSMHProvision(context.Background())
			if err == nil {
				t.Fatal("ProvisionSMH 失败时期望返回错误")
			}
			if persistedCode != tt.expectedCode {
				t.Errorf("持久化的错误码 = %q, 期望 %q", persistedCode, tt.expectedCode)
			}
		})
	}
}

// ---------- Test: 未开通 + ProvisionSMH 成功 + EnsureSearch 失败 → 返回错误 ----------

func TestSafeSMHProvision_ProvisionOK_SearchFail(t *testing.T) {
	task := newTestTask(&mockSMHProvisionDeps{
		getSiteConfig: func(ctx context.Context) model.SiteConfig { return model.SiteConfig{SMHEnabled: 0} },
		provisionSMH:  func(ctx context.Context) error { return nil },
		ensureLibrarySearchEnabled: func() error {
			return errors.New("ModifyLibrary to enable search: network error")
		},
	})

	err := task.safeSMHProvision(context.Background())
	if err == nil {
		t.Error("EnsureSearch 失败时期望返回错误")
	}
}

// ---------- Test: panic 恢复 → 返回错误而非 crash ----------

func TestSafeSMHProvision_PanicRecovery(t *testing.T) {
	task := newTestTask(&mockSMHProvisionDeps{
		getSiteConfig: func(ctx context.Context) model.SiteConfig { panic("unexpected nil pointer") },
	})

	err := task.safeSMHProvision(context.Background())
	if err == nil {
		t.Fatal("panic 场景期望返回错误")
	}
	if !errors.Is(err, hcommon.I18nError(i18n.MsgSMHProvisionPanic)) {
		t.Errorf("期望 panic 错误信息，实际=%v", err)
	}
}

// =====================================================================
// smhProvisionErrorMessage 测试
// =====================================================================

// ---------- Test: smhProvisionErrorMessage 错误码映射 ----------

func TestSmhProvisionErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		// ==== 云 API 错误（字符串匹配） ====

		// ---- INSUFFICIENT_BALANCE ----
		{
			name:     "BalanceLess 关键字",
			err:      errors.New(`[TencentCloudSDKError] Code=UnsupportedOperation.BalanceLess, Message=margin block balance less error`),
			expected: "INSUFFICIENT_BALANCE",
		},
		{
			name:     "INSUFFICIENT_BALANCE 关键字",
			err:      errors.New(`call order-center error: ERR_INSUFFICIENT_BALANCE`),
			expected: "INSUFFICIENT_BALANCE",
		},
		{
			name:     "balance less 关键字",
			err:      errors.New(`margin block balance less error`),
			expected: "INSUFFICIENT_BALANCE",
		},
		{
			name: "真实生产日志：余额不足完整错误",
			err: errors.New(
				`CreateLibrary: [TencentCloudSDKError] Code=UnsupportedOperation.BalanceLess, ` +
					`Message=margin block balance less error[[ERR_INSUFFICIENT_BALANCE]balance less error:` +
					`call order-center error:uin:100000679786 The total balance -2774 is less than the frozen amount 0]` +
					`[seqId:q4qtal1m-zreg-ttkd-qrio-03adcef94314], RequestId=13383cd0-8fe1-4797-aed1-a465ec676d19`),
			expected: "INSUFFICIENT_BALANCE",
		},

		// ---- STS_ROLE_NOT_FOUND ----
		{
			name:     "role not exist 关键字",
			err:      errors.New(`AssumeRole 失败: [TencentCloudSDKError] Code=InternalError.GetRoleError, Message=role not exist, RequestId=20d4c6fe`),
			expected: "STS_ROLE_NOT_FOUND",
		},
		{
			name:     "GetRoleError 关键字",
			err:      errors.New(`[TencentCloudSDKError] Code=InternalError.GetRoleError, Message=some error`),
			expected: "STS_ROLE_NOT_FOUND",
		},
		{
			name: "真实生产日志：STS 角色不存在完整错误",
			err: errors.New(
				`create SMH cloud client: STS 临时密钥刷新失败: AssumeRole 失败: ` +
					`[TencentCloudSDKError] Code=InternalError.GetRoleError, Message=role not exist, ` +
					`RequestId=20d4c6fe-4c01-48a0-a4f5-de949e533def`),
			expected: "STS_ROLE_NOT_FOUND",
		},

		// ---- 网络错误 → INTERNAL_ERROR ----
		{
			name:     "网络超时错误",
			err:      errors.New(`dial tcp 10.0.0.1:443: i/o timeout`),
			expected: "INTERNAL_ERROR",
		},
		{
			name:     "连接被拒绝",
			err:      errors.New(`dial tcp 127.0.0.1:3306: connection refused`),
			expected: "INTERNAL_ERROR",
		},
		{
			name:     "DNS 解析失败",
			err:      errors.New(`dial tcp: lookup smh.tencentcloudapi.com: no such host`),
			expected: "INTERNAL_ERROR",
		},

		// ==== 自定义错误类型（errors.As 提取错误码） ====

		// ---- PROVISION_IN_PROGRESS ----
		{
			name:     "分布式锁冲突",
			err:      &controller.SMHProvisionError{Code: "PROVISION_IN_PROGRESS", Err: errors.New("acquire smh provision lock: lock already held")},
			expected: "PROVISION_IN_PROGRESS",
		},

		// ---- CREATE_LIBRARY_FAILED ----
		{
			name:     "创建媒体库响应缺少字段",
			err:      &controller.SMHProvisionError{Code: "CREATE_LIBRARY_FAILED", Err: errors.New("CreateLibrary: response missing LibraryId or AccessDomain")},
			expected: "CREATE_LIBRARY_FAILED",
		},
		{
			name:     "创建媒体库发送失败",
			err:      &controller.SMHProvisionError{Code: "CREATE_LIBRARY_FAILED", Err: errors.New("CreateLibrary: send request failed")},
			expected: "CREATE_LIBRARY_FAILED",
		},
		{
			name: "CreateLibrary 因余额不足失败 — 云 API 错误优先于自定义错误码",
			err: &controller.SMHProvisionError{
				Code: "CREATE_LIBRARY_FAILED",
				Err:  errors.New(`CreateLibrary (requestId=xxx): UnsupportedOperation.BalanceLess - margin block balance less error`),
			},
			expected: "INSUFFICIENT_BALANCE",
		},

		// ---- UPDATE_LIBRARY_FAILED ----
		{
			name:     "更新媒体库配置失败",
			err:      &controller.SMHProvisionError{Code: "UPDATE_LIBRARY_FAILED", Err: errors.New("UpdateLibraryInternal: send request failed")},
			expected: "UPDATE_LIBRARY_FAILED",
		},

		// ---- DESCRIBE_SECRET_FAILED ----
		{
			name:     "获取媒体库密钥失败",
			err:      &controller.SMHProvisionError{Code: "DESCRIBE_SECRET_FAILED", Err: errors.New("DescribeLibrarySecret: response missing LibrarySecret")},
			expected: "DESCRIBE_SECRET_FAILED",
		},

		// ---- CREATE_SPACE_FAILED ----
		{
			name:     "创建空间失败",
			err:      &controller.SMHProvisionError{Code: "CREATE_SPACE_FAILED", Err: errors.New("CreateSpace(skillhub): 500 Internal Server Error")},
			expected: "CREATE_SPACE_FAILED",
		},

		// ---- 设置空间配额失败 → INTERNAL_ERROR ----
		{
			name:     "UpdateSpaceInternal 失败",
			err:      &controller.SMHProvisionError{Code: "INTERNAL_ERROR", Err: errors.New("UpdateSpaceInternal(skillhub): send request failed")},
			expected: "INTERNAL_ERROR",
		},
		{
			name:     "CreateSpaceQuota 失败",
			err:      &controller.SMHProvisionError{Code: "INTERNAL_ERROR", Err: errors.New("CreateSpaceQuota(common): 400 Bad Request")},
			expected: "INTERNAL_ERROR",
		},

		// ---- INTERNAL_ERROR ----
		{
			name:     "update SiteConfig 失败",
			err:      &controller.SMHProvisionError{Code: "INTERNAL_ERROR", Err: errors.New("update SiteConfig: database connection lost")},
			expected: "INTERNAL_ERROR",
		},
		{
			name:     "persist libraryId 失败",
			err:      &controller.SMHProvisionError{Code: "INTERNAL_ERROR", Err: errors.New("persist libraryId to SiteConfig: deadlock found")},
			expected: "INTERNAL_ERROR",
		},
		{
			name:     "UpsertSMHSpace 失败",
			err:      &controller.SMHProvisionError{Code: "INTERNAL_ERROR", Err: errors.New("UpsertSMHSpace(skillhub): duplicate entry")},
			expected: "INTERNAL_ERROR",
		},

		// ==== 未知错误 → INTERNAL_ERROR ====
		{
			name:     "未知错误",
			err:      errors.New(`some random error`),
			expected: "INTERNAL_ERROR",
		},
		{
			name:     "空错误信息",
			err:      errors.New(``),
			expected: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := smhProvisionErrorMessage(tt.err)
			if got != tt.expected {
				t.Errorf("smhProvisionErrorMessage(%q) = %q, 期望 %q", tt.err.Error(), got, tt.expected)
			}
		})
	}
}

// ---------- Test: smhProvisionErrorMessage 返回值长度不超过 64 字符 ----------

func TestSmhProvisionErrorMessage_MaxLength(t *testing.T) {
	errs := []error{
		errors.New("BalanceLess"),
		errors.New("role not exist"),
		errors.New("unknown error with a very long message that should still return a short code"),
	}

	for _, err := range errs {
		msg := smhProvisionErrorMessage(err)
		if len(msg) > 64 {
			t.Errorf("错误码 %q 长度 %d 超过 64 字符限制", msg, len(msg))
		}
	}
}

// ---------- Test: 错误码互斥性 — 每个已知错误只映射到一个错误码 ----------

func TestSmhProvisionErrorMessage_MutuallyExclusive(t *testing.T) {
	knownCodes := map[string]bool{
		"INSUFFICIENT_BALANCE":   true,
		"STS_ROLE_NOT_FOUND":     true,
		"PROVISION_IN_PROGRESS":  true,
		"CREATE_LIBRARY_FAILED":  true,
		"UPDATE_LIBRARY_FAILED":  true,
		"DESCRIBE_SECRET_FAILED": true,
		"CREATE_SPACE_FAILED":    true,
		"INTERNAL_ERROR":         true,
	}

	testErrors := []error{
		// 云 API 错误（字符串匹配）
		errors.New("BalanceLess"),
		errors.New("INSUFFICIENT_BALANCE"),
		errors.New("balance less"),
		errors.New("role not exist"),
		errors.New("GetRoleError"),
		errors.New("dial tcp: i/o timeout"),
		errors.New("connection refused"),
		errors.New("no such host"),
		// 自定义错误类型（errors.As 提取）
		&controller.SMHProvisionError{Code: "PROVISION_IN_PROGRESS", Err: errors.New("lock")},
		&controller.SMHProvisionError{Code: "CREATE_LIBRARY_FAILED", Err: errors.New("error")},
		&controller.SMHProvisionError{Code: "UPDATE_LIBRARY_FAILED", Err: errors.New("error")},
		&controller.SMHProvisionError{Code: "DESCRIBE_SECRET_FAILED", Err: errors.New("error")},
		&controller.SMHProvisionError{Code: "CREATE_SPACE_FAILED", Err: errors.New("error")},
		&controller.SMHProvisionError{Code: "INTERNAL_ERROR", Err: errors.New("error")},
		// 未知错误 → INTERNAL_ERROR
		errors.New("random"),
	}

	for _, err := range testErrors {
		code := smhProvisionErrorMessage(err)
		if !knownCodes[code] {
			t.Errorf("smhProvisionErrorMessage(%q) 返回了未知错误码 %q", err.Error(), code)
		}
	}
}
