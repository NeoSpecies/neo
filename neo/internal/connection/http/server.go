package http

import (
	"fmt"
	"neo/internal/config"
	"neo/internal/types"
	"net/http"
)

// Server 表示HTTP服务器实例
type Server struct {
	config     *types.IPCConfig // 使用types包中的IPCConfig
	httpServer *http.Server
}

// NewServer 创建新的HTTP服务器实例
func NewServer() *Server {
	// 从全局配置获取IPC配置
	cfg := config.GetGlobalConfig()
	return &Server{
		config: &cfg.IPC,
	}
}

// Start 启动HTTP服务器
func (s *Server) Start() error {
	mux := http.NewServeMux()
	// 注册HTTP处理器
	s.registerHandlers(mux)

	// 使用Host和Port字段构建地址（修复核心问题）
	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.config.Host, s.config.Port),
		Handler: mux,
	}

	return s.httpServer.ListenAndServe()
}

// Close 关闭HTTP服务器
func (s *Server) Close() error {
	if s.httpServer != nil {
		return s.httpServer.Close()
	}
	return nil
}

// registerHandlers 注册HTTP处理器
func (s *Server) registerHandlers(mux *http.ServeMux) {
	// 注册实际的HTTP处理路由
	mux.HandleFunc("/health", s.handleHealthCheck)
	// 添加其他路由...
}

// handleHealthCheck 处理健康检查请求
func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
