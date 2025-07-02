package types

import "time"

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

// 任务管理器接口
type TaskManager interface {
	CreateTask(callback Callback, expireTime time.Duration) *AsyncTask
	UpdateTaskStatus(taskID string, status TaskStatus, result interface{}, err error) error
}
