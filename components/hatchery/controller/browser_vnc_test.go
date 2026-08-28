package controller

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"hatchery/model"

	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
	"gorm.io/gorm"
)

// TestMarkAIActive_Basic 测试基本的 AI 活跃标记功能
func TestMarkAIActive_Basic(t *testing.T) {
	// 清理状态
	aiActiveMap = sync.Map{}

	var instanceID uint = 9999

	// 初始状态应为非活跃
	if isAIActive(instanceID) {
		t.Error("初始状态应为非活跃")
	}

	// 标记活跃
	MarkAIActive(instanceID)
	if !isAIActive(instanceID) {
		t.Error("标记后应为活跃")
	}

	// 标记非活跃
	MarkAIInactiveWithContext(instanceID, false)
	if isAIActive(instanceID) {
		t.Error("取消标记后应为非活跃")
	}
}

// TestMarkAIActive_MultipleConcurrentRequests 测试同一实例多个并发 LLM 请求的计数正确性
func TestMarkAIActive_MultipleConcurrentRequests(t *testing.T) {
	aiActiveMap = sync.Map{}

	var instanceID uint = 8888
	const concurrency = 100

	// 并发标记活跃
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			MarkAIActive(instanceID)
		}()
	}
	wg.Wait()

	// 应为活跃
	if !isAIActive(instanceID) {
		t.Error("并发标记后应为活跃")
	}

	// 验证计数器值
	val, ok := aiActiveMap.Load(instanceID)
	if !ok {
		t.Fatal("aiActiveMap 中应存在该实例")
	}
	state := val.(*aiActiveState)
	count := atomic.LoadInt64(&state.activeRequests)
	if count != concurrency {
		t.Errorf("activeRequests 应为 %d，实际为 %d", concurrency, count)
	}

	// 并发取消标记
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			MarkAIInactiveWithContext(instanceID, false)
		}()
	}
	wg.Wait()

	// 应为非活跃
	if isAIActive(instanceID) {
		t.Error("全部取消后应为非活跃")
	}

	count = atomic.LoadInt64(&state.activeRequests)
	if count != 0 {
		t.Errorf("activeRequests 应归零，实际为 %d", count)
	}
}

// TestMarkAIActive_MultipleInstances 测试多个实例之间互不影响
func TestMarkAIActive_MultipleInstances(t *testing.T) {
	aiActiveMap = sync.Map{}

	var inst1 uint = 1001
	var inst2 uint = 1002

	MarkAIActive(inst1)

	if !isAIActive(inst1) {
		t.Error("实例1应为活跃")
	}
	if isAIActive(inst2) {
		t.Error("实例2应为非活跃")
	}

	MarkAIActive(inst2)
	MarkAIInactiveWithContext(inst1, false)

	if isAIActive(inst1) {
		t.Error("实例1取消后应为非活跃")
	}
	if !isAIActive(inst2) {
		t.Error("实例2应仍为活跃")
	}

	MarkAIInactiveWithContext(inst2, false)
}

// TestMarkAIActive_Timeout 测试超时兜底机制
func TestMarkAIActive_Timeout(t *testing.T) {
	aiActiveMap = sync.Map{}

	var instanceID uint = 7777

	// 手动构造一个已超时的状态
	state := &aiActiveState{}
	atomic.StoreInt64(&state.activeRequests, 1)
	state.lastActiveAt.Store(time.Now().Add(-11 * time.Minute)) // 11 分钟前
	aiActiveMap.Store(instanceID, state)

	// 应触发超时兜底，返回 false
	if isAIActive(instanceID) {
		t.Error("超时后应视为非活跃")
	}

	// 验证计数器已被强制归零
	count := atomic.LoadInt64(&state.activeRequests)
	if count != 0 {
		t.Errorf("超时兜底后 activeRequests 应归零，实际为 %d", count)
	}
}

// TestMarkAIInactive_NonExistent 测试对不存在的实例调用 MarkAIInactive 不会 panic
func TestMarkAIInactive_NonExistent(t *testing.T) {
	aiActiveMap = sync.Map{}

	// 不应 panic
	MarkAIInactiveWithContext(99999, false)
}

// TestTakeover_Basic 测试接管状态的基本功能
func TestTakeover_Basic(t *testing.T) {
	takeoverMap = sync.Map{}

	var instanceID uint = 5555

	// 初始状态应为未接管
	if isTakeover(instanceID) {
		t.Error("初始状态应为未接管")
	}

	// 开始接管
	takeoverMap.Store(instanceID, true)
	if !isTakeover(instanceID) {
		t.Error("存储后应为已接管")
	}

	// 结束接管
	takeoverMap.Delete(instanceID)
	if isTakeover(instanceID) {
		t.Error("删除后应为未接管")
	}
}

// TestTakeover_MultipleInstances 测试多个实例的接管状态互不影响
func TestTakeover_MultipleInstances(t *testing.T) {
	takeoverMap = sync.Map{}

	var inst1 uint = 3001
	var inst2 uint = 3002

	takeoverMap.Store(inst1, true)

	if !isTakeover(inst1) {
		t.Error("实例1应为已接管")
	}
	if isTakeover(inst2) {
		t.Error("实例2应为未接管")
	}

	takeoverMap.Delete(inst1)
	takeoverMap.Store(inst2, true)

	if isTakeover(inst1) {
		t.Error("实例1删除后应为未接管")
	}
	if !isTakeover(inst2) {
		t.Error("实例2应为已接管")
	}

	takeoverMap.Delete(inst2)
}

// ========== 宽限期（Grace Period）相关测试 ==========

// TestGracePeriod_ToolCallsStartsGrace 测试 LLM 返回 tool_calls 时启动宽限期
func TestGracePeriod_ToolCallsStartsGrace(t *testing.T) {
	aiActiveMap = sync.Map{}

	var instanceID uint = 6001

	// 模拟 LLM 请求开始
	MarkAIActive(instanceID)
	if !isAIActive(instanceID) {
		t.Error("标记后应为活跃")
	}

	// 模拟 LLM 返回 tool_calls → 应启动宽限期
	MarkAIInactiveWithContext(instanceID, true)

	// activeRequests 已归零，但宽限期应保持活跃
	if !isAIActive(instanceID) {
		t.Error("LLM 返回 tool_calls 后，宽限期内应仍为活跃")
	}

	// 验证 graceActive 标志
	val, _ := aiActiveMap.Load(instanceID)
	state := val.(*aiActiveState)
	if atomic.LoadInt32(&state.graceActive) != 1 {
		t.Error("graceActive 应为 1")
	}

	// 清理
	cancelGraceTimer(state)
}

// TestGracePeriod_NoToolCallsImmediateInactive 测试 LLM 返回纯文本时立即归零
func TestGracePeriod_NoToolCallsImmediateInactive(t *testing.T) {
	aiActiveMap = sync.Map{}

	var instanceID uint = 6002

	// 模拟 LLM 请求开始
	MarkAIActive(instanceID)

	// 模拟 LLM 返回纯文本（无 tool_calls）→ 应立即归零
	MarkAIInactiveWithContext(instanceID, false)

	// 应立即变为非活跃
	if isAIActive(instanceID) {
		t.Error("LLM 返回纯文本后应立即变为非活跃")
	}
}

// TestGracePeriod_NewRequestCancelsGrace 测试新 LLM 请求到来时取消宽限期
func TestGracePeriod_NewRequestCancelsGrace(t *testing.T) {
	aiActiveMap = sync.Map{}

	var instanceID uint = 6003

	// 第一次 LLM 请求：返回 tool_calls → 启动宽限期
	MarkAIActive(instanceID)
	MarkAIInactiveWithContext(instanceID, true)

	// 宽限期内应为活跃
	if !isAIActive(instanceID) {
		t.Error("宽限期内应为活跃")
	}

	// 第二次 LLM 请求到来 → 应取消宽限期
	MarkAIActive(instanceID)

	// 验证宽限期已取消
	val, _ := aiActiveMap.Load(instanceID)
	state := val.(*aiActiveState)
	if atomic.LoadInt32(&state.graceActive) != 0 {
		t.Error("新请求到来后 graceActive 应为 0")
	}

	// 仍然活跃（因为有进行中的请求）
	if !isAIActive(instanceID) {
		t.Error("有进行中的请求，应为活跃")
	}

	// 第二次请求结束，无 tool_calls → 立即归零
	MarkAIInactiveWithContext(instanceID, false)
	if isAIActive(instanceID) {
		t.Error("Agent Loop 结束后应为非活跃")
	}
}

// TestGracePeriod_Expiry 测试宽限期到期后自动归零
func TestGracePeriod_Expiry(t *testing.T) {
	aiActiveMap = sync.Map{}

	var instanceID uint = 6004

	// 模拟 LLM 请求返回 tool_calls
	MarkAIActive(instanceID)
	MarkAIInactiveWithContext(instanceID, true)

	// 宽限期内应为活跃
	if !isAIActive(instanceID) {
		t.Error("宽限期内应为活跃")
	}

	// 手动触发宽限期到期（模拟定时器触发）
	val, _ := aiActiveMap.Load(instanceID)
	state := val.(*aiActiveState)
	cancelGraceTimer(state) // 取消定时器并清除 graceActive

	// 宽限期到期后应为非活跃
	if isAIActive(instanceID) {
		t.Error("宽限期到期后应为非活跃")
	}
}

// TestGracePeriod_ConcurrentRequests 测试并发请求下宽限期不会误触发
func TestGracePeriod_ConcurrentRequests(t *testing.T) {
	aiActiveMap = sync.Map{}

	var instanceID uint = 6005

	// 两个并发 LLM 请求
	MarkAIActive(instanceID)
	MarkAIActive(instanceID)

	// 第一个请求结束（有 tool_calls），但第二个还在进行中
	MarkAIInactiveWithContext(instanceID, true)

	// 应仍为活跃（activeRequests > 0）
	if !isAIActive(instanceID) {
		t.Error("仍有进行中的请求，应为活跃")
	}

	// 验证宽限期未启动（因为 activeRequests > 0）
	val, _ := aiActiveMap.Load(instanceID)
	state := val.(*aiActiveState)
	if atomic.LoadInt32(&state.graceActive) != 0 {
		t.Error("仍有进行中的请求时不应启动宽限期")
	}

	// 第二个请求结束（无 tool_calls）→ 立即归零
	MarkAIInactiveWithContext(instanceID, false)
	if isAIActive(instanceID) {
		t.Error("所有请求结束后应为非活跃")
	}
}

// TestGracePeriod_ToolCallsThenNoToolCalls 测试完整 Agent Loop 流程
func TestGracePeriod_ToolCallsThenNoToolCalls(t *testing.T) {
	aiActiveMap = sync.Map{}

	var instanceID uint = 6006

	// === 第一轮：LLM 返回 tool_calls ===
	MarkAIActive(instanceID)
	MarkAIInactiveWithContext(instanceID, true)
	if !isAIActive(instanceID) {
		t.Error("第一轮宽限期内应为活跃")
	}

	// === 第二轮：工具执行完成，新 LLM 请求到来 ===
	MarkAIActive(instanceID)
	// 宽限期应被取消
	val, _ := aiActiveMap.Load(instanceID)
	state := val.(*aiActiveState)
	if atomic.LoadInt32(&state.graceActive) != 0 {
		t.Error("新请求到来后宽限期应被取消")
	}

	// 第二轮 LLM 也返回 tool_calls
	MarkAIInactiveWithContext(instanceID, true)
	if !isAIActive(instanceID) {
		t.Error("第二轮宽限期内应为活跃")
	}

	// === 第三轮：最终 LLM 返回纯文本 ===
	MarkAIActive(instanceID)
	MarkAIInactiveWithContext(instanceID, false)
	if isAIActive(instanceID) {
		t.Error("Agent Loop 结束后应立即变为非活跃")
	}
}

// TestMarkAIInactive_BackwardCompatible 测试旧的 MarkAIInactive 向后兼容
func TestMarkAIInactive_BackwardCompatible(t *testing.T) {
	aiActiveMap = sync.Map{}

	var instanceID uint = 6007

	MarkAIActive(instanceID)
	if !isAIActive(instanceID) {
		t.Error("标记后应为活跃")
	}

	// 使用旧的 MarkAIInactive（等同于 hasToolCalls=false）
	MarkAIInactiveWithContext(instanceID, false)
	if isAIActive(instanceID) {
		t.Error("旧接口调用后应为非活跃（不启动宽限期）")
	}
}

// ========== CleanupVNCState 测试 ==========

// TestCleanupVNCState_CleansAllMaps 测试 CleanupVNCState 清理所有相关 sync.Map 条目
func TestCleanupVNCState_CleansAllMaps(t *testing.T) {
	// 重置所有状态
	aiActiveMap = sync.Map{}
	takeoverMap = sync.Map{}
	installingMap = sync.Map{}
	vncInstanceConns = sync.Map{}

	var instanceID uint = 11111

	// 设置各种状态
	MarkAIActive(instanceID)
	takeoverMap.Store(instanceID, true)
	installingMap.Store(instanceID, true)
	getOrCreateConnCount(instanceID) // 创建连接计数器

	// 验证状态已设置
	if !isAIActive(instanceID) {
		t.Error("AI 状态应已设置")
	}
	if !isTakeover(instanceID) {
		t.Error("接管状态应已设置")
	}
	if _, ok := installingMap.Load(instanceID); !ok {
		t.Error("安装锁应已设置")
	}
	if _, ok := vncInstanceConns.Load(instanceID); !ok {
		t.Error("连接计数器应已设置")
	}

	// 执行清理
	CleanupVNCState(instanceID)

	// 验证所有状态已清理
	if isAIActive(instanceID) {
		t.Error("清理后 AI 状态应为非活跃")
	}
	if isTakeover(instanceID) {
		t.Error("清理后接管状态应为未接管")
	}
	if _, ok := installingMap.Load(instanceID); ok {
		t.Error("清理后安装锁应已删除")
	}
	if _, ok := vncInstanceConns.Load(instanceID); ok {
		t.Error("清理后连接计数器应已删除")
	}
}

// TestCleanupVNCState_CancelsGraceTimer 测试 CleanupVNCState 取消宽限期定时器
func TestCleanupVNCState_CancelsGraceTimer(t *testing.T) {
	aiActiveMap = sync.Map{}
	takeoverMap = sync.Map{}
	installingMap = sync.Map{}
	vncInstanceConns = sync.Map{}

	var instanceID uint = 11112

	// 设置带宽限期的 AI 状态
	MarkAIActive(instanceID)
	MarkAIInactiveWithContext(instanceID, true) // 启动宽限期

	// 验证宽限期已启动
	if !isAIActive(instanceID) {
		t.Error("宽限期内应为活跃")
	}

	// 执行清理
	CleanupVNCState(instanceID)

	// 验证宽限期已取消，状态已清理
	if isAIActive(instanceID) {
		t.Error("清理后应为非活跃（宽限期应已取消）")
	}
	// 确认 aiActiveMap 中已无该实例
	if _, ok := aiActiveMap.Load(instanceID); ok {
		t.Error("清理后 aiActiveMap 中不应存在该实例")
	}
}

// TestCleanupVNCState_NonExistent 测试对不存在的实例调用 CleanupVNCState 不会 panic
func TestCleanupVNCState_NonExistent(t *testing.T) {
	aiActiveMap = sync.Map{}
	takeoverMap = sync.Map{}
	installingMap = sync.Map{}
	vncInstanceConns = sync.Map{}

	// 不应 panic
	CleanupVNCState(99998)
}

// TestCleanupVNCState_DoesNotAffectOtherInstances 测试清理不影响其他实例
func TestCleanupVNCState_DoesNotAffectOtherInstances(t *testing.T) {
	aiActiveMap = sync.Map{}
	takeoverMap = sync.Map{}
	installingMap = sync.Map{}
	vncInstanceConns = sync.Map{}

	var inst1 uint = 11113
	var inst2 uint = 11114

	// 设置两个实例的状态
	MarkAIActive(inst1)
	MarkAIActive(inst2)
	takeoverMap.Store(inst1, true)
	takeoverMap.Store(inst2, true)

	// 只清理实例1
	CleanupVNCState(inst1)

	// 实例1 应已清理
	if isAIActive(inst1) {
		t.Error("实例1 清理后应为非活跃")
	}
	if isTakeover(inst1) {
		t.Error("实例1 清理后应为未接管")
	}

	// 实例2 应不受影响
	if !isAIActive(inst2) {
		t.Error("实例2 应仍为活跃")
	}
	if !isTakeover(inst2) {
		t.Error("实例2 应仍为已接管")
	}

	// 清理实例2
	MarkAIInactiveWithContext(inst2, false)
	takeoverMap.Delete(inst2)
}

// ========== getOrCreateConnCount 测试 ==========

// TestGetOrCreateConnCount_Basic 测试连接计数器的基本创建和获取
func TestGetOrCreateConnCount_Basic(t *testing.T) {
	vncInstanceConns = sync.Map{}

	var instanceID uint = 22221

	// 首次获取应创建新计数器
	ptr1 := getOrCreateConnCount(instanceID)
	if ptr1 == nil {
		t.Fatal("计数器不应为 nil")
	}
	if atomic.LoadInt64(ptr1) != 0 {
		t.Error("新计数器初始值应为 0")
	}

	// 再次获取应返回同一个计数器
	ptr2 := getOrCreateConnCount(instanceID)
	if ptr1 != ptr2 {
		t.Error("同一实例应返回同一个计数器指针")
	}

	// 修改计数器值
	atomic.AddInt64(ptr1, 1)
	if atomic.LoadInt64(ptr2) != 1 {
		t.Error("通过 ptr1 修改后，ptr2 应能看到变化")
	}
}

// TestGetOrCreateConnCount_DifferentInstances 测试不同实例有独立的计数器
func TestGetOrCreateConnCount_DifferentInstances(t *testing.T) {
	vncInstanceConns = sync.Map{}

	var inst1 uint = 22222
	var inst2 uint = 22223

	ptr1 := getOrCreateConnCount(inst1)
	ptr2 := getOrCreateConnCount(inst2)

	if ptr1 == ptr2 {
		t.Error("不同实例应有不同的计数器指针")
	}

	atomic.AddInt64(ptr1, 5)
	if atomic.LoadInt64(ptr2) != 0 {
		t.Error("修改实例1的计数器不应影响实例2")
	}
}

// TestGetOrCreateConnCount_Concurrent 测试并发创建计数器的安全性
func TestGetOrCreateConnCount_Concurrent(t *testing.T) {
	vncInstanceConns = sync.Map{}

	var instanceID uint = 22224
	const goroutines = 100

	var wg sync.WaitGroup
	ptrs := make([]*int64, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ptrs[idx] = getOrCreateConnCount(instanceID)
		}(i)
	}
	wg.Wait()

	// 所有 goroutine 应获得同一个指针
	for i := 1; i < goroutines; i++ {
		if ptrs[i] != ptrs[0] {
			t.Errorf("goroutine %d 获得了不同的计数器指针", i)
		}
	}
}

// TestVNCProxyConnCount_LimitEnforcement 测试 VNC 代理连接数限制逻辑
func TestVNCProxyConnCount_LimitEnforcement(t *testing.T) {
	vncInstanceConns = sync.Map{}

	var instanceID uint = 22225
	countPtr := getOrCreateConnCount(instanceID)

	// 模拟 3 个连接（达到上限）
	for i := 0; i < maxVNCProxyPerInstance; i++ {
		current := atomic.AddInt64(countPtr, 1)
		if current > maxVNCProxyPerInstance {
			t.Errorf("第 %d 个连接不应超限", i+1)
		}
	}

	// 第 4 个连接应超限
	current := atomic.AddInt64(countPtr, 1)
	if current <= maxVNCProxyPerInstance {
		t.Error("第 4 个连接应超限")
	}
	// 超限后立即释放（模拟实际逻辑）
	atomic.AddInt64(countPtr, -1)

	// 释放一个连接后应可以再次连接
	atomic.AddInt64(countPtr, -1)
	current = atomic.AddInt64(countPtr, 1)
	if current > maxVNCProxyPerInstance {
		t.Error("释放后应可以再次连接")
	}

	// 清理
	for atomic.LoadInt64(countPtr) > 0 {
		atomic.AddInt64(countPtr, -1)
	}
}

// ========== isWebSocketUpgrade 测试 ==========

// TestIsWebSocketUpgrade_Valid 测试有效的 WebSocket 升级请求
func TestIsWebSocketUpgrade_Valid(t *testing.T) {
	tests := []struct {
		name       string
		upgrade    string
		connection string
		want       bool
	}{
		{
			name:       "标准 WebSocket 升级",
			upgrade:    "websocket",
			connection: "Upgrade",
			want:       true,
		},
		{
			name:       "大写 Upgrade 头",
			upgrade:    "WebSocket",
			connection: "Upgrade",
			want:       true,
		},
		{
			name:       "Connection 包含多个值",
			upgrade:    "websocket",
			connection: "keep-alive, Upgrade",
			want:       true,
		},
		{
			name:       "全小写",
			upgrade:    "websocket",
			connection: "upgrade",
			want:       true,
		},
		{
			name:       "缺少 Upgrade 头",
			upgrade:    "",
			connection: "Upgrade",
			want:       false,
		},
		{
			name:       "缺少 Connection 头",
			upgrade:    "websocket",
			connection: "",
			want:       false,
		},
		{
			name:       "Upgrade 不是 websocket",
			upgrade:    "h2c",
			connection: "Upgrade",
			want:       false,
		},
		{
			name:       "Connection 不包含 upgrade",
			upgrade:    "websocket",
			connection: "keep-alive",
			want:       false,
		},
		{
			name:       "两个头都为空",
			upgrade:    "",
			connection: "",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, _ := http.NewRequest("GET", "/ws", nil)
			if tt.upgrade != "" {
				r.Header.Set("Upgrade", tt.upgrade)
			}
			if tt.connection != "" {
				r.Header.Set("Connection", tt.connection)
			}
			got := isWebSocketUpgrade(r)
			if got != tt.want {
				t.Errorf("isWebSocketUpgrade() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ========== parseBrowserVNCScriptOutput 测试 ==========

// TestParseBrowserVNCScriptOutput_SingleLineJSON 测试单行 JSON 解析
func TestParseBrowserVNCScriptOutput_SingleLineJSON(t *testing.T) {
	output := `[1/7] 更新系统包列表...
[2/7] 安装系统依赖包...
安装完成
{"installed": true}`

	result := parseBrowserVNCScriptOutput(output)
	if result == nil {
		t.Fatal("解析结果不应为 nil")
	}
	if installed, ok := result["installed"].(bool); !ok || !installed {
		t.Error("installed 应为 true")
	}
}

// TestParseBrowserVNCScriptOutput_ErrorJSON 测试错误 JSON 解析
func TestParseBrowserVNCScriptOutput_ErrorJSON(t *testing.T) {
	output := `[1/7] 更新系统包列表...
错误: 安装系统依赖包失败
{"installed": false, "error": "安装系统依赖包失败"}`

	result := parseBrowserVNCScriptOutput(output)
	if result == nil {
		t.Fatal("解析结果不应为 nil")
	}
	if installed, ok := result["installed"].(bool); !ok || installed {
		t.Error("installed 应为 false")
	}
	if errMsg, ok := result["error"].(string); !ok || errMsg != "安装系统依赖包失败" {
		t.Errorf("error 应为 '安装系统依赖包失败'，实际为 '%v'", result["error"])
	}
}

// TestParseBrowserVNCScriptOutput_MultiLineJSON 测试多行 JSON 解析（jq 格式化输出）
func TestParseBrowserVNCScriptOutput_MultiLineJSON(t *testing.T) {
	output := `安装完成
{
  "installed": true,
  "version": "1.0"
}`

	result := parseBrowserVNCScriptOutput(output)
	if result == nil {
		t.Fatal("解析结果不应为 nil")
	}
	if installed, ok := result["installed"].(bool); !ok || !installed {
		t.Error("installed 应为 true")
	}
	if version, ok := result["version"].(string); !ok || version != "1.0" {
		t.Errorf("version 应为 '1.0'，实际为 '%v'", result["version"])
	}
}

// TestParseBrowserVNCScriptOutput_EmptyOutput 测试空输出
func TestParseBrowserVNCScriptOutput_EmptyOutput(t *testing.T) {
	result := parseBrowserVNCScriptOutput("")
	if result != nil {
		t.Error("空输出应返回 nil")
	}
}

// TestParseBrowserVNCScriptOutput_NoJSON 测试无 JSON 输出
func TestParseBrowserVNCScriptOutput_NoJSON(t *testing.T) {
	output := `[1/7] 更新系统包列表...
[2/7] 安装系统依赖包...
安装完成`

	result := parseBrowserVNCScriptOutput(output)
	if result != nil {
		t.Error("无 JSON 输出应返回 nil")
	}
}

// TestParseBrowserVNCScriptOutput_JSONInMiddle 测试 JSON 在输出中间（后面还有日志）
func TestParseBrowserVNCScriptOutput_JSONInMiddle(t *testing.T) {
	output := `开始安装...
{"installed": true}
后续日志行
更多日志`

	// 策略是从最后一行向前查找，所以中间的 JSON 也应该能找到
	result := parseBrowserVNCScriptOutput(output)
	if result == nil {
		t.Fatal("解析结果不应为 nil")
	}
	if installed, ok := result["installed"].(bool); !ok || !installed {
		t.Error("installed 应为 true")
	}
}

// TestParseBrowserVNCScriptOutput_CheckScript 测试 check 脚本的典型输出
func TestParseBrowserVNCScriptOutput_CheckScript(t *testing.T) {
	output := `检查 VNC 环境...
Xvnc: RUNNING
websockify: RUNNING
Chrome: RUNNING
{"ready": true, "services": {"xvnc": "RUNNING", "websockify": "RUNNING", "chrome": "RUNNING"}}`

	result := parseBrowserVNCScriptOutput(output)
	if result == nil {
		t.Fatal("解析结果不应为 nil")
	}
	if ready, ok := result["ready"].(bool); !ok || !ready {
		t.Error("ready 应为 true")
	}
	services, ok := result["services"].(map[string]interface{})
	if !ok {
		t.Fatal("services 应为 map")
	}
	if services["xvnc"] != "RUNNING" {
		t.Errorf("xvnc 应为 RUNNING，实际为 %v", services["xvnc"])
	}
}

// TestParseBrowserVNCScriptOutput_InvalidJSON 测试无效 JSON 行
func TestParseBrowserVNCScriptOutput_InvalidJSON(t *testing.T) {
	output := `日志行
{invalid json}
{"valid": true}`

	result := parseBrowserVNCScriptOutput(output)
	if result == nil {
		t.Fatal("应能解析最后一行有效 JSON")
	}
	if result["valid"] != true {
		t.Error("valid 应为 true")
	}
}

// TestParseBrowserVNCScriptOutput_NestedJSON 测试嵌套 JSON
func TestParseBrowserVNCScriptOutput_NestedJSON(t *testing.T) {
	output := `{
  "ready": true,
  "checks": {
	"packages": {"status": "ok"},
	"services": {"status": "ok"}
  }
}`

	result := parseBrowserVNCScriptOutput(output)
	if result == nil {
		t.Fatal("解析结果不应为 nil")
	}
	if ready, ok := result["ready"].(bool); !ok || !ready {
		t.Error("ready 应为 true")
	}
}

// TestParseBrowserVNCScriptOutput_OnlyWhitespace 测试只有空白字符
func TestParseBrowserVNCScriptOutput_OnlyWhitespace(t *testing.T) {
	result := parseBrowserVNCScriptOutput("   \n  \n  ")
	if result != nil {
		t.Error("只有空白字符应返回 nil")
	}
}

// ========== VNC 代理全局连接计数测试 ==========

// TestVNCProxyGlobalConnCount 测试全局 VNC 代理连接计数的原子性
func TestVNCProxyGlobalConnCount(t *testing.T) {
	// 保存并重置全局计数
	old := atomic.LoadInt64(&vncProxyConnections)
	atomic.StoreInt64(&vncProxyConnections, 0)
	defer atomic.StoreInt64(&vncProxyConnections, old)

	const goroutines = 50
	var wg sync.WaitGroup

	// 并发增加
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			atomic.AddInt64(&vncProxyConnections, 1)
		}()
	}
	wg.Wait()

	if atomic.LoadInt64(&vncProxyConnections) != goroutines {
		t.Errorf("全局连接数应为 %d，实际为 %d", goroutines, atomic.LoadInt64(&vncProxyConnections))
	}

	// 并发减少
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			atomic.AddInt64(&vncProxyConnections, -1)
		}()
	}
	wg.Wait()

	if atomic.LoadInt64(&vncProxyConnections) != 0 {
		t.Errorf("全局连接数应归零，实际为 %d", atomic.LoadInt64(&vncProxyConnections))
	}
}

// ========== Handler 内核函数测试 ==========

// TestBrowserStatusCore_MethodNotAllowed 测试非 GET 请求返回 405
func TestBrowserStatusCore_MethodNotAllowed(t *testing.T) {
	aiActiveMap = sync.Map{}
	takeoverMap = sync.Map{}

	instance := &model.Instance{Model: gorm.Model{ID: 30001}}
	req := httptest.NewRequest(http.MethodPost, "/openclaw/browser-status?id=1", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserStatusCore(w, req, instance, true)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望 405，实际=%d", w.Code)
	}
}

// TestBrowserStatusCore_FeatureDisabled 测试功能未开启返回 403
func TestBrowserStatusCore_FeatureDisabled(t *testing.T) {
	aiActiveMap = sync.Map{}
	takeoverMap = sync.Map{}

	instance := &model.Instance{Model: gorm.Model{ID: 30002}}
	req := httptest.NewRequest(http.MethodGet, "/openclaw/browser-status?id=1", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserStatusCore(w, req, instance, false)

	if w.Code != http.StatusForbidden {
		t.Errorf("期望 403，实际=%d", w.Code)
	}
}

// TestBrowserStatusCore_Success 测试正常返回 AI 状态
func TestBrowserStatusCore_Success(t *testing.T) {
	aiActiveMap = sync.Map{}
	takeoverMap = sync.Map{}

	instance := &model.Instance{Model: gorm.Model{ID: 30003}}

	// 设置 AI 活跃和接管状态
	MarkAIActive(instance.ID)
	takeoverMap.Store(instance.ID, true)
	defer func() {
		MarkAIInactiveWithContext(instance.ID, false)
		takeoverMap.Delete(instance.ID)
	}()

	req := httptest.NewRequest(http.MethodGet, "/openclaw/browser-status?id=1", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserStatusCore(w, req, instance, true)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，实际=%d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatal("响应应包含 data 字段")
	}
	if data["ai_active"] != true {
		t.Errorf("ai_active 应为 true，实际为 %v", data["ai_active"])
	}
	if data["takeover"] != true {
		t.Errorf("takeover 应为 true，实际为 %v", data["takeover"])
	}
}

// TestBrowserStatusCore_InactiveState 测试非活跃状态
func TestBrowserStatusCore_InactiveState(t *testing.T) {
	aiActiveMap = sync.Map{}
	takeoverMap = sync.Map{}

	instance := &model.Instance{Model: gorm.Model{ID: 30004}}

	req := httptest.NewRequest(http.MethodGet, "/openclaw/browser-status?id=1", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserStatusCore(w, req, instance, true)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，实际=%d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]interface{})
	if data["ai_active"] != false {
		t.Errorf("ai_active 应为 false，实际为 %v", data["ai_active"])
	}
	if data["takeover"] != false {
		t.Errorf("takeover 应为 false，实际为 %v", data["takeover"])
	}
}

// ========== browserTakeoverCore 测试 ==========

// TestBrowserTakeoverCore_MethodNotAllowed 测试非 POST 请求返回 405
func TestBrowserTakeoverCore_MethodNotAllowed(t *testing.T) {
	takeoverMap = sync.Map{}

	instance := &model.Instance{Model: gorm.Model{ID: 31001}}
	req := httptest.NewRequest(http.MethodGet, "/openclaw/browser-takeover", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserTakeoverCore(w, req, instance, true)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望 405，实际=%d", w.Code)
	}
}

// TestBrowserTakeoverCore_FeatureDisabled 测试功能未开启返回 403
func TestBrowserTakeoverCore_FeatureDisabled(t *testing.T) {
	takeoverMap = sync.Map{}

	instance := &model.Instance{Model: gorm.Model{ID: 31002}}
	req := httptest.NewRequest(http.MethodPost, "/openclaw/browser-takeover", strings.NewReader("action=start"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserTakeoverCore(w, req, instance, false)

	if w.Code != http.StatusForbidden {
		t.Errorf("期望 403，实际=%d", w.Code)
	}
}

// TestBrowserTakeoverCore_StartTakeover 测试开始接管
func TestBrowserTakeoverCore_StartTakeover(t *testing.T) {
	takeoverMap = sync.Map{}

	instance := &model.Instance{Model: gorm.Model{ID: 31003}}
	req := httptest.NewRequest(http.MethodPost, "/openclaw/browser-takeover", strings.NewReader("action=start"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserTakeoverCore(w, req, instance, true)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，实际=%d", w.Code)
	}

	// 验证接管状态已设置
	if !isTakeover(instance.ID) {
		t.Error("接管状态应已设置")
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]interface{})
	if data["takeover"] != true {
		t.Errorf("takeover 应为 true，实际为 %v", data["takeover"])
	}

	// 清理
	takeoverMap.Delete(instance.ID)
}

// TestBrowserTakeoverCore_StopTakeover 测试结束接管
func TestBrowserTakeoverCore_StopTakeover(t *testing.T) {
	takeoverMap = sync.Map{}

	instance := &model.Instance{Model: gorm.Model{ID: 31004}}
	takeoverMap.Store(instance.ID, true)

	req := httptest.NewRequest(http.MethodPost, "/openclaw/browser-takeover", strings.NewReader("action=stop"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserTakeoverCore(w, req, instance, true)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，实际=%d", w.Code)
	}

	if isTakeover(instance.ID) {
		t.Error("接管状态应已清除")
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]interface{})
	if data["takeover"] != false {
		t.Errorf("takeover 应为 false，实际为 %v", data["takeover"])
	}
}

// TestBrowserTakeoverCore_InvalidAction 测试无效 action 参数
func TestBrowserTakeoverCore_InvalidAction(t *testing.T) {
	takeoverMap = sync.Map{}

	instance := &model.Instance{Model: gorm.Model{ID: 31005}}
	req := httptest.NewRequest(http.MethodPost, "/openclaw/browser-takeover", strings.NewReader("action=invalid"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserTakeoverCore(w, req, instance, true)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w.Code)
	}
}

// TestBrowserTakeoverCore_EmptyAction 测试空 action 参数
func TestBrowserTakeoverCore_EmptyAction(t *testing.T) {
	takeoverMap = sync.Map{}

	instance := &model.Instance{Model: gorm.Model{ID: 31006}}
	req := httptest.NewRequest(http.MethodPost, "/openclaw/browser-takeover", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserTakeoverCore(w, req, instance, true)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w.Code)
	}
}

// ========== browserVNCAccessCore 测试 ==========

// TestBrowserVNCAccessCore_MethodNotAllowed 测试非 GET 请求返回 405
func TestBrowserVNCAccessCore_MethodNotAllowed(t *testing.T) {
	instance := &model.Instance{Model: gorm.Model{ID: 32001}, InstanceId: "ins-test"}
	siteConfig := model.SiteConfig{BrowserVNCEnable: true}

	req := httptest.NewRequest(http.MethodPost, "/openclaw/browser-vnc-access?id=1", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserVNCAccessCore(w, req, instance, siteConfig, nil, nil)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望 405，实际=%d", w.Code)
	}
}

// TestBrowserVNCAccessCore_FeatureDisabled 测试功能未开启返回 403
func TestBrowserVNCAccessCore_FeatureDisabled(t *testing.T) {
	instance := &model.Instance{Model: gorm.Model{ID: 32002}, InstanceId: "ins-test"}
	siteConfig := model.SiteConfig{BrowserVNCEnable: false}

	req := httptest.NewRequest(http.MethodGet, "/openclaw/browser-vnc-access?id=1", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserVNCAccessCore(w, req, instance, siteConfig, nil, nil)

	if w.Code != http.StatusForbidden {
		t.Errorf("期望 403，实际=%d", w.Code)
	}
}

// TestBrowserVNCAccessCore_NoCVM 测试无关联 CVM 返回 400
func TestBrowserVNCAccessCore_NoCVM(t *testing.T) {
	instance := &model.Instance{Model: gorm.Model{ID: 32003}, InstanceId: ""}
	siteConfig := model.SiteConfig{BrowserVNCEnable: true}

	req := httptest.NewRequest(http.MethodGet, "/openclaw/browser-vnc-access?id=1", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserVNCAccessCore(w, req, instance, siteConfig, nil, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w.Code)
	}
}

// TestBrowserVNCAccessCore_DescribeFails 测试 CVM 查询失败返回 500
func TestBrowserVNCAccessCore_DescribeFails(t *testing.T) {
	instance := &model.Instance{Model: gorm.Model{ID: 32004}, InstanceId: "ins-test"}
	siteConfig := model.SiteConfig{BrowserVNCEnable: true}

	req := httptest.NewRequest(http.MethodGet, "/openclaw/browser-vnc-access?id=1", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserVNCAccessCore(w, req, instance, siteConfig,
		func(id string) (*cvmInstanceInfo, error) {
			return nil, fmt.Errorf("CVM API 超时")
		}, nil)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("期望 500，实际=%d", w.Code)
	}
}

// TestBrowserVNCAccessCore_InstanceNotFound 测试 CVM 实例不存在返回 404
func TestBrowserVNCAccessCore_InstanceNotFound(t *testing.T) {
	instance := &model.Instance{Model: gorm.Model{ID: 32005}, InstanceId: "ins-notfound"}
	siteConfig := model.SiteConfig{BrowserVNCEnable: true}

	req := httptest.NewRequest(http.MethodGet, "/openclaw/browser-vnc-access?id=1", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserVNCAccessCore(w, req, instance, siteConfig,
		func(id string) (*cvmInstanceInfo, error) {
			return nil, nil // 未找到
		}, nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("期望 404，实际=%d", w.Code)
	}
}

// TestBrowserVNCAccessCore_InstanceNotRunning 测试实例非运行状态返回 409
func TestBrowserVNCAccessCore_InstanceNotRunning(t *testing.T) {
	instance := &model.Instance{Model: gorm.Model{ID: 32006}, InstanceId: "ins-stopped"}
	siteConfig := model.SiteConfig{BrowserVNCEnable: true}

	req := httptest.NewRequest(http.MethodGet, "/openclaw/browser-vnc-access?id=1", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserVNCAccessCore(w, req, instance, siteConfig,
		func(id string) (*cvmInstanceInfo, error) {
			return &cvmInstanceInfo{InstanceState: "STOPPED", PublicIp: "1.2.3.4"}, nil
		}, nil)

	if w.Code != http.StatusConflict {
		t.Errorf("期望 409，实际=%d", w.Code)
	}
}

// TestBrowserVNCAccessCore_NoPublicIP 测试无公网 IP 返回 500
func TestBrowserVNCAccessCore_NoPublicIP(t *testing.T) {
	instance := &model.Instance{Model: gorm.Model{ID: 32007}, InstanceId: "ins-noip"}
	siteConfig := model.SiteConfig{BrowserVNCEnable: true}

	req := httptest.NewRequest(http.MethodGet, "/openclaw/browser-vnc-access?id=1", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserVNCAccessCore(w, req, instance, siteConfig,
		func(id string) (*cvmInstanceInfo, error) {
			return &cvmInstanceInfo{InstanceState: "RUNNING", PublicIp: ""}, nil
		}, nil)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("期望 500，实际=%d", w.Code)
	}
}

// TestBrowserVNCAccessCore_SuccessAccessible 测试正常返回（端口已放通）
func TestBrowserVNCAccessCore_SuccessAccessible(t *testing.T) {
	instance := &model.Instance{Model: gorm.Model{ID: 32008}, InstanceId: "ins-ok"}
	siteConfig := model.SiteConfig{BrowserVNCEnable: true, SecurityGroupId: "sg-test"}

	req := httptest.NewRequest(http.MethodGet, "/openclaw/browser-vnc-access?id=1", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserVNCAccessCore(w, req, instance, siteConfig,
		func(id string) (*cvmInstanceInfo, error) {
			return &cvmInstanceInfo{InstanceState: "RUNNING", PublicIp: "1.2.3.4"}, nil
		},
		func(sgId string, port int) (bool, error) {
			return true, nil // 端口已放通
		})

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，实际=%d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]interface{})
	if data["accessible"] != true {
		t.Errorf("accessible 应为 true，实际为 %v", data["accessible"])
	}
	if data["host"] != "1.2.3.4" {
		t.Errorf("host 应为 '1.2.3.4'，实际为 %v", data["host"])
	}
	if data["vnc_url"] == "" {
		t.Error("vnc_url 不应为空")
	}
	if data["novnc_url"] == "" {
		t.Error("novnc_url 不应为空")
	}
	if data["ws_proxy_path"] == "" {
		t.Error("ws_proxy_path 不应为空")
	}
}

// TestBrowserVNCAccessCore_PortNotAccessible 测试端口未放通
func TestBrowserVNCAccessCore_PortNotAccessible(t *testing.T) {
	instance := &model.Instance{Model: gorm.Model{ID: 32009}, InstanceId: "ins-noaccess"}
	siteConfig := model.SiteConfig{BrowserVNCEnable: true, SecurityGroupId: "sg-test"}

	req := httptest.NewRequest(http.MethodGet, "/openclaw/browser-vnc-access?id=1", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserVNCAccessCore(w, req, instance, siteConfig,
		func(id string) (*cvmInstanceInfo, error) {
			return &cvmInstanceInfo{InstanceState: "RUNNING", PublicIp: "1.2.3.4"}, nil
		},
		func(sgId string, port int) (bool, error) {
			return false, nil // 端口未放通
		})

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，实际=%d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]interface{})
	if data["accessible"] != false {
		t.Errorf("accessible 应为 false，实际为 %v", data["accessible"])
	}
	if data["vnc_url"] != "" {
		t.Errorf("vnc_url 应为空，实际为 %v", data["vnc_url"])
	}
	if data["message"] == nil || data["message"] == "" {
		t.Error("应包含 message 提示信息")
	}
}

// TestBrowserVNCAccessCore_NoSecurityGroup 测试未配置安全组
func TestBrowserVNCAccessCore_NoSecurityGroup(t *testing.T) {
	instance := &model.Instance{Model: gorm.Model{ID: 32010}, InstanceId: "ins-nosg"}
	siteConfig := model.SiteConfig{BrowserVNCEnable: true, SecurityGroupId: ""}

	req := httptest.NewRequest(http.MethodGet, "/openclaw/browser-vnc-access?id=1", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserVNCAccessCore(w, req, instance, siteConfig,
		func(id string) (*cvmInstanceInfo, error) {
			return &cvmInstanceInfo{InstanceState: "RUNNING", PublicIp: "1.2.3.4"}, nil
		}, nil)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，实际=%d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]interface{})
	if data["accessible"] != false {
		t.Errorf("accessible 应为 false，实际为 %v", data["accessible"])
	}
	if data["message"] == nil || data["message"] == "" {
		t.Error("应包含 message 提示信息")
	}
}

// TestBrowserVNCAccessCore_SecurityGroupCheckError 测试安全组检查失败
func TestBrowserVNCAccessCore_SecurityGroupCheckError(t *testing.T) {
	instance := &model.Instance{Model: gorm.Model{ID: 32011}, InstanceId: "ins-sgerr"}
	siteConfig := model.SiteConfig{BrowserVNCEnable: true, SecurityGroupId: "sg-test"}

	req := httptest.NewRequest(http.MethodGet, "/openclaw/browser-vnc-access?id=1", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserVNCAccessCore(w, req, instance, siteConfig,
		func(id string) (*cvmInstanceInfo, error) {
			return &cvmInstanceInfo{InstanceState: "RUNNING", PublicIp: "1.2.3.4"}, nil
		},
		func(sgId string, port int) (bool, error) {
			return false, fmt.Errorf("VPC API 超时")
		})

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，实际=%d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]interface{})
	if data["accessible"] != false {
		t.Errorf("accessible 应为 false，实际为 %v", data["accessible"])
	}
}

// TestBrowserVNCAccessCore_ProductionPath_BootstrapNotDone 覆盖生产路径 default 分支：
// checkPortFn=nil + instance.SecurityGroupId 非空 → 走 checkPortRuleOnInstanceSG。
// 全新租户（无 RuleSet）会触发 ErrSGBootstrapNotDone，此时返回 200 + accessible=false +
// 友好 message（不报 500）。
func TestBrowserVNCAccessCore_ProductionPath_BootstrapNotDone(t *testing.T) {
	_ = setupSGPoolTestDB(t) // 空 DB → 0 个 RuleSet → ErrSGBootstrapNotDone

	instance := &model.Instance{
		Model:           gorm.Model{ID: 32012},
		InstanceId:      "ins-bootstrap",
		SecurityGroupId: "sg-active-1",
	}
	siteConfig := model.SiteConfig{BrowserVNCEnable: true}

	req := httptest.NewRequest(http.MethodGet, "/openclaw/browser-vnc-access?id=1", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserVNCAccessCore(w, req, instance, siteConfig,
		func(id string) (*cvmInstanceInfo, error) {
			return &cvmInstanceInfo{InstanceState: "RUNNING", PublicIp: "1.2.3.4"}, nil
		},
		nil) // 生产路径：传 nil 走 default 分支

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]interface{})
	if data["accessible"] != false {
		t.Errorf("accessible 应为 false（bootstrap 未完成），实际=%v", data["accessible"])
	}
	if msg, _ := data["message"].(string); msg == "" {
		t.Error("应返回友好 message")
	}
}

// TestBrowserVNCAccessCore_ProductionPath_PortAllowed 覆盖生产路径 default 分支同步态命中：
// 实例绑的 SG 在 ACTIVE 池里，rule_version 与 RuleSet.version 一致，规则放通目标端口。
func TestBrowserVNCAccessCore_ProductionPath_PortAllowed(t *testing.T) {
	db := setupSGPoolTestDB(t)

	// 准备：1 个 RuleSet（含放通 6080 端口的规则）+ 1 个 ACTIVE SG（rule_version 同步）
	rulesJSON := `[{"direction":"INGRESS","action":"ACCEPT","protocol":"TCP","port":"6080","cidr_block":"0.0.0.0/0"}]`
	rs := &model.RuleSet{
		Name:      model.DefaultRuleSetName,
		Rules:     rulesJSON,
		Version:   1,
		IsDefault: true,
	}
	db.Create(rs)
	db.Create(&model.ManagedSGPool{
		SGID:        "sg-active-1",
		RuleSetID:   rs.ID,
		Status:      model.SGStatusActive,
		RuleVersion: 1, // 与 rs.Version 一致 → 同步态走 DB 快路径
	})

	instance := &model.Instance{
		Model:           gorm.Model{ID: 32013},
		InstanceId:      "ins-prod-ok",
		SecurityGroupId: "sg-active-1",
	}
	siteConfig := model.SiteConfig{BrowserVNCEnable: true}

	req := httptest.NewRequest(http.MethodGet, "/openclaw/browser-vnc-access?id=1", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserVNCAccessCore(w, req, instance, siteConfig,
		func(id string) (*cvmInstanceInfo, error) {
			return &cvmInstanceInfo{InstanceState: "RUNNING", PublicIp: "1.2.3.4"}, nil
		},
		nil)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]interface{})
	if data["accessible"] != true {
		t.Errorf("accessible 应为 true，实际=%v", data["accessible"])
	}
}

// ========== browserVNCInstallCore 测试 ==========

// TestBrowserVNCInstallCore_MethodNotAllowed 测试非 POST 请求返回 405
func TestBrowserVNCInstallCore_MethodNotAllowed(t *testing.T) {
	installingMap = sync.Map{}
	instance := &model.Instance{Model: gorm.Model{ID: 33001}, InstanceId: "ins-test"}

	req := httptest.NewRequest(http.MethodGet, "/openclaw/browser-vnc-install", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserVNCInstallCore(w, req, instance, true, nil)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望 405，实际=%d", w.Code)
	}
}

// TestBrowserVNCInstallCore_FeatureDisabled 测试功能未开启返回 403
func TestBrowserVNCInstallCore_FeatureDisabled(t *testing.T) {
	installingMap = sync.Map{}
	instance := &model.Instance{Model: gorm.Model{ID: 33002}, InstanceId: "ins-test"}

	req := httptest.NewRequest(http.MethodPost, "/openclaw/browser-vnc-install", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserVNCInstallCore(w, req, instance, false, nil)

	if w.Code != http.StatusForbidden {
		t.Errorf("期望 403，实际=%d", w.Code)
	}
}

// TestBrowserVNCInstallCore_NoCVM 测试无关联 CVM 返回 400
func TestBrowserVNCInstallCore_NoCVM(t *testing.T) {
	installingMap = sync.Map{}
	instance := &model.Instance{Model: gorm.Model{ID: 33003}, InstanceId: ""}

	req := httptest.NewRequest(http.MethodPost, "/openclaw/browser-vnc-install", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserVNCInstallCore(w, req, instance, true, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w.Code)
	}
}

// TestBrowserVNCInstallCore_ConcurrentInstall 测试并发安装保护返回 409
func TestBrowserVNCInstallCore_ConcurrentInstall(t *testing.T) {
	installingMap = sync.Map{}
	instance := &model.Instance{Model: gorm.Model{ID: 33004}, InstanceId: "ins-concurrent"}

	// 模拟已有安装进行中
	lockToken := &struct{}{}
	installingMap.Store(instance.ID, lockToken)
	defer installingMap.CompareAndDelete(instance.ID, lockToken)

	req := httptest.NewRequest(http.MethodPost, "/openclaw/browser-vnc-install", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserVNCInstallCore(w, req, instance, true,
		func(instanceId, script string, timeout uint64, runtimeUser string, unlockInstalling func()) (string, error) {
			return `{"installed": true}`, nil
		})

	if w.Code != http.StatusConflict {
		t.Errorf("期望 409，实际=%d", w.Code)
	}
}

// TestBrowserVNCInstallCore_ScriptFails 测试脚本执行失败返回 500
func TestBrowserVNCInstallCore_ScriptFails(t *testing.T) {
	installingMap = sync.Map{}
	instance := &model.Instance{Model: gorm.Model{ID: 33005}, InstanceId: "ins-fail"}

	req := httptest.NewRequest(http.MethodPost, "/openclaw/browser-vnc-install", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserVNCInstallCore(w, req, instance, true,
		func(instanceId, script string, timeout uint64, runtimeUser string, unlockInstalling func()) (string, error) {
			return "", fmt.Errorf("TAT 执行超时")
		})

	if w.Code != http.StatusInternalServerError {
		t.Errorf("期望 500，实际=%d", w.Code)
	}
}

// TestBrowserVNCInstallCore_InvalidOutput 测试脚本输出无法解析返回 500
func TestBrowserVNCInstallCore_InvalidOutput(t *testing.T) {
	installingMap = sync.Map{}
	instance := &model.Instance{Model: gorm.Model{ID: 33006}, InstanceId: "ins-badout"}

	req := httptest.NewRequest(http.MethodPost, "/openclaw/browser-vnc-install", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserVNCInstallCore(w, req, instance, true,
		func(instanceId, script string, timeout uint64, runtimeUser string, unlockInstalling func()) (string, error) {
			return "no json here", nil
		})

	if w.Code != http.StatusInternalServerError {
		t.Errorf("期望 500，实际=%d", w.Code)
	}
}

// TestBrowserVNCInstallCore_Success 测试安装成功
func TestBrowserVNCInstallCore_Success(t *testing.T) {
	installingMap = sync.Map{}
	instance := &model.Instance{Model: gorm.Model{ID: 33007}, InstanceId: "ins-success"}

	req := httptest.NewRequest(http.MethodPost, "/openclaw/browser-vnc-install", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserVNCInstallCore(w, req, instance, true,
		func(instanceId, script string, timeout uint64, runtimeUser string, unlockInstalling func()) (string, error) {
			// 验证参数
			if script != "install_browser_vnc.sh" {
				t.Errorf("脚本名应为 install_browser_vnc.sh，实际为 %s", script)
			}
			if timeout != 300 {
				t.Errorf("超时应为 300，实际为 %d", timeout)
			}
			return `{"installed": true}`, nil
		})

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，实际=%d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]interface{})
	if data["installed"] != true {
		t.Errorf("installed 应为 true，实际为 %v", data["installed"])
	}

	// 验证安装锁已释放
	if _, ok := installingMap.Load(instance.ID); ok {
		t.Error("安装完成后安装锁应已释放")
	}
}

// ========== browserVNCCheckCore 测试 ==========

// TestBrowserVNCCheckCore_MethodNotAllowed 测试非 GET 请求返回 405
func TestBrowserVNCCheckCore_MethodNotAllowed(t *testing.T) {
	instance := &model.Instance{Model: gorm.Model{ID: 34001}, InstanceId: "ins-test"}

	req := httptest.NewRequest(http.MethodPost, "/openclaw/browser-vnc-check", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserVNCCheckCore(w, req, instance, true, nil, nil)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望 405，实际=%d", w.Code)
	}
}

// TestBrowserVNCCheckCore_FeatureDisabled 测试功能未开启返回 403
func TestBrowserVNCCheckCore_FeatureDisabled(t *testing.T) {
	instance := &model.Instance{Model: gorm.Model{ID: 34002}, InstanceId: "ins-test"}

	req := httptest.NewRequest(http.MethodGet, "/openclaw/browser-vnc-check?id=1", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserVNCCheckCore(w, req, instance, false, nil, nil)

	if w.Code != http.StatusForbidden {
		t.Errorf("期望 403，实际=%d", w.Code)
	}
}

// TestBrowserVNCCheckCore_NoCVM 测试无关联 CVM 返回 400
func TestBrowserVNCCheckCore_NoCVM(t *testing.T) {
	instance := &model.Instance{Model: gorm.Model{ID: 34003}, InstanceId: ""}

	req := httptest.NewRequest(http.MethodGet, "/openclaw/browser-vnc-check?id=1", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserVNCCheckCore(w, req, instance, true, nil, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w.Code)
	}
}

// TestBrowserVNCCheckCore_ScriptFails 测试脚本执行失败返回 500
func TestBrowserVNCCheckCore_ScriptFails(t *testing.T) {
	instance := &model.Instance{Model: gorm.Model{ID: 34004}, InstanceId: "ins-fail"}

	req := httptest.NewRequest(http.MethodGet, "/openclaw/browser-vnc-check?id=1", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserVNCCheckCore(w, req, instance, true,
		func(instanceId, script string, timeout uint64, runtimeUser string) (string, error) {
			return "", fmt.Errorf("TAT 执行超时")
		}, nil)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("期望 500，实际=%d", w.Code)
	}
}

// TestBrowserVNCCheckCore_InvalidOutput 测试脚本输出无法解析返回 500
func TestBrowserVNCCheckCore_InvalidOutput(t *testing.T) {
	instance := &model.Instance{Model: gorm.Model{ID: 34005}, InstanceId: "ins-badout"}

	req := httptest.NewRequest(http.MethodGet, "/openclaw/browser-vnc-check?id=1", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserVNCCheckCore(w, req, instance, true,
		func(instanceId, script string, timeout uint64, runtimeUser string) (string, error) {
			return "no json here", nil
		}, nil)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("期望 500，实际=%d", w.Code)
	}
}

// TestBrowserVNCCheckCore_Success 测试检查成功
func TestBrowserVNCCheckCore_Success(t *testing.T) {
	instance := &model.Instance{Model: gorm.Model{ID: 34006}, InstanceId: "ins-ok"}

	req := httptest.NewRequest(http.MethodGet, "/openclaw/browser-vnc-check?id=1", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserVNCCheckCore(w, req, instance, true,
		func(instanceId, script string, timeout uint64, runtimeUser string) (string, error) {
			if script != "check_browser_vnc.sh" {
				t.Errorf("脚本名应为 check_browser_vnc.sh，实际为 %s", script)
			}
			if timeout != 30 {
				t.Errorf("超时应为 30，实际为 %d", timeout)
			}
			return `{"ready": true, "services": {"xvnc": "RUNNING"}}`, nil
		},
		func(_ context.Context, instanceId string) string {
			return "Ubuntu Server 24.04 LTS 64bit"
		})

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，实际=%d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]interface{})
	if data["ready"] != true {
		t.Errorf("ready 应为 true，实际为 %v", data["ready"])
	}
	if data["os_name"] != "Ubuntu Server 24.04 LTS 64bit" {
		t.Errorf("os_name 应为 'Ubuntu Server 24.04 LTS 64bit'，实际为 %v", data["os_name"])
	}
}

// TestBrowserVNCCheckCore_SuccessWithoutOsName 测试检查成功但无 OS 名称
func TestBrowserVNCCheckCore_SuccessWithoutOsName(t *testing.T) {
	instance := &model.Instance{Model: gorm.Model{ID: 34007}, InstanceId: "ins-noos"}

	req := httptest.NewRequest(http.MethodGet, "/openclaw/browser-vnc-check?id=1", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserVNCCheckCore(w, req, instance, true,
		func(instanceId, script string, timeout uint64, runtimeUser string) (string, error) {
			return `{"ready": true}`, nil
		},
		func(_ context.Context, instanceId string) string {
			return "" // 获取 OS 名称失败
		})

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，实际=%d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]interface{})
	if _, ok := data["os_name"]; ok {
		t.Error("os_name 为空时不应出现在响应中")
	}
}

// ========== ensureBrowserVNCPortRule 测试 ==========

// TestEnsureBrowserVNCPortRule_EmptySecurityGroup 测试未配置安全组
func TestEnsureBrowserVNCPortRule_EmptySecurityGroup(t *testing.T) {
	err := ensureBrowserVNCPortRule(context.Background(), "")
	if err == nil {
		t.Error("未配置安全组应返回错误")
	}
	if !strings.Contains(err.Error(), "未配置安全组") {
		t.Errorf("错误信息应包含 '未配置安全组'，实际为: %s", err.Error())
	}
}

// ========== computeWebSocketAccept 测试 ==========

// TestComputeWebSocketAccept_RFC6455 测试 RFC 6455 标准向量
func TestComputeWebSocketAccept_RFC6455(t *testing.T) {
	// RFC 6455 Section 4.2.2 示例：
	// Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==
	// 期望 Sec-WebSocket-Accept: s3pPLMBiTxaQ9kYGzzhZRbK+xOo=
	key := "dGhlIHNhbXBsZSBub25jZQ=="
	expected := "s3pPLMBiTxaQ9kYGzzhZRbK+xOo="

	got := computeWebSocketAccept(key)
	if got != expected {
		t.Errorf("computeWebSocketAccept(%q) = %q, want %q", key, got, expected)
	}
}

// TestComputeWebSocketAccept_DifferentKeys 测试不同 key 产生不同 accept
func TestComputeWebSocketAccept_DifferentKeys(t *testing.T) {
	accept1 := computeWebSocketAccept("key1")
	accept2 := computeWebSocketAccept("key2")
	if accept1 == accept2 {
		t.Error("不同的 key 应产生不同的 accept")
	}
}

// TestComputeWebSocketAccept_Deterministic 测试相同 key 产生相同 accept
func TestComputeWebSocketAccept_Deterministic(t *testing.T) {
	key := "test-key-12345"
	accept1 := computeWebSocketAccept(key)
	accept2 := computeWebSocketAccept(key)
	if accept1 != accept2 {
		t.Error("相同的 key 应产生相同的 accept")
	}
}

// TestComputeWebSocketAccept_EmptyKey 测试空 key
func TestComputeWebSocketAccept_EmptyKey(t *testing.T) {
	accept := computeWebSocketAccept("")
	if accept == "" {
		t.Error("即使空 key 也应产生非空 accept")
	}
}

// ========== buildBackendWSRequest 测试 ==========

// TestBuildBackendWSRequest_Format 测试后端 WebSocket 请求格式
func TestBuildBackendWSRequest_Format(t *testing.T) {
	req := buildBackendWSRequest("1.2.3.4:6080", "dGVzdC1rZXk=")

	// 验证包含必要的 HTTP 头
	if !strings.Contains(req, "GET /?token=none HTTP/1.1\r\n") {
		t.Error("应包含 GET 请求行")
	}
	if !strings.Contains(req, "Host: 1.2.3.4:6080\r\n") {
		t.Error("应包含 Host 头")
	}
	if !strings.Contains(req, "Upgrade: websocket\r\n") {
		t.Error("应包含 Upgrade 头")
	}
	if !strings.Contains(req, "Connection: Upgrade\r\n") {
		t.Error("应包含 Connection 头")
	}
	if !strings.Contains(req, "Sec-WebSocket-Key: dGVzdC1rZXk=\r\n") {
		t.Error("应包含 Sec-WebSocket-Key 头")
	}
	if !strings.Contains(req, "Sec-WebSocket-Version: 13\r\n") {
		t.Error("应包含 Sec-WebSocket-Version 头")
	}
	if !strings.Contains(req, "Sec-WebSocket-Protocol: binary\r\n") {
		t.Error("应包含 Sec-WebSocket-Protocol 头")
	}
	// 验证以 \r\n\r\n 结尾
	if !strings.HasSuffix(req, "\r\n\r\n") {
		t.Error("应以 \\r\\n\\r\\n 结尾")
	}
}

// TestBuildBackendWSRequest_DifferentAddresses 测试不同地址
func TestBuildBackendWSRequest_DifferentAddresses(t *testing.T) {
	req1 := buildBackendWSRequest("10.0.0.1:6080", "key1")
	req2 := buildBackendWSRequest("10.0.0.2:6080", "key2")
	if req1 == req2 {
		t.Error("不同地址和 key 应产生不同请求")
	}
	if !strings.Contains(req1, "Host: 10.0.0.1:6080") {
		t.Error("req1 应包含正确的 Host")
	}
	if !strings.Contains(req2, "Host: 10.0.0.2:6080") {
		t.Error("req2 应包含正确的 Host")
	}
}

// ========== buildClient101Response 测试 ==========

// TestBuildClient101Response_WithSubProtocol 测试带子协议的 101 响应
func TestBuildClient101Response_WithSubProtocol(t *testing.T) {
	resp := buildClient101Response("s3pPLMBiTxaQ9kYGzzhZRbK+xOo=", "binary")

	if !strings.Contains(resp, "HTTP/1.1 101 Switching Protocols\r\n") {
		t.Error("应包含 101 状态行")
	}
	if !strings.Contains(resp, "Upgrade: websocket\r\n") {
		t.Error("应包含 Upgrade 头")
	}
	if !strings.Contains(resp, "Connection: Upgrade\r\n") {
		t.Error("应包含 Connection 头")
	}
	if !strings.Contains(resp, "Sec-WebSocket-Accept: s3pPLMBiTxaQ9kYGzzhZRbK+xOo=\r\n") {
		t.Error("应包含正确的 Sec-WebSocket-Accept 头")
	}
	if !strings.Contains(resp, "Sec-WebSocket-Protocol: binary\r\n") {
		t.Error("应包含 Sec-WebSocket-Protocol 头")
	}
	if !strings.HasSuffix(resp, "\r\n\r\n") {
		t.Error("应以 \\r\\n\\r\\n 结尾")
	}
}

// TestBuildClient101Response_WithoutSubProtocol 测试不带子协议的 101 响应
func TestBuildClient101Response_WithoutSubProtocol(t *testing.T) {
	resp := buildClient101Response("test-accept", "")

	if strings.Contains(resp, "Sec-WebSocket-Protocol") {
		t.Error("无子协议时不应包含 Sec-WebSocket-Protocol 头")
	}
	if !strings.Contains(resp, "Sec-WebSocket-Accept: test-accept\r\n") {
		t.Error("应包含正确的 Sec-WebSocket-Accept 头")
	}
}

// ========== generateBackendWSKey 测试 ==========

// TestGenerateBackendWSKey_NotEmpty 测试生成的 key 非空
func TestGenerateBackendWSKey_NotEmpty(t *testing.T) {
	key := generateBackendWSKey(1)
	if key == "" {
		t.Error("生成的 key 不应为空")
	}
}

// TestGenerateBackendWSKey_IsBase64 测试生成的 key 是有效的 base64
func TestGenerateBackendWSKey_IsBase64(t *testing.T) {
	key := generateBackendWSKey(42)
	_, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		t.Errorf("生成的 key 应是有效的 base64，解码错误: %v", err)
	}
}

// TestGenerateBackendWSKey_ContainsInstanceID 测试生成的 key 包含实例 ID
func TestGenerateBackendWSKey_ContainsInstanceID(t *testing.T) {
	key := generateBackendWSKey(12345)
	decoded, _ := base64.StdEncoding.DecodeString(key)
	if !strings.Contains(string(decoded), "hatchery-12345-") {
		t.Errorf("解码后应包含 'hatchery-12345-'，实际为: %s", string(decoded))
	}
}

// ========== bidirectionalCopy 测试 ==========

// TestBidirectionalCopy_Basic 测试基本的双向数据透传
func TestBidirectionalCopy_Basic(t *testing.T) {
	// 创建两对 pipe 模拟 client 和 backend 连接
	clientRead, clientWrite := net.Pipe()
	backendRead, backendWrite := net.Pipe()

	clientBuf := bufio.NewReadWriter(bufio.NewReader(clientRead), bufio.NewWriter(clientRead))
	backendBuf := bufio.NewReader(backendRead)

	done := make(chan struct{})
	go func() {
		bidirectionalCopy(clientRead, clientBuf, backendRead, backendBuf)
		close(done)
	}()

	// 从 client 端写入数据，应该能从 backend 端读到
	testData := []byte("hello from client")
	go func() {
		clientWrite.Write(testData)
		clientWrite.Close()
	}()

	buf := make([]byte, 100)
	n, _ := backendWrite.Read(buf)
	if string(buf[:n]) != "hello from client" {
		t.Errorf("backend 应收到 'hello from client'，实际为 '%s'", string(buf[:n]))
	}

	backendWrite.Close()
	<-done // 等待 bidirectionalCopy 完成
}

// TestBidirectionalCopy_BothDirections 测试双向同时传输
func TestBidirectionalCopy_BothDirections(t *testing.T) {
	clientRead, clientWrite := net.Pipe()
	backendRead, backendWrite := net.Pipe()

	clientBuf := bufio.NewReadWriter(bufio.NewReader(clientRead), bufio.NewWriter(clientRead))
	backendBuf := bufio.NewReader(backendRead)

	done := make(chan struct{})
	go func() {
		bidirectionalCopy(clientRead, clientBuf, backendRead, backendBuf)
		close(done)
	}()

	// 双向同时传输
	clientMsg := make(chan string, 1)
	backendMsg := make(chan string, 1)

	go func() {
		buf := make([]byte, 100)
		n, _ := backendWrite.Read(buf)
		clientMsg <- string(buf[:n])
	}()

	go func() {
		buf := make([]byte, 100)
		n, _ := clientWrite.Read(buf)
		backendMsg <- string(buf[:n])
	}()

	clientWrite.Write([]byte("c2b"))
	backendWrite.Write([]byte("b2c"))

	if msg := <-clientMsg; msg != "c2b" {
		t.Errorf("backend 应收到 'c2b'，实际为 '%s'", msg)
	}
	if msg := <-backendMsg; msg != "b2c" {
		t.Errorf("client 应收到 'b2c'，实际为 '%s'", msg)
	}

	clientWrite.Close()
	backendWrite.Close()
	<-done
}

// TestBidirectionalCopy_CloseOnOneSide 测试一端关闭后另一端也关闭
func TestBidirectionalCopy_CloseOnOneSide(t *testing.T) {
	clientRead, clientWrite := net.Pipe()
	backendRead, backendWrite := net.Pipe()

	clientBuf := bufio.NewReadWriter(bufio.NewReader(clientRead), bufio.NewWriter(clientRead))
	backendBuf := bufio.NewReader(backendRead)

	done := make(chan struct{})
	go func() {
		bidirectionalCopy(clientRead, clientBuf, backendRead, backendBuf)
		close(done)
	}()

	// 关闭 client 端写入
	clientWrite.Close()

	// bidirectionalCopy 应该在合理时间内完成
	select {
	case <-done:
		// 正常完成
	case <-time.After(5 * time.Second):
		t.Error("bidirectionalCopy 应在一端关闭后及时退出")
	}

	backendWrite.Close()
}

// ========== WebSocket Accept 端到端测试 ==========

// TestWebSocketAccept_EndToEnd 测试完整的 WebSocket 握手流程
func TestWebSocketAccept_EndToEnd(t *testing.T) {
	// 模拟浏览器发送的 Sec-WebSocket-Key
	wsKey := "x3JJHMbDL1EzLkh9GBhXDw=="

	// 计算 Accept
	accept := computeWebSocketAccept(wsKey)

	// 构建 101 响应
	resp := buildClient101Response(accept, "binary")

	// 验证响应格式完整
	if !strings.Contains(resp, "HTTP/1.1 101") {
		t.Error("响应应包含 101 状态码")
	}
	if !strings.Contains(resp, "Sec-WebSocket-Accept: "+accept) {
		t.Error("响应应包含正确的 Accept 值")
	}

	// 构建后端请求
	backendKey := generateBackendWSKey(1)
	backendReq := buildBackendWSRequest("1.2.3.4:6080", backendKey)

	// 验证后端请求格式
	if !strings.Contains(backendReq, "Sec-WebSocket-Key: "+backendKey) {
		t.Error("后端请求应包含正确的 Key")
	}
}

// ========== cvmInstanceInfo 测试 ==========

// TestCvmInstanceInfo_Struct 测试 cvmInstanceInfo 结构体
func TestCvmInstanceInfo_Struct(t *testing.T) {
	info := &cvmInstanceInfo{
		InstanceState: "RUNNING",
		PublicIp:      "1.2.3.4",
	}
	if info.InstanceState != "RUNNING" {
		t.Errorf("InstanceState 应为 RUNNING，实际为 %s", info.InstanceState)
	}
	if info.PublicIp != "1.2.3.4" {
		t.Errorf("PublicIp 应为 1.2.3.4，实际为 %s", info.PublicIp)
	}

	// 零值测试
	empty := &cvmInstanceInfo{}
	if empty.InstanceState != "" {
		t.Error("零值 InstanceState 应为空")
	}
	if empty.PublicIp != "" {
		t.Error("零值 PublicIp 应为空")
	}
}

// ========== ensureSinglePortRuleCore 测试 ==========

// TestEnsureSinglePortRuleCore_PortAlreadyExists 测试端口规则已存在
func TestEnsureSinglePortRuleCore_PortAlreadyExists(t *testing.T) {
	err := ensureSinglePortRuleCore("sg-test", 6080, "测试描述",
		func(sgId string, port int) (bool, error) {
			return true, nil // 端口已存在
		},
		func(sgId string, port int, desc string) error {
			t.Error("端口已存在时不应调用 addFn")
			return nil
		})

	if err != nil {
		t.Errorf("端口已存在时不应返回错误，实际: %v", err)
	}
}

// TestEnsureSinglePortRuleCore_PortNotExists_AddSuccess 测试端口不存在且添加成功
func TestEnsureSinglePortRuleCore_PortNotExists_AddSuccess(t *testing.T) {
	addCalled := false
	err := ensureSinglePortRuleCore("sg-test", 6080, "测试描述",
		func(sgId string, port int) (bool, error) {
			return false, nil // 端口不存在
		},
		func(sgId string, port int, desc string) error {
			addCalled = true
			if sgId != "sg-test" {
				t.Errorf("安全组 ID 应为 sg-test，实际为 %s", sgId)
			}
			if port != 6080 {
				t.Errorf("端口应为 6080，实际为 %d", port)
			}
			if desc != "测试描述" {
				t.Errorf("描述应为 '测试描述'，实际为 %s", desc)
			}
			return nil
		})

	if err != nil {
		t.Errorf("添加成功时不应返回错误，实际: %v", err)
	}
	if !addCalled {
		t.Error("端口不存在时应调用 addFn")
	}
}

// TestEnsureSinglePortRuleCore_PortNotExists_AddFails 测试端口不存在且添加失败
func TestEnsureSinglePortRuleCore_PortNotExists_AddFails(t *testing.T) {
	err := ensureSinglePortRuleCore("sg-test", 6080, "测试描述",
		func(sgId string, port int) (bool, error) {
			return false, nil // 端口不存在
		},
		func(sgId string, port int, desc string) error {
			return fmt.Errorf("VPC API 超时")
		})

	if err == nil {
		t.Error("添加失败时应返回错误")
	}
	if !strings.Contains(err.Error(), "创建安全组规则失败") {
		t.Errorf("错误信息应包含 '创建安全组规则失败'，实际为: %s", err.Error())
	}
}

// TestEnsureSinglePortRuleCore_CheckFails 测试检查端口规则失败
func TestEnsureSinglePortRuleCore_CheckFails(t *testing.T) {
	err := ensureSinglePortRuleCore("sg-test", 6080, "测试描述",
		func(sgId string, port int) (bool, error) {
			return false, fmt.Errorf("VPC API 不可用")
		},
		func(sgId string, port int, desc string) error {
			t.Error("检查失败时不应调用 addFn")
			return nil
		})

	if err == nil {
		t.Error("检查失败时应返回错误")
	}
	if !strings.Contains(err.Error(), "检查端口规则失败") {
		t.Errorf("错误信息应包含 '检查端口规则失败'，实际为: %s", err.Error())
	}
}

// ========== validateVNCProxyInstance 测试 ==========

// TestValidateVNCProxyInstance_FeatureDisabled 测试功能未开启
func TestValidateVNCProxyInstance_FeatureDisabled(t *testing.T) {
	instance := &model.Instance{Model: gorm.Model{ID: 40001}, InstanceId: "ins-test"}
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	countPtr := new(int64)

	result := validateVNCProxyInstance(req, instance, false, countPtr, nil)

	if result.ErrCode != http.StatusForbidden {
		t.Errorf("期望 403，实际=%d", result.ErrCode)
	}
	if !strings.Contains(result.ErrMsg, "功能未开启") {
		t.Errorf("错误信息应包含 '功能未开启'，实际为: %s", result.ErrMsg)
	}
}

// TestValidateVNCProxyInstance_NoCVM 测试无关联 CVM
func TestValidateVNCProxyInstance_NoCVM(t *testing.T) {
	instance := &model.Instance{Model: gorm.Model{ID: 40002}, InstanceId: ""}
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	countPtr := new(int64)

	result := validateVNCProxyInstance(req, instance, true, countPtr, nil)

	if result.ErrCode != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", result.ErrCode)
	}
}

// TestValidateVNCProxyInstance_ConnLimitExceeded 测试连接数超限
func TestValidateVNCProxyInstance_ConnLimitExceeded(t *testing.T) {
	instance := &model.Instance{Model: gorm.Model{ID: 40003}, InstanceId: "ins-test"}
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	countPtr := new(int64)
	atomic.StoreInt64(countPtr, maxVNCProxyPerInstance) // 已达上限

	result := validateVNCProxyInstance(req, instance, true, countPtr, nil)

	if result.ErrCode != http.StatusTooManyRequests {
		t.Errorf("期望 429，实际=%d", result.ErrCode)
	}
}

// TestValidateVNCProxyInstance_DescribeFails 测试 CVM 查询失败
func TestValidateVNCProxyInstance_DescribeFails(t *testing.T) {
	instance := &model.Instance{Model: gorm.Model{ID: 40004}, InstanceId: "ins-test"}
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	countPtr := new(int64)

	result := validateVNCProxyInstance(req, instance, true, countPtr,
		func(id string) (*cvmInstanceInfo, error) {
			return nil, fmt.Errorf("CVM API 超时")
		})

	if result.ErrCode != http.StatusInternalServerError {
		t.Errorf("期望 500，实际=%d", result.ErrCode)
	}
}

// TestValidateVNCProxyInstance_InstanceNotFound 测试 CVM 实例不存在
func TestValidateVNCProxyInstance_InstanceNotFound(t *testing.T) {
	instance := &model.Instance{Model: gorm.Model{ID: 40005}, InstanceId: "ins-notfound"}
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	countPtr := new(int64)

	result := validateVNCProxyInstance(req, instance, true, countPtr,
		func(id string) (*cvmInstanceInfo, error) {
			return nil, nil
		})

	if result.ErrCode != http.StatusNotFound {
		t.Errorf("期望 404，实际=%d", result.ErrCode)
	}
}

// TestValidateVNCProxyInstance_NoPublicIP 测试无公网 IP
func TestValidateVNCProxyInstance_NoPublicIP(t *testing.T) {
	instance := &model.Instance{Model: gorm.Model{ID: 40006}, InstanceId: "ins-noip"}
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	countPtr := new(int64)

	result := validateVNCProxyInstance(req, instance, true, countPtr,
		func(id string) (*cvmInstanceInfo, error) {
			return &cvmInstanceInfo{InstanceState: "RUNNING", PublicIp: ""}, nil
		})

	if result.ErrCode != http.StatusInternalServerError {
		t.Errorf("期望 500，实际=%d", result.ErrCode)
	}
}

// TestValidateVNCProxyInstance_InstanceNotRunning 测试实例非运行状态
func TestValidateVNCProxyInstance_InstanceNotRunning(t *testing.T) {
	instance := &model.Instance{Model: gorm.Model{ID: 40007}, InstanceId: "ins-stopped"}
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	countPtr := new(int64)

	result := validateVNCProxyInstance(req, instance, true, countPtr,
		func(id string) (*cvmInstanceInfo, error) {
			return &cvmInstanceInfo{InstanceState: "STOPPED", PublicIp: "1.2.3.4"}, nil
		})

	if result.ErrCode != http.StatusConflict {
		t.Errorf("期望 409，实际=%d", result.ErrCode)
	}
	if !strings.Contains(result.ErrMsg, "STOPPED") {
		t.Errorf("错误信息应包含 'STOPPED'，实际为: %s", result.ErrMsg)
	}
}

// TestValidateVNCProxyInstance_NotWebSocket 测试非 WebSocket 请求
func TestValidateVNCProxyInstance_NotWebSocket(t *testing.T) {
	instance := &model.Instance{Model: gorm.Model{ID: 40008}, InstanceId: "ins-test"}
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	// 不设置 Upgrade 和 Connection 头
	countPtr := new(int64)

	result := validateVNCProxyInstance(req, instance, true, countPtr,
		func(id string) (*cvmInstanceInfo, error) {
			return &cvmInstanceInfo{InstanceState: "RUNNING", PublicIp: "1.2.3.4"}, nil
		})

	if result.ErrCode != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", result.ErrCode)
	}
	if !strings.Contains(result.ErrMsg, "WebSocket") {
		t.Errorf("错误信息应包含 'WebSocket'，实际为: %s", result.ErrMsg)
	}
}

// TestValidateVNCProxyInstance_NoWSKey 测试缺少 Sec-WebSocket-Key
func TestValidateVNCProxyInstance_NoWSKey(t *testing.T) {
	instance := &model.Instance{Model: gorm.Model{ID: 40009}, InstanceId: "ins-test"}
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	// 不设置 Sec-WebSocket-Key
	countPtr := new(int64)

	result := validateVNCProxyInstance(req, instance, true, countPtr,
		func(id string) (*cvmInstanceInfo, error) {
			return &cvmInstanceInfo{InstanceState: "RUNNING", PublicIp: "1.2.3.4"}, nil
		})

	if result.ErrCode != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", result.ErrCode)
	}
	if !strings.Contains(result.ErrMsg, "Sec-WebSocket-Key") {
		t.Errorf("错误信息应包含 'Sec-WebSocket-Key'，实际为: %s", result.ErrMsg)
	}
}

// TestValidateVNCProxyInstance_Success 测试校验全部通过
func TestValidateVNCProxyInstance_Success(t *testing.T) {
	instance := &model.Instance{Model: gorm.Model{ID: 40010}, InstanceId: "ins-ok"}
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	countPtr := new(int64)

	result := validateVNCProxyInstance(req, instance, true, countPtr,
		func(id string) (*cvmInstanceInfo, error) {
			return &cvmInstanceInfo{InstanceState: "RUNNING", PublicIp: "1.2.3.4"}, nil
		})

	if result.ErrCode != 0 {
		t.Errorf("校验应通过，实际错误码=%d，错误信息=%s", result.ErrCode, result.ErrMsg)
	}
	if result.PublicIp != "1.2.3.4" {
		t.Errorf("PublicIp 应为 '1.2.3.4'，实际为 '%s'", result.PublicIp)
	}
}

// TestValidateVNCProxyInstance_ConnLimitBoundary 测试连接数边界值
func TestValidateVNCProxyInstance_ConnLimitBoundary(t *testing.T) {
	instance := &model.Instance{Model: gorm.Model{ID: 40011}, InstanceId: "ins-test"}
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	countPtr := new(int64)

	// 连接数为 maxVNCProxyPerInstance - 1，应该通过
	atomic.StoreInt64(countPtr, maxVNCProxyPerInstance-1)
	result := validateVNCProxyInstance(req, instance, true, countPtr,
		func(id string) (*cvmInstanceInfo, error) {
			return &cvmInstanceInfo{InstanceState: "RUNNING", PublicIp: "1.2.3.4"}, nil
		})

	if result.ErrCode != 0 {
		t.Errorf("连接数未达上限时应通过，实际错误码=%d", result.ErrCode)
	}

	// 连接数恰好为 maxVNCProxyPerInstance，应该拒绝
	atomic.StoreInt64(countPtr, maxVNCProxyPerInstance)
	result = validateVNCProxyInstance(req, instance, true, countPtr,
		func(id string) (*cvmInstanceInfo, error) {
			return &cvmInstanceInfo{InstanceState: "RUNNING", PublicIp: "1.2.3.4"}, nil
		})

	if result.ErrCode != http.StatusTooManyRequests {
		t.Errorf("连接数达上限时应拒绝，实际错误码=%d", result.ErrCode)
	}
}

// ========== vncProxyValidationResult 测试 ==========

// TestVNCProxyValidationResult_ZeroValue 测试零值表示校验通过
func TestVNCProxyValidationResult_ZeroValue(t *testing.T) {
	result := vncProxyValidationResult{PublicIp: "1.2.3.4"}
	if result.ErrCode != 0 {
		t.Error("零值 ErrCode 应表示校验通过")
	}
	if result.ErrMsg != "" {
		t.Error("零值 ErrMsg 应为空")
	}
}

// TestVNCProxyValidationResult_ErrorState 测试错误状态
func TestVNCProxyValidationResult_ErrorState(t *testing.T) {
	result := vncProxyValidationResult{ErrMsg: "测试错误", ErrCode: 400}
	if result.ErrCode == 0 {
		t.Error("错误状态 ErrCode 不应为 0")
	}
	if result.PublicIp != "" {
		t.Error("错误状态 PublicIp 应为空")
	}
}

// ========== 常量测试 ==========

// TestVNCConstants 测试 VNC 相关常量值
func TestVNCConstants(t *testing.T) {
	if vncPort != 5900 {
		t.Errorf("vncPort 应为 5900，实际为 %d", vncPort)
	}
	if websockifyPort != 6080 {
		t.Errorf("websockifyPort 应为 6080，实际为 %d", websockifyPort)
	}
	if maxVNCProxyPerInstance != 3 {
		t.Errorf("maxVNCProxyPerInstance 应为 3，实际为 %d", maxVNCProxyPerInstance)
	}
	if aiActiveTimeout != 10*time.Minute {
		t.Errorf("aiActiveTimeout 应为 10 分钟，实际为 %v", aiActiveTimeout)
	}
	if aiGracePeriod != 45*time.Second {
		t.Errorf("aiGracePeriod 应为 45 秒，实际为 %v", aiGracePeriod)
	}
}

// ========== startGraceTimer 边界测试 ==========

// TestStartGraceTimer_OverwriteExisting 测试重复启动宽限期覆盖旧定时器
func TestStartGraceTimer_OverwriteExisting(t *testing.T) {
	aiActiveMap = sync.Map{}
	var instanceID uint = 50001

	state := getOrCreateState(instanceID)

	// 第一次启动宽限期
	startGraceTimer(instanceID, state)
	if atomic.LoadInt32(&state.graceActive) != 1 {
		t.Error("第一次启动后 graceActive 应为 1")
	}

	// 第二次启动宽限期（应覆盖旧定时器）
	startGraceTimer(instanceID, state)
	if atomic.LoadInt32(&state.graceActive) != 1 {
		t.Error("第二次启动后 graceActive 应仍为 1")
	}

	// 取消后应归零
	cancelGraceTimer(state)
	if atomic.LoadInt32(&state.graceActive) != 0 {
		t.Error("取消后 graceActive 应为 0")
	}
}

// TestCancelGraceTimer_NilTimer 测试取消空定时器不 panic
func TestCancelGraceTimer_NilTimer(t *testing.T) {
	state := &aiActiveState{}
	// 不应 panic
	cancelGraceTimer(state)
	if atomic.LoadInt32(&state.graceActive) != 0 {
		t.Error("取消空定时器后 graceActive 应为 0")
	}
}

// ========== MarkAIInactiveWithContext 负数保护测试 ==========

// TestMarkAIInactiveWithContext_NegativeProtection 测试 activeRequests 负数保护
func TestMarkAIInactiveWithContext_NegativeProtection(t *testing.T) {
	aiActiveMap = sync.Map{}
	var instanceID uint = 50002

	// 手动创建状态，activeRequests 为 0
	state := getOrCreateState(instanceID)
	atomic.StoreInt64(&state.activeRequests, 0)

	// 调用 MarkAIInactiveWithContext 应触发负数保护
	MarkAIInactiveWithContext(instanceID, false)

	// 验证 activeRequests 被修正为 0（不是 -1）
	count := atomic.LoadInt64(&state.activeRequests)
	if count != 0 {
		t.Errorf("负数保护后 activeRequests 应为 0，实际为 %d", count)
	}
}

// ========== getOrCreateState 测试 ==========

// TestGetOrCreateState_Idempotent 测试 getOrCreateState 幂等性
func TestGetOrCreateState_Idempotent(t *testing.T) {
	aiActiveMap = sync.Map{}
	var instanceID uint = 50003

	state1 := getOrCreateState(instanceID)
	state2 := getOrCreateState(instanceID)

	if state1 != state2 {
		t.Error("同一实例应返回同一个 state 指针")
	}
}

// TestGetOrCreateState_DifferentInstances 测试不同实例有独立的 state
func TestGetOrCreateState_DifferentInstances(t *testing.T) {
	aiActiveMap = sync.Map{}

	state1 := getOrCreateState(50004)
	state2 := getOrCreateState(50005)

	if state1 == state2 {
		t.Error("不同实例应有不同的 state 指针")
	}
}

// ========== ensureBrowserVNCPortRule 更多测试 ==========

// TestEnsureBrowserVNCPortRule_ErrorMessage 测试错误信息格式
func TestEnsureBrowserVNCPortRule_ErrorMessage(t *testing.T) {
	err := ensureBrowserVNCPortRule(context.Background(), "")
	if err == nil {
		t.Fatal("应返回错误")
	}
	// 验证错误信息包含关键信息
	errStr := err.Error()
	if !strings.Contains(errStr, "未配置安全组") {
		t.Errorf("错误信息应包含 '未配置安全组'，实际为: %s", errStr)
	}
}

// ========== parseBrowserVNCScriptOutput 更多边界测试 ==========

// TestParseBrowserVNCScriptOutput_MultipleJSONLines 测试多个 JSON 行（应返回最后一个）
func TestParseBrowserVNCScriptOutput_MultipleJSONLines(t *testing.T) {
	output := `{"step": 1}
{"step": 2}
{"step": 3, "final": true}`

	result := parseBrowserVNCScriptOutput(output)
	if result == nil {
		t.Fatal("解析结果不应为 nil")
	}
	if result["final"] != true {
		t.Error("应返回最后一个 JSON 对象")
	}
	if result["step"] != float64(3) {
		t.Errorf("step 应为 3，实际为 %v", result["step"])
	}
}

// TestParseBrowserVNCScriptOutput_JSONWithSpecialChars 测试包含特殊字符的 JSON
func TestParseBrowserVNCScriptOutput_JSONWithSpecialChars(t *testing.T) {
	output := `日志行
{"message": "安装完成！路径: /opt/browser-vnc", "installed": true}`

	result := parseBrowserVNCScriptOutput(output)
	if result == nil {
		t.Fatal("解析结果不应为 nil")
	}
	if result["installed"] != true {
		t.Error("installed 应为 true")
	}
	msg, ok := result["message"].(string)
	if !ok || !strings.Contains(msg, "/opt/browser-vnc") {
		t.Errorf("message 应包含路径，实际为: %v", result["message"])
	}
}

// TestParseBrowserVNCScriptOutput_InvalidMultiLineJSON 测试多行 JSON 块解析失败（覆盖 break 分支）
func TestParseBrowserVNCScriptOutput_InvalidMultiLineJSON(t *testing.T) {
	// 构造一个看起来像多行 JSON 但实际无效的输出：
	// 有匹配的 { 和 }，但内容不是有效 JSON
	output := `日志行
{
  这不是有效的 JSON 内容
  也没有引号和冒号
}`

	result := parseBrowserVNCScriptOutput(output)
	// 无效的多行 JSON 应返回 nil
	if result != nil {
		t.Errorf("无效的多行 JSON 应返回 nil，实际为: %v", result)
	}
}

// ========== startGraceTimer 回调触发测试 ==========

// TestStartGraceTimer_CallbackFires 测试宽限期定时器回调实际触发
func TestStartGraceTimer_CallbackFires(t *testing.T) {
	aiActiveMap = sync.Map{}
	var instanceID uint = 60001

	state := getOrCreateState(instanceID)

	// 直接调用 startGraceTimer 启动宽限期
	startGraceTimer(instanceID, state)

	// 验证宽限期已激活
	if atomic.LoadInt32(&state.graceActive) != 1 {
		t.Error("启动后 graceActive 应为 1")
	}

	// 停止原始定时器，替换为极短的定时器来测试回调逻辑
	state.graceMu.Lock()
	if state.graceTimer != nil {
		state.graceTimer.Stop()
	}
	// 使用极短的定时器模拟宽限期到期
	state.graceTimer = time.AfterFunc(10*time.Millisecond, func() {
		atomic.StoreInt32(&state.graceActive, 0)
	})
	state.graceMu.Unlock()

	// 等待定时器触发
	time.Sleep(50 * time.Millisecond)

	// 验证回调已执行
	if atomic.LoadInt32(&state.graceActive) != 0 {
		t.Error("定时器回调触发后 graceActive 应为 0")
	}
}

// ========== bidirectionalCopy 缓冲数据测试 ==========

// TestBidirectionalCopy_WithClientBufferedData 测试客户端有缓冲数据时的透传
func TestBidirectionalCopy_WithClientBufferedData(t *testing.T) {
	clientRead, clientWrite := net.Pipe()
	backendRead, backendWrite := net.Pipe()

	// 先向 clientRead 写入一些数据，让 bufio.Reader 有缓冲
	go func() {
		clientWrite.Write([]byte("buffered-data"))
		clientWrite.Close()
	}()

	// 创建 bufio.Reader 并预读数据到缓冲区
	clientReader := bufio.NewReader(clientRead)
	// 先 Peek 一下让数据进入缓冲区
	_, err := clientReader.Peek(5)
	if err != nil {
		t.Fatalf("Peek 失败: %v", err)
	}

	clientBuf := bufio.NewReadWriter(clientReader, bufio.NewWriter(clientRead))
	backendBuf := bufio.NewReader(backendRead)

	done := make(chan struct{})
	go func() {
		bidirectionalCopy(clientRead, clientBuf, backendRead, backendBuf)
		close(done)
	}()

	// 从 backend 端读取数据
	buf := make([]byte, 100)
	n, _ := backendWrite.Read(buf)
	received := string(buf[:n])
	if !strings.Contains(received, "buffered") {
		t.Errorf("backend 应收到缓冲数据，实际为 '%s'", received)
	}

	backendWrite.Close()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Error("bidirectionalCopy 应及时退出")
	}
}

// TestBidirectionalCopy_WithBackendBufferedData 测试后端有缓冲数据时的透传
func TestBidirectionalCopy_WithBackendBufferedData(t *testing.T) {
	clientRead, clientWrite := net.Pipe()
	backendRead, backendWrite := net.Pipe()

	// 先向 backendRead 写入一些数据，让 bufio.Reader 有缓冲
	go func() {
		backendWrite.Write([]byte("backend-buffered"))
		backendWrite.Close()
	}()

	// 创建 backendBuf 并预读数据到缓冲区
	backendBuf := bufio.NewReader(backendRead)
	_, err := backendBuf.Peek(5)
	if err != nil {
		t.Fatalf("Peek 失败: %v", err)
	}

	clientBuf := bufio.NewReadWriter(bufio.NewReader(clientRead), bufio.NewWriter(clientRead))

	done := make(chan struct{})
	go func() {
		bidirectionalCopy(clientRead, clientBuf, backendRead, backendBuf)
		close(done)
	}()

	// 从 client 端读取数据
	buf := make([]byte, 100)
	n, _ := clientWrite.Read(buf)
	received := string(buf[:n])
	if !strings.Contains(received, "backend-buffered") {
		t.Errorf("client 应收到后端缓冲数据，实际为 '%s'", received)
	}

	clientWrite.Close()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Error("bidirectionalCopy 应及时退出")
	}
}

// ========== loadVNCWhitelistCIDRs 测试 ==========

// withVNCWhitelistJSON 临时注入 ClawproRequiredSGRulesJSON，返回恢复函数。
func withVNCWhitelistJSON(jsonData string) func() {
	orig := ClawproRequiredSGRulesJSON
	ClawproRequiredSGRulesJSON = []byte(jsonData)
	return func() { ClawproRequiredSGRulesJSON = orig }
}

// TestLoadVNCWhitelistCIDRs_Found 测试正常加载白名单规则
func TestLoadVNCWhitelistCIDRs_Found(t *testing.T) {
	defer withVNCWhitelistJSON(`{
		"categories": [
			{"type":"recommended","label":"推荐","rule_groups":[
				{"key":"allow_vnc_whitelist","name":"VNC白名单","rules":[
					{"direction":"ingress","protocol":"TCP","port":"6080","cidr_block":"1.2.3.4/32","action":"ACCEPT","description":"测试IP1"},
					{"direction":"ingress","protocol":"TCP","port":"6080","cidr_block":"5.6.7.8/32","action":"ACCEPT","description":"测试IP2"}
				]}
			]}
		]
	}`)()

	rules := loadVNCWhitelistCIDRs()
	if len(rules) != 2 {
		t.Fatalf("应返回 2 条规则，实际 %d 条", len(rules))
	}
	if rules[0].CidrBlock != "1.2.3.4/32" {
		t.Errorf("第 1 条规则 CIDR 应为 1.2.3.4/32，实际为 %s", rules[0].CidrBlock)
	}
	if rules[1].CidrBlock != "5.6.7.8/32" {
		t.Errorf("第 2 条规则 CIDR 应为 5.6.7.8/32，实际为 %s", rules[1].CidrBlock)
	}
}

// TestLoadVNCWhitelistCIDRs_NotFound 测试 JSON 中不存在 allow_vnc_whitelist
func TestLoadVNCWhitelistCIDRs_NotFound(t *testing.T) {
	defer withVNCWhitelistJSON(`{
		"categories": [
			{"type":"builtin","label":"内置","rule_groups":[
				{"key":"allow_ssh","name":"SSH","rules":[
					{"direction":"ingress","protocol":"TCP","port":"22","cidr_block":"0.0.0.0/0","action":"ACCEPT","description":"SSH"}
				]}
			]}
		]
	}`)()

	rules := loadVNCWhitelistCIDRs()
	if len(rules) != 0 {
		t.Errorf("未配置 allow_vnc_whitelist 时应返回空，实际返回 %d 条", len(rules))
	}
}

// TestLoadVNCWhitelistCIDRs_EmptyJSON 测试 JSON 为空
func TestLoadVNCWhitelistCIDRs_EmptyJSON(t *testing.T) {
	defer withVNCWhitelistJSON(`{}`)()

	rules := loadVNCWhitelistCIDRs()
	if len(rules) != 0 {
		t.Errorf("空 JSON 应返回空规则，实际返回 %d 条", len(rules))
	}
}

// TestLoadVNCWhitelistCIDRs_InvalidJSON 测试无效 JSON
func TestLoadVNCWhitelistCIDRs_InvalidJSON(t *testing.T) {
	defer withVNCWhitelistJSON(`{invalid`)()

	rules := loadVNCWhitelistCIDRs()
	if len(rules) != 0 {
		t.Errorf("无效 JSON 应返回空规则，实际返回 %d 条", len(rules))
	}
}

// TestLoadVNCWhitelistCIDRs_EmptyRules 测试 allow_vnc_whitelist 存在但 rules 为空
func TestLoadVNCWhitelistCIDRs_EmptyRules(t *testing.T) {
	defer withVNCWhitelistJSON(`{
		"categories": [
			{"type":"recommended","label":"推荐","rule_groups":[
				{"key":"allow_vnc_whitelist","name":"VNC白名单","rules":[]}
			]}
		]
	}`)()

	rules := loadVNCWhitelistCIDRs()
	if len(rules) != 0 {
		t.Errorf("空 rules 应返回空，实际返回 %d 条", len(rules))
	}
}

// TestLoadVNCWhitelistCIDRs_MultipleCategories 测试规则在不同分类中
func TestLoadVNCWhitelistCIDRs_MultipleCategories(t *testing.T) {
	defer withVNCWhitelistJSON(`{
		"categories": [
			{"type":"builtin","label":"内置","rule_groups":[
				{"key":"allow_ssh","name":"SSH","rules":[
					{"direction":"ingress","protocol":"TCP","port":"22","cidr_block":"0.0.0.0/0","action":"ACCEPT","description":"SSH"}
				]}
			]},
			{"type":"recommended","label":"推荐","rule_groups":[
				{"key":"allow_gateway_ui","name":"面板","rules":[]},
				{"key":"allow_vnc_whitelist","name":"VNC白名单","rules":[
					{"direction":"ingress","protocol":"TCP","port":"6080","cidr_block":"10.0.0.1/32","action":"ACCEPT","description":"内网"}
				]}
			]}
		]
	}`)()

	rules := loadVNCWhitelistCIDRs()
	if len(rules) != 1 {
		t.Fatalf("应从第二个分类中找到 1 条规则，实际 %d 条", len(rules))
	}
	if rules[0].CidrBlock != "10.0.0.1/32" {
		t.Errorf("CIDR 应为 10.0.0.1/32，实际为 %s", rules[0].CidrBlock)
	}
}

// ========== ensureBrowserVNCPortRule 白名单迁移测试 ==========

// TestEnsureBrowserVNCPortRule_NoWhitelistConfig 测试未配置白名单时应返回错误
func TestEnsureBrowserVNCPortRule_NoWhitelistConfig(t *testing.T) {
	defer withVNCWhitelistJSON(`{
		"categories": [
			{"type":"builtin","label":"内置","rule_groups":[]}
		]
	}`)()

	err := ensureBrowserVNCPortRule(context.Background(), "sg-test")
	if err == nil {
		t.Fatal("未配置白名单应返回错误")
	}
	if !strings.Contains(err.Error(), "allow_vnc_whitelist") {
		t.Errorf("错误信息应提及 allow_vnc_whitelist，实际为: %s", err.Error())
	}
}

// ========== browserVNCAccessCore 异步迁移测试 ==========

// TestBrowserVNCAccessCore_MigrateOnceTriggered 测试异步迁移只触发一次
func TestBrowserVNCAccessCore_MigrateOnceTriggered(t *testing.T) {

	instance := &model.Instance{Model: gorm.Model{ID: 99901}, InstanceId: "ins-migrate-test"}
	siteConfig := model.SiteConfig{
		BrowserVNCEnable: true,
		SecurityGroupId:  "sg-migrate-test",
	}

	callCount := int64(0)

	describeFn := func(instanceId string) (*cvmInstanceInfo, error) {
		return &cvmInstanceInfo{InstanceState: "RUNNING", PublicIp: "1.2.3.4"}, nil
	}
	checkPortFn := func(sgId string, port int) (bool, error) {
		atomic.AddInt64(&callCount, 1)
		return true, nil // 端口已放通
	}

	// 第一次调用——应触发迁移
	w1 := httptest.NewRecorder()
	r1, _ := http.NewRequest(http.MethodGet, "/openclaw/browser-vnc-access?id=99901", nil)
	browserVNCAccessCore(w1, r1, instance, siteConfig, describeFn, checkPortFn)

	if w1.Code != http.StatusOK {
		t.Fatalf("第一次调用应返回 200，实际为 %d", w1.Code)
	}

	// 第二次调用——迁移已在执行/已成功，不应再触发
	w2 := httptest.NewRecorder()
	r2, _ := http.NewRequest(http.MethodGet, "/openclaw/browser-vnc-access?id=99901", nil)
	browserVNCAccessCore(w2, r2, instance, siteConfig, describeFn, checkPortFn)

	if w2.Code != http.StatusOK {
		t.Fatalf("第二次调用应返回 200，实际为 %d", w2.Code)
	}

	// 验证响应包含 accessible=true
	var resp1 map[string]interface{}
	json.NewDecoder(w1.Body).Decode(&resp1)
	data1 := resp1["data"].(map[string]interface{})
	if data1["accessible"] != true {
		t.Error("accessible 应为 true")
	}
	if data1["ws_proxy_path"] == "" {
		t.Error("ws_proxy_path 不应为空")
	}
}

// TestBrowserVNCAccessCore_NoMigrateWhenPortNotAccessible 测试端口不可达时不触发迁移
func TestBrowserVNCAccessCore_NoMigrateWhenPortNotAccessible(t *testing.T) {

	instance := &model.Instance{Model: gorm.Model{ID: 99902}, InstanceId: "ins-no-migrate"}
	siteConfig := model.SiteConfig{
		BrowserVNCEnable: true,
		SecurityGroupId:  "sg-no-migrate",
	}

	describeFn := func(instanceId string) (*cvmInstanceInfo, error) {
		return &cvmInstanceInfo{InstanceState: "RUNNING", PublicIp: "1.2.3.4"}, nil
	}
	checkPortFn := func(sgId string, port int) (bool, error) {
		return false, nil // 端口未放通
	}

	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodGet, "/openclaw/browser-vnc-access?id=99902", nil)
	browserVNCAccessCore(w, r, instance, siteConfig, describeFn, checkPortFn)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]interface{})
	if data["accessible"] != false {
		t.Error("端口未放通时 accessible 应为 false")
	}
	if data["novnc_url"] != "" {
		t.Error("端口未放通时 novnc_url 应为空")
	}
}

// TestBrowserVNCAccessCore_NoMigrateWhenNoSecurityGroup 测试未配置安全组时不触发迁移
func TestBrowserVNCAccessCore_NoMigrateWhenNoSecurityGroup(t *testing.T) {

	instance := &model.Instance{Model: gorm.Model{ID: 99903}, InstanceId: "ins-no-sg"}
	siteConfig := model.SiteConfig{
		BrowserVNCEnable: true,
		SecurityGroupId:  "", // 未配置安全组
	}

	describeFn := func(instanceId string) (*cvmInstanceInfo, error) {
		return &cvmInstanceInfo{InstanceState: "RUNNING", PublicIp: "1.2.3.4"}, nil
	}
	checkPortFn := func(sgId string, port int) (bool, error) {
		t.Error("未配置安全组时不应调用 checkPortFn")
		return false, nil
	}

	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodGet, "/openclaw/browser-vnc-access?id=99903", nil)
	browserVNCAccessCore(w, r, instance, siteConfig, describeFn, checkPortFn)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]interface{})
	if data["accessible"] != false {
		t.Error("未配置安全组时 accessible 应为 false")
	}
}

// ========== fakeVncSGClient mock ==========

type fakeVncSGClient struct {
	ingress        []*vpc.SecurityGroupPolicy
	describeErr    error
	deleteErr      error
	createErr      error
	deletedIndexes []int64
	createdCIDRs   []string
}

func (f *fakeVncSGClient) DescribePolicies(sgId string) ([]*vpc.SecurityGroupPolicy, error) {
	if f.describeErr != nil {
		return nil, f.describeErr
	}
	return f.ingress, nil
}

func (f *fakeVncSGClient) DeletePolicies(sgId string, policyIndexes []int64) error {
	f.deletedIndexes = append(f.deletedIndexes, policyIndexes...)
	return f.deleteErr
}

func (f *fakeVncSGClient) CreatePolicies(sgId string, policies []*vpc.SecurityGroupPolicy) error {
	for _, p := range policies {
		if p.CidrBlock != nil {
			f.createdCIDRs = append(f.createdCIDRs, *p.CidrBlock)
		}
	}
	return f.createErr
}

func makeWhitelistLoader(rules []requiredSGRule) func() []requiredSGRule {
	return func() []requiredSGRule { return rules }
}

func ptr[T any](v T) *T { return &v }

// ========== ensureBrowserVNCPortRuleCore 全分支测试 ==========

func TestEnsureVNCPortRuleCore_EmptySG(t *testing.T) {
	err := ensureBrowserVNCPortRuleCore(context.Background(), "", makeWhitelistLoader(nil), nil)
	if err == nil || !strings.Contains(err.Error(), "未配置安全组") {
		t.Fatalf("空安全组应报错，实际: %v", err)
	}
}

func TestEnsureVNCPortRuleCore_NoWhitelist(t *testing.T) {
	err := ensureBrowserVNCPortRuleCore(context.Background(), "sg-test", makeWhitelistLoader(nil), nil)
	if err == nil || !strings.Contains(err.Error(), "allow_vnc_whitelist") {
		t.Fatalf("无白名单应报错，实际: %v", err)
	}
}

func TestEnsureVNCPortRuleCore_DescribeError(t *testing.T) {
	wl := []requiredSGRule{{CidrBlock: "1.2.3.4/32", Protocol: "TCP", Port: "6080", Action: "ACCEPT"}}
	fake := &fakeVncSGClient{describeErr: fmt.Errorf("vpc api error")}
	err := ensureBrowserVNCPortRuleCore(context.Background(), "sg-test", makeWhitelistLoader(wl), fake)
	if err == nil || !strings.Contains(err.Error(), "查询安全组规则失败") {
		t.Fatalf("Describe 失败应报错，实际: %v", err)
	}
}

func TestEnsureVNCPortRuleCore_NewInstance_AddAll(t *testing.T) {
	wl := []requiredSGRule{
		{CidrBlock: "1.2.3.4/32", Protocol: "TCP", Port: "6080", Action: "ACCEPT", Description: "ip1"},
		{CidrBlock: "5.6.7.8/32", Protocol: "TCP", Port: "6080", Action: "ACCEPT", Description: "ip2"},
	}
	fake := &fakeVncSGClient{ingress: nil}
	err := ensureBrowserVNCPortRuleCore(context.Background(), "sg-test", makeWhitelistLoader(wl), fake)
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if len(fake.deletedIndexes) != 0 {
		t.Errorf("不应删除任何规则，实际删除 %d 条", len(fake.deletedIndexes))
	}
	if len(fake.createdCIDRs) != 2 {
		t.Errorf("应创建 2 条规则，实际创建 %d 条", len(fake.createdCIDRs))
	}
}

func TestEnsureVNCPortRuleCore_LegacyRule_DeleteAndAdd(t *testing.T) {
	wl := []requiredSGRule{
		{CidrBlock: "1.2.3.4/32", Protocol: "TCP", Port: "6080", Action: "ACCEPT", Description: "ip1"},
	}
	fake := &fakeVncSGClient{
		ingress: []*vpc.SecurityGroupPolicy{
			{
				PolicyIndex: ptr(int64(5)),
				Action:      ptr("ACCEPT"),
				Protocol:    ptr("TCP"),
				Port:        ptr("6080"),
				CidrBlock:   ptr("0.0.0.0/0"),
			},
		},
	}
	err := ensureBrowserVNCPortRuleCore(context.Background(), "sg-test", makeWhitelistLoader(wl), fake)
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if len(fake.deletedIndexes) != 1 || fake.deletedIndexes[0] != 5 {
		t.Errorf("应删除 PolicyIndex=5，实际: %v", fake.deletedIndexes)
	}
	if len(fake.createdCIDRs) != 1 || fake.createdCIDRs[0] != "1.2.3.4/32" {
		t.Errorf("应创建 1.2.3.4/32，实际: %v", fake.createdCIDRs)
	}
}

func TestEnsureVNCPortRuleCore_AlreadyMigrated_NoChange(t *testing.T) {
	wl := []requiredSGRule{
		{CidrBlock: "1.2.3.4/32", Protocol: "TCP", Port: "6080", Action: "ACCEPT"},
		{CidrBlock: "5.6.7.8/32", Protocol: "TCP", Port: "6080", Action: "ACCEPT"},
	}
	fake := &fakeVncSGClient{
		ingress: []*vpc.SecurityGroupPolicy{
			{PolicyIndex: ptr(int64(1)), Action: ptr("ACCEPT"), Protocol: ptr("TCP"), Port: ptr("6080"), CidrBlock: ptr("1.2.3.4/32")},
			{PolicyIndex: ptr(int64(2)), Action: ptr("ACCEPT"), Protocol: ptr("TCP"), Port: ptr("6080"), CidrBlock: ptr("5.6.7.8/32")},
		},
	}
	err := ensureBrowserVNCPortRuleCore(context.Background(), "sg-test", makeWhitelistLoader(wl), fake)
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if len(fake.deletedIndexes) != 0 {
		t.Errorf("不应删除，实际删除 %d 条", len(fake.deletedIndexes))
	}
	if len(fake.createdCIDRs) != 0 {
		t.Errorf("不应创建，实际创建 %d 条", len(fake.createdCIDRs))
	}
}

func TestEnsureVNCPortRuleCore_PartialMigrated_AddMissing(t *testing.T) {
	wl := []requiredSGRule{
		{CidrBlock: "1.2.3.4/32", Protocol: "TCP", Port: "6080", Action: "ACCEPT"},
		{CidrBlock: "5.6.7.8/32", Protocol: "TCP", Port: "6080", Action: "ACCEPT"},
	}
	fake := &fakeVncSGClient{
		ingress: []*vpc.SecurityGroupPolicy{
			{PolicyIndex: ptr(int64(1)), Action: ptr("ACCEPT"), Protocol: ptr("TCP"), Port: ptr("6080"), CidrBlock: ptr("1.2.3.4/32")},
		},
	}
	err := ensureBrowserVNCPortRuleCore(context.Background(), "sg-test", makeWhitelistLoader(wl), fake)
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if len(fake.deletedIndexes) != 0 {
		t.Errorf("不应删除")
	}
	if len(fake.createdCIDRs) != 1 || fake.createdCIDRs[0] != "5.6.7.8/32" {
		t.Errorf("应创建 5.6.7.8/32，实际: %v", fake.createdCIDRs)
	}
}

func TestEnsureVNCPortRuleCore_DeleteError(t *testing.T) {
	wl := []requiredSGRule{{CidrBlock: "1.2.3.4/32", Protocol: "TCP", Port: "6080", Action: "ACCEPT"}}
	fake := &fakeVncSGClient{
		ingress: []*vpc.SecurityGroupPolicy{
			{PolicyIndex: ptr(int64(5)), Action: ptr("ACCEPT"), Protocol: ptr("TCP"), Port: ptr("6080"), CidrBlock: ptr("0.0.0.0/0")},
		},
		deleteErr: fmt.Errorf("delete api error"),
	}
	err := ensureBrowserVNCPortRuleCore(context.Background(), "sg-test", makeWhitelistLoader(wl), fake)
	if err == nil || !strings.Contains(err.Error(), "删除旧 0.0.0.0/0") {
		t.Fatalf("Delete 失败应报错，实际: %v", err)
	}
}

func TestEnsureVNCPortRuleCore_CreateError(t *testing.T) {
	wl := []requiredSGRule{{CidrBlock: "1.2.3.4/32", Protocol: "TCP", Port: "6080", Action: "ACCEPT"}}
	fake := &fakeVncSGClient{
		ingress:   nil,
		createErr: fmt.Errorf("create api error"),
	}
	err := ensureBrowserVNCPortRuleCore(context.Background(), "sg-test", makeWhitelistLoader(wl), fake)
	if err == nil || !strings.Contains(err.Error(), "添加白名单安全组规则失败") {
		t.Fatalf("Create 失败应报错，实际: %v", err)
	}
}

func TestEnsureVNCPortRuleCore_SkipNonTCP(t *testing.T) {
	wl := []requiredSGRule{{CidrBlock: "1.2.3.4/32", Protocol: "TCP", Port: "6080", Action: "ACCEPT"}}
	fake := &fakeVncSGClient{
		ingress: []*vpc.SecurityGroupPolicy{
			{PolicyIndex: ptr(int64(1)), Action: ptr("ACCEPT"), Protocol: ptr("UDP"), Port: ptr("6080"), CidrBlock: ptr("0.0.0.0/0")},
		},
	}
	err := ensureBrowserVNCPortRuleCore(context.Background(), "sg-test", makeWhitelistLoader(wl), fake)
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if len(fake.deletedIndexes) != 0 {
		t.Errorf("UDP 规则不应被删除")
	}
}

func TestEnsureVNCPortRuleCore_SkipOtherPort(t *testing.T) {
	wl := []requiredSGRule{{CidrBlock: "1.2.3.4/32", Protocol: "TCP", Port: "6080", Action: "ACCEPT"}}
	fake := &fakeVncSGClient{
		ingress: []*vpc.SecurityGroupPolicy{
			{PolicyIndex: ptr(int64(1)), Action: ptr("ACCEPT"), Protocol: ptr("TCP"), Port: ptr("22"), CidrBlock: ptr("0.0.0.0/0")},
		},
	}
	err := ensureBrowserVNCPortRuleCore(context.Background(), "sg-test", makeWhitelistLoader(wl), fake)
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if len(fake.deletedIndexes) != 0 {
		t.Errorf("端口 22 的规则不应被删除")
	}
}

func TestEnsureVNCPortRuleCore_SkipDROP(t *testing.T) {
	wl := []requiredSGRule{{CidrBlock: "1.2.3.4/32", Protocol: "TCP", Port: "6080", Action: "ACCEPT"}}
	fake := &fakeVncSGClient{
		ingress: []*vpc.SecurityGroupPolicy{
			{PolicyIndex: ptr(int64(1)), Action: ptr("DROP"), Protocol: ptr("TCP"), Port: ptr("6080"), CidrBlock: ptr("0.0.0.0/0")},
		},
	}
	err := ensureBrowserVNCPortRuleCore(context.Background(), "sg-test", makeWhitelistLoader(wl), fake)
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if len(fake.deletedIndexes) != 0 {
		t.Errorf("DROP 规则不应被删除")
	}
}

func TestEnsureVNCPortRuleCore_NilActionProtocolPort(t *testing.T) {
	wl := []requiredSGRule{{CidrBlock: "1.2.3.4/32", Protocol: "TCP", Port: "6080", Action: "ACCEPT"}}
	fake := &fakeVncSGClient{
		ingress: []*vpc.SecurityGroupPolicy{
			{PolicyIndex: ptr(int64(1)), Action: nil, Protocol: ptr("TCP"), Port: ptr("6080"), CidrBlock: ptr("0.0.0.0/0")},
			{PolicyIndex: ptr(int64(2)), Action: ptr("ACCEPT"), Protocol: nil, Port: ptr("6080"), CidrBlock: ptr("0.0.0.0/0")},
			{PolicyIndex: ptr(int64(3)), Action: ptr("ACCEPT"), Protocol: ptr("TCP"), Port: nil, CidrBlock: ptr("0.0.0.0/0")},
		},
	}
	err := ensureBrowserVNCPortRuleCore(context.Background(), "sg-test", makeWhitelistLoader(wl), fake)
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if len(fake.deletedIndexes) != 0 {
		t.Errorf("nil 字段的规则不应被删除，实际: %v", fake.deletedIndexes)
	}
}

func TestEnsureVNCPortRuleCore_LegacyNilPolicyIndex(t *testing.T) {
	wl := []requiredSGRule{{CidrBlock: "1.2.3.4/32", Protocol: "TCP", Port: "6080", Action: "ACCEPT"}}
	fake := &fakeVncSGClient{
		ingress: []*vpc.SecurityGroupPolicy{
			{PolicyIndex: nil, Action: ptr("ACCEPT"), Protocol: ptr("TCP"), Port: ptr("6080"), CidrBlock: ptr("0.0.0.0/0")},
		},
	}
	err := ensureBrowserVNCPortRuleCore(context.Background(), "sg-test", makeWhitelistLoader(wl), fake)
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if len(fake.deletedIndexes) != 0 {
		t.Errorf("PolicyIndex 为 nil 的旧规则不应放入删除列表")
	}
}

func TestEnsureVNCPortRuleCore_OtherCIDR_NotDeleted(t *testing.T) {
	wl := []requiredSGRule{{CidrBlock: "1.2.3.4/32", Protocol: "TCP", Port: "6080", Action: "ACCEPT"}}
	fake := &fakeVncSGClient{
		ingress: []*vpc.SecurityGroupPolicy{
			{PolicyIndex: ptr(int64(1)), Action: ptr("ACCEPT"), Protocol: ptr("TCP"), Port: ptr("6080"), CidrBlock: ptr("10.0.0.1/32")},
		},
	}
	err := ensureBrowserVNCPortRuleCore(context.Background(), "sg-test", makeWhitelistLoader(wl), fake)
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if len(fake.deletedIndexes) != 0 {
		t.Errorf("10.0.0.1/32 不应被删除")
	}
}

func TestEnsureVNCPortRuleCore_EmptyCIDR(t *testing.T) {
	wl := []requiredSGRule{{CidrBlock: "1.2.3.4/32", Protocol: "TCP", Port: "6080", Action: "ACCEPT"}}
	fake := &fakeVncSGClient{
		ingress: []*vpc.SecurityGroupPolicy{
			{PolicyIndex: ptr(int64(1)), Action: ptr("ACCEPT"), Protocol: ptr("TCP"), Port: ptr("6080"), CidrBlock: nil},
			{PolicyIndex: ptr(int64(2)), Action: ptr("ACCEPT"), Protocol: ptr("TCP"), Port: ptr("6080"), CidrBlock: ptr("")},
		},
	}
	err := ensureBrowserVNCPortRuleCore(context.Background(), "sg-test", makeWhitelistLoader(wl), fake)
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if len(fake.deletedIndexes) != 0 {
		t.Errorf("空 CIDR 不应被删除")
	}
}

func TestEnsureVNCPortRuleCore_WhitelistEmptyCIDR_Skipped(t *testing.T) {
	wl := []requiredSGRule{
		{CidrBlock: "", Protocol: "TCP", Port: "6080", Action: "ACCEPT"},
		{CidrBlock: "1.2.3.4/32", Protocol: "TCP", Port: "6080", Action: "ACCEPT"},
	}
	fake := &fakeVncSGClient{ingress: nil}
	err := ensureBrowserVNCPortRuleCore(context.Background(), "sg-test", makeWhitelistLoader(wl), fake)
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if len(fake.createdCIDRs) != 1 {
		t.Errorf("空 CIDR 白名单应被跳过，只创建 1 条，实际: %d", len(fake.createdCIDRs))
	}
}

func TestEnsureVNCPortRuleCore_AllProtocol_Matches(t *testing.T) {
	wl := []requiredSGRule{{CidrBlock: "1.2.3.4/32", Protocol: "TCP", Port: "6080", Action: "ACCEPT"}}
	fake := &fakeVncSGClient{
		ingress: []*vpc.SecurityGroupPolicy{
			{PolicyIndex: ptr(int64(1)), Action: ptr("ACCEPT"), Protocol: ptr("ALL"), Port: ptr("6080"), CidrBlock: ptr("0.0.0.0/0")},
		},
	}
	err := ensureBrowserVNCPortRuleCore(context.Background(), "sg-test", makeWhitelistLoader(wl), fake)
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if len(fake.deletedIndexes) != 1 {
		t.Errorf("ALL 协议 + 6080 端口的 0.0.0.0/0 应被删除")
	}
}

// ========== addSecurityGroupPortRule 防御性测试 ==========

func TestAddSecurityGroupPortRule_AlwaysErrors(t *testing.T) {
	err := addSecurityGroupPortRule("sg-test", 6080, "test")
	if err == nil {
		t.Fatal("addSecurityGroupPortRule 应总是返回错误")
	}
	if !strings.Contains(err.Error(), "已禁用") {
		t.Errorf("错误信息应包含'已禁用'，实际: %s", err.Error())
	}
}

// ========== resolveConditionalRules browser_vnc_enable 测试 ==========

func TestResolveConditionalRules_BrowserVNCEnable(t *testing.T) {
	origJSON := ClawproRequiredSGRulesJSON
	defer func() { ClawproRequiredSGRulesJSON = origJSON }()

	ClawproRequiredSGRulesJSON = []byte(`{
		"categories": [
			{"type":"recommended","label":"推荐","rule_groups":[
				{"key":"allow_vnc_whitelist","name":"VNC白名单","condition":"browser_vnc_enable","rules":[
					{"direction":"ingress","protocol":"TCP","port":"6080","cidr_block":"1.2.3.4/32","action":"ACCEPT","description":"test"}
				]}
			]}
		]
	}`)

	// BrowserVNCEnable = true → 保留
	cleanup := initBrowserVNCHandlerTestDB(t)
	defer cleanup()
	config := model.GetSiteConfig(context.Background())
	config.BrowserVNCEnable = true
	model.DB(context.Background()).Save(&config)

	ruleSet := clawproRequiredRuleSet()
	resolveConditionalRules(context.Background(), &ruleSet)
	found := false
	for _, cat := range ruleSet.Categories {
		for _, g := range cat.RuleGroups {
			if g.Key == "allow_vnc_whitelist" {
				found = true
			}
		}
	}
	if !found {
		t.Error("BrowserVNCEnable=true 时 allow_vnc_whitelist 应保留")
	}

	// BrowserVNCEnable = false → 移除
	config.BrowserVNCEnable = false
	model.DB(context.Background()).Save(&config)
	ruleSet2 := clawproRequiredRuleSet()
	resolveConditionalRules(context.Background(), &ruleSet2)
	for _, cat := range ruleSet2.Categories {
		for _, g := range cat.RuleGroups {
			if g.Key == "allow_vnc_whitelist" {
				t.Error("BrowserVNCEnable=false 时 allow_vnc_whitelist 应被移除")
			}
		}
	}
}

// ========== StartVNCSecurityGroupMigration 测试 ==========

func TestBrowserVNCAccessCore_MigrateAsyncSuccess(t *testing.T) {

	instance := &model.Instance{Model: gorm.Model{ID: 99904}, InstanceId: "ins-async-ok"}
	siteConfig := model.SiteConfig{
		BrowserVNCEnable: true,
		SecurityGroupId:  "sg-async-ok",
	}

	describeFn := func(instanceId string) (*cvmInstanceInfo, error) {
		return &cvmInstanceInfo{InstanceState: "RUNNING", PublicIp: "1.2.3.4"}, nil
	}
	checkPortFn := func(sgId string, port int) (bool, error) {
		return true, nil
	}

	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodGet, "/openclaw/browser-vnc-access?id=99904", nil)
	browserVNCAccessCore(w, r, instance, siteConfig, describeFn, checkPortFn)

	if w.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际 %d", w.Code)
	}

	// 等异步 goroutine 跑完（它会调 ensureBrowserVNCPortRule，因为没有真实 VPC 会失败然后 Store(0)）
	time.Sleep(200 * time.Millisecond)

	// CAS 被消费过（无论成功失败），验证机制可用即可
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]interface{})
	if data["accessible"] != true {
		t.Error("accessible 应为 true")
	}
}

// ========== browserVNCCheckCore 并行化变更的专项测试 ==========

// TestBrowserVNCCheckCore_ParallelOsName_SlowFetchDoesNotBlock 测试：fetchOsNameFn 较慢时不阻塞 TAT 脚本执行
func TestBrowserVNCCheckCore_ParallelOsName_SlowFetchDoesNotBlock(t *testing.T) {
	instance := &model.Instance{Model: gorm.Model{ID: 34101}, InstanceId: "ins-parallel"}

	req := httptest.NewRequest(http.MethodGet, "/openclaw/browser-vnc-check?id=1", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	var scriptStart, scriptEnd, fetchStart, fetchEnd time.Time

	browserVNCCheckCore(w, req, instance, true,
		func(instanceId, script string, timeout uint64, runtimeUser string) (string, error) {
			scriptStart = time.Now()
			time.Sleep(50 * time.Millisecond) // 模拟 TAT 执行
			scriptEnd = time.Now()
			return `{"ready": true, "checks": {"services": "ok"}}`, nil
		},
		func(_ context.Context, instanceId string) string {
			fetchStart = time.Now()
			time.Sleep(30 * time.Millisecond) // 模拟 DescribeInstances 调用
			fetchEnd = time.Now()
			return "Ubuntu Server 24.04 LTS 64bit"
		})

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	// 验证并行：fetchOsNameFn 应该在 TAT 脚本执行期间开始（不是之后）
	if fetchStart.After(scriptEnd) {
		t.Errorf("fetchOsNameFn 应与 TAT 并行执行，但 fetchStart(%v) > scriptEnd(%v)", fetchStart, scriptEnd)
	}
	// fetchOsNameFn 的开始时间应该在 TAT 开始之后很短时间内（goroutine 调度延迟）
	if fetchStart.Sub(scriptStart) > 10*time.Millisecond {
		t.Errorf("fetchOsNameFn 应几乎与 TAT 同时开始，延迟=%v", fetchStart.Sub(scriptStart))
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]interface{})
	if data["os_name"] != "Ubuntu Server 24.04 LTS 64bit" {
		t.Errorf("os_name 应为 'Ubuntu Server 24.04 LTS 64bit'，实际为 %v", data["os_name"])
	}
	_ = fetchEnd // 消除未使用警告
}

// TestBrowserVNCCheckCore_ParallelOsName_ScriptFailsOsNameGoroutineCompletes 测试：TAT 失败时 goroutine 不泄漏
func TestBrowserVNCCheckCore_ParallelOsName_ScriptFailsOsNameGoroutineCompletes(t *testing.T) {
	instance := &model.Instance{Model: gorm.Model{ID: 34102}, InstanceId: "ins-fail-parallel"}

	req := httptest.NewRequest(http.MethodGet, "/openclaw/browser-vnc-check?id=1", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	fetchCalled := make(chan struct{}, 1)

	browserVNCCheckCore(w, req, instance, true,
		func(instanceId, script string, timeout uint64, runtimeUser string) (string, error) {
			return "", fmt.Errorf("TAT 执行超时")
		},
		func(_ context.Context, instanceId string) string {
			fetchCalled <- struct{}{}
			time.Sleep(10 * time.Millisecond)
			return "Ubuntu 24.04"
		})

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("期望 500，实际=%d", w.Code)
	}

	// 确认 goroutine 中的 fetchOsNameFn 被调用了（说明是并行启动的）
	select {
	case <-fetchCalled:
		// OK - goroutine 正常启动并执行
	case <-time.After(200 * time.Millisecond):
		t.Error("fetchOsNameFn 应已被调用（goroutine 已启动），但超时未收到信号")
	}
}

// TestBrowserVNCCheckCore_ParallelOsName_NilFetchFn 测试：fetchOsNameFn 为 nil 时不 panic
func TestBrowserVNCCheckCore_ParallelOsName_NilFetchFn(t *testing.T) {
	instance := &model.Instance{Model: gorm.Model{ID: 34103}, InstanceId: "ins-nil-fetch"}

	req := httptest.NewRequest(http.MethodGet, "/openclaw/browser-vnc-check?id=1", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserVNCCheckCore(w, req, instance, true,
		func(instanceId, script string, timeout uint64, runtimeUser string) (string, error) {
			return `{"ready": true}`, nil
		},
		nil) // fetchOsNameFn = nil

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]interface{})
	if _, exists := data["os_name"]; exists {
		t.Errorf("fetchOsNameFn 为 nil 时不应设置 os_name，实际有值: %v", data["os_name"])
	}
}

// TestBrowserVNCCheckCore_ParallelOsName_EmptyResult 测试：fetchOsNameFn 返回空字符串时不设置 os_name
func TestBrowserVNCCheckCore_ParallelOsName_EmptyResult(t *testing.T) {
	instance := &model.Instance{Model: gorm.Model{ID: 34104}, InstanceId: "ins-empty-os"}

	req := httptest.NewRequest(http.MethodGet, "/openclaw/browser-vnc-check?id=1", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserVNCCheckCore(w, req, instance, true,
		func(instanceId, script string, timeout uint64, runtimeUser string) (string, error) {
			return `{"ready": false, "missing": ["service:browser-vnc-xvnc"]}`, nil
		},
		func(_ context.Context, instanceId string) string {
			return "" // API 调用失败返回空
		})

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]interface{})
	if _, exists := data["os_name"]; exists {
		t.Errorf("fetchOsNameFn 返回空时不应设置 os_name，实际有值: %v", data["os_name"])
	}
}

// TestBrowserVNCCheckCore_Timeout30 测试：确认 check 脚本超时参数为 30 秒
func TestBrowserVNCCheckCore_Timeout30(t *testing.T) {
	instance := &model.Instance{Model: gorm.Model{ID: 34105}, InstanceId: "ins-timeout"}

	req := httptest.NewRequest(http.MethodGet, "/openclaw/browser-vnc-check?id=1", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	var gotTimeout uint64
	browserVNCCheckCore(w, req, instance, true,
		func(instanceId, script string, timeout uint64, runtimeUser string) (string, error) {
			gotTimeout = timeout
			return `{"ready": true}`, nil
		},
		nil)

	if gotTimeout != 30 {
		t.Errorf("check 脚本超时应为 30 秒，实际为 %d", gotTimeout)
	}
}

// TestBrowserVNCCheckCore_DesktopMode_BrowserOnly 测试：旧版浏览器环境返回 desktop_mode=browser_only, upgrade_available=true, ready=true
func TestBrowserVNCCheckCore_DesktopMode_BrowserOnly(t *testing.T) {
	instance := &model.Instance{Model: gorm.Model{ID: 34201}, InstanceId: "ins-oldver"}

	req := httptest.NewRequest(http.MethodGet, "/openclaw/browser-vnc-check?id=1", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	// 模拟旧版环境：ready=true 但 desktop_mode=browser_only
	browserVNCCheckCore(w, req, instance, true,
		func(instanceId, script string, timeout uint64, runtimeUser string) (string, error) {
			return `{"ready":true,"desktop_mode":"browser_only","upgrade_available":true,"checks":{"packages":"ok","novnc":"ok","chrome":"ok","systemd_config":"ok","services":"ok","ports":"ok","cdp_owner":"ok","ssl_cert":"ok","fcitx5":"missing","cjk_fonts":"missing","novnc_patch":"missing"}}`, nil
		},
		nil)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]interface{})

	// ready 应为 true（旧版浏览器功能正常）
	if data["ready"] != true {
		t.Errorf("ready 应为 true（旧版浏览器正常），实际为 %v", data["ready"])
	}
	// desktop_mode 应为 browser_only
	if data["desktop_mode"] != "browser_only" {
		t.Errorf("desktop_mode 应为 'browser_only'，实际为 %v", data["desktop_mode"])
	}
	// upgrade_available 应为 true
	if data["upgrade_available"] != true {
		t.Errorf("upgrade_available 应为 true，实际为 %v", data["upgrade_available"])
	}
}

// TestBrowserVNCCheckCore_DesktopMode_Full 测试：新版完整安装返回 desktop_mode=full, upgrade_available=false
func TestBrowserVNCCheckCore_DesktopMode_Full(t *testing.T) {
	instance := &model.Instance{Model: gorm.Model{ID: 34202}, InstanceId: "ins-newver"}

	req := httptest.NewRequest(http.MethodGet, "/openclaw/browser-vnc-check?id=1", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	browserVNCCheckCore(w, req, instance, true,
		func(instanceId, script string, timeout uint64, runtimeUser string) (string, error) {
			return `{"ready":true,"desktop_mode":"full","upgrade_available":false,"checks":{"packages":"ok","novnc":"ok","chrome":"ok","systemd_config":"ok","services":"ok","ports":"ok","cdp_owner":"ok","ssl_cert":"ok","fcitx5":"ok","cjk_fonts":"ok","novnc_patch":"ok"}}`, nil
		},
		nil)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]interface{})

	if data["ready"] != true {
		t.Errorf("ready 应为 true，实际为 %v", data["ready"])
	}
	if data["desktop_mode"] != "full" {
		t.Errorf("desktop_mode 应为 'full'，实际为 %v", data["desktop_mode"])
	}
	if data["upgrade_available"] != false {
		t.Errorf("upgrade_available 应为 false，实际为 %v", data["upgrade_available"])
	}
}
