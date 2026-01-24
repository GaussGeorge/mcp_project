package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptrace"
	"rajomon-gateway/internal/model" // 引用你的 model 包以便解析 JSON
	"strconv"
	"strings"
	"time"
)

func main() {
	// 注意：如果你的 Gateway 实现了 SSE 转发，这里应该是 Gateway 的地址
	url := "http://localhost:8080/mcp/chat"

	myTokenBalance := 100 // 增加一点余额以便测试
	lastKnownPrice := 0

	for i := 1; i <= 20; i++ {
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
						fmt.Printf("   -> 📝 内容片段: %s\n", msg.Content)
					}
				} else if currentEvent == "usage" {
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

		time.Sleep(1 * time.Second)
	}
}
