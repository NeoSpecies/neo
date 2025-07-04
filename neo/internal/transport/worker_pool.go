package transport

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"neo/internal/types"
)

// SubmitTransportTask 提交内部任务
type TransportTask struct {
	Ctx    context.Context
	Result chan *types.TaskResult
}

// workerImpl 实现 types.Worker 接口
type workerImpl struct {
	id         int
	workerPool types.WorkerPool
	taskQueue  chan types.Task
	quit       chan struct{}
}

// NewWorker 创建新的工作协程
func NewWorker(id int, workerPool types.WorkerPool) types.Worker {
	return &workerImpl{
		id:         id,
		workerPool: workerPool,
		taskQueue:  make(chan types.Task),
		quit:       make(chan struct{}),
	}
}

// Start 启动工作协程
func (w *workerImpl) Start() {
	go func() {
		for {
			select {
			case task, ok := <-w.taskQueue:
				if !ok {
					return
				}
				// 执行任务
				w.processTask(task)
			case <-w.quit:
				return
			}
		}
	}()
}

// Stop 停止工作协程
func (w *workerImpl) Stop() {
	close(w.quit)
}

// processTask 处理任务
func (w *workerImpl) processTask(task types.Task) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("工作协程 %d 执行任务发生恐慌: %v", w.id, r)
			if taskResultChan, ok := task.(interface{ GetResultChan() chan *types.TaskResult }); ok {
				taskResultChan.GetResultChan() <- &types.TaskResult{Result: nil, Error: r.(error)}
			}
		}
	}()

	// 模拟任务处理
	resultData := []byte(fmt.Sprintf("任务处理完成 by worker %d", w.id))
	if taskResultChan, ok := task.(interface{ GetResultChan() chan *types.TaskResult }); ok {
		taskResultChan.GetResultChan() <- &types.TaskResult{Result: resultData, Error: nil}
	}
}

// Submit 实现 types.Worker 接口的 Submit 方法
func (w *workerImpl) Submit(task types.Task) chan types.TaskResult {
	resultChan := make(chan types.TaskResult, 1)

	go func() {
		// 模拟任务处理
		resultData := []byte(fmt.Sprintf("任务处理完成 by worker %d", w.id))
		resultChan <- types.TaskResult{
			TaskID: task.ID(),
			Result: resultData,
			Error:  nil,
		}
		close(resultChan)
	}()

	return resultChan
}

// workerPoolImpl 实现 types.WorkerPool 接口
type workerPoolImpl struct {
	workerCount int
	workers     []types.Worker
	taskQueue   chan types.Task
	quit        chan struct{}
	running     bool
	wg          sync.WaitGroup
	mu          sync.Mutex
	shutdown    bool
}

// NewWorkerPool 创建新的工作池
func NewWorkerPool(workerCount, queueSize int) types.WorkerPool {
	if workerCount <= 0 {
		workerCount = 10 // 默认工作协程数
	}
	if queueSize <= 0 {
		queueSize = 100 // 默认队列大小
	}

	return &workerPoolImpl{
		workerCount: workerCount,
		taskQueue:   make(chan types.Task, queueSize),
		quit:        make(chan struct{}),
		running:     false,
		wg:          sync.WaitGroup{},
		mu:          sync.Mutex{},
		shutdown:    false,
	}
}

// SetWorkerCount 设置工作协程数量
func (wp *workerPoolImpl) SetWorkerCount(count int) {
	if count <= 0 {
		return
	}

	// 如果工作池未运行，直接更新workerCount
	if !wp.running {
		wp.workerCount = count
		return
	}

	// 如果工作池已运行，这里可以添加动态调整worker数量的逻辑
	// 简化实现：仅更新workerCount字段
	wp.workerCount = count
	log.Printf("工作池worker数量已更新为: %d", count)
}

// Start 启动工作池
func (wp *workerPoolImpl) Start() {
	if wp.running {
		return
	}
	wp.running = true

	// 创建工作协程
	wp.workers = make([]types.Worker, wp.workerCount)
	for i := 0; i < wp.workerCount; i++ {
		worker := NewWorker(i, wp)
		wp.workers[i] = worker
		worker.Start()
	}

	// 启动任务分发协程
	wp.wg.Add(1)
	go wp.dispatch()
}

// Stop 停止工作池
func (wp *workerPoolImpl) Stop() {
	if !wp.running {
		return
	}
	wp.running = false

	// 关闭任务队列
	close(wp.taskQueue)

	// 停止所有工作协程
	for _, worker := range wp.workers {
		worker.Stop()
	}

	// 等待所有工作协程退出
	wp.wg.Wait()
	close(wp.quit)
}

// Submit 提交任务到工作池
func (wp *workerPoolImpl) Submit(task types.Task) chan types.TaskResult {
	resultChan := make(chan types.TaskResult, 1)

	// 将types.Task转换为transport内部任务
	transportTask := &TransportTask{
		Ctx:    context.Background(),
		Result: make(chan *types.TaskResult, 1),
	}

	// 提交transportTask到工作池
	err := wp.submitTransportTask(transportTask)
	if err != nil {
		resultChan <- types.TaskResult{
			TaskID: task.ID(),
			Error:  err,
		}
		close(resultChan)
		return resultChan
	}

	// 监听任务结果
	go func() {
		defer close(resultChan)
		select {
		case res := <-transportTask.Result:
			resultChan <- types.TaskResult{
				TaskID: task.ID(),
				Result: res.Result,
				Error:  res.Error,
			}
		case <-wp.quit:
			resultChan <- types.TaskResult{
				TaskID: task.ID(),
				Error:  errors.New("工作池已停止"),
			}
		}
	}()

	return resultChan
}

// submitTransportTask 处理内部任务
func (wp *workerPoolImpl) submitTransportTask(task *TransportTask) error {
	if !wp.running {
		return errors.New("工作池未运行")
	}

	select {
	case wp.taskQueue <- &taskWrapper{task: task}:
		return nil
	case <-time.After(5 * time.Second):
		return errors.New("任务提交超时")
	case <-wp.quit:
		return errors.New("工作池已停止")
	}
}

// taskWrapper 是 types.Task 接口的包装器
type taskWrapper struct {
	task *TransportTask
}

func (tw *taskWrapper) ID() string {
	return "" // 这里需要根据实际情况实现
}

func (tw *taskWrapper) Execute() (interface{}, error) {
	return nil, nil // 这里需要根据实际情况实现
}

// dispatch 分发任务到工作协程
func (wp *workerPoolImpl) dispatch() {
	defer wp.wg.Done()

	for task := range wp.taskQueue {
		// 找到一个空闲的工作协程
		var worker types.Worker
		for {
			select {
			case <-wp.quit:
				return
			default:
				// 简单的轮询调度
				worker = wp.workers[0]
				wp.workers = append(wp.workers[1:], worker)
				goto foundWorker
			}
		}

	foundWorker:
		// 将任务发送给工作协程
		select {
		case worker.(*workerImpl).taskQueue <- task:
		case <-wp.quit:
			return
		case <-time.After(5 * time.Second):
			log.Printf("任务分发超时")
			if taskResultChan, ok := task.(interface{ GetResultChan() chan *types.TaskResult }); ok {
				taskResultChan.GetResultChan() <- &types.TaskResult{Result: nil, Error: errors.New("任务分发超时")}
			}
		}
	}
}

// 实现Shutdown方法
func (wp *workerPoolImpl) Shutdown() {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.shutdown {
		return
	}

	wp.shutdown = true
	wp.Stop() // 调用现有的Stop方法
}
