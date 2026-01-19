package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"strconv"
	"time"
)

func main() {
	url := "http://localhost:8080/context"

	myTokenBalance := 20
	lastKnownPrice := 0

	for i := 1; i <= 10; i++ {
		fmt.Printf("\n--- 第 %d 次尝试 ---\n", i)

		if myTokenBalance < lastKnownPrice {
			fmt.Printf("🚫 [本地拦截] 没钱了! 余额(%d) < 最新报价(%d)。放弃请求，休眠等待降价...\n",
				myTokenBalance, lastKnownPrice)

			time.Sleep(2 * time.Second)
			lastKnownPrice = 0 //----
			continue
		}
	}

	trace := &httptrace.ClientTrace{
		GotConn: func(connInfo httptrace.GotConnInfo) {
			fmt.Printf("🔗 [Trace] 连接建立成功 (复用: %v)\n", connInfo.Reused)
		},

		WroteRequest: func(info httptrace.WroteRequestInfo) {
			fmt.Println("✉️ [Trace] 请求已发送，开始计时等待服务端...")
		},
		GotFirstResponseByte: func() {
			fmt.Println("👀 [Trace] 收到首字节 (服务端开始吐数据了)")
		},
	}

	ctx := httptrace.WithClientTrace(context.Background(), trace)

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("Token", fmt.Sprintf("%d", myTokenBalance))

	start := time.Now()
	client := &http.Client{}
	resp, err := client.Do(req)

	// totalTime := time.Since(start)

	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
		return
	}

	defer resp.Body.Close()

	priceStr := resp.Header.Get("Price")
	if priceStr != "" {
		newPrice, _ := strconv.Atoi(priceStr)
		// 只有价格变化了才打印，避免刷屏
		if newPrice != lastKnownPrice {
			fmt.Printf("🏷️ [情报] 服务端更新报价: %d -> %d\n", lastKnownPrice, newPrice)
			lastKnownPrice = newPrice
		}
	}

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("响应: %s\n", string(body))
	fmt.Printf("✅ 响应: %s ,⏱️ 总耗时: %v\n", string(body), time.Since(start))

	if resp.StatusCode == http.StatusTooManyRequests {
		fmt.Println("❌ [服务端拒绝] 即使发出去也被拒了 (可能刚好涨价)")
	}

	time.Sleep(1 * time.Second)
}
