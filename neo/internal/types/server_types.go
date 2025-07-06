package types

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
)

// IPC服务器配置
type IPCServerConfig struct {
	TCPConfig       TCPConfig
	WorkerPoolSize  int
	WorkerQueueSize int
}

// 具体任务实现结构体
type transportTask struct {
	ctx    context.Context
	result chan *TaskResult
}

// 实现Task接口的ID方法
func (t *transportTask) ID() string {
	return ""
}

// 实现Task接口的Execute方法
func (t *transportTask) Execute() (interface{}, error) {
	return nil, nil
}

// WorkerPoolAdapter 工作池适配器：解决WorkerPool接口适配问题
type WorkerPoolAdapter struct {
	WorkerPool WorkerPool
}

// Submit 实现WorkerPool接口的Submit方法
func (a *WorkerPoolAdapter) Submit(task Task) chan TaskResult {
	resultChan := make(chan TaskResult, 1)

	transportTask := &transportTask{
		ctx:    context.Background(),
		result: make(chan *TaskResult, 1),
	}

	go func() {
		defer close(transportTask.result)
		result, err := task.Execute()
		var data []byte
		if result != nil {
			if b, ok := result.([]byte); ok {
				data = b
			} else {
				data = []byte(fmt.Sprintf("%v", result))
			}
		}
		transportTask.result <- &TaskResult{
			Result: data,
			Error:  err,
		}
	}()

	// 使用WorkerPool接口的Submit方法而非SubmitTransportTask
	if err := a.WorkerPool.Submit(transportTask); err != nil {
		resultChan <- TaskResult{
			TaskID: task.ID(),
			Error:  fmt.Errorf("任务提交失败: %v", err),
		}
		close(resultChan)
		return resultChan
	}

	go func() {
		defer close(resultChan)
		select {
		case res := <-transportTask.result:
			resultChan <- TaskResult{
				TaskID: task.ID(),
				Result: res.Result,
				Error:  res.Error,
			}
		case <-transportTask.ctx.Done():
			resultChan <- TaskResult{
				TaskID: task.ID(),
				Error:  transportTask.ctx.Err(),
			}
		}
	}()

	return resultChan
}

// Start 启动工作池
func (a *WorkerPoolAdapter) Start() {
	// 接口中没有Start方法，需要移除或调整设计
}

// Stop 实现WorkerPool接口的Stop方法
func (a *WorkerPoolAdapter) Stop() {
	a.WorkerPool.Stop()
}

// SetWorkerCount 实现WorkerPool接口的SetWorkerCount方法
func (a *WorkerPoolAdapter) SetWorkerCount(count int) {
	a.WorkerPool.SetWorkerCount(count)
}

// Shutdown 实现WorkerPool接口的Shutdown方法
func (a *WorkerPoolAdapter) Shutdown() {
	a.WorkerPool.Shutdown()
}

// IPC服务器
type IPCServer struct {
	// 删除所有未使用的字段
}

// 从全局配置创建IPC服务器配置
func NewIPCServerConfigFromGlobal(globalConfig *GlobalConfig) IPCServerConfig {
	return IPCServerConfig{
		TCPConfig: TCPConfig{
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

// TCPServer 添加处理器工厂属性
type TCPServer struct {
	Listener       net.Listener       // 首字母大写导出
	Config         *TCPConfig         // 首字母大写导出
	Metrics        *Metrics           // 首字母大写导出
	Connections    *TCPConnectionPool // 首字母大写导出
	Callback       MessageCallback    // 首字母大写导出
	wg             sync.WaitGroup
	ctx            context.Context
	cancel         context.CancelFunc
	taskChan       chan func()
	isShutdown     int32             // 原子操作标记服务器状态
	handlerFactory TCPHandlerFactory // 新增：处理器工厂
}

// 新增Start方法实现Server接口
func (s *TCPServer) Start() error {
	// 创建TCP监听器
	listener, err := net.Listen("tcp", s.Config.Address)
	if err != nil {
		return fmt.Errorf("创建TCP监听器失败: %w", err)
	}
	s.Listener = listener

	// 启动接受连接循环
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.acceptLoop()
	}()

	return nil
}

// 添加Stop方法实现资源释放
func (s *TCPServer) Stop() error {
	if s.Listener != nil {
		s.Listener.Close()
	}
	s.cancel()
	s.wg.Wait()
	return nil
}

// 修改acceptLoop使用工厂创建处理器
// 修改acceptLoop使用正确的Listener字段和错误处理
// 修改acceptLoop方法，移除未定义的Metrics.RecordError调用
func (s *TCPServer) acceptLoop() {
	for {
		conn, err := s.Listener.Accept() // 修正为大写Listener
		if err != nil {
			select {
			case <-s.ctx.Done():
				// 正常关闭，不记录错误
				return
			default:
				// 移除未定义的Metrics.RecordError调用
				log.Printf("[ERROR] 接受连接失败: %v", err)
				return
			}
		}
		// 使用工厂创建处理器
		handler := s.handlerFactory(conn)
		go handler.Start()
	}
}

// 修正NewTCPServer函数签名和初始化逻辑
func NewTCPServer(config *TCPConfig, callback MessageCallback, metrics *Metrics, connections *TCPConnectionPool, ctx context.Context, factory TCPHandlerFactory) *TCPServer {
	ctx, cancel := context.WithCancel(ctx)
	return &TCPServer{
		Config:         config,
		Callback:       callback,
		Metrics:        metrics,
		Connections:    connections,
		ctx:            ctx,
		cancel:         cancel,
		taskChan:       make(chan func(), 100),
		isShutdown:     0,
		handlerFactory: factory,
	}
}
