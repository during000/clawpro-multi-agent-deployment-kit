package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"hatchery/model"
)

// ============================================================================
// 本文件聚焦 admin_images.go 里低覆盖率函数：
//   1. HandleDeleteImage            (L216, 0%) — 纯 DB
//   2. strVal / int64Val            (L445/L452, 0%) — trivial 纯函数
// HandleListCloudImages / SeedAvailableImages 需要 CVM SDK，跳过。
// ============================================================================

// ─── HandleDeleteImage ─────────────────────────────────────────────────────

func TestHandleDeleteImage_Unauthorized(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()
	req := httptest.NewRequest(http.MethodPost, "/admin/image/delete?id=1", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleDeleteImage(rr, req)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("未登录应 401/403，实际=%d", rr.Code)
	}
}

func TestHandleDeleteImage_NotFound(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	req := adminImagesReq(http.MethodPost, "/admin/image/delete?id=9999", "")
	rr := httptest.NewRecorder()
	HandleDeleteImage(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("镜像不存在应 404，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleDeleteImage_EnabledCannotDelete(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	img := &model.AIImage{ImageName: "enabled-img", ImageId: "img-1", Enabled: true}
	model.DB(context.Background()).Create(img)

	req := adminImagesReq(http.MethodPost, fmt.Sprintf("/admin/image/delete?id=%d", img.ID), "")
	rr := httptest.NewRecorder()
	HandleDeleteImage(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("启用状态镜像应 403 不允许删除，实际=%d", rr.Code)
	}

	var count int64
	model.DB(context.Background()).Model(&model.AIImage{}).Where("id = ?", img.ID).Count(&count)
	if count != 1 {
		t.Errorf("镜像不应被删除，实际 count=%d", count)
	}
}

func TestHandleDeleteImage_DisabledSuccess(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	img := &model.AIImage{ImageName: "disabled-img", ImageId: "img-2", Enabled: false}
	model.DB(context.Background()).Create(img)

	req := adminImagesReq(http.MethodPost, fmt.Sprintf("/admin/image/delete?id=%d", img.ID), "")
	rr := httptest.NewRecorder()
	HandleDeleteImage(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	// GORM 默认软删除，检查 deleted_at 是否已填
	var count int64
	model.DB(context.Background()).Model(&model.AIImage{}).Where("id = ?", img.ID).Count(&count)
	if count != 0 {
		t.Errorf("镜像应被软删除（查询不到），实际 count=%d", count)
	}
}

// ─── strVal / int64Val 纯函数 ─────────────────────────────────────────────

func TestStrVal(t *testing.T) {
	if StrVal(nil) != "" {
		t.Errorf("nil 应返回空串")
	}
	s := "hello"
	if StrVal(&s) != "hello" {
		t.Errorf("非空指针应返回值")
	}
	empty := ""
	if StrVal(&empty) != "" {
		t.Errorf("指向空串的指针应返回空串")
	}
}

func TestInt64Val(t *testing.T) {
	if Int64Val(nil) != 0 {
		t.Errorf("nil 应返回 0")
	}
	v := int64(42)
	if Int64Val(&v) != 42 {
		t.Errorf("非空指针应返回值，实际=%d", Int64Val(&v))
	}
	neg := int64(-1)
	if Int64Val(&neg) != -1 {
		t.Errorf("负数应被保留，实际=%d", Int64Val(&neg))
	}
}
