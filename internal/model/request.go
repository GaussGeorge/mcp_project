package model

// PriceRequest 定义价格请求的数据结构
type PriceRequest struct {
	ServiceID string  `json:"service_id"`
	Price     float64 `json:"price"`
}
