package transport

import (
    "sync"
    "neo/internal/common"
    "neo/internal/ipcprotocol"
)

// ServiceRegistry 服务注册表，实现common.ServiceRegistry接口
type ServiceRegistry struct {
    mu       sync.RWMutex
    handlers map[string]common.ServiceHandler
}

// NewServiceRegistry 创建新的服务注册表
func NewServiceRegistry() *ServiceRegistry {
    return &ServiceRegistry{
        handlers: make(map[string]common.ServiceHandler),
    }
}

// Register 注册服务处理器
func (r *ServiceRegistry) Register(service string, handler common.ServiceHandler) {
    if service == "" {
        panic("服务名称不能为空")
    }
    if handler == nil {
        panic("处理器不能为空")
    }

    r.mu.Lock()
    defer r.mu.Unlock()

    if _, exists := r.handlers[service]; exists {
        panic("服务已注册: " + service)
    }
    r.handlers[service] = handler
}

// RegisterFunc 注册服务处理器函数
func (r *ServiceRegistry) RegisterFunc(service string, handler func(request *ipcprotocol.Request) (*ipcprotocol.Response, error)) {
    r.Register(service, common.ServiceHandlerFunc(handler))
}

// GetHandler 获取服务处理器，实现common.ServiceRegistry接口
func (r *ServiceRegistry) GetHandler(service string) (common.ServiceHandler, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    handler, exists := r.handlers[service]
    return handler, exists
}
