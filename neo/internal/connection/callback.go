package connection

import (
	"sync"
	"time"
)

// 新增：定义回调函数类型
// Callback 是异步操作完成后的回调函数类型
// 参数：
//   result: 操作结果
//   err: 错误信息（非nil表示操作失败）
type Callback func(interface{}, error)

// CallbackManager 管理连接相关事件的回调函数
type CallbackManager struct {
	callbacks sync.RWMutex
	registry  map[string]Callback // 修改：使用新定义的Callback类型
}

// NewCallbackManager 创建回调管理器实例
func NewCallbackManager() *CallbackManager {
	return &CallbackManager{
		registry: make(map[string]Callback), // 修复：使用Callback类型而非匿名函数类型
	}
}

// Register 注册回调函数并设置超时清理
// msgID: 消息唯一标识
// cb: 回调函数
// timeout: 超时自动清理时间
func (m *CallbackManager) Register(msgID string, cb func(interface{}, error), timeout time.Duration) {
	m.callbacks.Lock()
	defer m.callbacks.Unlock()
	m.registry[msgID] = cb

	// 超时清理（解决原代码内存泄漏风险）
	time.AfterFunc(timeout, func() {
		m.callbacks.Lock()
		defer m.callbacks.Unlock()
		if _, exists := m.registry[msgID]; exists {
			delete(m.registry, msgID)
		}
	})
}

// HandleResponse 处理响应并触发回调
func (m *CallbackManager) HandleResponse(msgID string, result interface{}, err error) {
	m.callbacks.RLock()
	cb, exists := m.registry[msgID]
	m.callbacks.RUnlock()

	if exists {
		go cb(result, err) // 异步执行
		m.callbacks.Lock()
		delete(m.registry, msgID) // 执行后清理
		m.callbacks.Unlock()
	}
}

// 全局实例（供包内直接使用）
var defaultCallbackManager = NewCallbackManager()

// 包级快捷函数（简化外部调用）
func RegisterCallback(msgID string, cb func(interface{}, error), timeout time.Duration) {
	defaultCallbackManager.Register(msgID, cb, timeout)
}

func HandleCallbackResponse(msgID string, result interface{}, err error) {
	defaultCallbackManager.HandleResponse(msgID, result, err)
}
