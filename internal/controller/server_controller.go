package controller

import (
	"fmt"
	"rajomon-gateway/internal/metrics"
	"sync"
	"time"
)

type RajomonController struct {
	mu           sync.RWMutex
	CurrentPrice int

	// --- 多维指标 ---
	ewmaLatency float64 // 平均延迟 (ms)
	ewmaTokens  float64 // 平均 Token 消耗 (个)

	// --- 权重配置 ---
	alpha         float64 // 平滑因子
	latencyWeight float64 // 延迟在定价中的权重（比如0.5）
	tokenWeight   float64 // Token 消耗在定价中的权重（比如0.5）
	baseThreshold float64 // 综合成本阈值
}

func NewController() *RajomonController {
	return &RajomonController{
		CurrentPrice:  5,   // 初始价格
		ewmaLatency:   0,   // 初始延迟
		ewmaTokens:    0,   // 初始化 Token 消耗
		alpha:         0.2, // 权重：新数据占 20%，历史数据占 80%
		latencyWeight: 0.5, // 延迟权重 50%
		tokenWeight:   0.5, // Token 权重 50%
		baseThreshold: 200, // 综合分超过 200 就涨价
	}
}

// RecordLatency 同时接收延迟和Token消耗
func (c *RajomonController) RecordLatency(latency time.Duration, tokenCount int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 1. 将纳秒转换为毫秒 float64
	latencyMs := float64(latency.Milliseconds())
	tokens := float64(tokenCount)

	// 2. EWMA 公式：更新平均值
	if c.ewmaLatency == 0 {
		c.ewmaLatency = latencyMs // 第一次直接赋值
	} else {
		// 新平均值 = 0.2 * 本次耗时 + 0.8 * 旧平均值
		c.ewmaLatency = c.alpha*latencyMs + (1-c.alpha)*c.ewmaLatency
	}

	// 3. EWMA 更新 Token 消耗
	if c.ewmaTokens == 0 {
		c.ewmaTokens = tokens
	} else {
		c.ewmaTokens = c.alpha*tokens + (1-c.alpha)*c.ewmaTokens
	}

	// 4. 计算综合得分
	// 假设：1ms延迟 = 1分，1个Token = 1分 (你需要根据实际情况归一化)
	compositeCost := (c.latencyWeight * c.ewmaLatency) + (c.tokenWeight * c.ewmaTokens)

	// [新增] 埋点：记录计算出的综合成本
	metrics.CompositeCost.Set(compositeCost)

	// 5. 动态定价
	if compositeCost > c.baseThreshold {
		c.CurrentPrice++
		fmt.Printf("📈 [Controller] 成本过高(Lat:%.0f, Tok:%.0f, Cost:%.0f) -> 涨价至 %d\n",
			c.ewmaLatency, c.ewmaTokens, compositeCost, c.CurrentPrice)
	} else if compositeCost < c.baseThreshold/2 && c.CurrentPrice > 1 {
		c.CurrentPrice--
		fmt.Printf("📉 [Controller] 成本回落(Cost:%.0f) -> 降价至 %d\n", compositeCost, c.CurrentPrice)
	}

	// [新增] 埋点：记录最新价格
	metrics.CurrentPrice.Set(float64(c.CurrentPrice))
}

func (c *RajomonController) GetPrice() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.CurrentPrice
}

// Lock()（写锁/互斥锁）：
// 排他性：一旦某个 Goroutine 持有了写锁，其他任何 Goroutine（无论是想读还是想写）都必须等待，直到该锁被释放。
// 用途：用于修改数据（写操作）。

// RLock()（读锁）：
// 共享性：多个 Goroutine 可以同时持有读锁。只要没有 Goroutine 持有写锁，多个读操作可以并行执行。
// 用途：用于读取数据（读操作）。
