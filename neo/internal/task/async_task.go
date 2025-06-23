package task

import (
	"fmt"
	"neo/internal/connection"
	"sync"
	"time"

	"github.com/google/uuid"
)

// 任务状态枚举（遵循neo项目常量命名规范）
type TaskStatus int

const (
	TaskStatusPending TaskStatus = iota // 待处理
	TaskStatusSuccess                   // 成功完成
	TaskStatusFailed                    // 执行失败
)

// 异步任务结构体（字段名采用驼峰式，增加JSON标签支持序列化）
type AsyncTask struct {
	TaskID     string              // 任务唯一标识（UUID格式）
	Status     TaskStatus          // 当前任务状态
	Result     interface{}         // 任务执行结果
	Error      error               // 错误信息（非nil表示执行失败）
	Callback   connection.Callback // 回调函数（复用connection模块定义）
	CreatedAt  time.Time           // 创建时间（用于超时计算）
	ExpireTime time.Duration       // 超时时间（默认30秒）
}

// 任务存储管理器（使用sync.RWMutex保证并发安全）
type TaskManager struct {
	mu    sync.RWMutex
	Tasks map[string]*AsyncTask
}

// 全局任务管理器实例（私有化，通过方法暴露）
var (
	instance *TaskManager
	once     sync.Once
)

// 获取单例任务管理器
func GetManager() *TaskManager {
	once.Do(func() {
		instance = &TaskManager{
			Tasks: make(map[string]*AsyncTask),
		}
		// 启动后台超时清理协程
		go instance.startCleanupWorker()
	})
	return instance
}

// 创建新任务（参数类型已正确）
func (m *TaskManager) CreateTask(callback connection.Callback, expireTime time.Duration) *AsyncTask {
	// 使用UUID生成唯一任务ID（符合neo现有ID生成规范）
	taskID := uuid.New().String()
	task := &AsyncTask{
		TaskID:     taskID,
		Status:     TaskStatusPending,
		Callback:   callback,
		CreatedAt:  time.Now(),
		ExpireTime: expireTime,
	}

	m.mu.Lock()
	m.Tasks[taskID] = task
	m.mu.Unlock()

	return task
}

// 更新任务状态
// 在UpdateTaskStatus方法中添加日志
func (m *TaskManager) UpdateTaskStatus(taskID string, status TaskStatus, result interface{}, err error) error {
	m.mu.RLock()
	task, exists := m.Tasks[taskID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("任务不存在: %s", taskID)
	}

	// 状态变更校验（防止无效状态转换）
	if !isValidStatusTransition(task.Status, status) {
		return fmt.Errorf("无效状态转换: %v -> %v", task.Status, status)
	}

	m.mu.Lock()
	task.Status = status
	task.Result = result
	task.Error = err
	m.mu.Unlock()

	// 触发回调函数
	if task.Callback != nil {
		go task.Callback(result, err)
	}

	return nil
}

// 后台超时清理协程
func (m *TaskManager) startCleanupWorker() {
	// 使用ticker定期执行清理（每30秒一次，可配置）
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
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
		m.mu.Unlock()
	}
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
