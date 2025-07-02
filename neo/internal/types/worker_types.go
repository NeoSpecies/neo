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

// 工作池配置
type WorkerPoolConfig struct {
	WorkerCount    int
	MaxQueueSize   int
	MaxTaskRetries int
}
