package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"time"
)

func main() {
	url := "http://localhost:8080/context" // 假设这是你的慢接口

	// 1. 定义 Trace 钩子
	trace := &httptrace.ClientTrace{
		// 当由 Dial 完成 TCP 连接时调用
		GotConn: func(connInfo httptrace.GotConnInfo) {
			fmt.Printf("🔗 [Trace] 连接建立成功 (复用: %v)\n", connInfo.Reused)
		},

		// 当客户端写完请求，开始等待响应时调用
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			fmt.Println("✉️ [Trace] 请求已发送，开始计时等待服务端...")
		},

		// 关键：当收到服务端返回的第一个字节时调用
		// 这段时间 = 网络传输 + 服务端排队 + 服务端计算
		GotFirstResponseByte: func() {
			fmt.Println("👀 [Trace] 收到首字节 (服务端开始吐数据了)")
		},
	}

	// 2. 将 Trace 注入 Context
	ctx := httptrace.WithClientTrace(context.Background(), trace)

	// 3. 创建 Request 并使用带 Trace 的 Context
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("Token", "100")

	// 4. 发送请求
	start := time.Now()
	client := &http.Client{}
	resp, err := client.Do(req)

	totalTime := time.Since(start)

	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
		return
	}

	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("响应: %s\n", string(body))
	fmt.Printf("⏱️ 总耗时: %v\n", totalTime)

	// ContextTestClient()

	// ------------- Request版本--------------
	// url := "http://localhost:8080/mcp"
	// fmt.Println("正在构建请求...")

	// // ==========================================================
	// // 1. 构建请求对象 (Create Request)
	// // ==========================================================
	// // http.NewRequest 不会立刻发送，它只是创建一个对象让你配置
	// // 参数: Method("GET"), URL, Body(nil 表示没有请求体)
	// req, err := http.NewRequest("GET", url, nil)
	// if err != nil {
	// 	fmt.Println("构建请求失败:", err)
	// 	return
	// }

	// // ==========================================================
	// // 2. 注入 Token (Set Header) —— 关键步骤！
	// // ==========================================================
	// // 这里的 Key "Token" 必须和服务端 RajomonMiddleware 里
	// // r.Header.Get("Token") 保持一致（大小写不敏感，但建议统一）
	// req.Header.Set("Token", "")

	// // 你甚至可以模拟 Rajomon 论文中的不同用户
	// // req.Header.Set("User-ID", "user_001")

	// // ==========================================================
	// // 3. 发送请求 (Execute Request)
	// // ==========================================================
	// // 我们需要一个 Client 来执行这个 Request
	// client := &http.Client{}
	// fmt.Println("正在向服务端发送请求带 Token 的请求...")
	// resp, err := client.Do(req)
	// if err != nil {
	// 	fmt.Println("请求发送失败:", err)
	// 	return
	// }
	// defer resp.Body.Close()

	// // 读取响应
	// // 如果 Token 有效，这里应该能读到 "MCP Result Success"
	// // 如果 Token 无效，这里会读到 "Rajomon: No Token"
	// body, err := io.ReadAll(resp.Body)
	// if err != nil {
	// 	fmt.Println("读取服务端数据失败:", err)
	// 	return
	// }

	// fmt.Printf("状态码： %d\n", resp.StatusCode)
	// fmt.Printf("服务端响应数据: %s\n", string(body))

	// ---------------- 快捷函数测试版本 ----------------"http://localhost:8080/mcp"
	// // 1. 发起请求(Request)
	// // 相当于浏览器在地址栏输入 http://localhost:8080/
	// url := "http://localhost:8080/mcp"
	// fmt.Println("正在向服务端发送请求")

	// resp, err := http.Get(url)
	// if err != nil {
	// 	fmt.Println("请求失败")
	// 	return

	// }

	// // 2. 资源释放(Defer Close)
	// // 必须关闭 Body，否则 TCP 连接无法复用，会导致资源泄露
	// // 这在你的论文实验中非常关键，高并发下不关 Body 会直接崩
	// defer resp.Body.Close()

	// // 3. 读取响应(Response)
	// // 读取服务端返回的数据
	// body, err := io.ReadAll(resp.Body)
	// if err != nil {
	// 	fmt.Println("读取服务端数据失败:", err)
	// 	return
	// }

	// // 4. 打印结果(Print)
	// fmt.Printf("服务端响应数据：%s\n", string(body))
}
