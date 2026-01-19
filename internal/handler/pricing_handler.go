package handler

import (
	"fmt"
	"net/http"
	"time"
)

func PricingHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	time.Sleep(100 * time.Millisecond)

	latency := time.Since(start)

	currentPrice := 5
	if latency > 50*time.Millisecond {
		currentPrice = 20
	}

	w.Header().Set("Price", fmt.Sprintf("%d", currentPrice))
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK)

}
