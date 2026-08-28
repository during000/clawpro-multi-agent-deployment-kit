package model

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// ─── 通知类型常量测试 ─────────────────────────────────────────────────────────

func TestNotificationTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"NotifyTypeAdminDelete", NotifyTypeAdminDelete, "admin_delete"},
		{"NotifyTypeExternalDestroy", NotifyTypeExternalDestroy, "external_destroy"},
		{"NotifyTypeInstanceCreateSuccess", NotifyTypeInstanceCreateSuccess, "instance_create_success"},
		{"NotifyTypeInstanceDeleteSuccess", NotifyTypeInstanceDeleteSuccess, "instance_delete_success"},
		{"NotifyTypeInstanceUpgradeSuccess", NotifyTypeInstanceUpgradeSuccess, "instance_upgrade_success"},
		{"NotifyTypeInstanceReinstallSuccess", NotifyTypeInstanceReinstallSuccess, "instance_reinstall_success"},
		{"NotifyTypeInstanceCreateFailed", NotifyTypeInstanceCreateFailed, "instance_create_failed"},
		{"NotifyTypeInstanceReinstallFailed", NotifyTypeInstanceReinstallFailed, "instance_reinstall_failed"},
		{"NotifyTypeInstanceUpgradeFailed", NotifyTypeInstanceUpgradeFailed, "instance_upgrade_failed"},
		{"NotifyTypeQuotaExceeded", NotifyTypeQuotaExceeded, "quota_exceeded"},
		{"NotifyTypeModelConfigFailed", NotifyTypeModelConfigFailed, "model_config_failed"},
		{"NotifyTypeChannelConfigFailed", NotifyTypeChannelConfigFailed, "channel_config_failed"},
		{"NotifyTypeSkillInstallFailed", NotifyTypeSkillInstallFailed, "skill_install_failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("got %q, want %q", tt.got, tt.expected)
			}
		})
	}
}

// ─── 消息类别常量测试 ─────────────────────────────────────────────────────────

func TestNotifCategoryConstants(t *testing.T) {
	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"NotifCategorySuccess", NotifCategorySuccess, "success"},
		{"NotifCategoryError", NotifCategoryError, "error"},
		{"NotifCategoryNotice", NotifCategoryNotice, "notice"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("got %q, want %q", tt.got, tt.expected)
			}
		})
	}
}

// ─── NotifErrorDetail JSON 序列化测试 ────────────────────────────────────────

func TestNotifErrorDetail_Marshal(t *testing.T) {
	tests := []struct {
		name     string
		detail   NotifErrorDetail
		contains []string // JSON 中应包含的字段
		omits    []string // JSON 中不应包含的字段（omitempty 生效时）
	}{
		{
			name: "全字段",
			detail: NotifErrorDetail{
				Error:      "CVM 创建失败",
				Detail:     "quota exceeded",
				RequestId:  "req-12345",
				InstanceId: "ins-abc",
			},
			contains: []string{`"error"`, `"detail"`, `"request_id"`, `"instance_id"`},
		},
		{
			name:     "仅 Error 字段",
			detail:   NotifErrorDetail{Error: "简单错误"},
			contains: []string{`"error":"简单错误"`},
			omits:    []string{`"detail"`, `"request_id"`, `"instance_id"`},
		},
		{
			name:     "空结构体",
			detail:   NotifErrorDetail{},
			contains: []string{`"error":""`},
			omits:    []string{`"detail"`, `"request_id"`, `"instance_id"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := json.Marshal(tt.detail)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}
			s := string(b)

			for _, c := range tt.contains {
				if !containsStr(s, c) {
					t.Errorf("JSON should contain %q, got %s", c, s)
				}
			}
			for _, o := range tt.omits {
				if containsStr(s, o) {
					t.Errorf("JSON should NOT contain %q (omitempty), got %s", o, s)
				}
			}
		})
	}
}

func TestNotifErrorDetail_Unmarshal(t *testing.T) {
	jsonStr := `{"error":"测试错误","detail":"详细信息","request_id":"req-1","instance_id":"ins-1"}`
	var detail NotifErrorDetail
	if err := json.Unmarshal([]byte(jsonStr), &detail); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if detail.Error != "测试错误" {
		t.Errorf("Error: got %q, want %q", detail.Error, "测试错误")
	}
	if detail.Detail != "详细信息" {
		t.Errorf("Detail: got %q, want %q", detail.Detail, "详细信息")
	}
	if detail.RequestId != "req-1" {
		t.Errorf("RequestId: got %q, want %q", detail.RequestId, "req-1")
	}
	if detail.InstanceId != "ins-1" {
		t.Errorf("InstanceId: got %q, want %q", detail.InstanceId, "ins-1")
	}
}

// ─── 分页参数校验测试 ─────────────────────────────────────────────────────────

func TestNormalizePagination(t *testing.T) {
	tests := []struct {
		name             string
		inputPage        int
		inputPageSize    int
		expectedPage     int
		expectedPageSize int
	}{
		// 正常值
		{"正常参数", 1, 20, 1, 20},
		{"正常参数_第二页", 2, 50, 2, 50},
		{"正常参数_最大页大小", 1, 100, 1, 100},

		// page 异常值
		{"page为0时默认为1", 0, 20, 1, 20},
		{"page为负数时默认为1", -1, 20, 1, 20},
		{"page为极小负数时默认为1", -100, 20, 1, 20},

		// pageSize 异常值
		{"pageSize为0时默认为20", 1, 0, 1, 20},
		{"pageSize为负数时默认为20", 1, -1, 1, 20},
		{"pageSize为极小负数时默认为20", 1, -100, 1, 20},
		{"pageSize超过100时限制为100", 1, 101, 1, 100},
		{"pageSize远超100时限制为100", 1, 999, 1, 100},

		// page 和 pageSize 同时异常
		{"page和pageSize同时为0", 0, 0, 1, 20},
		{"page和pageSize同时为负数", -5, -10, 1, 20},
		{"page为0且pageSize超限", 0, 200, 1, 100},

		// 边界值
		{"pageSize恰好为1", 1, 1, 1, 1},
		{"pageSize恰好为100", 1, 100, 1, 100},
		{"page恰好为1", 1, 20, 1, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPage, gotPageSize := normalizePagination(tt.inputPage, tt.inputPageSize)
			if gotPage != tt.expectedPage {
				t.Errorf("page: got %d, want %d", gotPage, tt.expectedPage)
			}
			if gotPageSize != tt.expectedPageSize {
				t.Errorf("pageSize: got %d, want %d", gotPageSize, tt.expectedPageSize)
			}
		})
	}
}

// TestNormalizePagination_OffsetSafety 验证校验后计算 offset 不会出现负数
func TestNormalizePagination_OffsetSafety(t *testing.T) {
	dangerousInputs := []struct {
		page     int
		pageSize int
	}{
		{0, 0},
		{-1, -1},
		{0, 20},
		{-100, 50},
		{1, -10},
	}

	for _, input := range dangerousInputs {
		page, pageSize := normalizePagination(input.page, input.pageSize)
		offset := (page - 1) * pageSize
		if offset < 0 {
			t.Errorf("normalizePagination(%d, %d) => page=%d, pageSize=%d, offset=%d 为负数",
				input.page, input.pageSize, page, pageSize, offset)
		}
	}
}

// ─── 类型与类别映射一致性测试 ─────────────────────────────────────────────────

func TestNotifyTypeCategoryMapping(t *testing.T) {
	// 验证每个 Type 都有对应的 Category 归属
	typeToCategory := map[string]string{
		NotifyTypeAdminDelete:              NotifCategoryNotice,
		NotifyTypeExternalDestroy:          NotifCategoryNotice,
		NotifyTypeInstanceCreateSuccess:    NotifCategorySuccess,
		NotifyTypeInstanceDeleteSuccess:    NotifCategorySuccess,
		NotifyTypeInstanceUpgradeSuccess:   NotifCategorySuccess,
		NotifyTypeInstanceReinstallSuccess: NotifCategorySuccess,
		NotifyTypeInstanceCreateFailed:     NotifCategoryError,
		NotifyTypeInstanceReinstallFailed:  NotifCategoryError,
		NotifyTypeInstanceUpgradeFailed:    NotifCategoryError,
		NotifyTypeQuotaExceeded:            NotifCategoryError,
		NotifyTypeModelConfigFailed:        NotifCategoryError,
		NotifyTypeChannelConfigFailed:      NotifCategoryError,
		NotifyTypeSkillInstallFailed:       NotifCategoryError,
	}

	validCategories := map[string]bool{
		NotifCategorySuccess: true,
		NotifCategoryError:   true,
		NotifCategoryNotice:  true,
	}

	for typ, cat := range typeToCategory {
		t.Run(typ, func(t *testing.T) {
			if !validCategories[cat] {
				t.Errorf("type %q maps to invalid category %q", typ, cat)
			}
		})
	}
}

// ─── Notification 结构体字段默认值测试 ────────────────────────────────────────

func TestNotification_DefaultValues(t *testing.T) {
	n := Notification{}
	if n.Category != "" {
		t.Errorf("Category default should be empty string (GORM tag default handled at gdb level), got %q", n.Category)
	}
	if n.IsRead != false {
		t.Errorf("IsRead default should be false, got %v", n.IsRead)
	}
	if n.ReadAt != nil {
		t.Errorf("ReadAt default should be nil, got %v", n.ReadAt)
	}
	if n.ErrorDetail != "" {
		t.Errorf("ErrorDetail default should be empty string, got %q", n.ErrorDetail)
	}
}

// ─── 数据库集成测试 ───────────────────────────────────────────────────────────

var notificationTestDBCounter int64

// setupNotificationTestDB 创建隔离的 SQLite 内存数据库用于测试
func setupNotificationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	id := atomic.AddInt64(&notificationTestDBCounter, 1)
	dsn := fmt.Sprintf("file:notifTest%d?mode=memory&cache=shared", id)
	testDB, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开SQLite内存数据库失败: %v", err)
	}
	if err := testDB.AutoMigrate(&Notification{}); err != nil {
		t.Fatalf("自动迁移失败: %v", err)
	}
	return testDB
}

// TestCreateNotification_Success 测试创建通知成功
func TestCreateNotification_Success(t *testing.T) {
	testDB := setupNotificationTestDB(t)
	originalGdb := gdb
	defer func() { gdb = originalGdb }()
	gdb = testDB

	err := CreateNotification(context.Background(), 1, 100, "test-instance", NotifyTypeAdminDelete, "测试标题", "测试消息")
	if err != nil {
		t.Fatalf("创建通知失败: %v", err)
	}

	var notif Notification
	if err := testDB.First(&notif).Error; err != nil {
		t.Fatalf("查询通知失败: %v", err)
	}

	if notif.UserID != 1 {
		t.Errorf("UserID: 期望 1, 得到 %d", notif.UserID)
	}
	if notif.InstanceID != 100 {
		t.Errorf("InstanceID: 期望 100, 得到 %d", notif.InstanceID)
	}
	if notif.Type != NotifyTypeAdminDelete {
		t.Errorf("Type: 期望 %s, 得到 %s", NotifyTypeAdminDelete, notif.Type)
	}
	if notif.Category != NotifCategoryNotice {
		t.Errorf("Category: 期望 %s, 得到 %s", NotifCategoryNotice, notif.Category)
	}
	if notif.IsRead != false {
		t.Errorf("IsRead: 期望 false, 得到 %v", notif.IsRead)
	}
}

// TestCreateNotificationWithCategory_Success 测试创建带类别的通知
func TestCreateNotificationWithCategory_Success(t *testing.T) {
	testDB := setupNotificationTestDB(t)
	originalGdb := gdb
	defer func() { gdb = originalGdb }()
	gdb = testDB

	errDetail := &NotifErrorDetail{
		Error:      "CVM API 错误",
		Detail:     "实例配额已满",
		RequestId:  "req-123",
		InstanceId: "i-abc",
	}

	err := CreateNotificationWithCategory(context.Background(),
		2, 200, "prod-instance",
		NotifyTypeInstanceCreateFailed,
		NotifCategoryError,
		"创建失败",
		"实例创建失败",
		errDetail,
	)
	if err != nil {
		t.Fatalf("创建通知失败: %v", err)
	}

	var notif Notification
	if err := testDB.First(&notif).Error; err != nil {
		t.Fatalf("查询通知失败: %v", err)
	}

	if notif.Category != NotifCategoryError {
		t.Errorf("Category: 期望 %s, 得到 %s", NotifCategoryError, notif.Category)
	}
	if notif.ErrorDetail == "" {
		t.Errorf("ErrorDetail 不应为空")
	}

	var unmarshalDetail NotifErrorDetail
	if err := json.Unmarshal([]byte(notif.ErrorDetail), &unmarshalDetail); err != nil {
		t.Fatalf("反序列化错误详情失败: %v", err)
	}

	if unmarshalDetail.Error != "CVM API 错误" {
		t.Errorf("ErrorDetail.Error: 期望 'CVM API 错误', 得到 %q", unmarshalDetail.Error)
	}
}

// TestCreateSuccessNotification_Success 测试创建成功类通知
func TestCreateSuccessNotification_Success(t *testing.T) {
	testDB := setupNotificationTestDB(t)
	originalGdb := gdb
	defer func() { gdb = originalGdb }()
	gdb = testDB

	err := CreateSuccessNotification(context.Background(),
		3, 300, "upgrade-instance",
		NotifyTypeInstanceUpgradeSuccess,
		"升级成功",
		"实例升级已完成",
	)
	if err != nil {
		t.Fatalf("创建成功通知失败: %v", err)
	}

	var notif Notification
	if err := testDB.First(&notif).Error; err != nil {
		t.Fatalf("查询通知失败: %v", err)
	}

	if notif.Category != NotifCategorySuccess {
		t.Errorf("Category: 期望 %s, 得到 %s", NotifCategorySuccess, notif.Category)
	}
}

// TestMarkNotificationRead_Success 测试标记通知已读
func TestMarkNotificationRead_Success(t *testing.T) {
	testDB := setupNotificationTestDB(t)
	originalGdb := gdb
	defer func() { gdb = originalGdb }()
	gdb = testDB

	// 创建通知
	notif := &Notification{
		UserID:       1,
		InstanceID:   100,
		InstanceName: "test",
		Type:         NotifyTypeAdminDelete,
		Category:     NotifCategoryNotice,
		Title:        "测试",
		IsRead:       false,
	}
	if err := testDB.Create(notif).Error; err != nil {
		t.Fatalf("创建通知失败: %v", err)
	}

	notifID := notif.ID

	// 标记已读
	err := MarkNotificationRead(context.Background(), notifID, 1)
	if err != nil {
		t.Fatalf("标记已读失败: %v", err)
	}

	var updated Notification
	if err := testDB.First(&updated, notifID).Error; err != nil {
		t.Fatalf("查询已读通知失败: %v", err)
	}

	if !updated.IsRead {
		t.Errorf("IsRead: 期望 true, 得到 %v", updated.IsRead)
	}
	if updated.ReadAt == nil {
		t.Errorf("ReadAt 不应为 nil")
	}
}

// TestMarkNotificationRead_WrongUser 测试错误用户标记通知为已读
func TestMarkNotificationRead_WrongUser(t *testing.T) {
	testDB := setupNotificationTestDB(t)
	originalGdb := gdb
	defer func() { gdb = originalGdb }()
	gdb = testDB

	notif := &Notification{
		UserID:       1,
		InstanceID:   100,
		InstanceName: "test",
		Type:         NotifyTypeAdminDelete,
		Category:     NotifCategoryNotice,
		Title:        "测试",
		IsRead:       false,
	}
	if err := testDB.Create(notif).Error; err != nil {
		t.Fatalf("创建通知失败: %v", err)
	}

	// 用错误的用户ID标记
	err := MarkNotificationRead(context.Background(), notif.ID, 999)
	if err != nil {
		t.Fatalf("标记已读不应报错: %v", err)
	}

	var updated Notification
	if err := testDB.First(&updated, notif.ID).Error; err != nil {
		t.Fatalf("查询通知失败: %v", err)
	}

	// 通知不应被标记为已读
	if updated.IsRead {
		t.Errorf("IsRead: 期望 false, 得到 %v", updated.IsRead)
	}
}

// TestGetUnreadNotificationCount 测试获取未读通知数量
func TestGetUnreadNotificationCount(t *testing.T) {
	testDB := setupNotificationTestDB(t)
	originalGdb := gdb
	defer func() { gdb = originalGdb }()
	gdb = testDB

	// 创建3条未读通知
	for i := 0; i < 3; i++ {
		notif := &Notification{
			UserID:       1,
			InstanceID:   uint(100 + i),
			InstanceName: fmt.Sprintf("test-%d", i),
			Type:         NotifyTypeAdminDelete,
			Category:     NotifCategoryNotice,
			Title:        fmt.Sprintf("测试-%d", i),
			IsRead:       false,
		}
		if err := testDB.Create(notif).Error; err != nil {
			t.Fatalf("创建通知失败: %v", err)
		}
	}

	// 创建1条已读通知
	readNotif := &Notification{
		UserID:       1,
		InstanceID:   200,
		InstanceName: "test-read",
		Type:         NotifyTypeAdminDelete,
		Category:     NotifCategoryNotice,
		Title:        "已读",
		IsRead:       true,
		ReadAt:       &now,
	}
	if err := testDB.Create(readNotif).Error; err != nil {
		t.Fatalf("创建已读通知失败: %v", err)
	}

	count, err := GetUnreadNotificationCount(context.Background(), 1)
	if err != nil {
		t.Fatalf("获取未读数量失败: %v", err)
	}

	if count != 3 {
		t.Errorf("未读计数: 期望 3, 得到 %d", count)
	}
}

// TestGetUnreadNotificationCount_NoNotifications 测试没有通知时的计数
func TestGetUnreadNotificationCount_NoNotifications(t *testing.T) {
	testDB := setupNotificationTestDB(t)
	originalGdb := gdb
	defer func() { gdb = originalGdb }()
	gdb = testDB

	count, err := GetUnreadNotificationCount(context.Background(), 999)
	if err != nil {
		t.Fatalf("获取未读数量失败: %v", err)
	}

	if count != 0 {
		t.Errorf("未读计数: 期望 0, 得到 %d", count)
	}
}

// TestGetUnreadNotificationCountByCategory 测试按分类统计未读通知
func TestGetUnreadNotificationCountByCategory(t *testing.T) {
	testDB := setupNotificationTestDB(t)
	originalGdb := gdb
	defer func() { gdb = originalGdb }()
	gdb = testDB

	// 创建不同分类的未读通知
	testCases := []struct {
		category string
		count    int
	}{
		{NotifCategorySuccess, 2},
		{NotifCategoryError, 3},
		{NotifCategoryNotice, 1},
	}

	for _, tc := range testCases {
		for i := 0; i < tc.count; i++ {
			notif := &Notification{
				UserID:       1,
				InstanceID:   uint(1000 + i),
				InstanceName: fmt.Sprintf("test-%s-%d", tc.category, i),
				Type:         NotifyTypeAdminDelete,
				Category:     tc.category,
				Title:        fmt.Sprintf("测试-%s-%d", tc.category, i),
				IsRead:       false,
			}
			if err := testDB.Create(notif).Error; err != nil {
				t.Fatalf("创建通知失败: %v", err)
			}
		}
	}

	total, byCategory, err := GetUnreadNotificationCountByCategory(context.Background(), 1)
	if err != nil {
		t.Fatalf("获取分类统计失败: %v", err)
	}

	if total != 6 {
		t.Errorf("总数: 期望 6, 得到 %d", total)
	}

	for _, tc := range testCases {
		if byCategory[tc.category] != int64(tc.count) {
			t.Errorf("分类 %s 计数: 期望 %d, 得到 %d", tc.category, tc.count, byCategory[tc.category])
		}
	}
}

// TestGetUserNotifications 测试获取用户通知列表
func TestGetUserNotifications(t *testing.T) {
	testDB := setupNotificationTestDB(t)
	originalGdb := gdb
	defer func() { gdb = originalGdb }()
	gdb = testDB

	// 创建5条未读通知
	for i := 0; i < 5; i++ {
		notif := &Notification{
			UserID:       1,
			InstanceID:   uint(100 + i),
			InstanceName: fmt.Sprintf("test-%d", i),
			Type:         NotifyTypeAdminDelete,
			Category:     NotifCategoryNotice,
			Title:        fmt.Sprintf("测试-%d", i),
			IsRead:       false,
		}
		if err := testDB.Create(notif).Error; err != nil {
			t.Fatalf("创建通知失败: %v", err)
		}
	}

	// 获取第一页
	notifs, total, err := GetUserNotifications(context.Background(), 1, 1, 3, nil, "")
	if err != nil {
		t.Fatalf("获取通知列表失败: %v", err)
	}

	if total != 5 {
		t.Errorf("总数: 期望 5, 得到 %d", total)
	}
	if len(notifs) != 3 {
		t.Errorf("返回条数: 期望 3, 得到 %d", len(notifs))
	}
}

// TestGetUserNotifications_FilterByReadStatus 测试按已读状态筛选
func TestGetUserNotifications_FilterByReadStatus(t *testing.T) {
	testDB := setupNotificationTestDB(t)
	originalGdb := gdb
	defer func() { gdb = originalGdb }()
	gdb = testDB

	// 创建已读和未读通知
	for i := 0; i < 2; i++ {
		notif := &Notification{
			UserID:       1,
			InstanceID:   uint(100 + i),
			InstanceName: fmt.Sprintf("unread-%d", i),
			Type:         NotifyTypeAdminDelete,
			Category:     NotifCategoryNotice,
			Title:        fmt.Sprintf("未读-%d", i),
			IsRead:       false,
		}
		if err := testDB.Create(notif).Error; err != nil {
			t.Fatalf("创建未读通知失败: %v", err)
		}
	}

	for i := 0; i < 3; i++ {
		notif := &Notification{
			UserID:       1,
			InstanceID:   uint(200 + i),
			InstanceName: fmt.Sprintf("read-%d", i),
			Type:         NotifyTypeAdminDelete,
			Category:     NotifCategoryNotice,
			Title:        fmt.Sprintf("已读-%d", i),
			IsRead:       true,
			ReadAt:       &now,
		}
		if err := testDB.Create(notif).Error; err != nil {
			t.Fatalf("创建已读通知失败: %v", err)
		}
	}

	// 查询未读
	falseVal := false
	_, total, err := GetUserNotifications(context.Background(), 1, 1, 100, &falseVal, "")
	if err != nil {
		t.Fatalf("查询未读通知失败: %v", err)
	}

	if total != 2 {
		t.Errorf("未读总数: 期望 2, 得到 %d", total)
	}

	// 查询已读
	trueVal := true
	_, total, err = GetUserNotifications(context.Background(), 1, 1, 100, &trueVal, "")
	if err != nil {
		t.Fatalf("查询已读通知失败: %v", err)
	}

	if total != 3 {
		t.Errorf("已读总数: 期望 3, 得到 %d", total)
	}
}

// TestDeleteNotification_Success 测试删除通知
func TestDeleteNotification_Success(t *testing.T) {
	testDB := setupNotificationTestDB(t)
	originalGdb := gdb
	defer func() { gdb = originalGdb }()
	gdb = testDB

	notif := &Notification{
		UserID:       1,
		InstanceID:   100,
		InstanceName: "test",
		Type:         NotifyTypeAdminDelete,
		Category:     NotifCategoryNotice,
		Title:        "测试",
		IsRead:       false,
	}
	if err := testDB.Create(notif).Error; err != nil {
		t.Fatalf("创建通知失败: %v", err)
	}

	// 删除
	rowsAffected, err := DeleteNotification(context.Background(), notif.ID, 1)
	if err != nil {
		t.Fatalf("删除通知失败: %v", err)
	}

	if rowsAffected != 1 {
		t.Errorf("受影响行数: 期望 1, 得到 %d", rowsAffected)
	}

	// 验证已删除
	var count int64
	testDB.Model(&Notification{}).Where("id = ?", notif.ID).Count(&count)
	if count != 0 {
		t.Errorf("通知不应存在: 期望 0, 得到 %d", count)
	}
}

// TestCleanupExpiredNotifications 测试清理过期通知
func TestCleanupExpiredNotifications(t *testing.T) {
	testDB := setupNotificationTestDB(t)
	originalGdb := gdb
	defer func() { gdb = originalGdb }()
	gdb = testDB

	// 创建过期通知（31天前）
	oldTime := time.Now().AddDate(0, 0, -31)
	oldNotif := &Notification{
		UserID:       1,
		InstanceID:   100,
		InstanceName: "old",
		Type:         NotifyTypeAdminDelete,
		Category:     NotifCategoryNotice,
		Title:        "过期",
	}
	if err := testDB.Create(oldNotif).Error; err != nil {
		t.Fatalf("创建过期通知失败: %v", err)
	}
	testDB.Model(oldNotif).Update("created_at", oldTime)

	// 创建新通知（1天前）
	recentTime := time.Now().AddDate(0, 0, -1)
	recentNotif := &Notification{
		UserID:       1,
		InstanceID:   101,
		InstanceName: "recent",
		Type:         NotifyTypeAdminDelete,
		Category:     NotifCategoryNotice,
		Title:        "新",
	}
	if err := testDB.Create(recentNotif).Error; err != nil {
		t.Fatalf("创建新通知失败: %v", err)
	}
	testDB.Model(recentNotif).Update("created_at", recentTime)

	// 清理30天前的通知
	rowsAffected, err := CleanupExpiredNotifications(context.Background(), 30)
	if err != nil {
		t.Fatalf("清理过期通知失败: %v", err)
	}

	if rowsAffected != 1 {
		t.Errorf("删除行数: 期望 1, 得到 %d", rowsAffected)
	}

	// 验证结果
	var remaining int64
	testDB.Model(&Notification{}).Count(&remaining)
	if remaining != 1 {
		t.Errorf("剩余通知数: 期望 1, 得到 %d", remaining)
	}
}

// ─── 辅助函数 ────────────────────────────────────────────────────────────────

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && findSubstr(s, substr))
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

var now = time.Now()
