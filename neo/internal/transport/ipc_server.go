package transport

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"neo/internal/common"
	"neo/internal/config"
	"neo/internal/connection"
	"neo/internal/connection/tcp"
	"neo/internal/ipcprotocol"
	"neo/internal/metrics"
)

// IPC服务器配置
type IPCServerConfig struct {
	TCPConfig       tcp.ServerConfig
	WorkerPoolSize  int
	WorkerQueueSize int
}

// 从全局配置创建IPC服务器配置
func NewIPCServerConfigFromGlobal(globalConfig *config.GlobalConfig) IPCServerConfig {
	// 创建默认连接配置
	defaultConnConfig := &connection.Config{
		ConnectTimeout: 5 * time.Second,
		IdleTimeout:    30 * time.Second,
	}

	return IPCServerConfig{
		TCPConfig: tcp.ServerConfig{
			Address:           fmt.Sprintf("%s:%d", globalConfig.IPC.Host, globalConfig.IPC.Port),
			MaxConnections:    globalConfig.IPC.MaxConnections,
			ConnectionTimeout: time.Duration(globalConfig.Pool.IdleTimeout) * time.Second,
			HandlerConfig:     defaultConnConfig,
		},
		WorkerPoolSize:  10,
		WorkerQueueSize: 100,
	}
}

// WorkerPool适配器：解决*WorkerPool与common.WorkerPool接口不兼容问题
type workerPoolAdapter struct {
	workerPool *WorkerPool
}

// 实现common.WorkerPool接口的Submit方法
func (a *workerPoolAdapter) Submit(taskFunc func()) error {
	// 创建transport.Task实例
	task := &Task{
		Ctx:    context.Background(),
		Result: make(chan *TaskResult, 1),
	}

	// 将任务函数包装到Task的处理逻辑中
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("任务执行发生恐慌: %v", r)
				task.Result <- &TaskResult{Error: fmt.Errorf("任务执行恐慌: %v", r)}
			}
		}()

		// 执行任务函数
		taskFunc()
		// 发送成功结果
		task.Result <- &TaskResult{Data: []byte("任务执行成功")}
	}()

	// 提交transport.Task到工作池
	return a.workerPool.Submit(task)
}

// 实现common.WorkerPool接口的Stop方法
func (a *workerPoolAdapter) Stop() {
	a.workerPool.Stop()
}

// IPC服务器
type IPCServer struct {
	config          IPCServerConfig
	tcpServer       common.Server
	serviceRegistry *ServiceRegistry
	workerPool      *WorkerPool
	metrics         *metrics.Metrics

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu      sync.Mutex
	started bool
}

// 创建新的IPC服务器
func NewIPCServer(config IPCServerConfig) (*IPCServer, error) {
	// 初始化服务注册表
	serviceRegistry := NewServiceRegistry()

	// 初始化指标收集器
	metricsCollector := metrics.NewMetrics()

	// 初始化工作池
	workerPool := NewWorkerPool(
		config.WorkerPoolSize,
		config.WorkerQueueSize,
	)

	// 创建工作池适配器（关键修复点）
	workerPoolAdaptor := &workerPoolAdapter{
		workerPool: workerPool,
	}

	// 创建TCP服务器 - 使用接口类型而非具体实现
	var tcpServer common.Server
	if config.TCPConfig.HandlerConfig == nil {
		config.TCPConfig.HandlerConfig = &connection.Config{}
	}
	// 通过工厂方法创建TCP服务器，传递适配器而非原始workerPool
	tcpServer, err := createTCPServer(
		&config.TCPConfig,
		serviceRegistry,
		workerPoolAdaptor, // 使用适配器解决接口兼容问题
		metricsCollector,
	)
	if err != nil {
		return nil, fmt.Errorf("创建TCP服务器失败: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &IPCServer{
		config:          config,
		tcpServer:       tcpServer,
		serviceRegistry: serviceRegistry,
		workerPool:      workerPool,
		metrics:         metricsCollector,
		ctx:             ctx,
		cancel:          cancel,
	}, nil
}

// 添加TCP服务器工厂方法，避免直接导入tcp包
func createTCPServer(config common.ServerConfig, registry common.ServiceRegistry, workerPool common.WorkerPool, metrics *metrics.Metrics) (common.Server, error) {
	// 类型断言，将common.ServerConfig转换为*tcp.ServerConfig
	tcpConfig, ok := config.(*tcp.ServerConfig)
	if !ok {
		return nil, errors.New("invalid server config type")
	}
	return tcp.NewServer(tcpConfig, registry, workerPool, metrics)
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
	s.workerPool.Stop()

	// 等待所有协程退出
	s.wg.Wait()

	s.started = false
	log.Println("IPC服务器已成功停止")
	return nil
}

// 注册服务处理器
func (s *IPCServer) RegisterService(name string, handler common.ServiceHandler) {
	s.serviceRegistry.Register(name, handler)
	log.Printf("已注册服务: %s", name)
}

// 注册服务处理器函数
func (s *IPCServer) RegisterServiceFunc(name string, handler func(request *ipcprotocol.Request) (*ipcprotocol.Response, error)) {
	s.serviceRegistry.RegisterFunc(name, handler)
	log.Printf("已注册服务函数: %s", name)
}

// 获取服务注册表
func (s *IPCServer) ServiceRegistry() *ServiceRegistry {
	return s.serviceRegistry
}

// 获取指标收集器
func (s *IPCServer) Metrics() *metrics.Metrics {
	return s.metrics
}

// 获取TCP服务器
func (s *IPCServer) TCPServer() common.Server {
	return s.tcpServer
}

// 获取工作池
func (s *IPCServer) WorkerPool() *WorkerPool {
	return s.workerPool
}

// 判断服务器是否已启动
func (s *IPCServer) IsStarted() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.started
}

// 初始化TCP服务器
func (s *IPCServer) initTCPServer() error {
	// 创建工作池适配器
	workerPoolAdaptor := &workerPoolAdapter{
		workerPool: s.workerPool,
	}
	// 使用适配器作为WorkerPool参数
	tcpServer, err := tcp.NewServer(&s.config.TCPConfig, s.serviceRegistry, workerPoolAdaptor, s.metrics)
	if err != nil {
		return err
	}

	s.tcpServer = tcpServer
	return nil
}
