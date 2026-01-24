package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"rajomon-gateway/internal/controller"
	"rajomon-gateway/internal/handler"
	"rajomon-gateway/internal/metrics"
	"rajomon-gateway/internal/middleware"
	"strings"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// SimpleLoadBalancer 简单的轮询负载均衡器
type SimpleLoadBalancer struct {
	backends []*url.URL
	current  uint64
}

func NewLoadBalancer(targets []string) *SimpleLoadBalancer {
	var backends []*url.URL
	for _, target := range targets {
		u, err := url.Parse(target)
		if err != nil {
			log.Fatalf("后端地址解析失败: %s", err)
		}
		backends = append(backends, u)
	}
	return &SimpleLoadBalancer{backends: backends}
}

// ServeHTTP 实现反向代理转发
func (lb *SimpleLoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if len(lb.backends) == 0 {
		http.Error(w, "No backend available", http.StatusServiceUnavailable)
		return
	}

	// 1. 轮询算法选择后端
	idx := atomic.AddUint64(&lb.current, 1) % uint64(len(lb.backends))
	target := lb.backends[idx]

	// 2. 创建反向代理
	proxy := httputil.NewSingleHostReverseProxy(target)

	// 修改请求头，确保 Host 正确
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
		// 可以在这里加一个 Header 标识经过了网关
		req.Header.Set("X-Forwarded-By", "Rajomon-Gateway")
	}

	// 自定义错误处理 (比如后端挂了)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		fmt.Printf("❌ [LB] 转发失败 -> %s: %v\n", target.Host, err)
		w.WriteHeader(http.StatusBadGateway)
	}

	fmt.Printf("🔀 [LB] 转发请求 -> %s\n", target.Host)
	proxy.ServeHTTP(w, r)
}

func main() {
	// [新增] 0. 初始化 Metrics
	metrics.Init()

	// 1. 从环境变量获取后端列表
	// 格式: "http://backend-1:8080,http://backend-2:8080"
	backendEnv := os.Getenv("BACKEND_HOSTS")
	if backendEnv == "" {
		// 默认值，方便本地非Docker调试（假设本地起了backend在9001）
		backendEnv = "http://localhost:9001"
	}
	targets := strings.Split(backendEnv, ",")

	// 2. 初始化负载均衡器
	lb := NewLoadBalancer(targets)
	fmt.Printf("⚖️ 负载均衡器已就绪，后端节点: %v\n", targets)

	// 3. 初始化控制器
	rajomonCtrl := controller.NewController()
	mux := http.NewServeMux()

	// 4. 组装核心链路: Client -> Rajomon Middleware -> LoadBalancer -> Backend
	// 注意：我们把 lb 当作 next handler 传给 Middleware
	wrappedLB := middleware.RajomonMiddleware(rajomonCtrl, lb)

	// 注册路由
	mux.Handle("/mcp/chat", wrappedLB)

	// 保留 context 测试接口
	contextBizHandler := http.HandlerFunc(handler.ContextHandler)
	mux.Handle("/context", middleware.RajomonMiddleware(rajomonCtrl, contextBizHandler))

	// --- 🆕 新增: 注册 Prometheus Metrics 接口 ---
	// Prometheus 会来这里拉取数据
	mux.Handle("/metrics", promhttp.Handler())
	fmt.Println("👀 Prometheus Metrics 已暴露在 /metrics")

	// 5. 启动服务
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
