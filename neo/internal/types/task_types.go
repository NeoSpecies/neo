/*
 * 描述: 定义任务管理相关的核心类型和接口，包括任务状态枚举、异步任务结构体、任务管理器及任务清理机制
 * 作者: Cogito
 * 日期: 2025-06-18
 * 联系方式: neospecies@outlook.com
 */
package types

import (
    "context"
    "fmt"
    "sync"
    "time"

    "github.com/google/uuid"
)

// TaskStatus 任务状态枚举
// 定义任务在生命周期中的不同状态
// +---------------------+-----------------------------------+
// | 常量名              | 描述                              |
// +---------------------+-----------------------------------+
// | TaskStatusPending   | 待处理：任务已创建但尚未执行      |
// | TaskStatusSuccess   | 成功完成：任务执行成功            |
// | TaskStatusFailed    | 执行失败：任务执行过程中发生错误  |
// +---------------------+-----------------------------------+
type TaskStatus int

const (
    TaskStatusPending TaskStatus = iota // 待处理
    TaskStatusSuccess                   // 成功完成
    TaskStatusFailed                    // 执行失败
)

// AsyncTask 异步任务结构体
// 表示一个异步执行的任务，包含任务状态、结果、回调等信息
// +----------------+------------------+-----------------------------------+
// | 字段名         | 类型             | 描述                              |
// +----------------+------------------+-----------------------------------+
// | TaskID         | string           | 任务唯一标识（UUID格式）          |
// | Status         | TaskStatus       | 当前任务状态                      |
// | Result         | interface{}      | 任务执行结果                      |
// | Error          | error            | 错误信息（非nil表示执行失败）      |
// | Callback       | Callback         | 任务完成后的回调函数              |
// | CreatedAt      | time.Time        | 创建时间（用于超时计算）          |
// | ExpireTime     | time.Duration    | 超时时间（默认30秒）              |
// +----------------+------------------+-----------------------------------+
type AsyncTask struct {
    TaskID     string        // 任务唯一标识（UUID格式）
    Status     TaskStatus    // 当前任务状态
    Result     interface{}   // 任务执行结果
    Error      error         // 错误信息（非nil表示执行失败）
    Callback   Callback      // 回调函数
    CreatedAt  time.Time     // 创建时间（用于超时计算）
    ExpireTime time.Duration // 超时时间（默认30秒）
}

// TaskManager 任务存储管理器
// 负责异步任务的创建、状态更新和过期清理，使用sync.RWMutex保证并发安全
// +----------------+------------------+-----------------------------------+
// | 字段名         | 类型             | 描述                              |
// +----------------+------------------+-----------------------------------+
// | Mu             | sync.RWMutex     | 读写锁，确保并发安全              |
// | Tasks          | map[string]*AsyncTask | 任务存储映射表，键为任务ID     |
// +----------------+------------------+-----------------------------------+
type TaskManager struct {
    Mu    sync.RWMutex // 导出字段，使用大写开头
    Tasks map[string]*AsyncTask
}

// CreateTask 创建新任务
// 生成具有唯一ID的异步任务并添加到管理器中
// +----------------+------------------+-----------------------------------+
// | 参数           | 类型             | 描述                              |
// +----------------+------------------+-----------------------------------+
// | callback       | Callback         | 任务完成后的回调函数              |
// | expireTime     | time.Duration    | 任务超时时间                      |
// | 返回值         | *AsyncTask       | 新创建的异步任务实例              |
// +----------------+------------------+-----------------------------------+
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
// 修改指定任务的状态，并在状态变更后触发回调函数
// +----------------+------------------+-----------------------------------+
// | 参数           | 类型             | 描述                              |
// +----------------+------------------+-----------------------------------+
// | taskID         | string           | 任务唯一标识符                    |
// | status         | TaskStatus       | 新的任务状态                      |
// | result         | interface{}      | 任务执行结果                      |
// | err            | error            | 任务执行过程中发生的错误          |
// | 返回值         | error            | 操作结果，如任务不存在或状态转换无效时返回错误 |
// +----------------+------------------+-----------------------------------+
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

// isValidStatusTransition 状态转换校验
// 检查任务状态转换是否合法，只允许从Pending状态转换到Success或Failed
// +----------------+------------------+-----------------------------------+
// | 参数           | 类型             | 描述                              |
// +----------------+------------------+-----------------------------------+
// | old            | TaskStatus       | 原状态                            |
// | new            | TaskStatus       | 新状态                            |
// | 返回值         | bool             | 状态转换是否有效                  |
// +----------------+------------------+-----------------------------------+
func isValidStatusTransition(old, new TaskStatus) bool {
    // 只允许从Pending状态转换
    if old != TaskStatusPending {
        return false
    }
    // 目标状态必须是Success或Failed
    return new == TaskStatusSuccess || new == TaskStatusFailed
}

// TaskManagerInterface 任务管理器接口
// 定义任务管理器的基本操作规范
// +---------------------+-----------------------------------+
// | 方法名              | 描述                              |
// +---------------------+-----------------------------------+
// | CreateTask          | 创建新任务                        |
// | UpdateTaskStatus    | 更新任务状态                      |
// +---------------------+-----------------------------------+
type TaskManagerInterface interface {
    CreateTask(callback Callback, expireTime time.Duration) *AsyncTask
    UpdateTaskStatus(taskID string, status TaskStatus, result interface{}, err error) error
}

// NewTaskManager 创建新的任务管理器实例
// 初始化任务存储映射表并启动清理工作协程
// +----------------+-----------------------------------+
// | 返回值         | *TaskManager                      |
// +----------------+-----------------------------------+
func NewTaskManager() *TaskManager {
    tm := &TaskManager{
        Tasks: make(map[string]*AsyncTask),
    }
    // 启动清理工作协程
    go tm.startCleanupWorker()
    return tm
}

// startCleanupWorker 启动清理工作协程
// 定期检查并清理过期的任务，默认每30秒执行一次
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

// transportTask 具体任务实现结构体
// 实现Task接口，用于在工作池中传输任务
// +----------------+------------------+-----------------------------------+
// | 字段名         | 类型             | 描述                              |
// +----------------+------------------+-----------------------------------+
// | ctx            | context.Context  | 上下文，用于控制任务生命周期      |
// | result         | chan *TaskResult | 结果通道，用于返回任务执行结果    |
// +----------------+------------------+-----------------------------------+
type transportTask struct {
    ctx    context.Context
    result chan *TaskResult
}

// ID 返回任务ID，实现Task接口
// 目前返回空字符串，需根据实际需求实现
// +----------------+-----------------------------------+
// | 返回值         | string                            |
// +----------------+-----------------------------------+
func (t *transportTask) ID() string {
    return ""
}

// Execute 执行任务，实现Task接口
// 目前返回nil，需根据实际需求实现具体逻辑
// +----------------+-----------------------------------+
// | 返回值         | interface{}                       |
// |                | error                             |
// +----------------+-----------------------------------+
func (t *transportTask) Execute() (interface{}, error) {
    return nil, nil
}
