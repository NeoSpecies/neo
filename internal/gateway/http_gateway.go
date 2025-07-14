package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"neo/internal/core"
	"neo/internal/registry"
	"neo/internal/types"
	"net/http"
	"strings"
	"time"
)

// HTTPGateway HTTP网关，处理HTTP请求并转发到进程间服务
type HTTPGateway struct {
	service  core.Service
	registry registry.ServiceRegistry
	server   *http.Server
}

// NewHTTPGateway 创建新的HTTP网关
func NewHTTPGateway(service core.Service, registry registry.ServiceRegistry, addr string) *HTTPGateway {
	gw := &HTTPGateway{
		service:  service,
		registry: registry,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/", gw.handleAPIRequest)
	mux.HandleFunc("/health", gw.handleHealth)

	gw.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return gw
}

// Start 启动HTTP服务器
func (gw *HTTPGateway) Start() error {
	fmt.Printf("HTTP Gateway listening on %s\n", gw.server.Addr)
	return gw.server.ListenAndServe()
}

// Stop 停止HTTP服务器
func (gw *HTTPGateway) Stop(ctx context.Context) error {
	return gw.server.Shutdown(ctx)
}

// HandleAPIRequest exposes the API request handler for testing
func (gw *HTTPGateway) HandleAPIRequest(w http.ResponseWriter, r *http.Request) {
	gw.handleAPIRequest(w, r)
}

// HandleHealth exposes the health check handler for testing
func (gw *HTTPGateway) HandleHealth(w http.ResponseWriter, r *http.Request) {
	gw.handleHealth(w, r)
}

// handleAPIRequest 处理API请求，转发到进程间服务
func (gw *HTTPGateway) handleAPIRequest(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("handleAPIRequest: Received %s %s\n", r.Method, r.URL.Path)
	
	// 解析路径: /api/{service}/{method}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/"), "/")
	if len(parts) < 2 {
		fmt.Printf("handleAPIRequest: Invalid path, parts=%v\n", parts)
		http.Error(w, "Invalid API path. Expected: /api/{service}/{method}", http.StatusBadRequest)
		return
	}

	serviceName := parts[0]
	method := parts[1]
	fmt.Printf("handleAPIRequest: serviceName='%s', method='%s'\n", serviceName, method)

	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// 构建请求
	req := types.Request{
		ID:       generateRequestID(),
		Service:  serviceName,
		Method:   method,
		Body:     body,
		Metadata: make(map[string]string),
	}

	// 复制HTTP头到请求元数据
	for key, values := range r.Header {
		if len(values) > 0 {
			req.Metadata[key] = values[0]
		}
	}

	// 添加HTTP方法到元数据
	req.Metadata["http-method"] = r.Method

	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 调用服务
	fmt.Printf("handleAPIRequest: Calling service.HandleRequest for %s/%s\n", serviceName, method)
	resp, err := gw.service.HandleRequest(ctx, req)
	if err != nil {
		fmt.Printf("handleAPIRequest: Service call failed: %v\n", err)
		http.Error(w, fmt.Sprintf("Service call failed: %v", err), http.StatusInternalServerError)
		return
	}
	fmt.Printf("handleAPIRequest: Service call successful, status=%d\n", resp.Status)

	// 设置响应头
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.Status)

	// 写入响应体
	if _, err := w.Write(resp.Body); err != nil {
		fmt.Printf("Failed to write response: %v\n", err)
	}
}

// handleHealth 健康检查端点
func (gw *HTTPGateway) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// generateRequestID 生成请求ID
func generateRequestID() string {
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}