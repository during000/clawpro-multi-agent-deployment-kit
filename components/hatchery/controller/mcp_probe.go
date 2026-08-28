package controller

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
)

// ── MCP 连通性探测器 ─────────────────────────────────────────────────────────

// sseEvent SSE 流中读取到的事件
type sseEvent struct {
	data string // 格式: "eventType\x00dataContent"
	err  error
}

// McpProbeInput 探测输入
type McpProbeInput struct {
	ServiceID     string
	TransportType string
	ConfigJSON    string
}

// McpProbeResult 探测结果
type McpProbeResult struct {
	ServiceID        string   `json:"service_id"`
	ConnectionStatus string   `json:"connection_status"` // connected / failed / unsupported / unconfigured
	Tools            []string `json:"tools"`
	ErrorCode        string   `json:"error_code,omitempty"`
	Error            string   `json:"error,omitempty"`
}

// McpProber 全局探测器（进程唯一实例）
type McpProber struct {
	client   *http.Client
	sem      chan struct{} // 并发信号量
	inflight sync.Map      // 实例级防重：instanceID -> bool
}

// mcpProber 全局单例
var mcpProber = &McpProber{
	client: &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 2,
			IdleConnTimeout:     5 * time.Second,
			DisableKeepAlives:   true,
			DialContext: (&net.Dialer{
				Timeout: 5 * time.Second,
			}).DialContext,
		},
	},
	sem: make(chan struct{}, 5), // 最多 5 个并发探测
}

// placeholderRegex 匹配 <xxx> 格式的占位符
var placeholderRegex = regexp.MustCompile(`<[a-zA-Z_][a-zA-Z0-9_]*>`)

// containsPlaceholder 检测 config_json 中是否包含 <xxx> 格式的占位符
func containsPlaceholder(configJSON string) bool {
	return placeholderRegex.MatchString(configJSON)
}

// isPrivateIP 检查 host 是否为内网地址（SSRF 防护）
func isPrivateIP(host string) bool {
	// 去除端口
	h, _, err := net.SplitHostPort(host)
	if err != nil {
		h = host
	}

	ip := net.ParseIP(h)
	if ip == nil {
		// 尝试解析域名
		addrs, err := net.LookupIP(h)
		if err != nil || len(addrs) == 0 {
			return true // DNS 解析失败时 fail-closed（拒绝探测）
		}
		// 检查所有解析出的 IP（防止多 A 记录绕过）
		for _, addr := range addrs {
			if isPrivateAddr(addr) {
				return true
			}
		}
		return false
	}

	return isPrivateAddr(ip)
}

// isPrivateAddr 检查单个 IP 是否为内网地址
func isPrivateAddr(ip net.IP) bool {
	// 检查内网段
	privateRanges := []struct {
		network *net.IPNet
	}{
		{network: mustParseCIDR("127.0.0.0/8")},
		{network: mustParseCIDR("10.0.0.0/8")},
		{network: mustParseCIDR("172.16.0.0/12")},
		{network: mustParseCIDR("192.168.0.0/16")},
		{network: mustParseCIDR("169.254.0.0/16")},
		{network: mustParseCIDR("::1/128")},
		{network: mustParseCIDR("fc00::/7")},
	}

	for _, r := range privateRanges {
		if r.network.Contains(ip) {
			return true
		}
	}
	return false
}

func mustParseCIDR(s string) *net.IPNet {
	_, network, err := net.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	return network
}

// TryAcquireInstance 尝试获取实例级探测锁
func (p *McpProber) TryAcquireInstance(instanceID uint) bool {
	_, loaded := p.inflight.LoadOrStore(instanceID, true)
	return !loaded // 未被占用时返回 true
}

// ReleaseInstance 释放实例级探测锁
func (p *McpProber) ReleaseInstance(instanceID uint) {
	p.inflight.Delete(instanceID)
}

// Probe 批量探测多个 MCP 的连通性
func (p *McpProber) Probe(ctx context.Context, inputs []McpProbeInput) []McpProbeResult {
	results := make([]McpProbeResult, len(inputs))
	var wg sync.WaitGroup

	for i, input := range inputs {
		wg.Add(1)
		go func(idx int, in McpProbeInput) {
			defer wg.Done()

			// 获取信号量
			select {
			case p.sem <- struct{}{}:
				defer func() { <-p.sem }()
			case <-ctx.Done():
				results[idx] = McpProbeResult{
					ServiceID:        in.ServiceID,
					ConnectionStatus: "failed",
					ErrorCode:        probeCodeTimeout,
					Error:            probeErrTimeout,
				}
				return
			}

			results[idx] = p.probeOne(ctx, in)
		}(i, input)
	}

	wg.Wait()
	return results
}

// ── 面向用户的统一错误码与文案 ────────────────────────────────────────────────
const (
	probeCodeUnconfigured = "unconfigured"
	probeCodeUnsupported  = "unsupported"
	probeCodeConfigFmt    = "config_invalid"
	probeCodeBlocked      = "blocked"
	probeCodeConnect      = "connect_failed"
	probeCodeProtocol     = "protocol_error"
	probeCodeTimeout      = "timeout"
)

const (
	probeErrUnconfigured = "配置中包含未填写的参数"
	probeErrUnsupported  = "该类型不支持远程探测"
	probeErrConfigFmt    = "配置格式错误"
	probeErrBlocked      = "不支持探测内网地址"
	probeErrConnect      = "连接失败"
	probeErrProtocol     = "连接成功，但服务响应异常"
	probeErrTimeout      = "探测超时"
)

// probeError 同时携带 error_code 和 error 文案
type probeError struct {
	Code    string
	Message string
}

var (
	peUnconfigured = probeError{probeCodeUnconfigured, probeErrUnconfigured}
	peUnsupported  = probeError{probeCodeUnsupported, probeErrUnsupported}
	peConfigFmt    = probeError{probeCodeConfigFmt, probeErrConfigFmt}
	peBlocked      = probeError{probeCodeBlocked, probeErrBlocked}
	peConnect      = probeError{probeCodeConnect, probeErrConnect}
	peProtocol     = probeError{probeCodeProtocol, probeErrProtocol}
	peTimeout      = probeError{probeCodeTimeout, probeErrTimeout}
)

// classifyProbeError 将内部探测错误归类为面向用户的统一错误
func classifyProbeError(ctx context.Context, err error) probeError {
	if ctx.Err() != nil {
		return peTimeout
	}

	msg := err.Error()
	switch {
	case strings.Contains(msg, "connection refused"),
		strings.Contains(msg, "no such host"),
		strings.Contains(msg, "i/o timeout"),
		strings.Contains(msg, "SSE 连接失败"),
		strings.Contains(msg, "HTTP "):
		return peConnect
	case strings.Contains(msg, "内网地址"):
		return peBlocked
	default:
		return peProtocol
	}
}

// probeOne 探测单个 MCP
func (p *McpProber) probeOne(ctx context.Context, input McpProbeInput) McpProbeResult {
	result := McpProbeResult{
		ServiceID: input.ServiceID,
		Tools:     []string{},
	}

	setErr := func(status string, pe probeError) McpProbeResult {
		result.ConnectionStatus = status
		result.ErrorCode = pe.Code
		result.Error = pe.Message
		return result
	}

	// 预检 1：占位符检测
	if containsPlaceholder(input.ConfigJSON) {
		return setErr("unconfigured", peUnconfigured)
	}

	// 预检 2：STDIO 不支持远程探测
	if input.TransportType == "stdio" {
		return setErr("unsupported", peUnsupported)
	}

	// 解析 config_json
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(input.ConfigJSON), &config); err != nil {
		return setErr("failed", peConfigFmt)
	}

	// 获取 URL
	urlStr, _ := config["url"].(string)
	if urlStr == "" {
		return setErr("failed", peConfigFmt)
	}

	// SSRF 防护
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return setErr("failed", peConfigFmt)
	}
	if isPrivateIP(parsed.Host) {
		return setErr("failed", peBlocked)
	}

	// 提取 headers
	headers := extractHeaders(config)

	// 按 transport_type 选择探测方式
	var tools []string
	switch input.TransportType {
	case "streamable-http":
		tools, err = p.probeStreamableHTTP(ctx, urlStr, headers)
	case "sse":
		tools, err = p.probeSSE(ctx, urlStr, headers)
	default:
		return setErr("unsupported", peUnsupported)
	}

	if err != nil {
		return setErr("failed", classifyProbeError(ctx, err))
	}

	result.ConnectionStatus = "connected"
	result.Tools = tools
	return result
}

// extractHeaders 从 config 中提取 headers map
func extractHeaders(config map[string]interface{}) map[string]string {
	headers := make(map[string]string)
	if h, ok := config["headers"]; ok {
		if hm, ok := h.(map[string]interface{}); ok {
			for k, v := range hm {
				if vs, ok := v.(string); ok {
					headers[k] = vs
				}
			}
		}
	}
	return headers
}

// probeStreamableHTTP 通过 streamable-http 方式探测 MCP
func (p *McpProber) probeStreamableHTTP(ctx context.Context, mcpURL string, headers map[string]string) ([]string, error) {
	// 1. 发送 initialize 请求
	initBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]interface{}{
				"name":    "hatchery-probe",
				"version": "1.0.0",
			},
		},
	}

	initResp, err := p.doJSONRPC(ctx, mcpURL, headers, initBody)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgMcpProbeInitializeFailed)
	}

	// 检查 initialize 响应是否成功
	if _, ok := initResp["result"]; !ok {
		if errObj, ok := initResp["error"]; ok {
			return nil, hcommon.I18nError(i18n.MsgMcpProbeInitializeError, errObj)
		}
		return nil, hcommon.I18nError(i18n.MsgMcpProbeInitializeInvalidResp)
	}

	// 2. 发送 initialized 通知
	notifyBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	}
	// 通知不需要响应，忽略错误
	p.doJSONRPC(ctx, mcpURL, headers, notifyBody)

	// 3. 发送 tools/list 请求
	toolsBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
		"params":  map[string]interface{}{},
	}

	toolsResp, err := p.doJSONRPC(ctx, mcpURL, headers, toolsBody)
	if err != nil {
		// initialize 成功但 tools/list 失败，仍视为连接成功
		return []string{}, nil
	}

	// 解析工具列表
	return parseToolsList(toolsResp), nil
}

// probeSSE 通过 SSE 方式探测 MCP
// SSE transport 流程：
//  1. GET /sse → 建立 SSE 连接，收到 endpoint 事件
//  2. POST endpoint → 发送 JSON-RPC，服务返回 200/202
//  3. 如果 POST 返回 200 + 响应体 → 直接解析（兼容 streamable-http 降级）
//  4. 如果 POST 返回 202 → 从 SSE 流中读取 JSON-RPC 响应
func (p *McpProber) probeSSE(ctx context.Context, mcpURL string, headers map[string]string) ([]string, error) {
	// 使用独立 client（更长超时，不自动关闭连接）
	sseClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 1,
			IdleConnTimeout:     30 * time.Second,
			DialContext: (&net.Dialer{
				Timeout: 5 * time.Second,
			}).DialContext,
		},
	}

	// Step 1: GET 建立 SSE 连接
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mcpURL, nil)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgMcpProbeCreateSSERequestFailed)
	}
	req.Header.Set("Accept", "text/event-stream")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := sseClient.Do(req)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgMcpProbeSSEConnectFailed)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, hcommon.I18nError(i18n.MsgMcpProbeSSEHTTPError, resp.StatusCode)
	}

	// Step 2: 从 SSE 流读取 endpoint URL
	// 使用 goroutine + channel 异步读取 SSE 流，同时支持后续响应读取
	sseCh := make(chan sseEvent, 16)

	go func() {
		defer close(sseCh)
		scanner := bufio.NewScanner(io.LimitReader(resp.Body, 512*1024))
		var currentEvent string
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
			}
			line := scanner.Text()
			if strings.HasPrefix(line, "event:") {
				currentEvent = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
				continue
			}
			if strings.HasPrefix(line, "data:") {
				data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if data != "" {
					select {
					case sseCh <- sseEvent{data: currentEvent + "\x00" + data}:
					case <-ctx.Done():
						return
					}
				}
				continue
			}
			if line == "" {
				currentEvent = ""
			}
		}
		if err := scanner.Err(); err != nil {
			select {
			case sseCh <- sseEvent{err: err}:
			case <-ctx.Done():
			}
		}
	}()

	// 等待 endpoint 事件
	var endpointURL string
	for {
		select {
		case <-ctx.Done():
			return nil, hcommon.I18nRichError(ctx.Err(), i18n.MsgMcpProbeSSEEndpointTimeout)
		case ev, ok := <-sseCh:
			if !ok {
				return nil, hcommon.I18nError(i18n.MsgMcpProbeSSENoEndpoint)
			}
			if ev.err != nil {
				return nil, hcommon.I18nRichError(ev.err, i18n.MsgMcpProbeSSEReadError)
			}
			// 解析 event\x00data 格式
			parts := strings.SplitN(ev.data, "\x00", 2)
			if len(parts) < 2 {
				continue
			}
			eventType, data := parts[0], parts[1]
			if eventType == "endpoint" || eventType == "" {
				resolved, err := resolveEndpointURL(data, mcpURL)
				if err == nil {
					endpointURL = resolved
				}
			}
		}
		if endpointURL != "" {
			break
		}
	}

	// SSRF 校验 endpoint
	endpointParsed, err := url.Parse(endpointURL)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgMcpProbeSSEEndpointInvalidURL)
	}
	if isPrivateIP(endpointParsed.Host) {
		return nil, hcommon.I18nError(i18n.MsgMcpProbeSSEEndpointPrivate)
	}

	// Step 3: POST initialize
	initBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]interface{}{
				"name":    "hatchery-probe",
				"version": "1.0.0",
			},
		},
	}

	initResp, err := p.postSSE(ctx, sseClient, endpointURL, headers, initBody, sseCh)
	if err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgMcpProbeInitializeFailed)
	}

	if _, ok := initResp["result"]; !ok {
		if errObj, ok := initResp["error"]; ok {
			return nil, hcommon.I18nError(i18n.MsgMcpProbeInitializeError, errObj)
		}
		return nil, hcommon.I18nError(i18n.MsgMcpProbeInitializeInvalidResp)
	}

	// Step 4: POST notifications/initialized（通知，不等响应）
	notifyBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	}
	p.postNoWait(ctx, sseClient, endpointURL, headers, notifyBody)

	// Step 5: POST tools/list
	toolsBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
		"params":  map[string]interface{}{},
	}

	toolsResp, err := p.postSSE(ctx, sseClient, endpointURL, headers, toolsBody, sseCh)
	if err != nil {
		// initialize 成功但 tools/list 失败，仍视为连接成功
		return []string{}, nil
	}

	return parseToolsList(toolsResp), nil
}

// postSSE 发送 JSON-RPC POST，若返回 202 则从 SSE channel 等待响应
func (p *McpProber) postSSE(ctx context.Context, client *http.Client, endpoint string, headers map[string]string, body interface{}, sseCh <-chan sseEvent) (map[string]interface{}, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// 直接从 POST 响应体读取（streamable-http 兼容模式）
		limited := io.LimitReader(resp.Body, 512*1024)
		contentType := resp.Header.Get("Content-Type")
		if strings.Contains(contentType, "text/event-stream") {
			return parseSSEResponse(ctx, limited)
		}
		var result map[string]interface{}
		if err := json.NewDecoder(limited).Decode(&result); err != nil {
			return nil, hcommon.I18nRichError(err, i18n.MsgMcpProbeJSONParseFailed)
		}
		return result, nil

	case http.StatusAccepted:
		// 202：响应将通过 SSE 流推送，从 channel 等待
		return waitSSEResponse(ctx, sseCh)

	default:
		return nil, hcommon.I18nError(i18n.MsgMcpProbeHTTPError, resp.StatusCode)
	}
}

// waitSSEResponse 从 SSE channel 中等待一个 JSON-RPC 响应
func waitSSEResponse(ctx context.Context, sseCh <-chan sseEvent) (map[string]interface{}, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case ev, ok := <-sseCh:
			if !ok {
				return nil, hcommon.I18nError(i18n.MsgMcpProbeSSEClosed)
			}
			if ev.err != nil {
				return nil, ev.err
			}
			// 解析 event\x00data 格式
			parts := strings.SplitN(ev.data, "\x00", 2)
			if len(parts) < 2 {
				continue
			}
			data := parts[1]

			// 尝试解析为 JSON-RPC 响应
			var result map[string]interface{}
			if err := json.Unmarshal([]byte(data), &result); err != nil {
				continue // 跳过非 JSON 数据
			}
			// 必须包含 jsonrpc 字段才是有效响应
			if _, ok := result["jsonrpc"]; ok {
				return result, nil
			}
		}
	}
}

// postNoWait 发送通知型 POST（不等待响应）
func (p *McpProber) postNoWait(ctx context.Context, client *http.Client, endpoint string, headers map[string]string, body interface{}) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		slog.WarnContext(ctx, "[McpProbe] postNoWait json.Marshal 失败", "error", err, "endpoint", endpoint)
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		slog.WarnContext(ctx, "[McpProbe] postNoWait 创建请求失败", "error", err, "endpoint", endpoint)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		slog.DebugContext(ctx, "[McpProbe] postNoWait 请求失败", "error", err, "endpoint", endpoint)
		return
	}
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
}

// resolveEndpointURL 将 SSE data 解析为完整的 endpoint URL
func resolveEndpointURL(data, baseURL string) (string, error) {
	// 绝对 URL
	if strings.HasPrefix(data, "http://") || strings.HasPrefix(data, "https://") {
		return data, nil
	}

	// 相对路径（以 / 开头）或相对引用（如 messages?session=xxx）
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	ref, err := url.Parse(data)
	if err != nil {
		return "", err
	}

	// 使用标准 URL 解析：base.ResolveReference 处理所有相对路径情况
	resolved := parsed.ResolveReference(ref)
	return resolved.String(), nil
}

// doJSONRPC 发送 JSON-RPC 请求并返回响应。
// 支持两种响应格式：
//   - Content-Type: application/json → 直接 JSON 解码
//   - Content-Type: text/event-stream → 从 SSE 事件 data 行中提取 JSON
func (p *McpProber) doJSONRPC(ctx context.Context, endpoint string, headers map[string]string, body interface{}) (map[string]interface{}, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, hcommon.I18nError(i18n.MsgMcpProbeHTTPError, resp.StatusCode)
	}

	// 限制响应读取大小
	limited := io.LimitReader(resp.Body, 512*1024)

	// 根据 Content-Type 选择解析方式
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/event-stream") {
		return parseSSEResponse(ctx, limited)
	}

	// 默认按 JSON 解码
	var result map[string]interface{}
	if err := json.NewDecoder(limited).Decode(&result); err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgMcpProbeJSONParseFailed)
	}

	return result, nil
}

// parseSSEResponse 从 SSE 流中提取第一个包含 JSON-RPC 响应的 data 行
func parseSSEResponse(ctx context.Context, body io.Reader) (map[string]interface{}, error) {
	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" {
			continue
		}

		var result map[string]interface{}
		if err := json.Unmarshal([]byte(data), &result); err != nil {
			continue // 跳过非 JSON 的 data 行
		}
		return result, nil
	}

	if err := scanner.Err(); err != nil {
		return nil, hcommon.I18nRichError(err, i18n.MsgMcpProbeSSEResponseReadFailed)
	}
	return nil, hcommon.I18nError(i18n.MsgMcpProbeSSENoData)
}

// parseToolsList 从 tools/list 响应中提取工具名称列表
func parseToolsList(resp map[string]interface{}) []string {
	var tools []string
	result, ok := resp["result"]
	if !ok {
		return tools
	}
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return tools
	}
	toolsList, ok := resultMap["tools"]
	if !ok {
		return tools
	}
	toolsArr, ok := toolsList.([]interface{})
	if !ok {
		return tools
	}
	for _, t := range toolsArr {
		if tm, ok := t.(map[string]interface{}); ok {
			if name, ok := tm["name"].(string); ok && name != "" {
				tools = append(tools, name)
			}
		}
	}
	return tools
}
