package transport

import (
	"neo/internal/types"
	"sync"
)

// ServiceRegistry 服务注册表，实现common.ServiceRegistry接口
type ServiceRegistry struct {
	mu       sync.RWMutex
	handlers map[string]types.ServiceHandler
}

// NewServiceRegistry 创建新的服务注册表
func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		handlers: make(map[string]types.ServiceHandler),
	}
}

// Register 注册服务处理器
func (r *ServiceRegistry) Register(service string, handler types.ServiceHandler) {
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

// handlerWrapper 包装函数以实现ServiceHandler接口
type handlerWrapper struct {
	handler func(*types.Request) (*types.Response, error)
}

// Handle 实现common.ServiceHandler接口
func (h *handlerWrapper) Handle(req *types.Request) (*types.Response, error) {
	return h.handler(req)
}

// RegisterFunc 注册服务处理器函数
func (r *ServiceRegistry) RegisterFunc(service string, handler func(*types.Request) (*types.Response, error)) {
	r.Register(service, &handlerWrapper{handler: handler})
}

// GetHandler 获取服务处理器，实现common.ServiceRegistry接口
func (r *ServiceRegistry) GetHandler(service string) (types.ServiceHandler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	handler, exists := r.handlers[service]
	return handler, exists
}
