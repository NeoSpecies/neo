package types

import (
	"sync"
	"time"
)

// ServiceHandler 服务处理器接口定义
type ServiceHandler interface {
	Handle(request *Request) (*Response, error)
}

// ServiceHandlerFunc 函数类型实现ServiceHandler接口
type ServiceHandlerFunc func(request *Request) (*Response, error)

// Handle 实现ServiceHandler接口
func (f ServiceHandlerFunc) Handle(request *Request) (*Response, error) {
	return f(request)
}

// ServiceRegistry 服务注册表实现
type ServiceRegistry struct {
	mu       sync.RWMutex
	handlers map[string]ServiceHandler
}

// NewServiceRegistry 创建新的服务注册表
func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		handlers: make(map[string]ServiceHandler),
	}
}

// Register 注册服务处理器
func (r *ServiceRegistry) Register(service string, handler ServiceHandler) {
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
	handler func(*Request) (*Response, error)
}

// Handle 实现ServiceHandler接口
func (h *handlerWrapper) Handle(req *Request) (*Response, error) {
	return h.handler(req)
}

// RegisterFunc 注册服务处理器函数
func (r *ServiceRegistry) RegisterFunc(service string, handler func(*Request) (*Response, error)) {
	r.Register(service, &handlerWrapper{handler: handler})
}

// GetHandler 获取服务处理器
func (r *ServiceRegistry) GetHandler(service string) (ServiceHandler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	handler, exists := r.handlers[service]
	return handler, exists
}

// ServerConfig 服务器配置接口
type ServerConfig interface {
	GetAddress() string
	GetMaxConnections() int
	GetConnectionTimeout() time.Duration
	GetHandlerConfig() interface{}
}

// Server 服务器接口
type Server interface {
	Start() error
	Stop() error
}
