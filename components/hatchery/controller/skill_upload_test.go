package controller

import (
	"strings"
	"testing"

	"hatchery/i18n"
)

func TestSkillUploadMaxSize_Is100MB(t *testing.T) {
	const want = 100 << 20
	if skillUploadMaxSize != want {
		t.Fatalf("skillUploadMaxSize=%d, want %d (100MB)", skillUploadMaxSize, want)
	}
}

func TestIsSkillUploadTooLarge_Boundary(t *testing.T) {
	if isSkillUploadTooLarge(skillUploadMaxSize) {
		t.Fatal("size == skillUploadMaxSize should be allowed")
	}
	if !isSkillUploadTooLarge(skillUploadMaxSize + 1) {
		t.Fatal("size == skillUploadMaxSize+1 should be rejected")
	}
	if isSkillUploadTooLarge(0) {
		t.Fatal("size 0 should be allowed")
	}
}

func TestMsgSkillUploadFileSizeTooLarge_Mentions100MB(t *testing.T) {
	uploadKey := i18n.MsgSkillUploadFileSizeTooLarge.String()
	bundleKey := i18n.MsgSkillFileSizeTooLarge.String()
	if uploadKey == bundleKey {
		t.Fatal("上传与 Bundle 下载文案 Key 不应相同")
	}
	if !strings.Contains(uploadKey, "100MB") {
		t.Fatalf("上传文案应包含 100MB，实际=%q", uploadKey)
	}
	if !strings.Contains(bundleKey, "50MB") {
		t.Fatalf("Bundle 文案应保留 50MB，实际=%q", bundleKey)
	}
}
