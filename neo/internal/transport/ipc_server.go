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

// 本地WorkerPoolAdapter实现，避免在非本地类型上定义方法
type WorkerPoolAdapter struct {
	WorkerPool *WorkerPool
}

// Submit 实现types.WorkerPool接口的Submit方法
func (a *WorkerPoolAdapter) Submit(task types.Task) chan types.TaskResult {
	resultChan := make(chan types.TaskResult, 1)

	// 创建transport.Task实例
	transportTask := &Task{
		Ctx:    context.Background(),
		Result: make(chan *TaskResult, 1),
	}

	// 启动goroutine执行任务
	go func() {
		defer close(transportTask.Result)
		// 执行types.Task的Execute方法
		result, err := task.Execute()
		// 将结果转换为[]byte（根据实际情况调整转换逻辑）
		var data []byte
		if result != nil {
			if b, ok := result.([]byte); ok {
				data = b
			} else {
				data = []byte(fmt.Sprintf("%v", result))
			}
		}
		transportTask.Result <- &TaskResult{
			Data:  data,
			Error: err,
		}
	}()

	// 提交到工作池
	if err := a.WorkerPool.SubmitTransportTask(transportTask); err != nil {
		resultChan <- types.TaskResult{
			TaskID: task.ID(),
			Error:  fmt.Errorf("任务提交失败: %w", err),
		}
		close(resultChan)
		return resultChan
	}

	// 启动goroutine等待结果
	go func() {
		defer close(resultChan)
		select {
		case res := <-transportTask.Result:
			resultChan <- types.TaskResult{
				TaskID: task.ID(),
				Result: res.Data,
				Error:  res.Error,
			}
		case <-transportTask.Ctx.Done():
			resultChan <- types.TaskResult{
				TaskID: task.ID(),
				Error:  transportTask.Ctx.Err(),
			}
		}
	}()

	return resultChan
}

// Start 启动工作池
func (a *WorkerPoolAdapter) Start() {
	a.WorkerPool.Start()
}

// Stop 实现types.WorkerPool接口的Stop方法
func (a *WorkerPoolAdapter) Stop() {
	a.WorkerPool.Stop()
}

// SetWorkerCount 实现types.WorkerPool接口的SetWorkerCount方法
func (a *WorkerPoolAdapter) SetWorkerCount(count int) {
	a.WorkerPool.SetWorkerCount(count)
}

// Shutdown 实现types.WorkerPool接口的Shutdown方法
func (a *WorkerPoolAdapter) Shutdown() {
	a.WorkerPool.Shutdown()
}

// IPCServer 本地IPC服务器实现
type IPCServer struct {
	config          types.IPCServerConfig
	tcpServer       types.Server
	serviceRegistry *ServiceRegistry
	workerPool      *WorkerPoolAdapter // 修改为具体类型而非接口
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
	serviceRegistry := NewServiceRegistry()

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
func createTCPServer(config *types.TCPConfig, registry *ServiceRegistry, workerPool types.WorkerPool) (types.Server, error) {
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
	s.workerPool.Shutdown()

	s.started = false
	return nil
}
