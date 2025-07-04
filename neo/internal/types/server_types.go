package types

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
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

type TCPServer struct {
	Listener    net.Listener       // 首字母大写导出
	Config      *TCPConfig         // 首字母大写导出
	Metrics     *Metrics           // 首字母大写导出
	Connections *TCPConnectionPool // 首字母大写导出
	Callback    MessageCallback    // 首字母大写导出
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	taskChan    chan func()
	isShutdown  int32 // 原子操作标记服务器状态
}

// Start 开始监听和接受连接
func (s *TCPServer) Start() error {
	address := s.Config.GetAddress()

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %v", address, err)
	}

	s.Listener = listener
	fmt.Printf("TCP server started on %s\n", address)

	s.wg.Add(1)
	go s.acceptLoop()
	return nil
}

// 添加Stop方法
func (s *TCPServer) Stop() error {
	s.cancel() // 先取消上下文
	close(s.taskChan)

	// 给goroutine时间响应
	time.Sleep(100 * time.Millisecond)

	// 再关闭监听器
	if err := s.Listener.Close(); err != nil {
		fmt.Printf("[INFO] 监听器关闭完成: %v\n", err)
	}

	// 关闭所有连接
	s.Connections.Mu.Lock()
	for _, conn := range s.Connections.Connections {
		if !conn.Closed {
			// 设置超时确保关闭操作不会阻塞
			conn.Conn.SetDeadline(time.Now().Add(500 * time.Millisecond))
			conn.Conn.Close()
			conn.Closed = true
		}
	}
	s.Connections.Mu.Unlock()

	// 等待所有goroutine完成
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	// 等待完成或超时
	select {
	case <-done:
		log.Println("TCP服务器已优雅关闭")
	case <-time.After(2 * time.Second):
		log.Println("警告: 服务器关闭超时，可能存在资源泄漏")
	}

	return nil
}

// NewTCPServer 创建TCPServer实例
func NewTCPServer(config *TCPConfig, callback MessageCallback, metrics *Metrics, connections *TCPConnectionPool) *TCPServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &TCPServer{
		Config:      config,
		Callback:    callback,
		Metrics:     metrics,
		Connections: connections,
		ctx:         ctx,
		cancel:      cancel,
		taskChan:    make(chan func(), 100),
		isShutdown:  0,
	}
}

// acceptLoop 持续接受新连接
func (s *TCPServer) acceptLoop() {
	defer s.wg.Done()

	for {
		// 实现接受连接逻辑
	}
}
