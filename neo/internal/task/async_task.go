// Deprecated: 此文件已废弃，请使用 internal/types 包中的对应类型
package task

import (
	"neo/internal/types"
)

// 类型别名确保兼容性
type TaskStatus = types.TaskStatus
type AsyncTask = types.AsyncTask
type TaskManager = types.TaskManager
type TaskManagerInterface = types.TaskManagerInterface

// 移除所有方法定义，这些方法现在应在task_types.go中实现
