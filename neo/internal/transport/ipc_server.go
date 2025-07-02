package transport

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/prometheus/client_golang/prometheus"

	"neo/internal/connection/tcp"
	"neo/internal/ipcprotocol"
	"neo/internal/types"
)

// 实现common.WorkerPool接口的Submit方法
func (a *WorkerPoolAdapter) Submit(task types.Task) chan types.TaskResult {
	resultChan := make(chan types.TaskResult, 1)

	// 创建transport.Task包装types.Task
	transportTask := &Task{
		Ctx:    context.Background(),
		Result: make(chan *TaskResult, 1),
	}

	// 启动goroutine处理任务
	go func() {
		defer close(resultChan)

		// 执行types.Task的Execute方法
		execResult, execErr := task.Execute()

		// 转换为types.TaskResult
		typesResult := types.TaskResult{
			TaskID: task.ID(),
			Result: execResult,
			Error:  execErr,
		}

		// 发送结果
		resultChan <- typesResult
	}()

	// 提交到工作池
	if err := a.WorkerPool.Submit(transportTask); err != nil {
		resultChan <- types.TaskResult{
			TaskID: task.ID(),
			Error:  fmt.Errorf("任务提交失败: %w", err),
		}
	}

	return resultChan
}

// types.WorkerPool接口的Stop方法
func (a *WorkerPoolAdapter) Stop() {
	a.WorkerPool.Stop()
}

// SetWorkerCount 实现types.WorkerPool接口的SetWorkerCount方法
func (a *WorkerPoolAdapter) SetWorkerCount(count int) {
	a.WorkerPool.SetWorkerCount(count)
}

// Shutdown 实现types.WorkerPool接口的Shutdown方法
func (a *WorkerPoolAdapter) Shutdown() {
	a.WorkerPool.Stop()
}

// IPC服务器
type IPCServer struct {
	config          IPCServerConfig
	tcpServer       types.Server
	serviceRegistry *ServiceRegistry
	workerPool      *WorkerPool
	metrics         *types.Metrics

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu      sync.Mutex
	started bool
}

// 注册服务处理器
func (s *IPCServer) RegisterService(serviceName string, handler types.ServiceHandler) {
	s.serviceRegistry.Register(serviceName, handler)
}

// IPC服务器配置
type IPCServerConfig struct {
	TCPConfig       types.TCPConfig
	WorkerPoolSize  int
	WorkerQueueSize int
}

// 从全局配置创建IPC服务器配置
func NewIPCServerConfigFromGlobal(globalConfig *types.GlobalConfig) IPCServerConfig {
	return IPCServerConfig{
		TCPConfig: types.TCPConfig{
			MaxConnections:    globalConfig.IPC.MaxConnections,
			MaxMsgSize:        globalConfig.Protocol.MaxMessageSize,
			ReadTimeout:       globalConfig.IPC.ReadTimeout,
			WriteTimeout:      globalConfig.IPC.WriteTimeout,
			WorkerCount:       globalConfig.IPC.WorkerCount,
			ConnectionTimeout: globalConfig.IPC.ConnectionTimeout,
		},
		WorkerPoolSize:  10,
		WorkerQueueSize: 100,
	}
}

// WorkerPool适配器：解决*WorkerPool与common.WorkerPool接口不兼容问题
type WorkerPoolAdapter struct {
	WorkerPool *WorkerPool // 首字母大写使其可导出
}

// 创建新的IPC服务器
func NewIPCServer(config IPCServerConfig) (*IPCServer, error) {
	// 初始化服务注册表
	serviceRegistry := NewServiceRegistry()

	// 初始化指标收集器
	registry := prometheus.NewRegistry()
	metricsCollector := types.NewMetrics(registry)

	// 初始化工作池
	workerPool := NewWorkerPool(
		config.WorkerPoolSize,
		config.WorkerQueueSize,
	)

	// 创建工作池适配器
	workerPoolAdaptor := &WorkerPoolAdapter{
		WorkerPool: workerPool,
	}

	// 创建TCP服务器
	var tcpServer types.Server
	// 通过工厂方法创建TCP服务器
	tcpServer, err := createTCPServer(
		&config.TCPConfig,
		serviceRegistry,
		workerPoolAdaptor,
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

// 添加TCP服务器工厂方法
func createTCPServer(config *types.TCPConfig, registry *ServiceRegistry, workerPool types.WorkerPool, _metrics *types.Metrics) (types.Server, error) {
	// 创建消息回调函数
	callback := func(data []byte) ([]byte, error) {
		// 实现消息处理逻辑
		return ipcprotocol.ProcessMessage(data, registry, workerPool)
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
	s.workerPool.Stop()

	// 等待所有goroutine完成
	s.wg.Wait()

	s.started = false
	return nil
}
