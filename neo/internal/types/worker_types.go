/*
 * 描述: 定义工作池和任务处理相关的核心接口与结构体，包括任务接口、工作者接口、工作池接口及其适配器实现
 * 作者: Cogito
 * 日期: 2025-06-18
 * 联系方式: neospecies@outlook.com
 */
package types

import (
	"context"
	"fmt"
)

// Task 任务接口定义
// 所有可提交到工作池的任务必须实现此接口
// +----------------+-----------------------------------+
// | 方法名         | 描述                              |
// +----------------+-----------------------------------+
// | ID             | 返回任务唯一标识符                |
// | Execute        | 执行任务并返回结果和可能的错误    |
// +----------------+-----------------------------------+
type Task interface {
	ID() string
	Execute() (interface{}, error)
}

// TaskResult 任务结果结构体
// 封装任务执行后的结果信息
// +----------------+------------------+-----------------------------------+
// | 字段名         | 类型             | 描述                              |
// +----------------+------------------+-----------------------------------+
// | TaskID         | string           | 任务唯一标识符                    |
// | Result         | interface{}      | 任务执行结果                      |
// | Error          | error            | 执行过程中发生的错误（非nil表示失败） |
// +----------------+------------------+-----------------------------------+
type TaskResult struct {
	TaskID string
	Result interface{}
	Error  error
}

// Worker 工作者接口
// 定义单个工作者的基本操作
// +----------------+-----------------------------------+
// | 方法名         | 描述                              |
// +----------------+-----------------------------------+
// | Start          | 启动工作者，开始处理任务          |
// | Stop           | 停止工作者，释放相关资源          |
// | Submit         | 提交任务到工作者处理              |
// +----------------+-----------------------------------+
type Worker interface {
	Start()
	Stop()
	Submit(task Task) chan TaskResult
}

// WorkerPool 工作池接口定义
// 所有工作池实现必须遵循此接口
// 2025.06.18新增，解决跨包类型引用问题
// +----------------+-----------------------------------+
// | 方法名         | 描述                              |
// +----------------+-----------------------------------+
// | Submit         | 提交任务到工作池                  |
// | Stop           | 停止工作池                        |
// | SetWorkerCount | 设置工作者数量                    |
// | Shutdown       | 优雅关闭工作池，释放所有资源      |
// +----------------+-----------------------------------+
type WorkerPool interface {
	Submit(task Task) chan TaskResult
	Stop()
	SetWorkerCount(count int)
	Shutdown()
}

// WorkerPoolConfig 工作池配置
// 定义工作池的基本参数
// +----------------+------------------+-----------------------------------+
// | 字段名         | 类型             | 描述                              |
// +----------------+------------------+-----------------------------------+
// | WorkerCount    | int              | 工作者数量                        |
// | MaxQueueSize   | int              | 任务队列最大容量                  |
// | MaxTaskRetries | int              | 任务最大重试次数                  |
// +----------------+------------------+-----------------------------------+
type WorkerPoolConfig struct {
	WorkerCount    int
	MaxQueueSize   int
	MaxTaskRetries int
}

// WorkerPoolAdapter 工作池适配器
// 解决WorkerPool接口适配问题，转换任务格式并处理结果
// +----------------+------------------+-----------------------------------+
// | 字段名         | 类型             | 描述                              |
// +----------------+------------------+-----------------------------------+
// | WorkerPool     | WorkerPool       | 实际工作池实例                    |
// +----------------+------------------+-----------------------------------+
type WorkerPoolAdapter struct {
	WorkerPool WorkerPool
}

// Submit 实现WorkerPool接口的Submit方法
// 将任务转换为传输格式并提交到工作池，处理结果转换
// +----------------+------------------+-----------------------------------+
// | 参数           | 类型             | 描述                              |
// +----------------+------------------+-----------------------------------+
// | task           | Task             | 要提交的任务实例                  |
// | 返回值         | chan TaskResult  | 用于接收任务结果的通道            |
// +----------------+------------------+-----------------------------------+
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
