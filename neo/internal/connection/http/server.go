package http

import (
	"fmt"
	"neo/internal/types"
	"net/http"
)

// 新增包级错误变量，封装标准库错误
var (
	ErrServerClosed = http.ErrServerClosed
)

// Server 表示HTTP服务器实例
type Server struct {
	config     *types.HTTPConfig // 修改为HTTPConfig
	httpServer *http.Server
}

// NewServer 创建新的HTTP服务器实例
func NewServer(config *types.HTTPConfig) *Server {
	return &Server{
		config: config,
	}
}

// Start 启动HTTP服务器
func (s *Server) Start() error {
	// 使用自定义Router替代标准库ServeMux
	router := NewRouter()
	// 注册HTTP处理器
	s.registerHandlers(router)

	// 使用HTTP配置构建地址
	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.config.Host, s.config.Port),
		Handler: router, // 使用自定义Router作为处理器
	}

	fmt.Printf("Starting HTTP server on %s:%d...\n", s.config.Host, s.config.Port)

	if s.config.EnableHTTPS && s.config.CertFile != "" && s.config.KeyFile != "" {
		return s.httpServer.ListenAndServeTLS(s.config.CertFile, s.config.KeyFile)
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
// 修改registerHandlers方法，使用自定义Router
func (s *Server) registerHandlers(router *Router) {
	// 注册实际的HTTP处理路由
	router.Handle("/health", s.handleHealthCheck)
	// 添加其他路由...
}

// handleHealthCheck 处理健康检查请求
func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
