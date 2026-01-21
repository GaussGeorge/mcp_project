package main

import (
	"fmt"
	"log"
	"net/http"
	"rajomon-gateway/internal/controller"
	"rajomon-gateway/internal/handler"
	"rajomon-gateway/internal/metrics"
	"rajomon-gateway/internal/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// [新增] 0. 初始化 Metrics
	metrics.Init()

	// 1. 创建一个独立的路由器 (Mux)
	// 这是一个"干净"的路由表，不会被第三方库污染
	rajomonCtrl := controller.NewController()
	fmt.Println("🧠 Rajomon 控制器已启动 (EWMA 模式)")
	mux := http.NewServeMux()

	// 2. 注册路由
	// 场景 A: 测试 Context 超时控制
	contextBizHandler := http.HandlerFunc(handler.ContextHandler)
	// 现在的调用链：Request -> Middleware(写Price) -> ContextHandler(写Body)
	wrappedContextHandler := middleware.RajomonMiddleware(rajomonCtrl, contextBizHandler)
	mux.Handle("/context", wrappedContextHandler)

	// --- 注册 MCP SSE 接口 ---
    // 1. 创建 Handler
	mcpHandler := http.HandlerFunc(handler.HandleMCP)
	// 2. 包裹 Rajomon 中间件 (目前中间件还看不懂 SSE，下一步我们就要改造中间件)
	wrappedMCPHandler := middleware.RajomonMiddleware(rajomonCtrl,mcpHandler)
	// 3. 注册路由 (通常 LLM 风格是 /v1/chat/completions，这里演示简单用 /mcp/chat)
	mux.Handle("/mcp/chat", wrappedMCPHandler)

	// --- 🆕 新增: 注册 Prometheus Metrics 接口 ---
	// Prometheus 会来这里拉取数据
	mux.Handle("/metrics",promhttp.Handler())
	fmt.Println("👀 Prometheus Metrics 已暴露在 /metrics")


	// 场景 B: 测试 Rajomon 价格反馈 (原 fankui_handler)
	// myHandler := &handler.MyGovernanceHandler{Price: 10,}
	// mux.Handle("/price", myHandler)

	// 场景 C: 带有中间件的业务逻辑处理器
	// // 步骤 1: 实例化“内层”业务逻辑
	// bizHandler := &handler.RealBizHandler{}
	// // 步骤 2: 实例化“外层”中间件，并把内层塞进去
	// // 这就是“俄罗斯套娃”的关键一步
	// wrappedHandler := &handler.RajomonMiddleware{
	// 	Next: bizHandler,
	// }
	// // 步骤 3: 注册路由
	// // 注意：我们要把 wrappedHandler (最外层) 给 Server
	// // 如果你只给 bizHandler，那 Rajomon 的逻辑就不会执行
	// http.Handle("/mcp", wrappedHandler)

	// Handle 是面向**接口（Interface）**的，适合复杂的、需要状态的场景。
	// 参数: 接收一个实现了 http.Handler 接口的对象。
	// 接口定义: 该对象必须实现 ServeHTTP(w http.ResponseWriter, r *http.Request) 方法。
	// 适用场景: 当你的处理器（Handler）需要维护状态（例如数据库连接池、配置信息、缓存）时，通常会定义一个结构体（Struct），让它实现 http.Handler 接口，然后用 Handle 注册。

	// HandleFunc 是面向**函数（Function）**的，适合简单的、无状态的逻辑。
	// 参数: 接收一个具有特定签名的函数：func(w http.ResponseWriter, r *http.Request)。
	// 适用场景: 当你的逻辑非常简单，不需要维护额外的状态，或者你只是想快速写一个 API 时，使用函数会更简洁。

	// 3. 启动服务
	addr := ":8080"
	fmt.Printf("🚀 rajomon 服务端已启动，监听 %s\n", addr)
	// 这里传入 mux，而不是 nil
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	if err := server.ListenAndServe(); err != nil {
		log.Fatal("启动失败", err)
	}

}
