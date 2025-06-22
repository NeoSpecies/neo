package connection // 将包名从core修正为connection

import (
	"sync"
	"time"
	"errors"
)

// 修改回调存储结构，添加超时时间
var callbackMap = struct {
	sync.RWMutex
	m map[string]struct {
		cb      func(interface{}, error)
		timeout time.Duration
		timer   *time.Timer
	}
}{m: make(map[string]struct{
	cb      func(interface{}, error)
	timeout time.Duration
	timer   *time.Timer
})}

// 注册回调时添加超时参数
func RegisterCallback(msgID string, timeout time.Duration, cb func(interface{}, error)) {
	callbackMap.Lock()
	defer callbackMap.Unlock()

	// 先取消已存在的定时器
	if existing, ok := callbackMap.m[msgID]; ok {
		existing.timer.Stop()
	}

	// 创建新的超时清理定时器
	timer := time.AfterFunc(timeout, func() {
		callbackMap.Lock()
		delete(callbackMap.m, msgID)
		callbackMap.Unlock()
		cb(nil, errors.New("callback timeout"))
	})

	callbackMap.m[msgID] = struct {
		cb      func(interface{}, error)
		timeout time.Duration
		timer   *time.Timer
	}{cb: cb, timeout: timeout, timer: timer}
}

// 处理响应时停止定时器
func HandleResponse(msgID string, result interface{}, err error) {
	callbackMap.RLock()
	entry, exists := callbackMap.m[msgID]
	callbackMap.RUnlock()

	if exists {
		entry.timer.Stop() // 停止超时定时器
		go entry.cb(result, err)
		callbackMap.Lock()
		delete(callbackMap.m, msgID)
		callbackMap.Unlock()
	}
}
