package handler

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

func ContextHandler(w http.ResponseWriter, r *http.Request) {
	tokenStr := r.Header.Get("Token")

	if tokenStr == "" {
		http.Error(w, "No Token", http.StatusForbidden)
		return
	}
	fmt.Printf("[治理层] 收到请求，携带 Token: %s\n", tokenStr)

	ctx := r.Context()

	fmt.Println("[业务层] 开始处理繁重的 MCP 任务...")

	delay := time.Duration(rand.Intn(100)+150) * time.Millisecond
	select {
	case <-time.After(delay):
		fmt.Println("[业务层] MCP 任务处理完成")
		w.Write([]byte("MCP Task Success"))

	case <-ctx.Done():
		err := ctx.Err()
		fmt.Printf("[治理层] 任务被迫中断！原因: %v\n", err)
		http.Error(w, err.Error(), http.StatusGatewayTimeout)
	}
}
