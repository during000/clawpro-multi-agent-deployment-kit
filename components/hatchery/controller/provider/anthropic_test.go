package provider

import (
	"encoding/json"
	"testing"
)

func TestAnthropicToOpenAIIncludesCacheTokensInPromptTotals(t *testing.T) {
	ar := &anthropicResponse{
		ID:         "msg-cache",
		Role:       "assistant",
		StopReason: "end_turn",
		Content: []anthropicContentBlock{
			{Type: "text", Text: "ok"},
		},
	}
	ar.Usage.InputTokens = 100
	ar.Usage.CacheReadInputTokens = 30
	ar.Usage.CacheCreationInputTokens = 20
	ar.Usage.OutputTokens = 10

	body, err := anthropicToOpenAI(ar)
	if err != nil {
		t.Fatalf("anthropicToOpenAI() error = %v", err)
	}

	var got struct {
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
			PromptDetails    struct {
				CachedTokens        int `json:"cached_tokens"`
				CacheCreationTokens int `json:"cache_creation_tokens"`
			} `json:"prompt_tokens_details"`
			CacheReadInputTokens     int `json:"cache_read_input_tokens"`
			CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}

	if got.Usage.PromptTokens != 150 || got.Usage.CompletionTokens != 10 || got.Usage.TotalTokens != 160 {
		t.Fatalf("usage totals = prompt %d completion %d total %d, want prompt 150 completion 10 total 160", got.Usage.PromptTokens, got.Usage.CompletionTokens, got.Usage.TotalTokens)
	}
	if got.Usage.PromptDetails.CachedTokens != 30 || got.Usage.PromptDetails.CacheCreationTokens != 20 {
		t.Fatalf("prompt details = cached %d creation %d, want cached 30 creation 20", got.Usage.PromptDetails.CachedTokens, got.Usage.PromptDetails.CacheCreationTokens)
	}
	if got.Usage.CacheReadInputTokens != 30 || got.Usage.CacheCreationInputTokens != 20 {
		t.Fatalf("anthropic cache fields = read %d creation %d, want read 30 creation 20", got.Usage.CacheReadInputTokens, got.Usage.CacheCreationInputTokens)
	}
}

func TestApplyAnthropicUsageIncludesCacheTokensInPromptTotals(t *testing.T) {
	usage := &Usage{}
	applyAnthropicUsage(usage, map[string]interface{}{
		"input_tokens":                float64(100),
		"cache_read_input_tokens":     float64(30),
		"cache_creation_input_tokens": float64(20),
	})
	applyAnthropicUsage(usage, map[string]interface{}{
		"output_tokens": float64(10),
	})

	if usage.PromptTokens != 150 || usage.CompletionTokens != 10 || usage.TotalTokens != 160 {
		t.Fatalf("usage totals = prompt %d completion %d total %d, want prompt 150 completion 10 total 160", usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)
	}
	if usage.PromptCacheReadTokens != 30 || usage.PromptCacheWriteTokens != 20 {
		t.Fatalf("cache tokens = read %d write %d, want read 30 write 20", usage.PromptCacheReadTokens, usage.PromptCacheWriteTokens)
	}

	applyAnthropicUsage(usage, map[string]interface{}{
		"input_tokens":  float64(110),
		"output_tokens": float64(11),
	})
	if usage.PromptTokens != 160 || usage.TotalTokens != 171 {
		t.Fatalf("updated usage totals = prompt %d total %d, want prompt 160 total 171", usage.PromptTokens, usage.TotalTokens)
	}
}
