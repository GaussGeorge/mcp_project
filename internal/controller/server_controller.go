package controller

import (
	"fmt"
	"rajomon-gateway/internal/metrics"
	"math"
	"sync"
	"time"
)

type RajomonController struct {
	mu           sync.RWMutex

	// --- 1. 接口粒度控制 (Interface Granularity) ---
	// 使用 Map 存储每个接口/模型的状态
	// Key 通常是 URL Path (e.g., "/mcp/chat") 或 模型名称
	Prices 	map[string]int		// 各接口当前价格
	ewmaLatency map[string]float64 // 各接口平均延迟 (ms)
	ewmaTokens  map[string]float64 // 各接口平均 Token 消耗 (个)

	// --- 权重与阈值配置 ---
	alpha         float64 // 平滑因子
	latencyWeight float64 // 延迟在定价中的权重（比如0.5）
	tokenWeight   float64 // Token 消耗在定价中的权重（比如0.5）
	baseThreshold float64 // 综合成本阈值

	// --- 2. 比例价格更新参数 ---
	// 价格敏感度：每超出阈值多少分，价格 +1
	// 例如：阈值 200，敏感度 50。如果 Cost=350 `(超150)，则价格涨 int(150/50) = 3`
	priceStepUnit float64
}

func NewController() *RajomonController {
	return &RajomonController{
		Prices:		make(map[string]int),
		ewmaLatency: make(map[string]float64),
		ewmaTokens: make(map[string]float64),

		alpha:         0.2, // 权重：新数据占 20%，历史数据占 80%
		latencyWeight: 0.5, // 延迟权重 50%
		tokenWeight:   0.5, // Token 权重 50%
		baseThreshold: 200, // 综合分超过 200 就涨价
		priceStepUnit: 50.0, // 灵敏度：每超 50 分涨 1 块钱
	}
}

// GetPrice 获取指定接口的当前价格 (支持惰性初始化)
func (c *RajomonController) GetPrice(key string) int {
	c.mu.Lock() // 使用写锁，因为可能需要初始化 Map
	defer c.mu.Unlock()

	// 如果该接口是第一次访问，初始化默认价格
	if _, exists := c.Prices[key]; !exists {
		c.Prices[key] = 5 // 默认初始价格
		// 初始化 EWMA 状态，防止计算时取到 0 导致波动
		c.ewmaLatency[key] = 0
		c.ewmaTokens[key] = 0
	}
	return c.Prices[key]
}




// RecordLatency 同时接收延迟和Token消耗
func (c *RajomonController) RecordLatency(key string, latency time.Duration, tokenCount int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 1. 数据准备
	latencyMs := float64(latency.Milliseconds())
	tokens := float64(tokenCount)

	// 2. EWMA 更新 (针对特定 Key 更新对应的平均值)
	if val, exists := c.ewmaLatency[key]; !exists || val == 0 {
		c.ewmaLatency[key] = latencyMs
	} else {
		c.ewmaLatency[key] = c.alpha*latencyMs + (1-c.alpha)*val
	}

	if val, exists := c.ewmaTokens[key]; !exists || val == 0 {
		c.ewmaTokens[key] = tokens
	} else {
		c.ewmaTokens[key] = c.alpha*tokens + (1-c.alpha)*val
	}


	// 3. 计算综合成本
	currentLat := c.ewmaLatency[key]
	currentTok := c.ewmaTokens[key]
	compositeCost := (c.latencyWeight * currentLat) + (c.tokenWeight * currentTok)

	// [埋点] 记录该接口的成本 (Label=key)
	metrics.CompositeCost.WithLabelValues(key).Set(compositeCost)

	// --- 4. 比例价格更新 (Proportional Price Updates) ---
	currentPrice := c.Prices[key]

	// 5. 动态定价
	if compositeCost > c.baseThreshold {
		// 计算超出的部分
		excess := compositeCost - c.baseThreshold
		// 计算涨价步长：步长 = (超出量 / 单位量) + 基础步长
		// 必须保证至少涨 1 块
		step := int(math.Floor(excess / c.priceStepUnit))
		if step < 1 {
			step = 1
		}
		
		// 安全限制：防止单次涨幅过大导致震荡 (可选)
		if step > 10 {
			step = 10
		}
		c.Prices[key] += step
		fmt.Printf("📈 [Controller][%s] 成本过高(Cost:%.0f, Excess:%.0f) -> 猛涨 %d (现价:%d)\n",
			key, compositeCost, excess, step, c.Prices[key])
	} else if compositeCost < c.baseThreshold/2 && currentPrice > 1 {
		// 降价逻辑通常保持平缓（线性回落），避免系统震荡
		// 也可以按比例降价，但为了系统稳定性，推荐线性降价
		c.Prices[key]--
		fmt.Printf("📉 [Controller][%s] 成本回落(Cost:%.0f) -> 降价至 %d\n", key, compositeCost, c.Prices[key])
	}

	// [埋点] 记录最新价格
	metrics.CurrentPrice.WithLabelValues(key).Set(float64(c.Prices[key]))
}


// Lock()（写锁/互斥锁）：
// 排他性：一旦某个 Goroutine 持有了写锁，其他任何 Goroutine（无论是想读还是想写）都必须等待，直到该锁被释放。
// 用途：用于修改数据（写操作）。

// RLock()（读锁）：
// 共享性：多个 Goroutine 可以同时持有读锁。只要没有 Goroutine 持有写锁，多个读操作可以并行执行。
// 用途：用于读取数据（读操作）。
