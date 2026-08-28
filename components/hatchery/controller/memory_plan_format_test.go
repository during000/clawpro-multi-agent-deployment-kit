package controller

import (
	"reflect"
	"strings"
	"testing"
)

// --- parseSceneMeta ---

func TestParseSceneMeta_Valid(t *testing.T) {
	content := `-----META-START-----
summary: 场景摘要
heat: 42
created: 2026-04-01T00:00:00Z
updated: 2026-04-15T00:00:00Z
-----META-END-----

这里是场景正文内容。
多行也没问题。`

	meta, body := parseSceneMeta(content)
	if meta == nil {
		t.Fatal("meta 不应为 nil")
	}
	if meta["summary"] != "场景摘要" {
		t.Errorf("summary = %q, want 场景摘要", meta["summary"])
	}
	if meta["heat"] != "42" {
		t.Errorf("heat = %q, want 42", meta["heat"])
	}
	if meta["created"] != "2026-04-01T00:00:00Z" {
		t.Errorf("created = %q", meta["created"])
	}
	if !strings.Contains(body, "这里是场景正文内容") {
		t.Errorf("body 应包含正文，got %q", body)
	}
}

// 无 META 头时应返回 (nil, 原文内容)
func TestParseSceneMeta_NoMeta(t *testing.T) {
	content := "没有 META 头的纯文本内容"
	meta, body := parseSceneMeta(content)
	if meta != nil {
		t.Errorf("meta 应为 nil, got %v", meta)
	}
	if body != content {
		t.Errorf("body 应为原文，got %q", body)
	}
}

// META 标记顺序错误（END 在 START 前）
func TestParseSceneMeta_InvalidOrder(t *testing.T) {
	content := `-----META-END-----
kv
-----META-START-----`
	meta, body := parseSceneMeta(content)
	if meta != nil {
		t.Errorf("顺序错误时 meta 应为 nil, got %v", meta)
	}
	if body != content {
		t.Error("body 应为原文")
	}
}

// 只有 START 没有 END
func TestParseSceneMeta_MissingEnd(t *testing.T) {
	content := `-----META-START-----
summary: abc`
	meta, _ := parseSceneMeta(content)
	if meta != nil {
		t.Errorf("缺 END 时 meta 应为 nil")
	}
}

// 畸形 KV（无冒号）应被忽略
func TestParseSceneMeta_MalformedLine(t *testing.T) {
	content := `-----META-START-----
valid: 1
invalid-no-colon
another: 2
-----META-END-----

body`
	meta, _ := parseSceneMeta(content)
	if len(meta) != 2 {
		t.Errorf("meta len = %d, want 2 (畸形行应忽略)", len(meta))
	}
	if meta["valid"] != "1" || meta["another"] != "2" {
		t.Errorf("meta = %v", meta)
	}
}

// --- flattenVDBDocuments ---

func TestFlattenVDBDocuments_Valid(t *testing.T) {
	docs := []map[string]any{
		{
			"Id": "doc-1",
			"Fields": []any{
				map[string]any{"Name": "text", "Value": "hello"},
				map[string]any{"Name": "type", "Value": "persona"},
			},
		},
	}
	result := flattenVDBDocuments(docs)
	if len(result) != 1 {
		t.Fatalf("len = %d, want 1", len(result))
	}
	if result[0]["id"] != "doc-1" {
		t.Errorf("id = %v", result[0]["id"])
	}
	if result[0]["text"] != "hello" {
		t.Errorf("text = %v", result[0]["text"])
	}
	if result[0]["type"] != "persona" {
		t.Errorf("type = %v", result[0]["type"])
	}
}

// 无 Fields 的文档应正确处理
func TestFlattenVDBDocuments_NoFields(t *testing.T) {
	docs := []map[string]any{
		{"Id": "doc-empty"},
	}
	result := flattenVDBDocuments(docs)
	if len(result) != 1 {
		t.Fatalf("len = %d, want 1", len(result))
	}
	if result[0]["id"] != "doc-empty" {
		t.Errorf("id = %v", result[0]["id"])
	}
}

// 无 Name 的字段应被跳过
func TestFlattenVDBDocuments_SkipEmptyName(t *testing.T) {
	docs := []map[string]any{
		{
			"Id": "doc",
			"Fields": []any{
				map[string]any{"Name": "", "Value": "should-skip"},
				map[string]any{"Name": "valid", "Value": "keep"},
			},
		},
	}
	result := flattenVDBDocuments(docs)
	if _, has := result[0][""]; has {
		t.Error("空 Name 字段应被跳过")
	}
	if result[0]["valid"] != "keep" {
		t.Error("valid 字段应保留")
	}
}

// Fields 类型不对（不是 []any）应被跳过
func TestFlattenVDBDocuments_NonSliceFields(t *testing.T) {
	docs := []map[string]any{
		{
			"Id":     "doc",
			"Fields": "not-a-slice",
		},
	}
	result := flattenVDBDocuments(docs)
	if len(result) != 1 {
		t.Fatalf("len = %d", len(result))
	}
	// 只应有 id
	if len(result[0]) != 1 {
		t.Errorf("应只有 id，got %v", result[0])
	}
}

// Fields 内某项不是 map 应被跳过
func TestFlattenVDBDocuments_NonMapField(t *testing.T) {
	docs := []map[string]any{
		{
			"Id": "doc",
			"Fields": []any{
				"not-a-map", // 非 map 应跳过
				map[string]any{"Name": "ok", "Value": "v"},
			},
		},
	}
	result := flattenVDBDocuments(docs)
	if result[0]["ok"] != "v" {
		t.Error("有效字段应保留")
	}
}

func TestFlattenVDBDocuments_Empty(t *testing.T) {
	result := flattenVDBDocuments(nil)
	if result == nil {
		t.Error("nil input 应返回空 slice 而非 nil")
	}
	if len(result) != 0 {
		t.Errorf("len = %d, want 0", len(result))
	}
}

// --- formatProDocuments ---

func TestFormatProDocuments_Persona(t *testing.T) {
	docs := []map[string]any{
		{"id": "p1", "content": "persona-content-value"},
	}
	result := formatProDocuments(docs, "persona")
	if len(result) != 1 {
		t.Fatalf("len = %d", len(result))
	}
	if result[0]["id"] != "p1" {
		t.Errorf("id = %v", result[0]["id"])
	}
	if result[0]["content"] != "persona-content-value" {
		t.Errorf("content = %v", result[0]["content"])
	}
}

func TestFormatProDocuments_SceneWithMeta(t *testing.T) {
	content := `-----META-START-----
summary: 场景摘要
heat: 10
created: 2026-04-01T00:00:00Z
updated: 2026-04-15T00:00:00Z
-----META-END-----

场景正文内容。`
	docs := []map[string]any{
		{"id": "s1", "content": content, "filename": "scene.md"},
	}
	result := formatProDocuments(docs, "scene")
	doc := result[0]
	if doc["fileName"] != "scene.md" {
		t.Errorf("fileName = %v", doc["fileName"])
	}
	if doc["summary"] != "场景摘要" {
		t.Errorf("summary = %v", doc["summary"])
	}
	if doc["heat"] != "10" {
		t.Errorf("heat = %v", doc["heat"])
	}
	body, _ := doc["body"].(string)
	if !strings.Contains(body, "场景正文内容") {
		t.Errorf("body 应包含正文，got %q", body)
	}
}

func TestFormatProDocuments_SceneNoMeta(t *testing.T) {
	docs := []map[string]any{
		{"id": "s2", "content": "pure body without meta", "filename": "raw.md"},
	}
	result := formatProDocuments(docs, "scene")
	doc := result[0]
	// 无 META 时 summary 应不存在
	if _, has := doc["summary"]; has {
		t.Errorf("summary 应不存在，got %v", doc["summary"])
	}
	if doc["body"] != "pure body without meta" {
		t.Errorf("body = %v", doc["body"])
	}
}

func TestFormatProDocuments_Memory(t *testing.T) {
	docs := []map[string]any{
		{
			"id":            "m1",
			"text":          "memory-text",
			"type":          "episodic",
			"priority":      "70",
			"scene_name":    "work",
			"timestamp_str": "2026-04-10T10:00:00Z",
		},
	}
	result := formatProDocuments(docs, "memory")
	doc := result[0]
	if doc["content"] != "memory-text" {
		t.Errorf("content = %v, want memory-text", doc["content"])
	}
	if doc["type"] != "episodic" {
		t.Errorf("type = %v", doc["type"])
	}
	if doc["priority"] != "70" {
		t.Errorf("priority = %v", doc["priority"])
	}
	if doc["timestamp"] != "2026-04-10T10:00:00Z" {
		t.Errorf("timestamp = %v", doc["timestamp"])
	}
}

// recordType 以 "memory" 开头的派生类型也应命中 memory 分支
func TestFormatProDocuments_MemoryPrefix(t *testing.T) {
	docs := []map[string]any{
		{"id": "m-custom", "text": "mem-text"},
	}
	result := formatProDocuments(docs, "memory_all")
	if result[0]["content"] != "mem-text" {
		t.Errorf("memory_all 应命中 memory 分支")
	}
}

func TestFormatProDocuments_ConversationValidMS(t *testing.T) {
	docs := []map[string]any{
		{
			"id":             "c1",
			"role":           "user",
			"message_text":   "hello",
			"session_key":    "agent:main:main",
			"session_id":     "sess-001",
			"recorded_at_ms": "1776440405086",
			"timestamp":      "2026-04-10",
		},
	}
	result := formatProDocuments(docs, "conversation")
	doc := result[0]
	if doc["role"] != "user" {
		t.Errorf("role = %v", doc["role"])
	}
	if doc["content"] != "hello" {
		t.Errorf("content = %v", doc["content"])
	}
	if doc["sessionKey"] != "agent:main:main" {
		t.Errorf("sessionKey = %v", doc["sessionKey"])
	}
	if doc["sessionId"] != "sess-001" {
		t.Errorf("sessionId = %v", doc["sessionId"])
	}
	// recorded_at_ms 有效 → ISO8601 格式
	recordedAt, _ := doc["recordedAt"].(string)
	if !strings.Contains(recordedAt, "T") {
		t.Errorf("recordedAt 应转为 ISO8601, got %q", recordedAt)
	}
}

// 非数字的 recorded_at_ms 应原样保留
func TestFormatProDocuments_ConversationInvalidMS(t *testing.T) {
	docs := []map[string]any{
		{
			"id":             "c2",
			"recorded_at_ms": "not-a-number",
		},
	}
	result := formatProDocuments(docs, "conversation")
	if result[0]["recordedAt"] != "not-a-number" {
		t.Errorf("非数字的 ms 应原样保留，got %v", result[0]["recordedAt"])
	}
}

// 未知 type → 透传所有字段
func TestFormatProDocuments_UnknownType(t *testing.T) {
	docs := []map[string]any{
		{"id": "u", "foo": "bar", "baz": 123},
	}
	result := formatProDocuments(docs, "unknown-type")
	if !reflect.DeepEqual(result[0], docs[0]) {
		t.Errorf("unknown type 应透传原文档, got %v", result[0])
	}
}

// 空输入应返回非 nil 空 slice
func TestFormatProDocuments_Empty(t *testing.T) {
	result := formatProDocuments(nil, "persona")
	if result == nil {
		t.Error("nil 输入应返回空 slice")
	}
	if len(result) != 0 {
		t.Errorf("len = %d, want 0", len(result))
	}
}
