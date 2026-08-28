package controller

import (
	"errors"
	"testing"

	sdkerrors "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
)

// ─── isCLSTopicNotExist ──────────────────────────────────────────────────────

func TestIsCLSTopicNotExist_True(t *testing.T) {
	err := sdkerrors.NewTencentCloudSDKError("ResourceNotFound.TopicNotExist", "topic not found", "req-001")
	if !isCLSTopicNotExist(err) {
		t.Error("ResourceNotFound.TopicNotExist 应返回 true")
	}
}

func TestIsCLSTopicNotExist_OtherSDKError(t *testing.T) {
	err := sdkerrors.NewTencentCloudSDKError("InternalError", "internal error", "req-002")
	if isCLSTopicNotExist(err) {
		t.Error("其他 SDK 错误应返回 false")
	}
}

func TestIsCLSTopicNotExist_NonSDKError(t *testing.T) {
	err := errors.New("some generic error")
	if isCLSTopicNotExist(err) {
		t.Error("非 SDK 错误应返回 false")
	}
}

func TestIsCLSTopicNotExist_Nil(t *testing.T) {
	// nil 不是 *TencentCloudSDKError，应返回 false
	if isCLSTopicNotExist(nil) {
		t.Error("nil 错误应返回 false")
	}
}
