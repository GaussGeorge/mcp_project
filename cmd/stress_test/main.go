package main

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// 模拟三种用户
type UserType struct {
	Name    string
	Balance int // 用户的 Token 余额
	Count   int // 并发数
}

var (
	// 增加并发数，确保压力足够大
	PoorUsers   = UserType{Name: "Student", Balance: 10, Count: 5}
	MiddleUsers = UserType{Name: "Engineer", Balance: 20, Count: 5}
	RichUsers   = UserType{Name: "VIP_Boss", Balance: 100, Count: 5}

	targetURL = "http://localhost:8080/mcp/chat"
)

func main() {
	fmt.Println("🚀 Rajomon 压力测试器启动 (修正版)...")
	fmt.Println("🌊 正在模拟完整对话 (读取 Body)，迫使服务端计算满 700ms...")

	var wg sync.WaitGroup

	// 启动所有类型的模拟用户
	startUserGroup(&wg, PoorUsers)
	startUserGroup(&wg, MiddleUsers)
	startUserGroup(&wg, RichUsers)

	wg.Wait()
}

func startUserGroup(wg *sync.WaitGroup, u UserType) {
	for i := 0; i < u.Count; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			client := &http.Client{Timeout: 30 * time.Second}

			for {
				// 模拟用户思考间隔 (0.5s ~ 1.5s)
				time.Sleep(time.Duration(rand.Intn(1000)+500) * time.Millisecond)

				req, _ := http.NewRequest("GET", targetURL, nil)
				req.Header.Set("Token", strconv.Itoa(u.Balance))
				req.Header.Set("User-Agent", u.Name)

				start := time.Now()
				resp, err := client.Do(req)
				if err != nil {
					fmt.Printf("❌ [%s] 网络错误: %v\n", u.Name, err)
					continue
				}

				// 🔥【核心修正】必须读取完 Body，才能模拟出服务端的真实耗时！
				// io.Discard 就像一个黑洞，把数据读出来扔掉，但会消耗时间
				io.Copy(io.Discard, resp.Body)

				// 此时才算请求真正结束
				duration := time.Since(start)

				currentPrice := resp.Header.Get("Price")
				resp.Body.Close() // 读完再关

				if resp.StatusCode == 200 {
					fmt.Printf("✅ [%s] (Bal:%d) 成功 | 🏷️ 现价:%s | ⏱️ %v\n",
						u.Name, u.Balance, currentPrice, duration)
				} else if resp.StatusCode == 429 {
					fmt.Printf("⛔ [%s] (Bal:%d) 被拦截! | 🏷️ 现价:%s > 余额 | 🚫 429 Too Many Requests\n",
						u.Name, u.Balance, currentPrice)
				}
			}
		}(i)
	}
}
