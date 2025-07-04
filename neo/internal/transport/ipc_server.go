package transport

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"

	"neo/internal/connection/tcp"
	"neo/internal/ipcprotocol"
	"neo/internal/types"
)

// 删除原WorkerPoolAdapter定义

// IPCServer 本地IPC服务器实现
type IPCServer struct {
	config          types.IPCServerConfig
	tcpServer       types.Server
	serviceRegistry *types.ServiceRegistry
	workerPool      *types.WorkerPoolAdapter
	metrics         *types.Metrics
	ctx             context.Context
	cancel          context.CancelFunc
	started         bool
	mu              sync.Mutex
	wg              sync.WaitGroup
}

// RegisterService 注册服务处理器
func (s *IPCServer) RegisterService(serviceName string, handler types.ServiceHandler) {
	s.serviceRegistry.Register(serviceName, handler)
}

// 创建新的IPC服务器
func NewIPCServer(config types.IPCServerConfig) (*IPCServer, error) {
	// 初始化服务注册表
	serviceRegistry := types.NewServiceRegistry() // 使用types包的ServiceRegistry

	// 初始化工作池
	workerPool := NewWorkerPool(
		config.WorkerPoolSize,
		config.WorkerQueueSize,
	)

	// 创建工作池适配器
	workerPoolAdaptor := &types.WorkerPoolAdapter{ // 使用types包的WorkerPoolAdapter
		WorkerPool: workerPool,
	}

	// 创建TCP服务器
	var tcpServer types.Server
	// 通过工厂方法创建TCP服务器
	tcpServer, err := createTCPServer(
		&config.TCPConfig,
		serviceRegistry,
		workerPoolAdaptor,
	)
	if err != nil {
		return nil, fmt.Errorf("创建TCP服务器失败: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &IPCServer{
		config:          config,
		tcpServer:       tcpServer,
		serviceRegistry: serviceRegistry,
		workerPool:      workerPoolAdaptor,
		ctx:             ctx,
		cancel:          cancel,
	}, nil
}

// 添加TCP服务器工厂方法
func createTCPServer(config *types.TCPConfig, registry *types.ServiceRegistry, workerPool types.WorkerPool) (types.Server, error) {  // 更新参数类型
	// 创建消息回调函数
	callback := func(data []byte) ([]byte, error) {
		// 实现消息处理逻辑
		return ipcprotocol.ProcessMessage(data, registry, workerPool) // 移除*解引用操作符
	}
	return tcp.NewServer(config, callback)
}

// 启动服务器
func (s *IPCServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return errors.New("服务器已启动")
	}

	log.Println("启动IPC服务器...")

	// 启动工作池
	s.workerPool.Start()

	// 启动TCP服务器
	if err := s.tcpServer.Start(); err != nil {
		s.workerPool.Stop()
		return fmt.Errorf("启动TCP服务器失败: %w", err)
	}

	s.started = true
	log.Println("IPC服务器启动成功")
	return nil
}

// 停止服务器
func (s *IPCServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return errors.New("服务器未启动")
	}

	log.Println("停止IPC服务器...")

	// 取消上下文
	s.cancel()

	// 停止TCP服务器
	if err := s.tcpServer.Stop(); err != nil {
		log.Printf("停止TCP服务器时出错: %v", err)
	}

	// 停止工作池
	s.workerPool.Shutdown()

	s.started = false
	return nil
}

// IPC服务器配置
type IPCServerConfig struct {
    TCPConfig       types.TCPConfig  // 更新引用
    WorkerPoolSize  int
    WorkerQueueSize int
}
