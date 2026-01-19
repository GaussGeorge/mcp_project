package middleware

import (
	"fmt"
	"net/http"
	"time"
)

// LatencyMiddleware 用于计算排队延迟 (Rajomon 核心)
func LatencyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: 使用 httptrace 计算延迟
		// 1. 记录开始时间
		start := time.Now()

		// 2. 执行后续的业务逻辑(Next)
		next.ServeHTTP(w, r)

		// 3. 计算耗时
		duration := time.Since(start)

		// 4. 打印或记录日志(Rajomon 会在这里根据duration调整价格)
		fmt.Printf("[中间件] 请求路径: %s | 耗时: %v\n", r.URL.Path,duration)

		// ⚠️ 注意：这里通常不能再写 Header 了，因为 next.ServeHTTP 内部可能已经写回了响应。
		// 如果要写 Header，必须使用自定义的 ResponseWriter (这是进阶内容，暂时先不展开)
	})
}
