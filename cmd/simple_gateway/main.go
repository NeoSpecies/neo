package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// 简化的网关实现，用于测试基本功能
func main() {
	fmt.Println("Starting Simple Gateway on :8080")
	
	http.HandleFunc("/api/", handleAPI)
	http.HandleFunc("/health", handleHealth)
	
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

func handleAPI(w http.ResponseWriter, r *http.Request) {
	// 解析路径
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/"), "/")
	if len(parts) < 2 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	
	service := parts[0]
	method := parts[1]
	
	// 读取请求体
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()
	
	// 模拟响应
	response := map[string]interface{}{
		"service": service,
		"method":  method,
		"result":  "Simulated response",
		"time":    time.Now().Format(time.RFC3339),
	}
	
	// 特殊处理数学服务
	if service == "python.math" {
		var req map[string]interface{}
		json.Unmarshal(body, &req)
		
		switch method {
		case "add":
			if a, ok := req["a"].(float64); ok {
				if b, ok := req["b"].(float64); ok {
					response["result"] = a + b
				}
			}
		case "multiply":
			if a, ok := req["a"].(float64); ok {
				if b, ok := req["b"].(float64); ok {
					response["result"] = a * b
				}
			}
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}