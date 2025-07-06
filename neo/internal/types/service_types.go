/*
 * 描述: 定义服务处理相关的核心接口和结构体，包括服务处理器、服务注册表和服务器接口
 * 作者: Cogito
 * 日期: 2025-06-18
 * 联系方式: neospecies@outlook.com
 */
package types

import (
	"sync"
	"time"
)

// ServiceHandler 服务处理器接口定义
// 所有服务必须实现此接口以处理请求
// +----------------+-----------------------------------+
// | 方法名         | 描述                               |
// +----------------+-----------------------------------+
// | Handle         | 处理请求并返回响应                  |
// +----------------------------------------------------+

type ServiceHandler interface {
	Handle(request *Request) (*Response, error)
}

// ServiceHandlerFunc 函数类型实现ServiceHandler接口
// 允许将普通函数作为服务处理器
// +----------------+-----------------------------------+
// | 参数           | 描述                               |
// +----------------+-----------------------------------+
// | request        | 请求对象                           |
// | 返回值          | 响应对象和可能的错误                |
// +----------------+-----------------------------------+
type ServiceHandlerFunc func(request *Request) (*Response, error)

// Handle 实现ServiceHandler接口
// 调用函数本身处理请求
// +----------------+-----------------------------------+
// | 参数           | 描述                              |
// +----------------+-----------------------------------+
// | request        | 请求对象                          |
// | 返回值         | 响应对象和可能的错误              |
// +----------------+-----------------------------------+
func (f ServiceHandlerFunc) Handle(request *Request) (*Response, error) {
	return f(request)
}

// ServiceRegistry 服务注册表实现
// 用于注册和管理服务处理器的并发安全容器
// +----------------+------------------+-----------------------------------+
// | 字段名         | 类型             | 描述                              |
// +----------------+------------------+-----------------------------------+
// | mu             | sync.RWMutex     | 并发安全读写锁                    |
// | handlers       | map[string]ServiceHandler | 服务处理器映射表            |
// +----------------+------------------+-----------------------------------+
type ServiceRegistry struct {
	mu       sync.RWMutex
	handlers map[string]ServiceHandler
}

// NewServiceRegistry 创建新的服务注册表
// 初始化空的服务处理器映射表
// +----------------+-----------------------------------+
// | 返回值         | 初始化后的ServiceRegistry实例     |
// +----------------+-----------------------------------+
func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		handlers: make(map[string]ServiceHandler),
	}
}

// Register 注册服务处理器
// 将服务名称与处理器关联并存储
// +----------------+-----------------------------------+
// | 参数           | 描述                              |
// +----------------+-----------------------------------+
// | service        | 服务名称                          |
// | handler        | 服务处理器实例                    |
// +----------------+-----------------------------------+
// 注意: 如果服务名称已存在或处理器为nil，将触发panic
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
// 适配普通函数作为服务处理器
// +----------------+------------------+-----------------------------------+
// | 字段名         | 类型             | 描述                              |
// +----------------+------------------+-----------------------------------+
// | handler        | func(*Request)   | 实际处理请求的函数                |
// |                | (*Response, error)|                                   |
// +----------------+------------------+-----------------------------------+
type handlerWrapper struct {
	handler func(*Request) (*Response, error)
}

// Handle 实现ServiceHandler接口
// 调用包装的函数处理请求
// +----------------+-----------------------------------+
// | 参数           | 描述                              |
// +----------------+-----------------------------------+
// | req            | 请求对象                          |
// | 返回值         | 响应对象和可能的错误              |
// +----------------+-----------------------------------+
func (h *handlerWrapper) Handle(req *Request) (*Response, error) {
	return h.handler(req)
}

// RegisterFunc 注册服务处理器函数
// 便捷方法，直接注册普通函数作为服务处理器
// +----------------+-----------------------------------+
// | 参数           | 描述                              |
// +----------------+-----------------------------------+
// | service        | 服务名称                          |
// | handler        | 请求处理函数                      |
// +----------------+-----------------------------------+
func (r *ServiceRegistry) RegisterFunc(service string, handler func(*Request) (*Response, error)) {
	r.Register(service, &handlerWrapper{handler: handler})
}

// GetHandler 获取服务处理器
// 根据服务名称查找并返回注册的处理器
// +----------------+-----------------------------------+
// | 参数           | 描述                              |
// +----------------+-----------------------------------+
// | service        | 服务名称                          |
// | 返回值         | 服务处理器实例和存在标志          |
// +----------------+-----------------------------------+
func (r *ServiceRegistry) GetHandler(service string) (ServiceHandler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	handler, exists := r.handlers[service]
	return handler, exists
}

// ServerConfig 服务器配置接口
// 定义服务器配置信息的访问方法
// +----------------+-----------------------------------+
// | 方法名         | 描述                              |
// +----------------+-----------------------------------+
// | GetAddress     | 获取服务器地址                    |
// | GetMaxConnections | 获取最大连接数限制            |
// | GetConnectionTimeout | 获取连接超时时间            |
// | GetHandlerConfig | 获取处理器配置              |
// +----------------+-----------------------------------+
type ServerConfig interface {
	GetAddress() string
	GetMaxConnections() int
	GetConnectionTimeout() time.Duration
	GetHandlerConfig() interface{}
}

// Server 服务器接口
// 定义服务器的基本生命周期管理方法
// +----------------+-----------------------------------+
// | 方法名         | 描述                              |
// +----------------+-----------------------------------+
// | Start          | 启动服务器                        |
// | Stop           | 停止服务器                        |
// +----------------+-----------------------------------+
type Server interface {
	Start() error
	Stop() error
}
