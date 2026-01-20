// 主要变化点：

// 设置 Header Content-Type: text/event-stream。
// 使用 http.Flusher 强制将缓冲区数据推送到客户端。
// 循环发送“碎片数据”，最后发送“Usage数据”。

package handler

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"rajomon-gateway/internal/model"
	"time"
)

// HandleMCP 模拟 MCP 协议的流式响应
func HandleMCP(w http.ResponseWriter, r *http.Request) {
	// 1. 设置 SSE 必要的 Headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 2. 获取 Flusher (这是实现流式输出的关键)
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	fmt.Println("[Mock LLM] 开始流式生成内容...")

	// 3. 模拟分段输出内容 (Chunks)
	chunks := []string{"你好，", "这是一个", "基于", "Rajomon", "治理的", "模拟", "AI回复。"}

	// [新增] 预计算 Token (为了给中间件透传数据)
	// 在真实场景中，可能是在流结束时通过 HTTP Trailers 发送，
	// 但为了简化，我们在 Mock 时预先算好放在 Header 里
	totalPrompt := 20
	totalCompletion := len(chunks) * 2 // 假设每个 chunk 2 token
	totalUsage := totalPrompt + totalCompletion

	// 🔥 [关键] 写入侧信道 Header，中间件会读这个！
	w.Header().Set("X-Token-Usage", fmt.Sprintf("%d", totalUsage))

	fmt.Println("[Mock LLM] 开始流式生成内容...")

	for _, text := range chunks {
		// 模拟思考延迟 (制造抖动，方便后续测试 Rajomon 的 EWMA 算法)
		delay := time.Duration(rand.Intn(100)+50) * time.Millisecond
		time.Sleep(delay)

		// 构造数据
		respData := model.MockContent{Content: text}
		sendSSE(w, "message", respData)

		// 🚀 立即推送给客户端
		flusher.Flush()
	}

	// 4. 发送最终的 Token Usage (这是 Rajomon 定价的关键依据)
	usageData := model.MockUsage{
		PromptTokens:     totalPrompt,
		CompletionTokens: totalCompletion,
		TotalTokens:      totalUsage,
	}
	sendSSE(w, "usage", usageData)
	flusher.Flush()

	fmt.Printf("[Mock LLM] 响应结束. 总消耗 Tokens: %d\n", usageData.TotalTokens)
}

// 辅助函数：封装 SSE 格式 (data: {...}\n\n)
func sendSSE(w http.ResponseWriter, eventType string, data interface{}) {
	jsonData, _ := json.Marshal(data)
	if eventType != "" {
		fmt.Fprintf(w, "event: %s\n", eventType)
	}
	fmt.Fprintf(w, "data: %s\n\n", jsonData)
}
