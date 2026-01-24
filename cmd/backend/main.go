package main

import (
	"fmt"
	"log"
	"net/http"
	"rajomon-gateway/internal/handler"
	"time"
)

func main() {
	// 1. 注册路由 (只负责处理 MCP 业务，不负责治理)
	http.HandleFunc("/mcp/chat", handler.HandleMCP)

	// 2. 启动服务 (监听 9001，Docker 内部端口)
	// 注意：在 Docker 里我们通常让它监听 :8080，通过端口映射区分
	// 但为了本地也能跑，这里硬编码或者从环境变量读更好。
	// 为了简单，我们让 Backend 在容器里监听 :8080
	addr := ":8080"
	fmt.Printf("🤖 Mock LLM Backend 已启动，监听 %s\n", addr)

	// 增加一个简单的健康检查接口
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	server := &http.Server{
		Addr:         addr,
		ReadTimeout:  30 * time.Second, // 防止长连接断开
		WriteTimeout: 30 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
