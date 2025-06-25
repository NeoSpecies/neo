package types

import (
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

// ServiceRegistry 服务注册接口
type ServiceRegistry interface {
	Register(service string, handler ServiceHandler)
	RegisterFunc(service string, handler func(*Request) (*Response, error))
	GetHandler(service string) (ServiceHandler, bool)
}

// WorkerPool 工作池接口
type WorkerPool interface {
	Submit(task func()) error
	Stop()
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