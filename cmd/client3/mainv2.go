package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"net/http/httptrace"
	"rajomon-gateway/internal/model" // 引用你的 model 包以便解析 JSON
	"strconv"
	"strings"
	"sync"
	"time"
)

// 定义出价策略类型
type BidStrategy string

const (
	// 策略 A: 随机出价 (Rajomon 论文 Section 3.3 "Randomized Token Spending")
	// 行为: 在 0 到 当前余额 之间随机选择一个值作为出价。
	// 效果: 随价格上涨，请求被丢弃的概率线性增加。实现 "概率性负载丢弃"。
	BidStrategyRandom BidStrategy = "random"

	// 策略 B: 全额/固定出价
	// 行为: 总是使用当前钱包里的所有余额进行出价 (All-in)。
	// 效果: 只要余额 > 价格就一定通过。适合高优先级/VIP流量。
	BidStrategyFixed BidStrategy = "fixed"
)

type Wallet struct {
	balance int64
	mu      sync.Mutex
	max 	int64
}

func NewWallet(initial, max int64) *Wallet {
	return &Wallet{balance: initial , max: max}
}

// Add 充值代币
func (w *Wallet) Add(amount int64) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.balance += amount
	// 限制最大余额，防止无限囤积
	if w.balance > w.max {
		w.balance = w.max
	}
}

func (w *Wallet) TrySpend(amount int64) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.balance >= amount {
		w.balance -= amount
		return true
	}
	return false
}

func (w *Wallet) Balance() int64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.balance
}

// GetBalanceAndSpend 结合查询和扣费的原子操作
// strategy: 出价策略
// 返回值: 实际出价(bid), 是否成功扣费
func (w *Wallet) GetBalanceAndSpend(strategy BidStrategy) (int64, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()

	currentBalance := w.balance
	if currentBalance <= 0 {
		return 0, false
	}

	var bid int64

	if strategy == BidStrategyFixed {
		// --- 策略: 全力出价 (All-in) ---
		// 我有多少钱，就出多少价，确保最大概率通过
		bid = currentBalance
	} else {
		// --- 策略: 随机出价 (Random Uniform) ---
		// 模拟请求的“紧迫程度”是随机的。
		// Rajomon 论文核心: token = fastrand.Int63n(tok)
		// 注意: Int63n 参数必须 > 0
		bid = rand.Int63n(currentBalance) + 1
	}

	// 扣除钱包余额 (注意：这里简化为出价即扣除，真实场景可能是预扣或仅扣除实际价格)
	// 在 Rajomon 参考代码中，是先计算 tok，然后 DeductTokens(tok)
	w.balance -= bid
	return bid, true
}



// StartTokenGenerator 启动代币生成器
// distType: "poisson" (泊松), "fixed" (固定), "uniform" (均匀随机)
// rate: 平均生成间隔 (例如 200ms 一次)
// step: 每次生成的基准数量 (例如 10 个)
func (w *Wallet) StartTokenGenerator(distType string, rate time.Duration, step int64) {
	go func() {
		fmt.Printf("🔋 [Generator] 代币生成器启动 | 模式: %s | 速率: %v/次 | 步长: %d\n", distType, rate, step)
		if distType == "poisson" {
			// --- 模式 A: 泊松分布 (Poisson Process) ---
			// 模拟真实世界中具有随机性和突发性的到达
			// lambda = 1 / 期望间隔(ms)
			lambda := 1.0 / float64(rate.Milliseconds())
			for {
				// 1. 发放代币
				w.Add(step)

				// 2. 计算下一次间隔 (指数分布)
				// 公式: interval = -ln(U) / lambda
				// 这里的 interval 单位是 毫秒
				nextIntervalMs := -math.Log(rand.Float64()) / lambda
				if nextIntervalMs < 1 {
					nextIntervalMs = 1
				}
			}
		}else {
			// --- 模式 B & C: 基于 Ticker 的定期生成 ---
			ticker := time.NewTicker(rate)
			defer ticker.Stop()

			for range ticker.C {
				amount := step

				if distType == "fixed" {
					// 模式 B: 固定值 (Fixed) - 最死板，容易造成惊群效应
					amount = step
				} else if distType == "uniform" {
					// 模式 C: 均匀分布 (Uniform) - 在 0 ~ 2*step 之间波动
					// 平均值依然是 step，但每次给的不一样
					amount = rand.Int63n(step*2 + 1)
				}
				
				w.Add(amount)
			}
		}
	}()
}


func main() {

	// 初始化随机数种子
	rand.Seed(time.Now().UnixNano())

	// 注意：如果你的 Gateway 实现了 SSE 转发，这里应该是 Gateway 的地址
	targetURL := "http://localhost:8080/mcp/chat"

	// --- 🔥【实验配置区】请在这里修改参数 ---
	
	// 1. 代币补充策略 (收入)
	tokenRefillDist := "poisson" 
	tokenUpdateRate := 200 * time.Millisecond // 平均 200ms 发一次钱
	tokenUpdateStep := int64(15)              // 每次发 15 块钱 (也就是工资 75 Token/秒)

	// 2. 出价策略 (支出) - 这里就是你要的开关
	// 可选: BidStrategyRandom (随机/普通用户) 或 BidStrategyFixed (VIP/紧急任务)
	bidStrategy := BidStrategyRandom

	fmt.Printf("🛠️  当前出价策略: %s\n", bidStrategy)


	// 初始化钱包
	wallet := NewWallet(50,500)

	// 启动生成器
	wallet.StartTokenGenerator(tokenRefillDist, tokenUpdateRate, tokenUpdateStep)

	lastKnownPrice := int64(0)


	for i := 1; i <= 50; i++ {
		// 模拟用户请求的随机间隔 (思考时间)
		time.Sleep(time.Duration(rand.Intn(1000)+500) * time.Millisecond)

		fmt.Printf("\n--- 第 %d 次请求 ---\n", i)

		// // 1. 根据策略获取出价并扣费
		bid, ok := wallet.GetBalanceAndSpend(bidStrategy)

		if !ok || bid == 0 {
			fmt.Printf("💸 [本地拦截] 钱包空空如也，跳过本次请求\n")
			continue
		}


		// 2. 价格检查 (本地熔断)
		// 如果我们出的价(bid) 甚至低于 市场价(lastKnownPrice)，那就没必要发请求了，必挂。
		if lastKnownPrice > 0 && bid < lastKnownPrice {
			fmt.Printf("🚫 [本地拦截] 出价过低! 出价(%d) < 市价(%d) | 策略: %s\n", 
				bid, lastKnownPrice, bidStrategy)
			// 注意：这部分代币已经被扣除了，模拟“尝试成本”或者你可以选择退回
			continue
		}

		fmt.Printf("🚀 [发起请求] 出价: %d | 策略: %s | 预估市价: %d\n", bid, bidStrategy, lastKnownPrice)
		doRequest(targetURL, bid, &lastKnownPrice)
	}
}

func doRequest(url string, tokenAmount int64, lastPrice *int64) {
	// 定义 Trace
	trace := &httptrace.ClientTrace{
		GotFirstResponseByte: func() {
			// fmt.Println("👀 [Trace] 连接建立")
		},
	}
	ctx := httptrace.WithClientTrace(context.Background(), trace)

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Token", fmt.Sprintf("%d", tokenAmount))

	client := &http.Client{Timeout: 30 * time.Second}
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ 请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// 更新价格感知
	priceStr := resp.Header.Get("Price")
	if priceStr != "" {
		newPrice, _ := strconv.ParseInt(priceStr, 10, 64)
		if newPrice != *lastPrice {
			fmt.Printf("🏷️ [情报] 价格更新: %d -> %d\n", *lastPrice, newPrice)
			*lastPrice = newPrice
		}
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		fmt.Printf("❌ [服务端拒绝] 429 Too Many Requests (Token < Price)\n")
		return
	}

	// 简单读取流 (只读不解析，为了模拟耗时)
	scanner := bufio.NewScanner(resp.Body)
	var currentEvent string // 记录当前处理的事件类型

	// 逐行扫描
	for scanner.Scan() {
		line := scanner.Text()

		// 1. 解析事件类型 (如 event: message 或 event: usage)
		if strings.HasPrefix(line, "event:") {
			currentEvent = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			// fmt.Printf("   [Debug] 切换事件类型为: %s\n", currentEvent)
			continue
		}

		// SSE 格式通常是 "data: {json...}"
		// 2. 解析数据内容 (data: {...})
		if strings.HasPrefix(line, "data:") {
			dataContent := strings.TrimPrefix(line, "data:")

			if currentEvent == "message" {
				// 解析普通文本内容
				var msg model.MockContent
				if err := json.Unmarshal([]byte(dataContent), &msg); err == nil {
					// fmt.Printf("   -> 📝 内容片段: %s\n", msg.Content)
					// 简化输出，只打印点点点表示正在接收
					fmt.Print(".")
				}
			} else if currentEvent == "usage" {
				fmt.Print(" [Done]\n")
				// [重点] 解析 Token 消耗数据
				var usage model.MockUsage
				err := json.Unmarshal([]byte(dataContent), &usage)
				if err == nil {
					fmt.Printf("   -> 💰 [成本核算] Prompt: %d, Completion: %d, Total: %d\n",
						usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)
				} else {
					// 如果解析失败，打印出来
					fmt.Printf("   ❌ [错误] Usage 解析失败: %v, 内容: %s\n", err, dataContent)
				}
			} else {
				// 打印当前未知的 Event 类型，帮助排查是否 currentEvent 没切过来
				fmt.Printf("   -> 未知数据 (Event=%s): %s\n", currentEvent, dataContent)
			}
		}
		// SSE 消息通常以空行结束，重置事件类型
		if line == "" {
			currentEvent = ""
		}
	}

	resp.Body.Close() // 只有在流结束或出错时才关闭
	fmt.Printf("✅ 请求完成 ,⏱️ 总耗时: %v\n", time.Since(start))
}