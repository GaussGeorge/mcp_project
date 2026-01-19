// package middleware

// import (
// 	"fmt"
// 	"net/http"
// 	"time"
// )

// // 模拟一个全局控制器（实际项目中应该注入 Controller 实例）
// var currentPrice = 5

// func RajomonMiddleware(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		// --- 1. 准入控制 (Admission Control) ---
// 		token := r.Header.Get("Token")

// 		if token == "" {
// 			http.Error(w, "No Token", http.StatusForbidden)
// 			return
// 		}

// 		// --- 2. 【关键修改】捎带价格 (Piggybacking) ---
// 		// 必须在 next.ServeHTTP 之前设置 Header！
// 		// 这样无论业务逻辑什么时候 Write，Header 里都已经有了 Price
// 		w.Header().Set("Price", fmt.Sprintf("%d", currentPrice))

// 		start := time.Now()

// 		// --- 3. 执行业务逻辑 ---
// 		// 你的 ContextHandler 在这里执行。
// 		// 当它调用 w.Write 时，带着 Price 的 Header 会一同发出。
// 		next.ServeHTTP(w, r)

//			// --- 4. 反馈循环 (Feedback Loop) ---
//			// 计算耗时，准备更新下一次的价格
//			latency := time.Since(start)
//			if latency > 200*time.Millisecond {
//				currentPrice++ // 模拟涨价
//				fmt.Printf("⚠️ [中间件] 耗时 %v > 阈值，价格上涨至 %d\n", latency, currentPrice)
//			}
//		})
//	}
package middleware
