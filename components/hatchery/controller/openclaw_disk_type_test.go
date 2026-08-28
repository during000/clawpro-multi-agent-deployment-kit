package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
)

func TestParseCreateInstanceDiskType(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{name: "empty", raw: "", want: ""},
		{name: "bssd", raw: "CLOUD_BSSD", want: "CLOUD_BSSD"},
		{name: "premium lower", raw: " cloud_premium ", want: "CLOUD_PREMIUM"},
		{name: "ssd", raw: "CLOUD_SSD", want: "CLOUD_SSD"},
		{name: "hssd", raw: "CLOUD_HSSD", want: "CLOUD_HSSD"},
		{name: "unsupported", raw: "CLOUD_TSSD", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCreateInstanceDiskType(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHandleCreateInstance_InvalidDiskType(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()

	user := &model.User{Username: "disk-type-user", Password: "x", Role: "user", InstanceQuota: 10}
	if err := model.DB(context.Background()).Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	form := url.Values{}
	form.Set("name", "bad-disk")
	form.Set("disk_type", "CLOUD_TSSD")
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", "disk-type-user", form.Encode())
	rr := httptest.NewRecorder()

	HandleCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("invalid disk_type should return 400, got %d body=%s", rr.Code, rr.Body.String())
	}

	wantedErrMsg := hcommon.I18nError(i18n.MsgWrongDiskType).ErrorMessage(req.Context())
	if !strings.Contains(rr.Body.String(), wantedErrMsg) {
		t.Fatalf("error should mention disk_type, got %s", rr.Body.String())
	}
}

func TestApplyCreateInstanceDiskType(t *testing.T) {
	req := cvm.NewRunInstancesRequest()
	applyCreateInstanceDiskType(req, "CLOUD_PREMIUM")
	if req.SystemDisk == nil || req.SystemDisk.DiskType == nil || *req.SystemDisk.DiskType != "CLOUD_PREMIUM" {
		t.Fatalf("SystemDisk.DiskType should be %s, got %#v", "CLOUD_PREMIUM", req.SystemDisk)
	}

	req.SystemDisk.DiskSize = common.Int64Ptr(80)
	applyCreateInstanceDiskType(req, "CLOUD_BSSD")
	if req.SystemDisk.DiskType == nil || *req.SystemDisk.DiskType != "CLOUD_BSSD" {
		t.Fatalf("SystemDisk.DiskType should be %s, got %#v", "CLOUD_BSSD", req.SystemDisk)
	}
	if req.SystemDisk.DiskSize == nil || *req.SystemDisk.DiskSize != 80 {
		t.Fatalf("SystemDisk.DiskSize should be preserved, got %#v", req.SystemDisk.DiskSize)
	}
}
