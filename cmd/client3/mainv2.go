package main

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/http/httptrace"
	"strconv"
	"strings"
	"time"
)

func main() {
	// 注意：如果你的 Gateway 实现了 SSE 转发，这里应该是 Gateway 的地址
	url := "http://localhost:8080/context" 

	myTokenBalance := 100 // 增加一点余额以便测试
	lastKnownPrice := 0

	for i := 1; i <= 5; i++ {
		fmt.Printf("\n--- 第 %d 次尝试 (SSE 流式请求) ---\n", i)

		// 1. 本地拦截逻辑 (Rajomon 客户端侧)
		if myTokenBalance < lastKnownPrice {
			fmt.Printf("🚫 [本地拦截] 没钱了! 余额(%d) < 最新报价(%d)。休眠等待...\n",
				myTokenBalance, lastKnownPrice)
			time.Sleep(2 * time.Second)
			lastKnownPrice = 0 
			continue
		}

		// 2. 定义 Trace
		trace := &httptrace.ClientTrace{
			GotFirstResponseByte: func() {
				fmt.Println("👀 [Trace] 收到首字节 (SSE 流连接建立)")
			},
		}
		ctx := httptrace.WithClientTrace(context.Background(), trace)

		// 3. 构建请求
		req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
		
		// [新增] 告诉服务端我们期望 SSE 流
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Cache-Control", "no-cache")
		req.Header.Set("Connection", "keep-alive")
		
		// Rajomon Token
		req.Header.Set("Token", fmt.Sprintf("%d", myTokenBalance))

		start := time.Now()
		client := &http.Client{}
		resp, err := client.Do(req)

		if err != nil {
			fmt.Printf("❌ 请求失败: %v\n", err)
			return
		}

		// 4. 处理 Rajomon 价格头 (在建立连接时立刻读取)
		priceStr := resp.Header.Get("Price")
		if priceStr != "" {
			newPrice, _ := strconv.Atoi(priceStr)
			if newPrice != lastKnownPrice {
				fmt.Printf("🏷️ [情报] 服务端更新报价: %d -> %d\n", lastKnownPrice, newPrice)
				lastKnownPrice = newPrice
			}
		}

		// 5. 处理错误状态码
		if resp.StatusCode == http.StatusTooManyRequests {
			fmt.Println("❌ [服务端拒绝] HTTP 429: Too Many Requests")
			resp.Body.Close()
			time.Sleep(1 * time.Second)
			continue
		}

		if resp.StatusCode != 200 {
			fmt.Printf("❌ 非法状态码: %d\n", resp.StatusCode)
			resp.Body.Close()
			return
		}

		// ==========================================================
		// 6. [核心修改] 使用 Scanner 按行读取 SSE 流
		// ==========================================================
		fmt.Println("🌊 开始接收流式数据:")
		scanner := bufio.NewScanner(resp.Body)
		
		// 逐行扫描
		for scanner.Scan() {
			line := scanner.Text()

			// SSE 格式通常是 "data: {json...}"
			if strings.HasPrefix(line, "data:") {
				content := strings.TrimPrefix(line, "data:")
				// 打印接收到的数据片段
				fmt.Printf("   -> 收到片段: %s\n", strings.TrimSpace(content))
			}
			
			// 如果是 SSE 的心跳或空行，通常忽略
		}

		if err := scanner.Err(); err != nil {
			fmt.Printf("❌ 读取流出错: %v\n", err)
		}
		
		resp.Body.Close() // 只有在流结束或出错时才关闭
		fmt.Printf("✅ 请求完成 ,⏱️ 总耗时: %v\n", time.Since(start))

		time.Sleep(1 * time.Second)
	}
}