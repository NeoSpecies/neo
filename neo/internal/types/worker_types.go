package types

import (
    "context"
    "fmt"
)

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
