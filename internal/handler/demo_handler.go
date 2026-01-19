package handler

import (
	"fmt"
	"net/http"
	"time"
)

// 1. 最终的业务 Handler (最内层的娃娃)
type RealBizHandler struct{}

func (h *RealBizHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println(">>> 3. 执行真正的 MCP 业务逻辑")
	w.Write([]byte("MCP Result"))
}

// 2. Rajomon 中间件 (外层的娃娃)
// 它自己是一个 Handler，同时它内部还包含了另一个 Handler
type RajomonMiddleware struct {
	Next http.Handler // 下一个 Handler
}

func (m *RajomonMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// --- A. 事前控制 (拦截点) ---
	fmt.Println(">>> 1. Rajomon: 检查 Token 余额...")
	token := r.Header.Get("Token")
	if token == "" {
		http.Error(w, "No Token", 403)
		return
	}

	start := time.Now()

	// --- B. 调用下一个 Handler ---
	fmt.Println(">>> 2. Rajomon: 调用下一个 Handler")
	m.Next.ServeHTTP(w, r)

	// --- C. 事后处理 (拦截点) ---
	latency := time.Since(start)
	fmt.Printf(">>> 4. Rajomon: 请求完成，本次请求处理时长: %v\n", latency)
}
