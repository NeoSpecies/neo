package transport

import (
	"neo/internal/discovery"
	"sync"
)

// ServiceRegistry 专注于服务注册与发现
type ServiceRegistry struct {
	services  map[string]func(map[string]interface{}) (interface{}, error)
	mu        sync.RWMutex
	discovery *discovery.Discovery
}

// NewServiceRegistry 创建新的服务注册表
func NewServiceRegistry(discovery *discovery.Discovery) *ServiceRegistry {
	return &ServiceRegistry{
		services:  make(map[string]func(map[string]interface{}) (interface{}, error)),
		discovery: discovery,
	}
}

// Register 注册服务处理函数
func (r *ServiceRegistry) Register(name string, handler func(map[string]interface{}) (interface{}, error)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.services[name] = handler
}

// GetHandler 获取服务处理函数
func (r *ServiceRegistry) GetHandler(name string) (func(map[string]interface{}) (interface{}, error), bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	handler, exists := r.services[name]
	return handler, exists
}
