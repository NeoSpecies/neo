package types

import (
	"context"
	"sync"
)

// WorkerPool接口定义 - 移至server_types.go避免循环依赖
type WorkerPool interface {
	Submit(task Task) chan TaskResult
	Stop()
	SetWorkerCount(count int)
	Shutdown()
}

// IPC服务器配置
type IPCServerConfig struct {
	TCPConfig       TCPConfig
	WorkerPoolSize  int
	WorkerQueueSize int
}

// WorkerPool适配器：解决WorkerPool接口适配问题
type WorkerPoolAdapter struct {
	WorkerPool WorkerPool // 修正：使用接口类型而非指针
}

// 实现WorkerPool接口的Submit方法
func (a *WorkerPoolAdapter) Submit(task Task) chan TaskResult {
	resultChan := make(chan TaskResult, 1)

	// 此处保留原有适配逻辑
	go func() {
		defer close(resultChan)
		result, err := task.Execute()
		resultChan <- TaskResult{
			TaskID: task.ID(),
			Result: result,
			Error:  err,
		}
	}()

	return resultChan
}

// 实现WorkerPool接口的Stop方法
func (a *WorkerPoolAdapter) Stop() {
	a.WorkerPool.Stop()
}

// 实现WorkerPool接口的SetWorkerCount方法
func (a *WorkerPoolAdapter) SetWorkerCount(count int) {
	a.WorkerPool.SetWorkerCount(count)
}

// 实现WorkerPool接口的Shutdown方法
func (a *WorkerPoolAdapter) Shutdown() {
	a.WorkerPool.Shutdown()
}

// IPC服务器
type IPCServer struct {
	config          IPCServerConfig
	tcpServer       Server
	serviceRegistry *ServiceRegistry
	workerPool      WorkerPool // 修正：使用接口类型
	metrics         *Metrics

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu      sync.Mutex
	started bool
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
