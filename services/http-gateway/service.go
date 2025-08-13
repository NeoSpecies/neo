package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// HTTPGatewayService HTTP网关服务
type HTTPGatewayService struct {
	// IPC客户端
	ipcClient   *IPCClient
	serviceName string

	// HTTP服务器
	httpServer *http.Server
	httpAddr   string

	// 请求ID管理
	requestCounter uint64
	mu             sync.Mutex
}

// NewHTTPGatewayService 创建HTTP网关服务
func NewHTTPGatewayService(httpAddr string) *HTTPGatewayService {
	return &HTTPGatewayService{
		serviceName:    "http-gateway",
		httpAddr:       httpAddr,
		requestCounter: 0,
	}
}

// ConnectToIPC 连接到IPC服务器
func (s *HTTPGatewayService) ConnectToIPC(ipcAddr string) error {
	client, err := NewIPCClient(ipcAddr)
	if err != nil {
		return fmt.Errorf("failed to connect to IPC: %w", err)
	}
	s.ipcClient = client

	// 注册HTTP网关特有的处理器
	s.registerHandlers()

	// 注册服务
	metadata := map[string]string{
		"type":        "gateway",
		"protocol":    "http",
		"version":     "1.0.0",
		"description": "HTTP Gateway Service for Neo Framework",
	}
	
	if err := s.ipcClient.RegisterService(s.serviceName, metadata); err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}

	// 启动心跳
	s.ipcClient.StartHeartbeat()

	log.Printf("HTTP Gateway registered to IPC server at %s", ipcAddr)
	return nil
}

// registerHandlers 注册IPC处理器
func (s *HTTPGatewayService) registerHandlers() {
	// 处理获取网关信息的请求
	s.ipcClient.AddHandler("getInfo", func(msg *Message) (*Message, error) {
		info := map[string]interface{}{
			"service":     s.serviceName,
			"type":        "http-gateway",
			"httpAddress": s.httpAddr,
			"status":      "running",
			"version":     "1.0.0",
		}
		
		data, _ := json.Marshal(info)
		return &Message{
			Metadata: map[string]string{},
			Data:     data,
		}, nil
	})

	// 处理调用外部HTTP API的请求（供其他服务使用）
	s.ipcClient.AddHandler("callExternalAPI", func(msg *Message) (*Message, error) {
		var params map[string]interface{}
		if err := json.Unmarshal(msg.Data, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %w", err)
		}

		url, _ := params["url"].(string)
		method, _ := params["method"].(string)
		if method == "" {
			method = "GET"
		}

		// 这里可以实现调用外部HTTP API的逻辑
		result := map[string]interface{}{
			"status":  "success",
			"message": fmt.Sprintf("Would call %s %s", method, url),
		}

		data, _ := json.Marshal(result)
		return &Message{
			Metadata: map[string]string{},
			Data:     data,
		}, nil
	})
}

// StartHTTPServer 启动HTTP服务器
func (s *HTTPGatewayService) StartHTTPServer() error {
	mux := http.NewServeMux()
	
	// API路由
	mux.HandleFunc("/api/", s.handleAPIRequest)
	
	// 健康检查
	mux.HandleFunc("/health", s.handleHealth)
	
	// 网关信息
	mux.HandleFunc("/info", s.handleInfo)

	s.httpServer = &http.Server{
		Addr:         s.httpAddr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	log.Printf("HTTP Gateway listening on %s", s.httpAddr)
	log.Printf("API endpoint: http://localhost%s/api/{service}/{method}", s.httpAddr)
	log.Printf("Health check: http://localhost%s/health", s.httpAddr)
	
	return s.httpServer.ListenAndServe()
}

// handleAPIRequest 处理API请求
func (s *HTTPGatewayService) handleAPIRequest(w http.ResponseWriter, r *http.Request) {
	// 记录请求
	log.Printf("[HTTP] %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

	// 解析路径 /api/{service}/{method}
	path := strings.TrimPrefix(r.URL.Path, "/api/")
	parts := strings.Split(path, "/")
	
	if len(parts) < 2 {
		http.Error(w, `{"error":"Invalid API path. Expected: /api/{service}/{method}"}`, http.StatusBadRequest)
		return
	}

	targetService := parts[0]
	method := parts[1]

	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error":"Failed to read request body"}`, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// 如果body为空，设置为空JSON对象
	if len(body) == 0 {
		body = []byte("{}")
	}

	// 验证JSON格式
	var jsonCheck interface{}
	if err := json.Unmarshal(body, &jsonCheck); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"Invalid JSON: %s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	// 生成请求ID
	requestID := s.generateRequestID()
	
	log.Printf("[HTTP] Forwarding request %s to service '%s', method '%s'", requestID, targetService, method)

	// 通过IPC转发请求
	response, err := s.ipcClient.SendRequest(targetService, method, body, requestID)
	if err != nil {
		log.Printf("[HTTP] IPC request failed: %v", err)
		errorResp := map[string]interface{}{
			"error":     "Service call failed",
			"details":   err.Error(),
			"service":   targetService,
			"method":    method,
			"requestId": requestID,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(errorResp)
		return
	}

	// 检查响应是否包含错误
	if response.Metadata != nil && response.Metadata["error"] == "true" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(response.Data)
		log.Printf("[HTTP] Request %s returned error: %s", requestID, string(response.Data))
		return
	}

	// 返回成功响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(response.Data)
	
	log.Printf("[HTTP] Request %s completed successfully", requestID)
}

// handleHealth 健康检查
func (s *HTTPGatewayService) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"service":   s.serviceName,
		"timestamp": time.Now().Format(time.RFC3339),
		"ipc":       s.ipcClient != nil,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// handleInfo 网关信息
func (s *HTTPGatewayService) handleInfo(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"service":     s.serviceName,
		"type":        "http-gateway",
		"version":     "1.0.0",
		"httpAddress": s.httpAddr,
		"endpoints": map[string]string{
			"api":    "/api/{service}/{method}",
			"health": "/health",
			"info":   "/info",
		},
		"description": "HTTP Gateway Service for Neo Framework",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// generateRequestID 生成请求ID
func (s *HTTPGatewayService) generateRequestID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requestCounter++
	return fmt.Sprintf("http-%d-%d", time.Now().Unix(), s.requestCounter)
}

// Stop 停止服务
func (s *HTTPGatewayService) Stop() error {
	if s.httpServer != nil {
		if err := s.httpServer.Close(); err != nil {
			return err
		}
	}
	if s.ipcClient != nil {
		return s.ipcClient.Close()
	}
	return nil
}