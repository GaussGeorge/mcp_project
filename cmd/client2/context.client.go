package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

func ContextTestClient() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, _ := http.NewRequest("GET", "http://localhost:8080/context", nil)

	req.Header.Add("Token", "TOKEN_100")

	req = req.WithContext(ctx)

	fmt.Println("客户端：发送请求，限时 3 秒...")

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		// 预期结果：因为服务端要 2秒，这里 1秒 就会报错
		fmt.Printf("客户端：请求失败 (符合预期): %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("客户端：收到响应: %s\n", string(body))
}
