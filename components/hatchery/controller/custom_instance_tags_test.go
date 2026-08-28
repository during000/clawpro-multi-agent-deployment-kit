package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"hatchery/model"

	sdkerrors "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
)

func TestParseCreateInstanceTags(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		raw     string
		want    []createInstanceTag
		wantErr bool
	}{
		{name: "empty", raw: "", want: nil},
		{name: "empty array", raw: "[]", want: []createInstanceTag{}},
		{
			name: "multiple",
			raw:  `[{"key":"env","value":"prod"},{"key":"team","value":"business-1"}]`,
			want: []createInstanceTag{{Key: "env", Value: "prod"}, {Key: "team", Value: "business-1"}},
		},
		{name: "object", raw: `{"key":"env","value":"prod"}`, wantErr: true},
		{name: "unknown field", raw: `[{"key":"env","value":"prod","scope":"x"}]`, wantErr: true},
		{name: "trailing value", raw: `[{"key":"env","value":"prod"}] []`, wantErr: true},
		{name: "malformed", raw: `[{"key":]`, wantErr: true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseCreateInstanceTags(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseCreateInstanceTags: %v", err)
			}
			if fmt.Sprint(got) != fmt.Sprint(tt.want) {
				t.Fatalf("tags = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestMergeCreateInstanceTags_CustomOverridesDefault(t *testing.T) {
	t.Parallel()
	custom := []createInstanceTag{
		{Key: "env", Value: "dev"},
		{Key: "business", Value: "one"},
	}
	defaults := []model.TagItem{
		{Key: "env", Value: "prod"},
		{Key: "managed-by", Value: "clawpro"},
	}

	got := mergeCreateInstanceTags(custom, defaults)
	assertSDKTags(t, got, []createInstanceTag{
		{Key: "env", Value: "dev"},
		{Key: "business", Value: "one"},
		{Key: "managed-by", Value: "clawpro"},
	})
}

func TestMergeCreateInstanceTags_PreservesDuplicateCustomKeysForCloudValidation(t *testing.T) {
	t.Parallel()
	custom := []createInstanceTag{
		{Key: "env", Value: "prod"},
		{Key: "env", Value: "staging"},
	}
	got := mergeCreateInstanceTags(custom, nil)
	assertSDKTags(t, got, custom)
}

func TestCreateInstanceTagItemsForCache_MatchesMergedRunInstancesTags(t *testing.T) {
	t.Parallel()
	custom := []createInstanceTag{
		{Key: "env", Value: "dev"},
		{Key: "business", Value: "one"},
	}
	defaults := []model.TagItem{
		{Key: "env", Value: "prod"},
		{Key: "managed-by", Value: "clawpro"},
	}

	got := createInstanceTagItemsForCache(custom, defaults)
	want := []model.TagItem{
		{Key: "env", Value: "dev"},
		{Key: "business", Value: "one"},
		{Key: "managed-by", Value: "clawpro"},
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("cached tags = %+v, want %+v", got, want)
	}
}

func TestCvmRunInstancesHTTPStatus(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name               string
		err                error
		customTagsProvided bool
		want               int
	}{
		{
			name: "managed default tag failure remains internal",
			err:  &sdkerrors.TencentCloudSDKError{Code: "InvalidParameterValue.TagKeyNotFound"},
			want: http.StatusInternalServerError,
		},
		{
			name:               "custom illegal tag key",
			err:                &sdkerrors.TencentCloudSDKError{Code: "FailedOperation.IllegalTagKey"},
			customTagsProvided: true,
			want:               http.StatusBadRequest,
		},
		{
			name:               "custom illegal tag value",
			err:                &sdkerrors.TencentCloudSDKError{Code: "FailedOperation.IllegalTagValue"},
			customTagsProvided: true,
			want:               http.StatusBadRequest,
		},
		{
			name:               "custom reserved tag key",
			err:                &sdkerrors.TencentCloudSDKError{Code: "FailedOperation.TagKeyReserved"},
			customTagsProvided: true,
			want:               http.StatusBadRequest,
		},
		{
			name:               "custom duplicate tags",
			err:                &sdkerrors.TencentCloudSDKError{Code: "InvalidParameterValue.DuplicateTags"},
			customTagsProvided: true,
			want:               http.StatusBadRequest,
		},
		{
			name:               "custom tag key not found",
			err:                &sdkerrors.TencentCloudSDKError{Code: "InvalidParameterValue.TagKeyNotFound"},
			customTagsProvided: true,
			want:               http.StatusBadRequest,
		},
		{
			name:               "custom tag quota exceeded",
			err:                &sdkerrors.TencentCloudSDKError{Code: "InvalidParameterValue.TagQuotaLimitExceeded"},
			customTagsProvided: true,
			want:               http.StatusBadRequest,
		},
		{
			name:               "unrelated SDK error remains internal",
			err:                &sdkerrors.TencentCloudSDKError{Code: "InternalError"},
			customTagsProvided: true,
			want:               http.StatusInternalServerError,
		},
		{
			name:               "non SDK error remains internal",
			err:                fmt.Errorf("network error"),
			customTagsProvided: true,
			want:               http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := cvmRunInstancesHTTPStatus(tt.err, tt.customTagsProvided); got != tt.want {
				t.Fatalf("status = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestHandleCreateInstance_InvalidCustomTags(t *testing.T) {
	cleanup := initFiveHandlersTestDB(t)
	defer cleanup()
	user := &model.User{Username: "invalid-custom-tags", Password: "x", Role: "user", InstanceQuota: 10}
	if err := model.DB(t.Context()).Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	form := url.Values{}
	form.Set("name", "invalid-custom-tags")
	form.Set("tags", `{"key":"env","value":"prod"}`)
	req := jsonReqWithSession(t, http.MethodPost, "/openclaw/create", user.Username, form.Encode())
	rr := httptest.NewRecorder()
	HandleCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rr.Code, rr.Body.String())
	}
	var count int64
	if err := model.DB(t.Context()).Model(&model.Instance{}).Count(&count).Error; err != nil {
		t.Fatalf("count instances: %v", err)
	}
	if count != 0 {
		t.Fatalf("invalid tags created %d instance rows", count)
	}
}

func TestAdminCreate_TagsRejectUnknownNestedField(t *testing.T) {
	cleanup := setupAdminCreateValidationDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/admin/instances/create", strings.NewReader(
		`{"user_id":1,"name":"test","agent_type":"openclaw","tags":[{"key":"env","value":"prod","scope":"x"}]}`,
	))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+AdminToken)
	rr := httptest.NewRecorder()
	HandleAdminCreateInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rr.Code, rr.Body.String())
	}
}

func mockCVMRunInstancesCaptureServer(t *testing.T) (*httptest.Server, <-chan []byte) {
	t.Helper()
	captured := make(chan []byte, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Header.Get("X-TC-Action") == "DescribeInstances" {
			fmt.Fprint(w, `{"Response":{"InstanceSet":[{"InstanceId":"ins-mock-writectx","InstanceState":"RUNNING"}],"TotalCount":1,"RequestId":"mock-describe-req-id"}}`)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read RunInstances request: %v", err)
		} else {
			captured <- body
		}
		fmt.Fprint(w, `{"Response":{"InstanceIdSet":["ins-mock-writectx"],"RequestId":"mock-req-id"}}`)
	}))
	return server, captured
}

func assertCapturedInstanceTags(t *testing.T, captured <-chan []byte, want []createInstanceTag) {
	t.Helper()
	var body []byte
	select {
	case body = <-captured:
	case <-time.After(2 * time.Second):
		t.Fatal("RunInstances request was not captured")
	}
	var request struct {
		TagSpecification []struct {
			ResourceType *string `json:"ResourceType"`
			Tags         []struct {
				Key   *string `json:"Key"`
				Value *string `json:"Value"`
			} `json:"Tags"`
		} `json:"TagSpecification"`
	}
	if err := json.Unmarshal(body, &request); err != nil {
		t.Fatalf("decode RunInstances request: %v\nbody=%s", err, body)
	}
	if len(request.TagSpecification) != 1 {
		t.Fatalf("TagSpecification = %+v, want one instance entry", request.TagSpecification)
	}
	spec := request.TagSpecification[0]
	if spec.ResourceType == nil || *spec.ResourceType != "instance" {
		t.Fatalf("ResourceType = %v, want instance", spec.ResourceType)
	}
	got := make([]createInstanceTag, 0, len(spec.Tags))
	for _, tag := range spec.Tags {
		if tag.Key == nil || tag.Value == nil {
			t.Fatalf("tag contains nil key/value: %+v", tag)
		}
		got = append(got, createInstanceTag{Key: *tag.Key, Value: *tag.Value})
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("RunInstances tags = %+v, want %+v", got, want)
	}
}

func assertSDKTags(t *testing.T, gotTags []*cvm.Tag, want []createInstanceTag) {
	t.Helper()
	got := make([]createInstanceTag, 0, len(gotTags))
	for _, tag := range gotTags {
		if tag == nil || tag.Key == nil || tag.Value == nil {
			t.Fatalf("tag contains nil key/value: %+v", tag)
		}
		got = append(got, createInstanceTag{Key: *tag.Key, Value: *tag.Value})
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("tags = %+v, want %+v", got, want)
	}
}
