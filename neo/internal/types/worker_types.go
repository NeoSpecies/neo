package types

// 删除以下未使用的导入
// import (
// "sync"
// "time"
// )

// 任务接口定义
type Task interface {
	ID() string
	Execute() (interface{}, error)
}

// 任务结果结构体
type TaskResult struct {
	TaskID string
	Result interface{}
	Error  error
}

// 工作者接口
type Worker interface {
	Start()
	Stop()
	Submit(task Task) chan TaskResult
}

// 工作池配置
type WorkerPoolConfig struct {
	WorkerCount    int
	MaxQueueSize   int
	MaxTaskRetries int
}

// 工作池接口
type WorkerPool interface {
	Submit(task Task) chan TaskResult
	SetWorkerCount(count int)
	Shutdown()
}
