package types

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

// 工作池接口定义
// 所有工作池实现必须遵循此接口
// 2025.06.18新增，解决跨包类型引用问题
type WorkerPool interface {
	Submit(task Task) chan TaskResult
	Stop()
	SetWorkerCount(count int)
	Shutdown()
}

// 工作池配置
type WorkerPoolConfig struct {
	WorkerCount    int
	MaxQueueSize   int
	MaxTaskRetries int
}
