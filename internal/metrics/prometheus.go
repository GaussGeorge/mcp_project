package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// 定义全局指标变量
var (
	// 1. 计数器：记录请求总量，标签区分状态(accepted/rejected)
	RequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rajomon_requests_total",
			Help: "Total number of requests processed by the gateway",
		},
		[]string{"status","handler"},// labels: status=accepted/rejected, handler=mcp/context
	)

	// 2. 直方图：记录请求延迟分布
	RequestLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:  "rajomon_request_duration_seconds",
			Help:  "Request latency distributions",
			Buckets: prometheus.DefBuckets,// 使用默认桶 [0.005, ..., 10]
		},
		[]string{"handler"},
	)

	// 3. 直方图：记录 Token 消耗分布
	TokenUsage = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "rajomon_token_usage",
			Help:    "Token usage distributions per request",
			Buckets: []float64{10, 50, 100, 200, 500, 1000, 2000},
		},
		[]string{"handler"},
	)

	// 4. 仪表盘：当前服务的价格 (这是 Rajomon 的核心)
	CurrentPrice = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "rajomon_current_price",
			Help: "Current dynamic price of the service",
		},
	)

	// 5. 仪表盘：当前综合成本 (帮助调试 EWMA 算法)
	CompositeCost = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "rajomon_composite_cost",
			Help: "Current calculated composite cost (latency + tokens)",
		},
	)
)

// Init 注册所有指标
func Init() {
	prometheus.MustRegister(RequestsTotal)
	prometheus.MustRegister(RequestLatency)
	prometheus.MustRegister(TokenUsage)
	prometheus.MustRegister(CurrentPrice)
	prometheus.MustRegister(CompositeCost)
}