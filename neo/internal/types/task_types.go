package types

import (
    "context"
    "fmt"
    "sync"
    "time"

    "github.com/google/uuid"
)

// 任务状态枚举
type TaskStatus int

const (
    TaskStatusPending TaskStatus = iota // 待处理
    TaskStatusSuccess                   // 成功完成
    TaskStatusFailed                    // 执行失败
)

// 异步任务结构体
type AsyncTask struct {
    TaskID     string        // 任务唯一标识（UUID格式）
    Status     TaskStatus    // 当前任务状态
    Result     interface{}   // 任务执行结果
    Error      error         // 错误信息（非nil表示执行失败）
    Callback   Callback      // 回调函数
    CreatedAt  time.Time     // 创建时间（用于超时计算）
    ExpireTime time.Duration // 超时时间（默认30秒）
}

// 任务存储管理器（使用sync.RWMutex保证并发安全）
type TaskManager struct {
    Mu    sync.RWMutex // 导出字段，使用大写开头
    Tasks map[string]*AsyncTask
}

// CreateTask 创建新任务
func (m *TaskManager) CreateTask(callback Callback, expireTime time.Duration) *AsyncTask {
    // 使用UUID生成唯一任务ID（符合neo现有ID生成规范）
    taskID := uuid.New().String()
    task := &AsyncTask{
        TaskID:     taskID,
        Status:     TaskStatusPending,
        Callback:   callback,
        CreatedAt:  time.Now(),
        ExpireTime: expireTime,
    }

    m.Mu.Lock() // 修正为大写Mu
    m.Tasks[taskID] = task
    m.Mu.Unlock() // 修正为大写Mu

    return task
}

// UpdateTaskStatus 更新任务状态
func (m *TaskManager) UpdateTaskStatus(taskID string, status TaskStatus, result interface{}, err error) error {
    m.Mu.RLock() // 修正为大写Mu
    task, exists := m.Tasks[taskID]
    m.Mu.RUnlock() // 修正为大写Mu

    if !exists {
        return fmt.Errorf("任务不存在: %s", taskID)
    }

    // 状态变更校验（防止无效状态转换）
    if !isValidStatusTransition(task.Status, status) {
        return fmt.Errorf("无效状态转换: %v -> %v", task.Status, status)
    }

    m.Mu.Lock() // 修正为大写Mu
    task.Status = status
    task.Result = result
    task.Error = err
    m.Mu.Unlock() // 修正为大写Mu

    // 触发回调函数
    if task.Callback != nil {
        go task.Callback(result, err)
    }

    return nil
}

// 状态转换校验（私有辅助函数）
func isValidStatusTransition(old, new TaskStatus) bool {
    // 只允许从Pending状态转换
    if old != TaskStatusPending {
        return false
    }
    // 目标状态必须是Success或Failed
    return new == TaskStatusSuccess || new == TaskStatusFailed
}

// 任务管理器接口（重命名以避免冲突）
type TaskManagerInterface interface {
    CreateTask(callback Callback, expireTime time.Duration) *AsyncTask
    UpdateTaskStatus(taskID string, status TaskStatus, result interface{}, err error) error
}

// NewTaskManager 创建新的任务管理器实例
func NewTaskManager() *TaskManager {
    tm := &TaskManager{
        Tasks: make(map[string]*AsyncTask),
    }
    // 启动清理工作协程
    go tm.startCleanupWorker()
    return tm
}

// 添加startCleanupWorker方法到TaskManager
func (m *TaskManager) startCleanupWorker() {
    // 使用ticker定期执行清理（每30秒一次，可配置）
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        m.Mu.Lock()
        for taskID, task := range m.Tasks {
            // 检查任务是否已过期
            if time.Since(task.CreatedAt) > task.ExpireTime && task.Status == TaskStatusPending {
                // 更新为失败状态
                task.Status = TaskStatusFailed
                task.Error = fmt.Errorf("任务超时: %s", taskID)
                // 触发超时回调
                if task.Callback != nil {
                    go task.Callback(nil, task.Error)
                }
                // 从map中删除
                delete(m.Tasks, taskID)
            }
        }
        m.Mu.Unlock()
    }
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
