package http

import (
    "net/http"
    "fmt"
    "neo/internal/config"
    "neo/internal/transport"
)

// HTTPServer HTTP协议服务实现
type HTTPServer struct {
    server     *http.Server
    config     *config.HTTPConfig
    router     *http.ServeMux  // 使用标准库ServeMux替代未定义的Router
    workerPool *transport.WorkerPool
    isRunning  bool
}

// NewHTTPServer 创建HTTP服务器实例
func NewHTTPServer(cfg *config.HTTPConfig, workerPool *transport.WorkerPool) (*HTTPServer, error) {
    // 创建标准库路由
    router := http.NewServeMux()
    addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
    
    return &HTTPServer{
        config:     cfg,
        router:     router,
        workerPool: workerPool,
        server: &http.Server{
            Addr:    addr,
            Handler: router,
        },
    }, nil
}

// Start 启动HTTP服务器
func (s *HTTPServer) Start() error {
    if s.isRunning {
        return nil
    }
    
    s.isRunning = true
    // 启动服务器（非阻塞方式）
    go func() {
        if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            // 记录错误日志
            s.isRunning = false
        }
    }()
    return nil
}

// Stop 优雅停止HTTP服务器
func (s *HTTPServer) Stop() error {
    if !s.isRunning {
        return nil
    }
    
    // 优雅关闭服务器
    err := s.server.Close()
    if err == nil {
        s.isRunning = false
    }
    return err
}
