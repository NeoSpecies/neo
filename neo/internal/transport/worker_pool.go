// 独立协程池文件
package transport

import (
	"sync"
	"time"
)

// 新增Job类型定义
// Job 表示一个可执行的任务
type Job func()

// WorkerPool 协程池结构定义
type WorkerPool struct {
	minWorkers     int
	maxWorkers     int
	currentWorkers int
	jobs           chan Job
	wg             sync.WaitGroup
	mu             sync.Mutex
}

// 动态调整协程数量
func (p *WorkerPool) adjustWorkers() {
	// 实现动态扩缩容逻辑
	p.mu.Lock()
	defer p.mu.Unlock()

	jobCount := len(p.jobs)
	// 如果任务数大于当前工作协程数且未达到最大限制，则增加工作协程
	if jobCount > p.currentWorkers && p.currentWorkers < p.maxWorkers {
		// 计算需要增加的协程数，最多增加到maxWorkers
		needAdd := jobCount - p.currentWorkers
		if needAdd > p.maxWorkers-p.currentWorkers {
			needAdd = p.maxWorkers - p.currentWorkers
		}
		for i := 0; i < needAdd; i++ {
			p.startWorker()
			p.currentWorkers++
		}
	} else if jobCount == 0 && p.currentWorkers > p.minWorkers {
		// 如果没有任务且当前工作协程数大于最小限制，则减少到minWorkers
		p.currentWorkers = p.minWorkers
	}
}

// NewWorkerPool 创建新的协程池
func NewWorkerPool(minWorkers, maxWorkers int) *WorkerPool {
	if minWorkers <= 0 {
		minWorkers = 5 // 默认最小工作协程数
	}
	if maxWorkers <= minWorkers {
		maxWorkers = minWorkers * 2 // 确保最大工作协程数大于最小
	}

	wp := &WorkerPool{
		minWorkers:     minWorkers,
		maxWorkers:     maxWorkers,
		currentWorkers: 0,
		jobs:           make(chan Job, maxWorkers*2), // 缓冲区大小为最大工作协程数的2倍
		wg:             sync.WaitGroup{},
		mu:             sync.Mutex{},
	}

	// 启动初始工作协程
	for i := 0; i < minWorkers; i++ {
		wp.startWorker()
		wp.currentWorkers++
	}

	// 启动后台调整协程
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			wp.adjustWorkers()
		}
	}()

	return wp
}

// Submit 提交任务到协程池
func (wp *WorkerPool) Submit(job Job) {
	wp.jobs <- job
}

// startWorker 启动一个工作协程
func (wp *WorkerPool) startWorker() {
	wp.wg.Add(1)
	go func() {
		defer wp.wg.Done()
		for job := range wp.jobs {
			job() // 执行任务
		}
	}()
}

// Stop 停止协程池并等待所有任务完成
func (wp *WorkerPool) Stop() {
	close(wp.jobs) // 关闭任务通道
	wp.wg.Wait()  // 等待所有工作协程完成
}
