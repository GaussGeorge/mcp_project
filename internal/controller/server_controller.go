package controller

import (
	"fmt"
	"sync"
	"time"
)

type RajomonController struct {
	mu           sync.RWMutex
	CurrentPrice int

	// --- EWMA 核心字段 ---
	ewmaLaatency float64 // 当前的平均延迟 (毫秒)
	alpha        float64 // 平滑因子
	threshold    float64 // 目标阈值
}

func NewController() *RajomonController {
	return &RajomonController{
		CurrentPrice: 5,   // 初始价格
		ewmaLaatency: 0,   // 初始延迟
		alpha:        0.2, // 权重：新数据占 20%，历史数据占 80%
		threshold:    200, // 超过 200ms 就涨价
	}
}

// RecordLatency 是核心更新逻辑
func (c *RajomonController) RecordLatency(latency time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 1. 将纳秒转换为毫秒 float64
	latencyMs := float64(latency.Milliseconds())

	// 2. EWMA 公式：更新平均值
	if c.ewmaLaatency == 0 {
		c.ewmaLaatency = latencyMs // 第一次直接赋值
	} else {
		// 新平均值 = 0.2 * 本次耗时 + 0.8 * 旧平均值
		c.ewmaLaatency = c.alpha*latencyMs + (1-c.alpha)*c.ewmaLaatency
	}

	// 3. 基于“平滑后”的延迟来定价
	if c.ewmaLaatency > c.threshold {
		c.CurrentPrice++
		fmt.Printf("📈 [Controller] 平均延迟 %.2fms > 阈值，涨价至 %d\n", c.ewmaLaatency, c.CurrentPrice)
	} else if c.ewmaLaatency < c.threshold/2 && c.CurrentPrice > 1 {
		// 如果延迟很低 (小于 100ms)，慢慢降价
		c.CurrentPrice--
		fmt.Printf("📉 [Controller] 平均延迟 %.2fms < 阈值/2，降价至 %d\n", c.ewmaLaatency, c.CurrentPrice)
	}
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
