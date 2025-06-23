package transport

import (
	"fmt"
	"neo/internal/config"
	"neo/internal/discovery"
	"neo/internal/ipcprotocol"
	"sync"
)

// 服务注册表（设计文档服务发现机制）
var serviceRegistry = make(map[string]func(map[string]interface{}) (interface{}, error))
var registryLock sync.RWMutex

// 全局协程池
var workerPool *WorkerPool

// 初始化协程池
func init() {
	workerPool = NewWorkerPool(10, 100)
}

// 注册服务（供 Go/Python 服务调用）
func RegisterService(name string, handler func(map[string]interface{}) (interface{}, error)) {
	registryLock.Lock()
	defer registryLock.Unlock()
	serviceRegistry[name] = handler
}

// registerServiceHandler 实现服务注册逻辑
func registerServiceHandler(params map[string]interface{}) (interface{}, error) {
	serviceName, ok := params["service_name"].(string)
	if !ok {
		return nil, fmt.Errorf("缺少或无效的service_name参数")
	}

	// 实际服务注册逻辑将在这里实现
	return map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("服务 %s 注册成功", serviceName),
	}, nil
}

// StartIpcServer 启动 TCP 服务（传输层实现）
func StartIpcServer() error {
	cfg := config.Get()

	// 初始化组件
	// 修复：使用全局workerPool，移除局部变量声明
	metricsCollector := NewMetricsCollector()

	// 使用metricsCollector收集指标
	_ = metricsCollector

	storage := discovery.NewInMemoryStorage()
	discoveryInstance := discovery.New(storage)
	serviceRegistry := NewServiceRegistry(discoveryInstance)

	protocolHandler := NewDefaultProtocolHandler(ipcprotocol.MAGIC_NUMBER, ipcprotocol.ProtocolVersion1)
	errorHandler := &ErrorHandler{}

	connectionHandler := &ConnectionHandler{
		protocol:     protocolHandler,
		registry:     serviceRegistry,
		errorHandler: errorHandler,
	}

	// 注册服务
	serviceRegistry.Register("register", registerServiceHandler)

	// 创建并启动TCP服务器
	tcpServer := NewTCPServer(&cfg.IPC, connectionHandler, workerPool)
	return tcpServer.Start()
}

// 处理连接（协议解析核心）
var discoveryInstance *discovery.Discovery

// 初始化服务发现
func init() {
	workerPool = NewWorkerPool(10, 100)
	// 初始化服务发现（使用内存存储）
	storage := discovery.NewInMemoryStorage()
	discoveryInstance = discovery.New(storage)
	// 注册"register"服务处理函数
	RegisterService("register", registerServiceHandler)
}
