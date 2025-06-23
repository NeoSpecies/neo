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
    minWorkers    int
    maxWorkers    int
    currentWorkers int
    jobs          chan Job
    wg            sync.WaitGroup
    mu            sync.Mutex
}

// 动态调整协程数量
func (p *WorkerPool) adjustWorkers() {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        p.mu.Lock()
        jobCount := len(p.jobs)
        
        // 根据队列长度动态扩缩容
        if jobCount > p.currentWorkers && p.currentWorkers < p.maxWorkers {
            // 增加工作协程
        } else if jobCount == 0 && p.currentWorkers > p.minWorkers {
            // 减少工作协程
        }
        p.mu.Unlock()
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
        minWorkers:    minWorkers,
        maxWorkers:    maxWorkers,
        currentWorkers: 0,
        jobs:          make(chan Job, maxWorkers*2), // 缓冲区大小为最大工作协程数的2倍
        wg:            sync.WaitGroup{},
        mu:            sync.Mutex{},
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