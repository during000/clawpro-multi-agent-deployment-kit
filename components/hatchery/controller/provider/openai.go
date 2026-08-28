package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
)

// OpenAIProvider handles any OpenAI-compatible endpoint.
type OpenAIProvider struct{}

var gptVersionRx = regexp.MustCompile(`(?i)^gpt-(\d+)`)

// needsMaxCompletionTokens 判断模型是否用 max_completion_tokens 代替 max_tokens。
// GPT-5+（gpt-6/7…自动覆盖）和 o-series（o1/o3/o4…）。
func needsMaxCompletionTokens(model string) bool {
	lower := strings.ToLower(model)
	if m := gptVersionRx.FindStringSubmatch(lower); m != nil {
		v, _ := strconv.Atoi(m[1])
		return v >= 5
	}
	return len(lower) > 1 && lower[0] == 'o' && lower[1] >= '0' && lower[1] <= '9'
}

// normalizeMaxTokensParam 按模型统一 max_tokens / max_completion_tokens 参数名。
func normalizeMaxTokensParam(req map[string]interface{}, model string) {
	if needsMaxCompletionTokens(model) {
		if mt, ok := req["max_tokens"]; ok {
			if _, ok := req["max_completion_tokens"]; !ok {
				req["max_completion_tokens"] = mt
			}
			delete(req, "max_tokens")
		}
		return
	}
	if mct, ok := req["max_completion_tokens"]; ok {
		if _, ok := req["max_tokens"]; !ok {
			req["max_tokens"] = mct
		}
		delete(req, "max_completion_tokens")
	}
}

// sanitizeUsage finds the "usage" object in a response JSON body and replaces
// any null token counts with 0. Returns the original body unchanged if parsing
// fails or no usage field is present.
func sanitizeUsage(respBody []byte) []byte {
	var resp map[string]interface{}
	if json.Unmarshal(respBody, &resp) != nil {
		return respBody
	}
	usageRaw, ok := resp["usage"]
	if !ok || usageRaw == nil {
		return respBody
	}
	usage, ok := usageRaw.(map[string]interface{})
	if !ok {
		return respBody
	}

	changed := false
	tokenFields := []string{"prompt_tokens", "completion_tokens", "total_tokens"}
	for _, field := range tokenFields {
		if v, exists := usage[field]; exists && v == nil {
			usage[field] = 0
			changed = true
		}
	}
	if !changed {
		return respBody
	}

	out, err := json.Marshal(resp)
	if err != nil {
		return respBody
	}
	return out
}

func (p *OpenAIProvider) baseURL(apiBase string) string {
	return strings.TrimRight(apiBase, "/")
}

func (p *OpenAIProvider) ChatCompletion(ctx context.Context, apiKey, apiBase, model string, reqBody []byte, customHeaders map[string]string) (*CompletionResult, int, error) {
	// Inject the real model name into the request body.
	var req map[string]interface{}
	if err := json.Unmarshal(reqBody, &req); err != nil {
		return nil, 0, hcommon.I18nRichError(err, i18n.MsgProviderParseRequest)
	}
	req["model"] = model
	normalizeMaxTokensParam(req, model)
	body, err := json.Marshal(req)
	if err != nil {
		return nil, 0, hcommon.I18nRichError(err, i18n.MsgProviderMarshalRequest)
	}

	ssrfEnabled := hcommon.IsSSRFEnabledInSecurityPolicies(ctx)
	url := p.baseURL(apiBase) + "/chat/completions"
	if ssrfEnabled {
		if err := validateOutboundURL(ctx, url); err != nil {
			return nil, 0, err
		}
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, 0, hcommon.I18nRichError(err, i18n.MsgProviderCreateRequest)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	// 将用户自定义请求头应用到请求中
	applyCustomHeaders(httpReq, customHeaders)

	client := newSSRFSafeHTTPClient(5*time.Minute, ssrfEnabled)
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, 0, hcommon.I18nRichError(err, i18n.MsgProviderUpstreamRequest)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, hcommon.I18nRichError(err, i18n.MsgProviderReadResponse)
	}

	// Sanitize null usage values from non-standard endpoints.
	if resp.StatusCode == http.StatusOK {
		respBody = sanitizeUsage(respBody)
	}
	return &CompletionResult{Body: respBody, Usage: extractOpenAIUsage(string(respBody), nil)}, resp.StatusCode, nil
}

func (p *OpenAIProvider) ChatCompletionStream(ctx context.Context, apiKey, apiBase, model string, reqBody []byte, w http.ResponseWriter, flusher http.Flusher, customHeaders map[string]string) (*StreamResult, int, error) {
	// Inject model name and stream_options.
	var req map[string]interface{}
	if err := json.Unmarshal(reqBody, &req); err != nil {
		return nil, 0, hcommon.I18nRichError(err, i18n.MsgProviderParseRequest)
	}
	req["model"] = model
	req["stream"] = true
	req["stream_options"] = map[string]interface{}{"include_usage": true}
	normalizeMaxTokensParam(req, model)
	body, err := json.Marshal(req)
	if err != nil {
		return nil, 0, hcommon.I18nRichError(err, i18n.MsgProviderMarshalRequest)
	}

	ssrfEnabled := hcommon.IsSSRFEnabledInSecurityPolicies(ctx)
	url := p.baseURL(apiBase) + "/chat/completions"
	if ssrfEnabled {
		if err := validateOutboundURL(ctx, url); err != nil {
			return nil, 0, err
		}
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, 0, hcommon.I18nRichError(err, i18n.MsgProviderCreateRequest)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	// 将用户自定义请求头应用到请求中
	applyCustomHeaders(httpReq, customHeaders)

	client := newSSRFSafeHTTPClient(5*time.Minute, ssrfEnabled)
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, 0, hcommon.I18nRichError(err, i18n.MsgProviderUpstreamRequest)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Forward the error response as-is.
		respBody, _ := io.ReadAll(resp.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		w.Write(respBody)
		return nil, resp.StatusCode, nil
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var usage *Usage
	hasToolCalls := false
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Fprintf(w, "%s\n", line)
		flusher.Flush()

		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data != "[DONE]" {
				usage = extractOpenAIUsage(data, usage)
				// 精确检测流式响应中是否包含有效的 tool_calls（用于 Agent Loop 宽限期判断）
				// 注意：不能用 strings.Contains(data, "tool_calls")，因为某些 LLM（如 DeepSeek）
				// 在纯文本回复中也会返回 "tool_calls":null 或 "tool_calls":[]，导致误匹配。
				if !hasToolCalls {
					hasToolCalls = detectToolCallsInStreamChunk(data)
				}
			}
		}
	}
	return &StreamResult{Usage: usage, HasToolCalls: hasToolCalls}, http.StatusOK, nil
}

// detectToolCallsInStreamChunk 精确检测单个 SSE chunk 中是否包含有效的 tool_calls。
// 通过 JSON 解析确认 choices[].delta.tool_calls 是非空数组，避免误匹配
// "tool_calls":null 或 AI 回复文本中包含 "tool_calls" 字样的情况。
func detectToolCallsInStreamChunk(data string) bool {
	var chunk struct {
		Choices []struct {
			Delta struct {
				ToolCalls []json.RawMessage `json:"tool_calls"`
			} `json:"delta"`
			FinishReason *string `json:"finish_reason"`
		} `json:"choices"`
	}
	if json.Unmarshal([]byte(data), &chunk) != nil {
		return false
	}
	for _, c := range chunk.Choices {
		// 检查 delta.tool_calls 是否为非空数组
		if len(c.Delta.ToolCalls) > 0 {
			return true
		}
		// 检查 finish_reason 是否为 "tool_calls"（某些 provider 使用此值）
		if c.FinishReason != nil && *c.FinishReason == "tool_calls" {
			return true
		}
	}
	return false
}

// extractOpenAIUsage parses OpenAI-compatible usage from streaming chunks or
// non-streaming response bodies.
func extractOpenAIUsage(data string, prev *Usage) *Usage {
	var chunk struct {
		Usage *struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
			CachedTokens     int `json:"cached_tokens"` // Moonshot/Kimi.
			PromptDetails    struct {
				CachedTokens        int `json:"cached_tokens"`         // OpenAI Chat Completions.
				CacheWriteTokens    int `json:"cache_write_tokens"`    // OpenRouter.
				CacheCreationTokens int `json:"cache_creation_tokens"` // LiteLLM.
			} `json:"prompt_tokens_details"`
			InputDetails struct {
				CachedTokens int `json:"cached_tokens"` // OpenAI Responses.
			} `json:"input_tokens_details"`
			PromptCacheHitTokens int `json:"prompt_cache_hit_tokens"` // DeepSeek.
		} `json:"usage"`
	}
	if json.Unmarshal([]byte(data), &chunk) == nil && chunk.Usage != nil {
		cacheRead := firstPositiveInt(
			chunk.Usage.InputDetails.CachedTokens,
			chunk.Usage.PromptDetails.CachedTokens,
			chunk.Usage.CachedTokens,
			chunk.Usage.PromptCacheHitTokens,
		)
		cacheWrite := firstPositiveInt(
			chunk.Usage.PromptDetails.CacheWriteTokens,
			chunk.Usage.PromptDetails.CacheCreationTokens,
		)
		return &Usage{
			PromptTokens:           chunk.Usage.PromptTokens,
			CompletionTokens:       chunk.Usage.CompletionTokens,
			TotalTokens:            chunk.Usage.TotalTokens,
			PromptCacheReadTokens:  cacheRead,
			PromptCacheWriteTokens: cacheWrite,
		}
	}
	return prev
}

func firstPositiveInt(values ...int) int {
	for _, v := range values {
		if v > 0 {
			return v
		}
	}
	return 0
}

// CheckConnectivityWithChat probes an OpenAI-compatible endpoint by issuing a
// minimal POST /chat/completions request (one user message, max_tokens=1).
//
// This probe validates both network reachability, the apiKey, and that the
// supplied model name is actually invocable.
//
// On success it returns the round-trip latency and a nil error. On failure
// it returns a *ConnectivityError wrapping one of the Err* sentinels defined
// in provider.go; callers can use errors.Is to classify the failure.
func (p *OpenAIProvider) CheckConnectivityWithChat(ctx context.Context, apiKey, apiBase, model string) (time.Duration, error) {
	req := map[string]interface{}{
		"model": model,
		"messages": []map[string]interface{}{
			{"role": "user", "content": "hi"},
		},
		"max_tokens": 1,
		"stream":     false,
	}
	normalizeMaxTokensParam(req, model)
	body, err := json.Marshal(req)
	if err != nil {
		return 0, &ConnectivityError{Kind: ErrNetworkUnreachable, Cause: err}
	}

	ssrfEnabled := hcommon.IsSSRFEnabledInSecurityPolicies(ctx)
	url := p.baseURL(apiBase) + "/chat/completions"
	if ssrfEnabled {
		if err := validateOutboundURL(ctx, url); err != nil {
			return 0, &ConnectivityError{Kind: ErrNetworkUnreachable, Cause: err}
		}
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return 0, &ConnectivityError{Kind: ErrNetworkUnreachable, Cause: err}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := newSSRFSafeHTTPClient(15*time.Second, ssrfEnabled)
	start := time.Now()
	resp, err := client.Do(httpReq)
	latency := time.Since(start)
	if err != nil {
		return latency, &ConnectivityError{Kind: ErrNetworkUnreachable, Cause: err}
	}
	defer resp.Body.Close()

	if kind := classifyHTTPStatus(resp.StatusCode); kind != nil {
		snippet := readBodySnippet(resp.Body, 512)
		return latency, &ConnectivityError{
			Kind:       kind,
			StatusCode: resp.StatusCode,
			Snippet:    snippet,
		}
	}
	// Drain body so the connection can be reused.
	_, _ = io.Copy(io.Discard, resp.Body)
	return latency, nil
}
