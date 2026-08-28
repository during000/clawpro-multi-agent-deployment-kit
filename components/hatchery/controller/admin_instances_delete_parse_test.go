package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	hcommon "hatchery/common"
)

// TestParseAdminDeleteRequest_FormID 旧的 form/id 单删路径保持兼容。
func TestParseAdminDeleteRequest_FormID(t *testing.T) {
	form := url.Values{}
	form.Set("id", "42")
	req := httptest.NewRequest(http.MethodPost, "/admin/instances/delete", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ids, isBatch, err := parseAdminDeleteRequest(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if isBatch {
		t.Fatal("form id 不应走 batch 分支")
	}
	if len(ids) != 1 || ids[0] != 42 {
		t.Fatalf("ids 错: %v", ids)
	}
}

// TestParseAdminDeleteRequest_JSONID body 里只传 id 走单删。
func TestParseAdminDeleteRequest_JSONID(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/admin/instances/delete", strings.NewReader(`{"id": 5}`))
	req.Header.Set("Content-Type", "application/json")

	ids, isBatch, err := parseAdminDeleteRequest(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if isBatch {
		t.Fatal("仅传 JSON id 不应走 batch 分支")
	}
	if len(ids) != 1 || ids[0] != 5 {
		t.Fatalf("ids 错: %v", ids)
	}
}

// TestParseAdminDeleteRequest_JSONIDs body 传 ids 走批量分支。
func TestParseAdminDeleteRequest_JSONIDs(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/admin/instances/delete", strings.NewReader(`{"ids": [1, 2, 3]}`))
	req.Header.Set("Content-Type", "application/json")

	ids, isBatch, err := parseAdminDeleteRequest(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !isBatch {
		t.Fatal("传 ids 应走 batch 分支")
	}
	if len(ids) != 3 || ids[0] != 1 || ids[1] != 2 || ids[2] != 3 {
		t.Fatalf("ids 错: %v", ids)
	}
}

// TestParseAdminDeleteRequest_IDsPriority 同时传 id + ids，ids 优先、id 被忽略。
func TestParseAdminDeleteRequest_IDsPriority(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/admin/instances/delete?id=99", strings.NewReader(`{"id": 5, "ids": [1, 2]}`))
	req.Header.Set("Content-Type", "application/json")

	ids, isBatch, err := parseAdminDeleteRequest(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !isBatch {
		t.Fatal("ids 存在时应走 batch 分支")
	}
	if len(ids) != 2 || ids[0] != 1 || ids[1] != 2 {
		t.Fatalf("ids 被 id=99/5 污染: %v", ids)
	}
}

// TestParseAdminDeleteRequest_EmptyIDs ids=[] 必须 400（不允许空列表）。
func TestParseAdminDeleteRequest_EmptyIDs(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/admin/instances/delete", strings.NewReader(`{"ids": []}`))
	req.Header.Set("Content-Type", "application/json")

	_, isBatch, err := parseAdminDeleteRequest(req)
	if err == nil {
		t.Fatal("ids=[] 应报错")
	}
	if !isBatch {
		t.Fatal("ids 字段显式存在应走 batch 分支（即便为空）并返回 batch-语义的错误")
	}
	if !strings.Contains(hcommon.ErrorMessageWithCtx(context.Background(), err), "空列表") {
		t.Fatalf("错误消息应提示空列表，实际: %v", err)
	}
}

// TestParseAdminDeleteRequest_TooMany ids > 100 必须 400。
func TestParseAdminDeleteRequest_TooMany(t *testing.T) {
	// 构造 101 个元素
	var sb strings.Builder
	sb.WriteString(`{"ids": [`)
	for i := 1; i <= 101; i++ {
		if i > 1 {
			sb.WriteString(",")
		}
		sb.WriteString("1")
	}
	sb.WriteString("]}")

	req := httptest.NewRequest(http.MethodPost, "/admin/instances/delete", strings.NewReader(sb.String()))
	req.Header.Set("Content-Type", "application/json")

	_, _, err := parseAdminDeleteRequest(req)
	if err == nil {
		t.Fatal("ids>100 应报错")
	}
	if !strings.Contains(hcommon.ErrorMessageWithCtx(context.Background(), err), "上限") {
		t.Fatalf("错误消息应提示上限，实际: %v", err)
	}
}

// TestParseAdminDeleteRequest_Neither 两个都不传应 400。
func TestParseAdminDeleteRequest_Neither(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/admin/instances/delete", strings.NewReader(``))
	req.Header.Set("Content-Type", "application/json")

	_, _, err := parseAdminDeleteRequest(req)
	if err == nil {
		t.Fatal("两者都不传应报错")
	}
	if !strings.Contains(hcommon.ErrorMessageWithCtx(context.Background(), err), "缺少参数") {
		t.Fatalf("错误消息应提示缺少参数，实际: %v", err)
	}
}

// TestParseAdminDeleteRequest_IDsDedup 去重 + 过滤 0。
func TestParseAdminDeleteRequest_IDsDedup(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/admin/instances/delete", strings.NewReader(`{"ids": [1, 0, 2, 1, 3, 0, 2]}`))
	req.Header.Set("Content-Type", "application/json")

	ids, isBatch, err := parseAdminDeleteRequest(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !isBatch {
		t.Fatal("应走 batch 分支")
	}
	if len(ids) != 3 || ids[0] != 1 || ids[1] != 2 || ids[2] != 3 {
		t.Fatalf("去重/过滤 0 失败: %v", ids)
	}
}

// TestParseAdminDeleteRequest_AllZeroOrDup 全部无效 id 应报错（去重后空列表）。
func TestParseAdminDeleteRequest_AllZeroOrDup(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/admin/instances/delete", strings.NewReader(`{"ids": [0, 0, 0]}`))
	req.Header.Set("Content-Type", "application/json")

	_, isBatch, err := parseAdminDeleteRequest(req)
	if err == nil {
		t.Fatal("全 0 应报错")
	}
	if !isBatch {
		t.Fatal("应走 batch 分支")
	}
}

// TestParseAdminDeleteRequest_FormIDBadValue form id 非法应 400。
func TestParseAdminDeleteRequest_FormIDBadValue(t *testing.T) {
	form := url.Values{}
	form.Set("id", "abc")
	req := httptest.NewRequest(http.MethodPost, "/admin/instances/delete", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	_, _, err := parseAdminDeleteRequest(req)
	if err == nil {
		t.Fatal("id=abc 应报错")
	}
	if !strings.Contains(hcommon.ErrorMessageWithCtx(req.Context(), err), "无效") {
		t.Fatalf("错误消息应含 '无效'，实际: %v", err)
	}
}
