package transport

import (
	"context"
	"errors"
	"log"
	"time"
)

// Task 表示工作池中的任务
type Task struct {
	Ctx        context.Context
	Service    string
	Method     string
	Payload    []byte
	Connection interface{}
	Result     chan *TaskResult
}

// TaskResult 表示任务执行结果
type TaskResult struct {
	Data  []byte
	Error error
}

// Worker 工作协程
type Worker struct {
	ID          int
	WorkerPool  chan chan *Task
	TaskChannel chan *Task
	quit        chan struct{}
}

// WorkerPool 工作池
type WorkerPool struct {
	WorkerCount      int
	WorkerPool       chan chan *Task
	TaskQueue        chan *Task
	quit             chan struct{}
	metricsCollector *MetricsCollector
}

// NewWorker 创建新的工作协程
func NewWorker(id int, workerPool chan chan *Task) *Worker {
	return &Worker{
		ID:          id,
		WorkerPool:  workerPool,
		TaskChannel: make(chan *Task),
		quit:        make(chan struct{}),
	}
}

// Start 启动工作协程
func (w *Worker) Start() {
	go func() {
		for {
			// 将当前工作协程的任务通道注册到工作池
			w.WorkerPool <- w.TaskChannel

			select {
			case task := <-w.TaskChannel:
				// 执行任务
				result := &TaskResult{}
				var err error
				startTime := time.Now()

				// 模拟任务处理
				// 实际应用中这里应该是具体的业务逻辑处理
				result.Data, err = w.processTask(task)
				result.Error = err

				// 记录指标
				if task.Ctx != nil && task.Service != "" && task.Method != "" {
					metricsCollector := NewMetricsCollector()
					metricsCollector.CollectResponse(task.Ctx, task.Service, task.Method, startTime, err)
				}

				// 将结果发送回任务提交者
				select {
				case task.Result <- result:
				case <-time.After(5 * time.Second):
					log.Printf("任务结果发送超时，任务可能已取消")
				}

			case <-w.quit:
				// 退出工作协程
				return
			}
		}
	}()
}

// processTask 处理具体任务
func (w *Worker) processTask(task *Task) ([]byte, error) {
	// 这里应该是实际的任务处理逻辑
	// 示例：简单休眠模拟处理时间
	time.Sleep(10 * time.Millisecond)

	// 模拟1%的错误率
	if time.Now().UnixNano()%100 < 1 {
		return nil, errors.New("模拟任务处理错误")
	}

	return []byte("任务处理结果"), nil
}

// Stop 停止工作协程
func (w *Worker) Stop() {
	close(w.quit)
}

// NewWorkerPool 创建新的工作池
func NewWorkerPool(workerCount int, queueSize int) *WorkerPool {
	pool := make(chan chan *Task, workerCount)
	queue := make(chan *Task, queueSize)

	return &WorkerPool{
		WorkerCount:      workerCount,
		WorkerPool:       pool,
		TaskQueue:        queue,
		quit:             make(chan struct{}),
		metricsCollector: NewMetricsCollector(),
	}
}

// Start 启动工作池
func (wp *WorkerPool) Start() {
	// 创建并启动工作协程
	for i := 0; i < wp.WorkerCount; i++ {
		worker := NewWorker(i+1, wp.WorkerPool)
		worker.Start()
	}

	// 启动任务调度协程
	go wp.dispatch()
}

// dispatch 任务调度
func (wp *WorkerPool) dispatch() {
	for {
		select {
		case task := <-wp.TaskQueue:
			// 获取一个工作协程的任务通道
			workerTaskChannel := <-wp.WorkerPool

			// 将任务发送给工作协程
			go func(task *Task) {
				workerTaskChannel <- task
			}(task)

		case <-wp.quit:
			// 停止所有工作协程
			for i := 0; i < wp.WorkerCount; i++ {
				workerTaskChannel := <-wp.WorkerPool
				close(workerTaskChannel)
			}
			close(wp.WorkerPool)
			return
		}
	}
}

// Submit 提交任务到工作池
func (wp *WorkerPool) Submit(task *Task) error {
	select {
	case wp.TaskQueue <- task:
		// 记录请求指标
		if task.Ctx != nil && task.Service != "" && task.Method != "" {
			startTime := wp.metricsCollector.CollectRequest(task.Ctx, task.Service, task.Method)
			// 这里可以记录开始时间，但实际指标收集在任务完成时进行
			_ = startTime // 避免未使用变量错误
		}
		return nil
	case <-time.After(5 * time.Second):
		return errors.New("任务队列已满，提交任务超时")
	}
}

// Stop 停止工作池
func (wp *WorkerPool) Stop() {
	close(wp.quit)
}
