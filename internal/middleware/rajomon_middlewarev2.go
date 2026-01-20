package middleware

import (
	"fmt"
	"net/http"
	"rajomon-gateway/internal/controller"
	"strconv"
	"time"
)

// 模拟一个全局控制器（实际项目中应该注入 Controller 实例）
// var currentPrice = 5

func RajomonMiddleware(ctrl *controller.RajomonController, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// 1. 获取最新价格
		price := ctrl.GetPrice()

		// 2. 无论成功失败，先贴上价格标签 (Piggybacking)
		// 这是 Rajomon 的灵魂：通过报错来传播价格信息
		w.Header().Set("Price", fmt.Sprintf("%d", price))

		// 策略 B: 随机概率回传 (进阶优化，论文提到的点)
		// ====================================================
		// 只有 10% 的概率在 Header 里写价格，节省序列化开销
		// 但是！如果下面要拒绝请求，则必须强制写回 (见下方)
		/*
			shouldPiggyback := rand.Intn(100) < 10
			if shouldPiggyback {
				w.Header().Set("X-Rajomon-Price", fmt.Sprintf("%d", currentPrice))
			}
		*/

		// 3. 获取客户端带来的 Token
		tokenStr := r.Header.Get("Token")
		clientToken, _ := strconv.Atoi(tokenStr)

		// 4. 【关键】准入检查 (token < price)
		if tokenStr == "" {
			http.Error(w, "No Token", http.StatusForbidden)
			return
		} else if clientToken < price {
			// Log 一下，方便观察
			fmt.Printf("⛔ [拒绝] Token不足! 客户带了:%d < 当前价格:%d\n", clientToken, price)
			// 返回 429 错误
			http.Error(w, "System is busy (Price > Token)", http.StatusTooManyRequests)
			// 🛑 核心：直接返回，不要执行 next.ServeHTTP！
			// 这样保护了后面的业务逻辑不被压垮
			return
		}

		// --- 4. 计时 ---
		start := time.Now()

		// --- 5. 执行业务 ---
		next.ServeHTTP(w, r)

		// --- 6. [写大脑] 反馈延迟 ---
		// 请求结束，把耗时汇报给 Controller
		latency := time.Since(start)
		//因为 SSE 是流式请求，next.ServeHTTP(w, r) 会一直阻塞直到流结束。
		// 所以 latency := time.Since(start) 记录的将是整个流传输完成的时间（Session Duration）
		ctrl.RecordLatency(latency)
	})
}
