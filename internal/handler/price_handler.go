package handler

import (
	"fmt"
	"net/http"
)

// 定义一个普通的结构体
// 在你的 Rajomon 论文中，这个结构体可能叫 GovernanceNode
type MyGovernanceHandler struct {
	Price int // 比如：存储当前服务的“价格”
}

// 核心：让MyGovernanceHandler实现ServeHTTP方法
func (h *MyGovernanceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 这里是所有请求的入口
	fmt.Printf("收到请求，当前价格是：%d\n", h.Price)

	// 写入响应
	fmt.Fprintf(w, "Payment required: %d", h.Price)

}

// func main() {
// 	// 初始化我们的 Handler
// 	myHandler := &MyGovernanceHandler{Price: 10}

// 	// 启动服务
// 	// 注意ListenAndServe的第二个参数是我们的自定义Handler
// 	// 因为myHandler实现了ServeHTTP方法，所以可以传进去
// 	fmt.Println("Server is running at :8080")
// 	http.ListenAndServe(":8080", myHandler)
// }
