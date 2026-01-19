package model

// SSEEvent 定义 SSE 消息结构
type SSEEvent struct {
	Event string      `json:"event,omitempty"` // 事件类型 (如 "message", "usage")
	Data  interface{} `json:"data"`            // 数据负载
}

// MockContent 模拟 AI 返回的文本片段
type MockContent struct {
	Content string `json:"content"`
}

// MockUsage 模拟 Token 消耗统计 (关键数据)
type MockUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}