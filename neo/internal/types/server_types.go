package types

import (
	"context"
	"fmt"
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
			Error:  fmt.Errorf("任务提交失败: %w", err),
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
